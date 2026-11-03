package quill

import (
	"errors"
	"slices"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Tests for the hole and head bookkeeping in load(): which damaged sealed segments become holes,
// where the head is when the newest sealed segment is damaged, and when a marker above the last
// valid frame is refused (decisions 38, 39 and 98).

// rolledLog writes per records into each of n segments, rolling after each one, so the log
// ends with n sealed segments and an empty active segment, as after a Roll or an age rotation.
// The log is closed. Segment i holds the sequences i*per+1 to (i+1)*per.
func rolledLog(t *testing.T, s *fsx.Sim, n, per int) {
	t.Helper()
	l := openSim(t, s, nil)
	for i := 0; i < n; i++ {
		appendN(t, l, per)
		if err := l.Roll(); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != n+1 {
		t.Fatalf("segments %v, want %d", ids, n+1)
	}
}

// damageSegment flips the last byte of a segment file: a frame that fails its checksum.
func damageSegment(t *testing.T, s *fsx.Sim, id uint64) {
	t.Helper()
	path := "/log/" + segmentName(id)
	b := fileBytes(t, s, path)
	b[len(b)-1] ^= 0xff
	writeFile(t, s, path, b)
}

func setMarkerSeq(t *testing.T, s *fsx.Sim, seq uint64) {
	t.Helper()
	tl := readTail(t, s, "/log")
	tl.Seq = seq
	writeFile(t, s, "/log/"+tailName, encodeTail(tl))
}

// The newest sealed segment damaged and the active one empty is the routine state after a
// Roll: the marker still says where the log ends, so Open succeeds with the damaged range as a
// hole that ends at the head, and the next Append continues from the marker.
func TestADamagedNewestSealedSegmentBeforeAnEmptyActiveOneOpensWithATrailingHole(t *testing.T) {
	s := fsx.NewSim()
	rolledLog(t, s, 2, 4)
	damageSegment(t, s, 1) // 5..8
	l, err := Open(simConfig(s, "/log"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer l.Close()
	if !errors.Is(l.Err(), ErrCorrupt) {
		t.Fatalf("Err() %v", l.Err())
	}
	if l.Head() != 8 {
		t.Fatalf("head %d, want 8", l.Head())
	}
	if want := []Hole{{From: 5, To: 8}}; !slices.Equal(l.holes, want) {
		t.Fatalf("holes %+v, want %+v", l.holes, want)
	}
	// A hole that ends at the head is not counted as records.
	st, err := l.Stat()
	if err != nil || st.Head != 8 || st.Records != 4 {
		t.Fatalf("Stat: head %d records %d, %v; want head 8, records 4", st.Head, st.Records, err)
	}
	if seq, err := l.Append(payloadFor(9)); err != nil || seq != 9 {
		t.Fatalf("Append: %d, %v; want 9", seq, err)
	}
}

// Every sealed segment damaged and an empty active one: one hole from 1 to the marker.
func TestEverySealedSegmentDamagedAndAnEmptyActiveOneIsOneHoleFromOne(t *testing.T) {
	s := fsx.NewSim()
	rolledLog(t, s, 2, 4)
	damageSegment(t, s, 0)
	damageSegment(t, s, 1)
	l, err := Open(simConfig(s, "/log"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer l.Close()
	if want := []Hole{{From: 1, To: 8}}; !slices.Equal(l.holes, want) || l.Head() != 8 {
		t.Fatalf("holes %+v head %d, want %+v and 8", l.holes, l.Head(), want)
	}
	if st, err := l.Stat(); err != nil || st.Records != 0 {
		t.Fatalf("Stat: records %d, %v; want 0", st.Records, err)
	}
}

// Several damaged segments each keep their hole, and the damaged oldest one, though it is not
// the only segment, is still a hole from 1 that a new consumer meets as a gap (decision 38).
func TestSeveralDamagedSegmentsKeepTheirHolesAndTheLeadingOne(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	for i := 0; i < 3; i++ {
		appendN(t, l, 4)
		if err := l.Roll(); err != nil {
			t.Fatal(err)
		}
	}
	appendN(t, l, 4)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	damageSegment(t, s, 0) // 1..4
	damageSegment(t, s, 2) // 9..12
	l, err := Open(simConfig(s, "/log"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer l.Close()
	if want := []Hole{{From: 1, To: 4}, {From: 9, To: 12}}; !slices.Equal(l.holes, want) {
		t.Fatalf("holes %+v, want %+v", l.holes, want)
	}
	c, err := l.Consumer("c", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Offset() != 1 {
		t.Fatalf("a new consumer starts at %d, want 1 (the gap)", c.Offset())
	}
}

// A marker above the last valid frame is corruption (decision 39), by one as by more, whether
// the newest segment holds frames or is empty, and a damaged segment earlier in the log does not
// turn the difference into a trailing hole: only a damaged segment after the last good one can.
func TestAMarkerAboveTheLastValidFrameIsRefusedAfterAnEarlierDamagedSegment(t *testing.T) {
	t.Run("active segment with frames", func(t *testing.T) {
		build := func() *fsx.Sim {
			s := fsx.NewSim()
			l := openSim(t, s, nil)
			for i := 0; i < 2; i++ {
				appendN(t, l, 4)
				if err := l.Roll(); err != nil {
					t.Fatal(err)
				}
			}
			appendN(t, l, 4)
			mustFlush(t, l)
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			damageSegment(t, s, 1) // 5..8
			return s
		}
		l, err := Open(simConfig(build(), "/log"))
		if err != nil {
			t.Fatalf("unchanged marker: %v", err)
		}
		if l.Head() != 12 || !slices.Equal(l.holes, []Hole{{From: 5, To: 8}}) {
			t.Fatalf("head %d holes %+v", l.Head(), l.holes)
		}
		_ = l.Close()
		for _, seq := range []uint64{13, 14} {
			s := build()
			setMarkerSeq(t, s, seq)
			if l, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
				if err == nil {
					_ = l.Close()
				}
				t.Fatalf("marker %d: open %v, want ErrCorrupt", seq, err)
			}
		}
	})
	t.Run("empty active segment", func(t *testing.T) {
		build := func() *fsx.Sim {
			s := fsx.NewSim()
			rolledLog(t, s, 2, 4)
			damageSegment(t, s, 0) // 1..4; segment 1 (5..8) is good
			return s
		}
		l, err := Open(simConfig(build(), "/log"))
		if err != nil {
			t.Fatalf("unchanged marker: %v", err)
		}
		if l.Head() != 8 || !slices.Equal(l.holes, []Hole{{From: 1, To: 4}}) {
			t.Fatalf("head %d holes %+v", l.Head(), l.holes)
		}
		_ = l.Close()
		for _, seq := range []uint64{9, 10} {
			s := build()
			setMarkerSeq(t, s, seq)
			if l, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
				if err == nil {
					_ = l.Close()
				}
				t.Fatalf("marker %d: open %v, want ErrCorrupt", seq, err)
			}
		}
	})
}
