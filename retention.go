package quill

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
	probeFailed  []probeFailure // protected consumers kept live because their staleness probe errored
}

// probeFailure is a ConsumerTTL probe that could not run: the consumer is treated as live.
type probeFailure struct {
	name string
	err  error
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
			stale := false
			mtime, err := st.mtime(c.Name)
			if err == nil && now.Sub(mtime) >= l.cfg.ConsumerTTL {
				// The record at the offset, or the first retained one when a cap already
				// deleted it (decision 82).
				var recs []Record
				recs, err = l.Read(c.Offset, 1)
				var g *GapError
				if errors.As(err, &g) {
					recs, err = l.Read(g.To+1, 1)
				}
				if err == nil && len(recs) == 1 && now.Unix()-recs[0].At >= ttl {
					stale = true
				}
			}
			// A probe that errors cannot show the consumer is stale, so it stays live (it
			// protects), but the pass says the TTL was not evaluated.
			if err != nil {
				cv.probeFailed = append(cv.probeFailed, probeFailure{c.Name, err})
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
// (at most a quarter of the sealed segments, rounded up, unless force lifts the clock guard), then the
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
	if l.lockLostLocked() {
		l.broadcastLocked()
		return l.failed
	}
	if l.policyErr != nil {
		l.recordPolicy()
	}
	l.stale, l.offsets = cv.stale, cv.offsets
	for _, name := range cv.stale {
		l.logf(slog.LevelWarn, "consumer is stale and no longer protects segments", "consumer", name)
	}
	for _, f := range cv.probeFailed {
		l.logf(slog.LevelWarn, "consumer staleness probe failed; treating it as live", "consumer", f.name, "err", f.err)
	}
	// Age rotation: a quiet active segment is sealed so retention can reach it.
	if l.cfg.SegmentMaxAge > 0 && l.active.frames > 0 && now.Unix()-l.active.lastAt >= int64(l.cfg.SegmentMaxAge.Seconds()) {
		// A failed flush (a full disk, say) skips the rotation but not the pass: deleting expired
		// segments is how the log finds room, so the sweep and the caps still run. The flush
		// error stays in Err() and the next flush retries.
		if err := l.flushLocked(); err != nil {
			if l.failed != nil {
				return l.failed
			}
			l.logf(slog.LevelWarn, "age rotation skipped because the flush failed; the sweep and the caps still run", "err", err)
		} else if l.active.frames > 0 && !l.pendingPublish {
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
				// A newest record more than a day ahead of now is a clock that stepped forward
				// and was corrected (decision 44): this segment, and every newer one, is kept
				// until the clock reaches the stamp. The caps still bound the disk.
				if s.lastAt-now.Unix() > futureSkewSeconds {
					l.logf(slog.LevelWarn, "age sweep stopped at a segment stamped in the future; newer segments are kept until the clock reaches it",
						"segment", s.id, "last_at", s.lastAt, "kept", len(l.sealed)-i)
				}
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
	// The clock guard: at most a quarter of the sealed segments per pass, rounded up, so at least one.
	if limit := (len(l.sealed) + 3) / 4; n > limit && !force {
		l.logf(slog.LevelWarn, "age sweep limited by the clock guard; use prune -force if the clock is right",
			"expired", n, "deleting", limit, "sealed", len(l.sealed))
		n = limit
	}
	n = l.capsLocked(n)
	return l.deleteLocked(n, cv.offsets)
}

// futureSkewSeconds is how far ahead of now a sealed segment's newest record may be stamped
// before the age sweep warns about it: the same day that check uses (decision 44).
const futureSkewSeconds = 24 * 60 * 60

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

// deleteLocked unlinks the first n sealed segments, oldest first: the marker is published and
// the directory fsynced before each unlink (decision 65). offsets are the consumers' offsets,
// for Dropped.
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
	// The directory is synced before each unlink, not once before the first: a power loss may
	// persist any unsynced unlink (decision 61), and one that persists without the unlink
	// before it leaves a hole in the middle of the retained records. Synced one at a time,
	// the segments that survive a crash are always a suffix.
	var syncErr error
	for i := 0; i < n; i++ {
		if syncErr = l.fs.SyncDir(l.dir); syncErr != nil {
			n = i
			break
		}
		s := l.sealed[i]
		err := l.fs.Remove(s.path)
		if errors.Is(err, fs.ErrNotExist) {
			// Already gone (removed by hand under a live writer): the goal is met. Failing
			// here would block every segment behind this one, and the caps with it.
			l.logf(slog.LevelWarn, "segment was already gone", "segment", s.path)
		} else if err != nil {
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
		if s.bad != nil {
			// The file and its hole were the only artifacts of the damage (decision 98), and the
			// deletion removes both: this line is the one record that records were lost to damage
			// and not to age or a cap.
			l.logf(slog.LevelWarn, "deleted damaged segment", "segment", s.path, "last", last, "damage", s.bad)
		} else {
			l.logf(slog.LevelInfo, "deleted segment", "segment", s.path, "last", last)
		}
	}
	l.sealed = l.sealed[n:]
	l.refsValid = false
	first := l.firstRetainedLocked()
	// A fresh slice: segViews and Stat snapshots alias l.holes, so it is only ever reassigned.
	kept := make([]Hole, 0, len(l.holes))
	for _, h := range l.holes {
		if h.To >= first {
			kept = append(kept, h)
		}
	}
	l.holes = kept
	if syncErr != nil {
		l.fail(syncErr)
		return l.failed
	}
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
