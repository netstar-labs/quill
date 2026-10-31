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
