# Pass A6: format and version-skew audit

Date 2026-10-31. Pass A6 of the stage 2 audit, run with fresh eyes before a public release, against the tree at 988f1e5
(the parent of the first A3 fix). The specification is docs/design/scope.md, docs/design/decisions.md and docs/format.md;
the code was the thing under audit, so a passing test and a doc were both treated as claims, not evidence. The question
of the pass: is every encoded artifact pinned by bytes that were not computed from the code under test, and does a build
from HEAD read and write what the v0.1.0 tag reads and writes, in both directions, in the mixed-version state a real
deployment is in? The earlier passes are recorded in [a1-round2-findings.md](a1-round2-findings.md) and its siblings, and
[threat-model.md](threat-model.md) (A3); what this pass held back for the owner is in
[stage2-owner-decisions.md](stage2-owner-decisions.md), section "Pass A6".

Labels: **verified** means an auditor, skeptic or the recorder ran it or read the exact lines; **reported** names the
document that says it; **inferred** says from what. Anchors (file:line) are to the audited tree unless marked "now"; the
A3 fixes that landed after it (acf7f39 to 837c860) moved some lines. Reproduction programs, mutation harnesses and the
old and new binaries lived in throwaway copies of the repository made with `git archive`, in a scratch area outside the
tree, on macOS (Darwin 25.6, arm64, Go 1.27.3) and on a Linux x86-64 box (Debian 13, ext4). They are named below by
what they do and are not committed. Nothing under audit was edited by an auditor or skeptic. The recorder of this file
re-ran one test (see "What the recorder checked") and read the seven fix commits and the A3-8 commit; it did not
re-run any reproduction.

## Top line

9 items, none merged. 7 compat, 1 bug (A6-9), 1 hygiene (A6-8). 4 medium and 5 low, as rated by the auditors. No
critical and no high item. Nothing is a live defect in the encoded bytes: the format and the v0.1.0 to HEAD skew are
clean. What the pass found is missing pins (A6-1, A6-2, A6-3, A6-7), one CLI behaviour that destroys the file mode the
archive and consumer code now depend on (A6-9), and documentation gaps (A6-5, A6-6).

| verdict | items | ids |
|---------|-------|-----|
| CONFIRMED | 8 | A6-1, A6-2, A6-3, A6-4, A6-5, A6-7, A6-8, A6-9 |
| PLAUSIBLE | 1 | A6-6 |
| REFUTED | 0 | none |

| disposition (as the ledger recorded it) | items | ids |
|------------------------------------------|-------|-----|
| fixed, fix-skeptic FIXED-VERIFIED | 7 | A6-1, A6-2, A6-3, A6-5, A6-6, A6-7, A6-9 |
| fix-failed (the commit did not land) | 1 | A6-8 |
| held for the owner | 1 | A6-4 |

One correction to the ledger, found by the recorder (verified, `git show bb646c0` and `git log`): A6-4 was recorded as held
with "no fix written", but its proposed fix (refuse a rotation whose next id exceeds 9999999999, state the ceiling in
format.md, add a test seeded at the top id) is already in the tree as the A3 pass's fix for A3-8, commit bb646c0, which
is an ancestor of this commit. A6-4 is therefore closed in the tree this record is committed with, by another pass's
commit; the owner section keeps it as a "confirm and close" item. A6-8 is not an owner decision either: its fix was
written as a plain rename and the commit never landed, for a workflow reason (see its entry).

- Skeptic votes: 18 item votes (two per item). 17 CONFIRMED and 1 PLAUSIBLE (A6-6, one of its two voters). No vote
  REFUTED any item.
- Reproduced by execution: 7 of the 9 items were reproduced by at least one skeptic (A6-1, A6-2, A6-3, A6-9 by both;
  A6-4, A6-5, A6-7 by one). 11 of the 18 votes carried a reproduction. A6-6 (an absence of evidence) and A6-8 (a naming
  issue) have nothing to execute; their voters read the tree. The auditors also reproduced A6-1, A6-4, A6-5 and A6-9
  themselves before voting.
- Fix commits: 7, all ancestors of this commit (verified from `git log`): 21e7d4c (A6-1), f70ef3d (A6-2), 7dea97a (A6-3),
  709c5c5 (A6-9), 70ed460 (A6-5), ee11f57 (A6-6), 49bc53c (A6-7).
- Fix-skeptic verdicts: 7 of 7 FIXED-VERIFIED, through 49 lens verdicts (7 lenses each: test-fails-without-fix,
  root-cause, behaviour-preserved, no-new-gap, reachable, ladder, independent-route), every one "pass". Six of the seven
  fixes are test-only or documentation-only, so for them the test lens is a mutation of production code showing the new
  test fails (A6-1, A6-2, A6-3, A6-7) or a check of the doc's statements against the code (A6-5, A6-6). A6-9 changed
  production code and was shown red before and green after.
- Goldens: the frame, tail, format, consumer and segment goldens are unchanged by this pass; the quill.policy golden was
  replaced (A6-7). 20 mutations of the golden-protected encoders and decoders were run against the goldens; every one
  failed the golden tests (table below).
- Skew: v0.1.0 against HEAD, both directions, two platforms, seven experiment families and a 78-case state-file fuzz: the
  encoded bytes are identical; the differences are the seven decoder changes and a handful of CLI changes that CHANGELOG
  Unreleased already announces. No unannounced difference was found.
- Not green: the skew harness built in this pass was rejected by its reviewer (see "The skew harness").

## Coverage

Two auditors, both returned: **A6-inventory-and-pins** (inventory of encoded artifacts, golden and pin mutations,
doc-to-bytes checks, fast and slow tail decoder differential, name collisions) and **A6-skew-run** (old and new binaries
run against each other). No auditor failed to return.

Checked and found clean (every line below is verified by the auditor named, unless marked):

- Frame encoder and decoder. One encoder (the only caller of `appendFrame` is the append path) and one strict decoder; the
  CRC-less `frameHeader` prefix reader on the writer's own verified buffer is pinned by two mutations. The three frames
  in docs/format.md, decoded by hand and by an independent Python encoder, equal `recordedFrames` and
  testdata/golden/segment.log. A recorded negative `at` takes 10 bytes. The Linux x86-64 box passes the golden tests;
  the frame uses `encoding/binary`, so no endianness or word-size dependence exists (verified on x86-64 and arm64;
  32-bit not run).
- The strict tail decoder and its fast path (decision 105) agree: 400,000 random mutations of canonical tails over 13
  boundary values plus all 28,561 canonical combinations gave 0 disagreements (25,604 fast accepts, each accepted
  identically by the slow path).
