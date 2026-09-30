package continuity

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/memory"
)

func TestHandoffRoundTripAndConflict(t *testing.T) {
	root := t.TempDir()
	if _, err := exec.LookPath("git"); err == nil {
		cmd := exec.Command("git", "init")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init: %v\n%s", err, out)
		}
		branch := exec.Command("git", "checkout", "-b", "feature")
		branch.Dir = root
		_ = branch.Run()
	}
	storage := filepath.Join(t.TempDir(), "memory")
	mem, err := memory.New(config.MemoryConfig{Enabled: true, StoragePath: storage, MaxEntries: 20, SessionTTLDays: 30}, "other")
	if err != nil {
		t.Fatal(err)
	}
	svc := New(storage, mem)
	first, err := svc.Save(Input{Path: root, SourceClient: "claude", Objective: "share memory", NextStep: "open OpenCode"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != 1 {
		t.Fatalf("revision = %d", first.Revision)
	}
	second, err := svc.Save(Input{
		Path: root, HandoffID: first.HandoffID, ExpectedRevision: 1,
		SourceClient: "opencode", Objective: "share memory", NextStep: "run tests",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Revision != 2 {
		t.Fatalf("revision = %d", second.Revision)
	}
	_, err = svc.Save(Input{
		Path: root, HandoffID: first.HandoffID, ExpectedRevision: 1,
		SourceClient: "claude", Objective: "stale", NextStep: "do not overwrite",
	})
	conflict, ok := IsConflict(err)
	if !ok {
		t.Fatalf("expected conflict, got %v", err)
	}
	if conflict.Current.Revision != 2 || conflict.SavedAs == "" {
		t.Fatalf("conflict = %+v", conflict)
	}
	view, err := svc.Resume(root, "", first.HandoffID, "", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if view.Handoff == nil || view.Handoff.NextStep != "run tests" {
		t.Fatalf("resumed %#v project=%+v candidates=%d", view.Handoff, view.Project, len(view.Candidates))
	}
	text := FormatResume(view, 1000)
	if !strings.Contains(text, "run tests") || !strings.Contains(text, first.ProjectID) {
		t.Fatalf("formatted resume missing state:\n%s", text)
	}
}

func TestResumeDoesNotSelectAnotherBranch(t *testing.T) {
	root := t.TempDir()
	storage := filepath.Join(t.TempDir(), "memory")
	svc := New(storage, nil)
	saved, err := svc.Save(Input{Path: root, HandoffID: "other-work", Objective: "different task", NextStep: "wait"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		view, err := svc.Resume(root, saved.ProjectID, "", "", 500)
		if err != nil {
			t.Fatal(err)
		}
		if view.Handoff == nil {
			t.Fatal("without git branch information the only handoff should be selectable by id, not rejected")
		}
		return
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	exec.Command("git", "-C", root, "checkout", "-b", "main").Run()
	view, err := svc.Resume(root, "", "", "", 500)
	if err != nil {
		t.Fatal(err)
	}
	if view.Handoff != nil && view.Handoff.Branch != "" && view.Project.Branch != "" && view.Handoff.Branch != view.Project.Branch {
		t.Fatalf("selected handoff from another branch: %+v", view.Handoff)
	}
}
