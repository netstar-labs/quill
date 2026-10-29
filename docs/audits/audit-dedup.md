# A1 adversarial audit: dedup

Date 2026-10-29, against commit 1f4152d. Dimension B (duplication) of the A1 audit: every
candidate, each skeptic's verdict and reason, and the recommended fix. Anchors re-read at
synthesis time. Ranked by what the duplicate puts at risk.

Verdicts: 7 CONFIRMED (dedup-15 was not raised; the finder's numbering skips it). dedup-13 is
the same item as simplify-4 and is counted once.

| # | id | anchor | severity | verdict | recommendation |
|---|----|--------|----------|---------|----------------|
| 1 | dedup-19 | load.go:364 | DEDUP (semantic) | CONFIRMED | owner decision |
| 2 | dedup-13 | load.go:60 | DEDUP | CONFIRMED | apply now (same as simplify-4) |
| 3 | dedup-16 | load.go:329 | DEDUP | CONFIRMED | apply now |
| 4 | dedup-17 | archive.go:151 | DEDUP | CONFIRMED | apply now, after simplify-9 |
| 5 | dedup-14 | cmd/quill/verbs.go:353 | DEDUP | CONFIRMED | apply now |
| 6 | dedup-18 | consumer.go:80 | SIMPLIFY | CONFIRMED | apply now |
| 7 | dedup-20 | internal/fsx/crash.go:209 | DEDUP | CONFIRMED | apply now |

## 1. dedup-19: hole and overlap bookkeeping is written twice, with two differing edge guards (CONFIRMED, semantic)

**Anchors.** load.go:364-376 (sealed), load.go:380-391 (active), reader.go:326-340 (sealed),
reader.go:363-376 (active). Differences: overlap text "inside the range of the one before"
(load.go:367/384) vs "inside the one before" (reader.go:330/375); reader.go:369 guards the active
hole with `if activeFirst > from` while load.go:373, :390 and reader.go:337 do not, so a bad
segment before a good one whose first is 1 would append `Hole{1,0}` there. The tail steps
(load.go:402-405 refuses `head < tl.Seq`, decision 39; reader.go:380-384 reports a hole to
`tl.Seq`, decision 11) differ by design and must stay in their callers.

**Skeptics (2, both CONFIRMED).** Both verified the four sites and the string/guard differences;
`grep "inside the" *_test.go` hits only `t.Fatalf` text, so no test pins either string, and
every test that touches overlap relies on `%w ErrCorrupt`, which a helper keeps. The outputs are
pinned by recovery_test.go:106 (`Hole{1,5}`), :273 (`Hole{5,8}`), reader_test.go:200,
archive_test.go:145 (`Hole{9,12}`), retention_test.go:528, all of which a helper computes
identically. Two corrections: (a) only the reader's active site is hole-then-overlap; its
sealed site is overlap-then-hole like load (equivalent either way, since the overlap error
discards the view); (b) the `Hole{1,0}` divergence is textual, not reachable: segment 0 is the
lowest id and is created at log creation (log.go:405), every rotation needs frames in the active
segment (flush.go:104, :187-188), and an empty sealed segment is bad (load.go:210-212), so the
segment holding seq 1 is always the lowest id in a quill-produced directory. An empty hole would
be inert anyway (read.go:77-79 never matches it; stat.go:32-35 subtracts zero).

**Why owner decision.** The helper is sound and would put the arithmetic decision 38 and
`Stat().Records` depend on in one place, which the self-review (item 2 of "What the audit should
look at first") asked for. But it changes one of two error strings and adds a guard to three
sites, both observable on hand-crafted input, and the finder rated the risk medium. The owner
should approve the unified text and the guard; the synthesis recommends approving both, with the
helper in a new holes.go: `type contiguity struct { prevLast uint64; havePrev, pendingHole bool; holes []Hole }`,
`skip()`, `next(what string, first, last uint64) error`, and the two tail steps left where they are.

## 2. dedup-13: openSegment and openActive are the same function differing in one flag (CONFIRMED)

**Anchors.** load.go:60-84 vs :461-485; the only hunk is :68 `O_RDONLY` vs :469 `O_RDWR`.

**Skeptic (CONFIRMED).** `diff <(sed -n '61,84p') <(sed -n '462,485p')` gives one hunk. Callers:
`openActive` at load.go:318 only; `openSegment` at load.go:441, read.go:275, reader.go:198/232/280/344,
stat.go:151 (archive.go:134 reaches it through a `segRef.open` closure). No decision covers a
deliberate second copy; both constants exist in `internal/fsx`; load.go would shrink.

**Fix.** Same as simplify-4: one flag-taking helper, both names kept as wrappers so the seven
callers and load_test.go:304-338 are untouched.

## 3. dedup-16: four callers patch SegmentError.Path with the same errors.As block (CONFIRMED)

**Anchors.** archive.go:141-144, load.go:329-332, load.go:449-452, reader.go:240-243. Cause:
`verifySegment` (load.go:208, :211) and `recoverSegment` (:224, :232) build `SegmentError`
without `Path`, whereas `openSegment`/`openActive` (:66, :74, :77, :467, :475, :478) and
reader.go:206 set it at construction. `Error()` at load.go:50-52 would print "segment  damaged"
with an empty name if a caller forgot the patch.

**Skeptic (CONFIRMED).** `SegmentError` is constructed only in load.go, reader.go and stat.go,
so no other route delivers one; no decision mentions `SegmentError`, `Path` or the signatures
(decisions 39 and 43 govern recovery and sealing); recovery_test.go:75 asserts `se.Path == seg`
through load.go:329 and still passes with `Path` set in the constructor. Test-only callers
(load_test.go:19, :23, :266, :288, :292; fuzz_test.go:40, :50; golden_test.go:126) need a `""`
path argument. `verifyBounded` (stat.go:241) builds its own Path-less error and copies fields
into a `Problem` with its own `Path`, unaffected. The claim's "only half true" about the field
comment at load.go:44 overstates: the inconsistency is where in the chain `Path` is set.

**Fix.** Add `path string` to `verifySegment` and `recoverSegment`, set `Path: path` in the four
constructors, delete the four blocks; archive.go passes `fmt.Sprintf("segment %d", s.id)`.

## 4. dedup-17: copySegment re-implements atomicfile.Write's temp/fsync/rename protocol (CONFIRMED)

**Anchors.** archive.go:151-187 vs internal/atomicfile/atomicfile.go:31-56: same random temp
name under `tempPrefix` (= `atomicfile.TempPrefix`, state.go:32), same `O_RDWR|O_CREATE|O_EXCL`
open (archive.go:156 / atomicfile.go:36), `Chmod`, payload write (a loop vs one `WriteAt`),
`Sync`, close-once, `Rename`, `Remove` on failure.

**Skeptic (CONFIRMED).** Verified line by line; `filepath.Dir(path)` vs `dstDir` is equivalent
because `dst = Join(dstDir, segmentName)` with `dstDir` cleaned at archive.go:40. Decisions 28,
40 and 85 describe Archive as "temp file, fsync, rename" with no rationale for a separate copy,
and build-prompt.md:67 names `internal/atomicfile` as the shared primitive. The atomicfile
sequence is under fault injection at every op (atomicfile_test.go:80-105) and exhaustive crash
enumeration (:107+), whereas archive_test.go has no injection and crash_test/kill_test/recovery_test
never touch `copySegment`: the duplicate is the untested copy. `fsx.File` embeds `io.ReaderAt`
and `io.WriterAt` (internal/fsx/fs.go:40-42), so a `fill func(f fsx.File) error` callback is type-valid.

**Fix.** `atomicfile.WriteFrom(fsys, path, perm, fill)` holding the current `Write` body with
`fill(f)` in place of the `WriteAt`; `Write` becomes a three-line wrapper; `copySegment` keeps
only the copy inside the callback (the `io.CopyBuffer` form from simplify-9) and drops its
`crypto/rand` and `encoding/hex` imports. The atomicfile crash tests then cover the archive path.

## 5. dedup-14: eight verbs repeat the open-reader prologue, three the open-writer one (CONFIRMED)

**Anchors.** The five-line `r, code := c.reader() / if code != exitOK { return code } / defer r.Close()`
at cmd/quill/verbs.go:353-357, 400-404, 446-450, 512-516, 572-576, 607-611, 632-636, 644-648;
the writer twin at 474-478, 710-714, 728-732; cmdRecover 756-765 closes explicitly and checks
the error.

**Skeptic (CONFIRMED).** Every anchor verified. No decision fixes the helper shape (53, 66, 68,
84 set lock rules, exit codes and NoMaintenance); cli_test.go:81-105 asserts only exit codes,
which the closure form preserves because the usage checks at :569, :604 and :629 stay before
the open; defer order in cmdCommit/cmdRegister (consumer close inside the closure, reader close
in the helper) matches today's LIFO. Caveats: the `(T, int)` convention remains at `start`
(:374-394) and `consumer` (:557-566), so two styles coexist unless those follow; the closure adds
a nesting level at the loop-internal `return code` sites (:531, :662).

**Fix.** `withReader(fn func(*quill.Reader) int) int` and `withWriter(cfg, fn func(*quill.Log) int) int`
replacing `reader()`/`writer()`, keeping the `ErrLocked` message; leave the three usage checks
before the open and leave cmdRecover as is. Do not move the open into the dispatcher (exit 2
must keep winning over an unopenable `-dir`).

## 6. dedup-18: consumerStore.open repeats `_ = lock.Close(); return nil, err` on eight paths (CONFIRMED)

**Anchors.** consumer.go:70 (lock), :74 (`c := &Consumer{...}`), close-and-return pairs at
:80-81, 85-86, 90-91, 95-96, 99-100, 104-105, 108-109, 122-123.

**Skeptic (CONFIRMED).** Exact. No decision mandates explicit closes; consumer_test.go:72-89
asserts only the error values and never reopens the same name to prove the flock was released.
The same hand-close pattern exists at log.go:296-313 (three sites), so this is a convention
rather than a lapse; a fix should either cover both or be framed as a convention change. The
scenario (a ninth path that forgets the close) is forward-looking; there is no present leak.

**Fix.** Split the body after :74 into `func (c *Consumer) init(opts ConsumerOptions) error`;
`open` becomes construct, `init`, close-on-error once. Optionally apply the same shape to
log.go:296-313 in the same change.

## 7. dedup-20: fsx.Sim: seven directory operations open with the same inject/lock/walk prologue (CONFIRMED)

**Anchors.** internal/fsx/crash.go:209-217 (OpenFile), 242-250 (Mkdir), 264-272 (ReadDir),
292-300 (Lstat), 319-327 (Rename), 352-360 (Remove), 374-382 (SyncDir); the descend blocks at
273-282 and 383-392 differ only in the `Op:` string of their `PathError`s.

**Skeptic (CONFIRMED).** After masking the op constant, path variable and return arity, all
seven preludes hash identically. crash_file.go:23-24 documents `begin` as returning with the
lock held, the precedent the claim cites. Decision 80 and the Sim doc (crash.go:31-33, 136-137)
fix the hook-outside-the-lock ordering, which one `enter` preserves. Correction to the scenario:
`inject` takes `s.mu.Lock()` itself at crash.go:139, so a copy that locked first would
self-deadlock at once, which T27 would surface; hence DEDUP, not a bug. crash.go is 596 lines
against the ~600 cap; the helper nets roughly 45 lines out.

**Fix.** `func (s *Sim) enter(op Op, name string) (*simDir, string, error)` (inject; lock;
walk; unlock on walk error; otherwise return locked, documented like `begin`), each method
`d, base, err := s.enter(op, name); if err != nil { return ..., err }; defer s.mu.Unlock()`;
Rename calls `enter` for oldpath then `walk(newpath)` under the same lock; optionally
`dirAt(d, base, name, op)` for the two descend blocks.
