package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/tui"
)

type foregroundMutation struct {
	core.Backend
	entered chan context.Context
	release chan struct{}
	safe    chan struct{}
	calls   atomic.Int32
}

func (f *foregroundMutation) Inspect(context.Context) (core.Snapshot, error) {
	p := core.Defaults()
	p.BaseURL = "https://fixture.invalid/v1"
	p.CapacityConfirmed = true
	return core.Snapshot{Clients: []core.Client{{ID: "codex", Name: "Codex", Installed: true, Supported: true, Profile: p}}}, nil
}
func (f *foregroundMutation) Plan(_ context.Context, id string, d core.Desired) (core.Plan, error) {
	return core.Plan{ID: "fixture-plan", Revision: "fixture-revision", ClientID: id, Desired: d, Changes: []core.Change{{Field: "Management", Before: "false", After: "true"}}}, nil
}
func (f *foregroundMutation) mutate(ctx context.Context) error {
	f.calls.Add(1)
	f.entered <- ctx
	<-f.release
	close(f.safe)
	return ctx.Err()
}
func (f *foregroundMutation) Apply(ctx context.Context, _, _ string) (core.Receipt, error) {
	err := f.mutate(ctx)
	return core.Receipt{ID: "fixture-receipt", Outcome: "Cancelled"}, err
}
func (f *foregroundMutation) SaveKey(ctx context.Context, _ string, persistent bool) error {
	if !persistent {
		return errors.New("fixture requires persistence")
	}
	return f.mutate(ctx)
}
func (f *foregroundMutation) Service(ctx context.Context, _ string) (core.ServiceInfo, error) {
	err := f.mutate(ctx)
	return core.ServiceInfo{Status: "Unknown"}, err
}

type preparedForeground struct {
	tea.Model
	first tea.Cmd
}

func (m preparedForeground) Init() tea.Cmd { return m.first }

func prepareForeground(t *testing.T, m tea.Model, kind string) (tea.Model, tea.Cmd) {
	t.Helper()
	m, _ = m.Update(m.Init()())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	key := func(code rune, text string, mod tea.KeyMod) tea.Cmd {
		var cmd tea.Cmd
		m, cmd = m.Update(tea.KeyPressMsg{Code: code, Text: text, Mod: mod})
		return cmd
	}
	palette := func(label string) tea.Cmd {
		key('p', "", tea.ModCtrl)
		m, _ = m.Update(tea.PasteMsg{Content: label})
		return key(tea.KeyEnter, "", 0)
	}
	var work tea.Cmd
	switch kind {
	case "apply":
		key(tea.KeyF6, "", 0) // Nav -> Clients, independent of tab count.
		key(tea.KeySpace, " ", 0)
		plan := palette("Review selected client draft")
		if plan == nil {
			t.Fatal("missing plan")
		}
		m, _ = m.Update(plan())
		key(tea.KeyDown, "", 0)
		for i := 0; i < 80; i++ {
			key(tea.KeyEnter, "", 0)
		}
		key(tea.KeyDown, "", 0)
		work = key(tea.KeyEnter, "", 0)
	case "key":
		palette("Gateway")
		key(tea.KeyTab, "", 0) // Endpoint form -> key action; arrows remain local.
		key(tea.KeyEnter, "", 0)
		if !strings.Contains(m.View().Content, "API key: [hidden]") {
			t.Fatalf("key operation did not reach the hidden key editor: %s", m.View().Content)
		}
		m, _ = m.Update(tea.PasteMsg{Content: "synthetic-fixture-key"})
		key(tea.KeyTab, "", 0)
		key(tea.KeySpace, " ", 0)
		key(tea.KeyTab, "", 0)
		work = key(tea.KeyEnter, "", 0)
	case "service":
		palette("Service")
		for i := 0; i < 4; i++ {
			key(tea.KeyDown, "", 0)
		}
		key(tea.KeyEnter, "", 0)
		key(tea.KeyTab, "", 0)
		work = key(tea.KeyEnter, "", 0)
	}
	if work == nil {
		t.Fatalf("missing %s operation: %s", kind, m.View().Content)
	}
	return m, work
}
func TestFinalFixForegroundShutdown(t *testing.T) {
	for _, kind := range []string{"apply", "key", "service"} {
		t.Run(kind, func(t *testing.T) {
			root, cancel := context.WithCancel(context.Background())
			defer cancel()
			f := &foregroundMutation{entered: make(chan context.Context, 1), release: make(chan struct{}), safe: make(chan struct{})}
			var once sync.Once
			release := func() { once.Do(func() { close(f.release) }) }
			defer release()
			done := make(chan error, 1)
			programReturned := make(chan struct{})
			go func() {
				done <- runForeground(root, f, tui.Options{NoColor: true}, func(m tea.Model) error {
					m, cmd := prepareForeground(t, m, kind)
					_, err := tea.NewProgram(preparedForeground{m, cmd}, tea.WithContext(root), tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard), tea.WithoutRenderer(), tea.WithoutSignalHandler()).Run()
					close(programReturned)
					return err
				})
			}()
			var operation context.Context
			select {
			case operation = <-f.entered:
			case <-time.After(3 * time.Second):
				t.Fatal("mutation never entered")
			}
			cancel()
			select {
			case <-programReturned:
			case <-time.After(time.Second):
				t.Fatal("terminal program did not return")
			}
			select {
			case <-operation.Done():
			case <-time.After(time.Second):
				t.Error("root cancellation did not reach mutation")
			}
			select {
			case <-done:
				t.Error("production boundary returned before safe mutation cleanup")
			case <-time.After(50 * time.Millisecond):
			}
			release()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("boundary did not finish cleanup")
			}
			select {
			case <-f.safe:
			default:
				t.Fatal("exit preceded safe final outcome")
			}
		})
	}
}

