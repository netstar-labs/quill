package quill

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// rewindArchiver sets the source's _archive offset back to 1, which is how an operator makes
// the next run look at every segment again (quill commit -name _archive -offset 1).
func rewindArchiver(t *testing.T, src string) {
	t.Helper()
	path := filepath.Join(src, consumersDir, archiveConsumer)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Fields(string(b))
	if err := os.WriteFile(path, []byte(f[0]+" 1 "+f[2]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A rerun used to keep a destination segment of the same size as the source's without reading
// it, so a damaged copy survived the rewind that was meant to replace it, the run exited 0 and
// only check on the destination found it (A5-32). A same-size file is now read and verified: a
// damaged one is replaced by a fresh copy, a healthy one is left alone (not rewritten).
func TestArchiveRerunReplacesASameSizeDamagedDestinationSegmentAndKeepsAHealthyOne(t *testing.T) {
	src := archiveSource(t)
	dst := filepath.Join(t.TempDir(), "arch")
	if err := Archive(src, dst); err != nil {
		t.Fatal(err)
	}
	ids, err := listSegments(fsx.OS{}, dst)
	if err != nil || len(ids) < 2 {
		t.Fatalf("segments %v, %v", ids, err)
	}
	damaged := filepath.Join(dst, segmentName(ids[1]))
	healthy := filepath.Join(dst, segmentName(ids[0]))
	want, err := os.ReadFile(filepath.Join(src, segmentName(ids[1])))
	if err != nil {
		t.Fatal(err)
	}
	bad := slices.Clone(want)
	bad[len(bad)/2] ^= 0xff
	if err := os.WriteFile(damaged, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	beforeHealthy, err := os.Stat(healthy)
	if err != nil {
		t.Fatal(err)
	}
	rewindArchiver(t, src)
	if err := Archive(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(damaged)
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("the damaged same-size destination segment was kept after a rerun (read err %v)", err)
	}
	afterHealthy, err := os.Stat(healthy)
	if err != nil || !os.SameFile(beforeHealthy, afterHealthy) {
		t.Fatalf("a healthy same-size destination segment was rewritten (%v)", err)
	}
	if es := listing(t, dst); slices.ContainsFunc(es, func(n string) bool { return strings.HasPrefix(n, tempPrefix) }) {
		t.Fatalf("temp files left in the destination: %v", es)
	}
}

// A segment whose last frame reaches one sequence into the next segment's range is the overlap
// Open refuses (decision 102): copying it would publish a marker below the last record copied.
// One frame is the smallest overlap, so it is the edge of the check.
func TestArchiveRefusesASegmentOverlappingTheNextByOneFrame(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3) // segments 0..2 hold 1..4, 5..8, 9..12
	writeFile(t, s, "/log/"+segmentName(0), image(1, 5, 0, smallPayload))
	err := archive(s, "/log", "/arch")
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("%v, want ErrCorrupt", err)
	}
	if slices.Contains(segmentFiles(t, s, "/arch"), 0) {
		t.Fatal("copied the overlapping segment")
	}
}

// A destination segment is kept only when its size equals the source's: one with trailing
// bytes after a verifying prefix is not the source's copy and is replaced (decision 85).
func TestArchiveRerunReplacesADestinationSegmentWithTrailingBytes(t *testing.T) {
	src := archiveSource(t)
	dst := filepath.Join(t.TempDir(), "arch")
	if err := Archive(src, dst); err != nil {
		t.Fatal(err)
	}
	ids, err := listSegments(fsx.OS{}, dst)
	if err != nil || len(ids) < 2 {
		t.Fatalf("segments %v, %v", ids, err)
	}
	p := filepath.Join(dst, segmentName(ids[1]))
	want, err := os.ReadFile(filepath.Join(src, segmentName(ids[1])))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(slices.Clone(want), 1, 2, 3), 0o600); err != nil {
		t.Fatal(err)
	}
	rewindArchiver(t, src)
	if err := Archive(src, dst); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(p); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("destination segment holds %d bytes, source %d (%v)", len(got), len(want), err)
	}
}

// An intact destination segment of the source's size that holds other sequences (here the
// same frames shifted by one) is not the source's copy: it is replaced, not kept (decision 85).
func TestArchiveRerunReplacesASameSizeIntactDestinationSegmentOfAnotherSequenceRange(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 3)
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	src := fileBytes(t, s, "/log/"+segmentName(1)) // records 5..8
	var shifted []byte
	for pos := 0; pos < len(src); {
		rec, n, err := decodeFrame(src[pos:])
		if err != nil {
			t.Fatal(err)
		}
		rec.Seq++
		rec.Payload = bytes.Clone(rec.Payload)
		shifted = appendFrame(shifted, &rec)
		pos += n
	}
	if len(shifted) != len(src) || bytes.Equal(shifted, src) {
		t.Fatalf("setup: %d bytes against %d", len(shifted), len(src))
	}
	writeFile(t, s, "/arch/"+segmentName(1), shifted)
	r := openReaderSim(t, s)
	c, err := r.Consumer(archiveConsumer, ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Commit(1); err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	r.Close()
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	if got := fileBytes(t, s, "/arch/"+segmentName(1)); !bytes.Equal(got, src) {
		t.Fatal("the destination segment of another sequence range was kept")
	}
}

// With no writer holding the lock the lock check cannot refuse, so the same-file check alone
// stops Archive(dir, dir) from rewriting the source's quill.tail, which has one writer.
func TestArchiveRefusesTheClosedSourceLogAsItsOwnDestination(t *testing.T) {
	src := archiveSource(t)
	before, err := os.ReadFile(filepath.Join(src, tailName))
	if err != nil {
		t.Fatal(err)
	}
	if err := Archive(src, src); err == nil {
		t.Fatal("Archive(src, src) returned nil")
	}
	after, err := os.ReadFile(filepath.Join(src, tailName))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("marker rewritten: %q -> %q (%v)", before, after, err)
	}
}
