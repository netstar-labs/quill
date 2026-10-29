# A1 adversarial audit: correctness

Date 2026-10-29, against commit 1f4152d. Dimension C (correctness) of the A1 audit: every
candidate, each skeptic's verdict and reason, the reproduction, and the fix the synthesis
recommends. Reproductions were run in throwaway copies of the repository under the session
scratchpad; no repository file was edited by the audit. Anchors re-read at synthesis time.
Ranked by severity, then by reachability.

Verdicts: 12 CONFIRMED (correctness-27 by one CONFIRMED and one PLAUSIBLE vote). Two further
correctness bugs were raised under other lenses and are cross-referenced here: simplify-1
(Reader nil dereference on a seq-0 first frame, audit-simplify.md item 1) and docs-33
(offline verbs overwrite quill.policy, audit-docs.md item 1).

| # | id | anchor | severity | verdict | recommendation |
|---|----|--------|----------|---------|----------------|
| 1 | correctness-21 | retention.go:219 | BUG | CONFIRMED | apply now (snapshot variant) |
| 2 | correctness-22 | log.go:468 | BUG | CONFIRMED | apply now |
| 3 | correctness-23 | reader.go:323 | BUG | CONFIRMED | apply now |
| 4 | correctness-24 | cmd/quill/verbs.go:700 | BUG | CONFIRMED | apply now (absorbs simplify-3) |
| 5 | correctness-25 | retention.go:245 | MINOR | CONFIRMED | apply now |
| 6 | correctness-26 | reader.go:199 | MINOR | CONFIRMED | apply now |
| 7 | correctness-31 | load.go:453 | MINOR (semantic) | CONFIRMED | apply now |
| 8 | correctness-29 | load.go:418 | MINOR | CONFIRMED | apply now |
| 9 | correctness-27 | reader.go:280 | MINOR (semantic) | CONFIRMED | owner decision |
| 10 | correctness-28 | cmd/quill/verbs.go:478 | MINOR (semantic) | CONFIRMED | owner decision |
| 11 | correctness-32 | cmd/quill/verbs.go:756 | MINOR (semantic) | CONFIRMED | owner decision, with docs-33 |
| 12 | correctness-30 | read.go:268 | PERF | CONFIRMED | deferred |

## 1. correctness-21: deleteLocked indexes a stale n after publishLocked's nested rotation and caps shorten l.sealed (BUG)

**Anchors.** retention.go:219 `if err := l.publishLocked(); err != nil`, :227 `s := l.sealed[i]`,
:243 `l.sealed = l.sealed[n:]`; flush.go:104 rotates when
`l.active.durable >= l.cfg.SegmentBytes && l.active.frames > 0`, and :141 runs
`l.deleteLocked(l.capsLocked(0), offsets)` after a successful rotation.

**Claim.** `n` is computed by `pruneLocked` against the pre-rotation slice. When a rotation
failed earlier (decision 60 d: the log keeps appending past SegmentBytes with `pendingPublish`
set) and a cap is configured, the `publishLocked` inside `deleteLocked` rotates, the nested
delete removes a prefix of `l.sealed`, and the outer loop then deletes `n` more from the new
front, or indexes past the end and panics. The self-review (self-review:35) checked only that the
recursion is bounded, not that `n` stays valid.

**Skeptics (3, all CONFIRMED).** All three re-read the anchors and reproduced both outcomes.
Reachability: a segment-create failure leaves `pendingPublish` with the active segment growing
(flush.go:106-110, :129); neither `Prune` (retention.go:76-89) nor the background prune tick
(flush.go:266-275) retries the publish before computing `n`, and the age-rotation branch is
guarded by `!l.pendingPublish` (retention.go:103). The nested delete must remove at least one
segment, so a cap (MaxBytes, MinFreeBytes or MaxRecords) must be set. The third skeptic added two
sharper variants: records written after the clock advance (not expired) are deleted, and a
ProtectOn consumer committed at 16 loses its unread segment with `Stat.Dropped = 1` while the
cap required only one deletion, breaking decisions 24/57 (caps override protection only as far
as the cap requires). Both proposed fixes were applied in separate copies: all repro tests pass
and `GOWORK=off go test ./... -count=1` is green under each. The snapshot variant is preferable:
the `publishLocked(runCaps bool)` flag skips the caps at the one seal triggered from inside
`deleteLocked`, a wording deviation from decision 24 ("checked at every segment seal").

