// Package processowner owns native subprocess groups/Jobs and restores real
// terminal state before returning control to a suspended terminal application.
package processowner

import (
	"errors"
	"os"
	"os/exec"

	"github.com/charmbracelet/x/term"
)

var ErrPolicy = errors.New("native process ownership could not be established")

type terminalState struct {
	file  *os.File
	state *term.State
}

// Owner must be closed after Prepare, including when Start fails. Wait remains
// the caller's responsibility; Close terminates remaining owned descendants.
type Owner struct {
	cmd       *exec.Cmd
	native    *nativeOwner
	terminals []terminalState
}

func Prepare(cmd *exec.Cmd) (*Owner, error) {
	if cmd == nil || cmd.Env == nil || cmd.Process != nil {
		return nil, ErrPolicy
	}
	native, err := prepareNative(cmd)
	if err != nil {
		return nil, err
	}
	o := &Owner{cmd: cmd, native: native}
	// Windows consoles have separate input/output modes. POSIX descriptors can
	// alias one TTY; all snapshots precede child execution and agree in that case.
	for _, stream := range []any{cmd.Stdin, cmd.Stdout, cmd.Stderr} {
		file, ok := stream.(*os.File)
		if !ok || !term.IsTerminal(file.Fd()) {
			continue
		}
		state, err := term.GetState(file.Fd())
		if err != nil {
			return nil, errors.Join(err, native.close())
		}
		o.terminals = append(o.terminals, terminalState{file: file, state: state})
	}
	return o, nil
}

// Start establishes descendant ownership before user code runs on Windows.
// Cancellation waits for that assignment barrier rather than killing only the
// suspended direct child. An unsuccessful Start never leaves a child to reap.
func (o *Owner) Start() error {
	if o.cmd.Cancel != nil {
		o.cmd.Cancel = o.Terminate
	}
	if err := o.cmd.Start(); err != nil {
		return err
	}
	if err := o.native.started(); err != nil {
		_ = o.Terminate()
		_ = o.cmd.Process.Kill()
		_ = o.cmd.Wait()
		return err
	}
	return nil
}

func (o *Owner) Terminate() error { return o.native.terminate() }

func (o *Owner) Close() error {
	// Stop owned processes and restore foreground ownership first, then restore
	// the pre-child modes while this process again owns the terminal. Tea must
	// receive this canonical state before it captures a new raw-mode baseline.
	err := o.native.close()
	for _, saved := range o.terminals {
		err = errors.Join(err, term.Restore(saved.file.Fd(), saved.state))
	}
	return err
}
