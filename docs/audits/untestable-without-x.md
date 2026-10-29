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

### A backlog larger than memory on a real disk (needs: hours and a large disk)

`BenchmarkDrainBacklog` drains a backlog whose size `QUILL_BENCH_BACKLOG_BYTES` sets; the
baselines used 256 MiB, which fits in memory. The decision-15 question (does `pread` with
large chunks stay disk-bound on a backlog larger than memory?) is therefore inferred from
the warm figures, not measured. Run the benchmark with the variable above the machine's
memory to measure it.

### Lock release on process death across machines (needs: nothing more; stated for clarity)

`flock` release on `SIGKILL` is tested on the build machine (macOS) and in CI on Linux. A
network filesystem can lose or ignore locks, which is why NFS, SMB and FUSE are unsupported
rather than tested.
