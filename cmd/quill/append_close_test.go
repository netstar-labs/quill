package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// lockAtEOF is a stdin that removes write permission on dir (restored on cleanup) once it
// has been read to the end: the verb has opened the log by then and has not yet appended, so
// only the publication of quill.tail, which needs a new file in the directory, can fail.
// It skips when the permission does not bind (root, Windows).
type lockAtEOF struct {
	r   io.Reader
	dir string
	t   *testing.T
}

func newLockAtEOF(t *testing.T, dir, text string) *lockAtEOF {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions do not bind this user")
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	return &lockAtEOF{strings.NewReader(text), dir, t}
}

func (l *lockAtEOF) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	if err == io.EOF {
		if cerr := os.Chmod(l.dir, 0o500); cerr != nil {
			l.t.Error(cerr)
		}
	}
	return n, err
}

func TestAppendPrintsTheSequenceAndExitsZeroWhenCloseSucceeds(t *testing.T) {
	for _, sync := range []string{"-sync=true", "-sync=false"} {
		dir := filepath.Join(t.TempDir(), "log")
		code, out, errw := exec(t, "", "append", "-dir", dir, sync, "a", "b")
		if code != exitOK || out != "2\n" || errw != "" {
			t.Errorf("append %s: exit %d, stdout %q, stderr %q; want 0, \"2\\n\", empty", sync, code, out, errw)
		}
	}
}

func TestAppendExitsOneWhenCloseFails(t *testing.T) {
	for _, tc := range []struct {
		sync string
		out  string // nothing is durable without -sync; with it the records are
	}{{"-sync=false", ""}, {"-sync=true", "2\n"}} {
		dir := newLog(t, 1, func(int) []byte { return []byte("x") })
		var out, errw bytes.Buffer
		code := run([]string{"append", "-dir", dir, tc.sync}, newLockAtEOF(t, dir, "y\n"), &out, &errw)
		if code != exitError {
			t.Errorf("append %s: exit %d, stdout %q, stderr %q; want 1", tc.sync, code, out.String(), errw.String())
		}
		if out.String() != tc.out {
			t.Errorf("append %s: stdout %q; want %q", tc.sync, out.String(), tc.out)
		}
		if errw.Len() == 0 {
			t.Errorf("append %s: stderr is empty; the error must be reported", tc.sync)
		}
	}
}

func TestAppendReportsASyncedSequenceWhenCloseFailsAfterTheSync(t *testing.T) {
	var out, errw bytes.Buffer
	c := &cli{out: &out, err: &errw}
	if code := c.finishAppend(7, true, errors.New("close boom")); code != exitError {
		t.Errorf("exit %d; want 1", code)
	}
	if out.String() != "7\n" {
		t.Errorf("stdout %q; want the last sequence", out.String())
	}
	if !strings.Contains(errw.String(), "close boom") {
		t.Errorf("stderr %q; want the Close error", errw.String())
	}
	out.Reset()
	errw.Reset()
	if code := c.finishAppend(7, false, errors.New("close boom")); code != exitError || out.String() != "" {
		t.Errorf("not durable: exit %d, stdout %q; want 1 and nothing", code, out.String())
	}
}
