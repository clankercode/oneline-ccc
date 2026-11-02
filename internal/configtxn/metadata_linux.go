//go:build linux

package configtxn

import (
	"golang.org/x/sys/unix"
	"os"
)

func noExtraMetadata(f *os.File) error {
	n, err := unix.Flistxattr(int(f.Fd()), nil)
	if err != nil || n != 0 {
		return ErrUnsafe
	}
	return nil
}
