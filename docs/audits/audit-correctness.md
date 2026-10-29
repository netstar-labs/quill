# A1 adversarial audit: correctness

Date 2026-10-29, against HEAD b52b746 (the source is identical to 1f4152d; only docs/audits
changed in between). Dimension C (correctness) of the A1 audit: every candidate, each
skeptic's verdict and reason, the reproduction, and the recommended disposition. Reproductions
ran in throwaway copies of the repository under the session scratchpad
(written below as `scratchpad/`); no repository file was edited by the audit. Every anchor was
re-read from the tree at synthesis time. Ranked by severity, then by reachability.

Verdicts: 12 CONFIRMED (correctness-27 by one CONFIRMED and one PLAUSIBLE vote). Two further
correctness bugs were raised under other lenses and are cross-referenced: simplify-1 (a
Reader nil dereference on a seq-0 first frame, audit-simplify.md item 1) and docs-33 (offline
verbs overwrite quill.policy, audit-docs.md item 1, the same question as correctness-32).

| # | id | anchor | severity | votes | verdict | recommendation |
|---|----|--------|----------|-------|---------|----------------|
| 1 | correctness-21 | retention.go:219 | BUG (semantic) | 3 | CONFIRMED | owner decision: approve the snapshot fix |
| 2 | correctness-22 | log.go:468 | BUG (semantic) | 3 | CONFIRMED | owner decision: approve retire-if-pinned |
| 3 | correctness-23 | reader.go:323 | BUG | 3 | CONFIRMED | apply now |
| 4 | correctness-24 | cmd/quill/verbs.go:700 | BUG (semantic) | 3 | CONFIRMED | owner decision: strict decimals (absorbs simplify-3) |
| 5 | correctness-32 | cmd/quill/verbs.go:756 | MINOR (semantic) | 2 | CONFIRMED | owner decision, one issue with docs-33 |
| 6 | correctness-28 | cmd/quill/verbs.go:478 | MINOR (semantic) | 2 | CONFIRMED | owner decision |
| 7 | correctness-27 | reader.go:280 | MINOR (semantic) | 2 | CONFIRMED | owner decision |
| 8 | correctness-26 | reader.go:199 | MINOR | 1 | CONFIRMED | apply now |
| 9 | correctness-31 | load.go:453 | MINOR (semantic) | 2 | CONFIRMED | owner decision: approve |
| 10 | correctness-25 | retention.go:245 | MINOR | 1 | CONFIRMED | apply now |
| 11 | correctness-29 | load.go:418 | MINOR | 1 | CONFIRMED | apply now |
| 12 | correctness-30 | read.go:268 | PERF | 1 | CONFIRMED | deferred |

## 1. correctness-21: `deleteLocked` indexes a stale `n` after `publishLocked`'s nested rotation and caps shorten `l.sealed` (CONFIRMED, semantic)

**Anchors.** retention.go:215-243: `deleteLocked(n, offsets)` trusts the caller's `n`, calls
`l.publishLocked()` at :219, then `for i := 0; i < n; i++ { s := l.sealed[i]` at :226-227 and
`l.sealed = l.sealed[n:]` at :243. flush.go:102-146: `publishLocked` rotates when
`l.active.durable >= l.cfg.SegmentBytes && l.active.frames > 0` (:104) and after a successful
rotation calls `l.deleteLocked(l.capsLocked(0), offsets)` (:141), which removes a prefix of
`l.sealed` before control returns to the outer `deleteLocked`. The state is reachable because a
failed rotation leaves the active segment growing with `pendingPublish` set (flush.go:106-110,
:129; decision 60 d) and neither `Prune` (retention.go:76-89) nor `pruneLocked` (:91-146, whose
age-rotation branch is guarded by `!l.pendingPublish` at :103) retries the publish before
computing `n` at :113-123 and :144. The prune tick at flush.go:266-275 reaches `pruneLocked` the
same way, so it is not confined to `quill prune`.

**Skeptics (3, all CONFIRMED).** Skeptic 1 reproduced both outcomes on the simulator (the
`index out of range [0] with length 0` panic at retention.go:227, and with a smaller backlog the
silent deletion of a fresh sealed segment: 428 bytes against MaxBytes 400 required one
deletion, four happened) and found the self-review only establishes the recursion is bounded.
Skeptic 2 reproduced the same two outcomes with its own test and the recovered stack
(retention.go:227 <- :145 <- :88). Skeptic 3 added two sharper variants (records written after
the clock advance, so unexpired, still deleted; a `ProtectOn` consumer committed at 16 loses its
unread segment with `Stat.Dropped=1` while the cap required one deletion), noted the
precondition that at least one cap must be set, applied both candidate fixes in sibling copies
with all four tests and the full suite green, and preferred the snapshot variant because the
`publishLocked(runCaps bool)` variant skips the caps at the one seal triggered inside
`deleteLocked`, a wording deviation from decision 24 ("checked at every segment seal").

