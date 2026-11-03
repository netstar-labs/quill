package quill

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// T15: archive.

func TestArchiveCopiesSealedSegmentsVerifiedIdempotentlyAndTheDestinationReadsTheSame(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 5)
	appendN(t, l, 1) // in the active segment: not archived
	mustFlush(t, l)
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/arch"); len(ids) != 5 {
		t.Fatalf("archived %v", ids)
	}
	ar, err := OpenReader("/arch", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	defer ar.Close()
	recs, err := readerAll(ar)
	if err != nil || checkContiguous(t, recs, 1) != 20 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	if head, _ := ar.Head(); head != 20 {
		t.Fatalf("archive head %d", head)
	}
	list, _ := l.Consumers()
	if len(list) != 1 || list[0].Name != "_archive" || !list[0].Protected || list[0].Offset != 21 {
		t.Fatalf("%+v", list)
	}
	// A second run copies nothing.
	before := s.Ops()
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	if ops := s.Ops() - before; ops > 60 {
		t.Fatalf("a no-op archive did %d operations", ops)
	}
	// After a Roll the newest records are archived too, and the destination still reads.
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	recs, err = readerAll(ar)
	if err != nil || checkContiguous(t, recs, 1) != 21 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	// The archive is a log: a writable open works and continues the sequence.
	al, err := Open(simConfig(s, "/arch"))
	if err != nil {
		t.Fatal(err)
	}
	if seq, _ := al.Append([]byte("x")); seq != 22 {
		t.Fatalf("seq %d", seq)
	}
	_ = al.Close()
	// Another log's archive is refused.
	other := openSim(t, s, func(c *Config) { c.Dir = "/other" })
	appendN(t, other, 1)
	_ = other.Roll()
	defer other.Close()
	if err := archive(s, "/other", "/arch"); !errors.Is(err, ErrWrongLog) {
		t.Fatalf("%v", err)
	}
}

func TestArchiveCopiesEverySegmentByteForByteIncludingOneSpanningSeveralCopyChunks(t *testing.T) {
	compare := func(t *testing.T, s *fsx.Sim, src, dst string, want int) {
		t.Helper()
		ids := segmentFiles(t, s, src)
		if len(ids) < want {
			t.Fatalf("source has %d segments, want at least %d", len(ids), want)
		}
		sealed := 0
		for _, id := range ids {
			got := segmentFiles(t, s, dst)
			if !slices.Contains(got, id) {
				continue // the active segment is not archived
			}
			sealed++
			a := fileBytes(t, s, filepath.Join(src, segmentName(id)))
			b := fileBytes(t, s, filepath.Join(dst, segmentName(id)))
			if !bytes.Equal(a, b) {
				t.Fatalf("segment %d: copy of %d bytes differs from its %d byte source", id, len(b), len(a))
			}
		}
		if sealed < want {
			t.Fatalf("compared %d segments, want at least %d", sealed, want)
		}
	}
	t.Run("many small segments", func(t *testing.T) {
		s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
		defer l.Close()
		fillSegments(t, l, clk, 5)
		if err := archive(s, "/log", "/arch"); err != nil {
			t.Fatal(err)
		}
		compare(t, s, "/log", "/arch", 5)
	})
	t.Run("one record spanning chunks", func(t *testing.T) {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.MaxPayloadBytes = 4 * windowChunk })
		defer l.Close()
		big := []byte(strings.Repeat("0123456789abcdef", (2*windowChunk+123)/16+1))
		if _, err := l.Append(big); err != nil {
			t.Fatal(err)
		}
		if err := l.Roll(); err != nil {
			t.Fatal(err)
		}
		if err := archive(s, "/log", "/arch"); err != nil {
			t.Fatal(err)
		}
		compare(t, s, "/log", "/arch", 1)
	})
}

func TestArchiveFailsOnAShortWriteAndLeavesNoTempFileInTheDestination(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3)
	if err := s.Mkdir("/arch", 0o700); err != nil {
		t.Fatal(err)
	}
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite && filepath.Dir(name) == "/arch" && strings.HasPrefix(filepath.Base(name), tempPrefix) {
			return &fsx.ShortWrite{N: 5}
		}
		return nil
	}
	err := archive(s, "/log", "/arch")
	s.Inject = nil
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("archive with a short write: %v", err)
	}
	names, rerr := s.ReadDir("/arch")
	if rerr != nil {
		t.Fatal(rerr)
	}
	for _, n := range names {
		if strings.HasPrefix(n, tempPrefix) {
			t.Fatalf("temp file %q left in the destination: %v", n, names)
		}
	}
	if ids := segmentFiles(t, s, "/arch"); len(ids) != 0 {
		t.Fatalf("a segment was published after a failed copy: %v", ids)
	}
	// With the fault gone the same archive completes.
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if ids := segmentFiles(t, s, "/arch"); len(ids) != 3 {
		t.Fatalf("after the retry: %v", ids)
	}
}

