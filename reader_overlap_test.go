package quill

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// overlapLog builds, by hand, a directory the writer refuses (Open: ErrCorrupt) and that
// quill never produces: segment 0 holds sequences 1 to 10 and the active segment 1 holds 5 to
// 8, the marker naming 8. The payloads say which segment a record came from. With sealed set,
// segment 1 is sealed and an active segment 2 holds 9 to 12, the marker naming 12.
func overlapLog(t *testing.T, sealed bool) *fsx.Sim {
	t.Helper()
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 1)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	build := func(first, last uint64, name string) []byte {
		var b []byte
		for seq := first; seq <= last; seq++ {
			b = appendFrame(b, &Record{Seq: seq, At: 1000 + int64(seq), Payload: []byte(fmt.Sprintf("%s-%d", name, seq))})
		}
		return b
	}
	seg1 := build(5, 8, "seg1")
	writeFile(t, s, "/log/"+segmentName(0), build(1, 10, "seg0"))
	writeFile(t, s, "/log/"+segmentName(1), seg1)
	if sealed {
		seg2 := build(9, 12, "seg2")
		writeFile(t, s, "/log/"+segmentName(2), seg2)
		writeFile(t, s, "/log/"+tailName, encodeTail(tail{Seq: 12, Seg: 2, Off: uint64(len(seg2)), MaxSeg: 2}))
	} else {
		writeFile(t, s, "/log/"+tailName, encodeTail(tail{Seq: 8, Seg: 1, Off: uint64(len(seg1)), MaxSeg: 1}))
	}
	if _, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("setup: the writer's Open = %v, want ErrCorrupt", err)
	}
	return s
}

// Open refuses overlapping segments (decision 102); a Reader must not serve the earlier
// segment's records for sequences the next one also claims (guarantee 6).
func TestReaderRefusesOverlappingSegmentsAsCorrupt(t *testing.T) {
	const want = "inside the one before"
	t.Run("Read", func(t *testing.T) {
		r := openReaderSim(t, overlapLog(t, false))
		defer r.Close()
		for i := 0; i < 2; i++ { // the first call scans segment 0, the second has its result
			recs, err := r.Read(1, 100)
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("call %d: Read(1) = %v, %v, want ErrCorrupt", i, seqsOf(recs), err)
			}
			if len(recs) != 0 {
				t.Fatalf("call %d: served %v from an overlapping log", i, seqsOf(recs))
			}
		}
	})
	t.Run("ScanAll", func(t *testing.T) {
		r := openReaderSim(t, overlapLog(t, false))
		defer r.Close()
		for i := 0; i < 2; i++ {
			recs, err := readerAll(r)
			if !errors.Is(err, ErrCorrupt) || len(recs) != 0 {
				t.Fatalf("call %d: ScanAll = %v, %v, want ErrCorrupt and nothing served", i, seqsOf(recs), err)
			}
		}
	})
	t.Run("Check", func(t *testing.T) {
		r := openReaderSim(t, overlapLog(t, false))
		defer r.Close()
		res, err := r.Check()
		if err != nil {
			t.Fatal(err)
		}
		var got string
		for _, p := range res.Problems {
			got += p.String() + "\n"
		}
		if res.Healthy || !strings.Contains(got, want) || strings.Contains(got, "missing") {
			t.Fatalf("Check: healthy=%v problems:\n%s\nwant an overlap naming %q and no hole", res.Healthy, got, want)
		}
	})
}

// Archive reaches the segments through the same view as a read, so it must refuse the overlap
// too, not copy both segments, publish a marker below the last record it copied and report a
// gap whose range runs backwards.
func TestArchiveRefusesOverlappingSegmentsAndCopiesNeitherOfThem(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sealed bool
	}{{"two segments", false}, {"sealed overlap", true}} {
		t.Run(tc.name, func(t *testing.T) {
			s := overlapLog(t, tc.sealed)
			err := archive(s, "/log", "/arch")
			var gap *GapError
			if !errors.Is(err, ErrCorrupt) || errors.As(err, &gap) {
				t.Fatalf("Archive = %v, want ErrCorrupt and no GapError", err)
			}
			if !strings.Contains(err.Error(), "inside the next segment") {
				t.Fatalf("Archive = %v, want the overlap named", err)
			}
			if ids := segmentFiles(t, s, "/arch"); len(ids) != 0 {
				t.Fatalf("copied segments %v of an overlapping log", ids)
			}
			if dt, err := decodeTail(fileBytes(t, s, "/arch/"+tailName)); err != nil || dt.Seq != 0 {
				t.Fatalf("destination marker = %+v, %v, want the empty one", dt, err)
			}
		})
	}
}
