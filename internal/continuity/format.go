package continuity

import (
	"fmt"
	"strings"
	"time"

	"github.com/scopweb/mcp-go-context/internal/memory"
)

// FormatResume renders a bounded continuity brief.
func FormatResume(view ResumeView, maxTokens int) string {
	var b strings.Builder
	b.WriteString("# Continuity\n\n")
	if view.Project.ID == "" {
		b.WriteString("No stored project identity for this path.\n")
		if view.Project.Root != "" {
			b.WriteString(fmt.Sprintf("Detected root: `%s`\n", view.Project.Root))
		}
		b.WriteString("\nSave a handoff or decision with this path to start shared continuity. Do not invent prior work.\n")
		return clip(b.String(), maxTokens)
	}
	b.WriteString(fmt.Sprintf("Project: `%s` (%s)\n", view.Project.ID, view.Project.Name))
	if view.Project.Root != "" {
		b.WriteString(fmt.Sprintf("Root: `%s`\n", view.Project.Root))
	}
	if view.Project.Branch != "" || view.Project.Commit != "" {
		b.WriteString(fmt.Sprintf("Observed: branch `%s`, commit `%s`\n", empty(view.Project.Branch), shortCommit(view.Project.Commit)))
	}
	b.WriteString("\nFiles and Git remain the source of truth for code. This brief is the last shared work state.\n\n")

	if view.Handoff == nil {
		b.WriteString("## Handoff\n\n")
		if len(view.Candidates) == 0 {
			b.WriteString("No handoff for the current branch.\n\n")
		} else {
			b.WriteString("No handoff matches the current branch. Candidates were not selected automatically:\n\n")
			for _, item := range view.Candidates {
				b.WriteString(fmt.Sprintf("- `%s` branch `%s` revision %d updated %s — %s\n",
					item.HandoffID, empty(item.Branch), item.Revision, item.UpdatedAt.Format(time.RFC3339), oneLine(item.NextStep)))
			}
			b.WriteString("\nCall resume-context with handoffId to open one.\n\n")
		}
	} else {
		h := view.Handoff
		b.WriteString("## Handoff\n\n")
		b.WriteString(fmt.Sprintf("ID: `%s` | revision %d | updated %s", h.HandoffID, h.Revision, h.UpdatedAt.Format(time.RFC3339)))
		if h.SourceClient != "" {
			b.WriteString(fmt.Sprintf(" | source `%s`", h.SourceClient))
		}
		b.WriteString("\n")
		if h.Branch != "" || h.Commit != "" {
			b.WriteString(fmt.Sprintf("Saved worktree: branch `%s`, commit `%s`\n", empty(h.Branch), shortCommit(h.Commit)))
		}
		if view.Drift != "" {
			b.WriteString("Drift: " + view.Drift + "\n")
		}
		writeSection(&b, "Objective", h.Objective)
		writeList(&b, "Completed", h.Completed)
		writeList(&b, "Verification", h.Verification)
		writeList(&b, "Pending", h.Pending)
		writeSection(&b, "Next step", h.NextStep)
		writeList(&b, "References", h.References)
		b.WriteString("\n")
	}

	if len(view.Decisions) > 0 {
		b.WriteString("## Relevant decisions\n\n")
		for _, mem := range view.Decisions {
			writeMemory(&b, mem)
		}
	}
	if len(view.Memories) > 0 {
		b.WriteString("## Related memories\n\n")
		for _, mem := range view.Memories {
			writeMemory(&b, mem)
		}
	}
	return clip(b.String(), maxTokens)
}

func writeSection(b *strings.Builder, title, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	b.WriteString("\n### " + title + "\n\n")
	b.WriteString(text)
	b.WriteString("\n")
}

func writeList(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("\n### " + title + "\n\n")
	for _, item := range items {
		b.WriteString("- " + item + "\n")
	}
}

func writeMemory(b *strings.Builder, mem *memory.Memory) {
	if mem == nil {
		return
	}
	b.WriteString(fmt.Sprintf("- `%s`", mem.Key))
	if mem.DecisionType != "" {
		b.WriteString(" (" + mem.DecisionType + ")")
	}
	b.WriteString(": " + oneLine(truncate(mem.Content, 280)) + "\n")
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "(empty)"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func empty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func shortCommit(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	if s == "" {
		return "-"
	}
	return s
}

func clip(s string, maxTokens int) string {
	if maxTokens <= 0 {
		maxTokens = 2000
	}
	maxChars := maxTokens * 4
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars] + "\n\n... (truncated to token budget)\n"
}
