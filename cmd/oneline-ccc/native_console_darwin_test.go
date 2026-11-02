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
		return fmt.Sprintf("%+v", m), err
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

type nativeDarwinPTY struct {
	master, slave *os.File
	before        *unix.Termios
	cmd           *exec.Cmd
	wait          chan error
	done, closed  bool
	exitErr       error
	root          string
	tail          []byte
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
	p.cmd = exec.Command(os.Args[0], "-test.run=^TestNativeConsoleFixture$")
	p.cmd.Dir, p.cmd.Env = root, nativeConsoleEnv(root, scenario, noColor)
	p.cmd.Stdin, p.cmd.Stdout, p.cmd.Stderr = p.slave, p.slave, p.slave
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
	if !p.done && p.wait != nil {
		select {
		case p.exitErr = <-p.wait:
			p.done = true
		default:
		}
	}
	return p.done, p.exitErr
}
func (p *nativeDarwinPTY) drained() []byte { return p.tail }
func (p *nativeDarwinPTY) close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	var result error
	if p.wait != nil && !p.done {
		// A suspended TUI's synthetic command has its own foreground group.
		// Kill that owned group first and let runProcess wait/reap it on SIGTERM.
		result = errors.Join(result, p.killSyntheticGroup())
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-p.wait:
			p.done = true
		case <-time.After(5 * time.Second):
			_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
			// Exactly the same waiter, never a second cmd.Wait.
			<-p.wait
			p.done = true
			result = errors.Join(result, errors.New("Darwin fixture required forced termination"))
		}
	} else if p.done && p.exitErr != nil {
		result = errors.Join(result, p.killSyntheticGroup())
	}
	if p.master != nil && p.done {
		for {
			data, err := p.read(0)
			p.tail = append(p.tail, data...)
			if err != nil && !errors.Is(err, io.EOF) {
				result = errors.Join(result, err)
			}
			if err != nil || len(data) == 0 {
				break
			}
			if len(p.tail) > 2<<20 {
				result = errors.Join(result, errors.New("Darwin final drain exceeded output bound"))
				break
			}
		}
	}
	if p.slave != nil {
		if p.before != nil {
			after, err := unix.IoctlGetTermios(int(p.slave.Fd()), unix.TIOCGETA)
			if err != nil || !reflect.DeepEqual(p.before, after) {
				result = errors.Join(result, fmt.Errorf("slave modes not restored before teardown: %v", err))
			}
			// Failure-path repair happens after the assertion, never as its proof.
			result = errors.Join(result, unix.IoctlSetTermios(int(p.slave.Fd()), unix.TIOCSETA, p.before))
		}
		result = errors.Join(result, p.slave.Close())
	}
	if p.master != nil {
		result = errors.Join(result, p.master.Close())
	}
	return result
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
