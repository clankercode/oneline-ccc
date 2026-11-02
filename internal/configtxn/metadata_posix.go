//go:build linux || darwin

package configtxn

import (
	"os"
	"path/filepath"
	"syscall"
)

// The only supported POSIX metadata is ordinary mode bits, our current owner and
// effective group, and no ACL/xattr/extra native flags. Refusal avoids broadening
// an ACL mask into owning-group permission or inheriting a default ACL on stages.
func supportedMetadata(path string) error {
	if err := checkPath(path, false); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|safeOpenFlags(), 0)
	if err != nil {
		return ErrUnsafe
	}
	defer f.Close()
	return supportedHandleMetadata(f)
}
func supportedHandleMetadata(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return ErrUnsafe
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Geteuid()) || st.Gid != uint32(os.Getegid()) || (!info.IsDir() && !info.Mode().IsRegular()) || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return ErrUnsafe
	}
	return noExtraMetadata(f)
}
func targetParent(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		return ErrUnsafe
	}
	return supportedMetadata(path)
}
func parentForCreate(path string) error { return targetParent(filepath.Dir(path)) }
