package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// quill ls skips a directory only when it is not a log (ErrNoLog). A log it cannot open (a
// damaged or future quill.format) gets an "error:" row, the other logs are still listed, and
// the exit code is 1, as stat and check give for the same log (decision 53, A5-44).
func TestLsReportsALogItCannotOpenAndExits1(t *testing.T) {
	good := newLog(t, 3, func(i int) []byte { return []byte("r") })
	parent := filepath.Dir(good)
	bad := filepath.Join(parent, "bad")
	if err := os.MkdirAll(bad, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "quill.format"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "not-a-log"), 0o700); err != nil {
		t.Fatal(err)
	}

	code, out, errw := exec(t, "", "-dir", parent, "ls")
	if code != exitError {
		t.Errorf("ls with an unopenable log exits %d, want %d (stdout %q, stderr %q)", code, exitError, out, errw)
	}
	if !strings.Contains(out, "orders") {
		t.Errorf("the good log is not listed:\n%s", out)
	}
	var badRow string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "bad") {
			badRow = line
		}
	}
	if !strings.Contains(badRow, "error:") || !strings.Contains(badRow, "corrupt log") {
		t.Errorf("no error row for the log that cannot be opened:\n%s", out)
	}
	if strings.Contains(out, "not-a-log") {
		t.Errorf("a directory that is not a log is listed:\n%s", out)
	}
	if strings.Contains(errw, "no logs") {
		t.Errorf("says there are no logs though one was listed: %q", errw)
	}
	// The same log, asked of directly, is the same exit code.
	if code, _, _ := exec(t, "", "-dir", bad, "stat"); code != exitError {
		t.Errorf("stat on the bad log exits %d, want %d", code, exitError)
	}
}

// With no log that opens and one that does not, ls does not say "no logs": it is not true.
func TestLsOnlyAnUnopenableLogIsNotReportedAsNoLogs(t *testing.T) {
	parent := t.TempDir()
	bad := filepath.Join(parent, "bad")
	if err := os.MkdirAll(bad, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "quill.format"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errw := exec(t, "", "-dir", parent, "ls")
	if code != exitError || !strings.Contains(out, "error:") || strings.Contains(errw, "no logs") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errw)
	}
}

// A parent with only directories that are not logs still lists nothing, says so and exits 0.
func TestLsWithNoLogsIsUnchanged(t *testing.T) {
	parent := t.TempDir()
	if err := os.MkdirAll(filepath.Join(parent, "plain"), 0o700); err != nil {
		t.Fatal(err)
	}
	code, out, errw := exec(t, "", "-dir", parent, "ls")
	if code != exitOK || strings.Contains(out, "plain") || !strings.Contains(errw, "no logs under") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errw)
	}
}
