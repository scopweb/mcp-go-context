package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
)

// Pre-compiled regexes for performance
var (
	validKeyRegex  = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	validFileRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+\.json$`)
)

// Manager handles conversation memory persistence
type Manager struct {
	config   config.MemoryConfig
	sessions map[string]*Session
	mu       sync.RWMutex
	// LRU: slice of session IDs ordered by last used (most recent at end)
	lru []string
	// index: map[string][]*Memory // Para búsquedas rápidas por tag o palabra clave (futuro)
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

	// Validación de clave: solo letras, números, guiones y guiones bajos, 1-64 chars
	if !validKeyRegex.MatchString(key) {
		return errors.New("invalid key format")
	}
	// Validación de tags: cada tag igual que key, máximo 10 tags
	if len(tags) > 10 {
		return errors.New("too many tags")
	}
	for _, tag := range tags {
		if !validKeyRegex.MatchString(tag) {
			return errors.New("invalid tag format")
		}
	}
	// Validación de contenido: máximo 4096 caracteres
	if len(content) > 4096 {
		return errors.New("content too long")
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

	// Actualizar LRU
	m.updateLRU(session.ID)

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

	// Search across all sessions (mejorable con índice futuro)
	for id, session := range m.sessions {
		if memory, exists := session.Memories[key]; exists {
			memory.Usage++
			// Actualizar LRU
			m.updateLRU(id)
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
	m.lru = nil

	// Clear storage
	storageDir := filepath.Dir(m.config.StoragePath)
	entries, err := os.ReadDir(storageDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" && validFileRegex.MatchString(entry.Name()) {
			os.Remove(filepath.Join(storageDir, entry.Name()))
		}
	}

	return nil
}

// Private methods

func (m *Manager) getCurrentSession() *Session {
	// Simple: usar una sola sesión "current" (mejorable para multiusuario)
	sessionID := "current"

	if session, exists := m.sessions[sessionID]; exists {
		session.LastUsed = time.Now()
		m.updateLRU(sessionID)
		return session
	}

	// Carga bajo demanda si existe en disco
	storageDir := filepath.Dir(m.config.StoragePath)
	filename := filepath.Join(storageDir, sessionID+".json")
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err == nil {
			var session Session
			if json.Unmarshal(data, &session) == nil {
				m.sessions[sessionID] = &session
				m.updateLRU(sessionID)
				session.LastUsed = time.Now()
				return &session
			}
		}
	}

	session := &Session{
		ID:        sessionID,
		StartTime: time.Now(),
		LastUsed:  time.Now(),
		Memories:  make(map[string]Memory),
	}
	m.sessions[sessionID] = session
	m.updateLRU(sessionID)
	return session
}

// updateLRU actualiza la lista LRU de sesiones
func (m *Manager) updateLRU(sessionID string) {
	// Elimina si ya existe
	for i, id := range m.lru {
		if id == sessionID {
			m.lru = append(m.lru[:i], m.lru[i+1:]...)
			break
		}
	}
	// Añade al final
	m.lru = append(m.lru, sessionID)
	// Limita tamaño
	if m.config.MaxSessions > 0 && len(m.lru) > m.config.MaxSessions {
		// Elimina la menos reciente
		removeID := m.lru[0]
		m.lru = m.lru[1:]
		delete(m.sessions, removeID)
		storageDir := filepath.Dir(m.config.StoragePath)
		filename := filepath.Join(storageDir, removeID+".json")
		os.Remove(filename)
	}
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
			filename := filepath.Join(storageDir, id+".json")
			if validFileRegex.MatchString(id + ".json") {
				os.Remove(filename)
			}
		}
	}

	// LRU eviction ya se maneja en updateLRU
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
