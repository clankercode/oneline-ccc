package tui

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
)

type fakeBackend struct {
	snapshot                                                                       core.Snapshot
	inspectCalls, planCalls, applyCalls, keyCalls, serviceCalls                    int
	planned                                                                        core.Desired
	plannedID, appliedID, appliedRevision, serviceAction, commandAction, commandID string
	planErr, applyErr                                                              error
	persistent                                                                     bool
	sshAvailable                                                                   bool
}

func (f *fakeBackend) SSHSetupUnavailableReason() string {
	if f.sshAvailable {
		return ""
	}
	return "SSH handoff is unavailable in this fixture"
}

func fixture() *fakeBackend {
	p := core.Defaults()
	p.BaseURL = "https://private.invalid/v1"
	p.CapacityConfirmed = true
	p.TiersConfirmed = true
	return &fakeBackend{snapshot: core.Snapshot{Clients: []core.Client{
		{ID: "codex", Name: "Codex", Installed: true, Supported: true, Profile: p, Status: "Unverified", Executable: "/private/executable"},
		{ID: "pi", Name: "Pi", Installed: false, Supported: true, Reason: "Not installed", Profile: p},
	}}}
}
func (f *fakeBackend) Inspect(context.Context) (core.Snapshot, error) {
	f.inspectCalls++
	return f.snapshot, nil
}
func (f *fakeBackend) Plan(_ context.Context, id string, d core.Desired) (core.Plan, error) {
	f.planCalls++
	f.planned = d
	f.plannedID = id
	return core.Plan{ID: "opaque-plan", Revision: "exact-revision", ClientID: id, Desired: d, Changes: []core.Change{{Field: "model", Before: f.snapshot.Clients[0].Profile.Model, After: d.Profile.Model}}}, f.planErr
}
func (f *fakeBackend) Apply(_ context.Context, id, revision string) (core.Receipt, error) {
	f.applyCalls++
	f.appliedID = id
	f.appliedRevision = revision
	if f.applyErr != nil {
		return core.Receipt{}, f.applyErr
	}
	f.snapshot.Clients[0].Managed = f.planned.Enabled
	f.snapshot.Clients[0].Profile = f.planned.Profile
	return core.Receipt{ID: "receipt-1", ClientID: "codex", Outcome: "Succeeded", Summary: "Saved reviewed profile"}, nil
}
func (f *fakeBackend) Command(_ context.Context, a, id string) (*exec.Cmd, error) {
	f.commandAction = a
	f.commandID = id
	return exec.Command("true"), nil
}
func (f *fakeBackend) SaveKey(_ context.Context, _ string, p bool) error {
	f.keyCalls++
	f.persistent = p
	return nil
}
func (f *fakeBackend) Discover(context.Context, string) ([]core.ModelInfo, error) {
	return []core.ModelInfo{{ID: "exact-模型", Context: 500000, Source: "advertised"}}, nil
}
func (f *fakeBackend) Service(_ context.Context, a string) (core.ServiceInfo, error) {
	f.serviceCalls++
	f.serviceAction = a
	return core.ServiceInfo{Status: "Observed"}, nil
}
func (f *fakeBackend) RepairBrief(context.Context) (string, error) {
	return "Redacted manual repair brief", nil
}
func loaded(t *testing.T) (*model, *fakeBackend) {
	t.Helper()
	f := fixture()
	m := New(f, Options{NoColor: true, ASCII: true}).(*model)
	run(t, m, m.Init())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m, f
}
func run(t *testing.T, m *model, c tea.Cmd) {
	t.Helper()
	if c == nil {
		t.Fatal("expected asynchronous command")
	}
	_, next := m.Update(c())
	if next != nil {
		run(t, m, next)
	}
}
func press(m *model, s string) tea.Cmd {
	var k tea.KeyPressMsg
	switch s {
	case "enter":
		k.Code = tea.KeyEnter
	case "esc":
		k.Code = tea.KeyEscape
	case "tab":
		k.Code = tea.KeyTab
	case "shift+tab":
		k.Code = tea.KeyTab
		k.Mod = tea.ModShift
	case "ctrl+p":
		k.Code = 'p'
		k.Mod = tea.ModCtrl
	case "ctrl+c":
		k.Code = 'c'
		k.Mod = tea.ModCtrl
	case "down":
		k.Code = tea.KeyDown
	case "up":
		k.Code = tea.KeyUp
	case "pgdown":
		k.Code = tea.KeyPgDown
	default:
		k.Text = s
		k.Code = []rune(s)[0]
	}
	_, cmd := m.Update(k)
	return cmd
}
func activate(t *testing.T, m *model, id string) tea.Cmd {
	t.Helper()
	for i, c := range m.controls() {
		if c.id == id {
			m.focus = i
			return press(m, "enter")
		}
	}
	t.Fatalf("missing control %s on %s", id, m.screen)
	return nil
}
func readReview(t *testing.T, m *model) {
	t.Helper()
	for i := 0; i < 200 && !m.reviewRead; i++ {
		activate(t, m, "review:read")
	}
	if !m.reviewRead {
		t.Fatal("review never became readable")
	}
}

