# A1 adversarial audit, round 2: findings

Date 2026-10-29, against HEAD 1da37fc (v0.1.0 plus one docs-only commit; working tree clean).
Round 2 of the A1 audit, run with fresh eyes before a public release: eight independent
auditors in four dimensions (simplify, dedup, correctness, docs), every candidate attacked by
two to four skeptic votes that read the code and reran the reproductions, and every fix
re-attacked by a fix-skeptic. The specification is docs/design/scope.md, docs/design/decisions.md
and docs/format.md; the code was the thing under audit, so a passing test and a doc were both
treated as claims, not evidence. The per-dimension reports carry the full items, anchors and
reproductions: [simplify](a1-round2-simplify.md), [dedup](a1-round2-dedup.md),
[correctness](a1-round2-correctness.md), [docs](a1-round2-docs.md). What the audit held back for
the owner is in [stage2-owner-decisions.md](stage2-owner-decisions.md). Round 1 is recorded in
audit-findings.md and its siblings.

Labels: **verified** means an auditor or skeptic ran it or read the exact lines; **reported**
names the document that says it; **inferred** says from what. Anchors (file:line) are to the
audited tree 1da37fc; the fixes have since moved some lines. Reproduction programs lived in
throwaway copies of the repository in the session scratch area (written `scratch/<label>` below);
nothing under audit was edited by an auditor or skeptic.

## Top line

97 candidate ids were raised by the auditors; 24 were merged into 20 other items as duplicates,
leaving **73 distinct items** (verified: arithmetic over the ledger, 73 + 24 = 97). They were
judged by **142 skeptic votes** (125 CONFIRMED, 2 PLAUSIBLE, 15 REFUTED); **48 of the 73 items
were reproduced by execution** by at least one skeptic. The 7 items round 1 had already decided
were not re-judged (no votes). The auditors' own reproductions are reported in the items; they
are not counted in the 48.

| verdict | items | note |
|---------|-------|------|
| CONFIRMED | 58 | 40 fixed, 1 fix-failed, 17 held for the owner |
| PLAUSIBLE | 3 | A1-18 and A1-37 (one REFUTED vote each; fixed anyway), A1-91 (held) |
| REFUTED | 5 | kept, with the one-line reason |
| prior-deliberate | 7 | decided in round 1 (decisions 13, 38, 47, 53, 85, 101, 103, 105 and the allocation pass) |

| disposition | items | what it means |
|-------------|-------|---------------|
| fixed | 42 | a commit exists, is an ancestor of the tree this record is written on, and passed the fix-skeptic (FIXED-VERIFIED, no lens failed) |
| fix-failed | 1 | A1-62: a fix was written and the fix-skeptic reproduced a hole in it; it is not integrated |
| held for the owner | 18 | the fix changes observable behaviour or a guarantee's wording; each has a recommended answer |
| refuted | 5 | not changed, because the claim does not hold |
| round-1-decided | 7 | deliberately not changed in round 1; see the re-raise note below |

| dimension | items | fixed | fix-failed | held | refuted | round-1-decided |
|-----------|-------|-------|------------|------|---------|-----------------|
| simplify | 9 | 6 | 0 | 0 | 1 | 2 |
| dedup | 12 | 10 | 0 | 1 | 1 | 0 |
| correctness | 37 | 15 | 1 | 13 | 3 | 5 |
| docs | 15 | 11 | 0 | 4 | 0 | 0 |
| total | 73 | 42 | 1 | 18 | 5 | 7 |

By the severity the auditors gave: 5 high, 17 medium, 51 low. The five high items are A1-42
(`tail -follow` prints nothing until 4 KiB accumulates), A1-43 (a quill.tail offset of 2^63 or
more panics every Reader call) and A1-66 (the documented manual-corruption procedure fails at
`quill recover`), all fixed; and A1-67 (the documented `quill commit` rewind of an ahead consumer
cannot work) and A1-80 (a live protected consumer stops the writer from restarting), both held.

