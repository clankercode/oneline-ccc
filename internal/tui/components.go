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
	style := m.surface(role)
	// Lip Gloss preserves nested SGR resets verbatim. Restore the enclosing
	// surface after them so a heading/status cannot punch holes in its band.
	parent := ansi.Style{}.ForegroundColor(style.GetForeground()).BackgroundColor(style.GetBackground())
	if style.GetBold() {
		parent = parent.Bold()
	}
	resume := ansi.ResetStyle + parent.String()
	s = strings.NewReplacer(ansi.ResetStyle, resume, "\x1b[0m", resume).Replace(s)
	return style.Render(s)
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
	ellipsis := m.ellipsis()
	names := []string{"Dashboard", "Models", "Review", "Jobs", "More"}
	var tabs []string
	for i, name := range names {
		active := name == m.screen || (i == 4 && m.screen != "Dashboard" && m.screen != "Models" && m.screen != "Review" && m.screen != "Jobs")
		label := name
		if active {
			label = "[" + name + "]"
		}
		focused := m.modal == "" && m.editor == nil && m.focus == i
		if focused {
			label = "> " + label
		}
		if m.width >= 60 {
			label = " " + label + " "
		}
		if !m.options.NoColor {
			s := m.surface(surfaceBase)
			if active {
				s = m.surface(surfacePrimary)
			}
			if focused {
				s = s.Bold(true).Underline(true)
			}
			label = s.Render(label)
		}
		tabs = append(tabs, label)
	}
	status := strings.Join(strings.Fields(m.public(m.status)), " ")
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
	regionHint := "In " + m.region(m.focus, c) + " | F6 next  Tab/Shift+Tab"
	if m.width >= 60 {
		regionHint = "In " + m.region(m.focus, c) + " | F6 next  [ ] section  Tab/Shift+Tab"
	}
	hints := []string{regionHint, "Up/Down local  Enter act  Esc back"}
	if m.focus < navCount {
		hints[1] = "Left/Right tabs  Enter open  Esc back"
	}
	if strings.HasPrefix(c.id, "client:") || c.id == "enabled" || c.id == "capacity" || c.id == "tiers" || c.id == "ascii" || c.id == "color" || c.id == "motion" {
		hints[1] = "Space toggle  Enter act  Esc back"
	}
	if strings.HasPrefix(c.id, "client:") {
		hints[1] = "Up/Down  Space toggle  Enter  Esc back"
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
	if primary {
		if client, ok := m.client(); ok {
			name := ansi.Truncate(m.public(client.Name), max(4, width-20), m.ellipsis())
			switch c.id {
			case "configure":
				c.label = "Configure " + name
			case "plan":
				c.label = "Review " + name + " draft"
			case "launch":
				c.label = "Launch " + name
			}
		}
	}
	r := m.controlRow(control{id: c.id, label: c.label}, index)
	if primary {
		r.text = strings.TrimRight(r.text, " ")
		if !m.options.NoColor && c.reason == "" {
			role := surfaceActive
			if c.id == "configure" {
				role = surfacePrimary
			}
			r.text = m.surface(role).Padding(0, 1).Render(r.text)
		}
	} else {
		r.text = m.note(r.text)
	}
	rs := []row{r}
	if c.reason != "" {
		if index == m.focus {
			next := "Next: Configure to inspect; Esc keeps drafts."
			if _, ok := m.client(); !ok {
				next = "Next: r refresh inspection; ? Help."
			}
			rs = append(rs, row{text: m.note(next)})
		}
		rs = append(rs, row{text: m.note("Unavailable: " + m.public(c.reason))})
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

func (m *model) ellipsis() string {
	if m.options.ASCII {
		return "..."
	}
	return "…"
}

// Model IDs remain unchanged; the request tier is a separate accented value.
func (m *model) modelDefault(model, tier string) string {
	separator := " · "
	if m.options.ASCII {
		separator = " / "
	}
	return m.public(model) + separator + m.heading(m.public(tierLabel(tier)))
}

func (m *model) dashboardNotice(width int, concise bool) []row {
	title, message := "Select a client", "Choose a client or r refresh local inspection."
	if c, ok := m.client(); ok {
		switch {
		case m.editable() != "":
			title = "Inspection only"
			message = m.editable() + ". Configure to inspect; ? Help for supported setup."
		case m.drafts[c.ID] != m.baselines[c.ID]:
			title = "Draft waiting for review"
			message = "Review staged changes. Nothing changes until you apply the exact plan."
		case c.Managed && strings.EqualFold(c.Status, "CONFIGURED"):
			title = "Configured / no pending draft"
			message = "Launch with applied settings. Authentication and inference remain unverified."
		case c.Managed:
			title = "Managed / no pending draft"
			message = "Inspect local configuration before launch. Auth / API remains unverified."
		default:
			title = "Ready to configure"
			message = "Configure this client, enable management and review the draft before applying."
		}
	}
	if concise {
		switch title {
		case "Inspection only":
			message = "Read only; Configure to inspect. Detail below."
		case "Draft waiting for review":
			message = "Review draft before applying; current settings stay."
		case "Configured / no pending draft":
			message = "Local settings applied; Auth / API unverified."
		case "Managed / no pending draft":
			message = "Inspect local settings; Auth / API unverified."
		case "Ready to configure":
			message = "Configure, enable management, then review."
		}
		message = ansi.Truncate(message, max(1, width-2), m.ellipsis())
	}
	edge := "│ "
	if m.options.ASCII {
		edge = "| "
	}
	rs := []row{{text: m.fill(edge+m.heading(title), width, surfaceRaised)}}
	for _, line := range strings.Split(ansi.Wrap(m.public(message), max(1, width-2), ""), "\n") {
		rs = append(rs, row{text: m.fill(edge+m.note(line), width, surfaceRaised)})
	}
	return rs
}
