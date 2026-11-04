package quill

import (
	"errors"
	"math/rand/v2"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A completely full filesystem has no block for quill.tail's temp file. Open of an existing log
// must not need one unless the marker has to move (Stage 2 audit L3b-8): otherwise the sweeps
// that would free space could never start.

// noTempFiles fails the creation of every temp file under /log with ENOSPC, as a filesystem
// with zero free blocks does, and counts the attempts.
func noTempFiles(s *fsx.Sim) *int {
	var n int
	s.Inject = noTempFileInjector(&n)
	return &n
}

// noTempFileInjector is the Inject function noTempFiles installs; n, if not nil, counts the
// refused creations.
func noTempFileInjector(n *int) func(fsx.Op, string) error {
	return func(op fsx.Op, name string) error {
		if (op == fsx.OpCreate || op == fsx.OpOpen) && strings.HasPrefix(name, "/log/"+tempPrefix) {
			if n != nil {
				*n++
			}
			return syscall.ENOSPC
		}
		return nil
	}
}

func TestOpenOfACleanlyClosedLogNeedsNoFreeBlockAndTheSweepRunsOnceOpen(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	sf := &statfsSeam{free: 100_000}
	cfg := func(c *Config) {
		c.MinFreeBytes = 5000
		c.statfs = sf.statfs
	}
	l := openRetention(t, s, clk, cfg)
	fillSegments(t, l, clk, 6)
	appendN(t, l, 2)
	mustFlush(t, l)
	head := l.Head()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	before := readTail(t, s, "/log")

	noTempFiles(s)
	sf.free = 0 // the disk is full: the floor can only be met by deleting segments
	l = openRetention(t, s, clk, cfg)
	defer l.Close()
	if l.Head() != head {
		t.Fatalf("head %d after reopening, want %d", l.Head(), head)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err() after an Open that wrote nothing: %v", err)
	}
	if got := readTail(t, s, "/log"); got != before {
		t.Fatalf("marker changed: %+v, was %+v", got, before)
	}
	if recs, err := readAll(l); err != nil || checkContiguous(t, recs, 1) != head {
		t.Fatalf("log not readable up to its head %d: %d records, %v", head, len(recs), err)
	}
	sealed := len(sealedIDs(l))
	if err := l.Prune(false); err != nil {
		t.Fatalf("prune pass on a full disk: %v", err)
	}
	if after := len(sealedIDs(l)); after >= sealed {
		t.Fatalf("the MinFreeBytes sweep freed nothing: sealed %d -> %d", sealed, after)
	}
	if l.Head() != head {
		t.Fatalf("head %d after the sweep, want %d", l.Head(), head)
	}
}

// The marker age of a restart that did not rewrite the marker still counts from Open, for an
// empty log (markerage_test.go) and for one that holds records.
func TestWriterMarkerAgeOfARestartedLogWithRecordsCountsFromOpen(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	s.Clock = clk.Now
	l := openSim(t, s, func(c *Config) { c.clock = clk.Now })
	appendN(t, l, 3)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	clk.Advance(time.Hour)
	l = openSim(t, s, func(c *Config) { c.clock = clk.Now })
	defer l.Close()
	clk.Advance(30 * time.Minute)
	st, err := l.Stat()
	if err != nil || st.MarkerAge != 30*time.Minute {
		t.Fatalf("MarkerAge %v, %v; want 30m0s", st.MarkerAge, err)
	}
}

// When recovery has moved the head past the marker the marker has to be written, and with no
// free block Open still succeeds: the data is durable and the marker is a lower bound (decision
// 39). The failed publish is a pending one, as after a failed flush (decision 60 d): it is in
// Err() and the log, readers see the older head, and the next flush that can write publishes.
func TestOpenWithAStaleMarkerOnAFullDiskSucceedsWithThePublishPending(t *testing.T) {
	s := fsx.NewSim()
	var lc logCapture
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	appendN(t, l, 5)
	mustFlush(t, l)
	stale := fileBytes(t, s, "/log/"+tailName)
	appendN(t, l, 5)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	writeFile(t, s, "/log/"+tailName, stale)

	attempts := noTempFiles(s)
	l, err := Open(func() Config {
		c := simConfig(s, "/log")
		c.SegmentBytes = 1 << 20
		c.Logger = lc.logger()
		return c
	}())
	if err != nil {
		t.Fatalf("Open with a stale marker and no free block: %v", err)
	}
	defer l.Close()
	if *attempts == 0 {
		t.Fatal("the marker publish was not attempted")
	}
	if err := l.Err(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Err() %v, want the pending publish (ENOSPC)", err)
	}
	if !lc.has("cannot publish quill.tail") {
		t.Fatalf("the pending publish was not logged: %q", lc.msgs)
	}
	if l.Head() != 10 {
		t.Fatalf("head %d, want 10 (nothing lost)", l.Head())
	}
	// A reader sees the older, valid head, and exactly the records up to it.
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if h, err := r.Head(); err != nil || h != 5 {
		t.Fatalf("reader head %d, %v; want the stale 5", h, err)
	}
	if recs, err := r.Read(1, 100); err != nil || checkContiguous(t, recs, 1) != 5 {
		t.Fatalf("reader records %v, %v", seqsOf(recs), err)
	}
	// The log keeps appending without reusing or losing a sequence.
	if seq, err := l.Append(payloadFor(11)); err != nil || seq != 11 {
		t.Fatalf("append: %d, %v", seq, err)
	}
	// Still full: the flush writes its data, the marker waits.
	if err := l.Flush(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("flush on a full disk: %v", err)
	}
	if h, _ := r.Head(); h != 5 {
		t.Fatalf("reader head %d while the publish is pending, want 5", h)
	}
	// Space returns: the next flush publishes, and the reader sees the real head.
	s.Inject = nil
	if err := l.Flush(); err != nil {
		t.Fatalf("flush once space is back: %v", err)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err() after the publish: %v", err)
	}
	if h, err := r.Head(); err != nil || h != 11 {
		t.Fatalf("reader head %d, %v; want 11", h, err)
	}
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 11 {
		t.Fatalf("records %v, %v", seqsOf(recs), err)
	}
}

// Close retries a pending publish too, so a CLI verb that opened a log with a stale marker on
// a full disk either publishes it or says so; it never exits 0 with the marker still behind.
func TestClosePublishesTheMarkerAnOpenOnAFullDiskCouldNot(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	appendN(t, l, 5)
	mustFlush(t, l)
	stale := fileBytes(t, s, "/log/"+tailName)
	appendN(t, l, 5)
	mustFlush(t, l)
	_ = l.Close()
	writeFile(t, s, "/log/"+tailName, stale)

	noTempFiles(s)
	l = openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	if err := l.Close(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Close with the publish still failing: %v, want ENOSPC", err)
	}
	if tl := readTail(t, s, "/log"); tl.Seq != 5 {
		t.Fatalf("marker moved to %+v with no space", tl)
	}
	s.Inject = nil
	l = openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if tl := readTail(t, s, "/log"); tl.Seq != 10 {
		t.Fatalf("marker %+v after space returned, want seq 10", tl)
	}
}

// staleMarkerLog builds a closed log in the simulator whose quill.tail is behind the data: the
// marker is copied after `before` records' worth of segments and restored after `more` further
// segments. It returns the marker restored.
func staleMarkerLog(t *testing.T, s *fsx.Sim, clk *testClock, cfg func(*Config), before, more int) tail {
	t.Helper()
	l := openRetention(t, s, clk, cfg)
	fillSegments(t, l, clk, before)
	stale := fileBytes(t, s, "/log/"+tailName)
	fillSegments(t, l, clk, more)
	appendN(t, l, 2)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	writeFile(t, s, "/log/"+tailName, stale)
	return readTail(t, s, "/log")
}

// The headline of L3b-8: with the marker behind the data and no free block, Open succeeds, and
// the sweep that frees space runs even though the marker cannot be published. A pass deletes
// only the sealed segments below the one the marker on disk names: the marker is a lower bound
// and parses forward from that segment (decision 39), so every segment from it on is kept and
// a crash right after the unlink loses nothing the marker promised (decision 65).
func TestTheMinFreeBytesSweepRunsOnAFullDiskWhileTheOpenTimeMarkerPublishIsPending(t *testing.T) {
	for _, tc := range []struct {
		name         string
		before, more int
		wantSealed   []uint64
	}{
		// The marker names segment 6, the one after the sealed ones when it was written:
		// everything below it goes, it and the newer ones stay.
		{"marker names the newest sealed segment", 6, 0, []uint64{6}},
		// The marker names segment 3, long since sealed: it and the ones after stay.
		{"marker names an older segment", 3, 3, []uint64{3, 4, 5, 6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			clk := newClock()
			sf := &statfsSeam{free: 1 << 40}
			cfg := func(c *Config) {
				c.MinFreeBytes = 1 << 30
				c.statfs = sf.statfs
			}
			stale := staleMarkerLog(t, s, clk, cfg, tc.before, tc.more)
			total := uint64(4*(tc.before+tc.more) + 2)

			noTempFiles(s)
			sf.free = 0
			l := openRetention(t, s, clk, cfg)
			defer l.Close()
			if err := l.Err(); !errors.Is(err, syscall.ENOSPC) {
				t.Fatalf("Err() %v, want the pending publish", err)
			}
			if l.Head() != total {
				t.Fatalf("head %d, want %d", l.Head(), total)
			}
			if err := l.Prune(false); err != nil {
				t.Fatalf("prune pass with the marker publish pending: %v", err)
			}
			got := sealedIDs(l)
			if len(got) != len(tc.wantSealed) {
				t.Fatalf("sealed %v after the sweep, want %v", got, tc.wantSealed)
			}
			for i := range got {
				if got[i] != tc.wantSealed[i] {
					t.Fatalf("sealed %v after the sweep, want %v", got, tc.wantSealed)
				}
			}
			if cur := readTail(t, s, "/log"); cur != stale {
				t.Fatalf("marker moved to %+v with no free block", cur)
			}
			if err := l.Err(); !errors.Is(err, syscall.ENOSPC) {
				t.Fatalf("Err() %v after the sweep, want the publish still pending", err)
			}
			// The pending publish is still retried: once space is back the marker is published.
			s.Inject = nil
			if err := l.Flush(); err != nil {
				t.Fatalf("flush once space is back: %v", err)
			}
			if cur := readTail(t, s, "/log"); cur.Seq != total {
				t.Fatalf("marker %+v after the retry, want seq %d", cur, total)
			}
		})
	}
}

// A crash right after such a sweep, with every unsynced operation lost, reopens with no hole and
// no record lost beyond what the sweep deleted.
func TestACrashAfterASweepUnderAPendingOpenTimeMarkerLosesNothingTheMarkerPromised(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	sf := &statfsSeam{free: 1 << 40}
	cfg := func(c *Config) {
		c.MinFreeBytes = 1 << 30
		c.statfs = sf.statfs
	}
	staleMarkerLog(t, s, clk, cfg, 3, 3)
	noTempFiles(s)
	sf.free = 0
	l := openRetention(t, s, clk, cfg)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	s.Crash(func(int, int) bool { return false })
	s.Inject = nil
	_ = l.Close()
	l2 := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l2.Close()
	if l2.Head() != 26 {
		t.Fatalf("head %d after the crash, want 26", l2.Head())
	}
	recs, err := readAll(l2)
	// The last unlink of the pass is not synced until a later one, so a crash may bring segment 2
	// back (first record 9); never a hole, and never past segment 3, which the marker names.
	if err != nil || len(recs) == 0 || (recs[0].Seq != 13 && recs[0].Seq != 9) || checkContiguous(t, recs, recs[0].Seq) != 26 {
		t.Fatalf("records after the crash: %v, %v", seqsOf(recs), err)
	}
	if len(l2.holes) != 0 {
		t.Fatalf("holes %v", l2.holes)
	}
}

// Crash at every operation of such a sweep, with random subsets of the unsynced operations
// surviving: the log always reopens at its real head with no hole.
func TestACrashAtEveryStepOfASweepUnderAPendingOpenTimeMarkerReopensWhole(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 8))
	for at := 1; ; at++ {
		fired := false
		for k := 0; k < 20; k++ {
			mask := rng.Uint64()
			s := fsx.NewSim()
			clk := newClock()
			sf := &statfsSeam{free: 1 << 40}
			cfg := func(c *Config) {
				c.MinFreeBytes = 1 << 30
				c.statfs = sf.statfs
			}
			staleMarkerLog(t, s, clk, cfg, 3, 3)
			s.Crash(func(int, int) bool { return true }) // the setup is durable
			inner := noTempFileInjector(nil)
			s.Inject = inner
			sf.free = 0
			l := openRetention(t, s, clk, cfg)
			count, crashed := 0, false
			s.CrashKeep = func(i, _ int) bool { return mask&(1<<uint(i%64)) != 0 }
			s.Inject = func(op fsx.Op, name string) error {
				count++
				if crashed {
					return errors.New("dead")
				}
				if count == at {
					crashed, fired = true, true
					return fsx.ErrCrash
				}
				return inner(op, name)
			}
			_ = l.Prune(false)
			s.Inject = nil
			if !crashed {
				break
			}
			l2, err := Open(Config{Dir: "/log", FlushInterval: time.Hour, SegmentBytes: 60, Retention: -1, fs: s})
			if err != nil {
				t.Fatalf("crash before operation %d, mask %x: reopen: %v", at, mask, err)
			}
			if l2.Head() != 26 || len(l2.holes) != 0 {
				t.Fatalf("crash before operation %d, mask %x: head %d, holes %v", at, mask, l2.Head(), l2.holes)
			}
			_ = l2.Close()
		}
		if !fired {
			break
		}
	}
}

// The relief is for the publish Open could not write only: once that one lands, a marker publish
// that fails in the running writer stops a prune pass at the publish again (decision 60 d).
func TestAMarkerPublishFailingAfterTheOpenTimeOneLandedStillStopsAPrunePass(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	cfg := func(c *Config) { c.Retention = time.Hour }
	staleMarkerLog(t, s, clk, cfg, 3, 3)
	noTempFiles(s)
	l := openRetention(t, s, clk, cfg)
	defer l.Close()
	s.Inject = nil
	if err := l.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := l.Err(); err != nil {
		t.Fatalf("Err() after the Open-time publish landed: %v", err)
	}
	appendN(t, l, 1)
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpRename {
			return errENOSPC
		}
		return nil
	}
	_ = l.Flush()
	clk.Advance(2 * time.Hour)
	before := len(sealedIDs(l))
	if err := l.Prune(true); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("prune with the running writer's publish failing: %v, want ENOSPC", err)
	}
	if after := len(sealedIDs(l)); after != before {
		t.Fatalf("sealed %d -> %d: a pass stopped at the publish deleted segments", before, after)
	}
}
