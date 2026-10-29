# A1 adversarial audit: simplify

Date 2026-10-29, against HEAD b52b746 (the source is identical to 1f4152d; only docs/audits
changed in between). Dimension A (simplification) of the A1 audit: every candidate, each
skeptic's verdict and reason, and the recommended disposition. Every anchor below was re-read
from the tree at synthesis time; reproductions ran in throwaway copies under the session
scratchpad and no repository file was edited by the audit. Ranked by what the finding puts at
risk, then by lines saved.

Verdicts: 10 CONFIRMED, 1 PLAUSIBLE (simplify-7), 1 REFUTED (simplify-11). simplify-4 is the
same item as dedup-13; simplify-3 is absorbed by correctness-24 (same three parse sites, where
the correctness lens found the deletion consequence).

| # | id | anchor | severity | votes | verdict | recommendation |
|---|----|--------|----------|-------|---------|----------------|
| 1 | simplify-1 | reader.go:206 | BUG (semantic) | 2 | CONFIRMED | owner decision: approve the 1-line guard |
| 2 | simplify-12 | stat.go:215 | SIMPLIFY (semantic) | 2 | CONFIRMED | owner decision: `fatal` or `warn` |
| 3 | simplify-3 | cmd/quill/verbs.go:592 | SIMPLIFY (semantic) | 2 | CONFIRMED | owner decision, as part of correctness-24 |
| 4 | simplify-2 | cmd/quill/verbs.go:1 | SIMPLIFY | 1 | CONFIRMED | apply now |
| 5 | simplify-10 | cmd/quill/verbs.go:301 | SIMPLIFY | 1 | CONFIRMED | apply now |
| 6 | simplify-6 | stat.go:157 | SIMPLIFY | 1 | CONFIRMED | apply now |
| 7 | simplify-9 | archive.go:161 | SIMPLIFY | 1 | CONFIRMED | apply now, before dedup-17 |
| 8 | simplify-8 | cmd/quill/verbs.go:462 | SIMPLIFY | 1 | CONFIRMED | apply now |
| 9 | simplify-7 | cmd/quill/verbs.go:166 | DEDUP | 1 | PLAUSIBLE | apply now |
| 10 | simplify-4 | load.go:460 | DEDUP | 1 | CONFIRMED | apply now (= dedup-13) |
| 11 | simplify-5 | consumer.go:286 | DEDUP | 1 | CONFIRMED | apply now |
| 12 | simplify-11 | reader.go:121 | SIMPLIFY | 2 | REFUTED | not changed: the retry is specified |

## 1. simplify-1: the Reader panics when a sealed segment's first frame decodes with seq 0 (CONFIRMED, semantic)

**Anchors.** reader.go:204-206: `rec, _, ferr := w.frameAt(0); if ferr != nil || rec.Seq == 0 {
s.bad = &SegmentError{Path: path, Reason: "first frame: " + reason(ferr)} }`. `reason` at
load.go:256-264 has cases for `errTornFrame` and `errBadFrame` and then falls through to
`err.Error()`, so a CRC-valid frame carrying seq 0 reaches `reason(nil)` and dereferences a
nil interface. The writer-side walk rejects seq 0 at load.go:175 (`!havePrev && rec.Seq == 0`),
but `segment()` calls `frameAt` directly and never passes through it; the active-segment path
at reader.go:289-292 handles the same case without calling `reason`.

**Skeptics (2, both CONFIRMED).** Skeptic 1 reproduced it in a copy (Open, Append, Roll, Append,
Close; overwrite `0000000000.log` with one CRC-valid frame of seq 0; OpenReader; `Read(1,10)`
panics: `reason({0x0,0x0}) load.go:263 <- segment reader.go:206 <- view reader.go:272 <- Read
reader.go:148`) and found no test covering a CRC-valid seq-0 first frame on the Reader side
(load_test.go:145-149 covers `walk`, reader_test.go:217-224 a CRC-failing frame). Skeptic 2
ran the same probe, applied the guard in a copy, and found the full `GOWORK=off go test ./...`
green with the probe now returning `quill: records 1 to 1 are unavailable` (a `GapError`), which
is what decisions 11 and 38 and scope.md guarantee 14 (scope.md:354) require; decision 71
("the first frame of a segment must be at least 1") makes "bad frame" the right reason text.

