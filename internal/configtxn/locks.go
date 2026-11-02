package configtxn

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gofrs/flock"
)

type heldLocks []*flock.Flock

func (h heldLocks) close() {
	for i := len(h) - 1; i >= 0; i-- {
		_ = h[i].Close()
	}
}

func lock(ctx context.Context, path string) (*flock.Flock, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := checkPath(path, true); err != nil {
		return nil, err
	}
	if err := prepareLock(path); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(path); err == nil {
		if err = privateFile(path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, ErrIO
	}
	f := flock.New(path, flock.SetFlag(os.O_CREATE|os.O_RDWR|safeOpenFlags()), flock.SetPermissions(0600))
	ok, err := f.TryLockContext(ctx, 10*time.Millisecond)
	if err != nil || !ok {
		_ = f.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrIO
	}
	if err = privateFile(path); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func lockParents(ctx context.Context, paths []string) (heldLocks, error) {
	set := map[string]bool{}
	for _, p := range paths {
		set[pathKey(filepath.Dir(p))] = true
	}
	parents := make([]string, 0, len(set))
	for p := range set {
		parents = append(parents, p)
	}
	sort.Strings(parents)
	var locks heldLocks
	for _, p := range parents {
		if err := checkPath(p, false); err != nil {
			locks.close()
			return nil, err
		}
		if err := targetParent(p); err != nil {
			locks.close()
			return nil, err
		}
		f, err := lock(ctx, filepath.Join(p, ".ccc-configtxn.lock"))
		if err != nil {
			locks.close()
			return nil, err
		}
		locks = append(locks, f)
	}
	return locks, nil
}

func (s *Store) coordinator(ctx context.Context) (*flock.Flock, error) {
	if !platformSupported() {
		return nil, ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := canonical(s.Root)
	if err != nil {
		return nil, err
	}
	if err = ensurePrivateDir(root); err != nil {
		return nil, err
	}
	return lock(ctx, filepath.Join(root, "coordinator.lock"))
}
