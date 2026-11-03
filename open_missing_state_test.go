package quill

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A log that has lost its marker or its segments is refused, never silently re-created empty
// (decisions 62 and 81; guarantees 4 and 6). Each case damages a closed log holding records and
// checks the error text, because other checks refuse some of the same inputs with other words.
func TestOpenRefusesALogWhoseMarkerOrSegmentsAreGone(t *testing.T) {
	cases := []struct {
		name   string
		damage func(t *testing.T, dir string)
		want   string
	}{
		{"marker present and every segment deleted", func(t *testing.T, dir string) {
			segs, err := filepath.Glob(filepath.Join(dir, "*.log"))
			if err != nil || len(segs) == 0 {
				t.Fatalf("setup: segments %v, %v", segs, err)
			}
			for _, p := range segs {
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
			}
		}, "no segment files"},
		{"segments with frames and no marker", func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, tailName)); err != nil {
				t.Fatal(err)
			}
		}, "segments with frames but no " + tailName},
		{"garbage marker", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, tailName), []byte("garbage"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, tailName},
		{"marker naming a segment above the newest on disk", func(t *testing.T, dir string) {
			tl := tail{Seq: 5, Seg: 7, Off: 120, MaxSeg: 7}
			if err := os.WriteFile(filepath.Join(dir, tailName), encodeTail(tl), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "the newest on disk is 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "log")
			l := openDir(t, dir, nil)
			appendN(t, l, 3)
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			seg0 := filepath.Join(dir, segmentName(0))
			before, err := os.ReadFile(seg0)
			if err != nil || len(before) == 0 {
				t.Fatalf("setup: %s holds %d bytes, %v", segmentName(0), len(before), err)
			}
			tc.damage(t, dir)
			_, err = Open(Config{Dir: dir, Retention: -1})
			if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("open: %v, want ErrCorrupt containing %q", err, tc.want)
			}
			// The refusal must not repair anything: no re-created empty segment 0, no cut.
			after, err := os.ReadFile(seg0)
			if tc.name == "marker present and every segment deleted" {
				if err == nil {
					t.Fatalf("the refused open re-created %s", segmentName(0))
				}
				return
			}
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("the refused open changed %s: %d bytes -> %d bytes (%v)", segmentName(0), len(before), len(after), err)
			}
		})
	}
}

// "No segment holds a byte" is the whole test of an interrupted creation (decision 62): a
// segment of one byte already holds data, so the marker's absence is corruption, and an empty
// segment is the creation the open completes.
func TestOpenWithoutAMarkerRefusesAOneByteSegmentAndCompletesAnEmptyOne(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int
		ok   bool
	}{{"an empty segment", 0, true}, {"a one-byte segment", 1, false}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "log")
			l := openDir(t, dir, nil)
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(dir, tailName)); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, segmentName(0)), make([]byte, tc.size), 0o600); err != nil {
				t.Fatal(err)
			}
			l, err := Open(Config{Dir: dir, Retention: -1})
			if tc.ok {
				if err != nil {
					t.Fatalf("open: %v", err)
				}
				l.Close()
				return
			}
			if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "segments with frames but no "+tailName) {
				t.Fatalf("open: %v, want ErrCorrupt containing the missing-marker text", err)
			}
		})
	}
}
