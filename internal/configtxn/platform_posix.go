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

// Root-owned system ancestors are trusted, but writable foreign ancestors and
// group/other-writable paths cannot anchor a future configuration tree safely.
func trustedAncestor(path string, info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && (st.Uid == 0 || st.Uid == uint32(os.Geteuid())) && info.Mode().Perm()&0022 == 0
}
