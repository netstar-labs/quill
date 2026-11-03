# Quill — build prompt

*A prompt for a Claude Code session that builds this repository. Read [scope.md](scope.md) (what to build),
[review.md](review.md) (what the reference design does and what to leave behind) and
[decisions.md](decisions.md) (the choices already made) first; this file says how. The repository has already
been scaffolded from the house handbook's scaffold prompt (decision 56), and this build starts from that
scaffold.*

---

## Role and ground rules

You are building `github.com/netstar-labs/quill`: a small Go module and CLI — an embeddable append-only
log where **a log is a directory** with one assigned writer and any number of consumers, in the writer's
process or another. Optimise for **small, readable, auditable** code and for the recovery path being right,
not for features. It will be a public repository.

- Standard library only. `go.mod` has **no `require` line**, ever. A check that needs a dependency is
  written out in the file with the reason; stop and ask before adding one.
- Go: the latest stable toolchain, named in `go.mod` (the iterator API needs Go 1.23 or later). Always run Go
  with `GOWORK=off`. `go vet`, `gofmt`, `staticcheck`, `govulncheck` and `go mod tidy` clean at every commit. One
  commit-gate script (`hack/gate.sh`) runs gofmt, vet, build, `test -race`, staticcheck, deadcode, a tidy check
  and the fuzz targets at 20 seconds each (decision 70).