- The examples in docs/format.md equal the goldens byte for byte for the frame, tail, format and consumer files; the
  quill.policy example did not (A6-7, since fixed). The limits table (64 MiB body, payload 64 MiB less 32, 1.5 GiB
  segment, 1 MiB state file) matches the code.
- v0.1.0 against HEAD: see the experiments below.
- Name collisions: no two non-test functions in one package share a name; the only case-twin constants are A6-8.
  `parseUint` (refuses leading zeros, reads canonical files) against the CLI's `parseDecimal` (accepts `007`, reads user
  input) differ on purpose and are not a collision.
- Not raised, already known and decided: `quill archive` over a source hole exits 0 with the loss on stderr (A1-58,
  decision 85); consumer files created through a Reader take the log's mode where v0.1.0 used 0600 (A1-85, documented).
  No `.gitattributes` pins testdata/golden against line-ending conversion; that matters only on the unsupported Windows
  platform and is not raised.

Not covered (bounds stated by the auditors):

- Power loss: kill -9 only, so the marker and data ordering is exercised and fsync is not (docs/audits/power-loss.md owns
  that).
- A segment file at or over 1.5 GiB written by v0.1.0 and opened by HEAD (it needs a rotation failure to produce).
- A second user account (the access-loss half of A6-9 is inferred from POSIX permission rules; modes were checked, a
  second account was not used).
- Apple container linux/arm64 for the skew run; Linux/arm64 for the mutation run; Windows is out of scope.
- A real reference-implementation log (A6-6).
- A SegmentBytes of 1 GiB with a buffer over 512 MiB (the ErrConfig refusal announced in CHANGELOG), because it needs
  that much memory.
- The mutation run was static reading plus mutation on one platform (macOS arm64); the golden tests were also run on the
  Linux box.

Harness mistakes found and fixed during the skew run, none affecting results: an unset-argument run deleted the scratch
`new/` binary directory (rebuilt; guards added), a quadratic payload regeneration in the verify tool made the first
Linux kill run crawl (reordered; those runs were redone), and the `trunc=` column of the kill experiment over-reports when
recovery rotates a segment already at SegmentBytes (real tears were identified by head equal to the marker sequence with
trailing bytes below the smallest frame).

## Inventory of encoded artifacts

Every artifact quill writes or reads, verified by reading the encoders and decoders and docs/format.md. "Version" means
the artifact itself carries a version; only `quill.format` does. "Silent-change consequence" is what happens if a future
edit changed the bytes without anyone noticing, and is inferred from the decoders unless marked.

| artifact | encoding | version | one encoder | pinned by | silent-change consequence |
|----------|----------|---------|-------------|-----------|---------------------------|
| segment frame | uvarint(bodyLen), then the body (uvarint(seq), uvarint(at as uint64), uvarint(payloadLen), payload), then crc32 IEEE big-endian over the body (codec.go:75-82 now) | none | yes (`appendFrame`) | `recordedFrames`, testdata/golden/segment.log, format.md table | a layout change fails the CRC or the exact-length check, so it is a loud gap, unless the new bytes still match the CRC; a swapped `seq` and `at` would decode as wrong values with a valid CRC, which is why the goldens are checked-in hex |
| segment name | `%010d.log` (state.go:36) | none | yes | `TestParseSegmentNameIsTenDigitsAndLog` | a name that does not fit ten digits is treated as a foreign file (A6-4, closed by A3-8) |
| `quill.tail` | strict JSON `{"seq","seg","off","maxseg"}` plus LF | none | yes (`encodeTail`) | testdata/golden/quill.tail, `TestAppendTailMatchesJSON`, `TestDecodeTailFastAgreesWithSlow` | a missing or unknown field is ErrCorrupt in any process that re-reads it, so a shape change is loud |
| `quill.format` | `quill/1 <32 hex>` plus LF | yes, the only one | yes | testdata/golden/quill.format, `TestFormatMarkerRoundTripsAndRefusesTheFuture` | checked at open only (A6-5); a bump is invisible to running processes |
| `quill.policy` | indented JSON, tolerant decode (decision 48) | none | yes (`writePolicy`, `encodePolicy`) | testdata/golden/quill.policy, policy_fields_test.go (A6-1, A6-7) | a changed meaning of an existing key would be accepted: unknown fields are ignored, and before A6-1 the key-to-field mapping was unpinned |
| `consumers/{name}` | `<id> <next> <0 or 1>` plus LF | none | yes (`encodeConsumer`) | testdata/golden/consumer, consumer-default | a shape change is ErrCorrupt ("want 3 fields") in any process that re-reads it |
| `quill.lock`, `consumers/{name}.lock` | empty flock files | none | n/a | names pinned by tests | no bytes to pin; a renamed lock file would let two writers run |
| archive destination | `quill.format` + `consumers/` + `quill.tail` (maxseg is the last copied id) + copied segments, all through the same encoders (archive.go) | as above | yes | the artifacts above, plus the archive tests | none of its own |
| `.tmp-` prefix | temp files for atomic replace (internal/atomicfile) | none | yes | test of the prefix | a rename would leave temp files that a scan treats as foreign |
| CLI `-json` | `read`/`cat`/`tail` record form; `stat`, `check`, `consumers` objects built from map literals | none (documented keys, scope.md) | yes | record form by golden; the rest by A6-3 | a renamed key breaks every script that parses it |
| CLI exit codes | 0 ok, 1 error, 2 usage, 3 UNHEALTHY (decision 53) | none | yes | literals, since A6-2 | a changed value breaks scripts that branch on the exit status of `quill check` |
| hashes and ids | only `crc32.IEEE`, a 128-bit random log id, a random temp suffix | none | yes | goldens | no other hash exists (grep of crypto/ and hash/) |

Segment size limits (verified against the code at the audited tree): 64 MiB frame body, payload 64 MiB less 32, 1.5 GiB
segment file (the bound on `off`), 1 GiB the configuration ceiling for SegmentBytes, 1 MiB per state file. Anchors now:
codec.go:29-34 for the first two; load.go:19 for the segment file limit; log.go:97 for the SegmentBytes ceiling;
state.go:355 for the state file limit.

## Skew verdict

**No skew between the v0.1.0 tag and HEAD in any encoded artifact; each direction reads and appends to the other's
logs. Confidence high for the artifacts and states exercised, medium for the states listed under "Not covered".**

Grounds (verified unless marked):

- No encoder changed between the tag and the audited tree: `git diff` of codec.go is comments only, and the segment,
  tail, format, consumer and policy bytes produced by the two builds under a constant clock are identical (the format
  file differs only in its random log id).
- testdata/golden and golden_test.go were unchanged between the tag and the audited tree, so the goldens pin the same
  bytes for both.