**Reproduction** (`scratchpad/a1-reentrancy-lens/repro/audit_repro_test.go`, package quill,
Sim backend; the second skeptic's `scratchpad/quill-copy/zz_repro_deletelocked_reentrancy_test.go`
is the same shape with `openRetention`). Setup: SegmentBytes 4096, MaxBytes 16384, Retention 1h,
SegmentMaxAge -1, NoMaintenance, injected clock; fill three segments (`sealed [0,1,2]`); set
`s.Inject` to fail `fsx.OpCreate` for the next segment; append and `Flush` `extra` segments'
worth (each Flush returns the injected error, `pendingPublish=true`); clear the injection;
advance the clock 2h; `Prune(true)`.

```
GOWORK=off go test -run TestAudit -v -count=1 .
TestAuditDeleteLockedReentrancy (extra=4): active.durable=20240 head=35 pendingPublish=true
  PANIC in Prune: runtime error: index out of range [0] with length 0   (retention.go:227)
TestAuditDeleteLockedOverDeletes (extra=1): prune: <nil>; after prune: sealed=[] active.first=21
  Read(16): 0 recs, err=quill: records 16 to 20 are unavailable      (expected sealed=[3], first=16)
TestAuditOverDeleteFresh: same over-delete for records written after the clock advance
TestAuditOverDeleteProtected (+ ProtectOn consumer at 16): sealed=[] dropped=1
```

With the snapshot fix (`fixsnap`) or the flag fix (`fixflag`) applied: all four pass with
`sealed=[3[16-20]]`, `Read(16): 1 recs`, and `GOWORK=off go test ./... -count=1` green.

**Why owner decision.** `Prune` stops panicking and stops deleting segments the sweep did not
select; the fix has two variants and one of them bends decision 24. Recommended answer:
approve the snapshot variant. Fix: before `publishLocked` at retention.go:219 build `targets :=
make(map[*segment]bool, n)` over `l.sealed[:n]`; after it returns, recompute `n` as the length of
the leading run of `l.sealed` whose entries are in `targets`; the rest unchanged. Test: the four
reproduction tests above in a new retention_reentrancy_test.go (retention_test.go is 615
lines), each asserting no panic, `sealed == [3]`, `First == 16` and `Dropped == 0` for the
protected variant.

## 2. correctness-22: `Close` nils `active.f` under the lock while a pinned reader uses it without the lock (CONFIRMED, semantic)

**Anchors.** log.go:467-472: `if l.active != nil && l.active.f != nil { ... l.active.f.Close()
... l.active.f = nil }` with no look at `l.active.readers` or `retired`. read.go:265-266: `view`
increments `a.readers` under `l.mu` and captures `a`; the active `segRef`'s `open` at :288-289
returns `a.f` later, outside the lock; `scanSource` passes it into `window{r: r}` (read.go:200)
whose `bytes` calls `w.r.ReadAt` (load.go:121) with no nil check. Rotation honours the pin
(flush.go:165-169: `old.retired = true; if old.readers == 0 { close }`) and `release` closes at
zero (read.go:299-303); `Close` is the one path that does not. scope.md:77-78 promises a `*Log`
is safe for concurrent use.

**Skeptics (3, all CONFIRMED).** All three reproduced it independently on the simulator: one
sealed segment plus an active with two frames, `l.Read(1,100)` in a goroutine delayed 300 ms
on the sealed segment's `Lstat` via `Sim.Inject`, `Close` from the main goroutine after 100 ms;
`Close` returns nil and the reader panics with a nil dereference, stack load.go:121 <- :136 <-
:170 <- read.go:202 <- :104 <- :16, also under `-race`. Skeptic 1 noted Seek (seek.go:71) and the
archiver (archive.go:134) take the same `open()` path; skeptic 2 corrected the scenario's segment
id (the sealed one is `0000000000.log`, id 0); skeptic 3 applied the proposed fix in a copy:
the test then returns four records with `OpenFiles()` back to 0 after release, the full suite,
`-race` over the Close/Reader/Iterator/Follow/Rotation tests and `go vet` all clean, and
decision 53 (the lock file is still released at log.go:473) and decision 60 are untouched; the
one nuance is that a pinned descriptor's eventual `Close` error is dropped by the release path
(`_ = a.f.Close()`, read.go:301), as it already is after rotation.

