# What the tests cannot show, and why

Every guarantee in [scope.md](../design/scope.md) has a test that could falsify it, with two
kinds of exception recorded here: behaviour that needs hardware the suite does not have, and
behaviour the suite tests by simulation only. Each entry says what is tested instead and what
remains an assumption. Updated at every phase.

## Phase 1

### A real disk's behaviour on power loss (needs: a machine you can cut power to)

Power-loss behaviour is tested against the crash simulator in `internal/fsx` (decisions 52
and 61), whose model is: an unsynced page may be lost or torn at 4 KiB granularity and reads
back as zeros; `fsync(file)` makes data and size durable but not the directory entry;
`fsync(dir)` makes creates, renames and unlinks durable; each unsynced item persists or not
independently. A `SIGKILL` cannot show any of this because the page cache survives it.

What the simulator cannot show, because it is a model:

- **Whether `fsync` reaches the platter.** A drive with a volatile write cache that ignores
  flush commands, or a filesystem mounted with barriers off, breaks every durability claim
  quill makes. On macOS `File.Sync` issues `F_FULLFSYNC`; on Linux it is `fsync(2)`, whose
  meaning depends on the mount and the device. Assumed, not tested.
- **Sector-level tearing.** The model tears at page granularity. A drive that tears inside a
  512-byte sector, or writes a sector partially, produces a frame whose CRC fails, which recovery
  treats as a torn tail: the outcome is the same as the model's, but it is reasoned, not run.
- **Reordering across devices**, and a filesystem's own journal replay semantics (ext4
  `data=ordered` versus `writeback`, APFS). The model lets any independent item persist or not
  in any combination, which is a superset of what a journaling filesystem does in practice; a
  filesystem that violates even that (a rename that persists while the inode's synced data does
  not) would break the atomic-write primitive. Assumed.
- **Metadata that persists without the data it names** on a filesystem with `data=writeback`:
  the model lets a rename persist while unsynced pages do not, which is exactly the hazard; the
  atomic-write primitive fsyncs the data before the rename, and the test
  `TestARenameThatPersistsBeforeTheDataIsTheHazardAtomicWritesAvoid` shows the model exposes it.

The phase-4 benchmark and the T14 full-disk job run on a real filesystem but never cut power.

### Symbolic links, FIFOs and devices inside the simulator (needs: nothing; a choice)

The simulator has no symbolic links or special files. Every refusal of a link, a FIFO or an
oversize file is tested against the real filesystem (`fsx.OS`) in a temporary directory, on
the machine running the tests.

## Phases 2 to 4

### An OS crash on a real machine (needs: a machine you can cut power to)

**Partly done**: a Linux server was rebooted without sync (`sysrq b`) under a running
writer; see [power-loss.md](power-loss.md). What a disk's own write cache does on a true
power cut remains untested. Everything above still holds. Phases 2 and 3 tested the crash property with a crash injected
before every file operation of two scenarios, under every subset of surviving items when
few are unsynced and a sample otherwise, and under a real `SIGKILL` of a writer process. The
`SIGKILL` runs show that nothing the writer acknowledged is lost when the page cache survives;
the simulator runs show the same when pages and directory entries are lost independently.
What remains an assumption is that a real filesystem's power-loss behaviour is within the
simulator's model (see phase 1).

### A full disk (needs: a filesystem you can fill)

T14 fills a small tmpfs in CI (`.github/workflows/ci.yml`, job `fulldisk`). It was also run by
hand on a Debian 13 server (Linux 6.12, a 4 MiB tmpfs) on 2026-10-28: the filesystem filled
after 64 acknowledged 4 KiB records, every acknowledged record read back, `Check` was healthy,
and the writer recovered without a restart once the filler was removed. The short-write and
`ENOSPC` paths are also tested against the simulator (T3).

### A backlog larger than memory on a real disk (done)

`BenchmarkDrainBacklog` was run with a 12 GiB backlog on an 8 GiB Linux server (six rounds,
`benchmark-baselines.md`): the drain runs at about 330 MB/s, slower than both the disk (about
1.1 GB/s raw) and the warm decode (about 740 MB/s), because reading and decoding run in
series. That answers decision 15's question and leaves a measured follow-up. A real disk rather
than a virtual one has not been measured.

### Lock release on process death across machines (needs: nothing more; stated for clarity)

`flock` release on `SIGKILL` is tested on the build machine (macOS) and in CI on Linux. A
network filesystem can lose or ignore locks, which is why NFS, SMB and FUSE are unsupported
rather than tested.

