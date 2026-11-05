# Quill — review of the reference design

*Review of an existing, working private implementation (thousands of lines of Go: a library, a network
fan-in layer and three commands) that this project starts from, read from its source and docs on
2026-10-28. Quill keeps its core — CRC-framed segments, buffered fsync'd appends, ordered reads, consumer
offsets, retention — and changes its shape: **one log per directory** (a topic is a directory, with its own
lock and writer), **one lane per log**, and **readers in other processes** as a first-class part. Findings
below are inputs to [scope.md](scope.md) and [decisions.md](decisions.md); each says what to keep, change,
cut or add, and why. Claims are marked **verified** (read in the source or run), **reported** (stated by the
reference's own docs) or **inferred** (reasoned from the above, not run).*

## What the reference is

An embeddable, single-node, append-only topic log in one Go package, plus a network layer on top:

| Part | What it does |
|---|---|
| Core library | Many topics in one directory, sharded CRC-framed segments, buffered fsync'd appends, prefix-consistent reads, consumer offsets, retention and byte caps, read-only mode, an optional fixed-width format, a binary wire encoding |
| Network fan-in layer | Mutual-TLS transport: batch encoding, sensor key set, collector, shipper, watermarks |
| Commands | An inspect tool, a collector daemon, a shipper daemon |
| Helpers | Crash-safe writes, strict state-file parsing, a vendored mmap (BSD-3, carries a notice) |

The model, **verified** in its source: one queue per directory, enforced by an exclusive `flock`. State is
kept per **(topic, shard)** with its own `RWMutex`, buffer, active segment and sealed segments; each topic
has its own sequence counter from 1. So it has **one writer for the whole directory, not one per topic**:
many goroutines may append, calls on different topics never share a lock, and one background goroutine
flushes every shard each flush interval and runs retention. Its `Flush` walks the shards **serially**,
holding each shard's lock through its write and fsync.

Consumers are **named offsets**: a commit updates an in-memory map that is persisted to a state file on the
next flush. The writer owns that file. A second process can read a live directory only through a read-only
mode with an explicit refresh, which trusts a per-topic frontier file published by the writer.

## What is good — keep

- **Self-delimiting CRC frames** (`uvarint(len) | body | crc32`), so a torn tail is detected and
  truncated and nothing needs a delimiter scan. Pinned by golden-byte tests. **Quill keeps these bytes
  unchanged** (decision 3).
- **Ordered, durable-first reads.** A record is visible only once it is durable and nothing earlier is
  pending, so committing the highest sequence read is always safe. At one lane this comes almost free.
- **Whole-segment retention.** An O(1) unlink per expired segment, never a rewrite, never the active
  segment. Byte caps bound the disk.
- **Segment ids that are never reused**, so a deleted name cannot alias a live one.
- **One writer, enforced** by the exclusive directory lock.
- **Corruption is loud and contained.** A segment corrupt in the middle fails to load, is never
  modified, is reported by an error accessor and a check command, and the other segments still serve.
- **Bounded allocation on hostile input:** a declared frame length over 64 MiB is corruption; a segment
  over 1.5 GiB is refused; payloads are capped just under 64 MiB.
- **No module dependencies** and linux/darwin only.
- **Crash-safe state files:** temp file, fsync, rename; parsing that refuses damaged values.
- **Benchmarks and tests that already exist.** Their behaviours are the spec quill is built against
  ([build-prompt.md](build-prompt.md)).

## Findings — what to cut, change or add

Verdict key: **Cut** (not in quill), **Change** (kept, done differently), **Add** (new), **Keep**
(carried over). Impact is the size of the simplification or risk removed.

| # | Verdict | Impact | Finding | Why | Resolution |
|---|---|---|---|---|---|
| 1 | Cut | High | **Sharding.** Each topic is split over several lanes with a round-robin router, per-shard locks, a per-topic CAS sequence and a *stable frontier* computed across shards. | At one lane, file order is sequence order, so the merge, the cross-shard frontier and unordered `ScanAll` disappear. **Measured** on the reference (library level and through its network ingest path): reads were faster at one shard than at four, from the start of a log and near the tail; `ScanAll` was globally ordered only at one shard; one-producer durable writes were the same; four producers gained little from sharding; through the ingest path aggregate writes were the same within noise, because the collector serialises ingest behind one mutex. The reference's figures are not repeated here (decision 70; *Stage 2 audit, owner-approved 2026-11-03, A7-15: the ratios, rates and timings that this row and decisions 3, 5, 15 and 32 used to quote were removed*); quill's own are in [audits/reference/benchmark-baselines.md](../audits/reference/benchmark-baselines.md). | One lane per log. Shards, the router, the cross-shard frontier and the durable-sequence call go. |
| 2 | Change | High | **Many topics in one directory.** Topic names become file names, which forces a registry, a name-to-file mapping, a topic-name length limit, a case-collision check, per-topic maps in every state file and a global byte cap across topics. One lock means one writer process for all topics. | The owner's model is "one assigned writer per topic". A directory is the natural unit of ownership: it has its own lock, state, retention and backup, and a topic moves or is deleted as a directory. | **A log is a directory.** `Open(dir)` returns a `Log` for one topic. Multiple topics are multiple logs, in one process or several. Registry, case check, name limit and global caps go. Frame bytes do not change. |
| 3 | Cut | High | **The fixed-width format**: a second on-disk format with block CRCs, structural recovery of a CRC-less tail, header repair, and several of the reference's documented limitations (**reported**). | Its one benefit is O(1) cold start. Everything else is complexity on the recovery path. | Variable-length frames only. Cold start stays proportional to bytes; `.idx` sidecars for sealed segments are the answer if that bites (roadmap; the reference lists them as not planned). |
| 4 | Cut | High | **Network fan-in** (the transport layer, collector and shipper daemons, with TLS pinning, enrolment and watermarks). | A separate product that sits on the library's API. Quill is the log, not the transport. | Out of quill. It has no sockets, and it publishes nothing for a shipper (owner decision, 2026-10-28). |
| 5 | Cut | Low | **A binary wire encoding of a record.** | Used only by the network layer and one example. | Cut. |
| 6 | Change | High | **Cross-process reading is the reference's most complicated feature** — a read-only mode, an explicit refresh, a per-topic frontier file, a shared-flock probe to decide whether a writer is live, an option to skip unloadable segments, an error type per segment — a large part of the core (**verified** by function list), one known limitation and part of another (**reported**), and a documented case where a tight loop of probers refuses a starting writer. Yet it is a **requirement** here: one process writes, another consumes lazily. | A reader in another process needs (a) no writer lock, (b) a boundary that is durable, (c) somewhere to keep its own offset, (d) retention that knows about it. At one lane (a) and (b) shrink to one small file. | Keep the capability, rebuild it small: a lock-free `Reader` bounded by a **`quill.tail`** marker the writer publishes after each fsync (decision 8). No liveness probe; the reader trusts the marker only. |
| 7 | Change | High | **Consumer offsets live in the writer's state file.** Only the writer process can commit. | An external consumer cannot persist an offset in a file the writer owns, and the writer's retention cannot know about it. | **One file per consumer** under `consumers/`, written only by that consumer's process under its own lock, for in-process and external consumers alike (decision 9). The writer's retention reads them. |
| 8 | Change | Medium | **Several state files kept in step** — head, segment ids, frontier, offsets — each persisted separately on every flush (several fsyncs per durable flush). | One log needs one durable position and one highest segment id. | Merge into one **`quill.tail`** (durable seq, durable position, highest segment id), written after each fsync (decision 8). |
| 9 | Change | Medium | **Sequence gaps after a crash.** The reference persists the *assigned* head (**verified**: it reads the topic's atomic counter), so a crash leaves gaps; its docs say so, and a durable-sequence call exists to tell a gap from "not yet". | With one lane, readers only ever see durable records, so no consumer holds an unflushed sequence. | Sequences are **contiguous**: the next sequence is the durable tail + 1, and the numbers of an unflushed suffix are reassigned after a crash. Callers must `Flush` before acknowledging a sequence upstream (the reference's docs already say so). Pinned by a crash test (decision 10). |
| 10 | Change | Medium | **The variable-length segment format has no version** (a documented limitation, **reported**): an older binary treats an unknown layout as a torn tail and, if writable, truncates it. | A format that cannot announce itself cannot be changed safely. | Keep the segment bytes and add a **per-log format marker file** (decision 6). |
| 11 | Change | Medium | **The core is thousands of lines in one file.** | It cannot be audited in one sitting. | Split by responsibility, none over about 600 lines ([build-prompt.md](build-prompt.md)). |
| 12 | Change | Medium | **`Flush` is serial across topics and holds each topic's lock through its fsync** (**verified**). A flush with T dirty topics costs T fsyncs back to back. | With a log per directory each log flushes on its own goroutine, so topics flush in parallel by construction. A flush still blocks that log's producers for its duration. Not measured here. | Per-log flush loop. Moving the fsync outside the append lock is a measured follow-up. |
| 13 | Change | Medium | **Reads map segments into memory**: a cached mapping for sealed segments, and an open, `mmap` and `munmap` on every read of the active one, using vendored Go-authors code (BSD-3, so a notice travels with every binary) that was modified to expose the mapped bytes and to open without following symlinks. Active-segment reads are syscall-bound (most of read CPU, **reported**), a mapping must be remapped to see appended bytes, and a truncated mapped file can fault a reader (the reference fixed one such crash and lists the rest as a limitation, **reported**). | The mapping is a speed optimisation, not the fix for a far-behind consumer: that is the sparse index, skipping segments by their highest sequence without opening them, and stopping after `limit` records (finding 14). **Measured** on the reference's access pattern: a held-open `pread` was slower per 100-record read than a cached mapping, and drained a backlog more slowly, warm (decision 15 keeps the conclusion; the figures were removed). | **Read with `pread`**: no mmap, no cache, no vendored code (decision 15). Measure a backlog larger than memory on a real disk in phase 4. |
| 14 | Change | Medium | **Cold start reads and CRC-checks every byte** to rebuild the sparse index (**reported**: roughly resident bytes ÷ disk bandwidth). **A far-behind consumer** stays cheap for a different reason: the reference's reads skip segments whose highest sequence is below the offset without opening them, seek by the sparse index, and stop after `limit` records per segment; before that early stop, a low-offset read decoded everything from the offset to the end of the segment, quadratic in backlog size (**reported**, its comments and operational plan). | Cold start is fine for typical directories and minutes for 100+ GB of never-expiring log. Read latency must not grow with backlog. | Accept the cold start for v0.1.0, state it, measure it in phase 4. Keep the three read-side mechanisms and add a test that a read at the start of a log with a thousand segments costs about what a read at the tail does (test T19). |
| 15 | Change | Medium | **Protection from retention is one log-wide switch, off by default**, so unread data is deleted silently once retention passes (the only signal is an in-memory `Dropped` counter), and turning the switch on protects every consumer, including a stalled one. | With a lazy consumer, "starts after the retention window, finds nothing" is the failure to expect; with the switch on, a dead consumer pins the disk. | Protection is **per consumer**, off by default, and the archiver is protected (decision 57). Document that `Retention` must exceed the worst delay of an unprotected consumer, or use `-1`. |
| 16 | Add | Medium | **A crash property test** with a real `SIGKILL`, covering contiguous sequences and no durable record lost or duplicated, and a writer-plus-reader test. | Recovery is where logs fail. | Required tests in phases 2 and 3. |
| 17 | Cut | Low | **Options and calls that exist only for sharding, the fixed-width format, the topic registry or the network layer**: truncate-on-open, a pre-check of topic names, format detection, per-topic error narrowing, frontier accessors, the durable-sequence call, an explicit refresh method. | Each is dead weight without the feature it serves. | Cut. A reader refreshes itself on each call. |
| 18 | Keep | Low | **`At` is whole Unix seconds; a consumer's offset may be set to any value**, including a rewind (the reference's maintainers say not to add a monotonic guard). | Replay. | Carry over. |
| 19 | Change | High | **Disk bounds are weak by default.** The reference's byte caps run on the maintenance timer, not on the append path, so a burst can pass a cap by an interval's arrivals; they default to off, so its server is "bounded by time only" and a fast writer fills the disk inside the window; the active segment of every shard is a floor no cap removes; there is no free-space check (listed as not planned); and a real full disk was never run (all **reported**, its user guide, "Sizing and disk-full"). | A log that is meant to roll with no consumer attached must bound itself. | Per-log `MaxBytes` checked at every seal, `MinFreeBytes` via `statfs`, `MaxPayloadBytes`, a full-disk test in a container, and `check` warning on an unbounded log (decisions 24–26, 30). |
| 20 | Add | Medium | **Consumers never age out.** A registered consumer that never commits pins unread segments until it is deleted by hand (**reported**; a search of the source finds no expiry). With the retain option on, one dead consumer can hold the disk until a cap fires. | A log with several, possibly external, consumers will have dead ones. | `ConsumerTTL`: a consumer that is behind and silent for the TTL stops protecting segments; its offset is kept (decision 31). |

