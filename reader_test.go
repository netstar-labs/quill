package quill

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

func openReaderSim(t *testing.T, s *fsx.Sim) *Reader {
	t.Helper()
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatalf("open reader: %v", err)
	}
	return r
}

// A Reader opened with no options is bounded by DefaultMaxReadBytes (16 MiB), not unbounded: 40
// records of 512 KiB are 20 MiB, and the read stops after the 32nd, which reaches 16 MiB exactly.
func TestReaderDefaultMaxReadBytesBoundsARead(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 4 << 20 })
	defer l.Close()
	payload := make([]byte, 512<<10)
	for i := 0; i < 40; i++ {
		if _, err := l.Append(payload); err != nil {
			t.Fatal(err)
		}
		mustFlush(t, l)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	recs, err := r.Read(1, 0)
	if err != nil || len(recs) != 32 {
		t.Fatalf("%d records, %v; want 32", len(recs), err)
	}
}

func readerAll(r *Reader) ([]Record, error) {
	var out []Record
	err := r.ScanAll(func(rec Record) bool {
		rec.Payload = append([]byte(nil), rec.Payload...)
		out = append(out, rec)
		return true
	})
	return out, err
}

func TestAReaderSeesOnlyWhatTheMarkerNamesAndNeverAWritersUnpublishedBytes(t *testing.T) {
	s := fsx.NewSim()
	if _, err := OpenReader("/log", ReaderOptions{fs: s}); !errors.Is(err, ErrNoLog) {
		t.Fatalf("before creation: %v", err)
	}
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 200 })
	defer l.Close()
	r := openReaderSim(t, s)
	defer r.Close()
	if head, _ := r.Head(); head != 0 {
		t.Fatalf("head %d", head)
	}
	if recs, err := r.Read(1, 10); err != nil || len(recs) != 0 {
		t.Fatalf("empty log: %v %v", recs, err)
	}
	appendN(t, l, 5)
	if recs, _ := r.Read(1, 10); len(recs) != 0 {
		t.Fatalf("unflushed records visible: %v", seqsOf(recs))
	}
	mustFlush(t, l)
	recs, err := r.Read(1, 10)
	if err != nil || checkContiguous(t, recs, 1) != 5 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	// Bytes written past the marker (a writer's flush in progress) are never served, even
	// when they are intact, contiguous frames.
	appendN(t, l, 3)
	l.mu.Lock()
	buf := bytes.Clone(l.buf)
	off := l.active.durable
	l.mu.Unlock()
	f, _ := s.OpenFile("/log/"+segmentName(0), fsx.O_RDWR, 0)
	if _, err := f.WriteAt(buf, off); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	recs, err = r.Read(1, 100)
	if err != nil || checkContiguous(t, recs, 1) != 5 {
		t.Fatalf("served past the marker: %v %v", seqsOf(recs), err)
	}
	if head, _ := r.Head(); head != 5 {
		t.Fatalf("head %d", head)
	}
	mustFlush(t, l)
	recs, err = r.Read(1, 100)
	if err != nil || checkContiguous(t, recs, 1) != 8 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	// Across many segments, with the reader's per-segment index built lazily.
	for i := 0; i < 40; i++ {
		appendN(t, l, 3)
		mustFlush(t, l)
	}
	for offset := uint64(1); offset <= 130; offset += 7 {
		recs, err := r.Read(offset, 4)
		want := min(uint64(4), 129-offset)
		if err != nil || uint64(len(recs)) != want || checkContiguous(t, recs, offset) != offset+want-1 {
			t.Fatalf("offset %d: %v %v", offset, seqsOf(recs), err)
		}
	}
	if _, err := r.Read(1, 1); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	if _, err := r.Read(1, 1); !errors.Is(err, ErrClosed) {
		t.Fatalf("after close: %v", err)
	}
}

