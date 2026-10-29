package quill

import (
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
