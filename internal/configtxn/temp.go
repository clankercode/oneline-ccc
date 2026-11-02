package configtxn

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
)

func privateTemp(dir, prefix string) (*os.File, error) {
	for i := 0; i < 10; i++ {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			return nil, ErrIO
		}
		f, err := createFile(filepath.Join(dir, prefix+hex.EncodeToString(id)))
		if err == nil {
			return f, nil
		}
		if !os.IsExist(err) {
			return nil, ErrIO
		}
	}
	return nil, ErrIO
}
