package quill

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"runtime"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Flush writes the buffered records at the durable offset, fsyncs, then publishes
// quill.tail, rotating first when the segment is full. It returns the first error met; after a
// write error the records stay buffered for the next attempt, after an fsync error the log is
// failed, and after a publish error the records are durable but the marker is stale until the
// next flush republishes it (decision 60). On an open failed log Flush returns ErrFailed.
func (l *Log) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	return l.flushLocked()
}

func (l *Log) flushLocked() error {
	if l.bg[bgPolicy] != nil {
		l.recordPolicy()
	}
	if l.failed != nil {
		return l.failed
	}
	if l.lockLostLocked() {
		return l.failed
	}
	if l.bufCount == 0 {
		if l.pendingPublish {
			// Only a retry of a marker publish that had failed completes a flush that wrote;
			// a retried rotation on an idle log wrote nothing and leaves LastFlushOK alone.
			publishWasFailing := l.bg[bgPublish] != nil
			err := l.publishLocked()
			if publishWasFailing {
				l.noteFlushOKLocked()
			}
			l.broadcastLocked()
			return err
		}
		return nil
	}
	a := l.active
	off := a.durable
	if _, err := a.f.WriteAt(l.buf, off); err != nil {
		// (a) Truncate back so no partial frame remains; keep the records buffered.
		if terr := a.f.Truncate(off); terr != nil {
			l.fail("flush", terr)
			return l.failed
		}
		l.flushErr = err
		l.setBg(bgFlush, err)
		l.broadcastLocked()
		return err
	}
	if err := a.f.Sync(); err != nil {
		// (b) Fatal and sticky: a retried fsync can report success for lost data.
		l.fail("flush", err)
		return l.failed
	}
	l.fsyncs++
	// (c) Durable. Index what was written, release the buffer, let readers see it.
	l.indexBuffer(a, off)
	a.durable = off + int64(len(l.buf))
	l.head = l.bufFirst + uint64(l.bufCount) - 1
	l.buf, l.bufCount = l.buf[:0], 0
	l.flushErr = nil
	l.clearBg(bgFlush)
	// (d, e) Rotate if due and publish the marker; a failed create or marker write is retried next
	// time, a failed directory fsync fails the log.
	err := l.publishLocked()
	l.noteFlushOKLocked()
	l.broadcastLocked()
	return err
}

// noteFlushOKLocked stamps Stats.LastFlushOK after a flush that wrote has also published its
// marker: the marker publish did not fail and the log is not failed. A rotation or a cap that
// failed after a good publish does not hold it back; the records are durable and visible.
func (l *Log) noteFlushOKLocked() {
	if l.failed == nil && l.bg[bgPublish] == nil {
		l.lastFlushOK = l.cfg.clock()
	}
}

// lockLostLocked fails the log, and reports true, when the quill.lock this writer holds is no
// longer the file at that path: it was removed or replaced (a tmp cleaner, a restore over the
// directory), so a second writer can have opened and flock'd a new file, and the two would
// overwrite each other's acknowledged records. Any other lstat error is not a verdict and is
// ignored. Called with the log's lock held; costs one lstat per flush and prune pass.
func (l *Log) lockLostLocked() bool {
	if l.failed != nil {
		return false
	}
	held, err := l.lock.Stat()
	if err != nil {
		return false
	}
	cur, err := l.fs.Lstat(l.path(lockName))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if err == nil && fsx.SameFile(held, cur) {
		return false
	}
	l.fail("lock check", errors.New("quill.lock is no longer the file this writer holds: it was removed or replaced, so another writer may have opened the log"))
	return true
}

// fail puts the log in the failed state and wakes every waiter, so a ReadWait or Follow parked
// above the head returns the failure instead of waiting for a record that can no longer come
// (the wake is here, not at each call site: Roll and a retention pass fail the log without a
// flush to broadcast after it). It also writes the one Error record that says so through
// Config.Logger: op names the operation that failed, err is the cause. The state is sticky and
// keeps its first cause, so a later call changes nothing and logs nothing, and the record is
// written once. The caller holds l.mu, which Config.Logger's documented rule allows.
func (l *Log) fail(op string, err error) {
	if l.failed != nil {
		return
	}
	l.failed = fmt.Errorf("%w: %w", ErrFailed, err)
	// A failed log does not retry, so an earlier write failure is no longer "flushes are failing":
	// FlushFailing stays false (Stats). Every reader of flushErr checks failed first.
	l.flushErr = nil
	l.logf(slog.LevelError, "log failed: it accepts no more writes and every later call returns ErrFailed", "op", op, "error", err)
	l.broadcastLocked()
}