func TestArchiveFailsWhenTheDestinationCannotSyncAndLeavesNoTempFile(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3)
	if err := s.Mkdir("/arch", 0o700); err != nil {
		t.Fatal(err)
	}
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpSync && filepath.Dir(name) == "/arch" && strings.HasPrefix(filepath.Base(name), tempPrefix) {
			return syscall.ENOSPC
		}
		return nil
	}
	err := archive(s, "/log", "/arch")
	s.Inject = nil
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("archive with a failing fsync: %v", err)
	}
	names, rerr := s.ReadDir("/arch")
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(names) != 0 {
		t.Fatalf("destination holds %v after a failed fsync, want nothing", names)
	}
}

func TestArchiveStopsAtACorruptSegmentAndReportsASourceHole(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 6)
	seg2 := "/log/" + segmentName(2)
	data := fileBytes(t, s, seg2)
	data[len(data)-2] ^= 1
	writeFile(t, s, seg2, data)
	err := archive(s, "/log", "/arch")
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("%v", err)
	}
	if ids := segmentFiles(t, s, "/arch"); len(ids) != 2 {
		t.Fatalf("copied %v before the corrupt one", ids)
	}
	list, _ := l.Consumers()
	if list[0].Offset != 9 {
		t.Fatalf("committed %d", list[0].Offset)
	}
	// Restoring the file lets it continue.
	data[len(data)-2] ^= 1
	writeFile(t, s, seg2, data)
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/arch"); len(ids) != 6 {
		t.Fatalf("%v", ids)
	}
	// A source that lost a segment: the destination holds the hole and says so.
	s2 := fsx.NewSim()
	l2 := openRetention(t, s2, clk, func(c *Config) { c.Retention = -1 })
	defer l2.Close()
	fillSegments(t, l2, clk, 5)
	if err := s2.Remove("/log/" + segmentName(2)); err != nil {
		t.Fatal(err)
	}
	err = archive(s2, "/log", "/arch")
	if !isGap(err, 9, 12) {
		t.Fatalf("%v", err)
	}
	ar, err := OpenReader("/arch", ReaderOptions{fs: s2})
	if err != nil {
		t.Fatal(err)
	}
	defer ar.Close()
	// A reader learns a hole inside a sealed segment's derived range when it scans that
	// segment: Check does, and so does a read that crosses it.
	res, err := ar.Check()
	holes := 0
	for _, p := range res.Problems {
		if p.Kind == "hole" {
			holes++
		}
	}
	if err != nil || res.Healthy || holes != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := readerAll(ar); !isGap(err, 9, 12) {
		t.Fatalf("%v", err)
	}
	st, err := ar.Stat()
	if err != nil || len(st.Holes) != 1 || st.Holes[0] != (Hole{9, 12}) || st.Head != 20 {
		t.Fatalf("%+v %v", st, err)
	}
	if err := archive(s2, "/log", "/arch"); err != nil {
		t.Fatalf("a second run after the hole: %v", err)
	}
	// Retention lost the first segments before the archiver ran: a gap below the offset.
	s3 := fsx.NewSim()
	l3 := openRetention(t, s3, clk, nil)
	defer l3.Close()
	fillSegments(t, l3, clk, 3)
	if err := archive(s3, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	fillSegments(t, l3, clk, 3)
	if err := s3.Remove("/log/" + segmentName(3)); err != nil {
		t.Fatal(err)
	}
	if err := s3.Remove("/log/" + segmentName(4)); err != nil {
		t.Fatal(err)
	}
	// The archiver's offset is 13; segments 3 and 4 (13..20) are gone by hand.
	if err := archive(s3, "/log", "/arch"); !isGap(err, 13, 20) {
		t.Fatalf("%v", err)
	}
}

