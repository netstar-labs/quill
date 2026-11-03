package quill

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// patience bounds a wait that a correct log satisfies at once. It is a deadlock detector, not an
// assertion about speed: no test below compares an elapsed time with anything, so a loaded or
// slow machine cannot fail them, and a log that really blocks fails them by never getting there.
const patience = 30 * time.Second

// gate makes an injected operation stand still where the test can see it: hold marks that the
// operation has been entered, then waits until open, so a test learns that the flush is inside
// its fsync from the flush itself, not from a sleep that a loaded machine can outrun.
type gate struct {
	entered, release    chan struct{}
	enterOnce, openOnce sync.Once
}

func newGate() *gate { return &gate{entered: make(chan struct{}), release: make(chan struct{})} }

func (g *gate) hold() {
	g.enterOnce.Do(func() { close(g.entered) })
	<-g.release
}

func (g *gate) open() { g.openOnce.Do(func() { close(g.release) }) }

func (g *gate) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-g.entered:
	case <-time.After(patience):
		t.Fatal("the injected operation was never entered")
	}
}

// completes runs f and fails the test if it errors or does not return within patience.
func completes(t *testing.T, what string, f func() error) {
	t.Helper()
	errc := make(chan error, 1)
	go func() { errc <- f() }()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	case <-time.After(patience):
		t.Fatalf("%s did not complete", what)
	}
}

// T18: Sync.

func TestManyProducersSyncingConcurrentlyShareFsyncs(t *testing.T) {
	producersShareFsyncs(t)
}

// One CPU is a supported deployment (a one-CPU container limit sets GOMAXPROCS to 1), and the gate
// runs at the default only, so the one-proc case is pinned here.
func TestManyProducersSyncingConcurrentlyShareFsyncsOnOneProc(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	producersShareFsyncs(t)
}

func producersShareFsyncs(t *testing.T) {
	t.Helper()
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
	g := newGate()
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			g.hold()
			return errENOSPC
		}
		return nil
	}
	errc := make(chan error, 1)
	go func() { errc <- l.Sync(context.Background(), seq) }()
	g.waitEntered(t) // the flush is inside its write, so the waiter is parked on it
	// Release the write shortly after Close has started. Which comes first changes only which
	// of the two errors the waiter gets, and the check below accepts both.
	go func() { time.Sleep(20 * time.Millisecond); g.open() }()
	_ = l.Close()
	err := <-errc
	if !errors.Is(err, ErrClosed) && !errors.Is(err, errENOSPC) {
		t.Fatalf("pending waiter: %v", err)
	}
	// A waiter whose sequence became durable at Close gets nil.
	s = fsx.NewSim()
	l = openSim(t, s, nil)
	seq, _ = l.Append([]byte("x"))
	inWrite := make(chan struct{})
	var once sync.Once
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			once.Do(func() { close(inWrite) })
			time.Sleep(30 * time.Millisecond)
		}
		return nil
	}
	go func() { errc <- l.Sync(context.Background(), seq) }()
	select { // the flush the waiter asked for is under way; Close must not lose that waiter
	case <-inWrite:
	case <-time.After(patience):
		t.Fatal("the flush never reached its write")
	}
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

// The goroutine parked in Sync holds nothing: while the flush it asked for is inside its fsync,
// an append can be made, and both finish once the fsync does. (The flush itself holds the log's
// lock across the write and the fsync, so an append made meanwhile waits for it; that wait is
// not asserted, only that nothing waits for the parked Sync.)
func TestAProducerWaitingInSyncHoldsNoLock(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	g := newGate()
	defer g.open() // runs before l.Close above, so a failed test does not hang behind the fsync
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpSync && strings.HasSuffix(name, ".log") {
			g.hold()
		}
		return nil
	}
	seq, _ := l.Append([]byte("a"))
	syncDone := make(chan error, 1)
	go func() { syncDone <- l.Sync(context.Background(), seq) }()
	g.waitEntered(t)
	appendDone := make(chan error, 1)
	go func() { _, err := l.Append([]byte("b")); appendDone <- err }()
	g.open()
	for _, c := range []struct {
		what string
		ch   chan error
	}{{"the Sync", syncDone}, {"the append made during the fsync", appendDone}} {
		select {
		case err := <-c.ch:
			if err != nil {
				t.Fatalf("%s: %v", c.what, err)
			}
		case <-time.After(patience):
			t.Fatalf("%s did not complete after the fsync was released", c.what)
		}
	}
	// With the waiter gone nothing is held: an append completes.
	s.Inject = nil
	completes(t, "an append after the flush", func() error { _, err := l.Append([]byte("c")); return err })
}

// T5: isolation between logs in one process.

// The other log's appends and flushes finish while the first log's fsync is still held, which
// is an ordering, not a duration: they cannot have waited for it.
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
	g := newGate()
	defer g.open() // runs before the Closes above, so a failed test does not hang behind the fsync
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpSync && strings.HasPrefix(name, "/log/") {
			g.hold()
		}
		return nil
	}
	a.Append([]byte("slow"))
	done := make(chan struct{})
	go func() { defer close(done); _ = a.Flush() }()
	g.waitEntered(t) // the first log's flush is inside its fsync, and stays there
	completes(t, "the other log's appends and flushes during the slow fsync", func() error {
		for i := 0; i < 20; i++ {
			if _, err := b.Append([]byte("fast")); err != nil {
				return err
			}
			if err := b.Flush(); err != nil {
				return err
			}
		}
		return nil
	})
	g.open()
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
