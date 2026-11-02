package monitor

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type fakeService struct {
	loaded, active, enabled        bool
	fragment, invocation, filePath string
	calls                          [][]string
	fail                           string
	after                          func(string)
	unitPaths, effectiveArgv       []string
	dropInPaths                    string
}

func (f *fakeService) run(_ context.Context, name string, args ...string) (commandResult, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if filepath.Base(name) == "busctl" {
		var value any
		kind := "as"
		switch args[len(args)-1] {
		case "UnitPath":
			value = f.unitPaths
		case "ExecStart":
			kind = "a(sasbttttuii)"
			value = []any{[]any{f.effectiveArgv[0], f.effectiveArgv, false, 0, 0, 0, 0, 0, 0, 0}}
		default:
			return commandResult{Code: 1}, nil
		}
		data, _ := json.Marshal(struct {
			Type string `json:"type"`
			Data any    `json:"data"`
		}{Type: kind, Data: value})
		return commandResult{Output: string(data)}, nil
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "show ") {
		load := "not-found"
		active := "inactive"
		enabled := ""
		if f.loaded {
			load = "loaded"
		}
		if f.active {
			active = "active"
		}
		if f.enabled {
			enabled = "enabled"
		}
		output := "LoadState=" + load + "\nActiveState=" + active + "\nUnitFileState=" + enabled + "\nFragmentPath=" + f.fragment + "\nInvocationID=" + f.invocation + "\n"
		if strings.Contains(joined, "DropInPaths") {
			output += "DropInPaths=" + f.dropInPaths + "\nSourcePath=\nNames=" + unitName + "\n"
		}
		return commandResult{Output: output}, nil
	}
	action := ""
	for _, candidate := range []string{"daemon-reload", "enable", "disable", "start", "stop", "restart"} {
		for _, arg := range args {
			if arg == candidate {
				action = candidate
			}
		}
	}
	if action == f.fail {
		return commandResult{}, errors.New("fixture private path and key")
	}
	switch action {
	case "daemon-reload":
		if _, err := os.Stat(f.filePath); err == nil {
			f.loaded = true
			f.fragment = f.filePath
		} else {
			f.loaded = false
			f.fragment = ""
			if !f.active && !f.enabled {
				f.invocation = ""
			}
		}
	case "enable":
		f.enabled = true
	case "disable":
		f.enabled = false
	case "start":
		f.active = true
		f.invocation = strings.Repeat("1", 32)
	case "stop":
		f.active = false
	case "restart":
		f.active = true
		if f.invocation == strings.Repeat("2", 32) {
			f.invocation = strings.Repeat("3", 32)
		} else {
			f.invocation = strings.Repeat("2", 32)
		}
	}
	if f.after != nil {
		f.after(action)
	}
	return commandResult{}, nil
}
func setupService(t *testing.T) (*Manager, *fakeService) {
	t.Helper()
	m := fixture(t)
	spec, err := m.spec()
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeService{filePath: spec.artifact, effectiveArgv: m.argv(), unitPaths: []string{filepath.Dir(spec.artifact), filepath.Join(m.home, "runtime", "systemd", "user"), filepath.Join(m.home, "vendor", "systemd", "user"), filepath.Join(m.home, "runtime", "systemd", "user.control")}}
	m.runner = f.run
	return m, f
}
func mutationCount(f *fakeService) int {
	n := 0
	for _, call := range f.calls {
		if filepath.Base(call[0]) != "busctl" && !strings.Contains(strings.Join(call, " "), " show ") {
			n++
		}
	}
	return n
}
func TestSystemdLifecycleIsExplicitIdempotentAndPreservesClients(t *testing.T) {
	m, f := setupService(t)
	config := filepath.Join(m.codexHome, "config.toml")
	os.WriteFile(config, []byte("fixture-private"), 0600)
	if info := m.Status(context.Background()); info.Status != "Not installed" {
		t.Fatalf("wrong initial status: %s", info.Status)
	}
	if _, err := os.Stat(m.stateRoot); !os.IsNotExist(err) {
		t.Fatal("constructor/status created state")
	}
	dir := filepath.Join(m.home, ".config", "systemd", "user")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	os.Chmod(dir, 0755)
	for _, tc := range []struct {
		action, status string
		mutations      int
	}{{"install", "Stopped", 2}, {"install", "Stopped", 0}, {"start", "Running", 1}, {"start", "Running", 0}, {"restart", "Running", 1}, {"stop", "Stopped", 1}, {"stop", "Stopped", 0}, {"uninstall", "Not installed", 2}, {"uninstall", "Not installed", 0}} {
		before := mutationCount(f)
		info, err := m.Action(context.Background(), tc.action)
		if err != nil || info.Status != tc.status {
			t.Fatalf("%s: status=%s err=%v", tc.action, info.Status, err)
		}
		if mutationCount(f)-before != tc.mutations {
			t.Fatalf("%s repeated or omitted mutations", tc.action)
		}
	}
	if info, _ := os.Stat(dir); info.Mode().Perm() != 0755 {
		t.Fatal("operator directory mode changed")
	}
	raw, _ := os.ReadFile(config)
	if string(raw) != "fixture-private" {
		t.Fatal("client config changed")
	}
	if _, err := os.Stat(filepath.Join(m.stateRoot, "monitor", "ownership.json")); err != nil {
		t.Fatal("uninstall removed application state")
	}
	if _, err := m.Action(context.Background(), "repair"); err == nil {
		t.Fatal("unsupported action accepted")
	}
}
func TestForeignArtifactsAndNativeUnitsAreNeverAdopted(t *testing.T) {
	for _, foreign := range []string{"file", "registered", "symlink"} {
		t.Run(foreign, func(t *testing.T) {
			m, f := setupService(t)
			spec, _ := m.spec()
			os.MkdirAll(filepath.Dir(spec.artifact), 0700)
			switch foreign {
			case "file":
				os.WriteFile(spec.artifact, []byte("foreign-private"), 0600)
			case "registered":
				f.loaded = true
				f.fragment = filepath.Join(m.home, "foreign unit")
			case "symlink":
				target := filepath.Join(t.TempDir(), "foreign")
				os.WriteFile(target, []byte("foreign-private"), 0600)
				if os.Symlink(target, spec.artifact) != nil {
					t.Skip("symlink unavailable")
				}
			}
			info, err := m.Action(context.Background(), "install")
			if err == nil || info.Status != "Conflict" || mutationCount(f) != 0 || strings.Contains(err.Error(), m.home) || strings.Contains(err.Error(), "foreign-private") {
				t.Fatal("foreign artifact adopted/mutated/disclosed")
			}
		})
	}
}
func TestUncertainMutationIsReservedAndNeverBlindlyRetried(t *testing.T) {
	m, f := setupService(t)
	if _, err := m.Action(context.Background(), "install"); err != nil {
		t.Fatal(err)
	}
	f.fail = "start"
	before := mutationCount(f)
	info, err := m.Action(context.Background(), "start")
	if err == nil || info.Status != "Unknown" || strings.Contains(err.Error(), "fixture private") {
		t.Fatal("uncertain command not labelled")
	}
	f.fail = ""
	if _, err := m.Action(context.Background(), "start"); err == nil {
		t.Fatal("uncertain action blindly retried")
	}
	if mutationCount(f) != before+1 {
		t.Fatal("uncertain mutation repeated")
	}
	f.active = true
	f.invocation = fmt.Sprintf("%032x", 99)
	info, err = m.Action(context.Background(), "start")
	if err != nil || info.Status != "Running" || mutationCount(f) != before+1 {
		t.Fatal("confirmed postcondition should reconcile without command")
	}
}
func TestLifecycleRechecksArtifactAfterAwaitAndCancellation(t *testing.T) {
	m, f := setupService(t)
	spec, _ := m.spec()
	f.after = func(action string) {
		if action == "daemon-reload" {
			os.WriteFile(spec.artifact, []byte("foreign after await"), 0600)
		}
	}
	if _, err := m.Action(context.Background(), "install"); err == nil {
		t.Fatal("changed artifact accepted after command boundary")
	}
	if f.enabled {
		t.Fatal("enabled foreign changed artifact")
	}
	data, _ := os.ReadFile(spec.artifact)
	if string(data) != "foreign after await" {
		t.Fatal("foreign change overwritten")
	}
	m, f = setupService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Action(ctx, "install"); err == nil || len(f.calls) != 0 {
		t.Fatal("canceled action ran command")
	}
}
func TestUnitQuotingPinsFrozenCLIAndLiteralPaths(t *testing.T) {
	m := fixture(t)
	m.executable = filepath.Join(filepath.Dir(m.executable), `oneline-ccc 雪%$"\\`)
	spec, err := m.spec()
	if err != nil {
		t.Fatal(err)
	}
	text := string(spec.definition)
	if !strings.Contains(text, `"--state-dir"`) || !strings.Contains(text, `"--codex-home"`) || !strings.Contains(text, `"observe"`) || strings.Contains(text, "--state-root") || strings.Contains(text, "daemon\"") {
		t.Fatal("frozen CLI contract mismatch")
	}
	if !strings.Contains(text, "%%$$") || !strings.Contains(text, `\"`) || strings.Contains(text, "/bin/sh") || strings.Contains(text, "Restart=always") {
		t.Fatal("systemd did not preserve literal paths or used automatic restart")
	}
	m.os = "darwin"
	spec, err = m.spec()
	if err != nil {
		t.Fatal(err)
	}
	text = string(spec.definition)
	if !strings.Contains(text, "<string>--state-dir</string>") || !strings.Contains(text, "<string>observe</string>") || strings.Contains(text, "KeepAlive</key><true") || strings.Contains(text, "/bin/sh") {
		t.Fatal("LaunchAgent contract mismatch")
	}
}
func TestUnavailableWindowsAndMissingManagerAreTruthful(t *testing.T) {
	m, f := setupService(t)
	m.os = "windows"
	if info := m.Status(context.Background()); info.Status != "Unavailable" || !strings.Contains(info.Backend, "Scheduled Task") {
		t.Fatal("unsafe Windows lifecycle advertised")
	}
	if _, err := m.Action(context.Background(), "install"); err == nil || len(f.calls) != 0 {
		t.Fatal("unsafe Windows action ran")
	}
	m = fixture(t)
	m.runner = func(context.Context, string, ...string) (commandResult, error) {
		return commandResult{}, errors.New("private runner failure")
	}
	info, err := m.Action(context.Background(), "install")
	if err == nil || info.Status != "Unavailable" || strings.Contains(err.Error(), "private") {
		t.Fatal("missing manager misreported or leaked")
	}
	if _, err := os.Stat(m.stateRoot); !os.IsNotExist(err) {
		t.Fatal("unavailable manager created state")
	}
}

