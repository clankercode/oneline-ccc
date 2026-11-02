package codex

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
)

type privateStage struct {
	path     string
	identity os.FileInfo
}

func (a *Adapter) stage() (*privateStage, error) {
	if err := configtxn.PrivateDir(a.Scratch); err != nil {
		return nil, err
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, errors.New("Cannot create private Codex staging directory")
	}
	path := filepath.Join(a.Scratch, "codex-stage-"+hex.EncodeToString(id))
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("Cannot allocate a fresh Codex staging directory")
	}
	if err := configtxn.PrivateDir(path); err != nil {
		return nil, err
	}
	// File.Stat captures Windows file identity from its handle immediately;
	// Lstat can defer that lookup until SameFile, after a path substitution.
	directory, err := os.Open(path)
	if err != nil {
		return nil, errors.New("Cannot verify private Codex staging directory")
	}
	info, statErr := directory.Stat()
	closeErr := directory.Close()
	if statErr != nil || closeErr != nil {
		return nil, errors.New("Cannot verify private Codex staging directory")
	}
	return &privateStage{path: path, identity: info}, nil
}

func (s *privateStage) close() error {
	const message = "Cannot remove owned Codex staging directory; reconcile application staging state"
	info, err := os.Lstat(s.path)
	if err != nil || !info.IsDir() || !os.SameFile(s.identity, info) {
		return errors.New(message)
	}
	// Never adopt a substituted directory or follow a new reparse/symlink, and
	// never retry deleting a path whose ownership might have changed. RPC has
	// already drained its owned process tree before this single removal.
	if configtxn.PrivateDir(s.path) != nil || os.RemoveAll(s.path) != nil {
		return errors.New(message)
	}
	return nil
}
