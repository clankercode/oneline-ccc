package tui

func (m *model) utilitiesRows() []row {
	width := m.contentWidth()
	rs := []row{{text: m.heading("Utilities")}, {text: m.statusText("READ ONLY / lifecycle unavailable")}}
	rs = append(rs, m.detailRows("Installation", "Not inspected", width)...)
	rs = append(rs, row{}, row{text: m.heading("Operations / this backend")})
	for i, c := range m.controls() {
		if i >= navCount {
			rs = append(rs, m.manualActionRows(c, i, c.id == "back")...)
		}
	}
	rs = append(rs, row{}, row{text: m.heading("Platform / project identity")}, row{text: m.note("No verified platform installer or updater is available in this build.")}, row{text: m.note("No broader adapters are promised by this build.")}, row{text: m.note("Confirm the exact project before using its own setup instructions.")}, row{}, row{text: m.heading("Keep working")}, row{text: m.note("Back returns to your previous screen; the selected client and drafts are kept.")})
	return rs
}
