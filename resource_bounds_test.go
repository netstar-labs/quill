package quill

import (
	"context"
	"iter"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Resource bounds: peak descriptors per in-flight read, descriptors under parked Follows,
// goroutines after Close, and the MaxRecords cap under random operations (decisions 33 and 58).

// peakOpen makes s record the highest descriptor count seen at any file operation since the
// last reset, and returns functions that read the peak and reset it to zero. It samples
// through the Inject hook, so it adds no seam to the simulator; the hook runs without the
// simulator's lock, so OpenFiles may be called from it. Install it while no goroutine uses the
// simulator and remove it (s.Inject = nil) after they are done.
func peakOpen(s *fsx.Sim) (peak func() int, reset func()) {
	var hi atomic.Int64
	s.Inject = func(fsx.Op, string) error {
		n := int64(s.OpenFiles())
		for {
			p := hi.Load()
			if n <= p || hi.CompareAndSwap(p, n) {
				return nil
			}
		}
	}
	return func() int { return int(hi.Load()) }, func() { hi.Store(0) }
}

// A read opens, reads and closes one sealed segment at a time (decision 15): G concurrent calls
// hold a bounded number of descriptors however many segments there are. A regression that
// kept a descriptor across a yield or until the end of the call would hold about G times the
// segment count. The callback yields the processor so the calls overlap.
func TestConcurrentReadsHoldABoundedNumberOfDescriptors(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	defer l.Close()
	const segments, readers = 60, 8
	for i := 0; i < segments; i++ {
		appendN(t, l, 3)
		mustFlush(t, l)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	if st, _ := l.Stat(); st.Segments < segments*2/3 {
		t.Fatalf("only %d segments", st.Segments)
	}
	base := s.OpenFiles()
	peak, _ := peakOpen(s)
	var wg sync.WaitGroup
	errs := make(chan error, 4*readers)
	for g := 0; g < readers; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, src := range []interface {
				ScanAll(func(Record) bool) error
			}{l, r} {
				n := 0
				if err := src.ScanAll(func(Record) bool { n++; runtime.Gosched(); return true }); err != nil {
					errs <- err
				}
				if n < 3*segments {
					t.Errorf("ScanAll saw %d records", n)
				}
			}
			for _, recs := range []recsFn{l.Records, r.Records} {
				n := 0
				for _, err := range recs(context.Background(), 1) {
					if err != nil {
						errs <- err
						break
					}
					n++
					runtime.Gosched()
				}
				if n < 3*segments {
					t.Errorf("Records saw %d records", n)
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	s.Inject = nil
	for err := range errs {
		t.Fatal(err)
	}
	// The active segment's descriptor is the writer's; a read adds one for the segment it is
	// in (two at a hand-over). Twice the concurrent calls is the ceiling; a leak is segments times that.
	if extra := peak() - base; extra > 2*readers {
		t.Fatalf("peak %d descriptors over a base of %d with %d concurrent reads: %d extra, want at most %d", peak(), base, readers, extra, 2*readers)
	}
	if n := s.OpenFiles(); n != base {
		t.Fatalf("%d descriptors open after the reads, were %d", n, base)
	}
}

// Parked Follows hold no descriptor between polls: Log follows hold none at all, Reader
// follows add a transient one each (a tail poll) and the count is back at the base between
// polls; cancelling them returns it to where it was.
func TestParkedFollowsHoldNoDescriptorsAndReleaseOnCancel(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	defer l.Close()
	for i := 0; i < 10; i++ {
		appendN(t, l, 3)
		mustFlush(t, l)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	head := uint64(30)
	base := s.OpenFiles()

	const follows = 20
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	peak, reset := peakOpen(s)
	caught := make(chan struct{}, follows)
	var wg sync.WaitGroup
	park := func(follow recsFn) {
		for i := 0; i < follows; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for rec := range follow(ctx, 1) {
					if rec.Seq == head {
						caught <- struct{}{}
					}
				}
			}()
		}
		for i := 0; i < follows; i++ {
			select {
			case <-caught:
			case <-time.After(10 * time.Second):
				t.Fatalf("only %d of %d follows reached the end of the log", i, follows)
			}
		}
	}
	// A Log's follows wait on the writer, not on the disk: parked, they hold nothing.
	park(l.Follow)
	time.Sleep(50 * time.Millisecond)
	if n := s.OpenFiles(); n != base {
		t.Fatalf("%d descriptors open with %d Log follows parked, were %d", n, follows, base)
	}
	// A Reader's follows poll quill.tail every PollInterval: each holds one descriptor for the
	// length of a poll and none between polls. So the count may rise by up to one per follow at
	// an instant, but it must also fall back to the base: a follow that kept a descriptor
	// across the wait would hold the count at base+1 or more for as long as it is parked. Wait
	// for a sample at the base over many polls; the deadline is generous because a hit is
	// expected at once.
	park(r.Follow)
	reset()
	deadline := time.Now().Add(10 * time.Second)
	lowest := s.OpenFiles()
	for lowest > base && time.Now().Before(deadline) {
		time.Sleep(100 * time.Microsecond)
		lowest = min(lowest, s.OpenFiles())
	}
	if lowest != base {
		t.Fatalf("%d Reader follows parked for 10s: the descriptor count never fell to the base %d (lowest %d), so they hold descriptors between polls", follows, base, lowest)
	}
	if extra := peak() - base; extra > follows {
		t.Fatalf("%d Reader follows parked: peak %d descriptors over a base of %d", follows, peak(), base)
	}
	cancel()
	wg.Wait()
	s.Inject = nil
	if n := s.OpenFiles(); n != base {
		t.Fatalf("%d descriptors open after the follows were cancelled, were %d", n, base)
	}
}

// Close joins the flusher and leaves no goroutine behind: after Reader use, parked and
// cancelled Follows and a Consumer, the count is what it was before Open.
func TestCloseLeavesNoGoroutineBehind(t *testing.T) {
	runtime.GC()
	before := runtime.NumGoroutine()
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100; c.FlushInterval = time.Millisecond })
	appendN(t, l, 30)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	if _, err := r.Read(1, 10); err != nil {
		t.Fatal(err)
	}
	c, err := l.Consumer("c", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < 20; i++ {
		for _, follow := range []recsFn{l.Follow, r.Follow, func(ctx context.Context, _ uint64) iter.Seq2[Record, error] { return c.Follow(ctx) }} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range follow(ctx, 1) {
				}
			}()
		}
	}
	time.Sleep(20 * time.Millisecond)
	cancel()
	wg.Wait()
	if err := c.Commit(5); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	select { // Close joined the flusher: its done channel is closed when Close returns
	case <-l.done:
	default:
		t.Fatal("Close returned while the flusher was still running")
	}
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if g := runtime.NumGoroutine(); g > before {
		buf := make([]byte, 1<<16)
		t.Fatalf("%d goroutines after Close, were %d before Open:\n%s", g, before, buf[:runtime.Stack(buf, true)])
	}
	if n := s.OpenFiles(); n != 0 {
		t.Fatalf("%d descriptors open after Close", n)
	}
}

