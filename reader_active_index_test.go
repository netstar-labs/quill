package quill

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// activeIndexLog is a log on the simulator whose active segment holds several indexStrides of
// records and has not rotated, with a Reader on it.
func activeIndexLog(t *testing.T, records int) (*fsx.Sim, *Log, *Reader) {
	t.Helper()
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 8 << 20; c.MaxBytes = 0 })
	t.Cleanup(func() { _ = l.Close() })
	appendN(t, l, records)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	t.Cleanup(func() { _ = r.Close() })
	return s, l, r
}

// countReads counts the reads of path the simulator serves until the returned stop is called.
func countReads(s *fsx.Sim, path string) (n *atomic.Int64) {
	n = new(atomic.Int64)
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpRead && name == path {
			n.Add(1)
		}
		return nil
	}
	return n
}

// A cursorless Reader.Read near the end of a large active segment starts at the nearest anchor,
// not at byte 0 (decision 63, amended for A3-18): once the Reader has verified the segment's
// prefix, a read costs the preads of one stride, not of the whole offset.
func TestReaderReadNearTheEndOfALargeActiveSegmentStartsAtAnAnchor(t *testing.T) {
	s, l, r := activeIndexLog(t, 150000)
	active := segmentPaths(t, s)
	if len(active) != 1 {
		t.Fatalf("want one segment, have %v", active)
	}
	head := l.Head()
	if recs, err := r.Read(head, 1); err != nil || len(recs) != 1 || recs[0].Seq != head {
		t.Fatalf("warm read: %v, %v", recs, err)
	}
	n := countReads(s, active[0])
	defer func() { s.Inject = nil }()
	if recs, err := r.Read(head-3, 2); err != nil || len(recs) != 2 {
		t.Fatalf("read: %v, %v", recs, err)
	}
	if got := n.Load(); got > 3 {
		t.Fatalf("a read near the end of the active segment made %d reads of it; want at most 3 (first frame, one stride)", got)
	}
}

// A Reader that keeps its prefix index across views reads exactly what a Reader with no memory
// reads, at every offset, while the writer appends and rotates under it.
func TestReaderActiveIndexReadsLikeAFreshReader(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 300 << 10; c.MaxBytes = 0 })
	defer l.Close()
	long := openReaderSim(t, s)
	defer long.Close()
	same := func(step string) {
		t.Helper()
		head := l.Head()
		for _, off := range []uint64{1, 2, head / 4, head / 2, head - 70000, head - 5000, head - 3, head - 1, head} {
			if off == 0 || off > head {
				continue
			}
			got, gerr := long.Read(off, 6)
			fresh := openReaderSim(t, s)
			want, werr := fresh.Read(off, 6)
			_ = fresh.Close()
			if (gerr == nil) != (werr == nil) || len(got) != len(want) {
				t.Fatalf("%s: Read(%d): long %d records, %v; fresh %d records, %v", step, off, len(got), gerr, len(want), werr)
			}
			for i := range got {
				if got[i].Seq != want[i].Seq || got[i].At != want[i].At || !bytes.Equal(got[i].Payload, want[i].Payload) {
					t.Fatalf("%s: Read(%d) record %d: long %+v, fresh %+v", step, off, i, got[i], want[i])
				}
			}
		}
	}
	for round := 1; round <= 6; round++ {
		appendN(t, l, 9000) // about 300 KiB: a rotation about every round
		mustFlush(t, l)
		same("round " + string(rune('0'+round)))
	}
}

