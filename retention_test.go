package quill

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A clock the test moves, shared by the simulator (mtimes) and the log (At stamps).
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *testClock { return &testClock{now: time.Unix(1_700_000_000, 0)} }
func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}
func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// logCapture records what the log logs.
type logCapture struct {
	mu   sync.Mutex
	msgs []string
}

func (c *logCapture) logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(c, nil))
}
func (c *logCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, string(p))
	return len(p), nil
}
func (c *logCapture) has(sub string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range c.msgs {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}

// openRetention opens a log on the simulator with a test clock, small segments and
// background maintenance off (the tests call Prune themselves).
func openRetention(t *testing.T, s *fsx.Sim, clk *testClock, mut func(*Config)) *Log {
	t.Helper()
	s.Clock = clk.Now
	return openSim(t, s, func(c *Config) {
		c.clock = clk.Now
		c.SegmentBytes = 60 // four records of payloadFor per segment
		c.Retention = time.Hour
		c.NoMaintenance = true
		if mut != nil {
			mut(c)
		}
	})
}

// retentionLog is the usual retention-test prologue: a fresh simulator and test clock and
// a log opened on them by openRetention. It registers no cleanup, so tests that close
// and reopen the log keep control; callers defer l.Close() themselves.
func retentionLog(t *testing.T, mut func(*Config)) (*fsx.Sim, *testClock, *Log) {
	t.Helper()
	s := fsx.NewSim()
	clk := newClock()
	return s, clk, openRetention(t, s, clk, mut)
}

// fillSegments appends and flushes n sealed segments of four records, one per minute.
func fillSegments(t *testing.T, l *Log, clk *testClock, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		appendN(t, l, 4)
		mustFlush(t, l)
		clk.Advance(time.Minute)
	}
}

