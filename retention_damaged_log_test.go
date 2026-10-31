package quill

import (
	"testing"

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