- Decoders narrowed between the tag and HEAD (a tail `off` above 1.5 GiB, a tail `seq` of 2^64-1, a tail `off` at 2^63
  where v0.1.0 panicked with a slice bound, a policy with no retention). None of these is a value either build writes,
  so a log written by either is read by both. All are announced in CHANGELOG Unreleased (reported, CHANGELOG.md).
- The old build refuses a `quill/2` marker with its own ErrFutureVersion, and both builds write `quill/1` only, so HEAD
  does not create a directory that v0.1.0 refuses.

## Goldens added and the mutation check of each

The frame, tail, format, consumer and policy goldens are pasted bytes (`recordedFrames` hex and files under
testdata/golden), not computed from the code under test at test time. They were recorded from quill's own encoder
(golden_test.go record path, docs/audits/test-ledger.md:5); see A6-6 for what that does and does not prove.

Mutations the auditor ran against the goldens (root package, `go test -run Golden`; every one failed as intended):

| id | mutation | caught by |
|----|----------|-----------|
| g01 | swap `seq` and `at` in `appendFrame` | TestFrameGolden |
| g02 | CRC Castagnoli instead of IEEE | TestFrameGolden |
| g03 | CRC little-endian in the encoder | TestFrameGolden |
| g04 | CRC over the length prefix too | TestFrameGolden |
| g05 | `at` plus one | TestFrameGolden |
| g18 | payload-length varint dropped | TestFrameGolden |
| g19 | fixed 4-byte length prefix | TestFrameGolden |
| r01g | decoder swaps `seq` and `at` | TestFrameGolden ("frame DECODING CHANGED") |
| g06 | tail `seg` and `off` swapped | TestStateFileGoldens |
| g07 | tail key `maxseg` renamed `max_seg` | TestStateFileGoldens |
| g08 | tail without the newline | TestStateFileGoldens |
| g09 | format prefix `quill/` renamed `quill-` | TestStateFileGoldens |
| g10 | consumer fields reordered | TestStateFileGoldens |
| g11 | consumer tab separators | TestStateFileGoldens |
| g12 | consumer protect flag `1` written as `t` | TestStateFileGoldens |
| g13 | policy struct fields reordered | TestStateFileGoldens |
| g14 | policy tag renamed | TestStateFileGoldens |
| g15 | policy indent 2 changed to 4 | TestStateFileGoldens |
| g16 | policy without the final newline | TestStateFileGoldens |
| g17 | `formatVersion` 1 changed to 2 | TestStateFileGoldens |

Pins that held under mutation by tests other than the goldens (verified by the auditor; a failure at least once per
mutation): decoder `seq`/`at` swap, consumer protect flag inverted, future-version check loosened, tail fast path
without the `seg <= maxseg` check, `maxSegmentBytes` 1.5 GiB changed to 1 GiB, `maxFrameBody` 64 MiB changed to 32 MiB
(caught only indirectly, through the state-file and config-default tests), renames of `quill.tail`, `quill.format`,
`consumers`, `quill.lock`, `quill.policy`, the `.lock` and `.log` suffixes and the `.tmp-` prefix, `read -json` key and
field order, the `check` status value, and the writer's CRC-less header reader (two mutations), the policy
`segment_bytes` key.

Pins added by this pass, each with the mutation check its fix-skeptic ran (verified; details in each item):

| pin | file | mutations that fail it |
|-----|------|------------------------|
| every Config field to its own `quill.policy` key, written and read, nine distinct non-zero values (A6-1) | policy_fields_test.go | each of nine fields dropped on the read side and on the write side, the MaxBytes and MinFreeBytes swap on both sides, the Retention and SegmentMaxAge swap on the read side: all caught |
| exit codes 0, 1, 2, 3 by integer literal, and the help text (A6-2) | cmd/quill/exitcodes_test.go | each of the four constants changed, and both prose sites changed: all caught |
| exact key set and JSON type of `stat`, `check`, `consumers -json` (A6-3) | cmd/quill/json_shape_test.go | six renames or drops from the claim, plus three more (`error`, holes as null, a string for `marker_age_seconds`): all caught; the value of `unbounded` is pinned by type only, and a flipped value passes |
| the quill.policy golden with nine distinct non-zero values and a producible `protected_consumers`, compared whole (A6-7) | golden_test.go, testdata/golden/quill.policy | field swaps in `decodePolicy` (caught; they passed before), a tail decoder with `seg+1`, a one-byte edit of segment.log: all caught |
| offline write verbs keep the mode of `quill.tail` (A6-9) | cmd/quill/offline_perm_test.go | fails on the parent's write.go: tail 600 and segment 600 where 660 is wanted |

## Merge-or-pin calls

"Pin" means a test or golden was added or strengthened; "merge" means two paths were joined; "keep apart" is the
recorded decision not to merge. Each call and its reason (the reasoning is the recorder's reading of the ledger; the
facts are as cited).

| call | what | reason |
|------|------|--------|
| pin | the Config to `quill.policy` mapping, both directions (A6-1) | `quill prune` is destructive and takes its caps from `ReadPolicy`; the old golden had zero caps and equal durations, so nothing could tell two fields apart |
| pin | the exit codes and the `-json` key sets (A6-2, A6-3) | scripts depend on them; a test that reads the constant on both sides cannot fail |
| pin, not golden-bytes | `-json` objects by key set and type | scope.md says key order is not part of the contract, so a byte golden would over-pin |
| merge into one fixture | the policy golden and `distinctPolicyJSON` (A6-7) | the golden is asserted equal to the pasted literal the writer test already pins, so the file and the writer cannot drift apart |
| keep apart | the fast and slow tail decoders (decision 105) | the fast path is the hot path; the differential run (0 disagreements in 400,000 mutations) and `TestDecodeTailFastAgreesWithSlow` pin them to each other |
| keep apart | `frameHeader` (CRC-less prefix reader) and the strict frame decoder | the writer reads its own verified buffer; two mutations of the header reader fail the Batch and Read tests |
| keep apart | `parseUint` and the CLI's `parseDecimal` | one reads files that must be canonical, the other user input |
| not merged (A6-8) | `maxSegmentBytes` and `MaxSegmentBytes` | the rename was written but did not land; the exported name is API and stays |
| docs instead of code | the format check at open only (A6-5); no reference-written fixture (A6-6) | re-checking the marker per call costs I/O and is not needed while only format 1 exists; no real reference frame was available and fabricating one would repeat the gap |

## Real v0.1.0 against HEAD experiments

