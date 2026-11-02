package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// serviceRows presents backend observations separately from requested changes.
// ServiceInfo has no separate ownership attestation; do not invent one here.
func (m *model) serviceRows() []row {
	width := m.contentWidth()
	status := m.snapshot.Service.Status
	if status == "" {
		status = "Not inspected"
	}
	header := []row{{text: m.heading("BACKGROUND MONITOR")}, {text: "Observed: " + m.statusText(status)}}
	switch strings.ToLower(status) {
	case "unknown", "conflict", "unavailable", "failed":
		header = append(header, row{text: m.note("Inspect and reconcile native state before retrying.")})
	}
	if m.busy == "service" {
		header = append(header, row{text: m.statusText("Request pending / final observation awaited")})
	}
	facts := []row{{text: m.heading("BACKEND / OWNERSHIP")}}
	factWidth := width
	if width >= 96 {
		factWidth = (width - 3) / 2
	}
	backend := m.public(m.snapshot.Service.Backend)
	if backend == "" {
		backend = "Not reported"
	}
	facts = append(facts, m.detailRows("Backend", backend, factWidth)...)
	facts = append(facts, m.detailRows("Scope", "This account / user session", factWidth)...)
	facts = append(facts, m.detailRows("Ownership", "Not separately reported", factWidth)...)
	facts = append(facts, row{}, row{text: m.heading("OBSERVATION / RECOVERY")})
	if m.snapshot.Service.Detail != "" {
		facts = append(facts, row{text: m.public(m.snapshot.Service.Detail)})
	}
	guidance := "Optional observation; foreground setup works without it."
	switch strings.ToLower(status) {
	case "not installed":
		guidance = "Inspect status, then review installation if wanted."
	case "unknown", "conflict", "unavailable", "failed":
		guidance = "Inspect and reconcile native state before retrying. Foreground setup remains available."
	case "running":
		guidance = "Observed running; this does not verify client authentication."
	case "stopped":
		guidance = "Observed stopped; Start requests a new native observation."
	}
	facts = append(facts, row{text: m.note(guidance)}, row{}, row{text: m.heading("READ-ONLY MONITOR SCOPE")}, row{text: m.note("No client writes, auto updates, auto repair or model calls. No elevation.")})

	var actions []row
	for i, c := range m.controls() {
		if i < navCount {
			continue
		}
		switch c.id {
		case "service:status":
			actions = append(actions, row{text: m.heading("INSPECT / NO LIFECYCLE CHANGE")})
			c.label = "Inspect monitor status"
		case "service:start":
			actions = append(actions, row{}, row{text: m.heading("RUNNING SERVICE / REQUEST CHANGE")})
			availability := "Backend checks ownership and availability per request."
			switch strings.ToLower(status) {
			case "not installed":
				availability = "Requires installation; inspect and review install first."
			case "unknown", "conflict", "unavailable", "failed":
				availability = "Reconcile native state before lifecycle requests."
			}
			actions = append(actions, row{text: m.note(availability)})
		case "service:install":
			actions = append(actions, row{}, row{text: m.heading("REGISTRATION / REVIEW FIRST")})
			c.label = "Review monitor installation"
		case "service:uninstall":
			c.label = "Review monitor removal"
		case "back":
			actions = append(actions, row{}, row{text: m.note("Client configuration and credentials are separate.")})
		}
		actions = append(actions, m.controlRow(c, i))
	}
	return m.serviceSSHComposition(header, facts, actions)
}

// Both bounded pages use the existing column/surface roles. Compact layouts put
// the first useful action near the observation; all evidence stays pageable.
func (m *model) serviceSSHComposition(header, facts, actions []row) []row {
	width := m.contentWidth()
	var summary []row
	for i, r := range header {
		lines := m.column([]row{r}, width)
		if i == 1 {
			for j := range lines {
				lines[j].text = m.fill(lines[j].text, width, surfaceRaised)
			}
		}
		summary = append(summary, lines...)
	}
	if width < 96 {
		out := append(summary, actions...)
		return append(append(out, row{}), facts...)
	}
	leftWidth := (width - 3) / 2
	left := m.column(facts, leftWidth)
	right := m.column(actions, width-leftWidth-3)
	panel := func(r row, panelWidth int) string {
		role := surfaceBase
		if r.focus {
			role = surfaceActive
		}
		text := m.fill(r.text, panelWidth, role)
		if !m.options.NoColor && !r.focus {
			// The outer focused row supplies bold. Base colors alone do not
			// clear that intensity; restore normal after nested SGR resets too.
			normal := ansi.Style{}.Normal().String()
			text = normal + strings.NewReplacer(ansi.ResetStyle, ansi.ResetStyle+normal, "\x1b[0m", "\x1b[0m"+normal).Replace(text)
		}
		return text
	}
	out := append(summary, row{})
	for i := 0; i < max(len(left), len(right)); i++ {
		var l, r row
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		separator := " | "
		if !m.options.ASCII {
			separator = " │ "
		}
		out = append(out, row{text: panel(l, leftWidth) + panel(row{text: m.note(separator)}, 3) + panel(r, width-leftWidth-3), focus: r.focus})
	}
	return out
}
