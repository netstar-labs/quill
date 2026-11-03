package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// fellOutLog is a log whose MaxRecords cap deleted the oldest segments after two consumers were
// registered at the first record: "billing" is protected, "reports" is not. A third, "current",
// is registered after the deletions and sits at the first retained record. The cap overrides
// protection, so both early consumers are below the first retained sequence.
func fellOutLog(t *testing.T) string {
	t.Helper()
	cfg := quill.Config{Dir: filepath.Join(t.TempDir(), "orders"), Retention: -1, SegmentBytes: 4096,
		FlushInterval: time.Hour, MaxRecords: 20}
	l, err := quill.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "", "register", "-dir", cfg.Dir, "-name", "billing", "-protect")
	mustRun(t, "", "register", "-dir", cfg.Dir, "-name", "reports")
	if l, err = quill.Open(cfg); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 120; i++ {
		if _, err := l.Append([]byte(strings.Repeat("x", 200))); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := quill.OpenReader(cfg.Dir, quill.ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if st, err := r.Stat(); err != nil || st.First <= 1 {
		t.Fatalf("the cap deleted nothing: %+v %v", st, err)
	}
	mustRun(t, "", "register", "-dir", cfg.Dir, "-name", "current", "-protect")
	return cfg.Dir
}

func consumerRows(t *testing.T, dir string) map[string]map[string]any {
	t.Helper()
	rows := map[string]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(mustRun(t, "", "consumers", "-dir", dir, "-json")), "\n") {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		rows[row["name"].(string)] = row
	}
	return rows
}

// A consumer below the first retained sequence has lost records for good, and the offline
// surfaces say so (A5-18): a NOTE in the table, `behind_first` in the JSON (added only when true,
// so a parser written before sees the same keys for every other consumer), and a warning in check
// that leaves the log HEALTHY.
func TestConsumersNoteAConsumerBehindTheFirstRetainedSequence(t *testing.T) {
	dir := fellOutLog(t)
	rows := consumerRows(t, dir)
	for _, name := range []string{"billing", "reports"} {
		if rows[name]["behind_first"] != true {
			t.Errorf("%s: no behind_first in %v", name, rows[name])
		}
	}
	if _, ok := rows["current"]["behind_first"]; ok {
		t.Errorf("a consumer at the first retained record is flagged: %v", rows["current"])
	}
	for _, row := range rows {
		if _, ok := row["stale"]; ok {
			t.Errorf("no ConsumerTTL is set but %v is stale", row)
		}
	}
	var s struct{ First uint64 }
	st := mustRun(t, "", "stat", "-dir", dir, "-json")
	if err := json.Unmarshal([]byte(st), &s); err != nil || s.First <= 1 {
		t.Fatalf("stat %q: %v", st, err)
	}
	for _, line := range strings.Split(strings.TrimSpace(mustRun(t, "", "consumers", "-dir", dir)), "\n")[1:] {
		f := strings.Fields(line)
		behind := strings.Contains(line, "behind the first retained sequence")
		if (f[0] == "current") == behind {
			t.Errorf("table row %q", line)
		}
		if behind && !strings.Contains(line, "records 1 to "+strconv.FormatUint(s.First-1, 10)+" are gone") {
			t.Errorf("the note does not say which records are gone (first %d): %q", s.First, line)
		}
	}
	out := mustRun(t, "", "check", "-dir", dir)
	if !strings.HasPrefix(out, "HEALTHY\n") || strings.Count(out, "behind the first retained sequence") != 2 {
		t.Errorf("check:\n%s", out)
	}
}

// A protected consumer that has been behind and silent for ConsumerTTL no longer protects
// segments; the file still says protected, so the NOTE derives the staleness from the policy's
// TTL and the file's mtime, the way the writer's pass does (decision 45).
func TestConsumersNoteAStaleProtectedConsumer(t *testing.T) {
	for _, tc := range []struct {
		name  string
		ttl   time.Duration
		stale bool
	}{{"expired", time.Nanosecond, true}, {"within the TTL", time.Hour, false}, {"no TTL", 0, false}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := newLogWithTTL(t, tc.ttl)
			rows := consumerRows(t, dir)
			if got := rows["slow"]["stale"] == true; got != tc.stale {
				t.Errorf("slow: stale %v, want %v: %v", got, tc.stale, rows["slow"])
			}
			for _, name := range []string{"caughtup", "plain"} {
				if _, ok := rows[name]["stale"]; ok {
					t.Errorf("%s is not stale: %v", name, rows[name])
				}
			}
			out := mustRun(t, "", "consumers", "-dir", dir)
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "slow") && strings.Contains(line, "stale") != tc.stale {
					t.Errorf("note: %q", line)
				}
			}
		})
	}
}

