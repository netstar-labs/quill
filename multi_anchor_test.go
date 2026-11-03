package quill

import (
	"bytes"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// The sparse index (decision 63) is the one part of the read path that skips bytes: a read
// starts at the last anchor at or below the requested sequence, a time seek at the last
// anchor strictly before the target. Every other test keeps a segment under one stride, so
// the index holds a single anchor and the choice between anchors is never made.
func TestMultiAnchorSegmentReadsAndSeeks(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentBytes = 4 << 20; c.Retention = -1 })
	defer l.Close()
	const n, size, perSecond = 400, 1000, 10
	var all []Record
	for i := 1; i <= n; i++ {
		payload := bytes.Repeat([]byte{byte(i)}, size)
		seq, err := l.Append(payload)
		if err != nil || seq != uint64(i) {
			t.Fatalf("append %d: %d %v", i, seq, err)
		}
		all = append(all, Record{Seq: seq, At: clk.Now().Unix(), Payload: payload})
		if i%perSecond == 0 {
			clk.Advance(time.Second) // runs of equal times, several per anchor stride
		}
	}
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 2 { // the sealed segment holds every record, the new active one is empty
		t.Fatalf("segments %v, want one sealed segment and an empty active one", ids)
	}
	if got := len(fileBytes(t, s, "/log/"+segmentName(ids[0]))) / indexStride; got < 5 {
		t.Fatalf("the sealed segment spans only %d strides: too few anchors to choose between", got)
	}
	r := openReaderSim(t, s)
	defer r.Close()

	for name, src := range map[string]source{"log": l, "reader": r} {
		for offset := uint64(1); offset <= n; offset++ {
			recs, err := readFrom(src, offset, 3)
			if err != nil {
				t.Fatalf("%s: read %d: %v", name, offset, err)
			}
			want := all[offset-1 : min(offset+2, n)]
			if len(recs) != len(want) {
				t.Fatalf("%s: read %d: %d records, want %d", name, offset, len(recs), len(want))
			}
			for i, rec := range recs {
				if rec.Seq != want[i].Seq || !bytes.Equal(rec.Payload, want[i].Payload) {
					t.Fatalf("%s: read %d: record %d is seq %d, want %d", name, offset, i, rec.Seq, want[i].Seq)
				}
			}
		}
		for sec := all[0].At - 2; sec <= all[n-1].At+2; sec++ {
			want := linear(all, sec, 1, n)
			got, err := offsetForTime(src, time.Unix(sec, 0))
			if err != nil || got != want {
				t.Fatalf("%s: second %d: %d %v, want %d", name, sec, got, err, want)
			}
		}
	}
}
