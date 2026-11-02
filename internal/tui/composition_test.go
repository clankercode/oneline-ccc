package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func compositionFixture(t *testing.T) *model {
	t.Helper()
	m, _ := loaded(t)
	p := core.Defaults()
	p.TiersConfirmed = true
	m.snapshot.Clients = nil
	for i, name := range []string{"Codex", "Claude Code", "OpenCode", "Pi", "Grok Build", "Hermes"} {
		id := fmt.Sprintf("client-%d", i)
		if i == 0 {
			id = "codex"
		}
		c := core.Client{ID: id, Name: name, Installed: i != 3, Supported: i == 0, Managed: i == 0 || i == 2, Status: []string{"CONFIGURED", "NEEDS SETUP", "DRIFT", "NOT INSTALLED", "UNVERIFIED", "CUSTOM"}[i], Profile: p}
		m.snapshot.Clients = append(m.snapshot.Clients, c)
		m.drafts[id], m.baselines[id] = desiredClient(c), desiredClient(c)
	}
	m.selected = "codex"
	m.focusID("client:codex")
	return m
}

func TestCompositionMultilineClientsAndAppliedDetail(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 120, 40
	d := m.draft()
	d.Profile.Model = "unsaved-draft-model"
	m.drafts[m.selected] = d
	lines, _, _, _ := m.layout()
	for _, c := range m.snapshot.Clients {
		found := false
		for i, r := range lines {
			if strings.Contains(ansi.Strip(r.text), c.Name) && i+2 < len(lines) {
				near := ansi.Strip(lines[i+1].text + "\n" + lines[i+2].text)
				if strings.Contains(near, c.Profile.Model+" / Standard") && strings.Contains(near, "managed") {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("%s has no readable model/default and management subrows", c.Name)
		}
	}
	frame := ansi.Strip(m.View().Content)
	for _, label := range []string{"Management", "Gateway", "Model / default", "Budget", "Available tiers:", "Auth / API", "Selected:", "Draft"} {
		if !strings.Contains(frame, label) {
			t.Errorf("missing applied detail %q", label)
		}
	}
	if strings.Contains(frame, "unsaved-draft-model") {
		t.Fatal("Dashboard presents unsaved values as applied")
	}
}

func TestCompositionNarrowNavigationAndFooter(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 40, 12
	for _, id := range []string{"nav:Dashboard", "nav:More", "client:codex", "configure", "help", "palette"} {
		m.focusID(id)
		frame := ansi.Strip(m.View().Content)
		for _, name := range []string{"Dashboard", "Models", "Review", "Jobs", "More"} {
			if !strings.Contains(frame, name) {
				t.Errorf("focus %s hides navigation %s", id, name)
			}
		}
		for _, hint := range []string{"F6", "Esc", "? Help", "Ctrl+P"} {
			if !strings.Contains(frame, hint) {
				t.Errorf("focus %s hides footer hint %s", id, hint)
			}
		}
		assertFrame(t, m)
	}
}

func TestCompositionInsetsAndExactFrame(t *testing.T) {
	m := compositionFixture(t)
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		m.width, m.height = size[0], size[1]
		m.focusID("client:codex")
		frame := strings.Split(ansi.Strip(m.View().Content), "\n")
		if len(frame) != m.height {
			t.Errorf("frame has %d rows; want %d", len(frame), m.height)
		}
		for _, line := range frame {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, " ") {
				t.Fatalf("%dx%d lacks shared inset: %q", m.width, m.height, line)
			}
		}
		assertFrame(t, m)
	}
}

func TestCompositionClientFocusShowsConfiguration(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 40, 12
	for _, c := range m.snapshot.Clients {
		m.focusID("client:" + c.ID)
		m.scroll = -1
		frame := ansi.Strip(m.View().Content)
		if !strings.Contains(frame, c.Profile.Model+" / Standard") {
			t.Fatalf("focused %s loses its model/default: %s", c.Name, frame)
		}
		assertFrame(t, m)
	}
}

