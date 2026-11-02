//go:build visualcapture

package tui

// This fixture is deliberately test-only. It owns no layout, style or renderer.
// New, semantic commands/controls, Update, View and Bubble Tea are production code.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/clankercode/oneline-ccc/internal/core"
)

// The writer observes a completed renderer Write. A changing invisible window
// title requests a flush even when a key leaves the visible cells unchanged.
// The appended OSC marker is ignored by terminal emulators, never a visible row.
type visualOutput struct {
	out  io.Writer
	keys []string
}

var visualTitle = regexp.MustCompile(`\x1b\][02];native-capture:([0-9]+)(?:\x07|\x1b\\)`)

func (w visualOutput) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	if err == nil {
		for _, match := range visualTitle.FindAllSubmatch(p, -1) {
			if string(match[1]) == "0" {
				if _, err = fmt.Fprintf(w.out, "\x1b]777;native-capture:keys=%s\x07", strings.Join(w.keys, ",")); err != nil {
					break
				}
			}
			_, err = fmt.Fprintf(w.out, "\x1b]777;native-capture:%s\x07", match[1])
			if err != nil {
				break
			}
		}
	}
	return n, err
}

type visualModel struct {
	inner       *model
	epoch       int
	keys        []string
	view        tea.View
	initialView tea.View
}

func (m *visualModel) Init() tea.Cmd { return nil } // inspection was drained below
func (m *visualModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if k.String() == "f12" {
			return m, tea.Quit
		} // fixture-only clean exit
		m.keys = append(m.keys, k.String())
		m.epoch++
	}
	_, cmd := m.inner.Update(msg)
	return m, cmd
}
func (m *visualModel) View() tea.View {
	v := m.inner.View()
	m.view = v // exact production View, before the invisible test title
	if m.epoch == 0 {
		m.initialView = v
	}
	v.WindowTitle = fmt.Sprintf("native-capture:%d", m.epoch)
	return v
}

