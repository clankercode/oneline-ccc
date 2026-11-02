package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type reviewRole uint8

const (
	reviewValue reviewRole = iota
	reviewSection
	reviewBefore
	reviewAfter
)

type reviewLine struct {
	text, scope string
	role        reviewRole
}

// Pagination operates on sanitized, unstyled values. Rendering applies roles
// afterward, so a style cannot change which values the operator acknowledges.
func (m *model) reviewDocumentPages() [][]reviewLine {
	if m.plan == nil {
		return [][]reviewLine{{{text: "No plan"}}}
	}
	p := m.plan
	var document []reviewLine
	section, field := "Plan", ""
	add := func(role reviewRole, text string) {
		if role == reviewSection {
			section = text
		}
		if strings.HasPrefix(text, "Field: ") {
			field = strings.TrimPrefix(text, "Field: ")
		}
		scope := section
		if role == reviewBefore {
			scope = "Before / " + field
		} else if role == reviewAfter {
			scope = "After / " + field
		}
		for _, line := range strings.Split(ansi.Wrap(m.public(text), max(1, m.width), ""), "\n") {
			document = append(document, reviewLine{text: line, scope: scope, role: role})
		}
	}
	add(reviewSection, "PLAN / EXACT VALUES")
	add(reviewValue, "Client: "+p.ClientID)
	add(reviewValue, "Plan: "+p.ID)
	add(reviewValue, "Revision: "+p.Revision)
	add(reviewValue, "")
	add(reviewSection, "EFFECTIVE SETTINGS")
	add(reviewValue, fmt.Sprintf("Manage: %t", p.Desired.Enabled))
	add(reviewValue, "Exact model: "+p.Desired.Profile.Model)
	add(reviewValue, fmt.Sprintf("Budget: %d | Compact: %d", p.Desired.Profile.Budget, p.Desired.Profile.Compact))
	add(reviewValue, "Default tier: "+tierLabel(p.Desired.Profile.Tier))
	add(reviewValue, "Available tiers: "+tierAvailability(p.Desired.Profile.TiersConfirmed))
	add(reviewValue, "")
	add(reviewSection, "FIELD CHANGES")
	for i, change := range p.Changes {
		if i > 0 {
			add(reviewValue, "")
		}
		add(reviewValue, "Field: "+change.Field)
		add(reviewBefore, "  Before: "+change.Before)
		add(reviewAfter, "  After: "+change.After)
	}
	if p.Noop {
		add(reviewValue, "No changes; Apply is unavailable")
	}
	// Reserve the page title, Back/read/Apply controls, and one context row.
	// Help/Commands remain reachable in the same scrolling document. The full
	// authority page must fit while the acknowledgment control has focus.
	available := m.height - len(m.shellHeader()) - len(m.shellFooter(""))
	count := max(1, available-5)
	if count <= 2 {
		// On very short terminals, the pinned scope already names the group.
		// Do not make the operator acknowledge pages containing only decoration.
		compact := document[:0]
		for _, line := range document {
			if line.role != reviewSection && line.text != "" {
				compact = append(compact, line)
			}
		}
		document = compact
	}
	var pages [][]reviewLine
	for i := 0; i < len(document); i += count {
		pages = append(pages, document[i:min(i+count, len(document))])
	}
	return pages
}

// Keep the plain projection shared by input, acknowledgment and capture tools.
func (m *model) reviewPages() [][]string {
	var pages [][]string
	for _, page := range m.reviewDocumentPages() {
		var lines []string
		for _, line := range page {
			lines = append(lines, line.text)
		}
		pages = append(pages, lines)
	}
	return pages
}

func (m *model) reviewLineStyle(line reviewLine) string {
	switch line.role {
	case reviewSection:
		return m.fill(m.heading(line.text), max(1, m.width), surfaceRaised)
	case reviewBefore, reviewAfter:
		if m.options.NoColor || m.options.Theme == "mono" {
			return line.text
		}
		color := "#F5A3A3"
		if line.role == reviewAfter {
			color = "#91D6A5"
		}
		if m.options.Theme == "light" {
			color = "#9B2727"
			if line.role == reviewAfter {
				color = "#23613A"
			}
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(line.text)
	default:
		if label, value, ok := strings.Cut(line.text, ": "); ok {
			return m.note(label+": ") + value
		}
		return line.text
	}
}

func (m *model) reviewRows() []row {
	var rs []row
	if m.plan == nil {
		rs = append(rs, row{text: m.heading("Review changes")}, row{}, row{text: "No active review. Request a plan for the selected draft."})
	} else {
		pages := m.reviewDocumentPages()
		page := max(0, min(m.reviewPage, len(pages)-1))
		rs = append(rs, row{text: m.heading(fmt.Sprintf("Exact changes | Page %d of %d", page+1, len(pages)))})
		if pages[page][0].role != reviewSection {
			scope := ansi.Truncate(m.public(pages[page][0].scope), max(1, m.width), m.ellipsis())
			rs = append(rs, row{text: m.note(scope)})
		}
		for _, line := range pages[page] {
			rs = append(rs, row{text: m.reviewLineStyle(line)})
		}
	}
	for i, control := range m.controls() {
		if i < navCount {
			continue
		}
		r := m.controlRow(control, i)
		if control.id == "review:apply" && control.reason == "" && !m.options.NoColor {
			r.text = m.surface(surfacePrimary).Padding(0, 1).Render(r.text)
		} else if control.id == "review:read" {
			r.text = m.heading(r.text)
		} else if control.id == "help" || control.id == "palette" {
			r.text = m.note(r.text)
		}
		rs = append(rs, r)
	}
	return rs
}
