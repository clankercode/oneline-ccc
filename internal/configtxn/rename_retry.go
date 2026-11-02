package configtxn

import (
	"context"
	"os"
	"time"
)

type renameChecks struct {
	before func() error
	retry  func() error
}

// The bounded Windows sharing algorithm lives here so deterministic syscall
// boundary tests can exercise it without claiming native Windows filesystem proof.
func renameRetry(ctx context.Context, src, dst, hash string, mode os.FileMode, move func() error, sharing func(error) bool, checks renameChecks) error {
	before, err := Read(dst)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 8; attempt++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = verifySource(src, hash, mode); err != nil {
			return ErrConflict
		}
		if err = expected(before); err != nil {
			return err
		}
		if checks.before != nil {
			if err = checks.before(); err != nil {
				return err
			}
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		err = move()
		if err == nil {
			return nil
		}
		if !sharing(err) {
			return ErrIO
		}
		if checks.retry != nil {
			if err = checks.retry(); err != nil {
				return err
			}
		}
		if attempt < 7 {
			timer := time.NewTimer(20 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return ErrIO
}

func verifySource(path, hash string, mode os.FileMode) error {
	if err := verifyReplacement(path, hash, mode); err != nil {
		return ErrConflict
	}
	if sourceMetadata(path) != nil {
		return ErrConflict
	}
	return nil
}
