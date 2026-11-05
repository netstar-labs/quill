# Test harness inventory

Date 2026-11-03, against the tree at `f274b51`; updated the same day (pass L3b) for `hack/topology`, `hack/skew.sh` and the scratch-only box scripts, against `ba30fee`. Every harness, fixture and seam the tests use, what it can simulate, and what it cannot. A claim that a test "covers" something is
only as good as the harness under it, so the limits matter more than the features. Entries were read from the tree by the recorder (verified); statements about what a harness
cannot do are the recorder's reading of its doc comments unless marked reported (from an audit) or inferred. Companion files: [test-ledger.md](test-ledger.md) (which test covers which
guarantee), [untestable-without-x.md](untestable-without-x.md) (what no harness here can show), [mutation-hardening.md](../passes/l2/mutation-hardening.md) (which harness limits caused surviving mutants).

## 1. The crash simulator: `fsx.Sim` (`internal/fsx/crash.go`, `crash_file.go`)

An in-memory `fsx.FS` that models what a power loss does to unsynced state (decision 61). Every file has a durable image and a volatile image; every directory has a durable entry
table and a list of unsynced entry operations.

| Can simulate | Cannot simulate |
|---|---|
| A write changes the volatile image and marks the 4 KiB pages it touched, and the size if it changed, as unsynced. `Sync` makes data and size durable but not the directory entry | Tearing finer than a 4 KiB page, a partly written sector, or a drive's write cache ignoring flushes (decision 61 and untestable-without-x.md). A real `fsync` reaching the platter |
| Create, mkdir, rename and unlink are unsynced operations on their directory until `SyncDir`; a rename over an existing name yields the old or new content until synced; an unlink or a truncate may be undone | The ordering rules of a particular journal (ext4 `data=ordered` against `writeback`, APFS). The model lets each unsynced item persist or not independently, which is a superset of what a journaling filesystem does (reported, untestable-without-x.md) |
| `Crash(keep)` decides, for each unsynced page, size and directory operation, whether it persisted; a lost page of an extended file reads back as zeros; persisted directory operations apply in the order issued; every open `File` is dead afterwards | A crash inside a single operation. `Inject` can crash only before an operation (`ErrCrash`) or short-write (`*ShortWrite`) |
| `Inject(op, name)` runs before every operation (open, create, read, write, sync, truncate, mkdir, readdir, lstat, rename, remove, syncdir) and may fail it with any error, crash the simulator, or sleep to model a slow device. It runs without the simulator's lock | `Inject` must not call back into the `Sim`, and must be set only while no goroutine uses the simulator. A test cannot use it to change the file system between two operations of the code under test, so races such as a segment swapped between an `Lstat` and an open can only be modelled as an injected error (L2-45), not as a real swap |
| `OpenFiles()` counts open handles, `Ops()` counts operations, `Unsynced()` counts unsynced items, so a test can check descriptor leaks and drive "crash before every operation" | No symbolic links, FIFOs or device files (those are tested on `fsx.OS`). No `statfs`; free space is the separate `Config.statfs` seam |
| Advisory locks: `TryLock` is exclusive per file, released by `Close` and by a crash; `Chmod` sets the mode | Locks are not per process or per open file description. Whether the simulator enforces permission bits is not stated in its doc comment, so tests that need `EACCES` inject it. Inferred, not checked: no mode enforcement |
| `Clock` stamps modification times (nil uses `time.Now`) | Modification times of real files under real clocks, `Chtimes` on a real file (the CLI stale probe test uses `os.Chtimes` on the real FS for this) |

