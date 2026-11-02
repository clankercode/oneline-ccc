//go:build darwin

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

func init() {
	nativeConsoleModes = func() (string, error) {
		m, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%+v", nativeDarwinStableModes(*m)), nil
	}
	nativeConsoleBeforeExit = func() error {
		// The host must inspect its own slave descriptor while this controlling
		// session is alive. Darwin revokes the tty on session-leader exit.
		ready, release := os.NewFile(3, "a7-ready"), os.NewFile(4, "a7-release")
		defer ready.Close()
		defer release.Close()
		if _, err := ready.Write([]byte{'R'}); err != nil {
			return err
		}
		var ack [1]byte
		if _, err := io.ReadFull(release, ack[:]); err != nil {
			return err
		}
		if ack[0] != 'A' {
			return errors.New("invalid parent mode-check acknowledgement")
		}
		return nil
	}
	nativeConsoleRaw = func() error { _, err := term.MakeRaw(os.Stdin.Fd()); return err }
	nativeConsoleAssertRaw = func() error {
		m, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
		if err != nil {
			return err
		}
		if m.Lflag&(unix.ICANON|unix.ECHO|unix.ISIG) != 0 {
			return errors.New("TUI did not enter native raw mode")
		}
		return nil
	}
	nativeConsoleCooked = func() error {
		m, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
		if err != nil {
			return err
		}
		if m.Lflag&(unix.ICANON|unix.ECHO|unix.ISIG) != unix.ICANON|unix.ECHO|unix.ISIG {
			return errors.New("synthetic child received raw terminal")
		}
		group, err := unix.IoctlGetInt(int(os.Stdin.Fd()), unix.TIOCGPGRP)
		if err != nil || group != syscall.Getpgrp() {
			return fmt.Errorf("synthetic child is not foreground: %d: %v", group, err)
		}
		return nil
	}
	nativeConsoleContext = func(ctx context.Context) (context.Context, func(), error) { return ctx, func() {}, nil }
}

// XNU bsd/sys/termios.h calls PENDIN "retype pending input (state)";
// bsd/kern/tty.c sets it when ICANON is restored by TIOCSETA and preserves it
// independently of the requested flags. All other fields remain exact.
// https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/tty.c
func nativeDarwinStableModes(m unix.Termios) unix.Termios { m.Lflag &^= unix.PENDIN; return m }

func nativeDarwinModesRestored(before, after unix.Termios) bool {
	return reflect.DeepEqual(nativeDarwinStableModes(before), nativeDarwinStableModes(after))
}

type nativeDarwinPTY struct {
	master, slave  *os.File
	before         *unix.Termios
	cmd            *exec.Cmd
	wait           chan error
	done, closed   bool
	exitErr        error
	root           string
	tail           []byte
	ready, release *os.File
	modeChecked    bool
	modeErr        error
}

func startNativeDarwinPTY(t *testing.T, root, scenario, noColor string) *nativeDarwinPTY {
	t.Helper()
	p := &nativeDarwinPTY{root: root}
	t.Cleanup(func() {
		if err := p.close(); err != nil {
			t.Error(err)
		}
	})
	// Darwin's grant/unlock/name ioctls are the same primitive used by openpty
	// libraries. Use the already pinned x/sys instead of adding creack/pty.
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal("open native Darwin PTY:", err)
	}
	p.master = os.NewFile(uintptr(fd), "a7-darwin-master")
	for _, request := range []uint{unix.TIOCPTYGRANT, unix.TIOCPTYUNLK} {
		if err := unix.IoctlSetInt(fd, request, 0); err != nil {
			t.Fatal(err)
		}
	}
	var name [128]byte
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0])))
	if errno != 0 {
		t.Fatal("Darwin PTY name:", errno)
	}
	slaveFD, err := unix.Open(unix.ByteSliceToString(name[:]), unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	p.slave = os.NewFile(uintptr(slaveFD), "a7-darwin-slave")
	if err := p.resize(120, 40); err != nil {
		t.Fatal(err)
	}
	p.before, err = unix.IoctlGetTermios(slaveFD, unix.TIOCGETA)
	if err != nil {
		t.Fatal(err)
	}
	readyRead, readyWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	p.ready = readyRead
	defer readyWrite.Close()
	releaseRead, releaseWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	p.release = releaseWrite
	defer releaseRead.Close()
	p.cmd = exec.Command(os.Args[0], "-test.run=^TestNativeConsoleFixture$")
	p.cmd.Dir, p.cmd.Env = root, nativeConsoleEnv(root, scenario, noColor)
	p.cmd.Stdin, p.cmd.Stdout, p.cmd.Stderr = p.slave, p.slave, p.slave
	p.cmd.ExtraFiles = []*os.File{readyWrite, releaseRead}
	p.cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := p.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p.wait = make(chan error, 1)
	// There are no exec pipe goroutines: direct native handles are attached.
	// This is the sole Wait, including all failure/timeout cleanup paths.
	go func() { p.wait <- p.cmd.Wait() }()
	return p
}