**Reproduction.** Config: SegmentBytes 4096 (or the sim's 60), MaxBytes 16384, Retention 1h,
SegmentMaxAge -1, NoMaintenance; fill 3 sealed segments; `s.Inject` returns an error for
`fsx.OpCreate` on the next segment while 4 more segments' worth are appended and flushed (each
`Flush` returns the injected error, `pendingPublish = true`); clear the injection; advance the
clock 2h; `Prune(true)`.
`GOWORK=off go test -run TestAuditDeleteLockedReentrancy -v -count=1 .`:
`before prune: sealed=3 active.durable=680 pendingPublish=true head=28` then
`PANIC in Prune: runtime error: index out of range [0] with length 0`, stack
`deleteLocked retention.go:227 <- pruneLocked retention.go:145 <- Prune retention.go:88`.
Over-delete variant (one extra segment, MaxBytes 400, records 13..16 written fresh):
`after prune: err=<nil> sealed=0 stat={Head:16 First:17 ...}`, `Read(13): records 13 to 16 are unavailable`;
expected `sealed=[3]`, `first=13`. Original suite: green.

**Fix.** In `deleteLocked`, snapshot `targets` (the `*segment` pointers in `l.sealed[:n]`)
before `publishLocked`, then re-derive `n` as the length of the leading run of `l.sealed` whose
pointers are still in `targets`, and delete only those. Add both reproductions, plus the
protected-consumer variant, to retention_test.go (the file is at 615 lines; put them in a new
retention_reentrancy_test.go).

## 2. correctness-22: Close nils active.f under the lock while a pinned reader uses it without the lock (BUG)

**Anchors.** log.go:467-471 closes and nils `l.active.f` with no look at `l.active.readers`;
read.go:265-266 (`view` pins `a.readers`), read.go:289 (the active segRef's `open` returns `a.f`
later, outside the lock), load.go:121 (`w.r.ReadAt` on the nil interface). Rotation honours the
pin (flush.go:165-169; release closes at zero, read.go:299-303); Close does not. scope.md:77-78
promises a `*Log` is safe for concurrent use.

**Skeptics (3, all CONFIRMED).** All reproduced the nil dereference; one also under `-race`.
No existing test covers it (append_test.go:196 exercises rotation only; the Close tests at
append_test.go:263 and reader_test.go:102 start the read after Close). No decision exempts Close
(43 covers retiring at rotation). `Seek` (seek.go:71) and the archiver (archive.go:134) take the
same `open()` path. With the proposed fix the reproduction returns 4 records, `OpenFiles()` is 0
after the reader's release, the full suite, `-race` over the Close/Reader/Iterator/Follow/Rotation
tests, and `go vet` all pass. Nuance: when a reader is pinned, the eventual `f.Close()` error is
dropped by the release path, as rotation already does. Decision 53 is untouched (Close still
flushes, publishes, fsyncs the directory and releases the lock file).

**Reproduction.** One sealed segment (append 3, flush, Roll) and an active with 2 flushed
frames; `s.Inject` sleeps 300 ms on `OpLstat` of the sealed segment's path (`l.sealed[0].path`,
id 0); goroutine `l.Read(1, 100)` with recover; main sleeps 100 ms, `l.Close()`.
`GOWORK=off go test -count=1 -run TestAuditCloseUnderReader -v .`:
`Close err=<nil>; reader recs=0 err=<nil> panic=runtime error: invalid memory address or nil pointer dereference`;
stack `(*window).bytes load.go:121 <- frameAt load.go:136 <- walk load.go:170 <- scanSource read.go:202 <- readFrom read.go:104 <- (*Log).Read read.go:16`.

