package quill

import (
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"
)

// MinFreeBytes with a failing or full statfs is visible (Stage 2 audit A5-22): Err() reports
// both states, Stat() shows DiskFull and the free bytes of the last pass, and recovery is
// logged and clears them.

// statfsSeam is an injectable free-space function: failing makes it return statErr.
type statfsSeam struct {
	failing bool
	free    uint64
	err     error
}

func (f *statfsSeam) statfs(string) (uint64, error) {
	if f.failing {
		return 0, f.err
	}
	return f.free, nil
}

func TestFailingStatfsIsReportedByErrAndClearsWhenItRecovers(t *testing.T) {
	sf := &statfsSeam{free: 100_000, err: errors.New("statfs: input/output error")}
	_, clk, l := retentionLog(t, func(c *Config) {
		c.MinFreeBytes = 5000
		c.statfs = sf.statfs
	})
	defer l.Close()
	fillSegments(t, l, clk, 3)
	wantErrNil(t, l, "before statfs failed")

	sf.failing = true
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if err := l.Err(); !errors.Is(err, sf.err) {
		t.Fatalf("Err() with statfs failing: %v, want it to wrap the statfs error", err)
	}
	// One failure is the trigger: there is no count of consecutive failures.
	sf.failing = false
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	wantErrNil(t, l, "after statfs recovered")
}

func TestDiskFullIsInErrAndStatAndItsClearingIsLogged(t *testing.T) {
	sf := &statfsSeam{free: 100_000, err: errors.New("statfs: input/output error")}
	var lc logCapture
	_, clk, l := retentionLog(t, func(c *Config) {
		c.MinFreeBytes = 5000
		c.Logger = lc.logger()
		c.statfs = sf.statfs
	})
	defer l.Close()
	fillSegments(t, l, clk, 3)
	if st, _ := l.Stat(); st.DiskFull {
		t.Fatalf("DiskFull before the floor was breached: %+v", st)
	}

	sf.free = 100 // hopeless: everything sealed goes and appends fail
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Append([]byte("x")); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("append on a full disk: %v", err)
	}
	if err := l.Err(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Err() with diskFull set: %v, want ENOSPC", err)
	}
	st, _ := l.Stat()
	if !st.DiskFull || st.FreeBytesAtLastPass != 100 {
		t.Fatalf("Stat with diskFull set: DiskFull %v FreeBytesAtLastPass %d", st.DiskFull, st.FreeBytesAtLastPass)
	}

	// statfs failing while full: the state is held and both reasons are reported.
	sf.failing = true
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if err := l.Err(); !errors.Is(err, sf.err) || !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Err() with statfs failing while full: %v", err)
	}
	if st, _ = l.Stat(); !st.DiskFull {
		t.Fatalf("DiskFull dropped while statfs failed: %+v", st)
	}

	// statfs works again and shows room: nothing stays frozen.
	sf.failing = false
	sf.free = 100_000
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	wantErrNil(t, l, "after space returned")
	if st, _ = l.Stat(); st.DiskFull || st.FreeBytesAtLastPass != 100_000 {
		t.Fatalf("Stat after space returned: DiskFull %v FreeBytesAtLastPass %d", st.DiskFull, st.FreeBytesAtLastPass)
	}
	if _, err := l.Append([]byte("x")); err != nil {
		t.Fatalf("append after space returned: %v", err)
	}
	if !lc.has("free space is back above MinFreeBytes") {
		t.Fatalf("clearing was not logged: %v", lc.msgs)
	}
	if n := strings.Count(strings.Join(lc.msgs, "\n"), "free space below MinFreeBytes"); n != 1 {
		t.Fatalf("the transition Warn should fire once, fired %d: %v", n, lc.msgs)
	}
}

