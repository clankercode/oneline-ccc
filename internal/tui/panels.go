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
		var details []row
		detail := func(s string) { details = append(details, row{text: m.note(m.public(s))}) }
		if m.modal == "quit" {
			dirty := 0
			for id, d := range m.drafts {
				if d != m.baselines[id] {
					dirty++
				}
			}
			detail(fmt.Sprintf("Unapplied client drafts: %d", dirty))
			detail("Session-only credentials end when the console closes.")
		}
		if strings.HasPrefix(m.modal, "service:") {
			detail("Backend: " + m.snapshot.Service.Backend + " | Observed: " + m.snapshot.Service.Status)
			detail("Optional background observation; setup works without it.")
		}
		if m.modal == "discard" {
			detail("Prepared review is cleared. Applied configuration stays.")
		}
		if m.modal == "ssh" {
			detail("Explicit command handoff; this does not enable a server.")
			detail("Review account and firewall changes in the helper.")
		}
		scopeRows := m.layerBand(m.note(m.public(scope)), surfaceRaised)
		choices := m.layerChoice("Back; keep working", m.modalFocus == 0, false)
		choices = append(choices, m.layerChoice("Confirm: "+outcome, m.modalFocus == 1, true)...)
		if !m.layerFits(rs, details, scopeRows, choices) {
			details = nil
		}
		actionHint := []row{{text: m.heading("Choose explicitly / Esc keeps working")}}
		if !m.layerFits(rs, details, scopeRows, actionHint, choices) {
			actionHint = nil
		}
		gap := []row{{}}
		if m.layerFits(rs, gap, details, scopeRows, actionHint, choices) {
			rs = append(rs, gap...)
		}
		rs = append(rs, details...)
		rs = append(rs, scopeRows...)
		if m.layerFits(rs, gap, actionHint, choices) {
			rs = append(rs, gap...)
		}
		rs = append(rs, actionHint...)
		rs = append(rs, choices...)
	}
	return rs
}
func (m *model) editorRows() []row {
	e := m.editor
	rs := m.layerTitle("Editing " + e.field)
	if e.field == "key" {
		var inputs, actions []row
		labels := []string{"API key: [hidden]", checkbox(m.persistent) + " Persist using backend credential storage", "Save key", "Back; drop unsaved key"}
		for i, label := range labels {
			choice := m.layerChoice(label, i == e.focus, i == 2)
			if i < 2 {
				inputs = append(inputs, choice...)
			} else {
				actions = append(actions, choice...)
			}
		}
		notes := []row{{text: m.note("Session only unless persistence is selected.")}}
		if !m.layerFits(rs, inputs, notes, actions) {
			notes = nil
		}
		backendNote := []row{{text: m.note("Saving uses the backend; a storage request is not a saved key.")}}
		if m.layerFits(rs, inputs, notes, backendNote, actions) {
			notes = append(notes, backendNote...)
		}
		intro := []row{{text: m.note("The value stays hidden. Esc drops the unsaved key.")}}
		if !m.layerFits(rs, intro, inputs, notes, actions) {
			intro = nil
		}
		storageTitle := m.layerBand(m.note("Storage / choose explicitly"), surfaceRaised)
		if !m.layerFits(rs, intro, storageTitle, inputs, notes, actions) {
			storageTitle = nil
		}
		rs = append(rs, intro...)
		gap := []row{{}}
		if len(intro) > 0 && m.layerFits(rs, gap, storageTitle, inputs, notes, actions) {
			rs = append(rs, gap...)
		}
		rs = append(rs, storageTitle...)
		rs = append(rs, inputs...)
		if len(notes) > 0 && m.layerFits(rs, gap, notes, actions) {
			rs = append(rs, gap...)
		}
		rs = append(rs, notes...)
		if m.layerFits(rs, gap, actions) {
			rs = append(rs, gap...)
		}
		rs = append(rs, actions...)
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
	value := layerCaret(string(e.value[:e.pos]), string(e.value[e.pos:]), max(0, m.layerWidth()-3))
	// Field meaning, value/caret and complete guidance are the required form.
	// Redundant commit/cancel prose and blank separators spend only spare rows.
	form := []row{{text: m.heading(label)}, {text: m.fill("> "+value, m.layerWidth(), surfaceRaised), focus: true}}
	form = append(form, m.layerBand(m.note(boundary), surfaceRaised)...)
	form = append(form, row{text: m.note(hint)})
	var commitHint []row
	if e.field != "filter" {
		commitHint = []row{{text: m.note("Enter commits this field to the draft only.")}}
		if !m.layerFits(rs, form, commitHint) {
			commitHint = nil
		}
	}
	intro := []row{{text: m.note("Esc keeps the previous draft")}}
	if !m.layerFits(rs, intro, form, commitHint) {
		intro = nil
	}
	gap := []row{{}}
	rs = append(rs, intro...)
	if len(intro) > 0 && m.layerFits(rs, gap, form, commitHint) {
		rs = append(rs, gap...)
	}
	rs = append(rs, form...)
	if len(commitHint) > 0 && m.layerFits(rs, gap, commitHint) {
		rs = append(rs, gap...)
	}
	rs = append(rs, commitHint...)
	return rs
}
