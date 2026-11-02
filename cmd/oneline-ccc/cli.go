package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/tui"
)

var version = "0.1.0-dev" // Override with -ldflags '-X main.version=...'.
type engine interface {
	core.Backend
	Recover(context.Context) ([]configtxn.Result, error)
	Observe(context.Context, bool) error
}
type engineOptions struct{ StateRoot, CodexHome, CodexExecutable, Executable string }
type dependencies struct {
	in                               io.Reader
	out, errout                      io.Writer
	terminal                         bool
	lookup                           func(string) (string, bool)
	userHome, userConfig, executable func() (string, error)
	newEngine                        func(engineOptions) (engine, error)
	runUI                            func(core.Backend, tui.Options) error
	runChild                         func(context.Context, *exec.Cmd, io.Reader, io.Writer, io.Writer) error
}

func run(ctx context.Context, args []string, d dependencies) int {
	o, err := parse(args)
	if err != nil {
		fmt.Fprintln(d.errout, errArguments)
		return 2
	}
	if o.help {
		_, err := fmt.Fprint(d.out, usage)
		return resultCode(d, "Help output failed", err)
	}
	if o.showVersion || o.command == "version" {
		_, err := fmt.Fprintln(d.out, "oneline-ccc "+safe(version))
		return resultCode(d, "Version output failed", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(d, "Action not started", err)
	}
	opts, err := resolve(o, d)
	if err != nil {
		return fail(d, "Application paths could not be resolved", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(d, "Action not started", err)
	}
	e, err := d.newEngine(opts)
	if err != nil {
		return fail(d, "Application could not open protected state", err)
	}
	if useTUI(o, d) {
		_, noColor := d.lookup("NO_COLOR")
		return resultCode(d, "Terminal UI stopped", operationError(ctx, d.runUI(e, tui.Options{Theme: o.theme, ASCII: o.ascii, NoColor: noColor, ReducedMotion: o.reducedMotion, Version: version})))
	}
	if o.command == "" {
		o.command = "status"
	}
	if o.command == "launch" || o.command == "ssh" {
		return launch(ctx, o, e, d)
	}
	if o.command == "observe" && !o.once {
		return resultCode(d, "Observation stopped", operationError(ctx, e.Observe(ctx, false)))
	}
	bounded, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	switch o.command {
	case "status", "doctor":
		snapshot, err := e.Inspect(bounded)
		if err != nil {
			return fail(d, "Status inspection failed", operationError(bounded, err))
		}
		if err = printStatus(d.out, snapshot, o.json, o.command == "doctor"); err != nil {
			return fail(d, "Status output failed", err)
		}
		if o.command == "doctor" {
			for _, c := range snapshot.Clients {
				if c.ID == "codex" && c.Installed && c.Supported {
					return 0
				}
			}
			return 1
		}
	case "service":
		info, err := e.Service(bounded, o.arguments[0])
		if err != nil {
			return fail(d, "Monitor action unavailable or failed", operationError(bounded, err))
		}
		_, err = fmt.Fprintf(d.out, "Monitor backend: %s\nObserved status: %s\n%s\n", safe(info.Backend), safe(info.Status), safe(info.Detail))
		return resultCode(d, "Monitor output failed", err)
	case "observe":
		return resultCode(d, "Observation failed", operationError(bounded, e.Observe(bounded, true)))
	case "recover":
		receipts, err := e.Recover(bounded)
		var outputErr error
		for _, r := range receipts {
			_, e := fmt.Fprintf(d.out, "Recovery %s: %s (%d files, %d backups)\n", safe(r.ID), safe(r.Outcome), len(r.Changed), len(r.Backups))
			if outputErr == nil {
				outputErr = e
			}
		}
		if err != nil {
			return fail(d, "Recovery requires attention", operationError(bounded, err))
		}
		if len(receipts) == 0 {
			_, outputErr = fmt.Fprintln(d.out, "No interrupted transactions found")
		}
		return resultCode(d, "Recovery output failed", outputErr)
	case "repair":
		brief, err := e.RepairBrief(bounded)
		if err != nil {
			return fail(d, "Manual repair brief unavailable", operationError(bounded, err))
		}
		_, err = fmt.Fprintln(d.out, safe(brief))
		return resultCode(d, "Repair output failed", err)
	}
	return 0
}
func resolve(o flags, d dependencies) (engineOptions, error) {
	opts := engineOptions{StateRoot: o.stateDir, CodexHome: o.codexHome, CodexExecutable: o.codexExecutable}
	var err error
	if opts.StateRoot == "" {
		root, e := d.userConfig()
		if e != nil || root == "" {
			return opts, errArguments
		}
		opts.StateRoot = filepath.Join(root, "oneline-ccc")
	}
	if opts.CodexHome == "" {
		if value, ok := d.lookup("CODEX_HOME"); ok && value != "" {
			opts.CodexHome = value
		} else {
			home, e := d.userHome()
			if e != nil || home == "" {
				return opts, errArguments
			}
			opts.CodexHome = filepath.Join(home, ".codex")
		}
	}
	opts.Executable, err = d.executable()
	if err != nil || opts.Executable == "" {
		return opts, errArguments
	}
	opts.StateRoot, err = filepath.Abs(opts.StateRoot)
	if err != nil {
		return opts, err
	}
	opts.CodexHome, err = filepath.Abs(opts.CodexHome)
	return opts, err
}
func resultCode(d dependencies, label string, err error) int {
	if err == nil {
		return 0
	}
	return fail(d, label, err)
}
func fail(d dependencies, label string, err error) int {
	// Never forward provider/OS error strings: they can include endpoints,
	// environment values, executable paths or raw credential responses.
	suffix := "; use the redacted repair brief for diagnostics"
	switch {
	case errors.Is(err, context.Canceled):
		suffix = "; cancellation requested"
	case errors.Is(err, context.DeadlineExceeded):
		suffix = "; operation timed out"
	case errors.Is(err, configtxn.ErrUnsupported):
		suffix = "; protected native filesystem support is unavailable on this platform"
	}
	fmt.Fprintln(d.errout, label+suffix)
	if errors.Is(err, context.Canceled) {
		return 130
	}
	return 1
}

func operationError(ctx context.Context, err error) error {
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
