package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// A prune that did less than asked says so on stderr (A5-36): the clock guard and a consumer file
// that cannot be read are warnings with exit 0 (-force is the answer to the first), where an
// unlink failure exits 1 (prune_unlink_darwin_test.go).

// expiredLog is a log of nine sealed segments whose records are older than the one second of
// retention the tests pass.
func expiredLog(t *testing.T) string {
	t.Helper()
	cfg := quill.Config{Dir: filepath.Join(t.TempDir(), "orders"), Retention: -1, SegmentBytes: 4096, FlushInterval: time.Hour}
	l, err := quill.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		if _, err := l.Append([]byte("rec")); err != nil {
			t.Fatal(err)
		}
		if err := l.Roll(); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2100 * time.Millisecond)
	return cfg.Dir
}

func TestPruneSaysWhenTheClockGuardWithheldSegments(t *testing.T) {
	dir := expiredLog(t)
	code, out, errw := exec(t, "", "prune", "-dir", dir, "-retention", "1s")
	if code != exitOK || !strings.Contains(out, "segments 10 -> ") {
		t.Fatalf("exit %d: %s%s", code, out, errw)
	}
	if !strings.Contains(errw, "age sweep limited by the clock guard") || !strings.Contains(errw, "-force") {
		t.Fatalf("the clock guard is not reported: %q", errw)
	}
	// -force lifts the guard: the same pass finishes and says nothing.
	code, out, errw = exec(t, "", "prune", "-dir", dir, "-retention", "1s", "-force")
	if code != exitOK || strings.Contains(errw, "clock guard") {
		t.Fatalf("exit %d: %s%s", code, out, errw)
	}
}

func TestPruneSaysWhenAConsumerFileCannotBeRead(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("r") })
	if err := os.WriteFile(filepath.Join(dir, "consumers", "broken"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errw := exec(t, "", "prune", "-dir", dir, "-retention", "1h")
	if code != exitOK {
		t.Fatalf("exit %d: %s%s", code, out, errw)
	}
	if !strings.Contains(errw, "consumer file could not be read") || !strings.Contains(errw, "broken") {
		t.Fatalf("the unreadable consumer file is not reported: %q", errw)
	}
}

// The offline verbs print the messages a prune pass can produce when it does less than asked,
// and drop the retention lines they never printed.
func TestOfflineLogFilterPassesEveryPruneShortfallMessage(t *testing.T) {
	for _, msg := range []string{
		"cannot delete segment",
		"age sweep limited by the clock guard; use prune -force if the clock is right",
		"age sweep stopped at a segment stamped in the future; newer segments are kept until the clock reaches it",
		"consumer file could not be read; the age sweep keeps every segment this pass",
	} {
		var out strings.Builder
		lg := slog.New(lossOnly{slog.NewTextHandler(&out, nil)})
		lg.Log(context.Background(), slog.LevelWarn, msg)
		if !strings.Contains(out.String(), msg) {
			t.Errorf("%q is filtered out", msg)
		}
	}
	var out strings.Builder
	slog.New(lossOnly{slog.NewTextHandler(&out, nil)}).Log(context.Background(), slog.LevelWarn, "deleted segment")
	if out.Len() != 0 {
		t.Errorf("an undamaged deletion is printed: %q", out.String())
	}
}

// A writable open that cannot tighten a quill.lock an earlier build left wider (the file belongs to
// another account) logs a Warn and still succeeds. The offline verbs open the log writable and say
// so on standard error, because the process that owns that lock may be one nobody is watching.
func TestOfflineLogFilterPassesTheLockTighteningWarning(t *testing.T) {
	const msg = "quill.lock is wider than 0600 and could not be tightened: a process that can read it can take the writer's lock"
	var out strings.Builder
	slog.New(lossOnly{slog.NewTextHandler(&out, nil)}).Log(context.Background(), slog.LevelWarn, msg,
		"path", "/log/quill.lock", "error", "operation not permitted")
	if !strings.Contains(out.String(), msg) || !strings.Contains(out.String(), "operation not permitted") {
		t.Errorf("the lock tightening warning is filtered out: %q", out.String())
	}
}
