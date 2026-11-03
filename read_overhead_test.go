package quill

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// MaxReadBytes bounds the size of a batch, not the sum of the payloads: each record is charged
// its payload plus 48 bytes of overhead, so a log of empty or tiny payloads cannot make one
// read allocate for every record it holds (L2-53).

// viewCounter counts the scans a source is asked for: scanSource takes one view per call, so
// the count is the number of batches an iterator read, the last one empty.
type viewCounter struct {
	source
	views atomic.Int64
}

func (c *viewCounter) view() (*segView, func(), error) {
	c.views.Add(1)
	return c.source.view()
}

func TestMaxReadBytesChargesEveryRecordItsOverheadSoEmptyPayloadsAreBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("appends a million records")
	}
	const total = 1_000_000
	l := openDir(t, t.TempDir(), func(c *Config) { c.SegmentBytes = 8 << 20 })
	defer l.Close()
	batch := make([][]byte, 100_000)
	for i := 0; i < total/len(batch); i++ {
		if _, err := l.AppendBatch(batch); err != nil {
			t.Fatal(err)
		}
	}
	mustFlush(t, l)

	// The read stops once 48 bytes per record reach the cap: record 349,526 is the first to
	// carry the total to 16 MiB or past it.
	const perBatch = DefaultMaxReadBytes/48 + 1
	recs, err := l.Read(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != perBatch {
		t.Fatalf("Read(1, 0) of empty payloads returned %d records, want %d (16 MiB / 48, rounded up)", len(recs), perBatch)
	}
	for i, rec := range recs {
		if rec.Seq != uint64(i+1) || len(rec.Payload) != 0 {
			t.Fatalf("record %d is seq %d with %d payload bytes", i, rec.Seq, len(rec.Payload))
		}
	}

	// Records yields every record, in order, over several bounded batches.
	src := &viewCounter{source: l}
	var next uint64 = 1
	for rec, err := range records(context.Background(), src, 1, false) {
		if err != nil {
			t.Fatal(err)
		}
		if rec.Seq != next {
			t.Fatalf("record %d where %d was due", rec.Seq, next)
		}
		next++
	}
	if next != total+1 {
		t.Fatalf("Records yielded %d records, want %d", next-1, total)
	}
	// ceil(1,000,000 / 349,526) = 3 batches, then the empty scan that ends the iteration.
	if got := src.views.Load(); got != 4 {
		t.Fatalf("Records took %d scans, want 4 (three bounded batches and the empty one)", got)
	}

	// The public iterator yields the same records.
	var n int
	for _, err := range l.Records(context.Background(), 1) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != total {
		t.Fatalf("Log.Records yielded %d, want %d", n, total)
	}
}

func TestMaxReadBytesOverheadStillReturnsARecordWhoseOwnPayloadPassesTheCap(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.MaxReadBytes = 1000; c.SegmentBytes = 1 << 20 })
	defer l.Close()
	if _, err := l.Append(make([]byte, 5000)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 33; i++ {
		if _, err := l.Append(nil); err != nil {
			t.Fatal(err)
		}
	}
	mustFlush(t, l)
	recs, err := l.Read(1, 0)
	if err != nil || len(recs) != 1 || len(recs[0].Payload) != 5000 {
		t.Fatalf("%v %v; want the 5000-byte record alone", seqsOf(recs), err)
	}
	// 48 bytes each: the 21st empty record brings the total to 1008 and ends the read.
	recs, err = l.Read(2, 0)
	if err != nil || len(recs) != 21 {
		t.Fatalf("%d records, %v; want 21 (21*48 = 1008 >= 1000)", len(recs), err)
	}
}
