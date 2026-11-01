# Stage 2 audit: decisions held for the owner

Date 2026-10-29, against HEAD 1da37fc; updated 2026-10-30 (pass A1h) with the owner's approvals and what became of each. The stage 2 audit (pass A1, round 2: adversarial audit in four
dimensions) applied every finding that is behaviour-preserving, or that fixes a clear defect with a regression
test, and held back every finding whose fix changes observable behaviour, an exported error class, an exit code,
a CLI grammar, an on-disk or output shape, or the wording of a guarantee. This file lists those held items, one
section each, with the claim, why it was held (what the fix would change), a recommended answer and whatever fix
is already written. The full items (grounds, warrant, backing, rebuttal, skeptic votes and reproductions) are in
the per-dimension reports: [correctness](a1-round2-correctness.md), [docs](a1-round2-docs.md) and
[dedup](a1-round2-dedup.md); the counts are in [a1-round2-findings.md](a1-round2-findings.md).

The recommended answers are the recorder's reading of the auditor's proposed fix and the skeptics' notes; they were
recommendations when written. Ranked by severity, then reach.

**Update 2026-10-30.** The owner approved the recommended answer for 17 of the items below (every A1 item in the table
except A1-67 and A1-80, plus A4-1) on 2026-10-30. Each section keeps its recommendation text as first written (its
"Fix written" paragraph is the state on 2026-10-29) and now carries an **Owner approval** and a **Disposition**
paragraph, which are current. Result: 17 fixed, each in its own commit. Sixteen were FIXED-VERIFIED by a fix-skeptic (109 lens
verdicts, all pass; A1-82 is the 16th, fixed in bb608d3 on its third attempt, 7 of 7 lenses). The 17th, A4-1, is fixed in this
commit for its `build/quill` half only (the CI half was refuted because the pinned setup-go reads the toolchain line), after
rejections on the audit record alone (three attempts, each on the no-new-gap lens only); no verdict is claimed here for this commit. 0 fix-failed.
A1-67, A1-80, A1-62 and A1-59 are not in the ledger this update was written
from: commits for them exist in the tree (ba7c650, b5ad955, 2978378 and 7ef431c, verified from `git log`), but their
sections and table rows are unchanged here. The per-item argument, reproduction and fix-skeptic outcome are in
[a1-round2-findings.md](a1-round2-findings.md), section "Pass A1h".

**Update 2026-10-31.** The owner approved the recommended answer for A1-67 and A1-80 as well. Their fixes (ba7c650,
b5ad955) were already on `main` without a recorded fix-skeptic outcome; an independent fix-skeptic per commit returned
FIXED-VERIFIED (7 of 7 lenses each), and their sections below now carry an **Owner approval** and a **Disposition**.
A1-62 and A1-59 remain unrecorded here.

**Update 2026-10-31 (pass A5h).** The owner approved the recommended answer for the eleven held items of pass A5 (A5-10, A5-1, A5-36, A5-18, A5-22, A5-29, A5-3, A5-9, A5-23, A5-32 and A5-44; see section "Pass A5" below) on 2026-10-31. Ten are fixed, each in its own commit in the tree, and each was FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses each, 60 lens verdicts, all pass; reported by the skeptics). The eleventh, A5-1, was rejected once for a documentation defect and then fixed on a third attempt (99b140d, FIXED-VERIFIED for the approved scope; see its section). Each A5 section keeps its recommendation text as first written and now carries an **Owner approval** and a **Disposition** paragraph.

| id | sev | one line | recommended | status 2026-10-30 |
|----|-----|----------|-------------|----|
| A1-67 | high | the documented `quill commit` rewind of an ahead consumer fails with ErrAhead | fix the doc now; approve letting `commit` rewind | approved 2026-10-31; fixed ba7c650 (FIXED-VERIFIED, 7 of 7) |
| A1-80 | high | a live protected consumer stops the writer from restarting | approve | approved 2026-10-31; fixed b5ad955 (FIXED-VERIFIED, 7 of 7) |
| A1-25 | medium | after a failed rotation the active segment can outgrow the 1.5 GiB limit | approve a ceiling | approved; fixed c18a27a (FIXED-VERIFIED, 7 of 7 lenses) |
| A1-32 | medium | the reserved `_archive` consumer cannot be removed or rewound | approve | approved; fixed a3955cd (FIXED-VERIFIED, 7 of 7) |
| A1-47 | medium | extra positional arguments are silently ignored | approve (exit 2) | approved; fixed 2c4e4f4 (FIXED-VERIFIED, 7 of 7) |
| A1-48 | medium | `-from`, `-limit`, `-n` parse base 0 (`010` is octal) | approve | approved; fixed b4fe935 (FIXED-VERIFIED, 7 of 7) |
| A1-82 | medium | a read I/O error becomes ErrCorrupt or a skippable gap | approve | approved; fixed bb608d3 (FIXED-VERIFIED, 7 of 7) |
| A1-85 | medium | Reader-created consumer files ignore the log's Perm | approve | approved; fixed 279d7ce (FIXED-VERIFIED, 7 of 7) |
| A1-13 | medium | Archive returns ErrCorrupt, not ErrFutureVersion, for a newer destination | approve | approved; fixed b529db5 (FIXED-VERIFIED, 7 of 7) |
| A1-69 | medium | MaxBatchBytes is checked against a 34-byte-per-record estimate | document only | approved; fixed 5b30e7f, documentation only (FIXED-VERIFIED, 6 of 6) |
| A1-2 | low | a case-variant duplicate key in quill.tail wins silently | fix the duplicate check; document the rest | approved; fixed 89957ae (FIXED-VERIFIED, 7 of 7) |
| A1-29 | low | Open refuses a symlinked Dir that OpenReader accepts | follow the symlink | approved; fixed 8c0c531 (FIXED-VERIFIED, 7 of 7) |
| A1-35 | low | one future-stamped segment pins age expiry of every newer one | warn and document | approved; fixed 19c0996 (FIXED-VERIFIED, 7 of 7) |
| A1-50 | low | `-json` passes C1 control characters raw | approve | approved; fixed fd52463 (FIXED-VERIFIED, 7 of 7) |
| A1-52 | low | Read can exceed MaxReadBytes by one record | reword the docs | approved; fixed 25e96c2, documentation only (FIXED-VERIFIED, 6 of 6) |
| A1-57 | low | `stat -json` holes use capitalised keys and `null` | approve before release | approved; fixed c91329c (FIXED-VERIFIED, 7 of 7) |
| A1-87 | low | the clock guard rounds up where the text says "at most 25%" | reword the text | approved; fixed 50201e1, documentation only (FIXED-VERIFIED, 6 of 6) |
| A1-91 | low | a quill.policy of `{}` is acted on as retention 0 | approve, low priority | approved; fixed 5c603fc (FIXED-VERIFIED, 7 of 7) |

## A1-67: the documented rewind of an ahead consumer, `quill commit`, cannot work (high)

**Claim.** A consumer whose offset is above Head+1 (after a restore that rolled the log back) fails to open with
ErrAhead, and `quill commit` opens the consumer before committing, so the remedy named in docs/userguide.md:98 and
:222-223, scope.md:325 and decision 53 fails with the same error. `quill check` prints HEALTHY and `quill consumers`
shows the consumer with lag 0 and no note.

**Why held.** The only fix that makes the documented remedy true changes behaviour: `commit` (and the library path it
uses) would have to open an ahead consumer, which decisions 36 and 53 currently refuse at open. The doc-only fix is not
held in principle, but it names a sequence (unregister, register, commit) that the owner should agree is the supported
one, because `unregister` drops the protect flag.

**Recommended answer.** Both. Now: correct the user guide to name the working sequence (`quill unregister`, `quill
register [-protect]`, `quill commit -offset N`) and say that unregister drops the protect flag. Approve: let `quill
commit` (an option or an internal open path used only by commit) open an ahead consumer and accept a rewind to any
offset in [first, Head+1]; and have `quill check` or `quill consumers` flag a consumer ahead of the head.

**Fix written.** None. The proposed fix is the one above; the reproduction (a 5-record log, a consumer committed at 8,
the log restored to head 5, `commit -offset 2` failing) is in the docs report.

**Owner approval.** Approved on 2026-10-31, as recommended.

**Disposition.** Fixed in ba7c650 (on main since 2026-10-30; verified from `git log`); fix-skeptic FIXED-VERIFIED
(7 of 7 lenses pass; reported, from a skeptic that ran everything on a scratch copy of the tree). A new
`ConsumerOptions.Rewind` opens a consumer ahead of the head; `quill commit` is the only caller that sets it, and
`Commit` still accepts only offsets in [first, Head+1]. `register`, the default open, `Reader.Consumer`, `Open` with
`ProtectedConsumers` and the archiver still return ErrAhead. `quill consumers` notes the consumer ("ahead of the head:
rewind it with quill commit") and `-json` adds `"ahead": true` only when true; `quill check` warns and stays
HEALTHY. Reproduction after (verified by the skeptic, built binaries): a 5-record log, a consumer committed at 8, the
log restored to head 5; `commit -offset 2` prints `billing 2`, the protect flag is kept, and `register` still refuses.
The same works for `_archive`. The recommended documentation step (name the unregister, register, commit sequence) is
superseded: with the code fix the guide names `quill commit -offset N` directly. Decisions 36 and 53, scope.md, the user
guide and CHANGELOG are amended. Known and unchanged: a consumer ahead of the head protects no segments, so a writer
running with caps can delete the rewind target before `commit` runs (the commit then fails cleanly with
ErrOffsetRange); and once the head grows past a stale ahead offset the consumer silently resumes mid-log and `check`
stops flagging it. Test coverage is thin on the `-json` key, first-1 and Head+2 on an ahead consumer and `commit -time`;
the skeptic checked those by hand. The user guide omits `-dir` and `-name` from the example command (cosmetic).

## A1-80: a live protected consumer stops the writer from restarting (high)

**Claim.** `Config.ProtectedConsumers` makes Open, and `quill prune` (which feeds the policy back in), fail with
ErrLocked whenever a live consumer process holds that consumer's lock; the writer cannot be redeployed while its
protected consumer runs. `quill prune` also prints "a writer holds the log", naming the wrong lock. Reproduced on macOS
and, by the auditor, Linux ext4: a reopen takes 202 to 233 ms and fails.

**Why held.** The fix changes what Open does with `ProtectedConsumers` (decision 67: protect a lazy consumer before its
process starts) and how it interacts with the per-consumer lock (guarantee 5): Open would stop needing exclusive
ownership of an existing consumer file. It also changes the `prune` verb's Config.

**Recommended answer.** Approve. In Open, create the consumer file when missing without taking exclusive ownership of
an existing one; where the file exists and is parseable, accept it and set the protect flag only when it is absent or
off (under a short retry if the lock is held); make `quill prune` drop ProtectedConsumers from the policy-derived Config
(decision 99 already says offline verbs open without protected consumers); report ErrLocked with the right lock name.
The workaround today is to drop ProtectedConsumers from the Config after the first run.

**Fix written.** None. A regression test is proposed: reopen with a held consumer lock.

**Owner approval.** Approved on 2026-10-31, as recommended.

**Disposition.** Fixed in b5ad955 (on main since 2026-10-30; verified from `git log`); fix-skeptic FIXED-VERIFIED
(7 of 7 lenses pass; reported, from a skeptic that ran everything on a scratch copy of the tree). `Open` accepts an
existing consumer file without its lock when the file parses, is this log's, is already protected and is not ahead of
the head; every other state goes through the old locked open with the old errors (ErrCorrupt, ErrWrongLog, ErrAhead).
`openWriter` drops `ProtectedConsumers`, so `quill prune` and the other offline write verbs no longer fail beside a
running consumer and no longer name the wrong lock; protection read from the consumer files on disk is still honoured
(an offline `prune -retention 1s -force` over 8 segments deleted nothing with the consumer protected). Reproduction
before and after (verified by the skeptic): a writer reopen beside a live consumer process took 223 ms and failed
with ErrLocked before; after it takes 16 ms on macOS and 3.3 ms on Linux with no error. A stress run under `-race`
(1402 commits against 410 opens, 15 s) showed no torn read and the flag never cleared. Decisions 67 and 99, scope.md,
the user guide and CHANGELOG are amended. Known and as approved: a protected name whose file says unprotected, under
a live consumer, still fails `Open` with ErrLocked after the 200 ms retry; offline verbs no longer create a missing
protected consumer file (decision 99); the lock-free accept path has an inherent check-then-return window if a
consumer clears protection at that instant (inferred by the skeptic, not a regression).

## A1-25: after a failed rotation the active segment can outgrow the 1.5 GiB limit (medium)

**Claim.** While rotation keeps failing (decision 60(d) keeps appending), nothing bounds the active segment, so it can
pass the 1.5 GiB limit that Open and readers enforce; the log then cannot be opened although every record was
acknowledged durable. Reproduced on macOS: a read-only log directory, 400 appends of 4 MiB, a 1,677,729,073-byte segment,
and Open then refuses it. The same overshoot is reachable with no fault when SegmentBytes is 1 GiB and FlushBytes or
MaxBufferBytes exceed 512 MiB (read, not run).

**Why held.** The fix refuses appends in a state decision 60(d) deliberately keeps accepting them, so it changes the
contract of Append and amends the decision. It also adds a Config validation.

**Recommended answer.** Approve a ceiling. Once a pending rotation error exists, refuse new records with that error when
the active segment plus buffer would exceed SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes+34), and
always stay below `maxSegmentBytes`; validate that SegmentBytes plus the largest single-flush overshoot is under
`maxSegmentBytes`; make the SegmentError for an oversize but intact file say so instead of "damaged at byte 0 (intact to
byte 0)"; amend decision 60(d) and the user guide. Alternative: keep the behaviour and document that a supervisor must
watch `Err()`.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in c18a27a; fix-skeptic FIXED-VERIFIED (7 of 7 lenses pass). While rotation keeps failing, an
Append or AppendBatch that would push the active segment past SegmentBytes + max(FlushBytes, MaxBufferBytes) +
max(MaxBatchBytes, MaxPayloadBytes+34) is refused with the rotation error and takes no sequence number; Open refuses
(ErrConfig) a Config whose ceiling reaches the 1.5 GiB limit; a segment file over the limit is reported as "refused
without being read" by Open, Check and `quill check`, not as "damaged at byte 0". Decisions 16 and 60(d), the user
guide, format.md, scope.md and architecture.md are amended and CHANGELOG has a Fixed line. Reproduction before and
after (verified, macOS): a 1,677,729,073-byte segment and a refused Open before; 398 of 400 appends refused, an
8,388,644-byte segment and a successful reopen after. Not changed: docs/design/build-prompt.md keeps the historical
keep-appending wording (a build record); `quill check` still prints "damaged at byte 0" for a segment that is not a
regular file.