// This asserts semantic state only; expected visual content comes from View.
func visualScene(m *model, scenario, state, variant, theme string, cols, rows int) error {
	screens := map[string]string{"dashboard": "Dashboard", "models": "Models", "review": "Review", "jobs": "Jobs", "gateway": "Gateway", "service": "Service", "repair": "Repair", "ssh": "SSH", "utilities": "Utilities", "settings": "Settings"}
	modals := map[string][2]string{"help": {"Dashboard", "help"}, "palette": {"Dashboard", "palette"}, "discard": {"Models", "discard"}, "quit": {"Models", "quit"}, "ssh-confirm": {"SSH", "ssh"}, "monitor-install": {"Service", "service:install"}, "monitor-uninstall": {"Service", "service:uninstall"}, "more": {"Dashboard", "more"}}
	wantScreen, wantModal, wantEditor := screens[scenario], "", ""
	if target, ok := modals[scenario]; ok {
		wantScreen, wantModal = target[0], target[1]
	}
	if strings.HasPrefix(scenario, "edit-") {
		wantEditor = strings.TrimPrefix(scenario, "edit-")
		if wantEditor == "compaction" {
			wantEditor = "compact"
		}
		wantScreen = "Models"
		if wantEditor == "key" {
			wantScreen = "Gateway"
		}
		if wantEditor == "filter" {
			wantScreen = "Dashboard"
		}
	}
	field := ""
	if m.editor != nil {
		field = m.editor.field
	}
	if wantScreen == "" || m.screen != wantScreen || m.modal != wantModal || field != wantEditor {
		return fmt.Errorf("requested scene %s not reached: screen=%s modal=%s editor=%s", scenario, m.screen, m.modal, field)
	}
	if m.width != cols || m.height != rows || m.options.Theme != theme {
		return fmt.Errorf("requested scene geometry/theme not reached")
	}
	clients := m.snapshot.Clients
	if state == "empty" {
		if len(clients) != 0 {
			return fmt.Errorf("empty fixture state not reached")
		}
	} else {
		if len(clients) == 0 {
			return fmt.Errorf("fixture client missing")
		}
		c := clients[0]
		switch state {
		case "baseline", "unicode", "parity":
			if !c.Installed || !c.Supported {
				return fmt.Errorf("installed supported fixture state not reached")
			}
		case "missing":
			if c.Installed || c.Status != "NOT INSTALLED" {
				return fmt.Errorf("missing fixture state not reached")
			}
		case "unsupported":
			if c.Supported || c.Reason == "" {
				return fmt.Errorf("unsupported fixture state not reached")
			}
		case "drift":
			if c.Status != "DRIFT" {
				return fmt.Errorf("drift fixture state not reached")
			}
		default:
			return fmt.Errorf("unknown fixture state %s", state)
		}
		if state == "baseline" && (c.Name != "Codex" || c.Status != "CONFIGURED") {
			return fmt.Errorf("baseline fixture state not reached")
		}
		if state == "unicode" && c.Name != visualUnicodeName {
			return fmt.Errorf("unicode fixture state not reached: %q", c.Name)
		}
		if state == "parity" {
			if len(clients) != 6 {
				return fmt.Errorf("six-client parity state not reached")
			}
			names := []string{"Codex", "Claude Code", "OpenCode", "Pi", "Grok Build", "Hermes"}
			for i, c := range clients {
				if c.Name != names[i] || c.Installed != (c.ID != "pi") {
					return fmt.Errorf("parity client composition not reached")
				}
			}
			for _, c := range clients[1:] {
				if c.Supported {
					return fmt.Errorf("parity adapter must remain unsupported")
				}
			}
		}
	}
	if scenario == "review" && m.plan == nil {
		return fmt.Errorf("review plan missing")
	}
	switch variant {
	case "baseline":
	case "acknowledged":
		if scenario != "review" || !m.reviewRead || m.reviewPage != len(m.reviewPages())-1 {
			return fmt.Errorf("review not acknowledged on final page")
		}
	case "last-page":
		if scenario == "help" {
			page, err := visualHelpPage(m)
			if err != nil {
				return err
			}
			if page.Number != page.Total {
				return fmt.Errorf("last help page not reached: %+v", page)
			}
		} else if scenario != "review" || m.reviewPage != len(m.reviewPages())-1 {
			return fmt.Errorf("last review page not reached")
		}
	case "no-results":
		if scenario == "palette" {
			if m.query == "" || len(m.paletteCommands()) != 0 {
				return fmt.Errorf("palette zero-results state not reached")
			}
		} else if scenario == "dashboard" {
			if m.filter == "" {
				return fmt.Errorf("filter zero-results query missing")
			}
			for _, c := range clients {
				if m.matches(c.ID, c.Name) {
					return fmt.Errorf("filter zero-results state not reached")
				}
			}
		} else {
			return fmt.Errorf("no-results requires dashboard/palette")
		}
	case "invalid":
		if !visualValidatedEditor(m) || string(m.editor.value) != "invalid-input" {
			return fmt.Errorf("invalid editor value not reached")
		}
		// Ask the production validator on a private copy; never clone its rules.
		// Clone mutable drafts/editor so even a successful probe cannot change m.
		probe := visualClone(m)
		probe.status = ""
		before := probe.draft()
		probe.saveEditor()
		if probe.editor == nil || probe.draft() != before || probe.status == "" || m.status != probe.status {
			return fmt.Errorf("invalid editor validation refusal not reached")
		}
	default:
		return fmt.Errorf("variant %s has no supported semantic outcome on this baseline", variant)
	}
	return nil
}

const visualUnicodeName = "Codex 模型 — café · a deliberately long client name"

type visualPage struct {
	Number int `json:"number"`
	Total  int `json:"total"`
}

// Read the production Help marker, not a copy of its pagination/layout formula.
func visualHelpPage(m *model) (visualPage, error) {
	re := regexp.MustCompile(`Page ([0-9]+) of ([0-9]+)`)
	for _, r := range m.modalRows() {
		if parts := re.FindStringSubmatch(r.text); parts != nil {
			n, _ := strconv.Atoi(parts[1])
			total, _ := strconv.Atoi(parts[2])
			if n > 0 && n <= total {
				return visualPage{n, total}, nil
			}
		}
	}
	return visualPage{}, fmt.Errorf("production help page marker missing")
}

func visualValidatedEditor(m *model) bool {
	return m.editor != nil && (m.editor.field == "budget" || m.editor.field == "compact" || m.editor.field == "endpoint")
}

// Hash the complete comparable desired state, including private fields omitted
// from public JSON, without exporting the dummy endpoint or credential value.
func visualDraftHash(m *model) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%#v", m.draft()))))
}

