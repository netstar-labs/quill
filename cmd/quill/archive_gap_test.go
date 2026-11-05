package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// archiveGapSource builds a log of 40 records in several sealed segments and archives it, so the
// destination holds everything and the _archive consumer sits past the last sealed record.
func archiveGapSource(t *testing.T) (dir, dst string) {
	t.Helper()
	dir = newLog(t, 40, func(i int) []byte { return []byte(strings.Repeat("x", 200)) })
	dst = filepath.Join(t.TempDir(), "arch")
	mustRun(t, "", "roll", "-dir", dir)
	mustRun(t, "", "archive", "-dir", dir, "-to", dst)
	return dir, dst
}

// loseSourceRecords adds records the archiver never sees and has retention delete them with
// the older ones, so the source's first retained record is past what the archive holds.
func loseSourceRecords(t *testing.T, dir string) {
	t.Helper()
	rec := strings.Repeat("y", 200)
	for i := 0; i < 10; i++ { // ten sealed segments of four records each
		mustRun(t, "", "append", "-dir", dir, rec, rec, rec, rec)
		mustRun(t, "", "roll", "-dir", dir)
	}
	mustRun(t, "", "prune", "-dir", dir, "-max-records", "8")
}

// A source that lost records the archive never copied is a hole in the backup. `quill archive`
// says so on standard error and exits 3, the unhealthy code, so `quill archive && notify-ok`
// does not report success for an incomplete archive (A5-29, decisions 53 and 85). Before, it
// exited 0.
func TestArchiveExitsUnhealthyWhenTheSourceLostRecordsTheArchiveNeverCopied(t *testing.T) {
	dir, dst := archiveGapSource(t)
	loseSourceRecords(t, dir)
	code, out, errw := exec(t, "", "archive", "-dir", dir, "-to", dst)
	if code != 3 || !strings.Contains(errw, "the source lost records 41 to ") || !strings.Contains(errw, "; the archive does not have them\n") || strings.Contains(errw, "hole") {
		t.Fatalf("archive exit %d, want 3 and the gap named\n%s%s", code, out, errw)
	}
	if code, _, _ := exec(t, "", "check", "-dir", dst); code != 3 {
		t.Fatalf("check on the destination exits %d, want 3", code)
	}
}

// The documented reset, `quill unregister -name _archive`, recreates the consumer at the first
// retained sequence, so the loss is below no offset. Archive compares the destination's own
// quill.tail with the first segment it copies and reports the same gap (A5-29). Before, it
// printed nothing, exited 0 and left the destination holed.
func TestArchiveAfterUnregisterStillReportsTheRecordsTheDestinationLacks(t *testing.T) {
	dir, dst := archiveGapSource(t)
	loseSourceRecords(t, dir)
	mustRun(t, "", "unregister", "-dir", dir, "-name", "_archive")
	code, out, errw := exec(t, "", "archive", "-dir", dir, "-to", dst)
	if code != 3 || !strings.Contains(errw, "the source lost records 41 to ") {
		t.Fatalf("archive exit %d, want 3 and the gap named\n%s%s", code, out, errw)
	}
}

// A destination that starts after the lost records holds no hole: no damaged file was copied, so
// `check` on it prints HEALTHY. The message must be true there too, so it says the archive does
// not have the records, not that it holds the same hole (L5-2, owner-approved 2026-11-04). The
// exit code is 3 as in the case where the destination already held earlier records.
func TestArchiveOfASourceWithADamagedOldestSegmentIntoANewDestinationSaysTheArchiveDoesNotHaveTheRecords(t *testing.T) {
	dir := newLog(t, 40, func(i int) []byte { return []byte(strings.Repeat("x", 200)) })
	mustRun(t, "", "roll", "-dir", dir)
	first, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil || len(first) < 2 {
		t.Fatalf("segments %v, err %v", first, err)
	}
	f, err := os.OpenFile(first[0], os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("ZZZZZZZZ"), 20); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "arch")
	code, out, errw := exec(t, "", "archive", "-dir", dir, "-to", dst)
	if code != 3 || !strings.Contains(errw, "quill: the source lost records 1 to ") || !strings.Contains(errw, "; the archive does not have them\n") || strings.Contains(errw, "hole") {
		t.Fatalf("archive exit %d, want 3 and the loss named without the word hole\n%s%s", code, out, errw)
	}
	if code, out, _ := exec(t, "", "check", "-dir", dst); code != 0 || !strings.HasPrefix(out, "HEALTHY\n") {
		t.Fatalf("check on the destination exits %d: %s, want HEALTHY (the destination holds no hole)", code, out)
	}
}
