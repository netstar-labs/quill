package quill

import (
	"context"
	"fmt"
	"math"
)

// Append buffers one record and returns its sequence. The record is durable after the next
// Flush, Sync, or inline flush at FlushBytes; with SyncAppends it is durable when Append
// returns, and an error then means unknown, not lost.
//
// On an error the returned sequence says whether one was assigned: a failure after the record
// was buffered (the inline flush's fsync failed and the log is failed, or SyncAppends could not
// make it durable) returns the sequence with the error, and the record may still be on disk
// after a reopen, so look for it there; a failure before (ErrClosed, a failed log, ErrPayloadTooLarge,
// ErrSeqExhausted, the buffer or rotation bound) returns 0, and nothing was assigned.
func (l *Log) Append(payload []byte) (uint64, error) {
	return l.AppendBatch([][]byte{payload})
}

// AppendBatch buffers records with consecutive sequences under one lock acquisition and
// returns the first; the last is first + len(payloads) − 1. A batch lands in one segment and
// is not crash-atomic: after a crash a prefix of it may survive. A batch of more than one record
// is refused (decision 46) when the sum of len(payload)+34 over its records is over MaxBatchBytes: an
// upper bound of the encoded size (the sequence is not known yet), so the real size can be much
// smaller for small payloads. A single record is bounded by MaxPayloadBytes alone.
//
// On an error the first return is the first sequence of the batch when sequences were assigned
// (the inline flush's fsync failed, or SyncAppends could not make the batch durable: the
// records may still be on disk after a reopen) and 0 when nothing was assigned (ErrClosed, a
// failed log, a refused batch, ErrPayloadTooLarge, ErrSeqExhausted, the buffer or rotation bound).
func (l *Log) AppendBatch(payloads [][]byte) (uint64, error) {
	if len(payloads) == 0 {
		return 0, fmt.Errorf("%w: empty batch", ErrPayloadTooLarge)
	}
	size := 0
	for _, p := range payloads {
		if len(p) > l.cfg.MaxPayloadBytes {
			return 0, fmt.Errorf("%w: %d bytes, MaxPayloadBytes is %d", ErrPayloadTooLarge, len(p), l.cfg.MaxPayloadBytes)
		}
		// The sequence is not known yet; bound the frame by the largest varints.
		size += 3*maxVarint + len(p) + 4
	}
	if len(payloads) > 1 && size > l.cfg.MaxBatchBytes {
		return 0, fmt.Errorf("%w: batch of about %d bytes, MaxBatchBytes is %d", ErrPayloadTooLarge, size, l.cfg.MaxBatchBytes)
	}
	l.mu.Lock()
	first, err := l.appendLocked(payloads, size)
	l.mu.Unlock()
	if err != nil {
		return first, err // first is 0 when nothing was assigned
	}
	if l.cfg.SyncAppends {
		if err := l.Sync(context.Background(), first+uint64(len(payloads))-1); err != nil {
			return first, err
		}
	}
	return first, nil
}

// appendLocked assigns sequences and encodes the batch into the buffer, flushing inline at
// FlushBytes. While flushes are failing the buffer is bounded by MaxBufferBytes (decision 69).
func (l *Log) appendLocked(payloads [][]byte, size int) (uint64, error) {
	switch {
	case l.closed:
		return 0, ErrClosed
	case l.failed != nil:
		return 0, l.failed
	case l.diskFull:
		return 0, errDiskFull
	}
	n := uint64(len(payloads))
	// The ceiling is MaxUint64-1: the last sequence is never assigned, so last+1 (nextSeq, a
	// read cursor, a consumer offset) never wraps. Open refuses a log whose head is MaxUint64.
	if n > math.MaxUint64-l.nextSeq {
		return 0, ErrSeqExhausted
	}
	if l.flushErr != nil && l.bufCount > 0 && len(l.buf)+size > l.cfg.MaxBufferBytes {
		return 0, fmt.Errorf("%w (buffer at %d bytes, MaxBufferBytes %d)", l.flushErr, len(l.buf), l.cfg.MaxBufferBytes)
	}
	// Rotation keeps failing (decision 60 d): the segment may grow only to the ceiling, which
	// validate keeps under the file limit Open enforces (decision 16).
	if l.rotErr != nil {
		if end, ceiling := l.active.durable+int64(len(l.buf)+size), l.cfg.maxActiveBytes(); end > ceiling {
			return 0, fmt.Errorf("%w (active segment would reach %d bytes, ceiling %d)", l.rotErr, end, ceiling)
		}
	}
	first := l.nextSeq
	at := l.cfg.clock().Unix()
	if l.bufCount == 0 {
		l.bufFirst = first
	}
	for i, p := range payloads {
		rec := Record{Seq: first + uint64(i), At: at, Payload: p}
		l.buf = appendFrame(l.buf, &rec)
	}
	l.bufCount += len(payloads)
	l.nextSeq = first + n
	if len(l.buf) >= l.cfg.FlushBytes {
		_ = l.flushLocked() // a write error keeps the records buffered; an fsync error fails the log
		if l.failed != nil {
			return first, l.failed
		}
	}
	return first, nil
}
