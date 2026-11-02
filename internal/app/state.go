package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"regexp"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
)

const stateLimit = 64 << 10
const receiptLimit = 32

type state struct {
	Version  int             `json:"version"`
	Managed  map[string]bool `json:"managed"`
	Receipts []core.Receipt  `json:"receipts"`
}

var receiptID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func (e *Engine) readState() (state, configtxn.Write, error) {
	bad := errors.New("application settings are invalid or unsafe; preserve them and reconcile before continuing")
	if configtxn.PrivateDir(e.options.StateRoot) != nil {
		return state{}, configtxn.Write{}, bad
	}
	w, err := configtxn.Read(filepath.Join(e.options.StateRoot, "settings.json"))
	if err != nil {
		return state{}, configtxn.Write{}, bad
	}
	s := state{Version: 1, Managed: map[string]bool{}, Receipts: []core.Receipt{}}
	if !w.Existed {
		return s, w, nil
	}
	if w.Mode.Perm() != 0600 || len(w.Before) > stateLimit {
		return state{}, configtxn.Write{}, bad
	}
	fields, err := object(w.Before, "version", "managed", "receipts")
	if err != nil || len(fields) != 3 || json.Unmarshal(fields["version"], &s.Version) != nil || s.Version != 1 {
		return state{}, configtxn.Write{}, bad
	}
	flags, err := object(fields["managed"], "codex", "claude", "pi", "opencode", "grokbuild", "hermes")
	if err != nil {
		return state{}, configtxn.Write{}, bad
	}
	for id, raw := range flags {
		var value bool
		if bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &value) != nil {
			return state{}, configtxn.Write{}, bad
		}
		s.Managed[id] = value
	}
	var receipts []json.RawMessage
	if bytes.Equal(fields["receipts"], []byte("null")) || json.Unmarshal(fields["receipts"], &receipts) != nil || len(receipts) > receiptLimit {
		return state{}, configtxn.Write{}, bad
	}
	for _, raw := range receipts {
		f, err := object(raw, "id", "client_id", "outcome", "summary")
		var r core.Receipt
		if err != nil || len(f) != 4 || json.Unmarshal(raw, &r) != nil || !receiptID.MatchString(r.ID) || clientName(r.ClientID) == "" || r.Outcome != "Succeeded" || !validSummary(r.Summary) {
			return state{}, configtxn.Write{}, bad
		}
		s.Receipts = append(s.Receipts, r)
	}
	return s, w, nil
}
func validSummary(s string) bool {
	return s == "Client configuration applied." || s == "Client management disabled." || s == "Client management enabled."
}

// object rejects unknown/case-aliased/duplicate names instead of JSON's default
// last-key-wins behavior. Callers separately validate each nested schema.
func object(data []byte, allowed ...string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("invalid object")
	}
	keys := map[string]bool{}
	for _, key := range allowed {
		keys[key] = true
	}
	out := map[string]json.RawMessage{}
	for d.More() {
		token, err = d.Token()
		key, ok := token.(string)
		if err != nil || !ok || !keys[key] || out[key] != nil {
			return nil, errors.New("invalid field")
		}
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return nil, errors.New("invalid value")
		}
		out[key] = raw
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') {
		return nil, errors.New("invalid object")
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, errors.New("trailing settings")
	}
	return out, nil
}
