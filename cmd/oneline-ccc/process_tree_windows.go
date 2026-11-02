package main

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Jobs own descendants without changing console process groups or Ctrl+C
// delivery. A suspended start closes the spawn-before-assignment escape race.
type processTree struct {
	cmd   *exec.Cmd
	job   windows.Handle
	ready chan struct{}
}

func prepareProcessTree(cmd *exec.Cmd) (*processTree, error) {
	attr := syscall.SysProcAttr{}
	if cmd.SysProcAttr != nil {
		attr = *cmd.SysProcAttr
	}
	if attr.CreationFlags&windows.CREATE_SUSPENDED != 0 {
		return nil, errArguments
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)))
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	attr.CreationFlags |= windows.CREATE_SUSPENDED
	cmd.SysProcAttr = &attr
	return &processTree{cmd: cmd, job: job, ready: make(chan struct{})}, nil
}
func (p *processTree) started() error {
	// CommandContext cancellation can arrive immediately after Start. It must
	// wait for assignment/resume (or its failure), never bypass Job ownership.
	defer close(p.ready)
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.cmd.Process.Pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(p.job, process); err != nil {
		return err
	}
	return resumeInitialThread(uint32(p.cmd.Process.Pid))
}
func resumeInitialThread(pid uint32) error {
	// exec.Start closes CreateProcess's thread handle. Before any user code is
	// resumed, find the sole initial thread and open it explicitly for resume.
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	var threadID uint32
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID == pid {
			if threadID != 0 {
				return errArguments
			}
			threadID = entry.ThreadID
		}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return err
	}
	if threadID == 0 {
		return os.ErrProcessDone
	}
	thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, threadID)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(thread)
	count, err := windows.ResumeThread(thread)
	if err != nil {
		return err
	}
	if count != 1 {
		return errArguments
	}
	return nil
}
func (p *processTree) terminate() error {
	<-p.ready
	return windows.TerminateJobObject(p.job, 1)
}

type jobAccounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses     uint32
}

func (p *processTree) close() error {
	defer windows.CloseHandle(p.job)
	// TerminateJobObject is asynchronous. Observe active count reaching zero,
	// rather than returning while an owned process can still execute.
	if err := windows.TerminateJobObject(p.job, 1); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var accounting jobAccounting
		err := windows.QueryInformationJobObject(p.job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil)
		if err != nil {
			return err
		}
		if accounting.ActiveProcesses == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("native process tree cleanup timed out")
		}
		time.Sleep(time.Millisecond)
	}
}
