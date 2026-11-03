package quill

import (
	"slices"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// The time thresholds are compared in whole seconds at equality, and the code counts an age
// equal to its window as expired: "older than" in the documents means "at least as old as".
// Each test below steps the injected clock to one second short of the window and then onto it,
// so a comparison that drifts by a second in either direction fails.

// staleAfter runs a prune pass and returns the names Stat reports as stale consumers.
func staleAfter(t *testing.T, l *Log) []string {
	t.Helper()
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	st, err := l.Stat()
	if err != nil {
		t.Fatal(err)
	}
	return st.StaleConsumers
}

func protectedAt(t *testing.T, l *Log, name string, offset uint64) {
	t.Helper()
	c, err := l.Consumer(name, ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Commit(offset); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func ttlLog(t *testing.T, ttl time.Duration) (*fsx.Sim, *testClock, *Log) {
	t.Helper()
	return retentionLog(t, func(c *Config) { c.Retention = -1; c.ConsumerTTL = ttl })
}

// The record at the consumer's offset must be at least TTL old, whatever its file's age.
func TestAConsumerIsStaleWhenTheRecordAtItsOffsetIsExactlyTheTTLOld(t *testing.T) {
	const ttl = 30 * time.Minute
	_, clk, l := ttlLog(t, ttl)
	defer l.Close()
	protectedAt(t, l, "slow", 1) // its file is written now, and goes ten minutes older than the records
	clk.Advance(10 * time.Minute)
	appendN(t, l, 4)
	mustFlush(t, l)
	clk.Advance(ttl - time.Second)
	if got := staleAfter(t, l); len(got) != 0 {
		t.Fatalf("a record TTL-1s old: stale %v", got)
	}
	clk.Advance(time.Second)
	if got := staleAfter(t, l); !slices.Equal(got, []string{"slow"}) {
		t.Fatalf("a record exactly TTL old: stale %v", got)
	}
}

// The offset file's age must be at least TTL, whatever the record's age.
func TestAConsumerIsStaleWhenItsFileIsExactlyTheTTLOld(t *testing.T) {
	const ttl = 30 * time.Minute
	_, clk, l := ttlLog(t, ttl)
	defer l.Close()
	appendN(t, l, 4)
	mustFlush(t, l)
	clk.Advance(2 * ttl) // the records are far older than the TTL
	protectedAt(t, l, "slow", 1)
	clk.Advance(ttl - time.Second)
	if got := staleAfter(t, l); len(got) != 0 {
		t.Fatalf("a file TTL-1s old: stale %v", got)
	}
	clk.Advance(time.Second)
	if got := staleAfter(t, l); !slices.Equal(got, []string{"slow"}) {
		t.Fatalf("a file exactly TTL old: stale %v", got)
	}
}

// A consumer at offset Head still has the record at Head to read, so it is behind (decision
// 45: offset <= Head); one at Head+1 has read everything and is never stale.
func TestAConsumerAtTheHeadIsBehindAndOneAfterItIsCaughtUp(t *testing.T) {
	const ttl = 30 * time.Minute
	_, clk, l := ttlLog(t, ttl)
	defer l.Close()
	appendN(t, l, 4)
	mustFlush(t, l)
	protectedAt(t, l, "athead", l.Head())
	protectedAt(t, l, "caughtup", l.Head()+1)
	clk.Advance(2 * ttl)
	if got := staleAfter(t, l); !slices.Equal(got, []string{"athead"}) {
		t.Fatalf("stale %v, want only the consumer at the head", got)
	}
}

// A quiet active segment is sealed when its newest record is at least SegmentMaxAge old.
func TestAgeRotationActsExactlyAtSegmentMaxAge(t *testing.T) {
	const maxAge = 10 * time.Minute
	s, clk, l := retentionLog(t, func(c *Config) {
		c.SegmentBytes = 1 << 20 // size never rotates
		c.Retention = -1
		c.SegmentMaxAge = maxAge
	})
	defer l.Close()
	appendN(t, l, 2)
	mustFlush(t, l)
	clk.Advance(maxAge - time.Second)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 {
		t.Fatalf("sealed when the newest record was SegmentMaxAge-1s old: %v", ids)
	}
	clk.Advance(time.Second)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("not sealed when the newest record was exactly SegmentMaxAge old: %v", ids)
	}
}

// A sealed segment goes when its newest record is at least Retention old.
func TestTheAgeSweepActsExactlyAtRetention(t *testing.T) {
	const retention = time.Hour
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = retention })
	defer l.Close()
	fillSegments(t, l, clk, 1) // segment 0 sealed with its newest record one minute ago
	clk.Advance(retention - time.Minute - time.Second)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 2 {
		t.Fatalf("deleted when its newest record was Retention-1s old: %v", ids)
	}
	clk.Advance(time.Second)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("kept when its newest record was exactly Retention old: %v", ids)
	}
}
