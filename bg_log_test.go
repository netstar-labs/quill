package quill

import (
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A retryable background failure is logged once when it starts and once when it ends, never per
// retry (Stage 2 audit A5-1).

// countMsgs counts the log lines that contain every one of subs.
func (c *logCapture) countMsgs(subs ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, m := range c.msgs {
		ok := true
		for _, s := range subs {
			ok = ok && strings.Contains(m, s)
		}
		if ok {
			n++
		}
	}
	return n
}

// wantTransitions checks that op logged one Warn when it began to fail and, after clear ran, one
// Info when it recovered, however many retries failed in between.
func wantTransitions(t *testing.T, lc *logCapture, op string, retry func(), clear func()) {
	t.Helper()
	opAttr := "op=" + op // the text handler quotes a value with a space
	if strings.Contains(op, " ") {
		opAttr = "op=\"" + op + "\""
	}
	if n := lc.countMsgs("level=WARN", opAttr); n != 1 {
		t.Fatalf("%s: %d warnings after the first failure, want 1", op, n)
	}
	for i := 0; i < 3; i++ {
		retry()
	}
	if n := lc.countMsgs("level=WARN", opAttr); n != 1 {
		t.Fatalf("%s: %d warnings after three failed retries, want still 1", op, n)
	}
	if n := lc.countMsgs("level=INFO", opAttr); n != 0 {
		t.Fatalf("%s: recovery logged while still failing", op)
	}
	clear()
	if n := lc.countMsgs("level=INFO", opAttr); n != 1 {
		t.Fatalf("%s: %d recovery lines, want 1", op, n)
	}
	if n := lc.countMsgs("level=WARN", opAttr); n != 1 {
		t.Fatalf("%s: %d warnings after recovery, want 1", op, n)
	}
}

func TestAFailingFlushWriteIsLoggedOnEntryAndOnRecoveryNotPerRetry(t *testing.T) {
	var lc logCapture
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.Logger = lc.logger() })
	defer l.Close()
	appendN(t, l, 3)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpWrite && name == "/log/"+segmentName(0) {
			return syscall.EIO
		}
		return nil
	}
	_ = l.Flush()
	wantTransitions(t, &lc, "flush",
		func() { _ = l.Flush() },
		func() { s.Inject = nil; mustFlush(t, l) })
}

func TestAFailingMarkerPublishIsLoggedOnEntryAndOnRecoveryNotPerRetry(t *testing.T) {
	var lc logCapture
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.Logger = lc.logger() })
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	appendN(t, l, 3)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpRename {
			return errENOSPC
		}
		return nil
	}
	_ = l.Flush()
	wantTransitions(t, &lc, "marker publish",
		func() { _ = l.Flush() },
		func() { s.Inject = nil; mustFlush(t, l) })
}

func TestAFailingRotationIsLoggedOnEntryAndOnRecoveryNotPerRetry(t *testing.T) {
	var lc logCapture
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100; c.Logger = lc.logger() })
	defer l.Close()
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpCreate && name == "/log/"+segmentName(1) {
			return errENOSPC
		}
		return nil
	}
	appendN(t, l, 8)
	_ = l.Flush()
	wantTransitions(t, &lc, "rotation",
		func() { _ = l.Flush() },
		func() { s.Inject = nil; mustFlush(t, l) })
}

func TestAFailingPruneUnlinkIsLoggedOnEntryAndOnRecoveryNotPerPass(t *testing.T) {
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
		t.Fatal("prune: want the unlink error (decision 65)")
	}
	wantTransitions(t, &lc, "segment delete",
		func() { _ = l.Prune(true) },
		func() {
			s.Inject = nil
			if err := l.Prune(true); err != nil {
				t.Fatal(err)
			}
		})
}

func TestStatReportsWhatIsBufferedAndWhetherFlushesAreFailing(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	s.Clock = clk.Now
	l := openSim(t, s, func(c *Config) { c.clock = clk.Now })
	defer l.Close()
	opened := clk.Now()
	st, err := l.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if st.BufferedRecords != 0 || st.BufferedBytes != 0 || st.FlushFailing || !st.LastFlushOK.Equal(opened) {
		t.Fatalf("a fresh log: %+v", st)
	}

	clk.Advance(time.Minute)
	appendN(t, l, 3)
	st, _ = l.Stat()
	if st.BufferedRecords != 3 || st.BufferedBytes <= 0 || st.FlushFailing || !st.LastFlushOK.Equal(opened) {
		t.Fatalf("three records buffered: %+v", st)
	}
	bytes3 := st.BufferedBytes

	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpWrite && name == "/log/"+segmentName(0) {
			return syscall.EIO
		}
		return nil
	}
	clk.Advance(time.Minute)
	if err := l.Flush(); err == nil {
		t.Fatal("flush: want the injected error")
	}
	st, _ = l.Stat()
	if st.BufferedRecords != 3 || st.BufferedBytes != bytes3 || !st.FlushFailing || !st.LastFlushOK.Equal(opened) {
		t.Fatalf("after a failed flush: %+v", st)
	}

	s.Inject = nil
	clk.Advance(time.Minute)
	flushedAt := clk.Now()
	mustFlush(t, l)
	st, _ = l.Stat()
	if st.BufferedRecords != 0 || st.BufferedBytes != 0 || st.FlushFailing || !st.LastFlushOK.Equal(flushedAt) {
		t.Fatalf("after a good flush: %+v", st)
	}
	if st.Head != 3 {
		t.Fatalf("head %d", st.Head)
	}
}
