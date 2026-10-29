package quill

import (
	"bytes"
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

var errENOSPC = syscall.ENOSPC

// T2: recovery classification at the log level.

func TestAStaleMarkerWithAcknowledgedRecordsAboveItKeepsAllOfThem(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 5)
	mustFlush(t, l)
	stale := fileBytes(t, s, "/log/"+tailName)
	appendN(t, l, 5)
	mustFlush(t, l)
	_ = l.Close()
	writeFile(t, s, "/log/"+tailName, stale)
	l = openSim(t, s, nil)
	defer l.Close()
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 10 || l.Head() != 10 {
		t.Fatalf("%v %v head %d", seqsOf(recs), err, l.Head())
	}
	if tl := readTail(t, s, "/log"); tl.Seq != 10 {
		t.Fatalf("marker not republished: %+v", tl)
	}
}

func TestAZeroedPageFollowedByIntactFramesIsTruncatedAtTheHole(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	appendN(t, l, 3)
	mustFlush(t, l)
	_ = l.Close()
	seg := "/log/" + segmentName(0)
	data := fileBytes(t, s, seg)
	later := image(4, 3, 0, func(i int) []byte { return payloadFor(uint64(4 + i)) })
	writeFile(t, s, seg, append(append(data, make([]byte, 4096)...), later...))
	l = openSim(t, s, nil)
	defer l.Close()
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 3 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	if got := fileBytes(t, s, seg); !bytes.Equal(got, data) {
		t.Fatalf("file not truncated at the hole: %d bytes, want %d", len(got), len(data))
	}
	seq, _ := l.Append(payloadFor(4))
	if seq != 4 {
		t.Fatalf("next sequence %d", seq)
	}
}

func TestDamageBelowTheMarkerFailsOpenAndLeavesTheFileUnchanged(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 5)
	mustFlush(t, l)
	_ = l.Close()
	seg := "/log/" + segmentName(0)
	data := fileBytes(t, s, seg)
	damaged := bytes.Clone(data)
	damaged[len(data)/2] ^= 0xff
	writeFile(t, s, seg, damaged)
	_, err := Open(simConfig(s, "/log"))
	var se *SegmentError
	if !errors.Is(err, ErrCorrupt) || !errors.As(err, &se) || se.Path != seg || se.Intact == 0 {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(fileBytes(t, s, seg), damaged) {
		t.Fatal("the file was modified")
	}
	// Damage in a sealed segment: Open succeeds, the segment is a hole, the file is unchanged.
	// Opening with a smaller SegmentBytes seals segment 0 (1..5) at once; 6..10 and 11..12
	// land in segments 1 and 2.
	writeFile(t, s, seg, data)
	l = openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	appendN(t, l, 5)
	mustFlush(t, l)
	appendN(t, l, 2)
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); len(ids) != 3 {
		t.Fatalf("segments %v", ids)
	}
	_ = l.Close()
	sealedData := fileBytes(t, s, seg)
	damaged = bytes.Clone(sealedData)
	damaged[10] ^= 0x01
	writeFile(t, s, seg, damaged)
	l = openSim(t, s, nil)
	defer l.Close()
	if err := l.Err(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Err() %v", err)
	}
	if !bytes.Equal(fileBytes(t, s, seg), damaged) {
		t.Fatal("the sealed file was modified")
	}
	if l.holes[0] != (Hole{From: 1, To: 5}) {
		t.Fatalf("holes %+v", l.holes)
	}
	if _, err := l.Read(1, 10); !isGap(err, 1, 5) {
		t.Fatalf("read into the hole: %v", err)
	}
	if _, err := l.Read(3, 10); !isGap(err, 3, 5) {
		t.Fatalf("read inside the hole: %v", err)
	}
	recs, err := l.Read(6, 10)
	if err != nil || checkContiguous(t, recs, 6) != 12 {
		t.Fatalf("after the hole: %v %v", seqsOf(recs), err)
	}
	if recs, err := readAll(l); err != nil || checkContiguous(t, recs, 6) != 12 {
		t.Fatalf("ScanAll starts at the first retained sequence: %v %v", seqsOf(recs), err)
	}
	if err := l.Scan(1, 0, func(Record) bool { return true }); !isGap(err, 1, 5) {
		t.Fatalf("Scan from 1: %v", err)
	}
	if l.Head() != 12 {
		t.Fatalf("head %d", l.Head())
	}
	// A sealed segment with no frames is corruption too.
	writeFile(t, s, seg, nil)
	_ = l.Close()
	l = openSim(t, s, nil)
	if err := l.Err(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("empty sealed segment: %v", err)
	}
}