**Reproduction** (`scratchpad/quill-copy/zz_repro_close_under_reader_test.go`, package quill).
`s := fsx.NewSim(); l := openSim(t, s, nil); appendN(t, l, 3); mustFlush(t, l); l.Roll();
appendN(t, l, 2); mustFlush(t, l)`; `s.Inject` sleeps 300 ms on `OpLstat` of `l.sealed[0].path`;
goroutine `l.Read(1, 100)` under `recover()`; main sleeps 100 ms, `l.Close()`.

```
GOWORK=off go test -run TestReproCloseUnderReader -count=1 -v .
    sealed=1 sealedPath=/log/0000000000.log active.id=1 active.frames=2 head=5
    Close returned: <nil>
    reader: recs=0 err=<nil> panic=runtime error: invalid memory address or nil pointer dereference
```

With the fix in a copy: `reader goroutine result: <nil>`, `open handles after close: 0`.

**Why owner decision.** `Close` changes behaviour (a read-only descriptor may outlive `Close`
until the last in-flight reader releases) and the finder flagged it semantic; the change
mirrors what rotation already does under decision 43. Recommended answer: approve. Fix: in
`Close` replace log.go:467-472 with `if l.active != nil && l.active.f != nil { if
l.active.readers > 0 { l.active.retired = true } else { if cerr := l.active.f.Close(); err == nil
{ err = cerr }; l.active.f = nil } }`. Test: the reproduction as
`TestCloseUnderAnInFlightReadDoesNotPanic` asserting four records, no panic and
`s.OpenFiles() == 0` after the read returns; run under `-race`.

## 3. correctness-23: `Reader.view` reads `s.index`/`s.last` unsynchronised against `scan`'s `once.Do` write (CONFIRMED)

**Anchors.** reader.go:247 `s.index, s.last, s.frames, s.lastAt = ...` inside `s.once.Do`
(:231); reader.go:323 `if s.index != nil && s.last < last` and :342 `index: s.index` in `view`,
which never calls `Do` and holds `r.mu` only at :255-257 and in `segment()` (:212-218), neither
of which orders against the write. scope.md:77-78 and reader.go:33 promise a `*Reader` is safe
for concurrent use; CI runs `go test -race` (.github/workflows/ci.yml:39) but no test reads one
Reader from several goroutines.

**Skeptics (3, all CONFIRMED).** All three reproduced the detector report (write at :247 via
`sync.(*Once).doSlow` from `scanSource` read.go:163; previous read at :323 from `scanSource`
read.go:136; both from `Reader.Read` reader.go:148) with 8 goroutines calling `r.Read(7,3)`.
Skeptic 2 also saw the sibling report at read.go:192 (`slices.BinarySearchFunc` over an index
whose header came from the racy read) and ran 35,644 fresh-Reader rounds without the detector
with no wrong outcome, so the torn-read consequence stays inferred. Skeptics 2 and 3 corrected
the scenario: the `last < s.first` guard at :320 runs before :323, so a torn `last == 0` does
not skip the segment; it yields a `segRef` with `last = 0` and a spurious hole from the next
contiguity check (:326-337), still a spurious `GapError`. Decision 63 says the index is "built
on first use ... and cached" and nothing about publication; no test references
`readerSegment` fields; a mutex variant cannot deadlock because `scan` runs after `view` has
released `r.mu`.

**Reproduction** (`scratchpad/quill-race/zz_audit_race_test.go`; the second skeptic's copy uses
a real directory with `openDir`):

```go
func TestAuditReaderIndexRace(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
	for i := 0; i < 8; i++ { appendN(t, l, 5); mustFlush(t, l) }
	l.Close()
	r := openReaderSim(t, s); defer r.Close()
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ { wg.Add(1); go func() { defer wg.Done(); for i := 0; i < 20; i++ { if _, err := r.Read(7, 3); err != nil { t.Errorf("read: %v", err); return } } }() }
	wg.Wait()
}
```

```
GOWORK=off go test -race -count=5 -run TestAuditReaderIndexRace .
WARNING: DATA RACE
Write at ... by goroutine 12:  quill.(*readerSegment).scan.1()  reader.go:247  (sync.(*Once).doSlow; scanSource read.go:163)
Previous read at ... by goroutine 9:  quill.(*Reader).view()  reader.go:323  (scanSource read.go:136)
--- FAIL: TestAuditReaderIndexRace  testing.go:1867: race detected during execution of test
```

**Fix (apply now).** Replace the four fields with an `atomic.Pointer[scanResult]` where
`scanResult{index []anchor; last, frames uint64; lastAt int64}` is built inside `once.Do` and
stored once; `view` (:323, :342) and `ensureIndex` (:339-340 region) load the pointer once and
read its fields. Standard library only; no on-disk change. Test: the reproduction as
`TestConcurrentReadsOnOneReaderAreRaceFree` in reader_test.go, run under `-race` in the gate.

