package quill

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Offset 0 on Read, Records and ReadWait is below every sequence, so on a log whose oldest
// records were trimmed by retention it is a read from below First and meets the gap (decisions
// 38 and 84), whereas Scan(0) and ScanAll start at First. Read must not skip the trimmed prefix
// and hand back what is left as if nothing was lost.
func TestReadAndRecordsFromZeroAfterRetentionAreAGapButScanFromZeroIsNot(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 6)
	clk.Advance(time.Hour - 3*time.Minute - 30*time.Second) // the segments of minutes 0..2 expire
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.Stat(); st.First != 13 {
		t.Fatalf("%+v: want First 13", st)
	}
	r := openReaderSim(t, s)
	defer r.Close()

	type reads interface {
		Read(uint64, int) ([]Record, error)
		Records(context.Context, uint64) iter.Seq2[Record, error]
		Scan(uint64, int, func(Record) bool) error
	}
	for name, src := range map[string]reads{"log": l, "reader": r} {
		t.Run(name, func(t *testing.T) {
			if recs, err := src.Read(0, 10); len(recs) != 0 || !isGap(err, 1, 12) {
				t.Errorf("Read(0): %v, %v; want the gap 1..12 and no records", seqsOf(recs), err)
			}
			var yielded []uint64
			var gotErr error
			for rec, err := range src.Records(context.Background(), 0) {
				if err != nil {
					gotErr = err
					break
				}
				yielded = append(yielded, rec.Seq)
			}
			if len(yielded) != 0 || !isGap(gotErr, 1, 12) {
				t.Errorf("Records(0): %v, %v; want the gap 1..12 and no records", yielded, gotErr)
			}
			var scanned []uint64
			err := src.Scan(0, 0, func(rec Record) bool { scanned = append(scanned, rec.Seq); return true })
			if err != nil || len(scanned) != 12 || scanned[0] != 13 {
				t.Errorf("Scan(0): %v, %v; want records 13..24", scanned, err)
			}
		})
	}
}

// ReadWait at offset 0 on an empty log waits for a record and returns when its context ends. A
// wait that treats 0 as already satisfied returns at once, finds nothing and spins without ever
// looking at the context.
func TestReadWaitAtOffsetZeroOnAnEmptyLogWaitsAndHonoursItsContext(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	r := openReaderSim(t, s)
	defer r.Close()

	for name, wait := range map[string]func(context.Context, uint64, int) ([]Record, error){
		"log": l.ReadWait, "reader": r.ReadWait,
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				recs, err := wait(ctx, 0, 10)
				if len(recs) != 0 {
					err = errors.New("records on an empty log")
				}
				done <- err
			}()
			select {
			case err := <-done:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("ReadWait(0): %v; want the context's deadline", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("ReadWait(ctx, 0, n) on an empty log ignored its context")
			}
		})
	}
}
