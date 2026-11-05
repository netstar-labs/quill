# A1 adversarial audit: docs

Date 2026-10-29, against HEAD b52b746 (the source is identical to 1f4152d; only docs/audits
changed in between). Dimension D (documentation against code) of the A1 audit: every
candidate, the skeptic's verdict and reason, and the correct text. The rule applied by every
skeptic: docs/design/decisions.md is authoritative; when prose and code disagree, the decision
says which is wrong. Every anchor was re-read from the tree at synthesis time. Ranked by what
a reader would get wrong.

Verdicts: 14 CONFIRMED. docs-33 is the same question as correctness-32 (one issue); docs-37 is
resolved by simplify-2 (the code fix), since its doc-only alternative contradicts decision 20.

| # | id | anchor | severity | verdict | recommendation |
|---|----|--------|----------|---------|----------------|
| 1 | docs-33 | docs/format.md:161 | BUG (semantic) | CONFIRMED | owner decision, with correctness-32 |
| 2 | docs-34 | errors.go:10 | DOC | CONFIRMED | apply now |
| 3 | docs-36 | doc.go:22 | DOC | CONFIRMED | apply now |
| 4 | docs-35 | docs/userguide.md:240 | DOC | CONFIRMED | apply now |
| 5 | docs-37 | README.md:45 | DOC | CONFIRMED | apply now via simplify-2 |
| 6 | docs-38 | docs/design/scope.md:274 | DOC | CONFIRMED | apply now |
| 7 | docs-39 | docs/design/scope.md:347 | DOC | CONFIRMED | owner decision (a guarantee's text) |
| 8 | docs-41 | docs/format.md:103 | DOC | CONFIRMED | apply now |
| 9 | docs-42 | docs/format.md:193 | DOC | CONFIRMED | apply now |
| 10 | docs-43 | docs/design/scope.md:247 | DOC | CONFIRMED | apply now |
| 11 | docs-44 | docs/userguide.md:40 | DOC | CONFIRMED | apply now |
| 12 | docs-46 | cmd/quill/verbs.go:254 | DOC | CONFIRMED | apply now |
| 13 | docs-40 | docs/introduction.md:13 | DOC | CONFIRMED | apply now |
| 14 | docs-45 | README.md:71 | DOC | CONFIRMED | apply now |

## 1. docs-33: offline writable verbs overwrite quill.policy with their own Config (CONFIRMED, semantic)

**Anchors.** Prose: docs/format.md:161-163 ("the settings the writer last opened with ...
offline tools read it"), README.md:18 ("the writer's Config, for tools"), docs/userguide.md:174
and :199, scope.md:136 and :267, decision 29. Code: log.go:305 calls `l.writePolicy()`
unconditionally in `Open` (the only `NoMaintenance` consumers are flush.go:257 and the Config
comment at log.go:61-63); cmd/quill/verbs.go:474, :728, :756 open with `quill.Config{Retention:
-1}` through `writer` (:287-298), which sets only `Dir` and `NoMaintenance`; `policyConfig`
(:301-307) is used only by `cmdPrune` (:681).

**Skeptic (CONFIRMED).** Reproduced: a writer opened with `Retention: time.Hour, MaxBytes:
64<<20, MaxRecords: 1000, ProtectedConsumers: ["billing"]`, then `quill roll`; the policy went
from `{"retention":"1h0m0s", ..., "max_bytes":67108864, "max_records":1000,
"protected_consumers":["billing"]}` to `{"retention":"-1ns", ..., "max_bytes":0,
"max_records":0, "protected_consumers":[]}` (the finding's 4 MiB `MaxBytes` is rejected by
`validate`, "below twice SegmentBytes"; that does not affect the claim). Decision 29 exists
because "a tool looking at one directory cannot know its limits otherwise", decision 68 says
offline tools never prune and `prune` needs a parsed policy, decision 87 says `ReadPolicy`
exists "so quill prune and quill ls act on the writer's settings"; after a `roll` the file no
longer carries them, so the intent of 29/68/87 is defeated. The narrow reading that the offline
tool is itself "the writer that last opened" is what the code implements and what decision 29's
rationale contradicts.

**Disposition.** Owner decision, one issue with correctness-32 (audit-followups.md). Preferred:
the code fix (guard log.go:305 with `!cfg.NoMaintenance`, or seed the offline Config from the
policy). If the behaviour is kept, the correct text at docs/format.md:161 (and README.md:18,
userguide.md:174/199, scope.md:136/267, decision 29) is: "the settings of the last writable
open, including `quill append`, `roll` and `recover`, which rewrite it with their own Config
(retention off, no caps, no protected consumers); after one, `prune` must be given explicit
flags until the writer reopens."

## 2. docs-34: the `ErrCorrupt` comment says any unverifiable sealed segment makes `Open` refuse (CONFIRMED)

**Anchors.** errors.go:10-12: "ErrCorrupt reports damage below the durable marker: a sealed
segment that does not verify, or bytes of the newest segment below quill.tail.off that do not,
or a marker whose sequence is above the last valid frame. Open refuses to write". Code:
`loadSealed` stores a verify failure in `s.bad` (load.go:447-454); `load` records the first
into `l.loadErr`, sets `pendingHole` and continues (:357-362); `Open` returns nil. `ErrCorrupt`
from `load` comes only from a malformed `quill.format`/`quill.tail` (:278, :287), no segments or
a marker above the newest (:294, :298), the newest segment's `recoverSegment` failure
(:326-333), overlaps (:365-367, :382-384) and `head < tl.Seq` (:402-405).

**Skeptic (CONFIRMED).** Decision 39 (decisions.md:47) says "a hole is raised for readers
(decision 38), and if it is the newest segment Open fails with ErrCorrupt", and
recovery_test.go:92-107 asserts `Open` succeeds with `Err()` matching `ErrCorrupt` and
`Read(1,10)` returning a gap. The comment's first clause is true (a sealed segment's
`*SegmentError` does report `ErrCorrupt`, via `Err()` and `SegmentError.Is` at load.go:55); the
wrong part is "Open refuses to write" covering it. doc.go:22-23 has the same drift (docs-36).

**Correct text (apply now).** "ErrCorrupt reports damage below the durable marker. Open returns
it for the newest segment (bytes below quill.tail.off that do not verify, a marker above the
last valid frame, an overlap between segments, a damaged quill.tail or quill.format). A sealed
segment that does not verify does not fail Open: it becomes a hole, reads return a *GapError
over it, and Err() returns the *SegmentError, which matches ErrCorrupt. The user guide has the
procedure."

## 3. docs-36: the package comment says damage below the durable marker fails `Open` (CONFIRMED)

**Anchors.** doc.go:22-23: "a damaged or missing segment is a GapError, and damage below the
durable marker fails Open with ErrCorrupt". Every byte of a sealed segment is below the marker,
and the package's own definition of the phrase (errors.go:10-12, scope.md:75) includes sealed
segments.

**Skeptic (CONFIRMED).** Same evidence as docs-34: decision 39, load.go:357-362 and :409-418,
log.go:487-498, recovery_test.go:93-101. The sentence makes a false claim about an error path a
caller would build a startup health check on. The skeptic added decision 39's third case.

**Correct text (apply now).** "Corruption is never served silently: a damaged or missing sealed
segment becomes a hole that reads report as a GapError and Err() names; damage in the newest
segment below the durable marker, or a marker whose sequence is above the last valid frame,
fails Open with ErrCorrupt."

## 4. docs-35: user guide section 10 says damage in the newest's sealed predecessor fails `Open` (CONFIRMED)

**Anchors.** docs/userguide.md:240-242: "in a sealed segment that is the newest's predecessor,
or in the newest segment below quill.tail.off, or when the marker names a sequence above the
last valid frame". load.go:300-313 verifies every sealed segment through `loadSealed` with no
special case for the predecessor; the loop at :356-377 treats any `s.bad` alike; the hole is
closed at the active segment's first sequence (:380-391). The same section's step 3 (:249-251)
already presupposes the hole behaviour.

**Skeptic (CONFIRMED).** Reproduced the predecessor case directly (segments 0, 1, 2; one byte of
segment 1 flipped): `Open` succeeded, `Err()` named `/log/0000000001.log`, `holes=[{6 10}]`,
`Read(6,10)` returned `records 6 to 10 are unavailable`. Corrections to the finding: its line
anchors were ~25 lines off, and recovery_test.go:225 is a stale-marker test; the reproduction is
the direct evidence. docs/audits/passes/self-review-recovery-and-reader.md:16 states the same rule.

**Correct text (apply now), replacing lines 240-244.** "`Open` fails with **`ErrCorrupt`** when
the newest segment is damaged below `quill.tail.off`, when the marker names a sequence above the
last valid frame, when two segments overlap, or when `quill.tail` or `quill.format` is
malformed. Damage in any sealed segment, the newest's predecessor included, does not fail
`Open`: the segment becomes a hole, `Err()` names it, `quill check` reports it, and reads stop
before it with a gap. There is no `repair` verb in v0.1.0 (decision 64). The procedure:"

## 5. docs-37: README says no file is over about 600 lines; cmd/quill/verbs.go is 786 (CONFIRMED)

**Anchors.** README.md:45 "No file is over about 600 lines." `wc -l`: cmd/quill/verbs.go 786,
retention_test.go 615, internal/fsx/crash.go 596, internal/fsx/crash_test.go 570, log.go 534.
Decision 20 (decisions.md:28), build-prompt.md:41 (under a layout tree that includes
`cmd/quill/`), CONTRIBUTING.md:19 and decision 73 all state the rule without a library-only
qualifier; docs/architecture.md:31 narrows it to the library, which holds. hack/gate.sh has no
line-count step.

**Skeptic (CONFIRMED).** The README sentence is the faithful restatement of decision 20 and the
code is in breach by ~30%; the doc-only alternative ("No library file ...") would contradict
decision 20, build-prompt.md:41 and CONTRIBUTING.md:19 and should not be taken.

**Disposition (apply now via simplify-2).** Split the file as audit-simplify.md item 4 says;
README.md:45 stays; README.md:72 gains the new file names. 615 and 596 are "about 600".

## 6. docs-38: scope describes the in-memory ConsumerTTL timer that decision 45 replaced (CONFIRMED)

**Anchors.** docs/design/scope.md:273-274: "counted from the later of its offset file's
modification time and the first pass that saw it behind (kept in memory, reset when it catches
up)"; :277 "logged once per pass change". Code: retention.go:47-61 marks a consumer stale when
`offset <= head`, `now - mtime >= TTL`, and the `At` of the record at the offset (or of the
first retained record on a `GapError`, :53-57) is older than the TTL; `consumerView` is rebuilt
each pass (:28-31); the only persistent field is `stale []string` (log.go:231), names for Stat;
:92-95 warns for every stale name on every pass.

**Skeptic (CONFIRMED).** Decision 45 (decisions.md:53) says "with no in-memory state ... both
its offset file's mtime and the At of the record at its offset are older than the TTL" and
names the in-memory timestamp as the rejected alternative (writer restarts reset it); decision
82 adds the first-retained fallback. The Config-table row (scope.md:125) and userguide.md:107-108
are compatible with the At-based rule. Decision 50 states no logging frequency, so the code's
per-pass warning is the fact to document. Semantic for an operator sizing the TTL: a writer
restart does not reset the clock.

**Correct text (apply now).** Lines 273-276: "a protected consumer is stale when it is behind
(offset ≤ Head) and both its offset file's modification time and the At of the record at its
offset (or of the first retained record when a cap already deleted it, decision 82) are older
than the TTL; the prune pass reads that one frame and keeps no state between passes (decision
45)." Line 277: "logged at every prune pass while stale".

## 7. docs-39: the disk bound misses the single-record case that `AppendBatch` admits (CONFIRMED)

**Anchors.** scope.md:239 and :347 (guarantee 11), userguide.md:151-152 and decision 69 bound
segment bytes by `MaxBytes + SegmentBytes + MaxBufferBytes + MaxBatchBytes` "because one flush
writes at most the buffer plus one batch". append.go:31 applies `MaxBatchBytes` only when
`len(payloads) > 1`; a single record is bounded by `MaxPayloadBytes` (append.go:25-26), default
and ceiling 64 MiB − 32 (log.go:80, codec.go:29-33) against `DefaultMaxBatchBytes` 16 MiB
(log.go:77); the encoded size is `len + 3*maxVarint + 4 = len + 34` (append.go:29). The inline
flush (append.go:77-78) writes and fsyncs the whole buffer before `publishLocked` rotates and
the caps run (flush.go:38, :50, :104-109, :138-141).

**Skeptic (CONFIRMED).** The decisions are internally inconsistent: decision 46 says "An empty
buffer accepts any single record up to MaxPayloadBytes" and decision 26 sets that at 64 MiB,
which the code honours, while decision 69's arithmetic assumes `MaxBatchBytes`; 26/46 are the
authoritative pair. With defaults the understatement is about 48 MiB. The existing test
(retention_test.go:288-340) sums files only after each call returns and never appends a single
record over `MaxBatchBytes`. The code alternative (validate `MaxPayloadBytes <= MaxBatchBytes`)
would contradict decision 46 with default values. Inferred from the code path; not executed.

**Disposition.** Owner decision, because the text of guarantee 11 changes. Recommended text at
scope.md:239, :347, userguide.md:152 and decision 69: "`MaxBytes + SegmentBytes + MaxBufferBytes
+ max(MaxBatchBytes, MaxPayloadBytes + 34)`"; at scope.md:119 and append.go:17-18: "a batch of
more than one record whose encoded size is over MaxBatchBytes is refused; a single record is
bounded by MaxPayloadBytes alone".

## 8. docs-41: format.md attaches the quill.policy exception to the write path (CONFIRMED)

**Anchors.** docs/format.md:103-104: "Every state file except `quill.policy` is written by temp
file, fsync, rename, and refused when malformed". `writePolicy` (log.go:525-533) ends in
`writeState`, a one-line wrapper over `atomicfile.Write` (state.go:281-283;
internal/atomicfile/atomicfile.go:30-57), the same path as `quill.tail`; the exception that
holds is tolerant parsing (`decodePolicy`, state.go:234-249).

**Skeptic (CONFIRMED).** The exception binds to the subject and scopes both predicates, and
lines 106-107 ("never a torn one") inherit it; decisions 29 and 48 make the exception a parsing
one only; no other sentence in the policy section (:159-163) corrects the reader. Cosmetic, as
the finder said.

**Correct text (apply now).** "Every state file is written by temp file, fsync, rename. Every
one except `quill.policy` is refused when malformed: unknown fields, a missing field, ..." (the
rest unchanged).

## 9. docs-42: the durability table omits the directory fsync after the marker at rotation (CONFIRMED)

**Anchors.** docs/format.md:193 "| Rotation | The data, the new empty segment's creation
(directory fsync), then `quill.tail` |". `rotateLocked` fsyncs after creating the segment
(flush.go:157) and sets `justSealed` (:172); `publishLocked` writes the marker (:115) and,
because `sealed` is true, fsyncs the directory again (:120-127). format.md:94 and decision 65
already say so; decision 89 records the phase-4 fix that added it.

**Skeptic (CONFIRMED).** The row reads like the pre-fix behaviour decision 89 describes; the
Close row (format.md:194) does enumerate its terminal "then the directory", so a reader would
infer rotation leaves the marker undurable. Cosmetic; recovery tolerates a stale marker.

**Correct text (apply now).** "| Rotation | The data, the new empty segment's creation
(directory fsync), then `quill.tail`, then the directory again (decision 65), so the marker
naming the new segment is durable |".

## 10. docs-43: scope says `Stat().Records` is Head − first + 1; the code also subtracts holes (CONFIRMED)

**Anchors.** scope.md:247 "`Stat().Records` is `Head − first retained sequence + 1`, exact
because sequences are contiguous (decision 10)". stat.go:16 documents "minus the records inside
holes"; stat.go:30-36 subtracts every hole within `[first, head]`.

**Skeptic (CONFIRMED).** Decision 32 (decisions.md:40) says "minus the records inside holes" and
decision 71 relies on the same reading; the scenario checks by hand (First 1, Head 12, hole
{5,8}: 8, not 12). Cosmetic: equal when there are no holes.

**Correct text (apply now).** "`Stat().Records` is `Head − first retained sequence + 1` minus the
records inside holes, exact because sequences are contiguous (decisions 10, 32)."

## 11. docs-44: the user guide says the `FlushBytes` flush is a background flush (CONFIRMED)

**Anchors.** docs/userguide.md:40 "`Append`, background flush every `FlushInterval` or
`FlushBytes`". append.go:77-78 flushes inside `appendLocked` on the appending goroutine
(`flushLocked` writes and fsyncs, flush.go:25-56); the flusher goroutine (flush.go:253-284)
wakes only on the `FlushInterval` ticker, a Sync request or the prune ticker. The guide's own
Config row (:134 "Inline flush size") and log.go:26 say inline.

**Skeptic (CONFIRMED).** Decision 69 (decisions.md:77): "in normal operation an append that
reaches `FlushBytes` flushes inline". A caller sizing `FlushBytes` would expect the crossing
append to return before the fsync; it pays it.

**Correct text (apply now).** "Buffered (the default): `Append`; an inline flush at
`FlushBytes`, a background flush every `FlushInterval`".

## 12. docs-46: the generated reference and `help ls` call `-dir` "the log directory" though `ls` takes the parent (CONFIRMED)

**Anchors.** cmd/quill/verbs.go:153, :168, :192, :254 register `-dir` with one usage string for
every verb; `cmdLs` (:309-321) reads the directory as a parent and opens each child.
docs/reference.md:5-6 states the exception in prose while the generated ls table (:36) says
"the log directory"; cli_test.go:53-69 pins the file to the generator.

**Skeptic (CONFIRMED).** Ran `quill help ls`: the summary line ("under a parent directory")
and the flag line contradict each other on one screen; `quill reference | diff -
docs/reference.md` is empty, so the fix must go through the generator. Decision 86 fixes only
the command shape, not the usage string.

**Fix (apply now).** Add a `dirUsage` field to `verb` (default "the log directory", "the parent
directory holding the logs" for `ls`), use it at :168, :192, :254 (or in the `flagSet` helper of
simplify-7), then `go run ./cmd/quill reference > docs/reference.md`. Test: cli_test.go:53-69
(the reference pin) and :88-91.

## 13. docs-40: the introduction shows `quill.Open(dir)`; the signature is `Open(Config)` (CONFIRMED)

**Anchors.** docs/introduction.md:13 "`quill.Open(dir)` returns a `Log`". log.go:280 `func
Open(cfg Config) (*Log, error)`; `Dir` is a Config field (log.go:19) and `validate` rejects an
empty Dir (log.go:141-142). scope.md:22 and :56, userguide.md:8 and :25 use `Open(Config)`.

**Skeptic (CONFIRMED).** Decision 4 uses the same shorthand "`Open(dir)`" but is the
directory-ownership decision, not an API-shape one; decisions 29 and 77 presuppose a Config
argument. Cosmetic: the inline code would not compile if typed.

**Correct text (apply now).** "`quill.Open(quill.Config{Dir: dir})` returns a `*Log`".

## 14. docs-45: the README layout table omits hack/powertest (CONFIRMED)

**Anchors.** README.md:71 lists `hack/gate.sh`, `hack/fuzz.sh`; `find hack -type f` also returns
hack/powertest/main.go; README.md:35 advertises "the real power-loss runs"; decision 92
(decisions.md:100) and docs/audits/reference/power-loss.md:6 name `hack/powertest`; `grep powertest
README.md` is empty.

**Skeptic (CONFIRMED).** The table is otherwise exhaustive over top-level entries, so the
omission is a gap, not a summary. Cosmetic.

**Correct text (apply now).** Add after README.md:71: "| `hack/powertest/` | The real
power-loss harness behind docs/audits/reference/power-loss.md (decision 92); run by hand, not in the gate |".
