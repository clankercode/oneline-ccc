//go:build linux || darwin

package processowner

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// The syscall boundary is controlled per owner: no synthetic PID is ever
// passed to the kernel, and no foreign PID/PGID reuse is induced.
func posixSignalFixture(kill func(int, syscall.Signal) error) *Owner {
	cmd := &exec.Cmd{Process: &os.Process{Pid: 12345}}
	return &Owner{cmd: cmd, native: &nativeOwner{cmd: cmd, killGroup: kill}}
}

func TestPOSIXSuccessfulTerminationRetiresGroupSignal(t *testing.T) {
	calls := 0
	anchorRetired := false
	owner := posixSignalFixture(func(pid int, signal syscall.Signal) error {
		calls++
		if pid != -12345 || signal != syscall.SIGKILL {
			t.Error("unexpected group signal")
		}
		if anchorRetired {
			return syscall.EPERM // Darwin zombie-only result, or a recycled group.
		}
		return nil
	})
	if err := owner.Terminate(); err != nil {
		t.Fatal(err)
	}
	anchorRetired = true // Model the caller's sole Wait completing.
	if err := owner.Close(); err != nil {
		t.Errorf("Close repeated a successful terminal group signal: %v", err)
	}
	if err := owner.Terminate(); !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("repeat Terminate did not report completion: %v", err)
	}
	if calls != 1 {
		t.Errorf("retired group was signaled again: calls=%d", calls)
	}
}

func TestPOSIXFailedTerminationRemainsRetryable(t *testing.T) {
	for _, firstErr := range []error{syscall.EPERM, syscall.EINVAL, errors.New("synthetic signal refusal")} {
		for _, closeFirst := range []bool{false, true} {
			calls := 0
			owner := posixSignalFixture(func(int, syscall.Signal) error {
				calls++
				if calls == 1 {
					return firstErr
				}
				return nil
			})
			first := owner.Terminate
			if closeFirst {
				first = owner.Close
			}
			if err := first(); !errors.Is(err, firstErr) {
				t.Fatalf("first signal failure was lost: %v", err)
			}
			if err := owner.Terminate(); err != nil {
				t.Fatal("failed signal was not retryable", err)
			}
			if err := owner.Close(); err != nil || calls != 2 {
				t.Fatalf("retry/completion handling: calls=%d error=%v", calls, err)
			}
		}
	}
}

func TestPOSIXMissingGroupRetiresSignal(t *testing.T) {
	calls := 0
	owner := posixSignalFixture(func(int, syscall.Signal) error {
		calls++
		return syscall.ESRCH
	})
	if err := owner.Terminate(); !errors.Is(err, os.ErrProcessDone) {
		t.Fatal("missing group did not report completion", err)
	}
	if err := owner.Close(); err != nil || calls != 1 {
		t.Fatalf("missing group was signaled again: calls=%d error=%v", calls, err)
	}
}

func TestPOSIXConcurrentTerminateAndCloseShareSignal(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	owner := posixSignalFixture(func(int, syscall.Signal) error {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		return nil
	})
	var unblock sync.Once
	defer unblock.Do(func() { close(release) })
	const count = 12
	done := make(chan error, count+1)
	go func() { done <- owner.Terminate() }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first group signal did not start")
	}
	var callers sync.WaitGroup
	for i := range count {
		callers.Add(1)
		go func() {
			callers.Done()
			if i%2 == 0 {
				done <- owner.Terminate()
			} else {
				done <- owner.Close()
			}
		}()
	}
	callers.Wait()
	select {
	case err := <-done:
		t.Fatalf("cleanup returned before the in-flight signal completed: %v", err)
	default:
	}
	unblock.Do(func() { close(release) })
	for range count + 1 {
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, os.ErrProcessDone) {
				t.Error(err)
			}
		case <-time.After(time.Second):
			t.Fatal("concurrent cleanup did not complete")
		}
	}
	if calls.Load() != 1 {
		t.Errorf("concurrent callers repeated the terminal signal: calls=%d", calls.Load())
	}
}

func TestPOSIXCloseStillRestoresAfterTermination(t *testing.T) {
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	input.Close()
	defer output.Close()
	calls := 0
	owner := posixSignalFixture(func(int, syscall.Signal) error { calls++; return nil })
	owner.native.terminal = input
	if err := owner.Terminate(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := owner.Close(); !errors.Is(err, syscall.EBADF) {
			t.Fatal("foreground restoration was skipped or its failure lost", err)
		}
	}
	if calls != 1 {
		t.Fatalf("restoration retried group signaling: calls=%d", calls)
	}
}
