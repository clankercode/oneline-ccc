package monitor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

func prepareServiceDir(path string) error {
	if safeComponents(path) != nil {
		return errUnsafe
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if prepareServiceDir(filepath.Dir(path)) != nil {
			return errUnsafe
		}
		if err = os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
			return errIO
		}
		info, err = os.Lstat(path)
	}
	if err != nil || !info.IsDir() || !privateInfo(info) || info.Mode().Perm()&0022 != 0 {
		return errUnsafe
	}
	return nil
}
func (m *Manager) command(ctx context.Context, spec serviceSpec, args ...string) error {
	if ctx.Err() != nil {
		return errUncertain
	}
	present, owned, err := m.checkArtifact(spec)
	allowMissing := m.os == "linux" && len(args) > 3 && args[3] == "daemon-reload"
	if err != nil || !owned || (!present && !allowMissing) {
		return errUncertain
	}
	result, err := m.call(ctx, args...)
	if err != nil || result.Code != 0 || ctx.Err() != nil {
		return errUncertain
	}
	return nil
}
func (m *Manager) systemctl(ctx context.Context, spec serviceSpec, action string) error {
	args := []string{"--user", "--no-pager", "--no-ask-password", action}
	if action != "daemon-reload" {
		args = append(args, unitName)
	}
	return m.command(ctx, spec, args...)
}
func (m *Manager) removeArtifact(spec serviceSpec) error {
	present, owned, err := m.checkArtifact(spec)
	if err != nil || !owned {
		return errUnsafe
	}
	if !present {
		return nil
	}
	if os.Remove(spec.artifact) != nil {
		return errIO
	}
	return syncParent(spec.artifact)
}
func syncParent(path string) error {
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return errIO
	}
	defer dir.Close()
	if dir.Sync() != nil {
		return errIO
	}
	return nil
}
func (m *Manager) clearPending() error {
	path := filepath.Join(m.root(), "pending.json")
	if _, exists, err := privateRead(path); err != nil {
		return err
	} else if !exists {
		return nil
	}
	if os.Remove(path) != nil {
		return errIO
	}
	return syncParent(path)
}
func resolved(action string, before string, state nativeState, osName string) bool {
	switch action {
	case "install":
		return state.owned && state.artifact && state.loaded && (osName != "linux" || state.enabled)
	case "start":
		return state.owned && state.artifact && state.loaded && state.active
	case "stop":
		return !state.active && state.phase == ""
	case "restart":
		return state.owned && state.artifact && state.loaded && state.active && state.invocation != "" && state.invocation != before
	case "uninstall":
		return !state.artifact && !state.loaded
	}
	return false
}
func (m *Manager) execute(ctx context.Context, action string, spec serviceSpec, state nativeState) error {
	if action == "install" {
		if prepareServiceDir(filepath.Dir(spec.artifact)) != nil {
			return errUncertain
		}
		snap, err := configtxn.Read(spec.artifact)
		if err != nil {
			return errUncertain
		}
		if snap.Existed && !bytes.Equal(snap.Before, spec.definition) {
			return errUncertain
		}
		snap.Data = spec.definition
		if _, err = configtxn.New(filepath.Join(m.root(), "transactions")).Apply(ctx, []configtxn.Write{snap}); err != nil {
			return errUncertain
		}
		if m.os == "linux" {
			if err = m.systemctl(ctx, spec, "daemon-reload"); err != nil {
				return err
			}
			return m.systemctl(ctx, spec, "enable")
		}
		if state.loaded {
			return nil
		}
		return m.command(ctx, spec, "bootstrap", "gui/"+m.uid, spec.artifact)
	}
	if m.os == "linux" {
		if action == "uninstall" {
			if state.active {
				if err := m.systemctl(ctx, spec, "stop"); err != nil {
					return err
				}
			}
			if state.enabled {
				if err := m.systemctl(ctx, spec, "disable"); err != nil {
					return err
				}
			}
			if err := m.removeArtifact(spec); err != nil {
				return errUncertain
			}
			return m.systemctl(ctx, spec, "daemon-reload")
		}
		return m.systemctl(ctx, spec, action)
	}
	switch action {
	case "start", "restart":
		if !state.loaded {
			return m.command(ctx, spec, "bootstrap", "gui/"+m.uid, spec.artifact)
		}
		args := []string{"kickstart"}
		if action == "restart" {
			args = append(args, "-k")
		}
		return m.command(ctx, spec, append(args, spec.target)...)
	case "stop":
		if state.loaded {
			return m.command(ctx, spec, "bootout", spec.target)
		}
		return nil
	case "uninstall":
		if state.loaded {
			if err := m.command(ctx, spec, "bootout", spec.target); err != nil {
				return err
			}
		}
		if m.removeArtifact(spec) != nil {
			return errUncertain
		}
		return nil
	}
	return errUnavailable
}

