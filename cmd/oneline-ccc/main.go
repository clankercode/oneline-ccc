package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	onelineccc "github.com/clankercode/oneline-ccc"
	"github.com/clankercode/oneline-ccc/internal/app"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/tui"
)

func main() {
	code := func() int {
		ctx, stop := watchSignals(context.Background(), tuiRequested(os.Args[1:], production(context.Background())))
		defer stop()
		return run(ctx, os.Args[1:], production(ctx))
	}()
	os.Exit(code)
}
func production(ctx context.Context) dependencies {
	return dependencies{
		in: os.Stdin, out: os.Stdout, errout: os.Stderr,
		terminal: term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd()),
		lookup:   os.LookupEnv, userHome: os.UserHomeDir, userConfig: os.UserConfigDir, executable: os.Executable,
		newEngine: func(o engineOptions) (engine, error) {
			return app.New(app.Options{StateRoot: o.StateRoot, CodexHome: o.CodexHome, CodexExecutable: o.CodexExecutable, Executable: o.Executable, SSHCommand: func(commandContext context.Context) (*exec.Cmd, error) {
				return onelineccc.SSHCommand(commandContext, o.StateRoot)
			}})
		},
		runUI: func(backend core.Backend, options tui.Options) error {
			return runForeground(ctx, backend, options, func(model tea.Model) error {
				return runForegroundProgram(ctx, model)
			})
		},
		runChild: runProcess,
	}
}

// runForegroundProgram lets root cancellation stop owned work immediately while
// Tea retains ownership of its terminal reader until graceful shutdown. Passing
// the root directly to WithContext forces Tea's killed shutdown path, which
// closes the cancel reader without joining the read loop after tea.Exec resumes.
func runForegroundProgram(ctx context.Context, model tea.Model, options ...tea.ProgramOption) error {
	programContext, cancel := context.WithCancel(context.WithoutCancel(ctx))
	options = append([]tea.ProgramOption{tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)}, options...)
	options = append(options, tea.WithContext(programContext), tea.WithoutSignalHandler())
	program := tea.NewProgram(model, options...)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			program.Quit()
		case <-programContext.Done():
		}
	}()
	defer func() {
		// Also release a Quit blocked during startup if Run returns early.
		cancel()
		<-stopped
	}()
	_, err := program.Run()
	return errors.Join(err, ctx.Err())
}

// runForeground is the production boundary between the terminal program and
// main's exit. Tests can supply a headless Bubble Tea program at this same seam.
func runForeground(ctx context.Context, backend core.Backend, options tui.Options, program func(tea.Model) error) error {
	root, cancel := context.WithCancel(ctx)
	session := &foregroundSession{cancel: cancel}
	bound := rootBoundBackend{Backend: backend, root: root, session: session}
	defer session.shutdown()
	options.Context = root
	runner := options.Runner
	if runner == nil {
		runner = runProcess
	}
	options.Runner = func(caller context.Context, command *exec.Cmd, in io.Reader, out, errout io.Writer) error {
		_, err := foregroundCall(bound, caller, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, runner(ctx, command, in, out, errout)
		})
		return err
	}
	return program(tui.New(bound, options))
}
