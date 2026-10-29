package quill

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// T18: Sync.

func TestManyProducersSyncingConcurrentlyShareFsyncs(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.FlushInterval = time.Hour })
	defer l.Close()
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpSync {
			time.Sleep(2 * time.Millisecond)
		}
		return nil
	}
	const producers, each = 16, 10
	var wg sync.WaitGroup
	for p := 0; p < producers; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				seq, err := l.Append([]byte("x"))
				if err != nil {
					t.Error(err)
					return
				}
				if err := l.Sync(context.Background(), seq); err != nil {
					t.Error(err)
					return
				}
				if l.Head() < seq {
					t.Errorf("sync returned with head %d below %d", l.Head(), seq)
				}
			}
		}()
	}
	wg.Wait()
	l.mu.Lock()
	fsyncs := l.fsyncs
	l.mu.Unlock()
	if fsyncs >= producers*each {
		t.Fatalf("%d fsyncs for %d syncs", fsyncs, producers*each)
	}
	if fsyncs == 0 {
		t.Fatal("no fsyncs")
	}
}

func TestALoneSyncReturnsWellBeforeFlushIntervalAndErrBadSeqForAnUnassignedSequence(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.FlushInterval = 10 * time.Second })
	defer l.Close()
	seq, _ := l.Append([]byte("x"))
	start := time.Now()
	if err := l.Sync(context.Background(), seq); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("sync took %v", d)
	}
	if err := l.Sync(context.Background(), seq+1); !errors.Is(err, ErrBadSeq) {
		t.Fatalf("unassigned: %v", err)
	}
	if err := l.Sync(context.Background(), 0); err != nil {
		t.Fatalf("seq 0 is trivially durable: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	l.Append([]byte("y"))
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			time.Sleep(50 * time.Millisecond)
		}
		return nil
	}
	if err := l.Sync(ctx, seq+1); !errors.Is(err, context.Canceled) && err != nil {
		t.Fatalf("cancelled context: %v", err)
	}
}

func TestPendingSyncWaitersGetErrClosedAtCloseUnlessDurable(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	seq, _ := l.Append([]byte("x"))
	// Block the flush so the waiter is pending at Close.
	block := make(chan struct{})
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			<-block
			return errENOSPC
		}
		return nil
	}
	errc := make(chan error, 1)
	go func() { errc <- l.Sync(context.Background(), seq) }()
	time.Sleep(20 * time.Millisecond)
	go func() { time.Sleep(20 * time.Millisecond); close(block) }()
	_ = l.Close()
	err := <-errc
	if !errors.Is(err, ErrClosed) && !errors.Is(err, errENOSPC) {
		t.Fatalf("pending waiter: %v", err)
	}
	// A waiter whose sequence became durable at Close gets nil.
	s = fsx.NewSim()
	l = openSim(t, s, nil)
	seq, _ = l.Append([]byte("x"))
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			time.Sleep(30 * time.Millisecond)
		}
		return nil
	}
	go func() { errc <- l.Sync(context.Background(), seq) }()
	time.Sleep(5 * time.Millisecond)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("waiter made durable by close: %v", err)
	}
}

func TestWithAFlushFailingSyncAndASyncAppendReturnTheErrorMeaningUnknown(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SyncAppends = true })
	defer l.Close()
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			return errENOSPC
		}
		return nil
	}
	seq, err := l.Append([]byte("x"))
	if !errors.Is(err, errENOSPC) || seq != 1 {
		t.Fatalf("sync append: %d %v", seq, err)
	}
	if err := l.Sync(context.Background(), 1); !errors.Is(err, errENOSPC) {
		t.Fatalf("sync: %v", err)
	}
	// The record is still buffered and becomes durable once writes work: unknown, not lost.
	s.Inject = nil
	if err := l.Sync(context.Background(), 1); err != nil {
		t.Fatalf("sync after recovery: %v", err)
	}
	if recs, _ := l.Read(1, 1); len(recs) != 1 || recs[0].Seq != 1 {
		t.Fatalf("%v", seqsOf(recs))
	}
	seq, err = l.Append([]byte("y"))
	if err != nil || seq != 2 || l.Head() != 2 {
		t.Fatalf("sync append: %d %v head %d", seq, err, l.Head())
	}
}

func TestAProducerWaitingInSyncHoldsNoLock(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	const slow = 150 * time.Millisecond
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpSync && strings.HasSuffix(name, ".log") {
			time.Sleep(slow)
		}
		return nil
	}
	seq, _ := l.Append([]byte("a"))
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := l.Sync(context.Background(), seq); err != nil {
			t.Error(err)
		}
	}()
	time.Sleep(20 * time.Millisecond) // the flush is now inside its slow fsync
	start := time.Now()
	if _, err := l.Append([]byte("b")); err != nil {
		t.Fatal(err)
	}
	blocked := time.Since(start)
	<-done
	if blocked > slow+50*time.Millisecond {
		t.Fatalf("an append was blocked for %v, longer than the flush's fsync", blocked)
	}
	// After the flush, with the waiter gone, nothing is held: an append is immediate.
	s.Inject = nil
	start = time.Now()
	if _, err := l.Append([]byte("c")); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 20*time.Millisecond {
		t.Fatalf("append took %v", d)
	}
}

// T5: isolation between logs in one process.

func TestASlowFsyncOnOneLogNeverDelaysAnother(t *testing.T) {
	s := fsx.NewSim()
	a := openSim(t, s, nil)
	defer a.Close()
	cfg := simConfig(s, "/other")
	b, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpSync && len(name) > 5 && name[:5] == "/log/" {
			time.Sleep(300 * time.Millisecond)
		}
		return nil
	}
	a.Append([]byte("slow"))
	done := make(chan struct{})
	go func() { defer close(done); _ = a.Flush() }()
	time.Sleep(20 * time.Millisecond)
	start := time.Now()
	for i := 0; i < 20; i++ {
		if _, err := b.Append([]byte("fast")); err != nil {
			t.Fatal(err)
		}
		if err := b.Flush(); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Fatalf("the other log took %v while one was in a slow fsync", d)
	}
	<-done
}

// Sync of an unassigned sequence reads the last assigned number under the log's lock: with
// appends racing it, the race detector must stay quiet.
func TestSyncBadSeqReadsLastAssignedUnderLock(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.FlushInterval = time.Hour })
	defer l.Close()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if _, err := l.Append([]byte("x")); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 2000; i++ {
		if err := l.Sync(context.Background(), 1<<40); !errors.Is(err, ErrBadSeq) {
			t.Errorf("Sync(1<<40) = %v, want ErrBadSeq", err)
			break
		}
	}
	close(stop)
	wg.Wait()
}
