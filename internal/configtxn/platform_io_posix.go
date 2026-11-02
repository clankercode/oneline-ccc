//go:build linux || darwin

package configtxn

import (
	"context"
	"os"
)

func platformPath(path string) error                     { return nil }
func pathKey(path string) string                         { return path }
func checkComponent(path string, info os.FileInfo) error { return nil }
func validateHandle(f *os.File) error                    { return nil }
func snapshotMode(info os.FileInfo) os.FileMode          { return info.Mode().Perm() }
func ownedPath(path string, info os.FileInfo) bool {
	return owned(info) && supportedMetadata(path) == nil
}
func privateMetadata(path string, info os.FileInfo, dir bool) bool {
	mode := os.FileMode(0600)
	if dir {
		mode = 0700
	}
	return info.Mode().Perm() == mode && owned(info) && supportedMetadata(path) == nil
}
func openRead(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|safeOpenFlags(), 0)
}
func createFile(path string) (*os.File, error) {
	if err := parentForCreate(path); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|safeOpenFlags(), 0600)
}
func createDir(path string) error {
	if err := parentForCreate(path); err != nil {
		return err
	}
	return os.Mkdir(path, 0700)
}
func setMode(f *os.File, mode os.FileMode) error {
	if err := supportedHandleMetadata(f); err != nil {
		return err
	}
	return f.Chmod(mode)
}
func renameFile(ctx context.Context, src, dst, hash string, mode os.FileMode, checks renameChecks) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if checks.before != nil {
		if err := checks.before(); err != nil {
			return err
		}
	}
	if err := verifySource(src, hash, mode); err != nil {
		return err
	}
	return os.Rename(src, dst)
}
func prepareLock(path string) error { return nil }
func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return ErrIO
	}
	defer f.Close()
	if f.Sync() != nil {
		return ErrIO
	}
	return nil
}

func openMode(path string) (*os.File, error) { return openRead(path) }

func sourceMetadata(path string) error { return supportedMetadata(path) }
