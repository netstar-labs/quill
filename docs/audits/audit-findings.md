# A1 adversarial audit: findings

Date 2026-10-29, against commit 1f4152d. Synthesis of the A1 adversarial audit: four lenses
(simplify, dedup, correctness, docs), each candidate finding attacked by one to three
independent skeptics who read the code, reran the reproductions and tried to refute the claim.
The per-dimension reports carry the full reasoning, anchors and reproductions:
audit-simplify.md, audit-dedup.md, audit-correctness.md, audit-docs.md. The follow-up seed for
the tracker is audit-followups.md.

## Top line

45 candidates were raised; after merging duplicates, 41 distinct items; 40 survive the refutation.

| verdict | items | note |
|---------|-------|------|
| CONFIRMED | 43 | every anchor re-read at synthesis; 13 reproduced by execution |
| PLAUSIBLE | 1 | simplify-7: the fix is equivalent, the claim misdescribes the code by one line |
| REFUTED | 1 | simplify-11: the retry it would remove is specified at build-prompt.md:184-185 |

Duplicates: dedup-13 = simplify-4 (one item); simplify-3 is absorbed by correctness-24 (same
three sites, the correctness lens found the data-loss consequence); docs-37 is resolved by
simplify-2 (the code fix; the doc-only alternative contradicts decision 20); docs-33 and
correctness-32 are one question (what Config the offline writable verbs open with).

By recommendation (over the 40 survivors, duplicates counted once):

| recommendation | count | what it means |
|----------------|-------|---------------|
| apply now | 34 | low-risk, behaviour-preserving or restoring documented behaviour, each with a fix plan and a test plan |
| owner decision | 5 | changes a guarantee, an exit-code contract, an error type callers see, the offline-tool contract, or error text |
| deferred | 1 | correctness-30, a performance refactor too large for this pass |

**The five that matter most** (all CONFIRMED by execution, all apply now):

1. correctness-21: `Prune` can panic (`index out of range`) or silently delete unexpired and
   protected segments after a failed rotation when any cap is set (retention.go:219-243).
2. correctness-22: `Close` racing a `Read`/`Follow` crashes the reading goroutine with a nil
   dereference (log.go:467-471).
3. correctness-23: a data race on the Reader's per-segment index publication (reader.go:247 vs :323),
   on a type documented as safe for concurrent use.
4. correctness-24: `quill prune -max-records 1e6` deletes every sealed segment and
   `quill commit -offset 1e3` rewinds a consumer to 1, both exit 0 (cmd/quill/verbs.go:592, :700).
5. simplify-1: a lock-free Reader panics on a sealed segment whose first frame carries seq 0
   (reader.go:206).

## How items were classified

- **apply now**: CONFIRMED or PLAUSIBLE, low risk, and either behaviour-preserving (a dedup, a
  simplification, a doc fix) or a clear local bug fix with a regression test that *restores* a
  guarantee, a decision, or documented behaviour. A fix that rejects input no document ever
  promised to accept (correctness-24) is in this class.
- **owner decision**: anything that *changes* a guarantee, the public API, the on-disk format,
  the lock rules, an exit-code contract, an error type callers match on, the offline-tool
  contract, or an intended behaviour, or where the fix has a genuine design choice the audit
  should not make. The finder's `semantic` flag was a signal, not the rule: correctness-21, -22,
  -31 and simplify-1, -12 carry the flag but restore documented behaviour, so they are apply now.
- **deferred**: too large or too uncertain for this pass, with the reason.

Within cmd/quill/verbs.go, apply the content edits (correctness-24, simplify-7, -8, -10,
dedup-14, docs-46) before the file split (simplify-2) so the anchors hold; in archive.go apply
simplify-9 before dedup-17.

## Simplify (audit-simplify.md)

