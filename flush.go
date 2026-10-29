package quill

import (
	"context"
	"fmt"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Flush writes the buffered records at the durable offset, fsyncs, then publishes
// quill.tail, rotating first when the segment is full. It returns the first error met; after a
// write error the records stay buffered for the next attempt, after an fsync error the log is
// failed, and after a publish error the records are durable but the marker is stale until the
// next flush republishes it (decision 60).
func (l *Log) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	return l.flushLocked()
}

func (l *Log) flushLocked() error {
	if l.failed != nil {
		return l.failed
	}
	if l.bufCount == 0 {
		if l.pendingPublish {
			err := l.publishLocked()
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
			l.fail(terr)
			l.broadcastLocked()
			return l.failed
		}
		l.flushErr, l.bgErr = err, err
		l.broadcastLocked()
		return err
	}
	if err := a.f.Sync(); err != nil {
		// (b) Fatal and sticky: a retried fsync can report success for lost data.
		l.fail(err)
		l.broadcastLocked()
		return l.failed
	}
	l.fsyncs++
	// (c) Durable. Index what was written, release the buffer, let readers see it.
	l.indexBuffer(a, off)
	a.durable = off + int64(len(l.buf))
	l.head = l.bufFirst + uint64(l.bufCount) - 1
	l.buf, l.bufCount = l.buf[:0], 0
	l.flushErr = nil
	// (d, e) Rotate if due and publish the marker; failures there are retried next time.
	err := l.publishLocked()
	l.broadcastLocked()
	return err
}

// fail puts the log in the failed state.
func (l *Log) fail(err error) {
	l.failed = fmt.Errorf("%w: %w", ErrFailed, err)
	l.bgErr = l.failed
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
			l.bgErr = rotErr
		}
	}
	m := tail{Seq: l.head, Seg: l.active.id, Off: uint64(l.active.durable), MaxSeg: l.maxSeg}
	if m != l.marker || l.pendingPublish {
		if err := writeState(l.fs, l.path(tailName), encodeTail(m), l.cfg.Perm); err != nil {
			l.pendingPublish, l.bgErr = true, err
			return err
		}
		l.marker = m
	}
	l.pendingPublish = rotErr != nil
	return rotErr
}

// rotateLocked seals the active segment: create the next one, fsync the directory, retire
// the old descriptor (decision 43). The caller publishes the marker naming the new segment.
func (l *Log) rotateLocked() error {
	id := l.maxSeg + 1
	path := l.path(segmentName(id))
	f, err := l.fs.OpenFile(path, fsx.O_RDWR|fsx.O_CREATE|fsx.O_EXCL, l.cfg.Perm)
	if err != nil {
		return err
	}
	if err := l.fs.SyncDir(l.dir); err != nil {
		_ = f.Close()
		l.fail(err) // (e) a directory fsync error is fatal
		return l.failed
	}
	old := l.active
	l.sealed = append(l.sealed, &segment{id: old.id, path: old.path, first: old.first, last: l.head,
		size: old.durable, index: old.index, at: old.at, lastAt: old.lastAt})
	old.retired = true
	if old.readers == 0 {
		_ = old.f.Close()
		old.f = nil
	}
	l.maxSeg = id
	l.active = &activeSegment{id: id, path: path, f: f, first: l.head + 1}
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
		l.pendingPublish, l.bgErr = true, err
		return err
	}
	err := l.publishLocked()
	l.broadcastLocked()
	return err
}

// broadcastLocked wakes every waiter on the flush broadcast.
func (l *Log) broadcastLocked() {
	close(l.flushed)
	l.flushed = make(chan struct{})
	l.flushGen++
}

// Sync signals the flusher and blocks until seq is durable, sharing one fsync with every other
// waiter. An error means unknown, not lost: the records may still become durable (decision
// 42). seq above the last assigned sequence is ErrBadSeq; a waiter pending at Close gets
// ErrClosed unless its sequence became durable.
func (l *Log) Sync(ctx context.Context, seq uint64) error {
	l.mu.Lock()
	if seq >= l.nextSeq {
		l.mu.Unlock()
		return fmt.Errorf("%w: %d (last assigned %d)", ErrBadSeq, seq, l.nextSeq-1)
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
		ch := l.flushed
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
	for {
		select {
		case <-l.stop:
			return
		case <-t.C:
		case <-l.flushReq:
		}
		l.mu.Lock()
		if !l.closed && (l.bufCount > 0 || l.pendingPublish) {
			_ = l.flushLocked()
		}
		l.mu.Unlock()
	}
}
