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
func ownedPath(path string, info os.FileInfo) bool       { return owned(info) }
func privateMetadata(path string, info os.FileInfo, dir bool) bool {
	mode := os.FileMode(0600)
	if dir {
		mode = 0700
	}
	return info.Mode().Perm() == mode && owned(info)
}
func openRead(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|safeOpenFlags(), 0)
}
func createFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|safeOpenFlags(), 0600)
}
func createDir(path string) error                { return os.Mkdir(path, 0700) }
func setMode(f *os.File, mode os.FileMode) error { return f.Chmod(mode) }
func renameFile(ctx context.Context, src, dst string) error {
	if err := ctx.Err(); err != nil {
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
func targetParent(path string) error         { return nil }