| id | anchor | severity | verdict | recommendation |
|----|--------|----------|---------|----------------|
| simplify-1 | reader.go:206 | BUG | CONFIRMED | **apply now**: map a CRC-valid seq-0 first frame to `errBadFrame` before the `reason()` call; regression test from the probe |
| simplify-12 | stat.go:215 | SIMPLIFY | CONFIRMED | **apply now**: inline `list()` in `Check`, report a listing error through `fatal` (decision 67, build-prompt.md:218), delete `mustList` |
| simplify-3 | cmd/quill/verbs.go:592 | SIMPLIFY | CONFIRMED | **apply now** as part of correctness-24 |
| simplify-2 | cmd/quill/verbs.go:1 | SIMPLIFY | CONFIRMED | **apply now**: split into verbs.go, read.go, consumers.go, write.go; update README.md:72; resolves docs-37 |
| simplify-10 | cmd/quill/verbs.go:301 | SIMPLIFY | CONFIRMED | **apply now**: delete `policyConfig`, `cmdCat`, the dead `follow` parameter |
| simplify-6 | stat.go:157 | SIMPLIFY | CONFIRMED | **apply now**: one `verifyBounded` call, `io.ReaderAt`, `Healthy` set in `fatal` |
| simplify-4 | load.go:460 | DEDUP | CONFIRMED | **apply now**: `openSegmentFlag` helper, both names kept (same as dedup-13) |
| simplify-5 | consumer.go:286 | DEDUP | CONFIRMED | **apply now**: `first` as a method over a stored `src`; drop `open`'s `src` parameter (six callers) |
| simplify-7 | cmd/quill/verbs.go:166 | DEDUP | PLAUSIBLE | **apply now**: `flagSet(c, dirDefault)` helper; the reference test pins the output |
| simplify-8 | cmd/quill/verbs.go:462 | SIMPLIFY | CONFIRMED | **apply now**: `max(st.Head+1-min(c.n, st.Head), st.First)`; add tail tests for n > Head and n = 0 |
| simplify-9 | archive.go:161 | SIMPLIFY | CONFIRMED | **apply now**: `io.CopyBuffer` over a `SectionReader`/`OffsetWriter`, inside `if err == nil` |
| simplify-11 | reader.go:121 | SIMPLIFY | REFUTED | **not changed**: build-prompt.md:184-185 specifies the brief retry |

## Dedup (audit-dedup.md)

