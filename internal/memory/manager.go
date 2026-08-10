package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/scopweb/mcp-go-context/internal/config"
)

// Manager handles conversation memory persistence
type Manager struct {
	config        config.MemoryConfig
	activeProject string // current project slug (inferred from cwd at startup)
	sessions      map[string]*Session
	mu            sync.RWMutex
	// Inverted indexes for fast search
	tagIndex         map[string]map[string]bool // tag -> set of memory keys
	wordIndex        map[string]map[string]bool // word -> set of memory keys
	lastIndexRebuild time.Time
}

// Session represents a conversation session
type Session struct {
	ID        string            `json:"id"`
	StartTime time.Time         `json:"startTime"`
	LastUsed  time.Time         `json:"lastUsed"`
	Memories  map[string]Memory `json:"memories"`
}

// Memory represents a stored memory item
type Memory struct {
	Key       string    `json:"key"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	Timestamp time.Time `json:"timestamp"`
	Usage     int       `json:"usage"`
	// Project scope (Claude Desktop project, inferred from cwd slug). Empty = legacy / Unassigned.
	Project string `json:"project,omitempty"`
	// Promotion fields for Phase 6 convergence with SessionMemory
	Promoted   bool   `json:"promoted,omitempty"`   // true if promoted from session memory
	Confidence string `json:"confidence,omitempty"` // "low", "medium", "high" - stability of this memory
	// Decision-specific fields
	DecisionType string   `json:"decisionType,omitempty"` // architecture, fix, approach, etc.
	Reason       string   `json:"reason,omitempty"`       // why this decision was made
	Alternatives []string `json:"alternatives,omitempty"` // what else was considered
	// Related holds keys of auto-linked memories (shared tags + word overlap).
	// Links are bidirectional: storing A linked to B also adds A to B.Related.
	Related []string `json:"related,omitempty"`
}

// maxRelatedMemories caps auto-linked relations per memory.
const maxRelatedMemories = 5

// UnassignedProject is the placeholder label for memories with empty Project.
const UnassignedProject = "unassigned"

// WildcardProject disables project filtering when passed to Search.
const WildcardProject = "*"

// InferProject derives a normalized project slug from a working directory path.
// Returns "default" if the path is empty or yields no usable segment.
// Examples:
//
//	C:\MCPs\clone\mcp-go-context -> "mcp-go-context"
//	/home/user/my project        -> "my-project"
//	/home/user/café              -> "cafe"
func InferProject(cwd string) string {
	if cwd == "" {
		return "default"
	}
	clean := strings.TrimRight(strings.ReplaceAll(cwd, "\\", "/"), "/")
	segs := strings.Split(clean, "/")
	last := ""
	for i := len(segs) - 1; i >= 0; i-- {
		s := strings.TrimSpace(segs[i])
		if s == "" {
			continue
		}
		if strings.HasSuffix(s, ":") { // skip Windows drive letter like "C:"
			continue
		}
		last = s
		break
	}
	if last == "" {
		return "default"
	}
	slug := slugify(last)
	if slug == "" {
		return "default"
	}
	return slug
}

// slugify converts a string to a lowercase [a-z0-9-] slug.
// Folds common Latin accents to ASCII, replaces other chars with '-',
// and collapses repeated dashes.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ä', 'â', 'ã', 'å':
			r = 'a'
		case 'é', 'è', 'ë', 'ê':
			r = 'e'
		case 'í', 'ì', 'ï', 'î':
			r = 'i'
		case 'ó', 'ò', 'ö', 'ô', 'õ':
			r = 'o'
		case 'ú', 'ù', 'ü', 'û':
			r = 'u'
		case 'ñ':
			r = 'n'
		case 'ç':
			r = 'c'
		}
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// Non-Latin letter/digit we can't fold cleanly -> dash.
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// SearchResult represents a scored search result
type SearchResult struct {
	Memory *Memory
	Score  float64
}

// New creates a new memory manager scoped to a project slug.
// Pass InferProject(cwd) for auto-detection.
func New(cfg config.MemoryConfig, project string) (*Manager, error) {
	if project == "" {
		project = "default"
	}
	m := &Manager{
		config:        cfg,
		activeProject: project,
		sessions:      make(map[string]*Session),
		tagIndex:      make(map[string]map[string]bool),
		wordIndex:     make(map[string]map[string]bool),
	}

	if cfg.Enabled {
		// Ensure storage directory exists
		if err := os.MkdirAll(cfg.StoragePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create memory storage: %w", err)
		}

		// Load existing sessions
		if err := m.loadSessions(); err != nil {
			return nil, fmt.Errorf("failed to load sessions: %w", err)
		}

		// Build indexes
		m.rebuildIndexes()

		// Start cleanup routine
		go m.cleanupRoutine()
	}

	return m, nil
}

// Store saves a memory item
func (m *Manager) Store(key, content string, tags []string) error {
	return m.StoreWithType(key, content, tags, "", "", nil)
}

// StoreWithType saves a memory item with decision metadata
func (m *Manager) StoreWithType(key, content string, tags []string, decisionType, reason string, alternatives []string) error {
	if !m.config.Enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create current session
	session := m.getCurrentSession()

	// Check MaxEntries limit before adding
	if len(session.Memories) >= m.config.MaxEntries && m.config.MaxEntries > 0 {
		// Evict lowest scored entry instead of just oldest
		m.evictLowScored(session)
	}

	// Remove old key from indexes if updating
	if _, exists := session.Memories[key]; exists {
		m.removeFromIndexes(key, session.Memories[key])
	}

	// Store memory
	mem := Memory{
		Key:          key,
		Content:      content,
		Tags:         tags,
		Timestamp:    time.Now(),
		Usage:        0,
		Project:      m.activeProject,
		DecisionType: decisionType,
		Reason:       reason,
		Alternatives: alternatives,
	}
	// Auto-link before indexing so the new memory never matches itself
	mem.Related = m.findRelated(mem)
	session.Memories[key] = mem

	// Update indexes
	m.addToIndexes(key, mem)

	// Backlink: make relations navigable in both directions
	for _, relKey := range mem.Related {
		m.addBacklink(relKey, key)
	}

	// Save to disk
	return m.saveSession(session)
}

// Retrieve gets a memory item by key
func (m *Manager) Retrieve(key string) (*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Search across all sessions
	for _, session := range m.sessions {
		if memory, exists := session.Memories[key]; exists {
			// Increment usage in the stored value, not just in a copy.
			memory.Usage++
			session.Memories[key] = memory
			if err := m.saveSession(session); err != nil {
				return nil, err
			}
			return &memory, nil
		}
	}

	return nil, fmt.Errorf("memory not found: %s", key)
}

// Get returns a memory item by key without mutating usage statistics.
func (m *Manager) Get(key string) (*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, session := range m.sessions {
		if memory, exists := session.Memories[key]; exists {
			memoryCopy := memory
			return &memoryCopy, nil
		}
	}

	return nil, fmt.Errorf("memory not found: %s", key)
}

// ActiveProject returns the project slug this manager is scoped to.
func (m *Manager) ActiveProject() string {
	return m.activeProject
}

// Stats summarizes memory usage across all sessions and on-disk storage,
// inspired by memcached's stats command.
type Stats struct {
	ActiveProject  string    `json:"activeProject"`
	Sessions       int       `json:"sessions"`
	Memories       int       `json:"memories"`
	Promoted       int       `json:"promoted"`
	Decisions      int       `json:"decisions"`
	TotalUsage     int       `json:"totalUsage"`   // sum of per-memory usage counters (read hits)
	StorageBytes   int64     `json:"storageBytes"` // size of persisted session files
	OldestMemoryAt time.Time `json:"oldestMemoryAt,omitempty"`
	NewestMemoryAt time.Time `json:"newestMemoryAt,omitempty"`
	MaxEntries     int       `json:"maxEntries"`
	SessionTTLDays int       `json:"sessionTTLDays"`
}

// Stats returns aggregate memory statistics across every loaded session
// plus the size of persisted storage.
func (m *Manager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := Stats{
		ActiveProject:  m.activeProject,
		Sessions:       len(m.sessions),
		MaxEntries:     m.config.MaxEntries,
		SessionTTLDays: m.config.SessionTTLDays,
	}

	for _, session := range m.sessions {
		for _, mem := range session.Memories {
			stats.Memories++
			stats.TotalUsage += mem.Usage
			if mem.Promoted {
				stats.Promoted++
			}
			if mem.DecisionType != "" {
				stats.Decisions++
			}
			if stats.OldestMemoryAt.IsZero() || mem.Timestamp.Before(stats.OldestMemoryAt) {
				stats.OldestMemoryAt = mem.Timestamp
			}
			if mem.Timestamp.After(stats.NewestMemoryAt) {
				stats.NewestMemoryAt = mem.Timestamp
			}
		}
	}

	if entries, err := os.ReadDir(m.config.StoragePath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			if info, err := entry.Info(); err == nil {
				stats.StorageBytes += info.Size()
			}
		}
	}

	return stats
}

// matchesProject reports whether mem belongs to the requested project filter.
// filter == WildcardProject     -> always true.
// filter == ""                  -> matches the active project only.
// filter == UnassignedProject   -> matches legacy memories with empty Project.
// filter == "<slug>"            -> exact match.
func (m *Manager) matchesProject(mem *Memory, filter string) bool {
	if filter == WildcardProject {
		return true
	}
	target := filter
	if target == "" {
		target = m.activeProject
	}
	memProject := mem.Project
	if memProject == "" {
		memProject = UnassignedProject
	}
	return memProject == target
}

// Search finds memories with improved ranking. Filters to the active project by default;
// pass project="*" to search across all projects.
func (m *Manager) Search(query string, tags []string) ([]*Memory, error) {
	return m.SearchWithProject(query, tags, "")
}

// SearchWithProject is Search with an explicit project filter.
func (m *Manager) SearchWithProject(query string, tags []string, project string) ([]*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect candidates using inverted index when possible
	candidates := make(map[string]*Memory)
	queryLower := strings.ToLower(query)
	queryWords := tokenize(queryLower)

	// If query provided, use word index to find candidates
	if query != "" && len(queryWords) > 0 {
		for _, word := range queryWords {
			if keys, exists := m.wordIndex[word]; exists {
				for key := range keys {
					if mem := m.findMemory(key); mem != nil {
						candidates[key] = mem
					}
				}
			}
		}
	}

	// If tags provided, use tag index
	if len(tags) > 0 {
		for _, tag := range tags {
			tagLower := strings.ToLower(tag)
			if keys, exists := m.tagIndex[tagLower]; exists {
				for key := range keys {
					if mem := m.findMemory(key); mem != nil {
						candidates[key] = mem
					}
				}
			}
		}
	}

	// If no index hits, fall back to scanning all memories
	if len(candidates) == 0 {
		for _, session := range m.sessions {
			for _, memory := range session.Memories {
				memCopy := memory
				candidates[memory.Key] = &memCopy
			}
		}
	}

	// Score and filter results (apply project filter)
	var results []SearchResult
	for _, mem := range candidates {
		if !m.matchesProject(mem, project) {
			continue
		}
		score := m.calculateScore(mem, queryLower, queryWords, tags)
		if score > 0 {
			results = append(results, SearchResult{Memory: mem, Score: score})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Convert to []*Memory with limit
	var memories []*Memory
	for i, res := range results {
		if m.config.MaxResults > 0 && i >= m.config.MaxResults {
			break
		}
		memories = append(memories, res.Memory)
	}

	return memories, nil
}

// GetRecentMemories returns recent memories sorted by recency
func (m *Manager) GetRecentMemories(limit int) ([]*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var allMemories []*Memory

	for _, session := range m.sessions {
		for _, memory := range session.Memories {
			memoryCopy := memory
			allMemories = append(allMemories, &memoryCopy)
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(allMemories, func(i, j int) bool {
		return allMemories[i].Timestamp.After(allMemories[j].Timestamp)
	})

	if limit > 0 && len(allMemories) > limit {
		return allMemories[:limit], nil
	}

	return allMemories, nil
}

// ProjectStat aggregates memory count per project for dashboard listing.
type ProjectStat struct {
	Project string `json:"project"`
	Count   int    `json:"count"`
	Active  bool   `json:"active"`
}

// ListProjects returns unique projects with memory counts.
// Memories with empty Project are grouped under UnassignedProject.
// Sorted descending by count; active project first when counts tie.
func (m *Manager) ListProjects() []ProjectStat {
	if !m.config.Enabled {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[string]int)
	for _, session := range m.sessions {
		for _, mem := range session.Memories {
			p := mem.Project
			if p == "" {
				p = UnassignedProject
			}
			counts[p]++
		}
	}
	// Ensure the active project is always listed, even with zero memories.
	if _, ok := counts[m.activeProject]; !ok {
		counts[m.activeProject] = 0
	}

	stats := make([]ProjectStat, 0, len(counts))
	for p, c := range counts {
		stats = append(stats, ProjectStat{Project: p, Count: c, Active: p == m.activeProject})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count != stats[j].Count {
			return stats[i].Count > stats[j].Count
		}
		if stats[i].Active != stats[j].Active {
			return stats[i].Active
		}
		return stats[i].Project < stats[j].Project
	})
	return stats
}

// ListMemories returns all memories sorted by recency, optionally filtered by decision type.
func (m *Manager) ListMemories(limit int, decisionType string) ([]*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var allMemories []*Memory
	for _, session := range m.sessions {
		for _, memory := range session.Memories {
			if decisionType != "" && memory.DecisionType != decisionType {
				continue
			}
			memoryCopy := memory
			allMemories = append(allMemories, &memoryCopy)
		}
	}

	sort.Slice(allMemories, func(i, j int) bool {
		return allMemories[i].Timestamp.After(allMemories[j].Timestamp)
	})

	if limit > 0 && len(allMemories) > limit {
		return allMemories[:limit], nil
	}

	return allMemories, nil
}

// Delete removes a memory by key from whichever session contains it.
func (m *Manager) Delete(key string) error {
	if !m.config.Enabled {
		return fmt.Errorf("memory disabled")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		memory, exists := session.Memories[key]
		if !exists {
			continue
		}

		m.removeFromIndexes(key, memory)
		delete(session.Memories, key)
		m.removeKeyFromRelated(key)
		return m.saveSession(session)
	}

	return fmt.Errorf("memory not found: %s", key)
}

// Clear removes all memories
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*Session)
	m.tagIndex = make(map[string]map[string]bool)
	m.wordIndex = make(map[string]map[string]bool)

	// Clear storage
	entries, err := os.ReadDir(m.config.StoragePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			os.Remove(filepath.Join(m.config.StoragePath, entry.Name()))
		}
	}

	return nil
}

// Private methods

func (m *Manager) getCurrentSession() *Session {
	sessionID := "current"

	if session, exists := m.sessions[sessionID]; exists {
		session.LastUsed = time.Now()
		return session
	}

	session := &Session{
		ID:        sessionID,
		StartTime: time.Now(),
		LastUsed:  time.Now(),
		Memories:  make(map[string]Memory),
	}

	m.sessions[sessionID] = session
	return session
}

func (m *Manager) findMemory(key string) *Memory {
	for _, session := range m.sessions {
		if memory, exists := session.Memories[key]; exists {
			return &memory
		}
	}
	return nil
}

func (m *Manager) loadSessions() error {
	entries, err := os.ReadDir(m.config.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			sessionID := strings.TrimSuffix(entry.Name(), ".json")

			data, err := os.ReadFile(filepath.Join(m.config.StoragePath, entry.Name()))
			if err != nil {
				continue
			}

			var session Session
			if err := json.Unmarshal(data, &session); err != nil {
				continue
			}

			m.sessions[sessionID] = &session
		}
	}

	return nil
}

func (m *Manager) saveSession(session *Session) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	filename := filepath.Join(m.config.StoragePath, session.ID+".json")
	return os.WriteFile(filename, data, 0644)
}

func (m *Manager) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -m.config.SessionTTLDays)

	// Clean expired sessions
	for id, session := range m.sessions {
		if session.LastUsed.Before(cutoff) {
			// Remove from indexes
			for key, memory := range session.Memories {
				m.removeFromIndexes(key, memory)
				m.removeKeyFromRelated(key)
			}
			delete(m.sessions, id)
			os.Remove(filepath.Join(m.config.StoragePath, id+".json"))
		}
	}

	// Limit total sessions with LRU eviction
	if m.config.MaxSessions > 0 && len(m.sessions) > m.config.MaxSessions {
		sessions := make([]*Session, 0, len(m.sessions))
		for _, session := range m.sessions {
			sessions = append(sessions, session)
		}
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].LastUsed.Before(sessions[j].LastUsed)
		})

		toRemove := len(m.sessions) - m.config.MaxSessions
		for i := 0; i < toRemove && i < len(sessions); i++ {
			id := sessions[i].ID
			for key, memory := range sessions[i].Memories {
				m.removeFromIndexes(key, memory)
				m.removeKeyFromRelated(key)
			}
			delete(m.sessions, id)
			os.Remove(filepath.Join(m.config.StoragePath, id+".json"))
		}
	}
}

// evictLowScored removes the memory with lowest search score
func (m *Manager) evictLowScored(session *Session) {
	if len(session.Memories) == 0 {
		return
	}

	var lowestKey string
	var lowestScore float64 = -1

	for key, mem := range session.Memories {
		score := m.calculateScore(&mem, "", nil, nil)
		if lowestScore < 0 || score < lowestScore {
			lowestScore = score
			lowestKey = key
		}
	}

	if lowestKey != "" {
		m.removeFromIndexes(lowestKey, session.Memories[lowestKey])
		delete(session.Memories, lowestKey)
		m.removeKeyFromRelated(lowestKey)
	}
}

// findRelated returns up to maxRelatedMemories keys of memories related to
// mem: candidates must share at least one tag, ranked by shared tags
// (weight 10) plus content word overlap. Active project only; never self.
// Must be called with m.mu held and before mem itself is indexed.
func (m *Manager) findRelated(mem Memory) []string {
	scores := make(map[string]int)
	sharedTags := make(map[string]int)

	for _, tag := range mem.Tags {
		for key := range m.tagIndex[strings.ToLower(tag)] {
			if key == mem.Key {
				continue
			}
			sharedTags[key]++
			scores[key] += 10
		}
	}
	if len(sharedTags) == 0 {
		return nil
	}

	for _, word := range tokenize(strings.ToLower(mem.Content)) {
		for key := range m.wordIndex[word] {
			if key != mem.Key && sharedTags[key] > 0 {
				scores[key]++
			}
		}
	}

	type rel struct {
		key   string
		score int
	}
	var ranked []rel
	for key, score := range scores {
		target := m.findMemory(key)
		if target == nil || !m.matchesProject(target, "") {
			continue
		}
		ranked = append(ranked, rel{key, score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].key < ranked[j].key
	})
	if len(ranked) > maxRelatedMemories {
		ranked = ranked[:maxRelatedMemories]
	}

	keys := make([]string, 0, len(ranked))
	for _, r := range ranked {
		keys = append(keys, r.key)
	}
	return keys
}

// addBacklink appends fromKey to the Related list of toKey (if not already
// present), keeping the list capped. Must be called with m.mu held.
func (m *Manager) addBacklink(toKey, fromKey string) {
	for _, session := range m.sessions {
		target, exists := session.Memories[toKey]
		if !exists {
			continue
		}
		for _, r := range target.Related {
			if r == fromKey {
				return
			}
		}
		target.Related = append(target.Related, fromKey)
		if len(target.Related) > maxRelatedMemories {
			target.Related = target.Related[len(target.Related)-maxRelatedMemories:]
		}
		session.Memories[toKey] = target
		return
	}
}

// removeKeyFromRelated deletes dangling links to key from every memory.
// Must be called with m.mu held.
func (m *Manager) removeKeyFromRelated(key string) {
	for _, session := range m.sessions {
		for k, mem := range session.Memories {
			if len(mem.Related) == 0 {
				continue
			}
			filtered := make([]string, 0, len(mem.Related))
			for _, r := range mem.Related {
				if r != key {
					filtered = append(filtered, r)
				}
			}
			if len(filtered) != len(mem.Related) {
				mem.Related = filtered
				session.Memories[k] = mem
			}
		}
	}
}

// addToIndexes adds a memory to inverted indexes
func (m *Manager) addToIndexes(key string, mem Memory) {
	// Tag index
	for _, tag := range mem.Tags {
		tagLower := strings.ToLower(tag)
		if m.tagIndex[tagLower] == nil {
			m.tagIndex[tagLower] = make(map[string]bool)
		}
		m.tagIndex[tagLower][key] = true
	}

	// Word index
	words := tokenize(strings.ToLower(mem.Content))
	for _, word := range words {
		if m.wordIndex[word] == nil {
			m.wordIndex[word] = make(map[string]bool)
		}
		m.wordIndex[word][key] = true
	}
}

// removeFromIndexes removes a memory from inverted indexes
func (m *Manager) removeFromIndexes(key string, mem Memory) {
	// From tag index
	for _, tag := range mem.Tags {
		tagLower := strings.ToLower(tag)
		if m.tagIndex[tagLower] != nil {
			delete(m.tagIndex[tagLower], key)
			if len(m.tagIndex[tagLower]) == 0 {
				delete(m.tagIndex, tagLower)
			}
		}
	}

	// From word index
	words := tokenize(strings.ToLower(mem.Content))
	for _, word := range words {
		if m.wordIndex[word] != nil {
			delete(m.wordIndex[word], key)
			if len(m.wordIndex[word]) == 0 {
				delete(m.wordIndex, word)
			}
		}
	}
}

// rebuildIndexes rebuilds the inverted indexes from all sessions
func (m *Manager) rebuildIndexes() {
	m.tagIndex = make(map[string]map[string]bool)
	m.wordIndex = make(map[string]map[string]bool)

	for _, session := range m.sessions {
		for key, mem := range session.Memories {
			m.addToIndexes(key, mem)
		}
	}

	m.lastIndexRebuild = time.Now()
}

// tokenize splits text into normalized words
func tokenize(text string) []string {
	// Split on non-word characters
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	// Filter common stop words and short words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "was": true, "are": true, "were": true, "been": true,
		"be": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "can": true,
		"this": true, "that": true, "these": true, "those": true,
		"i": true, "you": true, "he": true, "she": true, "it": true,
		"we": true, "they": true, "what": true, "which": true, "who": true,
		"when": true, "where": true, "why": true, "how": true,
	}

	var result []string
	for _, word := range words {
		lower := strings.ToLower(word)
		if len(lower) >= 3 && !stopWords[lower] {
			result = append(result, lower)
		}
	}

	return result
}

// calculateScore computes a relevance score for a memory
func (m *Manager) calculateScore(mem *Memory, query string, queryWords []string, tags []string) float64 {
	if mem == nil {
		return 0
	}

	var score float64

	// Tag matching (30% weight)
	if len(tags) > 0 {
		tagScore := 0.0
		memTagsLower := make(map[string]bool)
		for _, tag := range mem.Tags {
			memTagsLower[strings.ToLower(tag)] = true
		}
		for _, tag := range tags {
			if memTagsLower[strings.ToLower(tag)] {
				tagScore++
			}
		}
		if len(tags) > 0 {
			score += (tagScore / float64(len(tags))) * 0.30
		}
	}

	// Content word matching (40% weight)
	if len(queryWords) > 0 {
		contentWords := tokenize(strings.ToLower(mem.Content))
		contentWordSet := make(map[string]bool)
		for _, w := range contentWords {
			contentWordSet[w] = true
		}

		matchCount := 0
		for _, qw := range queryWords {
			if contentWordSet[qw] {
				matchCount++
			}
		}
		if len(queryWords) > 0 {
			score += (float64(matchCount) / float64(len(queryWords))) * 0.40
		}
	}

	// Recency (20% weight) - logarithmic scale
	ageHours := time.Since(mem.Timestamp).Hours()
	if ageHours < 1 {
		score += 0.20 // Very recent
	} else if ageHours < 24 {
		score += 0.15
	} else if ageHours < 168 { // 1 week
		score += 0.10
	} else if ageHours < 720 { // 1 month
		score += 0.05
	}

	// Usage frequency (10% weight)
	if mem.Usage > 10 {
		score += 0.10
	} else if mem.Usage > 5 {
		score += 0.07
	} else if mem.Usage > 0 {
		score += 0.03
	}

	return score
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// SearchDecisions finds decision memories by type and keywords
func (m *Manager) SearchDecisions(decisionType, keyword string, limit int) ([]*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*Memory

	for _, session := range m.sessions {
		for _, memory := range session.Memories {
			// Filter by decision type
			if decisionType != "" && memory.DecisionType != decisionType {
				continue
			}

			// Filter by keyword in content, reason, or alternatives
			if keyword != "" {
				found := contains(memory.Content, keyword) ||
					contains(memory.Reason, keyword) ||
					contains(memory.Key, keyword)
				for _, alt := range memory.Alternatives {
					if contains(alt, keyword) {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			memCopy := memory
			results = append(results, &memCopy)
		}
	}

	// Sort by timestamp
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	if limit > 0 && len(results) > limit {
		return results[:limit], nil
	}

	return results, nil
}

// GetPromotedMemories returns only memories that have been promoted from session memory
func (m *Manager) GetPromotedMemories(limit int) ([]*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var promoted []*Memory
	for _, session := range m.sessions {
		for _, memory := range session.Memories {
			if memory.Promoted {
				memCopy := memory
				promoted = append(promoted, &memCopy)
			}
		}
	}

	// Sort by timestamp descending (most recent promoted first)
	sort.Slice(promoted, func(i, j int) bool {
		return promoted[i].Timestamp.After(promoted[j].Timestamp)
	})

	if limit > 0 && len(promoted) > limit {
		return promoted[:limit], nil
	}

	return promoted, nil
}

// Promote marks a memory as promoted from session memory (high-value item)
func (m *Manager) Promote(key string, confidence string) error {
	if !m.config.Enabled {
		return fmt.Errorf("memory disabled")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the memory across all sessions
	for _, session := range m.sessions {
		if memory, exists := session.Memories[key]; exists {
			memory.Promoted = true
			if confidence != "" {
				memory.Confidence = confidence
			}
			session.Memories[key] = memory
			return m.saveSession(session)
		}
	}

	return fmt.Errorf("memory not found: %s", key)
}

// Demote removes the promoted flag from a memory (reverts to session-only)
func (m *Manager) Demote(key string) error {
	if !m.config.Enabled {
		return fmt.Errorf("memory disabled")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		if memory, exists := session.Memories[key]; exists {
			memory.Promoted = false
			session.Memories[key] = memory
			return m.saveSession(session)
		}
	}

	return fmt.Errorf("memory not found: %s", key)
}

// GetDecisionTypes returns all unique decision types in memory
func (m *Manager) GetDecisionTypes() ([]string, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	typeSet := make(map[string]bool)
	for _, session := range m.sessions {
		for _, memory := range session.Memories {
			if memory.DecisionType != "" {
				typeSet[memory.DecisionType] = true
			}
		}
	}

	var types []string
	for t := range typeSet {
		types = append(types, t)
	}
	sort.Strings(types)
	return types, nil
}

// SuggestForPromotion returns the best candidates for promotion to persistent
// high-value memory (Fase 1). It uses a transparent heuristic combining
// decision structure, usage, recency, and content signals.
//
// Only non-promoted memories are considered.
// Results are sorted by promotion merit (highest first).
func (m *Manager) SuggestForPromotion(limit int) ([]*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}
	if limit <= 0 {
		limit = 5
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	type scoredMem struct {
		mem   *Memory
		score float64
	}

	var candidates []scoredMem

	// Weighted decision keywords (higher weight = stronger signal of durable value)
	weightedKeywords := map[string]float64{
		// High value structural signals
		"architecture": 0.12, "arquitectura": 0.12,
		"security": 0.11, "seguridad": 0.11,
		"performance": 0.10, "rendimiento": 0.10,
		"convention": 0.09, "convencion": 0.09, "standard": 0.08,
		"decision": 0.08, "decisión": 0.08,

		// Action / commitment language
		"decid": 0.09, "decidimos": 0.10, "elegimos": 0.09, "acordamos": 0.09,
		"fijamos": 0.08, "establecimos": 0.08,

		// Reflective / justification language
		"porque": 0.07, "por que": 0.07, "razón": 0.08, "reason": 0.07,
		"tradeoff": 0.08, "trade-off": 0.08, "alternativa": 0.07,
		"mejor": 0.06, "mejor enfoque": 0.07,

		// Common engineering value signals
		"fixed": 0.07, "fix": 0.06, "resolvimos": 0.07,
		"importante": 0.06, "critical": 0.07, "crítico": 0.07,
	}

	// Decision types that are inherently high-value for long-term memory
	highValueTypes := map[string]float64{
		"architecture": 0.12,
		"security":     0.11,
		"performance":  0.10,
		"convention":   0.08,
		"fix":          0.07,
	}

	for _, session := range m.sessions {
		for _, mem := range session.Memories {
			if mem.Promoted {
				continue
			}

			score := 0.0

			// === Structured decision quality (core signal) ===
			if mem.DecisionType != "" {
				score += 0.30
				if bonus, ok := highValueTypes[strings.ToLower(mem.DecisionType)]; ok {
					score += bonus
				}
			}
			if mem.Reason != "" {
				score += 0.22
			}
			if len(mem.Alternatives) > 0 {
				score += 0.10 // thoughtful decision (considered options)
			}

			// === Usage (LLM has found it valuable repeatedly) ===
			if mem.Usage >= 10 {
				score += 0.18
			} else if mem.Usage >= 6 {
				score += 0.13
			} else if mem.Usage >= 3 {
				score += 0.08
			}

			// === Recency (more recent = more likely still relevant) ===
			ageHours := time.Since(mem.Timestamp).Hours()
			if ageHours < 24 {
				score += 0.14
			} else if ageHours < 72 {
				score += 0.10
			} else if ageHours < 168 { // 1 week
				score += 0.06
			} else if ageHours < 720 { // 30 days
				score += 0.02
			}

			// === Content & tag signals (weighted) ===
			contentLower := strings.ToLower(mem.Content + " " + mem.Reason)
			for kw, weight := range weightedKeywords {
				if strings.Contains(contentLower, kw) {
					score += weight
				}
			}

			// Tag quality
			for _, tag := range mem.Tags {
				t := strings.ToLower(tag)
				if t == "architecture" || t == "security" || t == "performance" || t == "convention" || t == "decision" {
					score += 0.06
				} else if t == "fix" || t == "bug" || t == "important" {
					score += 0.04
				}
			}

			// Cap and apply minimum threshold
			if score > 0.98 {
				score = 0.98
			}

			if score > 0.22 { // raised threshold for higher quality suggestions
				memCopy := mem
				candidates = append(candidates, scoredMem{&memCopy, score})
			}
		}
	}

	// Sort by score descending (best promotion candidates first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Return top N
	result := make([]*Memory, 0, limit)
	for i := 0; i < len(candidates) && i < limit; i++ {
		result = append(result, candidates[i].mem)
	}

	return result, nil
}