func TestScanAllReadAndAReaderAgreeRecordForRecordAcrossManySegments(t *testing.T) {
	// T4.
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 150 })
	defer l.Close()
	for i := 0; i < 60; i++ {
		appendN(t, l, 1+i%4)
		if i%3 == 0 {
			mustFlush(t, l)
		}
	}
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()
	scanned, err := readAll(l)
	if err != nil {
		t.Fatal(err)
	}
	last := checkContiguous(t, scanned, 1)
	var read []Record
	for off := uint64(1); off <= last; {
		recs, err := l.Read(off, 7)
		if err != nil || len(recs) == 0 {
			t.Fatalf("read %d: %v %v", off, recs, err)
		}
		read = append(read, recs...)
		off = recs[len(recs)-1].Seq + 1
	}
	viaReader, err := readerAll(r)
	if err != nil {
		t.Fatal(err)
	}
	var readerBatches []Record
	for off := uint64(1); off <= last; {
		recs, err := r.Read(off, 5)
		if err != nil || len(recs) == 0 {
			t.Fatalf("reader read %d: %v %v", off, recs, err)
		}
		readerBatches = append(readerBatches, recs...)
		off = recs[len(recs)-1].Seq + 1
	}
	for _, other := range [][]Record{read, viaReader, readerBatches} {
		if len(other) != len(scanned) {
			t.Fatalf("%d records, want %d", len(other), len(scanned))
		}
		for i := range other {
			if !recEqual(other[i], scanned[i]) {
				t.Fatalf("record %d differs: %+v %+v", i, other[i], scanned[i])
			}
		}
	}
	if head, _ := r.Head(); head != last {
		t.Fatalf("reader head %d, want %d", head, last)
	}
}