| id | anchor | severity | verdict | recommendation |
|----|--------|----------|---------|----------------|
| dedup-19 | load.go:364 | DEDUP | CONFIRMED | **owner decision**: a `contiguity` helper unifies four sites but changes one of two overlap strings and adds the `first > from` guard to three sites; recommend approving both (the edge is unreachable from quill's own output) |
| dedup-13 | load.go:60 | DEDUP | CONFIRMED | **apply now**: same item as simplify-4 |
| dedup-16 | load.go:329 | DEDUP | CONFIRMED | **apply now**: `path` parameter on `verifySegment`/`recoverSegment`; delete the four `errors.As` patches |
| dedup-17 | archive.go:151 | DEDUP | CONFIRMED | **apply now**: `atomicfile.WriteFrom(fsys, path, perm, fill)`; `Write` becomes a wrapper; `copySegment` keeps only the copy |
| dedup-14 | cmd/quill/verbs.go:353 | DEDUP | CONFIRMED | **apply now**: `withReader`/`withWriter` closures; usage checks stay before the open; cmdRecover unchanged |
| dedup-18 | consumer.go:80 | SIMPLIFY | CONFIRMED | **apply now**: `(*Consumer).init`, one close-on-error in `open`; optionally the same shape at log.go:296-313 |
| dedup-20 | internal/fsx/crash.go:209 | DEDUP | CONFIRMED | **apply now**: `(*Sim).enter` returning with the lock held, documented like `begin` |

## Correctness (audit-correctness.md)

| id | anchor | severity | verdict | recommendation |
|----|--------|----------|---------|----------------|
| correctness-21 | retention.go:219 | BUG | CONFIRMED | **apply now**: snapshot the targets before `publishLocked`, delete only those still at the front; three regression tests (panic, over-delete, protected consumer) |
| correctness-22 | log.go:468 | BUG | CONFIRMED | **apply now**: in `Close`, retire the active segment when `readers > 0` instead of closing it; regression test under `-race` |
| correctness-23 | reader.go:323 | BUG | CONFIRMED | **apply now**: publish the scan result through an `atomic.Pointer` to an immutable struct; `-race` regression test |
| correctness-24 | cmd/quill/verbs.go:700 | BUG | CONFIRMED | **apply now**: `strconv` at :592, :700, :778 with `exitUsage` on error; tests for `1e6`, `1e3`, `12abc`; absorbs simplify-3 |
| correctness-25 | retention.go:245 | MINOR | CONFIRMED | **apply now**: `kept := make([]Hole, 0, len(l.holes))`; `-race` test plus a `Stat().Holes` stability assertion |
| correctness-26 | reader.go:199 | MINOR | CONFIRMED | **apply now**: cache a bad entry only for `*SegmentError`/bad-first-frame; return I/O errors uncached |
| correctness-31 | load.go:453 | MINOR | CONFIRMED | **apply now**: set `s.size` right after `openSegment` succeeds; test that `Log.Stat().Bytes` matches the files and `Reader.Stat().Bytes` |
| correctness-29 | load.go:418 | MINOR | CONFIRMED | **apply now**: close `l.active.f` when the final `SyncDir` fails; `OpenFiles()` test |
| correctness-27 | reader.go:280 | MINOR | CONFIRMED | **owner decision**: map ENOENT from a sealed segment's `open` to `*GapError` (recommended); the marker-segment retry is optional |
| correctness-28 | cmd/quill/verbs.go:478 | MINOR | CONFIRMED | **owner decision**: check `Close` explicitly in `append`; exit 1 on failure; how to report a durable-but-unpublished sequence under `-sync=true` |
| correctness-32 | cmd/quill/verbs.go:756 | MINOR | CONFIRMED | **owner decision**, with docs-33: offline verbs take `SegmentBytes` from the policy, keep `Retention: -1` and no caps, and `Open` skips `writePolicy` under `NoMaintenance` |
| correctness-30 | read.go:268 | PERF | CONFIRMED | **deferred**: a cached `[]*segRef` invalidated at `rotateLocked`/`deleteLocked`; its own change with a benchmark |

## Docs (audit-docs.md)

| id | anchor | severity | verdict | recommendation |
|----|--------|----------|---------|----------------|
| docs-33 | docs/format.md:161 | BUG | CONFIRMED | **owner decision**, with correctness-32: the code fix (skip `writePolicy` under `NoMaintenance`) is preferred over rewriting five documents to describe the defect |
| docs-34 | errors.go:10 | DOC | CONFIRMED | **apply now**: rewrite the `ErrCorrupt` comment with the text in audit-docs.md item 2 |
| docs-36 | doc.go:22 | DOC | CONFIRMED | **apply now**: text in audit-docs.md item 3 |
| docs-35 | docs/userguide.md:240 | DOC | CONFIRMED | **apply now**: replace lines 240-244 with the text in audit-docs.md item 4 |
| docs-37 | README.md:45 | DOC | CONFIRMED | **apply now via simplify-2**; do not weaken the sentence |
| docs-38 | docs/design/scope.md:274 | DOC | CONFIRMED | **apply now**: restate the ConsumerTTL rule per decisions 45/82; "logged at every prune pass while stale" |
| docs-39 | docs/design/scope.md:347 | DOC | CONFIRMED | **owner decision**: restate guarantee 11 as `+ max(MaxBatchBytes, MaxPayloadBytes + 34)` (recommended) or change admission/validation |
| docs-41 | docs/format.md:103 | DOC | CONFIRMED | **apply now**: split the sentence so the exception binds to "refused when malformed" only |
| docs-42 | docs/format.md:193 | DOC | CONFIRMED | **apply now**: add "then the directory again (decision 65)" to the Rotation row |
| docs-43 | docs/design/scope.md:247 | DOC | CONFIRMED | **apply now**: "minus the records inside holes" |
| docs-44 | docs/userguide.md:40 | DOC | CONFIRMED | **apply now**: "an inline flush at `FlushBytes`, a background flush every `FlushInterval`" |
| docs-46 | cmd/quill/verbs.go:254 | DOC | CONFIRMED | **apply now**: per-verb `-dir` usage string; regenerate docs/reference.md |
| docs-40 | docs/introduction.md:13 | DOC | CONFIRMED | **apply now**: "`quill.Open(quill.Config{Dir: dir})` returns a `*Log`" |
| docs-45 | README.md:71 | DOC | CONFIRMED | **apply now**: add the `hack/powertest/` row |

## What the audit did not find

Three things a reader should know were looked for and not found, since absence is also a result:
no finding touches the frame bytes, the state-file shapes, or the lock rules (decision 53); no
finding proposes a dependency; and the two reproductions that exercised the crash simulator's
fault injection on the writer path (correctness-21, -28) found the flush failure classes of
decision 60 behaving as specified; the bugs were in what happens around them.
