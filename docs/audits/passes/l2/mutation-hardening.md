# Pass L2: test, benchmark and hardening suite

Date 2026-11-03. Recorded against the tree at `f274b51`, which has every L2 fix applied. The auditors worked on an earlier
tree (the recovery auditor names `5a70379`; the others say "HEAD" without a hash), before the pass L1 fixes landed, so some
"uncovered" statements below are older than the tree (see the coverage statement). The specification is
[scope.md](../../../design/scope.md), [decisions.md](../../../design/decisions.md) and [format.md](../../../format.md). The code is what is under
audit: a passing test is not evidence that anything is right, and a doc is not evidence that the code does what it says.

Pass L2 asked one question: would the test suite notice if the code were wrong? It was put four ways. A coverage ledger on two
platforms, three hand-written mutation runs over the whole library and the command line tool, and an invariants-and-fuzz run
that tried to break the resource bounds and the isolation rules from outside. Files in this record:

| File | What it holds |
|---|---|
| this file | the summary, the coverage statement, the method, the mutant tables with scores, survivors judged, residual weaknesses, and one record per item |
| [test-ledger.md](../../reference/test-ledger.md) | the refreshed coverage ledger for both platforms, and the tests this pass added |
| [defects-log.md](defects-log.md) | each defect found: root cause, fix, regression test |
| [harness-inventory.md](../../reference/harness-inventory.md) | every harness and fixture, and what it can and cannot simulate |
| [untestable-without-x.md](../../reference/untestable-without-x.md) | what still cannot be tested, with the exact blocker and the artifact that would unblock it |
| [stage2-owner-decisions.md](../../decisions/stage2-owner-decisions.md) | L2-53, which was held: the recommended answer, the owner's approval of 2026-11-03 and the fix |

## Summary

| Counted over the 53 items | |
|---|---|
| items | 53 filed after merging 5 duplicates (58 raw ids, L2-1 to L2-58; merged: L2-54 into L2-1, L2-5 into L2-25, L2-7 into L2-17, L2-23 and L2-40 into L2-8). 52 are test gaps and 1 is a bug (L2-53, a resource bound) |
| by verdict | 47 CONFIRMED, 6 PLAUSIBLE (L2-11, L2-12, L2-55, L2-56, L2-57, L2-58), 0 REFUTED. Three individual votes were REFUTED and are kept with their reason on L2-55, L2-57 and L2-58 |
| by severity as filed | 7 high, 27 medium, 19 low. One voter on L2-24 would lower its severity to medium |
| by disposition | 52 fixed, 1 held for the owner (L2-53), 0 deliberately not changed. 0 fix-failed: the ledger records one fix per item and no rejected attempt. **Updated 2026-11-03:** the owner approved the recommended answer for L2-53 and it is fixed in 637ed42, FIXED-VERIFIED (7 of 7 lenses pass, reported by the skeptic), so 53 fixed and 0 held. The counts in the other rows are the state when the pass was recorded |
| fix-skeptic outcomes | 52 of 52 FIXED-VERIFIED, 6 lenses each (test fails without the fix, root cause, behaviour preserved, no new gap, reachable, ladder), 312 lens verdicts, all pass (reported by the skeptics). Added 2026-11-03: L2-53's fix, FIXED-VERIFIED with 7 lenses, all pass, makes 53 of 53 and 319 lens verdicts |
| finding-skeptic votes | 109: 101 CONFIRMED, 5 PLAUSIBLE, 3 REFUTED (50 items had 2 votes, L2-55, L2-57 and L2-58 had 3) |
| reproduced by execution | 100 of the 109 votes reported a reproduction; 52 of 53 items had at least one reproducing vote. The exception is L2-57, an absence claim checked by grep only. Tallied by the recorder from the ledger (inferred from it, not re-run) |
| mutants run | 530 valid mutants over four runs: 177 (write path), 195 (recovery and retention), 141 (reader and CLI), 17 (coverage run). About 24 first drafts that did not compile were rewritten and are not counted |
| kill tests | the auditors' kill tests lived in scratch only; the fixers wrote 52 commits of fresh tests (134 added `func Test` and `func Fuzz` lines in the diffs, which would also count a function whose header line was edited), plus changes to existing tests. Each fix-skeptic re-applied the named mutants to the fixed tree and watched the new tests fail |
| production code changed | five files, none a behaviour change: `disk.go` (an unexported `availBytes` extracted), `state.go` (an unexported `policy.config` extracted), and comment or doc wording in `log.go`, `retention.go` and `cmd/quill/consumers.go` (L2-31). `docs/design/decisions.md` (decision 45), `docs/design/scope.md` and `CHANGELOG.md` carry the same wording change |

### Mutation scores, before and after

Detected means killed by a failing assertion, a crash or a reproduced hang. "Equivalent" mutants are excluded from every
denominator. "Unobservable" mutants are kept in the denominator of the first two rows and of the like-for-like line of the third.

| Run | Valid mutants | Detected before | Equivalent (excluded) | Score before | Real gaps | Unobservable | Score after |
|---|---|---|---|---|---|---|---|
| Write path | 177 | 129 (121 killed + 8 by a reproduced hang) | 15, plus 1 no-op probe | 129 / 161 = 80.1% (79.1% without the 8 hang kills) | 26 | 6 | 155 / 161 = 96.3% |
| Recovery, retention, consumers, stat, archive, strictjson | 195 | 110 | 12 | 110 / 183 = 60.1% | 63 (53 root package, 10 strictjson) | 10 | 173 / 183 = 94.5% |
| Reader, iterators, seek, CLI | 141 | 77 (74 killed + 3 by hang) | 14 equivalent, 9 unobservable | as reported 77 / 141 = 55%; like-for-like 77 / 127 = 60.6% | 39 testable, 2 real but not deterministically testable | 9 (kept out of the reported 118) | as reported 116 / 141 = 82%, and 116 / 118 = 98% of the non-equivalent, non-unobservable; like-for-like 116 / 127 = 91.3% |
| Coverage run (hand-picked from uncovered code) | 17 | 0 (every one survived) | M21 judged unreachable and M10 (not among the 17) judged redundant, both inferred, below | 0 / 17 | 13 tied to an item by number (M1-M6, M11, M12, M14-M17, M20) | M8, M9, M21, M22 not tied to an item | not scored: the 13 are the targets of L2-2, 3, 4, 6, 9, 10, 11 |

How the "after" figures were obtained (reported, not re-run by the recorder): the auditors' own kill tests were verified in
scratch to fail on each real-gap mutant. The committed tests are different code, written by the fixers; each fix-skeptic then
applied the named mutants to the fixed tree and saw the new tests fail. So every "after" kill rests on a skeptic's run of the
committed test, except where a fix left a mutant alive on purpose, which is listed under "Residual weaknesses" below. The scores
are line-level hand mutation, not a mutation tool's output, and each row is a single textual edit judged by one auditor.

## Coverage statement

**Auditors.** Five returned: the coverage ledger, the three mutation runs (write path, recovery and retention, reader and
CLI) and the invariants-and-fuzz run. None failed to return.

**Checked and found clean** (reported by the auditors, one line each; the recorder re-ran none of it):

- *Coverage ledger.* Every test name in the old test-ledger.md resolves to an existing test function (372 names scanned); T1 to T27
  each have a test; T14 is skipped by default on both platforms as the ledger says. Function-level coverage showed the durability
  path of the writer (flushLocked ordering, rotateLocked, publishLocked), deleteLocked's publish-before-unlink and SyncDir-before-unlink,
  recoverSegment and verifySegment, `lock_unix.go` (all 100%), the consumer state writes and their revert on failure, atomicfile.Write,
  strictjson, capsLocked's MinFreeBytes accounting and the CLI's archive-gap exit 3 mapping, all at or near 100% with only trivial
  branches uncovered. The Linux and macOS merged profiles differ in exactly two blocks (see test-ledger.md).
- *Write-path mutants.* All mutants were detected for: CRC coverage and byte order on both sides; frame length and varint arithmetic;
  the exact-payload ceiling, non-minimal varints, the slack check; the tail encoder and decoder; the format and consumer-file decoders;
  the flush protocol (truncate-back, truncate failure fatal, data fsync removed or moved after the marker, fsync error fatal, head
  arithmetic, idle publish stamping, lock-lost detection, sticky failure, rotation marker, directory fsync after a new segment, Roll
  guards, Sync boundaries, the flusher's wake-ups); append limits (payload and sequence ceilings, diskFull, clock units, bufFirst);
  lock_unix's retry window and classification; atomicfile's chmod, fsync, rename, temp cleanup and name shapes; fsx.OS's
  O_EXCL, O_CREATE, O_NOFOLLOW, O_NONBLOCK, SyncDir NOFOLLOW, TryLock mapping, ReadDir ordering, Rename wrapping, SameFile; the crash simulator
  itself (a Sync that leaves pages dirty, a SyncDir that persists nothing, a Truncate that is not size-dirty each break tests); Open and
  Close paths (lock release on create, load and protect failures, stale temp removal, parent fsync, policy write rules, setBg logging
  once per failure, the ahead-of-head warning boundary, config defaults and boundaries).
- *Recovery-run mutants (110 kills).* walk() sequence rules, anchor index boundaries and stride, the empty-segment rule, durable
  greater than size, I/O error propagation, the truncate step and three reorderings of it, marker republish, sealed-segment read errors
  failing Open (decision 79), contiguity(), age-sweep break and continue, damaged-segment ride-along, protection-loop conditions, the
  clock guard's quarter and force flag, bgPrune, MaxBytes and MaxRecords loop bounds, publish-before-unlink, ErrNotExist tolerance,
  Commit range bounds, log-id re-check, write-error revert, protect flag handling, ProtectOn and ProtectOff, name validation,
  strictjson utf8, duplicate, unknown-field, trailing and fold checks, archive offset and skip conditions, lock refusal, the
  empty-destination rule, stale-temp sweep and per-segment verify. holes.go (5 of 5) and validName (5 of 5) were fully killed.
- *Reader and CLI mutants (74 kills).* Reader id and ErrWrongLog, first-frame and active-segment bounds, deleted-between-listing-and-open gaps,
  cache pruning, MarkerAge sign, index-scan error caching, locate and gap ranges, limit and byte accounting, copy-on-Read, index start,
  Log.ReadWait on a failed log, OffsetForTime bounds, CLI number parsing, usage and exit-code mapping, which verbs open with
  NoMaintenance, output escaping and the -json shape, tail -follow flush and cancel.
- *Invariants.* Resource bounds the auditor could not violate: the failing-flush buffer cap (50 seeds x 400 ops with random ENOSPC), the
  MaxBytes disk bound (120 seeds x 600 ops, worst ratio 0.652), MaxRecords (150 seeds), descriptors per Reader call and per Follow, goroutines and
  descriptors after Close, the Reader segment cache. Isolation: reads never take the writer's lock (grep, and against a live writer the real binary's
  stat, read, cat, tail, consumers, check, ls, commit, register, unregister and archive exited 0 while append, roll, recover and prune exited 1 `locked by
  another holder`); offline CLI verbs never rewrite quill.policy (three logs: policy present, absent, corrupt; inode, mtime and hash identical); a Reader never
  shows beyond quill.tail (ten valid frames and a whole segment planted past the marker, through Read, ScanAll, Scan, Records, Follow, Head, Stat,
  OffsetForTime, Consumer.Records and Commit). Hostile state files (1 MiB of nested objects, arrays, 38k keys, 524k `\u` escapes) were refused in at most 50 ms
  and 27 MB. Printer records with `At` at int64 extremes print without a panic (the wrap for MinInt64 is already in threat-model.md).
  Fuzz runs that found nothing: a 300 s differential of decodeTail against decodeTailSlow (4,809,478 execs), and shorter runs of
  FuzzFrameStrong, FuzzWalkChunks, FuzzReaderDir, FuzzPolicyConfig, FuzzParseTime and FuzzPrinter.

**Not covered.**

- Windows (not supported). No run used the Apple container (linux/arm64).
- The coverage numbers are line coverage at one commit on each platform, no `-race`. T14 (the full disk) and the benchmark tier were not run.
  The 20 s fuzz targets in the gate were not run by the coverage auditor; the invariants run fuzzed for minutes, not hours.
- Mutants were line-level hand edits: single textual edits to one file. The write run was full on macOS only, with a four-mutant Linux sample
  (O05, O07, T05, L27, same verdicts). The recovery run used Linux for 146 mutants and macOS for 49. The reader and CLI run used macOS for all
  and Linux for three survivors (R04, S21, C42, all survived there too). No run mutated `examples/` or `hack/`.
- The write run did not mutate the cmd/quill write path, read.go, load.go, retention.go, consumer.go or archive.go; the recovery run left out about 34
  lower-value mutants (for example caps' diskFull assignment and freeAtPass, ret-first-head, validConsumerName callers in open and remove, consumer lock
  release, Stat.Segments, Stat.Dropped copy, MarkerAge), and did not mutate state.go, read.go or the writer paths outside its eight files; the reader and CLI run
  read but did not mutate cmdArchive's argument errors, cmdHelp and cmdReference text, tabwriter formatting, the stat text form, and Reader.Stat, Check and
  Consumers. Absence from this record is not evidence these are caught.
- Archive's differential and the FuzzReaderDir run used the in-memory simulator, not real ext4 or APFS semantics; the descriptor counts are a macOS `/dev/fd` measure.
- The auditors' base predates the pass L1 fixes. For example, `Reader.ID`, `Reader.Scan`, `Consumer.Name`, `Consumer.Scan` showed 0.0% in the coverage
  ledger; `7497a80` (pass L1, "test Reader.Scan, Reader.ID, Consumer.Scan, Consumer.Name and the real freeBytes") added tests that call them
  (reader_test.go lines 724-744, read by the recorder, not run). The 0.0% figures in test-ledger.md are therefore as of the audited base.
- Not re-run after the 52 fixes: the coverage numbers. Command: `GOWORK=off go test -count=1 -coverpkg=./... -coverprofile=c.out ./... && go tool cover -func=c.out`.
- The recorder ran nothing in this pass (docs only). The commit SHAs, test names and file lines were read from this tree with `git log` and `git show`.

## Labels

*Verified* means the recorder read the exact line or commit in this tree. *Reported* means an auditor or voter said it and ran it; the source is named. *Inferred*
means built from other claims, with the grounds named. Anything about a mutant run, a probe program or a fuzz run is reported, not verified, because the recorder
ran none of it. Where the record says "the fix-skeptic applied the mutant", that is the skeptic's run of the committed test.

## Method

**Write-path run.** Hand-written textual mutants over log.go, flush.go, append.go, state.go, codec.go, lock_unix.go, internal/atomicfile and internal/fsx. Each
mutant is one edit to one production file; `go build ./...` must pass; then `GOWORK=off go test -count=1 -timeout 150s -failfast` over the root and `./internal/...`, then
`./cmd/quill` if those passed. 177 valid mutants; seven first drafts did not build and were rewritten (L18, F10, F11, F38, S19, K03, O12). Eight kills are reproduced
hangs under the 150 s timeout, run twice under different loads (F04, F12, F40, F41, K03, O09, R13, R17).

**Recovery run.** The same runner shape over load.go, holes.go, retention.go, consumer.go, validName in log.go, stat.go, archive.go and internal/strictjson (195 valid
mutants, 13 first drafts rewritten). Linux x86-64 for 146 of them and macOS arm64 for 49, with every macOS kill by a flaky-test line re-run on Linux.

**Reader and CLI run.** A fresh copy of the tree per mutant, one textual replacement, `go build`, then `GOWORK=off go test -count=1 -timeout 150s . ./cmd/quill` (only `./cmd/quill` for CLI mutants),
five in parallel on a loaded ten-core macOS box. 141 mutants after rewriting four. Three kills (I01, S04, S16) were classified from a hang, not from a named assertion.

**Coverage run.** Seventeen hand-picked mutants (the number the auditor states; the ids run from M1 to M22, and M7, M10, M13, M18 and M19 are not among the 17 candidates, though M10 is reported as surviving) chosen from the uncovered-branch list, run in a scratch copy of the tree made with `git archive`; the copy was restored and diffed after every mutant.

