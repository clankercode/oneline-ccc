package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/tui"
)

type fakeEngine struct {
	core.Backend
	inspect, service, observe, recover, command int
	action, id                                  string
	once                                        bool
	snapshot                                    core.Snapshot
	err                                         error
	cmd                                         *exec.Cmd
}

func (f *fakeEngine) Inspect(context.Context) (core.Snapshot, error) {
	f.inspect++
	return f.snapshot, f.err
}
func (f *fakeEngine) Service(_ context.Context, a string) (core.ServiceInfo, error) {
	f.service++
	f.action = a
	return core.ServiceInfo{Status: "Observed"}, f.err
}
func (f *fakeEngine) Observe(_ context.Context, once bool) error {
	f.observe++
	f.once = once
	return f.err
}
func (f *fakeEngine) Recover(context.Context) ([]configtxn.Result, error) {
	f.recover++
	return []configtxn.Result{{ID: "txn-id", Outcome: "Succeeded", Changed: []string{"/private/config"}}}, f.err
}
func (f *fakeEngine) Command(_ context.Context, a, id string) (*exec.Cmd, error) {
	f.command++
	f.action = a
	f.id = id
	return f.cmd, f.err
}
func (f *fakeEngine) RepairBrief(context.Context) (string, error) {
	return "Manual redacted repair brief", f.err
}
func harness(t *testing.T) (dependencies, *fakeEngine, *bytes.Buffer, *bytes.Buffer, *int) {
	t.Helper()
	root := t.TempDir()
	f := &fakeEngine{snapshot: core.Snapshot{Clients: []core.Client{{ID: "codex", Name: "Codex", Installed: true, Supported: true, Status: "Unverified", Executable: "/private/codex", Profile: core.Profile{BaseURL: "https://secret.invalid/v1", Model: "exact-模型", Budget: 353000, Tier: "Standard"}}}, Credential: "Missing"}}
	out, errout := &bytes.Buffer{}, &bytes.Buffer{}
	calls := new(int)
	d := dependencies{in: strings.NewReader(""), out: out, errout: errout, lookup: func(string) (string, bool) { return "", false }, userHome: func() (string, error) { return filepath.Join(root, "home"), nil }, userConfig: func() (string, error) { return filepath.Join(root, "config"), nil }, executable: func() (string, error) { return filepath.Join(root, "oneline-ccc"), nil }, newEngine: func(engineOptions) (engine, error) { *calls++; return f, nil }, runUI: func(core.Backend, tui.Options) error { t.Fatal("unexpected UI"); return nil }, runChild: func(context.Context, *exec.Cmd, io.Reader, io.Writer, io.Writer) error {
		t.Fatal("unexpected child")
		return nil
	}}
	return d, f, out, errout, calls
}
func TestHelpVersionAndInvalidFlagsNeverConstructEngine(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"status", "--help"}, {"--version"}, {"version"}, {"--unknown=private-value"}, {"--theme", "broken"}, {"status", "--key", "synthetic-secret"}, {"apply", "--yes"}, {"observe", "--json"}, {"status", "unexpected"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			d, _, out, errout, calls := harness(t)
			d.terminal = true
			d.lookup = func(k string) (string, bool) { return "", k == "NO_COLOR" }
			d.userHome = func() (string, error) { t.Fatal("help/invalid read paths"); return "", nil }
			d.userConfig = d.userHome
			d.executable = d.userHome
			code := run(context.Background(), args, d)
			if *calls != 0 {
				t.Fatal("constructed engine")
			}
			if strings.Contains(out.String()+errout.String(), "synthetic-secret") || strings.Contains(errout.String(), "private-value") {
				t.Fatal("invalid args leaked values")
			}
			if strings.Contains(strings.Join(args, " "), "help") || len(args) == 1 && (args[0] == "--version" || args[0] == "version") {
				if code != 0 {
					t.Fatal("help/version failed")
				}
			} else if code != 2 {
				t.Fatalf("invalid flags exit %d", code)
			}
		})
	}
}
func TestNonTTYAndPlainAreReadOnlyAndEscapeFree(t *testing.T) {
	for _, args := range [][]string{nil, {"--plain"}, {"--ascii", "status"}, {"status", "--json"}} {
		d, f, out, errout, _ := harness(t)
		if code := run(context.Background(), args, d); code != 0 {
			t.Fatalf("exit %d: %s", code, errout)
		}
		if f.inspect != 1 || f.service+f.observe+f.recover+f.command != 0 {
			t.Fatal("status performed mutations")
		}
		if strings.Contains(out.String(), "\x1b") || strings.Contains(out.String(), "secret.invalid") || strings.Contains(out.String(), "/private/codex") {
			t.Fatal("unsafe public output")
		}
		if !strings.Contains(out.String(), "exact-模型") {
			t.Fatal("exact model ID lost")
		}
	}
}
func TestNoColorKeepsInteractiveManagement(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		present     bool
	}{{"unset", "", false}, {"empty", "", true}, {"nonempty", "1", true}} {
		t.Run(tc.name, func(t *testing.T) {
			d, f, out, errout, calls := harness(t)
			d.terminal = true
			d.lookup = func(k string) (string, bool) { return tc.value, k == "NO_COLOR" && tc.present }
			uiCalls := 0
			d.runUI = func(b core.Backend, o tui.Options) error {
				uiCalls++
				if b != f || o.NoColor != tc.present {
					t.Fatalf("management backend or NoColor lost: NoColor=%t, want %t", o.NoColor, tc.present)
				}
				return nil
			}
			if code := run(context.Background(), nil, d); code != 0 {
				t.Fatalf("exit %d: %s", code, errout)
			}
			if uiCalls != 1 || *calls != 1 || f.inspect != 0 || out.Len() != 0 {
				t.Fatalf("interactive management replaced by status: UI=%d engines=%d inspect=%d", uiCalls, *calls, f.inspect)
			}
		})
	}
}
func TestTerminalPassesUIOptionsAndExplicitPaths(t *testing.T) {
	d, _, _, _, _ := harness(t)
	d.terminal = true
	called := false
	var opts engineOptions
	d.newEngine = func(o engineOptions) (engine, error) { opts = o; return &fakeEngine{}, nil }
	d.runUI = func(_ core.Backend, o tui.Options) error {
		called = true
		if o.Theme != "light" || !o.ASCII || o.NoColor || !o.ReducedMotion || o.Version != version {
			t.Fatal("UI options lost")
		}
		return nil
	}
	root := t.TempDir()
	if run(context.Background(), []string{"--state-dir", filepath.Join(root, "state"), "--theme=light", "--ascii", "--reduced-motion", "--codex-home", filepath.Join(root, "codex"), "--codex-executable", filepath.Join(root, "exe")}, d) != 0 || !called {
		t.Fatal("TUI not invoked")
	}
	if opts.StateRoot != filepath.Join(root, "state") || opts.CodexHome != filepath.Join(root, "codex") || opts.CodexExecutable != filepath.Join(root, "exe") {
		t.Fatal("explicit paths lost")
	}
}
func TestExplicitCommandsUseFrozenPorts(t *testing.T) {
	for _, args := range [][]string{{"doctor"}, {"service", "status"}, {"service", "install"}, {"service", "start"}, {"service", "stop"}, {"service", "restart"}, {"service", "uninstall"}, {"observe", "--once"}, {"recover"}, {"repair"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			d, f, out, errout, _ := harness(t)
			if code := run(context.Background(), args, d); code != 0 {
				t.Fatalf("exit %d: %s", code, errout)
			}
			switch args[0] {
			case "doctor":
				if f.inspect != 1 {
					t.Fatal("doctor not inspected")
				}
			case "service":
				if f.service != 1 || f.action != args[1] {
					t.Fatal("wrong service action")
				}
			case "observe":
				if f.observe != 1 || !f.once {
					t.Fatal("wrong observer")
				}
			case "recover":
				if f.recover != 1 || strings.Contains(out.String(), "/private/config") {
					t.Fatal("recovery leaked private paths")
				}
			case "repair":
				if !strings.Contains(out.String(), "Manual redacted") {
					t.Fatal("repair missing")
				}
			}
		})
	}
}
func TestChildEnvironmentCannotInheritAndStreamsForward(t *testing.T) {
	d, f, _, _, _ := harness(t)
	f.cmd = exec.Command("synthetic")
	if run(context.Background(), []string{"launch", "codex"}, d) == 0 {
		t.Fatal("nil environment accepted")
	}
	f.cmd.Env = []string{}
	called := false
	d.runChild = func(_ context.Context, c *exec.Cmd, in io.Reader, out, errout io.Writer) error {
		called = true
		if c != f.cmd || in != d.in || out != d.out || errout != d.errout || c.Env == nil {
			t.Fatal("command/environment/streams changed")
		}
		return nil
	}
	if run(context.Background(), []string{"launch", "codex"}, d) != 0 || !called || f.action != "launch" || f.id != "codex" {
		t.Fatal("launch not forwarded")
	}
}
func TestBackendErrorsNeverPrintRawDetails(t *testing.T) {
	d, f, _, errout, _ := harness(t)
	f.err = errors.New("https://secret.invalid/v1 key=synthetic-secret /private/codex\x1b[31m")
	if run(context.Background(), []string{"status"}, d) == 0 {
		t.Fatal("error reported success")
	}
	if strings.Contains(errout.String(), "synthetic-secret") || strings.Contains(errout.String(), "secret.invalid") || strings.Contains(errout.String(), "\x1b") {
		t.Fatal("backend error leaked")
	}
}
func TestDefaultResolutionDoesNotCreateDirectories(t *testing.T) {
	d, _, _, _, _ := harness(t)
	root := t.TempDir()
	var got engineOptions
	d.newEngine = func(o engineOptions) (engine, error) { got = o; return &fakeEngine{}, nil }
	d.lookup = func(k string) (string, bool) {
		if k == "CODEX_HOME" {
			return filepath.Join(root, "explicit-home"), true
		}
		return "", false
	}
	if run(context.Background(), []string{"status"}, d) != 0 {
		t.Fatal("status failed")
	}
	if got.CodexHome != filepath.Join(root, "explicit-home") {
		t.Fatal("CODEX_HOME ignored")
	}
	if _, err := os.Stat(got.StateRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("CLI created state during status")
	}
}
