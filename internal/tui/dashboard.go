package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func (m *model) wideDashboard() bool {
	return m.width >= 100 && m.screen == "Dashboard" && m.modal == "" && m.editor == nil
}

func (m *model) dashboardRows() []row {
	width := m.contentWidth()
	leftWidth, rightWidth := width, width
	if m.wideDashboard() {
		leftWidth = (width - 3) / 2
		rightWidth = width - leftWidth - 3
	}
	managed, pending, matches := 0, 0, 0
	for _, c := range m.snapshot.Clients {
		if c.Managed {
			managed++
		}
		if m.drafts[c.ID] != m.baselines[c.ID] {
			pending++
		}
		if m.matches(c.ID, c.Name) {
			matches++
		}
	}
	rs := []row{{text: m.heading("Your clients")}, {text: m.note(fmt.Sprintf("%d managed / %d pending drafts   / filter   r refresh", managed, pending))}, {}}
	left := []row{{text: m.heading("CLIENTS / MANAGEMENT")}}
	if m.filter != "" {
		left = append(left, row{text: m.note("Filter: " + m.public(m.filter))})
	}
	right := []row{{text: m.heading("SELECTED CLIENT / APPLIED VALUES")}, {}}
	c, ok := m.client()
	if ok {
		right = append(right, row{text: m.heading(m.public(c.Name)) + " / " + m.statusText(c.Status)})
		right = append(right, m.detailRows("Management", map[bool]string{true: "Enabled", false: "Not managed"}[c.Managed], rightWidth)...)
		right = append(right, m.detailRows("Gateway", "Gateway A / local settings", rightWidth)...)
		right = append(right, m.detailRows("Model / default", m.public(c.Profile.Model+" / "+tierLabel(c.Profile.Tier)), rightWidth)...)
		right = append(right, m.detailRows("Budget", fmt.Sprintf("%d tokens / compact %d", c.Profile.Budget, c.Profile.Compact), rightWidth)...)
		right = append(right, m.detailRows("Available tiers:", tierAvailability(m.appliedTiers()), rightWidth)...)
		right = append(right, m.detailRows("Auth / API", "Unverified / not tested", rightWidth)...)
		if c.Reason != "" {
			right = append(right, row{text: m.note(m.public(c.Reason))})
		}
		if m.drafts[c.ID] != m.baselines[c.ID] {
			right = append(right, row{text: m.heading("Draft pending / applied values shown above")})
		}
	} else {
		right = append(right, row{text: "No client selected. Inspect or refresh local settings."})
	}
	right = append(right, row{}, row{text: m.heading("NEXT ACTIONS")}, row{text: m.note("Configure, then review. Launch uses applied settings.")})
	aux := false
	for i, control := range m.controls() {
		if i < navCount {
			continue
		}
		if id, clientRow := strings.CutPrefix(control.id, "client:"); clientRow {
			for _, client := range m.snapshot.Clients {
				if client.ID == id {
					left = append(left, m.clientRows(client, i, leftWidth)...)
					break
				}
			}
		} else if control.id == "filter:clear" {
			message := "No matching clients. Clear filter or refresh inspection."
			if len(m.snapshot.Clients) == 0 {
				message = "No clients found. Refresh inspection or use Commands."
			}
			left = append(left, row{text: m.note(message)}, m.controlRow(control, i))
		} else {
			primary := control.id == "configure" || control.id == "plan" || control.id == "launch"
			if !primary && !aux {
				right = append(right, row{}, row{text: m.heading("Optional monitor / Help")})
				aux = true
			}
			right = append(right, m.actionRows(control, i, rightWidth, primary)...)
		}
	}
	selected := "None"
	if ok {
		selected = m.public(c.Name)
	}
	left = append(left, row{text: m.note(fmt.Sprintf("Selected: %s / %d matching clients", selected, matches))})
	if !m.wideDashboard() {
		// Full list remains traversable; focus follow keeps each entry readable.
		rs = append(rs, left...)
		rs = append(rs, row{}, row{text: m.rule(width)}, row{})
		return append(rs, right...)
	}
	left, right = m.column(left, leftWidth), m.column(right, rightWidth)
	for i := 0; i < max(len(left), len(right)); i++ {
		l, r := row{text: strings.Repeat(" ", leftWidth)}, row{}
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		divider := "│"
		if m.options.ASCII {
			divider = "|"
		}
		rs = append(rs, row{text: l.text + " " + m.note(divider) + " " + r.text, focus: l.focus || r.focus})
	}
	return rs
}

func (m *model) clientRows(c core.Client, index, width int) []row {
	selected, focused := c.ID == m.selected, index == m.focus
	prefix := "  "
	if focused {
		prefix = "> "
	}
	status := c.Status
	if !c.Installed && !strings.Contains(strings.ToLower(status), "not installed") {
		status += " / Not installed"
	}
	// Keep a bounded list summary, with the exact values in applied detail.
	// Long names/model IDs must not displace management and tier information.
	status = ansi.Truncate(m.public(status), max(1, min(16, width/2)), m.ellipsis())
	nameWidth := max(1, width-8-ansi.StringWidth(status))
	name := prefix + checkbox(m.drafts[c.ID].Enabled) + " " + ansi.Truncate(m.public(c.Name), nameWidth, m.ellipsis())
	gap := strings.Repeat(" ", max(2, width-ansi.StringWidth(name)-ansi.StringWidth(status)))
	label := name + gap + m.statusText(status)
	if selected {
		label = m.heading(name) + gap + m.statusText(status)
	}
	management := "Not managed"
	if c.Managed {
		management = "Applied: managed"
	}
	if selected {
		management += " / Selected"
	}
	if m.drafts[c.ID] != m.baselines[c.ID] {
		management += " / Draft"
	}
	tier := m.public(tierLabel(c.Profile.Tier))
	model := m.public(c.Profile.Model)
	if model == "" {
		model = "Custom configuration"
	}
	model = ansi.Truncate(model, max(1, width-9-ansi.StringWidth(tier)), m.ellipsis()) + " / " + tier
	rows := []row{{text: label, focus: focused, selected: selected}, {text: "      " + m.public(model), selected: selected}, {text: "      " + m.note(management), selected: selected}, {}}
	return m.column(rows, width)
}
func (m *model) detailRows(label, value string, width int) []row {
	if width < 38 {
		return []row{{text: m.note(label)}, {text: value}}
	}
	return []row{{text: m.note(fmt.Sprintf("%-17s", label)) + value}}
}

func (m *model) column(rows []row, width int) []row {
	var result []row
	for _, r := range rows {
		for i, line := range strings.Split(ansi.Wrap(r.text, max(1, width), ""), "\n") {
			line = ansi.Truncate(line, max(1, width), "")
			line += strings.Repeat(" ", max(0, width-ansi.StringWidth(line)))
			if r.focus && i == 0 {
				line = m.focusStyle(line, width)
			} else if r.selected {
				line = m.selectionStyle(line, width)
			}
			result = append(result, row{text: line, focus: r.focus && i == 0, selected: r.selected})
		}
	}
	return result
}
