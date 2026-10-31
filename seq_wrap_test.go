package quill

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Contiguity is "exactly one more than the previous frame", and previous+1 wraps to 0 at
// MaxUint64. A chain that climbs to MaxUint64 and carries on at 0, 1, 2 must not pass for a
// contiguous one, whether the frames sit below the marker's durable offset or only beyond it.
func TestWalkRefusesASequenceThatWrapsPastTheLastSequence(t *testing.T) {
	const top = uint64(math.MaxUint64)
	frames := func(seqs ...uint64) []byte {
		var img []byte
		for i, q := range seqs {
			rec := Record{Seq: q, At: int64(100 + i), Payload: smallPayload(i)}
			img = appendFrame(img, &rec)
		}
		return img
	}
	img := frames(top-2, top-1, top, 0, 1, 2)
	cases := []struct {
		name    string
		durable func(img []byte) int64
	}{
		{"all frames below the marker", func(img []byte) int64 { return int64(len(img)) }},
		{"marker before the wrap", func(img []byte) int64 { return int64(len(frames(top-2, top-1))) }},
		{"marker at the start", func(img []byte) int64 { return 0 }},
	}
	for _, c := range cases {
		s := fsx.NewSim()
		l := openSim(t, s, nil)
		if err := l.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
		writeFile(t, s, "/log/"+segmentName(0), img)
		off := c.durable(img)
		writeFile(t, s, "/log/quill.tail", encodeTail(tail{Seq: 2, Seg: 0, Off: uint64(off), MaxSeg: 0}))
		if l, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
			if err == nil {
				st, _ := l.Stat()
				t.Errorf("%s: Open accepted the wrapped chain: head %d first %d", c.name, st.Head, st.First)
				_ = l.Close()
				continue
			}
			t.Errorf("%s: Open -> %v, want ErrCorrupt", c.name, err)
		}
	}

	// walk itself: nothing may follow a frame numbered MaxUint64.
	w := &window{r: bytes.NewReader(img), size: int64(len(img)), chunk: windowChunk}
	info, err := walk(w, 0, 0, false, nil)
	if !errors.Is(err, errBadFrame) || info.last.Seq != top {
		t.Fatalf("walk -> last %d, err %v; want it to stop at %d with a bad frame", info.last.Seq, err, top)
	}
}