## A1-32: the reserved `_archive` consumer cannot be removed or rewound (medium)

**Claim.** The consumer `quill archive` creates is protected, and `register`, `unregister`, `commit` and the library's
`DeleteConsumer` all refuse its name ("a leading underscore is reserved"). It pins segments at or above its offset until the
next archive; after a restore that leaves it ahead of the head, `quill archive` fails with ErrAhead and the userguide's
remedy (`quill commit`) is refused for exactly this consumer. Deleting `consumers/_archive` by hand, or setting ConsumerTTL,
clears it; neither is documented for it.

**Why held.** It changes an exported function's behaviour and the CLI's accepted names, and decision 41 reserves the prefix
on purpose.

**Recommended answer.** Approve: let `DeleteConsumer`, `unregister`, `commit` and `register` accept exactly the reserved
archiver name (a second validator), or give `quill archive` a `-reset` option; document the remedy in the user guide.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in a3955cd; fix-skeptic FIXED-VERIFIED (7 of 7). One unexported validator accepts exactly
`_archive` on `Consumer` (Log and Reader), `DeleteConsumer`, and the CLI `register`, `unregister` and `commit`; every
other leading-underscore name is still refused, and so is `_archive` in `Config.ProtectedConsumers`. The built binary
rewinds an ahead `_archive` with `commit -name _archive -offset 1`, then `quill archive` succeeds. No `-reset` flag
was added. Decisions 16, 41 and 67, the user guide, format.md, scope.md, build-prompt.md and CHANGELOG are amended.
Consequence of the approved answer: `register -no-protect -name _archive` unprotects the archiver until the next
`quill archive`.

## A1-47: extra positional arguments are silently ignored (medium)

**Claim.** Every verb except `append` ignores positional arguments, and Go's flag parsing stops at the first one, so
`quill -dir L read 100` prints from the first retained record and `quill -dir L tail 5 -json` prints text, both exit 0.

**Why held.** It turns previously successful (if misdirected) invocations into exit 2.

**Recommended answer.** Approve. After `fs.Parse`, if the verb does not take arguments and `len(fs.Args()) > 0`, return a
usage error ("unexpected argument ...; flags go before arguments"). `append` and `help` (which takes a verb name) are
exempt.

**Fix written.** None; the proposed change is one check in cmd/quill/verbs.go.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 2c4e4f4; fix-skeptic FIXED-VERIFIED (7 of 7). A verb other than `append` and `help` given a
positional argument exits 2 with `unexpected argument "X"; flags go before arguments`, before anything is opened; the
exemption is one unexported flag on the verb table. Decision 53, the user guide and CHANGELOG are amended. Left as is,
inside the approved exemption: `help read x` ignores the second positional and `append x -sync=false` records the flag
as a line.

## A1-48: `-from`, `-limit` and `-n` parse numbers in base 0 (medium)

**Claim.** `-n 010` is 8 records, `-from 0x10` is 16 and `-limit 1_0` is 10, while `-offset`, `-max-bytes`,
`-min-free-bytes`, `-max-records` and `@unix` times are plain decimal (`010` is 10; `0x10` is refused).

**Why held.** A previously accepted spelling (hex, underscores, octal) changes meaning or is refused.

**Recommended answer.** Approve: register the three flags through a small `flag.Value` that calls `parseDecimal`, as the others
do. The silent octal reading of a zero-padded number is the dangerous part; hex is arguably a feature, but one rule across the
CLI is simpler to document.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in b4fe935; fix-skeptic FIXED-VERIFIED (7 of 7). The three flags take plain base-10 through a
small `decimalFlag` over `parseDecimal`: `010` is ten; `0x10`, `1_0`, `0b1`, `0o7`, signs and exponents exit 2 with
nothing on stdout. Decision 96, the user guide and CHANGELOG are amended. Cosmetic: `help` shows `value` where it
showed `uint`. The full gate could not complete `go mod tidy` in a git-archive copy (no repository there); every other
gate step passed.

## A1-82: a read I/O error is reported as ErrCorrupt or a skippable gap (medium)

**Claim.** In `scanSource`, any error from `walk`, frame errors and I/O errors alike, becomes ErrCorrupt (active segment, with
the cause flattened) or a `*GapError` over the whole segment (sealed). Decision 79 says an I/O error is returned as that error,
never as ErrCorrupt, and scope.md's resume idiom ("iterate again from To + 1") makes a consumer skip a whole segment after one
transient EIO.

**Why held.** It changes which error class the read paths return (ErrCorrupt and `*GapError` become the underlying I/O error),
which callers may match on.

**Recommended answer.** Approve: return non-frame errors unwrapped (wrapped with `%w` so `errors.Is(EIO)` works) for both
segment kinds; keep `*GapError` for real frame damage; add a Sim EIO-on-read test. The sealed-segment half is arguable under
decision 38, but the active-segment half contradicts decision 79 outright.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in bb608d3 (on main; verified from the commit log), the third attempt; fix-skeptic FIXED-VERIFIED
(7 of 7 lenses, all pass). The two earlier attempts were rejected: f6fcf26 and a5707a8
(the second attempt). a5707a8 had fixed `scanSource`, `ensureIndex`, `OffsetForTime` and the Reader's
active-segment probe, and was rejected for one path: the Reader's sealed first-frame probe still turned one EIO into a
`*GapError` over the whole segment (reproduced by that fix-skeptic against the changed tree: with EIO injected on reads
of sealed segment 2, `Reader.Read(21, 1)` returned `*quill.GapError: records 21 to 32 are unavailable`, and segment 1
gave "records 1 to 20"), and its test discarded the error. bb608d3 started from a5707a8 and closed both defects.

What the fix does (verified by reading the tree). An I/O error met while reading, in any segment and at any point,
including a sealed segment's first frame, is returned as itself (wrapped with `%w`, so `errors.Is(err, syscall.EIO)`
works), uncached, so the next call reads again. `Reader.segment` now returns `(*readerSegment, error)`: a non-frame
error from the first-frame probe is returned before the segment is cached (reader.go:246-247) and surfaces from
`view()`. Open failures (EMFILE, EACCES, not a regular file) stay a hole for one view (decision 94), and real frame
damage stays a `*GapError` or ErrCorrupt (`isFrameError`; `scanFailure`, read.go:276). `Open` is the second half:
`loadSealed` (load.go:449) now returns a read I/O error from `verifySegment` as a second result instead of recording
it in the segment's `bad` field, and `load()` fails `Open` with the first such error in segment order (load.go:321-330),
as it already did for the newest segment; damage and open failures stay holes (decision 98). Amended in place with the
note "Stage 2 audit, owner-approved 2026-10-30": decisions 38, 39, 79, 94 and 98; scope.md guarantee 14; user guide
section 3; the `GapError` doc comment (a run-on line wrapped); CHANGELOG Unreleased (Fixed). No exported symbol, flag
or file was added, and golden bytes are unchanged (verified by the fix-skeptic with `git diff --stat` over testdata).

Regression test (verified). `TestAReaderFirstFrameProbeIOErrorOfASealedSegmentIsReturnedAsThatError`
(read_ioerror_test.go:158) injects EIO on the reads of a sealed segment through the simulator, for segment 1 and for a
middle segment, and asserts through `wantReadEIO` (:48) that the read that hits the fault gives `errors.Is(EIO)` true,
`errors.As(*GapError)` false and `errors.Is(ErrCorrupt)` false, and that the same read succeeds once the fault clears.
The fix-skeptic reverted only errors.go, load.go, read.go, reader.go and seek.go to the parent versions and kept the
new tests: 9 tests failed, each for the claimed reason (the sealed-probe test with "records 1 to 20 are unavailable" for
the first segment and "records 21 to 32" for the middle one; the active-probe test with ErrCorrupt; the `Open` test with
"Open succeeded ... Read gave ... gap true"). The same tests pass on the fixed tree. The existing
`TestATransientScanErrorIsNotCachedByTheReader` was edited by design (reader_test.go:617): a scan-time open error is
now returned as itself, not as a gap.

Sites checked for a sibling that turns an I/O error into a gap or ErrCorrupt (verified by reading, by the fixer and again
by the fix-skeptic): reader.go `segment()` open path, `segment()` first-frame probe (fixed), `view()` active first-frame
probe (already raw), `view()` open of the active file (raw), `readerSegment.scan` (raw, not cached), the `ref.open`
closures in `view()` and `sealedRefsLocked` (ErrNotExist is a gap for retention, other errors raw); read.go `scanSource`
and `scanFailure`; seek.go `offsetForTime`; load.go `verifySegment`, `recoverSegment`, `loadSealed` (fixed) and the
newest-segment recovery in `load()`; log.go's Open creation path; iter.go (through `scanSource`); archive.go
`copySegment` and Archive's own `GapError`s (arithmetic on listed ranges, not I/O); stat.go `verifyBounded` (an I/O error
is reported by `Check` as a fatal segment problem carrying the error text, neither a gap nor ErrCorrupt); retention.go's
resume read (reads `g.To+1` after a real gap only). The two remaining gap-on-I/O cases are the ones the owner's answer
keeps: an open failure in `segment()` and ErrNotExist for retention.

Gate (verified by the fix-skeptic in an archive copy): gofmt clean, `go vet` and `go build` pass, `go test -race -count=1
./...` ok in every package, staticcheck v0.8.2 and deadcode v0.52.0 exit 0; the relevant tests also pass under `-race` on
the Linux x86-64 box. Not run by the fix-skeptic: the fuzz targets and `go mod tidy`. The fixer's own gate run is not in
the ledger, and this recorder re-ran nothing (docs only).

Residual objections, none against the claim (reported by the fix-skeptic; the first was reproduced): (1) Availability
trade-off. One EIO on a sealed segment's first frame now fails every call on that Reader that builds a view, not only the
read that needs the segment: a head read, `Reader.Stat` and `OffsetForTime` all fail while the fault lasts. A persistent
EIO on any sealed segment now makes `Open` fail where it used to record a hole, so retention cannot age that segment out
until the fault is fixed. This follows from decision 79 and from how the newest segment already behaved; the amended
decisions 94 and 98 and the CHANGELOG say that `Open` fails, but decision 94's "returned by the read that needs it"
understates the view-wide failure. (2) Decision 94 still carries the phrase "in an intermediate form, made a hole for one
view" (docs/design/decisions.md:102, verified by grep), which refers to an attempt that never shipped; it should be
deleted. This commit changes docs/audits only, so the phrase is left for the owner or the next fix. (3) A real EIO on an
`os.File` was not exercised, only the simulator's injection (inferred to behave the same).

## A1-85: Reader-created consumer files ignore the log's Perm (medium)

**Claim.** `Reader.consumerStore` hard-codes 0o600 and `Archive` hard-codes 0o600; `ReaderOptions` has no Perm. In the group
arrangement scope.md describes, a consumer process in another account creates a file the writer cannot read, and retention
then treats the consumer as broken (every segment protected, `check` UNHEALTHY). A skeptic ran the cross-user case on the Linux
box with `Config{Perm: 0660}` and confirmed it.

**Why held.** It changes the modes of files created through a Reader and the CLI, and may add a `ReaderOptions` field and a
flag.

**Recommended answer.** Approve: derive the mode of Reader-created files from the log (the permission bits of its existing
files, inferred as the least-surprise source), or add `ReaderOptions.Perm` and a CLI flag; use the same for Archive; document it.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 279d7ce; fix-skeptic FIXED-VERIFIED (7 of 7). `OpenReader` reads the permission bits of
`quill.tail` once (the writer sets them to Config.Perm exactly), ORs in 0o600, and the Reader's consumer files and
everything `Archive` creates use that mode; default-mode logs still get 0600. This is the first of the two options
recommended; no `ReaderOptions` field or flag was added. Decisions 41, 51 and 67, scope.md, format.md, the user guide,
the doc comments and CHANGELOG are amended. Not shown: a real second account (the cross-user case could not be run
because creating a user on the shared box was refused); the effect is shown by file modes and the passing tests. A
Reader-created lock file follows the umask (disclosed in the docs).

## A1-13: Archive returns ErrCorrupt, not ErrFutureVersion, for a newer destination format (medium)

