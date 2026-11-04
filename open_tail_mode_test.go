package quill

import (
	"bytes"
	"io/fs"
	"log/slog"
	"strings"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Stage 2 audit A1d-2: Config.Perm is the mode of quill.tail. Since the marker is no longer
// rewritten at Open when it is already right (L3b-8), a restart with a different Perm has to
// change the mode of the existing file in place, or an idle log keeps the old mode until its
// first marker-changing flush: a hardening restart would leave the marker group-readable, and
// a Reader, which takes its own mode from quill.tail, would create files with the old one.

func TestOpenWithADifferentPermChangesTheMarkersModeInPlaceWithoutWritingIt(t *testing.T) {
	for _, tc := range []struct{ from, to fs.FileMode }{
		{0o600, 0o640}, // the documented group arrangement
		{0o640, 0o600}, // a hardening restart
	} {
		t.Run(tc.from.String()+"->"+tc.to.String(), func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.Perm = tc.from })
			appendN(t, l, 3)
			mustFlush(t, l)
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			if got := modeOf(t, s, "/log/"+tailName); got != tc.from {
				t.Fatalf("setup: quill.tail is %v, want %v", got, tc.from)
			}
			before := readTail(t, s, "/log")

			// A full filesystem: no temp file can be created, so the mode must change without
			// the marker being rewritten. NoMaintenance keeps quill.policy, which Open writes
			// through a temp file, out of it.
			var refused int
			s.Inject = noTempFileInjector(&refused)
			l = openSim(t, s, func(c *Config) { c.Perm = tc.to; c.NoMaintenance = true })
			defer l.Close()
			if err := l.Err(); err != nil {
				t.Fatalf("Err() after Open: %v", err)
			}
			if refused != 0 {
				t.Fatalf("Open tried to create a temp file %d times", refused)
			}
			if got := modeOf(t, s, "/log/"+tailName); got != tc.to {
				t.Errorf("quill.tail is %v after Open with Perm %v", got, tc.to)
			}
			if got := readTail(t, s, "/log"); got != before {
				t.Errorf("marker changed: %+v, was %+v", got, before)
			}
		})
	}
}

// A marker that already has the mode is left alone: no open for writing, no chmod.
func TestOpenWithTheSamePermDoesNotTouchTheMarkersMode(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.Perm = 0o640 })
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	fail := &chmodTailFailFS{FS: s}
	l2, err := Open(func() Config {
		cfg := simConfig(s, "/log")
		cfg.Perm = 0o640
		cfg.fs = fail
		return cfg
	}())
	if err != nil {
		t.Fatal(err)
	}
	defer l2.Close()
	if fail.tries != 0 {
		t.Fatalf("Open chmodded a marker that already had the mode %d times", fail.tries)
	}
}

// chmodTailFailFS fails Chmod on quill.tail (a marker another account owns: EPERM), and
// counts the attempts.
type chmodTailFailFS struct {
	fsx.FS
	tries int
}

type chmodTailFailFile struct {
	fsx.File
	fs *chmodTailFailFS
}

func (f *chmodTailFailFS) OpenFile(name string, flag int, perm fs.FileMode) (fsx.File, error) {
	file, err := f.FS.OpenFile(name, flag, perm)
	if err != nil || !strings.HasSuffix(name, "/"+tailName) {
		return file, err
	}
	return &chmodTailFailFile{File: file, fs: f}, nil
}

func (f *chmodTailFailFile) Chmod(fs.FileMode) error {
	f.fs.tries++
	return &fs.PathError{Op: "chmod", Path: tailName, Err: syscall.EPERM}
}

// A marker whose mode cannot be changed does not stop Open (the log is intact; refusing would
// turn a mode change into an outage): Open warns on Config.Logger and the mode stays.
func TestOpenWithAMarkerItCannotChmodStillOpensAndWarns(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.Perm = 0o600 })
	appendN(t, l, 2)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	cfg := simConfig(s, "/log")
	fail := &chmodTailFailFS{FS: s}
	cfg.fs = fail
	cfg.Perm = 0o640
	cfg.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	l, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open with a marker it cannot chmod: %v", err)
	}
	defer l.Close()
	if fail.tries == 0 {
		t.Fatal("Open never tried to change the marker's mode")
	}
	if got := l.Head(); got != 2 {
		t.Errorf("head %d, want 2", got)
	}
	if out := buf.String(); !strings.Contains(out, tailName) || !strings.Contains(out, "EPERM") && !strings.Contains(out, "operation not permitted") {
		t.Errorf("no warning naming the marker and the cause: %q", out)
	}
	if got := modeOf(t, s, "/log/"+tailName); got != 0o600 {
		t.Errorf("quill.tail is %v, want it left at 0600", got)
	}
}
