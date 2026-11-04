# Pass A1d: adversarial audit of the v0.1.0..HEAD diff

Date 2026-11-03. The audited tree is the tip as it stood before this pass's fixes, `e2d35d1` (inferred: it is the parent of the first
fix commit, `d45f1f2`). Pass A1d is the diff-scoped form of the stage 2 adversarial audit (A1): instead of auditing the whole
tree, three auditors reviewed what changed between the `v0.1.0` tag and the tip, one lens each (code, tests, docs), and each
reported only what it could ground. Line numbers below are from the tree as it is now, except where a section says otherwise.

Labels: **verified** (run, or the exact lines read, by an auditor or skeptic named in the claim), **reported** (a document or an auditor says
it; the source is named), **inferred** (constructed from what is named). This file is a record: the recorder read the ledger the
auditors and skeptics produced and the commits' messages and diffstats, and ran nothing. Every "verified" below is the
auditor's or skeptic's, not the recorder's. Companion files: [stage2-owner-decisions.md](stage2-owner-decisions.md) (the one
item held for the owner, section "Held after pass A1d"), [a1-round2-findings.md](a1-round2-findings.md) (the earlier A1
passes), [live-integration.md](live-integration.md), [gap-audit.md](gap-audit.md), [mutation-hardening.md](mutation-hardening.md)
and [verification-optimization.md](verification-optimization.md) (the ledgers this pass's items cite).

## Summary

Fourteen items. Nine are CONFIRMED, one is REFUTED, and four were not put to a vote because an earlier pass had already decided
the same question (round 1 of A1, or the gap audit A7). No item is PLAUSIBLE.

| Verdict | Items | Disposition | Items |
|---|---|---|---|
| CONFIRMED | 9: A1d-2, 3, 4, 5, 6, 7, 9, 10, 12 | fixed, fix-skeptic FIXED-VERIFIED | 8: A1d-2, 3, 4, 5, 6, 7, 10, 12 |
| REFUTED | 1: A1d-8 | fix written, rejected by the fix-skeptic, not merged; held for the owner | 1: A1d-9 |
| decided in an earlier pass, not voted | 4: A1d-1, 11, 13, 14 | deliberately not changed (earlier decision stands) | 4: A1d-1, 11, 13, 14 |
| PLAUSIBLE | 0 | refuted, nothing changed | 1: A1d-8 |

- **By kind** (the kind is what the finding is about; the lens that raised it is inferred from the kind: code lens for bug and
  compat, tests lens for test, docs lens for doc and hygiene). bug 1 (A1d-1), compat 2 (A1d-2, 3), test 4 (A1d-4, 5, 6, 7), doc 6 (A1d-8, 9, 10,
  11, 12, 13), hygiene 1 (A1d-14).
- **Severity as filed.** medium 3 (A1d-1, 4, 5), low 11. No item was filed high.
- **Skeptic votes.** 21 votes on the 10 items that were voted on: nine items with two votes each, and A1d-8 with three. 19
  CONFIRMED, 2 REFUTED. Both REFUTED votes are on A1d-8; its third vote was CONFIRMED, so the finding was refuted 2 to 1.
  Every CONFIRMED item was unanimous.
- **Reproduced.** 13 of the 21 votes carry a reproduction (a program, a mutant or a command run in a scratch copy of the tree,
  with its output). Seven of the ten voted items have at least one reproducing vote: A1d-2, 3, 4, 5, 6, 7 and 12. A1d-3 has one
  reproducing vote of two, the other read the struct definitions. The three voted items with none (A1d-8, 9, 10) are documentation
  items settled by reading code and documents. A1d-1, which was not voted, was reproduced by its auditor.
- **Fix-skeptics.** 8 fixes, 8 FIXED-VERIFIED, 50 lens verdicts (7 each for A1d-2 and A1d-3, 6 each for the other six), all
  pass (reported by the fix-skeptics, who ran the tests and the gate's checks on copies of the tree; not re-run by the
  recorder). The ninth fix, for A1d-9, was rejected by its fix-skeptic and is not merged (see the item). Fixes before the
  accepted ones: A1d-2 and A1d-9 each had an earlier attempt rejected (reported by the fix summaries; neither earlier
  commit is named in the ledger).
- **Fix commits.** All eight are in the tree (verified by `git log`): `d45f1f2` (A1d-4), `6eb1bdd` (A1d-5), `96588fb`
  (A1d-2), `65b7cb0` (A1d-3), `ae71dba` (A1d-6), `41aaef3` (A1d-7), `652154f` (A1d-10), `85fecc9` (A1d-12). `f2287c7` (A1d-9) exists in
  the repository and is not an ancestor of the tip (verified by `git merge-base --is-ancestor`).
- **What changed in production behaviour.** One fix changes the program: A1d-2 (Open sets an existing `quill.tail` to
  `Config.Perm` in place). The other seven are tests (A1d-4, 5, 6, 7) or CHANGELOG text (A1d-3, 10, 12). None touches an on-disk byte.
- **No defect in the log format, in recovery or in the guarantees was found by this pass.** What it found is one behaviour
  regression of low reach (A1d-2, fixed), one cost under a lock that an earlier decision accepted (A1d-1), four gaps in tests that
  leave a reintroduced defect unnoticed or make a test flaky, and documentation that is stale or inaccurate.

### Coverage statement

Three auditors returned and none failed to return (code, tests, docs). What each checked and found clean, and what it did not
cover, follows. Where an auditor ran code, the toolchain was Go 1.27.3 with `GOWORK=off`.

**Code lens (A1d-code).**

- *Scope.* The whole non-test diff of the root package (append, flush, log, load, retention, read, reader, stat, state, consumer,
  archive, config, codec, seek, iter), `cmd/quill` (verbs, write, read, print, consumers), `internal/fsx`, `internal/atomicfile`,
  `internal/strictjson`, `build/quill`, `hack/gate.sh` and the CI workflow were read in full. `hack/topology`, `hack/skewopen`, `hack/powertest`,
  `docs/` and the test files were skimmed, not read line by line.
- *Race detector.* `go test -race -count=10` over the root package and `./internal/...` on macOS arm64: all packages ok (root package
  568 s). `cmd/quill` and `hack/` were not run under `-race`. A targeted Linux run was still going when the auditor
  finished, so it is not reported as clean.
- *On-disk bytes.* The only testdata change is the re-chosen policy golden. `encodePolicy` is unchanged, the v0.1.0 policy
  bytes still equal its output for the v0.1.0 fixture values and still decode at the tip (throwaway test, passed). Frame, segment, tail, format and
  consumer goldens are unchanged. `decodeFrame`'s refusal of non-minimal varints cannot reject anything the v0.1.0 encoder wrote.
- *Version skew.* A log written by a v0.1.0 build was opened, read in full, appended to, checked with `Reader.Check` and reopened by a
  build of the tip, and the reverse, with small segments, a protected and an unprotected consumer: head, record continuity, consumer
  offsets and the check were consistent. macOS only.
- *Exported API.* The `go doc -all` diff was read in full. Additions are in the CHANGELOG (Stats fields, `ConsumerOptions.Rewind`,
  Problem kind `overlap`, `ErrFailed` and `Err()` semantics, `Consumer.Close` and `Archive` docs), except the two unexported
  fields of A1d-3. The behaviour changes traced (Config validation, the `_archive` name, the reader-created consumer file mode,
  protected-consumer creation without the consumer lock, non-minimal varints, tail and policy decoding limits, Open writing the marker
  only when moved) and the CLI changes (`prune` prints `dropped N`, `archive` exits 3 on lost records, positional arguments exit 2,
  `-from`/`-limit`/`-n` take plain decimals, `roll`/`recover`/`prune` refuse a non-log directory) are covered by CHANGELOG entries.
- *Interaction pairs traced, no defect.* (1) the unpublished-segment branch of `deleteLocked` with the nested seal that runs the caps;
  (2) `pruneLocked` resetting its background state against `deleteLocked`'s early return (an idle pass clears `Err()` and logs one
  recovery, by design, pinned by a test); (3) `fail()` waking waiters against `appendLocked` ordering; (4) lock-loss checks in flush and
  prune against `Close`; (5) the scan closure with segment views and the Reader's overlap handling; (6) the hole-from-1 rebuild when a
  damaged segment becomes oldest; (7) `Stat()` built from the hole list against the old view-based first sequence; (8) the 2^64-1 sequence ceiling
  across append, walk, tail read and load; (9) `appendJSONString`'s plain-byte table against `encoding/json`; (10) Reader cache
  eviction under concurrent views (performance only).