// copyFile writes src's bytes to dst on another simulator, replacing what is there.
func copyFile(t *testing.T, from *fsx.Sim, src string, to *fsx.Sim, dst string) {
	t.Helper()
	in, err := from.OpenFile(src, fsx.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, info.Size())
	if _, err := in.ReadAt(data, 0); err != nil && info.Size() > 0 {
		t.Fatal(err)
	}
	_ = to.Remove(dst)
	out, err := to.OpenFile(dst, fsx.O_RDWR|fsx.O_CREATE|fsx.O_EXCL, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := out.WriteAt(data, 0); err != nil {
		t.Fatal(err)
	}
	if err := out.Sync(); err != nil {
		t.Fatal(err)
	}
}

// A prefix index belongs to one file. When another file takes the segment's name (a restore,
// a copy) the anchors of the old one are not used, whatever sizes the two have.
func TestReaderActiveIndexIsNotUsedForAnotherFile(t *testing.T) {
	s, l, r := activeIndexLog(t, 30000)
	if recs, err := r.Read(l.Head(), 1); err != nil || len(recs) != 1 {
		t.Fatalf("warm read: %v, %v", recs, err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	// A second log with the same segment id and marker shape but records of other lengths, so
	// no byte position of the first one is a frame boundary of the second.
	s2 := fsx.NewSim()
	l2 := openSim(t, s2, func(c *Config) { c.SegmentBytes = 8 << 20; c.MaxBytes = 0 })
	payload := bytes.Repeat([]byte("z"), 101)
	for i := 0; i < 30000; i++ {
		if _, err := l2.Append(payload); err != nil {
			t.Fatal(err)
		}
	}
	mustFlush(t, l2)
	head2 := l2.Head()
	if err := l2.Close(); err != nil {
		t.Fatal(err)
	}
	seg := segmentPaths(t, s)[0]
	copyFile(t, s2, seg, s, seg)
	copyFile(t, s2, "/log/"+tailName, s, "/log/"+tailName)
	fresh := openReaderSim(t, s)
	defer fresh.Close()
	for _, off := range []uint64{head2 - 2, head2 / 2, 3} {
		got, gerr := r.Read(off, 2)
		want, werr := fresh.Read(off, 2)
		if gerr != nil || werr != nil || len(got) != 2 || len(want) != 2 || got[0].Seq != want[0].Seq || !bytes.Equal(got[0].Payload, want[0].Payload) || !bytes.Equal(got[1].Payload, want[1].Payload) {
			t.Fatalf("Read(%d): long %v, %v; fresh %v, %v", off, got, gerr, want, werr)
		}
	}
}

// A view that was built before the writer went on, and is used after another read extended the
// prefix, still gets only anchors that start below its own marker offset.
func TestReaderActiveIndexOfAnOlderViewStopsAtItsMarkerOffset(t *testing.T) {
	_, l, r := activeIndexLog(t, 30000)
	v, release, err := r.view()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ref := v.segs[len(v.segs)-1]
	_, size, _, err := ref.open()
	if err != nil {
		t.Fatal(err)
	}
	appendN(t, l, 30000)
	mustFlush(t, l)
	if recs, err := r.Read(l.Head(), 1); err != nil || len(recs) != 1 {
		t.Fatalf("read past the old view: %v, %v", recs, err)
	}
	idx := ref.indexTo(v.head)
	if len(idx) == 0 {
		t.Fatal("no anchors for the older view")
	}
	for _, a := range idx {
		if a.pos >= size || a.seq > v.head {
			t.Fatalf("anchor %+v is past the older view's offset %d or head %d", a, size, v.head)
		}
	}
}

// Readers on one Reader, a writer that appends and rotates, and every record read must be the
// one written: the prefix index is shared by the readers and extended under them.
func TestReaderActiveIndexUnderConcurrentReadsAndRotation(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 200 << 10; c.MaxBytes = 0 })
	defer l.Close()
	appendN(t, l, 100)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()
	stop := make(chan struct{})
	errc := make(chan error, 4)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				head, err := r.Head()
				if err != nil {
					errc <- err
					return
				}
				off := head - uint64((i*7919+g*104729)%int(min(head, 60000)))
				recs, err := r.Read(off, 4)
				var gap *GapError
				if err != nil && !errors.As(err, &gap) {
					errc <- fmt.Errorf("Read(%d): %w", off, err)
					return
				}
				for _, rec := range recs {
					if !bytes.Equal(rec.Payload, payloadFor(rec.Seq)) {
						errc <- fmt.Errorf("record %d has payload %q", rec.Seq, rec.Payload)
						return
					}
				}
			}
		}(g)
	}
	for round := 0; round < 40; round++ {
		appendN(t, l, 1500)
		mustFlush(t, l)
	}
	close(stop)
	wg.Wait()
	select {
	case err := <-errc:
		t.Fatal(err)
	default:
	}
}

