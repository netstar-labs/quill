package quill

import (
	"crypto/rand"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// freeBytes is the only code that reads real free space, and the MinFreeBytes tests all
// inject Config.statfs (Stage 2 audit L2-9). TestFreeBytesReadsTheRealFilesystem only checks
// that it returns something and fails on a missing directory; these check the number.

// A made-up Statfs_t pins the unit (blocks times block size) and the field (Bavail, which
// excludes the root reserve, not Bfree) on every filesystem. A real one cannot pin the
// field: where it has no reserve, Bfree and Bavail are equal (a tmpfs, a macOS APFS volume).
func TestAvailBytesIsAvailableBlocksTimesBlockSize(t *testing.T) {
	var st syscall.Statfs_t
	st.Bsize = 4096
	st.Bavail = 10
	st.Bfree = 1000
	st.Blocks = 100000
	if got, want := availBytes(&st), uint64(40960); got != want {
		t.Fatalf("availBytes = %d, want %d (Bavail 10 x Bsize 4096; Bfree is the root's view)", got, want)
	}
}

// On the real filesystem the wiring from Statfs and the unit are checked by the effect of a
// write: 8 MiB of random (incompressible) data, fsynced, takes about 8 MiB. A missing Bsize
// factor would read a few kilobytes, a wrong factor far off. The bounds are loose because
// other processes use the same disk.
func TestFreeBytesTracksRealWrites(t *testing.T) {
	d := t.TempDir()
	before, err := freeBytes(d)
	if err != nil {
		t.Fatal(err)
	}
	const size = 8 << 20
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(d, "ballast"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(buf); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	after, err := freeBytes(d)
	if err != nil {
		t.Fatal(err)
	}
	if after >= before || before-after < 4<<20 || before-after > 64<<20 {
		t.Fatalf("free space went from %d to %d after writing %d bytes: want a fall of 4 MiB to 64 MiB", before, after, size)
	}
}

// Open with the real statfs: a floor of one byte is met, and a floor no disk can meet sets
// DiskFull once a prune pass has nothing left to delete.
func TestMinFreeBytesWithTheRealStatfs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		floor    int64
		wantFull bool
	}{
		{"floor of one byte", 1, false},
		{"floor above any disk", math.MaxInt64, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l, err := Open(Config{Dir: filepath.Join(t.TempDir(), "log"), MinFreeBytes: tc.floor, NoMaintenance: true})
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			if _, err := l.Append([]byte("x")); err != nil {
				t.Fatal(err)
			}
			if err := l.Prune(false); err != nil {
				t.Fatal(err)
			}
			st, err := l.Stat()
			if err != nil {
				t.Fatal(err)
			}
			if st.DiskFull != tc.wantFull {
				t.Fatalf("DiskFull = %v, want %v (FreeBytesAtLastPass %d)", st.DiskFull, tc.wantFull, st.FreeBytesAtLastPass)
			}
			if st.FreeBytesAtLastPass == 0 {
				t.Fatal("FreeBytesAtLastPass is 0: the real statfs was not read")
			}
		})
	}
}
