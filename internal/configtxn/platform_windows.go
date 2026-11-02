//go:build windows

package configtxn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func platformSupported() bool                   { return true }
func safeOpenFlags() int                        { return 0 }
func pathKey(path string) string                { return strings.ToLower(path) }
func snapshotMode(info os.FileInfo) os.FileMode { return 0600 }

// Local drive-qualified paths only: reject namespaces/UNC, ADS, Win32 aliases,
// device names and ambiguous trailing dots/spaces before touching the filesystem.
func platformPath(path string) error {
	if len(path) < 3 || path[1] != ':' || path[2] != '\\' || !((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) {
		return ErrUnsafe
	}
	for _, part := range strings.Split(path[3:], `\`) {
		if part == "" {
			continue
		}
		if strings.ContainsAny(part, `:<>"|?*`+"\x00") || strings.TrimRight(part, " .") != part {
			return ErrUnsafe
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CONIN$" || base == "CONOUT$" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
			return ErrUnsafe
		}
	}
	return nil
}

func checkComponent(path string, info os.FileInfo) error {
	h, err := nativeOpen(path, windows.FILE_READ_ATTRIBUTES)
	if err != nil {
		return ErrUnsafe
	}
	defer windows.CloseHandle(h)
	return checkNative(h, info.IsDir(), false)
}
func openRead(path string) (*os.File, error) {
	return openWindowsFile(path, windows.GENERIC_READ|windows.READ_CONTROL)
}
func openMode(path string) (*os.File, error) {
	return openWindowsFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE|windows.READ_CONTROL)
}
func openWindowsFile(path string, access uint32) (*os.File, error) {
	h, err := nativeOpen(path, access)
	if err != nil {
		return nil, ErrIO
	}
	f := os.NewFile(uintptr(h), path)
	if f == nil {
		windows.CloseHandle(h)
		return nil, ErrIO
	}
	if err = validateHandle(f); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func createFile(path string) (*os.File, error) {
	if err := checkPath(path, true); err != nil {
		return nil, err
	}
	sd, err := privateDescriptor(false)
	if err != nil {
		return nil, err
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, ErrUnsafe
	}
	h, err := windows.CreateFile(ptr, windows.GENERIC_READ|windows.GENERIC_WRITE|windows.READ_CONTROL, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, &sa, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	runtime.KeepAlive(sd)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(h), path)
	if f == nil {
		windows.CloseHandle(h)
		return nil, ErrIO
	}
	if err = validateHandle(f); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
func createDir(path string) error {
	if err := checkPath(path, true); err != nil {
		return err
	}
	sd, err := privateDescriptor(true)
	if err != nil {
		return err
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ErrUnsafe
	}
	err = windows.CreateDirectory(ptr, &sa)
	runtime.KeepAlive(sd)
	return err
}
func setMode(f *os.File, mode os.FileMode) error {
	if mode.Perm() != 0600 {
		return ErrUnsafe
	}
	return validateHandle(f)
}

// Windows has no portable, unprivileged equivalent of POSIX directory fsync.
// Verify the directory handle; file FlushFileBuffers and write-through renames
// provide the available durability. Power-loss persistence is not promised.
func syncDir(path string) error {
	h, err := nativeOpen(path, windows.FILE_READ_ATTRIBUTES)
	if err != nil {
		return ErrIO
	}
	defer windows.CloseHandle(h)
	return checkNative(h, true, false)
}

func prepareLock(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return privateFile(path)
	} else if !os.IsNotExist(err) {
		return ErrIO
	}
	f, err := createFile(path)
	if err != nil {
		if os.IsExist(err) {
			return privateFile(path)
		}
		return ErrIO
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return ErrIO
	}
	return syncDir(filepath.Dir(path))
}

// Never switch to truncate-in-place. Bounded sharing retries recheck the exact
// destination snapshot so a foreign edit during the retry delay is preserved.

func renameFile(ctx context.Context, src, dst, hash string, mode os.FileMode, checks renameChecks) error {
	from, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return ErrUnsafe
	}
	to, err := windows.UTF16PtrFromString(dst)
	if err != nil {
		return ErrUnsafe
	}
	return renameRetry(ctx, src, dst, hash, mode, func() error {
		return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
	}, isSharingError, checks)
}

func targetParent(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || !privateMetadata(path, info, true) {
		return ErrUnsafe
	}
	return nil
}

func trustedAncestor(path string, info os.FileInfo) bool {
	// Reparse/type checks are handle based in checkComponent. System ancestors
	// such as C:\Users are intentionally not required to have a private DACL.
	return checkComponent(path, info) == nil
}

func supportedMetadata(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !privateMetadata(path, info, info.IsDir()) {
		return ErrUnsafe
	}
	return nil
}

func sourceMetadata(path string) error {
	h, err := nativeOpen(path, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES)
	if err != nil {
		return ErrUnsafe
	}
	defer windows.CloseHandle(h)
	if err = checkNative(h, false, true); err != nil {
		return err
	}
	var info windows.ByHandleFileInformation
	if windows.GetFileInformationByHandle(h, &info) != nil || info.FileAttributes != windows.FILE_ATTRIBUTE_ARCHIVE {
		return ErrUnsafe
	}
	return nil
}

// MoveFileEx may report ACCESS_DENIED when an existing file is held without
// delete sharing. Both sources/destinations already pass the restrictive native
// policy on every attempt; retrying this class is bounded and grants no access.
func isSharingError(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
