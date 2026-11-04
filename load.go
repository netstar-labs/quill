package quill

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"runtime"
	"slices"
	"sync"

	"github.com/netstar-labs/quill/internal/fsx"
)

// This file verifies and recovers one segment (decision 39). The directory scan and the
// truncation of a recovered tail live here too; the holes and the first overlap are built by
// contiguity in holes.go (decision 102), which Log.load calls.

// maxSegmentFileBytes is the largest segment file a reader opens (guarantee 7).
const maxSegmentFileBytes = 1536 << 20

// indexStride is the byte distance between sparse-index anchors within a segment.
const indexStride = 64 << 10

// anchor is one sparse-index entry: the record at a byte position.
type anchor struct {
	seq uint64
	pos int64
	at  int64
}

// segInfo is what a scan learns about a segment.
type segInfo struct {
	first  Record // Seq and At of the first frame (no payload)
	last   Record // Seq and At of the last valid frame
	frames int
	end    int64    // the byte after the last valid frame
	index  []anchor // one anchor per indexStride of bytes, the first frame always
	stop   error    // recoverSegment: why the forward parse stopped short of the file's end (a frame error), else nil
}

// SegmentError describes damage in a segment file: the byte where the damage starts and the
// end of the last valid frame before it, which is what the manual procedure in the user
// guide needs.
type SegmentError struct {
	Path   string // set by the caller that knows the file
	Offset int64  // where the damage starts
	Intact int64  // the end of the last valid frame
	Reason string
	// oversize marks a file over maxSegmentFileBytes, which is refused whole, not damaged at a byte.
	oversize bool
}

// Error names the segment, where the damage starts and where the intact bytes end; a file over
// the size limit is refused unread, so no byte is named.
func (e *SegmentError) Error() string {
	if e.oversize {
		return fmt.Sprintf("quill: segment %s refused without being read: %s", e.Path, e.Reason)
	}
	return fmt.Sprintf("quill: segment %s damaged at byte %d (intact to byte %d): %s", e.Path, e.Offset, e.Intact, e.Reason)
}

// Is makes a *SegmentError match ErrCorrupt.
func (e *SegmentError) Is(target error) bool { return target == ErrCorrupt }

// openSegmentFlag opens a segment file with the given open flag: never through a symbolic
// link, a regular file only (checked before the open, so a FIFO cannot block it, and
// after, so the open file is what was checked), and at most maxSegmentFileBytes (guarantee 7).
func openSegmentFlag(fsys fsx.FS, path string, flag int) (fsx.File, int64, error) {
	info, err := fsys.Lstat(path)
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, &SegmentError{Path: path, Reason: "not a regular file"}
	}
	f, err := fsys.OpenFile(path, flag, 0)
	if err != nil {
		return nil, 0, err
	}
	info, err = f.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = &SegmentError{Path: path, Reason: "not a regular file"}
	}
	if err == nil && info.Size() > maxSegmentFileBytes {
		err = &SegmentError{Path: path, Reason: fmt.Sprintf("%d bytes, over the %d limit", info.Size(), maxSegmentFileBytes), oversize: true}
	}
	if err != nil {
		_ = f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}

// openSegment opens a segment file for reading with the checks of openSegmentFlag.
func openSegment(fsys fsx.FS, path string) (fsx.File, int64, error) {
	return openSegmentFlag(fsys, path, fsx.O_RDONLY)
}

// window reads a segment through an io.ReaderAt in chunks, handing out slices of frames.
// A read starts with a small chunk and doubles it on each further read, so a Read of a
// hundred records costs one small pread while a scan of a whole segment soon reads a
// megabyte at a time (decision 91). verifySegment and recoverSegment start large.
type window struct {
	r     io.ReaderAt
	size  int64
	buf   []byte
	off   int64 // position of buf[0] in the file
	chunk int64 // the next read's size; 0 means readChunk
	// noIndex stops walk recording anchors, for a caller that discards the segInfo.
	noIndex bool
}

