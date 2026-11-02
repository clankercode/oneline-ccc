package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func TestRepairWideFocusDoesNotRestyleExactBrief(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}} {
		for _, appearance := range []string{"dark", "light", "mono", "no-color"} {
			for _, state := range []string{"empty", "preparing", "prepared"} {
				t.Run(fmt.Sprintf("%dx%d/%s/%s", size[0], size[1], appearance, state), func(t *testing.T) {
					m, _ := loaded(t)
					m.width, m.height = size[0], size[1]
					m.options.Theme, m.options.NoColor, m.options.ASCII = appearance, appearance == "no-color", appearance == "no-color"
					if m.options.NoColor {
						m.options.Theme = "dark"
					}
					m.goTo("Repair")
					if state != "empty" {
						m.repair = strings.Repeat("exact public brief evidence row\n", 24)
					}
					if state == "preparing" {
						m.busy = "repair"
					}
					m.focus, m.scroll = 0, 0
					plain := strings.Split(m.View().Content, "\n")
					selected, draft, brief := m.selected, m.draft(), m.repair
					controls := append([]control(nil), m.controls()...)
					leftWidth := (m.contentWidth() - 3) / 2
					first, last := m.inset()+leftWidth+3, m.width-m.inset()
					for i, c := range controls {
						if i < navCount {
							continue
						}
						m.focus = i
						focused := strings.Split(m.View().Content, "\n")
						found := false
						for y := len(m.shellHeader()); y < m.height-len(m.shellFooter("")); y++ {
							a := uv.NewStyledString(focused[y]).Lines(ansi.GraphemeWidth)[0]
							b := uv.NewStyledString(plain[y]).Lines(ansi.GraphemeWidth)[0]
							for x := first; x < last; x++ {
								if !reflect.DeepEqual(a[x], b[x]) {
									t.Fatalf("%s focus changed exact-brief cell (%d,%d): focused=%#v plain=%#v", c.id, x, y, a[x], b[x])
								}
							}
							if strings.Contains(ansi.Strip(focused[y]), "> "+c.label) {
								found = true
								if !m.options.NoColor && reflect.DeepEqual(a[m.inset()+leftWidth-1].Style, b[m.inset()+leftWidth-1].Style) {
									t.Fatalf("%s lost its action-panel focus fill", c.id)
								}
							}
						}
						if !found {
							t.Fatalf("%s focused action/marker missing", c.id)
						}
						assertFrame(t, m)
					}
					if m.selected != selected || m.draft() != draft || m.repair != brief || m.scroll != 0 || !reflect.DeepEqual(controls, m.controls()) {
						t.Fatal("focus rendering changed document, selection, draft, paging or controls")
					}
				})
			}
		}
	}
}

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

func TestRepairCompactInitialFrameShowsDocumentState(t *testing.T) {
	for _, prepared := range []bool{false, true} {
		m, _ := loaded(t)
		m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m.goTo("Repair")
		m.focusID("repair")
		want := "No brief prepared"
		if prepared {
			want = "EXACT-PUBLIC-BRIEF-SENTINEL"
			m.repair = want
		}
		if !strings.Contains(ansi.Strip(m.View().Content), want) {
			t.Fatalf("document state is buried in compact initial frame (prepared=%t)", prepared)
		}
		assertFrame(t, m)
	}
}