**Claim.** The state-file load sequence is written six times (three for quill.format, three for quill.tail); Archive's copy wraps
every `decodeFormat` error as ErrCorrupt, so a destination whose quill.format is `quill/2` gives ErrCorrupt where Open and
OpenReader give ErrFutureVersion (decision 6, errors.go:16-17, format.md:137).

**Why held.** The fix changes the error class an exported function returns; the dedup into `loadFormat` and `loadTail` is net
zero lines.

**Recommended answer.** Approve; decision 6 already says ErrFutureVersion. If the owner prefers "any unusable destination
marker is ErrCorrupt" for Archive, keep archive.go's wrapping and merge only the load.go and reader.go pair.

**Fix written.** A prototype (the two helpers in state.go, six sites switched, +42/-42 lines) passed the full suite in a scratch
copy; it was not committed. A test that Archive into a `quill/2` destination matches ErrFutureVersion is proposed.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in b529db5; fix-skeptic FIXED-VERIFIED (7 of 7). `Archive` into a `quill/2` destination now
returns ErrFutureVersion; the three `quill.format` loads (load.go, reader.go, archive.go) go through one `loadFormat`
in state.go. The `quill.tail` half (`loadTail`) was not done. Decisions 6 and 85 and CHANGELOG are amended and the
Archive doc comment says so. Cosmetic: the ErrCorrupt text for a damaged destination marker no longer says
"destination".

## A1-69: MaxBatchBytes is checked against a 34-byte-per-record estimate (medium)

**Claim.** The docs call MaxBatchBytes the largest encoded batch, but append.go:30 checks `34 + len(payload)` per record, so 500,000
one-byte records (about 7.5 MB encoded, under half of 16 MiB) are refused, and `quill append` from stdin (one batch) refuses
everything. Reproduced on macOS and Linux.

**Why held.** Making the check exact, or chunking stdin, changes behaviour and (for stdin) the all-or-nothing property of one
batch.

**Recommended answer.** Document it now: say the limit is checked against `len(payload)+34` per record (an upper bound on the
encoded size) and that `quill append` from stdin is subject to it (about 480,000 one-byte lines per call at the default). Leave
the check conservative; the disk-bound formula depends on the over-count. Chunking stdin is optional and gives up atomicity.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 5b30e7f; fix-skeptic FIXED-VERIFIED (6 of 6). Documentation only, as recommended:
MaxBatchBytes is described as checked against `len(payload)+34` per record (an upper bound of the encoded size) and
`quill append` from stdin as one batch subject to it; the check is unchanged. The figure written is "about 479,000"
one-byte records at the default, not the "about 480,000" in the recommendation: the fix-skeptic measured 479,349
accepted and 479,350 refused (verified). Decision 46, scope.md, the user guide, the Config and AppendBatch comments,
the `append` verb summary and docs/reference.md (regenerated) and CHANGELOG are amended.

## A1-2: a case-variant duplicate key in quill.tail wins silently (low)

**Claim.** The slow path accepts a key such as `SEQ`, and `{"seq":1,"SEQ":2,...}` resolves to Seq 2 with no error, because
`checkDuplicateKeys` compares exact strings while `encoding/json` matches field names case-insensitively; it also accepts trailing
whitespace, reordered keys and a missing newline. format.md:103-105 says a duplicate key is refused.

**Why held.** The strictness of a state file is a guarantee-9 question, and decision 105 deliberately preserved the slow path's
verdicts; A1-1 (round-1-decided) would instead make the marker canonical-only.

**Recommended answer.** Make the duplicate check case-insensitive (the only part that contradicts "a duplicate key is refused") and
document in format.md which non-canonical shapes are tolerated (whitespace, order, a missing newline). Do not reopen decision 105's
canonical-only question unless taking A1-1.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 89957ae; fix-skeptic FIXED-VERIFIED (7 of 7). The duplicate-key check compares keys by the
case folding `encoding/json` uses, so `{"seq":1,"SEQ":2,...}` is refused; format.md lists the non-canonical shapes the
strict parser still accepts (whitespace, order, a missing newline). Decision 105's canonical-only question was not
reopened. Decisions 72 and 105 and CHANGELOG are amended.

## A1-29: Open refuses a symlinked Dir that OpenReader accepts (low)

**Claim.** `mkdirAll` uses Lstat, so `Open(Config{Dir: link})` returns ErrNoLog "is not a directory" for a symlink to a directory, while
`OpenReader(link)` and a symlinked parent both work. Nothing documents the writer-side restriction.

**Why held.** Either choice changes behaviour (follow the link, or keep refusing with a different error), and the refusal may be
deliberate hardening.

**Recommended answer.** Follow a symlinked final component of `Dir` (`os.Stat`), keeping O_NOFOLLOW for everything inside the log
directory, and make OpenReader and the CLI agree (a deployment that symlinks its data directory is common; the planted-link threat
needs write access to the parent, which SECURITY.md excludes, inferred). Otherwise keep the refusal, document it and say "Dir is a
symbolic link" in the error.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 8c0c531; fix-skeptic FIXED-VERIFIED (7 of 7), after an earlier attempt that missed a link in
an ancestor of a Dir that does not exist yet. `Open` follows a symbolic link anywhere in Dir (an unexported
`resolveDirLink` resolves it once with `filepath.EvalSymlinks`); a link to a file or to nothing stays ErrNoLog and the
message says "symbolic link"; files inside the log are still opened with O_NOFOLLOW. Decisions 51 and 62, scope.md,
the user guide, SECURITY.md and CHANGELOG are amended. A link swapped after open is not re-checked (inferred to be
inside the threat SECURITY.md excludes).

## A1-35: one future-stamped segment pins age expiry of every newer one (low)

**Claim.** The age sweep is a strict prefix rule (`if s.lastAt > cutoff { break }`), so a segment whose last record is stamped in the
future blocks age expiry of everything after it until the clock catches up; decision 44's guard covers only a clock that moves
forward. Nothing is logged. The caps still bound disk and `quill check` flags the record.

**Why held.** A fixed policy (judge such a segment by its seal time, or skip it) is a design choice; prefix deletion is deliberate.

**Recommended answer.** Minimum: log a warning when the sweep stops at a segment whose last stamp is more than a day in the future,
and document the hazard beside decision 44. Do not change the prefix rule.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 19c0996; fix-skeptic FIXED-VERIFIED (7 of 7). The prefix rule is unchanged; when the sweep
stops at a sealed segment whose newest record is stamped more than 24 h ahead, the prune pass logs a warning (segment
id, last_at, kept count). Decision 44, scope.md, the user guide and CHANGELOG are amended. The CLI sets no logger, so
`quill prune` shows no warning (as with the existing clock-guard warning); `quill check` still reports the record.

## A1-50: `-json` passes C1 control characters raw (low)

**Claim.** `appendJSONString` copies valid UTF-8 above 0x7f except U+2028 and U+2029, so U+0080 to U+009F (for example the CSI
introducer U+009B) reach the terminal raw in `-json`, contradicting userguide.md:198-199 and scope.md:188; the text form escapes them. DEL
(0x7f) also passes raw.

**Why held.** It changes the bytes `-json` emits (decision 35 pins them).

**Recommended answer.** Approve: escape U+0080 to U+009F (and 0x7f) as `\u00XX`; the JSON stays valid and lossless; update the test that
compares with `encoding/json`. Whether a given terminal honours 8-bit C1 in UTF-8 mode was not tested; the documentation promise is broken
either way.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in fd52463; fix-skeptic FIXED-VERIFIED (7 of 7). `-json` writes DEL and U+0080 to U+009F as
`\u00XX` in `payload`; the JSON stays valid and lossless (round-tripped through `encoding/json`). Decision 35,
scope.md, the user guide and CHANGELOG are amended. Whether a terminal honours 8-bit C1 in UTF-8 mode was not tested,
as the recommendation said.

## A1-52: Read can exceed MaxReadBytes by one record (low)

**Claim.** The cap is checked after each record, so a read returns up to cap plus one record (at defaults about 16 MiB plus up to 64 MiB - 32),
contradicting "at most `limit` records and `MaxReadBytes` of payload, but always at least one" (read.go:12-14, scope.md). Reproduced on macOS.

**Why held.** Either stopping earlier (an observable change to batch sizes) or rewording a documented bound.

**Recommended answer.** Reword the three docs to "stops once at least MaxReadBytes have been returned"; the shorter wordings already read
that way, the overshoot is bounded by one record, and a hard cap needs the frame length before cloning. Stopping before a record that would
exceed the cap unless `count == 0` is the alternative.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 25e96c2; fix-skeptic FIXED-VERIFIED (6 of 6). Documentation only: Read, ReadWait, Records
and Follow are documented as stopping once at least MaxReadBytes of payload have been returned, overshooting by at
most one record; a test pins the rule. Doc comments, the user guide, scope.md, decisions 50 and 79 and CHANGELOG are
amended.

## A1-57: `stat -json` holes use capitalised keys and `null` (low)

**Claim.** `quill stat -json` serialises a hole as `{"From":..,"To":..}` and an empty list as `null`, while every other CLI JSON key is
lowercase; no document fixes the stat, check or consumers schema.

**Why held.** The first public release freezes whatever scripts parse, so the shape is a decision, and changing it is a breaking change later.

**Recommended answer.** Approve before release: emit `[{"from":..,"to":..}]`, always an array, through a local tagged struct, and document the
stat, check and consumers JSON schemas in the reference.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in c91329c; fix-skeptic FIXED-VERIFIED (7 of 7). `quill stat -json` writes holes as
`[{"from":N,"to":N}]`, always an array, through a function-local tagged type (`quill.Hole` gains no tags); the stat,
check and consumers JSON schemas are documented in scope.md and pointed to from the user guide. Decision 35 and
CHANGELOG are amended.

## A1-87: the clock guard rounds up where the text says "at most 25%" (low)

**Claim.** `limit := (len(l.sealed) + 3) / 4` deletes the ceiling of a quarter (2 of 5, 4 of 15) where decision 44 and scope.md say "at most
25% (at least one)"; retention_test.go:266 pins the larger number.

**Why held.** Changing the code changes which segments a pass deletes and a test; changing the text amends a decision.

**Recommended answer.** Reword the text to "a quarter, rounded up, at least one" (the difference is at most three segments; the owner accepted
"25% per pass" without a rounding rule). The alternative is `max(1, n/4)` and a test change.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 50201e1; fix-skeptic FIXED-VERIFIED (6 of 6). Documentation only: every statement now reads
"at most a quarter of the sealed segments, rounded up (at least one)"; the code and retention_test.go are unchanged.
Decision 44 and CHANGELOG are amended.

## A1-91: a quill.policy of `{}` is acted on as retention 0 (low)

**Claim.** `{}`, `null` and `{"unknown":1}` parse as a valid policy of zeros; `quill ls` shows RETENTION 0s and `quill prune` runs with the default
24 h and no caps. A version-skewed policy (a future writer renaming fields) would produce such a file, and decision 68's rationale is that offline
tools must not prune with default retention. v0.1.0 writers always write every field (an off retention is `-1ns`).

**Why held.** The tolerant parse is deliberate (decisions 29 and 48) and one skeptic voted REFUTED (code and spec agree); the fix adds a refusal.

**Recommended answer.** Approve, low priority: treat a policy with no `retention` (or no known field) as unparseable for the offline tools, and print
the retention as "unknown" in `ls`. Declining is defensible.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 5c603fc; fix-skeptic FIXED-VERIFIED (7 of 7). `decodePolicy` fails when `retention` is
absent or empty, so `{}`, `null`, unknown fields only, or caps without retention are unparseable: `prune` refuses
without explicit flags, `ls` shows `?` and "policy unreadable" (the existing convention, not the word "unknown" the
recommendation used), `check` warns "cannot be parsed". Decisions 29, 48, 68 and 87, format.md, the `ReadPolicy`
comment and CHANGELOG are amended. Weakest point: a policy with `retention` present and every cap missing still parses
as caps of zero.

## Not held, but unresolved

**A1-62: `Config.Logger` is called with the log's mutex held.** A handler that calls back into the Log deadlocks it, and a blocked handler stalls
Append, Read, Sync and the flusher. A fix was written (commit fdcd880fe2e4, branch `stage2/A1-62`, not in this tree): records are queued under the lock
and delivered after it, on the caller's goroutine and, for the flusher's records, on a delivery goroutine that Close waits for. The fix-skeptic rejected
it: `logf` still calls the handler's `Enabled` method under `l.mu` (retention.go:294), so an `Enabled` that calls `l.Head()` deadlocks the log, and a handler
whose `Enabled` and `Handle` share a mutex stalls the flusher behind a blocked `Handle` (an Append and Sync loop did not finish in 8 s; the control finished in
0.38 s). The new documentation in Config.Logger, userguide.md:148 and scope.md promises more than the code gives. Recommended answer: finish the fix by not
calling `Enabled` under the lock (queue unconditionally and filter at delivery, or call `Enabled` in the deliverer) and add a test with a blocking `Enabled`; or
take the cheaper route and document that the handler runs under the log's lock and must neither call the Log nor block.

**A1-59: the Log's own reads return a raw ENOENT where the Reader returns a GapError.** Decided in round 1 (decision 103) and not held, but raised again
independently by two fresh auditors, with a real-concurrency reproduction (raw ENOENT in 25 of 1,601 reads on macOS and 17 of 1,483 on Linux). Recommended
answer: reopen the decision and take the one-line mapping at read.go:295, amending decision 103's sentence "a deleted one fails at its open as before", which
contradicts guarantee 14. See [a1-round2-correctness.md](a1-round2-correctness.md), item 16.

