package quill

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// vanishingLog builds three sealed segments of four records (1-4, 5-8, 9-12) and an active
// one, and returns the simulator, a reader on it, the path of the middle segment and the
// time of its first record.
func vanishingLog(t *testing.T) (*fsx.Sim, *Reader, string, int64) {
	t.Helper()
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	t.Cleanup(func() { _ = l.Close() })
	fillSegments(t, l, clk, 3)
	appendN(t, l, 1)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	t.Cleanup(func() { _ = r.Close() })
	recs, err := r.Read(5, 1)
	if err != nil || len(recs) != 1 {
		t.Fatalf("warm read: %v %v", recs, err)
	}
	var segs []string
	for _, id := range segmentFiles(t, s, "/log") {
		segs = append(segs, "/log/"+segmentName(id))
	}
	if len(segs) < 4 {
		t.Fatalf("segments %v", segs)
	}
	return s, r, segs[1], recs[0].At
}

// removeOnOpen deletes path from the simulator as the skip+1'th open of it begins, which is
// after the reader listed the directory and before it has the file.
func removeOnOpen(s *fsx.Sim, path string, skip int) {
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpOpen && name == path {
			if skip > 0 {
				skip--
				return nil
			}
			s.Inject = nil
			_ = s.Remove(path)
		}
		return nil
	}
}

func TestASealedSegmentDeletedBetweenListingAndOpeningIsAGapOverItsRange(t *testing.T) {
	t.Run("Read", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		recs, err := r.Read(6, 10) // starts mid-segment: the gap starts at the request
		if len(recs) != 0 || !isGap(err, 6, 8) {
			t.Fatalf("Read: %v, %v", seqsOf(recs), err)
		}
		if errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("a path error escaped: %v", err)
		}
		// The entry was forgotten: the next call lists again and the hole is the same gap.
		if _, err := r.Read(5, 10); !isGap(err, 5, 8) {
			t.Fatalf("Read after: %v", err)
		}
	})
	t.Run("Read after records", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		recs, err := r.Read(3, 10) // segment 0 delivers, the gap is for the next call
		if err != nil || len(recs) != 2 {
			t.Fatalf("Read: %v, %v", seqsOf(recs), err)
		}
	})
	t.Run("Follow", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var got error
		for _, err := range r.Follow(ctx, 5) {
			got = err
			break
		}
		if !isGap(got, 5, 8) {
			t.Fatalf("Follow: %v", got)
		}
	})
	t.Run("OffsetForTime", func(t *testing.T) {
		s, r, victim, at := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		if _, err := r.OffsetForTime(time.Unix(at+30, 0)); !isGap(err, 5, 8) {
			t.Fatalf("OffsetForTime: %v", err)
		}
	})
	t.Run("Archive", func(t *testing.T) {
		s, _, victim, _ := vanishingLog(t)
		if err := s.Mkdir("/arch", 0o700); err != nil {
			t.Fatal(err)
		}
		removeOnOpen(s, victim, 1) // the first open reads the segment's first frame
		if err := archive(s, "/log", "/arch"); !isGap(err, 5, 8) {
			t.Fatalf("archive: %v", err)
		}
	})
}

func TestTheMarkersOwnSegmentMissingStaysAnError(t *testing.T) {
	s, r, _, _ := vanishingLog(t)
	ids := segmentFiles(t, s, "/log")
	newest := "/log/" + segmentName(ids[len(ids)-1])
	if err := s.Remove(newest); err != nil {
		t.Fatal(err)
	}
	_, err := r.Read(5, 10)
	var gap *GapError
	if err == nil || errors.As(err, &gap) {
		t.Fatalf("the marker's segment missing: %v", err)
	}
}