func TestTheAgeSweepDeletesExpiredSealedSegmentsNeverTheActiveOneAndNeverReusesAnID(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 6)                              // segments 0..5 sealed at minutes 0..5, 6 active and empty
	clk.Advance(time.Hour - 3*time.Minute - 30*time.Second) // now 62.5 min: the segments of minutes 0..2 are older than an hour
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 4 || ids[0] != 3 {
		t.Fatalf("segments %v", ids)
	}
	st, _ := l.Stat()
	if st.First != 13 || st.Records != 12 || st.Bytes == 0 || st.Segments != 4 {
		t.Fatalf("%+v", st)
	}
	if _, err := l.Read(1, 10); !isGap(err, 1, 12) {
		t.Fatalf("below the first retained: %v", err)
	}
	// The marker was published and the directory synced before each unlink; the last unlink
	// may be undone by a crash, which resurrects a whole segment at the front: what survives
	// is a suffix, and the survivors are contiguous.
	s.Crash(nil)
	l2, err := Open(simConfig(s, "/log"))
	if err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) < 4 || len(ids) > 7 || ids[len(ids)-1] != 6 {
		t.Fatalf("after a crash %v", ids)
	}
	if recs, err := readAll(l2); err != nil || checkContiguous(t, recs, recs[0].Seq) != 24 {
		t.Fatalf("after a crash: %v %v", seqsOf(recs), err)
	}
	_ = l2.Close()
	// Everything expires: only the active segment is left, and the next id is fresh.
	l = openRetention(t, s, clk, nil)
	clk.Advance(2 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 || ids[0] != 6 {
		t.Fatalf("%v", ids)
	}
	if st, _ := l.Stat(); st.First != 25 || st.Records != 0 || st.Head != 24 {
		t.Fatalf("%+v", st)
	}
	appendN(t, l, 4)
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 || ids[1] != 7 {
		t.Fatalf("%v", ids)
	}
	// A late consumer: one started after retention expired gets a gap whose To is the
	// first retained sequence minus one (T10).
	c, err := l.Consumer("late", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Offset() != 25 {
		t.Fatalf("offset %d", c.Offset())
	}
	writeFile(t, s, "/log/consumers/older", encodeConsumer(consumerState{ID: logID(l.ID()), Next: 3}))
	older, err := l.Consumer("older", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer older.Close()
	if _, err := older.Read(10); !isGap(err, 3, 24) {
		t.Fatalf("%v", err)
	}
	if err := older.Commit(25); err != nil {
		t.Fatal(err)
	}
	recs, err := older.Read(10)
	if err != nil || checkContiguous(t, recs, 25) != 28 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	// Negative retention never sweeps.
	_ = l.Close()
	l = openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	clk.Advance(100 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("%v", ids)
	}
}

func TestAProtectedConsumerKeepsItsUnreadSegmentsAndACapOverridesItCountingDropped(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 6)
	p, err := l.Consumer("protected", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	u, err := l.Consumer("plain", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer u.Close()
	if err := p.Commit(6); err != nil { // read 1..5: segment 0 may go, segment 1 holds 6
		t.Fatal(err)
	}
	clk.Advance(2 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 6 || ids[0] != 1 {
		t.Fatalf("protected consumer at 6: segments %v", ids)
	}
	st, _ := l.Stat()
	if st.Dropped != 1 { // the plain consumer was at 1, inside segment 0
		t.Fatalf("dropped %d", st.Dropped)
	}
	// A consumer file that cannot be parsed protects everything.
	writeFile(t, s, "/log/consumers/broken", []byte("x"))
	if err := p.Commit(25); err != nil {
		t.Fatal(err)
	}
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 6 {
		t.Fatalf("broken consumer file: segments %v", ids)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	if res, _ := r.Check(); res.Healthy {
		t.Fatal("check is healthy with a broken consumer file")
	}
	if err := s.Remove("/log/consumers/broken"); err != nil {
		t.Fatal(err)
	}
	// MaxBytes overrides protection.
	if err := p.Commit(5); err != nil {
		t.Fatal(err)
	}
	_ = l.Close()
	l = openRetention(t, s, clk, func(c *Config) { c.MaxBytes = 160 })
	defer l.Close()
	before, _ := l.Stat()
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	st, _ = l.Stat()
	if st.Bytes > 160+60 || st.Dropped == 0 || st.First <= 5 {
		t.Fatalf("before %+v after %+v", before, st)
	}
}

func TestTheClockGuardDeletesAtMostAQuarterPerPassAndPruneForceLiftsIt(t *testing.T) {
	// T21: a forward step of the clock makes every sealed segment look expired.
	s := fsx.NewSim()
	clk := newClock()
	cap := &logCapture{}
	l := openRetention(t, s, clk, func(c *Config) { c.Logger = cap.logger() })
	defer l.Close()
	fillSegments(t, l, clk, 20)
	clk.Advance(365 * 24 * time.Hour)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 16 {
		t.Fatalf("one pass deleted %d segments", 21-len(ids))
	}
	if !cap.has("clock guard") {
		t.Fatalf("no warning logged: %v", cap.msgs)
	}
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 12 {
		t.Fatalf("second pass left %d", len(ids))
	}
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("force left %d", len(ids))
	}
	if !cap.has("deleted segment") {
		t.Fatal("deletions not logged")
	}
	// With few segments the guard still allows one.
	fillSegments(t, l, clk, 2)
	clk.Advance(2 * time.Hour)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("%v", ids)
	}
}

func TestDiskBoundsMaxBytesHoldsUnderAFastWriterAndMinFreeBytesDeletesOnlyThisLogsSegments(t *testing.T) {
	// T13.
	s := fsx.NewSim()
	clk := newClock()
	const maxBytes, segBytes = 1000, 200
	l := openRetention(t, s, clk, func(c *Config) {
		c.SegmentBytes = segBytes
		c.MaxBytes = maxBytes
		// Small, explicit limits keep the bound of decision 69 within a small factor of what the
		// traffic below reaches; at the defaults (1 MiB flush, 16 MiB buffer) it sits orders of
		// magnitude above a few KB of traffic and a cap several times too lax would pass.
		c.FlushBytes = 100
		c.MaxBufferBytes = 300
		c.MaxBatchBytes = 400
		c.MaxPayloadBytes = 800 // above MaxBatchBytes: the single-record term of the bound applies
	})
	defer l.Close()
	bound := int64(maxBytes + segBytes + l.cfg.MaxBufferBytes + max(l.cfg.MaxBatchBytes, l.cfg.MaxPayloadBytes+34))
	var peak int64
	for i := 0; i < 300; i++ {
		if i%50 == 49 {
			// One record larger than MaxBatchBytes: a single record is not subject to it.
			if _, err := l.Append(bytes.Repeat([]byte("y"), l.cfg.MaxPayloadBytes)); err != nil {
				t.Fatal(err)
			}
		} else if i%5 == 0 {
			if _, err := l.AppendBatch([][]byte{payloadFor(1), payloadFor(2), payloadFor(3)}); err != nil {
				t.Fatal(err)
			}
		} else {
			appendN(t, l, 1)
		}
		if i%3 == 0 {
			mustFlush(t, l)
		}
		var total int64
		for _, id := range segmentFiles(t, s, "/log") {
			total += int64(len(fileBytes(t, s, "/log/"+segmentName(id))))
		}
		peak = max(peak, total)
		if total > bound {
			t.Fatalf("segment bytes %d over the bound %d at step %d", total, bound, i)
		}
	}
	if peak < maxBytes/2 {
		t.Fatalf("the log never grew: peak %d", peak)
	}
	// A protected, stalled consumer does not hold the cap back.
	c, err := l.Consumer("stalled", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	st, _ := l.Stat()
	dropped := st.Dropped
	for i := 0; i < 100; i++ {
		appendN(t, l, 1)
		mustFlush(t, l)
	}
	st, _ = l.Stat()
	if st.Bytes > bound || st.Dropped <= dropped {
		t.Fatalf("%+v", st)
	}

	// MinFreeBytes with an injected free-space function.
	s2 := fsx.NewSim()
	free := uint64(10_000)
	l2 := openRetention(t, s2, clk, func(c *Config) {
		c.MinFreeBytes = 5000
		c.statfs = func(string) (uint64, error) { return free, nil }
	})
	defer l2.Close()
	fillSegments(t, l2, clk, 5)
	free = 4950 // below the floor by 50 bytes: one segment freed is enough
	if err := l2.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s2, "/log"); len(ids) != 5 || ids[0] != 1 {
		t.Fatalf("%v", ids)
	}
	free = 100 // hopeless: every sealed segment goes, the active stays, appends fail
	if err := l2.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s2, "/log"); len(ids) != 1 {
		t.Fatalf("%v", ids)
	}
	if _, err := l2.Append([]byte("x")); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("append on a full disk: %v", err)
	}
	free = 100_000
	if err := l2.Prune(false); err != nil {
		t.Fatal(err)
	}
	if _, err := l2.Append([]byte("x")); err != nil {
		t.Fatalf("after space returned: %v", err)
	}
}

func TestMaxRecordsKeepsRoughlyTheNewestRecords(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.MaxRecords = 10 })
	defer l.Close()
	fillSegments(t, l, clk, 6) // 24 records in 6 segments of 4
	st, _ := l.Stat()
	if st.Records > 10+4 || st.Records < 10-4 { // segment-granular on both sides
		t.Fatalf("%+v", st)
	}
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	st, _ = l.Stat()
	if st.Records != 8 || st.First != 17 { // the oldest sealed segments go until 10 or fewer remain; segment-granular
		t.Fatalf("%+v", st)
	}
}

