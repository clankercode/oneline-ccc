package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// These composers only arrange public facts and the existing control catalog.
// Credential availability is an observation; storage intent and verification
// remain separate, even when a key or a local endpoint is present.
func (m *model) gatewayRows() []row {
	width := m.gatewaySettingsPanelWidth()
	left := []row{m.gatewaySettingsSection("CONNECTION / LOCAL DRAFT", width)}
	left = append(left, m.gatewaySettingsFact("Alias", "Gateway A"))
	endpoint := "Private local value"
	if m.draft().Profile.BaseURL == "" {
		endpoint = "Not set in selected draft"
	}
	left = append(left, m.gatewaySettingsFact("Endpoint", endpoint))
	for i, c := range m.controls() {
		if c.id == "advanced:endpoint" {
			left = append(left, m.gatewaySettingsAction(c, i)...)
		}
	}
	verification := []row{m.gatewaySettingsSection("VERIFICATION / UNVERIFIED", width),
		m.gatewaySettingsFact("Route", m.statusText("Not checked")),
		m.gatewaySettingsFact("Auth / API", m.statusText("Not tested")),
		m.gatewaySettingsFact("Inference", "Not tested"),
		{text: m.note("An endpoint and key do not verify models or tiers.")},
	}
	if m.width >= 100 {
		left = append(append(left, row{}), verification...)
	}
	right := []row{m.gatewaySettingsSection("CREDENTIAL / VALUE NEVER SHOWN", width)}
	right = append(right, m.gatewaySettingsFact("Availability", m.gatewaySettingsCredential()))
	right = append(right, m.gatewaySettingsFact("Env reference", m.gatewaySettingsReference()))
	right = append(right, m.gatewaySettingsFact("Save default", "Session only"))
	right = append(right, row{text: m.note("Vault storage is explicit in key entry.")})
	for i, c := range m.controls() {
		if c.id == "key" {
			right = append(right, m.gatewaySettingsAction(c, i)...)
		}
	}
	right = append(right, row{text: m.note("Reference edits: Models / Advanced.")})
	rs := []row{{text: m.heading("Gateway & credentials")}}
	if m.height > 12 {
		rs = append(rs, row{text: m.note("Local draft / values stay private")})
	}
	if m.width >= 100 {
		rs = append(rs, row{})
	}
	rs = append(rs, m.gatewaySettingsPanels(left, right)...)
	if m.width < 100 {
		rs = append(append(rs, row{}), verification...)
	}
	return append(rs, m.gatewaySettingsReturn()...)
}

func (m *model) gatewaySettingsSection(label string, width int) row {
	return row{text: m.fill(m.heading(label), width, surfaceRaised)}
}

func (m *model) gatewaySettingsFact(label, value string) row {
	return row{text: m.note(fmt.Sprintf("%-14s", label)) + value}
}

func (m *model) gatewaySettingsCredential() string {
	if m.snapshot.Credential == "" {
		return "Not observed"
	}
	return m.public(m.snapshot.Credential)
}

func (m *model) gatewaySettingsReference() string {
	if _, ok := m.client(); !ok {
		return "No client selected"
	}
	if ref := m.draft().Profile.EnvKey; ref != "" {
		return m.public(ref)
	}
	return "Not set in selected draft"
}

func (m *model) gatewaySettingsAction(c control, index int) []row {
	// Keep the logical control's exact index and reason. A separately wrapped
	// reason cannot crowd its action or hide the focus marker in narrow panes.
	r := m.controlRow(control{id: c.id, label: c.label}, index)
	if c.reason == "" && (c.id == "key" || c.id == "advanced:endpoint") {
		r.text = m.heading(r.text)
	}
	rs := []row{r}
	if c.reason != "" {
		rs = append(rs, row{text: m.note("Unavailable: " + m.public(c.reason))})
	}
	return rs
}

func (m *model) gatewaySettingsToggle(label string, enabled bool) string {
	state := "Off"
	if enabled {
		state = "On"
	}
	return fmt.Sprintf("%-18s%s %s", label, checkbox(enabled), state)
}

func (m *model) gatewaySettingsPanelWidth() int {
	if m.width >= 100 {
		return (m.contentWidth() - 3) / 2
	}
	return m.contentWidth()
}

// Pre-fill each cell so a focused control's background stays in its own panel
// when the ordinary body column applies the outer focus treatment.
func (m *model) gatewaySettingsPanels(left, right []row) []row {
	if m.width < 100 {
		return append(append(left, row{}), right...)
	}
	width := m.gatewaySettingsPanelWidth()
	rightWidth := m.contentWidth() - width - 3
	left, right = m.column(left, width), m.column(right, rightWidth)
	var rs []row
	for i := 0; i < max(len(left), len(right)); i++ {
		l, r := row{}, row{}
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		cell := func(r row, cellWidth int) string {
			role := surfaceBase
			if r.focus {
				role = surfaceActive
			}
			text := m.fill(r.text, cellWidth, role)
			if !m.options.NoColor && !r.focus {
				// A base style sets colors but inherits an enclosing bold SGR.
				normal := ansi.Style{}.Normal().String()
				text = normal + strings.NewReplacer(ansi.ResetStyle, ansi.ResetStyle+normal, "\x1b[0m", "\x1b[0m"+normal).Replace(text)
			}
			return text
		}
		divider := "│"
		if m.options.ASCII {
			divider = "|"
		}
		rs = append(rs, row{text: cell(l, width) + m.fill(" "+divider+" ", 3, surfaceBase) + cell(r, rightWidth), focus: l.focus || r.focus})
	}
	return rs
}

func (m *model) gatewaySettingsReturn() []row {
	rs := []row{{}, m.gatewaySettingsSection("RETURN / HELP", m.contentWidth())}
	for i, c := range m.controls() {
		if c.id == "back" || c.id == "help" || c.id == "palette" {
			rs = append(rs, m.gatewaySettingsAction(c, i)...)
		}
	}
	return rs
}

// Keep literal cues in NO_COLOR/ASCII; all dynamic text enters through public.
func (m *model) gatewaySettingsVersion() string {
	return strings.TrimSpace(m.public(m.options.Version))
}
