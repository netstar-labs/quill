package quill

import (
	"errors"
	"io/fs"
	"sync"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// unwarmedVanishingLog is vanishingLog without the warm read, so no reader has scanned the
// middle segment (records 5-8) before a test damages it. It returns the simulator and the path
// of the middle segment.
func unwarmedVanishingLog(t *testing.T) (*fsx.Sim, string) {
	t.Helper()
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	t.Cleanup(func() { _ = l.Close() })
	fillSegments(t, l, clk, 3)
	appendN(t, l, 1)
	mustFlush(t, l)
	ids := segmentFiles(t, s, "/log")
	if len(ids) < 4 {
		t.Fatalf("segments %v", ids)
	}
	return s, "/log/" + segmentName(ids[1])
}

// A sealed segment that ends before its derived range does is served up to its real last
// record, and the gap is reported past it (decisions 38 and 101): the last record the scan
// found is not lost.
func TestAReaderServesTheLastRecordOfASealedSegmentThatEndsEarly(t *testing.T) {
	s, victim := unwarmedVanishingLog(t)
	data := fileBytes(t, s, victim)
	ends := frameEnds(t, data)
	if len(ends) != 4 {
		t.Fatalf("the middle segment has %d frames, want 4", len(ends))
	}
	writeFile(t, s, victim, data[:ends[0]]) // record 5 kept, 6 to 8 gone, cut at a frame boundary
	r := openReaderSim(t, s)
	t.Cleanup(func() { _ = r.Close() })

	recs, err := r.Read(5, 1)
	if err != nil || len(recs) != 1 || recs[0].Seq != 5 {
		t.Fatalf("Read(5, 1): %v, %v; want record 5", seqsOf(recs), err)
	}
	if _, err := r.Read(6, 1); !isGap(err, 6, 8) {
		t.Fatalf("Read(6, 1): %v; want the gap 6..8", err)
	}
}

// A sealed segment that vanishes between the listing and its full scan is a gap over its
// range, not the path error: a consumer resumes from To + 1 on a gap and would otherwise
// retry or stop (decisions 38, 94 and 101).
func TestASegmentDeletedBetweenListingAndScanIsAGap(t *testing.T) {
	s, victim := unwarmedVanishingLog(t)
	var mu sync.Mutex
	opens := 0
	s.Inject = func(op fsx.Op, name string) error {
		if op != fsx.OpOpen || name != victim {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		opens++
		if opens == 2 { // the first open reads the first frame when listing, the second scans
			return fs.ErrNotExist
		}
		return nil
	}
	r := openReaderSim(t, s)
	t.Cleanup(func() { _ = r.Close() })

	recs, err := r.Read(6, 1)
	if len(recs) != 0 || !isGap(err, 6, 8) {
		t.Fatalf("Read(6, 1): %v, %v; want the gap 6..8", seqsOf(recs), err)
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a path error escaped: %v", err)
	}
}
