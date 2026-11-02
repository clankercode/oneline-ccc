package tui

func (m *model) sshRows() []row {
	status := "Availability / not verified"
	if m.sshReason() != "" {
		status = "Unavailable / helper not ready"
	}
	if m.busy == "ssh" {
		status = "Handoff pending / outcome awaited"
	}
	header := []row{{text: m.heading("SSH HELPER / EXPLICIT HANDOFF")}, {text: m.statusText(status)}}
	width := m.contentWidth()
	if width >= 96 {
		width = (width - 3) / 2
	}
	facts := []row{{text: m.heading("SCOPE / ACCESS EVIDENCE")}}
	for _, fact := range [][2]string{
		{"Current action", "Terminal handoff only"},
		{"Server state", "Not inspected"},
		{"Firewall", "Not inspected"},
		{"Remote login", "Not verified"},
		{"Elevation", "Helper requests separately"},
		{"Access scope", "Review account, key and firewall in helper"},
	} {
		facts = append(facts, m.detailRows(fact[0], fact[1], width)...)
	}
	facts = append(facts, row{}, row{text: m.note("A helper exit does not verify SSH access. Windows setup-script success is separate from this native handoff.")})
	if reason := m.sshReason(); reason != "" {
		facts = append(facts, row{}, row{text: m.note(reason)})
	} else {
		facts = append(facts, row{}, row{text: m.note("The backend checks for a configured Windows helper on request; availability is not yet verified.")})
	}

	actions := []row{{text: m.heading("REVIEW BEFORE YIELDING TERMINAL")}}
	for i, c := range m.controls() {
		if i < navCount {
			continue
		}
		if c.id == "ssh" {
			c.label = "Review SSH handoff"
		}
		actions = append(actions, m.controlRow(c, i))
		if c.id == "ssh" {
			actions = append(actions, row{text: m.note("Opens confirmation / Back keeps this console.")}, row{}, row{text: m.heading("TERMINAL OWNERSHIP / STEPS")},
				row{text: "1. Review and explicitly confirm the handoff."},
				row{text: "2. Helper owns terminal prompts and administrator review."},
				row{text: "3. On return, inspect the exit result before retrying."},
				row{}, row{text: m.note("Opening review enables no server or firewall rule. Remote login needs separate verification.")}, row{})
		}
	}
	return m.serviceSSHComposition(header, facts, actions)
}