## Pass A4: dependency, supply-chain and release hygiene

The A4 pass (record: [supply-chain.md](supply-chain.md)) held back four items. Three of them are decisions or actions that only the
owner can take before a public release; the fourth (A4-1) was a written fix that did not reach main as first written, and has since
landed in this commit as the `build/quill` half only (the CI half was refuted). The recommended answers are the recorder's reading of
the auditor's proposed fix and the skeptics' notes, not decisions. Ranked by what the public flip needs.

| id | sev | one line | recommended | status 2026-10-30 |
|----|-----|----------|-------------|----|
| A4-5 | low | the release archives ship the Go runtime and standard library with no copy of its licence | add the Go licence text at the flip; reword decisions 22 and 15 | not in the A1h ledger; unchanged |
| A4-4 | medium | a workspace path is still in the v0.1.0 tag and in earlier history (the tip is fixed) | decide squash or filter, or accept, before pushing | not in the A1h ledger; unchanged |
| A4-3 | medium | the private reporting channel named in `SECURITY.md` exists only if the setting is enabled | enable private vulnerability reporting | not in the A1h ledger; unchanged |
| A4-1 | medium | the Go toolchain pin is enforced by the gate only; the `build/quill` half is fixed in this commit, the CI half was refuted | land the fix, rebased, with a fix-skeptic | approved; fixed in this commit (`build/quill` half only; the CI half was refuted because the pinned setup-go reads the toolchain line) |

## A4-5: the archives ship the Go runtime with no copy of its licence (low)

**Claim.** The release archives contain a statically linked Go runtime and standard library (BSD-3-Clause) and no copy of that licence or notice;
decision 22's reason for having no `THIRD-PARTY-NOTICES` ("no third-party code ships", decision 15) holds for vendored source but not for the linked
binaries. Two skeptic votes confirmed it; one read the licence's binary-redistribution clause and listed an existing archive.

**Why held.** The fix changes what ships in every archive and amends two recorded decisions (22 and 15), and whether a link satisfies the notice duty
is the owner's policy call. Decision 22 is explicitly a private-period rule that decision 54 supersedes when the `LICENSE` lands, so the right moment is
the flip.

**Recommended answer.** At the flip, add the Go licence text to each archive and to the repository (or a `THIRD-PARTY-NOTICES` file naming the Go
standard library and its licence) in the same change as quill's own `LICENSE`; copy it in `build/quill` beside the existing `LICENSE` copy; reword
decisions 22 and 15 to say that no third-party source is vendored while the standard library is linked in. Declining is defensible if the owner judges
the notice duty met by a link; record that choice in decision 22.

**Fix written.** None.

## A4-4: a workspace path remains in the v0.1.0 tag and in earlier history (medium; the tip is fixed)

**Claim.** `docs/audits/audit-correctness.md` named an absolute session path with a macOS user name and an internal parent-directory name. The tip no
longer does (main, commit 0b70e82, fix-skeptic FIXED-VERIFIED), but the text is still in the v0.1.0 tag tree and in every commit that contains the file
(75 commits by the auditor's count).

**Why held.** Removing it from history means rewriting history or moving to a new tag, and the audit may not move or create tags or push. The choice
also decides what the public repository's first commit looks like.

**Recommended answer.** Before the first push, choose one. (a) Publish a squashed or filtered history (for example `git filter-repo` replacing the
path), tag the result, and treat v0.1.0 as a private tag; this closes the exposure in the tag and the history, and the changelog line that mentions it
can then be reworded. (b) Accept the exposure: the path is not a secret and its parent-directory name is not on the forbidden-name list; the cost is a
user name and a directory name in public history. (c) Ship from a new tag on main and push only main plus that tag; this does not help, because main's
history still contains the path, so it is not an answer to the exposure and is listed only to rule it out. Recommendation: (a), done once, together with the
LICENSE commit. If (a) is taken, the filtered history must be re-checked with the same grep over every commit and the tag message.

**Fix written.** The tip edit (0b70e82). None for history.

## A4-3: the reporting channel in SECURITY.md needs a repository setting (medium)

**Claim.** `SECURITY.md` now (main, f2516d7) tells reporters to use the Security tab "Report a vulnerability" route (GitHub private vulnerability
reporting), with a fallback if the option is not shown, and says only the latest release is supported. The route exists only if the owner enables it;
the tree and the API the auditor had could not show whether it is enabled. The fix landed without a fix-skeptic and without a build or vet run (it is
documentation only), and the v0.1.0 tag still carries the old wording that pointed at `CODEOWNERS`.

**Why held.** Enabling a repository setting and choosing a contact are owner actions; the audit must not invent a mailbox or a key.

**Recommended answer.** Enable private vulnerability reporting before the flip and confirm that the Security tab shows "Report a vulnerability". If the
owner prefers a monitored role mailbox or a PGP key, name it in `SECURITY.md` in place of the fallback sentence. Release from a tag that contains the new
text (the v0.1.0 tag does not).

The "latest release only" support policy was introduced by f2516d7 and appears in no earlier document; confirm it or change it. The changelog line
(which ships in the release archives) no longer carries the enable-the-setting action; that action lives only here.

**Fix written.** f2516d7 on main (`SECURITY.md` and one changelog line), and a follow-up that removes the owner action from the changelog.

## A4-1: the Go toolchain pin is enforced only by the gate (medium)

**Claim.** The `toolchain go1.27.3` line in `go.mod` is a pin only in `hack/gate.sh`, which exports `GOTOOLCHAIN` from it. CI installs the newest 1.27.x
(setup-go ignores the toolchain line; **this half is REFUTED at the pinned v7.0.0, see the Correction below**) and `build/quill` uses whatever `go` is on PATH, although README.md:83-86 and CONTRIBUTING.md:40-42 call the go.mod
toolchain the one supported version. The comment at `ci.yml:65-66` ("the Go patch level comes from go.mod's toolchain line") was said to be false; it is true at the pinned setup-go (see the Correction below).

**Why held (as first written, 2026-10-29).** The fix was written but did not reach main: the fix workflow's merge check failed on a stray `go.mod` change (an audit probe, since restored by
c202882), and the commit is based on da2d638, before the A4-7 and A4-9 changes to `build/quill` and `ci.yml`. It has had no fix-skeptic. It changes what
the release build and every CI job run (the exact pinned Go instead of whatever is installed), which the owner should accept.

**Recommended answer.** Approve. Land f800bb7 (branch `stage2/A4-1-retry`) rebased onto main, then have a fix-skeptic attack it; confirm that CI still
passes with a workflow-level `GOTOOLCHAIN` on the new setup-go pin. If the owner prefers a floating patch in CI, drop the CI half, keep the `build/quill`
half, and reword README, CONTRIBUTING and the `ci.yml` comment to say that CI floats within the minor release.

**Fix written.** f800bb7: `build/quill` derives and exports `GOTOOLCHAIN` from go.mod's toolchain line as the gate does; `ci.yml` sets one workflow-level
`GOTOOLCHAIN` literal and corrects the comment; two tests (`toolchain_pin_test.go` and an edit to `build_script_test.go`) fail if the literal, the script
and go.mod drift apart (5 files, 159 insertions, 3 deletions).

**Owner approval.** Approved on 2026-10-30, as recommended (with its stated alternative of keeping only the
`build/quill` half).

**Disposition.** Fixed in this commit, for the `build/quill` half only. The recommended fix f800bb7 was superseded:
`build/quill` now derives and exports `GOTOOLCHAIN` from go.mod's toolchain line, as `hack/gate.sh` does, so the release
binaries and the SBOM come from the pinned compiler; `TestBuildScriptPinsToolchain` fails without it. The CI half of the
claim was refuted (see the correction below): the workflow pins setup-go v7.0.0 (`ci.yml` lines 24, 44, 54, 75 and 87),
which reads the toolchain line, so the CI half was dropped and the `ci.yml` comment about the patch level is true. The
first re-fix attempt (41c8aa4) carried the false CI claim and was not used.

