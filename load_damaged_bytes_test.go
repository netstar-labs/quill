package quill

import (
	"errors"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A damaged sealed segment still occupies its bytes (decision 98): Stat().Bytes is the sum of
// the segment files' sizes and MaxBytes counts it, so the oldest one can be deleted by the cap.

// damagedLog builds three sealed segments of four records (segments 0..2), closes the log
// and flips the last byte of segment 0 (a frame's CRC), so it fails verification.
func damagedLog(t *testing.T, s *fsx.Sim, clk *testClock) {
	t.Helper()
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = -1 })
	fillSegments(t, l, clk, 3)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	path := "/log/" + segmentName(0)
	data := fileBytes(t, s, path)
	data[len(data)-1] ^= 0xff
	writeFile(t, s, path, data)
}

func filesTotal(t *testing.T, s *fsx.Sim) int64 {
	t.Helper()
	var sum int64
	for _, id := range segmentFiles(t, s, "/log") {
		sum += int64(len(fileBytes(t, s, "/log/"+segmentName(id))))
	}
	return sum
}

func TestADamagedSealedSegmentCountsItsBytesInStatAndInMaxBytes(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	damagedLog(t, s, clk)
	total := filesTotal(t, s)
	seg0 := int64(len(fileBytes(t, s, "/log/"+segmentName(0))))

	l := openRetention(t, s, clk, func(c *Config) {
		c.SegmentMaxAge = -1
		c.MaxBytes = total - 1 // over by one byte: the oldest segment must go, and only it
	})
	defer l.Close()
	if sealed := sealedIDs(l); len(sealed) != 3 || l.sealed[0].bad == nil {
		t.Fatalf("setup: sealed %v, first bad: %v", sealed, l.sealed[0].bad)
	}
	st, err := l.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if st.Bytes != total {
		t.Fatalf("Stat().Bytes %d, the files hold %d", st.Bytes, total)
	}
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if rst, err := r.Stat(); err != nil || rst.Bytes != st.Bytes {
		t.Fatalf("Reader.Stat().Bytes %d (%v), Log %d", rst.Bytes, err, st.Bytes)
	}

	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := sealedIDs(l); len(ids) != 2 || ids[0] != 1 {
		t.Fatalf("after the MaxBytes pass the sealed segments are %v, want [1 2]", ids)
	}
	if got := filesTotal(t, s); got != total-seg0 {
		t.Fatalf("files hold %d bytes after the pass, want %d", got, total-seg0)
	}
	if st, _ := l.Stat(); st.Bytes != total-seg0 {
		t.Fatalf("Stat().Bytes %d after the pass, want %d", st.Bytes, total-seg0)
	}
}

func TestASealedSegmentThatCannotBeOpenedStillCountsItsFileSize(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	damagedLog(t, s, clk)
	total := filesTotal(t, s)
	seg0 := "/log/" + segmentName(0)
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpOpen && name == seg0 {
			return errors.New("injected: too many open files")
		}
		return nil
	}
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = -1 })
	s.Inject = nil
	defer l.Close()
	if l.sealed[0].bad == nil {
		t.Fatal("setup: segment 0 loaded")
	}
	if st, err := l.Stat(); err != nil || st.Bytes != total {
		t.Fatalf("Stat().Bytes %d (%v), the files hold %d", st.Bytes, err, total)
	}
}