func TestAMarkerAboveTheLastValidFrameFailsOpenUnlessTheLogHoldsNoFrames(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 5)
	mustFlush(t, l)
	_ = l.Close()
	writeFile(t, s, "/log/"+tailName, encodeTail(tail{Seq: 9, Seg: 0, Off: 0, MaxSeg: 0}))
	if _, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("marker above the frames: %v", err)
	}
	// Rolled, everything expired: an empty active segment and a marker with a sequence.
	writeFile(t, s, "/log/"+segmentName(3), nil)
	writeFile(t, s, "/log/"+tailName, encodeTail(tail{Seq: 9, Seg: 3, Off: 0, MaxSeg: 3}))
	if err := s.Remove("/log/" + segmentName(0)); err != nil {
		t.Fatal(err)
	}
	l = openSim(t, s, nil)
	defer l.Close()
	if l.Head() != 9 {
		t.Fatalf("head %d", l.Head())
	}
	seq, err := l.Append(payloadFor(10))
	if err != nil || seq != 10 {
		t.Fatalf("next %d %v", seq, err)
	}
	mustFlush(t, l)
	if tl := readTail(t, s, "/log"); tl.Seq != 10 || tl.Seg != 3 || tl.MaxSeg != 3 {
		t.Fatalf("%+v", tl)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("segments %v", ids)
	}
}

func TestTornTailsAtEveryPositionAboveTheMarkerAreCutAndBelowItAreCorruption(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	appendN(t, l, 4)
	mustFlush(t, l)
	_ = l.Close()
	seg := "/log/" + segmentName(0)
	durable := fileBytes(t, s, seg)
	unacked := image(5, 4, 0, func(i int) []byte { return payloadFor(uint64(5 + i)) })
	full := append(bytes.Clone(durable), unacked...)
	ends := frameEnds(t, full)
	for cut := 0; cut <= len(full); cut++ {
		writeFile(t, s, seg, full[:cut])
		l, err := Open(simConfig(s, "/log"))
		if cut < len(durable) {
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("cut at %d below the marker: %v", cut, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("cut at %d: %v", cut, err)
		}
		want := int64(len(durable))
		for _, e := range ends {
			if e <= int64(cut) {
				want = max(want, e)
			}
		}
		if got := fileBytes(t, s, seg); int64(len(got)) != want {
			t.Fatalf("cut at %d: file is %d bytes, want %d", cut, len(got), want)
		}
		recs, err := readAll(l)
		if err != nil {
			t.Fatal(err)
		}
		last := checkContiguous(t, recs, 1)
		if last < 4 || int64(last) != 4+int64(countLE(ends, int64(cut)))-4 {
			t.Fatalf("cut at %d: last %d", cut, last)
		}
		_ = l.Close()
	}
}

func countLE(ends []int64, x int64) int {
	n := 0
	for _, e := range ends {
		if e <= x {
			n++
		}
	}
	return n
}

func TestAMarkerNamingASealedSegmentAndAnEmptyNewestSegmentRecover(t *testing.T) {
	// A crash between rotation's create and its publish leaves a marker naming segment 0 while
	// segment 1 exists and is empty (decision 43).
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 3)
	mustFlush(t, l)
	_ = l.Close()
	marker := fileBytes(t, s, "/log/"+tailName)
	writeFile(t, s, "/log/"+segmentName(1), nil)
	writeFile(t, s, "/log/"+tailName, marker)
	l = openSim(t, s, nil)
	defer l.Close()
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 3 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	if tl := readTail(t, s, "/log"); tl.Seg != 1 || tl.Off != 0 || tl.Seq != 3 || tl.MaxSeg != 1 {
		t.Fatalf("%+v", tl)
	}
	seq, _ := l.Append(payloadFor(4))
	mustFlush(t, l)
	if seq != 4 || len(fileBytes(t, s, "/log/"+segmentName(1))) == 0 {
		t.Fatalf("seq %d", seq)
	}
	// With frames in the newest segment too, written after the stale marker.
	_ = l.Close()
	writeFile(t, s, "/log/"+tailName, marker)
	l = openSim(t, s, nil)
	recs, err = readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 4 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
}

