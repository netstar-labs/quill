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
	offsets      []consumerOffset // every readable consumer, for Dropped and the Warn that names it
	minProtected uint64           // the slowest live protected consumer's offset
	hasProtected bool
	broken       bool    // a consumer file could not be read: protect everything
	brokenWhy    []error // what could not be read, naming the file, for the Warn
	stale        []string
	probeFailed  []probeFailure // protected consumers kept live because their staleness probe errored
}

// consumerOffset is one readable consumer as a prune pass saw it: the name is what the Warn
// for a deletion past the consumer's offset prints.
type consumerOffset struct {
	name   string
	offset uint64
}

// probeFailure is a ConsumerTTL probe that could not run: the consumer is treated as live.
type probeFailure struct {
	name string
	err  error
}

// consumers reads the consumer files for a prune pass and decides which protected consumers
// are stale under ConsumerTTL (decision 45): behind (offset <= Head), with an offset file at
// least as old as the TTL, and the record at the offset at least as old as the TTL (an age equal
// to the TTL counts, to the second). It takes l.mu only briefly (Head and Read do), so call it without l.mu held.
func (l *Log) consumers(now time.Time) consumerView {
	var cv consumerView
	st := l.consumerStore()
	list, err := st.list()
	if err != nil {
		cv.broken = true
		cv.brokenWhy = append(cv.brokenWhy, fmt.Errorf("cannot list consumers: %w", err))
		return cv
	}
	head := l.Head()
	ttl := int64(l.cfg.ConsumerTTL.Seconds())
	for _, c := range list {
		if c.Err != nil {
			cv.broken = true
			cv.brokenWhy = append(cv.brokenWhy, fmt.Errorf("consumer %q: %w", c.Name, c.Err))
			continue
		}
		cv.offsets = append(cv.offsets, consumerOffset{c.Name, c.Offset})
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
//
// Prune returns an error when the pass did less than the policy asked because an unlink failed:
// the segment stays, retention stalls behind it, and the same error is in Err() until a later
// pass deletes it. A pass the clock guard limited, or one that kept segments for an unreadable
// consumer file, returns nil and logs a Warn: neither is a failure, and force answers the first.
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
		return l.failed
	}
	if l.bg[bgPolicy] != nil {
		l.recordPolicy()
	}
	l.stale, l.offsets = cv.stale, cv.offsets
	for _, name := range cv.stale {
		l.logf(slog.LevelWarn, "consumer is stale and no longer protects segments", "consumer", name)
	}
	// An unreadable consumer file protects every segment from the age sweep; a pass that kept
	// the segments silently would leave the operator to find out why nothing expires.
	for _, why := range cv.brokenWhy {
		l.logf(slog.LevelWarn, "consumer file could not be read; the age sweep keeps every segment until it is fixed or removed", "err", why)
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
				l.pendingPublish = true
				l.setBg(bgRotate, err)
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
	// A pass that reaches its deletions is a fresh attempt at every unlink an earlier pass failed:
	// the error those left in Err() is dropped here and set again by any unlink that fails now.
	// The failure stays logged as one failing state: only a pass whose deletions ran and ended
	// without a failed unlink logs a recovery. A pass that stopped before any unlink (the
	// publish that precedes them failed) retried nothing, so it neither recovers nor re-warns.
	l.bg[bgPrune] = nil
	if err := l.deleteLocked(n, cv.offsets); err != nil {
		return err
	}
	if l.bg[bgPrune] == nil {
		l.clearBg(bgPrune)
	}
	// An unlink that failed left its segment in place and is already in Err(): the pass did
	// less than it was asked, so the caller is told, not only the log (decision 65).
	return l.bg[bgPrune]
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
			// One failure is the trigger. The floor cannot be checked, so the last known state
			// stands (a set diskFull keeps refusing appends) and Err() says so until a pass reads.
			l.logf(slog.LevelWarn, "cannot read free space", "err", err)
			statErr := fmt.Errorf("quill: cannot read free space for MinFreeBytes: %w", err)
			if l.diskFull {
				statErr = errors.Join(errDiskFull, statErr)
			}
			l.setBg(bgDisk, statErr)
		} else {
			l.freeAtPass = free
			for free < uint64(l.cfg.MinFreeBytes) && n < len(l.sealed) {
				free += uint64(l.sealed[n].size)
				n++
			}
			full := free < uint64(l.cfg.MinFreeBytes)
			switch {
			case full && !l.diskFull:
				l.logf(slog.LevelWarn, "free space below MinFreeBytes with nothing left to delete; appends will fail", "free", free)
			case !full && l.diskFull:
				l.logf(slog.LevelInfo, "free space is back above MinFreeBytes; appends are accepted again", "free", free)
			}
			l.diskFull = full
			if full {
				l.setBg(bgDisk, errDiskFull)
			} else {
				l.clearBg(bgDisk)
			}
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
func (l *Log) deleteLocked(n int, offsets []consumerOffset) error {
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
			l.setBg(bgPrune, err, "segment", s.path)
			n = i
			break
		}
		last := l.lastOfLocked(i)
		// A consumer at or below the segment's last record loses records it had not read (or, when
		// it was already below the first retained sequence, loses more of them). Dropped counts
		// the segment once; the Warn names each consumer, since Dropped is only a count in this
		// process's memory (decision 12).
		counted := false
		for _, c := range offsets {
			if c.offset > last {
				continue
			}
			if !counted {
				l.dropped++
				counted = true
			}
			l.logf(slog.LevelWarn, "deleted segment past a consumer's offset; it has lost records",
				"consumer", c.name, "offset", c.offset, "segment", s.path, "last", last)
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
	// A damaged segment that is now the oldest file is a hole from 1 up to the first retained
	// sequence, as a fresh load and a Reader see it. Its own hole ended below first and was just
	// dropped with the deleted prefix, though the file is still there.
	if len(l.sealed) > 0 && l.sealed[0].bad != nil && first > 1 {
		kept = append([]Hole{{From: 1, To: first - 1}}, kept...)
	}
	l.holes = kept
	if l.loadErr != nil {
		// The load error names the first damaged segment still on disk: when that one is gone
		// it names the next, and when none is left there is nothing to report.
		l.loadErr = nil
		for _, s := range l.sealed {
			if s.bad != nil {
				l.loadErr = s.bad
				break
			}
		}
	}
	if syncErr != nil {
		l.fail("segment delete", syncErr)
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
