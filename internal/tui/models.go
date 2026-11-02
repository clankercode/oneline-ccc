package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

// The catalog retains semantic leaf identities for fixtures and restoration.
// Ordinary navigation visits only disclosed controls, never the whole catalog.
// Directly restored leaf focus reveals its containing choice/Advanced panel.
type modelsForm struct {
	advanced bool
	chooser  string
	origin   location
	location location
	visited  bool
}

func (m *model) modelsForm() modelsForm { return m.modelsForms[m.selected] }
func (m *model) setModelsForm(f modelsForm) {
	if m.modelsForms == nil {
		m.modelsForms = make(map[string]modelsForm)
	}
	m.modelsForms[m.selected] = f
}
func (m *model) modelsControls(why string, d core.Desired) []control {
	cs := []control{{"enabled", checkbox(d.Enabled) + " Manage client", why},
		{"field:model", "Model ID: " + d.Profile.Model + "  [Choose]", why}}
	add := func(id, label string) { cs = append(cs, control{id, label, why}) }
	known := false
	for _, id := range []string{"gpt-6.1-sol", "glm-5.3", "glm-5.3-flash"} {
		add("model:"+id, choice(d.Profile.Model == id)+" Model "+id)
		known = known || d.Profile.Model == id
	}
	if !known && d.Profile.Model != "" {
		add("model:"+d.Profile.Model, choice(true)+" Current exact ID "+d.Profile.Model)
	}
	for _, info := range m.models {
		add(discoveredControlID(info), "Advertised model "+info.ID+" | "+info.Source)
	}
	add("field:budget", fmt.Sprintf("Context budget: %d tokens  [Choose]", d.Profile.Budget))
	for _, b := range []int64{272000, 353000, 500000} {
		add(fmt.Sprintf("budget:%d", b), choice(d.Profile.Budget == b)+fmt.Sprintf(" Budget %d", b))
	}
	add("field:tier", "Default request tier: "+tierLabel(d.Profile.Tier)+"  [Choose]")
	for _, tier := range []string{"Standard", "Fast", "Ultrafast"} {
		add("tier:"+tier, choice(tierLabel(d.Profile.Tier) == tier)+" Default tier "+tier)
	}
	// Disclosure itself remains available when the client is read only.
	cs = append(cs, control{"models:advanced", "Advanced / custom settings", ""})
	add("capacity", checkbox(d.Profile.CapacityConfirmed)+" Confirm route supports this context budget")
	add("tiers", checkbox(d.Profile.TiersConfirmed)+" This route supports Fast and Ultrafast")
	for _, field := range []string{"model", "budget", "compact", "endpoint", "credential"} {
		add("advanced:"+field, "Advanced: edit "+field)
	}
	add("discover", "Discover advertised models")
	cs = append(cs, control{"plan", "Review client draft", why}, control{"discard", "Discard this client's draft", ""}, control{"back", "Back; keep draft", ""})
	if m.modelsForm().chooser != "" {
		cs = append(cs, control{"models:close", "Back to field; keep draft", ""})
	}
	return cs
}

