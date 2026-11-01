//go:build linux || darwin

package quill

import (
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// lockRetry is how long takeLock keeps trying a held lock before returning ErrLocked: long
// enough to ride over a predecessor's Close, short enough that a second writer fails fast.
const lockRetry = 200 * time.Millisecond

// writerLockPerm is the mode of quill.lock, whatever Config.Perm says (decision 51). flock
// needs no write access to the descriptor, so a lock file readable by a reader-only process
// would let it take the lock and keep every writer Open failing with ErrLocked. Only the
// writer and Archive's destination probe touch this file. Consumer locks keep the log's mode,
// because a group consumer must take its own.
const writerLockPerm = fs.FileMode(0o600)

// takeLock creates path if needed and takes an exclusive flock on it, retrying briefly. The
// returned file holds the lock until it is closed or the process ends; the file is never
// deleted (decision 41).
func takeLock(fsys fsx.FS, path string, perm fs.FileMode) (fsx.File, error) {
	f, err := fsys.OpenFile(path, fsx.O_RDWR|fsx.O_CREATE, perm)
	if err != nil {
		return nil, err
	}
	return holdLock(f, path)
}

// takeWriterLock is takeLock for quill.lock: created 0600, and a lock an earlier build left
// wider is chmodded to 0600 before the flock is tried, so a restart after a hold closes the
// hole too. Only the file's owner can chmod it. A failure does not fail the open: the lock,
// once held, is what keeps the log single-writer, so refusing would turn a hardening step
// into an outage. warn is told of it instead.
func takeWriterLock(fsys fsx.FS, path string, warn func(error)) (fsx.File, error) {
	f, err := fsys.OpenFile(path, fsx.O_RDWR|fsx.O_CREATE, writerLockPerm)
	if err != nil {
		return nil, err
	}
	if info, err := f.Stat(); err != nil || info.Mode().Perm() != writerLockPerm {
		if err := f.Chmod(writerLockPerm); err != nil {
			warn(err)
		}
	}
	return holdLock(f, path)
}

// holdLock takes the flock on the open file f, retrying briefly, and closes f on failure.
func holdLock(f fsx.File, path string) (fsx.File, error) {
	deadline := time.Now().Add(lockRetry)
	for {
		err := f.TryLock()
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, fsx.ErrLocked) {
			_ = f.Close()
			return nil, err
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("%w: %s", ErrLocked, path)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
