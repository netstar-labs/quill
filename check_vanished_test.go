package quill

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// segmentPaths lists the segment files of the simulated log, oldest first.
func segmentPaths(t *testing.T, s *fsx.Sim) []string {
	t.Helper()
	names, err := s.ReadDir("/log")
	if err != nil {
		t.Fatal(err)
	}
	var segs []string
	for _, n := range names {
		if _, ok := parseSegmentName(n); ok {
			segs = append(segs, "/log/"+n)
		}
	}
	return segs
}

// A sealed segment that retention deletes after Check listed the directory is not damage
// and leaves no hole: retention deletes oldest first, so the segments before it are gone
// too (decision 101 for Read; Check takes no lock and runs beside a live writer).
func TestCheckTreatsASealedSegmentDeletedAfterTheListingAsRetentionNotDamage(t *testing.T) {
	t.Run("oldest", func(t *testing.T) {
		s, r, _, _ := vanishingLog(t)
		segs := segmentPaths(t, s)
		removeOnOpen(s, segs[0], 0)
		res, err := r.Check()
		if err != nil || !res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
	})
	t.Run("middle, with its predecessors removed by the same sweep", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		segs := segmentPaths(t, s)
		var once sync.Once
		s.Inject = func(op fsx.Op, name string) error {
			if op == fsx.OpOpen && name == victim {
				once.Do(func() {
					_ = s.Remove(segs[0])
					_ = s.Remove(victim)
				})
			}
			return nil
		}
		res, err := r.Check()
		if err != nil || !res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
	})
}

// A segment already missing from the listing is still a hole, and so is a vanished
// active segment: only a sealed segment that disappears mid-scan is excused.
func TestCheckStillReportsAHoleAndAMissingActiveSegment(t *testing.T) {
	t.Run("hole in the listing", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		if err := s.Remove(victim); err != nil {
			t.Fatal(err)
		}
		res, err := r.Check()
		if err != nil || res.Healthy || !strings.Contains(fmt.Sprint(res.Problems), "sequences 5 to 8 are missing") {
			t.Fatalf("Check: %+v, %v", res, err)
		}
	})
	t.Run("active segment vanishes", func(t *testing.T) {
		s, r, _, _ := vanishingLog(t)
		segs := segmentPaths(t, s)
		removeOnOpen(s, segs[len(segs)-1], 0)
		res, err := r.Check()
		if err != nil || res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
	})
}

// The segment the marker names is the active one, and a writer that rolls it and a cap that
// deletes it can both happen between Check's read of the marker and its read of the file.
// Retention publishes the marker before every unlink (decision 65), so a missing segment that
// the marker no longer names is retention, not damage. One the marker still names is damage
// (TestCheckStillReportsAHoleAndAMissingActiveSegment).
func TestCheckTreatsTheMarkersSegmentRolledAndDeletedAfterTheReadAsRetentionNotDamage(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1; c.MaxBytes = 120 }) // the smallest cap: two segments
	t.Cleanup(func() { _ = l.Close() })
	appendN(t, l, 6)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	t.Cleanup(func() { _ = r.Close() })
	segs := segmentPaths(t, s)
	active := segs[len(segs)-1]
	var once sync.Once
	var werr error
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpOpen && name == active {
			once.Do(func() { // the writer moves on while Check is between the marker and the file
				for range 12 {
					if _, werr = l.Append(payloadFor(l.nextSeqForTest())); werr != nil {
						return
					}
				}
				werr = l.Flush()
			})
		}
		return nil
	}
	res, err := r.Check()
	if werr != nil {
		t.Fatalf("the writer: %v", werr)
	}
	if _, serr := s.Lstat(active); serr == nil {
		t.Fatal("setup: the segment the marker named is still there")
	}
	if err != nil || !res.Healthy {
		t.Fatalf("Check: %+v, %v", res, err)
	}
}
