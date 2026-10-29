package quill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// ReaderOptions configures OpenReader.
type ReaderOptions struct {
	// PollInterval is how often ReadWait and Follow re-read quill.tail (50 ms).
	PollInterval time.Duration
	// MaxReadBytes bounds the payload bytes one Read returns (16 MiB).
	MaxReadBytes int

	fs    fsx.FS
	clock func() time.Time
}

// DefaultPollInterval is the Reader's default poll interval.
const DefaultPollInterval = 50 * time.Millisecond

// Reader reads a log from another process without taking any lock (decisions 11, 63). It
// trusts quill.tail alone: every call re-reads the marker and re-lists the directory, reads
// no byte of the active segment past the marker's offset and no sequence past its sequence,
// and never decides whether a writer is alive. A Reader is safe for concurrent use.
type Reader struct {
	fs   fsx.FS
	dir  string
	id   logID
	opts ReaderOptions

	mu     sync.Mutex
	segs   map[uint64]*readerSegment // sealed segments by id, built on first use
	closed bool

	bufPool sync.Pool
}

// readerSegment is what a Reader has learned about one sealed, immutable segment.
type readerSegment struct {
	id   uint64
	path string
	size int64
	// From the first frame, read when the segment is first listed.
	first uint64
	at    int64
	bad   error // the first frame could not be read: the segment's range is a hole
	// From one full scan, built when the segment is first read (decision 63). The result is
	// published by one atomic store, so view can look at it while another call is scanning.
	result atomic.Pointer[scanResult]
	// scanMu serialises scans; scanErr holds damage only (a *SegmentError), which is a fact
	// about the file. An I/O error is a fact about the moment and is not kept (decision 94).
	scanMu  sync.Mutex
	scanErr error
}

// scanResult is what one full scan of a sealed segment learned; it is never modified after
// it is stored.
type scanResult struct {
	index  []anchor
	last   uint64
	frames int
	lastAt int64
}

// OpenReader opens a log directory for reading. It returns ErrNoLog when the directory holds
// no quill.format or no quill.tail yet (a reader that arrives mid-creation retries), and
// ErrFutureVersion for a marker newer than this build.
func OpenReader(dir string, opts ReaderOptions) (*Reader, error) {
	if opts.PollInterval <= 0 {
		opts.PollInterval = DefaultPollInterval
	}
	if opts.MaxReadBytes <= 0 {
		opts.MaxReadBytes = DefaultMaxReadBytes
	}
	if opts.fs == nil {
		opts.fs = fsx.OS{}
	}
	if opts.clock == nil {
		opts.clock = time.Now
	}
	r := &Reader{fs: opts.fs, dir: filepath.Clean(dir), opts: opts, segs: map[uint64]*readerSegment{}}
	r.bufPool.New = func() any { b := make([]byte, 0, windowChunk); return &b }
	data, err := readState(r.fs, r.path(formatName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s has no %s", ErrNoLog, r.dir, formatName)
		}
		return nil, err
	}
	id, err := decodeFormat(data)
	if err != nil {
		if errors.Is(err, ErrFutureVersion) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	r.id = id
	if _, err := r.readTail(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Reader) path(name string) string { return filepath.Join(r.dir, name) }

// ID returns the log's identity.
func (r *Reader) ID() string { return string(r.id) }

// Close releases the reader. Reads in progress finish; later calls return ErrClosed.
func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}
	r.closed = true
	return nil
}

// readTail reads and parses quill.tail, retrying briefly on a transient failure.
func (r *Reader) readTail() (tail, error) {
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		data, err := readState(r.fs, r.path(tailName))
		if err == nil {
			tl, err := decodeTail(data)
			if err == nil && tl.Seq == math.MaxUint64 {
				// The last sequence is never assigned, so head+1 cannot wrap (see appendLocked).
				return tail{}, fmt.Errorf("%w: %s names sequence %d, which is never assigned", ErrCorrupt, tailName, tl.Seq)
			}
			if err == nil {
				return tl, nil
			}
			last = fmt.Errorf("%w: %v", ErrCorrupt, err)
		} else if errors.Is(err, fs.ErrNotExist) {
			return tail{}, fmt.Errorf("%w: %s has no %s", ErrNoLog, r.dir, tailName)
		} else {
			last = err
		}
		time.Sleep(time.Millisecond)
	}
	return tail{}, last
}

