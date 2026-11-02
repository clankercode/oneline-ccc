package monitor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/clankercode/oneline-ccc/internal/core"
)

type ownership struct {
	Version int    `json:"version"`
	Digest  string `json:"definition_sha256"`
}
type pendingAction struct {
	Version          int    `json:"version"`
	Action           string `json:"action"`
	BeforeInvocation string `json:"before_invocation,omitempty"`
}
type nativeState struct {
	loaded, active, enabled, artifact, owned, failed bool
	phase, invocation                                string
}

func definitionDigest(spec serviceSpec) string {
	hash := sha256.Sum256(spec.definition)
	return hex.EncodeToString(hash[:])
}
func decodeRecord(data []byte, value any) error {
	if !strictJSON(data) {
		return errUnsafe
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(value) != nil {
		return errUnsafe
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errUnsafe
	}
	return nil
}
func (m *Manager) readOwner(spec serviceSpec) (bool, error) {
	data, exists, err := privateRead(filepath.Join(m.root(), "ownership.json"))
	if err != nil || !exists {
		return false, err
	}
	var owner ownership
	if decodeRecord(data, &owner) != nil || owner.Version != 1 || owner.Digest != definitionDigest(spec) {
		return false, errUnsafe
	}
	return true, nil
}
func (m *Manager) checkArtifact(spec serviceSpec) (present, owned bool, err error) {
	owned, err = m.readOwner(spec)
	if err != nil {
		return false, false, err
	}
	data, present, err := privateRead(spec.artifact)
	if err != nil {
		return false, owned, err
	}
	if present && (!owned || !bytes.Equal(data, spec.definition)) {
		return true, owned, errUnsafe
	}
	return present, owned, nil
}
func (m *Manager) call(ctx context.Context, args ...string) (commandResult, error) {
	name := "/usr/bin/systemctl"
	if m.os == "darwin" {
		name = "/bin/launchctl"
	}
	if m.runner == nil {
		return commandResult{}, errUnavailable
	}
	return m.runner(ctx, name, args...)
}
func (m *Manager) query(ctx context.Context, spec serviceSpec) (nativeState, error) {
	state := nativeState{}
	if m.os == "linux" {
		result, err := m.call(ctx, "--user", "--no-pager", "--no-ask-password", "show", unitName, "--property=LoadState,ActiveState,UnitFileState,FragmentPath,InvocationID")
		if err != nil || result.Code != 0 || len(result.Output) > recordLimit {
			return state, errUnavailable
		}
		fields := map[string]string{}
		for _, line := range strings.Split(strings.TrimSpace(result.Output), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				return state, errUnavailable
			}
			if _, dup := fields[key]; dup {
				return state, errUnavailable
			}
			fields[key] = value
		}
		if len(fields) != 5 {
			return state, errUnavailable
		}
		switch fields["LoadState"] {
		case "not-found":
			return state, nil
		case "loaded":
			state.loaded = true
		default:
			return state, errUnavailable
		}
		if fields["FragmentPath"] != spec.artifact {
			return state, errUnsafe
		}
		switch fields["ActiveState"] {
		case "active":
			state.active = true
		case "inactive":
		case "failed":
			state.failed = true
		case "activating", "deactivating", "reloading", "maintenance":
			state.phase = "Transitioning"
		default:
			return state, errUnavailable
		}
		switch fields["UnitFileState"] {
		case "enabled":
			state.enabled = true
		case "disabled", "static", "", "enabled-runtime":
		default:
			return state, errUnavailable
		}
		state.invocation = fields["InvocationID"]
		if state.invocation != "" {
			raw, err := hex.DecodeString(state.invocation)
			if err != nil || len(raw) != 16 || len(state.invocation) != 32 {
				return state, errUnavailable
			}
		}
		if state.active && state.invocation == "" {
			return state, errUnavailable
		}
	} else if m.os == "darwin" {
		result, err := m.call(ctx, "print", spec.target)
		if err != nil || len(result.Output) > recordLimit {
			return state, errUnavailable
		}
		if result.Code == 113 {
			return state, nil
		}
		if result.Code != 0 {
			return state, errUnavailable
		}
		fields := map[string]string{}
		for _, line := range strings.Split(result.Output, "\n") {
			line = strings.TrimSpace(line)
			key, value, ok := strings.Cut(line, " = ")
			if !ok {
				continue
			}
			switch key {
			case "path", "state", "pid":
				if _, duplicate := fields[key]; duplicate {
					return state, errUnavailable
				}
				fields[key] = value
			}
		}
		if fields["path"] != spec.artifact {
			return state, errUnsafe
		}
		state.loaded = true
		state.enabled = true
		state.invocation = fields["pid"]
		if state.invocation != "" {
			pid, err := strconv.ParseUint(state.invocation, 10, 32)
			if err != nil {
				return state, errUnavailable
			}
			if pid == 0 {
				state.invocation = ""
			}
		}
		switch fields["state"] {
		case "running":
			state.active = true
			if state.invocation == "" {
				return state, errUnavailable
			}
		case "not running", "waiting":
		default:
			return state, errUnavailable
		}
	} else {
		return state, errUnavailable
	}
	return state, nil
}
func (m *Manager) inspect(ctx context.Context, spec serviceSpec) (nativeState, error) {
	state := nativeState{}
	// Read-only check; do not chmod/adopt state paths while displaying status.
	for _, path := range []string{m.stateRoot, m.root()} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !privateInfo(info) {
			return state, errUnsafe
		}
	}
	present, owned, err := m.checkArtifact(spec)
	if err != nil {
		return state, err
	}
	state, err = m.query(ctx, spec)
	if err != nil {
		return state, err
	}
	if ctx.Err() != nil {
		return state, errUnavailable
	}
	// Querying a service manager is an await boundary. Recheck ownership and
	// exact artifact existence/bytes before projecting its state or mutating.
	currentPresent, currentOwned, checkErr := m.checkArtifact(spec)
	if checkErr != nil || currentPresent != present || currentOwned != owned {
		return state, errUnsafe
	}
	if state.loaded && !owned {
		return state, errUnsafe
	}
	state.artifact = present
	state.owned = owned
	return state, nil
}
func (m *Manager) info(state nativeState) core.ServiceInfo {
	status := "Not installed"
	if state.artifact || state.loaded {
		status = "Stopped"
		if state.active {
			status = "Running"
		}
		if state.failed {
			status = "Failed"
		}
		if state.phase != "" {
			status = state.phase
		}
	}
	detail := "Checked " + m.clock().Format(time.RFC3339) + "; native user-session state; no client mutation authority"
	if data, exists, err := privateRead(filepath.Join(m.root(), "observation.json")); err == nil && exists {
		var record observation
		if decodeRecord(data, &record) == nil && validObservation(record) {
			age := m.clock().Sub(record.ObservedAt)
			freshness := "fresh"
			if age < 0 || age > 90*time.Second {
				freshness = "stale"
			}
			detail += "; observation " + freshness + " at " + record.ObservedAt.UTC().Format(time.RFC3339)
		}
	}
	return core.ServiceInfo{Backend: m.backend(), Status: status, Detail: detail}
}
func (m *Manager) failure(err error) core.ServiceInfo {
	status := "Unavailable"
	if err == errUnsafe {
		status = "Conflict"
	}
	if err == errUncertain {
		status = "Unknown"
	}
	detail := "Observer unavailable; foreground use does not require it"
	if err == errUnsafe {
		detail = "Foreign or unsafe observer artifacts were preserved"
	}
	if err == errUncertain {
		detail = "Lifecycle outcome uncertain; inspect before reconciliation; no automatic retry"
	}
	if m.os == "windows" {
		detail = "Current-user logon Scheduled Task unavailable: observer ACL/reparse, literal-percent paths and native Scheduler ownership/absence proof pending; no system service fallback"
	}
	return core.ServiceInfo{Backend: m.backend(), Status: status, Detail: detail}
}

