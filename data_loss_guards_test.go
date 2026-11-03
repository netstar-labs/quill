package quill

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A directory that a marker says held records, but whose segment files are gone, is refused
// and never recreated as an empty log (decisions 39 and 62).
func TestOpenRefusesAMarkedLogWhoseSegmentFilesAreGone(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	appendN(t, l, 10)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	ids := segmentFiles(t, s, "/log")
	if len(ids) != 1 {
		t.Fatalf("setup: want one segment, got %v", ids)
	}
	if err := s.Remove("/log/" + segmentName(ids[0])); err != nil {
		t.Fatal(err)
	}
	_, err := Open(simConfig(s, "/log"))
	if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "no segment files") {
		t.Fatalf("open: %v, want ErrCorrupt: no segment files", err)
	}
	if got := segmentFiles(t, s, "/log"); len(got) != 0 {
		t.Fatalf("the refused open created segments %v", got)
	}
}

// Segments that hold bytes but no marker are refused: the marker is the only record of what
// was durable, so a fresh one would publish nothing and lose the log (decision 62).
func TestOpenRefusesSegmentsWithFramesAndNoMarker(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	appendN(t, l, 10)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove("/log/" + tailName); err != nil {
		t.Fatal(err)
	}
	_, err := Open(simConfig(s, "/log"))
	if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "no "+tailName) {
		t.Fatalf("open: %v, want ErrCorrupt: segments with frames but no %s", err, tailName)
	}
	if _, err := s.Lstat("/log/" + tailName); err == nil {
		t.Fatalf("the refused open wrote a %s", tailName)
	}
}

// A reader never serves a frame past the marker's sequence, even one that lies below the
// marker's offset and is intact: the marker alone publishes.
func TestAReaderStopsAtTheMarkersSequenceWhenFramesLieBelowItsOffset(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	appendN(t, l, 10)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	tl := readTail(t, s, "/log")
	if tl.Seq != 10 {
		t.Fatalf("setup: marker %+v", tl)
	}
	tl.Seq = 5 // Off still names the true end: five frames lie below it that it never published
	writeFile(t, s, "/log/"+tailName, encodeTail(tl))
	r := openReaderSim(t, s)
	defer r.Close()
	recs, err := readerAll(r)
	if err != nil || len(recs) != 5 || checkContiguous(t, recs, 1) != 5 {
		t.Fatalf("scan: %v %v, want exactly 1 to 5", seqsOf(recs), err)
	}
	recs, err = r.Read(3, 0)
	if err != nil || len(recs) != 3 || checkContiguous(t, recs, 3) != 5 {
		t.Fatalf("read from 3: %v %v, want exactly 3 to 5", seqsOf(recs), err)
	}
}

// Damage found at read time, after Open verified the log, is a gap over the sealed segment's
// range, and ErrCorrupt in the active segment below the durable offset (decision 63).
func TestDamageFoundAtReadTimeIsAGapInASealedSegmentAndCorruptInTheActiveOne(t *testing.T) {
	flip := func(t *testing.T, s *fsx.Sim, id uint64) {
		t.Helper()
		path := "/log/" + segmentName(id)
		b := fileBytes(t, s, path)
		b[0] ^= 0xff // the first frame: a read that starts there has no records to deliver first (decision 38)
		writeFile(t, s, path, b)
	}
	setup := func(t *testing.T) (*fsx.Sim, *Log) {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
		for len(l.sealed) < 1 || l.active.frames == 0 {
			appendN(t, l, 1)
			mustFlush(t, l)
		}
		return s, l
	}
	t.Run("sealed", func(t *testing.T) {
		s, l := setup(t)
		defer l.Close()
		sealed := l.sealed[0]
		flip(t, s, sealed.id)
		recs, err := l.Read(sealed.first, 0)
		var gap *GapError
		if !errors.As(err, &gap) || gap.From != sealed.first || gap.To != sealed.last || len(recs) != 0 {
			t.Fatalf("read: %v %v, want no records and a GapError %d to %d", seqsOf(recs), err, sealed.first, sealed.last)
		}
	})
	t.Run("active", func(t *testing.T) {
		s, l := setup(t)
		defer l.Close()
		flip(t, s, l.active.id)
		recs, err := l.Read(l.active.first, 0)
		if !errors.Is(err, ErrCorrupt) || len(recs) != 0 {
			t.Fatalf("read: %v %v, want no records and ErrCorrupt", seqsOf(recs), err)
		}
	})
}

