package fsx

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// T27: the simulator's own model (decision 61) is tested before anything is tested against it.

func mustOpen(t *testing.T, fsys FS, name string, flag int) File {
	t.Helper()
	f, err := fsys.OpenFile(name, flag, 0o600)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	return f
}

func mustWrite(t *testing.T, f File, data []byte, off int64) {
	t.Helper()
	if _, err := f.WriteAt(data, off); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func mustSync(t *testing.T, f File) {
	t.Helper()
	if err := f.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
}

func mustSyncDir(t *testing.T, s *Sim) {
	t.Helper()
	if err := s.SyncDir("/"); err != nil {
		t.Fatalf("syncdir /: %v", err)
	}
}

func durable(t *testing.T, s *Sim, name string) []byte {
	t.Helper()
	b, err := s.Durable(name)
	if err != nil {
		t.Fatalf("durable %s: %v", name, err)
	}
	return b
}

// subset enumerates every subset of n items as a keep function.
func subset(mask int) func(i, n int) bool { return func(i, _ int) bool { return mask&(1<<i) != 0 } }

func pattern(n int, b byte) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func TestAnUnsyncedWriteIsLostOrTornAtPageGranularityAndLostPagesReadAsZeros(t *testing.T) {
	// A file created, synced and named durably, then extended by 2.5 pages without a sync.
	const size = 2*pageSize + pageSize/2
	outcomes := map[string]bool{}
	for mask := 0; mask < 1<<4; mask++ { // the size and three pages: four items
		s := NewSim()
		f := mustOpen(t, s, "/a", O_RDWR|O_CREATE)
		mustSync(t, f)
		mustSyncDir(t, s)
		mustWrite(t, f, pattern(size, 0xab), 0)
		if n := s.Unsynced(); n != 4 {
			t.Fatalf("unsynced items = %d, want 4 (size + 3 pages)", n)
		}
		s.Crash(subset(mask))
		got := durable(t, s, "/a")
		if _, err := f.ReadAt(make([]byte, 1), 0); !errors.Is(err, ErrDead) {
			t.Fatalf("a handle from before the crash: %v, want ErrDead", err)
		}
		if mask&1 == 0 { // the size was lost: nothing is visible
			if len(got) != 0 {
				t.Fatalf("mask %b: size lost but durable length %d", mask, len(got))
			}
			outcomes["empty"] = true
			continue
		}
		if len(got) != size {
			t.Fatalf("mask %b: durable length %d, want %d", mask, len(got), size)
		}
		for p := 0; p < 3; p++ {
			lo, hi := p*pageSize, min((p+1)*pageSize, size)
			want := byte(0)
			if mask&(1<<(p+1)) != 0 {
				want = 0xab
			}
			for i := lo; i < hi; i++ {
				if got[i] != want {
					t.Fatalf("mask %b: page %d byte %d = %#x, want %#x", mask, p, i, got[i], want)
				}
			}
		}
		outcomes[string(got[0:1])+string(got[pageSize:pageSize+1])+string(got[2*pageSize:2*pageSize+1])] = true
	}
	if len(outcomes) != 9 { // empty, plus every combination of three pages kept or zero
		t.Fatalf("distinct outcomes = %d, want 9", len(outcomes))
	}
}

func TestFsyncOfAFileMakesItsDataDurableButNotItsDirectoryEntry(t *testing.T) {
	s := NewSim()
	f := mustOpen(t, s, "/a", O_RDWR|O_CREATE)
	mustWrite(t, f, []byte("hello"), 0)
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if n := s.Unsynced(); n != 1 {
		t.Fatalf("unsynced = %d, want 1 (the create)", n)
	}
	s.Crash(nil)
	if _, err := s.Durable("/a"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("after losing the create: %v, want not exist", err)
	}
	if _, err := s.OpenFile("/a", O_RDONLY, 0); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("open after the crash: %v, want not exist", err)
	}

	// The same, with the directory synced: the data is there.
	s = NewSim()
	f = mustOpen(t, s, "/a", O_RDWR|O_CREATE)
	mustWrite(t, f, []byte("hello"), 0)
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncDir("/"); err != nil {
		t.Fatal(err)
	}
	if n := s.Unsynced(); n != 0 {
		t.Fatalf("unsynced after both syncs = %d, want 0", n)
	}
	s.Crash(nil)
	if got := durable(t, s, "/a"); string(got) != "hello" {
		t.Fatalf("durable = %q, want hello", got)
	}

	// The create persisted but the data was never synced: the name exists and is empty.
	s = NewSim()
	f = mustOpen(t, s, "/a", O_RDWR|O_CREATE)
	mustWrite(t, f, []byte("hello"), 0)
	s.Crash(func(i, n int) bool { return i == 0 }) // items: [create, size, page 0]
	if got := durable(t, s, "/a"); len(got) != 0 {
		t.Fatalf("durable = %q, want an empty file", got)
	}
}