**Correction after the fix-skeptic (2026-10-30).** The CI half of the claim and of f800bb7 is false at the pinned action. actions/setup-go at b7ad1dad
(v7.0.0), `src/installer.ts` `parseGoVersionFile` (read from the action's repository at that commit), reads go.mod's `toolchain` line first, whenever
`GOTOOLCHAIN` is not `local`, and uses the `go` line only as a fallback; so CI already installs exactly the toolchain `go.mod` names. `src/main.ts`
then exports `GOTOOLCHAIN=local` to later steps through the runner env file, which would override a workflow-level literal (inferred from the source,
not run on a runner). The literal and its test were therefore dropped, and so was the "ignores the toolchain line" wording. What landed is the
`build/quill` half (with `TestBuildScriptPinsToolchain`, which fails without it), the corrected `ci.yml` comment and the README, CONTRIBUTING, decision 88
and changelog statements. The owner's alternative (CI floats within the minor release) is not needed.

## Not held, but worth a line at the flip

**A4-2 (refuted, 2 votes to 1).** `go 1.27` is the documented latest-toolchain policy, so importers need Go 1.27 or later. If the owner wants a lower
consumer floor for the first public release, that is a product decision: set `go 1.23` (or the measured minimum) and keep the toolchain line, then rerun
the gate (the `cmd/quill` tests already need Go 1.24 for `b.Loop`). Otherwise state the floor outright in README.md:86, CONTRIBUTING.md:42 and
`scope.md:94`.

**A4-6 (refuted, 2 votes to 0).** `build/quill` skips a missing `LICENSE` silently, which is intended in the private period. At the flip, consider making
the release form fail without one, in the same change that adds the `LICENSE`.

## Pass A3: security review and threat model

Date 2026-10-31. Pass A3 (record: [threat-model.md](threat-model.md)) applied eleven fixes, each re-attacked by a fix-skeptic
(11 of 11 FIXED-VERIFIED), and held back four items whose fix would change a documented file mode, a CLI behaviour or a
documented performance statement: A3-5, A3-14, A3-18 and A3-19. None of the four has a fix written; the recommended answers
below are the recorder's reading of the auditors' proposals and the skeptics' notes, and are recommendations, not
decisions. Two further items have a fix that was written and rejected by its fix-skeptic and is not in the tree (A3-10 and
A3-20); they are listed after the four, as "not held, but unresolved". Items the auditors raised that the project had
already decided in round 1, or that a vote refuted, are not held; they are listed last with the proposal on file.

| id | sev | one line | recommended | status |
|----|-----|----------|-------------|--------|
| A3-5 | medium | a reader-only process can flock `quill.lock` when Perm grants read, blocking writer Open | approve 0600 for the writer lock; settle Archive's probe first | approved 2026-10-31; fixed 47315f1 (FIXED-VERIFIED, 7 of 7) |
| A3-14 | medium | `quill archive -to` writes into a directory that has no `quill.format` | approve: require an empty directory | approved 2026-10-31; fixed 303b794 (FIXED-VERIFIED, 7 of 7) |
| A3-18 | low | a Reader's read cost scales with the log, not the request | reword now; cache sealed refs after the release | approved 2026-10-31; reworded 3c36fc0 (wording only); the cache is deferred |
| A3-19 | low | the first-frame probe reads and CRC-checks the whole first frame (up to 64 MiB) | correct decision 47's wording; leave the code | approved 2026-10-31; reworded 3c36fc0 (wording only); code unchanged |

## A3-5: the writer's lock file is readable by readers, so a reader can hold it (medium)

**Claim.** `quill.lock` is created with Config.Perm minus umask. When Perm grants group or other read, a process that can only
read can open the lock `O_RDONLY` and take `LOCK_EX` with flock, which makes every writer Open (a restart included) fail with
ErrLocked until the holder exits. Verified on Linux with two uids (uid 1000 writer, `nobody` holder); with Perm 0600 the
holder gets "permission denied". Consumer locks (`consumers/<name>.lock`) are created from the log mode too and are subject to
the same hold.

**Why held.** The fix changes a documented file mode. Decision 51 says a lock file "is created through the process umask",
deliberately, and the Perm row of scope.md calls a shared mode a supported configuration. The fix would make the writer lock
owner-only whatever Perm says, and so changes what an operator sees in the directory and what a different-uid tool can do with
the lock (see the Archive caveat below).

**Recommended answer.** Approve. Create and keep `quill.lock` at 0600 regardless of Config.Perm (only the writer and Archive's
destination probe touch it); record it in decision 51 and in the user guide's group-arrangement text; leave consumer locks
group-accessible, because a group consumer must be able to take its own lock, and state in decision 41 that a reader-only
process with read access can therefore hold a consumer's lock and stall that consumer (not the writer).

Caveat on the auditor's proposal (inferred, not run): the proposal changes Archive's destination probe (archive.go:116) to open
`O_RDONLY` so a non-owner archiver can still detect a live writer. A 0600 file owned by another uid cannot be opened `O_RDONLY`
either, so a different-uid archiver would get a permission error on the probe. Decide whether that error is treated as "a writer
may be live, refuse" (fail closed, recommended) and say so in the Archive doc. Test: a log created by one uid with Perm 0644,
a probe from another uid, before and after.

**Fix written.** None. The proposed change is in threat-model.md, A3-5.

**Owner approval.** Approved on 2026-10-31, as recommended.

**Disposition.** Fixed in 47315f1 (on main; verified from `git log`); fix-skeptic FIXED-VERIFIED (7 of 7 lenses pass; reported, from a skeptic that ran everything on scratch copies, on macOS and on the Linux box). `quill.lock` is always created 0600 whatever `Config.Perm` says; a wider lock left by an earlier build is tightened (fstat, then fchmod) before the flock is tried. A failed chmod does not fail `Open`: it logs a Warn on `Config.Logger`, because the held lock is what keeps the log single-writer and refusing would turn a hardening step into an outage (the fixer's choice where the recommendation was silent). Archive's destination probe fails closed: a permission error opening the destination's lock returns an error that is both `ErrLocked` and `fs.ErrPermission`. Consumer locks keep the log's mode, and decision 41 states that a reader-only process with read access can hold a consumer's lock and stall that consumer, never the writer. Decisions 41 and 51, scope.md, format.md, the user guide and CHANGELOG are amended. Reproduction before and after (verified by the fixer, Linux, uid 1000 against `nobody`): the lock was 0644 and `nobody` held it, so the owner's `Open` failed with a locked error; after the fix the lock is 0600, `nobody` gets "Permission denied" and the owner's `Open` succeeds. Gate green on macOS and Linux on the fix commit. Not observed: the Archive probe from a second account on an account-owned lock; the skeptic ran a root-owned 0600 destination lock on the Linux box (refused, both error classes satisfied, nothing written) and the fixer a mode-0000 lock. Two minor gaps, neither a regression: the CLI's offline verbs filter log output down to two messages, so the new "could not be tightened" Warn reaches only library users who set `Config.Logger` (A5-36 later widened that filter, but with other messages; this one is still not passed, and adding its prefix to `lossOrShortfall` in cmd/quill/write.go is a one-line change if the owner wants offline verbs to say so); and the Archive probe reports every permission error as "a writer may be live", including an unsearchable destination directory (fails closed, wording only).

## A3-14: `archive -to` writes into a directory that is not a log (medium)

**Claim.** `archiveDest` (archive.go:108 now) checks only same-file as the source, a held writer lock and the format id when
`quill.format` exists; when it is absent it writes one. `copySegment` then renames over `NNNNNNNNNN.log` whenever the existing
file's size differs (archive.go:178 now). A mistyped `-to` that names an existing directory without `quill.format` therefore
replaces an unrelated file named like a segment and adds `quill.format`, `quill.tail` and `consumers/`. Writable Open refuses the
same directory ("holds files but no quill.format", log.go:508 now). Reproduced on macOS with the built binary and through
`quill.Archive`: a 24-byte text file named `0000000000.log` became a segment copy and the command exited 0.

**Why held.** It makes a previously working invocation fail (archiving into a populated directory that is not yet a log), and it
changes the error an operator sees (ErrNoLog from Archive), which is an exported error class on that function.

**Recommended answer.** Approve. When `quill.format` is absent in the destination, require the directory to be empty, ignoring
`.tmp-*` entries and `quill.lock`, as `Log.create()` does, and otherwise return ErrNoLog; add a library test and a CLI test
that plant an unrelated `0000000000.log` and assert it is untouched; amend decision 85. Archive writes `quill.format` first, so a
destination left by an interrupted earlier run already has one and is unaffected (inferred from archive.go, not run).

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended.

**Disposition.** Fixed in 303b794 (on main; verified from `git log`); fix-skeptic FIXED-VERIFIED (7 of 7 lenses pass; reported). When `quill.format` is absent the destination must be empty, ignoring `.tmp-*` files and `quill.lock`, and otherwise `Archive` returns an error wrapping `ErrNoLog` before writing anything; the CLI exits 1. Reproduction before and after (verified by the skeptic with the built CLI): a destination holding an unrelated `0000000000.log` and `notes.txt` was changed (the log replaced, `quill.format`, `quill.tail`, `consumers/` added, exit 0) before the fix; after it exits 1 naming the directory and every file and directory is byte-identical. A destination left by an interrupted run already has `quill.format` and is unaffected (tested at every successive write of the destination, and by deleting trailing files). Decision 85, the `Archive` doc comment, the user guide and CHANGELOG say plainly that archiving into a populated directory that is not yet a log now fails. Gate green on macOS and Linux on the fix commit. Two minor points, neither caused by the fix: the message says "holds files" even when the destination holds only a subdirectory and does not say that an empty directory is the remedy; and a refused run still touches the source's protected `_archive` consumer, because `archive()` opens it before `archiveDest` (true of the other destination refusals too).

## A3-18: a Reader's read cost scales with the log, not the request (low)

**Claim.** A `Reader.Read`, `ReadWait`, `Scan` or `Consumer.Read` without an iterator cursor re-walks and CRC-checks the active
segment from byte 0 to the requested offset (read.go:203 now), and each call rebuilds a per-segment view in O(sealed segments)
(reader.go:334, a `ReadDir` plus a `segRef` and two closures per segment). Measured on the Linux box with 3,000,000 records in
one 343 MB active segment: `Reader.Read(2,999,000,1)` 379 ms against 112 us for the writer's `Log.Read`; 100 Reads of 1,000
records from 2,000,000 took 26.3 s against 0.33 s for 100,000 records through Records. One skeptic's runs at 5,000 to 40,000
one-record segments grew linearly (5.4 to 51 ms, 2.4 to 16 MB per call). This contradicts scope.md's "Read cost does not grow
with backlog" for the Reader's active segment.

**Why held.** Either fix changes a documented statement or a design choice. Decision 63 chose "the active segment is never
cached" on purpose, for correctness after a writer crash, and decision 103 gives the Log, not the Reader, a sealed-ref cache.
Caching in the Reader changes how fresh its view of a directory is.

**Recommended answer.** Two steps. Now, reword: scope.md's Performance gate and decision 63 should say that a cursorless Reader
read of the active segment costs O(offset within the segment), bounded by SegmentBytes (a few milliseconds at the default 8
MiB, an extrapolation both skeptics made, and the 1,000,000-record macOS run is consistent with it), that Records and Follow carry
a byte cursor and avoid it, and that a polling consumer should use them. After the release, if wanted, cache the sealed refs
between views the way `Log.sealedRefsLocked` does. A per-Reader anchor for the active segment, or a cursor returned from `Read`,
is the larger change and only worth it if polling consumers are expected.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: reword now; the sealed-ref cache after the release is deferred.

**Disposition.** Wording only, in 3c36fc0 (on main; verified from `git log`); no code changed and no fix-skeptic ran. scope.md's performance gate and decision 63 now state that a read with no iterator cursor (`Reader.Read`, `ReadWait`, `Scan`, `Consumer.Read`) re-walks and CRC-checks the active segment from byte 0 to the requested offset, O(offset within the segment) and bounded by `SegmentBytes`, and rebuilds the sealed-segment view, O(sealed segments), on every call; `Records` and `Follow` avoid both, and the user guide tells polling consumers to use them. The measured 379 ms is the auditor's, on the Linux box, at the top of a 343 MB active segment; "a few milliseconds at the default 8 MiB segment" is an extrapolation and is labelled as one in scope.md. Not done: caching sealed refs between views as `Log` does (decision 103).

## A3-19: the first-frame probe reads and CRC-checks the whole first frame (low, PLAUSIBLE)

**Claim.** The Reader's probe of each uncached sealed segment (reader.go:249 now, `window.frameAt(0)`) reads, allocates and
CRC-checks the whole first frame, up to 64 MiB, to learn two integers; decision 47 says "a few bytes". A directory of valid 64
MiB frames stored as sparse files made `quill stat` take 8.99 s and allocate 6.4 GiB on the Linux box (auditor; not re-run by a
skeptic). A skeptic's smaller run confirmed one full frame allocated per segment (40 MiB for five 8 MiB first frames).

**Why held.** The auditor's proposed fix is not behaviour-preserving. One skeptic patched the probe to read 64 bytes and skip
the CRC: two existing tests that assert hole detection failed, because an unchecked first-frame sequence feeds the segment
ranges. Skipping the CRC only above a size threshold leaves exactly the large first frames unchecked. The verdict is PLAUSIBLE
because the voters split on the remedy, not on the mechanism.

**Recommended answer.** Correct the wording, leave the code. Change decision 47 from "a few bytes" to "the whole first frame,
CRC-checked, up to 64 MiB, once per Reader per segment", and state the cost in the user guide. The cost needs write access to a
directory the operator reads (a restored log, or a subdirectory under `quill ls`), and is bounded at 64 MiB per call and
collectable. Revisit only if a bounded cheaper integrity check is designed, for example reading a header prefix and deferring
the CRC to the first scan of that segment, with a test that a damaged first frame is still reported as a hole.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: correct the wording, leave the code.

**Disposition.** Wording only, in 3c36fc0 (on main; verified from `git log`); no code changed and no fix-skeptic ran. Decision 47 now says the probe reads and CRC-checks the whole first frame, up to 64 MiB, once per `Reader` per segment, with the reason it is not cheaper (two hole-detection tests fail if the check is skipped), and the user guide states the cost for a directory whose first frames are very large. A cheaper bounded check remains a possible later change, with a test that a damaged first frame is still reported as a hole.

## Not held, but unresolved

**A3-10: a damaged oldest segment is skipped silently by a default-start read (low).** A fix was written (54b99d0, not in this
tree): a read from the beginning (`ScanAll`, a scan from offset 0, `quill read` and `cat` without `-from`, `tail -n` reaching the
oldest record) and a new consumer start at sequence 1 when a damaged oldest segment is a hole from 1, so they return its
GapError (CLI exit 1); a prefix deleted by retention, `-since` and `OffsetForTime` still start at `First`. Decision 41, scope.md's
Consumer row, the CLI help, the regenerated docs/reference.md, the user guide and the changelog carry the exception. The
fix-skeptic found the code sound and rejected it on the no-new-gap lens only: three statements still say a new consumer starts
at the first retained sequence, the godoc for `Log.Consumer` (consumer.go:330-331), the comment on `consumerStore.open`
(consumer.go:78-79) and the `Config.ProtectedConsumers` row at scope.md:121. Recommended answer: approve the behaviour (it is
what decision 38 and scope.md already say: "reported as a gap, never skipped"), update those three statements, and land the
commit. If the owner prefers expiry-like treatment of a damaged oldest segment, the alternative is to say so in decision 38 and
change no code.

**A3-20: a Reader serves overlapping segments (low).** A fix was written (e0df757, not in this tree): a Reader refuses overlapping
segments as ErrCorrupt and Check reports an "overlap" problem; "overlap" is added to the closed `kind` list for `quill check
-json` in scope.md. The fix-skeptic rejected it because Archive, another path through `Reader.view`, is not covered:
`copySegment` (archive.go:167-192) never compares the segment's real last sequence with the reference's derived last, so on a
hand-built overlap Archive copies both segments, writes a destination `quill.tail` with seq 8 and returns GapError{From:11,To:4}
("records 11 to 4 are unavailable"), the same as before the fix. Recommended answer: add an overlap check in `copySegment` or
`archive` (reject when `info.last.Seq` is above the reference's bounded last, or run the reference's `ensureIndex`), add an Archive
test on the layout in threat-model.md A3-20, and land the commit. The new `kind` value is an addition to a documented JSON
field, so the owner should accept it.

## Not held, but worth a line

**A3-26 (refuted, 1 to 2): unlinking `quill.lock` under a live writer lets a second writer open.** Out of the threat model and a
stated design rule, so not changed. Two hardening options were written up by the auditor, if the owner wants defence in depth:
compare the held descriptor's device and inode with `Lstat(quill.lock)` before each marker publish and fail the log with an
ErrLocked-wrapping error on mismatch (narrows the window to one flush); or also flock the log directory's own descriptor (probed
as exclusive on macOS and Linux). Cheapest step: one sentence in format.md and the user guide, "never delete `quill.lock`;
flock releases itself on process death", naming the consequence (two acknowledged writers, one of them silently overwritten,
`check` HEALTHY).

**A3-7 (refuted, 1 to 2): a symlinked `consumers/` is followed.** Hardening only: open `consumers/` once with
`O_DIRECTORY|O_NOFOLLOW` (or `Lstat` it in `create()` and every consumer operation) and require a real directory, or say in
decision 51 that only the last path component is protected.

**A3-24 (refuted, 1 to 2): `-json` passes bidi and format controls raw.** Escape U+200B-U+200F, U+202A-U+202E, U+2060-U+2069,
U+FEFF and U+061C as `\uXXXX` (still lossless) and print `time` as null outside years 0000-9999. It changes the `-json` output
shape, so it needs approval if wanted.

**Raised again, decided in round 1 (not re-opened):** A3-3 (file names printed unescaped; quote file-derived strings, skip or
flag invalid consumer names), A3-4 (a non-consumer file in `consumers/` protects every segment; ignore invalid names, report them
in `check`), A3-6 (Archive keeps a same-size destination segment without verifying it; verify or compare, and re-copy on
difference), A3-9 (a Reader takes its file mode from `quill.tail`; clamp it and intersect with the umask), A3-11 (`quill prune`
trusts `quill.policy`; print the policy and parse it strictly), A3-23 (offline verbs create a log at a mistyped `-dir`; fail with
ErrNoLog unless `quill.format` exists), A3-29 (name the filesystems where flock is not coherent, correct the bind-mount sentence,
optionally warn from an `fstatfs` check), A3-31 (ConsumerTTL staleness is clock-derived; document it), A3-32 (MinFreeBytes
re-measure after each unlink; document the cost). Each carries its round-1 reference and the auditor's proposal in
[threat-model.md](threat-model.md). A3-3, A3-6 and A3-23 were raised again in this pass with reproductions on the current tree,
so the owner may want to confirm the round-1 decision for them before the release.