**Fix.** In Close: `if l.active.readers > 0 { l.active.retired = true } else { close f; f = nil }`;
the existing release path then closes it when the last reader leaves. Add the reproduction as a
test and run it under `-race`.

## 3. correctness-23: Reader.view reads s.index/s.last unsynchronised against scan's once.Do write (BUG)

**Anchors.** reader.go:247 (`s.index, s.last, s.frames, s.lastAt = ...` inside `s.once.Do`,
:231) and reader.go:323 (`if s.index != nil && s.last < last`) plus :331/:333 (`index: s.index`);
`view` never calls `Do`, and `segment()` releases `r.mu` (:218) before returning the shared entry.

**Skeptics (3, all CONFIRMED).** All three reproduced "WARNING: DATA RACE" under `-race` with
the write at reader.go:247 and the read at :323, in 1-2 of 5 runs per invocation; one also saw
the sibling race at read.go:192 (`slices.BinarySearchFunc` over anchors written by `walk` at
load.go:183), same root cause. The Reader doc (reader.go:33) and scope.md:78 promise concurrent
safety; CI runs `-race` (.github/workflows/ci.yml:39) but no test runs two Reads on one Reader
through the sealed-index path. Decision 63 says the index is built once per segment id and says
nothing about publication. Two corrections: the torn-read consequence (a spurious GapError) is
inferred, not observed (0 wrong outcomes in 35,644 fresh-Reader rounds); and a torn `last == 0`
would not skip the segment at :320 but append a segRef with `last = 0` and raise a spurious hole
from the next contiguity check, still a gap over intact data. A mutex variant cannot deadlock:
`scan` runs from `scanSource` after `view` returned and `r.mu` is never held across either.

**Reproduction.** Log with >= 3 sealed segments; one `Reader`; 8 goroutines each calling
`r.Read(7, 3)`. `GOWORK=off go test -race -count=5 -run TestAuditReaderIndexRace .`:
`WARNING: DATA RACE / Write at ... by goroutine N: (*readerSegment).scan.1 reader.go:247 ... / Previous read at ... by goroutine M: (*Reader).view reader.go:323`,
`testing.go:1867: race detected during execution of test`, FAIL. Without `-race`: ok over 20 runs.

**Fix.** Publish the scan result atomically: an immutable `scanResult{index, last, frames, lastAt, err}`
stored in an `atomic.Pointer` by `scan` and loaded once by `view` and `ensureIndex`; or set the
four fields under `r.mu` in `scan` and read them under `r.mu` in `view`. The atomic pointer
also closes the read.go:192 sibling race (the slice is reached only through the published
pointer). Add the reproduction as a test; it is meaningful only under `-race`, which the gate runs.

## 4. correctness-24: prune/commit parse numbers with Sscanf %d, which silently truncates "1M", "1e6", "0x10" (BUG)

**Anchors.** cmd/quill/verbs.go:700 (`-max-bytes`, `-min-free-bytes`, `-max-records`), :592
(`commit -offset`), :778 (`@unix`); capsLocked at retention.go:205-208 applies `MaxRecords = 1`
literally and caps override protection (retention.go:144, :175-176).

