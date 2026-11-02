package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type helpBinding struct{ key, description string }
type helpGroup struct {
	title    string
	bindings []helpBinding
}

// Shortcut keys come from the same catalog as Commands. Help explains how to
// use them; unkeyed destinations remain in Commands and the visible More menu.
func (m *model) helpGroups() []helpGroup {
	catalog := make(map[string]command)
	for _, c := range m.catalog() {
		catalog[c.id] = c
	}
	binding := func(id, description string) helpBinding {
		c := catalog[id]
		if description == "" {
			description = c.label
		}
		return helpBinding{c.key, description}
	}
	groups := []helpGroup{
		{"MOVE & ACT", []helpBinding{
			binding("move", ""),
			binding("configure", "Activate focused control"),
			binding("enabled", "Toggle checkboxes only"),
			binding("select", "Move within list / form / actions"),
			binding("regions", ""),
			binding("page", ""),
			binding("back", ""),
		}},
		{"SECTIONS & COMMANDS", []helpBinding{
			binding("tabs", ""),
			binding("sections", ""),
			{"", "More tab: Enter opens secondary destinations."},
			binding("palette", ""), binding("filter", ""),
			binding("refresh", ""), binding("help", ""), binding("quit", ""),
		}},
		{"CURRENT CONTROL", m.helpControl()},
		{"TEXT ENTRY", []helpBinding{
			{"", "Type/paste stays in the editor; letters/Space are text. Ctrl+C still requests safe exit."},
			{"", "Left/Right moves the caret. Enter saves the field; Esc keeps the previous draft."},
			{"", "Key input: Enter focuses Save. Tab/Shift+Tab visits options; only input accepts text."},
			{"", "Persistence: Space/Enter toggles. Save stores the key; Back drops unsaved text."},
		}},
		{"HELP PAGES", []helpBinding{
			{"", "Down / Tab / PgDown: next page. Up / Shift+Tab / PgUp: previous page."},
			{"", "Stops at first/last; reverse works. Resize reflows. Close restores focus and keeps drafts."},
		}},
	}
	return groups
}

func (m *model) helpControl() []helpBinding {
	c := m.focused()
	label := m.public(c.label)
	if c.id == "nav:More" {
		label = "More destinations"
	}
	activation := "Enter activates this control."
	switch {
	case c.reason != "":
		activation = "Unavailable: " + m.public(c.reason)
	case strings.HasPrefix(c.id, "client:"):
		activation = "Enter opens client settings; Space stages management."
		target := *m
		target.selected = strings.TrimPrefix(c.id, "client:")
		if why := target.editable(); why != "" {
			activation = "Enter opens client settings. Space unavailable: " + why
		} else if m.busy != "" {
			activation = "Enter opens client settings. Space unavailable while working."
		}
	case strings.HasPrefix(c.id, "nav:"):
		activation = "Enter opens " + label + "."
		if m.focus < navCount {
			activation += " Left/Right moves tab focus."
		}
	case m.busy != "" && c.id != "help" && c.id != "palette" && c.id != "back" && c.id != "cancel":
		activation = "Wait for the backend's final outcome; activation is unavailable while working."
	case c.id == "enabled", c.id == "capacity", c.id == "tiers", c.id == "ascii", c.id == "color", c.id == "motion":
		activation = "Space toggles this control; Enter also activates it."
	}
	return []helpBinding{
		{"", m.screen + " / " + m.region(m.focus, c) + ": " + label},
		{"", activation},
	}
}

// Wrap sanitized text before styling. Hanging descriptions retain their key
// column on wide terminals, while compact rows leave room for useful content.
func (m *model) helpBindingRows(b helpBinding, width int) []string {
	if b.key == "" {
		return strings.Split(ansi.Wrap(m.public(b.description), width, ""), "\n")
	}
	key := m.public(b.key)
	prefix := key + " : "
	if width >= 56 {
		prefix = key + strings.Repeat(" ", max(1, 20-ansi.StringWidth(key))) + ": "
	}
	indent := ansi.StringWidth(prefix)
	if width-indent < 12 {
		return strings.Split(ansi.Wrap(m.heading(key)+" : "+m.public(b.description), width, ""), "\n")
	}
	lines := strings.Split(ansi.Wrap(m.public(b.description), width-indent, ""), "\n")
	lines[0] = m.heading(key) + strings.TrimPrefix(prefix, key) + lines[0]
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat(" ", indent) + lines[i]
	}
	return lines
}

const helpClose = "> Close help (Enter / Esc / ?)"

// Help shares its rendered page boundaries with input and resize handling.
// Count wrapped terminal rows after reserving the actual shell and page chrome.
func (m *model) helpPagination() (size, last int) {
	pages, size := m.helpPages()
	return size, len(pages) - 1
}

func (m *model) helpPages() ([][]string, int) {
	width := m.bodyWidth()
	wrap := func(text string) []string {
		return strings.Split(ansi.Wrap(m.public(text), width, ""), "\n")
	}
	type group struct {
		title   string
		entries [][]string
	}
	var groups []group
	total := 0
	for _, g := range m.helpGroups() {
		rendered := group{title: g.title}
		for _, b := range g.bindings {
			lines := m.helpBindingRows(b, width)
			rendered.entries = append(rendered.entries, lines)
			total += len(lines)
		}
		groups = append(groups, rendered)
	}
	// One row per page is the maximum possible page count; use its digit width
	// to reserve a stable heading even when a narrower terminal adds pages.
	title := fmt.Sprintf("Shortcut help | Page %d of %d", max(1, total), max(1, total))
	available := m.height - len(m.shellHeader()) - len(m.shellFooter(""))
	capacity := max(1, available-len(wrap(title))-len(wrap(helpClose)))
	pages := [][]string{{}}
	for _, g := range groups {
		heading := func(continued bool) []string {
			label := "-- " + g.title + " --"
			if continued {
				label = "-- " + g.title + " (cont.) --"
			}
			lines := wrap(label)
			for i, line := range lines {
				if !m.options.NoColor {
					line = m.surface(surfaceRaised).Bold(true).Render(line)
				}
				lines[i] = line
			}
			return lines
		}
		pending, continued := true, false
		for _, entry := range g.entries {
			head := heading(continued)
			last := len(pages) - 1
			need := len(entry)
			if pending {
				need += len(head)
			}
			// Entries that fit below a heading move intact to the next page.
			// Oversized prose continues with a repeated group heading instead
			// of dropping a remainder or leaving a heading alone at a boundary.
			if len(pages[last]) > 0 && len(pages[last])+need > capacity && (pending || len(entry) <= capacity-len(head)) {
				pages = append(pages, nil)
				last++
				pending = true
			}
			if pending {
				if len(pages[last]) > 0 && len(pages[last])+len(head)+len(entry)+1 <= capacity {
					pages[last] = append(pages[last], "")
				}
				// Below the supported 40x12 viewport, favor content over chrome.
				if len(head) < capacity {
					pages[last] = append(pages[last], head...)
				}
				pending, continued = false, true
			}
			for _, line := range entry {
				if len(pages[last]) == capacity {
					pages = append(pages, nil)
					last++
					head = heading(true)
					if len(head) < capacity {
						pages[last] = append(pages[last], head...)
					}
				}
				pages[last] = append(pages[last], line)
			}
		}
	}
	return pages, capacity
}
