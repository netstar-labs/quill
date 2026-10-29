# A1 adversarial audit: simplify

Date 2026-10-29, against commit 1f4152d. Dimension A (simplification) of the A1 audit:
every candidate the finder raised, each skeptic's verdict and reason, and the fix the
synthesis recommends. Line anchors were re-read at synthesis time against the tree at
1f4152d. Ranked by severity, then by what the fix removes.

Verdicts: 10 CONFIRMED, 1 PLAUSIBLE, 1 REFUTED. The refuted item stays below; it is a result.

| # | id | anchor | severity | verdict | recommendation |
|---|----|--------|----------|---------|----------------|
| 1 | simplify-1 | reader.go:206 | BUG | CONFIRMED | apply now |
| 2 | simplify-12 | stat.go:215 | SIMPLIFY (semantic) | CONFIRMED | apply now |
| 3 | simplify-3 | cmd/quill/verbs.go:592 | SIMPLIFY (semantic) | CONFIRMED | apply now, merged into correctness-24 |
| 4 | simplify-2 | cmd/quill/verbs.go:1 | SIMPLIFY (hard rule) | CONFIRMED | apply now, resolves docs-37 |
| 5 | simplify-10 | cmd/quill/verbs.go:301 | SIMPLIFY | CONFIRMED | apply now |
| 6 | simplify-6 | stat.go:157 | SIMPLIFY | CONFIRMED | apply now |
| 7 | simplify-4 | load.go:460 | DEDUP | CONFIRMED | apply now, same item as dedup-13 |
| 8 | simplify-5 | consumer.go:286 | DEDUP | CONFIRMED | apply now |
| 9 | simplify-7 | cmd/quill/verbs.go:166 | DEDUP | PLAUSIBLE | apply now |
| 10 | simplify-8 | cmd/quill/verbs.go:462 | SIMPLIFY | CONFIRMED | apply now |
| 11 | simplify-9 | archive.go:161 | SIMPLIFY | CONFIRMED | apply now, before dedup-17 |
| 12 | simplify-11 | reader.go:121 | SIMPLIFY | REFUTED | not changed |

## 1. simplify-1: Reader panics when a sealed segment's first frame decodes with seq 0 (BUG, CONFIRMED)

**Anchor.** reader.go:204-206:

```go
rec, _, ferr := w.frameAt(0)
if ferr != nil || rec.Seq == 0 {
    s.bad = &SegmentError{Path: path, Reason: "first frame: " + reason(ferr)}
```

**Claim.** A CRC-valid frame carrying `seq 0` leaves `ferr == nil`, and `reason(nil)` at
load.go:263 falls through to `err.Error()` on a nil interface. The writer-side walk
(load.go:175) rejects seq 0 as `errBadFrame`; `segment()` calls `frameAt` directly and has no
such guard. The active-segment path at reader.go:290-292 handles the case without calling
`reason()`.

**Skeptics (2, both CONFIRMED).** Both re-ran the probe (Open, Append, Roll, Append, Close;
overwrite `0000000000.log` with one CRC-valid seq-0 frame; `OpenReader`; `Read(1, 10)`) and got
the nil dereference at `reason load.go:263 <- segment reader.go:206 <- view reader.go:272`.
Nothing below `frameAt` rejects sequence 0 (load.go:134-157, codec.go:75-121); no test covers
the sealed-segment Reader case (load_test.go:145-149 is writer-side, reader_test.go:217-224 is a
CRC failure). With `if ferr == nil && rec.Seq == 0 { ferr = errBadFrame }` inserted before line
205 the suite stays green and the probe returns `GapError{1,1}`, which guarantees 6 and 14 and
decisions 11, 38 and 71 require.

**Fix.** The one-line mapping above, so the reason reads "first frame: bad frame" (decision 71:
the first frame of a segment must be at least 1). Add the probe as a test in reader_test.go.

## 2. simplify-12: mustList hides a consumers/ listing failure from Check (CONFIRMED, semantic)

**Anchor.** stat.go:215-218 `func mustList(st *consumerStore) []ConsumerInfo { list, _ := st.list(); return list }`,
sole caller stat.go:192.

