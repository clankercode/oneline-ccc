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
	case "streams":
		data, _ := io.ReadAll(os.Stdin)
		fmt.Printf("stdout:%s|ambient:%s", data, os.Getenv("CCC_AMBIENT"))
		fmt.Fprint(os.Stderr, "child-stderr")
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
