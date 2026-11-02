package tui

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// Inspect actual View cells after the outer body column applies focus styling.
// A right-panel action may change its own text/style, never peer evidence cells.
func TestServiceSSHFocusDoesNotRestylePeerPanel(t *testing.T) {
	for _, screen := range []string{"Service", "SSH"} {
		for _, width := range []int{100, 120} {
			for _, theme := range []string{"dark", "light", "mono"} {
				for _, noColor := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%d/%s/noColor=%t", screen, width, theme, noColor), func(t *testing.T) {
						m, backend := loaded(t)
						m.width, m.height = width, 40
						m.options.Theme, m.options.NoColor, m.options.ASCII = theme, noColor, noColor
						m.snapshot.Service.Backend = "Owned user-session backend"
						m.snapshot.Service.Detail = "Observed evidence / exact 模型-e\u0301 detail"
						m.goTo(screen)
						m.scroll, m.focus = 0, 0
						plain := strings.Split(m.View().Content, "\n")
						controls := append([]control(nil), m.controls()...)
						selected, drafts := m.selected, maps.Clone(m.drafts)
						leftEnd := m.inset() + (m.contentWidth()-3)/2
						for i, c := range controls {
							if i < navCount {
								continue
							}
							m.focus = i
							view := m.View().Content
							if noColor && strings.Contains(view, "\x1b") {
								t.Fatal("NoColor emitted ANSI")
							}
							focused := strings.Split(view, "\n")
							if len(focused) != len(plain) {
								t.Fatal("focus changed frame height")
							}
							found := false
							for y := len(m.shellHeader()); y < m.height-len(m.shellFooter("")); y++ {
								a, b := uv.NewScreenBuffer(width, 1), uv.NewScreenBuffer(width, 1)
								uv.NewStyledString(focused[y]).Draw(a, a.Bounds())
								uv.NewStyledString(plain[y]).Draw(b, b.Bounds())
								if ansi.StringWidth(focused[y]) != width || ansi.StringWidth(plain[y]) != width {
									t.Fatal("focus changed frame width")
								}
								for x := m.inset(); x < leftEnd; x++ {
									if !reflect.DeepEqual(a.CellAt(x, 0), b.CellAt(x, 0)) {
										t.Fatalf("%s restyled peer evidence at (%d,%d): focused=%#v plain=%#v", c.id, x, y, a.CellAt(x, 0), b.CellAt(x, 0))
									}
								}
								if !strings.Contains(ansi.Strip(focused[y]), "> ") {
									continue
								}
								if found {
									t.Fatal("multiple focused body rows")
								}
								found = true
								if !noColor {
									first := leftEnd + 3
									style := a.CellAt(first, 0).Style
									if style.Bg == nil || style.Attrs&uv.AttrBold == 0 || style.Equal(&b.CellAt(first, 0).Style) {
										t.Fatal("action lost its own focus emphasis")
									}
									for x := first; x < width-m.inset(); x++ {
										if !style.Equal(&a.CellAt(x, 0).Style) {
											t.Fatalf("%s focus surface stops before trailing action cell %d", c.id, x)
										}
									}
								}
							}
							if !found || m.focused().id != c.id || m.scroll != 0 {
								t.Fatal("render lost real control focus or changed scroll")
							}
						}
						if !reflect.DeepEqual(controls, m.controls()) || m.selected != selected || !reflect.DeepEqual(drafts, m.drafts) {
							t.Fatal("render changed controls, selection or drafts")
						}
						if backend.serviceCalls != 0 || backend.commandAction != "" {
							t.Fatal("render invoked backend")
						}
					})
				}
			}
		}
	}
}