## 4. correctness-24: `quill prune`/`commit` parse numbers with `Sscanf %d`, which silently truncates `1M`, `1e6`, `0x10` (CONFIRMED, semantic)

**Anchors.** cmd/quill/verbs.go:700 `fmt.Sscanf(f.s, "%d", f.dst)` for `-max-bytes`,
`-min-free-bytes`, `-max-records` (loop at :695-704); :592 `fmt.Sscanf(c.offset, "%d", &next)`
for `commit -offset`; :778 for `@unix`. The value flows into `c.writer(cfg)` (:710) and
`cons.Commit(next)` (:595). retention.go:205-208 applies `MaxRecords` literally and the comment
at :175-177 says caps override protection; consumer.go:175 accepts any `next` in `[first,
head+1]`.

**Skeptics (3, all CONFIRMED).** Skeptic 1 ran the built binary: a log of 11 records in 4
segments, `quill prune -dir d -max-records 1e6` printed `segments 4 -> 1, bytes 152 -> 26, first
1 -> 10`, exit 0; `commit -offset 10abc` committed 10 with exit 0. Skeptic 2 reproduced through
the CLI's own `run` entry point (below). Skeptic 3 traced `MaxRecords=1` through `capsLocked`
(`firstAfterLocked(len(sealed))` falls to `active.first`, retention.go:166-169, so every sealed
segment goes), confirmed `-force` affects only the clock guard, and noted `quill.parseUint`
(state.go:202) is unexported so the CLI must use `strconv`. No test passes a malformed number
(cli_test.go:232, :342, :361 are plain digits; no test passes the prune byte flags at all), and
decision 75 argues for strict decimals, decision 86 says nothing about numeric syntax.

**Reproduction** (`scratchpad/quill-copy/cmd/quill/zz_repro_sscanf_trailing_test.go`, package
main, using the test helper `exec`): a log with 3 sealed segments of 10 records plus an active
segment (`Retention: -1, FlushInterval: time.Hour, NoMaintenance: true`, `Roll` between).

```
GOWORK=off go test ./cmd/quill -run 'TestReproSscanfParse|TestReproPruneMaxRecords1e6|TestReproCommitOffset1e3' -count=1 -v
    Sscanf("1e6", %d) -> n=1 v=1 err=<nil>; "1e3" -> 1; "10abc" -> 10; "0x10" -> 0
    before: head=40 first=1 segments=4 records=40
    prune -max-records 1e6: exit=0 stdout="segments 4 -> 1, bytes 840 -> 210, first 1 -> 31\n"
    BUG REPRODUCED: -max-records 1e6 was applied as MaxRecords=1
    commit -offset 1e3: exit=0 stdout="c 1\n"; consumer c offset after commit: 1
    BUG REPRODUCED: -offset 1e3 committed offset 1 with exit 0
```

**Why owner decision.** Input accepted today (exit 0) becomes a usage error (exit 2), and
whether to accept K/M/G suffixes by design is a CLI-shape choice (decision 86 territory).
Recommended answer: strict decimals now, suffixes as a separate chore if wanted. Fix: at :592
`next, err := strconv.ParseUint(c.offset, 10, 64); if err != nil { return c.usage("-offset: %v",
err) }`; at :700 `v, err := strconv.ParseInt(f.s, 10, 64); if err != nil { return c.usage("%q: %v",
f.s, err) }; *f.dst = v`; at :778 `strconv.ParseInt(s[1:], 10, 64)`; add the `strconv` import.
Test: the three reproduction tests above, now asserting exit 2 and an unchanged log, plus
`"12abc"` and `"1_000"` cases; existing cli_test.go unchanged.

## 5. correctness-32: offline verbs open with the default 8 MiB `SegmentBytes` and rotate a larger active segment at load (CONFIRMED, semantic; one issue with docs-33)

**Anchors.** cmd/quill/verbs.go:474 (`cmdAppend`), :728 (`cmdRoll`), :756 (`cmdRecover`) each
call `c.writer(quill.Config{Retention: -1})`; `writer` (:287-298) adds only `Dir` and
`NoMaintenance`; only `cmdPrune` reads the policy (:681). `SegmentBytes` zero becomes
`DefaultSegmentBytes = 8 << 20` (log.go:75, :105-107). `load` ends with `l.publishLocked()`
(load.go:414) which rotates on `durable >= SegmentBytes && frames > 0` (flush.go:104);
`NoMaintenance` gates only the prune ticker and age rotation (log.go:61-63, flush.go:257-260).
`Open` also rewrites `quill.policy` from its own Config at log.go:305 (decision 29).