// Action executes one explicitly requested user-session lifecycle change. A
// durable reservation precedes native mutation. Uncertain operations are never
// retried unless a read-only native query proves the original postcondition;
// an unprovable result requires local reconciliation. Uninstall retains state.
func (m *Manager) Action(ctx context.Context, action string) (core.ServiceInfo, error) {
	if action == "status" {
		return m.Status(ctx), nil
	}
	switch action {
	case "install", "start", "stop", "restart", "uninstall":
	default:
		return m.failure(errUnavailable), errUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	fail := func(err error) (core.ServiceInfo, error) { return m.failure(err), err }
	if ctx.Err() != nil || !supportedFiles() || m.os == "windows" {
		return fail(errUnavailable)
	}
	spec, err := m.spec()
	if err != nil {
		return fail(err)
	}
	if _, err = m.inspect(ctx, spec); err != nil {
		return fail(err)
	}
	lock, err := m.lifecycleLock(ctx)
	if err != nil {
		return fail(err)
	}
	defer lock.Close()
	state, err := m.inspect(ctx, spec)
	if err != nil {
		return fail(err)
	}
	if data, exists, readErr := privateRead(filepath.Join(m.root(), "pending.json")); readErr != nil {
		return fail(readErr)
	} else if exists {
		var pending pendingAction
		if decodeRecord(data, &pending) != nil || pending.Version != 1 || !resolved(pending.Action, pending.BeforeInvocation, state, m.os) {
			return fail(errUncertain)
		}
		if err = m.clearPending(); err != nil {
			return fail(err)
		}
		if action == pending.Action {
			return m.info(state), nil
		}
	}
	if state.phase != "" {
		return fail(errUncertain)
	}
	noop := action == "install" && resolved(action, "", state, m.os) || action == "start" && state.active || action == "stop" && !state.active || (action == "uninstall" && !state.artifact && !state.loaded)
	if noop {
		return m.info(state), nil
	}
	if action != "install" && (!state.owned || !state.artifact) {
		return fail(errUnsafe)
	}
	if action == "install" || action == "start" || action == "restart" {
		info, statErr := os.Lstat(m.executable)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return fail(errUnsafe)
		}
	}
	if ctx.Err() != nil {
		return fail(errUnavailable)
	}
	if !state.owned {
		if err = atomicRecord(ctx, filepath.Join(m.root(), "ownership.json"), ownership{Version: 1, Digest: definitionDigest(spec)}); err != nil {
			return fail(err)
		}
	}
	pending := pendingAction{Version: 1, Action: action, BeforeInvocation: state.invocation}
	if err = atomicRecord(ctx, filepath.Join(m.root(), "pending.json"), pending); err != nil {
		return fail(err)
	}
	if err = m.execute(ctx, action, spec, state); err != nil {
		return fail(errUncertain)
	}
	after, err := m.inspect(ctx, spec)
	if err != nil || !resolved(action, state.invocation, after, m.os) {
		return fail(errUncertain)
	}
	if err = m.clearPending(); err != nil {
		return fail(errUncertain)
	}
	return m.info(after), nil
}
