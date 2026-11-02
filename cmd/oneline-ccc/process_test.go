package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCLIChildFixture(t *testing.T) {
	mode := os.Args[len(os.Args)-1]
	if !strings.HasPrefix(mode, "ccc-fixture:") {
		return
	}
	switch strings.TrimPrefix(mode, "ccc-fixture:") {
	case "ready-wait":
		fmt.Fprintf(os.Stdout, "READY %d\n", os.Getpid())
		_, _ = io.Copy(io.Discard, os.Stdin)
	case "streams":
		data, _ := io.ReadAll(os.Stdin)
		fmt.Printf("stdout:%s|ambient:%s", data, os.Getenv("CCC_AMBIENT"))
		fmt.Fprint(os.Stderr, "child-stderr")
	case "interrupt":
		self, _ := os.FindProcess(os.Getpid())
		_ = self.Signal(os.Interrupt)
		time.Sleep(time.Second)
		os.Exit(99)
	case "exit7":
		os.Exit(7)
	case "ui-signals":
		signalFixture(true)
	case "cli-signals":
		signalFixture(false)
	case "wait":
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}
func childFixture(mode string) *exec.Cmd {
	c := exec.Command(os.Args[0], "-test.run=^TestCLIChildFixture$", "--", "ccc-fixture:"+mode)
	c.Env = []string{}
	return c
}
func TestRunProcessForwardsStreamsAndIsolatesEmptyEnvironment(t *testing.T) {
	t.Setenv("CCC_AMBIENT", "injected-must-not-inherit")
	out, errout := &bytes.Buffer{}, &bytes.Buffer{}
	if err := runProcess(context.Background(), childFixture("streams"), strings.NewReader("input"), out, errout); err != nil {
		t.Fatal(err)
	}
	if out.String() != "stdout:input|ambient:" || errout.String() != "child-stderr" {
		t.Fatal("streams or empty environment changed")
	}
}
func TestDirectLaunchReturnsChildExitStatus(t *testing.T) {
	d, f, _, _, _ := harness(t)
	f.cmd = childFixture("exit7")
	d.runChild = runProcess
	if got := run(context.Background(), []string{"launch", "codex"}, d); got != 7 {
		t.Fatalf("child exit %d, wanted 7", got)
	}
}
func TestRunProcessCancelsCommandsWithoutCommandContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := runProcess(ctx, childFixture("wait"), strings.NewReader(""), io.Discard, io.Discard)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 3*time.Second {
		t.Fatal("child cancellation was not bounded")
	}
}
func TestRunProcessPrecancelledAndNilEnvironmentRefusal(t *testing.T) {
	c := childFixture("exit7")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(runProcess(ctx, c, nil, io.Discard, io.Discard), context.Canceled) || c.Process != nil {
		t.Fatal("precancelled command started")
	}
	c = childFixture("exit7")
	c.Env = nil
	if runProcess(context.Background(), c, nil, io.Discard, io.Discard) == nil || c.Process != nil {
		t.Fatal("nil environment started")
	}
}
func TestMonitorExactCommandContract(t *testing.T) {
	d, f, _, _, _ := harness(t)
	root := t.TempDir()
	state, home := filepath.Join(root, "state"), filepath.Join(root, "codex")
	var opts engineOptions
	d.newEngine = func(o engineOptions) (engine, error) { opts = o; return f, nil }
	if run(context.Background(), []string{"--state-dir", state, "--codex-home", home, "observe"}, d) != 0 || f.observe != 1 || f.once || opts.StateRoot != state || opts.CodexHome != home {
		t.Fatal("monitor launch vector changed")
	}
}

func TestDirectChildInterruptPreservesSignalExitStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX signal exit status; Windows console proof belongs to runner")
	}
	d, f, _, _, _ := harness(t)
	f.cmd = childFixture("interrupt")
	d.runChild = runProcess
	if got := run(context.Background(), []string{"launch", "codex"}, d); got != 130 {
		t.Fatalf("interrupted child returned %d, wanted 130", got)
	}
}

func TestCancelledLaunchNormalizesRacingNativeWaitError(t *testing.T) {
	// Cmd.Wait is documented to prefer a native exit error over cancellation.
	// Reproduce that completion ordering at the runner boundary without relying
	// on which arm of runProcess's select happens to win.
	nativeErr := childFixture("exit7").Run()
	var exit *exec.ExitError
	if !errors.As(nativeErr, &exit) {
		t.Fatalf("expected native exit error, got %T", nativeErr)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d, f, _, _, _ := harness(t)
	f.cmd = childFixture("wait")
	d.runChild = func(context.Context, *exec.Cmd, io.Reader, io.Writer, io.Writer) error {
		cancel() // Explicit barrier: cancellation precedes result projection.
		return nativeErr
	}
	if got := run(ctx, []string{"launch", "codex"}, d); got != 130 {
		t.Fatalf("cancelled native wait projected exit %d, want 130", got)
	}
}

func TestRunProcessStartFailureDoesNotWaitForAssignment(t *testing.T) {
	cmd := exec.Command(filepath.Join(t.TempDir(), "nonexistent-native-client"))
	cmd.Env = []string{}
	if err := runProcess(context.Background(), cmd, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("missing executable succeeded")
	}
	if cmd.Process != nil {
		t.Fatal("missing executable started")
	}
}
