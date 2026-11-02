package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/codexrpc"
	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/credentials"
)

type fixtureAdapter struct {
	home    string
	profile core.Profile
	prepare func(context.Context, core.Profile, []byte) ([]configtxn.Write, error)
	inspect func(context.Context) (core.Client, error)
	writes  []configtxn.Write
}

func (a *fixtureAdapter) Inspect(ctx context.Context) (core.Client, error) {
	if a.inspect != nil {
		return a.inspect(ctx)
	}
	return core.Client{ID: "codex", Name: "Codex", Installed: true, Supported: true, Version: "0.166.0", Status: "CONFIGURED", Profile: a.profile}, nil
}
func (a *fixtureAdapter) Prepare(ctx context.Context, p core.Profile, catalog []byte) ([]configtxn.Write, error) {
	if a.prepare != nil {
		return a.prepare(ctx, p, catalog)
	}
	w := configtxn.Write{Path: filepath.Join(a.home, "config.toml"), Mode: 0600}
	if _, err := os.Stat(a.home); err == nil {
		var err error
		w, err = configtxn.Read(w.Path)
		if err != nil {
			return nil, err
		}
	}
	w.Data = []byte("fixture-reviewed-model = " + strconv.Quote(p.Model) + "\n")
	if p.Catalog != "" {
		w.Data = append(w.Data, []byte("model_catalog_json = "+strconv.Quote(p.Catalog)+"\n")...)
	}
	a.writes = []configtxn.Write{w}
	return a.writes, nil
}

type fixtureCredentials struct {
	mu  sync.Mutex
	key string
	err error
}

func (c *fixtureCredentials) SetSession(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.key = key
	return c.err
}
func (c *fixtureCredentials) Save(_ context.Context, key string) error { return c.SetSession(key) }
func (c *fixtureCredentials) Resolve(context.Context, string) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return "", "Locked", c.err
	}
	if c.key == "" {
		return "", "Missing", errors.New("missing fixture")
	}
	return c.key, "Session only", nil
}
func (c *fixtureCredentials) Status(ctx context.Context, ref string) string {
	_, s, _ := c.Resolve(ctx, ref)
	return s
}

