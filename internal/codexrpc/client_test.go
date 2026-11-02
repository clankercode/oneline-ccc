package codexrpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRPCProcess(t *testing.T) {
	if !strings.Contains(strings.Join(os.Args, " "), "-- rpc-fixture") {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req struct {
			ID     int
			Method string
		}
		_ = json.Unmarshal(scanner.Bytes(), &req)
		if req.ID == 0 {
			continue
		}
		if req.Method == "hang" {
			time.Sleep(time.Minute)
			continue
		}
		if req.Method == "reject" {
			fmt.Fprintf(os.Stdout, "{\"id\":%d,\"error\":{\"code\":1,\"message\":\"private-fixture-marker\"}}\n", req.ID)
			continue
		}
		fmt.Fprintf(os.Stdout, "{\"method\":\"notification\"}\n{\"id\":%d,\"result\":{\"ok\":true}}\n", req.ID)
	}
	os.Exit(0)
}
func fixture(t *testing.T) Program {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Program{Path: exe, Args: []string{"-test.run=TestRPCProcess", "--", "rpc-fixture"}}
}
func TestRequestsIgnoreNotificationsAndRedactNativeErrors(t *testing.T) {
	stage := filepath.Join(t.TempDir(), "stage")
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), fixture(t), stage, func(c *Client) error {
		var response struct{ OK bool }
		if err := c.Call("config/read", map[string]any{"includeLayers": true}, &response); err != nil {
			return err
		}
		if !response.OK {
			t.Error("response missing")
		}
		err := c.Call("reject", nil, nil)
		if err == nil || strings.Contains(err.Error(), "private-fixture-marker") {
			t.Fatal("native error leaked or ignored")
		}
		return nil
	})
	if err != nil {
		var errno syscall.Errno
		found := errors.As(err, &errno)
		t.Logf("request/redaction cleanup diagnostic: native_errno_present=%t native_errno=%d", found, errno)
		t.Fatal(err)
	}
}
func TestHungRequestIsCancelledAndReaped(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := Run(ctx, fixture(t), t.TempDir(), func(c *Client) error { return c.Call("hang", nil, nil) })
	if err == nil {
		t.Fatal("hung request accepted")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("owned process wasn't reaped promptly")
	}
}
func TestConfigEnvironmentExcludesInheritedSecrets(t *testing.T) {
	t.Setenv("UNRELATED_API_KEY", "private-fixture-marker")
	t.Setenv("CODEX_PROFILE", "private-profile")
	env := strings.Join(ConfigEnvironment("fixture-root"), "\n")
	if strings.Contains(env, "private-fixture-marker") || strings.Contains(env, "private-profile") {
		t.Fatal("inherited secret/profile reached config process")
	}
}

type rpcTestInput struct{ bytes.Buffer }

func (*rpcTestInput) Close() error { return nil }

func TestCanceledCallDoesNotDispatchOrAdvanceID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	input := &rpcTestInput{}
	c := &Client{ctx: ctx, input: input, output: make(chan []byte), next: 7}
	err := c.Call("config/batchWrite", map[string]string{"fixture": "mutation"}, nil)
	if err == nil || input.Len() != 0 || c.next != 7 {
		t.Fatalf("canceled RPC changed dispatch state: bytes=%d next=%d error=%v", input.Len(), c.next, err)
	}
}

func TestCancellationBetweenRequestsDoesNotSendNextMutation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := &rpcTestInput{}
	responses := make(chan []byte, 1)
	responses <- []byte(`{"id":1,"result":{"ok":true}}`)
	c := &Client{ctx: ctx, input: input, output: responses}
	var result struct{ OK bool }
	if err := c.Call("config/read", nil, &result); err != nil || !result.OK {
		t.Fatalf("initial RPC failed: %v", err)
	}
	before := input.String()
	ready, proceed := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(ready)
		<-proceed
		done <- c.Call("config/batchWrite", map[string]string{"fixture": "mutation"}, nil)
	}()
	<-ready
	cancel()
	close(proceed)
	if err := <-done; err == nil || input.String() != before || c.next != 1 {
		t.Fatalf("next canceled mutation dispatched: next=%d error=%v", c.next, err)
	}
}

type blockedRPCParams struct {
	entered chan struct{}
	proceed chan struct{}
}

func (p blockedRPCParams) MarshalJSON() ([]byte, error) {
	close(p.entered)
	<-p.proceed
	return []byte(`{"fixture":"mutation"}`), nil
}

func TestCancellationDuringEncodingDoesNotDispatchOrAdvanceID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := &rpcTestInput{}
	c := &Client{ctx: ctx, input: input, output: make(chan []byte)}
	params := blockedRPCParams{entered: make(chan struct{}), proceed: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- c.Call("config/batchWrite", params, nil) }()
	<-params.entered
	cancel()
	close(params.proceed)
	if err := <-done; err == nil || input.Len() != 0 || c.next != 0 {
		t.Fatalf("mutation dispatched after encoding was canceled: bytes=%d next=%d error=%v", input.Len(), c.next, err)
	}
}
