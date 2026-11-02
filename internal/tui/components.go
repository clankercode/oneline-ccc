package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// All shell, list and action surfaces use these roles. Explicit themes pair
// foreground/background; NoColor retains the same spacing and literal cues.
type surfaceRole uint8

const (
	surfaceBase surfaceRole = iota
	surfaceRaised
	surfaceSelected
	surfaceActive
	surfacePrimary
)

func (m *model) surface(role surfaceRole) lipgloss.Style {
	fg, bg := "#E6EAF0", "#111820"
	light := m.options.Theme == "light"
	if light {
		fg, bg = "#17212B", "#F5F6F8"
	}
	if m.options.Theme == "mono" {
		fg, bg = "#EEEEEE", "#171717"
	}
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Background(lipgloss.Color(bg))
	switch role {
	case surfaceRaised:
		bg = "#1B2733"
		if light {
			bg = "#E3E9F0"
		}
		if m.options.Theme == "mono" {
			bg = "#242424"
		}
	case surfaceSelected:
		bg = "#23384B"
		if light {
			bg = "#D9E5F2"
		}
		if m.options.Theme == "mono" {
			bg = "#303030"
		}
	case surfaceActive:
		fg, bg = "#91C7FF", "#23384B"
		if light {
			fg, bg = "#164F91", "#D9E5F2"
		}
		if m.options.Theme == "mono" {
			fg, bg = "#FFFFFF", "#363636"
		}
		s = s.Bold(true)
	case surfacePrimary:
		fg, bg = "#111820", "#91C7FF"
		if light {
			fg, bg = "#FFFFFF", "#164F91"
		}
		if m.options.Theme == "mono" {
			fg, bg = "#171717", "#EEEEEE"
		}
		s = s.Bold(true)
	}
	return s.Foreground(lipgloss.Color(fg)).Background(lipgloss.Color(bg))
}

func (m *model) inset() int {
	if m.width >= 60 {
		return 2
	}
	return 1
}
func (m *model) contentWidth() int { return max(1, m.width-2*m.inset()) }
func (m *model) fill(s string, width int, role surfaceRole) string {
	s = ansi.Truncate(s, max(1, width), "")
	s += strings.Repeat(" ", max(0, width-ansi.StringWidth(s)))
	if m.options.NoColor {
		return s
	}
	return m.surface(role).Render(s)
}
func (m *model) shellLine(s string, role surfaceRole) string {
	return m.fill(strings.Repeat(" ", m.inset())+s, max(1, m.width), role)
}
func (m *model) rule(width int) string {
	char := "─"
	if m.options.ASCII {
		char = "-"
	}
	return m.note(strings.Repeat(char, max(1, width)))
}

func (m *model) shellHeader() []string {
	title := "oneline-ccc / " + m.screen
	if m.modal != "" {
		title += " / " + m.modal
	}
	if m.editor != nil {
		title += " / Edit " + m.editor.field
	}
	width := m.contentWidth()
	ellipsis := "…"
	if m.options.ASCII {
		ellipsis = "..."
	}
	names := []string{"Dashboard", "Models", "Review", "Jobs", "More"}
	var tabs []string
	for i, name := range names {
		label := name
		focused := m.modal == "" && m.editor == nil && m.focus == i
		if focused {
			label = "> " + label
		}
		if m.width >= 60 {
			label = " " + label + " "
		}
		active := name == m.screen || (i == 4 && m.screen != "Dashboard" && m.screen != "Models" && m.screen != "Review" && m.screen != "Jobs")
		if !m.options.NoColor {
			s := m.surface(surfaceBase)
			if active {
				s = m.surface(surfaceActive)
			}
			if focused {
				s = s.Bold(true).Underline(true)
			}
			label = s.Render(label)
		}
		tabs = append(tabs, label)
	}
	status := m.public(m.status)
	if m.busy != "" && !strings.HasPrefix(status, "Working:") {
		status = "Working: " + m.public(m.busy) + " | " + status
	}
	if status == "" {
		status = "Local settings; drafts kept"
	}
	return []string{
		m.shellLine(m.heading(ansi.Truncate(title, width, ellipsis)), surfaceRaised),
		m.shellLine(strings.Join(tabs, " "), surfaceBase),
		m.shellLine(m.note(ansi.Truncate(status, width, ellipsis)), surfaceRaised),
	}
}

