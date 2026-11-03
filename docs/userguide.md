# quill user guide

*How to run a log: the library calls, the `quill` command, the files, and day-2 operations.
The format itself is in [format.md](format.md); the design in [design/](design/).*

## 1. The model

A **log** is a directory holding one topic. `quill.Open(Config{Dir: dir})` creates it if
needed and takes an exclusive lock: that process is the log's one writer until `Close`.
Records are numbered from 1, contiguously. `Append` buffers a record; `Flush`, `Sync` or the
background flush makes it durable; only durable records are ever visible. A **consumer** is
a name plus a committed offset, the next sequence to read, stored in `consumers/<name>` and
written only by the process holding that consumer's lock. A **Reader** in another process
opens the same directory with `OpenReader`, takes no lock, and sees records up to
`quill.tail`, the marker the writer publishes after each fsync. Retention deletes whole
sealed segments, oldest first, by age and by caps; it does not wait for a consumer unless the
consumer is **protected**, and the caps win even then.

Several topics are several directories, in one process or in several. Each has its own lock,
buffer, flush goroutine, retention and consumers: topics share no lock, buffer or flusher, but on one
disk they still share the device.

## 2. Writing

```go
l, err := quill.Open(quill.Config{Dir: "/var/lib/app/orders"})
seq, err := l.Append(payload)                 // buffered; durable after the next flush
first, err := l.AppendBatch(payloads)         // consecutive sequences, one segment, one lock
err = l.Flush()                               // write, fsync, publish quill.tail
err = l.Sync(ctx, seq)                        // wait until seq is durable (group commit)
err = l.Roll()                                // seal the active segment now
err = l.Close()                               // flush, publish, fsync the directory, unlock
```

**Durability.** `Append` returns before the record is on disk. A caller that acknowledges a
record upstream must `Flush` or `Sync` first: after a crash the unflushed suffix is lost and
its sequence numbers are reassigned to the next records. Three ways to pay for durability:

| Mode | Cost | Use |
|---|---|---|
| Buffered (the default): `Append`; an inline flush at `FlushBytes`, a background flush every `FlushInterval` | One fsync per flush, shared by every record in it | Throughput; a window of `FlushInterval` can be lost |
| `Append` then `Sync(ctx, seq)` | One fsync per flush, shared by every producer waiting (group commit) | A durable write per call with many producers |
| `Config.SyncAppends` | Every `Append` returns durable; on an error it still returns the sequence it assigned | The simplest durable writer |

A flush costs an fsync of the segment and the marker's temp file: milliseconds on Linux,
about 20 ms on macOS (`F_FULLFSYNC`). Batch when you can.

