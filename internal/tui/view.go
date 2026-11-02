package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type row struct {
	text            string
	focus, selected bool
}

func (m *model) View() tea.View {
	w, h := max(1, m.width), max(1, m.height)
	title := "oneline-ccc / " + m.screen
	if m.modal != "" {
		title += " / " + m.modal
	}
	if m.editor != nil {
		title += " / Edit " + m.editor.field
	}
	nav := "Dashboard  Models  Review  Jobs"
	if m.modal == "" && m.editor == nil && m.focus < 4 {
		names := []string{"Dashboard", "Models", "Review", "Jobs"}
		names[m.focus] = "> " + names[m.focus]
		nav = strings.Join(names, "  ")
	}
	status := m.public(m.status)
	if m.busy != "" {
		status = "Working: " + m.busy + " | " + status
	}
	header := []string{m.band(ansi.Truncate(title, w, ""), w, true), m.band(ansi.Truncate(nav, w, ""), w, false), ansi.Truncate(status, w, "")}
	footer := []string{"Tab move  Enter act  Esc back", "? Help  Ctrl+P Commands  q Quit"}
	if m.editor != nil {
		footer = []string{"Type / paste  Enter save  Esc cancel", "Key: Tab options; raw value stays hidden"}
	}
	if m.modal == "palette" {
		footer = []string{"Type search  Up/Down  Enter select", "Esc close; disabled actions explain why"}
	}
	lines, focusLine, start, available := m.layout()
	pinFocus := focusLine >= 0 && (focusLine < start || focusLine >= start+available)
	displayCount := available
	if pinFocus {
		displayCount = max(0, available-1)
	}
	out := append([]string(nil), header...)
	for i := start; i < min(len(lines), start+displayCount); i++ {
		r := lines[i]
		line := r.text
		if r.focus {
			line = m.focusStyle(line, w)
		} else if r.selected {
			line = m.selectionStyle(line, w)
		}
		out = append(out, line)
	}
	if pinFocus {
		out = append(out, m.focusStyle(lines[focusLine].text, w))
	}
	for len(out) < h-len(footer) {
		out = append(out, "")
	}
	for _, f := range footer {
		out = append(out, ansi.Truncate(f, w, ""))
	}
	if len(out) > h {
		out = out[:h]
	}
	v := tea.NewView(strings.Join(out, "\n"))
	v.AltScreen = w >= 40 && h >= 12
	if !m.options.NoColor {
		switch m.options.Theme {
		case "dark":
			v.BackgroundColor = lipgloss.Color("#111820")
			v.ForegroundColor = lipgloss.Color("#E6EAF0")
		case "light":
			v.BackgroundColor = lipgloss.Color("#F5F6F8")
			v.ForegroundColor = lipgloss.Color("#17212B")
		}
	}
	return v
}
func (m *model) styles() (lipgloss.Style, lipgloss.Style) {
	base := lipgloss.NewStyle()
	accent := lipgloss.NewStyle().Bold(true)
	switch m.options.Theme {
	case "dark":
		base = base.Foreground(lipgloss.Color("#E6EAF0")).Background(lipgloss.Color("#111820"))
		accent = accent.Foreground(lipgloss.Color("#91C7FF")).Background(lipgloss.Color("#243446"))
	case "light":
		base = base.Foreground(lipgloss.Color("#17212B")).Background(lipgloss.Color("#F5F6F8"))
		accent = accent.Foreground(lipgloss.Color("#164F91")).Background(lipgloss.Color("#DEE6EF"))
	case "mono":
		accent = accent.Reverse(true)
	}
	return base, accent
}
func (m *model) band(s string, w int, bold bool) string {
	if m.options.NoColor {
		return s
	}
	base, accent := m.styles()
	if bold {
		return accent.Width(w).Render(s)
	}
	return base.Bold(true).Width(w).Render(s)
}
func (m *model) focusStyle(s string, w int) string {
	if m.options.NoColor {
		return s
	}
	_, accent := m.styles()
	return accent.Underline(true).Width(w).Render(s)
}
func (m *model) selectionStyle(s string, w int) string {
	if m.options.NoColor {
		return s
	}
	base, _ := m.styles()
	if m.options.Theme == "dark" {
		base = base.Background(lipgloss.Color("#1D2935"))
	}
	if m.options.Theme == "light" {
		base = base.Background(lipgloss.Color("#E7EDF3"))
	}
	return base.Width(w).Render(s)
}
func (m *model) controlRow(c control, index int) row {
	prefix := "  "
	focused := index == m.focus
	if focused {
		prefix = "> "
	}
	label := m.public(c.label)
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
	c, ok := m.client()
	d := m.draft()
	switch m.screen {
	case "Dashboard":
		if len(m.snapshot.Clients) == 0 {
			text("No clients found. Refresh inspection or use Commands.")
		}
		text("Gateway: Gateway A | Authentication / inference: not verified")
	case "Models":
		if ok {
			text("Selected: " + c.Name + " | " + c.Status)
			text(fmt.Sprintf("Installed: %t | Managed: %t | Supported: %t", c.Installed, c.Managed, c.Supported))
			if c.Reason != "" {
				text(c.Reason)
			}
			text("Model: " + d.Profile.Model)
			text(fmt.Sprintf("Context budget: %d | Compact: %d", d.Profile.Budget, d.Profile.Compact))
			text("Default tier: " + tierLabel(d.Profile.Tier))
			text("Available tiers: " + tierAvailability(d.Profile.TiersConfirmed))
			text("Chosen budgets are not verified model capacities.")
			if why := m.editable(); why != "" {
				text(why)
			}
		} else {
			text("No client selected; inspect a client from Dashboard.")
		}
	case "Review":
		if m.plan == nil {
			text("No active review. Request a plan for the selected draft.")
		} else {
			pages := m.reviewPages()
			page := min(m.reviewPage, len(pages)-1)
			text(fmt.Sprintf("Exact changes | Page %d of %d", page+1, len(pages)))
			for _, s := range pages[page] {
				text(s)
			}
		}
	case "Jobs":
		if m.busy == "" {
			text("No running action")
		} else {
			text("Running: " + m.busy)
			text("Cancellation requests await the backend's final outcome.")
		}
		receipts := append(append([]coreReceipt(nil), receiptRows(m.snapshot.Receipts)...), receiptRows(m.receipts)...)
		for _, r := range receipts {
			text(r.text)
		}
	case "Gateway":
		text("Gateway A | Endpoint edit is local only")
		text("Credential: " + m.snapshot.Credential)
		text("Session-only storage is the default; no key reveal.")
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
		text("Local display preferences | Version " + m.options.Version)
		text("Verified native configuration: Codex only.")
	}
	cs := m.controls()
	for i, c := range cs {
		if i < 4 {
			continue
		}
		if m.screen == "Dashboard" && c.id == "configure" && ok {
			text("Selected: " + m.selected + " | Model: " + m.clientModel())
			text(fmt.Sprintf("Applied budget: %d | Default tier: %s", m.snapshotProfileBudget(), tierLabel(m.appliedTier())))
			text("Available tiers: " + tierAvailability(m.appliedTiers()))
			text("Space stages management; Enter opens settings.")
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
func (m *model) clientModel() string          { c, _ := m.client(); return c.Profile.Model }
func (m *model) snapshotProfileBudget() int64 { c, _ := m.client(); return c.Profile.Budget }
func (m *model) appliedTier() string          { c, _ := m.client(); return c.Profile.Tier }
func (m *model) appliedTiers() bool           { c, _ := m.client(); return c.Managed && c.Profile.TiersConfirmed }