func TestARenameOverAnExistingNameYieldsTheOldOrTheNewContentUntilTheDirectoryIsSynced(t *testing.T) {
	outcomes := map[string]bool{}
	for mask := 0; mask < 1<<2; mask++ {
		s := NewSim()
		f := mustOpen(t, s, "/state", O_RDWR|O_CREATE)
		mustWrite(t, f, []byte("old"), 0)
		mustSync(t, f)
		mustSyncDir(t, s)
		tmp := mustOpen(t, s, "/.tmp-1", O_RDWR|O_CREATE)
		mustWrite(t, tmp, []byte("new"), 0)
		mustSync(t, tmp)
		if err := s.Rename("/.tmp-1", "/state"); err != nil {
			t.Fatal(err)
		}
		if n := s.Unsynced(); n != 2 {
			t.Fatalf("unsynced = %d, want 2 (create, rename)", n)
		}
		s.Crash(subset(mask))
		got := durable(t, s, "/state")
		outcomes[string(got)] = true
		if _, err := s.Durable("/.tmp-1"); mask == 1 && err != nil {
			t.Fatalf("create kept, rename lost: the temp file should remain: %v", err)
		}
	}
	if !outcomes["old"] || !outcomes["new"] || len(outcomes) != 2 {
		t.Fatalf("outcomes = %v, want exactly old and new", outcomes)
	}

	// Synced: only the new content.
	s := NewSim()
	f := mustOpen(t, s, "/state", O_RDWR|O_CREATE)
	mustWrite(t, f, []byte("old"), 0)
	tmp := mustOpen(t, s, "/.tmp-1", O_RDWR|O_CREATE)
	mustWrite(t, tmp, []byte("new"), 0)
	mustSync(t, f)
	mustSync(t, tmp)
	if err := s.Rename("/.tmp-1", "/state"); err != nil {
		t.Fatal(err)
	}
	mustSyncDir(t, s)
	s.Crash(nil)
	if got := durable(t, s, "/state"); string(got) != "new" {
		t.Fatalf("durable after synced rename = %q", got)
	}
}

func TestARenameThatPersistsBeforeTheDataIsTheHazardAtomicWritesAvoid(t *testing.T) {
	// Temp written, NOT synced, renamed over the target: the rename can persist while the
	// data does not, so the target reads back empty or zeros. This is why atomicfile fsyncs
	// the temp file before the rename.
	s := NewSim()
	f := mustOpen(t, s, "/state", O_RDWR|O_CREATE)
	mustWrite(t, f, []byte("old"), 0)
	mustSync(t, f)
	mustSyncDir(t, s)
	tmp := mustOpen(t, s, "/.tmp-1", O_RDWR|O_CREATE)
	mustWrite(t, tmp, []byte("new"), 0)
	if err := s.Rename("/.tmp-1", "/state"); err != nil {
		t.Fatal(err)
	}
	// items: create, rename, size, page 0. Keep the rename and the size, lose the page.
	s.Crash(func(i, n int) bool { return i == 1 || i == 2 })
	if got := durable(t, s, "/state"); string(got) != "\x00\x00\x00" {
		t.Fatalf("durable = %q, want three zero bytes", got)
	}
}

