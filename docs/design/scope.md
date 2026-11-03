# Quill — scope and roadmap

## One-line purpose

An embeddable, append-only log for Go on the standard library alone: **a log is a directory** with one
assigned writer and any number of consumers, in the same process or another — one lane per topic and
nothing more than that needs.

## Users

- **Go programs** that want a durable, ordered, replayable log with named consumers that resume where
  they stopped, and no broker to run.
- **Pipelines of processes**: one process writes a log, another starts consuming it lazily, possibly
  while the writer is still running.
- **Operators** who inspect, check, recover or prune a log from the command line.

It is not a message broker: no network protocol, no replication, no consumer groups (see Non-goals).
"The SQLite of commit logs" is closer than "a lightweight Kafka".

## The model in eight lines

1. A **log** is a directory holding one topic. `quill.Open(Config)` returns a `*Log` and takes an exclusive
   lock on it: that process is the log's **one assigned writer**.
2. **Multiple topics are multiple logs**, opened by one process or by several. Each has its own lock,
   buffer, state, retention and flush goroutine, so topics never share a lock.
3. Records are numbered from 1 per log. Sequences are **contiguous**: no gaps from the first retained
   record to the head, including after a crash; a damaged or missing segment is reported as a gap, never skipped.
4. A log is rotating CRC-framed segment files; file order is sequence order.
5. A **consumer** is a name plus a committed offset, stored in its own file in the log directory and
   written only by that consumer's process.
6. In-process reads see only flushed records. A **`Reader` in another process** sees records only up to
   `quill.tail`, a marker the writer publishes after each fsync.
7. Retention deletes whole sealed segments by age and by caps. It does not wait for consumers unless a
   consumer is **protected** (decision 57).
8. The frame is a fixed, documented format (`docs/format.md`, written in phase 1), frozen by
   golden tests.

## Scope — v0.1.0

### Concepts

