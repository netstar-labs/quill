# Changelog

All notable changes to quill. The format follows Keep a Changelog; versions are `v0.y.z`
for ever, and the on-disk format is versioned separately in `quill.format`.

## Unreleased

- Fixed: the SBOM's `dependencies` entry named two identifiers that no component declared, so a consumer that resolves the
  graph found no edge from quill to the Go standard library. Both components now carry a `bom-ref` equal to their purl.

- Fixed: `build/quill` no longer puts the clock or the builder into the release archives and the SBOM. It took the SBOM
  timestamp from the wall clock and let `tar` and `gzip` record the build time and the builder's uid, user and group
  names, so two builds of one commit gave different `SHA256SUMS` and every archive named the person who built it. The
  timestamp and every entry mtime are now the commit time, entries are sorted, owners are root with no names, and the
  gzip header carries no time. Two builds of one commit with the same tar, gzip and Go now match; archives built with
  GNU tar and with bsdtar still differ in bytes (their extracted files are identical), so compare binaries across
  systems, and with `-buildvcs=false` when one side builds from a source archive (inside a git checkout the commit is
  stamped into the binary).

- Fixed: `docs/audits/audit-correctness.md` named a personal scratch directory (a user name and an internal parent
  directory) in its opening paragraph; it now says only `scratchpad/`. Documentation only; behaviour is unchanged. The
  text remains in earlier commits and the v0.1.0 tag; the owner decides before the public flip whether history is
  squashed or accepted.

- Fixed: `SECURITY.md` named no usable private reporting channel (it pointed at `.github/CODEOWNERS`, which holds only a
  GitHub login). It now names the Security tab "Report a vulnerability" route and states that only the latest release is
  supported. The owner must enable private vulnerability reporting in the repository settings before the public flip.
  Documentation only; behaviour is unchanged.

