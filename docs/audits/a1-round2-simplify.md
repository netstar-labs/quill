# A1 adversarial audit, round 2: simplify

Date 2026-10-29, against HEAD 1da37fc. Dimension report for the simplify lens: 9 items, ranked by
what is at stake (a wrong comment that invites a harmful deletion, then dead surface, then line
count). The top line and the coverage statement are in
[a1-round2-findings.md](a1-round2-findings.md); the lens found no item that needed an owner
decision. Two of its nine items were decided in round 1 and are kept with their round-1 reference.

Votes: C confirmed, P plausible, R refuted. Reproduction for a simplify item means the skeptic
applied the proposed change to a scratch copy, ran the suite and, where the claim is about cost, the
benchmark. Anchors are to the audited tree 1da37fc. Claims not self-evident are labelled verified,
reported or inferred. Every number measured here is from macOS arm64 (M2 Pro) unless stated.

Tally: 6 fixed, 1 refuted, 2 round-1-decided.

| # | id | anchor | sev | verdict | votes | disposition |
|---|----|--------|-----|---------|-------|-------------|
| 1 | A1-1 | state.go:91-158; internal/strictjson | medium | prior-deliberate | - | round-1-decided (decision 105) |
| 2 | A1-8 | internal/fsx/os.go:16-17; fs.go:63 | low | CONFIRMED | CC | fixed 8f89e0d |
| 3 | A1-7 | internal/atomicfile/atomicfile.go:89; fsx/fs.go:56-57; disk.go:5-7 | low | CONFIRMED | CC | fixed af99624 |
| 4 | A1-3 | read.go:164-268 | low | CONFIRMED | CC | fixed 1b1ff46 |
| 5 | A1-4 | state.go:38-53, 264-276 | low | CONFIRMED | CC | fixed a2113b5 |
| 6 | A1-6 | stat.go:42-63 | low | CONFIRMED | CC | fixed f0cdcfa |
| 7 | A1-12 | read.go:293; load.go:312; flush.go:267-274 | low | CONFIRMED | CC | fixed becd44e |
| 8 | A1-5 | atomicfile.go:37-53; fsx/os.go:60-75 | low | prior-deliberate | - | round-1-decided (allocation pass) |
| 9 | A1-10 | read.go:145 | low | REFUTED | CRRR | refuted |

## 1. A1-1: quill.tail is decoded by two parsers; make it canonical-only and delete strictjson (medium, prior-deliberate)

**Claim.** `quill.tail` is decoded by a byte-exact fast path and by a 143-line `internal/strictjson`
pipeline plus `decodeTailSlow`. Making `decodeTail` accept only the canonical bytes `appendTail`
writes would delete `decodeTailSlow`, `internal/strictjson` and its tests: about 170 non-test and 130
test lines.

**Grounds.** state.go:91-96 (`decodeTail`), :116-136 (`decodeTailFast`), :138-158 (`decodeTailSlow`);
internal/strictjson/strictjson.go:1-143 has exactly one non-test caller, state.go:147. An experiment
in a scratch copy (decodeTail returns an errBadState-wrapped error whenever the fast path says no,
the slow path and the strictjson import deleted, `TestDecodeTailFastAgreesWithSlow` deleted) passed
the root, cmd, atomicfile and fsx tests on macOS; state.go 370 to 345 lines. No other test pins a
non-canonical marker being accepted; `TestAppendTailMatchesJSON` pins the writer's bytes.

**Warrant.** The slow path is reached only for input the fast path rejects, and the only producer of
the marker is `appendTail`, so for damage the slow path can only change the error text. It is stricter
only for non-canonical but valid JSON (whitespace, reordered fields, a missing newline, case-variant
keys); everything format.md:103-105 says must be refused stays refused.

**Backing.** Decision 105 (decisions.md:113) created the split for allocation cost and kept the slow
path only so non-canonical verdicts stay as they were; decisions 8 and 48; docs/format.md:109-112
(canonical form). v0.1.0 is not public, so there is no installed base; after release the accepted set
becomes a compatibility surface.

