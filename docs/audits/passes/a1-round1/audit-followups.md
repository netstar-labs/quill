# A1 adversarial audit: follow-ups

Date 2026-10-29, against HEAD b52b746 (the source is identical to 1f4152d). What the A1 audit
deliberately does not change in the apply-now pass, and why. This is the seed for the tracker:
each entry under "Owner decisions" and "Deferred" becomes a `type:audit` issue carrying the
`sev:` label shown; the trivial fixes batch under one `type:chore` umbrella at the end. The
anchors, each skeptic's reasoning and the reproductions are in audit-correctness.md,
audit-dedup.md, audit-docs.md and audit-simplify.md; the classification rule is in
audit-findings.md. Every owner decision below comes with a recommended answer; where that
answer is "approve", the fix and its test are already written out in the dimension report.

## Owner decisions (one issue each; both approved and applied in the first fixer chain)

### correctness-21: `Prune` panics or over-deletes after a failed rotation when a cap is set (sev:major)

retention.go:219 runs `publishLocked` inside `deleteLocked`; a pending rotation (decision 60 d)
completes there and its nested `deleteLocked(l.capsLocked(0), ...)` (flush.go:141) shortens
`l.sealed` under the outer loop's precomputed `n` (:226-227, :243). Reproduced three times:
`index out of range` at retention.go:227, silent deletion of unexpired segments, and a
`ProtectOn` consumer losing its unread segment with `Dropped=1`. Why owner: the finder flagged
it semantic (observable behaviour changes) and the fix has two variants, one of which
(`publishLocked(runCaps bool)`) bends decision 24's "checked at every segment seal". Recommended:
approve the snapshot variant (record the targets before `publishLocked`, delete only the
leading run of `l.sealed` still in the set); four regression tests in a new
retention_reentrancy_test.go.

*Applied as recommended (the snapshot variant), commit ad71017.*

### simplify-1: a Reader panics on a sealed segment whose first frame carries seq 0 (sev:major)

reader.go:204-206 calls `reason(ferr)` with `ferr == nil` when the frame is CRC-valid and seq 0;
`reason` (load.go:263) dereferences the nil error. Reproduced twice; with the one-line guard the
Reader returns `GapError{1,1}` as decisions 11/38 and guarantee 14 require. Why owner: flagged
semantic (panic becomes a gap). Recommended: approve; test from the probe.

*Applied as recommended, commit cc9cad4.*

## Applied

### docs-39: the disk bound omitted the single-record case (applied)

Decision taken: approved as recommended, docs only plus one comment. The bound is now
`MaxBytes + SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)` in scope.md
(the Bound paragraph and guarantee 11) and userguide.md; decision 69 is amended in place with a
dated note (no new row). scope.md's `MaxBatchBytes` row and the `AppendBatch` comment say a single
record is bounded by `MaxPayloadBytes` alone. Admission and validation are unchanged, so no test
was added.

### dedup-19: one `contiguity` helper for the hole/overlap bookkeeping (applied)

