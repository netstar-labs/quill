package quill

import (
	"errors"
	"io/fs"
	"strings"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// holdLock documents that it closes the file it was given when the lock is refused. A
// supervisor that probes for the writer slot by retrying Open against a live writer must not
// lose a descriptor per attempt, whether the refusal is ErrLocked or another TryLock error.

func TestAnOpenRefusedByALiveWritersLockLeaksNoDescriptor(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	before := s.OpenFiles()
	for i := 0; i < 3; i++ {
		if second, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrLocked) {
			if err == nil {
				_ = second.Close()
			}
			t.Fatalf("second Open: %v, want ErrLocked", err)
		}
	}
	if after := s.OpenFiles(); after != before {
		t.Fatalf("%d descriptors open after three refused Opens, were %d", after, before)
	}
}

// tryLockFailFS makes TryLock on the writer's lock file fail with an error that is not
// ErrLocked.
type tryLockFailFS struct{ fsx.FS }

type tryLockFailFile struct{ fsx.File }

func (f tryLockFailFS) OpenFile(name string, flag int, perm fs.FileMode) (fsx.File, error) {
	file, err := f.FS.OpenFile(name, flag, perm)
	if err != nil || !strings.HasSuffix(name, "/"+lockName) {
		return file, err
	}
	return tryLockFailFile{file}, nil
}

func (tryLockFailFile) TryLock() error {
	return &fs.PathError{Op: "flock", Path: lockName, Err: syscall.ENOLCK}
}

func TestAnOpenWhoseLockFailsWithAnotherErrorLeaksNoDescriptor(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	before := s.OpenFiles()
	for i := 0; i < 3; i++ {
		cfg := simConfig(s, "/log")
		cfg.fs = tryLockFailFS{s}
		second, err := Open(cfg)
		if err == nil {
			_ = second.Close()
			t.Fatal("Open succeeded although the lock failed")
		}
		if !errors.Is(err, syscall.ENOLCK) || errors.Is(err, ErrLocked) {
			t.Fatalf("Open: %v, want the TryLock error itself", err)
		}
	}
	if after := s.OpenFiles(); after != before {
		t.Fatalf("%d descriptors open after three failed Opens, were %d", after, before)
	}
}
