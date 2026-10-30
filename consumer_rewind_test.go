package quill

import (
	"errors"
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