Used by almost every root-package test through `openSim` and `simConfig` (`helpers_test.go`), and by `crash_test.go` (T1, T2, T27), `internal/fsx/crash_test.go` (the model's own tests), and the retention, recovery, archive and reader tests.

## 2. The real file system: `fsx.OS` (`internal/fsx/os.go`)

The production `FS`, tested on the machine running the tests in a temporary directory (`internal/fsx/os_test.go`, `internal/fsx/crash_test.go` `TestTheOSImplementationRefusesSymlinksAndItsLocksExclude`, `TestConformance`; and at the root `fifo_test.go`, `open_refused_test.go`, and `cmd/quill/symlink_dir_test.go`).

| Can simulate | Cannot simulate |
|---|---|
| Symlinks, FIFOs, oversize files, `O_NOFOLLOW` and `O_NONBLOCK` refusals, real `flock` exclusion between handles and between processes, real `rename` and `unlink`, real permission bits (as a non-root user) | A power cut (a `SIGKILL` keeps the page cache). Observing that `fsync` was called, or on what: no test can see it without syscall tracing (mutant O07). A signal landing on a syscall (the EINTR retry, O06) |
| macOS and Linux behaviour of the same code, by running the suite on each (CI matrix `ubuntu-latest`, `macos-latest`) | Windows (unsupported), network filesystems (NFS, SMB, FUSE are unsupported rather than tested) |

## 3. Seams in the code under test

Unexported fields of `Config` (`log.go:106-107`), reachable only from tests inside package `quill`:

| Seam | Meaning | Used by | Cannot do |
|---|---|---|---|
| `Config.fs` | the file system; a `*fsx.Sim` or a wrapper around one | all Sim tests | package `main` (the CLI) cannot reach it, so the CLI is tested against real files only (the reason for the missing CLI seam in untestable-without-x.md) |
| `Config.clock` | `func() time.Time`, nil is `time.Now` | `testClock`, `newClock`, `fixedClock`, retention and age tests | the CLI reads `time.Now()` itself, so CLI time boundaries are tested by calling `(*cli).stale` directly |
| `Config.statfs` | `func(dir string) (uint64, error)`, nil is `freeBytes` | `statfsSeam` (`diskfull_test.go`), caps tests | replaces `freeBytes` wholesale; the real function is covered by `disk_test.go` (L2-9): `availBytes` on a made-up `Statfs_t` for the field, an fsynced 8 MiB write on the real FS for the unit |
| `OpenReader` unexported fs option | a Reader on a Sim | reader tests | same CLI limit |

## 4. Fault-injecting wrappers (test files)

Each embeds an `fsx.FS` and fails one thing; all are test-package types.

| Wrapper | File | Fails | Used for |
|---|---|---|---|
| `failOps(s, op, armed)` | `wait_failed_test.go` | every `op` while an `atomic.Bool` is set, with EIO | the flusher, rotation and marker-publish failures (L2-15, L2-16) |
| `chmodFailFS` | `lock_mode_test.go` | `Chmod` on the writer's lock file, counts tries | lock mode tests |
| `openFailFS` | `lock_mode_test.go` | opens | lock and open refusals |
| `tryLockFailFS` | `lock_refused_leak_test.go` | `TryLock` with a non-`ErrLocked` error | descriptor leak on a refused lock (L2-20) |
| `policyFaultFS` | `policy_stale_test.go` | the rename onto `quill.policy`, optionally its removal | background policy-write failure |
| `consumersProbeFS` | `seal_consumers_io_test.go` | nothing; counts listings of `consumers/` and how many ran while the log lock was held | "a consumer must not block the writer" |
| `failStepFS` / `failStepFile` | `internal/atomicfile/atomicfile_test.go` | `File.Chmod` or `File.Close` with EIO, counts `Rename` | atomicfile failure steps (L2-21) |
| `eofAtEnd` | `load_boundaries_test.go` | a `ReaderAt` that returns `io.EOF` together with the final full read | window boundary (L2-38) |
| `peakOpen(s)` | `resource_bounds_test.go` | nothing; records the highest `OpenFiles()` through `Inject` | peak descriptors per read (L2-58) |
| `failWriter`, `hookWriter`, `countWriter` | `cmd/quill/*_test.go` | an output writer that fails, runs a hook on a line, or records write sizes | CLI output errors, the Linux unlink-failure test (L2-10), printer buffering |

**Limits common to all:** each wrapper injects a fault the author thought of. The Sim cannot inject a fault that depends on a second operation having happened between two others, and none of these
can fail the CLI's own file operations except by a real file system trick (a non-root user's `chmod 0500`, `chflags` on macOS).

## 5. Log and fixture builders (test files)

