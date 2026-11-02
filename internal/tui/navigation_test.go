package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func navKey(m *model, code rune, mod tea.KeyMod) tea.Cmd {
	_, cmd := m.Update(tea.KeyPressMsg{Code: code, Mod: mod})
	return cmd
}
func wantFocus(t *testing.T, m *model, id string) {
	t.Helper()
	if got := m.focused().id; got != id {
		t.Fatalf("focus = %s, want %s", got, id)
	}
}
func TestNavigationHorizontalEntryExitAndReverse(t *testing.T) {
	m, _ := loaded(t)
	navKey(m, tea.KeyRight, 0)
	wantFocus(t, m, "nav:Models")
	press(m, "down")
	wantFocus(t, m, "nav:Models")
	if m.screen != "Dashboard" {
		t.Fatal("arrow activated a tab")
	}
	navKey(m, tea.KeyLeft, 0)
	wantFocus(t, m, "nav:Dashboard")
	navKey(m, tea.KeyF6, 0)
	wantFocus(t, m, "client:codex")
	navKey(m, tea.KeyF6, tea.ModShift)
	wantFocus(t, m, "nav:Dashboard")
	// Every control remains in the complete forward/reverse Tab cycle.
	for range m.controls() {
		before := m.focused().id
		press(m, "tab")
		press(m, "shift+tab")
		wantFocus(t, m, before)
		press(m, "tab")
	}
	navKey(m, tea.KeyRight, 0)
	press(m, "enter")
	if m.screen != "Models" {
		t.Fatal("Enter did not activate tab")
	}
	wantFocus(t, m, "enabled")
	press(m, "shift+tab")
	wantFocus(t, m, "nav:More")
	press(m, "enter")
	press(m, "shift+tab")
	press(m, "enter")
	if m.screen != "Settings" {
		t.Fatal("reverse More traversal failed")
	}
	press(m, "esc")
	wantFocus(t, m, "nav:More")
}
func TestNavigationLocalArrowsAndRegions(t *testing.T) {
	m, _ := loaded(t)
	m.focusID("client:codex")
	press(m, "up")
	wantFocus(t, m, "client:codex")
	press(m, "down")
	wantFocus(t, m, "client:pi")
	press(m, "down")
	wantFocus(t, m, "client:pi")
	if m.selected != "pi" || m.dirty() {
		t.Fatal("local selection staged management")
	}
	navKey(m, tea.KeyF6, 0)
	wantFocus(t, m, "configure")
	press(m, "up")
	wantFocus(t, m, "configure")
	navKey(m, tea.KeyF6, tea.ModShift)
	wantFocus(t, m, "client:pi")
	m.goTo("Models")
	m.focusID("advanced:credential")
	navKey(m, tea.KeyF6, tea.ModShift)
	wantFocus(t, m, "nav:Models")
	navKey(m, tea.KeyF6, 0)
	wantFocus(t, m, "advanced:credential")
	press(m, "down")
	wantFocus(t, m, "advanced:credential")
	navKey(m, tea.KeyF6, 0)
	wantFocus(t, m, "discover")
	press(m, "up")
	wantFocus(t, m, "discover")
}
func TestNavigationVisibleMoreRoutes(t *testing.T) {
	for _, name := range []string{"Gateway", "Service", "Repair", "SSH", "Utilities", "Settings"} {
		t.Run(name, func(t *testing.T) {
			m, f := loaded(t)
			m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
			if !strings.Contains(m.View().Content, "More") {
				t.Fatal("secondary route hidden")
			}
			activate(t, m, "nav:More")
			for i := 0; i < 6 && !strings.Contains(m.View().Content, "> "+name); i++ {
				press(m, "down")
			}
			if !strings.Contains(m.View().Content, "> "+name) {
				t.Fatal("destination not visible in menu")
			}
			assertFrame(t, m)
			t.Logf("Synthetic native More 40x12 (%s):\n%s", name, m.View().Content)
			press(m, "enter")
			if m.screen != name || m.modal != "" {
				t.Fatalf("More route failed: %s / %s", m.screen, m.modal)
			}
			if f.applyCalls+f.keyCalls+f.serviceCalls != 0 {
				t.Fatal("navigation performed an action")
			}
			press(m, "esc")
			wantFocus(t, m, "nav:More")
		})
	}
}
func TestNavigationBackAndLayersRestoreLocation(t *testing.T) {
	m, _ := loaded(t)
	m.focusID("client:codex")
	m.scroll = 2
	press(m, "enter")
	activate(t, m, "model:glm-5.3")
	draft := m.draft()
	m.focusID("advanced:model")
	m.scroll = 3
	for _, modal := range []string{"help", "palette", "discard"} {
		m.openModal(modal)
		press(m, "down")
		press(m, "esc")
		wantFocus(t, m, "advanced:model")
		if m.scroll != 3 {
			t.Fatal("overlay lost scroll")
		}
	}
	activate(t, m, "advanced:model")
	press(m, "esc")
	wantFocus(t, m, "advanced:model")
	if m.scroll != 3 {
		t.Fatal("editor lost scroll")
	}
	press(m, "esc")
	wantFocus(t, m, "client:codex")
	if m.scroll != 2 || m.draft() != draft {
		t.Fatal("Back lost invoking location/draft")
	}
	press(m, "enter")
	wantFocus(t, m, "advanced:model")
	if m.scroll != 3 {
		t.Fatal("section return lost scroll")
	}
	// Exact review initially focuses Back, then returns to the invoking action.
	m.focusID("plan")
	m.scroll = 4
	run(t, m, press(m, "enter"))
	press(m, "enter")
	wantFocus(t, m, "plan")
	if m.scroll != 4 || m.draft() != draft {
		t.Fatal("review Back lost location/draft")
	}
}
func TestNavigationTextAndPasteGuards(t *testing.T) {
	for _, layer := range []string{"editor", "palette"} {
		t.Run(layer, func(t *testing.T) {
			m, f := loaded(t)
			if layer == "editor" {
				activate(t, m, "client:codex")
				activate(t, m, "advanced:model")
			} else {
				press(m, "ctrl+p")
			}
			screen, focus := m.screen, m.focused().id
			for _, key := range []string{"[", "]", "?", "q", "r", "/", " "} {
				press(m, key)
			}
			press(m, "ctrl+p")
			navKey(m, tea.KeyF6, 0)
			m.Update(tea.PasteMsg{Content: "[?q r/]"})
			text := m.query
			if m.editor != nil {
				text = string(m.editor.value)
			}
			if !strings.HasSuffix(text, "[]?qr/ [?q r/]") {
				t.Fatalf("text lost: %q", text)
			}
			if m.screen != screen || m.focused().id != focus || f.inspectCalls != 1 || m.dirty() {
				t.Fatal("typing triggered navigation/action")
			}
		})
	}
}
func TestNavigationRefreshResizeAndEditorStability(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	m.focusID("advanced:model")
	m.scroll = 3
	activate(t, m, "advanced:model")
	m.Update(tea.PasteMsg{Content: "-draft"})
	text := string(m.editor.value)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	if string(m.editor.value) != text {
		t.Fatal("resize lost editor draft")
	}
	// Simulate an inspection already in flight when the editor opened.
	f.snapshot.Clients = append(f.snapshot.Clients, core.Client{ID: "other", Name: "Other"})
	run(t, m, m.inspect())
	press(m, "esc")
	wantFocus(t, m, "advanced:model")
	if m.scroll != 3 || m.selected != "codex" {
		t.Fatal("refresh/resize lost location")
	}
	m.goTo("Dashboard")
	m.focusID("configure")
	m.scroll = 2
	run(t, m, press(m, "r"))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	wantFocus(t, m, "configure")
	if m.scroll != 2 || m.selected != "codex" {
		t.Fatal("refresh/resize reset location")
	}
}

func TestNavigationSectionAcceleratorAndOverlayHints(t *testing.T) {
	m, f := loaded(t)
	m.focusID("configure")
	m.scroll = 2
	press(m, "]")
	if m.screen != "Models" {
		t.Fatal("section accelerator did not switch from content")
	}
	press(m, "[")
	wantFocus(t, m, "configure")
	if m.scroll != 2 {
		t.Fatal("section accelerator lost location")
	}
	for _, modal := range []string{"more", "help", "discard"} {
		m.openModal(modal)
		view := m.View().Content
		if strings.Contains(view, "F6 region") || strings.Contains(view, "[ ] section") {
			t.Fatalf("%s advertises inactive global keys", modal)
		}
		press(m, "]")
		if m.screen != "Dashboard" {
			t.Fatal("overlay leaked section shortcut")
		}
		press(m, "esc")
	}
	if f.applyCalls+f.planCalls+f.serviceCalls != 0 || m.dirty() {
		t.Fatal("section switch mutated state")
	}
}
