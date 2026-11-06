package quill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"sync"
)

// Read returns flushed records with seq >= offset, ascending, payloads copied: at most limit
// records (limit <= 0 means no limit), and it stops once at least MaxReadBytes have been
// returned, each record counting its payload plus 48 bytes of overhead, so the batch is bounded
// in size and not only in payload bytes, and the total can exceed MaxReadBytes by up to one
// record. It always returns at least one record when one exists. It returns a *GapError where records are unavailable
// (decision 38) and an empty result when offset is past the head. An error met after at least
// one record was collected (a gap, an I/O error, corruption of any segment) is not returned with
// the records: they come back with a nil error and the next call, from where they end, meets it
// again, except that an I/O error may not recur. The one error returned together with records is
// a segment that ends before its last record.
func (l *Log) Read(offset uint64, limit int) ([]Record, error) {
	return readFrom(l, offset, limit)
}

// ReadWait is Read, blocking until records are flushed when offset is at the head. On a failed
// log (ErrFailed) it still returns the records that exist, but at the head it returns the
// failure instead of waiting, and a call already waiting is woken to return it: a failed log
// produces no more records.
func (l *Log) ReadWait(ctx context.Context, offset uint64, limit int) ([]Record, error) {
	return readWait(ctx, l, offset, limit)
}

// Scan is the zero-copy Read: fn is called with each record in order, its payload aliasing a
// buffer that is valid only during the call; fn returns false to stop. Scan stops at a gap
// with a *GapError after delivering what came before it.
func (l *Log) Scan(offset uint64, limit int, fn func(Record) bool) error {
	return scanFrom(l, offset, limit, fn)
}

// ScanAll scans every retained record in order, from sequence 1 when a damaged oldest segment
// is a hole from 1 (it then returns that *GapError first), otherwise from the first retained
// sequence. Scan with offset 0 is the same.
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
	// indexTo returns a Reader's active segment's anchors, extended through the frame holding
	// seq, or nil when it has none (decision 63). A read that has no cursor into the segment
	// starts at the last anchor at or below its sequence instead of at byte 0.
	indexTo func(seq uint64) []anchor
}

// segView is a snapshot of a log's segments: the good ones ascending, the holes, the head
// and the first retained sequence.
type segView struct {
	segs  []*segRef
	holes []Hole
	head  uint64
	first uint64 // the first retained sequence; head + 1 when the log holds no records
}

