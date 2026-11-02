//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/sys/unix"
)

// Only the descriptor presented at the selected setup boundary is invalid.
// The actual PTY stays open through Run and all cleanup observations. Subsequent
// Fd calls expose the original descriptor, including term.Restore's call.
type startupFaultFile struct {
	*os.File
	calls atomic.Int32
	fault func(int32) bool
}

func (f *startupFaultFile) Fd() uintptr {
	if f.fault(f.calls.Add(1)) {
		return ^uintptr(0)
	}
	return f.File.Fd()
}

type startupTerminalModel struct {
	init  func() tea.Cmd
	views int
}

type startupReadErrorFile struct {
	*os.File
	err error
}

func (f startupReadErrorFile) Read([]byte) (int, error) { return 0, f.err }

func (m *startupTerminalModel) Init() tea.Cmd                       { return m.init() }
func (m *startupTerminalModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *startupTerminalModel) View() tea.View {
	m.views++
	return tea.NewView("startup ownership fixture")
}

func TestStartupTerminalFailureRestoresModes(t *testing.T) {
	for _, mode := range []string{"size", "size-cancel", "partial-setup-panic", "input-init", "input-init-cancel", "model-init-panic", "read-error", "canceled-before-start", "quit"} {
		t.Run(mode, func(t *testing.T) {
			if strings.HasPrefix(mode, "input-init") && os.Getenv("ONELINE_A6_INPUT_ERROR_CHILD") != mode {
				// Pinned cancelreader allocates private wake-pipe descriptors
				// before EpollCtl, and does not close them if EpollCtl fails.
				// Isolate this inherited constructor-error leak; assertions below
				// still observe modes and goroutines before fixture process exit.
				cmd := exec.Command(os.Args[0], "-test.run=^TestStartupTerminalFailureRestoresModes$/^"+mode+"$", "-test.timeout=15s", "-test.v")
				cmd.Env = []string{"ONELINE_A6_INPUT_ERROR_CHILD=" + mode, "TERM=xterm-256color", "GOMAXPROCS=1"}
				out, err := cmd.CombinedOutput() // starts, waits and reaps this fixture
				if err != nil {
					t.Fatalf("input-init fixture: %v\n%s", err, out)
				}
				t.Logf("reaped input-init fixture:\n%s", out)
				return
			}
			baseline := foregroundInputStacks()
			master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			if err := unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
				t.Fatal(err)
			}
			n, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
			if err != nil {
				t.Fatal(err)
			}
			slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer slave.Close()
			before, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rawObserved := false
			observeRaw := func() {
				state, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
				if err != nil || reflect.DeepEqual(before, state) || state.Lflag&(unix.ICANON|unix.ECHO) != 0 {
					t.Errorf("fault boundary was not reached in actual raw mode: %v", err)
				}
				rawObserved = true
				if strings.HasSuffix(mode, "-cancel") {
					cancel()
				}
			}
			var input io.Reader = slave
			var output io.Writer = io.Discard
			readError := errors.New("A6 runtime input failure")
			switch {
			case mode == "read-error":
				input = startupReadErrorFile{slave, readError}
			case strings.HasPrefix(mode, "size"):
				output = &startupFaultFile{File: slave, fault: func(n int32) bool {
					if n == 2 {
						observeRaw()
						return true
					}
					return false
				}}
			case mode == "partial-setup-panic":
				output = &startupFaultFile{File: slave, fault: func(n int32) bool {
					if n == 1 {
						observeRaw()
						panic("A6 partial setup")
					}
					return false
				}}
			case strings.HasPrefix(mode, "input-init"):
				input = &startupFaultFile{File: slave, fault: func(n int32) bool {
					// Linux cancelreader's two EpollCtl descriptor calls follow
					// IsTerminal and MakeRaw. No raw reader has started yet.
					if n == 3 {
						observeRaw()
					}
					return n == 3 || n == 4
				}}
			}
			model := &startupTerminalModel{init: func() tea.Cmd {
				observeRaw()
				if mode == "model-init-panic" {
					panic("A6 model Init")
				}
				if mode == "read-error" {
					return func() tea.Msg {
						_, err := io.WriteString(master, "a") // wake the real epoll reader
						if err != nil {
							t.Error(err)
						}
						return nil
					}
				}
				return tea.Quit
			}}
			if mode == "canceled-before-start" {
				cancel()
			}
			err = runForegroundProgram(ctx, model, tea.WithInput(input), tea.WithOutput(output))
			if mode == "quit" {
				if err != nil || model.views < 2 {
					t.Errorf("graceful rendering lost: views=%d err=%v", model.views, err)
				}
			} else if mode == "canceled-before-start" {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("cancellation lost: %v", err)
				}
			} else if mode == "read-error" {
				if !errors.Is(err, readError) || !errors.Is(err, tea.ErrProgramKilled) || model.views == 0 {
					t.Errorf("runtime failure lost: views=%d err=%v", model.views, err)
				}
			} else {
				if !rawObserved {
					t.Error("fault not exercised after raw mode")
				}
				if err == nil {
					t.Error("startup error lost")
				}
				if strings.Contains(mode, "panic") && !errors.Is(err, tea.ErrProgramPanic) {
					t.Errorf("panic identity lost: %v", err)
				}
				if strings.HasPrefix(mode, "size") && !errors.Is(err, unix.EBADF) {
					t.Errorf("size error identity lost: %v", err)
				}
				if strings.HasPrefix(mode, "input-init") && !strings.Contains(err.Error(), "could not create cancelable reader") {
					t.Errorf("reader error lost: %v", err)
				}
				if strings.HasSuffix(mode, "-cancel") && !errors.Is(err, context.Canceled) {
					t.Errorf("simultaneous cancellation lost: %v", err)
				}
				if model.views != 0 {
					t.Errorf("rendered view after startup failure: %d", model.views)
				}
			}
			after, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Errorf("Run returned with terminal modes changed: before=%+v after=%+v err=%v", before, after, err)
			}
			for id, stack := range foregroundInputStacks() {
				if _, existed := baseline[id]; !existed {
					t.Errorf("Run returned before input completion:\n%s", stack)
				}
			}
		})
	}
}
