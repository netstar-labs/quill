# A1 adversarial audit, round 2: correctness

Date 2026-10-29, against HEAD 1da37fc. Dimension report for the four correctness lenses (write
path, recovery and retention, reader and CLI, concurrency and crash): 37 items, ranked by severity
and then by reachability. The top line and the coverage statement are in
[a1-round2-findings.md](a1-round2-findings.md); this file carries each item as Claim, Grounds,
Warrant, Backing and Rebuttal, with its verdict, reproduction and disposition. Held items are
repeated with a recommended answer in [stage2-owner-decisions.md](../../decisions/stage2-owner-decisions.md).

Verdicts: CONFIRMED, PLAUSIBLE, REFUTED, or prior-deliberate (decided in round 1; no round-2
votes). Votes: C confirmed, P plausible, R refuted. Reproductions ran in throwaway copies of the
tree under `scratch/<label>/`; the Linux box is the x86-64 Debian 13 ext4 machine. Anchors are to
the audited tree 1da37fc. Where a claim is not self-evident it is labelled verified (run or read
exactly by an auditor or skeptic), reported (a named document says it) or inferred.

Tally: 15 fixed, 1 fix-failed, 13 held, 3 refuted, 5 round-1-decided.

| # | id | anchor | sev | verdict | votes | disposition |
|---|----|--------|-----|---------|-------|-------------|
| 1 | A1-42 | cmd/quill/read.go:131-156 | high | CONFIRMED | CC | fixed 4337252 |
| 2 | A1-43 | reader.go:299-311 | high | CONFIRMED | CC | fixed a4f92e8 |
| 3 | A1-80 | log.go:325-333 | high | CONFIRMED | CC | held |
| 4 | A1-33 | retention.go:242-249 | medium | CONFIRMED | CC | fixed c2a7365 |
| 5 | A1-60 | reader.go:250-265 | medium | CONFIRMED | CC | fixed c8fb07b |
| 6 | A1-46 | cmd/quill/read.go:133 | medium | CONFIRMED | CC | fixed 9c6a8c3 |
| 7 | A1-51 | stat.go:138-186 | medium | CONFIRMED | CC | fixed 89211cc |
| 8 | A1-25 | flush.go:104-130 | medium | CONFIRMED | CC | held |
| 9 | A1-32 | consumer.go:73-79 | medium | CONFIRMED | CC | held |
| 10 | A1-47 | cmd/quill/verbs.go:159-169 | medium | CONFIRMED | CC | held |
| 11 | A1-48 | cmd/quill/verbs.go:71 | medium | CONFIRMED | CC | held |
| 12 | A1-82 | read.go:249-257 | medium | CONFIRMED | CC | held |
| 13 | A1-85 | consumer.go:313 | medium | CONFIRMED | CC | held |
| 14 | A1-34 | consumer.go:245-260 | medium | REFUTED | RCR | refuted |
| 15 | A1-49 | cmd/quill/write.go:19-41 | medium | REFUTED | CRR | refuted |
| 16 | A1-59 | read.go:295-296 | medium | prior-deliberate | - | round-1-decided; revisit |
| 17 | A1-26 | append.go:61-77 | low | CONFIRMED | CC | fixed 890c70e |
| 18 | A1-31 | retention.go:225-250 | low | CONFIRMED | PC | fixed e2f7801 |
| 19 | A1-37 | load.go:339-348 | low | PLAUSIBLE | CRC | fixed 89073cb |
| 20 | A1-64 | archive.go:74 | low | CONFIRMED | CC | fixed 5b486d2 |
| 21 | A1-53 | reader.go:42 | low | CONFIRMED | CC | fixed 8f29e05 |
| 22 | A1-61 | flush.go:226-229 | low | CONFIRMED | CC | fixed 3208dab |
| 23 | A1-62 | retention.go:94-274 | low | CONFIRMED | CC | fix-failed |
| 24 | A1-63 | archive.go:96 | low | CONFIRMED | CC | fixed 7672a8d |
| 25 | A1-65 | archive.go:146 | low | CONFIRMED | CC | fixed db30a94 |
| 26 | A1-78 | cmd/quill/verbs.go:45-47 | low | CONFIRMED | CC | fixed 8449425 |
| 27 | A1-29 | log.go:338-358 | low | CONFIRMED | CC | held |
| 28 | A1-35 | retention.go:111-123 | low | CONFIRMED | CC | held |
| 29 | A1-50 | cmd/quill/print.go:81-107 | low | CONFIRMED | CC | held |
| 30 | A1-52 | read.go:161-242 | low | CONFIRMED | CC | held |
| 31 | A1-57 | cmd/quill/read.go:81-82 | low | CONFIRMED | CC | held |
| 32 | A1-91 | state.go:300 | low | PLAUSIBLE | PRC | held |
| 33 | A1-27 | flush.go:61 | low | REFUTED | CRRR | refuted |
| 34 | A1-30 | flush.go:17-24 | low | prior-deliberate | - | round-1-decided |
| 35 | A1-54 | reader.go:281-303 | low | prior-deliberate | - | round-1-decided |
| 36 | A1-56 | reader.go:337-352 | low | prior-deliberate | - | round-1-decided |
| 37 | A1-58 | cmd/quill/consumers.go:142-147 | low | prior-deliberate | - | round-1-decided |

## 1. A1-42: `quill tail -follow` prints nothing until 4 KiB has accumulated (high, CONFIRMED, fixed)

**Claim.** `quill tail -follow`, and any follow or read to a pipe or terminal, prints nothing
until 4 KiB of output has accumulated or the process is interrupted: the printer is a
`bufio.Writer` flushed only at return.

**Grounds.** cmd/quill/read.go:131-133 (`p := newPrinter(...); defer p.flush()`); the loop at
read.go:141-156 never flushes between records; cmd/quill/print.go:29 and :72 (`bufio.NewWriter`,
`p.w.Write`).

**Warrant.** Follow waits for new records by design, but each record only reaches a 4096-byte
userspace buffer, so a human or a script following a quiet log sees nothing for an unbounded time.

**Backing.** scope.md "Command line" (`tail` with -follow); the verb help text "keep printing new
records until interrupted" (cmd/quill/verbs.go:72); the latency promise of decision 50.

**Rebuttal.** A busy log that produces over 4 KiB between polls shows output in 4 KiB steps, and
non-follow reads flush at exit, which is fine. No test exercised timing, so the suite passed.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS and on the Linux box by both skeptics. Program: a one-record
log; `quill -dir L tail -follow > f.out &`; append a second record; after 2 s `cat f.out`. Output:
empty; after SIGINT it holds both lines. A second run piped through `while read` showed every line
arriving in the SIGINT second although the appends were 2 s apart.

**Disposition.** Fixed in commit 433725221e3f. In follow mode `printRecords` now flushes the printer
after each record (read.go), so output reaches the pipe or file before the iterator blocks; the
test `TestTailFollowPrintsAtOnceNotAtExit` and a CHANGELOG line were added. Fix-skeptic
FIXED-VERIFIED: the test fails with the old read.go (empty before the interrupt) and a pipe run
after the fix showed arrivals matching the appends on macOS and Linux. Residual: the test asserts
the record present at follow start, not one appended later (the binary run covers that); one write
syscall per record in follow mode is the accepted cost.

## 2. A1-43: a quill.tail `off` of 2^63 or more panics every Reader call (high, CONFIRMED, fixed)

