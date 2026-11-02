package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestRepairBriefPagingPreservesPublicAuthorityAndDraft(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			m, _ := loaded(t)
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			m.goTo("Repair")
			exact := "exact-" + strings.Repeat("模型-e\u0301-", 28) + "-FINAL"
			m.repair = "Manual diagnostic snapshot\n\nExact model: " + exact + "\nEndpoint: " + m.snapshot.Clients[0].Profile.BaseURL + "\n  Keep indent and public punctuation: {value: 353000}\nEND-OF-BRIEF"
			selected, draft, brief := m.selected, m.draft(), m.repair
			m.focusID("repair")
			// Observe actual production frames reached through manual paging, including
			// the last row behind the pinned focus anchor on the smallest terminal.
			var frames []string
			for range 300 {
				frames = append(frames, ansi.Strip(m.View().Content))
				_, _, before, _ := m.layout()
				press(m, "pgdown")
				_, _, after, _ := m.layout()
				if after == before {
					break
				}
			}
			compact := func(s string) string {
				return strings.Map(func(r rune) rune {
					if unicode.IsSpace(r) {
						return -1
					}
					return r
				}, s)
			}
			seen := compact(strings.Join(frames, ""))
			// Focus may be pinned between wrapped value rows. Every wrapped document
			// segment must occur in a real viewport, in addition to the final sentinel.
			briefWidth := m.contentWidth()
			if m.contentWidth() >= 96 && m.height >= 30 {
				briefWidth -= (briefWidth-3)/2 + 3
			}
			for _, line := range strings.Split(ansi.Wrap(m.public(brief), briefWidth, ""), "\n") {
				if !strings.Contains(seen, compact(line)) {
					t.Fatalf("public brief row unreachable: %q", line)
				}
			}
			if strings.Contains(strings.Join(frames, ""), m.snapshot.Clients[0].Profile.BaseURL) {
				t.Fatal("private endpoint exposed")
			}
			if m.selected != selected || m.draft() != draft || m.repair != brief {
				t.Fatal("render/paging mutated selected target, draft or exact brief")
			}
		})
	}
}

func TestRepairUtilitiesRetainControlReasonsAndReturnState(t *testing.T) {
	for _, screen := range []string{"Repair", "Utilities"} {
		t.Run(screen, func(t *testing.T) {
			m, _ := loaded(t)
			m.goTo("Models")
			d := m.draft()
			d.Profile.Model = "retained-custom-model"
			m.drafts[m.selected] = d
			selected, baseline := m.selected, m.baselines[m.selected]
			m.goTo(screen)
			controls := append([]control(nil), m.controls()...)
			for i, c := range controls {
				if i < navCount {
					continue
				}
				m.focus = i
				rows := m.screenRows()
				focused := 0
				var document []string
				for _, r := range rows {
					document = append(document, ansi.Strip(r.text))
					if r.focus {
						focused++
						if !strings.Contains(ansi.Strip(r.text), c.label) {
							t.Fatalf("focused control %s replaced", c.id)
						}
					}
				}
				if focused != 1 {
					t.Fatalf("control %s has %d focus rows", c.id, focused)
				}
				text := strings.Join(document, "\n")
				if c.reason != "" && !strings.Contains(text, c.reason) {
					t.Fatalf("reason for %s lost", c.id)
				}
				if !reflect.DeepEqual(controls, m.controls()) {
					t.Fatal("composition changed control order/authority")
				}
			}
			activate(t, m, "back")
			if m.screen != "Models" || m.selected != selected || m.draft() != d || m.baselines[selected] != baseline {
				t.Fatal("Back lost screen, selection or draft")
			}
		})
	}
}

func TestUtilitiesCompactInitialFrameKeepsReturnReachable(t *testing.T) {
	for _, size := range [][2]int{{60, 20}, {40, 12}} {
		m, _ := loaded(t)
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m.goTo("Utilities")
		m.focusID("unavailable")
		if !strings.Contains(ansi.Strip(m.View().Content), "  Back") {
			t.Fatalf("Back is hidden in initial %dx%d frame", size[0], size[1])
		}
		assertFrame(t, m)
	}
}
