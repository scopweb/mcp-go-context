package continuity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scopweb/mcp-go-context/internal/memory"
	"github.com/scopweb/mcp-go-context/internal/project"
	"github.com/scopweb/mcp-go-context/internal/storage"
)

const schemaVersion = 1

// AutoHandoffID is the slot written by client hooks. It never replaces a confirmed handoff.
const AutoHandoffID = "auto"

const (
	maxText  = 4000
	maxItem  = 500
	maxItems = 30
	maxHist  = 5
)

// Handoff is one recoverable work context, not a durable decision.
type Handoff struct {
	SchemaVersion int       `json:"schemaVersion"`
	ProjectID     string    `json:"projectId"`
	HandoffID     string    `json:"handoffId"`
	Revision      int       `json:"revision"`
	UpdatedAt     time.Time `json:"updatedAt"`
	SourceClient  string    `json:"sourceClient,omitempty"`
	Branch        string    `json:"branch,omitempty"`
	Commit        string    `json:"commit,omitempty"`
	Worktree      string    `json:"worktree,omitempty"`
	Objective     string    `json:"objective,omitempty"`
	Completed     []string  `json:"completed,omitempty"`
	Verification  []string  `json:"verification,omitempty"`
	Pending       []string  `json:"pending,omitempty"`
	NextStep      string    `json:"nextStep,omitempty"`
	References    []string  `json:"references,omitempty"`
	History       []Handoff `json:"history,omitempty"`
}

// Input is the assistant-supplied handoff. Git fields are observed by the server.
type Input struct {
	Path             string
	ProjectID        string
	HandoffID        string
	ExpectedRevision int
	SourceClient     string
	Objective        string
	Completed        []string
	Verification     []string
	Pending          []string
	NextStep         string
	References       []string
}

// Result is a successful save.
type Result struct {
	ProjectID string
	HandoffID string
	Revision  int
	Branch    string
	Commit    string
}

// ConflictError preserves both sides of a concurrent update.
type ConflictError struct {
	Current Handoff
	SavedAs string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("handoff revision conflict: current revision %d kept; incoming saved as %s", e.Current.Revision, e.SavedAs)
}

// ResumeView is the continuity payload for another client.
type ResumeView struct {
	Project    project.Resolved
	Handoff    *Handoff
	Auto       *Handoff
	Candidates []Handoff
	Drift      string
	Decisions  []*memory.Memory
	Memories   []*memory.Memory
	NoIdentity bool
	NoHandoff  bool
	Depth      string
}

// Service stores handoffs beside the shared memory directory.
type Service struct {
	storage string
	memory  *memory.Manager
}

func New(storagePath string, mem *memory.Manager) *Service {
	return &Service{storage: storagePath, memory: mem}
}

func (s *Service) Resolve(path, projectID string, create bool) (project.Resolved, error) {
	return project.Resolve(s.storage, path, projectID, create)
}

func (s *Service) Save(in Input) (Result, error) {
	if err := validate(in); err != nil {
		return Result{}, err
	}
	resolved, err := project.Resolve(s.storage, in.Path, in.ProjectID, true)
	if err != nil {
		return Result{}, err
	}
	if resolved.ID == "" {
		return Result{}, fmt.Errorf("project identity was not created")
	}
	id := in.HandoffID
	if id == "" {
		id = defaultHandoffID(resolved.Branch)
	}
	id = sanitize(id)

	unlock, err := storage.Lock(s.storage)
	if err != nil {
		return Result{}, err
	}
	defer unlock()

	current, found, err := load(s.storage, resolved.ID, id)
	if err != nil {
		return Result{}, err
	}
	next := build(resolved, id, in)
	if found {
		if in.ExpectedRevision != current.Revision {
			savedAs, err := s.saveConflict(resolved.ID, next)
			if err != nil {
				return Result{}, err
			}
			return Result{}, &ConflictError{Current: stripHistory(current), SavedAs: savedAs}
		}
		prev := stripHistory(current)
		next.History = append([]Handoff{prev}, current.History...)
		if len(next.History) > maxHist {
			next.History = next.History[:maxHist]
		}
		next.Revision = current.Revision + 1
	} else {
		if in.ExpectedRevision != 0 {
			return Result{}, fmt.Errorf("handoff %s does not exist", id)
		}
		next.Revision = 1
	}
	next.UpdatedAt = time.Now().UTC()
	if err := save(s.storage, next); err != nil {
		return Result{}, err
	}
	return Result{
		ProjectID: resolved.ID,
		HandoffID: next.HandoffID,
		Revision:  next.Revision,
		Branch:    next.Branch,
		Commit:    next.Commit,
	}, nil
}

