package configtxn

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadUncreatedDoesNotCreateNestedParents(t *testing.T) {
	d, _ := fixture(t)
	p := filepath.Join(d, "absent", "nested", "config")
	w, err := ReadUncreated(p)
	if err != nil || w.Existed || w.Path != p || w.Mode != 0600 || len(w.Before) != 0 || len(w.Data) != 0 {
		t.Fatalf("snapshot metadata %+v error %v", w, err)
	}
	if _, err := os.Lstat(filepath.Join(d, "absent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("inspection created a directory", err)
	}
	if _, err := Read(p); err == nil {
		t.Fatal("strict Read changed")
	}
}
func TestReadUncreatedRejectsUnsafeAncestors(t *testing.T) {
	d, _ := fixture(t)
	link := filepath.Join(d, "link")
	if err := os.Symlink(d, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadUncreated(filepath.Join(link, "missing", "config")); !errors.Is(err, ErrUnsafe) {
		t.Fatal("symlink ancestor accepted", err)
	}
	broad := filepath.Join(d, "broad")
	if err := os.Mkdir(broad, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(broad, 0777); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadUncreated(filepath.Join(broad, "missing", "config")); !errors.Is(err, ErrUnsafe) {
		t.Fatal("writable foreign ancestor accepted", err)
	}
	p := filepath.Join(d, "file")
	put(t, p, "original", 0600)
	if _, err := ReadUncreated(filepath.Join(p, "missing", "config")); !errors.Is(err, ErrUnsafe) {
		t.Fatal("non-directory ancestor accepted", err)
	}
	w, err := ReadUncreated(p)
	if err != nil || !w.Existed || string(w.Before) != "original" {
		t.Fatal("existing read failed", err)
	}
	w.Data[0] = 'x'
	if string(w.Before) != "original" {
		t.Fatal("snapshot aliases input")
	}
}