**Skeptics (2, both CONFIRMED).** Skeptic 1 reproduced with the built binary: a writer with
`SegmentBytes: 16 << 20` and nine 1 MiB records, closed; `quill recover` printed `recovered: head
9, first 1, 2 segments`, leaving a new empty `0000000001.log`, `quill.tail` naming segment 1, and
`quill.policy` rewritten from `segment_bytes 16777216` to `8388608`. Skeptic 2 reproduced at the
library level and through `mustRun("recover")`, applied fix option A (an `offlineConfig()` taking
`SegmentBytes` from `ReadPolicy`, fallback `MaxSegmentBytes`) with `go vet` and the cmd/quill
suite green and recover then printing `1 segments`; it noted option B as worded (skip size
rotation under `NoMaintenance`) would not cover `quill append`, whose flush-time
`publishLocked` still rotates at 8 MiB, and that the policy clobbering (retention, caps,
protected consumers) is a sibling defect, which is docs-33.

**Reproduction** (`scratchpad/repro/main.go` with a `replace` to the repo, then the built CLI):

```
GOWORK=off GOFLAGS=-mod=mod go run . $S/log          # SegmentBytes 16 MiB, nine 1 MiB records
GOWORK=off go build -o $S/quill ./cmd/quill && $S/quill recover -dir $S/log
recovered: head 9, first 1, 2 segments
# before: 0000000000.log 9437328 bytes, policy segment_bytes 16777216
# after:  + empty 0000000001.log, quill.tail {"seq":9,"seg":1,"off":0,"maxseg":1}, policy segment_bytes 8388608
```

**Why owner decision.** It defines what Config the offline writable verbs open with, which
decisions 29, 68 and 87 describe but the code does not implement; see audit-followups.md for
the recommended answer (take `SegmentBytes` and `MaxPayloadBytes` from the policy, keep
`Retention: -1` and no caps, skip `writePolicy` under `NoMaintenance`, document that `roll`
rotates regardless).

## 6. correctness-28: `quill append -sync=false` exits 0 and prints the sequence when the deferred `Close`, its only flush, fails (CONFIRMED, semantic)

**Anchors.** verbs.go:478 `defer l.Close()` (error discarded); :497 `AppendBatch`; :502-506 `Sync`
only when `c.sync` (flag at :84, default true); :507-508 prints `last` and returns `exitOK`.
`AppendBatch` buffers (append.go:19-45; inline flush only past `FlushBytes` with the error
discarded at :77-78); the flusher goroutine discards flush errors (flush.go:281); `Close`'s
`flushLocked` (log.go:460) is the only error-reporting flush and its error is dropped by the
defer. `cmdRecover` (:761) is the one verb that checks `Close`.

**Skeptics (2, both CONFIRMED).** Skeptic 1 reproduced the shape on the simulator (inject
`ENOSPC` on `OpWrite`; `AppendBatch` returns seq 1; `Close` returns `no space left on device`;
the CLI would print 1 and exit 0; after reopen `head=0`) and noted the prior audit files had
already catalogued it. Skeptic 2 confirmed reachability, that a second `Close` returns
`ErrClosed` without touching anything (log.go:449-452) so the defer can stay as a safety net,
that decision 53 still holds, and that under `-sync=true` a Close failure after a successful
Sync would newly exit 1, consistent with decision 42 and scope.md:67 ("an error means unknown,
not lost").

**Reproduction** (`scratchpad/quill-repro/zz_throwaway_test.go`): open with `Retention -1`,
`NoMaintenance`; inject `syscall.ENOSPC` on every `OpWrite`; `AppendBatch` one record; `Close`
with the error discarded; reopen.

```
GOWORK=off go test -run TestThrowawayCmdAppendShapeLosesRecordOnENOSPC -v .
    AppendBatch: seq 1 err <nil>
    Close returned: no space left on device (errors.Is ENOSPC: true)
    CLI would print 1 and exit 0
    after reopen: head=0 (record 1 written? false)
```

**Why owner decision.** It changes a documented verb's exit-code contract (decision 53) and has a
real choice for `-sync=true`, where a Close failure after a successful Sync (decision 60 classes
d/e) leaves the records durable (decision 78). Recommended answer: explicit `Close` before
printing; exit 1 on failure; when `Sync` had succeeded, still print the sequence to stdout and
the error to stderr.

## 7. correctness-27: `Reader.view` returns a raw `*fs.PathError` when a listed segment is deleted between listing and opening (CONFIRMED by one vote, PLAUSIBLE by the other)

**Anchors.** reader.go:279-283: the marker's segment is opened once with no retry (unlike
`readTail` at :121-139); the sealed `open` closure at :343-347 forgets the entry and returns the
raw error; read.go:181-187 passes it to the caller when no record was delivered; iter.go:68-71
ends a Follow on it. The writer seals, writes the marker and deletes in one locked step
(flush.go:105, :114, :141; retention.go:228). Guarantee 14 (scope.md:354-355) and decision 11
call for a `GapError`.

