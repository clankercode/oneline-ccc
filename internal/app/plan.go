package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/clankercode/oneline-ccc/internal/codexrpc"
	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

type review struct {
	Plan     core.Plan
	Epoch    uint64
	Program  codexrpc.Program
	Identity string
	Writes   []configtxn.Write
	State    configtxn.Write
	Receipt  core.Receipt
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("cannot allocate a review identity")
	}
	return hex.EncodeToString(b), nil
}
func copyProgram(p codexrpc.Program) codexrpc.Program {
	p.Args = append([]string{}, p.Args...)
	return p
}
func cloneWrite(w configtxn.Write) configtxn.Write {
	w.Before = bytes.Clone(w.Before)
	w.Data = bytes.Clone(w.Data)
	return w
}
func changed(w configtxn.Write) bool { return !w.Existed || !bytes.Equal(w.Before, w.Data) }
func canonicalTier(tier string) string {
	switch tier {
	case "", "default", "Standard":
		return "Standard"
	case "Fast", "fast", "priority":
		return "Fast"
	case "Ultrafast", "ultrafast":
		return "Ultrafast"
	}
	return tier
}

func (e *Engine) Plan(ctx context.Context, id string, desired core.Desired) (core.Plan, error) {
	if ctx.Err() != nil {
		return core.Plan{}, canceled()
	}
	if clientName(id) == "" {
		return core.Plan{}, errors.New("unknown client selection")
	}
	e.mu.Lock()
	e.epoch[id]++
	epoch := e.epoch[id]
	delete(e.pending, id)
	e.mu.Unlock()
	s, stateWrite, err := e.readState()
	if err != nil {
		return core.Plan{}, err
	}
	r := &review{Epoch: epoch, State: cloneWrite(stateWrite)}
	original := desired
	var before core.Profile
	if desired.Enabled {
		if id != "codex" {
			return core.Plan{}, errors.New("this client has detection only; a verified native adapter is required")
		}
		if err = core.ValidateProfile(desired.Profile); err != nil {
			return core.Plan{}, err
		}
		if e.deps.childEnv(desired.Profile.EnvKey, "fixture-reference-validation") == nil {
			return core.Plan{}, errors.New("choose a safe credential environment reference")
		}
		desired.Profile.Tier = canonicalTier(desired.Profile.Tier)
		c, inspectErr := e.inspectCodex(ctx)
		if inspectErr != nil {
			return core.Plan{}, inspectErr
		}
		if !c.Supported {
			return core.Plan{}, errors.New("configuration changes require verified native Codex 0.166.0")
		}
		before = c.Profile
		r.Program = copyProgram(e.deps.program())
		r.Identity, err = e.deps.identity(r.Program)
		if err != nil {
			return core.Plan{}, errors.New("cannot verify the selected native executable")
		}
		if ctx.Err() != nil {
			return core.Plan{}, canceled()
		}
		var catalog []byte
		if desired.Profile.TiersConfirmed {
			if desired.Profile.Model != "gpt-6.1-sol" && desired.Profile.Model != "gpt-6-astra" {
				return core.Plan{}, errors.New("confirmed tier catalogs require an exact supported model ID; use Standard without tier confirmation for other models")
			}
			// This confirmation attests Fast and Ultrafast availability for the exact
			// route/model; the selected default remains a separate Profile field.
			catalog, err = e.deps.confirmed(ctx, c.Version, desired.Profile.Model, []string{"priority", "ultrafast"})
			if ctx.Err() != nil {
				return core.Plan{}, canceled()
			}
			if err != nil {
				return core.Plan{}, errors.New("complete version-matched tier catalog unavailable; retain the current configuration")
			}
		}
		writes, prepareErr := e.deps.adapter.Prepare(ctx, desired.Profile, bytes.Clone(catalog))
		if ctx.Err() != nil {
			return core.Plan{}, canceled()
		}
		if prepareErr != nil {
			return core.Plan{}, errors.New("native configuration staging failed; preserve existing files and inspect the supported profile")
		}
		if len(writes) == 0 || len(writes) > 2 {
			return core.Plan{}, errors.New("native adapter returned an invalid change set")
		}
		seen := map[string]bool{}
		for _, w := range writes {
			path, absErr := filepath.Abs(w.Path)
			if absErr != nil || seen[path] || (path != filepath.Join(e.options.CodexHome, "config.toml") && path != filepath.Join(e.options.CodexHome, "llmp.config.toml")) {
				return core.Plan{}, errors.New("native adapter returned an unexpected target")
			}
			seen[path] = true
			w.Path = path
			r.Writes = append(r.Writes, cloneWrite(w))
		}
		identity, identityErr := e.deps.identity(r.Program)
		if identityErr != nil || identity != r.Identity || !reflect.DeepEqual(copyProgram(e.deps.program()), r.Program) {
			return core.Plan{}, errors.New("native executable changed while planning; inspect again")
		}
	}
	if ctx.Err() != nil {
		return core.Plan{}, canceled()
	}
	if _, _, err = e.readState(); err != nil {
		return core.Plan{}, err
	}
	if !e.expected(r.State) {
		return core.Plan{}, errors.New("application state changed while planning; create a new review")
	}
	idToken, err := randomID()
	if err != nil {
		return core.Plan{}, err
	}
	p := core.Plan{ID: idToken, ClientID: id, Desired: desired, Changes: changes(before, desired, s.Managed[id]), Noop: s.Managed[id] == desired.Enabled}
	for _, w := range r.Writes {
		if changed(w) {
			p.Noop = false
		}
	}
	r.Receipt = core.Receipt{ID: p.ID, ClientID: id, Outcome: "Succeeded", Summary: "Client configuration applied."}
	if !desired.Enabled {
		r.Receipt.Summary = "Client management disabled."
	} else if len(r.Writes) == 0 {
		r.Receipt.Summary = "Client management enabled."
	}
	if !p.Noop {
		s.Managed[id] = desired.Enabled
		s.Receipts = append(s.Receipts, r.Receipt)
		if len(s.Receipts) > receiptLimit {
			s.Receipts = s.Receipts[len(s.Receipts)-receiptLimit:]
		}
		r.State.Data, err = json.Marshal(s)
		if err != nil || len(r.State.Data) > stateLimit {
			return core.Plan{}, errors.New("protected receipt state exceeds its limit")
		}
		r.State.Data = append(r.State.Data, '\n')
	}
	// The digest covers all private inputs, snapshots, staged bytes, native
	// executable identity and original attestation. Only the digest is public.
	input := struct {
		ID       string
		Desired  core.Desired
		Original core.Desired
		Options  [3]string
		Program  codexrpc.Program
		Identity string
		Writes   []configtxn.Write
		State    configtxn.Write
	}{p.ID, desired, original, [3]string{e.options.StateRoot, e.options.CodexHome, e.options.Executable}, r.Program, r.Identity, r.Writes, r.State}
	data, err := json.Marshal(input)
	if err != nil {
		return core.Plan{}, errors.New("cannot freeze the reviewed change")
	}
	hash := sha256.Sum256(data)
	p.Revision = hex.EncodeToString(hash[:])
	r.Plan = p
	r.Plan.Changes = append([]core.Change{}, p.Changes...)
	e.mu.Lock()
	defer e.mu.Unlock()
	if ctx.Err() != nil {
		return core.Plan{}, canceled()
	}
	if e.epoch[id] != epoch {
		return core.Plan{}, errors.New("a newer review superseded this plan")
	}
	e.pending[id] = r
	p.Changes = append([]core.Change{}, p.Changes...)
	return p, nil
}
func changes(before core.Profile, d core.Desired, managed bool) []core.Change {
	out := []core.Change{}
	add := func(field, left, right string) {
		if left != right {
			out = append(out, core.Change{Field: field, Before: left, After: right})
		}
	}
	add("Management", strconv.FormatBool(managed), strconv.FormatBool(d.Enabled))
	if !d.Enabled {
		return out
	}
	p := d.Profile
	if before.BaseURL != p.BaseURL {
		add("Endpoint", "Current setting", "Changed (value withheld)")
	}
	safeModel := func(v string) string {
		if len(v) > 256 || stringsUnsafe(v) {
			return "Exact value withheld"
		}
		return v
	}
	add("Model", safeModel(before.Model), safeModel(p.Model))
	add("Context budget", strconv.FormatInt(before.Budget, 10), strconv.FormatInt(p.Budget, 10))
	add("Compaction", strconv.FormatInt(before.Compact, 10), strconv.FormatInt(p.Compact, 10))
	add("Tier", canonicalTier(before.Tier), p.Tier)
	if before.EnvKey != p.EnvKey {
		add("Credential reference", "Current reference", "Changed (value withheld)")
	}
	return out
}
func stringsUnsafe(s string) bool {
	for _, r := range s {
		if r < 32 || r == 127 {
			return true
		}
	}
	return bytes.Contains([]byte(s), []byte("://")) || bytes.ContainsAny([]byte(s), "?#@")
}