// start is where a read from the beginning starts: the first retained sequence, or sequence 1
// when a damaged oldest segment left a hole from 1 below it. Retention deleting a prefix is
// expiry and is skipped; damage is loss, and the read meets it as a gap (decision 38).
func (v *segView) start() uint64 {
	if len(v.holes) > 0 && v.holes[0].From == 1 {
		return 1
	}
	return v.first
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

// withLast puts a copy of s that ends at last in s's place in the view and returns it. The
// references of a view may be shared with other views (a Reader keeps them between calls,
// decision 63), so a read that learns where a segment really ends never writes one.
func (v *segView) withLast(s *segRef, last uint64) *segRef {
	c := *s
	c.last = last
	i, _ := slices.BinarySearchFunc(v.segs, s.first, func(s *segRef, target uint64) int {
		if s.first > target {
			return 1
		}
		return -1
	})
	if i > 0 && v.segs[i-1] == s {
		v.segs[i-1] = &c
	}
	return &c
}

// readRecordOverhead is what each record delivered counts against MaxReadBytes besides its
// payload, so that a read of empty or tiny payloads is bounded in memory and not only in
// payload bytes (L2-53). The in-memory Record is 40 bytes, measured with unsafe.Sizeof on
// darwin/arm64 (8 for Seq, 8 for At, 24 for the Payload header); 48 is that plus 8 of
// margin (the figure the owner chose, not a measured one). It is a bound on the size of a batch, not an
// exact account: a Read allocates about 200 bytes per record in all (the slice grows by
// append, so its garbage counts), measured with -benchmem as 215 MB for a 1,000,000-record log
// of empty payloads before the charge and 70 MB after, with 349,526 records per call.
const readRecordOverhead = 48

// maxExpect bounds the slice a Read reserves up front, in records: a larger result grows by
// append past it.
const maxExpect = 4096

// cursor remembers where a scan ended, so the next one starts there without a seek.
type cursor struct {
	seq   uint64 // the next sequence to read
	segID uint64
	pos   int64 // byte position of seq in segID
	valid bool
}

func readFrom(src source, offset uint64, limit int) ([]Record, error) {
	var out []Record
	_, err := scanSource(src, cursor{seq: max(offset, 1)}, limit, src.maxReadBytes(), true, false, func(n int) {
		out = make([]Record, 0, n)
	}, func(rec Record) bool {
		out = append(out, rec)
		return true
	})
	return out, err
}

// scanFrom scans from offset; offset 0 means the beginning (segView.start).
func scanFrom(src source, offset uint64, limit int, fn func(Record) bool) error {
	_, err := scanSource(src, cursor{seq: offset}, limit, 0, false, true, nil, fn)
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

// scanSource drives every read: at most limit records (0: no limit), stopping once at least
// maxBytes (0: no bound) have been delivered, each record counting len(payload) +
// readRecordOverhead, so the last record may carry the total past maxBytes, and always at
// least one. copyPayloads makes each payload its own allocation.
// An error that goes through fail (a gap, an I/O error, corruption of a segment) met after
// records were delivered is returned at once when gapAfter is set (Scan: the callback already
// has the records) and otherwise left for the next call (Read: the records are the result,
// and that call meets the error again unless it was an I/O error that does not recur). A
// segment that ends before its last record does not go through fail: it is returned with the
// records either way. A from.seq of 0 means the beginning (segView.start). It returns
// where it stopped, for the next scan to continue from. expect, when not nil, is called once,
// just before the first record is delivered, with how many records the scan may deliver at
// most (at least 1, at most maxExpect), so a caller collecting them can size its slice
// once; it is not called when no record is delivered (an empty poll, a gap, an error).
func scanSource(src source, from cursor, limit int, maxBytes int, copyPayloads, gapAfter bool, expect func(n int), fn func(Record) bool) (cursor, error) {
	v, release, err := src.view()
	if err != nil {
		return from, err
	}
	defer release()
	cur := from.seq
	if cur == 0 {
		cur = v.start()
	}
	count, bytes := 0, 0
	pool := src.bufferPool()
	bufp := pool.Get().(*[]byte)
	defer pool.Put(bufp)
	stop := false
	next := cursor{seq: cur}
	// fail ends the scan with err, unless records were delivered and the caller leaves the error
	// for the next call (decisions 38, 79 and 84): then the records are the result and that call
	// reads again from where they end. Every error that goes through fail is deferred this way,
	// whatever its class; an I/O error (decision 79) is a fact about the moment, so the next
	// call may not meet it.
	fail := func(err error) (cursor, error) {
		if count > 0 && !gapAfter {
			return next, nil
		}
		return next, err
	}
	for !stop && cur <= v.head && (limit <= 0 || count < limit) && (maxBytes <= 0 || bytes < maxBytes) {
		s, err := v.locate(cur)
		if err != nil {
			return fail(err)
		}
		index := s.index
		var holeTo uint64 // when a scan showed the segment ends before its derived range does
		if index == nil && s.ensureIndex != nil {
			var last uint64
			if index, last, err = s.ensureIndex(); err != nil {
				return fail(scanFailure(err, cur, s.last))
			}
			if last < s.last {
				// The segment ends before the next one starts: a hole the first frames
				// could not show. Serve up to its real end; the gap is reported past it.
				holeTo, s = s.last, v.withLast(s, last)
				if cur > last {
					return fail(&GapError{From: cur, To: holeTo})
				}
			}
		}
		if index == nil && s.indexTo != nil && !(from.valid && from.segID == s.id && from.seq == cur) {
			index = s.indexTo(cur)
		}
		r, size, closer, err := s.open()
		if err != nil {
			var gap *GapError
			if errors.As(err, &gap) {
				err = &GapError{From: cur, To: gap.To} // a deleted segment: the gap starts at cur, the request or the sequence after the last record delivered
			}
			return fail(err)
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
		w := &window{r: r, size: size, buf: (*bufp)[:0], noIndex: true}
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
			if count == 0 && expect != nil {
				n := v.head - rec.Seq + 1
				if limit > 0 {
					n = min(n, uint64(limit))
				}
				expect(int(min(n, maxExpect)))
			}
			count++
			bytes += len(rec.Payload) + readRecordOverhead
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
			if !isFrameError(err) {
				return fail(err) // I/O: a fact about the moment, not the bytes (decision 79)
			}
			if s.active {
				return fail(fmt.Errorf("%w: the active segment below the durable offset: %s", ErrCorrupt, reason(err)))
			}
			return fail(&GapError{From: cur, To: s.last})
		}
		if !done && cur <= s.last {
			return next, fmt.Errorf("%w: segment %d ended at %d, expected %d", ErrCorrupt, s.id, cur-1, s.last)
		}
		if cur > s.last {
			next.valid = false // the next record is in another segment
			if holeTo > 0 && cur <= holeTo {
				return fail(&GapError{From: cur, To: holeTo})
			}
		}
	}
	return next, nil
}

// scanFailure classifies the error of a sealed segment's full scan: damage to the file, or the
// file gone (retention), is a gap over from..to; anything else is an I/O error and is returned
// as it is, so a resume from a gap's To + 1 never skips a segment over a transient fault
// (decisions 38, 79 and 94).
func scanFailure(err error, from, to uint64) error {
	var se *SegmentError
	if errors.As(err, &se) || errors.Is(err, fs.ErrNotExist) {
		return &GapError{From: from, To: to}
	}
	return err
}

// The Log as a source.

func (l *Log) maxReadBytes() int      { return l.cfg.MaxReadBytes }
func (l *Log) bufferPool() *sync.Pool { return &l.bufPool }

// sealedRefsLocked returns the good sealed segments as segRefs, built once and reused until
// l.sealed changes (decision 103). The slice and the refs are shared by every view and are
// never written: a view copies the slice before appending the active segment, and a sealed
// ref has no ensureIndex, so scanSource never writes its last. Called with l.mu held.
func (l *Log) sealedRefsLocked() []*segRef {
	if l.refsValid {
		return l.sealedRefs
	}
	refs := make([]*segRef, 0, len(l.sealed))
	for _, s := range l.sealed {
		if s.bad != nil {
			continue
		}
		refs = append(refs, &segRef{id: s.id, first: s.first, last: s.last, at: s.at, index: s.index,
			open: func() (io.ReaderAt, int64, func(), error) {
				f, size, err := openSegment(l.fs, s.path)
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						// Retention deleted it after the view was taken: a gap, not a path error
						// (decisions 101 and 103).
						return nil, 0, nil, &GapError{From: s.first, To: s.last}
					}
					return nil, 0, nil, err
				}
				if size != s.size {
					_ = f.Close()
					return nil, 0, nil, fmt.Errorf("%w: segment %d is %d bytes, loaded at %d", ErrCorrupt, s.id, size, s.size)
				}
				return f, size, func() { _ = f.Close() }, nil
			}})
	}
	l.sealedRefs, l.refsValid = refs, true
	return refs
}

func (l *Log) view() (*segView, func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, nil, ErrClosed
	}
	a := l.active
	a.readers++
	v := &segView{holes: l.holes, head: l.head}
	refs := l.sealedRefsLocked()
	v.segs = make([]*segRef, len(refs), len(refs)+1)
	copy(v.segs, refs)
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
		_ = a.closeIfIdle()
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
	if l.failed != nil {
		// Records that exist stay readable (the caller reads before it waits), but a failed
		// log produces no more: say so rather than block until the context ends.
		err := l.failed
		l.mu.Unlock()
		return err
	}
	ch := l.waitChanLocked()
	l.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
