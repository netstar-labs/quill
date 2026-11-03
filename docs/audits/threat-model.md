# Pass A3: security review and threat model

Date 2026-10-31. Pass A3 of the stage 2 audit, run with fresh eyes before a public release, against the tree at 988f1e5
(the commit under the first A3 fix; this is stated by one skeptic and inferred for the others, whose line numbers may
differ by a few lines). The specification is docs/design/scope.md, docs/design/decisions.md and docs/format.md; the
code was the thing under audit, so a passing test and a doc were both treated as claims, not evidence. The earlier
passes are recorded in [a1-round2-findings.md](a1-round2-findings.md) and its siblings; what this pass held back for the
owner is in [stage2-owner-decisions.md](stage2-owner-decisions.md), section "Pass A3".

Labels: **verified** means an auditor or skeptic ran it or read the exact lines; **reported** names the document that
says it; **inferred** says from what. Anchors (file:line) are to the audited tree unless marked "now"; the fixes listed
below have since moved some lines. Reproduction programs lived in throwaway copies of the repository made with
`git archive`, in a scratch area outside the tree, on macOS (Darwin 25.6, arm64) and on a Linux x86-64 box (Debian 13,
ext4); they are named below by the program label (for example `cmd/loop`) and are not committed. Nothing under audit was
edited by an auditor or skeptic. The recorder of this file did not re-run any reproduction. The recorder checked from
`git log` that the eleven fix commits named below are in the tree and that the two rejected attempts (54b99d0 and
e0df757) are not, and re-read a few current anchors in the unfixed code (marked "now").

## Top line

29 items (four of them merge a second candidate: A3-1 with A3-13, A3-26 with A3-2, A3-3 with A3-17, A3-4 with
A3-28). No CRITICAL and no HIGH item. 11 MEDIUM and 18 LOW, as rated by the auditors.

| verdict | items | ids |
|---------|-------|-----|
| CONFIRMED | 16 | A3-1, A3-15, A3-16, A3-27, A3-5, A3-14, A3-8, A3-10, A3-12, A3-20, A3-21, A3-22, A3-25, A3-30, A3-33, A3-18 |
| PLAUSIBLE | 1 | A3-19 |
| REFUTED | 3 | A3-26, A3-7, A3-24 |
| prior-deliberate (decided in round 1, not re-voted) | 7 | A3-3, A3-6, A3-9, A3-11, A3-29, A3-31, A3-32 |
| prior-refuted (decided in round 1, not re-voted) | 2 | A3-4, A3-23 |

| disposition | items | ids |
|-------------|-------|-----|
| fixed, fix-skeptic FIXED-VERIFIED | 12 | A3-1, A3-15, A3-16, A3-27, A3-8, A3-12, A3-21, A3-22, A3-25, A3-30, A3-33, A3-20 (since 2026-11-02) |
| fixed after fix-skeptic rejections; the last rejection was for two comment lines, fixed, with no skeptic re-run (not FIXED-VERIFIED) | 1 | A3-10 (badb254, 18662b6, ed518a2; 2026-11-02) |
| fix-failed (fix written, rejected by its fix-skeptic, not in the tree) | 0 | none |
| held for the owner (no fix written) | 4 | A3-5, A3-14, A3-18, A3-19 |
| refuted, deliberately not changed | 3 | A3-26, A3-7, A3-24 |
| decided in round 1, not re-opened | 9 | A3-3, A3-4, A3-6, A3-9, A3-11, A3-23, A3-29, A3-31, A3-32 |

- Skeptic votes: 44 item votes on the 20 items that were put to a vote (the 13 fixed or fix-failed items, two votes each; A3-5,
  A3-14 and A3-18, two each; A3-19, A3-26, A3-7 and A3-24, three each). 37 CONFIRMED and 7 REFUTED (A3-19 one, A3-26 two,
  A3-7 two, A3-24 two). The nine round-1 items were not voted again; they carry the auditors' own text and the round-1
  reference.
- Reproduced by execution: all 20 voted items were reproduced by at least one skeptic (A3-5: Linux, two uids; A3-26: macOS
  and Linux). Some individual voters did not reproduce (at least one voter on each of A3-5, A3-7,
  A3-12, A3-19, A3-24 and A3-26); where a figure was not re-measured the item says so. The nine round-1 items carry the auditors' own reproductions, which no skeptic re-ran in this pass.
- Fix commits: 11, all ancestors of the tree: acf7f39 (A3-1), 3105b42 (A3-15), 120c8ac (A3-16), 2bf7412 (A3-27), bb646c0
  (A3-8), 1f896c7 (A3-12), d6dc2b9 (A3-21), 46ff40f (A3-22), e6602b8 (A3-25), 495e457 (A3-30), 837c860 (A3-33).
- Fix-skeptic verdicts: 11 of 11 FIXED-VERIFIED, through 74 lens verdicts (derived by counting the ledger: seven lenses for
  eight fixes, six for A3-27, A3-12 and A3-33, which had no independent-route lens: 8x7 + 3x6),
  every one "pass". Each code fix was shown red before and green after by its fix-skeptic. A3-12 is documentation
  only, so its test lens is a check that existing tests pin the behaviour described. Two fixes were rejected: A3-10
  (54b99d0, rejected on the no-new-gap lens only) and A3-20 (e0df757, rejected for an unfixed Archive path). Neither is in the tree.
- **Update 2026-11-02 (pass A3h).** The owner approved the recommended answer for A3-10 and A3-20. A3-20 is now fixed in
  53a5d3b, fix-skeptic FIXED-VERIFIED (7 of 7 lenses pass, reported): the table above counts it with the fixed items, which
  makes 12 fix commits and 12 of 12 FIXED-VERIFIED (81 lens verdicts, derived: 74 + 7). A3-10 was attempted again (64837ac)
  and rejected again because decisions 84, 9 and 67 and two user-guide lines still stated the old rule; I then rebased it, fixed
  those and two `archive.go` comments a third skeptic named, gated it on both platforms and landed it (badb254, 18662b6,
  ed518a2). It is counted separately in the table below: no skeptic re-ran on the last commit. The record is in
  [stage2-owner-decisions.md](stage2-owner-decisions.md), sections "A3-10" and "A3-20". The rest of this file is as recorded
  on 2026-10-31 except the two Disposition paragraphs and the two table rows below.
- **Update 2026-11-03 (pass A3r).** The owner took the recommended answer for the round-1 decisions raised again, and
  approved A3-24 and the wording of A3-7 later the same day. Seven items changed disposition; each has a dated
  "Update 2026-11-03" paragraph under its Disposition and a new commit in its table row: A3-3 (116726b), A3-23 (18145b9),
  A3-24 (d15ac45, the escape set only), A3-29 (d916114, documentation), A3-31 (1cb2c93, documentation), A3-32 (d09e975,
  documentation plus one pinning test) and A3-7 (b765290, one sentence in decision 51 and SECURITY.md, no code). All seven
  were FIXED-VERIFIED by a fix-skeptic (45 lens verdicts, all pass: 7 each for A3-3, A3-23 and A3-24, 6 each for A3-29,
  A3-31, A3-32 and A3-7; reported by the skeptics). The verdict labels and the count tables above are as recorded on
  2026-10-31 and were not rewritten: read the verdict column as the verdict at audit time and the disposition column as
  current. Not changed and not approved: the A3-7 code hardening, the A3-24 `time` range check and A3-26 (all three held, see
  [stage2-owner-decisions.md](stage2-owner-decisions.md), "Held after pass A3r"). The decisions A3-4, A3-9 and A3-11 were
  confirmed unchanged on the same day (recorded in that file only, not here).

## Coverage