// Head returns the last durable sequence the marker names.
func (r *Reader) Head() (uint64, error) {
	tl, err := r.readTail()
	return tl.Seq, err
}

// Read is the Log's Read, bounded by the marker; ReadWait, Scan and ScanAll follow.
func (r *Reader) Read(offset uint64, limit int) ([]Record, error) { return readFrom(r, offset, limit) }

// ReadWait is Read, blocking until a record is available, ctx is done or the handle closes.
func (r *Reader) ReadWait(ctx context.Context, offset uint64, limit int) ([]Record, error) {
	return readWait(ctx, r, offset, limit)
}

// Scan is the zero-copy Read, as Log.Scan: at most limit records (limit <= 0: no limit), payloads valid only during fn.
func (r *Reader) Scan(offset uint64, limit int, fn func(Record) bool) error {
	return scanFrom(r, offset, limit, fn)
}

// ScanAll scans every retained record the marker names, in order.
func (r *Reader) ScanAll(fn func(Record) bool) error { return scanFrom(r, 0, 0, fn) }

func (r *Reader) maxReadBytes() int      { return r.opts.MaxReadBytes }
func (r *Reader) bufferPool() *sync.Pool { return &r.bufPool }

// wait polls the marker until it names offset or the context ends.
func (r *Reader) wait(ctx context.Context, offset uint64) error {
	t := time.NewTicker(r.opts.PollInterval)
	defer t.Stop()
	for {
		r.mu.Lock()
		closed := r.closed
		r.mu.Unlock()
		if closed {
			return ErrClosed
		}
		tl, err := r.readTail()
		if err != nil {
			return err
		}
		if tl.Seq >= offset {
			return nil
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// segment returns the cached entry for a sealed segment, reading its first frame on the
// first call. A cached entry costs no file operation; it is dropped when a read finds the
// file gone or changed in size or a view no longer lists it, and rebuilt on the next view (decision 63). A segment that
// is damaged (not a regular file, oversize, a bad first frame) is cached as bad; one that
// could not be opened for an I/O reason is not, so the next view re-examines it (decision 94).
func (r *Reader) segment(id uint64) *readerSegment {
	r.mu.Lock()
	s, ok := r.segs[id]
	r.mu.Unlock()
	if ok {
		return s
	}
	path := r.path(segmentName(id))
	s = &readerSegment{id: id, path: path}
	f, size, err := openSegment(r.fs, path)
	var se *SegmentError
	if err != nil && !errors.As(err, &se) {
		// An I/O error (EMFILE, EACCES, a transient ENOENT) says nothing lasting about the
		// file: this view sees a hole, the entry is not cached, and the next view looks again.
		s.bad = err
		return s
	}
	if err != nil {
		s.bad = err
	} else {
		s.size = size
		w := &window{r: f, size: size}
		rec, _, ferr := w.frameAt(0)
		if ferr == nil && rec.Seq == 0 {
			ferr = errBadFrame // a segment's first frame is at least 1 (decision 71)
		}
		if ferr != nil {
			s.bad = &SegmentError{Path: path, Reason: "first frame: " + reason(ferr)}
		} else {
			s.first, s.at = rec.Seq, rec.At
		}
		_ = f.Close()
	}
	r.mu.Lock()
	if cached, ok := r.segs[id]; ok {
		s = cached // another call built it meanwhile
	} else {
		r.segs[id] = s
	}
	r.mu.Unlock()
	return s
}

// forget drops a cached segment whose file changed or disappeared.
func (r *Reader) forget(id uint64) {
	r.mu.Lock()
	delete(r.segs, id)
	r.mu.Unlock()
}

// prune drops every cached segment that the latest listing no longer holds: retention deleted
// it, and nothing would ever look it up again (decision 63). It is a no-op while the cache is
// no larger than the listing, which is the case on every view of a log that is not rotating.
func (r *Reader) prune(listed []*readerSegment) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.segs) <= len(listed) {
		return
	}
	live := make(map[uint64]struct{}, len(listed))
	for _, s := range listed {
		live[s.id] = struct{}{}
	}
	for id := range r.segs {
		if _, ok := live[id]; !ok {
			delete(r.segs, id)
		}
	}
}

// scan builds a sealed segment's index by one full scan. Success and damage are final; an
// I/O error is returned but not kept, so the next call scans again (decision 94).
func (s *readerSegment) scan(fsys fsx.FS) error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	if s.result.Load() != nil || s.scanErr != nil {
		return s.scanErr
	}
	f, size, err := openSegment(fsys, s.path)
	if err == nil {
		defer f.Close()
		var info segInfo
		if info, err = verifySegment(f, size, s.path); err == nil {
			s.result.Store(&scanResult{index: info.index, last: info.last.Seq, frames: info.frames, lastAt: info.last.At})
			return nil
		}
	}
	var se *SegmentError
	if errors.As(err, &se) {
		s.scanErr = err
	}
	return err
}

