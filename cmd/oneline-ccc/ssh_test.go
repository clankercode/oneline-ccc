package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	onelineccc "github.com/clankercode/oneline-ccc"
)

func TestSSHUnavailableOffWindowsWithoutStaging(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows behavior")
	}
	root := filepath.Join(t.TempDir(), "absent")
	command, err := onelineccc.SSHCommand(context.Background(), root)
	if err == nil || command != nil || !strings.Contains(err.Error(), "Windows") {
		t.Fatal("non-Windows SSH did not give a useful refusal")
	}
	if _, err = os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("unavailable SSH staged files")
	}
}
func TestSSHCancelledBeforeStaging(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if command, err := onelineccc.SSHCommand(ctx, root); err == nil || command != nil {
		t.Fatal("cancelled SSH accepted")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("cancelled SSH staged files")
	}
}