// indexBuffer adds the buffer's frames, now durable at off, to the active segment's index.
func (l *Log) indexBuffer(a *activeSegment, off int64) {
	data := l.buf
	pos := off
	for len(data) > 0 {
		seq, at, total, ok := frameHeader(data)
		if !ok {
			return // cannot happen: the writer encoded these frames
		}
		if a.frames == 0 {
			a.first, a.at = seq, at
		}
		if pos >= a.nextAnchor {
			a.index = append(a.index, anchor{seq: seq, pos: pos, at: at})
			a.nextAnchor = pos + indexStride
		}
		a.lastAt = at
		a.frames++
		pos += int64(total)
		data = data[total:]
	}
}

// publishLocked rotates when the active segment has reached SegmentBytes, then writes
// quill.tail if it changed. A failure to rotate still publishes the current position (the
// data is durable and the marker is accurate); either failure leaves pendingPublish set so
// the next flush retries (decision 60 d).
func (l *Log) publishLocked() error {
	var rotErr error
	if l.active.durable >= l.cfg.SegmentBytes && l.active.frames > 0 {
		if rotErr = l.rotateLocked(); rotErr != nil {
			if l.failed != nil {
				return l.failed
			}
			l.setBg(bgRotate, rotErr)
		}
		l.rotErr = rotErr
	}
	sealed := l.justSealed
	m := tail{Seq: l.head, Seg: l.active.id, Off: uint64(l.active.durable), MaxSeg: l.maxSeg}
	if m != l.marker || l.pendingPublish || sealed {
		l.tailBuf = appendTail(l.tailBuf[:0], m)
		if err := writeState(l.fs, l.tailPath, l.tailBuf, l.cfg.Perm); err != nil {
			l.pendingPublish = true
			l.setBg(bgPublish, err)
			return err
		}
		l.marker, l.markerAt = m, l.cfg.clock()
		l.openUnpublished = false
		l.clearBg(bgPublish)
		if sealed {
			// The marker naming the new segment is made durable at rotation (decision 65).
			if err := l.fs.SyncDir(l.dir); err != nil {
				l.fail("rotation", err)
				return l.failed
			}
			l.justSealed = false
		}
	}
	l.pendingPublish = rotErr != nil
	if sealed && rotErr == nil {
		// The caps are enforced at every seal (decision 24); they override protection, so
		// only the consumers' offsets are needed, for Dropped. Those are the ones the last
		// prune pass read: listing consumers/ here would run under l.mu, and its cost grows
		// with the number of files in a directory other processes write to (decision 12).
		if err := l.deleteLocked(l.capsLocked(0), l.offsets); err != nil {
			return err
		}
	}
	return rotErr
}

// rotateLocked seals the active segment: create the next one, fsync the directory, retire
// the old descriptor (decision 43). The caller publishes the marker naming the new segment.
func (l *Log) rotateLocked() error {
	if l.maxSeg >= maxSegmentID {
		// The next name would be eleven digits, which no scan recognises (or would wrap).
		return fmt.Errorf("quill: segment ids exhausted: %d is the largest the ten-digit names hold", uint64(maxSegmentID))
	}
	id := l.maxSeg + 1
	path := l.path(segmentName(id))
	f, err := l.fs.OpenFile(path, fsx.O_RDWR|fsx.O_CREATE|fsx.O_EXCL, l.cfg.Perm)
	if err != nil {
		return err
	}
	if err := l.fs.SyncDir(l.dir); err != nil {
		_ = f.Close()
		l.fail("rotation", err) // (e) a directory fsync error is fatal
		return l.failed
	}
	old := l.active
	l.sealed = append(l.sealed, &segment{id: old.id, path: old.path, first: old.first, last: l.head,
		size: old.durable, index: old.index, at: old.at, lastAt: old.lastAt})
	l.refsValid = false
	old.retired = true
	_ = old.closeIfIdle()
	l.maxSeg = id
	l.active = &activeSegment{id: id, path: path, f: f, first: l.head + 1}
	l.justSealed = true
	l.clearBg(bgRotate)
	return nil
}

