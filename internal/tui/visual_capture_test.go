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
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
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
	case "inline-model", "inline-budget", "inline-tier", "advanced-expanded", "advanced-collapsed":
		if scenario != "models" {
			return fmt.Errorf("Models variant requires Models")
		}
		o, err := visualModels(m)
		if err != nil {
			return err
		}
		if o["selected_id"] != "codex" {
			return fmt.Errorf("Models selected identity not reached")
		}
		if strings.HasPrefix(variant, "inline-") {
			field := strings.TrimPrefix(variant, "inline-")
			if o["chooser"] != field || modelsLeafPanelForCapture(m.focused().id) != field {
				return fmt.Errorf("Models inline chooser/focus not reached")
			}
		} else if o["chooser"] != "" || o["advanced"] != (variant == "advanced-expanded") {
			return fmt.Errorf("Models Advanced disclosure not reached")
		}
	case "baseline":
		if scenario == "more" {
			o, err := visualMore(m)
			if err != nil {
				return err
			}
			if o.FocusedID != o.Destinations[0].ID {
				return fmt.Errorf("first More destination not focused")
			}
		}
	case "last-item":
		if scenario != "more" {
			return fmt.Errorf("last-item requires More")
		}
		o, err := visualMore(m)
		if err != nil {
			return err
		}
		if o.FocusedID != o.Destinations[len(o.Destinations)-1].ID {
			return fmt.Errorf("last More destination not focused")
		}
	case "pending-focus":
		if scenario != "dashboard" || m.focused().id != "client:codex" || m.selected != "claude" ||
			m.drafts["codex"] == m.baselines["codex"] || m.drafts["codex"].Enabled == m.baselines["codex"].Enabled ||
			m.drafts["claude"] != m.baselines["claude"] {
			return fmt.Errorf("pending Codex focus / selected Claude draft difference not reached")
		}
	case "filter-match", "filter-unicode":
		query := "claude"
		id := "claude"
		if variant == "filter-unicode" {
			query, id = visualUnicodeName, "codex"
		}
		matches := 0
		for _, c := range clients {
			if m.matches(c.ID, c.Name) {
				matches++
				if c.ID != id {
					return fmt.Errorf("unexpected matching filter client")
				}
			}
		}
		if scenario != "dashboard" || m.filter != query || matches != 1 {
			return fmt.Errorf("matching active filter state not reached")
		}
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
type visualDestination struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type visualMoreState struct {
	Destinations []visualDestination `json:"destinations"`
	FocusedID    string              `json:"focused_id"`
	FocusedLabel string              `json:"focused_label"`
	EnterScreen  string              `json:"enter_screen"`
}

// Inspect production rows and their catalog identities, then exercise Enter on
// an isolated model. No menu indices, labels, or rendered rows are synthesized.
func visualMore(m *model) (visualMoreState, error) {
	var out visualMoreState
	if m.modal != "more" {
		return out, fmt.Errorf("More modal missing")
	}
	for _, r := range m.modalRows() {
		label := strings.TrimSpace(strings.TrimPrefix(r.text, "> "))
		for _, c := range m.catalog() {
			if strings.HasPrefix(c.id, "nav:") && c.id != "nav:More" && c.label == label {
				out.Destinations = append(out.Destinations, visualDestination{c.id, c.label})
				if r.focus {
					if out.FocusedID != "" {
						return out, fmt.Errorf("multiple focused More destinations")
					}
					out.FocusedID, out.FocusedLabel = c.id, c.label
				}
				break
			}
		}
	}
	if len(out.Destinations) == 0 || out.FocusedID == "" {
		return out, fmt.Errorf("More destination rows/catalog focus missing")
	}
	probe := visualClone(m)
	if press(probe, "enter") != nil || probe.modal != "" || "nav:"+probe.screen != out.FocusedID {
		return out, fmt.Errorf("More Enter destination disagrees with focused production row")
	}
	out.EnterScreen = probe.screen
	return out, nil
}

