package app

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
)

func TestSSHAvailabilityDoesNotPrepareCommand(t *testing.T) {
	e, _, _, _ := fixture(t, true)
	if e.SSHSetupUnavailableReason() == "" {
		t.Fatal("missing helper was offered as available")
	}
	calls := 0
	e.options.SSHCommand = func(context.Context) (*exec.Cmd, error) {
		calls++
		return exec.Command("fixture-never-executed"), nil
	}
	reason := e.SSHSetupUnavailableReason()
	if calls != 0 || (reason == "") != (runtime.GOOS == "windows") {
		t.Fatal("availability prepared work or misreported the actual platform")
	}
	cmd, err := e.Command(context.Background(), "ssh", "")
	if runtime.GOOS == "windows" {
		if err != nil || cmd == nil || calls != 1 {
			t.Fatal("explicit Windows action did not prepare its helper")
		}
	} else if err == nil || cmd != nil || calls != 0 {
		t.Fatal("unavailable platform prepared a helper")
	}
}
