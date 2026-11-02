package tui

import (
	"fmt"

	"github.com/clankercode/oneline-ccc/internal/core"
)

func (m *model) jobsRows() []row {
	width := m.contentWidth()
	rs := []row{{text: m.heading("Jobs & receipts")}}
	if m.height > 12 {
		rs = append(rs, row{}, row{text: m.heading("CURRENT ACTIVITY")})
	}
	if m.busy == "" {
		rs = append(rs, row{text: m.note("No running action")})
	} else {
		rs = append(rs, row{text: m.heading("Running: ") + m.public(m.busy)}, row{text: m.note("Cancellation requests await the backend's final outcome.")})
	}
	appendControl := func(c control, i int) {
		r := m.controlRow(c, i)
		if c.reason != "" || c.id == "help" || c.id == "palette" {
			r.text = m.note(r.text)
		} else if c.id == "back" || c.id == "cancel" && m.busy != "" {
			if !m.options.NoColor {
				r.text = m.surface(surfaceActive).Padding(0, 1).Render(r.text)
			}
		}
		rs = append(rs, r)
	}
	// Keep useful actions and the latest outcome visible together. Disabled
	// cancellation and footer shortcuts follow history rather than displacing it.
	for i, c := range m.controls() {
		if i >= navCount && (c.id == "back" || c.id == "cancel" && m.busy != "") {
			appendControl(c, i)
		}
	}
	if m.height > 12 {
		rs = append(rs, row{}, row{text: m.heading("RECENT RECEIPTS")}, row{text: m.rule(width)})
	}
	receipts := append(append([]core.Receipt(nil), m.snapshot.Receipts...), m.receipts...)
	if len(receipts) == 0 {
		rs = append(rs, row{text: "No receipts yet. Configure a client, review its changes, then apply."})
	}
	// Both sources append chronologically; show the newest outcome first.
	for i := len(receipts) - 1; i >= 0; i-- {
		receipt := receipts[i]
		rs = append(rs, row{text: m.heading(fmt.Sprintf("Receipt %02d", len(receipts)-i)) + " / " + m.statusText(receipt.Outcome)})
		rs = append(rs, m.detailRows("Client", m.public(receipt.ClientID), width)...)
		rs = append(rs, m.detailRows("Receipt", m.public(receipt.ID), width)...)
		rs = append(rs, m.detailRows("Result", m.public(receipt.Summary), width)...)
		rs = append(rs, row{})
	}
	for i, c := range m.controls() {
		if i >= navCount && c.id != "back" && !(c.id == "cancel" && m.busy != "") {
			appendControl(c, i)
		}
	}
	return rs
}
