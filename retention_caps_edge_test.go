package quill

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// The byte and free-space caps are checked at their equality edges and together with the age
// sweep (Stage 2 audit L2-27). A cap deletes only what it needs: a set of bytes exactly at
// MaxBytes, or free space exactly at MinFreeBytes, meets the cap and loses nothing, and a
// segment the age sweep already chose counts as gone, not as bytes the cap must still shed.

// capsFixture closes a log of three sealed segments (and an empty active one) with no caps
// set, and returns the simulator, the clock, the size of segment 0 and of the whole log.
func capsFixture(t *testing.T) (s *fsx.Sim, clk *testClock, size0, total int64) {
	t.Helper()
	s, clk, l := retentionLog(t, nil)
	fillSegments(t, l, clk, 3) // sealed at minutes 0, 1 and 2
	st, err := l.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	size0 = int64(len(fileBytes(t, s, "/log/"+segmentName(0))))
	return s, clk, size0, st.Bytes
}

func TestMaxBytesCapCountsOnlyWhatTheAgeSweepLeaves(t *testing.T) {
	s, clk, size0, total := capsFixture(t)
	// A cap that segments 1 and 2 and the active one meet exactly: once the age sweep has
	// chosen segment 0 the caps hold, and nothing else may go.
	l := openRetention(t, s, clk, func(c *Config) { c.MaxBytes = total - size0 })
	defer l.Close()
	if l.cfg.MaxBytes < 2*l.cfg.SegmentBytes {
		t.Fatalf("fixture: MaxBytes %d is under twice SegmentBytes", l.cfg.MaxBytes)
	}
	// The clock is at minute 3 after the fill; move it to minute 60.5, so only segment 0 (newest
	// record at minute 0) is older than an hour.
	clk.Advance(time.Hour - 3*time.Minute + 30*time.Second)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 3 || ids[0] != 1 {
		t.Fatalf("segments %v, want [1 2 3]: the byte cap deleted more than the age sweep and the cap required", ids)
	}
}

func TestMaxBytesKeepsASetExactlyAtTheCap(t *testing.T) {
	s, clk, _, total := capsFixture(t)
	l := openRetention(t, s, clk, func(c *Config) { c.MaxBytes = total })
	defer l.Close()
	if st, _ := l.Stat(); st.Bytes != total {
		t.Fatalf("fixture: %d bytes after reopening, want %d", st.Bytes, total)
	}
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 4 || ids[0] != 0 {
		t.Fatalf("segments %v, want [0 1 2 3]: a set exactly at MaxBytes lost a segment", ids)
	}
	// One byte under the cap does delete: the first segment is enough.
	_ = l.Close()
	l = openRetention(t, s, clk, func(c *Config) { c.MaxBytes = total - 1 })
	defer l.Close()
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 3 || ids[0] != 1 {
		t.Fatalf("segments %v, want [1 2 3] one byte over MaxBytes", ids)
	}
}

func TestMinFreeBytesStopsDeletingOnceTheFloorIsMetExactly(t *testing.T) {
	s, clk, size0, _ := capsFixture(t)
	free := uint64(5000) - uint64(size0)
	l := openRetention(t, s, clk, func(c *Config) {
		c.MinFreeBytes = 5000
		c.statfs = func(string) (uint64, error) { return free, nil }
	})
	defer l.Close()
	// Deleting segment 0 lands free on the floor exactly: that meets it, and no more goes.
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 3 || ids[0] != 1 {
		t.Fatalf("segments %v, want [1 2 3]: deleting continued once free space met the floor", ids)
	}
	// Free space already exactly at the floor deletes nothing.
	free = 5000
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 3 || ids[0] != 1 {
		t.Fatalf("segments %v, want [1 2 3]: free space at the floor deleted a segment", ids)
	}
}

func TestMinFreeBytesIsMetAtExactlyTheFloor(t *testing.T) {
	// Nothing sealed, so only the full flag decides: free space equal to the floor is not full.
	free := uint64(4999)
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, func(c *Config) {
		c.MinFreeBytes = 5000
		c.statfs = func(string) (uint64, error) { return free, nil }
	})
	defer l.Close()
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.Stat(); !st.DiskFull {
		t.Fatal("DiskFull is false one byte under the floor")
	}
	if _, err := l.Append([]byte("x")); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("append one byte under the floor: %v", err)
	}
	free = 5000
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.Stat(); st.DiskFull {
		t.Fatal("DiskFull is true with free space exactly at the floor")
	}
	if _, err := l.Append([]byte("x")); err != nil {
		t.Fatalf("append with free space exactly at the floor: %v", err)
	}
}

// The byte cap counts the active segment and the record cap keeps no more than its count
// (Stage 2 audit L2-28). fillSegments leaves the active segment empty and the record-cap test
// ends at a count where one more record does not matter, so the first test below puts a record
// in the active segment and the second picks a cap that falls between segment sizes.

func TestMaxBytesCountsTheActiveSegment(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	fillSegments(t, l, clk, 3) // segments 0..2 sealed
	appendN(t, l, 1)           // one record in the active segment 3
	mustFlush(t, l)
	st, err := l.Stat()
	if err != nil {
		t.Fatal(err)
	}
	total := st.Bytes
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	// The whole set is one byte over the cap, so the oldest sealed segment must go. Counting
	// only the sealed segments, the set is under the cap by the active segment's bytes.
	l = openRetention(t, s, clk, func(c *Config) { c.MaxBytes = total - 1 })
	defer l.Close()
	if st, _ := l.Stat(); st.Bytes != total || st.Segments != 4 {
		t.Fatalf("fixture: %+v after reopening, want %d bytes in 4 segments", st, total)
	}
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 3 || ids[0] != 1 {
		t.Fatalf("segments %v, want [1 2 3]: the active segment's bytes were not counted against MaxBytes", ids)
	}
	if st, _ := l.Stat(); st.Bytes > total-1 {
		t.Fatalf("%d bytes kept under MaxBytes %d", st.Bytes, total-1)
	}
}

func TestMaxRecordsNeverLeavesMoreThanTheCap(t *testing.T) {
	_, clk, l := retentionLog(t, func(c *Config) { c.MaxRecords = 7 })
	defer l.Close()
	fillSegments(t, l, clk, 3) // records 1..12 in segments of 4
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	// Two segments hold 8 records, one more than the cap: both of the oldest must go.
	st, _ := l.Stat()
	if st.Records > 7 || st.Records != 4 || st.First != 9 {
		t.Fatalf("%+v: want the newest 4 records, 9..12, under MaxRecords 7", st)
	}
}
