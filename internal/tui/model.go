// Package tui implements the foreground operator console. All external work is
// performed by core.Backend; views and navigation never write configuration.
package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
)

type Options struct {
	Theme                         string
	ASCII, NoColor, ReducedMotion bool
	Version                       string
}

type model struct {
	backend                      core.Backend
	options                      Options
	snapshot                     core.Snapshot
	drafts                       map[string]core.Desired
	baselines                    map[string]core.Desired
	receipts                     []core.Receipt
	selected, screen, previous   string
	focus, width, height, scroll int
	modal                        string
	savedFocus, modalFocus       int
	savedControlID               string
	query, filter                string
	editor                       *editor
	busy, status                 string
	cancel                       context.CancelFunc
	token                        uint64
	initial                      tea.Cmd
	plan                         *core.Plan
	requested                    core.Desired
	requestedID                  string
	reconcileClient              string
	reviewPage                   int
	reviewRead                   bool
	models                       []core.ModelInfo
	repair                       string
	persistent                   bool
	observerDismissed            bool
}

func New(backend core.Backend, options Options) tea.Model {
	m := &model{backend: backend, options: options, drafts: make(map[string]core.Desired), baselines: make(map[string]core.Desired), screen: "Dashboard", width: 80, height: 24, scroll: -1, status: "Loading local inspection"}
	m.initial = m.begin("inspect", func(ctx context.Context) (any, error) { return backend.Inspect(ctx) })
	return m
}
func (m *model) Init() tea.Cmd { return m.initial }
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		if m.plan != nil && (v.Width != m.width || v.Height != m.height) {
			m.reviewPage = 0
			m.reviewRead = false
			m.status = "Review resized; read the reflowed pages before Apply"
		}
		m.width = max(1, v.Width)
		m.height = max(1, v.Height)
		m.scroll = -1
	case result:
		return m, m.complete(v)
	case exportResult:
		if v.err != nil {
			m.status = "Brief export failed"
		} else {
			m.status = "Redacted brief exported to terminal scrollback"
		}
	case tea.PasteMsg:
		if m.editor != nil {
			if m.editor.field != "key" || m.editor.focus == 0 {
				m.editor.insert(v.Content)
			}
		} else if m.modal == "palette" {
			m.query += clean(v.Content)
			m.modalFocus = 0
		}
	case tea.KeyPressMsg:
		return m, m.key(v)
	}
	return m, nil
}
func (m *model) client() (core.Client, bool) {
	for _, c := range m.snapshot.Clients {
		if c.ID == m.selected {
			return c, true
		}
	}
	return core.Client{}, false
}
func (m *model) draft() core.Desired { return m.drafts[m.selected] }
func (m *model) dirty() bool {
	for id, d := range m.drafts {
		if d != m.baselines[id] {
			return true
		}
	}
	return false
}
func (m *model) editable() string {
	c, ok := m.client()
	if !ok {
		return "Select a client first"
	}
	if c.ID != "codex" {
		return "Read only: this native build verifies only the Codex configuration path"
	}
	if !c.Installed {
		return "Not installed; installation is unavailable in this build"
	}
	if !c.Supported {
		return "Read only: " + c.Reason
	}
	return ""
}
func (m *model) changeProfile(p core.Profile) {
	d := m.draft()
	old := d.Profile
	if old.BaseURL != p.BaseURL || old.Model != p.Model || old.Budget != p.Budget || old.Compact != p.Compact || old.Tier != p.Tier {
		p.CapacityConfirmed = false
		p.TiersConfirmed = false
	}
	d.Profile = p
	m.drafts[m.selected] = d
	m.plan = nil
}
func (m *model) goTo(screen string) { m.screen = screen; m.focus = 4; m.scroll = -1; m.editor = nil }
func (m *model) openModal(name string) {
	m.savedFocus = m.focus
	m.savedControlID = m.focused().id
	m.modal = name
	m.modalFocus = 0
	m.query = ""
}
func (m *model) closeModal() {
	m.modal = ""
	m.focus = max(0, min(m.savedFocus, len(m.controls())-1))
	m.focusID(m.savedControlID)
	m.modalFocus = 0
	m.query = ""
}
func (m *model) focused() control {
	cs := m.controls()
	if len(cs) == 0 {
		return control{}
	}
	m.focus = max(0, min(m.focus, len(cs)-1))
	return cs[m.focus]
}
func (m *model) focusID(id string) {
	for i, c := range m.controls() {
		if c.id == id {
			m.focus = i
			return
		}
	}
}

type result struct {
	token uint64
	kind  string
	value any
	err   error
}

// Every backend request has a deadline and a cancellation handle. Interactive
// children receive a longer bound so normal sessions are not cut off early.
func (m *model) begin(kind string, work func(context.Context) (any, error)) tea.Cmd {
	if m.busy != "" {
		m.status = "Wait for the current action or request cancellation"
		return nil
	}
	timeout := 20 * time.Second
	if kind == "launch" || kind == "ssh" {
		timeout = 24 * time.Hour
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	m.cancel = cancel
	m.token++
	token := m.token
	m.busy = kind
	m.status = "Working: " + kind
	return func() tea.Msg { value, err := work(ctx); return result{token, kind, value, err} }
}
func (m *model) inspect() tea.Cmd {
	return m.begin("inspect", func(ctx context.Context) (any, error) { return m.backend.Inspect(ctx) })
}
