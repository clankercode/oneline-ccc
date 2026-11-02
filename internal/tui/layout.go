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
		lines = m.column(body, m.bodyWidth())
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
	lastStart := max(0, len(lines)-available)
	if m.scroll >= 0 && m.modal == "" && m.editor == nil && focusLine >= 0 {
		// A pinned focus anchor consumes one viewport row. Include the
		// document's final row when clamping explicit paging.
		candidate := min(start, lastStart)
		if focusLine < candidate || focusLine >= candidate+available {
			lastStart = max(0, len(lines)-available+1)
		}
	}
	start = max(0, min(start, lastStart))
	return
}

func (m *model) focusContext(lines []row, focus int) int {
	if m.screen == "Dashboard" && m.modal == "" && m.editor == nil && m.region(m.focus, m.focused()) == "Clients" {
		return min(3, len(lines)-focus)
	}
	if m.screen == "Dashboard" && m.modal == "" && m.editor == nil && m.focused().reason != "" {
		return min(4, len(lines)-focus)
	}
	return 1
}

// Review pages are authority-bearing and are currently paged at terminal width
// by panels.go. Preserve that body footprint until A3c can revise pagination
// and its behavioral coverage together; an inset must not hide reviewed rows.
func (m *model) bodyWidth() int {
	if m.screen == "Review" && m.modal == "" && m.editor == nil {
		return max(1, m.width)
	}
	return m.contentWidth()
}
