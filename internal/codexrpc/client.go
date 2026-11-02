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
	"time"
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

func ConfigEnvironment(stage string) []string {
	var env []string
	for _, name := range []string{"PATH", "SystemRoot", "WINDIR", "COMSPEC", "PATHEXT", "APPDATA", "LOCALAPPDATA", "USERPROFILE", "HOME", "TEMP", "TMP", "TMPDIR", "LANG", "LC_ALL", "TERM"} {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return append(env, "CODEX_HOME="+stage, "LLMP_API_KEY=fixture-only-config-validation", "OPENAI_API_KEY=fixture-only-config-validation", "RUST_LOG=error")
}

func Run(parent context.Context, program Program, stage string, action func(*Client) error) error {
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
	output, err := command.StdoutPipe()
	if err != nil {
		return errors.New("cannot open configuration transport")
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return errors.New("cannot open configuration transport")
	}
	if err = command.Start(); err != nil {
		return errors.New("cannot start the selected Codex executable")
	}
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	lines := make(chan []byte, 1)
	go func() {
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
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		select {
		case <-done:
		case <-time.After(time.Second):
			_ = command.Process.Kill()
			<-done
		}
		cancel()
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
	c.next++
	if err := json.NewEncoder(c.input).Encode(map[string]any{"jsonrpc": "2.0", "id": c.next, "method": method, "params": params}); err != nil {
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
