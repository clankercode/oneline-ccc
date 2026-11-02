package monitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/clankercode/oneline-ccc/internal/configtxn"
)

type configObservation struct {
	Exists     bool      `json:"exists"`
	SHA256     string    `json:"sha256,omitempty"`
	ModifiedAt time.Time `json:"modified_at"`
}
type observation struct {
	Version    int               `json:"version"`
	ObservedAt time.Time         `json:"observed_at"`
	Config     configObservation `json:"config"`
}

func (m *Manager) sample(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !supportedFiles() {
		return errUnavailable
	}
	if !m.validRoots() {
		return errUnsafe
	}
	path := filepath.Join(m.codexHome, "config.toml")
	if safeComponents(path) != nil {
		return errUnsafe
	}
	before, err := os.Lstat(path)
	record := observation{Version: 1, ObservedAt: m.clock()}
	if err == nil {
		if !before.Mode().IsRegular() {
			return errUnsafe
		}
		snap, readErr := configtxn.Read(path)
		if readErr != nil || !snap.Existed {
			return errUnsafe
		}
		after, statErr := os.Lstat(path)
		if statErr != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || before.Mode() != after.Mode() {
			return errUnsafe
		}
		hash := sha256.Sum256(snap.Before)
		record.Config = configObservation{Exists: true, SHA256: hex.EncodeToString(hash[:]), ModifiedAt: before.ModTime().UTC()}
	} else if !errors.Is(err, os.ErrNotExist) {
		return errIO
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if data, exists, err := privateRead(filepath.Join(m.root(), "observation.json")); err != nil {
		return err
	} else if exists {
		var prior observation
		if decodeRecord(data, &prior) != nil || !validObservation(prior) {
			return errUnsafe
		}
	}
	if configtxn.PrivateDir(m.stateRoot) != nil {
		return errUnsafe
	}
	return atomicRecord(ctx, filepath.Join(m.root(), "observation.json"), record)
}

// Observe reads only the selected config.toml; credentials, journals, backups,
// client settings and service managers are never read or changed by this loop.
// Output contains only fixed-schema hashes/existence/timestamps, atomically
// replaced without a retained history. Once returns after one sample.
func (m *Manager) Observe(ctx context.Context, once bool) error {
	if err := m.sample(ctx); err != nil {
		return err
	}
	if once {
		return nil
	}
	interval := m.interval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := m.sample(ctx); err != nil {
				return err
			}
		}
	}
}

func validObservation(record observation) bool {
	if record.Version != 1 || record.ObservedAt.IsZero() {
		return false
	}
	if !record.Config.Exists {
		return record.Config.SHA256 == "" && record.Config.ModifiedAt.IsZero()
	}
	digest, err := hex.DecodeString(record.Config.SHA256)
	return err == nil && len(digest) == 32 && !record.Config.ModifiedAt.IsZero()
}