// Reflection keeps the guarded tool compilable on pre-selector production.
// Only observed private scalar form state is read; no production state is set.
func modelsLeafPanelForCapture(id string) string {
	for _, prefix := range []string{"model:", "budget:", "tier:"} {
		if strings.HasPrefix(id, prefix) {
			return strings.TrimSuffix(prefix, ":")
		}
	}
	if strings.HasPrefix(id, "discovered:") {
		return "model"
	}
	return ""
}

func visualModels(m *model) (map[string]any, error) {
	forms := reflect.ValueOf(m).Elem().FieldByName("modelsForms")
	if !forms.IsValid() {
		return nil, fmt.Errorf("Models inline selectors require candidate production source")
	}
	form := forms.MapIndex(reflect.ValueOf(m.selected))
	chooser, advanced := "", false
	if form.IsValid() {
		chooser = form.FieldByName("chooser").String()
		advanced = form.FieldByName("advanced").Bool()
	}
	lines, _, _, _ := m.layout()
	body := make([]string, 0, len(lines))
	for _, r := range lines {
		body = append(body, strings.TrimRight(ansi.Strip(m.bodyLine(r.text)), " "))
	}
	return map[string]any{"chooser": chooser, "advanced": advanced, "focused_id": m.focused().id, "focused_row": ansi.Strip(m.controlRow(m.focused(), m.focus).text), "selected_id": m.selected, "draft_sha256": visualDraftHash(m), "model": m.draft().Profile.Model, "budget": m.draft().Profile.Budget, "tier": m.draft().Profile.Tier, "body_rows": body, "header_rows": len(m.shellHeader()), "footer_rows": len(m.shellFooter("")), "padding": strings.TrimRight(ansi.Strip(m.bodyLine("")), " ")}, nil
}

// Observe the actual composed layout and project client identities through the
// production clientRows first row. No capture-owned row/layout is rendered.
func visualDashboardProjection(m *model) map[string]any {
	lines, _, _, _ := m.layout()
	body := make([]map[string]any, 0, len(lines))
	for _, r := range lines {
		body = append(body, map[string]any{"text": strings.TrimRight(ansi.Strip(m.bodyLine(r.text)), " "), "focused": r.focus, "selected": r.selected})
	}
	controls := make([]map[string]any, 0)
	for i, c := range m.controls() {
		id, ok := strings.CutPrefix(c.id, "client:")
		if !ok {
			continue
		}
		for _, client := range m.snapshot.Clients {
			if client.ID != id {
				continue
			}
			rows := m.clientRows(client, i, m.contentWidth())
			first := strings.TrimRight(ansi.Strip(m.bodyLine(rows[0].text)), " ")
			indices := make([]int, 0)
			for n, r := range body {
				if r["text"] == first {
					indices = append(indices, n)
				}
			}
			controls = append(controls, map[string]any{"id": c.id, "full_name": client.Name, "first_row": first, "body_indices": indices, "focused": i == m.focus, "selected": id == m.selected})
		}
	}
	return map[string]any{"kind": "composed-dashboard-body-v1", "body_rows": body, "client_projections": controls, "header_rows": len(m.shellHeader()), "footer_rows": len(m.shellFooter("")), "padding": strings.TrimRight(ansi.Strip(m.bodyLine("")), " ")}
}