**The load-flake trap (reported; this matters for the next mutation run).** `TestOpenWithAProtectedConsumerRunningDoesNotNeedItsLock` asserted `d > lockRetry/2` and
fails under load. In the write run, seven mutants run in parallel reported 119 killed; 28 of those kills were false, all by this test. In the reader run, 14 first-run
kills (R06, R26, S09, S20, S21, S23, S24, S28, S30, I05, K08 and others) were the same test. Every kill was re-run alone under the killer test or with that test skipped;
the counts here are after that correction. L2-8 removed the assertion (commit `743a5c5`), so a later run should not need the `-skip`, but this is inferred: the recorder did not re-run a mutation pass.

**A second trap (reported).** The tool harness decodes six-character unicode escape text in tool input into the characters themselves, which turned the first strictjson test
rows into valid text. The final strictjson tests build escapes at run time (`string(rune(92))+"u"+hex`); anyone adding such rows by hand should do the same.

## Mutant tables per area

Mutant ids are per run, so the same label can name different edits in different runs (write R04 is a directory-fsync handling edit, reader R04 is `Reader.wait`). The
tables prefix each id with its run. The line numbers cited in the Grounds of each item are those of the audited base and have shifted in the tree.

### Table A. Write path (log.go, flush.go, append.go, state.go, codec.go, lock_unix.go, internal/atomicfile, internal/fsx)

| Item | Mutants (edit) | Killed by (tests now in the tree) | Fix commit |
|---|---|---|---|
| L2-12 | write: A14 `return first, l.failed` -> `return first, nil` (append.go) | `TestAnAppendWhoseInlineFlushFailsTheLogReturnsTheFailure` | `2fc3030` |
| L2-13 | write: L20 remove Close's final SyncDir | `TestCloseMakesTheMarkerDurable` | `3f77076` |
| L2-14 | write: F19 `MaxSeg: l.maxSeg` -> `l.active.id` (flush.go) | `TestKillMarkerKeepsMaxSegAboveTheActiveSegment` | `ab45579` |
| L2-15 | write: R04 (write run) turn the fatal handling of the post-marker SyncDir into `return err` | `TestDirFsyncErrorAfterTheRotationMarkerFailsTheLog` | `55a7e97` |
| L2-16 | write: F39 drop `\|\| l.pendingPublish` from the flusher's condition | `TestKillFlusherRetriesAFailedMarkerPublishOnAnIdleLog` | `216c111` |
| L2-17 | write: R11 delete the closed guard in Flush; R14 delete `case l.closed` in Sync (merged L2-7) | `TestFlushAfterCloseIsErrClosedAndDoesNotWrite`, `TestSyncAfterCloseOfARecordThatDidNotBecomeDurableIsErrClosedAtOnce` | `7716100` |
| L2-18 | write: A01 `+4` -> `+3`; A02 `size > MaxBatchBytes` -> `>=`; A06 `len(l.buf)+size > MaxBufferBytes` -> `>=`; A08 `end > ceiling` -> `>=`; A09 `len(l.buf) >= FlushBytes` -> `>`; F17 `durable >= SegmentBytes` -> `>` | `TestABatchIsAcceptedAtExactlyMaxBatchBytesAndRefusedAtOneByteMore`, `TestWhileFlushesFailAnAppendIsAcceptedAtExactlyMaxBufferBytesAndRefusedAtOneByteOver`, `TestAFailedRotationAcceptsAnAppendReachingExactlyTheCeilingAndRefusesOneByteMore`, `TestAnAppendThatFillsTheBufferToExactlyFlushBytesFlushesInline`, `TestASegmentRotatesAtExactlySegmentBytesAndNotAtOneByteUnder` | `0c08d71` |
| L2-19 | write: L03 `FlushInterval < time.Millisecond` -> `<=`; L05 `maxActiveBytes() >= maxSegmentFileBytes` -> `>`; L07 `+34` -> `+33`; L16 `info.Size() > 0` -> `> 1`; S08 slow-path `out.Off > maxSegmentFileBytes` -> `>=`; S17 `info.Size() > maxStateBytes` -> `>=`; S20 `maxSegmentID = 9_999_999_999` -> `...998` | `TestOpenWithoutAMarkerRefusesAOneByteSegmentAndCompletesAnEmptyOne`, `TestConfigRefusesAFlushThatReachesTheFileLimitExactlyAndAcceptsOneByteLess`, `TestConfigFlushIntervalOfOneMillisecondIsValidAndBelowIsRefused`, `TestTailDecodersAcceptTheInclusiveLimitsAndRefuseOneMore`, `TestReadStateAcceptsAFileOfExactlyMaxStateBytes` | `dfc1a9c` |
| L2-20 | write: K06 delete the Close on a non-ErrLocked TryLock error; K07 delete the Close before the ErrLocked return (lock_unix.go) | `TestAnOpenRefusedByALiveWritersLockLeaksNoDescriptor`, `TestAnOpenWhoseLockFailsWithAnotherErrorLeaksNoDescriptor` | `bcb3266` |
| L2-21 | write: T09 turn the Chmod guard into `if true`; T06 `false && err == nil` on the Close error (atomicfile.go) | `TestWriteFailsWhenChmodOrCloseFails` | `16207b1` |
| L2-22 | write: F14 `pos >= a.nextAnchor` -> `>`; F15 `a.nextAnchor = pos + indexStride` -> `+ indexStride + 1` | `TestWriterIndexEqualsTheScanIndex` | `8a81064` |


### Table B. Recovery, retention, consumers, stat, archive and strictjson

| Item | Mutants (edit) | Killed by (tests now in the tree) | Fix commit |
|---|---|---|---|
| L2-24 | recov: load.go `info.frames > 0` -> `false` in the tail walk | rows added to TestNothingAfterTheFirstBadFrameIsTrustedWhateverItContains | `dac7bfb` |
| L2-26 | recov: retention.go `c.Offset < cv.minProtected` -> `>`; `lastOfLocked(i) >= minProtected` -> `>`; damaged-segment bound without `- 1`; `cv.broken = true` -> `false` | `TestTheSlowestProtectedConsumerHoldsTheAgeSweepInEitherFileOrder`, `TestAProtectedConsumerAtTheLastRecordOfASegmentKeepsThatSegment`, `TestADamagedSegmentBelowAProtectedConsumersOffsetIsDeletedWithTheExpiredOnes`, `TestAnUnlistableConsumersDirectoryProtectsEverySegment` | `d29328e` |
| L2-32 | recov: consumer.go `state.Next <= head+1` -> `<`; drop `state.ID == st.id` | `TestOpenWithACaughtUpProtectedConsumerRunningDoesNotNeedItsLock`, `TestOpenWithAProtectedConsumerFileOfAnotherLogFails` | `c824158` |
| L2-33 | recov: consumer.go `\|\| err != nil` on the not-exist case; drop `c.state.Next = prev`; drop `HasPrefix(n, tempPrefix)`; `Next > head+1` -> `> head+2` | `TestAnUnreadableConsumerFileIsAnErrorAndNeverResetsTheOffset`, `TestAFailedCommitLeavesTheHandlesOffsetWhereItWas`, `TestALeftoverTempFileInConsumersIsNotAConsumerAndDoesNotStallRetention`, `TestAConsumerTwoAheadOfTheHeadIsRefusedAndOneAheadIsCaughtUp` | `143367d` |
| L2-25 | recov: load.go drop `head = tl.Seq`; `leadBad = leadBad \|\| len(good)==0` -> `len(good)==0`; drop `trailBad = false`; `else if leadBad` -> `false`; `head < tl.Seq` -> `head+1 < tl.Seq`; stat.go `h.To <= v.head` -> `<` (merged L2-5) | `TestADamagedNewestSealedSegmentBeforeAnEmptyActiveOneOpensWithATrailingHole`, `TestEverySealedSegmentDamagedAndAnEmptyActiveOneIsOneHoleFromOne`, `TestSeveralDamagedSegmentsKeepTheirHolesAndTheLeadingOne`, `TestAMarkerAboveTheLastValidFrameIsRefusedAfterAnEarlierDamagedSegment` | `d177d2c` |
| L2-27 | recov: retention.go `range l.sealed[n:]` -> `range l.sealed`; `bytes > MaxBytes` -> `>=`; loop `free < MinFree` -> `<=`; `full := free < MinFree` -> `<=` | `TestMaxBytesCapCountsOnlyWhatTheAgeSweepLeaves`, `TestMaxBytesKeepsASetExactlyAtTheCap`, `TestMinFreeBytesStopsDeletingOnceTheFloorIsMetExactly`, `TestMinFreeBytesIsMetAtExactlyTheFloor` | `b040221` |
| L2-28 | recov: retention.go `bytes := l.active.durable` -> `int64(0)`; drop `+1` in `head-first(n)+1 > MaxRecords` | `TestMaxBytesCountsTheActiveSegment`, `TestMaxRecordsNeverLeavesMoreThanTheCap` | `fcb08a5` |
| L2-29 | recov: retention.go drop `counted = true`; `c.offset > last` -> `>=` | `TestDroppedCountsEachDeletedSegmentOnceAndCountsAConsumerAtItsLastRecord` | `ebf0fcf` |
| L2-30 | recov: retention.go `if s.bad != nil { continue }` -> `if false` | `TestTheAgeSweepKeepsADamagedSegmentUntilAGoodOneAfterItExpires` | `aa039fa` |
| L2-34 | recov: archive.go drop `destSeq > 0`; `s.first != lastSeq+1` -> `> lastSeq+2`; `s.first > destSeq+1` -> `> destSeq+2` | `TestArchiveOfARetentionTrimmedSourceIntoANewDestinationReportsNoGap`, `TestArchiveReportsAHoleOfOneRecordBetweenCopiedSegments`, `TestArchiveReportsOneRecordLostBeforeTheFirstSegmentItCopies` | `fcaba49` |
| L2-35 | recov: archive.go `info.last.Seq > s.last` -> `> s.last+1`; `di.Size() == size` -> `>=`; range check -> `true`; `os.SameFile(si, di)` -> `false` | `TestArchiveRefusesASegmentOverlappingTheNextByOneFrame`, `TestArchiveRerunReplacesADestinationSegmentWithTrailingBytes`, `TestArchiveRerunReplacesASameSizeIntactDestinationSegmentOfAnotherSequenceRange`, `TestArchiveRefusesTheClosedSourceLogAsItsOwnDestination` | `8b0e6c2` |
| L2-36 | recov: stat.go `p.MaxBytes == 0 && p.MinFreeBytes == 0` -> `\|\|`; -> `p.MaxBytes == 0` | `TestCheckAndLsSayUnboundedOnlyForALogWithNeitherCap`, `TestReaderCheckReportsUnboundedOnlyWhenNeitherCapIsSet` | `5638419` |
| L2-31 | recov: retention.go `c.Offset <= head` -> `<`; `now.Sub(mtime) >= ConsumerTTL` -> `>`; `now.Unix()-recs[0].At >= ttl` -> `>`; `now-lastAt >= SegmentMaxAge` -> `>`; `s.lastAt > cutoff` -> `>=` | `TestConsumersStaleProbeActsExactlyAtTheTTL`, `TestAConsumerIsStaleWhenTheRecordAtItsOffsetIsExactlyTheTTLOld`, `TestAConsumerIsStaleWhenItsFileIsExactlyTheTTLOld`, `TestAConsumerAtTheHeadIsBehindAndOneAfterItIsCaughtUp`, `TestAgeRotationActsExactlyAtSegmentMaxAge`, `TestTheAgeSweepActsExactlyAtRetention` | `7c9a278` |
| L2-37 | recov: stat.go `<= prevLast` -> `<`; `!= prevLast+1` -> `> prevLast+2`; `> tl.Seq+1` -> `> tl.Seq+2`; `c.Offset < first` -> `c.Offset+1 < first`; `first := tl.Seq + 1` -> `tl.Seq`; `rec.At > future` -> `>=`; `v.head >= v.first` -> `>`; drop `l.active.durable` from Bytes | `TestCheckNamesAnOverlapAndAHoleOfOneSequence`, `TestCheckWarnsOfAConsumerOneAheadOrOneBehind`, `TestCheckDoesNotCallARecordExactlyADayAheadFutureDated`, `TestStatCountsASingleRetainedRecord`, `TestStatBytesCountsTheActiveSegmentToo` | `db9d3c7` |
| L2-38 | recov: load.go `bodyLen > maxFrameBody` -> `>=`; `avail >= maxVarint` -> `>`; `int64(got) < want` -> `<=`; `Size() > maxSegmentFileBytes` -> `>=`; `max(tl.MaxSeg, newest)` -> `newest` | `TestRecoverTornTailDeclaringExactlyMaxFrameBodyIsTornNotDamaged`, `TestRecoverTenByteVarintRunIsBadAndNineIsTorn`, `TestWindowAcceptsEOFWithAFullFinalRead`, `TestOpenSegmentAcceptsAFileOfExactlyTheSizeLimit`, `TestOpenNeverReusesASegmentIDBelowTheMarkersMaxSeg` | `9e278df` |
| L2-39 | recov (strictjson): remove checkEscapes; surrogate range bounds; length guards; duplicate-key handling after a string value; array frame treated as an object | `TestSurrogateEscapesAreCheckedInADeclaredField`, `TestCheckEscapesAtTheEndOfData`, `TestDuplicateKeyAfterAStringValueIsRefused`, `TestStringValuesAreNeverDuplicateKeys` | `6f26176` |


### Table C. Reader, iterators, seek and the command line tool

| Item | Mutants (edit) | Killed by (tests now in the tree) | Fix commit |
|---|---|---|---|
| L2-41 | read: C28 `cfg = quill.Config{Retention: -1}` -> `quill.Config{}`; C17 `&&` -> `\|\|` in openWriter's SegmentBytes default; C25 drop `\|\| c.ttl != ""` from `explicit` | `TestPrunePolicylessAppliesOnlyTheFlagsGiven` | `547e387` |
| L2-42 | read: C37 `c.from >= 0`; C42 drop `beginning(st)` from tail's start; C44 `\|\|` -> `&&` in tail's -from/-since test | `TestL2ReadAndTailOnALogWhoseHeadWasTrimmedByRetention` | `eb659ae` |
| L2-43 | read: R04 `tl.Seq > offset` in Reader.wait; I02 `src.wait(ctx, cur.seq+1)` in records | `TestSingleRecordWakesReadWaitAndFollowOnReaderAndLog` | `5f09ee6` |
| L2-44 | read: S20 `!done && cur < s.last` (was `<=`); S21 `if s.active` -> `if false` | `TestReaderRefusesAMarkerOneSequenceAheadOfItsFrames`, `TestDamageInTheActiveSegmentIsCorruptionNotAGap` | `cbc1fef` |
| L2-45 | read: S23 `cur > last` -> `cur >= last`; S24 drop `errors.Is(err, fs.ErrNotExist)` from scanFailure | `TestAReaderServesTheLastRecordOfASealedSegmentThatEndsEarly`, `TestASegmentDeletedBetweenListingAndScanIsAGap` | `ed6c6df` |
| L2-46 | read: R06 `if false && closed` in Reader.wait; R26 second Close returns nil | `TestAParkedReaderReadWaitReturnsErrClosedOnCloseAndASecondCloseIsErrClosed` | `d0e6ec2` |
| L2-47 | read: I05 pass 0 instead of `src.maxReadBytes()` in records; I03 make the `ctx.Err()` guard dead | `TestL2IteratorBatchesAreBoundedByMaxReadBytes`, `TestL2RecordsOnADoneContextYieldsTheContextErrorAndNoRecords`, `TestL2FollowCancelledMidBacklogStopsWithTheContextError` | `126ef3d` |
| L2-48 | read: S19 `a.seq > t+1` in scanSource's anchor search; K03 `a.at > target` in offsetForTime | `TestMultiAnchorSegmentReadsAndSeeks` | `ad06516` |
| L2-51 | read: C57 and C57b `ci.Offset >= head+1` (JSON and text); C63 `r.Read(gap.To, 1)` for `gap.To+1`; C46 `\|\|` for `&&` in `ls` | `TestL2ACaughtUpConsumerIsNotAhead`, `TestL2StaleProbeOfAConsumerBehindTheFirstRetainedSequence` | `1e24b7c` |
| L2-49 | read: S12 inner `bytes >= maxBytes` -> `>`; S13 outer `bytes < maxBytes` -> `<=`; R03 `<= 0` -> `< 0` for the default | `TestMaxReadBytesReachedExactlyStopsTheRead`, `TestReaderDefaultMaxReadBytesBoundsARead` | `3cb3269` |
| L2-50 | read: S08 and I04 `max(offset,1)` removed in readFrom and records; S09 removed in readWait | `TestReadAndRecordsFromZeroAfterRetentionAreAGapButScanFromZeroIsNot`, `TestReadWaitAtOffsetZeroOnAnEmptyLogWaitsAndHonoursItsContext` | `6292566` |
| L2-52 | read: R18 `size != s.size` -> `<`; R27 retry loop cut to one attempt; R01 `>= MaxUint64-1`; S17 `rec.Seq > v.head` guard removed; K09 frame error during OffsetForTime reported bare; C09 `code == exitOK` dropped; C22 `>=` at MaxBatchBytes; C02 and C26 MaxInt64 edges; C03 `-since 0s`; C30 a bad -consumer-ttl accepted | `TestUnhealthyExitIsKeptWhenStandardOutputFails`, `TestAppendStdinExactlyAtMaxBatchBytes`, `TestFlagParsingEdges`, `TestReaderRefusesASealedSegmentThatChangedSizeEitherWay`, `TestReaderRetriesATransientMarkerReadFailure`, `TestReaderAcceptsAHeadAtTheSequenceCeiling`, `TestReaderServesNoSequencePastTheMarkersSequence`, `TestOffsetForTimeOverABadFrameWalkIsAGapOverTheSegment` | `7340cc9` |


