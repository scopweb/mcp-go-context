package continuity

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// Checkpoint is the short state extracted from a client transcript.
type Checkpoint struct {
	Objective string
	NextStep  string
	Pending   []string
}

// ExtractCheckpoint reads a JSONL transcript and keeps the last real user and assistant text.
// Harness-injected wrappers are ignored. Missing or unreadable files return an empty checkpoint.
func ExtractCheckpoint(path string) (Checkpoint, error) {
	f, err := os.Open(path)
	if err != nil {
		return Checkpoint{}, err
	}
	defer f.Close()
	var user, assistant string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		role, text := transcriptText(line)
		if text == "" || isHarnessText(text) {
			continue
		}
		switch role {
		case "user":
			user = text
		case "assistant":
			assistant = text
		}
	}
	if err := sc.Err(); err != nil {
		return Checkpoint{}, err
	}
	cp := Checkpoint{
		Objective: clipText(user, 400),
		NextStep:  clipText(assistant, 400),
	}
	if cp.Objective == "" {
		cp.Objective = "Automatic checkpoint"
	}
	if cp.NextStep == "" {
		cp.NextStep = "Review the last session and continue from the confirmed handoff."
	}
	return cp, nil
}

func transcriptText(line string) (string, string) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return "", ""
	}
	role, _ := raw["role"].(string)
	if role == "" {
		role, _ = raw["type"].(string)
	}
	if msg, ok := raw["message"].(map[string]any); ok {
		if r, _ := msg["role"].(string); r != "" {
			role = r
		}
		if text := contentText(msg["content"]); text != "" {
			return role, text
		}
	}
	if text := contentText(raw["content"]); text != "" {
		return role, text
	}
	if text, _ := raw["text"].(string); strings.TrimSpace(text) != "" {
		return role, text
	}
	return role, ""
}

func contentText(v any) string {
	switch c := v.(type) {
	case string:
		return strings.TrimSpace(c)
	case []any:
		var parts []string
		for _, item := range c {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, _ := obj["text"].(string); strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func isHarnessText(text string) bool {
	trimmed := strings.TrimSpace(text)
	prefixes := []string{
		"<command-",
		"<local-command",
		"<task-notification",
		"<skill",
		"Caveat: The messages below",
		"<system-reminder>",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func clipText(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n]
}
