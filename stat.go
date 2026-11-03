package quill

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Stats describes a log.
type Stats struct {
	Head     uint64 // the last durable sequence
	First    uint64 // the first retained sequence; Head + 1 when nothing is retained
	Segments int    // segment files, the active one included
	Bytes    int64  // the sum of the segment files' sizes
	Records  uint64 // Head − First + 1, minus the records inside holes
	Holes    []Hole
	// Dropped counts segments deleted while some consumer's offset was below them, since
	// the writer opened the log. A cap that fires at a segment seal judges "below them" by the
	// consumer offsets of the last prune pass (or of Open, before the first), so a consumer that
	// moved or was created since can be missed or counted late by up to PruneInterval. Only
	// Log.Stat knows it; it is not persisted, so it is 0 after a restart. Each such deletion
	// is also logged at Warn naming the consumer, and quill consumers shows a consumer below
	// First without it.
	Dropped uint64
	// StaleConsumers are protected consumers that stopped protecting segments under
	// ConsumerTTL at the last prune pass. Only Log.Stat knows them.
	StaleConsumers []string
	// MarkerAge is how long ago quill.tail was last written.
	MarkerAge time.Duration
	// DiskFull is true while MinFreeBytes cannot be met and nothing is left to delete: every
	// append fails with an error wrapping ENOSPC until a later pass finds room. Err() reports
	// the same state. Only Log.Stat knows it.
	DiskFull bool
	// FreeBytesAtLastPass is the free space statfs reported at the last pass that could read
	// it, before that pass counted the segments it deleted; 0 when MinFreeBytes is off or no
	// pass has read it yet. Only Log.Stat knows it.
	FreeBytesAtLastPass uint64
	// BufferedRecords and BufferedBytes are what Append has acknowledged and no flush has made
	// durable yet: they are held in memory, invisible to readers, and lost if the process dies
	// (BufferedBytes is their encoded size). While flushes fail they grow only to MaxBufferBytes,
	// except that an append (a record or a batch) into an empty buffer is always taken, up to
	// MaxPayloadBytes for a record or MaxBatchBytes for a batch, so they can pass it by that one
	// append. Only Log.Stat knows them; a Reader reports 0.
	BufferedRecords int
	BufferedBytes   int
	// FlushFailing is true while the last flush that tried to write failed and the records
	// stay buffered for the next try; it goes false when a flush writes. Err() carries the
	// error. A log that has failed (Err() wraps ErrFailed: a data fsync error, or a truncate that
	// could not undo a failed write) does not retry, so FlushFailing stays false while the
	// records it holds are never written: alert on Err() wrapping ErrFailed as well. Only
	// Log.Stat knows it.
	FlushFailing bool
	// LastFlushOK is when the last flush that completed succeeded: it wrote its records and
	// published the marker, or it landed a marker publish that had failed (nothing new written),
	// or when the writer opened the log if none has since. It is the time of the
	// last successful write or publish, not a measure of how long anything has waited: a healthy log with
	// nothing to flush keeps an old LastFlushOK, and one Append after an hour of quiet reports
	// BufferedRecords of 1, FlushFailing false and a LastFlushOK an hour old, until the flusher
	// writes it (within FlushInterval). So never alert on time.Since(LastFlushOK) alone. Alert
	// on FlushFailing, or on BufferedRecords above zero together with FlushFailing, which is
	// acknowledged data that is not durable and not being made so, and on Err() wrapping
	// ErrFailed, where the log has stopped trying; read LastFlushOK then to
	// say how long the log has been unable to write. Only Log.Stat knows it; a Reader reports
	// the zero time.
	LastFlushOK time.Time
}

func (s *Stats) count(v *segView) {
	s.Head, s.First, s.Holes = v.head, v.first, v.holes
	if v.head >= v.first {
		s.Records = v.head - v.first + 1
		for _, h := range v.holes {
			if h.From >= v.first && h.To <= v.head {
				s.Records -= h.To - h.From + 1
			}
		}
	}
}

