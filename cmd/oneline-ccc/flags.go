package main

import (
	"errors"
	"flag"
	"io"
	"strings"
)

type flags struct {
	stateDir, codexHome, codexExecutable, theme                string
	ascii, plain, reducedMotion, help, showVersion, json, once bool
	command                                                    string
	arguments                                                  []string
}

var errArguments = errors.New("invalid command or options; use oneline-ccc --help")

func parse(args []string) (flags, error) {
	var o flags
	fs := flag.NewFlagSet("oneline-ccc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.stateDir, "state-dir", "", "private application state directory")
	fs.StringVar(&o.codexHome, "codex-home", "", "explicit Codex configuration directory")
	fs.StringVar(&o.codexExecutable, "codex-executable", "", "explicit Codex executable")
	fs.StringVar(&o.theme, "theme", "dark", "dark, light or mono")
	fs.BoolVar(&o.ascii, "ascii", false, "ASCII decoration")
	fs.BoolVar(&o.plain, "plain", false, "plain read-only status instead of TUI")
	fs.BoolVar(&o.reducedMotion, "reduced-motion", false, "static activity indication")
	fs.BoolVar(&o.help, "help", false, "help without opening application state")
	fs.BoolVar(&o.help, "h", false, "help")
	fs.BoolVar(&o.showVersion, "version", false, "version without opening application state")
	fs.BoolVar(&o.json, "json", false, "status JSON")
	fs.BoolVar(&o.once, "once", false, "one observation pass")
	// stdlib flag parsing stops at the first operand. Collect known flags first
	// so global options remain available before or after a command, with values
	// kept verbatim and never echoed in parser errors.
	var options, operands []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			operands = append(operands, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			operands = append(operands, arg)
			continue
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		f := fs.Lookup(name)
		if f == nil {
			return o, errArguments
		}
		options = append(options, arg)
		boolean, ok := f.Value.(interface{ IsBoolFlag() bool })
		if !hasValue && !(ok && boolean.IsBoolFlag()) {
			i++
			if i >= len(args) {
				return o, errArguments
			}
			options = append(options, args[i])
		}
	}
	if fs.Parse(options) != nil {
		return o, errArguments
	}
	if o.help || o.showVersion {
		return o, nil
	}
	invalidPath := false
	fs.Visit(func(f *flag.Flag) {
		if (f.Name == "state-dir" || f.Name == "codex-home" || f.Name == "codex-executable") && f.Value.String() == "" {
			invalidPath = true
		}
	})
	if invalidPath {
		return o, errArguments
	}
	if o.theme != "dark" && o.theme != "light" && o.theme != "mono" {
		return o, errArguments
	}
	if len(operands) > 0 {
		o.command = operands[0]
		o.arguments = operands[1:]
	}
	if o.command == "" && o.json {
		o.command = "status"
	}
	switch o.command {
	case "", "status", "doctor", "version", "recover", "ssh", "repair":
		if len(o.arguments) != 0 {
			return o, errArguments
		}
	case "launch":
		if len(o.arguments) != 1 || o.arguments[0] != "codex" {
			return o, errArguments
		}
	case "service":
		if len(o.arguments) != 1 {
			return o, errArguments
		}
		switch o.arguments[0] {
		case "status", "install", "start", "stop", "restart", "uninstall":
		default:
			return o, errArguments
		}
	case "observe":
		if len(o.arguments) != 0 {
			return o, errArguments
		}
	default:
		return o, errArguments
	}
	if o.json && o.command != "status" || o.once && o.command != "observe" {
		return o, errArguments
	}
	return o, nil
}

const usage = `oneline-ccc - native Codex configuration console

Usage: oneline-ccc [global options] [command]
  status [--json]       Inspect current state; no configuration writes
  doctor               Inspect native Codex readiness; no inference requests
  version              Print build version without opening state
  launch codex         Run verified native client; forward terminal streams
  service ACTION       status/install/start/stop/restart/uninstall
  observe [--once]     Local observation only; cancellation stops the loop
  recover              Explicitly reconcile interrupted transactions
  ssh                  Explicit Windows administrator SSH setup handoff
  repair               Print a redacted manual repair brief; execute no agent

Global options:
  --state-dir PATH        Private application state root
  --codex-home PATH       Codex home (otherwise CODEX_HOME or user/.codex)
  --codex-executable PATH Explicit native client executable
  --theme dark|light|mono --ascii --reduced-motion
  --plain                Plain read-only status when no command is supplied
  --help, -h             This help; no state or services opened
  --version              Build version; no state or services opened

No command starts the TUI when stdin and stdout are terminals and TERM is not dumb.
Redirected stdin/stdout, --plain and TERM=dumb use escape-free plain status.
NO_COLOR disables color styling while retaining TUI controls and management.
The TUI still uses cursor positioning; use --plain for escape-free output.
TUI controls: Tab/Shift+Tab, Enter, Space, Esc, ?, Ctrl+P; q quits safely.
Apply is available only through an explicitly reviewed TUI plan.
Source build: go build -o oneline-ccc ./cmd/oneline-ccc
`
