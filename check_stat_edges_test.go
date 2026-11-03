package quill

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// edgeLog builds a log by hand: the given segment images as segments 0, 1, ... with the last
// one active and wholly durable, the marker naming lastSeq, and a Reader on it that sees the
// given clock (nil is the real one). It returns the log's ID so a test can plant consumers.
func edgeLog(t *testing.T, s *fsx.Sim, clock func() time.Time, lastSeq uint64, imgs ...[]byte) (*Reader, string) {
	t.Helper()
	l := openSim(t, s, nil)
	appendN(t, l, 1)
	id := l.ID()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	for i, img := range imgs {
		writeFile(t, s, "/log/"+segmentName(uint64(i)), img)
	}
	last := uint64(len(imgs) - 1)
	writeFile(t, s, "/log/"+tailName, encodeTail(tail{Seq: lastSeq, Seg: last, Off: uint64(len(imgs[last])), MaxSeg: last}))
	r, err := OpenReader("/log", ReaderOptions{fs: s, clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r, id
}

func problemsOfKind(res CheckResult, kind string) []string {
	var out []string
	for _, p := range res.Problems {
		if p.Kind == kind {
			out = append(out, fmt.Sprintf("%s fatal=%v: %s", p.Path, p.Fatal, p.Detail))
		}
	}
	return out
}

// Two segments that share one sequence are an overlap, and two that miss one sequence are a
// hole of exactly that sequence: the off-by-one neighbours of each other (decision 102).
func TestCheckNamesAnOverlapAndAHoleOfOneSequence(t *testing.T) {
	now := time.Now().Unix()
	t.Run("overlap of one sequence", func(t *testing.T) {
		r, _ := edgeLog(t, fsx.NewSim(), nil, 6,
			image(1, 3, now, smallPayload),    // 1..3
			image(3, 2, now+10, smallPayload), // 3..4: starts at the last sequence of the one before
			image(5, 2, now+20, smallPayload), // 5..6
		)
		res, err := r.Check()
		if err != nil || res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
		want := []string{"/log/" + segmentName(1) + " fatal=true: segment 1 starts at 3, inside the one before (segment 0, ends at 3)"}
		if got := problemsOfKind(res, "overlap"); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("overlap findings:\n got %q\nwant %q", got, want)
		}
		if got := problemsOfKind(res, "hole"); len(got) != 0 {
			t.Fatalf("an overlap was also reported as a hole: %q", got)
		}
	})
	t.Run("hole of one sequence", func(t *testing.T) {
		r, _ := edgeLog(t, fsx.NewSim(), nil, 8,
			image(1, 3, now, smallPayload),    // 1..3
			image(5, 2, now+10, smallPayload), // 5..6: 4 is missing
			image(7, 2, now+20, smallPayload), // 7..8
		)
		res, err := r.Check()
		if err != nil || res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
		want := []string{"/log/" + segmentName(1) + " fatal=true: sequences 4 to 4 are missing before it"}
		if got := problemsOfKind(res, "hole"); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("hole findings:\n got %q\nwant %q", got, want)
		}
		if got := problemsOfKind(res, "overlap"); len(got) != 0 {
			t.Fatalf("a hole was also reported as an overlap: %q", got)
		}
	})
}