func TestCompositionLongContentEveryControlAndScroll(t *testing.T) {
	m := compositionFixture(t)
	m.snapshot.Clients[0].Name = strings.Repeat("模型e\u0301", 24)
	m.snapshot.Clients[0].Profile.Model = strings.Repeat("exact-模型-", 20)
	m.snapshot.Clients[0].Reason = strings.Repeat("Read only; inspect local configuration. ", 9)
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		m.width, m.height = size[0], size[1]
		for _, theme := range []string{"dark", "light", "mono"} {
			m.options.Theme = theme
			for _, noColor := range []bool{false, true} {
				m.options.NoColor, m.options.ASCII = noColor, noColor
				for i := range m.controls() {
					m.focus, m.scroll = i, -1
					assertFrame(t, m)
					m.scroll = 10000
					assertFrame(t, m)
				}
			}
		}
	}
}

func TestCompositionPrimaryActionHasSeparateTreatment(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 120, 40
	m.options.NoColor, m.options.ASCII, m.options.Theme = false, false, "dark"
	m.focusID("nav:Dashboard")
	lines, _, _, _ := m.layout()
	var primary, auxiliary string
	for _, r := range lines {
		if strings.Contains(ansi.Strip(r.text), "Configure Codex") {
			primary = r.text
		}
		if strings.Contains(ansi.Strip(r.text), "Continue without monitor") {
			auxiliary = r.text
		}
	}
	if primary == "" || auxiliary == "" {
		t.Fatal("primary or auxiliary action disappeared")
	}
	if !strings.Contains(primary, "\x1b[") || primary == ansi.Strip(primary) {
		t.Fatal("primary action has no visual treatment")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "Optional monitor") {
		t.Fatal("auxiliary monitor choices are not grouped separately")
	}
}

func TestCompositionEditorFooterWrapsWithoutHidingCancel(t *testing.T) {
	m, _ := loaded(t)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	m.startEditor("key", "")
	for i := 0; i < 4; i++ {
		m.editor.focus = i
		frame := ansi.Strip(m.View().Content)
		if !strings.Contains(frame, "Esc cancel") {
			t.Fatalf("key option %d hides cancel hint", i)
		}
		_, _, _, available := m.layout()
		if available < 1 || available > m.height-5 {
			t.Fatalf("invalid composed viewport %d", available)
		}
		assertFrame(t, m)
	}
}

func TestCompositionHelpRetainsScopeWhileCloseFollowsFocus(t *testing.T) {
	m, _ := loaded(t)
	m.width, m.height = 40, 12
	m.openModal("help")
	_, last := m.helpPagination()
	for page := 0; page <= last; page++ {
		m.modalFocus = page
		frame := ansi.Strip(m.View().Content)
		if !strings.Contains(frame, fmt.Sprintf("Shortcut help | Page %d of %d", page+1, last+1)) {
			t.Fatalf("page %d loses help scope: %s", page, frame)
		}
		assertFrame(t, m)
	}
}

func TestCompositionDisabledActionKeepsReasonAndNextStepVisible(t *testing.T) {
	m := compositionFixture(t)
	m.selected = "client-3"
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 12}} {
		m.width, m.height = size[0], size[1]
		for _, id := range []string{"plan", "launch"} {
			m.focusID(id)
			m.scroll = -1
			frame := ansi.Strip(m.View().Content)
			if !strings.Contains(frame, "Unavailable:") || !strings.Contains(frame, "Next:") {
				t.Fatalf("%dx%d focused %s loses reason/next step: %s", m.width, m.height, id, frame)
			}
			assertFrame(t, m)
		}
	}
}

func TestCompositionLongClientSummaryKeepsTierAndManagementNearby(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 40, 12
	m.snapshot.Clients[0].Name = strings.Repeat("長い名前", 20)
	m.snapshot.Clients[0].Profile.Model = strings.Repeat("exact-模型-", 20)
	frame := ansi.Strip(m.View().Content)
	for _, s := range []string{"Standard", "managed", "Selected"} {
		if !strings.Contains(frame, s) {
			t.Fatalf("long client hides %q near focus: %s", s, frame)
		}
	}
	assertFrame(t, m)
}

