package main

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// failWriter is a standard output that refuses every write, as /dev/full or a closed
// descriptor does.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("no space left on device") }

// A verb whose output could not be written must not exit 0, however little it printed:
// the printer buffers 64 KiB, so a small output used to fail only when the buffer was
// flushed, and that error was dropped.
func TestAFailedWriteToStandardOutputIsAnError(t *testing.T) {
	dir := newLog(t, 3, func(i int) []byte { return []byte("record") })
	for _, args := range [][]string{
		{"cat", "-dir", dir},
		{"cat", "-dir", dir, "-json"},
		{"read", "-dir", dir, "-from", "1"},
		{"tail", "-dir", dir, "-n", "2"},
		{"stat", "-dir", dir},
		{"stat", "-dir", dir, "-json"},
		{"check", "-dir", dir},
		{"check", "-dir", dir, "-json"},
		{"consumers", "-dir", dir},
		{"ls", "-dir", filepath.Dir(dir)},
		{"version"},
		{"help"},
	} {
		var errw bytes.Buffer
		code := run(args, strings.NewReader(""), failWriter{}, &errw)
		if code != exitError {
			t.Errorf("quill %v with a failing stdout: exit %d, want %d", args, code, exitError)
		}
		if !strings.Contains(errw.String(), "no space left on device") {
			t.Errorf("quill %v: stderr %q does not say why", args, errw.String())
		}
	}
}

// A follow that cannot write must stop rather than wait for records it cannot print.
func TestFollowStopsWhenStandardOutputFails(t *testing.T) {
	dir := newLog(t, 3, func(i int) []byte { return []byte("record") })
	done := make(chan int, 1)
	go func() {
		done <- run([]string{"tail", "-dir", dir, "-follow", "-from", "1"}, strings.NewReader(""), failWriter{}, io.Discard)
	}()
	select {
	case code := <-done:
		if code != exitError {
			t.Fatalf("exit %d, want %d", code, exitError)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("tail -follow kept waiting after its output failed")
	}
}
