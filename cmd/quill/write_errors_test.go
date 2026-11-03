package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// hookWriter calls fn once, the first time a write contains the marker, then keeps collecting.
type hookWriter struct {
	buf    bytes.Buffer
	marker string
	fn     func()
	fired  bool
}

func (w *hookWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if !w.fired && bytes.Contains(p, []byte(w.marker)) {
		w.fired = true
		w.fn()
	}
	return n, err
}

// damageSegment flips a byte in the first frame of a segment, so the records it held become a hole.
func damageSegment(t *testing.T, dir, name string) {
	t.Helper()
	p := filepath.Join(dir, name)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	data[10] ^= 0x01
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// threeSealedLog is a log of three sealed segments of three records each and an empty active one.
func threeSealedLog(t *testing.T, cfg quill.Config) string {
	t.Helper()
	cfg.Dir = filepath.Join(t.TempDir(), "orders")
	cfg.Retention, cfg.SegmentBytes, cfg.FlushInterval = -1, 4096, time.Hour
	l, err := quill.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if _, err := l.Append([]byte("rec")); err != nil {
				t.Fatal(err)
			}
		}
		if err := l.Roll(); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return cfg.Dir
}

// An unlink that fails leaves the segment, so the pass did less than asked: quill prune names the
// segment on stderr, still prints the counts line and exits 1. The darwin test makes the first unlink
// fail with the immutable flag; this one needs nothing the platform lacks: the pass logs the damaged
// segment it deleted first, and that line, written to the verb's stderr, is where the test removes
// the directory's write permission, so the next unlink fails with EACCES. The pass deletes the
// oldest segments first, so the second is the one named.
func TestPruneExitsOneWhenALaterUnlinkFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root unlinks in a directory it cannot write")
	}
	dir := threeSealedLog(t, quill.Config{})
	damageSegment(t, dir, "0000000000.log")
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	var out bytes.Buffer
	errw := &hookWriter{marker: "deleted damaged segment", fn: func() {
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Error(err)
		}
	}}
	code := run([]string{"prune", "-dir", dir, "-max-records", "2", "-force"}, strings.NewReader(""), &out, errw)
	if code != exitError {
		t.Fatalf("exit %d, want %d: %s%s", code, exitError, out.String(), errw.buf.String())
	}
	if !errw.fired {
		t.Fatalf("the first segment was not deleted, so the test never made an unlink fail: %q", errw.buf.String())
	}
	if e := errw.buf.String(); !strings.Contains(e, "cannot delete segment") || !strings.Contains(e, "0000000001.log") {
		t.Fatalf("the failed unlink is not named: %q", e)
	}
	// One of the four segments is gone and the rest stay.
	if !strings.Contains(out.String(), "segments 4 -> 3") {
		t.Fatalf("the counts line is missing or wrong: %q", out.String())
	}
}

// -consumer-ttl is parsed before the log is opened, and a bad value is a usage error.
func TestPruneRejectsAMalformedConsumerTTL(t *testing.T) {
	dir := newLog(t, 3, func(i int) []byte { return []byte("r") })
	code, out, errw := exec(t, "", "prune", "-dir", dir, "-retention", "1h", "-consumer-ttl", "soon")
	if code != exitUsage || !strings.Contains(errw, "-consumer-ttl") || out != "" {
		t.Fatalf("exit %d, want %d: %s%s", code, exitUsage, out, errw)
	}
}

// Nothing to append is a usage error, and it appends nothing.
func TestAppendOfNothingIsAUsageError(t *testing.T) {
	dir := newLog(t, 2, func(i int) []byte { return []byte("r") })
	code, out, errw := exec(t, "", "append", "-dir", dir)
	if code != exitUsage || !strings.Contains(errw, "nothing to append") || out != "" {
		t.Fatalf("exit %d, want %d: %s%s", code, exitUsage, out, errw)
	}
	var s struct{ Head uint64 }
	if err := json.Unmarshal([]byte(mustRun(t, "", "stat", "-dir", dir, "-json")), &s); err != nil || s.Head != 2 {
		t.Fatalf("head %d after a refused append: %v", s.Head, err)
	}
}

