package quill

import (
	"errors"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// The write-path size limits at their exact values. A limit tested far from its edge lets a
// one-byte error in the accounting through: decision 46 quotes the batch bound as numbers and
// decision 69's disk bound depends on the 34-byte over-count.

// fixedClock makes every record's At the same two-byte varint, so a test can compute a frame's
// exact length with frameLen instead of observing it.
func fixedClock() time.Time { return time.Unix(1000, 0) }

// The batch bound sums len(payload)+34 over the records (decision 46), so two records of 466
// bytes come to exactly MaxBatchBytes 1000 and are accepted, and one more byte is refused.
func TestABatchIsAcceptedAtExactlyMaxBatchBytesAndRefusedAtOneByteMore(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.MaxBatchBytes = 1000 })
	defer l.Close()
	if _, err := l.AppendBatch([][]byte{make([]byte, 467), make([]byte, 466)}); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("batch of 1001 by the bound: %v", err)
	}
	if l.Head() != 0 || l.nextSeqForTest() != 1 {
		t.Fatal("a refused batch assigned sequences")
	}
	if first, err := l.AppendBatch([][]byte{make([]byte, 466), make([]byte, 466)}); err != nil || first != 1 {
		t.Fatalf("batch of exactly 1000 by the bound: %d %v", first, err)
	}
	// Three records: 3*34 of overhead, so 898 payload bytes make 1000.
	if _, err := l.AppendBatch([][]byte{make([]byte, 300), make([]byte, 300), make([]byte, 299)}); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("three records of 1001 by the bound: %v", err)
	}
	if _, err := l.AppendBatch([][]byte{make([]byte, 300), make([]byte, 299), make([]byte, 299)}); err != nil {
		t.Fatalf("three records of exactly 1000 by the bound: %v", err)
	}
}

// While flushes fail, an append into a non-empty buffer is accepted when the buffer plus the
// record's len+34 bound is exactly MaxBufferBytes and refused at one byte over (decision 69).
func TestWhileFlushesFailAnAppendIsAcceptedAtExactlyMaxBufferBytesAndRefusedAtOneByteOver(t *testing.T) {
	const first, second = 10, 20
	b := frameLen(&Record{Seq: 1, At: fixedClock().Unix(), Payload: make([]byte, first)})
	for _, tc := range []struct {
		name string
		max  int
		ok   bool
	}{
		{"exactly at the bound", b + second + 34, true},
		{"one byte over the bound", b + second + 33, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.clock = fixedClock; c.FlushBytes = 50; c.MaxBufferBytes = tc.max })
			defer l.Close()
			s.Inject = func(op fsx.Op, name string) error {
				if op == fsx.OpWrite {
					return errENOSPC
				}
				return nil
			}
			if _, err := l.Append(make([]byte, first)); err != nil {
				t.Fatal(err)
			}
			if err := l.Flush(); !errors.Is(err, errENOSPC) {
				t.Fatalf("flush: %v", err)
			}
			l.mu.Lock()
			buffered := len(l.buf)
			l.mu.Unlock()
			if buffered != b {
				t.Fatalf("buffer holds %d bytes, the test assumed %d", buffered, b)
			}
			_, err := l.Append(make([]byte, second))
			if tc.ok && err != nil {
				t.Fatalf("refused at the bound: %v", err)
			}
			if !tc.ok && !errors.Is(err, errENOSPC) {
				t.Fatalf("accepted one byte over the bound: %v", err)
			}
		})
	}
}

// The ceiling is inclusive: while rotation keeps failing, an append that brings the active
// segment to exactly SegmentBytes + max(FlushBytes, MaxBufferBytes) + max(MaxBatchBytes,
// MaxPayloadBytes+34) is accepted and one byte more is refused with the rotation error.
func TestAFailedRotationAcceptsAnAppendReachingExactlyTheCeilingAndRefusesOneByteMore(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) {
		c.SegmentBytes, c.FlushBytes, c.MaxBufferBytes = 100, 64, 64
		c.MaxPayloadBytes, c.MaxBatchBytes = 300, 64
	})
	defer l.Close()
	const ceiling = 100 + 64 + 300 + 34
	if got := l.cfg.maxActiveBytes(); got != ceiling {
		t.Fatalf("ceiling %d, the test assumed %d", got, ceiling)
	}
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpCreate && name == "/log/"+segmentName(1) {
			return errENOSPC
		}
		return nil
	}
	state := func() (durable int64, buffered int, rotErr error) {
		l.mu.Lock()
		defer l.mu.Unlock()
		return l.active.durable, len(l.buf), l.rotErr
	}
	// Grow the durable part, one flushed record at a time, until one payload of at most 250
	// bytes (so one byte more is still under MaxPayloadBytes) lands the segment on the ceiling.
	for {
		d, _, _ := state()
		if ceiling-34-d <= 250 {
			break
		}
		appendN(t, l, 1)
		_ = l.Flush()
	}
	d, buffered, rotErr := state()
	if buffered != 0 || !errors.Is(rotErr, errENOSPC) {
		t.Fatalf("buffered %d, rotation error %v: the setup did not leave a failing rotation", buffered, rotErr)
	}
	p := int(ceiling - 34 - d)
	if _, err := l.Append(make([]byte, p+1)); !errors.Is(err, errENOSPC) {
		t.Fatalf("an append reaching the ceiling plus one byte: %v", err)
	}
	if _, err := l.Append(make([]byte, p)); err != nil {
		t.Fatalf("an append reaching exactly the ceiling: %v", err)
	}
}

// An inline flush happens when the buffer reaches FlushBytes: a record whose frame is exactly
// FlushBytes is durable when Append returns; with one byte more room it stays buffered.
func TestAnAppendThatFillsTheBufferToExactlyFlushBytesFlushesInline(t *testing.T) {
	n := frameLen(&Record{Seq: 1, At: fixedClock().Unix(), Payload: make([]byte, 30)})
	for _, tc := range []struct {
		name     string
		flush    int
		wantHead uint64
	}{
		{"frame equals FlushBytes", n, 1},
		{"frame is one byte under FlushBytes", n + 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.clock = fixedClock; c.FlushBytes = tc.flush })
			defer l.Close()
			if _, err := l.Append(make([]byte, 30)); err != nil {
				t.Fatal(err)
			}
			if h := l.Head(); h != tc.wantHead {
				t.Fatalf("head %d after the append, want %d", h, tc.wantHead)
			}
		})
	}
}

// A segment rotates when a flush leaves it with SegmentBytes or more: exactly SegmentBytes
// rotates, one byte less does not.
func TestASegmentRotatesAtExactlySegmentBytesAndNotAtOneByteUnder(t *testing.T) {
	n := frameLen(&Record{Seq: 1, At: fixedClock().Unix(), Payload: make([]byte, 30)})
	for _, tc := range []struct {
		name string
		seg  int64
		segs int
	}{
		{"segment holds exactly SegmentBytes", int64(n), 2},
		{"segment is one byte under SegmentBytes", int64(n) + 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.clock = fixedClock; c.SegmentBytes = tc.seg })
			defer l.Close()
			if _, err := l.Append(make([]byte, 30)); err != nil {
				t.Fatal(err)
			}
			mustFlush(t, l)
			if ids := segmentFiles(t, s, "/log"); len(ids) != tc.segs {
				t.Fatalf("segments %v, want %d", ids, tc.segs)
			}
		})
	}
}
