# Pass A7: gap audit

Date 2026-11-03, against HEAD 34947b2 (the fixes listed below were applied on top of it, the last being 0c83572). File and
line references are to the audited tree unless a section says otherwise; `log.go` lines that moved when A7-11 split the file
are cited as they were. The pass asks of each promise a repository makes (the specification, the build prompt, the
decisions, the README and the operator documents, the CI and build files, the audit records) three things: is it built, is
it wired and verified, is it documented and operable. The yardstick (appendix) is the numbered list of 184 promises the
auditors measured against; "Y31" below is row 31 of it.

Labels: **verified** means the recorder or a skeptic ran it or read the exact lines; **reported** means a document or an
auditor's note says it, and the source is named; **inferred** says from what. The counts and the per-item text come from the
pass ledger; the recorder re-checked ancestry of the fix commits with `git merge-base --is-ancestor` (verified) and nothing
else, because this record is docs only.

## Summary

17 gaps after merging duplicates (25 raw candidates; 8 were merged into a sibling: A7-3 and A7-5 and A7-9 and A7-10 and
A7-12 and A7-17 and A7-24 and A7-25).

| By verdict | Count | Items |
|---|---|---|
| CONFIRMED | 7 | A7-1, A7-4, A7-6, A7-11, A7-13, A7-15, A7-23 |
| PLAUSIBLE | 3 | A7-18, A7-19, A7-20 |
| REFUTED | 3 | A7-7, A7-21, A7-22 |
| prior-deliberate (decided in an earlier pass, not voted again) | 4 | A7-2, A7-8, A7-14, A7-16 |

| By disposition | Count | Items |
|---|---|---|
| Fixed, fix-skeptic FIXED-VERIFIED | 7 | A7-1 (47812f1), A7-4 (f0090db), A7-6 (3b86a4a), A7-11 (f47d6f0), A7-13 (1262a8f), A7-20 (730d525), A7-23 (0c83572) |
| Fix written, fix-skeptic objection; corrected and merged by hand 2026-11-03, no fix-skeptic re-run | 2 | A7-18 (08553bd, merged as e2f02b6), A7-19 (1a701fb, merged as ffae168) |
| Owner-approved 2026-11-03 and fixed (fix-skeptic pending when committed) | 1 | A7-15 (branch `stage2/A7-15-a2`) |
| Refuted, nothing to change | 3 | A7-7, A7-21, A7-22 |
| Deliberately not changed (decided in an earlier pass) | 4 | A7-2, A7-8, A7-14, A7-16 |

- **Severity.** 2 medium (A7-1, A7-15), 15 low. One item is a code bug (A7-1); the rest are documentation, hygiene or
  missing tests.
- **Skeptic votes.** 27 verdict votes on 13 items (two each, three on A7-21; the four prior-deliberate items were not
  voted). 16 CONFIRMED, 5 PLAUSIBLE, 6 REFUTED. The item verdict follows the votes: two CONFIRMED gives CONFIRMED, a split or
  two PLAUSIBLE gives PLAUSIBLE, a majority REFUTED gives REFUTED (A7-21 was 1 PLAUSIBLE, 2 REFUTED).
- **Reproduced.** 13 of the 27 votes are marked reproduced. Only 5 of those ran a program that exhibits the claim: the two
  probes of A7-1, the two mutants of A7-6 and the one mutant of A7-13 (a mutated copy of the tree on which the full suite
  still passed). The other 8 are reads and greps of the tree, marked reproduced by the voter. The one correctness bug, A7-1,
  was reproduced by the auditor and by both skeptics with three separate programs. There is no security item.
- **Fix-skeptics.** 7 fixes, 7 FIXED-VERIFIED, 43 lens verdicts (7 lenses on A7-1, 6 on each of the others), all pass.
  Two further fixes (A7-18, A7-19) were rejected on one objection each (below); the assistant corrected the one sentence each and
  merged them by hand on 2026-11-03 (e2f02b6, ffae168) with no fix-skeptic re-run and without the owner's word.
- **Ladder.** Every fix stopped at the lowest rung that closes the item: one line or one pure move or one table or one test;
  no new exported symbol, flag, file of code or config field was added by any of them (config.go in A7-11 is a move).

## Coverage

Three auditors returned; none failed to return. Everything ran on macOS (darwin/arm64) unless said otherwise.

**Checked and found clean** (verified by the auditors, from their reports):

- `deadcode -test ./...` with a local binary: empty report, exit 0; `go vet ./...` and `gofmt -l` clean; `go test -count=1 ./...`
  green (no `-race`). Examples `examples/log` and `examples/topics` run clean and `go test ./examples/...` passes.
- Every `Config` field is read by non-test code; `validate` refuses what the specification says it refuses; every `Stats`
  field is set; the exported API holds every error in the error set and every method the user guide, README, architecture,
  introduction, executive summary, `doc.go`, format and scope documents name.
- CLI: all 15 verbs plus help, version and the hidden reference verb exist; every flag in the verb table is read by its verb;
  every backticked flag in the docs exists in `docs/reference.md`, which a test keeps current. A scratch build was driven
  through append, stat, consumers, register, check, ls (including the `error:` row and exit 1), commit, tail, read, roll,
  recover, archive then cat, the refusals (a non-log directory, `_other`/upper-case names, `-protect` with `-no-protect`),
  and the usage exits.
- The manual corruption procedure (user guide section 10, seven steps) was run twice, by two auditors, with the real binary:
  flip a byte, check (UNHEALTHY, with the byte offset), cat to find the last good sequence, truncate, rewrite `quill.tail`
  by hand, recover, check (HEALTHY), append (next sequence follows). It works as written. The only test of it
  (`recovery_test.go:596`) is library-level, not the CLI.
- Wiring spot checks: log-ID check at `Consumer.Commit` and in the Reader's wait; `Err()` joining the load error and the
  background errors with per-operation clearing; the statfs failure reaching `Err()`; lock-lost check before flush and prune;
  torn/damaged kind on the cut warning; directory fsync before unlink, at rotation, after `quill.format` and at Close.
- The test ledger (117 backticked names) matches `go test -list`; every Benchmark is in `benchmark-baselines.md`; fuzz targets
  cover the decoders and parsers. An AST scan found no test without an assertion (five wrappers around asserting helpers).
- Platforms: freebsd, windows and openbsd fail to build; linux and darwin on amd64 and arm64 build.
- Hygiene: word-boundary `git grep` of the tracked tree and a search of every commit message, 401 commits, for the names of
  internal products, repositories and workspace paths found nothing at HEAD. History holds exactly one hit, the scratch path
  in the v0.1.0 tag's audit file (A4-4, an owner decision, see A7-16).
- Round-1 fixes: 28 items of `audit-findings.md` re-read in the code and none found unfixed; every `fixed <sha>` commit in
  `a1-round2-findings.md` is an ancestor of HEAD; five round-2 rows were spot-read. CHANGELOG v0.1.0 counts match
  `audit-followups.md`; 15 post-tag Unreleased entries were matched to commits; relative links in all `.md` files resolve.
- CI and gate: `ci.yml` jobs match README and the yardstick; the staticcheck v0.8.2 and deadcode v0.52.0 pins match
  `hack/gate.sh`; no build tag other than platform tags exists, so the opt-in test tiers are type-checked by plain `go vet`.
  `build/quill` refuses an untagged tree. SECURITY.md, CODEOWNERS, dependabot and the PR-issue-link workflow exist.
- Figures: ScanAll about 21 M records/s and Reader draining about 16 M records/s match `benchmark-baselines.md` (the other
  figures are A7-23).

**Not covered, do not read as clean:**

- Behaviours Y1-Y14 beyond the probes above (the auditors relied on test names plus a green suite and did not re-derive
  them); Y16-Y35 and Y36-Y127 as test-by-test coverage (only the rows named in the three reports were traced from promise to
  a named test); Y129-Y134 process claims (hand-backs, skeptic runs), which cannot be verified from the tree; Y169-Y172 prose
  and the executive summary; Y173-Y175 ledger rows beyond the ones cited.
- Not run by the auditors: the gate's pinned staticcheck and deadcode (the local staticcheck 0.8.1 cannot read go1.27 export
  data, verified error), govulncheck, `-race`, the 20 s fuzz targets, the Linux box, containers, test T14, `hack/gate.sh`,
  `hack/skew.sh`, `hack/powertest` (it reboots the machine) and the release build beyond its refusal path. That the
  pinned action SHAs and the toolchain are the latest stable was not checked (no network).
- Linux-specific behaviour was not exercised by the auditors. Fix-skeptics for A7-1, A7-6 and A7-13 ran the changed code
  and tests in a linux/arm64 container; the Linux x86-64 box was not reachable for them. 32-bit and unlisted GOARCH
  behaviour was not checked beyond building linux/386 and riscv64.
