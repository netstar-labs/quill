package atomicfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

func read(t *testing.T, fsys fsx.FS, path string) string {
	t.Helper()
	f, err := fsys.OpenFile(path, fsx.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, info.Size())
	if _, err := f.ReadAt(buf, 0); err != nil && info.Size() > 0 {
		t.Fatal(err)
	}
	return string(buf)
}

func TestWriteReplacesAndLeavesNoTempFileOnEitherFileSystem(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name string
		fsys fsx.FS
		path string
	}{{"os", fsx.OS{}, filepath.Join(dir, "state")}, {"sim", fsx.NewSim(), "/state"}} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Write(tc.fsys, tc.path, []byte("first"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := Write(tc.fsys, tc.path, []byte("second, longer than first"), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := read(t, tc.fsys, tc.path); got != "second, longer than first" {
				t.Fatalf("got %q", got)
			}
			names, err := tc.fsys.ReadDir(filepath.Dir(tc.path))
			if err != nil {
				t.Fatal(err)
			}
			for _, n := range names {
				if strings.HasPrefix(n, TempPrefix) {
					t.Fatalf("temp file left behind: %s", n)
				}
			}
			info, err := tc.fsys.Lstat(tc.path)
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("mode %v %v", info, err)
			}
		})
	}
}

func TestModeIsExactWhateverTheUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)
	path := filepath.Join(t.TempDir(), "state")
	if err := Write(fsx.OS{}, path, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want 0640 under umask 077", info.Mode())
	}
}

