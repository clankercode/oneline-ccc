//go:build linux || darwin

package processowner

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// CLI and TUI launches share an owned process group. Tea suspends terminal
// handling around its runner; interactive children receive foreground ownership
// here, and the caller's foreground group is restored before the runner returns.
type nativeOwner struct {
	cmd        *exec.Cmd
	terminal   *os.File
	foreground int
}

func prepareNative(cmd *exec.Cmd) (*nativeOwner, error) {
	attr := syscall.SysProcAttr{}
	if cmd.SysProcAttr != nil {
		attr = *cmd.SysProcAttr
	}
	// Never adopt or signal a group/session supplied by another owner.
	if attr.Pgid != 0 || attr.Setsid || attr.Setctty || attr.Foreground {
		return nil, ErrPolicy
	}
	tree := &nativeOwner{cmd: cmd}
	attr.Setpgid = true
	if terminal, ok := cmd.Stdin.(*os.File); ok {
		info, err := terminal.Stat()
		if err != nil {
			return nil, err
		}
		// Darwin pipes also implement TIOCGPGRP, returning their async I/O
		// group (initially zero). Only character devices can be terminals.
		if info.Mode()&os.ModeCharDevice != 0 {
			foreground, err := unix.IoctlGetInt(int(terminal.Fd()), unix.TIOCGPGRP)
			if err == nil {
				if foreground != syscall.Getpgrp() {
					return nil, ErrPolicy
				}
				tree.terminal, tree.foreground = terminal, foreground
				attr.Foreground, attr.Ctty = true, int(terminal.Fd())
			} else if !errors.Is(err, syscall.ENOTTY) {
				return nil, err
			}
		}
	}
	cmd.SysProcAttr = &attr
	return tree, nil
}
func (p *nativeOwner) started() error { return nil }
func (p *nativeOwner) terminate() error {
	err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
func (p *nativeOwner) close() error {
	var cleanupErr error
	if p.cmd.Process != nil {
		cleanupErr = p.terminate()
		if errors.Is(cleanupErr, os.ErrProcessDone) {
			cleanupErr = nil
		}
	}
	if p.terminal == nil {
		return cleanupErr
	}
	// tcsetpgrp from the temporarily background CLI would otherwise stop it.
	// CLI signal ownership is process-wide already; restore SIGTTOU's previous
	// ignored/default policy immediately after restoring terminal foreground.
	ignored := signal.Ignored(syscall.SIGTTOU)
	signal.Ignore(syscall.SIGTTOU)
	err := unix.IoctlSetPointerInt(int(p.terminal.Fd()), unix.TIOCSPGRP, p.foreground)
	if !ignored {
		signal.Reset(syscall.SIGTTOU)
	}
	return errors.Join(cleanupErr, err)
}
