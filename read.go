package quill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
)

// Read returns flushed records with seq >= offset, ascending, payloads copied: at most limit
// records (limit <= 0 means no limit) and MaxReadBytes of payload, but always at least one
// when one exists. It returns a *GapError where records are unavailable (decision 38) and an
// empty result when offset is past the head.
func (l *Log) Read(offset uint64, limit int) ([]Record, error) {
	var out []Record
	err := l.scan(offset, limit, l.cfg.MaxReadBytes, true, func(rec Record) bool {
		out = append(out, rec)
		return true
	})
	return out, err
}

// ReadWait is Read, blocking until records are flushed when offset is at the head.
func (l *Log) ReadWait(ctx context.Context, offset uint64, limit int) ([]Record, error) {
	for {
		recs, err := l.Read(offset, limit)
		if err != nil || len(recs) > 0 {
			return recs, err
		}
		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			return nil, ErrClosed
		}
		if l.head >= max(offset, 1) {
			l.mu.Unlock()
			continue // flushed between the read and here
		}
		ch := l.flushed
		l.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Scan is the zero-copy Read: fn is called with each record in order, its payload aliasing a
// buffer that is valid only during the call; fn returns false to stop. Scan stops at a gap
// with a *GapError after delivering what came before it.
func (l *Log) Scan(offset uint64, limit int, fn func(Record) bool) error {
	return l.scan(offset, limit, 0, false, fn)
}

// ScanAll scans every retained record in order.
func (l *Log) ScanAll(fn func(Record) bool) error {
	return l.scan(1, 0, 0, false, fn)
}

// readView is a snapshot of the log's segments taken under the lock. The active segment's
// descriptor is pinned until release.
type readView struct {
	sealed  []*segment
	active  *activeSegment
	durable int64
	head    uint64
	holes   []Hole
}

func (l *Log) view() (readView, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return readView{}, ErrClosed
	}
	l.active.readers++
	return readView{sealed: l.sealed, active: l.active, durable: l.active.durable, head: l.head, holes: l.holes}, nil
}

func (l *Log) release(v readView) {
	l.mu.Lock()
	defer l.mu.Unlock()
	v.active.readers--
	if v.active.retired && v.active.readers == 0 && v.active.f != nil {
		_ = v.active.f.Close()
		v.active.f = nil
	}
}

// first returns the first retained sequence of the view.
func (v readView) first() uint64 {
	if len(v.sealed) > 0 {
		for _, s := range v.sealed {
			if s.bad == nil {
				return s.first
			}
		}
	}
	return v.active.first
}

// locate finds the segment holding seq: a sealed one, the active one (nil, true), or a gap.
func (v readView) locate(seq uint64) (*segment, bool, error) {
	if first := v.first(); seq < first {
		return nil, false, &GapError{From: seq, To: first - 1}
	}
	for _, h := range v.holes {
		if seq >= h.From && seq <= h.To {
			return nil, false, &GapError{From: seq, To: h.To}
		}
	}
	// The last good sealed segment whose first sequence is at or below seq.
	i, _ := slices.BinarySearchFunc(v.sealed, seq, func(s *segment, target uint64) int {
		if s.bad != nil {
			return -1 // sorts as "below": it never holds the answer
		}
		if s.first > target {
			return 1
		}
		return -1
	})
	for j := i - 1; j >= 0; j-- {
		s := v.sealed[j]
		if s.bad != nil {
			continue
		}
		if seq <= s.last {
			return s, false, nil
		}
		break
	}
	if seq >= v.active.first && v.active.frames > 0 {
		return nil, true, nil
	}
	return nil, false, &GapError{From: seq, To: seq} // a position between segments that no hole covers: cannot happen
}

// scan drives Read and Scan: at most limit records (0: no limit) and maxBytes of payload
// (0: no bound), always at least one. copyPayloads makes each payload its own allocation.
func (l *Log) scan(offset uint64, limit int, maxBytes int, copyPayloads bool, fn func(Record) bool) error {
	v, err := l.view()
	if err != nil {
		return err
	}
	defer l.release(v)
	cur := max(offset, 1)
	count, bytes := 0, 0
	bufp := l.bufPool.Get().(*[]byte)
	defer l.bufPool.Put(bufp)
	stop := false
	for !stop && cur <= v.head && (limit <= 0 || count < limit) && (maxBytes <= 0 || bytes < maxBytes) {
		s, active, err := v.locate(cur)
		if err != nil {
			if count > 0 {
				return nil // the gap is reported by the next call
			}
			return err
		}
		var r io.ReaderAt
		var size int64
		var index []anchor
		var last uint64
		if active {
			r, size, index, last = v.active.f, v.durable, v.active.index, v.head
		} else {
			f, sz, err := openSegment(l.fs, s.path)
			if err != nil {
				if count > 0 {
					return nil
				}
				return err
			}
			r, size, index, last = f, sz, s.index, s.last
			defer f.Close()
			if sz != s.size {
				return fmt.Errorf("%w: segment %d is %d bytes, loaded at %d", ErrCorrupt, s.id, sz, s.size)
			}
		}
		// Seek by the sparse index to the last anchor at or below cur.
		start := int64(0)
		if i, _ := slices.BinarySearchFunc(index, cur, func(a anchor, t uint64) int {
			if a.seq > t {
				return 1
			}
			return -1
		}); i > 0 {
			start = index[i-1].pos
		}
		w := &window{r: r, size: size, buf: (*bufp)[:0]}
		done := false
		_, err = walk(w, start, 0, false, func(rec Record, _ int64) bool {
			if rec.Seq < cur {
				return true
			}
			if copyPayloads {
				rec.Payload = slices.Clone(rec.Payload)
			}
			count++
			bytes += len(rec.Payload)
			cur = rec.Seq + 1
			if !fn(rec) {
				stop, done = true, true
				return false
			}
			if (limit > 0 && count >= limit) || (maxBytes > 0 && bytes >= maxBytes) {
				done = true
				return false
			}
			return true
		})
		*bufp = w.buf[:0]
		if err != nil {
			if count > 0 {
				return nil
			}
			if active {
				return fmt.Errorf("%w: the active segment below the durable offset: %v", ErrCorrupt, err)
			}
			return &GapError{From: cur, To: last}
		}
		if !done && cur <= last {
			// The segment ended before its last sequence: cannot happen for a verified segment.
			return fmt.Errorf("%w: segment ended at %d, expected %d", ErrCorrupt, cur-1, last)
		}
	}
	return nil
}

var _ = errors.Is
