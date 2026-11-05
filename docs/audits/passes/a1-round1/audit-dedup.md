# A1 adversarial audit: dedup

Date 2026-10-29, against HEAD b52b746 (the source is identical to 1f4152d; only docs/audits
changed in between). Dimension B (duplication) of the A1 audit: every candidate, each skeptic's
verdict and reason, and the recommended disposition. Every anchor was re-read from the tree at
synthesis time. Ranked by what the duplicate puts at risk.

Verdicts: 7 CONFIRMED (the finder's numbering skips dedup-15; nothing was raised under it).
dedup-13 is the same item as simplify-4 and is counted once across the audit.

| # | id | anchor | severity | votes | verdict | recommendation |
|---|----|--------|----------|-------|---------|----------------|
| 1 | dedup-19 | load.go:364 | DEDUP (semantic) | 2 | CONFIRMED | owner decision: approve the helper |
| 2 | dedup-17 | archive.go:151 | DEDUP | 1 | CONFIRMED | apply now, after simplify-9 |
| 3 | dedup-16 | load.go:329 | DEDUP | 1 | CONFIRMED | apply now |
| 4 | dedup-14 | cmd/quill/verbs.go:353 | DEDUP | 1 | CONFIRMED | apply now |
| 5 | dedup-13 | load.go:60 | DEDUP | 1 | CONFIRMED | apply now (= simplify-4) |
| 6 | dedup-18 | consumer.go:80 | SIMPLIFY | 1 | CONFIRMED | apply now |
| 7 | dedup-20 | internal/fsx/crash.go:209 | DEDUP | 1 | CONFIRMED | apply now |

## 1. dedup-19: hole and overlap bookkeeping is written four times, with two differing edge guards (CONFIRMED, semantic)

**Anchors.** load.go:364-376 (sealed), load.go:380-391 (active), reader.go:326-340 (sealed),
reader.go:363-376 (active). The overlap text is "inside the range of the one before" at
load.go:367 and :384 and "inside the one before" at reader.go:330 and :375. reader.go:369 guards
the active hole with `if activeFirst > from`; load.go:373, :390 and reader.go:337 do not, so a bad
segment before a good one whose first frame is 1 would append `Hole{1,0}` there. The tail
steps after the loops differ by design and stay in their callers: load.go:402-405 refuses
`hasFrames && head < tl.Seq` with `ErrCorrupt` (decision 39); reader.go:380-384 reports a hole
up to `tl.Seq` (decision 11). `grep "inside the" *_test.go` hits only t.Fatalf text, so neither
string is pinned.

**Skeptics (2, both CONFIRMED).** Both verified the four sites, the strings and the guards, and
that the helper would compute every pinned hole identically (recovery_test.go:106 `Hole{1,5}`,
:273 `Hole{5,8}`, reader_test.go:200 and archive_test.go:145 `Hole{9,12}`, retention_test.go:528).
Both corrected sub-claim (3): only the reader's active site is hole-then-overlap (reader.go:364
before :373); the reader's sealed site is overlap-then-hole like load; the orders are
equivalent because the overlap error discards the view. Both showed the `Hole{1,0}` edge is
unreachable from quill's own output: segment 0 is created at log creation (log.go:405), every
rotation path requires frames in the active segment (flush.go:104, retention.go:97/101), and an
empty sealed segment is bad (load.go:210-212), so the segment holding seq 1 is always the
lowest id; an empty hole would be inert anyway (read.go:77-79 never matches it, stat.go:32-35
subtracts zero). No decision mandates two independent copies (decisions 38, 63, 83 describe
semantics, not shape).

