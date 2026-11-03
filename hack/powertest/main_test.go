package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// TestMain lets the tests run verify as a child process: it calls log.Fatalf on a failure.
func TestMain(m *testing.M) {
	if os.Getenv("POWERTEST_AS_MAIN") == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

// runVerify runs `powertest verify` on dir and returns its combined output and exit error.
func runVerify(t *testing.T, dir, acks string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-dir", dir, "-acks", acks, "verify")
	cmd.Env = append(os.Environ(), "POWERTEST_AS_MAIN=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// writeLog writes an uncrashed log of several segments, every record acknowledged.
func writeLog(t *testing.T) (dir, acks string) {
	t.Helper()
	dir = filepath.Join(t.TempDir(), "log")
	l, err := quill.Open(quill.Config{Dir: dir, Retention: -1, SegmentBytes: 4096, FlushInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	const n = 2000
	for seq := uint64(1); seq <= n; seq++ {
		if _, err := l.Append(payload(seq)); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	acks = filepath.Join(t.TempDir(), "acks")
	if err := os.WriteFile(acks, []byte(fmt.Sprintf("start head=0\nack %d\n", n)), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, acks
}

func TestVerifyAcceptsIntactLog(t *testing.T) {
	dir, acks := writeLog(t)
	if out, err := runVerify(t, dir, acks); err != nil || !strings.HasPrefix(out, "OK:") {
		t.Fatalf("verify of an intact log: %v\n%s", err, out)
	}
}

// A log whose oldest segment file vanished leaves no hole and no gap between scanned records;
// with Retention -1 nothing is deleted legitimately, so verify must still refuse it.
func TestVerifyRefusesLogMissingItsFirstSegment(t *testing.T) {
	dir, acks := writeLog(t)
	segs, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil || len(segs) < 3 {
		t.Fatalf("want several segments, got %v (%v)", segs, err)
	}
	if err := os.Remove(segs[0]); err != nil { // Glob sorts: the oldest
		t.Fatal(err)
	}
	out, err := runVerify(t, dir, acks)
	if err == nil || !strings.Contains(out, "FAIL: log starts at") {
		t.Fatalf("verify of a log without its first segment: err %v\n%s", err, out)
	}
}
