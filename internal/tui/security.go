package tui

import (
	"io"
	"regexp"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r != '\n' && (unicode.IsControl(r) || r == 0x202e || r == 0x202d || r == 0x202a || r == 0x202b || r == 0x202c || r == 0x2066 || r == 0x2067 || r == 0x2068 || r == 0x2069) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}

var urlPattern = regexp.MustCompile(`(?i)https?://[^\s]+`)

func (m *model) public(s string) string {
	s = clean(s)
	for _, c := range m.snapshot.Clients {
		for _, private := range []string{c.Profile.BaseURL, c.Executable} {
			if private != "" {
				s = strings.ReplaceAll(s, private, "[private]")
			}
		}
	}
	for _, d := range m.drafts {
		if d.Profile.BaseURL != "" {
			s = strings.ReplaceAll(s, d.Profile.BaseURL, "Gateway A")
		}
	}
	return urlPattern.ReplaceAllString(s, "[private route]")
}
func desiredClient(c core.Client) core.Desired {
	p := c.Profile
	if !c.Managed {
		p.TiersConfirmed = false
	}
	return core.Desired{Enabled: c.Managed, Profile: p}
}

type briefExport struct {
	text string
	out  io.Writer
}

func (e *briefExport) Run() error            { _, err := io.WriteString(e.out, e.text+"\n"); return err }
func (e *briefExport) SetStdin(io.Reader)    {}
func (e *briefExport) SetStdout(w io.Writer) { e.out = w }
func (e *briefExport) SetStderr(io.Writer)   {}
func (m *model) exportBrief() tea.Cmd {
	if m.repair == "" {
		m.status = "Prepare a redacted brief first"
		return nil
	}
	return tea.Exec(&briefExport{text: m.public(m.repair)}, func(err error) tea.Msg { return exportResult{err} })
}

type exportResult struct{ err error }
