package quill

import (
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A sealed segment that was removed by hand under a live writer is deleted from the log's
// point of view all the same: Config.Logger's documentation says it gives two records, the
// Warn "segment was already gone" and then the deletion's own Info, not one.
func TestASegmentAlreadyGoneGivesItsWarnAndThenTheDeletionRecord(t *testing.T) {
	var cap logCapture
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Logger = cap.logger() })
	defer l.Close()
	fillSegments(t, l, clk, 3)
	gone := "/log/" + segmentName(0)
	if err := s.Remove(gone); err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * time.Hour) // every sealed segment is past Retention
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	cap.mu.Lock()
	defer cap.mu.Unlock()
	var warn, info int
	for _, m := range cap.msgs {
		if !strings.Contains(m, gone) {
			continue
		}
		switch {
		case strings.Contains(m, "segment was already gone") && strings.Contains(m, "level=WARN"):
			warn++
		case strings.Contains(m, "deleted segment") && strings.Contains(m, "level=INFO"):
			info++
		}
	}
	if warn != 1 || info != 1 {
		t.Fatalf("records naming %s: %d already-gone Warn, %d deletion Info; log:\n%s", gone, warn, info, strings.Join(cap.msgs, ""))
	}
}
