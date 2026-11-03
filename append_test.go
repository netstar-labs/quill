package quill

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

func TestBatchSequencesAreConsecutiveAndABatchLandsInOneSegment(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 50 })
	defer l.Close()
	first, err := l.AppendBatch([][]byte{payloadFor(1), payloadFor(2), payloadFor(3)})
	if err != nil || first != 1 {
		t.Fatalf("%d %v", first, err)
	}
	seq, err := l.Append(payloadFor(4))
	if err != nil || seq != 4 {
		t.Fatalf("%d %v", seq, err)
	}
	mustFlush(t, l)
	// A batch larger than a segment still lands in one segment, which passes SegmentBytes.
	var big [][]byte
	for i := uint64(5); i <= 20; i++ {
		big = append(big, payloadFor(i))
	}
	if _, err := l.AppendBatch(big); err != nil {
		t.Fatal(err)
	}
	mustFlush(t, l)
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 3 { // 0 sealed by the first flush, 1 held the batch and was sealed, 2 active
		t.Fatalf("segments %v", ids)
	}
	if size := int64(len(fileBytes(t, s, "/log/"+segmentName(1)))); size <= 50 {
		t.Fatalf("segment 1 is %d bytes, should exceed SegmentBytes", size)
	}
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 20 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	if _, err := l.AppendBatch(nil); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("empty batch: %v", err)
	}
}

func TestABatchOverMaxBatchBytesAndAPayloadOverMaxPayloadBytesAreRefused(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.MaxBatchBytes = 1000; c.MaxPayloadBytes = 600 })
	defer l.Close()
	if _, err := l.AppendBatch([][]byte{make([]byte, 500), make([]byte, 500)}); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("batch: %v", err)
	}
	if _, err := l.AppendBatch([][]byte{make([]byte, 400), make([]byte, 400)}); err != nil {
		t.Fatalf("batch within: %v", err)
	}
	if _, err := l.Append(make([]byte, 601)); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("payload: %v", err)
	}
	if _, err := l.Append(make([]byte, 600)); err != nil {
		t.Fatalf("payload at the limit: %v", err)
	}
	if l.Head() != 0 {
		t.Fatal("refused records changed the head")
	}
	mustFlush(t, l)
	if l.Head() != 3 {
		t.Fatalf("head %d", l.Head())
	}
}

func TestARecordOfExactlyMaxPayloadBytesIsWrittenRecoveredAndRead(t *testing.T) {
	dir := t.TempDir()
	l := openDir(t, dir, func(c *Config) { c.SegmentBytes = 1 << 30 })
	payload := bytes.Repeat([]byte{0xab}, MaxPayloadBytes)
	if _, err := l.Append(payload); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Append(append(payload, 1)); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("one byte more: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	l = openDir(t, dir, func(c *Config) { c.SegmentBytes = 1 << 30; c.MaxReadBytes = 1 })
	defer l.Close()
	recs, err := l.Read(1, 10)
	if err != nil || len(recs) != 1 || !bytes.Equal(recs[0].Payload, payload) {
		t.Fatalf("read back %d records, %v", len(recs), err)
	}
}

func TestReadsFromAnyOffsetAcrossSegmentsAndWhileAppending(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 300; c.MaxReadBytes = 100 })
	defer l.Close()
	const n = 200
	for i := 0; i < n; i++ {
		appendN(t, l, 1)
		if i%7 == 6 {
			mustFlush(t, l)
		}
	}
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); len(ids) < 10 {
		t.Fatalf("want many segments, got %v", ids)
	}
	for offset := uint64(0); offset <= n+2; offset++ {
		recs, err := l.Read(offset, 5)
		if err != nil {
			t.Fatalf("read %d: %v", offset, err)
		}
		from := max(offset, 1)
		wantN := min(5, int(n-from+1))
		if from > n {
			wantN = 0
		}
		// MaxReadBytes of 100 bounds the count too, but always to at least one.
		if len(recs) == 0 && wantN > 0 || len(recs) > wantN {
			t.Fatalf("read %d: %d records, want up to %d", offset, len(recs), wantN)
		}
		checkContiguous(t, recs, from)
	}
	if recs, _ := l.Read(1, 0); len(recs) == 0 {
		t.Fatal("no limit")
	}
	// Unflushed records are invisible; flushed ones appear while appending continues.
	appendN(t, l, 3)
	if recs, _ := l.Read(n+1, 10); len(recs) != 0 {
		t.Fatalf("saw unflushed records %v", seqsOf(recs))
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			appendN(t, l, 2)
			mustFlush(t, l)
		}
	}()
	for i := 0; i < 50; i++ {
		recs, err := l.Read(1, 1000)
		if err != nil {
			t.Fatal(err)
		}
		checkContiguous(t, recs, 1)
	}
	wg.Wait()
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != n+103 {
		t.Fatalf("%d %v", len(recs), err)
	}
	// ScanAll and Read agree record for record.
	var scanned []uint64
	if err := l.ScanAll(func(r Record) bool { scanned = append(scanned, r.Seq); return true }); err != nil {
		t.Fatal(err)
	}
	if len(scanned) != len(recs) {
		t.Fatalf("scan %d read %d", len(scanned), len(recs))
	}
	// Scan payloads alias the buffer: a copy taken during the callback is right.
	if err := l.Scan(5, 1, func(r Record) bool {
		if string(r.Payload) != string(payloadFor(5)) {
			t.Errorf("scan payload %q", r.Payload)
		}
		return false
	}); err != nil {
		t.Fatal(err)
	}
}

