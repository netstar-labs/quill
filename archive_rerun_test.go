package quill

import (
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
