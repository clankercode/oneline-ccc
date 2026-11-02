package tui

import (
	"reflect"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// Focusing one panel must not change the unrelated panel's typography or
// surface. Parse actual ANSI cells, including the odd-width trailing cell.
func TestGatewaySettingsFocusDoesNotRestylePeerPanel(t *testing.T) {
	for _, width := range []int{100, 120} {
		for _, theme := range []string{"dark", "light", "mono"} {
			for screen, id := range map[string]string{"Gateway": "advanced:endpoint", "Settings": "theme"} {
				m, _ := loaded(t)
				m.width, m.height = width, 40
				m.options.NoColor, m.options.Theme = false, theme
				m.goTo(screen)
				m.focusID(id)
				m.scroll = 0
				focused := strings.Split(m.View().Content, "\n")
				m.focus = 0
				plain := strings.Split(m.View().Content, "\n")
				found := false
				for y, line := range focused {
					if !strings.Contains(ansi.Strip(line), "> ") {
						continue
					}
					found = true
					a := uv.NewStyledString(line).Lines(ansi.GraphemeWidth)[0]
					b := uv.NewStyledString(plain[y]).Lines(ansi.GraphemeWidth)[0]
					for x := m.inset() + m.gatewaySettingsPanelWidth() + 3; x < width-m.inset(); x++ {
						if !reflect.DeepEqual(a[x].Style, b[x].Style) {
							t.Fatalf("%s/%s/%d focus restyled unrelated panel at %d,%d: %#v vs %#v", screen, theme, width, x, y, a[x].Style, b[x].Style)
						}
					}
				}
				if !found {
					t.Fatal("focused control missing")
				}
			}
		}
	}
}

func TestGatewaySettingsLongFactsKeepEveryControlVisibleAndStateUnchanged(t *testing.T) {
	for _, screen := range []string{"Gateway", "Settings"} {
		m, f := loaded(t)
		m.snapshot.Credential = "Locked / " + strings.Repeat("長e\u0301", 45)
		m.snapshot.Clients[0].Supported = false
		m.snapshot.Clients[0].Reason = strings.Repeat("Inspect 模型 / ", 30)
		m.options.Version = strings.Repeat("build-模型-e\u0301", 30)
		d := m.draft()
		d.Profile.EnvKey = "EXACT_" + strings.Repeat("LONG_REFERENCE_", 25)
		m.drafts[m.selected] = d
		m.goTo(screen)
		drafts := map[string]any{}
		for id, draft := range m.drafts {
			drafts[id] = draft
		}
		controls := append([]control(nil), m.controls()...)
		for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
			m.width, m.height = size[0], size[1]
			for _, theme := range []string{"dark", "light", "mono"} {
				m.options.Theme = theme
				for _, plain := range []bool{false, true} {
					m.options.NoColor, m.options.ASCII = plain, plain
					for i := range m.controls() {
						m.focus, m.scroll = i, -1
						assertFrame(t, m)
						m.scroll = 10000
						assertFrame(t, m)
					}
				}
			}
		}
		for id, draft := range m.drafts {
			if !reflect.DeepEqual(drafts[id], draft) {
				t.Fatal("presentation mutated the draft")
			}
		}
		// Labels depend on display options; compare stable IDs/reasons only.
		for i, c := range m.controls() {
			if c.id != controls[i].id || c.reason != controls[i].reason {
				t.Fatal("presentation changed control identity or availability")
			}
		}
		if f.keyCalls != 0 || f.planCalls != 0 || f.applyCalls != 0 || f.serviceCalls != 0 {
			t.Fatal("rendering performed an external action")
		}
	}
}
