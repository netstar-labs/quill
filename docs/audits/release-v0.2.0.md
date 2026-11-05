# Release record: v0.2.0 rehearsal

A rehearsal of the v0.2.0 release, run end to end against commit `35d1185b92e357847fc2943420318d2786e77f07` (committed
2026-11-04T04:57:42Z) in a scratch clone with a local annotated tag `v0.2.0`. Nothing was pushed, no tag was created in
the real repository (its only tag is still `v0.1.0`) and no release was published. This document is the record of what
the rehearsal showed, the commands the real release will run, what stands between the tree and a public repository, and the
audit findings that are still open at the tag ("Known limitations and open items at the tag").

**Update 2026-11-04, tag day (owner's instructions).** `CHANGELOG.md` now reads `## v0.2.0 (2026-11-04)` where it read `## Unreleased`
(the release blocker below). `.github/CODEOWNERS` keeps `@zxdev`: the owner is the author and the login has admin on the repository (read
from the GitHub API). `SECURITY.md` lost its internal process wording (the "Stage 2 audit, owner-approved" parenthesis at lines 31-32); its
contact stays GitHub private vulnerability reporting, which returns 404 until the repository is public. The remote repository
`netstar-labs/quill` exists, is private and empty (read from the GitHub API). A scrub of every ref and every object in the local
repository finds the leaked name nowhere (verified, including the 8 unreachable objects). The checklist and findings below are as
they were at the rehearsal unless a line says otherwise.

Labels: **verified** means the rehearsal ran it or read the exact lines; **reported** names the document that says it;
**inferred** says from what. Line anchors are to the tree as of this commit.

## Top line

**The build, the gate and the reproducibility check are green; the tree is not releasable as it stands, for one reason
that a single commit fixes, and not publishable, for three reasons that are the owner's.**

- **Release blocker (a commit fixes it).** `CHANGELOG.md:6` is still `## Unreleased` and nothing mentions v0.2.0, so
  archives built from a v0.2.0 tag would ship a changelog with no v0.2.0 section. Retitle it in a commit before the real
  tag; the tag then sits on that commit and every hash below changes (the binaries embed the commit).
- **History: fixed 2026-11-04 (owner decision), by rewriting before any push.** A scratch-directory path in one audit document
  (old commit `98ff9e7`, removed at `0b70e82`) was in `main`'s history and in the `v0.1.0` tag's tree. The history was filtered
  once: 0 hits in any patch of any ref afterwards (verified), 451 commits kept, `main`'s tree identical, `v0.1.0` re-created
  on a new commit. 321 of `main`'s 370 commits have new hashes: see [commit-map.txt](commit-map.txt).
- **`LICENSE`: done 2026-11-04.** Apache-2.0 (owner decision), added to the repository; the Go standard library's notice in the archives (A4-5) is done 2026-11-04 as `THIRD-PARTY-NOTICES` (owner-approved, L5-4).
- **Public-flip blocker: scratch refs.** The local repository carries 484 scratch branches; only `main` and the release
  tag may be pushed.

## Pre-flight (verified)

| check | result |
|-------|--------|
| `FUZZTIME=20s hack/gate.sh` | exit 0, "OK: gate green" |
| gate steps | no dependencies; gofmt; vet for linux and darwin; freebsd refused; `go test -race` (8 packages ok; root 63.7 s, `cmd/quill` 32.2 s, `hack/topology` 22.9 s); staticcheck v0.8.2; deadcode v0.52.0; 15 fuzz targets at 20 s each; `go mod tidy` with no diff |
| govulncheck v1.9.0 (database dated 2026-11-02), Go 1.27.3 | "No vulnerabilities found" |
| tree state | `git status --porcelain` empty; `HEAD` equals `v0.2.0^{commit}` |
| toolchain | pinned by `go.mod` (`go 1.27`, `toolchain go1.27.3`) |

The gate does not run govulncheck (it reads a live database, so it can go red with no change in the repository;
`hack/gate.sh` header). It is a separate step in the release procedure below and must be run again at tag time.

**The gate in this table ran on macOS only (corrected: this record had implied it covered both platforms).** `hack/gate.sh`
vets for linux and darwin and refuses freebsd from any host, but its `go test -race`, its fuzz targets and the rest run on the
host's own operating system (verified: `hack/gate.sh`). The rehearsal's gate run was on the macOS build machine (inferred: the
record puts the clones and the release engineer on macOS and names no Linux gate run; the Linux box appears only in the
build comparison and the smoke test below). The Linux gate for this wave of fixes ran on tmpfs
only (reported by the critic's review of this record, audit item L4b-2; not re-derived), and the full-disk test on a real
tmpfs is a CI step the gate omits (`hack/gate.sh` header, verified). So the table is a macOS result: no Linux `go test -race` of
the tag is recorded here.

### Refusals of the release build (verified, in a second clone so the gated tree stayed clean)

`build/quill` refuses, exits 1 and produces nothing:

- untagged `HEAD`: `build/quill: HEAD is not tagged; tag it (v0.y.z) or use --dev`;
- a tagged tree with `README.md` modified: `the tree is dirty; commit or stash, or use --dev`;
- an untracked stray file: the same refusal.

Minor (verified for `git describe` only; the build was not run): the script accepts any tag name, though its message says
`v0.y.z`. A tag `nightly-x` makes `git describe --tags --exact-match` return `nightly-x`. This is audit item A4r-3 in
[supply-chain.md](passes/supply-chain.md), deliberately not changed. The release engineer must spell the tag `v0.2.0`.

## Semver rationale

Quill is `v0.y.z` for ever (decision 23; `README.md:114`): breaking changes bump the minor with a migration note, fixes
and additions bump the patch. The last tag is `v0.1.0`. **v0.2.0, a minor bump**, because the changes since `v0.1.0`
include source-level and behavioural breaks, not only fixes. A patch release would be wrong. The tree has 184 changed
files outside `docs/` since `v0.1.0` (`git diff --stat v0.1.0 HEAD`, verified).

The on-disk format is unchanged: it is `quill/1` and versioned separately in `quill.format` (reported:
`CHANGELOG.md` header and the "Upgrading and rolling back" paragraph in the user guide, section 9). So replacing the
binary needs no migration. This is reported by those documents, not re-derived here: the rehearsal ran no cross-version
test. The audit record [format-compatibility.md](reference/format-compatibility.md) covers the format.

### Migration table

Breaking and behaviour-visible changes since `v0.1.0`, one row each. **Who** is script (the `quill` command's exit
status or output), library (a Go caller of package `quill`) or operator (a person or a process that runs the log).
**Recorded** is the `CHANGELOG.md` line in the "Entries" section as of this commit, then the commit that made the change
(`git show <sha>`); `A..`/`L..` are the audit ids in those entries. The same rows, shorter, head the section in
`CHANGELOG.md` ("Breaking changes and upgrade notes", lines 8 to 57 now; 8 to 54 when the table was derived).

How the table was derived (verified: counts by `awk` over the file, read of every entry's first lines and of the full
text of those that name a script- or library-visible behaviour, and `git log --oneline v0.1.0..HEAD`: 287 commits, 252 of
them `stage2` commits).

```sh
awk '/^### Entries/{f=1;next} /^## /{f=0} f && /^- /{n++} END{print n}' CHANGELOG.md   # 144 when this table was derived (e8909b6); 146 at 2893a3f; 152 after pass L5; 153 after pass L5b (see below)
```

**Update 2026-11-04 (pass L5: counts and line numbers re-derived).** Pass L5 (`cfa7bf7`, `4ffd425`, `27cbbb1`, `29f785c`) added four entries to `CHANGELOG.md`
and three lines to the "Breaking changes and upgrade notes" list, and three rows (39 to 41) to the table below. Re-derived by running the command above on the file as it is
now (verified): it prints **152**. The two L5-era notes that stood here before (written by the fixers, who counted from 146) read 147 to 150 and missed the two entries
that the `LICENSE` commit (`ef46b90`) added at the top: the `Added:` entry for `LICENSE` and the `Fixed:` entry for the `Reader` view restart (decision 101, `456ef7c`, old hash
`2893a3f`). The count is reproduced this way: 144 at the derivation, plus 2 (the two doc-only entries at old lines 431 and 432, `2893a3f`), plus 2 (`LICENSE`, `Reader` restart), plus 4 (L5-1, L5-2,
L5-3, L5-4) is 152.

**Update 2026-11-04 (L5-5).** The `--version` fix adds one `Added:` entry at the top of "Entries" and no line to the breaking-changes list (its note is appended to the L5-1 line), and row 39a to the table: the command above prints **153**, entries with a row are **90**, entries with no row stay **63**, the list stays at **38** lines (all re-derived by the recorder of pass L5b with `awk` on the file as it is: 153, 38 and 191 for the whole section; 153 minus 63 is 90; verified). The fix is commit `cf7d789`, FIXED-VERIFIED by one fix-skeptic, 7 of 7 lenses (reported). Every entry line cited below by number sits one line lower than the numbers in this record (the new entry is above them all).

| | derivation (`e8909b6`) | now |
|---|---|---|
| entries under "Entries" (the command above) | 144 | **153** (152 after pass L5, plus L5-5) |
| entries with a row in the table | 86 | **90** (86 plus L5-1, L5-2, L5-3, rows 39, 40, 41, plus L5-5, row 39a) |
| entries with no row | 58 | **63** (the 58, plus the two doc-only entries of the pre-tag batch, the `Reader` restart fix, the `LICENSE` entry and the `THIRD-PARTY-NOTICES` entry) |
| `- ` lines in the "Breaking changes and upgrade notes" list | 35 | **38** (verified with `awk`) |
| the command applied to the whole "Unreleased" section | 179 | **191** (153 plus 38) |

The 86 and the 58 were reproduced from the table itself (verified, a script over the "Recorded" column of rows 1 to 38 and the entry lines of `CHANGELOG.md` as of `31ae18f`, the commit that
derived them): the column cites 87 distinct entry lines, all of them entries; line 66 is the extra one row 16 cites, and it is one of the 58, so 86 entries have a row and 58 do not, 144 in all. The
`LICENSE` and `THIRD-PARTY-NOTICES` entries are "no row" because nothing a script or a Go caller reads changed (the archives gained members; inferred from the entries' own words); the `Reader` restart is
"no row" for the reason given in the pre-tag update below (a rare `lstat` error became a retry; a judgement, and the owner may want a row, see "Known limitations"). The classification of the new
entries is the recorder's reading of their own text, not a re-run.

**Line numbers.** Every `CHANGELOG.md` line number cited in this section, in the table rows 1 to 38 and in the lists of entries with no row is the number in the file at the derivation. Matching each of the 144
entries by its text between `31ae18f` and the tree now (verified, a script), all 144 sit exactly **9 lines lower** (10 lower after pass L5b, which added one entry above them) (3 for the new list lines and 6 for the six entries added above them: L5-1 to L5-4, `LICENSE` and the `Reader` restart). Add 9 to any such citation to find the entry now. Rows 39 to 41 cite the lines as they are in the file now: line 63 (L5-1), line 62 (L5-2) and line 61 (L5-3); L5-4 is line 64. The
two entries added at the bottom by the pre-tag batch (old lines 431 and 432) are lines 440 and 441 now.

The count starts at the "Entries" heading, not at "Unreleased", because the "Breaking changes and upgrade notes" list above
it also has lines that start with `- ` (35 of them at the derivation, 38 now: the command prints 179, now 191, if it scans the whole section, and 143 on the
file before this change, which had no such list). It would stop working if that heading were retitled; the number
is what to compare, not the command.

`CHANGELOG.md` "Unreleased" holds **144 entries** (143 before the entry that records this change). **86 are covered by
a row below and 58 are not user-visible.** An entry is identified by the line it starts on (the "Entries" section holds the new entry at line 58 and the first older one at line 60). The 58 are, by line:

- documentation only, no behaviour change (37): 60, 64, 66, 67, 68, 69, 70, 73, 75, 76,
  77, 83, 87, 88, 89, 97, 99, 103, 108, 110, 111, 116, 140, 141, 157, 199, 206,
  231, 283, 320, 325, 329, 334, 348, 352, 357, 422;
- a refactor (1): 71;
- a comment or help text (3): 98, 100, 101;
- performance only, the result unchanged (6): 90, 92, 93, 94, 95, 96;
- test seams, harness and build or CI scripts (6): 102, 104, 105, 109, 302, 344;
- tests (1): 139; a latent bug no path reaches (1): 147; a data race and a cache-growth fix with no changed result (2):
  370, 374;
- the entry that records this change (1): line 58.

**Update 2026-11-04 (pre-tag batch).** The command above prints **146** at `2893a3f` (verified, run in the worktree of this record's commit),
not 144: two entries were added after the table was derived, both documentation only and both after the last covered
line (428), so no row's line number moved: line 431 (audit L4b-4, `be9adfb`) and line 432 (audit L4b-3, `e51c2e3`). The counts
therefore read **146 entries, 86 with a row, 60 not user-visible** (the 58 above plus lines 431 and 432). `CHANGELOG.md`
lines 10 and 58 still say "144" and "the other 58"; this record's commit does not edit `CHANGELOG.md` (the owner's wave
limited it to the files named under `docs/audits/`), so those two figures are stale by two entries until a later commit
corrects them. One more change since the table was derived has **no** `CHANGELOG.md` entry and no row: `2893a3f`
(`reader.go`, a `Reader` view overtaken by a rotation and a deletion starts over instead of returning the raw path error; the
same window exists at `v0.1.0`; verified: its diffstat names no `CHANGELOG.md` and a search of the file for the error text
finds nothing). It is a fix that turns a rare `lstat` error into a retry, not a break, and it carries a dated note on decision
101; whether it needs a line before the tag is the owner's.

Each of the 86 is named in the "Recorded" column (row 16 also cites line 66, a documentation-only entry that states the permission-error behaviour; coverage was checked by script). (Inferred: the classification of the 58
rests on each entry's own words, "documentation only", "no behaviour change", or a measured speed-up with the result
unchanged; none of them was re-run. A reviewer who disagrees moves a line between the lists and the counts follow.)

| # | What changed | Who | What to do | Recorded |
|---|--------------|-----|------------|----------|
| 1 | `ConsumerOptions` (new `Rewind`), `Problem` (unexported `refused`), `SegmentError` (unexported `oversize`) and `Stats` (six new fields) each gained a field, so an unkeyed composite literal of any of the four from another package no longer compiles | library | build them with field names | lines 62, 127, 128, 296; `65b7cb0`, `99b140d`, `1dda449`, `ba7c650` |
| 2 | `Open` returns `ErrConfig` for `MaxBufferBytes` below `FlushBytes` (including below the 64 KiB default `FlushBytes`), for a size near `math.MaxInt64`, and for a `SegmentBytes` that with the largest flush reaches the 1.5 GiB segment limit; while rotation keeps failing, an append past the bound is refused | library, operator | leave `MaxBufferBytes` at its default or set it at least `FlushBytes`; replace "unlimited" with a size | lines 122, 175, 270; `4d0e8c9`, `120c8ac`, `c18a27a` |
| 3 | `MaxReadBytes` bounds a batch, each record counting its payload plus 48 bytes | library, script | a `Read(from, 0)` on small records returns a first batch: call again from the next sequence (`Records`, `Follow` and the tool already do) | line 82; `637ed42` |
| 4 | `ReadWait` and `Follow` at the head of a failed log return `ErrFailed` instead of blocking | library | treat `ErrFailed` as the signal to close and reopen, not as "idle" | line 123; `1e0a8ed` |
| 5 | A writable `Open` writes `quill.tail` only when recovery moved it, may return with the publish pending, and sets an existing `quill.tail` to `Config.Perm` in place | library, operator | check `Err()` after `Open` | lines 65, 63; `29f5c6c`, `96588fb` |
| 6 | `Err()` reports what is failing now and can return to nil; it wraps `ENOSPC` or the `statfs` error while the `MinFreeBytes` floor cannot be met or checked; `Stat().Holes` no longer holds an empty hole, `FlushFailing` is false on a failed log, `MarkerAge` counts from `Open` | library, operator | do not treat a non-nil `Err()` as permanent; `errors.Is(err, ErrFailed)` is the terminal state | lines 127, 129, 106, 107, 133; `91c2fce`, `1dda449`, `932cadf`, `d9d4833`, `906d49c` |
| 7 | `Prune` returns the unlink error where it returned nil | library | treat a non-nil `Prune` error as "the pass did less than asked" | line 125; `2cbfdf2` |
| 8 | `AppendBatch` and `Append` return the first assigned sequence beside the error when the inline flush's fsync failed (they returned 0) | library | do not resubmit on a non-zero sequence with an error | line 86; `867f4ed` |
| 9 | A read from the beginning, `ScanAll` and a new consumer (the `_archive` one included) return the `GapError` of a damaged or empty oldest segment, where they started at the first good one | library, script, operator | resume with `Commit(To + 1)` after deciding the range is lost, or read with `-from` past it | line 160; `badb254` |
| 10 | A read I/O error (`EIO`) comes back as itself, not as `ErrCorrupt` or a `*GapError`; a `Log` read of a segment retention just deleted returns a `*GapError`, not a `*fs.PathError`; a `Reader` retries a transient I/O error | library | retry on the I/O error where `ErrCorrupt` or `GapError` was matched for it; resume from `To + 1` on the gap | lines 260, 278, 415; `bb608d3`, `7ef431c`, `c8fb07b` |
| 11 | Stricter validation: `Open`, `OpenReader` and `Check` refuse as `ErrCorrupt` a non-minimal varint, a frame after one numbered 2^64-1 (the ceiling is now 2^64-2), a `quill.tail` with `maxseg` above 9999999999, `off` above 1.5 GiB or a key repeated in another case, an oversize or non-regular state file (it matched no sentinel), and a `quill.policy` with no `retention` reads as unreadable; no log quill writes holds any of them | library, script, operator | a hand-built, restored or damaged log that `v0.1.0` opened may not open: `quill check` names the file; the user guide's corruption procedure applies | lines 112, 149, 153, 165, 193, 226, 394, 425; `d6dc2b9`, `46ff40f`, `bb646c0`, `5c603fc`, `89957ae`, `890c70e`, `a4f92e8`, `5579010` |
| 12 | A `Reader` and `Archive` refuse a sealed segment that holds frames past the next segment's first sequence (`ErrCorrupt`), and `Check` reports an `overlap` problem (`quill check -json` `kind` `overlap`) | library, script | none for a log quill wrote | line 115; `53a5d3b` |
| 13 | `Archive` refuses a destination that is the source log, has a live writer or a `quill.lock` it cannot open (`ErrLocked`, a permission error too), or is a populated directory that is not yet a log or archive (`ErrNoLog`, nothing written); a newer destination format is `ErrFutureVersion`, not `ErrCorrupt` | library, script | archive into a new or empty directory, or an existing archive | lines 130, 378, 117, 131, 236; `5b486d2`, `303b794`, `e293016`, `47315f1`, `b529db5` |
| 14 | `quill archive` exits 3 when the source had lost records the archive never copied (it exited 0); `Archive` returns the `*GapError` for a loss between the destination's last record and the first retained sequence, also after `quill unregister -name _archive` | script, library | treat 3 as "incomplete archive", as for `check`; `quill archive && notify-ok` now stops | line 120; `caad4c5` |
| 15 | An `Archive` rerun reads a same-size destination segment and replaces a damaged one (it kept it); it removes `.tmp-*` files older than an hour and syncs a new destination into its parent | operator | none; a rerun after a rewind reads each such segment once more | lines 119, 361, 365; `16cb0c8`, `db30a94`, `7672a8d` |
| 16 | `quill.lock` is created 0600 whatever `Config.Perm` is and `Open` tightens an older one; a writable `Open` under another account fails with the permission error on `quill.lock`, not `ErrLocked`; `Open` with `Config.ProtectedConsumers` no longer fails `ErrLocked` while a consumer process holds that consumer's lock | library, operator | a script that detects a live writer treats `ErrLocked` and the permission error alike | lines 131, 66, 288; `47315f1`, `7fa0744`, `b5ad955` |
| 17 | A consumer file made through a `Reader` or the CLI, the files `Archive` creates, and what the offline verbs (`append`, `roll`, `recover`, `prune`) write, take the log's `Config.Perm` mode (the verbs rewrote `quill.tail` 0600) | operator | none; a default-mode log is unchanged | lines 249, 142 |
| 18 | `Open` and `OpenReader` make a relative `Dir` absolute, `Open` follows a symbolic link in `Dir`, and a FIFO planted as a state file is refused as damage instead of hanging | library, operator | none | lines 144, 221, 183; `837c860`, `8c0c531`, `acf7f39` |
| 19 | `Consumer` and `DeleteConsumer` accept the name `_archive`; `ConsumerOptions.Rewind` opens a consumer ahead of the head (`quill commit` sets it); `Consumer.Commit` and the wait in `ReadWait` and `Follow` return `ErrWrongLog` if the directory was replaced by another log | library, script | none | lines 255, 296, 145; `a3955cd`, `ba7c650`, `495e457` |
| 20 | `Dropped` is counted at a seal from the last prune pass's offsets, so it may lag a consumer by up to `PruneInterval` | library | call `Prune` to refresh it | line 169; `2bf7412` |
| 21 | A writer whose `quill.lock` was deleted or replaced fails the log (`ErrFailed`, an `Error` record) at its next flush or prune pass | operator | do not delete `quill.lock` | lines 134, 114; `6acc23e`, `531fb35` |
| 22 | New `Config.Logger` records: background failures once on entry (`Warn`) and on recovery (`Info`), a cut of the newest segment, a deleted damaged segment, a deletion past a consumer's offset, a consumer ahead of the head, a failed staleness probe, a skipped age rotation, a segment stamped in the future; the offline verbs print the relevant ones on standard error and the warning that `quill.lock` could not be tightened | operator, script | a log handler or a script that matched standard error should allow for them | lines 113, 118, 126, 128, 132, 135, 137, 138, 216; `99b140d`, `84efe58`, `7b87e64`, `d451fac`, `66081fe` |
| 23 | Retention and recovery failure paths: the `MinFreeBytes` estimate credits segments the pass already deletes; a prune pass survives a failed age rotation, syncs the directory before each unlink and counts an already-missing segment as deleted; a refused `Open` no longer cuts the newest segment; a failed `quill.policy` write at `Open` removes the stale file | operator | none (fixes) | lines 72, 137, 389, 419, 384, 136; `47812f1`, `dd460a7`, `e2f7801`, `c2a7365`, `89073cb`, `1b68eee` |
| 24 | `-from`, `-limit` and `-n` take plain decimals: `010` is ten (it was octal eight), and `0x10`, `1_0`, `0b1`, signs and exponents exit 2 | script | write plain decimals | line 240; `b4fe935` |
| 25 | Any verb except `append` and `help` given a positional argument (`read 100`, `tail 5 -json`, `stat -json extra`) exits 2; it ignored the argument and every flag after it | script | flags first: `read -from 100` | line 244; `2c4e4f4` |
| 26 | `quill roll`, `recover` and `prune` (with a retention or cap flag) at a directory that is not a log exit 1 with `not a log: <dir> has no quill.format` and create nothing; they exited 0 and a mistyped `-dir` got a new log | script | create the log first (`append` still creates one) | line 78; `18145b9` |
| 27 | `quill ls` prints an `error:` row for a log it cannot open and exits 1 after the rest (it skipped it and exited 0); only a directory that is not a log is skipped | script | allow exit 1 where 0 meant "every log is fine" | line 121; `29e7db1` |
| 28 | `quill prune` and `roll` print `Close`'s error and exit 1 when it fails after a good pass; `prune` exits 1 when an unlink failed or a consumer file could not be read; its output line ends `, dropped N` | script | allow exit 1; a script that matched the whole `prune` line must allow for the suffix | lines 124, 125, 126; `46f30db`, `2cbfdf2`, `7b87e64` |
| 29 | `quill read`, `cat` and `tail -n` that reach a damaged or empty oldest segment print `records 1 to N are unavailable` and exit 1 (they started at the first good segment and exited 0); `quill archive`'s first run reports that range, exit 3 | script | `read -from` past the range, or resume after deciding it is lost | line 160; `badb254` |
| 30 | `quill cat`, `read`, `tail`, `stat`, `check`, `ls` and `consumers` exit 1 when writing to standard output fails (a short output exited 0); `tail -follow` stops | script | none | line 410; `9c6a8c3` |
| 31 | `quill tail -follow` writes each record as it is read (it held a 4 KiB buffer, so a pipe saw nothing until it filled) | script | none | line 428; `4337252` |
| 32 | `quill check` exits 3 (it printed HEALTHY) for a sealed segment truncated to zero bytes and reports an `overlap` problem; it no longer reports UNHEALTHY, a hole, or a consumer-ahead warning for a segment or consumer that changed during the scan | script, operator | none for a healthy log | lines 84, 115, 405, 80, 81; `ce47243`, `53a5d3b`, `89211cc`, `ba30fee`, `136577c` |
| 33 | `quill consumers` and `check` note a consumer behind the first retained sequence (`-json` adds `behind_first`, `stale`) or ahead of the head (`-json` adds `ahead`), each only when true | script, operator | none | lines 126, 296; `7b87e64`, `ba7c650` |
| 34 | `quill stat -json` writes holes as `{"from":N,"to":N}` and an empty list as `[]` (it wrote `{"From":N,"To":N}` and `null`); the `stat`, `check` and `consumers` JSON shapes are documented and pinned | script | read the lower-case keys and an empty array | line 202; `c91329c`, `7dea97a` |
| 35 | `-json` output of `cat`, `read` and `tail` escapes DEL, U+0080 to U+009F and the 170 Unicode format and bidi characters in `payload` as `\uXXXX`; the decoded value is the same | script | decode the line, or match the escape | lines 74, 211; `d15ac45`, `fd52463` |
| 36 | `ls`, `consumers`, `check` and error messages quote a name from the disk that holds a control, bidi or invalid UTF-8 character; a printable name prints as before | script, operator | none | line 79; `116726b` |
| 37 | `quill append`'s oversize-stdin message reads `stdin would be a batch over MaxBatchBytes (...); each line is counted as its length plus 34 bytes` and stdin is bounded while it is read; the limit, `ErrPayloadTooLarge` and exit 1 are unchanged. A library error prints one `quill:` prefix (it printed two), and a failed `quill commit` leaves no new consumer behind | script | a script that matched the old text must change | lines 85, 179, 339; `c2cddb7`, `8449425` |
| 38 | Release builds: `build/quill` pins the toolchain to `go.mod`'s, `GOAMD64=v1` and `GOARM64=v8.0`, and puts the commit time, not the clock or the builder, into the archives and the SBOM, whose components now carry a `bom-ref`; `--dev` works on a clean tree | operator (the releaser, a verifier) | a third-party verifier builds with `-buildvcs=false` or inside a checkout of the tag | lines 61, 188, 308, 311, 401; `5bd55ce`, `988f1e5`, `10e8062`, `1b17ea5`, `df7af4f` |
| 39 | `quill -h` and `quill --help` before a verb print the usage of `quill help` and exit 0, where they printed `quill: flag: help requested` and exited 2; a bare `quill` still exits 2 | script | a script that read exit 2 from `quill -h` as "help is not supported" now sees 0 | line 63 (current numbering) and the "Command line" list; audit L5-1; `cfa7bf7` |
| 39a | `quill --version` and `quill -version` print what `quill version` prints and exit 0, where they exited 2 with `flag provided but not defined: -version`; `quill read -version` (a version flag after a verb) is still a usage error | script | a script that relied on `--version` failing (exit 2) must change; it now succeeds | the "Command line" list (the note is on the L5-1 line, not a line of its own); audit L5-5 (owner-approved 2026-11-04); `cf7d789` |
| 40 | The line `quill archive` prints for a source gap reads `quill: the source lost records N to M; the archive does not have them`, where it read `...; the archive holds the same hole` (wrong when the destination starts after the lost records); exit 3 and the stream are unchanged, and the library's `*GapError` text is unchanged | script | a script that matched the text "same hole" must change; match the exit code 3 instead | line 62 (current numbering) and the "Command line" list; audit L5-2; `4ffd425` |
| 41 | The module refuses to build for a 32-bit target (linux 386, arm, mips, mipsle) with `invalid array length quillRequires64BitPlatform - 1 (constant -1 of type int)`, where it built; every 64-bit target still builds | library (and operator: a 32-bit release build or cross-compile) | a caller or packager that built quill for a 32-bit target now fails at compile time; build for a 64-bit target, there is no 32-bit port | line 61 (current numbering) and the first line of "Go callers"; audit L5-3; `27cbbb1` |

Nothing in the table changes a file's bytes: the on-disk format is `quill/1` and the encoders are untouched (reported:
`CHANGELOG.md` header; `hack/skew.sh` was not run here). The readers are stricter about a few malformed shapes (row 11).

**Cases run for this table (verified, `v0.1.0` and this tree built from `git archive` with go1.27.3, one log of 12
records).** `tail -n 010` printed 8 lines at `v0.1.0` and 10 now (row 24); `read 5` exited 0 with no message and now exits 2
with `unexpected argument "5"; flags go before arguments` (row 25); `ls` over a directory holding a log with
`quill.format` `quill/9` exited 0 and now exits 1 (row 27); `roll`, `recover` and `prune -retention 1h` at a directory that
is not a log exited 0 and created a directory entry, and now exit 1 and create nothing (row 26); `stat -json` printed
`"holes":null` and now `"holes":[]` (row 34); `cat -json` of a payload with byte 0x7f printed it raw and now prints
`\u007f` (row 35); `quill archive -to <populated non-log directory>` exited 0 and added `quill.format`,
`quill.tail` and `consumers/` to it, and now exits 1 and leaves the directory as it was (row 13); `read -from 0x2` exited 0
and now exits 2 (row 24). **Not run** (reported from the entries, not re-derived): every other row, including rows 3 to 12 and
14 to 23, 28 to 33 and 36 to 38. A probe of a closed standard output (`stat >&-`) exited 0 at both versions on macOS
because the log's own file reuses descriptor 1, so row 30 was not reproduced; the Linux `/dev/full` case was not run.

**The compile-time break (verified: `go doc -all` at `v0.1.0` and at this tree, both from `git archive`, go1.27.3, then a
diff of the exported declarations of package `quill` with comments removed).** No exported function, method, constant or
variable was added, removed or changed in signature. The exported struct types that changed are exactly four:

| type | change | an unkeyed literal |
|------|--------|--------------------|
| `ConsumerOptions` | new exported field `Rewind bool` (it had one field, `Protect`) | `ConsumerOptions{quill.ProtectOn}` no longer compiles |
| `Problem` | new unexported field `refused bool` | `Problem{"segment", path, 0, 0, "d", true}` no longer compiles |
| `SegmentError` | new unexported field `oversize bool` | no longer compiles |
| `Stats` | six new exported fields: `DiskFull`, `FreeBytesAtLastPass`, `BufferedRecords`, `BufferedBytes`, `FlushFailing`, `LastFlushOK` | no longer compiles |

`Log`, `Reader` and `Consumer` also gained unexported fields in the diff, but they already had them at `v0.1.0`, so no
caller could build one with a literal. `Config`, `ReaderOptions`, `GapError`, `Hole`, `Record`, `CheckResult` and
`ConsumerInfo` did not change. A keyed literal of any of the four still compiles. `Problem.Kind` has a new value,
`"overlap"`, which is a runtime difference, not a compile one. The sentence "the only compile-time break" at
`Problem` and `SegmentError` was therefore wrong; it is replaced by this table. (A caller that only reads these types, or
assigns `Stats` from `Stat()`, is unaffected.)

The release notes must carry the same list: it is the "Breaking changes and upgrade notes" list at the top of
`CHANGELOG.md`'s "Unreleased" section, which the step below retitles `## v0.2.0` with the list under it.

## Commands that will be run

Run from a clean checkout of `main` on the release engineer's machine (macOS with bsdtar and Apple gzip: see
"Reproducibility"). Nothing here pushes. All `go` commands run with `GOWORK=off`.

```sh
# 1. The release commit: retitle the changelog section (the heading style of v0.1.0 is "## v0.1.0").
#    Edit CHANGELOG.md line 6 from "## Unreleased" to "## v0.2.0" the migration list is already at the top of that section ("Breaking changes and upgrade notes"; check it still matches the table above).
git add CHANGELOG.md
git commit -m "docs: changelog for v0.2.0"

# 2. Pre-flight on that commit.
FUZZTIME=20s hack/gate.sh
GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
test -z "$(git status --porcelain)"

# 3. The annotated tag, on the release commit. Annotated; never move it, recreate it.
git tag -a v0.2.0 -m "quill v0.2.0"
test "$(git rev-parse v0.2.0^{commit})" = "$(git rev-parse HEAD)"

# 4. The release build: refuses a dirty or untagged tree; wipes and fills build/install/.
build/quill
(cd build/install && shasum -a 256 -c SHA256SUMS)

# 5. Smoke each binary (see "Smoke test"), then stop. The release engineer's step ends here.
```

The owner's step, **not run by the rehearsal and not to be run until the public-flip checklist below is closed**. Push
the branch and the one tag by name; never `--all` or `--mirror` (the scratch branches must not leave the machine):

```sh
git push origin main
git push origin v0.2.0
awk '/^## v0.2.0/{f=1;next} /^## /{f=0} f' CHANGELOG.md > build/install/notes.md   # the section of this release
gh release create v0.2.0 --verify-tag --title "quill v0.2.0" --notes-file build/install/notes.md \
    build/install/*.tar.gz build/install/*.cdx.json build/install/SHA256SUMS
```

(Inferred: the push and `gh` lines were not run and their flags were not exercised, because the repository is private
and unpushed. The `build/quill` and tag steps were run in the rehearsal clone as shown, with the tag named
`v0.2.0` there.)

## Artifacts (verified, from the rehearsal build of `35d1185`)

`build/install/` holds 4 archives, the SBOM and `SHA256SUMS`. `shasum -a 256 -c SHA256SUMS` reports all 5 OK.

```
e6123b5f5862d74876c700eebf976a18302c6de36b51b6aca5bd0aff5d49a775  quill_v0.2.0_darwin_amd64.tar.gz
4eceb144e91d788c074a7902813b7fd3a9efa0f9d9a3451bae2cf827deb1fb5c  quill_v0.2.0_darwin_arm64.tar.gz
6e4acc030793e2d4084093cce9619a4d7d6c2d5163ec2f4d759c85af52d8878c  quill_v0.2.0_linux_amd64.tar.gz
4a3ed7381912c9563b2ca03e5b59029a1776b6eb2f05009aae2450131995ba22  quill_v0.2.0_linux_arm64.tar.gz
b89582e03501aee506f4b50002ea224c402ac69696ab9f3c570bf11022c5a867  quill_v0.2.0.cdx.json
```

**These are the rehearsal's hashes, for the tag on `35d1185`. The real tag will sit on the changelog commit, so every one
of them will differ.** Record the real `SHA256SUMS` in the release notes after step 4.

The binaries inside the archives (verified, SHA-256 of the extracted `quill`):

| platform | SHA-256 of the binary |
|----------|-----------------------|
| darwin/amd64 | `ac2b9e610a6d5632affd98934c70ff25c42abb1f68d5ef5ebd4c9cc4026083fa` |
| darwin/arm64 | `8835d7b978986666c0ae5f9f9c8fcb9829a4a711e49b575590960189762146a8` |
| linux/amd64 | `6edbbe79ea2dfa1153c133040b884f1b9d18e50b6e61fc09a30b872770ee3e0d` |
| linux/arm64 | `70a64f61a801edaa25005c129a4f4c6903f9b9d235cc72a972021ec8df0697f7` |

What is in them (verified):

- Each archive holds one directory with `CHANGELOG.md` (84,716 bytes), `README.md` (7,989 bytes), `SECURITY.md` (2,800 bytes)
  and `quill` (3.4 to 3.6 MB). **No `LICENSE`**: none exists in the repository, and the script copies one only if present
  (`build/quill` copies `LICENSE` under `[ -f LICENSE ]`), so adding one needs no script change. *(Update 2026-11-04: `LICENSE` and `THIRD-PARTY-NOTICES` now exist and each archive carries both; this paragraph is the rehearsal as run, at `35d1185`.)*
- Entries are owned by 0/0 with no names, and every mtime is the commit time.
- The SBOM is valid JSON (parsed with a JSON loader), CycloneDX 1.5. Its root component is `quill v0.2.0` with
  `vcs.commit` `35d1185b...` and `go.version` `go1.27.3`; it has exactly one library component, `go (standard library)`
  `go1.27.3` (purl `pkg:golang/stdlib@1.27.3`), and one dependency edge. `go version -m` on a linux and a darwin binary
  lists no dependency lines, so the SBOM matches what reaches the binary. (Inferred, harmless: the SBOM's stdlib version
  string carries a `go` prefix and its purl does not.)
- The binaries record `GOAMD64=v1` or `GOARM64=v8.0`, `-trimpath=true`, `CGO_ENABLED=0` and toolchain `go1.27.3`.
- `build/quill` leaves the tree clean (`build/quill.version` and `build/install` are ignored by git).
- **`SHA256SUMS` is unsigned**: no `.sig` or `.asc` is produced. A checksum file served beside the archives proves
  integrity against download damage, not against a compromised release page.

## Reproducibility (verified)

**Reproducible.** A second clone at a different, deeper path, same tag, built from a different working directory with
`TZ=Asia/Tokyo` and `GOFLAGS` empty, gave byte-identical checksums for all four archives, the SBOM and `SHA256SUMS`. The
extracted binaries match the table above. No builder path is embedded in the binaries.

Limits, as the `build/quill` header says (reported, and the rehearsal agrees):

- Archive identity holds only for the same tar and gzip implementation. Both rehearsal builds used the same bsdtar 3.5.3
  and Apple gzip. A verifier on another system compares the extracted binaries, not the archives.
- **A cross-builder check** (verified): building the tag from a `git archive` on a Debian 13 machine (Go 1.27.3) with the
  script's flags gave the same four binary hashes as the same build on the macOS machine (`linux/amd64` `9cb6a1a3...`,
  `linux/arm64` `29fdfe9d...`, `darwin/arm64` `f18f94b4...`, `darwin/amd64` `ae0401fd...`). They differ from the release
  archives' binaries because a release builds inside a git checkout and `go build` then stamps `vcs.revision` and
  `vcs.time`. Building the clone with `GOFLAGS=-buildvcs=false` reproduces `9cb6a1a3...` for `linux/amd64`, so the vcs
  stamp is the only difference.
