package quill

import (
	"bytes"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// An offset file that cannot be read (an I/O or permission error, as opposed to one that is
// not there) is an error from Consumer and never a reason to start a new consumer at the
// first retained sequence: that would overwrite the stored offset and the consumer would
// reprocess the log. The file is not touched.
func TestAnUnreadableConsumerFileIsAnErrorAndNeverResetsTheOffset(t *testing.T) {
	for _, op := range []fsx.Op{fsx.OpOpen, fsx.OpLstat, fsx.OpRead} {
		for _, errno := range []error{syscall.EIO, syscall.EACCES} {
			s := fsx.NewSim()
			l := openSim(t, s, nil)
			appendN(t, l, 6)
			mustFlush(t, l)
			stored := encodeConsumer(consumerState{ID: logID(l.ID()), Next: 4, Protect: true})
			writeFile(t, s, "/log/consumers/x", stored)
			s.Inject = func(o fsx.Op, name string) error {
				if o == op && name == "/log/consumers/x" {
					return errno
				}
				return nil
			}
			c, err := l.Consumer("x", ConsumerOptions{})
			if !errors.Is(err, errno) {
				t.Fatalf("%s %v: err %v, want it to wrap %v", op, errno, err, errno)
			}
			if c != nil {
				_ = c.Close()
			}
			s.Inject = nil
			if got := fileBytes(t, s, "/log/consumers/x"); !bytes.Equal(got, stored) {
				t.Fatalf("%s %v: the file is now %q, was %q", op, errno, got, stored)
			}
			// The failed open released its lock: the consumer opens at the stored offset.
			c, err = l.Consumer("x", ConsumerOptions{})
			if err != nil || c.Offset() != 4 || !c.Protected() {
				t.Fatalf("%s %v: reopen %v", op, errno, err)
			}
			_ = c.Close()
			_ = l.Close()
		}
	}
}

// A commit whose file write fails leaves the handle's offset where it was, so Offset and
// the Read that follows it agree with what the file says.
func TestAFailedCommitLeavesTheHandlesOffsetWhereItWas(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 6)
	mustFlush(t, l)
	c, err := l.Consumer("x", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	before := fileBytes(t, s, "/log/consumers/x")
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpRename {
			return syscall.EIO
		}
		return nil
	}
	if err := c.Commit(4); !errors.Is(err, syscall.EIO) {
		t.Fatalf("commit: %v", err)
	}
	s.Inject = nil
	if c.Offset() != 1 {
		t.Fatalf("offset %d after a failed commit, the file says 1", c.Offset())
	}
	if recs, err := c.Read(1); err != nil || len(recs) != 1 || recs[0].Seq != 1 {
		t.Fatalf("read after a failed commit: %v %v", seqsOf(recs), err)
	}
	if got := fileBytes(t, s, "/log/consumers/x"); !bytes.Equal(got, before) {
		t.Fatalf("file %q, was %q", got, before)
	}
	if err := c.Commit(4); err != nil || c.Offset() != 4 {
		t.Fatalf("retry: %v, offset %d", err, c.Offset())
	}
}

// A temp file a killed commit left in consumers/ is not a consumer: Consumers does not list
// it, so it cannot be a damaged one, which would make every age sweep keep every segment.
func TestALeftoverTempFileInConsumersIsNotAConsumerAndDoesNotStallRetention(t *testing.T) {
	s, clk, l := retentionLog(t, nil)
	defer l.Close()
	fillSegments(t, l, clk, 4) // segments 0..3 sealed, 4 active and empty
	writeFile(t, s, "/log/consumers/"+tempPrefix+"deadbeef", []byte("half a commit"))
	list, err := l.Consumers()
	if err != nil || len(list) != 0 {
		t.Fatalf("consumers %+v %v", list, err)
	}
	clk.Advance(2 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("segments %v, want only the active one", ids)
	}
}

// The offset file of a consumer one past the caught-up position (Head + 2) is ahead of the
// log and refused; Head + 1 is a caught-up consumer and accepted.
func TestAConsumerTwoAheadOfTheHeadIsRefusedAndOneAheadIsCaughtUp(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	writeFile(t, s, "/log/consumers/two", encodeConsumer(consumerState{ID: logID(l.ID()), Next: 5}))
	if _, err := l.Consumer("two", ConsumerOptions{}); !errors.Is(err, ErrAhead) {
		t.Fatalf("head+2: %v", err)
	}
	writeFile(t, s, "/log/consumers/one", encodeConsumer(consumerState{ID: logID(l.ID()), Next: 4}))
	c, err := l.Consumer("one", ConsumerOptions{})
	if err != nil {
		t.Fatalf("head+1: %v", err)
	}
	_ = c.Close()
}
