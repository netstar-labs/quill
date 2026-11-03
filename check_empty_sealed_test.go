package quill

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A sealed segment holds at least one frame (decision 43, guarantee 6). Check once skipped a
// zero-byte sealed file as if it were the active segment and said the log was healthy, while
// Open and a Reader's Stat reported the records it held as a hole and the writer's Err named
// the file. Check now reports it as fatal damage, and only the newest segment may be empty.
func TestCheckReportsAnEmptySealedSegmentAndAgreesWithOpenAndStat(t *testing.T) {
	now := time.Now().Unix()
	const want = "a sealed segment with no frames"
	// 120 records in 3 segments of 40; the file at position empty is then emptied or overwritten.
	for _, tc := range []struct {
		name    string
		empty   int
		content []byte // what replaces the file; nil truncates it to zero bytes
	}{
		{"oldest truncated", 0, nil},
		{"middle truncated", 1, nil},
		{"oldest garbage", 0, bytes.Repeat([]byte{0xA5}, 21)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			imgs := [][]byte{
				image(1, 40, now, smallPayload),
				image(41, 40, now+100, smallPayload),
				image(81, 40, now+200, smallPayload),
			}
			imgs[tc.empty] = tc.content
			r, _ := edgeLog(t, s, nil, 120, imgs...)
			victim := "/log/" + segmentName(uint64(tc.empty))

			res, err := r.Check()
			if err != nil || res.Healthy {
				t.Fatalf("Check: %+v, %v; want unhealthy", res, err)
			}
			var found *Problem
			for i, p := range res.Problems {
				if p.Kind == "segment" && p.Path == victim {
					found = &res.Problems[i]
				}
			}
			if found == nil || !found.Fatal || found.Offset != 0 || found.Intact != 0 {
				t.Fatalf("no fatal segment problem at byte 0 naming %s: %+v", victim, res.Problems)
			}
			if tc.content == nil && !strings.Contains(found.Detail, want) {
				t.Fatalf("detail %q, want %q", found.Detail, want)
			}

			// The same layout through the writer's Open, and for the oldest segment the
			// Reader's Stat. (A Reader's Stat does not yet report a hole where a middle
			// segment is empty; that is not what Check is tested against here.)
			l := openSim(t, s, nil)
			defer l.Close()
			st, err := l.Stat()
			if err != nil || len(st.Holes) == 0 {
				t.Fatalf("writer stat: holes %+v, %v; want a hole", st.Holes, err)
			}
			if tc.empty == 0 {
				rst, err := r.Stat()
				if err != nil || len(rst.Holes) != len(st.Holes) || rst.Holes[0] != st.Holes[0] {
					t.Fatalf("writer holes %+v, reader holes %+v, %v; want the same", st.Holes, rst.Holes, err)
				}
			}
			if e := l.Err(); e == nil || !strings.Contains(e.Error(), segmentName(uint64(tc.empty))) {
				t.Fatalf("Err() = %v, want it to name %s", e, segmentName(uint64(tc.empty)))
			}
		})
	}

	t.Run("an empty active segment is healthy", func(t *testing.T) {
		r, _ := edgeLog(t, fsx.NewSim(), nil, 80,
			image(1, 40, now, smallPayload),
			image(41, 40, now+100, smallPayload),
			nil,
		)
		res, err := r.Check()
		if err != nil || !res.Healthy {
			t.Fatalf("Check: %+v, %v; want healthy", res, err)
		}
	})
}
