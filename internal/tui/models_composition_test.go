package tui

import (
	"context"
	"errors"
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
			if size[0] == 40 && !strings.Contains(frame, "Review draft") {
				t.Fatal("tiny form lost visible Review route")
			}
			if size[0] == 80 && (!strings.Contains(frame, "Applied") || !strings.Contains(frame, "Draft")) {
				t.Fatal("stacked form hides applied/draft comparison")
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

func TestModelsSelectorEveryTabStopVisible(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {40, 12}} {
		for _, field := range []string{"model", "budget", "tier"} {
			m, _ := loaded(t)
			activate(t, m, "client:codex")
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			activate(t, m, "field:"+field)
			before := m.draft()
			for range m.controls() {
				assertFrame(t, m)
				id := m.focused().id
				press(m, "tab")
				assertFrame(t, m)
				press(m, "shift+tab")
				wantFocus(t, m, id)
				press(m, "tab")
			}
			if m.draft() != before {
				t.Fatal("selector navigation changed draft")
			}
		}
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

func TestModelsDiscoveryEmptyAndErrorStayUnverified(t *testing.T) {
	for _, err := range []error{nil, errors.New("fixture inventory unavailable")} {
		m, _ := loaded(t)
		activate(t, m, "client:codex")
		activate(t, m, "field:model")
		m.focusID("discover")
		before := m.draft()
		run(t, m, m.begin("discover", func(context.Context) (any, error) { return []core.ModelInfo{}, err }))
		wantFocus(t, m, "discover")
		frame := ansi.Strip(m.View().Content)
		if m.draft() != before || !strings.Contains(frame, "No advertised inventory") || strings.Contains(frame, "Up to Ultrafast") {
			t.Fatal("absent inventory invented support or changed the draft")
		}
		if err != nil && !strings.Contains(frame, "fixture inventory unavailable") {
			t.Fatal("inventory failure hidden")
		}
		press(m, "esc")
		wantFocus(t, m, "field:model")
	}
}

func TestModelsCustomBudgetKeepsValidationAndPickerReturn(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "field:budget")
	activate(t, m, "advanced:budget")
	press(m, "ctrl+u")
	m.Update(tea.PasteMsg{Content: "-1"})
	before := m.draft()
	press(m, "enter")
	if m.editor == nil || m.draft() != before {
		t.Fatal("invalid budget escaped validation")
	}
	press(m, "ctrl+u")
	m.Update(tea.PasteMsg{Content: "123456"})
	press(m, "enter")
	wantFocus(t, m, "advanced:budget")
	if m.draft().Profile.Budget != 123456 || m.draft().Profile.Compact != 111110 || m.draft().Profile.CapacityConfirmed || m.draft().Profile.TiersConfirmed {
		t.Fatal("custom budget lost exact value, compaction, or invalidation")
	}
	press(m, "esc")
	wantFocus(t, m, "field:budget")
	activate(t, m, "field:budget")
	if !strings.Contains(m.View().Content, "Current budget: 123456 tokens") {
		t.Fatal("picker hides custom budget")
	}
	press(m, "esc")
	if m.draft().Profile.Budget != 123456 {
		t.Fatal("picker cancellation lost custom budget")
	}
}

func TestModelsTinyHeaderRetainsUnicodeClientIdentity(t *testing.T) {
	m, _ := loaded(t)
	m.snapshot.Clients[0].Name = "Codex 模型 — a long fixture name"
	activate(t, m, "client:codex")
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	if !strings.Contains(ansi.Strip(m.View().Content), "Codex 模型") {
		t.Fatal("shell prefix displaces selected-client identity")
	}
	assertFrame(t, m)
}
