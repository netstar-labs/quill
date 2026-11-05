# Pass L1: verification and optimisation

Date 2026-11-02. Audited tree: 5a70379 (the "stage 2 A1i record" commit). Fixes from this pass sit on top of it; the
branch tip when this record was written is 15ddde4. Pass L1 is the fresh-eyes validation of quill v0.1.0 plus its
stage 2 fixes before a public release, run by six independent auditors in four parts: comment against code (three
auditors, by file), dead code and debt, operational validation (macOS and Linux), and optimisation. The specification is
`docs/design/scope.md`, `docs/design/decisions.md` and `docs/format.md`; the code was the thing under audit, so a
passing test and a doc were both treated as claims, not evidence. Every candidate was attacked by two or three
skeptics, who read the code and re-ran the reproductions, and every fix was re-attacked by a fix-skeptic. What the
audit held back for the owner is in [stage2-owner-decisions.md](../decisions/stage2-owner-decisions.md), section "Pass L1".

Labels: **verified** means an auditor, skeptic or the recorder ran it or read the exact lines; **reported** names the
document that says it; **inferred** says from what. File and line anchors are to the audited tree 5a70379 unless a
commit is named; the fixes have since moved some lines. Reproduction programs lived in throwaway copies of the
repository made with `git archive` in the session scratch area; nothing under audit was edited by an auditor or
skeptic. This record was compiled from the pass ledger by the recorder and was not re-run: every number below is the
auditors' or skeptics' reported figure unless it says "recorder".

## Top line

50 items. 35 CONFIRMED, 6 PLAUSIBLE, 5 REFUTED, 4 prior-deliberate (decided in round 1, not re-voted).
Merged duplicates: L1-45 into L1-39, L1-37 into L1-22, L1-46 into L1-44.

| Disposition | Items | Count |
|---|---|---|
| Fixed, fix-skeptic FIXED-VERIFIED | L1-1, 2, 6, 7, 8, 9, 14, 15, 18, 19, 20, 22, 23, 24, 25, 27, 28, 29, 31, 39, 40, 43, 48, 49, 50, 52, 53 | 27 |
| Fix written, rejected by the fix-skeptic, not in the tree | L1-3, 10, 21 | 3 |
| Held for the owner | L1-4, 5, 11, 12, 16, 26, 51 | 7 |
| Recorded, not applied (confirmed safe, low value) | L1-32, 33, 35, 38 | 4 |
| Refuted (kept with the reason) | L1-13, 30, 41, 44, 47 | 5 |
| Decided in round 1 (prior-deliberate) | L1-17, 34, 36, 42 | 4 |

| Counted over the 50 items | |
|---|---|
| By severity as filed | 6 medium (L1-1, 11, 12, 39, 40, 42), 44 low |
| Skeptic votes | 98 (recorder's count from the ledger: 40 items with 2 votes, 6 items with 3, none for the 4 prior-deliberate items) |
| Reproduced | 32 of the 46 voted items have a skeptic vote recorded as reproduced (a program or test run; for L1-30 a grep). The other 14 have no skeptic reproduction: for 5 of them (L1-4, 5, 16, 21, 28) the auditor ran one, and the rest (L1-7, 8, 9, 10, 18, 19, 22, 27, 31) are comment-only findings settled by reading the exact lines |
| Fix-skeptic | 30 fixes attacked: 27 FIXED-VERIFIED (6 of 6 lenses pass; 7 of 7 for L1-14, L1-15 and L1-48), 3 rejected (L1-3, L1-10, L1-21) |
| Vote splits worth knowing | L1-10 (CONFIRMED, REFUTED, CONFIRMED, filed PLAUSIBLE), L1-12 (REFUTED, CONFIRMED, CONFIRMED, filed PLAUSIBLE), L1-31 (PLAUSIBLE, REFUTED, PLAUSIBLE), L1-41 (CONFIRMED, REFUTED, REFUTED), L1-47 (CONFIRMED, REFUTED, REFUTED), L1-51 (REFUTED, CONFIRMED, PLAUSIBLE), L1-13 (REFUTED, REFUTED) |

**Update 2026-11-03 (pass L1h).** The owner took the recommended answer on 2026-11-03 for L1-4, L1-5, L1-11, L1-12, L1-16 and
L1-26 and for L1-51 (keep the 64 KiB stride), and asked for the three rejected fixes (L1-3, L1-10, L1-21) to be finished. The
state of the 50 items is now:

| Disposition on 2026-11-03 | Items | Count |
|---|---|---|
| Fixed, fix-skeptic FIXED-VERIFIED | the 27 above, plus L1-5 (867f4ed), L1-11 (ce47243), L1-12 (be8cb15, wording only), L1-16 (923623c), L1-21 (41297fb) and L1-26 (c2cddb7) | 33 |
| Fix rejected twice by the fix-skeptic, then finished by hand (see below); no fix-skeptic re-run | L1-3 (0aa9659, 66c7ec3, 2aa5a21), L1-4 (9c1784d, a4ddc8f), L1-10 (041cc34, 8e45aa0) | 3 |
| Decided, deliberately not changed | L1-51 (owner kept 64 KiB; outside the L1h batch) | 1 |
| Held for the owner | none | 0 |
| Recorded, not applied | L1-32, 33, 35, 38 | 4 |
| Refuted | L1-13, 30, 41, 44, 47 | 5 |
| Decided in round 1 | L1-17, 34, 36, 42 | 4 |

The pass L1h fix-skeptics (reported): 9 fixes attacked in this pass, 6 FIXED-VERIFIED (L1-5, 11, 12, 16, 21, 26: 39 lens verdicts,
all pass, 7 lenses for L1-5, 11 and 26 and 6 for the others) and 3 rejected (L1-3, L1-4, L1-10; the objection of each is
under its item). No new skeptic votes on the findings. The L1-12 optional refusal is not implemented and stays an open optional
follow-up (see the L1-12 item and [stage2-owner-decisions.md](../decisions/stage2-owner-decisions.md)). **Finished by hand, 2026-11-03:** L1-3, L1-4 and L1-10 were each rejected a second time by their fix-skeptic, each on a documentation residual (a changelog entry, a design-prompt sentence and a runtime-message note, an "one record" claim). The recorder cherry-picked the last attempt of each onto `main` and closed exactly the named residual, as for A3-10; the commits are the ones listed in the table. Each is docs and tests only (no non-test code line changed; the mutation check of the new L1-3 tests is recorded in stage2-owner-decisions.md). No fix-skeptic ran on these three final states: their gate result and the recorder's own checks are the only verification, and the record says so in the item sections. Coverage of the fix-skeptics: see the
coverage statement for pass L1h in that file; in short, `staticcheck`, `deadcode` and the fuzz step were not run by the fix-skeptics
of L1-5, 11, 12 and 26, and the Linux x86-64 box was not used by those of L1-5, 11, 12 and 26. The three sections of this file that
list items "held" and "rejected" below are the record as written on 2026-11-02; each item carries its updated disposition.

What the pass found, in one paragraph. No data-loss or durability defect in the library was reported. Two behaviour bugs
(L1-14: `FlushFailing` stays true on a failed log; L1-15: an empty `Hole{1,0}` from `Open`), one error-class bug
(L1-1: an oversize or non-regular state file was not `ErrCorrupt`), all fixed with regression tests. One real
inconsistency held for the owner (L1-11: `quill check` reports a log healthy when its oldest sealed segment is empty).
One scheduling-dependent efficiency defect (L1-43: group commit shares almost no fsyncs at GOMAXPROCS 1), fixed with a
one-line yield. Five measured optimisations landed (parallel `Check`, pooled verify buffer, 4 KiB first-frame probe, run
copying in the CLI printer, 64 KiB CLI output buffer). The rest is documentation drift, now corrected, and test gaps,
now covered. The suite, the race detector, the fuzz targets and the pinned static tools were clean on both platforms.

## Coverage

Auditors: six returned, none failed to return. Their scope and what they found clean is in the sections that follow;
the bounds are here.

- **Platforms.** Operational and most reproduction runs were on macOS (Darwin 25.6, Apple M2 Pro, APFS, Go 1.27.3,
  `GOWORK=off`). One Linux x86-64 box (Debian 13, ext4, 4 vCPU, shared, benchmarks under a lock) ran the full suite,
  the benchmarks, the fuzz targets and the cross-checks of several fixes. The Apple container runtime was not used by
  any auditor. The comment auditors ran almost entirely on macOS: one made a single Linux call, to compare `statfs`
  `f_bsize` with `f_frsize`; the others made none.
- **Tree.** All auditors ran against 5a70379, before the 27 fixes. Test counts, coverage and benchmark figures below
  are for that tree, not for the tip. Each fix-skeptic ran the race suite on its own post-fix tree and reported it
  passing; those totals were not recorded.
- **Read.** Every comment block in every library and `cmd/quill` file, in `internal/*`, `hack/` and `examples/` was
  read by one of the three comment auditors. The dead-code auditor read in full `log.go`, `flush.go`, `read.go`,
  `retention.go`, `consumer.go`, `state.go`, `stat.go`, `archive.go`, `seek.go`, `holes.go`, `codec.go`, `errors.go`,
  `lock_unix.go`, `disk.go`, `internal/atomicfile`, `internal/strictjson`, `cmd/quill/verbs.go` and `write.go`; for
  `load.go` (lines 370-430 only), `iter.go`, `reader.go`, `append.go` beyond line 130, `cmd/quill/read.go`,
  `consumers.go` and `print.go`, `hack/powertest`, `hack/skewopen`, the examples, `internal/fsx/os.go` and the
  simulator it relied on `deadcode`, the field and parameter scans and the merged coverage profile.
- **Not run.** `hack/skew.sh` and `hack/powertest` were read, not executed (powertest needs an OS crash). No test
  of ENOSPC or EIO on a real disk beyond the opt-in full-disk test. No cold-disk drain (a backlog larger than memory),
  no rotational-disk run, no Linux CLI timings, no log of tens of thousands of small age-rotated segments.
- **Not examined (leads, not findings).** `disk.go` multiplies `Bavail` by `Bsize`; on ext4 and tmpfs `f_bsize` equals
  `f_frsize` (4096, checked on the Linux box), so a unit error on a filesystem where they differ is unexamined. In
  the simulator, `fsx`'s package-level `nextID` is incremented under per-`Sim` mutexes, so two simulators in parallel
  tests would race, and `Sim.Inject` is documented as "set only while no goroutine uses the simulator" although some
  tests set it while the log's flusher is live; neither was run under `-race`. `Close`'s second publish
  (`err == nil && l.pendingPublish`, log.go:620-622 in the audited tree, coverage count 0) looks unreachable by the
  dead-code auditor's reading of `publishLocked` and `flushLocked`; not proven, no reproduction, not filed.
- **Observed, not filed.** `Records(ctx, 0)` and `Read(0, n)` start at sequence 1 and so yield a `GapError` over an
  expiry-trimmed prefix (auditor test: first=261, "records 1 to 260 are unavailable"), while `Scan(0)` and `ScanAll`
  start at `First`. This agrees with read.go (`max(offset, 1)`) and decision 58's text, but the iterator docs do not
  mention it. A sharp edge for the owner to know about; no decision is asked.
- **Observed, recorder (verified by reading docs/design/scope.md:238-239 at the tip).** scope.md still says of `Sync`
  that "the log's lock is not held through" the fsync. The flusher does hold it (L1-42, decided in round 1 as
  A1-30 and not changed). The sentence is true of the producers' wait and untrue of the flusher; see L1-42.
- **Internal names.** A grep for internal product, repository and path names over every file in the comment auditors'
  scope found no hit (reported by auditor L1-comments-2). This record was written without them.

## Comment versus code, per file

About 600 distinct comment blocks and the claims inside them were checked (the sum of the three auditors' own tallies,
approximate, by reading, not by a tool). "Clean" means each claim was traced to the code and found accurate. Findings are the items below.

| File | Blocks (approx.) | Result | Findings |
|---|---|---|---|
| `doc.go` | 8 claims | One claim over-promises: Err() and a missing segment | L1-2 (fixed) |
| `errors.go` | 19 | ErrCorrupt over-broad for state-file refusals; ErrFailed says "every" Sync | L1-1 (fixed), L1-4 (held) |
| `log.go` | 75 | Config fields, `validate`, `create()`, struct fields clean except two comments | L1-8 (fixed), L1-10 (fix rejected) |
| `flush.go` | 28 | Failure classes (a)(b)(c), retry of a failed publish, `lockLostLocked` (one fstat plus one lstat), `fail()`, rotate and publish comments clean; (d, e) comment wrong; `fail()` left `flushErr` set | L1-7 (fixed), L1-14 (fixed) |
| `append.go` | 14 | Doc silent on the sequence returned with an error | L1-5 (held) |
| `state.go` | 34 | `decodeTail` and its slow and fast paths, `decodeFormat`, `decodeConsumer`, `decodePolicy` tolerance clean; one comment over-general; oversize and non-regular files were not `ErrCorrupt` | L1-1, L1-9 (fixed) |
| `codec.go` | 22 | Arithmetic clean: `uvarintLen` against `binary.AppendUvarint` at 11 values; a maximum frame (Seq MaxUint64-1, At -1, 64 MiB-32 payload) is 67108864 bytes and round-trips; the "+34" batch bound is an upper bound; 479,349 one-byte records fit the default `MaxBatchBytes` | none |
| `disk.go`, `lock_unix.go` | 2, 8 | Clean (see the `statfs` lead above) | none |
| `load.go` | about 46 | `openSegmentFlag`, window chunking, `frameAt`, `verifySegment`, `recoverSegment`, `load()` ordering, `loadSealed` clean; `walk` comment wrong; empty hole at marker seq 0 | L1-18, L1-15 (fixed) |
| `holes.go` | 2 | Contiguity doc matches the code; both callers use it as stated | none |
| `read.go` | about 28 | `Log.Read`, `ReadWait`, `Scan`, `locate`, the expect contract, `sealedRefsLocked` clean; the deferral of errors after records is under-described; `GapError.From` for scans | L1-28 (fixed), L1-3 (fix rejected) |
| `reader.go` | about 45 | `ReaderOptions` defaults, `OpenReader` errors, `readTail`, `segment()` cache rules, `view()` clean; one skip comment incomplete; a write-only struct field; the first-frame probe size | L1-29 (fixed), L1-32 (not applied), L1-50 (fixed) |
| `retention.go` | about 38 | Header, clock guard, protection rules, cap loops, `deleteLocked` clean; a Prune-clears-error claim | L1-13 (refuted), L1-22 (fixed) |
| `archive.go` | 12 | Refusal order, error identities, file modes (Perm 0660 under umask 022 gives 0660 segments, 0750 `consumers/`), overlap check clean; a new destination gets only the suffix | L1-12 (held), L1-22d (fixed) |
| `consumer.go` | about 37 | `Protect`, `protect()`, `Commit` range, `Rewind`, `Close`, `list()`, `remove()` clean; `ReadWait` "handle closes"; a duplicated field | L1-16 (held), L1-38 (not applied) |
| `stat.go` | about 32 | Problem kinds, fatal against warning, `Stats.count`, `LastFlushOK` rules clean; `Check` skips an empty oldest sealed file; `BufferedBytes` comment | L1-11 (held), L1-22 (fixed) |
| `seek.go`, `iter.go` | 6, 7 | `OffsetForTime` semantics, whole-second granularity, iterator batch-then-error rules clean; the cost claim is for a Log or warm Reader | L1-21 (fix rejected) |
| `internal/atomicfile` | 8 | Step order, temp-file removal, `tempName` clean | none |
| `internal/fsx` | 18, 8, 22, 8 | `osFlags`, `O_NONBLOCK` rationale, `Rename` EINTR retry, crash-model comments clean; `Remove` contract false for the real implementation | L1-6 (fixed) |
| `internal/strictjson` | 14 | Escape, surrogate and case-folded duplicate-key logic clean, including the `\\u` skip | L1-36 (decided in round 1) |
| `cmd/quill` | about 10 in `print.go`; the rest of the files | `appendQuoted` equals `strconv.Quote` byte for byte and `appendJSONString` equals `encoding/json` except DEL and C1 as decision 35 says, over every rune 0..0x10FFFF and 200,000 random strings; `lossOnly` prefixes (8) each match a live library message; exit codes and the lock column match decision 53; `parseDecimal` refusals run; `quill reference` is byte-identical to `docs/reference.md`; the "about 479,000 one-byte lines" is exact (479,349 accepted, 479,350 refused) | L1-24, 25, 26, 27, 30, 33 |
| `examples/` | about 25 | `log`, `topics` and the pipeline reader and writer were run; output matches the comments; README claims about temp-directory clean-up hold | L1-31 (fixed) |
| `hack/` | about 29 | `gate.sh` derives the toolchain from `go.mod`; staticcheck v0.8.2 and deadcode v0.52.0 equal the pins in `ci.yml`; `skew.sh` and `skewopen` headers match the code (read, not run); `powertest` flow matches | L1-19, 20, 23 (fixed) |

Also checked clean, by reading, for the library docs: `doc.go`'s remaining bullets (single writer plus `ErrLocked`
after the 200 ms retry, durability visibility, protected consumers, caps beating protection, `ErrCorrupt` for the
newest segment and a marker above the last frame) match load.go, the user guide and the decisions; format.md says
`quill.tail` accepts case variants of keys, whitespace and reordering, so `decodeTail`'s "strictly" is consistent.

