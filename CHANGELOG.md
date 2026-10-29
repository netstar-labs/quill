# Changelog

All notable changes to quill. The format follows Keep a Changelog; versions are `v0.y.z`
for ever, and the on-disk format is versioned separately in `quill.format`.

## Unreleased

- Fixed: `Close` racing a `Read`, `Follow`, `Seek` or `Archive` no longer crashes the reader with a
  nil dereference; a read in flight finishes and the last one out closes the active segment.
- Fixed: a damaged sealed segment now counts its file's bytes in `Stat().Bytes` and toward `MaxBytes`
  (and `MinFreeBytes`), so the cap can delete it when it is the oldest.

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
