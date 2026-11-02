package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/gofrs/flock"
	"golang.org/x/sys/unix"
)

func TestReviewApplyRechecksRevocationAfterTransactionLock(t *testing.T) {
	e, _, _, o := fixture(t, false)
	old := plan(t, e, core.Desired{Enabled: true, Profile: profile()})
	root := filepath.Join(o.StateRoot, "transactions")
	if err := configtxn.PrivateDir(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "coordinator.lock")
	if err := configtxn.PrivateFile(path, nil); err != nil {
		t.Fatal(err)
	}
	held := flock.New(path, flock.SetPermissions(0600))
	if err := held.Lock(); err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	if _, err = unix.InotifyAddWatch(fd, path, unix.IN_OPEN); err != nil {
		t.Fatal(err)
	}
	type applied struct {
		r   core.Receipt
		err error
	}
	done := make(chan applied, 1)
	go func() { r, err := e.Apply(context.Background(), old.ID, old.Revision); done <- applied{r, err} }()
	polls := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	if n, err := unix.Poll(polls, 2000); err != nil || n == 0 {
		t.Fatalf("Apply did not reach held transaction lock: %v", err)
	}
	buf := make([]byte, 4096)
	if _, err := unix.Read(fd, buf); err != nil {
		t.Fatal(err)
	}
	// The watched lock is opened only after Apply's epoch check; it cannot yet be acquired.
	newerProfile := profile()
	newerProfile.Model = "newer-reviewed-model"
	newer := plan(t, e, core.Desired{Enabled: true, Profile: newerProfile})
	if newer.ID == old.ID {
		t.Fatal("review IDs collided")
	}
	if err := held.Unlock(); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-done:
		data, _ := os.ReadFile(filepath.Join(o.CodexHome, "config.toml"))
		t.Logf("old result=%s err=%v old-bytes-committed=%t", result.r.Outcome, result.err, bytes.Contains(data, []byte("fixture-model")))
		if result.err == nil {
			t.Error("superseded review committed after waiting on the transaction lock")
		}
		if _, err := os.Stat(filepath.Join(o.CodexHome, "config.toml")); !os.IsNotExist(err) {
			t.Error("revoked Apply changed native config")
		}
		if _, err := os.Stat(filepath.Join(o.StateRoot, "settings.json")); !os.IsNotExist(err) {
			t.Error("revoked Apply changed protected state")
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.Name() != "coordinator.lock" {
				t.Errorf("revoked Apply allocated transaction artifact %s", entry.Name())
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Apply did not finish")
	}
}

type reviewHeldCredentials struct {
	*fixtureCredentials
	entered, release chan struct{}
}

func (c *reviewHeldCredentials) Resolve(ctx context.Context, ref string) (string, string, error) {
	close(c.entered)
	select {
	case <-c.release:
	case <-ctx.Done():
		return "", "Missing", ctx.Err()
	}
	return c.fixtureCredentials.Resolve(ctx, ref)
}
func TestReviewDiscoveryRechecksStateAfterCredentialWait(t *testing.T) {
	e, _, credentials, o := fixture(t, false)
	held := &reviewHeldCredentials{credentials, make(chan struct{}), make(chan struct{})}
	e.deps.credentials = held
	var requests atomic.Int32
	e.deps.discover = func(context.Context, string, string) ([]core.ModelInfo, error) {
		requests.Add(1)
		return []core.ModelInfo{{ID: "fixture-inventory"}}, nil
	}
	e.deps.enrich = func(_ context.Context, m []core.ModelInfo) ([]core.ModelInfo, error) { return m, nil }
	type response struct {
		models []core.ModelInfo
		err    error
	}
	done := make(chan response, 1)
	go func() { m, err := e.Discover(context.Background(), "codex"); done <- response{m, err} }()
	select {
	case <-held.entered:
	case <-time.After(time.Second):
		t.Fatal("resolver not entered")
	}
	if err := configtxn.PrivateFile(filepath.Join(o.StateRoot, "settings.json"), []byte("invalid synthetic state")); err != nil {
		t.Fatal(err)
	}
	close(held.release)
	select {
	case result := <-done:
		t.Logf("requests=%d returned-models=%d err=%v", requests.Load(), len(result.models), result.err)
		if requests.Load() != 0 || result.err == nil || len(result.models) != 0 {
			t.Error("discovery dispatched credentials and returned inventory after protected state became invalid")
		}
	case <-time.After(time.Second):
		t.Fatal("discovery did not finish")
	}
}
func TestReviewCommandRechecksConfigAfterCredentialWait(t *testing.T) {
	e, _, credentials, o := fixture(t, false)
	path := filepath.Join(o.CodexHome, "config.toml")
	if err := configtxn.PrivateFile(path, []byte("original fixture")); err != nil {
		t.Fatal(err)
	}
	held := &reviewHeldCredentials{credentials, make(chan struct{}), make(chan struct{})}
	e.deps.credentials = held
	done := make(chan error, 1)
	go func() {
		command, err := e.Command(context.Background(), "launch", "codex")
		if command != nil {
			done <- errors.New("stale command returned")
			return
		}
		done <- err
	}()
	select {
	case <-held.entered:
	case <-time.After(time.Second):
		t.Fatal("resolver not entered")
	}
	if err := os.WriteFile(path, []byte("foreign fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	close(held.release)
	select {
	case err := <-done:
		if err == nil || err.Error() == "stale command returned" {
			t.Fatal("credential wait allowed stale launch")
		}
	case <-time.After(time.Second):
		t.Fatal("launch did not finish")
	}
}