func TestAnUnlinkAndATruncateMayBeUndone(t *testing.T) {
	s := NewSim()
	f := mustOpen(t, s, "/a", O_RDWR|O_CREATE)
	mustWrite(t, f, []byte("hello"), 0)
	mustSync(t, f)
	mustSyncDir(t, s)
	if err := s.Remove("/a"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenFile("/a", O_RDONLY, 0); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("open after remove: %v", err)
	}
	s.Crash(nil) // the unlink is lost
	if got := durable(t, s, "/a"); string(got) != "hello" {
		t.Fatalf("after an undone unlink: %q", got)
	}

	f = mustOpen(t, s, "/a", O_RDWR)
	if err := f.Truncate(2); err != nil {
		t.Fatal(err)
	}
	if n := s.Unsynced(); n != 1 {
		t.Fatalf("unsynced = %d, want 1 (the size)", n)
	}
	s.Crash(nil)
	if got := durable(t, s, "/a"); string(got) != "hello" {
		t.Fatalf("after an undone truncate: %q", got)
	}
	f = mustOpen(t, s, "/a", O_RDWR)
	if err := f.Truncate(2); err != nil {
		t.Fatal(err)
	}
	s.Crash(func(int, int) bool { return true })
	if got := durable(t, s, "/a"); string(got) != "he" {
		t.Fatalf("after a persisted truncate: %q", got)
	}

	// Synced, the truncate holds and the unlink holds.
	f = mustOpen(t, s, "/a", O_RDWR)
	if err := f.Truncate(1); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove("/a"); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncDir("/"); err != nil {
		t.Fatal(err)
	}
	s.Crash(nil)
	if _, err := s.Durable("/a"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("after a synced unlink: %v", err)
	}
}

func TestAMkdirIsLostWithEverythingInItUntilTheParentIsSynced(t *testing.T) {
	s := NewSim()
	if err := s.Mkdir("/log", 0o700); err != nil {
		t.Fatal(err)
	}
	f := mustOpen(t, s, "/log/a", O_RDWR|O_CREATE)
	mustWrite(t, f, []byte("x"), 0)
	mustSync(t, f)
	if err := s.SyncDir("/log"); err != nil {
		t.Fatal(err)
	}
	s.Crash(nil)
	if _, err := s.Lstat("/log"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("directory whose mkdir was lost: %v", err)
	}
	if err := s.Mkdir("/log", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncDir("/"); err != nil {
		t.Fatal(err)
	}
	s.Crash(nil)
	names, err := s.ReadDir("/log")
	if err != nil || len(names) != 0 {
		t.Fatalf("synced empty directory: %v %v", names, err)
	}
}

func TestInjectedShortWritesAndErrorsSurfaceAsErrors(t *testing.T) {
	s := NewSim()
	f := mustOpen(t, s, "/a", O_RDWR|O_CREATE)
	s.Inject = func(op Op, name string) error {
		if op == OpWrite {
			return &ShortWrite{N: 3}
		}
		return nil
	}
	n, err := f.WriteAt([]byte("hello"), 0)
	if n != 3 || !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write: n=%d err=%v", n, err)
	}
	s.Inject = nil
	buf := make([]byte, 8)
	if n, err := f.ReadAt(buf, 0); n != 3 || err != io.EOF || string(buf[:3]) != "hel" {
		t.Fatalf("after the short write: %q %d %v", buf[:n], n, err)
	}

	s.Inject = func(op Op, name string) error {
		switch op {
		case OpWrite, OpCreate, OpRename, OpTruncate:
			return syscall.ENOSPC
		case OpSync, OpSyncDir:
			return syscall.EIO
		}
		return nil
	}
	if _, err := f.WriteAt([]byte("x"), 0); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("write: %v", err)
	}
	if _, err := s.OpenFile("/b", O_RDWR|O_CREATE, 0o600); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("create: %v", err)
	}
	if err := s.Rename("/a", "/c"); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("rename: %v", err)
	}
	if err := f.Truncate(0); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("truncate: %v", err)
	}
	if err := f.Sync(); !errors.Is(err, syscall.EIO) {
		t.Fatalf("sync: %v", err)
	}
	if err := s.SyncDir("/"); !errors.Is(err, syscall.EIO) {
		t.Fatalf("syncdir: %v", err)
	}
	// A failed operation changed nothing.
	s.Inject = nil
	if names, _ := s.ReadDir("/"); len(names) != 1 || names[0] != "a" {
		t.Fatalf("names after failed operations: %v", names)
	}
}

