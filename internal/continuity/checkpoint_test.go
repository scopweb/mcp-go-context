package continuity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCheckpointIgnoresHarnessNoise(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	body := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<local-command-stdout>noise</local-command-stdout>"}]}}`,
		`{"type":"user","message":{"role":"user","content":"Implement the three mechanisms"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Next: wire the stop hook."}]}}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ExtractCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Objective != "Implement the three mechanisms" {
		t.Fatalf("objective = %q", got.Objective)
	}
	if !strings.Contains(got.NextStep, "stop hook") {
		t.Fatalf("next = %q", got.NextStep)
	}
}

func TestSaveAutoDoesNotReplaceConfirmedHandoff(t *testing.T) {
	root := t.TempDir()
	storage := filepath.Join(t.TempDir(), "memory")
	svc := New(storage, nil)
	saved, err := svc.Save(Input{Path: root, Objective: "confirmed work", NextStep: "keep this"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveAuto(root, "claude-hook", "session topic", "hook next step", nil); err != nil {
		t.Fatal(err)
	}
	view, err := svc.Resume(root, "", "", "", "wake", 800)
	if err != nil {
		t.Fatal(err)
	}
	if view.Handoff == nil || view.Handoff.HandoffID != saved.HandoffID || view.Handoff.NextStep != "keep this" {
		t.Fatalf("confirmed handoff changed: %#v", view.Handoff)
	}
	if view.Auto == nil || view.Auto.NextStep != "hook next step" {
		t.Fatalf("auto checkpoint missing: %#v", view.Auto)
	}
	text := FormatResume(view, 800)
	if !strings.Contains(text, "Wake-up") || !strings.Contains(text, "hook next step") {
		t.Fatalf("wake brief:\n%s", text)
	}
	if strings.Contains(text, "Completed") {
		t.Fatalf("wake brief included full sections:\n%s", text)
	}
}