// Roll flushes and seals the active segment now, so it can be archived or reached by
// retention. It does nothing when the active segment holds no frames.
func (l *Log) Roll() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	if err := l.flushLocked(); err != nil {
		return err
	}
	if l.active.frames == 0 || l.pendingPublish {
		return nil
	}
	if err := l.rotateLocked(); err != nil {
		l.pendingPublish = true
		l.setBg(bgRotate, err)
		return err
	}
	err := l.publishLocked()
	l.broadcastLocked()
	return err
}

// broadcastLocked wakes every waiter on the flush broadcast. The channel is made lazily by
// waitChanLocked, so a flush with nobody waiting allocates nothing.
func (l *Log) broadcastLocked() {
	if l.flushed != nil {
		close(l.flushed)
		l.flushed = nil
	}
	l.flushGen++
}

// waitChanLocked returns the channel the next broadcastLocked closes, making it if no waiter
// has yet. The caller holds l.mu.
func (l *Log) waitChanLocked() chan struct{} {
	if l.flushed == nil {
		l.flushed = make(chan struct{})
	}
	return l.flushed
}

// Sync signals the flusher and blocks until seq is durable, sharing one fsync with every other
// waiter. An error means unknown, not lost: the records may still become durable (decision
// 42). seq above the last assigned sequence is ErrBadSeq; a waiter pending at Close gets
// ErrClosed unless its sequence became durable. On a failed log, Sync of a sequence that became
// durable before the failure returns nil, and Sync of any other assigned sequence returns an
// error wrapping ErrFailed.
func (l *Log) Sync(ctx context.Context, seq uint64) error {
	l.mu.Lock()
	if seq >= l.nextSeq {
		last := l.nextSeq - 1
		l.mu.Unlock()
		return fmt.Errorf("%w: %d (last assigned %d)", ErrBadSeq, seq, last)
	}
	gen := l.flushGen
	for {
		switch {
		case l.head >= seq:
			l.mu.Unlock()
			return nil
		case l.failed != nil:
			err := l.failed
			l.mu.Unlock()
			return err
		case l.flushGen != gen && l.flushErr != nil:
			err := l.flushErr
			l.mu.Unlock()
			return err
		case l.closed:
			l.mu.Unlock()
			return ErrClosed
		}
		ch := l.waitChanLocked()
		l.mu.Unlock()
		select {
		case l.flushReq <- struct{}{}:
		default:
		}
		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err()
		}
		l.mu.Lock()
	}
}

// flusher is the log's own goroutine: a flush every FlushInterval when there is something to
// write, and at once when Sync asks.
func (l *Log) flusher() {
	defer close(l.done)
	t := time.NewTicker(l.cfg.FlushInterval)
	defer t.Stop()
	var pruneC <-chan time.Time // nil with NoMaintenance: that case never fires
	if !l.cfg.NoMaintenance {
		prune := time.NewTicker(l.cfg.PruneInterval)
		defer prune.Stop()
		pruneC = prune.C
	}
	for {
		select {
		case <-l.stop:
			return
		case <-t.C:
		case <-l.flushReq:
			// Let the producers already runnable append and ask too: at GOMAXPROCS 1 the Sync that
			// woke this goroutine would otherwise hand it the processor at once, and every flush
			// would carry one record.
			runtime.Gosched()
		case <-pruneC:
			now := l.cfg.clock()
			cv := l.consumers(now)
			l.mu.Lock()
			if !l.closed && l.failed == nil {
				// An error from the pass is already in Err(): each step that can return one
				// (a flush, a publish, a rotation, an fsync) records it where it arises.
				_ = l.pruneLocked(now, cv, false)
			}
			l.mu.Unlock()
			continue
		}
		l.mu.Lock()
		if !l.closed && (l.bufCount > 0 || l.pendingPublish) {
			_ = l.flushLocked()
		}
		l.mu.Unlock()
	}
}