**Skeptics (3, all CONFIRMED).** Two ran the built CLI end to end: a log of 4 segments,
`quill prune -dir d -max-records 1e6` printed `segments 4 -> 1, bytes 152 -> 26, first 1 -> 10`
and exited 0, every sealed segment gone; `quill commit -dir d -name c -offset 1e3` on a
20-record log printed `c 1` and exited 0, the consumer's offset read back as 1 (a full rewind;
a correct parse of 1000 would have been refused as out of range); `-offset 10abc` committed 10.
No guard: the package's strict `parseUint` (state.go:200-211) is unexported and not used by the
CLI; cli_test.go uses only well-formed numbers and never passes a cap flag; decision 75 argues
for canonical decimals, decision 86 fixes shapes only, docs/reference.md:177-179 does not
specify number syntax. Strict parsing also rejects `" 7"` and `+5`, a CLI-only narrowing with no
documented contract.

**Reproduction.** `GOWORK=off go test ./cmd/quill -run 'TestReproPruneMaxRecords1e6|TestReproCommitOffset1e3' -v`
(tests drive `run` directly on a log built with `quill.Open` in a temp dir):
`prune -max-records 1e6: exit=0 stdout="segments 4 -> 1, bytes 840 -> 210, first 1 -> 31\n"`,
`after: head=40 first=31 segments=1 records=10`, `BUG REPRODUCED`;
`commit -offset 1e3: exit=0 stdout="c 1\n"`, `consumer c offset after commit: 1`, `BUG REPRODUCED`.

**Fix.** `strconv.ParseUint(c.offset, 10, 64)` at :592, `strconv.ParseInt(f.s, 10, 64)` at :700,
`strconv.ParseInt(s[1:], 10, 64)` at :778, each returning `c.usage(...)` (exit 2) on error; add
`"strconv"` to the import block. Unit suffixes (K/M/G) are a separate owner choice, listed in
audit-followups.md. Tests: `commit -offset 1e3` and `-offset 12abc` exit 2 and leave the
consumer file unchanged; `prune -max-records 1e6` exits 2 and deletes nothing; `-since @0`,
`-offset 4` keep passing. This absorbs simplify-3.

## 5. correctness-25: deleteLocked compacts l.holes in place while readers' views alias the same array (MINOR)

**Anchors.** retention.go:244-251 (`kept := l.holes[:0]` ... `l.holes = kept`); read.go:267
(`v := &segView{holes: l.holes, ...}` under the lock, no copy); read.go:77-81 (`locate` ranges
over `v.holes` after `view()` released the lock); stat.go:29 hands the same backing array to the
caller as `Stats.Holes`.

**Skeptic (CONFIRMED).** `-race` reproduced: `Read at ... (*segView).locate() read.go:77` vs
`Previous write at ... (*Log).deleteLocked() retention.go:248`. A second, deterministic
consequence the finding did not mention: a caller's `Stat().Holes` slice is rewritten under it by
the next compacting prune (`before={5,8}, after={13,16}, len still 2`). The torn-Hole GapError is
an inference, not reproduced. With `make`, `l.holes` is only ever reassigned (load.go:409,
retention.go:251) and never written in place after publication, which fixes both.

**Reproduction.** Sim fs, SegmentBytes 60, six segments of four records, close, remove segments
1 and 3 (holes {5,8}, {13,16}), reopen with MaxRecords 16 and NoMaintenance; `st := l.Stat()`;
four goroutines loop `l.Read(10, 1)`; after 2 ms `l.Prune(true)` (deletes segment 0, hole {5,8}
dropped, {13,16} shifted into slot 0). `GOWORK=off go test -race -count=10 -run TestZZHolesAliasPrune .`:
`WARNING: DATA RACE` with the two anchors above, `race detected during execution of test`.

**Fix.** `kept := make([]Hole, 0, len(l.holes))` at retention.go:245. Add the `-race` test and
an assertion that `Stat().Holes` taken before a prune is unchanged after it.

## 6. correctness-26: Reader caches any openSegment error as a permanent bad segment (MINOR)

**Anchors.** reader.go:198-200 (`s.bad = err` for any error, including a raw I/O error that
load.go:68-71 returns unwrapped), :212-218 (stored in `r.segs`), :223-227 (`forget`, called only
from a good segRef's `open` closure at :346, :351), :300-303 (a bad entry becomes
`pendingHole = true; continue` on every later view).

