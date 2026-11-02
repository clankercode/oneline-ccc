// Package app implements explicit foreground actions over client adapters.
package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/clankercode/oneline-ccc/internal/adapters/codex"
	"github.com/clankercode/oneline-ccc/internal/catalog"
	"github.com/clankercode/oneline-ccc/internal/codexrpc"
	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/credentials"
	"github.com/clankercode/oneline-ccc/internal/monitor"
)

type Options struct {
	StateRoot, CodexHome, CodexExecutable, Executable string
	SSHCommand                                        func(context.Context) (*exec.Cmd, error)
}
type adapter interface {
	Inspect(context.Context) (core.Client, error)
	Prepare(context.Context, core.Profile, []byte) ([]configtxn.Write, error)
}
type credentialStore interface {
	SetSession(string) error
	Save(context.Context, string) error
	Resolve(context.Context, string) (string, string, error)
	Status(context.Context, string) string
}
type serviceManager interface {
	Status(context.Context) core.ServiceInfo
	Action(context.Context, string) (core.ServiceInfo, error)
	Observe(context.Context, bool) error
}
type transactionStore interface {
	Apply(context.Context, []configtxn.Write) (configtxn.Result, error)
	Recover(context.Context) ([]configtxn.Result, error)
}
type activeApply struct {
	cancel context.CancelFunc
}
type dependencies struct {
	adapter     adapter
	program     func() codexrpc.Program
	credentials credentialStore
	childEnv    func(string, string) []string
	lookup      func(string) (string, error)
	identity    func(codexrpc.Program) (string, error)
	discover    func(context.Context, string, string) ([]core.ModelInfo, error)
	enrich      func(context.Context, []core.ModelInfo) ([]core.ModelInfo, error)
	confirmed   func(context.Context, string, string, []string) ([]byte, error)
	monitor     serviceManager
}
type Engine struct {
	options  Options
	deps     dependencies
	mu       sync.Mutex
	pending  map[string]*review
	epoch    map[string]uint64
	applying map[string]*activeApply
	mutating chan struct{}
	store    transactionStore
}

var _ core.Backend = (*Engine)(nil)

