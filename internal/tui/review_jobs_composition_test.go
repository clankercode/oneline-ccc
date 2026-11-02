package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestReviewCompositionPreservesEveryPageValue(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		for _, theme := range []string{"dark", "light", "mono"} {
			t.Run(fmt.Sprintf("%dx%d/%s", size[0], size[1], theme), func(t *testing.T) {
				m, f := loaded(t)
				m.width, m.height, m.options.Theme = size[0], size[1], theme
				m.plan = &core.Plan{ID: "plan-" + strings.Repeat("exact_", 25), Revision: "revision-" + strings.Repeat("source_", 25), ClientID: "codex", Desired: m.draft(), Changes: []core.Change{{Field: "model", Before: "before-" + strings.Repeat("旧e\u0301", 60), After: "after-" + strings.Repeat("新e\u0301", 60)}}}
				original, draft := *m.plan, m.draft()
				m.goTo("Review")
				m.focusID("review:read")
				var complete, frames strings.Builder
				pages := m.reviewPages()
				for page, lines := range pages {
					if m.reviewPage != page || m.reviewRead {
						t.Fatal("review skipped a page or acknowledged early")
					}
					frame := ansi.Strip(m.View().Content)
					frames.WriteString(frame)
					first := m.reviewDocumentPages()[page][0]
					if first.role == reviewBefore && !strings.Contains(frame, "Before / model") || first.role == reviewAfter && !strings.Contains(frame, "After / model") {
						t.Fatal("wrapped review continuation loses before/after context")
					}
					for _, line := range lines {
						if !strings.Contains(frame, strings.TrimRight(ansi.Strip(line), " ")) {
							t.Fatalf("page %d hides authority row %q: %s", page+1, line, frame)
						}
						complete.WriteString(ansi.Strip(line))
					}
					assertFrame(t, m)
					press(m, "enter")
				}
				for _, group := range []string{"PLAN / EXACT VALUES", "EFFECTIVE SETTINGS", "FIELD CHANGES"} {
					if !strings.Contains(frames.String(), group) {
						t.Fatalf("review loses group context %q", group)
					}
				}
				for _, value := range []string{original.ID, original.Revision, original.Changes[0].Before, original.Changes[0].After, "Before:", "After:"} {
					if !strings.Contains(complete.String(), value) {
						t.Fatalf("review loses %q", value)
					}
				}
				if !m.reviewRead || f.applyCalls != 0 || m.draft() != draft || m.plan.ID != original.ID {
					t.Fatal("reading changed authority, draft, or backend state")
				}
			})
		}
	}
}

func TestReviewCompositionTinyPagesContainValues(t *testing.T) {
	m, _ := loaded(t)
	m.width, m.height = 40, 12
	m.plan = &core.Plan{ID: "plan", Revision: "revision", ClientID: "codex", Desired: m.draft(), Changes: []core.Change{{Field: "model", Before: "old", After: "new"}}}
	m.goTo("Review")
	for _, page := range m.reviewPages() {
		if len(page) != 1 || strings.TrimSpace(page[0]) == "" || strings.Contains(page[0], " / EXACT VALUES") || page[0] == "EFFECTIVE SETTINGS" || page[0] == "FIELD CHANGES" {
			t.Fatalf("tiny review requires acknowledging a decoration-only page: %q", page)
		}
	}
	if !strings.Contains(m.reviewPages()[0][0], "Client: codex") {
		t.Fatal("first tiny review page does not start with an actual plan value")
	}
}

func TestReviewCompositionNoopRemainsUnavailable(t *testing.T) {
	m, f := loaded(t)
	m.plan = &core.Plan{ID: "no-change-plan", Revision: "same-revision", ClientID: "codex", Desired: m.draft(), Noop: true}
	m.goTo("Review")
	readReview(t, m)
	activate(t, m, "review:apply")
	if f.applyCalls != 0 || m.plan == nil || !m.plan.Noop {
		t.Fatal("no-op styling enabled Apply or consumed its plan")
	}
	var all strings.Builder
	for _, page := range m.reviewPages() {
		for _, line := range page {
			all.WriteString(ansi.Strip(line))
		}
	}
	if !strings.Contains(all.String(), "No changes; Apply is unavailable") {
		t.Fatal("no-op reason missing from review document")
	}
}

