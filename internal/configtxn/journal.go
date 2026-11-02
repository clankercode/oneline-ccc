package configtxn

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxWrites = 128

type entry struct {
	Path                  string
	Existed               bool
	Mode                  uint32
	BeforeHash, AfterHash string
	State                 string
}
type journal struct {
	Version     int
	ID, Outcome string
	Entries     []entry
}

func (s *Store) operation(id string) string { return filepath.Join(s.Root, "txn-"+id) }
func (s *Store) backup(id string, i int) string {
	return filepath.Join(s.operation(id), fmt.Sprintf("%d.before", i))
}
func stage(j *journal, i int, rollback bool) string {
	suffix := "stage"
	if rollback {
		suffix = "rollback"
	}
	return filepath.Join(filepath.Dir(j.Entries[i].Path), fmt.Sprintf(".ccc-%s-%d.%s", j.ID, i, suffix))
}

func (s *Store) save(j *journal) error {
	b, err := json.Marshal(j)
	if err != nil {
		return ErrJournal
	}
	d := s.operation(j.ID)
	if err = ensurePrivateDir(d); err != nil {
		return err
	}
	f, err := os.CreateTemp(d, "journal-")
	if err != nil {
		return ErrIO
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return ErrIO
	}
	if err = checkPath(filepath.Join(d, "journal.json"), true); err != nil {
		return err
	}
	if os.Rename(tmp, filepath.Join(d, "journal.json")) != nil {
		return ErrIO
	}
	return syncDir(d)
}

func (s *Store) begin(writes []Write) (*journal, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, ErrIO
	}
	j := &journal{Version: 1, ID: hex.EncodeToString(b), Outcome: "Applying"}
	for _, w := range writes {
		j.Entries = append(j.Entries, entry{Path: w.Path, Existed: w.Existed, Mode: uint32(w.Mode.Perm()), BeforeHash: digest(w.Before), AfterHash: digest(w.Data), State: "pending"})
	}
	// Never reuse an existing operation directory, including on ID collision.
	if err := os.Mkdir(s.operation(j.ID), 0700); err != nil {
		return nil, ErrIO
	}
	if err := syncDir(s.Root); err != nil {
		return nil, err
	}
	if err := s.save(j); err != nil {
		return nil, err
	}
	return j, nil
}

func validHex(s string, n int) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == n && s == strings.ToLower(s)
}

func (s *Store) load(id string) (*journal, error) {
	if !validHex(id, 16) {
		return nil, ErrJournal
	}
	if err := ensurePrivateDir(s.operation(id)); err != nil {
		return nil, err
	}
	b, err := readPrivate(filepath.Join(s.operation(id), "journal.json"), 1024*1024)
	if err != nil {
		return nil, ErrJournal
	}
	var j journal
	if json.Unmarshal(b, &j) != nil || j.Version != 1 || j.ID != id || len(j.Entries) == 0 || len(j.Entries) > maxWrites {
		return nil, ErrJournal
	}
	switch j.Outcome {
	case "Applying", "RollingBack", "Succeeded", "RolledBack", "Conflict":
	default:
		return nil, ErrJournal
	}
	seen := map[string]bool{}
	for _, e := range j.Entries {
		p, err := canonical(e.Path)
		if err != nil || p != e.Path || seen[p] || reserved(s.Root, p) || !validHex(e.BeforeHash, 32) || !validHex(e.AfterHash, 32) || e.Mode > 0777 {
			return nil, ErrJournal
		}
		seen[p] = true
		switch e.State {
		case "pending", "ready", "replacing", "applied", "restoring", "restored", "conflict":
		default:
			return nil, ErrJournal
		}
	}
	return &j, nil
}

func (s *Store) receipt(j *journal) Result {
	r := Result{ID: j.ID, Outcome: j.Outcome}
	for i, e := range j.Entries {
		if e.Existed {
			r.Backups = append(r.Backups, s.backup(j.ID, i))
		}
		if e.State == "applied" || e.State == "replacing" || e.State == "restoring" || e.State == "restored" || e.State == "conflict" {
			r.Changed = append(r.Changed, e.Path)
		}
	}
	return r
}

func reserved(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) || filepath.Base(path) == ".ccc-configtxn.lock" || strings.HasPrefix(filepath.Base(path), ".ccc-")
}

// Checkpoint errors model process interruption: leave durable intent intact and
// never include hook errors (which could contain config bytes) in a receipt.
func (s *Store) hit(name string) error {
	if s.checkpoint != nil && s.checkpoint(name) != nil {
		return ErrInterrupted
	}
	return nil
}
func (s *Store) hitEntry(name string, i int) error { return s.hit(fmt.Sprintf("%s:%d", name, i)) }
