package quill

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// countOps counts the operations of kind op on path that the simulator serves from now on.
func countOps(s *fsx.Sim, op fsx.Op, path string) *atomic.Int64 {
	n := new(atomic.Int64)
	s.Inject = func(o fsx.Op, name string) error {
		if o == op && name == path {
			n.Add(1)
		}
		return nil
	}
	return n
}

// outcome is what a Reader makes of Read(off, 3): the sequences it returns, then the gap or
// the error that ended the read.
func outcome(r *Reader, off uint64) string {
	recs, err := r.Read(off, 3)
	out := fmt.Sprint(seqsOf(recs))
	var g *GapError
	switch {
	case errors.As(err, &g):
		out += fmt.Sprintf(" gap[%d,%d]", g.From, g.To)
	case err != nil:
		out += " error: " + err.Error()
	}
	return out
}

// sameAsFresh checks that a Reader that has been reading all along says what a Reader opened
// now says: the same records, gaps and errors at every offset, and then the same statistics.
// The statistics are compared after the reads because a scan teaches a Reader where a segment
// really ends, which the other one has not scanned yet.
func sameAsFresh(t *testing.T, s *fsx.Sim, long *Reader, step string) {
	t.Helper()
	fresh := openReaderSim(t, s)
	defer fresh.Close()
	head, err := fresh.Head()
	if err != nil {
		t.Fatal(err)
	}
	for off := uint64(1); off <= head+1; off++ {
		if got, want := outcome(long, off), outcome(fresh, off); got != want {
			t.Fatalf("%s: Read(%d): long %s; fresh %s", step, off, got, want)
		}
	}
	lst, lerr := long.Stat()
	fst, ferr := fresh.Stat()
	if lerr != nil || ferr != nil || lst.Head != fst.Head || lst.First != fst.First || !slices.Equal(lst.Holes, fst.Holes) || lst.Records != fst.Records {
		t.Fatalf("%s: Stat: long %+v, %v; fresh %+v, %v", step, lst, lerr, fst, ferr)
	}
}

// A view built while nothing has changed does not list the directory: it reads the marker,
// checks that the oldest sealed segment is still there, and uses the references of the view
// before (decision 63, amended for A3-18). A rotation, which moves the marker to a new
// segment, makes the next view list once.
func TestReaderViewDoesNotListTheDirectoryWhileNothingChanged(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 40)
	r := openReaderSim(t, s)
	defer r.Close()
	if recs, err := r.Read(7, 1); err != nil || len(recs) != 1 {
		t.Fatalf("first read: %v, %v", recs, err)
	}
	lists := countOps(s, fsx.OpReadDir, "/log")
	for i := 0; i < 5; i++ {
		if recs, err := r.Read(uint64(7+i), 2); err != nil || len(recs) != 2 {
			t.Fatalf("read %d: %v, %v", i, recs, err)
		}
	}
	if n := lists.Load(); n != 0 {
		t.Fatalf("five reads of an unchanged log listed the directory %d times; want 0", n)
	}
	fillSegments(t, l, clk, 1) // a rotation: the marker names a new segment
	for i := 0; i < 3; i++ {
		if _, err := r.Read(7, 1); err != nil {
			t.Fatal(err)
		}
	}
	if n := lists.Load(); n != 1 {
		t.Fatalf("three reads after a rotation listed the directory %d times; want 1", n)
	}
}

// Through everything a live log does to its sealed segments (rotations, retention taking the
// oldest, the active segment growing), a Reader that keeps its references says what a new
// Reader says.
func TestReaderThatKeepsItsSealedReferencesReadsLikeAFreshOne(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, nil) // Retention one hour; a sweep takes what is older
	defer l.Close()
	long := openReaderSim(t, s)
	defer long.Close()
	fillSegments(t, l, clk, 12)
	sameAsFresh(t, s, long, "twelve segments")
	appendN(t, l, 2)
	mustFlush(t, l)
	sameAsFresh(t, s, long, "records in the active segment")
	fillSegments(t, l, clk, 3)
	sameAsFresh(t, s, long, "three more segments")
	// Retention takes the oldest segments: the probe finds the oldest one gone.
	clk.Advance(time.Hour - 7*time.Minute)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	sameAsFresh(t, s, long, "after an age sweep")
	fillSegments(t, l, clk, 2)
	sameAsFresh(t, s, long, "rotations after the sweep")
	clk.Advance(2 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	sameAsFresh(t, s, long, "after a sweep that took every sealed segment")
	fillSegments(t, l, clk, 2)
	sameAsFresh(t, s, long, "segments again")
}

// A scan of a sealed segment that finds it ends before the next one starts changes what a view
// says of it (a hole the first frames could not show), so a plan built before the scan is not
// used after it: the Reader's statistics show the hole, as a Reader that scanned it does.
func TestReaderPlanBuiltBeforeAScanDoesNotHideWhatTheScanFound(t *testing.T) {
	s, victim := unwarmedVanishingLog(t)
	data := fileBytes(t, s, victim)
	ends := frameEnds(t, data)
	writeFile(t, s, victim, data[:ends[0]]) // record 5 kept, 6 to 8 gone, cut at a frame boundary
	long := openReaderSim(t, s)
	defer long.Close()
	if recs, err := long.Read(1, 1); err != nil || len(recs) != 1 { // builds the plan; the victim is not scanned
		t.Fatalf("Read(1, 1): %v, %v", recs, err)
	}
	if recs, err := long.Read(5, 1); err != nil || len(recs) != 1 || recs[0].Seq != 5 { // scans the victim
		t.Fatalf("Read(5, 1): %v, %v", recs, err)
	}
	st, err := long.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.Holes, []Hole{{From: 6, To: 8}}) {
		t.Fatalf("Stat after the scan: holes %v; want [{6 8}]", st.Holes)
	}
}

