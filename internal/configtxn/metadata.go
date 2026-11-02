package configtxn

import (
	"os"
	"path/filepath"
)

// Refuse metadata outside the supported platform policy before creating state,
// locks or stages. Inspection remains read-only even for unsupported metadata.
func metadataPreflight(writes []Write) error {
	if !platformSupported() {
		return ErrUnsupported
	}
	if len(writes) > maxWrites {
		return ErrLimit
	}
	for _, w := range writes {
		p, err := canonical(w.Path)
		if err != nil {
			return err
		}
		if err = targetMetadata(p); err != nil {
			return err
		}
	}
	return nil
}
func targetMetadata(path string) error {
	if err := checkPath(path, true); err != nil {
		return err
	}
	if err := targetParent(filepath.Dir(path)); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() {
		return ErrUnsafe
	}
	return supportedMetadata(path)
}

func expectedOutput(e entry) error {
	if targetMetadata(e.Path) != nil {
		return ErrConflict
	}
	now, err := Read(e.Path)
	if err != nil || !output(e, now) {
		return ErrConflict
	}
	return nil
}
