package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m *model) wideDashboard() bool {
	return m.width >= 100 && m.screen == "Dashboard" && m.modal == "" && m.editor == nil
}

func (m *model) dashboardRows() []row {
	leftWidth := min(44, m.width*2/5)
	rightWidth := m.width - leftWidth - 3
	left := []row{{text: m.heading("CLIENTS")}, {text: m.note("Space manage / Enter configure")}, {}}
	right := []row{{text: m.heading("SELECTED CLIENT")}, {}}
	c, ok := m.client()
	if ok {
		right = append(right, row{text: m.heading(m.public(c.Name))}, row{text: m.public(c.Profile.Model + " / " + tierLabel(c.Profile.Tier))},
			row{text: fmt.Sprintf("Budget %d / Compact %d", c.Profile.Budget, c.Profile.Compact)},
			row{text: "Available tiers: " + tierAvailability(m.appliedTiers())},
			row{text: "Local configuration: " + m.statusText(c.Status)},
			row{text: m.note(fmt.Sprintf("Installed %t / Managed %t", c.Installed, c.Managed))})
		if c.Reason != "" {
			right = append(right, row{text: m.note(m.public(c.Reason))})
		}
	} else {
		right = append(right, row{text: "Select a client to inspect its settings."})
	}
	right = append(right, row{}, row{text: m.heading("ACTIONS")})
	for i, control := range m.controls() {
		if i < 4 {
			continue
		}
		r := m.controlRow(control, i)
		if strings.HasPrefix(control.id, "client:") || control.id == "filter:clear" {
			left = append(left, r)
		} else {
			right = append(right, r)
		}
	}
	left = m.column(left, leftWidth)
	right = m.column(right, rightWidth)
	rs := []row{{text: m.note(m.public("Gateway A / Credential: " + m.snapshot.Credential + " / Authentication and inference not verified"))}, {}}
	for i := 0; i < max(len(left), len(right)); i++ {
		l, r := row{text: strings.Repeat(" ", leftWidth)}, row{}
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		rs = append(rs, row{text: l.text + " | " + r.text, focus: l.focus || r.focus})
	}
	return rs
}

func (m *model) column(rows []row, width int) []row {
	var result []row
	for _, r := range rows {
		for i, line := range strings.Split(ansi.Wrap(r.text, width, ""), "\n") {
			line += strings.Repeat(" ", max(0, width-ansi.StringWidth(line)))
			if r.focus {
				line = m.focusStyle(line, width)
			} else if r.selected {
				line = m.selectionStyle(line, width)
			}
			result = append(result, row{text: line, focus: r.focus && i == 0, selected: r.selected})
		}
	}
	return result
}