// SaveAuto replaces the hook checkpoint without creating a conflict artifact.
func (s *Service) SaveAuto(path, client, objective, nextStep string, pending []string) (Result, error) {
	in := Input{
		Path:         path,
		HandoffID:    AutoHandoffID,
		SourceClient: client,
		Objective:    objective,
		Pending:      pending,
		NextStep:     nextStep,
	}
	if err := validate(in); err != nil {
		return Result{}, err
	}
	resolved, err := project.Resolve(s.storage, path, "", true)
	if err != nil {
		return Result{}, err
	}
	unlock, err := storage.Lock(s.storage)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	current, found, err := load(s.storage, resolved.ID, AutoHandoffID)
	if err != nil {
		return Result{}, err
	}
	next := build(resolved, AutoHandoffID, in)
	next.Revision = 1
	if found {
		prev := stripHistory(current)
		next.History = append([]Handoff{prev}, current.History...)
		if len(next.History) > maxHist {
			next.History = next.History[:maxHist]
		}
		next.Revision = current.Revision + 1
	}
	next.UpdatedAt = time.Now().UTC()
	if err := save(s.storage, next); err != nil {
		return Result{}, err
	}
	return Result{ProjectID: resolved.ID, HandoffID: AutoHandoffID, Revision: next.Revision, Branch: next.Branch, Commit: next.Commit}, nil
}

func (s *Service) Resume(path, projectID, handoffID, query, depth string, maxTokens int) (ResumeView, error) {
	resolved, err := project.Resolve(s.storage, path, projectID, false)
	if err != nil {
		return ResumeView{}, err
	}
	if depth == "" {
		depth = "wake"
	}
	view := ResumeView{Project: resolved, Depth: depth}
	if !resolved.Persisted || resolved.ID == "" {
		view.NoIdentity = true
		view.NoHandoff = true
		return view, nil
	}
	all, err := list(s.storage, resolved.ID)
	if err != nil {
		return ResumeView{}, err
	}
	for _, item := range all {
		if item.HandoffID == AutoHandoffID {
			copy := stripHistory(item)
			view.Auto = &copy
			break
		}
	}
	selected, candidates, ok := selectHandoff(all, handoffID, resolved.Branch)
	if ok {
		copy := selected
		view.Handoff = &copy
		view.Drift = drift(selected, resolved)
	} else {
		view.NoHandoff = len(all) == 0
		view.Candidates = candidates
	}
	if s.memory != nil {
		if query != "" {
			found, err := s.memory.SearchWithProject(query, nil, resolved.ID)
			if err != nil {
				return ResumeView{}, err
			}
			view.Memories = limitMemories(found, 5)
		}
		found, err := s.memory.SearchDecisionsScoped("", "", resolved.ID, 0)
		if err != nil {
			return ResumeView{}, err
		}
		for _, mem := range found {
			if mem.DecisionType == "" {
				continue
			}
			view.Decisions = append(view.Decisions, mem)
			if len(view.Decisions) == 5 {
				break
			}
		}
	}
	if maxTokens > 0 && maxTokens < 200 {
		view.Decisions = limitMemories(view.Decisions, 2)
		view.Memories = limitMemories(view.Memories, 2)
	}
	return view, nil
}

func (s *Service) saveConflict(projectID string, incoming Handoff) (string, error) {
	id := sanitize(fmt.Sprintf("%s-conflict-%d", incoming.HandoffID, time.Now().Unix()))
	incoming.HandoffID = id
	incoming.ProjectID = projectID
	incoming.Revision = 1
	incoming.History = nil
	incoming.UpdatedAt = time.Now().UTC()
	if err := save(s.storage, incoming); err != nil {
		return "", err
	}
	return id, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.Path) == "" && strings.TrimSpace(in.ProjectID) == "" {
		return fmt.Errorf("path or projectId is required")
	}
	if strings.TrimSpace(in.Objective) == "" && strings.TrimSpace(in.NextStep) == "" {
		return fmt.Errorf("objective or nextStep is required")
	}
	if len(in.Objective) > maxText || len(in.NextStep) > maxItem {
		return fmt.Errorf("handoff field exceeds size limit")
	}
	for _, list := range [][]string{in.Completed, in.Verification, in.Pending, in.References} {
		if len(list) > maxItems {
			return fmt.Errorf("handoff list exceeds %d items", maxItems)
		}
		for _, item := range list {
			if len(item) > maxItem {
				return fmt.Errorf("handoff list item exceeds %d characters", maxItem)
			}
		}
	}
	return nil
}

func build(resolved project.Resolved, id string, in Input) Handoff {
	return Handoff{
		SchemaVersion: schemaVersion,
		ProjectID:     resolved.ID,
		HandoffID:     id,
		SourceClient:  strings.TrimSpace(in.SourceClient),
		Branch:        resolved.Branch,
		Commit:        resolved.Commit,
		Worktree:      resolved.Root,
		Objective:     strings.TrimSpace(in.Objective),
		Completed:     cleanList(in.Completed),
		Verification:  cleanList(in.Verification),
		Pending:       cleanList(in.Pending),
		NextStep:      strings.TrimSpace(in.NextStep),
		References:    cleanList(in.References),
	}
}

