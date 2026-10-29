package quill

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
)

// Read returns flushed records with seq >= offset, ascending, payloads copied: at most limit
// records (limit <= 0 means no limit) and MaxReadBytes of payload, but always at least one
// when one exists. It returns a *GapError where records are unavailable (decision 38) and an
// empty result when offset is past the head.
func (l *Log) Read(offset uint64, limit int) ([]Record, error) {
	return readFrom(l, offset, limit)
}

// ReadWait is Read, blocking until records are flushed when offset is at the head.
func (l *Log) ReadWait(ctx context.Context, offset uint64, limit int) ([]Record, error) {
	return readWait(ctx, l, offset, limit)
}

// Scan is the zero-copy Read: fn is called with each record in order, its payload aliasing a
// buffer that is valid only during the call; fn returns false to stop. Scan stops at a gap
// with a *GapError after delivering what came before it.
func (l *Log) Scan(offset uint64, limit int, fn func(Record) bool) error {
	return scanFrom(l, offset, limit, fn)
}

// ScanAll scans every retained record in order.
func (l *Log) ScanAll(fn func(Record) bool) error {
	return scanFrom(l, 0, 0, fn)
}

// source is what the shared read, iterator and seek code needs from a Log, a Reader or a
// Consumer's underlying handle.
type source interface {
	// view snapshots the segments; release must be called when the read is done.
	view() (*segView, func(), error)
	// wait blocks until a record at offset may exist, the context ends, or the source closes.
	wait(ctx context.Context, offset uint64) error
	maxReadBytes() int
	bufferPool() *sync.Pool
}

// segRef is one readable segment in a view.
type segRef struct {
	id     uint64
	first  uint64
	last   uint64
	at     int64 // the first frame's At
	index  []anchor
	active bool
	// open returns the segment's data bounded to what may be read, and a closer.
	open func() (io.ReaderAt, int64, func(), error)
	// ensureIndex builds a sealed segment's index on first use (nil when index is set or the
	// segment is active) and returns the segment's exact last sequence, which a Reader only
	// learns by scanning: a last below the derived range end exposes a hole.
	ensureIndex func() ([]anchor, uint64, error)
}

// segView is a snapshot of a log's segments: the good ones ascending, the holes, the head
// and the first retained sequence.
type segView struct {
	segs  []*segRef
	holes []Hole
	head  uint64
	first uint64 // the first retained sequence; head + 1 when the log holds no records
}

// locate finds the segment holding seq, or a gap.
func (v *segView) locate(seq uint64) (*segRef, error) {
	if seq < v.first {
		return nil, &GapError{From: seq, To: v.first - 1}
	}
	for _, h := range v.holes {
		if seq >= h.From && seq <= h.To {
			return nil, &GapError{From: seq, To: h.To}
		}
	}
	i, _ := slices.BinarySearchFunc(v.segs, seq, func(s *segRef, target uint64) int {
		if s.first > target {
			return 1
		}
		return -1
	})
	if i > 0 && seq <= v.segs[i-1].last {
		return v.segs[i-1], nil
	}
	return nil, &GapError{From: seq, To: seq} // between segments with no hole recorded: cannot happen
}

// cursor remembers where a scan ended, so the next one starts there without a seek.
type cursor struct {
	seq   uint64 // the next sequence to read
	segID uint64
	pos   int64 // byte position of seq in segID
	valid bool
}

func readFrom(src source, offset uint64, limit int) ([]Record, error) {
	var out []Record
	_, err := scanSource(src, cursor{seq: max(offset, 1)}, limit, src.maxReadBytes(), true, false, func(rec Record) bool {
		out = append(out, rec)
		return true
	})
	return out, err
}

// scanFrom scans from offset; offset 0 means the first retained sequence.
func scanFrom(src source, offset uint64, limit int, fn func(Record) bool) error {
	_, err := scanSource(src, cursor{seq: offset}, limit, 0, false, true, fn)
	return err
}

func readWait(ctx context.Context, src source, offset uint64, limit int) ([]Record, error) {
	for {
		recs, err := readFrom(src, offset, limit)
		if err != nil || len(recs) > 0 {
			return recs, err
		}
		if err := src.wait(ctx, max(offset, 1)); err != nil {
			return nil, err
		}
	}
}

