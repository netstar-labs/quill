package quill

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A log that failed (ErrFailed) can never produce another record, so a goroutine blocked in
// ReadWait, Consumer.ReadWait or Follow must be told, not left to run until its context ends
// (audit A5-3). Reads of records that already exist stay valid.

// parkReadWait starts a ReadWait above the head and returns once it is blocked on the flush
// broadcast, with the error it returns on the channel. The context is long: a test that is
// not woken fails on the receive timeout in waitErr, not on the context.
func parkReadWait(t *testing.T, l *Log) (<-chan error, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan error, 1)
	go func() {
		_, err := l.ReadWait(ctx, l.Head()+1, 10)
		out <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		l.mu.Lock()
		parked := l.flushed != nil
		l.mu.Unlock()
		if parked {
			return out, cancel
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("ReadWait never blocked")
		}
		time.Sleep(time.Millisecond)
	}
}

func waitErr(t *testing.T, out <-chan error) error {
	t.Helper()
	select {
	case err := <-out:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("the waiter was not woken by the failure")
		return nil
	}
}

func failOps(s *fsx.Sim, op fsx.Op, armed *atomic.Bool) {
	s.Inject = func(o fsx.Op, name string) error {
		if o == op && armed.Load() {
			return errors.New("injected EIO")
		}
		return nil
	}
}

func TestReadWaitOnAFailedLogReturnsErrFailed(t *testing.T) {
	s := fsx.NewSim()
	var armed atomic.Bool
	failOps(s, fsx.OpSync, &armed)
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	c, err := l.Consumer("c", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	armed.Store(true)
	appendN(t, l, 1)
	if err := l.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Flush: %v, want ErrFailed", err)
	}
	armed.Store(false) // the log stays failed; the consumer's own file writes must work
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()

	// Records that exist stay readable, wait or not.
	recs, err := l.ReadWait(ctx, 1, 10)
	if err != nil || len(recs) != 3 {
		t.Fatalf("ReadWait of existing records on a failed log: %d records, %v; want 3, nil", len(recs), err)
	}
	if recs, err = c.ReadWait(ctx, 10); err != nil || len(recs) != 3 {
		t.Fatalf("Consumer.ReadWait of existing records: %d records, %v; want 3, nil", len(recs), err)
	}

	// Nothing more can arrive: the wait fails with the log's failure.
	if _, err := l.ReadWait(ctx, l.Head()+1, 10); !errors.Is(err, ErrFailed) {
		t.Fatalf("ReadWait at the head of a failed log: %v, want ErrFailed", err)
	}
	if err := c.Commit(l.Head() + 1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadWait(ctx, 10); !errors.Is(err, ErrFailed) {
		t.Fatalf("Consumer.ReadWait at the head of a failed log: %v, want ErrFailed", err)
	}
	var seen int
	var last error
	for _, err := range l.Follow(ctx, 1) {
		if err != nil {
			last = err
			break
		}
		seen++
	}
	if seen != 3 || !errors.Is(last, ErrFailed) {
		t.Fatalf("Follow on a failed log: %d records then %v; want 3 then ErrFailed", seen, last)
	}
	last = nil
	for _, err := range c.Follow(ctx) {
		if err != nil {
			last = err
			break
		}
	}
	if !errors.Is(last, ErrFailed) {
		t.Fatalf("Consumer.Follow at the head of a failed log: %v, want ErrFailed", last)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("the failures took %v: they waited for the context", d)
	}
}

// A waiter already parked when the log fails is woken with the failure, by every path that
// fails the log, including the ones that do not flush: a directory fsync that fails in Roll and
// in a retention pass.
func TestAParkedWaiterIsWokenWhenTheLogFails(t *testing.T) {
	t.Run("flush fsync", func(t *testing.T) {
		s := fsx.NewSim()
		var armed atomic.Bool
		failOps(s, fsx.OpSync, &armed)
		l := openSim(t, s, nil)
		defer l.Close()
		appendN(t, l, 2)
		mustFlush(t, l)
		out, cancel := parkReadWait(t, l)
		defer cancel()
		armed.Store(true)
		appendN(t, l, 1)
		if err := l.Flush(); !errors.Is(err, ErrFailed) {
			t.Fatalf("Flush: %v, want ErrFailed", err)
		}
		if err := waitErr(t, out); !errors.Is(err, ErrFailed) {
			t.Fatalf("parked ReadWait: %v, want ErrFailed", err)
		}
	})
	t.Run("roll directory fsync", func(t *testing.T) {
		s := fsx.NewSim()
		var armed atomic.Bool
		failOps(s, fsx.OpSyncDir, &armed)
		l := openSim(t, s, nil)
		defer l.Close()
		appendN(t, l, 2)
		mustFlush(t, l)
		out, cancel := parkReadWait(t, l)
		defer cancel()
		armed.Store(true)
		if err := l.Roll(); !errors.Is(err, ErrFailed) {
			t.Fatalf("Roll: %v, want ErrFailed", err)
		}
		if err := waitErr(t, out); !errors.Is(err, ErrFailed) {
			t.Fatalf("parked ReadWait: %v, want ErrFailed", err)
		}
	})
	t.Run("retention directory fsync", func(t *testing.T) {
		s, clk, l := retentionLog(t, nil)
		defer l.Close()
		fillSegments(t, l, clk, 6)
		clk.Advance(2 * time.Hour)
		out, cancel := parkReadWait(t, l)
		defer cancel()
		var armed atomic.Bool
		failOps(s, fsx.OpSyncDir, &armed)
		armed.Store(true)
		if err := l.Prune(true); !errors.Is(err, ErrFailed) {
			t.Fatalf("Prune: %v, want ErrFailed", err)
		}
		if err := waitErr(t, out); !errors.Is(err, ErrFailed) {
			t.Fatalf("parked ReadWait: %v, want ErrFailed", err)
		}
	})
}