// A consumer one sequence past the head (Head+2) is ahead of the log, one at Head+1 is
// caught up; a consumer one below the first retained sequence has lost a record, one at it has
// not (decisions 36 and 38). Offsets are next-to-read, so the edges are exactly these.
func TestCheckWarnsOfAConsumerOneAheadOrOneBehind(t *testing.T) {
	now := time.Now().Unix()
	plant := func(t *testing.T, s *fsx.Sim, id string, offsets map[string]uint64) {
		for name, next := range offsets {
			writeFile(t, s, "/log/consumers/"+name, encodeConsumer(consumerState{ID: logID(id), Next: next}))
		}
	}
	consumerLines := func(t *testing.T, r *Reader) []string {
		res, err := r.Check()
		if err != nil || !res.Healthy {
			t.Fatalf("Check: %+v, %v (a consumer finding is a warning, never fatal)", res, err)
		}
		return problemsOfKind(res, "consumer")
	}
	path := func(name string) string { return "/log/consumers/" + name }

	t.Run("ahead and caught up", func(t *testing.T) {
		s := fsx.NewSim()
		r, id := edgeLog(t, s, nil, 5, image(3, 3, now, smallPayload)) // 3..5
		plant(t, s, id, map[string]uint64{"onepast": 7, "caughtup": 6})
		want := []string{path("onepast") + " fatal=false: at 7, ahead of the head 5: rewind it with quill commit"}
		if got := consumerLines(t, r); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})
	t.Run("one behind the first retained and at it", func(t *testing.T) {
		s := fsx.NewSim()
		r, id := edgeLog(t, s, nil, 5, image(3, 3, now, smallPayload)) // 3..5
		plant(t, s, id, map[string]uint64{"onebehind": 2, "atfirst": 3})
		want := []string{path("onebehind") + " fatal=false: at 2, behind the first retained sequence 3: records 2 to 2 are gone"}
		if got := consumerLines(t, r); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})
	t.Run("a log with nothing retained", func(t *testing.T) {
		// Retention took every record: the first retained sequence is Head+1, so a consumer
		// at the head has lost the record it was about to read, and one at Head+1 has not.
		s := fsx.NewSim()
		r, id := edgeLog(t, s, nil, 5, nil)
		plant(t, s, id, map[string]uint64{"atthehead": 5, "caughtup": 6})
		want := []string{path("atthehead") + " fatal=false: at 5, behind the first retained sequence 6: records 5 to 5 are gone"}
		if got := consumerLines(t, r); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})
}

// "More than a day in the future" is strictly more: a record dated exactly a day ahead of the
// reader's clock is not flagged, one second more is, for a segment's first record and for the
// records after it.
func TestCheckDoesNotCallARecordExactlyADayAheadFutureDated(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	day := now.Add(24 * time.Hour).Unix()
	clock := func() time.Time { return now }
	futureLines := func(res CheckResult) []string {
		var out []string
		for _, p := range res.Problems {
			if p.Kind == "clock" && strings.Contains(p.Detail, "in the future") {
				out = append(out, p.Detail)
			}
		}
		return out
	}
	t.Run("a later record", func(t *testing.T) {
		r, _ := edgeLog(t, fsx.NewSim(), clock, 3, image(1, 3, day-1, smallPayload)) // day-1, day, day+1
		res, err := r.Check()
		if err != nil || !res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
		want := []string{fmt.Sprintf("record 3 at %d is more than a day in the future", day+1)}
		if got := futureLines(res); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})
	t.Run("a first record", func(t *testing.T) {
		r, _ := edgeLog(t, fsx.NewSim(), clock, 2, image(1, 1, day, smallPayload), image(2, 1, day+1, smallPayload))
		res, err := r.Check()
		if err != nil || !res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
		want := []string{fmt.Sprintf("record 2 at %d is more than a day in the future", day+1)}
		if got := futureLines(res); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})
}

// With exactly one retained record, Head equals First and Records is 1, from the writer and
// from a reader (Head - First + 1).
func TestStatCountsASingleRetainedRecord(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 1)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()
	for name, stat := range map[string]func() (Stats, error){"Log.Stat": l.Stat, "Reader.Stat": r.Stat} {
		st, err := stat()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if st.Head != 1 || st.First != 1 || st.Records != 1 {
			t.Errorf("%s: Head %d First %d Records %d, want 1, 1, 1", name, st.Head, st.First, st.Records)
		}
	}
}

// Bytes is the sum of the segment files' sizes, the active segment's durable bytes included.
func TestStatBytesCountsTheActiveSegmentToo(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	// 512-byte segments: flushing after every few records seals some of them, and the last
	// record is written after the last seal, so the active segment is not empty.
	for i := 0; i < 10; i++ {
		appendN(t, l, 3)
		mustFlush(t, l)
	}
	appendN(t, l, 1)
	mustFlush(t, l)
	var want int64
	ids := segmentFiles(t, s, "/log")
	for _, id := range ids {
		want += int64(len(fileBytes(t, s, "/log/"+segmentName(id))))
	}
	if len(ids) < 2 {
		t.Fatalf("setup: %d segments, want a sealed one and an active one", len(ids))
	}
	if n := len(fileBytes(t, s, "/log/"+segmentName(ids[len(ids)-1]))); n == 0 {
		t.Fatalf("setup: the active segment is empty (ids %v)", ids)
	}
	st, err := l.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if st.Segments != len(ids) || st.Bytes != want {
		t.Fatalf("Segments %d Bytes %d, want %d and %d", st.Segments, st.Bytes, len(ids), want)
	}
}
