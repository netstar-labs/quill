# Pass A5: silent-degradation audit

Date 2026-10-31. Pass A5 of the stage 2 audit, run with fresh eyes before a public release, against the tree at 988f1e5
(the parent of the first A3 fix). The specification is docs/design/scope.md, docs/design/decisions.md and docs/format.md;
the code was the thing under audit, so a passing test and a doc were both treated as claims, not evidence. The question of
the pass: for every guarantee, what does the library or the `quill` command do when that guarantee stops holding, and can an
operator tell that state from health? The earlier passes are recorded in [a1-round2-findings.md](a1-round2-findings.md)
and its siblings, [threat-model.md](threat-model.md) (A3) and [format-compatibility.md](format-compatibility.md) (A6);
what this pass held back for the owner is in [stage2-owner-decisions.md](stage2-owner-decisions.md), section "Pass A5".

Labels: **verified** means an auditor, skeptic or the recorder ran it or read the exact lines; **reported** names the
document or role that says it; **inferred** says from what. In the ledger, `[run]` is verified by execution, `[read]` is
verified by reading the exact lines, and `[inf]` is inferred. Anchors (file:line) are to the tree this record is committed
with (d451fac plus this commit) unless marked "audited tree"; the seven A5 fixes and the A3 and A6 fixes moved some lines.
Reproduction programs lived in throwaway copies of the repository made with `git archive`, in a scratch area outside the
tree, on macOS (Darwin 25.6, arm64) and, for some skeptic runs, on a Linux x86-64 box (Debian 13, ext4). They are named
below by what they do and are not committed. The recorder of this file did not re-run any reproduction; it read the
current tree to re-anchor the citations and to check each fix against the code that is there now.

## Top line

25 items after merging 20 duplicate ledger ids into them (45 raw ids). 13 low and 12 medium, as rated by the auditors;
no high or critical item. 23 are degradations (a signal is missing, latched, misplaced or silent) and 2 are bugs
(A5-21, A5-23). No item is a loss of acknowledged data by itself, with one exception: A5-16 (a second writer after
`quill.lock` is removed) lets two writers overwrite each other, and it was fixed as hardening.

| verdict | items | ids |
|---------|-------|-----|
| CONFIRMED | 19 | A5-1, A5-2, A5-3, A5-9, A5-10, A5-16, A5-18, A5-21, A5-22, A5-23, A5-29, A5-31, A5-32, A5-33, A5-36, A5-39, A5-40, A5-42, A5-44 |
| PLAUSIBLE | 0 | none |
| REFUTED | 3 | A5-5, A5-6, A5-34 |
| prior-deliberate (decided in round 1 or by a documented decision; no votes) | 2 | A5-7, A5-15 |
| prior-refuted (an earlier pass refuted it; no votes) | 1 | A5-17 |

| disposition | items | ids |
|-------------|-------|-----|
| fixed, fix-skeptic FIXED-VERIFIED | 7 | A5-2 (38d2644), A5-21 (dd460a7), A5-39 (1b68eee), A5-42 (66081fe), A5-16 (6acc23e), A5-40 (906d49c), A5-31 (d451fac) |
| resolved by an earlier commit, no A5 commit | 1 | A5-33 (1f896c7, pass A3, documentation) |
| held for the owner (every one changes observable behaviour, so semantic) | 11 | A5-1, A5-3, A5-9, A5-10, A5-18, A5-22, A5-23, A5-29, A5-32, A5-36, A5-44 |
| refuted | 3 | A5-5, A5-6, A5-34 |
| deliberately not changed (decided earlier) | 3 | A5-7, A5-15, A5-17 |

- Skeptic votes: 47 item votes on the 22 items that were voted (two each on 19 items, three each on A5-5, A5-6 and
  A5-34 after a split). 39 CONFIRMED, 2 PLAUSIBLE (A5-2 and A5-29, one voter each) and 6 REFUTED (two on each of the
  three refuted items). The three prior-* items were not voted; the A1 record is their ruling.
- Reproduced: 34 of the 47 votes carried a reproduction by execution. Every one of the 22 voted items was reproduced
  by at least one voter. Each item below names the program and the output. Two reproductions are weaker than the
  rest and are marked: A5-9 was reproduced only by injecting a directory-fsync error into a patched copy, because the CLI
  has no fault hook; A5-22 and A5-1 used injected faults on the in-memory file system, with a second run on real files for A5-1.
- Fix commits: 7, all ancestors of this commit (verified from `git log`). Fix-skeptic verdicts: 7 of 7 FIXED-VERIFIED,
  through 43 lens verdicts (6 or 7 per fix: test-fails-without-fix, root-cause, behaviour-preserved, no-new-gap,
  reachable, ladder, and independent-route for A5-21), every one "pass". Three fixes were rejected at least once
  before the recorded attempt (A5-2, A5-39, A5-42; reported by their fix summaries). The Linux box was used for the
  fix verification of A5-2, A5-16 and A5-40 (reported by the skeptics).
- Ledger: 123 rows, one per degraded state: at audit 49 OK, 30 LOG-ONLY, 44 NONE. After the A5 commits 11 rows carry an entry in the Now column: 4 moved to OK, 1 to OK with a stated window (A5-16), 1 to OK on the CLI and LOG-ONLY in the library (A5-2), 2 to LOG-ONLY (A5-31, A5-42), 1 (the nil-Logger row) to LOG-ONLY for the two loss records, and 2 stay NONE with a note (A5-5, A5-33).
- Not green: nothing failed. What is not covered is under Coverage.

## Coverage

Five auditors, one per group of guarantees, all returned: group 1 (G1 to G11, the read, durability and failure path),
group 2 (G12 to G22, rotation, batching, limits, locks), group 3 (G23 to G33, consumers and retention), group 4 (G34 to
G44, TTL, protection, archive, Close, Config, file types) and group 5 (G45 to G54, time, `quill check`, the CLI, policy,
backup). No auditor failed to return. G1 to G54 were the guarantees enumerated from scope.md, decisions.md and the code;
each auditor listed, per guarantee, the degraded states it tried and what signal each left. Their lists were merged into the
ledger by the recorder.

Checked and found clean (verified by the auditor named, by reading unless marked run):

- Every read-path violation of order or at-most-once refuses: `GapError` in `scanSource` and `ErrCorrupt` on a bad frame
  (group 1). A record is never visible before its data fsync; the head advances only after `Sync` (group 1).
- Every Open refusal named in G8 (marker above the data, marker segment missing, overlap, damaged newest segment below the
  marker) returns `ErrCorrupt` and leaves the files untouched; the marker-ahead case was compared byte for byte (group 1, run).
- A data fsync error, a directory fsync error at rotation, before an unlink and at Close, and a failed truncate after a
  failed write, all fail the log: `ErrFailed` from Append, Flush, Sync, Close and Err() (groups 1, 2; run for the data fsync).
- `MaxBufferBytes` refuses an append past the bound with the flush error and assigns no sequence (group 1, run: refusal at
  65414 of 65536 bytes); an empty buffer accepts any single record by design (group 2).
- Allocation bounds: oversize frame, segment, state file, payload and batch are refused before they are allocated (group 2).
- A sealed segment that is damaged, missing in the middle, over 1.5 GiB or unopenable becomes a hole at Open, is named by
  `Stat().Holes` and `quill check` (exit 3), and reaches readers as a `GapError` (groups 1, 2; run for damage and deletion).
- I/O errors while reading are returned as themselves and are retryable; a segment deleted between view and open is a
  `GapError`, not a path error (group 2, read).
- Second writer: `ErrLocked` after 200 ms; an unsupported or failing `flock` fails Open instead of being a no-op (group 2,
  read). Second open of one consumer name: `ErrLocked` (group 2).
- Consumer protocol: `ErrOffsetRange`, `ErrWrongLog`, `ErrAhead` and `ErrInvalidName` are returned; a failed Commit restores
  the in-memory offset; a new consumer file is directory-fsynced (group 3, read).
- Retention never deletes the active segment and never reuses a segment id; an unlink failure is logged and latched; a marker
  publish or directory fsync failure during a delete is returned or fails the log (groups 3, 4, read).
- Config is validated before any file is touched (`ErrConfig`); a marker newer than the build is `ErrFutureVersion` from
  every path; a symlinked or non-regular segment is refused (group 4, read).
- `deadcode -test ./...` on a copy returned nothing (groups 1, 2, 4, run; group 5 ran the pinned version and notes the result
  says nothing about exported fields and methods that tests read). Written-never-read fields: `Dropped` and `StaleConsumers`
  are read only by `Log.Stat` callers and never by the CLI; `fsyncs` is read only by tests; `loadErr`, `bgErr`, `diskFull`,
  `rotErr`, `flushErr` and `markerAt` are all read (grep, groups 1, 2, 4, 5).
- The flusher goroutine and prune ticker have no panic or early-exit path other than Close, and no heartbeat or
  last-success record exists for them (read; the missing record is part of A5-10).
- No capability was found that initialises as a no-op other than `NoMaintenance` (documented opt-out) and a nil
  `Config.Logger` (silent by default; see "The rule").

Disagreements between auditors, settled by execution: group 3 read that `Follow` on a failed log returns `ErrFailed`;
the A5-3 skeptics ran it and `Log.Follow`, `Log.ReadWait` and `Consumer.ReadWait` all block until the context ends, so the
ledger follows the run. Group 2 suspected that `Stat().MarkerAge` reads zero after Open on an idle log; the run refuted it
for a non-empty log, and a skeptic then found it true for an empty one (A5-40, fixed).

Not covered (stated by the auditors, unchanged):

- The Linux box and the Apple container were used only by two of the five auditors for one test (a `flock` check across a
  bind mount, group 2) and by skeptics for some reproductions. No auditor ran a real full disk, a real fsync EIO or a real
  statfs failure; every fault was injected on the in-memory `fsx.Sim`, on a statfs seam, by `chmod` or `chflags`, or in a
  patched copy.
