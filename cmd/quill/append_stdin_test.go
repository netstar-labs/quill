package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/netstar-labs/quill"
)

// newlines yields n newlines and counts what was read.
type newlines struct{ left, read int }

func (r *newlines) Read(p []byte) (int, error) {
	if r.left == 0 {
		return 0, io.EOF
	}
	n := min(len(p), r.left)
	for i := range p[:n] {
		p[i] = '\n'
	}
	r.left -= n
	r.read += n
	return n, nil
}

// An oversize stdin batch is refused while it is read, not after all of stdin is in memory
// (guarantee 7): the command stops reading shortly after the batch passes MaxBatchBytes.
func TestAppendStdinIsRefusedBeforeItIsAllBuffered(t *testing.T) {
	dir := t.TempDir()
	const total = 4 << 20 // empty lines: 4 million records, 8 times the 479,000 that fit
	in := &newlines{left: total}
	var out, errw bytes.Buffer
	code := run([]string{"append", "-dir", dir}, in, &out, &errw)
	if code != exitError || !strings.Contains(errw.String(), "MaxBatchBytes") {
		t.Fatalf("code %d, stderr %q: want exit %d naming MaxBatchBytes", code, errw.String(), exitError)
	}
	// Past the limit, only the scanner's read-ahead (one 1 MiB buffer) may have been consumed.
	if in.read > 2<<20 {
		t.Fatalf("read %d of %d bytes of stdin before refusing; the batch was buffered whole", in.read, total)
	}
	if o := out.String(); o != "" {
		t.Fatalf("stdout %q after a refused batch", o)
	}
}

// A single line is a batch of one record, bounded by MaxPayloadBytes alone (decision 46), so a
// line above MaxBatchBytes is still accepted.
func TestAppendStdinSingleLineAboveMaxBatchBytesIsAccepted(t *testing.T) {
	dir := t.TempDir()
	line := strings.Repeat("x", quill.DefaultMaxBatchBytes+1<<20)
	out := mustRun(t, line+"\n", "append", "-dir", dir)
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("append: %q", out)
	}
}

// The largest batch of empty lines the library accepts (34 accounted bytes each) is accepted.
func TestAppendStdinAtMaxBatchBytesIsAccepted(t *testing.T) {
	dir := t.TempDir()
	const n = quill.DefaultMaxBatchBytes / 34
	out := mustRun(t, strings.Repeat("\n", n), "append", "-dir", dir)
	if want := fmt.Sprint(n); strings.TrimSpace(out) != want {
		t.Fatalf("append: %q, want %s", out, want)
	}
}