- **A third-party verifier must pass `-buildvcs=false`, or build inside a checkout of the tag.** The release notes should
  say so.

## Version stamp and smoke test (verified)

The documented way to read the version is the verb: `quill version` prints `quill v0.2.0` on every binary
(`docs/reference.md:26` lists the verb). *Updated 2026-11-04 (audit L5-5, owner-approved):* `quill --version` and `quill -version`
print the same text and exit 0; at the time of this check they exited 2 with `flag provided but not defined: -version`.

The smoke script runs, per binary: append 100 records from standard input (`append` prints 100); `read` gives 100 lines,
`rec-1` to `rec-100`; `read -json -from 100` gives `rec-100`; `stat -json` shows 100 records; `check` exits 0 (HEALTHY);
`register -protect` then `commit -offset 51`; `consumers` shows offset 51, lag 50.

| platform | how it ran | result |
|----------|-----------|--------|
| darwin/arm64 | the release binary, natively on the build machine | SMOKE OK |
| linux/amd64 | a Debian 13 machine (kernel 6.12); the binary copied there, SHA-256 `6edbbe79...` equal to the archive's | SMOKE OK |
| linux/arm64 | an `alpine:3.20` container on arm64, no Go toolchain present, binary bind-mounted, SHA-256 `70a64f61...` equal to the archive's | SMOKE OK |
| darwin/amd64 | under Rosetta on the build machine | `quill version` only: prints `quill v0.2.0`; the smoke script was not run |