func visualDashboard(m *model) map[string]any {
	clients := make([]map[string]any, 0, len(m.snapshot.Clients))
	for _, c := range m.snapshot.Clients {
		d, b := m.drafts[c.ID], m.baselines[c.ID]
		hash := func(v core.Desired) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%#v", v)))) }
		clients = append(clients, map[string]any{"id": c.ID, "matching": m.matches(c.ID, c.Name), "dirty": d != b,
			"enabled": d.Enabled, "baseline_enabled": b.Enabled, "draft_sha256": hash(d), "baseline_sha256": hash(b)})
	}
	out := map[string]any{"focused_id": m.focused().id, "selected_id": m.selected, "filter": m.filter, "clients": clients}
	if m.screen == "Dashboard" && m.modal == "" && m.editor == nil {
		controls := make([]map[string]any, 0)
		for i, c := range m.controls() {
			if !strings.HasPrefix(c.id, "client:") {
				continue
			}
			r := m.controlRow(c, i)
			controls = append(controls, map[string]any{"id": c.id, "label": c.label, "row_text": ansi.Strip(r.text), "focused": r.focus, "selected": r.selected})
		}
		out["production_client_controls"] = controls
		all := visualClone(m)
		all.filter = ""
		inventory := make([]map[string]any, 0)
		for i, c := range all.controls() {
			if !strings.HasPrefix(c.id, "client:") {
				continue
			}
			r := all.controlRow(c, i)
			inventory = append(inventory, map[string]any{"id": c.id, "label": c.label, "row_text": ansi.Strip(r.text)})
		}
		out["unfiltered_production_client_controls"] = inventory
		out["production_view"] = m.View().Content
		out["production_projection"] = visualDashboardProjection(m)
	}
	return out
}

type visualSetup struct {
	Actions      []string
	Preparation  map[string]any
	Validation   *visualValidation
	ModelsBefore string
}

