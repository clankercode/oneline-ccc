package main

// A7's shared driver consumes bytes from a real native console. It never calls
// Model.View or supplies Tea input events. Native execution is intentionally
// confined to the Windows/Darwin files; the parser controls also run on Linux.
import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/clankercode/oneline-ccc/internal/core"
)

const nativeConsoleKey = "a7-dummy-only-never-display-8b21"

// Assigned only by the native files. The parent never opens its own console.
var nativeConsoleModes func() (string, error)
var nativeConsoleRaw func() error
var nativeConsoleCooked func() error
var nativeConsoleAssertRaw func() error
var nativeConsoleContext func(context.Context) (context.Context, func(), error)
var nativeConsoleDiagnostics func(io.Writer)
var nativeConsoleBeforeExit func() error

type nativeConsoleBackend struct {
	*fakeEngine
	forbidden atomic.Int32
	launches  atomic.Int32
	scenario  string
	child     *exec.Cmd
}

func (b *nativeConsoleBackend) Inspect(ctx context.Context) (core.Snapshot, error) {
	if b.scenario != "plain" {
		if err := nativeConsoleAssertRaw(); err != nil {
			return core.Snapshot{}, err
		}
	}
	return b.fakeEngine.Inspect(ctx)
}

func (b *nativeConsoleBackend) refuse() error {
	b.forbidden.Add(1)
	return errors.New("A7 fixture forbids backend effects")
}
func (b *nativeConsoleBackend) Plan(context.Context, string, core.Desired) (core.Plan, error) {
	return core.Plan{}, b.refuse()
}
func (b *nativeConsoleBackend) Apply(context.Context, string, string) (core.Receipt, error) {
	return core.Receipt{}, b.refuse()
}
func (b *nativeConsoleBackend) SaveKey(context.Context, string, bool) error { return b.refuse() }
func (b *nativeConsoleBackend) Discover(context.Context, string) ([]core.ModelInfo, error) {
	return nil, b.refuse()
}
func (b *nativeConsoleBackend) Service(context.Context, string) (core.ServiceInfo, error) {
	return core.ServiceInfo{}, b.refuse()
}
func (b *nativeConsoleBackend) RepairBrief(context.Context) (string, error) {
	return "", b.refuse()
}
func (b *nativeConsoleBackend) Command(ctx context.Context, action, id string) (*exec.Cmd, error) {
	if b.scenario != "handoff" || action != "launch" || id != "codex" || b.launches.Add(1) != 1 {
		return nil, b.refuse()
	}
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeConsoleFixture$")
	cmd.Env = nativeConsoleEnv(os.Getenv("ONELINE_A7_ROOT"), "child", "1")
	b.child = cmd
	return cmd, nil
}

func nativeInputStacks(stacks string) string {
	var found []string
	for _, stack := range strings.Split(stacks, "\n\n") {
		active, _, _ := strings.Cut(stack, "\ncreated by ")
		for _, name := range []string{"(*Program).readLoop(", "(*TerminalReader).StreamEvents(", "(*TerminalReader).sendEvents(", "(*TerminalReader).sendBytes(", "(*TerminalReader).streamData("} {
			if strings.Contains(active, name) {
				found = append(found, active)
				break
			}
		}
	}
	return strings.Join(found, "\n\n")
}

