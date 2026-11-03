package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// staleFixture is a log of five records with two protected consumers, "behind" at record 1 and
// "athead" at record 5 (the head). It returns the CLI value, a reader and the consumers' rows.
// The tests set each offset file's mtime and pass `now` to the stale probe themselves, so the
// boundaries are exact whatever the wall clock says.
func staleFixture(t *testing.T) (*cli, *quill.Reader, map[string]quill.ConsumerInfo, []quill.Record) {
	t.Helper()
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	mustRun(t, "", "register", "-dir", dir, "-name", "behind", "-protect")
	mustRun(t, "", "register", "-dir", dir, "-name", "athead", "-protect")
	mustRun(t, "", "register", "-dir", dir, "-name", "caughtup", "-protect")
	mustRun(t, "", "commit", "-dir", dir, "-name", "athead", "-offset", "5")
	mustRun(t, "", "commit", "-dir", dir, "-name", "caughtup", "-offset", "6")
	r, err := quill.OpenReader(dir, quill.ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	list, err := r.Consumers()
	if err != nil {
		t.Fatal(err)
	}
	infos := map[string]quill.ConsumerInfo{}
	for _, ci := range list {
		infos[ci.Name] = ci
	}
	if infos["behind"].Offset != 1 || infos["athead"].Offset != 5 || infos["caughtup"].Offset != 6 {
		t.Fatalf("offsets: %+v", infos)
	}
	head, err := r.Head()
	if err != nil || head != 5 {
		t.Fatalf("head %d %v", head, err)
	}
	recs, err := r.Read(1, 5)
	if err != nil || len(recs) != 5 {
		t.Fatalf("read: %v %v", recs, err)
	}
	return &cli{dir: dir}, r, infos, recs
}

func setMtime(t *testing.T, c *cli, name string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(filepath.Join(c.dir, "consumers", name), at, at); err != nil {
		t.Fatal(err)
	}
}

// `quill consumers` counts a consumer stale at the writer's boundaries (decision 45): the offset
// file at least TTL old, the record at the offset at least TTL old, the offset at most Head.
func TestConsumersStaleProbeActsExactlyAtTheTTL(t *testing.T) {
	const ttl = time.Hour
	c, r, infos, recs := staleFixture(t)
	head := uint64(5)
	// Records written across a second boundary carry different stamps, so each case reads the
	// stamp of the record at its own consumer's offset. Seconds are whole, so the comparisons
	// below meet at equality, not near it.
	recAt := func(name string) time.Time { return time.Unix(recs[infos[name].Offset-1].At, 0) }
	for _, tc := range []struct {
		name      string
		consumer  string
		now       time.Time // the probe's clock
		mtime     time.Time // the offset file's modification time
		wantStale bool
	}{
		// The file is old, so the record's age decides.
		{"record TTL-1s old", "behind", recAt("behind").Add(ttl - time.Second), recAt("behind").Add(-2 * ttl), false},
		{"record exactly TTL old", "behind", recAt("behind").Add(ttl), recAt("behind").Add(-2 * ttl), true},
		// The records are far older than the TTL, so the file's age decides.
		{"file TTL-1s old", "behind", recAt("behind").Add(3 * ttl), recAt("behind").Add(2*ttl + time.Second), false},
		{"file exactly TTL old", "behind", recAt("behind").Add(3 * ttl), recAt("behind").Add(2 * ttl), true},
		// Offset == Head is behind; Head+1 is caught up.
		{"offset == Head", "athead", recAt("athead").Add(3 * ttl), recAt("athead"), true},
		{"offset == Head+1", "caughtup", recAt("athead").Add(3 * ttl), recAt("athead"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setMtime(t, c, tc.consumer, tc.mtime)
			if got := c.stale(r, infos[tc.consumer], ttl, head, tc.now); got != tc.wantStale {
				t.Errorf("stale = %v, want %v", got, tc.wantStale)
			}
		})
	}
}
