//go:build windows

package configtxn

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
)

func holdDeleteSharing(t *testing.T, path string) windows.Handle {
	t.Helper()
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(ptr, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
func TestWindowsSharingRetryNeverInstallsTamperedSource(t *testing.T) {
	for _, kind := range []string{"bytes", "acl", "attributes", "hardlink", "stream"} {
		t.Run(kind, func(t *testing.T) {
			d, s := windowsFixture(t)
			p := filepath.Join(d, "config")
			if err := PrivateFile(p, []byte("original")); err != nil {
				t.Fatal(err)
			}
			w := snapshot(t, p, "reviewed")
			h := holdDeleteSharing(t, p)
			defer func() {
				if h != windows.InvalidHandle {
					windows.CloseHandle(h)
				}
			}()
			blocked := false
			s.checkpoint = func(name string) error {
				if name != "sharing-retry:0" {
					return nil
				}
				if blocked {
					t.Fatal("second sharing attempt reached before source refusal")
				}
				blocked = true
				stages, _ := filepath.Glob(filepath.Join(d, ".ccc-*.stage"))
				if len(stages) != 1 {
					t.Fatal("missing source stage")
				}
				src := stages[0]
				switch kind {
				case "bytes":
					put(t, src, "foreign-stage", 0600)
				case "acl":
					sid, _ := currentSID()
					sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)(A;;FR;;;WD)")
					if err != nil {
						t.Fatal(err)
					}
					acl, _, _ := sd.DACL()
					if err := windows.SetNamedSecurityInfo(src, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
						t.Fatal(err)
					}
				case "attributes":
					ptr, _ := windows.UTF16PtrFromString(src)
					if err := windows.SetFileAttributes(ptr, windows.FILE_ATTRIBUTE_ARCHIVE|windows.FILE_ATTRIBUTE_READONLY); err != nil {
						t.Fatal(err)
					}
				case "hardlink":
					if err := os.Link(src, filepath.Join(d, "alias")); err != nil {
						t.Fatal(err)
					}
				case "stream":
					if err := os.WriteFile(src+":foreign", []byte("foreign stream"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				windows.CloseHandle(h)
				h = windows.InvalidHandle
				return nil
			}
			r, err := s.Apply(context.Background(), []Write{w})
			if !blocked || !errors.Is(err, ErrConflict) || r.Outcome != "Conflict" {
				t.Fatalf("blocked=%t outcome=%s error=%v", blocked, r.Outcome, err)
			}
			content(t, p, "original")
			content(t, r.Backups[0], "original")
			assertWindowsACL(t, p)
		})
	}
}

func TestWindowsRollbackRetryNeverInstallsTamperedSource(t *testing.T) {
	d, s := windowsFixture(t)
	p := filepath.Join(d, "config")
	if err := PrivateFile(p, []byte("original")); err != nil {
		t.Fatal(err)
	}
	interruptAt(s, "replaced:0")
	if _, err := s.Apply(context.Background(), []Write{snapshot(t, p, "reviewed")}); !errors.Is(err, ErrInterrupted) {
		t.Fatal(err)
	}
	h := holdDeleteSharing(t, p)
	defer func() {
		if h != windows.InvalidHandle {
			windows.CloseHandle(h)
		}
	}()
	s = New(s.Root)
	blocked := false
	s.checkpoint = func(name string) error {
		if name == "rollback-sharing-retry:0" {
			blocked = true
			stages, _ := filepath.Glob(filepath.Join(d, ".ccc-*.rollback"))
			if len(stages) != 1 {
				t.Fatal("restore stage missing")
			}
			put(t, stages[0], "foreign-restore", 0600)
			windows.CloseHandle(h)
			h = windows.InvalidHandle
		}
		return nil
	}
	results, err := s.Recover(context.Background())
	if !blocked || !errors.Is(err, ErrConflict) || len(results) != 1 || results[0].Outcome != "Conflict" {
		t.Fatalf("blocked=%t results=%+v error=%v", blocked, results, err)
	}
	content(t, p, "reviewed")
	content(t, results[0].Backups[0], "original")
}
