package project

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/scopweb/mcp-go-context/internal/memory"
	"github.com/scopweb/mcp-go-context/internal/storage"
)

const schemaVersion = 2

// Resolved is a stable project identity plus the observed worktree.
type Resolved struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Roots      []string `json:"roots"`
	CommonDir  string   `json:"commonDir,omitempty"`
	RemoteURL  string   `json:"remoteURL,omitempty"`
	Root       string   `json:"root"`
	Branch     string   `json:"branch,omitempty"`
	Commit     string   `json:"commit,omitempty"`
	Persisted  bool     `json:"persisted"`
	CreatedNow bool     `json:"createdNow,omitempty"`
}

type record struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Roots     []string  `json:"roots"`
	CommonDir string    `json:"commonDir,omitempty"`
	RemoteURL string    `json:"remoteURL,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type registry struct {
	SchemaVersion int      `json:"schemaVersion"`
	Projects      []record `json:"projects"`
}

// Resolve maps a working path to a project. create persists a new identity or a new root alias.
// A lookup never writes inside the repository.
func Resolve(storagePath, workPath, explicitID string, create bool) (Resolved, error) {
	if strings.TrimSpace(workPath) == "" && strings.TrimSpace(explicitID) == "" {
		return Resolved{}, fmt.Errorf("path or projectId is required")
	}
	observed, err := observe(workPath)
	if err != nil && strings.TrimSpace(explicitID) == "" {
		return Resolved{}, err
	}
	unlock, err := storage.Lock(storagePath)
	if err != nil {
		return Resolved{}, err
	}
	defer unlock()

	reg, err := loadRegistry(storagePath)
	if err != nil {
		return Resolved{}, err
	}
	if explicitID != "" {
		if bound, ok := findByRoot(reg, observed); ok && bound.ID != sanitizeID(explicitID) {
			return Resolved{}, fmt.Errorf("path is already bound to project %s", bound.ID)
		}
	}
	if rec, ok := findRecord(reg, observed, explicitID); ok {
		if create && observed.Root != "" && !hasRoot(rec, observed.Root) {
			rec.Roots = append(rec.Roots, observed.Root)
			if rec.CommonDir == "" {
				rec.CommonDir = observed.CommonDir
			}
			if err := putRecord(&reg, rec); err != nil {
				return Resolved{}, err
			}
			if err := saveRegistry(storagePath, reg); err != nil {
				return Resolved{}, err
			}
		}
		return toResolved(rec, observed, true, false), nil
	}
	if !create {
		name := memory.InferProject(observed.Root)
		return Resolved{
			ID:        "",
			Name:      name,
			Root:      observed.Root,
			Branch:    observed.Branch,
			Commit:    observed.Commit,
			CommonDir: observed.CommonDir,
			RemoteURL: observed.RemoteURL,
			Persisted: false,
		}, nil
	}
	rec := record{
		ID:        newID(reg, explicitID, observed),
		Name:      memory.InferProject(observed.Root),
		Roots:     nonEmpty(observed.Root),
		CommonDir: observed.CommonDir,
		RemoteURL: observed.RemoteURL,
		CreatedAt: time.Now().UTC(),
	}
	if explicitID != "" {
		rec.ID = sanitizeID(explicitID)
	}
	if rec.Name == "" || rec.Name == "default" {
		rec.Name = rec.ID
	}
	reg.SchemaVersion = schemaVersion
	reg.Projects = append(reg.Projects, rec)
	if err := saveRegistry(storagePath, reg); err != nil {
		return Resolved{}, err
	}
	return toResolved(rec, observed, true, true), nil
}

type observation struct {
	Root      string
	Branch    string
	Commit    string
	CommonDir string
	RemoteURL string
}

func observe(workPath string) (observation, error) {
	if strings.TrimSpace(workPath) == "" {
		return observation{}, nil
	}
	abs, err := filepath.Abs(workPath)
	if err != nil {
		return observation{}, fmt.Errorf("resolve path: %w", err)
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil {
		return observation{}, fmt.Errorf("stat path: %w", err)
	}
	if !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	obs := observation{Root: abs}
	if top := git(abs, "rev-parse", "--show-toplevel"); top != "" {
		obs.Root = filepath.Clean(top)
	}
	obs.Branch = git(obs.Root, "rev-parse", "--abbrev-ref", "HEAD")
	obs.Commit = git(obs.Root, "rev-parse", "HEAD")
	if common := git(obs.Root, "rev-parse", "--git-common-dir"); common != "" {
		if !filepath.IsAbs(common) {
			common = filepath.Join(obs.Root, common)
		}
		obs.CommonDir = filepath.Clean(common)
	}
	obs.RemoteURL = git(obs.Root, "remote", "get-url", "origin")
	return obs, nil
}

func git(dir string, args ...string) string {
	if dir == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func findRecord(reg registry, obs observation, explicitID string) (record, bool) {
	if strings.TrimSpace(explicitID) == "" {
		explicitID = ""
	} else {
		explicitID = sanitizeID(explicitID)
	}
	for _, rec := range reg.Projects {
		if explicitID != "" && rec.ID == explicitID {
			return rec, true
		}
	}
	if explicitID != "" {
		return record{}, false
	}
	for _, rec := range reg.Projects {
		if obs.CommonDir != "" && samePath(rec.CommonDir, obs.CommonDir) {
			return rec, true
		}
		if hasRoot(rec, obs.Root) {
			return rec, true
		}
	}
	return record{}, false
}

func findByRoot(reg registry, obs observation) (record, bool) {
	for _, rec := range reg.Projects {
		if obs.CommonDir != "" && samePath(rec.CommonDir, obs.CommonDir) {
			return rec, true
		}
		if hasRoot(rec, obs.Root) {
			return rec, true
		}
	}
	return record{}, false
}

func hasRoot(rec record, root string) bool {
	for _, existing := range rec.Roots {
		if samePath(existing, root) {
			return true
		}
	}
	return false
}

func putRecord(reg *registry, updated record) error {
	for i := range reg.Projects {
		if reg.Projects[i].ID == updated.ID {
			reg.Projects[i] = updated
			return nil
		}
	}
	return fmt.Errorf("project %s not found", updated.ID)
}

func newID(reg registry, explicitID string, obs observation) string {
	if explicitID != "" {
		return sanitizeID(explicitID)
	}
	base := memory.InferProject(obs.Root)
	if base == "" || base == "default" {
		base = "project"
	}
	if !idTaken(reg, base) {
		return base
	}
	sum := sha1.Sum([]byte(identityKey(obs)))
	return base + "-" + hex.EncodeToString(sum[:])[:6]
}

func identityKey(obs observation) string {
	if obs.CommonDir != "" {
		return "common:" + normalize(obs.CommonDir)
	}
	return "root:" + normalize(obs.Root)
}

func idTaken(reg registry, id string) bool {
	for _, rec := range reg.Projects {
		if rec.ID == id {
			return true
		}
	}
	return false
}

func toResolved(rec record, obs observation, persisted, created bool) Resolved {
	root := obs.Root
	if root == "" && len(rec.Roots) > 0 {
		root = rec.Roots[0]
	}
	return Resolved{
		ID:         rec.ID,
		Name:       rec.Name,
		Roots:      append([]string(nil), rec.Roots...),
		CommonDir:  rec.CommonDir,
		RemoteURL:  rec.RemoteURL,
		Root:       root,
		Branch:     obs.Branch,
		Commit:     obs.Commit,
		Persisted:  persisted,
		CreatedNow: created,
	}
}

func nonEmpty(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

func sanitizeID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "project"
	}
	if len(out) > 80 {
		return out[:80]
	}
	return out
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(normalize(a), normalize(b))
	}
	return normalize(a) == normalize(b)
}

func normalize(p string) string {
	return filepath.Clean(p)
}

func registryPath(storagePath string) string {
	return filepath.Join(storagePath, "projects.json")
}

func loadRegistry(storagePath string) (registry, error) {
	data, err := os.ReadFile(registryPath(storagePath))
	if err != nil {
		if os.IsNotExist(err) {
			return registry{SchemaVersion: schemaVersion}, nil
		}
		return registry{}, err
	}
	var reg registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return registry{}, fmt.Errorf("read project registry: %w", err)
	}
	if reg.SchemaVersion == 0 {
		reg.SchemaVersion = schemaVersion
	}
	return reg, nil
}

func saveRegistry(storagePath string, reg registry) error {
	reg.SchemaVersion = schemaVersion
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return storage.WriteAtomic(registryPath(storagePath), append(data, '\n'))
}
