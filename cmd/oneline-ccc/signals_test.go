package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/tui"
)

type commandBackend struct {
	core.Backend
	command func(context.Context) (*exec.Cmd, error)
}

func (b commandBackend) Command(ctx context.Context, _, _ string) (*exec.Cmd, error) {
	return b.command(ctx)
}
func signalFixture(ui bool) {
	ctx, stop := watchSignals(context.Background(), ui)
	defer stop()
	mode := "wait"
	if !ui {
		mode = "ready-wait"
	}
	backend := commandBackend{command: func(ctx context.Context) (*exec.Cmd, error) {
		c := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIChildFixture$", "--", "ccc-fixture:"+mode)
		c.Env = []string{}
		return c, nil
	}}
	if !ui {
		c, _ := backend.Command(ctx, "launch", "codex")
		// READY comes from the running child through the actual CLI's
		// forwarded stdout. Its stdin pipe stays open until fixture cleanup.
		d := dependencies{in: os.Stdin, out: os.Stdout, errout: os.Stderr, runChild: runProcess}
		code := launch(ctx, flags{command: "launch"}, &fakeEngine{cmd: c}, d)
		if code != 130 {
			fmt.Fprintf(os.Stdout, "WRONG_DIRECT_RESULT exit=%d canceled=%t\n", code, ctx.Err() != nil)
			os.Exit(9)
		}
		fmt.Fprintln(os.Stdout, "CLI_CANCELLED")
		return
	}
	caller, cancel := context.WithCancel(context.Background())
	defer cancel()
	bound := rootBoundBackend{Backend: backend, root: ctx}
	child, err := bound.Command(caller, "launch", "codex")
	if err != nil {
		os.Exit(8)
	}
	if child.Start() != nil {
		os.Exit(7)
	}
	fmt.Fprintf(os.Stdout, "READY %d\n", child.Process.Pid)
	_ = child.Wait()
	// Allow the root signal handler to run; an interrupt must leave the root
	// alive throughout this observation window, rather than only winning a race
	// with the child wait callback.
	select {
	case <-ctx.Done():
	case <-time.After(150 * time.Millisecond):
	}
	if ctx.Err() != nil {
		fmt.Fprintln(os.Stdout, "ROOT_CANCELLED_CHILD_REAPED")
		return
	}
	fmt.Fprintln(os.Stdout, "CHILD_INTERRUPTED_ROOT_ALIVE")
	cancel()
	<-ctx.Done()
	fmt.Fprintln(os.Stdout, "TERM_RECEIVED")
}
func startSignalFixture(t *testing.T, mode string) (*exec.Cmd, *bufio.Reader, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	c := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIChildFixture$", "--", "ccc-fixture:"+mode)
	c.Env = []string{}
	// Keep stdin open so ready-wait blocks on input, not a guessed delay.
	input, err := c.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = input.Close() })
	pipe, err := c.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Process.Kill() })
	reader := bufio.NewReader(pipe)
	line, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "READY") {
		t.Fatalf("fixture not ready: %q %v", line, err)
	}
	pid := 0
	if fields := strings.Fields(line); len(fields) > 1 {
		pid, _ = strconv.Atoi(fields[1])
		if child, e := os.FindProcess(pid); e == nil {
			t.Cleanup(func() { _ = child.Kill() })
		}
	}
	return c, reader, pid
}
func TestNativeTUISIGINTKeepsRootAndSIGTERMTerminates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native POSIX signal delivery; Windows console proof belongs to runner")
	}
	c, reader, pid := startSignalFixture(t, "ui-signals")
	if pid == 0 {
		t.Fatal("no child PID")
	}
	if err := c.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	child, _ := os.FindProcess(pid)
	if err := child.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	line, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "CHILD_INTERRUPTED_ROOT_ALIVE" {
		t.Fatalf("SIGINT cancelled TUI root: %q %v", line, err)
	}
	if err = c.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	line, err = reader.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "TERM_RECEIVED" {
		t.Fatalf("SIGTERM not delivered: %q %v", line, err)
	}
	if err = c.Wait(); err != nil {
		t.Fatal(err)
	}
}
func TestNativeTUISIGTERMReapsActiveChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native POSIX signal delivery")
	}
	c, reader, _ := startSignalFixture(t, "ui-signals")
	if err := c.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	line, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "ROOT_CANCELLED_CHILD_REAPED" {
		t.Fatalf("SIGTERM did not reap child: %q %v", line, err)
	}
	if err = c.Wait(); err != nil {
		t.Fatal(err)
	}
}
func TestNativeDirectCLISIGINTCancels(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native POSIX signal delivery")
	}
	c, reader, _ := startSignalFixture(t, "cli-signals")
	if err := c.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	line, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "CLI_CANCELLED" {
		t.Fatalf("direct launch ignored SIGINT: %q %v", line, err)
	}
	if err = c.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestRawTUICtrlCRetainsDirtyQuitGuard(t *testing.T) {
	_, backend, _, _, _ := harness(t)
	m := tui.New(backend, tui.Options{NoColor: true})
	m, _ = m.Update(m.Init()())
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd != nil || !strings.Contains(m.View().Content, "Leave with unapplied drafts?") {
		t.Fatal("raw Ctrl+C bypassed draft guard")
	}
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil || !strings.Contains(m.View().Content, "Draft") {
		t.Fatal("dismissal lost staged draft")
	}
}
func TestTUISignalSelectionMatchesActualEntryMode(t *testing.T) {
	d, _, _, _, _ := harness(t)
	d.terminal = true
	if !tuiRequested(nil, d) {
		t.Fatal("terminal TUI not selected")
	}
	for _, args := range [][]string{{"--plain"}, {"status"}, {"launch", "codex"}, {"--help"}, {"--version"}, {"--json"}, {"--bad"}} {
		if tuiRequested(args, d) {
			t.Fatalf("wrong TUI signal mode for %v", args)
		}
	}
	d.lookup = func(k string) (string, bool) { return "", k == "NO_COLOR" }
	if tuiRequested(nil, d) {
		t.Fatal("NO_COLOR selected TUI signal mode")
	}
}