- *Supply chain and release inputs.* `go.mod` and `go.sum` untouched (no `require` line). The two CI action pins each resolve
  to a commit of the tagged version (checked with the GitHub API). `gofmt -l` empty and `go vet` clean for darwin and linux.
  `staticcheck`, `deadcode` and the fuzz step of the gate were not run by this auditor. A word-bounded search of the tip for the
  internal names finds nothing.
- *Not covered by this auditor.* `cmd/quill` and `hack/` under `-race`; any Linux full-suite race run; staticcheck, deadcode, fuzzing;
  line-by-line reading of the harnesses and of the tests.

**Tests lens (A1d-tests).**

- *Revert coverage.* 113 non-docs commits in the range change both production and test files (recount, Stage 2 L4b-4, 2026-11-04: "production" was not defined here. The 113 and the 74 below reproduce at `e2d35d1` when it means a non-test `*.go` file in any directory, `hack/` and `examples/` included: `git log --format=%H v0.1.0..e2d35d1 -- '*.go' ':!*_test.go'` lists 144 commits and `git log --format=%H v0.1.0..e2d35d1 -- '*_test.go'` lists 187; 113 are in both lists and 74 are in the second only. If "production" also means `build/` and `hack/*.sh`, as the restore set in the next sentence does, the split is 119 and 68. The 83, 12, 18 and 60 below are the auditor's and were not recounted). For each, a scratch copy of the tree at
  that commit had the commit's non-test Go, `build/` and `hack/*.sh` files restored to the parent's versions, and the tests the commit
  added or touched were run. 83 fail on an assertion, 12 fail to build (the test needs the new symbol), 18 pass. Of the 83, every newly
  added test function fails without the fix except 18 explicit "still works" guards, which the auditor read and judged to be guards by
  design. A reverse patch of each commit onto the tip conflicted for 60 of 113, so the per-commit revert is the primary result.
- *The 18 that pass without the fix.* 12 have comment-only or documentation-only production diffs; one pins a merge order of the
  parallel check; two are refactors whose tests pin behaviour; one is a data-race fix whose test is vacuous without `-race` and failed 3 of 3 with
  `-race` on the reverted code (the gate uses `-race`); one is a reword plus a pin; one is A1d-7.
- *The 12 that fail to build.* Five are renames, refactors or the new harness (no behavioural claim to mutate). For the commit that bounds
  sequence numbers by what segment names can hold, three production guards were mutated directly at the tip and each was killed by a test.
  For the commit that escapes names in output, see A1d-4 (two of four mutants survive). Five commits were not mutated individually: their coverage is
  bounded to "the test needs the symbol the fix introduced".
- *Targeted mutants of the tip, all killed.* directory sync before commit in `Archive`; directory sync on rotation; directory sync on
  `Close`; per-unlink directory sync; `MaxBytes` `>` to `>=`; the Reader's active-segment-shorter-than-marker check; the lock-file leak on a non-`ErrLocked`
  refusal and on an `ErrLocked` timeout; the tail walk's previous-sequence flag; the flusher ignoring a pending publish; an
  inline-flush failure returning nil; `build/quill`'s dirty-tree and untagged-tree refusals. One survivor was not claimed as a finding
  (see "Examined and not reported").
- *Flakiness.* On a Linux x86-64 machine (ext4, 4 vCPU, shared), `-race`: all new or changed tests of the root package and `cmd/quill` (318 and
  84 functions, excluding ten slow ones) 50 runs each, 0 failures; the same sets under `GOMAXPROCS=1` 15 runs each, 0
  failures; the ten slow tests 5 runs each, 0 failures; `hack/powertest` and `hack/topology` 5 runs; `internal/*` 50 runs; 0
  failures. A first attempt hit the default 10 minute test timeout, which was a harness error and not a hang. On macOS two full
  runs with `-shuffle=on` were green, and the four `hack/topology` tests passed 3 runs against 2x ncpu busy loops. Test binaries
  cross-compiled for linux/arm64 ran in a container as uid 0 on ext4 and as uid 1000 on a tmpfs: pass. The one organic failure
  found is A1d-6.
- *Leftovers.* A full run with `TMPDIR` on an empty scratch directory left no entries and no stray processes on macOS. On the Linux
  machine only the two entries from a run the auditor killed by hand remained.
- *Timing-dependent tests judged.* The rewrite of the Sync tests replaces wall-clock bounds with gates and is sound. The remaining sleeps are "let it
  park" pauses whose failure mode is a vacuous pass, not a flake. The two 2.1 s sleeps wait longer than the 1 s retention they test, so load only makes them safer.
- *Weakening check.* The golden policy rewrite strengthens the test and the v0.1.0 golden policy still decodes at the tip. The Sync test no longer asserts
  append latency, but a parked `Sync` that held the lock would still deadlock the flush and fail at 30 s. The reserved-name test was replaced by an
  equivalent public-path test.
- *Harnesses.* `hack/skew.sh` was run as a can-it-fail check: tip against tip passes, tip against a copy with the frame CRC table
  switched from IEEE to Castagnoli fails closed (exit 1, work directory kept). The topology, powertest and fuzz-script tests fail on revert of the
  commits that introduced their checks. The fuzz targets' oracles were read in full and run as seed tests only; nothing was fuzzed.
- *Not covered by this auditor.* The darwin-only `TestPruneExitsOneWhenAnUnlinkFails` never runs on Linux, so the CLI exit-1-on-unlink-failure
  mapping is covered only by the macOS leg of CI. `TestNamesThatAreNotUTF8AreEscaped` skips on APFS. Race-only tests are
  vacuous under plain `go test`. The 74 test-only commits were sampled by risk (14 production sites), not mutated one by one.
  `hack/topology` was not run in the container (it needs `go build`) and the full-disk test was skipped everywhere.

**Docs lens (A1d-docs).**

- *Forbidden names.* A search of the tip for the internal product, repo and path names finds only the module path and the code owners
  file (a repository path and a GitHub handle). It finds no home-directory path, no scratch path, no IP address, no hostname and no
  e-mail address other than a test's. The v0.1.0..HEAD commit messages are clean: no attribution lines, and one probe commit (author
  `t`, message `x`) which the supply-chain record documents and a later commit reverts. A1d-14 is the exception that is not a name
  but an operating detail.
