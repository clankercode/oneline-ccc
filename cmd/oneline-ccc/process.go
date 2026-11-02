package main

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"syscall"
	"time"
)

func launch(ctx context.Context, o flags, e engine, d dependencies) int {
	action, id := "ssh", ""
	if o.command == "launch" {
		action = "launch"
		id = "codex"
	}
	cmd, err := e.Command(ctx, action, id)
	label := "Native client could not launch"
	if action == "ssh" {
		label = "SSH setup unavailable; requires Windows, an Administrator terminal and protected script staging"
	}
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return fail(d, label, err)
	}
	if cmd == nil || cmd.Env == nil {
		return fail(d, "Child launch refused: backend must provide an explicit environment", errArguments)
	}
	err = d.runChild(ctx, cmd, d.in, d.out, d.errout)
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return 128 + int(status.Signal())
		}
		if exit.ExitCode() > 0 {
			return exit.ExitCode()
		}
	}
	return fail(d, label, err)
}

// runProcess retains the backend's command and context configuration. The
// explicit cancellation path also covers injected commands not built with
// exec.CommandContext. An empty non-nil Env deliberately inherits nothing.
func runProcess(ctx context.Context, cmd *exec.Cmd, in io.Reader, out, errout io.Writer) error {
	if cmd == nil || cmd.Env == nil {
		return errArguments
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errout
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Start(); err != nil {
		return err
	}
	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	select {
	case err := <-finished:
		return err
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-finished
		return ctx.Err()
	}
}
