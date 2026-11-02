package tea

import (
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type startupLifecycleModel struct {
	init  func() Cmd
	views int
}

func (m *startupLifecycleModel) Init() Cmd               { return m.init() }
func (m *startupLifecycleModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m *startupLifecycleModel) View() View              { m.views++; return NewView("startup fixture") }

type startupLifecycleRenderer struct {
	nilRenderer
	starts, closes int
	resizePanic    bool
	resizeGate     func()
}

func (r *startupLifecycleRenderer) start()       { r.starts++ }
func (r *startupLifecycleRenderer) close() error { r.closes++; return nil }
func (r *startupLifecycleRenderer) resize(int, int) {
	if r.resizeGate != nil {
		r.resizeGate()
	}
	if r.resizePanic {
		panic("A6 setup resize")
	}
}

func TestStartupKillWaitsForInitialization(t *testing.T) {
	entered, resume := make(chan struct{}), make(chan struct{})
	m := &startupLifecycleModel{init: func() Cmd { t.Error("initialized model after Kill in terminal setup"); return nil }}
	p := NewProgram(m, WithInput(strings.NewReader("")), WithOutput(io.Discard), WithoutSignalHandler())
	r := &startupLifecycleRenderer{resizeGate: func() { close(entered); <-resume }}
	p.renderer = r
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	<-entered
	killed := make(chan struct{})
	go func() { p.Kill(); close(killed) }()
	<-p.ctx.Done() // cancellation must precede waiting for initialization
	// Mutex waits are not durable blocks in synctest. Wait for the
	// concrete teardown lock stack or premature Kill completion instead.
	deadline := time.After(3 * time.Second)
waitForKill:
	for {
		select {
		case <-killed:
			t.Error("Kill returned while initialization still owned the terminal")
			break waitForKill
		case <-deadline:
			t.Error("Kill never reached the terminal acquisition barrier")
			break waitForKill
		default:
		}
		buf := make([]byte, 64<<10)
		stacks := string(buf[:runtime.Stack(buf, true)])
		for _, stack := range strings.Split(stacks, "\n\n") {
			if strings.Contains(stack, "(*Program).shutdown.func1(") && strings.Contains(stack, "sync.Mutex.Lock") {
				break waitForKill
			}
		}
		runtime.Gosched()
	}
	close(resume)
	if err := <-done; !errors.Is(err, ErrProgramKilled) || errors.Is(err, context.Canceled) {
		t.Errorf("wrong internal Kill identity: %v", err)
	}
	<-killed
	if r.starts != 0 || r.closes != 0 {
		t.Errorf("renderer lifetime: starts=%d closes=%d", r.starts, r.closes)
	}
	if m.views != 0 {
		t.Errorf("rendered after Kill during initialization: %d", m.views)
	}
}

func TestStartupKillFromModelInit(t *testing.T) {
	var p *Program
	m := &startupLifecycleModel{init: func() Cmd { p.Kill(); return nil }}
	p = NewProgram(m, WithInput(strings.NewReader("")), WithOutput(io.Discard), WithoutSignalHandler())
	r := &startupLifecycleRenderer{}
	p.renderer = r
	_, err := p.Run()
	if !errors.Is(err, ErrProgramKilled) || m.views != 0 || r.starts != 1 || r.closes != 1 {
		t.Errorf("Init Kill lifetime: err=%v views=%d starts=%d closes=%d", err, m.views, r.starts, r.closes)
	}
}

func TestStartupDoesNotStopUnstartedRenderer(t *testing.T) {
	m := &startupLifecycleModel{init: func() Cmd { t.Error("model initialized after setup failure"); return nil }}
	p := NewProgram(m, WithInput(strings.NewReader("")), WithOutput(io.Discard), WithoutSignalHandler())
	r := &startupLifecycleRenderer{resizePanic: true}
	p.renderer = r
	_, err := p.Run()
	if !errors.Is(err, ErrProgramPanic) {
		t.Errorf("wrong setup panic identity: %v", err)
	}
	if r.starts != 0 || r.closes != 0 || m.views != 0 {
		t.Errorf("unstarted renderer used: %+v, views=%d", r, m.views)
	}
	p.Kill() // repeated forced cleanup is a no-op
	p.Wait()
}

func TestStartupCanceledBeforeAcquisition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := &startupLifecycleModel{init: func() Cmd { t.Error("initialized canceled program"); return nil }}
	p := NewProgram(m, WithContext(ctx), WithInput(strings.NewReader("")), WithOutput(io.Discard), WithoutSignalHandler())
	r := &startupLifecycleRenderer{}
	p.renderer = r
	_, err := p.Run()
	if !errors.Is(err, ErrProgramKilled) || !errors.Is(err, context.Canceled) {
		t.Errorf("root cancellation identity lost: %v", err)
	}
	if r.starts != 0 || r.closes != 0 || m.views != 0 {
		t.Errorf("acquired canceled program: %+v, views=%d", r, m.views)
	}
}

func TestStartupPanicJoinsInputBeforeClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := lifetimeCancelReader{make(chan struct{}), make(chan struct{})}
		var p *Program
		m := &startupLifecycleModel{init: func() Cmd {
			// Retire the real EOF reader before installing a deterministic
			// completion barrier. This assertion complements the real PTY test.
			p.closeInputReader()
			p.cancelReader = r
			p.readLoopDone = make(chan struct{})
			panic("A6 initialized-input panic")
		}}
		p = NewProgram(m, WithInput(strings.NewReader("")), WithOutput(io.Discard), WithoutSignalHandler())
		done := make(chan error, 1)
		go func() { _, err := p.Run(); done <- err }()
		<-r.canceled
		synctest.Wait()
		select {
		case <-r.closed:
			t.Error("panic closed input before completion")
		default:
		}
		select {
		case <-done:
			t.Error("panic returned before input completion")
		default:
		}
		close(p.readLoopDone)
		if err := <-done; !errors.Is(err, ErrProgramPanic) {
			t.Errorf("panic lost: %v", err)
		}
		<-r.closed
	})
}

type a6ReviewCancelRenderer struct {
	startupLifecycleRenderer
	cancel context.CancelFunc
}

func (r *a6ReviewCancelRenderer) start() {
	r.startupLifecycleRenderer.start()
	r.cancel()
}

func TestA6ReviewCanceledDuringRendererStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	m := &startupLifecycleModel{init: func() Cmd {
		calls++
		return nil
	}}
	p := NewProgram(m, WithContext(ctx),
		WithInput(strings.NewReader("")), WithOutput(io.Discard),
		WithoutSignalHandler())
	r := &a6ReviewCancelRenderer{cancel: cancel}
	p.renderer = r
	_, err := p.Run()
	if !errors.Is(err, ErrProgramKilled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("error identity: %v", err)
	}
	if calls != 0 {
		t.Errorf("Model.Init ran %d time(s) after renderer startup canceled the root context; views=%d starts=%d closes=%d",
			calls, m.views, r.starts, r.closes)
	}
	p.Kill()
	p.Kill()
	p.Wait()
	if m.views != 0 || r.starts != 1 || r.closes != 1 {
		t.Errorf("canceled startup lifetime: views=%d starts=%d closes=%d", m.views, r.starts, r.closes)
	}
}