- *Commit ledger.* 329 distinct hex tokens in the documents were each resolved with `git cat-file` and checked for ancestry of the tip. Every commit
  that exists but is not an ancestor is labelled rejected, superseded or authored-on-branch in its own context, and each of those
  ledgers names the landed commit.
- *Ledger arithmetic recomputed from raw numbers,* all adding up: the gap audit (17 items; 27 votes; 49 lens verdicts), the live-integration
  record (12 items; 21 votes; 40 lens verdicts), pass L1h (46 lens verdicts; 22 votes), pass A1h (109 and 102 lens verdicts; 33 and 35 votes) and the round-2 findings
  (16 plus 1 fix commits).
- *CHANGELOG coverage.* Of 138 commits in the range that touch non-test, non-hack Go files (recount, Stage 2 L4b-4, 2026-11-04: `git log --format=%H v0.1.0..e2d35d1 -- '*.go' ':!*_test.go' ':!hack' | wc -l` gives 138; the 137 first written here reproduces only with `':!examples'` added as well, and the same command run at the tip of `main` on 2026-11-04, after later commits, gives 141 and 140), every user-visible one maps to an Unreleased entry.
  The ones without an entry are refactors, comment-only commits or behaviour-unchanged by their own text. The v0.1.0 section is untouched.
- *Exported API and reference.* `go doc -short` differs from v0.1.0 only by six new Stats fields, each in the CHANGELOG and the guides. The
  CLI's `reference` output is identical to `docs/reference.md`, which a test pins.
- *Decisions.* 105 rows before and after, none added, 63 amended. The arithmetic and statements of decisions 8, 39, 44, 50, 60, 65, 69 and 81 were
  checked against code and agree. No decision number above 105 is cited outside the audit records. Backticked identifiers in the amended documents exist in the code,
  except three historical references.
- *Documents against the built CLI.* Exit codes, error sentences, `-json` escaping, `ls` on a future-format log, `archive` into a populated non-log directory,
  and the `append` oversize arithmetic all match the documents. Every logger message string in the operator table exists in code, apart from the Open-time marker publish
  raised as A1d-9. The `format.md` additions match `codec.go`, `state.go` and the policy golden. `SECURITY.md`'s limits match the code. The CI
  changes the CHANGELOG describes are present.
- *Other.* No reference figures remain in `review.md` or the decisions listed in A7-15; the 5,370-line count reproduces; the README's "no file over about 600 lines" holds
  (largest 567).
- *Not covered by this auditor.* macOS arm64 only. User guide sections 1 to 6, 8 and 11, `architecture.md` and `introduction.md` were read by their diffs only. The
  threat-model, mutation-hardening, format-compatibility and defects-log records were checked for forbidden names, commit ids and test-name existence, not claim by
  claim. The full suite and the gate were not run.

### Examined and not reported

These were looked at and not filed; each has no reproduction unless stated.

- (code, inferred from reading) `cmd` commit deleting a consumer it did not see in its pre-listing if another process creates it between the
  listing and the open and the commit then fails: a very small window, can drop a consumer file.
- (code, inferred) `Archive`'s destination-sequence gap check can flag a destination that was archived earlier from the same source and skipped by an advanced
  `_archive` offset, giving exit 3 and a "source lost records" message. It is a heuristic and the ambiguity is documented in `Archive`'s godoc.
- (code, inferred) `archive` uses `os.SameFile` instead of `fsx.SameFile` for the source-is-destination check, so the check is skipped under the simulated file system only.
- (code, reported by the auditor) `tail -follow` flushes after every record; documented, affects throughput only.
- (tests, inferred from reading, not tested for equivalence) in `Close`, `if err == nil && l.pendingPublish { publishLocked }` mutated to `if false` survives both suites.
  `flushLocked` already publishes a pending marker with an empty buffer, so the line looks redundant or dead.

## Items

| Id | Kind | Severity | Verdict | Votes | Reproduced | Disposition |
|---|---|---|---|---|---|---|
| A1d-1 | bug | medium | decided in round 1, not voted | none | auditor | deliberately not changed |
| A1d-2 | compat | low | CONFIRMED | 2 of 2 | 2 of 2 | fixed in `96588fb`, FIXED-VERIFIED (7 of 7) |
| A1d-3 | compat | low | CONFIRMED | 2 of 2 | 1 of 2 | fixed in `65b7cb0`, FIXED-VERIFIED (7 of 7) |
| A1d-4 | test | medium | CONFIRMED | 2 of 2 | 2 of 2 | fixed in `d45f1f2`, FIXED-VERIFIED (6 of 6) |
| A1d-5 | test | medium | CONFIRMED | 2 of 2 | 2 of 2 | fixed in `6eb1bdd`, FIXED-VERIFIED (6 of 6) |
| A1d-6 | test | low | CONFIRMED | 2 of 2 | 2 of 2 | fixed in `ae71dba`, FIXED-VERIFIED (6 of 6) |
| A1d-7 | test | low | CONFIRMED | 2 of 2 | 2 of 2 | fixed in `41aaef3`, FIXED-VERIFIED (6 of 6) |
| A1d-8 | doc | low | REFUTED | 1 CONFIRMED, 2 REFUTED | none | refuted, nothing changed |
| A1d-9 | doc | low | CONFIRMED | 2 of 2 | none | fix written (`f2287c7`), rejected, not merged; owner-approved 2026-11-04 as documentation only, applied afresh in the `fix(stage2 A1d-9)` commit (see the item) |
| A1d-10 | doc | low | CONFIRMED | 2 of 2 | none | fixed in `652154f`, FIXED-VERIFIED (6 of 6) |
| A1d-11 | doc | low | decided in A7, not voted | none | none | deliberately not changed |
| A1d-12 | doc | low | CONFIRMED | 2 of 2 | 2 of 2 | fixed in `85fecc9`, FIXED-VERIFIED (6 of 6) |
| A1d-13 | doc | low | decided in A7, not voted | none | none | deliberately not changed |
| A1d-14 | hygiene | low | decided in A7, not voted | none | none | deliberately not changed |