func (p *nativeDarwinPTY) read(wait time.Duration) ([]byte, error) {
	fds := []unix.PollFd{{Fd: int32(p.master.Fd()), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, max(0, int((wait+time.Millisecond-1)/time.Millisecond)))
	if errors.Is(err, unix.EINTR) || err == nil && n == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 8192)
	n, err = unix.Read(int(p.master.Fd()), buf)
	if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
		return nil, nil
	}
	if errors.Is(err, unix.EIO) || n == 0 && err == nil {
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (p *nativeDarwinPTY) write(keys string) error {
	// Nonblocking master writes have an explicit bound, including queued input.
	deadline := time.Now().Add(3 * time.Second)
	for len(keys) != 0 {
		n, err := unix.Write(int(p.master.Fd()), []byte(keys))
		if n > 0 {
			keys = keys[n:]
		}
		if err != nil && !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EINTR) {
			return err
		}
		if len(keys) != 0 {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return errors.New("Darwin PTY input deadline")
			}
			_, err = unix.Poll([]unix.PollFd{{Fd: int32(p.master.Fd()), Events: unix.POLLOUT}}, int(remaining/time.Millisecond))
			if err != nil && !errors.Is(err, unix.EINTR) {
				return err
			}
		}
	}
	return nil
}
func (p *nativeDarwinPTY) resize(w, h int) error {
	return unix.IoctlSetWinsize(int(p.slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Col: uint16(w), Row: uint16(h)})
}
func (p *nativeDarwinPTY) cancel() error { return p.cmd.Process.Signal(syscall.SIGTERM) }
func (p *nativeDarwinPTY) exited() (bool, error) {
	if !p.modeChecked && p.ready != nil {
		if err := p.checkModesWhileAlive(); err != nil {
			return false, err
		}
	}
	if !p.done && p.wait != nil {
		select {
		case p.exitErr = <-p.wait:
			p.done = true
		default:
		}
	}
	if p.done && p.exitErr == nil && !p.modeChecked {
		return true, errors.New("successful fixture skipped live parent mode checkpoint")
	}
	return p.done, errors.Join(p.exitErr, p.modeErr)
}

