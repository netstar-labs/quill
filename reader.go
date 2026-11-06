package quill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// ReaderOptions configures OpenReader.
type ReaderOptions struct {
	// PollInterval is how often ReadWait and Follow re-read quill.tail (50 ms).
	PollInterval time.Duration
	// MaxReadBytes is the size after which one Read stops: each record counts its payload plus 48
	// bytes of overhead, so it bounds the size of a batch and not the sum of the payloads. A Read
	// returns at least one record, and the last record may carry the total past this bound (16 MiB;
	// at the default, about 349,000 empty records or 342,000 one-byte ones per call).
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
	// perm is the log's file mode, read from quill.tail when the Reader opened: the mode of
	// the consumer files this Reader creates and of what Archive creates (decisions 41, 51).
	perm fs.FileMode

	mu     sync.Mutex
	segs   map[uint64]*readerSegment // sealed segments by id, built on first use
	closed bool

	// actMu guards act, the verified prefix of the active segment (decision 63, amended for
	// A3-18). It is held while the prefix is extended, so concurrent reads of the active
	// segment on one Reader take turns.
	actMu sync.Mutex
	act   *activeIndex

	// plan is the last view's references to the sealed segments, kept while the marker still
	// names the same active segment and the oldest sealed segment is still there, so a view
	// that finds nothing changed does not list the directory (decision 63, amended for A3-18).
	// It is guarded by mu. scanGen counts finished scans of sealed segments: one tells a view
	// where a segment really ends, so a plan built before it is not used after it.
	plan    *sealedPlan
	scanGen atomic.Uint64

	bufPool sync.Pool
}

// sealedPlan is what one view made of the sealed segments: their references and the holes
// among them. Nothing in it is written once it is stored: a view copies the slice of
// references, and a read that learns where a segment really ends copies the reference
// (segView.withLast).
type sealedPlan struct {
	seg         uint64 // the marker's segment when the plan was built
	activeFirst uint64 // that segment's first sequence, 0 when the marker covered no frame of it
	seq         uint64 // the marker's sequence, which the plan depends on only when activeFirst is 0
	gen         uint64 // Reader.scanGen when the plan was built
	oldest      string // the path of the oldest sealed segment listed, "" when there was none
	refs        []*segRef
	holes       []Hole
}