## Dead code and debt

All verified on a `git archive` copy of 5a70379, macOS, Go 1.27.3 unless noted.

- **Pinned gate tools clean.** `go vet ./...` (no output), staticcheck v0.8.2 (no output), `deadcode -test ./...`
  v0.52.0 (empty report). The pins in `hack/gate.sh` and `.github/workflows/ci.yml` agree.
- **`deadcode` without `-test`.** The only unreachable functions are the crash simulator (`internal/fsx/crash.go`,
  `crash_file.go`: `Sim`, `simFile`, `fileInfo`, `ShortWrite.Error`, `NewSim` and so on), which exist for tests by design
  (decisions 52, 61, 74). Every other non-test function in the root package, `cmd/quill`, `internal/atomicfile` and
  `internal/strictjson` is reachable from a `main`. Its operation constants and hooks all have test users.
- **Write-only fields.** A `go/types` scan (writes through selectors, keyed literals, `++` and op-assign counted apart
  from reads; a first version double-counted and was fixed and re-run) found `scanResult.frames`, `scanResult.lastAt`
  and `Log.fsyncs` as the only non-test hits. The exported `Stats` and `Problem` fields are public output read by the
  CLI; `jsonHole.From`/`To` are JSON-tagged. No `Config`, `ReaderOptions` or `ConsumerOptions` field is unread. Items
  L1-32 and L1-34.
- **Unused parameters** in library code: none. `cmd/quill`'s `args []string` on verb handlers is the uniform signature
  and `dispatch` rejects stray positionals. Functions that always return nil error: none outside tests, `hack` and examples.
- **Flags.** Every flag field in the CLI struct is read. All 8 `lossOnly` prefixes match a live `logf` message.
- **Constants and sentinels.** Every exported `Err*`, `GapError`, the codec constants and the `Default*` limits are
  referenced from non-test code. `MaxPayloadBytes` is a thin exported alias of the unexported constant; left alone.
- **TODO, FIXME, XXX, HACK** in source and tests: none (only `mktemp`'s `XXXXXX` template in `hack/skew.sh`).
- **Skips.** Three on Linux and macOS: `TestAFullDiskFailsAppendsKeepsRecordsIntactAndRecoversWhenSpaceReturns` (needs
  `QUILL_FULLDISK_DIR`; CI runs it on a tmpfs at ci.yml:81), `TestRecordGoldens` (record mode only, asserts nothing by
  design), and the `ends_early` subtest of `TestAHelperIsKilledAndReapedWhenTheTestEndsBeforeItKillsIt` (`t.SkipNow`
  is the mechanism). Root-user, FIFO, sparse-file, `chflags` and tool-missing skips did not fire on either machine;
  the four build-script tests ran and passed on both, so the `shasum` skip reason is not stale.
- **Vacuous tests.** Scanning every `Test` and `Fuzz` function for the absence of `Fatal`, `Error` or `Fail` gave ten
  hits: `TestMain` (two), a panic-only fuzz target (`FuzzUnmarshalNeverPanics`, valid by design), five tests that
  assert inside helpers, and `TestOpenAheadWarningAbsentWithoutALogger` (L1-41, refuted: it pins the nil-Logger
  guard; with the guard removed it panics).
- **Duplicated fixtures.** A normalised-body comparison of every function of five or more lines found no identical
  pair. Near-duplicates (the `ReadDir`-style helpers) were not looked for.
- **Uncovered blocks** are almost entirely error-return plumbing in `archive.go`, `consumer.go`, `stat.go`, `seek.go`
  and `retention.go`; not treated as dead. The live uncovered guards are L1-39 and L1-40 (both fixed with tests).

## Operational validation

### macOS (Darwin 25.6, M2 Pro, go1.27.3), HEAD 5a70379, a `git archive` copy

- **Suite.** `GOWORK=off go test -race -count=1 -timeout 15m -v ./...`: exit 0, 44 s. 467 `RUN` lines: 350 top-level tests
  PASS, 2 top-level SKIP, 0 FAIL; counting subtests 464 PASS, 3 SKIP (verified, counted from the output).
- **Race, repeated.** Two more full runs with `-count=3 -v ./...`: both exit 0, about 2 minutes for the root package
  each, 1,392 PASS lines each (464 x 3), 0 FAIL, 9 SKIP, no `WARNING: DATA RACE`, no panic. With the first run that
  is 7 race-detector passes of the suite.
- **One CPU.** `go test -cpu 1 -count=2 ./...` failed only `TestManyProducersSyncingConcurrentlyShareFsyncs`
  (deterministic, 50 of 50 at `-cpu 1`, 0 of 50 at `-cpu 2`, 3, 4 and 10): L1-43, fixed.
- **Flakiness.** No other flaky test found; the timing-sensitive files (`sync_test.go`, `reader_test.go`,
  `iter_test.go`, `append_test.go`, `wait_failed_test.go`) passed in all 7 race runs and both `-cpu 1` runs. Not every
  test was run 50 times.
- **Real `statfs`.** `freeBytes` agreed with `df` and `MinFreeBytes` enforcement engaged after the first pass (run by hand).

### Linux (Debian 13, ext4, 4 vCPU), HEAD 5a70379, Go 1.27.3

- **Suite.** `go test -race -count=1 -timeout 20m -v ./...` with `QUILL_FULLDISK_DIR` pointing at a small tmpfs: all
  packages ok. 466 `RUN` lines, 464 PASS, 2 SKIP, 0 FAIL. The two skips are `TestRecordGoldens` and the `ends_early`
  subtest. The full-disk test ran and passed (backlog_test.go:86-105). No tool-missing, root or FIFO skip fired; the user
  is non-root. (macOS 467 and 3 against Linux 466 and 2 differ by exactly the full-disk test, skipped on macOS.)
- **Race, repeated.** `-race -count=5 -v ./...`: exit 0, 2,320 PASS (464 x 5), 10 SKIP, 0 FAIL, no race report. A plain
  non-race `-count=5` was not run.
- **GOMAXPROCS.** 1 and 8 under `-race`, count 1, full-disk dir set: both exit 0, 464 PASS, 2 SKIP. Whether the setting
  reaches re-exec'd helper subprocesses was not checked.

### Coverage (macOS, statement coverage)

| Package | Coverage |
|---|---|
| `quill` (root) | 94.2% |
| `cmd/quill` | 93.2% |
| `internal/atomicfile` | 96.6% |
| `internal/strictjson` | 94.5% |
| `internal/fsx` | 88.6% |
| `examples/pipeline` | 72.2% |
| `examples/log`, `examples/topics`, `hack/powertest`, `hack/skewopen` | no tests (0%) |
| Merged `-coverpkg=./...` | 89.4% (the root package alone is 71% of statements) |

Functions at 0%: `Consumer.Name`, `Consumer.Scan`, `Reader.ID`, `Reader.Scan`, `freeBytes`, `fsx` `CrashError.Error`, the
`cmd` handler `WithAttrs` and `WithGroup`, `main`. The first five are L1-39 (fixed). `GapError.Error` and
`fsx.SameFile` are 0% in the root package alone and 100% merged. Lowest non-zero: `examples/pipeline` reader 64.3%,
`fsx` `now` and `Chmod` 66.7%, `frameHeader` 71.4% (L1-35), `fsx` `Durable` and `SyncDir` 71.4%, `reason` 75%, `wait` 75%,
`Reader.MarkerAge` 75%, `newLogID` 75%, `cmdRoll` 75%.

### Benchmarks against the recorded baselines

The rule (docs/audits/reference/benchmark-baselines.md, decisions 21 and 70): a regression is a median more than 10% worse with
ranges that do not overlap. Six runs each (`-benchmem -count=6`). The recorded tables are for v0.1.0 (2026-10-29).

**macOS.** The machine was shared and loaded during the run (load average reached 20 early), so the ranges are wider
than the baseline's. Medians against the recorded table, as the auditor reported them (the ledger gives a range only
where one is shown):

