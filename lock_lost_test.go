package quill

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A quill.lock deleted under a live writer turns the lock into a no-op: a second writer then
// opens without ErrLocked. The first writer must notice that the file it holds is no longer the
// file at the path and fail (ErrFailed) instead of acknowledging records the second one will
// overwrite.
func TestWriterFailsWhenItsLockFileIsRemovedAndAnotherWriterOpens(t *testing.T) {
	s := fsx.NewSim()
	a := openSim(t, s, nil)
	defer a.Close()
	appendN(t, a, 1)
	mustFlush(t, a)

	if err := s.Remove("/log/" + lockName); err != nil {
		t.Fatal(err)
	}
	b := openSim(t, s, nil) // the lock is a no-op now: this open succeeds
	defer b.Close()

	if _, err := a.Append([]byte("from-a")); err != nil {
		t.Fatalf("append buffers before the check: %v", err)
	}
	if err := a.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Flush after the lock file was removed: %v, want ErrFailed", err)
	}
	if _, err := a.Append([]byte("again")); !errors.Is(err, ErrFailed) {
		t.Fatalf("Append on the failed log: %v, want ErrFailed", err)
	}
	if err := a.Err(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Err: %v, want ErrFailed", err)
	}
}

// The same on a real file system, where the held descriptor is compared with the path by
// device and inode, and for a lock file that is replaced rather than only removed.
func TestWriterFailsWhenItsLockFileIsReplacedOnDisk(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "log")
	a := openDir(t, dir, nil)
	defer a.Close()
	appendN(t, a, 1)
	mustFlush(t, a)

	lock := filepath.Join(dir, lockName)
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, nil, 0o600); err != nil { // a restore over the directory
		t.Fatal(err)
	}
	if _, err := a.Append([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := a.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Flush after the lock file was replaced: %v, want ErrFailed", err)
	}
}

// Prune is a retention pass that deletes data; it must not run for a writer that lost its lock.
func TestPruneFailsWhenTheLockFileIsRemoved(t *testing.T) {
	s := fsx.NewSim()
	a := openSim(t, s, nil)
	defer a.Close()
	appendN(t, a, 1)
	mustFlush(t, a)
	if err := s.Remove("/log/" + lockName); err != nil {
		t.Fatal(err)
	}
	if err := a.Prune(false); !errors.Is(err, ErrFailed) {
		t.Fatalf("Prune after the lock file was removed: %v, want ErrFailed", err)
	}
}
