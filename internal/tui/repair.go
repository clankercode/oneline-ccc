package tui

import "strings"

// RepairBrief has no target argument: selection is context, while the backend
// inspects the whole snapshot. A retained brief has no selected-client binding.
func (m *model) repairRows() []row {
	width := m.contentWidth()
	wide := width >= 96 && m.height >= 30
	leftWidth := width
	if wide {
		leftWidth = (width - 3) / 2
	}
	selected := "No client selected"
	if c, ok := m.client(); ok {
		selected = m.public(c.Name)
	}
	left := []row{{text: m.heading("Manual repair brief")}, {text: "Selected: " + selected + m.note(" / Manual only")}}
	scope := []row{{text: m.heading("Scope / operator led")}}
	scope = append(scope, m.detailRows("Evidence", "Whole inspected snapshot", leftWidth)...)
	scope = append(scope, m.detailRows("Execution", "No agent is started", leftWidth)...)
	scope = append(scope, row{text: m.note("Client configuration is unchanged by preparation.")})
	if wide {
		left = append(append(left, row{}), scope...)
		left = append(left, row{})
	}
	for i, c := range m.controls() {
		if i >= navCount {
			left = append(left, m.manualActionRows(c, i, c.id == "repair" || c.id == "repair:export")...)
		}
	}
	if !wide {
		left = append(append(left, row{}), scope...)
	}
	left = append(left, row{})
	left = append(left, m.manualNoticeRows("Review before sharing", "Inspect the exact public-safe text. Share only the non-secret context your helper needs.", leftWidth)...)
	left = append(left, row{text: m.note("Export writes to terminal scrollback. Review suggestions before ordinary Apply.")})
	right := []row{{text: m.heading("Exact brief / redacted")}}
	switch {
	case m.busy == "repair":
		right = append(right, row{text: m.statusText("Preparing redacted diagnostics")}, row{text: m.note("Wait for the final outcome before exporting.")})
	case m.repair == "":
		right = append(right, row{text: m.statusText("No brief prepared")}, row{text: m.note("Prepare the manual brief to inspect its exact contents here.")})
	default:
		right = append(right, row{text: m.note("Prepared text is retained; selection changes do not retarget it.")})
	}
	if m.repair != "" {
		// Match exportBrief's public projection exactly; keep every line and
		// value. Ordinary document paging makes long briefs fully reachable.
		right = append(right, row{}, row{text: m.public(m.repair)})
	}
	if !wide {
		return append(append(left, row{}), right...)
	}
	left, right = m.column(left, leftWidth), m.column(right, width-leftWidth-3)
	var rs []row
	separator := " │ "
	if m.options.ASCII {
		separator = " | "
	}
	for i := 0; i < max(len(left), len(right)); i++ {
		l, r := row{}, row{}
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		rs = append(rs, row{text: m.fill(l.text, leftWidth, surfaceBase) + m.note(separator) + r.text, focus: l.focus || r.focus})
	}
	return rs
}

// Wrap before filling: notice text must remain available at narrow widths.
func (m *model) manualNoticeRows(title, message string, width int) []row {
	rows := m.column([]row{{text: m.heading(title)}, {text: m.note(message)}}, width)
	for i := range rows {
		rows[i].text = m.fill(rows[i].text, width, surfaceRaised)
	}
	return rows
}

// Keep the production control's label, identity, reason and focus. Disabled
// actions use supporting text rather than a primary-action surface.
func (m *model) manualActionRows(c control, index int, primary bool) []row {
	r := m.controlRow(control{id: c.id, label: c.label}, index)
	if primary && c.reason == "" {
		if !m.options.NoColor {
			r.text = m.surface(surfacePrimary).Render(strings.TrimRight(r.text, " "))
		}
	} else {
		r.text = m.note(r.text)
	}
	if c.reason != "" {
		r.text += "\n" + m.note("  Unavailable: "+m.public(c.reason))
	}
	return []row{r}
}
