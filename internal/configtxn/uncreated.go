package configtxn

import (
	"errors"
	"os"
	"path/filepath"
)

// ReadUncreated inspects a target whose parent directories may not exist yet.
// It never creates them and verifies all existing ancestor components first.
func ReadUncreated(path string) (Write, error) {
	if !platformSupported() {
		return Write{}, ErrUnsupported
	}
	p, err := canonical(path)
	if err != nil {
		return Write{}, err
	}
	if _, err = os.Lstat(p); err == nil {
		return Read(p)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Write{}, ErrUnsafe
	}
	for ancestor := filepath.Dir(p); ; ancestor = filepath.Dir(ancestor) {
		info, err := os.Lstat(ancestor)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return Write{}, ErrUnsafe
			}
		} else {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || checkComponent(ancestor, info) != nil || !trustedAncestor(ancestor, info) {
				return Write{}, ErrUnsafe
			}
		}
		if filepath.Dir(ancestor) == ancestor {
			break
		}
	}
	return Write{Path: p, Mode: 0600}, nil
}