func TestConsumerAgingUnderConsumerTTL(t *testing.T) {
	// T16.
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.ConsumerTTL = 30 * time.Minute })
	defer l.Close()
	fillSegments(t, l, clk, 4) // minutes 0..3; now minute 4
	c, err := l.Consumer("slow", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Commit(5); err != nil { // behind: its file is written at minute 4
		t.Fatal(err)
	}
	_ = c.Close()
	clk.Advance(time.Hour + 10*time.Minute) // records older than an hour, the file older than the TTL
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	st, _ := l.Stat()
	if len(st.StaleConsumers) != 1 || st.StaleConsumers[0] != "slow" {
		t.Fatalf("%+v", st)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("a stale consumer still pinned: %v", ids)
	}
	if _, err := s.Lstat("/log/consumers/slow"); err != nil {
		t.Fatal("the stale consumer's file was deleted")
	}
	// The returning consumer resumes from its offset, now a gap, and a commit makes it live.
	c, err = l.Consumer("slow", ConsumerOptions{})
	if err != nil || c.Offset() != 5 {
		t.Fatalf("%v %d", err, c.Offset())
	}
	if _, err := c.Read(1); !isGap(err, 5, 16) {
		t.Fatalf("%v", err)
	}
	fillSegments(t, l, clk, 3)
	if err := c.Commit(17); err != nil {
		t.Fatal(err)
	}
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.Stat(); len(st.StaleConsumers) != 0 {
		t.Fatalf("live again: %+v", st)
	}
	// A caught-up idle consumer is never stale: after a quiet period longer than the TTL,
	// new records arrive and it is behind with an old file, but the record at its offset
	// is new.
	if err := c.Commit(l.Head() + 1); err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * time.Hour)
	fillSegments(t, l, clk, 2)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	st, _ = l.Stat()
	if len(st.StaleConsumers) != 0 {
		t.Fatalf("a caught-up idle consumer went stale: %+v", st)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) < 2 {
		t.Fatalf("its unread segment was deleted: %v", ids)
	}
	// Restarting the writer does not reset the clock: the file's mtime is what counts.
	_ = c.Close()
	_ = l.Close()
	clk.Advance(time.Hour + 10*time.Minute)
	l = openRetention(t, s, clk, func(c *Config) { c.ConsumerTTL = 30 * time.Minute })
	defer l.Close()
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.Stat(); len(st.StaleConsumers) != 1 {
		t.Fatalf("after a restart: %+v", st)
	}
}