func TestAFailureAtAnyStepLeavesTheTargetUntouchedAndNoTempFile(t *testing.T) {
	for _, failAt := range []fsx.Op{fsx.OpCreate, fsx.OpWrite, fsx.OpSync, fsx.OpRename} {
		s := fsx.NewSim()
		if err := Write(s, "/state", []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		s.Inject = func(op fsx.Op, name string) error {
			if op == failAt && (op != fsx.OpCreate || strings.Contains(name, TempPrefix)) {
				return syscall.ENOSPC
			}
			return nil
		}
		err := Write(s, "/state", []byte("new"), 0o600)
		if !errors.Is(err, syscall.ENOSPC) {
			t.Fatalf("fail at %s: %v", failAt, err)
		}
		s.Inject = nil
		if got := read(t, s, "/state"); got != "old" {
			t.Fatalf("fail at %s: target = %q", failAt, got)
		}
		names, _ := s.ReadDir("/")
		if len(names) != 1 {
			t.Fatalf("fail at %s: directory holds %v", failAt, names)
		}
	}
}

// writeDurable is Write followed by a directory fsync, so the rename survives a crash.
func writeDurable(fsys fsx.FS, path string, data []byte) error {
	if err := Write(fsys, path, data, 0o600); err != nil {
		return err
	}
	return fsys.SyncDir(filepath.Dir(path))
}

func TestACrashAtAnyPointLeavesTheOldOrTheWholeNewFile(t *testing.T) {
	// Crash before every operation of a Write, with every subset of unsynced items kept.
	for at := 1; ; at++ {
		s := fsx.NewSim()
		if err := writeDurable(s, "/state", []byte("old")); err != nil {
			t.Fatal(err)
		}
		count := 0
		crashed := false
		s.Inject = func(op fsx.Op, name string) error {
			count++
			if count == at {
				crashed = true
				return fsx.ErrCrash
			}
			return nil
		}
		s.CrashKeep = nil
		err := Write(s, "/state", []byte("new!"), 0o600)
		if !crashed {
			if err != nil {
				t.Fatal(err)
			}
			break // the whole write ran; every crash point has been tried
		}
		if !errors.Is(err, fsx.ErrCrash) {
			t.Fatalf("crash at op %d: %v", at, err)
		}
		// Replay the same point with every subset of the unsynced items.
		for mask := 0; ; mask++ {
			s := fsx.NewSim()
			if err := writeDurable(s, "/state", []byte("old")); err != nil {
				t.Fatal(err)
			}
			count := 0
			var n int
			s.CrashKeep = func(i, total int) bool { n = total; return mask&(1<<i) != 0 }
			s.Inject = func(op fsx.Op, name string) error {
				count++
				if count == at {
					return fsx.ErrCrash
				}
				return nil
			}
			_ = Write(s, "/state", []byte("new!"), 0o600)
			got, err := s.Durable("/state")
			if err != nil {
				t.Fatalf("crash at op %d mask %b: %v", at, mask, err)
			}
			if string(got) != "old" && string(got) != "new!" {
				t.Fatalf("crash at op %d mask %b: durable = %q", at, mask, got)
			}
			if mask >= 1<<n-1 {
				break
			}
		}
	}
}

func TestWriteOverASymlinkReplacesTheLinkAndNeverWritesItsTarget(t *testing.T) {
	// The temp file is created exclusively, without following links, and renamed over the
	// target name: a symbolic link planted at the target is replaced, and what it pointed at
	// is never opened. (The temp name is random, so a link planted there would only make the
	// exclusive create fail.)
	dir := t.TempDir()
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(dir, "state")); err != nil {
		t.Fatal(err)
	}
	if err := Write(fsx.OS{}, filepath.Join(dir, "state"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(victim); string(got) != "keep" {
		t.Fatalf("the link's target was written: %q", got)
	}
	info, err := os.Lstat(filepath.Join(dir, "state"))
	if err != nil || info.Mode()&fs.ModeSymlink != 0 {
		t.Fatalf("state is still a link: %v %v", info, err)
	}
}

func TestTempNameMatchesJoin(t *testing.T) {
	suffix := [8]byte{0x00, 0x01, 0xab, 0xcd, 0xef, 0x10, 0x7f, 0xff}
	for _, path := range []string{"/x", "x", "/a/b", "a/b", "/", "a/b/", "./x", "../x", "//x", "a//b", ""} {
		dir := filepath.Dir(path)
		want := filepath.Join(dir, TempPrefix+"0001abcdef107fff")
		if got := tempName(dir, suffix); got != want {
			t.Errorf("tempName(%q) = %q, want %q", dir, got, want)
		}
	}
}

// failStepFS opens files whose Chmod or Close fails, which the simulator has no injection
// point for, and counts the renames Write attempts.
type failStepFS struct {
	fsx.FS
	chmodErr, closeErr error
	renames            int
}

type failStepFile struct {
	fsx.File
	fs *failStepFS
}

func (f *failStepFS) OpenFile(name string, flag int, perm fs.FileMode) (fsx.File, error) {
	file, err := f.FS.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return &failStepFile{File: file, fs: f}, nil
}

func (f *failStepFS) Rename(oldpath, newpath string) error {
	f.renames++
	return f.FS.Rename(oldpath, newpath)
}

func (f *failStepFile) Chmod(mode fs.FileMode) error {
	if f.fs.chmodErr != nil {
		return f.fs.chmodErr
	}
	return f.File.Chmod(mode)
}

func (f *failStepFile) Close() error {
	cerr := f.File.Close() // the descriptor is released either way
	if f.fs.closeErr != nil {
		return f.fs.closeErr
	}
	return cerr
}

// A file that cannot be given its exact mode, or whose close reports an error (a writeback
// failure on some file systems), must not be installed: Write returns the error, removes the
// temp file and leaves the target as it was.
func TestWriteFailsWhenChmodOrCloseFails(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(f *failStepFS, err error)
	}{
		{"chmod", func(f *failStepFS, err error) { f.chmodErr = err }},
		{"close", func(f *failStepFS, err error) { f.closeErr = err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim := fsx.NewSim()
			if err := Write(sim, "/state", []byte("old"), 0o600); err != nil {
				t.Fatal(err)
			}
			w := &failStepFS{FS: sim}
			tc.setup(w, syscall.EIO)
			err := Write(w, "/state", []byte("new"), 0o640)
			if !errors.Is(err, syscall.EIO) {
				t.Fatalf("Write = %v, want the %s error", err, tc.name)
			}
			if w.renames != 0 {
				t.Fatalf("Write renamed the temp file over the target after a failed %s", tc.name)
			}
			if got := read(t, sim, "/state"); got != "old" {
				t.Fatalf("target = %q, want it untouched", got)
			}
			names, err := sim.ReadDir("/")
			if err != nil || len(names) != 1 {
				t.Fatalf("directory holds %v (%v), want only the target", names, err)
			}
		})
	}
}