| Builder | File | Produces |
|---|---|---|
| `openSim`, `openDir`, `simConfig`, `appendN`, `mustFlush`, `payloadFor` | `helpers_test.go` | a log on the Sim or in a temp directory, with a slow flusher so only the test drives file operations, small segments, deterministic payloads |
| `retentionLog`, `openRetention`, `fillSegments`, `testClock`, `newClock` | `retention_test.go` | a log with a test clock, sealed segments a minute apart (`fillSegments`) |
| `rolledLog`, `damagedLog`, `damageSegment`, `setMarkerSeq`, `flipLastByteOfFrame` | `load_holes_head_test.go`, `load_damaged_bytes_test.go`, `read_damage_after_open_test.go` | logs rolled into segments of fixed record counts, and logs with one or more segments damaged by a flipped byte or a changed marker |
| `vanishingLog`, `unwarmedVanishingLog`, `readEIOSetup`, `simLogWithHead`, `closedSimLogWith`, `edgeLog`, `oneRecordSegments`, `segmentsPerRecordSim`, `failedFinalFlush` | reader, stat, archive and sync tests | the fixtures for segments deleted between listing and scan, I/O errors on read, a head at the sequence ceiling, a hand-built directory with chosen images, single-record segments, and a closed log whose last flush failed |
| `image`, `frameEnds`, `recover_`, `verify`, `appendFrame`, `forgedFrame` | `codec_test.go`, `load_test.go` | byte images of valid or forged frames, and wrappers that run `recoverSegment` and `verifySegment` on a byte slice |
| `archiveSource`, `rewindArchiver` | `archive_dest_test.go`, `archive_rerun_test.go` | an archive source and a rewound `_archive` consumer |
| `newLog`, `buildLog`, `mustRun`, `exec`, `snapshot`, `forgeAge`, `rolledLog` (CLI copy) | `cmd/quill/*_test.go` | a real log on disk, the CLI run in process with captured output and exit code, a snapshot of a directory's files, and a rewrite of every record's timestamp with the CRC recomputed (the CLI cannot stamp old records) |

**Limits.** The builders make logs the writer could make, plus the damage their author imagined. A damaged state the auditors did not think to build is not tested; the fuzz targets (section 7) exist to cover that.
`forgeAge` and the frame forgers recompute the CRC, so they test the decoders and not the writer.

## 6. Real-process harnesses

| Harness | Where | What it does | Cannot do |
|---|---|---|---|
| SIGKILL helper | `helpers_test.go` (`TestMain` runs a helper role when `QUILL_HELPER` is `writer` or `holder`, controlled by `QUILL_HELPER_DIR`, `QUILL_HELPER_SEGBYTES`, `QUILL_HELPER_SYNC`), used by `kill_test.go`, `reader_test.go`, `log_test.go` | the test binary re-executes itself as a writer or a lock holder in another process and kills it | `SIGKILL` leaves the page cache intact, so it shows nothing about unsynced data |
| `hack/powertest` | `hack/powertest/main.go`, `main_test.go` (`POWERTEST_AS_MAIN`) | the real power-loss test: `run` writes and acknowledges to a separate fsynced file until the machine is crashed with `echo b > /proc/sysrq-trigger`; `verify` after the reboot checks every acknowledged record survived and the log is a contiguous prefix. Results in [power-loss.md](power-loss.md) | needs a machine you can crash; not part of any automatic run. A drive's own write cache on a true power cut is untested |
| `hack/skew.sh` and `hack/skewopen` | `hack/skew.sh <refA> <refB>`; `hack/skewopen/main.go` | builds the CLI at two refs (`git archive`, `GOWORK=off go build`) and runs the same experiments both ways, A writes and B reads and back: records of several size classes in several segments, a consumer committed in the middle, a byte-for-byte comparison of every read-back against the generated payloads, `check` on both, appends by the other version, `kill -9` of a writer mid-append then recovery by each version on its own copy of the killed bytes (the two directories must be identical) and a further append by the other version, and `quill.policy` and consumer files rewritten by each version. `SKEW_KILL_DELAYS` sets the kill timings and `SKEW_KEEP=1` keeps the work directory; it exits 0 only if both versions agree everywhere, and otherwise prints the evidence, exits 1 and keeps the directory | needs two refs and a shell (bash 3.2 and BSD tools suffice); macOS and Linux only; `skewopen` may use only the API every ref has. It is offline: no concurrent processes, so it shows nothing about skew under live traffic (pass L3b ran one version only), and a kill is at a few chosen delays, not at random |
| `hack/topology` and `hack/topology.sh` | `hack/topology/main.go`, `run.go`, `main_test.go`; `hack/topology.sh run\|selftest\|verify [flags]` builds `quill` and the harness into a temporary directory and runs the harness from the current directory | A deployment on one machine, as separate OS processes: one writer, readers that start at the first record, at the tail and at random offsets, consumers (one protected), an operator that runs the `quill` CLI in loops (`check`, `stat`, `read`, `archive`, `register`, `commit`, `unregister`, the writer-lock verbs), a pruner, and a chaos controller that SIGKILLs a role at random times, SIGSTOPs the protected consumer for `-stall` (with `-retention`), starts a second writer to probe the lock, and with `-fill` fills the filesystem under test (with `-ctl` on another filesystem). Every record carries a run id, a sequence number and a checksum; consumers log "intent N" and "done N" around each commit. At the end `verify` checks every acknowledged record, contiguity, payloads, `Check`, the consumer offsets against that oracle, and every reader's observed spans and gaps against the final log; a role that sees a skipped record stops the run at once. Output is a JSON summary; exit 1 on any violated invariant. `topology selftest` injects faults (a skipped record, a quiet skip, an early acknowledgement, a protected consumer passed, a TTL case) and requires each to be reported; `-quick` runs the four fast cases, which is what `go test` runs and so what the gate runs. Results: [live-integration.md](../passes/live-integration.md) | Scheduling is random and the verdicts are not, so a clean run is evidence about the interleavings it took. Its `chaos.kills_<role>` counter counted attempted kills before the L3b-3 fix; `restarts_<role>` is the delivered count and a lower bound. The chaos second-writer probe was not shown to be the check that fires (the pruner's roll failure fires first); the probe runs under the same uid, so it cannot show the different-uid refusal (L3b-7). It cannot cut power or reorder writes, and it runs one version of quill. The `-no-unregister` help text predates the L3b-4 fix. Roles hold up to 64 MiB records, so a small VM can be killed by the kernel for memory |
| Full-disk job | `backlog_test.go` (`QUILL_FULLDISK_DIR`), CI job `fulldisk` | fills a 4 MiB tmpfs for real | skipped unless the variable is set; was run by hand once on a Debian server (untestable-without-x.md) |
| Drain benchmark | `bench_test.go` (`QUILL_BENCH_BACKLOG_BYTES`) | measures a backlog drain | not run in pass L2 |

