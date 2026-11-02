package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/clankercode/oneline-ccc/internal/core"
	"golang.org/x/sys/unix"
)

const noColorDummyKey = "a5b-dummy-key-never-display-7f3a"

// Run the real production entry, signal selection, foreground lifetime and Tea
// renderer. Only the engine constructor and path resolvers are substituted.
func TestNoColorTerminalFixture(t *testing.T) {
	if os.Getenv("ONELINE_NO_COLOR_TERMINAL_FIXTURE") != "1" {
		return
	}
	root := os.Getenv("ONELINE_NO_COLOR_ROOT")
	var args []string
	if os.Getenv("ONELINE_NO_COLOR_MODE") == "plain" {
		args = []string{"--plain"}
	}
	ui := tuiRequested(args, production(context.Background()))
	ctx, stop := watchSignals(context.Background(), ui)
	defer stop()
	d := production(ctx)
	d.userHome = func() (string, error) { return filepath.Join(root, "home"), nil }
	d.userConfig = func() (string, error) { return filepath.Join(root, "config"), nil }
	d.executable = func() (string, error) { return filepath.Join(root, "oneline-ccc"), nil }
	profile := core.Defaults()
	profile.BaseURL = "https://no-color.invalid/v1"
	f := &fakeEngine{snapshot: core.Snapshot{
		Clients:    []core.Client{{ID: "codex", Name: "Codex", Installed: true, Supported: true, Version: "0.166.0", Status: "Unverified", Profile: profile}},
		Credential: "Session only", Service: core.ServiceInfo{Backend: "fixture", Status: "Stopped"},
	}}
	constructed := 0
	d.newEngine = func(engineOptions) (engine, error) { constructed++; return f, nil }
	fmt.Fprintf(os.Stderr, "ENTRY stdinTTY=%t stdoutTTY=%t signalUI=%t\n", term.IsTerminal(os.Stdin.Fd()), term.IsTerminal(os.Stdout.Fd()), ui)
	code := run(ctx, args, d)
	// Unimplemented Backend methods are deliberately unavailable: saving a key,
	// planning/applying or discovering cannot reach a vault, client or network.
	if code != 0 || constructed != 1 || f.inspect != 1 || f.command+f.service+f.observe+f.recover != 0 {
		t.Fatalf("unexpected backend activity: code=%d constructors=%d inspections=%d", code, constructed, f.inspect)
	}
	fmt.Fprintln(os.Stderr, "READ_ONLY_BACKEND_CONFIRMED")
}

type noColorPTY struct {
	master, slave *os.File
	app           *exec.Cmd
	cancel        context.CancelFunc
	before        *unix.Termios
	deadline      time.Time
	root          string
	transcript    []byte
	screen        *noColorScreen
	waited        bool
}

