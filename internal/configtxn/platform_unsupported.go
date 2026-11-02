//go:build !linux && !darwin && !windows

package configtxn

import (
	"context"
	"os"
)

func platformSupported() bool     { return false }
func safeOpenFlags() int          { return 0 }
func owned(info os.FileInfo) bool { return false }

func platformPath(path string) error                               { return ErrUnsupported }
func pathKey(path string) string                                   { return path }
func checkComponent(path string, info os.FileInfo) error           { return ErrUnsupported }
func validateHandle(f *os.File) error                              { return ErrUnsupported }
func snapshotMode(info os.FileInfo) os.FileMode                    { return 0 }
func ownedPath(path string, info os.FileInfo) bool                 { return false }
func privateMetadata(path string, info os.FileInfo, dir bool) bool { return false }
func openRead(path string) (*os.File, error)                       { return nil, ErrUnsupported }
func createFile(path string) (*os.File, error)                     { return nil, ErrUnsupported }
func createDir(path string) error                                  { return ErrUnsupported }
func setMode(f *os.File, mode os.FileMode) error                   { return ErrUnsupported }
func renameFile(ctx context.Context, src, dst string) error        { return ErrUnsupported }
func prepareLock(path string) error                                { return ErrUnsupported }
func syncDir(path string) error                                    { return ErrUnsupported }

func openMode(path string) (*os.File, error)             { return nil, ErrUnsupported }
func targetParent(path string) error                     { return ErrUnsupported }
func trustedAncestor(path string, info os.FileInfo) bool { return false }
