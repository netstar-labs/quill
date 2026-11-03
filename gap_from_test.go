package quill

import (
	"context"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// GapError.From is the first unavailable sequence: the offset asked for when the scan
// delivered nothing before the gap, otherwise the sequence after the last record it delivered
// (decision 84). The documentation says so; this pins it for every Scan form.
func TestScanGapFromIsTheFirstUnavailableSequence(t *testing.T) {
	scan := func(t *testing.T, r *Reader, offset uint64) ([]uint64, error) {
		t.Helper()
		var got []uint64
		var err error
		if offset == 0 {
			err = r.ScanAll(func(rec Record) bool { got = append(got, rec.Seq); return true })
		} else {
			err = r.Scan(offset, 0, func(rec Record) bool { got = append(got, rec.Seq); return true })
		}
		return got, err
	}
	t.Run("nothing delivered: the offset asked for", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		got, err := scan(t, r, 6)
		if len(got) != 0 || !isGap(err, 6, 8) {
			t.Fatalf("Scan(6): %v, %v", got, err)
		}
	})
	t.Run("records delivered: the sequence after the last one", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		got, err := scan(t, r, 3)
		if len(got) != 2 || !isGap(err, 5, 8) {
			t.Fatalf("Scan(3): %v, %v", got, err)
		}
	})
	t.Run("ScanAll, records delivered", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		got, err := scan(t, r, 0)
		if len(got) != 4 || !isGap(err, 5, 8) {
			t.Fatalf("ScanAll: %v, %v", got, err)
		}
	})
	// Consumer.Scan returns the gap at once after delivering records, as Scan does, so its From
	// is the sequence after the last record delivered, not the consumer's offset.
	consumerScan := func(t *testing.T, r *Reader, offset uint64) ([]uint64, error) {
		t.Helper()
		c, err := r.Consumer("gapfrom", ConsumerOptions{})
		if err != nil {
			t.Fatalf("Consumer: %v", err)
		}
		t.Cleanup(func() { _ = c.Close() })
		if err := c.Commit(offset); err != nil {
			t.Fatalf("Commit(%d): %v", offset, err)
		}
		var got []uint64
		err = c.Scan(0, func(rec Record) bool { got = append(got, rec.Seq); return true })
		return got, err
	}
	t.Run("Consumer.Scan, nothing delivered: the consumer's offset", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		got, err := consumerScan(t, r, 6)
		if len(got) != 0 || !isGap(err, 6, 8) {
			t.Fatalf("Consumer.Scan at 6: %v, %v", got, err)
		}
	})
	t.Run("Consumer.Scan, records delivered: the sequence after the last one", func(t *testing.T) {
		s, r, victim, _ := vanishingLog(t)
		removeOnOpen(s, victim, 0)
		got, err := consumerScan(t, r, 3)
		if len(got) != 2 || !isGap(err, 5, 8) {
			t.Fatalf("Consumer.Scan at 3: %v, %v", got, err)
		}
	})
}

// The Log's own scans and iterators share the code of a Reader's: a sealed segment that retention
// deletes after the view was taken and before it is opened is a gap, and its From is the first
// unavailable sequence, not always the offset asked for (decisions 84, 101, 103).
func TestLogGapFromIsTheFirstUnavailableSequence(t *testing.T) {
	vanishing := func(t *testing.T) *Log {
		t.Helper()
		s := fsx.NewSim()
		clk := newClock()
		l := openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
		t.Cleanup(func() { _ = l.Close() })
		fillSegments(t, l, clk, 3)
		appendN(t, l, 1)
		mustFlush(t, l)
		ids := segmentFiles(t, s, "/log")
		if len(ids) < 4 {
			t.Fatalf("segments %v", ids)
		}
		removeOnOpen(s, "/log/"+segmentName(ids[1]), 0) // records 5 to 8
		return l
	}
	scanFrom := func(l *Log, offset uint64) ([]uint64, error) {
		var got []uint64
		err := l.Scan(offset, 0, func(rec Record) bool { got = append(got, rec.Seq); return true })
		return got, err
	}
	t.Run("Scan, nothing delivered: the offset asked for", func(t *testing.T) {
		got, err := scanFrom(vanishing(t), 6)
		if len(got) != 0 || !isGap(err, 6, 8) {
			t.Fatalf("Scan(6): %v, %v", got, err)
		}
	})
	t.Run("Scan, records delivered: the sequence after the last one", func(t *testing.T) {
		got, err := scanFrom(vanishing(t), 3)
		if len(got) != 2 || !isGap(err, 5, 8) {
			t.Fatalf("Scan(3): %v, %v", got, err)
		}
	})
	t.Run("ScanAll, records delivered", func(t *testing.T) {
		var got []uint64
		err := vanishing(t).ScanAll(func(rec Record) bool { got = append(got, rec.Seq); return true })
		if len(got) != 4 || !isGap(err, 5, 8) {
			t.Fatalf("ScanAll: %v, %v", got, err)
		}
	})
	for _, form := range []string{"Records", "Follow"} {
		t.Run(form+", records delivered: the sequence after the last one", func(t *testing.T) {
			l := vanishing(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			seq := l.Records(ctx, 3)
			if form == "Follow" {
				seq = l.Follow(ctx, 3)
			}
			var got []uint64
			var gapErr error
			for rec, err := range seq {
				if err != nil {
					gapErr = err
					break
				}
				got = append(got, rec.Seq)
			}
			if len(got) != 2 || !isGap(gapErr, 5, 8) {
				t.Fatalf("%s(3): %v, %v", form, got, gapErr)
			}
		})
	}
}
