package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type command struct{ id, label, key, reason string }

// Help and command search are projections of this single catalog. Apply is
// deliberately absent: it is only reachable inside the exact plan review.
func (m *model) catalog() []command {
	cs := []command{}
	for _, name := range sections {
		cs = append(cs, command{"nav:" + name, name, "", ""})
	}
	return append(cs, []command{
		{"help", "Shortcut help", "?", ""}, {"palette", "Command search", "Ctrl+P", ""}, {"filter", "Filter clients", "/", ""}, {"refresh", "Refresh inspection; keep drafts", "r", ""},
		{"configure", "Configure selected client", "Enter", noClient(m)}, {"enabled", "Stage management checkbox", "Space", m.editable()},
		{"plan", "Review selected client draft", "", m.editable()}, {"launch", "Launch selected client", "", m.editable()},
		{"cancel", "Request cancellation (Jobs)", "", idleReason(m)}, {"quit", "Safe exit", "q / Ctrl+C", ""},
		{"move", "Move focus", "Tab / Shift+Tab", ""}, {"select", "Move within current list / form / actions", "Up / Down", ""}, {"tabs", "Move tab focus; Enter opens content", "Left / Right", ""}, {"regions", "Switch focus region", "F6 / Shift+F6", ""}, {"sections", "Switch section (outside text)", "[ / ]", ""}, {"nav:More", "More destinations menu", "", ""}, {"back", "Close top layer / keep drafts", "Esc", ""},
		{"page", "Scroll long content", "PgUp / PgDown", ""},
	}...)
}
func (m *model) paletteCommands() []command {
	var cs []command
	for _, c := range m.catalog() {
		if strings.Contains(strings.ToLower(c.label), strings.ToLower(m.query)) {
			cs = append(cs, c)
		}
	}
	return cs
}
func (m *model) modalKey(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	if key == "esc" {
		m.closeModal()
		return nil
	}
	switch m.modal {
	case "more":
		switch key {
		case "up", "shift+tab":
			m.modalFocus = (m.modalFocus + 5) % 6
		case "down", "tab":
			m.modalFocus = (m.modalFocus + 1) % 6
		case "enter":
			target := sections[4+m.modalFocus]
			m.closeModal()
			m.goTo(target)
		}
	case "help":
		_, last := m.helpPagination()
		m.modalFocus = max(0, min(m.modalFocus, last))
		if key == "enter" || key == "?" {
			m.closeModal()
		} else if key == "pgdown" || key == "down" || key == "tab" {
			m.modalFocus = min(last, m.modalFocus+1)
		} else if key == "pgup" || key == "up" || key == "shift+tab" {
			m.modalFocus = max(0, m.modalFocus-1)
		}
	case "palette":
		cs := m.paletteCommands()
		switch key {
		case "up":
			m.modalFocus = max(0, m.modalFocus-1)
		case "down", "tab":
			m.modalFocus = min(max(0, len(cs)-1), m.modalFocus+1)
		case "shift+tab":
			m.modalFocus = max(0, m.modalFocus-1)
		case "enter":
			if len(cs) == 0 {
				return nil
			}
			c := cs[min(m.modalFocus, len(cs)-1)]
			if c.reason != "" {
				m.status = c.reason
				return nil
			}
			m.closeModal()
			switch c.id {
			case "refresh":
				return m.inspect()
			case "filter":
				m.startEditor("filter", m.filter)
			case "quit":
				return m.quitRequest()
			case "move", "select", "page", "tabs", "regions", "sections":
				m.openModal("help")
			default:
				return m.activate(control{id: c.id})
			}
		case "backspace":
			rs := []rune(m.query)
			if len(rs) > 0 {
				m.query = string(rs[:len(rs)-1])
			}
			m.modalFocus = 0
		default:
			if k.Text != "" {
				m.query += clean(k.Text)
				m.modalFocus = 0
			}
		}
	default:
		if key == "tab" || key == "shift+tab" || key == "left" || key == "right" || key == "up" || key == "down" {
			m.modalFocus = 1 - m.modalFocus
			return nil
		}
		if key != "enter" {
			return nil
		}
		if m.modalFocus == 0 {
			m.closeModal()
			return nil
		}
		name := m.modal
		m.closeModal()
		switch {
		case name == "quit":
			if m.busy != "" {
				if m.cancel != nil {
					m.cancel()
				}
				m.status = "Cancellation requested; wait for the backend's final outcome, then quit again"
				return nil
			}
			return tea.Quit
		case name == "discard":
			if c, ok := m.client(); ok {
				m.drafts[c.ID] = desiredClient(c)
				m.plan = nil
				m.status = "Discarded this client's draft"
			}
		case name == "ssh":
			return m.command("ssh")
		case strings.HasPrefix(name, "service:"):
			return m.service(strings.TrimPrefix(name, "service:"))
		}
	}
	return nil
}
