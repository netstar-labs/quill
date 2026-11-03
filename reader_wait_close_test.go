package quill

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A ReadWait parked on a Reader ends with ErrClosed when the Reader closes, even with a context
// that never ends, and a second Close returns ErrClosed (the Reader.ReadWait and Reader.Close
// docs; decision 63). Without the closed check in the wait loop only the context could end it.
func TestAParkedReaderReadWaitReturnsErrClosedOnCloseAndASecondCloseIsErrClosed(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 1)
	mustFlush(t, l)
	r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	// If the wait never ends, cancelling releases the goroutine so the failure is reported
	// instead of leaking it.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	go func() { _, err := r.ReadWait(ctx, 2, 10); errc <- err }()
	select {
	case err := <-errc:
		t.Fatalf("ReadWait returned before the Reader closed: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	if err := r.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	select {
	case err := <-errc:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("parked ReadWait after close: %v", err)
		}
	case <-time.After(time.Second):
		cancel()
		t.Fatalf("a parked ReadWait did not end when the Reader closed (it ended with %v only once the context did)", <-errc)
	}
	if err := r.Close(); !errors.Is(err, ErrClosed) {
		t.Fatalf("second close: %v", err)
	}
}
