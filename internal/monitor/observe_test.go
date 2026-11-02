package monitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) *Manager {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("POSIX fixture requires native Linux/macOS protections")
	}
	root := t.TempDir()
	home := filepath.Join(root, "user home 雪%$")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	codex := filepath.Join(root, "client")
	if err := os.Mkdir(codex, 0700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, "ccc app 雪%$")
	if err := os.WriteFile(executable, []byte("fixture executable"), 0700); err != nil {
		t.Fatal(err)
	}
	m := New(filepath.Join(root, "state"), codex, executable)
	m.home = home
	m.os = "linux"
	m.uid = "1234"
	return m
}
func TestObserveOnceExactPrivateAndReadOnly(t *testing.T) {
	m := fixture(t)
	config := filepath.Join(m.codexHome, "config.toml")
	raw := []byte("fixture-private-endpoint and synthetic credential\n")
	if err := os.WriteFile(config, raw, 0600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(config)
	if err := m.Observe(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(m.stateRoot, "monitor", "observation.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "fixture-private") || strings.Contains(string(data), m.codexHome) || strings.Contains(string(data), m.stateRoot) {
		t.Fatal("observation disclosed content or private paths")
	}
	var record observation
	if json.Unmarshal(data, &record) != nil {
		t.Fatal("invalid observation")
	}
	digest := sha256.Sum256(raw)
	if record.Version != 1 || !record.Config.Exists || record.Config.SHA256 != hex.EncodeToString(digest[:]) || record.Config.ModifiedAt.IsZero() || record.ObservedAt.IsZero() {
		t.Fatal("inexact observation")
	}
	info, _ := os.Stat(path)
	dir, _ := os.Stat(filepath.Dir(path))
	if info.Mode().Perm() != 0600 || dir.Mode().Perm() != 0700 {
		t.Fatal("observation not private")
	}
	after, _ := os.Stat(config)
	actual, _ := os.ReadFile(config)
	if string(actual) != string(raw) || !before.ModTime().Equal(after.ModTime()) || !os.SameFile(before, after) {
		t.Fatal("client config changed")
	}
	if err := m.Observe(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatal("observer accumulated files/history")
	}
}
func TestObserveMissingAndCancellation(t *testing.T) {
	m := fixture(t)
	m.codexHome = filepath.Join(t.TempDir(), "missing", "also missing")
	if err := m.Observe(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(m.stateRoot, "monitor", "observation.json"))
	var record observation
	json.Unmarshal(data, &record)
	if record.Config.Exists || record.Config.SHA256 != "" || !record.Config.ModifiedAt.IsZero() {
		t.Fatal("missing config invented evidence")
	}
	m = fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if m.Observe(ctx, true) == nil {
		t.Fatal("cancellation ignored")
	}
	if _, err := os.Stat(m.stateRoot); !os.IsNotExist(err) {
		t.Fatal("canceled observer wrote state")
	}
	m = fixture(t)
	m.interval = time.Millisecond
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- m.Observe(ctx, false) }()
	deadline := time.After(time.Second)
	for {
		if _, err := os.Stat(filepath.Join(m.stateRoot, "monitor", "observation.json")); err == nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("observer never sampled")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("continuous observer ignored cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("observer did not stop")
	}
}
func TestObserveRefusesUnsafeAndOversizedWithoutDisclosure(t *testing.T) {
	for _, kind := range []string{"symlink", "large", "unsafe state", "foreign observation", "overlap"} {
		t.Run(kind, func(t *testing.T) {
			m := fixture(t)
			path := filepath.Join(m.codexHome, "config.toml")
			switch kind {
			case "symlink":
				target := filepath.Join(t.TempDir(), "private")
				os.WriteFile(target, []byte("fixture-private"), 0600)
				if err := os.Symlink(target, path); err != nil {
					t.Skip("symlinks unavailable")
				}
			case "large":
				f, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				f.Truncate(5*1024*1024 + 1)
				f.Close()
			case "unsafe state":
				os.Mkdir(m.stateRoot, 0755)
			case "foreign observation":
				os.MkdirAll(filepath.Join(m.stateRoot, "monitor"), 0700)
				os.WriteFile(filepath.Join(m.stateRoot, "monitor", "observation.json"), []byte("foreign"), 0644)
			case "overlap":
				m.stateRoot = m.codexHome
			}
			err := m.Observe(context.Background(), true)
			if err == nil || strings.Contains(err.Error(), m.codexHome) || strings.Contains(err.Error(), "fixture-private") {
				t.Fatal("unsafe observer succeeded or leaked details")
			}
		})
	}
}

func TestObserverRefusesForeignPrivateObservation(t *testing.T) {
	m := fixture(t)
	dir := filepath.Join(m.stateRoot, "monitor")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "observation.json")
	foreign := []byte(`{"foreign":"fixture-private"}`)
	os.WriteFile(path, foreign, 0600)
	if err := m.Observe(context.Background(), true); err == nil {
		t.Fatal("foreign private observation overwritten")
	}
	data, _ := os.ReadFile(path)
	if string(data) != string(foreign) {
		t.Fatal("foreign observation not preserved")
	}
}
