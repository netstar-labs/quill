package quill

import (
	"errors"
	"fmt"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A consumer ahead of the restored head is refused at open unless the caller asks to rewind it;
// the handle then commits any offset in [first, Head+1] and keeps the protect flag.
func TestRewindOpensAConsumerAheadOfTheHead(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	writeFile(t, s, "/log/consumers/ahead", encodeConsumer(consumerState{ID: logID(l.ID()), Next: 9, Protect: true}))
	if _, err := l.Consumer("ahead", ConsumerOptions{}); !errors.Is(err, ErrAhead) {
		t.Fatalf("without the option: %v", err)
	}
	c, err := l.Consumer("ahead", ConsumerOptions{Rewind: true})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Offset() != 9 || !c.Protected() {
		t.Fatalf("offset %d protected %v", c.Offset(), c.Protected())
	}
	if lag, err := c.Lag(); err != nil || lag != 0 {
		t.Fatalf("lag %d %v", lag, err)
	}
	if err := c.Commit(5); !errors.Is(err, ErrOffsetRange) || c.Offset() != 9 {
		t.Fatalf("commit past head+1: %v, offset %d", err, c.Offset())
	}
	if err := c.Commit(2); err != nil {
		t.Fatal(err)
	}
	if string(fileBytes(t, s, "/log/consumers/ahead")) != l.ID()+" 2 1\n" {
		t.Fatalf("file %q", fileBytes(t, s, "/log/consumers/ahead"))
	}
}

// The rewind accepts exactly [first, Head+1] and nothing outside it: first-1 and Head+2 are refused with
// ErrOffsetRange, and first, Head and Head+1 are accepted, each from a consumer that is ahead.
func TestRewindCommitAcceptsExactlyFirstThroughHeadPlusOne(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3) // first 1, head 3
	mustFlush(t, l)
	for i, c := range []struct {
		off uint64
		ok  bool
	}{{0, false}, {1, true}, {3, true}, {4, true}, {5, false}} {
		name := fmt.Sprintf("c%d", i)
		writeFile(t, s, "/log/consumers/"+name, encodeConsumer(consumerState{ID: logID(l.ID()), Next: 9, Protect: true}))
		cons, err := l.Consumer(name, ConsumerOptions{Rewind: true})
		if err != nil {
			t.Fatal(err)
		}
		err = cons.Commit(c.off)
		switch {
		case c.ok && err != nil:
			t.Errorf("Commit(%d) from an ahead consumer: %v, want accepted", c.off, err)
		case !c.ok && !errors.Is(err, ErrOffsetRange):
			t.Errorf("Commit(%d) from an ahead consumer: %v, want ErrOffsetRange", c.off, err)
		}
		if !c.ok && cons.Offset() != 9 {
			t.Errorf("a refused Commit(%d) moved the offset to %d", c.off, cons.Offset())
		}
		cons.Close()
	}
}
