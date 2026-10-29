package quill

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