| Term | Meaning |
|---|---|
| Log | A directory with one topic's records; one writer at a time |
| Record | `Seq` (per log, from 1), `At` (Unix seconds), `Payload` (opaque, under 64 MiB) |
| Offset | A `seq`; reads return `seq >= offset`. A consumer's committed offset is the **next sequence to read** (decision 36) |
| Consumer | A name (`[a-z0-9_-]`, up to 200 characters, a leading underscore reserved, except the archiver's `_archive`) plus a committed offset; no ownership or rebalancing |
| Protected consumer | A consumer whose unread segments retention keeps; unprotected is the default, and the archiver is protected (decision 57) |
| Segment | One log file; the newest is active, older ones are sealed and deleted whole by retention |
| Tail | `quill.tail`: the durable position (seq, segment, byte offset) and the highest segment id used |

### Library API (package `quill`, module `github.com/netstar-labs/quill`)

| Call | Purpose |
|---|---|
| `Open(Config) (*Log, error)` / `(*Log).Close()` | Open (taking the directory lock) and flush-and-stop; `Close` returns the final flush's error; a read in flight at `Close` finishes, and later calls return `ErrClosed` |
| `Append(payload)` / `AppendBatch(payloads)` | Buffer records; durable after the next flush. A batch is one lock acquisition and lands in one segment. On an error the sequence returned (the first, for a batch) is the one assigned when the failure came after assignment (the inline flush's fsync failed, or `SyncAppends` could not make the records durable) and 0 when nothing was assigned (`ErrClosed`, a failed log, a refused batch, `ErrPayloadTooLarge`, `ErrSeqExhausted`, the buffer or rotation bound) |
| `Flush()` | Write buffered records, fsync, then publish `quill.tail` |
| `Read(offset, limit)` | Flushed records with `seq >= offset`, ascending, payloads copied: at most `limit` records; it stops once at least `MaxReadBytes` have been returned, each record counting its payload plus 48 bytes of overhead (so the total can exceed it by up to one record, and a read of empty records is bounded too), but always returns at least one if available. Returns a `*GapError` where records are unavailable (decision 38) |
| `ReadWait(ctx, offset, limit)` | As `Read`, blocking until more is flushed when caught up; on a failed log it returns `ErrFailed` at the head instead of waiting (records that exist stay readable) |
| `Scan` / `ScanAll` | Zero-copy variants, globally ordered; offset 0 and `ScanAll` start at the first retained sequence, or at 1 when a damaged oldest segment is a hole from 1, which they report as its `GapError` (decision 38) |
| `Records(ctx, offset)` / `Follow(ctx, offset)` | Go iterators (`iter.Seq2[Record, error]`): `Records` reads to the current end and stops, `Follow` then waits for more until the context ends, or yields `ErrFailed` once the log has failed and the records that exist are delivered. Payloads are copies (decision 58) |
| `OffsetForTime(t)` | `(uint64, error)`: the smallest sequence whose `At` is at or after `t`, always a valid `Commit` argument; the error covers I/O failure and a damaged segment (decision 59) |
| `Archive(srcDir, dstDir)` | Copy sealed segments to a destination that opens as a log; the library function behind `quill archive` (decisions 28, 67) |
| `Head()` / `Stat()` / `Err()` | Last flushed sequence; `Stat`: first retained sequence, segment, byte and record counts, holes, `Dropped`, stale consumers, the marker's age, `DiskFull` and `FreeBytesAtLastPass`; `Log.Stat` alone adds `BufferedRecords`, `BufferedBytes`, `FlushFailing` and `LastFlushOK` (acknowledged records not yet durable, whether flushes are failing, and the time of the last flush that completed (a write and publish, or a landed marker publish), which is old on a quiet healthy log: alert on `FlushFailing`); `Err()`: the failed state or the sticky load error of a damaged sealed segment, joined with every background error still failing (each clears when the operation that set it succeeds, the load error when retention deletes its segment) |
| `Roll()` | Seal the active segment now, so it can be archived or reached by retention |
| `Sync(ctx, seq)` | Signal the flusher and block until `seq` is durable, sharing one fsync with every other waiter (group commit). An error means unknown, not lost (decision 42). `Append` then `Sync` is a durable write for one record; `AppendBatch` then `Sync(last)` for a batch |
| `(*Log).Consumer(name, ConsumerOptions)` | Take a consumer handle, creating its offset file at the first retained sequence if absent (at sequence 1 when a damaged oldest segment left a hole below it, so the first read reports the gap) (`Protect`: `ProtectAsStored` by default, `ProtectOn` or `ProtectOff`; a protected consumer's unread segments are kept by retention): `Offset()`, `Commit(next)`, `SeekToTime(t)`, `Records(ctx)`, `Follow(ctx)`, `Lag()`, `Close()`. Holds the consumer's lock. `Close` releases only that lock: the read methods ignore it (a blocked `ReadWait` ends with its context or when the `Log` or `Reader` closes), and `Commit`, `SeekToTime` and a second `Close` return `ErrClosed` (decision 41) |
| `DeleteConsumer(name)` / `Consumers()` | Administration; a missing handle's file is removed or listed |
| `OpenReader(dir, ReaderOptions) (*Reader, error)` | Lock-free reader for **another process**: `Read`, `ReadWait`, `Scan`, `ScanAll`, `Records`, `Follow`, `OffsetForTime`, `Head`, `Stat`, `Check`, `Consumer(name, opts)`, `Close`. The poll interval (default 50 ms) is a `ReaderOptions` field. Never reads past `quill.tail`; re-lists segments itself, so there is no `Refresh` |

Errors, matched with `errors.Is`: `ErrPayloadTooLarge`, `ErrSeqExhausted`, `ErrClosed` (a call on a closed `Log` or `Reader`; on a consumer only `Commit`, `SeekToTime` and a second `Close`), `ErrInvalidName`
(consumer names), `ErrLocked` (log or consumer held by another process), `ErrFutureVersion` (a newer
format marker), `ErrNoLog` (a reader opened a directory that is not a log), `ErrFailed` (the log hit an fsync
error and must be reopened), `ErrCorrupt` (damage below the durable marker), `ErrWrongLog` and `ErrAhead` (a
consumer file that belongs to another log, or is ahead of this one), `ErrOffsetRange`, `ErrBadSeq`, `ErrConfig`;
and `*GapError{From, To}`, matched with `errors.As`. A `*Log` and a `*Reader` are
safe for concurrent use.

### Iterators and seeking by time

```go
for rec, err := range log.Follow(ctx, offset) {      // or reader.Follow, consumer.Follow(ctx)
    if err != nil { /* *GapError, ErrClosed, ctx.Err() ... the sequence ends after an error */ break }
    process(rec)
    if batchDone { consumer.Commit(rec.Seq + 1) }     // commit explicitly; iterators never commit
}
```

`Records` stops at the current end; `Follow` waits for more until its context is cancelled (an in-process `Log`
wakes on each flush, a `Reader` polls). The consumer forms start at the consumer's committed offset. An error is
yielded once and ends the iteration; a `*GapError` is resumed by iterating again from `To + 1`, and an I/O error is retried from the same offset. Breaking out of
the loop releases every buffer and descriptor. Payloads are copies, valid after the loop moves on; `Scan` is the
zero-copy path. The API needs Go 1.23 or later.

`OffsetForTime(t)` returns (with an error for I/O failure or a damaged segment) the smallest sequence whose `At` is at or after `t`, at whole-second granularity (a
record stamped in the same second as `t` counts). A time before the first retained record returns the first
retained sequence and a time after the last returns `Head + 1`, so the result can always be committed:
`consumer.SeekToTime(t)` is `Commit(OffsetForTime(t))`, which starts a new consumer "from two hours ago". The search
compares O(log segments) first-frame times and scans one segment, but it starts from a view of all sealed segments, which costs
O(sealed segments) on every call; a `Reader`'s first call also opens every sealed segment to read its first frame. If timestamps are not monotonic (a clock stepped back; `check` flags it) the
result is a valid offset but may not be the first record at or after `t`. CLI: `-since <time>` on `read`, `cat` and
`tail`, `-time <time>` on `commit`.

### Configuration

| Field | Default | Notes |
|---|---|---|
| `Dir` | required | The log's directory, created if missing. Its base name is the topic name. A symbolic link to a directory is followed, as `Dir` or above it, by `Open` as by `OpenReader`; a link that leads to anything else is `ErrNoLog`. A relative `Dir` is made absolute at `Open` (and a relative `OpenReader` directory at `OpenReader`), so a later `os.Chdir` does not move the log |
| `Retention` | `24h` | Whole sealed segments at least this old are deleted (compared in whole seconds, so a newest record exactly this old is expired). `0` means the default; negative disables expiry. **Set it longer than the worst delay of any consumer** |
| `PruneInterval` | `min(Retention, 1m)`, ≥ 1 s | Sweep period; one minute when `Retention` is negative |
| `FlushBytes` / `FlushInterval` | 64 KiB / 250 ms | Buffer size and background flush period |
| `SyncAppends` | `false` | When true, every `Append` and `AppendBatch` returns only once its records are durable, as if it called `Sync` on its last sequence. A failed flush returns the error to the caller; the records stay buffered and may still become durable, so an error means unknown (decision 37) |
| `MaxBufferBytes` | 16 × `FlushBytes` | Applies only while flushes are failing: then an append is accepted only into an empty buffer or within this limit, and otherwise fails fast (decision 69). At least `FlushBytes`: `Open` refuses a smaller value with `ErrConfig`, so the bounded-disk formula stays exact |
| `SegmentBytes` | 8 MiB | Rotation size; at most 1 GiB |
| `SegmentMaxAge` | `Retention` | Rotates a quiet active segment so retention can delete it; the default follows `Retention`, so it is never when `Retention` is negative, but an explicit positive value rotates whatever `Retention` is, and a negative one is never; a no-op on an empty segment |
| `MaxBytes` | 0 (off) | Cap on this log's segment bytes, enforced at every segment seal and every prune pass: oldest sealed segments go first, the active segment is never deleted, and it overrides consumer protection. `Open` refuses a value below twice `SegmentBytes` |
| `MinFreeBytes` | 0 (off) | Free-space floor on the log's filesystem, measured with `statfs`: below it, the log deletes its own oldest sealed segments, and appends fail with an error wrapping `ENOSPC` if nothing is left to delete; that state, and a failing `statfs`, are reported by `Err()` and `Stat().DiskFull` |
| `MaxPayloadBytes` | 64 MiB − 32 B (the ceiling) | Largest accepted record; lower it to bound a single record. An empty buffer accepts any single record up to this, whatever `MaxBufferBytes` says |
| `MaxBatchBytes` | 16 MiB | Bound on an `AppendBatch` of more than one record, checked as the sum of `len(payload) + 34` per record (an upper bound of the encoded size, so about 479,000 one-byte records at the default); a larger one is refused (decision 46). A single record is bounded by `MaxPayloadBytes` alone, so it may exceed this |
| `MaxReadBytes` | 16 MiB | The size of a batch: a `Read` stops once at least this many bytes are returned, each record counting its payload plus 48 bytes of overhead, so it bounds the size of a batch and not the sum of the payloads (so it can exceed it by up to one record), and returns at least one record. `Records` and `Follow` batch by the same bound |
| `ProtectedConsumers` | none | Names the writer creates, protected, at the first retained sequence (at sequence 1 when a damaged oldest segment left a hole below it, decision 38) when it opens the log, so a lazy consumer is protected before its process starts (decision 67). A file that already says protected is left alone, so the writer can restart while that consumer runs; switching an unprotected file on needs the consumer's lock |
| `Logger` | nil (silent) | A `*slog.Logger` for retention deletions (`Info`, or the `Warn` below for a damaged segment), for a `Warn` when a deletion finds a segment already gone from the directory, for the `Warn` that names each consumer whose offset a deleted segment passes (decision 12), for the MinFreeBytes records (a `Warn` when the free space cannot be read, a `Warn` when it is below the floor with nothing left to delete, an `Info` when it is back above), for stale consumers, for the `Warn` for a consumer file that cannot be read (the age sweep then keeps every segment) and for a staleness probe that fails, for the `Warn` when age rotation is skipped because the flush failed and for the bulk-expiry warnings (the age sweep stopped at a segment stamped in the future, or limited by the clock guard). The deletion records and the MinFreeBytes records are written by a prune pass and also at every seal, because the caps run at each seal (decisions 24 and 25: a `Roll`, or any flush that rotates to a new segment (an `Append`'s inline flush, `Flush`, `Sync`, `Close` or the flusher's); a `Prune` whose age rotation seals runs them at the seal and again in the pass); the stale-consumer, unreadable-consumer-file, failed-probe, skipped-age-rotation and bulk-expiry records come from a prune pass only. A record about a condition is written again by every pass that finds the condition again, and the `Warn` for an unreadable free space is written by every pass and every seal whose `statfs` fails (six failing passes or seals write six); only the below-floor `Warn` and the back-above `Info` are one per transition, and a deletion is one record when it happens (plus a `Warn` for each consumer it passes; a segment already gone gives its `Warn` and then the same deletion record, so two), for the `Warn` when `quill.policy` cannot be written and the `Info` when a retry succeeds (decision 29), for one `Warn` when a flush, marker publish, rotation or prune unlink starts to fail and one `Info` when it recovers (decision 60), for the one `Error` it writes when it becomes failed (decision 60), for the `Warn` a prune pass gives when it deletes a damaged sealed segment (decision 98), for the `Warn` that names each consumer whose offset a deleted segment passes ("deleted segment past a consumer's offset; it has lost records", with `consumer`, `offset`, `segment` and `last`; decision 12), for the warning `Open` gives when it cuts the newest segment past the marker (decision 39), and for the `Warn` `Open` gives for each consumer whose offset is above `Head + 1` ("consumer offset is ahead of the head; rewind it with quill commit", with `consumer`, `offset` and `head`; decision 36). Its handler, `Enabled` included, runs synchronously with the log's lock held for every record above, so it must neither call the `Log` nor block nor take a lock that a caller of the `Log` may hold; hand the record to another goroutine and return; the warnings `Open` itself gives (the cut newest segment, a `quill.lock` it cannot tighten, an ahead consumer, and a first `quill.policy` write that fails) run before it returns, with no lock held (decision 50) |
| `Perm` | 0600 (directories 0700) | File mode; every state file is opened without following symlinks (decision 51). State files are exactly `Perm`; the writer's segments, consumer lock files and directories go through the process umask and can be narrower (`Perm` 0660 under umask 022: `0640` segments, `0750` directories) |
| `MaxRecords` | 0 (off) | Keep roughly the newest N records, segment-granular like every cap; the record count is exact because sequences are contiguous, and like the other caps it overrides consumer protection (decisions 33, 57) |
| `ConsumerTTL` | 0 (off) | A **protected** consumer that has been **behind** for this long, counted from its last commit or from when it fell behind, whichever is later, stops protecting segments. Its offset file is kept (decisions 31, 45) |


### On disk

```
orders/                        one log
├── 0000000000.log             segment files, ascending id, never reused
├── 0000000001.log
├── quill.tail                 durable position and highest segment id, rewritten after each fsync
├── quill.format               format marker and log ID, "quill/1 <id>"
├── quill.policy               the Config the last writer opened with, for tools (decisions 29, 99)
├── quill.lock                 the writer's exclusive flock (advisory, empty)
└── consumers/
    ├── billing                one file per consumer: the log ID, the next sequence to read, and whether it is protected
    └── billing.lock           held while a process has that consumer open
```

Frames: `frame = uvarint(bodyLen) | body | crc32(body)` (CRC 4 bytes, big-endian) and
`body = uvarint(seq) | uvarint(at) | uvarint(len(payload)) | payload`. The topic is not stored.
Normative text goes in `docs/format.md` (phase 1).

Segment files are **binary** (varints, the raw payload bytes, a CRC) and are not meant to be read with `cat` or
`grep`: inspect them with `quill cat`, `quill read` and `quill tail`. Every other file in a log directory is
text: JSON for `quill.tail` and `quill.policy`, one line for `quill.format`, one line per consumer file (the log ID, the next sequence to read and a protect flag).
The payload is opaque, so a caller who appends text stores readable bytes between binary frames. Binary is chosen
over a line-oriented text format because arbitrary payloads need no escaping, a torn write is detected exactly
by length and CRC, and framing costs about 14–18 bytes per record.

### Command line (`quill`)

Inspect and manage; no server. Verbs: `ls` (logs under a parent directory), `stat`, `read`, `tail`, `cat`,
`append`, `consumers`, `commit`, `register`, `unregister`, `check`, `prune`, `roll`, `archive`, and `recover`
(open writable, repair a torn tail, republish `quill.tail`, close). `roll` and `prune` need the writer's
lock, so they run only when no writer is live; a live writer calls `Roll()` itself. `archive` runs beside a
live writer (see Disk safety and archiving). `ls` shows each log's bytes, segments, retention and caps from
`quill.policy`; it skips a subdirectory that is not a log, and gives a log it cannot open an `error:` row and exit 1. Global flag `-dir` (the log). `cat`, `read` and `tail` open the log with `OpenReader`, so they work
beside a live writer without its lock and show only durable records; they print text by default and JSON with
`-json` (below).

**Locks and exit codes** (decision 53). `append`, `roll`, `prune` and `recover` open the log writable and fail with
`ErrLocked` beside a live writer. `commit`, `register`, `unregister` and `archive` take only the consumer's lock.
`cat`, `read`, `tail`, `stat`, `ls`, `consumers` and `check` take no lock. Exit codes: 0 ok, 1 error, 2 usage,
3 `check` found the log UNHEALTHY or `archive` found that the source had lost records the archive never copied
(it still copied what exists; Stage 2 audit, owner-approved 2026-10-31, A5-29). `append` closes the log before it prints and exits 1 when `Close` fails; under
`-sync=true`, where the records were already durable, it still prints the last sequence to stdout (decision 100). `roll` and `prune` exit 1 when `Close` fails after a successful pass, with the error on standard error (decision 100). `recover` is `Open` (which performs recovery) followed by `Close`. `register -protect` creates a protected
consumer; `read`, `cat` and `tail` take `-since <time>` and `commit` takes `-time <time>` (decision 59).
`prune` ends its line with `dropped N`, the sealed segments the pass deleted while a consumer's offset was
below them, and prints on standard error one `Warn` per such consumer and segment that names it (decision 12). A `prune` that did less
than asked says why on standard error: an unlink that failed exits 1 (the counts line is still printed; the library's
`Prune` returns the error), while the clock guard, a segment stamped in the future and a consumer file that cannot be
read are warnings with exit 0 (decisions 44, 53, 65, 67).

#### Readable output: `-json`

`quill cat -dir <log> -json` writes one JSON object per record, one per line (newline-delimited JSON), in
sequence order; `read` and `tail` take the same flag and encoder. `cat` takes `-from <seq>` and `-limit <n>`.

```
{"seq":42,"at":1793196000,"time":"2026-10-28T14:00:00Z","len":11,"payload":"hello world"}
{"seq":43,"at":1793196001,"time":"2026-10-28T14:00:01Z","len":4,"payload_b64":"AAECAw=="}
```

- `seq`, `at` (Unix seconds), `time` (the same instant, RFC 3339 UTC), `len` (payload bytes), and **exactly one**
  of `payload` or `payload_b64`. Field order is fixed.
- `payload` is a JSON string, used only when the payload bytes are valid UTF-8; any other payload is emitted as
  `payload_b64` (standard base64). `-b64` forces `payload_b64` for every record. Output is therefore **lossless**:
  decoding every line returns the original bytes. Invalid UTF-8 is never passed through the JSON encoder, which
  would silently replace it.
- Control characters in a text payload, including DEL and the C1 controls U+0080 to U+009F, are JSON-escaped, so a
  payload cannot inject lines or terminal escapes. The text (non-JSON) output escapes them the same way.

`stat`, `check` and `consumers` take `-json` too (Stage 2 audit, owner-approved 2026-10-30, A1-57). Every key is
lowercase snake_case, a key listed below is always present unless marked optional, and an empty list is `[]`,
never `null`. The order of keys within an object is not part of the contract; consumers must not rely on it.

- `quill stat -json` prints one object: `head`, `first`, `segments`, `bytes`, `records` (integers),
  `marker_age_seconds` (a number) and `holes`, an array of `{"from":N,"to":N}` objects (sequences `from` to `to`
  inclusive are missing), `[]` when there are none.
- `quill check -json` prints one object: `status` (`"HEALTHY"` or `"UNHEALTHY"`), `unbounded` (a boolean) and
  `problems`, an array of `{"kind","path","offset","intact","detail","fatal"}` objects (`kind` is one of `segment`,
  `hole`, `overlap`, `consumer`, `policy`, `clock`; `offset` and `intact` are integers; `fatal` is a boolean), `[]` when there
  are none. The exit code is 3 when the status is `UNHEALTHY`.
- `quill consumers -json` prints one object per consumer, one per line: `name`, `offset`, `lag` (integers) and
  `protected` (a boolean), plus `error` (a string) when the consumer's file cannot be read, `ahead` (`true`)
  when its offset is past the head by more than one, `behind_first` (`true`) when its offset is below the first
  retained sequence (records from `offset` to `first - 1` are gone, and `lag` counts them), or `stale` (`true`)
  when it is protected and the policy's `ConsumerTTL` would find it stale (decision 45; derived from the files, not
  stored). Each is optional and absent otherwise; the table form says the same in its NOTE column (Stage 2 audit,
  owner-approved 2026-10-31, A5-18).

### Durability: what makes a write survive a crash

Three separate things, easy to confuse:

| Mechanism | What it protects | Not to be confused with |
|---|---|---|
| **`Flush()`** (explicit, every `FlushInterval`, at `FlushBytes`, at `Close`) | The **record**: write plus fsync. `Append` returns *before* this, so an unflushed record is lost in a crash. After `Flush` returns, it is durable | |
| **`Consumer.Commit`** | The **consumer's position**: a file write plus fsync that survives a process crash; after an OS crash it may roll back to an earlier value, never forward, so consumers are at-least-once (decision 40) | A "commit" here is an offset, not a database commit |
| **Consumer protection** (`Protect`) | **Retention only**: whether age-based pruning may delete segments a protected consumer has not read | It has no effect on whether a write survives a crash |

To make a write durable, the producer calls `Flush` (or `AppendBatch` then `Flush`) and treats the record as
safe only when it returns. That is the cost: a flush is two fsyncs (the data and the marker), so one record per
flush is slow and batches amortise it. **Measured** on quill's build machine (macOS, Apple silicon, APFS;
[audits/benchmark-baselines.md](../audits/benchmark-baselines.md)): about 10 ms per flush, so about 100
durable records/s for one record per flush and about 50 K records/s with 512-record batches; a Linux server
with a cheaper `fsync` is several times faster (whether its fsync reaches the physical disk is not known).
Shortening `FlushInterval` narrows the window for callers that do not flush explicitly; it
does not close it. A crash loses at most the unflushed suffix and never a middle (guarantee 3); consumers and
readers never see that suffix (guarantee 2).

**Durability is a setting, not a design commitment.** Per call: `Append` then `Sync(seq)`, or `Flush`. Per log:
`SyncAppends`. The format and every guarantee are identical in each mode; only when `Append` returns changes.
`Sync` is group commit: producers append under the log's lock, release it, and wait for the next flush that
covers their sequence, so concurrent producers share one fsync and the log's lock is not held through it.
A lone producer pays the whole fsync per call; many producers pay it once between them. The default stays
buffered, for throughput and so a caller that does not need durability does not pay for it.

**When a write fails** (decisions 37, 60). A flush writes at the durable offset; on any write error it truncates back
to that offset and keeps the records buffered, and space exhaustion recovers by itself when space returns. An
**fsync** error is fatal: the log refuses every later valid `Append` and `Flush` on the open log, and every `Sync` of a sequence not already durable, with `ErrFailed` until it is
closed and reopened (`Sync` of a sequence that became durable before the failure returns nil; `Sync` of an unassigned sequence is `ErrBadSeq`), and a `ReadWait` or `Follow` at the head returns it rather than waiting, because a retried fsync can report success for data that never reached the disk. An error
from `Sync` or a `SyncAppends` append means *unknown*, not *lost*: the records may still become durable, so a
producer that retries must tolerate duplicates; the append returns the sequence it assigned beside the error (0 when it assigned none), so the caller knows which records to look for after reopening. Failures after the data is durable (creating the next segment,
writing the marker) are retried at the next flush and never rewrite the data, and while the next segment keeps failing
to be created appends are refused at a ceiling under the 1.5 GiB file limit; a directory-fsync error is fatal. An
`AppendBatch` is not crash-atomic: a prefix of its records may survive.

### Disk safety and archiving

A log rolls by itself, with or without consumers: retention and the caps below depend on nothing else.
Consumers matter only when protected, and even then the caps win.

| Goal | Knob | Notes |
|---|---|---|
| Forget after N hours (a TTL) | `Retention` | Per log. Segment-granular: a segment goes when its *newest* record is at least as old as the window (an age equal to it counts), so data can outlast it by up to about a segment (up to about twice the window for a quiet log; the reference's guide says so and marks it as inferred). `SegmentMaxAge` seals a quiet log's active segment so the TTL applies |
| Never exceed N bytes | `MaxBytes` | The real rolling guarantee. A time window alone does not bound size: a fast writer fills the disk inside it (the reference's server default does exactly that, **reported**) |
| Never fill the filesystem | `MinFreeBytes` | Backstop for a shared disk. Measured with `syscall.Statfs` (**verified**: runs on macOS, compiles for linux/amd64 and arm64, and on an ext4 Linux host `freeBytes` equaled `df -B1 --output=avail`). It frees only this log's own bytes, and it reports free space for the whole filesystem, so another application or log can use the room first |
| Keep the newest N records | `MaxRecords` | For uniform records, where "the last million events" is the natural unit. `MaxBytes` stays the hard disk guarantee |
| Bound one record | `MaxPayloadBytes` | Optional |

**Bound.** With `MaxBytes` set, a log's segment bytes stay within `MaxBytes + SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)`:
the cap is checked at every seal (an in-memory sum), where the reference checks only on a timer and so can
overshoot by a whole interval of arrivals (**reported**), and a segment passes `SegmentBytes` by the flush
that seals it, and a batch lands in one segment. The last term is the largest single append: a batch of more than one record is
refused when its `len(payload) + 34` sum is above `MaxBatchBytes` (an over-count of its encoded size, which keeps the bound safe), a single record is bounded by `MaxPayloadBytes` alone, and its encoded frame is at most
`MaxPayloadBytes + 34` bytes (the code accounts the payload, three varints of at most 10 bytes and the 4-byte CRC; the true worst case is 32, which the bound covers), so with the defaults the
second term is the larger. The state files and consumer files are small and extra. Memory held while flushes fail is
bounded separately by `MaxBufferBytes`. Defaults leave `MaxBytes` and `MinFreeBytes` **off**, so a log
configured with only `Retention` can fill a disk; `quill check` and `quill ls` mark such a log UNBOUNDED.

**Knowing the size.** Disk use is measured, not estimated: `Stat().Bytes` is the sum of the log's segment sizes
and `Stat().Records` is `Head − first retained sequence + 1` minus the records inside holes, exact because sequences are contiguous
(decisions 10, 32). **Verified** in quill's tests (T17): the byte count equals the bytes of the segment files on
disk across rotations, deletions and a crash. Framing is the length prefix, three varints and the CRC: about
14 to 18 bytes per record (the baselines' 128-byte records cost 145 bytes each). So for planning, `bytes ≈ records × (average payload + about 16)`, and
`MaxBytes ≈ target records × (average payload + 16)`; the overhead creeps up a few bytes as sequence numbers
grow, because they are varints. A record count alone does not bound disk unless record size is bounded, so
`MaxRecords` caps the count and `MaxBytes` caps the bytes, and the worst case under `MaxRecords` is
`MaxRecords × (MaxPayloadBytes + 32)`.

**Several logs on one disk** have no shared budget in v0.1.0 (decision 4 gives up a global cap). Size them so
the sum of every log's `MaxBytes + SegmentBytes` leaves headroom, and use `MinFreeBytes` as the backstop. A
shared budget is the `Set` helper on the roadmap.

**When the disk is full anyway.** A flush that fails keeps its records buffered; past `MaxBufferBytes`
appends fail fast with the flush error; nothing already flushed is touched; the writer recovers on its own
when space returns. The reference derived this from its code but never ran it on a real full disk
(**reported**); quill tests it in a container with a small `tmpfs` mount.

**Managing logs individually.** Everything is per directory: `ls`, `stat`, `check`, `prune`, `roll`,
`archive`; moving, renaming or deleting a log is a directory operation done while no writer holds it.
Policy (`Retention`, caps) is each writer's `Config`; `quill.policy` records what the writer last opened
with, so offline tools and `quill ls` can show and apply it (decision 29). Offline verbs never rewrite it
(decision 99).

**Aging out consumers.** The reference has no consumer expiry: a registered consumer that never commits pins
segments until someone calls `DeleteConsumer` (**reported**, its user guide). Quill adds `ConsumerTTL`
(default off). It matters only for protected consumers. At each prune pass, a protected consumer is
**stale** when it is **behind** (`offset ≤ Head`) and both its offset file's modification time and the `At` of the
record at its offset (or of the first retained record when a cap already deleted it, decision 82) are at least as old as the
TTL (an age equal to the TTL counts, in whole seconds for the record); the prune pass reads that one frame and keeps no state between passes (decision 45). Each
`Commit` writes a new file and creating a handle writes one, so the modification time is the last commit or
creation (**inferred** from temp-and-rename, to be tested); `cp` and `rsync` reset it. A stale consumer stops protecting
segments, is counted in `Stat().StaleConsumers` and logged at every prune pass while stale, and keeps its file, so it
resumes from its offset if it returns and its data still exists. A consumer that is caught up is never
stale, because it pins nothing. The command line shows the same state without a writer's memory (Stage 2
audit, owner-approved 2026-10-31, A5-18): `quill consumers` derives the stale note from the policy's TTL and the
files (a stale consumer's `protected` column still shows its file). Set the TTL longer than the longest time a live consumer can go between
commits while it has work to do.

**Archiving.** Sealed segments never change and their ids are never reused, so archiving is copying.
`quill archive -dir <log> -to <dest>` is **a consumer named `_archive`**: it copies each sealed segment it has
not yet copied, verifies every CRC while copying, writes `<dest>`'s `quill.tail` and `quill.format` so the
destination opens with `OpenReader`, then commits its offset. The offset belongs to the source, not to a destination: a new or empty destination receives only the records at or above it, so a destination archived after the consumer has advanced is a suffix of the source, not a complete copy, and `quill commit -name _archive -offset N` rewinds the consumer to copy more (Stage 2 audit, owner-approved 2026-11-03, L1-12). The archiver is a protected consumer, so
retention will not delete a segment the archiver has not copied, but `MaxBytes` and `MinFreeBytes` still
win, so archiving can fall behind and lose segments under disk pressure (`Dropped` counts them). A source that lost records the archive never copied gives the destination the same hole: `Archive` copies what exists and returns a `*GapError`, and `quill archive` prints the range and exits 3, whether the `_archive` offset is below the first retained sequence or the consumer was removed and recreated (the destination's `quill.tail` is then compared with the first segment copied); the run that reports it commits past it, and `quill check` on the destination keeps reporting it (Stage 2 audit, owner-approved 2026-10-31, A5-29). To archive
the newest records, `Roll()` first. `_archive` is the one reserved name `register`, `unregister`, `commit` and `DeleteConsumer` accept, so after a restore that leaves it ahead of the head it is rewound with `quill commit -name _archive -offset N` or removed with `quill unregister -name _archive`. Moving an archive to other storage is the operator's `rsync` or `tar`.

**A wrong clock** (decision 44). `At` is the writer's wall clock, so a clock stepped forward makes every sealed
segment look expired. A prune pass therefore deletes at most a quarter of a log's sealed segments by age, rounded up (at least one; 2 of 5, 4 of 15)
and warns through the logger when it would have deleted more (`quill prune` prints the warning on standard error and exits 0); `quill prune -force` lifts the limit, and `check`
flags records dated more than a day in the future or earlier than their predecessor. This slows the damage so an
operator can notice; it does not remove the hazard. The opposite hazard is not removed either: the age sweep deletes
a prefix of the sealed segments, so one segment whose newest record is stamped in the future (a clock that stepped
ahead and was corrected) keeps itself and every newer segment, expired or not, until the clock reaches the stamp. The
caps still bound the disk. A pass that stops at a segment stamped more than a day ahead logs a warning naming it.

### Delivery semantics (decision 49)

Consumers are **at-least-once**: commit after processing. Commit before processing for at-most-once. Exactly-once
needs processing that is idempotent on `seq`. A producer cannot know whether an unflushed append survived a crash,
and there is no idempotence key, so it flushes or syncs before acknowledging upstream. Competing consumers that
share one workload are not supported (there are no consumer groups); split the workload across several logs.

### Latency (decision 50)

A record becomes visible after its flush: at most `FlushInterval` (250 ms by default) after `Append`, or at once
with `Sync` or `SyncAppends`. An in-process `ReadWait` then wakes on the flush itself. An external `Reader` adds
up to one poll interval (default 50 ms), because it learns of new data by reading `quill.tail`. Read latency does
not grow with backlog.

### Filesystems, permissions and backup (decisions 51, 53)

Supported: a local POSIX filesystem with working `flock`, atomic `rename` and honest `fsync` (ext4, xfs, btrfs,
APFS). Not supported: NFS, SMB, FUSE; bind mounts and overlay filesystems are used at the operator's risk. On
macOS `File.Sync` issues `F_FULLFSYNC` (**verified** in the Go source), which is why a durable flush costs about
20 ms there against milliseconds on Linux. Directories are created 0700 and files 0600 by default, so a reader
in another user account needs a group arrangement; a process that only reads needs read access, and one that
holds a consumer needs write access to `consumers/`. Consumer files created through a `Reader` or the CLI, and an
`Archive` destination, take the log's mode (read from `quill.tail`), so the writer can read what a consumer in the
group creates; a consumer's lock file is created through the umask (Stage 2 audit, owner-approved 2026-10-30), while the writer's `quill.lock` is always 0600 (A3-5, decision 51). The writer's segments and every directory (the log directory, `consumers/`, an `Archive` destination) are created through the umask as well, so a group arrangement needs umask 002 for the writer, or a `chmod` after `Open` (Stage 2 audit, 2026-10-31, A3-12). Encryption at rest is a non-goal. **Hot backup:** copy
`quill.tail` first, then the segments in id order, then `consumers/`. A restored log keeps its log ID, so a
consumer that is ahead of the restored head fails with `ErrAhead` and must be rewound deliberately (`quill commit -offset N`, which opens it with `ConsumerOptions.Rewind`; Stage 2 audit, owner-approved 2026-10-30), before the writer is restarted: the writer's `Open` logs one `Warn` per such consumer, since once the head grows past the stale offset the consumer would resume mid-log unflagged (Stage 2 audit, owner-approved 2026-11-02, A1-67r).

### Guarantees every release must meet

1. **Order.** Records are returned by every read call, including `ScanAll`, in strictly increasing `seq`
   order, each at most once.
2. **Durable before visible.** In-process reads see only flushed records. A `Reader` never reads past
   `quill.tail`, which is written only after the data it names has been fsync'd. So committing the
   sequence after the highest one read is always safe.
3. **A crash loses a suffix, not a middle.** At most the unflushed tail is lost; a torn final frame is
   found by parsing forward from the marker's offset and truncated on the next writable open (decision 39). One
   residual: damage between a stale marker and the true end of the newest segment looks like a torn tail; the open reports every cut, and says whether the first rejected frame was torn or damaged. Tested with a real
   `SIGKILL` and against a crash-simulating file system (decision 52).
4. **Sequences are contiguous.** After any crash the next sequence is the last valid frame's plus one; a marker ahead of the data fails `Open`. The
   numbers of an unflushed suffix are reassigned, so a caller must `Flush` before acknowledging a
   sequence upstream. No consumer or reader can have seen such a number.
5. **One writer per log, one process per consumer name.** A second writable open, or a second open of the
   same consumer, fails fast with `ErrLocked`. The lock is the file `quill.lock`: if that file is removed or
   replaced under a live writer, the writer finds out at its next flush or prune pass and fails with `ErrFailed`
   (decision 62), but a second writer that opened in between is not stopped, so never delete it.
6. **Corruption is never served silently.** A segment corrupt in the middle fails to load, is never
   modified, is named by `Err()` (until retention deletes it) and `quill check`, and reaches readers as a gap: reads stop before it with a
   `GapError`, never skipping it. A sealed segment with no frames is corrupt in the same way, and `quill check` reports it
   (decision 43; Stage 2 audit, owner-approved 2026-11-03, L1-11).
7. **Allocation is bounded on hostile input:** a declared frame length over 64 MiB, a segment over
   1.5 GiB, an oversize payload or batch is refused before it is allocated. A read is bounded by `MaxReadBytes`, which counts each
   record as its payload plus 48 bytes of overhead, so a log of empty records cannot make one `Read`, `Records` or `Follow` batch
   return every record (Stage 2 audit, owner-approved 2026-11-03, L2-53).
8. **Retention deletes only whole sealed segments**, never the active one, and never reuses a segment id.
9. **State files are atomic and strict:** temp file, fsync, rename; unknown shapes are refused, not guessed.
10. **A stale marker hides, it never exposes.** After a process crash up to one flush of durable records can
    stay invisible to readers, and after an OS crash the marker can be older by any number of flushes, until the
    writer reopens or `quill recover` runs; readers do not guess.
11. **Bounded disk.** With `MaxBytes` set, a log's segment bytes stay within `MaxBytes + SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)`; with `MinFreeBytes` set, a log deletes its own oldest sealed segments before it lets the filesystem
    fall below the floor. A full disk never corrupts a log: appends fail with an error, and flushed records
    stay intact. The floor is only as good as its input: when `statfs` fails the floor is not enforced for that pass,
    and `Err()` says so until a pass can read the free space.
12. **A stale consumer cannot pin the disk.** With `ConsumerTTL` set, a protected consumer that is behind and silent for
    the TTL stops protecting segments, its offset is preserved, and `Stat` reports it.
13. **A failed write never corrupts.** A short write or `ENOSPC` leaves the log as it was; an fsync error makes
    the log refuse writes until reopened, and fail a read that would wait (decision 37).
14. **Reads never skip a gap.** Data removed by retention below the requested offset, and a hole in the middle,
    both surface as a `GapError`; the consumer advances only by an explicit commit (decision 38).
    An I/O error met while reading a segment (including its first frame, and while `Open` verifies the sealed segments)
    is not a gap: it is returned as that error and the read can be retried (decision 79). A sealed segment that
    cannot even be opened is a hole (decisions 94 and 98).

### Performance gate

Quill changes shape, not speed. Phase 4 records a baseline for each benchmark — buffered append, append with a
flush per call, read from the start, read near the tail, read of sealed segments, `ReadWait`, cold start, and
two new ones (several logs flushing in parallel, a `Reader` following a live writer) — on one machine, six
runs, in `docs/audits/benchmark-baselines.md`. A later release may not regress one beyond run-to-run noise.
The recorded baselines on the build machine: `Read(limit 100)` at about 23–39 µs depending on the segment,
a full `ScanAll` at about 21 M records/s over 400 K warm records, and a `Reader` draining a backlog at about
16 M records/s. The marker write and fsync on every flush, which the reference did not have, is about half
of a flush's 10 ms on macOS; the baselines state that cost. Read cost does not grow with the backlog of sealed segments the way a scan would: segments below the offset are skipped without
being opened (a `Reader` still lists the directory and rebuilds its view of the sealed segments on every call, O(sealed segments)),
the sparse index seeks within a sealed segment, and a read stops after `limit` records. A cursorless `Reader` read of the
*active* segment (`Read`, `ReadWait`, `Scan` or a `Consumer.Read` without an iterator) re-walks and CRC-checks that segment
from its first byte to the requested offset, so it costs O(offset within the segment), bounded by `SegmentBytes`: measured at
379 ms for a read at the top of a 343 MB active segment of 3,000,000 records (an audit measurement on a Linux box), and a few
milliseconds at the default 8 MiB segment (an extrapolation, not a benchmark). `Records` and `Follow` carry a byte cursor and
avoid it; a polling consumer should use them. `Commit` now writes a file (one fsync per commit), so consumers commit per batch.

### Platforms

Linux and darwin (arm64, amd64). Other platforms fail to build, on purpose.

## Format stability

The frame and every state file are specified in `docs/format.md` and frozen by golden tests. A change needs a
new format version, announced by `quill.format`, and a migration note. The directory layout above is part of
that contract.

## Non-goals

- A network service, authentication, TLS, or any socket. (A network layer would be a separate program on top.)
- Replication, clustering, failover.
- Consumer groups, partition assignment or rebalancing. A consumer is a name and an offset.
- Sharding, a fixed-width format, or a second on-disk layout.
- A shipper, or any feed for one.
- Transactions, compaction, keys, or deleting individual records.
- A global cap across several logs, and an atomic operation across logs.
- Windows and other platforms.

## Versioning

v0-forever: tags `v0.y.z`, never `v1`, breaking changes bump the minor with a migration note. The **module
version** and the **format marker** (`quill/1`) are separate: a binary reads the marker it knows and refuses a
newer one with `ErrFutureVersion`. The check happens when a handle is opened (and, for a `Reader` or consumer,
again at `Consumer.Commit` and while waiting), not on every read or append, so a process already running keeps
going across a marker bump. A migration note therefore has to require stopping or restarting every writer,
reader and consumer before the new format is written (`docs/format.md`, `quill.format`).

## Roadmap

| Release | Theme | Contents |
|---|---|---|
| **v0.1.0** | Core | Format and recovery, `Log`, `Reader`, consumers, retention and caps, `quill` CLI (including `roll` and `archive`), disk caps, golden and crash tests, benchmark baselines, a crash-simulating file-system seam, docs, examples. Tagged only after the gates are green and the A1 audit's findings are fixed (decision 55) |
| **v0.2.0** | Audited | The complete house audit sequence run against v0.1.0, every finding fixed, every fix validated by a skeptic (decision 55). The measured wins below are candidates for the sequence's verification-and-optimisation pass and land only if it measures them: flush outside the append lock; in-place CRC'd `quill.tail` with no per-flush fsync; batched `Commit`; `.idx` sidecars; a mapped read path for sealed segments or a descriptor cache; a `Set` helper that opens every log under a parent and shares a disk budget; `quill repair` (decision 64); `archive -skip-corrupt` |
| **later (still v0.x)** | Only if needed | Anything the audit sequence turns into requirements |

Each release is gated on: tests green with `-race`, fuzz targets run for a fixed budget, `go vet`,
`staticcheck` and `govulncheck` clean, `go mod tidy` a no-op, and the benchmark baselines recorded. Before the
repository goes public: a `LICENSE`, `CONTRIBUTING`, a changelog and a stated Go-version policy (decision 54).

## Examples

- `examples/log/` — open a log, append, read, commit, restart, resume.
- `examples/topics/` — three logs (`orders`, `payments`, `audit`) in one process, each with its own
  consumers, to show that topics share no lock, buffer or flusher (on one disk they still share the device).
- `examples/pipeline/` — two processes: a writer appends to a log; a separate process opens a `Reader`
  and a consumer, starts late, catches up, and keeps following. Run under `go test` with a helper binary.

## Open questions

None outstanding. **Settled by the owner on 2026-10-28**, with the detail in [decisions.md](decisions.md): a log is
a directory; no feed for a shipper; a durable marker file for external readers, trusted alone with no liveness
probe (decision 8); one offset file per consumer; the frame unchanged from the reference; sequences contiguous
after a crash (decision 10); an offset is the next sequence to read; an fsync error is fatal; reads never skip a
gap; commit once per batch (decision 9); the format marker file (decision 6); no `LICENSE` while private
(decision 22); the `quill.policy` file (decision 29); consumer protection per consumer, with the old log-wide
switch retired (decision 57); `MaxBytes` has no default (decision 24); `ConsumerTTL` off and `MaxRecords` included
(decisions 31, 33); `pread` for reads (decision 15); the quarter-per-pass clock guard (decision 44); Go iterators (decision 58);
seek by timestamp (decision 59).