## 7. Fuzz targets and goldens

| Harness | Targets or files | Oracle | Cannot do |
|---|---|---|---|
| `fuzz_test.go`, `fuzz_reader_test.go`, `fuzz_names_test.go`, `cmd/quill/fuzz_test.go`, `internal/strictjson/strictjson_test.go` | `FuzzDecodeFrame`, `FuzzRecoverSegment`, `FuzzDecodeTail`, `FuzzTailDiff`, `FuzzDecodeFormat`, `FuzzDecodeConsumer`, `FuzzDecodePolicy`, `FuzzWalkChunks`, `FuzzReaderDir`, `FuzzValidName`, `FuzzParseSegmentName`, `FuzzParseDecimal`, `FuzzParseTime`, `FuzzPrinter`, `FuzzUnmarshalNeverPanics`, run by `hack/fuzz.sh` at 20 s each in the gate | per target: round trips, differential against a slow path or a reference, no-panic, and for `FuzzReaderDir` "never a record above the marker, never seq 0, never a jump without an error" | 20 s per target is a smoke run, not a campaign. No committed corpus (no `testdata/fuzz`); seeds are written by hand or built from a real writer's directory (`FuzzReaderDir`). `FuzzReaderDir` ignores `r.Read` errors, so it checks safety and not liveness (reported by the L2-55 fix-skeptic) |
| `fuzz_script_test.go` | a test of `hack/fuzz.sh` itself (it must fail closed when a package does not compile) | the script's exit status | any fuzz target's own coverage |
| `testdata/golden` + `golden_test.go` | `consumer`, `consumer-default`, `quill.format`, `quill.policy`, `quill.tail`, `segment.log`, recorded once from the encoder (`QUILL_RECORD_GOLDENS=1 go test -run Golden`) | the bytes recorded, in both directions (encode equals the bytes, decode reads them) | a golden is the bytes of one version; it cannot show a format difference the recorded sample does not exercise. `TestRecordGoldens` is a recorder and is skipped by default |
| `TestArchiveAgainstTheSourceReaderOverDamagedSegments` | `archive_differential_test.go` | a seeded differential of Archive against the source Reader on the Sim with damaged segments | a weak oracle: see mutation-hardening.md, residual weaknesses for L2-57 |

## 8. The gate and CI

