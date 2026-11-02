//go:build linux || darwin

package main

import (
	"syscall"
	"testing"
)

func TestPOSIXTreeRefusesForeignGroupsAndPreservesBackendAttributes(t *testing.T) {
	for _, attr := range []*syscall.SysProcAttr{{Pgid: syscall.Getpgrp()}, {Setsid: true}, {Setctty: true}, {Foreground: true}} {
		cmd := childFixture("exit7")
		cmd.SysProcAttr = attr
		if _, err := prepareProcessTree(cmd); err == nil || cmd.Process != nil {
			t.Fatal("foreign group/session was accepted")
		}
	}
	cmd := childFixture("exit7")
	original := &syscall.SysProcAttr{Noctty: true}
	cmd.SysProcAttr = original
	if _, err := prepareProcessTree(cmd); err != nil {
		t.Fatal(err)
	}
	if cmd.SysProcAttr == original || original.Setpgid || !cmd.SysProcAttr.Noctty || !cmd.SysProcAttr.Setpgid || cmd.SysProcAttr.Pgid != 0 {
		t.Fatal("backend attributes mutated or owned group lost")
	}
}
