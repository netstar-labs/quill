package quill

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// archiveSource builds a log under a fresh directory with three sealed segments and returns
// its path. The writer is closed, so the log is a plain source for Archive.
func archiveSource(t *testing.T) string {
	t.Helper()
	src := filepath.Join(t.TempDir(), "log")
	l := openDir(t, src, func(c *Config) { c.SegmentBytes = 256 })
	for i := 0; i < 3; i++ {
		appendN(t, l, 8)
		mustFlush(t, l)
	}
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return src
}

// listing returns the sorted names directly inside dir.
func listing(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name())
	}
	slices.Sort(names)
	return names
}

// A destination that exists, holds files and has no quill.format is not a log, and a writable
// Open refuses it for that reason; Archive used to write a marker into it and rename a segment
// copy over an unrelated file named like a segment (A3-14). It now refuses it with ErrNoLog,
// before it writes anything, and the unrelated files are as they were.
func TestArchiveRefusesAPopulatedDestinationThatIsNotALogAndLeavesItUntouched(t *testing.T) {
	src := archiveSource(t)
	for _, tc := range []struct {
		name  string
		files map[string]string
	}{
		{"a file named like a segment", map[string]string{segmentName(0): "precious unrelated data", "notes.txt": "keep"}},
		{"an unrelated file only", map[string]string{"notes.txt": "keep"}},
		{"a tail with no format", map[string]string{tailName: "{}"}},
		{"a segment-named file beside a temp file and a lock", map[string]string{segmentName(0): "x", tempPrefix + "a": "t", lockName: ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := filepath.Join(t.TempDir(), "unrelated")
			if err := os.Mkdir(dst, 0o700); err != nil {
				t.Fatal(err)
			}
			for n, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dst, n), []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			before := listing(t, dst)
			err := Archive(src, dst)
			if !errors.Is(err, ErrNoLog) {
				t.Fatalf("got %v, want ErrNoLog", err)
			}
			if !strings.Contains(err.Error(), dst) || !strings.Contains(err.Error(), formatName) {
				t.Fatalf("the error does not name the directory and %s: %v", formatName, err)
			}
			if after := listing(t, dst); !slices.Equal(before, after) {
				t.Fatalf("the destination's entries changed from %v to %v", before, after)
			}
			for n, content := range tc.files {
				if got, err := os.ReadFile(filepath.Join(dst, n)); err != nil || string(got) != content {
					t.Fatalf("%s is %q (%v), want %q", n, got, err, content)
				}
			}
		})
	}
}

// The destinations Archive has always accepted still work: one that does not exist, an empty
// one, one holding only .tmp-* files and quill.lock (what Log.create ignores too), and one that
// is already an archive.
func TestArchiveStillAcceptsANewEmptyTempOnlyAndExistingDestination(t *testing.T) {
	for _, tc := range []struct {
		name string
		prep func(t *testing.T, src, dst string)
	}{
		{"does not exist", func(t *testing.T, src, dst string) {}},
		{"empty", func(t *testing.T, src, dst string) {
			if err := os.Mkdir(dst, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"only temp files and a lock", func(t *testing.T, src, dst string) {
			if err := os.Mkdir(dst, 0o700); err != nil {
				t.Fatal(err)
			}
			for _, n := range []string{tempPrefix + "left", lockName} {
				if err := os.WriteFile(filepath.Join(dst, n), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"already an archive", func(t *testing.T, src, dst string) {
			if err := Archive(src, dst); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A fresh source per case: a run commits the source's _archive consumer, so a reused
			// source has nothing left to copy.
			src := archiveSource(t)
			dst := filepath.Join(t.TempDir(), "arch")
			tc.prep(t, src, dst)
			if err := Archive(src, dst); err != nil {
				t.Fatal(err)
			}
			r, err := OpenReader(dst, ReaderOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			recs, err := readerAll(r)
			if err != nil || len(recs) == 0 {
				t.Fatalf("%d records, %v", len(recs), err)
			}
			checkContiguous(t, recs, 1)
		})
	}
}

// Archive writes the destination's quill.format before anything else, so a run killed at any
// later write leaves a directory that already has one (or, killed at the marker's own write,
// one holding only a temp file), and archiving again must accept it. Kill the run at each
// successive write to a temp file in the destination and re-run.
func TestArchiveRerunsAfterARunInterruptedAtAnyWrite(t *testing.T) {
	for n := 1; ; n++ {
		s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
		fillSegments(t, l, clk, 3)
		var seen atomic.Int64
		var killed atomic.Bool
		s.Inject = func(op fsx.Op, name string) error {
			if op == fsx.OpWrite && filepath.Dir(name) == "/arch" && strings.HasPrefix(filepath.Base(name), tempPrefix) {
				if int(seen.Add(1)) == n {
					killed.Store(true)
				}
			}
			if killed.Load() {
				return syscall.EIO
			}
			return nil
		}
		err := archive(s, "/log", "/arch")
		s.Inject = nil
		if !killed.Load() {
			if err != nil {
				t.Fatalf("no kill at write %d, yet %v", n, err)
			}
			_ = l.Close()
			if n < 3 {
				t.Fatalf("only %d temp writes in a whole archive run; the kill never landed", n)
			}
			return
		}
		if err == nil {
			t.Fatalf("the run killed at write %d succeeded", n)
		}
		if err := archive(s, "/log", "/arch"); err != nil {
			t.Fatalf("the re-run after a kill at temp write %d: %v", n, err)
		}
		if ids := segmentFiles(t, s, "/arch"); len(ids) != 3 {
			t.Fatalf("after the re-run following a kill at write %d the destination holds segments %v, want 3", n, ids)
		}
		_ = l.Close()
	}
}