**Why owner decision.** A Reader's observable result changes (panic to `GapError`) and the
finder flagged it semantic; the fix itself is one line and restores documented behaviour.
Recommended answer: approve. Fix: after reader.go:204 insert `if ferr == nil && rec.Seq == 0 {
ferr = errBadFrame }`. Test: a copy of the probe as `TestReaderReportsASeqZeroFirstFrameAsAHole`
in reader_test.go asserting `errors.As(err, &GapError)` with `From == 1` and no panic.

## 2. simplify-12: `mustList` hides a consumers/ listing failure from `Check` (CONFIRMED, semantic)

**Anchors.** stat.go:215-218 `func mustList(st *consumerStore) []ConsumerInfo { list, _ :=
st.list(); return list }`, sole caller stat.go:192. `list()` at consumer.go:230-234 returns the
`ReadDir` error unchanged. `Healthy` is cleared only from a `Fatal` problem (stat.go:207-211),
so a missing or unreadable consumers/ yields HEALTHY; retention treats the same condition as
`broken` and protects everything (retention.go:31-35).

**Skeptics (2, both CONFIRMED).** Skeptic 1 reproduced both the missing-directory and the
mode-000 case on the real OS backend (`retention: broken=true` against `check: healthy=true
problems=[]`) and noted decisions 62/81 make consumers/ exist before `quill.tail`, so a Reader
that passed OpenReader never sees a legitimately absent consumers/, and `fatal` cannot misfire
on a mid-creation log. Skeptic 2 reproduced the same with `fsx.Sim` injecting `fs.ErrPermission`
on `OpReadDir`, applied the inlined form in a copy (gofmt, vet, staticcheck v0.8.2 and the full
suite clean, stat.go 243 lines), and corrected the finder's citation: the governing text is
decision 67 and build-prompt.md:218/253 (an unreadable consumer file makes `check` UNHEALTHY),
not decision 44, which favours `fatal` over `warn`.

