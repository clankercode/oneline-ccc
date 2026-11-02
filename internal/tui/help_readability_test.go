package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Assert the complete page fits the real shell viewport. Merely finding a
// heading in the frame misses bindings scrolled offscreen to keep Close visible.
func TestHelpEveryPageIsReadable(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {60, 20}, {80, 24}, {100, 30}, {120, 40}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m, _ := loaded(t)
			m.width, m.height = size[0], size[1]
			m.focusID("configure")
			original := m.focused().id
			m.openModal("help")
			_, last := m.helpPagination()
			var all []string
			for page := 0; page <= last; page++ {
				if m.modalFocus != page {
					t.Fatalf("page=%d, want %d", m.modalFocus, page)
				}
				lines, _, start, available := m.layout()
				if start != 0 || len(lines) > available {
					t.Fatalf("page %d hides binding text: start=%d rows=%d viewport=%d", page+1, start, len(lines), available)
				}
				frame := ansi.Strip(m.View().Content)
				if !strings.Contains(frame, "Shortcut help") || !strings.Contains(frame, "Close help") {
					t.Fatalf("page %d lost heading or close action:\n%s", page+1, frame)
				}
				for _, line := range lines {
					if ansi.StringWidth(line.text) > m.bodyWidth() {
						t.Fatalf("page %d line exceeds viewport width", page+1)
					}
					all = append(all, ansi.Strip(line.text))
				}
				press(m, "down")
			}
			// Every catalog binding must occur in the complete readable pages,
			// including descriptions that wrap across terminal rows.
			joined := strings.Join(strings.Fields(strings.Join(all, " ")), " ")
			for _, c := range m.catalog() {
				if !strings.Contains(joined, strings.Join(strings.Fields(c.label), " ")) {
					t.Fatalf("missing readable binding %q", c.label)
				}
			}
			press(m, "down")
			press(m, "up")
			if m.modalFocus != max(0, last-1) {
				t.Fatal("last-page boundary did not reverse immediately")
			}
			press(m, "esc")
			wantFocus(t, m, original)
		})
	}
}

func TestHelpResizeRetainsReadablePage(t *testing.T) {
	m, _ := loaded(t)
	m.openModal("help")
	_, last := m.helpPagination()
	m.modalFocus = last
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	lines, _, start, available := m.layout()
	if start != 0 || len(lines) > available {
		t.Fatalf("resized Help hides text: start=%d rows=%d viewport=%d", start, len(lines), available)
	}
	_, last = m.helpPagination()
	m.modalFocus = last
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	_, last = m.helpPagination()
	if m.modalFocus > last {
		t.Fatal("resize retained an invisible page index")
	}
}