`hack/gate.sh` is the commit gate: no `require` line in `go.mod`, `gofmt -l`, `go vet`, `go build`, `go vet` for `GOOS=linux` and `darwin`, a `freebsd` build and a `linux/386` and `linux/arm` build that must fail, `go test -race -count=1 ./...`,
staticcheck `v0.8.2`, deadcode `v0.52.0 -test`, `hack/fuzz.sh` at 20 s per target (`FUZZTIME=0` skips it and the gate then ends PARTIAL), and `go mod tidy` with no diff. The Go toolchain is the one `go.mod` pins
(`toolchain go1.27.3`). `.github/workflows/ci.yml` runs the same steps as separate jobs on `ubuntu-latest` and `macos-latest`, plus `govulncheck`, the full-disk job, the examples and the release build script.
The gate cannot show: anything the machine it runs on does not have (a real power cut, a small filesystem other than the CI tmpfs), and `govulncheck` (it reads a live database, which a commit gate must not do).

## 9. Audit-time tools that are not in the repository

Pass L2's results rest on tools that live only in scratch space and are set out here so a later pass can rebuild them:

- A mutation runner: a copy of the tree made with `git archive` per mutant, one textual edit, `go build ./...`, then `GOWORK=off go test -count=1 -timeout 150s` over the chosen packages, run in parallel. It
  must run the load-sensitive tests alone or skip them (see mutation-hardening.md, "The load-flake trap").
- A Linux x86-64 box (Debian 13, ext4, four vCPU), reached through a script that unpacks `git archive <ref>` into a fresh directory and runs a command there with Go on `PATH` and a temporary directory on the disk, not the tmpfs.
  It was shared between agents; timing-sensitive jobs took a lock.
- The Apple `container` CLI runs linux/arm64 containers; it was not used in pass L2 (it was in pass L3b, below).
- The auditors' kill tests (a scratch `zz_kill*_test.go` set) are not in the repository; the fixers' committed tests replace them.
- Pass L3b (live multi-process validation) added scratch-only tools, none of them in the repository; the results are in [live-integration.md](../passes/live-integration.md). Host and path names are left out on purpose.
  - **Scratch drivers for the macOS host.** Shell scripts that run `hack/topology` in batches with chosen seeds, flags and `GOMAXPROCS`, and a summariser of the JSON results; and a throwaway program with five
    subcommands, run against a copy of the tree: appenders with a Roll loop and a Prune loop under in-process `Follow` and `ReadWait`; 150 logs in one process with descriptor and goroutine counts; a rotation
    stress that races two `Archive` loops against a 1-millisecond Roll and compares every destination record with the source; a load of eight concurrent loops spawning the online CLI verbs beside an in-process
    writer; and a stale-marker stress. It reaches only what the library API and the CLI expose.
  - **The Linux box scripts.** One script unpacks the archive of a ref into a fresh directory on the box and runs a command there with Go on `PATH` and the temporary directory on the disk, not on a tmpfs (a
    variant exists for the earlier full-disk job); anything whose timing matters takes a lock first; the box is shared and runs one heavy job per agent. Around it: a loop-device and `mkfs` script that builds an
    image for each of ext4 (ordered, writeback, journal, `nobarrier`, no journal), xfs, btrfs, f2fs, tmpfs and overlayfs, mounts it, runs the topology scenarios (base, kill, retention, caps, full disk) against it
    and tears it down, leaving no loop devices, device-mapper targets or mounts; a dm-log-writes script and a **throwaway replayer** that applies a prefix of the write log to a base image, mounts it, runs
    `hack/powertest verify` against the last acknowledgement mark at or before the cut and runs a read-only `fsck` (validated against the device byte for byte on every filesystem here except f2fs, whose replay
    differed and was not investigated, so f2fs results are weaker; it does not apply discard entries); a dm-flakey script that switches a running writer's device to `drop_writes` or `error_writes` without
    freezing the filesystem, kills the writer, remounts clean and verifies against a snapshot of the acknowledgements; an in-process leak program that reads `/proc/self/fd`; and a privileged different-uid probe.
    None of them models reordering beyond what dm-log-writes logs or tearing below a block, and none can say whether a real disk's cache honours a flush.
  - **The Apple container probes.** The container CLI runs linux/arm64 VMs; the harness and `quill` are cross-compiled on the host (never built in the container); a bind mount gives a directory shared across two
    VMs or with the host (virtiofs, an unsupported filesystem by decision 51); and a small probe program (a writer with synchronous appends that prints an acknowledgement line per durable append, a tight `Head()`
    loop, a follower) shows what each side sees. A default 1 GB VM can be killed for memory under the harness's largest records.
  - A real power cut remains `hack/powertest` with `echo b > /proc/sysrq-trigger` on the box (one cycle in L3b; [power-loss.md](power-loss.md) for the earlier ones).
