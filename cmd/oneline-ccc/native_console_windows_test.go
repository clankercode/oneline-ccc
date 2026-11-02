//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/windows"
)

var nativeConsoleKernel = windows.NewLazySystemDLL("kernel32.dll")
var nativeConsolePeekPipe = nativeConsoleKernel.NewProc("PeekNamedPipe")
var nativeConsoleUpdateAttribute = nativeConsoleKernel.NewProc("UpdateProcThreadAttribute")

func init() {
	nativeConsolePrepare = func() (func() error, error) {
		// ConPTY attachment and inherited process standard handles are distinct.
		// Match the PTY host's explicit stream attachment using real handles to
		// this child's already-attached console, never AttachConsole/operator IO.
		// https://learn.microsoft.com/windows/console/getstdhandle#remarks
		oldIn, oldOut := os.Stdin, os.Stdout
		oldInput, _ := windows.GetStdHandle(windows.STD_INPUT_HANDLE)
		oldOutput, _ := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
		in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
		if err != nil {
			return nil, fmt.Errorf("open private CONIN$: %w", err)
		}
		out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
		if err != nil {
			_ = in.Close()
			return nil, fmt.Errorf("open private CONOUT$: %w", err)
		}
		rollback := func() error {
			os.Stdin, os.Stdout = oldIn, oldOut
			return errors.Join(windows.SetStdHandle(windows.STD_INPUT_HANDLE, oldInput), windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, oldOutput), in.Close(), out.Close())
		}
		var mode uint32
		if err := windows.GetConsoleMode(windows.Handle(in.Fd()), &mode); err != nil {
			return nil, errors.Join(err, rollback())
		}
		if err := windows.GetConsoleMode(windows.Handle(out.Fd()), &mode); err != nil {
			return nil, errors.Join(err, rollback())
		}
		if err := windows.SetStdHandle(windows.STD_INPUT_HANDLE, windows.Handle(in.Fd())); err != nil {
			return nil, errors.Join(err, rollback())
		}
		if err := windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(out.Fd())); err != nil {
			return nil, errors.Join(err, rollback())
		}
		os.Stdin, os.Stdout = in, out
		return rollback, nil
	}
	nativeConsoleModes = func() (string, error) {
		var input, output uint32
		if err := windows.GetConsoleMode(windows.Handle(os.Stdin.Fd()), &input); err != nil {
			return "", fmt.Errorf("stdin GetConsoleMode: %w", err)
		}
		err := windows.GetConsoleMode(windows.Handle(os.Stdout.Fd()), &output)
		if err != nil {
			return "", fmt.Errorf("stdout GetConsoleMode: %w", err)
		}
		return fmt.Sprintf("input=%x output=%x", input, output), err
	}
	nativeConsoleDiagnostics = func(out io.Writer) {
		for _, stream := range []struct {
			name string
			file *os.File
			id   uint32
		}{
			{"stdin", os.Stdin, windows.STD_INPUT_HANDLE}, {"stdout", os.Stdout, windows.STD_OUTPUT_HANDLE}, {"stderr", os.Stderr, windows.STD_ERROR_HANDLE},
		} {
			h := windows.Handle(stream.file.Fd())
			standard, standardErr := windows.GetStdHandle(stream.id)
			kind, kindErr := windows.GetFileType(h)
			var mode uint32
			modeErr := windows.GetConsoleMode(h, &mode)
			fmt.Fprintf(out, "EARLY %s file=%#x standard=%#x standardError=%v type=%d typeError=%v mode=%#x modeError=%v\n", stream.name, h, standard, standardErr, kind, kindErr, mode, modeErr)
		}
	}
	nativeConsoleRaw = func() error { _, err := term.MakeRaw(os.Stdin.Fd()); return err }
	nativeConsoleAssertRaw = func() error {
		var mode uint32
		if err := windows.GetConsoleMode(windows.Handle(os.Stdin.Fd()), &mode); err != nil {
			return err
		}
		if mode&(windows.ENABLE_LINE_INPUT|windows.ENABLE_ECHO_INPUT|windows.ENABLE_PROCESSED_INPUT) != 0 {
			return errors.New("TUI did not enter native raw mode")
		}
		return nil
	}
	nativeConsoleCooked = func() error {
		var mode uint32
		if err := windows.GetConsoleMode(windows.Handle(os.Stdin.Fd()), &mode); err != nil {
			return err
		}
		const cooked = windows.ENABLE_LINE_INPUT | windows.ENABLE_ECHO_INPUT | windows.ENABLE_PROCESSED_INPUT
		if mode&cooked != cooked {
			return fmt.Errorf("synthetic child received raw console: %x", mode)
		}
		return nil
	}
	nativeConsoleContext = func(parent context.Context) (context.Context, func(), error) {
		if os.Getenv("ONELINE_A7_FIXTURE") != "cancel" {
			return parent, func() {}, nil
		}
		name, err := windows.UTF16PtrFromString(os.Getenv("ONELINE_A7_EVENT"))
		if err != nil {
			return nil, nil, err
		}
		event, err := windows.OpenEvent(windows.SYNCHRONIZE|windows.EVENT_MODIFY_STATE, false, name)
		if err != nil {
			return nil, nil, err
		}
		ctx, cancel := context.WithCancel(parent)
		joined := make(chan error, 1)
		go func() {
			_, err := windows.WaitForSingleObject(event, windows.INFINITE)
			cancel()
			joined <- err
		}()
		return ctx, func() {
			// Private external cancellation event, not the operator's console.
			if err := windows.SetEvent(event); err != nil {
				panic(err)
			}
			if err := <-joined; err != nil {
				panic(err)
			}
			cancel()
			if err := windows.CloseHandle(event); err != nil {
				panic(err)
			}
		}, nil
	}
}

