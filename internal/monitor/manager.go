// Package monitor implements an optional, read-only foreground or user-session
// observer. Constructors do not inspect configs, activate services or write state.
package monitor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	errUnsafe      = errors.New("observer refused unsafe, foreign or changed state; preserve it and reconcile locally")
	errUnavailable = errors.New("observer backend unavailable; use foreground observation where protected filesystem support is available")
	errIO          = errors.New("protected observer operation failed; no client configuration was changed")
	errUncertain   = errors.New("observer lifecycle outcome is uncertain; inspect native user service state before reconciliation; no automatic retry")
)

type commandResult struct {
	Code   int
	Output string
}
type commandRunner func(context.Context, string, ...string) (commandResult, error)

type Manager struct {
	stateRoot, codexHome, executable string
	home, os, uid                    string
	runner                           commandRunner
	interval                         time.Duration
	now                              func() time.Time
}

func New(stateRoot, codexHome, executable string) *Manager {
	home, _ := os.UserHomeDir()
	return &Manager{stateRoot: stateRoot, codexHome: codexHome, executable: executable, home: home, os: runtime.GOOS, uid: currentUID(), runner: runCommand, interval: 30 * time.Second, now: time.Now}
}
func safePath(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && utf8.ValidString(path) && strings.IndexFunc(path, unicode.IsControl) < 0
}
func inside(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
func (m *Manager) validRoots() bool {
	return safePath(m.stateRoot) && safePath(m.codexHome) && !inside(m.stateRoot, m.codexHome) && !inside(m.codexHome, m.stateRoot)
}
func (m *Manager) root() string { return filepath.Join(m.stateRoot, "monitor") }
func (m *Manager) clock() time.Time {
	if m.now != nil {
		return m.now().UTC()
	}
	return time.Now().UTC()
}
func (m *Manager) argv() []string {
	return []string{m.executable, "--state-dir", m.stateRoot, "--codex-home", m.codexHome, "observe"}
}
