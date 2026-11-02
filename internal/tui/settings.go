package tui

func (m *model) settingsRows() []row {
	width := m.contentWidth()
	rs := []row{{text: m.heading("Settings & navigation")}, {text: m.note("Display choices apply to this session")}, {}}
	rs = append(rs, m.gatewaySettingsSection("DISPLAY / CURRENT CHOICES", width))
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
		rs = append(rs, m.gatewaySettingsAction(c, i)...)
	}
	rs = append(rs, row{text: m.note("Enter appearance to cycle dark / light / mono.")})
	rs = append(rs, row{text: m.note("Display preferences do not change client configuration.")})
	rs = append(rs, row{}, m.gatewaySettingsSection("CREDENTIALS / STORAGE INTENT", width))
	rs = append(rs, m.gatewaySettingsFact("Availability", m.gatewaySettingsCredential()))
	rs = append(rs, m.gatewaySettingsFact("Env reference", m.gatewaySettingsReference()))
	rs = append(rs, m.gatewaySettingsFact("Save default", "Session only / vault is an explicit choice"))
	for i, c := range m.controls() {
		if c.id == "key" {
			rs = append(rs, m.gatewaySettingsAction(c, i)...)
		}
	}
	rs = append(rs, row{}, m.gatewaySettingsSection("TOOLS / NAVIGATION", width))
	for i, c := range m.controls() {
		switch c.id {
		case "nav:Utilities":
			c.label = "Utilities / inspect unavailable installer"
		case "nav:Service":
			c.label = "Background monitor / optional observation"
		default:
			continue
		}
		rs = append(rs, m.gatewaySettingsAction(c, i)...)
	}
	rs = append(rs, row{text: m.note("Native configuration: Codex only")})
	if version := m.gatewaySettingsVersion(); version != "" {
		rs = append(rs, m.gatewaySettingsFact("Version", version))
	}
	rs = append(rs, row{}, m.gatewaySettingsSection("RETURN / HELP", width))
	for i, c := range m.controls() {
		if c.id == "back" || c.id == "help" || c.id == "palette" {
			rs = append(rs, m.gatewaySettingsAction(c, i)...)
		}
	}
	return rs
}
