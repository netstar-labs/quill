package quill

import (
	"errors"
	"io/fs"
	"sync"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

func viewIDs(t *testing.T, l *Log) (*segView, func(), []uint64) {
	t.Helper()
	v, release, err := l.view()
	if err != nil {
		t.Fatal(err)
	}
	return v, release, idsOf(v)
}

func equalIDs(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The sealed refs are built once and shared by every view until l.sealed changes (decision
// 103). A view held across a rotation and a deletion keeps the segments it was made with and
// no later view is stale: the cache is invalidated at the two places that write l.sealed.
func TestSealedSegmentRefsAreSharedByViewsAndRebuiltAfterARotationAndADeletion(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3) // segments 0..2 sealed, 3 active and empty

	v1, rel1, ids1 := viewIDs(t, l)
	defer rel1()
	if !equalIDs(ids1, []uint64{0, 1, 2}) {
		t.Fatalf("first view holds segments %v", ids1)
	}
	v2, rel2, _ := viewIDs(t, l)
	defer rel2()
	for i := range v1.segs {
		if v1.segs[i] != v2.segs[i] {
			t.Fatalf("segment %d: two views with no change between them built separate refs", i)
		}
	}
	if &v1.segs[0] == &v2.segs[0] {
		t.Fatal("views share one slice: an append to it could be seen by the other")
	}

	// A rotation: a new sealed segment, and the held view does not gain it.
	fillSegments(t, l, clk, 1)
	v3, rel3, ids3 := viewIDs(t, l)
	defer rel3()
	if !equalIDs(ids3, []uint64{0, 1, 2, 3}) {
		t.Fatalf("view after a rotation holds segments %v, want 0..3", ids3)
	}
	if !equalIDs(idsOf(v1), ids1) {
		t.Fatalf("a held view changed across a rotation: %v", idsOf(v1))
	}

	// A deletion of the oldest two: the held views keep them, the next view lacks them.
	clk.Advance(time.Hour - 2*time.Minute - 30*time.Second)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	_, rel4, ids4 := viewIDs(t, l)
	defer rel4()
	if len(ids4) >= len(ids3) || ids4[0] == 0 {
		t.Fatalf("view after a deletion holds segments %v, was %v", ids4, ids3)
	}
	if !equalIDs(idsOf(v3), ids3) || !equalIDs(idsOf(v1), ids1) {
		t.Fatalf("a held view changed across a deletion: %v, %v", idsOf(v1), idsOf(v3))
	}

	// The active segment is empty, so the view is exactly the sealed list.
	if want := sealedIDs(l); !equalIDs(ids4, want) {
		t.Fatalf("view holds %v, the log's sealed list is %v", ids4, want)
	}
}

func idsOf(v *segView) []uint64 {
	ids := make([]uint64, len(v.segs))
	for i, s := range v.segs {
		ids[i] = s.id
	}
	return ids
}

// pruneAfterView is a source that takes the Log's view and then runs a retention pass before
// the read opens any segment: the window a concurrent retention pass has in a real read.
type pruneAfterView struct {
	*Log
	once sync.Once
	err  error
}

func (p *pruneAfterView) view() (*segView, func(), error) {
	v, release, err := p.Log.view()
	if err == nil {
		p.once.Do(func() { p.err = p.Log.Prune(true) })
	}
	return v, release, err
}

// A sealed segment that retention deletes between the Log taking its view and the read opening
// the segment is a gap over that segment's range, from the sequence asked for, as the Reader
// reports it (decisions 101 and 103), not the raw path error of the open.
func TestALogReadOfASealedSegmentDeletedAfterTheViewIsAGapNotAPathError(t *testing.T) {
	for _, tc := range []struct {
		name     string
		from     uint64
		wantFrom uint64
		wantTo   uint64
	}{
		{"first segment", 1, 1, 4},
		{"later segment, mid-range", 6, 6, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, clk, l := retentionLog(t, nil)
			defer l.Close()
			fillSegments(t, l, clk, 6) // segments 0..5 sealed at minutes 0..5, 6 active and empty
			clk.Advance(time.Hour - 3*time.Minute - 30*time.Second)
			src := &pruneAfterView{Log: l}
			_, err := readFrom(src, tc.from, 10)
			if src.err != nil {
				t.Fatal(src.err)
			}
			var pe *fs.PathError
			if errors.As(err, &pe) {
				t.Fatalf("the path error escaped: %v", err)
			}
			if !isGap(err, tc.wantFrom, tc.wantTo) {
				t.Fatalf("readFrom(%d): %v (%T), want a gap %d..%d", tc.from, err, err, tc.wantFrom, tc.wantTo)
			}
		})
	}
}
