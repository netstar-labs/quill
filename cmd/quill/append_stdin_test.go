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

// The refusal does not claim a size stdin does not have (audit L1-26): the limit is applied to
// the sum of len(line)+34 per line, so 479,350 one-byte lines (958,700 bytes of stdin, far under
// MaxBatchBytes) are refused, and the message says how lines are counted. 479,349 lines are
// accepted.
func TestAppendStdinRefusalMessageStatesTheAccounting(t *testing.T) {
	const fits = quill.DefaultMaxBatchBytes / 35 // 479,349 lines of "x\n"
	if fits != 479349 {
		t.Fatalf("setup: %d one-byte lines fit, want 479349", fits)
	}
	lines := func(n int) string { return strings.Repeat("x\n", n) }

	if out := mustRun(t, lines(fits), "append", "-dir", t.TempDir()); strings.TrimSpace(out) != fmt.Sprint(fits) {
		t.Fatalf("append of %d one-byte lines: %q, want %d", fits, out, fits)
	}

	in := lines(fits + 1)
	if len(in) != 958700 {
		t.Fatalf("setup: stdin is %d bytes, want 958700", len(in))
	}
	code, out, errw := exec(t, in, "append", "-dir", t.TempDir())
	if code != exitError || out != "" {
		t.Fatalf("append of %d one-byte lines: exit %d, stdout %q, stderr %q; want exit %d and no output", fits+1, code, out, errw, exitError)
	}
	for _, want := range []string{
		"payload too large",
		fmt.Sprintf("MaxBatchBytes (%d)", quill.DefaultMaxBatchBytes),
		"each line is counted as its length plus 34 bytes",
	} {
		if !strings.Contains(errw, want) {
			t.Errorf("stderr %q lacks %q", errw, want)
		}
	}
	// Stdin is 958,700 bytes, so a message saying it is a batch of more than MaxBatchBytes is false.
	if strings.Contains(errw, "more than") {
		t.Errorf("stderr %q states a size that stdin (%d bytes) does not have", errw, len(in))
	}
}