// recsFn is the shape of Records and Follow on a Log and on a Reader.
type recsFn = func(context.Context, uint64) iter.Seq2[Record, error]

// MaxRecords is a cap on the retained records, segment-granular (decision 33). After a roll
// or a prune the log holds at most MaxRecords records unless what is left is the active
// segment alone (it is never deleted); after a flush, which runs the caps only if it seals,
// at most MaxRecords plus the frames of the active segment. A delete leaves more than
// MaxRecords minus the frames of the last segment deleted (the cap does not over-delete).
// All hold under random record sizes, a protected consumer that lags (the cap overrides
// protection, decision 57) and random flush, roll and prune calls.
func TestMaxRecordsBoundHoldsUnderRandomOperations(t *testing.T) {
	for seed := uint64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewPCG(seed, 58))
		maxRecords := int64(3 + rng.IntN(28))
		s := fsx.NewSim()
		clk := newClock()
		l := openRetention(t, s, clk, func(c *Config) { c.SegmentBytes = 60; c.MaxRecords = maxRecords })
		cons, err := l.Consumer("slow", ConsumerOptions{Protect: ProtectOn})
		if err != nil {
			t.Fatal(err)
		}
		// snap is the first retained sequence and the sealed segments' ranges, taken before an
		// operation so that check can tell which segment it deleted last.
		type span struct{ first, last uint64 }
		snap := func() (uint64, []span) {
			l.mu.Lock()
			defer l.mu.Unlock()
			var sealed []span
			for _, sg := range l.sealed {
				sealed = append(sealed, span{sg.first, sg.last})
			}
			return l.firstRetainedLocked(), sealed
		}
		check := func(step int, op string, prevFirst uint64, prev []span) {
			t.Helper()
			st, err := l.Stat()
			if err != nil {
				t.Fatal(err)
			}
			l.mu.Lock()
			active := l.active.frames
			l.mu.Unlock()
			// The caps run at a seal and at a prune; frames added to the active segment since
			// are not yet counted against them.
			bound := maxRecords + int64(active)
			if op != "flush" {
				bound = max(maxRecords, int64(active))
			}
			if st.Records > uint64(bound) {
				t.Fatalf("seed %d step %d after %s: %d records retained, MaxRecords %d, %d in the active segment: %+v", seed, step, op, st.Records, maxRecords, active, st)
			}
			// When the segment just before the new First was one of those listed before the
			// operation, it was the last one deleted, and it was deleted because what was left
			// without it was still over the cap: the log is not left holding fewer than the
			// cap minus its frames.
			for i := 0; i+1 < len(prev) && st.First > prevFirst; i++ {
				if prev[i+1].first == st.First {
					if k := prev[i].last - prev[i].first + 1; st.Records+k <= uint64(maxRecords) {
						t.Fatalf("seed %d step %d after %s: %d records retained, MaxRecords %d, but the segment deleted last held %d: %+v", seed, step, op, st.Records, maxRecords, k, st)
					}
					break
				}
			}
		}
		for step := 0; step < 300; step++ {
			prevFirst, prev := snap()
			var op string
			switch k := rng.IntN(10); {
			case k < 6:
				op = "append"
				if _, err := l.Append(make([]byte, 1+rng.IntN(40))); err != nil {
					t.Fatal(err)
				}
				continue
			case k < 8:
				op = "flush"
				mustFlush(t, l)
			case k < 9:
				op = "roll"
				if err := l.Roll(); err != nil {
					t.Fatal(err)
				}
			default:
				op = "prune"
				mustFlush(t, l)
				if err := l.Prune(false); err != nil {
					t.Fatal(err)
				}
			}
			if rng.IntN(4) == 0 {
				st, _ := l.Stat()
				at := st.First + uint64(rng.Int64N(int64(st.Head-st.First)+2)) // First..Head+1
				if err := cons.Commit(at); err != nil {
					t.Fatalf("seed %d: commit: %v", seed, err)
				}
			}
			clk.Advance(time.Second)
			check(step, op, prevFirst, prev)
		}
		if err := cons.Close(); err != nil {
			t.Fatal(err)
		}
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