## Things deliberately **not** carried over

- Everything under findings 1, 3, 4, 5 and 17.
- The reference's example servers (HTTP, Unix socket, an AI-assistant server) and its live container test
  tier. They exercise the transport layers quill does not have. Three examples replace them.
- The sprawling inline limitation prose in the reference's architecture doc. Quill keeps its limitations
  list short and each item testable.

## Verdict

The reference's core is sound; its size comes from the options layered on it. Quill changes its shape, not
its bytes. Cutting the network layer and its two daemons, the
fixed-width format and the wire encoding removes a large share of it before sharding, the frontier and the
topic registry go, and before the reader and consumer files are added back (**inferred** from the table above,
not counted). Quill's own size, for orientation: **about 5,400 lines** of non-test Go in the root package
(`ls *.go | grep -v _test.go | xargs wc -l | tail -1` in the repository root; it drifts by tens of lines per commit, so re-run it
rather than trust this figure). Build it **fresh**, porting the
reference's behavioural tests that still apply as the specification, and not by deleting code from the reference. Findings 1, 2, 6 and 7 define the shape; 8–11
and 19–20 are improvements worth making on the way.

**Risks to this review.**
- **Sharding data:** measured at library and network-ingest level on a laptop and a VM, where the
  container's fsync is much cheaper than the host's and may not reach the physical disk. Phase 4
  recorded quill's own baselines on one machine instead of re-running the reference's benchmarks;
  the comparison at one and four shards stands on the reference's measurements alone, which are not repeated here.
- **Reader hazards:** the cross-process reader's durability hazard (a reader seeing page-cache bytes that an
  OS crash would drop) is **inferred** from how `write`/`fsync` work, not tested. An OS crash cannot be
  simulated from a test; it belongs in `docs/audits/reference/untestable-without-x.md`.
- **Counts:** the one line count here is quill's own, by `wc -l`, and includes comments.
