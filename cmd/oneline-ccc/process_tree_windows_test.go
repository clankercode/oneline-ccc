package main

import (
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestWindowsNativeTreePolicyPreservesConsoleAndOwnsDescendants(t *testing.T) {
	cmd := childFixture("exit7")
	original := &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_UNICODE_ENVIRONMENT}
	cmd.SysProcAttr = original
	tree, err := prepareProcessTree(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.close()
	if cmd.SysProcAttr == original || original.CreationFlags != windows.CREATE_UNICODE_ENVIRONMENT {
		t.Fatal("backend attributes mutated")
	}
	if cmd.SysProcAttr.CreationFlags != original.CreationFlags|windows.CREATE_SUSPENDED || !cmd.SysProcAttr.HideWindow {
		t.Fatal("suspended start or backend attributes lost")
	}
	// CREATE_NEW_PROCESS_GROUP disables Ctrl+C for the new group. Job ownership
	// must leave inherited console control handling unchanged.
	if cmd.SysProcAttr.CreationFlags&windows.CREATE_NEW_PROCESS_GROUP != 0 {
		t.Fatal("native console Ctrl+C policy changed")
	}
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	if err := windows.QueryInformationJobObject(tree.job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), nil); err != nil {
		t.Fatal(err)
	}
	if limits.BasicLimitInformation.LimitFlags != windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE {
		t.Fatal("job permits descendant escape or lacks lifetime ownership")
	}
}