// A segment removed out of order, which retention never does, is met when it is opened, as a
// gap over its range and not a path error (decision 101); the Reader then lists again and
// agrees with a new Reader.
func TestReaderPlanMeetsASegmentRemovedOutOfOrderAsAGapAndThenListsAgain(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 6)
	appendN(t, l, 1)
	mustFlush(t, l)
	long := openReaderSim(t, s)
	defer long.Close()
	if recs, err := long.Read(1, 1); err != nil || len(recs) != 1 {
		t.Fatalf("warm read: %v, %v", recs, err)
	}
	if recs, err := long.Read(13, 1); err != nil || len(recs) != 1 { // a view with the middle segment's first frame already known
		t.Fatalf("warm read: %v, %v", recs, err)
	}
	ids := segmentFiles(t, s, "/log")
	victim := "/log/" + segmentName(ids[2]) // records 9 to 12
	if err := s.Remove(victim); err != nil {
		t.Fatal(err)
	}
	if _, err := long.Read(9, 1); !isGap(err, 9, 12) {
		t.Fatalf("Read(9, 1) with the segment gone: %v; want the gap 9..12", err)
	}
	sameAsFresh(t, s, long, "after the out-of-order removal")
}

// A segment that could not be opened for an I/O reason is a hole in the view that met it and is
// looked at again by the next one (decision 94), so that view's references are not kept.
func TestReaderPlanIsNotKeptWhenASegmentCouldNotBeOpened(t *testing.T) {
	s, victim := unwarmedVanishingLog(t)
	var fail atomic.Bool
	fail.Store(true)
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpOpen && name == victim && fail.Load() {
			return errors.New("injected: too many open files")
		}
		return nil
	}
	r := openReaderSim(t, s)
	defer r.Close()
	// A read of the segment after the unopenable one: its scan finds nothing unexpected, so the
	// scan does not by itself end the view's life, and only the plan's own rule can.
	if recs, err := r.Read(9, 1); err != nil || len(recs) != 1 {
		t.Fatalf("Read(9, 1) beside the unopenable segment: %v, %v", recs, err)
	}
	fail.Store(false)
	if recs, err := r.Read(5, 1); err != nil || len(recs) != 1 || recs[0].Seq != 5 {
		t.Fatalf("Read(5, 1) after the fault cleared: %v, %v; want record 5", seqsOf(recs), err)
	}
}

// A plan is for one active segment: when the first sequence the marker's segment now starts at
// is not the one the plan was built for (a writer crash renumbered the segment, decision 63),
// the view lists the directory and builds its own.
func TestReaderPlanIsNotUsedForAnotherActiveFirstSequence(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 5)
	appendN(t, l, 1)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()
	if _, err := r.Read(3, 1); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	if r.plan == nil || r.plan.activeFirst == 0 {
		r.mu.Unlock()
		t.Fatalf("no plan for an active segment with frames: %+v", r.plan)
	}
	r.plan.activeFirst++ // the plan now says the active segment started elsewhere
	r.mu.Unlock()
	lists := countOps(s, fsx.OpReadDir, "/log")
	if _, err := r.Read(3, 1); err != nil {
		t.Fatal(err)
	}
	if n := lists.Load(); n != 1 {
		t.Fatalf("a view for another first sequence listed the directory %d times; want 1", n)
	}
}

// The references a Reader keeps are shared by every view built from them, and the reads that
// learn a segment ends early (a hole) must not write them: many reads across such a segment at
// once, under -race, with the references kept between calls.
func TestReaderReadsAcrossASegmentThatEndsEarlyShareReferencesWithoutWritingThem(t *testing.T) {
	s, victim := unwarmedVanishingLog(t)
	data := fileBytes(t, s, victim)
	ends := frameEnds(t, data)
	writeFile(t, s, victim, data[:ends[0]]) // record 5 kept, 6 to 8 gone
	r := openReaderSim(t, s)
	defer r.Close()
	if recs, err := r.Read(1, 1); err != nil || len(recs) != 1 {
		t.Fatalf("Read(1, 1): %v, %v", recs, err)
	}
	var wg sync.WaitGroup
	errc := make(chan error, 8)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if recs, err := r.Read(5, 1); err != nil || len(recs) != 1 || recs[0].Seq != 5 {
					errc <- fmt.Errorf("Read(5, 1): %v, %v", seqsOf(recs), err)
					return
				}
				if _, err := r.Read(6, 1); !isGap(err, 6, 8) {
					errc <- fmt.Errorf("Read(6, 1): %v; want the gap 6..8", err)
					return
				}
				if recs, err := r.Read(9, 1); err != nil || len(recs) != 1 || recs[0].Seq != 9 {
					errc <- fmt.Errorf("Read(9, 1): %v, %v", seqsOf(recs), err)
					return
				}
			}
		}()
	}
	wg.Wait()
	select {
	case err := <-errc:
		t.Fatal(err)
	default:
	}
}
