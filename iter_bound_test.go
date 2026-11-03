package quill

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// The iterators' two loop-level guarantees (decision 58): a batch is bounded by MaxReadBytes
// before it is yielded, and the context is checked on every pass, so cancellation ends a
// sequence that always finds records.

// segmentsPerRecordSim returns a simulator holding a log with one record in each of five
// segments, and a counter of the segment files opened since the counter was armed.
func segmentsPerRecordSim(t *testing.T) (*fsx.Sim, *Log, *atomic.Int64, *atomic.Bool) {
	t.Helper()
	s := fsx.NewSim()
	var opens atomic.Int64
	var armed atomic.Bool
	s.Inject = func(op fsx.Op, name string) error {
		if armed.Load() && op == fsx.OpOpen && strings.HasSuffix(name, ".log") {
			opens.Add(1)
		}
		return nil
	}
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1; c.MaxReadBytes = 1 })
	for i := 0; i < 5; i++ {
		appendN(t, l, 1)
		mustFlush(t, l)
	}
	// The sixth file is the empty segment the last rotation opened.
	if n := len(segmentFiles(t, s, "/log")); n != 6 {
		t.Fatalf("%d segment files, want 5 full and 1 empty", n)
	}
	return s, l, &opens, &armed
}

func openBoundedReaderSim(t *testing.T, s *fsx.Sim) *Reader {
	t.Helper()
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond, MaxReadBytes: 1})
	if err != nil {
		t.Fatalf("open reader: %v", err)
	}
	return r
}

func TestL2IteratorBatchesAreBoundedByMaxReadBytes(t *testing.T) {
	_, l, opens, armed := segmentsPerRecordSim(t)
	defer l.Close()
	// Only the Log is counted: a Reader's view probes every sealed segment's first frame on
	// each scan, so its file opens say nothing about how much a batch reads.
	for name, seq := range map[string]func(context.Context) func(func(Record, error) bool){
		"log records": func(ctx context.Context) func(func(Record, error) bool) { return l.Records(ctx, 1) },
		"log follow":  func(ctx context.Context) func(func(Record, error) bool) { return l.Follow(ctx, 1) },
	} {
		ctx, cancel := context.WithCancel(context.Background())
		opens.Store(0)
		armed.Store(true)
		var atFirst int64 = -1
		for rec, err := range seq(ctx) {
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			atFirst = opens.Load()
			if rec.Seq != 1 {
				t.Fatalf("%s: first record %d", name, rec.Seq)
			}
			break
		}
		armed.Store(false)
		cancel()
		// MaxReadBytes of 1 stops a batch after its first record, so the first record arrives
		// after at most two segment files were read (one record, then the stop), never after
		// the whole log was read and copied.
		if atFirst < 1 || atFirst > 2 {
			t.Fatalf("%s: %d segment files read before the first record, want at most 2 of 5", name, atFirst)
		}
	}
}

func TestL2RecordsOnADoneContextYieldsTheContextErrorAndNoRecords(t *testing.T) {
	s, l, _, _ := segmentsPerRecordSim(t)
	defer l.Close()
	r := openBoundedReaderSim(t, s)
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, seq := range map[string]func(func(Record, error) bool){
		"log records":    l.Records(ctx, 1),
		"log follow":     l.Follow(ctx, 1),
		"reader records": r.Records(ctx, 1),
		"reader follow":  r.Follow(ctx, 1),
	} {
		got, err := collect(seq, 0)
		if !errors.Is(err, context.Canceled) || len(got) != 0 {
			t.Fatalf("%s: %d records, error %v; want no records and context.Canceled", name, len(got), err)
		}
	}
}

func TestL2FollowCancelledMidBacklogStopsWithTheContextError(t *testing.T) {
	s, l, _, _ := segmentsPerRecordSim(t)
	defer l.Close()
	r := openBoundedReaderSim(t, s)
	defer r.Close()
	for name, mk := range map[string]func(context.Context) func(func(Record, error) bool){
		"log":    func(ctx context.Context) func(func(Record, error) bool) { return l.Follow(ctx, 1) },
		"reader": func(ctx context.Context) func(func(Record, error) bool) { return r.Follow(ctx, 1) },
	} {
		ctx, cancel := context.WithCancel(context.Background())
		var seqs []uint64
		var last error
		for rec, err := range mk(ctx) {
			if err != nil {
				last = err
				break
			}
			seqs = append(seqs, rec.Seq)
			if rec.Seq == 2 {
				cancel() // the backlog (seq 3 to 5) is still there to be read
			}
		}
		cancel()
		if !errors.Is(last, context.Canceled) || len(seqs) != 2 {
			t.Fatalf("%s: records %v, error %v; want 1,2 then context.Canceled", name, seqs, last)
		}
	}
}