**Claim.** A quill.tail whose `off` is 2^63 or more makes every Reader call (the OpenReader-based
verbs stat, cat, read, tail, ls and the consumers' reads) panic with a slice-bounds runtime error
instead of returning ErrCorrupt; Go's panic exit status 2 also collides with the documented
"usage" code.

**Grounds.** reader.go:299-311 compares `size < int64(tl.Off)` (a negative int64 never exceeds
size) and then builds `window{size: int64(tl.Off)}`; load.go:87-89 and :112-114
(`avail := int(min(w.size-pos, maxVarint))` goes negative and the slice bound panics); state.go:98-136
accepts any 1 to 19 digit value.

**Warrant.** The marker is on-disk input, and its `Off` flows through an unchecked uint64 to int64
conversion into a slice index, so any library user's process can crash, not only the CLI.

**Backing.** Guarantee 7 and SECURITY.md ("parsed strictly; unknown shapes refused"); guarantee 6
(corruption is an error, not a crash); decision 53 (exit codes).

**Rebuttal.** It needs a hand-edited or hostile quill.tail; ordinary bit rot of a short offset lands
below int64 and takes the existing "segment holds N bytes, the marker says M" path. SECURITY.md
excludes a process with write access to the directory, so the severity is robustness in a library
embedded in long-running services.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS (both skeptics) and Linux (auditor). Program: a valid log from
`quill append a b c`, then `printf '{"seq":3,"seg":0,"off":9300000000000000000,"maxseg":0}\n' >
L/quill.tail; quill -dir L stat`. Output: `panic: runtime error: slice bounds out of range
[:-9146744073709551616]`, exit 2, from `(*Reader).view`; the same for `cat`; 2^63 and 2^64-1 panic
too, and 2^63-1 gives the clean ErrCorrupt (the control). The auditor's example used `seg` 1, which
fails earlier on a fresh log; `seg` 0 reproduces. One skeptic saw `check` print HEALTHY for these
offsets (the finding said "hole"); `recover` was not tested by him.

**Disposition.** Fixed in commit a4f92e8c21cf. `decodeTail` (fast and slow paths) refuses an `off`
above `maxSegmentBytes` (1536 MiB), so such a marker is ErrCorrupt from OpenReader, the Reader
calls and the writer's open. Fix-skeptic FIXED-VERIFIED: the test panics without the fix; stat,
cat, read, tail, check, recover, consumers and roll all report a corrupt-log error with exit 1 on
macOS and Linux for offsets 2^62, 2^63, 2^64-1 and 20 digits; a 15 s fuzz of FuzzDecodeTail passed.
Residual: `check` and `ls` now treat an `off` above 1.5 GiB like any malformed marker (exit 1, log
omitted from `ls`) where they used to report UNHEALTHY; decision 105's text omits the new bound.

## 3. A1-80: a live protected consumer stops the writer from restarting (high, CONFIRMED, held)

**Claim.** `Config.ProtectedConsumers` makes Open (and `quill prune`, which feeds the policy back
in) fail with ErrLocked whenever a live consumer process holds that consumer's lock, so a writer
cannot restart while its protected consumer is running.

**Grounds.** log.go:325-333 calls `consumerStore().open(name, ProtectOn)` for every
ProtectedConsumers name; consumer.go:81 `takeLock` retries 200 ms then returns ErrLocked; any error
aborts Open. cmd/quill/write.go:36 then prints "(a writer holds the log; this verb needs its lock)"
for what is a consumer lock. No test holds the lock while reopening with ProtectedConsumers (T9
checks creation only).

**Warrant.** The consumer lock is per name and is held for a consumer process's whole life; Open
takes that lock only to create the file if absent, but never skips an existing file. Decision 67
(protect a lazy consumer before its process starts) therefore breaks as soon as the consumer is
running and the writer is redeployed.

**Backing.** scope.md Config row ProtectedConsumers and decision 67; guarantee 5; decision 68
(prune borrows only the log's lock); decision 99 (offline verbs open with no protected consumers,
which `prune` does not do).

**Rebuttal.** Does not hold if the consumer process has exited. Workaround: drop
ProtectedConsumers from the Config after the first run. Data is not lost; the writer cannot start.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS (both skeptics; the auditor on macOS and Linux ext4). Program:
`Open(Config{ProtectedConsumers: [billing]})`; a Reader takes `Consumer("billing")`; Close the Log;
reopen with the same Config. Output: `protected consumer "billing": quill: locked by another
holder ...billing.lock`, `errors.Is(ErrLocked)` true, 202 to 233 ms. CLI: with another process
holding consumers/billing.lock, `quill prune` exits 1 with the wrong message while `quill roll`
succeeds. Linux was not re-run by a skeptic (inferred identical: flock).

**Disposition.** Held for the owner. Recommended: approve the fix; see the owner file. No fix
written.

## 4. A1-33: a vanished sealed segment wedges retention (medium, CONFIRMED, fixed)

**Claim.** If a sealed segment's file has been removed while the writer is live, every later
deletion fails on that segment, and retention, including the MaxBytes and MinFreeBytes caps, stops
until the process restarts.

**Grounds.** retention.go:242-249: a `Remove` failing with ENOENT sets `bgErr`, logs, `n = i`, and
breaks; deletion is a prefix (`l.sealed[:n]`), so with the oldest file missing n is 0 on every pass.

**Warrant.** One permanently failing entry at the front blocks everything behind it; the cap
computation keeps counting the missing file's recorded size.

**Backing.** Scope: MaxBytes is "the real rolling guarantee"; docs/format.md: a segment "deleted by
hand" is a recognised state, reported as a hole.

**Rebuttal.** Deleting a segment under a live writer is unsupported and the failure is signalled
(a Warn log, a sticky `Err()`); a restart repairs it. Other Remove errors (EACCES, EIO) are
properly retried, so only ENOENT is wrong.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS by both skeptics. Program: MaxBytes 4096, SegmentBytes 1024,
SyncAppends, 60 appends of 100 bytes (steady state 4 segments), `os.Remove` of the oldest sealed
file, then 500 more appends and `Prune(true)`. Output: `files 53`, `err remove .../0000000003.log:
no such file or directory`, about 14 times the cap; after Close, reopen and Prune, 4 files.

**Disposition.** Fixed in commit c2a73656cf99. `deleteLocked` treats `errors.Is(err,
fs.ErrNotExist)` from Remove as already deleted (a warning, the segment counts as removed, `Err()`
unset); other errors keep the old path. Fix-skeptic FIXED-VERIFIED: the regression test fails in
both subtests without the fix; an independent real-filesystem test (oldest, then a middle, segment
removed) gave 4 files and nil `Err()` on macOS and Linux. Residual: a hand-removed segment logs
both a Warn and an Info "deleted segment", and `Dropped` still counts for consumers behind it.

## 5. A1-60 (with A1-44): a transient scan error is cached for the Reader's life (medium, CONFIRMED, fixed)

**Claim.** One transient I/O error (EMFILE, EIO, EACCES) while a Reader scans a sealed segment is
cached through `sync.Once`, so every later read of that range returns a `*GapError` although the
file is healthy. Decision 94 fixed this for the first-frame probe but not for the scan.

**Grounds.** reader.go:250-265: `scan()` runs the open and `verifySegment` in `s.once.Do` and stores
any error in `s.scanErr`; read.go:173-176 turns an ensureIndex error into `&GapError{From: cur,
To: s.last}`; the readerSegment stays in `r.segs`.

**Warrant.** `once.Do` cannot be re-armed, so the first outcome is final. A long-lived Reader
(Follow, consumers) hit by one EMFILE burst turns a healthy range into a permanent hole until
restart, and a consumer applying the documented rule resumes at `To + 1` and skips intact records.

**Backing.** Decision 94 ("damage is a fact about the file; an I/O error is a fact about the
moment"); guarantees 6 and 14; decision 63.

**Rebuttal.** It needs a non-damage error during the scan of an unindexed sealed segment; the
writer side is unaffected (its refs carry an index from load); ENOENT from retention is correctly
permanent; a Reader restart repairs it.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS and Linux. Program: an FS wrapper fails one open of segment 0
with EMFILE (the second open, since the first is the first-frame probe); the first Read returns
`records 1 to N are unavailable`; after the failure is disarmed three more Reads on the same Reader
still do; a fresh Reader reads them fine. The same error in the probe open recovers on the next
Read (decision 94), so the scan path alone was the exception.

**Disposition.** Fixed in commit c8fb07be6440. `readerSegment.scan` replaces `sync.Once` with a
mutex and caches a result or a `*SegmentError` only; other errors are returned for that call and
retried; ENOENT also forgets the entry. Fix-skeptic FIXED-VERIFIED: the test fails on the old
reader.go; an independent EIO-on-pread case and an eight-goroutine -race case passed after and
failed before; the Linux box passed under -race. Residual: the first read after the error still
returns a GapError (decision 94's "for that call"), and a persistent non-damage error re-scans on
every read; the new `forget` line was verified by reading only.

## 6. A1-46: output write failures are invisible to the exit code for outputs under 4 KiB (medium, CONFIRMED, fixed)

**Claim.** `quill cat`, `tail`, `read`, `stat` and `check` to `/dev/full` (or a closed stdout) exit
0 with the records lost for any output under 4 KiB.

**Grounds.** cmd/quill/read.go:133 `defer p.flush()` and print.go:32 `_ = p.w.Flush()` discard the
flush error; it surfaces only when the 4096-byte buffer overflows. Other verbs ignore the error of
`fmt.Fprint*` (read.go:84, 199-206; consumers.go:41, 43; verbs.go:211).

**Warrant.** Exit 0 is documented as "ok"; `quill cat -json > out.jsonl && rotate` treats a failed
copy as success.

**Backing.** scope.md "Locks and exit codes" (0 ok, 1 error); decision 100 states the same
principle for `append`'s Close.

**Rebuttal.** It matters only for a full disk, a closed descriptor or an unwritable file; a pipe
closed by the reader dies from SIGPIPE first (exit 141, verified).

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on the Linux box (both skeptics). Program: three records, then `quill
cat > /dev/full`, `stat`, `check`, `tail -json`, `consumers`. Output: exit 0 and no stderr for all;
a 3,000-record `cat` exits 1 with `write /dev/stdout: no space left on device`. The auditor's macOS
`>&-` case was never a failure: the Go runtime substitutes /dev/null for a descriptor closed at exec
(verified by the fix-skeptic with a probe program), so only a descriptor that is open but
unwritable fails.

**Disposition.** Fixed in commit 9c6a8c3e1a46. `run()` wraps stdout in a sticky-error writer and
turns an otherwise-OK exit into exit 1 with the error; `tail -follow` stops when its per-record
flush fails. Fix-skeptic FIXED-VERIFIED: the 12 new test cases fail without the fix; every verb
exits 1 on `/dev/full` on the Linux box; a verb that already returned non-zero keeps its code
(`check`'s 3). Residual: a stale duplicated doc line above `run` in verbs.go; the CHANGELOG and
userguide say a "closed descriptor" exits 1, which holds for an unwritable one but not for stdout
closed at exec; staticcheck could not run locally.

## 7. A1-51: `Reader.Check` reports a healthy log UNHEALTHY when retention deletes a segment mid-scan (medium, CONFIRMED, fixed)

**Claim.** `Reader.Check` (and `quill check`, which takes no lock and is meant to run beside a live
writer) reports a healthy log UNHEALTHY (exit 3) when retention deletes a listed segment before
Check opens it, and a bogus hole when a middle segment vanishes.

**Grounds.** stat.go:138-186: one ReadDir, then each openSegment error becomes
`fatal(Problem{Kind: "segment", ...})` (stat.go:152-155) with no distinction for `fs.ErrNotExist`,
and the contiguity test at stat.go:182-184 compares each survivor with its predecessor.

**Warrant.** A sealed segment deleted between listing and open is exactly the condition decision
101 maps to a gap for reads; Check calls it damage, with the wrong text "damaged at byte 0". The
window is as long as a full scan.

**Backing.** Decision 101; scope.md (`check` takes no lock); decision 53.

**Rebuttal.** It needs retention to run during the scan; the real race was not reproduced by the
auditor, only injected. A vanished middle segment is a real hole unless retention explains it.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by both skeptics; one on real timing. Program: a Sim log of three sealed
segments and an active one, a segment removed as its open begins; `r.Check()` gives `healthy=false`
with `damaged at byte 0 ... file does not exist`, and for a middle segment also `hole ...
sequences 5 to 8 are missing before it`. Real filesystem: a live writer with MaxBytes retention and
a looping Check gave 14 unhealthy results in 15,922 checks.

**Disposition.** Fixed in commit 89211cc2624f. Check skips a sealed segment (id below the marker's)
whose open fails with `fs.ErrNotExist` after the listing and resets `havePrev`; a segment missing
from the listing is still a hole and the marker's own segment missing is still fatal. Fix-skeptic
FIXED-VERIFIED: both subtests fail without the fix; a real-filesystem wrapper test returned
Healthy true for the oldest, a middle, and all-but-last segments vanishing. Residual: the skip
cannot tell retention from any other deletion during the scan (the next run catches a real hole);
the Check doc comment does not mention it.

## 8. A1-25 (with A1-81): while rotation keeps failing, the active segment can outgrow the 1.5 GiB limit (medium, CONFIRMED, held)

**Claim.** While rotation keeps failing, the writer keeps appending to the active segment with no
bound, so it can pass the 1.5 GiB limit that Open and readers enforce. The log then cannot be
opened although every record in it was acknowledged durable.

**Grounds.** flush.go:104-111 records a rotation error in `bgErr` and carries on; flush.go:130 keeps
`pendingPublish` set; nothing in append.go:51-85 or flush.go:25-67 looks at the active segment's
size. load.go:19 sets `maxSegmentBytes = 1536<<20` and load.go:76-77 makes `openActive`, and so
Open, refuse a larger file. The same overshoot is reachable with no fault: SegmentBytes of 1 GiB
plus a FlushBytes or MaxBufferBytes above 512 MiB (neither is capped by validate, log.go:137-169;
read, not run).

**Warrant.** Decision 60(d) lets the log keep appending past SegmentBytes after a failed rotation
with no ceiling; decision 16 and docs/format.md:57 refuse a file over 1.5 GiB. Rotation can fail
(EMFILE, ENFILE, a permission change, inode limits) while writes to the already-open descriptor
keep working.

**Backing.** Decision 60(d); decision 16; decisions 46 and 69 and scope.md (a stated disk bound);
guarantee 7.

**Rebuttal.** Decision 60(d) chose this on purpose and `Err()` reports it; reaching the limit needs
a fault lasting about 1.5 GiB of writes, which is hours at default size and moderate load.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS by both skeptics (the auditor's program). Program: SegmentBytes
1 MiB, the log directory chmod 0500 after Open, 400 rounds of `Append(4 MiB)` plus `Flush`. Output:
`active segment is 1677729073 bytes (SegmentBytes 1 MiB), head 400, Err ...permission denied`;
`Append` returned nil throughout (the Flush calls returned the error); after Close and chmod 0700,
Open returns `segment .../0000000000.log damaged at byte 0 (intact to byte 0): 1677729073 bytes,
over the 1610612736 limit`. quill.tail stayed at seq 0. Linux and the no-fault variant not run.

**Disposition.** Held for the owner. Recommended: approve a ceiling on the active segment. See the
owner file. No fix written.

## 9. A1-32 (with A1-86): the reserved `_archive` consumer cannot be committed, registered, unregistered or deleted (medium, CONFIRMED, held)

**Claim.** The reserved `_archive` consumer cannot be moved or removed through the public API or the
CLI. After one `quill archive` it protects the segments at and above its offset, and after a
restore that leaves it ahead of the head it wedges the archiver with no supported remedy.

**Grounds.** consumer.go:73-79 skips `validName` for the underscore name only when `internal` is
true and only Archive passes it; consumer.go:267-269 `remove` calls `validName` unconditionally;
log.go:182-184 rejects a leading underscore; cmd/quill/consumers.go:59 and :131 use the public
paths. retention.go:126-137 keeps every segment at or above a protected consumer's offset.

**Warrant.** A protected consumer whose offset file cannot be moved holds retention at its offset;
when it is ahead of the head Archive fails at open with ErrAhead and the userguide's remedy
(`quill commit`) is rejected by name validation for exactly this consumer.

**Backing.** Decision 57 (protection is per consumer); decision 36 (ErrAhead); userguide section 9
(an ahead consumer "must be rewound with `quill commit`"); decision 67 (the TTL off by default makes
the pin permanent by default).

**Rebuttal.** `rm consumers/_archive` or a ConsumerTTL clears it (undocumented for this consumer);
the caps still bound disk; after an archive the offset is at head+1, so it pins only segments
written since; the restore wedge needs `consumers/` restored from a later state than the segments.

**Verdict.** CONFIRMED by 2 of 2 votes, both narrowing the "pins for good" wording.

**Reproduction.** Verified on macOS (both skeptics). Program: `quill archive -dir u1 -to u1a` exits
0 and `quill consumers` lists `_archive ... protected=true`; `unregister`, `register` and `commit
-name _archive -offset 1` each exit 1 with `a leading underscore is reserved`. Rewriting
consumers/_archive to `<id> 99 1` gives `quill archive` exit 1 with ErrAhead and `commit` refused
again; deleting the file by hand clears it. A restore sequence producing the state was not run.

**Disposition.** Held for the owner. Recommended: approve. See the owner file.

## 10. A1-47: extra positional arguments are ignored by every verb but `append` (medium, CONFIRMED, held)

**Claim.** `quill -dir L read 100` prints from the first retained record and `quill -dir L tail 5
-json` prints text, both with exit 0, because positionals are never checked and Go's flag parsing
stops at the first positional.

**Grounds.** cmd/quill/verbs.go:159-169 (`fs.Parse(rest[1:])` then `v.run(c, fs.Args())`); the run
functions (read.go:115, 160, 74, 180; consumers.go; write.go:118, 164, 173) never read `args`.

**Warrant.** A script gets a different result than asked for, with status 0; the user's intent is
dropped without a message.

**Backing.** scope.md exit codes (2 is "usage"); `read` takes the offset only as `-from`.

**Rebuttal.** The documented grammar is flags only, so a strict reading says the user erred; a
change turns a previously successful, misdirected invocation into exit 2.

**Verdict.** CONFIRMED by 2 of 2 votes. One caveat: `help` takes a positional (`help read`), so it
must be exempt along with `append`.

**Reproduction.** Verified on macOS (both skeptics). Program: 13 appended records; `read 10` starts
at seq 1 with exit 0; `tail 5 -json` prints ten text lines and no JSON; `stat -json extra` exits 0;
`version foo` exits 0; `read -from 10` is correct (the control).

**Disposition.** Held for the owner. Recommended: approve (a usage error, exit 2).

## 11. A1-48 (with A1-75): `-from`, `-limit` and `-n` parse numbers in base 0 (medium, CONFIRMED, held)

**Claim.** `-from`, `-limit` and `-n` use `flag.Uint64Var` (base 0), so `-n 010` is 8 records,
`-from 0x10` is 16 and `-limit 1_0` is 10, while `-offset`, `-max-bytes`, `-min-free-bytes`,
`-max-records` and `@unix` times are strictly decimal: the same text means different numbers.

**Grounds.** cmd/quill/verbs.go:71 and :124-125 versus read.go:214-218 (`parseDecimal`, comment
"digits only ... `0x10` ... are all refused").

**Warrant.** A zero-padded script variable silently becomes octal for three flags.

**Backing.** cmd/quill/numbers_test.go's refused-numbers table and `parseDecimal`'s comment state
the intended rule; decision 96 and userguide.md:183-184 list only the other flags.

**Rebuttal.** It is the standard flag package's documented behaviour and hex may be a feature; only
the octal reading of `010` is plainly dangerous. The harm is limited to read-only verbs.

**Verdict.** CONFIRMED by 2 of 2 votes (a skeptic rates it low rather than medium).

**Reproduction.** Verified on macOS (both skeptics). Program: 12 records; `tail -n 010` gives 8
lines, `tail -n 10` 10 lines, `tail -n 08` exit 2, `cat -from 0x2` starts at seq 2, `cat -limit 1_0`
10 lines, `commit -name x -offset 010` gives `x 10`, and `-offset 0x10` is refused with exit 2.

**Disposition.** Held for the owner. Recommended: approve (a strict-decimal flag value).

## 12. A1-82: an I/O error while reading a segment is reported as ErrCorrupt or a whole-segment gap (medium, CONFIRMED, held)

**Claim.** An I/O error while reading a segment is reported as ErrCorrupt (active segment) or as a
`*GapError` over the whole segment (sealed segment), contradicting decision 79 and making a
transient fault look like loss that a documented resume idiom skips.

**Grounds.** read.go:249-257: any error from `walk` (frame errors and I/O errors alike) becomes
`ErrCorrupt: the active segment below the durable offset: %v` (the cause flattened, so
`errors.Is(EIO)` is false) for the active segment and `&GapError{From: cur, To: s.last}` for a
sealed one. Only the load path has a test (load_test.go:291).

**Warrant.** Decision 79 says an I/O error met while scanning is returned as that error and never as
ErrCorrupt. scope.md says "a `*GapError` is resumed by iterating again from `To + 1`", so a consumer
following the idiom after one EIO silently skips a segment, against the intent of guarantee 14.

**Backing.** Decisions 79, 38; scope.md guarantees 6 and 14 and the Iterators section.

**Rebuttal.** Decision 94 maps open-time I/O failures to a per-view hole, so a reader could call the
sealed case accepted (a skeptic notes decision 38 lists "unreadable" among hole causes). That does
not cover the active segment's ErrCorrupt, which decision 79 forbids.

**Verdict.** CONFIRMED by 2 of 2 votes; the sealed half is arguable, the active half is not.

**Reproduction.** Verified on macOS (both skeptics) with `fsx.Sim` returning EIO on `OpRead`.
Output: a sealed segment gives a `GapError` over its range (for example 1 to 5, 12 to 15, 20 to 20
at three offsets) with no ErrCorrupt and no EIO match; the active segment gives `corrupt log: the
active segment below the durable offset: input/output error`, `Is(ErrCorrupt)` true, `Is(EIO)`
false. Records and Scan share the path (inferred, not run); a real `os.File` EIO is inferred the
same.

**Disposition.** Held for the owner. Recommended: approve (return non-frame errors unwrapped).

## 13. A1-85 (with A1-77): consumer files created through a Reader or the CLI ignore the log's Perm (medium, CONFIRMED, held)

**Claim.** Consumer files created through a Reader or the CLI, and archive destinations, ignore the
log's Perm and are always 0600, so in the group arrangement the scope describes, the writer cannot
read them and retention treats the consumer as broken.

**Grounds.** consumer.go:313 (`Reader.consumerStore` perm 0o600); archive.go:25 hard-codes 0o600;
`ReaderOptions` has no Perm. retention.go:126 and decision 67 treat a consumer file the writer
cannot read as broken (every segment protected; `check` UNHEALTHY).

**Warrant.** scope.md says a reader in another account needs a group arrangement and a consumer
process needs write access to `consumers/`; a consumer process in that arrangement creates a file
only it can read.

**Backing.** scope.md Config row Perm and "Filesystems, permissions and backup"; decisions 41, 51,
67.

**Rebuttal.** Single-user deployments are unaffected; the documents never promise that a Reader
honours Perm. The failure is on the safe side (everything protected, `check` UNHEALTHY).

**Verdict.** CONFIRMED by 2 of 2 votes. The Archive destination half is less clear-cut (a separate
log created without a Config).

**Reproduction.** Verified, with the cross-user step run on the Linux box by one skeptic. Program
(macOS, umask 0, `Config{Perm: 0660}`): `consumers/vialog` is `-rw-rw----`, `consumers/viareader` and
its lock `-rw-------`. Linux: a writer with Perm 0660 in a setgid group directory (2770); a reader
run as another user in the group (`setpriv`) created `viareader` as 0600; the writer's
`Consumers()` then returned `permission denied` for it, and after 2.5 s `Prune(true)` kept segment
0, which the same run without that consumer had deleted. The CLI `register` goes through the same
code path (inferred from reading).

**Disposition.** Held for the owner. Recommended: approve.

## 14. A1-34 (with A1-92): any non-lock file in consumers/ is treated as a consumer (medium, REFUTED)

**Claim.** A stray `.DS_Store`, an rsync temp file or an editor backup in `consumers/` makes
`broken` true, disables the age sweep with no log line, and makes `check` report UNHEALTHY.

**Grounds.** consumer.go:245-248 skips only names ending `.lock` or starting `.tmp-`;
consumer.go:250-260 sets `info.Err` for an undecodable file; retention.go:38-41 sets `cv.broken`
(no `logf`); retention.go:126-131 sets `minProtected` to the first retained sequence.

**Warrant, backing.** Decision 67: a consumer file the writer cannot read protects everything and
`check` reports it.

**Rebuttal.** The same decision makes exactly this the documented fail-safe; `check`, `quill
consumers` and the caps still work; the "unknown files are ignored" sentence in format.md:22
covers the log directory, not `consumers/`.

**Verdict.** REFUTED (votes R, C, R). The mechanism is real and was reproduced on macOS (60 records
across 7 segments, clock +48 h, a `consumers/.DS_Store`: 3 forced prunes left 7 segments with nothing
logged; after removing the file one prune left 1; `quill check` exit 3 with the file, exit 0 without).
The two refuting skeptics hold that it is the documented fail-safe; the confirming skeptic counts the
silence and the name check never applied to directory entries as the defect.

**Disposition.** Not changed: refuted as a defect; the one-line reason is that decision 67 intends
the protection. The silence on the prune path remains a minor observability gap; a Warn each pass
while `broken` is set would close it (not done).

## 15. A1-49: `roll`, `prune` and `recover` against a mistyped -dir create a new log (medium, REFUTED)

**Claim.** The maintenance verbs run against a nonexistent `-dir` create a brand-new empty log there
and exit 0; against a non-log directory they refuse but leave a stray `quill.lock`.

**Grounds.** cmd/quill/write.go:19-41 `openWriter` calls `quill.Open`, which creates the log when the
directory is new (log.go:288-289).

**Warrant, backing.** scope.md: `recover` is "`Open` followed by `Close`"; decision 53; decision 62.

**Rebuttal.** The spec prescribes it: `Dir` is "created if missing" (scope.md:108), the verbs "open
the log writable", decision 62 has `Open` create and flock `quill.lock` "before deciding anything",
and decision 77 left that unchanged. The leftover lock file is documented.

**Verdict.** REFUTED (votes C, R, R). Reproduced on macOS by all three: `roll -dir nonexist`,
`recover -dir typo2` and `prune -dir typo3 -retention 1h` all exit 0 and create full log
directories; `roll -dir other` (holding file.txt) exits 1 "not a log" and leaves other/quill.lock.
The facts are not in dispute; the dispute is whether a decision makes them intended, and two
skeptics found that decisions 53, 62 and 77 do.

**Disposition.** Not changed: refuted as a defect (the spec prescribes it). A CLI check for an
existing log before `Open` would be a reasonable hardening and needs a new decision.

## 16. A1-59 (with A1-83): Log reads return a raw ENOENT where the Reader returns a GapError (medium, prior-deliberate)

**Claim.** `Log.Read`, `Scan`, `ScanAll`, `Records`, `Follow` and the in-process Consumer reads
return a raw `*fs.PathError` (ENOENT) instead of a `*GapError` when retention deletes a sealed
segment between the read taking its view and opening the segment. The Reader maps this (decision
101); the Log's own sealed refs do not.

**Grounds.** read.go:295-296: the open closure built in `sealedRefsLocked` returns `openSegment`'s
error unchanged; the Reader equivalent maps `fs.ErrNotExist` to `&GapError{...}`; `scanSource` only
rewrites an error that is already a `*GapError`.

**Warrant.** The view is taken once but each sealed segment is opened later, so the window is the
whole scan; retention unlinks under `l.mu` without waiting for readers (decision 103: no descriptor
held for sealed segments). A Follow loop applying the documented recovery then sees a fatal error.

**Backing.** scope.md guarantee 14; decisions 11 and 101; decision 103's own sentence "a deleted one
fails at its open as before".

**Rebuttal.** It does not hold when no retention pass overlaps a read; it is an error, not silent
loss, and a retry succeeds. Decision 103's wording can be read as making the raw error intended,
which contradicts guarantee 14.

**Verdict.** prior-deliberate (round 1, decision 103; audit-followups.md "Not changed"); no round-2
votes; raised independently twice. Auditor reproduction (verified by the auditor): a source wrapper
that takes `l.view()` then `l.Prune(true)` makes `readFrom(h,1,10)` return `lstat ...: no such file
or directory (*fs.PathError) isGap=false`; with 4 goroutines reading under -race, macOS gave ok=1519,
gap=57, rawENOENT=25 and Linux ok=1358, gap=108, rawENOENT=17, with no data race.

**Disposition.** Deliberately not changed in round 1 (decision 103). Recommendation: reopen it; the
fix is one mapping at read.go:295, amend decision 103, add a test using the hook-source technique.

## 17. A1-26 (with A1-96): at sequence exhaustion the API contradicts itself (low, CONFIRMED, fixed)

**Claim.** After the record numbered MaxUint64 is accepted, `Sync(MaxUint64)` returns ErrBadSeq
("not assigned"); with SyncAppends the `Append` that took the last sequence returns `(seq,
ErrBadSeq)` although the record is durable; a Read of the last record returns ErrCorrupt; the
ErrSeqExhausted path had no test.

**Grounds.** append.go:77 leaves `nextSeq` wrapped to 0; flush.go:227 tests `seq >= l.nextSeq`, true
for every seq once it wraps; read.go:259 (`cur <= s.last` with `cur` wrapping to 0) raises
ErrCorrupt.

**Warrant.** append.go:61 deliberately lets the last sequence be assigned, while flush.go and the
read cursor still treat `nextSeq` and `last+1` as if they could not wrap. Decision 42 defines
ErrBadSeq as "not assigned", which is false for this record.

**Backing.** Decision 42; errors.go:20 and :40; scope.md model line 3; docs/userguide.md:281.

**Rebuttal.** 2^64 records cannot be written in practice; only a crafted or restored log whose
first frame is near MaxUint64 gets there, and the next append correctly returns ErrSeqExhausted.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS (both skeptics, the auditor's program). Program: a log
hand-built so its head is MaxUint64-1, then Open, Append, Flush, `Sync(MaxUint64)`, a second Append
and `Read(MaxUint64, 5)`. Output: Append returns seq 18446744073709551615, head the same,
`Sync(MaxUint64)` returns `sequence not assigned: ... (last assigned ...)` (ErrBadSeq true), with
SyncAppends the Append itself returns it, the next append returns ErrSeqExhausted, and the Read
returns `corrupt log: segment 0 ended at ..., expected ...`.

**Disposition.** Fixed in commit 890c70e1770b. Append never assigns MaxUint64, so the ceiling is
MaxUint64-1; Open and OpenReader (via `readTail`) refuse a log whose last frame or marker is
MaxUint64 with ErrCorrupt; docs and CHANGELOG updated; a regression test file covers the boundary.
Fix-skeptic FIXED-VERIFIED: both tests fail without the fix and pass on macOS and Linux; the CLI at head
MaxUint64-1 appends refuse with "sequence numbers exhausted", `commit` at head+1 does not wrap.
Residual: the ErrSeqExhausted comment at errors.go:20 is slightly loose; a reader reports HEALTHY on
a log whose frames run ahead of the marker to MaxUint64 while the writer refuses it (benign; nothing
wraps); a multi-segment crafted layout ending at MaxUint64 was not tried.

## 18. A1-31: a multi-segment prune can leave a hole mid-log after a crash (low, CONFIRMED, fixed)

**Claim.** When one retention pass deletes several sealed segments, a power loss that persists a
later unlink but not an earlier one leaves a hole in the middle of the retained records, where
scope.md promises no gaps from the first retained record to the head, including after a crash.

**Grounds.** retention.go:225-250: `deleteLocked` unlinks oldest first with no directory fsync
between unlinks (a single sync before the loop).

**Warrant.** Under the simulator's model (decision 61: each unsynced directory operation persists
independently) deleting segments 0 and 1 can leave 0 present and 1 absent; recovery then sees 1-6
and 13 onward with 7-12 missing.

**Backing.** scope.md model line 3; decision 38; decision 65.

**Rebuttal.** Real journalled filesystems (ext4, XFS, APFS) commit directory operations in order, so
a persisted unlink implies the earlier ones (inferred, not tested); the records were due for
removal; the next prune heals it; one-at-a-time sweeps found no holes.

**Verdict.** CONFIRMED (votes P, C): one skeptic holds it reachable on real hardware only if a
supported filesystem can reorder unlinks.

**Reproduction.** Verified on the crash simulator by both skeptics. Program: 12 sealed segments,
retention expired, `Prune(true)`, a crash before each operation with random persist masks. Output:
`holes [{7 12}] files [0000000000.log 0000000002.log ...]` and `holes seen 365`; each such log
cleared the hole at its next prune. Not on a real power cut.

**Disposition.** Fixed in commit e2f7801d62ee. `deleteLocked` syncs the directory before each
segment unlink (oldest first), so the survivors are always a suffix; a SyncDir failure partway marks
the log failed after the bookkeeping for the segments already removed. Fix-skeptic FIXED-VERIFIED:
the new crash test failed without it (241 crash states left a hole); a SyncDir-failure sweep kept
the bookkeeping consistent. Residual: the SyncDir-failure branch has no test in the commit; the cost
is n directory fsyncs per n-segment prune; the real-filesystem ordering claim was not tested.

## 19. A1-37: a refused writable Open has already truncated the newest segment (low, PLAUSIBLE, fixed)

**Claim.** Writable Open truncates and fsyncs the newest segment before its remaining ErrCorrupt
checks (overlap with the previous segment; last valid sequence below the marker), so an Open that
fails with ErrCorrupt can have modified the file.

**Grounds.** load.go:339-348 (`f.Truncate(info.end)` then `f.Sync()`) runs before load.go:374-377
(overlap) and load.go:393-396 (head below the marker).

**Warrant.** Truncation is decided from the segment alone, but acceptability depends on later
checks; cutting first loses bytes a manual repair would use.

**Backing.** docs/format.md "Verification and recovery"; decisions 39 and 64; decision 77 as a
precedent that a refused Open leaves no side effects.

**Rebuttal.** The cut bytes are past the first undecodable frame and untrusted; "never modified" is
promised only for damage below the marker; no valid acknowledged frame is lost in the cases
constructed (250 random-damage Opens found no shrink below the durable offset).

**Verdict.** PLAUSIBLE (votes C, R, C). The ordering is certain from the code; the REFUTED skeptic
holds that no documented promise is broken. Not reproduced by the skeptics (the auditor's scenario
G: segment 2 replaced by a copy of segment 0 plus 2 stray bytes; Open returned `segment 2 starts at
1, inside the one before` and the file went from 1346 to 1344 bytes).

**Disposition.** Fixed in commit 89073cbbd3a9. The truncate and fsync moved after the overlap, the
head-below-marker and the sequence-ceiling checks, so an Open refused as corrupt leaves the newest
segment byte for byte as found. Fix-skeptic FIXED-VERIFIED: both subtests fail without it; the CLI
`quill recover` on a real filesystem cut 4262 to 4260 bytes before and left it at 4262 after, on
macOS and Linux. Residual: a failing Truncate or Sync after the move was not tested.

## 20. A1-64 (with A1-38, A1-45): Archive onto the live log overwrites its quill.tail (low, CONFIRMED, fixed)

**Claim.** `Archive(dir, dir)`, or a destination holding the same log (same log id), is accepted and
overwrites the live writer's quill.tail with a marker naming the last sealed segment as active,
hiding records from every Reader until the writer's next marker-changing flush.

**Grounds.** archive.go:74 writes the destination quill.tail unconditionally; `archiveDest`
(archive.go:94-126) only checks that the destination's quill.format id matches, which is trivially
true for the same directory.

**Warrant.** quill.tail has exactly one writer (decisions 8 and 62); Archive writes it without the
lock. The writer does not rewrite an unchanged marker, so the clobbered file stays until a flush
that changes it.

**Backing.** Guarantee 5, decision 8; guarantee 10 (a stale marker hides but never exposes).

**Rebuttal.** It needs an operator mistake or a live copy as destination; after a crash recovery is
correct, so the effect is transient invisibility, not loss.

**Verdict.** CONFIRMED by 2 of 2 votes. One qualifier: with no records yet in the active segment,
the clobbered marker hides nothing; the hiding appears only when flushed records sit past the sealed
segment.

**Reproduction.** Verified on macOS (both skeptics) and Linux (one). Program: 40 appends across
sealed segments plus 3 into the active one with the writer open, then `Archive(dir, dir)`. Output:
Archive returns nil, the tail changes from `{"seq":43,"seg":4,"off":81,"maxseg":4}` to
`{"seq":40,"seg":3,"off":320,"maxseg":3}`, a Reader's head is 40 while the writer's is 43.

**Disposition.** Fixed in commit 5b486d26067b. Archive refuses a destination that is the source log
(same quill.tail file by `os.SameFile`, so symlink and relative spellings included) and one whose
quill.lock is held (ErrLocked), before writing. Fix-skeptic FIXED-VERIFIED: both tests fail without
the fix; a CLI run against a real live writer went from exit 0 and a changed marker to exit 1 and an
unchanged marker. Residual: a time-of-check window (a writer opening the destination mid-copy is not
excluded); an unreadable destination quill.lock now fails Archive.

## 21. A1-53: a long-lived Reader's segment cache is never pruned (low, CONFIRMED, fixed)

**Claim.** A long-lived Reader's cache of sealed segments keeps one entry (and, once scanned, the
full index) for every sealed segment it ever listed, including ones retention deleted long ago, so
memory grows with every rotation.

**Grounds.** reader.go:42, :199-247 add entries in `segment()`; the only removal is `forget()` at
reader.go:243-247, called only when an open of that segment fails; `view()` (reader.go:289-294)
never drops ids absent from the listing.

**Warrant.** Each entry holds a readerSegment and an anchor slice of about size/64 KiB entries of 24
bytes, so a follower on a fast log with retention leaks steadily.

**Backing.** Decision 63 (cache per segment id, invalidated if the file disappears) and decision 11.

**Rebuttal.** Per-entry cost is about 100 bytes unscanned and about 3 KB scanned for an 8 MiB
segment; no data is wrong.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified by the auditor (stress: 20,000 records, 2 KiB segments, MaxRecords 30:
6 files on disk and `len(r.segs)` 184) and one skeptic (3,000 100-byte records, a `Read` every 12
appends: 124 entries against 7 files).

**Disposition.** Fixed in commit 8f29e057c272. `Reader.prune`, called from `view()`, deletes every
cached id absent from the listing when the cache is larger than the listing. Fix-skeptic
FIXED-VERIFIED: the test fails without it; a real-filesystem run went from 150 entries to 1.
Residual: the early return can leave a few stale entries while uncached I/O-error segments are
listed (bounded, dropped on later views).

## 22. A1-61: `Log.Sync` reads `l.nextSeq` after unlocking (low, CONFIRMED, fixed)

**Claim.** `Log.Sync` reads `l.nextSeq` after releasing `l.mu` to build the ErrBadSeq message: a data
race flagged by the race detector and a possibly wrong "last assigned" number.

**Grounds.** flush.go:226-229: `l.mu.Unlock()` then `fmt.Errorf(... l.nextSeq-1)`; `appendLocked`
writes `l.nextSeq` at append.go:77.

**Warrant.** A plain read of a lock-guarded field with the lock released, concurrent with a locked
write, is a race by the Go memory model.

**Backing.** log.go's `mu` comment and scope.md ("safe for concurrent use").

**Rebuttal.** On 64-bit hardware the read cannot tear, so the effect is a stale number in an error
string.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS and Linux. Program: a goroutine running 2,000 `Append`s against
one running 2,000 `Sync(ctx, 1<<40)`, under -race: `WARNING: DATA RACE`, write at append.go:77, read
at flush.go:228. It also fired in the 150-iteration chaos run.

**Disposition.** Fixed in commit 3208dab118ec. `last := l.nextSeq - 1` is captured before the
unlock. Fix-skeptic FIXED-VERIFIED: the regression test fails under -race on the reverted flush.go
on both platforms and is clean with the fix. Residual: the test is probabilistic (it fired first
run on both platforms).

## 23. A1-62: `Config.Logger` is called with the log's mutex held (low, CONFIRMED, fix-failed)

**Claim.** `Config.Logger` is invoked with the log's mutex held (`logf` is only called from
`*Locked` functions), so a handler that calls back into the Log deadlocks and a slow handler stalls
every Append, Read, Sync and the flusher. Neither the field's doc nor the user guide says so.

**Grounds.** retention.go:94, 140, 192, 200, 246, 257 call `l.logf` under `l.mu`; `logf`
(retention.go:274) calls `cfg.Logger.Log` synchronously.

**Warrant.** `Head()` takes `l.mu`, which the caller holds, and `sync.Mutex` is not reentrant; a
handler blocked on I/O holds `l.mu` as long as it blocks.

**Backing.** The Logger is a documented extension point; scope.md says the Log is safe for
concurrent use.

**Rebuttal.** Most handlers neither re-enter nor block, and the lines are rare; documenting the
restriction is a legitimate resolution.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS. Program: MaxRecords 5, 512-byte segments, a `slog.Handler`
whose `Handle` calls `l.Head()`; the writer hung and the 3 s watchdog fired with the handler entered
(`DEADLOCK: writer hung; handler entered=1`); a second handler that blocks stalled `Head()` for 2 s
until released. Linux was not re-run by a skeptic (inferred identical).

**Disposition.** Fix attempted and rejected. Commit fdcd880fe2e4 queues records under the lock and
delivers them after it (on the caller's goroutine, and for the flusher's records on a delivery
goroutine that Close waits for). The fix-skeptic reproduced two failures against it: `logf` still
calls the handler's `Enabled` under `l.mu` (retention.go:294), so an `Enabled` that calls `l.Head()`
deadlocks the log, and a handler whose `Enabled` and `Handle` share a mutex stalls the flusher
behind a blocked `Handle` (a loop of Append and Sync did not finish in 8 s; the control with a
lock-free `Enabled` finished in 0.38 s). The new text in Config.Logger, docs/userguide.md:148 and
scope.md promises more than the code gives. The rest of the fix is sound (the three new tests fail
without it and pass on macOS and the Linux box; the -race suite and goldens are unchanged). The
commit is not in this tree. Recommended: finish it by not calling `Enabled` under the lock (queue
unconditionally and filter at delivery, or call `Enabled` in the deliverer) and add a test with a
blocking `Enabled`; or take the first option and document the restriction. See the owner file,
last section.

## 24. A1-63: Archive does not fsync the parent after creating the destination directory (low, CONFIRMED, fixed)

**Claim.** After a power loss the archive destination can vanish while the `_archive` consumer's
committed offset says every segment was copied; the next Archive copies nothing and returns nil,
leaving the archive empty.

**Grounds.** archive.go:96 creates `dstDir` with Mkdir; archive.go:126 only does
`fsys.SyncDir(dstDir)`, which makes entries inside it durable but not its own entry in the parent;
`Log.mkdirAll` syncs each new directory into its parent (log.go mkdirAll).

**Warrant.** Per fsx/crash.go each unsynced directory operation persists independently; Commit
(archive.go:79) records progress the destination cannot back up, and decision 40 licenses a commit
rolling back, not running ahead of its data.

**Backing.** The `fsx.FS.SyncDir` contract; decision 62 (missing parents each fsynced into their
parent); the Archive doc.

**Rebuttal.** Local ext4 and XFS fsync commit the running transaction, which in practice includes
the parent's pending mkdir, and macOS F_FULLFSYNC flushes everything; a real loss needs a lazily
persisting destination filesystem and a power loss in that window. Not reproduced on a real cut.

**Verdict.** CONFIRMED by 2 of 2 votes (confidence moderate).

**Reproduction.** Verified on the crash simulator (macOS) by both skeptics. Program: `archive(s,
"/log", "/arch")` succeeds; `Crash` keeping every unsynced item except the root's mkdir of `/arch`.
Output: `/arch` absent after the crash, `_archive` durable at offset 21 (all 5 sealed segments), and
a second `archive()` returns nil with `/arch` holding 0 segment files.

**Disposition.** Fixed in commit 7672a8d39692. `archiveDest` calls `fsys.SyncDir(filepath.Dir(dstDir))`
right after the Mkdir (also on the ErrExist branch), before any copy or the Commit. Fix-skeptic
FIXED-VERIFIED: the new sweep test (each single unsynced item dropped after a successful run) fails
at item 0 without the fix. Residual: only a crash after a successful run is swept, not mid-run; an
unreadable parent directory now fails Archive.

## 25. A1-65: a killed Archive leaves a `.tmp-*` file nothing removes (low, CONFIRMED, fixed)

**Claim.** A process killed in the middle of copying a segment into an archive destination leaves a
`.tmp-*` file (up to one segment, 1 GiB at most) that nothing ever removes: only a writer's Open
sweeps temp files, and only in its own directory and `consumers/`.

**Grounds.** archive.go:146 uses `atomicfile.WriteFrom`, which removes its temp file only on an
in-process error; `removeStaleTemps` runs only from Open on `l.dir` and `consumers/`.

**Warrant.** Atomic replace by temp file is crash-safe for the target but not for the temp, and with
no sweeper in the destination the leak grows by one partial segment per killed run.

**Backing.** Decision 67 (stale `.tmp-*` removed, never a younger one that may belong to a live
writer), applied only to the log and `consumers/`.

**Rebuttal.** It needs a hard kill or power loss; the destination is operator-owned and nothing is
corrupted; the simulator reproduction emulates the kill by failing all later operations.

**Verdict.** CONFIRMED by 2 of 2 votes. Not reproduced by either skeptic (read-only trace of the
code); one notes "nothing ever removes" is not quite true, since a writer Open on a destination
would sweep the top level.

**Reproduction.** The auditor's simulator test: with every later operation failing, 1 temp file after
the crash and still 1 after a complete re-run of `archive()` (reported by the auditor, not re-run by
a skeptic).

**Disposition.** Fixed in commit db30a9452be4. Archive removes `.tmp-*` files older than an hour from
the destination (before the final SyncDir), on the reader's clock; the writer's sweep loop was
extracted into a shared `removeTempsBefore`. Fix-skeptic FIXED-VERIFIED: the test fails without it;
the CLI removed a stale file and kept a young one; staticcheck and deadcode ran clean. Residual: an
unrelated old `.tmp-*` file in an adopted destination is deleted (inferred, not tested); the
TempPrefix and Archive doc comments do not mention the sweep.

## 26. A1-78: doubled `quill: quill:` prefix, and a failed `commit` leaves a new consumer (low, CONFIRMED, fixed)

**Claim.** Every CLI error prints a doubled prefix whenever the library error already begins with
`quill: `; and `quill commit` creates the named consumer as a side effect even when it then fails.

**Grounds.** cmd/quill/verbs.go:45-47 prefixes every message; library errors begin `quill: `.
Observed: `quill: quill: offset out of range: 99 is outside [1, 6]`. `quill commit -dir orders -name
nope -offset 99` exited 1 and left `consumers/nope` and `nope.lock`. consumers.go:70 says "a
malformed number must not register the consumer".

**Warrant.** The prefix is a reading of the code plus observed output; the leftover is visible in
`quill consumers`.

**Backing.** Decision 41 (opening a handle or `register` creates the file); userguide.md:94.

**Rebuttal.** Both are cosmetic or low impact; the new consumer is unprotected; creation by commit
is consistent with decision 41, so only the failed-commit leftover contradicts the code comment (and
that comment covers only malformed numbers).

**Verdict.** CONFIRMED by 2 of 2 votes; one skeptic narrows it to the prefix.

**Reproduction.** Verified on macOS by both skeptics. Program: three appended records; `quill commit
-name nope -offset 99` gives the doubled prefix, exit 1, and `consumers/` holds `nope` and
`nope.lock`; `quill -dir nonexist stat` gives `quill: quill: not a log`.

**Disposition.** Fixed in commit 8449425c2457. `errorf` trims a leading `quill: `; a failed `commit`
removes a consumer it just created; a successful commit on a new name still registers it and the
user guide says so. Fix-skeptic FIXED-VERIFIED: both new tests fail without it and pass on macOS and
Linux. Residual: the empty `consumers/nope.lock` file remains (decision 41 and userguide line 100
say lock files stay); a race creating the same name between the listing and the open could delete
that consumer (inferred, negligible).

## 27. A1-29 (with A1-76): Open refuses a Dir that is a symlink to a directory (low, CONFIRMED, held)

**Claim.** Open refuses a `Dir` that is a symlink to a directory, with ErrNoLog and "is not a
directory". OpenReader accepts the same path, and parents that are symlinks work. Nothing documents
the writer-side restriction.

**Grounds.** log.go:340-343: `mkdirAll` uses Lstat, so `IsDir()` is false for a symlink.

**Warrant.** Decision 51's rule covers files inside the log, but `mkdirAll` applies it to the
directory itself; a deployment that symlinks its data directory (a common pattern) gets a
misleading ErrNoLog from the writer while its reader works.

**Backing.** Decision 51; scope.md Dir ("created if missing"); decision 62.

**Rebuttal.** The refusal may be deliberate hardening (a planted link cannot redirect the log) and
matches the O_NOFOLLOW stance; then only the message and the docs are wrong.

**Verdict.** CONFIRMED by 2 of 2 votes. Nothing in the tree records the refusal as deliberate.

**Reproduction.** Verified on macOS (both skeptics). Program: a real directory and a symlink to it:
`Open(Config{Dir: link})` returns `quill: not a log: <link> is not a directory`; `Dir: link/sub`
returns the same; `OpenReader(link)` returns nil; a symlinked parent with a real `Dir` below works.

**Disposition.** Held for the owner. Recommended: approve following the symlink.

## 28. A1-35: the age sweep is a strict prefix, so one future-stamped segment pins every newer one (low, CONFIRMED, held)

**Claim.** One segment stamped in the future (a clock that stepped ahead and was later corrected)
blocks age expiry of every newer segment until the wall clock reaches that stamp. Decision 44's
guard covers only the opposite case.

**Grounds.** retention.go:111-123: `for i, s := range l.sealed { ... if s.lastAt > cutoff { break } }`.

**Warrant.** The sweep stops at the first segment whose newest record is later than the cutoff.
Skeptics note it tests the last record's `At`, not the maximum, so the pin applies when the bad
stamp is on a segment's final record.

**Backing.** Decision 44; scope ("whole sealed segments by age"); userguide.md:133 ("older than the
window"), which the prefix rule contradicts here.

**Rebuttal.** Prefix deletion is deliberate (no middle hole); the caps still bound disk; `quill
check` reports the record. A fixed policy is a design choice.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** The auditor's test (reported, not re-run by the skeptics): segments 0 and 1
normal, segment 2 stamped a year ahead, 60 records at the corrected clock, then +48 h with
Retention 1 h; after 8 forced prunes `segs=8 first=21 head=90` and nothing logged. The skeptics
traced the code by hand.

**Disposition.** Held for the owner. Recommended: log a warning, and document it beside decision 44.

## 29. A1-50 (with A1-93): `-json` output passes C1 control characters raw (low, CONFIRMED, held)

**Claim.** `-json` output passes C1 controls (U+0080 to U+009F, for example the CSI introducer
U+009B) raw to the terminal although the guide says no raw control character reaches it; the text
form escapes them.

**Grounds.** cmd/quill/print.go:81-107 `appendJSONString` escapes only bytes below 0x20, quote,
backslash, `<`, `>`, `&` and U+2028 and U+2029; other valid UTF-8 is copied.

**Warrant.** A terminal that honours C1 in UTF-8 mode would treat U+009B as CSI (inferred from how
xterm and VTE behave; not tested), so a payload could inject escapes into `quill tail -json`.

**Backing.** docs/userguide.md:198-199 ("never pass a raw control character or invalid UTF-8");
scope.md:188 ("cannot inject lines or terminal escapes").

**Rebuttal.** Terminals vary; `encoding/json` also leaves C1 unescaped; the payload author must
already write to the log. The documentation promise is broken regardless.

**Verdict.** CONFIRMED by 2 of 2 votes. A skeptic also found DEL (0x7f) passes raw in `-json`
(the text form escapes it as `\x7f`).

**Reproduction.** Verified on macOS by both skeptics. Program: append `a\xc2\x9b31mred\xc2\x85b`;
`quill cat -json` emits bytes 302 233 and 302 205 unescaped, `quill cat` emits `\u009b` and `\u0085`.
A 265,792-payload differential test by the auditor otherwise found the JSON valid and lossless.

**Disposition.** Held for the owner. Recommended: approve escaping U+0080 to U+009F (and DEL).

## 30. A1-52 (with A1-88): Read returns more than MaxReadBytes of payload (low, CONFIRMED, held)

**Claim.** The cap is checked after each record is added, so the last record may push the total
over, contradicting "at most `limit` records and `MaxReadBytes` of payload, but always at least one".

**Grounds.** read.go:161 loop condition `bytes < maxBytes`; read.go:235 `bytes += len(rec.Payload)`;
read.go:242 stops only when `bytes >= maxBytes`.

**Warrant.** A record is admitted whenever the running total is below the cap regardless of its own
size, so the total is bounded by cap plus one record: at defaults about 16 MiB plus one record of up
to 64 MiB - 32, about 80 MiB, all cloned.

**Backing.** read.go:12-14 and scope.md Library API; guarantee 7. The shorter wordings (log.go:49,
userguide.md:145, scope.md:120) read as "stop once reached" and match the code.

**Rebuttal.** The "at least one" rule forces overshoot only for a single large record; the overshoot
is bounded by one record, not unbounded.

**Verdict.** CONFIRMED by 2 of 2 votes. No decision sanctions overshoot with more than one record.

**Reproduction.** Verified on macOS (both skeptics). Program: MaxReadBytes 16384, five 10 KiB records,
`Read(1, 0)`: 2 records, 20480 bytes. Second program: MaxReadBytes 1 MiB, one 1 MiB-1 record and two
5 MiB records: 2 records, 6291455 bytes. `Reader.Read`, Records and Follow share the bound (inferred
from the shared path).

**Disposition.** Held for the owner. Recommended: reword the docs; stopping earlier is the
alternative.

## 31. A1-57: `quill stat -json` serialises holes with capitalised keys and `null` (low, CONFIRMED, held)

**Claim.** `quill stat -json` writes each hole as `{"From":..,"To":..}` and an empty list as
`null`, while every other JSON key in the CLI is lowercase snake_case.

**Grounds.** cmd/quill/read.go:81-82 puts `st.Holes` (`[]quill.Hole`, log.go:253-255, no json tags)
in a map; `json.Marshal` gives `{"holes":[{"From":14,"To":26}]}` and `"holes":null` for none.

**Warrant.** The first public release freezes whatever scripts parse.

**Backing.** scope.md "Readable output: -json" fixes only the per-record schema for `cat`, `read` and
`tail`, not `stat`, `check` or `consumers`; v0 policy bumps the minor on a breaking change.

**Rebuttal.** The stat, check and consumers shapes are undocumented, so there is no contract to
break yet, which is the reason to settle it before release.

**Verdict.** CONFIRMED by 2 of 2 votes. Reproduced by one skeptic with a marshal program on an
equivalent type (the shipped binary was not run).

**Disposition.** Held for the owner. Recommended: approve lowercase keys and an always-present array,
and document the schemas.

## 32. A1-91: a quill.policy with no recognised fields parses as a valid policy of zeros (low, PLAUSIBLE, held)

**Claim.** `{}`, `null` and `{"unknown":1}` parse as a valid policy, which `quill prune` and `quill
ls` act on as "retention 0 = default 24 h, no caps".

**Grounds.** state.go:300 `decodePolicy` accepts any object; `quill ls` shows RETENTION 0s;
cmd/quill/write.go `cmdPrune` uses `ReadPolicy`'s ok and a zero Retention means 24 h
(`withDefaults`).

**Warrant.** Decision 68's rationale ("offline tools otherwise ran with default retention and could
prune a log") applies; a future writer that renames fields would produce such a file, which tolerant
parsing is meant to allow.

**Backing.** Decisions 29, 48, 68, 99.

**Rebuttal.** It needs a hand-edited or future-format file; v0.1.0 writers always write every field
(an off retention is written as `-1ns`, so retention missing or 0 is distinguishable from every real
policy); 24 h is the library's documented default; decisions 29, 48 and 87 define refusal as
"cannot parse".

**Verdict.** PLAUSIBLE (votes P, R, C). Reproduced on macOS by two skeptics: `{}`, `null` and
`{"unknown":1}` give RETENTION 0s and a `prune` that runs and exits 0; `{"retention":"bogus"}` and an
empty file are refused with exit 1. Aged-segment deletion under `{}` was not shown (fresh record
timestamps); an explicit `-retention 1ns` on the same log does delete. The REFUTED skeptic holds
that code and spec agree.

**Disposition.** Held for the owner. Recommended: approve treating a policy with no `retention` as
unparseable for the offline tools; the owner may decline (it is a hardening, not a violation).

## 33. A1-27: the write buffer never gives back capacity (low, REFUTED)

**Claim.** After one large record or batch `l.buf` keeps that capacity for the life of the log.

**Grounds.** flush.go:61 resets only the length (`l.buf[:0]`); nothing shrinks or nils it. Auditor
program: a 60 MiB append, flush, then small appends: `cap(buf)=62922752`, 60 MiB in use after a
forced GC.

**Warrant, backing.** A process with many topic logs can pin memory for rare spikes (scope.md model
line 2; decisions 46 and 69).

**Rebuttal.** It is a memory against regrowth trade-off bounded by the largest single append
(MaxBatchBytes 16 MiB, MaxPayloadBytes under 64 MiB); each Log has its own buffer, so "every other
open log in the process" is wrong.

**Verdict.** REFUTED (votes C, R, R, R). A skeptic measured the proposed shrink on a steady stream of
large batches: 0.3 MB per op rose to 8.4 MB per op with no wall-time gain. The factual part (capacity
is retained) holds; it is a deliberate reuse, not a defect.

**Disposition.** Not changed: refuted; the trade-off is intended.

## 34. A1-30: every inline flush stalls the Appends behind it (low, prior-deliberate)

**Claim.** Every inline flush holds `l.mu` across write, data fsync, marker write, marker fsync and
rename, stalling Append, Head and reads; about 0.2 percent of 128-byte appends see a 1 to 3 ms stall.

**Grounds.** flush.go:17-24 and append.go:78-79. Auditor measurement (reported, Linux ext4, 100k
128-byte appends, default config): Append p50 189 ns, p99 251 ns, p99.9 1.23 ms, max 3.24 ms; per
flush pwrite 85 us, fsync 652 us, marker fsync 472 us, rename 50 us.

**Warrant, backing.** Decision 13: "Moving the fsync outside the lock is a measured follow-up".

**Rebuttal.** The accepted design; dropping the marker's temp-file fsync would expose a torn
quill.tail; on macOS F_FULLFSYNC makes the stall about 20 ms per flush (decision 51).

**Verdict.** prior-deliberate (decision 13). No votes.

**Disposition.** Deliberately not changed. The proposed double-buffering is a larger concurrency
change in the most sensitive code and needs its own benchmark first.

## 35. A1-54: the Reader fails when a cap deletes the marker's own segment mid-view (low, prior-deliberate)

**Claim.** The Reader returns a raw path error (ending a Follow) when the marker's active segment is
sealed and immediately deleted by a cap between the marker read and the open.

**Grounds.** reader.go:281 reads the marker; reader.go:300-303 returns `openSegment`'s raw error with
no retry; flush.go:102-146 rotates, publishes the marker, then deletes under the caps in one locked
step. The test `TestTheMarkersOwnSegmentMissingStaysAnError` pins the raw error. A 15 s stress run
(MaxRecords 1 and 30) did not hit it; the window is reasoned from the code (inferred).

**Warrant, backing.** Decision 101 leaves the marker's own segment missing as an error on purpose
(a missing active segment is damage); guarantee 14; scope (a Follow ends only on a gap, ErrClosed or
a context error).

**Rebuttal.** Decision 101 makes it deliberate; the window is a few microseconds per rotation.

**Verdict.** prior-deliberate (decision 101; round-1 correctness-27 part 1). No votes.

**Disposition.** Deliberately not changed. Proposed, if revisited: re-read the marker once on
ENOENT for the active segment.

## 36. A1-56: `quill stat` reports no hole for a missing middle segment (low, prior-deliberate)

**Claim.** `quill stat` (`Reader.Stat`) reports `holes: null` and an exact-looking record count for a
log with a missing middle sealed segment; only `check` or a read that scans before the gap shows it.

**Grounds.** reader.go:337-352 derives each sealed segment's last sequence from the next good
segment's first frame, so a deleted segment's range is absorbed into its predecessor;
stat.go:29-39 counts Records from `v.holes` only. Verified by the auditor: a 4-segment log (1-13,
14-26, 27-39, 40) with 0000000001.log deleted: `stat` prints records 40, segments 3, no hole;
`check` exits 3 with `hole ...: sequences 14 to 26 are missing before it`.

**Warrant, backing.** Decision 47 (a Reader never scans to learn a segment's end) and decision 38;
scope lists holes among what `stat` prints.

**Rebuttal.** Intentional cost trade-off; `check` is the verb that verifies.

**Verdict.** prior-deliberate (decisions 47 and 38; round-1 docs-43). No votes.

**Disposition.** Deliberately not changed. Proposed: document that `stat`'s holes and records are
first-frame estimates, or add a `-verify` flag.

## 37. A1-58 (with A1-41): `quill archive` exits 0 when the source has a hole (low, prior-deliberate)

**Claim.** `quill archive` exits 0 when the source lost records, printing a message on stderr, so a
backup script cannot tell a complete archive from one with a known gap.

**Grounds.** cmd/quill/consumers.go:142-147 (`if errors.As(err, &gap) { fmt.Fprintf(c.err, ...);
return exitOK }`). Verified by the auditor: a log with a deleted middle segment prints `the source
lost records 14 to 26; the archive holds the same hole`, exit 0, and the archive's `check` is
UNHEALTHY (exit 3).

**Warrant, backing.** Exit codes in scope.md and decision 53; the library returns the `*GapError`
(archive.go:87-89) so the caller can decide.

**Rebuttal.** The message says so and scope says the archive "contains a hole and says so"; a
non-zero code would make an already-archived log fail on every later run until acknowledged.

**Verdict.** prior-deliberate (decisions 85 and 53). No votes; raised independently twice.

**Disposition.** Deliberately not changed. Proposed: a documented exit code or `-allow-gaps`, or at
minimum document that exit 0 does not imply no holes.