- Fix-skeptics ran gofmt, vet and `go test -race` on every fix. Staticcheck and deadcode at the pinned versions were run by
  the A7-11 skeptic (clean) and staticcheck by the A7-6 skeptic; the fuzz step of the gate was run by none, and none of the
  seven fixes touches a fuzzed path. The fixes were verified one at a time on the parent of each; the combined tree at
  0c83572 was not re-gated for this record.

## Ledger

Cost is the severity the auditor assigned. "Skeptic" is the fix-skeptic outcome.

| Id | Yardstick | Claim | Verdict | Cost | Disposition | Skeptic |
|---|---|---|---|---|---|---|
| A7-1 | Y31 Y32 Y92 | MinFreeBytes estimate does not credit segments already chosen for deletion in the same pass | CONFIRMED | medium | fixed 47812f1 | FIXED-VERIFIED |
| A7-4 | Y69 Y112 Y173 | scope.md still says the consumer-mtime premise is "to be tested" | CONFIRMED | low | fixed f0090db | FIXED-VERIFIED |
| A7-6 | Y177 Y30 Y11 Y33 | caps running at a seal made by Sync, Close, SyncAppends or the flusher is pinned by no test | CONFIRMED | low | fixed 3b86a4a | FIXED-VERIFIED |
| A7-11 | Y77 Y135 | docs state no file over about 600 lines; log.go is 845 | CONFIRMED | low | fixed f47d6f0 | FIXED-VERIFIED |
| A7-13 | Y64 Y62 Y171 | ErrFutureVersion at Consumer.Commit and in a Reader's wait has no test | CONFIRMED | low | fixed 1262a8f | FIXED-VERIFIED |
| A7-15 | Y128 | decision 90 says reference figures were replaced or softened; review.md and decisions 3, 15, 32 still carry them | CONFIRMED | medium | owner-approved 2026-11-03: figures deleted, decision 90 reworded (branch `stage2/A7-15-a2`; a1 was rejected on a duplicated cell in decision 90 and a surviving framing range) | pending |
| A7-23 | Y47 Y59 | docs quote 20 ms per macOS flush and 23-39 us per Read; the baselines say about 10 ms and 21-36 us | CONFIRMED | low | fixed 0c83572 | FIXED-VERIFIED |
| A7-18 | Y64 Y62 Y171 Y170 | no upgrade, migration or rollback note for an operator | PLAUSIBLE | low | fix written 08553bd, rejected; corrected by hand, merged e2f02b6, no re-run | objection (one) |
| A7-19 | Y75 Y141 Y166 | no install path documented; a go-installed binary reports `dev` | PLAUSIBLE | low | fix written 1a701fb, rejected; corrected by hand, merged ffae168, no re-run | objection (one) |
| A7-20 | Y33 Y153 | no message, meaning, action runbook for the log records and errors | PLAUSIBLE | low | fixed 730d525 | FIXED-VERIFIED |
| A7-7 | Y149 Y170 Y184 | README Layout table omits hack/topology, topology.sh, skew.sh, skewopen | REFUTED | low | no change | none |
| A7-21 | Y65 Y137 Y138 Y139 Y179 | v0.2.0 deferrals have no recorded disposition | REFUTED | low | no change | none |
| A7-22 | Y142 Y170 Y59 Y122 | by-hand release evidence is not named in the release list and is old | REFUTED | low | no change | none |
| A7-2 | Y21 Y45 Y15 Y113 Y178 | a Reader's Stat reports no hole and over-counts records when a middle sealed segment is bad | prior-deliberate | low | not changed, round-1 decision | none |
| A7-8 | Y180 Y73 Y176 | about 415 local scratch refs, one with a stray commit, safe only while nothing is pushed | prior-deliberate | low | not changed, round-1 decision | none |
| A7-14 | Y45 Y152 | `quill check` prints a policy line for a CLI-created log; no document lists it | prior-deliberate | low | not changed, round-1 decision | none |
| A7-16 | Y75 Y176 Y180 | no single public-flip checklist; several blockers open | prior-deliberate | medium | not changed, round-1 decision | none |

## Confirmed gaps

Each section follows Claim, Grounds, Warrant, Backing, Rebuttal. Votes are the two independent skeptics.

### A7-1: the MinFreeBytes estimate ignores segments the pass already deletes (bug, medium) CONFIRMED

**Claim.** The `MinFreeBytes` estimate in `capsLocked` does not credit the sealed segments already chosen for deletion in the
same pass (the age-sweep prefix, or the prefix a `MaxBytes` cap extended). A pass only slightly below the floor therefore
deletes extra non-expired, possibly protected, segments, up to every sealed segment.

**Grounds.** `retention.go:253-275` (`capsLocked`): `n` arrives holding the age-sweep prefix (`retention.go:201` calls
`capsLocked(n)`); the loop `for free < MinFreeBytes && n < len(l.sealed) { free += l.sealed[n].size; n++ }` starts at that `n`,
so `sealed[0:n]` are never added to `free`. The `MaxBytes` branch just above does count them. The same happens at a seal
(`flush.go:203` calls `capsLocked(0)`) when both caps are set. Empty search: no test in the tree combined an age-expired
prefix with a below-floor statfs (`diskfull_test.go` used statfs shortfalls with no expired segment). Reproduction
(verified, three programs): an in-package test on a scratch copy of HEAD with an injected clock and statfs, Retention 1h,
`MinFreeBytes` 1000, five sealed segments of 336 bytes, the oldest two expired, `Prune(true)`. statfs 1000 deleted exactly the
two expired (first sequence 7). statfs 999, a 1-byte shortfall that the 672 bytes freed by the expired pair already cover,
deleted three (first 10), losing the unexpired records 7-9. statfs 1 deleted all five sealed (first 16) where three suffice.
Skeptic one saw the same with free 1000, 900 and 700 leaving segments [2 3 4 5], [3 4 5] and [5]; skeptic two saw First 1 to 9
at the floor and 1 to 13 at 99,999 of 100,000.

**Warrant.** `Config.MinFreeBytes` (`log.go:59-62`) and the user guide (`userguide.md:151-153`) define the estimate as the one
statfs read plus the size of each segment the pass deletes. Age-swept and `MaxBytes`-chosen segments are deleted by the
pass, so they count. Over-deleting overrides consumer protection, so the loss is real and silent.

**Backing.** Y31, Y32, Y92; decision 25; earlier L2-27 (`mutation-hardening.md:545-557`) covered only the `MaxBytes` branch.

**Rebuttal.** Not a defect if over-deleting is the "up to one segment past its target" slack of Y92. The probe shows more
than one segment (statfs 1 deleted 5 where 3 suffice). Harmless when no age or `MaxBytes` deletion shares the pass with a
below-floor reading. The free reading is stale by the next pass, so only the extra loss persists.

**Disposition.** Fixed in 47812f1: `capsLocked` adds the size of `sealed[:n]` to `free` before the floor loop;
`FreeBytesAtLastPass` stays the raw statfs value. New test
`TestMinFreeBytesCreditsTheSegmentsTheAgeSweepAlreadyDeletes`. Fix-skeptic FIXED-VERIFIED, 7 of 7 lenses: the test fails in
all 3 subtests with only `retention.go` reverted; a separate probe on a `MaxBytes`-extended prefix and on the seal path
(Roll with both caps) differs between the old and new code; gofmt, vet and `-race` clean; `go doc -all` identical; hand trace
of the claim's scenario agrees; also run in a linux/arm64 container. Not run: the Linux x86-64 box, staticcheck, deadcode,
fuzz. The warning's `free` attribute now reports the credited estimate (verified by the skeptic). Docs touched: decision 25
note, scope.md and the CHANGELOG.

### A7-4: scope.md says the mtime premise is "to be tested" (doc, low) CONFIRMED

**Claim.** `docs/design/scope.md:319` describes the consumer-mtime premise as "inferred from temp-and-rename, to be tested"
although the test exists and passes.

**Grounds.** `consumer_test.go:259` `TestConsumerFilesAreWrittenAtomicallyAndCommitsLeaveTheirMtime` asserts that creation and
commit leave a file whose mtime is not before the call (lines 274-285); `test-ledger.md:47` maps T16 to it; the suite passed
on a scratch copy of HEAD. Empty search: `grep -rn 'to be tested'` over docs, README and `*.go` finds only `scope.md:319`.
Decision 31 already said "inferred; tested", so the two documents disagreed.

**Warrant.** The build prompt asserts the premise as T16; once the test exists the label is stale and a reader thinks the
premise unverified.

**Backing.** Y69, Y112, Y173.

**Rebuttal.** The test asserts a lower bound on mtime, not equality (creation has a one-second slack, `consumer_test.go:269`).
"Inferred" could stay for coarse-timestamp filesystems, minus "to be tested".