// activeIndex is what a Reader has verified of the active segment: the frames in [0, end) are
// whole, contiguous and CRC-good, and index holds one anchor per indexStride of them, the first
// frame always. Bytes below the marker's offset are durable and are never rewritten (decisions
// 39 and 65), so the prefix stays true for as long as it is the same file and has not shrunk.
// Nothing above the marker's offset, and no payload, is kept.
type activeIndex struct {
	file    fs.FileInfo // the file the prefix was verified in, compared with fsx.SameFile
	first   uint64      // the first frame's Seq and At as the view read them: a file system may reuse
	firstAt int64       // an inode, so a new segment can look like the file it replaced
	end     int64       // the byte after the last verified frame
	last    uint64      // that frame's Seq
	next    int64       // the position at or after which the next anchor is taken
	index   []anchor
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
	abs, err := filepath.Abs(dir) // bound to the working directory now, as Open does
	if err != nil {
		return nil, err
	}
	r := &Reader{fs: opts.fs, dir: abs, opts: opts, segs: map[uint64]*readerSegment{}}
	r.bufPool.New = func() any { b := make([]byte, 0, windowChunk); return &b }
	id, err := loadFormat(r.fs, r.path(formatName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s has no %s", ErrNoLog, r.dir, formatName)
		}
		return nil, err
	}
	r.id = id
	if _, err := r.readTail(); err != nil {
		return nil, err
	}
	// quill.tail is written with Config.Perm exactly (decision 74), so its mode is the log's.
	// The owner's read and write bits stay: this process must reopen what it creates.
	info, err := r.fs.Lstat(r.path(tailName))
	if err != nil {
		return nil, err
	}
	r.perm = info.Mode().Perm() | 0o600
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

// Read is the Log's Read, bounded by the marker, and leaves an error met after records the same
// way (see Log.Read); ReadWait, Scan and ScanAll follow.
func (r *Reader) Read(offset uint64, limit int) ([]Record, error) { return readFrom(r, offset, limit) }

// ReadWait is Read, blocking until a record is available, ctx is done or the handle closes.
func (r *Reader) ReadWait(ctx context.Context, offset uint64, limit int) ([]Record, error) {
	return readWait(ctx, r, offset, limit)
}

// Scan is the zero-copy Read, as Log.Scan: at most limit records (limit <= 0: no limit), payloads valid only during fn.
func (r *Reader) Scan(offset uint64, limit int, fn func(Record) bool) error {
	return scanFrom(r, offset, limit, fn)
}

// ScanAll scans every retained record the marker names, in order, as Log.ScanAll: from
// sequence 1 when a damaged oldest segment is a hole from 1, so the read meets the gap.
func (r *Reader) ScanAll(fn func(Record) bool) error { return scanFrom(r, 0, 0, fn) }

func (r *Reader) maxReadBytes() int      { return r.opts.MaxReadBytes }
func (r *Reader) bufferPool() *sync.Pool { return &r.bufPool }

// wait polls the marker until it names offset or the context ends. It also returns ErrWrongLog
// when the directory now holds another log than the one opened.
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
		// Once per poll: a directory replaced by another log since open is not waited on.
		if err := checkLogID(r.fs, r.dir, r.id); err != nil {
			return err
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
// is damaged (not a regular file, oversize, a bad first frame) is cached as bad. One that could
// not be opened for an I/O reason is not cached either: this view sees a hole and the next view
// looks again (decision 94). An I/O error reading the first frame is a read, not an open: it is
// returned, uncached, and fails the view, so it never becomes a gap (decisions 79 and 94).
func (r *Reader) segment(id uint64) (*readerSegment, error) {
	r.mu.Lock()
	s, ok := r.segs[id]
	r.mu.Unlock()
	if ok {
		return s, nil
	}
	path := r.path(segmentName(id))
	s = &readerSegment{id: id, path: path}
	f, size, err := openSegment(r.fs, path)
	var se *SegmentError
	if err != nil && !errors.As(err, &se) {
		// An I/O error (EMFILE, EACCES, a transient ENOENT) says nothing lasting about the
		// file: this view sees a hole, the entry is not cached, and the next view looks again.
		s.bad = err
		return s, nil
	}
	if err != nil {
		s.bad = err
	} else {
		s.size = size
		w := &window{r: f, size: size, chunk: firstFrameChunk}
		rec, _, ferr := w.frameAt(0)
		if ferr == nil && rec.Seq == 0 {
			ferr = errBadFrame // a segment's first frame is at least 1 (decision 71)
		}
		_ = f.Close()
		switch {
		case ferr != nil && !isFrameError(ferr):
			return nil, ferr // I/O: a fact about the moment, not the bytes (decision 79); not cached
		case ferr != nil:
			s.bad = &SegmentError{Path: path, Reason: "first frame: " + reason(ferr)}
		default:
			s.first, s.at = rec.Seq, rec.At
		}
	}
	r.mu.Lock()
	if cached, ok := r.segs[id]; ok {
		s = cached // another call built it meanwhile
	} else {
		r.segs[id] = s
	}
	r.mu.Unlock()
	return s, nil
}

// forget drops a cached segment whose file changed or disappeared.
func (r *Reader) forget(id uint64) {
	r.mu.Lock()
	delete(r.segs, id)
	r.plan = nil
	r.mu.Unlock()
}

// livePlan returns the kept plan when it still describes the sealed segments: the marker names
// the segment it was built for, no scan has finished since, and the oldest sealed segment is
// still there. Retention deletes oldest first (decision 65), so while the oldest is there the
// ones after it are too; a segment removed out of order is met when it is opened, as a gap
// (decision 101), and forget drops the plan then.
func (r *Reader) livePlan(tl tail, gen uint64) *sealedPlan {
	r.mu.Lock()
	p := r.plan
	r.mu.Unlock()
	if p == nil || p.seg != tl.Seg || p.gen != gen {
		return nil
	}
	if p.oldest != "" {
		if _, err := r.fs.Lstat(p.oldest); err != nil {
			return nil
		}
	}
	return p
}

// listSealed lists the segments below the marker's, in id order, building the Reader's
// record of each on first sight, and drops what the directory no longer holds.
func (r *Reader) listSealed(tl tail) ([]*readerSegment, error) {
	// Not listSegments: this runs on every call, and the extra id slice it builds costs more
	// allocations and bytes per view than iterating the names directly.
	names, err := r.fs.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}
	sealed := make([]*readerSegment, 0, len(names))
	for _, n := range names {
		if id, ok := parseSegmentName(n); ok && id < tl.Seg {
			s, err := r.segment(id)
			if err != nil {
				return nil, err
			}
			sealed = append(sealed, s)
		}
	}
	r.prune(sealed)
	return sealed, nil
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

// firstFrameChunk is the read size of a first-frame probe, of a sealed segment or the active
// one: a frame's header and a small payload, not the 64 KiB a scan starts with.
const firstFrameChunk = 4 << 10

// maxViewAttempts bounds how often view starts over when the segment its marker names is gone
// because the marker moved on meanwhile.
const maxViewAttempts = 5

// staleMarkerError is buildView's report that the marker it read named a segment that is gone
// and that the marker on disk now names another one: the writer rotated, and retention deleted
// the old active segment, between the read of the marker and the open of the segment.
type staleMarkerError struct{ err error }

func (e *staleMarkerError) Error() string { return e.err.Error() }
func (e *staleMarkerError) Unwrap() error { return e.err }

// view builds a view of the log, starting over with the newer marker when the one it read was
// overtaken by a rotation and a deletion (a live log does not hold still while a view is built).
// A marker that still names the missing segment is not stale: the error is returned as it is.
func (r *Reader) view() (*segView, func(), error) {
	for attempt := 1; ; attempt++ {
		v, release, err := r.buildView()
		var stale *staleMarkerError
		if !errors.As(err, &stale) {
			return v, release, err
		}
		if attempt == maxViewAttempts {
			return nil, nil, stale.err
		}
	}
}

// buildView reads the marker, lists the directory unless the last view's references are still
// good (livePlan), and builds the segment ranges from first frames (decision 47): a sealed
// segment's range ends where the next one starts.
func (r *Reader) buildView() (*segView, func(), error) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return nil, nil, ErrClosed
	}
	gen := r.scanGen.Load() // before anything below reads a scan's result
	tl, err := r.readTail()
	if err != nil {
		return nil, nil, err
	}
	plan := r.livePlan(tl, gen)
	var sealed []*readerSegment
	if plan == nil {
		if sealed, err = r.listSealed(tl); err != nil {
			return nil, nil, err
		}
	}
	// The active segment: its first frame, if the marker covers one.
	var activeFirst uint64
	var activeAt int64
	var activeFile fsx.File
	if tl.Off > 0 {
		f, size, err := openSegment(r.fs, r.path(segmentName(tl.Seg)))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// The segment may have been sealed and deleted since the marker was read. If
				// the marker has moved to another segment, this view is stale: start over.
				if now, terr := r.readTail(); terr == nil && now.Seg != tl.Seg {
					return nil, nil, &staleMarkerError{err}
				}
			}
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
		if ferr != nil && !isFrameError(ferr) {
			_ = f.Close()
			return nil, nil, ferr // I/O: not corruption (decision 79)
		}
		if ferr != nil || rec.Seq == 0 || rec.Seq > tl.Seq {
			_ = f.Close()
			return nil, nil, fmt.Errorf("%w: the active segment's first frame does not decode below the marker", ErrCorrupt)
		}
		activeFirst, activeAt, activeFile = rec.Seq, rec.At, f
	}
	if plan != nil && (plan.activeFirst != activeFirst || (activeFirst == 0 && plan.seq != tl.Seq)) {
		// The plan was built for another active segment or marker sequence (a writer crash
		// renumbered it, decision 63): list the directory after all.
		plan = nil
		if sealed, err = r.listSealed(tl); err != nil {
			if activeFile != nil {
				_ = activeFile.Close()
			}
			return nil, nil, err
		}
	}
	if plan != nil {
		v := &segView{head: tl.Seq, holes: slices.Clone(plan.holes)}
		v.segs = make([]*segRef, len(plan.refs), len(plan.refs)+1)
		copy(v.segs, plan.refs)
		return r.finishView(v, tl, activeFile, activeFirst, activeAt, new(segRef))
	}
	v := &segView{head: tl.Seq}
	// First the ranges of the good segments, then one contiguity check over them and the
	// active segment, then the segment references.
	type cand struct {
		s       *readerSegment
		last    uint64
		bounded bool // last was derived from the next segment's first sequence, not the marker
		index   []anchor
	}
	cands := make([]cand, 0, len(sealed))
	rs := make([]seqRange, 0, len(sealed)+1)
	leadBad := false   // a bad segment before the first kept one
	transient := false // a segment that could not be opened for an I/O reason is not cached (decision 94), so a plan holding it would hide that the next view looks again
	for i, s := range sealed {
		if s.bad != nil {
			var se *SegmentError
			transient = transient || !errors.As(s.bad, &se)
			leadBad = leadBad || len(cands) == 0
			continue
		}
		// The range ends at the next good segment's first sequence, or the active one's.
		var last uint64
		bounded := false
		for _, n := range sealed[i+1:] {
			if n.bad == nil {
				last, bounded = n.first-1, true
				break
			}
		}
		if !bounded {
			if activeFirst > 0 {
				last, bounded = activeFirst-1, true
			} else {
				last = tl.Seq
			}
		}
		if last < s.first {
			// An empty range: the marker predates it (a stale marker after a rotation crash), or
			// (bounded) the next good segment starts at or below this one's first sequence (a
			// duplicate or overlap). Either way the segment is left out of the view, never scanned,
			// so contiguity is not given an overlap to report: lost records show as a gap.
			continue
		}
		// One load: a scan may finish in another call while this loop runs.
		var index []anchor
		if res := s.result.Load(); res != nil && res.index != nil {
			index = res.index
			if res.last < last {
				last = res.last // scanned already: its real end is known; the next segment or the marker reports the hole
			} else if res.last > last && bounded {
				last = res.last // it runs into the next segment: contiguity reports the overlap, as Open does
			}
		}
		cands = append(cands, cand{s, last, bounded, index})
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
		s, last, bounded := c.s, c.last, c.bounded
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
				scanned := s.result.Load() != nil
				if err := s.scan(r.fs); err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						r.forget(s.id) // retention deleted it: the next view lists without it
					}
					return nil, 0, err
				}
				res := s.result.Load()
				if !scanned && res.last != last {
					// The segment ends elsewhere than the plan has it (a hole, or an overlap):
					// a plan built before this scan does not say so.
					r.scanGen.Add(1)
				}
				if bounded && res.last > last {
					// Frames past where the next segment starts: the overlap Open refuses (decision 102).
					return nil, 0, fmt.Errorf("%w: segment %d holds sequence %d, inside the next segment (starts at %d)", ErrCorrupt, s.id, res.last, last+1)
				}
				return res.index, res.last, nil
			}
		}
		v.segs = append(v.segs, ref)
	}
	if activeFile == nil {
		if len(cands) > 0 && tl.Seq > cands[len(cands)-1].last {
			v.holes = append(v.holes, Hole{From: cands[len(cands)-1].last + 1, To: tl.Seq})
		} else if leadBad && len(cands) == 0 && tl.Seq > 0 {
			v.holes = append(v.holes, Hole{From: 1, To: tl.Seq})
		}
	}
	if !transient {
		// v.segs holds the sealed references only, as yet: the next view starts from them.
		p := &sealedPlan{seg: tl.Seg, activeFirst: activeFirst, seq: tl.Seq, gen: gen,
			refs: slices.Clone(v.segs), holes: slices.Clone(v.holes)}
		if len(sealed) > 0 {
			p.oldest = sealed[0].path
		}
		r.mu.Lock()
		r.plan = p
		r.mu.Unlock()
	}
	return r.finishView(v, tl, activeFile, activeFirst, activeAt, &refs[len(cands)])
}

