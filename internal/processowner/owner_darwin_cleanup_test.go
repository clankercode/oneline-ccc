//go:build darwin

package processowner

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestDarwinZombieOnlyGroupDiagnostic(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestPOSIXCleanupFixture$", "--", "owned-cleanup-fixture", "exit")
	cmd.Env = []string{"GOMAXPROCS=2", "GORACE=atexit_sleep_ms=0"}
	cmd.Dir = t.TempDir()
	owner, err := Prepare(cmd)
	if err != nil {
		t.Fatal(err)
	}
	started, cleanupComplete := false, false
	defer func() {
		// Fatal probe exits still clean up while the unreaped root pins its
		// identity. Never signal this numeric group again after the sole Wait.
		if !cleanupComplete {
			_ = owner.Close()
		}
		if started {
			_ = cmd.Wait()
		}
	}()
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	started = true
	// Keep the fixture unreaped until both operations have been measured. Wait
	// still belongs to this caller, and the original PID cannot be reused here.
	awaitDarwinFixtureZombie(t, cmd)
	for _, operation := range []struct {
		name string
		run  func() error
	}{{"terminate-zombie", owner.Terminate}, {"close-zombie", owner.Close}} {
		err := operation.run()
		var errno syscall.Errno
		_ = errors.As(err, &errno)
		t.Logf("operation=%s native_errno=%d process_done=%t", operation.name, errno, errors.Is(err, os.ErrProcessDone))
		// Characterize the kernel's zombie-only group behavior; this must not
		// normalize EPERM for a group that might still have live members.
		if err != nil && !errors.Is(err, syscall.EPERM) && !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("unexpected zombie-only group result: %v", err)
		}
	}
	cleanupComplete = true
}

func awaitDarwinFixtureZombie(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		info, err := unix.SysctlKinfoProc("kern.proc.pid", cmd.Process.Pid)
		if err != nil {
			t.Fatal("cannot inspect owned fixture", err)
		}
		const szomb = 5 // Darwin sys/proc.h SZOMB; this observes only the root.
		if info.Proc.P_stat == szomb {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("owned fixture did not become a zombie")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestDarwinLeaderExitWithLiveHelperDiagnostic(t *testing.T) {
	stage := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestPOSIXCleanupFixture$", "--", "owned-cleanup-fixture", "root")
	cmd.Env = []string{"GOMAXPROCS=1", "GORACE=atexit_sleep_ms=0"}
	cmd.Dir = stage
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	owner, err := Prepare(cmd)
	if err != nil {
		t.Fatal(err)
	}
	started, waited, cleanupComplete := false, false, false
	defer func() {
		if !cleanupComplete {
			_ = owner.Close() // Only before the unreaped root anchor is retired.
		}
		if started && !waited {
			_ = cmd.Wait()
		}
	}()
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	started = true
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(output).ReadString('\n'); ready <- strings.TrimSpace(line) }()
	var address string
	select {
	case address = <-ready:
		if address == "" {
			t.Fatal("owned helper did not report readiness")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("owned helper readiness timed out")
	}
	defer func() {
		if conn, err := net.DialTimeout("tcp", address, time.Second); err == nil {
			fmt.Fprintln(conn, "stop")
			conn.Close()
		}
	}()
	helperRunning := func() bool {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			return false
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(time.Second))
		fmt.Fprintln(conn, "probe")
		line, _ := bufio.NewReader(conn).ReadString('\n')
		return line == "running\n"
	}
	_ = input.Close()
	awaitDarwinFixtureZombie(t, cmd) // No Wait or concurrent reaper exists yet.
	if !helperRunning() {
		t.Fatal("zombie leader did not retain a live helper for the control")
	}
	kill, signalCalls := owner.native.killGroup, 0
	owner.native.killGroup = func(group int, signal syscall.Signal) error {
		signalCalls++
		if signalCalls > 1 {
			return errors.New("diagnostic intercepted repeated group signal")
		}
		return kill(group, signal)
	}
	err = owner.Terminate()
	var errno syscall.Errno
	_ = errors.As(err, &errno)
	t.Logf("leader_zombie=true helper_live_before=true first_signal_errno=%d", errno)
	if err != nil {
		t.Fatal("first signal to owned live-helper group failed", err)
	}
	cleanupComplete = true
	waitErr := cmd.Wait()
	waited = true
	if err := owner.Close(); err != nil || signalCalls != 1 {
		t.Fatalf("cleanup after anchored signal: signal_calls=%d error=%v", signalCalls, err)
	}
	if helperRunning() {
		t.Fatal("owned helper still executes after cleanup")
	}
	if err := os.RemoveAll(stage); err != nil {
		t.Fatal("owned helper retained stage", err)
	}
	t.Logf("signal_calls=%d root_wait_error=%t helper_live_after=false", signalCalls, waitErr != nil)
}
