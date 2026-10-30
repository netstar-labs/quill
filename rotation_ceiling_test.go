package quill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// While rotation keeps failing the log keeps appending to the active segment (decision 60 d),
// but only up to SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes+34);
// past that an append is refused with the rotation error (Stage 2 audit A1-25).
func TestAFailedRotationRefusesAppendsAtTheCeilingAndRecoversWhenItSucceeds(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) {
		c.SegmentBytes, c.FlushBytes, c.MaxBufferBytes = 100, 64, 128
		c.MaxPayloadBytes, c.MaxBatchBytes = 64, 128
	})
	defer l.Close()
	const ceiling = 100 + 128 + 128
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpCreate && name == "/log/"+segmentName(1) {
			return errENOSPC
		}
		return nil
	}
	appendN(t, l, 8)
	if err := l.Flush(); !errors.Is(err, errENOSPC) {
		t.Fatalf("flush: %v", err)
	}
	var refused error
	for i := 0; i < 200 && refused == nil; i++ {
		if _, refused = l.Append(payloadFor(l.nextSeqForTest())); refused == nil {
			_ = l.Flush()
		}
	}
	if !errors.Is(refused, errENOSPC) {
		t.Fatalf("no append was refused with the rotation error; active segment is %d bytes, ceiling %d (err %v)",
			len(fileBytes(t, s, "/log/"+segmentName(0))), ceiling, refused)
	}
	_ = l.Flush()
	if size := len(fileBytes(t, s, "/log/"+segmentName(0))); size > ceiling {
		t.Fatalf("active segment is %d bytes, over the ceiling %d", size, ceiling)
	}
	// A refused append assigned nothing, and once rotation works again appends are accepted.
	next := l.nextSeqForTest()
	s.Inject = nil
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("segments after recovery %v", ids)
	}
	if seq, err := l.Append(payloadFor(next)); err != nil || seq != next {
		t.Fatalf("append after recovery: %d %v, want %d", seq, err, next)
	}
	mustFlush(t, l)
	recs, err := readAll(l)
	if err != nil || checkContiguous(t, recs, 1) != next {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
}

func TestConfigRefusesASegmentThatAFlushCouldPushPastTheFileLimit(t *testing.T) {
	for _, tc := range []struct {
		name string
		mut  func(*Config)
		ok   bool
	}{
		{"the largest segment with defaults", func(c *Config) { c.SegmentBytes = MaxSegmentBytes }, true},
		{"a large flush buffer", func(c *Config) { c.SegmentBytes, c.FlushBytes = MaxSegmentBytes, 600<<20 }, false},
		{"a large buffer bound", func(c *Config) { c.SegmentBytes, c.MaxBufferBytes = MaxSegmentBytes, 600<<20 }, false},
		{"a large buffer bound under a small segment", func(c *Config) { c.MaxBufferBytes = 600 << 20 }, true},
	} {
		c := Config{Dir: "/log"}
		tc.mut(&c)
		err := c.withDefaults().validate()
		if tc.ok != (err == nil) || (err != nil && !errors.Is(err, ErrConfig)) {
			t.Errorf("%s: ok=%v, got %v", tc.name, tc.ok, err)
		}
	}
}

// A segment file over the size limit is intact but refused whole, so no path that reports it
// says "damaged at byte 0 (intact to byte 0)": not Open, and not Check (Stage 2 audit A1-25).
func TestAnOversizeIntactSegmentFileIsReportedAsRefusedNotDamagedAtByteZero(t *testing.T) {
	dir := t.TempDir()
	l := openDir(t, dir, nil)
	appendN(t, l, 3)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	seg := filepath.Join(dir, segmentName(0))
	if err := os.Truncate(seg, 1700<<20); err != nil { // a sparse file: nothing is written
		t.Fatal(err)
	}
	const want = "refused without being read"
	_, err := Open(Config{Dir: dir})
	var se *SegmentError
	if !errors.Is(err, ErrCorrupt) || !errors.As(err, &se) || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "damaged at byte") {
		t.Fatalf("Open: %v", err)
	}
	r, rerr := OpenReader(dir, ReaderOptions{})
	if rerr != nil {
		t.Fatal(rerr)
	}
	defer r.Close()
	res, cerr := r.Check()
	if cerr != nil || res.Healthy || len(res.Problems) == 0 {
		t.Fatalf("Check: %+v %v", res, cerr)
	}
	for _, p := range res.Problems {
		if line := p.String(); p.Kind == "segment" && (strings.Contains(line, "damaged at byte") || !strings.Contains(line, want) || !strings.Contains(line, "over the 1610612736 limit")) {
			t.Errorf("problem line %q", line)
		}
	}
	if res.Problems[0].Kind != "segment" || !res.Problems[0].Fatal {
		t.Fatalf("first problem %+v", res.Problems[0])
	}
}