**Claim.** A missing or unreadable `consumers/` makes `Check` report HEALTHY, while retention
treats the same condition as `broken` and protects everything (retention.go:31-35).

**Skeptics (2, both CONFIRMED).** Both reproduced the divergence on the real OS backend
(`os.RemoveAll(dir/consumers)` and mode 000) and under the simulator (`fs.ErrPermission` on
`OpReadDir`): retention `broken=true`, `Check` `healthy=true problems=[]`. Decision 67 and
build-prompt.md:218/253 say an unreadable consumer file makes `check` UNHEALTHY, and `Check`
already returns the error for a failed `ReadDir` of the log directory (stat.go:138-141), so the
swallow is an inconsistency. Decisions 62/81 create `consumers/` before `quill.tail` (log.go:389
before :433), so the proposed `fatal` cannot misfire on a mid-creation log. The finding's
"decision 44" citation is the clock guard; decision 67 governs. The fix passed gofmt, vet,
staticcheck v0.8.2 and the full suite in a scratch copy.

**Fix.** Inline `list, err := r.consumerStore().list(); if err != nil { fatal(Problem{Kind: "consumer", Path: r.path(consumersDir), Detail: err.Error()}) }`
and delete `mustList`. `fatal`, not `warn`: decision 67 and build-prompt.md:218 class an
unreadable consumer file as UNHEALTHY, and an unreadable directory is the stronger case.

## 3. simplify-3: fmt.Sscanf("%d") silently accepts malformed numbers (CONFIRMED, semantic)

**Anchors.** cmd/quill/verbs.go:592 (`commit -offset`), :700 (`prune -max-bytes/-min-free-bytes/-max-records`), :778 (`@unix` times).

**Skeptics (2, both CONFIRMED).** Both re-ran the comparison: `"12abc"` -> 12, `"0x10"` -> 0,
`"1_000"` -> 1, `" 7"` -> 7, all `err == nil`; `strconv` rejects every one. No test uses a
malformed value (cli_test.go:342, 346, 361, 365 are well-formed); decision 86 fixes the shapes
but not the number syntax, and decision 75 argues for canonical digits-only numbers in state
files, so laxness here contradicts the project's stated attitude. The fix needs a `strconv`
import, which the proposal omits; applied in a copy, `go vet` and `go test ./cmd/quill/` pass.

**Disposition.** The same three sites are correctness-24 (BUG: `-max-records 1e6` deletes
every sealed segment; `-offset 1e3` rewinds to 1). One change fixes both; see
audit-correctness.md item 4 for the fix and test plan.

## 4. simplify-2: cmd/quill/verbs.go is 786 lines, over the ~600 ceiling (CONFIRMED)

**Anchor.** `wc -l cmd/quill/verbs.go` = 786 at synthesis time. Decision 20 (decisions.md:28)
and build-prompt.md:41 state the ceiling without a library-only qualifier; decision 73 reaffirms it.

**Skeptic (CONFIRMED).** Function boundaries match the claim to within a line (doc comments at
300 and 771 precede `policyConfig` at 301 and `parseTime` at 772). One sentence of the claim is
false but immaterial: the consumer verbs call `c.reader()` (verbs.go:512, 571, 607, 632) and
`parseTime` (:582), and several verbs call `printJSON` (print.go:76), so the groups do cross,
but all are same-package calls and a move-only split compiles. The reference generator's
self-description at verbs.go:235 and docs/reference.md:3 names verbs.go for the table, which
stays. README.md:72 lists the command's files by name and needs the new names.

**Fix.** Four files in package main: verbs.go (exit codes, types, table, `run`, help,
reference: ~276 lines), read.go (`reader`, `cmdLs`, `cmdStat`, `start`, `cmdRead`,
`printFrom`, `printRecords`, `cmdTail`, `cmdCheck`, `parseTime`), consumers.go (`cmdConsumers`,
`lag`, `consumer`, `cmdCommit`, `cmdRegister`, `cmdUnregister`), write.go (`writer`,
`cmdAppend`, `cmdPrune`, `cmdRoll`, `cmdArchive`, `cmdRecover`). Apply after the other
verbs.go edits (correctness-24, simplify-7, -8, -10, dedup-14, docs-46) so their anchors hold;
then update README.md:72. This resolves docs-37.

