package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func helpText(m *model) string {
	pages, _ := m.helpPages()
	var lines []string
	for _, page := range pages {
		lines = append(lines, page...)
	}
	return strings.Join(strings.Fields(ansi.Strip(strings.Join(lines, " "))), " ")
}

func TestHelpGroupsAndFullProseFitEveryPage(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {60, 20}, {80, 24}, {100, 30}, {120, 40}} {
		for _, theme := range []string{"dark", "light", "mono", "none"} {
			t.Run(fmt.Sprintf("%dx%d/%s", size[0], size[1], theme), func(t *testing.T) {
				m, _ := loaded(t)
				m.width, m.height = size[0], size[1]
				m.options.Theme, m.options.NoColor = theme, theme == "none"
				m.openModal("help")
				pages, capacity := m.helpPages()
				for page, lines := range pages {
					if len(lines) == 0 || len(lines) > capacity {
						t.Fatalf("page %d rows=%d capacity=%d", page+1, len(lines), capacity)
					}
					for i, line := range lines {
						text := strings.TrimSpace(ansi.Strip(line))
						if strings.HasPrefix(text, "-- ") && (i+1 == len(lines) || strings.TrimSpace(ansi.Strip(lines[i+1])) == "") {
							t.Fatalf("orphan group on page %d: %s", page+1, text)
						}
						if theme == "none" && line != ansi.Strip(line) {
							t.Fatal("no-color Help emits styles")
						}
					}
					m.modalFocus = page
					rows, _, start, available := m.layout()
					if start != 0 || len(rows) > available {
						t.Fatalf("page %d clipped: start=%d rows=%d available=%d", page+1, start, len(rows), available)
					}
				}
				text := helpText(m)
				for _, want := range []string{
					"Type/paste stays in the editor; letters/Space are text. Ctrl+C still requests safe exit.",
					"Left/Right moves the caret. Enter saves the field; Esc keeps the previous draft.",
					"Key input: Enter focuses Save. Tab/Shift+Tab visits options; only input accepts text.",
					"Persistence: Space/Enter toggles. Save stores the key; Back drops unsaved text.",
					"Stops at first/last; reverse works. Resize reflows. Close restores focus and keeps drafts.",
				} {
					if !strings.Contains(text, want) {
						t.Errorf("prose lost across page boundary: %q", want)
					}
				}
			})
		}
	}
}

func TestHelpUsefulFirstNarrowPage(t *testing.T) {
	m, _ := loaded(t)
	m.width, m.height = 40, 12
	m.openModal("help")
	frame := ansi.Strip(m.View().Content)
	for _, want := range []string{"MOVE & ACT", "Tab / Shift+Tab", "Move focus", "Enter", "Activate focused control", "Close help"} {
		if !strings.Contains(frame, want) {
			t.Errorf("first narrow page missing %q:\n%s", want, frame)
		}
	}
	pages, _ := m.helpPages()
	for _, page := range pages {
		for _, line := range page {
			for _, destination := range sections {
				if strings.TrimSpace(ansi.Strip(line)) == destination {
					t.Errorf("unkeyed destination %q belongs in Commands", destination)
				}
			}
		}
	}
}

func TestHelpWideKeepsGuidanceTogether(t *testing.T) {
	m, _ := loaded(t)
	m.width, m.height = 120, 40
	m.focusID("help")
	press(m, "?")
	frame := ansi.Strip(m.View().Content)
	for _, want := range []string{"Page 1 of 1", "TEXT ENTRY", "HELP PAGES", "first/last", "Close help"} {
		if !strings.Contains(frame, want) {
			t.Errorf("wide Help unnecessarily separates guidance %q:\n%s", want, frame)
		}
	}
}

func TestHelpGuidanceAndCatalogKeys(t *testing.T) {
	m, _ := loaded(t)
	m.openModal("help")
	text := helpText(m)
	for _, c := range m.catalog() {
		if c.key != "" && !strings.Contains(text, c.key) {
			t.Errorf("missing actual catalog key %q", c.key)
		}
	}
	for _, want := range []string{"SECTIONS & COMMANDS", "More", "TEXT ENTRY", "Type/paste", "Ctrl+C", "safe exit", "HELP PAGES", "first/last", "CURRENT CONTROL"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing guidance %q: %s", want, text)
		}
	}
	// Search retains the unkeyed destinations omitted from shortcut Help.
	press(m, "esc")
	press(m, "ctrl+p")
	for _, destination := range sections {
		found := false
		for _, c := range m.paletteCommands() {
			found = found || c.id == "nav:"+destination
		}
		if !found {
			t.Errorf("palette lost %s", destination)
		}
	}
}

func TestHelpCurrentControlAndRestore(t *testing.T) {
	for _, tc := range []struct{ id, want, absent string }{
		{"configure", "Configure selected client", "Space toggles this control"},
		{"client:codex", "Enter opens client settings", "Space does nothing here"},
		{"nav:More", "Enter opens More destinations", "Space toggles this control"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			m, f := loaded(t)
			m.focusID(tc.id)
			draft := m.draft()
			press(m, "?")
			text := helpText(m)
			if !strings.Contains(text, tc.want) || strings.Contains(text, tc.absent) {
				t.Fatalf("inaccurate context: %s", text)
			}
			for _, key := range []string{"down", "up", "pgdown", "pgup", "tab", "shift+tab"} {
				press(m, key)
			}
			press(m, "enter")
			wantFocus(t, m, tc.id)
			if m.draft() != draft || f.applyCalls+f.planCalls+f.keyCalls+f.serviceCalls != 0 {
				t.Fatal("reading Help mutated draft/backend")
			}
		})
	}
}

func TestHelpContextRespectsAvailabilityAndRegion(t *testing.T) {
	for _, tc := range []struct{ screen, id, busy, want, absent string }{
		{"Dashboard", "client:pi", "", "Space unavailable: Read only", "Space stages management"},
		{"Settings", "nav:Utilities", "", "Enter opens Utilities", "Left/Right moves tab focus"},
		{"Settings", "ascii", "", "Space toggles this control", "Unavailable:"},
		{"Jobs", "cancel", "", "Unavailable: No running action", "Enter activates this control"},
		{"Dashboard", "configure", "inspect", "Wait for the backend's final outcome", "Enter activates this control"},
	} {
		t.Run(tc.id+tc.busy, func(t *testing.T) {
			m, _ := loaded(t)
			m.goTo(tc.screen)
			m.focusID(tc.id)
			m.busy = tc.busy
			press(m, "?")
			text := helpText(m)
			if !strings.Contains(text, tc.want) || strings.Contains(text, tc.absent) {
				t.Fatalf("wrong current-control guidance: %s", text)
			}
		})
	}
}