type visualValidation struct {
	Field    string `json:"field"`
	Value    string `json:"value"`
	Status   string `json:"status"`
	Before   string `json:"before_draft_sha256"`
	After    string `json:"after_draft_sha256"`
	Retained bool   `json:"editor_retained"`
}
type visualSetup struct {
	Actions    []string
	Validation *visualValidation
}

func (s *visualSetup) outcomes(m *model) (map[string]any, error) {
	out := map[string]any{"status": m.status}
	if m.modal == "help" {
		page, err := visualHelpPage(m)
		if err != nil {
			return nil, err
		}
		out["page"] = page
	}
	if m.screen == "Review" && m.plan != nil {
		out["page"] = visualPage{m.reviewPage + 1, len(m.reviewPages())}
		out["review_acknowledged"] = m.reviewRead
	}
	count := 0
	for _, c := range m.snapshot.Clients {
		if m.matches(c.ID, c.Name) {
			count++
		}
	}
	clearLabel := ""
	for _, c := range m.controls() {
		if c.id == "filter:clear" {
			clearLabel = c.label
		}
	}
	out["filter"] = map[string]any{"query": m.filter, "matching_clients": count, "clear_control_label": clearLabel}
	out["palette"] = map[string]any{"query": m.query, "matching_commands": len(m.paletteCommands())}
	if visualValidatedEditor(m) {
		out["editor"] = map[string]any{"field": m.editor.field, "value": string(m.editor.value)}
	}
	if v := s.Validation; v != nil {
		if !visualValidatedEditor(m) || m.editor.field != v.Field || string(m.editor.value) != v.Value || m.status != v.Status || !v.Retained || v.Before != v.After || visualDraftHash(m) != v.After {
			return nil, fmt.Errorf("observed validation refusal/editor/draft outcome changed")
		}
		out["validation"] = v
	}
	return out, nil
}

func visualClone(m *model) *model {
	copy := *m
	copy.drafts = make(map[string]core.Desired)
	for id, d := range m.drafts {
		copy.drafts[id] = d
	}
	if m.editor != nil {
		e := *m.editor
		e.value = append([]rune(nil), m.editor.value...)
		copy.editor = &e
	}
	return &copy
}

// Probe the real model's two candidate key orders on private copies. Saturating
// modal navigation is not invertible in both directions at a boundary. No
// layout/focus index or A3 implementation is assumed. Refuse if neither restores.
func visualProtocol(m *model) ([]string, error) {
	before := m.View().Content
	for _, keys := range [][]string{{"tab", "shift+tab"}, {"shift+tab", "tab"}} {
		probe := visualClone(m)
		commands := false
		for _, key := range keys {
			if press(probe, key) != nil {
				commands = true
			}
		}
		if !commands && probe.View().Content == before && visualDraftHash(probe) == visualDraftHash(m) {
			return keys, nil
		}
	}
	return nil, fmt.Errorf("neither actual Tab pair restores the production scene")
}

func TestVisualProtocolRestoresClampedBoundary(t *testing.T) {
	for _, last := range []bool{false, true} {
		t.Run(fmt.Sprint("last=", last), func(t *testing.T) {
			m, _ := visualPrepare(t, "palette", "baseline", "baseline", "dark", 80, 24)
			if last {
				for range m.paletteCommands() {
					press(m, "down")
				}
			}
			before := m.View().Content
			keys, err := visualProtocol(m)
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range keys {
				press(m, key)
			}
			if m.View().Content != before {
				t.Fatal("actual Tab pair changed production View at clamped modal boundary")
			}
		})
	}
}

func TestVisualSceneRejectsWrongTarget(t *testing.T) {
	m, _ := visualPrepare(t, "dashboard", "parity", "baseline", "dark", 80, 24)
	m.screen = "Jobs"
	if visualScene(m, "dashboard", "parity", "baseline", "dark", 80, 24) == nil {
		t.Fatal("wrong target scene accepted")
	}
}

func TestVisualSceneRejectsWrongEditor(t *testing.T) {
	m, _ := visualPrepare(t, "edit-budget", "baseline", "baseline", "dark", 80, 24)
	m.editor.field = "model"
	if visualScene(m, "edit-budget", "baseline", "baseline", "dark", 80, 24) == nil {
		t.Fatal("wrong editor scene accepted")
	}
}

