package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSameFolderNamesStayDistinct(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "same")
	second := filepath.Join(root, "nested", "same")
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0o755); err != nil {
		t.Fatal(err)
	}
	storage := filepath.Join(root, "memory")
	a, err := Resolve(storage, first, "", true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve(storage, second, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" || b.ID == "" || a.ID == b.ID {
		t.Fatalf("expected distinct ids, got %q and %q", a.ID, b.ID)
	}
}

func TestSubdirectoryResolvesGitRoot(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	sub := filepath.Join(root, "internal", "memory")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	storage := filepath.Join(t.TempDir(), "memory")
	resolved, err := Resolve(storage, sub, "", false)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(resolved.Root) != filepath.Clean(want) {
		t.Fatalf("root = %s, want %s", resolved.Root, want)
	}
	if resolved.Persisted {
		t.Fatal("lookup must not create a project identity")
	}
}
