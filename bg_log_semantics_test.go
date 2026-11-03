package quill

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A prune pass that fails at the publish before its deletions retried no unlink: it must not log
// that the unlink recovered, and the pass that does delete must (Stage 2 audit A5-1).
func TestAPrunePassWhoseMarkerPublishFailsLogsNoRecoveryForTheUnlink(t *testing.T) {
	var lc logCapture
	s, clk, l := retentionLog(t, func(c *Config) { c.Logger = lc.logger() })
	defer l.Close()
	fillSegments(t, l, clk, 4)
	clk.Advance(2 * time.Hour)
	unlinkFails := true
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpRemove && name == "/log/"+segmentName(0) && unlinkFails {
			return syscall.EIO
		}
		return nil
	}
	if err := l.Prune(true); err == nil {
		t.Fatal("prune: want the unlink error (decision 65)")
	}
	if n := lc.countMsgs("level=WARN", "op=\"segment delete\""); n != 1 {
		t.Fatalf("%d unlink warnings, want 1", n)
	}

	// A flush whose marker publish fails leaves the publish pending, so the next pass must
	// publish before it deletes, and that publish fails too.
	appendN(t, l, 1)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpRename {
			return errENOSPC
		}
		return nil
	}
	_ = l.Flush()
	if err := l.Prune(true); err == nil {
		t.Fatal("prune: want the publish error")
	}
	if n := lc.countMsgs("level=INFO", "op=\"segment delete\""); n != 0 {
		t.Fatalf("recovery of the unlink logged by a pass that never retried it: %d", n)
	}
	if n := lc.countMsgs("level=WARN", "op=\"segment delete\""); n != 1 {
		t.Fatalf("%d unlink warnings after the publish failure, want still 1", n)
	}

	// Once the publish works and the unlink succeeds, the one recovery is logged.
	s.Inject = nil
	unlinkFails = false
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if n := lc.countMsgs("level=INFO", "op=\"segment delete\""); n != 1 {
		t.Fatalf("%d unlink recovery lines, want 1", n)
	}
	if n := lc.countMsgs("level=WARN", "op=\"segment delete\""); n != 1 {
		t.Fatalf("%d unlink warnings, want 1", n)
	}
}

// LastFlushOK is the time of the last flush that wrote and published; it is not the age of
// anything. A healthy idle log keeps an old one, and one Append after a quiet hour shows one
// buffered record, no failure and an hour-old LastFlushOK. That is why the documented alert is
// FlushFailing and not time.Since(LastFlushOK) (Stage 2 audit A5-1).
func TestLastFlushOKDoesNotAdvanceOnAnIdleLogAndFlushFailingStaysFalse(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	s.Clock = clk.Now
	l := openSim(t, s, func(c *Config) { c.clock = clk.Now })
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)
	wrote := clk.Now()

	// A quiet hour with nothing to flush, flushed over and over.
	clk.Advance(time.Hour)
	for i := 0; i < 3; i++ {
		mustFlush(t, l)
	}
	st, _ := l.Stat()
	if st.FlushFailing || st.BufferedRecords != 0 || !st.LastFlushOK.Equal(wrote) {
		t.Fatalf("idle log: %+v (LastFlushOK %v, want %v)", st, st.LastFlushOK, wrote)
	}

	// One Append now: buffered, not failing, and LastFlushOK is an hour old.
	appendN(t, l, 1)
	st, _ = l.Stat()
	if st.BufferedRecords != 1 || st.FlushFailing {
		t.Fatalf("one append after the quiet hour: %+v", st)
	}
	if age := clk.Now().Sub(st.LastFlushOK); age != time.Hour {
		t.Fatalf("time.Since(LastFlushOK) = %v, want the quiet hour; it is not how long the record has waited", age)
	}
}

// A flush whose write succeeded but whose marker did not publish has not completed: LastFlushOK
// waits for the publish (the retry that lands it counts), and FlushFailing stays false because
// the write did not fail; Err() carries the publish error.
func TestLastFlushOKWaitsForTheMarkerPublish(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	s.Clock = clk.Now
	l := openSim(t, s, func(c *Config) { c.clock = clk.Now })
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)
	wrote := clk.Now()

	clk.Advance(time.Minute)
	appendN(t, l, 1)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpRename {
			return errENOSPC
		}
		return nil
	}
	if err := l.Flush(); err == nil {
		t.Fatal("flush: want the publish error")
	}
	st, _ := l.Stat()
	if st.BufferedRecords != 0 || st.FlushFailing || !st.LastFlushOK.Equal(wrote) {
		t.Fatalf("written but not published: %+v", st)
	}
	if l.Err() == nil {
		t.Fatal("Err() is nil while the marker cannot be published")
	}

	s.Inject = nil
	clk.Advance(time.Minute)
	retried := clk.Now()
	mustFlush(t, l)
	st, _ = l.Stat()
	if !st.LastFlushOK.Equal(retried) {
		t.Fatalf("after the publish landed: LastFlushOK %v, want %v", st.LastFlushOK, retried)
	}
}

