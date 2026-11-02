package tui

import (
	"context"
	"io"
	"os/exec"
)

// tea.Exec suspends/restores the terminal around this adapter. Production injects
// the CLI runner, which owns process groups/jobs and terminal foreground. Keep
// those platform algorithms at that single boundary.
type terminalCommand struct {
	ctx     context.Context
	command *exec.Cmd
	runner  func(context.Context, *exec.Cmd, io.Reader, io.Writer, io.Writer) error
}

func (c *terminalCommand) Run() error {
	if err := c.ctx.Err(); err != nil {
		return err
	}
	if c.runner != nil {
		return c.runner(c.ctx, c.command, c.command.Stdin, c.command.Stdout, c.command.Stderr)
	}
	return c.command.Run()
}
func (c *terminalCommand) SetStdin(in io.Reader) {
	if c.command.Stdin == nil {
		c.command.Stdin = in
	}
}
func (c *terminalCommand) SetStdout(out io.Writer) {
	if c.command.Stdout == nil {
		c.command.Stdout = out
	}
}
func (c *terminalCommand) SetStderr(out io.Writer) {
	if c.command.Stderr == nil {
		c.command.Stderr = out
	}
}