// The documented reset, deleting the _archive consumer, recreates it at the first retained
// sequence, so a loss between the destination's last record and that sequence is below no
// offset. Archive compares the destination's quill.tail with the first segment it copies and
// returns the *GapError, as it does when the offset itself is behind the source (A5-29).
func TestArchiveAfterTheConsumerIsRecreatedReportsTheRecordsTheDestinationLacks(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3) // segments 0..2, records 1..12
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	fillSegments(t, l, clk, 3) // segments 3..5, records 13..24, never archived
	for id := 0; id <= 3; id++ {
		if err := s.Remove("/log/" + segmentName(uint64(id))); err != nil {
			t.Fatal(err)
		}
	}
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteConsumer(archiveConsumer); err != nil {
		t.Fatal(err)
	}
	r.Close()
	if err := archive(s, "/log", "/arch"); !isGap(err, 13, 16) {
		t.Fatalf("%v", err)
	}
	// The gap is reported by the run that finds it; the destination's tail now names what it holds.
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatalf("a run with nothing to copy: %v", err)
	}
}

// oneRecordSegments appends n records, each flushed and sealed in a segment of its own, so
// the loss of one segment file loses exactly one record.
func oneRecordSegments(t *testing.T, l *Log, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		appendN(t, l, 1)
		mustFlush(t, l)
		if err := l.Roll(); err != nil {
			t.Fatal(err)
		}
	}
}

// A source whose oldest segments retention already deleted is not a source with a hole: a new
// destination has no record the archive lacks, and the archive consumer starts at the first
// retained sequence (decisions 41, 67, 68). The first archive must return nil, not a gap.
func TestArchiveOfARetentionTrimmedSourceIntoANewDestinationReportsNoGap(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 6) // segments 0..5, records 1..24
	clk.Advance(time.Hour - 3*time.Minute - 30*time.Second)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) == 0 || ids[0] != 3 {
		t.Fatalf("segments %v, want the oldest three trimmed", ids)
	}
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatalf("first archive of a trimmed log: %v", err)
	}
	ar, err := OpenReader("/arch", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	defer ar.Close()
	st, err := ar.Stat()
	if err != nil || st.First != 13 || st.Head != 24 || len(st.Holes) != 0 {
		t.Fatalf("%+v %v", st, err)
	}
}

// The _archive offset belongs to the source, not to a destination: once the consumer has
// advanced, a second, new destination receives only the records at or above it, a suffix, and
// Archive returns nil (decision 85). Rewinding the consumer copies more.
func TestArchiveIntoANewDestinationAfterTheConsumerAdvancedCopiesOnlyTheSuffix(t *testing.T) {
	s, _, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	appendN(t, l, 30)
	mustFlush(t, l)
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if err := archive(s, "/log", "/dA"); err != nil {
		t.Fatal(err)
	}
	appendN(t, l, 30)
	mustFlush(t, l)
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if list, err := l.Consumers(); err != nil || len(list) != 1 || list[0].Offset != 31 {
		t.Fatalf("before the second archive: %+v %v", list, err)
	}
	if err := archive(s, "/log", "/dB"); err != nil {
		t.Fatalf("a new destination behind the consumer's offset: %v", err)
	}
	list, err := l.Consumers()
	if err != nil || len(list) != 1 || list[0].Name != "_archive" || list[0].Offset != 61 {
		t.Fatalf("%+v %v", list, err)
	}
	first := func(dir string) (uint64, uint64) {
		t.Helper()
		ar, err := OpenReader(dir, ReaderOptions{fs: s})
		if err != nil {
			t.Fatal(err)
		}
		defer ar.Close()
		st, err := ar.Stat()
		if err != nil || len(st.Holes) != 0 {
			t.Fatalf("%s: %+v %v", dir, st, err)
		}
		return st.First, st.Head
	}
	if f, h := first("/dA"); f != 1 || h != 30 {
		t.Fatalf("dA first %d head %d", f, h)
	}
	if f, h := first("/dB"); f != 31 || h != 60 {
		t.Fatalf("dB first %d head %d, want 31 and 60: the offset is the source's", f, h)
	}
}

// A hole of exactly one record between two copied segments is a gap like any other.
func TestArchiveReportsAHoleOfOneRecordBetweenCopiedSegments(t *testing.T) {
	s, _, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	oneRecordSegments(t, l, 5) // segments 0..4, records 1..5
	if err := s.Remove("/log/" + segmentName(2)); err != nil {
		t.Fatal(err)
	}
	if err := archive(s, "/log", "/arch"); !isGap(err, 3, 3) {
		t.Fatalf("%v", err)
	}
}

