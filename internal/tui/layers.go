package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// layout passes editors their actual inset width. Modals still receive the
// terminal width. Keep presentation sizing separate from input/model state.
func (m *model) layerWidth() int {
	if m.editor != nil {
		return max(1, m.width)
	}
	return m.contentWidth()
}

func (m *model) layerBand(text string, role surfaceRole) []row {
	var rows []row
	for _, line := range strings.Split(ansi.Wrap(text, m.layerWidth(), ""), "\n") {
		rows = append(rows, row{text: m.fill(line, m.layerWidth(), role)})
	}
	return rows
}

func (m *model) layerTitle(title string) []row {
	return m.layerBand(m.heading(m.public(title)), surfaceRaised)
}

// Modal/editor input consumes paging keys. Add supporting prose or spacing
// only if the entire form still fits, including every wrapped choice line.
func (m *model) layerFits(parts ...[]row) bool {
	room := max(1, m.height-len(m.shellHeader())-len(m.shellFooter("")))
	for _, part := range parts {
		room -= len(m.column(part, m.layerWidth()))
	}
	return room >= 0
}

func (m *model) layerChoice(label string, focused, primary bool) []row {
	var rows []row
	for i, line := range strings.Split(ansi.Wrap(m.public(label), max(1, m.layerWidth()-2), ""), "\n") {
		prefix := "  "
		if focused && i == 0 {
			prefix = "> "
		}
		text := prefix + line
		if primary {
			text = m.fill(text, min(m.layerWidth(), ansi.StringWidth(m.public(label))+4), surfacePrimary)
		}
		rows = append(rows, row{text: text, focus: focused && i == 0})
	}
	return rows
}

// A menu window counts wrapped terminal rows, including its header and the
// existing footer. A selected command's refusal reason is part of its block;
// it cannot be displaced by a fixed number of neighboring command labels.
func (m *model) layerMenu(header []row, choices [][]row, selected int) []row {
	available := max(1, m.height-len(m.shellHeader())-len(m.shellFooter("")))
	header = m.column(header, m.layerWidth())
	for i := range choices {
		choices[i] = m.column(choices[i], m.layerWidth())
	}
	selected = max(0, min(selected, len(choices)-1))
	for len(header) > 1 && len(header)+len(choices[selected]) > available {
		header = header[:len(header)-1]
	}
	start, end := selected, selected+1
	used := len(header) + len(choices[selected])
	// Keep useful context on both sides of the chosen result when it fits.
	for start > 0 || end < len(choices) {
		advanced := false
		if end < len(choices) && used+len(choices[end]) <= available {
			used += len(choices[end])
			end++
			advanced = true
		}
		if start > 0 && used+len(choices[start-1]) <= available {
			start--
			used += len(choices[start])
			advanced = true
		}
		if !advanced {
			break
		}
	}
	for _, block := range choices[start:end] {
		header = append(header, block...)
	}
	return header
}

// Keep the caret and adjacent wide/combining runes within a single field row.
// This is a view of the buffer: Home/End and ordinary input retain every rune.
func layerCaret(before, after string, space int) string {
	before, after = clean(before), clean(after)
	space = max(0, space)
	if ansi.StringWidth(before) > space {
		whole := before
		for cut := ansi.StringWidth(whole) - space; ansi.StringWidth(before) > space; cut++ {
			before = ansi.TruncateLeft(whole, cut, "")
		}
	}
	after = ansi.Truncate(after, max(0, space-ansi.StringWidth(before)), "")
	return before + "|" + after
}
