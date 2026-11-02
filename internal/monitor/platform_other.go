//go:build !linux && !darwin

package monitor

import "os"

// Native Windows lifecycle remains gated until observer ACL/reparse protection
// and Scheduler ownership/absence semantics have native CI proof. Never silently
// substitute a system service, password-based task or unprotected atomic write.
func currentUID() string           { return "" }
func privateInfo(os.FileInfo) bool { return false }
func supportedFiles() bool         { return false }
func lockFlags() int               { return 0 }