### Table D. Hand-picked mutants from the coverage run (areas the coverage ledger found uncovered)

| Item | Mutants (edit) | Killed by (tests now in the tree) | Fix commit |
|---|---|---|---|
| L2-1 | cov: codec.go decodeBody `n <= 0` -> `n == 0` (merged L2-54) | `TestACRCValidFrameWithAMalformedBodyIsBadNotAPanic` | `e874d01` |
| L2-2 | cov: M1 swap SyncDir(dst) and Commit; M6 ignore SyncDir's error (archive.go) | `TestArchiveSyncsTheDestinationDirectoryAfterItsMarkerAndBeforeCommittingTheOffset`, `TestArchiveFailingTheFinalDestinationDirectorySyncReturnsItAndLeavesTheOffsetUncommitted` | `5037042` |
| L2-3 | cov: M4 `size < int64(tl.Off)` -> `false` (reader.go); M14 the same check in Check (stat.go) | `TestAReaderCallsAnActiveSegmentThatFailsTheMarkerCorruptNotAnIOError` | `786a1c5` |
| L2-4 | cov: M5 `GapError{From: cur, To: s.last}` -> `To: cur` (read.go); M20 delete the OffsetForTime frame-error branch (seek.go) | `TestASealedSegmentDamagedAfterOpenIsAGapOverTheRestOfItAndAResumeAfterItReadsOn`, `TestTheActiveSegmentDamagedAfterOpenIsCorruptNeverAGap`, `TestOffsetForTimeOnASealedSegmentDamagedAfterOpenIsAGapOverTheSegment` | `168221a` |
| L2-6 | cov: M11 remove the `c.closed` check in Consumer.Commit | `TestAClosedConsumerHandleCannotOverwriteTheOffsetOfTheNewHolder`, `TestLogMaintenanceCallsOnAClosedLogReturnErrClosed` | `7b26d68` |
| L2-8 | none (the wall-clock test produced false mutant kills: 28 in the write run, 14 in the read run) | changed in place: TestOpenWithAProtectedConsumerRunningDoesNotNeedItsLock and the isolation and Sync tests in sync_test.go | `743a5c5` |
| L2-9 | cov: M12 `uint64(st.Bavail) * uint64(st.Bsize)` -> `uint64(st.Bavail)` | `TestAvailBytesIsAvailableBlocksTimesBlockSize`, `TestFreeBytesTracksRealWrites`, `TestMinFreeBytesWithTheRealStatfs` | `ddce2be` |
| L2-10 | cov: M15 drop the `hole a-b` loop in cmdRecover; M16 drop the GapError re-read in `stale`; M17 drop the `-name is required` check | `TestPruneExitsOneWhenALaterUnlinkFails`, `TestPruneRejectsAMalformedConsumerTTL`, `TestAppendOfNothingIsAUsageError`, `TestAppendOverThePolicyPayloadLimitExitsOneAndAppendsNothing`, `TestAppendStdinReadErrorExitsOne`, `TestRecoverPrintsEachHole`, `TestCommitAndRegisterRequireAName`, `TestConsumersStaleProbeOfAConsumerInAHole` | `199da64` |
| L2-11 | cov: M2 disable log.go:555-562; M3 disable load.go:309 (and six more by a voter) | `TestOpenRefusesALogWhoseMarkerOrSegmentsAreGone`, `TestAPrunePassWhoseAgeRotationFailsStillSweepsAndALaterPassRotates` | `0ecddf3` |


### Table E. Findings with no mutant (fuzz, differential and resource-bound coverage)

| Item | Mutants (edit) | Killed by (tests now in the tree) | Fix commit |
|---|---|---|---|
| L2-55 | none (coverage proposal; a voter used the anchor comparator mutant `a.seq > t+1`) | `FuzzWalkChunks`, `FuzzReaderDir` | `a891274` |
| L2-53 | none (found by the invariants-and-fuzz auditor, not by a mutant) | none | `(none)` |
| L2-56 | none (fuzz oracles; the fix-skeptic mutated the decoders) | `FuzzTailDiff`, `TestHostileStateFilesAreRefusedWithBoundedAllocation` | `5a27add` |
| L2-57 | none (fuzz and differential coverage; the fix-skeptic mutated print.go, the parsers, validName and archive.go) | `TestArchiveAgainstTheSourceReaderOverDamagedSegments`, `FuzzParseDecimal`, `FuzzParseTime`, `FuzzPrinter`, `FuzzValidName`, `FuzzParseSegmentName` | `1ef27ac` |
| L2-58 | none (resource bounds; the fix-skeptic mutated read.go, reader.go, log.go and retention.go) | `TestConcurrentReadsHoldABoundedNumberOfDescriptors`, `TestParkedFollowsHoldNoDescriptorsAndReleaseOnCancel`, `TestCloseLeavesNoGoroutineBehind`, `TestMaxRecordsBoundHoldsUnderRandomOperations` | `f274b51` |


## Survivors judged equivalent or unobservable

Judged by reading the code unless stated. "Equivalent" means no input, however built, gives a different result; "unobservable" means a different result exists but no test through
a public API, the simulator or the real file system can see it. These were excluded from the scores and filed as no items.

### Write-path run: 15 equivalent, 1 no-op probe, 6 unobservable