// A single record lost between the destination's last record and the first segment the next
// run copies is found from the destination's quill.tail even after the consumer was recreated.
func TestArchiveReportsOneRecordLostBeforeTheFirstSegmentItCopies(t *testing.T) {
	s, _, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	oneRecordSegments(t, l, 3) // segments 0..2, records 1..3
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	oneRecordSegments(t, l, 3) // segments 3..5, records 4..6, never archived
	for id := 0; id <= 3; id++ {
		if err := s.Remove("/log/" + segmentName(uint64(id))); err != nil {
			t.Fatal(err)
		}
	}
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteConsumer(archiveConsumer); err != nil {
		t.Fatal(err)
	}
	r.Close()
	if err := archive(s, "/log", "/arch"); !isGap(err, 4, 4) {
		t.Fatalf("%v", err)
	}
}

func TestTheArchiverIsAProtectedConsumerUntilACapOverridesIt(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 4)
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	fillSegments(t, l, clk, 4) // not yet archived
	clk.Advance(2 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 5 || ids[0] != 4 {
		t.Fatalf("the age sweep took unarchived segments: %v", ids)
	}
	_ = l.Close()
	l = openRetention(t, s, clk, func(c *Config) { c.MaxBytes = 120 })
	defer l.Close()
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) > 3 {
		t.Fatalf("the cap did not override: %v", ids)
	}
	st, _ := l.Stat()
	if st.Dropped == 0 {
		t.Fatal("dropped not counted")
	}
	// The archiver goes stale like any protected consumer.
	_ = l.Close()
	l = openRetention(t, s, clk, func(c *Config) { c.ConsumerTTL = time.Minute })
	defer l.Close()
	fillSegments(t, l, clk, 3)
	clk.Advance(3 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.Stat(); len(st.StaleConsumers) != 1 || st.StaleConsumers[0] != "_archive" {
		t.Fatalf("%+v", st)
	}
}

func TestArchiveOnTheRealFileSystem(t *testing.T) {
	src := filepath.Join(t.TempDir(), "log")
	dst := filepath.Join(t.TempDir(), "arch")
	l := openDir(t, src, func(c *Config) { c.SegmentBytes = 256 })
	defer l.Close()
	for i := 0; i < 5; i++ {
		appendN(t, l, 8)
		mustFlush(t, l)
	}
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if err := Archive(src, dst); err != nil {
		t.Fatal(err)
	}
	r, err := OpenReader(dst, ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	recs, err := readerAll(r)
	if err != nil || checkContiguous(t, recs, 1) != 40 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	if err := Archive(src, dst); err != nil {
		t.Fatalf("second run: %v", err)
	}
}

// An archive destination that is the live log itself, or a symbolic link to it, is refused
// and quill.tail, which only the writer may write, is left as it was (decisions 8, 28).
func TestArchiveRefusesTheSourceLogAsItsDestinationAndLeavesTheLiveMarker(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "log")
	l := openDir(t, dir, func(c *Config) { c.SegmentBytes = 256 })
	defer l.Close()
	for i := 0; i < 3; i++ {
		appendN(t, l, 8)
		mustFlush(t, l)
	}
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	appendN(t, l, 3) // flushed records in the active segment, past the last sealed one
	mustFlush(t, l)
	head := l.Head()
	before, err := os.ReadFile(filepath.Join(dir, tailName))
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, alias); err != nil {
		t.Fatal(err)
	}
	for _, dst := range []string{dir, dir + "/", filepath.Join(dir, "..", "log"), alias} {
		if err := Archive(dir, dst); err == nil {
			t.Fatalf("Archive(dir, %q) returned nil", dst)
		}
		after, err := os.ReadFile(filepath.Join(dir, tailName))
		if err != nil || !bytes.Equal(after, before) {
			t.Fatalf("Archive(dir, %q) rewrote the live marker %q to %q (%v)", dst, before, after, err)
		}
		r, err := OpenReader(dir, ReaderOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if h, _ := r.Head(); h != head {
			t.Fatalf("a reader sees head %d, the writer's is %d", h, head)
		}
		_ = r.Close()
	}
}

