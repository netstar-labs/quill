package main

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/netstar-labs/quill"
)

// A verb that runs through withWriter (prune, roll) must report Close's error: Close carries
// the final directory sync, the publication of quill.tail and the lock release (decision 100).
// The directory is made read-only inside fn, so only Close's publication of quill.tail can fail.
func TestWithWriterExitsOneWhenCloseFails(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions do not bind this user")
	}
	for _, tc := range []struct {
		name   string
		fnCode int
	}{{"fn succeeded", exitOK}, {"fn failed", exitError}} {
		dir := newLog(t, 1, func(int) []byte { return []byte("x") })
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
		var out, errw bytes.Buffer
		c := &cli{dir: dir, out: &out, err: &errw}
		code := c.withWriter(quill.Config{Retention: -1}, func(l *quill.Log) int {
			if _, err := l.Append([]byte("y")); err != nil {
				t.Error(err)
			}
			if err := os.Chmod(dir, 0o500); err != nil {
				t.Error(err)
			}
			return tc.fnCode
		})
		if code != exitError {
			t.Errorf("%s: exit %d; want 1", tc.name, code)
		}
		if tc.fnCode == exitOK && errw.Len() == 0 {
			t.Errorf("%s: stderr is empty; Close's error must be reported", tc.name)
		}
		_ = os.Chmod(dir, 0o700)
	}
}
