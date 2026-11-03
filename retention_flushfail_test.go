package quill

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Deleting expired segments is the only way a full disk frees itself, so a prune pass whose age
// rotation cannot flush must skip the rotation, say so, and still run the sweep and the caps.
func TestAPrunePassWhoseAgeRotationCannotFlushStillSweepsAndLogsIt(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	var lc logCapture
	l := openRetention(t, s, clk, func(c *Config) {
		c.SegmentMaxAge = time.Minute
		c.Logger = lc.logger()
	})
	defer l.Close()
	fillSegments(t, l, clk, 3) // segments 0..2 sealed, 3 active and empty
	appendN(t, l, 1)
	mustFlush(t, l)            // the active segment holds a durable record
	clk.Advance(2 * time.Hour) // every segment is past Retention, the active one past SegmentMaxAge
	appendN(t, l, 1)           // a buffered record: the next flush writes
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			return syscall.ENOSPC
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("setup: flush while the disk is full: %v", err)
	}
	before := len(sealedIDs(l))
	if err := l.Prune(true); err != nil {
		t.Fatalf("Prune while flushes fail: %v", err)
	}
	if after := len(sealedIDs(l)); after >= before {
		t.Fatalf("expired segments were not deleted: sealed %d -> %d", before, after)
	}
	if !lc.has("age rotation skipped") {
		t.Fatalf("the skipped rotation was not logged: %q", lc.msgs)
	}
}

// A failed age rotation does not fail the pass or stop the sweep: the error stays in Err(), the
// expired segments are still deleted, and a later pass makes the rotation (decision 60 d).
func TestAPrunePassWhoseAgeRotationFailsStillSweepsAndALaterPassRotates(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = time.Minute })
	defer l.Close()
	fillSegments(t, l, clk, 3) // segments 0..2 sealed, 3 active and empty
	appendN(t, l, 1)
	mustFlush(t, l)            // the active segment holds a durable record
	clk.Advance(2 * time.Hour) // every segment is past Retention, the active one past SegmentMaxAge
	next := "/log/" + segmentName(4)
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpCreate && name == next {
			return syscall.ENOSPC
		}
		return nil
	}
	before := len(sealedIDs(l))
	if err := l.Prune(true); err != nil {
		t.Fatalf("Prune while the next segment cannot be created: %v", err)
	}
	if err := l.Err(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Err() after the failed rotation: %v, want ENOSPC", err)
	}
	if after := len(sealedIDs(l)); after >= before {
		t.Fatalf("expired segments were not deleted: sealed %d -> %d", before, after)
	}
	if ids := segmentFiles(t, s, "/log"); ids[len(ids)-1] != 3 {
		t.Fatalf("segments %v: the failed rotation must have created nothing", ids)
	}
	s.Inject = nil
	if err := l.Prune(true); err != nil {
		t.Fatalf("the later pass: %v", err)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err() after the later pass: %v", err)
	}
	if ids := segmentFiles(t, s, "/log"); ids[len(ids)-1] != 4 {
		t.Fatalf("segments %v: the later pass did not rotate to segment 4", ids)
	}
}