const (
	windowChunk = 1 << 20 // the largest chunk, and the first for a whole-segment scan
	readChunk   = 64 << 10
)

// bytes returns n bytes at pos, n at most size-pos, reading from the file when they are not
// buffered. The slice is valid until the next call.
func (w *window) bytes(pos int64, n int) ([]byte, error) {
	if pos >= w.off && pos+int64(n) <= w.off+int64(len(w.buf)) {
		return w.buf[pos-w.off : pos-w.off+int64(n)], nil
	}
	if w.chunk == 0 {
		w.chunk = readChunk
	}
	want := max(int64(n), w.chunk)
	w.chunk = min(w.chunk*2, windowChunk)
	if pos+want > w.size {
		want = w.size - pos
	}
	if cap(w.buf) < int(want) {
		w.buf = make([]byte, want)
	}
	w.buf = w.buf[:want]
	got, err := w.r.ReadAt(w.buf, pos)
	if err != nil && (err != io.EOF || int64(got) < want) {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF // the file is shorter than its stated size
		}
		w.buf = w.buf[:0]
		return nil, err
	}
	w.off = pos
	return w.buf[:n], nil
}

// frameAt decodes the frame at pos. Payload aliases the window.
func (w *window) frameAt(pos int64) (Record, int, error) {
	avail := int(min(w.size-pos, int64(maxVarint)))
	hdr, err := w.bytes(pos, avail)
	if err != nil {
		return Record{}, 0, err
	}
	bodyLen, n := uvarint(hdr)
	switch {
	case n < 0, n > 0 && bodyLen > maxFrameBody:
		return Record{}, 0, errBadFrame
	case n == 0 && avail >= maxVarint:
		return Record{}, 0, errBadFrame
	case n == 0:
		return Record{}, 0, errTornFrame
	}
	total := n + int(bodyLen) + 4
	if pos+int64(total) > w.size {
		return Record{}, 0, errTornFrame
	}
	data, err := w.bytes(pos, total)
	if err != nil {
		return Record{}, 0, err
	}
	return decodeFrame(data)
}

// walk scans frames from start. A frame is accepted when it decodes and its sequence is exactly
// one more than the previous accepted frame's (the first frame from start must be prev+1 when
// prev is known, and at least 1 otherwise; nothing follows a frame numbered MaxUint64, where
// prev+1 would wrap to 0). Each accepted frame is counted into the returned info (frames, last,
// end) and only then passed to fn, so a frame for which fn returns false is still counted. If fn
// returns false the walk stops there with a nil error. Otherwise it ends at size (nil error) or
// at the first frame that is not accepted, with info.end at that frame's start and the error
// errTornFrame, errBadFrame, or an I/O error.
func walk(w *window, start int64, prev uint64, havePrev bool, fn func(rec Record, pos int64) bool) (segInfo, error) {
	info := segInfo{end: start}
	nextAnchor := start
	pos := start
	for pos < w.size {
		rec, total, err := w.frameAt(pos)
		if err != nil {
			info.end = pos
			return info, err
		}
		if (havePrev && (prev == math.MaxUint64 || rec.Seq != prev+1)) || (!havePrev && rec.Seq == 0) {
			info.end = pos
			return info, errBadFrame
		}
		if info.frames == 0 {
			info.first = Record{Seq: rec.Seq, At: rec.At}
		}
		if !w.noIndex && pos >= nextAnchor {
			info.index = append(info.index, anchor{seq: rec.Seq, pos: pos, at: rec.At})
			nextAnchor = pos + indexStride
		}
		info.last = Record{Seq: rec.Seq, At: rec.At}
		info.frames++
		prev, havePrev = rec.Seq, true
		pos += int64(total)
		info.end = pos
		if fn != nil && !fn(rec, pos-int64(total)) {
			return info, nil
		}
	}
	return info, nil
}