// A consumer at Head+1 has read everything: that is the normal state, not an ahead one, in the
// JSON row and in the table's NOTE. One at Head+2 is ahead (the boundary is exact).
func TestL2ACaughtUpConsumerIsNotAhead(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	mustRun(t, "", "register", "-dir", dir, "-name", "caughtup")
	mustRun(t, "", "commit", "-dir", dir, "-name", "caughtup", "-offset", "6")
	aheadConsumer(t, dir, "ahead", "7")
	rows := consumerRows(t, dir)
	if _, has := rows["caughtup"]["ahead"]; has {
		t.Errorf("a consumer at Head+1 is flagged ahead: %v", rows["caughtup"])
	}
	if rows["ahead"]["ahead"] != true {
		t.Errorf("a consumer at Head+2 is not flagged ahead: %v", rows["ahead"])
	}
	for _, line := range strings.Split(strings.TrimSpace(mustRun(t, "", "consumers", "-dir", dir)), "\n")[1:] {
		f := strings.Fields(line)
		if got := strings.Contains(line, "ahead of the head"); got != (f[0] == "ahead") {
			t.Errorf("table row %q: ahead note %v", line, got)
		}
	}
}

// A protected consumer below the first retained sequence is probed at the first retained record
// (decision 82), so it is still found stale; an unprotected one never is.
func TestL2StaleProbeOfAConsumerBehindTheFirstRetainedSequence(t *testing.T) {
	// Flush after every append so the cap deletes old segments but leaves records retained: the
	// stale probe needs a first retained record to read.
	cfg := quill.Config{Dir: filepath.Join(t.TempDir(), "orders"), Retention: -1, SegmentBytes: 4096,
		FlushInterval: time.Hour, MaxRecords: 20, ConsumerTTL: time.Nanosecond}
	l, err := quill.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	dir := cfg.Dir
	mustRun(t, "", "register", "-dir", dir, "-name", "billing", "-protect")
	mustRun(t, "", "register", "-dir", dir, "-name", "reports")
	if l, err = quill.Open(cfg); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 120; i++ {
		if _, err := l.Append([]byte(strings.Repeat("x", 200))); err != nil {
			t.Fatal(err)
		}
		if err := l.Flush(); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	var s struct{ First, Records uint64 }
	if err := json.Unmarshal([]byte(mustRun(t, "", "stat", "-dir", dir, "-json")), &s); err != nil || s.First <= 1 || s.Records == 0 {
		t.Fatalf("want a log whose cap deleted the oldest records but kept some: %+v %v", s, err)
	}
	old := time.Now().Add(-time.Hour)
	for _, n := range []string{"billing", "reports"} {
		if err := os.Chtimes(filepath.Join(dir, "consumers", n), old, old); err != nil {
			t.Fatal(err)
		}
	}
	rows := consumerRows(t, dir)
	if rows["billing"]["behind_first"] != true {
		t.Fatalf("billing is not behind the first retained sequence: %v", rows["billing"])
	}
	if rows["billing"]["stale"] != true {
		t.Errorf("a stale protected consumer behind the first retained sequence is not flagged: %v", rows["billing"])
	}
	if _, has := rows["reports"]["stale"]; has {
		t.Errorf("an unprotected consumer is flagged stale: %v", rows["reports"])
	}
	for _, line := range strings.Split(strings.TrimSpace(mustRun(t, "", "consumers", "-dir", dir)), "\n")[1:] {
		if strings.HasPrefix(line, "billing") && !strings.Contains(line, "stale") {
			t.Errorf("table row %q has no stale note", line)
		}
	}
}

// newLogWithTTL has a protected consumer that is behind ("slow"), one that is caught up and one
// that is unprotected and behind, all with offset files an hour old.
func newLogWithTTL(t *testing.T, ttl time.Duration) string {
	t.Helper()
	dir := buildLog(t, quill.Config{Retention: -1, SegmentBytes: 4096, FlushInterval: time.Hour, ConsumerTTL: ttl}, 10,
		func(i int) []byte { return []byte("record") })
	mustRun(t, "", "register", "-dir", dir, "-name", "slow", "-protect")
	mustRun(t, "", "register", "-dir", dir, "-name", "caughtup", "-protect")
	mustRun(t, "", "commit", "-dir", dir, "-name", "caughtup", "-offset", "11")
	mustRun(t, "", "register", "-dir", dir, "-name", "plain")
	old := time.Now().Add(-time.Hour)
	for _, n := range []string{"slow", "caughtup", "plain"} {
		if err := os.Chtimes(filepath.Join(dir, "consumers", n), old, old); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// prune prints how many segments it deleted under a consumer, and names the consumer on standard
// error, where the offline verbs already print what a pass lost to damage.
func TestPrunePrintsDroppedAndNamesTheConsumer(t *testing.T) {
	dir := newLog(t, 100, func(i int) []byte { return []byte(strings.Repeat("x", 200)) })
	mustRun(t, "", "register", "-dir", dir, "-name", "billing")
	code, out, errw := exec(t, "", "prune", "-dir", dir, "-max-records", "10")
	if code != exitOK {
		t.Fatalf("exit %d: %s%s", code, out, errw)
	}
	_, after, ok := strings.Cut(strings.TrimSpace(out), ", dropped ")
	if n, err := strconv.Atoi(after); !ok || err != nil || n == 0 {
		t.Errorf("prune printed %q, want a dropped count", out)
	}
	if !strings.Contains(errw, "consumer=billing") {
		t.Errorf("stderr does not name the consumer:\n%s", errw)
	}
	// Nothing to delete: dropped 0 is still printed, so the field is always there.
	if out := mustRun(t, "", "prune", "-dir", dir, "-max-records", "10"); !strings.HasSuffix(strings.TrimSpace(out), "dropped 0") {
		t.Errorf("second prune printed %q", out)
	}
}