// finishView adds the active segment's reference, if the marker covers a frame of it, to a view
// whose sealed references and holes are done, and returns it with the function that closes the
// active segment's file. ref is the storage for the active reference.
func (r *Reader) finishView(v *segView, tl tail, f fsx.File, first uint64, at int64, ref *segRef) (*segView, func(), error) {
	if f != nil {
		off := int64(tl.Off)
		*ref = segRef{id: tl.Seg, first: first, last: tl.Seq, at: at, active: true,
			open: func() (io.ReaderAt, int64, func(), error) { return f, off, func() {}, nil },
			indexTo: func(seq uint64) []anchor {
				return r.activeIndexTo(f, tl, first, at, seq)
			}}
		v.segs = append(v.segs, ref)
	}
	if len(v.segs) > 0 {
		v.first = v.segs[0].first
	} else {
		v.first = tl.Seq + 1
	}
	release := func() {
		if f != nil {
			_ = f.Close()
		}
	}
	return v, release, nil
}

// activeIndexTo returns the anchors of the active segment's verified prefix, extended first
// through the frame holding seq, so that a read starts at the last anchor at or below seq
// instead of at byte 0. The prefix is dropped when it no longer describes the file the view
// opened: another file, a file shorter than what was verified, or another first frame. It
// returns nil when there is nothing to start from; the read then walks from byte 0 as it
// always did. Only anchors that start below the view's marker offset are returned.
func (r *Reader) activeIndexTo(f fsx.File, tl tail, first uint64, firstAt int64, seq uint64) []anchor {
	info, err := f.Stat()
	if err != nil {
		return nil
	}
	off := int64(tl.Off)
	r.actMu.Lock()
	defer r.actMu.Unlock()
	a := r.act
	if a == nil || !fsx.SameFile(a.file, info) || info.Size() < a.end || a.first != first || a.firstAt != firstAt {
		a = &activeIndex{file: info, first: first, firstAt: firstAt}
		r.act = a
	}
	if a.last < seq && a.end < off {
		r.extendActive(a, f, tl, seq)
	}
	idx := a.index
	for len(idx) > 0 && idx[len(idx)-1].pos >= off {
		idx = idx[:len(idx)-1]
	}
	if len(idx) == 0 {
		return nil
	}
	return idx[:len(idx):len(idx)]
}

