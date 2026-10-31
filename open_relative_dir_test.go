package quill

import (
	"os"
	"path/filepath"
	"testing"
)

// A relative Config.Dir or OpenReader directory is bound to the working directory at open:
// a later os.Chdir neither fails the writer's flushes nor sends its files into another
// directory that happens to hold the same relative path (A3-33).
func TestRelativeDirSurvivesChdir(t *testing.T) {
	home, elsewhere := t.TempDir(), t.TempDir()
	rel := filepath.Join("data", "log")
	decoy := filepath.Join(elsewhere, rel)
	if err := os.MkdirAll(decoy, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(home)

	l := openDir(t, rel, nil)
	appendN(t, l, 1)
	mustFlush(t, l)
	r, err := OpenReader(rel, ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	t.Chdir(elsewhere)

	appendN(t, l, 1)
	if err := l.Flush(); err != nil {
		t.Fatalf("flush after chdir: %v", err)
	}
	if h, err := r.Head(); err != nil || h != 2 {
		t.Fatalf("reader head after chdir %d, %v; want 2", h, err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("close after chdir: %v", err)
	}
	if ents, err := os.ReadDir(decoy); err != nil || len(ents) != 0 {
		t.Fatalf("the same relative path in the new directory was written to: %v %v", ents, err)
	}
	if _, err := os.Stat(filepath.Join(home, rel, tailName)); err != nil {
		t.Fatalf("the log's own directory has no marker: %v", err)
	}
}
