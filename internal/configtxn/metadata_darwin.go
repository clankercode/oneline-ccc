//go:build darwin

package configtxn

import (
	"golang.org/x/sys/unix"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

func noExtraMetadata(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return ErrUnsafe
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Flags != 0 {
		return ErrUnsafe
	}
	n, err := unix.Flistxattr(int(f.Fd()), nil)
	if err != nil || n != 0 {
		return ErrUnsafe
	}
	// Native ACLs are not reliably enumerated by listxattr on macOS. Query the
	// open descriptor through fgetattrlist; errors/ambiguous data fail closed.
	attrs := unix.Attrlist{Bitmapcount: unix.ATTR_BIT_MAP_COUNT, Commonattr: unix.ATTR_CMN_EXTENDED_SECURITY}
	buf := make([]byte, 64*1024)
	_, _, errno := syscall.Syscall6(syscall.SYS_FGETATTRLIST, f.Fd(), uintptr(unsafe.Pointer(&attrs)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 0)
	runtime.KeepAlive(f)
	runtime.KeepAlive(attrs)
	if errno != 0 || !darwinACLAbsent(buf) {
		return ErrUnsafe
	}
	return nil
}