// A destination that is another log directory with a live writer is refused too: its marker is
// that writer's to write.
func TestArchiveRefusesADestinationHeldByALiveWriter(t *testing.T) {
	src := filepath.Join(t.TempDir(), "log")
	dst := filepath.Join(t.TempDir(), "arch")
	l := openDir(t, src, func(c *Config) { c.SegmentBytes = 256 })
	defer l.Close()
	for i := 0; i < 3; i++ {
		appendN(t, l, 8)
		mustFlush(t, l)
	}
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if err := Archive(src, dst); err != nil {
		t.Fatal(err)
	}
	aw := openDir(t, dst, func(c *Config) { c.SegmentBytes = 256 })
	defer aw.Close()
	appendN(t, aw, 2)
	mustFlush(t, aw)
	appendN(t, l, 8)
	mustFlush(t, l)
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dst, tailName))
	if err != nil {
		t.Fatal(err)
	}
	if err := Archive(src, dst); !errors.Is(err, ErrLocked) {
		t.Fatalf("got %v, want ErrLocked", err)
	}
	if after, err := os.ReadFile(filepath.Join(dst, tailName)); err != nil || !bytes.Equal(after, before) {
		t.Fatalf("the live destination's marker changed from %q to %q (%v)", before, after, err)
	}
}

// A power loss that persists the committed _archive offset but not the destination's own
// directory entry must not leave the archive empty: for every single unsynced item a crash
// could drop, a second Archive run ends with every sealed segment in the destination.
func TestArchiveSurvivesEverySingleLostItemAfterASuccessfulRun(t *testing.T) {
	setup := func() *fsx.Sim {
		s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
		fillSegments(t, l, clk, 5)
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
		if err := archive(s, "/log", "/arch"); err != nil {
			t.Fatal(err)
		}
		return s
	}
	n := setup().Unsynced()
	if n == 0 {
		t.Fatal("the simulator reports nothing unsynced after an archive")
	}
	for drop := 0; drop < n; drop++ {
		s := setup()
		s.Crash(func(i, _ int) bool { return i != drop })
		if err := archive(s, "/log", "/arch"); err != nil {
			t.Fatalf("dropping item %d of %d: second archive: %v", drop, n, err)
		}
		if ids := segmentFiles(t, s, "/arch"); len(ids) != 5 {
			t.Fatalf("dropping item %d of %d: the destination holds segments %v after the second archive, want 5", drop, n, ids)
		}
	}
}

// A destination whose quill.format names a newer version is ErrFutureVersion, as Open and
// OpenReader report it; any other unusable marker is ErrCorrupt. Neither marker is touched.
func TestArchiveIntoADestinationOfANewerFormatIsErrFutureVersionAndOtherDamageIsErrCorrupt(t *testing.T) {
	src := filepath.Join(t.TempDir(), "log")
	l := openDir(t, src, func(c *Config) { c.SegmentBytes = 256 })
	defer l.Close()
	appendN(t, l, 8)
	mustFlush(t, l)
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name   string
		marker string
		future bool
	}{
		{"newer", "quill/2 0123456789abcdef0123456789abcdef\n", true},
		{"garbage", "not a marker\n", false},
	} {
		dst := filepath.Join(t.TempDir(), "arch")
		if err := os.Mkdir(dst, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, formatName), []byte(c.marker), 0o600); err != nil {
			t.Fatal(err)
		}
		err := Archive(src, dst)
		if errors.Is(err, ErrFutureVersion) != c.future || errors.Is(err, ErrCorrupt) == c.future {
			t.Fatalf("%s: Archive = %v, want ErrFutureVersion %v and ErrCorrupt %v", c.name, err, c.future, !c.future)
		}
		if got, err := os.ReadFile(filepath.Join(dst, formatName)); err != nil || string(got) != c.marker {
			t.Fatalf("%s: the destination's marker is now %q (%v)", c.name, got, err)
		}
	}
}

// A run killed in the middle of a segment copy leaves its temp file in the destination, where
// no writer's Open ever looks. The next run removes the ones older than an hour and keeps a
// younger one, which may belong to a live archiver (decision 67).
func TestArchiveRemovesStaleTempFilesFromTheDestinationAfterAKilledRun(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3)
	if err := s.Mkdir("/arch", 0o700); err != nil {
		t.Fatal(err)
	}
	// The kill: from the first write to a temp file in the destination on, every operation
	// fails, so neither the error path nor anything else removes the file.
	var killed atomic.Bool
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite && filepath.Dir(name) == "/arch" && strings.HasPrefix(filepath.Base(name), tempPrefix) {
			killed.Store(true)
		}
		if killed.Load() {
			return syscall.EIO
		}
		return nil
	}
	s.Clock = func() time.Time { return time.Now().Add(-2 * time.Hour) } // the killed run began two hours ago
	if err := archive(s, "/log", "/arch"); err == nil {
		t.Fatal("the killed run succeeded")
	}
	s.Inject = nil
	tempFiles := func() []string {
		names, err := s.ReadDir("/arch")
		if err != nil {
			t.Fatal(err)
		}
		var temps []string
		for _, n := range names {
			if strings.HasPrefix(n, tempPrefix) {
				temps = append(temps, n)
			}
		}
		return temps
	}
	stale := tempFiles()
	if len(stale) != 1 {
		t.Fatalf("temp files after the killed run: %v, want exactly one", stale)
	}
	// A younger temp file, as a live archiver's copy in flight would leave.
	s.Clock = time.Now
	young := tempPrefix + "young"
	f, err := s.OpenFile(filepath.Join("/arch", young), fsx.O_RDWR|fsx.O_CREATE|fsx.O_EXCL, DefaultPerm)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatalf("the re-run: %v", err)
	}
	if got := tempFiles(); len(got) != 1 || got[0] != young {
		t.Fatalf("temp files after the re-run: %v, want only %q (the stale %v removed)", got, young, stale)
	}
	if ids := segmentFiles(t, s, "/arch"); len(ids) != 3 {
		t.Fatalf("segments after the re-run: %v", ids)
	}
}