// The Config.Logger documentation says which MinFreeBytes records are written once per transition
// (below the floor with nothing left to delete, and back above it) and which on every pass: the
// "cannot read free space" Warn is written by each pass whose statfs fails, not once per run of
// failures (Stage 2 audit L1-10).
func TestMinFreeBytesRecordsPerTransitionAndPerFailingPass(t *testing.T) {
	sf := &statfsSeam{free: 100_000, err: errors.New("statfs: input/output error")}
	var lc logCapture
	_, clk, l := retentionLog(t, func(c *Config) {
		c.MinFreeBytes = 5000
		c.Logger = lc.logger()
		c.statfs = sf.statfs
	})
	defer l.Close()
	fillSegments(t, l, clk, 3)
	count := func(sub string) int {
		lc.mu.Lock()
		defer lc.mu.Unlock()
		return strings.Count(strings.Join(lc.msgs, "\n"), sub)
	}
	const passes = 6
	sf.failing = true
	for i := 0; i < passes; i++ {
		if err := l.Prune(false); err != nil {
			t.Fatal(err)
		}
	}
	if n := count("cannot read free space"); n != passes {
		t.Fatalf("%d failing passes wrote %d \"cannot read free space\" Warns, want one per pass: %v", passes, n, lc.msgs)
	}

	sf.failing = false
	sf.free = 100
	for i := 0; i < passes; i++ {
		if err := l.Prune(false); err != nil {
			t.Fatal(err)
		}
	}
	if n := count("free space below MinFreeBytes"); n != 1 {
		t.Fatalf("%d passes below the floor wrote %d below-floor Warns, want one per transition: %v", passes, n, lc.msgs)
	}

	sf.free = 100_000
	for i := 0; i < passes; i++ {
		if err := l.Prune(false); err != nil {
			t.Fatal(err)
		}
	}
	if n := count("free space is back above MinFreeBytes"); n != 1 {
		t.Fatalf("%d passes back above the floor wrote %d Infos, want one per transition: %v", passes, n, lc.msgs)
	}
	if n := count("cannot read free space"); n != passes {
		t.Fatalf("a working statfs wrote more \"cannot read free space\" Warns: %d, want still %d", n, passes)
	}
}

// The caps, and so every MinFreeBytes record, run at each seal as well as in a prune pass
// (decision 24): with no Prune call at all, each seal reads the free space, so a failing statfs
// writes one "cannot read free space" Warn per seal, the below-floor Warn is written by the seal
// that finds the floor unmet, and the segments that seal deletes are logged from it (Stage 2 audit
// L1-10).
func TestMinFreeBytesRecordsAreWrittenAtEverySealWithoutAPrunePass(t *testing.T) {
	sf := &statfsSeam{free: 100_000, err: errors.New("statfs: input/output error")}
	var lc logCapture
	_, clk, l := retentionLog(t, func(c *Config) {
		c.MinFreeBytes = 5000
		c.Logger = lc.logger()
		c.statfs = sf.statfs
	})
	defer l.Close()
	count := func(sub string) int {
		lc.mu.Lock()
		defer lc.mu.Unlock()
		return strings.Count(strings.Join(lc.msgs, "\n"), sub)
	}
	const seals = 3
	sf.failing = true
	fillSegments(t, l, clk, seals) // each Flush seals one segment; Prune is never called
	if n := count("cannot read free space"); n != seals {
		t.Fatalf("%d seals with a failing statfs wrote %d \"cannot read free space\" Warns, want one per seal: %v", seals, n, lc.msgs)
	}

	sf.failing = false
	sf.free = 100 // hopeless: the seal deletes everything sealed and the floor still cannot be met
	appendN(t, l, 4)
	mustFlush(t, l)
	if n := count("free space below MinFreeBytes"); n != 1 {
		t.Fatalf("a seal below the floor wrote %d below-floor Warns, want one: %v", n, lc.msgs)
	}
	if n := count("deleted segment"); n != seals+1 {
		t.Fatalf("a seal below the floor logged %d deletions, want %d (every sealed segment): %v", n, seals+1, lc.msgs)
	}
	if n := count("cannot read free space"); n != seals {
		t.Fatalf("a working statfs wrote more \"cannot read free space\" Warns: %d, want still %d", n, seals)
	}
}

// A prune pass whose age rotation seals the quiet active segment runs the caps at that seal and
// again in the pass itself, so the "cannot read free space" Warn is written twice by one Prune
// (Stage 2 audit L1-10).
func TestAPruneWhoseAgeRotationSealsReadsTheFreeSpaceAtTheSealAndInThePass(t *testing.T) {
	sf := &statfsSeam{free: 100_000, err: errors.New("statfs: input/output error")}
	var lc logCapture
	_, clk, l := retentionLog(t, func(c *Config) {
		c.MinFreeBytes = 5000
		c.SegmentBytes = 1 << 20 // never seals by size
		c.SegmentMaxAge = time.Minute
		c.Logger = lc.logger()
		c.statfs = sf.statfs
	})
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)
	clk.Advance(2 * time.Minute)
	sf.failing = true
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	lc.mu.Lock()
	n := strings.Count(strings.Join(lc.msgs, "\n"), "cannot read free space")
	lc.mu.Unlock()
	if n != 2 {
		t.Fatalf("one Prune whose age rotation sealed wrote %d \"cannot read free space\" Warns, want 2 (the seal and the pass): %v", n, lc.msgs)
	}
}

// freeBytes is the real statfs reader that Open wires in; every other test injects a seam.
func TestFreeBytesReadsTheRealFilesystem(t *testing.T) {
	free, err := freeBytes(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if free == 0 {
		t.Fatal("freeBytes reports no free space on the test temp directory")
	}
	if _, err := freeBytes(t.TempDir() + "/missing"); err == nil {
		t.Fatal("freeBytes on a missing directory returned no error")
	}
}