**What matters most for release.** Held, not fixed: A1-80 and A1-67 (above), A1-25 (after a failed
rotation the active segment can outgrow the 1.5 GiB limit, and the log then cannot be reopened),
A1-82 (a transient read error is reported as ErrCorrupt, or as a gap that the documented resume
idiom skips) and A1-32 (the reserved archiver consumer cannot be removed or rewound). Fixed, and
worth the owner's eye because they change an error class, an exit code or the fsync count: A1-26
(the last sequence number is no longer assignable), A1-46 (a failed write to stdout now exits 1),
A1-64 (Archive refuses its own source as a destination) and A1-31 (a directory fsync before each
segment unlink, n fsyncs per n-segment prune).

## Round-1 items re-raised

Seven items were raised again by fresh auditors although round 1 had decided them: A1-1
(decision 105), A1-5 (allocation pass), A1-30 (decision 13), A1-54 (decision 101), A1-56
(decisions 47 and 38), A1-58 (decisions 85 and 53) and A1-59 (decision 103). Two of the seven
(A1-58 and A1-59) were found independently by two auditors (A1-41 and A1-83 were merged into
them), so nine raisings in all. The ledger does not record which auditor raised which id, so the
split by dimension is not given. That fresh eyes re-found a decided item is evidence the round-1
verdict deserves another look. Two stand out (the recommendations are the recorder's):

- **A1-59**: the Log's own reads return a raw ENOENT where the Reader returns a GapError. The
  finding's rebuttal says decision 103's sentence "a deleted one fails at its open as before"
  contradicts guarantee 14. The auditor's real-concurrency reproduction produced a raw ENOENT in
  25 of 1,601 reads on macOS and 17 of 1,483 on Linux (verified by the auditor, not re-run by a
  skeptic: this item had no round-2 votes). The proposed fix is one mapping in the open closure at
  read.go:295. Recommendation: reopen the round-1 decision and take the fix.
- **A1-54**: the Reader fails on the marker's own segment when a cap deletes it between the marker
  read and the open. The window is narrow (inferred; not caught live) and decision 101 makes the
  error deliberate. Recommendation: leave it.

The other five are cost or design trade-offs the round-1 text already argues: A1-1 would remove
about 170 non-test lines but makes the marker canonical-only; A1-5 saves about 63 ns of a 5 ms
flush; A1-30 is the accepted fsync under the lock; A1-56 is the cheap `stat`; A1-58 is the exit
code of `archive` over a hole. Items whose text cites a round-1 neighbour but were raised as new
(for example A1-66, A1-70, A1-72, A1-79 and A1-97, each a neighbouring row of a round-1 doc item)
are not counted here.

## Coverage

Eight auditors returned; none failed to return. Platforms: macOS arm64 for everything; the Linux
x86-64 box (Debian 13, ext4, 4 vCPU, Go 1.27.3) for the race suite, the real-ENOSPC test, the CLI
reproductions called out below and the fix verification; no auditor used an Apple container
(inferred acceptable: no finding is platform-specific except where noted). No real power cut was
possible; crash behaviour is shown only against the repository's crash simulator, whose unlink
persistence model is stricter than journalled filesystems. Not examined by anyone: XFS, btrfs,
APFS-specific behaviour, NFS and other network filesystems, 32-bit builds.

### Checked and found clean, by auditor

**A (simplify).** Read every non-test .go file in the root package (21), cmd/quill (7),
internal/atomicfile, internal/fsx (os.go, fs.go, crash.go, crash_file.go), internal/strictjson,
hack/powertest and the three examples, plus hack/gate.sh, hack/fuzz.sh and build/quill; did not read
docs/audits, CHANGELOG.md or git history. Clean or load-bearing, so not raised: `osFlags` (the
darwin constants differ, verified 0x200 and 0x800 against 0x40 and 0x80); the verbs table built in
`init()` (a package-level initializer would be an initialization cycle); `Log.mkdirAll` (the
per-parent fsync of decision 62); `openSegmentFlag`'s Lstat (a FIFO cannot block); the
`walk(prev, havePrev)` pair; `scanSource`'s `expect` callback; `deleteLocked`'s `chosen` map
(decision 93); `sealedRefs` caching (decision 103: 2.4 us and 30 allocations per read at a thousand
segments); the hand-written record printer in cmd/quill (2.5x to 7.7x faster than the
encoding/json and strconv formulation, 0 allocations against 6 to 7). Considered and left:
`Reader.readTail`'s five-attempt retry, the BinarySearchFunc comparators, `Reader.closed` as an
atomic, the retention offsets slice. Every field of the `cli` struct and every Config field is read
in non-test code. `deadcode` without `-test` reports only `WriteSyncDir` outside the simulator
(A1-7). Latent only: `var nextID` at internal/fsx/crash.go:218 is a package global that would race
if two simulators ran in parallel tests; no test uses `t.Parallel`. Platforms: timings on macOS
arm64; one `go test ./...` of the combined candidates on Linux; no -race, fuzz or container run.

**B (dedup).** Read every non-test Go file and ran a normalized-window duplicate finder (windows of
3, 4 and 5 lines) over all Go files; did not read line by line the bodies of retention_test,
archive_test, consumer_test, seek_test, iter_test, bench_test, fuzz_test, strictjson_test or
atomicfile_test; did not read docs/audits or CHANGELOG.md. Prototypes ran in copies only; macOS
only. Considered and not recommended, with reasons: the Log/Reader/Consumer wrapper triplets;
`appendTail` against `decodeTailFast` and `decodeTailSlow` (decision 105 needs the
allocation-free decoder); the four digit scanners (four different rules); `appendJSONString`
against `appendQuoted` (two unrelated formats pinned byte for byte by decision 35);
`OpenReader`'s defaults against `Config.withDefaults` (different inputs are errors); the
pooled-window pattern (three lifetimes); the two sealed-segment open closures; the three
frame-error conversions; `writeState` as the single seam; the `c.errorf("%v", err)` idiom;
`lag()` against `Consumer.Lag`; the Log.Stat and Reader.Stat prefix; the simulator's 26 PathError
literals. Clean checks: all lock handling goes through `takeLock`; all atomic writes go through
atomicfile; frame decoding has one home; `contiguity` is the single copy; option and name
validation exist once.

**C1 (write path).** Fsync ordering of every file operation in a rotation (segment, next segment,
SyncDir, marker, SyncDir) for new log, rotation, Roll and Close; deletion publishes the marker and
syncs the directory before any unlink; on macOS `File.Sync` is F_FULLFSYNC (verified in the Go
source). A crash sweep of the auditor's own (a crash before every file operation of a workload
with appends, flushes, Roll and retention, 150 random persist masks per point) found no violation:
the log reopens, has no holes and holds every record whose Flush or Roll returned nil. A
creation-protocol sweep (first Open at `/log` and `/a/b/log`, 80 masks each) found none. Error
injection at 191 points (ENOSPC, EIO, short writes) and a double fault (short write then failed
Truncate) found no duplicate sequence and no partial frame left, and `ErrFailed` is never cleared.
A real ENOSPC on a 2 MiB tmpfs on the Linux box: the flush fails wrapping ENOSPC, the segment size
and Head are unchanged, appends are accepted to MaxBufferBytes and then refused, and after space is
freed all 831 records come back. No descriptor leak after a failed Open at any operation or after
Close, and no goroutine leak over 200 open/close cycles under -race. The Sync and group-commit
state machine, rotation and Roll, and the overflow arithmetic of the batch check were read and
found right (64-bit only). `freeBytes` equals `df` on ext4 and tmpfs (XFS and btrfs not checked).
`Open` on a directory holding other files leaves `quill.lock` behind: that is decision 62's order,
not raised. hack/powertest was read, not run. `BenchmarkAppendBuffered` shows 0 allocations. The
auditor left three run directories on the Linux box.