func modelsLeafPanel(id string) string {
	switch {
	case strings.HasPrefix(id, "model:"), strings.HasPrefix(id, "discovered:"):
		return "model"
	case strings.HasPrefix(id, "budget:"):
		return "budget"
	case strings.HasPrefix(id, "tier:"):
		return "tier"
	case strings.HasPrefix(id, "advanced:"), id == "capacity", id == "tiers", id == "discover":
		return "advanced"
	}
	return ""
}
func (m *model) modelsPanel() string {
	if panel := m.modelsForm().chooser; panel != "" {
		return panel
	}
	return modelsLeafPanel(m.focused().id)
}
func (m *model) modelsVisible(c control) bool {
	if strings.HasPrefix(c.id, "nav:") || c.id == "help" || c.id == "palette" {
		return m.modelsForm().chooser == ""
	}
	panel := m.modelsPanel()
	if panel != "" && panel != "advanced" {
		return modelsLeafPanel(c.id) == panel || c.id == "models:close" || c.id == "field:"+panel ||
			(panel == "model" && (c.id == "advanced:model" || c.id == "discover")) || (panel == "budget" && c.id == "advanced:budget")
	}
	if modelsLeafPanel(c.id) != "" {
		return modelsLeafPanel(c.id) == "advanced" && (m.modelsForm().advanced || panel == "advanced")
	}
	return c.id != "models:close"
}
func (m *model) openModelsChoice(field string) {
	f := m.modelsForm()
	f.origin = m.location()
	f.chooser = field
	m.setModelsForm(f)
	id := ""
	p := m.draft().Profile
	switch field {
	case "model":
		id = "model:" + p.Model
	case "budget":
		id = fmt.Sprintf("budget:%d", p.Budget)
	case "tier":
		id = "tier:" + tierLabel(p.Tier)
	}
	// A custom budget remains visible in the field; its editor is the safe start.
	m.focusID(id)
	if m.focused().id == "field:"+field {
		if field == "budget" {
			m.focusID("advanced:budget")
		} else {
			m.focusID("models:close")
		}
	}
	m.scroll = -1
}
func (m *model) closeModelsChoice() {
	f := m.modelsForm()
	field, origin := f.chooser, f.origin
	f.chooser = ""
	m.setModelsForm(f)
	m.restore(origin)
	m.focusID("field:" + field)
}
func (m *model) finishModelsChoice() {
	if m.screen == "Models" && m.modelsForm().chooser != "" {
		m.closeModelsChoice()
	}
}
func (m *model) modelsKey(k tea.KeyPressMsg) bool {
	if m.screen != "Models" {
		return false
	}
	key := k.String()
	f := m.modelsForm()
	if key == "esc" {
		if f.chooser != "" {
			m.closeModelsChoice()
			return true
		}
		if f.advanced {
			f.advanced = false
			m.setModelsForm(f)
			m.focusID("models:advanced")
			m.scroll = -1
			return true
		}
	}
	if key != "tab" && key != "shift+tab" && key != "down" && key != "up" {
		return false
	}
	cs := m.controls()
	step := 1
	if key == "shift+tab" || key == "up" {
		step = -1
	}
	tab := key == "tab" || key == "shift+tab"
	region := m.region(m.focus, m.focused())
	if !tab && region == "Nav" {
		return true
	}
	for n := 1; n <= len(cs); n++ {
		i := m.focus + n*step
		if !tab && (i < 0 || i >= len(cs)) {
			return true
		}
		i = (i%len(cs) + len(cs)) % len(cs)
		if m.modelsVisible(cs[i]) && (tab || m.region(i, cs[i]) == region) {
			m.focus = i
			m.scroll = -1
			return true
		}
	}
	return true
}