Observation (verified, not a failure): `quill check` on a log made by `quill append` prints
`UNBOUNDED: neither MaxBytes nor MinFreeBytes is set` and a `quill.policy ... no such file or directory` line while
exiting 0 HEALTHY. It reads as noise for a log that no library writer ever opened.

### Source install in a clean module (verified, with a substitute)

A throwaway module outside the clone required `github.com/netstar-labs/quill v0.2.0` with a `replace` to the clone
(`GOFLAGS=-mod=mod GOPROXY=off GOWORK=off GOTOOLCHAIN=local`), and ran a copy of `examples/log/main.go`: it printed the
orders read, the resume, and `caught up: offset 6, lag 0`, exit 0. `go.mod` stayed unchanged, there is no `go.sum`, and the
dependency list is quill's three internal packages plus the standard library. From the clone, `go run ./examples/log`
and `go run ./examples/topics` exit 0 with correct output; the pipeline example needs `-role` (writer 20, then reader 20)
and exits 0. This proves the module builds and imports cleanly from source. It does **not** prove the module-proxy path.

## Rollback

One line, valid while nothing has been pushed (verified for the tag in a scratch clone; `git describe --exact-match`
then fails):

```sh
git tag -d v0.2.0 && rm -rf build/install build/quill.version
```

and revert or reset the changelog commit that carries the release notes. **After a push, once any Go module proxy has
fetched `v0.2.0`, it cannot be recalled** (the proxy cache is permanent; inferred from how the proxy is documented to
behave, not run here). Then the answer is a new tag (say `v0.2.1`) plus a `retract v0.2.0` line in `go.mod`,
`git push origin :refs/tags/v0.2.0`, and deleting the release. The tag is annotated: recreate it, never move it.

