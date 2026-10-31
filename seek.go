package quill

import (
	"errors"
	"slices"
	"time"
)

// Seeking by time (decision 59). OffsetForTime returns the smallest sequence whose At is at
// or after t, at whole-second granularity, clamped to [first retained, Head + 1] so the
// result is always a valid Commit argument. The search touches O(log segments) first frames
// and one segment. With non-monotonic times the result is the position the search lands on:
// a valid offset, not necessarily the first record at or after t.

// OffsetForTime returns the smallest sequence whose At is at or after t.
func (l *Log) OffsetForTime(t time.Time) (uint64, error) { return offsetForTime(l, t) }

// OffsetForTime returns the smallest sequence the marker covers whose At is at or after t.
func (r *Reader) OffsetForTime(t time.Time) (uint64, error) { return offsetForTime(r, t) }

// SeekToTime commits OffsetForTime(t), retrying once if retention moved the first retained
// sequence past the result meanwhile.
func (c *Consumer) SeekToTime(t time.Time) error {
	for attempt := 0; ; attempt++ {
		off, err := offsetForTime(c.src, t)
		if err != nil {
			return err
		}
		err = c.Commit(off)
		if err == nil || attempt > 0 || !errors.Is(err, ErrOffsetRange) {
			return err
		}
	}
}

func offsetForTime(src source, t time.Time) (uint64, error) {
	v, release, err := src.view()
	if err != nil {
		return 0, err
	}
	defer release()
	target := t.Unix()
	if len(v.segs) == 0 {
		return v.head + 1, nil
	}
	// The last segment whose first record is strictly before the target, or the first.
	i, _ := slices.BinarySearchFunc(v.segs, target, func(s *segRef, target int64) int {
		if s.at >= target {
			return 1
		}
		return -1
	})
	i = max(i-1, 0)
	s := v.segs[i]
	index := s.index
	if index == nil && s.ensureIndex != nil {
		if index, _, err = s.ensureIndex(); err != nil {
			return 0, scanFailure(err, s.first, s.last)
		}
	}
	// The last anchor strictly before the target, then a scan from it.
	start := int64(0)
	if j, _ := slices.BinarySearchFunc(index, target, func(a anchor, target int64) int {
		if a.at >= target {
			return 1
		}
		return -1
	}); j > 0 {
		start = index[j-1].pos
	}
	r, size, closer, err := s.open()
	if err != nil {
		return 0, err
	}
	defer closer()
	pool := src.bufferPool()
	bufp := pool.Get().(*[]byte)
	defer pool.Put(bufp)
	w := &window{r: r, size: size, buf: (*bufp)[:0]}
	found, seq := false, uint64(0)
	_, err = walk(w, start, 0, false, func(rec Record, _ int64) bool {
		if rec.Seq > s.last {
			return false
		}
		if rec.At >= target {
			found, seq = true, rec.Seq
			return false
		}
		return true
	})
	*bufp = w.buf[:0]
	if err != nil && !found {
		if isFrameError(err) {
			return 0, &GapError{From: s.first, To: s.last}
		}
		return 0, err
	}
	if !found {
		if i+1 < len(v.segs) {
			return v.segs[i+1].first, nil
		}
		return v.head + 1, nil
	}
	return max(seq, v.first), nil
}