func (s *visualSetup) outcomes(m *model) (map[string]any, error) {
	out := map[string]any{"status": m.status}
	if s.ModelsBefore != "" {
		o, err := visualModels(m)
		if err != nil {
			return nil, err
		}
		if visualDraftHash(m) != s.ModelsBefore {
			return nil, fmt.Errorf("Models fixture navigation changed draft")
		}
		o["before_draft_sha256"] = s.ModelsBefore
		out["models"] = o
	}
	if m.modal == "more" {
		more, err := visualMore(m)
		if err != nil {
			return nil, err
		}
		out["more"] = more
	}
	if s.Preparation != nil {
		out["dashboard"] = visualDashboard(m)
	}
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
	copy.locations = make(map[string]location)
	for k, v := range m.locations {
		copy.locations[k] = v
	}
	copy.regionFocus = make(map[string]string)
	for k, v := range m.regionFocus {
		copy.regionFocus[k] = v
	}
	copy.history = append([]navigationFrame(nil), m.history...)
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
	beforeDashboard, _ := json.Marshal(visualDashboard(m))
	for _, keys := range [][]string{{"tab", "shift+tab"}, {"shift+tab", "tab"}} {
		probe := visualClone(m)
		commands := false
		for _, key := range keys {
			if press(probe, key) != nil {
				commands = true
			}
		}
		afterDashboard, _ := json.Marshal(visualDashboard(probe))
		if !commands && probe.View().Content == before && bytes.Equal(beforeDashboard, afterDashboard) {
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

func TestVisualMoreBoundarySemantics(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {80, 24}} {
		m, _ := visualPrepare(t, "more", "baseline", "baseline", "dark", size[0], size[1])
		if err := visualScene(m, "more", "baseline", "baseline", "dark", size[0], size[1]); err != nil {
			t.Fatal(err)
		}
		press(m, "down")
		if visualScene(m, "more", "baseline", "baseline", "dark", size[0], size[1]) == nil {
			t.Fatal("nonfirst More destination accepted as baseline")
		}
		last, _ := visualPrepare(t, "more", "baseline", "last-item", "dark", size[0], size[1])
		if err := visualScene(last, "more", "baseline", "last-item", "dark", size[0], size[1]); err != nil {
			t.Fatal(err)
		}
		press(last, "up")
		if visualScene(last, "more", "baseline", "last-item", "dark", size[0], size[1]) == nil {
			t.Fatal("nonlast More destination accepted as last-item")
		}
	}
}

func TestVisualPreparedDashboardSemantics(t *testing.T) {
	for _, tc := range []struct{ state, variant, query string }{
		{"parity", "pending-focus", ""}, {"parity", "filter-match", "claude"}, {"unicode", "filter-unicode", visualUnicodeName},
	} {
		t.Run(tc.variant, func(t *testing.T) {
			m, _ := visualPrepare(t, "dashboard", tc.state, "baseline", "dark", 40, 12)
			if tc.variant == "pending-focus" {
				m.focusID("client:codex")
				press(m, " ")
				press(m, "down")
				press(m, "shift+tab")
			} else {
				press(m, "/")
				m.Update(tea.PasteMsg{Content: tc.query})
				press(m, "enter")
			}
			if err := visualScene(m, "dashboard", tc.state, tc.variant, "dark", 40, 12); err != nil {
				t.Fatalf("genuine prepared state: %v", err)
			}
			if tc.variant == "pending-focus" {
				m.selected = "codex"
			} else {
				m.filter = "zz-no-fixture-match"
			}
			if visualScene(m, "dashboard", tc.state, tc.variant, "dark", 40, 12) == nil {
				t.Fatal("incorrect prepared state accepted")
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
		// This is an explicit capable fixture, not host-platform access proof.
		f.sshAvailable = true
		nav("SSH")
		activateID("ssh")
	case "monitor-install":
		nav("Service")
		activateID("service:install")
	case "monitor-uninstall":
		f.snapshot.Service.Status = "Stopped"
		m.snapshot.Service = f.snapshot.Service
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
	case "inline-model", "inline-budget", "inline-tier", "advanced-expanded", "advanced-collapsed":
		if scenario != "models" {
			t.Fatal("Models variant requires Models")
		}
		setup.ModelsBefore = visualDraftHash(m)
		if _, err := visualModels(m); err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(variant, "inline-") {
			activateID("field:" + strings.TrimPrefix(variant, "inline-"))
		} else {
			activateID("models:advanced")
			if variant == "advanced-collapsed" {
				activateID("models:advanced")
			}
		}
		setup.Actions = append(setup.Actions, "prepared-Models-disclosure; direct Update, not PTY journey")
	case "baseline":
	case "pending-focus", "filter-match", "filter-unicode":
		if scenario != "dashboard" {
			t.Fatal("prepared variants require Dashboard")
		}
		before := visualDashboard(m)
		keys := []string{}
		query := ""
		if variant == "pending-focus" {
			if state != "parity" {
				t.Fatal("pending-focus requires parity clients including Claude")
			}
			m.focusID("client:codex")
			keys = []string{"space", "down", "shift+tab"}
			drain(press(m, " "))
			drain(press(m, "down"))
			drain(press(m, "shift+tab"))
		} else {
			if variant == "filter-match" && state != "parity" {
				t.Fatal("filter-match requires parity clients")
			}
			if variant == "filter-unicode" && state != "unicode" {
				t.Fatal("filter-unicode requires Unicode client")
			}
			query = "claude"
			if variant == "filter-unicode" {
				query = visualUnicodeName
			}
			keys = []string{"/", "paste", "enter"}
			drain(press(m, "/"))
			m.Update(tea.PasteMsg{Content: query})
			drain(press(m, "enter"))
		}
		setup.Preparation = map[string]any{"kind": "prepared-fixture-state", "transport": "direct production Update; setup keys are not PTY-delivered", "keys": keys, "query": query, "before": before, "after": visualDashboard(m)}
		setup.Actions = append(setup.Actions, "prepare:"+variant)
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
		// Derive the last destination from real rows/catalog. Stop before a
		// wrapping implementation returns to first; never assume clamping.
		more, err := visualMore(m)
		if err != nil {
			t.Fatal(err)
		}
		target := more.Destinations[len(more.Destinations)-1].ID
		for range more.Destinations {
			observed, err := visualMore(m)
			if err != nil {
				t.Fatal(err)
			}
			if observed.FocusedID == target {
				break
			}
			drain(press(m, "down"))
		}
		setup.Actions = append(setup.Actions, "prepare:More-last-destination")
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
	if setup.Preparation != nil {
		metadata["preparation"] = setup.Preparation
	}
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
