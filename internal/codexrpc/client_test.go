package codexrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
