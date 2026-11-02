//go:build linux

package configtxn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

const nodumpFlag = 0x40

func fixtureFlags(t *testing.T, path string, set bool) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	flags, err := unix.IoctlGetInt(int(f.Fd()), unix.FS_IOC_GETFLAGS)
	if err != nil {
		t.Fatal("native inode flag fixture unavailable", err)
	}
	if set {
		if err := unix.IoctlSetPointerInt(int(f.Fd()), unix.FS_IOC_SETFLAGS, flags|nodumpFlag); err != nil {
			t.Fatal("native nodump fixture unavailable", err)
		}
		flags |= nodumpFlag
	}
	return flags
}
func TestFinalFixNodumpRefusal(t *testing.T) {
	for _, point := range []string{"initial", "reviewed", "stage:0", "replaced:0"} {
		t.Run(point, func(t *testing.T) {
			d, s := fixture(t)
			p := filepath.Join(d, "config")
			put(t, p, "original", 0600)
			if point == "initial" {
				fixtureFlags(t, p, true)
			}
			w := snapshot(t, p, "ours")
			if point == "reviewed" {
				fixtureFlags(t, p, true)
			}
			before, _ := os.Stat(p)
			if point == "stage:0" {
				s.checkpoint = func(name string) error {
					if name == point {
						fixtureFlags(t, p, true)
					}
					return nil
				}
			}
			if point == "replaced:0" {
				interruptAt(s, point)
			}
			r, err := s.Apply(context.Background(), []Write{w})
			if point == "replaced:0" {
				if !errors.Is(err, ErrInterrupted) {
					t.Fatal("missing interruption", err)
				}
				fixtureFlags(t, p, true)
				before, _ = os.Stat(p)
				results, recoverErr := New(s.Root).Recover(context.Background())
				if !errors.Is(recoverErr, ErrConflict) || len(results) != 1 || results[0].Outcome != "Conflict" {
					t.Fatalf("recovery did not preserve foreign policy: %v", recoverErr)
				}
				content(t, p, "ours")
				if len(results[0].Backups) != 1 {
					t.Fatal("backup missing")
				}
				content(t, results[0].Backups[0], "original")
			} else {
				if err == nil {
					t.Fatal("unsupported inode policy accepted")
				}
				if point != "initial" && (!errors.Is(err, ErrConflict) || r.Outcome != "Conflict") {
					t.Fatal("policy drift not a conflict", err)
				}
				content(t, p, "original")
				if point == "initial" || point == "reviewed" {
					if _, err := os.Lstat(s.Root); !os.IsNotExist(err) {
						t.Fatal("state created before refusal")
					}
				}
			}
			after, _ := os.Stat(p)
			if !os.SameFile(before, after) || fixtureFlags(t, p, false)&nodumpFlag == 0 {
				t.Fatal("foreign inode/policy changed")
			}
		})
	}
}

func TestFinalFixInodeFlagQueryPolicy(t *testing.T) {
	for _, flags := range []int{0, 0x1000, 0x80000, 0x104c1000} {
		if err := supportedInodeFlags(flags, nil); err != nil {
			t.Fatalf("filesystem-managed flags refused: %x", flags)
		}
	}
	for _, flags := range []int{nodumpFlag, 0x80, 0x4, 0x800000, 0x20000000, 0x40000000, 0x80000000} {
		if err := supportedInodeFlags(flags, nil); !errors.Is(err, ErrConflict) {
			t.Fatalf("user/unknown policy accepted: %x", flags)
		}
	}
	for _, err := range []error{unix.ENOTTY, unix.EOPNOTSUPP, unix.ENOSYS, unix.EIO} {
		if got := supportedInodeFlags(0, err); !errors.Is(got, ErrUnsafe) {
			t.Fatal("unqueryable native metadata accepted")
		}
	}
}
