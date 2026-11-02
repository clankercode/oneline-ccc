package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestModelsMainPathUsesFields(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	before := m.draft()
	for _, id := range []string{"enabled", "field:model", "field:budget", "field:tier", "models:advanced", "plan"} {
		wantFocus(t, m, id)
		press(m, "tab")
	}
	if m.draft() != before || f.planCalls+f.applyCalls != 0 {
		t.Fatal("field navigation changed draft or called backend")
	}
}

func TestModelsSelectorCancelAndCommit(t *testing.T) {
	for _, field := range []string{"model", "budget", "tier"} {
		t.Run(field, func(t *testing.T) {
			m, _ := loaded(t)
			activate(t, m, "client:codex")
			before := m.draft()
			activate(t, m, "field:"+field)
			press(m, "down")
			if m.draft() != before {
				t.Fatal("arrow selected a value")
			}
			m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
			assertFrame(t, m)
			press(m, "esc")
			wantFocus(t, m, "field:"+field)
			if m.screen != "Models" || m.draft() != before {
				t.Fatal("selector Escape lost screen or draft")
			}
			activate(t, m, "field:"+field)
			id := map[string]string{"model": "model:glm-5.3", "budget": "budget:353000", "tier": "tier:Fast"}[field]
			activate(t, m, id)
			wantFocus(t, m, "field:"+field)
			if m.draft() == before || m.draft().Profile.CapacityConfirmed || m.draft().Profile.TiersConfirmed {
				t.Fatal("choice failed to change draft and invalidate route attestations")
			}
		})
	}
}

func TestModelsCompositionAndCompactContext(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			m, _ := loaded(t)
			activate(t, m, "client:codex")
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			frame := ansi.Strip(m.View().Content)
			if !strings.Contains(frame, "Codex") || strings.Contains(frame, "Advanced: edit") {
				t.Fatal("missing selected client or advanced fields exposed by default")
			}
			if size[0] >= 100 {
				if !strings.Contains(frame, "DRAFT / NOT YET APPLIED") || !strings.Contains(frame, "gpt-6.1-sol / Standard") || !strings.Contains(frame, "Available tiers:") {
					t.Fatalf("missing model/draft hierarchy: %s", frame)
				}
			}
			for _, id := range []string{"enabled", "field:model", "field:budget", "field:tier", "models:advanced", "plan", "back"} {
				m.focusID(id)
				m.scroll = -1
				assertFrame(t, m)
				if !strings.Contains(ansi.Strip(m.View().Content), "Codex") {
					t.Fatalf("selected client lost on %s", id)
				}
			}
		})
	}
}

func TestModelsAdvancedEditorAndDiscoveryIdentity(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "models:advanced")
	activate(t, m, "advanced:model")
	press(m, "ctrl+u")
	exact := "exact-模型:?q/r[]"
	m.Update(tea.PasteMsg{Content: exact})
	press(m, "enter")
	wantFocus(t, m, "advanced:model")
	if m.draft().Profile.Model != exact {
		t.Fatal("custom ID was rewritten")
	}
	activate(t, m, "models:advanced")
	wantFocus(t, m, "models:advanced")
	m.models = []core.ModelInfo{{ID: exact, Source: "source one"}, {ID: exact, Source: "source two"}}
	activate(t, m, "field:model")
	for _, info := range m.models {
		m.focusID(discoveredControlID(info))
		frame := ansi.Strip(m.View().Content)
		if !strings.Contains(frame, info.Source) {
			t.Fatal("source provenance missing")
		}
	}
	press(m, "esc")
	wantFocus(t, m, "field:model")
}

// The former flat-radio journey must now enter each deliberate disclosure.
// All actions here use real keyboard input, including exact Review/Apply.
func TestModelsKeyboardJourneyWithDisclosures(t *testing.T) {
	m, f := loaded(t)
	tabTo(t, m, "client:codex")
	press(m, " ")
	press(m, "enter")
	for _, choice := range [][2]string{{"field:model", "model:glm-5.3-flash"}, {"field:budget", "budget:272000"}} {
		tabTo(t, m, choice[0])
		press(m, "enter")
		tabTo(t, m, choice[1])
		press(m, "enter")
		wantFocus(t, m, choice[0])
	}
	tabTo(t, m, "models:advanced")
	press(m, "enter")
	for _, id := range []string{"capacity", "tiers"} {
		tabTo(t, m, id)
		press(m, " ")
	}
	tabTo(t, m, "plan")
	run(t, m, press(m, "enter"))
	wantFocus(t, m, "review:back")
	press(m, "enter")
	if m.screen != "Models" || m.draft().Profile.Model != "glm-5.3-flash" || m.draft().Profile.Budget != 272000 {
		t.Fatal("Review Back lost choices")
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
		t.Fatal("disclosed keyboard journey did not reconcile applied state")
	}
}

func TestModelsDisclosureStateFollowsClientAndBack(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "models:advanced")
	m.focusID("advanced:budget")
	before := m.draft()
	m.goTo("Jobs")
	press(m, "esc")
	wantFocus(t, m, "advanced:budget")
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	assertFrame(t, m)
	m.goTo("Dashboard")
	activate(t, m, "client:pi")
	if m.modelsForm().advanced || m.modelsForm().chooser != "" {
		t.Fatal("disclosure leaked to another client")
	}
	wantFocus(t, m, "enabled")
	if strings.Contains(m.View().Content, "Advanced: edit") {
		t.Fatal("prior client focus exposed Advanced for a new client")
	}
	m.goTo("Dashboard")
	activate(t, m, "client:codex")
	if !m.modelsForm().advanced || m.draft() != before {
		t.Fatal("client return lost Advanced or draft")
	}
	press(m, "esc")
	wantFocus(t, m, "models:advanced")
	if m.draft() != before {
		t.Fatal("collapsing Advanced changed the draft")
	}
}

func TestModelsFieldValuesDoNotClip(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	d := m.draft()
	d.Profile.Model = "exact-" + strings.Repeat("模型", 18) + "-end"
	m.drafts[m.selected] = d
	for _, size := range [][2]int{{40, 12}, {100, 30}, {120, 40}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m.focusID("field:model")
		lines, _, _, _ := m.layout()
		var document strings.Builder
		for _, line := range lines {
			document.WriteString(ansi.Strip(line.text))
		}
		if !strings.Contains(document.String(), "-end") || !strings.Contains(document.String(), "[Choose]") {
			t.Fatal("field value or action clipped before wrapping")
		}
		assertFrame(t, m)
	}
}
