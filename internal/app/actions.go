package app

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func (e *Engine) Command(ctx context.Context, action, id string) (*exec.Cmd, error) {
	if ctx.Err() != nil {
		return nil, canceled()
	}
	if action == "ssh" {
		if runtime.GOOS != "windows" || e.options.SSHCommand == nil {
			return nil, errors.New("SSH setup handoff is available only through the explicit Windows command")
		}
		command, err := e.options.SSHCommand(ctx)
		if ctx.Err() != nil {
			return nil, canceled()
		}
		if err != nil || command == nil {
			return nil, errors.New("Windows SSH handoff unavailable; inspect its separate setup workflow")
		}
		return command, nil
	}
	if action != "launch" || id != "codex" {
		return nil, errors.New("this action has no verified native implementation; no installer or updater was started")
	}
	if _, _, err := e.readState(); err != nil {
		return nil, err
	}
	before, err := configtxn.Read(filepath.Join(e.options.CodexHome, "config.toml"))
	if err != nil {
		return nil, errors.New("selected native configuration cannot be verified")
	}
	c, err := e.inspectCodex(ctx)
	if err != nil {
		return nil, err
	}
	if !c.Supported || core.ValidateEndpoint(c.Profile.BaseURL) != nil {
		return nil, errors.New("configure a supported native Codex endpoint before launch")
	}
	if !e.expected(before) {
		return nil, errors.New("selected native configuration changed during inspection")
	}
	program := copyProgram(e.deps.program())
	identity, err := e.deps.identity(program)
	if err != nil {
		return nil, errors.New("selected native executable unavailable")
	}
	// Resolve credentials immediately before command construction. Neither the
	// pending review mutex nor a terminal is held while an external resolver waits.
	key, _, err := e.deps.credentials.Resolve(ctx, c.Profile.EnvKey)
	if ctx.Err() != nil {
		return nil, canceled()
	}
	if err != nil {
		return nil, errors.New("credential unavailable; unlock its storage or enter a session key")
	}
	env := e.deps.childEnv(c.Profile.EnvKey, key)
	if env == nil || len(env) == 0 {
		return nil, errors.New("credential delivery refused an unsafe child environment; no client was started")
	}
	// Recheck both identities after the last credential wait.
	after, err := e.deps.identity(program)
	if ctx.Err() != nil {
		return nil, canceled()
	}
	if err != nil || after != identity || !reflect.DeepEqual(program, copyProgram(e.deps.program())) || !e.expected(before) {
		return nil, errors.New("native executable or configuration changed during launch preparation")
	}
	if _, _, err = e.readState(); err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, canceled()
	}
	selected := make([]string, 0, len(env)+1)
	for _, entry := range env {
		name, _, ok := strings.Cut(entry, "=")
		if ok && !strings.EqualFold(name, "CODEX_HOME") && !strings.EqualFold(name, "CODEX_PROFILE") {
			selected = append(selected, entry)
		}
	}
	selected = append(selected, "CODEX_HOME="+e.options.CodexHome)
	command := exec.CommandContext(ctx, program.Path, program.Args...)
	command.Env = selected
	command.WaitDelay = time.Second
	return command, nil
}
func (e *Engine) SaveKey(ctx context.Context, key string, persistent bool) error {
	if ctx.Err() != nil {
		return canceled()
	}
	var err error
	if persistent {
		err = e.deps.credentials.Save(ctx, key)
	} else {
		err = e.deps.credentials.SetSession(key)
	}
	if err != nil {
		return errors.New("credential could not be saved; use a valid session key or unlock supported storage")
	}
	if ctx.Err() != nil {
		return canceled()
	}
	return nil
}
func (e *Engine) Discover(ctx context.Context, id string) ([]core.ModelInfo, error) {
	if ctx.Err() != nil {
		return nil, canceled()
	}
	if id != "codex" {
		return nil, errors.New("model discovery requires a verified supported client")
	}
	if _, _, err := e.readState(); err != nil {
		return nil, err
	}
	c, err := e.inspectCodex(ctx)
	if err != nil {
		return nil, err
	}
	if !c.Supported || core.ValidateEndpoint(c.Profile.BaseURL) != nil {
		return nil, errors.New("configure a supported endpoint before explicit discovery")
	}
	key, _, err := e.deps.credentials.Resolve(ctx, c.Profile.EnvKey)
	if ctx.Err() != nil {
		return nil, canceled()
	}
	if err != nil {
		return nil, errors.New("discovery credential unavailable; enter a session key")
	}
	base, err := e.deps.discover(ctx, c.Profile.BaseURL, key)
	if ctx.Err() != nil {
		return nil, canceled()
	}
	if err != nil {
		return nil, errors.New("model discovery failed; check endpoint access and credential availability")
	}
	// Retain exact inventory IDs and order even when optional metadata is absent
	// or an enrichment result does not match the requested identity set.
	if !validModels(base) {
		return nil, errors.New("model inventory contained invalid or duplicate IDs")
	}
	base = append([]core.ModelInfo{}, base...)
	enriched, metadataErr := e.deps.enrich(ctx, append([]core.ModelInfo{}, base...))
	if ctx.Err() != nil {
		return nil, canceled()
	}
	if metadataErr == nil && validModels(enriched) && len(enriched) == len(base) {
		for i := range base {
			if enriched[i].ID != base[i].ID {
				return base, nil
			}
		}
		return enriched, nil
	}
	return base, nil
}
func validModels(models []core.ModelInfo) bool {
	if len(models) > 10000 {
		return false
	}
	seen := map[string]bool{}
	for _, m := range models {
		if m.ID == "" || len(m.ID) > 256 || !utf8.ValidString(m.ID) || strings.IndexFunc(m.ID, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 || seen[m.ID] {
			return false
		}
		seen[m.ID] = true
	}
	return true
}
func (e *Engine) Service(ctx context.Context, action string) (core.ServiceInfo, error) {
	if ctx.Err() != nil {
		return unavailableService(), canceled()
	}
	switch action {
	case "status", "install", "start", "stop", "restart", "uninstall":
	default:
		return unavailableService(), errors.New("unknown observer lifecycle action")
	}
	if e.deps.monitor == nil {
		if action == "status" {
			return unavailableService(), nil
		}
		return unavailableService(), errors.New("native observer backend unavailable; foreground management remains usable")
	}
	if _, _, err := e.readState(); err != nil {
		return unavailableService(), err
	}
	if action == "status" {
		s := e.deps.monitor.Status(ctx)
		if ctx.Err() != nil {
			return unavailableService(), canceled()
		}
		return safeService(s), nil
	}
	s, err := e.deps.monitor.Action(ctx, action)
	if ctx.Err() != nil {
		return safeService(s), canceled()
	}
	if err != nil {
		return safeService(s), errors.New("observer lifecycle did not complete; inspect native user backend state before retrying")
	}
	return safeService(s), nil
}
func (e *Engine) Observe(ctx context.Context, once bool) error {
	if ctx.Err() != nil {
		return canceled()
	}
	if e.deps.monitor == nil {
		return errors.New("observer backend unavailable")
	}
	// Observation intentionally does not read or recover configuration journals.
	if err := e.deps.monitor.Observe(ctx, once); err != nil {
		if ctx.Err() != nil {
			return canceled()
		}
		return errors.New("observer could not safely sample the selected configuration")
	}
	if ctx.Err() != nil {
		return canceled()
	}
	return nil
}
func (e *Engine) RepairBrief(ctx context.Context) (string, error) {
	s, err := e.Inspect(ctx)
	if err != nil {
		return "", err
	}
	id, err := randomID()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(struct {
		Version  int           `json:"version"`
		Purpose  string        `json:"purpose"`
		Snapshot core.Snapshot `json:"snapshot"`
	}{1, "Manual redacted diagnostic review; no automatic repair or coding agent was started.", s}, "", "  ")
	if err != nil || len(data) > stateLimit {
		return "", errors.New("redacted diagnostic export exceeds its limit")
	}
	if ctx.Err() != nil {
		return "", canceled()
	}
	path := filepath.Join(e.options.StateRoot, "diagnostics", "brief-"+id+".json")
	if err = configtxn.PrivateFile(path, append(data, '\n')); err != nil {
		return "", errors.New("private diagnostic export failed; no repair was started")
	}
	return string(data) + "\n", nil
}
func unavailableService() core.ServiceInfo {
	return core.ServiceInfo{Backend: "Unavailable", Status: "Unavailable", Detail: "Optional observer is unavailable; foreground management remains usable."}
}
func safeService(s core.ServiceInfo) core.ServiceInfo {
	out := unavailableService()
	switch s.Backend {
	case "systemd user", "systemd --user", "LaunchAgent", "launchd LaunchAgent", "Background task", "Scheduled Task (current user logon)", "Task Scheduler (current user)", "user-session observer", "Unavailable":
		out.Backend = s.Backend
	}
	switch s.Status {
	case "Not installed", "Installed", "Running", "Stopped", "Failed", "Unavailable", "Conflict", "Unknown", "Transitioning":
		out.Status = s.Status
	}
	// Native command/error text is never part of the public projection.
	out.Detail = "Checked " + time.Now().UTC().Format(time.RFC3339) + "; optional native user-session observer."
	switch out.Status {
	case "Unavailable":
		out.Detail = "Native user-session observer unavailable; foreground management remains usable."
	case "Failed":
		out.Detail = "Native user-session observer failed; inspect its native status before restarting."
	case "Conflict":
		out.Detail = "Foreign or unsafe observer state was preserved; reconcile locally."
	case "Unknown":
		out.Detail = "Lifecycle outcome is uncertain; inspect before reconciliation and do not automatically retry."
	}
	return out
}
