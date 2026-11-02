package configtxn

import "path/filepath"

// PrivateDir creates missing directories with owner-only access, or verifies an
// existing private directory. It never chmods or adopts an insecure directory.
func PrivateDir(path string) error {
	if !platformSupported() {
		return ErrUnsupported
	}
	p, err := canonical(path)
	if err != nil {
		return err
	}
	return ensurePrivateDir(p)
}

// PrivateFile creates a bounded, durable owner-only file exclusively. Missing
// private parents are created; existing destinations are never changed.
func PrivateFile(path string, data []byte) error {
	if !platformSupported() {
		return ErrUnsupported
	}
	p, err := canonical(path)
	if err != nil {
		return err
	}
	if len(data) > maxSnapshot {
		return ErrLimit
	}
	if err = ensurePrivateDir(filepath.Dir(p)); err != nil {
		return err
	}
	return writeExclusive(p, data, 0600)
}