func TestADamagedPredecessorSegmentIsAHole(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	appendN(t, l, 5)
	mustFlush(t, l)
	appendN(t, l, 5)
	mustFlush(t, l)
	appendN(t, l, 2)
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); len(ids) != 3 {
		t.Fatalf("segments %v", ids)
	}
	_ = l.Close()
	// Segment 1 (6..10) is the newest segment's predecessor.
	seg := "/log/" + segmentName(1)
	damaged := bytes.Clone(fileBytes(t, s, seg))
	damaged[10] ^= 0x01
	writeFile(t, s, seg, damaged)
	l, err := Open(simConfig(s, "/log"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer l.Close()
	if err := l.Err(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Err() %v", err)
	}
	if !bytes.Equal(fileBytes(t, s, seg), damaged) {
		t.Fatal("the sealed file was modified")
	}
	if len(l.holes) != 1 || l.holes[0] != (Hole{From: 6, To: 10}) {
		t.Fatalf("holes %+v", l.holes)
	}
	if _, err := l.Read(6, 10); !isGap(err, 6, 10) {
		t.Fatalf("read into the hole: %v", err)
	}
}

func TestAMiddleSegmentDeletedByHandIsAHoleFoundAtLoad(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
	for i := 0; i < 4; i++ {
		appendN(t, l, 4)
		mustFlush(t, l)
	}
	_ = l.Close()
	if err := s.Remove("/log/" + segmentName(1)); err != nil {
		t.Fatal(err)
	}
	l = openSim(t, s, nil)
	defer l.Close()
	if len(l.holes) != 1 || l.holes[0] != (Hole{From: 5, To: 8}) {
		t.Fatalf("holes %+v", l.holes)
	}
	if _, err := l.Read(6, 10); !isGap(err, 6, 8) {
		t.Fatalf("%v", err)
	}
	recs, err := l.Read(1, 100)
	if err != nil || checkContiguous(t, recs, 1) != 4 {
		t.Fatalf("before the hole: %v %v", seqsOf(recs), err)
	}
	recs, err = l.Read(9, 100)
	if err != nil || checkContiguous(t, recs, 9) != 16 {
		t.Fatalf("after the hole: %v %v", seqsOf(recs), err)
	}
	// An overlapping copy of a segment is corruption, not a hole.
	writeFile(t, s, "/log/"+segmentName(1), fileBytes(t, s, "/log/"+segmentName(2)))
	_ = l.Close()
	if _, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("overlap: %v", err)
	}
}

// T3: failed writes.

func TestAShortWriteOrENOSPCTruncatesBackAndRetriesFromTheDurableOffset(t *testing.T) {
	for _, inject := range []error{&fsx.ShortWrite{N: 7}, errENOSPC} {
		s := fsx.NewSim()
		l := openSim(t, s, nil)
		appendN(t, l, 3)
		mustFlush(t, l)
		seg := "/log/" + segmentName(0)
		before := fileBytes(t, s, seg)
		appendN(t, l, 3)
		s.Inject = func(op fsx.Op, name string) error {
			if op == fsx.OpWrite && name == seg {
				return inject
			}
			return nil
		}
		if err := l.Flush(); err == nil {
			t.Fatalf("%v: flush succeeded", inject)
		}
		if !bytes.Equal(fileBytes(t, s, seg), before) {
			t.Fatalf("%v: a partial frame remains", inject)
		}
		if l.Head() != 3 {
			t.Fatalf("%v: head %d", inject, l.Head())
		}
		s.Inject = nil
		mustFlush(t, l)
		recs, err := readAll(l)
		if err != nil || checkContiguous(t, recs, 1) != 6 {
			t.Fatalf("%v: %v %v", inject, seqsOf(recs), err)
		}
		if _, _, err := recover_(fileBytes(t, s, seg), 0); err != nil {
			t.Fatal(err)
		}
		_ = l.Close()
	}
}

