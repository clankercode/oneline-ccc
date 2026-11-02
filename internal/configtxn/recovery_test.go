package configtxn

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func interruptAt(s *Store, point string) {
	s.checkpoint = func(name string) error {
		if name == point {
			return errors.New("secret-bearing hook error must not escape")
		}
		return nil
	}
}

func TestRecoverEveryApplyBoundary(t *testing.T) {
	points := []string{"journal-created", "backup:0", "stage:0", "ready:0", "backup:1", "stage:1", "ready:1", "before-replace:0", "stage-permissions:0", "replaced:0", "applied:0", "before-replace:1", "stage-permissions:1", "replaced:1", "applied:1", "succeeded"}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			d, s := fixture(t)
			a := filepath.Join(d, "a")
			b := filepath.Join(d, "b")
			put(t, a, "old-a", 0640)
			put(t, b, "old-b", 0600)
			writes := []Write{snapshot(t, a, "new-a"), snapshot(t, b, "new-b")}
			interruptAt(s, point)
			r, err := s.Apply(context.Background(), writes)
			if !errors.Is(err, ErrInterrupted) || r.Outcome != "Unknown" || r.ID == "" {
				t.Fatalf("%+v %v", r, err)
			}
			if strings.Contains(err.Error(), "secret-bearing") {
				t.Fatal("leaked hook error")
			}
			recovered, err := New(s.Root).Recover(context.Background())
			if err != nil || len(recovered) != 1 {
				t.Fatalf("%+v %v", recovered, err)
			}
			wantOutcome := "RolledBack"
			wantA, wantB := "old-a", "old-b"
			if point == "succeeded" {
				wantOutcome = "Succeeded"
				wantA, wantB = "new-a", "new-b"
			}
			if recovered[0].Outcome != wantOutcome {
				t.Fatalf("outcome %s", recovered[0].Outcome)
			}
			content(t, a, wantA)
			content(t, b, wantB)
			if _, err = New(s.Root).Recover(context.Background()); err != nil {
				t.Fatal("recovery not idempotent", err)
			}
			stages, _ := filepath.Glob(filepath.Join(d, ".ccc-*.stage"))
			if len(stages) != 0 {
				t.Fatal("unremoved stages")
			}
		})
	}
}

func TestRecoveryPreservesForeignEdits(t *testing.T) {
	for _, when := range []string{"before-recovery", "before-rollback:0"} {
		t.Run(when, func(t *testing.T) {
			d, s := fixture(t)
			p := filepath.Join(d, "config")
			put(t, p, "original", 0600)
			w := snapshot(t, p, "ours")
			interruptAt(s, "replaced:0")
			_, err := s.Apply(context.Background(), []Write{w})
			if !errors.Is(err, ErrInterrupted) {
				t.Fatal(err)
			}
			s = New(s.Root)
			if when == "before-recovery" {
				put(t, p, "foreign", 0600)
			} else {
				s.checkpoint = func(name string) error {
					if name == when {
						put(t, p, "foreign", 0600)
					}
					return nil
				}
			}
			results, err := s.Recover(context.Background())
			if !errors.Is(err, ErrConflict) || len(results) != 1 || results[0].Outcome != "Conflict" {
				t.Fatalf("%+v %v", results, err)
			}
			content(t, p, "foreign")
			content(t, results[0].Backups[0], "original")
			fresh := snapshot(t, p, "next")
			if _, err = New(s.Root).Apply(context.Background(), []Write{fresh}); !errors.Is(err, ErrConflict) {
				t.Fatal("new mutation not blocked by unresolved conflict", err)
			}
			content(t, p, "foreign")
		})
	}
}

func TestRecoverNewFileAndRollbackBoundaries(t *testing.T) {
	for _, existed := range []bool{false, true} {
		for _, point := range []string{"rollback-intent", "rollback-staged:0", "rollback-permissions:0", "before-rollback:0", "rollback-replaced:0", "restored:0", "rolled-back"} {
			t.Run(fmt.Sprintf("%t/%s", existed, point), func(t *testing.T) {
				if !existed && (point == "rollback-staged:0" || point == "rollback-permissions:0") {
					t.Skip("absent original needs no restore stage")
				}
				d, s := fixture(t)
				p := filepath.Join(d, "config")
				if existed {
					put(t, p, "original", 0640)
				}
				w := snapshot(t, p, "ours")
				interruptAt(s, "replaced:0")
				if _, err := s.Apply(context.Background(), []Write{w}); !errors.Is(err, ErrInterrupted) {
					t.Fatal(err)
				}
				s = New(s.Root)
				interruptAt(s, point)
				if _, err := s.Recover(context.Background()); !errors.Is(err, ErrInterrupted) {
					t.Fatalf("checkpoint %s: %v", point, err)
				}
				results, err := New(s.Root).Recover(context.Background())
				if err != nil || len(results) != 1 || results[0].Outcome != "RolledBack" {
					t.Fatalf("%+v %v", results, err)
				}
				if existed {
					content(t, p, "original")
					info, _ := os.Stat(p)
					if info.Mode().Perm() != 0640 {
						t.Fatal("restore mode")
					}
				} else if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("first write not removed", err)
				}
			})
		}
	}
}

func TestCancellationAtCommitBoundary(t *testing.T) {
	for _, point := range []string{"stage:0", "before-replace:0", "applied:0"} {
		t.Run(point, func(t *testing.T) {
			d, s := fixture(t)
			a := filepath.Join(d, "a")
			b := filepath.Join(d, "b")
			put(t, a, "old-a", 0600)
			put(t, b, "old-b", 0600)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s.checkpoint = func(name string) error {
				if name == point {
					cancel()
				}
				return nil
			}
			_, err := s.Apply(ctx, []Write{snapshot(t, a, "new-a"), snapshot(t, b, "new-b")})
			if !errors.Is(err, context.Canceled) {
				t.Fatal("expected cancellation", err)
			}
			content(t, a, "old-a")
			content(t, b, "old-b")
		})
	}
}
