package configtxn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

var ErrConflict = errors.New("configuration transaction conflict; preserve foreign changes and reconcile")

func New(root string) *Store {
	p, err := canonical(root)
	if err == nil {
		root = p
	}
	return &Store{Root: root}
}

// Apply serializes cooperating invocations, recovers older incomplete operations,
// and replaces each changed file atomically. The group is recoverable, not atomic.
func (s *Store) Apply(ctx context.Context, writes []Write) (Result, error) {
	if err := ctx.Err(); err != nil {
		return resultError(err), err
	}
	if err := metadataPreflight(writes); err != nil {
		return resultError(err), err
	}
	l, err := s.coordinator(ctx)
	if err != nil {
		return Result{}, err
	}
	defer l.Close()
	if _, err = s.recoverLocked(ctx); err != nil {
		return resultError(err), err
	}
	conditions, changed, err := s.validate(writes)
	if err != nil {
		return resultError(err), err
	}
	paths := make([]string, len(conditions))
	for i, w := range conditions {
		paths[i] = w.Path
	}
	locks, err := lockParents(ctx, paths)
	if err != nil {
		return Result{}, err
	}
	defer locks.close()
	// The first-writer snapshot may have gone stale while waiting for parent locks.
	if err = expectedAll(conditions); err != nil {
		return resultError(err), err
	}
	if err = ctx.Err(); err != nil {
		return Result{Outcome: "Cancelled"}, err
	}
	if len(changed) == 0 {
		return Result{Outcome: "Noop"}, nil
	}
	j, err := s.begin(changed)
	if err != nil {
		return Result{}, err
	}
	if err = s.hit("journal-created"); err != nil {
		return s.unknown(j), err
	}
	for i, w := range changed {
		if w.Existed {
			if err = writeExclusive(s.backup(j.ID, i), w.Before, 0600); err != nil {
				return s.failed(j, err)
			}
			if err = s.hitEntry("backup", i); err != nil {
				return s.unknown(j), err
			}
		}
		if err = writeExclusive(stage(j, i, false), w.Data, 0600); err != nil {
			return s.failed(j, err)
		}
		if err = s.hitEntry("stage", i); err != nil {
			return s.unknown(j), err
		}
		j.Entries[i].State = "ready"
		if err = s.save(j); err != nil {
			return s.failed(j, err)
		}
		if err = s.hitEntry("ready", i); err != nil {
			return s.unknown(j), err
		}
	}
	// Validate all expected originals before the first replacement as well as each
	// immediately before its rename. A foreign editor can still race the rename.
	if err = expectedAll(conditions); err != nil {
		return s.failed(j, err)
	}
	for i, w := range changed {
		if err = ctx.Err(); err != nil {
			return s.failed(j, err)
		}
		j.Entries[i].State = "replacing"
		if err = s.save(j); err != nil {
			return s.failed(j, err)
		}
		if err = s.hitEntry("before-replace", i); err != nil {
			return s.unknown(j), err
		}
		if err = ctx.Err(); err != nil {
			return s.failed(j, err)
		}
		if err = verifyStage(j, i); err != nil {
			return s.failed(j, err)
		}
		if err = chmodSync(stage(j, i, false), w.Mode.Perm()); err != nil {
			return s.failed(j, ErrIO)
		}
		if err = s.hitEntry("stage-permissions", i); err != nil {
			return s.unknown(j), err
		}
		if err = ctx.Err(); err != nil {
			return s.failed(j, err)
		}
		if err = verifyReplacement(stage(j, i, false), j.Entries[i].AfterHash, w.Mode.Perm()); err != nil {
			return s.failed(j, err)
		}
		if err = expectedAll(conditions); err != nil {
			return s.failed(j, err)
		}
		if err = replace(ctx, stage(j, i, false), w.Path, j.Entries[i].AfterHash, w.Mode, renameChecks{before: func() error { return expectedAll(conditions) }, retry: func() error { return s.hitEntry("sharing-retry", i) }}); err != nil {
			if errors.Is(err, ErrInterrupted) {
				return s.unknown(j), err
			}
			return s.failed(j, err)
		}
		if err = s.hitEntry("replaced", i); err != nil {
			return s.unknown(j), err
		}
		advanceCondition(conditions, w)
		j.Entries[i].State = "applied"
		if err = s.save(j); err != nil {
			return s.failed(j, err)
		}
		if err = s.hitEntry("applied", i); err != nil {
			return s.unknown(j), err
		}
	}
	if err = expectedAll(conditions); err != nil {
		return s.failed(j, err)
	}
	j.Outcome = "Succeeded"
	if err = s.save(j); err != nil {
		return s.unknown(j), err
	}
	if err = s.hit("succeeded"); err != nil {
		return s.unknown(j), err
	}
	return s.receipt(j), nil
}

func resultError(err error) Result {
	outcome := "NeedsAttention"
	if errors.Is(err, ErrConflict) {
		outcome = "Conflict"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		outcome = "Cancelled"
	}
	return Result{Outcome: outcome}
}
func (s *Store) unknown(j *journal) Result { r := s.receipt(j); r.Outcome = "Unknown"; return r }
func (s *Store) failed(j *journal, cause error) (Result, error) {
	r, err := s.rollback(context.Background(), j)
	if err != nil {
		return r, err
	}
	if errors.Is(cause, ErrConflict) {
		r.Outcome = "Conflict"
	}
	return r, cause
}

func verifyStage(j *journal, i int) error {
	b, err := readPrivate(stage(j, i, false), maxSnapshot)
	if err != nil {
		return err
	}
	if digest(b) != j.Entries[i].AfterHash {
		return ErrConflict
	}
	return nil
}
func replace(ctx context.Context, src, dst, hash string, mode os.FileMode, checks renameChecks) error {
	if err := checkPath(src, false); err != nil {
		return err
	}
	if err := checkPath(dst, true); err != nil {
		return err
	}
	if err := renameFile(ctx, src, dst, hash, mode, checks); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, ErrConflict) || errors.Is(err, ErrInterrupted) {
			return err
		}
		return ErrIO
	}
	return syncDir(filepath.Dir(dst))
}