| Benchmark | Change in median | Range now | Recorded range | Reading |
|---|---|---|---|---|
| AppendBuffered | +64% | max 62 us | 21.9-22.8 us | load; ranges overlap in the auditor's read |
| AppendFlushEach | +47% | max 16.8 ms | 9.6-10.2 ms | load; reproduces at v0.1.0 speed when quiet: 9.74 ms v0.1.0, 10.0 ms HEAD (same-machine A/B) |
| AppendBatchSync | +37% | max 19.3 ms | 10.2-10.6 ms | load |
| ReadStart | +5.5% (25.9 us) | n/r | 24.4-25.5 us | within 10% |
| ReadTail | +9.4% (22.6 us) | n/r | 20.3-21.0 us | within 10% |
| ReaderReadActive | +8.7% (2.137 ms) | n/r | 1.92-2.13 ms | within 10% |
| ReadSealed | +3.1% (37.3 us) | n/r | 35.6-36.7 us | within 10% |
| ScanAll | +9.0% (20.59 ms) | n/r | 18.6-19.5 ms | within 10% |
| ReadWait | -3.8% (9.81 ms) | n/r | 10.0-10.4 ms | within noise |
| ReaderFollow | +4.6% (10.9 ms) | n/r | 9.89-10.8 ms | within 10% |
| DrainBacklog | +7.8% (121.7 ms) | 107.0-123.5 ms | 110.6-114.6 ms | within 10% |
| ReadStartThousandSegments | +4.1% (2.34 us) | n/r | 2.23-2.31 us | within 10% |
| ColdStart | +10.6% | 23.5-55.8 ms | 22.5-22.9 ms | literal rule met; A/B against v0.1.0 same machine: 22.45 against 22.32 ms, not a regression |
| ParallelLogsFlush | +10.2% | 26.2-28.3 ms | 20.8-26.0 ms | literal rule met; A/B at 30x: 23.8 against 24.1 ms, not a regression |
| ReaderReadStartThousandSegments | +12.9% | 181-187 us | 162-165 us | literal rule met; A/B (6 interleaved runs each): v0.1.0 median 167.7 us (163.3-175.5), HEAD 180.2 us (176.0-183.7), +7.5%, below the rule; B/op grew 346 KB to 411 KB gradually over several commits |

"n/r" is not reported in the ledger. Allocations match the baseline except in the cases of item L1-44 (flush 10
against 6, `ReaderFollow` about 244 against 162, `ParallelLogsFlush` 51 against 35, `ColdStart` 339 against 327,
`ReadWait` 20 against 16); ReadStart, ReadTail and ReadSealed 113, 106 and 113, `ReaderReadActive` 72, `ScanAll` 47,
`DrainBacklog` 89 against 88, the thousand-segment pair 30 and 2,061 are unchanged. The Linux table of the baseline was
not compared on macOS. Three of the macOS entries that meet the literal rule are therefore noise or sub-threshold on
the same-machine A/B; none is a confirmed regression.

**Linux** (ext4, flock, `TMPDIR` on the disk). New median (range) against the recorded median (range); the "change" is
the recorder's arithmetic from those medians.

| Benchmark | Now, ns/op | Recorded, ns/op | Change |
|---|---|---|---|
| AppendBuffered | 3,203 (2,868-3,417) | 2,944 (2,810-3,077) | +8.8% |
| AppendFlushEach | 960,166 (932,315-1,033,127) | 994,156 (950,052-1,044,095) | -3.4% |
| AppendBatchSync | 1,351,252 | 1,350,909 | 0.0% |
| ReadStart | 43,887 (41,816-46,649) | 41,656 (38,466-48,065) | +5.4% |
| ReadTail | 69,356 (67,665-72,544) | 68,017 (63,754-127,662) | +2.0% |
| ReaderReadActive | 7,267,782 (7,005,029-7,350,368) | 6,798,884 (6,630,854-7,099,823) | +6.9% |
| ReadSealed | 84,734 (76,170-93,363) | 76,863 (73,147-132,146) | +10.2%, ranges overlap |
| ReadWait | 1,065,722 (1,011,792-3,657,445) | 1,031,078 (1,024,260-1,082,676) | +3.4% |
| ParallelLogsFlush | 1,402,422 (1,325,469-1,439,157) | 1,336,699 (1,275,376-1,404,999) | +4.9% |
| ReaderFollow | 1,568,207 (1,429,750-2,636,204) | 1,407,812 (1,387,655-1,475,206) | +11.4%, ranges overlap |
| DrainBacklog | 382,994,350 (364,696,483-410,315,259) | 355,361,852 (348,752,705-369,701,060) | +7.8%, ranges overlap |
| ReadStartThousandSegments | 5,764 (5,326-6,253) | 5,742 (5,498-6,526) | +0.4% |
| ReaderReadStartThousandSegments | 440,366 (426,230-488,723) | 416,622 (407,842-451,619) | +5.7% |
| ColdStart | 31.7 ms (30.26-32.73) | 28.06 ms (26.58-28.64) | +13.1%, ranges do not overlap; same-box A/B against v0.1.0: +7.9%, so mostly box drift |
| ScanAll | 67.8 ms | 59.85 ms | +13.2%, ranges overlap; A/B against v0.1.0: +9.8% (61.7 ms to 67.8 ms) |

Only `ColdStart` against the recorded figure meets the literal rule, and the same-box A/B shows +7.9% with overlapping
ranges (L1-47, refuted). Note on the rule: it is defined on the macOS table, and the macOS machine was not re-run for
the full ns/op suite on the Linux side. The four `BenchmarkPrint*` benchmarks in `cmd/quill` have no recorded baseline;
0 allocs, 139-539 ns/op.

### Fuzz

