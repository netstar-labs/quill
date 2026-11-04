//go:build linux || darwin

// Package quill is an embeddable, append-only log on the standard library alone: a log is a
// directory with one assigned writer and any number of consumers, in the writer's process or
// another.
//
// The rules to know:
//
//   - One writer per log. Open takes an exclusive lock on the directory; a second writable open
//     fails with ErrLocked. Readers in other processes use OpenReader and take no lock. The lock
//     is a flock: it holds on a local filesystem written by one kernel, not over NFS, SMB, FUSE
//     or a directory shared between VMs or containers, and quill does not detect those (see Open).
//     The lock file quill.lock is always 0600, so a writable open under a different account than
//     the one that created the log, which cannot open that file, fails with the permission error
//     on quill.lock (EACCES), not ErrLocked, whether or not a writer is live. A caller that
//     wants to know if a writer may be live treats both as "not this account's log, or a writer
//     may be live".
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
//     for a consumer unless that consumer is protected, and the caps (MaxBytes, MinFreeBytes and
//     MaxRecords) win even then: a protected consumer holds data back from the age sweep only, and
//     can lose records to a cap, MinFreeBytes included.
//   - Corruption is never served silently: a damaged sealed segment becomes a hole that reads
//     report as a GapError and Err() names until retention deletes it; a missing one is a hole in
//     Stat().Holes and in reads, not in Err(); damage in the newest segment below the durable
//     marker, or a marker whose sequence is above the last valid frame, fails Open with ErrCorrupt.
//
// The on-disk format is specified in docs/format.md and frozen by golden tests.
package quill
