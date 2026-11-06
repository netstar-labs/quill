package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `tail -n N` is a read from the end, and when N reaches into a hole from sequence 1 (a damaged
// oldest segment) it follows the rule of decision 38: it reports the gap and prints nothing, even
// though the records after the hole are readable. N at the readable count prints them all; one
// more reaches the last lost record and exits 1 naming exactly that record. A3-10 point 3, owner-
// accepted 2026-11-06 as the behaviour, pinned here.
func TestTailThatReachesOneRecordIntoAHoleFromOneReportsTheGapAndPrintsNothing(t *testing.T) {
	dir := newLog(t, 400, func(i int) []byte { return []byte(strings.Repeat("x", 200)) })
	mustRun(t, "", "roll", "-dir", dir)
	segs, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil || len(segs) < 3 {
		t.Fatalf("segments %v, err %v", segs, err)
	}
	f, err := os.OpenFile(segs[0], os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("ZZZZZZZZ"), 20); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	_, st, _ := exec(t, "", "stat", "-dir", dir)
	var head, first, records, holeFrom, holeTo int
	for _, line := range strings.Split(st, "\n") {
		fmt.Sscanf(line, "head %d", &head)
		fmt.Sscanf(line, "first %d", &first)
		fmt.Sscanf(line, "records %d", &records)
		fmt.Sscanf(line, "hole %d-%d", &holeFrom, &holeTo)
	}
	if holeFrom != 1 || holeTo < 2 || records < 2 || first != holeTo+1 || head-holeTo != records {
		t.Fatalf("not a hole from 1 followed by readable records:\n%s", st)
	}

	code, out, errw := exec(t, "", "tail", "-dir", dir, "-n", fmt.Sprint(records))
	if code != 0 || strings.Count(out, "\n") != records || errw != "" {
		t.Fatalf("tail -n %d (all the readable records): exit %d, %d lines, stderr %q", records, code, strings.Count(out, "\n"), errw)
	}
	code, out, errw = exec(t, "", "tail", "-dir", dir, "-n", fmt.Sprint(records+1))
	want := fmt.Sprintf("quill: records %d to %d are unavailable\n", holeTo, holeTo)
	if code != 1 || out != "" || errw != want {
		t.Fatalf("tail -n %d (one into the hole): exit %d, stdout %q, stderr %q; want exit 1, no output, %q", records+1, code, out, errw, want)
	}
}
