package quill

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Err() reports what is failing now (Stage 2 audit A5-10): a background error is cleared when
// the operation that set it succeeds, and a sticky load error no longer hides it.

func wantErrNil(t *testing.T, l *Log, when string) {
	t.Helper()
	if err := l.Err(); err != nil {
		t.Fatalf("%s: Err() is %v, want nil", when, err)
	}
}

func TestErrClearsWhenARotationThatFailedSucceeds(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	defer l.Close()
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpCreate && name == "/log/"+segmentName(1) {
			return errENOSPC
		}
		return nil
	}
	appendN(t, l, 8)
	if err := l.Flush(); !errors.Is(err, errENOSPC) {
		t.Fatalf("flush with the blocker: %v", err)
	}
	if err := l.Err(); !errors.Is(err, errENOSPC) {
		t.Fatalf("Err() while rotation fails: %v", err)
	}
	s.Inject = nil
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("segments after the blocker went: %v", ids)
	}
	wantErrNil(t, l, "after the rotation succeeded")
}

func TestErrClearsWhenAFailedRollSucceeds(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpCreate && name == "/log/"+segmentName(1) {
			return errENOSPC
		}
		return nil
	}
	if err := l.Roll(); !errors.Is(err, errENOSPC) {
		t.Fatalf("roll with the blocker: %v", err)
	}
	if err := l.Err(); !errors.Is(err, errENOSPC) {
		t.Fatalf("Err() after the failed roll: %v", err)
	}
	s.Inject = nil
	if err := l.Roll(); err != nil {
		t.Fatalf("roll after the blocker went: %v", err)
	}
	wantErrNil(t, l, "after the roll succeeded")
}

func TestADamagedSealedSegmentDoesNotHideALiveRotationFailure(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	appendN(t, l, 8)
	mustFlush(t, l)
	appendN(t, l, 8)
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); len(ids) < 3 {
		t.Fatalf("setup: segments %v", ids)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	seg := "/log/" + segmentName(0)
	data := fileBytes(t, s, seg)
	data[10] ^= 0x01
	writeFile(t, s, seg, data)

	l = openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	defer l.Close()
	if err := l.Err(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Err() after Open: %v", err)
	}
	next := segmentName(l.maxSeg + 1)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpCreate && name == "/log/"+next {
			return errENOSPC
		}
		return nil
	}
	appendN(t, l, 8)
	if err := l.Flush(); !errors.Is(err, errENOSPC) {
		t.Fatalf("flush with the blocker: %v", err)
	}
	err := l.Err()
	if !errors.Is(err, ErrCorrupt) || !errors.Is(err, errENOSPC) {
		t.Fatalf("Err() must report both the damage and the live rotation failure: %v", err)
	}
	s.Inject = nil
	mustFlush(t, l)
	err = l.Err()
	if !errors.Is(err, ErrCorrupt) || errors.Is(err, errENOSPC) {
		t.Fatalf("after rotation recovered Err() must name the damage only: %v", err)
	}
}

func TestErrClearsWhenAFailedFlushWriteSucceeds(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	appendN(t, l, 3)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpWrite && name == "/log/"+segmentName(0) {
			return syscall.EIO
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, syscall.EIO) {
		t.Fatalf("flush: %v", err)
	}
	if err := l.Err(); !errors.Is(err, syscall.EIO) {
		t.Fatalf("Err() while the write fails: %v", err)
	}
	s.Inject = nil
	mustFlush(t, l)
	wantErrNil(t, l, "after the write succeeded")
}

func TestErrClearsWhenAFailedMarkerPublishSucceeds(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	appendN(t, l, 3)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpRename {
			return errENOSPC
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, errENOSPC) {
		t.Fatalf("flush: %v", err)
	}
	if err := l.Err(); !errors.Is(err, errENOSPC) {
		t.Fatalf("Err() while the publish fails: %v", err)
	}
	s.Inject = nil
	mustFlush(t, l)
	wantErrNil(t, l, "after the marker was published")
}

func TestErrClearsWhenAFailedPruneUnlinkSucceeds(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 4)
	clk.Advance(2 * time.Hour)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpRemove && name == "/log/"+segmentName(0) {
			return syscall.EIO
		}
		return nil
	}
	if err := l.Prune(true); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if err := l.Err(); !errors.Is(err, syscall.EIO) {
		t.Fatalf("Err() after the failed unlink: %v", err)
	}
	s.Inject = nil
	if err := l.Prune(true); err != nil {
		t.Fatalf("prune: %v", err)
	}
	wantErrNil(t, l, "after a clean prune pass")
}

func TestTheLoadErrorIsDroppedOnceItsSegmentAndHoleArePruned(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	fillSegments(t, l, clk, 3)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	seg := "/log/" + segmentName(0)
	data := fileBytes(t, s, seg)
	data[len(data)-1] ^= 0xff
	writeFile(t, s, seg, data)

	l = openRetention(t, s, clk, func(c *Config) { c.MaxRecords = 4 })
	defer l.Close()
	if err := l.Err(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Err() after Open: %v", err)
	}
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if st, err := l.Stat(); err != nil || len(st.Holes) != 0 {
		t.Fatalf("setup: holes %+v %v", st.Holes, err)
	}
	wantErrNil(t, l, "after the damaged segment was deleted and its hole pruned")
}
