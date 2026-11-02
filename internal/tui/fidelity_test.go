package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestFailedApplyKeepsRecoveryReceipt(t *testing.T) {
	for _, outcome := range []string{"Unknown", "Conflict", "RolledBack"} {
		t.Run(outcome, func(t *testing.T) {
			m, _ := loaded(t)
			activate(t, m, "client:codex")
			activate(t, m, "enabled")
			draft := m.draft()
			m.busy = "apply"
			m.plan = &core.Plan{ID: "reviewed", Desired: draft}
			receipt := core.Receipt{ID: "operation-to-recover", ClientID: "codex", Outcome: outcome, Summary: "recovery evidence retained"}
			m.complete(result{token: m.token, kind: "apply", value: receipt, err: errors.New("fixture refusal")})
			if len(m.receipts) != 1 || m.receipts[0] != receipt || m.draft() != draft || m.plan != nil {
				t.Fatal("failed apply lost recovery evidence or draft")
			}
			m.goTo("Jobs")
			if !strings.Contains(m.View().Content, receipt.ID) || !strings.Contains(m.View().Content, outcome) {
				t.Fatal("jobs hides failed receipt")
			}
		})
	}
}
func TestWideDashboardHasAdjacentListAndDetail(t *testing.T) {
	m, _ := loaded(t)
	for _, width := range []int{100, 120} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 40})
		m.goTo("Dashboard")
		m.focusID("client:codex")
		frame := ansi.Strip(m.View().Content)
		found := false
		for _, line := range strings.Split(frame, "\n") {
			if strings.Contains(line, "CLIENTS") && strings.Contains(line, "SELECTED CLIENT") {
				found = true
			}
		}
		if !found {
			t.Fatalf("%d columns has no adjacent list/detail headings", width)
		}
		if !strings.Contains(frame, "gpt-6.1-sol / Standard") {
			t.Fatal("model and selected default tier are separated")
		}
		assertFrame(t, m)
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assertFrame(t, m)
}
func TestModelSummaryKeepsDefaultSeparateFromAvailability(t *testing.T) {
	m, _ := loaded(t)
	m.goTo("Models")
	frame := ansi.Strip(m.View().Content)
	if !strings.Contains(frame, "gpt-6.1-sol / Standard") || !strings.Contains(frame, "Available tiers:") {
		t.Fatal("missing model/default or separate availability")
	}
}

func TestSemanticStatusColorsKeepWordsAndDowngrade(t *testing.T) {
	m, _ := loaded(t)
	m.options.NoColor = false
	for _, theme := range []string{"dark", "light"} {
		m.options.Theme = theme
		for _, status := range []string{"CONFIGURED", "NEEDS SETUP", "FAILED"} {
			rendered := m.statusText(status)
			if rendered == status || ansi.Strip(rendered) != status {
				t.Fatal("semantic styling lost label or color")
			}
		}
	}
	m.options.NoColor = true
	if m.statusText("FAILED") != "FAILED" {
		t.Fatal("NO_COLOR emitted styling")
	}
}