// New verifies app state only. Client creation, recovery and service activation
// require their explicit foreground operations.
func New(o Options) (*Engine, error) {
	if o.StateRoot == "" || o.CodexHome == "" {
		return nil, errors.New("choose explicit application and Codex state roots")
	}
	a := codex.New(o.CodexHome, filepath.Join(o.StateRoot, "staging"), o.CodexExecutable)
	d := dependencies{adapter: a, program: func() codexrpc.Program { return a.Program }, credentials: credentials.New(), childEnv: credentials.ChildEnv, lookup: exec.LookPath}
	e, err := newEngine(o, d)
	if err != nil {
		return nil, err
	}
	executable := o.Executable
	if executable == "" {
		executable, _ = os.Executable()
	}
	e.deps.monitor = monitor.New(e.options.StateRoot, e.options.CodexHome, executable)
	return e, nil
}
func newEngine(o Options, d dependencies) (*Engine, error) {
	if o.StateRoot == "" || o.CodexHome == "" {
		return nil, errors.New("choose explicit application and Codex state roots")
	}
	var err error
	if o.StateRoot, err = filepath.Abs(o.StateRoot); err != nil {
		return nil, errors.New("application state root unavailable")
	}
	if o.CodexHome, err = filepath.Abs(o.CodexHome); err != nil {
		return nil, errors.New("Codex root unavailable")
	}
	if within(o.StateRoot, o.CodexHome) || within(o.CodexHome, o.StateRoot) {
		return nil, errors.New("application state and Codex configuration must have separate roots")
	}
	if d.adapter == nil || d.program == nil || d.credentials == nil || d.childEnv == nil {
		return nil, errors.New("application dependencies unavailable")
	}
	if d.lookup == nil {
		d.lookup = exec.LookPath
	}
	if d.identity == nil {
		d.identity = codexrpc.Identity
	}
	if d.discover == nil {
		d.discover = catalog.Discover
	}
	if d.enrich == nil {
		d.enrich = catalog.Enrich
	}
	if d.confirmed == nil {
		d.confirmed = catalog.ConfirmedCodex
	}
	if configtxn.PrivateDir(o.StateRoot) != nil {
		return nil, errors.New("application state root must be private and safely owned")
	}
	e := &Engine{options: o, deps: d, pending: map[string]*review{}, epoch: map[string]uint64{}, applying: map[string]*activeApply{}, mutating: make(chan struct{}, 1), store: configtxn.New(filepath.Join(o.StateRoot, "transactions"))}
	if _, _, err = e.readState(); err != nil {
		return nil, err
	}
	return e, nil
}
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`)
var clients = []struct{ id, name, command string }{
	{"codex", "Codex", "codex"}, {"claude", "Claude Code", "claude"}, {"pi", "Pi", "pi"},
	{"opencode", "OpenCode", "opencode"}, {"grokbuild", "Grok Build", "grok"}, {"hermes", "Hermes Agent", "hermes"},
}

func clientName(id string) string {
	for _, c := range clients {
		if c.id == id {
			return c.name
		}
	}
	return ""
}
func (e *Engine) Inspect(ctx context.Context) (core.Snapshot, error) {
	if ctx.Err() != nil {
		return core.Snapshot{}, canceled()
	}
	s, _, err := e.readState()
	if err != nil {
		return core.Snapshot{}, err
	}
	c, err := e.inspectCodex(ctx)
	if err != nil {
		return core.Snapshot{}, err
	}
	c.Managed = s.Managed["codex"]
	out := core.Snapshot{Clients: []core.Client{c}, Receipts: append([]core.Receipt{}, s.Receipts...), Credential: "Missing", Service: unavailableService()}
	for _, spec := range clients[1:] {
		_, lookupErr := e.deps.lookup(spec.command)
		next := core.Client{ID: spec.id, Name: spec.name, Installed: lookupErr == nil, Managed: s.Managed[spec.id], Status: "NOT INSTALLED", Reason: "Client executable not found on PATH; installation is not automatic."}
		if next.Installed {
			next.Status = "READ ONLY"
			next.Reason = "PATH candidate detected; identity, version and native adapter are unverified. Configuration and launch unavailable."
		}
		out.Clients = append(out.Clients, next)
	}
	out.Credential = safeCredentialStatus(e.deps.credentials.Status(ctx, c.Profile.EnvKey))
	if e.deps.monitor != nil {
		out.Service = safeService(e.deps.monitor.Status(ctx))
	}
	if ctx.Err() != nil {
		return core.Snapshot{}, canceled()
	}
	// Never project stale/invalid state after awaiting native or credential helpers.
	s, _, err = e.readState()
	if err != nil {
		return core.Snapshot{}, err
	}
	for i := range out.Clients {
		out.Clients[i].Managed = s.Managed[out.Clients[i].ID]
		if out.Clients[i].ID == "codex" {
			out.Clients[i].Profile = restoreAttestation(out.Clients[i].Profile, out.Clients[i].Version, out.Clients[i].Managed, s.Attestations["codex"])
		}
	}
	out.Receipts = append([]core.Receipt{}, s.Receipts...)
	return out, nil
}
func (e *Engine) inspectCodex(ctx context.Context) (core.Client, error) {
	c, err := e.deps.adapter.Inspect(ctx)
	if ctx.Err() != nil {
		return core.Client{}, canceled()
	}
	if err != nil {
		return core.Client{}, errors.New("Codex inspection failed; verify its native executable and configuration")
	}
	c.ID = "codex"
	c.Name = "Codex"
	c.Managed = false
	c.Profile.CapacityConfirmed = false
	c.Profile.TiersConfirmed = false
	if len(c.Version) > 128 || !versionPattern.MatchString(c.Version) {
		c.Version = ""
	}
	c.Supported = c.Installed && c.Supported && c.Version == "0.166.0"
	if !c.Installed {
		c.Status = "NOT INSTALLED"
		c.Reason = "Native Codex executable not found; select or install the native client."
	} else if !c.Supported {
		c.Status = "READ ONLY"
		c.Reason = "Only verified native Codex 0.166.0 configuration is eligible; inspect its version and supported schema."
	} else {
		c.Reason = ""
		if c.Profile.BaseURL == "" {
			c.Status = "NEEDS SETUP"
		} else {
			c.Status = "CONFIGURED"
		}
	}
	return c, nil
}
func safeCredentialStatus(s string) string {
	switch s {
	case "Missing", "Locked", "Stored", "Session only":
		return s
	}
	return "Locked"
}
func canceled() error { return errors.New("operation canceled; no new action was authorized") }
func (e *Engine) lockMutation(ctx context.Context) error {
	if ctx.Err() != nil {
		return canceled()
	}
	select {
	case e.mutating <- struct{}{}:
		return nil
	case <-ctx.Done():
		return canceled()
	}
}
func (e *Engine) Recover(ctx context.Context) ([]configtxn.Result, error) {
	if err := e.lockMutation(ctx); err != nil {
		return nil, err
	}
	defer func() { <-e.mutating }()
	results, err := e.store.Recover(ctx)
	if err != nil {
		return nil, errors.New("configuration recovery needs attention; preserve files and inspect the recovery status")
	}
	e.mu.Lock()
	e.pending = map[string]*review{}
	for id := range e.epoch {
		e.epoch[id]++
	}
	e.mu.Unlock()
	return results, nil
}