func TestSizeAccountingIsMeasured(t *testing.T) {
	// T17, across rotations, deletions, a crash and a reopen.
	s, clk, l := retentionLog(t, nil)
	check := func(l *Log, label string) {
		t.Helper()
		st, err := l.Stat()
		if err != nil {
			t.Fatal(err)
		}
		var sum int64
		for _, id := range segmentFiles(t, s, "/log") {
			sum += int64(len(fileBytes(t, s, "/log/"+segmentName(id))))
		}
		if st.Bytes != sum {
			t.Fatalf("%s: Stat().Bytes %d, files %d", label, st.Bytes, sum)
		}
		count := uint64(0)
		for off := st.First; off <= st.Head; {
			recs, err := l.Read(off, 1000)
			var g *GapError
			if errors.As(err, &g) {
				off = g.To + 1
				continue
			}
			if err != nil || len(recs) == 0 {
				t.Fatalf("%s: read %d: %v", label, off, err)
			}
			count += uint64(len(recs))
			off = recs[len(recs)-1].Seq + 1
		}
		if st.Records != count {
			t.Fatalf("%s: Stat().Records %d, scanned %d", label, st.Records, count)
		}
	}
	fillSegments(t, l, clk, 5)
	appendN(t, l, 2)
	mustFlush(t, l)
	check(l, "after rotations")
	clk.Advance(time.Hour - 2*time.Minute - 30*time.Second) // segments 0..2 expire
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	check(l, "after deletions")
	if err := s.Remove("/log/" + segmentName(4)); err != nil { // the middle one of 3, 4, 5
		t.Fatal(err)
	}
	appendN(t, l, 3)
	s.Crash(func(int, int) bool { return true })
	_ = l.Close()
	l = openRetention(t, s, clk, nil)
	defer l.Close()
	if len(l.holes) != 1 {
		t.Fatalf("holes %+v", l.holes)
	}
	check(l, "after a crash, a hole and a reopen")
}

