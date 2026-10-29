package quill

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// T26: seek by time.

// timedLog builds a log whose records have known times: the clock advances by step after
// each record, with runs of equal timestamps, one of which starts at the end of a segment
// and continues into the next.
func timedLog(t *testing.T, s *fsx.Sim, clk *testClock) (*Log, []Record) {
	t.Helper()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentBytes = 60; c.Retention = -1 })
	var all []Record
	for i := 0; i < 80; i++ {
		seq, err := l.Append(payloadFor(uint64(i + 1)))
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, Record{Seq: seq, At: clk.Now().Unix()})
		switch {
		case i >= 14 && i <= 18: // a run of equal times across the segment boundary at 16|17
		case i%11 == 3: // other short runs
		default:
			clk.Advance(time.Second)
		}
		if i%4 == 3 {
			mustFlush(t, l)
		}
	}
	mustFlush(t, l)
	return l, all
}

func linear(all []Record, target int64, first, head uint64) uint64 {
	for _, r := range all {
		if r.Seq < first {
			continue
		}
		if r.At >= target {
			return r.Seq
		}
	}
	return head + 1
}

func TestOffsetForTimeEqualsALinearScanForEverySecond(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l, all := timedLog(t, s, clk)
	defer l.Close()
	if ids := segmentFiles(t, s, "/log"); len(ids) < 15 {
		t.Fatalf("want many segments, got %d", len(ids))
	}
	r := openReaderSim(t, s)
	defer r.Close()
	from, to := all[0].At-5, all[len(all)-1].At+5
	for sec := from; sec <= to; sec++ {
		want := linear(all, sec, 1, 80)
		for name, src := range map[string]source{"log": l, "reader": r} {
			got, err := offsetForTime(src, time.Unix(sec, 0))
			if err != nil || got != want {
				t.Fatalf("%s: second %d: %d %v, want %d", name, sec, got, err, want)
			}
		}
	}
	// Sub-second times round down to the second: a record stamped in the same second counts.
	if got, _ := l.OffsetForTime(time.Unix(all[20].At, 500_000_000)); got != linear(all, all[20].At, 1, 80) {
		t.Fatalf("sub-second: %d", got)
	}
	// After retention deleted a prefix (about the first half), on both.
	l.cfg.Retention = clk.Now().Sub(time.Unix(all[40].At, 0))
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	st, _ := l.Stat()
	if st.First <= 1 || st.First > 45 {
		t.Fatalf("first retained %d", st.First)
	}
	for sec := from; sec <= to; sec++ {
		want := linear(all, sec, st.First, 80)
		for name, src := range map[string]source{"log": l, "reader": r} {
			got, err := offsetForTime(src, time.Unix(sec, 0))
			if err != nil || got != want {
				t.Fatalf("%s after retention: second %d: %d %v, want %d", name, sec, got, err, want)
			}
		}
	}
	// SeekToTime then Records starts at the right record, and the cost on a warm reader
	// touches one segment: a handful of operations, whatever the number of segments.
	c, err := l.Consumer("seek", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	target := all[50].At
	if err := c.SeekToTime(time.Unix(target, 0)); err != nil {
		t.Fatal(err)
	}
	recs, err := collect(c.Records(context.Background()), 1)
	if err != nil || len(recs) != 1 || recs[0].Seq != linear(all, target, st.First, 80) {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	_, _ = r.OffsetForTime(time.Unix(target, 0)) // warm the one segment
	before := s.Ops()
	if _, err := r.OffsetForTime(time.Unix(target, 0)); err != nil {
		t.Fatal(err)
	}
	if ops := s.Ops() - before; ops > 25 {
		t.Fatalf("a warm seek cost %d file operations", ops)
	}
}

func TestOffsetForTimeOnDamageEmptyLogsAndNonMonotonicTimes(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	if off, err := l.OffsetForTime(clk.Now()); err != nil || off != 1 {
		t.Fatalf("empty log: %d %v", off, err)
	}
	// Non-monotonic: a clock stepped back. Never a panic, always an offset in range.
	for i := 0; i < 40; i++ {
		appendN(t, l, 1)
		if i%7 == 6 {
			clk.Advance(-20 * time.Second)
		} else {
			clk.Advance(5 * time.Second)
		}
		if i%4 == 3 {
			mustFlush(t, l)
		}
	}
	mustFlush(t, l)
	for sec := clk.Now().Unix() - 400; sec <= clk.Now().Unix()+10; sec++ {
		off, err := l.OffsetForTime(time.Unix(sec, 0))
		if err != nil || off < 1 || off > 41 {
			t.Fatalf("second %d: %d %v", sec, off, err)
		}
	}
	// A damaged chosen segment is a gap.
	seg := "/log/" + segmentName(2)
	data := fileBytes(t, s, seg)
	data[len(data)/2] ^= 0xff
	writeFile(t, s, seg, data)
	r := openReaderSim(t, s)
	defer r.Close()
	ids := segmentFiles(t, s, "/log")
	_ = ids
	var hit bool
	for sec := clk.Now().Unix() - 400; sec <= clk.Now().Unix(); sec++ {
		_, err := r.OffsetForTime(time.Unix(sec, 0))
		var g *GapError
		if errors.As(err, &g) {
			hit = true
		} else if err != nil {
			t.Fatalf("%v", err)
		}
	}
	if !hit {
		t.Fatal("the damaged segment was never chosen")
	}
}
