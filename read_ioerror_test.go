package quill

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// readEIOSetup returns a Sim log with several sealed segments and a non-empty active one, the
// segment file names oldest first, and the first sequence of the second segment.
func readEIOSetup(t *testing.T) (*fsx.Sim, *Log, []string, uint64) {
	t.Helper()
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	t.Cleanup(func() { _ = l.Close() })
	for len(segmentFiles(t, s, "/log")) < 3 {
		appendN(t, l, 4)
		mustFlush(t, l)
	}
	appendN(t, l, 2)
	mustFlush(t, l)
	var names []string
	for _, id := range segmentFiles(t, s, "/log") {
		names = append(names, filepath.Join("/log", segmentName(id)))
	}
	l.mu.Lock()
	mid := l.sealed[1].first
	l.mu.Unlock()
	return s, l, names, mid
}

func failReadsOf(s *fsx.Sim, name string) {
	s.Inject = func(op fsx.Op, n string) error {
		if op == fsx.OpRead && n == name {
			return syscall.EIO
		}
		return nil
	}
}

// wantReadEIO fails unless err is the I/O error itself: not a corruption, and not a gap that
// the resume idiom ("iterate again from To + 1") would skip (decisions 79 and 38).
func wantReadEIO(t *testing.T, what string, err error) {
	t.Helper()
	var gap *GapError
	switch {
	case err == nil:
		t.Fatalf("%s: no error", what)
	case errors.Is(err, ErrCorrupt):
		t.Fatalf("%s: an I/O error reported as corruption: %v", what, err)
	case errors.As(err, &gap):
		t.Fatalf("%s: an I/O error reported as the gap %v, which a resume would skip", what, gap)
	case !errors.Is(err, syscall.EIO):
		t.Fatalf("%s: want EIO, got %v", what, err)
	}
}

func TestAReadIOErrorInASealedSegmentIsReturnedAsThatError(t *testing.T) {
	s, l, names, mid := readEIOSetup(t)
	failReadsOf(s, names[1])
	_, err := l.Read(mid, 0)
	wantReadEIO(t, "Read", err)
	err = l.Scan(1, 0, func(Record) bool { return true })
	wantReadEIO(t, "Scan across the failing segment", err)
	s.Inject = nil
	if recs, err := l.Read(mid, 1); err != nil || len(recs) != 1 {
		t.Fatalf("the fault was transient, the next read must succeed: %v %d", err, len(recs))
	}
}

func TestAReadIOErrorInTheActiveSegmentIsReturnedAsThatError(t *testing.T) {
	s, l, names, _ := readEIOSetup(t)
	failReadsOf(s, names[len(names)-1])
	_, err := l.Read(l.Head(), 0)
	wantReadEIO(t, "Read", err)
	s.Inject = nil
	if recs, err := l.Read(l.Head(), 0); err != nil || len(recs) != 1 {
		t.Fatalf("the fault was transient, the next read must succeed: %v %d", err, len(recs))
	}
}

func TestAReaderReadIOErrorIsReturnedAsThatError(t *testing.T) {
	s, _, names, mid := readEIOSetup(t)
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	// One clean read scans and caches every sealed segment, so the next reads reach the walk.
	if _, err := r.Read(1, 0); err != nil {
		t.Fatal(err)
	}
	failReadsOf(s, names[1])
	_, err = r.Read(mid, 0)
	wantReadEIO(t, "Reader.Read of a sealed segment", err)
}

func TestAReaderIndexScanIOErrorIsReturnedAsThatError(t *testing.T) {
	s, l, names, mid := readEIOSetup(t)
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	// A read of the newest record caches every first frame but scans no sealed segment; the
	// first read of a sealed segment after that is its full scan.
	if _, err := r.Read(l.Head(), 1); err != nil {
		t.Fatal(err)
	}
	failReadsOf(s, names[1])
	_, err = r.Read(mid, 0)
	wantReadEIO(t, "Reader.Read, index scan", err)
	s.Inject = nil
	if recs, err := r.Read(mid, 1); err != nil || len(recs) != 1 {
		t.Fatalf("the fault was transient, the next read must succeed: %v %d", err, len(recs))
	}
}

