package quill

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// setUmask sets the process umask for one test and restores it after. Tests here are not
// parallel, so the process-wide change is not seen by another test.
func setUmask(t *testing.T, m int) {
	t.Helper()
	old := syscall.Umask(m)
	t.Cleanup(func() { syscall.Umask(old) })
}

// A3-5: quill.lock is the writer's alone, so it is 0600 whatever Config.Perm says. With a
// group or other read bit on it, a process that can only read could open it and take its flock,
// and no writer could Open until that process exited (decisions 41, 51). Consumer locks are
// not hardened: a group consumer must take its own, so they keep the log's mode through the
// umask.
func TestWriterLockIsOwnerOnlyWhateverPermSaysAndConsumerLocksKeepTheLogsMode(t *testing.T) {
	setUmask(t, 0o022)
	for _, tc := range []struct {
		perm     fs.FileMode
		consumer fs.FileMode // the consumer lock: Perm minus umask 022
	}{
		{0o660, 0o640},
		{0o644, 0o644},
		{0o600, 0o600},
	} {
		t.Run(tc.perm.String(), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "log")
			l := openDir(t, dir, func(c *Config) { c.Perm = tc.perm })
			defer l.Close()
			c, err := l.Consumer("group", ConsumerOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			for path, want := range map[string]fs.FileMode{
				filepath.Join(dir, lockName):                         0o600,
				filepath.Join(dir, consumersDir, "group"+lockSuffix): tc.consumer,
			} {
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if got := info.Mode().Perm(); got != want {
					t.Errorf("%s is %v, want %v", filepath.Base(path), got, want)
				}
			}
		})
	}
}

// On the simulator, which has no umask, the same holds exactly.
func TestWriterLockIsOwnerOnlyOnTheSimulatorAndConsumerLocksAreExact(t *testing.T) {
	s, _, l := retentionLog(t, func(c *Config) { c.Perm = 0o660; c.Retention = -1 })
	defer l.Close()
	c, err := l.Consumer("g", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if got := modeOf(t, s, "/log/"+lockName); got != 0o600 {
		t.Errorf("quill.lock is %v, want 0600", got)
	}
	if got := modeOf(t, s, "/log/consumers/g"+lockSuffix); got != 0o660 {
		t.Errorf("consumer lock is %v, want 0660", got)
	}
}

// A log written by an earlier build has a quill.lock at the log's mode. A writer Open tightens
// it to 0600 (it owns the file), whatever the umask.
func TestWriterOpenTightensAnExistingWiderLock(t *testing.T) {
	setUmask(t, 0o022)
	dir := filepath.Join(t.TempDir(), "log")
	l := openDir(t, dir, func(c *Config) { c.Perm = 0o660 })
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(dir, lockName)
	if err := os.Chmod(lock, 0o666); err != nil {
		t.Fatal(err)
	}
	l = openDir(t, dir, func(c *Config) { c.Perm = 0o660 })
	defer l.Close()
	info, err := os.Stat(lock)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("quill.lock is %v after Open, want 0600", got)
	}
}

// chmodFailFS makes Chmod fail on the writer's lock file (a lock owned by another account,
// say: EPERM), and counts the attempts.
type chmodFailFS struct {
	fsx.FS
	mu    sync.Mutex
	tries int
}

type chmodFailFile struct {
	fsx.File
	fs *chmodFailFS
}

func (f *chmodFailFS) OpenFile(name string, flag int, perm fs.FileMode) (fsx.File, error) {
	file, err := f.FS.OpenFile(name, flag, perm)
	if err != nil || !strings.HasSuffix(name, "/"+lockName) {
		return file, err
	}
	return &chmodFailFile{File: file, fs: f}, nil
}

func (f *chmodFailFile) Chmod(fs.FileMode) error {
	f.fs.mu.Lock()
	f.fs.tries++
	f.fs.mu.Unlock()
	return &fs.PathError{Op: "chmod", Path: lockName, Err: syscall.EPERM}
}

// A lock that cannot be tightened (not the owner) does not stop Open: the lock is held, which
// is what keeps the log single-writer. Open warns on Config.Logger instead.
func TestWriterOpenWithALockItCannotTightenStillOpensAndWarns(t *testing.T) {
	s := fsx.NewSim()
	// A lock an earlier build created wide.
	if err := s.Mkdir("/log", 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := s.OpenFile("/log/"+lockName, fsx.O_RDWR|fsx.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	var buf bytes.Buffer
	cfg := simConfig(s, "/log")
	fail := &chmodFailFS{FS: s}
	cfg.fs = fail
	cfg.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	l, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open with an untightenable lock: %v", err)
	}
	defer l.Close()
	if fail.tries == 0 {
		t.Fatal("Open never tried to tighten the lock")
	}
	if !strings.Contains(buf.String(), "quill.lock") {
		t.Fatalf("no warning about the lock mode: %q", buf.String())
	}
	// The lock is held: a second writer is refused.
	cfg2 := simConfig(s, "/log")
	if _, err := Open(cfg2); !errors.Is(err, ErrLocked) {
		t.Fatalf("second Open: %v, want ErrLocked", err)
	}
}

// openFailFS fails the read-only open of the destination's quill.lock with err, as a lock that
// is 0600 and owned by another account does (EACCES).
type openFailFS struct {
	fsx.FS
	err error
}

func (f *openFailFS) OpenFile(name string, flag int, perm fs.FileMode) (fsx.File, error) {
	if strings.HasSuffix(name, "/"+lockName) && flag&fsx.O_CREATE == 0 {
		return nil, &fs.PathError{Op: "open", Path: name, Err: f.err}
	}
	return f.FS.OpenFile(name, flag, perm)
}

// A3-5: a destination lock that cannot be opened (a 0600 lock owned by another account) means
// a writer may be live, so Archive refuses and writes nothing there.
func TestArchiveRefusesADestinationWhoseLockCannotBeOpened(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 2)
	// A destination with a lock file in it.
	if err := s.Mkdir("/arch", 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := s.OpenFile("/arch/"+lockName, fsx.O_RDWR|fsx.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	err = archive(&openFailFS{FS: s, err: syscall.EACCES}, "/log", "/arch")
	if err == nil {
		t.Fatal("Archive went ahead into a destination whose lock it cannot open")
	}
	if !errors.Is(err, ErrLocked) || !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("got %v, want an error that is ErrLocked and fs.ErrPermission", err)
	}
	if _, err := s.Lstat("/arch/" + formatName); err == nil {
		t.Fatal("Archive wrote quill.format into the refused destination")
	}
}

// The same on a real directory with a lock file at mode 0000 (the owner cannot open it either,
// so no second account is needed). Skipped as root, which opens anything.
func TestArchiveRefusesARealDestinationWhoseLockIsUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root opens a 0000 file")
	}
	src := filepath.Join(t.TempDir(), "src")
	l := openDir(t, src, func(c *Config) { c.SegmentBytes = 256 })
	defer l.Close()
	for i := 0; i < 3; i++ {
		appendN(t, l, 8)
		mustFlush(t, l)
	}
	if err := l.Roll(); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "dst")
	if err := os.Mkdir(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(dst, lockName)
	if err := os.WriteFile(lock, nil, 0o000); err != nil {
		t.Fatal(err)
	}
	err := Archive(src, dst)
	if !errors.Is(err, ErrLocked) || !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("got %v, want ErrLocked wrapping a permission error", err)
	}
	if _, err := os.Stat(filepath.Join(dst, formatName)); err == nil {
		t.Fatal("Archive wrote quill.format into the refused destination")
	}
}
