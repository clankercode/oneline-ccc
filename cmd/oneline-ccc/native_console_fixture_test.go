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

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/tui"
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
var nativeConsolePrepare func() (func() error, error)

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
	if nativeConsolePrepare != nil {
		restore, err := nativeConsolePrepare()
		if err != nil {
			fmt.Fprintf(diag, "FAIL stage=bind-private-console error=%v\n", err)
			t.Fatal(err)
		}
		defer func() {
			if err := restore(); err != nil {
				t.Error("fixture console cleanup:", err)
			}
		}()
		fmt.Fprintf(diag, "PREPARED stdinTTY=%t stdoutTTY=%t\n", term.IsTerminal(os.Stdin.Fd()), term.IsTerminal(os.Stdout.Fd()))
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
	if scenario == "cancel" || scenario == "abort-cleanup" {
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
		"NO_COLOR=" + noColor, "TERM=xterm-256color", "COLORTERM=truecolor", "GOMAXPROCS=1",
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

// One existing waiter owns process reaping. Drain must be nonblocking and is
// serviced until that waiter completes or the phase expires. A drain error
// remains a failure even if process exit subsequently succeeds.
func nativeConsoleWaitAndDrain(wait <-chan error, limit time.Duration, drain func() error) (bool, error, error) {
	timer := time.NewTimer(limit)
	defer timer.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	var drainErr error
	for {
		select {
		case err := <-wait:
			return true, err, drainErr
		case <-timer.C:
			return false, nil, drainErr
		case <-tick.C:
			if drain != nil {
				if err := drain(); err != nil && drainErr == nil {
					drainErr = err
				}
			}
		}
	}
}

func TestNativeConsoleCleanupWait(t *testing.T) {
	t.Run("unresponsive-waiter", func(t *testing.T) {
		wait := make(chan error, 1)
		fallback := time.AfterFunc(500*time.Millisecond, func() { wait <- nil })
		defer fallback.Stop()
		start := time.Now()
		done, _, err := nativeConsoleWaitAndDrain(wait, 20*time.Millisecond, nil)
		if done || err != nil {
			t.Fatalf("unresponsive waiter was not bounded: done=%t err=%v elapsed=%s", done, err, time.Since(start))
		}
	})
	t.Run("exit-needs-drain", func(t *testing.T) {
		wait := make(chan error, 1)
		fallback := time.AfterFunc(500*time.Millisecond, func() { wait <- errors.New("drain was never serviced") })
		defer fallback.Stop()
		drained := false
		done, exitErr, drainErr := nativeConsoleWaitAndDrain(wait, 100*time.Millisecond, func() error {
			if !drained {
				drained = true
				wait <- nil
			}
			return nil
		})
		if !done || exitErr != nil || drainErr != nil || !drained {
			t.Fatalf("wait prevented output drain: done=%t exit=%v drain=%v drained=%t", done, exitErr, drainErr, drained)
		}
	})
	t.Run("drain-error-remains-failure", func(t *testing.T) {
		wait := make(chan error, 1)
		fault := errors.New("fixture drain failed")
		fallback := time.AfterFunc(500*time.Millisecond, func() { wait <- nil })
		defer fallback.Stop()
		sent := false
		done, exitErr, drainErr := nativeConsoleWaitAndDrain(wait, 100*time.Millisecond, func() error {
			if !sent {
				sent = true
				wait <- nil
			}
			return fault
		})
		if !done || exitErr != nil || !errors.Is(drainErr, fault) {
			t.Fatalf("drain failure lost on normal exit: done=%t exit=%v drain=%v", done, exitErr, drainErr)
		}
	})
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
	p.await(t, p.send(t, "\x1b"), "> Enter API key (hidden)")
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
	p.await(t, from, "\n"+nativeConsoleEndpointRow(40, endpoint, len(endpoint))+"\n")
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

// Independent ASCII editor geometry: components.go documents symmetric shell
// insets of one column below 60 columns, two otherwise. The focus prefix uses
// two body columns and the caret uses one. Never derive this from observed text.
func nativeConsoleEndpointRow(width int, value string, caret int) string {
	inset := 1
	if width >= 60 {
		inset = 2
	}
	capacity := max(0, width-2*inset-len("> ")-len("|"))
	start := max(0, caret-capacity)
	end := min(len(value), caret+capacity-(caret-start))
	row := strings.Repeat(" ", inset) + "> " + value[start:caret] + "|" + value[caret:end]
	return row + strings.Repeat(" ", max(0, width-len(row)))
}

// This source regression drives the public production model separately from
// the native byte driver. It distinguishes a stale viewport oracle from input
// loss and keeps complete-value/draft checks after returning to a wide window.
func TestNativeConsoleEndpointViewport(t *testing.T) {
	for _, noColor := range []bool{false, true} {
		t.Run(fmt.Sprintf("NoColor=%t", noColor), func(t *testing.T) {
			profile := core.Defaults()
			profile.BaseURL = "https://a7-fixture.invalid/v1"
			b := &nativeConsoleBackend{fakeEngine: &fakeEngine{snapshot: core.Snapshot{
				Clients: []core.Client{{ID: "codex", Name: "Codex", Installed: true, Supported: true, Profile: profile}},
			}}, scenario: "plain"}
			m := tui.New(b, tui.Options{ASCII: true, NoColor: noColor, ReducedMotion: true})
			send := func(msg tea.Msg) {
				t.Helper()
				var cmd tea.Cmd
				m, cmd = m.Update(msg)
				if cmd != nil {
					t.Fatal("unexpected external work during editor navigation")
				}
			}
			key := func(code rune, text string, mod tea.KeyMod) {
				t.Helper()
				send(tea.KeyPressMsg{Code: code, Text: text, Mod: mod})
			}
			send(m.Init()())
			send(tea.WindowSizeMsg{Width: 120, Height: 40})
			key('p', "", tea.ModCtrl)
			key('G', "Gateway", 0)
			key(tea.KeyEnter, "", 0)
			key(tea.KeyEnter, "", 0)
			key('u', "", tea.ModCtrl)
			value := "https://a7-resize-retains-editor-draft.invalid/v1"
			for _, r := range value {
				key(r, string(r), 0)
			}
			assertRow := func(width, height, caret int) {
				t.Helper()
				send(tea.WindowSizeMsg{Width: width, Height: height})
				frame := ansi.Strip(m.View().Content)
				want := "\n" + nativeConsoleEndpointRow(width, value, caret) + "\n"
				if !nativeConsoleFrameHas(frame, "Editing endpoint", want) {
					t.Fatalf("%dx%d caret=%d expected %q in production frame:\n%s", width, height, caret, want, frame)
				}
			}
			assertRow(120, 40, len(value))
			for _, size := range [][2]int{{40, 12}, {59, 20}, {60, 20}, {120, 40}} {
				assertRow(size[0], size[1], len(value))
			}
			key(tea.KeyLeft, "", 0)
			assertRow(40, 12, len(value)-1)
			assertRow(120, 40, len(value)-1)
			key(tea.KeyEnd, "", 0)
			key('x', "x", 0)
			value += "x"
			assertRow(120, 40, len(value))
			key(tea.KeyEnter, "", 0)
			if !strings.Contains(ansi.Strip(m.View().Content), "Draft updated; review before applying") {
				t.Fatal("editor did not commit the draft")
			}
			key(tea.KeyEnter, "", 0)
			assertRow(120, 40, len(value))
			if b.inspect != 1 || b.forbidden.Load() != 0 || b.launches.Load() != 0 || b.snapshot.Clients[0].Profile != profile {
				t.Fatal("viewport journey changed backend state or performed external work")
			}
		})
	}
}

func TestNativeConsoleEndpointViewportReplay(t *testing.T) {
	// Exact final clear/redraw from both interactive NO_COLOR cases in hosted
	// Windows job 41537608013. The two retained VT tails have identical redraws.
	const redraw = "\x1b[2J\x1b[H oneline-ccc / Gateway / Edit endpoint  \r\n" +
		" Dashboard Models Review Jobs [More]    \r\n" +
		" Inspected local settings; drafts re... \r\n" +
		" Editing endpoint | Esc keeps the       \r\n" +
		" previous draft\x1b[K\r\n" +
		" Local-only endpoint. Public views use  \r\n" +
		" Gateway A.\x1b[K\r\n" +
		" > ize-retains-editor-draft.invalid/v1| \r\n" +
		" Enter commits this field to the draft  \r\n" +
		" 1-6/7 PgUp/Dn\x1b[K\r\n" +
		" Type / paste  Enter save  Esc cancel   \r\n" +
		" Left/Right caret; text keys stay here  "
	value := "https://a7-resize-retains-editor-draft.invalid/v1"
	row := nativeConsoleEndpointRow(40, value, len(value))
	if row != " > ize-retains-editor-draft.invalid/v1| " || len(row) != 40 {
		t.Fatalf("incorrect independent 40-column viewport: %q", row)
	}
	for _, chunk := range []int{len(redraw), 1} {
		s := newNativeConsoleScreen(120, 40)
		s.resize(40, 12)
		for start := 0; start < len(redraw); start += chunk {
			s.feed([]byte(redraw[start:min(len(redraw), start+chunk)]))
		}
		want := "\n" + row + "\n"
		frame := s.text()
		if s.err != nil || !nativeConsoleFrameHas(frame, want) {
			t.Fatalf("chunk=%d: recorded native redraw mismatch: %v\n%s", chunk, s.err, frame)
		}
		for _, tc := range []struct{ name, replacement string }{
			{"old-terminal-width", " > esize-retains-editor-draft.invalid/v1|"},
			{"dropped-character", strings.Replace(row, "ize", "ze", 1) + " "},
			{"altered-character", strings.Replace(row, "draft", "drXft", 1)},
			{"missing-caret", strings.Replace(row, "|", " ", 1)},
			{"misplaced-caret", strings.Replace(row, "v1|", "v|1", 1)},
			{"shifted-column", " " + row[:len(row)-1]},
		} {
			bad := strings.Replace(frame, row, tc.replacement, 1)
			if nativeConsoleFrameHas(bad, want) {
				t.Errorf("chunk=%d: %s corruption accepted", chunk, tc.name)
			}
		}
	}
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
