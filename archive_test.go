package quill

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// T15: archive.

func TestArchiveCopiesSealedSegmentsVerifiedIdempotentlyAndTheDestinationReadsTheSame(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 5)
	appendN(t, l, 1) // in the active segment: not archived
	mustFlush(t, l)
	if err := archive(s, "/log", "/arch", 0o600); err != nil {
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
	if err := archive(s, "/log", "/arch", 0o600); err != nil {
		t.Fatal(err)
	}
	if ops := s.Ops() - before; ops > 60 {
		t.Fatalf("a no-op archive did %d operations", ops)
	}
	// After a Roll the newest records are archived too, and the destination still reads.
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if err := archive(s, "/log", "/arch", 0o600); err != nil {
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
	if err := archive(s, "/other", "/arch", 0o600); !errors.Is(err, ErrWrongLog) {
		t.Fatalf("%v", err)
	}
}

func TestArchiveStopsAtACorruptSegmentAndReportsASourceHole(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 6)
	seg2 := "/log/" + segmentName(2)
	data := fileBytes(t, s, seg2)
	data[len(data)-2] ^= 1
	writeFile(t, s, seg2, data)
	err := archive(s, "/log", "/arch", 0o600)
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
	if err := archive(s, "/log", "/arch", 0o600); err != nil {
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
	err = archive(s2, "/log", "/arch", 0o600)
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
	if err := archive(s2, "/log", "/arch", 0o600); err != nil {
		t.Fatalf("a second run after the hole: %v", err)
	}
	// Retention lost the first segments before the archiver ran: a gap below the offset.
	s3 := fsx.NewSim()
	l3 := openRetention(t, s3, clk, nil)
	defer l3.Close()
	fillSegments(t, l3, clk, 3)
	if err := archive(s3, "/log", "/arch", 0o600); err != nil {
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
	if err := archive(s3, "/log", "/arch", 0o600); !isGap(err, 13, 20) {
		t.Fatalf("%v", err)
	}
}

func TestTheArchiverIsAProtectedConsumerUntilACapOverridesIt(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, nil)
	defer l.Close()
	fillSegments(t, l, clk, 4)
	if err := archive(s, "/log", "/arch", 0o600); err != nil {
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
