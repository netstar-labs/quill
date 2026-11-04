package quill

import (
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// Config configures a Log. The zero value of every field but Dir is the default below.
type Config struct {
	// Dir is the log's directory, created if missing. Its base name is the topic name. A
	// symbolic link to a directory is followed, as OpenReader follows one, whether it is Dir
	// itself or a directory above it; the files inside the log directory never are. A relative
	// Dir is made absolute at Open, so a later os.Chdir does not move the log. Dir must be on a
	// local filesystem (ext4, xfs, btrfs, APFS) written by one host or kernel: see Open.
	Dir string
	// Retention: whole sealed segments at least this old are deleted (an age equal to it counts). 0 means 24 h; negative
	// disables expiry. Set it longer than the worst delay of any unprotected consumer.
	Retention time.Duration
	// PruneInterval is the sweep period: min(Retention, 1 m), at least 1 s; one minute when
	// Retention is negative.
	PruneInterval time.Duration
	// FlushBytes is the buffer size at which an append flushes inline (64 KiB).
	FlushBytes int
	// FlushInterval is the background flush period (250 ms).
	FlushInterval time.Duration
	// SyncAppends makes every Append and AppendBatch return only once its records are durable,
	// as if it called Sync on its last sequence. An error then means unknown, not lost, and the
	// call returns the first sequence it assigned beside it (0 when it assigned none).
	SyncAppends bool
	// MaxBufferBytes applies only while flushes are failing: then an append is accepted only
	// into an empty buffer or within this limit (16 × FlushBytes). Below FlushBytes is refused,
	// so the disk bound of decision 69, which charges a flush at this value, is exact.
	MaxBufferBytes int
	// SegmentBytes is the rotation size (8 MiB), at most 1 GiB, and under the 1.5 GiB segment file
	// limit once the largest flush is added: max(FlushBytes, MaxBufferBytes) + max(MaxBatchBytes,
	// MaxPayloadBytes+34). While rotation keeps failing, an append that would pass that sum is
	// refused with the rotation error.
	SegmentBytes int64
	// SegmentMaxAge rotates a quiet active segment so retention can delete it (Retention).
	// Left at zero it takes Retention's value, so it is never when Retention is negative; an
	// explicit positive value still rotates, whatever Retention is, and a negative one is never.
	SegmentMaxAge time.Duration
	// MaxBytes caps this log's segment bytes (0: off). Below twice SegmentBytes is refused.
	MaxBytes int64
	// MinFreeBytes is a free-space floor on the log's filesystem (0: off). It is checked at each
	// seal and prune pass. With nothing left to delete, appends fail with an error wrapping
	// ENOSPC, and Err() and Stat().DiskFull report it until a pass finds room. When statfs fails,
	// Err() reports that until a later pass can read the free space, and the floor is not enforced
	// meanwhile.
	//
	// The floor overrides consumer protection. Below it the writer deletes the oldest sealed
	// segments (never the active one) until its estimate, the free space read once by statfs plus
	// each deleted segment's size, reaches the floor, or no sealed segment is left, whether or not
	// a protected consumer has read them. A protected consumer can therefore lose records, and the
	// Warn that names it ("deleted segment past a consumer's offset; it has lost records") and
	// Stat().Dropped are how that is seen. It is the same for MaxBytes and MaxRecords; only the
	// age sweep (Retention) waits for a protected consumer. Free space that another process uses
	// counts: it is paid for with this log's history.
	MinFreeBytes int64
	// MaxPayloadBytes is the largest accepted record (64 MiB − 32, the ceiling).
	MaxPayloadBytes int
	// MaxBatchBytes bounds an AppendBatch of more than one record (16 MiB). The check is on an
	// upper bound of the encoded size, len(payload)+34 per record, so a batch of many small records
	// is refused before its real encoded size reaches the limit (about 479,000 one-byte records at
	// the default).
	MaxBatchBytes int
	// MaxReadBytes is the size after which one Read stops: each record counts its payload plus 48
	// bytes of overhead, so it bounds the size of a batch and not the sum of the payloads. A Read
	// returns at least one record, and the last record may carry the total past this bound (16 MiB;
	// at the default, about 349,000 empty records or 342,000 one-byte ones per call).
	MaxReadBytes int
	// ProtectedConsumers names consumers the writer creates, protected, when it opens the log, at
	// the first retained sequence (at 1 when a damaged oldest segment is a hole from 1). A
	// file already protected is left alone, so a running consumer does not block a restart.
	//
	// Protection holds sealed segments back from the age sweep (Retention) only: the sweep stops
	// before the first segment with a record at or above the slowest protected consumer's offset,
	// until ConsumerTTL finds that consumer stale. MaxBytes, MinFreeBytes and MaxRecords override
	// it: each deletes the oldest sealed segments whatever a protected consumer has read (MaxBytes
	// while the log's segment bytes exceed it, MaxRecords while more than MaxRecords records are
	// retained, MinFreeBytes while the free space is estimated below the floor), so a protected
	// consumer can lose records. The Warn that names it, and Stat().Dropped, say so.
	ProtectedConsumers []string
	// Logger receives every record the log writes, all through one handler. They are:
	//   - about a condition a prune pass (Prune, or the flusher's background pass) finds: Warn for each
	//     stale consumer, for a consumer file that cannot be read (the age sweep then keeps every
	//     segment) and for a staleness probe that fails; Warn when age rotation is skipped because
	//     the flush failed; Warn when the age sweep stops at a segment stamped in the future or is
	//     limited by the clock guard (the bulk-expiry warnings). Each is written again by every pass
	//     that finds the condition again;
	//   - about a deletion and the free-space floor, from a prune pass and also from every seal (an
	//     Roll, or any flush that rotates to a new segment (an Append's inline flush, Flush, Sync, Close or
	//     the flusher's), runs the caps,
	//     MaxBytes, MinFreeBytes and MaxRecords, at the seal; a Prune whose age rotation seals runs
	//     them at the seal and again in the pass): Info for each segment deleted, Warn instead when it
	//     held damage; Warn for a segment already gone from the directory; Warn naming each consumer
	//     whose offset a deleted segment passes; and for MinFreeBytes, Warn when the free space cannot
	//     be read, Warn when it is below the floor with nothing left to delete and Info when it is
	//     back above. The "cannot read free space" Warn is written by every pass and every seal whose
	//     statfs fails: six failing passes or seals write six Warns. Only the below-floor Warn and
	//     the back-above Info are one record per transition, and a deletion is one record when it
	//     happens (plus a Warn for each consumer it passes; a segment already gone gives its Warn and
	//     then the same deletion record, so two);
	//   - one Warn when a flush write, a marker publish, a rotation or a prune unlink starts to fail and
	//     one Info when it recovers (never one per retry; Stat has the current state);
	//   - one Warn when quill.policy cannot be written and one Info when a retry succeeds (the retries
	//     run at each flush and prune pass);
	//   - one Error when the log becomes failed (ErrFailed; with the cause and the operation, written
	//     once);
	//   - from Open: a Warn when it cuts the newest segment past the marker (with the segment, the
	//     offset, the bytes cut and whether the first rejected frame was incomplete or damaged), a Warn
	//     when it cannot tighten quill.lock or give quill.tail the mode in Perm, a Warn for each consumer whose offset is above Head + 1 (a
	//     log restored to an older state; with the consumer, the offset and the head), and the policy
	//     Warn above if its first write fails.
	//
	// Its handler, Enabled included, runs synchronously with the log's lock held, on the
	// goroutine that made the record (an Append, Flush, Roll, Prune or Close call, or the
	// log's flusher). It must therefore neither call the Log, which deadlocks it, nor block,
	// which stalls every Append, Read, Sync and the flusher (a handler that writes straight to a
	// pipe or socket blocks when the reader does), nor take a lock that a caller of the Log may
	// hold while calling it: hand the record to another goroutine and return. This holds for every
	// record above except those Open gives itself, which run on the caller's goroutine before Open
	// returns, before there is a Log, with no lock held.
	Logger *slog.Logger
	// Perm is the file mode (0600); directories get the matching execute bits (0700). State
	// files (quill.format, quill.tail, quill.policy, consumer offsets) are exactly Perm; Open
	// gives an existing quill.tail the mode of its Perm in place (a restart with another Perm
	// changes it at once, even on an idle log, and needs no free block). The
	// writer's segments, its consumers' lock files and every directory are created through the
	// process umask, so under umask 022 Perm 0660 gives 0640 segments and 0750 directories.
	// quill.lock is always 0600, whatever Perm says, so a reader-only process cannot take the
	// writer's lock (decision 51).
	Perm fs.FileMode
	// MaxRecords keeps roughly the newest N records (0: off).
	MaxRecords int64
	// ConsumerTTL: a protected consumer behind for this long stops protecting segments (0: off).
	// It is stale when it is behind and both the age of its offset file and the age of the record
	// at its offset are at least ConsumerTTL (decision 45). The file's age is the filesystem's
	// modification time against the writer's clock, so a clock step, a restore that resets mtimes,
	// a copy that does not preserve them, or a filesystem with coarse or wrong timestamps can make
	// a live consumer look stale (and lose its protection) or a dead one look live. A stale
	// consumer only stops protecting: the age sweep still deletes at most a quarter of the sealed
	// segments per pass (the clock guard, decision 44), and a record or file stamped in the future
	// is never stale. The check has no guard of its own and no per-consumer exemption (Protect is
	// what makes a consumer subject to it); to keep protection whatever the clock does, leave
	// ConsumerTTL at 0.
	ConsumerTTL time.Duration
	// NoMaintenance disables the background prune pass and age rotation: for offline tools
	// that open a log briefly (decision 68). Flushing still runs. Open does not write
	// quill.policy under it, so a tool never replaces the writer's settings (decision 99).
	NoMaintenance bool

	fs     fsx.FS                           // the seam; nil is the real file system
	clock  func() time.Time                 // nil is time.Now
	statfs func(dir string) (uint64, error) // nil is freeBytes
}

// Defaults and ceilings.
const (
	DefaultRetention     = 24 * time.Hour
	DefaultFlushBytes    = 64 << 10
	DefaultFlushInterval = 250 * time.Millisecond
	DefaultSegmentBytes  = 8 << 20
	MaxSegmentBytes      = 1 << 30
	DefaultMaxBatchBytes = 16 << 20
	DefaultMaxReadBytes  = 16 << 20
	// MaxPayloadBytes is the default and the ceiling of Config.MaxPayloadBytes.
	MaxPayloadBytes = maxPayloadBytes
	DefaultPerm     = fs.FileMode(0o600)
)

// withDefaults returns the config with every zero field set to its default.
func (c Config) withDefaults() Config {
	if c.Retention == 0 {
		c.Retention = DefaultRetention
	}
	if c.PruneInterval == 0 {
		if c.Retention < 0 {
			c.PruneInterval = time.Minute
		} else {
			c.PruneInterval = max(min(c.Retention, time.Minute), time.Second)
		}
	}
	if c.FlushBytes == 0 {
		c.FlushBytes = DefaultFlushBytes
	}
	if c.FlushInterval == 0 {
		c.FlushInterval = DefaultFlushInterval
	}
	if c.MaxBufferBytes == 0 {
		c.MaxBufferBytes = 16 * c.FlushBytes
	}
	if c.SegmentBytes == 0 {
		c.SegmentBytes = DefaultSegmentBytes
	}
	if c.SegmentMaxAge == 0 {
		c.SegmentMaxAge = c.Retention
	}
	if c.MaxPayloadBytes == 0 {
		c.MaxPayloadBytes = MaxPayloadBytes
	}
	if c.MaxBatchBytes == 0 {
		c.MaxBatchBytes = DefaultMaxBatchBytes
	}
	if c.MaxReadBytes == 0 {
		c.MaxReadBytes = DefaultMaxReadBytes
	}
	if c.Perm == 0 {
		c.Perm = DefaultPerm
	}
	if c.fs == nil {
		c.fs = fsx.OS{}
	}
	if c.clock == nil {
		c.clock = time.Now
	}
	if c.statfs == nil {
		c.statfs = freeBytes
	}
	return c
}

// validate checks a config with defaults applied.
func (c Config) validate() error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrConfig, fmt.Sprintf(format, args...))
	}
	switch {
	case c.Dir == "":
		return bad("Dir is required")
	case c.PruneInterval < time.Second:
		return bad("PruneInterval %v below 1s", c.PruneInterval)
	case c.FlushBytes < 0, c.FlushInterval < 0, c.MaxBufferBytes < 0, c.SegmentBytes < 0,
		c.MaxBytes < 0, c.MinFreeBytes < 0, c.MaxPayloadBytes < 0,
		c.MaxBatchBytes < 0, c.MaxReadBytes < 0, c.MaxRecords < 0, c.ConsumerTTL < 0:
		return bad("a size, count or duration is negative") // Retention and SegmentMaxAge may be: negative means never
	case c.MaxBufferBytes < c.FlushBytes:
		// Decision 69 charges a flush at MaxBufferBytes; a flush holds at least FlushBytes.
		return bad("MaxBufferBytes %d below FlushBytes %d", c.MaxBufferBytes, c.FlushBytes)
	case c.FlushInterval < time.Millisecond:
		return bad("FlushInterval %v below 1ms", c.FlushInterval)
	case c.SegmentBytes > MaxSegmentBytes:
		return bad("SegmentBytes %d above %d", c.SegmentBytes, MaxSegmentBytes)
	case c.MaxPayloadBytes > MaxPayloadBytes:
		return bad("MaxPayloadBytes %d above the ceiling %d", c.MaxPayloadBytes, MaxPayloadBytes)
	case c.maxActiveBytes() >= maxSegmentFileBytes:
		return bad("SegmentBytes %d plus one flush (up to %d bytes) reaches the %d-byte segment file limit",
			c.SegmentBytes, c.maxActiveBytes()-c.SegmentBytes, int64(maxSegmentFileBytes))
	case c.MaxBytes != 0 && c.MaxBytes < 2*c.SegmentBytes:
		return bad("MaxBytes %d below twice SegmentBytes %d", c.MaxBytes, c.SegmentBytes)
	case c.Perm&0o600 != 0o600:
		return bad("Perm %v must include owner read and write", c.Perm)
	case c.Perm&^fs.ModePerm != 0:
		return bad("Perm %v holds mode bits other than permissions", c.Perm)
	}
	for _, name := range c.ProtectedConsumers {
		if err := validName(name); err != nil {
			return bad("ProtectedConsumers: %v", err)
		}
	}
	return nil
}

