# A1 adversarial audit: findings

Date 2026-10-29, against HEAD b52b746 (the source is identical to 1f4152d; only docs/audits
changed in between). Synthesis of the A1 adversarial audit: four lenses (simplify, dedup,
correctness, docs), each candidate attacked by one to three independent skeptics who read the
code, reran the reproductions and tried to refute the claim. The per-dimension reports carry
the anchors, each skeptic's reasoning and the reproductions: audit-simplify.md,
audit-dedup.md, audit-correctness.md, audit-docs.md. The tracker seed is audit-followups.md.
Every anchor cited here was re-read from the tree at synthesis time; no repository file
outside docs/audits was edited by the audit.

## Top line

45 candidates were raised (ids 1-46; dedup-15 was never raised), judged by 62 skeptic votes;
20 items carry at least one reproduction by execution. 44 survive refutation.

| verdict | items | note |
|---------|-------|------|
| CONFIRMED | 43 | correctness-27 by one CONFIRMED and one PLAUSIBLE vote |
| PLAUSIBLE | 1 | simplify-7: the fix is equivalent; the claim misdescribes the code by one line |
| REFUTED | 1 | simplify-11: the retry it would remove is specified at build-prompt.md:184-185 |

Duplicates, counted once below: dedup-13 = simplify-4 (one item); simplify-3 is absorbed by
correctness-24 (same three parse sites; the correctness lens found the deletion consequence);
docs-33 and correctness-32 are one question (what Config the offline writable verbs open with);
docs-37 is resolved by simplify-2 (its doc-only alternative contradicts decision 20).

By recommendation, over the 41 distinct surviving items:

| recommendation | count | what it means |
|----------------|-------|---------------|
| apply now | 29 | low risk, behaviour-preserving, not flagged semantic; each has a fix plan and a test plan |
| owner decision | 11 | flagged semantic by the finder, or touching a guarantee's text; each has a recommended answer |
| deferred | 1 | correctness-30, a performance refactor too large for this pass |

**The six that matter most.** All CONFIRMED by execution; five are owner decisions only
because their fixes change observable behaviour, and the recommended answer for each is
"approve", with the fix and test written out in the dimension report.

1. correctness-21: `Prune` panics (`index out of range`) or silently deletes unexpired and
   protected segments after a failed rotation when any cap is set (retention.go:219-243,
   flush.go:141). Reproduced three times, both outcomes.
2. correctness-22: `Close` racing a `Read`/`Follow` crashes the reading goroutine with a nil
   dereference (log.go:467-472 against read.go:265-266, :289). Reproduced three times.
3. correctness-23: a data race on the Reader's per-segment index publication (reader.go:247
   against :323) on a type documented as safe for concurrent use. Apply now.
4. correctness-24: `quill prune -max-records 1e6` deletes every sealed segment and `quill
   commit -offset 1e3` rewinds a consumer to 1, both exit 0 (cmd/quill/verbs.go:592, :700).
5. simplify-1: a lock-free Reader panics on a sealed segment whose first frame carries seq 0
   (reader.go:206, `reason(nil)` at load.go:263).
6. correctness-32 with docs-33: `quill append`, `roll` and `recover` open with the default
   Config, rotate any active segment over 8 MiB at load and rewrite `quill.policy` with
   retention off, no caps and no protected consumers (verbs.go:474/:728/:756, log.go:305).

## How items were classified

The rule from the task was applied literally, with the finder's `semantic` flag as the
discriminator:

- **apply now**: CONFIRMED or PLAUSIBLE, low risk, behaviour-preserving (a dedup, a
  simplification, a doc fix, or a local bug fix with a regression test) and not flagged
  semantic.
- **owner decision**: anything flagged semantic (it changes observable behaviour, an exit code,
  an error type callers match on, `Check`'s verdict, `Stat` numbers, or the offline-tool
  contract), plus docs-39, whose fix rewrites the text of guarantee 11 even though the finder
  did not flag it. Five of the eleven are bug fixes that restore documented behaviour
  (correctness-21, -22, -24, -31, simplify-1); they are listed with the answer "approve" so the
  owner's decision is one word each.
- **deferred**: too large or too uncertain for this pass, with the reason. correctness-30 is
  CONFIRMED rather than PLAUSIBLE but is a cache with invalidation on the hot read path, not a
  local fix.