**Skeptics (1 PLAUSIBLE, 1 CONFIRMED).** The PLAUSIBLE skeptic verified the mechanism by code
reading but found two overstatements: `MaxBytes` cannot delete the just-sealed segment unless
that segment alone exceeds `MaxBytes` (log.go:155-156 refuses `MaxBytes` below twice
`SegmentBytes`; retention.go:184-187 deletes oldest-first), though `MinFreeBytes` (:194-197) and
`MaxRecords` (:205-208) can; and on a segment's first read `ensureIndex` (read.go:160-166)
already maps the failure to a `GapError`, so the raw error escapes only once the index is
cached (a Follow or repeated-read loop). The CONFIRMED skeptic reproduced both escapes
deterministically with an `fsx.FS` wrapper that unlinks the victim on its Nth `Lstat`, applied
a sketch of both fix parts with the full suite green, and noted archive.go:134 and seek.go:71
would also see the `GapError`, which matches their documented returns.

**Reproduction** (`scratchpad/skeptic-r280-46034`, a `raceFS` wrapping `fsx.Sim`): SegmentBytes
60, three sealed segments of four records plus one active record; OpenReader through the
wrapper; warm `Read(5,10)` (caches segment 1's index); arm the wrapper for the next `Lstat` of
`/log/0000000001.log`; `Read(5,10)` again.

```
GOWORK=off go test -run TestRepro -v -count=1 .
    first Read(5): err=lstat /log/0000000001.log: file does not exist  isGap=false  isNotExist=true
    second Read(5): err=quill: records 5 to 8 are unavailable isGap(5,8)=true
    (marker's segment removed at its first Lstat): err=lstat /log/0000000001.log: file does not exist
```

With the sketch applied: the injected deletion gives `isGap(5,8)=true`; the marker-segment
retry returns the same `ErrNotExist` after ~5 ms when nobody republishes a marker.

**Why owner decision.** The fix changes the error type three callers see (Read/Follow, Seek,
Archive) and part 1 adds a retry loop with sleeps to `view`. Recommended answer: approve part 2
only (map `fs.ErrNotExist` from a sealed segment's `open` to `*GapError{From: cur, To: s.last}`
in read.go:181-187, after `forget`, since errors.go:47 documents `From` as the requested
offset); treat the marker-segment retry as optional.

## 8. correctness-26: a Reader caches any `openSegment` error as a permanent bad segment (CONFIRMED)

**Anchors.** reader.go:198-200 stores any error from `openSegment` (an unwrapped I/O error from
`fsys.OpenFile`, load.go:68-71, included) in `s.bad`, and :212-218 inserts the entry into
`r.segs`. `forget` (:223-227) is called only from the `open` closure of a good `segRef` (:346,
:351); a bad entry gets no `segRef` because `view` skips it with `pendingHole = true; continue`
(:300-303), so nothing re-examines it; the range becomes a hole (:332-337 or :380-383) and a
`GapError` (read.go:77-80). Decision 83 says an entry "is dropped when a read finds the file
gone or changed in size", which cannot happen for a bad entry.

**Skeptic (CONFIRMED).** Verified every anchor and that no other code touches `r.segs`
(reader.go:41, :81, :191, :213, :216, :225); `Log.loadSealed` (load.go:441-445) treats an
open-time I/O error the same way but is a one-shot load, not a long-lived cache; no test injects
an open failure on a Reader path; EMFILE is reachable because `Lstat` consumes no descriptor
but `OpenFile` does.

**Reproduction** (`scratchpad/quill-copy/zz_repro_test.go`): Sim log with SegmentBytes 200,
30 records appended and flushed (segment 0 sealed, 1 active); open a Reader; `s.Inject`
returns a one-off error for `OpOpen` on segment 0; `r.Read(1,5)`; clear `Inject`; read three
more times and `Stat`.

```
GOWORK=off go test -run TestReproTransientOpenErrorIsCachedAsHole -v -count=1 .
    first read (during injection): err="quill: records 1 to 30 are unavailable" hits=1
    read 0/1/2 after injection cleared: n=0 err="quill: records 1 to 30 are unavailable" isGap=true
    Stat: Head:30 First:31 Holes:[{From:1 To:30}]
    FAIL: reader never recovers after a transient open error
TestReproControlFreshReaderRecovers: fresh reader n=5 err=<nil>  PASS
```

**Fix (apply now).** In `segment()` cache the entry only when `err` is a `*SegmentError`
(`errors.As`) or the first-frame check failed; for any other error build the entry with
`s.bad = err` but return it without inserting into `r.segs`, so the next `view` re-examines the
file. Test: the reproduction as `TestATransientOpenErrorIsNotCachedByTheReader` asserting the
post-injection reads return five records, plus a control that a `*SegmentError` (FIFO or
oversize file) stays cached.

## 9. correctness-31: a damaged sealed segment's bytes are invisible to `MaxBytes` and `Stat.Bytes` (CONFIRMED, semantic)

**Anchors.** load.go:439-457: `loadSealed` takes `size` from `openSegment` at :441 but on a
`verifySegment` failure sets `s.bad = err; return s` at :453-454 before the only assignment of
`s.size` at :456. Bad segments stay in `l.sealed` (load.go:357-362). `capsLocked` sums
`bytes += s.size` (retention.go:181-182) and `Log.Stat` sums `seg.size` (stat.go:52-53), so
the file's real size counts toward neither. `Reader.Stat` uses `Lstat` sizes (stat.go:77-85),
so writer and reader already disagree on a damaged log.

**Skeptics (2, both CONFIRMED).** Both verified the anchors by reading and found no test
asserting `Log.Stat().Bytes` or `MaxBytes` behaviour with a damaged-but-present sealed segment
(`TestSizeAccountingIsMeasured`, retention_test.go:474-531, removes its hole with `s.Remove`);
decision 32 and scope.md:246 define `Bytes` as the sum of segment sizes with no exclusion;
scope.md:239/347 promise the bound the uncounted file exceeds. Skeptic 2 confirmed the fix is
safe: the only other reader of a sealed `size` is read.go:279-281, unreachable for bad segments
(skipped at :268-270); the reader side already sets `s.size` before verification
(reader.go:202); the deletion loop already walks over bad segments. Residual gap: when
`openSegment` itself fails (:442-444) `size` stays 0 under the fix.

**Why owner decision.** `MaxBytes` would evict sooner and `Stat().Bytes` would change on a
damaged log; the finder flagged it semantic; the change restores decision 32 and guarantee 11.
Recommended answer: approve. Fix: move `s.size = size` to right after :441 succeeds (before
`verifySegment`). Test: damage a sealed segment (flip one byte, as load_test.go:125-130 does),
reopen, assert `Log.Stat().Bytes` equals the sum of the files on disk and equals
`Reader.Stat().Bytes`; with `MaxBytes` set, assert the damaged segment is counted toward
eviction.

## 10. correctness-25: `deleteLocked` compacts `l.holes` in place while readers' views alias the backing array (CONFIRMED)

**Anchors.** retention.go:244-251: `kept := l.holes[:0]; for ... { kept = append(kept, h) };
l.holes = kept` rewrites the array in place when a leading hole is dropped. read.go:267 builds
`segView{holes: l.holes, ...}` under `l.mu` without copying (segs are rebuilt into a fresh slice
at :268-285; holes are not); `locate` ranges over `v.holes` at read.go:77-81 after the lock is
released. stat.go:29 hands the same backing array to callers as `Stats.Holes`.

**Skeptic (CONFIRMED).** Reproduced the race with the detector and a deterministic second
consequence the claim did not mention: a caller's `Stat().Holes` slice is rewritten under it by
the next compacting prune (`before={5,8}` became `{13,16}`, length still 2). The torn-`Hole`
`GapError` consequence stays inferred (not reproduced; it needs a specific interleaving of two
word reads). No decision addresses holes being shared; `l.holes` is only ever reassigned
(load.go:409, retention.go:251), so an allocated `kept` makes it immutable after publication.

**Reproduction** (`scratchpad/quillcopy/zz_holes_alias_test.go`): Sim, SegmentBytes 60, six
segments of four records, close, remove segments 1 and 3 (holes `{5,8}`, `{13,16}`), reopen with
`MaxRecords=16` and `NoMaintenance`; `st := l.Stat()`; four goroutines loop `l.Read(10, 1)`;
after 2 ms `l.Prune(true)`.

```
GOWORK=off go test -race -count=10 -run TestZZHolesAliasPrune .
WARNING: DATA RACE
Read at ... by goroutine 15:  quill.(*segView).locate() read.go:77  <- scanSource read.go:152 <- readFrom read.go:104 <- (*Log).Read read.go:16
Previous write at ... by goroutine 7:  quill.(*Log).deleteLocked() retention.go:248 <- pruneLocked retention.go:145 <- Prune retention.go:88
    l.holes after prune: [{From:13 To:16}]
    caller's Stat().Holes[0] before={From:5 To:8} after={From:13 To:16} (len 2)
```

**Fix (apply now).** `kept := make([]Hole, 0, len(l.holes))` at retention.go:245. Test: the
reproduction under `-race` plus an assertion that a `Stat().Holes` slice taken before the prune
is unchanged after it.

## 11. correctness-29: the active segment's descriptor leaks when `load` fails at its final directory fsync (CONFIRMED)

**Anchors.** load.go:409 sets `l.active = a` (`a.f` opened read-write by `openActive` at
:318); every failure up to the publish closes `f` (:414-417) but `return l.fs.SyncDir(l.dir)` at
:418 propagates a `SyncDir` error with `f` open; `Open`'s handler is only `_ = lock.Close();
return nil, err` (log.go:300-303); the only `l.active.f.Close()` on an Open failure path is the
protected-consumer branch at log.go:312. The real backend returns fsync errors from `SyncDir`
(internal/fsx/os.go:61-70).

**Skeptic (CONFIRMED).** Reproduced the leak (one Sim handle per failed `Open`, three after
three) and isolated it to load.go:418 by applying the fix and seeing the count drop to zero; no
existing test checks `OpenFiles()` after a failed `Open` (iter_test.go:184-210 covers iterators;
recovery_test.go:334 injects `OpSyncDir` only during `Flush`). Refinement: if `publishLocked` at
:414 rotated, `l.active.f` may no longer be `f`, so closing from `Open` is the more robust shape.

**Reproduction** (`scratchpad/quillcopy/zz_leak_test.go`): open a Sim log, append 3, flush,
close (`OpenFiles()==0`); `s.Inject` returns `syscall.EIO` for `fsx.OpSyncDir`; call
`Open(simConfig(s, "/log"))` three times.

```
GOWORK=off go test -run TestZZOpenSyncDirFailureLeaksActiveDescriptor -v .
    open handles before Open: 0 / after failed Open #1: 1 / #2: 2 / #3: 3   --- FAIL
with the fix: after failed Open #1: 0 / #2: 0 / #3: 0   --- PASS
```

**Fix (apply now).** In `Open`, on a `load` error close `l.active.f` when `l.active != nil &&
l.active.f != nil` before `lock.Close()` (covers the post-rotation case too); or at load.go:418
`if err := l.fs.SyncDir(l.dir); err != nil { _ = f.Close(); return err }`. Test: the reproduction
asserting `s.OpenFiles() == 0` after each failed `Open`.

## 12. correctness-30: `Log.view` rebuilds the sealed-segment view on every read, under the log mutex (CONFIRMED, PERF)

**Anchors.** read.go:259-306: `view()` takes `l.mu.Lock()` with `defer` (:260-261) and the loop at
:268-285 allocates a `segRef` and a closure per good sealed segment; every read path goes
through `scanSource` -> `src.view()` (read.go:136), and stat.go:42, consumer.go:287 and seek.go:37
call `view()` directly; `Append` (append.go:24) and the flusher (flush.go:17, :179, :211) lock
`l.mu` and wait behind it.

**Skeptic (CONFIRMED).** Reproduced the finder's figures exactly: `Read(head,1)` costs 9 allocs
at 0 sealed, 33 at 10, 417 at 200, 2019 at 1000 (`view()` alone 5/29/413/2015). Nothing bounds
the sealed count by default (Retention -1, MaxBytes 0); no decision describes a per-call
rebuild as deliberate; sealed `segRef`s are immutable (fields set only at load; `l.sealed`
replaced wholesale at flush.go:163 and retention.go:243), so a shared cache is sound. At the
default 8 MiB segment size 1000 sealed segments is an 8 GiB log and ~40 µs under the lock, a
modest cost.

**Reproduction** (`scratchpad/quillcopy/zz_skeptic_test.go`): Sim log, SegmentBytes 1 MiB, one
record and `Roll()` per sealed segment, `testing.AllocsPerRun(50, ...)` over `l.view()`+release
and `l.Read(head, 1)`.

```
GOWORK=off go test -run TestSkepticViewAllocs -v -count=1 .
    sealed=0 view allocs/op=5 Read(head,1) allocs/op=9
    sealed=10 view allocs/op=29 Read(head,1) allocs/op=33
    sealed=200 view allocs/op=413 Read(head,1) allocs/op=417
    sealed=1000 view allocs/op=2015 Read(head,1) allocs/op=2019
```

**Why deferred.** Not a defect, and the fix is a cache with invalidation on the hot read path
(a `[]*segRef` for the sealed segments rebuilt where `l.sealed` changes, shared across views),
larger than a local fix; it should land as its own change with an allocation benchmark.
