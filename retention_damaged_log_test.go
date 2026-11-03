package quill

import (
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Retention deletes a damaged sealed segment like any other (decision 98), but the deletion
// destroys the only artifacts of the damage (the file and the hole), so it must be logged as a
// loss to damage, at Warn, naming the damage; an undamaged segment is still an ordinary Info line.
func TestRetentionLogsTheDeletionOfADamagedSegmentAsALoss(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, nil)
	fillSegments(t, l, clk, 3) // segments 0..2 sealed, records 1..12
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	seg := "/log/" + segmentName(0)
	data := fileBytes(t, s, seg)
	data[len(data)-1] ^= 0xff
	writeFile(t, s, seg, data)

	var lc logCapture
	l = openRetention(t, s, clk, func(c *Config) { c.MaxRecords = 4; c.Logger = lc.logger() })
	defer l.Close()
	if st, err := l.Stat(); err != nil || len(st.Holes) != 1 {
		t.Fatalf("setup: holes %+v %v", st.Holes, err)
	}
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if !lc.has("deleted damaged segment") || !lc.has("level=WARN") || !lc.has("segment="+seg) || !lc.has("last=") || !lc.has("damage=") {
		t.Fatalf("the deletion of a damaged segment is not logged as damage: %q", lc.msgs)
	}
	// The undamaged segment deleted by the same pass stays an ordinary line.
	if !lc.has("level=INFO msg=\"deleted segment\" log=/log segment=/log/" + segmentName(1)) {
		t.Fatalf("the undamaged segment's deletion is not the ordinary line: %q", lc.msgs)
	}
}

// The age sweep never judges a damaged segment by its own age (it has no readable newest
// record), so it keeps the file until a good segment after it has expired and then deletes
// both. Without the skip a damaged segment would count as expired at once and be deleted
// while every good segment is young.
func TestTheAgeSweepKeepsADamagedSegmentUntilAGoodOneAfterItExpires(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	damagedLog(t, s, clk)
	l := openRetention(t, s, clk, func(c *Config) { c.SegmentMaxAge = -1 })
	defer l.Close()
	if ids := sealedIDs(l); len(ids) != 3 || l.sealed[0].bad == nil || l.sealed[1].bad != nil {
		t.Fatalf("setup: sealed %v, first bad: %v", ids, l.sealed[0].bad)
	}

	// Nothing is an hour old: the damaged oldest segment stays.
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := sealedIDs(l); len(ids) != 3 || ids[0] != 0 {
		t.Fatalf("damaged segment 0 deleted by age with nothing expired: %v", ids)
	}

	// Segment 1 (newest record at minute 1) expires, segment 2 (minute 2) does not: the damaged
	// segment goes with segment 1, and segment 2 stays.
	clk.Advance(time.Hour - 90*time.Second)
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if ids := sealedIDs(l); len(ids) != 1 || ids[0] != 2 {
		t.Fatalf("after segment 1 expired the sealed segments are %v, want [2]", ids)
	}
}