// Stat returns the log's statistics from the writer's memory.
func (l *Log) Stat() (Stats, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return Stats{}, ErrClosed
	}
	var s Stats
	s.count(&segView{head: l.head, first: l.firstRetainedLocked(), holes: l.holes})
	s.Segments = len(l.sealed) + 1
	for _, seg := range l.sealed {
		s.Bytes += seg.size
	}
	s.Bytes += l.active.durable
	s.Dropped = l.dropped
	s.StaleConsumers = append([]string(nil), l.stale...)
	s.DiskFull, s.FreeBytesAtLastPass = l.diskFull, l.freeAtPass
	s.BufferedRecords, s.BufferedBytes = l.bufCount, len(l.buf)
	s.FlushFailing = l.flushErr != nil
	s.LastFlushOK = l.lastFlushOK
	if !l.markerAt.IsZero() {
		s.MarkerAge = l.cfg.clock().Sub(l.markerAt)
	}
	return s, nil
}

// Stat returns what a reader can know: counts from the marker and the directory.
func (r *Reader) Stat() (Stats, error) {
	v, release, err := r.view()
	if err != nil {
		return Stats{}, err
	}
	release()
	var s Stats
	s.count(v)
	ids, err := listSegments(r.fs, r.dir)
	if err != nil {
		return Stats{}, err
	}
	for _, id := range ids {
		info, err := r.fs.Lstat(r.path(segmentName(id)))
		if err != nil {
			continue
		}
		s.Segments++
		s.Bytes += info.Size()
	}
	if age, err := r.MarkerAge(); err == nil {
		s.MarkerAge = age
	}
	return s, nil
}

// Problem is one finding of Check.
type Problem struct {
	Kind   string // "segment", "hole", "overlap", "consumer", "policy", "clock"
	Path   string
	Offset int64 // for a segment: where the damage starts
	Intact int64 // for a segment: the end of the last valid frame
	Detail string
	// refused marks a segment file refused whole, unread (over the size limit): intact as far
	// as anyone knows, so String names no damaged byte.
	refused bool
	// Fatal problems make the log UNHEALTHY; the others are warnings.
	Fatal bool
}

// String renders the problem on one line: its kind, its path if any, and its detail.
func (p Problem) String() string {
	s := p.Kind
	if p.Path != "" {
		s += " " + p.Path
	}
	switch {
	case p.refused:
		s += " refused without being read"
	case p.Kind == "segment":
		s += fmt.Sprintf(" damaged at byte %d, intact to byte %d", p.Offset, p.Intact)
	}
	if p.Detail != "" {
		s += ": " + p.Detail
	}
	return s
}

// CheckResult is what Check found.
type CheckResult struct {
	Healthy   bool
	Unbounded bool // neither MaxBytes nor MinFreeBytes in quill.policy
	Problems  []Problem
}

