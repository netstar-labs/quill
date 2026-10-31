# quill architecture

*Subsystems, data flow and the trade-offs behind them. The decisions themselves, with their
reasons, are numbered in [design/decisions.md](design/decisions.md); this document is the
map.*

## 1. Shape

```
            ┌──────────────────────────── one process ────────────────────────────┐
            │  Append ─► buffer ─► Flush: pwrite at the durable offset, fsync,    │
            │                      rotate if full, publish quill.tail             │
            │  Read/Scan ─► view (sealed ranges + active to the durable offset)   │
            │  flusher goroutine: FlushInterval, Sync requests, PruneInterval     │
            └──────────────┬──────────────────────────────────────────────────────┘
                           │ one directory, one flock
   ┌───────────────────────▼──────────────────────────┐
   │ 0000000000.log … 000000000N.log   quill.tail     │  segments: CRC frames, ids never reused
   │ quill.format  quill.policy  quill.lock           │  marker: durable (seq, seg, off, maxseg)
   │ consumers/<name>  consumers/<name>.lock          │  one file and one lock per consumer
   └───────────────────────▲──────────────────────────┘
                           │ no lock; reads quill.tail on every call
            ┌──────────────┴──────────────────────────────────────────────────────┐
            │  OpenReader: list segments below the marker, ranges from first      │
            │  frames, index per sealed segment built on first use, the active    │
            │  segment read only to the marker's offset; Consumer; Stat; Check    │
            └─────────────────────────────────────────────────────────────────────┘
```

A log is a directory; a topic is a log; the writer is the process holding `quill.lock`. The
library is one Go package split by responsibility, none over about 600 lines; every file
operation goes through `internal/fsx`, a seam with a real implementation and a crash
simulator.

## 2. The frame and the segment

`frame = uvarint(bodyLen) | body | crc32(body)`, `body = uvarint(seq) | uvarint(at) |
uvarint(len) | payload`. Frames are self-delimiting, so a torn tail is found by length and
CRC and nothing scans for delimiters. A segment is frames back to back; the newest is active,
the rest sealed. Sequences are contiguous within and across segments, so a sealed segment's
range is `[its first, the next one's first − 1]` and a reader needs only first frames to
locate a sequence. A sparse index (one anchor per 64 KiB: sequence, byte position, time)
lets a read seek within a segment and a time search bound its scan.

The frame bytes are the reference design's, unchanged (decision 3); the layout around them
(`quill.tail`, `quill.format`, per-consumer files) is new. [format.md](format.md) is
normative and the golden tests freeze it.

## 3. The write path

`Append` takes the log's mutex, assigns sequences, encodes into the buffer, and returns; at
`FlushBytes` it flushes inline. `Flush` holds the mutex through the write and the fsync
(decision 13): it `pwrite`s the buffer at the durable offset, fsyncs, and only then releases
the buffer and advances the head, so in-process readers never see a byte that is not durable.
Then it publishes the marker by temp file, fsync, rename, with no directory fsync; rotation
(create the next segment, fsync the directory, publish the marker naming it at offset 0) and
`Close` and deletion do fsync the directory (decision 65).

Each step has a failure class (decision 60): a write error truncates back and keeps the
records buffered; an fsync error fails the log for good; a marker or segment-creation error
after the data fsync is retried at the next flush and never rewrites durable data (while the
next segment cannot be created, appends are refused at a ceiling under the file limit); a
directory fsync error is fatal. `Sync` is group commit: producers append under the lock,
release it, signal the flusher and wait on a broadcast channel that every flush closes, so
concurrent producers share one fsync and a waiter holds no lock.

## 4. Recovery

Let the marker name segment S at offset X. Every byte of a sealed segment, and every byte of
S below X, was fsynced before the marker named it, so damage there is corruption: the file
is never modified, a sealed segment becomes a hole, and a corrupt newest segment fails `Open`
with `ErrCorrupt`. From X the newest segment is parsed forward, accepting each frame that is
intact and exactly one past its predecessor and truncating at the first that is not; nothing
after it is trusted, so a frame embedded in a payload is never reached (decision 39). The
marker is a lower bound, not the truth: after a power loss it can be older by the flushes
since the last directory fsync, and parse-forward recovers everything it missed. The one
residual: damage between a stale marker and the true end of the newest segment looks like a
torn tail.