func TestAnFsyncErrorIsFatalAndAReopenRecoversAConsistentLog(t *testing.T) {
	for _, op := range []fsx.Op{fsx.OpSync, fsx.OpSyncDir} {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
		appendN(t, l, 3)
		mustFlush(t, l)
		appendN(t, l, 3) // enough to rotate at this flush, so the directory fsync runs
		s.Inject = func(o fsx.Op, name string) error {
			if o == op {
				return syscall.EIO
			}
			return nil
		}
		err := l.Flush()
		if !errors.Is(err, ErrFailed) || !errors.Is(err, syscall.EIO) {
			t.Fatalf("%s: flush: %v", op, err)
		}
		s.Inject = nil
		if _, err := l.Append(nil); !errors.Is(err, ErrFailed) {
			t.Fatalf("%s: append: %v", op, err)
		}
		if err := l.Flush(); !errors.Is(err, ErrFailed) {
			t.Fatalf("%s: flush again: %v", op, err)
		}
		// After a data fsync failure nothing became durable: Sync fails. After a directory
		// fsync failure at rotation the data had been fsynced: Sync of it returns nil.
		if err := l.Sync(context_background(), 6); (op == fsx.OpSync) != errors.Is(err, ErrFailed) {
			t.Fatalf("%s: sync: %v", op, err)
		}
		if err := l.Sync(context_background(), 7); !errors.Is(err, ErrBadSeq) {
			t.Fatalf("%s: sync of a refused append: %v", op, err)
		}
		if err := l.Err(); !errors.Is(err, ErrFailed) {
			t.Fatalf("%s: Err: %v", op, err)
		}
		_ = l.Close()
		l = openSim(t, s, nil)
		recs, err := readAll(l)
		if err != nil {
			t.Fatal(err)
		}
		last := checkContiguous(t, recs, 1)
		if last < 3 || last > 6 {
			t.Fatalf("%s: last %d", op, last)
		}
		_ = l.Close()
	}
}

func TestAMarkerFailureAfterTheDataFsyncNeverRewritesDataAndTheNextFlushRepublishes(t *testing.T) {
	for _, op := range []fsx.Op{fsx.OpCreate, fsx.OpRename} {
		s := fsx.NewSim()
		l := openSim(t, s, nil)
		appendN(t, l, 3)
		mustFlush(t, l)
		seg := "/log/" + segmentName(0)
		appendN(t, l, 3)
		s.Inject = func(o fsx.Op, name string) error {
			if o == op && name != seg {
				return errENOSPC
			}
			return nil
		}
		err := l.Flush()
		if !errors.Is(err, errENOSPC) {
			t.Fatalf("%s: flush: %v", op, err)
		}
		after := fileBytes(t, s, seg)
		if l.Head() != 6 {
			t.Fatalf("%s: the data is durable, head should be 6, is %d", op, l.Head())
		}
		if err := l.Sync(context_background(), 6); err != nil {
			t.Fatalf("%s: sync of durable data: %v", op, err)
		}
		if tl := readTail(t, s, "/log"); tl.Seq != 3 {
			t.Fatalf("%s: marker advanced to %d", op, tl.Seq)
		}
		if err := l.Err(); !errors.Is(err, errENOSPC) {
			t.Fatalf("%s: Err: %v", op, err)
		}
		appendN(t, l, 1)
		s.Inject = nil
		mustFlush(t, l)
		if tl := readTail(t, s, "/log"); tl.Seq != 7 {
			t.Fatalf("%s: marker not republished: %+v", op, tl)
		}
		if got := fileBytes(t, s, seg); !bytes.Equal(got[:len(after)], after) {
			t.Fatalf("%s: durable data was rewritten", op)
		}
		recs, err := readAll(l)
		if err != nil || checkContiguous(t, recs, 1) != 7 {
			t.Fatalf("%s: %v %v", op, seqsOf(recs), err)
		}
		_ = l.Close()
	}
}

