package tui

import (
	"context"
	"strconv"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
)

type editor struct {
	field      string
	value      []rune
	pos, focus int
}

func (e *editor) insert(s string) {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	rs := []rune(s)
	if len(e.value)+len(rs) > 4096 {
		return
	}
	tail := append([]rune(nil), e.value[e.pos:]...)
	e.value = append(e.value[:e.pos], rs...)
	e.value = append(e.value, tail...)
	e.pos += len(rs)
}
func (m *model) startEditor(field, value string) {
	m.editor = &editor{field: field, value: []rune(value), pos: len([]rune(value))}
}
func (m *model) editKey(k tea.KeyPressMsg) tea.Cmd {
	e := m.editor
	key := k.String()
	switch key {
	case "esc":
		m.editor = nil
		return nil
	case "tab", "shift+tab":
		if e.field == "key" {
			step := 1
			if key == "shift+tab" {
				step = -1
			}
			e.focus = (e.focus + step + 4) % 4
		}
		return nil
	case "enter":
		if e.field == "key" {
			switch e.focus {
			case 0:
				e.focus = 2
			case 1:
				m.persistent = !m.persistent
			case 2:
				return m.saveEditor()
			case 3:
				m.editor = nil
			}
			return nil
		}
		return m.saveEditor()
	}
	if e.field == "key" && e.focus != 0 {
		if (key == " " || key == "space") && e.focus == 1 {
			m.persistent = !m.persistent
		}
		return nil
	}
	switch key {
	case "left":
		e.pos = max(0, e.pos-1)
	case "right":
		e.pos = min(len(e.value), e.pos+1)
	case "home", "ctrl+a":
		e.pos = 0
	case "end", "ctrl+e":
		e.pos = len(e.value)
	case "backspace":
		if e.pos > 0 {
			e.value = append(e.value[:e.pos-1], e.value[e.pos:]...)
			e.pos--
		}
	case "delete":
		if e.pos < len(e.value) {
			e.value = append(e.value[:e.pos], e.value[e.pos+1:]...)
		}
	case "ctrl+u":
		e.value = nil
		e.pos = 0
	default:
		if k.Text != "" {
			e.insert(k.Text)
		}
	}
	return nil
}
func (m *model) saveEditor() tea.Cmd {
	e := m.editor
	value := string(e.value)
	if e.field == "filter" {
		m.filter = value
		m.editor = nil
		m.focus = 4
		return nil
	}
	if e.field == "key" {
		if value == "" {
			m.status = "Enter a credential or choose Back"
			return nil
		}
		persistent := m.persistent
		m.editor = nil
		return m.begin("key", func(ctx context.Context) (any, error) {
			err := m.backend.SaveKey(ctx, value, persistent)
			value = ""
			return nil, err
		})
	}
	p := m.draft().Profile
	switch e.field {
	case "model":
		p.Model = value
	case "budget", "compact":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n <= 0 {
			m.status = "Use a positive whole token count"
			return nil
		}
		if e.field == "budget" {
			p.Budget = n
			p.Compact = compactBudget(n)
		} else {
			p.Compact = n
		}
	case "endpoint":
		if err := core.ValidateEndpoint(value); err != nil {
			m.status = err.Error()
			return nil
		}
		p.BaseURL = value
	case "credential":
		p.EnvKey = value
	}
	m.changeProfile(p)
	m.editor = nil
	m.status = "Draft updated; review before applying"
	return nil
}

// Splitting quotient and remainder avoids overflow for large custom budgets.
func compactBudget(n int64) int64 { return n/10*9 + n%10*9/10 }
