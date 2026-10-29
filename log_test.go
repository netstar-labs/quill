package quill

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

func context_background() context.Context { return context.Background() }

// T11: creation and the marker.

func TestAnEmptyDirectoryIsANewLogAndOneWithOtherFilesIsRefused(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	names, _ := s.ReadDir("/log")
	if got := strings.Join(names, " "); got != "0000000000.log consumers quill.format quill.lock quill.policy quill.tail" {
		t.Fatalf("a new log holds %q", got)
	}
	if l.Head() != 0 || readTail(t, s, "/log") != (tail{}) {
		t.Fatalf("head %d tail %+v", l.Head(), readTail(t, s, "/log"))
	}
	id := l.ID()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if l2 := openSim(t, s, nil); l2.ID() != id {
		t.Fatalf("reopened with id %s, want %s", l2.ID(), id)
	} else {
		_ = l2.Close()
	}
	// A directory holding something that is not a log.
	if err := s.Mkdir("/other", 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, s, "/other/notes.txt", []byte("x"))
	if _, err := Open(simConfig(s, "/other")); !errors.Is(err, ErrNoLog) {
		t.Fatalf("other files: %v", err)
	}
	// Only the lock and temp files do not count.
	if err := s.Mkdir("/fresh", 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, s, "/fresh/quill.lock", nil)
	writeFile(t, s, "/fresh/.tmp-abc", []byte("x"))
	l3, err := Open(simConfig(s, "/fresh"))
	if err != nil {
		t.Fatal(err)
	}
	_ = l3.Close()
	// Missing parents are created.
	l4, err := Open(simConfig(s, "/a/b/c/log"))
	if err != nil {
		t.Fatal(err)
	}
	_ = l4.Close()
	if _, err := Open(simConfig(s, "/other/notes.txt")); err == nil {
		t.Fatal("opened a file as a directory")
	}
}

func TestANewerFormatMarkerAndSegmentsWithoutOneAreRefused(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 3)
	mustFlush(t, l)
	_ = l.Close()
	writeFile(t, s, "/log/quill.format", []byte("quill/2 0123456789abcdef0123456789abcdef\n"))
	if _, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrFutureVersion) {
		t.Fatalf("future: %v", err)
	}
	if err := s.Remove("/log/quill.format"); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrNoLog) {
		t.Fatalf("no marker: %v", err)
	}
	writeFile(t, s, "/log/quill.format", []byte("garbage\n"))
	if _, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("damaged marker: %v", err)
	}
}

func TestACrashAtEveryStepOfCreationLeavesADirectoryOpenCompletesOrRefusesNeverAnotherLog(t *testing.T) {
	for at := 1; ; at++ {
		s := fsx.NewSim()
		count := 0
		s.CrashKeep = func(i, n int) bool { return (i*7+at)%3 != 0 } // a varied subset
		s.Inject = func(op fsx.Op, name string) error {
			count++
			if count == at {
				return fsx.ErrCrash
			}
			return nil
		}
		l, err := Open(simConfig(s, "/log"))
		if err == nil {
			_ = l.Close()
			if count < at {
				break // the whole creation ran without a crash: every step has been tried
			}
		}
		s.Inject = nil
		// What a reader could have seen: the durable quill.format, if quill.tail is durable too.
		var seenID string
		if _, err := s.Durable("/log/" + tailName); err == nil {
			if data, err := s.Durable("/log/" + formatName); err == nil {
				if id, err := decodeFormat(data); err == nil {
					seenID = string(id)
				}
			}
		}
		l2, err := Open(simConfig(s, "/log"))
		if err != nil {
			if !errors.Is(err, ErrNoLog) && !errors.Is(err, ErrCorrupt) {
				t.Fatalf("crash at op %d: reopen: %v", at, err)
			}
			continue
		}
		if seenID != "" && l2.ID() != seenID {
			t.Fatalf("crash at op %d: reopened as %s, a reader saw %s", at, l2.ID(), seenID)
		}
		if _, err := l2.Append([]byte("x")); err != nil {
			t.Fatalf("crash at op %d: append after reopen: %v", at, err)
		}
		if err := l2.Close(); err != nil {
			t.Fatalf("crash at op %d: close: %v", at, err)
		}
	}
}

func TestTwoProcessesCreatingTheSameLogAtOnceEndWithOneLogID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "log")
	ids := make(chan string, 8)
	for i := 0; i < 8; i++ {
		go func() {
			for {
				l, err := Open(Config{Dir: dir, Retention: -1})
				if errors.Is(err, ErrLocked) {
					continue
				}
				if err != nil {
					ids <- "error: " + err.Error()
					return
				}
				ids <- l.ID()
				_ = l.Close()
				return
			}
		}()
	}
	first := <-ids
	for i := 1; i < 8; i++ {
		if id := <-ids; id != first {
			t.Fatalf("ids differ: %s and %s", first, id)
		}
	}
}

// T22: config.

