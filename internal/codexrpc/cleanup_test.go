package codexrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This helper keeps its current directory until terminated. On Windows that
// prevents deleting the stage; the listener proves liveness on every platform.
func TestCleanupProcess(t *testing.T) {
	joined := strings.Join(os.Args, " ")
	if !strings.Contains(joined, "-- cleanup-") {
		return
	}
	if strings.Contains(joined, "-- cleanup-helper") {
		runCleanupHelper("")
	}
	spawn := func() string {
		child := exec.Command(os.Args[0], "-test.run=^TestCleanupProcess$", "--", "cleanup-helper")
		child.Env = []string{}
		child.Dir, _ = os.Getwd()
		out, err := child.StdoutPipe()
		if err != nil || child.Start() != nil {
			os.Exit(32)
		}
		line, err := bufio.NewReader(out).ReadString('\n')
		if err != nil {
			os.Exit(33)
		}
		out.Close()
		return strings.TrimSpace(line)
	}
	if strings.Contains(joined, "--version") {
		address := spawn()
		if os.WriteFile("helper-address", []byte(address), 0600) != nil {
			os.Exit(34)
		}
		fmt.Println("codex-cli 0.166.0")
		os.Exit(0)
	}
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		var request struct {
			ID     int
			Method string
		}
		if json.Unmarshal(scan.Bytes(), &request) != nil || request.ID == 0 {
			continue
		}
		result := map[string]string{}
		if request.Method == "spawn" {
			result["address"] = spawn()
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"id": request.ID, "result": result})
	}
	os.Exit(0)
}

func runCleanupHelper(addressFile string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.Exit(31)
	}
	_ = listener.(*net.TCPListener).SetDeadline(time.Now().Add(15 * time.Second))
	if addressFile == "" {
		fmt.Fprintln(os.Stdout, listener.Addr())
	} else if os.WriteFile(addressFile, []byte(listener.Addr().String()), 0600) != nil {
		os.Exit(35)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			os.Exit(0)
		}
		_ = conn.SetDeadline(time.Now().Add(time.Second))
		line, _ := bufio.NewReader(conn).ReadString('\n')
		if line == "stop\n" {
			conn.Close()
			os.Exit(0)
		}
		fmt.Fprintln(conn, "helper-still-running")
		conn.Close()
	}
}

func cleanupProgram(t *testing.T) Program {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Program{Path: exe, Args: []string{"-test.run=^TestCleanupProcess$", "--", "cleanup-rpc"}}
}

func checkHelperStopped(t *testing.T, address string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	fmt.Fprintln(conn, "probe")
	line, _ := bufio.NewReader(conn).ReadString('\n')
	if line == "helper-still-running\n" {
		t.Error("owned helper executing after configuration process returned")
	}
}

func releaseHelper(address string) {
	if conn, err := net.DialTimeout("tcp", address, time.Second); err == nil {
		fmt.Fprintln(conn, "stop")
		conn.Close()
	}
}

func TestRPCOwnsHelpersBeforeReturningAndStageRemoval(t *testing.T) {
	for _, outcome := range []string{"success", "refusal", "cancel"} {
		t.Run(outcome, func(t *testing.T) {
			stage := filepath.Join(t.TempDir(), "stage")
			if err := os.Mkdir(stage, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			address := ""
			defer func() { releaseHelper(address) }()
			err := Run(ctx, cleanupProgram(t), stage, func(c *Client) error {
				var result struct{ Address string }
				if err := c.Call("spawn", nil, &result); err != nil {
					return err
				}
				address = result.Address
				if outcome == "cancel" {
					cancel()
					return c.Call("next", nil, nil)
				}
				if outcome == "refusal" {
					return errors.New("synthetic validation refused")
				}
				return nil
			})
			if address == "" || (outcome == "success") != (err == nil) {
				t.Fatalf("outcome=%s action error=%v helper ready=%t", outcome, err, address != "")
			}
			checkHelperStopped(t, address)
			if err := os.RemoveAll(stage); err != nil {
				t.Fatalf("owned helper still locks stage: %v", err)
			}
		})
	}
}

func TestVersionOwnsHelpersBeforeReturning(t *testing.T) {
	stage := t.TempDir()
	version, err := Version(context.Background(), cleanupProgram(t), stage)
	if err != nil || version != "0.166.0" {
		t.Fatalf("version=%s error=%v", version, err)
	}
	data, err := os.ReadFile(filepath.Join(stage, "helper-address"))
	if err != nil {
		t.Fatal(err)
	}
	address := string(data)
	defer releaseHelper(address)
	checkHelperStopped(t, address)
	if err := os.RemoveAll(stage); err != nil {
		t.Fatalf("version helper still locks stage: %v", err)
	}
}
