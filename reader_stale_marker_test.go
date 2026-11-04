package quill

import (
	"errors"
	"io/fs"
	"sync"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// overtakenMarkerLog builds a log with sealed segments and an active one holding a record the marker
// covers, and a reader on it, and arranges that the writer, as the reader lists the
// directory (after it read the marker, before it opens the segment the marker names), appends
// enough records to rotate twice with MaxBytes deleting the segment the reader's marker names.
func overtakenMarkerLog(t *testing.T) (*fsx.Sim, *Reader, *bool, string) {
	t.Helper()
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1; c.MaxBytes = 150 })
	t.Cleanup(func() { _ = l.Close() })
	fillSegments(t, l, clk, 2)
	appendN(t, l, 1)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	t.Cleanup(func() { _ = r.Close() })
	ids := segmentFiles(t, s, "/log")
	active := "/log/" + segmentName(ids[len(ids)-1])

	var mu sync.Mutex
	fired := false
	s.Inject = func(op fsx.Op, name string) error {
		if op != fsx.OpReadDir || name != "/log" {
			return nil
		}
		mu.Lock()
		if fired {
			mu.Unlock()
			return nil
		}
		fired = true
		mu.Unlock()
		fillSegments(t, l, clk, 3) // seals the segment the marker names; the byte cap deletes it
		return nil
	}
	return s, r, &fired, active
}

// A marker overtaken between its read and the open of its segment names a segment that retention
// has deleted since: a live log does not hold still. The view starts over with the newer marker;
// the reader must not report the path error (guarantee 14: a deleted segment is a gap).
func TestAReaderViewOvertakenByARotationAndADeletionStartsOver(t *testing.T) {
	s, r, fired, active := overtakenMarkerLog(t)
	_, err := r.Read(9, 10) // record 9 is the active segment's first
	if !*fired {
		t.Fatal("the writer did not move on while the reader listed the directory")
	}
	if _, serr := s.Lstat(active); !errors.Is(serr, fs.ErrNotExist) {
		t.Fatalf("the segment the old marker names is still there (%v): the test does not exercise the race", serr)
	}
	var gap *GapError
	if err != nil && !errors.As(err, &gap) {
		t.Fatalf("Read: %v", err)
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a path error escaped: %v", err)
	}
}