// extendActive verifies the frames after a.end, up to and including the one holding seq or up
// to the marker's offset, and records their anchors. A frame or I/O error ends the prefix where
// it stands and is not reported: the read's own walk from the last anchor meets the same bytes
// and reports them as it always did. An extension that contradicts the marker (a frame past
// its sequence, or a first frame the view did not see) is dropped with the whole prefix.
// Called with r.actMu held.
func (r *Reader) extendActive(a *activeIndex, f fsx.File, tl tail, seq uint64) {
	bufp := r.bufPool.Get().(*[]byte)
	w := &window{r: f, size: int64(tl.Off), buf: (*bufp)[:0], noIndex: true}
	before, next := len(a.index), a.next
	info, _ := walk(w, a.end, a.last, a.end > 0, func(rec Record, pos int64) bool {
		if pos >= next {
			a.index = append(a.index, anchor{seq: rec.Seq, pos: pos, at: rec.At})
			next = pos + indexStride
		}
		return rec.Seq < seq
	})
	*bufp = w.buf[:0]
	r.bufPool.Put(bufp)
	if info.frames == 0 {
		return
	}
	if info.last.Seq > tl.Seq || (a.end == 0 && a.index[0].seq != a.first) {
		a.index = a.index[:before]
		r.act = nil
		return
	}
	a.end, a.last, a.next = info.end, info.last.Seq, next
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