Setup (verified): old binaries were built from `git archive v0.1.0` plus a throwaway driver (a `cmd/skew` program with
write, verify and read verbs, and a one-line clock hook), new binaries from `git archive HEAD` (988f1e5) plus the same two
files. `diff -rq` against fresh archives showed only those additions. The driver source is identical in both. Runs on
macOS arm64 (Go 1.27.3, umask 022) and Linux amd64 ext4 (umask 0002). Payload size classes in every full run: 0, 1, 17,
100, 4095, 4096, 4097, 1 MiB less 1, 1 MiB plus 1, and every 50th record at the maximum payload less 0, 1 and 2 (about
64 MiB); 7 to 11 segments per log; a consumer committed at 70 of 140; a protected consumer from
`Config.ProtectedConsumers`; retention 48 h, caps and ConsumerTTL set in the policy. Not run: Apple container
linux/arm64.

| experiment | what was run | result (verified) |
|------------|--------------|-------------------|
| 1 and 2: read and check, both directions | one version writes 140 records, the other runs `cat -b64`, `read -json -b64`, `check`, `consumers`, `consumers -json`, `check -json`, `ls`, `stat`; then the roles swap | all 140 records byte-identical (221,176,507 and 221,168,704 bytes); `skew verify` sha 7343a6d86f78c3b1 on both versions and both platforms; `check` HEALTHY; all outputs identical except `stat -json` holes, `null` in v0.1.0 and `[]` in HEAD (A1-57, announced) |
| determinism | the same appends under a constant clock (`-n 140 -fixed 1700000000`, then `-n 30 -small -seg 4096 -fixed 1700000100`); a log whose marker starts at seq 1099511627775 with timestamps -5, 4102444800, 253402300799 | segment files, `quill.tail`, `quill.policy` byte-identical; `quill.format` differs only in its random id; the edge frames (multi-byte varints, negative `at`) cross-appended and read by both with 0 bad payloads |
| 3: append with the other version | old writes 140, new opens and appends 60 (head 200), and the reverse; `quill append hello world ''`; `roll` by one, `recover` by the other | both readers see the same count (203 in the CLI case), equal sha, HEALTHY, state files parse in both |
| 4: kill -9 mid-append | hold a writer with SyncAppends false, SIGKILL after 0.05 to 0.9 s, alternate default and `-seg 65536`, recover and append 20 with the other version, verify and check with both; a small-record variant and a `-big` variant (512 KiB to 1.5 MiB records so the kill can land in a write) | 180 iterations (20 per direction on macOS, 25 per direction on Linux, two variants): every one recovered with rc 0, appended with rc 0, both versions read identical output, 0 bad payloads, 0 gaps, `check` HEALTHY. Torn tails occurred (for example 167,154 bytes beyond the marker, recovered to the marker sequence, then 20 appended). Marker lag (data beyond `quill.tail`) happened in 11 to 17 of every 20 to 25 iterations. The 80 macOS and 100 Linux pre-recovery copies were also read by both versions without recovery, identically. Bound: process kill only |
| torn tail, stale marker, hole, damaged middle segment | stale marker, 3 bytes cut, 50 garbage bytes, partial length prefix, middle segment deleted (hole 18-32), one byte flipped in a middle segment; `check` before, `recover` by each version, then the other appends 10 | segments, consumers and `quill.tail` byte-identical after recovery; identical holes (`hole 18-32`, exit 3 UNHEALTHY); identical on both platforms and directions |
| 5: consumer files and `quill.policy` | `register -protect`, `commit -offset`, `commit -time`, read, re-protect, unprotect, unregister, re-register, alternating versions; policy for default, retention off (-1ns), caps, 36 h plus MaxBytes, extreme values | formats identical, listings identical, HEALTHY; policy bytes identical; a retention-off policy written by the other version deleted nothing on `prune` with no flags |
| retention differential | segments aged 100 h, 60 h and now, a protected and a middle consumer; `prune` on copies with each version (default, `-max-records 100 -max-bytes 40000`, `-force`) | directories byte-identical after prune in every case. One documented divergence: after `register -no-protect prot` (also in the policy's `protected_consumers`), `prune -force` by v0.1.0 re-protects it and keeps everything, HEAD does not and deletes the expired segment (CHANGELOG: offline verbs open without the policy's ProtectedConsumers) |
| 6: archive | archive by one version, open and append to the destination with the other; incremental archive resumed across versions three times; both directions | segments byte-identical to the source; archive, source and both readers agree; `_archive` consumer read and advanced by both. Documented differences: `commit -name _archive` refused by v0.1.0 and accepted by HEAD (A1-32); archive files take the log's `quill.tail` mode in HEAD and 0600 in v0.1.0 (A1-85) |
| 7: following and live processes | an old Reader following from seq 1 while a new writer appends and rotates (`-seg 32768`, synced appends), and the reverse; the other version's verbs beside a live writer | followed count equals or is one above the last printed ack (319/319, 232 against 231, 3707, 3675), 0 bad, no gap; after kill and recover the whole log verifies. Beside a live writer: stat, check, consumers, tail, register, commit and archive work in the other version; append, roll, recover, prune and Open fail with "locked by another holder" and change nothing (flock interoperates) |
| future marker | `quill.format` `quill/2` and `quill/10`; malformed versions (99999999999999999999, 0, 01, 1.5, missing id, extra field, no newline, QUILL/1) | HEAD returns ErrFutureVersion from Open, OpenReader and every CLI verb; Archive into a `quill/2` destination is ErrFutureVersion in HEAD and ErrCorrupt in v0.1.0 (A1-13, announced); malformed versions are ErrCorrupt with the same wording in both; v0.1.0 also refuses `quill/2` with its own ErrFutureVersion. The exit-status column in that experiment's output is unreliable (a PIPESTATUS slip inside `$()`); the messages are the evidence |
| differential state-file fuzz | 78 mutated `quill.tail`, `consumers/c1`, `quill.format` and `quill.policy` files, each through stat, check, consumers, cat, ls, Open plus append and verify with both binaries | identical on both platforms except 7 cases, all announced in CHANGELOG Unreleased: tail seq 2^64-1 (HEAD ErrCorrupt, v0.1.0 accepts and reports a hole); tail `off` above 1.5 GiB (HEAD refuses at parse, v0.1.0 later, with a different message for 1610612737); tail `off` 2^63 (v0.1.0 panics with a slice bound in stat and cat, HEAD ErrCorrupt); a consumer ahead of the head (HEAD adds a note and a check warning); a policy with empty or null retention (HEAD unreadable, v0.1.0 reads zeros). Whitespace, order, case of one key, leading zeros, sign, `1e1`, CRLF, BOM, trailing newline, uppercase or short id, tabs: accepted or refused identically |
| the re-run | a small log: HEAD wrote 300 records and a consumer, v0.1.0 then appended 100 | both builds `check` HEALTHY, list the same consumers, `cat -json` byte-identical (300 lines), 400 records with identical sha |

