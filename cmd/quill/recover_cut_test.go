package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// staleMarkerLog returns a log whose quill.tail is stale (it names record 3 of 5) and whose
// last frame is damaged, the one case where an acknowledged record is cut at the next
// writable open.
func staleMarkerLog(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "orders")
	l, err := quill.Open(quill.Config{Dir: dir, Retention: -1, SegmentBytes: 4096, FlushInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := l.Append([]byte("rec")); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Flush(); err != nil {
		t.Fatal(err)
	}
	tailPath := filepath.Join(dir, "quill.tail")
	stale, err := os.ReadFile(tailPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := l.Append([]byte("rec")); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tailPath, stale, 0o600); err != nil {
		t.Fatal(err)
	}
	seg := filepath.Join(dir, "0000000000.log")
	data, err := os.ReadFile(seg)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 0xff
	if err := os.WriteFile(seg, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Every verb that opens the log writable can cut the newest segment, so each says so on stderr,
// and says nothing when it cuts nothing.
func TestEveryWritableVerbReportsACutOnStderr(t *testing.T) {
	for _, args := range [][]string{
		{"recover"},
		{"append", "zz"},
		{"roll"},
		{"prune", "-max-records", "1000", "-max-bytes", "100000000"},
	} {
		t.Run(args[0], func(t *testing.T) {
			dir := staleMarkerLog(t)
			argv := append([]string{args[0], "-dir", dir}, args[1:]...)
			code, _, errw := exec(t, "", argv...)
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, errw)
			}
			if !strings.Contains(errw, "cut the newest segment") || !strings.Contains(errw, "kind=damaged") {
				t.Fatalf("stderr does not report the cut: %q", errw)
			}
			// The log is now whole: the next writable verb has nothing to report.
			code, _, errw = exec(t, "", "recover", "-dir", dir)
			if code != exitOK || errw != "" {
				t.Fatalf("second recover: exit %d, stderr %q", code, errw)
			}
		})
	}
}
