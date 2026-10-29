# Changelog

All notable changes to quill. The format follows Keep a Changelog; versions are `v0.y.z`
for ever, and the on-disk format is versioned separately in `quill.format`.

## Unreleased

Nothing yet.

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
