# A1 adversarial audit: docs

Date 2026-10-29, against commit 1f4152d. Dimension D (documentation drift) of the A1 audit:
every candidate, the skeptic's verdict and reason, and the text the synthesis recommends. The
rule applied throughout: decisions.md is authoritative; where a prose document and the code
disagree, the one that contradicts the decision is the one that is wrong. Anchors re-read at
synthesis time. Ranked by what a reader would get wrong.

Verdicts: 14 CONFIRMED.

| # | id | anchor | severity | verdict | recommendation |
|---|----|--------|----------|---------|----------------|
| 1 | docs-33 | docs/format.md:161 | BUG (semantic) | CONFIRMED | owner decision, with correctness-32 |
| 2 | docs-34 | errors.go:10 | DOC | CONFIRMED | apply now |
| 3 | docs-36 | doc.go:22 | DOC | CONFIRMED | apply now |
| 4 | docs-35 | docs/userguide.md:240 | DOC | CONFIRMED | apply now |
| 5 | docs-37 | README.md:45 | DOC | CONFIRMED | apply now via simplify-2 (code), not the doc |
| 6 | docs-38 | docs/design/scope.md:274 | DOC | CONFIRMED | apply now |
| 7 | docs-39 | docs/design/scope.md:347 | DOC (a guarantee) | CONFIRMED | owner decision |
| 8 | docs-41 | docs/format.md:103 | DOC | CONFIRMED | apply now |
| 9 | docs-42 | docs/format.md:193 | DOC | CONFIRMED | apply now |
| 10 | docs-43 | docs/design/scope.md:247 | DOC | CONFIRMED | apply now |
| 11 | docs-44 | docs/userguide.md:40 | DOC | CONFIRMED | apply now |
| 12 | docs-46 | cmd/quill/verbs.go:254 | DOC | CONFIRMED | apply now |
| 13 | docs-40 | docs/introduction.md:13 | DOC | CONFIRMED | apply now |
| 14 | docs-45 | README.md:71 | DOC | CONFIRMED | apply now |

## 1. docs-33: offline writable verbs overwrite quill.policy with their own Config (BUG, semantic)

