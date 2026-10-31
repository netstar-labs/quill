package quill

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

func TestContiguityTable(t *testing.T) {
	r := func(id, first, last uint64) seqRange { return seqRange{id: id, first: first, last: last} }
	for _, tc := range []struct {
		name     string
		rs       []seqRange
		badFirst bool
		holes    []Hole
		overlap  string // a substring of the error, or "" for none
	}{
		{name: "nothing at all"},
		{name: "contiguous", rs: []seqRange{r(0, 1, 4), r(1, 5, 8), r(2, 9, 9)}},
		{name: "a bad first segment is a hole from 1 up to the first good start", rs: []seqRange{r(1, 5, 8), r(2, 9, 12)}, badFirst: true, holes: []Hole{{1, 4}}},
		{name: "a bad first segment before a log that starts at 1 leaves no empty hole", rs: []seqRange{r(1, 1, 4)}, badFirst: true},
		{name: "only bad segments leave the tail step to the caller", badFirst: true},
		{name: "a bad middle segment is the gap between its neighbours", rs: []seqRange{r(0, 1, 4), r(2, 9, 12)}, holes: []Hole{{5, 8}}},
		{name: "a bad first and a bad middle segment", rs: []seqRange{r(1, 5, 8), r(3, 13, 14)}, badFirst: true, holes: []Hole{{1, 4}, {9, 12}}},
		{name: "an empty active segment adds no range", rs: []seqRange{r(0, 1, 4), r(1, 5, 8)}},
		{name: "an active segment after a gap", rs: []seqRange{r(0, 1, 4), r(1, 5, 8), r(3, 12, 12)}, holes: []Hole{{9, 11}}},
		{name: "an active segment after a bad first one", rs: []seqRange{r(1, 3, 3)}, badFirst: true, holes: []Hole{{1, 2}}},
		{name: "a sealed range cut short by a scan leaves the rest as a hole", rs: []seqRange{r(0, 1, 6), r(1, 9, 9)}, holes: []Hole{{7, 8}}},
		{name: "a stale marker below a sealed segment's end is no overlap while the next segment still starts after it", rs: []seqRange{r(0, 1, 8), r(1, 9, 12)}},
		{name: "an active segment that starts inside a sealed one is an overlap", rs: []seqRange{r(0, 1, 8), r(1, 5, 5)}, overlap: "segment 1 starts at 5, inside the one before (segment 0, ends at 8)"},
		{name: "a sealed segment that starts at the previous end is an overlap", rs: []seqRange{r(0, 1, 4), r(1, 4, 8)}, overlap: "segment 1 starts at 4"},
		{name: "an overlap after a hole is still an overlap", rs: []seqRange{r(0, 1, 4), r(2, 8, 10), r(3, 9, 12)}, overlap: "segment 3 starts at 9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			holes, err := contiguity(tc.rs, tc.badFirst)
			if tc.overlap != "" {
				if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), tc.overlap) || holes != nil {
					t.Fatalf("got %+v, %v; want ErrCorrupt containing %q", holes, err, tc.overlap)
				}
				return
			}
			if err != nil || !slices.Equal(holes, tc.holes) {
				t.Fatalf("got %+v, %v; want %+v", holes, err, tc.holes)
			}
		})
	}
}

// A sealed segment whose bytes are another's puts the ranges out of order: the writer's Open
// refuses it and a Reader's view reports the same ErrCorrupt, through the one helper.
func TestOverlappingSealedSegmentsAreCorruptForTheWriterAndTheReader(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
	for i := 0; i < 4; i++ {
		appendN(t, l, 4)
		mustFlush(t, l)
	}
	_ = l.Close()
	writeFile(t, s, "/log/"+segmentName(1), fileBytes(t, s, "/log/"+segmentName(3)))
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.Read(1, 100); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("reader: %v", err)
	}
	if _, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("writer: %v", err)
	}
}