func (m *model) footerHints() []string {
	c := m.focused()
	hints := []string{"F6 " + m.region(m.focus, c) + "  [ ] section  Tab/Shift+Tab", "Up/Down local  Enter act  Esc back"}
	if m.focus < navCount {
		hints[1] = "Left/Right tabs  Enter open  Esc back"
	}
	if strings.HasPrefix(c.id, "client:") || c.id == "enabled" || c.id == "capacity" || c.id == "tiers" || c.id == "ascii" || c.id == "color" || c.id == "motion" {
		hints[1] = "Space toggle  Enter act  Esc back"
	}
	if m.modal == "more" {
		hints = []string{"Tab/Shift+Tab menu", "Up/Down menu  Enter open  Esc back"}
	}
	if m.modal == "help" {
		hints = []string{"Tab/Shift+Tab pages", "Up/Down pages  Enter/Esc close"}
	}
	if m.modal != "" && m.modal != "help" && m.modal != "more" && m.modal != "palette" {
		hints = []string{"Tab/Shift+Tab choice", "Arrows choose  Enter act  Esc back"}
	}
	if m.modal == "palette" {
		hints = []string{"Type search  Up/Down  Enter select", "Esc close; disabled actions explain why"}
	}
	if m.editor != nil {
		hints = []string{"Type / paste  Enter save  Esc cancel", "Left/Right caret; text keys stay here"}
		if m.editor.field == "key" {
			hints = []string{"Tab/Shift+Tab options  Esc cancel", ""}
			switch m.editor.focus {
			case 0:
				hints = []string{"Type/paste  Left/Right caret", "Tab  Enter focus Save  Esc cancel"}
			case 1:
				hints[1] = "Space/Enter toggle persistence"
			case 2:
				hints[1] = "Enter save key"
			case 3:
				hints[1] = "Enter drop unsaved key"
			}
		}
	}
	return hints
}

// Reserve actual wrapped rows, including the document scope/scroll line.
// The range is rendered later but has exactly the same one-row footprint.
func (m *model) shellFooter(scope string) []string {
	var lines []string
	if m.editor != nil {
		lines = append(lines, m.shellLine(scope, surfaceRaised))
	}
	for _, hint := range m.footerHints() {
		for _, line := range wrapHints(hint, m.contentWidth()) {
			lines = append(lines, m.shellLine(line, surfaceRaised))
		}
	}
	if m.editor == nil && m.modal == "" {
		label := "? Help  Ctrl+P Commands  | "
		if m.width < 60 {
			label = "? Help Ctrl+P | "
		}
		lines = append(lines, m.shellLine(label+scope, surfaceRaised))
	} else if m.editor == nil {
		lines = append(lines, m.shellLine(scope, surfaceRaised))
	}
	return lines
}
func (m *model) scrollScope(start, count, total int) string {
	scope := m.screen
	if m.modal != "" {
		scope = m.modal
	}
	if m.editor != nil {
		scope = "Edit " + m.editor.field
	}
	if total > count {
		if m.width < 60 {
			return fmt.Sprintf("%d-%d/%d PgUp/Dn", start+1, min(total, start+count), total)
		}
		return fmt.Sprintf("%s %d-%d/%d PgUp/PgDn", scope, start+1, min(total, start+count), total)
	}
	return scope
}

func (m *model) actionRows(c control, index, width int, primary bool) []row {
	r := m.controlRow(control{id: c.id, label: c.label}, index)
	if primary {
		r.text = strings.TrimRight(r.text, " ")
		if !m.options.NoColor && c.reason == "" {
			r.text = m.surface(surfacePrimary).Padding(0, 1).Render(r.text)
		}
	} else {
		r.text = m.note(r.text)
	}
	rs := []row{r}
	if c.reason != "" {
		rs = append(rs, row{text: m.note("    Unavailable: " + m.public(c.reason))})
		if index == m.focus {
			rs = append(rs, row{text: m.note("    Next: Configure to inspect; Esc keeps drafts.")})
		}
	}
	return m.column(rs, width)
}

// Double spaces separate key/action phrases. Keep a key and its action together
// when reflowing hints, rather than stranding e.g. "Esc" above "cancel".
func wrapHints(hint string, width int) []string {
	var lines []string
	current := ""
	for _, phrase := range strings.Split(hint, "  ") {
		for _, part := range strings.Split(ansi.Wrap(phrase, width, ""), "\n") {
			if current != "" && ansi.StringWidth(current)+2+ansi.StringWidth(part) > width {
				lines = append(lines, current)
				current = ""
			}
			if current != "" {
				current += "  "
			}
			current += part
		}
	}
	return append(lines, current)
}
