package quill

import (
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A Read sizes its result slice once, when the first record is delivered: a Read past the head
// and a Read that starts in a gap return a nil slice, as they did before the slice was sized,
// and a Read that delivers records returns exactly them.
func TestReadSizesItsResultOnlyWhenARecordIsDelivered(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.MaxRecords = 10 })
	defer l.Close()
	fillSegments(t, l, clk, 6) // 24 records
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}

	recs, err := l.Read(1, 0) // below the first retained sequence
	if recs != nil || err == nil {
		t.Fatalf("gap read: recs nil=%v err=%v", recs == nil, err)
	}
	recs, err = l.Read(25, 10) // past the head
	if recs != nil || err != nil {
		t.Fatalf("empty read: recs nil=%v err=%v", recs == nil, err)
	}
	recs, err = l.Read(17, 100) // limit above what remains: capped by head - seq + 1
	if err != nil || len(recs) != 8 || cap(recs) != 8 {
		t.Fatalf("read: len %d cap %d err %v", len(recs), cap(recs), err)
	}
	recs, err = l.Read(17, 3)
	if err != nil || len(recs) != 3 || cap(recs) != 3 {
		t.Fatalf("limited read: len %d cap %d err %v", len(recs), cap(recs), err)
	}
}
