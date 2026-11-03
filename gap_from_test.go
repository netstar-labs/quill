package quill

import "testing"

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
