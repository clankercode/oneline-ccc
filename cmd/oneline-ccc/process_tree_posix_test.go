//go:build linux || darwin

package main

import (
	"os"
	"runtime"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
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

func TestPOSIXProcessGroupPreparationWithPipeInput(t *testing.T) {
	input, heldOpen, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer heldOpen.Close()
	// Darwin pipes implement these socket-compatibility ioctls. A successful
	// TIOCGPGRP is not evidence of a terminal or foreground ownership.
	if runtime.GOOS == "darwin" {
		group, err := unix.IoctlGetInt(int(input.Fd()), unix.TIOCGPGRP)
		t.Logf("native pipe TIOCGPGRP: success=%t zero_group=%t", err == nil, group == 0)
	}
	cmd := childFixture("ready-wait")
	cmd.Stdin = input
	tree, err := prepareProcessTree(cmd)
	if err != nil {
		t.Fatalf("pipe input refused before child launch: %v", err)
	}
	if tree.terminal != nil || cmd.SysProcAttr.Foreground || !cmd.SysProcAttr.Setpgid {
		t.Fatal("pipe input incorrectly took terminal ownership")
	}
	if err := tree.close(); err != nil {
		t.Fatal(err)
	}
}