Decision taken: approved as recommended (decision 102). holes.go holds `contiguity(rs, badFirst)`:
from the ordered good segments' ids, first and last sequences (the active segment last, when it
holds frames) and a flag for a bad segment before the first good one, it returns the holes and the
first overlap as `ErrCorrupt`. `Log.load` and `Reader.view` call it; the writer's refusal when
`head < tl.Seq` (decision 39) and the reader's marker hole (decision 11) stay in their callers.
One overlap message replaces the two (`segment N starts at F, inside the one before (segment M,
ends at L)`), and an empty hole is never reported. The reader now collects the segment ranges
first and builds the references after the check; the ranges themselves are computed as before.
Tests in holes_test.go: a table over the helper (bad first segment, bad first at sequence 1, bad
middle, bad first and middle, empty active, gap before the active segment, a range cut short by
a scan, a stale marker below a sealed segment's end, three overlaps), and out-of-order sealed
segments make both `Open` and a `Reader` return `ErrCorrupt`.

### correctness-28: `quill append` ignored `Close`'s error (applied)

Decision taken: approved as recommended (decision 100). `append` calls `Close` before printing
and exits 1 when it fails. Under `-sync=true`, where `Sync` had made the records durable, it
still prints the last sequence to stdout and the error to stderr; without `-sync` it prints
nothing on failure. Regression tests in cmd/quill/append_close_test.go drive the real verb with
a stdin that removes write permission on the log directory once read, so `Close` cannot publish
`quill.tail`.

### correctness-22: `Close` nils the active descriptor under a pinned reader (applied)

Decision taken: approved as recommended (decision 95). `Close` retires the active segment and
closes its descriptor at once only when no reader holds it; otherwise the last reader's release
closes it, as rotation already does. A read in flight at `Close` finishes (or returns
`ErrClosed` if it began after); it never dereferences nil. The lock file is still released in
`Close` (decision 53). Regression tests in close_pinned_test.go: a stalled `Read` pinned on a
sealed and on the active segment while `Close` runs, with no panic and `OpenFiles() == 0`
afterwards.

### correctness-24 (absorbs simplify-3): `Sscanf %d` accepted `1e6`, `1M`, `0x10`, `12abc` (applied)

Decision taken: approved as recommended (decision 96). `commit -offset`, the three `prune`
cap flags and the `@unix` time form use base-10 `strconv.ParseUint` and refuse anything but
digits with exit 2 (`1e6`, `1M`, `0x10`, `-1`, `+1`, `1_000`, trailing junk, values over
`MaxInt64` for the caps). `commit` parses before opening the log, so a refused number
registers no consumer. K/M/G suffixes are not accepted; a separate chore if wanted. Regression
tests in cmd/quill/numbers_test.go: every refused form on `prune` and `commit` asserts exit 2
and a byte-identical log directory; plain decimals still work.

### simplify-12: `Check` said HEALTHY when consumers/ could not be listed (applied)

Decision taken: approved as recommended, `fatal` (decision 97). `Check` calls `list()` directly;
an error is a fatal `consumer` problem naming the `consumers/` directory, so the log is
UNHEALTHY and `quill check` exits 3, the same class as an unreadable consumer file (decisions
41 and 67). `mustList` is deleted. Regression test in check_consumers_test.go: the simulator
injects `fs.ErrPermission` on `OpReadDir` of `consumers/`; the log is healthy before and
UNHEALTHY with exactly one fatal `consumer` problem after.

### correctness-31: a damaged sealed segment's bytes counted toward neither `MaxBytes` nor `Stat.Bytes` (applied)

Decision taken: approved as recommended, and the residual closed (decision 98). `loadSealed`
records the file size on every sealed segment: from `openSegment` when it succeeds, before
verification; from an `Lstat` when the open fails. A file that cannot be `Lstat`ed counts 0.
`Log.Stat().Bytes` now equals the sum of the segment files' sizes and `Reader.Stat().Bytes`, and
`MaxBytes` (and `MinFreeBytes`) count a damaged segment and delete it when it is the oldest.
Regression tests in load_damaged_bytes_test.go: a sealed segment with a flipped CRC byte, reopened,
matches the files and the `Reader`, and a `MaxBytes` pass deletes exactly that segment; an
injected open failure still counts the file's size.

### correctness-32 + docs-33: what Config the offline writable verbs open with (applied)

Decision taken: approved as recommended (decision 99). `Open` skips `writePolicy` when
`Config.NoMaintenance` is set, so `append`, `roll`, `prune` and `recover` never rewrite
`quill.policy`. The command's writer helper (`openWriter`, cmd/quill/write.go) keeps `Retention:
-1`, no caps and no protected consumers, and takes `SegmentBytes` and `MaxPayloadBytes` from
`ReadPolicy` when it parses, else `MaxSegmentBytes` and the payload ceiling; `roll` rotates
regardless. One exception: `prune` with explicit caps and an unreadable policy keeps the default
segment size, since a cap below twice the ceiling would not validate. docs/format.md, README.md,
docs/userguide.md, docs/architecture.md and scope.md describe it. Regression tests in
cmd/quill/offline_policy_test.go (policy byte-identical after append, roll, recover and prune; a
9 MiB active segment is not rotated under a 1 GiB policy, is under a 64 KiB one; no policy, no
file created) and policy_nomaintenance_test.go.

### correctness-27: a Reader returned a raw `*fs.PathError` when a listed segment was deleted between listing and opening (applied)

Decision taken: approved, the mapping only, no retry loop (decision 101). In reader.go the sealed
segment's `open` closure, on `fs.ErrNotExist`, forgets the cached entry and returns a
`*GapError` over the segment's range; `scanSource` (read.go) sets `From` to the requested
sequence, so `Read`, `Follow`, `Seek` and `Archive` report a gap. The marker's own segment
missing stays an error. Regression tests in reader_vanished_test.go delete the segment from the
simulator's `Inject` hook on `OpOpen` for each of the four callers. Not changed: `Log`'s own
`view` has the same shape for its own sealed segments (read.go); the audit named reader.go only.

## Deferred, then applied in the allocation pass

### correctness-30: `Log.view` allocates ~2 objects per sealed segment per read, under the log mutex (sev:perf)

read.go:268-285; measured 2019 allocs/op at 1000 sealed segments, linear in the sealed count,
with Append and Flush waiting behind the mutex. CONFIRMED, not a defect. Deferred because the
fix is a cache with invalidation on the hot read path, larger than a local change: a
`[]*segRef` for the sealed segments rebuilt where `l.sealed` changes (`rotateLocked`,
flush.go:163; `deleteLocked`, retention.go:243), shared across views, landed with an allocation
benchmark.

*Applied in the allocation pass as alloc read-7 (decision 103), with
`TestSealedSegmentRefsAreSharedByViewsAndRebuiltAfterARotationAndADeletion` and
`BenchmarkReadStartThousandSegments` (2,041 to 31 allocs/op at 1,000 sealed segments).*

## Allocation pass (A1 follow-up, 2026-10-29)

Run as a workflow after the A1 fixes landed, before the tag: three finders (the write path, the
read path, the `Reader` and the command) grounded in `-benchmem` and escape analysis, two
skeptics per finding (one asking whether the site is hot and the saving above 5%, one whether the
change preserves behaviour), and one fixer per survivor in its own worktree, each required to
show allocs/op before and after and to record them in benchmark-baselines.md. 22 candidates, 9
refuted, 13 survived, 12 applied (decisions 103 to 105 where a design choice was settled), 1
deferred. Nothing was marked high risk.

### Deferred: read-8, carve payload copies from shared blocks (sev:perf)

`Read` and the iterators clone every payload separately (`slices.Clone` in `scanSource`), so
`Read(offset, 100)` pays 100 allocations for its payloads. The fixer built the arena: allocs/op
fell from 121 to 22 (`BenchmarkReadStart`) and from 114 to 15 (`BenchmarkReadTail`) and ns/op did
not move, but B/op rose by 700 to 800 bytes on every read benchmark, because the one block
(12,800 bytes of 128-byte payloads) rounds up to the 13,568-byte size class while a hundred
128-byte clones had no slack. The pass's acceptance rule was that allocs/op and B/op both
improve, so it was not committed. Open to the owner: relax the rule for this change (fewer,
larger allocations are what the GC prefers), add a decision recording that a retained payload
may pin up to 64 KiB of its batch, and land it with the full gate. The unapplied diff is in the
session's scratchpad, not the repository.

### Refuted

Every refutation was measured in a scratch copy; none was refuted on the behaviour lens.

| Finding | Why it was refuted (hot lens) |
|---|---|
| write-6: The write buffer grows by repeated `append` from nil, about 11 allocations once per log | Once per `Log`, not per call: the buffer keeps its capacity across flushes (`l.buf[:0]`), so steady-state appends already allocate nothing; the claimed B/op gain was benchmark setup amortised over the iteration count. |
| read-10: `walk` builds a sparse index that `Read`, `Scan` and `OffsetForTime` discard | One allocation per 64 KiB walked, about 3% of a `Read`'s allocations: under the 5% bar for the point reads. The same site was found again as reader-20, whose skeptics confirmed it on `ScanAll` (104 to 47 allocs/op), and it was applied there. |
| read-11: `records()` rebuilds its batch slice by growth every round | The slice is built once per round (once per `Records` call unless `MaxReadBytes` splits it), and hoisting it with a capacity hint saved about 3% of B/op and no wall time; the only large gain needed a 64 K-record hint the finding itself rejected. |
| read-12: Each sealed segment opened allocates a closer closure | One closure per segment opened, about 24 bytes: 136 to 135 allocs/op on `BenchmarkReadStart`, under 1%. |
| read-13: `Log.view` still makes five objects per call with a non-empty active segment | Hot but small: two allocations of 136 and about 150 bytes of 29 KB per `Read`; the finding's headline came from a benchmark that counted its own setup. The large-segment-count part of it is what read-7 (decision 103) fixed. |
| read-14: The pooled window buffer is created at 1 MiB for every `Read`; start at `readChunk` and grow | The point-read saving is real, but the change made `BenchmarkScanAll` allocate about 22% more B/op (the buffer regrows on every scan), and the finding's `ScanAll` figure did not reproduce. |
| reader-18: `fsx.OS.ReadDir` allocates a `DirEntry` per name and discards it | The microbenchmark gain reproduces, but the site is on no documented benchmark's path and the end-to-end saving is small; `os.ReadDir` is the simpler code. |
| reader-21: `encodeConsumer` formats the offset file through `fmt.Sprintf` and a `[]byte` copy | Byte-identical and saves four allocations per `Commit`, but a `Commit` is an atomic file write (about 40 ms here), so the saving is under 5% of its B/op and not measurable in time. |
| reader-22: The `Reader`'s buffer pool allocates 1 MiB on every miss where point reads need 64 KiB | A 64 KiB pool buffer gave no reproducible B/op change on `BenchmarkReaderFollow` (about 50 KB/op before and after, within noise) and the same allocs/op; the claimed baseline did not appear. |

## Deliberately not changed

### simplify-11: the five-attempt retry in `Reader.readTail` (REFUTED)

reader.go:121-139. The loop is specified: build-prompt.md:184-185 says each Reader call reads
`quill.tail` and "on a transient failure retry briefly". No transient failure is demonstrable on
the supported filesystems and the only cost is ~5 ms before a persistent `ErrCorrupt`/EACCES
report. Optional, not tracked: a decision row pointing at build-prompt.md:184 would make the
rationale findable from the code.

### The hand-close convention at log.go:296-313

dedup-18 fixes the eight-path version in `consumerStore.open`; the same shape exists three
times in `Open`. Left for the dedup-18 change to include or not; not a separate issue.

### Files near the ~600-line ceiling

retention_test.go is 615 lines and internal/fsx/crash.go is 596; both are "about 600". The
correctness-21 regression tests go in a new retention_reentrancy_test.go rather than growing
retention_test.go; dedup-20 nets ~45 lines out of crash.go.

### Transient-error caching in `Log.loadSealed` (load.go:441-445)

The writer treats an open-time I/O error as a permanent hole like the Reader does
(correctness-26), but `load` is a one-shot step at `Open`, not a long-lived cache, and the
operator sees it through `Err()`; not tracked.

## type:chore umbrella: trivial fixes batched

Doc edits, each one or two lines, all CONFIRMED, text in audit-docs.md: docs-34 (errors.go:10),
docs-36 (doc.go:22), docs-35 (docs/userguide.md:240-244), docs-38 (docs/design/scope.md:273-277),
docs-41 (docs/format.md:103), docs-42 (docs/format.md:193), docs-43 (docs/design/scope.md:247),
docs-44 (docs/userguide.md:40), docs-40 (docs/introduction.md:13), docs-45 (README.md:71),
docs-46 (per-verb `-dir` usage, regenerate docs/reference.md), and README.md:72 after the
simplify-2 split (docs-37).

Small code changes in the same batch, each behaviour-preserving under the existing suite:
simplify-8 (one-line tail start, plus two tail tests), simplify-10 (`policyConfig`, `cmdCat`,
`follow`), simplify-7 (`flagSet` helper), simplify-6 (`Check` shapes), simplify-5
(`consumerStore.first`), simplify-4/dedup-13 (`openSegmentFlag`), dedup-16 (`SegmentError.Path`
at construction), dedup-18 (`Consumer.init`), dedup-20 (`Sim.enter`), correctness-29 (close the
descriptor on a failed `load`), correctness-25 (`make` the compacted holes slice).

Everything else in the apply-now list (correctness-23, correctness-26, simplify-2, simplify-9
with dedup-17, dedup-14) is a change with its own regression test or its own file moves and
gets its own commit.
