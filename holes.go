package quill

import "fmt"

// seqRange is one good segment's place in the sequence space: its id and the first and last
// sequence it is taken to hold.
type seqRange struct {
	id          uint64
	first, last uint64
}

// contiguity is the one place that turns the ordered good segments into holes and the first
// overlap. Both the writer's load and the reader's view call it; each keeps its own tail step
// (the writer's refusal when the head is below the marker, decision 39; the reader's marker
// hole, decision 11). rs lists the good segments in id order, the active one last when it
// holds frames. badFirst says a bad segment came before the first one in rs: its range is
// unknown, so it is a hole from sequence 1 up to the first good segment's start. A gap
// between two good segments is a hole; a range starting at or below the one before is
// overlap, an ErrCorrupt (the holes are then discarded by the caller). An empty hole is
// never reported.
func contiguity(rs []seqRange, badFirst bool) ([]Hole, error) {
	var holes []Hole
	for i, r := range rs {
		from := uint64(1)
		switch {
		case i > 0:
			prev := rs[i-1]
			if r.first <= prev.last {
				return nil, fmt.Errorf("%w: segment %d starts at %d, inside the one before (segment %d, ends at %d)", ErrCorrupt, r.id, r.first, prev.id, prev.last)
			}
			from = prev.last + 1
		case !badFirst:
			continue
		}
		if r.first > from {
			holes = append(holes, Hole{From: from, To: r.first - 1})
		}
	}
	return holes, nil
}