func TestAReaderReportsGapsForDeletedAndDamagedSegmentsAndStatKnowsThem(t *testing.T) {
	// T7, the reader's side.
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
	defer l.Close()
	for i := 0; i < 5; i++ {
		appendN(t, l, 4)
		mustFlush(t, l) // segment i holds 4i+1 .. 4i+4
	}
	r := openReaderSim(t, s)
	defer r.Close()
	if err := s.Remove("/log/" + segmentName(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(1, 10); !isGap(err, 1, 4) {
		t.Fatalf("below the first retained: %v", err)
	}
	if _, err := r.Read(3, 10); !isGap(err, 3, 4) {
		t.Fatalf("below the first retained: %v", err)
	}
	if err := s.Remove("/log/" + segmentName(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Read(9, 10); !isGap(err, 9, 12) {
		t.Fatalf("hole: %v", err)
	}
	if _, err := r.Read(11, 10); !isGap(err, 11, 12) {
		t.Fatalf("inside the hole: %v", err)
	}
	recs, err := r.Read(5, 100)
	if err != nil || checkContiguous(t, recs, 5) != 8 {
		t.Fatalf("before the hole: %v %v", seqsOf(recs), err)
	}
	if _, err := readerAll(r); !isGap(err, 9, 12) {
		t.Fatalf("ScanAll stops at the gap: %v", err)
	}
	st, err := r.Stat()
	if err != nil || st.First != 5 || st.Head != 20 || len(st.Holes) != 1 || st.Holes[0] != (Hole{9, 12}) || st.Records != 12 || st.Segments != 4 {
		t.Fatalf("stat %+v %v", st, err)
	}
	// A CRC failure inside a sealed segment, found at read time: a gap over its range.
	seg3 := "/log/" + segmentName(3)
	data := fileBytes(t, s, seg3)
	data[len(data)-3] ^= 0xff
	writeFile(t, s, seg3, data)
	r2 := openReaderSim(t, s)
	defer r2.Close()
	if _, err := r2.Read(14, 10); !isGap(err, 14, 16) {
		t.Fatalf("damaged sealed segment: %v", err)
	}
	recs, err = r2.Read(17, 10)
	if err != nil || checkContiguous(t, recs, 17) != 20 {
		t.Fatalf("after it: %v %v", seqsOf(recs), err)
	}
	// Its first frame damaged: the segment's range is unknown, so it is a hole between its
	// neighbours.
	data[1] ^= 0xff
	writeFile(t, s, seg3, data)
	r3 := openReaderSim(t, s)
	defer r3.Close()
	if _, err := r3.Read(13, 10); !isGap(err, 13, 16) {
		t.Fatalf("first frame damaged: %v", err)
	}
}

func TestReaderCheckExaminesSealedSegmentsWholeAndTheActiveOneOnlyToTheMarker(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
	defer l.Close()
	for i := 0; i < 3; i++ {
		appendN(t, l, 4)
		mustFlush(t, l)
	}
	appendN(t, l, 1) // 35 bytes: segment 3 stays active
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()
	res, err := r.Check()
	if err != nil || !res.Healthy || len(res.Problems) != 0 || !res.Unbounded {
		t.Fatalf("healthy log: %+v %v", res, err)
	}
	// Junk past the marker in the active segment is a live writer's business.
	active := "/log/" + segmentName(3)
	data := fileBytes(t, s, active)
	writeFile(t, s, active, append(bytes.Clone(data), 0xde, 0xad, 0xbe, 0xef))
	if res, _ := r.Check(); !res.Healthy {
		t.Fatalf("junk past the marker: %+v", res)
	}
	// Damage below the marker.
	bad := bytes.Clone(data)
	bad[len(bad)-2] ^= 1
	writeFile(t, s, active, bad)
	res, _ = r.Check()
	if res.Healthy || len(res.Problems) != 2 || res.Problems[0].Kind != "segment" || res.Problems[0].Path != active || res.Problems[0].Intact != 0 || res.Problems[1].Kind != "hole" {
		t.Fatalf("damaged active: %+v", res)
	}
	writeFile(t, s, active, data)
	// A sealed segment damaged in its middle, a hole, an unparseable consumer file.
	seg1 := "/log/" + segmentName(1)
	d1 := fileBytes(t, s, seg1)
	d1[len(d1)/2] ^= 1
	writeFile(t, s, seg1, d1)
	if err := s.Remove("/log/" + segmentName(2)); err != nil {
		t.Fatal(err)
	}
	writeFile(t, s, "/log/consumers/broken", []byte("garbage"))
	res, _ = r.Check()
	kinds := map[string]int{}
	for _, p := range res.Problems {
		kinds[p.Kind]++
	}
	if res.Healthy || kinds["segment"] != 1 || kinds["hole"] != 1 || kinds["consumer"] != 1 {
		t.Fatalf("problems %+v", res.Problems)
	}
	// The clock: a record dated in the future, one earlier than its predecessor.
	s2 := fsx.NewSim()
	l2 := openSim(t, s2, nil)
	defer l2.Close()
	now := time.Now()
	future := image(1, 2, now.Add(48*time.Hour).Unix(), smallPayload)
	back := image(3, 1, now.Unix()-100, smallPayload)
	writeFile(t, s2, "/log/"+segmentName(0), append(future, back...))
	writeFile(t, s2, "/log/"+tailName, encodeTail(tail{Seq: 3, Seg: 0, Off: uint64(len(future) + len(back)), MaxSeg: 0}))
	r2, err := OpenReader("/log", ReaderOptions{fs: s2})
	if err != nil {
		t.Fatal(err)
	}
	res, _ = r2.Check()
	if !res.Healthy {
		t.Fatalf("clock problems are warnings: %+v", res.Problems)
	}
	clock := 0
	for _, p := range res.Problems {
		if p.Kind == "clock" {
			clock++
		}
	}
	if clock != 3 { // two in the future, one earlier than its predecessor
		t.Fatalf("clock problems %d: %+v", clock, res.Problems)
	}
}

func TestReaderReadWaitWakesWithinThePollIntervalAndMarkerAgeIsExposed(t *testing.T) {
	s := fsx.NewSim()
	now := time.Unix(1_700_000_000, 0)
	s.Clock = func() time.Time { return now }
	l := openSim(t, s, func(c *Config) { c.clock = func() time.Time { return now } })
	defer l.Close()
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond, clock: func() time.Time { return now.Add(90 * time.Second) }})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	done := make(chan []Record, 1)
	go func() {
		recs, err := r.ReadWait(context.Background(), 1, 10)
		if err != nil {
			t.Error(err)
		}
		done <- recs
	}()
	time.Sleep(20 * time.Millisecond)
	appendN(t, l, 2)
	mustFlush(t, l)
	select {
	case recs := <-done:
		checkContiguous(t, recs, 1)
	case <-time.After(time.Second):
		t.Fatal("ReadWait did not wake")
	}
	age, err := r.MarkerAge()
	if err != nil || age != 90*time.Second {
		t.Fatalf("marker age %v %v", age, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := r.ReadWait(ctx, 3, 10); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("%v", err)
	}
}

// A marker whose off no segment could reach (and one that does not fit an int64) is refused
// as corruption by every reader call; it must never reach a slice bound.
func TestAReaderRefusesAMarkerOffsetBeyondAnyPossibleSegment(t *testing.T) {
	for _, off := range []string{"9300000000000000000", "9223372036854775808", "18446744073709551615", "1610612737"} {
		s := fsx.NewSim()
		l := openSim(t, s, nil)
		appendN(t, l, 3)
		mustFlush(t, l)
		_ = l.Close()
		writeFile(t, s, "/log/quill.tail", []byte(`{"seq":3,"seg":0,"off":`+off+`,"maxseg":0}`+"\n"))
		r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
		if err != nil {
			if !errors.Is(err, ErrCorrupt) {
				t.Errorf("off %s: open: %v", off, err)
			}
			continue
		}
		if _, err := r.Stat(); !errors.Is(err, ErrCorrupt) {
			t.Errorf("off %s: Stat: %v", off, err)
		}
		if _, err := r.Read(1, 10); !errors.Is(err, ErrCorrupt) {
			t.Errorf("off %s: Read: %v", off, err)
		}
		r.Close()
	}
}

func TestAReaderRefusesANewerFormat(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	_ = l.Close()
	writeFile(t, s, "/log/quill.format", []byte("quill/2 0123456789abcdef0123456789abcdef\n"))
	if _, err := OpenReader("/log", ReaderOptions{fs: s}); !errors.Is(err, ErrFutureVersion) {
		t.Fatalf("%v", err)
	}
}

// T6: a writer in another process, a reader following it, the writer killed.
func TestAReaderFollowingAWriterInAnotherProcessNeverOutrunsTheMarkerOrRecovery(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "log")
	cmd, out := startHelper(t, "writer", "QUILL_HELPER_DIR="+dir, "QUILL_HELPER_SEGBYTES=4096")
	var err error
	// Wait for the log to exist.
	var r *Reader
	for i := 0; i < 500; i++ {
		r, err = OpenReader(dir, ReaderOptions{PollInterval: 2 * time.Millisecond})
		if err == nil {
			break
		}
		if !errors.Is(err, ErrNoLog) {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if r == nil {
		t.Fatal("the log never appeared")
	}
	defer r.Close()
	var seen []Record
	sc := bufio.NewScanner(out)
	var acked uint64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for sc.Scan() {
			if n, ok := parseAck(sc.Text()); ok {
				acked = n
			}
		}
	}()
	deadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(deadline) {
		next := uint64(1)
		if len(seen) > 0 {
			next = seen[len(seen)-1].Seq + 1
		}
		head, err := r.Head()
		if err != nil {
			t.Fatal(err)
		}
		recs, err := r.ReadWait(ctx, next, 50)
		if err != nil {
			t.Fatal(err)
		}
		for _, rec := range recs {
			if rec.Seq > head {
				// The marker may have advanced between the two calls; re-read it.
				head2, _ := r.Head()
				if rec.Seq > head2 {
					t.Fatalf("record %d beyond the marker %d", rec.Seq, head2)
				}
			}
		}
		seen = append(seen, recs...)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	last := checkContiguous(t, seen, 1)
	// Keep following after the kill: nothing new appears, and nothing read so far is lost.
	time.Sleep(20 * time.Millisecond)
	more, err := r.Read(last+1, 100)
	if err != nil {
		t.Fatal(err)
	}
	l, err := Open(Config{Dir: dir, Retention: -1, FlushInterval: time.Hour})
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	defer l.Close()
	recovered, err := readAll(l)
	if err != nil {
		t.Fatal(err)
	}
	recoveredLast := checkContiguous(t, recovered, 1)
	if recoveredLast < last || (len(more) > 0 && more[len(more)-1].Seq > recoveredLast) {
		t.Fatalf("the reader saw %d (then %d more), recovery holds %d", last, len(more), recoveredLast)
	}
	if recoveredLast < acked {
		t.Fatalf("acknowledged %d, recovered %d", acked, recoveredLast)
	}
	// After recovery the reader sees everything durable, with its cached segments intact.
	all, err := readerAll(r)
	if err != nil || checkContiguous(t, all, 1) != recoveredLast {
		t.Fatalf("after recovery: %d records %v, want %d", len(all), err, recoveredLast)
	}
	appendN(t, l, 3)
	mustFlush(t, l)
	all, err = readerAll(r)
	if err != nil || checkContiguous(t, all, 1) != recoveredLast+3 {
		t.Fatalf("after new records: %d %v", len(all), err)
	}
	t.Logf("reader saw %d, acked %d, recovered %d", last, acked, recoveredLast)
}

func TestReaderSealedSegmentWithSeqZeroFirstFrameIsAHole(t *testing.T) {
	// A sealed segment whose first frame decodes but carries seq 0 (decision 71: the first
	// frame of a segment must be at least 1) is damage, reported as a hole like any other
	// unreadable segment, not a crash.
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	if _, err := l.Append([]byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Append([]byte("two")); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := s.OpenFile("/log/"+segmentName(0), fsx.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(0); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt(appendFrame(nil, &Record{Seq: 0, At: 1, Payload: []byte("zero")}), 0); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	r := openReaderSim(t, s)
	defer r.Close()
	var gap *GapError
	recs, err := r.Read(1, 10)
	if !errors.As(err, &gap) || gap.From != 1 || gap.To != 1 {
		t.Fatalf("got %v %v, want a gap at 1", seqsOf(recs), err)
	}
}

func TestConcurrentReadsOnOneReaderAreRaceFree(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
	defer l.Close()
	for i := 0; i < 8; i++ {
		appendN(t, l, 4)
		mustFlush(t, l) // eight sealed segments, 4i+1 .. 4i+4
	}
	r := openReaderSim(t, s)
	defer r.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				recs, err := r.Read(7, 3)
				if err != nil {
					errs <- err
					return
				}
				if len(recs) != 3 || recs[0].Seq != 7 || recs[1].Seq != 8 || recs[2].Seq != 9 {
					errs <- fmt.Errorf("Read(7, 3) = %v", seqsOf(recs))
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestATransientOpenErrorIsNotCachedByTheReader(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 200 })
	defer l.Close()
	appendN(t, l, 30)
	mustFlush(t, l)
	if st, err := l.Stat(); err != nil || st.Segments < 2 {
		t.Fatalf("want a sealed segment: %+v %v", st, err)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	seg0 := "/log/" + segmentName(0)
	fail := true
	s.Inject = func(op fsx.Op, name string) error {
		if fail && op == fsx.OpOpen && name == seg0 {
			fail = false
			return errors.New("too many open files")
		}
		return nil
	}
	_, err := r.Read(1, 5)
	var ge *GapError
	if !errors.As(err, &ge) {
		t.Fatalf("first read: want a GapError, got %v", err)
	}
	s.Inject = nil
	recs, err := r.Read(1, 5)
	if err != nil || len(recs) != 5 || recs[0].Seq != 1 {
		t.Fatalf("second read: %v %v", seqsOf(recs), err)
	}
	st, err := r.Stat()
	if err != nil || len(st.Holes) != 0 {
		t.Fatalf("stat: holes %v, err %v", st.Holes, err)
	}
}

// A transient I/O error during the full scan of a sealed segment is a fact about the moment,
// not about the file (decision 94): the Reader must scan it again on the next read.
func TestATransientScanErrorIsNotCachedByTheReader(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 200 })
	defer l.Close()
	appendN(t, l, 30)
	mustFlush(t, l)
	if st, err := l.Stat(); err != nil || st.Segments < 2 {
		t.Fatalf("want a sealed segment: %+v %v", st, err)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	seg0 := "/log/" + segmentName(0)
	opens := 0
	armed := true
	errOpen := errors.New("too many open files")
	s.Inject = func(op fsx.Op, name string) error {
		if armed && op == fsx.OpOpen && name == seg0 {
			opens++
			if opens == 2 { // the first open is the first-frame probe, the second is the scan
				return errOpen
			}
		}
		return nil
	}
	_, err := r.Read(1, 5)
	var ge *GapError
	if !errors.Is(err, errOpen) || errors.As(err, &ge) { // the error itself, not a gap a resume would skip (decision 79)
		t.Fatalf("first read: want the open error, got %v", err)
	}
	if opens != 2 {
		t.Fatalf("the injected error did not hit the scan: %d opens", opens)
	}
	armed = false
	for i := 0; i < 3; i++ {
		recs, err := r.Read(1, 5)
		if err != nil || len(recs) != 5 || recs[0].Seq != 1 {
			t.Fatalf("read %d after the error cleared: %v %v", i+2, seqsOf(recs), err)
		}
	}
}

func TestADamagedSegmentStaysCachedAsBadAcrossReaderCalls(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 200 })
	appendN(t, l, 30)
	mustFlush(t, l)
	_ = l.Close()
	seg0 := "/log/" + segmentName(0)
	if err := s.Remove(seg0); err != nil {
		t.Fatal(err)
	}
	if err := s.Mkdir(seg0, 0o755); err != nil { // a directory where a segment should be: not a regular file
		t.Fatalf("mkdir: %v", err)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	if _, err := r.Read(1, 5); err == nil {
		t.Fatal("first read: want a gap")
	}
	opens := 0
	s.Inject = func(op fsx.Op, name string) error {
		if name == seg0 {
			opens++
		}
		return nil
	}
	if _, err := r.Read(1, 5); err == nil {
		t.Fatal("second read: want a gap")
	}
	if opens != 0 {
		t.Fatalf("a damaged segment was examined again: %d operations on it", opens)
	}
}

// Reader.Scan, Reader.ID, Consumer.Scan and Consumer.Name each forward one argument to
// shared code; a swapped offset or limit would compile and pass every other test.
func TestReaderAndConsumerScanNameAndIDForwardTheRightArguments(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 5)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()

	scan := func(scanFn func(func(Record) bool) error) []Record {
		t.Helper()
		var out []Record
		if err := scanFn(func(rec Record) bool {
			rec.Payload = append([]byte(nil), rec.Payload...)
			out = append(out, rec)
			return true
		}); err != nil {
			t.Fatal(err)
		}
		return out
	}
	wantFrom := func(recs []Record, first, n uint64) {
		t.Helper()
		if uint64(len(recs)) != n || checkContiguous(t, recs, first) != first+n-1 {
			t.Fatalf("want %d records from %d, got %v", n, first, seqsOf(recs))
		}
		for _, rec := range recs {
			if string(rec.Payload) != string(payloadFor(rec.Seq)) {
				t.Fatalf("seq %d payload %q", rec.Seq, rec.Payload)
			}
		}
	}

	if r.ID() != l.ID() || r.ID() == "" {
		t.Fatalf("reader id %q, log id %q", r.ID(), l.ID())
	}
	wantFrom(scan(func(fn func(Record) bool) error { return r.Scan(1, 0, fn) }), 1, 5)
	wantFrom(scan(func(fn func(Record) bool) error { return r.Scan(3, 0, fn) }), 3, 3)
	wantFrom(scan(func(fn func(Record) bool) error { return r.Scan(2, 2, fn) }), 2, 2)

	for _, tc := range []struct {
		from string
		open func(string) (*Consumer, error)
	}{
		{"log", func(name string) (*Consumer, error) { return l.Consumer(name, ConsumerOptions{}) }},
		{"reader", func(name string) (*Consumer, error) { return r.Consumer(name, ConsumerOptions{}) }},
	} {
		name := "scan-" + tc.from
		c, err := tc.open(name)
		if err != nil {
			t.Fatal(err)
		}
		if c.Name() != name {
			t.Fatalf("%s: consumer name %q, want %q", tc.from, c.Name(), name)
		}
		wantFrom(scan(func(fn func(Record) bool) error { return c.Scan(0, fn) }), 1, 5)
		wantFrom(scan(func(fn func(Record) bool) error { return c.Scan(2, fn) }), 1, 2)
		if err := c.Commit(4); err != nil {
			t.Fatal(err)
		}
		wantFrom(scan(func(fn func(Record) bool) error { return c.Scan(0, fn) }), 4, 2)
		wantFrom(scan(func(fn func(Record) bool) error { return c.Scan(1, fn) }), 4, 1)
		if c.Offset() != 4 {
			t.Fatalf("%s: Scan moved the offset to %d", tc.from, c.Offset())
		}
		c.Close()
	}
}

// A sealed segment's first frame is probed with a small first read (firstFrameChunk), so a
// first frame that is larger than that read, or just under or over it, must still decode.
func TestAReaderReadsSealedSegmentsWhoseFirstFrameIsAroundAndAboveTheProbeSize(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 10 })
	defer l.Close()
	sizes := []int{10, firstFrameChunk - 20, firstFrameChunk - 8, firstFrameChunk, firstFrameChunk + 1, 3 * firstFrameChunk, readChunk + 5, 3 * readChunk}
	var want [][]byte
	for i, n := range sizes {
		p := bytes.Repeat([]byte{byte('a' + i)}, n)
		want = append(want, p)
		if _, err := l.Append(p); err != nil {
			t.Fatal(err)
		}
		mustFlush(t, l) // a flush past SegmentBytes seals the segment: the next record starts a new one
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) < len(sizes) {
		t.Fatalf("want at least %d segments, got %v", len(sizes), ids)
	}
	r := openReaderSim(t, s)
	defer r.Close()
	got, err := readerAll(r)
	if err != nil || len(got) != len(want) {
		t.Fatalf("read %d records of %d, %v", len(got), len(want), err)
	}
	for i := range want {
		if got[i].Seq != uint64(i+1) || !bytes.Equal(got[i].Payload, want[i]) {
			t.Fatalf("record %d: seq %d, %d bytes; want %d bytes", i+1, got[i].Seq, len(got[i].Payload), len(want[i]))
		}
	}
}

// A waiter parked past the head wakes for exactly one new record. It must not want a second
// one to arrive first: the wake condition is head >= offset, not head > offset, on the Reader's
// poll and on the iterators' wait for the next sequence (decision 63).
func TestSingleRecordWakesReadWaitAndFollowOnReaderAndLog(t *testing.T) {
	for _, kind := range []string{"reader-readwait", "reader-follow", "log-follow", "log-readwait"} {
		t.Run(kind, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
			defer l.Close()
			appendN(t, l, 1)
			mustFlush(t, l)
			r := openReaderSim(t, s)
			defer r.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			got := make(chan Record, 1)
			go func() {
				switch kind {
				case "reader-readwait":
					if recs, err := r.ReadWait(ctx, 2, 10); err == nil && len(recs) > 0 {
						got <- recs[0]
					}
				case "reader-follow":
					for rec, err := range r.Follow(ctx, 2) {
						if err == nil {
							got <- rec
						}
						return
					}
				case "log-readwait":
					if recs, err := l.ReadWait(ctx, 2, 10); err == nil && len(recs) > 0 {
						got <- recs[0]
					}
				case "log-follow":
					for rec, err := range l.Follow(ctx, 2) {
						if err == nil {
							got <- rec
						}
						return
					}
				}
			}()
			time.Sleep(30 * time.Millisecond) // let the waiter park past the head
			appendN(t, l, 1)                  // exactly one record: seq 2
			mustFlush(t, l)
			select {
			case rec := <-got:
				if rec.Seq != 2 {
					t.Fatalf("got seq %d, want 2", rec.Seq)
				}
			case <-time.After(time.Second):
				t.Fatal("a single new record was not delivered: the wait wants one more record first")
			}
		})
	}
}

// A marker that names more bytes than the active segment holds, or a first frame that does not
// decode below the marker, is corruption on every reader call (decisions 39 and 79). A bare
// io.ErrUnexpectedEOF would read as a transient I/O fact, and a consumer loop would retry it.
func TestAReaderCallsAnActiveSegmentThatFailsTheMarkerCorruptNotAnIOError(t *testing.T) {
	cases := map[string]func(data []byte) []byte{
		"the file is shorter than the marker's offset": func(data []byte) []byte { return data[:30] },
		"a byte of the first frame is flipped": func(data []byte) []byte {
			data[10] ^= 0xff
			return data
		},
	}
	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
			appendN(t, l, 5)
			mustFlush(t, l)
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			tl := readTail(t, s, "/log")
			path := "/log/" + segmentName(tl.Seg)
			data := fileBytes(t, s, path)
			if int64(len(data)) != int64(tl.Off) || tl.Off <= 30 {
				t.Fatalf("segment holds %d bytes, the marker says %d", len(data), tl.Off)
			}
			writeFile(t, s, path, damage(data))
			r := openReaderSim(t, s)
			defer r.Close()
			if _, err := r.Read(1, 10); !errors.Is(err, ErrCorrupt) || errors.Is(err, io.ErrUnexpectedEOF) {
				t.Errorf("Read: %v; want ErrCorrupt and not an unexpected EOF", err)
			}
			if _, err := r.Stat(); !errors.Is(err, ErrCorrupt) || errors.Is(err, io.ErrUnexpectedEOF) {
				t.Errorf("Stat: %v; want ErrCorrupt and not an unexpected EOF", err)
			}
		})
	}
}

func TestReaderCheckReportsUnboundedOnlyWhenNeitherCapIsSet(t *testing.T) {
	sf := &statfsSeam{free: 1 << 30}
	for _, tc := range []struct {
		name string
		mut  func(*Config)
		want bool
	}{
		{"no cap", func(*Config) {}, true},
		{"MaxBytes only", func(c *Config) { c.MaxBytes = 1 << 20 }, false},
		{"MinFreeBytes only", func(c *Config) { c.MinFreeBytes = 5000; c.statfs = sf.statfs }, false},
		{"both", func(c *Config) { c.MaxBytes = 1 << 20; c.MinFreeBytes = 5000; c.statfs = sf.statfs }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, tc.mut)
			defer l.Close()
			appendN(t, l, 2)
			mustFlush(t, l)
			r := openReaderSim(t, s)
			defer r.Close()
			res, err := r.Check()
			if err != nil || !res.Healthy || res.Unbounded != tc.want {
				t.Fatalf("Check: %+v %v, want Unbounded %v", res, err, tc.want)
			}
		})
	}
}
