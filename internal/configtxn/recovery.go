package configtxn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Recover retains completed receipts and rolls incomplete operations back under
// the same coordinator and stable parent locks used by Apply.
func (s *Store) Recover(ctx context.Context) ([]Result, error) {
	l, err := s.coordinator(ctx)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	return s.recoverLocked(ctx)
}

func (s *Store) recoverLocked(ctx context.Context) ([]Result, error) {
	dirs, err := os.ReadDir(s.Root)
	if err != nil {
		return nil, ErrIO
	}
	var results []Result
	var combined error
	for _, d := range dirs {
		if !strings.HasPrefix(d.Name(), "txn-") {
			continue
		}
		if err = ctx.Err(); err != nil {
			return results, err
		}
		j, e := s.load(strings.TrimPrefix(d.Name(), "txn-"))
		if e != nil {
			return results, e
		}
		if j.Outcome == "Succeeded" || j.Outcome == "RolledBack" {
			results = append(results, s.receipt(j))
			continue
		}
		if j.Outcome == "Conflict" {
			results = append(results, s.receipt(j))
			combined = ErrConflict
			continue
		}
		paths := make([]string, len(j.Entries))
		for i, e := range j.Entries {
			paths[i] = e.Path
		}
		locks, e := lockParents(ctx, paths)
		if e != nil {
			if errors.Is(e, ErrUnsafe) {
				j.Outcome = "Conflict"
				if err = s.save(j); err != nil {
					return results, err
				}
				results = append(results, s.receipt(j))
				combined = ErrConflict
				continue
			}
			return results, e
		}
		r, e := s.rollback(ctx, j)
		locks.close()
		results = append(results, r)
		if e != nil {
			if !errors.Is(e, ErrConflict) {
				return results, e
			}
			combined = ErrConflict
		}
	}
	return results, combined
}

func original(e entry, w Write) bool {
	return e.Existed == w.Existed && (!e.Existed || (digest(w.Before) == e.BeforeHash && uint32(w.Mode.Perm()) == e.Mode))
}
func output(e entry, w Write) bool {
	return w.Existed && digest(w.Before) == e.AfterHash && uint32(w.Mode.Perm()) == e.Mode
}

func (s *Store) rollback(ctx context.Context, j *journal) (Result, error) {
	j.Outcome = "RollingBack"
	if err := s.save(j); err != nil {
		return s.unknown(j), err
	}
	if err := s.hit("rollback-intent"); err != nil {
		return s.unknown(j), err
	}
	conflict := false
	for i := len(j.Entries) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return s.unknown(j), err
		}
		e := &j.Entries[i]
		if e.State == "conflict" {
			conflict = true
			continue
		}
		if e.State == "replacing" || e.State == "applied" || e.State == "restoring" {
			err := s.restore(j, i)
			if err != nil {
				if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrUnsafe) {
					return s.unknown(j), err
				}
				e.State = "conflict"
				conflict = true
			} else {
				e.State = "restored"
			}
			if err = s.save(j); err != nil {
				return s.unknown(j), err
			}
			if err = s.hitEntry("restored", i); err != nil {
				return s.unknown(j), err
			}
		}
		if e.State != "conflict" {
			for _, rb := range []bool{false, true} {
				hash := e.AfterHash
				if rb {
					hash = e.BeforeHash
				}
				if err := cleanupStage(stage(j, i, rb), hash); err != nil {
					if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrUnsafe) {
						return s.unknown(j), err
					}
					e.State = "conflict"
					conflict = true
				}
			}
		}
	}
	j.Outcome = "RolledBack"
	if conflict {
		j.Outcome = "Conflict"
	}
	if err := s.save(j); err != nil {
		return s.unknown(j), err
	}
	if err := s.hit("rolled-back"); err != nil {
		return s.unknown(j), err
	}
	if conflict {
		return s.receipt(j), ErrConflict
	}
	return s.receipt(j), nil
}

func (s *Store) restore(j *journal, i int) error {
	e := &j.Entries[i]
	now, err := Read(e.Path)
	if err != nil {
		return ErrConflict
	}
	if original(*e, now) {
		return nil
	}
	if !output(*e, now) {
		return ErrConflict
	}
	e.State = "restoring"
	if err = s.save(j); err != nil {
		return err
	}
	if e.Existed {
		b, err := readPrivate(s.backup(j.ID, i), maxSnapshot)
		if err != nil {
			return ErrJournal
		}
		if digest(b) != e.BeforeHash {
			return ErrJournal
		}
		tmp := stage(j, i, true)
		if _, err = os.Lstat(tmp); errors.Is(err, os.ErrNotExist) {
			if err = writeExclusive(tmp, b, 0600); err != nil {
				return err
			}
		} else if err != nil {
			return ErrIO
		} else {
			old, err := Read(tmp)
			if err != nil || !old.Existed || digest(old.Before) != e.BeforeHash {
				return ErrConflict
			}
		}
		if err = s.hitEntry("rollback-staged", i); err != nil {
			return err
		}
		if err = chmodSync(stage(j, i, true), os.FileMode(e.Mode)); err != nil {
			return err
		}
		if err = s.hitEntry("rollback-permissions", i); err != nil {
			return err
		}
	}
	if err = s.hitEntry("before-rollback", i); err != nil {
		return err
	}
	if e.Existed {
		if err = verifyReplacement(stage(j, i, true), e.BeforeHash, os.FileMode(e.Mode)); err != nil {
			return err
		}
	}
	// Recheck after every staged restore and hook, immediately before replacement.
	now, err = Read(e.Path)
	if err != nil {
		return ErrConflict
	}
	if original(*e, now) {
		return nil
	}
	if !output(*e, now) {
		return ErrConflict
	}
	if e.Existed {
		if err = replace(stage(j, i, true), e.Path); err != nil {
			return err
		}
	} else {
		if err = os.Remove(e.Path); err != nil {
			return ErrIO
		}
		if err = syncDir(filepath.Dir(e.Path)); err != nil {
			return err
		}
	}
	return s.hitEntry("rollback-replaced", i)
}

// Remove only a known, app-owned temporary with its recorded content. Unexpected
// files are retained for reconciliation, even at private-looking stage names.
func cleanupStage(path, hash string) error {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	w, err := Read(path)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return ErrIO
	}
	if !w.Existed || !owned(info) || digest(w.Before) != hash {
		return ErrConflict
	}
	if os.Remove(path) != nil {
		return ErrIO
	}
	return syncDir(filepath.Dir(path))
}

func chmodSync(path string, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_RDONLY|safeOpenFlags(), 0)
	if err != nil {
		return ErrIO
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || !owned(info) {
		return ErrUnsafe
	}
	if f.Chmod(mode) != nil || f.Sync() != nil {
		return ErrIO
	}
	return nil
}

func verifyReplacement(path, hash string, mode os.FileMode) error {
	w, err := Read(path)
	if err != nil {
		return ErrConflict
	}
	info, err := os.Lstat(path)
	if err != nil || !w.Existed || !owned(info) || w.Mode.Perm() != mode || digest(w.Before) != hash {
		return ErrConflict
	}
	return nil
}
