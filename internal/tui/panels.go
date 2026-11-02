package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m *model) modalRows() []row {
	var rs []row
	text := func(s string) { rs = append(rs, row{text: m.public(s)}) }
	switch m.modal {
	case "more":
		text("More destinations | Enter opens section")
		for i, name := range sections[4:] {
			prefix := "  "
			if i == m.modalFocus {
				prefix = "> "
			}
			rs = append(rs, row{text: prefix + name, focus: i == m.modalFocus})
		}
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
		text("Search: " + m.query)
		cs := m.paletteCommands()
		if len(cs) == 0 {
			text("No commands match. Backspace edits; Esc closes.")
			return rs
		}
		selected := min(m.modalFocus, len(cs)-1)
		visible := max(1, m.height-7)
		start := max(0, selected-visible+1)
		for i := start; i < min(len(cs), start+visible); i++ {
			c := cs[i]
			label := c.label
			if c.reason != "" {
				label += " [Unavailable: " + c.reason + "]"
			}
			prefix := "  "
			if i == selected {
				prefix = "> "
			}
			rs = append(rs, row{text: prefix + m.public(label), focus: i == selected})
		}
	default:
		switch {
		case m.modal == "quit":
			text("Leave with unapplied drafts?")
			if m.busy != "" {
				text("Running: " + m.busy)
				text("Request cancellation and wait for its final outcome.")
			}
		case m.modal == "discard":
			text("Discard only " + m.selected + " draft? Other drafts stay.")
		case m.modal == "ssh":
			text("Hand terminal control to the backend SSH command?")
		case strings.HasPrefix(m.modal, "service:"):
			text("Confirm monitor " + strings.TrimPrefix(m.modal, "service:") + "?")
			text("Client configuration and credentials are separate.")
		}
		yes := "Confirm"
		if m.modal == "quit" && m.busy != "" {
			yes = "Request cancel; stay until outcome"
		}
		for i, label := range []string{"Back; keep working", yes} {
			prefix := "  "
			if i == m.modalFocus {
				prefix = "> "
			}
			rs = append(rs, row{text: prefix + label, focus: i == m.modalFocus})
		}
	}
	return rs
}
func (m *model) editorRows() []row {
	e := m.editor
	rs := []row{{text: "Editing " + e.field + " | Esc keeps the previous draft"}}
	if e.field == "endpoint" {
		rs = append(rs, row{text: "Local-only endpoint. Public views use Gateway A."})
	}
	if e.field == "key" {
		labels := []string{"API key: [hidden]", checkbox(m.persistent) + " Persist using backend credential storage", "Save key", "Back; drop unsaved key"}
		for i, label := range labels {
			prefix := "  "
			if i == e.focus {
				prefix = "> "
			}
			rs = append(rs, row{text: prefix + label, focus: i == e.focus})
		}
		return rs
	}
	before := clean(string(e.value[:e.pos]))
	after := clean(string(e.value[e.pos:]))
	space := max(0, m.width-3)
	if ansi.StringWidth(before) > space {
		whole := before
		for cut := ansi.StringWidth(whole) - space; ansi.StringWidth(before) > space; cut++ {
			before = ansi.TruncateLeft(whole, cut, "")
		}
	}
	after = ansi.Truncate(after, max(0, space-ansi.StringWidth(before)), "")
	rs = append(rs, row{text: "> " + before + "|" + after, focus: true}, row{text: "Enter commits this field to the draft only."})
	return rs
}
