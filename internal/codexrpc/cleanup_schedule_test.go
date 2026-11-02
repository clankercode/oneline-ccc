package codexrpc

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// This fixture never spawns descendants. Direct-child signals use os.Process;
// all numeric group operations in these schedules are intercepted.
func reapingFixture(t *testing.T) (*exec.Cmd, func() error, *atomic.Int32) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRPCProcess$", "--", "rpc-fixture")
	cmd.Env = []string{"GOMAXPROCS=1", "GORACE=atexit_sleep_ms=0"}
	cmd.Dir = t.TempDir()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	var calls atomic.Int32
	var waitErr error
	wait := func() error {
		once.Do(func() { calls.Add(1); waitErr = cmd.Wait() })
		return waitErr
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = wait() })
	return cmd, wait, &calls
}

func awaitCleanupBarrier(t *testing.T, ready <-chan struct{}) {
	t.Helper()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("controlled cleanup schedule did not reach its barrier")
	}
}

func TestRPCCleanupFallbackCanPrecedeWaitResultAfterReap(t *testing.T) {
	cmd, wait, waitCalls := reapingFixture(t)
	reaped, releaseWait, clockReady, terminated := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(releaseWait) }) })
	tick := make(chan time.Time, 1)
	terminateCalls, closeCalls, rootKillCalls := 0, 0, 0
	cleanup := rpcCleanup{
		wait: func() error {
			err := wait() // Real Cmd.Wait has reaped the real fixture here.
			close(reaped)
			<-releaseWait // Withhold delivery to the production cleanup select.
			return err
		},
		after: func(delay time.Duration) <-chan time.Time {
			if delay != time.Second {
				t.Error("production cleanup deadline changed")
			}
			close(clockReady)
			return tick
		},
		terminate: func() error {
			terminateCalls++
			select {
			case <-reaped:
			default:
				t.Error("fallback ran before the controlled root reap")
			}
			close(terminated)
			return syscall.EPERM // Synthetic refusal; never signal a retired PGID.
		},
		killRoot: func() error {
			rootKillCalls++
			err := cmd.Process.Kill()
			if !errors.Is(err, os.ErrProcessDone) {
				t.Error("reaped root was not retired by os.Process", err)
			}
			return err
		},
		close: func() error { closeCalls++; return nil },
	}
	done := make(chan error, 1)
	go func() { done <- cleanup.run() }()
	awaitCleanupBarrier(t, reaped)
	awaitCleanupBarrier(t, clockReady)
	tick <- time.Now()
	awaitCleanupBarrier(t, terminated)
	select {
	case <-done:
		t.Fatal("cleanup returned before its sole Wait result")
	default:
	}
	release.Do(func() { close(releaseWait) })
	select {
	case err := <-done:
		var errno syscall.Errno
		if !errors.As(err, &errno) || errno != syscall.EPERM || !strings.Contains(err.Error(), "cannot terminate owned Codex configuration processes") {
			t.Fatal("first fallback refusal was hidden or reclassified", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup did not finish after Wait delivery")
	}
	if waitCalls.Load() != 1 || terminateCalls != 1 || closeCalls != 1 || rootKillCalls != 1 {
		t.Fatalf("cleanup ownership changed: waits=%d terminate=%d close=%d root_kill=%d", waitCalls.Load(), terminateCalls, closeCalls, rootKillCalls)
	}
	t.Log("real root reaped -> synthetic timeout -> intercepted first group refusal -> sole Wait delivery -> Close")
}

func TestRPCCleanupWaitCompletionStillClosesDescendants(t *testing.T) {
	_, wait, waitCalls := reapingFixture(t)
	rootReaped, helperRunning := false, true
	closeCalls := 0
	cleanup := rpcCleanup{
		wait:  func() error { err := wait(); rootReaped = true; return err },
		after: func(time.Duration) <-chan time.Time { return make(chan time.Time) },
		terminate: func() error {
			t.Error("graceful Wait unexpectedly used forced termination")
			return nil
		},
		killRoot: func() error { t.Error("graceful Wait unexpectedly killed the root"); return nil },
		close: func() error {
			closeCalls++
			if !rootReaped || !helperRunning {
				t.Error("descendant cleanup did not follow root completion")
			}
			helperRunning = false // Controlled live-descendant obligation.
			return nil
		},
	}
	if err := cleanup.run(); err != nil || helperRunning || closeCalls != 1 || waitCalls.Load() != 1 {
		t.Fatalf("leader exit skipped descendant cleanup: waits=%d close=%d helper_running=%t error=%v", waitCalls.Load(), closeCalls, helperRunning, err)
	}
}

func TestRPCCleanupFailuresKeepSafeCauses(t *testing.T) {
	releaseWait := make(chan struct{})
	tick := make(chan time.Time, 1)
	tick <- time.Now()
	cleanup := rpcCleanup{
		wait:  func() error { <-releaseWait; return nil },
		after: func(time.Duration) <-chan time.Time { return tick },
		terminate: func() error {
			return &os.PathError{Op: "private-fixture-operation", Path: "private-fixture-path", Err: syscall.EPERM}
		},
		killRoot: func() error { close(releaseWait); return os.ErrProcessDone },
		close: func() error {
			return &os.PathError{Op: "private-fixture-operation", Path: "private-fixture-path", Err: syscall.EINVAL}
		},
	}
	err := cleanup.run()
	if err == nil || !errors.Is(err, syscall.EPERM) || !errors.Is(err, syscall.EINVAL) || strings.Contains(err.Error(), "private-fixture") {
		t.Fatal("cleanup failures were hidden or private causes exposed", err)
	}
}
