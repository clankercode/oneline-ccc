package configtxn

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const maxSnapshot = 5 * 1024 * 1024

var (
	ErrUnsafe      = errors.New("unsafe transaction path or permissions")
	ErrLimit       = errors.New("transaction snapshot or operation exceeds size limit")
	ErrIO          = errors.New("protected transaction filesystem operation failed")
	ErrJournal     = errors.New("protected transaction journal requires reconciliation")
	ErrInterrupted = errors.New("transaction interrupted; run recovery")
	ErrUnsupported = errors.New("protected transactions unavailable: native owner-only ACL and reparse protection not implemented on this platform")
)

func canonical(path string) (string, error) {
	if path == "" {
		return "", ErrUnsafe
	}
	p, err := filepath.Abs(path)
	if err != nil {
		return "", ErrUnsafe
	}
	return p, nil
}

// Check every component rather than accepting symlinks which happen to resolve
// inside the approved tree. Missing destination is allowed, missing parents are not.
func checkPath(path string, absent bool) error {
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil {
			if !(absent && p == path && errors.Is(err, os.ErrNotExist)) {
				return ErrUnsafe
			}
		} else if info.Mode()&os.ModeSymlink != 0 || (p != path && !info.IsDir()) {
			return ErrUnsafe
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}

// Read captures a bounded, regular-file snapshot and independently owned Data.
func Read(path string) (Write, error) {
	if !platformSupported() {
		return Write{}, ErrUnsupported
	}
	p, err := canonical(path)
	if err != nil {
		return Write{}, err
	}
	w := Write{Path: p, Mode: 0600}
	if err = checkPath(p, true); err != nil {
		return Write{}, err
	}
	info, err := os.Lstat(p)
	if errors.Is(err, os.ErrNotExist) {
		return w, nil
	}
	if err != nil {
		return Write{}, ErrIO
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeType != 0 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return Write{}, ErrUnsafe
	}
	f, err := os.OpenFile(p, os.O_RDONLY|safeOpenFlags(), 0)
	if err != nil {
		return Write{}, ErrIO
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return Write{}, ErrUnsafe
	}
	if opened.Size() > maxSnapshot {
		return Write{}, ErrLimit
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSnapshot+1))
	if err != nil {
		return Write{}, ErrIO
	}
	if len(b) > maxSnapshot {
		return Write{}, ErrLimit
	}
	end, err := f.Stat()
	if err != nil || end.Size() != opened.Size() || !end.ModTime().Equal(opened.ModTime()) || end.Mode() != opened.Mode() {
		return Write{}, ErrConflict
	}
	w.Existed = true
	w.Mode = opened.Mode().Perm()
	w.Before = b
	w.Data = bytes.Clone(b)
	return w, nil
}

func expected(w Write) error {
	now, err := Read(w.Path)
	if err != nil {
		return err
	}
	if now.Existed != w.Existed || !bytes.Equal(now.Before, w.Before) || (w.Existed && now.Mode.Perm() != w.Mode.Perm()) {
		return ErrConflict
	}
	return nil
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return ErrIO
	}
	defer f.Close()
	if f.Sync() != nil {
		return ErrIO
	}
	return nil
}

func writeExclusive(path string, b []byte, mode os.FileMode) error {
	if err := checkPath(path, true); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|safeOpenFlags(), 0600)
	if err != nil {
		return ErrIO
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Chmod(mode)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return ErrIO
	}
	return syncDir(filepath.Dir(path))
}

func ensurePrivateDir(path string) error {
	if err := checkPath(path, true); err != nil {
		// Create only missing ancestors, never chmod existing operator directories.
		if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
			return err
		}
		if e := ensurePrivateDir(filepath.Dir(path)); e != nil {
			return e
		}
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(path, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return ErrIO
		}
		if err = syncDir(filepath.Dir(path)); err != nil {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !owned(info) {
		return ErrUnsafe
	}
	return checkPath(path, false)
}

func privateFile(path string) error {
	if err := checkPath(path, false); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || !owned(info) {
		return ErrUnsafe
	}
	return nil
}

func readPrivate(path string, limit int64) ([]byte, error) {
	if err := privateFile(path); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|safeOpenFlags(), 0)
	if err != nil {
		return nil, ErrIO
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, ErrIO
	}
	if int64(len(b)) > limit {
		return nil, ErrLimit
	}
	return b, nil
}