Documented differences between the tag and HEAD seen in the runs and not raised, because CHANGELOG Unreleased states
them (reported, CHANGELOG.md): the `stat -json` holes shape (A1-57); `prune` no longer applies the policy's
ProtectedConsumers; `_archive` can be committed and unregistered; the doubled `quill: quill:` error prefix is gone;
Archive, Reader and consumer file modes; a policy with no retention is unreadable; ahead-of-head consumer notes; a
SegmentBytes plus flush overshoot refused as ErrConfig (the last was not run).

Reproducing the experiments needs the throwaway driver and the experiment scripts, which are not committed. The
commands above name what each ran; the old and new trees are `git archive v0.1.0` and `git archive 988f1e5` plus the two
added files.

## The skew harness

A harness was built in this pass to rerun the old-against-new experiments from the tree (hack/skew.sh in the working
copy it was built in). It is not in the tree this record is committed with (`git ls-files` lists only hack/fuzz.sh,
hack/gate.sh and hack/powertest), and its reviewer rejected it. Objection, verified by the reviewer: the harness
cannot detect a cross-version difference in recovery. The reviewer changed the new version's `recoverSegment` (load.go)
so that it dropped the post-marker frames v0.1.0 salvages; the run still ended "OK: no divergence between v0.1.0 and HEAD"
with rc 0. The kill phase recovers each killed directory with one version only and checks that the survivors are a
contiguous, unchanged prefix; it never recovers a copy of the same killed bytes with both versions and compares. The
frame-width, CRC and policy faults were caught. The fix the reviewer proposed is cheap: copy the killed directory,
recover one copy with each version, and `diff -r` the two before the append. The manual kill experiment above does make
that comparison (recovery by each version of identical torn bytes, results identical), so the verdict above stands; what
is missing is the harness that would keep proving it. Status: open, listed in
[stage2-owner-decisions.md](stage2-owner-decisions.md).

## Items

Each item: Claim, Grounds, Warrant, Backing, Rebuttal, Verdict, Disposition. Skeptic reasoning is summarised; "reproduced"
says what a skeptic ran.

### A6-1: neither direction of the Config to quill.policy mapping is pinned (medium, compat)

**Claim.** Any wrong or dropped field in `ReadPolicy` (state.go:381-383 audited, 392-408 now) or `writePolicy` (log.go:643-645
audited, 655-664 now) passes the whole suite, yet `quill prune` takes retention and every cap from `ReadPolicy`.

**Grounds.** Mutations in a scratch copy of HEAD, each run with `go test ./...` and no failure (verified): swap MaxBytes and
MinFreeBytes in `ReadPolicy`; drop Retention, SegmentMaxAge, MaxBytes, MinFreeBytes, MaxRecords, MaxPayloadBytes,
ConsumerTTL or ProtectedConsumers in `ReadPolicy`; swap Retention and SegmentMaxAge; swap MaxBytes and MinFreeBytes in
`writePolicy`; drop MaxRecords in `writePolicy`. Only the SegmentBytes drop (TestOfflineAppendTakesTheSegmentSizeFromThePolicy)
and one write-side mutation, incidentally, were caught. Cause: no test calls `ReadPolicy` directly, and the golden policy
had `max_bytes`, `min_free_bytes`, `max_records` of 0 and retention equal to segment_max_age (24h0m0s), so no value could
tell two fields apart.

**Warrant.** The offline verbs act on what `ReadPolicy` returns, and prune is destructive and irreversible. A mapping error
is silent, and only a test that sets a distinct non-zero value per field would see it. Mixed versions are the normal
deployment, so the mapping is a cross-version contract.

**Backing.** docs/format.md (offline tools take settings from quill.policy); decisions 68 and 99; scope.md "Format
stability" (every state file frozen by golden tests).

**Rebuttal.** The mapping is correct today (read line by line). This is a missing pin, not a live defect, and the deletion
needs a policy with a non-zero `max_bytes`.

**Verdict.** CONFIRMED, 2 of 2 votes, both reproduced. Reproduction (verified): the auditor's program made a log with
`max_bytes` 1125899906842624 and five sealed segments (100 records); `quill prune` built from HEAD printed `segments 6 -> 6`;
built from the swap mutation it printed `segments 6 -> 1, bytes 11200 -> 0, first 1 -> 101` (the byte cap read as a
free-space floor of 1 PiB; all records deleted). A skeptic repeated it from a fresh copy: the swapped build deleted all
100 records (`first 1 -> 101`) where HEAD kept them, and the full suite passed with the swap.

**Disposition.** Fixed in 21e7d4c (test only, policy_fields_test.go, 93 lines); fix-skeptic FIXED-VERIFIED, 7 of 7 lenses.
`TestOpenWritesEveryConfigFieldToItsOwnPolicyKey` opens a log with nine distinct non-zero fields and compares
`quill.policy` byte for byte with a pasted literal; `TestReadPolicyTakesEveryPolicyKeyIntoItsOwnConfigField` reads that
literal and compares field by field and whole. The fix-skeptic made 21 mutations; all nine fields were caught on each
side, and the swaps. The proposed flagless-`prune` CLI test was not added: `cmdPrune` passes the `ReadPolicy` Config
through with no mapping of its own (cmd/quill/write.go, 130-135 audited), so the pin is at the root; the wiring from
Config to caps has coverage in cmd/quill/offline_policy_test.go. Residual: the gate's `go mod tidy` step could not run in
the skeptic's git-archive copy (no repository there); every other gate step passed.

### A6-2: the CLI exit codes are not pinned (medium, compat)

**Claim.** The exit codes (0 ok, 1 error, 2 usage, 3 UNHEALTHY; decision 53, docs/reference.md:6) are compared in tests
against the constants, so changing a constant's value passes every test.

**Grounds.** cmd/quill/verbs.go:14-19 defines the constants; cmd/quill/cli_test.go (360, 364, 376, 380 and others) assert
`code != exitUnhealthy` and `exitUsage`. Mutations exitUnhealthy 3 to 4 and exitUsage 2 to 64: `go test ./...` all ok
(verified). The documented prose (verbs.go:245, 273, generated into docs/reference.md) is a string literal, so nothing ties
it to the value.

**Warrant.** A test that reads the constant on both sides is a tautology. Scripts distinguish 3 (unhealthy) from 1 (could
not run); a changed value is a silent contract break.

**Backing.** Decision 53; docs/reference.md and `quill help`; scope.md CLI section.

**Rebuttal.** The constants are correct today. The auditor mutated only 2 and 3 and allowed that 0 and 1 might be pinned
by literal elsewhere.

**Verdict.** CONFIRMED, 2 of 2 votes, both reproduced. The rebuttal was refuted: one skeptic changed exitError 1 to 9,
exitUsage 2 to 64 and exitUnhealthy 3 to 4 together and `go test ./cmd/quill/` passed; the other mutated exitOK and exitError
each to 9 and both passed, and built a binary that exited 64 on a usage error while `quill help` still said "2 usage".