func TestFinalFixForegroundClosesAdmission(t *testing.T) {
	root, cancel := context.WithCancel(context.Background())
	session := &foregroundSession{cancel: cancel}
	f := &foregroundMutation{entered: make(chan context.Context, 1), release: make(chan struct{}), safe: make(chan struct{})}
	b := rootBoundBackend{Backend: f, root: root, session: session}
	activeDone := make(chan error, 1)
	go func() { _, err := b.Apply(context.Background(), "plan", "revision"); activeDone <- err }()
	operation := <-f.entered
	stopped := make(chan struct{})
	go func() { session.shutdown(); close(stopped) }()
	<-operation.Done() // Cancellation originates in shutdown, after admission closes.
	for _, call := range []func() error{
		func() error { _, err := b.Apply(context.Background(), "plan", "revision"); return err },
		func() error { return b.SaveKey(context.Background(), "fixture-key", true) },
		func() error { _, err := b.Service(context.Background(), "install"); return err },
		func() error { _, err := b.Inspect(context.Background()); return err },
		func() error { _, err := b.Plan(context.Background(), "codex", core.Desired{}); return err },
		func() error { _, err := b.Command(context.Background(), "launch", "codex"); return err },
		func() error { _, err := b.Discover(context.Background(), "codex"); return err },
		func() error { _, err := b.RepairBrief(context.Background()); return err },
	} {
		if !errors.Is(call(), context.Canceled) {
			t.Error("closed session admitted a backend operation")
		}
	}
	if f.calls.Load() != 1 {
		t.Error("late mutation reached backend")
	}
	select {
	case <-stopped:
		t.Error("shutdown abandoned active mutation")
	default:
	}
	close(f.release)
	<-activeDone
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not drain")
	}
	if _, err := b.Apply(context.Background(), "plan", "revision"); !errors.Is(err, context.Canceled) {
		t.Fatal("completed shutdown reopened enrollment")
	}
}
