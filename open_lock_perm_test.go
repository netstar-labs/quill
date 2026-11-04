package quill

import (
	"errors"
	"io/fs"
	"strings"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A writable Open by an account that cannot open the 0600 quill.lock another account created
// returns the permission error itself, naming quill.lock, and not ErrLocked: the lock was never
// tried, so quill cannot say a writer is live (decision 51, L3b-7). The seam fails the open of
// the lock file with EACCES, as the kernel does for another uid without root, while a first
// writer holds the log.
func TestOpenUnderAnAccountThatCannotOpenTheLockIsAPermissionErrorNotErrLocked(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer func() { _ = l.Close() }()
	s.Inject = func(op fsx.Op, name string) error {
		if (op == fsx.OpOpen || op == fsx.OpCreate) && name == "/log/"+lockName {
			return &fs.PathError{Op: "open", Path: name, Err: syscall.EACCES}
		}
		return nil
	}
	l2, err := Open(simConfig(s, "/log"))
	s.Inject = nil
	if err == nil {
		_ = l2.Close()
		t.Fatal("second Open under an unopenable lock succeeded")
	}
	if errors.Is(err, ErrLocked) {
		t.Fatalf("error matches ErrLocked: %v", err)
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("error is not a permission error: %v", err)
	}
	if !strings.Contains(err.Error(), lockName) {
		t.Fatalf("error does not name %s: %v", lockName, err)
	}
	// Same account, lock openable: the live writer is reported as ErrLocked.
	if l3, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrLocked) {
		if err == nil {
			_ = l3.Close()
		}
		t.Fatalf("second Open with an openable lock: %v, want ErrLocked", err)
	}
}
