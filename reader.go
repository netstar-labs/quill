package quill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sync"
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
	// From one full scan, built when the segment is first read (decision 63).
	index   []anchor
	last    uint64
	frames  int
	lastAt  int64
	scanErr error
	once    sync.Once
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

// Read, ReadWait, Scan and ScanAll are the Log's, bounded by the marker.
func (r *Reader) Read(offset uint64, limit int) ([]Record, error) { return readFrom(r, offset, limit) }
func (r *Reader) ReadWait(ctx context.Context, offset uint64, limit int) ([]Record, error) {
	return readWait(ctx, r, offset, limit)
}
func (r *Reader) Scan(offset uint64, limit int, fn func(Record) bool) error {
	return scanFrom(r, offset, limit, fn)
}
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
// file gone or changed in size, and rebuilt on the next view (decision 63).
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

// scan builds a sealed segment's index by one full scan, once.
func (s *readerSegment) scan(fsys fsx.FS) error {
	s.once.Do(func() {
		f, size, err := openSegment(fsys, s.path)
		if err != nil {
			s.scanErr = err
			return
		}
		defer f.Close()
		info, err := verifySegment(f, size, s.path)
		if err != nil {
			s.scanErr = err
			return
		}
		s.index, s.last, s.frames, s.lastAt = info.index, info.last.Seq, info.frames, info.last.At
	})
	return s.scanErr
}

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
	names, err := r.fs.ReadDir(r.dir)
	if err != nil {
		return nil, nil, err
	}
	var sealed []*readerSegment
	for _, n := range names {
		if id, ok := parseSegmentName(n); ok && id < tl.Seg {
			sealed = append(sealed, r.segment(id))
		}
	}
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
		w := &window{r: f, size: int64(tl.Off)}
		rec, _, ferr := w.frameAt(0)
		if ferr != nil || rec.Seq == 0 || rec.Seq > tl.Seq {
			_ = f.Close()
			return nil, nil, fmt.Errorf("%w: the active segment's first frame does not decode below the marker", ErrCorrupt)
		}
		activeFirst, activeAt, activeFile = rec.Seq, rec.At, f
	}
	v := &segView{head: tl.Seq}
	var prevLast uint64
	havePrev, pendingHole := false, false
	for i, s := range sealed {
		if s.bad != nil {
			pendingHole = true
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
		if s.index != nil && s.last < last {
			last = s.last // scanned already: its real end is known; the next segment or the marker reports the hole
		}
		if havePrev && s.first <= prevLast {
			if activeFile != nil {
				_ = activeFile.Close()
			}
			return nil, nil, fmt.Errorf("%w: segment %d starts at %d, inside the one before (ends at %d)", ErrCorrupt, s.id, s.first, prevLast)
		}
		if (havePrev && s.first != prevLast+1) || (!havePrev && pendingHole) {
			from := uint64(1)
			if havePrev {
				from = prevLast + 1
			}
			v.holes = append(v.holes, Hole{From: from, To: s.first - 1})
		}
		pendingHole = false
		prevLast, havePrev = last, true
		s := s
		v.segs = append(v.segs, &segRef{id: s.id, first: s.first, last: last, at: s.at, index: s.index,
			open: func() (io.ReaderAt, int64, func(), error) {
				f, size, err := openSegment(r.fs, s.path)
				if err != nil {
					r.forget(s.id)
					return nil, 0, nil, err
				}
				if size != s.size {
					_ = f.Close()
					r.forget(s.id)
					return nil, 0, nil, fmt.Errorf("%w: segment %d changed size from %d to %d", ErrCorrupt, s.id, s.size, size)
				}
				return f, size, func() { _ = f.Close() }, nil
			},
			ensureIndex: func() ([]anchor, uint64, error) {
				if err := s.scan(r.fs); err != nil {
					return nil, 0, err
				}
				return s.index, s.last, nil
			}})
	}
	if activeFile != nil {
		if (havePrev && activeFirst != prevLast+1) || (!havePrev && pendingHole) {
			from := uint64(1)
			if havePrev {
				from = prevLast + 1
			}
			if activeFirst > from {
				v.holes = append(v.holes, Hole{From: from, To: activeFirst - 1})
			}
		}
		if havePrev && activeFirst <= prevLast {
			_ = activeFile.Close()
			return nil, nil, fmt.Errorf("%w: the active segment starts at %d, inside the one before (ends at %d)", ErrCorrupt, activeFirst, prevLast)
		}
		off := int64(tl.Off)
		v.segs = append(v.segs, &segRef{id: tl.Seg, first: activeFirst, last: tl.Seq, at: activeAt, active: true,
			open: func() (io.ReaderAt, int64, func(), error) { return activeFile, off, func() {}, nil }})
	} else if havePrev && tl.Seq > prevLast {
		v.holes = append(v.holes, Hole{From: prevLast + 1, To: tl.Seq})
	} else if pendingHole && !havePrev && tl.Seq > 0 {
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
