package monitor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// This remains an executing test on Windows; POSIX fixture guards do not skip
// the package or conceal the unsupported native lifecycle contract.
func TestWindowsUnavailableNoNativeActionsOrFilesystemWrites(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native Windows unsupported-platform contract")
	}
	root := t.TempDir()
	state := filepath.Join(root, "state")
	codex := filepath.Join(root, "client")
	m := New(state, codex, filepath.Join(root, "oneline-ccc.exe"))
	calls := 0
	m.runner = func(context.Context, string, ...string) (commandResult, error) { calls++; return commandResult{}, nil }
	if info := m.Status(context.Background()); info.Status != "Unavailable" {
		t.Fatal("Windows monitor support incorrectly advertised")
	}
	for _, action := range []string{"install", "start", "stop", "restart", "uninstall"} {
		info, err := m.Action(context.Background(), action)
		if err == nil || info.Status != "Unavailable" {
			t.Fatalf("Windows %s was not refused", action)
		}
	}
	for _, once := range []bool{true, false} {
		if m.Observe(context.Background(), once) == nil {
			t.Fatal("unsupported observer wrote state")
		}
	}
	if calls != 0 {
		t.Fatal("unsupported Windows lifecycle called native runner")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("unsupported Windows observer created filesystem state")
	}
}
