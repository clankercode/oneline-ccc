package tui

import (
	"context"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func (m *model) complete(r result) tea.Cmd {
	if r.token != m.token {
		return nil
	}
	focusedID := m.focused().id
	// Command construction and terminal execution share a cancellation lifetime.
	if r.err == nil && (r.kind == "launch" || r.kind == "ssh") {
		command, ok := r.value.(*exec.Cmd)
		if !ok || command == nil {
			r.err = errUnavailable
		} else {
			token := m.token
			return tea.ExecProcess(command, func(err error) tea.Msg { return result{token: token, kind: "child", err: err} })
		}
	}
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.busy = ""
	if r.err != nil {
		if r.kind == "inspect" {
			m.reconcileClient = ""
		}
		m.status = "Action refused or failed: " + m.public(r.err.Error())
		if r.kind == "apply" {
			if receipt, ok := r.value.(core.Receipt); ok && receipt.ID != "" {
				m.receipts = append(m.receipts, receipt)
				m.status = "Receipt: " + m.public(receipt.Outcome) + " - " + m.public(receipt.Summary)
			}
		}
		if r.kind == "key" {
			m.status = "Credential operation failed; value hidden"
		}
		if r.kind == "apply" || r.kind == "plan" {
			m.plan = nil
			m.goTo("Models")
		}
		return nil
	}
	switch r.kind {
	case "inspect":
		m.snapshot = r.value.(core.Snapshot)
		for _, c := range m.snapshot.Clients {
			if _, ok := m.drafts[c.ID]; !ok || m.reconcileClient == c.ID {
				m.drafts[c.ID] = desiredClient(c)
				if m.reconcileClient == c.ID {
					m.reconcileClient = ""
				}
			}
			m.baselines[c.ID] = desiredClient(c)
		}
		m.reconcileClient = ""
		if m.selected == "" && len(m.snapshot.Clients) > 0 {
			m.selected = m.snapshot.Clients[0].ID
		}
		m.plan = nil
		m.status = "Inspected local settings; drafts retained"
		m.focus = max(0, min(m.focus, len(m.controls())-1))
		m.focusID(focusedID)
	case "plan":
		p := r.value.(core.Plan)
		if p.ID == "" || p.Revision == "" || p.ClientID != m.requestedID || p.Desired != m.requested {
			m.plan = nil
			m.status = "Backend plan did not match the requested client and draft; no changes applied"
			return nil
		}
		m.plan = &p
		m.previous = m.screen
		m.goTo("Review")
		m.reviewPage = 0
		m.reviewRead = false
		m.focusID("review:back")
		m.status = "Review the exact backend changes before Apply"
	case "apply":
		receipt := r.value.(core.Receipt)
		m.receipts = append(m.receipts, receipt)
		// Only a successful backend result advances the applied baseline. Inspect
		// then reconciles actual state; failures preserve the previous snapshot.
		if m.plan != nil {
			m.drafts[m.plan.ClientID] = m.plan.Desired
			if receipt.Outcome == "Succeeded" {
				m.reconcileClient = m.plan.ClientID
			}
		}
		m.plan = nil
		m.goTo("Dashboard")
		m.status = "Receipt: " + m.public(receipt.Outcome) + " - " + m.public(receipt.Summary)
		return m.inspect()
	case "discover":
		m.models = r.value.([]core.ModelInfo)
		m.focusID(focusedID)
		m.status = "Inventory advertised; capacity and tiers remain unverified"
	case "service":
		m.snapshot.Service = r.value.(core.ServiceInfo)
		m.status = "Monitor action returned observed status"
	case "repair":
		m.repair = r.value.(string)
		m.status = "Redacted manual brief ready; select export to write it to terminal scrollback"
	case "key":
		m.status = "Credential saved; value hidden"
		return m.inspect()
	case "child":
		m.status = "Client returned; authentication and inference were not verified"
	}
	return nil
}
func (m *model) makePlan() tea.Cmd {
	if why := m.editable(); why != "" {
		m.status = why
		return nil
	}
	d := m.draft()
	if d.Enabled {
		if err := core.ValidateProfile(d.Profile); err != nil {
			m.status = m.public(err.Error())
			return nil
		}
	}
	id := m.selected
	if m.busy != "" {
		m.status = "Wait for the current action"
		return nil
	}
	m.requested = d
	m.requestedID = id
	return m.begin("plan", func(ctx context.Context) (any, error) { return m.backend.Plan(ctx, id, d) })
}
func (m *model) apply() tea.Cmd {
	if m.plan == nil || m.plan.Noop || !m.reviewRead {
		m.status = "Read the complete review before Apply"
		return nil
	}
	p := *m.plan
	if m.drafts[p.ClientID] != p.Desired {
		m.plan = nil
		m.status = "Draft changed; request a new plan"
		return nil
	}
	return m.begin("apply", func(ctx context.Context) (any, error) { return m.backend.Apply(ctx, p.ID, p.Revision) })
}
func (m *model) command(kind string) tea.Cmd {
	id := m.selected
	if kind == "ssh" {
		id = ""
	}
	return m.begin(kind, func(ctx context.Context) (any, error) { return m.backend.Command(ctx, kind, id) })
}
func (m *model) service(action string) tea.Cmd {
	return m.begin("service", func(ctx context.Context) (any, error) { return m.backend.Service(ctx, action) })
}