func profile() core.Profile {
	return core.Profile{BaseURL: "https://fixture.invalid/v1", Model: "fixture-model", Budget: 128000, Compact: 100000, Tier: "Standard", EnvKey: "ENGINE_FIXTURE_KEY", CapacityConfirmed: true}
}
func fixture(t *testing.T, absent bool) (*Engine, *fixtureAdapter, *fixtureCredentials, Options) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "private")
	if err := configtxn.PrivateDir(root); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "codex")
	if !absent {
		if err := configtxn.PrivateDir(home); err != nil {
			t.Fatal(err)
		}
	}
	program := filepath.Join(root, "fixture-native")
	if err := configtxn.PrivateFile(program, []byte("fixture executable identity")); err != nil {
		t.Fatal(err)
	}
	o := Options{StateRoot: filepath.Join(root, "state"), CodexHome: home, CodexExecutable: program, Executable: program}
	a := &fixtureAdapter{home: home, profile: profile()}
	c := &fixtureCredentials{key: "fixture-session-key"}
	d := dependencies{adapter: a, program: func() codexrpc.Program { return codexrpc.Program{Path: program} }, credentials: c, childEnv: credentials.ChildEnv, lookup: func(string) (string, error) { return "", exec.ErrNotFound }}
	e, err := newEngine(o, d)
	if err != nil {
		t.Fatal(err)
	}
	return e, a, c, o
}
func plan(t *testing.T, e *Engine, d core.Desired) core.Plan {
	t.Helper()
	p, err := e.Plan(context.Background(), "codex", d)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReviewFreezesBytesAndCommitsStateWithClient(t *testing.T) {
	e, a, _, o := fixture(t, false)
	p := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	if _, err := os.Stat(filepath.Join(o.CodexHome, "config.toml")); !os.IsNotExist(err) {
		t.Fatal("Plan edited client")
	}
	a.writes[0].Data[0] = 'X'
	p.Desired.Profile.Model = "caller-mutated"
	p.Changes[0].After = "caller-mutated"
	r, err := e.Apply(context.Background(), p.ID, p.Revision)
	if err != nil || r.Outcome != "Succeeded" {
		t.Fatalf("apply: %#v %v", r, err)
	}
	got, _ := os.ReadFile(filepath.Join(o.CodexHome, "config.toml"))
	if !bytes.Contains(got, []byte(`fixture-reviewed-model = "fixture-model"`)) {
		t.Fatal("reviewed bytes were not frozen")
	}
	s, err := e.Inspect(context.Background())
	if err != nil || !s.Clients[0].Managed || len(s.Receipts) != 1 {
		t.Fatalf("state and receipt missing: %#v %v", s, err)
	}
	data, err := os.ReadFile(filepath.Join(o.StateRoot, "settings.json"))
	if err != nil || bytes.Contains(data, []byte("fixture.invalid")) || bytes.Contains(data, []byte("fixture-session-key")) {
		t.Fatal("unsafe settings")
	}
	if _, err = e.Apply(context.Background(), p.ID, p.Revision); err == nil {
		t.Fatal("plan replayed")
	}
}
func TestNewPlansInvalidateOlderAndForeignDriftRefuses(t *testing.T) {
	e, _, _, o := fixture(t, false)
	first := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	second := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	if _, err := e.Apply(context.Background(), first.ID, first.Revision); err == nil {
		t.Fatal("stale review accepted")
	}
	if err := configtxn.PrivateFile(filepath.Join(o.CodexHome, "config.toml"), []byte("foreign")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Apply(context.Background(), second.ID, second.Revision); err == nil {
		t.Fatal("foreign drift accepted")
	}
	data, _ := os.ReadFile(filepath.Join(o.CodexHome, "config.toml"))
	if string(data) != "foreign" {
		t.Fatal("foreign bytes overwritten")
	}
	if _, err := os.Stat(filepath.Join(o.StateRoot, "settings.json")); !os.IsNotExist(err) {
		t.Fatal("failed apply persisted state")
	}
}
func TestAbsentClientRootCreatedOnlyByExplicitApply(t *testing.T) {
	e, _, _, o := fixture(t, true)
	if _, err := e.Inspect(context.Background()); err != nil {
		t.Fatal(err)
	}
	p := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	if _, err := os.Stat(o.CodexHome); !os.IsNotExist(err) {
		t.Fatal("read or plan created client root")
	}
	if _, err := e.Apply(context.Background(), p.ID, p.Revision); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(o.CodexHome)
	if err != nil || !info.IsDir() || configtxn.PrivateDir(o.CodexHome) != nil {
		t.Fatal("Apply did not create private client root")
	}
}
func TestNoopAndDisablePreserveConfiguration(t *testing.T) {
	e, _, _, o := fixture(t, false)
	d := core.Desired{Enabled: true, Profile: profile()}
	p := plan(t, e, d)
	if _, err := e.Apply(context.Background(), p.ID, p.Revision); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(o.StateRoot, "settings.json")
	before, _ := os.Stat(path)
	state, _ := os.ReadFile(path)
	p = plan(t, e, d)
	if !p.Noop {
		t.Fatal("identical plan not Noop")
	}
	r, err := e.Apply(context.Background(), p.ID, p.Revision)
	if err != nil || r.Outcome != "Noop" {
		t.Fatal("no-op failed")
	}
	after, _ := os.Stat(path)
	now, _ := os.ReadFile(path)
	if !os.SameFile(before, after) || !bytes.Equal(state, now) {
		t.Fatal("Noop rewrote state or receipt")
	}
	clientBefore, _ := os.ReadFile(filepath.Join(o.CodexHome, "config.toml"))
	p = plan(t, e, core.Desired{Enabled: false, Profile: core.Profile{BaseURL: "invalid"}})
	if _, err := e.Apply(context.Background(), p.ID, p.Revision); err != nil {
		t.Fatal(err)
	}
	clientAfter, _ := os.ReadFile(filepath.Join(o.CodexHome, "config.toml"))
	if !bytes.Equal(clientBefore, clientAfter) {
		t.Fatal("disable edited native config")
	}
	s, err := e.Inspect(context.Background())
	if err != nil || s.Clients[0].Managed {
		t.Fatal("disable did not clear management")
	}
}
func TestExecutableDriftAndCanceledApplyDoNotMutate(t *testing.T) {
	for _, cancelCase := range []bool{false, true} {
		t.Run(map[bool]string{false: "executable", true: "cancel"}[cancelCase], func(t *testing.T) {
			e, _, _, o := fixture(t, true)
			p := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelCase {
				cancel()
			} else {
				if err := os.WriteFile(o.CodexExecutable, []byte("replaced"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := e.Apply(ctx, p.ID, p.Revision); err == nil {
				t.Fatal("unsafe apply accepted")
			}
			if _, err := os.Stat(o.CodexHome); !os.IsNotExist(err) {
				t.Fatal("refused apply created client root")
			}
			if _, err := os.Stat(filepath.Join(o.StateRoot, "settings.json")); !os.IsNotExist(err) {
				t.Fatal("refused apply wrote state")
			}
		})
	}
}
func TestLaunchRejectsNilChildEnvAndSanitizesErrors(t *testing.T) {
	e, a, c, _ := fixture(t, false)
	e.deps.childEnv = func(string, string) []string { return nil }
	if cmd, err := e.Command(context.Background(), "launch", "codex"); cmd != nil || err == nil {
		t.Fatal("nil Env would inherit ambient credentials")
	}
	c.err = errors.New("fixture-session-key https://secret.invalid/?token=hidden")
	if _, err := e.Command(context.Background(), "launch", "codex"); err == nil || strings.Contains(err.Error(), "hidden") || strings.Contains(err.Error(), "fixture-session") {
		t.Fatal("credential error leaked")
	}
	a.inspect = func(context.Context) (core.Client, error) { return core.Client{}, c.err }
	if _, err := e.Inspect(context.Background()); err == nil || strings.Contains(err.Error(), "hidden") {
		t.Fatal("adapter error leaked")
	}
}
func TestJSONAndDiagnosticsDoNotExposePrivateInputs(t *testing.T) {
	e, _, _, _ := fixture(t, false)
	p := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	s, err := e.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []any{p, s} {
		data, err := json.Marshal(v)
		if err != nil || bytes.Contains(data, []byte("fixture.invalid")) || bytes.Contains(data, []byte("fixture-session-key")) || bytes.Contains(data, []byte("fixture-native")) {
			t.Fatal("public JSON leaked private input")
		}
	}
	brief, err := e.RepairBrief(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(brief)) || !strings.Contains(brief, `"snapshot"`) {
		t.Fatal("repair brief must return redacted JSON content for preview and export")
	}
	paths, err := filepath.Glob(filepath.Join(e.options.StateRoot, "diagnostics", "brief-*.json"))
	if err != nil || len(paths) != 1 {
		t.Fatal("private diagnostic export missing")
	}
	export, err := configtxn.Read(paths[0])
	if err != nil || !export.Existed || export.Mode.Perm() != 0600 {
		t.Fatal("diagnostic not private")
	}
	data, _ := os.ReadFile(paths[0])
	if string(data) != brief || bytes.Contains(data, []byte("fixture.invalid")) || bytes.Contains(data, []byte("fixture-native")) {
		t.Fatal("diagnostic leaked")
	}
}
