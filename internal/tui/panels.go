package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func (m *model) reviewPages() [][]string {
	if m.plan == nil {
		return [][]string{{"No plan"}}
	}
	p := m.plan
	lines := []string{"Client: " + p.ClientID, "Plan: " + p.ID, "Revision: " + p.Revision, fmt.Sprintf("Manage: %t", p.Desired.Enabled), "Exact model: " + p.Desired.Profile.Model, fmt.Sprintf("Budget: %d | Compact: %d", p.Desired.Profile.Budget, p.Desired.Profile.Compact), "Default tier: " + tierLabel(p.Desired.Profile.Tier), "Available tiers: " + tierAvailability(p.Desired.Profile.TiersConfirmed)}
	for _, c := range p.Changes {
		lines = append(lines, "Field: "+c.Field, "  Before: "+c.Before, "  After: "+c.After)
	}
	if p.Noop {
		lines = append(lines, "No changes; Apply is unavailable")
	}
	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, strings.Split(ansi.Wrap(m.public(line), max(1, m.width), ""), "\n")...)
	}
	count := max(1, m.height-11)
	var pages [][]string
	for i := 0; i < len(wrapped); i += count {
		pages = append(pages, wrapped[i:min(i+count, len(wrapped))])
	}
	return pages
}

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
		text(fmt.Sprintf("Shortcut help | Page %d of %d", page+1, len(pages)))
		for _, line := range pages[page] {
			text(line)
		}
		rs = append(rs, row{text: "> Close help (Enter / Esc)", focus: true})
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

type coreReceipt struct{ text string }

func receiptRows(rs []core.Receipt) []coreReceipt {
	out := make([]coreReceipt, 0, len(rs))
	for _, r := range rs {
		out = append(out, coreReceipt{fmt.Sprintf("Receipt %s | %s | %s | %s", r.ID, r.ClientID, r.Outcome, r.Summary)})
	}
	return out
}