// Check examines a log without a lock: every sealed segment whole (one with no frames is
// damage, as it is to Open; only the active segment may be empty), the active segment up
// to the marker, holes, the consumers/ directory and its files, the policy and the clock (decisions 44, 66). It
// reads every byte, so it costs what a cold start costs, and like a cold start it verifies the
// segments in parallel.
func (r *Reader) Check() (CheckResult, error) {
	res := CheckResult{Healthy: true}
	fatal := func(p Problem) { p.Fatal = true; res.Healthy = false; res.Problems = append(res.Problems, p) }
	warn := func(p Problem) { res.Problems = append(res.Problems, p) }
	tl, err := r.readTail()
	if err != nil {
		return res, err
	}
	ids, err := listSegments(r.fs, r.dir)
	if err != nil {
		return res, err
	}
	future := r.opts.clock().Add(24 * time.Hour).Unix()
	// Each segment is verified on its own goroutine, as Open does (at most GOMAXPROCS at a
	// time), and the findings are then merged in segment order, so the report is the one a
	// serial pass makes. A segment's scan cannot know the At of the record before its first,
	// so the first record's clock test is made in the merge.
	var todo []uint64
	for _, id := range ids {
		if id <= tl.Seg {
			todo = append(todo, id)
		}
	}
	outs := make([]checkedSegment, len(todo))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, id := range todo {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			outs[i] = r.checkSegment(id, tl, future)
		}()
	}
	wg.Wait()
	var prevAt int64
	var prevLast, prevID uint64
	havePrev := false
	first := tl.Seq + 1 // the first retained sequence: the first good segment's, none retained is head + 1
	haveFirst := false
	for i, id := range todo {
		o := &outs[i]
		path := r.path(segmentName(id))
		if err := o.openErr; err != nil {
			if id < tl.Seg && errors.Is(err, fs.ErrNotExist) {
				// Retention deleted a sealed segment after the listing (decision 101). It
				// deletes oldest first, so what was scanned before it is going too: start
				// the contiguity test again at the next survivor.
				havePrev = false
				continue
			}
			var se *SegmentError
			if errors.As(err, &se) && se.oversize {
				fatal(Problem{Kind: "segment", Path: path, Detail: se.Reason, refused: true})
				continue
			}
			fatal(Problem{Kind: "segment", Path: path, Detail: err.Error()})
			continue
		}
		if o.short != nil {
			fatal(*o.short)
			continue
		}
		info := o.info
		if info.frames > 0 {
			if info.first.At > future {
				warn(clockProblem(path, info.first.Seq, info.first.At, "is more than a day in the future"))
			}
			if info.first.At < prevAt {
				warn(clockProblem(path, info.first.Seq, info.first.At, fmt.Sprintf("is earlier than its predecessor at %d", prevAt)))
			}
			prevAt = info.last.At // the records before a damaged frame count: they were accepted
		}
		for _, p := range o.clock {
			warn(p)
		}
		if err := o.err; err != nil {
			var se *SegmentError
			if errors.As(err, &se) {
				fatal(Problem{Kind: "segment", Path: path, Offset: se.Offset, Intact: se.Intact, Detail: se.Reason})
			} else {
				fatal(Problem{Kind: "segment", Path: path, Detail: err.Error()})
			}
			continue
		}
		if info.frames == 0 {
			// Only the active segment may be empty; a sealed one is corruption, as Open
			// finds it (decision 43).
			if id < tl.Seg {
				fatal(Problem{Kind: "segment", Path: path, Detail: "a sealed segment with no frames"})
			}
			continue
		}
		if !haveFirst {
			first, haveFirst = info.first.Seq, true
		}
		switch {
		case havePrev && info.first.Seq <= prevLast:
			// The overlap Open refuses (decision 102), not a hole.
			fatal(Problem{Kind: "overlap", Path: path, Detail: fmt.Sprintf("segment %d starts at %d, inside the one before (segment %d, ends at %d)", id, info.first.Seq, prevID, prevLast)})
		case havePrev && info.first.Seq != prevLast+1:
			fatal(Problem{Kind: "hole", Path: path, Detail: fmt.Sprintf("sequences %d to %d are missing before it", prevLast+1, info.first.Seq-1)})
		}
		prevLast, prevID, havePrev = info.last.Seq, id, true
	}
	if havePrev && prevLast < tl.Seq {
		fatal(Problem{Kind: "hole", Path: r.path(tailName), Detail: fmt.Sprintf("the marker names %d, the last frame is %d", tl.Seq, prevLast)})
	}
	consumers, err := r.consumerStore().list()
	if err != nil {
		fatal(Problem{Kind: "consumer", Path: r.path(consumersDir), Detail: err.Error()})
	}
	for _, c := range consumers {
		if c.Err != nil {
			fatal(Problem{Kind: "consumer", Path: r.path(filepath.Join(consumersDir, c.Name)), Detail: c.Err.Error()})
		} else if c.Offset > tl.Seq+1 {
			// The log is intact; the consumer is what a restore left behind (decision 36).
			warn(Problem{Kind: "consumer", Path: r.path(filepath.Join(consumersDir, c.Name)), Detail: fmt.Sprintf("at %d, ahead of the head %d: rewind it with quill commit", c.Offset, tl.Seq)})
		} else if c.Offset < first {
			// Retention deleted records the consumer had not read (decision 38): the log is
			// intact, the consumer's next read is a GapError, and this is the only offline sign.
			warn(Problem{Kind: "consumer", Path: r.path(filepath.Join(consumersDir, c.Name)), Detail: fmt.Sprintf("at %d, behind the first retained sequence %d: records %d to %d are gone", c.Offset, first, c.Offset, first-1)})
		}
	}
	data, err := readState(r.fs, r.path(policyName))
	if err != nil {
		warn(Problem{Kind: "policy", Path: r.path(policyName), Detail: err.Error()})
		res.Unbounded = true
	} else if p, ok := decodePolicy(data); !ok {
		warn(Problem{Kind: "policy", Path: r.path(policyName), Detail: "cannot be parsed"})
		res.Unbounded = true
	} else {
		res.Unbounded = p.MaxBytes == 0 && p.MinFreeBytes == 0
	}
	return res, nil
}

