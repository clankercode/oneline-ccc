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
	// Put useful actions before potentially long history. Merely opening Jobs
	// never scrolls past receipts to focus a disabled cancellation command.
	for i, c := range m.controls() {
		if i < navCount {
			continue
		}
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
	if m.height > 12 {
		rs = append(rs, row{})
	}
	rs = append(rs, row{text: m.heading("RECENT RECEIPTS")}, row{text: m.rule(width)})
	receipts := append(append([]core.Receipt(nil), m.snapshot.Receipts...), m.receipts...)
	if len(receipts) == 0 {
		return append(rs, row{text: "No receipts yet. Configure a client, review its changes, then apply."})
	}
	for i, receipt := range receipts {
		rs = append(rs, row{text: m.heading(fmt.Sprintf("Receipt %02d", i+1)) + " / " + m.statusText(receipt.Outcome)})
		rs = append(rs, m.detailRows("Client", m.public(receipt.ClientID), width)...)
		rs = append(rs, m.detailRows("Receipt", m.public(receipt.ID), width)...)
		rs = append(rs, m.detailRows("Result", m.public(receipt.Summary), width)...)
		rs = append(rs, row{})
	}
	return rs
}
