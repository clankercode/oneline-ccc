//go:build linux

package configtxn

import (
	"os"

	"golang.org/x/sys/unix"
)

func noExtraMetadata(f *os.File) error {
	n, err := unix.Flistxattr(int(f.Fd()), nil)
	if err != nil || n != 0 {
		return ErrUnsafe
	}
	flags, err := unix.IoctlGetInt(int(f.Fd()), unix.FS_IOC_GETFLAGS)
	return supportedInodeFlags(flags, err)
}

func supportedInodeFlags(flags int, queryErr error) error {
	// Unsupported queries (including ENOTTY/EOPNOTSUPP) cannot establish the
	// metadata policy. Do not silently assume an unqueryable inode has no flags.
	if queryErr != nil {
		return ErrUnsafe
	}
	// Index/extents/large-file/inline-data/EOF-block bits describe filesystem
	// representation. Other bits, including nodump/noatime/compression/nocow,
	// carry policy which replacement cannot preserve. Unknown bits fail closed.
	const managed = 0x00001000 | 0x00040000 | 0x00080000 | 0x00400000 | 0x10000000
	if flags & ^managed != 0 {
		return ErrConflict
	}
	return nil
}