**Skeptic (CONFIRMED).** Nothing else touches `r.segs` (reader.go:41, 81, 191, 213, 216, 225);
a bad entry never gets a segRef so `forget` can never run for it. Decision 83 says an entry "is
dropped when a read finds the file gone or changed in size", which by construction cannot happen
for a bad entry: a gap in the design, not a choice. The only precedent, `Log.loadSealed`
(load.go:441-445), is a one-shot load, not a long-lived cache. No test injects an open failure on
a Reader path. EMFILE is reachable because `openSegment`'s Lstat consumes no descriptor but
`OpenFile` does.

**Reproduction.** Sim log, SegmentBytes 200, 30 records (segment 0 sealed, 1 active); open a
Reader; `s.Inject` returns a one-off error for `OpOpen` on segment 0; `r.Read(1, 5)`; clear the
injection; read three more times; `Stat`. `GOWORK=off go test -run TestReproTransientOpenErrorIsCachedAsHole -v -count=1 .`:
first read `records 1 to 30 are unavailable`; every later read `n=0 err=records 1 to 30 are unavailable isGap=true`;
`Stat: Head:30 First:31 Holes:[{From:1 To:30}]`. Control: a fresh Reader on the same directory
reads 5 records, so only the poisoned cache is at fault.

**Fix.** In `segment()`, cache the entry only when `openSegment`'s error is a `*SegmentError`
(not a regular file, over the size limit) or the first frame is bad; for any other error
(`*fs.PathError`, EMFILE, EACCES, transient ENOENT) return the entry without inserting it into
`r.segs`, so the next view re-examines the file. The call still reports the segment as a hole,
which decision 79 ("an I/O error met while scanning a segment is returned as that error") does
not forbid for the listing step. Add the reproduction as a test asserting the reads after the
injection is cleared succeed.

## 7. correctness-31: a damaged sealed segment's bytes are invisible to MaxBytes and Stat.Bytes (MINOR, semantic)

**Anchors.** load.go:453-456 (`s.bad = err; return s` before the only assignment of `s.size`
at :456); retention.go:182 (`bytes += s.size`); stat.go:53 (`s.Bytes += seg.size`); by contrast
`Reader.Stat` (stat.go:81-86) uses `Lstat(...).Size()`, so writer and reader already disagree on
`Bytes` for the same directory.

**Skeptics (2, both CONFIRMED).** Decision 32 and scope.md:246 define `Stat().Bytes` as "the
sum of segment sizes" with no exclusion; scope.md:239/347 promise the disk bound that the
uncounted file exceeds by its size. T17 (retention_test.go:474-531) compares `Bytes` with the
on-disk files, but its only hole is a removed segment, which has no bytes. Mitigation: once any
deletion is needed the loop at retention.go:185-188 walks over the bad segment and deletes it in
the prefix, so the overshoot is bounded by the bad segment's size. The reader side already sets
`s.size = size` before verification (reader.go:202), an in-tree precedent for the ordering.
Residual gap: when `openSegment` itself fails (:442-444), size stays 0 under the fix too.

**Reproduction.** By code reading (both skeptics); the scenario (SegmentBytes 1 GiB, MaxBytes
4 GiB, a flipped byte in a 1 GiB sealed segment: 5 GiB on disk while capsLocked believes 4 GiB)
follows from load.go:453-456 and retention.go:182 and is not executed.

**Fix.** In `loadSealed`, assign `s.size = size` right after `openSegment` succeeds, before
verification. Test: damage a sealed segment, reopen, assert `Log.Stat().Bytes` equals the sum
of the segment files and equals `Reader.Stat().Bytes`; with MaxBytes set, assert the damaged
segment is evicted when the counted total crosses the cap. Classified apply-now because it
restores decision 32 and guarantee 11 rather than changing them.

