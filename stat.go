package quill

import (
	"errors"
	"fmt"
	"path/filepath"
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
	// the writer opened the log. Only Log.Stat knows it.
	Dropped uint64
	// StaleConsumers are protected consumers that stopped protecting segments under
	// ConsumerTTL at the last prune pass. Only Log.Stat knows them.
	StaleConsumers []string
	// MarkerAge is how long ago quill.tail was last written.
	MarkerAge time.Duration
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
	v, release, err := l.view()
	if err != nil {
		return Stats{}, err
	}
	release()
	var s Stats
	s.count(v)
	l.mu.Lock()
	defer l.mu.Unlock()
	s.Segments = len(l.sealed) + 1
	for _, seg := range l.sealed {
		s.Bytes += seg.size
	}
	s.Bytes += l.active.durable
	s.Dropped = l.dropped
	s.StaleConsumers = append([]string(nil), l.stale...)
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
	names, err := r.fs.ReadDir(r.dir)
	if err != nil {
		return Stats{}, err
	}
	for _, n := range names {
		if _, ok := parseSegmentName(n); !ok {
			continue
		}
		info, err := r.fs.Lstat(r.path(n))
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
	Kind   string // "segment", "hole", "consumer", "policy", "clock"
	Path   string
	Offset int64 // for a segment: where the damage starts
	Intact int64 // for a segment: the end of the last valid frame
	Detail string
	// Fatal problems make the log UNHEALTHY; the others are warnings.
	Fatal bool
}

func (p Problem) String() string {
	s := p.Kind
	if p.Path != "" {
		s += " " + p.Path
	}
	if p.Kind == "segment" {
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

// Check examines a log without a lock: every sealed segment whole, the active segment up
// to the marker, holes, consumer files, the policy and the clock (decisions 44, 66). It
// reads every byte, so it costs what a cold start costs.
func (r *Reader) Check() (CheckResult, error) {
	res := CheckResult{Healthy: true}
	fatal := func(p Problem) { p.Fatal = true; res.Problems = append(res.Problems, p) }
	warn := func(p Problem) { res.Problems = append(res.Problems, p) }
	tl, err := r.readTail()
	if err != nil {
		return res, err
	}
	names, err := r.fs.ReadDir(r.dir)
	if err != nil {
		return res, err
	}
	future := r.opts.clock().Add(24 * time.Hour).Unix()
	var prevAt int64
	var prevLast uint64
	havePrev := false
	for _, n := range names {
		id, ok := parseSegmentName(n)
		if !ok || id > tl.Seg {
			continue
		}
		path := r.path(n)
		f, size, err := openSegment(r.fs, path)
		if err != nil {
			fatal(Problem{Kind: "segment", Path: path, Detail: err.Error()})
			continue
		}
		var info segInfo
		if id == tl.Seg {
			if size < int64(tl.Off) {
				fatal(Problem{Kind: "segment", Path: path, Offset: size, Intact: size, Detail: fmt.Sprintf("%d bytes, the marker says %d are durable", size, tl.Off)})
				_ = f.Close()
				continue
			}
			info, err = verifyBounded(f, int64(tl.Off), &prevAt, future, func(seq uint64, at int64, what string) {
				warn(Problem{Kind: "clock", Path: path, Detail: fmt.Sprintf("record %d at %d %s", seq, at, what)})
			})
		} else {
			info, err = verifyBounded(f, size, &prevAt, future, func(seq uint64, at int64, what string) {
				warn(Problem{Kind: "clock", Path: path, Detail: fmt.Sprintf("record %d at %d %s", seq, at, what)})
			})
		}
		_ = f.Close()
		if err != nil {
			var se *SegmentError
			if errors.As(err, &se) {
				fatal(Problem{Kind: "segment", Path: path, Offset: se.Offset, Intact: se.Intact, Detail: se.Reason})
			} else {
				fatal(Problem{Kind: "segment", Path: path, Detail: err.Error()})
			}
			continue
		}
		if info.frames == 0 {
			continue
		}
		if havePrev && info.first.Seq != prevLast+1 {
			fatal(Problem{Kind: "hole", Path: path, Detail: fmt.Sprintf("sequences %d to %d are missing before it", prevLast+1, info.first.Seq-1)})
		}
		prevLast, havePrev = info.last.Seq, true
	}
	if havePrev && prevLast < tl.Seq {
		fatal(Problem{Kind: "hole", Path: r.path(tailName), Detail: fmt.Sprintf("the marker names %d, the last frame is %d", tl.Seq, prevLast)})
	}
	for _, c := range mustList(r.consumerStore()) {
		if c.Err != nil {
			fatal(Problem{Kind: "consumer", Path: r.path(filepath.Join(consumersDir, c.Name)), Detail: c.Err.Error()})
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
	for _, p := range res.Problems {
		if p.Fatal {
			res.Healthy = false
		}
	}
	return res, nil
}

func mustList(st *consumerStore) []ConsumerInfo {
	list, _ := st.list()
	return list
}

// verifyBounded verifies the first size bytes of a segment as a sealed one would be
// (contiguous, ending exactly at size; an empty range is fine for the active segment) and
// reports clock anomalies: a record dated after future, or earlier than its predecessor.
func verifyBounded(f interface {
	ReadAt([]byte, int64) (int, error)
}, size int64, prevAt *int64, future int64, clock func(seq uint64, at int64, what string)) (segInfo, error) {
	w := &window{r: f, size: size, chunk: windowChunk}
	info, err := walk(w, 0, 0, false, func(rec Record, _ int64) bool {
		if rec.At > future {
			clock(rec.Seq, rec.At, "is more than a day in the future")
		}
		if rec.At < *prevAt {
			clock(rec.Seq, rec.At, fmt.Sprintf("is earlier than its predecessor at %d", *prevAt))
		}
		*prevAt = rec.At
		return true
	})
	if err != nil {
		if !isFrameError(err) {
			return info, err
		}
		return info, &SegmentError{Offset: info.end, Intact: info.end, Reason: reason(err)}
	}
	return info, nil
}
