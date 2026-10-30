package quill

import (
	"errors"
	"fmt"
)

// Errors returned by the package, matched with errors.Is. A *GapError is matched with errors.As.
var (
	// ErrCorrupt reports damage below the durable marker. Open returns it for the newest segment
	// (bytes below quill.tail.off that do not verify, a marker above the last valid frame, an
	// overlap between segments, a damaged quill.tail or quill.format). A sealed segment that does
	// not verify does not fail Open: it becomes a hole, reads return a *GapError over it, and
	// Err() returns the *SegmentError, which matches ErrCorrupt. The user guide has the procedure.
	ErrCorrupt = errors.New("quill: corrupt log")
	// ErrFutureVersion reports a quill.format marker newer than this build knows.
	ErrFutureVersion = errors.New("quill: log format is newer than this build")
	// ErrPayloadTooLarge reports a payload above MaxPayloadBytes, a batch above MaxBatchBytes, or an
	// empty batch (AppendBatch with no payloads).
	ErrPayloadTooLarge = errors.New("quill: payload too large")
	// ErrSeqExhausted reports that the log has used every sequence number.
	ErrSeqExhausted = errors.New("quill: sequence numbers exhausted")
	// ErrClosed reports a call on a closed Log, Reader or Consumer, or a Sync waiter cut off by Close.
	ErrClosed = errors.New("quill: closed")
	// ErrInvalidName reports a consumer name outside [a-z0-9_-], 1 to 200 characters, or one with
	// a leading underscore, which is reserved (except the archiver's "_archive").
	ErrInvalidName = errors.New("quill: invalid consumer name")
	// ErrLocked reports a log or consumer held by another process (or another handle).
	ErrLocked = errors.New("quill: locked by another holder")
	// ErrNoLog reports a directory that is not a log: no quill.format, or no quill.tail yet.
	ErrNoLog = errors.New("quill: not a log")
	// ErrFailed reports a log that hit an fsync error: every later Append, Flush and Sync fails
	// until the log is closed and reopened (decision 37).
	ErrFailed = errors.New("quill: log failed; close and reopen")
	// ErrWrongLog reports a consumer file whose log id is another log's.
	ErrWrongLog = errors.New("quill: consumer belongs to another log")
	// ErrAhead reports a consumer offset above Head + 1: the log was restored to an older state.
	ErrAhead = errors.New("quill: consumer offset is ahead of the log")
	// ErrOffsetRange reports a Commit outside [first retained, Head + 1].
	ErrOffsetRange = errors.New("quill: offset out of range")
	// ErrBadSeq reports a Sync on a sequence that has not been assigned.
	ErrBadSeq = errors.New("quill: sequence not assigned")
	// ErrConfig reports a Config that cannot be applied.
	ErrConfig = errors.New("quill: invalid config")
)

// GapError reports that the records at the requested position are unavailable: deleted by
// retention (an offset below the first retained sequence) or missing or unreadable in the
// middle (a hole). From is the requested offset; To is the last missing sequence, so a consumer
// resumes with Commit(To + 1). Reads never skip a gap silently (decision 38).
type GapError struct {
	From, To uint64
}

// Error describes the missing range.
func (e *GapError) Error() string {
	return fmt.Sprintf("quill: records %d to %d are unavailable", e.From, e.To)
}
