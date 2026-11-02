package tui

func (m *model) utilitiesRows() []row {
	width := m.contentWidth()
	compact := m.height <= 20
	rs := []row{{text: m.heading("Utilities") + m.note(" / Read only")}}
	if !compact {
		rs = append(rs, row{text: m.fill(m.statusText("READ ONLY / lifecycle unavailable"), width, surfaceRaised)})
		rs = append(rs, m.detailRows("Installation", "Not inspected", width)...)
		rs = append(rs, row{}, row{text: m.heading("Operations / this backend")})
	}
	reason := ""
	for i, c := range m.controls() {
		if i >= navCount {
			if compact && c.reason != "" {
				// Keep Back in the tiny initial viewport without changing
				// control order. The full backend reason follows these controls.
				reason = c.reason
				c.reason = ""
			}
			rs = append(rs, m.manualActionRows(c, i, c.id == "back")...)
		}
	}
	if compact {
		rs = append(rs, row{}, row{text: m.heading("Why unavailable")}, row{text: m.note(m.public(reason))})
		rs = append(rs, m.detailRows("Installation", "Not inspected", width)...)
	}
	rs = append(rs, row{}, row{text: m.heading("Platform / project identity")})
	rs = append(rs, m.detailRows("Platform support", "Not verified", width)...)
	rs = append(rs, row{text: m.note("No broader adapters are promised by this build.")}, row{text: m.note("Confirm the exact project before using its own setup instructions.")}, row{})
	rs = append(rs, m.manualNoticeRows("Keep working", "Back returns to your previous screen; the selected client and drafts are kept.", width)...)
	return rs
}
