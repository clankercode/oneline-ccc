package tui

import "strings"

var sections = []string{"Dashboard", "Models", "Review", "Jobs", "Gateway", "Service", "Repair", "SSH", "Utilities", "Settings"}

const navCount = 5 // Four primary tabs and the visible More menu.

type location struct {
	id            string
	focus, scroll int
}

func (m *model) location() location { return location{m.focused().id, m.focus, m.scroll} }
func (m *model) restore(p location) {
	m.focus = max(0, min(p.focus, len(m.controls())-1))
	m.focusID(p.id)
	m.scroll = p.scroll
}
func (m *model) remember() {
	if m.locations == nil {
		m.locations = make(map[string]location)
	}
	m.locations[m.screen] = m.location()
}
func (m *model) enter(screen string) {
	m.screen = screen
	m.editor = nil
	if p, ok := m.locations[screen]; ok {
		m.restore(p)
	} else {
		m.focus, m.scroll = navCount, -1
	}
}
func (m *model) goTo(screen string) {
	if screen == m.screen {
		return
	}
	m.remember()
	m.history = append(m.history, m.screen)
	m.enter(screen)
}
func (m *model) back() {
	if m.screen == "Dashboard" && len(m.history) == 0 {
		return
	}
	m.remember()
	target := "Dashboard"
	if n := len(m.history); n > 0 {
		target = m.history[n-1]
		m.history = m.history[:n-1]
	} else if m.screen == "Review" {
		target = "Models"
	}
	m.enter(target)
}
func (m *model) switchSection(step int) {
	for i, name := range sections {
		if name == m.screen {
			m.goTo(sections[(i+step+len(sections))%len(sections)])
			return
		}
	}
}

// Tab visits every control. Arrows stay in a local region; F6 crosses regions.
// Determine regions from stable control IDs, never dynamic list lengths.
func (m *model) region(i int, c control) string {
	if i < navCount {
		return "Nav"
	}
	if c.id == "help" || c.id == "palette" {
		return "Help"
	}
	if strings.HasPrefix(c.id, "client:") || c.id == "filter:clear" {
		return "Clients"
	}
	if c.id == "enabled" || c.id == "capacity" || c.id == "tiers" || c.id == "theme" || c.id == "ascii" || c.id == "color" || c.id == "motion" || strings.HasPrefix(c.id, "model:") || strings.HasPrefix(c.id, "discovered:") || strings.HasPrefix(c.id, "budget:") || strings.HasPrefix(c.id, "tier:") || strings.HasPrefix(c.id, "advanced:") {
		return "Form"
	}
	return "Actions"
}
func (m *model) moveLocal(step int, horizontal bool) {
	cs := m.controls()
	current := m.region(m.focus, m.focused())
	if horizontal != (current == "Nav") {
		return
	}
	for i := m.focus + step; i >= 0 && i < len(cs); i += step {
		if m.region(i, cs[i]) == current {
			m.focus = i
			m.scroll = -1
			if !horizontal {
				m.selectFocused()
			}
			return
		}
	}
}
func (m *model) moveRegion(step int) {
	cs := m.controls()
	current := m.region(m.focus, m.focused())
	if m.regionFocus == nil {
		m.regionFocus = make(map[string]string)
	}
	m.regionFocus[m.screen+current] = m.focused().id
	var groups []string
	for i, c := range cs {
		r := m.region(i, c)
		if len(groups) == 0 || groups[len(groups)-1] != r {
			groups = append(groups, r)
		}
	}
	for i, r := range groups {
		if r != current {
			continue
		}
		target := groups[(i+step+len(groups))%len(groups)]
		id := m.regionFocus[m.screen+target]
		if id == "" && target == "Nav" {
			id = "nav:" + m.screen
			for j, name := range sections {
				if name == m.screen && j >= 4 {
					id = "nav:More"
				}
			}
		}
		first := -1
		for j, c := range cs {
			if m.region(j, c) != target {
				continue
			}
			if first < 0 {
				first = j
			}
			if c.id == id || (id == "" && c.id == "client:"+m.selected) {
				first = j
				break
			}
		}
		m.focus = first
		m.scroll = -1
		return
	}
}