## 8. correctness-29: the active segment's descriptor leaks when load fails at its final directory fsync (MINOR)

**Anchors.** load.go:409 (`l.active = a`), :414-417 (close `f` on publish failure), :418
(`return l.fs.SyncDir(l.dir)` with `f` still open); log.go:300-303 (Open's load-failure path
closes only the lock); log.go:312 (the one place Open closes `l.active.f` on failure, the
protected-consumer path).

**Skeptic (CONFIRMED).** The real backend returns fsync errors from `SyncDir`
(internal/fsx/os.go:61-70); decision 60(e) says a directory fsync error is fatal and says
nothing about descriptor disposal at open. The only `OpenFiles` leak check is for iterators
(iter_test.go:184-210); recovery_test.go:334 injects `OpSyncDir` only during Flush. Refinement:
if `publishLocked` at :414 rotated, `l.active.f` may no longer be `f`, so close `l.active.f`.

**Reproduction.** Sim log: append 3, flush, close (`OpenFiles() == 0`); `s.Inject` returns
`syscall.EIO` for `fsx.OpSyncDir`; `Open` three times. `GOWORK=off go test -run TestZZOpenSyncDirFailureLeaksActiveDescriptor -v .`:
`after failed Open #1: open handles 1 / #2: 2 / #3: 3`, FAIL. With the fix: `0 / 0 / 0`, PASS.

**Fix.** `if err := l.fs.SyncDir(l.dir); err != nil { _ = l.active.f.Close(); l.active.f = nil; return err }; return nil`
at load.go:418 (closing `l.active.f` rather than `f` covers the rotated case, since
`rotateLocked` already closed the old file when no reader pinned it). Add the reproduction.

## 9. correctness-27: Reader.view returns a raw ENOENT when the marker's segment, or a listed sealed segment, is deleted between listing and opening (MINOR, semantic)

**Anchors.** reader.go:279-283 (one `openSegment` of the marker's segment, no retry, error
returned raw; `readTail` at :121-139 retries); the sealed `open` closure at :343-347 forgets the
entry and returns the raw error, which read.go:181-187 passes to the caller and iter.go:68-71
ends a Follow on; the writer's seal path is one locked step: rotate (flush.go:105), marker
(:114), `deleteLocked(capsLocked(0), ...)` (:141).

**Skeptics (PLAUSIBLE, CONFIRMED).** Both agree on the mechanism and both correct the scenario:
MaxBytes cannot delete the just-sealed segment unless that segment alone exceeds MaxBytes
(MaxBytes >= 2 x SegmentBytes, log.go:155; capsLocked deletes oldest-first, retention.go:184-187),
which needs an oversized segment (decision 60 d/78 overshoot); MinFreeBytes (:194-197) and
MaxRecords (:205-208) have no such guard, and any ordinary prune in the window between the
ReadDir (reader.go:265) and the open hits the sealed case. Narrower than claimed: on a segment's
first read `ensureIndex` (read.go:160-166) already maps a scan failure to a GapError; the raw
`*fs.PathError` escapes only after this Reader cached the segment's index, i.e. in a Follow or
repeated-read loop. The second skeptic reproduced both escapes deterministically with an `fsx.FS`
wrapper that unlinks the file on the open's Lstat, then applied a sketch of both fix parts: vet
clean, full suite green, the injected deletion returns `GapError{5,8}`; a genuinely missing
active segment still returns ErrNotExist after ~5 ms. Refinements: errors.go:47 documents
`GapError.From` as the requested offset, so mapping in read.go:181-187 with `From: cur` is
tidier than `From: s.first` in the closure; archive.go:134 and seek.go:71 would then see a
GapError for a vanished segment, matching archive.go's documented return and decision 59.