func TestReviewPreservesSemanticEmptyValueLines(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		m, _ := loaded(t)
		m.width, m.height = size[0], size[1]
		m.plan = &core.Plan{ID: "p", Revision: "r", ClientID: "codex", Desired: m.draft(), Changes: []core.Change{{Field: "model", Before: "alpha\n\nomega", After: "alpha\nomega"}}}
		var before, after []string
		for _, page := range m.reviewDocumentPages() {
			for _, line := range page {
				if line.role == reviewBefore {
					before = append(before, line.text)
				} else if line.role == reviewAfter {
					after = append(after, line.text)
				}
			}
		}
		if strings.Join(before, "\n") != "  Before: alpha\n\nomega" || strings.Join(after, "\n") != "  After: alpha\nomega" {
			t.Fatalf("%dx%d review changes semantic newlines: before=%q after=%q", size[0], size[1], before, after)
		}
	}
}

func TestJobsCompositionIdleStartsAtUsefulAction(t *testing.T) {
	m, _ := loaded(t)
	m.goTo("Jobs")
	if m.focused().id != "back" || m.focused().reason != "" {
		t.Fatal("idle Jobs starts at an unavailable cancellation command")
	}
	frame := ansi.Strip(m.View().Content)
	for _, value := range []string{"Jobs & receipts", "No running action", "No receipts yet", "Back; keep draft"} {
		if !strings.Contains(frame, value) {
			t.Fatalf("idle Jobs lacks %q", value)
		}
	}
	press(m, "down")
	if m.focused().id != "cancel" || m.focused().reason != "No running action" {
		t.Fatal("unavailable cancellation lost its reachable explanation")
	}
	press(m, "enter")
	if m.status != "No running action" {
		t.Fatal("unavailable action did not explain itself")
	}
	m, _ = loaded(t)
	m.width, m.height = 40, 12
	m.snapshot.Receipts = []core.Receipt{{ID: "olderReceipt", ClientID: "codex", Outcome: "Succeeded", Summary: "Earlier profile"}}
	m.receipts = []core.Receipt{{ID: "latestReceipt", ClientID: "codex", Outcome: "Conflict", Summary: "Latest apply conflicted"}}
	m.goTo("Jobs")
	if !strings.Contains(ansi.Strip(m.View().Content), "Conflict") {
		t.Fatal("tiny Jobs hides the latest outcome behind unavailable actions")
	}
	var history strings.Builder
	for n := 0; n < 20; n++ {
		history.WriteString(ansi.Strip(m.View().Content))
		history.WriteByte('\n')
		press(m, "pgdown")
	}
	readable := strings.Join(strings.Fields(history.String()), " ")
	for _, value := range []string{"olderReceipt", "latestReceipt", "Succeeded", "Conflict", "Earlier profile", "Latest apply conflicted"} {
		if !strings.Contains(readable, value) {
			t.Fatalf("history loses %q", value)
		}
	}
}

func TestJobsCompositionLongReceiptsRemainReadable(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m, _ := loaded(t)
			m.width, m.height = size[0], size[1]
			m.options.NoColor, m.options.ASCII = true, true
			var values []string
			for i, outcome := range []string{"Succeeded", "Failed", "Conflict", "Interrupted"} {
				var summary []string
				for n := 0; n < 15; n++ {
					summary = append(summary, fmt.Sprintf("result%dtoken%02d", i, n))
				}
				r := core.Receipt{ID: fmt.Sprintf("receiptExact%d", i), ClientID: fmt.Sprintf("clientExact%d", i), Outcome: outcome, Summary: strings.Join(summary, " ")}
				m.receipts = append(m.receipts, r)
				values = append(values, r.ID, r.ClientID, r.Outcome)
				values = append(values, summary...)
			}
			m.goTo("Jobs")
			m.scroll = 0
			var observed strings.Builder
			for n := 0; n < 200; n++ {
				observed.WriteString(ansi.Strip(m.View().Content))
				observed.WriteByte('\n')
				assertFrame(t, m)
				_, _, before, _ := m.layout()
				press(m, "pgdown")
				_, _, after, _ := m.layout()
				if before == after {
					break
				}
			}
			for _, value := range values {
				if !strings.Contains(observed.String(), value) {
					t.Fatalf("Jobs scrolling loses %q", value)
				}
			}
			if len(m.receipts) != 4 {
				t.Fatal("rendering or paging changed receipt history")
			}
		})
	}
}
