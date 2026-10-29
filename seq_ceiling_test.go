package quill

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// simLogWithHead returns a simulated file system holding a closed log whose n frames end at
// sequence head, built by hand: the format accepts any non-zero first sequence.
func simLogWithHead(t *testing.T, head uint64, n int) *fsx.Sim {
	t.Helper()
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	if err := l.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	img := image(head-uint64(n)+1, n, 100, smallPayload)
	writeFile(t, s, "/log/"+segmentName(0), img)
	writeFile(t, s, "/log/quill.tail", encodeTail(tail{Seq: head, Seg: 0, Off: uint64(len(img)), MaxSeg: 0}))
	return s
}

// The sequence ceiling is MaxUint64-1: the last sequence is never assigned, so last+1 (the
// next-to-assign counter, a read cursor, a consumer offset) never wraps to 0.
func TestSeqCeilingRefusesTheLastSequence(t *testing.T) {
	const top = uint64(math.MaxUint64)
	for _, syncAppends := range []bool{false, true} {
		s := simLogWithHead(t, top-2, 2)
		cfg := simConfig(s, "/log")
		cfg.SyncAppends = syncAppends
		l, err := Open(cfg)
		if err != nil {
			t.Fatalf("SyncAppends=%v: open: %v", syncAppends, err)
		}
		// A batch whose last record would be numbered top is refused whole.
		if first, err := l.AppendBatch([][]byte{[]byte("a"), []byte("b")}); !errors.Is(err, ErrSeqExhausted) {
			t.Fatalf("SyncAppends=%v: batch ending at MaxUint64 -> (%d, %v), want ErrSeqExhausted", syncAppends, first, err)
		}
		// The last assignable sequence is top-1 and it behaves like any other.
		seq, err := l.Append([]byte("last"))
		if err != nil || seq != top-1 {
			t.Fatalf("SyncAppends=%v: append -> (%d, %v), want (%d, nil)", syncAppends, seq, err, top-1)
		}
		if err := l.Sync(context.Background(), top-1); err != nil {
			t.Fatalf("SyncAppends=%v: Sync(%d): %v", syncAppends, top-1, err)
		}
		if got := l.Head(); got != top-1 {
			t.Fatalf("SyncAppends=%v: head %d, want %d", syncAppends, got, top-1)
		}
		recs, err := l.Read(top-1, 5)
		if err != nil || len(recs) != 1 || recs[0].Seq != top-1 || string(recs[0].Payload) != "last" {
			t.Fatalf("SyncAppends=%v: Read(%d) -> %v, %v", syncAppends, top-1, recs, err)
		}
		if _, err := l.Append([]byte("more")); !errors.Is(err, ErrSeqExhausted) {
			t.Fatalf("SyncAppends=%v: append past the ceiling: %v, want ErrSeqExhausted", syncAppends, err)
		}
		// Only what was assigned can be synced.
		if err := l.Sync(context.Background(), top); !errors.Is(err, ErrBadSeq) {
			t.Fatalf("SyncAppends=%v: Sync(MaxUint64): %v, want ErrBadSeq", syncAppends, err)
		}
		if err := l.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
		// Reopened, the log stays at the ceiling.
		l, err = Open(cfg)
		if err != nil {
			t.Fatalf("SyncAppends=%v: reopen: %v", syncAppends, err)
		}
		if _, err := l.Append([]byte("x")); !errors.Is(err, ErrSeqExhausted) {
			t.Fatalf("SyncAppends=%v: append after reopen: %v, want ErrSeqExhausted", syncAppends, err)
		}
		_ = l.Close()
	}
}

// A log whose head is already MaxUint64 (crafted or restored: the format accepts any non-zero
// first sequence) breaks the same rule from the other side, so Open and OpenReader refuse it as
// corrupt rather than serve a log where Sync, Read and a consumer offset would have to wrap.
func TestSeqCeilingRefusesALogThatEndsAtTheLastSequence(t *testing.T) {
	const top = uint64(math.MaxUint64)
	cases := []struct {
		name   string
		marker uint64 // what quill.tail names as durable
	}{
		{"marker at the top", top},
		{"marker behind the frames", top - 1},
	}
	for _, c := range cases {
		s := simLogWithHead(t, top, 2)
		img := image(top-1, 2, 100, smallPayload)
		writeFile(t, s, "/log/quill.tail", encodeTail(tail{Seq: c.marker, Seg: 0, Off: uint64(len(img)), MaxSeg: 0}))
		if l, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
			if err == nil {
				_ = l.Close()
			}
			t.Fatalf("%s: Open -> %v, want ErrCorrupt", c.name, err)
		}
	}
	// A reader names the head from the marker alone.
	s := simLogWithHead(t, top, 2)
	if r, err := OpenReader("/log", ReaderOptions{fs: s}); !errors.Is(err, ErrCorrupt) {
		if err == nil {
			_ = r.Close()
		}
		t.Fatalf("OpenReader -> %v, want ErrCorrupt", err)
	}
}