// verifyBounded verifies the first size bytes of a segment as a sealed one would be
// (contiguous, ending exactly at size; an empty range is fine for the active segment) and
// reports clock anomalies for every record after the first: one dated after future, or
// earlier than its predecessor. The first record's predecessor is in another segment, so
// its anomalies are the caller's to find from the info.
func verifyBounded(f io.ReaderAt, size int64, path string, future int64) (segInfo, []Problem, error) {
	w, release := verifyWindow(f, size)
	defer release()
	var clock []Problem
	var prevAt int64
	haveFirst := false
	info, err := walk(w, 0, 0, false, func(rec Record, _ int64) bool {
		if haveFirst {
			if rec.At > future {
				clock = append(clock, clockProblem(path, rec.Seq, rec.At, "is more than a day in the future"))
			}
			if rec.At < prevAt {
				clock = append(clock, clockProblem(path, rec.Seq, rec.At, fmt.Sprintf("is earlier than its predecessor at %d", prevAt)))
			}
		}
		prevAt, haveFirst = rec.At, true
		return true
	})
	if err != nil {
		if !isFrameError(err) {
			return info, clock, err
		}
		return info, clock, &SegmentError{Offset: info.end, Intact: info.end, Reason: reason(err)}
	}
	return info, clock, nil
}

// checkedSegment is what verifying one segment found, before the findings are put in order.
type checkedSegment struct {
	openErr error    // the open failed
	short   *Problem // the active segment is shorter than the marker says
	info    segInfo
	err     error     // the verification error
	clock   []Problem // clock findings for the records after the first
}

// checkSegment opens and verifies segment id of the log the marker tl describes (the active
// one only up to the marker). It touches nothing but the segment, so segments may be
// checked at the same time.
func (r *Reader) checkSegment(id uint64, tl tail, future int64) checkedSegment {
	path := r.path(segmentName(id))
	f, size, err := openSegment(r.fs, path)
	if err != nil {
		return checkedSegment{openErr: err}
	}
	defer f.Close()
	bound := size
	if id == tl.Seg {
		if size < int64(tl.Off) {
			return checkedSegment{short: &Problem{Kind: "segment", Path: path, Offset: size, Intact: size, Detail: fmt.Sprintf("%d bytes, the marker says %d are durable", size, tl.Off)}}
		}
		bound = int64(tl.Off)
	}
	var out checkedSegment
	out.info, out.clock, out.err = verifyBounded(f, bound, path, future)
	return out
}

func clockProblem(path string, seq uint64, at int64, what string) Problem {
	return Problem{Kind: "clock", Path: path, Detail: fmt.Sprintf("record %d at %d %s", seq, at, what)}
}
