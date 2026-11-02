package configtxn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNoopInputsRemainBatchPreconditions(t *testing.T) {
	for _, point := range []string{"stage:0", "before-replace:0", "applied:0"} {
		t.Run(point, func(t *testing.T) {
			d, s := fixture(t)
			a := filepath.Join(d, "unchanged")
			b := filepath.Join(d, "changed")
			put(t, a, "old-a", 0600)
			put(t, b, "old-b", 0600)
			writes := []Write{snapshot(t, a, "old-a"), snapshot(t, b, "new-b")}
			s.checkpoint = func(name string) error {
				if name == point {
					put(t, a, "foreign-a", 0600)
				}
				return nil
			}
			r, err := s.Apply(context.Background(), writes)
			if !errors.Is(err, ErrConflict) || r.Outcome != "Conflict" {
				t.Fatalf("stale reviewed batch accepted: outcome=%s error=%v", r.Outcome, err)
			}
			content(t, a, "foreign-a")
			content(t, b, "old-b")
		})
	}
}

func TestNoopInputParentIsLocked(t *testing.T) {
	d, s := fixture(t)
	parent := filepath.Join(d, "other-parent")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(parent, "unchanged")
	b := filepath.Join(d, "changed")
	put(t, a, "old-a", 0600)
	put(t, b, "old-b", 0600)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	blocker, err := lock(context.Background(), filepath.Join(parent, ".ccc-configtxn.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()
	journalStarted := false
	s.checkpoint = func(name string) error {
		if name == "journal-created" {
			journalStarted = true
		}
		return nil
	}
	_, err = s.Apply(ctx, []Write{snapshot(t, a, "old-a"), snapshot(t, b, "new-b")})
	if !errors.Is(err, context.DeadlineExceeded) || journalStarted {
		t.Fatal("no-op parent lock bypassed", err)
	}
	content(t, b, "old-b")
}
