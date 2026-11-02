package configtxn

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func fixture(t *testing.T) (string, *Store) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("protected POSIX implementation only")
	}
	if os.Getenv("TMPDIR") == "" || !bytes.Contains([]byte(os.Getenv("TMPDIR")), []byte("/.cache/")) {
		t.Fatal("TMPDIR must be worktree .cache")
	}
	d := t.TempDir()
	return d, New(filepath.Join(d, "state"))
}

func put(t *testing.T, path, data string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), mode); err != nil {
		t.Fatal(err)
	}
}
func snapshot(t *testing.T, path, data string) Write {
	t.Helper()
	w, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	w.Data = []byte(data)
	return w
}
func content(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != want {
		t.Fatal("unexpected target content")
	}
}

func TestReadApplyBackupAndNoop(t *testing.T) {
	d, s := fixture(t)
	p := filepath.Join(d, "config")
	put(t, p, "# retain\nforeign = true\n", 0640)
	w, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Existed || string(w.Before) != "# retain\nforeign = true\n" || !bytes.Equal(w.Data, w.Before) {
		t.Fatal("Read did not capture existing bytes")
	}
	w.Data[0] = '!'
	if w.Before[0] != '#' {
		t.Fatal("snapshot aliases proposed bytes")
	}
	w.Data[0] = '#'
	before, _ := os.Stat(p)
	r, err := s.Apply(context.Background(), []Write{w})
	if err != nil || r.Outcome != "Noop" {
		t.Fatalf("no-op: %+v %v", r, err)
	}
	after, _ := os.Stat(p)
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("no-op rewrote file")
	}
	w.Data = []byte("# retain\nforeign = true\nowned = 1\n")
	r, err = s.Apply(context.Background(), []Write{w})
	if err != nil || r.Outcome != "Succeeded" || len(r.Backups) != 1 {
		t.Fatalf("apply: %+v %v", r, err)
	}
	content(t, p, string(w.Data))
	content(t, r.Backups[0], string(w.Before))
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0640 {
		t.Fatal("lost permissions")
	}
	info, _ = os.Stat(r.Backups[0])
	if info.Mode().Perm() != 0600 {
		t.Fatal("backup not private")
	}
	info, _ = os.Stat(s.Root)
	if info.Mode().Perm() != 0700 {
		t.Fatal("store not private")
	}
}

func TestConcurrentFirstWriters(t *testing.T) {
	d, _ := fixture(t)
	p := filepath.Join(d, "absent")
	w1 := snapshot(t, p, "one")
	w2 := snapshot(t, p, "two")
	// Separate coordinators still serialize on the stable target parent.
	stores := []*Store{New(filepath.Join(d, "s1")), New(filepath.Join(d, "s2"))}
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, w := range []Write{w1, w2} {
		wg.Add(1)
		go func(i int, w Write) {
			defer wg.Done()
			<-start
			_, errs[i] = stores[i].Apply(context.Background(), []Write{w})
		}(i, w)
	}
	close(start)
	wg.Wait()
	good, conflicts := 0, 0
	for _, err := range errs {
		if err == nil {
			good++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if good != 1 || conflicts != 1 {
		t.Fatalf("successes %d conflicts %d", good, conflicts)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0600 {
		t.Fatal("new target not private")
	}
}

func TestDriftBeforeAndAfterStaging(t *testing.T) {
	for _, point := range []string{"before-apply", "stage:0", "before-replace:0"} {
		t.Run(point, func(t *testing.T) {
			d, s := fixture(t)
			p := filepath.Join(d, "config")
			put(t, p, "original", 0600)
			w := snapshot(t, p, "ours")
			if point == "before-apply" {
				put(t, p, "foreign", 0600)
			} else {
				s.checkpoint = func(name string) error {
					if name == point {
						put(t, p, "foreign", 0600)
					}
					return nil
				}
			}
			r, err := s.Apply(context.Background(), []Write{w})
			if !errors.Is(err, ErrConflict) || r.Outcome != "Conflict" {
				t.Fatalf("%+v %v", r, err)
			}
			content(t, p, "foreign")
		})
	}
}

func TestRefusalsAndCancellation(t *testing.T) {
	for _, kind := range []string{"duplicate", "symlink-parent", "symlink-target", "directory", "large", "large-output", "cancelled", "insecure-store"} {
		t.Run(kind, func(t *testing.T) {
			d, s := fixture(t)
			p := filepath.Join(d, "config")
			put(t, p, "original", 0600)
			w := snapshot(t, p, "ours")
			writes := []Write{w}
			ctx := context.Background()
			switch kind {
			case "duplicate":
				writes = append(writes, w)
			case "symlink-parent":
				link := filepath.Join(d, "link")
				if err := os.Symlink(d, link); err != nil {
					t.Fatal(err)
				}
				writes[0].Path = filepath.Join(link, "config")
			case "symlink-target":
				link := filepath.Join(d, "link")
				if err := os.Symlink(p, link); err != nil {
					t.Fatal(err)
				}
				writes[0].Path = link
			case "directory":
				writes[0].Path = d
			case "large":
				put(t, p, string(bytes.Repeat([]byte("x"), 5*1024*1024+1)), 0600)
			case "large-output":
				writes[0].Data = bytes.Repeat([]byte("x"), 5*1024*1024+1)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "insecure-store":
				if err := os.Mkdir(s.Root, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.Apply(ctx, writes); err == nil {
				t.Fatal("unsafe write accepted")
			}
			if kind != "large" {
				content(t, p, "original")
			}
		})
	}
}
