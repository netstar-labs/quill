package quill

import (
	"bytes"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A damaged sealed segment, an empty newest segment and a marker still at sequence 0 (a first
// rotation whose marker write was lost) leave nothing to be missing: the log has no records and
// the marker promises none. No hole is reported, as holes.go says, and the writer and the
// Reader agree; the damage is still named by Err.
func TestADamagedSealedSegmentBesideAMarkerAtZeroReportsNoHole(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if tl := readTail(t, s, "/log"); tl.Seq != 0 {
		t.Fatalf("setup: marker at %d, want 0", tl.Seq)
	}
	writeFile(t, s, "/log/"+segmentName(0), bytes.Repeat([]byte{0xA5}, 21))
	writeFile(t, s, "/log/"+segmentName(1), nil)

	r := openReaderSim(t, s)
	defer r.Close()
	rst, err := r.Stat()
	if err != nil || len(rst.Holes) != 0 {
		t.Fatalf("reader stat: holes %+v, %v; want none", rst.Holes, err)
	}

	l = openSim(t, s, nil)
	defer l.Close()
	st, err := l.Stat()
	if err != nil || len(st.Holes) != 0 || st.Head != 0 || st.Records != 0 {
		t.Fatalf("writer stat: holes %+v head %d records %d, %v; want no hole, head 0, no records", st.Holes, st.Head, st.Records, err)
	}
	if l.Err() == nil {
		t.Fatal("Err() is nil, want it to name the damaged segment")
	}
}