func TestOwnershipRecordsRejectDuplicateAndTrailingMalformedJSON(t *testing.T) {
	for _, kind := range []string{"duplicate", "trailing"} {
		t.Run(kind, func(t *testing.T) {
			m, f := setupService(t)
			if _, err := m.Action(context.Background(), "install"); err != nil {
				t.Fatal(err)
			}
			spec, _ := m.spec()
			raw := `{"version":1,"definition_sha256":"` + definitionDigest(spec) + `"}`
			if kind == "duplicate" {
				raw = `{"version":1,"version":1,"definition_sha256":"` + definitionDigest(spec) + `"}`
			} else {
				raw += "trailing invalid"
			}
			os.WriteFile(filepath.Join(m.root(), "ownership.json"), []byte(raw), 0600)
			before := mutationCount(f)
			info, err := m.Action(context.Background(), "start")
			if err == nil || info.Status != "Conflict" || mutationCount(f) != before {
				t.Fatal("malformed ownership authorized native mutation")
			}
		})
	}
}
func TestStatusRechecksContextAfterNativeAwait(t *testing.T) {
	m := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.runner = func(context.Context, string, ...string) (commandResult, error) {
		cancel()
		return commandResult{Output: "LoadState=not-found\nActiveState=inactive\nUnitFileState=\nFragmentPath=\nInvocationID=\n"}, nil
	}
	if info := m.Status(ctx); info.Status != "Unavailable" {
		t.Fatal("canceled native query published fresh status")
	}
}

