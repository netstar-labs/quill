package quill

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"sync"

	"github.com/netstar-labs/quill/internal/fsx"
)

// This file verifies and recovers one segment (decision 39). The directory scan, hole
// detection and the truncation of a recovered tail are phase 2 and live here too.

// maxSegmentBytes is the largest segment file a reader opens (guarantee 7).
const maxSegmentBytes = 1536 << 20

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
}

// SegmentError describes damage in a segment file: the byte where the damage starts and the
// end of the last valid frame before it, which is what the manual procedure in the user
// guide needs.
type SegmentError struct {
	Path   string // set by the caller that knows the file
	Offset int64  // where the damage starts
	Intact int64  // the end of the last valid frame
	Reason string
}

func (e *SegmentError) Error() string {
	return fmt.Sprintf("quill: segment %s damaged at byte %d (intact to byte %d): %s", e.Path, e.Offset, e.Intact, e.Reason)
}

// Is makes a *SegmentError match ErrCorrupt.
func (e *SegmentError) Is(target error) bool { return target == ErrCorrupt }

// openSegmentFlag opens a segment file with the given open flag: never through a symbolic
// link, a regular file only (checked before the open, so a FIFO cannot block it, and
// after, so the open file is what was checked), and at most maxSegmentBytes (guarantee 7).
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
	if err == nil && info.Size() > maxSegmentBytes {
		err = &SegmentError{Path: path, Reason: fmt.Sprintf("%d bytes, over the %d limit", info.Size(), maxSegmentBytes)}
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
	bodyLen, n := binary.Uvarint(hdr)
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

// walk scans frames from start, calling fn for each one accepted. A frame is accepted when it
// decodes, its sequence is exactly one more than the previous accepted frame's (the first
// frame from start must be prev+1 when prev is known, and at least 1 otherwise), and fn
// returns true. It returns the info gathered and, when the scan stopped before size, the
// error at that point: errTornFrame, errBadFrame, or an I/O error.
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
		if (havePrev && rec.Seq != prev+1) || (!havePrev && rec.Seq == 0) {
			info.end = pos
			return info, errBadFrame
		}
		if info.frames == 0 {
			info.first = Record{Seq: rec.Seq, At: rec.At}
		}
		if pos >= nextAnchor {
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

// verifySegment checks a sealed segment whole: every byte is durable, so every frame must be
// intact and contiguous, the last must end exactly at size, and there must be at least one.
// Anything else is a *SegmentError (decision 39, decision 43) carrying path, which names the
// segment for the caller's message.
func verifySegment(r io.ReaderAt, size int64, path string) (segInfo, error) {
	w := &window{r: r, size: size, chunk: windowChunk}
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
// 39), then republishes the marker. The lock is held and creation is complete.
func (l *Log) load() error {
	data, err := readState(l.fs, l.path(formatName))
	if err != nil {
		return err
	}
	id, err := decodeFormat(data)
	if err != nil {
		if errors.Is(err, ErrFutureVersion) {
			return err
		}
		return fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	l.id = id
	data, err = readState(l.fs, l.path(tailName))
	if err != nil {
		return err
	}
	tl, err := decodeTail(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	ids, err := l.listSegments()
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
	var wg sync.WaitGroup
	sem := make(chan struct{}, max(runtime.GOMAXPROCS(0), 1))
	for i, id := range ids[:len(ids)-1] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			sealed[i] = l.loadSealed(id)
		}()
	}
	wg.Wait()

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
	if cut {
		err = f.Truncate(info.end)
		if err == nil {
			err = f.Sync()
		}
		if err != nil {
			_ = f.Close()
			return err
		}
	}
	a.f, a.durable, a.frames, a.index = f, info.end, info.frames, info.index
	a.nextAnchor = info.end
	if info.frames > 0 {
		a.first, a.at, a.lastAt = info.first.Seq, info.first.At, info.last.At
	}

	// Contiguity across segments: holes, overlaps, the head.
	var holes []Hole
	var prevLast uint64
	havePrev := false
	var pendingHole bool // a bad segment seen since the last good one
	for _, s := range sealed {
		if s.bad != nil {
			if l.loadErr == nil {
				l.loadErr = s.bad
			}
			pendingHole = true
			continue
		}
		if havePrev {
			if s.first <= prevLast {
				_ = f.Close()
				return fmt.Errorf("%w: segment %d starts at %d, inside the range of the one before (ends at %d)", ErrCorrupt, s.id, s.first, prevLast)
			}
			if s.first != prevLast+1 {
				holes = append(holes, Hole{From: prevLast + 1, To: s.first - 1})
			}
		} else if pendingHole {
			holes = append(holes, Hole{From: 1, To: s.first - 1}) // from "unknown": the first good segment's start bounds it
		}
		pendingHole = false
		prevLast, havePrev = s.last, true
	}
	head := tl.Seq
	hasFrames := havePrev || a.frames > 0
	if a.frames > 0 {
		if havePrev {
			if a.first <= prevLast {
				_ = f.Close()
				return fmt.Errorf("%w: the active segment starts at %d, inside the range of the one before (ends at %d)", ErrCorrupt, a.first, prevLast)
			}
			if a.first != prevLast+1 {
				holes = append(holes, Hole{From: prevLast + 1, To: a.first - 1})
			}
		} else if pendingHole {
			holes = append(holes, Hole{From: 1, To: a.first - 1})
		}
		head = info.last.Seq
	} else if havePrev {
		head = prevLast
		if pendingHole && tl.Seq > prevLast {
			holes = append(holes, Hole{From: prevLast + 1, To: tl.Seq})
			head = tl.Seq
		}
	} else if pendingHole {
		holes = append(holes, Hole{From: 1, To: tl.Seq})
	}
	if hasFrames && head < tl.Seq {
		_ = f.Close()
		return fmt.Errorf("%w: the last valid sequence is %d but %s says %d is durable", ErrCorrupt, head, tailName, tl.Seq)
	}
	if a.frames == 0 {
		a.first = head + 1
	}
	l.active, l.sealed, l.holes, l.head = a, sealed, holes, head
	l.nextSeq = head + 1
	l.maxSeg = max(tl.MaxSeg, newest)
	// Republish the marker from what was really on disk, and make it durable.
	l.marker = tail{} // force a write
	if err := l.publishLocked(); err != nil {
		_ = f.Close()
		return err
	}
	return l.fs.SyncDir(l.dir)
}

// listSegments returns the segment ids in the directory, ascending. Other names are ignored.
func (l *Log) listSegments() ([]uint64, error) {
	names, err := l.fs.ReadDir(l.dir)
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

// loadSealed verifies one sealed segment; a failure makes it a bad segment whose range is a
// hole, and the file is never modified.
func (l *Log) loadSealed(id uint64) *segment {
	s := &segment{id: id, path: l.path(segmentName(id))}
	f, size, err := openSegment(l.fs, s.path)
	if err != nil {
		s.bad = err
		return s
	}
	defer f.Close()
	info, err := verifySegment(f, size, s.path)
	if err != nil {
		s.bad = err
		return s
	}
	s.first, s.last, s.size, s.index, s.at, s.lastAt = info.first.Seq, info.last.Seq, size, info.index, info.first.At, info.last.At
	return s
}

// openActive opens the newest segment read-write with the checks of openSegment.
func openActive(fsys fsx.FS, path string) (fsx.File, int64, error) {
	return openSegmentFlag(fsys, path, fsx.O_RDWR)
}
