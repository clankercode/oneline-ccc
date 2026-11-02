//go:build windows

package configtxn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func windowsFixture(t *testing.T) (string, *Store) {
	t.Helper()
	cache := os.Getenv("TMPDIR")
	if cache == "" {
		t.Fatal("TMPDIR must point to worktree cache")
	}
	d, err := os.MkdirTemp(cache, "windows-txn-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	root := filepath.Join(d, "private")
	if err := PrivateDir(root); err != nil {
		t.Fatal(err)
	}
	return root, New(filepath.Join(root, "state"))
}

func assertWindowsACL(t *testing.T, path string) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	owner, _, err := sd.Owner()
	if err != nil || !windows.EqualSid(owner, user.User.Sid) {
		t.Fatal("owner is not current account", err)
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("DACL is not protected", err)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil || acl.AceCount != 3 {
		t.Fatal("unexpected DACL", err)
	}
	allowed := map[string]bool{user.User.Sid.String(): true, "S-1-5-18": true, "S-1-5-32-544": true}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, i, &ace); err != nil {
			t.Fatal(err)
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || !allowed[sid.String()] {
			t.Fatal("unexpected principal")
		}
		delete(allowed, sid.String())
	}
	if len(allowed) != 0 {
		t.Fatal("missing intended principal")
	}
}

func TestWindowsPrivateStagingAndBroadRefusal(t *testing.T) {
	d, _ := windowsFixture(t)
	p := filepath.Join(d, "config")
	if err := PrivateFile(p, []byte("private fixture")); err != nil {
		t.Fatal(err)
	}
	assertWindowsACL(t, d)
	assertWindowsACL(t, p)
	if err := PrivateFile(p, []byte("overwrite")); err == nil {
		t.Fatal("exclusive write overwrote")
	}
	content(t, p, "private fixture")
	broad := filepath.Join(d, "broad")
	if err := os.Mkdir(broad, 0700); err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, _ := sd.DACL()
	if err := windows.SetNamedSecurityInfo(broad, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	if err := PrivateDir(broad); !errors.Is(err, ErrUnsafe) {
		t.Fatal("broad ACL accepted", err)
	}
	if err := PrivateFile(filepath.Join(broad, "secret"), nil); !errors.Is(err, ErrUnsafe) {
		t.Fatal("broad parent accepted", err)
	}
}

func TestWindowsFirstWritersAndRecovery(t *testing.T) {
	d, s := windowsFixture(t)
	p := filepath.Join(d, "first")
	w1 := snapshot(t, p, "one")
	w2 := snapshot(t, p, "two")
	start := make(chan struct{})
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, w := range []Write{w1, w2} {
		wg.Add(1)
		go func(i int, w Write) {
			defer wg.Done()
			<-start
			_, errs[i] = New(s.Root).Apply(context.Background(), []Write{w})
		}(i, w)
	}
	close(start)
	wg.Wait()
	ok, conflict := 0, 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else if errors.Is(err, ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatal("first writers not serialized")
	}
	assertWindowsACL(t, p)
	for _, foreign := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback", true: "foreign-conflict"}[foreign], func(t *testing.T) {
			d, s := windowsFixture(t)
			a := filepath.Join(d, "a")
			b := filepath.Join(d, "b")
			if err := PrivateFile(a, []byte("old-a")); err != nil {
				t.Fatal(err)
			}
			if err := PrivateFile(b, []byte("old-b")); err != nil {
				t.Fatal(err)
			}
			interruptAt(s, "replaced:0")
			r, err := s.Apply(context.Background(), []Write{snapshot(t, a, "new-a"), snapshot(t, b, "new-b")})
			if !errors.Is(err, ErrInterrupted) {
				t.Fatal(err)
			}
			assertWindowsACL(t, r.Backups[0])
			if foreign {
				put(t, a, "foreign", 0600)
			}
			results, err := New(s.Root).Recover(context.Background())
			if foreign {
				if !errors.Is(err, ErrConflict) || len(results) != 1 || results[0].Outcome != "Conflict" {
					t.Fatalf("%+v %v", results, err)
				}
				content(t, a, "foreign")
			} else {
				if err != nil || len(results) != 1 || results[0].Outcome != "RolledBack" {
					t.Fatalf("%+v %v", results, err)
				}
				content(t, a, "old-a")
			}
			content(t, b, "old-b")
			assertWindowsACL(t, a)
		})
	}
}

func TestWindowsRejectsADSAndJunction(t *testing.T) {
	d, s := windowsFixture(t)
	p := filepath.Join(d, "config")
	if err := PrivateFile(p, []byte("original")); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(p + ":stream"); !errors.Is(err, ErrUnsafe) {
		t.Fatal("ADS accepted", err)
	}
	if err := PrivateFile(p+":stream", nil); !errors.Is(err, ErrUnsafe) {
		t.Fatal("ADS creation accepted", err)
	}
	// Junctions require no symlink privilege. Construct a mount-point reparse buffer.
	link := filepath.Join(d, "junction")
	if err := os.Mkdir(link, 0700); err != nil {
		t.Fatal(err)
	}
	name := windows.StringToUTF16(`\??\` + d)
	data := make([]byte, 16+len(name)*2)
	header := (*[4]uint16)(unsafe.Pointer(&data[8]))
	header[1] = uint16((len(name) - 1) * 2)
	header[2] = 0
	*(*uint32)(unsafe.Pointer(&data[0])) = windows.IO_REPARSE_TAG_MOUNT_POINT
	*(*uint16)(unsafe.Pointer(&data[4])) = uint16(len(data) - 8)
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(&data[16])), len(name)), name)
	ptr, _ := windows.UTF16PtrFromString(link)
	h, err := windows.CreateFile(ptr, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatal(err)
	}
	var n uint32
	err = windows.DeviceIoControl(h, windows.FSCTL_SET_REPARSE_POINT, &data[0], uint32(len(data)), nil, 0, &n, nil)
	windows.CloseHandle(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Read(filepath.Join(link, "config")); !errors.Is(err, ErrUnsafe) {
		t.Fatal("junction parent accepted", err)
	}
	if _, err := ReadUncreated(filepath.Join(link, "absent", "config")); !errors.Is(err, ErrUnsafe) {
		t.Fatal("junction ancestor accepted for uncreated target", err)
	}
	if err := PrivateDir(link); !errors.Is(err, ErrUnsafe) {
		t.Fatal("junction target accepted", err)
	}
	if _, err := s.Apply(context.Background(), []Write{{Path: filepath.Join(link, "new"), Data: []byte("no")}}); err == nil {
		t.Fatal("junction write accepted")
	}
	content(t, p, "original")
}
