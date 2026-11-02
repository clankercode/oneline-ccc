package tui

import "strings"

var sections = []string{"Dashboard", "Models", "Review", "Jobs", "Gateway", "Service", "Repair", "SSH", "Utilities", "Settings"}

const navCount = 5 // Four primary tabs and the visible More menu.

type location struct {
	id            string
	focus, scroll int
}

// A Back frame belongs to one visit. The per-screen cache below serves normal
// revisits and must never overwrite an earlier visit still on the Back stack.
type navigationFrame struct {
	screen       string
	location     location
	modelsClient string
	modelsForm   modelsForm
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
	if m.screen == "Models" {
		f := m.modelsForm()
		f.location, f.visited = m.location(), true
		m.setModelsForm(f)
	}
}
func (m *model) enter(screen string) {
	m.screen = screen
	m.editor = nil
	if screen == "Models" {
		f := m.modelsForm()
		if f.visited {
			m.restore(f.location)
		} else {
			m.focus, m.scroll = navCount, -1
		}
		return
	}
	if p, ok := m.locations[screen]; ok {
		m.restore(p)
	} else {
		m.focus, m.scroll = navCount, -1
		// First entry offers useful work. Revisited controls retain their exact
		// location, including unavailable entries the operator chose to inspect.
		for i, c := range m.controls()[navCount:] {
			if c.reason == "" {
				m.focus = navCount + i
				break
			}
		}
	}
}
func (m *model) goTo(screen string) {
	if screen == m.screen {
		return
	}
	m.remember()
	frame := navigationFrame{screen: m.screen, location: m.locations[m.screen]}
	if m.screen == "Models" {
		frame.modelsClient, frame.modelsForm = m.selected, m.modelsForm()
	}
	m.history = append(m.history, frame)
	m.enter(screen)
}
func (m *model) back() {
	if m.screen == "Dashboard" && len(m.history) == 0 {
		return
	}
	m.remember()
	target := "Dashboard"
	if n := len(m.history); n > 0 {
		frame := m.history[n-1]
		m.history = m.history[:n-1]
		m.screen, m.editor = frame.screen, nil
		if frame.screen == "Models" {
			m.selected = frame.modelsClient
			m.setModelsForm(frame.modelsForm)
		}
		m.restore(frame.location)
		return
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
	if strings.HasPrefix(c.id, "field:") || c.id == "models:advanced" || c.id == "enabled" || c.id == "capacity" || c.id == "tiers" || c.id == "theme" || c.id == "ascii" || c.id == "color" || c.id == "motion" || strings.HasPrefix(c.id, "model:") || strings.HasPrefix(c.id, "discovered:") || strings.HasPrefix(c.id, "budget:") || strings.HasPrefix(c.id, "tier:") || strings.HasPrefix(c.id, "advanced:") {
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
		if m.screen == "Models" && !m.modelsVisible(c) {
			continue
		}
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
		if m.screen == "Models" && target == "Form" && modelsLeafPanel(id) == "advanced" {
			f := m.modelsForm()
			f.advanced = true
			m.setModelsForm(f)
		}
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
			if m.screen == "Models" && !m.modelsVisible(c) {
				continue
			}
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
		// Missing anchors stay in the destination region at its first control.
		m.focus = first
		m.scroll = -1
		return
	}
}