**Disposition.** Fixed in f70ef3d (test only, cmd/quill/exitcodes_test.go, 55 lines); fix-skeptic FIXED-VERIFIED, 7 of 7.
The tests drive `run()` and compare to integer literals (healthy check 0, help 0, version 0, missing directory 1, unknown
flag 2, unknown verb 2, no verb 2, missing `-dir` 2, damaged check 3 in text and JSON); a second test pins the prose
"0 ok, 1 error, 2 usage, 3" at both sites. The fix-skeptic changed each of the four constants and both prose sites, and
each failed. The pinned staticcheck could not run locally for the fix-skeptic (the installed version failed on the
toolchain's export data); the commit adds only test code.

### A6-3: the documented `-json` shapes of stat, check and consumers are mostly unpinned (medium, compat)

**Claim.** Renaming or dropping documented keys of `quill stat`, `check` and `consumers -json` (scope.md:191-204) passes the
suite. Only stat's `holes`, check's `status` and the read/cat/tail record form were pinned.

**Grounds.** Mutations, `go test ./...` no failure (verified): stat `marker_age_seconds` renamed `marker_age`
(cmd/quill/read.go:92); stat drops `records`; check `fatal` renamed `Fatal` (read.go:210); check `intact` renamed
`intact_to`; consumers `lag` renamed `behind` (consumers.go:25); consumers `ahead` renamed `is_ahead` (consumers.go:29).
Caught: the status value, the `read -json` key and order, stat holes. The output is built from `map[string]any` literals,
so each key is a string no test read.

**Warrant.** scope.md says every documented key is always present and snake_case. A rename ships silently and breaks every
consumer of the JSON; v0.1.0 to HEAD already changed `stat -json` once (A1-57).

**Backing.** scope.md "Readable output: -json"; docs/userguide.md:227; decision 35.

**Rebuttal.** Key order is explicitly not part of the contract (scope.md:193), so a byte golden would over-pin; assert key
sets. The current output matches scope.md.

**Verdict.** CONFIRMED, 2 of 2 votes, both reproduced (three silent renames, and a built binary whose output contradicted
scope.md while `go test ./cmd/quill/` passed).

**Disposition.** Fixed in 7dea97a (test only, cmd/quill/json_shape_test.go, 145 lines); fix-skeptic FIXED-VERIFIED, 7 of 7.
Three tests decode each verb's `-json` into `map[string]json.RawMessage` and assert the exact key set and JSON type of
every documented key, including a healthy check (`problems` is `[]`), a check problem with all six keys, plain, ahead and
unreadable consumers. Key order is not compared. The fix-skeptic ran ten mutations; all keys and types were caught. Residual
(skeptic, outside the claim): the value of `unbounded` is pinned by type only; the check test covers the `consumer` problem
kind and not segment, hole, policy or clock.

### A6-4: the segment id has no ceiling, so id 10000000000 is invisible to every scan (low, compat)

**Claim.** The segment id is written `%010d` and parsed as exactly ten digits, but nothing stopped the writer creating id
10000000000: that file is invisible to every scan, the newest-segment check then refuses the log (ErrCorrupt), and records
acknowledged into it are unreachable by writable open.

**Grounds.** flush.go:153 `id := l.maxSeg + 1` had no ceiling; state.go:36 `segmentName`; state.go:39-41
(`len(digits) != 10`). Reproduction (verified by the auditor, and by a skeptic on macOS): a log renamed to
9999999999.log with `quill.tail` seg and maxseg 9999999999; Open, Append, Roll, Append, Close all succeeded; the directory
then held 10000000000.log and 9999999999.log and `quill.tail` named seg 10000000000; the next Open failed with `quill:
corrupt log: quill.tail names segment 10000000000, the newest on disk is 9999999999`, while OpenReader succeeded (head 3).
A second, easier trigger (skeptic): only `maxseg` in `quill.tail` changed to 99999999999 on a normal log; Open and one
Roll succeeded, the next Open failed.

**Warrant.** The format fixes the name width, and a name that does not fit it is treated as a foreign file, so a writer
past that id produces acknowledged data nothing can find. The sequence ceiling (2^64-2) is already enforced; the id
ceiling was an unstated gap of the same kind.

**Backing.** docs/format.md (`{id:010d}.log`; a file named like a segment that fails validation is a load error);
scope.md guarantee 8 (ids never reused); decision 62.

**Rebuttal.** About 10^10 rotations are needed (months of continuous rotation at one per millisecond with SegmentBytes 1);
the reproductions seeded the id or edited the marker by hand. No real log is expected to reach it.

**Verdict.** CONFIRMED, 2 of 2 votes, one reproduced (the other read flush.go, state.go and load.go).

**Disposition.** Ledger: held for the owner (semantic change: the writer would stop with an error at a documented limit),
no fix written, proposed fix to refuse a rotation whose next id would exceed 9999999999 and state the ceiling in
format.md. **Closed in the tree by commit bb646c0 (A3-8, the A3 pass's fix; FIXED-VERIFIED by that pass's fix-skeptic,
reported from threat-model.md).** Verified by the recorder from `git show bb646c0`: `decodeTailFast` and `decodeTailSlow`
reject `maxseg` above `maxSegmentID` (9,999,999,999, state.go), and so `seg`; `rotateLocked` returns "segment ids
exhausted" when `l.maxSeg >= maxSegmentID` (flush.go); docs/format.md states both; CHANGELOG has a Fixed line;
`TestRotationRefusesAnIDTheSegmentNamesCannotHold` (state_test.go) seeds the top ids. The recorder ran it at this commit:
`ok` (verified, `go test -count=1 -run 'TestRotationRefusesAnID|Maxseg|MaxSeg' .`). Not re-run: the A6-4 reproduction.
The skeptic's second trigger (a forged `maxseg` of 99999999999) is now refused at marker decode (inferred from the
decoder lines in bb646c0, not run). The owner is asked only to confirm and close, see stage2-owner-decisions.md.

### A6-5: the format marker protects only processes that open the log after a version bump (low, compat)

**Claim.** A running reader, consumer or writer never re-reads `quill.format`, so "a binary refuses a newer marker"
(format.md header, scope.md Versioning, decision 48) is not a guard for the normal mixed-version state, and no document
said how a migration must treat running processes.

**Grounds.** `loadFormat` is called at writer open (load.go:285), OpenReader (reader.go:98 audited, 102 now) and Archive's
destination (archive.go:135 audited, 137 now). Reproduction (verified by the auditor and a skeptic): with a Log and a Reader
open, `quill.format` rewritten to `quill/2 <id>`; the running reader continued (head 2, read 2 records, no error) and
the writer closed cleanly; only a fresh OpenReader and a fresh Open returned ErrFutureVersion.