func TestVisualSceneRejectsWrongFixtureState(t *testing.T) {
	m, _ := visualPrepare(t, "dashboard", "baseline", "baseline", "dark", 80, 24)
	if visualScene(m, "dashboard", "missing", "baseline", "dark", 80, 24) == nil {
		t.Fatal("wrong fixture state accepted")
	}
}

func TestVisualSceneDistinguishingOutcomes(t *testing.T) {
	tests := []struct{ scenario, state, variant string }{
		{"dashboard", "unicode", "baseline"},
		{"help", "baseline", "last-page"},
		{"edit-budget", "baseline", "invalid"},
		{"edit-compaction", "baseline", "invalid"},
		{"edit-endpoint", "baseline", "invalid"},
	}
	for _, tc := range tests {
		t.Run(tc.scenario+"/"+tc.state+"/"+tc.variant, func(t *testing.T) {
			good, _ := visualPrepare(t, tc.scenario, tc.state, tc.variant, "dark", 40, 12)
			if err := visualScene(good, tc.scenario, tc.state, tc.variant, "dark", 40, 12); err != nil {
				t.Fatalf("genuine positive: %v", err)
			}
			wrong, _ := visualPrepare(t, tc.scenario, "baseline", "baseline", "dark", 40, 12)
			if err := visualScene(wrong, tc.scenario, tc.state, tc.variant, "dark", 40, 12); err == nil {
				t.Fatal("same-screen wrong semantic outcome accepted")
			}
		})
	}
}

func TestVisualSceneInvalidRequiresRefusal(t *testing.T) {
	for _, scenario := range []string{"edit-budget", "edit-compaction", "edit-endpoint"} {
		t.Run(scenario, func(t *testing.T) {
			m, _ := visualPrepare(t, scenario, "baseline", "baseline", "dark", 80, 24)
			press(m, "ctrl+u")
			m.Update(tea.PasteMsg{Content: "invalid-input"})
			if visualScene(m, scenario, "baseline", "invalid", "dark", 80, 24) == nil {
				t.Fatal("unsubmitted invalid input accepted without validation refusal")
			}
		})
	}
}

func TestVisualSceneAllDeclaredStatesAndVariants(t *testing.T) {
	for _, state := range []string{"baseline", "empty", "missing", "unsupported", "drift", "unicode", "parity"} {
		t.Run(state, func(t *testing.T) {
			m, setup := visualPrepare(t, "dashboard", state, "baseline", "dark", 120, 40)
			if err := visualScene(m, "dashboard", state, "baseline", "dark", 120, 40); err != nil {
				t.Fatal(err)
			}
			if _, err := setup.outcomes(m); err != nil {
				t.Fatal(err)
			}
			if state != "baseline" {
				wrong, _ := visualPrepare(t, "dashboard", "baseline", "baseline", "dark", 120, 40)
				if visualScene(wrong, "dashboard", state, "baseline", "dark", 120, 40) == nil {
					t.Fatal("baseline accepted as distinguishing client state")
				}
			}
		})
	}
	for _, tc := range []struct{ scenario, variant string }{
		{"dashboard", "no-results"}, {"palette", "no-results"}, {"review", "last-page"}, {"review", "acknowledged"},
	} {
		t.Run(tc.scenario+"/"+tc.variant, func(t *testing.T) {
			m, setup := visualPrepare(t, tc.scenario, "baseline", tc.variant, "dark", 40, 12)
			if err := visualScene(m, tc.scenario, "baseline", tc.variant, "dark", 40, 12); err != nil {
				t.Fatal(err)
			}
			if _, err := setup.outcomes(m); err != nil {
				t.Fatal(err)
			}
			wrong, _ := visualPrepare(t, tc.scenario, "baseline", "baseline", "dark", 40, 12)
			if visualScene(wrong, tc.scenario, "baseline", tc.variant, "dark", 40, 12) == nil {
				t.Fatal("baseline accepted as distinguishing variant")
			}
		})
	}
}