func selectHandoff(all []Handoff, requested, branch string) (Handoff, []Handoff, bool) {
	if requested != "" {
		want := sanitize(requested)
		for _, item := range all {
			if item.HandoffID == want {
				return item, nil, true
			}
		}
		return Handoff{}, summarize(all), false
	}
	pool := currentHandoffs(all)
	var matches []Handoff
	for _, item := range pool {
		if branch != "" && item.Branch == branch {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 && len(pool) == 1 && (branch == "" || pool[0].Branch == "" || pool[0].Branch == branch) {
		matches = pool
	}
	if len(matches) == 1 {
		return matches[0], nil, true
	}
	if len(matches) > 1 {
		preferred := defaultHandoffID(branch)
		var preferredMatch *Handoff
		for i := range matches {
			if matches[i].HandoffID == preferred {
				preferredMatch = &matches[i]
				break
			}
		}
		if preferredMatch != nil {
			return *preferredMatch, nil, true
		}
		best := matches[0]
		for _, item := range matches[1:] {
			if item.UpdatedAt.After(best.UpdatedAt) {
				best = item
			}
		}
		return best, nil, true
	}
	return Handoff{}, summarize(all), false
}

func currentHandoffs(all []Handoff) []Handoff {
	out := make([]Handoff, 0, len(all))
	for _, item := range all {
		if isConflictHandoff(item.HandoffID) || item.HandoffID == AutoHandoffID {
			continue
		}
		out = append(out, item)
	}
	return out
}

func isConflictHandoff(id string) bool {
	const marker = "-conflict-"
	i := strings.LastIndex(id, marker)
	if i < 0 {
		return false
	}
	suffix := id[i+len(marker):]
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func summarize(all []Handoff) []Handoff {
	out := make([]Handoff, 0, len(all))
	for _, item := range all {
		out = append(out, stripHistory(item))
	}
	if len(out) > 8 {
		return out[:8]
	}
	return out
}

func drift(saved Handoff, now project.Resolved) string {
	if saved.Branch != "" && now.Branch != "" && saved.Branch != now.Branch {
		return fmt.Sprintf("saved on branch %s, current branch is %s", saved.Branch, now.Branch)
	}
	if saved.Commit != "" && now.Commit != "" && saved.Commit != now.Commit {
		return fmt.Sprintf("saved at %s, current commit is %s", short(saved.Commit), short(now.Commit))
	}
	return ""
}

func short(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

func defaultHandoffID(branch string) string {
	if branch == "" || branch == "HEAD" {
		return "default"
	}
	return sanitize(branch)
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prev := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			prev = false
			continue
		}
		if !prev && b.Len() > 0 {
			b.WriteByte('-')
			prev = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "default"
	}
	if len(out) > 80 {
		return out[:80]
	}
	return out
}

func cleanList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func stripHistory(h Handoff) Handoff {
	h.History = nil
	return h
}

func limitMemories(items []*memory.Memory, n int) []*memory.Memory {
	if len(items) > n {
		return items[:n]
	}
	return items
}

func dir(storagePath, projectID string) string {
	return filepath.Join(storagePath, "handoffs", projectID)
}

func path(storagePath, projectID, id string) string {
	return filepath.Join(dir(storagePath, projectID), id+".json")
}

func load(storagePath, projectID, id string) (Handoff, bool, error) {
	data, err := os.ReadFile(path(storagePath, projectID, id))
	if err != nil {
		if os.IsNotExist(err) {
			return Handoff{}, false, nil
		}
		return Handoff{}, false, err
	}
	var h Handoff
	if err := json.Unmarshal(data, &h); err != nil {
		return Handoff{}, false, fmt.Errorf("read handoff: %w", err)
	}
	return h, true, nil
}

func list(storagePath, projectID string) ([]Handoff, error) {
	entries, err := os.ReadDir(dir(storagePath, projectID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Handoff
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		item, found, err := load(storagePath, projectID, id)
		if err != nil {
			return nil, err
		}
		if found {
			out = append(out, item)
		}
	}
	return out, nil
}

func save(storagePath string, h Handoff) error {
	h.SchemaVersion = schemaVersion
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir(storagePath, h.ProjectID), 0o755); err != nil {
		return err
	}
	return storage.WriteAtomic(path(storagePath, h.ProjectID, h.HandoffID), append(data, '\n'))
}

// IsConflict reports whether err is a retained concurrent update.
func IsConflict(err error) (*ConflictError, bool) {
	var conflict *ConflictError
	if errors.As(err, &conflict) {
		return conflict, true
	}
	return nil, false
}