func TestAFailureCreatingTheNextSegmentKeepsAppendingToTheCurrentOne(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	defer l.Close()
	appendN(t, l, 8)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpCreate && name == "/log/"+segmentName(1) {
			return errENOSPC
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, errENOSPC) {
		t.Fatalf("flush: %v", err)
	}
	if tl := readTail(t, s, "/log"); tl.Seq != 8 || tl.Seg != 0 {
		t.Fatalf("the marker should still be published for the current segment: %+v", tl)
	}
	appendN(t, l, 8)
	if err := l.Flush(); !errors.Is(err, errENOSPC) {
		t.Fatalf("second flush: %v", err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("segments %v", ids)
	}
	if size := len(fileBytes(t, s, "/log/"+segmentName(0))); size <= 200 {
		t.Fatalf("segment 0 is %d bytes", size)
	}
	if tl := readTail(t, s, "/log"); tl.Seq != 16 || tl.Seg != 0 {
		t.Fatalf("marker %+v", tl)
	}
	s.Inject = nil
	appendN(t, l, 1)
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("segments after recovery %v", ids)
	}
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 17 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
}

func TestRollDoesNothingOnAnEmptySegmentAndSealsOtherwise(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	// The marker naming a new segment is durable at rotation (decision 65): a crash right
	// after Roll keeps it.
	appendN(t, l, 1)
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	data, err := s.Durable("/log/" + tailName)
	if err != nil {
		t.Fatal(err)
	}
	if tl, err := decodeTail(data); err != nil || tl.Seg != 1 || tl.Seq != 1 {
		t.Fatalf("durable marker after rotation: %+v %v", tl, err)
	}
	_ = l.Close()
	s.Crash(nil)
	l = openSim(t, s, nil)
	defer l.Close()
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 || readTail(t, s, "/log").Seg != 1 {
		t.Fatalf("after a crash: %v %+v", ids, readTail(t, s, "/log"))
	}
	_ = l.Close()
	s = fsx.NewSim()
	l = openSim(t, s, nil)
	defer l.Close()
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("roll on an empty log made %v", ids)
	}
	appendN(t, l, 2)
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("segments %v", ids)
	}
	if tl := readTail(t, s, "/log"); tl.Seq != 2 || tl.Seg != 1 || tl.Off != 0 || tl.MaxSeg != 1 {
		t.Fatalf("%+v", tl)
	}
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("a second roll on an empty segment made %v", ids)
	}
}

func TestAFailedOpenLeaksNoDescriptor(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 3)
	if err := l.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if n := s.OpenFiles(); n != 0 {
		t.Fatalf("open handles after Close: %d", n)
	}
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpSyncDir {
			return syscall.EIO
		}
		return nil
	}
	for i := 1; i <= 3; i++ {
		if _, err := Open(simConfig(s, "/log")); !errors.Is(err, syscall.EIO) {
			t.Fatalf("attempt %d: want EIO, got %v", i, err)
		}
		if n := s.OpenFiles(); n != 0 {
			t.Fatalf("attempt %d: %d handles left open after a failed Open", i, n)
		}
	}
}

// The manual procedure in docs/userguide.md section 10: truncating the newest segment below
// quill.tail.off is refused until quill.tail is rewritten to the truncation point and the last
// valid sequence, after which Open succeeds and the next sequence follows on.
func TestManualProcedureNeedsTheMarkerRewrittenAfterTruncatingTheNewestSegment(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 4)
	mustFlush(t, l)
	seg := "/log/" + segmentName(0)
	end := len(fileBytes(t, s, seg)) // the end of the last valid frame
	appendN(t, l, 1)
	mustFlush(t, l)
	_ = l.Close()
	damaged := fileBytes(t, s, seg)
	damaged[end] ^= 0xff // a byte of the fifth frame: damage below the marker
	writeFile(t, s, seg, damaged[:end])
	_, err := Open(simConfig(s, "/log"))
	var se *SegmentError
	if !errors.Is(err, ErrCorrupt) || !errors.As(err, &se) || se.Path != seg {
		t.Fatalf("open after truncating only: %v, want ErrCorrupt naming %s", err, seg)
	}
	tl := readTail(t, s, "/log")
	writeFile(t, s, "/log/"+tailName, []byte(fmt.Sprintf("{\"seq\":%d,\"seg\":%d,\"off\":%d,\"maxseg\":%d}\n", 4, tl.Seg, end, tl.MaxSeg)))
	l = openSim(t, s, nil)
	defer l.Close()
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 4 || l.Head() != 4 {
		t.Fatalf("%v %v head %d", seqsOf(recs), err, l.Head())
	}
	if seq, _ := l.Append(payloadFor(5)); seq != 5 {
		t.Fatalf("next sequence %d", seq)
	}
}
