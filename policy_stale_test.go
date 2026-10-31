package quill

import (
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A policy write that fails at Open must not leave the previous writer's quill.policy standing as
// the current settings: check, ls and an offline prune read that file as the writer's promise.
// The file is removed at once (so offline tools see none and refuse), and written again by the
// first retry: a flush, a prune pass or Close, whichever comes first.
func TestAFailedPolicyWriteAtOpenNeverLeavesThePreviousWritersSettings(t *testing.T) {
	for _, alsoRemove := range []bool{false, true} {
		name := "rename-only"
		if alsoRemove {
			name = "rename-and-remove"
		}
		t.Run(name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.Retention = time.Hour; c.MaxBytes = 4096 })
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			if p := readSimPolicy(t, s); p.MaxBytes != 4096 {
				t.Fatalf("setup: first run's policy %+v", p)
			}

			ffs := &policyFaultFS{FS: s, err: syscall.EIO, failRemove: alsoRemove}
			var lc logCapture
			l = openSim(t, s, func(c *Config) { c.Retention = 168 * time.Hour; c.Logger = lc.logger(); c.fs = ffs })
			if err := l.Err(); !errors.Is(err, syscall.EIO) {
				t.Fatalf("Err() after the failed policy write: %v", err)
			}
			if !alsoRemove {
				if _, err := s.Lstat("/log/quill.policy"); err == nil {
					t.Fatalf("the previous run's quill.policy is still on disk after the failed write: %+v", readSimPolicy(t, s))
				}
			}
			if !lc.has("quill.policy") {
				t.Fatalf("the failed policy write was not logged: %q", lc.msgs)
			}

			// The fault clears; a writer that stops before any prune pass still leaves its own settings.
			ffs.err = nil
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			if p := readSimPolicy(t, s); p.Retention != (168*time.Hour).String() || p.MaxBytes != 0 {
				t.Fatalf("after Close the policy is %+v, want the second run's 168h and no cap", p)
			}
		})
	}
}

// The retry also runs at a flush, and a recovered write clears the error it left in Err().
func TestAPolicyWriteRetriedAtFlushClearsItsError(t *testing.T) {
	s := fsx.NewSim()
	ffs := &policyFaultFS{FS: s, err: syscall.ENOSPC}
	l := openSim(t, s, func(c *Config) { c.MaxRecords = 77; c.fs = ffs })
	defer l.Close()
	if err := l.Err(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Err() after the failed policy write: %v", err)
	}
	if err := l.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := l.Err(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Err() while the retry still fails: %v", err)
	}
	ffs.err = nil
	if err := l.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err() after the policy was written: %v", err)
	}
	if p := readSimPolicy(t, s); p.MaxRecords != 77 {
		t.Fatalf("policy after the retry: %+v", p)
	}
}

// An idle writer has nothing to flush: its prune pass is what writes the policy again.
func TestAPolicyWriteRetriedAtPrunePass(t *testing.T) {
	s := fsx.NewSim()
	ffs := &policyFaultFS{FS: s, err: syscall.EIO}
	l := openSim(t, s, func(c *Config) { c.MaxRecords = 78; c.fs = ffs })
	defer l.Close()
	ffs.err = nil
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err() after the policy was written: %v", err)
	}
	if p := readSimPolicy(t, s); p.MaxRecords != 78 {
		t.Fatalf("policy after the prune pass: %+v", p)
	}
}

// policyFaultFS fails the rename onto quill.policy (and, with failRemove, its removal) with err
// while err is set; every other operation goes through.
type policyFaultFS struct {
	fsx.FS
	err        error
	failRemove bool
}

func (f *policyFaultFS) Rename(oldpath, newpath string) error {
	if f.err != nil && strings.HasSuffix(newpath, "/"+policyName) {
		_ = f.FS.Remove(oldpath)
		return f.err
	}
	return f.FS.Rename(oldpath, newpath)
}

func (f *policyFaultFS) Remove(name string) error {
	if f.err != nil && f.failRemove && strings.HasSuffix(name, "/"+policyName) {
		return f.err
	}
	return f.FS.Remove(name)
}

func readSimPolicy(t *testing.T, s *fsx.Sim) policy {
	t.Helper()
	p, ok := decodePolicy(fileBytes(t, s, "/log/quill.policy"))
	if !ok {
		t.Fatal("quill.policy does not parse")
	}
	return p
}