**Why owner decision.** Unifying the text changes one of the two overlap strings and adding the
guard changes `Stat().Holes` on hand-crafted input; the finder rated the risk medium. The
self-review (docs/audits/passes/self-review-recovery-and-reader.md, "What the audit should look at
first", item 2) names the reader's hole bookkeeping as the place most in need of a hostile eye,
and the writer's copy gets none of that scrutiny today. Recommended answer: approve. Fix: a
`contiguity` struct in a new holes.go (`prevLast uint64; havePrev, pendingHole bool; holes
[]Hole`) with `skip()` and `next(what string, first, last uint64) error` doing the overlap check
(one message), the guarded hole append and the state advance; the four sites become one call
each, keeping load's `_ = f.Close()`/`loadErr` handling and the reader's `activeFile.Close()`,
the empty-range `continue` at reader.go:320-322 and the scanned-end adjustment at :323-325 in
place; the two tail steps stay in their callers. Test: the existing hole assertions listed
above plus one test per site for the overlap error (`errors.Is(err, ErrCorrupt)`).

## 2. dedup-17: `copySegment` re-implements `atomicfile.Write`'s temp/fsync/rename protocol (CONFIRMED)

**Anchors.** archive.go:151-187 against internal/atomicfile/atomicfile.go:31-56, step by step:
8 random bytes into `tempPrefix+hex` (:151-155 vs :31-35; state.go:32 sets `tempPrefix =
atomicfile.TempPrefix`), `OpenFile(tmp, O_RDWR|O_CREATE|O_EXCL, perm)` (:156 vs :36),
`f.Chmod(perm)` (:160 vs :40), the payload write (:161-174 loop vs :42 one `WriteAt`),
`f.Sync()` (:175-177 vs :44-46), close-once (:178-180 vs :47-49), `Rename` (:181-183 vs :50-52),
`Remove` on failure (:184-186 vs :53-55).

**Skeptic (CONFIRMED).** Verified every step by direct read; `filepath.Dir(path)` vs `dstDir`
is equivalent because `dst = Join(dstDir, segmentName)` with `dstDir` cleaned at archive.go:40.
No decision makes the duplicate deliberate (decisions 28, 40, 85 describe Archive as "temp
file, fsync, rename"; build-prompt.md:67 names internal/atomicfile as the shared primitive).
The atomicfile copy is under fault injection at every op and exhaustive crash enumeration
(atomicfile_test.go:80-105, :107+), whereas archive_test.go has no injection and
crash_test.go/kill_test.go never touch Archive: the archive copy is the untested one.

**Fix (apply now, after simplify-9).** In internal/atomicfile add `func WriteFrom(fsys fsx.FS,
path string, perm fs.FileMode, fill func(f fsx.File) error) error` holding the current body
with `fill(f)` in place of the `WriteAt` at :42; `Write` becomes `WriteFrom(fsys, path, perm,
func(f fsx.File) error { _, err := f.WriteAt(data, 0); return err })`. `copySegment` keeps only
its copy (the `io.CopyBuffer` form from simplify-9) inside the callback and drops the
`crypto/rand` and `encoding/hex` imports. `fsx.File` embeds `io.ReaderAt` and `io.WriterAt`
(internal/fsx/fs.go:40-42), so the callback type-checks. Test: atomicfile_test.go unchanged
(now covering the archive path too), archive_test.go, full suite.

## 3. dedup-16: four callers patch `SegmentError.Path` with the same `errors.As` block (CONFIRMED)

**Anchors.** The block `var se *SegmentError; if errors.As(err, &se) { se.Path = ... }` is at
archive.go:141-144 (`"segment %d"`), load.go:329-332 (`a.path`), load.go:449-452 (`s.path`),
reader.go:240-243 (`s.path`). `verifySegment` builds `&SegmentError{...}` without `Path` at
load.go:208 and :211; `recoverSegment` at :224 and :232; `openSegment`/`openActive` set `Path`
at construction (:66, :74, :77, :467, :475, :478), as does reader.go:206. `Error()` (load.go:50-52)
prints `segment %s damaged at byte %d`, so an unpatched error reads "segment  damaged".

**Skeptic (CONFIRMED).** Verified every anchor; found no decision about `SegmentError`
plumbing (decisions 39 and 43 govern recovery and sealing); recovery_test.go:75 asserts
`se.Path == seg` through the load.go:329 route and keeps passing with `Path` set at
construction; test-only callers (load_test.go:19, :23, :266, :288, :292; fuzz_test.go:40, :50;
golden_test.go:126) need a `""` path argument. stat.go's `verifyBounded` (:241) builds its own
Path-less `SegmentError` and copies fields into a `Problem` with its own Path (:173-177), so
it is unaffected. The field comment at load.go:44 is true of both styles; the inconsistency is
where in the chain `Path` is set.

**Fix (apply now).** `verifySegment(r io.ReaderAt, size int64, path string)` and
`recoverSegment(r, size, durable int64, path string)` set `Path: path` in the four constructors;
delete the four blocks; archive.go passes `fmt.Sprintf("segment %d", s.id)`; update the test
callers. Test: recovery_test.go:75 and the full suite.

## 4. dedup-14: eight verbs repeat the open-reader prologue, three the open-writer one (CONFIRMED)

**Anchors.** `r, code := c.reader(); if code != exitOK { return code }; defer r.Close()` at
cmd/quill/verbs.go:353-357 (`cmdStat`), :400-404 (`printFrom`), :446-450 (`cmdTail`), :512-516
(`cmdConsumers`), :572-576 (`cmdCommit`), :607-611 (`cmdRegister`), :632-636 (`cmdUnregister`),
:644-648 (`cmdCheck`); the writer twin at :474-478 (`cmdAppend`), :710-714 (`cmdPrune`),
:728-732 (`cmdRoll`); `cmdRecover` :756-763 closes explicitly and checks the error. Helpers at
:278-284 and :287-298 (the `ErrLocked` message at :293).

**Skeptic (CONFIRMED).** Verified every site; no decision fixes the helper shape (53, 66, 68,
84 set lock rules, exit codes and NoMaintenance) and cli_test.go:81-105 asserts only exit codes,
which the closure form preserves because the usage checks at :569, :604, :629 stay before the
open; `defer` ordering in `cmdCommit`/`cmdRegister` (consumer close inside the closure, reader
close in the helper) matches today's LIFO order. Caveats: the `(T, int)` convention stays in
`start` (:374-394) and `consumer` (:557-566), and the loop-internal `return code` at :531 and
:662 gain one nesting level.

**Fix (apply now).** `func (c *cli) withReader(fn func(*quill.Reader) int) int` and
`withWriter(cfg quill.Config, fn func(*quill.Log) int) int` replacing `reader()`/`writer()`,
keeping the `ErrLocked` message; each verb becomes `return c.withReader(func(r *quill.Reader)
int { ... })`. Do not move the open into the dispatcher (three verbs validate flags first and
would change from exit 2 to exit 1 on a bad `-dir`). Leave `cmdRecover` as is. Test: `go test
./cmd/quill`.

## 5. dedup-13 (= simplify-4): `openSegment` and `openActive` differ in one flag (CONFIRMED)

**Anchors.** load.go:60-84 vs :461-485; one hunk from `diff`: :68 `fsx.O_RDONLY` vs :469
`fsx.O_RDWR`. Callers: `openActive` at load.go:318 only; `openSegment` at load.go:441,
read.go:275, reader.go:198, :232, :280, :344, stat.go:151 and load_test.go:304-338 (the
"archive.go via segRef.open" caller is indirect through the closures at read.go:273 and
reader.go:342).

**Skeptic (CONFIRMED).** Mechanical diff, caller grep, no decision covering a second copy, hard
rules respected (`fsx.O_RDONLY`/`O_RDWR` already exist; all opens stay behind fsx). The fix and
test plan are under simplify-4 in audit-simplify.md item 10.

## 6. dedup-18: `consumerStore.open` repeats `_ = lock.Close(); return nil, err` on eight paths (CONFIRMED)

**Anchors.** consumer.go:62-127: `takeLock` at :70, `c := &Consumer{...}` at :74, and `_ =
lock.Close()` followed by `return nil, ...` at :80, :85, :90, :95, :99, :104, :108, :122, with no
defer or named result.

**Skeptic (CONFIRMED).** No decision mandates hand closes; consumer_test.go:72-89 exercises the
ErrAhead/ErrWrongLog/ErrCorrupt paths but never reopens the name afterwards, so no test would
catch a leaked flock from a future ninth path. The only counter-argument is consistency:
log.go:296-313 uses the same hand-close pattern three times, so this is the codebase's
convention; a fix should say so or cover both.

**Fix (apply now).** Split the body after :74 into `func (c *Consumer) init(opts
ConsumerOptions) error` (returning `err` with no `Close` calls); `open` becomes `c :=
&Consumer{...}; if err := c.init(opts); err != nil { _ = lock.Close(); return nil, err };
return c, nil`. Coordinate with simplify-5 (which drops `open`'s `src` parameter). Optionally
apply the same shape at log.go:296-313; not tracked separately. Test: add
`TestAFailedConsumerOpenReleasesItsLock` (force ErrWrongLog, then reopen the same name and
expect success, not ErrLocked); consumer_test.go.

## 7. dedup-20: seven `fsx.Sim` directory operations open with the same inject/lock/walk prologue (CONFIRMED)

**Anchors.** internal/fsx/crash.go:209-217 (`OpenFile`), :242-250 (`Mkdir`), :264-272
(`ReadDir`), :292-300 (`Lstat`), :319-327 (`Rename`), :352-360 (`Remove`), :374-382 (`SyncDir`):
`inject(op, name)`, error return, `s.mu.Lock()`, `defer s.mu.Unlock()`, `walk(name)`, error
return. `ReadDir` :273-282 and `SyncDir` :383-392 share a descend block differing only in the
`fs.PathError` op string ("readdir" vs "syncdir"). `simFile.begin` (crash_file.go:23-24) is
documented as returning "with the simulator's lock held (the caller unlocks)".

**Skeptic (CONFIRMED).** Hashed the seven preludes after masking the op, path variable and
return arity: identical. Decision 80 and the Sim doc (crash.go:31-33, :136-137) fix the
hook-outside-the-lock ordering, which a single `enter` preserves. One correction that
strengthens the scenario: `inject` takes `s.mu.Lock()` itself (:139), so a copy that locked
first would self-deadlock immediately, which T27 would surface, hence DEDUP not BUG. crash.go
is 596 lines against the ~600 cap and the helper nets roughly 45 lines out.

**Fix (apply now).** `func (s *Sim) enter(op Op, name string) (*simDir, string, error)`: inject,
return on error, lock, walk, unlock and return on walk error, else return with the lock held
(documented like `begin`); each method becomes `d, base, err := s.enter(op, name); if err != nil
{ return ..., err }; defer s.mu.Unlock()`; `Rename` calls `enter` for `oldpath` then `s.walk(newpath)`
under the same lock. Optional `dirAt(d, base, name, op)` for the two descend blocks. Test:
`go test ./internal/fsx/` (T27 exercises every op).
