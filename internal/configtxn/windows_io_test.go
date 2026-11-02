//go:build windows

package configtxn

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWindowsReadUncreated(t *testing.T) {
	d, _ := windowsFixture(t)
	p := filepath.Join(d, "absent", "nested", "config")
	w, err := ReadUncreated(p)
	if err != nil || w.Existed || w.Mode != 0600 || len(w.Before) != 0 || len(w.Data) != 0 {
		t.Fatal("uncreated snapshot", err)
	}
	if _, err := os.Lstat(filepath.Join(d, "absent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("inspection created directory", err)
	}
}

func TestWindowsNoopAndCustomPermissions(t *testing.T) {
	d, s := windowsFixture(t)
	p := filepath.Join(d, "config")
	if err := PrivateFile(p, []byte("original")); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(p)
	w, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Apply(context.Background(), []Write{w})
	if err != nil || r.Outcome != "Noop" {
		t.Fatalf("%+v %v", r, err)
	}
	after, _ := os.Stat(p)
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("no-op rewrote target")
	}
	sid, err := currentSID()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("O:" + sid.String() + "D:P(A;;FA;;;" + sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)(A;;FR;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, _ := sd.DACL()
	if err := windows.SetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(p); !errors.Is(err, ErrUnsafe) {
		t.Fatal("custom ACL not refused", err)
	}
	w.Data = []byte("ours")
	if _, err := s.Apply(context.Background(), []Write{w}); err == nil {
		t.Fatal("custom ACL replaced")
	}
	content(t, p, "original")
}

func TestWindowsSharingAndCancelledCommit(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "sharing", true: "cancelled"}[cancelled], func(t *testing.T) {
			d, s := windowsFixture(t)
			p := filepath.Join(d, "config")
			if err := PrivateFile(p, []byte("original")); err != nil {
				t.Fatal(err)
			}
			w := snapshot(t, p, "ours")
			// Permit snapshots but deny delete sharing, as an editor may do.
			ptr, _ := windows.UTF16PtrFromString(p)
			h, err := windows.CreateFile(ptr, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if h != windows.InvalidHandle {
					windows.CloseHandle(h)
				}
			}()
			ctx := context.Background()
			if cancelled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 30*time.Millisecond)
				defer cancel()
			}
			start := time.Now()
			r, err := s.Apply(ctx, []Write{w})
			if err == nil {
				t.Fatal("sharing lock ignored")
			}
			if time.Since(start) > 5*time.Second {
				t.Fatal("unbounded sharing retries")
			}
			if cancelled && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("%+v %v", r, err)
			}
			content(t, p, "original")
			windows.CloseHandle(h)
			h = windows.InvalidHandle
			r, err = New(s.Root).Apply(context.Background(), []Write{w})
			if err != nil || r.Outcome != "Succeeded" {
				t.Fatalf("retry after sharing release: %+v %v", r, err)
			}
			content(t, p, "ours")
		})
	}
}

func TestWindowsAllApplyInterruptionBoundaries(t *testing.T) {
	for _, point := range []string{"journal-created", "backup:0", "stage:0", "ready:0", "before-replace:0", "stage-permissions:0", "replaced:0", "applied:0", "succeeded"} {
		t.Run(point, func(t *testing.T) {
			d, s := windowsFixture(t)
			p := filepath.Join(d, "config")
			if err := PrivateFile(p, []byte("original")); err != nil {
				t.Fatal(err)
			}
			interruptAt(s, point)
			if _, err := s.Apply(context.Background(), []Write{snapshot(t, p, "ours")}); !errors.Is(err, ErrInterrupted) {
				t.Fatal(err)
			}
			results, err := New(s.Root).Recover(context.Background())
			if err != nil || len(results) != 1 {
				t.Fatalf("%+v %v", results, err)
			}
			want := "original"
			if point == "succeeded" {
				want = "ours"
			}
			content(t, p, want)
			assertWindowsACL(t, p)
		})
	}
}

func TestWindowsExistingNamedStreamRefused(t *testing.T) {
	d, _ := windowsFixture(t)
	p := filepath.Join(d, "config")
	if err := PrivateFile(p, []byte("original")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p+":foreign", []byte("extra stream"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(p); !errors.Is(err, ErrUnsafe) {
		t.Fatal("existing named stream was not refused", err)
	}
	content(t, p, "original")
}