// A line over the payload limit the policy records is refused by the library: exit 1, the error on
// stderr, nothing on stdout, and not even the lines before it are appended (one batch).
func TestAppendOverThePolicyPayloadLimitExitsOneAndAppendsNothing(t *testing.T) {
	dir := buildLog(t, quill.Config{Retention: -1, SegmentBytes: 4096, MaxPayloadBytes: 16, FlushInterval: time.Hour}, 1,
		func(i int) []byte { return []byte("seed") })
	code, out, errw := exec(t, "ok\n"+strings.Repeat("x", 17)+"\n", "append", "-dir", dir)
	if code != exitError || out != "" || !strings.Contains(errw, "payload") {
		t.Fatalf("exit %d, want %d: %s%s", code, exitError, out, errw)
	}
	var s struct{ Head uint64 }
	if err := json.Unmarshal([]byte(mustRun(t, "", "stat", "-dir", dir, "-json")), &s); err != nil || s.Head != 1 {
		t.Fatalf("head %d after a refused batch, want 1: %v", s.Head, err)
	}
	if out := mustRun(t, "", "append", "-dir", dir, strings.Repeat("y", 16)); strings.TrimSpace(out) != "2" {
		t.Fatalf("a payload at the limit: %q", out)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

// A stdin that fails while it is read exits 1 naming stdin, and appends nothing.
func TestAppendStdinReadErrorExitsOne(t *testing.T) {
	dir := newLog(t, 1, func(i int) []byte { return []byte("r") })
	var out, errw bytes.Buffer
	code := run([]string{"append", "-dir", dir}, io.MultiReader(strings.NewReader("a\nb\n"), failingReader{}), &out, &errw)
	if code != exitError || !strings.Contains(errw.String(), "stdin: boom") || out.String() != "" {
		t.Fatalf("exit %d, want %d: %s%s", code, exitError, out.String(), errw.String())
	}
	var s struct{ Head uint64 }
	if err := json.Unmarshal([]byte(mustRun(t, "", "stat", "-dir", dir, "-json")), &s); err != nil || s.Head != 1 {
		t.Fatalf("head %d after a failed read, want 1: %v", s.Head, err)
	}
}

// recover prints one line for the recovered state and one for each hole.
func TestRecoverPrintsEachHole(t *testing.T) {
	dir := threeSealedLog(t, quill.Config{})
	damageSegment(t, dir, "0000000000.log")
	var s struct{ Holes []struct{ From, To uint64 } }
	if err := json.Unmarshal([]byte(mustRun(t, "", "stat", "-dir", dir, "-json")), &s); err != nil || len(s.Holes) != 1 {
		t.Fatalf("setup: want one hole: %+v %v", s, err)
	}
	out := mustRun(t, "", "recover", "-dir", dir)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	want := fmt.Sprintf("hole %d-%d", s.Holes[0].From, s.Holes[0].To)
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "recovered: ") || lines[1] != want {
		t.Fatalf("recover printed %q, want a recovered line and %q", out, want)
	}
}

// commit and register name the consumer with -name; without it they are usage errors, and neither
// creates anything.
func TestCommitAndRegisterRequireAName(t *testing.T) {
	dir := newLog(t, 3, func(i int) []byte { return []byte("r") })
	for _, args := range [][]string{
		{"register", "-dir", dir},
		{"register", "-dir", dir, "-protect"},
		{"commit", "-dir", dir, "-offset", "2"},
	} {
		code, out, errw := exec(t, "", args...)
		if code != exitUsage || !strings.Contains(errw, "-name is required") || out != "" {
			t.Errorf("quill %v: exit %d, want %d: %s%s", args, code, exitUsage, out, errw)
		}
	}
	if ents, _ := os.ReadDir(filepath.Join(dir, "consumers")); len(ents) != 0 {
		t.Errorf("a refused verb left consumer files: %v", ents)
	}
}

// A protected consumer whose offset lies in a hole is probed at the first record after the hole
// (the stale probe matches the writer's pass, decision 45), so it is still found stale.
func TestConsumersStaleProbeOfAConsumerInAHole(t *testing.T) {
	dir := threeSealedLog(t, quill.Config{ConsumerTTL: time.Nanosecond})
	mustRun(t, "", "register", "-dir", dir, "-name", "slow", "-protect")
	mustRun(t, "", "commit", "-dir", dir, "-name", "slow", "-offset", "2")
	damageSegment(t, dir, "0000000000.log")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "consumers", "slow"), old, old); err != nil {
		t.Fatal(err)
	}
	if rows := consumerRows(t, dir); rows["slow"]["stale"] != true {
		t.Fatalf("a consumer in a hole is not flagged stale: %v", rows["slow"])
	}
	for _, line := range strings.Split(strings.TrimSpace(mustRun(t, "", "consumers", "-dir", dir)), "\n")[1:] {
		if strings.HasPrefix(line, "slow") && !strings.Contains(line, "stale") {
			t.Errorf("table row %q has no stale note", line)
		}
	}
}
