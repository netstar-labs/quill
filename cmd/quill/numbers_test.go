package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// snapshot maps every path under dir to its bytes, so two snapshots compare equal only
// when no file was created, removed or changed.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			m[rel] = "<dir>"
			return nil
		}
		b, err := os.ReadFile(p)
		m[rel] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// manySegments writes 40 records into 256-byte segments with a retention-off policy.
func manySegments(t *testing.T) string {
	t.Helper()
	return buildLog(t, quill.Config{Retention: -1, SegmentBytes: 256, FlushInterval: time.Hour, NoMaintenance: true}, 40,
		func(i int) []byte { return []byte(fmt.Sprintf("record %02d", i)) })
}

var refusedNumbers = []string{"1e6", "1e3", "1M", "1K", "0x10", "-1", "+1", "1_000", "12abc", "10 ", " 10", "1.5", "", "99999999999999999999"}

func TestPruneRefusesANumberThatIsNotPlainDecimalAndChangesNothing(t *testing.T) {
	dir := manySegments(t)
	before := snapshot(t, dir)
	for _, flag := range []string{"-max-bytes", "-min-free-bytes", "-max-records"} {
		for _, bad := range refusedNumbers {
			if bad == "" {
				continue // an empty flag value means the flag was not given
			}
			code, out, _ := exec(t, "", "prune", "-dir", dir, flag, bad)
			if code != exitUsage || out != "" {
				t.Errorf("prune %s %q: exit %d, stdout %q; want %d and nothing", flag, bad, code, out, exitUsage)
			}
			if !reflect.DeepEqual(snapshot(t, dir), before) {
				t.Fatalf("prune %s %q changed the log", flag, bad)
			}
		}
	}
	// Above int64 is refused too: the caps are int64.
	if code, _, _ := exec(t, "", "prune", "-dir", dir, "-max-records", "9223372036854775808"); code != exitUsage {
		t.Errorf("over int64: exit %d", code)
	}
	if !reflect.DeepEqual(snapshot(t, dir), before) {
		t.Fatal("an over-large cap changed the log")
	}
}

func TestPruneAcceptsAPlainDecimalCap(t *testing.T) {
	dir := manySegments(t)
	out := mustRun(t, "", "prune", "-dir", dir, "-max-records", "1000000")
	if want := "segments"; len(out) < len(want) || out[:len(want)] != want {
		t.Fatalf("%q", out)
	}
	st, err := quill.OpenReader(dir, quill.ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if s, _ := st.Stat(); s.First != 1 {
		t.Fatalf("a cap of a million records deleted records: first %d", s.First)
	}
}

func TestCommitRefusesAnOffsetThatIsNotPlainDecimalAndChangesNothing(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("r") })
	mustRun(t, "", "register", "-dir", dir, "-name", "c")
	mustRun(t, "", "commit", "-dir", dir, "-name", "c", "-offset", "3")
	before := snapshot(t, dir)
	for _, bad := range append([]string{"18446744073709551616"}, refusedNumbers...) {
		code, out, _ := exec(t, "", "commit", "-dir", dir, "-name", "c", "-offset", bad)
		if code != exitUsage || out != "" { // "" is the exactly-one-of rule, also usage
			t.Errorf("commit -offset %q: exit %d, stdout %q", bad, code, out)
		}
		if !reflect.DeepEqual(snapshot(t, dir), before) {
			t.Fatalf("commit -offset %q changed the log", bad)
		}
	}
}

func TestCommitWithAMalformedNumberDoesNotRegisterTheConsumer(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("r") })
	before := snapshot(t, dir)
	for _, args := range [][]string{{"-offset", "1e3"}, {"-time", "@1e6"}, {"-time", "@-1"}, {"-time", "@0x10"}, {"-time", "@5s"}} {
		code, _, _ := exec(t, "", append([]string{"commit", "-dir", dir, "-name", "fresh"}, args...)...)
		if code != exitUsage {
			t.Errorf("commit %v: exit %d", args, code)
		}
		if !reflect.DeepEqual(snapshot(t, dir), before) {
			t.Fatalf("commit %v created consumer state", args)
		}
	}
}

func TestParseTimeUnixFormIsPlainDecimalOnly(t *testing.T) {
	now := time.Now()
	if got, err := parseTime("@1700000000", now); err != nil || !got.Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("%v %v", got, err)
	}
	for _, bad := range []string{"@", "@1e6", "@-1", "@+1", "@0x10", "@1_000", "@12abc", "@ 5", "@9223372036854775808"} {
		if _, err := parseTime(bad, now); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