// Status performs bounded read-only native queries. It never installs, recovers,
// starts or rewrites anything and never includes native command output/paths.
func (m *Manager) Status(ctx context.Context) core.ServiceInfo {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if ctx.Err() != nil || !supportedFiles() || m.os == "windows" {
		return m.failure(errUnavailable)
	}
	spec, err := m.spec()
	if err != nil {
		return m.failure(err)
	}
	state, err := m.inspect(ctx, spec)
	if err != nil {
		return m.failure(err)
	}
	if ctx.Err() != nil {
		return m.failure(errUnavailable)
	}
	if _, exists, err := privateRead(filepath.Join(m.root(), "pending.json")); err != nil {
		return m.failure(err)
	} else if exists {
		return m.failure(errUncertain)
	}
	return m.info(state)
}

// Fixed-schema records are small. Reject duplicate object fields before Go's
// JSON decoder can silently pick the last ownership or pending field.
func strictJSON(data []byte) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if !recordValue(d, 0) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}
func recordValue(d *json.Decoder, depth int) bool {
	if depth > 8 {
		return false
	}
	token, err := d.Token()
	if err != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return true
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return false
			}
			key, ok := token.(string)
			if !ok || seen[key] {
				return false
			}
			seen[key] = true
			if !recordValue(d, depth+1) {
				return false
			}
		}
		token, err = d.Token()
		return err == nil && token == json.Delim('}')
	case '[':
		for d.More() {
			if !recordValue(d, depth+1) {
				return false
			}
		}
		token, err = d.Token()
		return err == nil && token == json.Delim(']')
	default:
		return false
	}
}
