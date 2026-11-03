package quill

import (
	"slices"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// The writer indexes the buffer it has just made durable (indexBuffer) and a reopen indexes
// the same bytes by scanning them (walk). They are two implementations of one rule, one
// anchor per indexStride bytes and the first frame always, and the index only makes a seek
// cheap, so a drift between them changes no result a test of the public API can see. This
// pins them to each other. Every frame is exactly 64 bytes, so a frame starts at each stride
// boundary and an anchor taken one byte early or late lands on a different frame.
func TestWriterIndexEqualsTheScanIndex(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	open := func() *Log {
		return openRetention(t, s, clk, func(c *Config) { c.SegmentBytes = 4 << 20; c.Retention = -1 })
	}
	l := open()
	const frames, frameSize = 3000, 64
	for i := 1; i <= frames; i++ {
		// The payload length that makes this frame frameSize bytes, whatever the sequence's varint width.
		p := -1
		for n := 0; n < frameSize; n++ {
			if len(appendFrame(nil, &Record{Seq: uint64(i), At: clk.Now().Unix(), Payload: make([]byte, n)})) == frameSize {
				p = n
				break
			}
		}
		if p < 0 {
			t.Fatalf("no payload size makes frame %d %d bytes", i, frameSize)
		}
		if _, err := l.Append(make([]byte, p)); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	if err := l.Flush(); err != nil {
		t.Fatal(err)
	}
	written := slices.Clone(l.active.index)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if want := frames * frameSize / indexStride; len(written) != want+1 {
		t.Fatalf("writer index has %d anchors, want %d for %d bytes", len(written), want+1, frames*frameSize)
	}

	l = open()
	defer l.Close()
	if got := l.active.index; !slices.Equal(got, written) {
		t.Fatalf("scan index %+v differs from the writer's %+v", got, written)
	}
}
