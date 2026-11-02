package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type row struct {
	text            string
	focus, selected bool
}

func (m *model) View() tea.View {
	w, h := max(1, m.width), max(1, m.height)
	header := m.shellHeader()
	if m.screen == "Models" && m.modal == "" && m.editor == nil {
		header[0] = m.modelsHeader()
	}
	lines, focusLine, start, available := m.layout()
	footer := m.shellFooter("")
	out := append([]string(nil), header...)
	remaining := available
	// Dialog/editor scope is independent of the scrolling body.
	if start > 0 && (m.modal != "" || m.editor != nil) && len(lines) > 0 && remaining > 1 {
		out = append(out, m.bodyLine(lines[0].text))
		remaining--
	}
	pinned := focusLine >= 0 && (focusLine < start || focusLine >= start+remaining)
	reserve := 0
	if pinned {
		// Manual paging advances by available-1 rows. Keep one anchor
		// so document rows remain contiguous and none are skipped.
		reserve = min(remaining, 1)
	}
	visible := remaining - reserve
	footer = m.shellFooter(m.scrollScope(start, visible, len(lines)))
	for i := start; i < min(len(lines), start+visible); i++ {
		out = append(out, m.bodyLine(lines[i].text))
	}
	if pinned {
		for i := focusLine; i < min(len(lines), focusLine+reserve); i++ {
			out = append(out, m.bodyLine(lines[i].text))
		}
	}
	for len(out) < h-len(footer) {
		out = append(out, m.bodyLine(""))
	}
	out = append(out, footer...)
	if len(out) > h {
		out = out[:h]
	}
	v := tea.NewView(strings.Join(out, "\n"))
	v.AltScreen = w >= 40 && h >= 12
	if !m.options.NoColor {
		switch m.options.Theme {
		case "dark":
			v.BackgroundColor, v.ForegroundColor = lipgloss.Color("#111820"), lipgloss.Color("#E6EAF0")
		case "light":
			v.BackgroundColor, v.ForegroundColor = lipgloss.Color("#F5F6F8"), lipgloss.Color("#17212B")
		case "mono":
			v.BackgroundColor, v.ForegroundColor = lipgloss.Color("#171717"), lipgloss.Color("#EEEEEE")
		}
	}
	return v
}
func (m *model) bodyLine(s string) string {
	if m.screen == "Review" && m.modal == "" && m.editor == nil {
		return m.fill(s, max(1, m.width), surfaceBase)
	}
	return m.shellLine(s, surfaceBase)
}
func (m *model) focusStyle(s string, w int) string {
	if m.options.NoColor {
		return s
	}
	return m.fill(s, w, surfaceActive)
}
func (m *model) selectionStyle(s string, w int) string { return m.fill(s, w, surfaceSelected) }
func (m *model) controlRow(c control, index int) row {
	prefix := "  "
	focused := index == m.focus
	if focused {
		prefix = "> "
	}
	label := m.public(c.label)
	if strings.HasPrefix(c.id, "client:") {
		for _, client := range m.snapshot.Clients {
			if c.id == "client:"+client.ID && client.Status != "" {
				label = strings.Replace(label, m.public(client.Status), m.statusText(client.Status), 1)
				break
			}
		}
	}
	if c.reason != "" {
		if m.screen == "Review" {
			label += " [locked]"
		} else {
			label += " [Unavailable: " + m.public(c.reason) + "]"
		}
	}
	return row{prefix + label, focused, strings.HasPrefix(c.id, "client:") && c.id == "client:"+m.selected}
}
func (m *model) screenRows() []row {
	var rs []row
	text := func(s string) { rs = append(rs, row{text: m.public(s)}) }
	switch m.screen {
	case "Models":
		return m.modelsRows()
	case "Review":
		return m.reviewRows()
	case "Jobs":
		return m.jobsRows()
	case "Gateway":
		return m.gatewayRows()
	case "Service":
		text("Optional background observation; setup works without it.")
		text("Backend: " + m.snapshot.Service.Backend)
		text("Observed: " + m.snapshot.Service.Status)
		text(m.snapshot.Service.Detail)
	case "Repair":
		text("Manual repair brief only; no automatic agent execution.")
		if m.repair != "" {
			text(m.repair)
		}
	case "SSH":
		text("Explicit handoff only. This does not enable a server.")
		text("The backend may report this action unavailable.")
	case "Utilities":
		text("Utility install/update is unavailable: no verified installer.")
		text("No broader adapters are promised by this build.")
	case "Settings":
		return m.settingsRows()
	}
	cs := m.controls()
	for i, c := range cs {
		if i < navCount {
			continue
		}

		rs = append(rs, m.controlRow(c, i))
	}
	return rs
}
func tierLabel(t string) string {
	if t == "" || t == "default" {
		return "Standard"
	}
	return t
}
func tierAvailability(confirmed bool) string {
	if confirmed {
		return "Up to Ultrafast"
	}
	return "Not verified"
}
func (m *model) appliedTiers() bool { c, _ := m.client(); return c.Managed && c.Profile.TiersConfirmed }