## What was not tested, and why

- **`go install github.com/netstar-labs/quill/cmd/quill@v0.2.0` from the module proxy.** The repository is private and
  unpushed. With the `replace` removed, `go` fails with `reading github.com/netstar-labs/quill/go.mod at revision v0.2.0
  ... Repository not found` (verified). The host's `GOPRIVATE` made `go` go direct to the host despite `GOPROXY=off`; the
  probe was read-only. The README install line and the module path are therefore untested against a real remote.
- **Verification of published artifacts**: downloading the archives from a release page and checking `SHA256SUMS`
  against them, and checking that the release page serves what `build/install` held. There is no release page.
- **A build by a person other than the release engineer on a different archive toolchain** (GNU tar and gzip). Binary
  identity was shown across two builders; archive identity only on one toolchain.
- **Native darwin/amd64 hardware.** The binary was run under Rosetta for `quill version` only.
- **GitHub CI on the real repository** (the `ci` job, Dependabot, branch rulesets, private vulnerability reporting).
  None of them exist until the repository does; the gate is the local equivalent of CI's checks (`hack/gate.sh`
  header). The CI workflow's action pins were checked against the remote tags (see the checklist).
- **Signature verification**: there is no signature to verify.
- **Anything on a platform quill does not support** (Windows; FreeBSD and the 32-bit Linux targets are refused at build time by design, and the gate checks it).
- **A fresh run of govulncheck after the database moves.** It reads a live database; run it again at tag time.
- **The Linux ext4 filesystem matrix and the power-cut replays, for the tag.** Deferred by the owner on 2026-11-04 (no
  power-off is possible); the Linux gate of the pre-tag batch ran on tmpfs only. See "What was not run at the tag" below.