## 5. simplify-10: policyConfig pass-through, cmdCat duplicate, dead follow parameter (CONFIRMED)

**Anchors.** cmd/quill/verbs.go:301-307 (`policyConfig`, sole caller :681), :396-397
(`cmdRead`/`cmdCat` both `printFrom(c, false)`), :399 (`printFrom(c, follow)` only ever `false`;
`cmdTail` calls `printRecords` directly at :470).

**Skeptic (CONFIRMED).** `policyConfig` returns exactly what `ReadPolicy` returns at
state.go:290-298. Decision 86 says "`read` and `cat` are the same printer with different
defaults", but no default differs in code (both resolve the start via `c.start` to `st.First`);
the verb entries, summaries and flag sets stay separate, so help and the reference are
unchanged. Applied in a copy: gofmt clean, vet clean, full suite green, verbs.go 786 -> 776.

**Fix.** Delete `policyConfig` and write `cfg, ok := quill.ReadPolicy(c.dir)` at :681; delete
`cmdCat` and set `run: cmdRead` on the `cat` entry (:75); drop `follow` from `printFrom`.

## 6. simplify-6: Check duplicates verifyBounded, declares io.ReaderAt by hand, derives Healthy in a post-pass (CONFIRMED)

**Anchors.** stat.go:157-170 (two `verifyBounded` calls with byte-identical clock closures,
differing only in `int64(tl.Off)` vs `size`); stat.go:223-225 (`f interface { ReadAt([]byte, int64) (int, error) }`,
which is `io.ReaderAt`, and the value already flows into `window{r: f}` whose field is
`io.ReaderAt`, load.go:91); stat.go:207-211 (loop over `res.Problems` setting `Healthy=false`
for any `Fatal`, though `Fatal` is set in exactly one place, the `fatal` closure at :131).

**Skeptic (CONFIRMED).** Applied all three edits in a copy: gofmt clean, build, vet, full suite
green (`ok quill 10.470s`), stat.go 245 -> 235 lines; the `Check` tests that exercise
Healthy/Fatal/clock (reader_test.go:241-301, archive_test.go:133-138, retention_test.go:221,
backlog_test.go:142, the CLI exit-3 test) pass unchanged. One nit: `info, err :=` compiles only
once `var info segInfo` at :156 is dropped. The self-review's row (self-review:47) checks only
that `verifyBounded(f, tl.Off)` is used for the marker's segment, which the merged form keeps.

**Fix.** `bound := size; if id == tl.Seg { ...; bound = int64(tl.Off) }` and one call; `f io.ReaderAt`;
`res.Healthy = false` inside `fatal`; delete the trailing loop.

## 7. simplify-4: openActive is a 25-line copy of openSegment (CONFIRMED; same item as dedup-13)

**Anchors.** load.go:60-84 and load.go:461-485, differing only at :68 `fsx.O_RDONLY` vs :469 `fsx.O_RDWR`.

**Skeptic (CONFIRMED).** `diff` of the two bodies produces one hunk. `openActive` has one caller
(load.go:318); `openSegment` seven plus load_test.go:304-338. The doc comment at load.go:460
("with the checks of openSegment") already promises parity. No decision forbids a flag-taking
helper; `fsx.FS.OpenFile` already takes the flag.

**Fix.** `openSegmentFlag(fsys, path, flag)` holding the body; keep both names as one-line wrappers.

## 8. simplify-5: consumerStore duplicates an identical `first` closure (CONFIRMED)

**Anchors.** consumer.go:286-293 (Log) and :314-321 (Reader); both receivers already satisfy
`source` (read.go:38-45), and `open()` receives the same value again as `src` (consumer.go:62).

**Skeptic (CONFIRMED).** Only `head` differs (log.go:481 `Head()` returns a bare uint64;
reader.go:142 `Head` already has the `(uint64, error)` shape). Other `consumerStore` users
(retention.go:30, flush.go:134, stat.go:192) call only `list()`. Correction: `open()` has six
callers, not four (consumer.go:299, :326, archive.go:35, log.go:310, consumer_test.go:185, :188).