func TestMaxReadBytesReturnsAtLeastOneRecordAndLimitCountsAcrossSegments(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100; c.MaxReadBytes = 1 })
	defer l.Close()
	for i := 0; i < 10; i++ {
		appendN(t, l, 1)
		mustFlush(t, l)
	}
	recs, err := l.Read(1, 10)
	if err != nil || len(recs) != 1 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	l.cfg.MaxReadBytes = 1 << 20
	recs, err = l.Read(1, 7)
	if err != nil || len(recs) != 7 || recs[6].Seq != 7 {
		t.Fatalf("limit across segments: %v %v", seqsOf(recs), err)
	}
}

func TestReadStopsOnceMaxReadBytesIsReachedSoTheLastRecordMayPassIt(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.MaxReadBytes = 16 })
	defer l.Close()
	for _, n := range []int{10, 10, 10} {
		if _, err := l.Append(make([]byte, n)); err != nil {
			t.Fatal(err)
		}
	}
	mustFlush(t, l)
	recs, err := l.Read(1, 0)
	total := 0
	for _, r := range recs {
		total += len(r.Payload)
	}
	// The second record is admitted because 10 < 16, and it carries the total to 20; the third is not.
	if err != nil || len(recs) != 2 || total != 20 {
		t.Fatalf("%v %d bytes %v", seqsOf(recs), total, err)
	}
}

func TestDescriptorsAcrossRotationAReadHoldsTheOldActiveSegmentOpen(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 100 })
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	l.mu.Lock()
	old := l.active
	l.mu.Unlock()
	_, release, err := l.view()
	if err != nil {
		t.Fatal(err)
	}
	appendN(t, l, 5) // rotates at this flush
	mustFlush(t, l)
	if !old.retired || old.f == nil {
		t.Fatalf("retired %v f nil %v", old.retired, old.f == nil)
	}
	if _, err := old.f.ReadAt(make([]byte, 1), 0); err != nil {
		t.Fatalf("the pinned descriptor: %v", err)
	}
	release()
	if old.f != nil {
		t.Fatal("the retired descriptor was not closed after the last reader left")
	}
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != 8 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
}

func TestReadWaitWakesOnAFlushAndEndsWithTheContextOrClose(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 1)
	mustFlush(t, l)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := l.ReadWait(ctx, 2, 10); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout: %v", err)
	}
	done := make(chan []Record, 1)
	go func() {
		recs, err := l.ReadWait(context.Background(), 2, 10)
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
		if checkContiguous(t, recs, 2) != 3 {
			t.Fatalf("%v", seqsOf(recs))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ReadWait did not wake")
	}
	errc := make(chan error, 1)
	go func() { _, err := l.ReadWait(context.Background(), 4, 10); errc <- err }()
	time.Sleep(20 * time.Millisecond)
	_ = l.Close()
	if err := <-errc; !errors.Is(err, ErrClosed) {
		t.Fatalf("after close: %v", err)
	}
	if _, err := l.Read(1, 1); !errors.Is(err, ErrClosed) {
		t.Fatalf("read after close: %v", err)
	}
	if _, err := l.Append(nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("append after close: %v", err)
	}
	if err := l.Close(); !errors.Is(err, ErrClosed) {
		t.Fatalf("double close: %v", err)
	}
}

func TestWhileFlushesFailAnAppendIsAcceptedOnlyIntoAnEmptyBufferOrWithinMaxBufferBytes(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.FlushBytes = 100; c.MaxBufferBytes = 300; c.MaxPayloadBytes = 1000 })
	defer l.Close()
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			return errENOSPC
		}
		return nil
	}
	// An empty buffer accepts any record, even one over MaxBufferBytes.
	if _, err := l.Append(make([]byte, 500)); err != nil {
		t.Fatalf("into an empty buffer: %v", err)
	}
	if err := l.Flush(); !errors.Is(err, errENOSPC) {
		t.Fatalf("flush: %v", err)
	}
	if _, err := l.Append(make([]byte, 10)); !errors.Is(err, errENOSPC) {
		t.Fatalf("beyond MaxBufferBytes while failing: %v", err)
	}
	s.Inject = nil
	mustFlush(t, l)
	if _, err := l.Append(make([]byte, 10)); err != nil {
		t.Fatalf("after recovery: %v", err)
	}
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpWrite {
			return errENOSPC
		}
		return nil
	}
	_ = l.Flush()
	if _, err := l.Append(make([]byte, 200)); err != nil {
		t.Fatalf("within MaxBufferBytes: %v", err)
	}
	if _, err := l.Append(make([]byte, 200)); !errors.Is(err, errENOSPC) {
		t.Fatalf("past MaxBufferBytes: %v", err)
	}
	s.Inject = nil
	mustFlush(t, l)
	if l.Head() != 3 {
		t.Fatalf("head %d", l.Head())
	}
}

func TestAnAppendWhoseInlineFlushFailsTheLogReturnsTheFailure(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.FlushBytes = 64 })
	defer l.Close()
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpSync && name == "/log/"+segmentName(0) {
			return errors.New("injected_fsync_EIO")
		}
		return nil
	}
	seq, err := l.Append(bytes.Repeat([]byte("x"), 200)) // crosses FlushBytes, so it flushes inline
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("Append whose inline fsync failed: %d, %v; want ErrFailed", seq, err)
	}
	if e := l.Err(); !errors.Is(e, ErrFailed) {
		t.Fatalf("Err: %v, want ErrFailed", e)
	}
	if _, err := l.Append([]byte("y")); !errors.Is(err, ErrFailed) {
		t.Fatalf("later Append: %v, want ErrFailed", err)
	}
}
