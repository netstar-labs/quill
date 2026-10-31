package quill

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A3-1: a FIFO planted where quill expects a state file must be refused as damage, not block
// the open(2) that was meant to find that out (decisions 51, 74).

// within runs fn and fails the test if it has not returned in a few seconds. A blocked fn is
// released by opening the FIFO's other end, so the test does not leak a goroutine.
func within(t *testing.T, what, fifo string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); fn() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		if f, err := os.OpenFile(fifo, os.O_RDWR, 0); err == nil {
			defer f.Close()
			<-done
		}
		t.Fatalf("%s blocked on the FIFO %s", what, fifo)
	}
}

func plantFIFO(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
}

// makeLog creates a log with a record in it and closes it, so every state file exists.
func makeLog(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "log")
	l := openDir(t, dir, nil)
	appendN(t, l, 3)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAFIFOInPlaceOfAStateFileIsRefusedNotWaitedOn(t *testing.T) {
	for _, name := range []string{tailName, formatName} {
		t.Run(name, func(t *testing.T) {
			dir := makeLog(t)
			fifo := filepath.Join(dir, name)
			plantFIFO(t, fifo)
			within(t, "OpenReader", fifo, func() {
				if r, err := OpenReader(dir, ReaderOptions{}); err == nil {
					r.Close()
					t.Error("OpenReader of a log whose " + name + " is a FIFO succeeded")
				}
			})
			within(t, "Open", fifo, func() {
				cfg := Config{Dir: dir, FlushInterval: time.Hour, Retention: -1}
				if l, err := Open(cfg); err == nil {
					l.Close()
					t.Error("Open of a log whose " + name + " is a FIFO succeeded")
				}
			})
		})
	}
	t.Run(policyName, func(t *testing.T) {
		dir := makeLog(t)
		fifo := filepath.Join(dir, policyName)
		plantFIFO(t, fifo)
		within(t, "ReadPolicy", fifo, func() {
			if _, ok := ReadPolicy(dir); ok {
				t.Error("ReadPolicy accepted a FIFO")
			}
		})
	})
}

func TestAFIFOInConsumersIsListedAsDamageAndDoesNotStopRotation(t *testing.T) {
	dir := makeLog(t)
	fifo := filepath.Join(dir, consumersDir, "x")
	plantFIFO(t, fifo)
	r, err := OpenReader(dir, ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	within(t, "Reader.Consumers", fifo, func() {
		list, err := r.Consumers()
		if err != nil || len(list) != 1 || list[0].Name != "x" || list[0].Err == nil {
			t.Errorf("Consumers = %+v, %v; want one entry x with Err set", list, err)
		}
	})

	// The seal lists the consumers under the log's mutex: a blocked listing froze the log.
	l := openDir(t, dir, func(c *Config) { c.SegmentBytes = 1024 })
	defer l.Close()
	within(t, "append across rotations", fifo, func() {
		for i := 0; i < 40; i++ {
			if _, err := l.Append(payloadFor(uint64(i))); err != nil {
				t.Error(err)
				return
			}
			if err := l.Flush(); err != nil {
				t.Error(err)
				return
			}
		}
	})
}

// A lock file is opened read-write, which never waits on a FIFO; this keeps it that way.
func TestAFIFOInPlaceOfALockFileDoesNotBlockOpenOrArchive(t *testing.T) {
	dir := makeLog(t)
	fifo := filepath.Join(dir, lockName)
	plantFIFO(t, fifo)
	within(t, "Open", fifo, func() {
		cfg := Config{Dir: dir, FlushInterval: time.Hour, Retention: -1}
		if l, err := Open(cfg); err == nil {
			l.Close()
		}
	})
	dst := filepath.Join(t.TempDir(), "dst")
	if err := os.Mkdir(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	fifo = filepath.Join(dst, lockName)
	plantFIFO(t, fifo)
	within(t, "Archive", fifo, func() { _ = Archive(makeLog(t), dst) })
}
