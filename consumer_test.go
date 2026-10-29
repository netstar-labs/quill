package quill

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// T8: offsets.

func TestCommitAndLagAgreeAndCommitRefusesOutsideTheRange(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 10)
	mustFlush(t, l)
	c, err := l.Consumer("billing", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Offset() != 1 || c.Protected() {
		t.Fatalf("new consumer at %d protected %v", c.Offset(), c.Protected())
	}
	if lag, _ := c.Lag(); lag != 10 {
		t.Fatalf("lag %d", lag)
	}
	recs, err := c.Read(4)
	if err != nil || checkContiguous(t, recs, 1) != 4 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	if err := c.Commit(recs[len(recs)-1].Seq + 1); err != nil {
		t.Fatal(err)
	}
	if c.Offset() != 5 {
		t.Fatalf("offset %d", c.Offset())
	}
	if lag, _ := c.Lag(); lag != 6 {
		t.Fatalf("lag %d", lag)
	}
	recs, _ = c.Read(100)
	if err := c.Commit(recs[len(recs)-1].Seq + 1); err != nil {
		t.Fatal(err)
	}
	if lag, _ := c.Lag(); lag != 0 || c.Offset() != 11 {
		t.Fatalf("caught up: lag %d offset %d", lag, c.Offset())
	}
	if recs, err := c.Read(10); err != nil || len(recs) != 0 {
		t.Fatalf("caught up read: %v %v", recs, err)
	}
	for _, bad := range []uint64{0, 12, 100} {
		if err := c.Commit(bad); !errors.Is(err, ErrOffsetRange) {
			t.Fatalf("commit %d: %v", bad, err)
		}
	}
	if err := c.Commit(3); err != nil { // a rewind
		t.Fatal(err)
	}
	recs, _ = c.Read(2)
	if checkContiguous(t, recs, 3) != 4 {
		t.Fatalf("%v", seqsOf(recs))
	}
	if string(fileBytes(t, s, "/log/consumers/billing")) != l.ID()+" 3 0\n" {
		t.Fatalf("file %q", fileBytes(t, s, "/log/consumers/billing"))
	}
}

func TestAConsumerAheadOfTheLogOrFromAnotherLogIsRefused(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	writeFile(t, s, "/log/consumers/ahead", encodeConsumer(consumerState{ID: logID(l.ID()), Next: 9}))
	if _, err := l.Consumer("ahead", ConsumerOptions{}); !errors.Is(err, ErrAhead) {
		t.Fatalf("ahead: %v", err)
	}
	writeFile(t, s, "/log/consumers/other", encodeConsumer(consumerState{ID: testID, Next: 1}))
	if _, err := l.Consumer("other", ConsumerOptions{}); !errors.Is(err, ErrWrongLog) {
		t.Fatalf("other log: %v", err)
	}
	writeFile(t, s, "/log/consumers/broken", []byte("garbage\n"))
	if _, err := l.Consumer("broken", ConsumerOptions{}); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("broken: %v", err)
	}
	// Head + 1 exactly is fine: a caught-up consumer.
	writeFile(t, s, "/log/consumers/caught", encodeConsumer(consumerState{ID: logID(l.ID()), Next: 4}))
	c, err := l.Consumer("caught", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	list, err := l.Consumers()
	if err != nil || len(list) != 4 {
		t.Fatalf("%+v %v", list, err)
	}
	bad := 0
	for _, c := range list {
		if c.Err != nil {
			bad++
		}
	}
	if bad != 2 { // other and broken
		t.Fatalf("%+v", list)
	}
}

