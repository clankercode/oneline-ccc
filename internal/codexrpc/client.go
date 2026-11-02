// Package codexrpc reuses Codex's own configuration API in a disposable root.
package codexrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/clankercode/oneline-ccc/internal/processowner"
)

type Program struct {
	Path string
	Args []string
}
type Client struct {
	ctx    context.Context
	input  io.WriteCloser
	output <-chan []byte
	next   int
}

// Retain only the numeric native cause for hermetic cleanup diagnostics. Native
// errors can include paths or child text; neither formatting nor unwrapping this
// error may expose those details through the public configuration API.
type cleanupFailure struct {
	message string
	errno   syscall.Errno
}

func (e *cleanupFailure) Error() string { return e.message }
func (e *cleanupFailure) Unwrap() error {
	if e.errno == 0 {
		return nil
	}
	return e.errno
}

func ownedCleanupFailure(message string, cause error) error {
	var errno syscall.Errno
	_ = errors.As(cause, &errno)
	return &cleanupFailure{message: message, errno: errno}
}

func ConfigEnvironment(stage string) []string {
	var env []string
	for _, name := range []string{"PATH", "SystemRoot", "WINDIR", "COMSPEC", "PATHEXT", "APPDATA", "LOCALAPPDATA", "USERPROFILE", "HOME", "TEMP", "TMP", "TMPDIR", "LANG", "LC_ALL", "TERM"} {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return append(env, "CODEX_HOME="+stage, "LLMP_API_KEY=fixture-only-config-validation", "OPENAI_API_KEY=fixture-only-config-validation", "RUST_LOG=error")
}

// rpcCleanup keeps the existing cleanup policy at one boundary so a controlled
// schedule can distinguish root reaping from delivery of the Wait result. The
// production callbacks below retain the sole command.Wait and native owner.
type rpcCleanup struct {
	wait, terminate, close, killRoot func() error
	after                            func(time.Duration) <-chan time.Time
}

func (c rpcCleanup) run() (result error) {
	done := make(chan error, 1)
	go func() { done <- c.wait() }()
	select {
	case <-done:
	case <-c.after(time.Second):
		if err := c.terminate(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			result = ownedCleanupFailure("cannot terminate owned Codex configuration processes", err)
			_ = c.killRoot()
		}
		<-done
	}
	if err := c.close(); err != nil {
		result = errors.Join(result, ownedCleanupFailure("cannot clean up owned Codex configuration processes", err))
	}
	return result
}

func Run(parent context.Context, program Program, stage string, action func(*Client) error) (result error) {
	ctx, cancel := context.WithTimeout(parent, 25*time.Second)
	defer cancel()
	args := append(append([]string{}, program.Args...), "app-server", "--listen", "stdio://")
	command := exec.CommandContext(ctx, program.Path, args...)
	command.Env = ConfigEnvironment(stage)
	command.Dir = stage
	command.WaitDelay = time.Second
	input, err := command.StdinPipe()
	if err != nil {
		return errors.New("cannot open configuration transport")
	}
	defer input.Close()
	output, err := command.StdoutPipe()
	if err != nil {
		return errors.New("cannot open configuration transport")
	}
	defer output.Close()
	stderr, err := command.StderrPipe()
	if err != nil {
		return errors.New("cannot open configuration transport")
	}
	defer stderr.Close()
	owner, err := processowner.Prepare(command)
	if err != nil {
		return errors.New("cannot establish configuration process ownership")
	}
	if err = owner.Start(); err != nil {
		if cleanupErr := owner.Close(); cleanupErr != nil {
			return ownedCleanupFailure("cannot start or clean up the selected Codex executable", cleanupErr)
		}
		return errors.New("cannot start the selected Codex executable")
	}
	stderrDone := make(chan struct{})
	go func() { defer close(stderrDone); _, _ = io.Copy(io.Discard, stderr) }()
	lines := make(chan []byte, 1)
	stdoutDone := make(chan struct{})
	go func() {
		defer close(stdoutDone)
		defer close(lines)
		scan := bufio.NewScanner(output)
		scan.Buffer(make([]byte, 8192), 5<<20)
		for scan.Scan() {
			line := append([]byte(nil), scan.Bytes()...)
			select {
			case lines <- line:
			case <-ctx.Done():
				return
			}
		}
	}()
	defer func() {
		_ = input.Close()
		// Even a clean direct-process exit can leave descendants holding CWD or
		// handles in the stage. Complete owned-tree cleanup before the adapter
		// is allowed to remove it, and report failure without native path text.
		cleanup := rpcCleanup{wait: command.Wait, terminate: owner.Terminate, close: owner.Close, killRoot: command.Process.Kill, after: time.After}
		if cleanupErr := cleanup.run(); cleanupErr != nil {
			result = errors.Join(result, cleanupErr)
		}
		cancel()
		_ = output.Close()
		_ = stderr.Close()
		<-stdoutDone
		<-stderrDone
	}()
	client := &Client{ctx: ctx, input: input, output: lines}
	if err = client.Call("initialize", map[string]any{"clientInfo": map[string]string{"name": "oneline-ccc", "version": "0.1.0-dev"}, "capabilities": map[string]bool{"experimentalApi": true}}, nil); err != nil {
		return err
	}
	if err = json.NewEncoder(input).Encode(map[string]string{"method": "initialized"}); err != nil {
		return errors.New("configuration transport closed")
	}
	return action(client)
}

func (c *Client) Call(method string, params any, result any) error {
	if c.ctx.Err() != nil {
		return errors.New("Codex configuration operation cancelled or timed out")
	}
	next := c.next + 1
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": next, "method": method, "params": params})
	if err != nil {
		return errors.New("configuration transport closed")
	}
	request = append(request, '\n')
	// Parameter encoding can call custom marshalers. Recheck immediately at
	// dispatch so cancellation during encoding sends nothing and consumes no ID.
	if c.ctx.Err() != nil {
		return errors.New("Codex configuration operation cancelled or timed out")
	}
	c.next = next
	if n, err := c.input.Write(request); err != nil || n != len(request) {
		return errors.New("configuration transport closed")
	}
	for {
		select {
		case <-c.ctx.Done():
			return errors.New("Codex configuration operation cancelled or timed out")
		case line, ok := <-c.output:
			if !ok {
				return errors.New("Codex configuration process stopped before validation completed")
			}
			var response struct {
				ID     *int            `json:"id"`
				Result json.RawMessage `json:"result"`
				Error  json.RawMessage `json:"error"`
			}
			if json.Unmarshal(line, &response) != nil {
				return errors.New("Codex returned invalid configuration data")
			}
			if response.ID == nil || *response.ID != c.next {
				continue
			}
			if len(response.Error) > 0 && strings.TrimSpace(string(response.Error)) != "null" {
				return errors.New("Codex rejected the configuration operation; inspect supported fields and version")
			}
			if result != nil && json.Unmarshal(response.Result, result) != nil {
				return errors.New("Codex configuration response had an unsupported shape")
			}
			return nil
		}
	}
}