func TestAnOffsetForTimeIOErrorIsReturnedAsThatError(t *testing.T) {
	s, l, names, _ := readEIOSetup(t)
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.Read(l.Head(), 1); err != nil { // caches the first frames, scans no sealed segment
		t.Fatal(err)
	}
	failReadsOf(s, names[0]) // a time before every record lands on the first segment, whose scan fails
	_, err = r.OffsetForTime(time.Unix(1, 0))
	wantReadEIO(t, "OffsetForTime", err)
}

func TestAReaderFirstFrameProbeIOErrorOfTheActiveSegmentIsReturnedAsThatError(t *testing.T) {
	s, l, names, _ := readEIOSetup(t)
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	failReadsOf(s, names[len(names)-1]) // the first read's probe of the active segment's first frame
	_, err = r.Read(l.Head(), 0)
	wantReadEIO(t, "Reader.Read, active first-frame probe", err)
	s.Inject = nil
	if recs, err := r.Read(l.Head(), 0); err != nil || len(recs) != 1 {
		t.Fatalf("the fault was transient, the next read must succeed: %v %d", err, len(recs))
	}
}

// A Reader reads a sealed segment's first frame to learn the segment's range. That is a read,
// not an open (decision 94 covers an open that fails): an I/O error comes back from the read
// that needs it, as that error, and is not remembered, so the next read looks again.
func TestAReaderFirstFrameProbeIOErrorOfASealedSegmentIsReturnedAsThatError(t *testing.T) {
	for _, tc := range []struct {
		name string
		seg  int // index into the segment files, oldest first
		read func(mid uint64) uint64
	}{
		{"the first segment", 0, func(uint64) uint64 { return 1 }},
		{"a middle segment", 1, func(mid uint64) uint64 { return mid }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, names, mid := readEIOSetup(t)
			r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			seq := tc.read(mid)
			failReadsOf(s, names[tc.seg]) // the first read's probe of that segment's first frame
			_, err = r.Read(seq, 1)
			wantReadEIO(t, "Reader.Read, sealed first-frame probe", err)
			// Still failing: still that error, never a gap cached by the first failure.
			_, err = r.Read(seq, 1)
			wantReadEIO(t, "Reader.Read again, sealed first-frame probe", err)
			s.Inject = nil
			if recs, err := r.Read(seq, 1); err != nil || len(recs) != 1 || recs[0].Seq != seq {
				t.Fatalf("the fault was transient, the next read must succeed: %v %v", err, recs)
			}
		})
	}
}

// A sealed segment that Open cannot verify for an I/O reason is not a hole: Open returns the
// error, as it does for the newest segment, so no read ever sees a gap that one transient
// fault made and that a resume from To + 1 would skip (decisions 38, 79).
func TestAnOpenThatMeetsAReadIOErrorVerifyingASealedSegmentFailsWithThatError(t *testing.T) {
	for _, seg := range []int{0, 1} {
		t.Run(fmt.Sprintf("segment %d", seg), func(t *testing.T) {
			s, l, names, mid := readEIOSetup(t)
			want := uint64(1) // the first record of the segment that fails
			if seg == 1 {
				want = mid
			}
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			failReadsOf(s, names[seg])
			if l2, err := Open(simConfig(s, "/log")); err == nil {
				var gap *GapError
				_, rerr := l2.Read(want, 1)
				_ = l2.Close()
				t.Fatalf("segment %d: Open succeeded with a sealed segment unverified; Read gave %v (gap %v)", seg, rerr, errors.As(rerr, &gap))
			} else {
				wantReadEIO(t, "Open", err)
			}
			s.Inject = nil
			l2, err := Open(simConfig(s, "/log"))
			if err != nil {
				t.Fatalf("the fault was transient, Open must succeed: %v", err)
			}
			if recs, err := l2.Read(want, 1); err != nil || len(recs) != 1 || recs[0].Seq != want {
				t.Fatalf("segment %d: Read after the fault cleared: %v %v", seg, err, recs)
			}
			if l2.Err() != nil {
				t.Fatalf("segment %d: Err() after the fault cleared: %v", seg, l2.Err())
			}
			_ = l2.Close()
		})
	}
}