## Public-flip checklist

"Who decides" is the owner unless it says otherwise. States are as of this commit.

| item | state | who decides |
|------|-------|-------------|
| `LICENSE` (decisions 22, 54) | **DONE 2026-11-04** (Apache-2.0, owner decision; the text is the canonical `LICENSE-2.0.txt`, sha256 `cfc7749b...`, 11,358 bytes, fetched from apache.org by the assistant). Before it: no `LICENSE` file existed (verified: the tree and the archives). `decisions.md:30` and `:62`, `README.md:114` and `scope.md:499` say it lands before the repository goes public. The house default is Apache-2.0 with the company copyright line, to be confirmed. Once added, `build/quill` ships it with no change. | owner |
| Third-party notices | **CLEAN; the caveat is closed 2026-11-04 (A4-5, owner-approved, L5-4): `THIRD-PARTY-NOTICES` carries the Go licence text unchanged and every archive holds it beside `LICENSE`. The row below is the finding as first recorded.** **CLEAN, one caveat.** Verified: `go.mod` has no `require` line and there is no `go.sum`; `go version -m` of the binaries lists no dependencies; no copyright, SPDX, "adapted from" or licence strings in any tracked `.go`, `.sh` or `go.mod` file; `testdata/golden` is generated by quill. No `THIRD-PARTY-NOTICES` is needed for code. Inferred, for the owner or counsel: the binaries statically link the Go runtime and standard library (BSD-3, which asks for its notice in binary redistributions), the SBOM lists the standard library, and the archives carry no Go licence text. Many projects ship without; decide knowingly (see A4-5 in [supply-chain.md](passes/supply-chain.md)). | owner, or counsel |
| No internal names in tracked files at `HEAD` | **PASS** (verified): word-bounded and prefix searches for the internal product and repository names over `HEAD` find nothing. Items to decide, none a product name: `.github/CODEOWNERS` names a personal account handle and the organisation's repository path; `docs/design/audit-prompt.md` lines 39, 229, 241 and 280 still carry operator detail of the private test setup (a lock-file path in a home directory, a Linux machine, a remote runner), which `docs/audits/passes/a1-diff-scoped.md:451` already reported; `docs/audits/` carries working-process text (agents, stage 2 owner decisions). | owner |
| No internal names in history | **FIXED 2026-11-04** (verified: `git log --all -p` and a pickaxe search find nothing; 451 commits, `HEAD` tree unchanged; hashes in this record before this date are the old ones, see [commit-map.txt](commit-map.txt); the old `v0.1.0` tag is gone and the new one has a tree that differs in one document). Before the fix: **FAIL** (verified). A history search finds an internal name in main's history: commit `98ff9e7` ("docs(audit): A1 adversarial audit findings") added a scratch path that names an internal product and a home directory, and `0b70e82` removed it. `98ff9e7` is an ancestor of `main` and of `v0.1.0`, so a normal push of either publishes it. Commit and tag messages are clean (verified: zero hits). Also in history: `faa8eaf` added the audit prompt with the operator detail above (still at `HEAD`); commit `da2d638` has the message "x" and a placeholder author; the other 360 commits carry one GitHub noreply identity (owner to confirm it is meant to be public). The owner chose a filtered history (2026-11-04) and it is done. The rest of this row is still open: `faa8eaf` operator detail, the placeholder author of `da2d638`, and the identity question. See A4r-1 in [supply-chain.md](passes/supply-chain.md). | owner |
| Scratch refs must not be pushed | **OPEN** (verified). The local repository has 484 branches (`stage2/*`, `failed/*`, `worktree-*`) and `git log --all` reaches them. Push `main` and the release tag by name, never `--all` or `--mirror` (`docs/audits/passes/gap-audit.md:511` and `supply-chain.md` say the same). | owner |
| Figures from the reference implementation in `review.md` and `scope.md` | **MOSTLY DONE** (verified by reading; the A7-15 record in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md) lists what was removed). Residue: `review.md:74` keeps "minutes for 100+ GB of never-expiring log", which the record says the owner may delete. `review.md` still describes the reference's architecture in detail without a name (a network fan-in layer with a mutually authenticated transport, collector and shipper daemons, a key set); how much of the origin to describe is an owner call (A4r-5). `scope.md` has no figure taken from the reference. | owner |
| `SECURITY.md` contact | **OPEN.** `SECURITY.md` relies on GitHub private vulnerability reporting (Security tab) with an "open an issue asking for a private contact" fallback and no email address. It works only if the owner enables private reporting in the repository settings once the repository exists; it cannot be checked before then. `SECURITY.md:31-32` cites "Stage 2 audit, owner-approved 2026-11-03, A3-7", internal process wording; consider softening. | owner |
| Repository metadata (description, topics, homepage, default branch) | **PENDING**, not run. Set a description, topics (`go`, `append-only-log`, `wal`, `embedded`, `durable-log`), a homepage and the default branch `main` (the local default is `main`, verified). `README.md:93` and `go.mod` hard-code `github.com/netstar-labs/quill`: the public repository must live at exactly that path, or the module path, imports and examples change first. | owner |
| Branch protection and the `ci` aggregator check | **PENDING.** `.github/workflows/ci.yml` defines one `ci` job (`needs:` every other job in the file, `if: always()`, fails unless all results are success; verified). The ruleset must require the check named `ci`, and can only do so after it has run once on a push or pull request to `main`. `pr-issue-link.yml` makes every pull request body carry `Closes #N`: confirm that contributor friction is intended. Rulesets and private reporting do not exist yet. | owner |
| CI actions pinned | **PASS** (verified). All 10 `uses:` lines are full commit SHAs: `actions/checkout@3d3c42e5...` (v7.0.1) and `actions/setup-go@b7ad1dad...` (v7.0.0); `git ls-remote` against the actions' repositories confirms the SHAs are those tags. Dependabot is configured for GitHub Actions (weekly). Workflow permissions are `contents: read`; `pr-issue-link.yml` has `permissions: {}`. The tool pins in `ci.yml` match `hack/gate.sh` (staticcheck v0.8.2, deadcode v0.52.0), and govulncheck v1.9.0 is pinned in CI. | none |
| `CHANGELOG.md` ready for the tag | **FAIL** for tagging as the tree stands (verified): `CHANGELOG.md:6` is `## Unreleased`. Retitle it `## v0.2.0` in a commit before the real tag (the migration list is already under it); the tag then sits on that commit and the rehearsal's hashes will differ. | release engineer |

