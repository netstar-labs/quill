package quill

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Consumer.Close releases the consumer's lock and nothing else: it does not wake or end a
// ReadWait, and Read, ReadWait, Scan, Records and Follow ignore it. Only Commit, SeekToTime
// (which commits) and a second Close return ErrClosed. The documentation says so; this pins
// it (decision 41).
func TestConsumerCloseDoesNotEndReadsOnlyCommitsAndSeeks(t *testing.T) {
	l := openDir(t, t.TempDir(), nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	c, err := l.Consumer("w", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Commit(4); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		recs []Record
		err  error
	}
	done := make(chan result, 1)
	go func() {
		recs, err := c.ReadWait(ctx, 10)
		done <- result{recs, err}
	}()
	time.Sleep(100 * time.Millisecond) // let it park at the head

	if err := c.Close(); err != nil {
		t.Fatalf("Close with a ReadWait blocked: %v", err)
	}
	select {
	case r := <-done:
		t.Fatalf("ReadWait returned after Consumer.Close: %d records, %v", len(r.recs), r.err)
	case <-time.After(300 * time.Millisecond):
	}
	cancel()
	select {
	case r := <-done:
		if !errors.Is(r.err, context.Canceled) {
			t.Fatalf("ReadWait after cancel: %v, want context.Canceled", r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ReadWait did not return when ctx was cancelled")
	}

	// Reads on the closed consumer still work, from its offset (4: nothing to read).
	if recs, err := c.Read(10); err != nil || len(recs) != 0 {
		t.Fatalf("Read after Close: %d records, %v", len(recs), err)
	}
	n := 0
	if err := c.Scan(10, func(Record) bool { n++; return true }); err != nil || n != 0 {
		t.Fatalf("Scan after Close: %d records, %v", n, err)
	}
	for _, err := range c.Records(context.Background()) {
		t.Fatalf("Records after Close yielded %v, want nothing", err)
	}
	if got, want := c.Offset(), uint64(4); got != want {
		t.Fatalf("Offset after Close = %d, want %d", got, want)
	}
	// A record appended later is read through the closed handle.
	appendN(t, l, 1)
	mustFlush(t, l)
	if recs, err := c.Read(10); err != nil || len(recs) != 1 || recs[0].Seq != 4 {
		t.Fatalf("Read of a new record after Close: %v, %v", seqsOf(recs), err)
	}
	fctx, fcancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer fcancel()
	var seen []uint64
	var ferr error
	for rec, err := range c.Follow(fctx) {
		if err != nil {
			ferr = err
			break
		}
		seen = append(seen, rec.Seq)
	}
	if len(seen) != 1 || seen[0] != 4 || !errors.Is(ferr, context.DeadlineExceeded) {
		t.Fatalf("Follow after Close: %v, %v", seen, ferr)
	}

	// Commit, SeekToTime and a second Close are the calls that return ErrClosed.
	if err := c.Commit(5); !errors.Is(err, ErrClosed) {
		t.Fatalf("Commit after Close: %v, want ErrClosed", err)
	}
	if err := c.SeekToTime(time.Now()); !errors.Is(err, ErrClosed) {
		t.Fatalf("SeekToTime after Close: %v, want ErrClosed", err)
	}
	if err := c.Close(); !errors.Is(err, ErrClosed) {
		t.Fatalf("second Close: %v, want ErrClosed", err)
	}
}