Three auditors, all of which returned: **A3-filesystem-boundary**, **A3-input-and-dos**, **A3-process-boundary**. No
auditor failed to return. What each checked and found clean is in the last part of this file ("What was checked and
found sound"). What was not covered, in one place:

- **Not run at all:** hard links (reasoned only: a hard-linked file as the newest segment would be opened read-write and
  truncated by recovery if the writer's uid can write it; Linux `protected_hardlinks` and ownership make that
  unreachable for a non-owner, inferred); a FIFO swapped in between the segment `Lstat` and its open (needs directory
  write); ENOSPC and EMFILE behaviour; real NFS, SMB and FUSE (none available, so the claim for them is reported from the
  docs); setuid or sticky-directory behaviour (a `/tmp`-style 1777 parent); a real clock step and a real full disk (no
  root; a fake clock and a fake `statfs` were used in-package); fork-without-exec descriptor inheritance (cgo
  daemonizing); other uids beyond the lock hold (single local user, permission reasoning only); 32-bit builds.
- **Not looked at:** the crash simulator (internal/fsx/crash*.go), examples/, hack/powertest, hack/ programs, and the
  container build scripts.
- **Platform limits:** the FIFO hangs and the lock-unlink race were reproduced on macOS and on Linux; the reader-only
  flock hold was reproduced on both and across uids (1000 and `nobody`) on Linux. The remaining reproductions
  (consumers/ symlink, same-size archive skip, junk consumer file, maxseg, tail mode, policy-driven prune, terminal
  escapes, leading hole, umask modes) ran on macOS only, except where a skeptic repeated one on Linux (A3-7, A3-18,
  A3-1, A3-8, A3-15, A3-16, A3-21, A3-27, A3-30, A3-33 and the fix checks). The Apple container (linux/arm64) was used only
  for A3-29 and a virtiofs check.
- **Fuzz limits:** the mutation fuzz used payloads of at most 400 bytes, so 64 MiB frames were exercised only by
  the amplification programs; it did not create FIFOs or symlinks (those were tested separately).

## The model

### Assets

| # | asset | what must hold |
|---|-------|----------------|
| S1 | Records that `Sync` or `Flush` acknowledged | not lost, not altered, not served in a changed form (guarantees 1, 6, 13) |
| S2 | The one-writer rule | at most one process appends to a log (guarantee 5) |
| S3 | `quill.tail`, the commit marker | names only what is durable; strictly parsed (guarantee 9) |
| S4 | Consumer offset files and locks | protect segments a consumer has not read; belong to one log (decision 6) |
| S5 | The writer's availability | a reader, consumer or stray file never blocks Append, Flush, Sync, Close (scope.md model) |
| S6 | Retention decisions | a segment is deleted by policy and the clock guard, not by an outside party's file or clock |
| S7 | The operator's terminal and logs | record bytes and file names cannot inject lines or escapes |
| S8 | Host memory, disk, descriptors | bounded by declared limits, not by hostile lengths or input size (guarantee 7) |
| S9 | Files outside the log | quill never reads, writes or deletes a file that is not part of the log it was pointed at |

### Adversaries

The default arrangement is one account, directory mode 0700 and files 0600 (SECURITY.md). SECURITY.md excludes "a
process with write access to the log directory". Several findings are about that excluded party anyway, because a
silent deadlock or silent overwrite is a different kind of failure from an error, and because the same hole is often
reachable by accident. Each is marked in the findings.

| # | adversary | can do | in the threat model? |
|---|-----------|--------|----------------------|
| P1 | A producer of payloads (any caller of Append) | choose payload bytes, including CRC-valid frames inside payloads; chosen-plaintext against CRC32 | yes (S1, S7, S8) |
| P2 | A damaged, restored or hand-built log directory (bit rot, bad restore, foreign producer) | present any bytes in segments and state files | yes: "hostile or damaged files on disk" (SECURITY.md) |
| P3 | A reader-only process (a different account, read access under a shared Perm) | read files; open them read-only; take a flock on any file it can open | yes: reader access is a supported configuration (decision 41) |
| P4 | A group consumer process | write access to `consumers/` (decision 41); create and replace consumer files | partly: the supported group arrangement |
| P5 | A party with write access to the log directory | create, replace, unlink or plant any entry (FIFO, symlink, junk file, lock) | no: excluded by SECURITY.md; included here as a hardening question |
| P6 | A party with write access to a parent of the log directory | plant a symlink for the directory; plant a subdirectory that `quill ls` lists; supply a destination for `archive -to` | partly (SECURITY.md names the parent's permissions as part of the boundary) |
| P7 | The host program or its configuration | choose a Config; chdir; replace the directory under a running handle | no: a trusted caller; included where a bad value passes validation |
| P8 | The operator | choose argv, stdin, `-dir`, `-to`; run destructive verbs | no: trusted, but mistakes are expected |
| P9 | The clock and free-space environment (an NTP source, another application on the same disk) | step the wall clock; consume the filesystem | no for the clock (decision 44: "slows the damage"); yes as an accident |
| P10 | A different kernel mounting the same directory (virtiofs, 9p, host mounts) | open the log without seeing the other kernel's flock | no: unsupported filesystems (decision 51); included as a documentation gap |

### Trust-boundary map

| boundary | across it | controls found in place | findings |
|----------|-----------|-------------------------|----------|
| B1 host program to the Log API (Config, payloads) | Config values, payload bytes | `validate` (ErrConfig), MaxPayloadBytes, MaxBatchBytes, the 1.5 GiB segment ceiling | A3-16, A3-33 |
| B2 writer to the log directory (its own files) | segments, `quill.tail`, temp files, lock | `O_NOFOLLOW` on every open (internal/fsx/os.go), `Lstat` then open for segments, temp files `O_EXCL` with a random suffix, atomic rename | A3-1, A3-8, A3-12, A3-5, A3-26, A3-7 |
| B3 reader and consumer processes to the directory | read of segments and state; consumer files; per-consumer lock | readers take no writer lock; consumer files are strict; names `[a-z0-9_-]` | A3-5, A3-27, A3-30, A3-4, A3-9 |
| B4 directory bytes to the decoder | segment frames, `quill.tail`, consumer files, `quill.format` | CRC, strict JSON (duplicates, unknown keys, trailing data refused), length caps before allocation | A3-8, A3-20, A3-21, A3-22, A3-25, A3-10, A3-19 |
| B5 record payload and file names to the operator's output | `cat`, `read`, `tail`, `ls`, `consumers`, `check`, `-json` | text mode quotes everything `strconv.IsPrint` rejects; JSON escapes controls, DEL, C1, U+2028/9 | A3-3, A3-24 |
| B6 operator to the CLI | argv, stdin, `quill.policy`, destination directories | numbers and names validated; destructive verbs need the writer's lock | A3-15, A3-11, A3-23, A3-14 |
| B7 one log directory to another | `Archive` destination, `ls` parent, a symlinked Dir | same-file check against the source, a held lock, a format id match | A3-14, A3-6, A3-7, A3-23 |
| B8 environment to retention | wall clock, free space | clock guard (at most a quarter of sealed segments per pass), caps override protection | A3-31, A3-32 |
| B9 one kernel to another | `flock` coherence | `ErrLocked` within one kernel, including bind mounts (verified) | A3-29, A3-26 |
| B10 cost to the size of the log | read cost, listing cost | sparse index in a sealed segment; byte cursors in Records and Follow | A3-18, A3-19, A3-27 |

## Findings, ranked

No CRITICAL or HIGH item was raised. Within a severity, confirmed items come first, then plausible, refuted and
the round-1 items.

| id | sev | kind | verdict | disposition | one line |
|----|-----|------|---------|-------------|----------|
| A3-1 | medium | bug | CONFIRMED | fixed acf7f39 | a FIFO in a state file or `consumers/` blocks quill, and at a seal blocks the writer's mutex |
| A3-15 | medium | security | CONFIRMED | fixed 3105b42 | `quill append` buffers all of stdin before the batch limit is checked |
| A3-16 | medium | security | CONFIRMED | fixed 120c8ac | a Config near MaxInt64 wraps `maxActiveBytes` negative and passes validation |
| A3-27 | medium | perf | CONFIRMED | fixed 2bf7412 | a seal lists and reads every consumer file under the log mutex |
| A3-5 | medium | security | CONFIRMED | held for the owner | a read-only process can flock `quill.lock` when Perm grants read, blocking writer Open |
| A3-14 | medium | security | CONFIRMED | held for the owner | `archive -to` writes into and overwrites files in any directory with no `quill.format` |
| A3-26 | medium | bug | REFUTED | not changed | unlinking `quill.lock` under a live writer lets a second writer open and overwrite |
| A3-3 | medium | security | prior-deliberate | round 1; changed 116726b (2026-11-03, owner-approved, FIXED-VERIFIED) | file and directory names are printed unescaped by `ls`, `consumers` and `check` |
| A3-4 | medium | bug | prior-refuted | round 1 | any non-consumer file in `consumers/` makes retention protect everything |
| A3-6 | medium | bug | prior-deliberate | round 1 | Archive keeps a same-size destination segment without verifying it |
| A3-29 | medium | gap | prior-deliberate | round 1; documented d916114 (2026-11-03, owner-approved, FIXED-VERIFIED) | no detection of filesystems where flock is not coherent; names missing from the docs |
| A3-8 | low | bug | CONFIRMED | fixed bb646c0 | `quill.tail` accepts a segment id the ten-digit file names cannot hold |
| A3-10 | low | gap | CONFIRMED | fixed badb254, 18662b6, ed518a2 (2026-11-02; no skeptic re-run on the last, see stage2-owner-decisions.md) | a damaged oldest segment is skipped silently by a default-start read |
| A3-12 | low | doc | CONFIRMED | fixed 1f896c7 | segments and directories go through the umask; decision 51 said "exact" |
| A3-20 | low | bug | CONFIRMED | fixed 53a5d3b (e0df757 rejected earlier) | a Reader serves overlapping segments that writable Open refuses |
| A3-21 | low | bug | CONFIRMED | fixed d6dc2b9 | a non-minimal varint passes Check and Open, then breaks the read cursor |
| A3-22 | low | bug | CONFIRMED | fixed 46ff40f | `walk`'s contiguity test wraps at 2^64-1 and accepts seq 0 after it |
| A3-25 | low | bug | CONFIRMED | fixed e6602b8 | `checkDuplicateKeys` loses the parent object after a container member (latent) |
| A3-30 | low | bug | CONFIRMED | fixed 495e457 | a long-lived Reader or Consumer never re-checks the log identity |
| A3-33 | low | hygiene | CONFIRMED | fixed 837c860 | a relative Dir stays relative, so a later chdir moves the log |
| A3-18 | low | degradation | CONFIRMED | held for the owner | a Reader's read cost scales with the log, not the request |
| A3-19 | low | perf | PLAUSIBLE | held for the owner | the first-frame probe reads and CRC-checks the whole first frame, up to 64 MiB |
| A3-7 | low | security | REFUTED | not changed in code; wording only b765290 (2026-11-03, owner-approved, FIXED-VERIFIED) | a symlinked `consumers/` directory is followed |
| A3-24 | low | security | REFUTED | escape set changed d15ac45 (2026-11-03, owner-approved, FIXED-VERIFIED); time field not changed | `-json` passes bidi and format controls raw; the time field is unchecked |
| A3-9 | low | hygiene | prior-deliberate | round 1 | a Reader takes the mode of its files from `quill.tail`, and chmods exactly |
| A3-11 | low | gap | prior-deliberate | round 1 | `quill prune` acts on `quill.policy` as trusted configuration |
| A3-23 | low | degradation | prior-refuted | round 1; changed 18145b9 (2026-11-03, owner-approved, FIXED-VERIFIED) | offline verbs create a log at a mistyped `-dir` |
| A3-31 | low | degradation | prior-deliberate | round 1; documented 1cb2c93 (2026-11-03, owner-approved, FIXED-VERIFIED) | a forward clock step makes every protected consumer stale under ConsumerTTL |
| A3-32 | low | degradation | prior-deliberate | round 1; documented d09e975 (2026-11-03, owner-approved, FIXED-VERIFIED) | MinFreeBytes lets another process's disk use drive deletion of this log's history |

### MEDIUM

#### A3-1 (merges A3-13): a FIFO as a state file or a consumers/ entry blocks quill (medium, bug)

**Claim.** A FIFO planted as any state file, or as an entry of `consumers/`, blocks quill forever. With a FIFO in
`consumers/`, the next segment rotation deadlocks the whole Log (Append, Head, Read and Sync all block) and the flusher
and Close hang. A FIFO at `quill.tail`, `quill.format`, `quill.policy` or a consumer file hangs OpenReader, Open and the
CLI verbs. SECURITY.md says segments are accepted only as regular files, but only the segment path had the pre-open
type check.

**Attack scenario.** P5 (or a stray `mkfifo` by accident) creates one FIFO; the writer then stops at its next seal with
no error and no log line, because the listing runs under `l.mu`.

**Grounds.** `readState` opened first and checked the type after (state.go, `OpenFile(O_RDONLY|O_NOFOLLOW)` then
`f.Stat()` and `IsRegular`; state.go:358-370 now). `O_NOFOLLOW` does not stop a FIFO open from blocking, and there
was no `O_NONBLOCK` or `Lstat` first. `openSegmentFlag` already did `Lstat` first (load.go:68-75). `consumerStore.list`
(consumer.go:244-262 at audit) read every entry of `consumers/` except `*.lock` and `.tmp-*`; `publishLocked` called it
with `l.mu` held at every seal (flush.go:136); the flusher called `l.consumers(now)` inline (flush.go:282).

**Warrant.** `open(2)` of a FIFO with `O_RDONLY` and no writer blocks in the kernel before `fstat` can run, so the
`IsRegular` guard after the open never executes. One blocked syscall under the mutex stops every operation that takes
`l.mu`, and Close waits on `<-l.done`.

**Backing.** SECURITY.md "Symlink substitution"; decision 51; the comment at load.go:66 ("so a FIFO cannot block it")
shows the intent was applied to segments only; decision 41 and 51 describe a group arrangement in which consumers
need write access to `consumers/`.

**Rebuttal.** The attacker needs create access in the log directory or `consumers/`, which SECURITY.md excludes, and
default modes give no other uid that access. Such a party can also delete or replace `quill.tail` and the segments, so
the FIFO adds no capability; it does add a silent deadlock under the writer's mutex. Both skeptics rated it
low-to-medium hardening, and one noted that decision 41 gives a consumer process exactly the write access needed.

**Verdict.** CONFIRMED (2 of 2). Medium is generous but defensible.

**Reproduction (verified).** `repro8` (macOS): `quill.Open` with SegmentBytes 1 KiB and NoMaintenance, `syscall.Mkfifo`
on `consumers/x`, then Append and Flush in a loop: "appender stuck after 5s", and `Head()` blocked too. `repro1` (macOS and
Linux): FIFO planted, then Sync after the next prune pass: "context deadline exceeded 3s" and Close blocked past 5 s.
CLI on macOS: a FIFO at `consumers/zz` hangs `consumers` and `check`; at `quill.tail` hangs `stat` and `append`; at
`quill.format` hangs `stat`; at `quill.policy` hangs `check` and `quill ls <parent>`, so one hostile subdirectory hangs
`ls` for the whole parent. A skeptic re-ran the rotation case on macOS and the CLI cases at `quill.tail` and
`consumers/zz`.

**Disposition.** Fixed in acf7f39: `fsx.OS.OpenFile` adds `O_NONBLOCK` to every read-only open (internal/fsx/os.go,
`osFlags`), so the open returns at once and `readState`'s existing `IsRegular` check refuses the file as `errBadState`.
Chosen over an `Lstat`-first check because it leaves no window between the check and the open and covers every
caller in one place. Fix-skeptic **FIXED-VERIFIED**, 7 of 7 lenses: the new tests fail on the parent (OpenReader,
ReadPolicy and `Reader.Consumers` blocked on the FIFO) and pass on macOS and Linux; the built CLI returned at once
for a FIFO at `quill.tail`, `quill.format`, in `consumers/`, and in place of `consumers/` itself; the race tests, vet,
staticcheck and deadcode were clean. Its two residual points: the proposed "do not list consumers under `l.mu`" change
was not made in this commit (closed afterwards by A3-27, whose fix-skeptic grepped every listing site and found none
under `l.mu`); and the pre-fix CLI hang was not independently re-run by that skeptic (the red unit tests cover it).
Decision 74 carries the amendment.

#### A3-15: `quill append` buffers all of stdin before the batch limit is checked (medium, security)

**Claim.** `quill append` with no arguments buffers the whole of stdin before the library's MaxBatchBytes check, so
a small amount of input allocates about 40 times its size and an unbounded pipe is unbounded memory. This contradicts
guarantee 7 and SECURITY.md ("an oversize payload or batch is refused before it is allocated").

**Attack scenario.** A pipe that never ends (`tail -f x | quill append`) or a large file of newlines grows until the
OOM killer acts, and nothing is appended. The input is chosen by the operator (P8), so this is a robustness and
resource-bound defect more than an attack.

**Grounds.** cmd/quill/write.go:95-100: `for sc.Scan() { payloads = append(payloads, append([]byte(nil), sc.Bytes()...)) }`
runs to EOF; only then does `AppendBatch` (append.go) compare the sum with MaxBatchBytes. Each empty line costs a
24-byte slice header.

**Warrant.** The loop keeps one slice header plus an allocation per line and never consults the 16 MiB limit that is
going to refuse the batch, so peak memory is proportional to the input with a per-line overhead of 24 to 32 bytes against
1 to 2 bytes of input.

**Backing.** SECURITY.md "What quill defends against"; scope.md guarantee 7; the CLI help for `append` says
MaxBatchBytes applies. Related: A1-69 (documented stdin as one batch subject to MaxBatchBytes).

**Rebuttal.** stdin is chosen by the operator, not an attacker; normal input (under 479,000 lines) never gets near
the problem; a caller can `ulimit`; the library is not affected. Both skeptics rated it lower than medium and read the
SECURITY.md wording as written about hostile files on disk. Decision 46 says stdin is one batch subject to
MaxBatchBytes and rejected chunking, which an early refusal does not undo.

**Verdict.** CONFIRMED (2 of 2).

**Reproduction (verified).** macOS: 100,000,000 bytes of `\n` piped to `quill -dir t5 append` printed "payload too
large: batch of about 3400000000 bytes, MaxBatchBytes is 16777216" with maximum resident set size 4,029,644,800 bytes
(`/usr/bin/time -l`), 5.3 s. A skeptic measured 1.76 GB at 20 MB of input and 3.2 to 3.3 GB at 100 MB (about 32 times;
the 40 times in the claim is the auditor's own figure and slightly overstated). A single normal line printed 1 and
succeeded.

**Disposition.** Fixed in 3105b42: the scan loop in cmd/quill/write.go `appendRecords` keeps a running sum of
`len(line)+34` and refuses with the same ErrPayloadTooLarge as soon as it exceeds `quill.DefaultMaxBatchBytes` (the
CLI opens the log without setting MaxBatchBytes and the policy does not carry it), so the line count is bounded too
(about 493,000 empty lines). A single line is still bounded by MaxPayloadBytes alone, as in `AppendBatch`.
Fix-skeptic **FIXED-VERIFIED**, 7 of 7: the new test fails without the fix (read 4 MiB of 4 MiB before refusing);
100 MB of newlines went from 3,332,489,216 bytes in 8.0 s to 44,253,184 bytes in 0.36 s; an endless `yes ''` is
refused at 44 MB; 493,447 empty lines accepted and 493,448 refused by both the library arithmetic and the binary; also
verified on Linux. Residual (not a defect of the fix): the refusal wording differs from the library's, and an
unterminated endless stream still reaches about 140 MB (the scanner buffer at MaxPayloadBytes), which is constant and
pre-existing.

#### A3-16: Config validation bypassed by integer overflow in `maxActiveBytes` (medium, security)

**Claim.** `maxActiveBytes()` sums `int` terms and wraps negative when MaxBatchBytes (or MaxBufferBytes) is near
math.MaxInt64, so Open accepts a configuration whose single batch can exceed the 1.5 GiB segment file limit. Records that
Sync reported durable then read back as a hole.

**Attack scenario.** Not an attacker: a host program choosing MaxInt64 as "unlimited" (P7), then one AppendBatch of more
than 1.5 GiB (about 26 payloads of nearly 64 MiB, several GB of RAM).

**Grounds.** log.go:195-197, `c.SegmentBytes + int64(max(FlushBytes, MaxBufferBytes)+max(MaxBatchBytes, MaxPayloadBytes+34))`,
had no overflow guard; `validate()` (log.go:172) tested the wrapped result against `maxSegmentBytes`.

**Warrant.** `validate` is the only thing keeping a flush below `maxSegmentBytes` (decision 60 d, A1-25). The wrap
turns the sum negative, so the guard passes; `appendLocked`'s batch bound is then never hit and the flush writes a
1.69 GB file. `openSegmentFlag` refuses files over 1.5 GiB and a sealed oversize file becomes a hole, so
acknowledged data is unreadable to every reader.

**Backing.** docs/format.md Limits; scope.md Config table (Open refuses such a Config with ErrConfig); guarantee 13.

**Rebuttal.** The Config comes from the host program; it needs a deliberate huge value and a single huge batch;
defaults are safe. The loss is reported as an explicit unavailable-records error, not served as corruption. Both
skeptics read it as a validation hole, medium-to-low.

**Verdict.** CONFIRMED (2 of 2).

**Reproduction (verified).** macOS (`internal/repro/cfg`): MaxBatchBytes `2<<30` gave ErrConfig, but MaxBatchBytes
MaxInt64, MaxInt64-1000, and MaxBufferBytes MaxInt64-100000 gave Open err = nil. Linux (`internal/repro/brick`,
MaxBatchBytes MaxInt64, one AppendBatch of 26 x 62 MiB): AppendBatch ok, Sync nil, "durable: head=26 bytes=1690304980
(file limit 1610612736)", reopen nil, then `Read(1,1)` returned "records 1 to 26 are unavailable". One skeptic repeated
the whole sequence on macOS; the other reproduced the config acceptance only.

**Disposition.** Fixed in 120c8ac: `maxActiveBytes` caps each flush term at `maxSegmentBytes` before summing (the
saturating-arithmetic option), so Open refuses such a Config with ErrConfig, and the `rotErr` ceiling in `appendLocked`,
which uses the same function, stays positive. Fix-skeptic **FIXED-VERIFIED**, 7 of 7: three of the new test rows fail
on the parent ("batch wraps int64: <nil>", "buffer near wrap: <nil>", "batch near wrap: <nil>"); an Open probe gave
identical ErrConfig results on macOS and Linux for MaxInt64, MaxInt64-1000, 1<<60 and exactly 1.5 GiB, and the defaults,
1 GiB and 1.4 GiB still open. Residual: the "flush wraps int64" test row also passes without the fix (the
negative-size check catches it), so it proves nothing by itself; the full race suite ran on macOS only.

#### A3-27: the writer lists and reads every consumer file under the log mutex at every seal (medium, perf)

**Claim.** The writer reads every file in `consumers/` while holding the log mutex at every segment seal. Stall time
grows linearly with the number of entries, so a log with many consumer files stalls every Append, Flush, Read and Sync
at each rotation; a process that can create consumer files (any group consumer through `Reader.Consumer`) can make
the writer unavailable for seconds per rotation.

**Attack scenario.** P4 (or an accumulation of per-job consumer names, which are never expired by default) creates tens
of thousands of valid consumer files; each seal then stalls the writer.

**Grounds.** flush.go:132-143 inside `publishLocked`, reached from `flushLocked` (flush.go:64) and `Roll` (flush.go:194)
with `l.mu` held: `l.consumerStore().list()` opens, stats and reads every entry (consumer.go:270-295). The result was
used only to count Dropped. The retention pass already shows the safe pattern: `l.consumers(now)` runs before `l.mu` is
taken (flush.go:282, retention.go:80).

**Warrant.** O(N) open/read/close calls run under the lock that serialises all appends, and the measured latencies
scale about linearly with N on both operating systems.

**Backing.** scope.md model line 6 and the denial-of-service lens (a reader or consumer operation must never block the
writer); architecture.md section 3 (the flush holds the lock for write and fsync only).

**Rebuttal.** Creating files in `consumers/` needs write access to that directory, which SECURITY.md treats as inside the
trust boundary; a legitimate cause also exists (per-job names accumulate); the stall happens only at rotation. One
skeptic noted that two of the proposed fixes were unworkable: skipping Dropped for cap-driven deletes breaks decision 12
and `TestDiskBounds`, and computing before the lock cannot work for the inline Append flush, which cannot know it
will seal.

**Verdict.** CONFIRMED (2 of 2).

**Reproduction (verified).** `cmd/many` (N valid consumer files, then appends for 6 s with 64 KiB segments). macOS APFS:
N=0 worst Append 25 ms, 3072 appends; N=5000 166 ms; N=20000 496 ms; N=100000 4.705 s, 259 appends. Linux ext4 (under the
benchmark lock): N=5000 65 ms; N=100000 1.477 s, 519 appends against 3248 at N=5000. A skeptic's in-package test at N=20000:
6630 to 195 appends in 4 s, worst Append 2.24 s.

**Disposition.** Fixed in 2bf7412: a seal no longer lists `consumers/` at all, capped or not. `publishLocked` passes
`deleteLocked` the offsets the last prune pass read (the new unexported `Log.offsets`, set in `pruneLocked` from offsets
the flusher and Prune read before taking `l.mu`; Open takes the first set). Cost, now stated in decision 12, the Stats
doc, architecture.md and the changelog: Dropped can be off for up to PruneInterval. Fix-skeptic **FIXED-VERIFIED**, 6
of 6: the new test fails on the parent in all three subtests ("9 listings of consumers/ ran with the log's lock held",
using `TryLock` inside the filesystem seam); `cmd/many` at N=100000 went from worst 4.74 s to 25.1 ms on macOS and
to 26.3 ms on Linux. Residual: Dropped can now over-count as well as under-count (a consumer that caught up since the
last pass is still counted); under NoMaintenance the staleness is unbounded, not "up to PruneInterval", until an
explicit Prune; Open pays one O(N) listing.

#### A3-5: a read-only process can flock `quill.lock` when Perm grants read (medium, security)

**Claim.** The writer's `quill.lock` is created with Config.Perm minus umask. When Perm grants group or other read (the
documented way to let reader processes in), a process that can only read can open the lock file `O_RDONLY` and take
`LOCK_EX` with flock. That blocks every writer Open (ErrLocked) for as long as it holds it, including a writer restart.
The consumer locks are created from the log mode too and are subject to the same hold.

**Attack scenario.** P3: a reader-only account under a shared Perm holds the flock and the owner's service cannot
restart until the holder exits.

**Grounds.** lock_unix.go:22 creates the lock with Config.Perm (`OpenFile(path, O_RDWR|O_CREATE, perm)`, line 22 now);
log.go passes `cfg.Perm`; flock does not require write access to the descriptor; scope.md and decision 41 say a
reader-only process needs only read access.

**Warrant.** A reader never needs the writer lock, so there is no reason for the lock to be readable by readers; because
flock is advisory and works through a read-only descriptor, read permission on the lock file amounts to the ability to
take the lock.

**Backing.** scope.md model line 1 and the Perm row (a shared mode is a supported configuration); decision 51 ("a lock
file ... is created through the process umask"); decision 41.

**Rebuttal.** Default Perm 0600 is not affected; the attacker needs a deliberately shared Perm; it cannot stop a
running writer, only Open; the hold ends when the attacker process exits. Consumer locks must stay group-accessible when
group consumers write them, so only the writer lock can be hardened this way. Under a group-write arrangement the attacker
could do worse anyway.

**Verdict.** CONFIRMED (2 of 2); medium looks right for a shared Perm. One skeptic did not re-run the two-uid test and
relied on reading the code (decision 51 makes the umask-created lock deliberate but does not address this effect).

**Reproduction (verified).** Linux, two uids: log created by uid 1000 with Perm 0644; `sudo -u nobody` (uid 65534, no
write access) opened `quill.lock` `O_RDONLY` and printed "uid 65534 flock: <nil>"; the owner's `quill.Open` then failed
"quill: locked by another holder: .../quill.lock"; after the holder was killed the next Open returned nil. With
default Perm 0600 the same actor got "permission denied" (auditor). Same-uid with Perm 0640 on macOS and Linux gave the
same result (auditor). A skeptic repeated the two-uid sequence on Linux, without the 0600 control.

**Disposition.** Held for the owner (it changes a documented file mode). Recommended answer and the proposed, unwritten
fix are in stage2-owner-decisions.md, "A3-5".

#### A3-14: `quill archive -to` writes into and overwrites files in a directory that is not a log (medium, security)

**Claim.** `quill archive -to <dir>` (and `quill.Archive`) writes into, and overwrites files in, any existing directory
that has no `quill.format`. Writable Open refuses the same directory with "holds files but no quill.format". A mistyped
`-to` silently replaces an unrelated file named like a segment (`NNNNNNNNNN.log`) and drops `quill.format`,
`quill.tail` and `consumers/` into the directory.

**Attack scenario.** Operator error (P8) with a plausible typo; or P6 supplying a directory name.

**Grounds.** archive.go:106-155 (`archiveDest`, line 108 now) checks only same-file as the source, a held writer lock
and the format id when `quill.format` exists; when it is absent it writes one. `copySegment` then renames over
`dstDir/NNNNNNNNNN.log` whenever the existing file's size differs (archive.go:176; 178 now). log.go shows the check Open
makes ("holds files but no quill.format", log.go:508 now) and `archiveDest` lacks.

**Warrant.** The two paths guard the same condition (a directory that is not a log) differently; Archive creates a
log-shaped directory inside whatever it is pointed at, and its atomic temp-and-rename replaces a regular file of any
content.

**Backing.** decision 62 and `Log.create()`; Archive's doc comment promises a destination "that opens as a log".
SECURITY.md makes no promise either way for destructive tooling; the asymmetry with Open is the licence.

**Rebuttal.** Needs operator error: `-to` must be an existing directory without `quill.format`, and only files whose
names match a source segment id and differ in size are destroyed. A mistyped nested path fails (`Mkdir` without
parents, verified). A destination that is empty or was created by Archive is unaffected. One skeptic would rate it
low-to-medium.

**Verdict.** CONFIRMED (2 of 2).

**Reproduction (verified).** macOS, built binary: `mkdir unrelated; echo 'precious unrelated data' > unrelated/0000000000.log;
echo keep > unrelated/notes.txt; quill -dir src archive -to unrelated` exited 0; `0000000000.log` (24 bytes of text)
became the 39-byte segment copy and the directory gained `quill.format`, `quill.tail` and `consumers/`; `quill -dir
unrelated append x` on the same directory printed "not a log: unrelated holds files but no quill.format". A skeptic
repeated it through `quill.Archive` (1250-byte segment copy over the 24-byte file).

**Disposition.** Held for the owner (it makes a previously working invocation fail). See stage2-owner-decisions.md, "A3-14".

#### A3-26 (merges A3-2): unlinking `quill.lock` under a live writer lets a second writer open (medium, bug)

**Claim.** The writer lock is an flock on a deletable file. If `quill.lock` is unlinked while a writer is live, a second
Open succeeds and both writers then acknowledge flushes as durable. One writer's acknowledged records are silently
overwritten and `quill check` reports HEALTHY. Guarantee 5 does not hold once the file is removed.

**Attack scenario.** P5, or an accident: `rm *.lock` as "stale lock" cleanup, an age-based temporary-file cleaner (an
empty lock file's mtime never changes), `rsync --delete` during a restore. These accidental triggers are inferred, not run.

**Grounds.** lock_unix.go:21-40 (`takeLock`: `OpenFile(O_CREATE)` then flock on that inode; no later check that the
path still names the held inode); log.go is the only writer lock; flush.go:37-67 writes with `WriteAt` at the in-memory
durable offset.

**Warrant.** flock binds to the inode, not the path. Unlink and recreate yields a different inode, so the first writer's
lock excludes nobody. Neither writer detects the other; overlapping `WriteAt` calls lose data without any CRC or marker
signal because the survivor's frames are all individually valid.

**Backing.** scope.md guarantee 5; format.md:197-201 ("Lock files are never deleted"); decisions 17, 41, 62. The documents
assume nobody deletes the file but state no consequence if it is deleted.

**Rebuttal.** Needs a party with write access to the log directory, which SECURITY.md:28-31 puts out of the threat
model; format.md says lock files are never deleted by quill (and no code path removes one); a second writer after an
external unlink is the standard behaviour of flock on a path. Decision 41 records that unlinking a lock file lets two
processes hold the same lock.

**Verdict.** REFUTED (votes: REFUTED, CONFIRMED, REFUTED). The mechanism was reproduced and not disputed. It is refuted as
a defect, not as a fact: it is out of model and a deliberate design rule. The confirming voter would treat it as a
defence-in-depth or documentation gap.

**Reproduction (verified).** Auditor, macOS, `cmd/lab` "double": after `os.Remove(quill.lock)` the second Open returns nil,
both Flush return nil, both report head 1, one 10-byte record survives. `cmd/loop` (macOS and Linux x86-64 ext4): writer A
started, `quill.lock` removed after 1 s, writer B started. macOS: A acknowledged 30, B acknowledged 30 (head 37); `quill cat`
shows A-00..A-29 and only B-23..B-29, so 23 of B's acknowledged records are gone. Linux: B acknowledged 30, 40 records
on disk, 10 of B's survive. `quill check` prints HEALTHY in both. A skeptic reproduced a control second Open failing with
ErrLocked, the unlink, the second Open succeeding, both Sync nil, and only B's five records on disk, on macOS and Linux.

**Disposition.** Deliberately not changed: out of the threat model and a stated design rule (never delete the lock). The
two hardening options the auditor wrote (compare the held inode with `Lstat(quill.lock)` at each marker publish; or flock
the directory descriptor as well, which the auditor probed as exclusive on macOS and Linux) and the one documentation
sentence are noted in stage2-owner-decisions.md, "Not held, but worth a line".

#### A3-3 (merges A3-17): file and directory names are printed unescaped (medium, security)

**Claim.** A file or directory name controlled by an untrusted party is printed unescaped by `quill ls`, `quill consumers`
and `quill check`. The attacker can inject terminal escape sequences and newlines that forge table rows. scope.md
promises escaping for payloads only, but the same threat applies to names.

**Attack scenario.** P6 (for `ls`, create access in the parent directory, possibly a lower privilege than write access to a
log) or P4 (for `consumers` and `check`, a file in `consumers/`).

**Grounds.** cmd/quill/read.go:51 and :65 print `e.Name()` of every subdirectory of the parent with `%s`;
cmd/quill/consumers.go:46 prints `ci.Name`, which consumer.go:250-254 takes from the directory listing with no
`validName` filter; `check` prints paths built from the name. The `ls` case needs only a minimal `quill.format` and
`quill.tail` pair, which OpenReader accepts.

**Warrant.** Writing attacker-chosen bytes below 0x20 to a terminal is interpreted by the terminal.

**Backing.** scope.md "Readable output" (the stated intent covers record bytes; names are the same class of untrusted input);
SECURITY.md "Hostile or damaged files on disk".

**Rebuttal.** The attacker needs create access in the parent or `consumers/`; in the default 0700 arrangement that is
only the operator's own uid; `-json` is safe for names (`encoding/json` escapes controls).

**Verdict.** prior-deliberate (decided in round 1, not re-voted): a1-round2-findings.md C3 coverage note, "Mentioned, not
raised: terminal escapes in file names printed by consumers, check and ls".

**Reproduction (auditor, macOS, not re-run by a skeptic).** A subdirectory named newline, `ESC ]0;PWNED BEL`, `ESC [2J`
holding only `quill.format` and `quill.tail`: `quill -dir parent ls | cat -v` printed "evil" on one line and
`^[]0;PWNED^G^[[2J` as a forged row. A file `consumers/x ESC[31mRED ESC[0m` made `quill consumers` and `quill check` print
raw ESC bytes (`cat -v` shows `^[[31m`).

**Disposition.** Round-1 decision stands; not re-opened. The auditor's proposed fix (quote file-derived strings as
`printer.appendQuoted` does; skip or flag names that fail `validConsumerName`) is recorded here for the owner; no fix was
written.

**Update 2026-11-03 (owner-approved; fixed in 116726b, fix-skeptic FIXED-VERIFIED, 7 of 7 lenses, reported).** The owner
took the recommended answer and the round-1 decision is changed. In text output `ls`, `consumers`, `check` and error
messages pass every string derived from the disk through one helper, `safeText`: a string that is valid UTF-8 with every
rune printable prints unchanged, anything else is quoted by the existing payload escaper `appendQuoted`. `appendQuoted` also
writes a byte that is not valid UTF-8 as `\xNN` (it used to print U+FFFD); payloads reach it only after `utf8.Valid`, so
payload output is unchanged. `-json` is unchanged. The fix-skeptic (reported) ran the built binary on macOS and, on
linux/arm64 in an alpine container, with names holding a newline, ESC, bidi controls and invalid UTF-8: all printed
quoted, no raw control byte and no forged line. Not done, as the approved text asked only for quoting: the auditor's other
half (skip or flag names that fail `validConsumerName`) is the A3-4 question and stays with that item. Non-blocking notes
from the skeptic: the "stray segment-like file" in the new test is ignored by `check`, so that part of the test asserts
nothing; the Linux x86-64 harness was absent, so Linux evidence is arm64.

#### A3-4 (merges A3-28): any non-consumer file in `consumers/` makes retention protect everything (medium, bug)

**Claim.** Any non-consumer file in `consumers/` (`.DS_Store`, an editor backup, an attacker's file) is treated as an
unparseable consumer file. Retention then protects everything, with no log line and no way to clear it through quill,
until the file is deleted by hand. ConsumerTTL cannot expire it; only the caps (off by default) bound the disk.

**Attack scenario.** P4 or P5 disables age retention with one byte; or an operator's file manager does it by accident.

**Grounds.** `consumerStore.list` (consumer.go:244-267) lists every entry that does not end in `.lock` or start with
`.tmp-`; `Log.consumers` (retention.go:39-43) sets `cv.broken` for any entry with an error; `pruneLocked` (retention.go:134-138)
then uses `firstRetainedLocked` as the protected minimum; the TTL logic (retention.go:48) is reached only for entries that
parsed; `unregister` cannot remove such a file because `validConsumerName` rejects the name.

**Warrant.** The broken flag is a deliberate fail-safe for a damaged real consumer file but is raised for names that
cannot be consumers.

**Backing.** decision 41, decision 45 and ConsumerTTL, decision 67.

**Rebuttal.** Fail-safe is documented by decision 41; a party who can write `consumers/` could register a legitimate
protected consumer at offset 1 and pin segments the same way (except the TTL would release that one); caps still
override.

**Verdict.** prior-refuted (A1-34 with A1-92, a1-round2-correctness.md item 14; decisions 41 and 67; not re-voted).

**Reproduction (auditor, macOS, `repro7`).** Retention 2 s, PruneInterval 1 s, SegmentBytes 1 KiB, 5 sealed segments.
Control: 5 segments to 1 after 6 s. With a one-byte file `consumers/.DS_Store`: "segments before=5 after 6s=5 first=1".

**Disposition.** Round-1 decision stands. The auditor's proposal (names failing `validConsumerName` plus `_archive` are
not consumers: ignore them in retention, report them in `check` and the Logger; keep fail-safe for valid names; let
ConsumerTTL apply to a broken valid-named entry by mtime) changes decision 41 and needs owner approval; no fix was written.

#### A3-6: Archive keeps an existing destination segment of the same size without verifying it (medium, bug)

**Claim.** Archive keeps an existing destination segment of the same size without verifying its content, reports
success, and commits `_archive` past it. A damaged or tampered destination segment of the right size is never repaired,
even when the operator rewinds `_archive` to re-copy, and the source's protection is released.

**Attack scenario.** Bit rot, a restore or P5 damages a destination segment without changing its size.

**Grounds.** archive.go:176-178 (178 now): if `Lstat(dst)` is regular and `Size() == size`, return success without
reading it.

**Warrant.** The only evidence the destination holds the right data is its length. Atomic write makes a same-size file from
an earlier run complete, not a file damaged afterwards; rewinding `_archive` is the supported way to ask for a re-copy
(decision 41, A1-32), and the damage is exactly what the operator is trying to fix.

**Backing.** Archive's doc comment ("verifying every frame's CRC on the way"); decisions 28 and 67.

**Rebuttal.** The size check exists for idempotence after a crash between the copy and the commit; an attacker with write
access to the destination can corrupt it anyway; `check` on the destination detects the damage. The finding is about false
success and the release of source protection.

**Verdict.** prior-deliberate (decision 85: "a destination file of the same size as the source is kept, so a rerun copies
nothing"; not re-voted).

**Reproduction (auditor, macOS).** Three segments, `archive -to dst` (rc 0); overwrite `dst/0000000000.log` with random
bytes of the same size; `commit -name _archive -offset 1`; `archive -to dst` returned rc 0 and `consumers` showed
`_archive` at 6 (lag 0); `quill -dir dst check` said UNHEALTHY ("segment ... damaged at byte 0 ... torn frame") and
`stat` showed "hole 1-3".

**Disposition.** Round-1 decision stands. The auditor's proposal (verify an existing same-size file with `verifySegment` or
compare it with the source, and re-copy when it differs) is recorded for the owner; no fix was written.

#### A3-29: no detection of filesystems where flock is not coherent (medium, gap)

**Claim.** Quill does no detection of filesystems where flock is not coherent between holders, and the documentation does
not name the filesystems most developers share a log directory over. On a virtiofs share (Apple container, and by inference
Docker Desktop, Lima, Colima) a host writer and a container writer both open the same directory without ErrLocked; both
acknowledge 30 durable flushes and the log ends corrupt.

**Attack scenario.** Not an attacker: a developer bind-mounts a host directory into a container (P10).

**Grounds.** internal/fsx/os.go:93-98 (`TryLock` is a bare `syscall.Flock`; no filesystem-type check anywhere in Open,
log.go:346-395).

**Warrant.** flock on virtiofs from a guest kernel is guest-local, so a holder on the host and a holder in the guest never
see each other, and Open has no other mutual exclusion.

**Backing.** scope.md "Filesystems" and decision 51 (supported: local POSIX with working flock; unsupported: NFS, SMB, FUSE;
bind mounts at the operator's risk); guarantee 5.

**Rebuttal.** The limitation is documented (userguide.md:267-269, scope.md:318-319, decision 51, SECURITY.md:34) and
virtiofs is FUSE-protocol based. The "bind mounts at operator's risk" sentence is overly pessimistic: a same-kernel bind
mount was verified exclusive. What is missing is a named list, the consequence (two acknowledged writers, corruption, no
error) and any warning. Real NFS, SMB and FUSE were not available, so those claims are inferred from the docs.

**Verdict.** prior-deliberate (decision 51 and scope.md Filesystems; not re-voted).

**Reproduction (auditor).** macOS host `lab hold`, then `container run -v share:/share alpine:3.20 /share/lab try /share/L`
printed "try open err: <nil>" (mount shows `virtiofs on /share type virtiofs`); the host's own second try printed
ErrLocked. `cmd/loop` on the host and in the container on one directory: HOST acknowledged 30, CTR acknowledged 30 (head
45); `quill check` UNHEALTHY ("segment 0000000000.log damaged at byte 285 ... bad frame"). Positive control on Linux ext4
with `unshare -rm` and `mount --bind`: the second Open returned ErrLocked.

**Disposition.** Round-1 decision stands; the documentation proposal (name virtiofs, 9p and Docker-Desktop-style host
mounts, state that the failure is a silent double writer, correct the bind-mount sentence) is recorded; the optional
`fstatfs` warning through Config.Logger is not written.

**Update 2026-11-03 (owner-approved; documented in d916114, fix-skeptic FIXED-VERIFIED, 6 of 6 lenses, reported).** The
owner took the recommended answer: keep the behaviour, no detection, name the filesystems. The README, userguide, scope.md,
the godoc of Open, Config.Dir and the package, SECURITY.md, decisions 51 and 62 and the CHANGELOG now say that the
single-writer lock and a Reader's view need a local filesystem (ext4, xfs, btrfs, APFS) written by one host or kernel;
that NFS, SMB/CIFS, FUSE (sshfs, s3fs) and directories shared between kernels (virtiofs, 9p, a container bind mount of a
host directory) are unsupported and are not detected; and that two writers there can both believe they hold the lock. The
"bind mounts at the operator's risk" sentence is corrected: a same-kernel bind mount stays exclusive (the positive control
in the reproduction above). The `fstatfs` warning stays unwritten. The 9p entry is listed as unsupported without a
measurement, and NFS, SMB and FUSE were not exercised (live-integration.md); the text says they are a documented limit, not
a measurement.

### LOW

#### A3-8: `quill.tail` accepts a segment id the file names cannot hold (low, bug)

**Claim.** `decodeTail` accepted any `maxseg` (and `seg`) up to 2^64-1, but segment names hold ten decimal digits. A
`maxseg` at or above 9999999999 makes the writer create 11-digit segment names no listing recognises, which bricks the
log on the next Open; 2^64-1 wraps the next id to 0 and every rotation fails.

**Attack scenario.** A hostile or corrupted marker (P2 or P5); a single bit flip cannot produce it from a small number.

**Grounds.** state.go:127 and :152 checked only `seg <= maxseg` and `off <= 1.5 GiB`; `rotateLocked` used
`id := l.maxSeg + 1` (flush.go:153); `segmentName` is `%010d` (state.go:36) and `parseSegmentName` requires exactly ten digits
(state.go:39-45); load.go:417 sets `maxSeg = max(tl.MaxSeg, newest)`.

**Warrant.** The marker's range was not tied to what the file-name scheme can represent, so a damaged marker turns a valid
rotation into a state the directory scan cannot see; records appended to the invisible segment are acknowledged but
unreachable until an operator fixes the marker.

**Backing.** docs/format.md (ids are `{id:010d}`; no id limit was stated); guarantee 9.

**Rebuttal.** Needs a writer of `quill.tail` (excluded) or a corruption that keeps valid JSON; no legitimate log reaches
10^10 segments; impact is availability, not loss.

**Verdict.** CONFIRMED (2 of 2). SECURITY.md lists hostile or damaged files on disk as defended, so an out-of-range marker is
a gap in the stated defence.

**Reproduction (verified).** macOS: `printf '{"seq":3,"seg":0,"off":39,"maxseg":9999999999}\n' > quill.tail; quill roll` returned
rc 0 and created `10000000000.log`; afterwards every Open or `recover` failed "corrupt log: quill.tail names segment
10000000000, the newest on disk is 0". With maxseg 18446744073709551615, `roll` failed "file exists" (id wrapped to 0) and
`append` still worked. One skeptic used `off` 26 to match a 26-byte segment (the auditor's `off` 39 fails earlier with a
different error) and still reproduced both failures.

**Disposition.** Fixed in bb646c0: `decodeTail` (fast and slow path) refuses `maxseg`, and so `seg`, above 9999999999 (the new
unexported `maxSegmentID`), and `rotateLocked` refuses to seal into an id above it, which covers the 2^64-1 wrap. format.md
and the changelog say so. Fix-skeptic **FIXED-VERIFIED**, 7 of 7: both new tests fail on the parent; the built CLI on
macOS and Linux gave: maxseg 9999999998 rolls to `9999999999.log`; 9999999999 makes `roll` fail "segment ids exhausted" with
the marker and segments intact and `recover` and `append` working; 10^10 and 2^64-1 are refused by every verb as "above
the largest segment id". `FuzzDecodeTail` for 15 s passed. Residual: the comment at state.go:112 and decision 105 describe the
fast-path acceptance conditions without the new bound (incomplete, not false); the 2^64-1 half of the rotation test also
passes without the fix (the simulator's `O_EXCL` catches the wrapped id).

#### A3-10: a damaged oldest segment is skipped silently by a default-start read (low, gap)

**Claim.** A damaged or truncated oldest segment makes `quill cat` and `read` (default start) and `Log`/`Reader.ScanAll`
silently start at the first good segment and exit 0. The lost range appears only in `stat` and `check`, although decision 38
says reads never skip a gap and scope.md says a damaged segment is "reported as a gap, never skipped".

**Attack scenario.** Anyone who can corrupt the oldest segment can hide its records from a bulk reader without an error
(P2, P5).

**Grounds.** The CLI default start is `st.First` (cmd/quill/read.go:118-122); `scanFrom` with offset 0 starts at `v.first`
(read.go:155-157); a new consumer also starts at `first` (consumer.go:101); `v.first` is `segs[0].first`, which skips the
leading bad segment (reader.go:486-490), so the hole is recorded but not reported to a scan that starts after it.

**Warrant.** A damaged leading segment and a retention-deleted prefix look the same to the default start, although one is
data loss and the other is expected expiry.

**Backing.** decision 38; scope.md model line 3; decision 102 (a bad first segment is a hole from 1); decision 11 and
guarantee 6 ("never skipping it").

**Rebuttal.** A read at an explicit offset below `first` does return the GapError, and `stat` and `check` report the hole;
"first retained sequence" is deliberately where a scan starts; decision 98 treats the oldest damaged segment as deletable.
This may be intended, in which case the fix is documentation. No test asserts the default-start behaviour either way
(inferred from a grep by one skeptic).

**Verdict.** CONFIRMED (2 of 2). Not tested by the skeptics: `Log.ScanAll` and a fresh Consumer (they follow by reading,
since `v.first` is built the same way in `Log.view`).

**Reproduction (verified).** macOS: after a same-size garbage overwrite of the oldest segment, `quill -dir dst stat` printed
"first 4", "records 2", "hole 1-3", but `quill -dir dst cat` printed only records 4 and 5 with rc 0. `cat -from 1` failed
with "records 1 to 3 are unavailable", rc 1. Both skeptics reproduced it with five 1500-byte records in two segments.

**Disposition.** Fix-failed. The fix written (54b99d0, not in the tree) makes a read from the beginning, `tail -n` reaching
the oldest record and a new consumer start at sequence 1 when a damaged oldest segment is a hole from 1, so they return its
GapError (CLI exit 1); a prefix deleted by retention, `-since` and `OffsetForTime` still start at `First`. The fix-skeptic
found the code sound (the new tests fail without it, the race run passes, the API surface is unchanged) and rejected it on
the no-new-gap lens only: three statements still said a new consumer starts at the first retained sequence (the godoc for
`Log.Consumer`, consumer.go:330-331, the comment on `consumerStore.open`, consumer.go:78-79, and the
`Config.ProtectedConsumers` row at scope.md:121), including the consumer a writer creates for each protected name. Needed:
update those three with the same exception decision 41 and the scope.md Consumer row now carry; no code or test change.

**Owner approval and second attempt (2026-11-02).** The owner approved the recommended answer (the behaviour change is
intended). A second attempt, 64837ac (later rebased and landed, see below), cherry-picked 54b99d0, updated the three statements above, and also fixed a root cause the first attempt
left: `deleteLocked` (retention.go) dropped the hole from 1 when the retention pass deleted the clean prefix in front of a
damaged segment, so a live `Log` still skipped it while a `Reader` and a fresh `Open` reported it (reported, from the fixer's
summary). It was rejected again (reported, from the ledger's objection; no per-lens table is recorded): decision 84
(decisions.md:92) still says `ScanAll`, and `Scan` with offset 0, start at the first retained sequence; decisions 9 and 67
and two user-guide lines (318 and 335 in the fix tree) still state the old rule without the exception. The skeptic judged
the code and tests sound. Disposition: fixed and on `main` (2026-11-02) as badb254 (the second attempt rebased), 18662b6 (decisions 9, 67, 84, 85 and the two
user-guide lines) and ed518a2 (two `archive.go` comments and a user-guide note); full gate green on macOS and Linux (verified). The
last fix-skeptic passed the code lenses and rejected for the two comments; no skeptic re-ran after they were fixed. Open points
and the exact sign-off: stage2-owner-decisions.md, section "A3-10".

#### A3-12: segments and directories go through the umask; decision 51 said "exact" (low, doc)

**Claim.** Decision 51 says "offset files, quill.format, quill.tail and segments are exact" and the Perm row says directories
get matching execute bits, but segments and directories are created through the umask. Perm 0660 under umask 022 yields 0640
segments and 0750 directories, which silently denies the group's write access to `consumers/` that the documented group
arrangement needs.

**Attack scenario.** None; the mode is never wider than requested. It is a failed-closed mismatch.

**Grounds.** Segments are created with `OpenFile(..., cfg.Perm)` and never chmodded (flush.go:155, log.go:514); directories
with `Mkdir(dirPerm)` (log.go:447, 498); only atomicfile files are chmodded (internal/atomicfile/atomicfile.go:67).

**Warrant.** The decision text overstated what is exact, and a group consumer cannot create its files in a 0750
`consumers/`. Two corrections from the skeptics: the scope.md Perm row (line 123) says only "directories 0700"; the "matching
execute bits" wording is the Config.Perm comment at log.go:71; and decision 51 does not claim directories are exact.

**Backing.** decision 51 and decision 74 ("Chmod sets the exact mode after a create").

**Rebuttal.** Failure is closed, the lock-file exception is already documented, and operators sharing a log are expected to
set the umask.

**Verdict.** CONFIRMED (2 of 2). One voter relied on code reading and a grep for chmod and umask handling (nothing else in
non-test code); the other reproduced it.

**Reproduction (verified).** macOS, `(umask 022; Open{Perm: 0660})`: segments, `quill.lock` and consumer locks `-rw-r-----`;
`quill.format`, `quill.policy`, `quill.tail` and consumer files `-rw-rw----`; the log directory and `consumers/`
`drwxr-x---`. The fix-skeptic repeated it, with Archive, on macOS and Linux with identical output.

**Disposition.** Fixed in 1f896c7, documentation and comments only (the author took the finding's second option): the text now
says that the writer's own segments, every lock file and every directory go through the process umask, that state files and
the segments Archive copies are chmodded exactly, and what a group arrangement needs (umask 002 for the writer, or a chmod).
Fix-skeptic **FIXED-VERIFIED**, 6 of 6; its test lens is "not applicable, existing tests pin the exact-mode path", and the
probe above matched every statement of the new text. Residual nit: archive.go's comment lumps the source's consumer files
under "exactly" while `_archive.lock` beside them is umask-governed; format.md and decision 51 state the lock-file exception.

#### A3-20: a Reader serves overlapping segments without detecting them (low, bug)

**Claim.** When a sealed segment holds frames past the next segment's first sequence, `Reader.Read` and `ScanAll` deliver the
earlier segment's records for sequences the next segment also claims. Writable Open refuses the same directory with
ErrCorrupt. Check flags it, but with a nonsensical hole ("sequences 11 to 4 are missing").

**Attack scenario.** A hand-built or mis-restored directory (P2); quill itself never produces it, since segments are
immutable and ids never reused.

**Grounds.** reader.go (~410-420) derives a sealed segment's range as `[first, next.first-1]`; the post-scan adjustment handles
only `res.last < last` (read.go ~158); the `scanSource` walk callback (read.go:217-224) checks `rec.Seq > v.head` but never
`rec.Seq > s.last`.

**Warrant.** The Reader's contiguity check runs on derived ranges, where an overlap cannot exist by construction, so it is
served silently, which `Log.load` (contiguity on real last values) does not do.

**Backing.** guarantee 6; decision 102 ("a range that starts at or below the one before is an overlap, an ErrCorrupt").

**Rebuttal.** Needs a hand-built directory; every served record is a CRC-valid frame in strictly increasing order; Check does
report it as UNHEALTHY.

**Verdict.** CONFIRMED (2 of 2).

**Reproduction (verified).** In-package test: segment 0 holds seq 1..10 (`seg0-N`), active segment 1 holds seq 5..8 (`seg1-N`),
marker seq 8. Writable Open: ErrCorrupt ("segment 1 starts at 5, inside the one before (segment 0, ends at 10)").
`Reader.Read(1,100)` and `ScanAll`: seq 1..8 with nil error, payloads `seg0-5..seg0-8` for 5..8; `Read(5,100)` on the same Reader
returned `seg1-5..seg1-8`. Check: healthy=false with "sequences 11 to 4 are missing". Both skeptics reproduced it.

**Disposition.** Fix-failed. The fix written (e0df757, not in the tree) makes a Reader refuse overlapping segments as ErrCorrupt
(`view` passes the scanned last to the existing `contiguity` helper; `ensureIndex` returns ErrCorrupt when its scan runs past
the bound) and Check report an "overlap" problem, with "overlap" added to the closed `kind` list for `quill check -json` in
scope.md. The fix-skeptic found Read, Scan, ScanAll, Records, Follow, OffsetForTime, Consumer reads, `quill cat` and Check
fixed, but Archive, another path through `Reader.view`, not: `copySegment` (archive.go:167-192) verifies CRCs and returns the
segment's real last sequence but never compares it with the reference's derived last. On a hand-built layout (segment 0 =
1..10, segment 1 = 5..8 sealed, segment 2 = 9..12 active, marker 12) the fixed code copies both overlapping segments, writes a
destination `quill.tail` with seq 8, and returns GapError{From:11,To:4} ("records 11 to 4 are unavailable"), identical to
before the fix; in the two-segment layout it copies segment 0 and fails "offset out of range: 11 is outside [1, 9]". Needed: an
overlap check in `copySegment` or `archive` (reject when `info.last.Seq` is above the reference's bounded last, or run the
reference's `ensureIndex`) and an Archive test.

**Owner approval and fix (2026-11-02).** The owner approved the recommended answer, including the new `overlap` value in the
closed `kind` list for `quill check -json` (scope.md:206). Disposition: **fixed in 53a5d3b**, fix-skeptic
**FIXED-VERIFIED**, 7 of 7 lenses pass (reported). The commit cherry-picks e0df757 and adds the check the first skeptic asked
for: `copySegment` compares the verified `info.last.Seq` with the derived `s.last` and returns ErrCorrupt before anything is
written (archive.go:235-236, now). A Reader refuses the layout as ErrCorrupt and `Check` reports kind `overlap`
(stat.go:247, now). `reader_overlap_test.go` has three Reader and two Archive subtests on the layout above; with the fix
reverted all five fail for the claimed reasons, and each half (archive.go, reader.go) is load-bearing on its own (reported,
from the skeptic's reverts). On real files the built `cat` and `archive` exit 1 with "corrupt log: segment 0 holds sequence
10, inside the next segment (starts at 5)", the destination holds no segment and a zero marker, and `check -json` gives
`overlap` with exit 3 (reported). Residual, stated in the CHANGELOG: a fresh Reader whose first read starts in the later
segment (`Read(5)`, `Read(9)`) still serves that segment's own records, because the earlier segment is never scanned; `Stat`
and `OffsetForTime` do not scan. Two optional wording nits: decision 85 and the ErrCorrupt comment in errors.go are not
amended. See stage2-owner-decisions.md, section "A3-20".

#### A3-21: a non-minimal varint passes Check and Open, then breaks the read cursor (low, bug)

**Claim.** A frame that uses a redundant (non-minimal) varint passes Check and writable Open, but any read that ends a batch
after it resumes at a wrong byte and fails with "corrupt log: ... torn frame". Verification and the read cursor disagree
about where the frame ends.

**Attack scenario.** A hand-crafted or foreign-written segment (P2). With a crafted file (CRC bytes and the next frame
chosen by its author) the misaligned start could land on bytes that decode as a different valid frame (inferred from the code,
not built; the fix closes it because every accepted frame's real length now equals `frameLen`).

**Grounds.** codec.go `decodeFrame` and `decodeBody` use `binary.Uvarint`, which accepts overlong encodings; the CRC covers only
the body; read.go:237 sets the resume cursor to `pos + int64(frameLen(&rec))`, which recomputes the canonical length, while
`walk` uses the real total.

**Warrant.** After the first record of a batch the cursor points one byte before the real end of frame 1, and the next scan
starts inside the CRC.

**Backing.** docs/format.md says uvarint is "the unsigned LEB128 of encoding/binary (AppendUvarint)", which is canonical, but
the decoder did not enforce it; guarantee 6.

**Rebuttal.** The writer never emits non-minimal varints; under default settings a batch boundary falls inside a segment only
after 16 MiB of payload or at the live end of a Follow.

**Verdict.** CONFIRMED (2 of 2).

**Reproduction (verified).** `internal/repro/noncanon`: frame 1's length prefix padded by one byte (`0x8c 0x00`), frames 2 and 3
normal: "Check healthy: true"; `Records` with MaxReadBytes 1 yielded record 1 then "corrupt log: the active segment below
the durable offset: torn frame"; writable Open returned nil and the same `Log.Records` error. A control without padding read
all three. Both skeptics reproduced it. Not built: the sealed-segment GapError variant.

**Disposition.** Fixed in d6dc2b9: a helper `uvarint()` in codec.go wraps `binary.Uvarint` and returns n=-1 when `n != uvarintLen(v)`;
it is used in `decodeFrame`, `decodeBody` and `window.frameAt`, so Check, writable Open and every read agree that such a frame is
corruption. format.md and the changelog are updated. Fix-skeptic **FIXED-VERIFIED**, 7 of 7: both new tests fail on the parent;
five variants (padded length prefix in the active and in a sealed segment, and CRC-correct crafted frames with padded seq, `at`
and payload length) were all accepted before and are all refused after through the public API and the built CLI (`check` exit 3);
`hack/fuzz.sh` at 5 s per target passed for all seven targets. Residual, pre-existing and not part of this claim: writable Open
does not verify sealed segments, so it still accepts a sealed segment with a padded varint, as it does for a flipped byte, and
the CLI `read` over a damaged sealed segment skips its records without an error.

#### A3-22: `walk`'s contiguity test wraps at 2^64-1 (low, bug)

**Claim.** `walk`'s test `rec.Seq != prev+1` wraps at 2^64-1, so a frame chain MaxUint64-2, MaxUint64-1, MaxUint64, 0, 1, 2
is accepted by writable Open, although format.md says a log whose last frame names 2^64-1 is refused and sequences are never
assigned 2^64-1.

**Attack scenario.** A hand-written segment with valid CRCs (P2); the writer cannot reach it (it stops at ErrSeqExhausted).

**Grounds.** load.go:190, `if (havePrev && rec.Seq != prev+1) || (!havePrev && rec.Seq == 0)`; the only 2^64-1 guard (`head ==
math.MaxUint64`, load.go:396) looks at the last frame only; the first-frame rule `Seq == 0` is not applied to later frames.

**Warrant.** `prev+1` is 0 when `prev` is MaxUint64, so seq 0 follows it, and the log then reports First greater than Head.

**Backing.** docs/format.md "seq" (the writer never assigns 2^64-1); guarantee 1 (strictly increasing seq). A skeptic noted
format.md:35 literally refuses only a log whose last frame or marker names 2^64-1, so the broken guarantee is strictly
increasing seq.

**Rebuttal.** Needs a hand-written segment; no panic or out-of-bounds access was observed.

**Verdict.** CONFIRMED (2 of 2).

**Reproduction (verified).** `internal/repro/wrap` (hand-built newest segment, marker seq 0 off 0): "wrap: writer Open err = <nil>",
"head=2 first=18446744073709551613 records=0", then "next append seq 3". A skeptic repeated it with the repo's simulator, with
the marker at seq 0 and at seq 2.

**Disposition.** Fixed in 46ff40f: one extra term in the condition at load.go:190, `prev == math.MaxUint64 ||`, plus a doc-comment
sentence, so `walk` accepts no frame after one numbered MaxUint64. The author did not take "reject Seq == MaxUint64": a last
frame at MaxUint64 is already refused as ErrCorrupt by the head check, and rejecting it in `walk` would have turned an
ErrCorrupt open into a silent truncation of the tail. Fix-skeptic **FIXED-VERIFIED**, 7 of 7: with the term reverted the new test
fails ("Open accepted the wrapped chain: head 2 first 18446744073709551613"); six hand-built chains (single and two-segment
wrap layouts) were all refused through Open, OpenReader and Check.

#### A3-25: `checkDuplicateKeys` loses the parent object after a container member (low, bug; latent)

**Claim.** `strictjson.checkDuplicateKeys` loses track of the enclosing object after any array or object member, so a duplicate
key that follows a container-valued member is not detected, contrary to format.md ("a duplicate key ... refused").

**Attack scenario.** None today: the only caller (state.go:142; 146 at audit) decodes into a struct of four `*uint64` fields with
`DisallowUnknownFields`, so a container value is already a decode error and no document with a duplicate key survives. It would
become live if a state struct gained a container field.

**Grounds.** internal/strictjson/strictjson.go:93-135: `top` is read before the switch, and on a closing delimiter the frame is
popped but `top.expectKey = true` is applied to the popped frame, never to its parent.

**Warrant.** After a nested value closes, the parent's `expectKey` stays false, so the next key string is taken for a value and
the following key is never recorded.

**Backing.** docs/format.md State files; guarantee 9; A1-2 (89957ae, case-variant duplicates), decision 72.

**Rebuttal.** Not reachable today, as above: a latent defect, not a present bypass. Both skeptics said "contrary to format.md"
overstates the impact.

**Verdict.** CONFIRMED (2 of 2), low and latent.

**Reproduction (verified).** In-package test: `{"a":1,"A":2}` returns ErrDuplicateKey, but `{"a":[],"b":1,"B":2}`,
`{"a":{},"b":1,"b":2}` and `{"a":{"x":1},"a":2}` all return nil; the same container inputs through `Unmarshal` fail in the decoder with
a type error.

**Disposition.** Fixed in e6602b8: after the pop, `top` is reassigned to the parent frame (nil at the root) so the shared "value
consumed" step lands on the parent. Fix-skeptic **FIXED-VERIFIED**, 7 of 7: with the parent file reverted 5 of the 6 new error cases
fail ("got <nil>, want ErrDuplicateKey"); a 200,000-document differential test against an independent recursive reference
(62,455 containing a duplicate) had zero mismatches after the fix and failed at once before. Residual: the "after two containers"
test case passes even on the unfixed code, so it adds no coverage.

#### A3-30: a long-lived Reader or Consumer never re-verifies the log identity (low, bug)

**Claim.** If the log directory is replaced by a different log while a Reader or Consumer handle is live, the handle serves the
new log's records at the old offset, and `Consumer.Commit` writes a consumer file stamped with the old log ID into the new log. A
fresh open of that consumer fails ErrWrongLog, and the stamped file pins the new log's retention through the broken-consumer path.

**Attack scenario.** An operator swaps or recreates the directory under a running consumer (P7, P8). The bypass works only when the
new log is at least as long as the old offset (a shorter log just waits, and Commit fails ErrOffsetRange).

**Grounds.** reader.go:96-117 fixes `r.id` at OpenReader; `readTail` (reader.go:136-158) and `view` (reader.go:321) never re-read
`quill.format`; consumer.go:122-163 checks ErrWrongLog and ErrAhead only in init; `Commit` (consumer.go:197-221) writes state that
carries the old ID, with no identity check, and its range check (consumer.go:211) uses only the marker head.

**Warrant.** ErrWrongLog was a property of the open call only.

**Backing.** scope.md errors list (ErrWrongLog is "a consumer file that belongs to another log"); decision 6. userguide.md:264 limits
the rule to "while no writer holds it" and does not mention readers.

**Rebuttal.** The scenario needs an operator to swap the directory under a running handle; restarting the consumer would hit
ErrWrongLog; no doc promises handle-level identity checking, so the "bug" label is arguable.

**Verdict.** CONFIRMED (2 of 2).

**Reproduction (verified).** `cmd/swap` (macOS): logs A and B with different IDs; Reader and consumer `c` open on A; `c.Commit(31)`; rename
A to A.old and B to A. `r.Read(31)` returned B's records (`B-000031`) with nil; `c.Records` yielded 30 records from B; `Commit(61)`
succeeded and `consumers/c` in the new log read `088121bf... 61 0` while `quill.format` said `8da2f230...`; a fresh open gave "consumer
belongs to another log ... ErrWrongLog". Both skeptics reproduced it.

**Disposition.** Fixed in 495e457: `Consumer.Commit` and `Reader.wait` (so ReadWait and Follow, for Reader and Consumer) re-read `quill.format`
through the new unexported `checkLogID` (built on `loadFormat`) and return ErrWrongLog when the directory holds a different log. Plain
Read and Scan are deliberately not re-checked, as the proposed fix specified; the changelog and user guide say so. Fix-skeptic
**FIXED-VERIFIED**, 7 of 7: the new test fails on the parent for Commit and for ReadWait; on macOS with the real filesystem Commit(61)
returns ErrWrongLog and no file lands in the new `consumers/`, ReadWait and Follow return ErrWrongLog, and a plain Read still serves
the new log. Residual: the ErrWrongLog godoc (errors.go:35) and userguide.md:105 still describe only a consumer file.

#### A3-33: a relative Config.Dir stays relative, so a later chdir moves the log (low, hygiene)

**Claim.** Open and OpenReader keep a relative `Config.Dir` as a relative path, so any later `os.Chdir` in the process redirects every
later file operation (marker publish, rotation, consumer files). The writer's flushes then fail with ENOENT, or write markers and
segments into an unrelated directory with the same relative path.

**Attack scenario.** None by an outside party; a host program that chdirs after opening (P7).

**Grounds.** log.go:351 (`filepath.Clean(cfg.Dir)`, no Abs), log.go:357 and 342 (`l.path` joins `l.dir`), reader.go:96.

**Warrant.** Paths are resolved by the kernel at each syscall against the process cwd; only the segment descriptor is bound to the inode.

**Backing.** The Config.Dir doc (log.go:21) does not require an absolute path; scope.md's Dir row says it "is the log's directory".

**Rebuttal.** The failure is loud (an error from Flush and Close); library users rarely chdir after opening.

**Verdict.** CONFIRMED (2 of 2). The wrong-directory variant was not constructed by either voter; it follows from kernel path resolution
(inferred) and the fix-skeptic's test demonstrates it.

**Reproduction (verified).** `cmd/chd` (macOS): `Open Dir:"data/log"`, flush ok, `os.Chdir` elsewhere, Append: "flush after chdir: open
data/log/.tmp-...: no such file or directory"; a reader in the original location still saw marker head 1 (data durable but invisible);
Close returned the same error.

**Disposition.** Fixed in 837c860: Open takes `filepath.Abs(cfg.Dir)` (replacing `filepath.Clean`) before `resolveDirLink`, and OpenReader takes
`filepath.Abs(dir)`. A regression test, a Config.Dir doc sentence, a scope.md Dir row note and a changelog line were added.
Fix-skeptic **FIXED-VERIFIED**, 6 of 6: with the parent code the new test fails ("the same relative path in the new directory was written
to: [- quill.tail]", the decoy-directory variant); it passes after, also on Linux. Residual: if the working directory has been deleted,
Open and OpenReader with a relative directory now fail in `Abs` where the relative path used to be tried; judged negligible.

#### A3-18: a Reader's read cost scales with the log, not the request (low, degradation)

**Claim.** Every `Reader.Read`, `ReadWait`, `Scan` or `Consumer.Read` that does not carry an iterator cursor re-walks and CRC-checks the active
segment from byte 0 up to the requested offset, and every call rebuilds a per-segment view in O(number of sealed segments). This
contradicts scope.md's "Read cost does not grow with backlog".

**Attack scenario.** None; a polling consumer that issues a fresh Read from its committed offset per batch is quadratic in the active
segment's size. SegmentBytes may be 1 GiB, so one call can read about a gigabyte.

**Grounds.** read.go:203 `start := int64(0)` when there is no anchor (203 now); the active segment's segRef has no index (reader.go:478,
decision 63: "the active segment is never cached"); the per-call `ReadDir` plus a `segRef` and two closures per segment (reader.go:334),
unlike `Log.sealedRefsLocked` (decision 103).

**Warrant.** `scanSource` skips ahead only from an anchor or a cursor; with neither, `walk` starts at the beginning of the segment and verifies
every frame's CRC to reach the offset.

**Backing.** scope.md "Performance gate" ("Read cost does not grow with backlog: ... the sparse index seeks within a segment") and decision
63. The T19 backlog test compares only a start read with a tail read on tiny segments, so it cannot catch this (inferred by a skeptic).

**Rebuttal.** Decision 63 chose not to cache the active segment, for correctness after a writer crash; Records and Follow avoid the cost with a
byte cursor; with the default 8 MiB segments the per-call cost is at most a few milliseconds; the segment-count cost needs hundreds of
thousands of segments.

**Verdict.** CONFIRMED (2 of 2). Both skeptics said the headline is overstated: the active-segment cost is bounded by SegmentBytes and does
not grow with backlog.

**Reproduction (verified).** Linux box (3,000,000 records, 343 MB in one active segment): `Reader.Read(10,1)` 0.85 ms (auditor) or 70 us (skeptic);
`Read(1,000,000,1)` 135 ms; `Read(2,000,000,1)` 253 ms; `Read(2,999,000,1)` 379 ms; the writer's `Log.Read(2,999,000,1)` 112 us. 100 Reads of 1,000
records from 2,000,000 took 26.3 s against 0.33 s for 100,000 records through Records (a skeptic: 22.5 s and 322 ms). 200,000 one-record
sealed segments: every `Reader.Read(n,1)` took 270 ms and allocated 79 MB (the skeptic, at n = 5,000 to 40,000, measured 5.4 to 51 ms and 2.4 to
16 MB per call, linear in the segment count). A smaller macOS run (1,000,000 records in one segment): 0.53, 44 and 53 ms against 0.22 ms, 17
us and 22 us. The 200,000-segment case was not re-run by a skeptic.

**Disposition.** Held for the owner; see stage2-owner-decisions.md, "A3-18".

#### A3-19: the first-frame probe reads and CRC-checks the whole first frame (low, perf)

**Claim.** The Reader's first-frame probe of each sealed segment reads, allocates and CRC-checks the whole first frame, up to 64 MiB, to learn two
integers. A directory of sparse files makes `quill stat` or any first read cost about 8,000 times the disk it occupies. Decision 47 says only "a few
bytes" are read.

**Attack scenario.** P5, or a log restored from another party, plants valid 64 MiB frames at no disk cost (CRC32 over zeros is computable offline);
each cheap inode forces a 64 MiB read and a 64 MiB allocation per Reader process (the cache is per Reader).

**Grounds.** reader.go:238-240 (`w := &window{r: f, size: size}; rec, _, ferr := w.frameAt(0)`; line 249 now); `window.frameAt` (load.go) reads the
full frame and verifies it. The allocation is bounded at 64 MiB, so guarantee 7 is met only in the weakest sense.

**Warrant.** The probe needs seq and `at` from byte 0 but uses the same decode as a scan, which insists on the full CRC.

**Backing.** decision 47 ("a Log or Reader reads only the first frame of each segment (a few bytes, cached per immutable segment id)"); guarantee 7.

**Rebuttal.** Needs write access to a directory the operator reads, or a legitimate log with huge first frames (paid once per Reader); memory per
call is bounded and garbage collected. The CRC is load-bearing: one skeptic patched the probe to read 64 bytes and skip the CRC and two existing tests
that assert hole detection failed, because an unchecked first-frame seq corrupted the segment ranges.

**Verdict.** PLAUSIBLE (votes: CONFIRMED, REFUTED, CONFIRMED). The mechanism is not disputed; the REFUTED voter refuted the proposed fix as not
behaviour-preserving, and the other two said that argument is about the remedy, not the finding. The amplification ratio is unverified.

**Reproduction.** Auditor, Linux box (`internal/repro/amp` B: 100 files, each a valid frame with a 64 MiB-64 payload, sparse; apparent 6.3 GiB, on disk 816
KB): `quill stat` took 8.99 s (7.9 s system); `Reader.Stat` reported 19.2 s and 6406 MiB allocated; `quill ls` over a parent holding it 3.7 s cold. A skeptic's
smaller run: 5 sealed segments with 8 MiB first frames, fresh Reader, `Stat()`: 40 MiB allocated, one full frame per segment (verified). The sparse 64 MiB
case, the timings and the 8,000-times ratio were not re-run by a skeptic.

**Disposition.** Held for the owner; see stage2-owner-decisions.md, "A3-19".

#### A3-7: a symlinked `consumers/` directory is followed (low, security)

**Claim.** Only the final component of a file path is opened with `O_NOFOLLOW`. A symlinked `consumers/` makes quill create `<name>.lock` files and write or delete
offset files in an arbitrary directory, and makes the writer's Open delete old `.tmp-*` files there.

**Attack scenario.** P5 replaces `consumers/` with a symlink to a victim directory; the next `register` or `recover` acts there (a confused deputy with a
constrained effect: names `[a-z0-9_-]` plus `.lock`, a fixed one-line content, deletion of `.tmp-*` files older than an hour).

**Grounds.** `consumerStore.path` joins `dir`, `consumers` and the name (consumer.go:46 at audit; 73 now); `OpenFile` uses `O_NOFOLLOW` only on the last element
(fsx/os.go:36-41); `removeTempsBefore` (log.go:624-637) does `ReadDir`, `Lstat` and `Remove` on the resolved directory; `create()` checks only that the name
`consumers` exists, not that it is a directory (log.go:469).

**Warrant.** The kernel follows intermediate symlinks and the code never `Lstat`s the `consumers/` component.

**Backing.** SECURITY.md "Symlink substitution"; decision 51 ("every state file ... is opened without following symlinks"; A1-29 limits the rule to what is
inside the log directory).

**Rebuttal.** Requires write access to the log directory (excluded); no stated guarantee is broken (every file is still opened with `O_NOFOLLOW` on its last
component; Open and OpenReader already follow a symlink to a directory in Dir by decision 51 and A1-29). Cheap to harden.

**Verdict.** REFUTED (votes: REFUTED, CONFIRMED, REFUTED). Two voters held that it is a hardening gap, not a defect against any stated guarantee; the confirming
voter reproduced the mechanism and called it a gap against the wording of SECURITY.md and decision 51.

**Reproduction (verified).** macOS and Linux: `quill -dir lg append hello`; replace `consumers/` with a symlink to an empty `victim/`; `quill register -name notes`
failed with ELOOP (rc 1) but had already left `victim/notes` and `victim/notes.lock`; with `victim/.tmp-abc` dated 2020, `quill recover` removed it. The auditor's
"read victim/notes (refused as malformed)" did not match the confirming skeptic's run (register created a well-formed file); that detail does not change the verdict.

**Disposition.** Deliberately not changed: hardening, outside the threat model, and consistent with the documented scope of the symlink rule. The auditor's
proposal (open `consumers/` once with `O_DIRECTORY|O_NOFOLLOW`, or `Lstat` it in `create()` and every consumer operation; otherwise document the
intermediate-component behaviour in decision 51) is noted in stage2-owner-decisions.md, "Not held, but worth a line".

**Update 2026-11-03 (owner-approved later that day, wording only; b765290, fix-skeptic FIXED-VERIFIED, 6 of 6 lenses,
reported).** Decision 51 and SECURITY.md's "Symlink substitution" paragraph now say that the rule covers the final
component of each state file's path, and that a log directory (and its `consumers/` subdirectory) that an untrusted party
can replace with a symlink is outside the threat model, as write access to the log directory is. A dated note and a
CHANGELOG line carry it. No code changed; the hardening (open `consumers/` with `O_DIRECTORY|O_NOFOLLOW`) is not approved
and is held (stage2-owner-decisions.md, "Held after pass A3r"). The fix-skeptic re-ran the reproduction above (a symlinked
`consumers/`, `register` then `recover`) with the built binary on macOS and in a linux/arm64 container and saw what the
text states (reported).

#### A3-24: `-json` passes bidi and format controls raw; the time field is unchecked (low, security)

**Claim.** JSON output passes Unicode format and bidi controls (U+202A-202E, U+2066-2069, U+200B-200F, U+FEFF) through raw in the `payload` string, while text
output escapes them. The `time` field is not RFC 3339 outside years 0000-9999, and wraps to a wrong instant for extreme `At`.

**Attack scenario.** A producer chooses a payload that visually reorders or hides text in a bidi-aware viewer reading `-json` output. The `At` outliers need a
hand-crafted file, because the writer's clock (an unexported field, append.go:79) is the only source of `At`.

**Grounds.** cmd/quill/print.go:79-100 (`appendJSONString`, line 78 now) escapes only controls, DEL, `<`, `>`, `&`, C1 and U+2028/2029; `appendQuoted` (text) uses
`strconv.IsPrint` and so escapes every Cf character; the time field comes from `time.Unix(rec.At, 0).UTC().AppendFormat(RFC3339)` with no range check.

**Warrant.** The two output modes disagree about what is safe.

**Backing.** format.md ("a payload cannot inject lines or terminal escapes"; the documented set is controls, DEL, C1) and `time` as "the same instant, RFC 3339
UTC"; decision 35 names DEL and C1 as the owner-approved deviation from `encoding/json`; decision 86 says text mode quotes as `strconv.Quote` does.

**Rebuttal.** By the letter of the documentation JSON escapes exactly the listed set; bidi controls are not terminal escapes; the unescaped form decodes
losslessly; the text-mode time field has the same unchecked format.

**Verdict.** REFUTED (votes: REFUTED, CONFIRMED, REFUTED). The behaviour is real and reproduced; it breaks no stated guarantee. A hardening note is
reasonable, since bidi controls were in scope for this pass. How a real bidi-aware viewer renders the raw bytes was not verified.

**Reproduction (verified).** Printer driven in-package with an in-memory record on macOS: payload U+202E U+200B U+202E printed in text mode as escapes and in
`cat -json` as raw bytes (`od`: 342 200 256 342 200 213 342 200 256). At 253402300800 printed "10000-01-01T00:00:00Z"; At MinInt64 printed
"292277026596-12-04T15:30:08Z", the same string as MaxInt64; the `at` field stays exact.

**Disposition.** Deliberately not changed (see the verdict). The auditor's proposal (escape U+200B-U+200F, U+202A-U+202E, U+2060-U+2069, U+FEFF and U+061C as
`\uXXXX`, still lossless; print `time` as null or omit it outside 0..253402300799) is noted in stage2-owner-decisions.md, "Not held, but worth a line".

**Update 2026-11-03 (owner-approved later that day, escape set only; fixed in d15ac45, fix-skeptic FIXED-VERIFIED, 7 of 7
lenses, reported).** `-json` now escapes every Unicode format character (category Cf, 170 code points, which is the proposed
list and the rest of Cf; all of them are rejected by `strconv.IsPrint`, pinned by a test) as `\uXXXX`, with a surrogate
pair above U+FFFF, so text and JSON output agree on that set. The decoded value of every payload is unchanged and a
plain payload prints byte-identically; only the raw bytes of a payload holding those code points change. Other
non-printable code points (U+00A0, private use, unassigned) stay raw in JSON, as the owner did not approve escaping them.
Decision 35 carries the dated note. The `time` field is not range-checked: that part was not approved and is held
(stage2-owner-decisions.md, "Held after pass A3r"). The fix-skeptic wrote a log with every non-surrogate code point and ran
`quill cat -json` before and after: decoded payloads identical, exactly 170 characters changed from raw to escaped, no raw
Cf left (reported). Residual noted by the skeptic: row B5 of the boundary table above still describes the audited tree
("JSON escapes controls, DEL, C1, U+2028/9"); it is the 2026-10-31 text and was left as recorded.

#### A3-9: a Reader takes the mode of its files from `quill.tail` (low, hygiene)

**Claim.** A Reader takes the mode of the files it creates from `quill.tail`'s permission bits (plus owner rw), and atomicfile chmods to that mode exactly, so the
umask no longer applies. A `quill.tail` with mode 0666 (restored from a backup, copied through a filesystem without Unix modes, or planted) makes `quill archive`
write the archive segments and every Reader-created consumer file 0666.

**Attack scenario.** Whoever controls the mode of `quill.tail` controls the mode of newly written copies of the records (P5, or a restore accident); the exposure is
limited by the directory mode.

**Grounds.** reader.go:111-115 sets `r.perm = info.Mode().Perm() | 0o600`; `atomicfile.WriteFrom` chmods to perm (atomicfile.go:62-68); Archive passes `r.perm` for destination
segments, format and tail (archive.go:70, 83, 138, 152); `consumerStore` uses it for locks and offsets (consumer.go:56, 142).

**Warrant.** An untrusted file mode becomes the mode of new files and nothing narrows it; the writer sets `quill.tail` to Config.Perm exactly, but a Reader cannot tell a
configured mode from a restored one.

**Backing.** decision 51 and A1-85 (owner-approved: a Reader reads it from the permission bits of `quill.tail`, no ReaderOptions field); SECURITY.md (0600 by default).

**Rebuttal.** Documented and owner-approved behaviour; the writer reapplies Config.Perm to `quill.tail` at the next flush; the parent directory's mode keeps other users out.

**Verdict.** prior-deliberate (A1-85, decision 51; not re-voted).

**Reproduction (auditor, macOS).** `chmod 666 quill.tail`, then `(umask 077; quill archive -to dst; quill register -name lazy)`: `dst/*.log`, `dst/quill.format`, `dst/quill.tail`,
`consumers/_archive` and `consumers/lazy` were `-rw-rw-rw-`. A 0755 destination directory around 0666 files under umask 022 is inferred, not run.

**Disposition.** Round-1 decision stands. The auditor's proposal (clamp the derived mode, for example `&^ 0o002` and no execute bits, and intersect it with the Reader's umask; or
an explicit `ReaderOptions.Perm`) changes decision 51's text; no fix was written.

#### A3-11: `quill prune` acts on `quill.policy` as trusted configuration (low, gap)

**Claim.** A `quill.policy` with a tiny `retention` or `max_records` makes `quill prune` delete all sealed segments in one pass, with no confirmation and without `-force`.
`quill.policy` is documented as informational and parsed tolerantly.

**Attack scenario.** A stale policy restored from a backup, or edited by another user, changes what the operator's command deletes; adversaries are secondary.

**Grounds.** `cmdPrune` loads `ReadPolicy` and uses its Retention, MaxBytes, MinFreeBytes, MaxRecords and ConsumerTTL (cmd/quill/write.go:121-160); `capsLocked` runs after the clock
guard and overrides protection (retention.go:147-152); `decodePolicy` uses plain `json.Unmarshal` (state.go:314-327), so duplicate keys are accepted. Invalid values fail closed.

**Warrant.** The tolerant parse is fine for display in `ls`, but the same file drives an irreversible deletion.

**Backing.** decisions 29 and 99 (`quill.policy` for tools), decision 68 (offline tools act on the policy), state.go:21-23.

**Rebuttal.** Only someone who can write `quill.policy` can do this, and that someone could delete the segments directly; the behaviour is documented and the operator chose to run `prune`.

**Verdict.** prior-deliberate (decisions 68, 87, 29, 99; A1-91 fixed in 5c603fc; not re-voted).

**Reproduction (auditor, macOS).** Three records in three segments, `quill.policy` = `{"retention":"1ns","max_records":1}`, `quill prune -force`: "segments 3 -> 1, bytes 39 -> 0, first 1 -> 4".
The duplicate-key variant `{"retention":"1h","retention":"1ns",...}` is accepted too.

**Disposition.** Round-1 decision stands. The auditor's proposal (print the policy about to be applied and require an explicit flag when its caps would delete more than the clock guard
allows; parse the policy used for destructive action with `strictjson`) is recorded; no fix was written.

#### A3-23: offline verbs create a log at a mistyped `-dir` (low, degradation)

**Claim.** `recover`, `prune`, `roll` and `append` create a new empty log, with all missing parent directories, at a mistyped `-dir` and exit 0 with a success-looking message. On an existing
non-log directory they refuse but leave an empty `quill.lock` behind.

**Attack scenario.** Operator error (P8).

**Grounds.** `Open` calls `mkdirAll` (log.go ~455) and `create()`; openWriter in cmd/quill does not require an existing log; `takeLock` creates `quill.lock` before `create()` decides the directory is not a log.

**Warrant.** Open's documented creation behaviour is used by verbs whose job is to act on an existing log.

**Backing.** scope.md (`recover` is "open writable, repair a torn tail, republish quill.tail, close"; only `append` is described as able to create a log; Config.Dir "created if missing").

**Rebuttal.** Creating the log is documented for Open and harmless; the new logs hold no data; a user who wants `append` to create a log relies on it; the leftover lock is an empty file.

**Verdict.** prior-refuted (A1-49, a1-round2-correctness.md item 15; decisions 53, 62, 77; not re-voted).

**Reproduction (auditor, macOS).** `quill -dir data/ordrs recover` printed "recovered: head 0, first 1, 1 segments" and exited 0, creating `data/ordrs` with `0000000000.log`, `consumers`, `quill.format`,
`quill.tail`; `quill -dir deeper/a/b/c roll` exit 0 created four directories; `quill -dir plain roll` (a directory holding a file) said "not a log: plain holds files but no quill.format" and left `plain/quill.lock`.

**Disposition.** Round-1 decision stands. The auditor's proposal (recover, prune and roll fail with ErrNoLog unless `quill.format` exists, and take the lock after `create()`'s directory check) is
recorded; no fix was written.

**Update 2026-11-03 (owner-approved; changed in 18145b9, fix-skeptic FIXED-VERIFIED, 7 of 7 lenses, reported).** The owner
took the recommended answer, narrowed by a check the recorder ran against c768634: `ls`, `stat`, `check`, `read`,
`consumers` and `prune` already exited 1 and created nothing at a directory that is not a log, and `archive` was fixed by
A3-14; `roll` and `recover` still created a log and exited 0 (the recorder's check, reported). The fix-skeptic's run of the
parent binary found one more: `prune` with a policy flag (for example `-retention`) also created the log and succeeded,
while a bare `prune` refused. Now `roll`, `recover` and `prune` check for `quill.format` in `-dir` before opening the log; if it is missing they print
`quill: not a log: <abs dir> has no quill.format` (the text the other verbs use, quoted per A3-3), exit 1 and create
nothing, not even the directory or `quill.lock`. A usage error still exits 2 first. `append` creating the log on first use
is intended and unchanged (decision 62; the userguide now says so). `commit`, `register`, `unregister` and `archive`
already refused and are unchanged; `append` at a populated directory that is not a log still refuses and still leaves an
empty `quill.lock`, listed as out of scope in the commit. The check is an `Lstat` of `quill.format`, not `OpenReader`, so
`recover` still repairs a log whose `quill.tail` was lost. The fix-skeptic ran every writable verb against a missing, an
empty, a regular-file and a populated non-log directory with the parent and the fixed binary, and on linux/arm64 in a
container (reported). Decision 53 carries the dated note; the CHANGELOG has a "Changed:" line.

#### A3-31: a forward clock step makes every protected consumer stale under ConsumerTTL (low, degradation)

**Claim.** With ConsumerTTL set, a forward wall-clock step by another party makes every protected consumer "stale" at the next prune pass, so its unread segments are deleted at the clock guard's
pace (about 25% of sealed segments per pass), within about ten passes. Without ConsumerTTL, protection is unaffected.

**Attack scenario.** P9: a party that controls the time source (needs CAP_SYS_TIME or the NTP source).

**Grounds.** retention.go:48-66 (stale = file mtime and the record's `At` both older than `now - TTL`, with `now` from the writer's clock); retention.go:147-151 (the guard, a quarter rounded up).

**Warrant.** Staleness is computed from the same wall clock as the age sweep, with no skew guard of its own; the guard limits only the rate.

**Backing.** scope.md "A wrong clock" (decision 44: "slows the damage so an operator can notice; it does not remove the hazard"); guarantee 12.

**Rebuttal.** The hazard class is documented and ConsumerTTL is opt-in; a warning is logged only if a Logger was configured; it requires a step of at least ConsumerTTL.

**Verdict.** prior-deliberate (decision 44; A1-35, A1-87; not re-voted).

**Reproduction (auditor, in-package fake clock, +400 days, 21 sealed segments, ProtectedConsumers=[slow], Retention 1h).** ConsumerTTL=0: pass 1 deletes nothing (21 -> 21). ConsumerTTL=1h: 21->16->12->9->7->5->4->3->2->1
over ten passes, final first=401 head=400 dropped=20. A real `date -s` was not used (needs root).

**Disposition.** Round-1 decision stands. The auditor's proposal (say in the clock paragraph that ConsumerTTL staleness is also clock-derived; optionally ignore stale when `now` is more than a day past the
newest record's `At`) is recorded; no fix was written.

**Update 2026-11-03 (owner-approved; documented in 1cb2c93, fix-skeptic FIXED-VERIFIED, 6 of 6 lenses, reported).** The
owner took the recommended answer: keep the behaviour, document it. The Config.ConsumerTTL godoc, userguide, scope.md
("A wrong clock", the config table, guarantee 12), architecture.md, decisions 44 and 45 (dated notes) and the CHANGELOG now
say that a protected consumer is stale when the age of its offset file's modification time and the age of its record's `At`
are both at least ConsumerTTL against the writer's clock, so a clock step, a restore that resets mtimes, a copy that does not
keep them or a filesystem with wrong timestamps can make a live consumer look stale or a dead one look live. A future-stamped
file or record has a negative age and is never stale. The clock guard (a quarter of sealed segments, rounded up, per pass)
limits only the age sweep and is lifted by `-force`; staleness itself has no guard. One deviation from the approved text,
found by the fixer and confirmed by the fix-skeptic from retention.go and decision 57: the text said "Protect makes a
consumer exempt from the TTL". That is backwards. ConsumerTTL applies to protected consumers only, and an unprotected
consumer holds nothing back. The documentation states the code's rule, and the CHANGELOG says so. No code changed.

#### A3-32: MinFreeBytes lets another process's disk use drive deletion of this log's history (low, degradation)

**Claim.** With MinFreeBytes set, another process that consumes space on the same filesystem makes this log delete its own sealed history, overriding protected consumers and bypassing the clock guard,
and the code assumes each deletion frees the segment's size rather than re-measuring.

**Attack scenario.** P9: a neighbour's disk use on a shared filesystem.

**Grounds.** retention.go:201-215 (`for free < MinFreeBytes && n < len(sealed) { free += sealed[n].size; n++ }`: one `statfs`, then an estimate); the loop runs after the age guard and `capsLocked` adds segments
without limit.

**Warrant.** A neighbour's disk use is an unbounded deletion trigger for this log's sealed data.

**Backing.** scope.md:233 and guarantee 11 ("with MinFreeBytes set, a log deletes its own oldest sealed segments before it lets the filesystem fall below the floor"); scope.md:98 ("caps override consumer protection").

**Rebuttal.** This is the documented design ("another application or log can use the room first"); on a real filesystem the unlinks free real bytes so deletion stops near the floor.

**Verdict.** prior-deliberate (decisions 24 and 25; scope.md:233; not re-voted).

**Reproduction (auditor, in-package fake `statfs`).** 21 segments, protected consumer `slow` at 1, `statfs` faked to 0: one Prune pass leaves segments=1 first=401, dropped=20. "Delete everything" is the upper
bound under a fake that holds free at 0; a real full disk was not run.

**Disposition.** Round-1 decision stands. The auditor's proposal (re-measure `statfs` after each unlink or batch; say in scope.md that a neighbour's usage is paid for with this log's history, including protected and
unarchived segments) is recorded; no fix was written.

**Update 2026-11-03 (owner-approved; documented in d09e975, fix-skeptic FIXED-VERIFIED, 6 of 6 lenses, reported).** The
owner took the recommended answer: keep the behaviour (including the single `statfs` and the estimate), document it. The
MinFreeBytes, ProtectedConsumers and Consumer.Protect godoc, doc.go, userguide, scope.md (config table, "Disk safety",
guarantee 11), architecture.md, decisions 25 and 57 (dated notes) and the CHANGELOG now state the rule for each cap: the age
sweep stops at the first segment a protected consumer has not passed, while MaxBytes, MaxRecords and the MinFreeBytes floor
override protection; below the floor the writer deletes the oldest sealed segments even if a protected consumer has not read
them, so the consumer can lose records, and the Warn naming the consumer is how that is seen. A new test,
`TestMinFreeBytesDeletesSegmentsAProtectedConsumerHasNotRead`, pins it through the existing `statfs` seam; the fix-skeptic
showed it fails when the MinFreeBytes deletion loop is disabled (reported). No production code changed.

## What was checked and found sound

Each line is an auditor's statement of a check that came out clean. The label is the auditor's: **verified** where it ran the program or read the exact lines. Anchors are as audited.

### Filesystem boundary (A3-filesystem-boundary)

- **`O_NOFOLLOW` coverage (verified by reading every production call site).** Every file open in non-test code goes through `fsx.OS.OpenFile`, which adds `syscall.O_NOFOLLOW` (internal/fsx/os.go:36): state files
  (state.go:335), segments (load.go:76), temp files (atomicfile.go:63, `O_EXCL`), segment creation (flush.go:155, log.go:514, both `O_EXCL`), the lock (lock_unix.go:22), Archive's lock probe (archive.go:116) and `SyncDir`
  (os.go:79). No bare `os.Open` or `os.Create` in library code; the CLI uses `os.ReadDir` on `-dir` only (read.go:32). The gap is that only the final path component is protected (A3-7).
- **Archive destination symlinks (verified).** A symlink planted as the destination `quill.format` makes Archive fail closed (ELOOP); planted as a destination segment or `quill.tail` it is replaced by rename (the link
  is replaced, not the target). The source-equals-destination check uses `os.SameFile` on `quill.tail` with the `Lstat` result, which resolves directory symlinks (archive.go:110-114), so a symlinked destination directory
  is refused.
- **Atomic temp files (verified).** Random 8-byte suffix, `O_EXCL`, `fchmod` on the fd, fsync, rename; failures remove the temp. No predictable name or symlink-race path.
- **Consumer names (verified).** `validName` and `validConsumerName` restrict names to `[a-z0-9_-]`, 1 to 200 characters, no leading underscore except `_archive`, applied in open and remove (consumer.go:53, 273) and in
  `Config.validate` for ProtectedConsumers (log.go:182). No path separator, dot, NUL or traversal is possible through the Consumer, DeleteConsumer, register, commit or unregister API or CLI `-name`; a 200-character name plus
  `.lock` stays under 255 bytes. Names read back from the directory are unvalidated (A3-3, A3-4).
- **Hostile `quill.tail` (verified through the CLI).** `decodeTail` accepts only canonical bytes on the fast path; the slow path (`strictjson`) rejects duplicate keys including case variants, unknown keys, trailing data,
  invalid UTF-8, lone surrogates, missing fields and 20-digit numbers; `off` above 1.5 GiB and `seg` above `maxseg` are refused. Tried: seq 2^64-1 (refused), seq near 2^64 (accepted; head huge, no loop or allocation, commit
  refused), `off` above the file size (corrupt), a duplicate seq key (refused), a huge seq with a short file (`check` UNHEALTHY). Deep nesting and 1 MiB inputs took 4 to 6 ms and 1 to 5 MiB of allocation (the JSON max depth is
  enforced), so it is not a poll-loop denial of service. The one gap was the `maxseg` bound (A3-8).
- **Hostile consumer files (verified).** `decodeConsumer` is strict (three fields, 32-hex ID, plain decimal offset up to 2^64-1, flag 0 or 1, trailing newline); a wrong log gives ErrWrongLog, an offset above head+1 gives
  ErrAhead, and `head+1` cannot overflow (head is capped below 2^64-1). State files above 1 MiB are refused before allocation (state.go:347). `readState` on a directory is refused after the open (type check); only the FIFO case
  blocked (A3-1).
- **Allocation from declared sizes (verified).** `frameAt` rejects a `bodyLen` above `maxFrameBody` and a total beyond the file size before allocating (load.go:157-166); a segment above 1.5 GiB is refused whole (load.go:84);
  window buffers are at most 1 MiB or one frame; a sparse 1.5 GiB file fails at its first (zero) frame without reading the rest; `Read`'s preallocation is capped at 4096 records (read.go:99).
- **Segment type and time of check (verified).** `openSegmentFlag` does `Lstat` before the open and `fstat` after, so a FIFO, device or directory named like a segment is a bad segment and a swap after the open is caught. The
  residual window is a FIFO swapped in between `Lstat` and open (not run; needs directory write).
- **Rename and unlink races (verified by reading).** Segment deletion unlinks by name after publishing the marker, so a replaced name loses the link and not the target; `quill.tail` is replaced atomically; the Reader checks the
  segment size at open and returns ErrCorrupt or a GapError on change. A segment replaced by another valid segment passes the CRC; CRC is not authentication (out of scope per SECURITY.md).
- **Hostile payload bytes (verified).** Text output of `cat` escapes ESC, control, DEL, C1, U+2028 and bidi controls (checked with a payload holding ESC[2J, newline, CR, U+0085, U+202E, U+2028 and DEL); JSON output escapes
  control, DEL, C1 and U+2028. Only the raw U+202E in `-json` output was noted (A3-24).
- **Directory symlink behaviour (verified).** Open and OpenReader follow a symlinked Dir as documented (decision 51, A1-29); a link to a non-directory is ErrNoLog; the CLI `ls` skips symlinked subdirectories
  (`DirEntry.IsDir` is false for a link).

### Input and denial of service (A3-input-and-dos)

- **Decoder and recovery against adversarial segment bytes (verified by execution).** A black-box mutation driver (`internal/repro/fuzzdir`, Linux box, about 8,700 iterations over 3 seeds plus smoke runs) on a 400-record,
  12-segment log, a quarter of whose payloads embedded CRC-valid frames for the next sequence numbers. Mutations: truncation at random offsets, byte flips, zeroed ranges, random markers (stale and ahead), segment delete, swap and
  duplicate, spliced ranges, garbage tails, huge declared lengths, forged valid frames appended. After each, Reader (Stat, Check, Read at six offsets including 2^62 and 2^64-1, ScanAll, Follow, OffsetForTime), Archive and a writable
  Open with ScanAll ran. Result: no panic, no hang, no record served that was not originally written (payload and `At` compared), no order violation. The only hits were the driver's own appended CRC-valid contiguous frames, which
  recovery is meant to keep.
- **Frame injection through parse-forward recovery (inferred by reading load.go and read.go; not reproducible).** Recovery starts at the marker's offset, which must be a frame boundary of an intact chain from byte 0
  (`recoverSegment` fails with a SegmentError if the chain overshoots or stops short); the first bad or torn frame ends the scan and the file is truncated at that frame's start, so a frame embedded in a payload is never reached
  because its host frame fails first. The length prefix is not under the CRC, but the body's own payload length must equal the remainder, so a damaged prefix cannot yield a shorter valid frame. No scan resynchronises on frame bytes.
  The one way found to start a parse inside a frame was the non-canonical-varint cursor mismatch (A3-21).
- **CRC collisions by a chosen-plaintext producer (inferred).** CRC32 is linear, so a producer can set a frame's CRC, but only for its own frame; no code path trusts a CRC for anything but detecting damage, and the format makes no
  tamper-evidence claim, so nothing was filed.
- **Allocation from untrusted lengths (verified by reading and the fuzz runs).** A declared body over 64 MiB is `errBadFrame` before any allocation; a frame longer than the remaining file is `errTornFrame` before any allocation; a
  segment over 1.5 GiB is refused at open; a state file over 1 MiB is refused before the buffer is made; a 64 MiB frame allocates 64 MiB once per scan and the pooled buffer is collectable; `Reader.Read` and `expect()`
  preallocation is capped at 4096 records. (The probe's full-frame read is A3-19.)
- **Panics from the CLI with chosen input (verified by execution).** A Go driver calling `run()` with random verbs, flags and values (empty, signed, octal, hex, overflow, 2^64, durations, `@unix` extremes, RFC 3339 edge dates, ESC,
  invalid UTF-8, `../x`, 201-character names, repeated `-dir`) and random stdin, against a log with escape, bidi and invalid-UTF-8 payloads and consumers. Linux: 257,301 invocations on a clean log in 240 s, then 162,074 on logs whose
  files were randomly mutated in 180 s. No panic, no hang, and `cat`, `read` and `tail` never emitted raw ESC, BEL or invalid UTF-8 (text mode also never emitted raw bidi controls). `follow` was excluded because it blocks by design.
- **Terminal and JSON injection through record payloads (verified by hand).** Every byte value is covered by construction (valid UTF-8 becomes a quoted or JSON string, otherwise base64). ESC, BEL, DEL, C1 (including CSI U+009B and NEL),
  U+2028/2029, `<`, `>` and `&` are escaped in JSON; text mode escapes everything `strconv.IsPrint` rejects. An OSC 52 sequence in a payload is escaped in both modes. `consumers -json`, `check -json` and `stat -json` use `json.Marshal`
  and are safe, including for hostile names. The gap is JSON's raw format and bidi controls (A3-24).
- **Integer overflow and sign (verified by reading and the argument fuzz).** Consumer offsets, `head+1`, `lag()`, `tail -n` arithmetic, `Stats.Records` and hole subtraction, MaxRecords/MaxBytes/MinFreeBytes comparisons,
  `parseDecimal` and the `@unix` 2^63 guard are sound. Not clean: `Config.maxActiveBytes` (A3-16) and `walk`'s `prev+1` (A3-22). Extreme `At` values reach the clock guards (`now-lastAt` can overflow for `At` near MinInt64, which only
  turns off an age rotation) but cannot be produced by the writer.
- **Unbounded growth (verified by reading).** `Reader.segs` is pruned to the listing; Follow and Records batches are bounded by MaxReadBytes plus one record; a read buffer for a failing flush is bounded by MaxBufferBytes;
  descriptors are released on every error path read (`view`, `scanSource`, `copySegment`, Consumer open); no goroutine is started per call. A Reader over N sealed segments is linear in N per call (A3-18).
- **Destructive verbs (verified by reading).** `prune`, `roll` and `recover` need the writer's lock (ErrLocked beside a live writer); `unregister`, `commit` and `register` validate `-name` (no path traversal), and `unregister` takes the
  consumer's lock. Open refuses a non-empty directory without `quill.format` (but see A3-23 and A3-14). Archive refuses a destination that is the source (`os.SameFile`) or has a live writer lock, and one with another log id
  (ErrWrongLog).
- **Symlinks on state files and segments (verified by reading, not re-tested).** State files and segments open with `O_NOFOLLOW` and segments are `Lstat`ed as regular files before open; a symlink as `quill.tail`, a segment or a
  consumer file gives an error, not a follow. The segment path is `Lstat`-then-open, so a swap between the two is the one gap.

### Process boundary (A3-process-boundary)

- **Second writer by an alias path (verified, macOS, `cmd/lab` "double").** A symlink alias and a `..` path both give ErrLocked, because flock is per open file description.
- **Bind mount (verified, Linux ext4 under `unshare -rm` and `mount --bind`).** A writer on one path and an Open through the other returned ErrLocked.
- **SIGKILL of the lock holder (verified, macOS and Linux x86-64).** The lock is released, and a child exec'd by `os/exec` (`sleep 60`) that survives the parent does not keep it: a new Open succeeds right after the kill with the
  child still alive (Go opens `O_CLOEXEC`). Fork-without-exec (cgo daemonizing) was not tested.
- **Probe that takes and releases its own flock (verified, `cmd/ofd`).** Opening `quill.lock`, taking and releasing a flock and closing does not release the writer's lock; there is no POSIX `fcntl` close-any-fd hazard because flock is used.
- **Two processes with one consumer name (verified, `cmd/cons`).** The second gets ErrLocked from `Reader.Consumer`; `quill commit` and `quill unregister` against the held name exit 1 with ErrLocked.
- **Writer restart with ProtectedConsumers naming a running unprotected consumer (verified, `cmd/prot`).** Open fails with ErrLocked after about 221 ms. This is documented (scope.md ProtectedConsumers row), so it is not a finding.
- **Reader never takes the writer's lock (verified by grep).** `takeLock` and `TryLock` appear only in Open (log.go:362), consumer open and remove (consumer.go:82, 302) and Archive's non-blocking probe of the destination's lock
  (archive.go:116-121, released at once). No reader verb takes any lock. The one reader-triggered stall was the `consumers/` listing under `l.mu` (A3-27).
- **Reader racing retention (verified, `cmd/race`).** A writer with SegmentBytes 2048, MaxBytes 4096 and FlushInterval 1 ms plus two Follow readers for 20 s. Linux: 119,756 records written, reader 1 read 114,047 with 2 gaps, the slow reader
  48,513 with 992 gaps; zero unexpected errors, zero order or below-offset violations. macOS: 11,798 written, both readers 11,476, 1 gap, 0 unexpected errors. Retention under a reader surfaces as GapError.
- **Reader trusting `quill.tail` ahead of the data (verified).** A marker with seq 15 over 10 real records (offset correct) makes reads fail with ErrCorrupt ("segment 0 ended at 10, expected 15") and `quill check` UNHEALTHY; reads are not
  served silently. A marker whose `off` is larger than the file is ErrCorrupt (reader.go:358-361). `Head()`, `quill stat` (head 15, records 15) and consumer lag still trust the marker.
- **Live Reader with a cached damaged-then-repaired segment (verified, `cmd/heal`).** It keeps reporting the gap until reopened, while a fresh Reader sees the repair. This follows the stated sticky-damage rule and the user guide's "stop
  every process that holds the log".
- **Clock guard in isolation (verified in-package with a fake clock).** At most `ceil(sealed/4)` per pass is deleted by age; a protected consumer with no ConsumerTTL is fully immune to a +400-day step (21 to 21). The future-stamp prefix hold
  is documented (scope.md "A wrong clock"). A clock stepped back and corrected makes data written during the wrong period look old, which the same guard covers (inferred from retention.go:114-129, not run).
- **Offset-file replacement while a consumer holds the lock (verified by reading).** `Commit` overwrites by temp and rename from in-memory state, and every other writer of that file needs the same lock, so a holder cannot be raced by
  quill's own tools. A foreign process replacing the file is noticed only by the identity gap (A3-30).
- **Archive destination checks (verified by reading).** It refuses the source as destination and a destination whose `quill.lock` is held (ErrLocked), and takes only the `_archive` consumer lock on the source (archive.go:37-47,
  106-123). The probe-then-release window on the destination's lock (a writer opening the destination between archive.go:116 and the `quill.tail` write at 83) was read, not reproduced, and is not raised.
- **Reader view racing a seal-and-delete (code read only, not reproduced).** `Reader.view` returns a raw ENOENT (reader.go:354-357) if the age pass seals and deletes the segment the marker named between the Reader's `readTail` and its open.
  It needs a microsecond window once per Retention, the error is retryable per decision 79, and the 20 s race runs did not hit it. (Compare A1-59, raised separately for the Log's own reads.)
