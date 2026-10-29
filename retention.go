package quill

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"syscall"
	"time"
)

// Retention deletes whole sealed segments, oldest first, by age and by caps (decisions 24
// to 26, 31, 33, 44, 45, 57, 65). The age sweep respects protected consumers and the clock
// guard; the caps override protection. Nothing ever deletes the active segment.

// consumerView is what a prune pass learned from the consumer files, read without the lock.
type consumerView struct {
	offsets      []uint64 // every readable consumer's offset, for Dropped
	minProtected uint64   // the slowest live protected consumer's offset
	hasProtected bool
	broken       bool // a consumer file could not be read: protect everything
	stale        []string
}

// consumers reads the consumer files for a prune pass and decides which protected consumers
// are stale under ConsumerTTL (decision 45): behind, with an offset file older than the
// TTL, and the record at the offset older than the TTL. It takes no lock.
func (l *Log) consumers(now time.Time) consumerView {
	var cv consumerView
	st := l.consumerStore()
	list, err := st.list()
	if err != nil {
		cv.broken = true
		return cv
	}
	head := l.Head()
	ttl := int64(l.cfg.ConsumerTTL.Seconds())
	for _, c := range list {
		if c.Err != nil {
			cv.broken = true
			continue
		}
		cv.offsets = append(cv.offsets, c.Offset)
		if !c.Protected {
			continue
		}
		if l.cfg.ConsumerTTL > 0 && c.Offset <= head {
			mtime, err := st.mtime(c.Name)
			stale := false
			if err == nil && now.Sub(mtime) >= l.cfg.ConsumerTTL {
				// The record at the offset, or the first retained one when a cap already
				// deleted it (decision 82).
				recs, err := l.Read(c.Offset, 1)
				var g *GapError
				if errors.As(err, &g) {
					recs, err = l.Read(g.To+1, 1)
				}
				if err == nil && len(recs) == 1 && now.Unix()-recs[0].At >= ttl {
					stale = true
				}
			}
			if stale {
				cv.stale = append(cv.stale, c.Name)
				continue
			}
		}
		if !cv.hasProtected || c.Offset < cv.minProtected {
			cv.minProtected, cv.hasProtected = c.Offset, true
		}
	}
	return cv
}

// Prune runs a retention pass now: age rotation of a quiet active segment, the age sweep
// (at most a quarter of the sealed segments unless force lifts the clock guard), then the
// caps. The flusher runs the same pass every PruneInterval unless NoMaintenance is set.
func (l *Log) Prune(force bool) error {
	now := l.cfg.clock()
	cv := l.consumers(now)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	if l.failed != nil {
		return l.failed
	}
	return l.pruneLocked(now, cv, force)
}

func (l *Log) pruneLocked(now time.Time, cv consumerView, force bool) error {
	l.stale = cv.stale
	for _, name := range cv.stale {
		l.logf(slog.LevelWarn, "consumer is stale and no longer protects segments", "consumer", name)
	}
	// Age rotation: a quiet active segment is sealed so retention can reach it.
	if l.cfg.SegmentMaxAge > 0 && l.active.frames > 0 && now.Unix()-l.active.lastAt >= int64(l.cfg.SegmentMaxAge.Seconds()) {
		if err := l.flushLocked(); err != nil {
			return err
		}
		if l.active.frames > 0 && !l.pendingPublish {
			if err := l.rotateLocked(); err != nil {
				l.pendingPublish, l.bgErr = true, err
			} else if err := l.publishLocked(); err != nil {
				return err
			}
		}
	}
	// The age sweep: the longest prefix of sealed segments whose newest record is older
	// than Retention. A damaged segment rides along with an expired good one after it.
	n := 0
	if l.cfg.Retention > 0 {
		cutoff := now.Unix() - int64(l.cfg.Retention.Seconds())
		for i, s := range l.sealed {
			if s.bad != nil {
				continue
			}
			if s.lastAt > cutoff {
				break
			}
			n = i + 1
		}
	}
	// Protection: stop before the first segment holding records at or above the slowest
	// protected consumer's offset. A damaged segment's range is bounded by its neighbours.
	if cv.broken || cv.hasProtected {
		minProtected := cv.minProtected
		if cv.broken {
			minProtected = l.firstRetainedLocked()
		}
		for i := 0; i < n; i++ {
			if l.lastOfLocked(i) >= minProtected {
				n = i
				break
			}
		}
	}
	// The clock guard: at most a quarter of the sealed segments per pass, at least one.
	if limit := (len(l.sealed) + 3) / 4; n > limit && !force {
		l.logf(slog.LevelWarn, "age sweep limited by the clock guard; use prune -force if the clock is right",
			"expired", n, "deleting", limit, "sealed", len(l.sealed))
		n = limit
	}
	n = l.capsLocked(n)
	return l.deleteLocked(n, cv.offsets)
}

