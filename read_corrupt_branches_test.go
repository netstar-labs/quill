package quill

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// scanSource separates damage that is loss from a marker that disagrees with the bytes. These
// tests pin the two branches that tell them apart, each at the boundary that distinguishes it:
// a marker one sequence ahead of the frames its offset covers (the segment ends at head - 1,
// not at head) and damage inside the durable part of the active segment (corruption, never a
// gap a consumer could resume past).

// closedSimLogWith returns a Sim holding a closed log of n records in one segment.
func closedSimLogWith(t *testing.T, n int) *fsx.Sim {
	t.Helper()
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20; c.SegmentMaxAge = -1 })
	appendN(t, l, n)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestReaderRefusesAMarkerOneSequenceAheadOfItsFrames(t *testing.T) {
	s := closedSimLogWith(t, 3)
	tl := readTail(t, s, "/log")
	if tl.Seq != 3 || tl.Off == 0 {
		t.Fatalf("setup: marker %+v", tl)
	}
	tl.Seq++ // 4: the offset still covers only the frames of 1 to 3
	writeFile(t, s, filepath.Join("/log", tailName), encodeTail(tl))

	r := openReaderSim(t, s)
	defer r.Close()
	type result struct {
		recs []Record
		err  error
	}
	done := make(chan result, 1)
	go func() {
		recs, err := r.Read(1, 100)
		done <- result{recs, err}
	}()
	select {
	case res := <-done:
		if !errors.Is(res.err, ErrCorrupt) {
			t.Fatalf("Read(1, 100): %d records, %v; want ErrCorrupt, the segment ends at 3 and the marker says 4", len(res.recs), res.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Read(1, 100) did not return: a marker one sequence ahead of its frames must be ErrCorrupt, not a wait for a record that is not there")
	}
}

func TestDamageInTheActiveSegmentIsCorruptionNotAGap(t *testing.T) {
	s := closedSimLogWith(t, 4)
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 1 {
		t.Fatalf("setup: segments %v", ids)
	}
	flipLastByteOfFrame(t, s, ids[0], 2) // the frame of 3, inside the durable part

	r := openReaderSim(t, s)
	defer r.Close()
	recs, err := r.Read(3, 100)
	var gap *GapError
	if !errors.Is(err, ErrCorrupt) || errors.As(err, &gap) || len(recs) != 0 {
		t.Fatalf("Read(3, 100): %d records, %v; want ErrCorrupt and not a gap (a resume past a gap would skip 3 and 4)", len(recs), err)
	}
}