func TestVisualSceneValidationObservationCannotDrift(t *testing.T) {
	m, setup := visualPrepare(t, "edit-budget", "baseline", "invalid", "dark", 80, 24)
	before := m.draft()
	if err := visualScene(m, "edit-budget", "baseline", "invalid", "dark", 80, 24); err != nil {
		t.Fatal(err)
	}
	if m.draft() != before {
		t.Fatal("production validation probe mutated real draft")
	}
	m.changeProfile(core.Defaults())
	if _, err := setup.outcomes(m); err == nil {
		t.Fatal("changed draft accepted after refusal")
	}
}

func visualPrepare(t *testing.T, scenario, state, variant, theme string, cols, rows int) (*model, *visualSetup) {
	t.Helper()
	f := fixture()
	p := core.Defaults()
	p.BaseURL = "https://capture.invalid/v1"
	p.CapacityConfirmed, p.TiersConfirmed = true, true
	p.Budget, p.Compact = 500000, 450000
	f.snapshot.Clients = []core.Client{
		{ID: "codex", Name: "Codex", Installed: true, Supported: true, Managed: true, Version: "fixture-1", Status: "CONFIGURED", Profile: p},
		{ID: "pi", Name: "Pi", Status: "NOT INSTALLED", Supported: true, Profile: p, Reason: "Not installed"},
		{ID: "grok", Name: "Grok Build", Installed: true, Status: "UNVERIFIED", Profile: p, Reason: "Adapter not verified"},
	}
	f.snapshot.Credential = "Session only"
	f.snapshot.Service = core.ServiceInfo{Backend: "Not configured", Status: "Not installed", Detail: "Optional monitor has not been installed"}
	f.snapshot.Receipts = []core.Receipt{{ID: "fixture-receipt-1", ClientID: "codex", Outcome: "Succeeded", Summary: "Saved reviewed profile"}}
	switch state {
	case "baseline":
	case "empty":
		f.snapshot.Clients = nil
		f.snapshot.Receipts = nil
		f.snapshot.Credential = "Missing"
	case "missing":
		f.snapshot.Clients[0].Installed = false
		f.snapshot.Clients[0].Status = "NOT INSTALLED"
	case "unsupported":
		f.snapshot.Clients[0].Supported = false
		f.snapshot.Clients[0].Reason = "Fixture adapter unsupported"
	case "drift":
		f.snapshot.Clients[0].Status = "DRIFT"
	case "unicode":
		f.snapshot.Clients[0].Name = visualUnicodeName
	case "parity":
		// Match the approved reference's client composition without claiming
		// this native build implements adapters other than Codex.
		profile := func(model string, budget int64) core.Profile {
			copy := p
			copy.Model, copy.Budget, copy.Compact = model, budget, compactBudget(budget)
			copy.CapacityConfirmed, copy.TiersConfirmed = false, false
			return copy
		}
		f.snapshot.Clients = append(f.snapshot.Clients[:1], []core.Client{
			{ID: "claude", Name: "Claude Code", Installed: true, Status: "NEEDS SETUP", Reason: "Read only: adapter not verified in this native build", Profile: profile("gpt-6.1-sol", 500000)},
			{ID: "opencode", Name: "OpenCode", Installed: true, Managed: true, Status: "DRIFT", Reason: "Read only: adapter not verified in this native build", Profile: profile("glm-5.3", 353000)},
			{ID: "pi", Name: "Pi", Status: "NOT INSTALLED", Reason: "Not installed; adapter not verified", Profile: profile("glm-5.3-flash", 272000)},
			{ID: "grok", Name: "Grok Build", Installed: true, Status: "UNVERIFIED", Reason: "Read only: adapter not verified in this native build", Profile: profile("gpt-6.1-sol", 272000)},
			{ID: "hermes", Name: "Hermes", Installed: true, Status: "CUSTOM", Reason: "Read only: custom configuration; adapter not verified", Profile: profile("Custom configuration", 0)},
		}...)
	default:
		t.Fatalf("unknown fixture state %q", state)
	}
	m := New(f, Options{Theme: theme, Version: "visual-fixture", Context: context.Background(), NoColor: os.Getenv("ONELINE_CAPTURE_COLOR") == "none", ASCII: os.Getenv("ONELINE_CAPTURE_ASCII") == "1", ReducedMotion: true}).(*model)
	run(t, m, m.Init())
	m.Update(tea.WindowSizeMsg{Width: cols, Height: rows})
	setup := &visualSetup{}
	drain := func(cmd tea.Cmd) {
		if cmd != nil {
			run(t, m, cmd)
		}
	}
	activateID := func(id string) {
		setup.Actions = append(setup.Actions, id)
		for _, c := range m.controls() {
			if c.id == id {
				m.focusID(id)
				drain(press(m, "enter"))
				return
			}
		}
		for _, c := range m.catalog() {
			if c.id == id {
				drain(m.activate(control{id: id, reason: c.reason}))
				return
			}
		}
		t.Fatalf("capture fixture: semantic control/command %q unavailable on %s", id, m.screen)
	}
	nav := func(name string) { activateID("nav:" + name) }
	screens := map[string]string{"dashboard": "Dashboard", "models": "Models", "review": "Review", "jobs": "Jobs", "gateway": "Gateway", "service": "Service", "repair": "Repair", "ssh": "SSH", "utilities": "Utilities", "settings": "Settings"}
	if name, ok := screens[scenario]; ok {
		nav(name)
	}
	switch scenario {
	case "dashboard", "models", "jobs", "gateway", "service", "repair", "ssh", "utilities", "settings":
	case "review":
		nav("Models")
		activateID("model:glm-5.3")
		activateID("capacity")
		activateID("tiers")
		activateID("plan")
		if m.plan == nil {
			t.Fatal("fixture could not produce an exact review")
		}
	case "help", "palette":
		activateID(scenario)
	case "more":
		// A3 adds this semantic ID. On the rejected baseline it is absent;
		// fail closed instead of constructing an imaginary menu here.
		activateID("nav:More")
		if m.modal != "more" {
			t.Fatal("More requires the integrated A3 navigation")
		}
	case "discard", "quit":
		nav("Models")
		activateID("model:glm-5.3")
		if scenario == "discard" {
			activateID("discard")
		} else {
			drain(press(m, "q"))
		}
	case "ssh-confirm":
		nav("SSH")
		activateID("ssh")
	case "monitor-install":
		nav("Service")
		activateID("service:install")
	case "monitor-uninstall":
		nav("Service")
		activateID("service:uninstall")
	case "edit-filter":
		drain(press(m, "/"))
	case "edit-key":
		nav("Gateway")
		activateID("key")
		m.Update(tea.PasteMsg{Content: os.Getenv("ONELINE_CAPTURE_DUMMY_KEY")})
	default:
		if strings.HasPrefix(scenario, "edit-") {
			field := strings.TrimPrefix(scenario, "edit-")
			if field == "compaction" {
				field = "compact"
			}
			nav("Models")
			activateID("advanced:" + field)
			if m.editor == nil {
				t.Fatal("fixture editor is unavailable in this state")
			}
			// Endpoint editor intentionally starts cleared: this production editor
			// can expose its local value. Never export a fixture route in artifacts.
			if field == "endpoint" {
				drain(press(m, "ctrl+u"))
				setup.Actions = append(setup.Actions, "clear-local-endpoint")
			}
		} else {
			t.Fatalf("unknown capture scenario %q", scenario)
		}
	}
	switch variant {
	case "baseline":
	case "no-results":
		if scenario == "palette" {
			m.Update(tea.PasteMsg{Content: "zz-no-fixture-match"})
		} else if scenario == "dashboard" {
			drain(press(m, "/"))
			m.Update(tea.PasteMsg{Content: "zz-no-fixture-match"})
			drain(press(m, "enter"))
		} else {
			t.Fatal("no-results only supports palette/dashboard")
		}
	case "last-page":
		if scenario == "help" {
			for i := 0; i < len(m.catalog()); i++ {
				drain(press(m, "pgdown"))
			}
		} else if scenario == "review" {
			for m.reviewPage < len(m.reviewPages())-1 {
				activateID("review:read")
			}
		} else {
			t.Fatal("last-page only supports help/review")
		}
	case "last-item":
		if scenario != "more" {
			t.Fatal("last-item requires More")
		}
		// Drive the real bounded menu with keys; no focus-index contract.
		for range m.catalog() {
			drain(press(m, "down"))
		}
	case "acknowledged":
		if scenario != "review" {
			t.Fatal("acknowledged requires review")
		}
		readReview(t, m)
	case "invalid":
		if m.editor == nil || (m.editor.field != "budget" && m.editor.field != "compact" && m.editor.field != "endpoint") {
			t.Fatal("invalid requires a validated editor")
		}
		drain(press(m, "ctrl+u"))
		m.Update(tea.PasteMsg{Content: "invalid-input"})
		v := &visualValidation{Field: m.editor.field, Value: string(m.editor.value), Before: visualDraftHash(m)}
		drain(press(m, "enter"))
		v.After, v.Status, v.Retained = visualDraftHash(m), m.status, m.editor != nil
		setup.Validation = v
	default:
		t.Fatalf("unknown variant %q", variant)
	}
	return m, setup
}