func (p *nativeDarwinPTY) checkModesWhileAlive() error {
	fds := []unix.PollFd{{Fd: int32(p.ready.Fd()), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, 0)
	if errors.Is(err, unix.EINTR) || err == nil && n == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	var ready [1]byte
	_, err = io.ReadFull(p.ready, ready[:])
	if err != nil || ready[0] != 'R' {
		return fmt.Errorf("missing live terminal checkpoint: %v", err)
	}
	p.modeChecked = true
	after, modeErr := unix.IoctlGetTermios(int(p.slave.Fd()), unix.TIOCGETA)
	if modeErr != nil {
		p.modeErr = fmt.Errorf("live slave mode inspection: %w", modeErr)
	} else if !nativeDarwinModesRestored(*p.before, *after) {
		p.modeErr = fmt.Errorf("live slave modes not restored: before=%+v after=%+v", p.before, after)
	}
	// Release even on rejection so the one waiter can join; no failed check can
	// become success when the subprocess subsequently exits normally.
	_, releaseErr := p.release.Write([]byte{'A'})
	p.modeErr = errors.Join(p.modeErr, releaseErr)
	return p.modeErr
}
func (p *nativeDarwinPTY) drained() []byte { return p.tail }

func (p *nativeDarwinPTY) drainOutput() (int, error) {
	if p.master == nil {
		return 0, nil
	}
	data, err := p.read(0)
	remaining := max(0, (2<<20)-len(p.tail))
	p.tail = append(p.tail, data[:min(len(data), remaining)]...)
	if len(data) > remaining {
		err = errors.Join(err, errors.New("Darwin final drain exceeded output bound"))
	}
	return len(data), err
}

func (p *nativeDarwinPTY) close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	var result error
	closeFile := func(f **os.File) {
		if *f != nil {
			result = errors.Join(result, (*f).Close())
			*f = nil
		}
	}
	if p.wait != nil && !p.done {
		// A suspended TUI's synthetic command has its own foreground group.
		// Kill that owned group first and let runProcess wait/reap it on SIGTERM.
		result = errors.Join(result, p.killSyntheticGroup())
		if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
			result = errors.Join(result, fmt.Errorf("signal owned Darwin fixture: %w", err))
		}
		drain := func() error {
			// Production may have returned and be waiting for the independent
			// parent termios check. Service that checkpoint during failure cleanup.
			var modeErr error
			if !p.modeChecked && p.ready != nil {
				modeErr = p.checkModesWhileAlive()
			}
			_, readErr := p.drainOutput()
			if errors.Is(readErr, io.EOF) {
				readErr = nil
			}
			return errors.Join(modeErr, readErr)
		}
		waitPhase := func(limit time.Duration, read func() error) {
			var drainErr error
			p.done, p.exitErr, drainErr = nativeConsoleWaitAndDrain(p.wait, limit, read)
			result = errors.Join(result, drainErr)
		}
		// Darwin can drain a controlling tty during exit. Keep consuming the
		// master instead of waiting for an exit that may itself require reads.
		waitPhase(5*time.Second, drain)
		if !p.done {
			result = errors.Join(result, errors.New("Darwin fixture required forced termination"))
			// Target the owned, unreaped root process, not a numeric group that
			// might no longer contain it. Synthetic groups are checked above.
			if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				result = errors.Join(result, fmt.Errorf("kill owned Darwin fixture: %w", err))
			}
			waitPhase(2*time.Second, drain)
		}
		if !p.done {
			result = errors.Join(result, errors.New("Darwin fixture did not reap after SIGKILL; hanging up private PTY"))
			// Break any remaining private tty/checkpoint dependency before the
			// last bounded wait. These descriptors cannot refer to operator IO.
			closeFile(&p.release)
			closeFile(&p.master)
			waitPhase(2*time.Second, nil)
			if !p.done {
				result = errors.Join(result, errors.New("Darwin fixture waiter still pending after private PTY hangup"))
			}
		}
	} else if p.done && p.exitErr != nil {
		result = errors.Join(result, p.killSyntheticGroup())
	}
	if p.done {
		result = errors.Join(result, p.exitErr)
	}
	if p.done && p.wait != nil && p.exitErr == nil && !p.modeChecked {
		result = errors.Join(result, errors.New("fixture exited without independent live mode check"))
	}
	result = errors.Join(result, p.modeErr)
	if p.master != nil && p.done {
		for {
			n, err := p.drainOutput()
			if err != nil && !errors.Is(err, io.EOF) {
				result = errors.Join(result, err)
			}
			if err != nil || n == 0 {
				break
			}
		}
	}
	if p.slave != nil {
		// Session-leader exit revokes the Darwin tty (XNU kern_exit.c,
		// VNOP_REVOKE). Inspecting/repairing this descriptor now is invalid;
		// the independent parent inspection already ran before release above.
		closeFile(&p.slave)
	}
	closeFile(&p.ready)
	closeFile(&p.release)
	closeFile(&p.master)
	return result
}

