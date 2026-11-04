# Changelog

All notable changes to quill. The format follows Keep a Changelog; versions are `v0.y.z`
for ever, and the on-disk format is versioned separately in `quill.format`.

## Unreleased

### Breaking changes and upgrade notes

Changes since `v0.1.0` that a script, a Go caller or an operator may need to act on, taken from the entries below; the on-disk format `quill/1` is unchanged. The list is drawn from 86 of the 144 entries below (the other 58 are documentation, tests, performance or internal tooling); each line names the entry's subject, and the entry gives the reason and the audit id.

Go callers:

- An unkeyed composite literal of `ConsumerOptions`, `Problem`, `SegmentError` or `Stats` no longer compiles because each gained a field, so build them with field names; no function or method signature changed.
- `Open` returns `ErrConfig` for a `MaxBufferBytes` below `FlushBytes`, for a `MaxBatchBytes`, `MaxBufferBytes` or `FlushBytes` near `math.MaxInt64`, and for a `SegmentBytes` that with the largest flush reaches the 1.5 GiB segment limit, so set real sizes.
- `Read`, `Records` and `Follow` count 48 bytes per record against `MaxReadBytes`, so `Read(from, 0)` may return a first batch instead of the whole tail; call again from the next sequence.
- `ReadWait` and `Follow` at the head of a failed log return `ErrFailed` instead of blocking until the context ends.
- `Open` writes `quill.tail` only when recovery moved it and may return with the publish pending, so check `Err()` after `Open`; it also sets an existing `quill.tail` to `Config.Perm` in place.
- `Err()` reports what is failing now and can return to nil after recovery (`errors.Is(err, ErrFailed)` is the terminal state), and it wraps `ENOSPC` or the `statfs` error while the `MinFreeBytes` floor cannot be met or checked.
- `Prune` returns the unlink error where it returned nil, so treat a non-nil `Prune` error as "the pass did less than asked".
- `AppendBatch` and `Append` return the first assigned sequence with the error when the inline flush's fsync fails, where they returned 0, so do not resubmit records on a non-zero sequence with an error.
- A read from the beginning, `ScanAll` and a new consumer return the `GapError` of a damaged or empty oldest segment where they started silently at the first good one, so resume with `Commit(To + 1)` or read from past it.
- A read I/O error (`EIO`) is returned as itself, not as `ErrCorrupt` or a `*GapError` (and a `Reader` retries a transient one instead of remembering it), and a `Log` read of a segment that retention just deleted returns a `*GapError` where it returned a `*fs.PathError`.
- `Open`, `OpenReader`, `Check` and `Archive` refuse as `ErrCorrupt` a non-minimal varint, a frame after one numbered 2^64-1, a `quill.tail` with `maxseg` above 9999999999 or `off` above 1.5 GiB or a key repeated in another case, and overlapping sealed segments (`Check` reports kind `overlap`), and a `quill.policy` with no `retention` reads as unreadable; sequence 2^64-1 is never assigned.
- `Archive` refuses a destination that is the source log, a populated directory that is not a log (`ErrNoLog`), or one whose `quill.lock` is held or cannot be opened (`ErrLocked`), and a newer destination format is `ErrFutureVersion`, not `ErrCorrupt`.
- `Archive` returns the `*GapError` for a loss between the destination's last record and the first retained sequence, also after `quill unregister -name _archive`.
- `quill.lock` is always created 0600 and `Open` tightens an older one, so a writable `Open` under another account fails with the permission error on `quill.lock`, not `ErrLocked`; `Open` with `Config.ProtectedConsumers` no longer fails with `ErrLocked` while a consumer process runs.
- `Open` and `OpenReader` make a relative `Dir` absolute and follow a symbolic link in `Dir`, and a FIFO or oversize state file is reported as `ErrCorrupt` instead of hanging or matching no sentinel.
- `Consumer` and `DeleteConsumer` accept the name `_archive`, `ConsumerOptions.Rewind` opens a consumer ahead of the head, and `Consumer.Commit` returns `ErrWrongLog` if the directory was replaced by another log.
- `Stats.Dropped` may lag a consumer by up to `PruneInterval` (`Prune` refreshes it), and `Stats` has six new fields (`DiskFull`, `FreeBytesAtLastPass`, `BufferedRecords`, `BufferedBytes`, `FlushFailing`, `LastFlushOK`); `Stat().Holes` no longer reports an empty hole, `FlushFailing` is false once the log has failed, and `MarkerAge` counts from `Open`.
- Retention and recovery failure paths changed, each a fix with no migration: the `MinFreeBytes` estimate credits the segments the pass already deletes, a prune pass survives a failed age rotation, syncs the directory before each unlink and counts an already-missing segment as deleted, a refused `Open` no longer cuts the newest segment, a failed `quill.policy` write at `Open` removes the stale file, and `Archive` removes stale `.tmp-*` files, syncs a new destination into its parent and replaces a damaged same-size segment on a rerun.
- A writer whose `quill.lock` was deleted or replaced fails with `ErrFailed` at its next flush, and a log that fails writes one `Error` record to `Config.Logger`.
- `Config.Logger` receives new `Warn`, `Info` and `Error` records (background failures on entry and recovery, a cut of the newest segment, a deleted damaged segment, a consumer ahead of the head or behind the first retained sequence, a failed staleness probe, a segment stamped in the future).

Command line:

- `-from`, `-limit` and `-n` take plain decimals, so `010` is ten (it was octal eight) and `0x10`, `1_0`, signs and exponents exit 2.
- Any verb except `append` and `help` given a positional argument (`read 100`, `tail 5 -json`) exits 2 where it ignored the argument and every flag after it, so put flags first.
- `quill roll`, `recover` and `prune` with a retention or cap flag at a directory that is not a log exit 1 and create nothing, where they exited 0 and created a log.
- `quill ls` prints an `error:` row and exits 1 for a log it cannot open, where it skipped it and exited 0; only a directory that is not a log is skipped.
- `quill prune` and `roll` exit 1 when `Close` fails, and `prune` exits 1 when an unlink failed or a consumer file could not be read, and its output line now ends `, dropped N`.
- `quill read`, `cat` and `tail` exit 1 on the `GapError` of a damaged oldest segment, where they started at the first good segment and exited 0.
- `quill cat`, `read`, `tail`, `stat`, `check`, `ls` and `consumers` exit 1 when writing to standard output fails, where a short output exited 0.
- `quill archive` exits 3 when the source had lost records the archive never copied, where it exited 0, and exits 1 for a populated destination that is not a log, where it wrote into it.
- `quill check` exits 3 for a sealed segment truncated to zero bytes (it printed HEALTHY), reports an `overlap` problem, no longer reports UNHEALTHY or a consumer-ahead warning for a change made during the scan, and with `quill consumers` now notes a consumer behind the first retained sequence or ahead of the head (`-json` adds `behind_first`, `stale` and `ahead` only when true).
- `quill stat -json` writes holes as `{"from":N,"to":N}` and an empty list as `[]`, where it wrote `{"From":N,"To":N}` and `null`.
- `quill cat`, `read` and `tail` with `-json` escape DEL, the C1 controls and the Unicode format and bidi characters in `payload` as `\uXXXX`, so a script that matched the raw bytes must decode the line.
- `quill ls`, `consumers`, `check` and error messages quote a name from the disk that holds a control, bidi or invalid UTF-8 character, `quill append`'s oversize-stdin message changed text, and a library error prints one `quill:` prefix, not two.
- The offline verbs print on standard error the warnings a writable open logs (a `quill.lock` that could not be tightened, a cut of the newest segment, a deleted damaged segment, a deletion past a consumer), and `quill tail -follow` now writes each record as it is read.
- The offline verbs (`append`, `roll`, `recover`, `prune`) now keep a log's `Config.Perm` mode on `quill.tail` and new segments, and a consumer file or an archive made through a `Reader` or the CLI takes the log's mode.