**Reproduction.** Log with SegmentBytes 60, three sealed segments of four records plus one
active record; `OpenReader` through a wrapper that removes `0000000001.log` right before its Nth
Lstat; warm `Read(5, 10)`; arm the wrapper; `Read(5, 10)`.
`GOWORK=off go test -run TestRepro -v -count=1 .`:
`first Read(5): err=lstat /log/0000000001.log: file does not exist isGap=false isNotExist=true`;
`second Read(5): err=quill: records 5 to 8 are unavailable isGap(5,8)=true`. Marker's segment
removed at its first Lstat: the same raw error.

**Why owner decision.** Guarantee 14 and decision 11 call for a GapError, so the direction is
settled, but the fix changes the error type three callers see (Read/Follow, Seek, Archive) and
part 1 adds a bounded retry with sleeps to `view`. Recommended: approve part 2 (map
`fs.ErrNotExist` from a sealed segment's `open` to a `*GapError` with `From: cur`, after
`forget`) as the fix; treat part 1 (re-read the tail and rebuild on ENOENT for the marker's
segment, bounded like `readTail`) as optional, since it only narrows a few-syscall window.

## 10. correctness-28: quill append -sync=false exits 0 and prints the sequence when the deferred Close, the only flush, fails (MINOR, semantic)

**Anchors.** cmd/quill/verbs.go:478 `defer l.Close()` (error discarded), :497 `AppendBatch`,
:502-506 Sync only when `c.sync` (flag at :84, default true), :507-508 print and exit 0;
`Close` (log.go:447-479) does the only write via `flushLocked` (:460), whose error it returns.
cmdRecover (:761) does check `Close`.