Ordering within files: in cmd/quill/verbs.go land the content edits (simplify-7, -8, -10,
dedup-14, docs-46, and correctness-24 if approved) before the file split (simplify-2) so the
anchors hold, or re-anchor after; in archive.go land simplify-9 before dedup-17; in consumer.go
land simplify-5 (drops `open`'s `src` parameter) with or before dedup-18 (restructures `open`).

## Simplify (audit-simplify.md)

| id | anchor | severity | verdict | recommendation |
|----|--------|----------|---------|----------------|
| simplify-1 | reader.go:206 | BUG (semantic) | CONFIRMED | **owner decision**, recommend approve: `if ferr == nil && rec.Seq == 0 { ferr = errBadFrame }` before :205; regression test from the probe |
| simplify-12 | stat.go:215 | SIMPLIFY (semantic) | CONFIRMED | **owner decision**, recommend `fatal`: inline `list()` in `Check`, report a listing error as a `consumer` problem (decision 67), delete `mustList` |
| simplify-3 | cmd/quill/verbs.go:592 | SIMPLIFY (semantic) | CONFIRMED | **owner decision** as part of correctness-24 |
| simplify-2 | cmd/quill/verbs.go:1 | SIMPLIFY | CONFIRMED | **apply now**: split into verbs.go, read.go, consumers.go, write.go; update README.md:72; resolves docs-37 |
| simplify-10 | cmd/quill/verbs.go:301 | SIMPLIFY | CONFIRMED | **apply now**: delete `policyConfig`, `cmdCat`, the dead `follow` parameter |
| simplify-6 | stat.go:157 | SIMPLIFY | CONFIRMED | **apply now**: one `verifyBounded` call, `io.ReaderAt`, `Healthy` set in `fatal` |
| simplify-9 | archive.go:161 | SIMPLIFY | CONFIRMED | **apply now**: `io.CopyBuffer` over a `SectionReader`/`OffsetWriter`, inside `if err == nil`; before dedup-17 |
| simplify-8 | cmd/quill/verbs.go:462 | SIMPLIFY | CONFIRMED | **apply now**: `max(st.Head+1-min(c.n, st.Head), st.First)`; add tail tests for n > Head and n = 0 |
| simplify-7 | cmd/quill/verbs.go:166 | DEDUP | PLAUSIBLE | **apply now**: `flagSet(c, dirDefault)` helper; the reference test pins the output |
| simplify-4 | load.go:460 | DEDUP | CONFIRMED | **apply now**: `openSegmentFlag` helper, both names kept (= dedup-13) |
| simplify-5 | consumer.go:286 | DEDUP | CONFIRMED | **apply now**: `first` as a method over a stored `src`; drop `open`'s `src` parameter (six callers) |
| simplify-11 | reader.go:121 | SIMPLIFY | REFUTED | **not changed**: build-prompt.md:184-185 specifies "on a transient failure retry briefly" |

## Dedup (audit-dedup.md)

| id | anchor | severity | verdict | recommendation |
|----|--------|----------|---------|----------------|
| dedup-19 | load.go:364 | DEDUP (semantic) | CONFIRMED | **owner decision**, recommend approve: a `contiguity` helper in holes.go for the four sites; it unifies one of two overlap strings and adds the `first > from` guard (unreachable edge); the two tail steps stay in their callers |
| dedup-17 | archive.go:151 | DEDUP | CONFIRMED | **apply now**: `atomicfile.WriteFrom(fsys, path, perm, fill)`; `Write` becomes a wrapper; `copySegment` keeps only the copy; after simplify-9 |
| dedup-16 | load.go:329 | DEDUP | CONFIRMED | **apply now**: `path` parameter on `verifySegment`/`recoverSegment`; delete the four `errors.As` patches |
| dedup-14 | cmd/quill/verbs.go:353 | DEDUP | CONFIRMED | **apply now**: `withReader`/`withWriter` closures; usage checks stay before the open; `cmdRecover` unchanged |
| dedup-13 | load.go:60 | DEDUP | CONFIRMED | **apply now**: same item as simplify-4 |
| dedup-18 | consumer.go:80 | SIMPLIFY | CONFIRMED | **apply now**: `(*Consumer).init`, one close-on-error in `open`; add a lock-release test; the same shape at log.go:296-313 is optional |
| dedup-20 | internal/fsx/crash.go:209 | DEDUP | CONFIRMED | **apply now**: `(*Sim).enter` returning with the lock held, documented like `begin` |

## Correctness (audit-correctness.md)

| id | anchor | severity | verdict | recommendation |
|----|--------|----------|---------|----------------|
| correctness-21 | retention.go:219 | BUG (semantic) | CONFIRMED | **owner decision**, recommend approve the snapshot variant: record the targets before `publishLocked`, delete only the leading run still present; four regression tests (panic, over-delete, fresh, protected) in a new retention_reentrancy_test.go |
| correctness-22 | log.go:468 | BUG (semantic) | CONFIRMED | **owner decision**, recommend approve: in `Close`, retire the active segment when `readers > 0` instead of closing it (as rotation does); regression test under `-race` |
| correctness-23 | reader.go:323 | BUG | CONFIRMED | **apply now**: publish the scan result through an `atomic.Pointer` to an immutable struct; `-race` regression test |
| correctness-24 | cmd/quill/verbs.go:700 | BUG (semantic) | CONFIRMED | **owner decision**, recommend strict decimals: `strconv` at :592, :700, :778 with `exitUsage`; tests for `1e6`, `1e3`, `12abc`; K/M/G suffixes as a separate chore if wanted; absorbs simplify-3 |
| correctness-32 | cmd/quill/verbs.go:756 | MINOR (semantic) | CONFIRMED | **owner decision**, one issue with docs-33: offline verbs take `SegmentBytes` and `MaxPayloadBytes` from the policy, keep `Retention: -1` and no caps; `Open` skips `writePolicy` under `NoMaintenance`; decision 68 records that `roll` rotates regardless |
| correctness-28 | cmd/quill/verbs.go:478 | MINOR (semantic) | CONFIRMED | **owner decision**: explicit `Close` before printing, exit 1 on failure; under `-sync=true` after a successful Sync still print the sequence and the error to stderr |
| correctness-27 | reader.go:280 | MINOR (semantic) | CONFIRMED | **owner decision**, recommend part 2 only: map `fs.ErrNotExist` from a sealed segment's `open` to `*GapError{From: cur, To: s.last}` after `forget`; the marker-segment retry is optional |
| correctness-26 | reader.go:199 | MINOR | CONFIRMED | **apply now**: cache a bad entry only for a `*SegmentError` or a bad first frame; return I/O errors uncached; regression test with a one-off injected open failure |
| correctness-31 | load.go:453 | MINOR (semantic) | CONFIRMED | **owner decision**, recommend approve: set `s.size` right after `openSegment` succeeds; test that `Log.Stat().Bytes` matches the files and `Reader.Stat().Bytes` on a damaged log |
| correctness-25 | retention.go:245 | MINOR | CONFIRMED | **apply now**: `kept := make([]Hole, 0, len(l.holes))`; `-race` test plus a `Stat().Holes` stability assertion |
| correctness-29 | load.go:418 | MINOR | CONFIRMED | **apply now**: close `l.active.f` when `load` fails after setting it; `OpenFiles()` test |
| correctness-30 | read.go:268 | PERF | CONFIRMED | **deferred**: a cached `[]*segRef` invalidated at `rotateLocked`/`deleteLocked`; its own change with a benchmark |

## Docs (audit-docs.md)

| id | anchor | severity | verdict | recommendation |
|----|--------|----------|---------|----------------|
| docs-33 | docs/format.md:161 | BUG (semantic) | CONFIRMED | **owner decision**, with correctness-32: the code fix is preferred over rewriting five documents to describe the defect; the replacement text is in audit-docs.md item 1 if the behaviour is kept |
| docs-34 | errors.go:10 | DOC | CONFIRMED | **apply now**: rewrite the `ErrCorrupt` comment with the text in audit-docs.md item 2 |
| docs-36 | doc.go:22 | DOC | CONFIRMED | **apply now**: text in audit-docs.md item 3 |
| docs-35 | docs/userguide.md:240 | DOC | CONFIRMED | **apply now**: replace lines 240-244 with the text in audit-docs.md item 4 |
| docs-37 | README.md:45 | DOC | CONFIRMED | **apply now via simplify-2**; do not weaken the sentence |
| docs-38 | docs/design/scope.md:274 | DOC | CONFIRMED | **apply now**: restate the ConsumerTTL rule per decisions 45/82; "logged at every prune pass while stale" |
| docs-39 | docs/design/scope.md:347 | DOC | CONFIRMED | **owner decision**: restate guarantee 11 as `+ max(MaxBatchBytes, MaxPayloadBytes + 34)` (recommended) rather than change admission or validation |
| docs-41 | docs/format.md:103 | DOC | CONFIRMED | **apply now**: split the sentence so the exception binds to "refused when malformed" only |
| docs-42 | docs/format.md:193 | DOC | CONFIRMED | **apply now**: add "then the directory again (decision 65)" to the Rotation row |
| docs-43 | docs/design/scope.md:247 | DOC | CONFIRMED | **apply now**: "minus the records inside holes" |
| docs-44 | docs/userguide.md:40 | DOC | CONFIRMED | **apply now**: "an inline flush at `FlushBytes`, a background flush every `FlushInterval`" |
| docs-46 | cmd/quill/verbs.go:254 | DOC | CONFIRMED | **apply now**: per-verb `-dir` usage string; regenerate docs/reference.md |
| docs-40 | docs/introduction.md:13 | DOC | CONFIRMED | **apply now**: "`quill.Open(quill.Config{Dir: dir})` returns a `*Log`" |
| docs-45 | README.md:71 | DOC | CONFIRMED | **apply now**: add the `hack/powertest/` row |

## What the audit did not find

Absence is also a result. No finding touches the frame bytes, the state-file shapes, or the
lock rules (decision 53); no finding proposes a dependency; every proposed fix stays inside
internal/fsx for file operations. The reproductions that exercised the crash simulator's
fault injection on the writer path (correctness-21, -28, -29) found the flush failure classes
of decision 60 behaving as specified; the bugs were in what happens around them (a stale
index after a nested delete, a dropped Close error, an unclosed descriptor). The one REFUTED
item (simplify-11) was refuted by the specification, not by the code: the loop it would have
removed is what build-prompt.md:184-185 asks for.
