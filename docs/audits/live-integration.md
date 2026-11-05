# Live integration: passes L3 and L3b

Date 2026-11-03. The audited tree is `a63bd6b` (the multi-process harness landed on it); the four fixes of this pass sit on top of it, up to `ba30fee`. Line numbers below are from
the tree as it is now. Labels: **verified** (run, or the exact lines read, by the recorder or by an auditor named in the claim), **reported** (a document or an auditor says it; the source is named),
**inferred** (constructed from what is named). Companion files: [harness-inventory.md](harness-inventory.md) (the harnesses used here), [power-loss.md](power-loss.md) (the earlier
real power cuts), [untestable-without-x.md](untestable-without-x.md), and [stage2-owner-decisions.md](stage2-owner-decisions.md) (the two items first held for the owner, and the owner's answers).

**Update 2026-11-03 (pass L3c).** The two items this pass first held for the owner, L3b-7 and L3b-8, were answered the same day: the owner took the recommended answer for both. L3b-7 is documented (no code change; commit `7fa0744`) and L3b-8 is fixed in the two steps recommended, in that order (commit `29f5c6c`), each FIXED-VERIFIED by a fix-skeptic. Nothing is held for the owner from this pass any more. The Summary below, the two item sections and the "Items" dispositions are updated in place; the text as first written is kept where it states what was found. The owner's answers and the fix-skeptics' lens evidence are in [stage2-owner-decisions.md](stage2-owner-decisions.md), sections L3b-7 and L3b-8. Not re-run by the recorder: this update is a record of the fixes and of what the fix-skeptics reported; the recorder read the two commits' diffs and ran nothing.

## Summary

Twelve items came out of pass L3b after merging (L3b-1 and L3b-10 into L3b-4, L3b-5 into L3b-2).

| Verdict | Items | Disposition | Items |
|---|---|---|---|
| CONFIRMED | 6 (L3b-4, L3b-3, L3b-6, L3b-14, L3b-7, L3b-8) | fixed | 6 (L3b-4, L3b-3, L3b-6, L3b-14, L3b-8 by code, L3b-7 by documentation only), each FIXED-VERIFIED |
| REFUTED | 3 (L3b-9, L3b-13, L3b-15) | held for the owner | 0 (L3b-7 and L3b-8 were held until the owner's answers of 2026-11-03) |
| decided in round 1, not put to a vote | 3 (L3b-2, L3b-11, L3b-12) | deliberately not changed | 3 (L3b-2, L3b-11, L3b-12), reason in each item |
| PLAUSIBLE | 0 | refuted, nothing changed | 3 (L3b-9, L3b-13, L3b-15) |

- **Skeptic votes.** 21 votes on the 9 items that were voted on (two each on the six CONFIRMED items, three each on the three REFUTED ones): 15 CONFIRMED, 6 REFUTED. The three REFUTED items were split 2 to 1
  against the finding, each with one dissent that is recorded in the item. The three round-1 items were not voted on.
- **Fix-skeptics.** 6 of 6 fixes FIXED-VERIFIED; 40 lens verdicts (7 + 6 + 7 + 7 for the first four; 6 for L3b-7 and 7 for L3b-8, added 2026-11-03), all pass (reported by the fix-skeptics, who ran the tests, the gate's checks and the stress programs on copies of the tree; not re-run by the recorder). Each of the last two raised an objection it could not turn into a rejection; they are in the item sections.
- **Reproduced.** A voting skeptic reproduced 7 of the 9 voted items (L3b-4, L3b-6, L3b-14, L3b-8, L3b-9, L3b-13, L3b-15; 10 of the 21 votes carry a reproduction). L3b-3 and L3b-7 were judged by reading the code
  (L3b-7 was reproduced by the auditor on Linux, and again by its fix-skeptic on the Linux box at `7fa0744`; L3b-3 is read off the harness's own run summaries). The L3b-8 fix-skeptic reproduced the symptom on the parent and the fixed behaviour on a real tmpfs at zero free bytes (see the item). Of the three round-1 items, the auditors reproduced L3b-2 and L3b-11; L3b-12 is inferred, from the L3b-11 runs
  and not from a test of its own.
- **Severity as filed.** high 1 (L3b-11, a documentation item), medium 4 (L3b-8, L3b-9, L3b-12, L3b-13), low 7.
- **No defect in the log format, in recovery, or in the guarantees on a supported filesystem was found.** No acknowledged record was lost or altered, and no wrong payload was served, in any supported
  configuration run (counts under "What ran"). Every finding is a false verdict or a refusal on a healthy log (L3b-4, L3b-14, L3b-2), a harness defect (L3b-3, L3b-6), a refusal to open on a completely full disk
  (L3b-8, fixed 2026-11-03), or a statement the documents make about an unsupported or unusual configuration (L3b-7, documented 2026-11-03; L3b-9, L3b-11, L3b-12, L3b-13).

### Coverage statement

Four auditors returned and none failed to return: macOS on APFS, Linux on ext4, a Linux filesystem matrix (nine filesystems, power-cut replay), and Apple containers over virtiofs. A fifth party built and reviewed
the harness. What they checked and found clean:

- **One writer, many readers and consumers, killed at random.** About 1,800 writer restarts on macOS and about 1,100 delivered restarts on Linux ext4 across the soaks reported here; readers following across restarts
  never saw a skipped, reordered or altered record; every acknowledged record was present and contiguous at the end of every run; `quill check` was healthy before and after recovery (apart from L3b-4 and L3b-14).
- **Second writer.** About 550 probes (library `Open` and `quill append`) beside a live writer on macOS and Linux, and 78 CLI writer-verb invocations in a dedicated load run, were all refused with `ErrLocked` / `locked`.
  flock worked on every filesystem tried, overlayfs and tmpfs included.
- **Retention against lagging readers and consumers, caps against protection.** Protected consumers held records back against a one-second retention; `MaxBytes` passed a protected consumer and reported it as a gap, as
  documented; no silent skip anywhere; disk stayed bounded (largest `*.log` total 1,048,594 bytes against `MaxBytes` 65,536, which is one record of about 1 MiB in the active segment, inferred from the workload's size classes).
- **Archive racing rotation.** 1,314 archive runs in the harness and six 20-second runs of a dedicated rotation stress at 4 KiB segments: every destination byte-equal to the source and contiguous.
- **Roll, prune and Sync under load, many logs in one process.** Descriptors and goroutines returned to baseline after Close (macOS 4 to 454 to 4 for 150 logs; Linux 10,000 reader cycles and 2,000 Follow cancellations
  left 10 descriptors and 2 goroutines).
- **Full disk with live roles.** Real ENOSPC on nine filesystems (L3b-8 is what it found); no acknowledged record lost; the writer recovered when space came back. After the L3b-8 fix the fix-skeptic ran `hack/topology.sh run -fill` on a 16 MiB tmpfs for 90 s: ok, 725 writer restarts, 4 fills, no violations (reported by the fix-skeptic; the recorder did not re-run it).
- **Power loss.** One real cut on ext4 (`echo b > /proc/sysrq-trigger`), 240 dm-log-writes crash states over six journalling filesystems, 64 dm-flakey cycles: all verified, see below.
- **CLI under load.** About 14,000 invocations of the online verbs beside a rotating writer exited with the documented codes; the clean-room run in an Alpine container matched the user guide.

What was **not** covered is in the section "What the tier does not reach"; the short form is: no real power cut except one, no NFS/SMB/FUSE, no run over 540 seconds, one kernel, one Linux box, one macOS host.

## L3: why live third-party systems do not apply

Pass L3 asks what a library does against the live systems it talks to: a hosted database, a cloud API, a queue. quill talks to none. It is a Go library and a command line tool on the standard library alone
(no `require` line in `go.mod`; the gate checks it), it opens no socket, and its whole environment is a directory on a local filesystem and the operating system's file and lock calls. There is no
external service to authenticate to, rate-limit, version-skew against, or fail over. L3 therefore has no object here, and no test of that kind was written. (verified by the recorder: `GOWORK=off go list -deps ./cmd/quill` lists 126 packages and none beginning `net` or `crypto/tls`, and a grep of every `.go` file for an import of `net`, `net/http` or `net/rpc` finds none; reported: [supply-chain.md](supply-chain.md) says the binary has no network packages.)

What replaces it is L3b: quill's real environment is *other processes on the same machine* (a writer, readers, consumers, an operator running the CLI, a pruner, a crash) and *the filesystem under it*. Those are the
live systems the tier exercises.

## L3b: what the multi-process tier proves

The harness is `hack/topology` (see [harness-inventory.md](harness-inventory.md), section 6). One binary starts, as separate OS processes, one writer, readers that start at the beginning, at the tail and at random
offsets, consumers (one protected), an operator that runs the CLI in loops (`check`, `stat`, `read`, `archive`, `register`, `commit`, `unregister`, the writer-lock verbs), a pruner and a chaos controller. Chaos kills roles with
SIGKILL at random moments, freezes a protected consumer with SIGSTOP, starts a second writer to probe the lock, and under `-fill` fills the filesystem. Every record carries a run id, a sequence number and a checksum.
When the run ends, a verifier checks every acknowledged record, contiguity, payloads, `Check`, the consumer offsets (against an oracle of "intent N / done N" written around each commit), and every reader's observed spans
plus gaps against the final log.

What a pass of it proves, when run on a filesystem:

1. **Acknowledged means present.** A record the writer reported durable is in the log after any number of SIGKILLs of the writer, with the log contiguous from its first record and every payload intact.
2. **Readers never get a wrong answer from a healthy log.** Across writer restarts a reader's spans and gaps leave no record out, no reader sees a record above the final head, no consumer offset goes back or lies outside
   the interval its oracle allows, and a protected consumer is never passed except by a configured cap.
3. **One writer.** A second opener is refused while a live writer holds the lock, and a killed writer's lock is free for the next one at once.
4. **Retention and caps are honoured under concurrency** and a retention that removes a record a reader wanted is reported as a gap and never a silent skip.
5. **Online verbs and the library agree** with the log while it changes.

What it does not prove is the next-but-one section. Two points that bound the reading of any count below. First, the harness's `kills_<role>` counter counted *attempted* kills before the L3b-3 fix, so a reported "kills"
figure from a run before it can overstate the faults delivered by 20 to 35 percent (L3b-3); where a figure is a delivered count it is called "restarts". Second, scheduling is random and the verdicts are not: a run that finds nothing
is evidence for the interleavings it happened to take, not for all of them.

## Platform constraints found

| Platform | What it offers this tier | What it lacks |
|---|---|---|
| macOS 15-class host (Darwin 25.6, arm64, APFS), Go 1.27.3 | The product's own second platform: real `flock`, `File.Sync` issuing `F_FULLFSYNC`, `lsof` to count descriptors, real processes and signals | No `/proc`, no loop devices, no dm targets, no way to cut power or drop writes; a SIGKILL cannot lose page-cache data, so "unacknowledged and lost" is checked only at process level. `F_FULLFSYNC` was not exercised or timed. The host was shared (load average up to 17), so no throughput figure is comparable |
| Linux x86-64 box (Debian 13, kernel 6.12.113, KVM, 4 vCPU, ext4 on a virtio disk) | `/proc/<pid>/fd`, a real `sysrq` reboot, loop-device images, `mkfs` for ext4 variants, xfs, btrfs and f2fs (installed with the distribution's tools), the dm-flakey and dm-log-writes targets, tmpfs and overlayfs, a 4 MiB tmpfs for real ENOSPC | One kernel, one virtual disk: whether the disk's own cache honours `fsync` cannot be tested from a VM. dm-flakey and dm-log-writes model a volatile cache that is flushed in order; they do not reorder beyond what they log and do not tear below a block. The box is shared, so timing runs took a lock |
| Apple `container` (macOS 26.6, CLI 1.4.1, linux/arm64 VMs, kernel 6.18.15) | Two real lock domains: separate VMs sharing one directory over virtiofs, and a macOS host operating on the same directory a guest writes; an Alpine clean-room with no Go toolchain | virtiofs is not a supported filesystem (decision 51); a default 1 GB VM is killed by the kernel out of memory when several role processes hold the harness's largest records (inferred from `dmesg`; not a quill defect, quill's reader memory was not profiled); no power cut |
| Linux arm64 in a container (binary cross-compiled on the host) | The same code on a second architecture with the container's own ext4 | The container has no `sysrq` and no dm |

Constraints that shaped what was possible, stated once: a power cut can be done for real only on the Linux box (and only once, 40 seconds of downtime); *partial* power-cut models need device-mapper and so only Linux; the lock domain
across two kernels needs VMs and only the container tool gave it; and macOS can say nothing about the journal behaviour of any filesystem but APFS.

## What ran, per platform and filesystem

Counts of faults are the faults delivered, not attempted, where the source says so. "Restarts" is the delivered count of the harness; it is a lower bound (a kill sent to a child that died but was not yet reaped counts as one).

### macOS, APFS (verified by the auditor from the harness summaries; 20 harness runs, plus dedicated drivers)

| Scenario | Runs | Faults really injected | Result |
|---|---|---|---|
| Baseline, 60 s, default and `GOMAXPROCS=1` | 2 (plus 1 kept first run) | 22 writer restarts, 3-4 reader restarts, 3 second-writer probes per run; 455 and 421 `check`, 98 and 100 `archive` | ok; head 11,874 and 13,044, acked equals head. The kept first run failed once, on L3b-4 |
| Writer SIGKILL storms (three shapes, 90-100 s) | 6 | 235, 210, 201, 194, 177, 173 writer restarts (about 1,800 over all 20 runs, at most 2 per run were exit-75 retries) | ok: no hole, contiguous, no recovery moved the head back, `Check` healthy before and after |
| Reader and consumer SIGKILL | 2 | about 890 reader and 620 consumer restarts over all runs; 52-53 reader, 38-39 consumer restarts in these two | ok; a consumer's offset always between its last returned and last attempted commit; a killed consumer's lock was free on restart |
| Second writer | all | 353 harness probes, 78 CLI `append`, `prune`, `roll` invocations beside a live in-process writer | every one refused (`ErrLocked`, or exit 1 `locked by another holder`) |
| Retention racing lagging readers | 2 | 9,569 and 9,068 records first retained at head 9,576 and 9,077; gaps 454/313 (readers), 261/184 (consumers) | ok; no silent skip; 4 and 2 consumer `Commit` range errors are the documented unprotected case |
| Protected consumer, `MaxBytes` | 2 + 2 | 8 SIGSTOP freezes each; 23 and 17 consumer gaps when a cap passed the protected consumer | ok; retention kept 761 and 734 records where 160 would have fitted, as protection requires |
| Archive racing rotation | 12 (two concurrent archive loops, 4 KiB segments) | 12,609-92,373 records and 300-450 archives per run, plus 1,314 archives in the harness | destination byte-equal and contiguous every time |
| Roll, prune, Sync, Follow in one process | 4 lossless, 4 lossy | 736 and 1,235 rolls, 267 and 618 prunes (lossless); 13,324 and 58,321 records with gaps (lossy) | head equals appended, no silent skip, descriptors 4 to 4, goroutines 1 to 1 |
| 150 logs in one process | 2 | 58,407 and 10,762 records, 685 and 551 gaps | descriptors 4 to 454 to 4; 300 open/close cycles left 4 and 1 |
| CLI online verbs under load | 4 | about 7,000 invocations per run at each `GOMAXPROCS`; 1,087 and 610 lossy archives | only documented exit codes; the one anomaly was L3b-4 (reported as the check race) |
| Descriptor sampling | 1 (60 s) | no restarts; 629 reader and 357 consumer gaps | no role's descriptor count grew (writer 13-15, readers 11, consumers 13-14) |

### Linux, ext4 on the box (verified by the auditor; logs on the disk, not tmpfs)

| Scenario | Faults really injected | Result |
|---|---|---|
| Baseline, 180 s | 150 writer, 36 consumer, 39 reader kills (the counter then counted attempts), 26 probes refused | ok, 73,470 records contiguous, 467 `check` healthy |
| Writer SIGKILL soak, 540 s | 739 delivered writer restarts (1,127 kill signals counted, including to an already-dead role), 310 consumer and 415 reader kills | ok; 134,472 acknowledged records present; 797 checks healthy; 18,501 commits |
| Soak at a 60 ms kill interval, 150 s | 1,059 writer kills counted, 356 delivered restarts | ok, 14,455 commits, 0 violations |
| Soak at segment 8,192 with no pace, 120 s | not separately counted | ok |
| Retention against lagging and stalled protected consumers | 12 runs; up to 20 SIGSTOP stalls per run | all checks passed; the violations raised were L3b-4, L3b-2 and L3b-6 |
| Archive during rotation | 51-261 archives per run | no crash, no unexpected exit code |
| Different uid on a 0777/0666 log | 4 writer verbs as another user | refused with a bare permission error (L3b-7, documented 2026-11-03, behaviour unchanged); nothing modified |
| Full disk, 4 MiB tmpfs, three 120 s runs | 12-16 fills per run; 1,014 writer restarts in the run that filled the disk itself | ok; every acknowledged record survived; the writer recovered when space returned |
| Descriptor and goroutine leaks, in process | 10,000 reader cycles, 10,000 bare open/close, 2,000 Follow cancellations, 3,000 consumer cycles, 2,000 writer cycles, 100 refused second opens | 10 fds / 2 goroutines throughout; 8 / 1 after the writer closed. Counted in process, not by an external sampler |
| One real power cut | `sysrq` reboot after 90 s with no sync; about 118 s until ssh answered | `powertest verify` OK: 1,610,165 acknowledged, all present, contiguous, 88 segments, `Check` healthy. The marker was fresh, so the stale-marker case (decision 39) did not occur in this cycle |

### Linux filesystem matrix (verified by the auditor; loop images of 2 GB, 512 MB for the replays)

Topology scenarios base (40 s), kill (60 s, about 95 writer SIGKILLs plus reader and consumer kills), retention (40 s), caps (40 s) and full disk (60 s, image ballasted to about 6 MiB free), on ext4 in four modes
(`data=writeback`, `data=journal`, `nobarrier`, no journal), xfs, btrfs, f2fs, tmpfs, and overlayfs (upper and work on ext4, empty lower).

- Base, kill, retention and caps: **pass on all nine**, 0 violations.
- Full disk: pass on ext4 writeback, ext4 journal, btrfs and tmpfs. xfs, f2fs and overlay failed only at the verifier's final `Open` (L3b-8); ext4 `nobarrier`, ext4 without a journal and f2fs
  (caps) failed on L3b-4, and passed when re-run with `-no-unregister`. In every full-disk run the writer's flush failed with ENOSPC, no log was corrupted and every acknowledged record survived on ext4, btrfs,
  tmpfs and overlay (xfs and f2fs had clean checks from every role before the final open).
- **Crash replay with dm-log-writes** (a throwaway replayer, not in the repository: it was validated by reproducing the device byte for byte with the full log on ext4 in three modes, journal-less ext4, xfs and btrfs;
  on f2fs the full replay differed from the device and was not investigated, so f2fs results are weaker). Forty cuts per filesystem, half exactly at an acknowledgement mark: ext4 ordered, writeback and journal, xfs,
  btrfs, f2fs, **240 of 240 verify OK** with a read-only `fsck` clean. The marker was stale by 27 records on all but f2fs, and by up to about 18,000 on f2fs (rename persisted only at a checkpoint); parse-forward
  recovery (decision 39) recovered every time. Journal-less ext4 failed in this model: see L3b-9.
- **dm-flakey**: 8 `drop_writes` cycles on each of six filesystems (48 of 48 OK), 4 `error_writes` cycles on each of four (16 of 16 OK; the writer logged the `ErrFailed` message and exited, no acknowledged record lost).
  Journal-less ext4 lost the marker in 2 of 19 cycles (L3b-9).
- **Overlayfs over a lower layer holding a log** killed mid-run: the overlay mount, writer `Open`, `verify`, a `quill tail -follow` that spanned the copy-up (189,148 records delivered, equal to the last acknowledgement)
  all passed. One writer kill only, not a campaign.
- Single-run durable flush rates (1,500 `Append`+`Flush`, 100-byte records, noisy shared box): ext4 ordered 296 per second, writeback 523, journal 418, `nobarrier` 3,100, journal-less 964, xfs 621, btrfs 346, f2fs 862,
  tmpfs 29,960. tmpfs's fsync is a no-op, so nothing is crash-durable there (expected). The `nobarrier` rate is inferred to mean fsync does not flush the device cache; loss was not tested.
- What quill assumed that a filesystem here did not give: atomic rename on journal-less ext4 (L3b-9) and free space for the 44-byte marker temp file on a completely full filesystem (L3b-8). `O_NOFOLLOW`, unlink of open files and
  the `statfs` read behind `MinFreeBytes` raised nothing on any of them.

### Apple containers over virtiofs (verified by the auditor; one run per scenario, 60-90 s)

| Scenario | Faults really injected | Result |
|---|---|---|
| Baseline on the container's ext4, 60 s | 16 writer SIGKILLs | ok, 32,869 records verified, 897 segments |
| Retention and caps on ext4, 60 s (4 GB VM) | 15 writer kills | ok. The first try in a 1 GB VM ended in a kernel OOM kill of a role process |
| Explicit writer SIGKILL loop on ext4, `SyncAppends` | 40 kills at random 0.3-0.8 s | 1,701 acknowledged, 0 missing, 0 overwritten, contiguous, `check` healthy; 29 durable-but-unacknowledged records (kill between fsync and print), expected |
| Lock release after SIGKILL | 2 (ext4, virtiofs) | next writer opened 20 ms later on both |
| Roles in one VM on virtiofs, with a macOS host operator loop | 266 host iterations over 75 s, 604,218 host `Head()` calls | host side 0 errors, 0 head regressions. Guest side hit L3b-13; nothing lost (2,543 of 2,543 verified) |
| Container stop and start with two writers | one stop, plus a second SIGKILL round | 894 and 888 acknowledged present; 1,375 of 1,389 verified after the second round (the remainder are the durable-but-unacknowledged tail, inferred); a 37-byte garbage tail past the marker was cut at the next writable `Open` with a WARN |
| Two VMs on one mount: writer in one, reader in the other | all writer time | never a wrong, reordered or skipped record; `ErrNoLog` flakiness only (L3b-13) |
| Two writers in two VMs, and host plus guest | 2 abuse runs | **both opened; acknowledged records lost** (L3b-11, decided in round 1) |
| Same-VM second writer on virtiofs | 1 | refused with `ErrLocked` |
| Alpine clean-room, no Go | release-flag binary | `version`, `append`, `read`, `stat`, `tail`, `cat`, `check`, `register`, `commit`, `consumers`, `roll`, `recover` behaved as the user guide says |

## What the tier does not reach

Stated explicitly so a later pass does not read a pass here as coverage:

- **Power loss beyond one real cut and a model.** One real `sysrq` cycle (marker fresh, so the stale-marker path of decision 39 did not occur in that cycle; it occurred in the replays); 240 volatile-cache replay states and 64
  flakey cycles, all on one kernel in a VM. Disk write-cache honesty is untestable from a VM. dm-log-writes and dm-flakey do not reorder beyond what they log and do not tear below a block. The f2fs replayer did not reproduce
  the device byte for byte, so its 40 cuts are weaker. The sampling is small (40 cuts and 8 flakey cycles per filesystem; the replay's acknowledgement marks are sparse, so a mid-write cut is checked against a lagging
  acknowledgement). SIGKILL alone proves nothing about unsynced data.
- **Consumer offset-file durability under a power cut** (`powertest` exercises only the log) and **Archive under a crash**.
- **macOS durability**: no power cut, and `F_FULLFSYNC` neither exercised nor timed. No Linux-on-macOS-host or other host.
- **Unsupported filesystems**: NFS, SMB, FUSE (decision 51); Docker Desktop and Lima file sharing; btrfs with compression, `nodatacow` or several devices; xfs with non-default `mkfs` options; ext4 with other mount options than those
  listed; kernels other than 6.12.113 (and 6.18.15 in the containers); `MinFreeBytes` on btrfs and f2fs full disks except through the topology fill run.
- **ENOSPC at one chosen byte.** Fills were driven by a filler process and the harness; a failure at a chosen byte offset was not.
- **Duration.** The longest run was 540 seconds (Linux ext4), the longest on macOS 100 seconds. No soak of hours, no growth of a log past a few million records under concurrency, no clock change, no sleep and wake.
- **Platforms and architectures**: Windows (unsupported), linux/amd64 inside a container, a host-local volume shared by two containers (no virtiofs).
- **Hostile peers.** The roles are cooperative processes with the same uid; a process that writes garbage into the log, or a second uid with its own permissions, was covered only by the one different-uid probe (L3b-7).
- **Detection power of the chaos second-writer probe.** For the second-writer fault, the pruner's roll failure fires first, so the probe was not shown to be what catches a second writer in the harness
  (reported by the harness reviewer). The probe's refusals counted above are real, but a harness that regressed the lock might still pass by that other route. Library-level lock tests cover the lock itself.
- **`GOMAXPROCS=1` reaches the child roles** by environment inheritance (inferred from `exec.Command`; not observed per child).
- **Interleavings.** A random schedule with a seed: a clean run is evidence about the interleavings taken.
- **Reader memory with the harness's largest records** was not profiled (the one OOM in a small VM).
- **Cross-version skew under live traffic.** `hack/skew.sh` does the offline cross-version check; this tier ran one version.

## Items

Order: fixed, then held, then refuted, then decided in round 1. "Reported" means an auditor or skeptic says it; the fix-skeptic's lens evidence is reported, not re-run by the recorder, apart from the commit hashes (verified from
`git log`: `136577c`, `53fcb36`, `d88c9c1`, `ba30fee`).

### L3b-4: a consumer unregistered after the listing makes `check` UNHEALTHY, a prune pass skip its age sweep, and `Consumers()` list an ENOENT row (low, bug)

Merges L3b-1 and L3b-10 (the same race seen through the CLI and the harness).

- **Claim.** A consumer file removed by a legitimate unregister between the directory listing and the read made `Reader.Check` report a fatal problem (UNHEALTHY, `quill check` exit 3) on a healthy live log,
  made the writer's prune pass treat the vanished file as unreadable (warn and skip that pass's age sweep), and put an ENOENT row into `Consumers()` and `quill consumers`.
- **Grounds.** Before the fix `consumerStore.list` did `ReadDir`, then `readState` per name, and stored an ENOENT from after the listing in `info.Err`. `Check` turns any `c.Err` into a fatal problem
  (stat.go:307-308 now) and the prune pass's consumer view sets `broken` for any `c.Err` (retention.go:57-58), after which the age sweep keeps every segment.
- **Warrant.** The listing is not atomic with the reads. A file that vanished after `ReadDir` because its consumer was unregistered is a normal event, not damage; calling it damage is a false UNHEALTHY for a monitor
  that runs `quill check` beside an operator.
- **Backing.** Decision 101 and its `Check` addendum (a sealed segment that vanishes after the listing is skipped); decision 41 (consumer deletion is ordinary). The userguide describes `check` as reporting damage.
- **Rebuttal.** Needs a concurrent unregister: 0 of 247,581 `Check` calls failed in runs that were not racing deletes. The prune effect is one skipped pass and a Warn, which heals on the next. A file that is truly
  unreadable still has to be reported, so the skip applies to `fs.ErrNotExist` only.
- **Verdict.** CONFIRMED, 2 of 2 votes.
- **Reproduction.** (verified) Program: one goroutine loops `Reader.Consumer("op", ProtectOn)`, `Close`, `DeleteConsumer`; two loop `Reader.Check` and `Reader.Consumers`; one loops `Log.Prune`. Linux ext4 (auditor):
  247,581 checks, 545 UNHEALTHY for the vanished consumer, 500 `Consumers()` rows with ENOENT, 973 prune warnings (a second run: 584 and 1,038); macOS 15 s: 79 and 153. The two skeptics reproduced it on their own programs
  (148,844 calls, 128 ENOENT rows; 88 and 99 false UNHEALTHY, 71 and 83 ENOENT rows, 182 prune Warns). Independently, a topology run on Linux got `quill check` exit 3 with a fatal consumer ENOENT while the operator was
  unregistering; 2 of 4 runs with retention and `MaxBytes` failed that way.
- **Disposition.** Fixed in `136577c`; fix-skeptic FIXED-VERIFIED, 7 of 7 lenses pass. `consumerStore.list` skips a name whose read fails with `fs.ErrNotExist` (consumer.go:302); any other error is still stored in `info.Err`.
  Decision 41 has a dated note (decisions.md:49); CHANGELOG has a line. Test: `consumer_vanished_test.go` (`TestAConsumerFileRemovedAfterTheListingIsNotDamageNorAConsumer`, four subtests; without the fix three fail and the
  fourth, an EIO that must still be reported, passes both ways). On Linux the stress program gave 303 UNHEALTHY and 310 error rows before and 0 and 0 after (8 s, 112,811 checks).
- **Residuals** (reported by the fix-skeptic). With `ConsumerTTL > 0`, a protected, behind consumer that vanishes between `list` and the staleness `mtime` read (retention.go:68) gives one spurious "staleness probe failed"
  Warn; the age sweep is not skipped, so it is cosmetic. macOS was not re-run after the fix. The harness's `-no-unregister` help text (hack/topology/run.go:107) and the operator comment at hack/topology/main.go:730-732 still describe the race as live;
  the flag still works, but the text predates the fix.

### L3b-3: the harness's `kills_<role>` counted attempted kills, not delivered ones (low, hygiene)

- **Claim.** `chaos.kills_<role>` counted every attempt, so a summary overstated the faults delivered. `restarts_<role>` is the delivered count (it also includes exit-75 retries).
- **Grounds.** `kill` called `p.signal(SIGKILL, true)` and added to the counter unconditionally, while `signal` sends only when the child is up (hack/topology/run.go:159). Observed: writer kills against restarts of 313
  against 235 (default), 316 against 210 (`GOMAXPROCS=1`), 331 against 201 and 321 against 194.
- **Warrant.** A kill sent between a child's exit and its "ready" line is dropped but was counted, so 20 to 35 percent of the counted writer kills never happened. A claim that recovery was exercised N times must use restarts.
- **Backing.** A fault must be established before a claim is made about recovery from it; the summary is what that rule reads.
- **Rebuttal.** Reported figure only; no check or verdict reads it (the test at main_test.go asserts `!= 0`).
- **Verdict.** CONFIRMED, 2 of 2 votes, both by reading the code. Neither reproduced it; the ratio follows from the mechanism.
- **Reproduction.** (reported) the counts above, from the auditor's run summaries; the skeptics traced the path by hand.
- **Disposition.** Fixed in `53fcb36`; fix-skeptic FIXED-VERIFIED, 6 of 6 lenses pass. `proc.signal` returns whether it sent; `orch.kill` (hack/topology/run.go:321) adds to the counter only then. Test:
  `TestKillCountsOnlySentSignals` in hack/topology/main_test.go (a real child; fails, by mutation, with "kills_writer = 1 after a kill of a child that is not ready, want 0"). On a Linux run of the fixed commit, `kills_writer` 62
  equalled `restarts_writer` 62, `kills_reader` 30 and `kills_consumer` 32 equalled their restarts.
- **Residual.** A SIGKILL to a child that died but is not yet reaped still counts as sent, and two kills before one reap give one restart, so restarts is a lower bound. The runs reported in this file before the fix
  are labelled accordingly.

### L3b-6: the topology stall scheduler spins at full CPU after a run is cancelled (low, bug in the harness)

- **Claim.** After a violation cancels a run with `-retention` set, `stalls()` busy-spun at 100 percent CPU until the original deadline, and the 30 s watchdog then added a spurious second violation ("the roles did not stop 30s after SIGTERM").
- **Grounds.** hack/topology/run.go:390 (stretch loop; before the fix `for {`): on a cancelled context `sleep` and `o.stall` return at once, so the loop ran only until the wall-clock passed its limit.
- **Warrant.** A loop whose exits depend on wall-clock time and not on the context never stops on cancellation.
- **Backing.** The harness's contract is that the run stops at the first violation.
- **Rebuttal.** No effect on a passing run, none without `-retention`; the first violation is still reported correctly; quill is unaffected.
- **Verdict.** CONFIRMED, 2 of 2 votes.
- **Reproduction.** (verified by a skeptic, macOS) `topology run -retention 2s -duration 100s -fault skip -seed 31`: cancel at about 1 s, the orchestrator above 60 percent CPU for 44 s or more, exit after 79 s, stderr ends with the
  extra watchdog violation, SIGQUIT shows `stalls` at the loop. The auditor's Linux goroutine dump (reported) showed it `[runnable]` in `stalls` 15 s after the violation.
- **Disposition.** Fixed in `d88c9c1`; fix-skeptic FIXED-VERIFIED, 7 of 7 lenses pass. The loop header became `for o.ctx.Err() == nil {` (hack/topology/run.go:390). Test: `TestStallsReturnsWhenRunIsCancelled` (fails without the fix:
  "stalls was still running 2s after the run was cancelled"). A separate test returned 17 microseconds after cancel on macOS and 32 on Linux, against 3.12 s of CPU in 3 s before.
- **Residual.** No end-to-end run with a real violation and `-retention` after the fix; the cause is covered at unit level.

### L3b-14: `quill check` on a live log warns "ahead of the head" for a consumer that committed during the scan (low, bug)

- **Claim.** `Reader.Check` read `quill.tail` first, scanned, then read the consumers, so a consumer that committed to the grown head in between looked like a restored-log leftover and drew "at N, ahead of the head M: rewind it with quill commit".
- **Grounds.** The tail snapshot was taken at the start (stat.go:186), consumers after the scan (stat.go:291), and the warning fired on `c.Offset > tl.Seq+1`.
- **Warrant.** A consumer's offset is at most the head at its commit time plus one; the offset exceeds the stale snapshot only because the snapshot is older. The operator is told to rewind a healthy consumer; following the advice would make it re-read records.
- **Backing.** Decision 36 (the warning is for a log restored to an older state); the guarantee that `check` is lock-free beside a live writer.
- **Rebuttal.** A warning only: exit 0, never UNHEALTHY; needs the log to grow during the scan; the harness's operator never saw it on ext4 in 421 checks because those scans are short. A genuinely ahead consumer is still reported.
- **Verdict.** CONFIRMED, 2 of 2 votes.
- **Reproduction.** (reported by the auditor; reproduced by one skeptic) container ext4, a 2.4-million-record log with a writer, a loop of `quill commit -offset HEAD+1`, 15 `quill check` runs: 9 of 15 printed the warning. The skeptic's program on
  macOS (1.5 M records, writer plus committer): 2 of 10 runs, Healthy stayed true. A skeptic's Linux stress on the parent tree: 15 of 15 checks warned.
- **Disposition.** Fixed in `ba30fee`; fix-skeptic FIXED-VERIFIED, 7 of 7 lenses pass. When an offset exceeds the head read at the start, `Check` re-reads the marker once, after the consumer list was read, and warns only if the offset is above that
  fresher head, which the message names (stat.go:295-311). Decision 36 has a dated paragraph; CHANGELOG has a line. Test: `check_live_consumer_test.go` (two subtests; both fail without the fix). Linux stress after the fix: 0 of 15.
- **Residual** (reported by the fix-skeptic). A consumer taken from the writer's own `Log` could commit to `head+1` while a failed marker publish leaves `quill.tail` behind; `Check` would still warn after its single refresh. Pre-existing and separate.

### L3b-7: a second writer under another uid gets a bare permission error, not `ErrLocked` (low, doc, documented 2026-11-03)

- **Claim.** On a log created by one uid, a writer or a writer-lock verb under a different uid fails with `open .../quill.lock: permission denied`, not `ErrLocked`. `doc.go` and the user guide say a second `Open` and the writer verbs fail with `ErrLocked`; only the `archive` verb documents the combined error.
- **Grounds.** `takeWriterLock` opens the lock `O_RDWR|O_CREATE` (lock_unix.go:42) and returns the error unwrapped; `ErrLocked` is added only after a flock conflict. The lock is always 0600 (decision 51). Docs: userguide.md:249 (`no: ErrLocked`), doc.go:10, userguide.md:343 (archive only).
- **Warrant.** A caller testing `errors.Is(err, ErrLocked)` to detect a live writer gets false for another uid, and `quill` exits 1, the code of any failure, with a message that does not say a writer may be live.
- **Backing.** The single-writer guarantee and the `ErrLocked` contract (errors.go).
- **Rebuttal.** Intended by decision 51 (a reader-only user must not take the lock); nothing is damaged; a second writer under another uid is unusual; the harness probe uses the same uid.
- **Verdict.** CONFIRMED, 2 of 2 votes (both by reading; the auditor's run is the one observation).
- **Reproduction.** (verified by the auditor, Linux ext4) log created by uid A with umask 000 and `Perm` 0666, writer live, `sudo -u nobody quill append|prune|roll|recover`: all four print `quill: open .../quill.lock: permission denied`, exit 1; as the owner they print `locked by another holder`. `stat`, `check`, `read` and `register` as the other uid worked.
- **Disposition.** First held for the owner, recommended answer: document it. The owner took that answer on 2026-11-03 (the runner-up, wrapping the error as `Archive` does, was not approved). **Fixed by documentation only in `7fa0744`; fix-skeptic FIXED-VERIFIED, 6 of 6 lenses pass (reported).** No behaviour, exported error class or CLI message changed. The error stays a plain permission error naming `quill.lock`. The commit says so in the `Open` and package godoc (`log.go`, `doc.go`), the user guide table and a paragraph under it, scope guarantee 5 and the Locks paragraph, decision 51 (dated note "Stage 2 audit, owner-approved 2026-11-03, L3b-7") and a `Changed:` line in the CHANGELOG. It states that root gets `ErrLocked`, that `prune` without retention flags stops earlier on an unreadable `quill.policy` (the fix-skeptic's objection to the first text, which claimed the permission error for every verb), that a script detecting a live writer treats both as "not this account's log, or a writer may be live", and that `Archive` differs because it only refuses. The reason is stated as inferred, as in the owner-decisions section.
- **Pinning test.** `open_lock_perm_test.go`, `TestOpenUnderAnAccountThatCannotOpenTheLockIsAPermissionErrorNotErrLocked`: through the `fsx.Sim` `Inject` seam (no root needed) the open of `quill.lock` fails with EACCES while a first writer holds the log; the second `Open` must not match `ErrLocked`, must match `fs.ErrPermission` and name `quill.lock`, and with the lock openable again a second `Open` must return `ErrLocked`. It cannot be red on the parent, because the commit changes no code; the fix-skeptic mutated `takeWriterLock` to wrap the error with `ErrLocked` (the rejected runner-up) and the test failed with `error matches ErrLocked` (reported).
- **Fix-skeptic evidence** (reported). On the Linux box (Debian, ext4) at `7fa0744`, a holder with `Perm` 0666 under umask 000: as the same uid `quill append` printed `locked by another holder`; as uid `nobody` `roll`, `recover`, `prune`, `prune -retention 1h` and `prune -max-bytes` printed `quill: open .../quill.lock: permission denied`, exit 1, and `stat` worked; as root `append` gave `ErrLocked`. The `append` run as `nobody` exited 2 because of an argument-order slip in the skeptic's script, so `append` was not exercised as another uid there (the `Open` path is the one the other three use). The unreadable-`quill.policy` `prune` case was confirmed from the code (`cmd/quill/write.go:213-222`), not run. `gofmt`, `go vet` and `go test -race -count=1 ./...` clean; no golden or testdata file in the diff; `go doc -all .` differs only in the added paragraphs.
- **Residual** (reported by the fix-skeptic; not a defect of the fix). `docs/reference.md:100, 170, 186, 207` (generated from `cmd/quill/verbs.go:265`) still read "fails with ErrLocked beside a live writer" with no mention of the other-account case. The sentence is true for an account that can open the lock, and the approved answer forbids changing CLI text, so it was left. Recorded; not held.

### L3b-8: a completely full filesystem makes every writable open of an existing log fail with ENOSPC (medium, degradation, fixed 2026-11-03)

- **Claim.** With zero free bytes, `Open` of an existing log fails, and so do the CLI verbs `prune`, `recover`, `roll` and `append`, before retention, `MinFreeBytes` and `MaxBytes` can run; the log stays unopenable until something outside quill frees a block. The same condition in a running writer is retried by design (decision 60 d).
- **Grounds.** `load` sets `l.pendingPublish = true` unconditionally (load.go:456) and returns the error of `publishLocked()`; `flush` then rewrites `quill.tail` through a temp file, fsync and rename (flush.go:179-181), which needs a free block.
- **Warrant.** A 44-byte rename-based rewrite cannot allocate its temp file at zero free blocks, so `Open` returns ENOSPC. With one free block everything works (step 4 below), so the republish write is the sole cause, and it is avoidable because the marker is rewritten even when it is already correct.
- **Backing.** scope.md guarantee 11 and `MinFreeBytes` ("backstop for a shared disk"); the CHANGELOG entry that a prune pass must not be skipped exactly when the disk is full; decision 60 d and userguide.md:48-60.
- **Rebuttal.** Needs free space of exactly zero for the unprivileged user; a restart is needed (a live writer retries); an operator can free space outside quill; no document promises `Open` works on a full disk; creating a new log legitimately needs space. btrfs and f2fs were not tested with a standalone program.
- **Verdict.** CONFIRMED, 2 of 2 votes.
- **Reproduction.** (verified by a skeptic on the Linux box, 8 MiB tmpfs; also by the auditor on a 16 MiB ext4 image) program: open with 64 KiB segments, `MinFreeBytes` 2 MiB, append 3,072 records of 1 KiB, close (head 3,072, 49 segments); fill the rest with `dd` so `df` shows 0 available; `Open` gives `write .../.tmp-...: no space left on device`, exit 3; `quill prune` and `quill recover` exit 1 with the same error; truncate the filler by one 4 KiB block and `Open` succeeds and within 2-3 s the `MinFreeBytes` sweep takes the log from 49 to 17 segments (3,191,681 to 1,063,936 bytes). A second skeptic reproduced the code path with an injected ENOSPC on the marker's temp file. The topology full-disk scenario hit it on its own on xfs, f2fs and overlayfs ("Open after the run: open .../.tmp-...: no space left on device").
- **Disposition.** First held for the owner with no fix written; recommended answer: fix it in two steps, in order. The owner took that answer on 2026-11-03. **Fixed in `29f5c6c`; fix-skeptic FIXED-VERIFIED, 7 of 7 lenses pass (reported).** The first attempt was rejected on one objection (see below), and `29f5c6c` is the corrected commit.
  - *Step 1.* `load()` keeps the marker it decoded (`l.marker, l.markerAt = tl, clock()`, `load.go`), so `publishLocked` writes `quill.tail` at `Open` only when recovery found a different position. A cleanly closed log opens on a full disk with no write. `MarkerAge` counts from `Open` (the empty-log case the old comment at `load.go:451-455` named: the marker is the zero value and the age is set without a write).
  - *Step 2.* When the publish is needed and fails, `Open` returns the log with the publish pending (an unexported `openUnpublished` flag set in `load()`, cleared at the next successful marker write): `Err()` wraps the cause, `Config.Logger` gets the `marker publish` Warn, a `Reader` sees the older head, and the flusher, `Close` and prune passes retry (decision 60 d). A failed log still fails `Open`.
  - *Beyond the recommended text, and why.* The fix-skeptic's first objection was that a prune pass stops at the publish that precedes its unlinks, so with the `Open`-time publish pending nothing was freed and the stated purpose of step 2 was not met. `deleteLocked` now, while that publish is pending, unlinks the chosen sealed segments below the segment the marker on disk names and keeps that segment and every later one (`retention.go`; decision 65 note). The marker is a lower bound that recovery parses forward from (decision 39), and the directory sync before each unlink makes the unlink durable first. A publish that fails in a running writer still stops the pass (decision 60 d unchanged; the flag exists for that). This clamp is an addition to the owner's approved text, made to carry out its stated aim; it is flagged here so the owner can object.
  - *Documents.* Decisions 39, 60, 65 and 81 carry dated notes "Stage 2 audit, owner-approved 2026-11-03, L3b-8"; scope, user guide (full-disk paragraph near `userguide.md:48-60`), architecture, `docs/reference.md`, the `recover` help text, the `Open` and `MarkerAge` godoc and a `Changed:` CHANGELOG line are updated. No exported symbol, flag, file or config field was added; no golden file changed.
  - *Tests.* `open_fulldisk_test.go`, eight tests, written first: a clean log opens with no free block and the sweep runs; `MarkerAge` of a restarted log with records counts from `Open`; a stale marker on a full disk opens with the publish pending, a `Reader` sees the older head, no record lost or duplicated; `Close` publishes the marker an `Open` could not; the `MinFreeBytes` sweep runs while the publish is pending and keeps the marker's segment; a crash after a sweep under a pending publish loses nothing; a crash at every step of such a sweep reopens whole; a publish failing after the `Open`-time one landed still stops a prune pass. Fix-skeptic: with the five source files reverted to the parent and the test file kept, 7 of the 8 fail with `open: no space left on device` from `Open`; the eighth pins preserved behaviour and passes on the parent by design. Mutating `>=` to `>` in the clamp fails the sweep test; dropping the `!l.openUnpublished` guard fails the two decision-60 d tests (reported).
  - *Fix-skeptic evidence* (reported). Real 8 MiB tmpfs at 0 available bytes on the Linux box. Parent: `Open` fails, `recover` and `roll` exit 1. Fixed: (A) clean log opens, `recover` and `roll` exit 0, `prune` took 49 to 17 segments, `check` HEALTHY, `append` exits 1 with a clear ENOSPC message; a writer with `PruneInterval` 1 s freed 49 to 17 in 3 s. (B) with a stale marker (seq 2500, true head 3072), a maintenance writer took 48 to 16 and the marker reached seq 3072 after space returned; a `Reader` and `check` beside the pending writer saw head 2500, HEALTHY; `recover` exits 1 with a clear ENOSPC message. (C) torn tail plus stale marker: `recover` cut the 18 garbage bytes before failing on the marker write, `prune` then went 48 to 17, and `recover` after space returned exited 0. `FUZZTIME=20s bash hack/gate.sh` exited 0 (gofmt, vet on linux and darwin, race tests, staticcheck, deadcode, every fuzz target, `go mod tidy`); `go test -race -count=1 ./...` passed on macOS and on the box. `hack/topology.sh run -fill`, 16 MiB tmpfs, 90 s: ok, 725 writer restarts, 4 fills, no violations. The fixer's own live check reported `quill prune` 49 to 25 segments on a stale-marker tmpfs and ran `-fill` only on the earlier attempt; the fix-skeptic's numbers above are from its own runs on the final commit (the two differ in detail; both are reported, neither re-run by the recorder).
  - *Residuals* (reported by the fix-skeptic; none a regression, none held). (a) With the active segment over `SegmentBytes` and segment creation failing (a rotation due at `Open`), a prune pass frees nothing (sealed 6 to 6); a running writer in the same state behaves the same, so the restart path is at parity with decision 60 d. The same symptom is reachable on a full disk after a failed rotation, outside the L3b-8 claim. (b) Only the sweep test guards the "keep the marker's own segment" rule; the two crash tests did not fail when it was removed. (c) The scope `Config.Logger` paragraph's list of `Open`-time warnings omits the marker-publish Warn. (d) `docs/format.md:241` and `docs/design/build-prompt.md:133` still say the next open "republishes" the marker unconditionally; true when the marker is behind, and the build prompt is a historical document. (e) `MarkerAge` counts from `Open` while a publish is pending, while `Err()` reports it. (f) After a clamped pass `Stat().DiskFull` can read false (noted in decision 65).

### L3b-9: journal-less ext4 loses `quill.tail` after a power cut, though ext4 is listed as supported (medium as filed, REFUTED)

- **Claim.** The supported-filesystem statement (scope.md:350, decision 51) is not true for ext4 made with `mkfs.ext4 -O ^has_journal`: after a power cut the marker is lost in a large fraction of crash states and the log is refused until the marker is rebuilt by hand; a smaller fraction lose acknowledged data.
- **Grounds.** Two models on the Linux box: dm-log-writes replay of `hack/powertest` (60 cuts: 27 fail verify, 26 with "not a log: ... has no quill.tail", 1 with a damaged segment; a skeptic's run: 20 of 40 fail, 18 and 2) and dm-flakey `drop_writes` (2 marker losses in 19 cycles). Controls on ext4 `data=ordered`, `data=writeback`, `data=journal`, xfs, btrfs and f2fs: 0 failures. `quill` reacts by failing closed (decisions 62, 81; `open_missing_state_test.go`).
- **Warrant.** The marker is replaced by temp file, fsync and rename with no per-flush directory fsync (decision 40), which is safe only if rename is crash-atomic. Without a journal, rename is a plain directory update and the entry can point at a freed inode.
- **Backing.** scope.md:350 and decision 51 require "a local POSIX filesystem with working `flock`, atomic `rename` and honest `fsync`"; [power-loss.md](power-loss.md) and [untestable-without-x.md](untestable-without-x.md) list journal modes as not exercised.
- **Rebuttal.** The same sentence requires atomic rename, which journal-less ext4 does not give, so quill is within its contract and fails closed with no silent loss in the marker states; the data is intact and the documented rebuild procedure (userguide.md:420-424) recovers. The one damaged-segment state may be `e2fsck` repair collateral (it cleared an extent); not run down. No code defect is claimed.
- **Verdict.** REFUTED, 2 of 3 votes. The reason: the support statement is conditional (it names ext4 as an example of a filesystem that meets the stated qualifiers, not as a promise for every configuration), and the finding's own evidence puts journal-less ext4 outside it. The dissent reproduced the failure (20 of 40 states fail without a journal, 0 of 40 with one) and agreed there is no code defect, and judged an unconditional "ext4" in the documents a gap; that part is not disputed.
- **Reproduction.** (verified by the auditor and by the dissenting skeptic, Linux 6.12.113, dm-log-writes, 512 MB image, `e2fsck -fy` before mounting) results above. The third skeptic did not rerun it.
- **Disposition.** Refuted: nothing changed. An optional footnote that journal-less ext4 is outside the contract, and a Warn at `Open` on a filesystem known to have no journal, were offered by the auditor and are not held as owner items; an automatic marker rebuild is `quill repair`, deferred by decision 64.

### L3b-13: on virtiofs a live `Reader` intermittently gets `ErrNoLog` from a healthy log (medium as filed, REFUTED)

- **Claim.** On a virtiofs bind mount, a live `Reader`, the read verbs and `check` fail against a healthy log with "not a log: ... has no quill.tail" because `Reader.readTail` returns `ErrNoLog` at the first ENOENT, with no retry, while the writer's rename onto `quill.tail` is in flight. Related: `check` UNHEALTHY on consumer files being replaced, and a "readdirent" ENOENT.
- **Grounds.** reader.go:156-157 returns `ErrNoLog` at the first ENOENT, while the loop around it retries every other error. Auditor: same-VM about 1.2 percent of `Head()` calls on virtiofs against 0 of 14.3 million on the container's ext4; two VMs 93 to 99 percent while a writer is active; a scratch retry cut the same-VM rate to zero but still left 13 percent across VMs and four other ENOENT violations.
- **Warrant.** A file atomically replaced must not be reported missing; the loop's retry shows the intent to tolerate a racing replace.
- **Backing.** Readers take no lock and see records up to `quill.tail`; reader.go:82-84 reserves `ErrNoLog` for a reader that arrives mid-creation.
- **Rebuttal.** Bind mounts are "at the operator's risk" (decision 51, userguide.md:364). The macOS host reading the same directory saw 0 errors in 604,218 `Head()` calls; no data was lost or wrongly served in any run; a short retry only mitigates the cross-VM case. The ENOENT comes from the virtiofs guest kernel, not from quill's logic.
- **Verdict.** REFUTED, 2 of 3 votes. The reason: `rename(2)` never makes its target briefly vanish on a filesystem that meets the atomic-rename requirement, so ENOENT from a rename-over is a failure of the filesystem and virtiofs is outside the contract; a blanket retry would also delay the report of a genuinely missing marker. The dissent reproduced it (97,044 `Head()` calls, 11 errors on virtiofs, 0 of 10.99 million on ext4; a lower rate than the 1.2 percent filed, so the filed rates are unverified) and called a bounded retry a cheap robustness gain; it did not reproduce the CLI symptoms (0 of 175 `stat`, 0 of 175 `check`).
- **Reproduction.** (verified by the auditor and the dissent; the rates differ with writer cadence.)
- **Disposition.** Refuted as a quill defect: nothing changed. That virtiofs breaks rename visibility is inferred from the symptom, not shown by a trace. The related documentation gap is L3b-11.

### L3b-15: `quill --version` and `--help` are usage errors (low, hygiene, REFUTED)

- **Claim.** `quill --version` and `-version` exit 2 with "flag provided but not defined"; `quill --help` and `-h` print only "quill: flag: help requested" and exit 2; the version is available only as the verb `quill version`.
- **Grounds.** (verified) a clean-room Alpine run of a release-flag binary and the host build; verbs.go:197 handles `flag.ErrHelp` for verbs, not for the global flags.
- **Warrant.** A script probing `--version` gets a usage error that looks like a broken install.
- **Backing.** No document promises `--version` or `--help`; the user guide names only the `version` and `help` verbs.
- **Rebuttal.** By design the CLI takes verbs; both verbs work.
- **Verdict.** REFUTED, 2 of 3 votes: a request for new CLI surface, not a divergence from a document. The dissent confirmed the behaviour (as did all three) and noted that an internal audit instruction expects `quill --version` to work.
- **Reproduction.** (verified by two skeptics and the auditor) `quill --version`, `-version`, `--help`, `-h` exit 2 with the messages above; `quill version` and `quill help` exit 0.
- **Disposition.** Deliberately not changed. The bare "flag: help requested" with exit 2 on `-h` is the one wart, with no documented expectation; accepting `--version` and printing usage for `-h` would add CLI surface and is the owner's call if wanted (not held as an item).
- **Update 2026-11-04 (L5-1, owner-approved).** `quill -h` and `quill --help` (also `-help`) before a verb now print the usage of `quill help` and exit 0 (`cmd/quill/verbs.go`, `dispatch`; test `TestGlobalHelpFlagPrintsTheUsageOfTheHelpVerbAndExitsZero`, which failed on the earlier code with exit 2 and `flag: help requested`). `quill --version` is **unchanged**: still a usage error, exit 2.

### L3b-2: the marker's active segment can be gone when a reader opens it, giving a raw `lstat ENOENT` (low, bug, decided in round 1)

Merges L3b-5 (the same cause seen by `Reader.Check`).

- **Claim.** The reader reads the marker, then opens the segment it names. If the writer rotates and retention deletes that segment before the open, `Read`, `Commit` and the other view-taking calls return a raw `*fs.PathError` and `Reader.Check` reports a fatal "segment ...: no such file or directory" on a healthy log. Neither is a `GapError`; a retry succeeds.
- **Grounds.** `view` reads the marker (reader.go:342), lists, and opens segment `tl.Seg` (reader.go:367), returning any error as it is. `Check` excuses ENOENT only for `id < tl.Seg` (stat.go:227). Observed at `GOMAXPROCS=1` only: 40 UNHEALTHY results in 30 s with 4 KiB segments and `MaxBytes` 8,192 (every next check healthy; 45 at 16 KiB and 32 KiB; 0 at 64 KiB and 1 MiB); 14-15 raw errors in 25 s with 150 logs on one process, each with the marker one segment later on an immediate re-read (a scratch-only print).
- **Warrant.** The marker is published after rotation; a reader that read it just before is naming a segment that is now sealed and may be deleted at once. The reader never re-reads the marker to find out.
- **Backing.** Guarantee 14 and decision 79 (an I/O error is not a gap; the read can be retried); decision 101 (a sealed segment gone after the listing is a gap). The last sentence of decision 101 ("the marker's own (active) segment missing stays an error") is the only text that licenses it.
- **Rebuttal.** Decision 101 says in words that this stays an error, and it is retriable, not a wrong answer or a loss. It needs rotation plus deletion of that very segment inside one call, and appeared only at `GOMAXPROCS=1` under heavy contention; it did not appear in 20 topology runs, including six with 173 to 235 writer kills and retention of 1 to 2 s. A consumer loop that treats any non-gap error as fatal would exit.
- **Verdict.** Not voted: a prior deliberate decision (decision 101 last sentence and its `Check` addendum; audit-followups.md correctness-27).
- **Reproduction.** (verified by the auditor) the counts above; not re-run.
- **Disposition.** Deliberately not changed: decision 101 chose this, and the pass found nothing that changes its premise, since the failure is retriable and needs the one-in-thousands contention. The auditor's proposed fix (re-read the marker once when the active segment is missing and restart the view if `Seg` advanced; the same re-read in `Check`; amend decision 101) remains available and is not applied. Inferred: a quiet log with `SegmentMaxAge` close to `Retention` can seal an already-expired segment, so it is not impossible with realistic sizes.

### L3b-11: two writers across lock domains on a shared directory both open and destroy each other's records (high as filed, doc, decided in round 1)

- **Claim.** Two writers on one directory shared across lock domains (two VMs, or a container and the macOS host, over virtiofs) both open without `ErrLocked` and silently destroy each other's acknowledged records. The documents say only "bind mounts ... at the operator's risk" while guarantee 5 says a second writable open "fails fast with `ErrLocked`" without a qualifier; the CLI writer verbs run from the host are not stopped either.
- **Grounds.** (verified by the auditor) same-VM control refused. Two VMs on the mount, writer A started 1.5 s before B: B printed `OPEN head=79` and 400 acknowledgements, A 400, both closed cleanly; afterwards 800 acknowledged, 647 missing, 37 overwritten by the other writer's valid frames; `quill.tail` says head 479, `quill read` stops at 116 with a bad frame, `quill check` UNHEALTHY (exit 3), `quill recover` and a writable `Open` refuse. Host and guest: a container writer, then `quill append` from the macOS host printed 3 and exited 0; the guest run then had 300 acknowledged, 220 missing; `quill roll` from the host also exited 0. Docs: userguide.md:363-365, scope.md:350-351, scope.md:375 (guarantee 5), userguide.md:249; README.md and docs/introduction.md carry no filesystem statement.
- **Warrant.** The lock is the only writer exclusion (lock_unix.go); flock on virtiofs is not coherent between guest kernels or between guest and host, so both opens succeed. The loss is silent at write time, detected only afterwards by CRC and marker checks.
- **Backing.** Guarantees 3-5; decision 51 ("a log on a network filesystem can lose locks silently").
- **Rebuttal.** The documents say bind mounts and network filesystems are unsupported or at the operator's risk and decision 51 names silent lock loss, so this is under-promising rather than a breach. The probe uses synchronous appends and a 5 ms pace, so the overlap is dense; the loss (220 to 684 of 300 to 800) depends on timing. Not tested: a host-local shared volume across two containers, NFS.
- **Verdict.** Not voted: decided in round 1 (threat-model.md A3-29, P10, B9, G21; decision 51; scope.md "Filesystems").
- **Disposition.** Deliberately not changed, because the configuration and its consequence were decided in round 1 and the code is as the unsupported-configuration text allows. The auditor's proposed text (a "Shared directories" paragraph in the user guide, a sentence in the README, a qualifier on guarantee 5, naming bind mounts, virtiofs, VM shared folders, NFS and SMB and the silent loss) is not applied here and is a documentation change the owner can take; it would be a low-risk one, and the dedicated runs show the loss is dense and silent rather than rare.

### L3b-12: the writer never cross-checks `quill.tail` at flush, so a second writer past the lock is not detected until Close (medium as filed, gap, decided in round 1)

- **Claim.** A second writer that got past a non-coherent lock is not detected at write time and keeps writing until `Close`; the only runtime defence is the flock plus the decision-62 lock-file identity check, which does not help when flock is not shared.
- **Grounds.** The L3b-11 abuse run: both writers ran 400 appends to `CLOSE <nil>` with no error. scope.md:375-378 describes the lock-file check as the only self-detection. Not read in flush.go beyond that description by the auditor (inferred: no marker re-read exists).
- **Warrant.** Re-reading `quill.tail` at each flush and comparing it with the last published value could fail the log (`ErrFailed`) after one foreign write instead of hundreds. It narrows the window but cannot close it (read-check-write is not atomic).
- **Backing.** Guarantees 5 and 6 ("corruption is never served silently"); decisions 37 and 62.
- **Rebuttal.** One small read per flush on every writer's hot path to help an unsupported configuration; the check is racy; the owner may prefer documentation only. Not prototyped.
- **Verdict.** Not voted: prior deliberate (threat-model.md A3-29 disposition, A5-16 residual, the lock-removal finding "out of the threat model").
- **Disposition.** Deliberately not changed: hot-path cost for an unsupported configuration, and decided before. Option left for the owner if runtime detection is ever wanted: at flush, before the write, read `quill.tail` and fail the log with a distinct error when it names a seq/off other than the writer's last publication.

## Repeating the runs

`hack/topology.sh run -dir DIR -duration 60s -kill-every 150ms -pace 5ms -segment 65536 -seed N` builds `quill` and the harness and runs them (read the header of `hack/topology.sh` and `topology help` for the flags); add `-retention 1s -max-bytes 65536`
for caps, `-fill` with `-ctl` on another filesystem for a full disk, `-no-unregister` (now only a precaution since L3b-4). Prefix `GOMAXPROCS=1` for the single-processor variant that surfaced L3b-2. The scratch drivers, the replayer and the box scripts
are described in [harness-inventory.md](harness-inventory.md), section 9, and are not in the repository.