// firstRetainedLocked is the first retained sequence.
func (l *Log) firstRetainedLocked() uint64 {
	return l.firstAfterLocked(0)
}

// firstAfterLocked is the first retained sequence once the first k sealed segments are gone.
func (l *Log) firstAfterLocked(k int) uint64 {
	for _, s := range l.sealed[k:] {
		if s.bad == nil {
			return s.first
		}
	}
	if l.active.frames > 0 {
		return l.active.first
	}
	return l.head + 1
}

// lastOfLocked is the last sequence of sealed segment i; for a damaged one, the sequence
// before the next good segment's first.
func (l *Log) lastOfLocked(i int) uint64 {
	if s := l.sealed[i]; s.bad == nil {
		return s.last
	}
	return l.firstAfterLocked(i+1) - 1
}

// capsLocked extends a deletion prefix of n sealed segments until the caps hold: MaxBytes,
// then MinFreeBytes, then MaxRecords. They override protection. With nothing left to delete
// under MinFreeBytes, appends fail with ENOSPC until a later pass finds room.
func (l *Log) capsLocked(n int) int {
	if l.cfg.MaxBytes > 0 {
		bytes := l.active.durable
		for _, s := range l.sealed[n:] {
			bytes += s.size
		}
		for bytes > l.cfg.MaxBytes && n < len(l.sealed) {
			bytes -= l.sealed[n].size
			n++
		}
	}
	if l.cfg.MinFreeBytes > 0 {
		free, err := l.cfg.statfs(l.dir)
		if err != nil {
			l.logf(slog.LevelWarn, "cannot read free space", "err", err)
		} else {
			for free < uint64(l.cfg.MinFreeBytes) && n < len(l.sealed) {
				free += uint64(l.sealed[n].size)
				n++
			}
			full := free < uint64(l.cfg.MinFreeBytes)
			if full && !l.diskFull {
				l.logf(slog.LevelWarn, "free space below MinFreeBytes with nothing left to delete; appends will fail", "free", free)
			}
			l.diskFull = full
		}
	}
	if l.cfg.MaxRecords > 0 {
		for n < len(l.sealed) && l.head >= l.firstAfterLocked(n) && l.head-l.firstAfterLocked(n)+1 > uint64(l.cfg.MaxRecords) {
			n++
		}
	}
	return n
}

// deleteLocked unlinks the first n sealed segments: the marker is published and the
// directory fsynced first (decision 65). offsets are the consumers' offsets, for Dropped.
//
// The publish may seal an active segment that outgrew SegmentBytes after a failed rotation
// (decision 60 d), and that seal runs the caps (decision 24), which delete a prefix of
// l.sealed through a nested call. So the segments chosen are recorded first and n is
// re-derived afterwards as what is left of them at the front (decision 93).
func (l *Log) deleteLocked(n int, offsets []uint64) error {
	if n == 0 {
		return nil
	}
	chosen := make(map[*segment]bool, n)
	for _, s := range l.sealed[:n] {
		chosen[s] = true
	}
	if err := l.publishLocked(); err != nil {
		return err
	}
	n = 0
	for n < len(l.sealed) && chosen[l.sealed[n]] {
		n++
	}
	if n == 0 {
		return nil
	}
	if err := l.fs.SyncDir(l.dir); err != nil {
		l.fail(err)
		return l.failed
	}
	for i := 0; i < n; i++ {
		s := l.sealed[i]
		if err := l.fs.Remove(s.path); err != nil {
			l.bgErr = err
			l.logf(slog.LevelWarn, "cannot delete segment", "segment", s.path, "err", err)
			n = i
			break
		}
		last := l.lastOfLocked(i)
		for _, off := range offsets {
			if off <= last {
				l.dropped++
				break
			}
		}
		l.logf(slog.LevelInfo, "deleted segment", "segment", s.path, "last", last)
	}
	l.sealed = l.sealed[n:]
	first := l.firstRetainedLocked()
	kept := l.holes[:0]
	for _, h := range l.holes {
		if h.To >= first {
			kept = append(kept, h)
		}
	}
	l.holes = kept
	return nil
}

// logf writes to the configured logger, if any.
func (l *Log) logf(level slog.Level, msg string, args ...any) {
	if l.cfg.Logger != nil {
		l.cfg.Logger.Log(context.Background(), level, msg, append([]any{"log", l.dir}, args...)...)
	}
}

// errDiskFull is the append error while MinFreeBytes cannot be met.
var errDiskFull = fmt.Errorf("quill: free space below MinFreeBytes and nothing left to delete: %w", syscall.ENOSPC)
