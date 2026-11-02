package configtxn

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrivateStagingHelpers(t *testing.T) {
	d, _ := fixture(t)
	dir := filepath.Join(d, "nested", "staging")
	if err := PrivateDir(dir); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Dir(dir), dir} {
		info, err := os.Stat(p)
		if err != nil || info.Mode().Perm() != 0700 {
			t.Fatal("directory not private", err)
		}
	}
	p := filepath.Join(dir, "config")
	if err := PrivateFile(p, []byte("sensitive fixture")); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0600 {
		t.Fatal("file not private")
	}
	if err := PrivateFile(p, []byte("overwrite")); err == nil {
		t.Fatal("exclusive helper overwrote existing file")
	}
	content(t, p, "sensitive fixture")
	link := filepath.Join(d, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if err := PrivateDir(link); err == nil {
		t.Fatal("accepted symlink directory")
	}
	if err := PrivateFile(filepath.Join(link, "escape"), nil); err == nil {
		t.Fatal("accepted symlink parent")
	}
	broad := filepath.Join(d, "broad")
	if err := os.Mkdir(broad, 0755); err != nil {
		t.Fatal(err)
	}
	if err := PrivateDir(broad); err == nil {
		t.Fatal("adopted insecure directory")
	}
}

func TestUnsupportedPlatformRefusesBeforeMutation(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Skip("native unsupported-platform test")
	}
	d := t.TempDir()
	p := filepath.Join(d, "absent")
	if err := PrivateDir(p); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if err := PrivateFile(p, nil); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := New(p).Apply(t.Context(), []Write{{Path: filepath.Join(d, "config"), Data: []byte("ours")}}); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := New(p).Recover(t.Context()); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unsupported platform mutated state")
	}
}
