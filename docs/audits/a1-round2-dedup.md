# A1 adversarial audit, round 2: dedup

Date 2026-10-29, against HEAD 1da37fc. Dimension report for the dedup lens (and the test-hygiene
items its duplicate finder surfaced): 12 items, ranked by what the duplication cost (a divergence
already present first, then invariants, then pure line count). The top line and the coverage
statement are in [a1-round2-findings.md](a1-round2-findings.md). The one held item is repeated with a
recommended answer in [stage2-owner-decisions.md](stage2-owner-decisions.md).

Votes: C confirmed, P plausible, R refuted. Reproduction for a dedup item means the skeptic applied
the proposed change to a scratch copy and ran the suite (and, where the claim is about cost, the
benchmark). Anchors are to the audited tree 1da37fc. Claims not self-evident are labelled verified,
reported or inferred.

Tally: 10 fixed, 1 held, 1 refuted.

| # | id | anchor | sev | verdict | votes | disposition |
|---|----|--------|-----|---------|-------|-------------|
| 1 | A1-13 | load.go:277-296; archive.go:99-111 | medium | CONFIRMED | CC | held |
| 2 | A1-21 | internal/fsx/crash_file.go:45-156 | low | CONFIRMED | CC | fixed 8fe2264 |
| 3 | A1-17 | kill_test.go:22-34 and two more | low | CONFIRMED | CC | fixed 860c483 |
| 4 | A1-14 | flush.go:167-171; log.go:483-493; read.go:333-341 | low | CONFIRMED | CC | fixed afe3daa |
| 5 | A1-16 | load.go:413-427 and four more sites | low | CONFIRMED | CC | fixed d39a915 |
| 6 | A1-22 | cmd/quill/verbs.go:166, 203, 243, 249 | low | CONFIRMED | CC | fixed 0614afc |
| 7 | A1-9 | archive.go:95 | low | CONFIRMED | CC | fixed 48e7491 |
| 8 | A1-19 | cmd/quill/cli_test.go:35-51 and two more | low | CONFIRMED | CC | fixed 0e3b548 |
| 9 | A1-20 | archive_test.go:20 and eight more | low | CONFIRMED | CC | fixed 74b1fc8 |
| 10 | A1-18 | internal/fsx/crash_test.go | low | PLAUSIBLE | CRC | fixed e37f4ee |
| 11 | A1-24 | kill_test.go:84-87 | low | CONFIRMED | CC | fixed cba6016 |
| 12 | A1-11 | codec.go:75-85; load.go:147-155 | low | REFUTED | CRR | refuted |

## 1. A1-13: the state-file load sequence is written six times and the Archive copy disagrees (medium, CONFIRMED, held)

**Claim.** The load sequence (read state, decode, classify the error) is written out three times for
quill.format and three for quill.tail, and the third copy of the format sequence disagrees with the
other two: Archive turns a newer destination format into ErrCorrupt instead of ErrFutureVersion. One
pair of helpers, `loadFormat` and `loadTail`, removes the divergence.

**Grounds.** quill.format sites: load.go:277-287 and reader.go:91-104 (both pass ErrFutureVersion
through and wrap anything else as ErrCorrupt) and archive.go:99-111 (wraps every `decodeFormat` error
as ErrCorrupt, including the future-version one). quill.tail sites: load.go:289-296, reader.go:132-138,
log.go:413-418 (any decode error becomes ErrCorrupt "no segment files").

**Warrant.** Two copies encode the rule "a newer marker is ErrFutureVersion, other damage is
ErrCorrupt" and the third does not, so the rule lives in the call sites; a shared helper makes it
impossible for a new caller to forget the carve-out. The Archive divergence is proof that hand copying
has already drifted.