## Pass A6: format and version-skew audit

Date 2026-10-31. Pass A6 (record: [format-compatibility.md](format-compatibility.md)) applied seven fixes, each re-attacked
by a fix-skeptic (7 of 7 FIXED-VERIFIED): A6-1, A6-2, A6-3, A6-5, A6-6, A6-7 and A6-9. It left two ledger items open,
A6-4 (recorded as held for the owner) and A6-8 (fix written, commit did not land), and one built tool, the skew harness,
that its reviewer rejected. The recommended answers are the recorder's reading of the auditors' proposals and the
skeptics' notes, and are recommendations, not decisions.

**Update 2026-10-31.** The owner approved A6-4 (confirm and close) and A6-8 (do it), both recorded in their sections below. The skew harness the reviewer rejected was rebuilt as 01cee10 (`hack/skew.sh`, `hack/skewopen`, CONTRIBUTING.md). The missing check was added: in the kill phase each version runs `recover` on its own copy of the same killed directory and the two copies must be identical (`diff -r`). Verified on macOS: v0.1.0 against 3c36fc0 passes in both directions, ten kills each, every kill leaving part of the stream; and a deliberate fault (recovery stops at the marker, dropping the frames after it) now makes it fail with `rc=1` and the two recovered `quill.tail` files in the evidence (seq 490 against 425). The previous script is reported, by its reviewer, to have passed that same fault; I did not re-run it. The gate is green on macOS and Linux on the harness commit. Not run: the cross-version run on Linux (the Linux box ran only the gate).

| id | sev | one line | recommended | status |
|----|-----|----------|-------------|--------|
| A6-4 | low | the writer could rotate into segment id 10000000000, which no scan recognises | confirm and close; the fix is already in the tree as A3-8 (bb646c0) | approved 2026-10-31 (confirm and close); closed, reproduction re-run on the fixed tree |
| A6-8 | low | `MaxSegmentBytes` (1 GiB) and `maxSegmentBytes` (1.5 GiB) differ only by case | rename the unexported one, one commit | approved 2026-10-31 ("do it"); fixed 003de2c (FIXED-VERIFIED, 6 of 6) |

## A6-4: the segment id has no ceiling (low)

**Claim.** The segment id is written `%010d` and parsed as exactly ten digits, but nothing stopped the writer creating id
10000000000. That file is invisible to every scan; the newest-segment check then refuses the log (ErrCorrupt), and
records acknowledged into it are unreachable by a writable open (readers still serve them). Reproduced on macOS by the
auditor and a skeptic: a log whose segment was renamed 9999999999.log with `quill.tail` seg and maxseg at that value,
then Open, Append, Roll, Append, Close all succeeded and the next Open failed with "quill.tail names segment
10000000000, the newest on disk is 9999999999". A forged `maxseg` of 99999999999 in an ordinary log gave the same
failure after one roll.

**Why held.** The fix makes the writer stop with an error at a documented limit (a changed behaviour at the ceiling, like
the sequence ceiling) and states a new limit in format.md. The ledger marked it semantic. Reaching the ceiling by
rotation needs about 10^10 rotations, so no real log is expected to get there; the cost of the decision is low either way.

**Recommended answer.** Confirm and close. The proposed fix, refuse a rotation whose next id would exceed 9999999999 and say
so in format.md, is already in the tree: the A3 pass fixed the same defect as A3-8 in bb646c0 (an ancestor of the A6
record commit). Verified by the recorder from `git show bb646c0`: `decodeTail` rejects `maxseg`, and so `seg`, above
`maxSegmentID` (9,999,999,999) on both the fast and slow paths; `rotateLocked` returns "segment ids exhausted" when
`l.maxSeg >= maxSegmentID`; docs/format.md and CHANGELOG say so; `TestRotationRefusesAnIDTheSegmentNamesCannotHold` seeds
the top ids and passed when the recorder ran it. The A3 pass's fix-skeptic verdict, FIXED-VERIFIED, is reported from
[threat-model.md](threat-model.md). What the owner decides is only whether the A3-8 wording ("the largest id ten digits
hold", an error at maxseg 9999999999, so the id 9999999999 is the last one usable) is the intended limit.

**Fix written.** bb646c0, landed under A3-8. No separate A6 commit exists, and no A6 fix-skeptic ran on it. The A6-4
reproduction itself was not re-run against the fixed tree (inferred from the decoder and rotation lines).

**Owner approval.** Approved on 2026-10-31, as recommended (confirm and close).

**Disposition.** Closed. The fix is bb646c0 under A3-8 (fix-skeptic FIXED-VERIFIED in the A3 pass; reported, from [threat-model.md](threat-model.md)), so the A6 ledger item moves from held to closed with no new commit. The record said the A6-4 reproduction had not been re-run on the fixed tree; it was re-run on 2026-10-31 with built CLIs (verified). A log whose only segment is renamed 9999999999.log, with `quill.tail` `seg` and `maxseg` at that value, then `roll`: before bb646c0 (its parent), `roll` exited 0 and created 10000000000.log, and the next `append` failed with "corrupt log: quill.tail names segment 10000000000, the newest on disk is 9999999999"; on main, `roll` is refused ("segment ids exhausted: 9999999999 is the largest the ten-digit names hold", exit 1), no 10000000000.log is created and `append` still succeeds. A forged `maxseg` of 99999999999 is refused at read ("maxseg 99999999999 above the largest segment id, 9999999999"). The limit as written in A3-8 stands: 9999999999 is the last usable segment id.

## A6-8: two constants differ only by letter case (low)

**Claim.** `MaxSegmentBytes` (1 GiB, the SegmentBytes configuration ceiling, log.go:97 now) and `maxSegmentBytes` (1.5 GiB,
the segment file limit and the bound on `quill.tail` `off`, load.go:19 now) differ only by the case of the first letter.
Both are used correctly today and a change to the lowercase value is caught by several tests; the risk is a mistyped
capital moving an on-disk limit by 0.5 GiB with the compiler silent.

**Why held.** It was not held for a decision. A fix was written (rename the unexported constant to `maxSegmentFileBytes`,
keep the exported name, no API effect) and its commit did not land: the workflow handed its fix-skeptic the placeholder
`PENDING` in place of a sha, `git rev-parse --verify PENDING` failed, and nothing was merged or reviewed. The tree still
has both names (verified by the recorder, grep).

**Recommended answer.** Do the rename in one mechanical commit (five files use the lowercase name: fuzz_test.go,
load_test.go, load.go, log.go, state.go), gate it, and have a fix-skeptic check that no behaviour moved. Do not rename the
exported constant, which is public API. Declining is defensible: this is hygiene, and a mutation of the lowercase value is
already caught by the segment-limit tests.

**Fix written.** A proposal only; no commit.

**Owner approval.** Approved on 2026-10-31 ("do it"), as recommended: rename the unexported constant, keep the exported one.

**Disposition.** Fixed in 003de2c (on main; verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported). The unexported `maxSegmentBytes` (1536 MiB, the segment file limit and the bound on `quill.tail` `off`) is now `maxSegmentFileBytes` in the five files that used it; `MaxSegmentBytes` (1 GiB, the `SegmentBytes` ceiling) is untouched. Verified by the recorder: the word-level diff is 17 identifier pairs and nothing else, and the gate is green on macOS and Linux. The skeptic checked that the exported API (`go doc -all`) is byte-identical, the value is unchanged, no live document keeps the old name, and that changing the new constant to 1 GiB makes three tests fail, so the segment-limit tests still guard it.

## Not held, but unresolved

**The skew harness cannot see a cross-version difference in recovery.** A script to rerun the v0.1.0 against HEAD
experiments (hack/skew.sh in the working copy it was built in) was built in pass A6. It is not in the tree, and its
reviewer rejected it: with the new version's `recoverSegment` changed to drop the post-marker frames v0.1.0 salvages, the
run still printed "OK: no divergence between v0.1.0 and HEAD" and exited 0. Its kill phase recovers each killed
directory with one version only and checks that the survivors are a contiguous, unchanged prefix; it never recovers a
copy of the same killed bytes with both versions and compares. The frame-width, CRC and policy faults were caught. Recommended
answer: if the owner wants the harness in the release, copy each killed directory, recover one copy with each version,
and `diff -r` the two before the append; then repeat the reviewer's deliberate-fault test and require the run to fail.
If the harness is not wanted, nothing in the format verdict depends on it: the manual kill experiment (180 iterations,
recovery by each version of identical torn bytes) made exactly that comparison and found no difference, as
[format-compatibility.md](format-compatibility.md) records.

**A6-6 stays PLAUSIBLE.** No fixture in the tree is a frame written by the reference implementation; docs/format.md now
says so (ee11f57), and compatibility with the reference rests on the specification. If the owner can supply a short
reference-written segment (a few records, one with an empty payload and one with a negative timestamp), check it in under
testdata with its provenance given as "the reference implementation" and decode it in `TestFrameGolden`; that
would settle decision 3 by bytes.

## Pass A5: silent-degradation audit

Date 2026-10-31. Pass A5 (record: [silent-degradation.md](silent-degradation.md)) applied seven fixes, each re-attacked by a
fix-skeptic (7 of 7 FIXED-VERIFIED): A5-2, A5-16, A5-21, A5-31, A5-39, A5-40 and A5-42. It held eleven items for the owner.
Every held item changes observable behaviour (a `Stats` field, an `Err()` rule, an exit code, a CLI line, a validation
refusal or documented output), so none was applied. No fix is written for any of them; each section gives the auditors'
proposal as the recommended answer. The recommended answers are the recorder's reading of the auditors' proposals and the
skeptics' notes, and are recommendations, not decisions. The full items (grounds, warrant, backing, rebuttal, votes,
reproductions) are in the record.

**Update 2026-10-31 (pass A5h).** The paragraph above is the state when the pass was recorded. The owner approved all eleven recommended answers on 2026-10-31. Result: 10 fixed and FIXED-VERIFIED (A5-10 91c2fce, A5-22 1dda449, A5-18 7b87e64, A5-36 2cbfdf2, A5-9 46f30db, A5-3 1e0a8ed, A5-23 4d0e8c9, A5-44 29e7db1, A5-29 caad4c5, A5-32 16cb0c8), and A5-1, which failed its first fix-skeptic and was fixed on the third attempt (99b140d, FIXED-VERIFIED, 2026-10-31). The table's status column and each section's **Disposition** are current; each section's "Fix written" line is the state at recording. Two approvals differ from the recommendation text: A5-22 declined the "N consecutive statfs failures" option, and A5-29 reverses the round-1 exit-0 ruling (A1-58, decisions 85 and 53). The per-item reproduction and fix-skeptic evidence are in [silent-degradation.md](silent-degradation.md). In `git log` order the commits are A5-10, A5-22, A5-18, A5-36, A5-9, A5-3, A5-23, A5-44, A5-29, A5-32 (verified), all ancestors of 16cb0c8, on which the A5-32 skeptic ran staticcheck, deadcode and the full race suite, all clean (reported).

Three of the items share one design question and should be decided together: A5-10 (what `Err()` means), A5-1 (what a
background failure reports) and A5-22 (what `diskFull` reports). Two share another, the CLI's stderr: A5-36 and the
broken-consumer Warn left over from A5-17. The offline write verbs now carry a stderr Logger (cmd/quill/write.go, the
`lossOnly` filter, from A5-2 and A5-42) that passes two messages only; widening that filter is the smallest route for the
second group (inferred from reading write.go:74).

| id | sev | one line | recommended | status |
|----|-----|----------|-------------|--------|
| A5-10 | medium | `Err()` is one latched slot: a load error masks later background errors and a recovered error never clears | stop the masking, clear on recovery | approved 2026-10-31; fixed 91c2fce (FIXED-VERIFIED, 6 of 6) |
| A5-1 | medium | flush, publish, rotate and prune failures reach only `Err()`; nothing is logged, timed or counted | log each transition; add `Stats` fields | approved 2026-10-31; fixed 99b140d on the third attempt (FIXED-VERIFIED) |
| A5-36 | medium | `quill prune` exits 0 with no message when an unlink fails, a consumer file is corrupt or the clock guard withholds segments | widen the stderr filter; exit 1 on `Err()` | approved 2026-10-31; fixed 2cbfdf2 (FIXED-VERIFIED, 6 of 6) |
| A5-18 | medium | a consumer below the first retained sequence reads healthy in `quill consumers` and `quill check` | derive a NOTE from the files | approved 2026-10-31; fixed 7b87e64 (FIXED-VERIFIED, 6 of 6) |
| A5-22 | medium | a failing statfs leaves `MinFreeBytes` unenforced or `diskFull` frozen, and `Err()` stays nil | set `Err()`, add `Stats.DiskFull` | approved 2026-10-31 (no N-failure rule); fixed 1dda449 (FIXED-VERIFIED, 6 of 6) |
| A5-29 | medium | `quill archive` reports a source gap once with exit 0, and not at all after `unregister` | exit 3 on a gap; compare with the destination's tail | approved 2026-10-31; fixed caad4c5 (FIXED-VERIFIED, 6 of 6) |
| A5-3 | low | `ReadWait` and `Follow` on a failed log block until the context ends | return `ErrFailed` from the wait | approved 2026-10-31; fixed 1e0a8ed (FIXED-VERIFIED, 6 of 6) |
| A5-9 | low | `quill prune` and `quill roll` discard `Close`'s error | report it, exit 1 | approved 2026-10-31; fixed 46f30db (FIXED-VERIFIED, 6 of 6) |
| A5-23 | low | the published disk bound is exceeded when `MaxBufferBytes` is below `FlushBytes` | refuse that Config in `validate` | approved 2026-10-31; fixed 4d0e8c9 (FIXED-VERIFIED, 6 of 6) |
| A5-32 | low | an archive rerun keeps a same-size damaged destination segment | verify the CRC and replace it | approved 2026-10-31; fixed 16cb0c8 (FIXED-VERIFIED, 6 of 6) |
| A5-44 | low | `quill ls` omits a log whose reader cannot open | skip only `ErrNoLog`; print an error row | approved 2026-10-31; fixed 29e7db1 (FIXED-VERIFIED, 6 of 6) |

