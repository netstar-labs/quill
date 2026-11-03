package quill

import (
	"strings"
	"testing"
	"time"
)

// The MinFreeBytes floor overrides consumer protection (decision 25, Stage 2 audit A3-32): a
// protected consumer holds the sealed segments back from the age sweep, and when the free space
// is below the floor the writer deletes the oldest sealed segments past it, so the consumer
// loses records and the Warn naming it says so.
func TestMinFreeBytesDeletesSegmentsAProtectedConsumerHasNotRead(t *testing.T) {
	sf := &statfsSeam{free: 100_000}
	var lc logCapture
	s, clk, l := retentionLog(t, func(c *Config) {
		c.MinFreeBytes = 5000
		c.Logger = lc.logger()
		c.statfs = sf.statfs
	})
	defer l.Close()
	fillSegments(t, l, clk, 4)
	slow, err := l.Consumer("slow", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	defer slow.Close()
	if slow.Offset() != 1 {
		t.Fatalf("the protected consumer starts at %d, want 1", slow.Offset())
	}

	// Room above the floor and every record past Retention: the consumer's protection holds them.
	clk.Advance(2 * time.Hour)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := segmentFiles(t, s, "/log"); len(ids) != 5 {
		t.Fatalf("with room, the protected consumer at 1 should keep every segment: %v", ids)
	}
	if st, _ := l.Stat(); st.Dropped != 0 {
		t.Fatalf("dropped %d with room above the floor", st.Dropped)
	}

	// Below the floor: the oldest sealed segments go although the consumer has read none of them.
	sf.free = 100
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	st, _ := l.Stat()
	if st.First <= 1 || st.Dropped == 0 {
		t.Fatalf("MinFreeBytes deleted nothing under a protected consumer: %+v", st)
	}
	if slow.Offset() >= st.First {
		t.Fatalf("consumer at %d, first retained %d: it has not lost records", slow.Offset(), st.First)
	}
	found := false
	lc.mu.Lock()
	for _, m := range lc.msgs {
		found = found || (strings.Contains(m, "level=WARN") && strings.Contains(m, "lost records") && strings.Contains(m, "consumer=slow "))
	}
	lc.mu.Unlock()
	if !found {
		t.Fatalf("no Warn names the consumer that lost records: %q", lc.msgs)
	}
}