func TestLaunchAgentLifecycleAndLiteralArgv(t *testing.T) {
	m := fixture(t)
	m.os = "darwin"
	m.uid = "1234"
	spec, err := m.spec()
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Dict struct {
			Array struct {
				Values []string `xml:"string"`
			} `xml:"array"`
		} `xml:"dict"`
	}
	if xml.Unmarshal(spec.definition, &document) != nil || !reflect.DeepEqual(document.Dict.Array.Values, m.argv()) {
		t.Fatal("LaunchAgent XML did not preserve exact Unicode/percent/literal argv")
	}
	loaded, active := false, false
	pid := 100
	mutations := 0
	m.runner = func(ctx context.Context, name string, args ...string) (commandResult, error) {
		if name != "/bin/launchctl" || len(args) < 2 {
			t.Fatal("wrong native LaunchAgent runner")
		}
		if args[0] == "print" {
			if !loaded {
				return commandResult{Code: 113}, nil
			}
			phase := "waiting"
			if active {
				phase = "running"
			}
			return commandResult{Output: "job = {\n path = " + spec.artifact + "\n state = " + phase + "\n pid = " + strconv.Itoa(pid) + "\n}"}, nil
		}
		mutations++
		switch args[0] {
		case "bootstrap":
			if !reflect.DeepEqual(args, []string{"bootstrap", "gui/1234", spec.artifact}) {
				t.Fatal("bootstrap paths not literal")
			}
			loaded = true
			active = true
			pid++
		case "kickstart":
			loaded = true
			active = true
			pid++
		case "bootout":
			if !reflect.DeepEqual(args, []string{"bootout", spec.target}) {
				t.Fatal("foreign bootout target")
			}
			loaded = false
			active = false
		default:
			t.Fatal("unexpected LaunchAgent mutation")
		}
		return commandResult{}, nil
	}
	for _, tc := range []struct {
		action, status string
		changes        int
	}{{"install", "Running", 1}, {"install", "Running", 0}, {"start", "Running", 0}, {"restart", "Running", 1}, {"stop", "Stopped", 1}, {"stop", "Stopped", 0}, {"start", "Running", 1}, {"uninstall", "Not installed", 1}, {"uninstall", "Not installed", 0}} {
		before := mutations
		info, err := m.Action(context.Background(), tc.action)
		if err != nil || info.Status != tc.status || mutations-before != tc.changes {
			t.Fatalf("LaunchAgent %s: %s %v", tc.action, info.Status, err)
		}
	}
}
func TestFrozenCLIIsExact(t *testing.T) {
	m := fixture(t)
	expected := []string{m.executable, "--state-dir", m.stateRoot, "--codex-home", m.codexHome, "observe"}
	if !reflect.DeepEqual(m.argv(), expected) || m.interval != 30*time.Second {
		t.Fatal("frozen CLI or sample interval drift")
	}
}
func TestStatusLabelsStaleObservationWithoutReadingConfig(t *testing.T) {
	m, _ := setupService(t)
	if err := m.Observe(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	m.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	// Status must not follow or even inspect the selected client config.
	target := filepath.Join(t.TempDir(), "private")
	os.WriteFile(target, []byte("fixture-private"), 0600)
	if os.Symlink(target, filepath.Join(m.codexHome, "config.toml")) != nil {
		t.Skip("symlink unavailable")
	}
	info := m.Status(context.Background())
	if info.Status != "Not installed" || !strings.Contains(info.Detail, "observation stale") || strings.Contains(info.Detail, m.codexHome) || strings.Contains(info.Detail, "fixture-private") {
		t.Fatal("status lost freshness, read config or disclosed details")
	}
}
func TestMutationReservationSurvivesManagerReconstruction(t *testing.T) {
	m, f := setupService(t)
	if _, err := m.Action(context.Background(), "install"); err != nil {
		t.Fatal(err)
	}
	f.fail = "restart"
	if _, err := m.Action(context.Background(), "restart"); err == nil {
		t.Fatal("missing injected uncertainty")
	}
	before := mutationCount(f)
	next := New(m.stateRoot, m.codexHome, m.executable)
	next.home = m.home
	next.os = m.os
	next.uid = m.uid
	next.runner = f.run
	f.fail = ""
	if _, err := next.Action(context.Background(), "restart"); err == nil || mutationCount(f) != before {
		t.Fatal("new process lost uncertain-mutation reservation")
	}
}
func TestCappedNativeOutputAndSanitizedFailures(t *testing.T) {
	output := &cappedOutput{}
	if _, err := output.Write(make([]byte, recordLimit)); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte("x")); err == nil || len(output.data) > recordLimit || !output.exceeded {
		t.Fatal("native output not bounded")
	}
	m, _ := setupService(t)
	m.runner = func(context.Context, string, ...string) (commandResult, error) {
		return commandResult{Code: 1, Output: "fixture-private-path and synthetic credential"}, nil
	}
	info, err := m.Action(context.Background(), "install")
	if err == nil || strings.Contains(info.Detail, "fixture-private") || strings.Contains(err.Error(), "credential") {
		t.Fatal("command failure escaped sanitizer")
	}
}

