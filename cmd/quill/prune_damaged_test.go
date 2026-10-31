package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netstar-labs/quill"
)

// An offline prune that deletes a damaged sealed segment erases the file and the hole, the only
// artifacts of the damage, so it says so on stderr; undamaged deletions stay silent as before.
func TestOfflinePruneReportsTheDeletionOfADamagedSegment(t *testing.T) {
	dir := buildLog(t, quill.Config{Retention: -1, SegmentBytes: 4096}, 0, nil)
	l, err := quill.Open(quill.Config{Dir: dir, Retention: -1, SegmentBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	// Three sealed segments, then an active one.
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
	seg := filepath.Join(dir, "0000000000.log")
	data, err := os.ReadFile(seg)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 0xff
	if err := os.WriteFile(seg, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := exec(t, "", "check", "-dir", dir); code == exitOK || strings.Contains(out, "HEALTHY") && !strings.Contains(out, "UNHEALTHY") {
		t.Fatalf("setup: check is not unhealthy (exit %d): %s", code, out)
	}
	code, _, errw := exec(t, "", "prune", "-dir", dir, "-max-records", "3", "-force")
	if code != exitOK {
		t.Fatalf("prune: exit %d: %s", code, errw)
	}
	if !strings.Contains(errw, "deleted damaged segment") || !strings.Contains(errw, "0000000000.log") || !strings.Contains(errw, "damage=") {
		t.Fatalf("prune did not report the damaged segment it deleted: %q", errw)
	}
	if strings.Contains(errw, "deleted segment") {
		t.Fatalf("prune reported an undamaged deletion: %q", errw)
	}
}