func TestInjectCanCrashBeforeAnOperationAndOpsCountsEveryOperation(t *testing.T) {
	s := NewSim()
	f := mustOpen(t, s, "/a", O_RDWR|O_CREATE)
	mustWrite(t, f, []byte("hello"), 0)
	mustSync(t, f)
	before := s.Ops()
	if before != 3 {
		t.Fatalf("ops = %d, want 3 (create, write, sync)", before)
	}
	s.CrashKeep = func(i, n int) bool { return true }
	s.Inject = func(op Op, name string) error {
		if op == OpSyncDir {
			return ErrCrash
		}
		return nil
	}
	if err := s.SyncDir("/"); !errors.Is(err, ErrCrash) {
		t.Fatalf("syncdir: %v, want ErrCrash", err)
	}
	if got := durable(t, s, "/a"); string(got) != "hello" {
		t.Fatalf("durable with everything kept: %q", got)
	}
	if _, err := f.ReadAt(make([]byte, 1), 0); !errors.Is(err, ErrDead) {
		t.Fatalf("old handle: %v", err)
	}
	if s.Ops() != before+1 {
		t.Fatalf("ops = %d, want %d", s.Ops(), before+1)
	}
}

func TestLocksAreExclusiveReleasedByCloseAndByACrash(t *testing.T) {
	s := NewSim()
	a := mustOpen(t, s, "/quill.lock", O_RDWR|O_CREATE)
	b := mustOpen(t, s, "/quill.lock", O_RDWR)
	if err := a.TryLock(); err != nil {
		t.Fatal(err)
	}
	if err := a.TryLock(); err != nil {
		t.Fatalf("re-locking by the holder: %v", err)
	}
	if err := b.TryLock(); !errors.Is(err, ErrLocked) {
		t.Fatalf("second holder: %v, want ErrLocked", err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if err := b.TryLock(); err != nil {
		t.Fatalf("after the holder closed: %v", err)
	}
	s.Crash(func(int, int) bool { return true })
	c := mustOpen(t, s, "/quill.lock", O_RDWR)
	if err := c.TryLock(); err != nil {
		t.Fatalf("after a crash: %v", err)
	}
}

func TestTheOSImplementationRefusesSymlinksAndItsLocksExclude(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	var fsys FS = OS{}
	if _, err := fsys.OpenFile(link, O_RDONLY, 0); err == nil {
		t.Fatal("opened through a symlink")
	}
	if _, err := fsys.OpenFile(link, O_RDWR|O_CREATE, 0o600); err == nil {
		t.Fatal("created through a symlink")
	}
	if err := fsys.SyncDir(link); err == nil {
		t.Fatal("synced a directory through a symlink")
	}
	a := mustOpen(t, fsys, filepath.Join(dir, "lock"), O_RDWR|O_CREATE)
	b := mustOpen(t, fsys, filepath.Join(dir, "lock"), O_RDWR)
	if err := a.TryLock(); err != nil {
		t.Fatal(err)
	}
	if err := b.TryLock(); !errors.Is(err, ErrLocked) {
		t.Fatalf("second open file description: %v, want ErrLocked", err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if err := b.TryLock(); err != nil {
		t.Fatalf("after close: %v", err)
	}
	if err := b.Chmod(0o640); err != nil {
		t.Fatal(err)
	}
	if info, _ := b.Stat(); info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v", info.Mode())
	}
}

// Both implementations agree on the ordinary semantics.
func TestConformance(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name string
		fsys FS
		root string
	}{{"os", OS{}, dir}, {"sim", NewSim(), "/"}} {
		t.Run(tc.name, func(t *testing.T) {
			fsys, p := tc.fsys, func(s string) string { return filepath.Join(tc.root, s) }
			if err := fsys.Mkdir(p("log"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := fsys.Mkdir(p("log"), 0o700); !errors.Is(err, fs.ErrExist) {
				t.Fatalf("mkdir twice: %v", err)
			}
			if _, err := fsys.OpenFile(p("log/x"), O_RDONLY, 0); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("open missing: %v", err)
			}
			f := mustOpen(t, fsys, p("log/x"), O_RDWR|O_CREATE|O_EXCL)
			if _, err := fsys.OpenFile(p("log/x"), O_RDWR|O_CREATE|O_EXCL, 0o600); !errors.Is(err, fs.ErrExist) {
				t.Fatalf("excl twice: %v", err)
			}
			mustWrite(t, f, []byte("hello world"), 0)
			mustWrite(t, f, []byte("W"), 6)
			buf := make([]byte, 20)
			n, err := f.ReadAt(buf, 0)
			if n != 11 || err != io.EOF || string(buf[:n]) != "hello World" {
				t.Fatalf("read: %q %d %v", buf[:n], n, err)
			}
			if _, err := f.ReadAt(buf[:1], 11); err != io.EOF {
				t.Fatalf("read at end: %v", err)
			}
			info, err := f.Stat()
			if err != nil || info.Size() != 11 || !info.Mode().IsRegular() {
				t.Fatalf("stat: %v %v", info, err)
			}
			if err := f.Truncate(5); err != nil {
				t.Fatal(err)
			}
			if err := f.Truncate(8); err != nil {
				t.Fatal(err)
			}
			n, _ = f.ReadAt(buf, 0)
			if string(buf[:n]) != "hello\x00\x00\x00" {
				t.Fatalf("after truncate down and up: %q", buf[:n])
			}
			if err := f.Sync(); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); !errors.Is(err, fs.ErrClosed) {
				t.Fatalf("double close: %v", err)
			}
			ro := mustOpen(t, fsys, p("log/x"), O_RDONLY)
			if _, err := ro.WriteAt([]byte("x"), 0); err == nil {
				t.Fatal("wrote to a read-only handle")
			}
			_ = ro.Close()
			if err := fsys.Rename(p("log/x"), p("log/y")); err != nil {
				t.Fatal(err)
			}
			if _, err := fsys.Lstat(p("log/x")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("lstat after rename: %v", err)
			}
			li, err := fsys.Lstat(p("log/y"))
			if err != nil || li.Size() != 8 || li.Name() != "y" {
				t.Fatalf("lstat y: %v %v", li, err)
			}
			if li, err := fsys.Lstat(p("log")); err != nil || !li.IsDir() {
				t.Fatalf("lstat dir: %v %v", li, err)
			}
			names, err := fsys.ReadDir(p("log"))
			if err != nil || len(names) != 1 || names[0] != "y" {
				t.Fatalf("readdir: %v %v", names, err)
			}
			if _, err := fsys.ReadDir(p("log/y")); err == nil {
				t.Fatal("readdir of a file")
			}
			if err := fsys.Remove(p("log")); err == nil {
				t.Fatal("removed a non-empty directory")
			}
			if err := fsys.SyncDir(p("log")); err != nil {
				t.Fatal(err)
			}
			if err := fsys.Remove(p("log/y")); err != nil {
				t.Fatal(err)
			}
			if err := fsys.Remove(p("log/y")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("remove twice: %v", err)
			}
		})
	}
}