The order below is by id.

### A1d-1: a live retention pass syncs the directory once per deleted segment while holding the log's lock (medium)

**Claim.** A live writer's retention pass does one directory fsync per deleted segment while holding `l.mu`, so a pass over many expired
segments stalls every `Append`, `Read`, `Sync`, `Stat` and `Head` for the whole pass. v0.1.0 did one fsync for the pass.

**Grounds.** `retention.go:363` calls `l.fs.SyncDir(l.dir)` inside the delete loop, once per segment. `Prune` takes `l.mu` at `retention.go:110` and holds it
through `pruneLocked` and `deleteLocked`; the flusher's pass does the same. Reproduction (the auditor's program, verified): 401 expired
sealed segments, `Prune(true)` with a concurrent `Head()` poller. macOS/APFS: v0.1.0 "prune took 31.3 ms; Head() latency max 29.9 ms"; the tip "prune took 2.128 s; Head() latency max
2.128 s". Linux ext4 (loaded by another run): v0.1.0 15.6 ms, the tip 203.9 ms, `Head()` maximum 202.8 ms. Linear in the number of segments, about 0.5 ms
per segment on ext4 and 5 ms on APFS.

**Warrant.** Every public call that needs `l.mu` waits for the whole pass, and a live pass may delete up to a quarter of the sealed
segments (the clock guard in `retention.go`), so a burst of simultaneously expiring segments stalls the writer for seconds. That
conflicts with the scope document's latency statement (decision 50): a record is visible at most `FlushInterval` after `Append`.

**Backing.** `docs/design/scope.md` "Latency (decision 50)"; decisions 61 and 65 (directory-sync ordering) are the intent behind the per-unlink sync.
`CHANGELOG.md` discloses "n directory syncs instead of one" and does not say that they run under the lock.

**Rebuttal.** The per-unlink sync is deliberate: it makes what survives a crash always a suffix (round 1, A1-31, fixed in `e2f7801`; see
[a1-round2-correctness.md](a1-round2-correctness.md) section 18). A steady-state log deletes one or two segments per pass. The offline
`quill prune -force` has no concurrent appender. The scope text speaks of flush-to-visibility latency, not of retention passes.

**Verdict.** Decided in round 1 (decision 65, A1-31), not put to a vote. The auditor's own reproduction is the only evidence. The round-1 record (section 18)
discusses the ordering and does not mention that the syncs run under `l.mu` (inferred from reading that section).

**Reproduction.** As under Grounds: a program that expires 401 sealed segments, calls `Prune(true)` and polls `Head()` from a second goroutine, built once against
v0.1.0 and once against the tip. Output above.