**Fix.** Store `src source` in `consumerStore`, make `first` a method, drop `open`'s `src`
parameter, update all six callers.

## 9. simplify-7: the per-verb FlagSet is built three times (PLAUSIBLE)

**Anchors.** cmd/quill/verbs.go:166-171 (`run`), :191-195 (`cmdHelp`), :253-257 (`cmdReference`).

**Skeptic (PLAUSIBLE).** The dedup is real and the proposed `flagSet(c, dirDefault)` helper is
behaviourally equivalent, but the claim misdescribes the code: `run` has `fs.SetOutput(io.Discard)`
at :167 and the other two do not, so the blocks differ in two things and `run` is six lines, not
five. Adding `SetOutput(io.Discard)` to all three is safe: `printVerbHelp` re-sets output to
`c.out` (:217) before `PrintDefaults`, and `cmdReference` only walks the set (:259-265). The
reference test (cli_test.go:53-69) and `help tail` test (:88-91) pin the output.

**Fix.** Add the helper; replace the three blocks. Low risk, pinned by the reference test.

## 10. simplify-8: cmdTail's start is seven lines of branches for a one-line max/min (CONFIRMED)

**Anchor.** cmd/quill/verbs.go:462-468.

**Skeptic (CONFIRMED).** All three operands are uint64; `min`/`max` builtins are already used in
the module. Compared old vs new over the 10^3 edge-value triples and 5,000,000 random triples:
0 mismatches; the replacement compiles and `go test ./cmd/quill/` passes. Correction to the
proposal: both existing tail tests are n <= Head (cli_test.go:197, :227); n > Head and `-n 0`
are not exercised, so the risk rests on the equivalence proof, not on the suite.

**Fix.** `from = max(st.Head+1-min(c.n, st.Head), st.First)`; add tail tests for n > Head and n = 0.

## 11. simplify-9: copySegment hand-rolls a ReadAt/WriteAt loop that io.CopyBuffer replaces (CONFIRMED)

**Anchor.** archive.go:161-174.

**Skeptic (CONFIRMED).** Re-derived from the go1.27.3 sources: `*io.SectionReader` and
`*io.OffsetWriter` implement neither `WriteTo` nor `ReadFrom`, so `CopyBuffer` honours the 1 MiB
buffer and issues the same ReadAt/WriteAt sequence (same count, size, offset) to the `fsx.File`.
Only divergence: `io.EOF` vs `io.ErrUnexpectedEOF` on a short source, unreachable in practice
because `verifySegment(r, size)` at archive.go:139 already read the whole segment, and nothing
matches `io.EOF` on this path. Nit: the replacement must sit inside `if err == nil { ... }` so
the copy is skipped when the preceding `Chmod` failed, as the loop condition does today.

**Fix.** The three-line `io.CopyBuffer` form, guarded by `if err == nil`. Apply before dedup-17,
whose `fill` callback then holds this form.

## 12. simplify-11: readTail retries every error five times (REFUTED)

**Anchor.** reader.go:121-139.

**Skeptics (REFUTED, PLAUSIBLE).** The code description is accurate (five attempts, 1 ms
sleep, retrying decode and I/O errors other than `fs.ErrNotExist`) and no transient failure is
demonstrable on the supported filesystems, but the loop is specified: build-prompt.md:184-185
says of the Reader "Each call reads `quill.tail` (verify it parses; on a transient failure retry
briefly)". The finding's "undocumented intent" premise is therefore false and deleting the loop
would depart from the build spec; the ENOENT exclusion is decision 62's caller-side retry. The
second skeptic concedes the same and adds that the only cost is ~5 ms before a persistent
`ErrCorrupt`/EACCES report, and that a `zz_skeptic_test.go` from another lens briefly appeared
in its copy during the run (unrelated).

**Disposition.** Not changed. If the owner wants the loop's rationale findable from the code, a
one-line decision naming build-prompt.md:184 would do; that is a documentation nit, not a
simplification.
