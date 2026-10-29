# Test ledger

Which guarantee and which numbered test ([build prompt](../design/build-prompt.md)) is
covered by which test functions. The reference implementation's behavioural tests were not
available to this build, so every test here was written from the design; the frame goldens
were recorded from quill's encoder and equal the bytes the design specifies.

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
| T12 | `FuzzDecodeFrame`, `FuzzRecoverSegment`, `FuzzDecodeTail`, `FuzzDecodeFormat`, `FuzzDecodeConsumer`, `FuzzDecodePolicy`, `FuzzUnmarshalNeverPanics` | done; 20 s each in the gate |
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