- **Tests first** for every guarantee in scope.md. A guarantee without a test that could falsify it is not
  done. Process-death tests use a real `SIGKILL` of a helper process; **power-loss behaviour is tested against
  the crash-simulating file-system seam** (decisions 52, 61), never assumed from `SIGKILL`, which leaves the page
  cache intact. What neither can show (a real disk's behaviour) is written into
  `docs/audits/untestable-without-x.md` with the reasoning.
- Commit in small steps with conventional messages. **Do not push or create a remote** unless asked. Tag only
  where the release procedure below says to.
- **Do not copy or reference any outside project in code, comments, docs, commit messages, test names, env
  var names or file magic.** Everything is named `quill`. If the person running the build gives you access to
  the existing implementation this design was reviewed from, you may read it and port its behavioural tests
  that still apply, but never name it in anything you write, and do not modify it.
- Add no `LICENSE` while the repository is private (decision 22); one lands before it goes public
  (decision 54).
- When something is ambiguous, prefer the smaller, safer behaviour and record the choice in `decisions.md`.

## Repository layout

The library is the root package so the import path is short. **No file over about 600 lines** (decision 20).
All file I/O in the library goes through `internal/fsx` (decision 52).

```
quill/
  go.mod  README.md  SECURITY.md  CONTRIBUTING.md  CHANGELOG.md          (no LICENSE yet: decision 22)
  doc.go          package comment: the rules to know
  log.go          Log, Config, defaults, validation, Open (creation protocol), Close, errors
  load.go         open: directory scan, parse-forward recovery, tail truncation, hole detection
  holes.go        contiguity: the holes and the first overlap from the good segments' ranges (Open and Reader)
  append.go       Append, AppendBatch, buffer room, MaxBufferBytes, MaxBatchBytes, MaxPayloadBytes
  flush.go        Flush, the flush goroutine, pwrite at the durable offset, failure classes, rotation, Roll, Sync
  read.go         Read, ReadWait, Scan, ScanAll, GapError, segment ranges from first frames (Log and Reader)
  reader.go       OpenReader: lock-free, bounded by quill.tail, lazily built per-sealed-segment index, polls
  iter.go         Records and Follow (iter.Seq2) for Log, Reader and Consumer
  seek.go         OffsetForTime: the time search over segment first frames and sparse-index anchors
  stat.go         Log.Stat, Reader.Stat, Reader.Check
  consumer.go     Consumer handle: lifecycle, per-consumer lock, offset file, Commit, Lag, log-ID check, Protect
  archive.go      Archive(srcDir, dstDir): the _archive consumer behind `quill archive`
  retention.go    prune, rotate by age, caps at every seal, the clock guard, Dropped, protection, ConsumerTTL
  disk.go         statfs free-space reading (a variable, so tests can inject it)
  state.go        quill.tail, quill.format, quill.policy, consumer files: encode, strict decode, atomic write
  codec.go        the frame: encode, decode, CRC
  lock_unix.go    flock helpers for quill.lock and consumers/*.lock
  cmd/quill/      main.go  verbs.go                  (verb table drives help and the reference)
  internal/
    fsx/          fs.go (the interface)  os.go (the real implementation)  crash.go (the crash simulator)
    atomicfile/   temp file, fsync, rename, directory fsync, through fsx
    strictjson/   state-file parsing that refuses damaged values
  examples/       log/  topics/  pipeline/
  testdata/       golden segment fixtures
  docs/           design/  format.md  introduction.md  executive-summary.md  userguide.md  architecture.md
                  reference.md (generated)  audits/
  hack/gate.sh
```

## On-disk format (normative; write it into `docs/format.md` in phase 1)

The **frame is fixed** (decision 3):

```
frame = uvarint(bodyLen) | body | crc32(body)        crc is 4 bytes, big-endian
body  = uvarint(seq) | uvarint(at) | uvarint(len(payload)) | payload

{log}/{id:010d}.log     segment files, ids ascending, never reused
{log}/quill.tail        {"seq":N,"seg":ID,"off":BYTES,"maxseg":ID}
{log}/quill.format      quill/1 <32 hex digits: the log ID>
{log}/quill.policy      the writer's Config, informational, tolerant
{log}/quill.lock        the writer's flock
{log}/consumers/{name}       <log id> <next> <protect 0|1>      one line
{log}/consumers/{name}.lock  held while a process has the consumer open; never deleted
{log}/**/.tmp-*              temp files; ignored by scans; the writer removes those older than an hour
```

Rules that must hold:

- A declared `bodyLen` over 64 MiB is corruption, not an allocation. `MaxPayloadBytes` is at most 64 MiB − 32
  bytes so a legal record's body always fits (decision 69).
- A segment is strictly increasing in `seq`; within the newest segment a frame's `seq` is exactly the previous
  frame's plus one.
- **A segment is sealed exactly when a higher-numbered segment exists**; nothing is written to mark it
  (decision 43). A sealed segment with no frames is corruption.
- **Recovery treats the marker as a lower bound and parses forward** (decision 39). Let the marker name segment S
  at offset X. Every byte of a sealed segment, and every byte of the newest segment below X (below 0 when the
  marker names an older segment), was durable: damage there is **corruption**. The file is never modified, a hole
  is raised for readers, and if it is the newest segment `Open` fails with `ErrCorrupt`. In the newest segment,
  from X (or from byte 0 when the marker names an older segment), parse forward accepting each frame that is
  intact (CRC), within the length limit and sequence-contiguous; truncate at the first frame that is not, and trust
  nothing after it, whatever it contains. If the log holds frames and its last valid sequence is below
  `quill.tail.seq`, fail `Open` with `ErrCorrupt`; if it holds none, the next sequence is `quill.tail.seq + 1`.
  Residual, to be documented: damage between a stale marker and the true end of the newest segment looks like a
  torn tail.
- **Holes.** At load compare each segment's first sequence with the previous segment's last. A discontinuity,
  including one left by a damaged file that cannot be parsed or a segment deleted by hand, is a hole;
  retention deletes only the oldest segments, so a middle hole is always damage (decision 38).
- Segment files are opened without following symlinks (`O_NOFOLLOW`), as regular files only, at most 1.5 GiB;
  every other state file is opened without following symlinks too (decision 51). `AppendBatch` refuses a batch
  over `MaxBatchBytes`. A segment passes `SegmentBytes` by the flush that seals it.
- Every state file is strict JSON or one strict text line, written by temp file, fsync, rename, and refused when
  malformed, except `quill.policy`, which is parsed tolerantly and never blocks `Open` (decision 48).
- `quill.format` is required: a directory with segments and no marker, or a marker newer than `quill/1`, fails
  `Open` (`ErrFutureVersion` for the latter). Creation and interrupted creation are in the write path.

## Write path

1. **Create and open** (decision 62). Create the directory and any missing parents (each fsynced into its parent),
   then create and `flock` `quill.lock` **before** deciding anything (retry briefly, then `ErrLocked` naming the
   lock). The directory is *new* when, ignoring `quill.lock` and `.tmp-*` files, it is empty; creation writes
   `quill.format` (temp, fsync, rename), `consumers/`, the first empty segment and the first `quill.tail`
   (`seq 0, seg 0, off 0`), then fsyncs the directory. A directory with `quill.format` but no `quill.tail` and no
   frames is an interrupted creation: finish it. A directory with no `quill.format` that holds other files is
   refused. Then validate `Config` (`ErrConfig`: `MaxBytes` below twice `SegmentBytes`, and the like), load and
   recover, remove stale `.tmp-*` files, write `quill.policy`, create any `Config.ProtectedConsumers`, republish
   `quill.tail`, and start the log's own flush goroutine.
2. `Append` and `AppendBatch` take the log's mutex, assign sequence numbers under it (consecutive within a
   batch), encode into the buffer and return. A batch is one lock acquisition, lands in one segment, and is not
   crash-atomic. While flushes are failing an append is accepted only into an empty buffer or within
   `MaxBufferBytes`; otherwise one that reaches `FlushBytes` flushes inline (decision 69).
