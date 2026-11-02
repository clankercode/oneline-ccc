package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// layout is shared by rendering and paging. Negative scroll follows focus;
// explicit zero means the operator requested the beginning of the document.
func (m *model) layout() (lines []row, focusLine, start, available int) {
	var body []row
	switch {
	case m.editor != nil:
		body = m.editorRows()
	case m.modal != "":
		body = m.modalRows()
	default:
		if m.wideDashboard() {
			body = m.dashboardRows()
		} else {
			body = m.screenRows()
		}
	}
	available = max(1, m.height-5)
	focusLine = -1
	for _, r := range body {
		if r.focus {
			focusLine = len(lines)
		}
		for _, s := range strings.Split(ansi.Wrap(r.text, max(1, m.width), ""), "\n") {
			lines = append(lines, row{s, r.focus, r.selected})
		}
	}
	start = m.scroll
	if m.editor != nil || m.modal != "" {
		start = -1
	}
	if start < 0 {
		start = 0
		if focusLine >= available {
			start = focusLine - available + 1
		}
	}
	start = max(0, min(start, max(0, len(lines)-available)))
	return
}