**Why owner decision.** `quill check` would start reporting UNHEALTHY (exit 3) for a condition
it reports HEALTHY today, and the finder left `fatal` versus `warn` open. Recommended answer:
`fatal`, matching retention's `broken` and decision 67. Fix: replace stat.go:192 with `list, err
:= r.consumerStore().list(); if err != nil { fatal(Problem{Kind: "consumer", Path:
r.path(consumersDir), Detail: err.Error()}) }; for _, c := range list {`, delete `mustList`.
Test: the two skeptic tests (missing dir on the OS backend; `OpReadDir` injection on the Sim)
asserting `Healthy == false` and one `consumer` problem.

## 3. simplify-3: `fmt.Sscanf("%d")` accepts trailing garbage at three sites (CONFIRMED, semantic; see correctness-24)

**Anchors.** cmd/quill/verbs.go:592 (`commit -offset`), :700 (`prune -max-bytes`,
`-min-free-bytes`, `-max-records`), :778 (`@unix` in `parseTime`). None checks the remainder.

**Skeptics (2, both CONFIRMED).** Skeptic 1 re-ran the comparison ("12abc" -> 12, "0x10" -> 0,
"1_000" -> 1, " 7" -> 7, all `err=nil`; `strconv` rejects each) and found no decision that makes
lax parsing deliberate; decision 75 requires digits-only decimals for state files, the opposite
attitude. Skeptic 2 applied the `strconv` fix in a copy (the `strconv` import must be added,
which the proposal omits) with `go vet` and `go test ./cmd/quill/` passing; cli_test.go uses
only well-formed values (:232, :342, :346, :361, :365).

**Disposition.** The same sites, with the deletion consequence, are correctness-24; this item
is closed by that owner decision and is not tracked separately.

## 4. simplify-2: cmd/quill/verbs.go is 786 lines, over the ~600 ceiling (CONFIRMED)

**Anchors.** `wc -l cmd/quill/verbs.go` = 786. Decision 20 (decisions.md:28), build-prompt.md:41,
CONTRIBUTING.md:19 and README.md:45 state the rule without a library-only qualifier. Function
boundaries from `grep -n '^func'`: `init` 65, `readFlags` 131, `findVerb` 139, `run` 149,
`cmdHelp` 185, `printVerbHelp` 214, `lockText` 221, `cmdReference` 232; `reader` 278, `writer`
287, `policyConfig` 301, `cmdLs` 309, `cmdStat` 352, `start` 374, `cmdRead`/`cmdCat` 396-397,
`printFrom` 399, `printRecords` 414, `cmdTail` 445, `cmdAppend` 473, `cmdConsumers` 511, `lag`
550, `consumer` 557, `cmdCommit` 568, `cmdRegister` 603, `cmdUnregister` 628, `cmdCheck` 643,
`cmdPrune` 680, `cmdRoll` 727, `cmdArchive` 739, `cmdRecover` 755, `parseTime` 772.

**Skeptic (CONFIRMED).** Confirmed the count, the rule's repo-wide scope (decision 73 reaffirms
the ceiling by splitting errors.go for the same reason) and the proposed sizes (verbs.go ~276,
read.go ~222, consumers.go ~131, write.go ~148, plus headers); corrected one sentence: the
consumer verbs call `c.reader()` (:512, :571, :607, :632), `parseTime` (:582) and `printJSON`
(print.go:76), so the groups do cross, but as same-package calls, so a move-only split compiles.
The reference-regeneration test (cli_test.go:53-69) keeps passing because the table stays in
verbs.go, which docs/reference.md:3 names. README.md:72 lists the cmd/quill files and needs
the two new names.

**Fix (apply now).** Keep verbs.go = lines 1-276 (exit codes, `verb`/`cli` types, `errorf`,
`usage`, `init` table, `readFlags`, `findVerb`, `run`, `cmdHelp`, `printVerbHelp`, `lockText`,
`cmdReference`); new read.go = `reader`, `cmdLs`, `cmdStat`, `start`, `cmdRead`, `printFrom`,
`printRecords`, `cmdTail`, `cmdCheck`, `parseTime`; new consumers.go = `cmdConsumers`, `lag`,
`consumer`, `cmdCommit`, `cmdRegister`, `cmdUnregister`, `cmdArchive` (it takes only the
consumer lock, verbs.go:739-753); new write.go = `writer`, `cmdAppend`, `cmdPrune`, `cmdRoll`,
`cmdRecover`. Run gofmt, `go vet ./cmd/quill`, `go test ./cmd/quill`; add read.go, consumers.go,
write.go to README.md:72. Land the other verbs.go edits (simplify-7, -8, -10, dedup-14,
docs-46 and, if approved, correctness-24) first so their anchors hold, or re-anchor them after.

## 5. simplify-10: `policyConfig` is a pass-through; `cmdCat` duplicates `cmdRead`; `printFrom`'s `follow` is always false (CONFIRMED)

**Anchors.** verbs.go:301-307 returns exactly what `quill.ReadPolicy` returned (state.go:290-298
returns `Config{}, false` on both failure paths); sole caller verbs.go:681. verbs.go:396-397 are
identical one-liners; the table entry at :75 is `run: cmdCat`. `printFrom(c, follow)` at :399 is
called only with `false`; `cmdTail` calls `printRecords` directly at :470.

**Skeptic (CONFIRMED).** Applied the three deletions in a copy: gofmt clean, `go vet`, full
`GOWORK=off go test ./...` green, verbs.go 786 -> 776. Decision 86's "`read` and `cat` are the
same printer with different defaults" is preserved because the verb entries, summaries and
flag sets stay separate; only the run pointer changes.

**Fix (apply now).** Delete `policyConfig` and write `cfg, ok := quill.ReadPolicy(c.dir)` at
:681; delete `cmdCat`, set `run: cmdRead` at :75; drop `follow` from `printFrom` and pass
`false` to `printRecords`. Test: existing `go test ./cmd/quill` (reference pin at
cli_test.go:53-69, cat/read tests at :126-182, :225).

## 6. simplify-6: `Check` duplicates a `verifyBounded` call, declares `io.ReaderAt` by hand, and derives `Healthy` in a trailing loop (CONFIRMED)

**Anchors.** stat.go:157-170: the two arms call `verifyBounded` with byte-identical clock
closures, differing only in the bound (`int64(tl.Off)` vs `size`). stat.go:223-225 declares
`f interface { ReadAt([]byte, int64) (int, error) }`, which is `io.ReaderAt`; `window.r` is
already `io.ReaderAt` (load.go:91). stat.go:207-211 sets `Healthy=false` from `Fatal` flags
that only the `fatal` closure at :131 sets (grep: the only other hits are the struct field and
this loop).

**Skeptic (CONFIRMED).** Applied the three edits in a copy (gofmt clean, stat.go 245 -> 235,
build, vet and the full suite green, including the Check tests at reader_test.go:241-301,
archive_test.go:133-138, retention_test.go:221, backlog_test.go:142 and the cmd/quill exit-3
test). docs/audits/self-review-recovery-and-reader.md:47 only checks that the marker's segment
is bounded by `tl.Off`, which the merged form keeps. `info, err :=` compiles once `var info
segInfo` at :156 is dropped.

**Fix (apply now).** `bound := size; if id == tl.Seg { if size < int64(tl.Off) { fatal(...); _ =
f.Close(); continue }; bound = int64(tl.Off) }; info, err := verifyBounded(f, bound, &prevAt,
future, func(seq uint64, at int64, what string) { warn(...) })`; `f io.ReaderAt` with `io`
imported; `fatal := func(p Problem) { p.Fatal = true; res.Healthy = false; res.Problems =
append(res.Problems, p) }` and delete :207-211. Test: existing suite.

## 7. simplify-9: `copySegment` hand-rolls the copy loop that `io.CopyBuffer` replaces (CONFIRMED)

**Anchors.** archive.go:161-174: `buf := make([]byte, windowChunk)` and a `ReadAt`/`WriteAt`
loop treating `io.EOF` with `pos < size` as an error.

**Skeptic (CONFIRMED).** Checked the toolchain actually used (go1.27.3): `*io.SectionReader`
and `*io.OffsetWriter` implement neither `WriterTo` nor `ReadFrom`, so `CopyBuffer` keeps the
1 MiB buffer and issues the same `ReadAt`/`WriteAt` sequence (count, size and offset) to the
`fsx.File`; the simulator's short write surfaces as `io.ErrShortWrite` from both forms. The only
divergence is `io.EOF` vs `io.ErrUnexpectedEOF` on a source shorter than `size`, practically
unreachable because `verifySegment(r, size)` at :139 already read the whole file, and nothing
matches `io.EOF` on this path. One nit: the proposed snippet placed after the `Chmod` at :160
must be wrapped in `if err == nil { ... }` to keep the current control flow.

**Fix (apply now).** Replace :161-174 with `if err == nil { n, cerr :=
io.CopyBuffer(io.NewOffsetWriter(f, 0), io.NewSectionReader(r, 0, size), make([]byte,
windowChunk)); if cerr == nil && n < size { cerr = io.ErrUnexpectedEOF }; err = cerr }`. Land
before dedup-17, which moves the result into a callback. Test: archive_test.go and the full
suite.

## 8. simplify-8: `cmdTail`'s start computation is seven lines for a one-line max/min (CONFIRMED)

**Anchors.** verbs.go:462-468 as quoted; `c.n`, `st.Head`, `st.First` are all `uint64`
(verbs.go:45, stat.go:12-13); the `min`/`max` builtins are already used (go.mod `go 1.27`).

**Skeptic (CONFIRMED).** Proved equivalence by a throwaway program over all triples of 10 edge
values plus 5,000,000 random triples (0 mismatches) and by `go test ./cmd/quill/` on a copy;
corrected the proposal's claim about coverage: both tail tests (cli_test.go:197, :227) are
`n <= Head`, so `n > Head` and `-n 0` are not exercised today.

**Fix (apply now).** `from = max(st.Head+1-min(c.n, st.Head), st.First)`. Test: add tail cases
for `-n` greater than Head (expect from 1, then `max(.., First)`) and `-n 0` (expect Head+1).

## 9. simplify-7: the per-verb FlagSet is built three times (PLAUSIBLE)

**Anchors.** verbs.go:166-171 (`run`, six lines, includes `fs.SetOutput(io.Discard)` at :167),
:191-195 (`cmdHelp`), :253-257 (`cmdReference`). The `-dir` default is `c.dir` at :168 and `""`
at :192 and :254.

**Skeptic (PLAUSIBLE).** The dedup is real and the helper is behaviourally equivalent, but the
claim's "the only difference is the -dir default" and "3 x 5 lines" are wrong at :167: `run`
also sets the output to `io.Discard`. Adding `SetOutput(io.Discard)` to all three is safe:
`printVerbHelp` re-sets the output to `c.out` at :217 before `PrintDefaults`, and `cmdReference`
only walks the set via `VisitAll`/`Lookup` (:259-265). The reference test
(cli_test.go:53-69, not 60-68) and `TestUsageErrorsExitTwoAndHelpWorks` (:88-91) pin the
output.

**Fix (apply now).** `func (v *verb) flagSet(c *cli, dirDefault string) *flag.FlagSet` doing
`NewFlagSet`, `SetOutput(io.Discard)`, the `-dir` registration and `v.flags(fs, c)`; call
`v.flagSet(c, c.dir)` in `run` and `v.flagSet(c, "")` in `cmdHelp` and `cmdReference`. Test:
the existing reference pin and help tests.

## 10. simplify-4 (= dedup-13): `openActive` is `openSegment` with one flag changed (CONFIRMED)

**Anchors.** load.go:60-84 and :461-485 differ only at :68 `fsx.O_RDONLY` vs :469 `fsx.O_RDWR`
(verified by diff). `openActive` has one caller (load.go:318); `openSegment` has seven non-test
callers (load.go:441, read.go:275, reader.go:198, :232, :280, :344, stat.go:151) plus
load_test.go:304-338. The doc comment at :460 already says "with the checks of openSegment".

**Skeptic (CONFIRMED).** Diffed the bodies, counted the callers, and found nothing in
decisions.md, build-prompt.md or docs/audits forbidding a flag-taking helper; `fsx.FS.OpenFile`
already takes the flag (internal/fsx/fs.go:21). load.go is 485 lines, so the wrapper fits.

**Fix (apply now).** `func openSegmentFlag(fsys fsx.FS, path string, flag int) (fsx.File,
int64, error)` holding the body; `openSegment` and `openActive` become one-line wrappers so
all call sites and load_test.go stay untouched. Test: load_test.go:304-338 (symlink, FIFO,
oversize, missing) and the recovery tests that exercise `openActive` through `Open`.

## 11. simplify-5: `consumerStore` duplicates the `first` closure for Log and Reader (CONFIRMED)

**Anchors.** consumer.go:286-293 and :314-321 differ only in the receiver (`l.view()` vs
`r.view()`); `head` genuinely differs (:285 wraps `l.Head()`, :313 uses `r.Head`). Both
receivers satisfy `source` (read.go:38-45), proven by the existing `open(l, ...)` at
consumer.go:299 and log.go:310 and `open(r, ...)` at consumer.go:326 and archive.go:35. `open`
(consumer.go:62) receives the same value again as `src`.

**Skeptic (CONFIRMED).** Confirmed the duplication and that retention.go:30, flush.go:134 and
stat.go:192 only call `list()`; corrected the caller count: six `open()` callers, not four
(consumer.go:299, :326, archive.go:35, log.go:310, consumer_test.go:185, :188).

**Fix (apply now).** Add `src source` to `consumerStore`, make `first` a method over
`st.src.view()`, keep `head` as the differing func, drop `open`'s `src` parameter and update the
six callers. Coordinate with dedup-18, which restructures `open`'s body. Test: consumer_test.go
and the full suite.

## 12. simplify-11: `readTail` retries every error five times with sleeps (REFUTED)

**Anchors.** reader.go:121-139 as quoted: five attempts, 1 ms sleep, `fs.ErrNotExist` returned
at once as `ErrNoLog`, decode failures wrapped as `ErrCorrupt` and retried.

**Skeptics (1 REFUTED, 1 PLAUSIBLE).** The refuting skeptic found the loop specified:
build-prompt.md:184-185, which the task names as part of the specification, says of the Reader
"Each call reads `quill.tail` (verify it parses; on a transient failure retry briefly)", so the
finding's premise "undocumented intent" is false and removing the loop would contradict the
spec; the `ErrNotExist` exclusion is decision 62's caller-side retry (reader_test.go:369-378).
The PLAUSIBLE skeptic agreed the code description is accurate, that no concrete transient
failure is demonstrable on linux/darwin with atomic rename, that the full suite (with `-race`)
passes with the loop removed, and that the only cost is ~5 ms before a persistent
`ErrCorrupt`/EACCES report; it concurred that the build prompt makes deletion the wrong option.

**Disposition.** Not changed. Optional, not tracked: a decision row pointing at
build-prompt.md:184 would make the rationale findable from the code.
