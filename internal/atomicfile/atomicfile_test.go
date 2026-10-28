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

func TestACrashAtAnyPointLeavesTheOldOrTheWholeNewFile(t *testing.T) {
	// Crash before every operation of a Write, with every subset of unsynced items kept.
	for at := 1; ; at++ {
		s := fsx.NewSim()
		if err := WriteSyncDir(s, "/state", []byte("old"), 0o600); err != nil {
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
			if err := WriteSyncDir(s, "/state", []byte("old"), 0o600); err != nil {
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
