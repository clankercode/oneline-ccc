package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/clankercode/oneline-ccc/internal/core"
)

func useTUI(o flags, d dependencies) bool {
	_, noColor := d.lookup("NO_COLOR")
	term, _ := d.lookup("TERM")
	return o.command == "" && !o.help && !o.showVersion && d.terminal && !o.plain && !noColor && term != "dumb"
}
func tuiRequested(args []string, d dependencies) bool {
	o, err := parse(args)
	return err == nil && useTUI(o, d)
}
func watchSignals(parent context.Context, ui bool) (context.Context, func()) {
	if !ui {
		return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	}
	// Consume the parent's SIGINT without cancelling the foreground session.
	// The owned-child runner hands terminal foreground to the cooked child's
	// group; raw TUI Ctrl+C remains a tea key event.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	ctx, stop := signal.NotifyContext(parent, syscall.SIGTERM)
	return ctx, func() { signal.Stop(interrupts); stop() }
}

type rootBoundBackend struct {
	core.Backend
	root    context.Context
	session *foregroundSession
}

func (b rootBoundBackend) Command(caller context.Context, action, id string) (*exec.Cmd, error) {
	done, err := b.enter()
	if err != nil {
		return nil, err
	}
	defer done()
	// TUI command contexts outlive terminal suspension. Bind them to SIGTERM as
	// well, otherwise terminal execution can block shutdown waiting for a child.
	merged, cancel := context.WithCancel(caller)
	stopRoot := context.AfterFunc(b.root, cancel)
	stopCaller := context.AfterFunc(caller, func() { stopRoot(); cancel() })
	if b.root.Err() != nil {
		cancel()
	}
	command, err := b.Backend.Command(merged, action, id)
	if err != nil || command == nil {
		stopCaller()
		stopRoot()
		cancel()
	}
	// On normal return the TUI cancels caller after the child callback. That
	// removes the root cancellation registration without a polling goroutine.
	return command, err
}

// Enrollment and closure share a mutex: no Add can race the shutdown Wait.
// The session owns every backend await, including read/preparation work, because
// those calls can also have protected cleanup to finish after cancellation.
type foregroundSession struct {
	mu     sync.Mutex
	active sync.WaitGroup
	closed bool
	cancel context.CancelFunc
}

func (b rootBoundBackend) enter() (func(), error) {
	if b.session == nil {
		if err := b.root.Err(); err != nil {
			return nil, err
		}
		return func() {}, nil
	}
	s := b.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, context.Canceled
	}
	if err := b.root.Err(); err != nil {
		return nil, err
	}
	s.active.Add(1)
	return s.active.Done, nil
}
func (s *foregroundSession) shutdown() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	// Native synchronous writes and transaction rollback must reach their own
	// final boundary. Do not put a timeout around this wait and abandon them.
	s.active.Wait()
}
func foregroundCall[T any](b rootBoundBackend, caller context.Context, work func(context.Context) (T, error)) (T, error) {
	done, err := b.enter()
	if err != nil {
		var zero T
		return zero, err
	}
	defer done()
	ctx, cancel := context.WithCancel(caller)
	stop := context.AfterFunc(b.root, cancel)
	defer func() { stop(); cancel() }()
	// AfterFunc is asynchronous; propagate an already observed root cancellation
	// before invoking a backend, rather than waiting for its callback scheduling.
	if b.root.Err() != nil {
		cancel()
	}
	return work(ctx)
}
func (b rootBoundBackend) Inspect(ctx context.Context) (core.Snapshot, error) {
	return foregroundCall(b, ctx, b.Backend.Inspect)
}
func (b rootBoundBackend) Plan(ctx context.Context, id string, d core.Desired) (core.Plan, error) {
	return foregroundCall(b, ctx, func(ctx context.Context) (core.Plan, error) { return b.Backend.Plan(ctx, id, d) })
}
func (b rootBoundBackend) Apply(ctx context.Context, id, revision string) (core.Receipt, error) {
	return foregroundCall(b, ctx, func(ctx context.Context) (core.Receipt, error) { return b.Backend.Apply(ctx, id, revision) })
}
func (b rootBoundBackend) SaveKey(ctx context.Context, key string, persistent bool) error {
	_, err := foregroundCall(b, ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, b.Backend.SaveKey(ctx, key, persistent)
	})
	return err
}
func (b rootBoundBackend) Discover(ctx context.Context, id string) ([]core.ModelInfo, error) {
	return foregroundCall(b, ctx, func(ctx context.Context) ([]core.ModelInfo, error) { return b.Backend.Discover(ctx, id) })
}
func (b rootBoundBackend) Service(ctx context.Context, action string) (core.ServiceInfo, error) {
	return foregroundCall(b, ctx, func(ctx context.Context) (core.ServiceInfo, error) { return b.Backend.Service(ctx, action) })
}
func (b rootBoundBackend) RepairBrief(ctx context.Context) (string, error) {
	return foregroundCall(b, ctx, b.Backend.RepairBrief)
}