// verifyBufs holds the read buffers of verifySegment and verifyBounded, which run once per
// segment and, on Open and Check, in several goroutines at once: a log of many mid-size
// segments would otherwise allocate and zero a fresh buffer for each. A buffer holds no
// state between uses (the segInfo a walk returns copies what it keeps), so any of them
// serves any segment. A zero-capacity buffer grows once to what the first segment needs,
// so a small log does not pay for a megabyte.
var verifyBufs = sync.Pool{New: func() any { return new([]byte) }}

// verifyWindow returns a whole-segment window over the first size bytes of r, reading into a
// pooled buffer, and the function that returns the buffer to the pool. The window, and
// anything it handed out, must not be used after release is called.
func verifyWindow(r io.ReaderAt, size int64) (w *window, release func()) {
	bufp := verifyBufs.Get().(*[]byte)
	w = &window{r: r, size: size, buf: (*bufp)[:0], chunk: windowChunk}
	return w, func() {
		*bufp = w.buf[:0]
		verifyBufs.Put(bufp)
	}
}

// verifySegment checks a sealed segment whole: every byte is durable, so every frame must be
// intact and contiguous, the last must end exactly at size, and there must be at least one.
// Anything else is a *SegmentError (decision 39, decision 43) carrying path, which names the
// segment for the caller's message.
func verifySegment(r io.ReaderAt, size int64, path string) (segInfo, error) {
	w, release := verifyWindow(r, size)
	defer release()
	info, err := walk(w, 0, 0, false, nil)
	if err != nil {
		if !isFrameError(err) {
			return info, err // I/O
		}
		return info, &SegmentError{Path: path, Offset: info.end, Intact: info.end, Reason: reason(err)}
	}
	if info.frames == 0 {
		return info, &SegmentError{Path: path, Reason: "a sealed segment with no frames"}
	}
	return info, nil
}

// recoverSegment scans the newest segment. The bytes below durable were fsynced before the
// marker named them, so they must verify whole and end on a frame boundary exactly at durable;
// damage there is a *SegmentError (corruption). From durable it parses forward, accepting
// each frame that is intact and contiguous, and stops at the first that is not: the returned
// info.end is where the caller truncates, and cut reports whether anything was cut. Nothing
// after the first bad frame is trusted, whatever it contains. A *SegmentError carries path.
func recoverSegment(r io.ReaderAt, size int64, durable int64, path string) (info segInfo, cut bool, err error) {
	if durable > size {
		return segInfo{}, false, &SegmentError{Path: path, Offset: size, Intact: size, Reason: fmt.Sprintf("the file holds %d bytes but the marker says %d are durable", size, durable)}
	}
	w := &window{r: r, size: durable, chunk: windowChunk}
	info, err = walk(w, 0, 0, false, nil)
	if err != nil {
		if !isFrameError(err) {
			return info, false, err // I/O
		}
		return info, false, &SegmentError{Path: path, Offset: info.end, Intact: info.end, Reason: reason(err) + " below the durable offset"}
	}
	w.size = size
	tail, err := walk(w, durable, info.last.Seq, info.frames > 0, nil)
	if err != nil && !isFrameError(err) {
		return info, false, err // I/O
	}
	cut = tail.end < size
	if tail.frames > 0 {
		if info.frames == 0 {
			info.first = tail.first
		}
		info.last = tail.last
		info.frames += tail.frames
		info.index = append(info.index, tail.index...)
	}
	info.end = tail.end
	info.stop = err
	return info, cut, nil
}

func isFrameError(err error) bool {
	return errors.Is(err, errTornFrame) || errors.Is(err, errBadFrame)
}

func reason(err error) string {
	switch {
	case errors.Is(err, errTornFrame):
		return "torn frame"
	case errors.Is(err, errBadFrame):
		return "bad frame"
	}
	return err.Error()
}

