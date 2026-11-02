package configtxn

import (
	"bytes"
	"os"
)

func (s *Store) validate(writes []Write) ([]Write, []Write, error) {
	if len(writes) > maxWrites {
		return nil, nil, ErrLimit
	}
	seen := map[string]bool{}
	var all, changed []Write
	for _, w := range writes {
		p, err := canonical(w.Path)
		if err != nil {
			return nil, nil, err
		}
		w.Path = p
		if seen[pathKey(p)] || reserved(s.Root, p) || (!w.Existed && len(w.Before) != 0) || w.Mode&^os.FileMode(0777) != 0 {
			return nil, nil, ErrUnsafe
		}
		seen[pathKey(p)] = true
		if len(w.Before) > maxSnapshot || len(w.Data) > maxSnapshot {
			return nil, nil, ErrLimit
		}
		if err = expected(w); err != nil {
			return nil, nil, err
		}

		w.Before = bytes.Clone(w.Before)
		w.Data = bytes.Clone(w.Data)
		if !w.Existed {
			w.Mode = 0600
		}
		all = append(all, w)
		if !w.Existed || !bytes.Equal(w.Before, w.Data) {
			changed = append(changed, w)
		}
	}
	return all, changed, nil
}

func expectedAll(writes []Write) error {
	for _, w := range writes {
		if err := expected(w); err != nil {
			return err
		}
	}
	return nil
}
func advanceCondition(conditions []Write, committed Write) {
	for i := range conditions {
		if pathKey(conditions[i].Path) == pathKey(committed.Path) {
			conditions[i].Before = bytes.Clone(committed.Data)
			conditions[i].Existed = true
			conditions[i].Mode = committed.Mode
			return
		}
	}
}