// The destination's names are made durable before the _archive offset moves (decisions 40, 65
// and 85): the offset is what lets retention delete a sealed segment, so a power loss that
// kept the offset and lost the destination's rename would lose the segment from both places.
// Inject sees a rename's old path, the temp file, so renames are matched by their directory.
func TestArchiveSyncsTheDestinationDirectoryAfterItsMarkerAndBeforeCommittingTheOffset(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3)
	var ops []string
	s.Inject = func(op fsx.Op, name string) error {
		switch {
		case op == fsx.OpRename && filepath.Dir(name) == "/arch":
			ops = append(ops, "arch-rename")
		case op == fsx.OpSyncDir && name == "/arch":
			ops = append(ops, "arch-syncdir")
		case op == fsx.OpRename && filepath.Dir(name) == "/log/consumers":
			ops = append(ops, "offset-commit")
		}
		return nil
	}
	err := archive(s, "/log", "/arch")
	s.Inject = nil
	if err != nil {
		t.Fatal(err)
	}
	last := func(what string) int {
		for i := len(ops) - 1; i >= 0; i-- {
			if ops[i] == what {
				return i
			}
		}
		t.Fatalf("no %s in %v", what, ops)
		return -1
	}
	marker, sync, commit := last("arch-rename"), last("arch-syncdir"), last("offset-commit")
	if !(marker < sync && sync < commit) {
		t.Fatalf("the last destination rename, the last destination directory sync and the offset commit are at %d, %d, %d of %v, want them in that order", marker, sync, commit, ops)
	}
}

// A failed sync of the destination directory after the marker is written returns its error and
// leaves the _archive offset where it was, so the next run copies again from there.
func TestArchiveFailingTheFinalDestinationDirectorySyncReturnsItAndLeavesTheOffsetUncommitted(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3)
	offset := func() uint64 {
		t.Helper()
		list, err := l.Consumers()
		if err != nil || len(list) != 1 || list[0].Name != archiveConsumer {
			t.Fatalf("consumers %+v, %v", list, err)
		}
		return list[0].Offset
	}
	// archiveDest syncs /arch once before any copy, so the fault goes on the second sync.
	syncs := 0
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpSyncDir && name == "/arch" {
			if syncs++; syncs == 2 {
				return syscall.EIO
			}
		}
		return nil
	}
	err := archive(s, "/log", "/arch")
	s.Inject = nil
	if !errors.Is(err, syscall.EIO) {
		t.Fatalf("archive with a failing directory sync: %v", err)
	}
	if syncs != 2 {
		t.Fatalf("the destination directory was synced %d times, want the fault on the second and no later sync", syncs)
	}
	// The fault fell after the segments and the marker were written, not before them.
	if ids := segmentFiles(t, s, "/arch"); len(ids) != 3 {
		t.Fatalf("destination holds %v at the fault, want all 3 segments", ids)
	}
	if b := fileBytes(t, s, "/arch/"+tailName); len(b) == 0 {
		t.Fatal("the marker was not written before the fault")
	}
	if got := offset(); got != 1 {
		t.Fatalf("_archive offset %d after the failed sync, want 1 (uncommitted)", got)
	}
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if got := offset(); got != 13 {
		t.Fatalf("_archive offset %d after the rerun, want 13", got)
	}
	if ids := segmentFiles(t, s, "/arch"); len(ids) != 3 {
		t.Fatalf("after the rerun: %v", ids)
	}
}
