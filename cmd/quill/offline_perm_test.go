package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// The group arrangement is a log created with Config.Perm 0660, and the Reader, the CLI's
// register and Archive take their mode from quill.tail. An offline write verb opens with no
// Perm of its own, so it must carry the tail's mode forward rather than reset it to 0600.
func TestOfflineWriteVerbsKeepTheLogsModeOnQuillTail(t *testing.T) {
	old := syscall.Umask(0)
	defer syscall.Umask(old)
	dir := buildLog(t, quill.Config{Retention: -1, SegmentBytes: 4096, FlushInterval: time.Hour, Perm: 0o660}, 5,
		func(int) []byte { return []byte("rec") })
	tail := filepath.Join(dir, "quill.tail")
	mode := func(p string) fs.FileMode {
		t.Helper()
		info, err := os.Lstat(p)
		if err != nil {
			t.Fatal(err)
		}
		return info.Mode().Perm()
	}
	for _, args := range [][]string{
		{"recover", "-dir", dir},
		{"roll", "-dir", dir},
		{"append", "-dir", dir, "x"},
		{"prune", "-dir", dir, "-max-records", "1000", "-max-bytes", "100000000"},
	} {
		if err := os.Chmod(tail, 0o660); err != nil {
			t.Fatal(err)
		}
		mustRun(t, "", args...)
		if got := mode(tail); got != 0o660 {
			t.Errorf("after quill %v: quill.tail is %o, want 660", args[0], got)
		}
	}
	segs, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil || len(segs) == 0 {
		t.Fatalf("segments %v: %v", segs, err)
	}
	if got := mode(segs[len(segs)-1]); got != 0o660 {
		t.Errorf("newest segment after roll and append is %o, want 660", got)
	}
	// A log written at the default 0600 stays 0600: the mode is carried, never widened.
	plain := newLog(t, 3, func(int) []byte { return []byte("r") })
	mustRun(t, "", "roll", "-dir", plain)
	if got := mode(filepath.Join(plain, "quill.tail")); got != 0o600 {
		t.Errorf("default-mode log: quill.tail is %o after roll, want 600", got)
	}
}
