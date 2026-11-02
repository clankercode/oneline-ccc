package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func tabTo(t *testing.T, m *model, id string) {
	t.Helper()
	for i := 0; i <= len(m.controls()); i++ {
		if m.focused().id == id {
			return
		}
		press(m, "tab")
	}
	t.Fatalf("Tab could not reach %s", id)
}
func TestKeyboardOnlyJourneyAndReviewBack(t *testing.T) {
	m, f := loaded(t)
	tabTo(t, m, "client:codex")
	press(m, " ")
	if m.screen != "Dashboard" || !m.draft().Enabled {
		t.Fatal("selection and staging coupled")
	}
	press(m, "enter")
	for _, id := range []string{"model:glm-5.3-flash", "budget:272000", "capacity", "tiers"} {
		tabTo(t, m, id)
		press(m, "enter")
	}
	tabTo(t, m, "plan")
	run(t, m, press(m, "enter"))
	if m.focused().id != "review:back" {
		t.Fatal("review initially focuses destructive action")
	}
	press(m, "enter")
	d := m.draft()
	if m.screen != "Models" || d.Profile.Model != "glm-5.3-flash" {
		t.Fatal("review Back lost draft")
	}
	tabTo(t, m, "plan")
	run(t, m, press(m, "enter"))
	tabTo(t, m, "review:read")
	for !m.reviewRead {
		press(m, "enter")
	}
	tabTo(t, m, "review:apply")
	run(t, m, press(m, "enter"))
	if f.applyCalls != 1 || m.dirty() {
		t.Fatal("keyboard Apply did not reconcile actual snapshot")
	}
}
func TestET1SingleFocusMarker(t *testing.T) {
	m, _ := loaded(t)
	for _, screen := range []string{"Models", "Jobs", "Dashboard"} {
		m.goTo(screen)
		for range m.controls() {
			assertFrame(t, m)
			press(m, "tab")
		}
	}
}
func TestUninstalledAndUnsupportedEvidenceInspectable(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:pi")
	v := m.View().Content
	if !strings.Contains(v, "Not installed") || !strings.Contains(v, "Managed: false") {
		t.Fatal("actual installation/management evidence is missing")
	}
}
func TestSpecimenFrames(t *testing.T) {
	m, _ := loaded(t)
	for _, size := range [][2]int{{80, 24}, {40, 12}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m.goTo("Dashboard")
		m.focusID("client:codex")
		assertFrame(t, m)
		t.Logf("Synthetic dashboard %dx%d:\n%s", size[0], size[1], m.View().Content)
	}
}
