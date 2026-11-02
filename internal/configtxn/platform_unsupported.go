//go:build !linux && !darwin

package configtxn

import "os"

func platformSupported() bool     { return false }
func safeOpenFlags() int          { return 0 }
func owned(info os.FileInfo) bool { return false }