**Rebuttal.** Not equivalent for non-canonical valid JSON: `{"seq":1, "seg":0, ...}` or a file without
the trailing newline opens today and would fail with ErrCorrupt. An operator who restored quill.tail
through a reformatting tool or hand-repaired it would be affected; decision 105 chose to preserve that
verdict. If another writer of this format exists that emits `json.Marshal` output without the newline,
it breaks; the reference implementation's marker bytes were not inspected (unchecked).

**Verdict.** prior-deliberate: decided in round 1 (decision 105; the allocation pass reader-15 and the
CHANGELOG). No round-2 votes.

**Disposition.** Deliberately not changed in round 1 (decision 105). Re-raised by a fresh auditor. If the
owner wants the 170 lines, the decision to take is "one accepted byte form for quill.tail", with the
amended text for decision 105 and the format.md State files paragraph; otherwise keep the slow path and
settle A1-2 (held, [a1-round2-docs.md](a1-round2-docs.md)).

## 2. A1-8: two comments say the fsx flag constants equal the os package's values on every platform (low, CONFIRMED, fixed)

**Claim.** internal/fsx/os.go:16-17 ("The values are equal on every supported platform") and
internal/fsx/fs.go:63 ("Flags, with the values of the os package") are false on darwin, so `osFlags` is
load-bearing and its comment is wrong.

**Grounds.** Verified by a program on darwin/arm64: `os.O_CREATE` is 0x200 against fsx 0x40 and
`os.O_EXCL` is 0x800 against fsx 0x80 (the fsx values are the Linux ones; Linux x86-64 confirmed by the
fix-skeptic). `osFlags` (os.go:18-33) is the translation and cannot be dropped.

**Warrant.** A reader who trusts the comment might delete the mapping; passing 0x40 and 0x80 straight to
`os.OpenFile` on macOS would set unrelated bits.

**Backing.** Decision 19 (Linux and macOS only).

**Rebuttal.** Documentation only; the code is correct, and only a Linux-only CI would never show the
mismatch.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by both skeptics: a Go program printing `os.O_WRONLY`, `O_RDWR`, `O_CREATE`
and `O_EXCL` on darwin/arm64 gives 0x1 0x2 0x200 0x800.