Seven targets, matching `go test -list '^Fuzz'` over every package (`hack/fuzz.sh`'s enumeration). Three minutes each,
one at a time, no crasher, no "Failing input", all exit 0, no new `testdata/fuzz` files.

| Target | macOS execs (new interesting, corpus) | Linux execs |
|---|---|---|
| `FuzzDecodeFrame` | 28.6 M (0, 17) | 6.77 M |
| `FuzzRecoverSegment` | 27.5 M (0, 38) | 6.52 M |
| `FuzzDecodeTail` | 5.5 M (+173, 728) | 2.27 M |
| `FuzzDecodeFormat` | 30.9 M (+7, 151) | 7.07 M |
| `FuzzDecodeConsumer` | 24.4 M (0, 12) | 6.20 M |
| `FuzzDecodePolicy` | 23.4 M (+11, 820) | 6.20 M |
| `strictjson` `FuzzUnmarshalNeverPanics` | 8.0 M (+196, 793) | 2.07 M |

Caveat (reported by the macOS auditor): baseline corpus sizes include entries from the shared build cache's fuzz cache
(earlier runs by other agents), so "new interesting" understates total growth. `FuzzDecodeFrame`,
`FuzzRecoverSegment` and `FuzzDecodeConsumer` found nothing new either because they are saturated or because their
seeds and mutators are narrow; not determined. One three-minute pass per machine is bounded coverage.

## Optimisation

Measured on macOS 25.6 (M2 Pro, APFS, `F_FULLFSYNC`) and the Linux box, Go 1.27.3, warm page cache. The Linux box has
no `strace` or `perf` and no root for `dtruss`, so syscall counts were taken at the `fsx` seam with a counting
filesystem plus a read of Go's `os/file_unix.go`. Fixtures: 1.5 M records, 224 MB, 30 segments; 400 K records in 787
segments of 64 KiB; 60 K records in 2,145 segments of about 4 KB; 600 K text records.

### Optimised (fixed in this pass; figures are the auditor's, re-measured by the fix-skeptic where noted)

| Item | Change | Measurement |
|---|---|---|
| L1-43 | The flusher yields once (`runtime.Gosched`) after a Sync request, before taking the lock | Fsyncs for 160 syncs at one CPU: 160 before, 12-13 after (sim); real disk macOS 129-140 before, 12-13 after; Linux 11-13 after. Fix-skeptic: unchanged at `-cpu 4` and 10 |
| L1-48 | `Reader.Check` verifies segments in parallel, merging in id order | macOS big (30 x 8 MiB) 70.7-72.5 ms to 12.2-12.5 ms; many (787 x 64 KiB) 34.4-35.4 to 9.5-9.8 ms; tiny (2,145 x 4 KB) 35.5-37.3 to 25.8-26.2 ms. Linux big about 219 to about 62 ms; many 75-83 to 19-21 ms. Output identical: 40 random logs in the auditor's differential test, 3,000 in the fix-skeptic's. Cold (page cache dropped) Linux: 2.7x to 3.7x faster. Rotational disk not measured |
| L1-49 | `verifySegment` and `verifyBounded` read through a pooled buffer | `Open` of 787 x 64 KiB: 32.7-33.7 to 20.2-21.1 ms macOS, 46.6-55.4 to 22.8-24.9 ms Linux; B/op 59.1 MB to 2.1 MB; at 8 MiB segments within noise on macOS, 8-13% on Linux |
| L1-50 | The sealed-segment first-frame probe reads 4 KiB, not 64 KiB | `Reader.Stat` cold, 787 x 64 KiB: 20.2-20.8 to 13.4-13.9 ms macOS (-34%), Linux -45% reported, about -35% in the fix-skeptic's smaller fixture; CLI `tail -n 10` 25.1-27.9 to 18.3-20.6 ms. Segments under 4 KiB unchanged; a first frame over about 4 KiB takes a second read, no regression seen at 20 KB or 600 KB frames |
| L1-52 | The CLI text printers copy runs of plain bytes via a lookup table | `PrintJSONText` 337-346 to 99-101 ns (3.4x), `PrintTextText` 155-158 to 81-82 ns (1.9x); `quill cat` of 600 K records 214-218 to 156-161 ms; output byte-identical (3 M random strings, every 2-byte string and all strings to length 5 over a hostile alphabet). Linux JSON text about 2x, text form noisy |
| L1-53 | The CLI output buffer is 64 KiB, not 4 KiB | macOS pipe 155-167 to 142-147 ms, file sys time 0.09 to 0.03-0.04 s; the claimed 29% wall-time gain to a file was not reproduced to the digit (0.16-0.19 to 0.11-0.12 s in the fix-skeptic's run); Linux 7-10% with overlapping ranges, none to /dev/null |

### Held for the owner

- **L1-51.** `indexStride` 64 KiB to 16 KiB makes a 100-record read at a random sealed position about 20-27% faster
  (macOS Sealed 37.3-38.3 to 29.0-29.8 us; Linux 74.3-82.8 to 52.6-59.2 us; Head unchanged), at 4x the resident index
  (about 0.15% of the log's size, about 1.5 GB per Log per TiB of retained log). Tuning trade-off; see the owner file.

### Measured and refused, with the reason

| Candidate | Measurement | Why not |
|---|---|---|
| Coalesce `time.Now` per `Append` (about 18% of a 210-245 ns buffered append) | verified | Changes `At` accuracy and the injectable clock: a semantic change |
| Remove a fsync, or the marker's temp file | the flush is 2 fsyncs and 9 other calls: `fstat` and `lstat` of the lock, `pwrite`, `fsync`, marker create, `fchmod`, `pwrite`, `fsync`, `close`, `rename` | Nothing redundant; the marker design is decisions 8 and 40. `AppendFlushEach` is 9.8 ms on macOS (two `F_FULLFSYNC`) and about 1 ms on Linux; the on-CPU profile shows `rename` 0.65 ms and the exclusive create 0.3 ms per flush on APFS: platform-bound |
| `Readdirnames` plus sort in place of `os.ReadDir` | 355-374 us against 369-399 us on a 787-entry directory: ranges overlap | No gain; `ReadDir` is 70-85% of `Reader.Read(1,100)` there: platform-bound |
| A different window chunk size | `windowChunk` 2^20, 2^18, 2^16 gave the same `ScanAll`, `Open` and `Check` within noise (ScanAll 83.6-86.7, 81.5-85.9, 85.4-88.5 ms) | Not a lever: 62% `pread` copy plus 20% CRC |
| One-stat "has the marker changed" check in the Reader poll | One tick is 8 file operations, about 23.7 us and 13 allocs: 0.05% of a core at 50 ms, 2.4% at 1 ms | Unsafe: inode numbers can be reused across the create-and-rename publishes, so a poll could miss two publishes in a row |
| Drop `O_NONBLOCK` (netpoller registration) | open/fstat/pread/close 12.5-13.2 us with and without it on macOS | No cost measured; on Linux an `epoll_ctl` that fails with `EPERM` is inferred, not measured |
| Remove the `Lstat` before each segment open | 3-5% on `Log.Read` (23.5 to 22.4 us Head, 37.5 to 36.3 us Sealed), about 5% on cold `Stat` | Below what is worth proposing, and it weakens the guard against a device node or FIFO under a segment name |
| A per-batch payload slab for `Reader.Records` and `Follow` | 600 K text records: `Records` 55.0-56.2 ms and 600 K allocs against `Scan` 26.3-27.2 ms and 109 allocs; the copy is about half of an iteration | A retained record would pin a whole batch: a memory-retention change (decision 58); measured, not prototyped |
| A goroutine pool prewarming first frames in `Reader.view` | A further 12-16% on top of L1-50 (`StatManyCold` 13.4 to 11.8 ms, `StatTinyCold` 38.9 to 32.5 ms) | Concurrency in the view path for a modest gain |
| Faster `Open` of a large log | `Open` is 25 ms for 224 MB on macOS and 77 ms on Linux | Every sealed segment is verified whole by design (decision 39); not challenged |
| `Reader.Read` near the head of the active segment (2.0-2.3 ms) | 85% `pread` memcpy | Documented by design (decision 63, benchmark-baselines.md:30); `Records` and `Follow` avoid it |
| Cold-disk drain read-ahead | not measured | Needs a backlog larger than RAM on a shared box; listed as a candidate at benchmark-baselines.md:126 (reported) |

## Items

Each item: claim, grounds, warrant, backing, rebuttal, verdict (with votes), then disposition. Anchors are to 5a70379.
"FV" is fix-skeptic FIXED-VERIFIED.

### Fixed (27)

**L1-1 (medium, CONFIRMED, 2 of 2, both reproduced).**
*Claim.* A `quill.tail` or `quill.format` over 1 MiB or not a regular file makes `Open` and `OpenReader` fail with
the unexported `errBadState`, which matches no exported sentinel, not `ErrCorrupt`; errors.go, `loadFormat`'s
comment, the user guide and format.md say damage to these files is `ErrCorrupt`.
*Grounds.* state.go:368-373 returns `errBadState`; load.go:293-296 returns it raw (`ErrCorrupt` is added only on the
`decodeTail` error at load.go:297-299); state.go:236-243 wraps only `decodeFormat` errors; state.go:228,
errors.go:19-24, docs/format.md:77 ("A larger one is damage"), docs/userguide.md:377-379.
*Reproduction.* Scratch test in package `quill`: create a log, close, replace `quill.tail` or `quill.format` with a
2 MiB file or a directory, call `Open` and `OpenReader`, log `errors.Is(err, ErrCorrupt)`. All 8 combinations gave
"quill: malformed state file: <path>: 2097152 bytes" or "...: not a regular file" with `ErrCorrupt` false.
*Warrant.* The docs name this input class as damage and damage as `ErrCorrupt`; a caller that branches on
`ErrCorrupt` misclassifies it. *Backing.* ErrCorrupt doc; decision 6; format.md:77; guarantee 7; decision 74
("refused as damage"). *Rebuttal.* Fails if `ErrCorrupt` meant only "parsed and found invalid"; nothing says so. No CLI
code branches on `ErrCorrupt`, so only library callers see it. Skeptic doubt: `loadFormat`'s comment says other read
failures are returned "for the caller to classify", and `state_test.go:287` pins `errBadState`.
*Disposition.* Fixed in 5579010, FV (6 of 6). `readState` returns an error that wraps both `ErrCorrupt` and
`errBadState` for oversize and non-regular, and maps `ELOOP` from the `O_NOFOLLOW` open (a symlink) to `ErrCorrupt`
(the fix-skeptic's earlier symlink objection). All callers go through `readState`. Both new tests fail without
`state.go` (12 subtests) and pass on macOS and Linux. Open residuals: a damaged `quill.policy` read by `check` now
carries the "corrupt log:" prefix in the warning text (cosmetic, not pinned); the gate's fuzz step was not run.

**L1-2 (low, CONFIRMED, 2 of 2; one run).**
*Claim.* doc.go says a damaged or missing sealed segment becomes a hole that `Err()` names; for a segment deleted by
hand `Err()` is nil. *Grounds.* doc.go:20-23; `Err()`'s doc (log.go:648-654), errors.go:19-24, userguide.md:380-382
limit it to a segment that fails to verify (sticky `loadErr`).
*Reproduction.* A 4-segment log (seq 1-20) with segment 1 removed while closed, then reopened: `Open` succeeds,
`Err()` is nil, `Stat().Holes` is `[{6 10}]`, `Read(6,100)` returns "records 6 to 10 are unavailable". A skeptic's
12-segment variant printed the same.
*Warrant.* A missing file has no `*SegmentError` to hold in `loadErr`. *Backing.* Decision 38. *Rebuttal.* None
that holds; "names" cannot be read as `Stat().Holes` because the sentence is about `Err()`.
*Disposition.* Fixed in 34fcc65 (doc.go reworded and rewrapped, CHANGELOG line), FV (6 of 6). Fix-skeptic note: a
deleted oldest segment is not a hole at all (`First` moves, silent-degradation.md G2); the old text had the same
imprecision.

**L1-6 (low, CONFIRMED, 2 of 2; one reproduced).**
*Claim.* The `fsx.FS.Remove` contract "unlinks a file (never a directory)" is false for `fsx.OS`, whose `Remove` is
`os.Remove` and removes an empty directory; `Sim.Remove` refuses a directory. *Grounds.* fs.go:34-35; os.go:66;
crash.go:347-363.
*Reproduction.* `fsx.OS{}.Remove` on an empty directory returned nil and a later `Lstat` said "no such file".
*Warrant.* `os.Remove` removes empty directories, so the real implementation and its test double diverge.
*Backing.* Package doc, decisions 61 and 74. *Rebuttal.* Reachable impact tiny: callers pass names they built.
*Disposition.* Fixed in 1a2086c, FV (6 of 6): `OS.Remove` calls `unlink(2)` (EINTR retried, error wrapped as
`*fs.PathError`, so `errors.Is(err, fs.ErrNotExist)` at retention.go:341 still matches). `TestOSRemove` fails without
the change and passes on macOS and Linux (EISDIR there, EPERM on macOS).

**L1-7 (low, CONFIRMED, 2 of 2).** *Claim.* The comment "(d, e) Rotate if due and publish the marker; failures there are
retried next time" (flush.go:78) is wrong for (e): a directory-fsync failure is fatal. *Grounds.* flush.go:183-188,
218-222; decision 60. *Warrant.* The file's own lettering makes (e) the fatal class. *Backing.* Decision 60.
*Rebuttal.* "(d, e)" read as plain step names; rejected because flush.go:66 marks (b) "Fatal" in the same lettering.
No run (comment). *Disposition.* Fixed in 21f54b6, FV (6 of 6); comment only.

**L1-8 (low, CONFIRMED, 2 of 2).** *Claim.* The `create()` comment applies "when the log holds no segments and no
non-zero marker" to all three repairs; `consumers/` is created whenever missing. *Grounds.* log.go:503-505 against
log.go:547-552 (unconditional), 553-576, 577-596. *Warrant.* The condition binds only the segment and marker repairs.
*Backing.* Decisions 62, 81. *Rebuttal.* "when" may bind only the last two; strained. Tail creation is also gated
on no segment holding a byte, so the comment was loose twice. No run. *Disposition.* Fixed in f588adb, FV (6 of 6);
comment only, corrects the tail clause as well.

**L1-9 (low, CONFIRMED, 2 of 2).** *Claim.* `decodePolicy`'s comment and decision 29 say an off retention is written as
`-1ns`; `writePolicy` writes `c.Retention.String()`, so `-time.Hour` is `-1h0m0s`. *Grounds.* state.go:335-338,
log.go:775. *Warrant.* Only `-1ns` gives `-1ns`. *Backing.* Decision 29; format.md:219. *Rebuttal.* Descriptive only;
the empty-string check is what matters. *Disposition.* Fixed in 82461fc, FV (6 of 6); comment only. Residual: the same
wording survives in a test comment, CHANGELOG.md:94 and decision 29 (the claim's fix said to leave the decision).

**L1-14 (low, CONFIRMED, 2 of 2; one reproduced; bug).**
*Claim.* `Stats.FlushFailing` stays true on a log that has failed when a write failure preceded the fatal fsync
failure; stat.go:47-53 says it stays false once failed. *Grounds.* stat.go:99 (`l.flushErr != nil`), flush.go:60
(set), flush.go:76 (cleared only after a successful write and fsync), flush.go:125-132 (`fail()` never touches it).
*Reproduction.* On `fsx.Sim`: inject `ENOSPC` on write, `Flush`: `FlushFailing` true; inject `EIO` on sync, `Flush`:
"log failed; close and reopen: EIO"; `Stat().FlushFailing` is still true. Auditor test `TestX1FlushFailingAfterFailed`;
a skeptic reproduced with its own in-package test.
*Warrant.* `fail()` leaves `flushErr` set, so the documented "false once failed" holds only if the first failure is fatal.
*Backing.* The `FlushFailing` doc, scope.md alert guidance. *Rebuttal.* Low impact: `Err()` wraps `ErrFailed` and the
documented alert on `FlushFailing` still fires.
*Disposition.* Fixed in d9d4833, FV (7 of 7): `fail()` sets `l.flushErr = nil`. The regression test fails with the
change reverted ("FlushFailing is true on a failed log") and passes after. A sibling truncate-failure path also checked.

**L1-15 (low, CONFIRMED, 2 of 2, both reproduced; bug).**
*Claim.* `Log.load` can record an empty hole `Hole{From:1, To:0}` when every sealed segment is damaged, the newest is
empty and the marker says seq 0, against holes.go's and decision 102's "an empty hole is never reported"; the Reader
guards the same step with `tl.Seq > 0`. *Grounds.* load.go:392-394 (no guard); reader.go:500; holes.go:19-20.
*Reproduction.* Fresh log (marker seq 0), `0000000000.log` overwritten with garbage, an empty `0000000001.log` added.
`OpenReader` `Stat`: holes `[]`; `Open` `Stat`: holes `[{From:1 To:0}]`, head 0, records 0, `Err()` names the damaged
file (`TestT3EmptyHole`, reproduced by both skeptics). A skeptic noted this state is reachable without hand-crafting:
a failed first-rotation marker write (flush.go:173-179) then a crash leaves the marker at 0.
*Warrant.* `contiguity()` cannot produce it; the writer's tail step can. *Backing.* holes.go:12-20; decision 102.
*Rebuttal.* Needs a doubly-faulted layout; the visible effect is a malformed `Hole` in `Stat`, `stat` and `check`.
*Disposition.* Fixed in 932cadf, FV (7 of 7): `&& tl.Seq > 0` on the `leadBad` branch. CLI `quill recover` printed
"hole 1-0" before and nothing after; `check` still exits 3 and names the damaged segment.

**L1-18 (low, CONFIRMED, 2 of 2).** *Claim.* `walk`'s doc says a frame is accepted only when `fn` returns true and a
stopped scan returns the error at that point; the code counts the frame before `fn` runs and an `fn` stop returns nil.
*Grounds.* load.go:176-181, 203-210. *Warrant.* `fn` is called after acceptance. *Backing.* The call sites
(`scanSource`, `offsetForTime`, `verifyBounded`) rely on the nil-error stop. *Rebuttal.* Doc precision only. No run.
*Disposition.* Fixed in dfbc8ad, FV (6 of 6); comment only. Nit: the new text lists `frames, last, end` but `walk` also
updates `first` and `index`.

**L1-19 (low, CONFIRMED, 2 of 2).** *Claim.* The header of `hack/gate.sh` says the one check CI runs and the gate does
not is govulncheck; CI also runs cross-platform vet, the full-disk tmpfs test, the examples and the release build
script. *Grounds.* gate.sh:2-5; ci.yml:36-37, 80-92, 94-104. *Warrant.* A macOS committer never vets the Linux build
before pushing. *Backing.* gate.sh header, decision 70. *Rebuttal.* `go test ./...` covers the examples' tests and the
build-script test; full-disk is excluded on purpose by decision 70; cross-vet is the real omission. No run.
*Disposition.* Fixed in 389375c, FV (6 of 6): the gate now vets `GOOS=linux` and `GOOS=darwin`; header lists what CI
alone runs; README, CONTRIBUTING and audit-prompt updated. Demonstrated: a type error in a Linux-only file passes
native `go vet` and fails `GOOS=linux go vet`.

**L1-20 (low, CONFIRMED, 2 of 2, both reproduced; hygiene).** *Claim.* `hack/fuzz.sh` exits 0 and skips a package's
targets when that package fails to compile. *Grounds.* fuzz.sh:12 (`|| true` on `go test -list`; the claim cited :14);
gate.sh:8 "Fail-closed". *Reproduction.* A syntax error added to the root package: `FUZZTIME=1s bash hack/fuzz.sh`
printed the compile error 6 or 7 times, ran exactly one target (`strictjson`), and exited 0. *Warrant.* A compile error
leaves its listing empty, so zero targets are enumerated. *Backing.* gate.sh:8. *Rebuttal.* The gate's vet, build and test
run first and the aggregate CI job still fails, so the exposure is a green fuzz job in a red run; a renamed target is
also silently dropped. *Disposition.* Fixed in f67c7a1, FV (6 of 6): fails when a listing fails or when no target ran in
the whole module; `TestFuzzScriptFailsClosed` (four subtests) fails against the old script. Residual: a target renamed
away from `^Fuzz` in one package is not caught while another package still runs a target.

**L1-22 (low, CONFIRMED, 2 of 2; merged L1-37).** *Claim.* Several comments are stale: (a) `consumers()` "takes no lock"
while it takes `l.mu` (via `Head` and `Read`); (b) `consumerMtime` names the wrong function; (c) `Stats.BufferedBytes`
says the buffer grows only to `MaxBufferBytes` while flushes fail, but a single append into an empty buffer may
exceed it; (d) archive.go's public godoc cites a private audit id; (e) `Consumer.Records` and `Follow` read the
committed offset when called, not when ranged; plus load.go:16-17 ("hole detection" is in holes.go; "phase 2").
*Grounds.* retention.go:43, 72, 108-109; consumer.go:322; stat.go:41-44 with append.go:68; archive.go:28-29;
iter.go:41, 49. *Rebuttal.* (a) can be read as "no consumer lock"; (e) matters only if the iterator is kept. No run.
*Disposition.* Fixed in b59b387, FV (6 of 6); comments only. Nits: one new comment line is long; "always taken" in
stat.go does not mention rotation or disk-full refusals.

**L1-23 (low, CONFIRMED, 2 of 2; one reproduced).** *Claim.* `hack/powertest verify` claims every acknowledged record
survived and the log is a contiguous prefix, but never asserts the first scanned record is sequence 1; a log that
lost its oldest segment would pass. *Grounds.* hack/powertest/main.go:3-6, 107-120. *Reproduction.* Crash a powertest
log (`kill -9` after 15 s), copy it, `rm` the oldest segment file in the copy, `verify`: prints OK, exit 0, "recovered
6831 of 26075 records". *Warrant.* The scan starts at the first retained record; with `Retention -1` nothing is deleted,
so `First` must be 1. *Backing.* Guarantees 3 and 4 ("a crash loses a suffix, not a middle", scope.md:367).
*Rebuttal.* Needs a pure unlink of segment 0; only an OS-crash harness uses this check. *Disposition.* Fixed in
7d72dd4, FV (6 of 6): fails with "FAIL: log starts at N, not 1"; the new test runs the real `verify` as a child process
and fails without the change. Residual: docs/audits/reference/power-loss.md:16 does not list the new check.

**L1-24 (low, CONFIRMED, 2 of 2, both reproduced).** *Claim.* The `withReader` comment says a verb checks its usage
before calling it, so a bad flag still exits 2 when `-dir` cannot be opened; true only for some verbs. *Grounds.*
cmd/quill/read.go:20-21; consumers.go:108-111, 119-139, 167-186, 190; read.go:113-118, 147-154, 197-205.
*Reproduction.* Built CLI against a nonexistent `-dir`: `register`, `commit -offset 5`, `read -since junk`, `tail -since junk`
and `cat -since junk` exit 1 ("not a log"); `unregister` exits 2 ("-name is required"). *Warrant.* Four code paths contradict
a universal statement. *Backing.* Decision 53; `cmdCommit`'s "Parse before opening anything". *Rebuttal.* With an openable
`-dir` all of these still exit 2; the alternative (hoisting the checks) would change exit codes. *Disposition.* Fixed in
232d04f, FV (6 of 6): the comment names the verbs; no behaviour change.

**L1-25 (low, CONFIRMED, 2 of 2, both reproduced).** *Claim.* The shared `-from` usage text is wrong for `tail` (the default
is the last `-n` records and `-n` is ignored once `-from` or `-since` is given), `cat` is summarised as "every retained
record" but honours `-from`, `-since` and `-limit`, and `-since` silently beats `-from`. *Grounds.* verbs.go:70-78,
128-134; read.go:197-212. *Reproduction.* Built CLI on a 6-record log: `tail -n 2` printed 5 and 6; `tail -n 2 -from 1`
printed 1-6; `cat -from 4` printed 4 on; `read -from 3 -since @1` printed from 1. *Warrant.* The help is the only place a
user learns the flags. *Backing.* verbs.go:21 (the table drives help and `docs/reference.md`). *Rebuttal.* Decision 86
says `read` and `cat` share flags by design; the `cat` half is wording. *Disposition.* Fixed in a41feaf, FV (6 of 6):
`tail` has its own `-from` text, the `cat` summary and `-since` text say what they honour, `docs/reference.md`
regenerated; no behaviour change.

**L1-27 (low, CONFIRMED, 2 of 2).** *Claim.* `run`'s doc comment opens with `dispatch`'s description. *Grounds.*
verbs.go:145-156, 173-205. *Rebuttal.* Unexported, cosmetic. No run. *Disposition.* Fixed in b596844, FV (6 of 6): the
sentence moved onto `dispatch`.

**L1-28 (low, CONFIRMED, 2 of 2).** *Claim.* The `fail` closure comment and `scanSource` doc say only a gap met after
records were delivered is deferred to the next call; every error through `fail` is, including I/O errors and active-segment
`ErrCorrupt`. *Grounds.* read.go:179-187, 267, 270, 156-158. *Reproduction.* Reader over a log whose active segment
had a flipped frame byte: `Read(8,100)` returned 7 records and nil; `Read(15,100)` then returned "corrupt log: the active
segment below the durable offset: bad frame"; `Scan(8,0,fn)` delivered 7 then the error. *Warrant.* The comment names one
class, the code defers three. *Backing.* Decisions 38, 79, 84. *Rebuttal.* Behaviour is intended and safe; "every error
is deferred" slightly overstates (the "segment N ended at" error at read.go:275 is returned with the records).
*Disposition.* Fixed in d103957, FV (6 of 6): Read, Reader.Read and Consumer.Read docs, `scanSource` and the `fail` comment
name the deferred classes and the exception; no code change. The fix-skeptic ran a throwaway EIO test to confirm an I/O
error may not recur on the next call.

**L1-29 (low, CONFIRMED, 2 of 2, both reproduced).** *Claim.* The comment on `last < s.first` in `Reader.view` names a
stale marker as the only cause; the branch is also taken when the next good segment starts at or before this one's first
sequence (a duplicate or overlap), dropping the segment before `contiguity` can report it. *Grounds.* reader.go:407-433.
*Reproduction.* 5 segments, `0000000000.log` copied over `0000000001.log`: `quill check` reports the overlap (exit 3), `recover`
(Open) refuses, `cat` printed seq 1-10 then "records 11 to 20 are unavailable", `stat` printed no hole. A skeptic's variant
had `Read(1,100)` return records 1-10 with no error while `Check` reported the overlap and the hole. *Warrant.* A second
cause, a different outcome. *Backing.* Decision 102, decision 38. *Rebuttal.* Needs operator damage; safe (no wrong
record served). Whether the Reader should report the overlap is a separate, semantic question. *Disposition.* Fixed in
f155feb, FV (6 of 6); comment only.

**L1-31 (low, PLAUSIBLE; votes PLAUSIBLE, REFUTED, PLAUSIBLE).** *Claim.* examples/topics/main.go says topics "share
nothing, so one slow fsync never delays another", true of process-level state but not of logs on one filesystem.
*Grounds.* examples/topics/main.go:1-3; examples/README.md:9; all three logs under one temp directory. Inferred from
how fsync behaves on a shared journal; not measured. *Warrant.* "Never" is a guarantee the library cannot give below
itself. *Backing.* scope.md, decision 4. *Rebuttal.* Probably meant at the library level, where it is true; scope.md:25
scopes the guarantee to locks and scope.md:262, 286 say logs on one disk share the disk. *Disposition.* Fixed in 1bcdd67,
FV (6 of 6), on a second attempt: all four sites (example, examples README, userguide, scope.md) now say what is not
shared (lock, buffer, flush goroutine) and that the device is. Left: build-prompt.md:305 (an in-process library test).

**L1-39 (medium, CONFIRMED, 2 of 2; merged L1-45).** *Claim.* `Consumer.Scan`, `Reader.Scan`, `Consumer.Name` and
`Reader.ID` have no caller in the repository; `freeBytes` runs only in the opt-in full-disk test; the deadcode gate cannot
see this. *Grounds.* consumer.go:181, 258-260; reader.go:126, 179-181; disk.go:7; all 0.0% in a merged `-coverpkg=./...`
profile; `deadcode -whylive` answers "function not found in program". *Reproduction.* A skeptic broke each wrapper in a
scratch copy; the root package, `cmd/quill` and `examples/pipeline` still passed. *Warrant.* A one-line wrapper with
swapped arguments passes vet, staticcheck, deadcode and every test. *Backing.* userguide.md:165 lists `Scan` on `Reader`.
*Rebuttal.* Trivially correct today; the shared `scanFrom` is covered via `Log.Scan`. *Disposition.* Fixed in 7497a80, FV
(6 of 6), tests only: eight mutations each turned a new test red. Residual: the proposed gate check (fail on an exported
function at 0%) is not implemented.

**L1-40 (medium, CONFIRMED, 2 of 2, both reproduced).** *Claim.* Seven live data-loss guards are never exercised by any
test: log.go:556-562 ("no segment files"), log.go:586-590 ("segments with frames but no quill.tail"), read.go:236-238
(never serve a frame past the marker's sequence), read.go:267-272 (read-time CRC failure becomes a `GapError` or
`ErrCorrupt`), stat.go:219, stat.go:277-279, load.go:389-394. *Reproduction.* Probes: a 10-record log with segment 0 deleted:
"corrupt log: no segment files"; the same with `quill.tail` deleted: "segments with frames but no quill.tail"; a Reader on
a forged marker `seq:5` returned 5 records, not 10; one byte flipped after `Open`: `Read(1,0)` returned a `GapError`
1 to 30. A skeptic removed or changed each guard and the suite stayed green. *Warrant.* An uncovered guard can be
inverted with the suite green. *Backing.* Decisions 39, 62, 63. *Rebuttal.* A test gap, not a defect; one refusal
is partly redundant (a later check also returns `ErrCorrupt`). *Disposition.* Fixed in e3021b7, FV (6 of 6): one new test
file; each guard deleted or inverted in turn turned its test red (the three guards the first skeptic named but did
not run are now covered too).

**L1-43 (low, CONFIRMED, 2 of 2, both reproduced; filed as a test, affects the product).**
*Claim.* `TestManyProducersSyncingConcurrentlyShareFsyncs` fails deterministically at `GOMAXPROCS` 1, and on a real disk group
commit shares far fewer fsyncs there (about 134 of 160 syncs against about 42-54 at 10 procs). The gate runs at the default
`GOMAXPROCS` only. *Grounds.* `go test -run '^TestManyProducersSyncingConcurrentlyShareFsyncs$' -count=50 -cpu N`: 50 of 50 fail at `-cpu 1`
("160 fsyncs for 160 syncs"), 0 of 50 at 2, 3, 4, 10. Real disk, macOS, 16 producers x 10 `Append`+`Sync`: `-cpu 1` 134/140/129 fsyncs
(1.10-1.20 s), `-cpu 2` 87/93/54, `-cpu 4` 74/60/48, `-cpu 10` 54/42/48 (0.45-0.55 s). *Warrant.* Batching depends on
parallelism in the product, not only the sim. *Backing.* scope.md:67, 238-239; userguide.md:41; decision 34 makes no
`GOMAXPROCS` promise. *Rebuttal.* Durability and ordering are right; the cause (the flusher takes Go's `runnext` at one P
and flushes one record while holding `l.mu`) is inferred, not proven. *Disposition.* Fixed in 31f17bc, FV (6 of 6):
the flusher calls `runtime.Gosched()` once after a Sync request, before taking the lock; a one-proc regression test fails
30 of 30 without it and passes 30 of 30 with it. Decisions 13 and 34 (the flush holds the lock through the fsync) are
unchanged. Residual: the test bound is only `fsyncs < syncs` (real count about 12).

**L1-48 (low, CONFIRMED, 2 of 2, both reproduced; perf).** *Claim.* `Reader.Check` verifies segments serially; parallel
verification as `Open` does is 3.5x to 6x faster with identical output. *Grounds.* stat.go:194-250, 288; load.go:317; figures above.
*Warrant.* The per-segment work is independent except for the `At` carried from the previous segment, which the merge
supplies. *Backing.* Decisions 44, 66 (what `Check` examines, not how fast), decision 39 (Open's parallel loop).
*Rebuttal.* Warm cache only in the original figures; one goroutine per segment, all cores for the duration; gain
overstated for tiny segments (a skeptic saw 1.5 to 2x on 4 KB segments). *Disposition.* Fixed in 1aff9f0, FV (7 of 7):
sem-bounded goroutines as in `Open`, merge in id order, the first record's clock tests done in the merge. A new test
fails under five hand mutations of the merge. Residual: rotational disk unmeasured; the prototype's pooled buffer was left
for L1-49.

**L1-49 (low, PLAUSIBLE, 2 of 2 as CONFIRMED, PLAUSIBLE; perf).** *Claim.* `verifySegment` and `verifyBounded` allocate a
fresh read buffer per segment; a pool makes `Open` and `Check` of a many-segment log 14% to 55% faster. *Grounds.*
load.go:219-220, 121-146; stat.go:289. *Warrant.* Per-segment allocation and zeroing is a large share of work for 64
KiB to 1 MiB segments. *Backing.* The Log and Reader already pool this buffer (log.go:399, reader.go:101). *Rebuttal.* Not a win at
8 MiB segments on macOS; the buffer is not always 1 MiB; macOS timings were unstable under load (one skeptic saw `CheckMany`
8% worse then 12% better); the B/op drop reproduced exactly. *Disposition.* Fixed in a36e59c, FV (6 of 6): a
package-level `sync.Pool` through `verifyWindow`, zero-capacity at first so a small log allocates little. Mutating the
pool (stale length kept) failed dozens of tests. Residual: no committed benchmark guards the saving; a pooled buffer
that grew for a frame of up to 64 MiB stays until the next GC.

**L1-50 (low, CONFIRMED, 2 of 2, both reproduced; perf).** *Claim.* `Reader.segment` probes a sealed segment's first frame
through a 64 KiB first read; 4 KiB (`firstFrameChunk`, as the active probe uses) makes the first cold view 28% to 45%
faster. *Grounds.* reader.go:249, 375; load.go:116. *Warrant.* A header and a short payload need not copy 64 KiB per segment;
`window.bytes` enlarges a read to the frame size. *Backing.* Decisions 91, 63. *Rebuttal.* Only cold views benefit; a frame over
4 KiB needs a second read. *Disposition.* Fixed in ea4ca7c, FV (6 of 6): one line, a dated note on decision 91, and a test for
first frames below, at and above the probe size (mutating `window.bytes` to never enlarge makes it panic).

**L1-52 (low, CONFIRMED, 2 of 2, both reproduced; perf).** *Claim.* `printer.print` copies text payloads byte by byte
through a switch; copying runs via a lookup table makes the CLI's `cat`, `tail` and `read` of text records 27% faster with
byte-identical output. *Grounds.* print.go:82, 116 (80% of print's CPU for text in pprof). *Warrant.* Per-byte appends for
mostly ASCII payloads. *Backing.* Decision 35 fixes the bytes; `TestPrintMatchesReference`. *Rebuttal.* Only valid UTF-8 text
benefits; terminal rendering dominates on a tty. *Disposition.* Fixed in 81d2699, FV (6 of 6): two `[256]bool` tables; six
hand mutations all failed the existing reference test; a fix-skeptic differential found no difference. Residual: no new
committed differential test; one now-unreachable `case` remains in `appendQuoted`.

**L1-53 (low, PLAUSIBLE; votes CONFIRMED, PLAUSIBLE; perf).** *Claim.* The CLI printer wraps stdout in a 4 KiB `bufio.Writer`;
64 KiB cuts write syscalls 16x. *Grounds.* print.go:29; figures above. *Warrant.* Sys time fell about 3x. *Backing.* Decision 35
fixes the bytes, not buffering; follow mode still flushes per record. *Rebuttal.* A slow non-follow `cat` shows output in larger
steps; a kill loses up to 64 KiB of unflushed output instead of 4 KiB; the 29% figure to a file was not reproduced
(7-12% to a pipe). *Disposition.* Fixed in 15ddde4, FV (6 of 6); `TestPrinterWritesInLargeBlocks` fails with the old size;
CHANGELOG states the trade-offs.

### Fix written, rejected by the fix-skeptic, not in the tree (3)

These three fixes exist on attempt branches (52be0dd, 8192ca0, 9a3f648; verified not ancestors of the tip). Each was
rejected for a specific defect, named below, and is open.

**L1-3 (low, CONFIRMED, 2 of 2; one reproduced).**
*Claim.* `GapError`'s doc says "From is the requested offset"; for `Scan`, `Reader.Scan` and `ScanAll`, a gap met after
records were delivered carries `From` = the first missing sequence. *Grounds.* read.go:149-153, 164-167, 272, 280; seek.go:94;
reader.go:463; errors.go:51-53. *Reproduction.* 8 sealed segments of 5 records, segment 3 removed while open: `l.Scan(1,0,fn)`
delivered 15 records and returned `GapError{From:16, To:20}`; a skeptic's variant printed "requested 1; delivered 1..8; err
records 9 to 12 are unavailable", while `Read(1)` returned 8 records and nil. *Warrant.* The scan advances `cur` before it meets
the gap. *Backing.* Decisions 38, 84 ("Scan returns a gap error even after delivering records"), 101. *Rebuttal.* `Read`,
`ReadWait` and `Consumer.Read` leave the gap for the next call, where `From` equals the new offset; `Commit(To+1)` stays correct.
*Disposition.* Fix 52be0dd (docs only: `GapError`, userguide, decisions 101 and 103) was REJECTED. Objection (reported by the fix-skeptic,
run on the attempt tree): `Consumer.Scan` also returns the gap immediately after delivering records (delivered [3 4], then
"records 5 to 8 unavailable"), so the CHANGELOG sentence "Read, ReadWait and the consumers' reads leave such a gap for the next
call" is false for `Consumer.Scan`; decisions 101 and 103 and the commit message list "Scan, ScanAll or Reader.Scan" and omit
`Consumer.Scan`; the test covers only `Reader.Scan` and `ScanAll`. Remaining work: correct that sentence to name
`Consumer.Read` and `Consumer.ReadWait`, add `Consumer.Scan` to the lists, add a `Consumer.Scan` case. See the owner file.
*Disposition, updated 2026-11-03.* Owner-approved to finish. Attempt 47df71b documents `From` as the first unavailable sequence for every call form, `Records` and `Follow` included, with a test per form, and was REJECTED again: CHANGELOG.md:197 (an Unreleased `Fixed:` entry) still says a Log's `Read`, `Scan`, `ScanAll`, `Records`, `Follow` and consumer reads return a `GapError` "starting at the offset asked for", which is false for the scans after delivered records and for `Records` and `Follow`, and the new test covers the Reader and a Reader-backed consumer only (the recorder read the changelog line on the branch). Open; remaining work and the rebasing note are in [stage2-owner-decisions.md](../decisions/stage2-owner-decisions.md), "Open after a rejected fix".

**L1-10 (low, PLAUSIBLE; votes CONFIRMED, REFUTED, CONFIRMED).** *Claim.* The `Config.Logger` doc enumerates what is logged but
omits the MinFreeBytes transitions, "age rotation skipped because the flush failed", the unreadable-consumer-file Warn, the
staleness-probe-failure Warn and the clock-guard warnings. *Grounds.* retention.go:129-137, 148, 172, 196, 268, 283, 285 against
log.go:71-79. *Warrant.* The doc is the only list of records and the handler constraints apply to all of them. *Backing.* Decision
60 A5-1, decision 25. *Rebuttal.* The sentence has no "only"; the REFUTED voter found no spec text making the list exhaustive,
and the clock-guard part is covered by "bulk-expiry warnings". *Disposition.* Fix 8192ca0 (a complete grouped list in log.go,
scope.md:122 and userguide.md:205) was REJECTED. Objection (fix-skeptic, reproduced: 6 identical Warns for 6 passes): the new text
says the MinFreeBytes records are "one record per transition, not per pass", but the "cannot read free space" Warn
(retention.go:268) is written on every pass where `statfs` fails, with no guard; the fix swapped an omission for a false statement
in three places. Remaining work: say the cannot-read Warn is written on every failing pass and keep "one per transition" only for
the below-floor Warn and the back-above Info (or add a transition guard in code). See the owner file.
*Disposition, updated 2026-11-03.* Owner-approved to finish as documentation only (no transition guard in code). Attempt 0a77730 lists every record and says the "cannot read free space" Warn is written on every failing pass and seal, and was REJECTED again: it says a deleted or already-gone segment "is one record when it goes" in four places, but `deleteLocked` logs the Warn "segment was already gone" and then also the deletion record (the skeptic reproduced two records for a segment removed by hand and then pruned; the recorder read the code on the branch, retention.go:345-374). Open; see [stage2-owner-decisions.md](../decisions/stage2-owner-decisions.md), "Open after a rejected fix".

**L1-21 (low, CONFIRMED, 2 of 2; reproduced by the auditor).** *Claim.* The O(log segments) cost of `OffsetForTime` holds for a
Log and a warm Reader; a Reader's first call opens about every sealed segment. *Grounds.* seek.go:11-12; scope.md ("O(log
segments) plus one segment"); reader.go:343-353. *Reproduction.* `TestZ1SeekCost` with a counting filesystem on a 401-segment
log: the first `Reader.OffsetForTime` opens 403 files, the second 2. *Warrant.* The binary search is O(log n) over a view
built from O(n) probes. *Backing.* Decisions 59, 47, 63 (partly discloses the per-segment probe). *Rebuttal.* Accurate after
the first call and for the writer. *Disposition.* Fix 9a3f648 (seek.go, scope.md, decisions 59 and 63 and a CHANGELOG line state
the cost as O(log segments) comparisons plus one segment scan over a view costing O(sealed segments) per call) was REJECTED.
Objection (fix-skeptic): docs/design/build-prompt.md:385 (test T19) still reads "cost touches O(log segments) first frames and
one segment", contradicting the amended text; commit 50201e1 shows the repo updates that file alongside such wording fixes.
Remaining work: that one sentence. See the owner file.
*Disposition, updated 2026-11-03.* Owner-approved to finish. Fixed in 41297fb (on `main`), FIXED-VERIFIED by a fix-skeptic, 6 of 6 lenses pass (reported): build-prompt.md:385 (T19) and every other statement of the cost now read O(log segments) comparisons plus one segment scan over a view costing O(sealed segments) per call, and the dated notes on decisions 59 and 63 say "Stage 2 audit, owner-approved 2026-11-03". Docs and comments only; the non-comment lines of seek.go are unchanged. The skeptic did not re-run the 403-opens reproduction.

### Held for the owner (7)

Full sections, with the recommended answer, are in [stage2-owner-decisions.md](../decisions/stage2-owner-decisions.md), "Pass L1".

**L1-4 (low, CONFIRMED, 2 of 2, neither skeptic ran it; reproduced by the auditor on `fsx.Sim`).** *Claim.* `ErrFailed`'s doc, decision 37 and the
user guide say "every later Append, Flush and Sync fails" on a failed log; `Sync(seq)` for an already durable sequence
returns nil, and for an unassigned one `ErrBadSeq`. *Reproduction.* 3 records appended, 1-2 flushed; a data-fsync error
injected; `Flush`: "log failed; close and reopen: EIO"; `Sync(ctx,1)` nil, `Sync(ctx,3)` the `ErrFailed` error, `Sync(ctx,99)`
`ErrBadSeq`. *Grounds.* flush.go:285-288, 293-295; errors.go:31-36. *Warrant.* The switch tests `l.head >= seq` before
`l.failed`. *Backing.* Decisions 37, 42 (unless durable). *Rebuttal.* The behaviour is right (those records are durable);
the wording is wrong. *Disposition.* Held: the reworded guarantee is a spec text change.
*Disposition, updated 2026-11-03.* Owner-approved (reword; do not change the code). Fix de3cfd4 rewords the claim in errors.go, decision 37, the user guide and scope.md and pins all three `Sync` behaviours on `fsx.Sim`; REJECTED by the fix-skeptic for an unfinished sweep: docs/design/build-prompt.md:300 (T3) and the runtime log message at flush.go:135 with its quotation at docs/userguide.md:53 keep "every later call returns ErrFailed" (the recorder read the lines on the branch). Not on `main`; open. See [stage2-owner-decisions.md](../decisions/stage2-owner-decisions.md), section L1-4.

**L1-5 (low, CONFIRMED, 2 of 2, neither skeptic ran it; reproduced by the auditor).** *Claim.* `Append` and `AppendBatch` document no error-path return value and the code
is inconsistent: `appendLocked` returns `(first, ErrFailed)` when an inline flush's fsync fails, `AppendBatch` discards it
and returns `(0, err)`; after a `SyncAppends` failure it returns `(first, err)`. *Reproduction (auditor, `fsx.Sim`,
`FlushBytes=100`, fsync error injected).* `Append` of 200 bytes returned seq 0 and the fatal error while `nextSeq=2` and
`bufCount=1`; `AppendBatch` also returned 0. *Grounds.* append.go:38-46, 91-97, 10-12; log.go:33-35. *Warrant.* The sequence is
consumed and the record is buffered, but 0 is returned. *Backing.* Decision 42; decision 100 is about the CLI only.
*Rebuttal.* "0 on error" may mean "no sequence promised"; after a fatal fsync the log is reopened anyway and recovery adopts
verified frames; survival is inferred from format.md:104, not run. *Disposition.* Held: either choice is an API contract.
*Disposition, updated 2026-11-03.* Owner-approved (return `first` on every failure after sequences were assigned). Fixed in 867f4ed (on `main`), FIXED-VERIFIED by a fix-skeptic, 7 of 7 lenses pass (reported): `AppendBatch` returns `(first, err)` on an inline-flush failure (append.go:52), 0 only where nothing was assigned; documented; tests in append_error_seq_test.go; red on the parent for the two inline-flush cases (reported). Residual: the text does not warn that an unpersisted record's sequence can be reassigned.

**L1-11 (medium, CONFIRMED, 2 of 2, both reproduced; bug).**
*Claim.* `Reader.Check` reports HEALTHY when the oldest sealed segment is truncated to zero bytes, though `Open` reports a hole
from 1 and a sticky load error. *Grounds.* stat.go:173-175, 285-287, 238-240 (`if info.frames == 0 { continue }` applies to
sealed and active alike), 241-251; load.go:228-230; decision 43 and format.md:88 ("a sealed segment with no frames is corruption").
*Reproduction.* A 120-record, 3-segment log with `0000000000.log` truncated to 0 bytes: `Check` returns healthy, no problems;
the writer's `Stat` shows holes `[{1 120}]` and `Err()` "damaged at byte 0 ... a sealed segment with no frames"; the Reader's
`Stat` shows holes `[{1 120}]`, first 121. A skeptic's 400-record variant: `Check` healthy, `Stat` first 401, holes `[{1 400}]`;
garbage over the same file made `Check` report a "segment" problem. (An earlier "middle-empty" comparison actually truncated
the active segment and is not valid.) *Warrant.* `Check` skips zero-frame files without a problem and starts contiguity at the
next survivor, so a hole at the front is not reported. *Backing.* Guarantee 6, decision 38. *Rebuttal.* quill cannot produce an empty
sealed file itself, so it needs outside damage (a tmp cleaner, a bad restore); a middle hole is still reported (as a hole
naming the next file). *Disposition.* Held: reports a new fatal finding where `Check` today says healthy, an operator-visible
exit-code change (0 to 3); see the owner file for the recommended answer.
*Disposition, updated 2026-11-03.* Owner-approved (report it, as `Open` does). Fixed in ce47243 (on `main`), FIXED-VERIFIED by a fix-skeptic, 7 of 7 lenses pass (reported): `Reader.Check` reports a sealed segment with no frames as a fatal `segment` problem (stat.go:272), `quill check` exits 3 where it exited 0; red on the parent for the oldest and the middle position (reported). Residual: the Reader's `Stat` does not report an empty middle or last sealed segment as a hole (a sibling, not filed; see the owner file).

**L1-12 (medium as filed, PLAUSIBLE; votes REFUTED, CONFIRMED, CONFIRMED; all three reproduced).**
*Claim.* `Archive` into a new or empty destination after the `_archive` consumer has advanced copies only records from the
consumer's offset, returns nil, and gives no signal the destination is incomplete. *Grounds.* archive.go:17-18, 80-82, 94, 75;
userguide.md:328-331, 341-345. *Reproduction.* 30 records, `Roll`, `Archive(src, dA)` nil; 30 more, `Roll`, `Archive(src, dB)` with
`dB` new: nil. dA first 1, head 30; dB first 31, head 60, `check` HEALTHY exit 0, holes `[]`; source first 1, head 60. Also
via the CLI: `archive -to dA` then `archive -to dB`, both exit 0. *Warrant.* The copy decision is made only from the source-side
consumer offset and gap detection needs the offset below the source's first or a destination tail above 0. *Backing.* Decisions 28,
85. *Rebuttal.* The consumer is per source by design (decision 28), a standing single destination is unaffected, and `quill
commit -name _archive -offset N` rewinds it. The REFUTED voter said decision 85 documents it; the arbiter (third vote) did not
find that text and sided with a false comment and misleading docs. *Disposition.* Held: the minimum is wording; the real fix
(refuse or `GapError` for a new destination) is a design change.
*Disposition, updated 2026-11-03.* Owner-approved, wording only. Fixed in be8cb15 (on `main`), FIXED-VERIFIED by a fix-skeptic, 6 of 6 lenses pass (reported); no code changed, and the test (archive_test.go:370) pins the documented behaviour (it fails under a mutation that copies the whole source). The optional refusal (an error, not a `GapError`, when the destination is new and the source's first retained sequence is below the `_archive` offset) is not implemented and stays an open optional follow-up that needs its own owner decision.

**L1-16 (low, CONFIRMED, 2 of 2; reproduced by the auditor).** *Claim.* `Consumer.ReadWait` is documented to return when "the
handle closes", but `Consumer.Close` does not wake it. *Grounds.* consumer.go:249, 253-255, 262-271. *Reproduction.*
`TestT4ReadWaitClose`: a caught-up consumer, `ReadWait` in a goroutine, `c.Close()` returns nil, `ReadWait` still blocked 2 s later;
afterwards `Read` returns 0 records and nil, `Records` iterates normally. *Warrant.* In consumer.go "handle" means the
Consumer. *Backing.* consumer.go:249 text. *Rebuttal.* "The handle" may mean the Log or Reader (the sentence is copied from
`Reader.ReadWait`). A secondary mismatch noted by a skeptic: errors.go:23 says `ErrClosed` covers "a call on a closed Log,
Reader or Consumer", but only `Commit` and `Close` return it on a closed Consumer. *Disposition.* Held: reword, or make
`Close` wake waiters (a behaviour change).
*Disposition, updated 2026-11-03.* Owner-approved (reword; do not change the code). Fixed in 923623c (on `main`), FIXED-VERIFIED by a fix-skeptic, 6 of 6 lenses pass (reported): `Consumer.Close` releases only the lock; `Read`, `ReadWait`, `Scan`, `Records` and `Follow` ignore it; on a closed Consumer only `Commit`, `SeekToTime` and a second `Close` return `ErrClosed`; code unchanged, the test (consumer_close_test.go:14) fails under mutation of the closed checks.

**L1-26 (low, CONFIRMED, 2 of 2, both reproduced).** *Claim.* `quill append` on an oversize stdin prints "stdin is a batch of more
than 16777216 bytes", which is not stdin's size: the limit applies to the sum of `len(line)+34`. *Grounds.* cmd/quill/write.go:162-168;
decision 46. *Reproduction.* `yes x | head -n 479349 | quill -dir t3 append` exit 0; 479,350 lines (958,700 bytes on stdin): "quill:
payload too large: stdin is a batch of more than 16777216 bytes, MaxBatchBytes is 16777216", exit 1. *Warrant.* Presented as a
byte count of stdin; false for stdin under 1 MB. *Backing.* Decision 46; the code comment at write.go:159-161 is right.
*Rebuttal.* "Batch" can be read as the encoded batch; the skeptics note the encoded batch is about 7 MB, so that does not
hold either. *Disposition.* Held: any test or script pinning the message text changes.
*Disposition, updated 2026-11-03.* Owner-approved (reword the message). Fixed in c2cddb7 (on `main`), FIXED-VERIFIED by a fix-skeptic, 7 of 7 lenses pass (reported): the message is now "stdin would be a batch over MaxBatchBytes (16777216); each line is counted as its length plus 34 bytes" (cmd/quill/write.go:166); limit, error identity and exit code unchanged; red on the parent (reported). The library's own "batch of about N bytes" at append.go:46 is a similar imprecision outside the claim.

**L1-51 (low, PLAUSIBLE; votes REFUTED, CONFIRMED, PLAUSIBLE; one skeptic reproduced a smaller gain; perf).** *Claim.* Narrowing
`indexStride` from 64 KiB to 16 KiB makes a 100-record `Read` at a random sealed position or near the tail 22-27% faster, at 4x the
index memory. *Grounds.* load.go:23; benchmark-baselines.md:105 (already a follow-up); numbers above. *Warrant.* A positioned read
pays about half a stride of frames and `pread`. *Backing.* Decisions 63, 91 fix the idea of an index, not the stride; architecture.md:42
says "one anchor per 64 KiB". *Rebuttal.* Memory is the cost (24 B per anchor; about 1.5 GB per TiB retained, per Log and per Reader that
scanned every segment); the Tail benchmark's alignment effect is partly an artefact (32K shows nothing on Tail); one skeptic saw
37.8 to about 32 us on Sealed, a smaller gain. The full suite passes at 16 KiB (a skeptic ran it). *Disposition.* Held: a
memory-for-latency trade-off.
*Disposition, updated 2026-11-03.* Owner-approved on 2026-11-03: keep 64 KiB. Deliberately not changed, no code change; reopen only if a real workload shows random-position reads dominating (then 32 KiB, with architecture.md:42, the `load.go` comment and the baselines updated together).

### Recorded, not applied (4)

All four are confirmed safe by two skeptics who applied the patch in a scratch copy (it builds, vets and passes the package
tests). They are low-value cleanups with no behavioural effect; none was applied in this pass. Apply as a later tidy-up.

- **L1-32 (low, CONFIRMED, 2 of 2, both reproduced).** `scanResult.frames` and `scanResult.lastAt` (reader.go:76-77, literal at
  reader.go:314) are written once and never read. The type is unexported. Fix: delete both fields and both initialisers;
  gofmt then realigns the other two.
- **L1-33 (low, CONFIRMED, 2 of 2; one reproduced).** `sort.Strings(names)` in `cmdReference` (verbs.go:290-291) is redundant:
  `flag.FlagSet.VisitAll` visits in lexicographic order (verified in the Go 1.27.3 source). A single `VisitAll` loop produces a
  byte-identical `docs/reference.md`; the `sort` import goes. Rebuttal: the ordering is a library guarantee, documented since Go 1.0.
- **L1-35 (low, CONFIRMED, 2 of 2; one reproduced).** `frameHeader`'s four failure returns (codec.go:142, 146, 151, 155) and the
  `!ok` branch in `indexBuffer` (flush.go:141) are unreachable: `l.buf` has one producer, `appendFrame` (append.go:85), and
  five blocks are at coverage count 0. Either drop the `ok` result or turn the `return` into a panic carrying the cause (the
  louder option; dropping it surfaces a broken invariant as an out-of-range slice panic with `l.mu` held, after the bytes are
  already durable).
- **L1-38 (low, CONFIRMED, 2 of 2; one reproduced).** `Consumer.src` duplicates `Consumer.store.src` (set once at consumer.go:89,
  never reassigned; 6 reads, not the 4 listed). Fix: delete the field and use `c.store.src`. Rebuttal: stylistic.

### Refuted (5, kept with the reason)

- **L1-13 (filed low, bug; REFUTED, 2 of 2; reproduced).** A prune pass that deletes nothing clears the sticky unlink error and logs
  "background operation recovered" while the undeletable segment is still on disk. Reproduced on macOS with `chflags uchg` and a newly
  protecting consumer (`TestW1PruneClear`). Refuted because decision 60 (A5-1 amendment, owner-approved 2026-10-30) says it in so
  many words: "a prune pass that finds nothing left to delete after a failed unlink (the segment became protected, say) logs it and
  clears Err()", userguide.md:68 repeats it, and `bg_log_semantics_test.go:187-235` pins it. What survives is wording drift in the
  Prune doc (retention.go:102-105, "until a later pass deletes it") and the comment at retention.go:201-205.
- **L1-30 (low, hygiene; REFUTED, 2 of 2).** Three `cmd/quill` comments cite audit ids (`A5-44`, `A1-57`, `A1-50`). Refuted: the
  documents the ids resolve against (decisions.md and docs/audits/) are tracked and ship with the repository, and the decision
  numbers the comments cite carry the same ids. Cosmetic. (A similar id in the `Archive` godoc was a public-text defect and was
  fixed under L1-22d.)
- **L1-41 (low, test; REFUTED, 2 of 3).** `TestOpenAheadWarningAbsentWithoutALogger` "asserts almost nothing". Refuted by mutation: with
  the nil-Logger guard in `logf` removed it panics at ahead_warn_test.go:135, so it pins that path. What survives is a naming
  nit (the title says "warning absent", the body pins "no crash"). One voter, CONFIRMED, found it weak but not vacuous.
- **L1-44 (low, perf; REFUTED, 2 of 2; reproduced; merged L1-46).** docs/audits/reference/benchmark-baselines.md and CHANGELOG.md "no
  longer describe HEAD": a flush now allocates 10 times (1,256-1,311 B), not 6, and a Reader poll about 80 more per 10 ms.
  Bisected (reported by the auditor) to 6acc23e (A5-16, an `fstat` plus `lstat` of the lock per flush) and 495e457 (A3-30,
  `checkLogID` re-reading `quill.format` per poll). The allocation facts reproduce and no time moved beyond noise
  (`AppendFlushEach` 9.74 ms at v0.1.0, 10.0 ms at HEAD). Refuted because the baseline file and the v0.1.0 CHANGELOG section are
  dated records of v0.1.0 and are accurate for it, the Unreleased section already states both costs, both are owner-approved
  correctness fixes, and the rule is about time, not allocations. What survives: re-record the baselines at the release gate
  (scope.md:459, reported by a voter). Optional, an owner trade-off: re-check `quill.format` less often than every poll.
- **L1-47 (low, perf; REFUTED, 2 of 3).** Warm decode throughput on Linux about 8-10% slower at HEAD than at v0.1.0. Skeptics
  reproduced +6% to +9% with overlapping ranges; under the 10% rule it is not a regression. No cause identified (candidates:
  the non-minimal-varint refusal A3-21 in `codec.go`'s `uvarint` wrapper, A3-22); not bisected. One voter, CONFIRMED the slowdown
  exists, with no action. To apply the rule where it is defined, re-run `ScanAll` and `ColdStart` on the macOS baseline machine.

### Decided in round 1 (4, prior-deliberate, not re-voted)

- **L1-17.** The `Archive` doc and the user guide say `quill check` on the destination keeps reporting the hole; for a loss at the
  front of a new destination, `check` reports it healthy (auditor test `TestU2FrontLoss`: after `Archive` returned the
  "records 31 to 290 are unavailable" `GapError`, `OpenReader(d2).Check()` healthy, first 291, holes `[]`). Already recorded as open
  point 2 of A3-10 in stage2-owner-decisions.md (verified by reading it); not reopened here.
- **L1-34.** `Log.fsyncs` (log.go:328, incremented at flush.go:70) is read only by sync_test.go:50. Decided in round 1 (A1-7,
  a1-round2-simplify.md); the ledger records the test-side alternative (count `OpSync` on `.log` files in the `Inject` callback with
  an atomic) and its cost.
- **L1-36.** `utf8.Valid`, `checkEscapes`, `hex4` and `ErrBadString` never change an accept or reject verdict for the one production
  caller (a struct of four `*uint64` fields). A 60 s differential fuzz (about 340,000 executions) never differed. Decided in round 1
  (A1-1; decisions 72 and 105): defence in depth for a general package and a documented contract.
- **L1-42.** scope.md:238-239 says the log's lock is not held through the `Sync` fsync, but the flusher holds `l.mu` across the segment and
  marker fsyncs, so in-process `Append`, `Read`, `Head` and `Stat` stall for the flush (repro: a 300 ms injected sync stalled `Stat`,
  `Head`, `Append` and `Read` for about 500 ms; real macOS: `Read(0,1)` p50 19.9 ms, p99 25.4 ms while another goroutine loops
  `Append`+`Sync`, against about 25 us idle). Round 1 (A1-30, decision 13 "moving the fsync outside the lock is a measured
  follow-up") deliberately left the code. Recorder's observation (verified by reading scope.md:239 at the tip): the sentence still
  reads as the auditor quoted it. It is true of the producers' wait, not of the flusher. Wording, not code, is what would change;
  it is not re-opened here.
