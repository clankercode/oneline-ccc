//go:build linux || darwin

package monitor

import (
	"os"
	"strconv"
	"syscall"
)

func currentUID() string { return strconv.Itoa(os.Geteuid()) }
func privateInfo(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Geteuid()) && (info.IsDir() || st.Nlink == 1)
}
func supportedFiles() bool { return true }
func lockFlags() int       { return syscall.O_NOFOLLOW | syscall.O_NONBLOCK }
