//go:build windows

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	onelineccc "github.com/clankercode/oneline-ccc"
	"github.com/clankercode/oneline-ccc/internal/configtxn"
)

func TestWindowsSSHCanonicalImmutableScriptAndMinimalEnvironment(t *testing.T) {
	root := t.TempDir()
	systemRoot := filepath.Join(root, "Windows")
	powershell := filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if err := os.MkdirAll(filepath.Dir(powershell), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(powershell, []byte("unexecuted fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SystemRoot", systemRoot)
	t.Setenv("LLMP_API_KEY", "synthetic-not-inherited")
	t.Setenv("PSModulePath", "synthetic-injection")
	state := filepath.Join(root, "state")
	cmd, err := onelineccc.SSHCommand(context.Background(), state)
	if errors.Is(err, configtxn.ErrUnsupported) {
		t.Skip("protected Windows staging unavailable; fails closed")
	}
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := os.ReadFile(filepath.Join("..", "..", "scripts", "setup-ssh-server.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(canonical)
	script := filepath.Join(state, "helpers", fmt.Sprintf("setup-ssh-server-%x.ps1", hash))
	got, err := os.ReadFile(script)
	if err != nil || !bytes.Equal(got, canonical) {
		t.Fatal("embedded helper differs from canonical source")
	}
	if !reflect.DeepEqual(cmd.Args, []string{powershell, "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script}) {
		t.Fatal("PowerShell argv changed")
	}
	if cmd.Env == nil {
		t.Fatal("nil environment")
	}
	for _, entry := range cmd.Env {
		if strings.HasPrefix(entry, "LLMP_API_KEY=") || strings.HasPrefix(entry, "PSModulePath=") {
			t.Fatal("unrelated environment inherited")
		}
	}
	if _, err = onelineccc.SSHCommand(context.Background(), state); err != nil {
		t.Fatal("identical stage not reusable")
	}
	if err = os.WriteFile(script, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if c, e := onelineccc.SSHCommand(context.Background(), state); e == nil || c != nil {
		t.Fatal("mismatched immutable helper accepted")
	}
	if got, _ = os.ReadFile(script); string(got) != "tampered" {
		t.Fatal("foreign stage was overwritten")
	}
}