**Disposition.** Deliberately not changed: the earlier decision stands. Recommended answer if the owner wants it reopened (the auditor's proposal): keep
the sync-before-each-unlink order but do not hold the lock across the loop, deleting in chunks and re-validating the sealed set after reacquiring, or, if left
as is, state the stall in decisions 61 and 65, the user guide and the Latency section. Not held in this pass's ledger; it is noted under "Not held, but worth the owner's eye" in
[stage2-owner-decisions.md](stage2-owner-decisions.md).

### A1d-2: Open no longer changes `quill.tail`'s mode when `Config.Perm` differs (low, CONFIRMED, fixed)

**Claim.** Reopening an existing log with a different `Config.Perm` no longer changes `quill.tail`'s mode until the first marker publish; `quill.policy`
changes at once. v0.1.0 changed both at Open. The `Config.Perm` godoc and the user guide still say state files are exactly `Perm`.

**Grounds.** `load.go:459` sets `l.marker = tl` (the decoded marker) and `flush.go:179` publishes only if the marker moved, a publish is pending or a seal
happened (commit `29f5c6c`, L3b-8). Reproduction (verified by the auditor and by both skeptics; programs built once against v0.1.0 and once against the tip): create a
log, close, reopen with `Perm` 0640 and no append. Tip: `tail -rw------- policy -rw-r-----`. v0.1.0: `tail -rw-r----- policy -rw-r-----`. After one `Append` and
flush the tip's tail becomes `-rw-r-----`. The tightening case (a 0640 log reopened with 0600) leaves the tail group-readable until the first publish, even after `Close`
(one skeptic ran it, verified).

**Warrant.** A Reader derives its own mode from `quill.tail` and creates consumer files and `Archive` output with it; a group reader cannot read a 0600 tail. On an idle log
the documented group arrangement (restart the writer with `Perm` 0660) does not take effect, and a hardening restart does not tighten the marker, until traffic arrives.

**Backing.** `config.go` `Perm` godoc and `docs/userguide.md` ("State files are exactly that mode"); decisions 41, 51 and 74. The CHANGELOG entry for L3b-8 says nothing about file modes.

**Rebuttal.** Any append that flushes republishes the marker within one `FlushInterval`, so on a live log the window is short. Segments, consumer lock files and
directories already did not follow a later `Perm` in v0.1.0. No data or format effect.

**Verdict.** CONFIRMED, 2 of 2, both reproduced.

**Disposition.** Fixed in `96588fb`; fix-skeptic FIXED-VERIFIED, 7 of 7 lenses (reported). `Open` now sets an existing `quill.tail` to `Config.Perm` in place
(`setStateMode` in `state.go`, called from `load.go:465`): open read-only, `fstat`, `fchmod` only when the mode differs. No marker rewrite, so no free block is needed and the L3b-8
full-disk restart still works; a chmod failure (a file owned by another account) logs a Warn and does not fail `Open`. An earlier attempt that rewrote the marker was rejected
(it needs a free block, and made "Open writes no file" false). Documentation updated to match: `Config.Perm` and `Config.Logger` godoc, the user guide's Perm and Logger rows and a new row in the
operator table for the Warn, `scope.md`, `architecture.md`, dated notes on decisions 39 and 51, and a CHANGELOG "Fixed" line. Fix-skeptic evidence (reported): without the call the new tests fail
for 0600 to 0640 and 0640 to 0600; a real-filesystem program shows the mode follows in both directions and a Reader-created consumer inherits 0640; the Linux machine passes the new tests; the API
diff is the doc text only. Residual noted by the fix-skeptic, not in this item's scope: `quill.format` is written only at creation and does not follow a changed `Perm`, before or after this
commit, while the user guide still says every state file is exactly `Perm` (the Reader's mode comes from `quill.tail`, so nothing depends on it). Not run by the fix-skeptic: `staticcheck`, `deadcode`, fuzz.

### A1d-3: Problem and SegmentError gained unexported fields (low, CONFIRMED, fixed)

**Claim.** The exported structs `Problem` (field `refused`, `stat.go:148`) and `SegmentError` (field `oversize`, `load.go:52`) gained unexported fields, so `go doc` says "Has unexported fields" for them
and they can no longer be built with an unkeyed composite literal outside the package.

**Grounds.** The `go doc -all .` diff between v0.1.0 and the tip is otherwise additive (Config text, `ConsumerOptions.Rewind`, the six Stats fields, Problem kind `overlap`); no exported
identifier was removed or re-typed (verified by the auditor). Reproduction (second skeptic, verified): the same `main.go` using `quill.Problem{...six values...}` and `quill.SegmentError{...}`
in two scratch modules, one replaced to v0.1.0 and one to the tip: v0.1.0 builds, the tip fails with three compile errors ("implicit assignment to unexported field ... in struct literal").

**Warrant.** Code that compiled against v0.1.0 fails against the tip: a source-compatibility break.

**Backing.** The Go 1 compatibility convention on adding unexported fields.

**Rebuttal.** Both types are produced by quill and returned to callers, never meant to be constructed; `go vet` already flags unkeyed literals of imported structs. The module is v0.y.z. Comparability is
unchanged. The CHANGELOG had no line for these two fields.

**Verdict.** CONFIRMED, 2 of 2 (one reproduced, one read).

**Disposition.** Fixed in `65b7cb0`; fix-skeptic FIXED-VERIFIED, 7 of 7 (reported). One CHANGELOG line under Unreleased naming the two fields, the unkeyed-literal break and the keyed-literal workaround. No code change. The
fix-skeptic re-ran the break against both trees and confirmed the line's text is accurate in both directions. Not checked: the other five exported structs that show "Has unexported fields" were not diffed against v0.1.0.

### A1d-4: the terminal-escape fix is pinned at only some of its sites (medium, CONFIRMED, fixed)

**Claim.** The A3-3 fix (`116726b`, escape disk-derived names in text output) is pinned only at some sites: deleting `safeText` from the `quill ls` success row or from the common error path (`c.errorf`) leaves the whole `cmd/quill` suite green, although raw ESC bytes then reach the terminal.

**Grounds.** Mutants of the tip in a scratch copy: Q, `cmd/quill/read.go:75` `safeText(e.Name())` to `e.Name()`; U, `cmd/quill/verbs.go:51` `safeText` removed. `go test ./cmd/quill` prints ok for both. Reproduction (a scratch test, verified):
(1) a valid log in a directory named with ESC, an OSC title sequence and a clear-screen sequence, then `quill -dir parent ls`: the tip prints the name quoted and escaped, mutant Q prints raw bytes ("raw non-printable U+001B reaches the terminal").
(2) `quill -dir <tmp>/evil<ESC>[2Jgone stat`: the tip prints the quoted message, mutant U prints the raw ESC on stderr. The existing test builds its hostile "log" from `quill.format` and `quill.tail` only, with no segment, so `ls` takes the error row (`read.go:61`, which is pinned: mutant R is killed) and never reaches the success row. No test sends a hostile path through `errorf`. A third mutant, T (escape on the consumers NOTE column), is equivalent: the error text carries no raw file content, so it is not claimed.

**Warrant.** A mutation that reintroduces the exact vulnerability fixed in `116726b` passes every test, though the fix's message lists `ls`, `consumers`, `check` and error output as covered.

**Backing.** Stage 2 item A3-3 and `116726b`'s stated scope; decision 35 (no raw control character in output); the rule that a test must fail on the change it was written for.

**Rebuttal.** The tip itself is correct. The gap matters only if a later edit drops `safeText`. A hostile directory name needs write access to the directory `ls` scans; the `-dir` path in the `errorf` case is typed by the operator, although scripts feed it directory entries.

**Verdict.** CONFIRMED, 2 of 2, both reproduced (both mutants applied to a copy of the tip; both leave the suite green; a new test fails on the mutants and passes on the tip).

**Disposition.** Fixed in `d45f1f2`; fix-skeptic FIXED-VERIFIED, 6 of 6 (reported). Two test functions added to `cmd/quill/names_escape_test.go`: `TestLsSuccessRowEscapesTheLogDirectoryName` (a real log opened with `quill.Open` in each hostile-named directory, then `ls`) and `TestErrorMessageEscapesAHostileDirPath`
(`stat` with `-dir` at an existing non-log directory with a hostile name). Test-only; 56 lines; no production code, no golden file. The fix-skeptic applied the claim's two mutants to the fixed tree: each fails the matching new test for all five hostile names; the unmutated tree passes under `-race`, the Linux machine passes, `go vet` is clean. Not run by the fix-skeptic: `staticcheck`, `deadcode`. The new subtests skip on a file system that refuses the names (neither macOS nor the Linux machine did).

### A1d-5: the Reader's marker-below-first-frame check on the active segment is unpinned (medium, CONFIRMED, fixed)

**Claim.** The Reader's check that the active segment's first frame is not above the marker (`rec.Seq > tl.Seq`, `reader.go:386`) is unpinned: removing it leaves the root and `cmd/quill` suites green, and the Reader then serves a silently truncated log instead of `ErrCorrupt`.

**Grounds.** Mutant G: `reader.go:386` `if ferr != nil || rec.Seq == 0 || rec.Seq > tl.Seq {` to `if ferr != nil || rec.Seq == 0 {`; `go test . ./cmd/quill` passes (the root run skipping the flaky test of A1d-6). The commit that claims to pin it, `786a1c5`
(`TestAReaderCallsAnActiveSegmentThatFailsTheMarkerCorruptNotAnIOError`) has two cases, a file shorter than the marker offset and a flipped byte in the first frame; neither reaches the `> tl.Seq` term (inferred from the green mutant run). Reproduction (verified by the auditor and both skeptics): a log of 43 records
whose active segment is segment 2 holding 41 to 43 at offset 69, with `quill.tail` edited to sequence 40, segment and offset kept (one skeptic: marker 23 lowered to 20 on a simulated file system, and another: sequence 0). The tip: zero records and
`ErrCorrupt: the active segment's first frame does not decode below the marker`, and `Stat` returns `ErrCorrupt`. Mutant G: 40 records (last 38 39 40) with `err == nil`, and `Stat` returns nil. [mutation-hardening.md](mutation-hardening.md) (the line near 631) relies on this check as the backstop that makes another mutant need a hand-damaged tail.

**Warrant.** Guarantee 6 (corruption is never served silently) depends on this branch to tell a marker that disagrees with the bytes from a legitimately stale one. Without the branch no test notices that a damaged marker is served as a shorter, healthy-looking log.

**Backing.** `scope.md` guarantees 6 and 10; decisions 38 and 79.

**Rebuttal.** The state needs a hand-edited or damaged `quill.tail`: the writer never publishes a marker with offset above zero and a sequence below that segment's first frame. This is defence in depth. The mutated Reader still hides records rather than exposes them (guarantee 10). `Reader.Check` returned no error in both builds in the repro, so it does not distinguish them.

**Verdict.** CONFIRMED, 2 of 2, both reproduced.

**Disposition.** Fixed in `6eb1bdd`; fix-skeptic FIXED-VERIFIED, 6 of 6 (reported). One new test, `TestAReaderCallsAMarkerBelowTheActiveSegmentsFirstFrameCorrupt` in `reader_test.go` (+32 lines): 5 records, a roll, 3 more, close, rewrite `quill.tail` with sequence 3 and segment and offset unchanged; expects `ErrCorrupt` from `Read` and `Stat`. It is a separate function and not a third case of the existing table, because that table damages segment bytes and this case damages the marker. Test-only. The fix-skeptic applied mutant G: the test fails with `Read: <nil>; want ErrCorrupt` and the same for `Stat`; with `>` mutated to `>=` other existing reader tests fail, so the pair brackets the term. The test pins the `>` direction at one distance (3 against 6), not every boundary. The Linux machine was not used (the test uses the simulated file system).

### A1d-6: TestFreeBytesTracksRealWrites is flaky when another process frees more than 4 MiB (low, CONFIRMED, fixed)

**Claim.** `TestFreeBytesTracksRealWrites` (`disk_test.go`, added in `ddce2be`) fails on any machine where another process frees more than about 4 MiB on the same volume during its few milliseconds: it compares the free space of a shared volume before and after an 8 MiB write against a 4 to 64 MiB window.

**Grounds.** One organic failure on macOS while two packages ran in parallel: "free space went from 128148168704 to 128145186816 after writing 8388608 bytes: want a fall of 4 MiB to 64 MiB" (a fall of 2.98 MB for an 8 MiB write). Reproduced on purpose: the compiled test run `-test.count=30` beside a loop creating and removing 24 MiB files on the same volume failed 2 of 30, both with a free-space rise; both skeptics reproduced it (10 of 30 and 2 of 60). The Linux ext4 machine, `-race`, 50 runs, and `GOMAXPROCS=1`, 15 runs: no failure (shared with other jobs whose deletes were smaller than the window).

**Warrant.** The assertion reads before and after from a volume the test does not own, so any concurrent net delete over about 4 MiB (another package's temp directory cleanup under `go test ./...`, another job, a build-cache trim) fails it with no defect in `freeBytes`.

**Backing.** The test's own comment ("the bounds are loose because other processes use the same disk"); `hack/gate.sh` runs packages in parallel under `go test -race ./...`; the gate's header says a commit gate must not go red with no change in the repository.

**Rebuttal.** A quiet machine never trips it: 0 failures in 80 Linux runs. The deliberate churn is heavier than ordinary traffic. A tmpfs or a dedicated file system would not trip it.

**Verdict.** CONFIRMED, 2 of 2, both reproduced.

**Disposition.** Fixed in `ae71dba`; fix-skeptic FIXED-VERIFIED, 6 of 6 (reported). The test repeats the before, write, after measurement up to five times and passes on the first attempt inside the window; test-only, one file. Under the same 24 MiB churn the fix-skeptic measured 14 failures in 100 runs before and 0 in 100 after. Mutations of `availBytes` (the block-size factor dropped, a x100 factor) fail on every attempt, so the retry does not mask a wrong unit. `staticcheck` passes (reported); `deadcode` was not run. Residual (inferred): churn that outlasts all five short attempts would still fail it; the unit and field check stays in `TestAvailBytesIsAvailableBlocksTimesBlockSize`.

### A1d-7: decision 91's 4 KiB first-frame probe is pinned by nothing (low, CONFIRMED, fixed)

**Claim.** Decision 91's amendment (a sealed or active segment's first-frame probe reads 4 KiB, not 64 KiB; commit `ea4ca7c`) is not pinned: the test written for it passes without the change.

**Grounds.** The per-commit revert of `ea4ca7c` still passes `TestAReaderReadsSealedSegmentsWhoseFirstFrameIsAroundAndAboveTheProbeSize`. Mutant P at the tip, `reader.go:252` `&window{r: f, size: size, chunk: firstFrameChunk}` to `&window{r: f, size: size}` (and, in the skeptics' form, at both call sites, `:252` and `:378`): the whole root-package suite passes. The test compares record bytes only, and the commit message says "Results are identical". Reproduced by both skeptics (a perl edit of both sites; vet clean, tests ok).

**Warrant.** The test is a correctness guard for first frames around and above 4 KiB, but it cannot detect the optimisation being lost, so the numbered decision's claim and the benchmark rationale behind it have no test.

**Backing.** `docs/design/decisions.md` decision 91, amendment of L1-50; the rule that a test added with a change should fail without it.

**Rebuttal.** The change is performance only with identical results, so a result-based test cannot fail by design. The boundary test still guards the riskier 4 KiB path. The simulator counts operations (`read_overhead_test.go`), so a pin is possible but was judged not worth it.

**Verdict.** CONFIRMED, 2 of 2, both reproduced. One skeptic noted that the proposal's mention of the `Inject` hook is slightly off, since `Inject` receives no read size.

**Disposition.** Fixed in `41aaef3`; fix-skeptic FIXED-VERIFIED, 6 of 6 (reported). `TestAReaderProbesASegmentsFirstFrameWithASmallRead` (+103 lines in `reader_test.go`) wraps the `fsx` seam through the Reader's existing test option to record `ReadAt` lengths per segment and asserts that the first-frame probe of every segment over 4 KiB reads exactly `firstFrameChunk` bytes and that a 12 KiB first frame takes exactly one more read. Test-only. The fix-skeptic removed the chunk at the sealed site, at the active site and at both: the new test fails in each case ("first probe read 6720 bytes, want 4096"), and the older boundary test still passes, which confirms the gap. `gofmt`, `go vet` (both OSes), `staticcheck` and `deadcode` pass; 20 runs under `-race` show no flake. Residual: the test asserts exact read counts, so a legitimate future change that adds a read to the probe needs the test updated.

### A1d-8: silent-degradation.md still describes the A5-40 fix as an unconditional pendingPublish (low, REFUTED)

**Claim.** `docs/audits/silent-degradation.md` (near line 935) says Open's forced write "is now `l.pendingPublish = true` (`load.go:432`)"; `29f5c6c` (L3b-8) removed that line, and the A5-40 section carries no dated note saying so.

**Grounds.** The section header says "fixed 906d49c". At the tip `grep` finds `pendingPublish = true` only in `flush.go` (twice) and `retention.go`. `load.go:454-459` now sets `l.marker, l.markerAt = tl, l.cfg.clock()`. `906d49c` is an ancestor of the tip. The A5-40 behaviour (the marker age counts from Open) still holds; the auditor ran the `MarkerAge` and clean-open tests and they passed. The auditor's reading of the live-integration and owner-decision records matches (they name the old line as the L3b-8 defect).

**Warrant.** The audit lens requires "fixed in <sha>" to name a fix that is in the tree; the described mechanism was replaced.

**Backing.** The lens rule; decisions 39 and 81.

**Rebuttal.** The ledger is a dated record and the behaviour it protects is unchanged. A5-40 is not wrong about what `906d49c` did.

**Verdict.** REFUTED, 2 to 1. Reason, in one line: a dated ledger scopes its line anchors to the tree it was committed with (its own header says so, and other anchors in it are stale in the same way), and the supersession is already recorded in the L3b-8 ledgers and decisions 39, 60, 65 and 81; no document requires an old ledger to be amended after a later change. The dissenting vote held it was a real but minor gap.

**Disposition.** Refuted, nothing changed. The optional one-line dated note proposed by the auditor was not written.

### A1d-9: the logger documentation's list of records Open gives itself is incomplete (low, CONFIRMED, fix rejected)

**Claim.** The `Config.Logger` rule in `docs/userguide.md` (the `Logger` row, near line 242) and decision 50 lists four records `Open` gives itself (a cut newest segment, a `quill.lock` it cannot tighten, an ahead consumer, a failing first
`quill.policy` write) as the exceptions to "every record above runs with the lock held". After L3b-8 `Open` also gives the marker-publish Warn ("cannot publish quill.tail...") and, if `publishLocked` rotates, a rotation Warn; like the other four, they run before `Open` returns with no lock held. By the list's own wording they would run under the lock.

**Grounds.** `load.go:459-472` calls `publishLocked` from `load()`, which `Open` calls before any `l.mu.Lock`. On failure the background-state path (`log.go:488-493`) calls `logf` with the "marker publish" Warn. `open_fulldisk_test.go:112-140` asserts the Warn right after `Open` returns. `CHANGELOG.md` says `Config.Logger` gets the marker-publish Warn. The `config.go` Logger doc has a "from Open:" bullet listing the same four. No handler probing the lock state was run (inferred from reading `log.go`, `load.go` and `flush.go`).

**Warrant.** A record that has the property but is not in the list makes the list, and the sentence about every other record, untrue for it.

**Backing.** Decision 50's own rule, restated at L1-10 (2026-11-03) as exact; the A1-62 commit message says the rule was stated 2026-11-02 after a fix-skeptic found it untrue of `Open`. L3b-8 (`29f5c6c`) came after and added a record without updating the list.

**Rebuttal.** `config.go:126-127` says generically "except those Open gives itself", which covers the new Warn; only the parenthetical lists are incomplete. A handler written to the documented rule is safe either way. Documentation only.

**Verdict.** CONFIRMED, 2 of 2, neither reproduced (read only).

**Disposition.** Fix written, rejected by the fix-skeptic, not merged: held for the owner (see [stage2-owner-decisions.md](stage2-owner-decisions.md), section "Held after pass A1d"). A first attempt (a longer list adding the new records) was rejected because the enumeration would go stale again (reported by the fix summary). The second, `f2287c7`, states the exception by where a record is written rather than as a list, in the user guide, `scope.md`, decision 50 and the `config.go` doc. The fix-skeptic's objection: the dated A1d-9 note on decision 50 lists "the consumer-passed Warn" among the records `Open` writes through the seal its recovery makes, and the code cannot produce it (`l.offsets` is nil until `log.go:217`, after `load()` has run `publishLocked` and its seal; the fix-skeptic ran a protected consumer at offset 1 plus a seal at `Open` and got only the "deleted segment" Info). The same sentence also still enumerates `Open`'s own warnings and omits the `quill.tail` mode Warn added by `96588fb`. `f2287c7` is not an ancestor of the tip (verified).

**Resolution (2026-11-04, owner-approved, documentation only).** Not `f2287c7`: a fresh change that states the exception by where a record is written and lists no record. The records `Open` itself writes run on the caller's goroutine before `Open` returns, before there is a `Log`, with no lock held; every other record runs with the lock held; the handler rule (do not call the `Log`, do not block) still applies to both. The user guide's `Logger` row, `scope.md`'s `Logger` row, decision 50 (with a dated A1d-9 note) and the `Config.Logger` doc say so. Before the wording was written, `Open` was run in a scratch copy under a recording handler that tried the log's lock at each record (verified; a control record after `Open` saw it held). Records seen, all with the lock free: the cut-newest-segment Warn; the `quill.lock` untightenable Warn; the `quill.tail` mode Warn; the ahead-consumer Warn; the first `quill.policy` write Warn; the "cannot publish quill.tail" Warn (injected ENOSPC on the temp file, stale marker); the "cannot create the next segment" Warn (rotation failing during recovery); and, for a seal during recovery (a stale marker, a reopen with a smaller `SegmentBytes`), the "deleted segment" Info (`MaxRecords` 3) and the "cannot read free space" Warn (`MinFreeBytes` unreachable). The consumer-passed Warn did not appear (a protected consumer at offset 1 with a seal at `Open` gave only the "deleted segment" Info), so it is not named.

### A1d-10: the CHANGELOG pool entry says Open now allocates about 1 MB, the ledger says 2.1 MB (low, CONFIRMED, fixed)

**Claim.** The CHANGELOG pool entry says `Open` of 787 x 64 KiB "allocated about 59 MB and now about 1 MB". The ledger the figure came from (`verification-optimization.md`, L1-49) says 59.1 MB to 2.1 MB.

**Grounds.** `CHANGELOG.md:38`; `verification-optimization.md:322`. The auditor's own benchmark (a throwaway, a log of about 760 segments of 64 KiB, 1000-byte payloads, no retention, `-benchmem`, macOS arm64, whole `Open`): 1,948,033 and 2,302,536 B/op (about 16,000 allocations). The fixture is not the ledger's: 760 against 787 segments. No committed benchmark guards the saving (the ledger says so), so neither figure can be re-run from the repository.

**Warrant.** "About 1 MB" is half of either measurement and is supported by neither.

**Backing.** The rule that the Unreleased section is accurate; the L1-49 row is the measured source.

**Rebuttal.** "About 1 MB" might mean the pooled buffers alone, and the entry does say the pool holds about 1 MiB per processor. The skeptics rejected this reading: the sentence compares `Open`'s total before and after.

**Verdict.** CONFIRMED, 2 of 2, neither reproduced (read against the ledger; the auditor's measurement is reported).

**Disposition.** Fixed in `652154f`; fix-skeptic FIXED-VERIFIED, 6 of 6 (reported). One word: "now about 1 MB" to "now about 2 MB". The fix-skeptic measured three runs on its own fixture (787 segments of 64 KiB, real disk, whole `Open`): 2,652,118, 2,585,315 and 2,430,902 B/op, which supports "about 2 MB" over "about 1 MB". The `gofmt`, `vet` and `-race` checks pass.

### A1d-11: the README layout table does not list the stage 2 harnesses (low, decided in A7)

**Claim.** The README Layout table lists `hack/gate.sh`, `hack/fuzz.sh` and `hack/powertest/` only. `hack/skew.sh`, `hack/skewopen/`, `hack/topology/` and `hack/topology.sh` are absent, and the topology harness is described nowhere outside the audit records and its own header.

**Grounds.** `git ls-files hack` lists the missing files. `CONTRIBUTING.md` documents `skew.sh` and `skewopen`. A search of README, CONTRIBUTING, the user guide, architecture, CHANGELOG and the CI configuration for "topology" returns nothing. Commit `a63bd6b` says the gate now runs the harness's deterministic fault self-test through `go test ./...`.

**Warrant.** The Layout table is the map of what each path is for.

**Backing.** The lens requirement that the README still describes the code after stage 2.

**Rebuttal.** These are test and operator tools, not library or CLI surface; the table never claimed to list every file.

**Verdict.** Prior-refuted (A7-7, [gap-audit.md](gap-audit.md) near lines 444-459); not put to a vote.

**Disposition.** Deliberately not changed: the earlier pass decided the Layout table is a summary and not an inventory. The auditor's proposal if the owner reverses that: add rows for the skew check and the topology harness.

### A1d-12: the CHANGELOG A7-11 entry cites a 845-line log.go that no release had (low, CONFIRMED, fixed)

**Claim.** The CHANGELOG entry says `Config` moved "from `log.go` (845 lines)". At v0.1.0 `log.go` was 555 lines; 845 is a mid-stage-2 intermediate.

**Grounds.** `CHANGELOG.md:15`. `git show v0.1.0:log.go | wc -l` gives 555; the same for the parent of the A7-11 move (`f47d6f0^`) gives 845. Root non-test files now: `log.go` 567, `reader.go` 531, `load.go` 525. Reproduced by both skeptics with read-only git commands.

**Warrant.** The Unreleased section is read against v0.1.0; the 845 figure implies a file-size violation that never shipped. The "no file over about 600 lines" property held at v0.1.0 and holds now.

**Backing.** The rule that Unreleased lists changes accurately; decision 20 and the 600-line rule in `CONTRIBUTING.md`.

**Rebuttal.** The entry was true on the day A7-11 landed ([gap-audit.md](gap-audit.md) near line 221), and is labelled "no behaviour change".

**Verdict.** CONFIRMED, 2 of 2, both reproduced.

**Disposition.** Fixed in `85fecc9`; fix-skeptic FIXED-VERIFIED, 6 of 6 (reported). The entry now says `log.go` was 555 lines at v0.1.0 and grew to 845 during the audit, and that the move keeps every library file under about 600 lines. The fix-skeptic re-derived both figures and checked current maxima. Objection, not held against it: a changelog read against v0.1.0 now mentions an intermediate no user saw.

### A1d-13: the executive summary says v0.2.0 is "the same code after the full audit sequence" (low, decided in A7)

**Claim.** `docs/executive-summary.md:46` says "v0.2.0 is the same code after the full audit sequence", though the CHANGELOG Unreleased section (about 100 entries) lists new exit codes, new refusals, changed semantics and new Stats fields.

**Grounds.** The sentence at `executive-summary.md:45-47`. The auditor confirmed several CHANGELOG entries by running the built CLI at the tip (`quill read 100` exits 2; `roll` and `recover` on a non-log exit 1; `archive -to <populated non-log dir>` exits 1 and leaves it untouched; `tail -n 08` is eight records). Six new exported Stats fields.

**Warrant.** "The same code" is false for a release whose CHANGELOG changes exit codes and tightens refusals.

**Backing.** The lens requirement that the README and guides describe the code; `scope.md:494`; the user guide's "Upgrading and rolling back" says the CHANGELOG is the authority on differences.

**Rebuttal.** A forward-looking plan sentence written at v0.1.0, readable as "the same code base, audited and corrected"; the release number is not fixed.

**Verdict.** Prior-refuted (A7-21, [gap-audit.md](gap-audit.md) near lines 456-470); not put to a vote.

**Disposition.** Deliberately not changed: the earlier pass treated the sentence as a roadmap. If the owner reverses that, the auditor's wording: "v0.2.0 is v0.1.0 after the full audit sequence: the same format and guarantees, with the findings fixed (see the CHANGELOG for the behaviour changes)", or drop the sentence.

### A1d-14: audit-prompt.md carries operating details of the private test machine (low, hygiene, decided in A7)

**Claim.** `docs/design/audit-prompt.md`, an in-tree file for a repository headed to public release, still carries operator details of the private test infrastructure: a benchmark lock file path in a home directory, a home-relative image directory, a note that the test machine has privileged access without a password and can install packages, a helper that mounts a small scratch file system, and references to "other agents" and "the remote runner". None is a host name or an address.

**Grounds.** `audit-prompt.md` lines 39, 229, 241, 280 and 284 (the strings are not repeated here, since repeating them in this record would commit them a second time). A search of the tip for home-relative paths finds these as the only ones outside code comments. Several lines of the file already use placeholders, so it was partly sanitised. `supply-chain.md:81` claims that tracked files contain no host names, addresses, e-mail addresses or paths; that record predates the file (`faa8eaf`).

**Warrant.** Committed text should carry no internal path or file name; if the repository goes public those lines publish how the maintainers' test machine is set up.

**Backing.** The ground rules for committed text, and the `supply-chain.md` hygiene claim. Decision 55 says the stage 2 record is part of the release.

**Rebuttal.** No host name, address or credential is present; the paths and the mount point are generic conventions (the CI workflow uses the same mount point); the machine is anonymous; the file is a prompt for a future session and arguably a record. The auditor rated it hygiene, not a leak.

**Verdict.** Prior-deliberate (A7-16, [gap-audit.md](gap-audit.md) near lines 530-548); not put to a vote.

**Disposition.** Deliberately not changed: the earlier pass decided to keep the file as a record. Recommended answer if the owner reverses that, before the repository is public: replace the machine-specific lines with neutral placeholders ("a benchmark lock file", "a scratch directory on the test machine"), or move the file out of the public tree into the private build records.

## What this pass did not reach

- No fuzzing, no `staticcheck`, no `deadcode` and no fuzz step of the gate by the code and docs auditors; of the fix-skeptics, only those of A1d-6 (`staticcheck`) and A1d-7 (both) ran them, and the others report that they did not.
- One platform per auditor: the code and docs auditors ran on macOS arm64 only (the code auditor's Linux race run did not finish). The tests auditor covered Linux x86-64, a Linux arm64 container as root and as an unprivileged user, and macOS.
- Harnesses, tests and the user guide's untouched sections were skimmed or read by diff, as listed under each lens.
- The 74 test-only commits and five of the 12 build-failing reverts were not mutated individually.
- A1d-1's stall was measured on two file systems on shared machines; the figures are indicative, not a benchmark.
