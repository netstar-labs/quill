package quill

import (
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// The moment a log becomes failed it writes one Error record through Config.Logger, with the
// cause and the operation, and never another (Stage 2 audit A5-1f). A failed log accepts no more
// writes, so this line is how a log that nobody polls through Err() says so.

const failedLogMsg = "log failed: it accepts no more writes and every later call returns ErrFailed"

// wantOneFailedLine checks that lc holds exactly one failed-log record, at Error level, naming
// op and carrying cause, and that no other Error record was written.
func wantOneFailedLine(t *testing.T, lc *logCapture, op, cause string) {
	t.Helper()
	if n := lc.countMsgs(failedLogMsg); n != 1 {
		t.Fatalf("%d failed-log records, want 1: %q", n, lc.msgs)
	}
	if n := lc.countMsgs("level=ERROR"); n != 1 {
		t.Fatalf("%d Error records, want exactly the failed-log one: %q", n, lc.msgs)
	}
	opAttr := "op=" + op
	if n := lc.countMsgs("level=ERROR", failedLogMsg, opAttr, "error=", cause); n != 1 {
		t.Fatalf("the Error record does not carry %s and cause %q: %q", opAttr, cause, lc.msgs)
	}
}

func TestADataFsyncErrorWritesOneErrorLineWithTheCause(t *testing.T) {
	var lc logCapture
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.Logger = lc.logger() })
	defer l.Close()
	appendN(t, l, 2)
	s.Inject = func(o fsx.Op, name string) error {
		if o == fsx.OpSync && name == "/log/"+segmentName(0) {
			return errors.New("injected_fsync_EIO")
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Flush: %v, want ErrFailed", err)
	}
	wantOneFailedLine(t, &lc, "flush", "injected_fsync_EIO")
	for i := 0; i < 3; i++ { // every later call returns the error and writes nothing
		_ = l.Flush()
		_, _ = l.Append([]byte("x"))
	}
	wantOneFailedLine(t, &lc, "flush", "injected_fsync_EIO")
}

func TestATruncateThatCannotUndoAFailedWriteWritesOneErrorLineWithTheCause(t *testing.T) {
	var lc logCapture
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.Logger = lc.logger() })
	defer l.Close()
	appendN(t, l, 2)
	s.Inject = func(o fsx.Op, name string) error {
		switch o {
		case fsx.OpWrite:
			return syscall.EIO
		case fsx.OpTruncate:
			return errors.New("injected_truncate_EPERM")
		}
		return nil
	}
	if err := l.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Flush: %v, want ErrFailed", err)
	}
	wantOneFailedLine(t, &lc, "flush", "injected_truncate_EPERM")
}

// failed is sticky and keeps its first cause: a second failure neither replaces the error that
// Err returns nor writes a second line.
func TestASecondFailureKeepsTheFirstCauseAndWritesNoSecondLine(t *testing.T) {
	var lc logCapture
	l := openSim(t, fsx.NewSim(), func(c *Config) { c.Logger = lc.logger() })
	defer l.Close()
	l.mu.Lock()
	l.fail("flush", errors.New("first_cause"))
	l.fail("rotation", errors.New("second_cause"))
	l.mu.Unlock()
	if err := l.Err(); !errors.Is(err, ErrFailed) || !strings.Contains(err.Error(), "first_cause") || strings.Contains(err.Error(), "second_cause") {
		t.Fatalf("Err: %v, want ErrFailed wrapping the first cause only", err)
	}
	wantOneFailedLine(t, &lc, "flush", "first_cause")
	if lc.has("second_cause") {
		t.Fatalf("the second failure was logged: %q", lc.msgs)
	}
}

// A lost quill.lock already logged its own Error line; it is now this one, not both.
func TestALostLockFileWritesOneErrorLine(t *testing.T) {
	var lc logCapture
	s := fsx.NewSim()
	a := openSim(t, s, func(c *Config) { c.Logger = lc.logger() })
	defer a.Close()
	appendN(t, a, 1)
	mustFlush(t, a)
	if err := s.Remove("/log/" + lockName); err != nil {
		t.Fatal(err)
	}
	b := openSim(t, s, nil)
	defer b.Close()
	appendN(t, a, 1)
	if err := a.Flush(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Flush: %v, want ErrFailed", err)
	}
	_ = a.Flush()
	wantOneFailedLine(t, &lc, "\"lock check\"", "quill.lock")
}