## Pass L2 (stage 2, 2026-11-03)

Pass L2 ran hand mutation over the whole library and the CLI (530 mutants, see [mutation-hardening.md](mutation-hardening.md)). Every survivor that a test could kill was killed by a committed test. What follows is the
residue: survivors and uncovered branches that no test in the present harness can see, each with its exact blocker and the artifact that would unblock it. Labels: *verified* (the recorder read it or ran it), *reported*
(an auditor or skeptic says so), *inferred* (the recorder's proposal, not tried). The harnesses are set out in [harness-inventory.md](harness-inventory.md).

### Observing an fsync on a real file system (mutant O07; needs: syscall tracing or a fault-injecting block device)

**Blocker.** `fsx.OS.SyncDir` (`internal/fsx/os.go`) is one `fsync` call on a directory handle. Deleting it passes every test on macOS and on Linux (reported, write run, including a Linux sample). The simulator models directory
fsync independently of the OS wrapper, and a real `SIGKILL` keeps the page cache, so no test observes that the syscall was made. The same holds for the data `fsync` on a real file: the simulator pins that the writer *calls* `Sync`; nothing pins
that `OS.Sync` reaches the kernel. **Artifact that would unblock it (inferred).** Either a Linux CI step that runs the OS-backed tests under `strace -f -e trace=fsync,fdatasync` and asserts the directory and file descriptors named; or a device-mapper
target that logs writes (dm-log-writes) or injects flush faults (dm-flakey), mounted for one CI job; or, in code, a package-level function variable for the syscall that a test replaces and counts. The cheapest is the function variable.
Everything in power-loss.md about a real disk still applies.

### A write-only open that no caller produces (mutant O05, `fsx/os.go:21-23`; needs: a unit test of `fsx.OS` alone, or deletion)

**Blocker.** `osFlags` translates `O_WRONLY` to a read-write open; no caller outside the simulator passes `O_WRONLY` (reported, grep of the tree), so no black-box test through the library reaches the translation. **Artifact (inferred).**
A direct test in `internal/fsx/os_test.go`: open a temporary file with `O_WRONLY` through `fsx.OS.OpenFile` and require that `ReadAt` on the handle succeeds. Not tried. If the owner would rather not keep an unused translation, deleting
it (class d, dead) closes the item.

### A signal landing on a rename (mutant O06; `fsx/os.go:76`; needs: a syscall seam or a signal storm)

**Blocker.** `OS.Rename` retries on `EINTR`; a test needs a signal delivered at exactly that syscall (reported). **Artifact (inferred).** A package-level `var rename = os.Rename` that a test replaces with a function returning `syscall.EINTR` once; or a
stress test that sends `SIGURG` to the process in a tight loop while renaming (probabilistic, not a guarantee of coverage).

### A collision in a 64-bit random temp name (mutant T05; needs: an injectable random source)

**Blocker.** `atomicfile` opens its temp file with `O_EXCL` and a 64-bit random suffix; dropping `O_EXCL` is invisible unless two writers draw the same suffix (reported). **Artifact (inferred).** A function variable for the suffix source (`tempName`) that a test fixes to
a constant, so the second `Write` collides and must fail or retry.

### `crypto/rand` failure (`state.go:174`, `atomicfile.go:60`; needs: nothing; the branches cannot run)

**Blocker (verified).** `go doc crypto/rand.Read` on the pinned toolchain (go1.27.3) says `Read` "never returns an error" and "crashes the program irrecoverably if an error is returned". The two `if _, err := rand.Read(...); err != nil` branches
are therefore unreachable. **Artifact.** None can exist; deleting the error branches (class d) is the only way to close the item. Not done in this pass.

### The path a `Lstat` error names (mutant L27; needs: a non-searchable parent directory as a non-root user)

**Blocker.** `resolveDirLink`'s non-`ErrNotExist` `Lstat` error differs from the mutant only in which path the error names (reported). The simulator has no symlinks, so the case needs a real directory. **Artifact (inferred).** A test on `fsx.OS` or
`OpenReader` with a `chmod 0` parent in a temporary directory, asserting the error text names the component that failed; it must skip under root.

### A failed `Roll` leaves `pendingPublish` set (mutant F36; needs: a count of marker rewrites)

**Blocker.** Differs by one idempotent marker rewrite (reported): the visible state is the same. **Artifact (inferred).** `Sim.Inject` counting `OpRename` onto `quill.tail` across a failed `Roll` and the next flush, asserting the count. Not tried; it may be a legitimate gap
and not an unobservable one.

### A hook between two operations (mutants K08, `reader.go:266`; branches `load.go:83-84`, `read.go:329`; needs: a hook the `Sim` can run between two operations)

**Blocker.** Four paths need something to change *between* two file operations of the code under test: `SeekToTime`'s retry when retention moves `First` between `OffsetForTime` and `Commit` (`seek.go`, K08, reported as real but not testable); a segment swapped to a non-regular
file, or resized, between `Lstat` and open (`load.go:83-84`, `read.go:329`); and a race between two builders of the same `readerSegment` (`reader.go:266`, flaky by construction). `Sim.Inject` runs before an operation without the simulator's lock but "must not call back into the Sim"
(`internal/fsx/crash.go`, verified), so it can fail an operation and cannot mutate the file system around it. **Artifact (inferred).** Either relax that rule with a documented, locked `Sim` method a hook may call (a new exported symbol in an internal package), or an unexported hook variable in
`seek.go` between the two calls. Without either, the L2-45 tests model the vanishing segment as an injected `ErrNotExist`.

### The CLI has no file system seam (mutant C33, and CLI error paths: `cmdAppend`'s `Sync` error, `cmdPrune`'s `Stat` failure; needs: a seam for package `main`)

**Blocker.** `Config.fs` is unexported, so package `main` cannot inject a fault; `cmdCommit`'s `lerr == nil &&` needs `Consumers()` to fail and then `Commit` to fail, and would delete a pre-existing consumer's offset file (reported, real but not filed). The Linux `prune` unlink failure
was closed without a seam by a `chmod 0500` of the log directory from the verb's own stderr hook (L2-10); that trick skips under root and depends on the text and order of one log line (reported by the fix-skeptic). **Artifact (inferred).** An unexported hook in `cmd/quill`
(a function variable for the file operations the verbs call), or a test-only build tag that exports `Config.fs` to package `main`. Either is a code change for the owner to approve.

### An injectable clock in the CLI (mutants C61 and C62; needs: a clock seam)

**Blocker.** `quill consumers`' stale boundary at `now - mtime == ttl` reads `time.Now()` inside the verb (reported). L2-31 pinned the same boundary by calling `(*cli).stale` directly, which tests the function and not the verb's call into it. **Artifact (inferred).** A `now func() time.Time` on `cli`.

### Races with no deterministic test (mutant S28, `Log.wait`'s entry check)

**Blocker.** The entry check `l.head >= offset` matters only when a flush lands between `readFrom` and `wait`, a window of a few instructions (reported). **Artifact.** The same hook between two operations as above, in `read.go`; or a stress test that appends from another goroutine in a
tight loop (probabilistic).

### Fuzzing beyond a smoke run (needs: hours of CPU and a corpus)

**Blocker.** The gate runs each of 15 fuzz targets for 20 s. The auditors ran minutes (a 300 s differential of the tail decoders, 4.8 million execs, no divergence; 30 to 120 s on the others), with execs stalling for about 15 s at high machine load (reported). No target has a committed
corpus. **Artifact.** A scheduled CI job with `FUZZTIME=30m` per target and `-test.fuzzcachedir` persisted, plus `testdata/fuzz` for any input that ever failed.

### Linux on ARM, and a systematic Linux mutation run (needs: a runner)

**Blocker.** All mutation runs but the recovery run were on macOS arm64; the Linux x86-64 box ran 146 of 195 recovery mutants and small samples of the others. No run used linux/arm64 (the Apple `container` CLI was available and unused). `hack/powertest` and `hack/skewopen` are not measured by the
coverage run. **Artifact.** The `container` CLI with a cross-compiled test binary (`GOOS=linux GOARCH=arm64 go test -c`) bind-mounted into `alpine:3.20`; and the mutation runner from harness-inventory.md pointed at the Linux box for the whole set.

### Closed in pass L2

The Linux half of the `prune` unlink-failure exit (above), `freeBytes` (L2-9: the unit by a real write, the field on a made-up `Statfs_t`), `Reader.ID`, `Reader.Scan`, `Consumer.Name` and `Consumer.Scan` (pass L1 `7497a80`), the wall-clock assertions that made the gate flaky (L2-8), and every
mutant listed as a "real gap" in mutation-hardening.md. The phase 1 to 4 items above (power loss on a real machine, a full disk, a larger-than-memory backlog, lock release across machines) are unchanged.