**Warrant.** A future format 2 that changes a frame cannot rely on ErrFutureVersion to stop v1 processes already
running; a v1 reader would read v2 frames under v1 rules. The strict tail decoder and the CRC would usually make such a
change loud, but only for changes that happen to break them.

**Backing.** Decision 48; format.md header (migration note).

**Rebuttal.** Only format 1 exists; the first bump can document "stop all processes first". The strict decoders refuse
unknown fields. This is a documentation gap, not a defect in v1.

**Verdict.** CONFIRMED, 2 of 2 votes (one reproduced).

**Disposition.** Fixed in 70ed460 (documentation only; format.md, scope.md, decision 48, CHANGELOG); fix-skeptic
FIXED-VERIFIED, 7 of 7. The text says the version check happens when a handle is opened and again in `Consumer.Commit` and
in a Reader's wait, never in a running writer or in plain Read, Scan or Head, and that a migration note must require
stopping or restarting every writer, reader and consumer. The fix-skeptic checked each statement with a scratch test
(Reader Head, Read and Scan: no error; Reader ReadWait at head and Follow: ErrFutureVersion; Commit on a Reader-made and a
Log-made consumer: ErrFutureVersion; Log.ReadWait at head: no check). Residual (skeptic): format.md and scope.md say "the
wait inside ReadWait and Follow" re-checks; only the Reader's wait does (reader.go:202), the writer's `Log.wait` and a
Log-made consumer's wait do not. The error is on the safe side. Re-checking `quill.format` in running handles (the optional
half of the proposal) was not done: per-call I/O, not needed while only format 1 exists.

### A6-6: no fixture is a frame written by the reference implementation (low, compat)

**Claim.** Decision 3 ("the frame bytes are the reference's, unchanged") is proven only against quill's own encoder and
the specification text, not against a byte sequence from the reference implementation.

**Grounds.** docs/audits/test-ledger.md:5 says the frame goldens "were recorded from quill's encoder"; `recordedFrames`
and testdata/golden/segment.log were produced by `QUILL_RECORD_GOLDENS` (golden_test.go record path). The auditor decoded
the three recorded frames in an independent Python program (uvarint, IEEE CRC-32 big-endian) and they equal the golden
hex and the docs/format.md table, so the goldens are consistent with the spec text (verified).

**Warrant.** A golden pins quill against itself and the spec, which protects future versions. It cannot show that a log
written by the reference decodes: the reference's real bytes (varint widths for a negative `at`, the CRC polynomial, the
CRC byte order) appear in no test. Decision 3's justification is migration compatibility.

**Backing.** Decision 3; scope.md "the frame unchanged from the reference".

**Rebuttal.** The reference was not available to the auditor; the layout is standard (LEB128, CRC-32 IEEE) and the spec
was written from it; one reference-written log would settle it. It matters only if reference logs are to be read by quill.

**Verdict.** PLAUSIBLE: one vote CONFIRMED, one PLAUSIBLE. Neither voter ran anything, because the finding is about
missing evidence; the premise (no reference-written fixture) was verified by reading and by searching the tree.

**Disposition.** Fixed in ee11f57 (documentation only; docs/format.md, CHANGELOG), taking the finding's second option;
fix-skeptic FIXED-VERIFIED, 7 of 7. docs/format.md now says, under the recorded-frames table, that the goldens were
recorded from quill's own encoder and that no fixture is a reference-written frame, so compatibility with the reference
rests on the specification. The fix-skeptic decoded the three frames again independently and confirmed the new sentences.
Not delivered: a real reference-written fixture (none was available; fabricating one would repeat the gap). It would
close this item properly and is the one piece of evidence the pass could not get.

### A6-7: the golden tests are weaker than they look in four places (low, compat)

**Claim.** (1) The policy golden cannot tell fields apart; (2) it contains a value no writer can produce; (3) decode-direction
failures omit the version and re-record instruction; (4) the segment-image failure labels are inverted.

**Grounds.** (1) testdata/golden/quill.policy had max_bytes, min_free_bytes, max_records 0 and retention equal to
segment_max_age (24h0m0s), and the decode check read only four fields. (2) It listed `protected_consumers`
`["_archive","billing"]`, but `Open(Config{ProtectedConsumers: ["_archive"]})` returns `invalid consumer name: "_archive": a
leading underscore is reserved`, and the writer's real file prints one array element per line while format.md printed
`["billing", "reports"]` on one line. (3) The frame, tail, format, consumer and policy "DECODING CHANGED" failures said
only that decoding changed; only `failGolden` named the new-version rule and forbade re-recording. (4) golden_test.go:124
passed the checked-in file as "got" and the pasted frames as "want", so the message printed the file as "now" and said
ENCODING CHANGED when two recorded artifacts merely disagreed.

**Warrant.** A fixture with equal or zero values detects a reorder but not a swapped meaning of two fields; an
unproducible value pins a file shape that never occurs; a message without the remedy on the dangerous direction leaves the
person free to edit the expected values.

**Backing.** golden_test.go header ("A failure means the format changed. Do not re-record"); format.md ("Normative ...
frozen by golden tests").

**Rebuttal.** All 19 encoder-side and decoder-side golden mutations the auditor ran failed; these are quality gaps, not missed
regressions. A decoding failure cannot be fixed by re-recording, so the omission is less dangerous than it looks.

**Verdict.** CONFIRMED, 2 of 2 votes, one reproduced (`Open` rejecting `_archive`, the one-per-line policy bytes, a name
swap in `state.go` that passed the golden tests, and the mislabelled message).

**Disposition.** Fixed in 49bc53c (golden_test.go, testdata/golden/quill.policy, docs/format.md, CHANGELOG);
fix-skeptic FIXED-VERIFIED, 7 of 7. The policy golden now has nine distinct non-zero values and a producible
`protected_consumers` list (`billing`, `audit`), is compared whole with `reflect.DeepEqual`, and is asserted equal to the
pasted literal the writer test pins; a helper `failGoldenDecode` gives every decode-direction failure the new-version,
do-not-re-record text; the segment-image check says the file and `recordedFrames` DISAGREE; the format.md policy example is
byte-identical to the golden. The fix-skeptic applied the claim's fault (a `decodePolicy` that swaps MaxBytes with
MinFreeBytes and Retention with SegmentMaxAge): the old tree's golden tests passed it, the new tree failed with the new
text. Note for the owner: this is the one golden file this pass changed. It was replaced because the old content was not a
file any writer produces; the frame, tail, format, consumer and segment goldens are byte-for-byte unchanged (verified by the
fix-skeptic with `git diff` over testdata), and the old policy still decodes. Residual (skeptic): the new assertion in
TestStateFileGoldens ignores the `os.ReadFile` error; a missing file still fails the comparison.