func TestCompositionReviewAckRetainsWholePageViewport(t *testing.T) {
	m, _ := loaded(t)
	m.plan = &core.Plan{ID: strings.Repeat("plan-", 20), Revision: strings.Repeat("revision-", 20), ClientID: "codex", Desired: m.draft(), Changes: []core.Change{{Field: "model", Before: strings.Repeat("before_", 30), After: strings.Repeat("after_", 30)}}}
	m.goTo("Review")
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}} {
		m.width, m.height = size[0], size[1]
		m.focusID("review:read")
		m.scroll = -1
		for page := range m.reviewPages() {
			m.reviewPage = page
			_, _, start, _ := m.layout()
			if start != 0 {
				t.Fatalf("%dx%d page %d acknowledgment hides first review rows (start %d)", m.width, m.height, page, start)
			}
			assertFrame(t, m)
		}
	}
}

func TestCompositionManualPagingDoesNotSkipDetailBehindPinnedFocus(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 40, 12
	var words []string
	for i := 0; i < 75; i++ {
		words = append(words, fmt.Sprintf("notice%02d", i))
	}
	m.snapshot.Clients[0].Reason = strings.Join(words, " ")
	m.scroll = 0
	var frames string
	for i := 0; i < 100; i++ {
		frames += ansi.Strip(m.View().Content) + "\n"
		_, _, before, _ := m.layout()
		press(m, "pgdown")
		_, _, after, _ := m.layout()
		if before == after {
			break
		}
	}
	for _, word := range words {
		if !strings.Contains(frames, word) {
			t.Fatalf("manual paging skipped %q behind pinned focus", word)
		}
	}
	assertFrame(t, m)
}

func TestCompositionMultilineStatusStaysInItsShellRow(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 40, 12
	m.status = "Inspection failed:\n" + strings.Repeat("very long evidence\n", 12)
	assertFrame(t, m)
	frame := strings.Split(ansi.Strip(m.View().Content), "\n")
	if len(frame) != m.height || !strings.Contains(frame[2], "Inspection failed:") {
		t.Fatal("multiline status displaced composed frame rows")
	}
}

// Use the renderer's actual ANSI cell parser: visible words alone cannot prove
// that a background surface survives nested heading/status resets.
func TestCompositionSurfaceCoversNestedTextAndTrailingCells(t *testing.T) {
	m := compositionFixture(t)
	m.options.NoColor = false
	for _, theme := range []string{"dark", "light", "mono"} {
		m.options.Theme = theme
		for _, text := range []string{m.shellHeader()[0], m.shellHeader()[2], m.selectionStyle(m.heading("Codex")+"  "+m.statusText("CONFIGURED"), m.width), m.focusStyle(m.heading("Codex")+"  "+m.statusText("CONFIGURED"), m.width)} {
			cells := uv.NewStyledString(text).Lines(ansi.GraphemeWidth)[0]
			if len(cells) != m.width || cells[0].Style.Bg == nil {
				t.Fatal("missing surface cells")
			}
			r, g, b, a := cells[0].Style.Bg.RGBA()
			for i, c := range cells {
				if c.Style.Bg == nil {
					t.Fatalf("%s nested text reset the surface at column %d", theme, i)
				}
				r2, g2, b2, a2 := c.Style.Bg.RGBA()
				if r != r2 || g != g2 || b != b2 || a != a2 {
					t.Fatalf("%s surface changes at column %d", theme, i)
				}
			}
		}
	}
}

func TestCompositionSmallDashboardStartsWithClientScope(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 40, 12
	frame := strings.Split(ansi.Strip(m.View().Content), "\n")
	if !strings.Contains(frame[3], "Codex") {
		t.Fatalf("small Dashboard starts without its selected client: %q", frame[3])
	}
	assertFrame(t, m)
}