// A sealed segment that is damaged while the active segment is empty cannot be told from a
// missing tail by looking at frames: the marker's sequence is the only witness, so the damaged
// range becomes a hole up to it and the head stays where the marker says (decisions 38 and 39).
func TestDamagedSealedSegmentsBeforeAnEmptyActiveSegmentBecomeHolesUpToTheMarker(t *testing.T) {
	// build leaves segments 0..sealed-1 holding four records each, an empty active segment, and a
	// marker at sequence 4*sealed with offset 0.
	build := func(t *testing.T, sealed int) *fsx.Sim {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
		for i := 0; i < sealed; i++ {
			appendN(t, l, 4)
			mustFlush(t, l)
		}
		appendN(t, l, 1) // opens the next segment; it is emptied below
		mustFlush(t, l)
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
		ids := segmentFiles(t, s, "/log")
		if len(ids) != sealed+1 {
			t.Fatalf("setup: segments %v, want %d", ids, sealed+1)
		}
		writeFile(t, s, "/log/"+segmentName(ids[sealed]), nil)
		tl := readTail(t, s, "/log")
		tl.Seq, tl.Seg, tl.Off = uint64(4*sealed), ids[sealed], 0
		writeFile(t, s, "/log/"+tailName, encodeTail(tl))
		return s
	}
	damage := func(t *testing.T, s *fsx.Sim, id uint64) {
		t.Helper()
		path := "/log/" + segmentName(id)
		writeFile(t, s, path, bytes.Repeat([]byte{0xA5}, len(fileBytes(t, s, path))))
	}
	check := func(t *testing.T, s *fsx.Sim, want Hole, head uint64) {
		t.Helper()
		l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
		defer l.Close()
		st, err := l.Stat()
		if err != nil || !slices.Equal(st.Holes, []Hole{want}) || st.Head != head {
			t.Fatalf("stat: holes %+v last %d, %v; want the one hole %+v and head %d", st.Holes, st.Head, err, want, head)
		}
		if seq, err := l.Append([]byte("next")); err != nil || seq != head+1 {
			t.Fatalf("append after the hole: %d, %v; want %d", seq, err, head+1)
		}
	}
	t.Run("the only sealed segment", func(t *testing.T) {
		s := build(t, 1)
		damage(t, s, 0)
		check(t, s, Hole{From: 1, To: 4}, 4)
	})
	t.Run("the newest of two sealed segments", func(t *testing.T) {
		s := build(t, 2)
		damage(t, s, 1)
		check(t, s, Hole{From: 5, To: 8}, 8)
	})
}

// Check says what the marker promises and the active segment lacks: the file is shorter than
// the offset the marker publishes as durable.
func TestCheckReportsAnActiveSegmentShorterThanTheMarkersOffset(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	appendN(t, l, 10)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	tl := readTail(t, s, "/log")
	path := "/log/" + segmentName(tl.Seg)
	data := fileBytes(t, s, path)
	short := int64(len(data)) - 3
	writeFile(t, s, path, data[:short])
	r := openReaderSim(t, s)
	defer r.Close()
	res, err := r.Check()
	if err != nil || res.Healthy || len(res.Problems) == 0 {
		t.Fatalf("Check: %+v, %v; want an unhealthy log", res, err)
	}
	p := res.Problems[0]
	want := fmt.Sprintf("%d bytes, the marker says %d are durable", short, tl.Off)
	if !p.Fatal || p.Kind != "segment" || p.Path != path || p.Offset != short || p.Intact != short || p.Detail != want {
		t.Fatalf("problem %+v, want a fatal segment problem at %d saying %q", p, short, want)
	}
}

// A quill.policy that is not valid is a warning, never a failure of the log, and the log
// counts as unbounded because nothing can be said to bound it.
func TestCheckWarnsAboutAPolicyThatCannotBeParsed(t *testing.T) {
	for name, body := range map[string]string{
		"not json":           "garbage",
		"no retention":       `{"max_bytes":5}`,
		"a bad duration":     `{"retention":"forever"}`,
		"a bad consumer ttl": `{"retention":"1h","consumer_ttl":"soon"}`,
	} {
		t.Run(name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, nil)
			appendN(t, l, 3)
			mustFlush(t, l)
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			writeFile(t, s, "/log/"+policyName, []byte(body))
			r := openReaderSim(t, s)
			defer r.Close()
			res, err := r.Check()
			if err != nil || !res.Healthy || !res.Unbounded || len(res.Problems) != 1 {
				t.Fatalf("Check: %+v, %v; want a healthy, unbounded log with one problem", res, err)
			}
			if p := res.Problems[0]; p.Kind != "policy" || p.Fatal || p.Detail != "cannot be parsed" || p.Path != "/log/"+policyName {
				t.Fatalf("problem %+v, want a non-fatal policy problem: cannot be parsed", p)
			}
		})
	}
}

// The marker's maxseg is the highest segment id ever created and is never lowered. After an
// Open that finds it above the active segment (the newest segments were removed or damaged),
// the next flush must republish the loaded value, not the active id: otherwise a crash before
// the next rotation lets a later start create a segment id that was already used.
func TestKillMarkerKeepsMaxSegAboveTheActiveSegment(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 2)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	tl := readTail(t, s, "/log")
	if tl.Seq != 2 || tl.Seg != 0 || tl.MaxSeg != 0 {
		t.Fatalf("setup: marker %+v, want seq 2 in segment 0 with maxseg 0", tl)
	}
	tl.MaxSeg = 9
	writeFile(t, s, "/log/"+tailName, encodeTail(tl))

	l = openSim(t, s, nil)
	defer l.Close()
	if _, err := l.Append(payloadFor(3)); err != nil {
		t.Fatal(err)
	}
	mustFlush(t, l)
	if got := readTail(t, s, "/log"); got.Seq != 3 || got.Seg != 0 || got.MaxSeg != 9 {
		t.Fatalf("published marker %+v, want seq 3 in segment 0 with maxseg 9 kept", got)
	}
}