Release builds:

- `build/quill` pins the toolchain, `GOAMD64=v1` and `GOARM64=v8.0`, and puts the commit time, not the clock or the builder, into the archives and the SBOM, so a verifier builds with `-buildvcs=false` or inside a checkout of the tag.

### Entries

- Changed: documentation only, no behaviour change. The list above, and the migration table in `docs/audits/release-v0.2.0.md`, are rebuilt from the entries below and from `git log v0.1.0..HEAD`: the release record had listed 8 breaking changes and called the unkeyed-literal break on `Problem` and `SegmentError` the only compile-time one, and it now has one row for each of 86 of the 144 entries below that a script, a Go caller or an operator may see (the other 58 are listed there by line as not user-visible; the classification is from each entry's own words and most rows were not re-run, as the record says, so the list is a best effort, not a proof of completeness), and names the four types (`ConsumerOptions`, `Problem`, `SegmentError`, `Stats`) that stop an unkeyed composite literal compiling, from a diff of the exported declarations at `v0.1.0` and at this tree (audit L4b-1, owner-approved 2026-11-04).

- Changed: documentation only, no behaviour change. The `Config.Logger` rule no longer lists the warnings `Open` gives itself (the list had missed the marker-publish and rotation Warns and the records of the seal recovery may make, and would have gone stale again). The user guide, the scope, decision 50 and the `Config.Logger` doc now say that the records `Open` itself writes run on the caller's goroutine before `Open` returns, before there is a `Log`, with no lock held, that every other record runs with the lock held, and that the handler rule (do not call the `Log`, do not block) still applies to both (audit A1d-9, owner-approved 2026-11-04).
- Fixed: `build/quill` did not pin the compiler's architecture level, so a releaser with `GOAMD64` or `GOARM64` set (in the shell, or by `go env -w`) shipped binaries built for AVX2 or ARMv9 instructions: not the baseline build, not reproducible by a verifier on the defaults, and able to die with SIGILL on an older CPU. The script now exports `GOAMD64=v1` and `GOARM64=v8.0` (the toolchain defaults), so the default hashes are unchanged for a default builder. Decision 88 carries the note; a test pins it (audit A4r-2).
- Changed: source compatibility, no behaviour change. `Problem` and `SegmentError` each gained an unexported field since `v0.1.0` (`refused` and `oversize`, which mark a segment file refused whole, unread, for being over the size limit), so `go doc` shows "Has unexported fields" for them and an unkeyed composite literal of either from another package (`quill.Problem{"segment", path, 0, 0, "d", true}`) no longer compiles. Both types are only produced by quill and returned to callers, and `go vet` already flags unkeyed literals of imported structs; build one, if a test needs it, with field names (`quill.Problem{Kind: "segment", Path: path}`). Their comparability and every exported field are unchanged (audit A1d-3).
- Fixed: `Open` of an existing log with a different `Config.Perm` left `quill.tail` at its old mode until the first flush that moved the marker (audit A1d-2, a regression from the L3b-8 change below, which removed the marker rewrite that applied `Perm` at every `Open`; `quill.policy` was unaffected and changed at once, and `v0.1.0` changed both). On an idle log a restart with `Perm` 0640 left the marker 0600, so a group reader could not read it and a `Reader` or `Archive` created its files 0600, and a restart with `Perm` 0600 left it group-readable. `Open` now sets `quill.tail` to `Perm` in place (an `fchmod`, which needs no free block, so the full-disk restart is unaffected); if the chmod fails because another account owns the file, `Open` still succeeds and logs a `Warn`. Decisions 39 and 51, the `Config.Perm` documentation, the user guide and the scope say so.
- Changed: documentation only, no behaviour change. `docs/design/review.md` and decisions 3, 5, 7, 15, 16, 20, 21 and 32 no longer carry any line count, ratio, timing, rate or framing size taken from the reference implementation: each says what the reference measured (faster or slower) and keeps the design conclusion, with a dated note, and quill's own measurements stay in `docs/audits/benchmark-baselines.md`. Decision 90 now says what was done (it had said the figures were already replaced or softened), the framing cost per record is now one figure computed from quill's own encoder (12 to 18 bytes) in decision 32, scope.md and format.md, and review.md's size ceiling, derived from the reference's line count, is replaced by quill's own count of the root package.
- Changed: a writable `Open` of an existing log no longer needs a free block when the marker is already right. It used to rewrite `quill.tail` through a temp file every time, so on a completely full filesystem `Open`, and `quill prune`, `roll`, `recover` and `append` with it, failed with `ENOSPC` before the `MinFreeBytes` and `MaxBytes` sweeps could run, and the log stayed unopenable until something outside quill freed a block (audit L3b-8, owner-approved 2026-11-03). `Open` now writes the marker only when recovery moved the position (a crash left it behind the data). If that write fails, `Open` still returns the log with the marker publish pending, as after a failed flush: `Err()` wraps the cause, `Config.Logger` gets the `marker publish` `Warn`, a `Reader` sees the older head, and the flusher, `Close` and prune passes retry; until then a prune pass, and the `MinFreeBytes` sweep, deletes the sealed segments below the one the marker on disk names and keeps that one and every later one, so the sweeps that free space still run (decision 65). A caller that took a nil error from `Open` to mean "the marker is published" should check `Err()`. `Stats.MarkerAge` of a writer counts from `Open` when it wrote no marker. Decisions 39, 60 and 81 carry the note.
- Changed: documentation only, no behaviour change. A writable `Open`, or `append`, `roll`, `prune` or `recover`, under a different account than the one that created the log fails with the permission error from opening `quill.lock` (EACCES, always 0600; the message names the file and matches `fs.ErrPermission`), not `ErrLocked`, whether or not a writer is live; root, which bypasses file permissions, gets `ErrLocked`. `prune` with no retention flags stops earlier, on an unreadable `quill.policy`. A script that detects a live writer should treat `ErrLocked` and that permission error alike as "this is not this account's log, or a writer may be live". `Archive` still returns an error that is both, because it only refuses. The `Open` and package godoc, the user guide, the scope (guarantee 5) and decision 51 say so; a test pins the error through the `fsx` seam (audit L3b-7, owner-approved 2026-11-03).
- Fixed: documentation only. The user guide, scope and decision 51 said a durable flush costs about 20 ms on macOS, and decision 39 priced a third fsync at about 20 ms, while the recorded baselines (`docs/audits/benchmark-baselines.md`) give about 10 ms for the flush's two `F_FULLFSYNC`s (about 5 ms each); the scope's `Read(limit 100)` range was 23-39 µs against the recorded 21-36 µs. All now match the baselines (audit A7-23).
- Changed: documentation only, no behaviour change. Section 9 of the user guide has a table of the log records and errors that need an operator action (message, meaning, action): `log failed`, the `quill.lock` that could not be tightened, `cannot read free space`, free space below `MinFreeBytes`, `quill.policy` not written, an unreadable consumer file, the clock guard and `ErrFutureVersion`. The `Config.Logger` row points to it (audit A7-20).
- Changed: documentation only, no behaviour change. The user guide (section 9) now has an "Upgrading and rolling back" paragraph: the format is `quill/1` and unchanged, so replacing the binary needs no migration and the old binary can be put back; which processes a future format change would require stopping (a running writer, and a reader that only calls `Read`, `Scan` or `Head`, never re-reads `quill.format`; `Consumer.Commit` and a `Reader`'s waits do); where the compatibility evidence is (`hack/skew.sh`, `docs/audits/format-compatibility.md`); and that `ErrFutureVersion` means the binary is older than the log (audit A7-18).
- Changed: documentation only, no behaviour change. The README now has an Installing section: `go get` for the library, `go install` or a release archive for the command, that a go-installed binary reports `dev` for `quill version` (only the archives are stamped), and that the archives do not hold `docs/` (audit A7-19).
- Changed: no behaviour change. `Config`, its defaults and validation and the consumer-name rules moved from `log.go` to a new `config.go`. `log.go` was 555 lines at v0.1.0 and had grown to 845 during the audit; the move keeps no library file over about 600 lines, as the README, CONTRIBUTING and decision 20 say (audit A7-11).
- Fixed: the `MinFreeBytes` estimate did not credit the sealed segments the same pass already deleted for age (`Retention`) or for `MaxBytes`, so a pass only slightly below the floor deleted further segments, protected ones included, up to every sealed segment, though the segments already chosen made up the shortfall (five sealed segments, the oldest two expired, a reading one byte under the floor: three were deleted, and with a larger shortfall all five). The estimate now counts every segment the pass deletes, as `Config.MinFreeBytes` documents; `Stat().FreeBytesAtLastPass` is still the raw `statfs` reading. Decision 25 amended (audit A7-1).
- Changed: documentation only, no behaviour change. Decision 51 and SECURITY.md now say what the symlink rule covers: the final component of each state file's path, the one opened with `O_NOFOLLOW`. A symbolic link at an earlier component is followed, so a `consumers/` directory replaced by a symlink makes `register` create the consumer's files in the target and `recover` or `Open` remove `.tmp-*` files older than an hour there. A log directory or `consumers/` that an untrusted party can replace is outside the threat model, as write access to the log directory is (audit A3-7, owner-approved 2026-11-03).
- Changed: `quill cat`, `read` and `tail` with `-json` now escape the Unicode format and bidi controls in a `payload` as `\uXXXX` (a surrogate pair above U+FFFF), as text mode already did: the bidi overrides and isolates (U+202A to U+202E, U+2066 to U+2069, U+061C), the zero-width characters (U+200B to U+200F, U+2060 to U+2064), the byte order mark U+FEFF and the rest of category Cf (170 code points, among them U+00AD and the tag characters). The raw bytes of a `-json` line whose payload contains one of them change; the decoded value does not (a JSON parser returns the same string), and a payload without them prints exactly as before. A script that matched those raw bytes in the line text must match the escape or decode the line. The `time` field is unchanged. Decision 35, the scope and the user guide say so (audit A3-24).
- Changed: documentation only, no behaviour change. The `MinFreeBytes` floor overrides consumer protection, and the documents now say so with the exact rule for each cap. Protection holds segments back from the age sweep (`Retention`) only; `MaxBytes`, `MaxRecords` and `MinFreeBytes` delete the oldest sealed segments whether or not a protected consumer has read them, so a protected consumer can lose records. For `MinFreeBytes`, when the filesystem is below the floor (another process's disk use can put it there) the writer deletes this log's oldest sealed segments, counting each as freeing its size from one `statfs` read, and the `Warn` naming the consumer ("deleted segment past a consumer's offset; it has lost records") and `Stat().Dropped` are how the loss is seen. The `Config.MinFreeBytes`, `Config.ProtectedConsumers` and `ProtectOn` godoc, the package overview, the user guide, `scope.md` (guarantee 11) and `architecture.md` say it; decisions 25 and 57 are amended. A test pins it (a protected consumer, a `statfs` seam below the floor, a prune pass).
- Changed: documentation only, no behaviour change. `ConsumerTTL` staleness is derived from the clock and the filesystem: a protected consumer is stale when it is behind and the age of its offset file (its modification time against the writer's clock) and of the record at its offset are at least as old as the TTL, so a clock step, a restore that resets mtimes, a copy that does not preserve them, or a filesystem with coarse or wrong timestamps can make a live consumer look stale (and lose its protection) or a dead one look live. A forward step of at least the TTL makes every behind protected consumer stale at once; the clock guard (decision 44) then paces only the age sweep that follows, and the staleness check has no guard of its own. There is no per-consumer exemption (`Protect` is what subjects a consumer to the TTL); leaving `ConsumerTTL` off, the default, keeps protection whatever the clock does. The `Config.ConsumerTTL` documentation, the user guide, the design scope, the architecture and decisions 44 and 45 say so (audit A3-31).
- Changed: documentation only, no behaviour change. The README, user guide, scope, the godoc of `Open`, `Config.Dir` and the package, and SECURITY.md now name the filesystems where the single-writer lock and a `Reader`'s view are not coherent: NFS, SMB/CIFS, FUSE filesystems (sshfs, s3fs and the like) and a directory shared between VMs or containers with separate kernels (virtiofs, 9p, a host directory bind-mounted into a container). Quill does not detect them, and two writers there can both believe they hold the lock and corrupt the log with no error at write time; the log directory must be on a local filesystem (ext4, xfs, btrfs, APFS) with exactly one host or kernel writing. The old text, "bind mounts at the operator's risk", is corrected: a bind mount within one kernel keeps the lock exclusive.
- Changed: `quill roll` and `quill recover` at a directory that is not a log now exit 1, printing `quill: not a log: <dir> has no quill.format` as the read verbs do, where they exited 0; `quill prune` with a retention or cap flag does the same. Before, a mistyped `-dir` (one that does not exist, or an empty directory) got a new empty log, with every missing parent directory, and a populated non-log directory was refused but left an empty `quill.lock`; the three verbs now look for `quill.format` first and create nothing. `quill append` still creates the log on first use, as `Open` does. Decision 53, the user guide, the design scope and the command reference say so (audit A3-23).
- Fixed: `quill ls`, `consumers`, `check` and the error messages printed names taken from the disk (a log directory, a consumer file, a path built from one) raw, so a file name holding a newline or an ESC sequence could forge a table row or send terminal sequences to the operator. In text output such a string is now quoted with the escaping used for record payloads (a newline, ESC or other control, C1 or bidi control, or a byte that is not valid UTF-8 shows as an escape); a name of printable characters prints exactly as before, and `-json` is unchanged. Decision 35 and the scope say so (audit A3-3).
- Fixed: `Reader.Check` (and so `quill check`) beside a live writer could warn "consumer ... at N, ahead of the head M: rewind it with quill commit" for a healthy consumer. It read `quill.tail` first, scanned every segment, and read the consumer offsets last, so a consumer that committed to the grown head in between looked like a restored-log leftover (nine of fifteen checks on a log taking writes). When an offset is above the head it read at the start, `Check` now reads the marker again and warns only if the offset is above that head, which it names. A consumer that is really ahead is still reported. Decision 36 amended (audit L3b-14).
- Fixed: a consumer unregistered while `Reader.Check`, `Consumers` or a prune pass was reading the `consumers/` directory made `Check` (and so `quill check`) report a fatal `consumer` problem (UNHEALTHY, exit 3) on a healthy log, added a row with a "no such file" `Err` to `Consumers()` and `quill consumers`, and made the writer's prune pass warn and skip its age sweep. A consumer file that is gone by the time it is read is now skipped; a file that exists and cannot be read or parsed is still reported. Decision 41 amended (audit L3b-4).
- Changed: `MaxReadBytes` now bounds the size of a batch, not the sum of the payloads: each record counts its payload plus 48 bytes of overhead against it. A read of empty or tiny records used to return every record of the log in one call, however many (`Read(1, 0)` and each batch of `Records` and `Follow` on a log of 1,000,000 empty records returned all of them and allocated 215 MB against a 16 MiB bound); at the default it now returns about 349,000 empty records, or 342,000 one-byte ones, per call and allocates 70 MB for the 1,000,000-record log (measured with `-benchmem`, macOS arm64). A read still returns at least one record, a record whose payload passes the bound is returned alone, and `Scan` is unaffected. A caller that read with limit 0 and expected the whole tail of a log of small records now gets a first batch and must call again from the next sequence; `Records`, `Follow` and the command line tool continue across batches and lose and reorder nothing. Decisions 50, 58 and 79, guarantee 7, the scope, the user guide and the `Config` and `ReaderOptions` comments say so (audit L2-53).
- Changed: documentation only, no behaviour change. `Archive`'s comment, its doc and the user guide implied that a new or empty destination receives the whole source. The `_archive` offset belongs to the source, so a new or empty destination receives only the records at or above it and the run returns nil (exit 0): a destination archived after the consumer has advanced is a suffix, not a complete copy, and `quill check` on it reports it healthy. `quill commit -name _archive -offset N` rewinds the consumer to copy more. Decision 85 and the scope say so; a test pins it.
- Fixed: `Reader.Check` (and so `quill check`) reported a log HEALTHY when a sealed segment had been truncated to zero bytes (by a tmp cleaner or a bad restore, say), though `Open` reported its records as a hole and the writer's `Err()` named the file; garbage in the same file was already reported. `Check` now reports a sealed segment with no frames as a fatal `segment` problem ("a sealed segment with no frames", byte 0), so `quill check` on such a log exits 3 where it exited 0. Only the newest, active segment may be empty. Decision 43 amended, guarantee 6 and the user guide say so (audit L1-11).
- Fixed: `quill append` refused an oversize stdin with "stdin is a batch of more than 16777216 bytes, MaxBatchBytes is 16777216", a size stdin does not have: the limit applies to the sum of `len(line) + 34` per line, so 479,350 one-byte lines (958,700 bytes) are refused. The message now reads "stdin would be a batch over MaxBatchBytes (16777216); each line is counted as its length plus 34 bytes". The limit, the error (`ErrPayloadTooLarge`) and the exit code (1) are unchanged; a script that matched the old text must change (audit L1-26, decision 46 amended).
- Fixed: `AppendBatch` (so `Append`) returned sequence 0 with the error when the inline flush's fsync failed, though the sequences had been assigned, the records were buffered and may survive a reopen; after a `SyncAppends` failure it returned the first sequence. It now returns the first assigned sequence with the error in both cases, and 0 only where nothing was assigned (`ErrClosed`, a failed log, a refused batch, `ErrPayloadTooLarge`, `ErrSeqExhausted`, the buffer or rotation bound). The `Append`, `AppendBatch` and `Config.SyncAppends` documentation, the user guide, the design scope and decision 42 now say so. The command line tool is unchanged (audit L1-5).
- Changed: documentation only, no behaviour change. `Consumer.ReadWait` was documented to return when "the handle closes", and `ErrClosed` to cover "a call on a closed Log, Reader or Consumer". `Consumer.Close` releases the consumer's lock and nothing more: it does not wake a blocked `ReadWait` (which ends with its context or when the underlying `Log` or `Reader` closes), `Read`, `ReadWait`, `Scan`, `Records` and `Follow` keep working after it, and on a closed consumer only `Commit`, `SeekToTime` (which commits) and a second `Close` return `ErrClosed`. The `Consumer` documentation, `ErrClosed`, the user guide, the design scope and decision 41 now say so, and a test pins it (audit L1-16).
- Changed: the documents said that on a failed log (`ErrFailed`) "every later `Append`, `Flush` and `Sync` fails". `Sync` of a sequence that became durable before the failure returns nil, and `Sync` of an unassigned sequence is `ErrBadSeq`, as the code has always done. The wording is now "every later valid `Append` and `Flush` on the open log, and every `Sync` of a sequence not already durable, fails with `ErrFailed`; `Sync` of an unassigned sequence is `ErrBadSeq`", in the `ErrFailed` documentation, the `Sync` and `Flush` comments, the user guide, the design scope and decision 37; a test pins the three `Sync` outcomes on a failed log. No code changes; the runtime text of the one `Error` record a failed log writes ("every later call returns ErrFailed") is left as it is, since changing it is a code change, and the same loose reading applies to it. Owner-approved 2026-11-03 (L1-4).
- Changed: the documents and comments said a consumer is stale, a segment expired and a quiet segment sealed when it is "older than" `ConsumerTTL`, `Retention` and `SegmentMaxAge`. The code has always counted an age equal to the window (in whole seconds, and the file's age to the nanosecond for `ConsumerTTL`) as past it, and `offset == Head` as behind, in the writer's prune pass and in `quill consumers`; the wording now says "at least as old" (decision 45 amended). New tests pin each boundary at one second short and at equality. Documentation and tests only; behaviour is unchanged.
- Changed: `quill cat`, `read` and `tail` hold their output in a 64 KiB buffer instead of a 4 KiB one, so a large output reaches a pipe or file in a sixteenth of the write calls. On a macOS laptop `quill cat` of 600 thousand text records took about 9% less wall time to a pipe and the system time to a file fell from about 0.09 s to 0.03 s; the gain is smaller on Linux and nil to `/dev/null`. The output is byte for byte the same. A non-follow run on a very slow producer now shows output in larger steps, and a killed process can lose up to 64 KiB of already-read output instead of 4 KiB; `-follow` still flushes after every record and every run flushes on exit and on error.

- Changed: `quill cat`, `read` and `tail` print a text record's payload by copying whole runs of plain bytes instead of one byte at a time. On a macOS laptop the JSON form of a 75-byte text record took about 3.4 times less time to format and the text form about 1.9 times less, and `quill cat` of 600 thousand such records into a pipe took about a quarter less; payloads with many escapes gain less, and a payload printed as base64 is unchanged. The output is byte for byte the same (decision 35).
- Changed: a `Reader` view (and so `quill tail`, `stat` and every other command that opens the log read-only) reads the first frame of each sealed segment with a 4 KiB read instead of a 64 KiB one, as it already did for the active segment. On a cold view of 787 segments of 64 KiB it took about 34% less time on a macOS laptop (45% on Linux) and allocated about a tenth; `quill tail -n 10` on that log took about 25% less. A first frame whose payload is over about 4 KiB now takes a second read, a small cost for logs of large records; segments under 4 KiB are unchanged. Results are identical. Decision 91 amended.
- Changed: `Open`, `Reader.Check` and `quill check` read each sealed segment into a buffer taken from a pool instead of allocating a new one per segment (up to 1 MiB each, zeroed and discarded). On a log of 787 segments of 64 KiB, `Open` allocated about 59 MB and now about 2 MB, and on Linux took about half the time; on 8 MiB segments the gain is under 15% and on a macOS laptop within noise. The pool is shared by every log in the process and holds about 1 MiB per processor between garbage collections. What is read and checked is unchanged.
- Changed: `Reader.Check` (and so `quill check`) verifies the segments in parallel, as `Open` does, instead of one after another. On a warm log of 30 segments of 8 MiB it took about a fifth of the time on an M-series Mac, and about a third with the page cache dropped on a Linux virtual disk; a log of one segment, or of many tiny ones, gains little, and on a rotational disk it is unmeasured. It uses up to `GOMAXPROCS` goroutines at a time. The report is unchanged: the same problems in the same order.
- Fixed: group commit barely shared fsyncs at `GOMAXPROCS` 1 (including a one-CPU container limit): the `Sync` that signalled the flusher handed it the processor at once, so it flushed one record while the other producers were still runnable, and a 16-producer run on a real disk did about 134 fsyncs for 160 syncs against about 50 at 10 procs. The flusher now yields once after a `Sync` request before it takes the log's lock, so producers that are ready append first; the same run does about a dozen fsyncs at one proc. Durability and ordering were never affected. `Sync` is not slower for a lone producer beyond one scheduler yield.
- Changed: the `topics` example, its row in `examples/README.md`, `docs/userguide.md` and the design scope said topics "share nothing" and, in the example, that one slow fsync "never delays another". That holds for the library's own state (lock, buffer, flush goroutine, retention, consumers) but not below it: logs on one filesystem share its journal and device queue, which quill does not isolate. Each now says what is not shared and that the device is. Documentation and comments only; behaviour is unchanged.
- Changed: the comment on the `last < s.first` skip in `Reader.view` gave one cause, a stale marker after a rotation crash. The skip is also taken for a bounded segment whose next good segment starts at or below its first sequence (a duplicated or overlapping segment file); the segment is then left out of the view without being scanned, so a `Reader` reports a gap, not the overlap that `Open` and `quill check` report. The comment now says both. Comment only; behaviour is unchanged.
- Changed: the documentation of `Read` said only that it returns a `*GapError` where records are unavailable, and the comments in the read path said only a gap met after records is left for the next call. In fact every error that goes through that path is: a gap, an I/O error, or corruption of a sealed or active segment, once at least one record was collected, comes back as those records with a nil error, and the next call from where they end meets it again (an I/O error may not recur). `Log.Read`, `Reader.Read` and `Consumer.Read` now say so; a segment that ends before its last record is still returned together with the records. Documentation and comments only; behaviour is unchanged.
- Fixed: `quill help tail` gave `-from` the default of `read` and `cat` ("the start of the log"), but `tail` with no `-from` or `-since` prints the last `-n` records, and `-n` is ignored once either is given. `tail` now has its own `-from` text saying so; the `cat` summary no longer says "every retained record" when `-from` and `-since` narrow it; and `-since`'s text now says it wins over `-from`, which the code always did. Help and `docs/reference.md` only; behaviour is unchanged.
- Changed: the comment on the command line tool's `withReader` said every verb checks its usage before opening the log, so a bad flag exits 2 even when `-dir` cannot be opened. That holds only for some checks: `commit`'s `-offset` and `-time`, `register`'s `-protect` and `-no-protect`, and `unregister`'s `-name` are checked first and exit 2; `-name` on `register` and `commit`, and a malformed `-since` on `read`, `cat` and `tail`, are checked once the log is open, so with an unopenable `-dir` they exit 1. The comment now says which. Documentation only; behaviour is unchanged.
- Fixed: `hack/powertest verify` claimed the log is a contiguous prefix but never checked that the first record is sequence 1, so a log that had lost its oldest segment file passed with every acknowledged record before the survivors gone. It now fails when the first scanned record is not sequence 1 (the run keeps every record: retention is off). Tooling only; the library and the command line tool are unchanged.
- Changed: the stated cost of `OffsetForTime` is exact. The search compares O(log segments) first-frame times and scans one segment, but it starts from a view that costs O(sealed segments) on every call, and a `Reader`'s first call opens every sealed segment to read its first frame; the comment in `seek.go`, `scope.md`, the build prompt (test T19) and decisions 59 and 63 said O(log segments) plus one segment. Documentation only; behaviour is unchanged.
- Fixed: `hack/fuzz.sh` ("every fuzz target in the module") exited 0 and skipped a package's fuzz targets when that package did not compile, because it ignored the failure of `go test -list`; the CI `fuzz` job could be green while the package's targets never ran. It now fails when a package's targets cannot be listed and when no target ran at all. Tooling only; the library and the command line tool are unchanged.
- Fixed: the header of `hack/gate.sh` said govulncheck was the one check CI runs and the commit gate does not; CI also runs the full-disk tmpfs test, the examples and the release build script, and the gate did not vet the other platform. The header now lists what CI alone runs, and the gate vets both linux and darwin, so a committer on one sees a compile error in the other's files before pushing. README.md and the audit prompt list the new step.
- Fixed: `Open` reported the empty hole `{From: 1, To: 0}` in `Stat().Holes` (and so in `quill stat` and `check`) when every sealed segment was damaged, the newest segment was empty and `quill.tail` still said sequence 0, though an empty hole is never reported and a `Reader` reported none. The writer now reports none either; `Err()` still names the damaged segment.
- Fixed: `Stats.FlushFailing` stayed true on a log that had failed when a write failure (`ENOSPC`, say) preceded the fatal data fsync (or truncate) failure, though its documentation says it stays false once the log has failed. A failed log now reports false, as documented; `Err()` wrapping `ErrFailed` is the signal for that state, and the documented alerts are unchanged.
- Changed: the `Config.Logger` documentation (`log.go`, the user guide, `scope.md`) now lists every record the log writes. It left out the `MinFreeBytes` records, the warnings for an unreadable consumer file, a failed staleness probe, age rotation skipped after a failed flush, a segment already gone, the age sweep stopped at a future-stamped segment and the `quill.policy` write failure and recovery, and placed the policy write among the records that run with no lock held, though only the first write in `Open` does. It also says which records repeat and where they run: the "cannot read free space" `Warn` is written by every prune pass and every seal whose `statfs` fails (the caps, and so the deletion records and the `MinFreeBytes` records, run at each seal as well as in a pass), as are the other records about a condition (a stale consumer, say, from a prune pass only), while the below-floor `Warn` and the back-above `Info` are one per transition. Documentation and tests only; behaviour is unchanged (decisions 25 and 50 amended).
- Fixed: the file-system seam's `Remove` is documented to unlink a file and never a directory, but the real implementation called `os.Remove`, which also removes an empty directory, while the crash simulator refused one. It now calls `unlink(2)` and refuses a directory. quill only removes names it built itself (segments, temp files, consumer files), so no known path reached the difference; an empty directory planted at such a name is now left alone and the removal fails.
- Changed: the `GapError` documentation no longer says `From` is always the requested offset. `From` is the first unavailable sequence: the offset asked for when nothing was delivered before the gap, otherwise the sequence after the last record the scan delivered; `To` is the last missing sequence, so a consumer resumes with `Commit(To + 1)`. `Scan`, `ScanAll`, `Reader.Scan` and `Consumer.Scan` return the gap at once after delivering records, so for them `From` can be above the offset asked for; `Read`, `Consumer.Read` and `Consumer.ReadWait` return the records first and leave such a gap for the next call, where it is the offset. Documentation and tests only (user guide, `GapError`, decisions 101 and 103); behaviour is unchanged (decision 84).
- Changed: the package overview (`doc.go`) no longer says `Err()` names a missing sealed segment. A damaged sealed segment is a hole that reads report as a `GapError` and `Err()` names; a segment deleted from the directory is a hole in `Stat().Holes` and in reads only, and `Err()` stays nil, as `Err`'s own documentation and the user guide already said. Documentation only; behaviour is unchanged.
- Fixed: `Open` and `OpenReader` returned an error that matched no exported sentinel, not `ErrCorrupt`, when `quill.tail` or `quill.format` was over 1 MiB or was not a regular file (a directory, a FIFO, a device or a symbolic link, a dangling one included), though `errors.Is(err, ErrCorrupt)` is the documented way to tell damage from an I/O or configuration error. They now match `ErrCorrupt`; the refusal itself is unchanged, and so are the words after the sentinel. A consumer offset file read through the same path behaves the same way: for an oversize one or a symbolic link, the `Err` that `Consumers` lists and the error `Consumer` returns now match `ErrCorrupt`. The user guide and `format.md` say so.
- Added: the writer's `Open` logs one `Warn` through `Config.Logger` for each consumer whose offset is above `Head + 1` (a log restored to an older state): "consumer offset is ahead of the head; rewind it with quill commit", with `consumer`, `offset` and `head`. Once the head grew past such an offset the consumer resumed mid-log at sequences that name different records, and `quill check` stopped flagging it. `Open` accepts and refuses what it did, `Stat` is unchanged and the offline verbs do not print the line. The user guide's restore section says to rewind consumers before restarting the writer, and that a consumer ahead of the head protects no segments, so a writer with caps can delete the rewind target before `quill commit` runs (the commit then fails with `ErrOffsetRange`): read `quill stat` just before, or rewind to the first retained sequence or by `-time`.
- Changed: a log that becomes failed (`ErrFailed`: a data fsync error, a truncate that cannot undo a failed write, a directory fsync error, a lost `quill.lock`) now writes one `Error` record to `Config.Logger`, "log failed: it accepts no more writes and every later call returns ErrFailed", with `error` (the cause) and `op`; before, only `Err()` knew, and only a lost lock logged. The state keeps its first cause and a later failure logs nothing, so it is one line; the lost-lock `Error` line is replaced by it. `Stats` and the offline verbs' standard-error filter are unchanged. Decision 60 amended. Owner-approved 2026-11-02 (A5-1f).
- Fixed: a `Reader` served the earlier segment's records for sequences that the next segment also claims, when a sealed segment held frames past the next segment's first sequence (a hand-built or mis-restored directory; `Open` already refused it). `Read`, `ScanAll` and the other reads now fail with `ErrCorrupt` once the overlapping segment is scanned (a read that starts in the later segment does not scan the earlier one), and `Check` reports an "overlap" problem naming both segments instead of the nonsensical hole "sequences 11 to 4 are missing"; `quill check -json` gives that problem the new `kind` `overlap` (scope.md lists it). `Archive` refuses the same layout with `ErrCorrupt` before it copies the overlapping segment (it used to copy both segments, write a destination marker below the last record copied, and return a `GapError` whose range ran backwards). Decision 102 is amended. Owner-approved 2026-11-02 (A3-20).
- Changed: the documented rule for `Config.Logger` is now exact. The handler, `Enabled` included, runs under the log's lock and must not call the `Log`, block, or take a lock that a caller of the `Log` may hold; a handler that writes straight to a pipe or socket blocks when the reader does. The warnings `Open` itself gives (a cut newest segment, lock tightening, a policy write) run on the caller's goroutine before there is a `Log`, with no lock held, which the earlier wording did not say. Documentation only; behaviour is unchanged.
- Changed: two `Archive` refusals say what is wrong. A destination that is not empty and has no `quill.format` now says so, that it is not empty (it said "holds files" even for a subdirectory only), and that Archive needs a new or empty directory or an existing archive; it is still `ErrNoLog`, exit 1. A destination directory that cannot be searched is refused with a permission error that names the directory and claims no writer; before, it was reported as `ErrLocked` ("a writer may be live"), which is now kept for a `quill.lock` that can be seen but not opened. Nothing is written in either case.
- Changed: the offline verbs (`append`, `roll`, `prune`, `recover`) now print on standard error the warning a writable open logs when it cannot tighten a `quill.lock` an earlier build left wider than 0600 (the file belongs to another account). The verb still succeeds and exits as before; the line appears only in that case.
- Fixed: an `Archive` rerun kept a destination segment that existed with the source's size without reading it, so a damaged copy survived the rewind (`quill commit -name _archive -offset N`) meant to replace it, the run exited 0, and only `quill check -dir <dest>` found the damage. A same-size destination segment is now read and verified whole (every CRC, and the source's first and last sequence); a damaged one is replaced by a fresh copy and a healthy one is still not rewritten. No flag. Cost: a rerun after a rewind reads each such destination segment once more; a run with nothing to copy reads none. Decision 85 amended; `docs/userguide.md` documents the cost. Owner-approved 2026-10-31 (A5-32).
- Changed: `quill archive` exits 3 (the code `check` uses for UNHEALTHY) when the source had lost records the archive never copied, where it used to exit 0 after one line on standard error, so `quill archive && notify-ok` no longer reports success for an incomplete archive; the run still copies what exists. `Archive` also compares the destination's `quill.tail` with the first segment it copies, so after `quill unregister -name _archive` (the documented reset) a loss between the destination's last record and the first retained sequence returns the `*GapError` too, where it used to leave a hole in the destination with no message. The run that reports a gap commits past it, so the next run with nothing to copy exits 0; `quill check -dir <dest>` keeps reporting it. Decisions 53 and 85 amended; `docs/userguide.md` no longer says the verb says so for every case.
- Fixed: `quill ls` silently omitted any log it could not open (a corrupt or future `quill.format`, no permission) and exited 0, so a broken log looked like one that did not exist while `stat` and `check` on it exited 1. It now skips only a directory that is not a log (`ErrNoLog`); any other failure prints an `error:` row, the remaining logs are still listed, and `ls` exits 1 afterwards (a failed `Stat` row now exits 1 too). Behaviour change for a script that read `ls`'s exit 0 as "every log is fine". Decision 53 amended. Owner-approved 2026-10-31 (A5-44).
- Changed: `Open` refuses a `MaxBufferBytes` below `FlushBytes` with `ErrConfig`. A flush is sized at the larger of the two, but the published disk bound charged `MaxBufferBytes` alone, so a smaller value let the on-disk peak pass the bound (4,200,444 bytes against 3,949,577 in the reproduction). The bound `MaxBytes + SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes + 34)` is now exact. Behaviour change for a `Config` that set `MaxBufferBytes` under `FlushBytes` (including one under the 64 KiB default `FlushBytes`); the default, 16 times `FlushBytes`, is unaffected. Decision 69 amended. Owner-approved 2026-10-31 (A5-23).
- Changed: `ReadWait` and `Follow` on a failed log return `ErrFailed` at the head instead of blocking until the context ends, so a consumer goroutine can tell a log that can never produce another record from an idle one. `Log.ReadWait`, `Log.Follow` and the `Consumer` forms of a consumer taken from a `Log` are affected, and a goroutine already waiting is woken when the log fails, including by a failed directory fsync in `Roll` or a retention pass, which did not wake waiters before. Records that exist stay readable on a failed log; `Reader` is unchanged (decisions 37 and 58; audit A5-3).
- Fixed: `quill prune` and `quill roll` discarded the error of `Close`, which carries the final directory sync, the publication of `quill.tail` and the lock release, so a fatal failure at that point exited 0 with nothing on standard error (`append` and `recover` already reported it). The two verbs now print the `Close` error and exit 1 when it fails after a successful pass; `prune` has printed its counts line by then. Behaviour change for a script that read their exit 0 as success. Decisions 53 and 100 amended. Owner-approved 2026-10-31 (A5-9).
- Fixed: `quill prune` exited 0 with nothing on standard error when the pass did less than asked: an unlink that failed (the segment stayed and retention stalled behind it), a consumer file that could not be read (the age sweep kept every segment) or the clock guard withheld segments, so a script that checked the exit code saw success and the counts line (`6 -> 6`) was the only sign. The offline verbs' standard-error filter now also passes "cannot delete segment", "age sweep limited by the clock guard", "age sweep stopped at a segment stamped in the future" and a new `Warn`, "consumer file could not be read", which every pass logs for each unreadable consumer file (or an unlistable `consumers/`) on `Config.Logger`; `quill prune` exits 1 when an unlink failed, after printing its counts line; and the library's `Prune` returns the unlink error instead of nil (the flusher's passes still leave it in `Err()`). The clock guard stays a warning with exit 0, since `-force` is the documented answer. Behaviour change for a caller that treated `Prune`'s nil as "the pass ran". Decisions 44, 53, 65 and 67 amended. Owner-approved 2026-10-31 (A5-36).
- Changed: a consumer that fell out of retention no longer reads healthy. `quill consumers` adds a NOTE for a consumer whose offset is below the first retained sequence ("behind the first retained sequence N: records X to N-1 are gone"; `LAG` is unchanged and still counts those records) and, from the policy's `ConsumerTTL` and the offset file's mtime, for a protected consumer that is stale (its `protected` column still shows the file); `-json` adds `"behind_first": true` and `"stale": true` only when true, so a parser sees no change for any other consumer. `quill check` warns of a consumer behind the first retained sequence and stays HEALTHY. A deletion that passes a consumer's offset is logged at `Warn` on `Config.Logger` ("deleted segment past a consumer's offset; it has lost records", with `consumer`, `offset`, `segment` and `last`), and the offline verbs print that line on standard error. `quill prune` ends its output line with `, dropped N`, which a script that matched the whole line must allow for. Nothing is persisted: `Dropped` still resets at a restart. Decisions 12, 36, 45 and 66 are amended.
- Changed: a `MinFreeBytes` floor that cannot be met or checked is reported instead of leaving `Err()` nil. `Err()` now wraps `ENOSPC` while the log refuses appends for want of room, and wraps the `statfs` error from any pass that could not read the free space (one failure is enough; the floor is not enforced for that pass, and a `diskFull` already set is kept), and each clears at the first pass that finds room, which logs it at `Info`. `Stats` gains `DiskFull` and `FreeBytesAtLastPass` (writer only). Decision 25 is amended.
- Changed: a retried background failure, a flush write, a marker publish, a rotation or a prune unlink, is now written to `Config.Logger` once when it starts (`Warn`, with `op` and `err`) and once when it recovers (`Info`), never once per retry; before, only `Err()` knew. "Recovered" means a later pass completed without a failure, not that the failing step was retried. `Log.Stat()` gains `BufferedRecords`, `BufferedBytes`, `FlushFailing` and `LastFlushOK`; alert on `FlushFailing`, or on `BufferedRecords` above zero together with it, and on `Err()` wrapping `ErrFailed` (a failed log does not retry, so `FlushFailing` stays false). `LastFlushOK` is the time of the last flush that completed (it wrote and published its records, or landed a marker publish that had failed), so an idle healthy log keeps an old one: it is not the age of the oldest acknowledged record and `time.Since(LastFlushOK)` alone is not an alert. A `Reader`'s `Stat` reports all four as zero. The "cannot delete segment" `Warn` now carries `op` and is logged on the first failed pass only, not on every pass that fails again.
- Changed: `Log.Err()` reports what is failing now. A sticky load error (a damaged sealed segment) no longer hides a later background error, and a background error is no longer latched for the life of the log: `Err()` returns the `errors.Join` of the failed state or the load error and every live background error (`errors.Is` matches each), a flush clears the write error, a rotation (at a flush, `Roll` or an age rotation) the rotation error, a marker publish the publish error, a prune pass that reaches its deletions the unlink error and a policy write the policy error, and the load error is dropped when retention deletes the segment it names. A caller that treated a non-nil `Err()` as permanent now sees it return to nil after recovery. Decisions 60, 78 and 98 are amended.
- Fixed: `quill archive -to <dir>` and `Archive` wrote into any existing directory that had no `quill.format`: a mistyped `-to` that named a populated directory that is not a log added `quill.format`, `quill.tail` and `consumers/` to it and replaced any file there named like a segment (`NNNNNNNNNN.log`) whose size differed from the copy. A writable `Open` already refused such a directory. Behaviour change: archiving into a populated directory that is not yet a log now fails with `ErrNoLog` ("holds files but no quill.format") and writes nothing. The destination must be new, empty (ignoring `.tmp-*` files and `quill.lock`), or already an archive; an archive left by an interrupted run is unaffected, because `Archive` writes `quill.format` first.
- Fixed: `quill.lock` was created with `Config.Perm` minus the umask, so under a shared `Perm` (0660, 0644) a process that could only read the log could open the lock read-only and take its flock, and every writer `Open`, a restart included, failed with `ErrLocked` until that process exited. The writer's lock is now always 0600, and `Open` tightens one an earlier build left wider (a failed chmod, because another account owns it, does not fail `Open`; it logs a `Warn` on `Config.Logger`). `Archive` now refuses a destination whose lock it cannot open (a 0600 lock owned by another account) as a writer that may be live: the error is `ErrLocked` and a permission error. Consumer locks keep the log's mode, because a group consumer must take its own, so a process with read access to `consumers/` can still hold one and stall that consumer (never the writer).
- Fixed: when the `ConsumerTTL` probe of a behind, silent protected consumer failed (the offset file could not be stat'ed, or the segment at its offset could not be read, with `EACCES` or `EMFILE`, say), the consumer was silently kept live: it kept pinning segments, `StaleConsumers` stayed empty and nothing was logged, so the operator could not tell the TTL was not being evaluated. The consumer still stays live (the failure protects data, and a read error is never taken for a gap), but each prune pass now logs a `Warn` on `Config.Logger` ("consumer staleness probe failed; treating it as live") with the consumer and the error.
- Fixed: on an empty log (a new one, or one closed and reopened with no records), the writer's `Stat().MarkerAge` read 0, "just written", until the first marker publish after `Open`, because `Open`'s republish of the marker was skipped when the marker was the zero value. It now counts from `Open`. A log holding records was not affected, and a `Reader`'s `MarkerAge` (the file's mtime) was always right.
- Fixed: the writer never re-checked that the `quill.lock` it holds is still the file at that path, so after the file was deleted under a live writer (a temporary-file cleaner, a restore over the directory) a second writer opened without `ErrLocked` and the two silently overwrote each other's acknowledged records, with `quill check` healthy. The writer now compares its lock file with the path (one `lstat`) before each flush and prune pass, and a mismatch fails the log with `ErrFailed` and an `Error` on `Config.Logger`. A second writer that opened before the next flush is not prevented: do not delete `quill.lock`.
- Fixed: when retention deleted a damaged sealed segment, nothing recorded that the data was lost to damage: the deletion was an ordinary `Info` "deleted segment", the hole left `Stat().Holes`, `quill check` turned healthy, and a restart lost `Err()`. The deletion (still allowed, decision 98) is now logged at `Warn` as "deleted damaged segment" with `segment`, `last` and `damage` on `Config.Logger`, and an offline `quill prune` prints the same record on stderr. No format change.
- Fixed: when the write of `quill.policy` failed at `Open` (an `EIO` or `ENOSPC` at that one write), `Open` succeeded and the previous run's file stayed on disk, so `quill check`, `quill ls` and an offline `quill prune` acted on caps and retention the writer no longer ran with; only `Err()` said so. The writer now removes the stale file, logs a `Warn` on `Config.Logger`, and retries the write at each flush, each prune pass and at `Close` until it succeeds, then clears the error from `Err()`. A policy failure still never fails `Open` (decisions 29, 48).
- Fixed: a prune pass whose age rotation could not flush (a full disk, with a quiet active segment older than `SegmentMaxAge`) returned the flush error and never reached the age sweep or the caps, so the retention that would free the space was skipped exactly when the disk was full. The pass now skips the rotation, logs a `Warn` on `Config.Logger` ("age rotation skipped because the flush failed"), and still runs the sweep and the caps. A fatal fsync failure still stops the pass.
- Fixed: a writable `Open` that truncated the newest segment past the marker recorded nothing, so an operator could not tell a routine torn tail from a cut that removed a complete-length frame failing its CRC (possibly an acknowledged record, the documented residual of guarantee 3). `Open` now logs a `Warn` on `Config.Logger` with the segment, offset, bytes cut and `kind` (`torn` or `damaged`), and every offline CLI write verb (`append`, `roll`, `prune`, `recover`) prints the same record on stderr. The truncation itself is unchanged.
- Fixed: the golden tests were weaker than they looked. The `quill.policy` golden had zeros and equal values for most fields and a protected consumer name no writer can produce, and its decode check read four fields; it now gives every field a distinct non-zero value, is the file the writer writes for those settings, and is compared whole. Every decode-direction golden failure now carries the new-version, do-not-re-record text, and the segment image mismatch no longer presents two recorded artifacts as "now" and "recorded". The `quill.policy` example in `docs/format.md` is now byte-identical to the golden (arrays one element per line). No format change.
- Documented: the frame goldens were recorded from quill's own encoder, and no fixture is a frame written by the reference implementation, so frame compatibility with the reference is by specification, not by fixture (`docs/format.md`).
- Documented: the format marker is checked when a handle is opened (and again by `Consumer.Commit` and the wait inside `ReadWait` and `Follow`), not by a running writer or by plain reads, so a future migration note must require stopping or restarting every process (`docs/format.md`, `docs/design/scope.md` Versioning, decision 48).
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
- Fixed: a damaged or truncated-to-nothing oldest segment was skipped silently by a read from the beginning: `quill read` and `quill cat` without `-from`, `ScanAll`
  and a scan from offset 0, and a newly created consumer all started at the first good segment and, for the command line, exited 0, while `stat` and `check` alone
  showed the lost range (`cat -from 1` already failed). They now start at sequence 1 when the oldest segment is damaged and meet the gap (`records 1 to N are
  unavailable`, exit 1); `-from` past it works as before. `quill tail -n N` meets it when N reaches back to the oldest record; `-since` and `OffsetForTime` still start at the first retained sequence (decision 59). A new consumer, including the one `quill archive` creates on its first run, now starts at 1, so that first `archive` reports the source's lost range (`records 1 to N`) as `Archive` already documents. A prefix deleted by retention is still skipped. A writer's retention pass that deletes the clean prefix in front of a damaged segment, leaving it the oldest file, used to drop its hole from the live `Log` (`Stat` showed none, and `ScanAll` and a new consumer started past it, while a `Reader` and a fresh `Open` of the same directory reported it); the hole from 1 is now kept until the damaged segment is itself deleted. Decisions 38 and 41, scope.md, the godoc and the command help are amended (owner-approved 2026-11-02).

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
  returns a `*GapError`. They now return a `*GapError` over that segment's range, starting at the first unavailable sequence (the offset asked for when nothing was delivered before the gap; for `Scan`, `ScanAll`, `Records` and `Follow` after delivered records, the sequence after the last record delivered), so the
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
