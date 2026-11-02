package tui

func (m *model) settingsRows() []row {
	width := m.gatewaySettingsPanelWidth()
	left := []row{m.gatewaySettingsSection("DISPLAY / CURRENT CHOICES", width)}
	for i, c := range m.controls() {
		switch c.id {
		case "theme":
			c.label = "Appearance        " + m.public(m.options.Theme) + " (current)"
		case "ascii":
			c.label = m.gatewaySettingsToggle("ASCII decoration", m.options.ASCII)
		case "color":
			c.label = m.gatewaySettingsToggle("No color", m.options.NoColor)
		case "motion":
			c.label = m.gatewaySettingsToggle("Reduced motion", m.options.ReducedMotion)
		default:
			continue
		}
		left = append(left, m.gatewaySettingsAction(c, i)...)
	}
	if m.height > 12 {
		left = append(left, row{text: m.note("This session / Enter cycles appearance.")})
	}
	if m.width >= 100 {
		left = append(left, row{}, row{text: m.note("Display choices keep client configuration unchanged.")})
	}
	right := []row{m.gatewaySettingsSection("CREDENTIALS / STORAGE INTENT", width)}
	right = append(right, m.gatewaySettingsFact("Availability", m.gatewaySettingsCredential()))
	right = append(right, m.gatewaySettingsFact("Env reference", m.gatewaySettingsReference()))
	right = append(right, m.gatewaySettingsFact("Save default", "Session only / vault is opt-in"))
	for i, c := range m.controls() {
		if c.id == "key" {
			c.label = "Enter API key (hidden)"
			right = append(right, m.gatewaySettingsAction(c, i)...)
		}
	}
	right = append(right, row{}, m.gatewaySettingsSection("TOOLS / NAVIGATION", width))
	for i, c := range m.controls() {
		switch c.id {
		case "nav:Utilities":
			c.label = "Utilities / unavailable installer"
		case "nav:Service":
			c.label = "Background monitor / optional"
		default:
			continue
		}
		right = append(right, m.gatewaySettingsAction(c, i)...)
	}
	right = append(right, row{text: m.note("Native configuration: Codex only")})
	if version := m.gatewaySettingsVersion(); version != "" {
		right = append(right, m.gatewaySettingsFact("Version", version))
	}
	rs := []row{{text: m.heading("Settings & navigation / this session")}}
	if m.width >= 100 {
		rs = append(rs, row{})
	}
	rs = append(rs, m.gatewaySettingsPanels(left, right)...)
	return append(rs, m.gatewaySettingsReturn()...)
}
