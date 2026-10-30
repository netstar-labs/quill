package quill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func protectedCfg(c *Config) { c.ProtectedConsumers = []string{"billing"} }

func reopenProtected(dir string) (*Log, error) {
	return Open(Config{Dir: dir, FlushInterval: time.Hour, SegmentBytes: 4096, Retention: -1,
		ProtectedConsumers: []string{"billing"}})
}

// A1-80: a writer redeployed while its protected consumer runs. Open must not need the
// consumer's lock when the file already says protected.
func TestOpenWithAProtectedConsumerRunningDoesNotNeedItsLock(t *testing.T) {
	dir := t.TempDir()
	l := openDir(t, dir, protectedCfg)
	if _, err := l.Append([]byte("a")); err != nil {
		t.Fatal(err)
	}
	r, err := OpenReader(dir, ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	c, err := r.Consumer("billing", ConsumerOptions{}) // the live consumer process: holds the lock
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	l2, err := reopenProtected(dir)
	if err != nil {
		t.Fatalf("reopen beside the live consumer: %v", err)
	}
	defer l2.Close()
	if d := time.Since(start); d > lockRetry/2 {
		t.Fatalf("reopen took %v: it waited on the consumer's lock", d)
	}
	if !c.Protected() {
		t.Fatal("the consumer lost its flag")
	}
}

// The writer still protects a consumer whose file says unprotected when it can have the
// lock; with a live holder that cannot be done safely (the holder would overwrite the flag
// at its next commit) and the error names the consumer's lock.
func TestOpenProtectsAnUnprotectedConsumerOnlyWithItsLock(t *testing.T) {
	dir := t.TempDir()
	l := openDir(t, dir, nil)
	r, err := OpenReader(dir, ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	c, err := r.Consumer("billing", ConsumerOptions{}) // unprotected, lock held
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = reopenProtected(dir)
	if !errors.Is(err, ErrLocked) || !strings.Contains(err.Error(), "billing.lock") {
		t.Fatalf("held and unprotected: %v", err)
	}
	_ = c.Close()
	l2, err := reopenProtected(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l2.Close()
	infos, err := l2.Consumers()
	if err != nil || len(infos) != 1 || !infos[0].Protected {
		t.Fatalf("%+v %v", infos, err)
	}
}

// The ErrAhead guard of decision 36 stays on the lock-free path: a consumer file above
// Head + 1 (a log restored from backup) still stops the writer and is left untouched.
func TestOpenWithAProtectedConsumerAheadOfTheHeadStillFails(t *testing.T) {
	dir := t.TempDir()
	l := openDir(t, dir, protectedCfg)
	for i := 0; i < 5; i++ {
		if _, err := l.Append([]byte("a")); err != nil {
			t.Fatal(err)
		}
	}
	id := l.ID()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"1", "0"} {
		path := filepath.Join(dir, "consumers", "billing")
		want := id + " 99 " + flag + "\n"
		if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := reopenProtected(dir); !errors.Is(err, ErrAhead) {
			t.Fatalf("flag %s: %v", flag, err)
		}
		if got, _ := os.ReadFile(path); string(got) != want {
			t.Fatalf("flag %s: the file was rewritten: %q", flag, got)
		}
	}
}