### A6-8: two format-relevant constants differ only by letter case (low, hygiene)

**Claim.** `MaxSegmentBytes` (1 GiB, the SegmentBytes configuration ceiling, log.go:93 audited, 97 now) and `maxSegmentBytes`
(1.5 GiB, the segment file limit and the bound on `quill.tail` `off`, load.go:20 audited, 19 now) differ only by the case of
the first letter and mean different things.

**Grounds.** A case-insensitive scan of all non-test declarations (verified in a scratch copy) found exactly these, plus
the benign `maxPayloadBytes`/`MaxPayloadBytes` alias and the `Archive`/`archive` wrapper. state.go (`decodeTail`) and load.go
use the lowercase one; cmd/quill/write.go uses the uppercase one for a ceiling.

**Warrant.** A mistyped capital silently moves a documented on-disk limit by 0.5 GiB; the compiler accepts both and the
format.md limits table lists both numbers in adjacent rows.

**Backing.** docs/format.md limits table; the A6 lens (name collisions).

**Rebuttal.** Both are used correctly today, and mutating the lowercase value is caught by several tests. The exported name
is public API, so renaming it is a breaking change.

**Verdict.** CONFIRMED, 2 of 2 votes, neither reproduced (a naming issue). One voter noted the cmd/quill/write.go use is at
lines 25 and 27, not 26-35 as cited.

**Disposition.** Not changed. The fix (rename the unexported constant to `maxSegmentFileBytes`, keep the exported name) was
written and its commit did not land: the workflow handed the fix-skeptic the placeholder `PENDING` instead of a commit
sha, `git rev-parse --verify PENDING` failed, nothing was merged, and the skeptic returned "fix-failed". No skeptic
attacked the rename itself. Verified by the recorder in the tree now: both constants still exist (load.go:19, log.go:97),
and the five files that use the lowercase name are fuzz_test.go, load_test.go, load.go, log.go and state.go. A3-8 added
`maxSegmentID`, which does not collide with either. Recommended answer in stage2-owner-decisions.md.

### A6-9: every offline write verb rewrites `quill.tail` with mode 0600 (medium, bug)

**Claim.** `recover`, `roll`, `append` and `prune` rewrite `quill.tail` with mode 0600 in a log created with Config.Perm
0660, and `roll` and `append` create the new segment 0600. Because HEAD derives the mode of Reader- and CLI-created
consumer files and of every Archive file and directory from `quill.tail`'s permission bits, one routine `quill roll` or
`quill recover` silently turns the group arrangement off. The same happens in v0.1.0, so this is not a skew regression, but
fix A1-85 depends on a mode the CLI itself destroyed.

**Grounds.** cmd/quill/write.go:20-26 (`openWriter` built the Config for every offline verb and never set `cfg.Perm`);
log.go:137-138 (Perm 0 becomes DefaultPerm 0600); flush.go:117 (the marker is written with `l.cfg.Perm`); reader.go
`OpenReader` takes `r.perm` from `Lstat(quill.tail)`; archive.go and `consumerStore` use `r.perm`. Reproduction (verified,
HEAD binary, macOS umask 022 and Linux umask 0002): a log written with `-perm 432` (0660), then for each verb
`chmod 660 quill.tail; quill <verb>`. Linux output: `recover`: tail 600 (was 660), newest segment 660; `roll`: tail 600,
newest segment 600; `append x`: tail 600; `prune -max-records 1000 -max-bytes 100000000`: tail 600. Afterwards `register -name
zz` made `consumers/zz` 600 in a 770 directory, and `archive -to` made a 700 directory with a 600 `quill.format`. The v0.1.0
binary left the tail 600 after `recover` and `roll` too (macOS).

**Warrant.** docs/userguide.md:164 and scope.md:341-344 say the CLI and Archive create files with the mode of the log's
`quill.tail`. If the CLI's own verbs rewrite the tail at 0600, the premise of that rule no longer holds after the first
such verb. A 0600 marker owned by the operator cannot be read by group readers (inferred from POSIX semantics; no second
account was used).

**Backing.** Decisions 74 and 51; scope.md "Perm" row (docs/design/scope.md:123); CHANGELOG Unreleased (A1-85);
docs/userguide.md:164.

**Rebuttal.** Harmless if the operator is the writer's own account and nobody else needs the files, or if the deployment
reopens the log with Perm 0660 afterwards (the tail is restored; a segment created 0600 by `roll` is not). The docs never
literally promise that offline verbs preserve the mode; the offline verbs open with the default Config by design
(decisions 68 and 99). Segment files go through the umask in any case. Cross-account access loss was not demonstrated.

**Verdict.** CONFIRMED, 2 of 2 votes, both reproduced (macOS; `prune` only with its full flags, the first try having
failed on usage).

**Disposition.** Fixed in 709c5c5 (cmd/quill/write.go, cmd/quill/offline_perm_test.go, docs/userguide.md, CHANGELOG);
fix-skeptic FIXED-VERIFIED, 7 of 7. `openWriter` now sets `cfg.Perm` from the mode of an existing regular-file
`quill.tail` (OR 0600) when the caller gave none, so the four verbs keep a 0660 log at 0660; a missing or non-regular tail,
or a mode below 0600, gives 0600 as before; a wider mode is carried, never widened. The test failed on the parent's
write.go ("quill.tail is 600, want 660" for each verb, and the new segment 600) and passes now. The fix-skeptic also built
the CLI from before and after on the Linux box with a Perm 0660 log (umask 002): before, tail and segment 600, consumer
file 600, archive 700 and 600; after, 660 and 660, consumer file 660, archive 770 and 660. Not changed: the library (Open
with an explicit Perm is the documented contract, decision 74). Residual (skeptic): the decisions log is not amended to
say the CLI carries the tail's mode, nothing there contradicts it; access from a second account was not tested; staticcheck
and deadcode were not run by the fix-skeptic for this commit.

## What the recorder checked

Verified by the recorder in this worktree: the seven fix commits and bb646c0 are ancestors (`git log`), and the file lists
of the seven (above) match the fix summaries; `hack/` lists no skew harness; `maxSegmentBytes` and `MaxSegmentBytes` both
still exist; the diff of bb646c0; `go test -count=1 -run 'TestRotationRefusesAnID|Maxseg|MaxSeg' .` printed `ok`. Not
re-run by the recorder: every reproduction, every mutation run, every skew experiment and the gate. Every figure above is
reported from the ledger of the auditors and skeptics unless it is marked as the recorder's.