## A5-10: `Err()` is masked and never clears (medium)

**Claim.** `Log.Err` returns the failed state, then the sealed-segment load error, then the last background error, so an old
hole permanently hides every later rotation, flush or prune failure, and a background error is never cleared when the
operation recovers. After any recovery `Err()` reports a failure that is over. Reproduced by a program on the public API on
macOS: after a rotation blocker was removed and rotation succeeded, `Err()` still returned the old EEXIST; with a damaged
sealed segment, a live rotation failure was invisible in `Err()`.

**Why held.** The fix changes what the documented signal means (docs/userguide.md tells operators to watch `Err()`;
scope.md calls it the background error and the sticky load error), and either adds `Stats` fields or changes what `Err()`
returns. That is a public contract at v0.1.0.

**Recommended answer.** Keep the sticky load error but stop it shadowing: return the join of the failed or load error and
the background error, or add `Stats.BackgroundErr` and `Stats.BackgroundErrAt`. Clear the background error, or stamp a
recovery time, when the operation that set it succeeds (flush clears the flush error, a rotation the rotation error, a clean
prune pass the prune error). Drop the load error once the segment it names has been deleted and its hole pruned.

**Fix written.** None. One case already works this way since A5-39: a policy write that fails and later succeeds clears its
own `Err()` (log.go:664), which is the pattern to extend.

**Owner approval.** Approved on 2026-10-31, as recommended: the join (not the `Stats.BackgroundErr` alternative), clear on recovery, drop the load error with its hole.

**Disposition.** Fixed in 91c2fce (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). `Err()` (log.go:634) no longer masks or latches. Background errors are held one slot per operation in `Log.bg` (flush write, marker publish, rotation, prune unlink and policy write; A5-22 later added a disk slot), each set by `setBg` and cleared by `clearBg` when its operation succeeds (log.go:671-674). `Err()` returns the failed state, or the sticky load error when the log is not failed, joined by `errors.Join` with every live slot, so `errors.Is` matches each; a single error comes back unwrapped. The load error is recomputed to the first remaining damaged sealed segment, or nil, when retention deletes segments. The separate `policyErr` field and the displacement logic in `recordPolicy` are folded into the policy slot (reported, fix summary). Residuals, reported by the skeptic and not fixed. (1) `Roll` (flush.go:230) and the age rotation in the prune pass (retention.go:152) call `setBg(bgRotate, err)` without the `l.failed` guard that `publishLocked` has (flush.go:143), so when the rotation fails fatally on the directory fsync `Err()` holds the failed error twice and returns `errors.Join(failed, failed)`: the same line twice. `errors.Is` and the nil or non-nil contract are unaffected, and before the fix the error came back once. I read the three sites in the tree as it is now and the guard is still absent from the two (verified). A one-line `if l.failed == nil` in each, or a dedupe in `Err()`, removes it; cosmetic, recommended, not applied. (2) A failed age rotation stays in `Err()` after its blocker is gone until the next rotation of either kind, which is arguably accurate because none has succeeded since. Not run by the skeptic: the fuzz step and `go mod tidy`.

## A5-1: background failures are not logged, timed or counted (medium)

**Claim.** Every retryable background failure is reported through one latched field, surfaced only by `Err()`. Nothing
reaches `Config.Logger` although the field documents "warnings and failures", `Stats` has no last-flush time, buffered
count or failing flag, and nothing says how much acknowledged data is currently only in memory. Reproduced on the in-memory
file system (580 appends acknowledged with durable head 1, Logger empty) and, by a second voter, on real files with
`chmod 0500` (`Err()` still EACCES after the fault cleared and a clean flush).

**Why held.** The minimum fix writes new log lines (a changed output); the fuller one adds exported `Stats` fields, which is
an API addition.

**Recommended answer.** Do the minimum now: log once on each transition into and out of a failing state (flush write,
publish, rotate, prune) at Warn and Info through the existing Logger. Add `Stats.LastFlushOK`, `Stats.BufferedRecords` and
`BufferedBytes` and `Stats.FlushFailing` before the first release: adding them later is only additive, but dashboards will
be built on `Err()` in the meantime. Decide with A5-10.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: log each transition (Warn on entering, Info on leaving, never per retry) and add the four `Stats` fields, building on A5-10.

**Disposition.** Fixed in 99b140d on the third attempt (on main; verified from `git log`; authored as 27ccd31 on branch `stage2/A5-1-a3`); fix-skeptic FIXED-VERIFIED for the approved scope, with follow-ups that were closed before landing (reported, from a skeptic that ran everything on scratch copies, on macOS and on the Linux box). The first two attempts (574b078 and bab952f) are not merged. bab952f was rejected for one documentation defect: `LastFlushOK`'s docs said that `time.Since(LastFlushOK)` with `BufferedRecords` above zero is how long the oldest acknowledged record has waited, which false-alarms on a quiet log (the skeptic's reproduction: `BufferedRecords` 1, `FlushFailing` false, `time.Since(LastFlushOK)` 1h0m0s). The third attempt: through `Config.Logger` each retried background failure (flush write, marker publish, rotation, prune unlink) logs one Warn on entry (with `op` and `err`) and one Info on recovery, never per retry (the skeptic: 50 failing retries gave exactly one Warn each); `Stats` gains `BufferedRecords`, `BufferedBytes`, `FlushFailing` and `LastFlushOK`, additive only. `LastFlushOK` is documented, and now stamped, as the time of the last flush that wrote and published successfully (the time the writer opened the log before the first). The documented alert is `FlushFailing`, or `BufferedRecords` above zero together with it, and `Err()` wrapping `ErrFailed`. Verified by the skeptic on real files (RLIMIT_FSIZE) with a fake clock: over 50 failing flushes `FlushFailing` is true with `BufferedRecords` 4 and `BufferedBytes` 119; after the fault clears and one flush succeeds they are false, 0 and 0; an idle log given one `Append` after an hour does not trip the alert. The secondary edge was real (a prune pass whose pre-delete publish failed logged a false "recovered" for the unlink) and is fixed, with a test the skeptic showed red on the old code. Decisions 60, 66 and 78, scope.md, the user guide, the doc comments and CHANGELOG are amended. Two gaps the skeptic raised were closed before landing: a failed log (after a data fsync error) keeps its buffered records without setting `FlushFailing`, so the docs now also say to alert on `Err()` wrapping `ErrFailed`; and a test that a retried rotation on an idle log does not stamp `LastFlushOK`, which fails when the guard in `flushLocked` is removed (verified by the recorder). Left as is, low severity: a prune pass that finds nothing to delete after a failed unlink logs "recovered" and clears `Err()`, consistent with A5-10 though the unlink was never retried; an idle log whose only failure was a marker publish after `Roll` advances `LastFlushOK` when it lands although no record was written (the doc says "wrote"). Not logged: a failed log itself, outside the retryable scope of the approval. Gate green on macOS and Linux on 27ccd31.

## A5-36: `quill prune` exits 0 when it did less than asked (medium)

**Claim.** An unlink failure, a corrupt consumer file and the clock guard each leave `quill prune` printing a counts line
(`6 -> 6`, `10 -> 7`) and exiting 0 with empty stderr. Reproduced with the built CLI by both voters for the unlink failure
(`chflags uchg`) and the clock guard, and for the corrupt consumer file by one.

**Why held.** The fix adds stderr output to a verb and makes `quill prune` exit 1 in a case where it exits 0 today, and
changes the library's `Prune` to return the unlink error instead of nil.

**Recommended answer.** Widen the offline verbs' stderr filter to the three messages the pass can produce ("cannot delete
segment", "age sweep limited by the clock guard", "age sweep stopped at a segment stamped in the future") and add the
broken-consumer Warn (A5-17); have `cmdPrune` exit 1 when `Err()` is non-nil after the pass; have `Prune` return the unlink
error. The clock guard stays a warning with exit 0, since `-force` is the documented answer to it.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: three messages through the offline filter, the broken-consumer Warn, exit 1 on an unlink failure, library `Prune` returns the error, the clock guard stays a warning with exit 0.

**Disposition.** Fixed in 2cbfdf2 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI on macOS and on the Linux box). `quill prune` now says why it did less than asked and exits 1 when an unlink failed. The offline stderr filter (`lossOrShortfall`, cmd/quill/write.go:87) passes "cannot delete segment", the clock-guard message and the future-stamp message. A new Warn, "consumer file could not be read", names each unreadable consumer file (or an unlistable `consumers/` directory) on every pass (retention.go:133, `brokenWhy`). Library `Prune` returns the unlink error instead of nil, and `cmdPrune` exits 1 when `Prune` errs. The clock guard stays a warning with exit 0. Residuals, reported by the skeptic. (1) The skeptic read the new `Prune` doc ("the same error is in `Err()`") as untrue when a sealed-segment load error is latched, because `Err()` then returned the load error first. That was the A5-10 masking; A5-10 (91c2fce) is an ancestor of this commit and `Err()` now joins the load error with the unlink error (verified by reading log.go:634), so the objection no longer holds on this tree. (2) The unlink-failure CLI test is darwin-only; Linux coverage is the library test with an injected fault plus the skeptic's manual `chattr +i` run, which is not committed. (3) The consumer Warn fires on every pass, flusher passes included: approved, but noisy for a library user whose consumer file stays broken. (4) The other writable verbs (`roll`, `append`, `recover`) now print the new Warns on stderr when their open path hits an unlink failure, and still exit 0; outside the approved scope.

## A5-18: a consumer that fell out of retention reads healthy (medium)

**Claim.** A consumer below the first retained sequence, whether unprotected, overridden by a cap or stale under
`ConsumerTTL`, shows in `quill consumers` a lag that counts deleted records, `protected` true and an empty NOTE, and `quill
check` prints HEALTHY. `Dropped` and `StaleConsumers` are never printed by the CLI and `Dropped` is 0 after a writer
restart. Reproduced by both voters with the built CLI (lag 80 with 16 records left; `Dropped` 8 then 0 after restart).

**Why held.** The fix changes CLI output and the `ConsumerInfo` shape (a new field or note) and may change what `Lag`
counts. Decisions 12 and 66 deliberately keep the counters in writer memory.

**Recommended answer.** Derive, do not persist. `quill consumers` NOTE and a JSON field when the offset is below the first
retained sequence; compute staleness for protected consumers from the policy's `ConsumerTTL` and the file mtime; log Warn
with the consumer's name when a deletion passes a consumer's offset; print `dropped` in `quill prune`. This follows the
precedent of decision 36, which added the "ahead of the head" note.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: derive, do not persist (decisions 12 and 66 stand).

**Disposition.** Fixed in 7b87e64 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI against real logs on macOS and the new tests on the Linux box). Nothing is persisted. `quill consumers` adds a NOTE ("behind the first retained sequence N: records a to b are gone") and `-json` `behind_first` and `stale`, each present only when true, so existing parsers see no change. Staleness for a protected consumer is derived from the policy's `ConsumerTTL` and the consumer file's mtime (the CLI repeats decision 45's rule in a helper of about 15 lines, because the library exports no way to evaluate it and no new exported symbol was allowed). `quill check` warns of a consumer behind the first retained sequence and stays HEALTHY, which goes beyond the literal approved answer and follows decision 36's "ahead of the head" note. A deletion past a consumer's offset logs a Warn naming the consumer (keys `consumer`, `offset`, `segment`, `last`), the offline verbs print that line, and `quill prune` prints `, dropped N`. The only library change is internal: `Log.offsets` and `consumerView.offsets` carry `{name, offset}` instead of a bare offset. Residuals, reported by the skeptic. (1) The deletion Warn inherits the A3-27 lag that decision 12 documents: seal-time caps judge consumers by the offsets of the last prune pass, so a consumer created after the writer started and before the first pass gets no Warn and no `Dropped` count for earlier cap deletions (the CLI still flags it from the files). (2) The CLI repeats decision 45's staleness rule and no test cross-checks it against the library, so the two can drift. (3) An abandoned consumer below `First` gets a Warn for every later deletion: noisy, but what the owner approved. (4) Inferred, not run: `check` derives the first retained sequence from the first good segment and `consumers` from `Stat().First`, so they could word a damaged-oldest-segment case differently; that log is already UNHEALTHY.

## A5-22: `MinFreeBytes` fails open when statfs fails (medium)

**Claim.** On a statfs error `capsLocked` logs one Warn and leaves `diskFull` as it was, so the floor is unenforced, or a
frozen `diskFull` keeps every append failing with ENOSPC after space returns, and `Err()` is nil in every state; `diskFull`
is in no `Stats` field and its clearing is not logged. Reproduced on the statfs seam by one voter; the other traced the
code.

**Why held.** The fix sets `Err()` in a new state and adds `Stats` fields.

**Recommended answer.** Set the background error when statfs fails or `diskFull` is set, log when `diskFull` clears, add
`Stats.DiskFull` and `Stats.FreeBytesAtLastPass`; consider treating the floor as unmet only after N consecutive statfs
failures rather than on the first. Fold into the `Err()` decision (A5-10).

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended except that the option of waiting for N consecutive statfs failures was declined: one failure is the trigger, and the documents say so.

**Disposition.** Fixed in 1dda449 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). A failing statfs, or a set `diskFull`, now sets a background error (the `bgDisk` slot, set in `capsLocked`, retention.go:246), so `Err()` reports it; `Stats` gains `DiskFull` and `FreeBytesAtLastPass` (stat.go:36-40); the clearing of `diskFull` is logged at Info. A later statfs success recomputes the state and clears both `diskFull` and the slot, so a frozen `diskFull` cannot outlive a pass that reads room. An earlier attempt was rejected for a broken table row in decision 25; the recorded attempt amends it inside the Why cell (reported by the fix summary). Accepted limit, stated in the documents: while statfs fails persistently and `diskFull` is set, `diskFull` stays frozen, appends keep failing with ENOSPC, and `Err()` reports both the ENOSPC and the statfs error until a pass can read room. That is the owner's "one failure is the trigger, no N-failure rule". The skeptic did not run an injected statfs failure on the Linux box (the seam is internal); the new tests passed there.