// firstFrameChunk is the read size of the active segment's first-frame probe: a frame's
// header and a small payload, not the 64 KiB a scan starts with.
const firstFrameChunk = 4 << 10

// view reads the marker, lists the directory, and builds the segment ranges from first
// frames (decision 47): a sealed segment's range ends where the next one starts.
func (r *Reader) view() (*segView, func(), error) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return nil, nil, ErrClosed
	}
	tl, err := r.readTail()
	if err != nil {
		return nil, nil, err
	}
	// Not listSegments: this runs on every call, and the extra id slice it builds costs more
	// allocations and bytes per view than iterating the names directly.
	names, err := r.fs.ReadDir(r.dir)
	if err != nil {
		return nil, nil, err
	}
	sealed := make([]*readerSegment, 0, len(names))
	for _, n := range names {
		if id, ok := parseSegmentName(n); ok && id < tl.Seg {
			sealed = append(sealed, r.segment(id))
		}
	}
	r.prune(sealed)
	// The active segment: its first frame, if the marker covers one.
	var activeFirst uint64
	var activeAt int64
	var activeFile fsx.File
	if tl.Off > 0 {
		f, size, err := openSegment(r.fs, r.path(segmentName(tl.Seg)))
		if err != nil {
			return nil, nil, err
		}
		if size < int64(tl.Off) {
			_ = f.Close()
			return nil, nil, fmt.Errorf("%w: segment %d holds %d bytes, the marker says %d", ErrCorrupt, tl.Seg, size, tl.Off)
		}
		// One frame is decoded and only its Seq and At are kept, so the pooled buffer is
		// returned straight away and nothing aliases it afterwards.
		bufp := r.bufPool.Get().(*[]byte)
		w := &window{r: f, size: int64(tl.Off), buf: (*bufp)[:0], chunk: firstFrameChunk}
		rec, _, ferr := w.frameAt(0)
		*bufp = w.buf[:0]
		r.bufPool.Put(bufp)
		if ferr != nil || rec.Seq == 0 || rec.Seq > tl.Seq {
			_ = f.Close()
			return nil, nil, fmt.Errorf("%w: the active segment's first frame does not decode below the marker", ErrCorrupt)
		}
		activeFirst, activeAt, activeFile = rec.Seq, rec.At, f
	}
	v := &segView{head: tl.Seq}
	// First the ranges of the good segments, then one contiguity check over them and the
	// active segment, then the segment references.
	type cand struct {
		s     *readerSegment
		last  uint64
		index []anchor
	}
	cands := make([]cand, 0, len(sealed))
	rs := make([]seqRange, 0, len(sealed)+1)
	leadBad := false // a bad segment before the first kept one
	for i, s := range sealed {
		if s.bad != nil {
			leadBad = leadBad || len(cands) == 0
			continue
		}
		// The range ends at the next good segment's first sequence, or the active one's.
		var last uint64
		nextKnown := false
		for _, n := range sealed[i+1:] {
			if n.bad == nil {
				last, nextKnown = n.first-1, true
				break
			}
		}
		if !nextKnown {
			if activeFirst > 0 {
				last = activeFirst - 1
			} else {
				last = tl.Seq
			}
		}
		if last < s.first {
			continue // an empty range: the marker predates it (a stale marker after a rotation crash)
		}
		// One load: a scan may finish in another call while this loop runs.
		var index []anchor
		if res := s.result.Load(); res != nil && res.index != nil {
			index = res.index
			if res.last < last {
				last = res.last // scanned already: its real end is known; the next segment or the marker reports the hole
			}
		}
		cands = append(cands, cand{s, last, index})
		rs = append(rs, seqRange{id: s.id, first: s.first, last: last})
	}
	if activeFile != nil {
		rs = append(rs, seqRange{id: tl.Seg, first: activeFirst, last: tl.Seq})
	}
	holes, err := contiguity(rs, leadBad)
	if err != nil {
		if activeFile != nil {
			_ = activeFile.Close()
		}
		return nil, nil, err
	}
	v.holes = holes
	// One slab holds every reference, the active one last: a view is short-lived and so are
	// its refs. A scanned segment already has its index, so only the others get ensureIndex.
	refs := make([]segRef, len(cands)+1)
	v.segs = make([]*segRef, 0, len(cands)+1)
	for i, c := range cands {
		s, last := c.s, c.last
		ref := &refs[i]
		*ref = segRef{id: s.id, first: s.first, last: last, at: s.at, index: c.index,
			open: func() (io.ReaderAt, int64, func(), error) {
				f, size, err := openSegment(r.fs, s.path)
				if err != nil {
					r.forget(s.id)
					if errors.Is(err, fs.ErrNotExist) {
						// Retention deleted it after the listing: a gap, not a path error (decision 101).
						return nil, 0, nil, &GapError{From: s.first, To: last}
					}
					return nil, 0, nil, err
				}
				if size != s.size {
					_ = f.Close()
					r.forget(s.id)
					return nil, 0, nil, fmt.Errorf("%w: segment %d changed size from %d to %d", ErrCorrupt, s.id, s.size, size)
				}
				return f, size, func() { _ = f.Close() }, nil
			}}
		if c.index == nil {
			ref.ensureIndex = func() ([]anchor, uint64, error) {
				if err := s.scan(r.fs); err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						r.forget(s.id) // retention deleted it: the next view lists without it
					}
					return nil, 0, err
				}
				res := s.result.Load()
				return res.index, res.last, nil
			}
		}
		v.segs = append(v.segs, ref)
	}
	if activeFile != nil {
		off := int64(tl.Off)
		ref := &refs[len(cands)]
		*ref = segRef{id: tl.Seg, first: activeFirst, last: tl.Seq, at: activeAt, active: true,
			open: func() (io.ReaderAt, int64, func(), error) { return activeFile, off, func() {}, nil }}
		v.segs = append(v.segs, ref)
	} else if len(cands) > 0 && tl.Seq > cands[len(cands)-1].last {
		v.holes = append(v.holes, Hole{From: cands[len(cands)-1].last + 1, To: tl.Seq})
	} else if leadBad && len(cands) == 0 && tl.Seq > 0 {
		v.holes = append(v.holes, Hole{From: 1, To: tl.Seq})
	}
	if len(v.segs) > 0 {
		v.first = v.segs[0].first
	} else {
		v.first = tl.Seq + 1
	}
	release := func() {
		if activeFile != nil {
			_ = activeFile.Close()
		}
	}
	return v, release, nil
}

// MarkerAge returns how long ago quill.tail was last written: the only liveness hint a
// reader gets (decision 47).
func (r *Reader) MarkerAge() (time.Duration, error) {
	info, err := r.fs.Lstat(r.path(tailName))
	if err != nil {
		return 0, err
	}
	return r.opts.clock().Sub(info.ModTime()), nil
}
