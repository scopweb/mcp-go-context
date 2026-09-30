package memory

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/scopweb/mcp-go-context/internal/storage"
)

const storageSchemaVersion = 2

// ErrCapacity means every remaining entry is protected and nothing was evicted.
var ErrCapacity = errors.New("memory capacity reached; promoted memories were not evicted")

func (m *Manager) begin() (func(), error) {
	if !m.config.Enabled || m.config.StoragePath == "" {
		m.mu.Lock()
		return func() { m.mu.Unlock() }, nil
	}
	release, err := storage.Lock(m.config.StoragePath)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	if err := m.reloadLocked(); err != nil {
		m.mu.Unlock()
		_ = release()
		return nil, err
	}
	return func() {
		m.mu.Unlock()
		_ = release()
	}, nil
}

func (m *Manager) reloadLocked() error {
	m.sessions = make(map[string]*Session)
	if err := m.loadSessions(); err != nil {
		return err
	}
	m.rebuildIndexes()
	return nil
}

func isSessionFile(name string) bool {
	switch name {
	case "projects.json", "schema.json":
		return false
	}
	return strings.HasSuffix(name, ".json")
}

// StoreForProject saves a memory in an explicit project, preserving promotion and usage on update.
func (m *Manager) StoreForProject(project, key, content string, tags []string, decisionType, reason string, alternatives []string) error {
	if !m.config.Enabled {
		return nil
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("memory key is required")
	}
	if project == "" {
		project = m.activeProject
	}
	unlock, err := m.begin()
	if err != nil {
		return err
	}
	defer unlock()

	loc, found := m.findInProject(key, project)
	session := loc.session
	mapKey := loc.mapKey
	if !found {
		session = m.getCurrentSession()
		mapKey = m.freshMapKey(key, project)
		if m.config.MaxEntries > 0 && len(session.Memories) >= m.config.MaxEntries {
			if !m.evictLowScored(session) {
				return ErrCapacity
			}
		}
	} else {
		m.removeFromIndexes(mapKey, loc.mem)
	}

	mem := Memory{
		Key:          key,
		Content:      content,
		Tags:         tags,
		Timestamp:    time.Now().UTC(),
		Usage:        0,
		Project:      project,
		DecisionType: decisionType,
		Reason:       reason,
		Alternatives: alternatives,
	}
	if found {
		mem.Usage = loc.mem.Usage
		mem.Promoted = loc.mem.Promoted
		mem.Confidence = loc.mem.Confidence
		if mem.DecisionType == "" {
			mem.DecisionType = loc.mem.DecisionType
		}
		if mem.Reason == "" {
			mem.Reason = loc.mem.Reason
		}
		if len(mem.Alternatives) == 0 {
			mem.Alternatives = loc.mem.Alternatives
		}
		if len(mem.Tags) == 0 {
			mem.Tags = loc.mem.Tags
		}
	}
	mem.Related = m.findRelated(mem)
	session.Memories[mapKey] = mem
	m.addToIndexes(mapKey, mem)
	for _, relKey := range mem.Related {
		m.addBacklink(relKey, mapKey)
	}
	return m.saveSession(session)
}

func (m *Manager) findInProject(userKey, project string) (located, bool) {
	for _, session := range m.sessions {
		for mapKey, mem := range session.Memories {
			if mem.Key == "" {
				mem.Key = mapKey
			}
			if mem.Key != userKey {
				continue
			}
			if m.matchesProject(&mem, project) {
				return located{session: session, mapKey: mapKey, mem: mem}, true
			}
		}
	}
	return located{}, false
}

type located struct {
	session *Session
	mapKey  string
	mem     Memory
}

func (m *Manager) freshMapKey(userKey, project string) string {
	for _, session := range m.sessions {
		for mapKey, mem := range session.Memories {
			key := mem.Key
			if key == "" {
				key = mapKey
			}
			if key == userKey {
				return project + "::" + userKey
			}
		}
	}
	return userKey
}

func (m *Manager) evictLowScored(session *Session) bool {
	var lowestKey string
	var lowestScore float64
	found := false
	for key, mem := range session.Memories {
		if mem.Promoted {
			continue
		}
		score := m.calculateScore(&mem, "", nil, nil)
		if !found || score < lowestScore {
			lowestScore = score
			lowestKey = key
			found = true
		}
	}
	if !found {
		return false
	}
	m.removeFromIndexes(lowestKey, session.Memories[lowestKey])
	delete(session.Memories, lowestKey)
	return true
}

func prepareStorage(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	release, err := storage.Lock(path)
	if err != nil {
		return err
	}
	defer release()
	return migrateLocked(path)
}

func migrateLocked(path string) error {
	schemaPath := filepath.Join(path, "schema.json")
	if _, err := os.Stat(schemaPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() || !isSessionFile(entry.Name()) {
			continue
		}
		sessions = append(sessions, entry.Name())
	}
	if len(sessions) > 0 {
		backup := filepath.Join(path, "backups", time.Now().UTC().Format("20060102T150405Z"))
		if err := os.MkdirAll(backup, 0o755); err != nil {
			return err
		}
		for _, name := range sessions {
			if err := copyFile(filepath.Join(path, name), filepath.Join(backup, name)); err != nil {
				return fmt.Errorf("backup %s: %w", name, err)
			}
		}
	}
	return storage.WriteAtomic(schemaPath, []byte("{\n  \"schemaVersion\": 2\n}\n"))
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// SearchDecisionsScoped filters decisions by project. An empty project uses the active project.
func (m *Manager) SearchDecisionsScoped(decisionType, keyword, project string, limit int) ([]*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}
	unlock, err := m.begin()
	if err != nil {
		return nil, err
	}
	defer unlock()

	var results []*Memory
	for _, session := range m.sessions {
		for mapKey, mem := range session.Memories {
			if mem.Key == "" {
				mem.Key = mapKey
			}
			if !m.matchesProject(&mem, project) {
				continue
			}
			if decisionType != "" && mem.DecisionType != decisionType {
				continue
			}
			if keyword != "" && !decisionMatches(mem, keyword) {
				continue
			}
			memCopy := mem
			results = append(results, &memCopy)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	return trimMemories(results, limit), nil
}

func decisionMatches(mem Memory, keyword string) bool {
	if contains(mem.Content, keyword) || contains(mem.Reason, keyword) || contains(mem.Key, keyword) {
		return true
	}
	for _, alt := range mem.Alternatives {
		if contains(alt, keyword) {
			return true
		}
	}
	return false
}

func trimMemories(results []*Memory, limit int) []*Memory {
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func searchableText(mem *Memory) string {
	return strings.ToLower(mem.Key + "\n" + mem.Content + "\n" + mem.Reason + "\n" + strings.Join(mem.Alternatives, "\n"))
}
