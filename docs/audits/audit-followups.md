# A1 adversarial audit: follow-ups

Date 2026-10-29, against HEAD b52b746 (the source is identical to 1f4152d). What the A1 audit
deliberately does not change in the apply-now pass, and why. This is the seed for the tracker:
each entry under "Owner decisions" and "Deferred" becomes a `type:audit` issue carrying the
`sev:` label shown; the trivial fixes batch under one `type:chore` umbrella at the end. The
anchors, each skeptic's reasoning and the reproductions are in audit-correctness.md,
audit-dedup.md, audit-docs.md and audit-simplify.md; the classification rule is in
audit-findings.md. Every owner decision below comes with a recommended answer; where that
answer is "approve", the fix and its test are already written out in the dimension report.

## Owner decisions (one issue each)

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

### simplify-1: a Reader panics on a sealed segment whose first frame carries seq 0 (sev:major)

reader.go:204-206 calls `reason(ferr)` with `ferr == nil` when the frame is CRC-valid and seq 0;
`reason` (load.go:263) dereferences the nil error. Reproduced twice; with the one-line guard the
Reader returns `GapError{1,1}` as decisions 11/38 and guarantee 14 require. Why owner: flagged
semantic (panic becomes a gap). Recommended: approve; test from the probe.

### correctness-32 + docs-33: what Config the offline writable verbs open with (sev:major)

`quill append`, `roll` and `recover` open with `Config{Retention: -1}` plus defaults
(verbs.go:474, :728, :756; `writer` :287-298), so they (a) rotate any active segment over the
default 8 MiB at load (load.go:414 -> flush.go:104) and (b) rewrite `quill.policy` with their
own settings (log.go:305), after which a flagless `quill prune` runs with retention off and no
caps and `ls`/`check` print UNBOUNDED until the real writer reopens. Both reproduced. Decisions
29, 68 and 87 say the file holds the writer's settings for tools to read. Why owner: seeding the
offline Config from the whole policy would make the policy's caps live during an offline
append's flush-time seal (`publishLocked` -> `capsLocked`), which decision 68 ("never prunes")
forbids; guarding `writePolicy` on `NoMaintenance` redefines that flag as "offline tool"; and
skip-rotation-under-NoMaintenance alone would not cover `quill append`. Recommended: offline
verbs keep `Retention: -1` and no caps, take `SegmentBytes` and `MaxPayloadBytes` from
`ReadPolicy` when it parses (else `MaxSegmentBytes`); `Open` skips `writePolicy` when
`cfg.NoMaintenance` is set; decision 68 records both and that `roll` rotates regardless. If the
behaviour is kept instead, docs/format.md:161, README.md:18, docs/userguide.md:174/199,
scope.md:136/267 and decision 29 must say that any writable open rewrites the file (text in
audit-docs.md item 1).

### simplify-12: `Check` says HEALTHY when consumers/ cannot be listed (sev:minor)

stat.go:215-218 `mustList` drops the `ReadDir` error; retention treats the same condition as
`broken` (retention.go:31-35). Reproduced for a missing and a mode-000 directory on the OS
backend and with `fs.ErrPermission` injected on the Sim. Why owner: `quill check` would start
exiting 3 for a condition it passes today, and the finder left `fatal` versus `warn` open.
Recommended: `fatal`, per decision 67 and build-prompt.md:218; inline `list()` and delete
`mustList`.

### correctness-31: a damaged sealed segment's bytes count toward neither `MaxBytes` nor `Stat.Bytes` (sev:minor)

load.go:453-456 assigns `s.size` only on the verify-success path. Why owner: `MaxBytes` evicts
earlier and `Log.Stat().Bytes` changes on a damaged log (today it disagrees with
`Reader.Stat().Bytes`, which uses `Lstat`); flagged semantic. Recommended: approve (set `s.size`
right after `openSegment` succeeds), restoring decision 32 and guarantee 11. Residual: a segment
whose `openSegment` itself fails still counts 0.

### correctness-28: `quill append -sync=false` exits 0 when the deferred `Close`, its only flush, fails (sev:minor)

verbs.go:478 discards `Close`'s error; reproduced on the simulator (prints the sequence, exit
0, record never written). Why owner: it changes a documented verb's exit-code contract
(decision 53) and has a real choice for `-sync=true`, where a Close failure after a successful
Sync (decision 60 classes d/e) leaves the records durable (decision 78). Recommended: explicit
`Close` before printing; exit 1 on failure; when Sync had succeeded, still print the sequence to
stdout and the error to stderr.

### correctness-27: a Reader returns a raw `*fs.PathError` when a listed segment is deleted between listing and opening (sev:minor)

reader.go:280-283 and :343-347; read.go:181-187 passes the raw error up once the segment's
index is cached; reproduced deterministically with an unlink-on-Lstat wrapper. Guarantee 14 and
decision 11 call for a `GapError`. Why owner: the fix changes the error type three callers see
(Read/Follow, Seek, Archive) and part 1 adds a retry loop with sleeps to `view`. Recommended:
approve part 2 only (map `fs.ErrNotExist` from a sealed segment's `open` to `*GapError{From:
cur, To: s.last}` after `forget`), which one skeptic applied with the full suite green; treat
the marker-segment retry as optional.

### dedup-19: one `contiguity` helper for the four copies of the hole/overlap bookkeeping (sev:minor)

load.go:364-376, :380-391, reader.go:326-340, :363-376. Why owner: unifying the text changes one
of two overlap error strings (untested) and adds the `first > from` guard to three sites; both
are observable on hand-crafted input and the finder rated the risk medium. Both skeptics showed
the `Hole{1,0}` edge is unreachable from quill's own output and that the helper computes every
pinned hole identically. Recommended: approve, in a new holes.go, keeping the writer's `head <
tl.Seq` refusal (decision 39) and the reader's marker hole (decision 11) in their callers.

### docs-39: the disk bound understates the single-record case by up to `MaxPayloadBytes + 34 − MaxBatchBytes` (sev:minor)

append.go:31 applies `MaxBatchBytes` only to batches of more than one record; decisions 26/46
admit a single 64 MiB record while decision 69's arithmetic assumes `MaxBatchBytes`. Inferred
from the code path, not executed. Why owner: either fix rewrites the text of guarantee 11 or
contradicts decision 46. Recommended: restate the bound as `MaxBytes + SegmentBytes +
MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)` at scope.md:239, :347,
userguide.md:152 and decision 69, and say at scope.md:119 / append.go:17-18 that a single record
is bounded by `MaxPayloadBytes` alone.

## Applied

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

## Deferred (one issue)

### correctness-30: `Log.view` allocates ~2 objects per sealed segment per read, under the log mutex (sev:perf)

read.go:268-285; measured 2019 allocs/op at 1000 sealed segments, linear in the sealed count,
with Append and Flush waiting behind the mutex. CONFIRMED, not a defect. Deferred because the
fix is a cache with invalidation on the hot read path, larger than a local change: a
`[]*segRef` for the sealed segments rebuilt where `l.sealed` changes (`rotateLocked`,
flush.go:163; `deleteLocked`, retention.go:243), shared across views, landed with an allocation
benchmark.

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