func TestAFailedConsumerOpenReleasesItsLock(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	writeFile(t, s, "/log/consumers/other", encodeConsumer(consumerState{ID: testID, Next: 1}))
	if _, err := l.Consumer("other", ConsumerOptions{}); !errors.Is(err, ErrWrongLog) {
		t.Fatalf("first open: %v", err)
	}
	// A second attempt must reach the same verdict, not ErrLocked from a leaked lock.
	if _, err := l.Consumer("other", ConsumerOptions{}); !errors.Is(err, ErrWrongLog) {
		t.Fatalf("second open: %v", err)
	}
	// Once the file is repaired the name opens.
	writeFile(t, s, "/log/consumers/other", encodeConsumer(consumerState{ID: logID(l.ID()), Next: 1}))
	c, err := l.Consumer("other", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
}

// T9: lifecycle.

func TestConsumerLifecycleProtectionFlagsLocksAndNames(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.ProtectedConsumers = []string{"audit"} })
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)
	if string(fileBytes(t, s, "/log/consumers/audit")) != l.ID()+" 1 1\n" {
		t.Fatalf("protected consumer created at open: %q", fileBytes(t, s, "/log/consumers/audit"))
	}
	c, err := l.Consumer("billing", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	if !c.Protected() || c.Offset() != 1 {
		t.Fatalf("%v %d", c.Protected(), c.Offset())
	}
	if _, err := l.Consumer("billing", ConsumerOptions{}); !errors.Is(err, ErrLocked) {
		t.Fatalf("second handle: %v", err)
	}
	if err := c.Commit(2); err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	if _, err := s.Lstat("/log/consumers/billing.lock"); err != nil {
		t.Fatal("the lock file was deleted")
	}
	// Default options keep the stored flag; a commit never changes it.
	c, err = l.Consumer("billing", ConsumerOptions{})
	if err != nil || !c.Protected() || c.Offset() != 2 {
		t.Fatalf("%v %v %d", err, c.Protected(), c.Offset())
	}
	if err := c.Commit(3); err != nil || !c.Protected() {
		t.Fatal(err)
	}
	_ = c.Close()
	c, err = l.Consumer("billing", ConsumerOptions{Protect: ProtectOff})
	if err != nil || c.Protected() {
		t.Fatalf("%v %v", err, c.Protected())
	}
	_ = c.Close()
	if string(fileBytes(t, s, "/log/consumers/billing")) != l.ID()+" 3 0\n" {
		t.Fatalf("%q", fileBytes(t, s, "/log/consumers/billing"))
	}
	// Delete leaves the lock file; a held consumer cannot be deleted.
	c, _ = l.Consumer("billing", ConsumerOptions{})
	if err := l.DeleteConsumer("billing"); !errors.Is(err, ErrLocked) {
		t.Fatalf("delete while held: %v", err)
	}
	_ = c.Close()
	if err := l.DeleteConsumer("billing"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Lstat("/log/consumers/billing"); err == nil {
		t.Fatal("offset file remains")
	}
	if _, err := s.Lstat("/log/consumers/billing.lock"); err != nil {
		t.Fatal("lock file deleted")
	}
	if err := l.DeleteConsumer("billing"); err == nil {
		t.Fatal("deleting a missing consumer succeeded")
	}
	// Names.
	for _, bad := range []string{"Billing", "_archive", "a b", ""} {
		if _, err := l.Consumer(bad, ConsumerOptions{}); !errors.Is(err, ErrInvalidName) {
			t.Fatalf("%q: %v", bad, err)
		}
		if err := l.DeleteConsumer(bad); !errors.Is(err, ErrInvalidName) {
			t.Fatalf("delete %q: %v", bad, err)
		}
	}
	// The reserved name works through the internal path.
	if _, err := l.consumerStore().open("_archive", ConsumerOptions{Protect: ProtectOn}, false); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("reserved through the public path: %v", err)
	}
	a, err := l.consumerStore().open("_archive", ConsumerOptions{Protect: ProtectOn}, true)
	if err != nil || !a.Protected() {
		t.Fatalf("internal: %v", err)
	}
	_ = a.Close()
	// A consumer through a Reader in the same process shares the lock namespace.
	r := openReaderSim(t, s)
	defer r.Close()
	rc, err := r.Consumer("external", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Consumer("external", ConsumerOptions{}); !errors.Is(err, ErrLocked) {
		t.Fatalf("held by the reader's consumer: %v", err)
	}
	recs, err := rc.Read(10)
	if err != nil || checkContiguous(t, recs, 1) != 2 {
		t.Fatalf("%v %v", seqsOf(recs), err)
	}
	if err := rc.Commit(3); err != nil {
		t.Fatal(err)
	}
	if lag, _ := rc.Lag(); lag != 0 {
		t.Fatalf("lag %d", lag)
	}
	_ = rc.Close()
	if err := r.DeleteConsumer("external"); err != nil {
		t.Fatal(err)
	}
}

func TestTempFilesYoungerThanAnHourSurviveTheWritersOpenAndOlderOnesAreRemoved(t *testing.T) {
	s := fsx.NewSim()
	now := time.Unix(1_700_000_000, 0)
	s.Clock = func() time.Time { return now }
	l := openSim(t, s, nil)
	_ = l.Close()
	writeFile(t, s, "/log/.tmp-old", []byte("x"))
	writeFile(t, s, "/log/consumers/.tmp-old", []byte("x"))
	now = now.Add(2 * time.Hour)
	writeFile(t, s, "/log/.tmp-young", []byte("x"))
	writeFile(t, s, "/log/consumers/.tmp-young", []byte("x"))
	l = openSim(t, s, func(c *Config) { c.clock = func() time.Time { return now } })
	defer l.Close()
	for _, dir := range []string{"/log", "/log/consumers"} {
		names, _ := s.ReadDir(dir)
		joined := strings.Join(names, " ")
		if strings.Contains(joined, ".tmp-old") || !strings.Contains(joined, ".tmp-young") {
			t.Fatalf("%s holds %v", dir, names)
		}
	}
}

func TestConsumerFilesAreWrittenAtomicallyAndCommitsLeaveTheirMtime(t *testing.T) {
	// The premise of ConsumerTTL (decision 45): a commit and a creation each leave a file
	// whose mtime is the time of the call.
	dir := t.TempDir()
	l := openDir(t, dir, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	before := time.Now().Add(-time.Second)
	c, err := l.Consumer("billing", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	info, _ := os.Lstat(dir + "/consumers/billing")
	if info.ModTime().Before(before) {
		t.Fatalf("creation mtime %v before %v", info.ModTime(), before)
	}
	time.Sleep(20 * time.Millisecond)
	mid := time.Now()
	time.Sleep(20 * time.Millisecond)
	if err := c.Commit(2); err != nil {
		t.Fatal(err)
	}
	info, _ = os.Lstat(dir + "/consumers/billing")
	if info.ModTime().Before(mid) {
		t.Fatalf("commit mtime %v before %v", info.ModTime(), mid)
	}
	entries, _ := os.ReadDir(dir + "/consumers")
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Fatalf("temp file left: %s", e.Name())
		}
	}
}
