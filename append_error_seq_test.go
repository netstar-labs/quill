package quill

import (
	"bytes"
	"errors"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// failDataFsync makes the fsync of the first segment fail, so the log fails (decision 37).
func failDataFsync(s *fsx.Sim) {
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpSync && name == "/log/"+segmentName(0) {
			return errors.New("injected_fsync_EIO")
		}
		return nil
	}
}

// An append that fails after sequences were assigned returns the first of them with the error,
// so the caller can look for the records after a reopen; one that fails before returns 0
// (Stage 2 audit L1-5).
func TestAnAppendThatFailsAfterAssigningSequencesReturnsTheFirstWithTheError(t *testing.T) {
	t.Run("Append, inline flush", func(t *testing.T) {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.FlushBytes = 100 })
		defer l.Close()
		failDataFsync(s)
		seq, err := l.Append(bytes.Repeat([]byte("x"), 200))
		if !errors.Is(err, ErrFailed) || seq != 1 {
			t.Fatalf("Append: (%d, %v), want (1, ErrFailed)", seq, err)
		}
	})
	t.Run("AppendBatch, inline flush", func(t *testing.T) {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.FlushBytes = 100 })
		defer l.Close()
		if _, err := l.Append([]byte("a")); err != nil {
			t.Fatal(err)
		}
		failDataFsync(s)
		batch := [][]byte{bytes.Repeat([]byte("x"), 60), bytes.Repeat([]byte("y"), 60), []byte("z")}
		first, err := l.AppendBatch(batch)
		if !errors.Is(err, ErrFailed) || first != 2 {
			t.Fatalf("AppendBatch: (%d, %v), want (2, ErrFailed)", first, err)
		}
	})
	t.Run("SyncAppends", func(t *testing.T) {
		s := fsx.NewSim()
		l := openSim(t, s, func(c *Config) { c.FlushBytes = 1 << 20; c.SyncAppends = true })
		defer l.Close()
		failDataFsync(s)
		seq, err := l.Append([]byte("x")) // under FlushBytes: the failure comes from the Sync
		if !errors.Is(err, ErrFailed) || seq != 1 {
			t.Fatalf("SyncAppends Append: (%d, %v), want (1, ErrFailed)", seq, err)
		}
	})
}

func TestAnAppendThatAssignsNoSequenceReturnsZeroWithTheError(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.FlushBytes = 40; c.MaxPayloadBytes = 64; c.MaxBatchBytes = 256 })
	big := bytes.Repeat([]byte("x"), 65)
	if seq, err := l.Append(big); !errors.Is(err, ErrPayloadTooLarge) || seq != 0 {
		t.Fatalf("oversize payload: (%d, %v)", seq, err)
	}
	if seq, err := l.AppendBatch(nil); !errors.Is(err, ErrPayloadTooLarge) || seq != 0 {
		t.Fatalf("empty batch: (%d, %v)", seq, err)
	}
	many := make([][]byte, 10)
	for i := range many {
		many[i] = make([]byte, 40)
	}
	if seq, err := l.AppendBatch(many); !errors.Is(err, ErrPayloadTooLarge) || seq != 0 {
		t.Fatalf("refused batch: (%d, %v)", seq, err)
	}
	failDataFsync(s)
	if _, err := l.Append(bytes.Repeat([]byte("x"), 64)); !errors.Is(err, ErrFailed) {
		t.Fatalf("setup: %v", err)
	}
	if seq, err := l.Append([]byte("y")); !errors.Is(err, ErrFailed) || seq != 0 {
		t.Fatalf("append on a failed log: (%d, %v)", seq, err)
	}
	s.Inject = nil
	_ = l.Close()
	if seq, err := l.Append([]byte("z")); !errors.Is(err, ErrClosed) || seq != 0 {
		t.Fatalf("append on a closed log: (%d, %v)", seq, err)
	}
}