func TestInspectRenderAndNavigationNeverWrite(t *testing.T) {
	m, f := loaded(t)
	for i := 0; i < 4; i++ {
		m.View()
		press(m, "tab")
		press(m, "?")
		press(m, "esc")
	}
	if f.inspectCalls != 1 || f.planCalls+f.applyCalls+f.keyCalls+f.serviceCalls != 0 {
		t.Fatalf("unexpected calls: %+v", f)
	}
}
func TestCompleteDraftPlanApplyIdentity(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "enabled")
	activate(t, m, "model:glm-5.3")
	activate(t, m, "budget:353000")
	activate(t, m, "capacity")
	activate(t, m, "tiers")
	run(t, m, activate(t, m, "plan"))
	if f.plannedID != "codex" || !f.planned.Enabled || f.planned.Profile.Model != "glm-5.3" || f.planned.Profile.Budget != 353000 {
		t.Fatalf("wrong desired: %+v", f.planned)
	}
	if m.controls()[m.focus].id != "review:back" {
		t.Fatal("review must initially focus Back")
	}
	readReview(t, m)
	run(t, m, activate(t, m, "review:apply"))
	if f.appliedID != "opaque-plan" || f.appliedRevision != "exact-revision" {
		t.Fatal("apply not bound to reviewed plan")
	}
	if m.screen != "Dashboard" || m.drafts["codex"] != f.planned || len(m.receipts) != 1 {
		t.Fatal("receipt/dashboard not updated")
	}
}
func TestRefreshAndBackPreserveIndependentDrafts(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "model:glm-5.3-flash")
	d := m.drafts["codex"]
	activate(t, m, "enabled")
	activate(t, m, "enabled")
	if m.drafts["codex"].Profile != d.Profile {
		t.Fatal("disable erased profile")
	}
	press(m, "esc")
	run(t, m, press(m, "r"))
	if m.drafts["codex"].Profile != d.Profile || m.drafts["pi"].Profile.Model == d.Profile.Model {
		t.Fatal("refresh clobbered or shared drafts")
	}
}
func TestInvalidAndStaleRefusalKeepDraft(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "enabled")
	activate(t, m, "model:glm-5.3")
	activate(t, m, "plan")
	if f.planCalls != 0 {
		t.Fatal("unconfirmed profile planned")
	}
	activate(t, m, "capacity")
	activate(t, m, "tiers")
	d := m.drafts["codex"]
	run(t, m, activate(t, m, "plan"))
	readReview(t, m)
	f.applyErr = errors.New("stale revision")
	run(t, m, activate(t, m, "review:apply"))
	if m.drafts["codex"] != d || m.plan != nil {
		t.Fatal("stale refusal lost draft or retained plan")
	}
}
func TestTypingOverlayFocusAndQuitCancel(t *testing.T) {
	m, f := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "advanced:model")
	m.Update(tea.PasteMsg{Content: "exact-模型?r q /"})
	press(m, "?")
	press(m, "r")
	if m.modal != "" || f.inspectCalls != 1 {
		t.Fatal("text invoked global shortcut")
	}
	press(m, "esc")
	focus := m.focus
	press(m, "?")
	press(m, " ")
	press(m, "esc")
	if m.focus != focus || m.drafts["codex"].Enabled {
		t.Fatal("overlay changed background or focus")
	}
	press(m, "ctrl+p")
	m.Update(tea.PasteMsg{Content: "no such command"})
	press(m, "enter")
	if m.modal != "palette" {
		t.Fatal("zero result activated command")
	}
	press(m, "esc")
	activate(t, m, "enabled")
	press(m, "q")
	if m.modal != "quit" {
		t.Fatal("dirty quit not guarded")
	}
	press(m, "esc")
	if !m.drafts["codex"].Enabled {
		t.Fatal("quit cancel erased draft")
	}
}
func TestTierAvailabilitySeparateFromDefault(t *testing.T) {
	m, _ := loaded(t)
	activate(t, m, "client:codex")
	activate(t, m, "tier:Ultrafast")
	v := m.View().Content
	if !strings.Contains(v, "gpt-6.1-sol / Ultrafast") || !strings.Contains(v, "Available tiers: Not verified") {
		t.Fatal(v)
	}
	if m.drafts["codex"].Profile.Model != "gpt-6.1-sol" {
		t.Fatal("tier suffix leaked into model ID")
	}
}