**Anchors.** docs/format.md:161-163 ("the settings the writer last opened with ... offline tools
read it"), README.md:18, docs/userguide.md:174 and :199, decision 29. Code: `Open` calls
`l.writePolicy()` unconditionally at log.go:305; cmdAppend (cmd/quill/verbs.go:474), cmdRoll
(:728) and cmdRecover (:756) open with `quill.Config{Retention: -1}` through `c.writer()`
(:287-298), which adds only `Dir` and `NoMaintenance`.

**Skeptic (CONFIRMED).** Reproduced: a writer with Retention 1h, MaxBytes 64 MiB, MaxRecords 1000,
ProtectedConsumers [billing]; after `quill roll` the file reads retention -1ns, max_bytes 0,
max_records 0, protected_consumers []. Decision 29 says the file exists because a tool "cannot
know its limits otherwise", decision 68 says offline tools "never prune" and `prune` needs a
parsed policy, and decision 87 says `ReadPolicy` exists "so `quill prune` and `quill ls` act on
the writer's settings"; after one offline verb a flagless `prune` acts on Retention -1 with no
caps and no protected consumers, and `ls`/`check` print UNBOUNDED. The narrow reading that the
offline tool is itself "the writer that last opened" is what the code implements and contradicts
those three decisions. (The finding's MaxBytes 4 MiB example is not a valid config, since
MaxBytes must be at least twice SegmentBytes; the claim is unaffected.)

**Why owner decision.** The fix is a one-condition code change in `Open`, not a doc edit, and it
interacts with correctness-32 (the same verbs also substitute the default 8 MiB SegmentBytes
and rotate at load). The options and the synthesis's recommendation are in audit-findings.md
under correctness-32/docs-33. If the behaviour is kept, docs/format.md:161, README.md:18,
docs/userguide.md:174/199, scope.md:136/267 and decision 29 must say that any writable open,
`quill append`/`roll`/`recover` included, rewrites `quill.policy` with its own Config, and that
`prune` must be given explicit flags after one.

## 2. docs-34: ErrCorrupt's comment says a sealed segment that does not verify makes Open refuse (DOC)

**Anchor.** errors.go:10-13.

**Skeptic (CONFIRMED).** The closing sentence "Open refuses to write" covers all three listed
cases, but load.go:439-454 stores a sealed segment's verify failure in `s.bad`, load.go:356-361
records it in `l.loadErr` and continues, and `Open` returns nil; the error surfaces only through
`Log.Err()` (log.go:494-495), matching `ErrCorrupt` via `SegmentError.Is` (load.go:55). The
`ErrCorrupt` returns from `load` are: malformed `quill.format`/`quill.tail` (load.go:278, :287),
no segments or a marker beyond the newest (:294, :298), the newest segment failing
`recoverSegment` (:326-333), overlaps (:367, :384), head below `tl.Seq` (:404). Decision 39 says
"if it is the newest segment Open fails with ErrCorrupt"; recovery_test.go:92-107 asserts Open
succeeds with `Err()` matching `ErrCorrupt` after damaging segment 0. The first clause of the
comment is true (the error does "report" `ErrCorrupt`, through `Err()`); the "Open refuses"
sentence is the wrong part. doc.go:22-23 has the same drift (docs-36).

**Text.** "ErrCorrupt reports damage below the durable marker. Open returns it for the newest
segment (bytes below quill.tail.off that do not verify), for a marker above the last valid
frame, for an overlap between segments, and for a damaged quill.tail or quill.format. A sealed
segment that does not verify does not fail Open: it becomes a hole, reads return a *GapError
over it, and Err() returns the *SegmentError, which matches ErrCorrupt. The user guide has the
procedure."

## 3. docs-36: package comment says damage below the durable marker fails Open (DOC)

**Anchor.** doc.go:22-23.

**Skeptic (CONFIRMED).** Every byte of a sealed segment is below the durable marker, and the
package's own definition of the phrase (errors.go:10-12, scope.md:75) includes sealed segments,
so the sentence tells a reader a damaged sealed segment fails Open. Code, decision 39 and
recovery_test.go:93-101 say otherwise (see docs-34). The sentence makes a false claim about an
error path a caller would build a startup health check on.

**Text.** "Corruption is never served silently: a damaged or missing sealed segment becomes a
hole that reads report as a GapError and Err() names; damage in the newest segment below the
durable marker, or a marker whose sequence is above the last valid frame, fails Open with
ErrCorrupt."

## 4. docs-35: user guide says damage in the newest segment's sealed predecessor fails Open (DOC)

**Anchor.** docs/userguide.md:240-242.

**Skeptic (CONFIRMED).** load.go:300-313 verifies every sealed segment through `loadSealed`
with no special case for the predecessor; the contiguity loop at :358-375 treats any `s.bad`
alike. Reproduced with segments 0, 1, 2 and a flipped byte in segment 1 (the predecessor): Open
succeeds, `Err()` names `/log/0000000001.log`, `holes = [{6 10}]`, `Read(6, 10)` returns the
gap. The finding's line anchors (332-347, 356-365) are ~25 lines off and recovery_test.go:225 is
a stale-marker test rather than a predecessor test, so the reproduction is the direct evidence.
Step 3 of the same section (userguide.md:249-251) already presupposes the hole behaviour, so the
paragraph contradicts itself.

**Text.** Replace lines 240-244 with: "`Open` fails with `ErrCorrupt` when the newest segment is
damaged below `quill.tail.off`, when the marker names a sequence above the last valid frame,
when two segments overlap, or when `quill.tail` or `quill.format` is malformed. Damage in any
sealed segment, the newest's predecessor included, does not fail `Open`: the segment becomes a
hole, `Err()` names it, `quill check` reports it, and reads stop before it with a gap."

## 5. docs-37: README claims no file is over about 600 lines; cmd/quill/verbs.go is 786 (DOC)

**Anchor.** README.md:45. Measured: verbs.go 786, retention_test.go 615, internal/fsx/crash.go 596.

**Skeptic (CONFIRMED).** Decision 20 (decisions.md:28), build-prompt.md:41 (whose layout tree
includes `cmd/quill/`), CONTRIBUTING.md:19 and decision 73 all state the ceiling repo-wide; the
README sentence is the faithful restatement and verbs.go is what is in breach, 31% over, which
no reading of "about" covers (615 and 596 arguably do). hack/gate.sh has no line-count step.
The doc-only alternative ("No library file...") would contradict decision 20 and must not be taken.

**Resolution.** Code: the split in simplify-2. No doc change; README.md:72 gains the new file names.

## 6. docs-38: scope's ConsumerTTL paragraph describes the in-memory timer decision 45 rejected (DOC)

**Anchor.** docs/design/scope.md:273-277.

**Skeptic (CONFIRMED).** Decision 45 (decisions.md:53) says "with no in-memory state ... both
its offset file's mtime and the `At` of the record at its offset are older than the TTL" and
names the in-memory "first saw it behind" timestamp as the rejected alternative (writer restarts
reset it); decision 82 adds the first-retained fallback. retention.go:47-61 implements exactly
that and `consumerView` is rebuilt every pass (:28-31); the only persistent field is
`stale []string` (log.go:231), names for `Stat`. Also :277 "logged once per pass change":
retention.go:92-95 warns for every stale name on every pass. The Config row (scope.md:125) and
userguide.md:107-108 are compatible; only 273-277 drift. Semantic for a reader sizing the TTL.

**Text.** Replace 273-276 with: "a protected consumer is stale when it is behind
(`offset ≤ Head`) and both its offset file's modification time and the `At` of the record at
its offset (or of the first retained record when a cap already deleted it, decision 82) are older
than the TTL; the prune pass reads that one frame and keeps no state between passes (decision
45)." At :277: "logged at every prune pass while stale".

## 7. docs-39: the disk bound omits the single-record case that AppendBatch admits (DOC, a guarantee)

**Anchors.** scope.md:239 and :347 (guarantee 11), userguide.md:151-152, decision 69; append.go:31
`if len(payloads) > 1 && size > l.cfg.MaxBatchBytes`; MaxPayloadBytes default 64 MiB − 32
(log.go:80, codec.go:33) vs MaxBatchBytes 16 MiB (log.go:77).

**Skeptic (CONFIRMED).** A single payload is checked only against MaxPayloadBytes (append.go:25-26);
`validate` imposes no relation between the two; the encoded size is len + 34 (append.go:29,
codec.go:36). `appendLocked` flushes inline (append.go:77-78) and `flushLocked` writes and fsyncs
before `publishLocked` rotates and only then runs the caps (flush.go:38, :50, :104-109, :138-141),
so the fsynced peak can exceed the stated bound by MaxPayloadBytes + 34 − MaxBatchBytes ≈ 48 MiB
with defaults. The decisions are internally inconsistent: 46 ("an empty buffer accepts any single
record up to MaxPayloadBytes") and 26 (64 MiB) match the code, while 69's arithmetic ("one flush
writes at most the buffer plus one batch") is false for a one-record batch. retention_test.go:288-340
sums files only after each call returns and never appends one record above MaxBatchBytes.
Inferred from the code path, not executed.

**Why owner decision.** The fix either restates guarantee 11 (doc) or changes validation or
admission (code); both touch a stated guarantee. Recommended text, per decisions 26/46 over 69's
arithmetic: "`MaxBytes + SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)`"
at scope.md:239, :347, userguide.md:152 and decision 69, and at scope.md:119 / append.go:17-18:
"a batch of more than one record whose encoded size is over MaxBatchBytes is refused; a single
record is bounded by MaxPayloadBytes alone".

## 8. docs-41: format.md says every state file except quill.policy is written atomically (DOC)

**Anchor.** docs/format.md:103-104.

**Skeptic (CONFIRMED).** `writePolicy` (log.go:525-533) ends in `writeState`, a one-line
wrapper over `atomicfile.Write` (state.go:281-283; atomicfile.go:30-57), the same path as
`quill.tail`, `quill.format` and consumer files; the only exception that holds is tolerant
parsing (`decodePolicy`, state.go:234-249), which decisions 29 and 48 describe. The exception
in the sentence binds to the subject and so scopes both predicates, and lines 106-107 inherit it.

**Text.** "Every state file is written by temp file, fsync, rename. Every one except
`quill.policy` is refused when malformed: unknown fields, a missing field, ..." (rest unchanged).

## 9. docs-42: the durability table omits the directory fsync after the rotation marker (DOC)

**Anchor.** docs/format.md:193.

**Skeptic (CONFIRMED).** `rotateLocked` fsyncs the directory after creating the segment
(flush.go:157); `publishLocked` writes the marker (:114) and, because `sealed` is true, fsyncs the
directory again (:120-127, comment citing decision 65). Decision 65 says so; decision 89 records
that the phase-2 build lacked exactly this fsync, so the row reads like the pre-fix behaviour.
format.md:94 already states the stronger guarantee, and the Close row (:194) does enumerate its
terminal "then the directory", so a reader of the table would infer rotation leaves the marker
undurable.

**Text.** "| Rotation | The data, the new empty segment's creation (directory fsync), then
`quill.tail`, then the directory again (decision 65), so the marker naming the new segment is
durable |"

## 10. docs-43: scope says Stat().Records is Head − first retained + 1; the code subtracts holes (DOC)

**Anchor.** docs/design/scope.md:247.

**Skeptic (CONFIRMED).** stat.go:16 and :30-36 subtract every hole within [first, head];
decision 32 (decisions.md:40) says "minus the records inside holes" and decision 71 relies on the
same reading. Hand check: First 1, Head 12, hole {5,8}: scope 12, code 8. Equal when there are no holes.

**Text.** "`Stat().Records` is `Head − first retained sequence + 1` minus the records inside
holes, exact because sequences are contiguous (decisions 10, 32)."

## 11. docs-44: the durability table says the FlushBytes flush is a background flush (DOC)

**Anchor.** docs/userguide.md:40.

**Skeptic (CONFIRMED).** append.go:77-78 flushes inside `appendLocked` on the appending
goroutine, and `flushLocked` (flush.go:25-56) writes and fsyncs (:50) before returning; the
flusher goroutine (flush.go:253-284) wakes only on the FlushInterval ticker, a Sync request or
the prune ticker. Decision 69 says "an append that reaches `FlushBytes` flushes inline";
userguide.md:134 and log.go:26 agree with the code.

**Text.** "Buffered (the default): `Append`; an inline flush at `FlushBytes`, a background flush
every `FlushInterval`".

## 12. docs-46: the generated reference says -dir for ls is "the log directory"; ls takes the parent (DOC)

**Anchors.** cmd/quill/verbs.go:153, :168, :192, :254 register `-dir` with one usage string;
cmdLs (:309-321) does `os.ReadDir(c.dir)` and opens each child; docs/reference.md:36 (generated)
and reference.md:5-6 (prose exception).

**Skeptic (CONFIRMED).** `quill help ls` prints "list the logs under a parent directory" and
then "-dir string / the log directory" on the same screen. `go run ./cmd/quill reference | diff - docs/reference.md`
is empty and cli_test.go:60-68 pins it, so the fix goes in the generator. No decision fixes the
usage string (86 fixes the shape only).

**Fix.** A per-verb usage field on `verb` (default "the log directory"; ls: "the parent directory
holding the logs") used at :168, :192, :254, then `go run ./cmd/quill reference > docs/reference.md`.
Folds naturally into simplify-7's `flagSet` helper.

## 13. docs-40: introduction shows quill.Open(dir); the signature is Open(Config) (DOC)

**Anchor.** docs/introduction.md:13; log.go:280 `func Open(cfg Config) (*Log, error)`, `Dir`
at log.go:19, `validate` rejects an empty Dir at :141-142.

**Skeptic (CONFIRMED).** scope.md:22 and :56, userguide.md:8 and :25 use `Open(Config{Dir: dir})`;
decisions 29 and 77 presuppose a Config argument. Decision 4 uses the same shorthand "`Open(dir)`",
but that row is the directory-ownership decision, not an API-shape one. Typed verbatim, the
introduction's code does not compile.

**Text.** "`quill.Open(quill.Config{Dir: dir})` returns a `*Log`".

## 14. docs-45: README layout table omits hack/powertest (DOC)

**Anchor.** README.md:71; `find hack -type f` gives hack/fuzz.sh, hack/gate.sh, hack/powertest/main.go.

**Skeptic (CONFIRMED).** The table is otherwise exhaustive over top-level entries; decision 92
(decisions.md:100) and docs/audits/power-loss.md:6 name `hack/powertest` as release evidence;
README.md:35 advertises "the real power-loss runs" yet `grep powertest README.md` is empty.

**Text.** Add after :71: "| `hack/powertest/` | The real power-loss harness behind
docs/audits/power-loss.md (decision 92); run by hand, not in the gate |".