func (m *model) modelsRows() []row {
	width := m.contentWidth()
	cs := m.controls()
	controlAt := func(id string) (control, int) {
		for i, c := range cs {
			if c.id == id {
				return c, i
			}
		}
		return control{}, -1
	}
	render := func(id string, w int) []row {
		c, i := controlAt(id)
		if i < 0 {
			return nil
		}
		if id == "plan" || id == "back" || id == "discard" {
			return m.actionRows(c, i, w, id == "plan")
		}
		if id == "models:advanced" {
			c.label = "[+] Advanced / custom settings"
			if m.modelsForm().advanced || m.modelsPanel() == "advanced" {
				c.label = "[-] Advanced / custom settings"
			}
		}
		if m.height <= 12 {
			switch id {
			case "field:model":
				c.label = "Model: " + m.draft().Profile.Model + " [Choose]"
			case "field:budget":
				c.label = fmt.Sprintf("Budget: %d tokens [Choose]", m.draft().Profile.Budget)
			case "field:tier":
				c.label = "Tier: " + tierLabel(m.draft().Profile.Tier) + " [Choose]"
			}
		}
		r := m.controlRow(c, i)
		rs := m.column([]row{r}, w)
		if strings.HasPrefix(id, "field:") {
			for j := range rs {
				rs[j].text = m.fill(rs[j].text, w, surfaceRaised)
			}
		}
		return rs
	}
	panel := m.modelsPanel()
	if panel != "" && panel != "advanced" {
		title := map[string]string{"model": "Choose model ID", "budget": "Choose context budget", "tier": "Choose default request tier"}[panel]
		rs := []row{{text: m.heading(title)}, {text: m.note("Arrows focus; Enter chooses. Esc keeps draft.")}, {}}
		if panel == "model" {
			rs = append(rs, row{text: m.note("Suggested IDs / availability unverified")})
		}
		for _, c := range cs {
			if m.modelsVisible(c) && !strings.HasPrefix(c.id, "field:") && !strings.HasPrefix(c.id, "nav:") && c.id != "help" && c.id != "palette" {
				rs = append(rs, render(c.id, width)...)
			}
		}
		if panel == "model" {
			evidence := "No advertised inventory; use an exact ID or Discover."
			if len(m.models) > 0 {
				evidence = "Advertised inventory only; capacity and tiers unverified."
			}
			if m.busy == "discover" {
				evidence = "Discovering inventory; existing choices remain unverified."
			}
			rs = append(rs, row{}, row{text: m.note(evidence)})
			for _, info := range m.models {
				rs = append(rs, row{text: m.note(fmt.Sprintf("Source: %s / %s / advertised context %d", m.public(info.Source), m.public(info.ID), info.Context))})
			}
		}
		if panel == "tier" {
			rs = append(rs, row{}, row{text: "Model / default: " + m.modelDefault(m.draft().Profile.Model, m.draft().Profile.Tier)}, row{text: m.note("Available tiers: " + tierAvailability(m.draft().Profile.TiersConfirmed))}, row{text: m.note("Fast / Ultrafast require explicit route confirmation in Advanced.")})
		}
		return rs
	}

	d := m.draft()
	client, ok := m.client()
	leftWidth, rightWidth := width, width
	wide := m.width >= 100
	if wide {
		leftWidth = (width - 3) / 2
		rightWidth = width - leftWidth - 3
	}
	var left, right []row
	compact := m.height <= 12
	if !compact {
		left = append(left, row{text: m.heading("MANAGEMENT PREFERENCE")}, row{})
		if ok && m.editable() != "" {
			installed := "Installed"
			if !client.Installed {
				installed = "Not installed"
			}
			left = append(left, row{text: m.note(fmt.Sprintf("%s | Managed: %t", installed, client.Managed))})
		}
	}
	left = append(left, render("enabled", leftWidth)...)
	if !compact {
		message := "Settings stay in this client's draft until reviewed."
		if why := m.editable(); why != "" {
			message = why
		}
		left = append(left, row{text: m.note(m.public(message))}, row{})
	}
	for _, id := range []string{"field:model", "field:budget", "field:tier"} {
		left = append(left, render(id, leftWidth)...)
		if !compact {
			left = append(left, row{})
		}
	}
	if !compact {
		left = append(left, row{text: m.note("Available tiers: " + tierAvailability(d.Profile.TiersConfirmed))}, row{text: m.note("Budget is a choice; capacity is not verified.")}, row{})
	}
	left = append(left, render("models:advanced", leftWidth)...)
	if m.modelsForm().advanced || panel == "advanced" {
		left = append(left, row{text: m.heading("CUSTOM SETTINGS / ROUTE ATTESTATIONS")})
		for _, id := range []string{"capacity", "tiers", "advanced:model", "advanced:budget", "advanced:compact", "advanced:endpoint", "advanced:credential", "discover"} {
			left = append(left, render(id, leftWidth)...)
		}
	}
	right = append(right, row{text: m.heading("DRAFT / NOT YET APPLIED")})
	if ok {
		state := "NO CHANGES"
		if d != m.baselines[m.selected] {
			state = "UNSAVED"
		}
		right = append(right, row{text: m.heading(m.public(client.Name)) + " / " + state}, row{})
		right = append(right, m.detailRows("Applied", m.modelDefault(client.Profile.Model, client.Profile.Tier), rightWidth)...)
		right = append(right, m.detailRows("Draft", m.modelDefault(d.Profile.Model, d.Profile.Tier), rightWidth)...)
		right = append(right, m.detailRows("Budget", fmt.Sprintf("%d tokens", d.Profile.Budget), rightWidth)...)
		right = append(right, m.detailRows("Management", map[bool]string{true: "Enabled", false: "Disabled"}[d.Enabled], rightWidth)...)
	} else {
		right = append(right, row{text: "No client selected; return to Dashboard."})
	}
	right = append(right, row{}, row{text: m.heading("Review before applying")}, row{text: m.note("Only this client's draft. No settings written here.")}, row{})
	if !wide {
		// Put useful actions before the secondary summary in a short viewport.
		var actions []row
		for _, id := range []string{"plan", "discard", "back", "help", "palette"} {
			actions = append(actions, render(id, width)...)
		}
		return append(append(append(left, row{}), actions...), append([]row{{}}, right...)...)
	}
	for _, id := range []string{"plan", "discard", "back", "help", "palette"} {
		right = append(right, render(id, rightWidth)...)
	}
	left, right = m.column(left, leftWidth), m.column(right, rightWidth)
	var rs []row
	for i := 0; i < max(len(left), len(right)); i++ {
		l, r := row{}, row{}
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		separator := " │ "
		if m.options.ASCII {
			separator = " | "
		}
		rs = append(rs, row{text: m.fill(l.text, leftWidth, surfaceBase) + m.note(separator) + r.text, focus: l.focus || r.focus})
	}
	return rs
}

func (m *model) modelsHeader() string {
	name := "No client selected"
	if c, ok := m.client(); ok {
		name = m.public(c.Name)
	}
	return m.shellLine(m.heading(ansi.Truncate("oneline-ccc / Models / Selected: "+name, m.contentWidth(), m.ellipsis())), surfaceRaised)
}