// A Reader that has indexed the active segment keeps reading what a new Reader reads when the
// writer crashes and recovers under it, whatever the crash made of the bytes the marker did
// not yet cover (they may persist, vanish, or come back as other records, which is why the
// first frame was never cached, decision 63): the prefix index holds only bytes below the
// marker, and those a crash does not change.
func TestReaderActiveIndexAcrossAWriterCrashAndRecovery(t *testing.T) {
	keeps := map[string]func(i, n int) bool{
		"nothing persists":     nil,
		"everything persists":  func(i, n int) bool { return true },
		"alternate items kept": func(i, n int) bool { return i%2 == 0 },
	}
	crashAt := map[string]func(op fsx.Op, name string) bool{
		"before the marker publish": func(op fsx.Op, name string) bool { return op == fsx.OpRename },
		"before the data fsync":     func(op fsx.Op, name string) bool { return op == fsx.OpSync && name == "/log/"+segmentName(0) },
	}
	for kname, keep := range keeps {
		for cname, at := range crashAt {
			t.Run(kname+", "+cname, func(t *testing.T) {
				s := fsx.NewSim()
				s.CrashKeep = keep
				cfg := simConfig(s, "/log")
				cfg.SegmentBytes, cfg.MaxBytes = 8<<20, 0
				l, err := Open(cfg)
				if err != nil {
					t.Fatal(err)
				}
				appendN(t, l, 20000)
				mustFlush(t, l)
				long := openReaderSim(t, s)
				defer long.Close()
				head := l.Head()
				if recs, err := long.Read(head, 1); err != nil || len(recs) != 1 { // indexes the prefix
					t.Fatalf("warm read: %v, %v", recs, err)
				}
				appendN(t, l, 4000)
				s.Inject = func(op fsx.Op, name string) error {
					if at(op, name) {
						return fsx.ErrCrash
					}
					return nil
				}
				_ = l.Flush() // the crash happens inside
				s.Inject = nil
				l2, err := Open(cfg)
				if err != nil {
					t.Fatalf("recovery: %v", err)
				}
				defer l2.Close()
				for i := 0; i < 6000; i++ { // other records than the ones the crash may have kept or lost
					if _, err := l2.Append(bytes.Repeat([]byte{'q'}, 40+i%7)); err != nil {
						t.Fatal(err)
					}
				}
				mustFlush(t, l2)
				newHead := l2.Head()
				if newHead <= head {
					t.Fatalf("head %d after recovery, was %d", newHead, head)
				}
				fresh := openReaderSim(t, s)
				defer fresh.Close()
				for _, off := range []uint64{1, head / 2, head - 70000 + 1, head - 5, head, head + 1, head + 2000, newHead - 3, newHead} {
					if off == 0 || off > newHead {
						continue
					}
					got, gerr := long.Read(off, 5)
					want, werr := fresh.Read(off, 5)
					if (gerr == nil) != (werr == nil) || len(got) != len(want) {
						t.Fatalf("Read(%d): long %d records, %v; fresh %d records, %v", off, len(got), gerr, len(want), werr)
					}
					for i := range got {
						if got[i].Seq != want[i].Seq || got[i].At != want[i].At || !bytes.Equal(got[i].Payload, want[i].Payload) {
							t.Fatalf("Read(%d) record %d: long %+v; fresh %+v", off, i, got[i], want[i])
						}
					}
				}
			})
		}
	}
}
