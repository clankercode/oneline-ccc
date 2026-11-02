package monitor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/gofrs/flock"
)

const recordLimit = 16 * 1024

// safeComponents permits absent tails and refuses symlinks/non-directory
// ancestors. It does not create or chmod any client or operator directory.
func safeComponents(path string) error {
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return errUnsafe
		}
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || (p != path && !info.IsDir())) {
			return errUnsafe
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}
func privateRead(path string) ([]byte, bool, error) {
	if !supportedFiles() {
		return nil, false, errUnavailable
	}
	if safeComponents(path) != nil {
		return nil, false, errUnsafe
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || !privateInfo(info) || info.Size() > recordLimit {
		return nil, false, errUnsafe
	}
	snap, err := configtxn.Read(path)
	if err != nil || !snap.Existed || len(snap.Before) > recordLimit {
		return nil, false, errUnsafe
	}
	end, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, end) || !info.ModTime().Equal(end.ModTime()) || info.Mode() != end.Mode() || !privateInfo(end) {
		return nil, false, errUnsafe
	}
	return snap.Before, true, nil
}
func atomicRecord(ctx context.Context, path string, value any) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := configtxn.PrivateDir(filepath.Dir(path)); err != nil {
		return errUnsafe
	}
	before, existed, err := privateRead(path)
	if err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) > recordLimit {
		return errIO
	}
	random := make([]byte, 16)
	if _, err = rand.Read(random); err != nil {
		return errIO
	}
	stage := filepath.Join(filepath.Dir(path), ".observation-"+hex.EncodeToString(random))
	if err = configtxn.PrivateFile(stage, data); err != nil {
		return errIO
	}
	defer os.Remove(stage)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	now, exists, err := privateRead(path)
	if err != nil || exists != existed || !bytes.Equal(before, now) {
		return errUnsafe
	}
	if os.Rename(stage, path) != nil {
		return errIO
	}
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
func (m *Manager) lifecycleLock(ctx context.Context) (*flock.Flock, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if configtxn.PrivateDir(m.stateRoot) != nil || configtxn.PrivateDir(m.root()) != nil {
		return nil, errUnsafe
	}
	path := filepath.Join(m.root(), "lifecycle.lock")
	if _, exists, err := privateRead(path); err != nil {
		return nil, err
	} else if !exists {
		if err := configtxn.PrivateFile(path, nil); err != nil {
			if _, exists, check := privateRead(path); check != nil || !exists {
				return nil, errUnsafe
			}
		}
	}
	l := flock.New(path, flock.SetFlag(os.O_RDWR|lockFlags()), flock.SetPermissions(0600))
	ok, err := l.TryLockContext(ctx, 10*time.Millisecond)
	if err != nil || !ok {
		l.Close()
		return nil, errIO
	}
	if _, _, err = privateRead(path); err != nil {
		l.Close()
		return nil, err
	}
	return l, nil
}
