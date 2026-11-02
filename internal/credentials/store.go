// Package credentials resolves explicit session, existing environment and vault
// references. It never persists plaintext or includes provider errors in output.
package credentials

import (
	"context"
	"errors"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/zalando/go-keyring"
)

const service = "oneline-ccc"
const account = "default"
const maxKeyBytes = 2560 // Windows Credential Manager's native blob limit.

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type vault interface {
	Get(service, account string) (string, error)
	Set(service, account, key string) error
}
type nativeVault struct{}

func (nativeVault) Get(service, account string) (string, error) { return keyring.Get(service, account) }
func (nativeVault) Set(service, account, key string) error      { return keyring.Set(service, account, key) }

type unavailableVault struct{}

func (unavailableVault) Get(string, string) (string, error) {
	return "", errors.New("persistent vault unavailable")
}
func (unavailableVault) Set(string, string, string) error {
	return errors.New("persistent vault unavailable")
}

// Manager holds a foreground-session credential; its zero value is safe and
// supports session-only use. No vault or network access occurs in New.
type Manager struct {
	mu            sync.RWMutex
	session       string
	storedSession bool
	vault         vault
	lookup        func(string) (string, bool)
}

func New() *Manager {
	var v vault = unavailableVault{}
	// v0.2.8 uses an uncancellable /usr/bin/security subprocess on Darwin
	// (early failures after Start can skip Wait). Linux Secret Service unlock
	// waits on a prompt signal without a deadline. Do not hide those operations
	// in goroutines: cancellation could then race a late persistent write.
	// Windows uses synchronous native CredRead/CredWrite without subprocesses.
	if runtime.GOOS == "windows" {
		v = nativeVault{}
	}
	return &Manager{vault: v, lookup: os.LookupEnv}
}
func validKey(key string) bool {
	return key != "" && len(key) <= maxKeyBytes && utf8.ValidString(key) && strings.IndexFunc(key, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) < 0
}
func validReference(name string) bool {
	if !envNamePattern.MatchString(name) || essentialEnv(name) {
		return false
	}
	switch strings.ToUpper(name) {
	case "CODEX_PROFILE", "CODEX_HOME", "LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES", "DYLD_LIBRARY_PATH", "NODE_OPTIONS", "NODE_PATH", "BASH_ENV", "ENV", "PYTHONPATH", "PYTHONHOME", "GIT_CONFIG", "GIT_CONFIG_COUNT":
		return false
	}
	return true
}
func (m *Manager) SetSession(key string) error {
	if !validKey(key) {
		return errors.New("enter a nonempty API key without whitespace or control characters (maximum 2560 bytes)")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != key {
		m.storedSession = false
	}
	m.session = key
	return nil
}
func (m *Manager) Save(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return errors.New("credential save canceled; enter a session key to continue")
	}
	if !validKey(key) {
		return errors.New("enter a nonempty API key without whitespace or control characters (maximum 2560 bytes)")
	}
	if m.vault == nil {
		return errors.New("persistent vault unavailable; use a session key")
	}
	if _, disabled := m.vault.(unavailableVault); disabled {
		return errors.New("persistent vault disabled on this platform: bounded native access unavailable; use a session key")
	}
	if err := m.vault.Set(service, account, key); err != nil {
		return errors.New("credential vault could not save the key; unlock the vault or use a session key")
	}
	// A successful explicit entry selects this exact key for the foreground
	// session too. Publish only after the synchronous vault write succeeds.
	m.mu.Lock()
	m.session = key
	m.storedSession = true
	m.mu.Unlock()
	return nil
}
func (m *Manager) Resolve(ctx context.Context, envName string) (key, status string, err error) {
	if ctx.Err() != nil {
		return "", "Locked", errors.New("credential lookup canceled; retry or enter a session key")
	}
	if !validReference(envName) {
		return "", "Missing", errors.New("select a valid credential environment reference")
	}
	m.mu.RLock()
	key = m.session
	stored := m.storedSession
	m.mu.RUnlock()
	if key != "" {
		if stored {
			return key, "Stored", nil
		}
		return key, "Session only", nil
	}
	lookup := m.lookup
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if value, ok := lookup(envName); ok && value != "" {
		if !validKey(value) {
			return "", "Missing", errors.New("credential environment reference is invalid; enter a session key")
		}
		return value, "Stored", nil
	}
	if m.vault == nil {
		return "", "Locked", errors.New("persistent vault unavailable; enter a session key")
	}
	if _, disabled := m.vault.(unavailableVault); disabled {
		return "", "Locked", errors.New("persistent vault disabled on this platform: bounded native access unavailable; enter a session key")
	}
	key, err = m.vault.Get(service, account)
	if errors.Is(err, keyring.ErrNotFound) || (err == nil && key == "") {
		return "", "Missing", errors.New("credential missing; save a key or enter a session key")
	}
	if err != nil || !validKey(key) {
		return "", "Locked", errors.New("credential vault unavailable or locked; unlock it or enter a session key")
	}
	if ctx.Err() != nil {
		return "", "Locked", errors.New("credential lookup canceled; retry or enter a session key")
	}
	return key, "Stored", nil
}
func (m *Manager) Status(ctx context.Context, envName string) string {
	_, status, _ := m.Resolve(ctx, envName)
	return status
}
