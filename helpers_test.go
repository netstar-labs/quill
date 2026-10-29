package quill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Test helpers shared by the phase-2 tests.

// simConfig is a config on the simulator with a slow background flusher, so only the test
// drives file operations, and small segments so rotation happens.
func simConfig(s *fsx.Sim, dir string) Config {
	return Config{Dir: dir, FlushInterval: time.Hour, FlushBytes: 1 << 20, SegmentBytes: 512, Retention: -1, fs: s}
}

func openSim(t *testing.T, s *fsx.Sim, mut func(*Config)) *Log {
	t.Helper()
	cfg := simConfig(s, "/log")
	if mut != nil {
		mut(&cfg)
	}
	l, err := Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return l
}

func openDir(t *testing.T, dir string, mut func(*Config)) *Log {
	t.Helper()
	cfg := Config{Dir: dir, FlushInterval: time.Hour, SegmentBytes: 4096, Retention: -1}
	if mut != nil {
		mut(&cfg)
	}
	l, err := Open(cfg)
	if err != nil {
		t.Fatalf("open %s: %v", dir, err)
	}
	return l
}

// payloadFor is the deterministic payload of a sequence, so a recovered record can be checked.
func payloadFor(seq uint64) []byte {
	return []byte(fmt.Sprintf("record-%d-%s", seq, strings.Repeat("x", int(seq%37))))
}

func appendN(t *testing.T, l *Log, n int) (first, last uint64) {
	t.Helper()
	for i := 0; i < n; i++ {
		seq, err := l.Append(payloadFor(l.nextSeqForTest()))
		if err != nil {
			t.Fatalf("append: %v", err)
		}
		if i == 0 {
			first = seq
		}
		last = seq
	}
	return first, last
}

func (l *Log) nextSeqForTest() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.nextSeq
}

func mustFlush(t *testing.T, l *Log) {
	t.Helper()
	if err := l.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
}

// readAll returns every retained record, copied, and the error that ended the scan.
func readAll(l *Log) ([]Record, error) {
	var out []Record
	err := l.ScanAll(func(r Record) bool {
		r.Payload = append([]byte(nil), r.Payload...)
		out = append(out, r)
		return true
	})
	return out, err
}

func seqsOf(recs []Record) []uint64 {
	out := make([]uint64, len(recs))
	for i, r := range recs {
		out[i] = r.Seq
	}
	return out
}

// checkContiguous asserts the records run 1..n (or first..last) with the payloads payloadFor
// gives, and returns the last sequence.
func checkContiguous(t *testing.T, recs []Record, first uint64) uint64 {
	t.Helper()
	for i, r := range recs {
		want := first + uint64(i)
		if r.Seq != want {
			t.Fatalf("record %d has seq %d, want %d (seqs %v)", i, r.Seq, want, seqsOf(recs))
		}
		if string(r.Payload) != string(payloadFor(want)) {
			t.Fatalf("seq %d payload %q, want %q", r.Seq, r.Payload, payloadFor(want))
		}
	}
	if len(recs) == 0 {
		return first - 1
	}
	return recs[len(recs)-1].Seq
}

func fileBytes(t *testing.T, fsys fsx.FS, path string) []byte {
	t.Helper()
	f, err := fsys.OpenFile(path, fsx.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, info.Size())
	if n, err := f.ReadAt(buf, 0); err != nil && n < len(buf) {
		t.Fatalf("read %s: %v", path, err)
	}
	return buf
}

func writeFile(t *testing.T, fsys fsx.FS, path string, data []byte) {
	t.Helper()
	f, err := fsys.OpenFile(path, fsx.O_RDWR|fsx.O_CREATE, 0o600)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	if err := f.Truncate(0); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt(data, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func readTail(t *testing.T, fsys fsx.FS, dir string) tail {
	t.Helper()
	tl, err := decodeTail(fileBytes(t, fsys, filepath.Join(dir, tailName)))
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	return tl
}

func segmentFiles(t *testing.T, fsys fsx.FS, dir string) []uint64 {
	t.Helper()
	names, err := fsys.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var ids []uint64
	for _, n := range names {
		if id, ok := parseSegmentName(n); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// The SIGKILL helper. TestMain runs a helper role when QUILL_HELPER is set, so a test can
// start this binary as a writer in another process and kill it.
func TestMain(m *testing.M) {
	switch os.Getenv("QUILL_HELPER") {
	case "":
		os.Exit(m.Run())
	case "writer":
		helperWriter()
	case "holder":
		helperHolder()
	}
	os.Exit(0)
}

// helperWriter appends records with payloadFor, acknowledging each sequence on stdout once it
// is durable ("ack N"): with QUILL_HELPER_SYNC set, by Sync per record; otherwise by a Flush
// every few records. It runs until killed.
func helperWriter() {
	dir := os.Getenv("QUILL_HELPER_DIR")
	segBytes, _ := strconv.Atoi(os.Getenv("QUILL_HELPER_SEGBYTES"))
	l, err := Open(Config{Dir: dir, SegmentBytes: int64(segBytes), Retention: -1, FlushInterval: 10 * time.Millisecond})
	if err != nil {
		fmt.Fprintln(os.Stderr, "helper open:", err)
		os.Exit(1)
	}
	sync := os.Getenv("QUILL_HELPER_SYNC") != ""
	for i := 0; ; i++ {
		seq, err := l.Append(payloadFor(l.nextSeqForTest()))
		if err != nil {
			fmt.Fprintln(os.Stderr, "helper append:", err)
			os.Exit(1)
		}
		switch {
		case sync:
			if err := l.Sync(context_background(), seq); err != nil {
				fmt.Fprintln(os.Stderr, "helper sync:", err)
				os.Exit(1)
			}
			fmt.Printf("ack %d\n", seq)
		case i%5 == 4:
			if err := l.Flush(); err != nil {
				fmt.Fprintln(os.Stderr, "helper flush:", err)
				os.Exit(1)
			}
			fmt.Printf("ack %d\n", seq)
		}
	}
}

// helperHolder opens the log, prints "locked" and waits to be killed.
func helperHolder() {
	l, err := Open(Config{Dir: os.Getenv("QUILL_HELPER_DIR"), Retention: -1})
	if err != nil {
		fmt.Fprintln(os.Stderr, "helper open:", err)
		os.Exit(1)
	}
	fmt.Println("locked")
	_ = l
	select {}
}

// isGap reports whether err is a *GapError with the given bounds.
func isGap(err error, from, to uint64) bool {
	var g *GapError
	return errors.As(err, &g) && g.From == from && g.To == to
}
