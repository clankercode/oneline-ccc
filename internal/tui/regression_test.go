package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestDiscoveryVisibleWithoutAssertingCapacityOrTiers(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	run(t, m, activate(t, m, "discover"))
	activate(t, m, discoveredControlID(m.models[0]))
	if m.draft().Profile.Model != "exact-模型" || m.draft().Profile.Budget != 500000 || m.draft().Profile.CapacityConfirmed || m.draft().Profile.TiersConfirmed {
		t.Fatal("discovery inferred capabilities or lost exact ID")
	}
	m.focus = 4
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	v := m.View().Content
	if !strings.Contains(v, "advertised") || !strings.Contains(v, "exact-模型") {
		t.Fatal("discovery inventory is not visible")
	}
}
func TestDraftSurvivesClientDisappearanceAndStillGuardsQuit(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "enabled")
	d := m.draft()
	f.snapshot.Clients = f.snapshot.Clients[1:]
	run(t, m, press(m, "r"))
	if m.selected != "codex" || m.drafts["codex"] != d || !m.dirty() {
		t.Fatal("refresh lost selected identity or detached dirty draft")
	}
	press(m, "q")
	if m.modal != "quit" {
		t.Fatal("orphan draft bypassed dirty quit")
	}
}

type wrongPlanBackend struct{ *fakeBackend }

func (b wrongPlanBackend) Plan(ctx context.Context, id string, d core.Desired) (core.Plan, error) {
	p, _ := b.fakeBackend.Plan(ctx, id, d)
	p.Desired.Profile.Model = "different-model"
	return p, nil
}
func TestPlanMustMatchCapturedDesiredBeforeReview(t *testing.T) {
	f := fixture()
	m := New(wrongPlanBackend{f}, Options{NoColor: true}).(*model)
	run(t, m, m.Init())
	activate(t, m, "client:codex")
	activate(t, m, "enabled")
	activate(t, m, "tiers")
	d := m.draft()
	run(t, m, activate(t, m, "plan"))
	if m.plan != nil || m.screen == "Review" || m.draft() != d {
		t.Fatal("backend mismatch accepted as exact review")
	}
}
func TestLongEditorShowsCaretAndManualScrollKeepsFocus(t *testing.T) {
	m, _ := loaded(t)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	activate(t, m, "client:codex")
	activate(t, m, "advanced:model")
	m.Update(tea.PasteMsg{Content: strings.Repeat("長", 90) + "tail-here"})
	if !strings.Contains(m.View().Content, "tail-here") {
		t.Fatal("typing caret is offscreen")
	}
	press(m, "esc")
	m.goTo("Models")
	m.focusID("plan")
	press(m, "pgup")
	press(m, "pgdown")
	assertFrame(t, m)
}
func TestPasteIntoKeySaveDoesNotChangeCredential(t *testing.T) {
	m, f := loaded(t)
	m.goTo("Gateway")
	activate(t, m, "key")
	m.Update(tea.PasteMsg{Content: "initial-synthetic"})
	press(m, "enter")
	m.Update(tea.PasteMsg{Content: "unfocused-paste"})
	if string(m.editor.value) != "initial-synthetic" {
		t.Fatal("unfocused key input accepted paste")
	}
	press(m, "esc")
	if f.keyCalls != 0 {
		t.Fatal("paste caused save")
	}
}

func TestCustomBudgetDoesNotOverflowCompaction(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "advanced:budget")
	press(m, "ctrl+u")
	m.Update(tea.PasteMsg{Content: "9223372036854775807"})
	press(m, "enter")
	p := m.draft().Profile
	if p.Compact != 8301034833169298226 {
		t.Fatal("valid int64 budget overflowed compaction")
	}
}
func TestReviewResizeCannotSkipUnreadAuthorityPages(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "enabled")
	run(t, m, activate(t, m, "plan"))
	for i := 0; i < 50; i++ {
		m.plan.Changes = append(m.plan.Changes, core.Change{Field: "field", Before: "before", After: "after"})
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	activate(t, m, "review:read")
	activate(t, m, "review:read")
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.reviewPage != 0 || m.reviewRead {
		t.Fatal("resizing skipped unread beginning of the review")
	}
}

func TestAsyncDiscoveryRestoresLogicalFocus(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	run(t, m, activate(t, m, "discover"))
	if m.focused().id != "discover" {
		t.Fatalf("async results moved focus to %s", m.focused().id)
	}
}
func TestRefreshWhileHelpOpenRestoresLogicalFocus(t *testing.T) {
	m, f := loaded(t)
	m.focusID("configure")
	cmd := press(m, "r")
	press(m, "?")
	f.snapshot.Clients = append(f.snapshot.Clients, core.Client{ID: "unknown", Name: "Unknown"})
	run(t, m, cmd)
	press(m, "esc")
	if m.focused().id != "configure" {
		t.Fatalf("overlay focus moved to %s", m.focused().id)
	}
}

func TestSmallScreenCanScrollBackToAppliedEvidence(t *testing.T) {
	m, _ := loaded(t)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	activate(t, m, "client:codex")
	m.focusID("plan")
	press(m, "pgup")
	for i := 0; i < 20; i++ {
		press(m, "pgup")
	}
	if !strings.Contains(m.View().Content, "Selected: Codex") {
		t.Fatal("auto-focus prevents scrolling to top evidence")
	}
	assertFrame(t, m)
}
