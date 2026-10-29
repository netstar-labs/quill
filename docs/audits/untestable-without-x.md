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