func TestNativeConsoleDarwinModeOracle(t *testing.T) {
	before := unix.Termios{Iflag: 11010, Oflag: 3, Cflag: 19200, Lflag: 1483, Ispeed: 9600, Ospeed: 9600}
	pending := before
	pending.Lflag |= unix.PENDIN
	if reflect.DeepEqual(before, pending) || !nativeDarwinModesRestored(before, pending) {
		t.Fatal("native PENDIN state transition should preserve terminal settings")
	}
	for _, tc := range []struct {
		name   string
		change func(*unix.Termios)
	}{
		{"raw", func(m *unix.Termios) { m.Lflag &^= unix.ICANON | unix.ECHO | unix.ISIG }},
		{"canonical", func(m *unix.Termios) { m.Lflag ^= unix.ICANON }},
		{"echo", func(m *unix.Termios) { m.Lflag ^= unix.ECHO }},
		{"signals", func(m *unix.Termios) { m.Lflag ^= unix.ISIG }},
		{"other-local", func(m *unix.Termios) { m.Lflag ^= unix.NOFLSH }},
		{"input", func(m *unix.Termios) { m.Iflag ^= unix.ICRNL }},
		{"output", func(m *unix.Termios) { m.Oflag ^= unix.OPOST }},
		{"control", func(m *unix.Termios) { m.Cflag ^= unix.CLOCAL }},
		{"characters", func(m *unix.Termios) { m.Cc[0] ^= 1 }},
		{"input-speed", func(m *unix.Termios) { m.Ispeed++ }},
		{"output-speed", func(m *unix.Termios) { m.Ospeed++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			after := pending
			tc.change(&after)
			if nativeDarwinModesRestored(before, after) {
				t.Fatal("restoration defect accepted with PENDIN set")
			}
		})
	}
}

func (p *nativeDarwinPTY) killSyntheticGroup() error {
	data, err := os.ReadFile(filepath.Join(p.root, "child-pid"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil || pid <= 1 || p.cmd == nil || p.cmd.Process == nil {
		return errors.New("invalid owned synthetic child identity")
	}
	group, err := unix.Getpgid(pid)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return err
	}
	session, err := unix.Getsid(pid)
	if err != nil {
		return err
	}
	if group != pid || session != p.cmd.Process.Pid {
		return errors.New("synthetic child ownership changed; refusing to signal")
	}
	return syscall.Kill(-pid, syscall.SIGKILL)
}

func TestNativeConsoleDarwinPTY(t *testing.T) {
	for _, tc := range []struct{ scenario, noColor string }{{"interactive", ""}, {"interactive", "1"}, {"cancel", "1"}, {"handoff", "1"}, {"plain", "1"}} {
		t.Run(tc.scenario+"/NO_COLOR="+tc.noColor, func(t *testing.T) {
			root := nativeConsoleRoot(t)
			host := startNativeDarwinPTY(t, root, tc.scenario, tc.noColor)
			nativeConsoleJourney(t, nativeConsoleDriverFor(t, host, root), tc.scenario)
		})
	}
}

func TestNativeConsoleDarwinAbortCleanup(t *testing.T) {
	root := nativeConsoleRoot(t)
	host := startNativeDarwinPTY(t, root, "abort-cleanup", "1")
	p := nativeConsoleDriverFor(t, host, root)
	p.await(t, 0, "Inspected local settings; drafts retained")
	p.await(t, p.send(t, "?"), "Shortcut help | Page")
	// Exercise the same close called by an assertion's t.Cleanup, with the
	// production reader/renderer still active. Do not manufacture a failed test.
	started := time.Now()
	err := host.close()
	if err != nil || !host.done || !host.modeChecked || host.exitErr != nil {
		t.Fatalf("abort cleanup did not join/restoration-check: elapsed=%s done=%t checked=%t exit=%v cleanup=%v", time.Since(started), host.done, host.modeChecked, host.exitErr, err)
	}
	if err := p.consumeDrain(); err != nil {
		t.Fatal(err)
	}
	diag, err := os.ReadFile(filepath.Join(root, "diagnostics"))
	if err != nil || !strings.Contains(string(diag), "A7_COMPLETE code=130 engines=1 inspect=1 forbidden=0 launches=0 modes=restored input=joined") {
		t.Fatalf("abort cleanup lost production completion: read=%v diagnostics=%s", err, diag)
	}
	if err := host.close(); err != nil {
		t.Fatal("second close:", err)
	}
}