**Disposition.** Fixed in commit 8f89e0d64cca (comment-only, on branch stage2/A1-8): the constants are
documented as the Linux values, macOS differing, which is why `osFlags` translates them. Fix-skeptic
FIXED-VERIFIED: the call sites (log.go:421, flush.go:154, atomicfile.go:63) go through `osFlags`;
O_RDONLY, O_WRONLY and O_RDWR are 0, 1, 2 on both. Residual: the new fs.go wording ("not the os
package's on every platform") is a little clumsy.

## 3. A1-7: internal surface used only by tests or never varied (low, CONFIRMED, fixed)

**Claim.** `atomicfile.WriteSyncDir`, the `fsx.File.Name` interface method, and the `freeBytes` function
variable whose comment promises a seam nobody uses.

**Grounds.** `deadcode` without `-test` (the gate's pinned v0.52.0) reports, outside the crash
simulator, only internal/atomicfile/atomicfile.go:89 `WriteSyncDir`; its callers are
atomicfile_test.go:111 and :138. `fsx.File.Name` (fs.go:56-57) has no production caller (its only caller
is crash_test.go:27). disk.go:5-7 says `freeBytes` is "a variable, so tests can inject a value", but
nothing assigns it; the test seam is `Config.statfs` (log.go:68).

**Warrant.** Production never reaches these; removing them cannot change library or CLI behaviour.

**Backing.** Decision 70 (the gate runs `deadcode -test`, which is why test-only code is invisible to it);
the least-code ladder.

**Rebuttal.** `WriteSyncDir` is mentioned in the package doc as the option for callers who need a durable
rename and the crash tests use it to show what a directory fsync changes; deleting it loses that
demonstration unless the test calls Write then SyncDir. fsx is internal, so no external API is affected.
`Log.fsyncs` is the same kind of test-only counter and is kept (the group-commit test depends on it).

**Verdict.** CONFIRMED by 2 of 2 votes (static; neither skeptic ran deadcode).

**Disposition.** Fixed in commit af99624f9d49. `WriteSyncDir` and `File.Name` (interface and
`simFile.Name`) are removed, `freeBytes` is a plain function with an accurate comment, and the two crash
tests call `Write` then `fsys.SyncDir` through a test-local `writeDurable`; the package doc now says to call
`fsx.FS.SyncDir` after `Write`. Fix-skeptic FIXED-VERIFIED: deadcode v0.52.0 without `-test` now reports only
the simulator; with `-test` it is clean; staticcheck v0.8.2 clean; `GOOS=linux GOARCH=arm64 go vet` clean.
Residual: `mustWrite` in crash_test.go lost the file name from its failure message.

## 4. A1-3 (with A1-15): `scanSource` repeats the same "gap after delivered records is left for the next call" tail six times (low, CONFIRMED, fixed)

**Claim.** A local closure removes about 11 lines and puts the `count > 0 && !gapAfter` rule in one place.

**Grounds.** read.go:164-167, 174-177, 184-187, 193-200, 251-257 and 265-268 each repeat `if count > 0 &&
!gapAfter { return next, nil }` followed by `return next, <err>`. The scratch change `fail := func(err
error) (cursor, error) {...}` reduced read.go from 363 to 352 lines; `go test ./...` passes on macOS and
the Linux box; allocations are unchanged (BenchmarkReadTail 106 allocs/op before and after, ReadSealed
113, ScanAll 47; the AllocsPerRun guard at bench_test.go:404 passes).

**Warrant.** `count` and `next` are read when the closure is called, so each site sees the same values as
the inline check; the `errors.As` rewrite of a deleted segment's GapError (read.go:196-199) is pure, so
moving it before the count check changes nothing observable.

**Backing.** Decision 38 and the comment at read.go:136-144; decisions 101 and 103 for the gap shapes.

**Rebuttal.** Some reviewers prefer explicit returns, and a future site that must return a non-nil error
even after delivery would bypass the closure.

**Verdict.** CONFIRMED by 2 of 2 votes. Both note the neighbouring ErrCorrupt return at read.go:260 must stay
explicit, and the active-segment ErrCorrupt return at :251 must go through `fail`; one skeptic's first
prototype missed that (their mistake, corrected in the verified patch).

**Disposition.** Fixed in commit 1b1ff466f74f: a local `fail` closure after `next := cursor{seq: cur}`, six
sites converted, read.go 363 to 354 lines (net -9). Fix-skeptic FIXED-VERIFIED: mutating the closure to
`if false && ...` fails four gap tests. Residual: the net saving is 9 lines, not 11, and error values are now
built before the suppression check (error paths only); staticcheck and deadcode were not run.

## 5. A1-4: `parseUint` and `parseSegmentName` hand-roll digit loops that `strconv.ParseUint` already performs (low, CONFIRMED, fixed)

**Claim.** Keeping only the leading-zero rule saves 12 lines.

**Grounds.** state.go:38-53 and :264-276. `ParseUint(s, 10, 64)` already rejects the empty string, signs,
underscores (only allowed for base 0), non-ASCII digits and overflow. A differential test over 3,000,000
random strings (digits, signs, `_`, `x`, Arabic-Indic digits, space, NUL, `e`, `.`, lengths 0 to 21),
old against new for both functions, found zero mismatches (581,782 accepted segment names).

**Warrant.** Both old and new accept exactly ASCII digits with no leading zero (`parseUint`) or exactly ten
ASCII digits plus `.log` (`parseSegmentName`); ten digits always fit a uint64.

**Backing.** Decision 96 (decisions.md:104) specifies the CLI's plain-decimal parsing as base-10
`ParseUint`, which cmd/quill/read.go:214-218 does.

**Rebuttal.** The hand-rolled form makes "digits only" visible without knowing ParseUint's rules; a future
Go change for base 10 would be caught by the golden and fuzz tests.

**Verdict.** CONFIRMED by 2 of 2 votes (each ran their own differential, 2,000,000 and 3,000,000 strings,
zero mismatches).

**Disposition.** Fixed in commit a2113b5096f2: both functions use `strconv.ParseUint(s, 10, 64)` and
`parseUint` keeps only the leading-zero rule (state.go 375 to 363 lines). Fix-skeptic FIXED-VERIFIED:
changing the base to 0 fails `TestParseUintIsPlainDecimalOnly` and the new segment-name test
(`1_0`, `000000_001.log`, `0x00000001.log`). Residual: the Linux box and staticcheck were not run for this one.

## 6. A1-6: `Log.Stat` takes a full `segView` for three fields, then locks again, so Stats can mix two moments (low, CONFIRMED, fixed)

**Claim.** Reading everything under one lock is simpler and gives one snapshot.

**Grounds.** stat.go:42-63: `l.view()` (read.go:311-343) locks, bumps `a.readers`, copies the sealed ref slice and
builds a closure; `release()` is called, the view is used for head, first and holes, and only then is `l.mu`
taken again (stat.go:50) to sum sealed sizes. `l.firstRetainedLocked()` (retention.go:149-164) equals
`view.first`. The scratch change (lock once, `firstRetainedLocked()`) passes `go test ./...` on macOS and Linux;
Stat 22 to 21 lines.

**Warrant.** Between the two sections the flusher can flush, rotate or delete, so Head, First and Records can be
older than Segments, Bytes and Dropped (inferred, not reproduced: an 8 s loop of 2,970 snapshots found none
inconsistent, because the window is narrow).

**Backing.** scope.md describes Stat as one set of counts; retention.go:262 (Stat snapshots alias `l.holes`,
preserved).

**Rebuttal.** Stat on a live writer is inherently racy against the next flush and the inconsistency is cosmetic;
if a future change makes `segView.first` diverge from `firstRetainedLocked`, two paths must be kept in step.

**Verdict.** CONFIRMED by 2 of 2 votes (neither reproduced the two-moment mix).

**Disposition.** Fixed in commit f0cdcfa781d7: `Log.Stat` takes `l.mu` once, returns ErrClosed if closed, and builds
the view fields (head, `firstRetainedLocked()`, holes) under that lock; no view allocation. Fix-skeptic
FIXED-VERIFIED: replacing `first` with `head + 1` fails six retention tests. Residual: no test asserts the one-lock
property, and nothing pins `segView.first` and `firstRetainedLocked` as agreeing.

## 7. A1-12: three redundant constructs (low, CONFIRMED, fixed)

**Claim.** `s := s` at read.go:293, `max(runtime.GOMAXPROCS(0), 1)` at load.go:312, and a prune Ticker that is
created, stopped and has its channel field set to nil under NoMaintenance (flush.go:267-274).

**Grounds.** go.mod:3 declares go 1.27, so each loop iteration has its own `s` (load.go:315 relies on the same
semantics); `GOMAXPROCS(0)` returns at least 1; PruneInterval is always valid by then, so `NewTicker` cannot panic.

**Warrant.** The first two are no-ops; the third works but mutates a Ticker's exported field where a nil channel
would express the same `select` behaviour.

**Backing.** Go per-iteration loop variables; the `runtime.GOMAXPROCS` documentation.

**Rebuttal.** None for the first two; for the ticker the existing form is correct and tested, so the change is
taste.

**Verdict.** CONFIRMED by 2 of 2 votes (read, not run).

**Disposition.** Fixed in commit becd44efa7fa: the copy and the `max` removed; the prune ticker built only when
maintenance is on, with a nil `<-chan time.Time` otherwise. Fix-skeptic FIXED-VERIFIED: making the prune case fire
under NoMaintenance fails `TestTheBackgroundPassRunsUnlessMaintenanceIsOff`. Residual: only a Linux vet and the
macOS race run; the full gate was not run.

## 8. A1-5: two hand-optimised helpers save less than the noise of the syscalls around them (low, prior-deliberate)

**Claim.** `atomicfile.tempName` (and its pinning test) and the raw-syscall `fsx.OS.Rename` could be `filepath.Join`
and `os.Rename`, removing about 32 non-test lines and 10 test lines.

**Grounds.** internal/atomicfile/atomicfile.go:37-53 and `TestTempNameMatchesJoin` (atomicfile_test.go:191-200).
Measured by the auditor on macOS arm64: `tempName` 30.9 ns and 1 allocation against 93.7 ns and 2 for
`filepath.Join` plus hex, while a real `atomicfile.Write` of the 40-byte marker costs 5,259,225 ns, so the saving
is about 63 ns of 5.26 ms (about 0.001 percent). internal/fsx/os.go:60-75 reimplements `os.Rename` to skip one
Lstat and to retry EINTR; `os.Rename` 4.1 to 7.9 ms against `syscall.Rename` 4.0 to 6.1 ms over three runs,
indistinguishable.

**Warrant, backing.** Stdlib before hand-rolled; decision 40 puts the cost of `writeState` at temp, fsync, rename.
Neither helper is part of a decision or guarantee.

**Rebuttal.** The author tracks allocations per flush (the comment "one allocation instead of three"); on Linux
ext4 fsync is faster but still hundreds of microseconds (not measured). For Rename: rename(2) returns EISDIR where
`os.Rename` synthesises EEXIST and does not retry EINTR; no caller-visible difference was found, but the EINTR retry
is real behaviour (inferred; local filesystems rarely return EINTR from rename).

**Verdict.** prior-deliberate: decided in the round-1 allocation pass (write-2 and write-3, commits cfcf076 and
09f2e3d; benchmark-baselines.md:139-142). No round-2 votes.

**Disposition.** Deliberately not changed in round 1; re-raised. Only the Rename half is worth revisiting, and only if
the EINTR retry is not wanted.

## 9. A1-10: `scanSource`'s `copyPayloads` and `gapAfter` are always opposites (low, REFUTED)

**Claim.** At all three call sites `copyPayloads` and `gapAfter` are opposite, and `maxBytes` is zero exactly when
`gapAfter` is set, so three parameters carry one bit; one `batch bool` would remove two parameters of an
eight-parameter function.

**Grounds.** read.go:145 (signature); callers read.go:109 (`readFrom`: `maxReadBytes()`, copy, no gap-after),
iter.go:57 (`records`: same), read.go:120 (`scanFrom`: 0, no copy, gap-after); a grep finds no other caller.

**Warrant.** The function is unexported with a closed set of callers.

**Backing.** The doc comment at read.go:136-144 describes the two modes (Read and Scan) as the only intent.

**Rebuttal.** A future zero-copy bounded caller (a streaming CLI) needs the knobs split again; the saving is about two
lines plus readability.

**Verdict.** REFUTED (votes C, R, R, R). The premise is true (verified by grep by every skeptic) but the three knobs
govern different things (payload lifetime, the result-size bound, and when a gap error is reported) and the doc comment
documents each; the `records` iterator uses the Read-mode gap behaviour though it is a callback caller, so "a callback
has the records" does not determine `gapAfter`; a single flag would hide a semantic choice. The saving is about two
lines. The confirming skeptic accepted the premise but called it a low-value simplification.

**Disposition.** Not changed: refuted; the knobs are distinct documented contracts that current callers happen to pair.