## A5-29: `quill archive` and source loss (medium)

**Claim.** Archive reports a source gap once, on stderr, exit 0; the next run is silent while the destination is UNHEALTHY.
After `quill unregister -name _archive` (the documented reset) Archive prints nothing and exits 0 while the destination gets
the same hole, because it never compares the committed offset with the destination's `quill.tail`. Reproduced by both voters
with the built CLI. The user guide's "the verb says so" is false for the `unregister` case.

**Why held.** The fix changes an exit status and Archive's documented return. The exit-0 half was ruled deliberate in round 1
(A1-58, decisions 85 and 53); the silent-after-`unregister` half is new. One voter would not rule out that an explicit
reset counts as acknowledging the loss.

**Recommended answer.** Return a distinct non-zero exit (the unhealthy code, 3) for a gap, and have Archive compare the
committed offset with the destination's `quill.tail` so the `unregister` case returns the `*GapError` too. If the owner keeps
exit 0, correct the user guide sentence for the `unregister` case now.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: exit 3 for a source gap, Archive compares the destination's tail, reversing the round-1 exit-0 ruling (A1-58, decisions 85 and 53).

**Disposition.** Fixed in caad4c5 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI on macOS and the archive tests on the Linux box). `quill archive` exits 3, the code `check` uses for UNHEALTHY, when the source had lost records the archive never copied (cmd/quill/consumers.go:209 at the fix commit), and `Archive` compares the destination's `quill.tail` with the first segment it copies (archive.go:80-85 and 105-107 at the fix commit), so after `quill unregister -name _archive` it returns the `*GapError` instead of leaving a silent hole. It builds on A3-14's destination check (303b794, an ancestor). The run that reports a gap still commits past it, so a later run with nothing to copy exits 0; `quill check -dir <dest>` keeps reporting the hole. Residuals, reported by the skeptic, matching the approved answer. (1) A gap is reported only by the run that finds it, because that run commits past the hole; a later run with nothing to copy exits 0 while `check -dir <dest>` still exits 3. (2) The destination comparison runs only when a sealed segment is copied (archive.go:105 at the fix commit), so after unregister a loss ahead of a still-active first retained segment is reported on the first run after that segment seals; the offset path (archive.go:74) reports at once (inferred from reading, with a probe). (3) After a manual forward `quill commit -name _archive -offset N` the message says "the source lost records" although those segments may still be in the source; the archive does have the hole, so exit 3 is right and only the wording is loose.

## A5-3: waiting on a failed log never fails (low)

**Claim.** After an fsync error `Log.ReadWait`, `Consumer.ReadWait` and `Log.Follow` block until the context ends and return
only the context error, while `Append`, `Flush` and `Close` return `ErrFailed`. Reproduced by both voters on the in-memory
file system.

**Why held.** The fix changes documented `ReadWait` behaviour: it would return an error where it returned a context error.
A caller may prefer to keep draining.

**Recommended answer.** In `Log.wait` return the failed error when `l.failed != nil` and the head is below the offset; reads
of existing records stay valid. Document it on `ReadWait` and `Follow`.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: `ErrFailed` from the wait when the head is below the offset; reads of existing records stay valid.

**Disposition.** Fixed in 1e0a8ed (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). `Log.wait` (read.go:358) returns `l.failed` when the head is below the offset, and `fail()` now broadcasts, so a waiter already parked when the log fails is woken with the error. The broadcast in `fail()` answers the skeptic's objection to the first draft: `Roll` and the retention directory-fsync failure call `fail()` with no flush to broadcast after it, so the check in `wait()` alone left a parked waiter hanging. Four `broadcastLocked` calls that followed a `fail()` became redundant and were removed. `Log.ReadWait`, `Consumer.ReadWait`, `Log.Follow` and `Consumer.Follow` all go through `Log.wait`; `Reader.wait` is deliberately unchanged because a Reader runs in another process and cannot know the writer failed. Reads of records that exist still succeed. Residuals, reported by the skeptic. (1) scope.md:90 still says `Follow` waits until its context is cancelled, without the failed-log exception; the table row at scope.md:62 states it, so the prose is incomplete, not false. (2) A waiter parked on an idle log whose `quill.lock` was removed learns of it only at the next flush or prune: pre-existing, not A5-3. (3) The skeptic could not run staticcheck or deadcode (local binaries mismatched the toolchain and the pinned versions need the network); the A5-32 skeptic's later run on the tip covers both (see the update at the top).

## A5-9: `quill prune` and `quill roll` discard `Close`'s error (low)

**Claim.** `withWriter` (cmd/quill/write.go:85) discards the error of `Close`, which carries the final directory fsync, so
a fatal fsync failure after a successful prune or roll exits 0; `append` and `recover` already report it (decision 100).
Reproduced once, with an fsync error injected into a patched copy of the CLI (the shipped binary has no fault hook).

**Why held.** It changes an exit code.

**Recommended answer.** Report `Close` in `withWriter` as `cmdAppend` and `cmdRecover` do and exit 1.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: report `Close`'s error as `cmdAppend` and `cmdRecover` do (decision 100) and exit 1.

**Disposition.** Fixed in 46f30db (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). `withWriter` (cmd/quill/write.go:104, used by `quill prune` and `quill roll`) calls `Close` inline instead of deferring it and, when `fn` succeeded, prints `Close`'s error on stderr and exits 1; when `fn` failed its own exit code stands. `prune` has printed its counts line before `Close` runs, so the summary still appears. Residuals, reported by the skeptic. `withWriter` no longer defers `Close`, so a panic inside `fn` would skip it, which is moot because the process exits and the lock goes with it. The test makes the directory read-only inside `fn`, so it proves the publication-of-`quill.tail` path only, not the directory fsync or the lock release; all three reach the same `Close` return, the only thing the fix changes. The skeptic did not run deadcode (the commit adds no symbol).

## A5-23: the published disk bound is wrong for `MaxBufferBytes` below `FlushBytes` (low)

**Claim.** The bound in scope.md and decision 69 uses `MaxBufferBytes` for the flush term, but the code (`maxActiveBytes`,
log.go:203) uses the larger of it and `FlushBytes`, and `validate` accepts a smaller `MaxBufferBytes`. Reproduced by both
voters: with `MaxBufferBytes` 1 the on-disk peak was 4,200,444 bytes against a published bound of 3,949,577.

**Why held.** Either the guarantee's wording or `Open`'s acceptance of a Config changes.

**Recommended answer.** Refuse `MaxBufferBytes < FlushBytes` in `validate` with `ErrConfig`: one line, nobody has a reason to
set it, and the published formula becomes exact. The runner-up, rewording the formula with the maximum, loses because it
keeps a Config that cannot do what its name says.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: `validate` refuses `MaxBufferBytes < FlushBytes` with `ErrConfig`, so the published bound is exact.

**Disposition.** Fixed in 4d0e8c9 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). `Config.validate` refuses `MaxBufferBytes < FlushBytes` with `ErrConfig` (log.go:180, verified), placed after the negative-value case so a wrapped default is still reported as negative; equal values are accepted. `max(FlushBytes, MaxBufferBytes)` stays in `maxActiveBytes` and in the ceiling text: it is now always `MaxBufferBytes`, and changing it would touch decision 60, the user guide and CHANGELOG for no behaviour change (the fixer's choice, stated in decision 69's note). Residual: `max(FlushBytes, MaxBufferBytes)` at log.go:216 (at the fix commit) is now redundant but true; a later reader may wonder why it stays, and decision 69's note says. The skeptic did not re-run staticcheck or deadcode; the A5-32 skeptic's later run on the tip covers both (see the update at the top).

## A5-32: an archive rerun keeps a same-size damaged destination segment (low)

**Claim.** archive.go:178 returns success for a destination segment that exists with the same size, without reading it, so a
damaged copy survives a rewound rerun, the run exits 0, and only `quill check -dir dst` finds it. Reproduced by one voter
with the built CLI.

**Why held.** It changes documented behaviour (the rerun is idempotent and cheap because it skips).

**Recommended answer.** Verify the CRC of a same-size destination file and replace it when it fails, or require a flag for a
forced re-copy. Failing both, document that a destination is audited with `quill check`, not by a rerun.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: read and CRC-verify a same-size destination segment and replace it when verification fails, no flag.

**Disposition.** Fixed in 16cb0c8 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI on macOS and on the Linux box). `copySegment` now calls `destIntact` (archive.go:250, verified) for a destination segment of the source's size: it must pass `verifySegment` (the CRC of every frame) and carry the source's first and last sequence, or it is replaced through the existing temp-file-and-rename. A healthy same-size file is left alone, not rewritten (its inode is unchanged). The sequence comparison is an addition to the approved answer: a valid-CRC segment of the same size from elsewhere would otherwise pass. A rewound run with a same-size destination segment that cannot be read fails with exit 1; decision 85 says so. Residuals, reported by the skeptic. (1) The first and last sequence branch of `destIntact` has no unit test; a mutation removing it would pass the new test (the skeptic exercised it only through the CLI). (2) The user guide does not mention the exit 1 for an unreadable same-size destination segment; decision 85 does. This skeptic ran staticcheck v0.8.2, deadcode v0.52.0 and the full `-race` suite, all clean, on 16cb0c8, the tip of the chain.

## A5-44: `quill ls` omits a log it cannot open (low)

**Claim.** cmd/quill/read.go:45-47 skips every `OpenReader` error, so a log with a corrupt `quill.format`, a future format or
no permission vanishes from the listing and `ls` exits 0, while `stat` and `check` on it exit 1. Reproduced by both voters.

**Why held.** It changes `ls` output and possibly its exit code.

**Recommended answer.** Skip only `errors.Is(err, quill.ErrNoLog)`; for anything else print an "error:" row as the `Stat`
failure branch does, and decide whether `ls` then exits 1.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: skip only `ErrNoLog`, print an "error:" row for anything else, list everything, and exit 1 afterwards as `stat` and `check` do for the same log.

**Disposition.** Fixed in 29e7db1 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI on macOS and the new tests on the Linux box). `cmdLs` (cmd/quill/read.go:34) skips a subdirectory only when `errors.Is(err, quill.ErrNoLog)`; any other open or `Stat` failure prints an "error:" row, the other logs are still listed, and `ls` exits 1. "no logs under" is printed only when nothing was listed and nothing failed. The `OpenReader` and `Stat` failures share the one existing error-row branch and a `failed` counter. Residuals, reported by the skeptic. (1) A directory with `quill.format` and no `quill.tail` is still `ErrNoLog` (reader.go:155), so `ls` omits it and exits 0 while `stat` exits 1 (reproduced); this follows the approved "skip only `ErrNoLog`" rule and the mid-creation design, and the user guide's "no quill.format" wording omits it. (2) Any unreadable non-log subdirectory, such as a root-only `lost+found`, now makes `ls` exit 1: a consequence of the approval. Staticcheck, deadcode and fuzz were not run by the skeptic.

## Not held, but worth the owner's eye

- **A5-33 (closed by a pass A3 commit).** The umask finding was closed by 1f896c7, which documents that the writer's
  segments and directories go through the umask. The finding's optional second half, a `quill check` warning when a segment's
  mode differs from the marker's, was not applied: it would flag every writer run under a tighter umask, which the
  documents now call normal. Apply it only as a new decision.
- **A5-5 (refuted; hardening remains).** A Reader's plain `Read` still serves a replacement log from the old cursor; `Commit`
  and a Reader's wait now return `ErrWrongLog` (495e457). If the owner wants plain reads to check too, it is one constant-cost
  read of `quill.format` per view, which decision 83 declined on cost.
- **A5-17 (prior-refuted).** The age sweep is still suspended silently by an unparseable consumer file; a Warn per pass naming
  it was recorded as "would close it (not done)" in round 1 and is part of the A5-36 recommendation above.
- **Residuals of the seven fixes** (reported by the fix-skeptics, in the record): A5-16 does not stop a second writer that
  opens before the first writer's next flush; A5-21 leaves a pending marker publish on a full disk blocking every deletion,
  which the skeptic flagged as a possible separate finding and did not test; A5-2 and A5-42 report only to a configured
  Logger in the library.