// load reads the state files, scans the directory and recovers the newest segment (decision
// 39), then republishes the marker if recovery moved it. The lock is held and creation is complete.
func (l *Log) load() error {
	id, err := loadFormat(l.fs, l.path(formatName))
	if err != nil {
		return err
	}
	l.id = id
	data, err := readState(l.fs, l.path(tailName))
	if err != nil {
		return err
	}
	tl, err := decodeTail(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	ids, err := listSegments(l.fs, l.dir)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("%w: no segment files", ErrCorrupt)
	}
	newest := ids[len(ids)-1]
	if tl.Seg > newest {
		return fmt.Errorf("%w: %s names segment %d, the newest on disk is %d", ErrCorrupt, tailName, tl.Seg, newest)
	}

	// Sealed segments are verified whole, in parallel.
	sealed := make([]*segment, len(ids)-1)
	readErrs := make([]error, len(ids)-1)
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, id := range ids[:len(ids)-1] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			sealed[i], readErrs[i] = l.loadSealed(id)
		}()
	}
	wg.Wait()
	// An I/O error while verifying a segment says nothing about its bytes: it fails the open, as
	// it does for the newest segment, and never becomes a hole that a resume would skip (decision 79).
	for _, err := range readErrs {
		if err != nil {
			return err
		}
	}

	// The newest segment is recovered by parsing forward from the marker.
	a := &activeSegment{id: newest, path: l.path(segmentName(newest))}
	f, size, err := openActive(l.fs, a.path)
	if err != nil {
		return err
	}
	durable := int64(0)
	if tl.Seg == newest {
		durable = int64(tl.Off)
	}
	info, cut, err := recoverSegment(f, size, durable, a.path)
	if err != nil {
		_ = f.Close()
		return err
	}
	a.f, a.durable, a.frames, a.index = f, info.end, info.frames, info.index
	a.nextAnchor = info.end
	if info.frames > 0 {
		a.first, a.at, a.lastAt = info.first.Seq, info.first.At, info.last.At
	}

	// Contiguity across segments: holes, overlaps, the head.
	var good []seqRange
	var leadBad, trailBad bool // a bad segment before the first good one; since the last good one
	for _, s := range sealed {
		if s.bad != nil {
			if l.loadErr == nil {
				l.loadErr = s.bad
			}
			trailBad = true
			leadBad = leadBad || len(good) == 0
			continue
		}
		good = append(good, seqRange{id: s.id, first: s.first, last: s.last})
		trailBad = false
	}
	rs := good
	if a.frames > 0 {
		rs = append(rs, seqRange{id: a.id, first: a.first, last: info.last.Seq})
	}
	holes, err := contiguity(rs, leadBad)
	if err != nil {
		_ = f.Close()
		return err
	}
	head := tl.Seq
	hasFrames := len(good) > 0 || a.frames > 0
	if a.frames > 0 {
		head = info.last.Seq
	} else if len(good) > 0 {
		prevLast := good[len(good)-1].last
		head = prevLast
		if trailBad && tl.Seq > prevLast {
			holes = append(holes, Hole{From: prevLast + 1, To: tl.Seq})
			head = tl.Seq
		}
	} else if leadBad && tl.Seq > 0 { // a marker at 0 promises nothing: no empty hole
		holes = append(holes, Hole{From: 1, To: tl.Seq})
	}
	if hasFrames && head < tl.Seq {
		_ = f.Close()
		return fmt.Errorf("%w: the last valid sequence is %d but %s says %d is durable", ErrCorrupt, head, tailName, tl.Seq)
	}
	if head == math.MaxUint64 {
		_ = f.Close()
		return fmt.Errorf("%w: the log ends at sequence %d, which is never assigned (the ceiling is %d)", ErrCorrupt, uint64(math.MaxUint64), uint64(math.MaxUint64-1))
	}
	// Every check that can refuse the log has passed: only now is the untrusted tail cut, so
	// an Open that fails with ErrCorrupt leaves the newest segment as it found it.
	if cut {
		err = f.Truncate(info.end)
		if err == nil {
			err = f.Sync()
		}
		if err != nil {
			_ = f.Close()
			return err
		}
		// Say what was cut. A torn frame is the routine crash tail; a complete-length frame that
		// failed its checks is damage, and may have been an acknowledged record (decision 39).
		kind := "damaged"
		if errors.Is(info.stop, errTornFrame) {
			kind = "torn"
		}
		l.logf(slog.LevelWarn, "cut the newest segment past the marker", "segment", a.path,
			"offset", info.end, "bytes", size-info.end, "kind", kind)
	}
	if a.frames == 0 {
		a.first = head + 1
	}
	l.active, l.sealed, l.holes, l.head = a, sealed, holes, head
	l.nextSeq = head + 1
	l.maxSeg = max(tl.MaxSeg, newest)
	// The marker on disk is the one just decoded. When it already names what was really found
	// there is nothing to republish, and Open writes no file, so it needs no free block on a
	// full filesystem (L3b-8). Only a recovery that moved the head, the segment, the offset or
	// the highest segment id makes publishLocked write. The marker age counts from Open either
	// way: an empty log's marker is the zero value, equal to what is found, and a write was
	// what used to set markerAt (MarkerAge read 0).
	l.marker, l.markerAt = tl, l.cfg.clock()
	// Config.Perm is the marker's mode, and a restart may give a different one. The write that
	// used to apply it at every Open is gone, so change the mode of the file as it is: an
	// fchmod needs no free block, where a rewrite would. A Reader takes its own mode from this
	// file, so a log that stays idle must not keep the old one. A mode that cannot be changed
	// (not the owner) does not fail the Open, as with quill.lock: the data is intact.
	if err := setStateMode(l.fs, l.tailPath, l.cfg.Perm); err != nil {
		l.logf(slog.LevelWarn, "quill.tail could not be given the mode in Config.Perm", "path", l.tailPath, "perm", l.cfg.Perm, "error", err)
	}
	// From here l.active owns the descriptor (publishLocked may even have rotated it): a
	// failure is returned with it open, and Open closes whatever l.active holds.
	if err := l.publishLocked(); err != nil {
		if l.failed != nil {
			return l.failed
		}
		// A marker that could not be written, a segment that could not be created: the data is
		// durable and the marker is a lower bound (decision 39), so the log opens as it would
		// after the same failure in a flush. pendingPublish is set, Err() and the logger report
		// it, and the flusher, Close and every prune pass retry it (decision 60 d).
		l.openUnpublished = l.bg[bgPublish] != nil
		return nil
	}
	return l.fs.SyncDir(l.dir)
}

