//go:build linux

// This acceptance probe runs a real terminal and renderer in a subprocess.
// Keep the renderer and raw child enabled, including in the race gate, so
// cancelled terminal reinitialization exercises the real input-reader lifetime.
package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/clankercode/oneline-ccc/internal/tui"
	"golang.org/x/sys/unix"
)

type terminalForegroundBackend struct{ *foregroundMutation }

func (b terminalForegroundBackend) Command(ctx context.Context, _, _ string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestForegroundTerminalFixture$")
	cmd.Env = []string{"ONELINE_FOREGROUND_TERMINAL_FIXTURE=child"}
	return cmd, nil
}

func TestForegroundTerminalFixture(t *testing.T) {
	switch os.Getenv("ONELINE_FOREGROUND_TERMINAL_FIXTURE") {
	case "child":
		// A forcibly terminated interactive client cannot run its own deferred
		// terminal cleanup. Leave raw mode behind just as the native client does.
		if _, err := term.MakeRaw(os.Stdin.Fd()); err != nil {
			os.Exit(8)
		}
		fmt.Fprintf(os.Stdout, "FOREGROUND_CHILD_RAW_READY %d\n", os.Getpid())
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(7)
	case "ui":
		ctx, stop := watchSignals(context.Background(), true)
		defer stop()
		backend := terminalForegroundBackend{&foregroundMutation{}}
		err := runForeground(ctx, backend, tui.Options{NoColor: true}, func(m tea.Model) error {
			m, _ = m.Update(m.Init()())
			m, _ = m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
			m, _ = m.Update(tea.PasteMsg{Content: "Launch selected client"})
			m, launch := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if launch == nil {
				return errors.New("fixture could not request launch")
			}
			// Keep the renderer enabled: WithoutRenderer would skip Tea's actual
			// terminal initialization/restoration and mask this regression.
			return runForegroundProgram(ctx, preparedForeground{m, launch})
		})
		if !errors.Is(err, context.Canceled) {
			os.Exit(9)
		}
		os.Exit(0)
	}
}

func TestForegroundProgramLifetime(t *testing.T) {
	for _, kind := range []string{"quit", "canceled-before-start", "canceled-while-running", "startup-error"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var first tea.Cmd
			switch kind {
			case "quit":
				first = tea.Quit
			case "canceled-before-start", "startup-error":
				cancel()
			case "canceled-while-running":
				first = func() tea.Msg { cancel(); return nil }
			}
			var model tea.Model = preparedForeground{tui.New(&foregroundMutation{}, tui.Options{NoColor: true}), first}
			if kind == "startup-error" {
				model = nil
			}
			done := make(chan error, 1)
			go func() {
				done <- runForegroundProgram(ctx, model, tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard), tea.WithoutRenderer())
			}()
			select {
			case err := <-done:
				if kind == "quit" {
					if err != nil {
						t.Fatal(err)
					}
				} else if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation not preserved: %v", err)
				}
				if kind == "startup-error" && (err == nil || !strings.Contains(err.Error(), "InitialModel cannot be nil")) {
					t.Fatalf("startup error not preserved: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("foreground program did not join its shutdown watcher")
			}
		})
	}
}

func TestForegroundSIGTERMRestoresTerminalAfterRawChild(t *testing.T) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	if err := unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 40, Col: 120}); err != nil {
		t.Fatal(err)
	}
	before, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	app := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestForegroundTerminalFixture$")
	app.Env = []string{"ONELINE_FOREGROUND_TERMINAL_FIXTURE=ui", "TERM=xterm-256color"}
	var diagnostics bytes.Buffer
	app.Stdin, app.Stdout, app.Stderr = slave, slave, &diagnostics
	app.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		if !waited {
			_ = app.Process.Kill()
			_ = app.Wait()
		}
	})
	ready := make(chan int, 1)
	go func() {
		reader := bufio.NewReader(master)
		for {
			line, err := reader.ReadString('\n')
			if index := strings.Index(line, "FOREGROUND_CHILD_RAW_READY "); index >= 0 {
				pid, _ := strconv.Atoi(strings.TrimSpace(line[index+len("FOREGROUND_CHILD_RAW_READY "):]))
				ready <- pid
				_, _ = io.Copy(io.Discard, reader)
				return
			}
			if err != nil {
				ready <- 0
				return
			}
		}
	}()
	var child int
	select {
	case child = <-ready:
		if child <= 0 {
			t.Fatal("raw interactive child did not report its PID")
		}
	case <-ctx.Done():
		t.Fatal("raw interactive child never started")
	}
	t.Cleanup(func() {
		if child > 0 {
			_ = unix.Kill(child, unix.SIGKILL)
		}
	})
	during, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil || during.Lflag&(unix.ICANON|unix.ECHO) != 0 {
		t.Fatal("child did not leave terminal in raw mode", err)
	}
	if err := app.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	err = app.Wait()
	waited = true
	if err != nil {
		t.Fatalf("foreground helper failed to finish cancellation: %v\n%s", err, diagnostics.String())
	}
	if err := unix.Kill(child, 0); !errors.Is(err, unix.ESRCH) {
		t.Fatal("foreground helper returned before reaping its child", err)
	}
	child = 0 // Do not signal an already reaped PID during cleanup.
	after, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("terminal modes not restored after SIGTERM: iflag %x -> %x, oflag %x -> %x, lflag %x -> %x", before.Iflag, after.Iflag, before.Oflag, after.Oflag, before.Lflag, after.Lflag)
	}
}
