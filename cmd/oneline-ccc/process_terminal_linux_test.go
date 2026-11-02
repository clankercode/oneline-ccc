package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestDirectTerminalFixture(t *testing.T) {
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "ccc-terminal-child":
		fmt.Fprintln(os.Stdout, "CHILD_READY")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(line) != "input" {
			os.Exit(8)
		}
		fmt.Fprintln(os.Stdout, "CHILD_READ_INPUT")
		_, _ = io.Copy(io.Discard, reader)
		os.Exit(9)
	case "ccc-terminal-cli":
		ctx, stop := watchSignals(context.Background(), false)
		defer stop()
		child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDirectTerminalFixture$", "--", "ccc-terminal-child")
		child.Env = []string{}
		d := dependencies{in: os.Stdin, out: os.Stdout, errout: os.Stderr, runChild: runProcess}
		code := launch(ctx, flags{command: "launch"}, &fakeEngine{cmd: child}, d)
		foreground, err := unix.IoctlGetInt(int(os.Stdin.Fd()), unix.TIOCGPGRP)
		if err != nil || foreground != syscall.Getpgrp() || code != 130 {
			fmt.Fprintf(os.Stdout, "BAD_TERMINAL_RESULT %d %d %v\n", code, foreground, err)
			os.Exit(7)
		}
		fmt.Fprintln(os.Stdout, "CLI_130_FOREGROUND_RESTORED")
		os.Exit(0)
	}
}
func TestDirectCLIProcessGroupTerminalInputInterruptAndRestore(t *testing.T) {
	for _, interrupt := range []string{"canonical-ctrl-c", "sigterm"} {
		t.Run(interrupt, func(t *testing.T) {
			master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			if err = unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
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
			root := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDirectTerminalFixture$", "--", "ccc-terminal-cli")
			root.Env = []string{}
			root.Stdin, root.Stdout, root.Stderr = slave, slave, slave
			root.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
			if err = root.Start(); err != nil {
				t.Fatal(err)
			}
			defer root.Process.Kill()
			lines := make(chan string, 16)
			go func() {
				scanner := bufio.NewScanner(master)
				for scanner.Scan() {
					lines <- strings.TrimSpace(scanner.Text())
				}
				close(lines)
			}()
			await := func(want string) {
				t.Helper()
				for {
					select {
					case line, ok := <-lines:
						if !ok {
							t.Fatalf("PTY closed before %s", want)
						}
						if strings.HasSuffix(line, want) {
							return
						}
						if strings.Contains(line, "BAD_TERMINAL_RESULT") {
							t.Fatal(line)
						}
					case <-ctx.Done():
						t.Fatalf("PTY timeout before %s", want)
					}
				}
			}
			await("CHILD_READY")
			if _, err = master.WriteString("input\n"); err != nil {
				t.Fatal(err)
			}
			await("CHILD_READ_INPUT") // Proves no background-read SIGTTIN deadlock.
			if interrupt == "sigterm" {
				err = root.Process.Signal(syscall.SIGTERM)
			} else {
				_, err = master.Write([]byte{3})
			}
			if err != nil {
				t.Fatal(err)
			}
			await("CLI_130_FOREGROUND_RESTORED")
			if err = root.Wait(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