// listSegments returns the segment ids in the directory, ascending. Other names are ignored.
func listSegments(fsys fsx.FS, dir string) ([]uint64, error) {
	names, err := fsys.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var ids []uint64
	for _, n := range names {
		if id, ok := parseSegmentName(n); ok {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

// loadSealed verifies one sealed segment; damage, or a file that cannot be opened, makes it a bad
// segment whose range is a hole, and the file is never modified. A bad segment still occupies
// its bytes, so its size is recorded whenever it can be learned (decision 98): from the open, or
// from an Lstat when the open fails. An I/O error while reading the segment is returned as the
// second result and not recorded: the caller fails the open with it (decision 79).
func (l *Log) loadSealed(id uint64) (*segment, error) {
	s := &segment{id: id, path: l.path(segmentName(id))}
	f, size, err := openSegment(l.fs, s.path)
	if err != nil {
		s.bad = err
		if info, lerr := l.fs.Lstat(s.path); lerr == nil {
			s.size = info.Size()
		}
		return s, nil
	}
	defer f.Close()
	s.size = size
	info, err := verifySegment(f, size, s.path)
	if err != nil {
		var se *SegmentError
		if !errors.As(err, &se) {
			return nil, err
		}
		s.bad = err
		return s, nil
	}
	s.first, s.last, s.index, s.at, s.lastAt = info.first.Seq, info.last.Seq, info.index, info.first.At, info.last.At
	return s, nil
}

// openActive opens the newest segment read-write with the checks of openSegment.
func openActive(fsys fsx.FS, path string) (fsx.File, int64, error) {
	return openSegmentFlag(fsys, path, fsx.O_RDWR)
}