func TestVisualCapture(t *testing.T) {
	if os.Getenv("ONELINE_VISUAL_CAPTURE") != "1" {
		t.Skip("guarded: run tests/tui-visual/capture.py")
	}
	cols, err := strconv.Atoi(os.Getenv("ONELINE_CAPTURE_COLS"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := strconv.Atoi(os.Getenv("ONELINE_CAPTURE_ROWS"))
	if err != nil {
		t.Fatal(err)
	}
	m, setup := visualPrepare(t, os.Getenv("ONELINE_CAPTURE_SCENARIO"), os.Getenv("ONELINE_CAPTURE_STATE"), os.Getenv("ONELINE_CAPTURE_VARIANT"), os.Getenv("ONELINE_CAPTURE_THEME"), cols, rows)
	checkScene := func() {
		if err := visualScene(m, os.Getenv("ONELINE_CAPTURE_SCENARIO"), os.Getenv("ONELINE_CAPTURE_STATE"), os.Getenv("ONELINE_CAPTURE_VARIANT"), os.Getenv("ONELINE_CAPTURE_THEME"), cols, rows); err != nil {
			t.Fatal(err)
		}
		if _, err := setup.outcomes(m); err != nil {
			t.Fatal(err)
		}
	}
	checkScene()
	beforeOutcomes, err := setup.outcomes(m)
	if err != nil {
		t.Fatal(err)
	}
	protocol, err := visualProtocol(m)
	if err != nil {
		t.Fatal(err)
	}
	profile := colorprofile.TrueColor
	switch os.Getenv("ONELINE_CAPTURE_COLOR") {
	case "256":
		profile = colorprofile.ANSI256
	case "16":
		profile = colorprofile.ANSI
	case "none":
		profile = colorprofile.NoTTY
	}
	wrapper := &visualModel{inner: m}
	_, err = tea.NewProgram(wrapper, tea.WithInput(os.Stdin), tea.WithOutput(visualOutput{out: os.Stdout, keys: protocol}), tea.WithWindowSize(cols, rows), tea.WithColorProfile(profile), tea.WithEnvironment(os.Environ())).Run()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(wrapper.keys, ",") != strings.Join(protocol, ",") {
		t.Fatalf("unexpected PTY input: %v", wrapper.keys)
	}
	checkScene()
	if wrapper.view.Content != wrapper.initialView.Content {
		t.Fatal("actual PTY key pair changed production View")
	}
	field := ""
	if m.editor != nil {
		field = m.editor.field
	}
	metadata := map[string]any{"screen": m.screen, "modal": m.modal, "editor": field, "selected": m.selected, "pty_keys": wrapper.keys, "setup_actions": setup.Actions, "fixture": "disposable-fakeBackend-v1", "profile": m.draft().Profile, "ascii": m.options.ASCII, "no_color": m.options.NoColor, "clients": m.snapshot.Clients, "credential_provenance": "Injected dummy session key; no real credential access", "public_credential_status": m.snapshot.Credential}
	for _, key := range []string{"scenario", "state", "variant", "theme"} {
		metadata[key] = os.Getenv("ONELINE_CAPTURE_" + strings.ToUpper(key))
	}
	metadata["cols"], metadata["rows"] = cols, rows
	outcomes, err := setup.outcomes(m)
	if err != nil {
		t.Fatal(err)
	}
	metadata["semantic_outcomes"] = outcomes
	metadata["semantic_before"] = beforeOutcomes
	metadata["protocol_keys"] = protocol
	metadata["expected_view"] = map[string]any{"content": wrapper.view.Content, "alt_screen": wrapper.view.AltScreen}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("capture.invalid")) {
		t.Fatal("fixture endpoint in metadata")
	}
	if err := os.WriteFile(os.Getenv("ONELINE_CAPTURE_META"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
