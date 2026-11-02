//go:build linux || darwin

package processowner

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Only this disposable test binary is executed. The root deliberately does not
// reap its helper, matching codexrpc's cleanup fixture rather than a real Codex.
func TestPOSIXCleanupFixture(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "owned-cleanup-fixture" {
		return
	}
	if os.Args[len(os.Args)-1] == "exit" {
		os.Exit(0)
	}
	if os.Args[len(os.Args)-1] == "helper" {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			os.Exit(31)
		}
		_ = listener.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second))
		fmt.Fprintln(os.Stdout, listener.Addr())
		for {
			conn, err := listener.Accept()
			if err != nil {
				os.Exit(0)
			}
			_ = conn.SetDeadline(time.Now().Add(time.Second))
			line, _ := bufio.NewReader(conn).ReadString('\n')
			if line == "stop\n" {
				os.Exit(0)
			}
			fmt.Fprintln(conn, "running")
			conn.Close()
		}
	}
	child := exec.Command(os.Args[0], "-test.run=^TestPOSIXCleanupFixture$", "--", "owned-cleanup-fixture", "helper")
	child.Env = []string{"GOMAXPROCS=2", "GORACE=atexit_sleep_ms=0"}
	out, err := child.StdoutPipe()
	if err != nil || child.Start() != nil {
		os.Exit(32)
	}
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		os.Exit(33)
	}
	out.Close()
	fmt.Print(line)
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

func TestPOSIXCleanupAcrossRootWait(t *testing.T) {
	for _, tc := range []struct{ name, raceOptions string }{
		{"default-race-exit", ""},
		{"no-race-exit-delay", "atexit_sleep_ms=0"},
		{"forced-race-exit-overlap", "atexit_sleep_ms=1200"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stage := filepath.Join(t.TempDir(), "stage")
			if err := os.Mkdir(stage, 0700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestPOSIXCleanupFixture$", "--", "owned-cleanup-fixture", "root")
			cmd.Dir = stage
			cmd.Env = []string{"GOMAXPROCS=2", "GORACE=" + tc.raceOptions}
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
			var rootWait chan error
			defer func() {
				// Early readiness failures retain the unreaped root as an
				// ownership anchor until cleanup. After the measured Close,
				// retire all deferred group signals, including fatal exits.
				if !cleanupComplete {
					_ = owner.Close()
				}
				if started && !waited {
					if rootWait == nil {
						_ = cmd.Wait()
					} else {
						<-rootWait
					}
				}
			}()
			if err := owner.Start(); err != nil {
				t.Fatal(err)
			}
			started = true
			ready := make(chan string, 1)
			go func() {
				line, _ := bufio.NewReader(output).ReadString('\n')
				ready <- line
			}()
			var address string
			select {
			case line := <-ready:
				if len(line) == 0 {
					t.Fatal("owned helper did not report readiness")
				}
				address = line[:len(line)-1]
			case <-time.After(3 * time.Second):
				t.Fatal("owned helper readiness timed out")
			}
			defer func() {
				if conn, err := net.DialTimeout("tcp", address, time.Second); err == nil {
					fmt.Fprintln(conn, "stop")
					conn.Close()
				}
			}()
			_ = input.Close()
			rootWait = make(chan error, 1)
			go func() { rootWait <- cmd.Wait() }()
			forced := false
			var waitErr error
			select {
			case waitErr = <-rootWait:
			case <-time.After(time.Second):
				forced = true
				err := owner.Terminate()
				var errno syscall.Errno
				_ = errors.As(err, &errno)
				t.Logf("terminate native_errno=%d process_done=%t", errno, errors.Is(err, os.ErrProcessDone))
				if err != nil && !errors.Is(err, os.ErrProcessDone) {
					t.Errorf("owned group termination failed: %v", err)
					_ = cmd.Process.Kill()
				}
				waitErr = <-rootWait
			}
			waited = true
			err = owner.Close()
			cleanupComplete = true
			var errno syscall.Errno
			_ = errors.As(err, &errno)
			t.Logf("forced=%t root_wait_error=%t close_native_errno=%d", forced, waitErr != nil, errno)
			if err != nil {
				t.Errorf("owned group cleanup failed: %v", err)
			}
			if conn, dialErr := net.DialTimeout("tcp", address, time.Second); dialErr == nil {
				_ = conn.SetDeadline(time.Now().Add(time.Second))
				fmt.Fprintln(conn, "probe")
				line, _ := bufio.NewReader(conn).ReadString('\n')
				conn.Close()
				if line == "running\n" {
					t.Fatal("owned helper executing after Close")
				}
			}
			if err := os.RemoveAll(stage); err != nil {
				t.Fatal("owned helper retained the stage", err)
			}
		})
	}
}
