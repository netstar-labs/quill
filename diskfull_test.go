package quill

import (
	"errors"
	"strings"
	"syscall"
	"testing"
)

// MinFreeBytes with a failing or full statfs is visible (Stage 2 audit A5-22): Err() reports
// both states, Stat() shows DiskFull and the free bytes of the last pass, and recovery is
// logged and clears them.

// statfsSeam is an injectable free-space function: failing makes it return statErr.
type statfsSeam struct {
	failing bool
	free    uint64
	err     error
}

func (f *statfsSeam) statfs(string) (uint64, error) {
	if f.failing {
		return 0, f.err
	}
	return f.free, nil
}

func TestFailingStatfsIsReportedByErrAndClearsWhenItRecovers(t *testing.T) {
	sf := &statfsSeam{free: 100_000, err: errors.New("statfs: input/output error")}
	_, clk, l := retentionLog(t, func(c *Config) {
		c.MinFreeBytes = 5000
		c.statfs = sf.statfs
	})
	defer l.Close()
	fillSegments(t, l, clk, 3)
	wantErrNil(t, l, "before statfs failed")

	sf.failing = true
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if err := l.Err(); !errors.Is(err, sf.err) {
		t.Fatalf("Err() with statfs failing: %v, want it to wrap the statfs error", err)
	}
	// One failure is the trigger: there is no count of consecutive failures.
	sf.failing = false
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	wantErrNil(t, l, "after statfs recovered")
}

func TestDiskFullIsInErrAndStatAndItsClearingIsLogged(t *testing.T) {
	sf := &statfsSeam{free: 100_000, err: errors.New("statfs: input/output error")}
	var lc logCapture
	_, clk, l := retentionLog(t, func(c *Config) {
		c.MinFreeBytes = 5000
		c.Logger = lc.logger()
		c.statfs = sf.statfs
	})
	defer l.Close()
	fillSegments(t, l, clk, 3)
	if st, _ := l.Stat(); st.DiskFull {
		t.Fatalf("DiskFull before the floor was breached: %+v", st)
	}

	sf.free = 100 // hopeless: everything sealed goes and appends fail
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Append([]byte("x")); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("append on a full disk: %v", err)
	}
	if err := l.Err(); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Err() with diskFull set: %v, want ENOSPC", err)
	}
	st, _ := l.Stat()
	if !st.DiskFull || st.FreeBytesAtLastPass != 100 {
		t.Fatalf("Stat with diskFull set: DiskFull %v FreeBytesAtLastPass %d", st.DiskFull, st.FreeBytesAtLastPass)
	}

	// statfs failing while full: the state is held and both reasons are reported.
	sf.failing = true
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if err := l.Err(); !errors.Is(err, sf.err) || !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Err() with statfs failing while full: %v", err)
	}
	if st, _ = l.Stat(); !st.DiskFull {
		t.Fatalf("DiskFull dropped while statfs failed: %+v", st)
	}

	// statfs works again and shows room: nothing stays frozen.
	sf.failing = false
	sf.free = 100_000
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	wantErrNil(t, l, "after space returned")
	if st, _ = l.Stat(); st.DiskFull || st.FreeBytesAtLastPass != 100_000 {
		t.Fatalf("Stat after space returned: DiskFull %v FreeBytesAtLastPass %d", st.DiskFull, st.FreeBytesAtLastPass)
	}
	if _, err := l.Append([]byte("x")); err != nil {
		t.Fatalf("append after space returned: %v", err)
	}
	if !lc.has("free space is back above MinFreeBytes") {
		t.Fatalf("clearing was not logged: %v", lc.msgs)
	}
	if n := strings.Count(strings.Join(lc.msgs, "\n"), "free space below MinFreeBytes"); n != 1 {
		t.Fatalf("the transition Warn should fire once, fired %d: %v", n, lc.msgs)
	}
}

// freeBytes is the real statfs reader that Open wires in; every other test injects a seam.
func TestFreeBytesReadsTheRealFilesystem(t *testing.T) {
	free, err := freeBytes(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if free == 0 {
		t.Fatal("freeBytes reports no free space on the test temp directory")
	}
	if _, err := freeBytes(t.TempDir() + "/missing"); err == nil {
		t.Fatal("freeBytes on a missing directory returned no error")
	}
}