func TestEveryConfigDefaultAndRefusal(t *testing.T) {
	c := Config{Dir: "x"}.withDefaults()
	want := Config{Dir: "x", Retention: 24 * time.Hour, PruneInterval: time.Minute, FlushBytes: 64 << 10, FlushInterval: 250 * time.Millisecond,
		MaxBufferBytes: 1 << 20, SegmentBytes: 8 << 20, SegmentMaxAge: 24 * time.Hour, MaxPayloadBytes: 64<<20 - 32,
		MaxBatchBytes: 16 << 20, MaxReadBytes: 16 << 20, Perm: 0o600}
	c.fs, c.clock = nil, nil
	if c.Dir != want.Dir || c.Retention != want.Retention || c.PruneInterval != want.PruneInterval || c.FlushBytes != want.FlushBytes ||
		c.FlushInterval != want.FlushInterval || c.MaxBufferBytes != want.MaxBufferBytes || c.SegmentBytes != want.SegmentBytes ||
		c.SegmentMaxAge != want.SegmentMaxAge || c.MaxPayloadBytes != want.MaxPayloadBytes || c.MaxBatchBytes != want.MaxBatchBytes ||
		c.MaxReadBytes != want.MaxReadBytes || c.Perm != want.Perm || c.MaxBytes != 0 || c.MinFreeBytes != 0 || c.MaxRecords != 0 || c.ConsumerTTL != 0 {
		t.Fatalf("defaults %+v", c)
	}
	if d := (Config{Dir: "x", Retention: -1}).withDefaults(); d.PruneInterval != time.Minute || d.SegmentMaxAge != -1 {
		t.Fatalf("negative retention: prune %v age %v", d.PruneInterval, d.SegmentMaxAge)
	}
	if d := (Config{Dir: "x", Retention: 10 * time.Second}).withDefaults(); d.PruneInterval != 10*time.Second {
		t.Fatalf("short retention: prune %v", d.PruneInterval)
	}
	if d := (Config{Dir: "x", Retention: 100 * time.Millisecond}).withDefaults(); d.PruneInterval != time.Second {
		t.Fatalf("tiny retention: prune %v", d.PruneInterval)
	}
	if d := (Config{Dir: "x", Perm: 0o640}).withDefaults(); d.dirPerm() != 0o750 {
		t.Fatalf("dir perm %v", d.dirPerm())
	}
	refused := map[string]Config{
		"no dir":               {},
		"segment too large":    {Dir: "x", SegmentBytes: 1<<30 + 1},
		"payload over ceiling": {Dir: "x", MaxPayloadBytes: 64 << 20},
		"maxbytes too small":   {Dir: "x", MaxBytes: 16<<20 - 1},
		"negative flushbytes":  {Dir: "x", FlushBytes: -1},
		"negative maxbytes":    {Dir: "x", MaxBytes: -1},
		"negative records":     {Dir: "x", MaxRecords: -1},
		"negative ttl":         {Dir: "x", ConsumerTTL: -1},
		"negative readbytes":   {Dir: "x", MaxReadBytes: -1},
		"prune too short":      {Dir: "x", PruneInterval: time.Millisecond},
		"perm no owner write":  {Dir: "x", Perm: 0o400},
		"perm with setuid":     {Dir: "x", Perm: 0o600 | fs.ModeSetuid},
		"bad protected name":   {Dir: "x", ProtectedConsumers: []string{"Bad"}},
		"reserved name":        {Dir: "x", ProtectedConsumers: []string{"_archive"}},
	}
	for name, cfg := range refused {
		if err := cfg.withDefaults().validate(); !errors.Is(err, ErrConfig) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := Open(Config{Dir: t.TempDir(), MaxBytes: 1}); !errors.Is(err, ErrConfig) {
		t.Fatalf("open with bad config: %v", err)
	}
	if entries, _ := os.ReadDir(t.TempDir()); len(entries) != 0 {
		t.Fatal("a refused config touched the directory")
	}
	ok := Config{Dir: "x", MaxBytes: 16 << 20, Perm: 0o640, ProtectedConsumers: []string{"billing", "a-b_c9"}}
	if err := ok.withDefaults().validate(); err != nil {
		t.Fatal(err)
	}
}

func TestConsumerNamesAreValidated(t *testing.T) {
	for _, good := range []string{"a", "billing", "a-b_c9", strings.Repeat("z", 200)} {
		if err := validName(good); err != nil {
			t.Errorf("%q: %v", good, err)
		}
	}
	for _, bad := range []string{"", "A", "a b", "a/b", "_archive", strings.Repeat("z", 201), "é", "a.b"} {
		if err := validName(bad); !errors.Is(err, ErrInvalidName) {
			t.Errorf("%q accepted", bad)
		}
	}
}

// T23: locks (library).

func TestASecondWritableOpenFailsFastWithErrLocked(t *testing.T) {
	dir := t.TempDir()
	l := openDir(t, dir, nil)
	defer l.Close()
	start := time.Now()
	_, err := Open(Config{Dir: dir, Retention: -1})
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("second open: %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("second open took %v", time.Since(start))
	}
	// Within one process, the simulator agrees.
	s := fsx.NewSim()
	a := openSim(t, s, nil)
	if _, err := Open(simConfig(s, "/log")); !errors.Is(err, ErrLocked) {
		t.Fatalf("sim second open: %v", err)
	}
	_ = a.Close()
	b := openSim(t, s, nil)
	_ = b.Close()
}

func TestTheLockIsReleasedWhenTheHolderIsKilled(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "log")
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "QUILL_HELPER=holder", "QUILL_HELPER_DIR="+dir)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 16)
	n, _ := out.Read(buf)
	if string(buf[:n]) != "locked\n" {
		t.Fatalf("helper said %q", buf[:n])
	}
	if _, err := Open(Config{Dir: dir, Retention: -1}); !errors.Is(err, ErrLocked) {
		t.Fatalf("while held by another process: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	l, err := Open(Config{Dir: dir, Retention: -1})
	if err != nil {
		t.Fatalf("after the holder was killed: %v", err)
	}
	_ = l.Close()
}