func TestStartPostconditionRequiresOwnedArtifactAfterAwait(t *testing.T) {
	m, f := setupService(t)
	if _, err := m.Action(context.Background(), "install"); err != nil {
		t.Fatal(err)
	}
	spec, _ := m.spec()
	f.after = func(action string) {
		if action == "start" {
			os.Remove(spec.artifact)
		}
	}
	info, err := m.Action(context.Background(), "start")
	if err == nil || info.Status != "Unknown" {
		t.Fatal("missing owned artifact accepted after start await")
	}
}
func TestFailedNativeUnitIsNotReportedAsStopped(t *testing.T) {
	m, f := setupService(t)
	if _, err := m.Action(context.Background(), "install"); err != nil {
		t.Fatal(err)
	}
	m.runner = func(ctx context.Context, name string, args ...string) (commandResult, error) {
		result, err := f.run(ctx, name, args...)
		result.Output = strings.Replace(result.Output, "ActiveState=inactive", "ActiveState=failed", 1)
		return result, err
	}
	if info := m.Status(context.Background()); info.Status != "Failed" {
		t.Fatal("failed user service misreported as stopped")
	}
}
func TestNativeInvocationCannotPersistUnvalidatedOutput(t *testing.T) {
	m, f := setupService(t)
	if _, err := m.Action(context.Background(), "install"); err != nil {
		t.Fatal(err)
	}
	f.invocation = "fixture-private-path-and-secret"
	info, err := m.Action(context.Background(), "restart")
	if err == nil || info.Status != "Unavailable" {
		t.Fatal("native output accepted as invocation identity")
	}
	data, _ := os.ReadFile(filepath.Join(m.root(), "pending.json"))
	if strings.Contains(string(data), "fixture-private") {
		t.Fatal("private native output persisted")
	}
}

func TestArtifactRevocationDuringNativeQueryPreventsStart(t *testing.T) {
	m, f := setupService(t)
	if _, err := m.Action(context.Background(), "install"); err != nil {
		t.Fatal(err)
	}
	spec, _ := m.spec()
	queries := 0
	m.runner = func(ctx context.Context, name string, args ...string) (commandResult, error) {
		result, err := f.run(ctx, name, args...)
		if len(args) > 3 && args[3] == "show" {
			queries++
			if queries == 2 {
				os.Remove(spec.artifact)
			}
		}
		return result, err
	}
	before := mutationCount(f)
	info, err := m.Action(context.Background(), "start")
	if err == nil || info.Status != "Conflict" || mutationCount(f) != before {
		t.Fatal("artifact revoked during native query still started")
	}
}
