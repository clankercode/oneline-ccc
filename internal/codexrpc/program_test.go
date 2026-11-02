package codexrpc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This is the only executable used by version tests; it never reads a client
// configuration or contacts a network service.
func TestVersionProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "version-fixture" || i+1 >= len(os.Args) {
			continue
		}
		mode := os.Args[i+1]
		switch mode {
		case "pipes":
			child := exec.Command(os.Args[0], "-test.run=^TestVersionProcess$", "--", "version-fixture", "hold-pipes")
			child.Env = append(os.Environ(), "GORACE=atexit_sleep_ms=0")
			child.Stdout, child.Stderr = os.Stdout, os.Stderr
			if child.Start() != nil {
				os.Exit(3)
			}
			fmt.Print("codex-cli 0.166.0\n")
		case "hold-pipes":
			runCleanupHelper("pipe-helper-address")
		case "oversize":
			fmt.Print("codex-cli " + strings.Repeat("x", 2<<20))
		case "stderr":
			fmt.Fprint(os.Stderr, "private-fixture-marker")
			os.Exit(2)
		case "hang":
			time.Sleep(10 * time.Second)
		case "output":
			fmt.Print(os.Args[i+2])
		}
		os.Exit(0)
	}
}

func versionFixture(mode string, args ...string) Program {
	return Program{Path: os.Args[0], Args: append([]string{"-test.run=^TestVersionProcess$", "--", "version-fixture", mode}, args...)}
}

func TestVersionBoundsInheritedPipes(t *testing.T) {
	stage := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		value string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		value, err := Version(ctx, versionFixture("pipes"), stage)
		done <- result{value, err}
	}()
	// Wait for a genuinely active inherited-pipe holder before cancellation;
	// process startup speed must not decide whether this regression is exercised.
	var address string
	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for address == "" {
		select {
		case <-ticker.C:
			data, _ := os.ReadFile(filepath.Join(stage, "pipe-helper-address"))
			address = string(data)
		case <-deadline:
			t.Fatal("inherited-pipe helper did not become ready")
		}
	}
	defer releaseHelper(address)
	start := time.Now()
	cancel()
	select {
	case got := <-done:
		if got.err == nil || got.value != "" {
			t.Error("canceled version returned success or child output")
		}
	case <-time.After(time.Second):
		t.Fatal("inherited pipes exceeded bounded cancellation")
	}
	if time.Since(start) > time.Second {
		t.Error("inherited pipes exceeded bounded cancellation")
	}
	checkHelperStopped(t, address)
	if err := os.RemoveAll(stage); err != nil {
		t.Fatal("inherited-pipe helper retained the stage", err)
	}
}

func TestVersionOutputLimit(t *testing.T) {
	value, err := Version(context.Background(), versionFixture("oversize"), t.TempDir())
	if err == nil || value != "" {
		t.Errorf("oversized output accepted: returned bytes=%d", len(value))
	}
}

func TestVersionExactIdentityAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		name, output, want string
	}{
		{"stable", "codex-cli 0.166.0\n", "0.166.0"},
		{"windows-newline", "codex-cli 0.166.0\r\n", "0.166.0"},
		{"no-newline", "codex-cli 0.166.0", "0.166.0"},
		{"prerelease", "codex-cli 0.167.0-alpha.11\n", "0.167.0-alpha.11"},
		{"private-trailer", "codex-cli 0.166.0\nprivate-fixture-marker\n", ""},
		{"ansi-trailer", "codex-cli 0.166.0\x1b[31m", ""},
		{"leading-space", " codex-cli 0.166.0\n", ""},
		{"trailing-space", "codex-cli 0.166.0 \n", ""},
		{"extra-newline", "codex-cli 0.166.0\n\n", ""},
		{"wrong-identity", "other-cli 0.166.0\n", ""},
		{"non-version", "codex-cli private-fixture-marker\n", ""},
		{"leading-zero", "codex-cli 00.166.0\n", ""},
		{"empty-prerelease", "codex-cli 0.166.0-\n", ""},
		{"empty-component", "codex-cli 0.166.0-alpha..1\n", ""},
		{"numeric-prerelease-zero", "codex-cli 0.166.0-alpha.01\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := Version(context.Background(), versionFixture("output", tc.output), t.TempDir())
			if tc.want != "" {
				if err != nil || value != tc.want {
					t.Fatalf("valid identity rejected: %v", err)
				}
				return
			}
			if err == nil || value != "" || strings.Contains(err.Error(), "private-fixture-marker") {
				t.Fatal("malformed identity accepted or child text exposed")
			}
		})
	}
	value, err := Version(context.Background(), versionFixture("stderr"), t.TempDir())
	if err == nil || value != "" || strings.Contains(err.Error(), "private-fixture-marker") {
		t.Fatal("child stderr exposed or failure accepted")
	}
}

func TestVersionReapsBlockedProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	value, err := Version(ctx, versionFixture("hang"), t.TempDir())
	if err == nil || value != "" || time.Since(start) > time.Second {
		t.Fatal("non-version process was not stopped promptly")
	}
}