type nativeWindowsConPTY struct {
	inputRead, inputWrite, outputRead, outputWrite windows.Handle
	console, job, process, thread, event           windows.Handle
	done, closed                                   bool
	exitErr                                        error
	tail                                           []byte
}

func startNativeWindowsConPTY(t *testing.T, root, scenario, noColor string) *nativeWindowsConPTY {
	t.Helper()
	p := &nativeWindowsConPTY{}
	t.Cleanup(func() {
		if err := p.close(); err != nil {
			t.Error(err)
		}
	})
	// Missing ConPTY on a supported native runner is a failure, never a skip.
	if err := nativeConsoleKernel.NewProc("CreatePseudoConsole").Find(); err != nil {
		t.Fatal("A7 test host needs native ConPTY (this is not an application OS floor):", err)
	}
	if err := windows.CreatePipe(&p.inputRead, &p.inputWrite, nil, 65536); err != nil {
		t.Fatal(err)
	}
	if err := windows.CreatePipe(&p.outputRead, &p.outputWrite, nil, 65536); err != nil {
		t.Fatal(err)
	}
	if err := windows.CreatePseudoConsole(windows.Coord{X: 120, Y: 40}, p.inputRead, p.outputWrite, 0, &p.console); err != nil {
		t.Fatal("CreatePseudoConsole:", err)
	}
	var err error
	p.job, err = windows.CreateJobObject(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(p.job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		t.Fatal(err)
	}
	env := nativeConsoleEnv(root, scenario, noColor)
	// Only OS installation metadata is added; no operator environment lookup.
	windowsDir, err := windows.GetWindowsDirectory()
	if err != nil {
		t.Fatal(err)
	}
	env = append(env, "SystemRoot="+windowsDir)
	if scenario == "cancel" {
		eventName := fmt.Sprintf("Local\\oneline-a7-%d-%s", os.Getpid(), filepath.Base(root))
		name, err := windows.UTF16PtrFromString(eventName)
		if err != nil {
			t.Fatal(err)
		}
		p.event, err = windows.CreateEvent(nil, 1, 0, name)
		if err != nil {
			t.Fatal(err)
		}
		env = append(env, "ONELINE_A7_EVENT="+eventName)
	}
	sort.Slice(env, func(i, j int) bool { return strings.ToUpper(env[i]) < strings.ToUpper(env[j]) })
	block := utf16.Encode([]rune(strings.Join(env, "\x00") + "\x00\x00"))
	attributes, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		t.Fatal(err)
	}
	defer attributes.Delete()
	// PSEUDOCONSOLE's value is HPCON itself (not &HPCON). Call the Windows
	// entry directly so an opaque handle need not become a Go unsafe.Pointer.
	ok, _, callErr := nativeConsoleUpdateAttribute.Call(uintptr(unsafe.Pointer(attributes.List())), 0,
		windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, uintptr(p.console), unsafe.Sizeof(p.console), 0, 0)
	if ok == 0 {
		t.Fatal("pseudoconsole startup attribute:", callErr)
	}
	startup := windows.StartupInfoEx{ProcThreadAttributeList: attributes.List()}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	app, _ := windows.UTF16PtrFromString(executable)
	command, _ := windows.UTF16PtrFromString(windows.ComposeCommandLine([]string{executable, "-test.run=^TestNativeConsoleFixture$"}))
	directory, _ := windows.UTF16PtrFromString(root)
	var process windows.ProcessInformation
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_SUSPENDED)
	if err := windows.CreateProcess(app, command, nil, nil, false, flags, &block[0], directory, &startup.StartupInfo, &process); err != nil {
		t.Fatal("native fixture CreateProcess:", err)
	}
	runtime.KeepAlive(block)
	p.process, p.thread = process.Process, process.Thread
	// Microsoft documents retaining these communication endpoints through
	// CreateProcess, then freeing the host references (not before child setup):
	// https://learn.microsoft.com/windows/console/creating-a-pseudoconsole-session
	if err := nativeWindowsCloseHandle(&p.inputRead); err != nil {
		t.Fatal(err)
	}
	if err := nativeWindowsCloseHandle(&p.outputWrite); err != nil {
		t.Fatal(err)
	}
	// Assign before ResumeThread: descendants cannot escape between spawn and
	// Job ownership. Close/terminate the Job even after the root process exits.
	if err := windows.AssignProcessToJobObject(p.job, p.process); err != nil {
		t.Fatal(err)
	}
	if count, err := windows.ResumeThread(p.thread); err != nil || count != 1 {
		t.Fatalf("native fixture resume count=%d err=%v", count, err)
	}
	if err := nativeWindowsCloseHandle(&p.thread); err != nil {
		t.Fatal(err)
	}
	return p
}