- No multi-process timing scenario (a Reader racing a live writer's flush), no SIGKILL scenario, no cross-account reader
  (A5-33's reader-side EACCES is inferred), no `-race` run by the auditors (the fix-skeptics ran it on the fixed tree),
  and `hack/gate.sh` was not run by any auditor.
- `Reader.Follow`, `Records` and `Archive` were examined only as far as they share `scanSource`; `quill tail -follow` across a
  replaced directory is inferred.
- The ledger below marks the state at the audited tree, with a "Now" column for the seven fixes. The three A3 fixes that
  touch these states (495e457, 2bf7412, 120c8ac) landed between the audited tree and this record and are noted at
  A5-5, A5-18 and A5-23; the recorder read their diffs, did not re-run the reproductions against them.

## The ledger

One row per degraded state. **Mark** is the audit's mark: OK means a signal exists and an operator could act on it;
LOG-ONLY means the state is discoverable after the fact (a log line, a latched error, a pull-only check) but not alertable;
NONE means it is indistinguishable from health. Where a state has both a loud and a quiet surface the row is marked for
the quieter one and the signal column says so. **Now** is the mark after the A5 commits, left as "-" where nothing changed
for that state. **Item** points to the record below; "excl." points to "Deliberate exclusions". Rows from different
auditors that covered one state were merged.

| G | Degraded state | Signal | Mark | Now | Item |
|---|----------------|--------|------|-----|------|
| G1 | a read-path state that breaks order or at-most-once while still answering | `GapError`, `ErrCorrupt`; none found that answers silently [read] | OK | - | - |
| G1, G4 | Reader or Reader-backed Consumer over a directory replaced by a different log keeps serving the new log from the old cursor | none on plain Read (`ID()` unchanged, Check HEALTHY); incidental `ErrCorrupt` by cached segment size [run]; since 495e457 `Commit` and `ReadWait` return `ErrWrongLog` [read] | NONE | NONE for Read | A5-5 |
| G1 | bare Reader whose cursor is ahead of a rolled-back head waits forever | none; a Consumer open gets `ErrAhead` [read]. The auditor marked it LOG-ONLY; no signal exists for a bare Reader, so NONE by the definition above | NONE | - | excl. |
| G2 | sealed segment deleted from the middle | `Stat().Holes`, `GapError`, `quill check` exit 3 [run]; `Err()` stays nil | OK | - | A5-10 |
| G2 | sealed segment damaged | `Err()` `SegmentError`, `Stat().Holes`, check exit 3, `GapError` [run] | OK | - | - |
| G2 | sealed segment damaged mid-file seen through `Reader.Stat`, `quill stat`, `quill ls` | no hole, records overstated, exit 0; `quill check` exit 3, `quill cat` exit 1 [run] | NONE | - | A5-7 |
| G2 | oldest segments deleted by hand | `First` moves forward, no hole, `Dropped` 0: looks like retention [run] | NONE | - | excl. |
| G2 | marker names a missing segment, marker above the last valid frame, overlapping segments, segment below the active one deleted | `Open` returns `ErrCorrupt`, files untouched; a Reader's Check says UNHEALTHY [run] | OK | - | - |
| G2, G53 | head at 2^64-1 | `Open` refused, `ErrSeqExhausted` on append [read] | OK | - | - |
| G2, G18 | sealed segment that cannot be opened at Open (EACCES, EMFILE) is a hole for the writer's life although the file is healthy | `Err()` (raw open error), `Stat().Holes`, check; not self-healing until restart [read] | OK | - | excl. |
| G16 | sealed-segment damage that appears after Open, under a live writer | writer `Stat` and `Err()` unaware; reads return `SegmentError`; check finds it [read] | LOG-ONLY | - | excl. |
| G16 | second and later bad sealed segments | `Stat().Holes` and check list all; `Err()` names the first only [read] | OK | - | excl. |
| G3 | background flush failing (ENOSPC, EIO): appends are acknowledged and held in memory, invisible to readers | `Err()` only; no buffered count; Logger silent [run: 580 appends acknowledged, durable head 1] | LOG-ONLY | - | A5-1 |
| G3 | unflushed record visible to a reader | not reachable: the head advances only after the fsync (`flush.go`) [read] | OK | - | - |
| G3, G11 | `ReadWait`, `Consumer.ReadWait`, `Follow` on a failed log block until the context ends | `Err()`, `Append`, `Close` return the failure elsewhere [run] | LOG-ONLY | - | A5-3 |
| G4 | other-process Reader reads past the marker | refused or clamped by construction [read] | OK | - | - |
| G4 | writer dead, wedged or idle: a Reader sees a frozen head | `Stat().MarkerAge`, which grows the same for an idle healthy writer; by decision 47 [run] | LOG-ONLY | - | A5-6 |
| G4 | `Log.Stat().MarkerAge` reads 0 on an empty log when no marker has been published since Open | none: "never published" shows as "just published" [run] | NONE | OK | A5-40 |
| G4 | `Reader.Stat().MarkerAge` stays 0 if the `Lstat` fails | none; reachable only by a race with the marker being replaced [read] | NONE | - | excl. |
| G5, G12 | marker publish failing while data writes succeed: data durable, readers frozen at the old marker, retention and caps suspended | `Err()` (latched), `Flush` return; readers see `MarkerAge` grow [run: 50 of 50 appends nil, head 1 for the reader] | LOG-ONLY | - | A5-1, A5-10 |
| G5, G12 | next-segment create failing: the active segment grows past `SegmentBytes`, `MaxBytes` not applied meanwhile | `Err()`; append refused at the ceiling [run: 1 segment, 6720 bytes against `MaxBytes` 2048] | LOG-ONLY | - | A5-10 |
| G5, G11 | directory fsync error at rotation, before an unlink, at Close | `ErrFailed` from every call, `Err()` [read] | OK | - | - |
| G5, G39, G47 | `quill prune` and `quill roll` discard `Close`'s error (the final directory fsync, lock release) | none, exit 0 [read; reproduced with an injected fsync error] | NONE | - | A5-9 |
| G6 | stale marker after an OS crash or rollback hides durable records until reopen or `quill recover` | none: `quill stat` shows the older head, Check HEALTHY, `MarkerAge` is the marker's own (rolled-back) mtime [run] | NONE | - | A5-6 |
| G7 | writable Open or `quill recover` cuts the newest segment past the marker, including a durable acknowledged frame under a stale marker | none at audit: no log, no field, `recovered: head N` only [run: head 19 against 20 acknowledged, segment 641 to 599 bytes] | NONE | OK (CLI), LOG-ONLY (library) | A5-2 |
| G7 | torn tail below the durable offset | `Open` `ErrCorrupt`, files untouched [read] | OK | - | - |
| G8, G21 | two writers after `quill.lock` is removed under a live one | none: both acknowledge, records overwritten, Check HEALTHY [run] | NONE | OK (window) | A5-16 |
| G9 | `Sync` or `SyncAppends` returns nil although the marker publish failed or the log failed on the directory fsync (decision 78) | `Err()`, `Flush` [run] | LOG-ONLY | - | A5-1 |
| G9 | `Sync` error means unknown not lost; `ErrBadSeq`; waiters at Close get `ErrClosed` unless durable; `SyncAppends` error wraps the flush error | returned errors [read] | OK | - | - |
| G10 | write error truncates back and keeps records buffered, retried every `FlushInterval`, recovers by itself | recovery works [run: head 1 to 581 after the fault cleared]; `Err()` never clears | LOG-ONLY | - | A5-1, A5-10 |
| G10 | inline flush in append and the flusher discard the flush error | `Err()` (documented, decision 60) [read] | LOG-ONLY | - | A5-1 |
| G10, G13 | `MaxBufferBytes` reached while flushes fail | append refused with the wrapped flush error [run] | OK | - | - |
| G10, G13 | window between an acknowledged append and its durability while flushes fail | no counter, timestamp or flag in `Stats` [read] | NONE | - | A5-1 |
| G11 | `Err()` hides a live background failure behind a load error, and keeps a load error after its hole is gone | `Err()` only [run] | NONE | - | A5-10 |
| G11 | no flush, publish, rotate or fsync failure is written to `Config.Logger`, though the field documents "warnings and failures" | none [read: `logf` calls; run: empty Logger buffer in three fault runs] | NONE | - | A5-1 |
| G11 | data fsync error | sticky `ErrFailed` on four calls and `Err()` [run] | OK | - | - |
| G12 | rotation failing: marker still published, appends continue to the ceiling | append error at the ceiling [run: 29 accepted, then refused at 5608 bytes]; `Err()` while unmasked | LOG-ONLY | - | A5-10 |
| G12 | rotation failing with nobody appending | flusher retries every `FlushInterval`; `Err()` set [run: recovered within 600 ms] | OK | - | A5-10 |
| G12, G27 | age rotation of a quiet segment failing: old data outlives `Retention` | `Err()` only, zero log lines [run] | LOG-ONLY | - | A5-10 |
| G12, G27, G31 | prune pass aborted at the age-rotation flush while flushes fail: sweep and caps skipped, which is the only self-recovery on a full disk | `Err()` shows the flush error, nothing shows the pass stopped [run] | LOG-ONLY | OK | A5-21 |
| G13 | `MaxBufferBytes` gives up while looking drained | not found: `flushErr` is cleared only after a successful flush [read] | OK | - | - |
| G14, G15 | `AppendBatch` refusals; frame, segment, state-file and payload bounds | typed errors before any allocation or sequence [read] | OK | - | - |
| G15 | sealed segment over 1.5 GiB | refused unread, becomes a hole; `Err()`, `Stat().Holes`, check [read] | OK | - | - |
| G15, G29 | consumer state file over 1 MiB, or any unparseable file in `consumers/`, halts the age sweep | writer: none; check exit 3 and `quill consumers` by pull [run] | NONE | - | A5-17 |
| G17 | retention or a hole below a requested offset | `GapError` to the reading consumer [read] | OK | - | - |
| G17, G23, G34 | consumer fell below `First` and is not reading | `Lag` counts deleted records; `quill consumers` NOTE empty; check HEALTHY; `protected` still true [run] | NONE | - | A5-18 |
| G17, G30 | `Dropped` counts every segment deleted past an abandoned consumer file, names no consumer, resets at writer restart | `Log.Stat().Dropped` only; the CLI never prints it [run: 8 then 0 after restart] | LOG-ONLY | - | A5-18 |
| G17, G23 | `Consumer.Lag` returns 0 for a consumer ahead of the head | `quill consumers` NOTE "ahead of the head", Check warning, `ErrAhead` at open; the library `Lag()` is a reassuring 0 [read] | OK | - | excl. |
| G19 | segment deleted between view and open | `GapError` [read] | OK | - | - |
| G19, G37 | source with a gap: `quill archive` prints one stderr line and exits 0; later runs are silent while the destination stays holed | one-shot stderr line; `quill check -dir dst` exit 3 [run] | LOG-ONLY | - | A5-29 |
| G19, G35, G37 | `_archive` consumer removed or recreated, then records lost: the destination gets a hole with no message | none; check on the destination exit 3 [run] | NONE | - | A5-29 |
| G19, G37 | archiver stalled (cron died) pins every segment from its offset | `quill consumers` lag, Check UNBOUNDED line; `ConsumerTTL` is off by default [read] | LOG-ONLY | - | excl. |
| G37 | rerun leaves a same-size damaged destination segment in place | exit 0; check on the destination exit 3 [run] | LOG-ONLY | - | A5-32 |
| G37 | corrupt source segment or copy error | error return, exit 1, progress kept to the last good segment [read] | OK | - | - |
| G20, G51 | policy write failing at Open: the previous run's quill.policy stays on disk and Check, ls and offline prune read it as current | `Err()` only, which a load error can mask [run] | NONE | OK | A5-39 |
| G20 | stale `.tmp-*` files: removed only at writer Open; leaked ones not counted or reported | none [read] | NONE | - | excl. |
| G20, G46 | policy file unparseable or missing | Check warning plus UNBOUNDED, `ls` NOTE [run] | OK | - | - |
| G21 | second writable open; `flock` unsupported or erroring | `ErrLocked`; Open fails with the error [read] | OK | - | - |
| G21 | bind mount across a VM boundary: the lock is not coordinated, two writers possible | none; documented unsupported (scope.md "Filesystems") [run in an Apple container] | NONE | - | excl. |
| G22 | consumer lock file deleted by a third party | none; same class as the writer lock [inf] | NONE | - | excl. |
| G22 | writer reads a protected consumer file lock-free | by design, so a running consumer does not block a restart [read] | OK | - | - |
| periodic | no last-success time or per-pass result for the prune pass, the age rotation or the flusher | none in `Stats` or `Err()` [read] | NONE | - | A5-10 |
| periodic | slow disk or bulk prune holds the log lock past `FlushInterval` | none; 3.05 s block on APFS for 400 forced segments, 0.24 to 0.44 s at the default guard [run] | NONE | - | A5-34 |
| periodic | a blocking Logger handler stalls the flusher | documented for the Logger [read]; not run | NONE | - | excl. |
| periodic | `NoMaintenance` set by mistake by an embedder | nothing in `Stat` says maintenance is off; explicit opt-out [read] | NONE | - | excl. |
| G23 | `Commit` out of range; consumer file from another log; consumer ahead at open; bad name | `ErrOffsetRange`, `ErrWrongLog`, `ErrAhead`, `ErrInvalidName` [read] | OK | - | - |
| G23 | restored log whose head later grows past a stale consumer offset | `ErrAhead` is checked only at open; records between are skipped silently [read; inferred from decision 36's own rationale] | NONE | - | excl. |
| G24 | `Commit` write failure; new consumer file creation | error returned and offset restored; creation fsyncs `consumers/` [read] | OK | - | - |
| G24 | OS crash rolls a commit back | documented at-least-once [read] | OK | - | - |
| G24, G42 | Perm tightened on reopen leaves existing files at the old mode, Check healthy | none [run, out of this pass's guarantees] | LOG-ONLY | - | excl. |
| G25 | iterators and consumer reads commit; error mid-iteration | no Commit call exists; the error is yielded once [read] | OK | - | - |
| G25 | `Scan` payload aliases a pooled buffer | documented contract [read] | OK | - | - |
| G26 | segment id reuse; retention touching the active segment | not found [read] | OK | - | - |
| G26 | unlink fails (EPERM, immutable file): retention stalls; `Prune` returns nil | Warn "cannot delete segment" and latched `Err()`; the CLI sets no Logger at audit and exits 0 [run] | LOG-ONLY | - | A5-36 |
| G26 | segment already removed by hand | Warn "segment was already gone" [read] | LOG-ONLY | - | excl. |
| G27, G28 | quill prune did less than asked: unlink failure, corrupt consumer file, clock guard | exit 0, counts line only, nothing on stderr [run: `6 -> 6`, `10 -> 7`] | NONE | - | A5-36 |
| G27 | `Retention` negative and no caps | Check prints UNBOUNDED as text, exit 0 [read] | LOG-ONLY | - | excl. |
| G27, G28 | active segment stamped in the future; quiet log never seals when `SegmentMaxAge` or `Retention` is negative | none until it seals / by configuration [inf] | NONE | - | excl. |
| G28 | age sweep limited by the clock guard | Warn on the Logger only [run] | LOG-ONLY | - | A5-36 |
| G28, G45 | a segment stamped more than a day ahead holds back all newer ones, even with `-force` | Warn on the Logger; Check warns, HEALTHY, exit 0 [run] | LOG-ONLY | - | A5-36 |
| G28 | check prints one Problem per record for a clock anomaly | 17 lines for 17 records; large logs not measured [run] | LOG-ONLY | - | excl. |
| G29, G34 | stale protected consumer stops protecting (`ConsumerTTL`) | `Stats.StaleConsumers` (frozen between passes, empty before the first and with `NoMaintenance`, 0 after restart) and a Warn per pass; the CLI shows `protected` true [run] | LOG-ONLY | - | A5-18 |
| G29 | `unregister` of a protected consumer, or a listed consumer made unprotected while the writer runs | none until the next writer Open re-applies `ProtectedConsumers`; explicit operator action [inf] | NONE | - | excl. |
| G30 | a cap deletes under a protected consumer's unread data | `Dropped` and an Info line without the consumer's name; offline prune never prints `Dropped` [run] | LOG-ONLY | - | A5-18 |
| G30 | consumer file unreadable at pass time is not counted in `Dropped` | none [inf] | NONE | - | excl. |
| G31 | `MaxBufferBytes` smaller than `FlushBytes`: the published disk bound is exceeded | none, `Open` accepts the Config [run: peak 4,200,444 against 3,949,577 published] | NONE | - | A5-23 |
| G31 | `MaxBytes` unmet because unlink fails | Warn, `Err()`, `Stat().Bytes` against the policy [run] | OK | - | A5-36 |
| G31 | `MaxBytes` below twice `SegmentBytes` | `Open` refuses [read] | OK | - | - |
| G32 | `MinFreeBytes` with a failing statfs: the floor is not enforced, or a frozen `diskFull` keeps refusing | one Warn per pass if a Logger is set; `Err()` nil; no `Stats` field [run] | LOG-ONLY | - | A5-22 |
| G32 | `diskFull` set: every append fails with ENOSPC while `Err()` is nil; recovery is not logged | loud at the appender; nothing in `Err()` or `Stat` [run] | LOG-ONLY | - | A5-22 |
| G32 | floor crossed between passes; unlink frees nothing while a reader or archiver holds the segment | none; evaluated at seals and passes only [read; the second is inferred] | NONE | - | excl. |
| G33 | `MaxRecords` with the active segment alone above N; worst-case disk of `MaxRecords` times the record size | `Stat().Records` is pollable; documented non-guarantee [read] | OK | - | excl. |
| G34 | `ConsumerTTL` probe errors (mtime `Lstat`, or a non-gap read error): the consumer silently stays live | none at audit [run: `StaleConsumers` empty, `Err()` nil, 5 segments pinned] | NONE | LOG-ONLY | A5-31 |
| G34 | consumer file mtime reset by `cp` or `rsync` | stale at once or fresh for one more TTL; documented in scope.md [read] | LOG-ONLY | - | excl. |
| G34 | `StaleConsumers` before the first pass and under `NoMaintenance` reads empty, meaning "not evaluated" | none [read] | NONE | - | excl. |
| G35 | writer cannot create a `ProtectedConsumers` entry at Open | `Open` returns the error [read] | OK | - | - |
| G36 | `Log.Stat().Bytes` after an out-of-band delete or truncation of a segment | none; sums sizes from memory [inf] | NONE | - | excl. |
| G36 | `MaxBytes` and `MinFreeBytes` both 0 (the defaults) | Check UNBOUNDED, `ls` marks it, `CheckResult.Unbounded` [run] | OK | - | - |
| G36, G38, G54 | `Err()` after a damaged segment was loaded: later background errors masked, the load error stays after recovery | `Err()` returns the load error forever [run] | NONE | - | A5-10 |
| G36 | `Reader.Stat` zero `Dropped` and nil `StaleConsumers` read as "none" though the Reader cannot know | none; the CLI never prints them [run] | LOG-ONLY | - | A5-15 |
| G38 | flusher goroutine dying | cannot, except by panic [read] | OK | - | - |
| G39, G40 | use after Close; invalid Config | `ErrClosed`; `ErrConfig` before any file is touched [read] | OK | - | - |
| G40 | `ConsumerTTL` without protected consumers; `Retention` 0 meaning 24 h | none; documented defaults [read] | NONE | - | excl. |
| G41 | symlinked or non-regular sealed segment | open refused, hole; check UNHEALTHY exit 3; `quill stat` unchanged [run] | OK | - | A5-7 |
| G42 | segments and `consumers/` tighter than `Perm` under a stricter umask | none; check HEALTHY [run] | NONE | NONE, documented | A5-33 |
| G43 | interrupted creation finished by Open; foreign directory | silent by design (decision 62); `ErrNoLog`, `ErrCorrupt` [read] | OK | - | - |
| G44 | marker newer than the build | `ErrFutureVersion` from every path [read] | OK | - | - |
| G44, G29 | consumer file from another log blocks the age sweep silently | `quill consumers` Err column, check fatal [run] | NONE | - | A5-17 |
| G45 | writer clock stepped back | writer none; Check warns once per record [run] | LOG-ONLY | - | excl. |
| G45 | `OffsetForTime` returns a valid but wrong offset after a back-step; `-since` earlier than `First` clamps | no error; documented (scope.md, decision 59) [run] | NONE | - | excl. |
| G46 | consumer ahead of the head | Check warning, HEALTHY, `quill consumers` NOTE [run] | OK | - | - |
| G46 | marker names data the copy lacks (naive restore) | check UNHEALTHY exit 3; stat and read exit 1 [run] | OK | - | - |
| G46 | policy stale after a failed write | Check reported the old caps as current [run] | NONE | OK | A5-39 |
| G46 | live writer's unflushed tail | not scanned by design [read] | OK | - | - |
| G47 | `quill ls` omits a log whose reader cannot open | exit 0, row absent [run] | NONE | - | A5-44 |
| G47 | offline verbs beside a live writer | `ErrLocked`, exit 1 [read] | OK | - | - |
| G48, G49 | stdout unwritable; append Close failure; JSON bidi controls pass raw | exit 1; exit 1 and the sequence printed; the guarantee covers C0, C1 only [read, run for bidi] | OK | - | excl. |
| G50 | other platform build; macOS `fsync` falling back to plain `fsync` on ENOTSUP | build fails on purpose; no probe, on filesystems decision 51 already declares unsupported [run, read] | OK | - | excl. |
| G52 | restore with a consumer ahead; marker taken after the segments | `ErrAhead`; Open refuses, Reader reports corrupt [read, run] | OK | - | - |
| G52 | copy order itself | not validated by code; the failure is loud [inf] | NONE | - | excl. |
| G54 | damaged sealed segment deleted by caps or the age sweep: hole, `Err()` latch and Check evidence all go | at audit an Info "deleted segment" line without the word damaged [run] | NONE | LOG-ONLY | A5-42 |
| G54 | damaged segment counts toward `Bytes` and caps | `Stat().Bytes` includes it [read] | OK | - | - |
| all | `Config.Logger` is nil by default, so every LOG-ONLY row above is NONE for a writer started without one, which included every CLI write verb at audit | silent [read] | NONE | LOG-ONLY for the two loss records on the CLI | A5-2, A5-42 |

Counts: 123 rows, 49 OK, 30 LOG-ONLY, 44 NONE at audit (recorder, by counting the Mark column with a script); the Now column is summarised in the top line.

## Records of the 25 items

Held items first, then fixed, then the rest. Each item: Claim, Grounds, Warrant, Backing, Rebuttal, Verdict with the
votes and the reproduction, and Disposition. Anchors are to the current tree; the audited-tree line is given where a
later commit moved it and the citation matters. "Votes" lists each voter in order: C confirmed, P plausible, R refuted;
"reproduced" says which voters ran something.

### Held for the owner

#### A5-10: `Err()` is a single latched slot, masked and frozen (medium, held)

**Claim.** `Err()`, the signal the user guide names for rotation and flush failure, cannot tell current from past. A
sealed-segment load error permanently shadows every background error, and a background error is never cleared when the
operation recovers. Merged ids: A5-4, A5-20, A5-27, A5-41.

**Grounds.** `Log.Err` (log.go:618-628) returns the failed state, then `loadErr`, then `bgErr`. `loadErr` is set once at
load (load.go:362) and nothing clears it. `bgErr` is assigned at flush.go:56, 107, 144, 153, 224 and 319 and
retention.go:132 and 296, and is never reset, with one exception added after the audit: A5-39's fix clears it when the
policy write it latched succeeds again (log.go:668). Reproductions (a program on the public API with a directory placed at
the next segment name): (1) a fresh log: 29 appends past `SegmentBytes` accepted, then refused with EEXIST at the ceiling;
`Err()` showed EEXIST; after the blocker was removed the next append and rotation succeeded (1 to 2 segments) and `Err()`
still returned the same EEXIST. (2) A bit flipped in sealed segment 0: Open succeeded, `Err()` was the `SegmentError`, holes
`[{1 28}]`; with the same blocker 30 appends were accepted and then refused at the ceiling while `Err()` still named only
the damage. (3) The hole's segment deleted by retention: `Stat().Holes` empty, `Err()` still named the deleted segment.
(4) Age rotation failing: `Err()` set, zero log lines.

**Warrant.** docs/userguide.md says a failure to publish the marker or create the next segment is "reported by `Err()`" and
tells the operator to watch it before the ceiling. If `Err()` is masked by an old load error, or already non-nil from a
recovered fault, watching it says nothing about a failure that is happening now. For prune unlinks, age rotation, marker
publish with no appends and the pass itself, nothing else is returned to any caller.

**Backing.** G12 (scope.md guarantee on failures after durability, decision 60(d), decision 78); G16 caveat (`Err()` shows
the first bad segment only); scope.md and the `Err` doc call it the "background error" and the "sticky load error".

**Rebuttal.** The wording is literal ("last background error"), so nothing documented promises that `Err()` clears or
reports both. An appender receives the rotation or flush error itself at the ceiling or at `MaxBufferBytes`, and
`Stat().Holes` and `quill check` show the hole independently. The loss is for a monitor that polls `Err()`, and for the
window before the ceiling (about 73 MiB of appends past `SegmentBytes` at default settings, per the auditor).

**Verdict.** CONFIRMED (C, C). Reproduced by one voter (program above, macOS); the other read the assignments and found no
nil store. Whether a different `Err()` contract is wanted is the owner's call.

**Disposition.** Held for the owner (semantic). Recommended: keep the sticky load error but stop it shadowing (return the
join of the failed or load error and `bgErr`, or add `Stats.BackgroundErr` and `BackgroundErrAt`); clear `bgErr`, or stamp
a recovery time, when the operation that set it succeeds (a flush clears the flush error, a rotation the rotation error, a
clean prune pass the prune error); drop `loadErr` once the segment it names has been deleted and its hole pruned. The
policy case already works this way (A5-39). Decide this one together with A5-1 and A5-22, which need the same field.

#### A5-1: background failures reach only a latched `Err()`, never the Logger (medium, held)

**Claim.** Every retryable background failure (data write, marker publish, next-segment create, prune sweep) is reported
through `bgErr` and `Err()` only. It carries no time or count, nothing is logged, and no `Stats` field says how much
acknowledged data is held in memory, so an operator cannot alert on it, see it resolve, or see how much is unflushed.

**Grounds.** `logf` is called only from retention.go and (since the A5 fixes) load.go, log.go and flush.go for the new
records; no flush, publish, rotate or fsync failure is written to `Config.Logger`, though the field documents "warnings and
failures" (log.go). `Stats` (stat.go) has no last-flush time, buffered count or failing flag. Reproductions: (1) fault
injection on the in-memory file system, write to `.log` failing with ENOSPC: 580 appends returned nil with durable head 1,
then a refusal at `MaxBufferBytes`; `Err()` ENOSPC; the Logger buffer empty; after the fault cleared the head was 581 and
`Err()` still returned ENOSPC. (2) Temp-file create failing with `SyncAppends` on: 50 of 50 appends and `Sync` returned nil,
`Flush` returned ENOSPC, `Err()` ENOSPC, Logger empty. (3) Next-segment create failing with `MaxBytes` 2048: 1 segment,
6720 bytes, `Err()` ENOSPC, Logger empty. (4) After a transient EIO, a clean `Flush` and a clean `Prune`, `Err()` still EIO.
A voter repeated (4) on the real file system with `chmod 0500` and a slog buffer: `Flush` and `Err()` returned EACCES, the
Logger stayed empty, and after `chmod 0700` and a clean flush (head 22) `Err()` still returned EACCES, also 500 ms later.

**Warrant.** An alert needs a transition event or a pollable current-state signal, and a way to clear. `Err()` is the only
channel for the failures `Sync` does not surface (publish and rotation, where `Sync` returns nil by decision 78; prune
failures; non-`Sync` appends inside the `MaxBufferBytes` window). It is poll-only, not logged, and latched.

**Backing.** docs/userguide.md "Watch `Err()` and fix the cause before the ceiling"; decisions 60(d) and 78; scope.md
names `Err()` as the background error.

**Rebuttal.** `Err()` is documented as the last background error, not the current one. `Flush` and `Sync` with `ErrFailed`
do surface write and fsync failures to callers who use them, and `MaxBufferBytes` and the segment ceiling turn a persistent
fault into a loud refusal, so for a `Sync`-using producer the write case is not silent. The state matters most for publish,
rotation and age-rotation failures. Two parts of the original claim were wrong and were corrected by the voters: a failed
segment delete is logged at Warn (retention.go:297), and `Stat().MarkerAge` is a weak pollable proxy for a stuck publish.

**Verdict.** CONFIRMED (C, C). Both voters reproduced.

**Disposition.** Held for the owner (semantic). Recommended minimum, no API change: log once on each transition into and
out of a failing state (flush write, publish, rotate, prune) through the existing Logger at Warn and Info. Fuller, additive:
`Stats.LastFlushOK`, `Stats.BufferedRecords` and `BufferedBytes`, `Stats.FlushFailing`. A5-10's `Err()` change is the
precondition; do both in one pass.

#### A5-36: `quill prune` exits 0 when it did less than asked (medium, held)

**Claim.** `quill prune` prints a counts line and exits 0 with nothing on stderr when the pass did less than asked: an
unlink failure, a corrupt consumer file, or the clock guard. Merged ids: A5-11, A5-19, A5-28.

**Grounds.** At audit the CLI built its Config with no Logger and `cmdPrune` never read `Err()`; `Prune` returns nil when an
unlink fails because `deleteLocked` stores the error in `bgErr` and returns only the directory-fsync error (retention.go:296).
The clock-guard warning is only a `logf` (retention.go:176). Reproductions with the built CLI on macOS: (S3) `chflags uchg`
on the oldest segment, `quill prune -retention 1s -force`: `segments 7 -> 7`, exit 0, empty stderr; (S1) 9 or 10 expired
sealed segments without `-force`: `segments 10 -> 7` or `11 -> 8`, exit 0, empty stderr; (S2) a corrupt consumer file
(check says UNHEALTHY, exit 3): `segments 7 -> 7`, exit 0, empty stderr (reproduced by one voter, read by the other). The
library gave `Prune(true)` nil with segments 6 to 6 and `Err()` the injected EACCES.

**Warrant.** The operator asked for a retention pass and got success; the reason it did nothing sits in a logger the CLI
never configured and an error it never reads. An unlink failure is indistinguishable from nothing to delete except by
comparing the numbers by hand.

**Backing.** scope.md and docs/userguide.md exit codes (0 ok, 1 error); decision 44 says the guard "logs a warning ... to give
an operator time to notice", which the CLI could not carry.

**Rebuttal.** The before and after counts are printed, so an attentive human sees `6 -> 6`; the clock guard is documented and
`-force` exists; the numbers are not wrong, only unexplained. A script that checks only the exit code sees success.

**Verdict.** CONFIRMED (C, C). Both reproduced two of the three causes with the built binary.

**Disposition.** Held for the owner (semantic: a stderr line and an exit code). State of the tree now: A5-2 and A5-42 gave
the offline write verbs a stderr Logger (cmd/quill/write.go, `lossOnly`), but it passes only two messages, "cut the newest
segment" and "deleted damaged segment", so none of the three prune causes is shown (verified, write.go:74). Recommended:
widen that filter to the three prune messages ("cannot delete segment", "age sweep limited by the clock guard", "age sweep
stopped at a segment stamped in the future"), add a broken-consumer Warn (see A5-17), make `cmdPrune` exit 1 when `Err()` is
non-nil after the pass, and have the library's `Prune` return the unlink error instead of nil.

#### A5-18: a consumer that fell out of retention reads healthy on every offline surface (medium, held)

**Claim.** A consumer below the first retained sequence (unprotected, protected and overridden by a cap, or protected and
stale under `ConsumerTTL`) is reported healthy: `quill consumers` prints a lag that counts records that no longer exist,
`protected` true for a consumer the writer already stopped protecting, and an empty NOTE; `quill check` prints HEALTHY;
the CLI never prints `Dropped` or `StaleConsumers`; the deletion is logged at Info without the consumer's name; `Dropped`
lives in the writer's memory and is 0 after a restart. Merged ids: A5-12, A5-25, A5-26, A5-37.

**Grounds.** Fault run (protected consumer `billing` at offset 1, a `MaxBytes` cap, 80 appends): writer `Stat` head 80,
first 65, `Dropped` 8; a Warn-level Logger saw nothing; `Consumers()` gave `{billing, Offset 1, Protected true, Err nil}`;
the handle's `Lag` was 80 and `Read` returned a `GapError` for records 1 to 64; after Close and Open `Dropped` was 0. CLI
on that directory (a voter's run, 80 records, cap leaving 16): `quill consumers` printed `billing 1 80 true`, NOTE empty;
`quill check` HEALTHY exit 0; `quill stat` has no dropped line. Stale case: `StaleConsumers` empty until the first pass,
frozen after the consumer caught up, 0 after restart, and `Reader.Consumers` still reports `Protected` true. Code:
`Dropped++` then an Info "deleted segment" without the consumer (retention.go); `quill consumers` lag is
`head+1-offset` with no `first` test (cmd/quill/consumers.go); `quill prune` prints segments, bytes and first only.

**Warrant.** A consumer below `First` has lost records for good. Every operator surface for consumers reads as healthy and
the counter that records the loss is process-local, never printed, and reset on restart. `Lag` overstates what can still be
read, so a lost consumer looks like a large but recoverable backlog.

**Backing.** Decision 38 (reads never skip a gap), decision 12 (`Dropped` is in-memory and resets; a persisted counter is not
built), decision 66 (`Dropped` and `StaleConsumers` live in `Log.Stat`), decision 36 (only the "ahead" note was added).

**Rebuttal.** Deliberate and documented. The consumer gets a loud `GapError` at its next read, and an embedder that polls
`Log.Stat().Dropped` has the counter; nothing derivable from the consumer file alone says "stale" without the policy's
`ConsumerTTL`. One part of the claim was overstated and a voter corrected it: stale consumers are logged at Warn with the
consumer's name at every pass (retention.go); the missing named line is the deletion's, which is Info.

**Verdict.** CONFIRMED (C, C). Both reproduced. Since the audited tree, 2bf7412 (A3-27) made seals count `Dropped` from the
last prune pass's offsets; it does not change this item.

**Disposition.** Held for the owner (semantic: CLI output and `ConsumerInfo`). Recommended: derive, do not persist. `quill
consumers` NOTE and a JSON field when `offset < first` ("behind the first retained sequence: records offset to first-1 are
gone"), with lag counting only retained records or the note saying otherwise; for protected consumers compute staleness
from the policy's `ConsumerTTL` and the file mtime; log Warn, with the consumer's name, when a deletion passes a consumer's
offset; print `dropped` in `quill prune`.

#### A5-22: `MinFreeBytes` stops being enforced when statfs fails, and `diskFull` is invisible (medium, held)

**Claim.** When statfs fails the floor is not enforced (appends keep succeeding below it) or, if `diskFull` was already set,
frozen (every append keeps failing with ENOSPC even after space returns). `Err()` stays nil in all three states and no
`Stats` field shows `diskFull`. Merged ids: A5-14, A5-43.

**Grounds.** In `capsLocked`, a statfs error logs one Warn and leaves `diskFull` unchanged (retention.go:229-243); the
transition Warn fires only on false to true and recovery is not logged; `diskFull` is read only at append.go:59 and written
at retention.go:242. Reproduction on the statfs seam: floor breached, `diskFull` true and append error "free space below
MinFreeBytes ... no space left on device"; statfs then failing with 1 GiB reported free: `diskFull` stayed true, appends
still failed, `Err()` nil; with statfs failing from the start nothing was deleted, appends succeeded, `Err()` nil, the only
trace one Warn per pass (and nothing without a Logger). When statfs worked again `diskFull` cleared.

**Warrant.** docs/userguide.md tells operators to watch `Err()`, but the state in which the log rejects every append is not
in it, and the floor silently reverts to unenforced when its input fails.

**Backing.** scope.md and decision 25 (checked at each seal and prune pass; appends fail wrapping ENOSPC when nothing is
left to delete); the last clause of the bounded-disk guarantee.

**Rebuttal.** Appends fail loudly with a wrapped ENOSPC, so producers see `diskFull`; statfs failing on a live directory is
rare; evaluation at seals and passes is documented as best effort. The `free += size` assumption (the unlink may free
nothing while a reader or archiver holds the segment open) was inferred and not reproduced.

**Verdict.** CONFIRMED (C, C). One voter reproduced on the seam; the other traced the code and did not run. Voters rated it
low to medium because the trigger is rare.

**Disposition.** Held for the owner (semantic: `Stats` additions). Recommended: set `bgErr` when statfs fails or `diskFull`
is set, log when `diskFull` clears, add `Stats.DiskFull` and `Stats.FreeBytesAtLastPass`; consider treating the floor as
unmet only after N consecutive statfs failures. Fold into the `Err()` decision (A5-10).

#### A5-29: `quill archive` reports source loss once, and not at all after `unregister` (medium, held)

**Claim.** Archive reports a source gap exactly once, on stderr, with exit 0; later runs are silent success while the
destination is UNHEALTHY. Separately, when the `_archive` consumer is absent or recreated, Archive does not reconcile the
committed position with the destination's tail, so records lost in between produce a hole in the destination with no message.

**Grounds.** cmd/quill/consumers.go:160 prints "the source lost records %d to %d; the archive holds the same hole" and
returns `exitOK`; Archive raises the gap only when the offset is below `First` or between segments copied in this run, and
then commits past it, so the next run sees none. `_archive` is opened with `ProtectOn`, which creates a missing consumer at
the first retained sequence; the destination's `quill.tail` is never read. Reproductions (built CLI; scripts of `append`,
`roll`, `archive`, `prune -max-records`): A, archive, add records, prune: archive #2 printed the message, exit 0; #3 printed
nothing, exit 0; `quill check -dir dst` UNHEALTHY "sequences 61 to 110 are missing", exit 3. B, archive, `quill unregister
-name _archive` (the documented reset, docs/userguide.md), add records, prune: archive #2 printed nothing, exit 0; check on
the destination exit 3 with the same hole.

**Warrant.** The archive is the backup. A script running `quill archive && notify-ok` sees success for an incomplete archive
in A and no message in B; the only durable evidence is `quill check` on the destination, which nothing prompts.

**Backing.** scope.md and docs/userguide.md ("a source that lost segments gives an archive with the same hole, and the verb
says so"); decisions 28, 67, 68.

**Rebuttal.** The verb does say so, once. Exit 0 on a gap is deliberate (consumers.go; A1-58, decisions 85 and 53). In B the
missing records were deleted before any archiver protected them, so there is arguably no archive-side loss to report; the
destination hole is detectable by `quill check` (verified, exit 3).

**Verdict.** CONFIRMED (P, C). Both reproduced. The PLAUSIBLE voter treated A as the prior-deliberate A1-58 (see below,
and the A1 record) and B as new but could not rule out that an explicit operator reset counts as acknowledging the loss.
The user guide sentence is false for B (verified by both runs).

**Disposition.** Held for the owner (semantic: an exit status and Archive's documented return). Recommended: a distinct
non-zero exit (the unhealthy code 3) for a gap, and have Archive compare the committed offset with the destination's
`quill.tail` (`dest.Seq+1` against the first segment it copies) so B returns `*GapError` as A does. Until then correct the
user guide sentence for the `unregister` case.

#### A5-23: the published disk bound is exceeded when `MaxBufferBytes` is below `FlushBytes` (low, held, bug)

**Claim.** The bound `MaxBytes + SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes+34)` is exceeded when
`MaxBufferBytes < FlushBytes`, because the real flush term is `max(FlushBytes, MaxBufferBytes)` and `validate` does not
require `MaxBufferBytes >= FlushBytes`.

**Grounds.** `maxActiveBytes` uses the larger of the two (log.go:203, with the A3-16 cap per term from
120c8ac); `validate` has no such rule; scope.md:256 and the bounded-disk guarantee state the formula with `MaxBufferBytes`
alone, docs/userguide.md uses the maximum for the ceiling. Reproduction (two voters, a test that sums the directory's file
sizes at every caps evaluation through the statfs seam): `SegmentBytes` 1 MiB, `FlushBytes` 700000, `MaxBytes` 2,900,000,
100-byte payloads. `MaxBufferBytes` 1: published bound 3,949,577, peak 4,200,444 (also 4,229,330 in the other voter's run),
exceeded; control with the default (16 times `FlushBytes`): bound 15,149,576, not exceeded.

**Warrant.** A numeric guarantee is violated by a Config that `Open` accepts, with no warning. The overshoot is the
transient between a seal and the cap's unlink.

**Backing.** scope.md bounded-disk guarantee and decision 69; neighbours fixed earlier: docs-39 (single-record term) and
A1-25 (ceiling uses the maximum).

**Rebuttal.** Needs a non-default `MaxBufferBytes` below `FlushBytes`, which only matters while flushes fail, so nobody has
a reason to set it. The peak is a transient inside one locked seal and `Stat()` cannot observe it.

**Verdict.** CONFIRMED (C, C). Both reproduced.

**Disposition.** Held for the owner (semantic). Recommended: have `validate` refuse `MaxBufferBytes < FlushBytes` with
`ErrConfig` (one line, and the published formula becomes exact); the alternative is to reword the formula with the maximum.

#### A5-3: `ReadWait` and `Follow` on a failed log block until the context ends (low, held)

**Claim.** After an fsync error `Log.ReadWait`, `Consumer.ReadWait` and the `Follow` iterators keep blocking until the
context ends instead of returning the failure; to a consumer goroutine a log that can never produce another record looks
like an idle one.

**Grounds.** `Log.wait` (read.go:355-373) checks `l.closed` and `l.head >= offset`, never `l.failed`; `fail()` broadcasts so
waiters wake, find the head unchanged and block again. Reproduction (two voters, in-memory file system, fsync EIO
injected): `Flush`, `Append` and `Err()` return the `ErrFailed` error; `ReadWait` with a 700 ms context returned no records
and "context deadline exceeded" after 701 ms; `Consumer.ReadWait` and `Log.Follow` did the same; plain `Read` of existing
records still worked (3 records). `Close` returns the failure.

**Warrant.** `ReadWait`'s only exits are a record, the context or Close; after failure none arrives from the log, so a
consumer loop with a long context sees no error and no data forever while appenders see `ErrFailed`.

**Backing.** scope.md and errors.go (a failed log stays failed until reopened); decision 37.

**Rebuttal.** Reads of durable data stay valid on a failed log and `ReadWait` promises only to block until records are
flushed. The failure is visible to the same process through `Err()`, `Append` and `Close`, so a consumer that polls `Err()`
is fine. Returning `ErrFailed` would change documented behaviour (semantic), and a caller may prefer to keep draining.

**Verdict.** CONFIRMED (C, C). Both reproduced. A voter noted the claim slightly overstates "indistinguishable": `Err()` does
expose it.

**Disposition.** Held for the owner (semantic). Recommended: in `Log.wait`, when `l.failed != nil` and `l.head < offset`,
return `l.failed` (reads of existing records still succeed), and document it on `ReadWait` and `Follow`.

#### A5-9: `quill prune` and `quill roll` discard `Close`'s error (low, held)

**Claim.** The CLI verbs that run through `withWriter` (prune, roll) discard `Close`'s error, which carries the final
directory fsync and the lock release, so a fatal directory-fsync failure at that point exits 0.

**Grounds.** cmd/quill/write.go:83-92 `defer l.Close()` with the comment "discarding Close's error"; `Close` runs the
closing `SyncDir` and returns its error; `deleteLocked` syncs the directory before each unlink, so the last unlink's
durability is confirmed only by `Close`'s `SyncDir`. Reproduction (one voter): a patched scratch copy whose `SyncDir`
returns EIO on the Nth call (the shipped CLI has no fault hook); `quill prune -retention 1s -force` printed its summary and
exited 0; `quill roll` exited 0; `quill recover`, which reports `Close`, printed "input/output error" and exited 1. The other
voter read the code only. A real fsync failure was not provoked.

**Warrant.** G11 says a directory-fsync error is fatal; the CLI turns it into success for these two verbs.

**Backing.** Decision 60(e); decision 100 already fixed the same pattern for `append` and `recover`.

**Rebuttal.** Bounded: earlier unlinks were each synced first, so after a crash the surviving segments are still a suffix.
For `roll` the rotation already syncs the directory, so the exposure is `Close`'s flush, publish and lock-release errors.

**Verdict.** CONFIRMED (C, C). One reproduction, with the fault simulated.

**Disposition.** Held for the owner (semantic: an exit code). Recommended: report `Close` in `withWriter` the way
`cmdAppend` and `cmdRecover` do, exit 1 on a `Close` error.

#### A5-32: a rerun never verifies a same-size destination segment (low, held)

**Claim.** `Archive` keeps a destination segment that already exists with the same size, without opening it, so a damaged
copy survives a rerun, the run exits 0, and only `quill check` on the destination detects it.

**Grounds.** archive.go:178 (`di.Mode().IsRegular() && di.Size() == size` returns success); the doc on `copySegment`
says so. Reproduction (one voter, CLI): archive 7 records over 3 segments, flip a byte in `dst/0000000001.log`, `quill
commit -name _archive -offset 1`, archive again: silent, exit 0; `quill check -dir dst` UNHEALTHY with the damaged segment,
exit 3. The other voter read the code.

**Warrant.** Rewinding the archiver is the natural way to re-copy after finding damage, and it reports success while keeping
the bad file.

**Backing.** scope.md and decision 28 (archive verifies every frame's CRC; idempotent).

**Rebuttal.** Archive verifies the source's CRCs, not the destination's; `quill check -dir dst` is the intended destination
audit. Skipping rewrites is what makes a rerun idempotent and cheap.

**Verdict.** CONFIRMED (C, C). One reproduction.

**Disposition.** Held for the owner (semantic: documented behaviour). Recommended: on a same-size destination file verify its
CRC and replace it when it fails, or require a flag to force a re-copy.

#### A5-44: `quill ls` drops any log it cannot open (low, held)

**Claim.** `quill ls` silently omits a log directory whose `OpenReader` fails (corrupt `quill.format`, future format,
permission denied) and exits 0; nothing is printed unless no log at all was found.

**Grounds.** cmd/quill/read.go:45-47 `if err != nil { continue }`; the `Stat` failure branch a few lines later prints an
"error:" row (read.go:50-53). Reproduction (two voters, built CLI): `parent/good` and `parent/bad` with `quill.format`
overwritten by "garbage": `quill -dir parent ls` printed the header and the `good` row only, exit 0; `quill -dir parent/bad
stat` and `check` exit 1 with "corrupt log: ... not a quill marker". With `chmod 000` on a log, `ls` omitted it likewise.

**Warrant.** An inventory command that omits a broken log is indistinguishable from the log not existing. The skip exists
for non-log directories (`ErrNoLog`) but swallows `ErrCorrupt`, `ErrFutureVersion` and permission errors too.

**Backing.** scope.md and docs/reference.md list `ls` as showing the logs under a parent; the `Stat` branch shows an
"error:" row is the intended style.

**Rebuttal.** `ls` lists logs, it does not validate them, and a skip is the only way to ignore unrelated directories;
`check` and `stat` on the specific log fail loudly. A voter noted `ErrNoLog` is a distinct error the skip could be limited to.

**Verdict.** CONFIRMED (C, C). Both reproduced.

**Disposition.** Held for the owner (semantic: exit code and output). Recommended: skip only `errors.Is(err, quill.ErrNoLog)`;
for anything else print an "error:" row, and decide whether `ls` then exits 1.

### Fixed

Each fix was reviewed by a fix-skeptic through the lenses listed; "red before" is the lens test-fails-without-fix: the new
tests were run against the tree with only the production files reverted.

#### A5-2: a writable Open and `quill recover` cut the newest segment past the marker without a trace (medium, fixed 38d2644)

**Claim.** A writable Open, and `quill recover`, truncate the newest segment past the marker and record nothing (no log
line, no `Stats` field, no Open warning, no CLI output), including a durable, `Sync`-acknowledged record when a stale marker
is combined with damage in the final frame (the documented residual). The head silently moves backwards relative to what the
writer acknowledged.

**Grounds.** At the audited tree load.go performed `f.Truncate(info.end)` and `Sync` with no `logf`; `recoverSegment` returned
only a bool that was dropped; `cmdRecover` printed `recovered: head %d, first %d, %d segments`. Reproduction (an in-package
test and the CLI on a fixture): marker at offset 266, seq 10 (stale), a 641-byte file with 20 frames, last frame damaged;
a writer Open with a Logger gave head 19, `Err()` nil, Logger output empty, segment 641 to 599 bytes; `quill stat` head 10,
exit 0; `quill check` HEALTHY exit 0; `quill recover` printed `recovered: head 19`, exit 0. A voter reproduced the same
with a torn tail, a complete-length garbage frame and a damaged acknowledged frame ending in one observable state.

**Warrant.** Truncating a torn tail is correct under the crash guarantee, but an operator cannot tell a routine torn tail
(zero bytes or a partial frame) from a cut that removed a complete-length frame failing its CRC (bit rot, or a stale-marker
residual, possibly acknowledged data). Both leave identical observable state.

**Backing.** scope.md and decision 39 (the residual of the crash guarantee); docs/userguide.md recovery section.

**Rebuttal.** Truncation is the designed, documented behaviour, decision 39 accepts that acknowledged records there can be
lost "silently", and the Logger is optional. The cut cannot be replayed. The Logger doc never promised such a line, so the
severity was argued down by both voters.

**Verdict.** CONFIRMED (P, C). One voter reproduced.

**Disposition.** Fixed in 38d2644, FIXED-VERIFIED by the fix-skeptic (6 of 6 lenses pass). The truncation is unchanged. A
writable Open that cuts now logs a Warn "cut the newest segment past the marker" (segment, offset, bytes, `kind=torn` or
`damaged`), load.go:420; every offline write verb prints that one record on stderr (cmd/quill/write.go, the `lossOnly`
handler, write.go:74). Red before: reverting load.go, log.go and write.go failed both new tests ("the cut was not logged";
"stderr does not report the cut"). Reported by the skeptic: all four verbs (`recover`, `append`, `roll`, `prune`) print the
line from the built binary, stdout and exit codes unchanged. Residual limits (reported): a bit flip in the length byte that
overshoots end of file is classed torn; the library reports only when `Config.Logger` is set. The first attempt, which
reported only from `recover`, was rejected (reported by the fix summary).

#### A5-21: a prune pass aborts at the age-rotation flush while flushes fail (medium, fixed dd460a7, bug)

**Claim.** While flushes fail (for example ENOSPC) the prune pass aborts at the age-rotation flush and never reaches the age
sweep or the caps, so the job that would free space is skipped exactly when the disk is full; `Err()` shows the flush error
and nothing says retention stopped.

**Grounds.** At the audited tree retention.go:98-101 called `flushLocked` for a quiet active segment at least
`SegmentMaxAge` old and returned its error before the sweep and `capsLocked`. Reproduction (two voters, in-memory file system
and a flusher run, writes failing with ENOSPC; `SegmentMaxAge` 1 min, `Retention` 1 h, sealed segments 2 h old): `Prune`
returned the error, segments 2 to 2, first 1, `Err()` ENOSPC, nothing logged; control with flushes working: `Prune` nil,
first 61. A disk modelled as full until a `.log` file is removed stayed wedged for three passes and 3.5 s of the background
flusher; with `SegmentMaxAge` negative the same pass freed space and flushing resumed.

**Warrant.** Expired sealed segments are deletable without a successful flush and deleting them is the log's only
self-recovery; the abort turns a recoverable full disk into a wedge until someone frees space outside quill.

**Backing.** scope.md and decisions 24 and 25 (caps run at every pass); decision 60 and scope.md (the writer recovers on its
own when space returns).

**Rebuttal.** Needs `SegmentMaxAge` above 0 (default `Retention`, 24 h) and a quiet active segment at least that old when
flushes start failing; the failure is loud on the producer side and `Err()` is non-nil; the errors were injected, not a real
ENOSPC, and not run on Linux.

**Verdict.** CONFIRMED (C, C). Both reproduced.

**Disposition.** Fixed in dd460a7, FIXED-VERIFIED (7 of 7 lenses pass). `pruneLocked` no longer returns when the age-rotation
flush fails: it logs a Warn "age rotation skipped because the flush failed; the sweep and the caps still run"
(retention.go:129) and continues; a fatal failure (an fsync error that sets `l.failed`) still returns. Red before: with only
retention.go reverted the new test failed at "Prune while flushes fail: no space left on device". Reported by the skeptic: the
flusher's own ticker went from 3 sealed segments unchanged to 3 to 1 with the fix. Residual (reported): the same pass can
still return early when the marker publish after a successful age rotation fails, which blocks deletion by design, and
`deleteLocked` publishes first, so a pending marker publish on a full disk still blocks every deletion; the skeptic flagged
the second as a possible separate finding and did not test it.

#### A5-39: a failed policy write left the previous run's `quill.policy` as current (medium, fixed 1b68eee)

**Claim.** If the policy write at Open fails, Open succeeds and the previous run's `quill.policy` stays on disk. `quill
check` reports the old caps as current (`Unbounded` false for a writer that now has no cap), an offline `quill prune` acts on
the old retention, and `quill ls` shows it. The only signal is the writer's latched `Err()`.

**Grounds.** At the audited tree log.go recorded a `writePolicy` failure only in `bgErr`; Check reads the file as truth
(stat.go), `cmdPrune` uses `ReadPolicy`, `ls` shows it. Reproduction (in-package, a file system wrapper failing the rename
onto `quill.policy`; first run `MaxBytes` 4096, second `Retention` 168 h with no caps): `Open` returned nil, `Err()` was the
injected EIO, `Reader.Check` gave healthy true, unbounded false, no problems, and the file still said `max_bytes 4096`. A
voter reproduced with real files and the built CLI (`chflags uchg` on `quill.policy`): `quill check` HEALTHY, exit 0, old
settings, and `quill ls` the old settings.

**Warrant.** The one artifact that tells offline tools what the writer promised is silently stale, and its consumers have no
staleness test.

**Backing.** Decisions 29, 48, 87 and docs/format.md (Config is authoritative, the file informational); scope.md (an offline
tool "sees" the settings).

**Rebuttal.** The trigger was injected; a realistic one is ENOSPC or EIO at exactly that write, because `quill.tail` is
written first. Offline prune needs the writer stopped, and the file is documented as informational.

**Verdict.** CONFIRMED (C, C). One voter reproduced on both routes.

**Disposition.** Fixed in 1b68eee, FIXED-VERIFIED (6 of 6 lenses pass). A failed write now removes the stale file, logs a
Warn once, keeps the error in `Err()`, and retries in `flushLocked` (which Close calls) and in `pruneLocked` until a write
succeeds, then clears the error and logs an Info (`recordPolicy`, log.go:664). Check, `ls` and offline prune then see no
policy: Check warns and reports UNBOUNDED, `prune` exits 1 "quill.policy cannot be read; pass the retention flags
explicitly" (the skeptic ran all three on the built binary). Red before: with log.go, flush.go and retention.go reverted,
four tests failed (stale file still on disk, failure not logged, error not cleared after recovery on flush and on prune).
Residuals (reported): if the failure-time remove also fails the stale file stays until a retry succeeds (decision 29 says so,
and a test covers it); during a persistent fault every flush does a temp write, a rename and a remove, with one Warn. The
first attempt, a retry at the prune pass only, was rejected for missing a short-lived writer (reported). The A5-10 clearing of
`bgErr` now exists for this error only.

#### A5-42: retention deleted a damaged sealed segment and the evidence went with it (medium, fixed 66081fe)

**Claim.** When retention deletes a damaged sealed segment the deletion is logged at Info as an ordinary "deleted segment"
without the word damaged, `Stat().Holes` drops the hole, Check becomes healthy, and a restart loses the `Err()` latch.
Nothing records that data was lost to damage rather than aged out.

**Grounds.** At the audited tree retention.go logged `deleted segment` with path and last only and filtered the holes below
the new first; `loadErr` is memory only. Reproduction (a voter, four segments of three records, byte flipped in segment 0,
`MaxRecords` 7): before the prune holes `[{1 3}]`, `Err()` "damaged at byte 112", Check unhealthy; after `Prune(true)` holes
empty, Check healthy, two Info lines and no mention of damage; after a reopen `Err()` nil.

**Warrant.** The hole and the damaged file are the only artifacts of corruption; a cap can delete them before an operator
ever runs Check, and the log then looks pristine.

**Backing.** Decision 98 (a damaged segment is deleted like any other); scope.md (damaged data counts toward the caps).

**Rebuttal.** Deleting is licensed by decision 98; the writer's first Open already put the hole in `Stat` and `Err()`; the
data was unreadable anyway. A line exists, though without the damage note.

**Verdict.** CONFIRMED (C, C). One voter reproduced; severity argued down to low or medium by both.

**Disposition.** Fixed in 66081fe, FIXED-VERIFIED (6 of 6 lenses pass). Deleting a damaged segment is now logged at Warn,
"deleted damaged segment", with segment, last and damage (retention.go:312), and the offline write verbs pass that record to
stderr (the `lossOnly` filter, which was `cutOnly` and was renamed because the old name became false). Decision 98 carries a
dated amendment; the deletion itself is unchanged. Red before: with retention.go and write.go reverted both new tests
failed (the line was Info, and offline stderr was empty). Reported by the skeptic: `quill prune -max-records 3 -force` on a
log with a damaged first segment printed the Warn and Check went UNHEALTHY to HEALTHY as predicted. Residual (reported):
only a configured Logger or an offline prune sees it; a library user with a nil Logger gets nothing, and no persistent marker
was added (the claim's own proposal made that optional). The first attempt, which fixed the library only, was rejected for
leaving offline `quill prune` silent (reported).

#### A5-16: the writer never re-checks that its `quill.lock` is still the file at that path (low, fixed 6acc23e)

**Claim.** If `quill.lock` is removed under a live writer (a tmp cleaner, `rm *.lock`, a restore over the directory), a
second writer opens without `ErrLocked` and the two overwrite each other's acknowledged durable records, with no error from
either and Check HEALTHY. Merged id: A5-8.

**Grounds.** The lock is held by an `flock` on a file descriptor opened once at Open (lock_unix.go); nothing compared the
descriptor with the path afterwards. Reproduction (two voters, a program on the public API, macOS and the Linux box): writer
A appended and flushed seq 1; `os.Remove("quill.lock")`; writer B `Open` returned nil; both `Append` and `Flush` returned
nil, A head 2 and B head 2; a Reader then saw `2 B2-from-writer-B` (A's durable record 2 lost) and Check healthy true.

**Warrant.** G21 (one writer per log) rests on the file's existence; losing it turns the lock into a silent no-op whose
failure mode is loss of acknowledged data.

**Backing.** G21 (scope.md, decisions 17 and 62); decision 41 (lock files are never deleted by quill).

**Rebuttal.** Deleting the lock file is an operator or third-party act outside the supported model, an advisory lock cannot
defend against tampering, and the bind-mount case is documented unsupported. Hardening, not a violation, so low.

**Verdict.** CONFIRMED (C, C). One voter ran it on both platforms; the other read the code.

**Disposition.** Fixed in 6acc23e as hardening, FIXED-VERIFIED (6 of 6 lenses pass). Before each flush and prune pass the
writer compares `l.lock.Stat()` with `Lstat("quill.lock")`; on a mismatch or a missing file the log is failed (`ErrFailed`) and
an Error is logged (`lockLostLocked`, flush.go:79-100, called at flush.go:35 and retention.go:106); the one new helper is
`fsx.SameFile` in an internal package, because `os.SameFile` cannot compare the simulator's file info. Red before: with
flush.go, retention.go, errors.go and internal/fsx/fs.go reverted, all three new tests failed (flush after removal and after
replacement, prune after removal returned nil). Reported by the skeptic: a public-API test with a background flusher on macOS
and on the Linux box showed A's next `Sync` return `ErrFailed` naming the lost lock, B appending, and a Reader at head 2.
Residuals (reported, and stated in scope.md): a second writer that opens before the first writer's next flush is not
prevented, one flush can still overwrite in the window, and `consumers/<name>.lock` has the same no-recheck property.

#### A5-40: `Stats.MarkerAge` read 0 on an empty log before the first publish (low, fixed 906d49c)

**Claim.** `Stat().MarkerAge` on a writer reads 0 when no marker has been published since Open; on a fresh empty log with
publishing failing, "never published" shows as "just published".

**Grounds.** stat.go:62 leaves `MarkerAge` 0 while `markerAt` is zero. Open's "force a write" set `l.marker = tail{}`, which
an empty log's zero marker already equalled, so `publishLocked` skipped the write and `markerAt` stayed zero. Reproduction
(two voters): a new log, directory `chmod 0500`, three Append and Flush across 30 fake minutes: `Flush` "permission denied",
writer `MarkerAge` 0 s, head 3, reader head 0, `Err()` set. A voter showed it also affects an idle fresh log with no failure
(one fake hour, `MarkerAge` 0) and an empty log closed and reopened, contradicting the auditor's rebuttal that a reopened
log is unaffected.

**Warrant.** Zero is returned both for "just written" and for "no marker written by this process".

**Backing.** stat.go documents "how long ago quill.tail was last written"; decision 47 calls it the only liveness hint.

**Rebuttal.** `Err()` is non-nil in the repro and only a log with no frames is affected; the Reader's `MarkerAge` (file
mtime) is correct.

**Verdict.** CONFIRMED (C, C). Both reproduced.

**Disposition.** Fixed in 906d49c, FIXED-VERIFIED (6 of 6 lenses pass). One line: Open's forced write is now
`l.pendingPublish = true` (load.go:432), which `publishLocked` honours and clears. Red before: restoring the old line failed
the new test ("new empty log: MarkerAge 0s, want 30m0s"). Reported by the skeptic: on real files after `chmod 0500` the
writer's `MarkerAge` went from 0 s to 2.015 s after 2 s. Not run by the skeptic: deadcode and the fuzz step, which a
one-line change cannot affect.

#### A5-31: a failed `ConsumerTTL` probe silently kept a consumer live (low, fixed d451fac)

**Claim.** When the `ConsumerTTL` probe of a behind, silent protected consumer errors (mtime `Lstat` error, or a non-gap
read error such as EACCES or EMFILE on the segment at its offset), the consumer is silently treated as live: it keeps
pinning segments, `StaleConsumers` stays empty, and nothing is logged.

**Grounds.** At the audited tree the staleness test in `consumers()` (retention.go) set `stale` only when both the `Lstat`
and the read succeeded; every other path left it false with no `logf`. Reproduction (two voters, real directory, `ConsumerTTL`
1 s, protected consumer behind for 2.2 s, `chmod 000` on the first segment): `Prune` nil, `StaleConsumers` empty, 5 (or 2)
segments, `Err()` nil, Logger empty; after `chmod 600` the next pass gave `StaleConsumers [c1]`, segments 1, a Warn and the
delete lines.

**Warrant.** The probe's failure is indistinguishable from "not stale yet", in the protective direction, so data is kept and
the disk grows until a cap fires or the error clears, and nothing says the TTL guarantee is not being evaluated.

**Backing.** Decisions 45 and 82; A1-82 (a read I/O error is never treated as a gap), which the probe respects.

**Rebuttal.** Fails safe. Causes are mostly transient and the same error would surface on every other read. Caps still
bound the disk.

**Verdict.** CONFIRMED (C, C). Both reproduced.

**Disposition.** Fixed in d451fac, FIXED-VERIFIED (6 of 6 lenses pass), log-only as the finding proposed. The consumer stays
live, and the prune pass logs a Warn "consumer staleness probe failed; treating it as live" with the consumer and the error
(retention.go:118), collected without the lock by `consumers()` and logged in `pruneLocked`. Red before: with retention.go
reverted both subtests (read error, mtime `Lstat` error) failed. Reported by the skeptic: on real files with `chmod 0` the
Warn appeared with `err="open ...: permission denied"` and recovery was unchanged. No `Stats` field was added, so this
state is LOG-ONLY and, with a nil Logger, still NONE; the Warn repeats each pass while the failure lasts. The skeptic noted
the second read after a gap is handled but not covered by the new test, and that the Logger docs do not list the new Warn.

#### A5-33: segments and `consumers/` ignore `Perm` under a tighter umask (low, resolved by an earlier commit)

**Claim.** With a umask tighter than `Config.Perm` the segments and `consumers/` are created more restrictive than `Perm`
while the state files are exact, and `quill check` and the writer report nothing, so the documented group-read arrangement
is silently not provided.

**Grounds.** Reproduction (two voters, macOS and the Linux box, umask 077, `Perm` 0660): segments 0600, `consumers/` 0700,
`quill.format`, `quill.policy` and `quill.tail` 0660; `quill check` printed HEALTHY, exit 0. Code: segments are created with
`OpenFile(..., Perm)` and no chmod, directories with `Mkdir(dirPerm)`; only `writeState` chmods.

**Warrant.** The marker, which a reader in another account can open, is 0660 while the segments it names are 0600, so that
account sees the log exist and cannot read it; nothing on the writer side signals it.

**Backing.** scope.md and decisions 51 and 74 as they read at the audited tree ("segments are exact whatever the umask").

**Rebuttal.** The primary defect is a correctness divergence rather than a degradation; the reader-side EACCES is inferred
(no second account was available) and is loud on the reader.

**Verdict.** CONFIRMED (C, C). Both reproduced the mode divergence.

**Disposition.** Resolved by 1f896c7 (A3-12), which landed before any A5 commit: the documents were changed to say that the
writer's segments, lock files and every directory go through the umask and are never chmodded, and that state files and
Archive's segments are exact (log.go `Config.Perm`, scope.md, format.md, docs/userguide.md, decision 51 and 74 amendments
dated 2026-10-31, CHANGELOG). Verified by the recorder: the code matches (segments and directories are created without a
chmod). No A5 commit and no A5 fix-skeptic. The ledger's "fix sha" field for this item carried 6acc23e, which is A5-16's
commit; that was a ledger error and is not recorded as a fix. Not applied: the finding's optional Check warning for a mode
mismatch would flag every writer run under a tighter umask, which A3-12 documents as normal; if the owner wants it, it is a
new decision (see stage2-owner-decisions.md).

### Refuted

#### A5-5: a long-lived Reader does not notice that its directory was replaced by another log (medium, refuted)

**Claim.** A Reader, and a Reader-backed Consumer, keep serving the new log's records from the old cursor with no error while
`Reader.ID()` still reports the old id and Check says HEALTHY.

**Grounds.** `OpenReader` loads `quill.format` once (reader.go:102) and `view` never re-reads it. Reproduction (an in-package
test and a second program): a reader read 10 records, the directory was removed and a new log written to 30 records;
`r.Read(11, 100)` returned 20 records, nil error, first payload "NEW", `ID()` unchanged. With sealed segments `Read(41,5)`
returned the new log's records; `Read(1,5)` failed with `ErrCorrupt` "segment 0 changed size from 270 to 310", a
size-cache accident; `Stat` head 60, `Check` healthy true; the Reader-backed consumer returned NEW records.

**Warrant.** Sequence numbers of two logs interleave on one cursor; the only error is incidental and misattributes the cause.

**Backing.** Decision 62 and docs/userguide.md (`ErrWrongLog` "the log was recreated" shows recreation is a recognised event);
guarantee 1.

**Rebuttal.** Needs an operator to wipe and recreate the directory under a live Reader; CLI invocations build a new Reader
each time, except `tail -follow` (inferred); re-reading `quill.format` on every view costs a read on a hot path.

**Verdict.** REFUTED (R, C, R). Reproduced by the CONFIRMED voter and by the third voter (who ruled for
REFUTED on a split); the first refuter read the code and decision only. Reason: decision 83 deliberately does not re-check cached files on every call and says a
changed size "can only mean replacement, which the next open sees"; no guarantee covers directory replacement under a live
Reader; guarantee 1 holds per call; the documented restore flow keeps the log id, so an id check would not fire there. The
CONFIRMED voter argued the fix is one constant-cost check per view. The deciding voter recorded that the ruling went to
REFUTED because it was uncertain whether this is intended, and that neither side's evidence was wrong.

**Disposition.** Refuted as a defect; a hardening remains. Since the audited tree, 495e457 (A3-30) made `Commit` and a
Reader's `wait` re-check `quill.format` once per commit or poll and return `ErrWrongLog` on a mismatch (verified by the
recorder, reader.go `wait` and consumer.go `Commit`; the A3-30 message says plain `Read` and `Scan` are not re-checked). So
the stamping of the old id into the new log is closed; plain `Read` on a replaced directory still serves the new log. The
ledger keeps NONE for that surface.

#### A5-6: a stale marker is invisible to every reader-side tool (low, refuted)

**Claim.** A stale marker that hides durable records is not shown by `quill check` or `Reader.Check` (HEALTHY), `quill stat`
exits 0, and no field compares the active segment's file size with the marker's offset, so "writer idle", "writer failing to
publish" and "marker rolled back by an OS crash" look the same.

**Grounds.** Reproduction (voters, in-package and CLI on a fixture): marker seq 10 offset 150, 20 more intact durable frames
(file 450 bytes), marker restored with an mtime one hour old: `Reader.Stat` head 10, records 10, bytes 450, marker age
1h0m0s; `Reader.Check` healthy true, no problems; `quill stat` head 10, exit 0; `quill check` HEALTHY exit 0. Another run
with 50 durable records beyond a marker whose publish was failing: `Reader.Stat` head 1, `Check` healthy. Code: `Check` bounds
the active segment at the marker and only flags a file shorter than the marker (stat.go:176-181).

**Warrant.** A persistent surplus (file longer than the marker, marker not recently written) can only mean a failed publish, a
rolled-back marker or a failed-state log; Check has both numbers and discards the information.

**Backing.** scope.md guarantee 10 (a stale marker hides until reopen or `quill recover`); decision 8.

**Rebuttal.** Between a flush's fsync and its marker rename the file is legitimately longer, and a failed log can end past
the marker, so a fatal would be a false alarm; the warning would have to depend on persistence.

**Verdict.** REFUTED (R, C, R). The CONFIRMED voter reproduced and confirmed the observation. The refuting voters read it as
accepted design: decision 8 ("a Reader does not probe whether a writer is live", confirmed 2026-10-28), decisions 8 and 40
and guarantee 10 ("readers do not guess"), decision 47 and docs/userguide.md (`MarkerAge` is the one liveness hint, printed
as "marker age" by `quill stat`), and decision 66 (the active segment is checked only up to the marker). The proposed warning
is the guess the design declined.

**Disposition.** Deliberately not changed. An optional diagnostic remains possible (a non-fatal Problem when the file exceeds
the marker and the marker's mtime is older than a threshold); no decision is needed unless the owner wants it.

#### A5-34: a bulk prune holds the log lock past `FlushInterval` with nothing recording it (low, refuted)

**Claim.** The visibility-latency bound has no observable: no last-flush time, buffered count or flush duration in `Stats`.
A prune pass holds the log lock through a directory fsync and an unlink per deleted segment on the flusher's single goroutine,
so a bulk expiry stalls Append, Head, Read and flush for seconds.

**Grounds.** retention.go (`deleteLocked`: `SyncDir` and `Remove` per segment under `l.mu`) and flush.go (prune and flush
share one goroutine). Reproduction (two voters, a test that times a concurrent `Head()` during one forced pass over expired
one-record segments): macOS/APFS, 400 segments: 3.05 s (one run) and about 1.9 s (the other), at the default guard 0.24 to
0.44 s, above the 250 ms `FlushInterval`; `Err()` nil throughout; on the Linux box 400 small segments 0.15 s and 100 real 8
MiB segments 0.25 s.

**Warrant.** While retention holds the lock a record appended just before is not flushed until the pass ends, and a reader's
view lags with no way to see that.

**Backing.** scope.md and decision 13 (a flush holds the lock through write and fsync); the soft promise of G38.

**Rebuttal.** Soft promise by design. Ordinary expiry removes one segment at a time; 400 expired segments is a bulk event; APFS
directory fsync is slower than ext4.

**Verdict.** REFUTED (R, C, R). The CONFIRMED voter reproduced the stall and the missing observable and rated it low. The
refuting voters: the per-unlink `SyncDir` is deliberate (retention.go cites decision 61: a crash always leaves a suffix);
decision 13 accepts the lock held through fsync; A1-30 recorded the same kind of stall as prior-deliberate; the trigger is
capped at a quarter of the sealed segments per pass unless forced; the large numbers are macOS-only (the Linux figures are
reported by the other voter, not verified by the refuter); `Stats` has `MarkerAge` and the Logger carries per-segment
lines, so "nothing records it" is overstated. What remains is an additive `Stats` field request, not a broken guarantee.

**Disposition.** Deliberately not changed. If the owner wants the observable, it is the `Stats.LastFlushOK` and
`BufferedRecords` of A5-1, which would serve this too.

### Decided earlier

These three were raised independently by the auditors and were already ruled on; they were not voted.

#### A5-7: `quill stat`, `quill ls` and `Reader.Stat` do not show a mid-file damaged sealed segment (low, prior-deliberate)

**Claim.** Records and holes are computed from first frames only, so a sealed segment damaged mid-file shows as healthy (full
count, no hole, exit 0) until a read or `quill check` scans it. Merged ids: A5-30, A5-45 (an absent middle segment, and a
symlinked or non-regular segment seen the same way).

**Grounds.** Reproduction (a fixture with segment 1 damaged at byte 372, 15 records unreachable): `quill stat` head 40, records
40, no hole line, exit 0; `quill check` UNHEALTHY, hole 21 to 35, exit 3; `quill cat` exit 1 "records 21 to 35 are
unavailable". The same holds for an Archive destination after an archived gap (group 2, run).

**Warrant, backing.** The in-process `Log.Stat` knows the hole (verified at Open) but a Reader has no index; decision 47 (ranges
from first frames, no scan); docs/reference.md lists `stat` as printing holes with no caveat.

**Rebuttal.** By design: a full scan is what `check` is for; `check` and read errors detect it and exit non-zero.

**Verdict.** prior-deliberate: A1-56 (a1-round2-correctness.md item 36), decisions 47 and 38, round-1 docs-43. No votes.

**Disposition.** Deliberately not changed. Proposed and still open as documentation only: say on `stat`, `ls` and
`Reader.Stat` that `Records` and `Holes` do not verify segment bytes and point to `quill check`.

#### A5-15: `Reader.Stat` returns `Dropped` 0 and `StaleConsumers` nil (low, prior-deliberate)

**Claim.** Both fields read as "nothing dropped, nothing stale" though a Reader cannot know either (absent shown as zero).
The CLI never prints them, so the lie is for library callers. Merged id: A5-38.

**Grounds.** A program after a run where the writer had `Dropped` 2: `Reader.Stat()` printed `Dropped=0 StaleConsumers=[]`;
stat.go field comments say "Only Log.Stat knows it".

**Warrant, backing.** A Reader-based dashboard can only ever show 0; decision 66; the house rule that zero while not running is
a lie in the reassuring direction.

**Rebuttal.** Documented in the field comments and decision 66; the CLI omits them; changing the type is an exported API
break at v0.1.0.

**Verdict.** prior-deliberate: decisions 66 and 12. No votes.

**Disposition.** Deliberately not changed. The recommendation if the owner reopens it: separate the writer-only fields into
their own type or make them a pair, before the first release, since afterwards it is an API break.

#### A5-17: an unparseable consumer file silently suspends the age sweep (medium, prior-refuted)

**Claim.** A `consumers/` entry that does not parse (a stray `.DS_Store`, an editor swap file, a consumer file copied from
another log, or an unlistable `consumers/`) suspends the whole age sweep with no Logger line, no `Err()` and no `Stat` field;
only `quill check` or `quill consumers` shows it. Merged ids: A5-13, A5-24, A5-35.

**Grounds.** `consumers()` sets `broken` on any list or file error (retention.go:41, 48) and `pruneLocked` then protects every
segment from the first retained one (retention.go:162-164); `list()` treats every name not ending `.lock` or starting
`.tmp-` as a consumer. Reproduction (an in-package test; six sealed segments 48 h old, `Retention` 1 h, `Prune(true)`
three times): control segments 6 to 1; with `consumers/.DS_Store` 6 to 6, first 1, `Err()` nil, `Dropped` 0, Logger empty
at Debug level; with a garbage consumer file, identical; Check says unhealthy with a consumer problem. The CLI prune
reproduction is in A5-36 (S2).

**Warrant.** The guarantee given up is "whole sealed segments older than `Retention` are deleted"; the pass still runs and
returns nil. The stale-consumer case next to it logs a Warn per pass; this case, with the larger blast radius, logs nothing.

**Backing.** scope.md and decisions 41, 67 and 97 (an unreadable consumer file protects everything; Check reports it fatally).

**Rebuttal.** Deliberately conservative; `quill check` exits 3 and `quill consumers` shows the error.

**Verdict.** prior-refuted: A1-34 (a1-round2-correctness.md item 14, REFUTED, RCR), the documented protect-everything
fail-safe. That record kept the "silence on the prune path" as a residual and said a Warn each pass "would close it (not
done)"; this pass raised it a second time with a reproduction on the current tree.

**Disposition.** Deliberately not changed here. Recommended, and cheap: record which file or listing error made the view
broken and emit one Warn per pass naming it, like the stale-consumer Warn; include the message in the CLI's stderr filter
(A5-36). A `Stats` field is optional.

## Deliberate exclusions

States the ledger marks LOG-ONLY or NONE that were not raised as items, each with the reason. None is a finding.

- **Inherent to the contract.** Oldest segments deleted by hand look like retention (G2): the guarantee is "contiguous from
  the first retained record". An idle writer, a wedged writer and a dead writer look alike to a Reader (G4): decision 47,
  the Reader never decides whether a writer is alive (see A5-6). A restored log whose head grows back past a stale consumer
  offset is not re-flagged after open (G23): inherent to an offset-only consumer, mitigated by the restore procedure in
  docs/userguide.md. `quill.lock` removed, or a lock held across a bind mount or a VM boundary (G21, G22): the lock is
  advisory; the bind-mount case was run in an Apple container (the second `Open` succeeded) and is documented unsupported
  (scope.md, "Filesystems").
- **Explicit operator or embedder action.** `NoMaintenance` set by mistake, `unregister` of a protected consumer, a listed
  consumer made unprotected while the writer runs, a configuration that disables expiry or sealing (G27, G29, G40, G35).
  A damaged segment is a hole for a writer that could not open it, until restart (G18, decision 94).
- **Documented non-guarantees and best effort.** The floor can be crossed between passes (G32, scope.md); `MaxRecords` worst
  case in bytes (G33, scope.md); `cp` and `rsync` reset the consumer mtime the TTL uses (G34, scope.md); the clock stepping
  back leaves a wrong `OffsetForTime` and `-since` clamps silently to the first record (G45, decision 59); bidirectional
  controls pass raw in JSON strings because the guarantee covers C0, DEL and C1 only (G49); `Reader.Stat` zero fields
  (A5-15, decision 66); Check output is one Problem per record for a clock anomaly (G28, a volume observation, not measured on a
  large log).
- **Unreachable or negligible.** `Reader.Stat().MarkerAge` zero when the `Lstat` fails (a race with the marker being
  replaced, G4); leaked `.tmp-*` files not counted in `Bytes` (bounded by the crash count, small files, G20); unlink frees no
  bytes while a reader or archiver holds the segment open (inferred, G32); a consumer file unreadable at pass time is not
  counted in `Dropped` (narrow, inferred, G30); an active segment stamped in the future ages when it seals (low impact, G28);
  `Log.Stat().Bytes` after an out-of-band delete (G36); a damaged segment that appears after Open under a live writer is not in
  the writer's `Stat` (G16: a read and `quill check` find it); copy order of a hot backup is not validated, and the failure is
  loud (G52); a Logger handler that blocks stalls the flusher (documented for the Logger).
- **Out of this pass's guarantees.** `Perm` tightened on reopen leaves existing files at the old mode while Check says
  healthy (G24, G42): noted by group 3 and not raised. macOS `fsync` falls back to plain `fsync` on ENOTSUP on mounts decision 51
  already declares unsupported (G50).
- **Presentation.** `quill check` on an Archive destination prints a policy warning and an UNBOUNDED line because no
  `quill.policy` is written there; it trains operators to ignore Check output but changes no state (G37, cosmetic).
- **Round-1 decisions not re-raised.** Archive exits 0 on a source gap and prints once: A1-58 (decisions 85 and 53) is the
  ruling for the exit code; this pass raised the destination-side silence and the `unregister` case separately (A5-29).

## The rule the pass produced

A state that degrades a guarantee is only handled when something can tell it from health. The pass found 25 places where
something was nearly there and was not enough; they fail the same four ways, so the rule has four clauses.

1. **Say "now", not "once".** A signal that latches (`Err()`), freezes (`StaleConsumers` between passes, `diskFull`) or
   reports the first of several (the first bad segment) cannot tell a current failure from a past one, and hides the next.
   Every degraded state needs a signal that is true while the state holds and clears when it ends, or a transition event for
   each direction. (A5-10, A5-1, A5-22.)
2. **Put it where the operator is.** A Logger that is nil by default, a `Stats` field that only the library process holds, and
   an `Err()` that nothing in the CLI reads are three places an operator is not. A state the CLI can cause or sees must reach
   the CLI's own exit status or stderr, and a state the library can enter must reach a pollable field, not only a log line.
   (A5-36, A5-18, A5-44, A5-9.)
3. **Zero means zero.** A field that reads 0 or empty for "not evaluated", "never ran" or "cannot know" is a statement of
   health. Use an absent value, or a separate flag. (A5-40, A5-15.)
4. **Loss of evidence is itself a loss.** An action that erases the only artifact of damage (the cut tail, a deleted damaged
   segment, an overwritten policy) must record what it erased in a place that outlives the action. (A5-2, A5-42, A5-39.)

Applied to new code: for each new background job, failure path or capability that can be switched off, name its row in the
ledger above and its mark, and do not mark it OK unless all four clauses hold for it. The recorder's reading of what would
enforce this, inferred and not built: one fault-injection test per ledger row that asserts a signal on the surface the
operator uses, which is what the seven A5 fixes each added for their own row.
