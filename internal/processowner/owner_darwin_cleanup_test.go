//go:build darwin

package processowner

import (
	"errors"
	"os"
	"os/exec"
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
	deadline := time.Now().Add(3 * time.Second)
	for {
		info, err := unix.SysctlKinfoProc("kern.proc.pid", cmd.Process.Pid)
		if err != nil {
			t.Fatal("cannot inspect owned fixture", err)
		}
		const szomb = 5 // Darwin sys/proc.h SZOMB; no runnable group member remains.
		if info.Proc.P_stat == szomb {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("owned fixture did not become a zombie")
		}
		time.Sleep(time.Millisecond)
	}
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
