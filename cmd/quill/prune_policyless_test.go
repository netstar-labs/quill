package main

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// forgeAge rewrites every frame of every segment with At = at. Frames keep their length (both
// stamps are five-byte varints), so quill.tail stays valid. The CLI cannot set the writer's
// clock, so this is how a test gets a log whose records are old.
func forgeAge(t *testing.T, dir string, at int64) {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(dir, "*.log"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var out []byte
		for len(b) > 0 {
			bl, n := binary.Uvarint(b)
			body := b[n : n+int(bl)]
			seq, sn := binary.Uvarint(body)
			_, an := binary.Uvarint(body[sn:])
			rest := body[sn+an:]
			nb := binary.AppendUvarint(nil, seq)
			nb = binary.AppendUvarint(nb, uint64(at))
			if len(nb) != sn+an {
				t.Fatal("forged frame changes length")
			}
			nb = append(nb, rest...)
			out = binary.AppendUvarint(out, uint64(len(nb)))
			out = append(out, nb...)
			out = binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(nb))
			b = b[n+int(bl)+4:]
		}
		if err := os.WriteFile(f, out, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// rolledLog is a log of nseg sealed segments of per records each.
func rolledLog(t *testing.T, nseg, per int) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "orders")
	l, err := quill.Open(quill.Config{Dir: dir, Retention: -1, SegmentBytes: 4096, FlushInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	for s := 0; s < nseg; s++ {
		for i := 0; i < per; i++ {
			if _, err := l.Append([]byte(strings.Repeat("x", 200))); err != nil {
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
	return dir
}

// With flags and no readable quill.policy, prune applies the flags given and nothing else
// (decisions 68 and 99). Retention 0 would mean the 24 h default and an age sweep of data the
// operator did not ask to delete, so the log here is years old and every flag is a cap far
// above its size: none may delete anything. The same runs catch a segment-size default that
// makes a large -max-bytes invalid, and -consumer-ttl alone not counting as an explicit flag.
func TestPrunePolicylessAppliesOnlyTheFlagsGiven(t *testing.T) {
	dir := rolledLog(t, 8, 10)
	forgeAge(t, dir, 1_700_000_000)
	if err := os.Remove(filepath.Join(dir, "quill.policy")); err != nil {
		t.Fatal(err)
	}
	before := segmentFiles(t, dir)
	if before < 4 {
		t.Fatalf("setup: %d segments", before)
	}
	for _, args := range [][]string{
		{"prune", "-dir", dir, "-max-bytes", "100000000"},
		{"prune", "-dir", dir, "-consumer-ttl", "1h"},
		{"prune", "-dir", dir, "-max-records", "100000"},
	} {
		code, out, errw := exec(t, "", args...)
		if code != exitOK || segmentFiles(t, dir) != before {
			t.Fatalf("quill %v: exit %d, %d segments of %d: %s%s", args, code, segmentFiles(t, dir), before, out, errw)
		}
	}
}
