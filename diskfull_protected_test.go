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

// The MinFreeBytes estimate counts every segment the pass deletes (Stage 2 audit A7-1): the
// sealed segments the age sweep already chose free their size too, so a reading just below the
// floor that those segments make up for deletes nothing more, and a larger shortfall deletes
// only as many further segments as it still needs.
func TestMinFreeBytesCreditsTheSegmentsTheAgeSweepAlreadyDeletes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		shortfall func(expired uint64) uint64 // the floor minus the statfs reading, from the bytes the expired two free
		deleted   int                         // sealed segments gone, of five; the oldest two are expired
	}{
		{"one byte short", func(uint64) uint64 { return 1 }, 2},
		{"exactly what the expired two free", func(expired uint64) uint64 { return expired }, 2},
		{"one byte more than the expired two free", func(expired uint64) uint64 { return expired + 1 }, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sf := &statfsSeam{}
			s, clk, l := retentionLog(t, func(c *Config) {
				c.statfs = sf.statfs
			})
			defer l.Close()
			fillSegments(t, l, clk, 5)
			clk.Advance(time.Hour - 3*time.Minute - 30*time.Second) // now minute 61.5: minutes 0 and 1 are at least an hour old, minute 2 is not
			// The floor is set after the fill so that no seal ran it: one pass, one reading.
			l.mu.Lock()
			expired := uint64(l.sealed[0].size + l.sealed[1].size)
			floor := 10 * expired
			l.cfg.MinFreeBytes = int64(floor)
			l.mu.Unlock()
			sf.free = floor - tc.shortfall(expired)
			if err := l.Prune(true); err != nil {
				t.Fatal(err)
			}
			// Sealed segments have ids 0..4 and the active one 5: the first id left is the count deleted.
			ids := segmentFiles(t, s, "/log")
			if got := int(ids[0]); got != tc.deleted {
				t.Fatalf("deleted %d sealed segments, want %d (left %v)", got, tc.deleted, ids)
			}
			if st, _ := l.Stat(); st.FreeBytesAtLastPass != sf.free {
				t.Fatalf("FreeBytesAtLastPass %d, want the statfs reading %d", st.FreeBytesAtLastPass, sf.free)
			}
		})
	}
}
