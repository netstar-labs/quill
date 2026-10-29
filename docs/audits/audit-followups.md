# A1 adversarial audit: follow-ups

Date 2026-10-29, against commit 1f4152d. What the A1 audit deliberately does not change in the
apply-now pass, and why. This is the seed for the tracker: each entry below becomes a
`type:audit` issue carrying the `sev:` label shown; the trivial fixes batch under one
`type:chore` umbrella at the end. The full reasoning, anchors and reproductions are in
audit-correctness.md, audit-dedup.md, audit-docs.md and audit-simplify.md; the classification
rule is in audit-findings.md.

## Owner decisions (one issue each)

### correctness-32 + docs-33: what Config the offline writable verbs open with (sev:major)

`quill append`, `roll` and `recover` open with `Config{Retention: -1}` plus defaults
(cmd/quill/verbs.go:474, :728, :756; `writer` at :287-298), so they (a) rotate any active
segment over the default 8 MiB at load (load.go:414 -> flush.go:104) and (b) rewrite
`quill.policy` with their own settings (log.go:305), after which a flagless `quill prune` runs
with retention off and no caps, and `ls`/`check` print UNBOUNDED, until the real writer
restarts. Both reproduced. Decisions 29, 68 and 87 say the file holds the writer's settings for
tools to read.

Why not applied: seeding the offline Config from the whole policy would make the policy's caps
live during an offline append's flush-time seal (`publishLocked` -> `capsLocked`), which
decision 68 ("never prunes") forbids; and guarding `writePolicy` on `NoMaintenance` redefines
that flag as "offline tool". The owner should choose. Recommended: offline verbs keep
`Retention: -1` and no caps, take `SegmentBytes` and `MaxPayloadBytes` from `ReadPolicy` when
it parses (else `MaxSegmentBytes`); `Open` skips `writePolicy` when `cfg.NoMaintenance` is set;
decision 68 records both and that `roll` rotates regardless. If the behaviour is kept instead,
docs/format.md:161, README.md:18, docs/userguide.md:174/199, scope.md:136/267 and decision 29
must say that any writable open rewrites the file.

### correctness-28: `quill append -sync=false` exits 0 when the deferred Close, its only flush, fails (sev:minor)

verbs.go:478 discards `Close`'s error; reproduced on a nearly full filesystem (prints the
sequence, exit 0, record never written) and under the simulator. Why not applied: it changes a
documented verb's exit-code contract (decision 53) and has a real choice for `-sync=true`, where
a Close failure after a successful Sync (decision 60 classes d/e) leaves the records durable
(decision 78). Recommended: explicit `Close` before printing; exit 1 on failure; when Sync had
succeeded, still print the sequence to stdout and the error to stderr.

### correctness-27: a Reader returns a raw `*fs.PathError` when a listed segment is deleted between listing and opening (sev:minor)

