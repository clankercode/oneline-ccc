//go:build !linux && !darwin && !windows

package processowner

import "os/exec"

type nativeOwner struct{}

// Unsupported ownership policies fail before starting a native process.
func prepareNative(*exec.Cmd) (*nativeOwner, error) { return nil, ErrPolicy }
func (*nativeOwner) started() error                 { return ErrPolicy }
func (*nativeOwner) terminate() error               { return ErrPolicy }
func (*nativeOwner) close() error                   { return nil }
