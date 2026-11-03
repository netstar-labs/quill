# Test ledger

Which guarantee and which numbered test ([build prompt](../design/build-prompt.md)) is
covered by which test functions. The reference implementation's behavioural tests were not
available to this build, so every test here was written from the design; the frame goldens
were recorded from quill's encoder and equal the bytes the design specifies.
Updated in pass L2 (2026-11-03): the coverage ledger for both platforms and the tests that pass added are at the end.

## Guarantees ([scope.md](../design/scope.md))

| # | Guarantee | Tests |
|---|---|---|
| 1 | Order | `TestScanAllReadAndAReaderAgreeRecordForRecordAcrossManySegments` (T4), `TestReadsFromAnyOffsetAcrossSegmentsAndWhileAppending`, `TestRecordsYieldsExactlyWhatReadReturns…` |
| 2 | Durable before visible | `TestAReaderSeesOnlyWhatTheMarkerNames…`, `TestTheVerbsRunBesideALiveWriter…`, `TestAReaderFollowingAWriterInAnotherProcess…` (T6) |
| 3 | A crash loses a suffix, not a middle | `TestCrashPropertyAtEveryOperationAgainstTheSimulator`, `TestCrashPropertyWithConsumersPruningAndAReader` (T1), `TestAKilledWriterLosesNoAcknowledgedRecord…`, `TestTornTailsAtEveryPositionAboveTheMarker…`, `TestNothingAfterTheFirstBadFrameIsTrusted…` (T2) |
| 4 | Sequences are contiguous | the T1 tests, `TestAMarkerAboveTheLastValidFrameFailsOpen…`, `TestASealedSegmentWithDamageAnywhereIsCorruption` |
| 5 | One writer, one process per consumer | `TestASecondWritableOpenFailsFast…`, `TestTheLockIsReleasedWhenTheHolderIsKilled` (T23), `TestConsumerLifecycleProtectionFlagsLocksAndNames` (T9) |
| 6 | Corruption is never served silently | `TestDamageBelowTheMarkerFailsOpenAndLeavesTheFileUnchanged`, `TestAReaderReportsGapsForDeletedAndDamagedSegments…` (T7), `TestReaderCheckExaminesSealedSegmentsWhole…` |
| 7 | Bounded allocation | `TestDecodeFrameRefusesEveryDamage`, `TestOpenSegmentRefusesSymlinksFIFOsAndOversizeFiles`, `TestStateFilesAreWrittenAtomicallyAndRead…`, `TestABatchOverMaxBatchBytes…` (T22) |
| 8 | Retention deletes only whole sealed segments, never reuses an id | `TestTheAgeSweepDeletesExpiredSealedSegmentsNeverTheActiveOne…`, `TestAMarkerAboveTheLastValidFrame…` (ids after expiry) |
| 9 | State files atomic and strict | `TestTailRoundTripsAndIsStrict`, `TestAppendTailMatchesJSON`, `TestDecodeTailFastAgreesWithSlow` (decision 105), `TestFormatMarkerRoundTrips…`, `TestConsumerFileRoundTrips…`, `TestACrashAtAnyPointLeavesTheOldOrTheWholeNewFile`, the fuzz targets (T12) |
| 10 | A stale marker hides, never exposes | `TestAStaleMarkerWithAcknowledgedRecordsAboveItKeepsAllOfThem`, `TestAReaderSeesOnlyWhatTheMarkerNames…`, T6 |
| 11 | Bounded disk | `TestDiskBoundsMaxBytesHoldsUnderAFastWriter…` (T13), `TestAFullDiskFailsAppendsKeepsRecordsIntact…` (T14, CI only) |
| 12 | A stale consumer cannot pin the disk | `TestConsumerAgingUnderConsumerTTL` (T16), `TestTheArchiverIsAProtectedConsumerUntilACapOverridesIt` |
| 13 | A failed write never corrupts | `TestAShortWriteOrENOSPCTruncatesBack…`, `TestAnFsyncErrorIsFatal…`, `TestAMarkerFailureAfterTheDataFsync…`, `TestAFailureCreatingTheNextSegment…` (T3) |
| 14 | Reads never skip a gap | T7 tests, `TestAMiddleSegmentDeletedByHandIsAHoleFoundAtLoad`, `TestAnIteratorYieldsAnErrorOnceAndAGapIsResumedFromToPlusOne` (T25) |

