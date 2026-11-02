package tui

import (
	"fmt"
	"strings"
)

func (m *model) modalRows() []row {
	var rs []row
	text := func(s string) { rs = append(rs, row{text: m.note(m.public(s))}) }
	switch m.modal {
	case "more":
		rs = m.layerTitle("More destinations | Enter opens section")
		purposes := []string{"Endpoint and hidden credentials", "Optional background observation", "Redacted manual repair brief", "Explicit terminal helper handoff", "Install/update availability", "Display and navigation"}
		compact := m.height-len(m.shellHeader())-len(m.shellFooter("")) < 21
		if !compact {
			text("Current section: " + m.screen + " | Esc returns here")
			rs = append(rs, row{})
		}
		for i, name := range sections[4:] {
			// In a tiny viewport put the purpose immediately before its focus
			// anchor, so following focus cannot strand the explanatory line.
			if compact {
				text("  " + purposes[i])
			}
			prefix := "  "
			if i == m.modalFocus {
				prefix = "> "
			}
			rs = append(rs, row{text: prefix + name, focus: i == m.modalFocus})
			if !compact {
				text("  " + purposes[i])
				rs = append(rs, row{})
			}
		}
		// Leave room for View's pinned scope row at the end of the menu.
		rs = append(rs, row{})
	case "help":
		pages, _ := m.helpPages()
		page := max(0, min(m.modalFocus, len(pages)-1))
		rs = append(rs, row{text: m.heading(fmt.Sprintf("Shortcut help | Page %d of %d", page+1, len(pages)))})
		for _, line := range pages[page] {
			// helpPages sanitizes source text before applying Help styles.
			rs = append(rs, row{text: line})
		}
		rs = append(rs, row{text: helpClose, focus: true})
	case "palette":
		query := layerCaret(m.public(m.query), "", max(0, m.layerWidth()-9))
		rs = m.layerBand(m.heading("Search: ")+query, surfaceRaised)
		cs := m.paletteCommands()
		if len(cs) == 0 {
			rs = append(rs, row{})
			rs = append(rs, row{text: m.heading("> No matching commands"), focus: true})
			text("No commands match. Backspace edits; Esc closes.")
			return rs
		}
		selected := max(0, min(m.modalFocus, len(cs)-1))
		text(fmt.Sprintf("Commands / %d results / %d of %d", len(cs), selected+1, len(cs)))
		var choices [][]row
		for i, c := range cs {
			block := m.layerChoice(c.label, i == selected, false)
			if c.reason != "" {
				block = append(block, m.layerBand(m.statusText("  Unavailable: "+c.reason), surfaceRaised)...)
			} else if i == selected && c.key != "" && m.height > 12 {
				block = append(block, row{text: m.note("  " + c.key)})
			}
			choices = append(choices, block)
		}
		return m.layerMenu(rs, choices, selected)
	default:
		var title, scope, outcome string
		switch {
		case m.modal == "quit":
			title, scope, outcome = "Leave with unapplied drafts?", "Client files stay; unapplied drafts are dropped.", "Leave without applying"
			if m.busy != "" {
				title = "Running: " + m.busy
				scope = "Request cancellation; wait for its final outcome."
				outcome = "Request cancel; stay until outcome"
			}
		case m.modal == "discard":
			title = "Discard only " + m.selected + " draft?"
			scope, outcome = "Other drafts stay; client files stay.", "Discard "+m.selected+" draft"
		case m.modal == "ssh":
			title, scope, outcome = "SSH command handoff", "Helper owns the terminal until exit. Server changes are separate.", "Hand off terminal"
		case strings.HasPrefix(m.modal, "service:"):
			action := strings.TrimPrefix(m.modal, "service:")
			title, scope = "Confirm monitor "+action+"?", "Client configuration and credentials are separate."
			outcome = "Install monitor"
			if action == "uninstall" {
				outcome = "Uninstall monitor"
			}
		}
		rs = m.layerTitle(title)
		if m.height > 12 {
			rs = append(rs, row{})
			if m.modal == "quit" {
				dirty := 0
				for id, d := range m.drafts {
					if d != m.baselines[id] {
						dirty++
					}
				}
				text(fmt.Sprintf("Unapplied client drafts: %d", dirty))
				text("Session-only credentials end when the console closes.")
			}
			if strings.HasPrefix(m.modal, "service:") {
				text("Backend: " + m.snapshot.Service.Backend + " | Observed: " + m.snapshot.Service.Status)
				text("Optional background observation; setup works without it.")
			}
			if m.modal == "discard" {
				text("Prepared review is cleared. Applied configuration stays.")
			}
			if m.modal == "ssh" {
				text("Explicit command handoff; this does not enable a server.")
				text("Review account and firewall changes in the helper.")
			}
		}
		rs = append(rs, m.layerBand(m.note(m.public(scope)), surfaceRaised)...)
		if m.height > 12 {
			rs = append(rs, row{}, row{text: m.heading("Choose explicitly / Esc keeps working")})
		}
		rs = append(rs, m.layerChoice("Back; keep working", m.modalFocus == 0, false)...)
		rs = append(rs, m.layerChoice("Confirm: "+outcome, m.modalFocus == 1, true)...)
	}
	return rs
}
func (m *model) editorRows() []row {
	e := m.editor
	rs := m.layerTitle("Editing " + e.field)
	if e.field == "key" {
		if m.height > 12 {
			rs = append(rs, row{text: m.note("The value stays hidden. Esc drops the unsaved key.")}, row{})
			rs = append(rs, m.layerBand(m.note("Storage / choose explicitly"), surfaceRaised)...)
		}
		labels := []string{"API key: [hidden]", checkbox(m.persistent) + " Persist using backend credential storage", "Save key", "Back; drop unsaved key"}
		for i, label := range labels {
			if i == 2 && m.height > 12 {
				rs = append(rs, row{}, row{text: m.note("Session only unless persistence is selected.")})
				rs = append(rs, row{text: m.note("Saving uses the backend; a storage request is not a saved key.")}, row{})
			}
			rs = append(rs, m.layerChoice(label, i == e.focus, i == 2)...)
		}
		return rs
	}
	label, boundary, hint := "Value", "Draft only", "Enter keeps this field; review before applying."
	switch e.field {
	case "filter":
		label, boundary, hint = "Client name", "Enter applies the filter", "Filtering preserves the selected client and drafts."
	case "model":
		label, hint = "Model ID", "Preserve the exact ID; availability is verified separately."
	case "budget":
		label, hint = "Budget in tokens", "Positive whole tokens; recalculates the compact budget."
	case "compact":
		label, hint = "Compact tokens", "Positive whole tokens; chosen budget is not verified capacity."
	case "endpoint":
		label, boundary, hint = "Endpoint URL", "Local-only endpoint", "Public views use Gateway A. Use an absolute HTTP or HTTPS URL."
	case "credential":
		label, boundary, hint = "Environment variable name", "Variable name, not the key", "The draft is validated at review; the credential stays separate."
	}
	if m.height > 12 {
		rs = append(rs, row{text: m.note("Esc keeps the previous draft")}, row{})
	}
	rs = append(rs, row{text: m.heading(label)})
	value := layerCaret(string(e.value[:e.pos]), string(e.value[e.pos:]), max(0, m.layerWidth()-3))
	rs = append(rs, row{text: m.fill("> "+value, m.layerWidth(), surfaceRaised), focus: true})
	rs = append(rs, m.layerBand(m.note(boundary), surfaceRaised)...)
	if m.height > 12 {
		rs = append(rs, row{text: m.note(hint)}, row{})
	}
	if e.field != "filter" {
		rs = append(rs, row{text: m.note("Enter commits this field to the draft only.")})
	}
	return rs
}
