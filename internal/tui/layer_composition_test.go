package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func layerText(s string) string { return strings.Join(strings.Fields(ansi.Strip(s)), " ") }

// Real catalog entries vary in height. Moving to one must expose its complete
// label and refusal reason, even with the small terminal's wrapped footer.
func TestLayerPaletteSelectedReasonInViewport(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			m, f := loaded(t)
			m.width, m.height = size[0], size[1]
			activate(t, m, "client:pi")
			before := m.draft()
			m.openModal("palette")
			for i, c := range m.paletteCommands() {
				m.modalFocus = i
				assertFrame(t, m)
				frame := layerText(m.View().Content)
				for _, text := range []string{c.label, c.reason, "Search:"} {
					if !strings.Contains(frame, text) {
						t.Fatalf("selected %q loses %q: %s", c.id, text, frame)
					}
				}
			}
			if m.draft() != before || f.planCalls+f.applyCalls+f.keyCalls+f.serviceCalls != 0 {
				t.Fatal("rendering commands changed a draft or invoked the backend")
			}
		})
	}
}

func TestLayerMoreKeepsDestinationAndPurposeTogether(t *testing.T) {
	m, _ := loaded(t)
	m.width, m.height = 40, 12
	m.goTo("Settings")
	m.openModal("more")
	for _, want := range []struct{ name, purpose string }{
		{"Gateway", "Endpoint and hidden credentials"},
		{"Service", "Optional background observation"},
		{"Repair", "Redacted manual repair brief"},
		{"SSH", "Explicit terminal helper handoff"},
		{"Utilities", "Install/update availability"},
		{"Settings", "Display and navigation"},
	} {
		assertFrame(t, m)
		frame := layerText(m.View().Content)
		for _, text := range []string{"> " + want.name, want.purpose} {
			if !strings.Contains(frame, text) {
				t.Fatalf("destination loses %q: %s", text, frame)
			}
		}
		press(m, "tab")
	}
	press(m, "shift+tab")
	if m.modalFocus != 5 || m.screen != "Settings" {
		t.Fatal("More changed traversal or navigated during focus movement")
	}
}

func TestLayerConfirmationNamesScopeAndOutcome(t *testing.T) {
	for _, tc := range []struct{ modal, outcome, scope string }{
		{"quit", "Leave without applying", "Client files stay"},
		{"discard", "Discard codex draft", "Other drafts stay"},
		{"ssh", "Hand off terminal", "Helper owns the terminal until exit"},
		{"service:install", "Install monitor", "Client configuration and credentials"},
		{"service:uninstall", "Uninstall monitor", "Client configuration and credentials"},
	} {
		t.Run(tc.modal, func(t *testing.T) {
			m, f := loaded(t)
			m.width, m.height = 40, 12
			m.openModal(tc.modal)
			before := m.draft()
			for i := 0; i < 2; i++ {
				assertFrame(t, m)
				frame := layerText(m.View().Content)
				for _, want := range []string{tc.outcome, tc.scope, "Back; keep working"} {
					if !strings.Contains(frame, want) {
						t.Fatalf("choice %d loses %q: %s", i, want, frame)
					}
				}
				press(m, "tab")
			}
			if m.modalFocus != 0 || m.draft() != before || f.serviceCalls != 0 || f.commandAction != "" {
				t.Fatal("confirmation rendering changed state")
			}
			press(m, "enter")
			if m.modal != "" || m.draft() != before || f.serviceCalls != 0 || f.commandAction != "" {
				t.Fatal("initial Back performed the operation")
			}
		})
	}
}

func TestLayerEditorsExplainTheirActualCommitBoundary(t *testing.T) {
	for _, tc := range []struct{ field, label, boundary string }{
		{"filter", "Client name", "Enter applies the filter"},
		{"model", "Model ID", "Draft only"},
		{"budget", "Budget in tokens", "Draft only"},
		{"compact", "Compact tokens", "Draft only"},
		{"endpoint", "Endpoint URL", "Local-only endpoint"},
		{"credential", "Environment variable name", "Variable name, not the key"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			m, _ := loaded(t)
			m.startEditor(tc.field, "sample-value")
			frame := layerText(m.View().Content)
			for _, want := range []string{tc.label, tc.boundary, "> sample-value|", "Esc keeps the previous"} {
				if !strings.Contains(frame, want) {
					t.Errorf("editor lacks %q: %s", want, frame)
				}
			}
		})
	}
}

