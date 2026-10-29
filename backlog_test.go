package quill

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// T19: backlog independence. The cost of a read is measured in file operations on the
// simulator, which is what grows with a backlog if anything does.

func TestAReadAtTheStartOfAThousandSegmentsCostsAboutWhatOneAtTheTailDoes(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 70; c.FlushBytes = 1 << 20 })
	defer l.Close()
	const segments = 1000
	for i := 0; i < segments; i++ {
		appendN(t, l, 4)
		mustFlush(t, l)
	}
	head := l.Head()
	cost := func(f func()) int {
		before := s.Ops()
		f()
		return s.Ops() - before
	}
	start := cost(func() {
		if recs, err := l.Read(1, 100); err != nil || len(recs) != 100 {
			t.Fatalf("%d %v", len(recs), err)
		}
	})
	tailCost := cost(func() {
		if recs, err := l.Read(head-100, 100); err != nil || len(recs) != 100 {
			t.Fatalf("%d %v", len(recs), err)
		}
	})
	if start > 3*tailCost+10 {
		t.Fatalf("a read at the start cost %d operations, at the tail %d", start, tailCost)
	}
	// A Reader: the first touch of each segment costs one scan, excluded; then indexed.
	r := openReaderSim(t, s)
	defer r.Close()
	warm := func(offset uint64) {
		if _, err := r.Read(offset, 100); err != nil {
			t.Fatal(err)
		}
	}
	warm(1)
	warm(head - 100)
	rStart := cost(func() { warm(1) })
	rTail := cost(func() { warm(head - 100) })
	if rStart > 3*rTail+10 {
		t.Fatalf("reader: at the start %d operations, at the tail %d", rStart, rTail)
	}
	// Draining a long backlog never slows as it falls further behind: the operations per
	// batch stay flat from the first batch to the last.
	var costs []int
	for off := uint64(1); off <= head; {
		var recs []Record
		c := cost(func() {
			var err error
			if recs, err = r.Read(off, 100); err != nil {
				t.Fatal(err)
			}
		})
		costs = append(costs, c)
		off = recs[len(recs)-1].Seq + 1
	}
	lo, hi := costs[0], costs[0]
	for _, c := range costs[1:] {
		lo, hi = min(lo, c), max(hi, c)
	}
	if hi > 3*lo+10 {
		t.Fatalf("batch costs ranged from %d to %d over %d batches", lo, hi, len(costs))
	}
}

// T14: a full disk, for real. Needs QUILL_FULLDISK_DIR, a directory on a small filesystem
// (a tmpfs of a few megabytes in CI). Not part of the default gate.

func TestAFullDiskFailsAppendsKeepsRecordsIntactAndRecoversWhenSpaceReturns(t *testing.T) {
	root := os.Getenv("QUILL_FULLDISK_DIR")
	if root == "" {
		t.Skip("set QUILL_FULLDISK_DIR to a directory on a small filesystem")
	}
	dir := filepath.Join(root, "log")
	_ = os.RemoveAll(dir)
	filler := filepath.Join(root, "filler")
	_ = os.Remove(filler)
	var st syscall.Statfs_t
	if err := syscall.Statfs(root, &st); err != nil {
		t.Fatal(err)
	}
	free := uint64(st.Bavail) * uint64(st.Bsize)
	// Leave room for about 300 KiB of log: a filler takes the rest.
	if free < 1<<20 {
		t.Fatalf("only %d bytes free; the test wants a small filesystem with at least 1 MiB free", free)
	}
	if err := os.WriteFile(filler, make([]byte, free-300<<10), 0o600); err != nil {
		t.Fatalf("filler: %v", err)
	}
	l, err := Open(Config{Dir: dir, Retention: -1, SegmentBytes: 64 << 10, FlushInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	payload := make([]byte, 4096)
	var flushErr error
	var acked uint64
	for i := 0; i < 10_000 && flushErr == nil; i++ {
		seq, err := l.Append(payload)
		if err != nil {
			flushErr = err
			break
		}
		if i%4 == 3 {
			if flushErr = l.Flush(); flushErr == nil {
				acked = seq
			}
		}
	}
	if flushErr == nil {
		t.Fatal("the disk never filled")
	}
	t.Logf("filled after %d acknowledged records: %v", acked, flushErr)
	if acked == 0 {
		t.Fatal("nothing was written before the disk filled")
	}
	recs, err := l.Read(1, 0)
	if err != nil || uint64(len(recs)) < acked {
		t.Fatalf("records still read back: %d %v, acked %d", len(recs), err, acked)
	}
	r, err := OpenReader(dir, ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if res, err := r.Check(); err != nil || !res.Healthy {
		t.Fatalf("check: %+v %v", res, err)
	}
	if err := os.Remove(filler); err != nil {
		t.Fatal(err)
	}
	if err := l.Flush(); err != nil {
		t.Fatalf("flush after space returned: %v", err)
	}
	seq, err := l.Append(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Flush(); err != nil {
		t.Fatal(err)
	}
	if l.Head() != seq {
		t.Fatalf("head %d, want %d", l.Head(), seq)
	}
	recs, err = l.Read(1, 0)
	if err != nil || uint64(len(recs)) != seq {
		t.Fatalf("%d %v", len(recs), err)
	}
}