## Known limitations and open items at the tag

What the audit left open, accepted or deferred, and what was not run, collected from the records so that the owner reads one
list before the tag. Each entry has the id with the section of the record that holds it, one sentence, a state and who
decides. States: **open** (a defect or a wrong sentence that stays unless someone acts), **accepted** (decided, deliberately
not changed), **deferred** (to after the release), **not run**. Labels as at the top of this record: every entry is read from
the record it names, as the tree was at `e8909b6`, unless it says it was run here (two entries were: `quill -h` and the
architectures). "Owner" is the owner of the repository. **Updated 2026-11-04 at `2893a3f`** (pre-tag batch, audit pass L4b; see
"Pre-tag batch 2026-11-04" in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)): the L4b-3 row and A3-10 point 2 now say
what became of the two wrong sentences; nothing else in this section was re-derived at `2893a3f`.
**Updated 2026-11-04 after pass L5** (pre-tag code batch; "Pre-tag code batch 2026-11-04" in
[stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)): the three items below that the critic raised are resolved (rows L3b-15 for `-h` and `--help`, L4b-3 and "Platforms"), `THIRD-PARTY-NOTICES` closes the
Go licence caveat (A4-5), and one new open point was added in row L3b-15 (the test that pins `quill --version`); that point was closed the same day by audit L5-5 (`cf7d789`, see row L3b-15). Everything else in this section is as it was at `2893a3f`.