3. `Flush` (explicit, background, at `FlushBytes`, or at `Close`) takes the log's lock and **writes with `pwrite`
   at the durable offset** (`quill.tail.off`), then fsyncs. Failure classes (decisions 37, 60):
   - a `pwrite` error (short write, `ENOSPC`): truncate back to the durable offset, keep the records buffered, retry
     from the same offset; if the truncate fails the log is failed;
   - a data `fsync` error: fatal and sticky. The log enters a failed state and every later valid `Append` and `Flush` on the open log, and
     every `Sync` of a sequence not already durable, returns an error wrapping `ErrFailed` until it is closed and reopened
     (`Sync` of a sequence already durable returns nil; of an unassigned sequence, `ErrBadSeq`);
   - once the data fsync has succeeded the records are durable: release the buffer, let in-process readers see
     them, and never truncate or rewrite that data;
   - a failure creating the next segment or writing or renaming the marker's temp file is retryable: keep appending
     to the current segment, retry at the next flush, report it through `Err()`;
   - a directory `fsync` error is fatal.

   **Rotation** (decision 43), when `SegmentBytes` is reached: after the data fsync, create the new empty segment,
   fsync the directory, then publish `quill.tail` naming the new segment at offset 0 (temp, fsync, rename). `Roll`
   flushes first and does nothing when the active segment has no frames; so does age rotation. Ordinary flushes
   publish the marker without a directory fsync.
4. **Deletion** (decision 65). Retention publishes `quill.tail` (so `maxseg` and `seq` are current), **fsyncs the
   directory**, and only then unlinks segments.
5. `Sync(ctx, seq)` signals the flusher at once and returns when `quill.tail.seq >= seq`, waiting on the same
   flush broadcast `ReadWait` uses and holding no lock. `seq` above the last assigned sequence is `ErrBadSeq`;
   waiters pending at `Close` get `ErrClosed` unless their sequence became durable; an error means *unknown*,
   not *lost* (decision 42). `SyncAppends` is `Append` or `AppendBatch` followed by `Sync` of the last sequence.

## Load and recovery

Scan segments in id order, rebuilding each sparse index (each anchor holds the sequence, byte position and `At`)
and CRC-checking every frame (parallelise across segments). Sealed segments are verified whole; the newest is
recovered by parsing forward as above. Detect holes. Next sequence = the last valid frame's plus one, or
`quill.tail.seq + 1` when there are no frames; the next segment id is above `maxseg` and above every id on disk.
`quill recover` is `Open` followed by `Close`; there is no automatic way past `ErrCorrupt` in v0.1.0: the user guide documents a manual procedure (decision 64).

## Read path