func TestCompositionActiveTabDiffersFromSelectedAndFocus(t *testing.T) {
	m := compositionFixture(t)
	m.options.NoColor, m.options.ASCII, m.options.Theme = false, false, "dark"
	cells := uv.NewStyledString(m.shellHeader()[1]).Lines(ansi.GraphemeWidth)[0]
	selected := uv.NewStyledString(m.selectionStyle("Selected", m.width)).Lines(ansi.GraphemeWidth)[0]
	if cells[3].Style.Bg == nil || selected[3].Style.Bg == nil {
		t.Fatal("missing active or selected surface")
	}
	r, g, b, a := cells[3].Style.Bg.RGBA()
	r2, g2, b2, a2 := selected[3].Style.Bg.RGBA()
	if r == r2 && g == g2 && b == b2 && a == a2 {
		t.Fatal("current section uses the same modest surface as selection/focus")
	}
	m.options.NoColor = true
	if !strings.Contains(m.View().Content, "[Dashboard]") {
		t.Fatal("no-color current section lacks a structural cue")
	}
	assertFrame(t, m)
}

func TestCompositionNoticeMatchesSelectedClientState(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 120, 40
	for _, state := range []struct{ name, want string }{{"configured", "Configured / no pending draft"}, {"draft", "Draft waiting for review"}, {"setup", "Ready to configure"}, {"readonly", "Inspection only"}} {
		m.selected = "codex"
		c := &m.snapshot.Clients[0]
		c.Managed = true
		m.drafts[c.ID] = m.baselines[c.ID]
		switch state.name {
		case "draft":
			d := m.draft()
			d.Profile.Budget++
			m.drafts[c.ID] = d
		case "setup":
			c.Managed = false
			m.baselines[c.ID] = desiredClient(*c)
			m.drafts[c.ID] = m.baselines[c.ID]
		case "readonly":
			m.selected = "client-3"
		}
		frame := ansi.Strip(m.View().Content)
		if !strings.Contains(frame, state.want) {
			t.Fatalf("%s state lacks %q", state.name, state.want)
		}
	}
}

func TestCompositionUnicodeModelTierAndClientActionLabels(t *testing.T) {
	m := compositionFixture(t)
	m.options.NoColor, m.options.ASCII, m.options.Theme = false, false, "dark"
	m.width, m.height = 120, 40
	frame := ansi.Strip(m.View().Content)
	for _, label := range []string{"gpt-6.1-sol · Standard", "Configure Codex", "Review Codex draft", "Launch Codex", "Available tiers:"} {
		if !strings.Contains(frame, label) {
			t.Errorf("missing preferred native label %q", label)
		}
	}
	assertFrame(t, m)
}

func TestCompositionCompactDefaultShowsSummaryAndPrimaryActions(t *testing.T) {
	m := compositionFixture(t)
	for _, dirty := range []bool{false, true} {
		m.drafts["codex"] = m.baselines["codex"]
		if dirty {
			d := m.draft()
			d.Profile.Budget++
			m.drafts["codex"] = d
		}
		for _, size := range [][2]int{{80, 24}, {60, 20}, {40, 12}} {
			m.width, m.height = size[0], size[1]
			m.focusID("client:codex")
			m.scroll = -1
			frame := ansi.Strip(m.View().Content)
			for _, value := range []string{"Codex", "gpt-6.1-sol / Standard", "Configure Codex", "Review Codex draft", "Launch Codex"} {
				if !strings.Contains(frame, value) {
					t.Errorf("%dx%d dirty=%t compact view hides %q", m.width, m.height, dirty, value)
				}
			}
			if dirty && !strings.Contains(frame, "Draft") {
				t.Fatal("compact summary hides pending draft")
			}
			assertFrame(t, m)
		}
	}
}

func TestCompositionCompactWindowFollowsLocalArrowsWithoutStaging(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 40, 12
	before := make(map[string]core.Desired)
	for id, d := range m.drafts {
		before[id] = d
	}
	for _, c := range m.snapshot.Clients {
		frame := ansi.Strip(m.View().Content)
		if m.selected != c.ID || !strings.Contains(frame, c.Name) {
			t.Fatalf("compact client window did not follow %s", c.Name)
		}
		assertFrame(t, m)
		press(m, "down")
	}
	for id, d := range before {
		if m.drafts[id] != d {
			t.Fatalf("compact traversal staged %s", id)
		}
	}
}