**When a write fails** (decisions 37, 60). A short write or `ENOSPC` truncates the file back
to the durable offset and keeps the records buffered; the next flush retries from the same
offset, and space exhaustion recovers by itself. An **fsync** error is fatal: every later
valid `Append` and `Flush` on the open log, and every `Sync` of a sequence not already durable, returns an error wrapping `ErrFailed` until the log is closed and
reopened (`Sync` of a sequence that became durable before the failure returns nil, and `Sync` of an unassigned sequence is `ErrBadSeq`), and a `ReadWait` or `Follow` at the head returns it too (a goroutine already waiting is woken to return it; records that exist stay readable), because a retried fsync can report success for data that never reached the disk. The moment the log fails it writes one `Error` record on `Config.Logger`, "log failed: it accepts no more writes and every later call returns ErrFailed", with `error` (the first cause) and `op` (`flush`, `rotation`, `segment delete` or `lock check`); a later failure changes neither `Err()` nor the log, so there is never a second line, and `Err()` returns an error wrapping `ErrFailed`. With no `Logger` there is no line: poll `Err()`. An
error from `Sync` or a `SyncAppends` append means *unknown*, not *lost*: the records may still
become durable, so a producer that retries must tolerate duplicates. **The sequence beside an error:** `Append` and `AppendBatch` return the sequence they assigned (the first, for a batch) together with the error when they fail after assigning one, that is when the inline flush's fsync fails (the log is then failed, `ErrFailed`) or a `SyncAppends` append cannot make its records durable; the records may be on disk after a reopen, so that is the number to look for. They return 0 when nothing was assigned: `ErrClosed`, a failed log, a refused or empty batch, `ErrPayloadTooLarge`, `ErrSeqExhausted`, and an append refused at the `MaxBufferBytes` or rotation bound. A failure to publish the
marker or create the next segment after the data fsync is retried at the next flush, reported
by `Err()` while it lasts, and never rewrites durable data. While flushes fail the buffer is bounded by
`MaxBufferBytes`; past it, appends fail fast with the flush error. While creating the next
segment keeps failing, the writer keeps appending to the current one, but only to a ceiling of
`SegmentBytes + max(FlushBytes, MaxBufferBytes) + max(MaxBatchBytes, MaxPayloadBytes + 34)`, which
`Open` keeps under the 1.5 GiB file limit; past it an append fails with the rotation error and
assigns no sequence, until a flush creates the segment. Watch `Err()` and fix the cause (descriptors,
permissions, inodes) before the ceiling. `Err()` is what is failing now: a background error clears
when the operation that set it succeeds (a flush, a rotation, a marker publish, a prune pass), and
an old damaged segment does not hide it. It is nil when nothing is failing and no sealed segment is
damaged; when more than one thing is wrong it is their `errors.Join`, and `errors.Is` matches each.
Each of those retried failures (the flush write, the marker publish, the rotation, a prune unlink) is also
logged on `Config.Logger`: one `Warn` when it starts, with `op` (`flush`, `marker publish`, `rotation` or
`segment delete`) and `err`, and one `Info`, "background operation recovered", when it ends, never one line per
retry. "Recovered" means a later pass completed without a failure, not that the failing step was retried: a prune pass that finds nothing to delete after a failed unlink (the segment became protected, say) logs it and clears `Err()`. A failure that is already logged and fails again is not logged again, and a pass that stops before it
retries the failing step (the publish that precedes a prune pass's deletions failing) neither recovers nor warns
again. To alert on a flush that is not keeping up, poll `Log.Stat()`: `FlushFailing` is true while the last flush
that tried to write failed, and `BufferedRecords` and `BufferedBytes` are what has been acknowledged and is held
only in memory (lost if the process dies, invisible to readers). Alert on `FlushFailing`, or on
`BufferedRecords > 0` together with `FlushFailing`, and on `Err()` wrapping `ErrFailed` (a log that has failed
after a data fsync error does not retry, so `FlushFailing` stays false while its buffered records are never
written); do not alert on `BufferedRecords > 0` alone, which is normal for up to a flush interval after every
`Append`. `LastFlushOK` is the time of the last flush that completed
(it wrote and published its records, or it landed a marker publish that had failed, with nothing new written), or when the log opened, before the first; a log with nothing to flush keeps an old
one and is healthy, so one `Append` after an hour of quiet shows `BufferedRecords` of 1, `FlushFailing` false and
a `LastFlushOK` an hour old. It does not measure how long the oldest acknowledged record has waited, and
`time.Since(LastFlushOK)` alone is never an alert; read it once `FlushFailing` is true, to say how long writes
have been failing. A `Reader` reports none of them.

## 3. Reading in the writer's process

```go
recs, err := l.Read(offset, limit)            // copies; at most limit records, stops once MaxReadBytes (payload + 48 bytes per record) is reached
recs, err := l.ReadWait(ctx, offset, limit)   // blocks at the head until a flush (ErrFailed if the log failed)
err = l.Scan(offset, limit, func(r quill.Record) bool { ... })   // zero-copy, payload valid in the callback
for rec, err := range l.Records(ctx, offset) { ... }             // to the current end
for rec, err := range l.Follow(ctx, offset) { ... }              // and then wait for more
```

Reads return records with `Seq >= offset` in order. Where records are unavailable they return
a `*quill.GapError{From, To}`: `From` is the first unavailable sequence (the offset asked for when nothing was delivered before the gap; otherwise, for a `Scan`, `ScanAll`, `Reader.Scan`, `Consumer.Scan`, `Records` or `Follow` that delivered records first, the sequence after the last record delivered), `To` the last missing sequence, so a consumer resumes with `Commit(To+1)`.
Two causes: retention deleted them (the offset is below `Stat().First`), or a hole in the
middle (a damaged or deleted segment, including one a retention sweep deleted while a reader was
opening it). A damaged oldest segment is a hole from 1, not expiry: `ScanAll`, a scan from offset 0, `quill read`
and `quill cat` without `-from`, and a new consumer start below it and meet the gap, where a deleted prefix is skipped
because the scan starts at `Stat().First`; `quill tail -n N` meets it when N reaches back that far, while `-since` and
`OffsetForTime` still start at `Stat().First`. Nothing is skipped silently; resume from `To + 1`
deliberately. An I/O error while reading (`EIO`, for example) is not a gap and not `ErrCorrupt`: the read returns that
error, matchable with `errors.Is`, and reading again retries it; `Open` fails with it if it meets one verifying a
sealed segment. A sealed segment that cannot be opened at all (too many open files, permission denied) is a hole. An iterator yields an error once and ends; a gap is resumed by iterating again
from `To + 1`.

## 4. Consumers

```go
c, err := l.Consumer("billing", quill.ConsumerOptions{})         // or r.Consumer on a Reader
recs, err := c.Read(100)                                          // from c.Offset()
err = c.Commit(recs[len(recs)-1].Seq + 1)                         // the next sequence to read
for rec, err := range c.Follow(ctx) { process(rec); c.Commit(rec.Seq + 1) }
lag, err := c.Lag()                                               // Head + 1 − offset
err = c.SeekToTime(t)                                             // Commit(OffsetForTime(t))
err = c.Close()
```

- A consumer's offset is the **next sequence to read**; the idiom is `Commit(lastRead + 1)`.
  `Commit` accepts values from the first retained sequence to `Head + 1` and refuses others
  with `ErrOffsetRange`; a rewind is allowed.
- Opening a handle creates the file at the first retained sequence if absent (at sequence 1 when a damaged
  oldest segment left a hole below it, so the consumer's first read reports the gap), and takes
  `consumers/<name>.lock`; a second open of the same name, in any process, fails with
  `ErrLocked`. Names are `[a-z0-9_-]`, 1 to 200 characters; a leading underscore is reserved,
  except `_archive`, the archiver's own consumer, which `Consumer`, `DeleteConsumer` and the CLI accept.
- `Close` releases the consumer's lock and nothing more. `Read`, `ReadWait`, `Scan`, `Records` and `Follow` ignore it:
  they read through the `Log` or `Reader` the consumer came from and keep working, and a blocked `ReadWait` or `Follow`
  is not woken (it ends when its context is done or the `Log` or `Reader` closes). On a closed consumer only `Commit`,
  `SeekToTime` (which commits) and a second `Close` return `ErrClosed`. To stop a waiting consumer, cancel its context.
- **Commit once per batch, not per record**: each commit is a file write and an fsync.
- Delivery is at-least-once: commit after processing. Commit before processing for
  at-most-once. Exactly-once needs processing that is idempotent on `Seq`.
- `ErrWrongLog`: the consumer file belongs to another log (the log was recreated).
  `ErrAhead`: the offset is above `Head + 1` (the log was restored to an older state); Open
  refuses it unless `ConsumerOptions.Rewind` is set, which opens it so `Commit` can rewind it
  to any offset in `[first, Head + 1]`. `quill commit` sets it, so `quill commit -dir <log> -name <consumer> -offset N`
  is the remedy and keeps the protect flag; `quill register` still refuses. `quill consumers`
  notes such a consumer and `quill check` warns of it (the log stays HEALTHY). `quill commit` registers the consumer when the name is
  new, so a mistyped `-name` creates one; a commit that fails leaves no new consumer behind.
- `DeleteConsumer(name)` removes the offset file; the lock file stays for good.

**Protection.** A consumer opened with `Protect: quill.ProtectOn` (or `quill register
-protect`) keeps its unread sealed segments from the age sweep. The caps (`MaxBytes`,
`MinFreeBytes`, `MaxRecords`) still win, and `Stat().Dropped` counts segments deleted under a
consumer; the deletion is also logged at `Warn` on `Config.Logger`, naming the consumer ("deleted
segment past a consumer's offset; it has lost records", with `consumer`, `offset`, `segment` and
`last`). `Dropped` lives in the writer's memory and is 0 after a restart, so to find a consumer that
has fallen out of retention without the writer, run `quill consumers`: a consumer whose offset is below
the first retained sequence gets a NOTE ("behind the first retained sequence N: records X to N-1 are
gone"; `-json` adds `"behind_first": true`), and its LAG counts those records although they can no
longer be read. Its next read is a `*GapError`; `Commit(To+1)` or `quill commit` moves it past the gap.
`quill check` warns of the same consumer and stays HEALTHY. Unprotected is the default, so `Retention` must be longer than the worst delay of an
unprotected consumer, or negative. `Config.ProtectedConsumers` names consumers the writer
creates, protected, when it opens the log, so a lazy consumer is protected before its process
starts. A file that already says protected is left alone, so the writer can be redeployed while
that consumer runs; a consumer whose file says unprotected and whose process is running makes
`Open` fail with `ErrLocked` until that process opens it with `ProtectOn`, or stops and
`quill register -protect` runs. `ConsumerTTL` lets a protected consumer that has been behind and
silent for the TTL stop protecting segments; its file is kept and `Stat().StaleConsumers` lists it.
The `protected` column of `quill consumers` is what the file says, so it still reads true for a stale
consumer; the command derives staleness from the policy's `ConsumerTTL` and the file's mtime and adds a
`stale` NOTE (`-json`: `"stale": true`).

## 5. Reading from another process

```go
r, err := quill.OpenReader(dir, quill.ReaderOptions{PollInterval: 50 * time.Millisecond})
head, err := r.Head()                 // the marker's sequence
recs, err := r.Read(offset, limit)    // Read, ReadWait, Scan, ScanAll, Records, Follow, OffsetForTime
c, err := r.Consumer("billing", quill.ConsumerOptions{})
st, err := r.Stat()                   // first, head, segments, bytes, records, holes, marker age
res, err := r.Check()                 // every byte verified, up to the marker
```

A `Reader` takes no lock and never decides whether a writer is alive; `Stat().MarkerAge` is
the only liveness hint. It re-reads the marker on every call and sees a new record within
one poll interval of its flush. It needs read access to the directory; a consumer needs
write access to `consumers/`. `ErrNoLog` means the directory is not a log yet (a reader that
arrives during creation retries).

Two costs to know. A `Read`, `ReadWait`, `Scan` or `Consumer.Read` without an iterator re-walks and checks the active
segment from its start up to the offset you ask for, and rebuilds the view of the sealed segments, on every call: cheap
at the default 8 MiB `SegmentBytes`, but proportional to the segment, so a consumer that polls should use `Records` or
`Follow`, which keep a byte cursor. And the first time a `Reader` looks at a sealed segment it reads and CRC-checks that
segment's whole first frame, up to 64 MiB, once per `Reader` per segment; a directory with very large first frames
(a restored log, or a directory under `quill ls` that others can write to) costs that much per segment on a first `quill
stat` or `quill ls`.

## 6. Configuration

| Field | Default | Notes |
|---|---|---|
| `Dir` | required | Created if missing; its base name is the topic name. A symbolic link to a directory is followed, as `Dir` or above it (the data directory may be a link); a link that leads to anything else is `ErrNoLog` |
| `Retention` | 24 h | Sealed segments whose newest record is older are deleted; negative never |
| `PruneInterval` | min(Retention, 1 m), ≥ 1 s | The sweep period |
| `FlushBytes`, `FlushInterval` | 64 KiB, 250 ms | Inline flush size; background flush period |
| `SyncAppends` | off | Every append returns durable |
| `MaxBufferBytes` | 16 × `FlushBytes` | The buffer bound while flushes are failing. At least `FlushBytes`: `Open` refuses a smaller value (`ErrConfig`), so the bounded-disk formula below is exact |
| `SegmentBytes` | 8 MiB (≤ 1 GiB) | Rotation size; a segment passes it by the flush that seals it. `Open` refuses a value whose sum with the largest flush (`max(FlushBytes, MaxBufferBytes) + max(MaxBatchBytes, MaxPayloadBytes + 34)`) reaches the 1.5 GiB file limit |
| `SegmentMaxAge` | `Retention` | Rotates a quiet active segment so retention can reach it. The default is never when `Retention` is negative; an explicit positive value rotates whatever `Retention` is |
| `MaxBytes` | off | Cap on segment bytes, checked at every seal and pass; ≥ 2 × `SegmentBytes` |
| `MinFreeBytes` | off | Free-space floor (`statfs`); below it the log deletes its own oldest segments, then refuses appends with `ENOSPC` (`Err()` and `Stat().DiskFull` say so) |
| `MaxRecords` | off | Keep roughly the newest N records |
| `MaxPayloadBytes` | 64 MiB − 32 (the ceiling) | Largest record |
| `MaxBatchBytes` | 16 MiB | Bound on a batch of more than one record, checked as the sum of `len(payload) + 34` per record, an upper bound of its encoded size (the sequence is not known at the check): about 479,000 one-byte records fit at the default, though they encode to about 7 MB. A single record is bounded by `MaxPayloadBytes` alone |
| `MaxReadBytes` | 16 MiB | The size of a batch: a `Read` (and each batch of `Records` and `Follow`) stops once at least this many bytes are returned, each record counting its payload plus 48 bytes of overhead, so it bounds the memory of a read and not the sum of the payloads (at the default, about 349,000 empty records per call). It can be exceeded by up to one record, and a read returns at least one record |
| `ProtectedConsumers` | none | Created, protected, at open; an already protected file is left alone, so a running consumer does not block a restart |
| `ConsumerTTL` | off | See §4 |
| `Logger` | nil | A `*slog.Logger` for deletions, warnings and failures (a prune pass also gives a `Warn` for a consumer file that cannot be read, for a staleness probe that fails, for age rotation skipped because the flush failed and for the age sweep stopped at a segment stamped in the future or limited by the clock guard; the deletion records (a `Warn` for a segment already gone from the directory) and the `MinFreeBytes` records (a `Warn` when the free space cannot be read, a `Warn` when it is below the floor with nothing left to delete and an `Info` when it is back above) come from a prune pass and also from every seal, because the caps run at each seal: a `Roll`, or any flush that rotates to a new segment (an `Append`'s inline flush, `Flush`, `Sync`, `Close` or the flusher's); a `Prune` whose age rotation seals runs them at the seal and again in the pass. The unreadable-free-space `Warn` is written by every pass and every seal whose `statfs` fails, the other two once per transition; the records about a condition (this one, a stale consumer, an unreadable consumer file, a failed probe, a skipped age rotation, the two bulk-expiry warnings) are written again by every pass that finds the condition again, while a deletion is one record when it happens (plus a `Warn` for each consumer it passes; a segment already gone gives its `Warn` and then the same deletion record, so two); `quill.policy` has a `Warn` when it cannot be written and an `Info` when a retry succeeds), for the `Warn` and `Info` that mark a flush, marker publish, rotation or prune unlink starting to fail and recovering (one line each way, `op` names the step), for the one `Error` the log writes when it becomes failed ("log failed: it accepts no more writes and every later call returns ErrFailed", with `error` and `op`), for the `Warn` a prune pass gives when it deletes a damaged sealed segment ("deleted damaged segment", with `segment`, `last` and `damage`), for the `Warn` that names each consumer whose offset a deleted segment passes ("deleted segment past a consumer's offset; it has lost records", with `consumer`, `offset`, `segment` and `last`), and for the warning `Open` gives when it cuts the newest segment past the marker (`segment`, `offset`, `bytes`, and `kind`: `torn` for an incomplete final frame, the routine crash tail, or `damaged` for a complete-length frame that failed its checks, which may have been an acknowledged record), and for the `Warn` `Open` gives for each consumer whose offset is above `Head + 1` (a log restored to an older state: "consumer offset is ahead of the head; rewind it with quill commit", with `consumer`, `offset` and `head`). Its handler, `Enabled` included, runs with the log's lock held, so it must neither call the `Log` (that deadlocks it) nor block (that stalls `Append`, `Read`, `Sync` and the flusher; a handler that writes straight to a pipe or socket blocks when the reader does), nor take a lock that a caller of the `Log` may hold while calling it: hand the record to another goroutine and return. Every record above runs with the lock held except the warnings `Open` itself gives (the cut newest segment, a `quill.lock` it cannot tighten, an ahead consumer, a first `quill.policy` write that fails), which run before it returns, before there is a `Log`, with no lock held |
| `Perm` | 0600 (dirs 0700) | File mode. A `Reader`, the CLI and `Archive` create their files (consumer offsets, an archive's files) with the mode of the log's `quill.tail`, so a consumer in the log's group is readable by the writer. State files are exactly that mode; the writer's segments, the consumers' lock files and every directory go through the process umask, so for a group arrangement run the writer with umask 002 (`Perm` 0660 under umask 022 gives `0640` segments and `0750` directories, and a group consumer cannot create its files in `consumers/`). `quill.lock` is the exception: it is always 0600 whatever `Perm` says, and `Open` tightens one an earlier build left wider (if that chmod fails, because another account owns the file, `Open` still succeeds and logs a warning, which every CLI verb that opens the log writable also prints on standard error), so a reader-only process cannot take the writer's lock. A consumer's lock keeps the log's mode, because a group consumer must take its own; a process with read access to `consumers/` can therefore hold a consumer's lock and stall that consumer, never the writer. The CLI's offline write verbs (`append`, `roll`, `recover`, `prune`) open with no `Perm` of their own and keep the mode `quill.tail` already has |
| `NoMaintenance` | off | For offline tools: no background prune or age rotation, and `quill.policy` is not written |

**Bounded disk.** With `MaxBytes` set, segment bytes stay within
`MaxBytes + SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)`: the last term is the largest single
append, and a single record is bounded by `MaxPayloadBytes` alone, so with the defaults it is the larger. Defaults leave `MaxBytes` and
`MinFreeBytes` off, so a log with only `Retention` can fill a disk; `quill check` and
`quill ls` mark it UNBOUNDED. For planning, `bytes ≈ records × (average payload + about 16)`.
Several logs on one disk share no budget: size them so the sum of every log's
`MaxBytes + SegmentBytes` leaves headroom, and use `MinFreeBytes` as the backstop.

**When the floor cannot be met or checked** (decision 25). The floor is checked at each seal and prune pass. When nothing is
left to delete and the disk is still below it, the log refuses appends with an error wrapping `ENOSPC`; `Err()` returns the
same error and `Stat().DiskFull` is true, with `Stat().FreeBytesAtLastPass` the free space the last pass read. The state
clears at the first pass that finds room, which logs it at `Info`. When `statfs` itself fails, a single failure is enough:
`Err()` wraps the `statfs` error from that pass until a later pass can read the free space, and the floor is not enforced
in the meantime (a `DiskFull` already set is kept, and `Err()` reports both). Alert on `Err()` and `Stat().DiskFull`, not on
the `Warn` lines alone: with no `Logger` there are none.

**A wrong clock.** `At` is the writer's clock. A prune pass deletes at most a quarter of the
sealed segments by age, rounded up (so at least one; 2 of 5, 4 of 15), and warns through the logger when it would delete more
("age sweep limited by the clock guard"; `quill prune` prints it on standard error and exits 0);
`quill prune -force` lifts the limit; `quill check` flags records dated more than a day in the
future or earlier than their predecessor.

The age sweep deletes a prefix, so one sealed segment whose newest record is stamped in the future (a clock that
stepped ahead and was corrected) keeps itself and every newer segment until the clock reaches the stamp, however
old they are. The caps (`MaxBytes`, `MaxRecords`) still bound the log. A pass that stops at a segment stamped more than a day
ahead warns through the logger, naming the segment; `quill check` reports the record.

## 7. The `quill` command

`quill [-dir DIR] <verb> [flags]`; the flags of every verb are in [reference.md](reference.md).

| Verb | Lock | Beside a live writer |
|---|---|---|
| `cat`, `read`, `tail`, `stat`, `ls`, `consumers`, `check` | none | yes; shows records up to `quill.tail` |
| `commit`, `register`, `unregister`, `archive` | the consumer's | yes |
| `append`, `roll`, `prune`, `recover` | the writer's | no: `ErrLocked` |

Of the four, only `append` creates a log: at a `-dir` that does not exist or is empty it makes one, as `Open` does, and takes the first record. `roll`, `prune` and `recover` act on an existing log: at a `-dir` with no `quill.format` (a mistyped path) they print `quill: not a log: <dir> has no quill.format`, exit 1 and create nothing, as the read verbs do.

Exit codes: 0 ok, 1 error, 2 usage, 3 `check` found the log UNHEALTHY or `archive` found the source had lost records (the run still copied what exists). `ls` skips a subdirectory that is not a log
(no `quill.format`) and lists every log it can open; a log it cannot open or read (a damaged or future `quill.format`, no
permission) gets a row ending in `error: ...`, and `ls` exits 1 after listing the rest, as `stat` and `check` do for that log.
Only `append` (the records) and `help` (a verb name)
take positional arguments; any other verb given one, as in `quill read 100` or `quill tail 5 -json`, is a usage error (exit 2,
"unexpected argument ...; flags go before arguments"): flag parsing stops at the first positional, so the flags after it
would otherwise be dropped. `append` exits 1 when closing
the log fails (with `-sync=false` that close is the only flush that can report an error); with
`-sync=true` the records were already durable, so it still prints the last sequence to stdout
beside the error on stderr. `roll` and `prune` likewise exit 1, with the error on stderr, when closing
the log fails after the pass succeeded (`prune` has printed its counts line by then). A verb that could not write its output to standard output (a full disk,
a closed descriptor) exits 1 instead of 0. Offline verbs run with
background maintenance off and never rewrite `quill.policy`; `append`, `roll` and `recover` take
the segment size from it (the 1 GiB ceiling when it cannot be read), so they do not seal a large
active segment; `roll` seals regardless. `prune` needs `quill.policy` or at least one retention flag
and refuses otherwise; like the others it opens without the policy's `ProtectedConsumers`, so a running
consumer does not block it. It prints `segments`, `bytes` and `first` before and after, and `dropped N`,
the segments the pass deleted while a consumer's offset was below them; each such consumer is named on
standard error. When the pass did less than the policy allows it says why on standard error: "cannot delete
segment" (an unlink failed; `Prune` returns the error and `quill prune` exits 1 after printing its counts, and
`Err()` keeps it until a later pass deletes the segment), "age sweep limited by the clock guard" (exit 0; `-force`
lifts it), "age sweep stopped at a segment stamped in the future" (exit 0) and "consumer file could not be read"
(exit 0; the age sweep keeps every segment until the file is fixed or removed, and every pass says so). Numeric flags (`commit -offset`, `prune -max-bytes`,
`-min-free-bytes`, `-max-records`, `read`/`cat`/`tail` `-from` and `-limit`, `tail -n`, and
`@<unix seconds>`) take plain decimal integers, so `010` is ten; `1e6`,
`1M`, `0x10`, `1_0`, `-1` and trailing text are usage errors, not guesses.

```
quill cat -dir orders -json | jq .seq          # newline-delimited JSON, lossless
quill tail -dir orders -n 20 -follow           # like tail -f
quill read -dir orders -since 2h -limit 100    # RFC 3339, @unix, or a duration ago
quill commit -dir orders -name billing -time 2026-10-28T14:00:00Z
quill check -dir orders && echo healthy
quill archive -dir orders -to /backup/orders
```

**Output.** Text lines are `seq`, the RFC 3339 time, and the payload quoted with control
characters escaped (binary payloads as `b64:…`). `-json` prints one object per record:
`seq`, `at`, `time`, `len`, and exactly one of `payload` (valid UTF-8) or `payload_b64`;
`-b64` forces base64. Both forms are lossless and never pass a raw control character (C0, DEL or C1) or
invalid UTF-8 to the terminal. The same holds for names read from the disk: in text output `ls`, `consumers`,
`check` and error messages print a log, consumer or file name (and a path built from one) as it is when every character is
printable, and quoted with the same escapes (`"a\nb"`, `\x1b`, `\u202e`) when it is not, so a hostile file name cannot add a
line or a terminal sequence. A plain name looks as it always did. `-json` is unchanged.

`stat`, `check` and `consumers` also take `-json`: lowercase keys throughout, and a list is always an array (`"holes":[]`, never `null`; a hole is
`{"from":N,"to":N}`). The schemas are in the scope document, "Readable output".

## 8. The files

```
orders/
├── 0000000000.log      segments: ascending ids, never reused; the newest is active
├── quill.tail          {"seq","seg","off","maxseg"}: the durable position
├── quill.format        "quill/1 <log id>"
├── quill.policy        the last writer's settings, for tools
├── quill.lock          the writer's flock
└── consumers/
    ├── billing         "<log id> <next> <protect>"
    └── billing.lock    held while a process has the consumer open
```

Segment files are binary: inspect them with `quill cat`, never with `cat`. Every other file
is one line of text or JSON. Temp files are `.tmp-*`; the writer removes those older than an
hour when it opens the log, and `Archive` does the same in its destination before it copies.

## 9. Operations

**Backup.** Copy `quill.tail` first, then the segments in id order, then `consumers/`. A
restored log keeps its log ID; a consumer ahead of the restored head fails with `ErrAhead`
and must be rewound, and it must be rewound before the writer is restarted: once the head grows past a stale offset the
consumer resumes mid-log, at sequences that name different records than before the restore, and nothing flags it any more.
The writer's `Open` logs one `Warn` through `Config.Logger` for each consumer that is ahead of the head ("consumer offset
is ahead of the head; rewind it with quill commit", with `consumer`, `offset` and `head`), which is the signal if you forgot; the
offline verbs do not print it. List them with `quill consumers -dir <log>` (the ones ahead carry the note "ahead of the head"), then for each
type `quill commit -dir <log> -name <consumer> -offset N`, with N from `first` to `Head + 1` (`-time <RFC 3339>` instead of
`-offset` picks the first record at or after that time: a time before the first record gives `first`, one after the last
gives `Head + 1`). It opens the consumer for the purpose and keeps its protect flag. A consumer that is ahead of the head protects no segments, so a
writer running with caps can delete the sequence you chose before `commit` runs, and the commit then fails with `ErrOffsetRange`: read `quill stat -dir <log>` just before committing, or
rewind to the first retained sequence (the `first` that `stat` prints) or by `-time`, either of which is always a valid target. That holds for the archiver's `_archive` consumer too:
`quill commit -dir <log> -name _archive -offset N` rewinds it, or `quill unregister -name _archive` removes it (the next
`quill archive` recreates it at the first retained sequence, or at 1 when a damaged oldest segment left a gap below it). A naive copy that takes the marker after the
segments can name data the copy lacks.

**Archiving.** `quill archive -dir <log> -to <dest>` copies each sealed segment not yet
copied, verifying every CRC, and commits the `_archive` consumer, which is protected. The `_archive` offset belongs to the
source log, not to a destination: a new or empty destination receives only the records at or above it, and the run exits 0.
A destination archived after the consumer has advanced (an earlier run copied to another destination) is therefore a suffix
of the source, not a complete copy, and `quill check` on it reports it healthy; to copy more, rewind the consumer with
`quill commit -dir <log> -name _archive -offset N` (the `first` that `quill stat` prints copies the whole retained source)
and archive again. Archive into one standing destination per source. A segment that runs into the next one's sequences (two overlapping segments, which `Open` refuses too) stops the verb with `ErrCorrupt` before that segment is copied. The
destination is itself a log that `OpenReader` opens; the verb refuses the source log as its own
destination and a destination that has a live writer (`ErrLocked`). The destination must be a new or empty directory (a
`.tmp-*` file or `quill.lock` does not count) or one that is already an archive (it has a `quill.format`); a
directory that holds anything else and has no `quill.format` is refused with `ErrNoLog` and left untouched, so a
mistyped `-to` cannot replace an unrelated file. Because `quill.lock` is always 0600, an archiver in a different account than the destination's writer cannot open that lock; the verb treats the permission error as a writer that may be live and refuses (the error is both `ErrLocked` and a permission error), so archive into a directory the archiver's own account owns. A destination directory it cannot search at all is refused with a plain permission error that names the directory and claims no writer (it is not `ErrLocked`), and a destination that holds anything but a log is refused with `ErrNoLog` saying it is not empty and that Archive needs a new or empty directory or an existing archive. To archive the newest records, `quill
roll` first (or `Roll()` in the writer). Under disk pressure the caps still win, so an
archive can fall behind; `Stat().Dropped` counts the loss. A source that lost segments gives
an archive with the same hole. The run that finds the hole copies what exists, prints
`the source lost records N to M; the archive holds the same hole` on standard error, and exits 3 (the
code `check` uses for UNHEALTHY; the library returns a `*GapError`), so `quill archive && notify-ok` does not
report success for an incomplete archive. It finds it two ways: the `_archive` offset is below the source's first
retained sequence, or the destination's `quill.tail` names a last record below the first segment the run copies, which
is how it shows after `quill unregister -name _archive` recreated the consumer at the first retained sequence (at 1 when a damaged oldest segment left a gap below it). The
run that reports the hole commits past it, so the next run has nothing to copy and exits 0 (unless it copied nothing, as when every sealed segment is damaged: nothing is committed then, and every run exits 3 until `quill commit -name _archive -offset N` goes past the gap); `quill check -dir <dest>`
keeps reporting the hole (exit 3).

A rerun after rewinding (`quill commit -name _archive -offset N`) verifies the segments it meets again. A destination segment of the
same size as the source's is read and CRC-checked: a damaged one is replaced by a fresh copy, a healthy one is left as it is, so
re-copying a damaged archive is a rewind and a rerun. The cost is one sequential read of each such destination segment, in
addition to the read of the source segment that is verified on every run; a run with nothing to copy reads no destination file.

**Moving, renaming, deleting a log** are directory operations done while no writer holds it and no reader or consumer has it open. A handle opened before a directory was replaced by another log
refuses `Commit`, and gives up `ReadWait` and `Follow` when they wait, with `ErrWrongLog`; plain reads do not re-check, so close and reopen.

**Filesystems.** Supported: a local POSIX filesystem with working `flock`, atomic `rename` and
honest `fsync` (ext4, xfs, btrfs, APFS). Not supported: NFS, SMB, FUSE; bind mounts and overlay
filesystems at the operator's risk. Linux and macOS only.

**Checking.** `quill check` verifies every sealed segment whole (a sealed segment with no frames, for
example one emptied by a tmp cleaner or a bad restore, is damage and makes it UNHEALTHY; only the
active segment may be empty) and the active one up to the
marker, finds holes, unparseable consumer files (or a `consumers/` directory that cannot be listed) and clock anomalies, warns of a consumer ahead of the head or behind the first retained sequence, and reports UNBOUNDED
when neither cap is set. Exit 3 means UNHEALTHY. `check` takes no lock, so a sealed segment that
retention deletes while it scans is skipped, not reported; one already missing from the listing is a hole.

## 10. Recovery and the manual corruption procedure

A writable open (`Open`, or `quill recover`) performs recovery: it verifies every sealed
segment, parses the newest segment forward from the marker's offset, truncates a torn tail,
and republishes `quill.tail`. A torn tail after a crash needs nothing else.

Whenever that truncation removes bytes, the open reports it: a `Warn` record on `Config.Logger`
("cut the newest segment past the marker", with `segment`, `offset`, `bytes` and `kind`), and on
standard error from every CLI verb that opens the log writable (`append`, `roll`, `prune`,
`recover`), in the same words. `kind=torn` means the first rejected frame was incomplete, the
routine crash tail. `kind=damaged` means it was complete-length but failed its CRC or sequence
check: bit rot, or damage between a stale marker and the end of the segment, where the cut may
have removed an acknowledged record (decision 39). The verb still exits 0, and the cut is not
repeated on the next open, so keep the line if you need to know.

`Open` fails with **`ErrCorrupt`** when the newest segment is damaged below `quill.tail.off`,
when the marker names a sequence above the last valid frame, when two segments overlap, or
when `quill.tail` or `quill.format` is malformed, over 1 MiB, or not a regular file (a
directory, a FIFO, a device or a symbolic link). A reader meets the same overlap as
`ErrCorrupt` once it has scanned the segment that runs into the next, and `quill check`
reports it as an overlap. Damage in any sealed segment, the newest's
predecessor included, does not fail `Open`: the segment becomes a hole, `Err()` names it until retention deletes it,
`quill check` reports it, and reads stop before it with a gap. Retention deletes a damaged sealed
segment like any other, and with it the file and the hole, the only artifacts of the damage, after
which `Stat().Holes` is empty and `quill check` is healthy; so the pass logs the deletion at `Warn` as
"deleted damaged segment" (`segment`, `last`, `damage`) on `Config.Logger`, and an offline `quill
prune` prints the same record on standard error. Keep that line if you need to know. There is no `repair` verb in
v0.1.0 (decision 64). The procedure:

1. Stop every process that holds the log.
2. `quill check -dir <log>`: for the damaged file it prints the byte where the damage starts
   and the end of the last valid frame. For a damaged newest segment, also note the last
   valid sequence now, before you change anything: `quill cat -dir <log> 2>/dev/null | tail -n 1`
   prints the last record before the damage, and the number on its line is the sequence.
   (After step 4 `quill cat` refuses the log, so it can no longer tell you.)
3. Decide. For the **newest** segment, truncating it at the end of the last valid frame drops
   every record after that point; those sequences are lost for good and will be reassigned to
   new records, so any consumer that acknowledged them upstream has duplicates ahead. For a
   **sealed** segment, truncation is not an option (its range would then overlap nothing and
   the records behind the damage are intact): restore the file from a backup or an archive,
   or accept the hole and move the consumers past it with `quill commit`.
4. For the newest segment: `truncate -s <end of last valid frame> <log>/<segment>.log`.
5. Rewrite `quill.tail` by hand. It still names the old, longer length, and `Open` refuses a
   newest segment shorter than the marker's `off`, so `quill recover` fails until you do.
   Keep `seg` and `maxseg` as they are, set `off` to the length you truncated to and `seq` to
   the last valid sequence from step 2 (`0` if no frame is left), keeping the one-line form
   and the trailing newline:
   `{"seq":<last valid sequence>,"seg":<newest segment id>,"off":<truncation length>,"maxseg":<highest id on disk>}`.
6. `quill recover -dir <log>`, which reopens the log, verifies, and republishes the marker.
7. `quill check -dir <log>` again, then restart the writer.

A damaged `quill.tail` or `quill.format` is refused outright; restore them from a backup (the
marker can be rebuilt by hand from the format in [format.md](format.md), with the fields as in
step 5: `seq` is the last valid sequence, `seg` the newest segment, `off` its length, `maxseg`
the highest id on disk).

## 11. Limits

| Limit | Value |
|---|---|
| Payload | 64 MiB − 32 bytes |
| Batch of more than one record | `MaxBatchBytes`, 16 MiB by default, checked as `len(payload) + 34` per record (about 479,000 one-byte records) |
| Segment file | 1.5 GiB (`SegmentBytes` at most 1 GiB) |
| Consumer name | `[a-z0-9_-]`, 1 to 200 characters |
| Sequence | 2^64 − 2, then `ErrSeqExhausted` |
| Timestamps | whole seconds, the writer's clock |