Sealed segments are verified whole and in parallel at open. Holes are found by comparing
each segment's first sequence with the previous one's last; an overlap is corruption. The
next sequence is the last valid frame's plus one, or `quill.tail.seq + 1` when the log holds no
frames; a marker above the last valid frame means acknowledged data is missing, and `Open`
refuses.

## 5. The read path

One `scanSource` serves `Log`, `Reader` and `Consumer` through a `source` interface: take a
view (segment ranges, holes, head, first retained), locate the segment holding the offset,
seek by the sparse index, read through a chunked window into a pooled buffer with `pread`,
decode in place, copy payloads for `Read` and iterators, alias them for `Scan`. A gap is
reported where records are missing, never skipped. The log keeps the active segment's
descriptor open and pins it for reads in progress across a rotation; other segments are
opened, read and closed (decision 15: `pread`, no mmap, no cache, no vendored code).

A `Reader` builds its view on every call: read `quill.tail`, list the directory, treat the
marker's segment as active and ignore anything above it. A sealed segment's first frame is
read once and cached; its index is built by one full scan on first use and cached; the active
segment is never cached and is read only to the marker's offset. A cached entry costs no file
operation until a read finds the file gone or changed. `Follow` keeps a byte cursor so a long
iteration does not re-seek.

## 6. Consumers and retention

A consumer file holds the log id, the next sequence to read and a protect flag, and is
written only under the consumer's own `flock`, by temp file, fsync, rename. The writer's prune
pass reads every consumer file: an unparseable one protects everything; a protected one
protects its unread sealed segments unless `ConsumerTTL` finds it stale (behind, with an old
file and an old record at its offset); the caps override protection and `Dropped` counts the
loss. The caps that run at every seal do not read the consumer files, which would stall every append under the log's lock: they count `Dropped` against the offsets the last prune pass read. The age sweep takes the longest expired prefix of sealed segments, cuts it at protection,
then at a quarter of the sealed segments per pass, rounded up (the clock guard), then the caps extend it;
the marker is published and the directory fsynced before any unlink, and again before each
further one so that a crash leaves the surviving segments a suffix. The same caps run at
every seal.

`quill.policy` records the last writer's settings (retention, caps, segment size, payload limit)
for tools that see only the directory. An open with `NoMaintenance`, which is how the offline
verbs `append`, `roll`, `prune` and `recover` open, never writes it. All four take the segment
size and payload limit from it, so opening does not seal a segment the writer's own size allows
(decision 99). `append`, `roll` and `recover` open with no caps; `prune` opens with the policy's
caps, overridden by any retention flag, and needs the policy or a flag (decision 68).

## 7. Testing

The crash simulator keeps a durable and a volatile image per file and per directory: unsynced
pages persist or tear independently, directory operations persist or not until the directory
is fsynced, and every crash state can be enumerated. The crash property test runs a scenario
with a crash injected before every file operation and asserts that recovery yields a
contiguous prefix holding every acknowledged record, under every subset of surviving items
when few and a sample otherwise; the same property is tested under a real `SIGKILL` of a
writer process. What neither can show is in
[audits/untestable-without-x.md](audits/untestable-without-x.md).

## 8. Trade-offs, stated

- **One lane per log.** No sharding, so file order is sequence order and a single producer's
  durable throughput is bounded by one fsync at a time. Many topics scale by many logs.
- **A marker per flush.** Every flush writes and fsyncs `quill.tail`'s temp file as well as the
  data: the price of a reader that needs no coordination. The baselines state it.
- **Parse-forward over a directory fsync per flush.** A stale marker after a power loss costs
  nothing but a moment of hidden records; the residual above is the trade.
- **`pread` over mmap.** A few microseconds per read, for no cache, no vendored code and no
  faulting reader when a file is truncated under it.
- **Cold start reads every byte.** Verification of every sealed segment at open is
  proportional to the log's size; `.idx` sidecars are the answer if it bites (roadmap).
- **Segment-granular retention.** Data outlasts its window by up to a segment, and a quiet
  log by up to about twice the window until age rotation seals it.
