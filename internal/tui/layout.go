package tui

// layout is shared by rendering and paging. Negative scroll follows focus;
// explicit zero means the operator requested the beginning of the document.
func (m *model) layout() (lines []row, focusLine, start, available int) {
	var body []row
	switch {
	case m.editor != nil:
		// Editors size the caret window from width. Supply the actual inset
		// viewport without changing input state or the terminal dimensions.
		render := *m
		render.width = m.contentWidth()
		body = render.editorRows()
	case m.modal != "":
		body = m.modalRows()
	case m.screen == "Dashboard":
		body = m.dashboardRows()
	default:
		body = m.screenRows()
	}
	if m.wideDashboard() {
		lines = body
	} else {
		lines = m.column(body, m.contentWidth())
	}
	available = max(1, m.height-len(m.shellHeader())-len(m.shellFooter("")))
	focusLine = -1
	for i, r := range lines {
		if r.focus {
			focusLine = i
			break
		}
	}
	start = m.scroll
	if m.editor != nil || m.modal != "" {
		start = -1
	}
	if start < 0 {
		start = 0
		if focusLine >= 0 {
			context := min(available, m.focusContext(lines, focusLine))
			if focusLine+context > available {
				start = focusLine + context - available
				if (m.modal != "" || m.editor != nil) && available > 1 {
					start++
				}
			}
		}
	}
	start = max(0, min(start, max(0, len(lines)-available)))
	return
}

func (m *model) focusContext(lines []row, focus int) int {
	if m.screen == "Dashboard" && m.modal == "" && m.editor == nil && m.region(m.focus, m.focused()) == "Clients" {
		return min(3, len(lines)-focus)
	}
	return 1
}