func nativeWindowsCloseHandle(handle *windows.Handle) error {
	if *handle == 0 {
		return nil
	}
	err := windows.CloseHandle(*handle)
	*handle = 0
	return err
}

func (p *nativeWindowsConPTY) read(wait time.Duration) ([]byte, error) {
	deadline := time.Now().Add(wait)
	for {
		var available uint32
		ok, _, err := nativeConsolePeekPipe.Call(uintptr(p.outputRead), 0, 0, 0, uintptr(unsafe.Pointer(&available)), 0)
		if ok == 0 {
			if errors.Is(err, windows.ERROR_BROKEN_PIPE) {
				return nil, io.EOF
			}
			return nil, err
		}
		if available != 0 {
			// Sole consumer, read at most the bytes known to be ready. No blocked
			// ReadFile goroutine survives cancellation or console closure.
			buf := make([]byte, min(available, 8192))
			var n uint32
			err := windows.ReadFile(p.outputRead, buf, &n, nil)
			return buf[:n], err
		}
		if time.Now().After(deadline) {
			return nil, nil
		}
		// Poll cadence affects CPU only. Assertions use content and deadlines.
		timer := time.NewTimer(min(5*time.Millisecond, time.Until(deadline)))
		<-timer.C
	}
}
func (p *nativeWindowsConPTY) write(keys string) error {
	// Writes stay below the private 64KiB input pipe. The fixture consumes them
	// concurrently; the supported native test command also has a process timeout.
	var n uint32
	err := windows.WriteFile(p.inputWrite, []byte(keys), &n, nil)
	if err == nil && n != uint32(len(keys)) {
		return io.ErrShortWrite
	}
	return err
}
func (p *nativeWindowsConPTY) resize(w, h int) error {
	return windows.ResizePseudoConsole(p.console, windows.Coord{X: int16(w), Y: int16(h)})
}
func (p *nativeWindowsConPTY) cancel() error { return windows.SetEvent(p.event) }
func (p *nativeWindowsConPTY) exited() (bool, error) {
	if !p.done && p.process != 0 {
		status, err := windows.WaitForSingleObject(p.process, 0)
		if err != nil {
			return false, err
		}
		if status == windows.WAIT_OBJECT_0 {
			p.done = true
			var code uint32
			p.exitErr = windows.GetExitCodeProcess(p.process, &code)
			if p.exitErr == nil && code != 0 {
				p.exitErr = fmt.Errorf("native test subprocess exit=%d", code)
			}
		}
	}
	return p.done, p.exitErr
}

type nativeWindowsJobAccounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses     uint32
}

func (p *nativeWindowsConPTY) drained() []byte { return p.tail }
func (p *nativeWindowsConPTY) close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	var result error
	if p.job != 0 {
		result = errors.Join(result, windows.TerminateJobObject(p.job, 1))
	}
	if p.process != 0 {
		if !p.done {
			// Also covers CreateProcess success followed by Job assignment failure.
			_ = windows.TerminateProcess(p.process, 1)
			status, err := windows.WaitForSingleObject(p.process, 5000)
			if err != nil || status != windows.WAIT_OBJECT_0 {
				result = errors.Join(result, fmt.Errorf("native process wait status=%d err=%v", status, err))
			}
			p.done = status == windows.WAIT_OBJECT_0
		}
	}
	if p.console != 0 {
		// ClosePseudoConsole can block writing final output. Close it on one
		// goroutine WHILE draining; never wait for a pipe producer before closing
		// ConPTY. No blocking output reader is created by this fixture.
		joined := make(chan struct{})
		go func() { windows.ClosePseudoConsole(p.console); close(joined) }()
		deadline := time.Now().Add(5 * time.Second)
	drain:
		for {
			data, err := p.read(10 * time.Millisecond)
			p.tail = append(p.tail, data...)
			if err != nil && !errors.Is(err, io.EOF) {
				result = errors.Join(result, err)
			}
			select {
			case <-joined:
				break drain
			default:
			}
			if time.Now().After(deadline) || len(p.tail) > 2<<20 {
				result = errors.Join(result, errors.New("ConPTY close/drain exceeded bound"))
				// Close the consumer to break a stuck producer, then join the
				// closer. This path cannot report success with a detached goroutine.
				result = errors.Join(result, nativeWindowsCloseHandle(&p.outputRead))
				<-joined
				break
			}
		}
		p.console = 0
		if p.outputRead != 0 {
			for {
				data, err := p.read(0)
				p.tail = append(p.tail, data...)
				if err != nil || len(data) == 0 {
					break
				}
			}
		}
	}
	if p.job != 0 {
		deadline := time.Now().Add(5 * time.Second)
		for {
			var accounting nativeWindowsJobAccounting
			err := windows.QueryInformationJobObject(p.job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil)
			if err != nil || accounting.ActiveProcesses == 0 {
				result = errors.Join(result, err)
				break
			}
			if time.Now().After(deadline) {
				result = errors.Join(result, errors.New("owned Job still has active descendants after root exit"))
				break
			}
			timer := time.NewTimer(time.Millisecond)
			<-timer.C
		}
	}
	for _, handle := range []*windows.Handle{&p.thread, &p.process, &p.event, &p.inputRead, &p.inputWrite, &p.outputRead, &p.outputWrite, &p.job} {
		result = errors.Join(result, nativeWindowsCloseHandle(handle))
	}
	return result
}

func TestNativeConsoleWindowsConPTY(t *testing.T) {
	for _, tc := range []struct{ scenario, noColor string }{{"interactive", ""}, {"interactive", "1"}, {"cancel", "1"}, {"handoff", "1"}, {"plain", "1"}} {
		t.Run(tc.scenario+"/NO_COLOR="+tc.noColor, func(t *testing.T) {
			root := nativeConsoleRoot(t)
			host := startNativeWindowsConPTY(t, root, tc.scenario, tc.noColor)
			nativeConsoleJourney(t, nativeConsoleDriverFor(t, host, root), tc.scenario)
		})
	}
}
