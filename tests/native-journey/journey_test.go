package nativejourney

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/app"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestNativeEngineReviewApplyRerun(t *testing.T) {
	binary := os.Getenv("ONELINE_CODEX_TEST_BINARY")
	if binary == "" {
		t.Skip("requires explicit isolated native Codex fixture")
	}
	root := t.TempDir()
	home := filepath.Join(root, "codex")
	state := filepath.Join(root, "state")
	e, err := app.New(app.Options{StateRoot: state, CodexHome: home, CodexExecutable: binary})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	snapshot, err := e.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Clients) != 6 || !snapshot.Clients[0].Supported {
		t.Fatal("actual native adapter not available")
	}
	p := core.Defaults()
	p.BaseURL = "http://127.0.0.1:9/v1"
	p.CapacityConfirmed = true
	desired := core.Desired{Enabled: true, Profile: p}
	plan, err := e.Plan(ctx, "codex", desired)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Noop || len(plan.Changes) == 0 || plan.ID == "" || plan.Revision == "" {
		t.Fatal("missing real reviewed changes")
	}
	if _, err = os.Stat(home); !os.IsNotExist(err) {
		t.Fatal("review created client home")
	}
	receipt, err := e.Apply(ctx, plan.ID, plan.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Outcome != "Succeeded" {
		t.Fatalf("apply outcome %s", receipt.Outcome)
	}
	config := filepath.Join(home, "config.toml")
	info, err := os.Stat(config)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = e.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := snapshot.Clients[0]
	if !client.Managed || client.Profile.Model != p.Model || client.Profile.Budget != p.Budget || client.Profile.Tier != "Standard" {
		t.Fatal("applied values not inspected")
	}
	safe, _ := json.Marshal(snapshot)
	if bytes.Contains(safe, []byte("127.0.0.1")) || bytes.Contains(safe, []byte(home)) {
		t.Fatal("snapshot exposed private endpoint or paths")
	}
	plan, err = e.Plan(ctx, "codex", core.Desired{Enabled: true, Profile: client.Profile})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Noop {
		t.Fatal("unchanged rerun is not a no-op")
	}
	if _, err = e.Apply(ctx, plan.ID, plan.Revision); err != nil {
		t.Fatal(err)
	}
	now, _ := os.Stat(config)
	after, _ := os.ReadFile(config)
	if !bytes.Equal(before, after) || !info.ModTime().Equal(now.ModTime()) {
		t.Fatal("no-op rewrote client config")
	}
	p.Budget = 353000
	p.Compact = 300000
	stale, err := e.Plan(ctx, "codex", core.Desired{Enabled: true, Profile: p})
	if err != nil {
		t.Fatal(err)
	}
	foreign := append(bytes.Clone(before), []byte("\n# foreign edit after review\n")...)
	if err = os.WriteFile(config, foreign, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Apply(ctx, stale.ID, stale.Revision); err == nil {
		t.Fatal("stale real plan applied")
	}
	after, _ = os.ReadFile(config)
	if !bytes.Equal(foreign, after) {
		t.Fatal("stale plan overwrote foreign edit")
	}
}

func TestNativeLaunchUsesSessionKeyWithoutParentSecrets(t *testing.T) {
	binary := os.Getenv("ONELINE_CODEX_TEST_BINARY")
	if binary == "" {
		t.Skip("requires explicit native fixture")
	}
	root := t.TempDir()
	home := filepath.Join(root, "codex")
	e, err := app.New(app.Options{StateRoot: filepath.Join(root, "state"), CodexHome: home, CodexExecutable: binary})
	if err != nil {
		t.Fatal(err)
	}
	p := core.Defaults()
	p.BaseURL = "http://127.0.0.1:9/v1"
	p.CapacityConfirmed = true
	ctx := context.Background()
	plan, err := e.Plan(ctx, "codex", core.Desired{Enabled: true, Profile: p})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Apply(ctx, plan.ID, plan.Revision); err != nil {
		t.Fatal(err)
	}
	key := "fixture-session-key-never-an-operator-key"
	t.Setenv("UNRELATED_API_KEY", "fixture-unrelated-secret")
	if err = e.SaveKey(ctx, key, false); err != nil {
		t.Fatal(err)
	}
	command, err := e.Command(ctx, "launch", "codex")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(command.Env, "\n")
	if command.Env == nil || !strings.Contains(joined, "LLMP_API_KEY="+key) || strings.Contains(joined, "fixture-unrelated-secret") {
		t.Fatal("child credential boundary failed")
	}
	command.Args = append(command.Args, "debug", "prompt-input", "Offline native launch fixture")
	output, err := command.Output()
	if err != nil {
		t.Fatal("offline native launch failed")
	}
	if len(output) == 0 || bytes.Contains(output, []byte(key)) {
		t.Fatal("offline prompt missing or credential exposed")
	}
	after, _ := os.ReadFile(filepath.Join(home, "config.toml"))
	if bytes.Contains(after, []byte(key)) {
		t.Fatal("session credential persisted")
	}
}
