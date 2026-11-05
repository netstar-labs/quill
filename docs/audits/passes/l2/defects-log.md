# Defects log: pass L2

Date 2026-11-03, against the tree at `f274b51`. Pass L2 (test, benchmark and hardening suite) looked for places where the suite would not notice wrong code, and
for resource bounds and isolation rules that could be broken from outside. The full record, with every item's claim, grounds, warrant, backing, rebuttal, votes and
reproduction, is [mutation-hardening.md](mutation-hardening.md). This file lists only what was actually wrong, with its root cause, its fix and the regression test.

**Result.** One production defect candidate, held for the owner (D1). No production defect was fixed in this pass: every mutant that survived the suite turned out to be a
missing test, and each voter and fix-skeptic reported that the new test passes on the unmodified code (reported by the skeptics). Three defects in the tests themselves were fixed
(D2 to D4). One group of weak tests was recorded and left alone (D5). The five production files the pass changed hold behaviour-preserving
extractions and wording only (see the summary in mutation-hardening.md).

| Id | Item | What | Where | Status |
|---|---|---|---|---|
| D1 | L2-53 | `MaxReadBytes` bounds payload bytes only, so a read of tiny records is unbounded in memory | production, `read.go` and `iter.go` | owner-approved 2026-11-03; fixed (charge 48 bytes per record) |
| D2 | L2-2 | a data race in two test hooks made the `-race` gate red | `check_vanished_test.go`, `reader_vanished_test.go` | fixed in `5037042` |
| D3 | L2-8 | a wall-clock assertion and sleep-based synchronisation made the gate flaky and the mutation runs unreliable | `consumer_protect_test.go`, `sync_test.go` | fixed in `743a5c5` |
| D4 | L2-39 | strictjson tests that named surrogate and UTF-8 checks could not fail for that reason | `internal/strictjson/strictjson_test.go` | fixed in `6f26176` |
| D5 | (no item) | three tests assert less than their names say; one accepts two outcomes | see below | recorded, not changed |

## D1. A read of tiny records allocates without a bound (L2-53, medium, CONFIRMED, held)

**Symptom.** With the default `Config`, `Log.Read(1, 0)` on a log of 4,000,000 empty-payload records returned all 4,000,000 and allocated 788 MB; on 17,000,000 one-byte records it returned
16,777,216 records, allocated 3,892 MB in total and the process peaked at 2.78 GB resident, against a documented `MaxReadBytes` of 16 MiB. `Records()` iterated the same counts with
787 MB and 3,935 MB. Two voters reproduced it independently (1,000,000 records: 206 MB; 2,000,000 records: about 403 MB). All figures are reported by the auditor and the voters; the recorder ran nothing.

**Root cause (verified by reading the tree).** `scanSource` in `read.go` keeps a running count `bytes` of payload bytes only (line 259, `bytes += len(rec.Payload)`) and stops when it reaches the
cap (lines 197 and 266). A record with an empty payload adds 0, so the loop is bounded only by `limit`. `Read(offset, 0)` passes `limit` 0, which means no limit, and so does the iterator (`iter.go:63`:
`scanSource(src, cur, 0, src.maxReadBytes(), ...)`). Each record held in the batch costs a `Record` value and a copy of its payload that the byte count does not include; at the 17,000,000-record
measurement that is about 230 bytes allocated per record (3,892 MB over 16,777,216 records; inferred from the reported totals). The cap therefore bounds the payload bytes of a read, and
not the memory a read takes.

**Why it is not a crash and not clearly a defect.** The memory is garbage after the call. `MaxReadBytes` is documented as payload bytes (scope.md, decision 79, `Config`), and a caller of `Read`
can pass a limit. `Records`, `Follow` and the consumers' iterators have no limit parameter; their batch size is the iterator's own choice, and it is the byte cap that is supposed to bound it
(iter.go header, decision 58). A log of counters or heartbeats is an ordinary workload, not only a hostile one.

**Fix.** Owner-approved 2026-11-03 and written: `scanSource` charges each record `len(payload) + 48` bytes (`readRecordOverhead` in `read.go`) against `MaxReadBytes`, which `Read`, `Records` and `Follow` all pass. The recommended answer is in [stage2-owner-decisions.md](../../decisions/stage2-owner-decisions.md), section "L2-53".

**Regression test (written; `read_overhead_test.go`, red before the fix).** 1,000,000 empty-payload records, default `MaxReadBytes`: assert `len(Read(1,0))` is far below 1,000,000 and that `Records` batches are bounded.
If the owner prefers to document the behaviour instead, no test is needed and the doc wording changes.

## D2. A data race in the vanished-segment test hooks (found while fixing L2-2)

**Symptom.** `go test -race ./...` failed, so the commit gate was red. On the tree before `5037042`, `go test -race -count=30` over the two vanished-segment tests printed 2 `DATA RACE`
reports and `FAIL`; after the fix, 0 (reported by the L2-2 fix-skeptic).

**Root cause.** Two test hooks assigned `s.Inject` from inside a hook, the one that removes a file on its first open (`removeOnOpen`) and the one in the middle-segment `Check` case. `Sim` reads `Inject`
under its own lock before every operation and documents that `Inject` is to be set only while no goroutine is using the simulator (`internal/fsx/crash.go`). A hook that writes the field while the
code under test runs other goroutines is the unsynchronised writer. Whether `Reader.Check`'s parallel segment verification (pass L1, `1aff9f0`) exposed it is inferred and was not checked.

**Fix.** The hooks stop assigning `s.Inject`. `removeOnOpen` keeps its state under a `sync.Mutex`; the middle-segment case uses a `sync.Once`. A locked setter in `fsx.Sim` would have added an exported symbol and
was not needed.

