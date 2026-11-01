//go:build darwin

package main

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// An unlink that fails leaves the segment, so the pass did less than asked: quill prune names
// the segment on stderr, prints the counts and exits 1 (A5-36). The immutable flag makes unlink
// fail with EPERM for the owner, which is how a segment the process may not delete looks in practice.
func TestPruneExitsOneWhenAnUnlinkFails(t *testing.T) {
	dir := expiredLog(t)
	seg := filepath.Join(dir, "0000000000.log")
	if err := syscall.Chflags(seg, 2); err != nil { // UF_IMMUTABLE
		t.Skipf("chflags: %v", err)
	}
	defer syscall.Chflags(seg, 0)
	code, out, errw := exec(t, "", "prune", "-dir", dir, "-retention", "1s", "-force")
	if code != exitError {
		t.Fatalf("exit %d, want %d: %s%s", code, exitError, out, errw)
	}
	if !strings.Contains(errw, "cannot delete segment") || !strings.Contains(errw, "0000000000.log") {
		t.Fatalf("the failed unlink is not named: %q", errw)
	}
	if !strings.Contains(out, "segments 10 -> 10") {
		t.Fatalf("the counts line is missing: %q", out)
	}
}