func TestAQuietActiveSegmentIsRotatedByAgeSoRetentionCanReachIt(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = 10 * time.Minute })
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("rotated too early: %v", ids)
	}
	clk.Advance(11 * time.Minute)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("not rotated: %v", ids)
	}
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("an empty active segment was rotated: %v", ids)
	}
	clk.Advance(time.Hour)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("the rotated segment did not expire: %v", ids)
	}
	// Never with negative retention.
	_ = l.Close()
	l = openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)
	clk.Advance(100 * time.Hour)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("%v", ids)
	}
}

func TestTheBackgroundPassRunsUnlessMaintenanceIsOff(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	for _, off := range []bool{false, true} {
		dir := "/log-" + map[bool]string{false: "on", true: "off"}[off]
		l := openSim(t, s, func(c *Config) {
			c.clock = clk.Now
			c.SegmentBytes = 60
			c.Retention = time.Second
			c.PruneInterval = time.Second
			c.NoMaintenance = off
			c.Dir = dir
		})
		fillSegments(t, l, clk, 3)
		clk.Advance(time.Hour)
		deadline := time.Now().Add(2500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if len(segmentFiles(t, s, dir)) < 4 {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		n := len(segmentFiles(t, s, dir))
		_ = l.Close()
		if off && n != 4 {
			t.Fatalf("maintenance off: %d segments", n)
		}
		if !off && n == 4 {
			t.Fatal("maintenance on: the background pass never ran")
		}
	}
	_ = context.Background
	_ = bytes.Equal
}

// A sealed segment removed behind a live writer's back must not stop retention: its removal
// is already done, and the segments behind it still have to go.
func TestASealedSegmentRemovedBehindTheWritersBackDoesNotStopRetention(t *testing.T) {
	t.Run("age sweep", func(t *testing.T) {
		s, clk, l := retentionLog(t, nil)
		defer l.Close()
		fillSegments(t, l, clk, 6) // segments 0..5 sealed, 6 active and empty
		if err := s.Remove("/log/" + segmentName(0)); err != nil {
			t.Fatal(err)
		}
		clk.Advance(2 * time.Hour)
		if err := l.Prune(true); err != nil {
			t.Fatal(err)
		}
		if ids := segmentFiles(t, s, "/log"); len(ids) != 1 || ids[0] != 6 {
			t.Fatalf("segments left %v", ids)
		}
		if st, _ := l.Stat(); st.Segments != 1 {
			t.Fatalf("%+v", st)
		}
	})
	t.Run("MaxBytes", func(t *testing.T) {
		s := fsx.NewSim()
		clk := newClock()
		l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1; c.MaxBytes = 400 })
		defer l.Close()
		fillSegments(t, l, clk, 3) // under the cap, so nothing is deleted yet
		if err := s.Remove("/log/" + segmentName(0)); err != nil {
			t.Fatal(err)
		}
		fillSegments(t, l, clk, 30)
		if err := l.Err(); err != nil {
			t.Fatalf("Err %v", err)
		}
		if st, _ := l.Stat(); st.Bytes > 400+60 {
			t.Fatalf("%d bytes kept under MaxBytes 400: %v", st.Bytes, segmentFiles(t, s, "/log"))
		}
		if ids := segmentFiles(t, s, "/log"); len(ids) > 8 {
			t.Fatalf("%d files kept under MaxBytes 400: %v", len(ids), ids)
		}
	})
}
