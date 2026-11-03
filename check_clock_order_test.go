package quill

import (
	"fmt"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// clockLog writes a three-segment log (segments 0 and 1 sealed, 2 active and wholly durable)
// from the given images and returns a Reader on it.
func clockLog(t *testing.T, s *fsx.Sim, imgs [3][]byte, lastSeq uint64) *Reader {
	t.Helper()
	l := openSim(t, s, nil)
	t.Cleanup(func() { l.Close() })
	for i, img := range imgs {
		writeFile(t, s, "/log/"+segmentName(uint64(i)), img)
	}
	writeFile(t, s, "/log/"+tailName, encodeTail(tail{Seq: lastSeq, Seg: 2, Off: uint64(len(imgs[2])), MaxSeg: 2}))
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func clockLines(res CheckResult) []string {
	var out []string
	for _, p := range res.Problems {
		if p.Kind == "clock" {
			out = append(out, p.Path+" "+p.Detail)
		}
	}
	return out
}

// The clock tests run across segment boundaries: a segment's first record is judged against
// the last record of the segment before it, and the warnings come in segment order, then
// record order, whatever order the segments were verified in.
func TestCheckClockFindingsCrossSegmentBoundariesInSegmentOrder(t *testing.T) {
	now := time.Now().Unix()
	ahead := now + 48*3600
	seg := func(id uint64) string { return "/log/" + segmentName(id) }

	t.Run("first record earlier than the previous segment's last, and in the future", func(t *testing.T) {
		s := fsx.NewSim()
		r := clockLog(t, s, [3][]byte{
			image(1, 2, now, smallPayload),    // 1 at now, 2 at now+1
			image(3, 2, now-50, smallPayload), // 3 at now-50 (earlier than now+1), 4 at now-49
			image(5, 2, ahead, smallPayload),  // 5 and 6 are more than a day ahead
		}, 6)
		res, err := r.Check()
		if err != nil || !res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
		want := []string{
			seg(1) + fmt.Sprintf(" record 3 at %d is earlier than its predecessor at %d", now-50, now+1),
			seg(2) + fmt.Sprintf(" record 5 at %d is more than a day in the future", ahead),
			seg(2) + fmt.Sprintf(" record 6 at %d is more than a day in the future", ahead+1),
		}
		got := clockLines(res)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("clock findings:\n got %q\nwant %q", got, want)
		}
	})

	t.Run("the records before a damaged frame still set the predecessor", func(t *testing.T) {
		s := fsx.NewSim()
		d1 := image(3, 3, now+10, smallPayload) // 3, 4, 5 at now+10, +11, +12
		d1[len(d1)-2] ^= 1                      // record 5's frame is bad: 3 and 4 are accepted
		r := clockLog(t, s, [3][]byte{
			image(1, 2, now, smallPayload),
			d1,
			image(5, 1, now+5, smallPayload), // earlier than record 4 at now+11
		}, 5)
		res, err := r.Check()
		if err != nil || res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
		want := []string{seg(2) + fmt.Sprintf(" record 5 at %d is earlier than its predecessor at %d", now+5, now+11)}
		if got := clockLines(res); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("clock findings:\n got %q\nwant %q", got, want)
		}
	})

	t.Run("a segment with no good frame leaves the predecessor as it was", func(t *testing.T) {
		s := fsx.NewSim()
		d1 := image(3, 2, now+10, smallPayload)
		d1[2] ^= 1 // the first frame is bad
		r := clockLog(t, s, [3][]byte{
			image(1, 2, now, smallPayload), // last at now+1
			d1,
			image(5, 1, now-5, smallPayload),
		}, 5)
		res, err := r.Check()
		if err != nil || res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
		want := []string{seg(2) + fmt.Sprintf(" record 5 at %d is earlier than its predecessor at %d", now-5, now+1)}
		if got := clockLines(res); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("clock findings:\n got %q\nwant %q", got, want)
		}
	})
}
