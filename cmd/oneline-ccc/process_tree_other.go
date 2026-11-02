//go:build !linux && !darwin && !windows

package main

import "os/exec"

type processTree struct{}

// Unsupported ownership policies fail before starting a native process.
func prepareProcessTree(*exec.Cmd) (*processTree, error) { return nil, errArguments }
func (*processTree) started() error                      { return errArguments }
func (*processTree) terminate() error                    { return errArguments }
func (*processTree) close() error                        { return nil }