Because sequences are contiguous, a sealed segment's range is `[its first sequence, the next segment's first − 1]`:
read only the first frame of each segment, binary-search the list, and skip segments whose range is below the offset
without opening them; the active segment's range ends at `quill.tail.seq` (decision 47). Within a segment seek by
the sparse index and decode until `limit` records **in total** or `MaxReadBytes` of payload, always returning at
least one record when one exists. Each read is a `ReadAt` into a pooled buffer bounded by the durable length and
decoded in place; the log keeps the active segment's descriptor open, and any other segment is opened, read and
closed (decision 15). `Scan` payloads alias the buffer and are valid only during the callback.

**Gaps** (decision 38): an offset below the first retained sequence, or a position inside a hole, returns
`*GapError{From, To}` (`To` is the last missing sequence, for retention `first retained − 1`); `Scan` and `ScanAll`
stop at a gap with the same error. Nothing is skipped silently. If a segment is deleted between listing and
opening, re-list, then report the gap.

**A `Reader`** (decision 63) takes no lock. Each call reads `quill.tail` (verify it parses; on a transient failure
retry briefly), lists the directory, and treats the marker's segment as active and segments above it as not yet
published. Segments below the marker's are sealed: build each one's first frame and sparse index on first use by one
sequential scan and cache them per segment id, invalidated if the file's identity changes or it disappears. The
active segment is never cached: read its first frame afresh, trust it only if it ends at or below `tail.off`, and
clamp every read by `tail.seq`. Hold descriptors while reading so a concurrent delete cannot break a read. A CRC
failure found in a sealed segment is a `GapError` over that segment's range. `ReadWait` polls `quill.tail` at the
poll interval (default 50 ms). A reader never decides whether a writer is alive (decision 8); it exposes the
marker's age. `Reader.Stat` and `Reader.Check` (decision 66) report only what a reader can know and check the
active segment only up to the marker.

**Iterators** (decision 58). `Records(ctx, offset)` and `Follow(ctx, offset)` on `Log`, `Reader` and `Consumer` return
`iter.Seq2[Record, error]`. They keep a byte cursor, read in batches bounded by `MaxReadBytes`, yield copies of the
payloads, yield an error once and end, and release every buffer and descriptor when the loop breaks. `Follow` waits
on the flush broadcast for a `Log` and polls `quill.tail` for a `Reader`, and returns when the context ends. The
consumer forms start at the consumer's offset and never commit.

**Seeking by time** (decision 59). `OffsetForTime(t) (uint64, error)`: let `target = t.Unix()`; pick the last
segment whose first frame's `At` is **strictly less than** `target` (the first segment if none); find the first record
in it with `At >= target` through the sparse-index anchors (the Reader's lazily built index); if there is none the
answer is the next segment's first sequence, or `Head + 1` after the last. Clamp to `[first retained, Head + 1]`.
Errors: I/O, `ErrClosed`, a `*GapError` for a damaged chosen segment. With non-monotonic times return the position
the search lands on.

## Consumers and retention

- **Offset = next sequence to read** (decision 36). `Consumer(name, opts)` creates `consumers/{name}` if absent
  (file and `consumers/` directory fsynced) at the first retained sequence, takes `consumers/{name}.lock`
  (`ErrLocked` if held), and checks the stored log ID (`ErrWrongLog`) and that the offset is not above `Head + 1`
  (`ErrAhead`). `Commit(next)` accepts `first retained ≤ next ≤ Head + 1` (`ErrOffsetRange` otherwise), writes the
  file by temp, fsync, rename, and updates the handle. `Lag()` is `Head + 1 − offset`. Commit once per batch, not
  per record. `ConsumerOptions.Protect` is `ProtectAsStored` (zero value), `ProtectOn` or `ProtectOff`; commits
  never change the flag (decisions 57, 67).
- A consumer file that cannot be read or parsed protects every segment from the first retained one and makes
  `check` report UNHEALTHY. `DeleteConsumer` and `unregister` take the consumer's lock and delete only the offset
  file; lock files are never deleted. Consumer names `[a-z0-9_-]`, 1–200 characters; a leading underscore is
  reserved and refused by the public API, except `_archive` (Stage 2 audit, owner-approved 2026-10-30, A1-32).
- Retention deletes whole sealed segments whose newest record is older than `Retention`; negative disables the
  age sweep; `0` means 24 h. The active segment is never deleted. **Clock guard** (decision 44): a pass deletes at
  most a quarter of the sealed segments by age, rounded up (at least one), logs a warning through `Config.Logger` when it would have
  deleted more, and `prune -force` lifts the limit. `MaxBytes` deletes the oldest sealed segments first. Each prune
  pass reads `consumers/*` and keeps every sealed segment holding records at or above the slowest **protected**
  consumer's offset; `MaxBytes`, `MinFreeBytes` and `MaxRecords` override protection (decision 57).
- `Dropped` counts segments deleted while some consumer's offset was below them (in memory, reset on restart).
- **Disk bounds** (decisions 24–26, 69). At every seal, and in every prune pass, compute the log's segment bytes
  from memory and delete the oldest sealed segments until under `MaxBytes`; then, if `MinFreeBytes` is set, read
  the free space with `syscall.Statfs` (`Bavail * Bsize`) and keep deleting this log's oldest sealed segments until
  the floor holds or only the active segment remains; then, if the floor still fails, make appends return an error
  wrapping `ENOSPC`. `Retention` is checked in the same pass. A cap may delete up to one segment past its target.
- **`MaxRecords`** (decision 33): in the same pass, delete the oldest sealed segments while the retained record
  count (`Head − first retained + 1`) exceeds it.
- **`ConsumerTTL`** (decisions 31, 45): ignore for protection any protected consumer that is behind
  (`offset ≤ Head`) and whose offset file's mtime and the `At` of the record at its offset (read in the prune pass)
  are both older than the TTL; never delete its file; report it in `Stat().StaleConsumers`. No in-memory timestamps.
- **`Archive(srcDir, dstDir)`** (decisions 28, 67, 68) is the `_archive` protected consumer, opened like any
  consumer (the public name check accepts exactly that name): list sealed segments at or above its offset, copy each to the destination (`O_EXCL` temp, CRC-verify
  every frame while copying, fsync, rename), write the destination's `quill.format` and `quill.tail`, then commit.
  It is idempotent and safe beside a live writer, reports a hole when the source lost segments, and stops with an error at a
  corrupt sealed segment (decision 68).

## CLI

`quill` verbs: `ls`, `stat`, `read`, `tail`, `cat`, `append`, `consumers`, `commit`, `register`, `unregister`,
`check`, `prune`, `roll`, `archive`, `recover` (scope.md). `register` takes `-protect` or `-no-protect`;
`read`, `cat` and `tail` take `-since <time>`; `commit` takes `-time <time>`. Global flag `-dir`. Lock rules and exit
codes are in decision 53: `append`, `roll`, `prune` and `recover` take the log's lock; `commit`, `register`,
`unregister` and `archive` only the consumer's; the rest none and use `Reader`. Exit 0 ok, 1 error, 2 usage, 3
UNHEALTHY. Offline tools that open the log writable run with background maintenance disabled; `prune` needs a parsed
`quill.policy` or explicit retention flags (decision 68). There is no `repair` verb in v0.1.0 (decision 64): `check` reports the damaged file, the byte offset of
the damage and the end of the last valid frame, which is what the documented manual procedure needs. `check` reports a log UNHEALTHY for any segment that fails to load,
a hole, or an unreadable or unparseable consumer file, and UNBOUNDED for a log with neither `MaxBytes` nor
`MinFreeBytes`. `cat`, `read` and `tail` share one record printer with a text mode and a `-json` mode (decision 35):
newline-delimited JSON, fixed field order, exactly one of `payload` (valid UTF-8 only) or `payload_b64`, `-b64` to
force base64, never the JSON encoder's lossy handling of invalid UTF-8. The verb reference is generated from the verb
table, and a test asserts every documented verb exists.

## Tests (required)

Behavioural tests, by area. If you were given the existing implementation (see ground rules), port the ones that
still apply and note each disposition in `docs/audits/test-ledger.md`; otherwise write them from this design.

- **Frame and segment (phase 1):** encode/decode round trips; golden bytes recorded once and decoded back; a torn
  tail at every byte position; corruption in the middle; a frame at or below its predecessor; oversize declared
  length; a crafted tail that tries to force quadratic search; symlink, FIFO and oversize segment files.
- **Append and read (phases 2–3):** batch sequences; a batch over `MaxBatchBytes`; reading from any offset across
  segments; reads while appending; reads racing retention deletes; a segment truncated or deleted under a reader
  gives an error or a short read, never a crash; descriptors across rotation.
- **State files (phase 1):** strict parsing, every malformed shape refused; atomic replacement; ids never reused.

New tests quill must add. The phase in brackets is the first at which the test can be written (decision 70).

- **T27 Simulator self-test** (phase 1). The crash simulator's own model (decision 61) is tested first: an unsynced
  write can be lost or torn at page granularity and reads back as zeros; `fsync(file)` makes data durable but not the
  directory entry; a create, rename-over-existing or unlink is lost or kept until `fsync(dir)`; a truncate may be
  undone; injected short writes, `ENOSPC` and `EIO` surface as errors. Everything below that uses it depends on this.
- **T12 Fuzz** (phase 1) the frame decoder and the state-file parsers (`quill.tail`, `quill.format`, consumer files,
  and `quill.policy`, which must never fail `Open`).
- **T1 Crash property.** Against the crash simulator, with a crash injected at every file-system operation of a
  scenario (exhaustive for short scenarios, random for long ones), and also under a real `SIGKILL`. *Phase 2:*
  appends, flushes, syncs, rotation and `Roll`. *Phase 3:* add consumer commits, pruning and a reader. After
  recovery: `Open` succeeds; the log is a contiguous prefix; it contains **every record whose `Flush` or `Sync`
  returned**, including when the marker rename was lost (the case that decision 39 exists for); it may contain more
  (records written but not yet acknowledged survive a `SIGKILL`, so absence of unacknowledged records is **not**
  asserted); nothing durable is lost or duplicated; a reader's view is a subset of the recovered log; no consumer
  offset is ahead of the log; sequences are reassigned only for records that never became durable.
- **T2 Recovery classification** (phase 2). A stale marker with acknowledged records above it keeps all of them; a
  power-loss hole (zeroed pages) followed by intact frames is truncated at the hole; a payload that embeds a whole
  valid segment, cut inside the embedded bytes, is truncated and the embedded frames are never reached; damage below
  the marker in the newest segment, or in a segment that has a higher one above it, fails `Open` with `ErrCorrupt` or
  raises a hole and leaves the file unchanged; a marker whose `seq` is above the last valid frame fails `Open`, but
  the same marker with **no frames at all** (rolled, everything expired) does not; a sealed segment with no frames is
  corruption; torn tails at every position.
- **T3 Failed writes** (phase 2). With the simulator injecting a short write and `ENOSPC`: the file is truncated back,
  the retry starts at the durable offset, and no partial frame remains. With an fsync failure on the data file or the
  directory: every later call returns `ErrFailed`, a reopen recovers a consistent log, and a record whose `Sync`
  returned an error may or may not be present. With the marker's temp-file write or rename failing after the data
  fsync: the data is **not** truncated or rewritten, no sequence is duplicated, the next flush republishes, and a
  failure creating the next segment keeps appending to the current one.
- **T4 Order** (phase 3). `ScanAll`, `Read` and a `Reader` agree record for record, in sequence order, across many
  segments.
- **T5 Isolation** (phase 2). Two logs in one process: a slow fsync injected on one never delays appends or flushes
  on the other.
- **T6 Writer plus reader in two processes** (phase 3). A writer process appends continuously; a reader process
  follows. The reader never returns a record beyond `quill.tail`; after the writer is `SIGKILL`ed the reader never
  returns a record that the recovered log lacks; after `Open` recovers, it sees everything durable; a reader that
  cached a segment's first frame before a writer crash does not serve wrong sequences afterwards.
- **T7 Gaps** (phase 3). An offset below the first retained sequence returns `GapError{From, To}`; deleting a middle
  segment by hand makes a hole found at load; the consumer resumes with `Commit(To+1)`; `Stat` reports `First` and the
  holes; `ScanAll` stops at a gap; a retention race between listing and opening re-lists; a CRC failure in a sealed
  segment found at read time is a `GapError` over that segment's range.
- **T8 Offsets** (phase 3). `Commit(lastRead+1)` and `Lag` agree on a caught-up consumer (`offset = Head + 1`,
  `Lag = 0`); `ErrOffsetRange` outside `[first retained, Head + 1]`; a log restored to an older state gives
  `ErrAhead`; a consumer file from another log gives `ErrWrongLog`; a rewind works.
- **T9 Consumer lifecycle** (phase 3; the CLI parts in phase 4). Opening a handle creates the file at the first
  retained sequence, and a protected one keeps its unread segments before any commit; reopening with default options
  keeps the stored flag and a commit never changes it; `ProtectedConsumers` creates protected consumers when the log
  is opened; `.tmp-*` files younger than an hour survive the writer's open and older ones are removed; an unparseable
  or unreadable consumer file protects everything and `Check` is UNHEALTHY; `DeleteConsumer` leaves the lock file; a
  second process opening a held name gets `ErrLocked`; upper-case and reserved names are refused by the public API
  while `Archive` still works.
- **T10 Late consumer** (phase 3). A consumer started after the writer has finished reads the whole log; one started
  after `Retention` expired gets a `GapError` whose `To` is the first retained sequence minus one.
- **T11 Creation and marker** (phase 2). A newer `quill.format` is refused; segments with no marker are refused; an
  empty directory is a new log, and one with other files is refused; two processes creating the same log at once end
  with one log ID; a crash at every step of creation (simulator) either leaves a directory that `Open` completes, or
  one that is refused, and never one that opens as a different log; a reader that arrives mid-creation gets `ErrNoLog`.
- **T13 Disk bounds** (phase 3). A log with `MaxBytes` and a fast writer never holds more than `MaxBytes +
  SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)` of segment bytes (decision 69) at any sample, with no consumer attached; with a
  protected, stalled consumer the cap still wins and `Dropped` rises; `MaxBytes` below twice `SegmentBytes` is
  refused. `MinFreeBytes` with an injected free-space function deletes only this log's oldest sealed segments, never
  the active one, and appends fail with `ENOSPC` when none are left.
- **T14 Full disk, for real** (phase 3, gated). In a Linux container with a small `tmpfs` mount, fill the log's
  filesystem: appends fail with the flush error, existing records still read back, nothing is corrupted, and once
  space is freed the writer recovers with no restart. It runs in CI and under an environment variable on a developer
  machine, and is not part of the default gate.
- **T15 Roll, rotation and archive** (phase 3). `Roll()` and rotation do nothing on an empty active segment; a crash
  at each step of rotation (simulator) leaves a log that recovers, including a marker that names a sealed segment;
  `Archive` copies sealed segments with a CRC check, a second run copies nothing, a corrupt source segment stops it, the destination opens with `OpenReader` and reads
  the same records, a source with lost segments yields a destination that reports the hole, and the archiver being a
  protected consumer, a segment it has not copied is kept until a cap overrides it.
- **T16 Consumer aging** (phase 3). With `ConsumerTTL`: a protected consumer that is behind, with an old mtime and an
  old record at its offset, stops pinning segments and shows in `Stat().StaleConsumers`; a `Commit` makes it live
  again; a caught-up idle consumer is **not** stale when new records arrive after a quiet period longer than the TTL;
  restarting the writer does not reset the clock; the archiver can go stale like any protected consumer; its file
  survives and a returning consumer resumes. Use an injected clock and set mtime explicitly; also assert the premise
  that a `Commit` and a creation each leave a file whose mtime is the time of the call.
- **T17 Size accounting** (phase 3). `Stat().Bytes` equals the sum of the segment files' sizes; `Stat().Records` is
  `Head − first retained + 1` minus the records inside holes and equals the number of records a scan that resumes
  past each gap returns, across rotations, retention deletes and a crash and reopen.
- **T18 Sync** (phase 2). After `Sync(seq)` returns, a `SIGKILL` never loses `seq` or anything before it. Many
  producers calling `Sync` concurrently cause fewer fsyncs than calls (count them with an injected hook). A lone
  `Sync` returns well before `FlushInterval` (it signals the flusher). `ErrBadSeq` for an unassigned sequence; pending
  waiters get `ErrClosed` at `Close`. With a flush failing, `Sync` and a `SyncAppends` append return the error, which
  means unknown. A producer waiting in `Sync` holds no lock: with a slow fsync injected, appenders are blocked only for
  the duration of that flush's write and fsync, not for the waiter's whole wait.
- **T19 Backlog independence** (phase 3). A `Read` of 100 records at the start of a log of a thousand segments costs
  about what it does at the tail (within a small factor), and a reader draining a long backlog never slows as it falls
  further behind. A `Reader`'s first touch of each segment costs one scan and is excluded from the comparison; later
  reads in a segment are indexed.
- **T21 Clock guard** (phase 3; the `check` part in phase 4). A forward step of the clock deletes at most a quarter of the
  sealed segments per pass, rounded up, and logs a warning; `prune -force` lifts the limit; `check` flags future-dated and
  non-monotonic records.
- **T22 Config and limits** (phases 2–3). Every `Config` default and refusal; a record of exactly `MaxPayloadBytes`
  (64 MiB − 32) is accepted, written, recovered and read, and one byte more is `ErrPayloadTooLarge`; a batch over
  `MaxBatchBytes` is refused; with flushes failing an append is accepted only into an empty buffer or within
  `MaxBufferBytes`; `MaxReadBytes` returns at least one record even when it exceeds the bound.
- **T23 Locks** (phase 2 for the library; phase 4 for the CLI). A second `Log` on a directory returns `ErrLocked`; the
  lock is released when the holder is `SIGKILL`ed, so a reopen succeeds. The CLI lock matrix in decision 53 holds for
  every verb.
- **T25 Iterators** (phase 3). `Records` yields exactly what `Read` returns over the same range, in order; `Follow`
  yields new records after a flush (a `Log`) and within the poll interval (a `Reader`) and returns promptly when its
  context is cancelled; an error is yielded once and the sequence ends, and a `GapError` is resumed from `To + 1`;
  breaking out early leaks no descriptor and no goroutine (count open descriptors); mutating a yielded payload never
  changes the log or a later record; the consumer forms start at the committed offset and never commit.
- **T26 Seek by time** (phase 3). Across many segments with monotonic times, `OffsetForTime` equals a linear scan for
  every second in the range, including a run of equal timestamps that **starts at the end of one segment and
  continues into the next** (the first of the run is returned); a time before the first retained record returns the
  first retained sequence and a time after the last returns `Head + 1`; it works after retention has deleted a prefix,
  on a `Reader` (bounded by `quill.tail`) and on a `Log`; a damaged chosen segment returns a `GapError`; non-monotonic
  input never panics and returns an offset in range; `SeekToTime` then `Records` starts at the right record; the
  search compares O(log segments) first-frame times and scans one segment, over a view that costs O(sealed segments)
  per call and, on a `Reader`'s first call, one open per sealed segment.
- **T20 Readable output** (phase 4). `cat -json` over records holding every byte value 0–255, invalid UTF-8, NUL,
  newlines, quotes, a 1 MiB payload and an empty payload: every line is valid JSON on its own, has exactly one of
  `payload` or `payload_b64`, and decoding all lines returns the original bytes and sequences. `-b64` gives
  `payload_b64` throughout. A payload containing terminal escape sequences prints escaped in both text and JSON
  modes. The verbs run beside a live writer without taking its lock and show nothing beyond `quill.tail`.
- **T24 Benchmark baselines** (phase 4): buffered append, append with a flush per call, read from the start, read near
  the tail, read of sealed segments, `ReadWait`, cold start, several logs flushing in parallel, a reader following a
  live writer, and draining a backlog larger than memory from a real disk (the evidence for decision 15). Six runs
  on one machine, recorded in `docs/audits/benchmark-baselines.md`, with the cost of the second fsync per flush
  stated. A later release regresses a benchmark when the median of six runs is more than 10% worse and the ranges do
  not overlap (decision 70).

## Phased execution

Stop after each phase and hand back (below). Do not start the next phase until the owner says so.

1. **Format, segments and the seam.** `internal/fsx` (real and crash-simulating) with T27, `codec.go`,
   `internal/atomicfile`, `internal/strictjson`, `state.go`, parse-forward recovery of a single segment, golden
   fixtures, T12, and **`docs/format.md`**, written now because the golden fixtures freeze it.
2. **The log.** `Open` (the creation protocol) and `Close`, the lock, load and recovery with hole detection, append
   (with `MaxBatchBytes` and `MaxPayloadBytes`), flush with the failure classes, rotation and `Roll`, `Sync`,
   `quill.tail`, read and scan, the format marker; T1 (phase-2 part), T2, T3, T5, T11, T18, T22 (config and limits),
   T23 (library).
3. **Readers, consumers and retention.** `OpenReader` with its index, gaps, `Stat` and `Check`, consumer handles and
   offset files, the log ID, protection, rotation by age, `MaxBytes`, `MinFreeBytes`, `MaxRecords`, `MaxReadBytes`,
   `ConsumerTTL`, the clock guard, `Dropped`, `Archive`, the logger, iterators, seek by time; T1 (phase-3 part), T4,
   T6–T10, T13, T14, T15, T16, T17, T19, T21 (guard), T22 (read bound), T25, T26. Build the reader and consumer core
   first, then retention and caps, and iterators, seek by time and `Archive` last; if the phase runs long, those three
   move to v0.2.0 without touching the core (decision 70).
4. **CLI, docs and release readiness.** `cmd/quill` (including `archive`, `recover` and `-json`), the
   examples, **`docs/introduction.md`** and **`docs/executive-summary.md`**, `docs/userguide.md` (with the manual corruption procedure),
   `docs/architecture.md`, the generated `docs/reference.md`, `docs/audits/` seeded (test ledger,
   untestable-without-x, benchmark baselines), `CONTRIBUTING.md`, `CHANGELOG.md`, the Go-version policy in the
   README, CI (Linux and macOS, `-race`, the fuzz budget, `govulncheck`, `staticcheck`, `deadcode`, the `go mod tidy`
   gate, the opt-in test tiers type-checked, the T14 job), a build script with version stamping and an SBOM that
   refuses a dirty or untagged tree and produces release archives, and a self-review of the recovery path and the
   reader before the audit; and, before the repository goes public, replacing every figure in `review.md` and
   `scope.md` that came from the private reference with quill's own measurements, or softening it (decision 70); T9
   and T21 (CLI parts), T20, T23 (CLI), T24.

## Release procedure (decision 55)

**Stage 1 — this build, ending at v0.1.0.**

1. Phases 1–4 are done and each was handed back to the owner.
2. **Every gate is green** on a clean tree: `hack/gate.sh`, the fuzz budget, the crash-simulation suite, vet and
   staticcheck under the opt-in build tags, the T14 job, and the benchmark baselines recorded.
3. Run the **A1 adversarial audit** — the house prompt, which includes the four-dimension A2 pass, a skeptic per
   candidate told to refute (default verdict "refuted"), a reproduction of every correctness or security finding on
   a copy, and apply-and-reverify against the full gate — **as a workflow**.
4. **Fix everything** that survives the skeptics (CONFIRMED and PLAUSIBLE; REFUTED findings are recorded, not
   fixed). Re-run every gate. Update `docs/audits/` and `decisions.md`.
5. **Tag `v0.1.0`** on the commit where every gate is green after the fixes, and stop. Do not push unless asked.

**Stage 2 — a later session, possibly a different model, ending at v0.2.0.** It is not part of this build; it is
stated here so the build leaves the repository ready for it (audit records, test ledger and decisions current). The
later session runs the complete house audit sequence **as a workflow**, in the recommended order: A1 with A2 (the
baseline), A4 supply chain, A3 threat model, A5 silent degradation, A6 format compatibility (golden bytes), L1
verification and optimisation, L2 test, benchmark and hardening, L3 live integration with L3b topology (quill is
multi-process: a writer and readers), A7 gap audit, A1 diff-scoped with A4 refreshed, and L4 release packaging. It
fixes everything, a skeptic then validates every fix (told to refute it; default "refuted"), every gate is re-run,
and `v0.2.0` is tagged when the sequence is complete.

## What to hand back at each stop

- What was built and what was cut, and anything in this design you found wrong. Say so plainly rather than
  working around it.
- `gate.sh` output, the test count, and which guarantee in scope.md has which test (T-number) so far.
- Every new entry in `decisions.md`, and every open question you could not settle.
- At phase 4: the baseline table and the module's line count against the ceiling in review.md.
- After the A1 audit: the findings by verdict, what was fixed, and the tag.

## Out of scope — do not build

Sharding, routers or cross-shard frontier logic; the fixed-width format; any socket, TLS, enrolment, collector or
shipper, or any file whose only purpose is to feed one; a liveness probe of the writer; several topics in one
directory; a call that removes a directory under the lock; consumer groups or rebalancing; replication; a binary
wire encoding; compression; per-record keys; encryption at rest; mmap or a segment cache (decision 15); the `Set`
helper (roadmap); and anything that needs a dependency. Each is a recorded cut (review.md) or a roadmap item
(scope.md). If one seems necessary to meet a guarantee, stop and ask.