func TestLayerKeyAllControlsKeepLiteralLabelsAndMask(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		m, f := loaded(t)
		m.width, m.height = size[0], size[1]
		m.goTo("Gateway")
		activate(t, m, "key")
		empty := m.View().Content
		m.Update(tea.PasteMsg{Content: "synthetic-A15-private-buffer"})
		if m.View().Content != empty {
			t.Fatal("hidden field reveals credential length or value")
		}
		for _, focus := range []int{0, 3, 2, 1} {
			if m.editor.focus != focus {
				t.Fatal("key reverse traversal changed")
			}
			assertFrame(t, m)
			label := []string{"API key: [hidden]", "[ ] Persist using backend credential storage", "Save key", "Back; drop unsaved key"}[focus]
			if !strings.Contains(layerText(m.View().Content), "> "+label) {
				t.Fatalf("%v focused label lost: %s", size, m.View().Content)
			}
			press(m, "shift+tab")
		}
		press(m, "esc")
		if f.keyCalls != 0 || m.editor != nil {
			t.Fatal("cancel saved or retained the credential editor")
		}
	}
}

func TestLayerPresentationPreservesStateAndSemanticStyles(t *testing.T) {
	m, _ := loaded(t)
	m.options.NoColor = false
	m.options.Theme = "light"
	m.openModal("quit")
	before, draft := m.location(), m.draft()
	focus, modal, query, status := m.modalFocus, m.modal, m.query, m.status
	frame := m.View().Content
	if before != m.location() || draft != m.draft() || focus != m.modalFocus || modal != m.modal || query != m.query || status != m.status {
		t.Fatal("presentation mutated the model")
	}
	if !strings.Contains(frame, "\x1b[") || !strings.Contains(frame, "Confirm: Leave without applying") {
		t.Fatal("confirmation lacks semantic styling and a specific result")
	}
}

func TestLayerEmptySearchAndEveryThemeFit(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		for _, theme := range []string{"dark", "light", "mono"} {
			for _, noColor := range []bool{false, true} {
				m, _ := loaded(t)
				m.width, m.height = size[0], size[1]
				m.options.Theme, m.options.NoColor, m.options.ASCII = theme, noColor, noColor
				for _, modal := range []string{"more", "palette", "quit", "discard", "ssh", "service:install", "service:uninstall"} {
					m.openModal(modal)
					for i := 0; i < 2; i++ {
						assertFrame(t, m)
						press(m, "tab")
					}
					press(m, "esc")
				}
				for _, field := range []string{"filter", "model", "budget", "compact", "endpoint", "credential", "key"} {
					m.startEditor(field, "研究-e\u0301-"+strings.Repeat("長", 60))
					assertFrame(t, m)
					press(m, "esc")
				}
				m.openModal("palette")
				m.Update(tea.PasteMsg{Content: "\x1b[31mno-matching-commands-研究"})
				assertFrame(t, m)
				if !strings.Contains(layerText(m.View().Content), "No commands match.") {
					t.Fatal("empty search lost its recovery instruction")
				}
			}
		}
	}
}

func TestLayerCaretEditsExactUnicodeAndCancelKeepsDraft(t *testing.T) {
	m, _ := loaded(t)
	m.width, m.height = 40, 12
	m.goTo("Models")
	m.focusID("advanced:model")
	origin, draft := m.location(), m.draft()
	value := "exact-模型-e\u0301-" + strings.Repeat("長", 60) + "tail"
	m.startEditor("model", value)
	if !strings.Contains(layerText(m.View().Content), "tail|") {
		t.Fatal("end of the exact Unicode buffer is hidden")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	if !strings.Contains(layerText(m.View().Content), "> |exact-模型-e\u0301-") {
		t.Fatal("Home cannot reach the start of the exact buffer")
	}
	m.Update(tea.PasteMsg{Content: "prefix-\x1b[31m"})
	if !strings.HasPrefix(string(m.editor.value), "prefix-[31mexact-模型-e\u0301-") {
		t.Fatal("paste behavior changed")
	}
	press(m, "esc")
	if m.draft() != draft || m.location() != origin {
		t.Fatal("cancel changed the draft or return focus")
	}
}
