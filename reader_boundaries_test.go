package quill

import (
	"errors"
	"math"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A sealed segment is immutable (decision 63): a Reader that cached its size refuses it when the
// size is anything else, longer as well as shorter. Only the shrinking half was reachable by a
// test before: a comparison that let a grown segment through changed no result.
func TestReaderRefusesASealedSegmentThatChangedSizeEitherWay(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(data []byte) []byte
	}{
		{"grew", func(d []byte) []byte { return append(d, 0, 0, 0) }},
		{"shrank", func(d []byte) []byte { return d[:len(d)-3] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, r, victim, _ := vanishingLog(t) // vanishingLog warms the cache on the middle segment
			writeFile(t, s, victim, tc.change(fileBytes(t, s, victim)))
			recs, err := r.Read(5, 4)
			if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "changed size") {
				t.Fatalf("Read(5, 4) over a segment that %s: %v, %v; want ErrCorrupt naming the size change", tc.name, seqsOf(recs), err)
			}
		})
	}
}

// readTail retries a transient failure before it gives up (it re-reads the marker a few times,
// a millisecond apart), and gives the last error up after that.
func TestReaderRetriesATransientMarkerReadFailure(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()

	const tailPath = "/log/quill.tail"
	failures := 0
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpRead && name == tailPath && failures < 2 {
			failures++
			return syscall.EIO
		}
		return nil
	}
	head, err := r.Head()
	if err != nil || head != 3 {
		t.Fatalf("Head after two failed marker reads: %d, %v; want 3, nil", head, err)
	}
	if failures != 2 {
		t.Fatalf("%d marker reads failed, want 2: the injection did not run as planned", failures)
	}

	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpRead && name == tailPath {
			return syscall.EIO
		}
		return nil
	}
	if _, err := r.Head(); !errors.Is(err, syscall.EIO) {
		t.Fatalf("Head with a marker that never reads: %v, want the I/O error", err)
	}
}

// The ceiling is MaxUint64-1 (the last sequence is never assigned): a marker naming the highest
// assignable sequence is a good log, and only MaxUint64 is refused.
func TestReaderAcceptsAHeadAtTheSequenceCeiling(t *testing.T) {
	const top = uint64(math.MaxUint64)
	s := simLogWithHead(t, top-1, 2)
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatalf("OpenReader at head %d: %v", top-1, err)
	}
	defer r.Close()
	if head, err := r.Head(); err != nil || head != top-1 {
		t.Fatalf("Head: %d, %v; want %d, nil", head, err, top-1)
	}
	recs, err := r.Read(top-1, 5)
	if err != nil || len(recs) != 1 || recs[0].Seq != top-1 {
		t.Fatalf("Read(%d): %v, %v; want that one record", top-1, seqsOf(recs), err)
	}
}

// A Reader serves no sequence past the one the marker names, even when the active segment holds
// whole, valid frames beyond it within the marker's byte offset (a marker damaged by hand).
func TestReaderServesNoSequencePastTheMarkersSequence(t *testing.T) {
	s := simLogWithHead(t, 5, 5) // frames 1..5
	img := fileBytes(t, s, "/log/"+segmentName(0))
	writeFile(t, s, "/log/quill.tail", encodeTail(tail{Seq: 3, Seg: 0, Off: uint64(len(img)), MaxSeg: 0}))
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer r.Close()
	recs, err := r.Read(1, 10)
	if got := seqsOf(recs); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("Read(1, 10): %v, %v; want sequences 1 to 3 only", got, err)
	}
	var seen []uint64
	if err := r.ScanAll(func(rec Record) bool { seen = append(seen, rec.Seq); return true }); err != nil || len(seen) != 3 {
		t.Fatalf("ScanAll: %v, %v; want sequences 1 to 3 only", seen, err)
	}
}

// OffsetForTime over a sealed segment whose bytes went bad after its index was built is a gap
// over the segment, as a damaged segment is everywhere else, not the bare frame error.
func TestOffsetForTimeOverABadFrameWalkIsAGapOverTheSegment(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 4)
	base := l.sealed[0].at
	r := openReaderSim(t, s)
	defer r.Close()
	if recs, err := r.Read(9, 4); err != nil || len(recs) != 4 { // builds segment 2's index
		t.Fatalf("warm Read(9, 4): %v, %v", seqsOf(recs), err)
	}
	flipLastByteOfFrame(t, s, 2, 1)
	target := time.Unix(base+130, 0) // after segment 2's stamp, before segment 3's
	off, err := r.OffsetForTime(target)
	if !isGap(err, 9, 12) {
		t.Fatalf("Reader.OffsetForTime: %d, %v; want the gap 9 to 12", off, err)
	}
}