**Backing.** Decision 6 (docs/design/decisions.md:14: "A marker newer than quill knows is refused with
ErrFutureVersion"); errors.go:16-17 and docs/format.md:137 say the same (reported).

**Rebuttal.** The net line count is zero (prototype +42/-42), so this is a consistency change, not a
size one. It changes the error class Archive returns for a future-format destination (semantic). If
the owner intends "any unusable destination marker is ErrCorrupt" for Archive, archive.go keeps its
wrapping and only the load.go and reader.go pair merges. Reader.readTail's NotExist mapping and its
five-try retry, and consumer.go:109-112 (a different message), stay at their call sites. The tail
half is a looser dedup than "three copies" suggests (log.go:417 discards the decode error; the Reader
adds a retry).

**Verdict.** CONFIRMED by 2 of 2 votes. No existing test pins either class for Archive.

**Reproduction.** Verified by the auditor's test and one skeptic's re-run: an Archive destination
whose quill.format is `quill/2 000...0` returns `corrupt log: destination quill.format: quill: log
format is newer than this build: quill/2, this build reads up to quill/1`, with
`errors.Is(ErrFutureVersion)` false and `errors.Is(ErrCorrupt)` true, where OpenReader on the same file
gives ErrFutureVersion. The prototype (the helpers added to state.go and six sites switched) is +42/-42
lines, passes `go vet` and the full suite, and then reports future true and corrupt false.

**Disposition.** Held for the owner: the fix changes the error class an exported function returns.
Recommended: approve (decision 6 already says ErrFutureVersion). See the owner file.

## 2. A1-21: the crash simulator's Chmod and TryLock check handle liveness more weakly than the other methods (low, CONFIRMED, fixed)

**Claim.** `simFile.Chmod` and `simFile.TryLock` re-implement the handle-liveness check in a weaker
form than `simFile.alive`, so a handle opened before a Crash can still Chmod and take a lock
afterwards. In the same file pair `Sim.ReadDir` and `Sim.SyncDir` repeat a ten-line "resolve base to a
directory" block.

**Grounds.** crash_file.go:45-53 (`alive`: closed gives a PathError; dead or a generation mismatch
gives ErrDead) is used by `begin()` for ReadAt, WriteAt, Sync, Truncate and Stat; crash_file.go:135-143
(Chmod) and :145-156 (TryLock) test only `f.closed || f.dead`, and `f.dead` is set only for handles that
held a lock at the crash (crash.go:579-581). The block duplication is crash.go:277-286 and :371-380.

**Warrant.** One liveness predicate used by every method makes the stale-handle behaviour uniform;
today two of eight methods disagree.

**Backing.** internal/fsx/fs.go:39-58 and crash.go:28-29 ("every open File is dead" after a crash);
the ErrDead doc ("returned by every method of a File opened before a crash"). A skeptic notes the
ErrDead doc sits in crash.go, not crash_file.go.

**Rebuttal.** It is test-support code; no test is known to be wrong today; using `alive()` changes a
closed handle's Chmod and TryLock error from ErrDead to a PathError wrapping `fs.ErrClosed`; the
directory-lookup dedup saves about 16 lines and is cosmetic.

**Verdict.** CONFIRMED by 2 of 2 votes. A stale TryLock would also register the stale handle in
`s.locks`, so it could make a fresh handle's TryLock return ErrLocked (inferred).

**Reproduction.** Verified by the auditor (repro on a copy): open a file, SyncDir, `Crash(keep all)`;
WriteAt and Sync return `fsx: file handle from before a crash`, but Chmod and TryLock return nil.
Skeptics traced the code, and one replaced the check with `f.alive(OpOpen)` and ran the full suite
green.

**Disposition.** Fixed in commit 8fe2264d7cfd. Chmod and TryLock call `f.alive(OpOpen)` under the lock,
so a pre-crash handle returns ErrDead like the other methods; a regression test was added. The
ReadDir and SyncDir lookup block was deliberately left (cosmetic, proposed optional). Fix-skeptic
FIXED-VERIFIED: the test fails on the reverted crash_file.go for Chmod and, with only TryLock
reverted, for TryLock; the new test also checks that a fresh handle still takes the lock. Residual: a
closed handle now returns the PathError, not ErrDead (intended, no test compares).

## 3. A1-17: three tests spawn the helper process the same way and none kills it when an assertion fails first (low, CONFIRMED, fixed)

**Claim.** Three test functions spawn the helper process with the same ten lines, and none kills the
child when an assertion fails first, so a failing test leaves a process running and makes `go test`
wait about a minute. A shared `startHelper` with `t.Cleanup(kill)` fixes both.

**Grounds.** Spawn copies in package quill: kill_test.go:22-34 (`QUILL_HELPER=writer`), log_test.go:264-272
(`holder`), reader_test.go:359-367 (`writer`); a fourth in examples/pipeline/pipeline_test.go:26-37
(another package, cannot share). Each kills only on the success path (kill_test.go:47, log_test.go:282,
reader_test.go:424) after `t.Fatal` calls that can return first. Ack parsing is also copied
(kill_test.go:38-45, reader_test.go:393-396, hack/powertest/main.go:98-101).

**Warrant.** The boilerplate is the cause of the missing cleanup: each author wrote only the
success-path Kill; one helper registering `t.Cleanup` guarantees the kill on every exit path.

**Backing.** The tests' purpose (kill_test.go:16-18: the writer is killed at a random moment) presumes
the child is reaped; `testing.T.Cleanup` runs on Fatal.

**Rebuttal.** Only a failing run is affected and CI passes, so this is a diagnosability gain; the copies
differ in environment, so the helper takes variadic environment; about twelve lines saved.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by one skeptic and the auditor. Program: a test copying log_test.go:264-272
up to the first failure point, then `t.Fatalf` before Kill: `go test` reported FAIL after 60.4 s and
`pgrep` afterwards still listed `quill.test -test.run=^$`, the holder, which runs `select {}`
(helpers_test.go:241) and never exits.

**Disposition.** Fixed in commit 860c48397c5c. `startHelper` (spawns the helper, registers `t.Cleanup`
kill and wait) and `parseAck` in helpers_test.go replace the three spawn copies and two ack parsers; a
regression test ends a test early and checks the child was reaped. Test-only. Fix-skeptic
FIXED-VERIFIED: removing the cleanup fails the regression test; a forced early `t.Fatalf` in the lock
test ended in 0.95 s with no orphan; the helper tests pass under -race on the Linux box. Residual: the
regression test guards `startHelper` itself, not that every spawn site uses it; the pipeline example's
spawn and hack/powertest's ack parser are separate packages and remain.

## 4. A1-14: the "close the retired active descriptor once, after the last read" rule is written in three places (low, CONFIRMED, fixed)

**Claim.** The rule is hand-written in flush.go:167-171 (rotateLocked), log.go:483-493 (Close) and
read.go:333-341 (view release); one method on `activeSegment` would hold the invariant.

**Grounds.** All three implement one refcount protocol (retired, `readers == 0`, `f` non-nil, close
exactly once, nil the field); they differ only in whether they keep Close's error and whether they
test `f != nil` first.

**Warrant.** A fourth site (a future Archive or Seek holder) would copy the protocol again, and a
drift between copies already caused a bug (decision 95 records correctness-22, a nil dereference in
Close).

**Backing.** Decision 95 (decisions.md:103: "the last in-flight read's release closes it, the rule
rotation already follows (decision 43)") (reported).

**Rebuttal.** It saves no lines (net +2 in the prototype, +3 as committed) and each copy is three or
four lines, so the case rests on invariant locality; the method must be called with `l.mu` held, which
a comment can say and the compiler cannot check.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by one skeptic and the auditor: the prototype `closeIfIdle() error` with the
three sites calling it vets clean and passes the suite; `go test -race` on the Close, Pinned, Reader and
Follow tests passes.

**Disposition.** Fixed in commit afe3daaac331. `activeSegment.closeIfIdle() error` in log.go; the three
sites call it; Close keeps the close error and its outer `a.f != nil` guard. Fix-skeptic
FIXED-VERIFIED: removing the `readers > 0` clause, or making the method never close, fails the Pinned
tests (nil panic, "2 descriptors leaked") and append_test.go:211-219; the Linux race run passes.
Residual: the lock requirement is unenforced, as before.

## 5. A1-16: "list a log directory and keep the segment ids" is written as a method plus five loops (low, CONFIRMED, fixed)

**Claim.** A free `listSegments(fsys, dir)` would serve the writer load, `Reader.Stat`,
`Reader.Check` and the test helper, but it must not be used on `Reader.view`'s hot path.

**Grounds.** Production: load.go:413-427 (`Log.listSegments`), reader.go:285-294 (`Reader.view`),
stat.go:74-88 (`Reader.Stat`), stat.go:138-149 (`Reader.Check`). Tests: helpers_test.go:168-181
(`segmentFiles`, a copy of `Log.listSegments`) and reader_vanished_test.go:31-38 and :122-130, which
hand-roll the loop although `segmentFiles` exists.

**Warrant.** The loop is identical at every site except what the caller does with each id;
`segmentName(id)` round-trips `parseSegmentName` exactly (state.go:35-53).

**Backing.** The `fsx.FS.ReadDir` contract (fs.go:26: "sorted"); decisions 63 and 47 (the Reader
re-lists on every call, so `Reader.view` is hot).

**Rebuttal.** Applied to `Reader.view` the change is a measured regression: on 1,000 segments
`BenchmarkReaderReadStartThousandSegments` went from 2,060 to 2,073 allocations per operation and from
about 360 KB to 395 KB (+13 allocations, +10 percent bytes, +3 to +7 percent time; two runs on macOS
arm64, reported by the auditor), so that site stays; the saving is about six production lines plus
fourteen test lines. `Reader.Stat` keeps its per-file continue-on-Lstat-error and `Check` its
`id > tl.Seg` filter in the caller.

**Verdict.** CONFIRMED by 2 of 2 votes; the skeptics did not re-run the benchmark (the fix leaves
`Reader.view` untouched).

**Reproduction.** Verified by one skeptic: the proposed refactor (all sites except `Reader.view`)
vets clean and passes `go test`.

**Disposition.** Fixed in commit d39a91530c94. `listSegments(fsys, dir)` is a free function in load.go
used by the writer load, `Reader.Stat`, `Reader.Check` and `segmentFiles`; reader_vanished_test.go now
calls `segmentFiles` and `segmentName(id)`; `Reader.view` is unchanged apart from a comment saying why;
the `slices.Sort` is kept. Fix-skeptic FIXED-VERIFIED: dropping the newest id from the listing fails
dozens of tests. Residual: check_vanished_test.go:12-24 (`segmentPaths`) is still a hand-rolled copy of
the same loop (a follow-up); the `Reader.view` benchmark and the Linux box were not re-run.

## 6. A1-22: the special verbs are string literals in four places, not fields of the verb table (low, CONFIRMED, fixed)

**Claim.** cmd/quill names `help`, `version` and `reference` as literals at verbs.go:166 (no `-dir`
needed), :203 (the help list skips `reference`), :243 (the reference table skips `reference`) and :249
(the per-verb sections skip all three), though the file says the table drives help, the reference and the
tests. The text at :237 is a fifth statement of the same fact (and omits `reference`).

**Warrant.** Adding a verb that needs no `-dir` means finding four literals; missing the one at :166 gives
a usage error `quill: X: -dir is required`.

**Backing.** verbs.go:21; decision 53.

**Rebuttal.** It saves no lines and fails loudly (exit 2 with a clear message); docs/reference.md is
generated from the table and a test keeps it current (cli_test.go:53-80), so a rewrite must leave the
generated bytes identical.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by one skeptic: the proposed `noDir` and `hidden` fields applied to a copy;
`go build`, `go test ./cmd/quill` pass and `quill reference | diff - docs/reference.md` is empty.

**Disposition.** Fixed in commit 0614afc5877a. Unexported `noDir` and `hidden` fields on the verb struct
(help, version and reference are `noDir`; reference is `hidden`) replace the four literal checks.
Fix-skeptic FIXED-VERIFIED: dropping `noDir` or `hidden` from the right verbs fails the reference-currency
test; `quill reference` and `quill help` are byte-identical before and after. Residual: the prose
sentence "Every verb but `help`, `version` and `ls` takes `-dir`" is still hand-written (the fifth
statement, not derivable from `noDir` alone because `ls` is special); `noDir` also suppresses the
per-verb section, which would drop a future no-dir verb that has flags.

## 7. A1-9 (with A1-23): Archive duplicates the directory-mode formula and threads a `perm` that is always 0o600 (low, CONFIRMED, fixed)

**Claim.** archive.go:95 `dirPerm := perm | (perm&0o444)>>2` repeats log.go:172-174 (`Config.dirPerm`),
and `archive`, `archiveDest` and `copySegment` carry a `perm` parameter that every call sets to 0o600
(archive.go:25 and all test call sites).

**Warrant.** With one value in every call the parameter can be `DefaultPerm` (log.go:82) and the
directory mode can use the shared helper; a duplicated bit trick could drift.

**Backing.** Config.Perm default 0600 and the matching directory mode (scope.md Configuration table).

**Rebuttal.** `archive()` is the unexported seam taking the filesystem and the parameter may be meant for
a future option; nothing does so today; about three lines plus the call-site arguments. A skeptic counts
17 test call sites, not the 10 listed.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by one skeptic: the three signatures changed and `Config{Perm:
DefaultPerm}.dirPerm()` used in a copy; `go vet` and `go test .` pass.

**Disposition.** Fixed in commit 48e7491ee886 (an earlier attempt left stray backup files; the commit
contains only the three Go files). The parameter is gone and `archiveDest` uses `Config{Perm:
DefaultPerm}.dirPerm()`. Fix-skeptic FIXED-VERIFIED: mutating the directory mode to 0 fails
`TestArchiveOnTheRealFileSystem`; staticcheck clean. Residual: no test asserts the archive's file modes
(a 0o666 mutation survives; pre-existing).

## 8. A1-19: four test fixtures in cmd/quill build a log the same way (low, CONFIRMED, fixed)

**Claim.** `newLog`, `manySegments`, `bigActive` (and an inline copy at cli_test.go:243) each Open, append
n records, Close and return the directory with different Configs; `newLog` could take the Config.

**Grounds.** cmd/quill/cli_test.go:35-51, numbers_test.go:40-55, offline_policy_test.go:17-34; the
duplicate finder reports the closing `l.Close()` block at cli_test.go:47, numbers_test.go:52,
offline_policy_test.go:31.

**Warrant.** They differ only in Config and payload; about 25 lines would go.

**Backing.** None (test code).

**Rebuttal.** `newLog` has many callers, so it stays as a wrapper; the three special fixtures keep their
documented intent in their names. A skeptic found the inline copy at cli_test.go:243 is not part of the
pattern, so three fixtures are the whole duplication.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by one skeptic: the proposed `buildLog` applied to a copy; `go vet` and `go
test ./cmd/quill` pass; 707 to 683 lines.

**Disposition.** Fixed in commit 0e3b5481efb8. `buildLog(t, cfg, n, payload)` in cli_test.go; the three
fixtures are thin callers with their original Configs (net -17 lines). Fix-skeptic FIXED-VERIFIED
(staticcheck and deadcode clean; the Config values match field for field). Residual: the inline copy at
cli_test.go:252 is left; the mutation check of `buildLog` was denied by the sandbox, so that the tests
would catch a broken helper is inferred.

## 9. A1-20: the retention and archive tests start the same four lines nine times (low, CONFIRMED, fixed)

**Claim.** `fsx.NewSim()`, a fake clock, `openRetention`, `defer Close`, repeated at archive_test.go:20,
112, 141, 182, 211, 301, seek_test.go:121 and retention_test.go:88, 179.

**Warrant.** A one-call fixture returning `(s, clk, l)` saves about two lines per site (a skeptic counts
2, not the 3 claimed).

**Backing.** None (test code).

**Rebuttal.** About 25 lines; several tests close and reopen mid-test, so a fixture registering its own
cleanup could double-close; a reviewer may prefer explicit setup in tests whose subject is retention. The
proposed helper registers no cleanup.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by reading all nine sites (identical apart from the mutator argument).

**Disposition.** Fixed in commit 74b1fc853a77. `retentionLog(t, mut) (*fsx.Sim, *testClock, *Log)` in
retention_test.go, used at 11 sites (the nine plus two of the same shape); no cleanup registered.
Fix-skeptic FIXED-VERIFIED (staticcheck clean). Residual: dropping the helper's `mut` argument changes
nothing, so those tests do not depend on their mutators (pre-existing).

## 10. A1-18: crash_test.go has `mustOpen` and `mustWrite` but no `mustSync` or `mustSyncDir` (low, PLAUSIBLE, fixed)

**Claim.** The fatal-on-error `f.Sync()` is spelled out 13 times (lines 58, 106, 124, 154, 162, 188, 191,
213, 235, 277, 299, 379, 522) and the `SyncDir("/")` twin 10 times.

**Warrant.** Every site is the same three-line statement; a helper turns each into one line.

**Backing.** Consistency with `mustOpen` and `mustWrite` (crash_test.go:15-30); test code, no decision.

**Rebuttal.** About forty lines; some tests spell out the durability steps on purpose
(`TestFsyncOfAFileMakesItsDataDurableButNotItsDirectoryEntry`), so the helper belongs only at setup
sites; not worth a separate commit.

**Verdict.** PLAUSIBLE (votes C, R, C). The duplication is real; the SyncDir count is overstated (lines 365
and 393 assert an expected error, so 8 sites, not 10); the REFUTED skeptic holds it is purely cosmetic and
blurs the tests whose subject is which call makes what durable.

**Disposition.** Fixed in commit e37f4ee48720. `mustSync(t, f)` and `mustSyncDir(t, s)` beside
`mustWrite`, used at 9 Sync sites and 5 `SyncDir("/")` sites that are setup steps (net -14 lines).
Fix-skeptic FIXED-VERIFIED (making the helpers no-ops fails five crash tests; staticcheck and deadcode
clean). Residual: lines 267, 273 and 300 look like setup steps and were not converted; the failure
message changed to `sync: ...`.

## 11. A1-24: kill_test.go ends with two statements that only keep imports alive (low, CONFIRMED, fixed)

**Claim.** kill_test.go:84 `_ = fmt.Sprint` and 85-87 `if !errors.Is(err, nil) { t.Fatal(err) }` exist only to
keep the `fmt` and `errors` imports; the second can never fail.

**Grounds.** At that point `err` is the value checked at line 77, which is nil; the later `l.Close()` error is
scoped to its own `if`. `fmt` and `errors` are used nowhere else in the file.

**Warrant.** `errors.Is(nil, nil)` is true, so the Fatal is unreachable; the lines exist to satisfy the
compiler.

**Backing.** The general rule that no code is kept only to silence an unused import.

**Rebuttal.** Harmless and outside the duplication lens; listed because it sits in a file read for the
child-process cluster.

**Verdict.** CONFIRMED by 2 of 2 votes (traced by hand, not run).

**Disposition.** Fixed in commit cba6016e5419 (six lines deleted, test-only). Fix-skeptic FIXED-VERIFIED.
Residual, flagged for a follow-up: the same keep-an-import-alive pattern remains at retention_test.go:617-618
(`_ = context.Background`, `_ = bytes.Equal`) and bench_test.go:336 (`_ = fmt.Sprint`); they were outside
the claim's stated file.

## 12. A1-11: the uvarint length-prefix classification is written twice (low, REFUTED)

**Claim.** The classification of `binary.Uvarint(hdr)` (bad, torn or ok) is written in `decodeFrame`
(codec.go:75-85) and in `window.frameAt` (load.go:147-155); a helper `frameHeaderLen` would serve both.

**Grounds.** n<0 or length>maxFrameBody gives errBadFrame; n==0 with at least `maxVarint` bytes available gives
errBadFrame, else errTornFrame; `frameAt` feeds it `avail` bytes, `decodeFrame` the whole slice.

**Warrant.** About six lines saved and the two sites cannot drift.

**Backing.** Decision 39, guarantee 7; docs/format.md frame section.

**Rebuttal.** `frameAt` needs the length again for `total`; the code sits on the hottest read path and the
helper must stay inlinable; the duplication is eight lines and well commented.

**Verdict.** REFUTED (votes C, R, R). Two skeptics built the helper in a scratch copy and the compiler reports
it cannot be inlined (`cost 115 exceeds budget 80`, and cost 119 for the other build), so each frame read would
pay two extra calls on the hot path; the file grew from 602 to 604 lines; the two sites agree today and no
defect was found. The confirming skeptic accepted the duplication but not the value.

**Disposition.** Not changed: refuted; the proposed helper is not inlinable and saves no lines.