func TestNativeConsoleFixture(t *testing.T) {
	scenario := os.Getenv("ONELINE_A7_FIXTURE")
	if scenario == "" {
		return
	}
	if nativeConsoleModes == nil {
		t.Fatal("native fixture requested on an unsupported host")
	}
	if scenario == "child" {
		watchdog := time.AfterFunc(15*time.Second, func() { os.Exit(84) })
		defer watchdog.Stop()
		if err := nativeConsoleCooked(); err != nil {
			t.Fatal("synthetic child did not receive cooked/foreground terminal:", err)
		}
		if err := nativeConsoleRaw(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(os.Getenv("ONELINE_A7_ROOT"), "child-pid"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(os.Stdout, "A7_SYNTHETIC_RAW_READY")
		var key [1]byte
		if _, err := io.ReadFull(os.Stdin, key[:]); err != nil || key[0] != '!' {
			t.Fatal("synthetic child input:", err)
		}
		// Deliberately leave raw mode behind. Tea must restore/reacquire it.
		return
	}
	root := os.Getenv("ONELINE_A7_ROOT")
	diag, err := os.OpenFile(filepath.Join(root, "diagnostics"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer diag.Close()
	fmt.Fprintf(diag, "FIXTURE scenario=%s stage=diagnostics-open\n", scenario)
	if nativeConsoleDiagnostics != nil {
		nativeConsoleDiagnostics(diag)
	}
	// This is the subprocess's stderr only. stdin/stdout stay native handles.
	previousStderr := os.Stderr
	os.Stderr = diag
	defer func() { os.Stderr = previousStderr }()
	before, err := nativeConsoleModes()
	fmt.Fprintf(diag, "MODE_BEFORE snapshot=%s error=%v\n", before, err)
	if err != nil {
		t.Fatal(err)
	}
	parent, finishControl, err := nativeConsoleContext(context.Background())
	if err != nil {
		fmt.Fprintf(diag, "FAIL stage=cancellation-context error=%v\n", err)
		t.Fatal(err)
	}
	defer finishControl()
	args := []string{"--ascii", "--reduced-motion"}
	if scenario == "plain" {
		args = append(args, "--plain")
	}
	ui := tuiRequested(args, production(context.Background()))
	ctx, stop := watchSignals(parent, ui)
	defer stop()
	d := production(ctx)
	d.userHome = func() (string, error) { return filepath.Join(root, "home"), nil }
	d.userConfig = func() (string, error) { return filepath.Join(root, "config"), nil }
	d.executable = func() (string, error) { return filepath.Join(root, "oneline-ccc"), nil }
	p := core.Defaults()
	p.BaseURL = "https://a7-fixture.invalid/v1"
	b := &nativeConsoleBackend{fakeEngine: &fakeEngine{snapshot: core.Snapshot{
		Clients:    []core.Client{{ID: "codex", Name: "Codex", Installed: true, Supported: true, Version: "0.166.0", Status: "Unverified", Profile: p}},
		Credential: "Session only", Service: core.ServiceInfo{Backend: "fixture", Status: "Stopped"},
	}}, scenario: scenario}
	constructed := 0
	d.newEngine = func(o engineOptions) (engine, error) {
		constructed++
		if o.StateRoot != filepath.Join(root, "config", "oneline-ccc") || o.CodexHome != filepath.Join(root, "home", ".codex") {
			return nil, errors.New("fixture path boundary changed")
		}
		return b, nil
	}
	fmt.Fprintf(diag, "ENTRY stdinTTY=%t stdoutTTY=%t signalUI=%t\n", term.IsTerminal(os.Stdin.Fd()), term.IsTerminal(os.Stdout.Fd()), ui)
	code := run(ctx, args, d)
	after, err := nativeConsoleModes()
	if err != nil || before != after {
		t.Fatalf("console modes not restored before fixture teardown: before=%s after=%s err=%v", before, after, err)
	}
	stacks := make([]byte, 2<<20)
	n := runtime.Stack(stacks, true)
	if n == len(stacks) || nativeInputStacks(string(stacks[:n])) != "" {
		t.Fatalf("input/decoder still active after production return: %s", nativeInputStacks(string(stacks[:n])))
	}
	wantCode, wantLaunch := 0, int32(0)
	if scenario == "cancel" {
		wantCode = 130
	}
	if scenario == "handoff" {
		wantLaunch = 1
		if b.child == nil || b.child.ProcessState == nil || !b.child.ProcessState.Exited() || !b.child.ProcessState.Success() {
			t.Fatal("synthetic child was not successfully waited/reaped before production return")
		}
	}
	if code != wantCode || constructed != 1 || b.inspect != 1 || b.forbidden.Load() != 0 || b.launches.Load() != wantLaunch || b.service+b.command+b.observe+b.recover != 0 {
		t.Fatalf("backend boundary: code=%d engines=%d inspect=%d forbidden=%d launches=%d", code, constructed, b.inspect, b.forbidden.Load(), b.launches.Load())
	}
	fmt.Fprintf(diag, "A7_COMPLETE code=%d engines=1 inspect=1 forbidden=0 launches=%d modes=restored input=joined\n", code, wantLaunch)
	if nativeConsoleBeforeExit != nil {
		if err := nativeConsoleBeforeExit(); err != nil {
			fmt.Fprintf(diag, "FAIL stage=parent-restoration-checkpoint error=%v\n", err)
			t.Fatal(err)
		}
	}
}

func nativeConsoleEnv(root, scenario, noColor string) []string {
	return []string{
		"ONELINE_A7_FIXTURE=" + scenario, "ONELINE_A7_ROOT=" + root,
		"NO_COLOR=" + noColor, "TERM=xterm-256color", "COLORTERM=truecolor", "GOMAXPROCS=2",
		"HOME=" + filepath.Join(root, "home"), "USERPROFILE=" + filepath.Join(root, "home"),
		"APPDATA=" + filepath.Join(root, "config"), "LOCALAPPDATA=" + filepath.Join(root, "config"),
		"XDG_CONFIG_HOME=" + filepath.Join(root, "config"), "XDG_CACHE_HOME=" + filepath.Join(root, "cache"),
		"TMPDIR=" + root, "TMP=" + root, "TEMP=" + root,
	}
}

// poll and exit are bounded, and finish must join/reap before returning. close
// also owns failure cleanup; it is registered before any process is started.
type nativeConsoleHost interface {
	read(time.Duration) ([]byte, error)
	write(string) error
	resize(int, int) error
	exited() (bool, error)
	cancel() error
	close() error
	drained() []byte
}

type nativeConsoleDriver struct {
	host       nativeConsoleHost
	root       string
	deadline   time.Time
	transcript []byte
	screen     *nativeConsoleScreen
}

func nativeConsoleRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"home", "config", "cache"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func nativeConsoleDriverFor(t *testing.T, host nativeConsoleHost, root string) *nativeConsoleDriver {
	t.Helper()
	p := &nativeConsoleDriver{host: host, root: root, deadline: time.Now().Add(25 * time.Second), screen: newNativeConsoleScreen(120, 40)}
	t.Cleanup(func() {
		if err := host.close(); err != nil {
			t.Error("native fixture cleanup:", err)
		}
		if t.Failed() {
			diag, _ := os.ReadFile(filepath.Join(root, "diagnostics"))
			t.Logf("A7 diagnostics: %s\nobserved screen:\n%s\nVT tail: %q", diag, p.screen.text(), string(p.transcript[max(0, len(p.transcript)-4096):]))
		}
	})
	return p
}

func (p *nativeConsoleDriver) read(t *testing.T, wait time.Duration) {
	t.Helper()
	data, err := p.host.read(wait)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatal("native console read:", err)
	}
	if err := p.consume(data); err != nil {
		t.Fatal(err)
	}
}

// Both ordinary reads and the final native-host drain go through this oracle.
// screen.err latches transient cell exposure during feed, before later updates
// can erase it. Raw and stripped transcript checks remain complementary.
func (p *nativeConsoleDriver) consume(data []byte) error {
	if len(data) > 2<<20-len(p.transcript) {
		return errors.New("native console output exceeded fixture bound")
	}
	p.transcript = append(p.transcript, data...)
	p.screen.feed(data)
	if p.screen.err != nil {
		return p.screen.err
	}
	return nativeConsoleOutputError(p.transcript)
}

func (p *nativeConsoleDriver) consumeDrain() error { return p.consume(p.host.drained()) }

func (p *nativeConsoleDriver) await(t *testing.T, from int, wants ...string) {
	t.Helper()
	for {
		if strings.Contains(ansi.Strip(string(p.transcript)), "Local inspection | Gateway A") || strings.Contains(p.screen.text(), "Local inspection | Gateway A") {
			t.Fatal("native NO_COLOR entry fell back to plain status")
		}
		if len(p.transcript) > from && nativeConsoleFrameHas(p.screen.text(), wants...) {
			return
		}
		if time.Now().After(p.deadline) {
			t.Fatalf("native console deadline waiting for %q", wants)
		}
		p.read(t, min(50*time.Millisecond, time.Until(p.deadline)))
		if done, err := p.host.exited(); done || err != nil {
			t.Fatalf("native fixture exited before %q: %v", wants, err)
		}
	}
}

func (p *nativeConsoleDriver) send(t *testing.T, keys string) int {
	t.Helper()
	from := len(p.transcript)
	if err := p.host.write(keys); err != nil {
		t.Fatal("native console input:", err)
	}
	return from
}

func (p *nativeConsoleDriver) palette(t *testing.T, command, want string) {
	t.Helper()
	p.await(t, p.send(t, "\x10"), "Search:")
	p.await(t, p.send(t, command), "Search: "+command)
	p.await(t, p.send(t, "\r"), want)
}

func (p *nativeConsoleDriver) finish(t *testing.T, scenario string) {
	t.Helper()
	for {
		p.read(t, 20*time.Millisecond)
		done, err := p.host.exited()
		if err != nil {
			t.Fatal("native process wait:", err)
		}
		if done {
			break
		}
		if time.Now().After(p.deadline) {
			t.Fatal("native process did not finish and join before deadline")
		}
	}
	// close drains ConPTY while closing it, then closes owned handles. Darwin
	// compares slave termios before closing either side. Both reap exactly once.
	if err := p.host.close(); err != nil {
		t.Fatal("native console final cleanup:", err)
	}
	if err := p.consumeDrain(); err != nil {
		t.Fatal(err)
	}
	code, launches, interactive := 0, 0, scenario != "plain"
	if scenario == "cancel" {
		code = 130
	}
	if scenario == "handoff" {
		launches = 1
	}
	diag, err := os.ReadFile(filepath.Join(p.root, "diagnostics"))
	want := fmt.Sprintf("A7_COMPLETE code=%d engines=1 inspect=1 forbidden=0 launches=%d modes=restored input=joined", code, launches)
	entry := fmt.Sprintf("ENTRY stdinTTY=true stdoutTTY=true signalUI=%t", interactive)
	if err != nil || !strings.Contains(string(diag), entry) || !strings.Contains(string(diag), want) {
		t.Fatalf("native completion proof missing: read=%v diagnostics=%s", err, diag)
	}
	for _, name := range []string{"home", "config", "cache"} {
		entries, err := os.ReadDir(filepath.Join(p.root, name))
		if err != nil || len(entries) != 0 {
			t.Fatalf("disposable %s changed: %v", name, err)
		}
	}
	if scenario != "plain" {
		// Exit controls can arrive during ClosePseudoConsole's final drain; the
		// native host retains those bytes for this final renderer check.
		if !strings.Contains(string(p.transcript), "\x1b[?1049h") || !strings.Contains(string(p.transcript), "\x1b[?1049l") {
			t.Fatal("renderer did not enter/leave the alternate screen before process exit")
		}
	}
}

func nativeConsoleJourney(t *testing.T, p *nativeConsoleDriver, scenario string) {
	t.Helper()
	if scenario == "plain" {
		p.finish(t, scenario)
		// ConPTY synthesizes its own VT transport even for plain stdout. Check
		// observed status and the production signalUI=false completion marker;
		// escape-free application output has separate redirected CLI coverage.
		if !strings.Contains(p.screen.text(), "Local inspection | Gateway A") || strings.Contains(string(p.transcript), "\x1b[?1049h") {
			t.Fatal("--plain control did not produce noninteractive status")
		}
		return
	}
	p.await(t, 0, "Inspected local settings; drafts retained")
	if scenario == "handoff" {
		p.palette(t, "Launch selected client", "A7_SYNTHETIC_RAW_READY")
		p.await(t, p.send(t, "!"), "Client returned; authentication and inference were not verified")
		p.await(t, p.send(t, "?"), "Shortcut help | Page")
		p.await(t, p.send(t, "\x1b"), "CLIENTS")
		p.send(t, "q")
		p.finish(t, scenario)
		return
	}
	p.await(t, p.send(t, "?"), "Shortcut help | Page")
	if scenario == "cancel" {
		// Queue ordinary input plus a deliberately incomplete paste sequence.
		// Cancellation must join native input and decoder even with pending data.
		p.send(t, strings.Repeat("\t", 1024)+"\x1b[200~pending-a7-paste")
		if err := p.host.cancel(); err != nil {
			t.Fatal("external fixture cancellation:", err)
		}
		p.finish(t, scenario)
		return
	}
	p.await(t, p.send(t, "\x1b"), "CLIENTS")
	p.palette(t, "Gateway", "> Edit local endpoint")
	p.await(t, p.send(t, "\t"), "> Enter API key (hidden)")
	p.await(t, p.send(t, "\r"), "> API key: [hidden]")
	p.send(t, nativeConsoleKey)
	p.await(t, p.send(t, "\t"), "Editing key", "> [ ] Persist using backend credential storage")
	p.await(t, p.send(t, "\x1b[Z"), "Editing key", "> API key: [hidden]")
	p.await(t, p.send(t, "\x1b"), "Session-only storage is the default; no key reveal.")
	p.palette(t, "Configure selected client", "> [ ] Manage client")
	p.await(t, p.send(t, " "), "> [x] Manage client")
	// Focus a text editor with actual input; resize via the OS and require its
	// clipped editor value, then restore and continue editing the same draft.
	p.palette(t, "Gateway", "> Enter API key (hidden)")
	p.await(t, p.send(t, "\x1b[Z"), "> Edit local endpoint")
	p.await(t, p.send(t, "\r"), "Editing endpoint")
	endpoint := "https://a7-resize-retains-editor-draft.invalid/v1"
	p.await(t, p.send(t, "\x15"+endpoint), "> "+endpoint+"|")
	from := len(p.transcript)
	p.screen.resize(40, 12)
	if err := p.host.resize(40, 12); err != nil {
		t.Fatal(err)
	}
	p.await(t, from, "> "+endpoint[len(endpoint)-37:]+"|")
	from = len(p.transcript)
	p.screen.resize(120, 40)
	if err := p.host.resize(120, 40); err != nil {
		t.Fatal(err)
	}
	p.await(t, from, "Editing endpoint", "> "+endpoint+"|")
	p.await(t, p.send(t, "x"), "> "+endpoint+"x|")
	p.await(t, p.send(t, "\r"), "Draft updated; review before applying", "> Edit local endpoint")
	p.palette(t, "Configure selected client", "> [x] Manage client")
	p.await(t, p.send(t, "\x03"), "Leave with unapplied drafts?", "> Back; keep working")
	p.await(t, p.send(t, "\x1b"), "> [x] Manage client")
	p.await(t, p.send(t, "q"), "Leave with unapplied drafts?")
	p.await(t, p.send(t, "\t"), "> Confirm")
	p.send(t, "\r")
	p.finish(t, scenario)
}

func nativeConsoleFrameHas(frame string, wants ...string) bool {
	for _, want := range wants {
		if !strings.Contains(frame, want) {
			return false
		}
	}
	return true
}

func nativeConsoleOutputError(raw []byte) error {
	text := string(raw)
	if strings.Contains(text, nativeConsoleKey) || strings.Contains(ansi.Strip(text), nativeConsoleKey) {
		return errors.New("dummy credential leaked into console bytes")
	}
	var color bool
	p := ansi.NewParser()
	p.SetHandler(ansi.Handler{HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
		if cmd.Final() != 'm' {
			return
		}
		for i := range params {
			n, _, _ := params.Param(i, 0)
			if n >= 30 && n <= 38 || n >= 40 && n <= 48 || n == 58 || n >= 90 && n <= 97 || n >= 100 && n <= 107 {
				color = true
			}
		}
	}})
	for _, b := range raw {
		p.Advance(b)
	}
	if color {
		return errors.New("NO_COLOR console emitted color SGR")
	}
	return nil
}