**Skeptics (2, both CONFIRMED).** One reproduced against the real binary on a 2 MB HFS+ image:
with 1-3 free 4K blocks `quill append -sync=false <20 KB>` printed `2`, exited 0, and `stat`
still showed head 1 with the segment at 16 bytes (record lost); with 4 free blocks the data
landed but the marker publish failed, still exit 0. On a disk that is already full, Open itself
fails at `writePolicy` (exit 1), so the window is "room for Open's small files but not the
flush". The other reproduced under the simulator (`ENOSPC` on `OpWrite`: `Close returned: no
space left on device`, head 0 after reopen) and applied the proposed fix: vet and the whole
`cmd/quill` suite pass; no test runs `-sync=false`. Refinement from the second skeptic: with
`-sync=true`, a Close failure in decision 60 classes (d)/(e) after a successful Sync would now
exit 1 without printing the sequence although the records are durable (decision 78: "a failed
state does not unsay a durable fact").

**Reproduction.** As above; the simulator form:
`GOWORK=off go test -run TestThrowawayCloseReturnsENOSPCAndRecordIsLost -v .`:
`AppendBatch returned seq 1, err nil`, `Close returned: no space left on device`,
`after reopen: head=0 (record 1 written? false)`.

**Why owner decision.** The fix changes a documented verb's exit-code contract (decision 53's
classes) and has a real choice for the `-sync=true` case. Recommended shape: call `l.Close()`
explicitly before printing; if it fails and `Sync` had not run, print only the error and exit 1;
if it fails after a successful `Sync`, print `last` to stdout, the error to stderr, and exit 1;
keep the defer as a safety net (the second Close returns `ErrClosed`, harmless).

## 11. correctness-32: offline verbs open with the default 8 MiB SegmentBytes and silently rotate a larger active segment at load (MINOR, semantic)

**Anchors.** cmd/quill/verbs.go:474, :728, :756 (`c.writer(quill.Config{Retention: -1})`);
`writer` (:287-298) adds only Dir and NoMaintenance; log.go:105-107 (default 8 MiB); load.go:414
(`publishLocked` at the end of load); flush.go:104 (size rotation); NoMaintenance gates only the
prune ticker (flush.go:257-260). Only cmdPrune reads the policy (:681).

**Skeptics (2, both CONFIRMED).** Both reproduced: a writer with SegmentBytes 16 MiB and a
9 MiB active segment, closed; `quill recover` prints `recovered: head 9, first 1, 2 segments`,
the directory gains an empty `0000000001.log`, `quill.tail` names seg 1, and `quill.policy` now
reads `segment_bytes: 8388608` (the docs-33 overwrite). Decision 79 makes rotation-at-open
deliberate for an operator who chose a smaller size, not for a tool substituting a default;
decision 68's own rationale ("offline tools otherwise ran with default retention") is the same
hazard, fixed for retention and not for SegmentBytes. Fix option A (an `offlineConfig()` that
takes SegmentBytes from `ReadPolicy`, falling back to MaxSegmentBytes) applied in a copy: vet and
the `cmd/quill` suite pass, recover then prints `1 segments`. Option B (skip size rotation at
load under NoMaintenance) would not cover `quill append`, whose flush-time publish still rotates.

**Reproduction.** `GOWORK=off go test -run TestZZOfflineOpenRotatesAtDefaultSegmentBytes -v -count=1 .`:
`segments before offline open: 1, after: 2 (head 9)`; policy `segment_bytes` 16777216 before,
8388608 after. CLI: `quill recover -dir <log>` -> `recovered: head 9, first 1, 2 segments`.

**Why owner decision.** This and docs-33 are one question: what Config do the offline writable
verbs open with. Seeding `c.writer()` from the whole policy would make the policy's caps live
during an offline append's flush-time seal (`publishLocked` -> `capsLocked`), which decision 68
("never prunes") forbids. Recommended: (1) offline verbs keep `Retention: -1` and no caps, take
`SegmentBytes` (and `MaxPayloadBytes`) from `ReadPolicy` when it parses, else MaxSegmentBytes;
(2) `Open` skips `writePolicy` when `cfg.NoMaintenance` is set (log.go:305), so the file keeps the
real writer's settings; (3) document in decision 68 that `roll` rotates regardless. The owner
should confirm (2), since it redefines `NoMaintenance` as "offline tool" (its current doc at
log.go:61-63 says only what it disables).

## 12. correctness-30: Log.view rebuilds the sealed-segment view on every read, under the log mutex (PERF)

**Anchors.** read.go:259-306 (`view` under `l.mu`, :260-261); :268-285 allocate a `segRef` and a
closure per good sealed segment; callers: every read path through `scanSource` (read.go:136),
`Stat` (stat.go:42), `consumerStore.first` (consumer.go:287), `OffsetForTime` (seek.go:37);
Append (append.go:24) and the flusher (flush.go:17, :179, :211) wait on the same mutex.

**Skeptic (CONFIRMED).** Measured independently: `Read(head, 1)` costs 9 allocs at 0 sealed,
33 at 10, 417 at 200, 2019 at 1000, the finding's figures exactly (~2 per sealed segment).
Nothing bounds the sealed count by default; no decision describes a per-call rebuild as
deliberate. The cache is sound because a sealed segment's fields are set only at load
(load.go:443, :453) and `l.sealed` is replaced wholesale at flush.go:163 and retention.go:243.
Severity caveat: at the default 8 MiB segment, 1000 sealed segments is an 8 GiB log and the hold
is ~40 µs, a modest cost.

**Reproduction.** `GOWORK=off go test -run TestSkepticViewAllocs -v -count=1 .`:
`sealed=0 view allocs/op=5 Read(head,1) allocs/op=9`, `sealed=10 ... 33`, `sealed=200 ... 417`,
`sealed=1000 ... 2019`.

**Why deferred.** Not a defect; a cache with invalidation on the hot read path, larger than a
local fix and best landed as its own change with an allocation benchmark. Plan when taken: a
`[]*segRef` for the sealed segments rebuilt only where `l.sealed` changes (`rotateLocked`,
`deleteLocked`), shared across views; `view` then allocates the `segView` and the active ref only.