## Numbered tests

| T | Test functions | Disposition |
|---|---|---|
| T1 | `TestCrashPropertyAtEveryOperationAgainstTheSimulator`, `TestACrashDuringRecoveryItselfRecovers`, `TestCrashPropertyWithConsumersPruningAndAReader`, `TestAKilledWriterLosesNoAcknowledgedRecordAndRecoversAContiguousPrefix` | done; sampled subsets when more than six items are unsynced |
| T2 | `TestAStaleMarker…`, `TestAZeroedPageFollowedByIntactFrames…`, `TestDamageBelowTheMarker…`, `TestAMarkerAboveTheLastValidFrame…`, `TestTornTailsAtEveryPosition…`, `TestAMarkerNamingASealedSegment…`, `TestNothingAfterTheFirstBadFrame…`, `TestRecoveryParsesForward…` | done |
| T3 | `TestAShortWriteOrENOSPC…`, `TestAnFsyncErrorIsFatal…`, `TestAMarkerFailureAfterTheDataFsync…`, `TestAFailureCreatingTheNextSegment…` | done |
| T4 | `TestScanAllReadAndAReaderAgree…` | done |
| T5 | `TestASlowFsyncOnOneLogNeverDelaysAnother` | done |
| T6 | `TestAReaderFollowingAWriterInAnotherProcess…` | done |
| T7 | `TestAReaderReportsGaps…`, `TestAMiddleSegmentDeletedByHand…`, `TestTheAgeSweep…` (resume with Commit(To+1)) | done |
| T8 | `TestCommitAndLagAgree…`, `TestAConsumerAheadOfTheLogOrFromAnotherLogIsRefused` | done |
| T9 | `TestConsumerLifecycle…`, `TestTempFilesYoungerThanAnHour…`, `TestAProtectedConsumerKeeps…` (unparseable file), CLI: `TestCheckReportsUnhealthyWithExitThree…` | done |
| T10 | `TestTheAgeSweep…` (late consumer) | done |
| T11 | `TestAnEmptyDirectoryIsANewLog…`, `TestANewerFormatMarker…`, `TestACrashAtEveryStepOfCreation…`, `TestTwoProcessesCreatingTheSameLogAtOnce…`, reader `ErrNoLog` in `TestAReaderSeesOnly…` | done |
| T12 | `FuzzDecodeFrame`, `FuzzRecoverSegment`, `FuzzDecodeTail`, `FuzzDecodeFormat`, `FuzzDecodeConsumer`, `FuzzDecodePolicy`, `FuzzUnmarshalNeverPanics`, `FuzzWalkChunks`, `FuzzReaderDir` (the read path from mid-file, and a Reader over a whole directory; stage 2), `FuzzValidName`, `FuzzParseSegmentName`, and in cmd/quill `FuzzParseDecimal`, `FuzzParseTime`, `FuzzPrinter` (stage 2), with `TestArchiveAgainstTheSourceReaderOverDamagedSegments` as the seeded differential for Archive | done; 20 s each in the gate |
| T13 | `TestDiskBoundsMaxBytesHolds…`, `TestEveryConfigDefaultAndRefusal` | done |
| T14 | `TestAFullDiskFailsAppends…` | done; runs in CI on a tmpfs (`QUILL_FULLDISK_DIR`) and was run by hand on a Linux server (see untestable-without-x.md) |
| T15 | `TestRollDoesNothingOnAnEmptySegment…`, T1's `Roll` steps (a crash at each step of rotation), `TestAMarkerNamingASealedSegment…`, `TestArchiveCopies…`, `TestArchiveStopsAtACorruptSegment…`, `TestTheArchiverIsAProtectedConsumer…`, `TestArchiveOnTheRealFileSystem` | done |
| T16 | `TestConsumerAgingUnderConsumerTTL`, `TestConsumerFilesAreWrittenAtomicallyAndCommitsLeaveTheirMtime` (the premise) | done |
| T17 | `TestSizeAccountingIsMeasured` | done |
| T18 | `TestManyProducersSyncingConcurrentlyShareFsyncs`, `TestALoneSyncReturnsWellBefore…`, `TestPendingSyncWaitersGetErrClosed…`, `TestWithAFlushFailingSync…`, `TestAProducerWaitingInSyncHoldsNoLock`, `TestAKilledWriter…` (sync rounds) | done |
| T19 | `TestAReadAtTheStartOfAThousandSegmentsCostsAboutWhatOneAtTheTailDoes`, `TestReadAllocationsDoNotGrowWithTheSealedSegmentCount` (decision 103) | done; cost measured in file operations on the simulator |
| T20 | `TestReadableOutputIsLosslessInTextAndJSON`, `TestPrintMatchesReference` (the printer's appenders against `encoding/json`, `strconv.Quote` and `fmt`, 20,000 random records per mode), `TestTheVerbsRunBesideALiveWriter…` | done |
| T21 | `TestTheClockGuardDeletesAtMostAQuarter…`, `TestReaderCheckExamines…` (clock flags), CLI `prune -force` in `TestTheVerbsRunBesideALiveWriter…` | done |
| T22 | `TestEveryConfigDefaultAndRefusal`, `TestARecordOfExactlyMaxPayloadBytes…`, `TestABatchOverMaxBatchBytes…`, `TestWhileFlushesFailAnAppendIsAccepted…`, `TestMaxReadBytesReturnsAtLeastOneRecord…` | done |
| T23 | `TestASecondWritableOpenFailsFast…`, `TestTheLockIsReleasedWhenTheHolderIsKilled`, CLI `TestTheVerbsRunBesideALiveWriter…` | done |
| T24 | `bench_test.go`, `benchmark-baselines.md` | done; the larger-than-memory drain was run on the Linux server, and the allocation pass's before/after rows are in the baselines |
| T25 | `TestRecordsYieldsExactlyWhatReadReturns…`, `TestAnIteratorYieldsAnErrorOnce…`, `TestBreakingOutOfAnIteratorLeaksNoDescriptor…` | done |
| T26 | `TestOffsetForTimeEqualsALinearScanForEverySecond`, `TestOffsetForTimeOnDamageEmptyLogsAndNonMonotonicTimes` | done |
| T27 | `internal/fsx/crash_test.go` | done |

## Coverage ledger (refreshed in pass L2, 2026-11-03)

The numbers below were measured at the start of pass L2 by the coverage auditor, on a scratch copy of the tree made with `git archive` (the recovery auditor names `5a70379` as the base; the coverage auditor
says "HEAD"), before the 52 L2 fixes and before some pass L1 fixes. They have **not** been re-measured on the current tree (`f274b51`); the 52 L2 commits and the L1 test commits only add tests, so every figure
should be equal or higher, which is inferred. To refresh: `GOWORK=off go test -count=1 -cover ./...` per platform, and `GOWORK=off go test -count=1 -coverpkg=./... -coverprofile=c.out ./... && go tool cover -func=c.out`.
Method (reported): `go test -count=1 -cover`, no `-race`, go1.27.3; macOS on darwin/arm64, Linux on Debian 13 / ext4 / linux/amd64 through the shared box. Neither platform ran T14 or the 20 s fuzz targets.

### Per package

| Package | macOS (darwin/arm64) | Linux (Debian 13, ext4, linux/amd64) |
|---|---|---|
| `quill` (root) | 94.2% | 94.2% |
| `cmd/quill` | 93.2% | 93.0% |
| `internal/fsx` | 88.6% | 88.6% |
| `internal/atomicfile` | 96.6% | 96.6% |
| `internal/strictjson` | 94.5% | 94.5% |
| `examples/pipeline` | 72.2% | 72.2% |
| `examples/log`, `examples/topics` | 0.0% (no tests) | not reported |
| `hack/powertest`, `hack/skewopen` | 0.0% as reported | not reported |

`hack/powertest` has a `main_test.go`, so "no tests" is not exact for it; its test re-executes the test binary as `main` in a child process (`POWERTEST_AS_MAIN`), which `go test -cover` does not count (inferred from the file, not checked). `hack/skewopen` has no test file; it is driven by `hack/skew.sh`.

Cross-package profile (`-coverpkg=./...`, merged): **89.4%** of all statements on both platforms. Root package 94.5% (2053 of 2173 statements on macOS, 2054 on Linux), `cmd/quill` 93.2% (macOS) and 93.0% (Linux), `internal/fsx` 95.0% (higher than its own 88.6%
because the root package's tests exercise it), `internal/atomicfile` 96.6%, `internal/strictjson` 94.5%.

### Where the two platforms differ

Diffing the two merged profiles block by block (reported), exactly two blocks differ:

| Block | Reached on | Why |
|---|---|---|
| `cmd/quill/write.go:238-239` (`cmdPrune`'s `pruneErr != nil` return) | macOS only | its only test, `prune_unlink_darwin_test.go`, uses `chflags` (class b). Pass L2 added `TestPruneExitsOneWhenALaterUnlinkFails` (L2-10, `199da64`), which makes the second unlink fail on Linux by `chmod 0500` of the log directory from the verb's own stderr hook. The L2-10 fix-skeptic's Linux coverage run, as uid 1000, shows the block at count 1 (reported). The test skips under root |
| `reader.go:266` (`s = cached // another call built it meanwhile`) | Linux only | a concurrency race between two builders of the same readerSegment: flaky by construction (class c, a timing seam). No test pins it |

No `lock_unix.go` or `disk.go` branch differs between the platforms. Skipped on both: `TestAFullDiskFailsAppendsKeepsRecordsIntactAndRecoversWhenSpaceReturns` (T14, needs `QUILL_FULLDISK_DIR`; the ledger says it runs in CI) and `TestRecordGoldens` (recording mode); the darwin file is excluded on Linux.

### Uncovered branches at the audited base, by class

Line numbers are those of the audited base and have shifted in the tree.

| Class | Meaning | Locations (reported) |
|---|---|---|
| (a) error path an injected fault would reach (`Sim.Inject`) | one-line returns whose absence would be loud | `archive.go` 58, 72, 167, 171, 177, 187, 205, 212, 244, 253, 270, 279; `consumer.go` 68, 109, 132, 148, 214, 218, 236; `stat.go` 111, 118, 182, 186, 234, 302; `reader.go` 98, 208, 340, 521; `fsx/os.go` 90-93 (directory fsync failure). The auditor's note: one test per file that injects EIO on `OpReadDir` or `OpLstat` of `quill.tail` would cover most. L2-2, L2-3, L2-4 and L2-6 closed some of these |
| (b) platform branch | reachable on one OS only | `cmd/quill/write.go:238`, above |
| (c) needs a seam that does not exist | | `write.go:238` on Linux (since closed by a real-file-system trick, above); `reader.go:266` (race); `fsx/os.go:76` (EINTR retry on rename); `load.go:83-84` and `read.go:329` (a file swapped to a non-regular file between the `Lstat` and the open, or a size change between view and open: only a hook between two operations could model it). See [untestable-without-x.md](untestable-without-x.md) |
| (d) dead or defensive | no path reaches it, or a comment says it cannot happen | `fsx/os.go:21-23` (O_WRONLY translation: no non-simulator caller passes O_WRONLY); `flush.go:141` and `codec.go:142-156` (frameHeader on frames the writer just encoded); `read.go:109`; `read.go:275` (a segment ending before `s.last` while its size is unchanged, barred by the size check at `reader.go:467`); `read.go:236` (a record past the marker; mutant M21 left the suite green, inferred redundant, no case constructed); `log.go:620-622` (Close's extra publish, redundant with `flushLocked`, mutant M10); `load.go:305-307` (no segment files after create, reachable only by a concurrent delete); `state.go:174` and `atomicfile.go:60` (`rand.Read` failure: unreachable by the documented contract, see untestable-without-x.md); `disk.go:10-11` (Statfs error: since covered by the ENOENT assertion in `diskfull_test.go`, reported by the L2-9 fix-skeptic) |

### Exported API at 0.0% at the audited base

`Reader.ID` (reader.go:126), `Reader.Scan` (reader.go:179), `Consumer.Name` (consumer.go:181) and `Consumer.Scan` (consumer.go:258). The tree now has calls to all four (`reader_test.go` lines 724 to 744, added by the pass L1 commit `7497a80`;
read by the recorder, not run), so this item is closed, and the coverage figure above predates it.

### Ledger checks (reported by the coverage auditor)

- Every `Test` and `Fuzz` name in the table above resolved to an existing function at the audited base (exact name, or the prefix before an ellipsis; a script run over 372 names). T1 to T27 each had at least one named test.
- Tests that cannot fail for their stated reason, found by an AST scan, are listed in [defects-log.md](defects-log.md) (D5).

## Tests added in pass L2

52 commits, one per fixed item, each test-only except three with behaviour-preserving production edits (L2-9, L2-31 wording, L2-56). Each new test was shown by a fix-skeptic to fail under the mutant it names. The per-item
record, with the mutants and the test names, is [mutation-hardening.md](mutation-hardening.md). By guarantee in this ledger:

| # | Guarantee | Added in pass L2 (item, commit) |
|---|---|---|
| 1 | Order | single-record wake of `ReadWait` and `Follow` (L2-43 `5f09ee6`); reads from every offset across several sparse-index anchors (L2-48 `ad06516`); offset 0 (L2-50 `6292566`) |
| 2 | Durable before visible | marker one sequence ahead of its frames, and damage in the active segment (L2-44 `cbc1fef`); an active segment shorter than the marker (L2-3 `786a1c5`); Close makes the marker durable (L2-13 `3f77076`); the flusher retries a failed publish on an idle log (L2-16 `216c111`) |
| 3 | A crash loses a suffix, not a middle | a CRC-valid wrong-sequence frame at a non-empty marker (L2-24 `dac7bfb`); forged frame bodies (L2-1 `e874d01`); a directory-fsync error after the rotation marker fails the log (L2-15 `55a7e97`); an inline-flush failure returns `ErrFailed` (L2-12 `2fc3030`) |
| 4 | Sequences are contiguous | hole and head bookkeeping in `load()` (L2-25 `d177d2c`); `maxseg` never lowered (L2-14 `ab45579`, L2-38 `9e278df`); Archive gap edges (L2-34 `fcaba49`) |
| 5 | One writer, one process per consumer | a closed consumer handle cannot overwrite (L2-6 `7b26d68`); a refused lock leaks no descriptor (L2-20 `bcb3266`); the writer restarts beside a caught-up protected consumer (L2-32 `c824158`) |
| 6 | Corruption is never served silently | read-time gap and corrupt mapping after Open (L2-4 `168221a`); gap edges (L2-45 `ed6c6df`); Check and Stat edges (L2-37 `db9d3c7`, L2-36 `5638419`); the Reader fuzz targets (L2-55 `a891274`) |
| 7 | Bounded allocation | `MaxReadBytes` and the iterators' batch bound (L2-49 `3cb3269`, L2-47 `126ef3d`); size-limit boundaries (L2-18 `0c08d71`, L2-19 `dfc1a9c`); hostile state-file allocation (L2-56 `5a27add`); descriptors, goroutines and the `MaxRecords` bound (L2-58 `f274b51`). L2-53 (fixed 2026-11-03: each record counts its payload plus 48 bytes against `MaxReadBytes`, `read_overhead_test.go`) was the known hole |
| 8 | Retention deletes only whole sealed segments | the age sweep keeps a damaged segment (L2-30 `aa039fa`); time thresholds at equality (L2-31 `7c9a278`) |
| 9 | State files atomic and strict | atomicfile `Chmod` and `Close` failures (L2-21 `16207b1`); strictjson surrogates and duplicates (L2-39 `6f26176`); tail, policy and strictjson fuzz oracles (L2-56) |
| 11 | Bounded disk | `MaxBytes` and `MinFreeBytes` at equality and with the age sweep (L2-27 `b040221`); the active segment and `MaxRecords` (L2-28 `fcb08a5`); the real `freeBytes` (L2-9 `ddce2be`) |
| 12 | A stale consumer cannot pin the disk | protected-consumer retention (L2-26 `d29328e`); consumer-file errors (L2-33 `143367d`); `Dropped` accounting (L2-29 `ebf0fcf`); Archive ordering and copy guards (L2-2 `5037042`, L2-35 `8b0e6c2`) |
| 14 | Reads never skip a gap | L2-4, L2-44 and L2-45 above |

Other additions: the CLI error mappings and boundaries (L2-10 `199da64`, L2-41 `547e387`, L2-42 `eb659ae`, L2-51 `1e24b7c`, L2-52 `7340cc9`); Open's refusals when state is gone (L2-11 `0ecddf3`); `ErrClosed` after Close (L2-17 `7716100`);
the sparse index equals the scan index (L2-22 `8a81064`); five fuzz targets and an Archive differential (L2-57 `1ef27ac`); `Reader.ReadWait` ends on Close (L2-46 `d0e6ec2`); and the removal of wall-clock bounds (L2-8 `743a5c5`).
The T12 row above already lists the new fuzz targets.
