package credentials

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

type fakeVault struct {
	value      string
	err        error
	gets, sets int
}

func (v *fakeVault) Get(service, account string) (string, error) {
	if service != "oneline-ccc" || account != "default" {
		panic("wrong vault reference")
	}
	v.gets++
	return v.value, v.err
}
func (v *fakeVault) Set(service, account, key string) error {
	if service != "oneline-ccc" || account != "default" {
		panic("wrong vault reference")
	}
	v.sets++
	if v.err == nil {
		v.value = key
	}
	return v.err
}
func manager(v *fakeVault, env map[string]string) *Manager {
	return &Manager{vault: v, lookup: func(name string) (string, bool) { v, ok := env[name]; return v, ok }}
}
func TestResolutionPrecedence(t *testing.T) {
	v := &fakeVault{value: "fixture-vault"}
	m := manager(v, map[string]string{"SELECTED_KEY": "fixture-env"})
	if err := m.SetSession("fixture-session"); err != nil {
		t.Fatal(err)
	}
	key, status, err := m.Resolve(context.Background(), "SELECTED_KEY")
	if key != "fixture-session" || status != "Session only" || err != nil || v.gets != 0 {
		t.Fatal("session precedence failed")
	}
	m = manager(v, map[string]string{"SELECTED_KEY": "fixture-env"})
	key, status, err = m.Resolve(context.Background(), "SELECTED_KEY")
	if key != "fixture-env" || status != "Stored" || err != nil || v.gets != 0 {
		t.Fatal("environment precedence failed")
	}
	m = manager(v, nil)
	key, status, err = m.Resolve(context.Background(), "SELECTED_KEY")
	if key != "fixture-vault" || status != "Stored" || err != nil {
		t.Fatal("vault resolution failed")
	}
}
func TestVaultFailuresAreActionableAndRedacted(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status string
	}{{keyring.ErrNotFound, "Missing"}, {errors.New("fixture-secret private endpoint"), "Locked"}} {
		m := manager(&fakeVault{err: tc.err}, nil)
		key, status, err := m.Resolve(context.Background(), "SELECTED_KEY")
		if key != "" || status != tc.status || err == nil || strings.Contains(err.Error(), "fixture-secret") || !strings.Contains(err.Error(), "session") {
			t.Fatal("unsafe vault failure")
		}
		if m.Status(context.Background(), "SELECTED_KEY") != tc.status {
			t.Fatal("incorrect status")
		}
		if err := m.Save(context.Background(), "fixture-secret"); err == nil || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatal("unsafe save error")
		}
		if err := m.SetSession("fixture-session"); err != nil {
			t.Fatal(err)
		}
		if _, status, err := m.Resolve(context.Background(), "SELECTED_KEY"); err != nil || status != "Session only" {
			t.Fatal("session must bypass locked vault")
		}
	}
}
func TestSaveAndCancellation(t *testing.T) {
	v := &fakeVault{}
	m := manager(v, nil)
	if err := m.Save(context.Background(), "fixture-secret"); err != nil || v.value != "fixture-secret" || v.sets != 1 {
		t.Fatal("save failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Save(ctx, "fixture-secret"); err == nil || v.sets != 1 {
		t.Fatal("canceled save wrote")
	}
	if key, _, err := m.Resolve(ctx, "SELECTED_KEY"); err == nil || key != "" || v.gets != 0 {
		t.Fatal("canceled resolve read")
	}
	for _, key := range []string{"", "fixture\nsecret", "fixture\x00secret", strings.Repeat("x", 2561)} {
		if m.SetSession(key) == nil || m.Save(context.Background(), key) == nil {
			t.Fatal("invalid credential accepted")
		}
	}
	if key, _, err := m.Resolve(context.Background(), "BAD=NAME"); key != "" || err == nil {
		t.Fatal("bad reference accepted")
	}
}
func TestNewDoesNotAccessVault(t *testing.T) {
	m := New()
	if err := m.SetSession("fixture-session"); err != nil {
		t.Fatal(err)
	}
	if m.Status(context.Background(), "SELECTED_KEY") != "Session only" {
		t.Fatal("constructor/session failed")
	}
}
func TestChildEnvDelivery(t *testing.T) {
	if os.Getenv("ONELINE_FIXTURE_CHILD") == "1" {
		if os.Getenv("SELECTED_KEY") != "fixture-child-secret" || os.Getenv("OTHER_API_KEY") != "" || os.Getenv("GITHUB_TOKEN") != "" || os.Getenv("CODEX_PROFILE") != "" || os.Getenv("LD_PRELOAD") != "" {
			os.Exit(2)
		}
		os.Exit(0)
	}
	t.Setenv("SELECTED_KEY", "fixture-parent")
	t.Setenv("OTHER_API_KEY", "fixture-other")
	t.Setenv("GITHUB_TOKEN", "fixture-token")
	t.Setenv("CODEX_PROFILE", "fixture-profile")
	before := os.Getenv("SELECTED_KEY")
	cmd := exec.Command(os.Args[0], "-test.run=^TestChildEnvDelivery$")
	cmd.Env = append(ChildEnv("SELECTED_KEY", "fixture-child-secret"), "ONELINE_FIXTURE_CHILD=1")
	if err := cmd.Run(); err != nil {
		t.Fatal("synthetic child rejected environment")
	}
	if os.Getenv("SELECTED_KEY") != before {
		t.Fatal("parent environment changed")
	}
	for _, name := range []string{"BAD=NAME", "PATH", "CODEX_PROFILE", "LD_PRELOAD"} {
		if ChildEnv(name, "fixture-secret") != nil {
			t.Fatal("unsafe env reference accepted")
		}
	}
	if ChildEnv("SELECTED_KEY", "bad\x00key") != nil {
		t.Fatal("invalid credential accepted")
	}
	for _, entry := range ChildEnv("SELECTED_KEY", "fixture-child-secret") {
		name, _, _ := strings.Cut(entry, "=")
		if name != "SELECTED_KEY" && !essentialEnv(name) {
			t.Fatal("environment was not allowlisted")
		}
	}
}

func TestUnavailableVaultNeverFallsBackToPersistence(t *testing.T) {
	m := &Manager{vault: unavailableVault{}, lookup: func(string) (string, bool) { return "", false }}
	if key, status, err := m.Resolve(context.Background(), "SELECTED_KEY"); key != "" || status != "Locked" || err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatal("disabled vault not explicit")
	}
	if err := m.Save(context.Background(), "fixture-session"); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatal("disabled vault save accepted")
	}
	if err := m.SetSession("fixture-session"); err != nil {
		t.Fatal(err)
	}
	if key, status, err := m.Resolve(context.Background(), "SELECTED_KEY"); key != "fixture-session" || status != "Session only" || err != nil {
		t.Fatal("disabled vault blocked session entry")
	}
}
func TestInvalidEnvironmentDoesNotFallBackToVault(t *testing.T) {
	v := &fakeVault{value: "fixture-vault"}
	m := manager(v, map[string]string{"SELECTED_KEY": "invalid\nfixture"})
	if key, _, err := m.Resolve(context.Background(), "SELECTED_KEY"); key != "" || err == nil || v.gets != 0 {
		t.Fatal("invalid credential silently fell back")
	}
}
