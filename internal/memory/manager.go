package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
)

// Manager handles conversation memory persistence
type Manager struct {
	config   config.MemoryConfig
	sessions map[string]*Session
	mu       sync.RWMutex
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
}

// New creates a new memory manager
func New(cfg config.MemoryConfig) (*Manager, error) {
	m := &Manager{
		config:   cfg,
		sessions: make(map[string]*Session),
	}

	if cfg.Enabled {
		// Ensure storage directory exists
		storageDir := filepath.Dir(cfg.StoragePath)
		if err := os.MkdirAll(storageDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create memory storage directory: %w", err)
		}

		// Load existing sessions
		if err := m.loadSessions(); err != nil {
			return nil, fmt.Errorf("failed to load sessions: %w", err)
		}

		// Start cleanup routine
		go m.cleanupRoutine()
	}

	return m, nil
}

// Store saves a memory item
func (m *Manager) Store(key, content string, tags []string) error {
	if !m.config.Enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create current session
	session := m.getCurrentSession()

	// Store memory
	session.Memories[key] = Memory{
		Key:       key,
		Content:   content,
		Tags:      tags,
		Timestamp: time.Now(),
		Usage:     0,
	}

	// Save to disk
	return m.saveSession(session)
}

// Retrieve gets a memory item by key
func (m *Manager) Retrieve(key string) (*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search across all sessions
	for _, session := range m.sessions {
		if memory, exists := session.Memories[key]; exists {
			memory.Usage++
			return &memory, nil
		}
	}

	return nil, fmt.Errorf("memory not found: %s", key)
}

// Search finds memories by tags or content
func (m *Manager) Search(query string, tags []string) ([]*Memory, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("memory disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*Memory

	for _, session := range m.sessions {
		for _, memory := range session.Memories {
			// Check tags
			if len(tags) > 0 {
				matched := false
				for _, tag := range tags {
					for _, memTag := range memory.Tags {
						if tag == memTag {
							matched = true
							break
						}
					}
				}
				if !matched {
					continue
				}
			}

			// Check content
			if query != "" && !contains(memory.Content, query) && !contains(memory.Key, query) {
				continue
			}

			memoryCopy := memory
			results = append(results, &memoryCopy)
		}
	}

	return results, nil
}

// GetRecentMemories returns recent memories
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

	// Sort by timestamp (implement sorting)
	// For now, just return up to limit
	if len(allMemories) > limit {
		return allMemories[:limit], nil
	}

	return allMemories, nil
}

// Clear removes all memories
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*Session)

	// Clear storage
	storageDir := filepath.Dir(m.config.StoragePath)
	entries, err := os.ReadDir(storageDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			os.Remove(filepath.Join(storageDir, entry.Name()))
		}
	}

	return nil
}

// Private methods

func (m *Manager) getCurrentSession() *Session {
	// Simple implementation: use a single "current" session
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

func (m *Manager) loadSessions() error {
	storageDir := filepath.Dir(m.config.StoragePath)
	
	entries, err := os.ReadDir(storageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			sessionID := strings.TrimSuffix(entry.Name(), ".json")

			data, err := os.ReadFile(filepath.Join(storageDir, entry.Name()))
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

	storageDir := filepath.Dir(m.config.StoragePath)
	filename := filepath.Join(storageDir, session.ID+".json")
	return os.WriteFile(filename, data, 0644)
}

func (m *Manager) cleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -m.config.SessionTTLDays)
	storageDir := filepath.Dir(m.config.StoragePath)

	for id, session := range m.sessions {
		if session.LastUsed.Before(cutoff) {
			delete(m.sessions, id)
			os.Remove(filepath.Join(storageDir, id+".json"))
		}
	}

	// Limit total sessions
	if len(m.sessions) > m.config.MaxSessions {
		// TODO: Implement LRU eviction
	}
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
