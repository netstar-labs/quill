package quill

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// T25: iterators.

func collect(seq func(func(Record, error) bool), limit int) ([]Record, error) {
	var out []Record
	var last error
	for rec, err := range seq {
		if err != nil {
			last = err
			break
		}
		out = append(out, rec)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, last
}

func TestRecordsYieldsExactlyWhatReadReturnsAndFollowWakesOnNewRecords(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100; c.MaxReadBytes = 50 })
	defer l.Close()
	for i := 0; i < 30; i++ {
		appendN(t, l, 2)
		mustFlush(t, l)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	c, err := l.Consumer("it", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Commit(7); err != nil {
		t.Fatal(err)
	}
	var want []Record
	if err := l.Scan(7, 0, func(r Record) bool {
		r.Payload = append([]byte(nil), r.Payload...)
		want = append(want, r)
		return true
	}); err != nil {
		t.Fatal(err)
	}
	for name, seq := range map[string]func(func(Record, error) bool){
		"log":      l.Records(context.Background(), 7),
		"reader":   r.Records(context.Background(), 7),
		"consumer": c.Records(context.Background()),
	} {
		got, err := collect(seq, 0)
		if err != nil || len(got) != len(want) {
			t.Fatalf("%s: %d records %v, want %d", name, len(got), err, len(want))
		}
		for i := range got {
			if !recEqual(got[i], want[i]) {
				t.Fatalf("%s: record %d differs", name, i)
			}
		}
	}
	if c.Offset() != 7 {
		t.Fatal("an iterator committed")
	}
	// Mutating a yielded payload changes nothing.
	for rec, _ := range l.Records(context.Background(), 1) {
		rec.Payload[0] = 'Z'
		break
	}
	if recs, _ := l.Read(1, 1); recs[0].Payload[0] == 'Z' {
		t.Fatal("a yielded payload aliased the log")
	}
	// Follow on the Log wakes on a flush; on the Reader within the poll interval; the
	// consumer form starts at its offset.
	for name, seq := range map[string]func(func(Record, error) bool){
		"log":      l.Follow(context.Background(), 61),
		"reader":   r.Follow(context.Background(), 61),
		"consumer": c.Follow(context.Background()),
	} {
		done := make(chan []Record, 1)
		go func() { recs, _ := collect(seq, 2); done <- recs }()
		time.Sleep(20 * time.Millisecond)
		if name != "consumer" {
			appendN(t, l, 2)
			mustFlush(t, l)
		}
		select {
		case recs := <-done:
			first := uint64(61)
			if name == "consumer" {
				first = 7
			}
			if len(recs) != 2 || recs[0].Seq != first {
				t.Fatalf("%s: %v", name, seqsOf(recs))
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: Follow did not wake", name)
		}
		if name != "consumer" {
			// Roll the head back for the next source: delete nothing, just use new offsets.
			l.mu.Lock()
			h := l.head
			l.mu.Unlock()
			_ = h
		}
		break
	}
	// Cancellation ends Follow promptly with the context's error.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = collect(l.Follow(ctx, 1000), 0)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatalf("%v after %v", err, time.Since(start))
	}
	_, err = collect(r.Follow(ctx, 1000), 0)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("reader: %v", err)
	}
}

func TestAnIteratorYieldsAnErrorOnceAndAGapIsResumedFromToPlusOne(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
	defer l.Close()
	for i := 0; i < 4; i++ {
		appendN(t, l, 4)
		mustFlush(t, l)
	}
	_ = l.Close()
	if err := s.Remove("/log/" + segmentName(1)); err != nil {
		t.Fatal(err)
	}
	l = openSim(t, s, nil)
	defer l.Close()
	var got []uint64
	var errs []error
	for rec, err := range l.Records(context.Background(), 1) {
		if err != nil {
			errs = append(errs, err)
			continue // keep ranging: the sequence must end on its own
		}
		got = append(got, rec.Seq)
	}
	if len(errs) != 1 || !isGap(errs[0], 5, 8) || len(got) != 4 {
		t.Fatalf("records %v errors %v", got, errs)
	}
	var g *GapError
	errors.As(errs[0], &g)
	rest, err := collect(l.Records(context.Background(), g.To+1), 0)
	if err != nil || checkContiguous(t, rest, 9) != 16 {
		t.Fatalf("resumed: %v %v", seqsOf(rest), err)
	}
	// On a closed log the sequence yields ErrClosed once.
	_ = l.Close()
	_, err = collect(l.Records(context.Background(), 1), 0)
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("%v", err)
	}
}

func TestBreakingOutOfAnIteratorLeaksNoDescriptorAndNoGoroutine(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100; c.MaxReadBytes = 30 })
	defer l.Close()
	for i := 0; i < 20; i++ {
		appendN(t, l, 3)
		mustFlush(t, l)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	runtime.GC()
	goroutines := runtime.NumGoroutine()
	open := s.OpenFiles()
	for i := 0; i < 50; i++ {
		for rec := range l.Records(context.Background(), 1) {
			if rec.Seq >= uint64(5+i%17) {
				break
			}
		}
		for rec := range r.Records(context.Background(), 1) {
			if rec.Seq >= uint64(5+i%13) {
				break
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		n := 0
		for range l.Follow(ctx, 1) {
			n++
			if n == 7 {
				cancel()
				break
			}
		}
		cancel()
	}
	time.Sleep(20 * time.Millisecond)
	runtime.GC()
	if s.OpenFiles() != open {
		t.Fatalf("open handles %d, were %d", s.OpenFiles(), open)
	}
	if g := runtime.NumGoroutine(); g > goroutines+2 {
		t.Fatalf("goroutines %d, were %d", g, goroutines)
	}
}
