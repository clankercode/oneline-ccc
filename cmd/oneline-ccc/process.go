package main

import (
	"context"
	"errors"
	"io"
	"os"
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
	// A CommandContext wait may report the native kill/exit error instead of
	// context cancellation. Resolve that race before projecting an exit status.
	err = operationError(ctx, d.runChild(ctx, cmd, d.in, d.out, d.errout))
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
func runProcess(ctx context.Context, cmd *exec.Cmd, in io.Reader, out, errout io.Writer) (result error) {
	if cmd == nil || cmd.Env == nil || cmd.Process != nil {
		return errArguments
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errout
	tree, err := prepareProcessTree(cmd)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, tree.close()) }()
	if cmd.Cancel != nil {
		cmd.Cancel = tree.terminate
	}
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := tree.started(); err != nil {
		_ = tree.terminate()
		_ = cmd.Process.Kill() // Also covers a failed suspended Windows assignment.
		_ = cmd.Wait()
		return err
	}
	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	select {
	case err := <-finished:
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	case <-ctx.Done():
		cleanupErr := tree.terminate()
		<-finished
		if errors.Is(cleanupErr, os.ErrProcessDone) {
			cleanupErr = nil
		}
		return errors.Join(ctx.Err(), cleanupErr)
	}
}