| Mutant | Why it is equivalent |
|---|---|
| A07 drops `l.bufCount > 0` (append.go) | flushErr is set only with a non-empty buffer and cleared only with the buffer reset, so flushErr != nil implies bufCount > 0 |
| A12 syncs `first` not `first+n-1` | a batch is added to one buffer under one lock hold and one flush makes the whole buffer durable |
| C11 `n <= 0` to `n < 0` on the first varint | an empty body is caught by the third check |
| C14 drops frameHeader's length check | it reads only frames the writer encoded |
| F18 drops `active.frames > 0` in publishLocked | SegmentBytes is at least 1, so durable >= SegmentBytes implies frames > 0 |
| F21 drops `\|\| sealed` | rotation changes Seg, so the marker always differs |
| F30 `first: l.head+1` on a new active segment | a.first is read only behind `frames > 0` and flush.go overwrites it on the first append |
| F35 drops `\|\| l.pendingPublish` in Roll, and L19 (Close's `err == nil && l.pendingPublish` retry) | after a nil flushLocked, pendingPublish is false (every nil return of publishLocked sets it from the rotation error): the branch is dead |
| L17 drops the fsync of the empty first segment | nothing to persist; the directory fsync names it |
| L18 drops create()'s final SyncDir | Open ends every path with a SyncDir |
| S03, S04, S05 (fast-path inclusive limits) | a fast-path refusal falls to decodeTailSlow, which accepts: same verdict, only allocations differ |
| S19 | ReadAt never returns an error with a full read |
| A17 (the probe) | `p[:len(p):len(p)]` was a no-op, not a mutant |

Unobservable (6): **O07** (fsx.OS.SyncDir does not call fsync) passes everything including on Linux: no test can see an fsync on a real file system without syscall
tracing, and the simulator models directory fsync independently of the OS wrapper, so that one line is pinned by reading only. **O05** (osFlags drops O_WRONLY) has no
write-only caller. **O06** (Rename EINTR retry) needs a signal at the exact syscall. **T05** (atomicfile drops O_EXCL) guards a 64-bit random name collision. **L27**
(resolveDirLink non-NotExist Lstat error) differs only in which path an error names. **F36** (a failed Roll leaves pendingPublish set) differs by one idempotent marker rewrite.
Each has an entry in [untestable-without-x.md](../../reference/untestable-without-x.md).

### Recovery run: 12 equivalent, 10 unobservable

Equivalent (inferred from reading; the last is argued, not proven): load.go `cut = tail.end < size` to `!=` (tail.end never exceeds size); load.go `a.first = head + 1` to `head`
(a dead store, since the first append sets it and view() reads it only when frames > 0); retention.go `first > 1` to `first > 0` (a damaged oldest segment implies first > 1);
consumer.go Lag `>` to `>=` (both return 0); stat.go `h.From >= v.first` to `>` (a hole cannot start at the first retained sequence); six strictjson re-spellings (`i+12` to `i+11`,
`i += 11` to `10`, `i += 5` to `4`, the malformed-escape `i++` removed, the loop bound `i+1 < len` to `i+2 < len`, foldRune keep-min to keep-max), argued from hex4's length guard and
the fact that a skipped byte is never a backslash; retention.go drop `!l.pendingPublish` (a redundant guard: a second rotateLocked attempt is the retry the flusher would make).

Unobservable (10 mutants in these groups): load.go drop the `tail.index` append (the sparse index of the active segment only; results are the same, only slower seeks); load.go drop the
Sync after Truncate (recovery is idempotent, a resurrected cut is cut again at the next Open); load.go `tl.Seq > prevLast` to `>=` (needs a marker equal to the last good segment's last while a
damaged segment follows, which publish ordering never produces); archive.go gap-before-copy-error order (the spec is silent on which wins); archive.go either conjunct of the destIntact range
check alone (redundant for natural inputs); archive.go `n < size` short-copy check (needs a sealed source that shrinks mid-copy); retention.go `>` to `>=` in the future-skew Warn (log text only);
strictjson hex4 length guards (killable only by calling checkEscapes directly, since Unmarshal's decoder rejects those inputs anyway).

### Reader and CLI run: 14 equivalent, 9 unobservable, 2 real but not deterministically testable

Equivalent (verified by reading, per the auditor): S03 and S22 (the fall-through returns GapError{seq,seq}, identical at equality), C38 (hole From is never 0), C59 (lag at head+1 is 0 either way), C64 (a
caught-up consumer reads empty and is not stale either way), C18 and C19 (a policy value at the ceiling or 0 falls back to the same default), C20 (exit code is 1 either way; only a second stderr line), R08
(the `rec.Seq == 0` re-check is redundant: a sequence-0 first frame already yields hole 1..2, probed with a hand-built image), R28 (scan() is reached only when result is nil), R21 (the cache entry is
dropped by prune on the next view), R22 and R15 (need a sealed segment with frames past a marker's Seq, i.e. a hand-damaged marker), S30 (a bad ref sorts first-below-any-target and holes are checked first).

Unobservable (verified by reading): C11, C12, C13 (Retention -1 against 0 in append, roll and recover: with NoMaintenance only Prune reads Retention); R02 and C16 (`| 0o600`: Config.Perm must include
owner read and write, so only a hand-chmodded marker differs); C23 (`+33` per stdin line: the library re-refuses at AppendBatch with exit 1, only the message differs); C61 and C62 (stale boundary on
`now - mtime == ttl`: no injectable clock in the CLI); S28 (Log.wait's entry check: a lost wake-up only in a race window).

Real but not filed (no deterministic test without a hook): **K08** (SeekToTime's retry when retention moves First between OffsetForTime and Commit, seek.go) and **C33** (`lerr == nil &&` in cmdCommit: needs
Consumers() to fail and then Commit to fail; would delete a pre-existing consumer's offset file). Both are in untestable-without-x.md.

### Coverage run: judged dead or defensive code (inferred)

M10 (Close's extra publish, log.go:620-622, redundant with flushLocked) and M21 (removing the read.go `a record past the marker` guard left the suite green; the auditor inferred the window sizes already stop at the marker
and did not construct a case). Neither was filed. M21 is the one worth a second look: if the guard is reachable the Reader could serve a record past the marker, which is guarantee 2.
The invariants run planted valid frames past the marker and the Reader never served them (see the coverage statement), which is evidence for, not proof of, the guard being redundant.

## Residual weaknesses after the fixes

Reported by the fix-skeptics: the committed tests leave these mutants alive or these paths unpinned. None is a new finding; they are the edges of each fix.

| Item | What still survives or is unpinned |
|---|---|
| L2-2 | the Commit error return at archive.go:123-125 is untested; a stray extra commit after the sync could slip past the ordering test |
| L2-8 | wall-clock bounds with margins of 1 s or more remain in sync_test.go:128, log_test.go:271, iter_test.go:123 and wait_failed_test.go:129 |
| L2-10 | cmdAppend's Sync-error exit and cmdPrune's Stat-failure path stay uncovered (they need a seam); the Linux unlink test skips under root |
| L2-11 | the load.go `len(ids)==0` branch after Open's creation step (a concurrent delete) |
| L2-35 | the two destIntact conjuncts, each alone, still survive |
| L2-39 | six equivalent strictjson mutants; `TestAGenuineReplacementCharacterAndAValidPairStay` still asserts only the absence of an error |
| L2-55 | FuzzWalkChunks and FuzzReaderDir do not detect removing the seq-0 guard in walk (no seed has a CRC-valid seq-0 frame; an example test does) nor a Reader that stops serving at an anchor (FuzzReaderDir discards r.Read errors) |
| L2-56 | the maxseg > maxSegmentID check is caught by unit tests, not by the fuzz assertion; the 64x allocation threshold sits above the measured 35x |
| L2-57 | the Archive differential does not catch a dropped overlap check, a removed GapError, or a destination marker off by one, and skips the destination when OpenReader fails |
| L2-47 | Consumer.Records and Consumer.Follow are not tested directly for the batch bound |
| L2-49 | R03 is killed through Reader.Read only |
| L2-58 | the randomised MaxBytes bound was not added (covered by TestDiskBounds) |

## Behaviour pinned that the spec does not state

The new tests freeze these. Each is the current behaviour; the fixer or a voter noted that no decision or document states it. If the owner wants a different answer the test is the one line to change. They are not held
items because the code was left as it is.

- **L2-12.** An Append whose inline flush fails the log returns `ErrFailed` itself (decision 60(b), 37; decision 100 says the inline flush discards its error). The auditor's alternative: pin `(first, nil)` instead.
- **L2-50.** `Read(0)` and `Records(0)` on a log whose prefix retention deleted return a GapError (decision 84 and 38 imply it; neither names offset 0). `Scan(0)` starts at First.
- **L2-27 and L2-18.** Equality edges of `MaxBytes`, `MinFreeBytes`, `MaxBatchBytes`, `MaxBufferBytes`, the rotation ceiling, `FlushBytes` and `SegmentBytes` (inclusive or exclusive). Decision 46 quotes the 34-byte constant and the batch numbers; the rest follow the Config doc wording.
- **L2-31.** An age exactly equal to `ConsumerTTL`, `Retention` or `SegmentMaxAge` counts as past it; `offset == Head` counts as behind. The fix changed the wording in decision 45, scope.md, CHANGELOG and three comments from "older than" to "at least as old", so the doc was brought to the code rather than the reverse. A decision text was edited by the fixer; the owner may want to confirm.
- **L2-47.** A pre-cancelled context yields the context error and no records from `Records` and `Follow`.
- **L2-52.** `check` keeps exit 3 when standard output fails (the `run` comment in verbs.go says a verb that would have exited 0 exits 1 instead); the owner may prefer exit 1.
- **L2-19.** The 1 ms floor on `FlushInterval` exists only in code (log.go), with no document.

## Benchmarks

Pass L2 did not run the benchmark tier and filed no benchmark finding. `bench_test.go` has 15 benchmarks (reported by the recorder's `grep -c '^func Bench'`); their baselines are in [benchmark-baselines.md](../../reference/benchmark-baselines.md).

## Item records

One entry per item, in severity order. "Voters" are the two or three skeptics who tried to refute the item. Everything under Grounds, Rebuttal and Reproduction is reported by the auditors and voters; the recorder ran nothing. The fix commit and the test names were read from this tree.

### L2-24 (high, CONFIRMED)
- **Claim.** No test checks that recovery rejects a CRC-valid frame with a non-contiguous sequence placed exactly at the marker when the durable prefix is non-empty.
- **Grounds.** The tail walk in load.go (`walk(w, durable, info.last.Seq, info.frames > 0, nil)`) lost its contiguity seed and the whole suite stayed green. The only wrong-sequence row passes durable 0, where the first walk has already set the previous sequence after frame 1.
- **Warrant.** With the seed dropped, walk accepts any Seq >= 1 as the first frame, so a frame numbered 9, a repeat 3 or a regression 2 directly after a marker at seq 3 is accepted into the segment.
- **Backing.** recoverSegment contract ('accepting each frame that is intact and contiguous') and decision 39 (reported, from code comments).
- **Rebuttal.** A real crash leaves a torn prefix, not a complete CRC-valid frame with a wrong sequence; reaching it needs stale or foreign bytes. One voter would lower the severity to medium.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Voters applied the mutant to a copy: suite green. A test appending a seq-9 frame (then 3, then 2) plus contiguous frames after a marker at frameEnds(head)[2] returned cut=false and frames 7 under the mutant, and cut=true, end=durable, frames 3 on the tree. The fix-skeptic ran seq 3 and seq 2 alone: each kills the mutant.
- **Disposition.** Fixed in `dac7bfb`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: rows added to TestNothingAfterTheFirstBadFrameIsTrustedWhateverItContains.
- **Left open by the fix** (reported by the fix-skeptic). Fatalf stops at the first failing row; the three rows were each shown to kill the mutant separately.

### L2-26 (high, CONFIRMED)
- **Claim.** Protected-consumer retention conditions are not pinned: the slowest of several protected consumers, a consumer sitting on a segment's last record, a damaged segment's upper bound, and fail-closed on an unlistable consumers directory.
- **Grounds.** Four mutants survived. Under the first two a protected consumer loses records it has not read; under the fourth retention deletes with consumers unseen; the third over-protects (safe side).
- **Warrant.** The existing multi-consumer tests never put a slow consumer beside a fast one, nor a consumer at offset == last of a segment.
- **Backing.** retention.go comments ('stop before the first segment holding records at or above the slowest protected consumer's offset'; an unlistable directory protects everything); decisions 45 and 57 (reported, from code comments).
- **Rebuttal.** The `- 1` mutant errs on the safe side (low severity). Caps override protection by design, so only the age sweep is affected.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Voters applied each mutant to a clean copy: root suite green for all four. Probe tests (slowest consumer in both file orders, Commit(4) on a 4-record segment, OpReadDir EACCES on consumers/) failed under mutants 1, 2 and 4 and passed on the tree; the over-protect mutant was probed only by the fix-skeptic.
- **Disposition.** Fixed in `d29328e`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestTheSlowestProtectedConsumerHoldsTheAgeSweepInEitherFileOrder`, `TestAProtectedConsumerAtTheLastRecordOfASegmentKeepsThatSegment`, `TestADamagedSegmentBelowAProtectedConsumersOffsetIsDeletedWithTheExpiredOnes`, `TestAnUnlistableConsumersDirectoryProtectsEverySegment`.
- **Left open by the fix** (reported by the fix-skeptic). None; the fix-skeptic also tried first-wins and last-wins variants of the slowest-consumer mutant and both fail.

### L2-32 (high, CONFIRMED)
- **Claim.** The writer-restart fast path for protected consumers is untested for a caught-up consumer and for a consumer file that belongs to another log.
- **Grounds.** Two mutants survived. With `<`, a live, caught-up protected consumer forces Open to take its lock (ErrLocked after the lock retry); without the ID check a foreign log's consumer file is accepted silently instead of ErrWrongLog. The existing test creates its consumer behind the head.
- **Warrant.** A caught-up consumer is the steady state, so under the first mutant every redeploy of the writer beside a running consumer fails.
- **Backing.** consumer.go doc (A1-80: Open must not need the lock when the file already says protected, parses, is this log's and is not ahead); decision 67.
- **Rebuttal.** ErrLocked is a loud failure at deploy; the foreign-file mutant is silent but needs a copied or restored consumers/ directory.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Voters ran the full suite under each mutant (green) and a test per scenario: ErrLocked under the `<` mutant, a nil error under the dropped ID check. Both pass on the tree. The fix-skeptic also ran them on the Linux box.
- **Disposition.** Fixed in `c824158`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestOpenWithACaughtUpProtectedConsumerRunningDoesNotNeedItsLock`, `TestOpenWithAProtectedConsumerFileOfAnotherLogFails`.
- **Left open by the fix** (reported by the fix-skeptic). None recorded.

### L2-33 (high, CONFIRMED)
- **Claim.** Consumer file error handling is not pinned: an unreadable file silently re-created at the first retained sequence, a failed Commit leaving the in-memory offset advanced, a leftover .tmp in consumers/ listed as a damaged consumer (which stalls the age sweep), and ErrAhead one beyond the boundary.
- **Grounds.** Four mutants survived. Each gives a wrong user-visible result: the stored offset overwritten after an EIO, an offset that says 4 while the file says 1, retention stalled by a temp file, a log accepted that is ahead.
- **Warrant.** No test injects an error into a consumer file read or a failed rename, or writes a temp file into consumers/.
- **Backing.** consumer.go Commit contract; 'A lock file or a temp file is not a consumer'; decision 36 (ahead refusal); retention.go comment (an unreadable file protects everything).
- **Rebuttal.** The triggers are fault-injection or crash states, not the main path, which is why they were untested.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** One voter ran each mutant against the suite and saw it pass; the other judged survival by reading the tests and ran its own four tests, which pass on the tree and fail on the mutants. The fix-skeptic applied all four (`_ = prev` for the second, since a plain deletion does not compile) and each fails exactly its test.
- **Disposition.** Fixed in `143367d`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestAnUnreadableConsumerFileIsAnErrorAndNeverResetsTheOffset`, `TestAFailedCommitLeavesTheHandlesOffsetWhereItWas`, `TestALeftoverTempFileInConsumersIsNotAConsumerAndDoesNotStallRetention`, `TestAConsumerTwoAheadOfTheHeadIsRefusedAndOneAheadIsCaughtUp`.
- **Left open by the fix** (reported by the fix-skeptic). The unreadable-file test fails at its first (open, EIO) iteration under the mutant, so the other op/errno combinations are not independently shown to kill it; a lock-file or failed-SyncDir variant is not covered.

### L2-41 (high, CONFIRMED)
- **Claim.** Nothing pins that `quill prune` with flags and no readable quill.policy applies only the flags given.
- **Grounds.** All three mutants survived ./ and ./cmd/quill. C28 makes a cap-only prune also run the 24 h age sweep. C17 makes the same prune fail with `invalid config: MaxBytes ... below twice SegmentBytes 1073741824`; C25 makes `prune -consumer-ttl 1h` exit 1 `quill.policy cannot be read`.
- **Warrant.** Config.Retention 0 means 24 h and Prune runs the age sweep for any Retention > 0, so the `-1` is the only thing separating 'apply -max-bytes' from 'apply -max-bytes and a 24 h sweep'. The CLI cannot stamp records older than 24 h, so no existing test could see the difference.
- **Backing.** Decisions 68 and 99, the guard comment in cmd/quill/write.go, and TestPolicyWithoutRetentionIsRefusedByPruneAndLs.
- **Rebuttal.** Production code is correct today; reachable only for a missing or unreadable policy, and the clock guard limits one pass to a quarter of the sealed segments.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Reproduced by the auditor and both voters. A log of 10 sealed segments stamped 72 h old with the policy removed: baseline `prune -max-bytes 100000000` prints `segments 10 -> 10`; under C28 it prints `segments 10 -> 8, first 1 -> 4` with a clock-guard warning (a voter's run: `9 -> 7`).
- **Disposition.** Fixed in `547e387`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestPrunePolicylessAppliesOnlyTheFlagsGiven`.
- **Left open by the fix** (reported by the fix-skeptic). None recorded.

### L2-42 (high, CONFIRMED)
- **Claim.** `quill read`, `cat` and `tail` are never tested on a log whose oldest records retention deleted, and `tail -from` / `-since` are never tested for content.
- **Grounds.** All three mutants survived. On a 10-record log pruned with `prune -max-records 3` (head 10, first 8): C37 makes `read` exit 1 `records 1 to 7 are unavailable`; C42 does the same for plain `tail`; C44 makes `tail -from 9` print 8-10 and `tail -since @0 -n 1` print only record 10. The only existing gap tests use a damaged oldest segment, never a trimmed prefix.
- **Warrant.** A retention-trimmed log is the normal steady state; Records(ctx, 0) maps offset 0 to sequence 1, so any CLI path that passes 0 or a start below First meets a GapError. Each mutant removes one CLI guard against that.
- **Backing.** Decision 38 (a prefix deleted by retention is expiry; a read from the beginning starts at First) and the verbs.go summary.
- **Rebuttal.** Behaviour is correct today; a tail asking for fewer records than are retained is unaffected by C42, so the test must trim below n.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Voters built the log with append, roll and prune, ran the three mutants (suite green) and a probe test that fails on each. The fix-skeptic found that `-since` makes `tail` ignore `-n`; a voter noted this is not documented in so many words.
- **Disposition.** Fixed in `eb659ae`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestL2ReadAndTailOnALogWhoseHeadWasTrimmedByRetention`.
- **Left open by the fix** (reported by the fix-skeptic). The test name carries the audit label; setup tolerates First anywhere in 2..~50.

### L2-43 (high, CONFIRMED)
- **Claim.** No test appends exactly one record to a parked Reader.ReadWait, Reader.Follow, Log.Follow or Log.ReadWait.
- **Grounds.** R04 and I02 survived (R04 also on the Linux box). Existing wake tests append two records at once, so `head > offset` and `head >= offset` agree. Either mutant delivers a new record only after the next one arrives.
- **Warrant.** A consumer on a quiet log with one new record per interval would see every record one interval late, and the last record never until more traffic arrives.
- **Backing.** Reader.ReadWait and Follow docs ('blocking until a record is available'; 'waking on each flush'); decision 63 (cited loosely: decision 63 is the index cache).
- **Rebuttal.** Log.wait's own entry check (mutant S28) is a lost wake-up in a race window and is not deterministically testable without a hook; not claimed.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Both mutants pass the full suite; the single-record test fails on R04 (reader-readwait, reader-follow) and I02 (reader-follow). A voter and the fix-skeptic found that the log-follow and log-readwait subtests pass under both mutants (Log.wait wakes on the flush channel whatever the offset), so the Log cases are regression coverage only; the claim's statement that I02 fails log-follow is inaccurate.
- **Disposition.** Fixed in `5f09ee6`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestSingleRecordWakesReadWaitAndFollowOnReaderAndLog`.
- **Left open by the fix** (reported by the fix-skeptic). None recorded.

### L2-1 (medium, CONFIRMED)
- **Claim.** The panic guard in decodeBody for a CRC-valid frame whose `at` varint is padded or overflowing is pinned by no test (L2-54 filed the same gap from the fuzz side and was merged here).
- **Grounds.** The block at codec.go:124-125 is uncovered on both OSes. Loosened to `n == 0`, the root suite stays green; the probe body [5,0x89,0x00,2,'a','b'] with correct length and CRC32C panics with `slice bounds out of range [-1:]`. noncanonical_varint_test.go forges a CRC-valid frame only for a padded seq; FuzzDecodeFrame feeds raw bytes and reaches decodeBody only by guessing a 32-bit CRC.
- **Warrant.** uvarint returns n < 0 for a non-minimal or overflowing varint and `body[n:]` with n = -1 panics; the guard is the only barrier between a forged segment file and a panic in Open, Read, Check or Archive.
- **Backing.** Guarantees 6 and 7; build prompt T12 (parser boundaries are fuzz targets); format.md requires minimal varints.
- **Rebuttal.** HEAD is correct; the CRC makes accidental damage almost never reach decodeBody and a forger needs write access to the log directory.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Both voters: mutant green on the suite, probe panics under the mutant and returns errBadFrame on the tree. The fix-skeptic loosened each of the three `n <= 0` guards (seq, at, plen) separately: each fails the table test.
- **Disposition.** Fixed in `e874d01`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestACRCValidFrameWithAMalformedBodyIsBadNotAPanic`.
- **Left open by the fix** (reported by the fix-skeptic). The table has one duplicate row ('seq only' and 'seq with no at' are the same body); a panic aborts the test binary at the first bad case.

### L2-2 (medium, CONFIRMED)
- **Claim.** Archive's durability ordering (destination marker, then SyncDir(dst), then the _archive Commit) and the error returns around it are untested.
- **Grounds.** Blocks at archive.go:120-125 are uncovered. M1 and M6 left the root suite green. The existing crash tests crash only after a fully successful run or kill only temp-file writes; the one sync-fault test does not assert the consumer offset.
- **Warrant.** If Commit lands first, a power loss that persists the offset but not the destination's rename loses sealed segments from the archive while the source may delete them.
- **Backing.** Decisions 40, 65, 85; guarantee 12 (the archiver is a protected consumer). The ordering is inferred from the design; no decision states it.
- **Rebuttal.** A rerun re-copies from the offset; the single-lost-item test covers the case where the offset persists and the destination entry does not. HEAD is correct.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Both mutants green; an order test and an EIO-injection test pass on the tree and fail on M1 (order sd=105 cm=104) and M6 (nil returned). The SyncDir fault must fire on the second SyncDir of /arch because archiveDest syncs it earlier.
- **Disposition.** Fixed in `5037042`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. The same commit fixed a data race in two test hooks (defects-log.md, D2). Tests: `TestArchiveSyncsTheDestinationDirectoryAfterItsMarkerAndBeforeCommittingTheOffset`, `TestArchiveFailingTheFinalDestinationDirectorySyncReturnsItAndLeavesTheOffsetUncommitted`.
- **Left open by the fix** (reported by the fix-skeptic). The Commit error return at archive.go:123-125 is still untested; the ordering test uses the last positions, so a stray extra commit after the sync could slip past. The commit also removed a pre-existing data race in two test hooks (defects-log.md, D2).

### L2-3 (medium, CONFIRMED)
- **Claim.** A Reader whose marker names more bytes than the active segment holds must fail with ErrCorrupt; no test exercises it, nor the first-frame check or Check's short-active branch.
- **Grounds.** M4: suite green; with the active segment truncated to 30 bytes (marker says 266) Read returns a bare `unexpected EOF`, not ErrCorrupt. M14: Check changes from `damaged at byte 30, intact to byte 30` to `damaged at byte 0, intact to byte 0`, understating the salvageable prefix.
- **Warrant.** A consumer loop treats a non-ErrCorrupt, non-gap error as retryable, so a truncated active segment would spin instead of surfacing corruption (inferred; no consumer was run).
- **Backing.** Decision 79 (I/O errors are facts about the moment), guarantee 6, scope.md stale-marker rules.
- **Rebuttal.** A live writer fsyncs data before the marker, so size < Off needs outside truncation or a restore. Check still reports unhealthy without the stat.go branch.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Both voters reproduced the bare EOF under M4 and the degraded Check text under M14. The CLI probe on a truncated real segment gives `corrupt log: segment 0 holds 30 bytes, the marker says 65`, exit 1; `quill check` exits 3.
- **Disposition.** Fixed in `786a1c5`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestAReaderCallsAnActiveSegmentThatFailsTheMarkerCorruptNotAnIOError`.
- **Left open by the fix** (reported by the fix-skeptic). The 30-byte case overlaps the first-frame check; each branch is still killed by the case aimed at it. M14 is killed by an earlier test (data_loss_guards_test.go) that already pinned Offset, Intact and Detail.

### L2-4 (medium, CONFIRMED)
- **Claim.** Damage that appears after Open takes the read-time GapError and ErrCorrupt mapping, which no test executes.
- **Grounds.** Blocks at read.go:269-272 and seek.go:94-96 are uncovered; M5 and M20 survive. On HEAD a flipped byte in the last frame of a sealed segment gives records 1..19 with nil error, then a gap `20 to 20`; the same in the active segment defers the error.
- **Warrant.** GapError.To tells a consumer where to resume (Commit(To+1)); a To that is too small re-enters the damaged segment and cycles one gap per record. A frame error in the active segment must be ErrCorrupt, never a resumable gap.
- **Backing.** Guarantee 14, T7 and T25; decisions 38, 79, 94.
- **Rebuttal.** Needs bit rot after a successful open. The auditor's last-frame flip would not kill M5 (cur == s.last); a first- or middle-frame flip is needed (voter finding). The OffsetForTime branch needs a distinct At per segment, via the clock seam.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Both mutants green on the suite; a first-frame flip under M5 returns 26..26 where 26..30 is wanted. The fix-skeptic added a mutant for the active-segment branch (ErrCorrupt replaced by a GapError): the active test fails.
- **Disposition.** Fixed in `168221a`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestASealedSegmentDamagedAfterOpenIsAGapOverTheRestOfItAndAResumeAfterItReadsOn`, `TestTheActiveSegmentDamagedAfterOpenIsCorruptNeverAGap`, `TestOffsetForTimeOnASealedSegmentDamagedAfterOpenIsAGapOverTheSegment`.
- **Left open by the fix** (reported by the fix-skeptic). None recorded.

### L2-6 (medium, CONFIRMED)
- **Claim.** Consumer.Commit after Close returns ErrClosed but no test calls any consumer method on a closed handle; removing the check lets a stale handle overwrite the offset of the process that now owns the lock.
- **Grounds.** The block at consumer.go:205-206 is uncovered; M11 left the root suite green. Auditor probe on the mutant: handle a closed, handle b opened and Commit(8), then a.Commit(3) returned nil and the file read back 3.
- **Warrant.** Close releases the consumer's flock; without the guard two holders can write offsets, rewinding (re-delivery) or advancing (loss) a consumer with no error.
- **Backing.** Guarantee 5; the Commit doc comment; T9.
- **Rebuttal.** Needs a caller to use a handle after closing it. Read, Lag and Offset deliberately keep working after Close. One voter did not re-run the two-handle probe.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Mutant green on the suite (including ./...); one voter's probe returned nil on the mutant with b.Offset() still 8. The fix-skeptic mutated the guards in Commit, Consumer.Close, Flush, Roll, Prune and Stat: each fails its test.
- **Disposition.** Fixed in `7b26d68`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestAClosedConsumerHandleCannotOverwriteTheOffsetOfTheNewHolder`, `TestLogMaintenanceCallsOnAClosedLogReturnErrClosed`.
- **Left open by the fix** (reported by the fix-skeptic). The first test stops at the Commit assertion under the mutant, so the file-overwrite assertion is not exercised on its own.

### L2-8 (medium, CONFIRMED)
- **Claim.** Wall-clock assertions make the gate's `go test -race ./...` fail, or pass vacuously, under CPU load (merged L2-23 and L2-40, the same test seen from two mutation runs).
- **Grounds.** TestOpenWithAProtectedConsumerRunningDoesNotNeedItsLock asserted `d > lockRetry/2` and failed 3 of 3 reruns at load about 7-8 (`reopen took 117ms`). sync_test.go rested on 20 ms sleeps meaning 'the flush is inside its fsync' and on 20, 200 and 150+50 ms bounds.
- **Warrant.** The timing assertion is redundant: if Open needed the live consumer's lock it returns ErrLocked after the lock retry, which the test already checks. Under load the 20 ms sleeps can elapse before the flusher reaches the injected sleep, so the isolation tests then pass without testing anything (inferred from reading).
- **Backing.** hack/gate.sh runs the race suite as the commit gate and says a gate that fails with no repo change is not a gate; T5 and T18.
- **Rebuttal.** Passes on an idle machine; the vacuous-under-load part is inferred, not demonstrated.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED (one did not reproduce). Reproduced by: 1 of 2.
- **Reproduction.** One voter did not reproduce (16 and 32 CPU hogs, all runs passed, reopen 35-62 ms); the other saw 3 of 20 runs fail under disk load (`reopen took 106-164ms`) and showed the err check alone fails when the fast path is disabled. The fix-skeptic could not reproduce the original flake on macOS but showed the replaced tests kill three production mutants.
- **Disposition.** Fixed in `743a5c5`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: changed in place: TestOpenWithAProtectedConsumerRunningDoesNotNeedItsLock and the isolation and Sync tests in sync_test.go.
- **Left open by the fix** (reported by the fix-skeptic). TestAProducerWaitingInSyncHoldsNoLock now catches a Sync that holds the lock only by hanging (the flush holds the lock across the fsync by design). Other `time.Since` bounds with margins of 1 s or more remain (sync_test.go:128, log_test.go:271, iter_test.go:123, wait_failed_test.go:129).

### L2-12 (medium, PLAUSIBLE)
- **Claim.** No test asserts that an Append whose inline flush fails the log returns the failure; Append can return (seq, nil) on a log that is already failed.
- **Grounds.** A14 builds and the full suite passes. Existing tests check Flush, Sync and later Appends only.
- **Warrant.** The inline-flush fsync failure is the one path where a caller learns of the failure from the Append itself.
- **Backing.** Decision 60 (b) (a data fsync error is fatal and sticky), decision 37, the Flush doc. Decision 100 says the inline flush discards its error, which is true of the flush error but not of the failed state the code then returns.
- **Rebuttal.** Append without SyncAppends promises only buffering, so (first, nil) is not a lost acknowledgement in the strict sense; the next Append returns ErrFailed. If the owner treats the nil as intended, the test should pin that instead.
- **Verdict.** PLAUSIBLE. Votes: 1 CONFIRMED, 1 PLAUSIBLE. Reproduced by: both voters.
- **Reproduction.** Both voters: A14 green on the suite; a test (FlushBytes 64, OpSync error, Append of 200 bytes) fails on A14 with `append: <nil>`. One voter rated the item PLAUSIBLE because the spec does not decide the behaviour for the triggering call.
- **Disposition.** Fixed in `2fc3030`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestAnAppendWhoseInlineFlushFailsTheLogReturnsTheFailure`.
- **Left open by the fix** (reported by the fix-skeptic). The test now pins ErrFailed from the triggering Append, which no decision states (see 'behaviour pinned that the spec does not state' in mutation-hardening.md).

### L2-13 (medium, CONFIRMED)
- **Claim.** Close's final directory fsync, which decisions 65 and 40 promise, is not tested.
- **Grounds.** log.go:623-625 `if serr := l.fs.SyncDir(l.dir)`; L20 removes the call and the full suite passes. The crash property tests never crash after Close.
- **Warrant.** The marker rename is not directory-fsynced by an ordinary flush, so Close's fsync is the only thing that makes the last flushes' marker durable on a clean shutdown; a power loss right after Close leaves an older marker and readers see fewer records until a writer reopens.
- **Backing.** Decision 40 (the directory fsync happens at rotation, at Close and before any deletion); decision 65; scope.md Close row.
- **Rebuttal.** Records are not lost (the data fsync made them durable and Open republishes the marker): the harm is a stale, hiding marker, which is documented behaviour.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** A test (Sim, append 3, Close, Crash(nil), decode the durable quill.tail: Seq 3) fails on L20 with `Seq:0`. The fix-skeptic also moved the SyncDir before the flush in Close: the test fails, so it pins the order as well as the call.
- **Disposition.** Fixed in `3f77076`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestCloseMakesTheMarkerDurable`.
- **Left open by the fix** (reported by the fix-skeptic). None recorded.

### L2-14 (medium, CONFIRMED)
- **Claim.** The marker's maxseg is not pinned to the highest id ever created.
- **Grounds.** load.go sets `l.maxSeg = max(tl.MaxSeg, newest)` so maxSeg can exceed the active id after Open; F19 lowers it on the first flush and the suite passes.
- **Warrant.** After such an Open the first flush lowers the published maxseg; if the higher segments were removed or damaged, a later restart can re-create a lower id, the reuse the field exists to prevent.
- **Backing.** format.md and decisions (maxseg is the highest id ever created, never reused); the state.go field comment; decision 65.
- **Rebuttal.** Needs an operator or fault to remove or damage the newest segments; the lowered value self-heals at the next rotation. If the owner holds that load.go never needs to survive a flush, the mutant is equivalent in practice.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** A test closes a 2-record log, rewrites the marker as {seg 0, maxseg 9}, reopens, appends, flushes and requires maxseg 9: it fails with `maxseg 0` under F19.
- **Disposition.** Fixed in `ab45579`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestKillMarkerKeepsMaxSegAboveTheActiveSegment`.
- **Left open by the fix** (reported by the fix-skeptic). Pins the flush.go publish only; the archive's marker writer is a different directory and out of scope.

### L2-15 (medium, CONFIRMED)
- **Claim.** A directory-fsync error after the marker that names a new segment must fail the log (decision 60 e), but no test injects it.
- **Grounds.** flush.go `if err := l.fs.SyncDir(l.dir); err != nil { l.fail("rotation", err) ... }`; the mutant survives. Existing tests fail every SyncDir, which fails the earlier sync first, so this second call site is never reached.
- **Warrant.** A retried fsync can report success for lost data, which is why directory fsync errors are sticky.
- **Backing.** Decision 60 (e), decision 65, decision 78; the Flush doc comment.
- **Rebuttal.** The first rotation fsync and the prune-time fsync are tested; this is one of three call sites.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Mutant green on the suite; a test with the first SyncDir passing and the second failing returns a raw EIO under the mutant. The fix-skeptic mutated the other two fatal sites (rotateLocked in flush.go, and retention.go): existing tests kill both.
- **Disposition.** Fixed in `55a7e97`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestDirFsyncErrorAfterTheRotationMarkerFailsTheLog`.
- **Left open by the fix** (reported by the fix-skeptic). Pins in-process ErrFailed and stickiness only, not that a reopen recovers a consistent log.

### L2-16 (medium, CONFIRMED)
- **Claim.** The background flusher's retry of a failed marker publish on an idle log is untested.
- **Grounds.** The flusher condition `l.bufCount > 0 || l.pendingPublish` in flush.go; the mutant survives. Retry tests call Flush explicitly, and almost every test sets FlushInterval to an hour, which disables the flusher.
- **Warrant.** The flusher is the only caller on an idle log; without the condition readers in other processes keep an old marker until the next append or Close.
- **Backing.** Decision 60 (d) (a failed marker publish is retried at the next flush); the pendingPublish comment.
- **Rebuttal.** Flush, an Append crossing FlushBytes and Close still retry, so only the idle-log, no-caller case is affected. The test polls with a 3 s budget at a 5 ms interval.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Mutant green on the full suite; the test (fail OpRename for the marker, Append+Flush, clear the fault, poll quill.tail) fails on the mutant after 3.00 s and passes on the tree. The fix-skeptic ran it 300 times under -race and 500 times each at -cpu 1 and 4.
- **Disposition.** Fixed in `216c111`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestKillFlusherRetriesAFailedMarkerPublishOnAnIdleLog`.
- **Left open by the fix** (reported by the fix-skeptic). Timing-dependent by nature (3 s deadline); not flaky in 1,300 runs.

### L2-25 (medium, CONFIRMED)
- **Claim.** Hole and head bookkeeping in load() is untested for a damaged newest sealed segment before an empty active segment, for several damaged segments, for a damaged oldest segment that is not the only one, and for a marker exactly one above the last valid frame; Stat.Records at a hole that ends at the head is unchecked.
- **Grounds.** Six mutants survived. Dropping `head = tl.Seq` makes Open fail ErrCorrupt on a log whose newest sealed segment has one flipped bit and whose active segment is empty, a routine state after Roll.
- **Warrant.** Each mutant changes an observable outcome (Open succeeds or ErrCorrupt, the hole list, where a new consumer starts, Stat.Records) in states the suite never reaches.
- **Backing.** load.go comments; decisions 38, 39, 98, 102 (cited in code comments).
- **Rebuttal.** A Reader's lazy view does not show the trailing hole until it scans, so only the writer side is asserted; the one-above-the-marker case needs a hand-edited quill.tail.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** All six survive; the voters wrote five small tests (the commit adds four). The fix-skeptic applied all six to the fixed tree and each fails for its reason.
- **Disposition.** Fixed in `d177d2c`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestADamagedNewestSealedSegmentBeforeAnEmptyActiveOneOpensWithATrailingHole`, `TestEverySealedSegmentDamagedAndAnEmptyActiveOneIsOneHoleFromOne`, `TestSeveralDamagedSegmentsKeepTheirHolesAndTheLeadingOne`, `TestAMarkerAboveTheLastValidFrameIsRefusedAfterAnEarlierDamagedSegment`.
- **Left open by the fix** (reported by the fix-skeptic). The one-above-the-marker case is reached only by a synthetic quill.tail edit.

### L2-27 (medium, CONFIRMED)
- **Claim.** Byte and free-space caps are untested at their equality edges and in interplay with the age sweep, so over-deletion goes unnoticed.
- **Grounds.** Four mutants survive. The slice mutant makes the byte cap count segments the age sweep already chose and delete a second segment the policy did not require; the others delete one more segment than needed at equality, or set DiskFull when free equals the floor.
- **Warrant.** Each deletes data or refuses appends where the written policy says keep or accept.
- **Backing.** The retention.go capsLocked comment ('extends a deletion prefix ... until the caps hold'); Config docs for MaxBytes and MinFreeBytes; decisions 24, 25, 46 (a cap may overshoot by up to one segment).
- **Rebuttal.** Exact equality is rare in production; the slice mutant needs the age sweep and a byte cap in one pass, which is common with both set. Equality is not named in the docs ('below the floor' fits `<`), so the two equality assertions pin an unstated boundary.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** All four survive the suite; the tests fail under each mutant and pass on the tree. The fix-skeptic confirmed each fails only its own test.
- **Disposition.** Fixed in `b040221`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestMaxBytesCapCountsOnlyWhatTheAgeSweepLeaves`, `TestMaxBytesKeepsASetExactlyAtTheCap`, `TestMinFreeBytesStopsDeletingOnceTheFloorIsMetExactly`, `TestMinFreeBytesIsMetAtExactlyTheFloor`.
- **Left open by the fix** (reported by the fix-skeptic). The tests use the simulator with an injected statfs; the Linux box was not used for them.

### L2-28 (medium, CONFIRMED)
- **Claim.** The caps' accounting of the active segment and the MaxRecords off-by-one are unchecked.
- **Grounds.** Both survive. The existing MaxRecords test (MaxRecords 10 with 4-record segments) reaches only counts where the +1 does not matter, and every MaxBytes test leaves the active segment empty.
- **Warrant.** With the mutants the log retains 8 records under a cap of 7, or keeps a segment whose deletion the byte cap requires.
- **Backing.** Config docs and the retention.go comment that the caps 'hold' after the pass; decision 33 ('roughly the newest N records', segment granular).
- **Rebuttal.** Caps are segment-granular, so a test must pick a cap between segment sizes; a test-design gap, not a product bug.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Both survive; MaxRecords 7 over 3 sealed 4-record segments leaves 8 under the `+1` mutant (4 records 9..12 wanted); MaxBytes = total-1 with one record in the active segment keeps the oldest sealed segment under the active-bytes mutant.
- **Disposition.** Fixed in `fcb08a5`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestMaxBytesCountsTheActiveSegment`, `TestMaxRecordsNeverLeavesMoreThanTheCap`.
- **Left open by the fix** (reported by the fix-skeptic). Staticcheck and deadcode were not run by the fix-skeptic for this commit.

### L2-29 (medium, CONFIRMED)
- **Claim.** Stat().Dropped accounting is tested only for a single lagging consumer well below the deleted segment.
- **Grounds.** Both survive. Without `counted = true` Dropped increments once per lagging consumer (6 instead of 3 for two consumers over three segments); with `>=` a consumer whose next-to-read record is the segment's last is not counted.
- **Warrant.** Dropped is the operator-visible loss counter; the mutants change its value in ordinary two-consumer or boundary situations.
- **Backing.** Decision 12 and the Stats.Dropped doc ('segments deleted while some consumer had not read past them').
- **Rebuttal.** Dropped is documented as best-effort ('counted late by up to PruneInterval'); the unit and the boundary are not part of that softness. The Warn lines still name each loss.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Suite green under both; a test with two consumers at 1 and three expired segments (Dropped 3), one consumer at 4 (3) and one at 5 (2) fails under each mutant (6 against 3, and 2 against 3).
- **Disposition.** Fixed in `ebf0fcf`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestDroppedCountsEachDeletedSegmentOnceAndCountsAConsumerAtItsLastRecord`.
- **Left open by the fix** (reported by the fix-skeptic). Covers the age-prune path only, not a cap-driven seal deletion; both reach the same deleteLocked.

### L2-30 (medium, CONFIRMED)
- **Claim.** No test asserts that a damaged sealed segment is kept by the age sweep until a good segment after it expires.
- **Grounds.** The mutant survives (Linux). A damaged segment has lastAt 0, so without the skip it counts as expired at once and is deleted even when nothing is an hour old.
- **Warrant.** The comment says a damaged segment rides along only with an expired good one after it; the mutant deletes the evidence file immediately.
- **Backing.** retention.go comment; decision 98. The intent is stated at comment level only, not in a numbered decision (voter note).
- **Rebuttal.** A damaged segment is data already lost; deleting it early loses forensic evidence, not readable records, and the Warn still records it.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Mutant green; the kill test fails with `damaged segment 0 deleted by age with nothing expired: [1 2 3]`. The fix-skeptic also changed the `continue` to a `break`: the test's second stage fails.
- **Disposition.** Fixed in `aa039fa`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestTheAgeSweepKeepsADamagedSegmentUntilAGoodOneAfterItExpires`.
- **Left open by the fix** (reported by the fix-skeptic). Asserts file retention by id only, not the Warn line or the remaining hole.

### L2-34 (medium, CONFIRMED)
- **Claim.** Archive's gap detection is untested for a first archive of a retention-trimmed log (must not report a gap) and for a one-record loss between copied segments or before the first copied segment.
- **Grounds.** Three mutants survive. Dropping `destSeq > 0` makes every first archive of a log that retention already trimmed fail with `records 1 to N are unavailable`; the other two miss a one-record gap.
- **Warrant.** A spurious error and exit code for the likelier user-facing case; a missed one-record hole for the rarer.
- **Backing.** archive.go comment ('A zero Seq is a new destination ... says nothing about a hole'); decisions 41, 67, 68.
- **Rebuttal.** One-record holes need single-record segments (a Roll after every append), uncommon but legal.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** All three survive; three tests (trimmed source into a new destination, a one-record hole between segments GapError{3,3}, one record lost before the first copied segment GapError{4,4}) each kill exactly one mutant.
- **Disposition.** Fixed in `fcaba49`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestArchiveOfARetentionTrimmedSourceIntoANewDestinationReportsNoGap`, `TestArchiveReportsAHoleOfOneRecordBetweenCopiedSegments`, `TestArchiveReportsOneRecordLostBeforeTheFirstSegmentItCopies`.
- **Left open by the fix** (reported by the fix-skeptic). Decision 68 was not read by the fix-skeptic; deadcode was not run (the pinned binary would not load).

### L2-35 (medium, CONFIRMED)
- **Claim.** Archive's copy guards are untested at their edges: a one-frame overlap, a destination segment with trailing bytes or of another range being kept, and archiving a closed log onto itself.
- **Grounds.** Four survive. The self-archive guard is shielded in TestArchiveRefusesTheSourceLogAsItsOwnDestination because that test keeps a live writer, so the lock check refuses first; with a closed writer Archive(src, src) proceeds and rewrites quill.tail (a voter saw the marker move from seg 2 to seg 1, off 340).
- **Warrant.** Each guard has its own purpose in comments (decisions 102, 85, 8, 62) but the tests reach it only through a path another check also catches.
- **Backing.** archive.go comments; decisions 8, 62, 85, 102.
- **Rebuttal.** The two destIntact conjuncts are individually redundant for natural inputs and were not killed separately; the self-archive harm is a rewritten marker that parse-forward recovery repairs (inferred).
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** All four survive the existing suite; each kill test fails only its own mutant.
- **Disposition.** Fixed in `8b0e6c2`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestArchiveRefusesASegmentOverlappingTheNextByOneFrame`, `TestArchiveRerunReplacesADestinationSegmentWithTrailingBytes`, `TestArchiveRerunReplacesASameSizeIntactDestinationSegmentOfAnotherSequenceRange`, `TestArchiveRefusesTheClosedSourceLogAsItsOwnDestination`.
- **Left open by the fix** (reported by the fix-skeptic). First-only and last-only destIntact conjunct mutants still survive. Under the overlap mutant the test fails through a downstream gap error, not an overlap assertion.

### L2-36 (medium, CONFIRMED)
- **Claim.** Check's UNBOUNDED verdict is tested only for the all-zero policy.
- **Grounds.** Both mutants survive; under them `quill check` prints 'UNBOUNDED: a fast writer can fill the disk' for a log that has a cap.
- **Warrant.** That output feeds operators' alerting and no test saw it.
- **Backing.** The stat.go field doc ('neither MaxBytes nor MinFreeBytes in quill.policy'); the output text in cmd/quill/read.go; decision 66.
- **Rebuttal.** A false alarm, not a missed one; the operator would notice the contradiction.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Both mutants leave the suite green. The fix-skeptic also mutated the `quill ls` note in cmd/quill/read.go and it fails too; that is the C46 mutant of L2-51, so L2-51's test for it was dropped as redundant.
- **Disposition.** Fixed in `5638419`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestCheckAndLsSayUnboundedOnlyForALogWithNeitherCap`, `TestReaderCheckReportsUnboundedOnlyWhenNeitherCapIsSet`.
- **Left open by the fix** (reported by the fix-skeptic). The CLI MinFreeBytes case uses the real statfs with MinFreeBytes 1 (assumes a filesystem reporting some free space).

### L2-44 (medium, CONFIRMED)
- **Claim.** Two corruption cases in scanSource are untested: a marker one sequence ahead of the frames its offset covers (S20) and damage inside the durable part of the active segment (S21).
- **Grounds.** Both survive. S20 turns ErrCorrupt into an endless loop; S21 turns ErrCorrupt into a GapError that a resume from To+1 silently skips.
- **Warrant.** The branches separate loss that is a resumable gap from damage that is not, and from a marker that disagrees with the bytes; no test sits at the boundary.
- **Backing.** Decisions 38, 79 and A1-82; the comment at read.go:269-271.
- **Rebuttal.** S20 needs a hand-damaged quill.tail. The reader's own `rec.Seq > tl.Seq` check in view() rejects other forms.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Suite green under each; the tests fail on the mutants (S20 hangs, so the test runs the read in a goroutine under a 3 s timeout; S21 returns a GapError). A voter noted that Read(1,100) returns records 1 and 2 with a nil error (the damage shows on Read(3,100)): the claim's wording is imprecise.
- **Disposition.** Fixed in `cbc1fef`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestReaderRefusesAMarkerOneSequenceAheadOfItsFrames`, `TestDamageInTheActiveSegmentIsCorruptionNotAGap`.
- **Left open by the fix** (reported by the fix-skeptic). On an S20 regression the read goroutine keeps spinning after the test fails; harmless in a failing run.

### L2-45 (medium, CONFIRMED)
- **Claim.** Reader gap edges are untested: the last real record of a sealed segment that ends early, and a segment deleted between listing and scan.
- **Grounds.** Both survive. S23 reports the last real record as lost; S24 turns a segment deleted between listing and scan from a GapError into a raw path error.
- **Warrant.** A segment lost to retention is a gap over its range and a consumer resumes from To+1; a raw error makes it retry or stop.
- **Backing.** Decisions 38, 79, 94, 101; the scanFailure doc.
- **Rebuttal.** S24 needs a retention race in a two-open window and shows as an error; S23 needs a sealed segment cut at a frame boundary (operator damage).
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Suite green under each; tests (a truncated middle segment; an injected ErrNotExist on the second open) fail on the matching mutant. The S23 mutant's gap is {5,8}, not the claim's {5,6}.
- **Disposition.** Fixed in `ed6c6df`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestAReaderServesTheLastRecordOfASealedSegmentThatEndsEarly`, `TestASegmentDeletedBetweenListingAndScanIsAGap`.
- **Left open by the fix** (reported by the fix-skeptic). None recorded.

### L2-46 (medium, CONFIRMED)
- **Claim.** A ReadWait parked on a Reader is never tested for ending when the Reader closes, nor is a second Close tested.
- **Grounds.** Both survive once the wall-clock test is skipped (they first read as KILLED, but only by that flaky test). With R06 a parked ReadWait with a background context never returns after Close.
- **Warrant.** A service that closes its Reader on shutdown would leak the goroutine and block its own shutdown.
- **Backing.** Reader.ReadWait and Reader.Close docs ('later calls return ErrClosed'). The test comment's citation of decision 63 is loose.
- **Rebuttal.** Callers that always pass a cancellable context are unaffected by R06; if the owner prefers an idempotent Close, the R26 assertion should go.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Suite green under each mutant; the park, Close, second-Close test hangs under R06 and fails under R26. The fix-skeptic ran it 100 times under -race and 20 on Linux.
- **Disposition.** Fixed in `d0e6ec2`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestAParkedReaderReadWaitReturnsErrClosedOnCloseAndASecondCloseIsErrClosed`.
- **Left open by the fix** (reported by the fix-skeptic). None recorded.

### L2-47 (medium, CONFIRMED)
- **Claim.** The iterators' memory bound and cancellation check are untested.
- **Grounds.** Both survive. I05 makes Records and Follow read and copy the whole remaining log before yielding the first record (13 file reads against 1 in a probe); I03 delivers records from an already-cancelled context.
- **Warrant.** iter.go's header promises each batch is read, copied and released before it is yielded; that is the only thing bounding memory for Follow from sequence 1 on a multi-GB log.
- **Backing.** Decision 58 and iter.go:8-12; the Follow docs ('until the context ends').
- **Rebuttal.** The pre-cancelled-context behaviour is not stated as a guarantee, so that assertion pins current behaviour; counting file reads is a proxy for memory.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Suite green under each; the tests fail on I05 and I03. A Follow cancelled mid-backlog is the better-grounded I03 test and was added.
- **Disposition.** Fixed in `126ef3d`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestL2IteratorBatchesAreBoundedByMaxReadBytes`, `TestL2RecordsOnADoneContextYieldsTheContextErrorAndNoRecords`, `TestL2FollowCancelledMidBacklogStopsWithTheContextError`.
- **Left open by the fix** (reported by the fix-skeptic). The byte-bound test counts opens on the Log only; Consumer iterators are not tested directly.

### L2-48 (medium, CONFIRMED)
- **Claim.** No test reads from every offset inside a segment with more than one sparse-index anchor, or seeks by time in one.
- **Grounds.** Both survive. Existing multi-offset tests use segments below 64 KiB (one anchor each). S19 skips a record when the anchor's sequence is exactly cur+1; K03 returns a later sequence than the smallest with At >= t.
- **Warrant.** The sparse index is the only part of the read path that skips bytes; a wrong choice returns the wrong record or skips one with no error.
- **Backing.** Decisions 59 and 63; doc.go read guarantees; the stride is in load.go.
- **Rebuttal.** S19 bites only when reading the record just before an anchor, so the test must sweep every offset. Frame sizes must be calibrated (the sequence varint grows at 128 and 16384).
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Full suite green under both; a 400 x 1000-byte single-segment sweep fails at `offset 65: got record 66` under S19 and `second 1700000006: 66, want 61` under K03.
- **Disposition.** Fixed in `ad06516`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestMultiAnchorSegmentReadsAndSeeks`.
- **Left open by the fix** (reported by the fix-skeptic). Does not cover a run of equal times that spans a segment boundary.

### L2-51 (medium, CONFIRMED)
- **Claim.** `quill consumers` is untested at the caught-up boundary and for the stale probe of a consumer behind the first retained sequence; `quill ls` UNBOUNDED needs both caps unset.
- **Grounds.** All four survive. C57 puts a false 'ahead of the head: rewind it' on every fully caught-up consumer; C63 disables staleness for a consumer behind First.
- **Warrant.** Offset == Head+1 is the normal state of a consumer that has read everything.
- **Backing.** Decision 82 (staleness uses the first retained record when a cap deleted the offset), A5-18, the verb summaries.
- **Rebuttal.** Diagnostic output only: a false alarm or a missed warning.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Suite green under each; tests fail one per mutant. The C46 test was not added here because commit 5638419 (L2-36) already covers it; the fix-skeptic confirmed C46 dies in that commit's test when the whole cmd/quill package runs.
- **Disposition.** Fixed in `1e24b7c`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestL2ACaughtUpConsumerIsNotAhead`, `TestL2StaleProbeOfAConsumerBehindTheFirstRetainedSequence`.
- **Left open by the fix** (reported by the fix-skeptic). C46's kill comes from the L2-36 test and shows only with the full package, not under a narrow -run filter. The C63 test does a 120-append run with a Flush per append (about 1.5 s under -race).

### L2-55 (medium, PLAUSIBLE)
- **Claim.** There is no fuzz or differential target for the Reader's whole-directory behaviour, nor for the window/walk read path as the Reader uses it (chunk-doubling reads, firstFrameChunk probe, resume from a sparse-index anchor).
- **Grounds.** FuzzRecoverSegment uses window{chunk: windowChunk} over the whole file and never starts mid-file; Reader.view, segment, scan and scanSource have no target. The auditor's two scratch targets ran clean: FuzzWalkChunks 90 s, 6.4M execs; FuzzReaderDir 120 s, 503k execs.
- **Warrant.** These are the surfaces a damaged directory reaches from outside the writer; the invariants (ascending, contiguous, never above quill.tail) are checkable in a few lines.
- **Backing.** Guarantees 1, 2, 6 and 7. T12 asks only for the frame decoder and the state-file parsers; decision 70 sets the 20 s budget.
- **Rebuttal.** No defect found, so this is a coverage proposal. The crash-property suite already opens a Reader on damaged and torn directories. The auditor's seeds were built with image(), not from a real writer.
- **Verdict.** PLAUSIBLE. Votes: 2 CONFIRMED, 1 REFUTED. Reproduced by: 2 of 3. Dissent (REFUTED): a coverage preference, not a bug; T12 and decision 70 do not require a Reader target and the crash-property suite already covers it.
- **Reproduction.** Not a mutation reproduction. A voter showed the anchor comparator mutant `a.seq > t+1` survives the full suite and that a read-from-every-offset test (L2-48) is the right check for it, not a fuzz target.
- **Disposition.** Fixed in `a891274`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. The commit also edited the T12 row of test-ledger.md. Tests: `FuzzWalkChunks`, `FuzzReaderDir`.
- **Left open by the fix** (reported by the fix-skeptic). The committed targets do not detect two faults (reported by the fix-skeptic): removing the seq-0 guard in walk (no seed has a CRC-valid seq-0 frame; an example test catches it) and a Reader that stops serving at an anchor (FuzzReaderDir discards r.Read errors, so it checks safety, not liveness).

### L2-53 (medium, CONFIRMED) held for the owner; approved and fixed 2026-11-03
- **Claim.** MaxReadBytes bounds payload bytes only, so a single Read(offset, 0), and every Records or Follow batch, can allocate gigabytes when payloads are empty or tiny.
- **Grounds.** read.go loop condition `maxBytes <= 0 || bytes < maxBytes` and `bytes += len(rec.Payload)`; iter.go passes limit 0 and maxBytes = MaxReadBytes. 4,000,000 empty-payload records: Log.Read(1,0) returned all 4,000,000, TotalAlloc 788 MB. 17,000,000 one-byte records at the default 16 MiB: 16,777,216 records, TotalAlloc 3,892 MB, process max RSS 2.78 GB.
- **Warrant.** An empty payload adds 0 to a total that must reach the cap, so the record count, and with it the 40-byte Record slice plus per-record clones, is bounded only by the log size; the 16 MiB figure understates the memory about 240-fold for 1-byte records.
- **Backing.** Decisions 50 and 79 justify the cap by 'a read into memory needs a byte bound'; guarantee 7; the scope.md Library API row. A1-52 addressed only the one-record overshoot.
- **Rebuttal.** By the letter MaxReadBytes is documented as payload bytes. Records and Follow have no limit parameter. Not a crash; the memory is garbage after the call. A reviewer can rank it a documentation gap.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Auditor program (default Config, real disk, macOS arm64) as above. Voter 1: 1,000,000 nil payloads via AppendBatch, Read(1,0) returned 1,000,000 records, 206 MB. Voter 2: 2,000,000 empty payloads, Read(1,0) and Records() each returned or iterated all 2,000,000 and allocated about 403 MB against a 16 MiB cap. Neither re-ran the 17M case.
- **Disposition.** Held for the owner, recommended answer in [stage2-owner-decisions.md](../../decisions/stage2-owner-decisions.md), section "L2-53". No fix is written, so there is no fix-skeptic outcome and no regression test yet. No fix written. Held for the owner: see stage2-owner-decisions.md, section L2-53.
- **Disposition (updated 2026-11-03).** Owner-approved on 2026-11-03: the recommended answer (charge each record `len(payload) + 48` bytes against `MaxReadBytes`; at least one record per call; the cap reworded as a bound on the size of a batch, not on the sum of payloads). Fixed in 637ed42, FIXED-VERIFIED by a fix-skeptic, 7 of 7 lenses pass (test fails without the fix, root cause, behaviour preserved, no new gap, reachable, ladder, independent route; reported by the skeptic). The one accumulation line in `scanSource` now adds `readRecordOverhead` (read.go:125, value 48; the line is read.go:271, both verified in the tree) and every read path reaches it with the cap; `Scan` is unaffected. Tests: `read_overhead_test.go` (:27 and :91) and adapted constants in append_test.go. Reported by the fix-skeptic on 1,000,000 empty records at the default cap: every read form returned 349,526 records per call, `Records` and `Follow` yielded all 1,000,000 in order, and `Read(1,0)` allocated 216 MB before and 70 MB after. The earlier sentence in this item, "No fix is written, so there is no fix-skeptic outcome and no regression test yet", is the state when the pass was recorded. Left by the skeptic: userguide.md:207 still says the cap "bounds the memory of a read" although a bounded read allocates about 4 times the cap; `staticcheck` and `deadcode` not run by the skeptic; the Linux x86-64 box not used (linux/arm64 in an Apple container instead). Details: [stage2-owner-decisions.md](../../decisions/stage2-owner-decisions.md), section "L2-53".

### L2-9 (low, CONFIRMED)
- **Claim.** The only code that reads real free space, freeBytes, is executed by no test on either OS (0% on macOS and Linux); MinFreeBytes tests always inject Config.statfs.
- **Grounds.** M12 left the root suite green. Auditor probes: macOS APFS freeBytes 34,749,247,488 against df 33,934,800 KiB; ext4 freeBytes 149,867,196,416 equals df Available exactly.
- **Warrant.** Decision 25 makes this the source of truth for the MinFreeBytes floor that overrides consumer protection and deletes segments.
- **Backing.** Decision 25 and Config.MinFreeBytes; guarantee 11.
- **Rebuttal.** Four lines, probed correct on both platforms: no known defect. Bsize against Frsize can differ on some network or FUSE filesystems (inferred from statfs(2), not tested).
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Coverage 0.0% for freeBytes (root alone and with -coverpkg); M12 green across ./....
- **Disposition.** Fixed in `ddce2be`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Production change, behaviour preserved: `disk.go` extracts the unexported `availBytes` (Bavail times Bsize) from `freeBytes` so a test can pin the field on a made-up Statfs_t. Tests: `TestAvailBytesIsAvailableBlocksTimesBlockSize`, `TestFreeBytesTracksRealWrites`, `TestMinFreeBytesWithTheRealStatfs`.
- **Left open by the fix** (reported by the fix-skeptic). Bfree against Bavail is pinned only on a made-up Statfs_t (the real-write test cannot separate them on APFS or tmpfs). Bsize against Frsize on network filesystems stays untested.

### L2-10 (low, CONFIRMED)
- **Claim.** The CLI's destructive and write verbs have unpinned error mappings: prune's unlink-failure exit on Linux, append error exits, recover's hole lines, -consumer-ttl parsing, a missing -name on commit and register.
- **Grounds.** The Linux profile shows write.go:238-239 at count 0 (macOS counts 1; the only test is built for darwin and uses chflags). M15, M16 and M17 each left ./cmd/quill green. A voter saw M17 give exit 1 `invalid consumer name` where exit 2 is meant.
- **Warrant.** Exit codes 1, 2 and 3 and the stdout lines are what scripts parse.
- **Backing.** Decision 100, A5-36, the user guide's exit-code table, decision 45.
- **Rebuttal.** The library is well covered and the wrappers are thin; the recover hole lines are not documented in docs/reference.md and may be incidental.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Voters applied M15-M17 to copies: suite green. The fix-skeptic applied further mutants and each fails its own test: dropped pruneErr exit (also on Linux, uid 1000), empty stdin, AppendBatch error ignored, stdin read error ignored, -consumer-ttl error exit and ignore.
- **Disposition.** Fixed in `199da64`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestPruneExitsOneWhenALaterUnlinkFails`, `TestPruneRejectsAMalformedConsumerTTL`, `TestAppendOfNothingIsAUsageError`, `TestAppendOverThePolicyPayloadLimitExitsOneAndAppendsNothing`, `TestAppendStdinReadErrorExitsOne`, `TestRecoverPrintsEachHole`, `TestCommitAndRegisterRequireAName`, `TestConsumersStaleProbeOfAConsumerInAHole`.
- **Left open by the fix** (reported by the fix-skeptic). The cmdAppend Sync-error exit and cmdPrune's Stat-failure path (write.go:229-232) stay uncovered (they need a seam). The Linux unlink test skips under root and depends on a log line's text and the oldest-first pass order.

### L2-11 (low, PLAUSIBLE)
- **Claim.** Open's refusal branches for marker damage, and age-rotation failure inside Prune, are untested.
- **Grounds.** log.go:555-562, 588-590, load.go:299-300 and 309-311 and retention.go:146-155 are uncovered on both OSes. A voter applied 8 mutants: the suite stayed green for all. M2: Open still refused through recoverSegment with a different message.
- **Warrant.** These are the decisions that stop a log with missing state from being silently re-created empty (guarantees 4 and 6) and the retention pass's behaviour when rotation fails.
- **Backing.** Decisions 62, 81, 60 d; T11 and T2.
- **Rebuttal.** Partly redundant (a second check refuses the same case with a different message); the load.go:309 branch needs a hand-edited marker. No wrong behaviour observed at HEAD.
- **Verdict.** PLAUSIBLE. Votes: 1 CONFIRMED, 1 PLAUSIBLE. Reproduced by: 1 of 2.
- **Reproduction.** Auditor probes on real files; one voter applied the mutants (green); the other read the coverage profile. The fix-skeptic disabled four Open refusals and the Prune rotation branches: each fails.
- **Disposition.** Fixed in `0ecddf3`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestOpenRefusesALogWhoseMarkerOrSegmentsAreGone`, `TestAPrunePassWhoseAgeRotationFailsStillSweepsAndALaterPassRotates`.
- **Left open by the fix** (reported by the fix-skeptic). Mutating load.go:329 (`len(ids)==0` after Open's creation step) still leaves the suite green; it is reachable only by a concurrent delete and is outside the listed branches. The garbage-marker case pins only that the text contains quill.tail.

### L2-17 (low, CONFIRMED)
- **Claim.** The documented ErrClosed behaviour after Close is only partly pinned: Flush after Close and Sync after Close.
- **Grounds.** Both survive. R15 and R16 (the Append-side equivalents) are killed. sync_test.go:92 accepts either ErrClosed or ENOSPC, so deleting the Sync case does not fail it.
- **Warrant.** A call on a closed log must not reach the file system or block; without the Flush guard flushLocked runs on a released lock and a closed descriptor (R11 makes the new test crash with a nil-pointer panic), and without the Sync case a Sync on a never-durable record waits for its context.
- **Backing.** scope.md Close row ('later calls return ErrClosed'); the Sync doc ('a waiter pending at Close gets ErrClosed unless its sequence became durable'); decision 42.
- **Rebuttal.** Flush after a clean Close with an empty buffer returns nil either way; the Sync case is reachable only after a failed final flush.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Green under both mutants; the Flush test panics under R11 (a nil pointer in flushLocked, read from the stack by the fix-skeptic) and the Sync test returns `context deadline exceeded` after 2 s under R14.
- **Disposition.** Fixed in `7716100`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestFlushAfterCloseIsErrClosedAndDoesNotWrite`, `TestSyncAfterCloseOfARecordThatDidNotBecomeDurableIsErrClosedAtOnce`.
- **Left open by the fix** (reported by the fix-skeptic). Under R11 the failure is a crash, not an assertion message.

### L2-18 (low, CONFIRMED)
- **Claim.** The write-path size limits are tested with values far from the limit, so a one-byte error in the accounting survives.
- **Grounds.** All six survive (A08 after the wall-clock test was skipped). The nearest test uses MaxBatchBytes 1000 with batches of 1068 and 868. A `len+33` mutant moves the accepted count by about 14,000 records at the default.
- **Warrant.** Decision 46 states exact numbers (500,000 one-byte records refused, about 479,000 accepted) and decision 69 says the disk bound is exact.
- **Backing.** Decisions 46, 60 d, 69; Config doc comments.
- **Rebuttal.** Equality (`>` against `>=`) is a boundary the owner may not wish to freeze; only the 34-byte constant (A01) and the MaxBatchBytes equality (A02) are quoted as numbers. The ceiling test depends on frame sizes, so it fixes the clock.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Five mutants run against the full suite by a voter (green); A08 not run by that voter. The five boundary tests fail on their mutants, and opposite-direction variants (+5, +1 slack) fail too (fix-skeptic).
- **Disposition.** Fixed in `0c08d71`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestABatchIsAcceptedAtExactlyMaxBatchBytesAndRefusedAtOneByteMore`, `TestWhileFlushesFailAnAppendIsAcceptedAtExactlyMaxBufferBytesAndRefusedAtOneByteOver`, `TestAFailedRotationAcceptsAnAppendReachingExactlyTheCeilingAndRefusesOneByteMore`, `TestAnAppendThatFillsTheBufferToExactlyFlushBytesFlushesInline`, `TestASegmentRotatesAtExactlySegmentBytesAndNotAtOneByteUnder`.
- **Left open by the fix** (reported by the fix-skeptic). The tests freeze the inclusive boundaries as behaviour; they match the decisions and Config docs as written.

### L2-19 (low, CONFIRMED)
- **Claim.** Config validation and the state decoders are tested away from their inclusive limits.
- **Grounds.** All seven survive (L05, L07 and S20 after the wall-clock test was skipped). validate does arithmetic only, so the 1.5 GiB boundary is tested without allocation.
- **Warrant.** The limits are stated as numbers in documents but the tests either use the constant itself or sit far from the limit.
- **Backing.** Guarantee 7; decisions 16, 60 d, 62, 69; format.md (ten-digit names).
- **Rebuttal.** Boundary values nobody is likely to hit; S03-S05 (the same limits in decodeTailFast) are equivalent because a fast-path refusal falls to the slow path.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** One voter re-ran three mutants (L03, L16, S17) green; the other ran all seven and saw L05, L07, L16, S20 green, while L03, S08 and S17 failed only the load-sensitive tests, so those three survive by the auditor's clean run, not cleanly by the voter's. The fix-skeptic applied all seven: each fails the new tests.
- **Disposition.** Fixed in `dfc1a9c`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestOpenWithoutAMarkerRefusesAOneByteSegmentAndCompletesAnEmptyOne`, `TestConfigRefusesAFlushThatReachesTheFileLimitExactlyAndAcceptsOneByteLess`, `TestConfigFlushIntervalOfOneMillisecondIsValidAndBelowIsRefused`, `TestTailDecodersAcceptTheInclusiveLimitsAndRefuseOneMore`, `TestReadStateAcceptsAFileOfExactlyMaxStateBytes`.
- **Left open by the fix** (reported by the fix-skeptic). L07 is caught by the test's own arithmetic assertion rather than validate's verdict; the FlushBytes term of maxActiveBytes is not tested alone.

### L2-20 (low, CONFIRMED)
- **Claim.** A refused lock (ErrLocked or any other TryLock error) leaking the descriptor is not tested.
- **Grounds.** Both survive. Compared with Sim.OpenFiles() before and after, three refused Opens leak descriptors on each path (a voter saw `leak 2 -> 5`).
- **Warrant.** holdLock documents 'closes f on failure'; a supervisor that retries Open against a live writer would leak one descriptor per attempt.
- **Backing.** The lock_unix.go doc comment; guarantee 7 and decision 16.
- **Rebuttal.** Reclaimed at process exit; the harm needs a long-lived retrying process. A leaked descriptor does not hold the flock.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** The full suite is green with both Closes removed; the two tests fail on K06 and K07 respectively.
- **Disposition.** Fixed in `bcb3266`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestAnOpenRefusedByALiveWritersLockLeaksNoDescriptor`, `TestAnOpenWhoseLockFailsWithAnotherErrorLeaksNoDescriptor`.
- **Left open by the fix** (reported by the fix-skeptic). The non-ErrLocked path uses a wrapper FS, not a real OS failure; the ErrLocked test takes about 0.6 s (the lock retry window).

### L2-21 (low, CONFIRMED)
- **Claim.** atomicfile.Write's chmod and close errors are unchecked in tests.
- **Grounds.** Both survive. The simulator has no injection point for Chmod or Close, so the existing atomicfile tests cannot reach them.
- **Warrant.** Write promises exact Perm and removal of the temp file on a failure at any step; a swallowed Chmod error installs the umask-reduced mode, a swallowed Close error installs a file whose writeback error was never seen.
- **Backing.** atomicfile.go doc ('chmod to perm exactly ... a failure at any step removes the temp file'); the Config.Perm doc; decision 40.
- **Rebuttal.** A Chmod failure on a file just created is rare; Close errors on a synced file are rare on local disks. T05 (dropping O_EXCL) is unobservable and not claimed.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Full suite green under each; a wrapper fsx.FS whose File.Chmod or File.Close fails kills each mutant exactly.
- **Disposition.** Fixed in `16207b1`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestWriteFailsWhenChmodOrCloseFails`.
- **Left open by the fix** (reported by the fix-skeptic). The test does not assert the resulting file mode (the failure path never installs a file).

### L2-22 (low, CONFIRMED)
- **Claim.** The writer's own sparse index is never compared with the index a scan of the same bytes builds.
- **Grounds.** Both survive. load_test.go checks anchor density for the scan-built index only. With 64-byte frames the writer's anchors drift to 65664 and 131264 while the scan's are at 65536 and 131072.
- **Warrant.** The two implementations are duplicates by design and nothing ties them together; a drift costs seek time up to one stride, not correctness.
- **Backing.** The load.go index description; decision 103.
- **Rebuttal.** A performance property, unobservable through the public API; the kill test is white-box (it reads l.active.index) and needs frame-aligned sizes.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Green on the suite; the equality test fails with the anchor positions shown above under F14 and F15. The frames must be calibrated to exactly 64 bytes (the sequence varint width changes).
- **Disposition.** Fixed in `8a81064`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestWriterIndexEqualsTheScanIndex`.
- **Left open by the fix** (reported by the fix-skeptic). One flush of one workload; a drift that appears only when anchors straddle several flushes is not covered.

### L2-31 (low, CONFIRMED)
- **Claim.** Time thresholds are tested only away from equality: ConsumerTTL staleness (mtime age, record age, offset == head), the age rotation and the age sweep.
- **Grounds.** Five survive, all in the keep-longer direction. The offset == head case is documented (`offset <= Head`); the time cases were documented as 'older than' while the code counts equality as past.
- **Warrant.** One second of drift in either direction passes the suite.
- **Backing.** retention.go comments, decisions 44 and 45; a voter noted that 'older than' (strict) in the docs did not match the code at equality.
- **Rebuttal.** A one-second disagreement harms nothing; the value is regression protection only.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** All five green on the suite. The fix pinned each threshold at TTL-1 s and at equality, in the library and in `quill consumers` (cmd/quill/consumers.go), and amended decision 45, scope.md, CHANGELOG and the log.go comments from 'older than' to 'at least as old' (wording only).
- **Disposition.** Fixed in `7c9a278`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Wording change, no behaviour change: comments in `log.go`, `retention.go` and `cmd/quill/consumers.go`, plus decision 45, scope.md and CHANGELOG, now say "at least as old". Tests: `TestConsumersStaleProbeActsExactlyAtTheTTL`, `TestAConsumerIsStaleWhenTheRecordAtItsOffsetIsExactlyTheTTLOld`, `TestAConsumerIsStaleWhenItsFileIsExactlyTheTTLOld`, `TestAConsumerAtTheHeadIsBehindAndOneAfterItIsCaughtUp`, `TestAgeRotationActsExactlyAtSegmentMaxAge`, `TestTheAgeSweepActsExactlyAtRetention`.
- **Left open by the fix** (reported by the fix-skeptic). The CLI test calls stale() directly because the verb reads time.Now(). 'older than' remains in build-prompt.md and the historical audit documents.

### L2-37 (low, CONFIRMED)
- **Claim.** Check and Stat are asserted only on their main paths: one-sequence overlaps and holes, a consumer one ahead or one behind, an empty log, a record exactly a day ahead, a single retained record and the active segment's bytes.
- **Grounds.** Eight survive. Each produces a wrong number or a missed or misnamed finding on a small, legal log (Records 0 with exactly one retained record; Bytes short by the active segment).
- **Warrant.** Stats.Records, Stats.Bytes and the Check problems are the operator's observability surface.
- **Backing.** The stat.go field docs and decisions 44, 66, 102.
- **Rebuttal.** Individually small boundary cases; the daily effect is Stat on a one-record log or a log with a non-empty active segment.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** All eight survive the root suite; five tests fail on them (the fix-skeptic confirmed each mutant fails at least one).
- **Disposition.** Fixed in `db9d3c7`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestCheckNamesAnOverlapAndAHoleOfOneSequence`, `TestCheckWarnsOfAConsumerOneAheadOrOneBehind`, `TestCheckDoesNotCallARecordExactlyADayAheadFutureDated`, `TestStatCountsASingleRetainedRecord`, `TestStatBytesCountsTheActiveSegmentToo`.
- **Left open by the fix** (reported by the fix-skeptic). The Bytes assertion compares with real file sizes; the CLI form was not exercised.

### L2-38 (low, CONFIRMED)
- **Claim.** Frame, window and state-restoration boundaries in load.go have no equality tests.
- **Grounds.** Five survive. codec_test.go tests the decodeFrame twin of the first boundary, but frameAt has its own copy of the comparison.
- **Warrant.** Only the maxseg row guards a documented guarantee that is more than a label (ids never reused).
- **Backing.** codec.go maxFrameBody and decision 69; the load.go size limit; the io.ReaderAt contract (reported).
- **Rebuttal.** No legal writer produces a frame of exactly maxFrameBody, so the first row changes a log label; the EOF-with-full-read row never occurs with *os.File, bytes.Reader or the Sim (a custom ReaderAt only); MaxSeg above newest needs a hand-edited marker.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** All five survive; a voter's own tests kill four, and the size-limit test follows the existing +1 pattern. It uses a 1.5 GiB sparse file and skips where the filesystem cannot make one; it ran on both macOS and the Linux box.
- **Disposition.** Fixed in `9e278df`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestRecoverTornTailDeclaringExactlyMaxFrameBodyIsTornNotDamaged`, `TestRecoverTenByteVarintRunIsBadAndNineIsTorn`, `TestWindowAcceptsEOFWithAFullFinalRead`, `TestOpenSegmentAcceptsAFileOfExactlyTheSizeLimit`, `TestOpenNeverReusesASegmentIDBelowTheMarkersMaxSeg`.
- **Left open by the fix** (reported by the fix-skeptic). Deadcode, govulncheck and the fuzz step were not run by the fix-skeptic.

### L2-39 (low, CONFIRMED)
- **Claim.** The strictjson tests for lone surrogates, the invalid-UTF-8 row and the surrogate range bounds are vacuous: their rows put the bad string in a field x that the test struct does not declare, so DisallowUnknownFields rejects them regardless.
- **Grounds.** internal/strictjson/strictjson_test.go: the four rows place the payload in "x". Removing the checkEscapes call, nine bound edits and the duplicate-key and array-frame edits all survived the existing suite.
- **Warrant.** A row that fails for another reason proves nothing about the check it names, so the package's mutation score for surrogate logic was 0.
- **Backing.** The strictjson.go package contract; decision 72; guarantee 9; format.md (a lone surrogate escape is refused).
- **Rebuttal.** The only production caller decodes into a numeric-only struct, so no reachable input distinguishes these behaviours; the gap is in the package's contract tests.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Removing checkEscapes leaves ./internal/... and the root suite green; 14 edits to checkEscapes and hex4 all survive; a test with a declared string field kills them. A voter corrected the claim: the invalid-UTF-8 gap is covered elsewhere, and the string-value mutant gives false positives, not missed duplicates.
- **Disposition.** Fixed in `6f26176`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestSurrogateEscapesAreCheckedInADeclaredField`, `TestCheckEscapesAtTheEndOfData`, `TestDuplicateKeyAfterAStringValueIsRefused`, `TestStringValuesAreNeverDuplicateKeys`.
- **Left open by the fix** (reported by the fix-skeptic). Six equivalents (i+12 to i+11, i+=11 to 10, i+=5 to 4, the extra i++ on a malformed escape, `i+1 < len` to `i+2`, foldRune min to max) stay as survivors. TestAGenuineReplacementCharacterAndAValidPairStay still asserts only the absence of an error.

### L2-49 (low, CONFIRMED)
- **Claim.** MaxReadBytes boundaries are untested.
- **Grounds.** All three survive. With 10-byte records and MaxReadBytes 20 the tree returns 2 records; S12 and S13 return 3. A Reader opened with no options returns 32 records over 40 x 512 KiB on the tree and 40 under R03.
- **Warrant.** The doc promises the read 'stops once at least MaxReadBytes of payload have been returned' and a 16 MiB default; R03 lifts the memory cap for every default Reader, including the CLI.
- **Backing.** The read.go Read doc, the ReaderOptions.MaxReadBytes doc, DefaultMaxReadBytes (log.go).
- **Rebuttal.** S12 and S13 add at most one extra record per call; R03 matters only for logs with more than 16 MiB readable at once.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** All three survive; S12 fails only the single-segment subtest, S13 both subtests and the default test, R03 the default test (a record-per-segment subtest needs SegmentBytes 1, not 100).
- **Disposition.** Fixed in `3cb3269`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestMaxReadBytesReachedExactlyStopsTheRead`, `TestReaderDefaultMaxReadBytesBoundsARead`.
- **Left open by the fix** (reported by the fix-skeptic). R03 is killed only through Reader.Read; the iterator path shares scanSource and the default but is not tested separately.

### L2-50 (low, CONFIRMED)
- **Claim.** Offset 0 on Read, Records and ReadWait is not pinned.
- **Grounds.** All three survive. On a trimmed log the tree returns GapError{1, First-1} for Read(0) and Records(0); S08 and I04 return records instead. Under S09 ReadWait(ctx, 0, 10) on an empty log never returns even after the context expires.
- **Warrant.** The code distinguishes Read(0) from Scan(0) on purpose, and S09 turns the same line into a hang, so the line is load-bearing.
- **Backing.** Decisions 84 and 38. The documents do not say in so many words that offset 0 is such an offset (MODERATE).
- **Rebuttal.** If the owner intends Read(0) to behave like Scan(0) the first assertion should go; the S09 hang assertion stands either way.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** Suite green under each; the gap test fails on S08 and I04, and the ReadWait test spins past a 2 s guard under S09. The fix-skeptic confirmed the CLI never passes 0 to Records (beginning(st) maps it).
- **Disposition.** Fixed in `6292566`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestReadAndRecordsFromZeroAfterRetentionAreAGapButScanFromZeroIsNot`, `TestReadWaitAtOffsetZeroOnAnEmptyLogWaitsAndHonoursItsContext`.
- **Left open by the fix** (reported by the fix-skeptic). The Read(0) and Records(0) assertions pin current behaviour, not a documented guarantee.

### L2-52 (low, CONFIRMED)
- **Claim.** Boundary and error-path cases in the Reader and the CLI that no test reaches (eleven mutants).
- **Grounds.** All survive the existing suite and each has a test that passes on the tree and fails on the mutant.
- **Warrant.** Each is a documented limit or error class (decisions 53, 59, 63, 94, 96; the sequence ceiling; the Reader doc's 'no sequence past its sequence') with only the far side of the boundary tested.
- **Backing.** Decisions 53, 59, 63, 94, 96; the Reader doc in reader.go; the exit-code comment of run in verbs.go.
- **Rebuttal.** Low individually. S17 needs a hand-damaged marker; C09 rests on a code comment and the owner may prefer exit 1; K09 and R27 need fault injection.
- **Verdict.** CONFIRMED. Votes: 2 CONFIRMED. Reproduced by: both voters.
- **Reproduction.** The voters ran all ten or eleven mutants against the existing suite (green) and their own tests pass on the tree and fail on each (R27 was not probed by one voter). The fix-skeptic applied all 11: each fails for the stated reason.
- **Disposition.** Fixed in `7340cc9`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestUnhealthyExitIsKeptWhenStandardOutputFails`, `TestAppendStdinExactlyAtMaxBatchBytes`, `TestFlagParsingEdges`, `TestReaderRefusesASealedSegmentThatChangedSizeEitherWay`, `TestReaderRetriesATransientMarkerReadFailure`, `TestReaderAcceptsAHeadAtTheSequenceCeiling`, `TestReaderServesNoSequencePastTheMarkersSequence`, `TestOffsetForTimeOverABadFrameWalkIsAGapOverTheSegment`.
- **Left open by the fix** (reported by the fix-skeptic). The C09 test pins the verbs.go comment ('a verb that would have exited 0 exits 1 instead'); the owner's preference is not settled. R27 and K09 depend on the simulator's Inject hook.

### L2-56 (low, PLAUSIBLE)
- **Claim.** The tail, strict-JSON and policy fuzz targets have weak invariants: FuzzDecodeTail does not assert seg <= maxseg <= maxSegmentID or fast-path agreement; FuzzUnmarshalNeverPanics asserts only no-panic; FuzzDecodePolicy never feeds an accepted policy to Config validation.
- **Grounds.** Auditor runs: a differential of decodeTail against decodeTailSlow, 300 s, 4,809,478 execs, 555 interesting inputs, no divergence; a policy-to-Config fuzz 45 s, 2.3M execs, no panic; hostile 1 MiB state files refused at 3-133 ms and about 27 MB.
- **Warrant.** Decision 105 rests on 'the fast path cannot widen what is accepted', asserted by one mutation test and not by the fuzzer.
- **Backing.** Decision 105; guarantees 7 and 9; maxStateBytes caps a state file at 1 MiB.
- **Rebuttal.** The differential found nothing and the strict path is the only one that decides anything. Memory is acceptable (bounded by maxStateBytes).
- **Verdict.** PLAUSIBLE. Votes: 1 CONFIRMED, 1 PLAUSIBLE. Reproduced by: 1 of 2.
- **Reproduction.** One voter judged it overstated: TestDecodeTailFastAgreesWithSlow already kills most fast-path widening and the real surviving case is `Off > maxSegmentFileBytes+1` in decodeTailFast. The other showed that removing checkEscapes and 14 bound edits in the strictjson path survive the suite. The fix-skeptic killed the widening mutants with FuzzTailDiff seeds or by fuzzing.
- **Disposition.** Fixed in `5a27add`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Production change, behaviour preserved: `state.go` extracts the unexported `policy.config` from `ReadPolicy` so the fuzz target can reuse the policy-to-Config conversion. Tests: `FuzzTailDiff`, `TestHostileStateFilesAreRefusedWithBoundedAllocation`.
- **Left open by the fix** (reported by the fix-skeptic). The seg<=maxseg<=maxSegmentID assertion has no seed that trips it; a dropped maxseg>maxSegmentID check is caught by existing unit tests, not by the fuzz target. The 64x allocation threshold sits above the measured 35x worst case.

### L2-57 (low, PLAUSIBLE)
- **Claim.** The CLI's number and time parsing, its record printer, consumer-name and segment-name validation, and Archive have no fuzz target; Archive has no differential against the source Reader.
- **Grounds.** No Fuzz function exists under cmd/quill or for Archive, validName or parseSegmentName. The auditor's scratch runs ran clean: FuzzParseTime 30 s (1.5M execs), FuzzPrinter 40 s (128k), an Archive differential of 1,500 seeds on the Sim with up to two damages per log.
- **Warrant.** These are the remaining places where operator-supplied bytes become values.
- **Backing.** Guarantees 7 and 9; decision 35; decisions 28, 67, 68, 85, 102; the audit prompt lists CLI parsing and Archive's reader as surfaces needing a fuzz target.
- **Rebuttal.** The printer differential and numbers_test.go cover the CLI well, Archive has seven dedicated test files, and A3-3 decided file names print unescaped.
- **Verdict.** PLAUSIBLE. Votes: 2 CONFIRMED, 1 REFUTED. Reproduced by: 0 of 3. Dissent (REFUTED): a missing fuzz target is a hardening preference, not an assertion gap; the behaviours it names are already asserted by deterministic tests.
- **Reproduction.** Absence checked by grep only; no voter re-ran the clean scratch runs.
- **Disposition.** Fixed in `1ef27ac`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. The commit also edited the T12 row of test-ledger.md. Tests: `TestArchiveAgainstTheSourceReaderOverDamagedSegments`, `FuzzParseDecimal`, `FuzzParseTime`, `FuzzPrinter`, `FuzzValidName`, `FuzzParseSegmentName`.
- **Left open by the fix** (reported by the fix-skeptic). The committed Archive differential is a weak oracle (reported by the fix-skeptic): it does not catch dropping the copySegment overlap check, removing the GapError construction or a destination marker off by one, and it skips the destination check when OpenReader fails.

### L2-58 (low, PLAUSIBLE)
- **Claim.** Three resource bounds hold when tried to violate them but no test asserts them: peak descriptors per in-flight Reader or Follow call, goroutines after Log.Close, and the MaxRecords bound under random sizes.
- **Grounds.** Auditor programs under -race on real files: 301 sealed segments with 8 goroutines of ScanAll and Records gave a peak /dev/fd of 21 over a base of 7 (about 1.75 per call); 40 parked Follows held 0 extra descriptors; after Reader use, 50 Follows, a Consumer and Close, goroutines 1 and fds 5 as at the start; 150 seeds x 500 random ops never exceeded MaxRecords plus active frames.
- **Warrant.** A regression that held a descriptor across a yield, left a goroutine behind Close, or let a record cap drift would pass every present test.
- **Backing.** Guarantees 8 and 11, decision 58, and the claim that Close stops the flusher (doc.go and scope.md do not state that last point, a voter's finding).
- **Rebuttal.** MaxBytes, the failing-flush buffer and the Reader cache are already asserted; the Reader has no goroutines and Close joins the flusher.
- **Verdict.** PLAUSIBLE. Votes: 2 PLAUSIBLE, 1 REFUTED. Reproduced by: 1 of 3. Dissent (REFUTED): partly covered already, the rest targets behaviour no document promises, and a peak counter needs a new fsx.Sim seam.
- **Reproduction.** One voter showed a +3 slack on the MaxRecords check fails the existing test while a +1 slack lies inside the documented segment granularity; another reported that removing the Close join `<-l.done` passes the suite. The fix-skeptic mutated a descriptor held across a call, an open held across a Follow wait, the join, an added goroutine in Close and two MaxRecords slacks: each fails its test.
- **Disposition.** Fixed in `f274b51`, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass; reported by the skeptic). Test change only; no production file changed. Tests: `TestConcurrentReadsHoldABoundedNumberOfDescriptors`, `TestParkedFollowsHoldNoDescriptorsAndReleaseOnCancel`, `TestCloseLeavesNoGoroutineBehind`, `TestMaxRecordsBoundHoldsUnderRandomOperations`.
- **Left open by the fix** (reported by the fix-skeptic). The randomised MaxBytes bound was not added (already covered by TestDiskBounds). The parked Reader Follow test depends on a wall-clock sample reaching the base count within a 10 s deadline.
