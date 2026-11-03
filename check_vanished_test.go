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
