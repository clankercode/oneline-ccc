package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clankercode/oneline-ccc/internal/core"
)

func TestCLIReviewANSIPathRedaction(t *testing.T) {
	for _, args := range [][]string{{"status", "--json"}, {"status"}, {"doctor"}} {
		for _, pattern := range []string{"/private/codex", "/private/\x1b[32mcodex"} {
			d, f, out, _, _ := harness(t)
			f.snapshot.Clients[0].Executable = pattern
			f.snapshot.Clients[0].Reason = "native error at /private/\x1b[31mcodex"
			if run(context.Background(), args, d) != 0 {
				t.Fatal("status failed")
			}
			if strings.Contains(out.String(), "/private/codex") {
				t.Fatal("ANSI normalization revealed private path")
			}
			if f.snapshot.Clients[0].Executable != pattern {
				t.Fatal("backend mutated")
			}
		}
	}
}

func TestCLIReviewJSONPreservesExactUnicodeModel(t *testing.T) {
	d, f, out, _, _ := harness(t)
	model := "provider/👩\u200d💻"
	f.snapshot.Clients[0].Profile.Model = model
	if run(context.Background(), []string{"status", "--json"}, d) != 0 {
		t.Fatal("status failed")
	}
	var got core.Snapshot
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Clients[0].Profile.Model != model {
		t.Fatalf("model changed: %q", got.Clients[0].Profile.Model)
	}
}

type failWriteNumber struct{ calls, failAt int }

func (w *failWriteNumber) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}
func TestCLIReviewPlainOutputReportsFirstWriteError(t *testing.T) {
	for _, command := range []string{"status", "doctor"} {
		d, f, _, _, _ := harness(t)
		counter := &failWriteNumber{}
		if err := printStatus(counter, f.snapshot, false, command == "doctor"); err != nil {
			t.Fatal(err)
		}
		for _, at := range []int{1, counter.calls} {
			w := &failWriteNumber{failAt: at}
			if err := printStatus(w, f.snapshot, false, command == "doctor"); !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("%s write %d lost first error: %v", command, at, err)
			}
			d.out = &failWriteNumber{failAt: at}
			if run(context.Background(), []string{command}, d) == 0 {
				t.Fatalf("%s write %d reported success", command, at)
			}
		}
	}
}

// These native fixtures never read ambient environment or operator paths. A
// descendant owns the listener until exit, so connection failure after cleanup
// proves it stopped; no guessed delay is used as evidence of absence.
func TestCLIReviewTreeFixture(t *testing.T) {
	args := os.Args
	if len(args) < 4 || args[len(args)-3] != "ccc-tree" {
		return
	}
	mode, marker := args[len(args)-2], args[len(args)-1]
	if mode == "parent" {
		c := reviewTreeCommand("descendant", marker)
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		if c.Start() != nil {
			os.Exit(8)
		}
		_ = c.Wait()
		os.Exit(0)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.Exit(9)
	}
	_ = listener.(*net.TCPListener).SetDeadline(time.Now().Add(15 * time.Second))
	fmt.Fprintf(os.Stdout, "READY %s\n", listener.Addr())
	for {
		c, err := listener.Accept()
		if err != nil {
			os.Exit(0)
		}
		_ = c.SetDeadline(time.Now().Add(time.Second))
		line, _ := bufio.NewReader(c).ReadString('\n')
		if strings.TrimSpace(line) == "release" {
			_ = c.Close()
			os.Exit(0)
		}
		if _, err := os.Stat(marker); err == nil {
			_ = os.WriteFile(marker+".result", []byte("still-running"), 0600)
			fmt.Fprintln(c, "still-running")
		}
		_ = c.Close()
	}
}
func reviewTreeCommand(mode, marker string) *exec.Cmd {
	c := exec.Command(os.Args[0], "-test.run=^TestCLIReviewTreeFixture$", "--", "ccc-tree", mode, marker)
	c.Env = []string{}
	return c
}

type treeReadyWriter struct {
	sync.Mutex
	pending string
	ready   chan string
}

func (w *treeReadyWriter) Write(p []byte) (int, error) {
	w.Lock()
	defer w.Unlock()
	w.pending += string(p)
	if before, _, ok := strings.Cut(w.pending, "\n"); ok && strings.HasPrefix(before, "READY ") {
		select {
		case w.ready <- strings.TrimPrefix(before, "READY "):
		default:
		}
		w.pending = ""
	}
	return len(p), nil
}
func TestCLIReviewCancellationStopsOwnedDescendant(t *testing.T) {
	for _, commandContext := range []bool{false, true} {
		t.Run(fmt.Sprint(commandContext), func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "probe")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cmd := reviewTreeCommand("parent", marker)
			if commandContext {
				cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
				cmd.Env = []string{}
			}
			output := &treeReadyWriter{ready: make(chan string, 1)}
			done := make(chan error, 1)
			go func() { done <- runProcess(ctx, cmd, nil, output, io.Discard) }()
			var addr string
			select {
			case addr = <-output.ready:
			case err := <-done:
				t.Fatalf("fixture ended before readiness: %v", err)
			case <-time.After(8 * time.Second):
				t.Fatal("fixture readiness timeout")
			}
			defer func() {
				if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
					fmt.Fprintln(c, "release")
					_ = c.Close()
				}
			}()
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("cleanup did not return")
			}
			if err := os.WriteFile(marker, []byte("probe"), 0600); err != nil {
				t.Fatal(err)
			}
			if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
				_ = c.SetDeadline(time.Now().Add(time.Second))
				fmt.Fprintln(c, "probe")
				response, _ := bufio.NewReader(c).ReadString('\n')
				_ = c.Close()
				if strings.TrimSpace(response) == "still-running" {
					t.Fatal("owned descendant executing after cancellation returned")
				}
			}
			if _, err := os.Stat(marker + ".result"); err == nil {
				t.Fatal("descendant wrote after runner returned")
			}
			if cmd.ProcessState == nil {
				t.Fatal("direct child not reaped")
			}
		})
	}
}

func TestTerminalIdentityNormalizationCannotRevealPrivatePath(t *testing.T) {
	d, f, out, _, _ := harness(t)
	model := "/private/\u200dcodex"
	f.snapshot.Clients[0].Profile.Model = model
	if run(context.Background(), []string{"status"}, d) != 0 || strings.Contains(out.String(), "/private/codex") {
		t.Fatal("terminal identity normalization revealed private path")
	}
	if f.snapshot.Clients[0].Profile.Model != model {
		t.Fatal("backend identity mutated")
	}
}
