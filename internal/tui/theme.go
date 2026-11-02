package tui

import (
	"charm.land/lipgloss/v2"
	"strings"
)

func (m *model) heading(s string) string {
	if m.options.NoColor {
		return s
	}
	_, accent := m.styles()
	return accent.Background(lipgloss.NoColor{}).Bold(true).Render(s)
}
func (m *model) note(s string) string {
	if m.options.NoColor {
		return s
	}
	color := "#B3BDC9"
	if m.options.Theme == "light" {
		color = "#465463"
	}
	if m.options.Theme == "mono" {
		return s
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(s)
}
func (m *model) statusText(value string) string {
	s := m.public(value)
	if m.options.NoColor || m.options.Theme == "mono" {
		return s
	}
	green, amber, red := "#91D6A5", "#F0CC81", "#F5A3A3"
	if m.options.Theme == "light" {
		green, amber, red = "#23613A", "#704700", "#9B2727"
	}
	color := amber
	upper := strings.ToUpper(s)
	switch {
	case strings.Contains(upper, "FAILED"), strings.Contains(upper, "BLOCKED"), strings.Contains(upper, "CONFLICT"), strings.Contains(upper, "UNKNOWN"):
		color = red
	case strings.Contains(upper, "CONFIGURED"), strings.Contains(upper, "SUCCEEDED"), strings.Contains(upper, "COMPLETED"):
		color = green
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(s)
}
func (m *model) renderRow(r row, width int) string {
	if m.wideDashboard() {
		return r.text
	}
	if r.focus {
		return m.focusStyle(r.text, width)
	}
	if r.selected {
		return m.selectionStyle(r.text, width)
	}
	return r.text
}
