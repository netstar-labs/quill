# Changelog

All notable changes to quill. The format follows Keep a Changelog; versions are `v0.y.z`
for ever, and the on-disk format is versioned separately in `quill.format`.

## Unreleased

- Fixed: the CLI's offline write verbs (`append`, `roll`, `recover`, `prune`) opened the log with no `Perm`, so each rewrote `quill.tail` (and `roll` and `append` created the new segment) with mode 0600 in a log created with `Perm` 0660. Because a `Reader`, the CLI's `register` and `Archive` take their mode from `quill.tail`, one routine `roll` or `recover` switched the group arrangement off. The verbs now keep the mode `quill.tail` already has (never below 0600).

- Fixed: `Open` and `OpenReader` kept a relative directory relative, so a later `os.Chdir` in the process redirected every later file operation: the writer's flushes failed with `ENOENT`, or wrote the marker and segments into an unrelated directory holding the same relative path. The directory is now made absolute when the log is opened.
- Fixed: a `Reader` or consumer kept open while its log directory was replaced by a different log never re-checked the log id, so `Consumer.Commit` wrote a consumer file stamped with the old id into the new log (a fresh open then failed `ErrWrongLog`, and the file pinned the new log's retention as a broken consumer). `Commit` and the wait inside `ReadWait` and `Follow` now re-read `quill.format` and return `ErrWrongLog` on a mismatch. Plain `Read` and `Scan` still do not re-check, to keep their cost; close and reopen after replacing a log.

- Fixed: the duplicate-key check lost track of the enclosing object after any array or object member, so a repeated key following a container-valued member (`{"a":[],"b":1,"b":2}`) went undetected. No path reaches it today: `quill.tail` has four integer fields, so a container value is already a decode error. The check is now correct for any document.

- Fixed: a segment whose frames climbed to sequence 2^64-1 and carried on at 0, 1, 2 passed a writable `Open`, because the contiguity test (previous + 1) wrapped to 0. `Open` then
  reported a `First` above `Head` and the next append was numbered 3. Nothing is now accepted after a frame numbered 2^64-1, so such a log is `ErrCorrupt`. No log quill writes
  contains one: the writer stops at `ErrSeqExhausted` before 2^64-1.

- Fixed: a frame whose varints were not minimal (a length prefix padded as `0x8c 0x00`, say) passed `Check` and a writable `Open`, but a read that ended a batch after it
  resumed one byte inside the frame and reported `ErrCorrupt` ("torn frame") or, in a sealed segment, a `GapError`. The decoder now refuses a non-minimal varint, as
  `docs/format.md` always specified, so `Check`, `Open` and every read agree that such a frame is corruption. No log quill writes contains one.

- Fixed: the documentation said the writer's segments and the log's directories took `Config.Perm` exactly. They are created through the process umask and never chmodded, so
  `Perm` 0660 under umask 022 gives `0640` segments and `0750` directories (state files and the segments `Archive` copies are exact); a group that needs write access to
  `consumers/` needs umask 002 for the writer. Behaviour is unchanged, only the documented claim. Decisions 51 and 74 are amended.

- Fixed: a `quill.tail` with `maxseg` at or above 9999999999 was accepted, so the next rotation created an eleven-digit segment name that no directory scan
  recognises (every later `Open` then failed with `ErrCorrupt`), and a `maxseg` of 2^64-1 wrapped the next id to 0 so every rotation failed. `maxseg` (and so `seg`)
  above 9999999999 is now a malformed marker, and the writer refuses to seal into an id above it. No log quill produces is affected.

- Fixed: at every segment seal the writer read every file in `consumers/` while holding the log's lock, to count `Dropped`, so a log with many consumer files
  (they are never expired, and any process that can create one can add more) stalled every `Append`, `Flush`, `Read` and `Sync` at each rotation: about 0.5 s
  with 20,000 files and 4.7 s with 100,000 (macOS). The seal now counts `Dropped` against the offsets the last prune pass (or `Open`) read and lists nothing,
  with or without a cap. `Dropped` can therefore lag a consumer that moved or was created since that pass by up to `PruneInterval`; `Prune` refreshes it.
  Decision 12 is amended.

- Fixed: `Open` accepted a `Config` whose `MaxBatchBytes`, `MaxBufferBytes` or `FlushBytes` was near `math.MaxInt64` (a natural "unlimited"), because the sum that
  `validate` compares to the 1.5 GiB segment file limit wrapped negative. A single `AppendBatch` over 1.5 GiB then wrote an oversize segment that `Sync`
  acknowledged and every reader reported as an unavailable range. Each term is now capped before the sum, so such a `Config` is refused with `ErrConfig`.

- Fixed: `quill append` with no arguments read the whole of stdin into memory before `MaxBatchBytes` was checked, so memory was proportional to the input (100 MB of
  newlines took about 4 GB, and a pipe that never ends grew until the process was killed). It now applies the same bound while it reads and refuses the batch,
  with the same `payload too large` error, as soon as the sum of `len(line) + 34` passes the limit, after at most one more scanner buffer. Nothing is appended.

- Fixed: a FIFO planted as `quill.tail`, `quill.format`, `quill.policy` or an entry of `consumers/` blocked the read-only `open(2)` that was meant to
  refuse it, because `readState` checked the file type only after the open returned. `Open`, `OpenReader`, `ReadPolicy` and the command line verbs hung, and
  a FIFO in `consumers/` froze the writer at the next segment rotation (the consumer listing runs under the log's mutex), so `Append`, `Sync` and `Close`
  hung with it. Every read-only open now returns at once and the FIFO is reported as damage, as a segment's always was. Decision 74 is amended.

- Fixed: the Go toolchain named in `go.mod` was pinned exactly only by `hack/gate.sh`; `build/quill` used whichever `go` was on `PATH`, and
  under `GOTOOLCHAIN=auto` a `toolchain` line is only a minimum, so the release binaries and the SBOM's `go.version` could come from a newer
  compiler than the gate tested. `build/quill` now exports `GOTOOLCHAIN` derived from `go.mod` exactly as the gate does, and a test fails if it
  does not. CI needed no change: the pinned setup action already installs the toolchain `go.mod` names. Decision 88 is amended.

- Fixed: a `quill.policy` of `{}`, `null` or only unknown fields parsed as a valid policy of zeros, so `quill ls` showed `RETENTION 0s` and
  `quill prune` ran with the 24 h default and no caps, as a version-skewed writer's file would have caused. A policy with no `retention` is now
  unparseable like a damaged one: `ls` shows `?` and "policy unreadable", `prune` refuses without explicit flags, `check` warns, and
  `ReadPolicy` returns `ok == false`. Every writer writes `retention` (an off one as `-1ns`), so no policy a writer produced is affected.
  Decisions 29, 48, 68 and 87 are amended.

- Changed: documentation only, no behaviour change. The clock guard limits an age sweep to at most a quarter of the sealed segments per pass rounded up (2 of 5,
  4 of 15, at least one), not "at most 25%" as the docs and the `Prune` comment said; they now state the rounding. Decision 44 is amended.

- Changed: `quill stat -json` wrote each hole as `{"From":N,"To":N}` and a log with no holes as `"holes":null`, the only capitalised keys and the only
  `null` list in the command line's JSON. It now writes `{"from":N,"to":N}` and `"holes":[]`. A script that read `From`, `To` or a null `holes` must
  change. The `stat`, `check` and `consumers` JSON shapes are now documented (scope.md, "Readable output"). Decision 35 is amended.

- Changed: documentation only, no behaviour change. `Read`, `ReadWait`, `Records` and `Follow` check `MaxReadBytes` after each record, so a
  batch stops once at least `MaxReadBytes` of payload have been returned and can exceed it by up to one record (at the defaults, 16 MiB plus up to
  64 MiB - 32); the docs and the `Config` and `ReaderOptions` comments said "at most `MaxReadBytes`". They now state the actual rule, and a read
  still returns at least one record. Decisions 50 and 79 are amended.

- Fixed: `quill cat`, `read` and `tail` with `-json` passed DEL (0x7f) and the C1 control characters U+0080 to U+009F, such as the CSI introducer
  U+009B, to the terminal raw inside the `payload` string, although the guide says no raw control character reaches it and the text form escaped them.
  They are now written as `\u007f` and `\u0080` to `\u009f`; the JSON stays valid and lossless. The bytes `-json` emits change for payloads that
  contain those characters. Decision 35 is amended.

- Changed: when the age sweep stops at a sealed segment whose newest record is stamped more than a day in the future (a clock that stepped ahead
  and was later corrected), the prune pass now logs a warning naming the segment; it used to log nothing while that one segment kept every newer
  segment from expiring by age until the clock reached the stamp. The prefix rule itself is unchanged and is now documented beside the clock
  guard. Decision 44 is amended.

- Fixed: `Open` (and so the CLI's writer verbs) refused a `Dir` that is a symbolic link to a directory, and a `Dir` that does not exist yet below
  such a link, with `ErrNoLog` "is not a directory", while `OpenReader` and a link above an existing `Dir` worked. A link in `Dir` is now followed: it
  is resolved once at open and the log lives in the target. A link that leads to a file or to nothing is `ErrNoLog` saying it is a symbolic link, and
  every file inside the log directory is still opened without following a link. Decisions 51 and 62 are amended.

- Fixed: a `quill.tail` that repeated a key in another letter case, such as `{"seq":1,"SEQ":2,...}`, was read as `seq` 2 with no error, because the
  duplicate check compared exact strings while the decoder matches field names case-insensitively. It is now refused as a duplicate key, like a
  byte-identical repeat. The writer never produces one. format.md now says which non-canonical shapes the strict parser still accepts
  (whitespace, field order, a missing trailing newline, a field name in other case). Decisions 72 and 105 are amended.

- Changed: documentation only, no behaviour change. `MaxBatchBytes` was described as the largest encoded batch, but the check sums
  `len(payload) + 34` per record (an upper bound, since the sequence is unknown at the check), so 500,000 one-byte records (about
  7.5 MB encoded) are refused at the default and about 479,000 are accepted. The docs, the `Config` comment and `quill append`'s help now say so,
  and that `quill append` reading stdin is one batch subject to it (split a larger input across several calls). Decision 46 is amended.

- Fixed: `Archive` into a destination whose `quill.format` is newer than this build (`quill/2`) returned `ErrCorrupt`, where `Open` and
  `OpenReader` return `ErrFutureVersion` as decision 6 says; it now returns `ErrFutureVersion`. Any other unusable destination marker is still
  `ErrCorrupt`. Decisions 6 and 85 are amended.

- Changed: `-from`, `-limit` and `-n` (`read`, `cat`, `tail`) take plain decimal integers, like `commit -offset`, `prune -max-bytes` and the
  `@unix` times. They used to parse in base 0, so `-n 010` printed 8 records (octal), `-from 0x10` started at 16 and `-limit 1_0` meant 10;
  now `010` is 10 and `0x10`, `1_0`, `0b1`, `0o7`, signs and exponents are usage errors (exit 2). Decision 96 is amended.

- Changed: a verb other than `append` and `help` given a positional argument (`quill -dir L read 100`, `quill -dir L tail 5 -json`,
  `stat -json extra`) now exits 2 with `unexpected argument ...; flags go before arguments`. It used to ignore the argument, and flag parsing
  stops at the first positional, so every flag after it was ignored too: `read 100` printed from the first retained record and
  `tail 5 -json` printed text, both exiting 0. Put flags before arguments, and give `read` its offset as `-from`. Decision 53 is amended.

- Fixed: a consumer file created through a `Reader` or the CLI (`register`, `commit`, `archive`) was always mode 0600, and so were the
  files and directories `Archive` created, so in the group arrangement the scope describes (`Config.Perm` 0660, a consumer in another
  account) the writer could not read the consumer's file, retention protected every segment and `check` reported UNHEALTHY. They now take the
  log's mode, read from `quill.tail`; a default-mode log still gets 0600, and a lock file is created through the umask. No new option or
  flag. Decisions 41, 51 and 67 are amended.

- Fixed: the archiver's `_archive` consumer could not be rewound or removed, so after a restore left it ahead of the head `quill archive` failed
  with `ErrAhead` and the documented remedy, `quill commit`, was refused for it by name. `Consumer`, `DeleteConsumer` and the CLI's
  `register`, `unregister` and `commit` now accept exactly the name `_archive`; every other leading-underscore name, and
  `Config.ProtectedConsumers`, still refuse it. Decisions 16, 41 and 67 are amended.

- Fixed: an I/O error while reading a segment (`EIO`, for example) was reported as `ErrCorrupt` (the active segment) or as a
  `*GapError` (a sealed one), so a transient fault looked like loss and a consumer that resumed from `To + 1` skipped the
  records. `Read`, `Scan`, `ScanAll`, `Records`, `Follow`, `OffsetForTime` and consumer reads, on a `Log` and on a `Reader`,
  now return the I/O error itself (`errors.Is(err, syscall.EIO)`), and so does a `Reader`'s read of a segment's first frame.
  `Open` now fails with the error when it meets one verifying a sealed segment, as it did for the newest segment, instead of
  recording a hole that lasted until the next `Open`. `ErrCorrupt` and `*GapError` stay for a bad or torn frame, a damaged
  file or a file retention deleted, and a sealed segment that cannot be opened at all (too many open files, permission
  denied) is still a hole for that one view of a `Reader`. Nothing is remembered: the next call reads again.
  Decisions 38, 39, 79, 94 and 98 are amended.

- Fixed: while creating the next segment kept failing, the writer kept appending to the active segment with no bound, so it could
  pass the 1.5 GiB limit that `Open` and readers enforce and the log could then not be opened although every record had been
  acknowledged durable. Now, while a rotation that is due keeps failing, an `Append` or `AppendBatch` that would take the segment
  past `SegmentBytes + max(FlushBytes, MaxBufferBytes) + max(MaxBatchBytes, MaxPayloadBytes + 34)` is refused with the rotation
  error and assigns no sequence, and `Open` refuses (`ErrConfig`) a `Config` whose `SegmentBytes` plus that overshoot reaches the
  limit (for example a 1 GiB `SegmentBytes` with a `FlushBytes` or `MaxBufferBytes` above 512 MiB). A segment file over the limit
  is now reported as refused unread, not as "damaged at byte 0 (intact to byte 0)". Decisions 16 and 60 are amended.

- Fixed: a `Log`'s own `Read`, `Scan`, `ScanAll`, `Records`, `Follow` and consumer reads returned a raw `*fs.PathError` (no such
  file) when retention deleted a sealed segment after the read took its view and before it opened the segment, where a `Reader`
  returns a `*GapError`. They now return a `*GapError` over that segment's range, starting at the offset asked for, so the
  documented recovery (resume from `To + 1`) applies. Decisions 101 and 103 are amended.

- Changed: `Config.Logger`'s handler, `Enabled` included, runs with the log's lock held, and the documentation now says so: a handler
  that calls back into the `Log` deadlocks it and one that blocks stalls `Append`, `Read`, `Sync` and the flusher, so it must hand
  the record to another goroutine and return. Behaviour is unchanged; `Config.Logger`, the user guide and the scope document
  state the rule.

- Fixed: `Open` with `Config.ProtectedConsumers` failed with `ErrLocked` after 200 ms whenever a consumer process held that
  consumer's lock, so a writer could not be redeployed while its protected consumer ran, and `quill prune`, which fed the
  policy back in, failed the same way with a message about the writer's lock. `Open` now accepts an existing consumer file
  that already says protected, parses, belongs to the log and is not ahead of the head, without taking its lock; a missing
  file is still created and an unprotected one still switched on under the lock (`ErrLocked`, naming the consumer's lock,
  if a process holds it), and a file ahead of the head is still refused with `ErrAhead`. The offline verbs, `prune`
  included, now open without the policy's `ProtectedConsumers`.

- Fixed: the documented rewind of a consumer ahead of a restored log, `quill commit`, failed with `ErrAhead` because it opened
  the consumer first. `ConsumerOptions` gains `Rewind`, which opens such a consumer so `Commit` can rewind it to any offset in
  `[first, Head + 1]`; `quill commit` sets it, keeps the protect flag, and `register` and the default open still refuse an
  ahead consumer. `quill consumers` now notes a consumer ahead of the head (`-json` adds `"ahead": true`) and `quill check`
  prints a non-fatal warning for it; the log stays HEALTHY.

- Fixed: CI supply-chain hygiene. `pr-issue-link.yml` declared no `permissions:` and so ran with the repository's default
  token scope; it now declares `permissions: {}`. The `actions/checkout` and `actions/setup-go` pins (still exact commit
  SHAs) moved from v4.4.0 and v5.6.0, which run on the node20 action runtime, to v7.0.1 and v7.0.0 (node24). A
  `dependabot.yml` for `github-actions` now proposes pin updates, and the meaningless `GOPRIVATE` line is gone from `ci.yml`
  (the module has no dependencies). CI configuration only; the library and the tool are unchanged.

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
  GitHub login). It now names the Security tab "Report a vulnerability" route (GitHub private vulnerability reporting) and
  states that only the latest release is supported. Documentation only; behaviour is unchanged.

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
