package tui

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func assertFrame(t *testing.T, m *model) {
	t.Helper()
	v := m.View().Content
	lines := strings.Split(v, "\n")
	if len(lines) > m.height {
		t.Fatalf("height %d exceeds %d", len(lines), m.height)
	}
	markers := 0
	for _, line := range lines {
		if ansi.StringWidth(line) > m.width {
			t.Fatalf("width %d exceeds %d: %q", ansi.StringWidth(line), m.width, line)
		}
		markers += strings.Count(ansi.Strip(line), "> ")
	}
	if !utf8.ValidString(v) {
		t.Fatal("broken Unicode")
	}
	if markers != 1 {
		t.Fatalf("wanted one focus marker; got %d at %dx%d screen %s focus %d: %q", markers, m.width, m.height, m.screen, m.focus, v)
	}
	if m.options.NoColor && strings.Contains(v, "\x1b") {
		t.Fatal("ANSI in NoColor")
	}
}
func TestEveryControlFitsRequestedDimensions(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		for _, theme := range []string{"dark", "light", "mono"} {
			t.Run(fmt.Sprintf("%dx%d/%s", size[0], size[1], theme), func(t *testing.T) {
				m, _ := loaded(t)
				m.options.Theme = theme
				m.options.NoColor = false
				m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				for _, screen := range []string{"Dashboard", "Models", "Review", "Jobs", "Gateway", "Service", "Repair", "SSH", "Utilities", "Settings"} {
					m.goTo(screen)
					for i := range m.controls() {
						m.focus = i
						m.scroll = 0
						assertFrame(t, m)
					}
				}
			})
		}
	}
}
func TestUnicodeExactIDsAndReviewPages(t *testing.T) {
	m, _ := loaded(t)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	p := m.draft().Profile
	p.Model = "exact-模型-e\u0301-" + strings.Repeat("長", 35)
	m.drafts["codex"] = core.Desired{Enabled: true, Profile: p}
	m.plan = &core.Plan{ID: "opaque-id", Revision: "rev-1", ClientID: "codex", Desired: m.draft(), Changes: []core.Change{{Field: "model", Before: "old", After: p.Model}}}
	m.goTo("Review")
	m.focusID("review:back")
	if len(m.reviewPages()) < 3 {
		t.Fatal("long review was not paged")
	}
	for range m.reviewPages() {
		assertFrame(t, m)
		activate(t, m, "review:read")
	}
	if !m.reviewRead {
		t.Fatal("last page never acknowledged")
	}
	rendered := strings.Join(func() []string {
		var all []string
		for _, page := range m.reviewPages() {
			all = append(all, strings.Join(page, ""))
		}
		return all
	}(), "")
	if !strings.Contains(rendered, p.Model) {
		t.Fatalf("exact Unicode ID lost: %q", rendered)
	}
}
func TestPaletteHelpAndEditorFitSmallTerminal(t *testing.T) {
	m, _ := loaded(t)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	for _, modal := range []string{"help", "palette", "quit", "service:install", "ssh"} {
		m.openModal(modal)
		for i := 0; i < 5; i++ {
			assertFrame(t, m)
			press(m, "down")
		}
		press(m, "esc")
	}
	activate(t, m, "client:codex")
	activate(t, m, "advanced:model")
	m.Update(tea.PasteMsg{Content: strings.Repeat("長", 100)})
	assertFrame(t, m)
}

func TestExplicitThemePairsAndNoColor(t *testing.T) {
	m, _ := loaded(t)
	for _, theme := range []string{"dark", "light"} {
		m.options.Theme = theme
		m.options.NoColor = false
		v := m.View()
		if v.BackgroundColor == nil || v.ForegroundColor == nil {
			t.Fatal("explicit theme lacks paired terminal colors")
		}
		m.options.NoColor = true
		v = m.View()
		if v.BackgroundColor != nil || v.ForegroundColor != nil || strings.Contains(v.Content, "\x1b") {
			t.Fatal("NoColor sets terminal colors")
		}
	}
}

func TestNarrowSummaryWrapsWords(t *testing.T) {
	m, _ := loaded(t)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	m.focusID("client:codex")
	v := m.View().Content
	if strings.Contains(v, "\ntandard") || !strings.Contains(v, "Standard") {
		t.Fatal("narrow summary split the tier label mid-word")
	}
}
