//go:build linux || darwin

// Package quill is an embeddable, append-only log on the standard library alone: a log is a
// directory with one assigned writer and any number of consumers, in the writer's process or
// another.
//
// The rules to know:
//
//   - One writer per log. Open takes an exclusive lock on the directory; a second writable open
//     fails with ErrLocked. Readers in other processes use OpenReader and take no lock.
//   - Records are numbered from 1 and sequences are contiguous. Append returns before the record
//     is durable; Flush, Sync or SyncAppends make it so. After a crash the unflushed suffix is
//     lost and its numbers are reassigned, so a caller flushes before acknowledging a sequence
//     upstream.
//   - Readers see only durable records: in-process reads see flushed records, and a Reader in
//     another process sees records up to the quill.tail marker, which the writer publishes after
//     each fsync.
//   - A consumer is a name plus a committed offset, the next sequence to read, stored in its own
//     file and written only by the process holding that consumer. Commit once per batch.
//   - Retention deletes whole sealed segments, oldest first, by age and by caps. It does not wait
//     for a consumer unless that consumer is protected, and the caps win even then.
//   - Corruption is never served silently: a damaged or missing sealed segment becomes a hole that
//     reads report as a GapError and Err() names until retention deletes it; damage in the newest segment below the durable
//     marker, or a marker whose sequence is above the last valid frame, fails Open with ErrCorrupt.
//
// The on-disk format is specified in docs/format.md and frozen by golden tests.
package quill
