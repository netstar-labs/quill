package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The CLI's writer verbs and its readers agree about a -dir reached through a symbolic link:
// the data directory, or a directory below it that the first append creates (A1-29).
func TestWriterVerbsFollowASymbolicLinkInDir(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{link, filepath.Join(link, "orders")} {
		mustRun(t, "", "-dir", dir, "append", "one", "two")
		if out := mustRun(t, "", "-dir", dir, "cat"); !strings.Contains(out, "one") || !strings.Contains(out, "two") {
			t.Fatalf("-dir %s: cat printed %q", dir, out)
		}
	}
	if _, err := os.Stat(filepath.Join(real, "orders", "quill.tail")); err != nil {
		t.Fatalf("the log is not below the link's target: %v", err)
	}
}
