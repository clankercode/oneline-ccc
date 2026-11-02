package configtxn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestForeignEditAfterPermissionDurability(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "apply", true: "rollback"}[rollback], func(t *testing.T) {
			d, s := fixture(t)
			p := filepath.Join(d, "config")
			put(t, p, "original", 0640)
			w := snapshot(t, p, "ours")
			point := "stage-permissions:0"
			if rollback {
				interruptAt(s, "replaced:0")
				if _, err := s.Apply(context.Background(), []Write{w}); !errors.Is(err, ErrInterrupted) {
					t.Fatal(err)
				}
				s = New(s.Root)
				point = "rollback-permissions:0"
			}
			hit := false
			s.checkpoint = func(name string) error {
				if name == point {
					hit = true
					put(t, p, "foreign", 0640)
				}
				return nil
			}
			var err error
			if rollback {
				_, err = s.Recover(context.Background())
			} else {
				_, err = s.Apply(context.Background(), []Write{w})
			}
			if !hit || !errors.Is(err, ErrConflict) {
				t.Fatalf("hit=%t error=%v", hit, err)
			}
			content(t, p, "foreign")
		})
	}
}

func TestRecoveryRejectsChangedParentsAndLargeForeignFile(t *testing.T) {
	for _, kind := range []string{"symlink-parent", "missing-parent", "large"} {
		t.Run(kind, func(t *testing.T) {
			d, s := fixture(t)
			dir := filepath.Join(d, "target")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(dir, "config")
			put(t, p, "original", 0600)
			w := snapshot(t, p, "ours")
			interruptAt(s, "replaced:0")
			if _, err := s.Apply(context.Background(), []Write{w}); !errors.Is(err, ErrInterrupted) {
				t.Fatal(err)
			}
			if kind == "large" {
				put(t, p, string(make([]byte, maxSnapshot+1)), 0600)
			} else {
				moved := filepath.Join(d, "moved")
				if err := os.Rename(dir, moved); err != nil {
					t.Fatal(err)
				}
				if kind == "symlink-parent" {
					if err := os.Symlink(moved, dir); err != nil {
						t.Fatal(err)
					}
				}
			}
			results, err := New(s.Root).Recover(context.Background())
			if !errors.Is(err, ErrConflict) || len(results) != 1 || results[0].Outcome != "Conflict" {
				t.Fatalf("%+v %v", results, err)
			}
			if kind != "large" {
				content(t, filepath.Join(d, "moved", "config"), "ours")
			}
		})
	}
}

func TestStageTamperingNeverChangesOriginal(t *testing.T) {
	d, s := fixture(t)
	p := filepath.Join(d, "config")
	put(t, p, "original", 0600)
	s.checkpoint = func(name string) error {
		if name == "stage:0" {
			matches, _ := filepath.Glob(filepath.Join(d, ".ccc-*.stage"))
			if len(matches) != 1 {
				t.Fatal("stage missing")
			}
			put(t, matches[0], "foreign-stage", 0600)
		}
		return nil
	}
	r, err := s.Apply(context.Background(), []Write{snapshot(t, p, "ours")})
	if !errors.Is(err, ErrConflict) || r.Outcome != "Conflict" {
		t.Fatalf("%+v %v", r, err)
	}
	content(t, p, "original")
}

func TestRecoverRefusesCorruptedBackup(t *testing.T) {
	d, s := fixture(t)
	p := filepath.Join(d, "config")
	put(t, p, "original", 0600)
	interruptAt(s, "replaced:0")
	r, err := s.Apply(context.Background(), []Write{snapshot(t, p, "ours")})
	if !errors.Is(err, ErrInterrupted) {
		t.Fatal(err)
	}
	put(t, r.Backups[0], "tampered", 0600)
	if _, err = New(s.Root).Recover(context.Background()); !errors.Is(err, ErrJournal) {
		t.Fatal("corrupted backup accepted", err)
	}
	content(t, p, "ours")
}

func TestApplyAutomaticallyRecoversInterruptedOperation(t *testing.T) {
	d, s := fixture(t)
	p := filepath.Join(d, "config")
	put(t, p, "original", 0600)
	w := snapshot(t, p, "ours")
	interruptAt(s, "replaced:0")
	if _, err := s.Apply(context.Background(), []Write{w}); !errors.Is(err, ErrInterrupted) {
		t.Fatal(err)
	}
	r, err := New(s.Root).Apply(context.Background(), []Write{w})
	if err != nil || r.Outcome != "Succeeded" {
		t.Fatalf("%+v %v", r, err)
	}
	content(t, p, "ours")
}

func TestForeignStageAtFinalBoundaryIsNotInstalled(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "apply", true: "rollback"}[rollback], func(t *testing.T) {
			d, s := fixture(t)
			p := filepath.Join(d, "config")
			put(t, p, "original", 0640)
			w := snapshot(t, p, "ours")
			point, suffix := "stage-permissions:0", "stage"
			want := "original"
			if rollback {
				interruptAt(s, "replaced:0")
				if _, err := s.Apply(context.Background(), []Write{w}); !errors.Is(err, ErrInterrupted) {
					t.Fatal(err)
				}
				s = New(s.Root)
				point, suffix, want = "rollback-permissions:0", "rollback", "ours"
			}
			s.checkpoint = func(name string) error {
				if name == point {
					matches, _ := filepath.Glob(filepath.Join(d, ".ccc-*."+suffix))
					if len(matches) != 1 {
						t.Fatal("missing stage")
					}
					put(t, matches[0], "tampered-stage", 0640)
				}
				return nil
			}
			var err error
			if rollback {
				_, err = s.Recover(context.Background())
			} else {
				_, err = s.Apply(context.Background(), []Write{w})
			}
			if !errors.Is(err, ErrConflict) {
				t.Fatal(err)
			}
			content(t, p, want)
		})
	}
}