**C2 (recovery and retention).** Parse-forward recovery on a real filesystem against a mutated
log: a torn tail and a zero-filled tail past the durable offset are truncated; a bad CRC below the
durable offset, an empty newest file with the marker at off>0, a marker naming a higher segment
than any on disk, and a marker ahead of the data each give ErrCorrupt with the file unchanged; a
stale marker naming an older segment still recovers. Sealed-segment damage (bad CRC, empty file,
deleted middle segment) leaves Open succeeding with the right hole and the file untouched. A
randomized damage harness (250 iterations, seed 7, macOS) never modified a sealed file, never
shrank the active file below the durable offset on a successful Open, and every Open failure was
ErrCorrupt. Deletion ordering, the retention arithmetic (age prefix, protection, the MaxBytes and
MaxRecords loops, Dropped, the clock guard 20 to 16 to 12), ConsumerTTL, the consumer file
protocol, name validation and archive resume were read and exercised. Not covered: Linux and
container runs; most of reader.go, iter.go and seek.go; MinFreeBytes on a real full disk; the race
between the unlocked `consumers()` scan and a delete (read, not exercised).

**C3 (reader and CLI).** Read reader.go, read.go, iter.go, seek.go, internal/strictjson and
cmd/quill/*.go line by line, plus codec.go, load.go, flush.go, holes.go, stat.go and archive.go
where the CLI reaches them. A -race stress run (15 s, 20,000 records, 2 KiB segments, tiny
retention, a 1 ms reader poll) returned strictly increasing sequences with matching payloads and
only *GapError when retention overtook the reader; torn bytes after the marker are never served.
`OffsetForTime` and `SeekToTime` matched a linear scan in a 30-round differential test for every
second from first-2 to last+2, and extreme times return without overflow. Iterator and Read
buffers, cancellation, integer handling and CLI number parsing (`1e6`, `1M`, `-1`, ` 10`, a
20-digit value and others are refused with exit 2) behave. `-json` was checked on 265,792
payloads (every byte, every byte pair, 200,000 random mixes): every line valid, lossless, one line
per record; only C1 controls differ from the text form (A1-50). Writer verbs beside a live writer
exit 1 and change nothing; the read verbs and `archive`, `commit`, `register`, `unregister` and
`check` run beside it. Mentioned, not raised: terminal escapes in file names printed by
`consumers`, `check` and `ls`, and a FIFO in place of quill.tail blocking `stat` (SECURITY.md
excludes a process with write access to the directory); the JSON and text `time` field outside
years 0 to 9999; `Reader.Stat` counting every segment name including ids above the marker. Linux
was used only for the /dev/full, `tail -follow` and oversized-offset reproductions.

**C4 (concurrency and crash).** `go test -race -count=1 ./...` passed on the Linux box for every
package. A chaos test (150 iterations, macOS, -race: three appenders, a Flush, Roll and Prune loop,
Sync, Log.Follow, Reader.Follow, a consumer Read and Commit loop, repeated Archive, all racing
Close at a random moment) found, after the one Sync race (A1-61), no race, deadlock or goroutine
leak, no unexpected error, and every directory reopened with zero holes; the chaos test is not in
the repository. Lock ordering (consumer.mu then log.mu, never the other way), Close, timer and
goroutine lifecycles, flush failure classes, rotation crash points and the sharing of slices were
read and found right. The crash model was read in full: production relies only on properties the
simulator grants, except Archive's parent-directory entry (A1-63). The existing crash property
tests do not drive Archive; that gap is why A1-63, A1-64 and A1-65 were not caught before. Not
covered: a real power cut, a kill -9 or fault-injection loop on the box, containers, network
filesystems, and the claim that ext4 and XFS fsync commit the parent's pending mkdir (general
knowledge, not tested).

**D1 (user docs).** `docs/reference.md` is byte-identical to `quill reference` (verified by diff)
and a test keeps it current. The verb table, help text and exit codes 0, 1, 2 and 3 match. Every
user guide section 7 command was run against scratch logs and the output formats match. `archive`
(idempotent, readable destination), all three examples (`go test ./examples/...` is green), the
Go doc comments' behaviours (ErrLocked after about 200 ms, name rules, the ErrConfig cases,
ErrNoLog, double Close, Read(0,n)), the Config defaults and ceilings in section 6, architecture
sections 3 to 6, every relative link in README, CONTRIBUTING, SECURITY, examples/README and
docs/*.md, the gate and CI tool pins, the unsupported-platform build refusal, the file-size bound,
CODEOWNERS, and a grep for forbidden internal names (none outside docs/audits and CHANGELOG.md)
were checked clean. Recovery and ErrCorrupt rules other than the procedure itself match. Not run:
the full gate (staticcheck and deadcode need the network, so the gate finding is from reading);
benchmark figures; fsync-failure, full-disk, ConsumerTTL and Dropped behaviours; docs/format.md
and docs/design were read only for the numbers the user docs cite. The ErrAhead,
corruption-procedure, 500k-line append, symlink and build/quill reproductions ran on Linux too.

**D2 (design docs).** Read scope.md, decisions.md (1 to 105), format.md and build-prompt.md in
full, every non-test source file, and the tests for T13, T16, T17, T21, T23 and those named in the
candidates. Guarantees were checked against about 25 single-site mutants: guarantees 2, 3, 4, 5, 6,
7, 8, 9, 11, 12, 13 and 14 each had a named test fail on the mutant (guarantee 11 only at 100x or
no cap, weak at 4x: A1-94); guarantees 1 and 10 were not mutated separately. Other mutants all
caught: ErrAhead removed, archive without CRC verify, a non-strict seek comparison (decision 59),
an empty sealed segment accepted, a future format accepted, MaxBytes below twice SegmentBytes
accepted, ErrBadSeq removed (the suite hangs, which is a detection), the clock guard rounding
changed. The three golden frames in format.md match `appendFrame` exactly; the Config table, the
error list (every error except ErrSeqExhausted is exercised by a test, now A1-26's regression test
too), the CLI table and build-prompt.md's "Out of scope" list match. Decisions with no drift
found: 1, 3-9, 11-20, 22-28, 30-35, 37-40, 42-43, 47-48, 50-52, 54-56, 58-59, 61-62, 64-66, 68,
70-71, 73-77, 79 (load path), 81-85, 87-95, 97-98, 100-102, 104-105. The suite passes on macOS and
the Linux box including T14 on a 4 MiB tmpfs; freebsd, windows and openbsd refuse to build as
scope.md says. Release scaffolding per decisions 54 and 88 (no LICENSE while private,
CONTRIBUTING, CHANGELOG, SECURITY, a build script that refuses a dirty or untagged tree and writes
SHA256SUMS and a CycloneDX 1.5 SBOM) is present. Not run: cross-user permission effects (A1-85
says so), the Alpine container. The mutation sample is not exhaustive.

### Not covered by anyone

Real power loss; XFS, btrfs, NFS and APFS-specific behaviour; 32-bit targets; the simulator's own
tests as a subject; docs/audits and CHANGELOG.md (A, B and D1 did not read them by design);
guarantees 1 and 10 by mutation; any benchmark on a second machine.

## Fixes and the fix-skeptic

Forty-two fixes were written one per branch (`stage2/A1-<n>`), each re-attacked by a fix-skeptic
through six or seven lenses (the claim's test fails without the fix; root cause closed including
sibling forms; behaviour preserved with gofmt, vet, `go test -race`, golden and `go doc` diffs; no
new gap; reachable by the real path; least-code ladder; and on some an independent route). All 42
returned FIXED-VERIFIED with no lens failed (one lens was not run for A1-19, whose mutation check
the sandbox denied). Doc-only fixes were checked more lightly (A1-73 ran tests without -race,
A1-70 raced only ./examples/...). Only some fix-skeptics ran staticcheck and deadcode; none
reports a failure. Their objections that stand, none of which blocks a fix, are in the dimension
reports under each item. All 42 commits are ancestors of the tree this record is written on
(verified with git). The one fix that did not pass is A1-62 (commit fdcd880, which is not in this
tree).

## Index

Legend: votes C = CONFIRMED, P = PLAUSIBLE, R = REFUTED; "repro" Y means at least one skeptic
reproduced it by execution; the commit is shown by its first 7 characters.

### Correctness (37 items, a1-round2-correctness.md)

| id | anchor | sev | verdict | votes | repro | disposition |
|----|--------|-----|---------|-------|-------|-------------|
| A1-42 | cmd/quill/read.go:131-156 | high | CONFIRMED | CC | Y | fixed 4337252 |
| A1-43 | reader.go:299-311, state.go:98-136 | high | CONFIRMED | CC | Y | fixed a4f92e8 |
| A1-80 | log.go:325-333, consumer.go:81 | high | CONFIRMED | CC | Y | held |
| A1-33 | retention.go:242-249 | medium | CONFIRMED | CC | Y | fixed c2a7365 |
| A1-60 | reader.go:250-265 | medium | CONFIRMED | CC | Y | fixed c8fb07b |
| A1-46 | cmd/quill/read.go:133, print.go:32 | medium | CONFIRMED | CC | Y | fixed 9c6a8c3 |
| A1-51 | stat.go:138-186 | medium | CONFIRMED | CC | Y | fixed 89211cc |
| A1-25 | flush.go:104-130, load.go:76 | medium | CONFIRMED | CC | Y | held |
| A1-32 | consumer.go:73-79, 267-269 | medium | CONFIRMED | CC | Y | held |
| A1-47 | cmd/quill/verbs.go:159-169 | medium | CONFIRMED | CC | Y | held |
| A1-48 | cmd/quill/verbs.go:71, 124-125 | medium | CONFIRMED | CC | Y | held |
| A1-82 | read.go:249-257 | medium | CONFIRMED | CC | Y | held |
| A1-85 | consumer.go:313, archive.go:25 | medium | CONFIRMED | CC | Y | held |
| A1-34 | consumer.go:245-260, retention.go:38-41 | medium | REFUTED | RCR | Y | refuted |
| A1-49 | cmd/quill/write.go:19-41 | medium | REFUTED | CRR | Y | refuted |
| A1-59 | read.go:295-296 | medium | prior-deliberate | - | N | round-1-decided (decision 103); revisit |
| A1-26 | append.go:61-77, flush.go:227 | low | CONFIRMED | CC | Y | fixed 890c70e |
| A1-31 | retention.go:225-250 | low | CONFIRMED | PC | Y | fixed e2f7801 |
| A1-37 | load.go:339-348 | low | PLAUSIBLE | CRC | N | fixed 89073cb |
| A1-64 | archive.go:74, 94-126 | low | CONFIRMED | CC | Y | fixed 5b486d2 |
| A1-53 | reader.go:42, 199-247 | low | CONFIRMED | CC | Y | fixed 8f29e05 |
| A1-61 | flush.go:226-229 | low | CONFIRMED | CC | Y | fixed 3208dab |
| A1-62 | retention.go:94-274 | low | CONFIRMED | CC | Y | fix-failed (fdcd880) |
| A1-63 | archive.go:96, 126 | low | CONFIRMED | CC | Y | fixed 7672a8d |
| A1-65 | archive.go:146 | low | CONFIRMED | CC | N | fixed db30a94 |
| A1-78 | cmd/quill/verbs.go:45-47 | low | CONFIRMED | CC | Y | fixed 8449425 |
| A1-29 | log.go:338-358 | low | CONFIRMED | CC | Y | held |
| A1-35 | retention.go:111-123 | low | CONFIRMED | CC | N | held |
| A1-50 | cmd/quill/print.go:81-107 | low | CONFIRMED | CC | Y | held |
| A1-52 | read.go:161-242 | low | CONFIRMED | CC | Y | held |
| A1-57 | cmd/quill/read.go:81-82 | low | CONFIRMED | CC | Y | held |
| A1-91 | state.go:300 | low | PLAUSIBLE | PRC | Y | held |
| A1-27 | flush.go:61 | low | REFUTED | CRRR | Y | refuted |
| A1-30 | flush.go:17-24 | low | prior-deliberate | - | N | round-1-decided (decision 13) |
| A1-54 | reader.go:281-303 | low | prior-deliberate | - | N | round-1-decided (decision 101) |
| A1-56 | reader.go:337-352 | low | prior-deliberate | - | N | round-1-decided (decisions 47, 38) |
| A1-58 | cmd/quill/consumers.go:142-147 | low | prior-deliberate | - | N | round-1-decided (decisions 85, 53) |

### Docs (15 items, a1-round2-docs.md)

| id | anchor | sev | verdict | votes | repro | disposition |
|----|--------|-----|---------|-------|-------|-------------|
| A1-66 | docs/userguide.md:243-271 | high | CONFIRMED | CC | Y | fixed 3439946 |
| A1-67 | consumer.go:120-122, docs/userguide.md:98 | high | CONFIRMED | CC | Y | held |
| A1-68 | build/quill:25 | medium | CONFIRMED | CC | Y | fixed df7af4f |
| A1-69 | append.go:30-33 | medium | CONFIRMED | CC | Y | held |
| A1-70 | examples/README.md:3-4 | low | CONFIRMED | CC | N | fixed 4aec960 |
| A1-71 | log.go:38-39, retention.go:97 | low | CONFIRMED | CC | N | fixed 88a44b9 |
| A1-72 | docs/userguide.md:181-182 | low | CONFIRMED | CC | N | fixed da4d1ce |
| A1-73 | SECURITY.md:14 | low | CONFIRMED | CC | N | fixed 082bbcd |
| A1-74 | hack/gate.sh:48-59 | low | CONFIRMED | CC | N | fixed e498890 |
| A1-79 | docs/reference.md:5, README.md, godoc | low | CONFIRMED | CC | N | fixed a593c45 |
| A1-94 | retention_test.go:289-330 | low | CONFIRMED | CC | Y | fixed 47a5ebb |
| A1-95 | retention.go:238, 126, flush.go:130 | low | CONFIRMED | CC | Y | fixed 9810a56 |
| A1-97 | docs/format.md:179 and others | low | CONFIRMED | CC | N | fixed f1df4ae |
| A1-2 | strictjson.go:30-49, state.go:138-158 | low | CONFIRMED | CC | Y | held |
| A1-87 | retention.go:139 | low | CONFIRMED | CC | N | held |

### Dedup (12 items, a1-round2-dedup.md)

| id | anchor | sev | verdict | votes | repro | disposition |
|----|--------|-----|---------|-------|-------|-------------|
| A1-13 | load.go:277-296, archive.go:99-111 | medium | CONFIRMED | CC | Y | held |
| A1-9 | archive.go:95 | low | CONFIRMED | CC | Y | fixed 48e7491 |
| A1-14 | flush.go:167-171, log.go:483-493, read.go:333-341 | low | CONFIRMED | CC | Y | fixed afe3daa |
| A1-16 | load.go:413-427 and four more sites | low | CONFIRMED | CC | Y | fixed d39a915 |
| A1-17 | kill_test.go:22-34 and two more | low | CONFIRMED | CC | Y | fixed 860c483 |
| A1-18 | internal/fsx/crash_test.go | low | PLAUSIBLE | CRC | N | fixed e37f4ee |
| A1-19 | cmd/quill/cli_test.go:35-51 and two more | low | CONFIRMED | CC | Y | fixed 0e3b548 |
| A1-20 | archive_test.go:20 and eight more | low | CONFIRMED | CC | Y | fixed 74b1fc8 |
| A1-21 | internal/fsx/crash_file.go:45-156 | low | CONFIRMED | CC | N | fixed 8fe2264 |
| A1-22 | cmd/quill/verbs.go:166, 203, 243, 249 | low | CONFIRMED | CC | Y | fixed 0614afc |
| A1-24 | kill_test.go:84-87 | low | CONFIRMED | CC | N | fixed cba6016 |
| A1-11 | codec.go:75-85, load.go:147-155 | low | REFUTED | CRR | Y | refuted |

### Simplify (9 items, a1-round2-simplify.md)

| id | anchor | sev | verdict | votes | repro | disposition |
|----|--------|-----|---------|-------|-------|-------------|
| A1-1 | state.go:91-158, internal/strictjson | medium | prior-deliberate | - | N | round-1-decided (decision 105) |
| A1-3 | read.go:164-268 | low | CONFIRMED | CC | Y | fixed 1b1ff46 |
| A1-4 | state.go:38-53, 264-276 | low | CONFIRMED | CC | Y | fixed a2113b5 |
| A1-5 | atomicfile.go:37-53, fsx/os.go:60-75 | low | prior-deliberate | - | N | round-1-decided (allocation pass) |
| A1-6 | stat.go:42-63 | low | CONFIRMED | CC | N | fixed f0cdcfa |
| A1-7 | atomicfile.go:89, fsx/fs.go:56-57, disk.go:5-7 | low | CONFIRMED | CC | N | fixed af99624 |
| A1-8 | internal/fsx/os.go:16-17, fs.go:63 | low | CONFIRMED | CC | Y | fixed 8f89e0d |
| A1-12 | read.go:293, load.go:312, flush.go:267-274 | low | CONFIRMED | CC | N | fixed becd44e |
| A1-10 | read.go:145 | low | REFUTED | CRRR | N | refuted |

The assignment of an item to a dimension is the recorder's, made from the item's kind and from the
auditor evidence in its grounds (the ledger carries no auditor field); it changes no verdict.
