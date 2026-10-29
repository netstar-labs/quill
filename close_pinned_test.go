package quill

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// closeUnderRead starts a Read across a sealed and the active segment, stalls it on the first
// OpRead of the active segment (or of the first sealed one), closes the log while it is stalled, then lets it go.
func closeUnderRead(t *testing.T, stallActive bool) {
	t.Helper()
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	for len(segmentFiles(t, s, "/log")) < 2 {
		appendN(t, l, 4)
		mustFlush(t, l)
	}
	appendN(t, l, 2) // the active segment holds records, so the read ends in it
	mustFlush(t, l)
	l.mu.Lock()
	activeFrames := l.active.frames
	l.mu.Unlock()
	if activeFrames == 0 {
		t.Fatal("setup: the active segment is empty")
	}
	head := l.Head()
	ids := segmentFiles(t, s, "/log")
	stallID := ids[0]
	if stallActive {
		stallID = ids[len(ids)-1]
	}
	stallName := filepath.Join("/log", segmentName(stallID))
	entered, proceed := make(chan struct{}), make(chan struct{})
	var once bool
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpRead && name == stallName && !once {
			once = true
			close(entered)
			<-proceed
		}
		return nil
	}
	type result struct {
		recs []Record
		err  error
		pan  any
	}
	done := make(chan result, 1)
	go func() {
		var r result
		defer func() { r.pan = recover(); done <- r }()
		r.recs, r.err = l.Read(1, 0)
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the read never reached the stalled segment")
	}
	if err := l.Close(); err != nil {
		t.Fatalf("close under a read: %v", err)
	}
	close(proceed)
	r := <-done
	s.Inject = nil
	if r.pan != nil {
		t.Fatalf("a read racing Close panicked: %v", r.pan)
	}
	if r.err != nil && !errors.Is(r.err, ErrClosed) {
		t.Fatalf("a read racing Close returned %v, want nil or ErrClosed", r.err)
	}
	if r.err == nil {
		checkContiguous(t, r.recs, 1)
		if got := r.recs[len(r.recs)-1].Seq; got != head {
			t.Fatalf("read ended at %d, want %d", got, head)
		}
	}
	if n := s.OpenFiles(); n != 0 {
		t.Fatalf("%d descriptors leaked after Close and the read's release", n)
	}
}

func TestCloseWhileAReadIsPinnedOnASealedSegmentDoesNotPanicOrLeakTheActiveDescriptor(t *testing.T) {
	closeUnderRead(t, false)
}

func TestCloseWhileAReadIsPinnedOnTheActiveSegmentLetsTheReadFinishAndReleasesTheDescriptor(t *testing.T) {
	closeUnderRead(t, true)
}