**Raised by the wave 6 completeness critic, not approved for change (state: open).** The owner's approval of 2026-11-04 covered
documentation only. Three things the critic raised need a code change or a change to a message, an exit code or a build
constraint, so none was touched and each stays open, for the owner's decision: (1) `quill -h` and `quill --help` print
`flag: help requested` and exit 2 (row L3b-15 below; resolved 2026-11-04, audit L5-1; `--version` resolved 2026-11-04, audit L5-5); (2) the `quill archive` message "the archive holds the same hole",
inaccurate when the destination starts after the lost records (row L4b-3 below; resolved 2026-11-04, audit L5-2, the message now reads "the archive does not have them"); (3) the module builds for 32-bit and
big-endian Linux targets that the documents do not list, and the tests do not compile for 32-bit (section "Platforms" below).
Each has a held-item section, with a recommended answer, in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md) ("Held after
the pre-tag batch"). *(Update 2026-11-04, L5-3: item (3), the 32-bit build, is resolved: the module now refuses a 32-bit target at compile time; see "Platforms" below.)*

### Behaviour and documentation that stay as they are

| id and record | what | state | who decides |
|---|---|---|---|
| L4b-3: "`quill check -dir <dest>` keeps reporting the hole after archiving a source with a damaged oldest segment" ("Follow-up to open point 2 (L4b-3, owner-approved 2026-11-04)" in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)) | The documents are corrected (documentation only; fixed in `e51c2e3`, fix-skeptic FIXED-VERIFIED, 6 of 6 lenses, reported, with the reproduction run again by the skeptic). The message `the source lost records N to M; the archive holds the same hole` that `quill archive` prints was inaccurate when the destination starts after the lost records (a new destination, or a damaged oldest segment): the destination then holds no hole and `check` on it is HEALTHY. Reproduced on a build of the tree at `bdb1522` (exit 3, then `check -dir <dest>` HEALTHY exit 0, a second `archive` exit 0). **Resolved 2026-11-04 (owner-approved, audit L5-2; commit `4ffd425`):** the message now reads `the source lost records N to M; the archive does not have them`, true in both cases; exit 3 is unchanged. | resolved (message reworded, L5-2) | owner |
| A1d-1: "a live retention pass syncs the directory once per deleted segment while holding the log's lock" ([a1-diff-scoped.md](passes/a1-diff-scoped.md); "Not held, but worth the owner's eye: A1d-1" in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)) | A live pass fsyncs the directory once per deleted segment (`retention.go:363`) with the log's lock held, so `Append`, `Read`, `Stat` and `Head` wait for the whole pass. **Reported by the auditor, not re-run here:** for 401 expired sealed segments, 2.128 s on macOS APFS against 31.3 ms at `v0.1.0`, and 203.9 ms on Linux ext4 against 15.6 ms (about 5 ms and 0.5 ms per segment), on shared machines, "indicative, not a benchmark" in that record; a live pass deletes at most a quarter of the sealed segments. Decisions 61 and 65, the user guide and the Latency section of `scope.md` say nothing about a pass blocking the log (verified: a search of those files for the words finds nothing about a retention pass); the record recommends saying so if the code stays. | accepted (decided in round 1, decision 65); the lock-held cost is undocumented | owner |
| A3-7: "a symlinked `consumers/` directory is followed" ([threat-model.md](reference/threat-model.md); "A3-7 (code hardening)" in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)) | `register`, `recover` and `Open` still follow a symlinked `consumers/`; only the wording changed (`b765290`, `SECURITY.md:27-32` (corrected from 28-32: the symlink sentence starts at line 27), decision 51). The `O_NOFOLLOW` or `Lstat` hardening was not approved; the recommended answer is to leave the code, because the exposure needs write access to the log directory. | open (held, no fix written) | owner |
| A3-24: "`-json` passes bidi and format controls raw; the time field is unchecked" ([threat-model.md](reference/threat-model.md); "A3-24 (`time` field)" in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)) | The `time` field is `time.Unix(rec.At, 0).UTC()` with no range check: `At` 253402300800 prints year 10000 and `MinInt64` prints the same string as `MaxInt64`; `at` stays exact and only a hand-crafted file reaches it. The escape set was fixed (`d15ac45`); the range check was not approved. | open (held, no fix written) | owner |
| A3-26 (merges A3-2): "unlinking `quill.lock` under a live writer lets a second writer open" ([threat-model.md](reference/threat-model.md); "A3-26" in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)) | An unlinked or replaced `quill.lock` still lets a second writer open until the first writer's next flush or prune pass, where `6acc23e` fails the first writer (`ErrFailed`); records the two acknowledge in that window can overwrite each other. Reported by the threat-model note of 2026-11-03, from the commit, not re-run. | open (held; the window is what remains after the refuted verdict) | owner |
| A3-18: "a Reader's read cost scales with the log, not the request" ([stage2-owner-decisions.md](decisions/stage2-owner-decisions.md); [threat-model.md](reference/threat-model.md)) | A `Reader` read with no iterator cursor re-walks the active segment from byte 0 to the offset and rebuilds the sealed-segment view on every call (the auditor's figures, reported and not re-run: `Reader.Read(2,999,000,1)` 379 ms against 112 us for `Log.Read`, in a 343 MB active segment). `3c36fc0` reworded `scope.md` and decision 63; caching sealed refs as `Log` does (decision 103) is deferred. | deferred (after the release) | owner |
| L1-12, the optional `Archive` refusal ("`quill archive` into a new destination copies only a suffix and exits 0"; [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)) | `Archive` into a new destination after the `_archive` offset advanced copies only the suffix and exits 0; the wording is fixed (`be8cb15`); the optional refusal, an error that is not a `*GapError`, is not implemented and needs its own decision. | open (optional, not approved) | owner |
| A5-5: "a long-lived Reader does not notice that its directory was replaced by another log" ([silent-degradation.md](passes/silent-degradation.md); "Not held, but worth the owner's eye" in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)) | A `Reader`'s plain `Read` and `Scan` are not re-checked and serve a replacement log from the old cursor; `Commit` and a Reader's wait return `ErrWrongLog` (`495e457`). The owner declined the check on cost on 2026-11-02 (decision 83); the record says decision 83 declines a per-segment check, not one `quill.format` read per view (the recorder's inference). | accepted (declined, no code change) | owner |
| A4r-3: "`build/quill` accepts any tag, not only `v0.y.z`" ([supply-chain.md](passes/supply-chain.md)) | A stray tag on `HEAD` makes a release-named build; the release engineer must spell the tag `v0.2.0`. The one-line check is in the record. | accepted (deliberately not changed) | owner |
| L3b-15: "`quill --version` and `--help` are usage errors" ([live-integration.md](passes/live-integration.md)) | **Resolved for `-h` and `--help` (owner-approved 2026-11-04, audit L5-1; commit `cfa7bf7`):** `quill -h`, `quill --help` and `quill -help` print the usage of `quill help` and exit 0 (they printed `quill: flag: help requested` and exited 2). **`quill --version` was left unchanged by that fix, and the test that pinned it was held. Resolved 2026-11-04 (audit L5-5, owner-approved; commit `cf7d789`, FIXED-VERIFIED, 7 of 7 lenses, reported):** `quill --version` and `quill -version` print what `quill version` prints and exit 0, and `{"--version"}` is gone from `TestBareQuillAndOtherGlobalFlagErrorsStayUsageErrors`; `TestGlobalVersionFlagIsTheVersionVerb` covers the flag, stamped and unstamped. | resolved (`-h`, `--help` and `--version`) | owner |

### Open points of the A3-10 fix

All four are the fix-skeptic's, from "A3-10: a damaged oldest segment is skipped silently by a default-start read" in
[stage2-owner-decisions.md](decisions/stage2-owner-decisions.md) ("Open points the fix-skeptic left, none fixed here"); reported by that
skeptic on `18662b6`, not re-run here. The behaviour change itself is owner-approved (2026-11-02).

| id and record | what | state | who decides |
|---|---|---|---|
| A3-10 point 1 | When every sealed segment of the source is damaged, `Archive` copies nothing, so it never commits and every run exits 3 until `quill commit -name _archive -offset N` goes past the gap (it exited 0 before the fix); the user guide says so. | open | owner |
| A3-10 point 2 | After an archive of a source with a damaged oldest segment, **`quill check -dir <archive>` reports HEALTHY, exit 0**, because a hole from 1 leaves no damaged file in the destination. `docs/userguide.md:393-394` said `quill check -dir <dest>` "keeps reporting the hole (exit 3)" and `scope.md:339` said `quill check` on the destination "keeps reporting it" (the A3-10 record cites `scope.md:321`; the sentence was at line 339). Both held only when a damaged file was copied. **Corrected 2026-11-04 in `e51c2e3` (documentation only, audit L4b-3, FIXED-VERIFIED):** the user guide, `scope.md`, decision 85 and the `Archive` godoc now say that `check` on the destination reports the hole only when the destination already held records below it, and prints HEALTHY when the destination starts after the lost records, so the exit 3 of the run that found the loss is the only report. A reader of this record at `e8909b6` or earlier will find the old sentences; they are gone at `e51c2e3` (verified by reading `docs/userguide.md:387-399` at `2893a3f`). The behaviour itself is unchanged; the message text was reworded 2026-11-04 (audit L5-2, row L4b-3 above). | fixed (wording, `e51c2e3`); behaviour and message unchanged | owner |
| A3-10 point 3 | `quill tail -n 21` on a log with 10 records in a hole from 1 and 20 readable records exits 1 ("records 10 to 10 are unavailable"), because 21 reaches one record into the hole; that follows the rule, no test pins it, and the owner's wording ("reaching the oldest record") is looser. | open | owner |
| A3-10 point 4 | The A5-18 notes (decision 36, the user guide, `scope.md`) say a consumer behind `First` means the log is intact and `check` stays HEALTHY; with a damaged oldest segment `check` is UNHEALTHY for the hole and also warns of the new consumer at 1. | open | owner |

### Fixes finished by hand, with no fix-skeptic re-run

Each was rejected by its fix-skeptic on one stated objection and then finished by the recorder, who closed that objection and
ran the tests; no fix-skeptic re-ran on the final state. Source: "Open after a rejected fix" and "Held after pass A7" in
[stage2-owner-decisions.md](decisions/stage2-owner-decisions.md) (reported). Each is local and revertible, as its section says.

| id and record | what | state | who decides |
|---|---|---|---|
| A7-18 ("A7-18: no upgrade, migration or rollback note for an operator"; [gap-audit.md](passes/gap-audit.md)) | The user guide's "Upgrading and rolling back" note, `e2f02b6`; whether an operator note is wanted at all was not put to the owner. | open (no re-run) | owner |
| A7-19 ("A7-19: no install path documented; a go-installed binary reports `dev`"; [gap-audit.md](passes/gap-audit.md)) | The README "Installing" section, `ffae168`; its text can only be true once the repository is public, and whether the owner wants it before the flip was not asked. | open (no re-run) | owner |
| L1-3 | `GapError.From` is the first unavailable sequence: `0aa9659`, `66c7ec3`, `2aa5a21`. | open (no re-run) | owner |
| L1-4 | "every later Sync fails" reworded to what the code does: `9c1784d`, `a4ddc8f`; the runtime text of the failed-log `Error` record is left as written on purpose. | open (no re-run) | owner |
| L1-10 | The `Config.Logger` records: `041cc34`, `8e45aa0`; one point was not checked, that a `Sync` or `Close` flush that rotates runs the caps at that seal (read from `publishLocked`, no test). | open (no re-run) | owner |
| A3-10 | `badb254` (the second attempt rebased onto main, one `scope.md` conflict resolved by hand) and `ed518a2` (two comments in `archive.go` and a user-guide note). The skeptic of `18662b6` passed the code and tests and rejected on the comments; no skeptic ran on `ed518a2`, whose diff against `18662b6` is comments and documentation only (reported). | open (no re-run of the final state) | owner |

### Platforms: the 64-bit-only claim

`README.md:109` ("Linux and darwin only (arm64, amd64); other platforms fail to build on purpose") and `scope.md:461` ("Linux and
darwin (arm64, amd64). Other platforms fail to build, on purpose") list two architectures and say other platforms fail to
build. **Verified here** (go1.27.3 on darwin/arm64, `GOWORK=off`, 2026-11-04):

- `GOOS=linux go build ./...` exits 0 for `GOARCH` 386, arm, riscv64 and s390x: the module builds for architectures the
  documents do not list, two of them 32-bit (386, arm) and one big-endian (s390x). `GOOS=freebsd go build ./...` exits 1, as
  the gate requires.
- `GOOS=linux go test -c .` exits 1 for 386 and arm (`log_test.go:195` onward: `cannot use math.MaxInt64 ... as int value in
  struct literal (overflows)`), so the test suite does not compile for a 32-bit target. It compiles (exit 0) for riscv64 and
  s390x.
- **Not run:** the library or any test on a 32-bit or big-endian target, so nothing here shows the code is correct there. The
  gap audit says the same: "32-bit and unlisted GOARCH behaviour was not checked beyond building linux/386 and riscv64"
  ([gap-audit.md](passes/gap-audit.md), Coverage).

State: **resolved 2026-11-04 (owner-approved, audit L5-3; `27cbbb1`)**. The owner chose
enforcement: only 64-bit builds. `config.go` declares `quillRequires64BitPlatform = int(^uint(0) >> 63)` and an array type of
length `quillRequires64BitPlatform - 1`, which the compiler rejects when `int` is 32 bits, so the root package, and with it
`cmd/quill`, does not compile for a 32-bit target. The refusal is for the width of `int`, not an architecture allow-list.

**Run for the fix (verified, go1.27.3 on darwin/arm64, `GOWORK=off`, 2026-11-04), `GOOS=linux GOARCH=<arch> go build ./...` for
every `linux/*` pair in `go tool dist list`:**

- Fail, with `config.go:19:8: invalid array length quillRequires64BitPlatform - 1 (constant -1 of type int)`: 386, arm, mips,
  mipsle (exactly the four 32-bit pairs).
- Build, exit 0: amd64, arm64, loong64, mips64, mips64le, ppc64, ppc64le, riscv64, s390x.
- `GOOS=darwin GOARCH=amd64` and `arm64` build; `go vet ./...` passes for `GOOS=linux` and `GOOS=darwin`; `GOOS=freebsd go build
  ./...` still exits 1; `GOOS=linux GOARCH=386 go build ./cmd/quill` fails with the same message, through the import.
- `hack/gate.sh` and `.github/workflows/ci.yml` now assert that linux/386 and linux/arm do not build.

**Still not run (stated, not hidden):** the library or any test on any target but the host's arm64 and, in earlier waves,
Linux amd64 and arm64. The 64-bit targets other than arm64 and amd64 build and are untested; the documents say "tested on arm64
and amd64". For s390x, the only big-endian 64-bit target that builds here: `codec.go` encodes the frame with
`encoding/binary` varints and `binary.BigEndian` for the CRC, and a search of the non-test code finds no `unsafe` and no
`NativeEndian` (verified by `grep`), so the frame bytes do not depend on the host's byte order by that reading; no test ran on
a big-endian machine, so this is inferred from the code, not shown.

### What was not run at the tag

"The tag" means the commit it will sit on. It is 41 commits after `ba30fee`, the last commit [live-integration.md](passes/live-integration.md)
names: 41 commits as of `e8909b6`, 45 as of `2893a3f` (verified: `git rev-list --count ba30fee..HEAD` at each).

**The Linux gate of the pre-tag batch (2026-11-04) ran on tmpfs only** (reported by the owner's conditions for the batch, as
given with the task: the box is used through the harness script, which keeps the snapshot, `TMPDIR` and the Go cache on a RAM
tmpfs; not re-derived here). This record's own commit changes documentation only and ran nothing heavy. The ext4 filesystem
matrix and the power-cut replays were **deferred by the owner** (no power-off is possible) and were not run for this batch; the
first two rows below are therefore not new evidence for the tag.

**Pass L5 (2026-11-04, the four code fixes `cfa7bf7`, `4ffd425`, `27cbbb1`, `29f785c`): what ran and what did not (reported by the fix-skeptics; the recorder of this entry ran nothing).** Each fix-skeptic ran `gofmt`,
`go vet` and `go test -race -count=1 ./...` on its commit (L5-1, L5-2, L5-4), or `gofmt`, `go vet` for the host, linux and darwin, the race tests and `staticcheck` v0.8.2 (L5-3). The ledger records `staticcheck` and `deadcode` as clean for the fixer of L5-1 and says "all checks passed" for L5-4; for L5-2 and L5-3 it does not say. **Not run by a skeptic:** `staticcheck` and `deadcode` for L5-1, L5-2 and L5-4, `deadcode` for L5-3, the fuzz step and `go mod tidy`
of `hack/gate.sh`, and `hack/gate.sh` as a whole script for any of the four (the L5-3 skeptic ran its two build assertions by hand). The ext4 matrix and the power-cut replays stay deferred by the owner. The Linux box was
used once: the L5-3 skeptic built, vetted and cross-built `GOARCH=386` at `27cbbb1` on Linux amd64 through the harness. The 64-bit targets other than arm64 and amd64 were built, not run; no test ran on
a 32-bit or a big-endian machine. For L5-4 the reproducibility claim is two `build/quill --dev` runs with identical `SHA256SUMS` (the skeptic's run, reported) and the fixture test; the archives
carry one more member, so the hashes of a real release differ from the rehearsal's table above, which predates `LICENSE` and `THIRD-PARTY-NOTICES`.

| what | what exists | state | who decides |
|---|---|---|---|
| Power-cut replays: a real `sysrq` cut, `dm-log-writes` crash states, `dm-flakey` cycles | One real cut, 240 `dm-log-writes` states over six journalling filesystems and 64 `dm-flakey` cycles, run on the Linux box in pass L3b against `a63bd6b` and the fixes up to `ba30fee` (reported: "Power loss" in [live-integration.md](passes/live-integration.md)); earlier real cuts of 2026-10-28 ([power-loss.md](reference/power-loss.md)). None re-run on the tag. | deferred by the owner (no power-off is possible); not run | owner |
| The Linux filesystem matrix (ext4 in four modes, xfs, btrfs, f2fs, tmpfs, overlayfs) and the long topology runs | Ran once in pass L3b on the same tree (reported: "Linux filesystem matrix" in [live-integration.md](passes/live-integration.md)); the longest run was 540 s on Linux and 100 s on macOS ("What the tier does not reach"). Not re-run on the tag. | deferred by the owner; not run | owner |
| A day-long soak | "No soak of hours" ([live-integration.md](passes/live-integration.md), "Duration"). | not run | owner |
| A 10^5-segment `Reader` or `Check` run | No record holds one: the largest segment counts in the records are 40,000 one-record segments (a skeptic's `Reader` runs for A3-18 in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md)) and 401 (A1d-1). The `Reader`'s per-call view is O(sealed segments) (A3-18), so the cost at 10^5 is **inferred**, not measured. | not run | owner |
| `go install github.com/netstar-labs/quill/cmd/quill@v0.2.0` from the module proxy | See "What was not tested, and why" above: the repository is private and unpushed. | not run | owner (it cannot run until the repository is public) |
| Branch rulesets and private vulnerability reporting | They do not exist until the repository does (public-flip checklist above). | not run | owner |

The deferral in the first two rows is the owner's, given with the request for this section (audit item L4b-2); no earlier record
states it, so it is reported here and not read from one.

## Failures and findings, ranked

1. **Release blocker.** `CHANGELOG.md:6` is `## Unreleased`; fix in a commit before tagging.
2. **Done 2026-11-04.** The scratch path in history was filtered out (see above); no longer a blocker. Previously: filter or start
   fresh. Also `da2d638` and the 484 scratch branches that must not be pushed.
3. **Done 2026-11-04.** The Go licence text is in the archives as `THIRD-PARTY-NOTICES` (A4-5, owner-approved, L5-4); `LICENSE` itself is done (Apache-2.0, 2026-11-04). No longer a blocker.
4. **Minor.** `quill --version` exited 2 (only the `version` verb worked; fixed 2026-11-04, audit L5-5: it now prints the version and exits 0); `build/quill` accepts any tag name;
   `docs/design/audit-prompt.md` still carries operator detail; `.github/CODEOWNERS` names a personal handle;
   `SHA256SUMS` is unsigned.
5. **Not testable.** The real `go install ...@v0.2.0` path, private vulnerability reporting, rulesets and repository
   metadata, until the repository exists.
6. **Open at the tag, owner's to read.** The audit findings that stayed open, accepted or deferred, the fixes finished by
   hand without a fix-skeptic re-run, the 64-bit claim and what was not run: "Known limitations and open items at the tag"
   above.

Deviation from the planned order (verified): the gate ran in the main rehearsal clone, while the two build refusals were
proven in a separate scratch clone of the same tag, so the refusals did not dirty the gated tree. The clean build and
the reproducibility runs were then made in the clones the plan named.