// scanSource drives every read: at most limit records (0: no limit) and maxBytes of payload
// (0: no bound), always at least one. copyPayloads makes each payload its own allocation.
// A gap met after records were delivered is returned at once when gapAfter is set (Scan:
// the callback already has the records) and otherwise left for the next call (Read: the
// records are the result). A from.seq of 0 means the first retained sequence. It returns
// where it stopped, for the next scan to continue from.
func scanSource(src source, from cursor, limit int, maxBytes int, copyPayloads, gapAfter bool, fn func(Record) bool) (cursor, error) {
	v, release, err := src.view()
	if err != nil {
		return from, err
	}
	defer release()
	cur := from.seq
	if cur == 0 {
		cur = v.first
	}
	count, bytes := 0, 0
	pool := src.bufferPool()
	bufp := pool.Get().(*[]byte)
	defer pool.Put(bufp)
	stop := false
	next := cursor{seq: cur}
	for !stop && cur <= v.head && (limit <= 0 || count < limit) && (maxBytes <= 0 || bytes < maxBytes) {
		s, err := v.locate(cur)
		if err != nil {
			if count > 0 && !gapAfter {
				return next, nil // the gap is reported by the next call
			}
			return next, err
		}
		index := s.index
		var holeTo uint64 // when a scan showed the segment ends before its derived range does
		if index == nil && s.ensureIndex != nil {
			var last uint64
			if index, last, err = s.ensureIndex(); err != nil {
				if count > 0 && !gapAfter {
					return next, nil
				}
				return next, &GapError{From: cur, To: s.last}
			}
			if last < s.last {
				// The segment ends before the next one starts: a hole the first frames
				// could not show. Serve up to its real end; the gap is reported past it.
				holeTo, s.last = s.last, last
				if cur > last {
					if count > 0 && !gapAfter {
						return next, nil
					}
					return next, &GapError{From: cur, To: holeTo}
				}
			}
		}
		r, size, closer, err := s.open()
		if err != nil {
			if count > 0 {
				return next, nil
			}
			return next, err
		}
		// Start at the remembered position, or the last anchor at or below cur.
		start := int64(0)
		if from.valid && from.segID == s.id && from.seq == cur {
			start = from.pos
		} else if i, _ := slices.BinarySearchFunc(index, cur, func(a anchor, t uint64) int {
			if a.seq > t {
				return 1
			}
			return -1
		}); i > 0 {
			start = index[i-1].pos
		}
		w := &window{r: r, size: size, buf: (*bufp)[:0]}
		done := false
		_, err = walk(w, start, 0, false, func(rec Record, pos int64) bool {
			if rec.Seq < cur {
				return true
			}
			if rec.Seq > v.head {
				done = true
				return false // past the marker: a writer's unpublished bytes, never served
			}
			if copyPayloads {
				rec.Payload = slices.Clone(rec.Payload)
			}
			count++
			bytes += len(rec.Payload)
			cur = rec.Seq + 1
			next = cursor{seq: cur, segID: s.id, pos: pos + int64(frameLen(&rec)), valid: true}
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
		closer()
		if err != nil {
			if count > 0 {
				return next, nil
			}
			if s.active {
				return next, fmt.Errorf("%w: the active segment below the durable offset: %v", ErrCorrupt, err)
			}
			return next, &GapError{From: cur, To: s.last}
		}
		if !done && cur <= s.last {
			return next, fmt.Errorf("%w: segment %d ended at %d, expected %d", ErrCorrupt, s.id, cur-1, s.last)
		}
		if cur > s.last {
			next.valid = false // the next record is in another segment
			if holeTo > 0 && cur <= holeTo {
				if count > 0 && !gapAfter {
					return next, nil
				}
				return next, &GapError{From: cur, To: holeTo}
			}
		}
	}
	return next, nil
}

// The Log as a source.

func (l *Log) maxReadBytes() int      { return l.cfg.MaxReadBytes }
func (l *Log) bufferPool() *sync.Pool { return &l.bufPool }

func (l *Log) view() (*segView, func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, nil, ErrClosed
	}
	a := l.active
	a.readers++
	v := &segView{holes: l.holes, head: l.head}
	for _, s := range l.sealed {
		if s.bad != nil {
			continue
		}
		s := s
		v.segs = append(v.segs, &segRef{id: s.id, first: s.first, last: s.last, at: s.at, index: s.index,
			open: func() (io.ReaderAt, int64, func(), error) {
				f, size, err := openSegment(l.fs, s.path)
				if err != nil {
					return nil, 0, nil, err
				}
				if size != s.size {
					_ = f.Close()
					return nil, 0, nil, fmt.Errorf("%w: segment %d is %d bytes, loaded at %d", ErrCorrupt, s.id, size, s.size)
				}
				return f, size, func() { _ = f.Close() }, nil
			}})
	}
	durable := a.durable
	if a.frames > 0 {
		v.segs = append(v.segs, &segRef{id: a.id, first: a.first, last: l.head, at: a.at, index: a.index, active: true,
			open: func() (io.ReaderAt, int64, func(), error) { return a.f, durable, func() {}, nil }})
	}
	if len(v.segs) > 0 {
		v.first = v.segs[0].first
	} else {
		v.first = l.head + 1
	}
	release := func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		a.readers--
		if a.retired && a.readers == 0 && a.f != nil {
			_ = a.f.Close()
			a.f = nil
		}
	}
	return v, release, nil
}

func (l *Log) wait(ctx context.Context, offset uint64) error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return ErrClosed
	}
	if l.head >= offset {
		l.mu.Unlock()
		return nil
	}
	ch := l.flushed
	l.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