**Disposition.** Fixed in f0090db: the sentence now reads inferred from temp-and-rename and names the test, with the bounds
stated. FIXED-VERIFIED, 6 of 6 lenses (the named test passes under `-race`; the stated bounds match the test; no sibling stale
label). Cosmetic residue: the edited line is longer than its neighbours.

### A7-6: caps at a Sync, Close or flusher seal are pinned by no test (test, low) CONFIRMED

**Claim.** That the caps (`MaxBytes`, `MinFreeBytes`, `MaxRecords`) run at a seal made by Sync, Close, SyncAppends or the
flusher is stated in the docs but pinned by no test; only Flush-driven seals were tested.

**Grounds.** The statement is in `log.go:99-103`, `userguide.md:234` and the L1-10 note at `stage2-owner-decisions.md`
("read from `publishLocked`, not tested"). The existing seal test (`diskfull_test.go:176-213`) seals only through Flush.
Empty search: `git grep` for "every seal", "AtSeal" and "caps run" in `*_test.go` found nothing for Sync or Close. Reproduction
(verified): the auditor's probe (SegmentBytes 256, MaxBytes 512, `NoMaintenance`, Prune never called, 60 100-byte appends)
showed the behaviour holds today. Both skeptics then mutated scratch copies so the caps call is skipped on one path: skipping
it at the Close flush or at the flusher (the path Sync and SyncAppends use) left the whole existing suite green; skipping it
at the inline Append flush fails one test; removing it everywhere fails four.

**Warrant.** A refactor that moved the caps call out of `publishLocked` would break the Sync and Close paths with no failing
test.

**Backing.** Y177, Y30, Y11, Y33.

**Rebuttal.** Not a gap if the T13 test already sealed through a non-Flush path; the mutants show it does not (it covers the
Append path only).

**Disposition.** Fixed in 3b86a4a: `TestTheCapsRunAtASealMadeByFlushSyncSyncAppendsOrClose`, 4 seal paths by 3 caps, no
Prune, `NoMaintenance`. Test only. FIXED-VERIFIED, 6 of 6: with the caps call removed the 12 subtests fail; per-path mutants
(Flush-only, skip at Close, skip in the flusher) fail exactly the corresponding rows. Residual: Roll is not a row (it shares
`publishLocked`), the assertion is that the oldest segment id changed, not the exact retained set, and the Linux x86-64 box
was unreachable (container used).

### A7-11: log.go is 845 lines against a stated 600-line ceiling (hygiene, low) CONFIRMED

**Claim.** README, CONTRIBUTING and architecture.md say no file is over about 600 lines; `log.go` was 845 and no decision or
audit recorded the exception.

**Grounds.** README.md:52, CONTRIBUTING.md:19, `docs/architecture.md:31`, decision 20. `wc -l` of tracked non-test Go files
outside hack/, examples/, build/: `log.go` 845; next `internal/fsx/crash.go` 584, `reader.go` 531, `load.go` 514.
`hack/topology/main.go` (812) and `run.go` (726) also exceed it. The earlier audit (docs-37, `audit-docs.md:124-138`) split
`verbs.go` at 786 and accepted 615 and 596 as "about 600". Empty search: `git grep '845\|log.go.*lines\|600 lines'` over
docs/audits and CHANGELOG found nothing about `log.go`. Merged from A7-5 and A7-24 (reported by two auditors).

**Warrant.** 845 is 40 percent over a ceiling stated as fact; the previous audit treated the same claim as a finding.

**Backing.** Y77, Y135, decision 20.

**Rebuttal.** "About 600" is loose, decision 20 concerns the library and hack/ programs are harnesses. Softening the
statements was the auditor's cheapest option, but docs-37 rejected softening, so a split is the consistent fix.

**Disposition.** Fixed in f47d6f0: `Config`, `withDefaults`, `validate`, `maxActiveBytes`, `dirPerm`, `archiveConsumer`,
`validConsumerName` and `validName` moved to `config.go` unchanged; `log.go` is 542 lines, `config.go` 313. README file table
and CHANGELOG updated. FIXED-VERIFIED, 6 of 6: sorted-line diff of the old `log.go` against the new pair shows only package
and import lines; `go doc -all` identical; gofmt, vet, `-race` and staticcheck v0.8.2 and deadcode v0.52.0 clean; a mutation
of the moved `MaxBytes` check is caught by `TestEveryConfigDefaultAndRefusal`. Residue: `hack/topology` is still over 600
lines (CONTRIBUTING.md:19 is unqualified), and two dated notes (decisions.md L1-10, CHANGELOG.md:45) still say the Logger
documentation is in `log.go`; both were judged not blocking.

### A7-13: ErrFutureVersion at Commit and in the Reader's wait is untested (test, low) CONFIRMED

**Claim.** The promise that a newer `quill.format` is refused with `ErrFutureVersion` at `Consumer.Commit` and while a Reader
waits has no test; the swap test covers only a different log id (`ErrWrongLog`).

**Grounds.** `scope.md:471-475` and `format.md:168` say the marker is re-checked at Commit and while waiting; `consumer.go:220`
and `reader.go:206` call `checkLogID`, which calls `loadFormat`. Empty search: `git grep ErrFutureVersion` in `*_test.go`
finds only the open and decode paths (`archive_test.go`, `log_test.go:77`, `reader_test.go:396`, `state_test.go:66/70`);
`identity_swap_test.go:15-53` asserts `ErrWrongLog` only. Verified by the auditor's scratch program (rewrite `quill.format` to
`quill/2 <id>` after opening: Commit and ReadWait both return the error) and by a skeptic's mutation: making `checkLogID`
return nil on `ErrFutureVersion` left every package green.

**Warrant.** The migration-note design depends on the check; a refactor of `checkLogID` could drop it silently.

**Backing.** Y64, Y62, Y171, Y72.

**Rebuttal.** `loadFormat`'s version check is tested through Open and OpenReader, but the early return in `checkLogID` is
exactly what the mutation added and nothing caught it. The docs word the wait check as "ReadWait and Follow"; only the
Reader's wait re-checks, and Follow shares that function.

**Disposition.** Fixed in 1262a8f: `TestAHandleOpenAcrossANewerFormatRefusesCommitAndWaitWithErrFutureVersion`; a refused
Commit leaves the consumer file and `Offset()` unchanged. Test only. FIXED-VERIFIED, 6 of 6: four production mutants (in
`checkLogID`, at each of the two call sites, and in `loadFormat`) each fail it; also run in a linux/arm64 container. Residue:
`Consumer.Commit`'s doc comment (`consumer.go:209-210`) lists `ErrWrongLog` but not `ErrFutureVersion`; Follow and
`Consumer.ReadWait` are covered transitively.

### A7-15: reference figures remain although decision 90 says they were removed (gap, medium) CONFIRMED

**Claim.** Decisions 70 and 90 say every figure in review.md and scope.md that came from the private reference was replaced by
quill's own measurement or softened to "the reference measured" without numbers. The tree does not match: review.md and
decisions 3, 15 and 32 still carry the reference's line counts and timings. The public-release prerequisite is undone and a
decision row says it is done. Merged with A7-12.