reader.go:280-283 and :343-347; reproduced deterministically with an unlink-on-Lstat wrapper.
Guarantee 14 and decision 11 call for a `GapError`. Why not applied: the fix changes the error
type three callers see (Read/Follow, Seek, Archive) and the finder's part 1 adds a retry loop
with sleeps to `view`. Recommended: approve part 2 only (map `fs.ErrNotExist` from a sealed
segment's `open` to `*GapError{From: cur, To: s.last}` after `forget`), which the second skeptic
applied with the full suite green; treat the marker-segment retry as optional.

### dedup-19: one `contiguity` helper for the four copies of the hole/overlap bookkeeping (sev:minor)

load.go:364-376, :380-391, reader.go:326-340, :363-376. Why not applied: unifying the text
changes one of two overlap error strings (untested) and adds the `first > from` guard to three
sites; both are observable on hand-crafted input and the finder rated the risk medium. Both
skeptics showed the `Hole{1,0}` edge is unreachable from quill's own output and the helper
computes every pinned hole identically. Recommended: approve, in a new holes.go, keeping the
writer's `head < tl.Seq` refusal (decision 39) and the reader's marker hole (decision 11) in
their callers.

### docs-39: the disk bound understates the single-record case by up to MaxPayloadBytes + 34 − MaxBatchBytes (sev:minor)

append.go:31 applies MaxBatchBytes only to batches of more than one record; decisions 26/46
admit a single 64 MiB record while decision 69's arithmetic assumes MaxBatchBytes. Inferred from
the code path, not executed. Why not applied: either fix touches a stated guarantee (11).
Recommended: restate the bound as `MaxBytes + SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)`
at scope.md:239, :347, userguide.md:152 and decision 69, and say at scope.md:119 /
append.go:17-18 that a single record is bounded by MaxPayloadBytes alone.

### correctness-24 follow-on: unit suffixes for byte flags (sev:trivial)

Once the strict `strconv` parse lands, `-max-bytes 1M` is refused with exit 2. Whether to
accept K/M/G suffixes by design is a CLI-shape choice (decision 86 territory), not an audit fix.

## Deferred (one issue)

### correctness-30: `Log.view` allocates ~2 objects per sealed segment per read, under the log mutex (sev:perf)

read.go:268-285; measured 2019 allocs/op at 1000 sealed segments, linear in the sealed count,
with Append and Flush waiting behind the mutex. CONFIRMED, not a defect. Deferred because it is
a cache with invalidation on the hot read path, larger than a local fix: a `[]*segRef` for the
sealed segments rebuilt where `l.sealed` changes (`rotateLocked`, flush.go:163; `deleteLocked`,
retention.go:243), shared across views, landed with an allocation benchmark.

## Deliberately not changed

### simplify-11: the five-attempt retry in `Reader.readTail` (REFUTED)

reader.go:121-139. The loop is specified: build-prompt.md:184-185 says each Reader call reads
`quill.tail` and "on a transient failure retry briefly". No transient failure is demonstrable on
the supported filesystems and the only cost is ~5 ms before a persistent `ErrCorrupt`/EACCES
report. Optional nit, not tracked: a decision row naming build-prompt.md:184 would make the
rationale findable from the code.

### The hand-close convention at log.go:296-313

dedup-18 fixes the eight-path version in `consumerStore.open`; the same shape exists three
times in `Open`. Left for the dedup-18 change to include or not; not a separate issue.

### Files near the ~600-line ceiling

retention_test.go is 615 lines and internal/fsx/crash.go is 596; both are "about 600". The
correctness-21 regression tests go in a new retention_reentrancy_test.go rather than growing
retention_test.go; dedup-20 nets ~45 lines out of crash.go.

## type:chore umbrella: trivial fixes batched

Doc edits, each one or two lines, all CONFIRMED, text in audit-docs.md: docs-34 (errors.go:10),
docs-36 (doc.go:22), docs-35 (docs/userguide.md:240), docs-38 (docs/design/scope.md:273-277),
docs-41 (docs/format.md:103), docs-42 (docs/format.md:193), docs-43 (docs/design/scope.md:247),
docs-44 (docs/userguide.md:40), docs-40 (docs/introduction.md:13), docs-45 (README.md:71),
docs-46 (per-verb `-dir` usage, regenerate docs/reference.md), and README.md:72 after the
simplify-2 split.

Small code simplifications in the same batch: simplify-8 (one-line tail start), simplify-10
(`policyConfig`, `cmdCat`, `follow`), simplify-7 (`flagSet` helper), simplify-6 (`Check` shapes),
simplify-5 (`consumerStore.first`), simplify-4/dedup-13 (`openSegmentFlag`), dedup-16
(`SegmentError.Path` at construction), dedup-18 (`Consumer.init`), dedup-20 (`Sim.enter`),
correctness-29 (close the descriptor on the final `SyncDir` failure), correctness-25
(`make` the compacted holes slice).

Everything else in the apply-now list (correctness-21, -22, -23, -24, -26, -31, simplify-1,
-2, -9/dedup-17, -12, dedup-14) is a change with its own regression test and gets its own commit.
