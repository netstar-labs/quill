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

// takeLock creates path if needed and takes an exclusive flock on it, retrying briefly. The
// returned file holds the lock until it is closed or the process ends; the file is never
// deleted (decision 41).
func takeLock(fsys fsx.FS, path string, perm fs.FileMode) (fsx.File, error) {
	f, err := fsys.OpenFile(path, fsx.O_RDWR|fsx.O_CREATE, perm)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(lockRetry)
	for {
		err = f.TryLock()
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