func TestCompositionFooterSeparatesCurrentRegionFromF6Action(t *testing.T) {
	m := compositionFixture(t)
	m.width, m.height = 40, 12
	frame := ansi.Strip(m.View().Content)
	if !strings.Contains(frame, "In Clients") || !strings.Contains(frame, "F6 next") {
		t.Fatal("current region reads as the target of F6")
	}
}

func TestCompositionTinyDraftSurvivesFocusSelectionSeparation(t *testing.T) {
	for _, long := range []bool{false, true} {
		t.Run(fmt.Sprintf("unicode=%t", long), func(t *testing.T) {
			m := compositionFixture(t)
			m.width, m.height = 40, 12
			m.options.ASCII, m.options.NoColor = false, false
			if long {
				m.snapshot.Clients[0].Name = strings.Repeat("模型e\u0301", 24)
				m.snapshot.Clients[0].Profile.Model = strings.Repeat("exact-模型-", 20)
				m.drafts["codex"] = desiredClient(m.snapshot.Clients[0])
				m.baselines["codex"] = m.drafts["codex"]
			}
			before := make(map[string]core.Desired)
			for id, d := range m.drafts {
				before[id] = d
			}
			press(m, "space")
			press(m, "down")
			press(m, "shift+tab")
			want := before["codex"]
			want.Enabled = !want.Enabled
			if m.selected != "client-1" || m.focused().id != "client:codex" || m.drafts["codex"] != want {
				t.Fatal("three-key journey lost draft, selection or focus identity")
			}
			for id, d := range before {
				if id != "codex" && m.drafts[id] != d {
					t.Fatalf("traversal staged another client: %s", id)
				}
			}
			frame := ansi.Strip(m.View().Content)
			for _, cue := range []string{"Draft: off", "Applied: on", "Selected: Claude Code", "Configure Claude Code", "Review Claude Code draft", " · Standard"} {
				if !strings.Contains(frame, cue) {
					t.Errorf("tiny focused non-selected client hides %q:\n%s", cue, frame)
				}
			}
			if !long && !strings.Contains(frame, "gpt-6.1-sol · Standard") {
				t.Error("short exact model/default adjacency lost")
			}
			assertFrame(t, m)
			t.Logf("three-key frame:\n%s", frame)
		})
	}
}

func TestCompositionTinyMatchingFilterContextAndClear(t *testing.T) {
	for i, query := range []string{"Codex", strings.Repeat("模型e\u0301", 24)} {
		t.Run(fmt.Sprintf("query=%d", i), func(t *testing.T) {
			m := compositionFixture(t)
			m.width, m.height = 40, 12
			m.snapshot.Clients[0].Name = query
			before := make(map[string]core.Desired)
			for id, d := range m.drafts {
				before[id] = d
			}
			press(m, "/")
			m.Update(tea.PasteMsg{Content: query})
			press(m, "enter")
			frame := ansi.Strip(m.View().Content)
			for _, cue := range []string{"Filter: ", "/ edit/clear", "Configure ", "Review ", "Launch ", "Selected 1/1"} {
				if !strings.Contains(frame, cue) {
					t.Errorf("matching tiny filter hides %q:\n%s", cue, frame)
				}
			}
			if m.filter != query || !strings.Contains(frame, "Filter: "+string([]rune(query)[:1])) {
				t.Error("matching filter value/context lost")
			}
			assertFrame(t, m)
			t.Logf("matching-filter frame:\n%s", frame)
			press(m, "/")
			m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
			press(m, "enter")
			frame = ansi.Strip(m.View().Content)
			if m.filter != "" || strings.Contains(frame, "Filter:") || !strings.Contains(frame, "Selected 1/6") {
				t.Fatalf("clear did not restore full client context:\n%s", frame)
			}
			for id, d := range before {
				if m.drafts[id] != d {
					t.Fatalf("filter editing staged %s", id)
				}
			}
			assertFrame(t, m)
		})
	}
}
