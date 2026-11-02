//go:build linux || darwin

package configtxn

import (
	"os"
	"syscall"
)

func platformSupported() bool { return true }
func safeOpenFlags() int      { return syscall.O_NOFOLLOW | syscall.O_NONBLOCK }
func owned(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Geteuid()) && (info.IsDir() || st.Nlink == 1)
}