func (e *Engine) Apply(ctx context.Context, id, revision string) (core.Receipt, error) {
	e.mu.Lock()
	var r *review
	for client, p := range e.pending {
		if p.Plan.ID == id {
			r = p
			delete(e.pending, client)
			break
		}
	}
	e.mu.Unlock()
	if r == nil || revision != r.Plan.Revision {
		return core.Receipt{}, errors.New("review is invalid, stale or already used; create a new plan")
	}
	if err := e.lockMutation(ctx); err != nil {
		return core.Receipt{}, err
	}
	defer func() { <-e.mutating }()
	if ctx.Err() != nil {
		return core.Receipt{}, canceled()
	}
	if r.Plan.Desired.Enabled {
		identity, err := e.deps.identity(r.Program)
		if ctx.Err() != nil {
			return core.Receipt{}, canceled()
		}
		if err != nil || identity != r.Identity || !reflect.DeepEqual(copyProgram(e.deps.program()), r.Program) {
			return core.Receipt{}, errors.New("native executable changed after review; create a new plan")
		}
	}
	if _, _, err := e.readState(); err != nil {
		return core.Receipt{}, err
	}
	for _, w := range append(append([]configtxn.Write{}, r.Writes...), r.State) {
		if !e.expected(w) {
			return core.Receipt{}, errors.New("reviewed files changed; preserve them and create a new plan")
		}
	}
	e.mu.Lock()
	current := e.epoch[r.Plan.ClientID] == r.Epoch
	e.mu.Unlock()
	if !current {
		return core.Receipt{}, errors.New("a newer review superseded this plan")
	}
	if ctx.Err() != nil {
		return core.Receipt{}, canceled()
	}
	if r.Plan.Noop {
		out := r.Receipt
		out.Outcome = "Noop"
		out.Summary = "No configuration or management changes."
		return out, nil
	}
	if r.Plan.Desired.Enabled {
		if err := configtxn.PrivateDir(e.options.CodexHome); err != nil {
			return core.Receipt{}, errors.New("Codex configuration root must be private and safely owned")
		}
	}
	writes := append(append([]configtxn.Write{}, r.Writes...), r.State)
	result, err := e.store.Apply(ctx, writes)
	if err != nil {
		return core.Receipt{ID: r.Plan.ID, ClientID: r.Plan.ClientID, Outcome: safeOutcome(result.Outcome), Summary: "Configuration was not confirmed; inspect recovery before retrying."}, errors.New("configuration transaction did not succeed; preserve files and inspect recovery")
	}
	if result.Outcome != "Succeeded" {
		return core.Receipt{}, errors.New("transaction returned an unexpected outcome; inspect before retrying")
	}
	return r.Receipt, nil
}
func (e *Engine) expected(w configtxn.Write) bool {
	now, err := configtxn.Read(w.Path)
	if err != nil && !w.Existed && within(e.options.CodexHome, w.Path) {
		// A first-run home may not yet exist. Validate the first missing ancestor
		// without creating it; PrivateDir is reserved for the explicit Apply above.
		parent := filepath.Dir(w.Path)
		for {
			_, statErr := os.Lstat(parent)
			if statErr == nil {
				break
			}
			if !os.IsNotExist(statErr) {
				return false
			}
			next := filepath.Dir(parent)
			if next == parent {
				return false
			}
			if _, readErr := configtxn.Read(parent); readErr == nil {
				return len(w.Before) == 0
			}
			parent = next
		}
	}
	return err == nil && now.Existed == w.Existed && bytes.Equal(now.Before, w.Before) && (!w.Existed || now.Mode.Perm() == w.Mode.Perm())
}
func safeOutcome(outcome string) string {
	switch outcome {
	case "Noop", "Succeeded", "RolledBack", "Conflict", "Cancelled", "NeedsAttention", "Unknown":
		return outcome
	}
	return "NeedsAttention"
}