**Regression test.** The two tests themselves, under `-race`: `go test -race -count=30` on them, 0 reports after. The commit also added the tests for L2-2 proper
(`TestArchiveSyncsTheDestinationDirectoryAfterItsMarkerAndBeforeCommittingTheOffset`, `TestArchiveFailingTheFinalDestinationDirectorySyncReturnsItAndLeavesTheOffsetUncommitted`).

## D3. A load-sensitive assertion and sleep-as-synchronisation (L2-8)

**Symptom.** `TestOpenWithAProtectedConsumerRunningDoesNotNeedItsLock` failed 3 of 3 immediate reruns on an unmodified tree at load average 7 to 8 (`reopen took 117ms`); another voter saw 3 failures
in 20 runs under disk load; the fix-skeptic could not reproduce it on macOS under CPU load (reported). In the mutation runs the same test produced 28 false kills (write run) and 14 (reader run),
so every kill had to be re-run alone or with that test skipped.

**Root cause.** The test asserted `time.Since(start) > lockRetry/2` (a stopwatch), which is redundant: if `Open` needed the live consumer's lock it returns `ErrLocked` after the lock-retry window, which the
test already checks. `sync_test.go` used 20 ms sleeps to mean "the flush is inside its fsync", and 20 ms, 200 ms and 150 ms plus 50 ms bounds. Under load the sleep can elapse before the flusher
reaches the injected sleep, so an isolation test passes without testing anything (inferred from reading, not demonstrated).

**Fix (`743a5c5`).** The stopwatch is deleted. The 20 ms sleeps are replaced by a gate the injected fsync or write signals, and the time bounds by completion checks with a 30 s deadlock-detector timeout.
`TestAProducerWaitingInSyncHoldsNoLock` no longer asserts that the append completes before the fsync is released, because the flush holds the log lock across the fsync by design.

**Regression test.** The rewritten tests. The fix-skeptic applied three production mutants and each fails a rewritten test: `consumerStore.protect` skipping its lock-free path, a package-level mutex
around the data fsync (so two logs share a lock), and `Sync` holding the log lock while it waits. The tests also passed 15 times under -race with 24 busy loops on macOS and 8 on the Linux box.

**Left open.** Other `time.Since` bounds with a margin of 1 s or more remain (sync_test.go:128, log_test.go:271, iter_test.go:123, wait_failed_test.go:129).

## D4. Strictjson tests that could not fail for the reason they named (L2-39)

**Symptom.** Removing the whole `checkEscapes` call from `strictjson.Unmarshal`, or editing any of its surrogate range bounds, left the strictjson and root suites green.

**Root cause.** Four rows of `TestUnmarshalRefusesEveryDamagedShape` (lone low surrogate, lone high surrogate, high then non-low, invalid UTF-8) put the bad string in a field `x` that the test struct
does not declare, so `DisallowUnknownFields` rejected each row regardless of what `checkEscapes` does. A row that fails for another reason proves nothing about the check it names. Duplicate-key
detection after a string-valued member and inside arrays was also unasserted.

**Fix (`6f26176`).** The surrogate tests decode into a struct that declares `X`, with escapes built at run time; new tests pin every surrogate range bound, the pair-length rule, `checkEscapes` at end of data, duplicate keys
after a string value, and string values and array elements never being keys. The four vacuous rows were removed.

**Regression test.** `TestSurrogateEscapesAreCheckedInADeclaredField`, `TestCheckEscapesAtTheEndOfData`, `TestDuplicateKeyAfterAStringValueIsRefused`, `TestStringValuesAreNeverDuplicateKeys`. The fix-skeptic killed
deleting `checkEscapes`, all the bound edits at the cited lines, `i+=11` to 12 and `i+=5` to 6, dropping the `i++` at the malformed-escape skip, and the duplicate-key and array-frame edits. Six equivalent re-spellings still survive
(listed in mutation-hardening.md).

**Reach.** Low. The only production caller, `decodeTailSlow`, decodes into a numeric-only struct, so no reachable input distinguishes these behaviours; the gap was in the package's own contract tests.

## D5. Tests that assert less than their names say (recorded, not changed)

The coverage auditor scanned every test function in the module, excluding `examples`, for a test with no failure call guarded by anything other than `err != nil`, and read each hit by hand. Genuine and low value:

- `TestAGenuineReplacementCharacterAndAValidPairStay` (internal/strictjson/strictjson_test.go:93): checks only that `Unmarshal` returns no error; it never inspects the decoded string. Still so in the tree (verified by reading).
- `TestOpenAheadWarningAbsentWithoutALogger` (ahead_warn_test.go:132): asserts only that `reopenWithLogger` does not fail; its stated claim is "Open succeeds". Still so in the tree (verified by reading).
- `TestAFIFOInPlaceOfALockFileDoesNotBlockOpenOrArchive` (fifo_test.go): ignores the `Open` and `Archive` results and asserts only that nothing hangs, which matches its name (reported).
- `sync_test.go:92` accepts two outcomes (`ErrClosed` or `ENOSPC`). Pass L2 added `TestSyncAfterCloseOfARecordThatDidNotBecomeDurableIsErrClosedAtOnce` (L2-17), which pins the first of the two.

Several tests delegate their assertion to a helper that does assert (`wantTransitions`, `closeUnderRead`, `wantShape`, `wantReadEIO`); those are not vacuous. `TestRecordGoldens` is a recorder, skipped
by default. The scan cannot see an assertion that is present but trivially true; none of that kind was audited. No ledger item was filed for D5, so no commit changed these tests.