- Fixed: design-record drift. The `quill.policy` example in `docs/format.md` listed `_archive` as a protected
  consumer, which `Open` refuses; the disk-bound formula (decisions 46 and 69, the build prompt), the worst-case frame
  overhead (32 bytes, with the code's 34-byte accounting), the creation durability row (decision 81), decision 86 and
  the `statfs` note now agree with the code. Documentation and one comment only; behaviour is unchanged.

- Fixed: small documentation drift. The command reference said `ls` takes no `-dir` and then listed it; the README
  layout table now lists `holes.go`; the `ErrPayloadTooLarge` comment now says it also covers an empty batch; and the
  exported `ReadWait`, `Scan`, `ScanAll`, `ProtectOn`, `ProtectOff` and the `Error`/`String` methods have doc
  comments. Documentation only; behaviour is unchanged.

- Fixed: every `quill` error that came from the library printed a doubled prefix (`quill: quill: ...`); it now
  prints one. A `quill commit` that fails no longer leaves behind the consumer it had just created (the empty
  lock file stays, as for any consumer); a successful commit on a new name still registers it, now stated in
  the user guide.

- Fixed: `hack/gate.sh` printed `OK: gate green` even when `FUZZTIME=0` had skipped the fuzz step. It now ends with
  `PARTIAL: fuzz skipped`; the exit status is unchanged (0). The README gate summary now lists the no-require and
  unsupported-platform steps.

- Fixed: `SECURITY.md` said every state file is parsed strictly. `quill.policy` is informational and parsed
  tolerantly (decisions 29 and 48); the other state files are strict. Documentation only; behaviour is
  unchanged.

- Fixed: the user guide said `quill prune` refuses a log whose policy it cannot read, and the architecture
  document said every offline verb opens with no caps. `prune` refuses only when there is neither a readable
  `quill.policy` nor a retention flag, and it applies the policy's caps. Documentation only; behaviour is
  unchanged.

- Fixed: the `SegmentMaxAge` documentation said it never rotates when `Retention` is negative; that holds
  only for the default, and an explicit positive value rotates quiet segments whatever `Retention` is.
  Documentation only; behaviour is unchanged.

- Fixed: a process killed in the middle of copying a segment in `Archive` (or `quill archive`) left a
  `.tmp-*` file in the destination that nothing removed; the next run now removes those older than an
  hour from the destination, the rule the writer applies to its own directory.

- Fixed: `Archive` (and `quill archive`) created the destination directory without syncing it into its
  parent, so after a power loss the destination could vanish while the `_archive` consumer's committed
  offset still said every segment was copied, and the next run copied nothing; it now syncs the parent
  before copying or committing.

- Fixed: `Sync` of a sequence that had not been assigned read the last assigned sequence for its `ErrBadSeq`
  message after releasing the log's lock, a data race with concurrent appends that the race detector
  reported; the number is now read under the lock.

- Fixed: a long-lived `Reader` (or `Follow`) kept a cache entry, and the index of every segment it had
  scanned, for each sealed segment it ever listed, including ones retention had deleted, so its memory
  grew with every rotation; it now drops the entries of segments that are no longer in the directory.

- Fixed: `Archive` (and `quill archive`) accepted the source log itself, or a directory with a live writer,
  as the destination and overwrote that writer's `quill.tail` with a marker naming the last sealed segment,
  hiding the newest records from every `Reader` until the writer's next flush that changed the marker; it
  now refuses the source log as its own destination, and a destination whose `quill.lock` is held with
  `ErrLocked`, before writing to it.

- Fixed: a writable `Open` that failed with `ErrCorrupt` could already have truncated and synced the
  newest segment, because the untrusted tail was cut before the overlap and last-sequence-below-the-marker
  checks ran; the cut is now made only after every check has passed, so a refused `Open` leaves the file
  as it found it.

- Fixed: a prune pass that deleted several sealed segments synced the directory once, before the first
  unlink, so a power loss could persist a later unlink without an earlier one and leave a hole in the
  middle of the retained records; the directory is now synced before each unlink, so what survives a
  crash is always a suffix. Deleting n segments costs n directory syncs instead of one.

- Fixed: at the sequence ceiling the API contradicted itself: after the record numbered 2^64-1 was
  accepted, `Sync` of it returned `ErrBadSeq`, so did `Append` with `SyncAppends` although the record
  was durable, and `Read` of it returned `ErrCorrupt`. The last sequence, 2^64-1, is now never
  assigned; the ceiling is 2^64-2, after which `Append` returns `ErrSeqExhausted`. A log that
  already ends at 2^64-1 (crafted or restored) is refused with `ErrCorrupt` by `Open` and
  `OpenReader` instead of being opened with the same contradictions.

- Fixed: `build/quill --dev` exited 1 with no output on a clean working tree, so the development
  build (and the CI step that runs it) produced nothing unless the tree was dirty; the version
  string is now computed without a failing command under errexit.

- Fixed: `quill check` (and `Reader.Check`) reported a healthy log UNHEALTHY when retention deleted a
  sealed segment after the check had listed the directory, as damage at byte 0 and, for a middle
  one, a hole; a sealed segment that vanishes mid-scan is now skipped and the hole test restarts
  at the next survivor. A segment missing from the listing is still a hole.

- Fixed: `quill cat`, `read`, `tail`, `stat`, `check`, `ls` and `consumers` exited 0 when writing to
  standard output failed (a full disk, a closed descriptor) unless the output was over 4 KiB, so
  records could be lost without the exit status saying so; a failed write now exits 1 with the
  error, and `tail -follow` stops instead of waiting on.

- Fixed: one transient I/O error (such as too many open files) while a `Reader` scanned a sealed
  segment was remembered for the life of the `Reader`, so every later read of that segment's
  range returned a `GapError` although the file was intact; only damage is remembered now, and
  an I/O error is retried on the next read.
- Fixed: a sealed segment file removed by hand while the writer was live made every later deletion
  fail on it, so the age sweep and the `MaxBytes` and `MinFreeBytes` caps stopped until the process
  restarted; a segment that is already gone is now counted as deleted and retention goes on.
- Fixed: the manual corruption procedure in the user guide (section 10) failed at `quill recover`
  after truncating the newest segment, because `quill.tail` still named the old length; it now
  says to rewrite `quill.tail` (and to read the last valid sequence first).
- Fixed: a `quill.tail` whose `off` was 2^63 or more made every reader call panic with a
  slice-bounds error (exit status 2 from the command line tool); an `off` above the 1.5 GiB
  segment limit is now refused as corrupt.
- Fixed: `quill tail -follow` held its output in a 4 KiB buffer, so to a pipe or file nothing
  appeared until the buffer filled or the command was interrupted; each record is now written
  as it is read.

## v0.1.0

The first release: the core, built in four phases from the design in `docs/design/`.

- The on-disk format `quill/1`: CRC-framed segments, `quill.tail`, `quill.format`,
  `quill.policy`, one file and one lock per consumer (`docs/format.md`).
- `Log`: `Open` with the creation protocol and the directory lock, `Append`, `AppendBatch`,
  `Flush`, `Sync` (group commit), `Roll`, `Read`, `ReadWait`, `Scan`, `ScanAll`, `Records`,
  `Follow`, `OffsetForTime`, `Stat`, `Prune`, `Close`; parse-forward recovery with hole
  detection; the failure classes of a flush.
- `Reader`: lock-free reading from another process, bounded by `quill.tail`, with `Stat`,
  `Check`, consumers, iterators and seek by time.
- Consumers: named offsets with per-consumer locks and protection; `ProtectedConsumers`;
  `ConsumerTTL`.
- Retention: the age sweep with the clock guard, `MaxBytes`, `MinFreeBytes`, `MaxRecords`,
  age rotation, `Dropped`.
- `Archive`, the `_archive` protected consumer.
- The `quill` command: `ls`, `stat`, `read`, `tail`, `cat`, `append`, `consumers`, `commit`,
  `register`, `unregister`, `check`, `prune`, `roll`, `archive`, `recover`; `-json` output.
- The crash simulator (`internal/fsx`) and the test suite T1 to T27; benchmark baselines.

From the A1 adversarial audit, fixed before the tag:

- Fixed: `Close` racing a `Read`, `Follow`, `Seek` or `Archive` no longer crashes the reader with a
  nil dereference; a read in flight finishes and the last one out closes the active segment.
- Fixed: a damaged sealed segment now counts its file's bytes in `Stat().Bytes` and toward `MaxBytes`
  (and `MinFreeBytes`), so the cap can delete it when it is the oldest.
- Fixed: the offline verbs (`append`, `roll`, `prune`, `recover`) no longer rewrite `quill.policy`
  (`Open` skips it under `NoMaintenance`), and `append`, `roll` and `recover` take the segment size
  from it instead of the 8 MiB default, so they do not seal a larger active segment.
- Fixed: `quill append` closes the log before printing and exits 1 when `Close` fails (it exited 0
  after a failed final flush with `-sync=false`); with `-sync=true` it still prints the last
  sequence, which was already durable.
- Fixed: a `Reader` whose listed sealed segment was deleted before it could open it (retention
  racing a read) now returns a `*GapError` over the segment's range from `Read`, `Follow`, `Seek`
  and `Archive`, instead of a raw path error.
- Changed: `Open` and a `Reader` build the holes and detect overlapping segments through one helper;
  the overlap `ErrCorrupt` message is now the same for both, and a bad first segment before a log
  that starts at sequence 1 no longer yields an empty hole.

From the allocation pass that followed the audit (three finders, two skeptics per finding, every
change measured before and after in `docs/audits/benchmark-baselines.md`; 22 candidates, 9
refuted, 12 applied, 1 deferred):

- Changed: `quill.tail` is encoded without `encoding/json` and, when its bytes are the canonical
  shape the writer produces, decoded without it too; any other input still takes the strict
  decoder with the same verdicts and messages (decision 105). The file's bytes are unchanged.
- Changed: the marker's path and the sealed segments' read references are built once and reused
  (decision 103); a `Read` sizes its result slice once (decision 104); the flush broadcast channel
  is made only when something waits; `fsx.OS.Rename` calls `rename(2)` directly; the atomic file
  writer's temp name costs one allocation; a scan no longer builds an index it discards; a
  `Reader`'s view probes the active segment's first frame in a pooled buffer and allocates its
  references as one slab. A flush went from 14 to 6 allocations, `Read(offset, 100)` from 136 to
  114, a `Reader` poll from about 540 to about 165, and a full scan from 104 to 47 per
  benchmark iteration.
- Changed: the `quill` command's printer builds each record in a reused buffer, zero allocations
  per record; the output bytes are identical (a differential test against the old formulation).
