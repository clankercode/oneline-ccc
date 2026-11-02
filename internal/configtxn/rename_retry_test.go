package configtxn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSharingRetryRetainsReviewedSource(t *testing.T) {
	for _, kind := range []string{"bytes", "mode", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			d, _ := fixture(t)
			src, dst := filepath.Join(d, "stage"), filepath.Join(d, "config")
			put(t, src, "reviewed", 0600)
			put(t, dst, "original", 0600)
			sharingErr := errors.New("synthetic sharing violation")
			attempts := 0
			move := func() error {
				attempts++
				if attempts == 1 {
					switch kind {
					case "bytes":
						put(t, src, "foreign-stage", 0600)
					case "mode":
						if err := os.Chmod(src, 0644); err != nil {
							t.Fatal(err)
						}
					case "hardlink":
						if err := os.Link(src, filepath.Join(d, "alias")); err != nil {
							t.Fatal(err)
						}
					}
					return sharingErr
				}
				return os.Rename(src, dst)
			}
			err := renameRetry(context.Background(), src, dst, digest([]byte("reviewed")), 0600, move, func(err error) bool { return errors.Is(err, sharingErr) }, renameChecks{})
			if !errors.Is(err, ErrConflict) || attempts != 1 {
				t.Fatalf("tampered retry accepted: attempts=%d error=%v", attempts, err)
			}
			content(t, dst, "original")
		})
	}
}

func TestSharingRetryPreservesForeignDestinationAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "destination", true: "cancelled"}[cancelled], func(t *testing.T) {
			d, _ := fixture(t)
			src, dst := filepath.Join(d, "stage"), filepath.Join(d, "config")
			put(t, src, "reviewed", 0600)
			put(t, dst, "original", 0600)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sharingErr := errors.New("sharing")
			attempts := 0
			move := func() error {
				attempts++
				if cancelled {
					cancel()
				} else {
					put(t, dst, "foreign-destination", 0600)
				}
				return sharingErr
			}
			err := renameRetry(ctx, src, dst, digest([]byte("reviewed")), 0600, move, func(err error) bool { return errors.Is(err, sharingErr) }, renameChecks{})
			if cancelled {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				content(t, dst, "original")
			} else {
				if !errors.Is(err, ErrConflict) {
					t.Fatal(err)
				}
				content(t, dst, "foreign-destination")
			}
			if attempts != 1 {
				t.Fatal("unexpected second mutation attempt")
			}
		})
	}
}

func TestSharingRetryRechecksNoopConditions(t *testing.T) {
	d, _ := fixture(t)
	src, dst, a := filepath.Join(d, "stage"), filepath.Join(d, "config"), filepath.Join(d, "unchanged")
	put(t, src, "reviewed", 0600)
	put(t, dst, "original", 0600)
	put(t, a, "old-a", 0600)
	conditions := []Write{snapshot(t, a, "old-a")}
	sharingErr := errors.New("sharing")
	attempts := 0
	move := func() error { attempts++; put(t, a, "foreign-a", 0600); return sharingErr }
	err := renameRetry(context.Background(), src, dst, digest([]byte("reviewed")), 0600, move, func(err error) bool { return errors.Is(err, sharingErr) }, renameChecks{before: func() error { return expectedAll(conditions) }})
	if !errors.Is(err, ErrConflict) || attempts != 1 {
		t.Fatal("no-op precondition bypassed during sharing retry", err, attempts)
	}
	content(t, dst, "original")
	content(t, a, "foreign-a")
}

func TestSharingRetryCannotAcceptFreshForeignDestinationBaseline(t *testing.T) {
	d, _ := fixture(t)
	src, dst := filepath.Join(d, "restore-stage"), filepath.Join(d, "config")
	put(t, src, "original", 0600)
	put(t, dst, "foreign", 0600)
	e := entry{Path: dst, Existed: true, Mode: 0600, AfterHash: digest([]byte("ours"))}
	attempts := 0
	err := renameRetry(context.Background(), src, dst, digest([]byte("original")), 0600, func() error { attempts++; return os.Rename(src, dst) }, func(error) bool { return false }, renameChecks{before: func() error { return expectedOutput(e) }})
	if !errors.Is(err, ErrConflict) || attempts != 0 {
		t.Fatal("accepted newly read foreign destination", err, attempts)
	}
	content(t, dst, "foreign")
}
