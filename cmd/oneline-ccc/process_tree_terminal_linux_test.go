package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

func TestTerminalStateFixture(t *testing.T) {
	mode := os.Args[len(os.Args)-1]
	if mode == "raw-state-child" {
		if _, err := term.MakeRaw(os.Stdin.Fd()); err != nil {
			os.Exit(51)
		}
		fmt.Fprintln(os.Stdout, "RAW_STATE_READY")
		var input [1]byte
		_, _ = os.Stdin.Read(input[:])
		os.Exit(0) // Deliberately leaves the terminal raw, like a killed client.
	}
	if mode != "raw-state-runner" {
		return
	}
	before, err := term.GetState(os.Stdin.Fd())
	if err != nil {
		os.Exit(52)
	}
	ctx, stop := watchSignals(context.Background(), false)
	defer stop()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTerminalStateFixture$", "--", "raw-state-child")
	cmd.Env = []string{}
	err = runProcess(ctx, cmd, os.Stdin, os.Stdout, os.Stderr)
	after, stateErr := term.GetState(os.Stdin.Fd())
	foreground, foregroundErr := unix.IoctlGetInt(int(os.Stdin.Fd()), unix.TIOCGPGRP)
	if stateErr != nil || foregroundErr != nil || foreground != syscall.Getpgrp() || !reflect.DeepEqual(before, after) {
		fmt.Fprintln(os.Stdout, "TERMINAL_STATE_NOT_RESTORED")
		os.Exit(53)
	}
	if err != nil && ctx.Err() == nil {
		os.Exit(54)
	}
	fmt.Fprintln(os.Stdout, "TERMINAL_STATE_RESTORED")
	os.Exit(0)
}

func TestOwnedRunnerRestoresPreChildTerminalState(t *testing.T) {
	for _, termination := range []string{"normal", "sigterm"} {
		t.Run(termination, func(t *testing.T) {
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
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			root := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTerminalStateFixture$", "--", "raw-state-runner")
			root.Env = []string{}
			root.Stdin, root.Stdout, root.Stderr = slave, slave, slave
			root.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
			if err := root.Start(); err != nil {
				t.Fatal(err)
			}
			defer root.Process.Kill()
			lines := make(chan string, 16)
			go func() {
				defer close(lines)
				scan := bufio.NewScanner(master)
				for scan.Scan() {
					lines <- strings.TrimSpace(scan.Text())
				}
			}()
			await := func(want string) {
				t.Helper()
				for {
					select {
					case line, ok := <-lines:
						if !ok || line == "TERMINAL_STATE_NOT_RESTORED" {
							t.Fatalf("terminal restoration failed before %s: %s", want, line)
						}
						if line == want {
							return
						}
					case <-ctx.Done():
						t.Fatal("terminal restoration timeout")
					}
				}
			}
			await("RAW_STATE_READY")
			if termination == "sigterm" {
				err = root.Process.Signal(syscall.SIGTERM)
			} else {
				_, err = io.WriteString(master, "x")
			}
			if err != nil {
				t.Fatal(err)
			}
			await("TERMINAL_STATE_RESTORED")
			if err := root.Wait(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
