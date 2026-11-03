package quill

import (
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// protectedConsumerAt registers a protected consumer and, when next > 1, commits it there.
func protectedConsumerAt(t *testing.T, l *Log, name string, next uint64) {
	t.Helper()
	c, err := l.Consumer(name, ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	if next > 1 {
		if err := c.Commit(next); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

// With several protected consumers the age sweep stops at the slowest one, whichever of them
// the pass reads first (decision 45): a pass that kept the fastest consumer's offset instead
// would delete records the slow one has not read. Consumers are read in name order, so each
// order of slow and fast is a name pair.
func TestTheSlowestProtectedConsumerHoldsTheAgeSweepInEitherFileOrder(t *testing.T) {
	for _, names := range [][2]string{{"a-slow", "b-fast"}, {"b-slow", "a-fast"}} {
		slow, fast := names[0], names[1]
		s, clk, l := retentionLog(t, nil)
		protectedConsumerAt(t, l, slow, 1)
		fillSegments(t, l, clk, 6) // segments 0..5 sealed, 6 active and empty: records 1..24
		protectedConsumerAt(t, l, fast, 21)
		clk.Advance(2 * time.Hour) // every sealed segment is expired
		if err := l.Prune(true); err != nil {
			t.Fatal(err)
		}
		if ids := segmentFiles(t, s, "/log"); len(ids) != 7 {
			t.Fatalf("slow %q fast %q: segments %v, want all 7 kept for the consumer at 1", slow, fast, ids)
		}
		if st, _ := l.Stat(); st.Dropped != 0 || st.First != 1 {
			t.Fatalf("slow %q fast %q: %+v", slow, fast, st)
		}
		_ = l.Close()
	}
}

// A segment whose last record is the protected consumer's offset still holds a record it has
// not read (an offset is the next sequence to read), so the sweep keeps that segment.
func TestAProtectedConsumerAtTheLastRecordOfASegmentKeepsThatSegment(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 3) // segments 0..2 sealed: records 1..4, 5..8, 9..12
	protectedConsumerAt(t, l, "billing", 4)
	clk.Advance(2 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 4 || ids[0] != 0 {
		t.Fatalf("segments %v, want segment 0 kept (record 4 is unread)", ids)
	}
	if st, _ := l.Stat(); st.Dropped != 0 || st.First != 1 {
		t.Fatalf("%+v", st)
	}
}

// A damaged segment's upper bound is the sequence before the next good segment's first, so a
// protected consumer at exactly that next sequence no longer needs the damaged segment.
func TestADamagedSegmentBelowAProtectedConsumersOffsetIsDeletedWithTheExpiredOnes(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, nil)
	fillSegments(t, l, clk, 3) // segments 0..2 sealed: records 1..4, 5..8, 9..12
	protectedConsumerAt(t, l, "billing", 9)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	seg := "/log/" + segmentName(1)
	data := fileBytes(t, s, seg)
	data[len(data)-1] ^= 0xff
	writeFile(t, s, seg, data)
	l = openRetention(t, s, clk, nil)
	defer l.Close()
	if st, err := l.Stat(); err != nil || len(st.Holes) != 1 {
		t.Fatalf("setup: holes %+v %v", st.Holes, err)
	}
	clk.Advance(2 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 2 || ids[0] != 2 {
		t.Fatalf("segments %v, want 0 and the damaged 1 deleted and 2 kept", ids)
	}
}

// When the consumers directory cannot be listed the pass cannot tell which consumers are
// protected, so it protects every segment from the age sweep (retention.go, consumers).
func TestAnUnlistableConsumersDirectoryProtectsEverySegment(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 4)
	clk.Advance(2 * time.Hour)
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpReadDir && name == "/log/consumers" {
			return syscall.EACCES
		}
		return nil
	}
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	s.Inject = nil
	if ids := segmentFiles(t, s, "/log"); len(ids) != 5 {
		t.Fatalf("segments %v: the sweep ran with the consumers unseen", ids)
	}
	if st, _ := l.Stat(); st.Dropped != 0 || st.First != 1 {
		t.Fatalf("%+v", st)
	}
}
