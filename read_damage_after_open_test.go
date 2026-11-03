package quill

import (
	"errors"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Damage that appears after Open (bit rot, an outside write) in a segment the log already
// verified, without changing its size, is met by a read when it scans the frame (decisions
// 38, 79 and 94). In a sealed segment it is a GapError over the rest of that segment, from the
// record that failed to the segment's last, so a resume at To + 1 never re-enters the damaged
// file; in the active segment it is ErrCorrupt, which a consumer must not skip (guarantee 14).

// flipLastByteOfFrame flips the last byte (a CRC byte) of the k-th frame (from 0) of a segment
// file and keeps its size.
func flipLastByteOfFrame(t *testing.T, s *fsx.Sim, id uint64, k int) {
	t.Helper()
	path := "/log/" + segmentName(id)
	data := fileBytes(t, s, path)
	pos := 0
	for i := 0; i <= k; i++ {
		_, _, total, ok := frameHeader(data[pos:])
		if !ok || pos+total > len(data) {
			t.Fatalf("frame %d of %s is not whole", i, path)
		}
		pos += total
	}
	data[pos-1] ^= 0xff
	writeFile(t, s, path, data)
}

func TestASealedSegmentDamagedAfterOpenIsAGapOverTheRestOfItAndAResumeAfterItReadsOn(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 5) // sealed segments 0..4 hold 1..4, 5..8, 9..12, 13..16, 17..20
	if ids := sealedIDs(l); len(ids) != 5 || l.sealed[2].first != 9 || l.sealed[2].last != 12 {
		t.Fatalf("setup: sealed %v", ids)
	}
	// A Reader opened before the damage meets it the same way.
	r := openReaderSim(t, s)
	defer r.Close()
	flipLastByteOfFrame(t, s, 2, 1) // the frame of 10, in the middle of the segment

	recs, err := l.Read(1, 100)
	if err != nil || len(recs) != 9 {
		t.Fatalf("Read(1, 100): %d records, %v; want 1..9 and the gap left for the next call", len(recs), err)
	}
	checkContiguous(t, recs, 1)
	recs, err = readAll(l)
	if !isGap(err, 10, 12) || len(recs) != 9 {
		t.Fatalf("ScanAll: %d records, %v; want 9 records, then the gap 10 to 12", len(recs), err)
	}
	if recs, err := l.Read(10, 10); !isGap(err, 10, 12) || len(recs) != 0 {
		t.Fatalf("Read(10, 10): %v, %v; want the gap 10 to 12 and no records", recs, err)
	}
	if _, err := l.Read(11, 10); !isGap(err, 11, 12) {
		t.Fatalf("Read(11, 10): %v; want the gap 11 to 12, the rest of the segment", err)
	}
	recs, err = l.Read(13, 100) // Commit(To + 1)
	if err != nil || len(recs) != 8 {
		t.Fatalf("Read(13, 100): %d records, %v; want 13..20", len(recs), err)
	}
	checkContiguous(t, recs, 13)

	if _, err := r.Read(10, 10); !isGap(err, 10, 12) {
		t.Fatalf("Reader.Read(10, 10): %v; want the gap 10 to 12", err)
	}
}

func TestTheActiveSegmentDamagedAfterOpenIsCorruptNeverAGap(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = -1; c.SegmentBytes = 1 << 20 })
	defer l.Close()
	appendN(t, l, 4) // all in the active segment
	mustFlush(t, l)
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 1 || len(sealedIDs(l)) != 0 {
		t.Fatalf("setup: segments %v, sealed %v", ids, sealedIDs(l))
	}
	flipLastByteOfFrame(t, s, ids[0], 1) // the frame of 2

	recs, err := l.Read(1, 100)
	if err != nil || len(recs) != 1 {
		t.Fatalf("Read(1, 100): %d records, %v; want record 1 and the error left for the next call", len(recs), err)
	}
	recs, err = readAll(l)
	var gap *GapError
	if !errors.Is(err, ErrCorrupt) || errors.As(err, &gap) || len(recs) != 1 {
		t.Fatalf("ScanAll: %d records, %v; want 1 record, then ErrCorrupt and not a gap", len(recs), err)
	}
	if _, err := l.Read(2, 10); !errors.Is(err, ErrCorrupt) || errors.As(err, &gap) {
		t.Fatalf("Read(2, 10): %v; want ErrCorrupt and not a gap", err)
	}
}

func TestOffsetForTimeOnASealedSegmentDamagedAfterOpenIsAGapOverTheSegment(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 4) // segment k holds 4k+1..4k+4, stamped base + k minutes
	base := l.sealed[0].at
	if l.sealed[2].at != base+120 {
		t.Fatalf("setup: segment 2 at %d, base %d", l.sealed[2].at, base)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	flipLastByteOfFrame(t, s, 2, 1)
	// Between the stamp of segment 2 and of segment 3: the search lands in segment 2 and finds
	// nothing at or after the target in its frames before the damage.
	target := time.Unix(base+130, 0)
	if off, err := l.OffsetForTime(target); !isGap(err, 9, 12) {
		t.Fatalf("OffsetForTime: %d, %v; want the gap 9 to 12", off, err)
	}
	if off, err := r.OffsetForTime(target); !isGap(err, 9, 12) {
		t.Fatalf("Reader.OffsetForTime: %d, %v; want the gap 9 to 12", off, err)
	}
}