// A damaged oldest segment is a hole from 1, not expiry: a scan from the beginning, a new
// consumer and the command line's default start meet the gap instead of starting silently
// at the first good segment (decision 38). Expiry (a deleted prefix) still starts at First.
func TestAScanFromTheBeginningReportsADamagedOldestSegmentAsAGap(t *testing.T) {
	build := func(t *testing.T) (*fsx.Sim, uint64) {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
		for i := 0; i < 4; i++ {
			appendN(t, l, 4)
			mustFlush(t, l)
		}
		_ = l.Close()
		segs := segmentPaths(t, s)
		old := fileBytes(t, s, segs[0])
		writeFile(t, s, segs[0], bytes.Repeat([]byte{0xA5}, len(old)))
		r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		st, err := r.Stat()
		if err != nil || len(st.Holes) == 0 || st.Holes[0].From != 1 || st.First < 2 {
			t.Fatalf("setup: %+v, %v", st, err)
		}
		return s, st.First
	}
	scanAll := func(src interface{ ScanAll(func(Record) bool) error }) ([]uint64, error) {
		var got []uint64
		err := src.ScanAll(func(rec Record) bool { got = append(got, rec.Seq); return true })
		return got, err
	}
	t.Run("reader", func(t *testing.T) {
		s, first := build(t)
		r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		if got, err := scanAll(r); len(got) != 0 || !isGap(err, 1, first-1) {
			t.Fatalf("ScanAll: %v, %v; want a gap 1 to %d", got, err, first-1)
		}
		c, err := r.Consumer("new", ConsumerOptions{})
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if c.Offset() != 1 {
			t.Fatalf("a new consumer starts at %d, want 1", c.Offset())
		}
		if _, err := c.Read(10); !isGap(err, 1, first-1) {
			t.Fatalf("consumer Read: %v", err)
		}
		if err := c.Commit(first); err != nil {
			t.Fatalf("Commit past the hole: %v", err)
		}
		if recs, err := c.Read(100); err != nil || recs[0].Seq != first {
			t.Fatalf("after Commit: %v, %v", seqsOf(recs), err)
		}
	})
	t.Run("writer", func(t *testing.T) {
		s, first := build(t)
		l := openSim(t, s, nil)
		defer l.Close()
		if got, err := scanAll(l); len(got) != 0 || !isGap(err, 1, first-1) {
			t.Fatalf("ScanAll: %v, %v; want a gap 1 to %d", got, err, first-1)
		}
	})
	t.Run("expiry is not a gap", func(t *testing.T) {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
		for i := 0; i < 4; i++ {
			appendN(t, l, 4)
			mustFlush(t, l)
		}
		_ = l.Close()
		if err := s.Remove(segmentPaths(t, s)[0]); err != nil {
			t.Fatal(err)
		}
		r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		if got, err := scanAll(r); err != nil || len(got) == 0 || got[0] < 2 {
			t.Fatalf("ScanAll: %v, %v", got, err)
		}
	})
}

// Retention deleting the clean prefix in front of a damaged segment leaves the damaged segment
// as the oldest file on disk: the writer must still call its range a hole from 1, as a fresh
// Open and a Reader do, so a scan from the beginning and a new consumer meet the gap and the
// Log's Stat agrees with them (decision 38). The pass used to drop the hole because its end is
// below the new first retained sequence, though the damaged file is still there.
func TestARetentionPassThatLeavesADamagedSegmentOldestKeepsItsHoleFromOne(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, nil)
	fillSegments(t, l, clk, 4) // segments 0..3 sealed, records 1..16
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	seg := "/log/" + segmentName(1) // records 5 to 8
	data := fileBytes(t, s, seg)
	data[len(data)-1] ^= 0xff
	writeFile(t, s, seg, data)

	// 8 records may stay: segment 0 goes, and the pass stops before the damaged segment 1.
	l = openRetention(t, s, clk, func(c *Config) { c.MaxRecords = 8 })
	defer l.Close()
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	want := []Hole{{From: 1, To: 8}}
	st, err := l.Stat()
	if err != nil || st.First != 9 || !slices.Equal(st.Holes, want) {
		t.Fatalf("Stat after the pass: first %d holes %+v, %v; want first 9, holes %+v", st.First, st.Holes, err, want)
	}
	var got []uint64
	err = l.ScanAll(func(rec Record) bool { got = append(got, rec.Seq); return true })
	if len(got) != 0 || !isGap(err, 1, 8) {
		t.Fatalf("ScanAll: %v, %v; want a gap 1 to 8", got, err)
	}
	c, err := l.Consumer("new", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if c.Offset() != 1 {
		t.Fatalf("a new consumer starts at %d, want 1", c.Offset())
	}
	_ = c.Close()
	// A later pass that deletes the damaged segment too clears the hole: expiry again.
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	l.cfg.MaxRecords = 4
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if st, err := l.Stat(); err != nil || len(st.Holes) != 0 || st.First != 13 {
		t.Fatalf("after the damaged segment is deleted: first %d holes %+v, %v", st.First, st.Holes, err)
	}
	if err := l.ScanAll(func(Record) bool { return true }); err != nil {
		t.Fatalf("ScanAll after the damaged segment is deleted: %v", err)
	}
}
