package main

import (
	"context"
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
			_, err := tea.NewProgram(tui.New(rootBoundBackend{Backend: backend, root: ctx}, options), tea.WithContext(ctx), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout), tea.WithoutSignalHandler()).Run()
			return err
		},
		runChild: runProcess,
	}
}
