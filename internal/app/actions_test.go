package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestInventoryHasSixCandidatesWithoutWrapperExecution(t *testing.T) {
	e, _, _, o := fixture(t, true)
	var looked []string
	e.deps.lookup = func(name string) (string, error) { looked = append(looked, name); return "/fixture/not-executed", nil }
	s, err := e.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"codex", "claude", "pi", "opencode", "grokbuild", "hermes"}
	var ids []string
	for i, c := range s.Clients {
		ids = append(ids, c.ID)
		if i > 0 && (!c.Installed || c.Supported || c.Reason == "") {
			t.Fatal("unverified PATH candidate advertised as supported")
		}
	}
	if !reflect.DeepEqual(ids, want) || !reflect.DeepEqual(looked, []string{"claude", "pi", "opencode", "grok", "hermes"}) {
		t.Fatal("wrong candidate identities")
	}
	if _, err := os.Stat(filepath.Join(o.StateRoot, "transactions")); !os.IsNotExist(err) {
		t.Fatal("Inspect performed recovery")
	}
}
func TestLaunchDeliversSelectedKeyWithoutAmbientCredentials(t *testing.T) {
	e, _, c, o := fixture(t, false)
	t.Setenv("OTHER_API_KEY", "unrelated-fixture")
	t.Setenv("CODEX_PROFILE", "ambient-fixture")
	t.Setenv("CODEX_HOME", "ambient-root")
	c.key = "fixture-selected-key"
	cmd, err := e.Command(context.Background(), "launch", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != o.CodexExecutable || len(cmd.Args) != 1 || cmd.Env == nil {
		t.Fatal("launch did not use literal native command")
	}
	env := map[string]string{}
	for _, entry := range cmd.Env {
		k, v, _ := strings.Cut(entry, "=")
		env[k] = v
	}
	if env["ENGINE_FIXTURE_KEY"] != c.key || env["CODEX_HOME"] != o.CodexHome || env["OTHER_API_KEY"] != "" || env["CODEX_PROFILE"] != "" {
		t.Fatal("unsafe launch environment")
	}
	if strings.Contains(strings.Join(cmd.Args, " "), c.key) || os.Getenv("CODEX_HOME") != "ambient-root" {
		t.Fatal("launch mutated parent or put key in argv")
	}
	for _, id := range []string{"claude", "pi", "opencode", "grokbuild", "hermes"} {
		if cmd, err := e.Command(context.Background(), "launch", id); cmd != nil || err == nil {
			t.Fatal("unsupported client launched")
		}
	}
}
func TestDiscoveryIsExplicitAndMetadataFailureKeepsExactIDs(t *testing.T) {
	e, _, c, _ := fixture(t, false)
	calls := 0
	e.deps.discover = func(_ context.Context, url, key string) ([]core.ModelInfo, error) {
		calls++
		if url != profile().BaseURL || key != c.key {
			t.Fatal("wrong discovery inputs")
		}
		return []core.ModelInfo{{ID: "provider/Exact-ID", Source: "gateway inventory"}}, nil
	}
	e.deps.enrich = func(context.Context, []core.ModelInfo) ([]core.ModelInfo, error) {
		return nil, errors.New("metadata fixture secret")
	}
	if _, err := e.Inspect(context.Background()); err != nil || calls != 0 {
		t.Fatal("Inspect discovered models")
	}
	got, err := e.Discover(context.Background(), "codex")
	if err != nil || calls != 1 || len(got) != 1 || got[0].ID != "provider/Exact-ID" {
		t.Fatal("discovery did not retain exact base IDs")
	}
	e.deps.discover = func(context.Context, string, string) ([]core.ModelInfo, error) {
		return nil, errors.New("private-key endpoint.invalid")
	}
	if _, err = e.Discover(context.Background(), "codex"); err == nil || strings.Contains(err.Error(), "private-key") {
		t.Fatal("discovery error leaked")
	}
}
func TestTierCatalogRequiresExactAttestedEligibleRoute(t *testing.T) {
	e, a, _, _ := fixture(t, false)
	calls := 0
	e.deps.confirmed = func(_ context.Context, version, model string, tiers []string) ([]byte, error) {
		calls++
		if version != "0.166.0" || model != "gpt-6.1-sol" || !reflect.DeepEqual(tiers, []string{"priority", "ultrafast"}) {
			t.Fatal("catalog must advertise both explicitly attested tiers")
		}
		return []byte("fixture-complete-catalog"), nil
	}
	a.prepare = func(_ context.Context, p core.Profile, data []byte) ([]configtxn.Write, error) {
		if string(data) != "fixture-complete-catalog" {
			t.Fatal("catalog not passed to native adapter")
		}
		return []configtxn.Write{{Path: filepath.Join(a.home, "config.toml"), Mode: 0600, Data: []byte("fixture = true\n")}}, nil
	}
	p := profile()
	p.Model = "gpt-6.1-sol"
	p.Tier = "Ultrafast"
	if _, err := e.Plan(context.Background(), "codex", core.Desired{Enabled: true, Profile: p}); err == nil || calls != 0 {
		t.Fatal("unattested tier queried")
	}
	p.TiersConfirmed = true
	if _, err := e.Plan(context.Background(), "codex", core.Desired{Enabled: true, Profile: p}); err != nil || calls != 1 {
		t.Fatal("confirmed exact catalog not used")
	}
	p.Model = "GLM-fixture"
	if _, err := e.Plan(context.Background(), "codex", core.Desired{Enabled: true, Profile: p}); err == nil || calls != 1 {
		t.Fatal("invented unknown model catalog")
	}
}
func TestOverlappingPlansCannotResurrectOlderReview(t *testing.T) {
	e, a, _, _ := fixture(t, false)
	entered := make(chan struct{})
	release := make(chan struct{})
	var count atomic.Int32
	a.prepare = func(ctx context.Context, p core.Profile, _ []byte) ([]configtxn.Write, error) {
		if count.Add(1) == 1 {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return []configtxn.Write{{Path: filepath.Join(a.home, "config.toml"), Mode: 0600, Data: []byte("fixture = true\n")}}, nil
	}
	done := make(chan error, 1)
	go func() {
		_, err := e.Plan(context.Background(), "codex", core.Desired{Enabled: true, Profile: profile()})
		done <- err
	}()
	<-entered
	newer := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	close(release)
	if err := <-done; err == nil {
		t.Fatal("older async review resurrected")
	}
	if _, err := e.Apply(context.Background(), newer.ID, newer.Revision); err != nil {
		t.Fatal(err)
	}
}
func TestCanceledStagingDoesNotPublishReview(t *testing.T) {
	e, a, _, o := fixture(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	a.prepare = func(context.Context, core.Profile, []byte) ([]configtxn.Write, error) {
		cancel()
		return []configtxn.Write{{Path: filepath.Join(o.CodexHome, "config.toml"), Mode: 0600, Data: []byte("fixture = true\n")}}, nil
	}
	if p, err := e.Plan(ctx, "codex", core.Desired{Enabled: true, Profile: profile()}); err == nil || p.ID != "" {
		t.Fatal("canceled stage published")
	}
	if _, err := os.Stat(o.CodexHome); !os.IsNotExist(err) {
		t.Fatal("canceled stage created root")
	}
}
func TestInvalidStateFailsClosedAndNeverExposesRawContent(t *testing.T) {
	cases := []string{`{"version":2,"managed":{},"receipts":[]}`, `{"version":1,"version":1,"managed":{},"receipts":[]}`, `{"Version":1,"managed":{},"receipts":[]}`, `{"version":1,"managed":{"codex":null},"receipts":[]}`, `{"version":1,"managed":{},"receipts":[],"endpoint":"fixture-secret"}`, `{"version":1,"managed":{},"receipts":null}`,
		`{"version":1,"managed":{},"receipts":[],"attestations":null}`,
		`{"version":1,"managed":{},"receipts":[],"attestations":{"claude":{"capacity":"fixture-secret"}}}`,
		`{"version":1,"managed":{},"receipts":[],"attestations":{"codex":{}}}`,
		`{"version":1,"managed":{},"receipts":[],"attestations":{"codex":{"capacity":null}}}`,
		`{"version":1,"managed":{},"receipts":[],"attestations":{"codex":{"capacity":"fixture-secret"}}}`,
		`{"version":1,"managed":{},"receipts":[],"attestations":{"codex":{"Capacity":"` + strings.Repeat("a", 64) + `"}}}`,
		`{"version":1,"managed":{},"receipts":[],"attestations":{"codex":{"capacity":"` + strings.Repeat("a", 64) + `","capacity":"` + strings.Repeat("b", 64) + `"}}}`,
		`{"version":1,"managed":{},"receipts":[],"attestations":{"codex":{"capacity":"` + strings.Repeat("A", 64) + `"}}}`,
	}
	for _, data := range cases {
		t.Run(data, func(t *testing.T) {
			e, _, _, o := fixture(t, false)
			path := filepath.Join(o.StateRoot, "settings.json")
			if err := configtxn.PrivateFile(path, []byte(data)); err != nil {
				t.Fatal(err)
			}
			if _, err := e.Inspect(context.Background()); err == nil || strings.Contains(err.Error(), "fixture-secret") {
				t.Fatal("unsafe state accepted or echoed")
			}
			if _, err := newEngine(o, e.deps); err == nil {
				t.Fatal("constructor accepted invalid existing state")
			}
		})
	}
}
func TestSaveKeyDefaultsToSessionAndServicesAreExplicit(t *testing.T) {
	e, _, c, _ := fixture(t, false)
	if err := e.SaveKey(context.Background(), "fixture-new-session", false); err != nil || c.key != "fixture-new-session" {
		t.Fatal("session save failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.SaveKey(ctx, "must-not-save", false); err == nil || c.key != "fixture-new-session" {
		t.Fatal("canceled save changed credential")
	}
	if _, err := e.Service(context.Background(), "start"); err == nil {
		t.Fatal("missing backend reported successful activation")
	}
	if cmd, err := e.Command(context.Background(), "update", "codex"); cmd != nil || err == nil {
		t.Fatal("unsigned update offered")
	}
	if cmd, err := e.Command(context.Background(), "ssh", "codex"); cmd != nil || err == nil {
		t.Fatal("uninjected SSH command offered")
	}
	_ = exec.ErrNotFound
}
