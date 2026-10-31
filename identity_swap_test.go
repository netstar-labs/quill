package quill

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A log directory replaced by another log while a Reader and a consumer are open must not let
// the consumer stamp its old log id into the new log's consumers/, nor leave a Reader waiting
// on a log it never opened.
func TestAHandleOpenAcrossAReplacedLogRefusesCommitAndWaitWithErrWrongLog(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 5)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()
	c, err := r.Consumer("c", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Commit(3); err != nil {
		t.Fatalf("commit before the swap: %v", err)
	}
	before, err := s.Durable("/log/consumers/c")
	if err != nil {
		t.Fatal(err)
	}
	if string(r.id) == string(testID) {
		t.Fatal("the replacement id must differ")
	}
	// The directory now holds another log: same marker and segments, another identity.
	writeFile(t, s, "/log/"+formatName, encodeFormat(testID))

	if err := c.Commit(4); !errors.Is(err, ErrWrongLog) {
		t.Fatalf("Commit after the swap: %v, want ErrWrongLog", err)
	}
	if got, err := s.Durable("/log/consumers/c"); err != nil || string(got) != string(before) {
		t.Fatalf("consumer file changed after a refused commit: %q %v", got, err)
	}
	if c.Offset() != 3 {
		t.Fatalf("offset moved to %d by a refused commit", c.Offset())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := r.ReadWait(ctx, 6, 1); !errors.Is(err, ErrWrongLog) {
		t.Fatalf("ReadWait after the swap: %v, want ErrWrongLog", err)
	}
}
