package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestFix1BackRestoresEachVisitFrame(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "model:glm-5.3")
	draft := m.draft()
	m.focusID("advanced:model")
	m.scroll = 2
	press(m, "]") // Models visit 1 -> Review
	m.focusID("review:back")
	m.scroll = 4
	press(m, "[") // Review -> Models visit 2
	m.focusID("advanced:budget")
	m.scroll = 7
	press(m, "esc")
	wantFocus(t, m, "review:back")
	if m.screen != "Review" || m.scroll != 4 {
		t.Fatal("first Back lost Review visit")
	}
	press(m, "esc")
	wantFocus(t, m, "advanced:model")
	if m.screen != "Models" || m.scroll != 2 || m.draft() != draft {
		t.Fatal("second Back lost original Models visit/draft")
	}
	if f.planCalls+f.applyCalls+f.keyCalls+f.serviceCalls != 0 {
		t.Fatal("Back invoked a backend action")
	}
}

// Locate by exact ID and provenance in the rendered control data, without
// assuming the format of the production control ID being tested.
func advertisedControl(t *testing.T, m *model, info core.ModelInfo) string {
	t.Helper()
	for _, c := range m.controls() {
		if c.label == "Advertised model "+info.ID+" | "+info.Source {
			return c.id
		}
	}
	t.Fatalf("missing advertised fixture model %q from %q", info.ID, info.Source)
	return ""
}
func discoveryUpdate(t *testing.T, m *model, models []core.ModelInfo, away bool) {
	t.Helper()
	cmd := m.begin("discover", func(context.Context) (any, error) { return models, nil })
	if away {
		m.goTo("Jobs")
	}
	run(t, m, cmd)
	if away {
		press(m, "esc")
	}
}
func TestFix1DiscoveredFocusTracksExactIdentity(t *testing.T) {
	// Same ID from two sources, plus escaped/Unicode fields, must stay distinct.
	a := core.ModelInfo{ID: "exact-模型\"\\:A", Source: "route:\"one", Context: 500000}
	b := core.ModelInfo{ID: a.ID, Source: "route:\"two", Context: 353000}
	c := core.ModelInfo{ID: "other-B", Source: a.Source}
	for _, away := range []bool{false, true} {
		t.Run(fmt.Sprintf("away=%t", away), func(t *testing.T) {
			m, f := loaded(t)
			activate(t, m, "client:codex")
			m.models = []core.ModelInfo{a, b, c}
			id := advertisedControl(t, m, a)
			m.focusID(id)
			m.scroll = 5
			draft := m.draft()
			discoveryUpdate(t, m, []core.ModelInfo{b, c, a}, away)
			wantFocus(t, m, advertisedControl(t, m, a))
			if m.draft() != draft || m.scroll != 5 {
				t.Fatal("reorder changed draft/scroll")
			}
			press(m, "enter")
			p := m.draft().Profile
			if p.Model != a.ID || p.CapacityConfirmed || p.TiersConfirmed {
				t.Fatal("activation lost exact model ID or inferred authority")
			}
			if f.applyCalls+f.planCalls != 0 {
				t.Fatal("discovery bypassed exact review")
			}
		})
	}
}
func TestFix1DisappearingDiscoveryFocusIsSafe(t *testing.T) {
	a := core.ModelInfo{ID: "removed-模型", Source: "fixture"}
	b := core.ModelInfo{ID: "replacement", Source: "fixture"}
	for _, away := range []bool{false, true} {
		t.Run(fmt.Sprintf("away=%t", away), func(t *testing.T) {
			m, f := loaded(t)
			activate(t, m, "client:codex")
			m.models = []core.ModelInfo{a, b}
			stale := advertisedControl(t, m, a)
			m.focusID(stale)
			m.scroll = 3
			draft := m.draft()
			discoveryUpdate(t, m, []core.ModelInfo{b}, away)
			wantFocus(t, m, "discover")
			if m.draft() != draft || m.scroll != 3 {
				t.Fatal("disappearing model changed draft/scroll")
			}
			// A stale identity must never resolve by its former array position.
			m.activate(control{id: stale})
			if m.draft() != draft || !strings.Contains(m.status, "no longer") {
				t.Fatal("stale model silently activated a replacement")
			}
			run(t, m, press(m, "enter")) // Safe fallback deliberately rediscovers.
			if m.draft() != draft || f.planCalls+f.applyCalls != 0 {
				t.Fatal("fallback staged/applied a replacement")
			}
		})
	}
}
func TestFix1HelpPagingBoundariesAndResize(t *testing.T) {
	for _, keys := range [][2]string{{"tab", "shift+tab"}, {"down", "up"}, {"pgdown", "pgup"}} {
		t.Run(keys[0], func(t *testing.T) {
			m, _ := loaded(t)
			m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
			m.openModal("help")
			declaredLast := func() int {
				t.Helper()
				var page, count int
				if _, err := fmt.Sscanf(m.modalRows()[0].text, "Shortcut help | Page %d of %d", &page, &count); err != nil || count < 1 {
					t.Fatal("Help must declare its page range", err)
				}
				return count - 1
			}
			last := declaredLast()
			check := func(want int) {
				t.Helper()
				if m.modalFocus != want {
					t.Fatalf("stored Help page = %d, want %d", m.modalFocus, want)
				}
				// Movement must match the displayed range. Wrapped-row readability
				// and complete binding coverage are checked independently.
				if !strings.Contains(m.modalRows()[0].text, fmt.Sprintf("Page %d of %d", want+1, last+1)) {
					t.Fatal("stored/rendered Help page disagree")
				}
				assertFrame(t, m)
			}
			for i := 0; i < 10; i++ {
				press(m, keys[1])
			}
			check(0)
			for i := 0; i < last+10; i++ {
				press(m, keys[0])
			}
			check(last)
			press(m, keys[1])
			check(last - 1)
			press(m, keys[0])
			check(last)
			m.Update(tea.WindowSizeMsg{Width: 80, Height: 16})
			last = declaredLast()
			check(last)
			press(m, keys[1])
			check(last - 1)
			// Enlarging to one page must normalize state before shrinking again.
			m.Update(tea.WindowSizeMsg{Width: 120, Height: 80})
			last = 0
			check(0)
			m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
			last = declaredLast()
			check(0)
			press(m, keys[0])
			check(1)
		})
	}
}
func TestFix1CredentialHintsFollowFocusAndKeys(t *testing.T) {
	for focus, name := range []string{"input", "persistence", "save", "back"} {
		t.Run(name, func(t *testing.T) {
			m, f := loaded(t)
			m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
			m.goTo("Gateway")
			activate(t, m, "key")
			m.Update(tea.PasteMsg{Content: "synthetic-fixture"})
			for i := 0; i < focus; i++ {
				press(m, "tab")
			}
			if m.editor.focus != focus {
				t.Fatal("credential Tab traversal failed")
			}
			lines := strings.Split(m.View().Content, "\n")
			hints := strings.Join(lines[len(lines)-2:], "\n")
			if !strings.Contains(hints, "Tab") || !strings.Contains(hints, "Esc") {
				t.Fatal("editor lacks traversal/cancel hints")
			}
			switch focus {
			case 0:
				for _, s := range []string{"Type/paste", "Left/Right caret", "Enter focus Save"} {
					if !strings.Contains(hints, s) {
						t.Fatalf("input hint missing %q: %s", s, hints)
					}
				}
				if strings.Contains(hints, "toggle") {
					t.Fatal("input advertises Space toggle")
				}
				frame := m.View().Content
				press(m, " ")
				if string(m.editor.value) != "synthetic-fixture " || m.persistent || m.View().Content != frame {
					t.Fatal("input Space/mask behavior changed")
				}
				navKey(m, tea.KeyLeft, 0)
				if m.editor.pos != len(m.editor.value)-1 {
					t.Fatal("input caret does not move left")
				}
				navKey(m, tea.KeyRight, 0)
				press(m, "enter")
				if m.editor.focus != 2 || f.keyCalls != 0 {
					t.Fatal("input Enter must focus Save")
				}
			case 1:
				if !strings.Contains(hints, "Space/Enter toggle") || strings.Contains(hints, "Type/paste") {
					t.Fatalf("wrong persistence hints: %s", hints)
				}
				press(m, " ")
				if !m.persistent {
					t.Fatal("persistence Space did not toggle")
				}
				press(m, "enter")
				if m.persistent || string(m.editor.value) != "synthetic-fixture" {
					t.Fatal("persistence Enter changed the key or did not toggle back")
				}
			case 2, 3:
				want := "Enter save key"
				if focus == 3 {
					want = "Enter drop unsaved key"
				}
				if !strings.Contains(hints, want) || strings.Contains(hints, "toggle") || strings.Contains(hints, "Type/paste") {
					t.Fatalf("wrong button hints: %s", hints)
				}
				before := string(m.editor.value)
				press(m, " ")
				m.Update(tea.PasteMsg{Content: "ignored"})
				if string(m.editor.value) != before || m.persistent {
					t.Fatal("button Space/paste changed key/opt-in")
				}
				cmd := press(m, "enter")
				if focus == 2 {
					run(t, m, cmd)
					if f.keyCalls != 1 || f.persistent {
						t.Fatal("Save was not explicit/session-only")
					}
				} else if cmd != nil || f.keyCalls != 0 {
					t.Fatal("Back saved the key")
				}
				if m.editor != nil {
					t.Fatal("button Enter did not close key editor")
				}
			}
			if m.editor != nil {
				press(m, "esc")
			}
			wantFocus(t, m, "key")
			if focus != 2 && f.keyCalls != 0 {
				t.Fatal("editor action saved unexpectedly")
			}
		})
	}
}

func TestFix1MissingDiscoveryDoesNotTrapRegionTraversal(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	a := core.ModelInfo{ID: "removed", Source: "fixture"}
	m.models = []core.ModelInfo{a}
	m.focusID(advertisedControl(t, m, a))
	navKey(m, tea.KeyF6, 0)
	wantFocus(t, m, "discover")
	draft := m.draft()
	discoveryUpdate(t, m, nil, false)
	navKey(m, tea.KeyF6, tea.ModShift)
	// A missing region anchor uses the first control in that region, rather
	// than a vanished row's array position or a control in another region.
	wantFocus(t, m, "enabled")
	navKey(m, tea.KeyF6, tea.ModShift)
	wantFocus(t, m, "nav:Models")
	if m.draft() != draft {
		t.Fatal("region fallback staged a replacement")
	}
}