// A rotation that failed after a good flush is retried by later flushes of an idle log. Those
// retries wrote nothing, so they leave LastFlushOK at the flush that did, whether they fail or
// land the rotation: the time stays the time of the last successful write.
func TestLastFlushOKIsNotStampedByTheRetriesOfAFailedRotationOnAnIdleLog(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	s.Clock = clk.Now
	l := openSim(t, s, func(c *Config) { c.clock = clk.Now; c.SegmentBytes = 100 })
	defer l.Close()
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpCreate && name == "/log/"+segmentName(1) {
			return errENOSPC
		}
		return nil
	}
	clk.Advance(time.Minute)
	appendN(t, l, 8)
	_ = l.Flush() // the write and the publish succeed; the rotation it triggers fails
	wrote := clk.Now()
	st, _ := l.Stat()
	if l.Err() == nil || !st.LastFlushOK.Equal(wrote) || st.BufferedRecords != 0 {
		t.Fatalf("after the write: Err %v, %+v (want LastFlushOK %v and a failing rotation)", l.Err(), st, wrote)
	}

	clk.Advance(time.Minute)
	if err := l.Flush(); err == nil {
		t.Fatal("idle retry with the rotation still failing: want an error")
	}
	if st, _ = l.Stat(); !st.LastFlushOK.Equal(wrote) {
		t.Fatalf("a failing idle retry moved LastFlushOK to %v, want %v", st.LastFlushOK, wrote)
	}

	s.Inject = nil
	clk.Advance(time.Minute)
	mustFlush(t, l) // the idle retry that lands the rotation
	if l.Err() != nil {
		t.Fatalf("rotation landed but Err() = %v", l.Err())
	}
	if st, _ = l.Stat(); !st.LastFlushOK.Equal(wrote) {
		t.Fatalf("an idle retry that landed a rotation moved LastFlushOK to %v, want %v: it wrote nothing", st.LastFlushOK, wrote)
	}
}

// "Recovered" means a pass completed without a failure, not that the failing step was retried: a
// prune pass that finds nothing to delete after a failed unlink (the segment became protected) logs
// the one Info and clears Err(), though the segment is still on disk. Documented in decision 60 and
// the user guide; pinned here so a change to it is a decision (Stage 2 audit A5-1).
func TestAnIdlePrunePassAfterAFailedUnlinkLogsRecoveryAndClearsErrWithoutRetryingIt(t *testing.T) {
	var lc logCapture
	s, clk, l := retentionLog(t, func(c *Config) { c.Logger = lc.logger() })
	defer l.Close()
	fillSegments(t, l, clk, 4)
	clk.Advance(2 * time.Hour)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpRemove && name == "/log/"+segmentName(0) {
			return syscall.EIO
		}
		return nil
	}
	if err := l.Prune(true); err == nil {
		t.Fatal("prune: want the unlink error")
	}
	if l.Err() == nil || lc.countMsgs("level=WARN", "op=\"segment delete\"") != 1 {
		t.Fatalf("after the failed unlink: Err %v, %d warnings", l.Err(), lc.countMsgs("level=WARN", "op=\"segment delete\""))
	}

	// A protected consumer at the first sequence makes every segment ineligible: the next pass has
	// nothing to delete, so the failing unlink is never retried.
	c, err := l.Consumer("keep", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := l.Prune(true); err != nil {
		t.Fatalf("the idle pass: %v", err)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err() after a pass that completed without a failure: %v", err)
	}
	if n := lc.countMsgs("level=INFO", "op=\"segment delete\""); n != 1 {
		t.Fatalf("%d recovery lines, want 1", n)
	}
	if _, err := s.Lstat("/log/" + segmentName(0)); err != nil {
		t.Fatalf("the segment whose unlink failed is gone: %v", err)
	}
}

// FlushFailing stays false once the log has failed, as Stats documents, even when an earlier
// write failure had set it: a write error, then a fatal data fsync error (Stage 2 audit L1-14).
func TestFlushFailingIsFalseOnceTheLogHasFailedAfterAnEarlierWriteFailure(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)
	appendN(t, l, 1)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpWrite {
			return errENOSPC
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, errENOSPC) {
		t.Fatalf("flush with a failing write: %v", err)
	}
	if st, _ := l.Stat(); !st.FlushFailing || st.BufferedRecords != 1 {
		t.Fatalf("after the write failure: %+v", st)
	}
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpSync {
			return syscall.EIO
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("flush with a failing fsync: want ErrFailed, got %v", err)
	}
	if !errors.Is(l.Err(), ErrFailed) {
		t.Fatalf("Err %v: want ErrFailed", l.Err())
	}
	if st, _ := l.Stat(); st.FlushFailing {
		t.Fatalf("FlushFailing is true on a failed log: %+v", st)
	}
}
