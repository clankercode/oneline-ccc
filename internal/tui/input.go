package tui

import (
	"context"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m *model) key(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	if key == "ctrl+c" {
		m.editor = nil
		if m.modal != "" {
			m.closeModal()
		}
		return m.quitRequest()
	}
	if m.editor != nil {
		return m.editKey(k)
	}
	if m.modal != "" {
		return m.modalKey(k)
	}
	switch key {
	case "?":
		m.openModal("help")
	case "ctrl+p":
		m.openModal("palette")
	case "/":
		m.startEditor("filter", m.filter)
	case "r":
		return m.inspect()
	case "q":
		return m.quitRequest()
	case "esc":
		m.back()
	case "f6":
		m.moveRegion(1)
	case "shift+f6":
		m.moveRegion(-1)
	case "[":
		m.switchSection(-1)
	case "]":
		m.switchSection(1)
	case "left":
		m.moveLocal(-1, true)
	case "right":
		m.moveLocal(1, true)
	case "tab", "shift+tab":
		n := len(m.controls())
		step := 1
		if key == "shift+tab" {
			step = -1
		}
		m.focus = (m.focus + step + n) % n
		m.scroll = -1
	case "down", "up":
		step := 1
		if key == "up" {
			step = -1
		}
		m.moveLocal(step, false)
	case "pgdown":
		_, _, start, available := m.layout()
		m.scroll = start + max(1, available-1)
	case "pgup":
		_, _, start, available := m.layout()
		m.scroll = max(0, start-max(1, available-1))
	case "enter":
		return m.activate(m.focused())
	case " ", "space":
		c := m.focused()
		if strings.HasPrefix(c.id, "client:") {
			m.selected = strings.TrimPrefix(c.id, "client:")
			return m.activate(control{id: "enabled"})
		}
		if c.id == "enabled" || c.id == "capacity" || c.id == "tiers" || c.id == "ascii" || c.id == "color" || c.id == "motion" {
			return m.activate(c)
		}
	}
	return nil
}
func (m *model) selectFocused() {
	if id, ok := strings.CutPrefix(m.focused().id, "client:"); ok {
		m.selected = id
	}
}
func (m *model) activate(c control) tea.Cmd {
	if c.reason != "" {
		m.status = c.reason
		return nil
	}
	id := c.id
	if id == "nav:More" {
		m.openModal("more")
		return nil
	}
	if strings.HasPrefix(id, "nav:") {
		m.goTo(strings.TrimPrefix(id, "nav:"))
		if m.focus < navCount {
			m.focus = navCount
			m.scroll = -1
		}
		return nil
	}
	if strings.HasPrefix(id, "client:") {
		m.selected = strings.TrimPrefix(id, "client:")
		m.goTo("Models")
		return nil
	}
	if m.busy != "" && id != "help" && id != "palette" && id != "back" && id != "cancel" {
		m.status = "Working: " + m.busy + "; wait or request cancellation"
		return nil
	}
	p := m.draft().Profile
	switch {
	case strings.HasPrefix(id, "discovered:"):
		i, err := strconv.Atoi(strings.TrimPrefix(id, "discovered:"))
		if err == nil && i >= 0 && i < len(m.models) {
			p.Model = m.models[i].ID
			m.changeProfile(p)
		}
	case strings.HasPrefix(id, "model:"):
		p.Model = strings.TrimPrefix(id, "model:")
		m.changeProfile(p)
	case strings.HasPrefix(id, "budget:"):
		p.Budget, _ = strconv.ParseInt(strings.TrimPrefix(id, "budget:"), 10, 64)
		p.Compact = compactBudget(p.Budget)
		m.changeProfile(p)
	case strings.HasPrefix(id, "tier:"):
		p.Tier = strings.TrimPrefix(id, "tier:")
		m.changeProfile(p)
	case strings.HasPrefix(id, "advanced:"):
		if why := m.editable(); why != "" {
			m.status = why
			return nil
		}
		field := strings.TrimPrefix(id, "advanced:")
		value := ""
		switch field {
		case "model":
			value = p.Model
		case "budget":
			value = strconv.FormatInt(p.Budget, 10)
		case "compact":
			value = strconv.FormatInt(p.Compact, 10)
		case "endpoint":
			value = p.BaseURL
		case "credential":
			value = p.EnvKey
		}
		m.startEditor(field, value)
	case strings.HasPrefix(id, "service:"):
		a := strings.TrimPrefix(id, "service:")
		if a == "install" || a == "uninstall" {
			m.openModal("service:" + a)
		} else {
			return m.service(a)
		}
	default:
		switch id {
		case "configure":
			m.goTo("Models")
		case "enabled":
			if why := m.editable(); why != "" {
				m.status = why
				return nil
			}
			d := m.draft()
			d.Enabled = !d.Enabled
			m.drafts[m.selected] = d
			m.plan = nil
		case "capacity":
			p.CapacityConfirmed = !p.CapacityConfirmed
			d := m.draft()
			d.Profile = p
			m.drafts[m.selected] = d
			m.plan = nil
		case "tiers":
			p.TiersConfirmed = !p.TiersConfirmed
			d := m.draft()
			d.Profile = p
			m.drafts[m.selected] = d
			m.plan = nil
		case "plan":
			return m.makePlan()
		case "review:apply":
			return m.apply()
		case "review:read":
			pages := m.reviewPages()
			if m.reviewPage+1 < len(pages) {
				m.reviewPage++
			} else {
				m.reviewRead = true
				m.status = "Complete review acknowledged; Apply is now available"
			}
		case "review:back":
			m.back()
		case "back":
			m.back()
		case "filter:clear":
			m.filter = ""
		case "help":
			m.openModal("help")
		case "palette":
			m.openModal("palette")
		case "discard":
			m.openModal("discard")
		case "cancel":
			if m.cancel != nil {
				m.cancel()
				m.status = "Cancellation requested; waiting for backend outcome before leaving"
			}
		case "launch":
			return m.command("launch")
		case "ssh":
			m.openModal("ssh")
		case "key":
			m.persistent = false
			m.startEditor("key", "")
		case "discover":
			client := m.selected
			return m.begin("discover", func(ctx context.Context) (any, error) { return m.backend.Discover(ctx, client) })
		case "repair":
			return m.begin("repair", func(ctx context.Context) (any, error) { return m.backend.RepairBrief(ctx) })
		case "repair:export":
			return m.exportBrief()
		case "observer:off":
			m.observerDismissed = true
			m.status = "Monitor remains optional; foreground setup is available"
		case "theme":
			switch m.options.Theme {
			case "dark":
				m.options.Theme = "light"
			case "light":
				m.options.Theme = "mono"
			default:
				m.options.Theme = "dark"
			}
		case "ascii":
			m.options.ASCII = !m.options.ASCII
		case "color":
			m.options.NoColor = !m.options.NoColor
		case "motion":
			m.options.ReducedMotion = !m.options.ReducedMotion
		}
	}
	return nil
}
func (m *model) quitRequest() tea.Cmd {
	if m.dirty() || m.busy != "" {
		m.openModal("quit")
		return nil
	}
	if m.cancel != nil {
		m.cancel()
	}
	return tea.Quit
}
