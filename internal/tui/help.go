package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Help shares its rendered page boundaries with input and resize handling.
// Count wrapped terminal rows after reserving the actual shell and page chrome.
func (m *model) helpPagination() (size, last int) {
	pages, size := m.helpPages()
	return size, len(pages) - 1
}

func (m *model) helpPages() ([][]string, int) {
	width := m.bodyWidth()
	wrap := func(text string) []string {
		return strings.Split(ansi.Wrap(m.public(text), width, ""), "\n")
	}
	var entries [][]string
	total := 0
	for _, c := range m.catalog() {
		label := c.label
		if c.key != "" {
			label = c.key + "  " + label
		}
		lines := wrap(label)
		entries = append(entries, lines)
		total += len(lines)
	}
	// One row per page is the maximum possible page count; use its digit width
	// to reserve a stable heading even when a narrower terminal adds pages.
	title := fmt.Sprintf("Shortcut help | Page %d of %d", max(1, total), max(1, total))
	available := m.height - len(m.shellHeader()) - len(m.shellFooter(""))
	capacity := max(1, available-len(wrap(title))-len(wrap("> Close help (Enter / Esc)")))
	pages := [][]string{{}}
	for _, entry := range entries {
		// Keep a binding and its description together whenever it fits a page.
		if len(entry) <= capacity && len(pages[len(pages)-1])+len(entry) > capacity {
			pages = append(pages, nil)
		}
		for _, line := range entry {
			last := len(pages) - 1
			if len(pages[last]) == capacity {
				pages = append(pages, nil)
				last++
			}
			pages[last] = append(pages[last], line)
		}
	}
	return pages, capacity
}
