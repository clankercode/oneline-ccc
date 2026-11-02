//go:build linux

package configtxn

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func linuxACL() []byte {
	b := make([]byte, 44)
	binary.LittleEndian.PutUint32(b, 2)
	entries := [][3]uint32{{1, 6, 0xffffffff}, {2, 4, 65534}, {4, 0, 0xffffffff}, {16, 4, 0xffffffff}, {32, 0, 0xffffffff}}
	for i, e := range entries {
		o := 4 + i*8
		binary.LittleEndian.PutUint16(b[o:], uint16(e[0]))
		binary.LittleEndian.PutUint16(b[o+2:], uint16(e[1]))
		binary.LittleEndian.PutUint32(b[o+4:], e[2])
	}
	return b
}
func setFixtureACL(t *testing.T, path, name string) {
	t.Helper()
	if err := syscall.Setxattr(path, name, linuxACL(), 0); err != nil {
		t.Fatalf("native ACL fixture unavailable: %v", err)
	}
}
func TestRefuseACLAndXattrBeforeMutation(t *testing.T) {
	for _, kind := range []string{"acl", "xattr", "parent-default-acl"} {
		t.Run(kind, func(t *testing.T) {
			d, s := fixture(t)
			dir := filepath.Join(d, "target")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(dir, "config")
			put(t, p, "original", 0640)
			switch kind {
			case "acl":
				setFixtureACL(t, p, "system.posix_acl_access")
			case "xattr":
				if err := syscall.Setxattr(p, "user.synthetic", []byte("retained"), 0); err != nil {
					t.Fatal(err)
				}
			case "parent-default-acl":
				setFixtureACL(t, dir, "system.posix_acl_default")
			}
			w := snapshot(t, p, "ours")
			before, _ := os.Stat(p)
			r, err := s.Apply(context.Background(), []Write{w})
			if !errors.Is(err, ErrUnsafe) {
				t.Fatalf("unsupported metadata accepted: %s %v", r.Outcome, err)
			}
			after, _ := os.Stat(p)
			if !os.SameFile(before, after) {
				t.Fatal("refusal replaced target")
			}
			content(t, p, "original")
			if _, err := os.Lstat(s.Root); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("metadata refusal happened after state mutation", err)
			}
			if kind == "acl" {
				n, err := syscall.Getxattr(p, "system.posix_acl_access", nil)
				if err != nil || n != 44 {
					t.Fatal("original ACL lost", err)
				}
			}
		})
	}
}
func TestMetadataDriftAndRecoveryPreserveForeignACL(t *testing.T) {
	for _, point := range []string{"stage:0", "replaced:0"} {
		t.Run(point, func(t *testing.T) {
			d, s := fixture(t)
			p := filepath.Join(d, "config")
			put(t, p, "original", 0640)
			w := snapshot(t, p, "ours")
			if point == "stage:0" {
				s.checkpoint = func(name string) error {
					if name == point {
						setFixtureACL(t, p, "system.posix_acl_access")
					}
					return nil
				}
				r, err := s.Apply(context.Background(), []Write{w})
				if !errors.Is(err, ErrConflict) || r.Outcome != "Conflict" {
					t.Fatalf("%+v %v", r, err)
				}
				content(t, p, "original")
			} else {
				interruptAt(s, point)
				if _, err := s.Apply(context.Background(), []Write{w}); !errors.Is(err, ErrInterrupted) {
					t.Fatal(err)
				}
				setFixtureACL(t, p, "system.posix_acl_access")
				results, err := New(s.Root).Recover(context.Background())
				if !errors.Is(err, ErrConflict) || len(results) != 1 || results[0].Outcome != "Conflict" {
					t.Fatalf("%+v %v", results, err)
				}
				content(t, p, "ours")
			}
			n, err := syscall.Getxattr(p, "system.posix_acl_access", nil)
			if err != nil || n != 44 {
				t.Fatal("foreign ACL lost", err)
			}
		})
	}
}
func TestPrivateHelpersRefuseInheritedMetadata(t *testing.T) {
	d, _ := fixture(t)
	setFixtureACL(t, d, "system.posix_acl_default")
	if err := PrivateDir(filepath.Join(d, "child")); !errors.Is(err, ErrUnsafe) {
		t.Fatal("private directory inherited unsupported ACL", err)
	}
	if err := PrivateFile(filepath.Join(d, "file"), []byte("private")); !errors.Is(err, ErrUnsafe) {
		t.Fatal("private file inherited unsupported ACL", err)
	}
	for _, name := range []string{"child", "file"} {
		if _, err := os.Lstat(filepath.Join(d, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("helper mutated unsupported parent", err)
		}
	}
}

func TestRefuseForeignGroupBeforeMutation(t *testing.T) {
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	foreign := -1
	for _, gid := range groups {
		if gid != os.Getegid() {
			foreign = gid
			break
		}
	}
	if foreign < 0 {
		t.Skip("no supplemental group for unprivileged fixture chown")
	}
	d, s := fixture(t)
	p := filepath.Join(d, "config")
	put(t, p, "original", 0640)
	w := snapshot(t, p, "ours")
	if err := os.Chown(p, -1, foreign); err != nil {
		t.Fatal(err)
	}
	r, err := s.Apply(context.Background(), []Write{w})
	if !errors.Is(err, ErrUnsafe) {
		t.Fatalf("%+v %v", r, err)
	}
	content(t, p, "original")
	info, _ := os.Stat(p)
	if info.Sys().(*syscall.Stat_t).Gid != uint32(foreign) {
		t.Fatal("foreign group changed")
	}
	if _, err := os.Lstat(s.Root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("state created on foreign group refusal", err)
	}
}