// maxActiveBytes is the most the active segment can hold: SegmentBytes, which a flush reaches
// and then rotates, plus one flush: the buffer (at most the larger of FlushBytes and
// MaxBufferBytes) and the one append that overshoots it (at most MaxBatchBytes, or a single
// record's MaxPayloadBytes plus 34; the accounting of decision 69). The ceiling an append is
// refused at while rotation keeps failing, and what validate keeps under the file limit.
// Each flush term is capped at the file limit before the sum, so a size near math.MaxInt64
// cannot wrap the sum negative past validate; a capped term alone already reaches the limit.
func (c Config) maxActiveBytes() int64 {
	capped := func(n int) int64 { return min(int64(n), maxSegmentFileBytes) }
	return c.SegmentBytes + capped(max(c.FlushBytes, c.MaxBufferBytes)) +
		max(capped(c.MaxBatchBytes), capped(c.MaxPayloadBytes)+34)
}

// dirPerm is the directory mode for a file mode: execute wherever read is granted.
func (c Config) dirPerm() fs.FileMode {
	return c.Perm | (c.Perm&0o444)>>2
}

// archiveConsumer is the consumer `Archive` keeps. Its leading underscore is reserved for it
// alone (decision 41).
const archiveConsumer = "_archive"

// validConsumerName is validName plus exactly the archiver's name, so that the consumer
// `Archive` creates can be rewound and removed through the public paths (decision 41).
func validConsumerName(name string) error {
	if name == archiveConsumer {
		return nil
	}
	return validName(name)
}

// validName checks a consumer name: [a-z0-9_-], 1 to 200 characters, no leading underscore
// (reserved). Config.ProtectedConsumers is held to it; the archiver's name is not accepted there.
func validName(name string) error {
	if name == "" || len(name) > 200 {
		return fmt.Errorf("%w: %q: 1 to 200 characters", ErrInvalidName, name)
	}
	if name[0] == '_' {
		return fmt.Errorf("%w: %q: a leading underscore is reserved", ErrInvalidName, name)
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return fmt.Errorf("%w: %q: only [a-z0-9_-]", ErrInvalidName, name)
		}
	}
	return nil
}
