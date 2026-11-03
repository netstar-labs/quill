package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// OS.Rename calls rename(2) itself; it must replace a file, report a missing source as
// fs.ErrNotExist in an *os.LinkError, and refuse a directory destination.
func TestOSRename(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (OS{}).Rename(src, dst); err != nil {
		t.Fatalf("rename over a file: %v", err)
	}
	if b, err := os.ReadFile(dst); err != nil || string(b) != "new" {
		t.Fatalf("destination = %q, %v; want new", b, err)
	}

	err := (OS{}).Rename(src, dst)
	var le *os.LinkError
	if !errors.As(err, &le) || le.Op != "rename" || le.Old != src || le.New != dst || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing source: %v, want a rename *os.LinkError wrapping ErrNotExist", err)
	}

	d := filepath.Join(dir, "d")
	if err := os.Mkdir(d, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (OS{}).Rename(src, d); err == nil {
		t.Fatal("rename of a file over a directory succeeded")
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source lost after the refused rename: %v", err)
	}
}

// FS.Remove unlinks a file and never a directory, as Sim does: an empty directory stays,
// a missing name is fs.ErrNotExist, and the error is an *fs.PathError naming the call.
func TestOSRemove(t *testing.T) {
	dir := t.TempDir()
	f, d := filepath.Join(dir, "f"), filepath.Join(dir, "d")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(d, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := (OS{}).Remove(f); err != nil {
		t.Fatalf("remove a file: %v", err)
	}
	if _, err := os.Lstat(f); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("file after remove: %v, want ErrNotExist", err)
	}
	err := (OS{}).Remove(f)
	var pe *fs.PathError
	if !errors.As(err, &pe) || pe.Op != "remove" || pe.Path != f || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing name: %v, want a remove *fs.PathError wrapping ErrNotExist", err)
	}
	if err := (OS{}).Remove(d); err == nil {
		t.Fatal("remove of an empty directory succeeded; the contract says never a directory")
	}
	if fi, err := os.Lstat(d); err != nil || !fi.IsDir() {
		t.Fatalf("directory after refused remove: %v, %v; want it intact", fi, err)
	}
}
