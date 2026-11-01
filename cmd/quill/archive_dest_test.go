package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `quill archive -to` at an existing directory that holds files and has no quill.format (a
// mistyped path) exits non-zero with a message naming the directory, and neither replaces the
// unrelated segment-named file in it nor adds quill.format, quill.tail or consumers/ (A3-14).
func TestArchiveToAPopulatedDirectoryThatIsNotALogFailsAndLeavesItUntouched(t *testing.T) {
	src := newLog(t, 20, func(i int) []byte { return []byte("record") })
	mustRun(t, "", "roll", "-dir", src)
	dst := filepath.Join(t.TempDir(), "unrelated")
	if err := os.Mkdir(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	const precious = "precious unrelated data"
	if err := os.WriteFile(filepath.Join(dst, "0000000000.log"), []byte(precious), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "notes.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errw := exec(t, "", "archive", "-dir", src, "-to", dst)
	if code != exitError {
		t.Fatalf("exit %d, want %d\n%s%s", code, exitError, out, errw)
	}
	if !strings.Contains(errw, dst) || !strings.Contains(errw, "quill.format") {
		t.Fatalf("the message does not name the directory and quill.format: %q", errw)
	}
	if got, err := os.ReadFile(filepath.Join(dst, "0000000000.log")); err != nil || string(got) != precious {
		t.Fatalf("0000000000.log is %q (%v), want it untouched", got, err)
	}
	es, err := os.ReadDir(dst)
	if err != nil || len(es) != 2 {
		t.Fatalf("the directory now holds %v (%v), want only the two planted files", es, err)
	}
}
