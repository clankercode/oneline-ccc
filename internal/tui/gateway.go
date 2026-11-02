package tui

import (
	"fmt"
	"strings"
)

// These composers only arrange public facts and the existing control catalog.
// Credential availability is an observation; storage intent and verification
// remain separate, even when a key or a local endpoint is present.
func (m *model) gatewayRows() []row {
	width := m.contentWidth()
	rs := []row{{text: m.heading("Gateway & credentials")}, {text: m.note("Local draft / values stay private")}, {}}
	rs = append(rs, m.gatewaySettingsSection("CONNECTION / LOCAL DRAFT", width))
	rs = append(rs, m.gatewaySettingsFact("Alias", "Gateway A"))
	endpoint := "Private local value"
	if m.draft().Profile.BaseURL == "" {
		endpoint = "Not set in selected draft"
	}
	rs = append(rs, m.gatewaySettingsFact("Endpoint", endpoint))
	for i, c := range m.controls() {
		if c.id == "advanced:endpoint" {
			rs = append(rs, m.gatewaySettingsAction(c, i)...)
		}
	}
	rs = append(rs, row{}, m.gatewaySettingsSection("CREDENTIAL / VALUE NEVER SHOWN", width))
	rs = append(rs, m.gatewaySettingsFact("Availability", m.gatewaySettingsCredential()))
	rs = append(rs, m.gatewaySettingsFact("Env reference", m.gatewaySettingsReference()))
	rs = append(rs, m.gatewaySettingsFact("Save default", "Session only"))
	rs = append(rs, row{text: m.note("Vault storage is an explicit choice in hidden key entry.")})
	for i, c := range m.controls() {
		if c.id == "key" {
			rs = append(rs, m.gatewaySettingsAction(c, i)...)
		}
	}
	rs = append(rs, row{}, m.gatewaySettingsSection("VERIFICATION / NO REQUEST PROOF", width))
	rs = append(rs, m.gatewaySettingsFact("Route", m.statusText("Not checked")))
	rs = append(rs, m.gatewaySettingsFact("Auth / API", m.statusText("Not tested")))
	rs = append(rs, m.gatewaySettingsFact("Model / tiers", "Not verified by an endpoint or key"))
	rs = append(rs, row{}, m.gatewaySettingsSection("RETURN / HELP", width))
	for i, c := range m.controls() {
		if c.id == "back" || c.id == "help" || c.id == "palette" {
			rs = append(rs, m.gatewaySettingsAction(c, i)...)
		}
	}
	return rs
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
	if c.reason == "" && c.id != "back" && c.id != "help" && c.id != "palette" {
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

// Keep literal cues in NO_COLOR/ASCII; all dynamic text enters through public.
func (m *model) gatewaySettingsVersion() string {
	return strings.TrimSpace(m.public(m.options.Version))
}
