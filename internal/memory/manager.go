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
	config   config.MemoryConfig
	sessions map[string]*Session
	mu       sync.RWMutex
	// Inverted indexes for fast search
	tagIndex     map[string]map[string]bool // tag -> set of memory keys
	wordIndex    map[string]map[string]bool // word -> set of memory keys
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
	// Decision-specific fields
	DecisionType string    `json:"decisionType,omitempty"` // architecture, fix, approach, etc.
	Reason      string    `json:"reason,omitempty"`       // why this decision was made
	Alternatives []string `json:"alternatives,omitempty"`  // what else was considered
}

// SearchResult represents a scored search result
type SearchResult struct {
	Memory *Memory
	Score  float64
}

// New creates a new memory manager
func New(cfg config.MemoryConfig) (*Manager, error) {
	m := &Manager{
		config:   cfg,
		sessions: make(map[string]*Session),
		tagIndex: make(map[string]map[string]bool),
		wordIndex: make(map[string]map[string]bool),
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
		DecisionType: decisionType,
		Reason:       reason,
		Alternatives: alternatives,
	}
	session.Memories[key] = mem

	// Update indexes
	m.addToIndexes(key, mem)

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
			return &memory, nil
		}
	}

	return nil, fmt.Errorf("memory not found: %s", key)
}

// Search finds memories with improved ranking
func (m *Manager) Search(query string, tags []string) ([]*Memory, error) {
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

	// Score and filter results
	var results []SearchResult
	for _, mem := range candidates {
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
