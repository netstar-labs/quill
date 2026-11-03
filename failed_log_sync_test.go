package quill

import (
	"context"
	"errors"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// On a failed log, Sync is not "always an error": a record that became durable before the failure
// stays durable, so its Sync is nil; a record assigned but not yet durable is unknown, not
// lost (decision 42), so its Sync is ErrFailed; and a sequence never assigned is ErrBadSeq, as on
// any log (decisions 37, 42; Stage 2 audit L1-4).
func TestSyncOnAFailedLogDistinguishesDurableUndurableAndUnassigned(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)  // sequences 1 and 2 are durable
	appendN(t, l, 1) // sequence 3 is assigned and buffered
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpSync && name == "/log/"+segmentName(0) {
			return errors.New("injected_fsync_EIO")
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Flush: %v, want ErrFailed", err)
	}
	if err := l.Err(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Err: %v, want ErrFailed", err)
	}
	ctx := context.Background()
	if err := l.Sync(ctx, 1); err != nil {
		t.Fatalf("Sync of a sequence durable before the failure: %v, want nil", err)
	}
	if err := l.Sync(ctx, 2); err != nil {
		t.Fatalf("Sync of the last durable sequence: %v, want nil", err)
	}
	if err := l.Sync(ctx, 3); !errors.Is(err, ErrFailed) {
		t.Fatalf("Sync of an assigned sequence that is not durable: %v, want ErrFailed", err)
	}
	if err := l.Sync(ctx, 4); !errors.Is(err, ErrBadSeq) || errors.Is(err, ErrFailed) {
		t.Fatalf("Sync of the next unassigned sequence: %v, want ErrBadSeq and not ErrFailed", err)
	}
	if err := l.Sync(ctx, 99); !errors.Is(err, ErrBadSeq) || errors.Is(err, ErrFailed) {
		t.Fatalf("Sync of a far unassigned sequence: %v, want ErrBadSeq and not ErrFailed", err)
	}
	// Append and Flush, by contrast, fail with ErrFailed when valid; a request refused for its
	// own reasons first keeps that error.
	if _, err := l.Append(make([]byte, l.cfg.MaxPayloadBytes+1)); !errors.Is(err, ErrPayloadTooLarge) || errors.Is(err, ErrFailed) {
		t.Fatalf("Append of an oversized payload: %v, want ErrPayloadTooLarge and not ErrFailed", err)
	}
	if _, err := l.AppendBatch(nil); !errors.Is(err, ErrPayloadTooLarge) || errors.Is(err, ErrFailed) {
		t.Fatalf("AppendBatch of nothing: %v, want ErrPayloadTooLarge and not ErrFailed", err)
	}
	if _, err := l.Append([]byte("z")); !errors.Is(err, ErrFailed) {
		t.Fatalf("later Append: %v, want ErrFailed", err)
	}
	if err := l.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("later Flush: %v, want ErrFailed", err)
	}
}
