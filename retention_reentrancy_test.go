package quill

import (
	"errors"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A prune pass chooses its deletion prefix against l.sealed, then publishes the marker before
// unlinking. That publish may seal the active segment and run the caps (decision 24), which
// delete a prefix of l.sealed through a nested call, so the pass must re-derive what is left
// of its choice afterwards (decision 93). The tests build the state that reaches it: a failed
// rotation (decision 60 d) leaves the active segment growing past SegmentBytes, so the next
// publish rotates and the caps fire while a prune pass is already committed to its prefix.

// openThreeSealed opens a log with MaxBytes set and three sealed segments of four records
// (1..12; 94, 110 and 129 bytes); segment 3 is active and empty.
func openThreeSealed(t *testing.T, s *fsx.Sim, clk *testClock, maxBytes int64) *Log {
	t.Helper()
	l := openRetention(t, s, clk, func(c *Config) {
		c.SegmentMaxAge = -1
		c.MaxBytes = maxBytes
	})
	fillSegments(t, l, clk, 3)
	return l
}

// flushWithRotationFailing flushes n segments' worth of records into the active segment while
// the next segment cannot be created: each Flush returns the injected error and leaves the
// publish pending (decision 60 d). The injection is cleared on return, so the next publish
// seals the oversized segment and runs the caps.
func flushWithRotationFailing(t *testing.T, s *fsx.Sim, l *Log, n int) {
	t.Helper()
	errCreate := errors.New("injected: cannot create the next segment")
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpCreate {
			return errCreate
		}
		return nil
	}
	for i := 0; i < n; i++ {
		appendN(t, l, 4)
		if err := l.Flush(); !errors.Is(err, errCreate) {
			t.Fatalf("flush %d while the next segment cannot be created: %v", i, err)
		}
	}
	s.Inject = nil
	l.mu.Lock()
	sealed, durable, pending := len(l.sealed), l.active.durable, l.pendingPublish
	l.mu.Unlock()
	if sealed != 3 || durable < int64(n)*l.cfg.SegmentBytes || !pending {
		t.Fatalf("setup: sealed=%d durable=%d pendingPublish=%v", sealed, durable, pending)
	}
}

// prune runs Prune(true) and reports a panic as a test failure rather than ending the binary.
func prune(t *testing.T, l *Log) (err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Prune panicked: %v", r)
		}
	}()
	return l.Prune(true)
}

func sealedIDs(l *Log) []uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	ids := make([]uint64, len(l.sealed))
	for i, s := range l.sealed {
		ids[i] = s.id
	}
	return ids
}

func TestAPrunePassDeletesOnlyTheSegmentsItChoseWhenItsPublishSealsAndTheCapsFire(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openThreeSealed(t, s, clk, 700) // about four segments
	defer l.Close()
	flushWithRotationFailing(t, s, l, 4) // records 13..28 in the oversized segment 3
	clk.Advance(2 * time.Hour)           // every sealed segment is expired; the oversized active one too
	if err := prune(t, l); err != nil {
		t.Fatal(err)
	}
	// The pass chose 0..2; the seal inside its publish ran the cap, which took all three.
	// Segment 3, the one just sealed, is the cap's remainder and is never the pass's to take.
	if ids := sealedIDs(l); len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("sealed %v, want [3]", ids)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 || ids[0] != 3 || ids[1] != 4 {
		t.Fatalf("segment files %v, want [3 4]", ids)
	}
	st, _ := l.Stat()
	if st.First != 13 || st.Head != 28 || st.Dropped != 0 {
		t.Fatalf("%+v", st)
	}
	recs, err := l.Read(13, 100)
	if err != nil || checkContiguous(t, recs, 13) != 28 {
		t.Fatalf("read 13..: %v %v", seqsOf(recs), err)
	}
}

func TestAPrunePassKeepsRecordsWrittenAfterTheCutoffWhenItsPublishSealsAndTheCapsFire(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openThreeSealed(t, s, clk, 400) // about two segments
	defer l.Close()
	clk.Advance(2 * time.Hour)           // segments 0..2 are expired
	flushWithRotationFailing(t, s, l, 1) // records 13..16, fresh, in the active segment 3
	if err := prune(t, l); err != nil {
		t.Fatal(err)
	}
	// The pass chose 0..2; its publish sealed segment 3 and the cap took 0 and 1, so the
	// pass deletes 2 alone. Segment 3 holds the fresh records and stays.
	if ids := sealedIDs(l); len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("sealed %v, want [3]", ids)
	}
	st, _ := l.Stat()
	if st.First != 13 || st.Head != 16 || st.Dropped != 0 {
		t.Fatalf("%+v", st)
	}
	recs, err := l.Read(13, 100)
	if err != nil || checkContiguous(t, recs, 13) != 16 {
		t.Fatalf("read 13..: %v %v", seqsOf(recs), err)
	}
	if _, err := l.Read(1, 1); !isGap(err, 1, 12) {
		t.Fatalf("below the first retained: %v", err)
	}
}

func TestAPrunePassDoesNotDropAProtectedConsumerBeyondWhatTheCapRequired(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openThreeSealed(t, s, clk, 950) // the cap needs one of the three expired segments
	defer l.Close()
	flushWithRotationFailing(t, s, l, 4) // records 13..28 in the oversized segment 3
	c, err := l.Consumer("protected", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Commit(16); err != nil { // read 1..15: segments 0..2 may go, the oversized one holds 16
		t.Fatal(err)
	}
	clk.Advance(2 * time.Hour)
	if err := prune(t, l); err != nil {
		t.Fatal(err)
	}
	// The pass chose 0..2 and the cap took 0; the pass deletes 1 and 2. The consumer's unread
	// segment 3 survives and nothing was dropped under it (decisions 24 and 57).
	if ids := sealedIDs(l); len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("sealed %v, want [3]", ids)
	}
	st, _ := l.Stat()
	if st.Dropped != 0 || st.First != 13 {
		t.Fatalf("%+v", st)
	}
	recs, err := l.Read(16, 1)
	if err != nil || len(recs) != 1 || recs[0].Seq != 16 {
		t.Fatalf("read 16: %v %v", seqsOf(recs), err)
	}
}
