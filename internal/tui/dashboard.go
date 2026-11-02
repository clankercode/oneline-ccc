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
	if m.height <= 12 {
		rs = []row{{text: m.heading(fmt.Sprintf("Your clients / %d managed", managed))}}
	}
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
		right = append(right, m.detailRows("Model / default", m.modelDefault(c.Profile.Model, c.Profile.Tier), rightWidth)...)
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
	if !m.wideDashboard() {
		return m.compactDashboardRows(right, matches, managed)
	}
	right = append(right, row{})
	right = append(right, m.dashboardNotice(rightWidth, false)...)
	right = append(right, row{}, row{text: m.heading("NEXT ACTIONS")})
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
			if primary {
				right = append(right, row{})
			}
		}
	}
	selected := "None"
	if ok {
		selected = m.public(c.Name)
	}
	left = append(left, row{text: m.note(fmt.Sprintf("Selected: %s / %d matching clients", selected, matches))})
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
	if m.height <= 12 && !selected && m.drafts[c.ID] != m.baselines[c.ID] {
		staged := "off"
		if m.drafts[c.ID].Enabled {
			staged = "on"
		}
		// Reserve state before fitting the name; the tiny non-selected entry
		// has no separate management/draft row to recover by scrolling.
		status += " / Draft: " + staged
	}
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
	model = m.modelDefault(ansi.Truncate(model, max(1, width-9-ansi.StringWidth(tier)), m.ellipsis()), tier)
	rows := []row{{text: label, focus: focused, selected: selected}, {text: "      " + model, selected: selected}, {text: "      " + m.note(management), selected: selected}, {}}
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

// Compact terminals show a window on the logical client list, then the selected
// summary and primary actions. All control IDs/order stay in controls(); local
// arrows move this window, and full applied detail remains in the document.
func (m *model) compactDashboardRows(details []row, matches, managed int) []row {
	width := m.contentWidth()
	cs := m.controls()
	chosen, position := -1, 0
	var clients []int
	for i, c := range cs {
		if strings.HasPrefix(c.id, "client:") {
			clients = append(clients, i)
			if chosen < 0 || c.id == "client:"+m.selected {
				chosen = i
			}
		}
	}
	for p, i := range clients {
		if i == m.focus {
			chosen = i
			position = p
			break
		}
		if i == chosen {
			position = p
		}
	}
	var rs []row
	tiny := m.height <= 12
	if !tiny {
		rs = append(rs, row{text: m.heading("Your clients") + m.note(fmt.Sprintf(" / %d of %d / %d managed", min(matches, position+1), matches, managed))}, row{text: m.note("Up/Down clients / Space manage / / filter")})
		if m.filter != "" {
			rs = append(rs, row{text: m.note(ansi.Truncate("Filter: "+m.public(m.filter), width, m.ellipsis()))})
		}
	}
	if chosen >= 0 {
		id := strings.TrimPrefix(cs[chosen].id, "client:")
		for _, c := range m.snapshot.Clients {
			if c.ID != id {
				continue
			}
			other := c.ID != m.selected
			if other {
				selected, _ := m.client()
				rs = append(rs, row{text: m.note(ansi.Truncate("Selected: "+m.public(selected.Name)+" / applied settings", width, m.ellipsis()))})
			}
			entry := m.clientRows(c, chosen, width)
			if tiny && other {
				management := "Applied: off"
				if c.Managed {
					management = "Applied: on"
				}
				tier := m.public(tierLabel(c.Profile.Tier))
				tail := "  " + m.note(management)
				modelWidth := max(1, width-2-ansi.StringWidth(m.modelDefault("", tier))-ansi.StringWidth(tail))
				name := ansi.Truncate(m.public(c.Profile.Model), modelWidth, m.ellipsis())
				entry = []row{entry[0], {text: "  " + m.modelDefault(name, tier) + tail}}
			} else if tiny {
				management := "not managed / Selected"
				if c.Managed {
					management = "Applied: managed / Selected"
				}
				if m.drafts[c.ID] != m.baselines[c.ID] {
					management = "not managed / Selected / Draft"
					if c.Managed {
						management = "managed / Selected / Draft"
					}
				}
				entry = []row{entry[0], entry[1], {text: "  " + m.note(fmt.Sprintf("%s %d/%d", management, position+1, matches)), selected: true}}
			} else {
				entry = entry[:len(entry)-1]
			}
			rs = append(rs, entry...)
			break
		}
	} else {
		message := "No matching clients. Clear filter or refresh."
		if len(m.snapshot.Clients) == 0 {
			message = "No clients found. r refresh inspection."
		}
		rs = append(rs, row{text: m.note(message)})
		for i, c := range cs {
			if c.id == "filter:clear" {
				rs = append(rs, m.controlRow(c, i))
			}
		}
	}
	if !tiny {
		if c, ok := m.client(); ok {
			rs = append(rs, row{text: m.note(fmt.Sprintf("Budget %d / Compact %d", c.Profile.Budget, c.Profile.Compact))}, row{text: m.note("Available tiers: " + tierAvailability(m.appliedTiers()))})
		}
		rs = append(rs, m.dashboardNotice(width, true)...)
		rs = append(rs, row{text: m.heading("NEXT ACTIONS")})
	}
	for i, c := range cs {
		if c.id == "configure" || c.id == "plan" || c.id == "launch" {
			rs = append(rs, m.actionRows(c, i, width, true)...)
		}
	}
	rs = append(rs, row{}, row{text: m.heading("Optional monitor / Help")})
	for i, c := range cs {
		if i < navCount || strings.HasPrefix(c.id, "client:") || c.id == "filter:clear" || c.id == "configure" || c.id == "plan" || c.id == "launch" {
			continue
		}
		rs = append(rs, m.actionRows(c, i, width, false)...)
	}
	rs = append(rs, row{}, row{text: m.rule(width)})
	return append(rs, details...)
}