func startNoColorPTY(t *testing.T, mode, value string) *noColorPTY {
	t.Helper()
	p := &noColorPTY{root: t.TempDir(), screen: newNoColorScreen()}
	// Direct poll/read below avoids a read goroutine and makes every wait bounded.
	var err error
	p.master, err = os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if p.app != nil && p.app.Process != nil && !p.waited {
			_ = p.app.Process.Kill()
			_ = p.app.Wait()
		}
		if p.cancel != nil {
			p.cancel()
		}
		if p.slave != nil {
			if p.before != nil {
				if err := unix.IoctlSetTermios(int(p.slave.Fd()), unix.TCSETS, p.before); err != nil {
					t.Error("restore PTY during cleanup:", err)
				}
			}
			_ = p.slave.Close()
		}
		_ = p.master.Close()
	})
	if err = unix.IoctlSetPointerInt(int(p.master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(int(p.master.Fd()), unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	p.slave, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = unix.IoctlSetWinsize(int(p.slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 40, Col: 120}); err != nil {
		t.Fatal(err)
	}
	p.before, err = unix.IoctlGetTermios(int(p.slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"home", "config"} {
		if err := os.Mkdir(filepath.Join(p.root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	file := func(name string) *os.File {
		t.Helper()
		f, err := os.OpenFile(filepath.Join(p.root, name), os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		return f
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	p.cancel = cancel
	p.deadline, _ = ctx.Deadline()
	p.app = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNoColorTerminalFixture$")
	p.app.Env = []string{
		"ONELINE_NO_COLOR_TERMINAL_FIXTURE=1", "ONELINE_NO_COLOR_ROOT=" + p.root, "ONELINE_NO_COLOR_MODE=" + mode,
		"NO_COLOR=" + value, "TERM=xterm-256color", "COLORTERM=truecolor", "GOMAXPROCS=1",
		"HOME=" + filepath.Join(p.root, "home"), "XDG_CONFIG_HOME=" + filepath.Join(p.root, "config"), "TMPDIR=" + p.root,
	}
	p.app.Stdin, p.app.Stdout, p.app.Stderr = p.slave, p.slave, file("diagnostics")
	ctty := 0
	if mode == "redirected-stdin" {
		p.app.Stdin = file("stdin")
		ctty = 1
	}
	if mode == "redirected-stdout" {
		p.app.Stdout = file("stdout")
	}
	p.app.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: ctty}
	if err := p.app.Start(); err != nil {
		t.Fatal(err)
	}
	return p
}

func (p *noColorPTY) read(t *testing.T, wait time.Duration) bool {
	t.Helper()
	fds := []unix.PollFd{{Fd: int32(p.master.Fd()), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, int((wait+time.Millisecond-1)/time.Millisecond))
	if errors.Is(err, unix.EINTR) || n == 0 && err == nil {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8192)
	n, err = unix.Read(int(p.master.Fd()), buf)
	if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
		return false
	}
	if err != nil || n == 0 {
		t.Fatalf("PTY read ended: %v", err)
	}
	p.transcript = append(p.transcript, buf[:n]...)
	if len(p.transcript) > 1<<20 {
		t.Fatal("unexpectedly large PTY transcript")
	}
	// Ordinary reads and finish's final drain share this incremental oracle.
	// Feed the complete batch so a later clear cannot hide an earlier exposure.
	p.screen.feed(buf[:n])
	if p.screen.err != nil {
		t.Fatal(p.screen.err)
	}
	return true
}

func (p *noColorPTY) await(t *testing.T, from int, want string) {
	t.Helper()
	for {
		if strings.Contains(ansi.Strip(string(p.transcript)), "Local inspection | Gateway A") {
			t.Fatal("NO_COLOR production entry fell back to read-only status instead of the TUI")
		}
		text := p.screen.text()
		if len(p.transcript) > from && strings.Contains(text, want) {
			return
		}
		remaining := time.Until(p.deadline)
		if remaining <= 0 {
			t.Fatalf("PTY deadline waiting for %q; output: %q", want, strings.ReplaceAll(text, noColorDummyKey, "[dummy hidden]"))
		}
		p.read(t, remaining)
	}
}

// Reconstruct observed cells rather than concatenating renderer diffs: Tea can
// replace just the changed letters of a label. The existing ANSI parser handles
// sequences split across reads. This fixed-size ASCII fixture retains both its
// parser state and the first exposure/unsupported-operation error across reads.
type noColorScreen struct {
	cells  [40][120]rune
	parser *ansi.Parser
	err    error
}

func newNoColorScreen() *noColorScreen {
	s := &noColorScreen{}
	cells := &s.cells
	x, y, savedX, savedY := 0, 0, 0, 0
	top, bottom, wrap := 0, 39, true
	blank := [120]rune{}
	var last rune
	scroll := func(first, n int, down bool) {
		first = max(top, min(bottom, first))
		n = max(1, min(bottom-first+1, n))
		if down {
			copy(cells[first+n:bottom+1], cells[first:bottom+1-n])
			for row := first; row < first+n; row++ {
				cells[row] = blank
			}
		} else {
			copy(cells[first:bottom+1-n], cells[first+n:bottom+1])
			for row := bottom + 1 - n; row <= bottom; row++ {
				cells[row] = blank
			}
		}
	}
	newline := func() {
		if y == bottom {
			scroll(top, 1, false)
		} else {
			y = min(39, y+1)
		}
	}
	print := func(r rune) {
		if x >= 120 {
			if wrap {
				x = 0
				newline()
			} else {
				x = 119
			}
		}
		cells[y][x], last = r, r
		s.latchKeyExposure(y)
		x++
	}
	p := ansi.NewParser()
	s.parser = p
	p.SetHandler(ansi.Handler{
		Print: print,
		Execute: func(b byte) {
			switch b {
			case '\r':
				x = 0
			case '\n', '\v', '\f':
				newline()
			case '\b':
				x = max(0, x-1)
			case '\t':
				x = min(119, (x/8+1)*8)
			}
		},
		HandleEsc: func(cmd ansi.Cmd) {
			switch cmd.Final() {
			case '7':
				savedX, savedY = x, y
			case '8':
				x, y = savedX, savedY
			case 'M':
				if y == top {
					scroll(top, 1, true)
				} else {
					y = max(0, y-1)
				}
			case 'D', 'E':
				newline()
				if cmd.Final() == 'E' {
					x = 0
				}
			}
		},
		HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
			param := func(i, def int) int { n, _, _ := params.Param(i, def); return n }
			n := max(1, param(0, 1))
			x = min(119, x)
			switch cmd.Final() {
			case 'H', 'f':
				y, x = max(0, min(39, n-1)), max(0, min(119, param(1, 1)-1))
			case 'A':
				y = max(0, y-n)
			case 'B', 'e':
				y = min(39, y+n)
			case 'C', 'a':
				x = min(119, x+n)
			case 'D':
				x = max(0, x-n)
			case 'G', '`':
				x = min(119, n-1)
			case 'd':
				y = min(39, n-1)
			case 'I':
				x = min(119, (x/8+n)*8)
			case 'Z':
				x = max(0, ((x-1)/8-n+1)*8)
			case 'J', 'K', 'X':
				start, end := y*120+x, 40*120
				if cmd.Final() == 'K' {
					end = (y + 1) * 120
				}
				if cmd.Final() == 'X' {
					end = min((y+1)*120, start+n)
				} else if mode := param(0, 0); mode == 1 {
					start, end = 0, start+1
					if cmd.Final() == 'K' {
						start = y * 120
					}
				} else if mode >= 2 {
					start = 0
					if cmd.Final() == 'K' {
						start = y * 120
					}
				}
				for i := start; i < end; i++ {
					cells[i/120][i%120] = 0
				}
			case 'L', 'M':
				scroll(y, n, cmd.Final() == 'L')
			case 'S', 'T':
				scroll(top, n, cmd.Final() == 'T')
			case 'P', '@':
				n = min(n, 120-x)
				if cmd.Final() == 'P' {
					copy(cells[y][x:120-n], cells[y][x+n:])
					clear(cells[y][120-n:])
				} else {
					copy(cells[y][x+n:], cells[y][x:120-n])
					clear(cells[y][x : x+n])
				}
				s.latchKeyExposure(y)
			case 'b':
				for range n {
					print(last)
				}
			case 'r':
				top, bottom, x, y = min(39, n-1), min(39, param(1, 40)-1), 0, 0
			case 's':
				savedX, savedY = x, y
			case 'u':
				if cmd.Prefix() == 0 { // Prefixed variants negotiate the Kitty keyboard protocol.
					x, y = savedX, savedY
				}
			case 'h', 'l':
				if cmd.Prefix() == '?' && n == 7 {
					wrap = cmd.Final() == 'h'
				}
			case 'm', 'c', 'n', 'p', 'q', 't', 'W', 'g': // Styling, queries and tab-stop setup do not move cells.
			default:
				if s.err == nil {
					s.err = fmt.Errorf("unsupported terminal cursor operation %q", cmd.Final())
				}
			}
		},
	})
	return s
}

// Only printing/repeating and horizontal insert/delete can assemble new text
// within a row. Check that affected row before subsequent mutation erases it;
// whole-row scrolling and cell erasure cannot create a new contiguous key.
func (s *noColorScreen) latchKeyExposure(y int) {
	if s.err != nil {
		return
	}
	row := &s.cells[y]
	for start := 0; start+len(noColorDummyKey) <= len(row); start++ {
		matched := true
		for i := range noColorDummyKey {
			if row[start+i] != rune(noColorDummyKey[i]) {
				matched = false
				break
			}
		}
		if matched {
			s.err = errors.New("dummy credential exposed in terminal cells")
			return
		}
	}
}

func (s *noColorScreen) feed(data []byte) {
	for _, b := range data {
		s.parser.Advance(b)
	}
}

func (s *noColorScreen) text() string {
	var text strings.Builder
	for _, row := range s.cells {
		for _, r := range row {
			if r == 0 {
				r = ' '
			}
			text.WriteRune(r)
		}
		text.WriteByte('\n')
	}
	return text.String()
}

func (p *noColorPTY) send(t *testing.T, keys string) int {
	t.Helper()
	from := len(p.transcript)
	if _, err := p.master.WriteString(keys); err != nil {
		t.Fatal(err)
	}
	return from
}

func (p *noColorPTY) palette(t *testing.T, command, want string) {
	t.Helper()
	p.await(t, p.send(t, "\x10"), "Search:")
	p.await(t, p.send(t, command+"\r"), want)
}

func (p *noColorPTY) assertKeyHidden(t *testing.T, focus string) {
	t.Helper()
	if p.screen.err != nil {
		t.Fatal(p.screen.err)
	}
	frame := p.screen.text()
	hasEditorTitle := false
	for _, line := range strings.Split(frame, "\n") {
		if strings.TrimSpace(line) == "Editing key" {
			hasEditorTitle = true
		}
	}
	if !hasEditorTitle || !strings.Contains(frame, "API key: [hidden]") || !strings.Contains(frame, focus) {
		t.Fatal("credential editor or expected focus missing before cancellation")
	}
	for _, text := range []string{frame, string(p.transcript), ansi.Strip(string(p.transcript))} {
		if strings.Contains(text, noColorDummyKey) {
			t.Fatal("dummy key visible before credential cancellation")
		}
	}
}

func (p *noColorPTY) finish(t *testing.T, entry string) {
	t.Helper()
	err := p.app.Wait()
	p.waited = true
	for p.read(t, 0) {
	}
	diagnostics, readErr := os.ReadFile(filepath.Join(p.root, "diagnostics"))
	if err != nil || readErr != nil || !strings.Contains(string(diagnostics), entry) || !strings.Contains(string(diagnostics), "READ_ONLY_BACKEND_CONFIRMED") {
		t.Fatalf("fixture exit=%v diagnostics read=%v: %s", err, readErr, diagnostics)
	}
	after, err := unix.IoctlGetTermios(int(p.slave.Fd()), unix.TCGETS)
	if err != nil || !reflect.DeepEqual(p.before, after) {
		t.Fatalf("original terminal modes not restored before cleanup: %v", err)
	}
	for _, path := range []string{"home", "config"} {
		entries, err := os.ReadDir(filepath.Join(p.root, path))
		if err != nil || len(entries) != 0 {
			t.Fatalf("disposable %s changed: %v", path, err)
		}
	}
}

func noColorSGR(transcript string) string {
	for _, sequence := range regexp.MustCompile(`\x1b\[([0-9;:]*)m`).FindAllStringSubmatch(transcript, -1) {
		for _, parameter := range strings.Split(sequence[1], ";") {
			code, _, _ := strings.Cut(parameter, ":")
			n, _ := strconv.Atoi(code)
			if n >= 30 && n <= 38 || n >= 40 && n <= 48 || n == 58 || n >= 90 && n <= 97 || n >= 100 && n <= 107 {
				return sequence[0]
			}
		}
	}
	return ""
}

func TestNoColorTerminalInteractive(t *testing.T) {
	for _, value := range []string{"", "1"} {
		t.Run("value="+value, func(t *testing.T) {
			p := startNoColorPTY(t, "interactive", value)
			p.await(t, 0, "Inspected local settings; drafts retained")
			during, err := unix.IoctlGetTermios(int(p.slave.Fd()), unix.TCGETS)
			if err != nil || during.Lflag&(unix.ICANON|unix.ECHO|unix.ISIG) != 0 {
				t.Fatal("production TUI did not enter raw mode", err)
			}
			p.await(t, p.send(t, "?"), "Shortcut help | Page")
			p.await(t, p.send(t, "\x1b"), "CLIENTS")
			p.palette(t, "Gateway", "> Edit local endpoint")
			// Move between adjacent local controls, independent of the top-level
			// tab count (including the upcoming More tab).
			p.await(t, p.send(t, "\t"), "> Enter API key (hidden)")
			p.await(t, p.send(t, "\r"), "> API key: [hidden]")
			// Masking deliberately has no visible length/value feedback. A focus
			// change after typing forces an observable editor update before Esc;
			// these checks do not assert that the hidden value was accepted.
			p.send(t, noColorDummyKey)
			p.await(t, p.send(t, "\t"), "> [ ] Persist using backend credential storage")
			p.assertKeyHidden(t, "> [ ] Persist using backend credential storage")
			p.await(t, p.send(t, "\x1b[Z"), "> API key: [hidden]")
			p.assertKeyHidden(t, "> API key: [hidden]")
			p.await(t, p.send(t, "\x1b"), "> Enter API key (hidden)")
			p.palette(t, "Configure selected client", "> [ ] Manage client")
			p.send(t, " ")
			p.palette(t, "Dashboard", "Draft pending / applied values shown above")
			p.await(t, p.send(t, "\x03"), "Leave with unapplied drafts?")
			p.await(t, p.send(t, "\x1b"), "CLIENTS")
			p.palette(t, "Configure selected client", "> [x] Manage client")
			p.await(t, p.send(t, "q"), "Leave with unapplied drafts?")
			p.send(t, "\t\r") // Confirm the guarded, normal exit.
			p.finish(t, "ENTRY stdinTTY=true stdoutTTY=true signalUI=true")
			transcript := string(p.transcript)
			if strings.Contains(transcript, noColorDummyKey) || strings.Contains(ansi.Strip(transcript), noColorDummyKey) {
				t.Fatal("ordinary masked typing leaked the dummy key")
			}
			if sgr := noColorSGR(transcript); sgr != "" {
				t.Fatalf("NO_COLOR renderer emitted color SGR %q", sgr)
			}
			if !strings.Contains(transcript, "\x1b[?1049h") || !strings.Contains(transcript, "\x1b[?1049l") {
				t.Fatal("real renderer did not enter and leave the alternate screen")
			}
		})
	}
}

func TestNoColorTerminalPlainAndRedirected(t *testing.T) {
	for _, tc := range []struct{ mode, entry string }{
		{"plain", "ENTRY stdinTTY=true stdoutTTY=true signalUI=false"},
		{"redirected-stdin", "ENTRY stdinTTY=false stdoutTTY=true signalUI=false"},
		{"redirected-stdout", "ENTRY stdinTTY=true stdoutTTY=false signalUI=false"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			p := startNoColorPTY(t, tc.mode, "1")
			p.finish(t, tc.entry)
			output := p.transcript
			if tc.mode == "redirected-stdout" {
				var err error
				output, err = os.ReadFile(filepath.Join(p.root, "stdout"))
				if err != nil {
					t.Fatal(err)
				}
			}
			if strings.ContainsAny(string(output)+string(p.transcript), "\x1b") || !strings.Contains(string(output), "Local inspection | Gateway A") || !strings.Contains(string(output), "Credential: Session only") {
				t.Fatal("plain/redirected entry did not produce escape-free read-only status")
			}
		})
	}
}

func TestNoColorTerminalSGRDetection(t *testing.T) {
	for _, sequence := range []string{"\x1b[31m", "\x1b[44m", "\x1b[38;2;1;2;3m", "\x1b[48:2::1:2:3m", "\x1b[58;5;4m", "\x1b[91m", "\x1b[107m"} {
		if noColorSGR(sequence) == "" {
			t.Fatalf("missed color SGR %q", sequence)
		}
	}
	if sgr := noColorSGR("\x1b[?1049h\x1b[2J\x1b[5;1H\x1b[0;39;49;59m\x1b[?1049l"); sgr != "" {
		t.Fatalf("rejected cursor controls or color resets: %q", sgr)
	}
}

func TestNoColorTerminalScreenReadsCellDiffs(t *testing.T) {
	raw := "\x1b[HSession-only storage\x1b[3Garch:\x1b[K\r\n  First\r\n> Second\x1b[2;1H>\x1b[3;1H "
	s := newNoColorScreen()
	s.feed([]byte(raw))
	if s.err != nil {
		t.Fatal(s.err)
	}
	screen := s.text()
	if !strings.Contains(screen, "Search:") || !strings.Contains(screen, "> First") || strings.Contains(screen, "Session-only") || strings.Contains(screen, "> Second") {
		t.Fatal("partial label/erase/focus updates were not reconstructed from terminal bytes")
	}
}