**Grounds.** Verified by reading and grep at HEAD. `docs/design/review.md:3` (about 9,000 lines), `:18-20` (about 4,700,
2,000, 1,400, and about 900), `:63` (about 670), `:64` (about 2,900), `:71` (about 3,300), `:94` ("at most about 5,400" of the
reference's 9,000). Finding 1 (`review.md:61`) keeps ratios such as "several times faster at one shard than at four".
`decisions.md:23` (decision 15) lists per-read timings and drain rates of the reference; decision 3 (`decisions.md:11`) says
"85% of read CPU"; decision 32 (`decisions.md:40`) gives a framing size measured on the reference. Decision 90
(`decisions.md:98`) asserts the opposite. Empty search: no ledger records the task as undone (docs, README and CHANGELOG
searched). `scope.md` was read at lines 272, 278, 303, 312, 437 and 501 and holds only qualitative references (reported by
the auditor; one skeptic did not re-check it). review.md also describes the reference's architecture at length, in text
README.md:34-35 links as design documentation (the recorder omits the detail here).

**Warrant.** Y128, decision 70 and `build-prompt.md:424-426` require replacing or softening these figures before the
repository is public.

**Backing.** Decisions 70 and 90; Y74.

**Rebuttal.** Decision 70 allows "or softens", so the owner may accept the figures as descriptive. Then decision 90 must stop
claiming they were removed. Decision 90 names scope.md and review.md, so decisions 3, 15 and 32 are not strictly within its
sentence, and decision 15's timings are labelled as measured on the reference's access pattern; the review.md line counts
alone carry the finding.

**Disposition.** Held for the owner, who approved the recommended answer on 2026-11-03: delete the reference's numbers. Done on
branch `stage2/A7-15-a2` (documentation only; the framing figure is one computed range in decision 32, scope.md and format.md): review.md, decisions 3, 5, 7, 15, 16, 20, 21 and 32 carry no figure from the
reference, each with a dated note; decisions 70 and 90 say what was done (Y128 is satisfied by deletion, not replacement);
the CHANGELOG has the line. Fix-skeptic outcome: attempt 1 rejected (two defects, both corrected in attempt 2); attempt 2 not yet run when the commit was made. Section "A7-15" in
[stage2-owner-decisions.md](stage2-owner-decisions.md) lists every figure found and what became of it.

### A7-23: operator documents quote figures that disagree with the baselines (hygiene, low) CONFIRMED

**Claim.** `userguide.md:48` says a flush costs "about 20 ms on macOS (F_FULLFSYNC)" and `scope.md:373` says the same, while
`benchmark-baselines.md:26` records `AppendFlushEach` at 9,793,069 ns (range 9,615,704-10,242,545) and `:41-45` says about
10 ms with two full fsyncs; `scope.md:238` itself says about 10 ms. `scope.md:436` gives Read(limit 100) as "about 23-39 us";
the re-recorded table has 24,592, 20,696 and 36,192 ns (`benchmark-baselines.md:28-31`), that is 20.7-36.2 us (inferred:
the older range predates the allocation-pass re-record).

**Grounds.** Verified by reading. Nothing in the text says 20 ms is another machine or a worst case. `scope.md` contradicted
itself (10 ms at `:238`, 20 ms at `:373`). Decision 51 (`decisions.md:59`) repeats the 20 ms and decision 39 gives a
third-fsync cost of 20 ms that does not fit 10 ms for two. Empty search: `grep '20 ms'` and `grep '23-39'` over the docs; the
three sites above plus decisions 39 and 51 are the hits.

**Warrant.** Y47 and Y59 say these figures are measured on quill's build machine and cite the baselines file; the 20 ms
figure is the one an operator uses for sizing.

**Backing.** Y47, Y59; `benchmark-baselines.md`.

**Rebuttal.** Not a gap if "about 20 ms" meant another machine or "about" covers a 2x spread; the voters found no such
qualification. The finding's reference to "decisions 47 and 59" is wrong (those decisions are about other things); the
discrepancy stands on `scope.md:238` and the baselines file.

**Disposition.** Fixed in 0c83572: the macOS flush is about 10 ms in the user guide, scope.md and decision 51; the Read range
is 21-36 us; decision 39's third-fsync figure is now about 5 ms, labelled inferred from the baselines there. CHANGELOG line
added. FIXED-VERIFIED, 6 of 6. Residue (the skeptic's): `a1-round2-correctness.md:1097`, a dated audit record, still quotes
about 20 ms for decision 51; the CHANGELOG states the 5 ms without the "inferred" label.

## Plausible gaps

### A7-20: no runbook for the documented log records and errors (gap, low) PLAUSIBLE (votes: PLAUSIBLE, CONFIRMED)

**Claim.** The `Config.Logger` row (`userguide.md:234`) is one table cell of about 3,300 characters cataloguing roughly 20
records, with no actions; some messages an operator can see have no operator text.

**Grounds.** Verified. `logf` messages such as "quill.lock is wider than 0600" (`log.go:465`), "cannot read free space"
(`retention.go:269`) and "quill.policy could not be written" (`log.go:829`) appear in the guide only as clause fragments in
that cell; `ErrFutureVersion` appears 0 times in the guide. The corruption procedure, which the auditor ran end to end, is a
real runbook. Empty search: `grep -c` on the guide for the three message fragments gave 0 outside line 234.

**Warrant.** The audit's place-5 check lists "no runbook for the documented failure messages".

**Backing.** Y33, Y153; `userguide.md:53-76`, `:345-415`.

**Rebuttal.** Most records are self-describing and the failed-log record is explained at `userguide.md:53`; no spec requires
a runbook; the voter who rated it PLAUSIBLE could not tell a defect from a documentation-scope judgement. Hence PLAUSIBLE.

**Disposition.** Fixed in 730d525: a message, meaning, action table of eight rows in user guide section 9, a pointer from the
Logger row, a CHANGELOG line. FIXED-VERIFIED, 6 of 6: each message string and action was checked against the code; the CLI
forms were run (unregister of a garbage consumer file turns check HEALTHY; ls on a `quill/9` marker prints an `error:` row,
exit 1). Imprecisions the skeptic left: "that group" for the lock row (the check is any mode other than 0600), and a failed
reopen after a lock-check failure may meet `ErrLocked` if another writer really holds the log.

### A7-18: no upgrade, migration or rollback note (gap, low) PLAUSIBLE (votes: CONFIRMED, PLAUSIBLE)

**Claim.** Nothing in the operator documents says what upgrading a live log between releases requires or whether the old
binary can be put back.

**Grounds.** Verified by search: `upgrad|downgrad|rollback|older binary|newer|skew` over the user guide, README,
architecture and introduction find only unrelated hits; `format.md:164-168` says a migration note for a new format version
must require stopping every process, but none exists because no format change has happened; the user guide has 0 mentions of
`ErrFutureVersion`. The evidence exists elsewhere: `format-compatibility.md` (A6) says the v0.1.0 and later binaries share a
log directory in both directions with announced divergences, and `hack/skew.sh` checks it. The CHANGELOG Unreleased section
(lines 6 to about 366, 147 bullets; the auditor's "6-262, 60+" was too low, per the skeptic) has no upgrading summary.

**Warrant.** Place 5 names an upgrade or migration note and rollback.

**Backing.** Y64, Y62, Y171, Y170; `format-compatibility.md`.

**Rebuttal.** No spec requires a note while the format is unchanged, and the CHANGELOG could be accepted as the whole note;
hence PLAUSIBLE.

**Disposition.** Fix written and rejected: 08553bd adds a 14-line "Upgrading and rolling back" paragraph to user guide
section 9 and a CHANGELOG line. The fix-skeptic raised one objection and did not return a FIXED-VERIFIED outcome: the text
says `format-compatibility.md` records "seven announced differences in edge cases" with three examples; in that audit "seven"
counts decoder changes and fuzz divergences, the three examples are not among them, its own list of documented differences is
longer, and the `quill.lock` mode change is in the CHANGELOG but not in that audit. Everything else the skeptic checked held
(the `ErrFutureVersion` behaviour, which calls re-read `quill.format`, the exit code, the `skew.sh` usage, the link, gofmt,
vet, `-race`). Commit 08553bd exists in the repository (verified, `git cat-file`) but is not an ancestor of this branch.
Recommended re-fix: drop the count or say "several", point to the CHANGELOG Unreleased section, and name only differences
that the audit or CHANGELOG records; see [stage2-owner-decisions.md](stage2-owner-decisions.md), section "A7-18".

Hand finish 2026-11-03: corrected as recommended and merged as e2f02b6 (the count and the examples are gone; the reader is pointed to
the CHANGELOG Unreleased section). No fix-skeptic re-run, and the owner was not asked.

### A7-19: no install path documented; a go-installed binary reports `dev` (gap, low) PLAUSIBLE (votes: PLAUSIBLE, PLAUSIBLE)

**Claim.** README, user guide and introduction never say how to obtain the `quill` command or the library, and a binary built
with `go install` prints version `dev`.

**Grounds.** Verified by search: `go get|go install|release archive|download` over README, user guide, introduction,
executive summary, reference and `examples/README.md` returns nothing; `README.md:83` names `build/quill` only as a Layout
row; the library import path appears at `README.md:81` (a voter's correction: the library is half-documented).
`cmd/quill/main.go:7-8` has `var version = "dev"`, stamped only by `build/quill`'s `-X` flag, with no `ReadBuildInfo`
fallback (inferred from the code, not run: no network and no remote). Release archives hold README, CHANGELOG, SECURITY and
LICENSE if present, not the user guide, format or reference (`build/quill:69-71`).

**Warrant.** Place 5 asks for an install path; it matters at the public flip.

**Backing.** Y75, Y141, Y166; decision 88.

**Rebuttal.** The repository is private and pre-release, README defers public-release items to the flip, and no spec requires
an install section; the proposed `go install` lines cannot work until the repository is published. Hence PLAUSIBLE.

**Disposition.** Fix written and rejected: 1a701fb adds an "Installing" section to README.md and a CHANGELOG line. The
fix-skeptic raised one objection: the new text lists LICENSE among the archive contents as fact, but `build/quill:70` copies
it only if it exists and the tree has none (`README.md:113-114` and decision 22 say no LICENSE while private); the fixer
dropped the finding's own "if present" qualifier. The `quill dev` statement and the omission of `docs/` from archives were
verified correct. Commit 1a701fb exists (verified) and is not an ancestor of this branch. Recommended re-fix: say the
archives carry LICENSE once one lands, or drop it from the list; see [stage2-owner-decisions.md](stage2-owner-decisions.md),
section "A7-19".

Hand finish 2026-11-03: corrected as recommended and merged as ffae168 (LICENSE "once the repository has one"). No fix-skeptic
re-run, and the owner was not asked.

## Refuted

REFUTED findings are kept with the reason.

### A7-7: README Layout table omits the multi-process harness and skew scripts (doc, low) REFUTED (2 of 2)

**Claim.** The README Layout table promises to name every file but omits `hack/topology`, `hack/topology.sh`,
`hack/skew.sh` and `hack/skewopen`; the committed docs outside `docs/audits` do not describe the multi-process harness.
**Grounds.** `README.md:54-84` lists `gate.sh`, `fuzz.sh` and `powertest` only; `grep topology` over README, CONTRIBUTING,
user guide, architecture and CHANGELOG finds nothing. **Warrant.** Y149 (the table names every file); stated harness flags are
promises (Y184). **Backing.** Y149, Y170, Y184. **Rebuttal.** Harnesses are internal tooling and `harness-inventory.md`
covers them. **Reason refuted.** The README never says the table names every file and it is selective (it omits `docs/`,
`go.mod`, CHANGELOG, SECURITY and most of `.github`); CONTRIBUTING.md:52-72 documents `skew.sh` and `skewopen` in detail and
`harness-inventory.md` covers `topology`. **Note.** Both skeptics also said the cited Y-numbers "do not exist in the
repository"; they are the audit's own numbering, which this file now commits, so that part of their reasoning is not
evidence either way. The README point stands on its own. **Disposition.** No change (a nice-to-have: two README rows).

### A7-21: v0.2.0 deferrals have no recorded disposition (gap, low) REFUTED (votes: PLAUSIBLE, REFUTED, REFUTED)

**Claim.** Decision 64 says `quill repair` is "deferred to v0.2.0" and `scope.md:482` lists candidates that "land only if
measured"; the optimisation pass recorded nothing for most (verified: no hits for idx, batched, in-place, Set helper or
skip-corrupt in `verification-optimization.md` or `stage2-owner-decisions.md`); `executive-summary.md:46` says "v0.2.0 is
the same code after the full audit sequence", which the CHANGELOG contradicts (257 commits since v0.1.0, 217 fix or feat);
read-8 has its owner question in `audit-followups.md:162-173` and its diff only in a scratchpad. **Warrant.** Place 7
(handed on, not closed). **Backing.** Y65, Y137-Y139, Y179, decisions 55 and 64. **Rebuttal.** The sequence is still
running; "not yet reached" is a valid reading. **Reason refuted.** v0.2.0 is untagged and the audit sequence is running;
`scope.md:482` already supplies the default (a candidate lands only if it measures, otherwise it goes to the "later" row), and
the reconciliation is a tag-time edit. Two minor residues are real and are not a gap: the executive-summary sentence is
loosely worded, and the read-8 diff is not in the repository. **Disposition.** No change now; the tag-time edits (move the
candidates to "later", amend decision 64, reword the sentence, note read-8) are the recommended answer.

### A7-22: by-hand release evidence is not named in the release list (test, low) REFUTED (2 of 2)

**Claim.** The real power-loss run, the version-skew check and the benchmark baselines were recorded against code long before
HEAD (power-loss runs of 2026-10-28; the skew run at 988f1e5, 91 non-test commits ago; baselines at the v0.1.0 commit), and the
release list at `build-prompt.md:433-434` does not name them. Merged with A7-3. **Warrant.** Y142 and Y170 make them release
evidence; Y59 and Y122 need a current baseline run. **Backing.** Decision 92; CONTRIBUTING.md:67-72. **Rebuttal.** The
claim is about the written list, not about the runs being skipped. **Reason refuted.** `build-prompt.md:433-434` is the closed
v0.1.0 procedure; v0.2.0 is governed by decision 55 (the full audit sequence, which includes the skew check, the baselines and
the topology work); CONTRIBUTING.md:52-72 already says to run `skew.sh` before a release; and a real sysrq power cut was
re-run on stage-2 code in pass L3b (`live-integration.md:125`, reported). The skeptics' point that "decisions 142 and 170 do
not exist" is the Y-numbering issue noted at A7-7. **Disposition.** No change; the remaining point (no single v0.2.0 checklist
naming `powertest` at the tagged commit) is cosmetic.

## Decided in an earlier pass

These four were not voted again: each was already decided in an earlier pass, and the recorder found no new fact against the
earlier reason.

### A7-2: a Reader's Stat misses a bad middle sealed segment (gap, low)

**Claim.** A Reader's Stat, so `quill stat`, reports no hole and an over-counted `records` when a middle sealed segment is
empty or has an unreadable first frame. **Grounds.** Reproduced by the auditor with the CLI built from HEAD: four segments,
the middle one truncated to 0 bytes; `quill stat -json` gives `holes: []` and `records: 10`, `quill cat` reads 1-3 then fails
with "records 4 to 6 are unavailable", `quill check` reports the empty segment as fatal and the hole 4-6, and after
`quill recover` the stat is unchanged. A middle file replaced with 20 random bytes gives the same stat. The cause is in
`reader.go` `view()`: a bad sealed segment is skipped and the previous good segment's range is extended to the next good
first sequence, so the discontinuity is only seen once that segment has been scanned. Merged from A7-9 and A7-17.
**Warrant.** Y21, Y45 and Y15 say Stat reports holes and a damaged segment is a gap. **Backing.** Y113, Y178;
`check_empty_sealed_test.go:57-59` records "does not yet". `userguide.md:196` lists `holes` for `r.Stat()` unqualified.
**Rebuttal.** Recorded as a known residual ("left", `stage2-owner-decisions.md`, `verification-optimization.md:705`, L1-11
sibling); Y89 says Reader.Stat reports only what a reader can know; finding the hole needs a scan of the preceding segment,
which is O(bytes); `quill check` does report it. The recorded residual names only the empty segment, while the reproduction
shows it for any bad first frame. **Disposition.** Deliberately not changed. Owner decision to revisit: say in the guide, the
package doc and Stat's godoc that a Reader's Stat sees holes only from first frames, and have `quill stat` print a line
pointing to `quill check` when a sealed segment is bad; building detection means scanning the preceding segment, which changes
the cost claims in Y19 and Y89. The broader-than-recorded scope of the residual (any bad first frame) is the one new fact this
pass adds.

### A7-8: about 415 local scratch refs (hygiene, low)

**Claim.** The local repository carries about 415 scratch refs (`stage2/*`, `failed/*`, `worktree-*`), one with a throwaway
commit authored `t <t@x>` with message `x` that changes `go.mod`; they are safe only while nothing is pushed.
**Grounds.** Verified: `git branch -a` counted 415; `git remote -v` prints nothing, so none is pushed. **Warrant.** Y180
forbids pushing `stage2/*`; a `git push --all` or `--mirror` would publish all. **Backing.** Y180, Y73, Y176.
**Rebuttal.** Working state of the audit workflow; safe without a remote. **Disposition.** Deliberately not changed; recorded
in `supply-chain.md:138` and ranked action 5 (`:422`), A4-4. Recommended: one sentence in the A4-4 owner decision (push
`main` and tags only; delete or archive `stage2/*`, `failed/*` and `worktree-*` before adding a remote).

### A7-14: `quill check` prints a policy line for a CLI-created log (doc, low)

**Claim.** After `quill append` on a new directory, `quill check` prints HEALTHY, the UNBOUNDED line, then a `policy ...: no
such file or directory` line (exit 0); no document lists the policy problem among what `check` reports. **Grounds.**
Verified. The cause is deliberate: offline verbs run with maintenance off and never write `quill.policy` (decisions 86 and 99,
`offline_policy_test.go:98`) and `stat.go:318-325` emits a non-fatal `Problem{Kind:"policy"}`; `userguide.md:413` does not
list it; `scope.md:215` lists "policy" only as a JSON kind. **Warrant.** The documented first steps meet an unexplained line.
**Backing.** Y45, Y152. **Rebuttal.** A warning, not fatal, and the kind is in the JSON shape. **Disposition.** Deliberately
not changed; recorded as G37 in `silent-degradation.md` (":1195-1196", cosmetic). Recommended: one sentence in the user guide
"Checking" section saying a missing or unparseable `quill.policy` is a non-fatal `policy` problem.

### A7-16: no single public-flip checklist (hygiene, medium)

**Claim.** The public flip has no single checklist in the repository and several blockers are open: no LICENSE (and
`build/quill:71` silently builds archives without one); no notice for the Go standard library the archives embed; the v0.1.0
tag still holds a workspace path (commits 98ff9e7 and fb83e37 added it; 0b70e82 removed it on `main` but the tag is
immutable); 214 unmerged local branches (205 `stage2/*`, 9 `failed/*`), of which only `stage2/*` is named as not to be pushed;
CODEOWNERS routes to one login nobody has confirmed can write; private vulnerability reporting is not yet enabled; box-local
tooling names in `audit-prompt.md` (low weight). **Grounds.** Verified: `git ls-files` shows no LICENSE; `build/quill:70-71`
copies LICENSE only if it exists; `build_script_test.go:126` writes a LICENSE fixture so the no-LICENSE tagged path is
untested; `build/quill` with no flag exits 1 "HEAD is not tagged"; `.github/CODEOWNERS:1-12` says to confirm each login, and no
remote exists to run that against; `SECURITY.md:9-12` depends on private reporting (A4-3). **Warrant.** Y75, Y176 and Y180
make these preconditions. **Backing.** Decisions 22 and 54; `supply-chain.md` A4-3 to A4-6 and the ranked actions
(`:414-426`); `stage2-owner-decisions.md:713-715, 807-814`. **Rebuttal.** Every item except the `failed/*` branches and the
audit-prompt names is already recorded as an owner action, and decision 22 makes "no LICENSE while private" correct today;
the claim is only that the list lives in three audit files and a changelog line, and that `build/quill` does not enforce its
half (A4-6 was refuted). **Disposition.** Deliberately not changed; round-1 decisions A4-3 to A4-6 stand. Recommended answer:
one short "Before the repository is public" list (CONTRIBUTING.md or the top of `stage2-owner-decisions.md`) naming LICENSE
and the Go licence text, the history path, private vulnerability reporting, "push `main` and the tag only", the CODEOWNERS
login and the README Status line.

## Deliberately not open

Nothing below is open work for this pass. Each row says why it was not changed.

| Item | Why not changed | Where decided |
|---|---|---|
| A7-2 Reader Stat misses a bad middle sealed segment | Finding it needs a scan of the preceding segment, which changes the cost claims; `quill check` reports it; recorded as a known residual | L1-11 sibling, `check_empty_sealed_test.go:57-59` |
| A7-8 local scratch refs | No remote exists; flip procedure is to push `main` and tags only | `supply-chain.md:138, 422`, A4-4 |
| A7-14 policy line in `quill check` | Deliberate: offline verbs never write `quill.policy`; non-fatal; cosmetic | G37, decisions 86 and 99 |
| A7-16 flip checklist and blockers | Every blocker is already an owner action; decision 22 makes "no LICENSE while private" correct | A4-3 to A4-6, decision 22 |
| A7-7 README Layout omits harnesses | The table is selective by its own wording; CONTRIBUTING and `harness-inventory.md` cover the scripts | refuted here |
| A7-21 v0.2.0 deferrals | Untagged release, sequence running; `scope.md:482` already gives the default; a tag-time edit | refuted here |
| A7-22 by-hand release evidence | Decision 55 governs v0.2.0; CONTRIBUTING already names `skew.sh`; a power cut was re-run in L3b | refuted here |
| Tag message "46 candidates, 39 fixes" against 45 raised | The tag is immutable and the difference is trivial | auditor note, not raised |
| README "iterator API needs Go 1.23" against `go 1.27` | Refuted 2 to 1 earlier | A4-2, `supply-chain.md:186-202` |
| Issue and PR templates absent | Nothing promises them; decision 73 records labels not bootstrapped | Y164, decision 73 |
| User guide gives no cold-start figure | Covered by `architecture.md:144` and `benchmark-baselines.md:60` | auditor note, not raised |
| Chaos-counter fixes absent from the CHANGELOG | Harness-internal | auditor note, not raised |

## The rule this pass produced

**A sentence that says something is done, measured or tested must name the artifact that shows it (a test, a baseline row, a
commit), and the pass re-derives it from that artifact.** The seven confirmed gaps are instances. Four were statements that
had drifted from their artifact: "to be tested" beside an existing test (A7-4), a decision recording a replacement that was
not made (A7-15), a flush figure that disagreed with the baseline row (A7-23) and a size ceiling that the tree exceeded
(A7-11). Two were behaviours the docs state and no test would notice losing (A7-6, A7-13). One was code that did not
implement the sentence that defines it (A7-1).

Corollary, from the two fixes that failed review: a documentation fix that quotes a count, a list or a contents claim copies
it from the document or script it cites, with that source's own qualifiers (A7-18 miscounted the audit's divergences; A7-19
dropped "if present" from the archive contents). Both were caught by the fix-skeptic reading the new text against its source,
not by any test.

This is the recorder's reading of the ledger, not a decision recorded by the owner.

## Appendix: the yardstick

The 184 rows the auditors measured against, condensed to one line each; the source column gives the text each row was taken
from. A row is a promise, not a finding. "build-prompt" is `docs/design/build-prompt.md`, "scope" is `docs/design/scope.md`,
"decisions" is `docs/design/decisions.md`.

| Y | Promise | Source |
|---|---|---|
| 1 | Guarantee 1: every read returns records in strictly increasing sequence, each at most once | scope:383-384 |
| 2 | Guarantee 2: durable before visible; a Reader never passes `quill.tail`, written only after its data is fsynced | scope:385-387 |
| 3 | Guarantee 3: a crash loses a suffix, not a middle; torn tail truncated at the next writable open and reported torn or damaged | scope:388-391 |
| 4 | Guarantee 4: sequences are contiguous; a marker ahead of the data fails Open; unflushed numbers are reassigned | scope:392-394 |
| 5 | Guarantee 5: one writer per log, one process per consumer name, `ErrLocked`; lock removed under a live writer gives `ErrFailed` at the next flush or prune | scope:395-399 |
| 6 | Guarantee 6: corruption never served silently; a bad middle segment fails to load, is named by `Err()` and `check`, reaches readers as `GapError` | scope:400-403 |
| 7 | Guarantee 7: allocation bounded on hostile input (64 MiB frame, 1.5 GiB segment); `MaxReadBytes` counts payload plus 48 bytes per record | scope:404-407 |
| 8 | Guarantee 8: retention deletes only whole sealed segments, never the active one, never reuses an id | scope:408 |
| 9 | Guarantee 9: state files atomic and strict | scope:409 |
| 10 | Guarantee 10: a stale marker hides, never exposes | scope:410-412 |
| 11 | Guarantee 11: bounded disk (`MaxBytes` formula, `MinFreeBytes` overrides protection, statfs failure reported) | scope:413-417, 277-285 |
| 12 | Guarantee 12: with `ConsumerTTL` a stale consumer cannot pin the disk | scope:418-420 |
| 13 | Guarantee 13: a failed write never corrupts; an fsync error refuses writes until reopen | scope:421-422 |
| 14 | Guarantee 14: reads never skip a gap; I/O errors are returned and retryable | scope:423-427 |
| 15 | The eight-line model: one directory, one writer, separate topics, contiguous sequences, gaps reported, consumer file written by its own process, frozen format | scope:20-36 |
| 16 | Open, Close, Append, AppendBatch, Flush semantics, error sequence rules | scope:56-58 |
| 17 | Read, ReadWait, Scan, ScanAll semantics and bounds | scope:59-61 |
| 18 | Records and Follow iterators on Log, Reader and Consumer | scope:62, 80-94 |
| 19 | `OffsetForTime` clamping, cost statement, CLI `-since` and `-time` | scope:63, 96-103 |
| 20 | `Archive(srcDir, dstDir)` copies sealed segments to a destination that opens as a log | scope:64 |
| 21 | Head, Stat, Err: fields, Log.Stat extras, `Err()` joining background errors that clear | scope:65 |
| 22 | `Roll()` and `Sync(ctx, seq)` as group commit | scope:66-67 |
| 23 | `(*Log).Consumer` and its methods and `ErrClosed` rules | scope:68 |
| 24 | `DeleteConsumer`, `Consumers`, `OpenReader` and `ReaderOptions` (50 ms poll), no Refresh | scope:69-70 |
| 25 | The exported error set and `*GapError`; Log and Reader safe for concurrent use | scope:72-78 |
| 26 | `Config.Dir`: required, created, topic name, symlink rule, made absolute | scope:109 |
| 27 | `Retention`, `PruneInterval`, `SegmentMaxAge` defaults and whole-second comparison | scope:110-111, 116 |
| 28 | `FlushBytes`, `FlushInterval`, `SyncAppends`, `MaxBufferBytes` and its refusal | scope:112-114 |
| 29 | `SegmentBytes`, `MaxPayloadBytes`, `MaxBatchBytes` (len+34), `MaxReadBytes` | scope:115, 119-121 |
| 30 | `MaxBytes` (refused below 2 x SegmentBytes) and `MaxRecords` | scope:117, 125 |
| 31 | `MinFreeBytes` via statfs, overrides protection, ENOSPC when nothing is left, reported by `Err()` and `Stat` | scope:118 |
| 32 | `ProtectedConsumers`; protection holds back the age sweep only | scope:122 |
| 33 | `Config.Logger`: the enumerated records and repeat rules | scope:123 |
| 34 | `Perm`: state files exactly Perm, no symlink following | scope:124 |
| 35 | `ConsumerTTL` semantics and caveats | scope:126, 312-325 |
| 36 | On-disk layout and frame format; normative text in format.md | scope:131-146 |
| 37 | Segments binary, everything else text; framing about 14-18 bytes per record | scope:148-153 |
| 38 | The 15 CLI verbs plus help and version, the lock rules, global `-dir` | scope:155-168; build-prompt:248-253 |
| 39 | `ls` shows size and policy, skips non-logs, `error:` row and exit 1 | scope:161-162 |
| 40 | Exit codes 0 to 3; append closes before printing; roll and prune exit 1 on Close failure | scope:168-171 |
| 41 | Only append creates a log; roll, prune, recover at a non-log exit 1 and create nothing | scope:171 |
| 42 | `prune` output: `dropped N`, per-consumer warnings, reasons on stderr, exit codes | scope:173-177 |
| 43 | `-json` shape on cat, read, tail; `-b64`; lossless | scope:179-194 |
| 44 | Control and Cf characters escaped in JSON and text; disk-derived names quoted | scope:195-205 |
| 45 | `stat`, `check`, `consumers` `-json` shapes | scope:207-224 |
| 46 | Durability: what makes a record durable; commit semantics; two fsyncs per flush | scope:226-241 |
| 47 | Measured figures on the build machine (flush cost, durable records per second, macOS F_FULLFSYNC) | scope:238-241, 372-373 |
| 48 | Durability is a setting; Sync is group commit | scope:246-251 |
| 49 | When a write fails: truncate and retry, fsync error fatal, post-durable failures retried, ceiling under 1.5 GiB | scope:253-261 |
| 50 | Disk-safety knob table; UNBOUNDED when neither `MaxBytes` nor `MinFreeBytes` is set | scope:265-285 |
| 51 | `Stat().Bytes` equals segment bytes on disk; `Stat().Records` formula; planning formula | scope:287-295 |
| 52 | When the disk is full: buffer, fail fast past `MaxBufferBytes`, recover when space returns; tested in a container | scope:301-304 |
| 53 | Everything per directory; `quill.policy` records the Config; offline verbs never rewrite it | scope:306-310 |
| 54 | Archiving semantics: `_archive` consumer, CRC-verified copy, suffix-only destination, `GapError`, exit 3 | scope:327-333 |
| 55 | Wrong clock: quarter-per-pass guard, `prune -force`, check flags future and non-monotonic records | scope:335-345 |
| 56 | Delivery semantics: at-least-once, no idempotence key, no competing consumers | scope:347-352 |
| 57 | Latency: visible after flush, one poll interval for an external Reader | scope:354-359 |
| 58 | Supported filesystems, backup order, restore and rewind | scope:361-379 |
| 59 | Performance gate and the quoted figures (Read 23-39 us, ScanAll 21 M/s, Reader 16 M/s) | scope:429-438 |
| 60 | Cursorless Reader read of the active segment is O(offset in segment) | scope:438-445 |
| 61 | Platforms: Linux and darwin on arm64 and amd64; others fail to build on purpose | scope:447-449 |
| 62 | Format stability: frozen by golden tests; change needs a new version and migration note | scope:451-455; format:3-5 |
| 63 | Non-goals (network, replication, groups, sharding, transactions, Windows, and so on) | scope:457-466 |
| 64 | Versioning: v0.y.z for ever; marker checked at open, at Commit and while waiting; migration note must stop every process | scope:468-475 |
| 65 | Roadmap: v0.1.0 Core; v0.2.0 Audited plus measured optimisation candidates | scope:477-483 |
| 66 | Release gates: tests with `-race`, fuzz, vet, staticcheck, govulncheck, tidy, baselines; LICENSE, CONTRIBUTING, changelog, Go-version policy before public | scope:485-487; decisions:62 |
| 67 | Examples: `log`, `topics`, `pipeline` | scope:489-495; examples/README:3-9 |
| 68 | Open questions: none outstanding; settled items listed | scope:497-507 |
| 69 | The consumer-mtime premise is stated as tested once T16 exists | scope:319; build-prompt:350-351 |
| 70 | Standard library only, no `require` line | build-prompt:18-19; decisions:22 |
| 71 | Latest stable toolchain in `go.mod`; clean gofmt, vet, staticcheck, govulncheck, tidy; one gate script | build-prompt:20-23; decisions:78 |
| 72 | Tests first for every guarantee; real SIGKILL; crash-simulating seam; untestable list | build-prompt:24-28 |
| 73 | Do not push or create a remote; tag only per the release procedure | build-prompt:29-30 |
| 74 | Never name or copy the origin; everything is named quill | build-prompt:31-34 |
| 75 | No LICENSE while private; one before public with CONTRIBUTING, changelog, Go policy, version stamping, build script, SBOM, archives | build-prompt:35-36; decisions:30, 62; README:105-106 |
| 76 | When ambiguous prefer the smaller, safer behaviour and record the choice | build-prompt:37 |
| 77 | Library is the root package; no file over about 600 lines; all file I/O through `internal/fsx`; the named layout | build-prompt:41-75; decisions:28 |
| 78 | Recovery rules: frame bounds, contiguity, marker as lower bound, parse forward, `ErrCorrupt` conditions | build-prompt:95-112 |
| 79 | Holes: discontinuity between segments is a hole; a middle hole is damage | build-prompt:113-115 |
| 80 | Segment files opened without following symlinks, regular only, at most 1.5 GiB; batch limit; seal by flush | build-prompt:116-118 |
| 81 | State files strict and atomic except tolerant `quill.policy`; `quill.format` required; `ErrFutureVersion` | build-prompt:119-122 |
| 82 | Create and open protocol, lock first, creation order, validate Config, stale `.tmp-*` cleanup | build-prompt:126-134 |
| 83 | Flush failure classes and rotation order | build-prompt:139-155 |
| 84 | Deletion: publish marker, fsync directory, then unlink | build-prompt:156-157; decisions:73 |
| 85 | `Sync(ctx, seq)` mechanics, `ErrBadSeq`, `ErrClosed` for waiters | build-prompt:158-161 |
| 86 | Load and recovery; `quill recover` is Open then Close; manual procedure in the guide | build-prompt:165-169 |
| 87 | Read path: first-frame ranges, skipped segments, pooled buffer, Scan payload lifetime | build-prompt:173-179 |
| 88 | Gaps: `GapError{From,To}`, re-list on a deleted segment | build-prompt:181-184 |
| 89 | A Reader takes no lock, caches sealed segments, never caches the active one; Stat and Check report only what a reader can know | build-prompt:186-195 |
| 90 | `OffsetForTime` algorithm and errors | build-prompt:203-208 |
| 91 | Consumers: offset semantics, `Commit` bounds, `Lag`, names, unreadable file protects everything, delete rules | build-prompt:212-222 |
| 92 | Retention and caps: age sweep with guard, `MaxBytes`, protection, `MinFreeBytes`, slack of one segment, `Dropped` | build-prompt:223-236 |
| 93 | `MaxRecords` and `ConsumerTTL` mechanics | build-prompt:235-239 |
| 94 | `Archive` mechanics: O_EXCL temp, CRC verify, idempotent, reports holes, stops at corruption | build-prompt:240-244 |
| 95 | CLI build rules: flags, maintenance off offline, `prune` needs policy or flags, no repair verb, `check` kinds, generated reference | build-prompt:248-260 |
| 96 | Required behavioural tests for frames, segments, append, read and state files | build-prompt:264-273 |
| 97 | T27 simulator self-test | build-prompt:277-280 |
| 98 | T12 fuzz the frame decoder and state-file parsers | build-prompt:281-283 |
| 99 | T1 crash property under injection and real SIGKILL | build-prompt:284-290 |
| 100 | T2 recovery classification | build-prompt:291-297 |
| 101 | T3 failed writes | build-prompt:298-303 |
| 102 | T4 order across read paths; T5 isolation between logs | build-prompt:304-307 |
| 103 | T6 writer and reader in two processes | build-prompt:308-311 |
| 104 | T7 gaps | build-prompt:312-315 |
| 105 | T8 offsets | build-prompt:316-318 |
| 106 | T9 consumer lifecycle | build-prompt:319-325 |
| 107 | T10 late consumer | build-prompt:326-327 |
| 108 | T11 creation and marker | build-prompt:328-331 |
| 109 | T13 disk bounds | build-prompt:332-336 |
| 110 | T14 full disk on a real tmpfs, gated | build-prompt:337-340; ci.yml:70-81 |
| 111 | T15 roll, rotation and archive | build-prompt:341-345 |
| 112 | T16 consumer aging with an injected clock; mtime assertions | build-prompt:346-351 |
| 113 | T17 size accounting | build-prompt:352-354 |
| 114 | T18 sync | build-prompt:355-360 |
| 115 | T19 backlog independence | build-prompt:361-364 |
| 116 | T21 clock guard | build-prompt:365-367 |
| 117 | T22 config and limits, including `MaxReadBytes` returning at least one record | build-prompt:368-371 |
| 118 | T23 locks and the CLI lock matrix | build-prompt:372-374 |
| 119 | T25 iterators | build-prompt:375-379 |
| 120 | T26 seek by time | build-prompt:380-387 |
| 121 | T20 readable output | build-prompt:388-392 |
| 122 | T24 benchmark baselines; regression rule (median of six, 10 percent) | build-prompt:393-398; decisions:78 |
| 123 | Phase 1 deliverables | build-prompt:404-406 |
| 124 | Phase 2 deliverables | build-prompt:407-410 |
| 125 | Phase 3 deliverables | build-prompt:411-416 |
| 126 | Phase 4 deliverables (CLI, examples, documents, audit seeds, CONTRIBUTING, CHANGELOG) | build-prompt:417-426 |
| 127 | Phase 4 CI and build-script deliverables, opt-in tiers type-checked, T14 job | build-prompt:421-424 |
| 128 | Replace or soften every reference-derived figure in review.md and scope.md before public | build-prompt:424-426; decisions:78, 98; review:3, 18-20, 61-64, 71, 94 |
| 129 | Stage 1 step 1: phases 1-4 done and handed back | build-prompt:431-432 |
| 130 | Stage 1 step 2: every gate green on a clean tree | build-prompt:433-434 |
| 131 | Stage 1 step 3: the A1 adversarial audit as a workflow | build-prompt:435-437 |
| 132 | Stage 1 step 4: fix everything that survives the skeptics | build-prompt:438-439 |
| 133 | Stage 1 step 5: tag v0.1.0 and stop; do not push | build-prompt:440 |
| 134 | Stage 2: the full house audit sequence, every finding fixed and every fix skeptic-validated, then v0.2.0 | build-prompt:442-449; decisions:63 |
| 135 | Hand-back content; line-count ceiling in review.md; `log.go` size | build-prompt:451-457; review:94; decisions:28 |
| 136 | Out of scope: sharding, sockets, groups, replication, compression, keys, encryption, mmap, Set helper, dependencies | build-prompt:460-467 |
| 137 | Decisions 13 and 15: descriptor cache or mapped read and fsync outside the lock are measured follow-ups | decisions:21, 23, 98; review:72 |
| 138 | Decision 64: `quill repair` deferred; manual procedure in the guide | decisions:72 |
| 139 | Decision 68: `quill archive` stops at a corrupt sealed segment; skipping is a later option | decisions:76; scope:482 |
| 140 | Decision 73: six scaffold documents; labels not bootstrapped; harness copy gitignored | decisions:81; .gitignore |
| 141 | Decision 88: `build/quill` refuses dirty or untagged trees, four targets, SBOM, reproducible | decisions:96; build/quill:1-30 |
| 142 | Decision 92: a real crash-without-sync run is release evidence, run by hand | decisions:100; power-loss.md |
| 143 | Decisions 12 and 36: `Dropped` is in memory; an ahead consumer protects nothing | decisions:20, 44; userguide:349-358 |
| 144 | Decision 37: the failed-log Error text left as written; no `Sync(ctx, 0)` tightening | decisions:45; flush.go:135 |
| 145 | Decisions 62, 65, 81, 89: lock-file comparison before each flush and prune; directory fsync before unlink, at rotation and after `quill.format` | decisions:70, 73, 89, 97 |
| 146 | Decisions 71, 72, 75-77: contiguity in every segment, strict JSON, plain decimals, Config validated before any file is touched | decisions:79, 80, 83-85 |
| 147 | Decisions 78, 79, 93-95, 101: marker on next-segment failure, bounded reads, pruning only what was chosen, bad-segment caching rule | decisions:86, 87, 101-103, 109 |
| 148 | Decisions 86, 87, 99, 100: offline verbs, append closes itself, `-dir` placement, payload quoting, `ReadPolicy` | decisions:94, 95, 107, 108 |
| 149 | README: what quill is, filesystem requirement, document index, Layout table, gate, CI, Go-version policy | README:3-30, 32-47, 54-84, 87-100 |
| 150 | README Status: private, pre-release; v0.y.z; one LICENSE before public | README:102-106 |
| 151 | User guide section 7: lock table, positional arguments, plain-decimal flags, offline verbs, prune warnings | userguide:266-325 |
| 152 | User guide section 9: backup, restore and rewind, archiving, moving a log, filesystems, what `check` verifies | userguide:345-415 |
| 153 | User guide section 10: recovery reporting and the seven-step manual corruption procedure, which must work as written | userguide:417-471; build-prompt:169, 418 |
| 154 | User guide sections 6 and 11: Config and Limits tables agree with the code and scope | userguide:214-264, 473-482 |
| 155 | Help text for global flags, `ls`, `stat`, `consumers`, `check` | cmd/quill/verbs.go:64-68, 85-87, 103-105, 183, 207-209 |
| 156 | Help text for `read`, `cat`, `tail` | verbs.go:69-79, 129-140 |
| 157 | Help text for `append` | verbs.go:80-84 |
| 158 | Help text for `commit`, `register`, `unregister` | verbs.go:88-102; decisions:94 |
| 159 | Help text for `prune`, `roll`, `archive`, `recover` | verbs.go:106-121 |
| 160 | `quill help` exit codes, `quill version`, the hidden `reference` verb | verbs.go:122-125, 230-311; docs/reference.md |
| 161 | CI test job on Linux and macOS: gofmt, vet, build, cross-vet, freebsd must not build, `-race` | ci.yml:17-38 |
| 162 | CI fuzz and static jobs; pins match the gate | ci.yml:40-68; gate.sh:22-23 |
| 163 | CI full-disk, examples and aggregate jobs | ci.yml:70-106 |
| 164 | PR-issue-link workflow, dependabot, CODEOWNERS; no templates promised | README:84; CONTRIBUTING; .github |
| 165 | `hack/gate.sh` is fail-closed and its steps and header | gate.sh:1-74; CONTRIBUTING |
| 166 | `build/quill`: refuses dirty or untagged, `--dev`, output wiped each run, no clock dependence | build/quill:1-40; decisions:96 |
| 167 | CHANGELOG v0.1.0 section and the allocation-pass counts | CHANGELOG:367-423 |
| 168 | CHANGELOG Unreleased lists every Stage 2 behaviour change; the tag still holds a workspace path; SECURITY says latest release only | CHANGELOG:6-262 |
| 169 | SECURITY.md: private reporting, one-week acknowledgement, scope of defence | SECURITY:5-47 |
| 170 | CONTRIBUTING rules and `hack/skew.sh` as a release check against the previous tag | CONTRIBUTING:1-60; hack/skew.sh |
| 171 | `docs/format.md` is normative and frozen by golden tests; migration note must stop or restart every process | format:3-5, 40-45, 84-132, 150-171, 225-241 |
| 172 | Introduction and executive-summary claims | introduction:31-33; executive-summary:24-48 |
| 173 | `test-ledger.md` maps every guarantee and T-number to named tests | test-ledger:1-65 |
| 174 | `untestable-without-x.md` records what cannot be tested without special hardware | untestable-without-x:1-60 |
| 175 | `benchmark-baselines.md` records six runs of every benchmark | benchmark-baselines:1-165; decisions:98 |
| 176 | Open owner decisions for the public flip: A4-3 to A4-6 and the LICENSE | stage2-owner-decisions:713-715, 807-814; supply-chain:126, 414-422 |
| 177 | Items recorded as finished without a final fix-skeptic (L1-3, L1-4, L1-10) and the unrun caps-at-seal claim | stage2-owner-decisions:1723-1756 |
| 178 | Known residuals recorded as left (Reader Stat and an empty middle segment, archive touching `_archive`, A5 recount) | check_empty_sealed_test.go:57-59; stage2-owner-decisions:1504; silent-degradation |
| 179 | Allocation pass read-8 deferred and open to the owner | audit-followups:162-170; CHANGELOG:403-405 |
| 180 | Hygiene: no internal names or workspace paths anywhere; `stage2/*` never pushed; no third-party code ships | build-prompt:31-34; CHANGELOG:254-258; supply-chain:422; decisions:30 |
| 181 | Examples run under the system temp directory; `pipeline` runs under `go test` | examples/README:3-9; ci.yml:90-91 |
| 182 | Package doc rules in `doc.go` | doc.go:1-32 |
| 183 | Test-tier gating by environment variable; opt-in tiers type-checked although no build tags gate them | backlog_test.go:88; golden_test.go:81; bench_test.go:304-308; build_script_test.go:26 |
| 184 | `hack/powertest`, `hack/topology` and `hack/skew.sh` as shipped harnesses whose behaviours and flags are promises | hack/ sources; decisions:100 |
