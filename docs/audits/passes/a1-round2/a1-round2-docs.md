# A1 adversarial audit, round 2: docs

Date 2026-10-29, against HEAD 1da37fc. Dimension report for the two docs lenses (user docs and
design docs): 15 items, ranked by severity and then by reach. The design-docs lens also checked the
guarantees by mutation, so the two test-strength items (A1-94, A1-95) and the spec-against-code
divergence A1-2 are filed here. The top line and the coverage statement are in
[a1-round2-findings.md](a1-round2-findings.md). Held items are repeated with a recommended answer in
[stage2-owner-decisions.md](../../decisions/stage2-owner-decisions.md).

Verdicts: CONFIRMED, PLAUSIBLE, REFUTED. Votes: C confirmed, P plausible, R refuted. Reproductions
ran in throwaway copies under `scratch/<label>/`. Anchors are to the audited tree 1da37fc. Claims
not self-evident are labelled verified, reported or inferred.

Tally: 11 fixed, 4 held.

| # | id | anchor | sev | verdict | votes | disposition |
|---|----|--------|-----|---------|-------|-------------|
| 1 | A1-66 | docs/userguide.md:243-271 | high | CONFIRMED | CC | fixed 3439946 |
| 2 | A1-67 | consumer.go:120-122 | high | CONFIRMED | CC | held |
| 3 | A1-68 | build/quill:25 | medium | CONFIRMED | CC | fixed df7af4f |
| 4 | A1-69 | append.go:30-33 | medium | CONFIRMED | CC | held |
| 5 | A1-2 | strictjson.go:30-49 | low | CONFIRMED | CC | held |
| 6 | A1-87 | retention.go:139 | low | CONFIRMED | CC | held |
| 7 | A1-97 | docs/format.md:179 and others | low | CONFIRMED | CC | fixed f1df4ae |
| 8 | A1-95 | retention.go:238, 126; flush.go:130 | low | CONFIRMED | CC | fixed 9810a56 |
| 9 | A1-94 | retention_test.go:289-330 | low | CONFIRMED | CC | fixed 47a5ebb |
| 10 | A1-79 | docs/reference.md:5 and others | low | CONFIRMED | CC | fixed a593c45 |
| 11 | A1-74 | hack/gate.sh:48-59 | low | CONFIRMED | CC | fixed e498890 |
| 12 | A1-73 | SECURITY.md:14 | low | CONFIRMED | CC | fixed 082bbcd |
| 13 | A1-72 | docs/userguide.md:181-182 | low | CONFIRMED | CC | fixed da4d1ce |
| 14 | A1-71 | log.go:38-39 | low | CONFIRMED | CC | fixed 88a44b9 |
| 15 | A1-70 | examples/README.md:3-4 | low | CONFIRMED | CC | fixed 4aec960 |

## 1. A1-66: the manual corruption procedure in the user guide fails at `quill recover` (high, CONFIRMED, fixed)

**Claim.** The manual corruption procedure in docs/userguide.md section 10 (lines 243-271) does not
work for the case it targets. After step 4 (truncate the newest segment at the end of the last valid
frame), step 5 (`quill recover`) fails with ErrCorrupt because quill.tail still names the old,
longer durable offset. The steps never say to rewrite quill.tail first.

**Grounds.** load.go:230-232 (`recoverSegment` returns a SegmentError when durable exceeds the file
size) and load.go:395 (a marker sequence above the last valid frame is ErrCorrupt).

**Warrant.** Steps 1 to 6 are presented as the complete recovery. Step 4 shrinks the file below
quill.tail's `off`, and Open (which `quill recover` uses) refuses any file shorter than the marker's
offset, so step 5 cannot succeed; the only route is a hand edit of quill.tail that the procedure
does not list.

**Backing.** Decision 64 (v0.1.0 ships with docs/userguide.md documenting the manual procedure, so a
failing procedure leaves the one promised remedy for ErrCorrupt broken); decision 39 (the marker is a
lower bound; damage below it is corruption and never repaired in place).

**Rebuttal.** The last paragraph of section 10 describes rebuilding the marker, but for a damaged
quill.tail, not as a step of this procedure. The procedure works unchanged when the damage lies only
above the marker's `off` (a torn tail).

**Verdict.** CONFIRMED by 2 of 2 votes. Round-1 neighbour: docs-35 (rewrote the text before the
steps, not the steps).

**Reproduction.** Verified on macOS (both skeptics) and Linux (auditor). Program: 10 records via
`quill append -dir lg`, one byte flipped mid-segment (file 271 bytes, quill.tail off 271); `quill
check` prints `segment lg/0000000000.log damaged at byte 135, intact to byte 135: torn frame` and
exits 3; `truncate -s 135`; `quill recover -dir lg` prints `the file holds 135 bytes but the marker
says 271 are durable` and exits 1, quill.tail unchanged. After writing quill.tail as
`{"seq":5,"seg":0,"off":135,"maxseg":0}`, `recover` succeeds (`recovered: head 5 ...`) and `check`
prints HEALTHY. A skeptic notes that after the truncation `quill cat` refuses the log and prints
nothing, so the last valid sequence must be read before truncating.

**Disposition.** Fixed in commit 34399462ac7a. Section 10 now says to note the last valid sequence
(`quill cat`) before truncating and has a new step 5 that rewrites quill.tail (`seq`, `off`; keep
`seg` and `maxseg`) before `quill recover`; steps renumbered to 7; decision 64 amended in place;
CHANGELOG line; a new test pins the sequence (truncate below the marker refuses Open with
ErrCorrupt; rewriting the marker makes Open succeed and the next append gets the following
sequence). Fix-skeptic FIXED-VERIFIED: the steps run literally on a build of the fixed tree gave
`recovered: head 5`, `check` HEALTHY and the next append seq 6. Residual: step 5's "0 if no frame is
left" is imprecise when older segments exist (the true value is the previous segment's last
sequence); both 0 and 3 recover to head 3 in the sibling case, so the procedure still works.

## 2. A1-67 (with A1-84): the documented rewind of an ahead consumer, `quill commit`, cannot work (high, CONFIRMED, held)

**Claim.** The documented way to rewind a consumer that is ahead of a restored log, `quill commit`,
cannot work. A consumer whose offset is above Head+1 fails to open with ErrAhead, and `quill commit`
opens the consumer before committing, so commit fails with the same error. docs/userguide.md:98 and
:222-223, scope.md:325 and decision 53 name `quill commit` or an explicit rewind as the remedy.

**Grounds.** consumer.go:120-122 (`init` returns ErrAhead when `state.Next > head+1`);
cmd/quill/consumers.go:59 (`c.consumer` calls `r.Consumer`) and :86-98 (`cmdCommit` goes through
it). The library has the same limit (`Log.Consumer` and `Reader.Consumer` both run `init`).
`quill consumers` shows the ahead consumer as offset 8, lag 0, with no note, and `quill check` prints
HEALTHY (stat.go:194-198 flags only consumers that fail to parse or belong to another log).

**Warrant.** The only operation that could rewind (Commit) is unreachable for exactly that state,
because acquiring the handle refuses first, so the documented remedy fails.

**Backing.** scope.md:325 and decision 53 (a consumer ahead of the restored head fails with ErrAhead
and must be rewound deliberately); decisions 36, 9; userguide section 9 (Backup).

**Rebuttal.** `quill unregister` then `quill register -protect` then `quill commit -offset N` does
reach any offset, so a CLI path exists; the scope text says only "rewound deliberately". But the
userguide names the verb literally, and that sequence resets protection to the flag given and is
documented nowhere. A skeptic rates the severity medium because a workaround exists.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS (both skeptics) and Linux (auditor). Program: a 5-record log
copied to `o_old`; 3 more appended; `register -name billing -protect`; `commit -name billing
-offset 8` (prints `billing 8`); restore `o_old`'s quill.tail and segment over the log; `quill -dir
o commit -name billing -offset 2` prints `consumer offset is ahead of the log: consumer "billing" is
at 8, the head is 5`, exit 1, the file unchanged. Only unregister, register, commit worked. A
skeptic confirms `commit` fails with ErrAhead for any consumer name, not only this one.

**Disposition.** Held for the owner. Recommended: apply the doc fix now and approve the code
change. See the owner file. No fix written.

## 3. A1-68: `build/quill --dev` exits 1 silently on a clean tree (medium, CONFIRMED, fixed)

**Claim.** `build/quill --dev` exits 1 with no output on a clean working tree, so the dev build
documented in the script's header and run by the CI `examples` job never produces `build/install`;
it works only when the tree is dirty.

**Grounds.** build/quill:8 sets `set -euo pipefail`; line 25 is `version="dev-$(git rev-parse --short
HEAD)$( [ -n "$(git status --porcelain)" ] && echo -dirty )"`. On a clean tree the last command
substitution is `[ -n "" ] && echo`, which returns 1; that becomes the status of the assignment and
errexit ends the script. ci.yml's `examples` job runs `build/quill --dev && test -s
build/install/SHA256SUMS` on a fresh checkout.

**Warrant.** A simple assignment returns the status of its last command substitution, and `set -e`
applies; a clean checkout is the normal CI state, so the documented command fails where it is used.

**Backing.** README.md:75 (the release build); the script header ("`--dev` does a development build
with no refusal"); ci.yml.

**Rebuttal.** The release path (a tagged clean tree) does not use line 25. CI history was not
checked, so whether CI has been red is unknown.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS (bash 3.2.57) and Linux (bash 5.2.37). Program: a clean `git
init`, `add`, `commit` of `git archive HEAD`; `bash build/quill --dev; echo exit=$?` prints `exit=1`
with no output and no `build/install`. After `touch untracked.txt` it exits 0 and produces four
tarballs, SHA256SUMS and a `.cdx.json`.

**Disposition.** Fixed in commit df7af4fdcf16. The suffix is set with `dirty=""; [ -z "$(git status
--porcelain)" ] || dirty=-dirty` before the version assignment, so no substitution returns 1 under
errexit; `TestBuildScriptDevVersion` runs the real script in a throwaway git module, clean and
dirty. Fix-skeptic FIXED-VERIFIED: the clean subtest fails with the old script; the unmodified fixed
script gave exit 0 with 4 archives, an SBOM and SHA256SUMS on macOS and Linux, and the tagged release
form is unaffected. Residual: staticcheck and deadcode were not run (irrelevant to a shell change).

## 4. A1-69 (with A1-28, A1-89): MaxBatchBytes is documented as the largest encoded batch, but is checked against a worst-case estimate (medium, CONFIRMED, held)

**Claim.** The limit is checked against 34 bytes of framing per record plus the payload, so a batch
of many small records is refused although its real encoded size is under half the limit. `quill
append` reading a large stdin hits the same limit and refuses everything.

**Grounds.** append.go:30 adds `3*maxVarint + len(p) + 4` per record; append.go:32-33 refuses batches
of more than one record over MaxBatchBytes on that sum. Docs: userguide.md:144 ("Largest encoded
batch"), :278, log.go:47, append.go:17-19, decision 46. cmd/quill/write.go:104 sends all stdin lines
as one `AppendBatch`.

**Warrant.** The documented limit is on encoded bytes; the enforced one is 34 bytes per record plus
payload, more than twice the real size for 1-byte payloads. The disk-bound formula stays safe because
the estimate over-counts.

**Backing.** Decision 46 and scope.md:119 ("encoded").

**Rebuttal.** The error says "about", and the estimate is deliberately conservative because the
sequence number is unknown at validation (the comment at append.go:29); payloads over a few hundred
bytes differ by under 10 percent. Changing to the exact size changes behaviour.

**Verdict.** CONFIRMED by 2 of 2 votes (one skeptic rates it low to medium).

**Reproduction.** Verified on macOS (both skeptics) and Linux (auditor). Program: 100,000 and
400,000 one-byte records are accepted (400,000 occupy 5,983,490 bytes on disk, about 15 bytes per
record); `python3 -c "print('x\n'*500000, end='')" | quill append -dir big` is refused with `payload
too large: batch of about 17500000 bytes, MaxBatchBytes is 16777216`, exit 1, nothing appended.
Extrapolated real size of the refused batch: about 7.5 MB.

**Disposition.** Held for the owner. Recommended: document the estimate now; chunking stdin is
optional. See the owner file.

## 5. A1-2 (with A1-36, A1-55, A1-90): the slow path for quill.tail is not as strict as format.md says (low, CONFIRMED, held)

**Claim.** The strict slow path accepts a key whose spelling is not a documented field name, and a
case-variant duplicate key silently wins; it also accepts trailing whitespace and a missing trailing
newline.

**Grounds.** strictjson.Unmarshal (strictjson.go:30-49) runs `checkDuplicateKeys` with a
case-sensitive map (strictjson.go:123) and then decodes through `encoding/json`, which matches field
names case-insensitively. decodeTailSlow is at state.go:138-158.

**Warrant.** format.md:103-105 says every state file but quill.policy is refused for "unknown fields
... a duplicate key", and :109-112 describes the file as "fixed field order, trailing newline".

**Backing.** docs/format.md:103-112; guarantee 9 (strictjson.go:6-7: "unknown shapes are refused, not
guessed"); decisions 8, 48.

**Rebuttal.** No writer emits these inputs; "duplicate key" can be read as byte-identical keys, which
the code does refuse. One skeptic holds that whitespace, reordering and a missing newline are not
listed among the refusals, that decision 105 sends non-canonical input to the strict path by design
("exactly what they were"), and that only the case-variant half, above all the `seq` plus `SEQ`
duplicate, is a real divergence.

**Verdict.** CONFIRMED by 2 of 2 votes, the second narrowing it to the case-variant duplicate.
Round-1 neighbours: decision 72 (exact-string duplicate refusal only); decision 105 kept the slow
path unchanged.

**Reproduction.** Verified on macOS (both skeptics; first in the auditor's probe). Program: call
`decodeTail` on five inputs. `{"SEQ":1,"seg":0,"off":0,"maxseg":0}\n` returns Seq 1 and nil error;
`{"seq":1,"SEQ":2,"seg":0,"off":0,"maxseg":0}\n` returns Seq 2 and nil; trailing whitespace and no
final newline both return nil; reordered keys return nil; only a byte-identical repeated key is
refused (`duplicate key`).

**Disposition.** Held for the owner. Recommended: make the duplicate check case-insensitive and
document the tolerated shapes. See the owner file.

## 6. A1-87 (with A1-40): the clock guard deletes ceil(25%) of the sealed segments, not "at most 25%" (low, CONFIRMED, held)

**Claim.** The clock guard deletes the ceiling of 25 percent of the sealed segments per pass, where
decision 44 and scope.md say "at most 25% (at least one)"; a test pins the larger number.

**Grounds.** retention.go:139 `limit := (len(l.sealed) + 3) / 4`. With 5 sealed segments the limit is
2 (40 percent); with 15 it is 4 (26.7 percent). retention_test.go:266 expects 12 files left after the
second pass over 15 sealed segments (4 deleted); a `max(1, n/4)` mutant fails exactly that line.

**Warrant.** "At most 25 percent (at least one)" is floor with a minimum of one; the code rounds up.

**Backing.** Decision 44; scope.md "A wrong clock"; build-prompt.md retention bullet; T21.

**Rebuttal.** Rounding up is a defensible reading of "at least one" for tiny logs and the difference
is at most three segments; the owner accepted "25% per pass" without a rounding rule.

**Verdict.** CONFIRMED by 2 of 2 votes (hand arithmetic; the mutant was not run by the skeptics).

**Disposition.** Held for the owner. Recommended: reword the text to "a quarter, rounded up, at least
one".

## 7. A1-97 (with A1-39): six design-record statements disagree with the code or each other (low, CONFIRMED, fixed)

**Claim.** format.md's policy example cannot be produced; the disk-bound formula and the frame
overhead are stated inconsistently; format.md's creation row and decision 86 are stale; scope.md's
statfs note is out of date.

**Grounds.** (1) docs/format.md:179 shows `protected_consumers ["_archive", "billing"]`, but
`validate()` (log.go:163-166, `validName`) rejects `_archive` there with ErrConfig (verified by
scratch test). (2) The bound is `MaxBytes + SegmentBytes + MaxBatchBytes` in decision 46,
`... + MaxBufferBytes + MaxBatchBytes` in build-prompt T13, and the `max(...)` form in scope
guarantee 11 and decision 69. (3) Worst-case frame overhead is 32 bytes (verified: `frameLen -
payload = 32` at the ceiling), but scope.md says three varints of 10 plus CRC = 34 and omits the
length prefix; decision 69 says 24 and format.md says 30. (4) format.md's Creation row omits the
directory fsync right after quill.format that decision 81 and the create path perform. (5) Decision
86 says `read` and `cat` are "the same printer with different defaults"; both use `readFlags` and
`cmdRead` with identical defaults. (6) scope.md says statfs was "not run on Linux"; `freeBytes` equals
`df -B1 --output=avail` on the ext4 box (156930945024, reported by the auditor).

**Warrant, backing.** Each is a statement a reader could rely on that the tree contradicts or states
differently elsewhere; format.md is declared normative.

**Rebuttal.** None changes behaviour; items 2 and 3 are conservative bounds, so no guarantee is
weakened.

**Verdict.** CONFIRMED by 2 of 2 votes (not run; read against the lines cited). Round-1 neighbours:
docs-39 and docs-42.

**Disposition.** Fixed in commit f1df4aed1036. `_archive` removed from the format.md example;
decision 46 marked superseded by 69 and build-prompt T13 aligned to the `max(...)` form; frame
overhead stated once (32 worst case: 4 prefix + 10 + 10 + 4 varints + 4 CRC, with the code's 34 as a
conservative bound) in format.md, the codec.go comment, decision 69 and scope.md; the post-format
directory fsync added to the Creation row; decision 86 reworded; the statfs sentence updated;
CHANGELOG line. Fix-skeptic FIXED-VERIFIED: the 32-byte worst case re-derived by hand from codec.go
(4 + 10 + 10 + 4 + 4 minus payload); the Creation row matches log.go's create path. Residual:
decision 69 and format.md:58 say recovery "enforces" the 64 MiB frame limit, but recovery enforces the
declared body length; the statfs `df` comparison has no repeatable artifact in the tree.

## 8. A1-95: three documented mechanisms survive removal in mutation runs (low, CONFIRMED, fixed)

**Claim.** The directory fsync before unlinking a segment (decision 65), protection by a broken
consumer file when no protected consumer exists (decision 41), and the `pendingPublish` retry after a
failed rotation (decisions 60 d and 78) can each be deleted with the whole suite green.

**Grounds.** Scratch mutants: removing the SyncDir at retention.go:238; `cv.broken ||
cv.hasProtected` to `cv.hasProtected` at retention.go:126 (retention_test.go:209-217 always has a
caught-up protected consumer); `l.pendingPublish = rotErr != nil` to false at flush.go:130.

**Warrant.** A guarantee whose mechanism can be deleted with a green suite is not verified by it. For
decision 65 the fsync may be redundant after decision 89 (inferred); the mutant still changes
observable behaviour (error path and operation order).

**Backing.** decisions 41, 60, 65, 78, 89; format.md Durability table (Deletion row); build-prompt
T1 and T9.

**Rebuttal.** Decision 65's fsync may be redundant defence after decision 89, in which case it is a
stale rationale, not a missing test.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Reproduction.** Verified on macOS by both skeptics. Each mutant applied to a copy of HEAD left
`go test ./...` green; a short Sim test per mutant (a broken consumer file only; ENOSPC on segment
create then a second Flush; EIO on SyncDir during Prune) passes on the unmodified tree and fails on
its mutant.

**Disposition.** Fixed in commit 9810a56681658. Three Sim tests in retention_test.go, one per mutant,
each passing on HEAD and failing on its mutant; no product code changed. The decision 65 test
asserts that a directory sync immediately precedes each segment unlink rather than a crash-subset
outcome. Fix-skeptic FIXED-VERIFIED: each mutant, and a variant that syncs once before the loop,
fails its test; the tests pass 30 times under -race and on the Linux box. Residual: the SyncDir-error
branch is not covered.

## 9. A1-94: T13's disk-bound assertion cannot fail at the sizes it runs (low, CONFIRMED, fixed)

**Claim.** T13's bound includes `MaxBufferBytes` (16 times FlushBytes = 16 MiB) while the test
writes a few KB, so a cap 4x too lax passes it.

**Grounds.** retention_test.go:289-330: FlushBytes 1<<20, bound = 1000 + 200 + MaxBufferBytes + 400,
about 16.8 MB, traffic under 10 KB. The mutant `bytes > 4*MaxBytes` (retention.go:184) keeps
`TestDiskBoundsMaxBytesHolds...` green; other tests catch it. No test sets MaxPayloadBytes above
MaxBatchBytes, so decision 69's single-record term is never exercised.

**Warrant.** Guarantee 11 is a numeric bound, and the test meant to falsify it asserts one four
orders of magnitude above the traffic.

**Backing.** scope.md guarantee 11; build-prompt T13; decision 69.

**Rebuttal.** The "at any sample" bound includes a term that applies only while flushes fail, so a
tight bound needs a different test; the Dropped assertion does exercise cap enforcement (the 100x
and "no cap at seal" mutants are caught by it).

**Verdict.** CONFIRMED by 2 of 2 votes. The test's own formula also omits the amended
`MaxPayloadBytes + 34` term.

**Reproduction.** Verified on macOS by both skeptics: with the mutant applied, T13 still passes and
exactly two other tests fail (`TestADamagedSealedSegmentCountsItsBytesInStatAndInMaxBytes` and
`TestAProtectedConsumerKeepsItsUnreadSegmentsAndACapOverridesItCountingDropped`).

**Disposition.** Fixed in commit 47a5ebb84a27. T13 now uses FlushBytes 100, MaxBufferBytes 300,
MaxBatchBytes 400 and MaxPayloadBytes 800 (bound 2334 against an observed peak of 1175 instead of
about 16.8 MB), appends a record of MaxPayloadBytes (above MaxBatchBytes) every 50th step, and uses
`max(MaxBatchBytes, MaxPayloadBytes+34)` as decision 69 states. Fix-skeptic FIXED-VERIFIED: the 4x
and 100x mutants now fail the test. Residual: a 2x-lax mutant still passes (the bound is about twice
the peak), and the decision 69 term is exercised by traffic but not load-bearing for the assertion.

## 10. A1-79: several small user-facing inaccuracies (low, CONFIRMED, fixed)

**Claim.** (a) docs/reference.md:5 (generated from cmd/quill/verbs.go:237) says every verb but help,
version and ls takes `-dir`; (b) README.md says docs/audits holds five kinds of file but holds
eleven; (c) the README's load.go row lists `hole detection`, which lives in holes.go, and holes.go is
missing from the layout table; (d) README Status says "Private, pre-release, under construction";
(e) Reader.ReadWait, Reader.Scan, Reader.ScanAll, Consumer.ReadWait, Consumer.Scan, ProtectOn,
ProtectOff, GapError.Error, SegmentError.Error and Problem.String lack godoc comments; (f)
errors.go says ErrPayloadTooLarge covers payloads and batches over the limit, but `AppendBatch(nil)`
also returns it as "empty batch".

**Grounds.** (a) the `-dir` row for `quill ls` in the same file; (c) load.go:11-12 and holes.go:21;
(e) a go/ast scan of exported identifiers; (f) append.go:21-23.

**Warrant, backing.** Each is a statement or omission that does not match the tree; reference.md is
generated and guarded by a test (cmd/quill/cli_test.go:60-68), so (a) is fixed in verbs.go.

**Rebuttal.** All are cosmetic; (d) is expected release state, not drift.

**Verdict.** CONFIRMED by 2 of 2 votes, narrowed: (a) is followed by "(`ls` takes the parent)" at
reference.md:6, so awkward but not false; (b) misquotes the README (it lists five things without
claiming to be exhaustive); (d) is not drift; in (e) Reader.ReadWait, Scan and ScanAll are covered by
a group comment at reader.go:155 and Consumer.ReadWait and Scan by one at consumer.go:210 (a linter
would still flag them); (c) and (f) are true. Round-1 neighbours: docs-46 and docs-45.

**Disposition.** Fixed in commit a593c455d59d. The `-dir` sentence reworded in verbs.go and
reference.md regenerated; holes.go added to the README layout table; the README Audits bullet
widened; ErrPayloadTooLarge now says it also covers an empty batch; doc comments added to the
exported identifiers that lacked a specific one; (d) left alone. Fix-skeptic FIXED-VERIFIED: the
reference-currency test fails with the old verbs.go; a go/ast scan went from 10 undocumented
exported identifiers to none; `quill reference` is byte-identical to reference.md.

## 11. A1-74: `hack/gate.sh` prints `OK: gate green` after FUZZTIME=0 skipped the fuzz step (low, CONFIRMED, fixed)

**Claim.** The gate exits 0 and prints `OK: gate green` even when FUZZTIME=0 skipped fuzz, while
CONTRIBUTING.md:34-35 says the skip is "not a green gate"; the README gate summary (README.md:70 and
:88) omits the unsupported-platform and no-require steps that CONTRIBUTING lists.

**Grounds.** hack/gate.sh:48-49 prints the red skip line; the last line is an unconditional `echo
"OK: gate green"`. Read, not run: the full gate needs staticcheck and deadcode from the network.

**Warrant.** A caller or hook reading the final line or the exit status sees success for a gate the
docs call not green.

**Backing.** CONTRIBUTING.md:34-35 and the script's own comment at gate.sh:19.

**Rebuttal.** The red warning is printed; a non-zero exit would break `FUZZTIME=0 hack/gate.sh` as a
quick local run. A skeptic notes CONTRIBUTING lists the unsupported-platform step but not the
no-require step, so only the README is missing both.

**Verdict.** CONFIRMED by 2 of 2 votes (read, not run).

**Disposition.** Fixed in commit e4988908c288. The script ends `PARTIAL: fuzz skipped (FUZZTIME=0),
not a green gate` in red instead of `OK`; the exit status stays 0 as a deliberate choice, and the docs
say so; README and CONTRIBUTING list the same steps; CHANGELOG line. Fix-skeptic FIXED-VERIFIED: the
real script run with a `go` shim stubbing staticcheck, deadcode and the race test ended PARTIAL for
FUZZTIME=0 and `OK: gate green` for FUZZTIME=1s. Residual: a hook that reads only the exit status
still sees success.

## 12. A1-73: SECURITY.md says every state file is parsed strictly (low, CONFIRMED, fixed)

**Claim.** SECURITY.md:14 says "Every state file is parsed strictly; unknown shapes are refused, not
guessed", but quill.policy is parsed tolerantly: unknown fields are ignored and a parse failure never
fails Open.

**Grounds.** state.go:21-23 ("except quill.policy, which is informational and parsed tolerantly");
state.go:298-312 (`decodePolicy` uses `json.Unmarshal`); decision 48; stat.go:203-205 treats an
unparseable policy as a warning.

**Warrant.** A security statement that overstates strictness; the file feeds `prune`, `ls`, `check`
and the segment size of offline `append`, `roll` and `recover` (write.go:19-32).

**Backing.** Decisions 29 and 48.

**Rebuttal.** SECURITY.md excludes a process with write access to the directory; the policy's
effects are bounded by validation in write.go:25-31 and Open's re-validation.

**Verdict.** CONFIRMED by 2 of 2 votes.

**Disposition.** Fixed in commit 082bbcd22b2d. SECURITY.md now says the files the log depends on
(quill.tail, quill.format, consumer files) are parsed strictly and quill.policy is informational and
parsed tolerantly (unknown fields ignored, a parse failure never blocks Open); CHANGELOG line.
Fix-skeptic FIXED-VERIFIED against state.go:142, :191-260 and :293. Residual: README.md:66 still
describes state.go as doing a "strict decode" for quill.policy as well (a sibling row, not the
claim's line; suggested as a follow-up).

## 13. A1-72: two doc statements about offline `prune` are broader than the code (low, CONFIRMED, fixed)

**Claim.** docs/userguide.md:181-182 says `prune` "refuses a log whose policy it cannot read", and
docs/architecture.md:116-118 says the offline verbs, `prune` included, "open with no caps".

**Grounds.** cmd/quill/write.go:119-126: `prune` refuses only when the policy is unreadable and no
retention flag is given; with a flag it runs on the flags alone; it opens with the policy's caps
(write.go:119 and :150). Only `append`, `roll` and `recover` open with no caps (write.go:57, 165,
174). Verified by the auditor on a log with no quill.policy: `quill prune -dir src` exits 1
`quill.policy cannot be read; pass the retention flags explicitly`; with `-retention 1h` it runs and
exits 0.

**Warrant, backing.** The code matches decision 68 ("a parsed quill.policy or explicit retention
flags, and refuses otherwise") and decision 99.

**Rebuttal.** The userguide sentence is ambiguous ("or the retention flags" sits in the previous
clause); impact is low because the code matches the decisions.

**Verdict.** CONFIRMED by 2 of 2 votes (read, not run by the skeptics). Round-1 neighbour: docs-33.

**Disposition.** Fixed in commit da4d1ce0ca6a. The two sentences reworded to match `cmdPrune`;
CHANGELOG line; no code or test change. Fix-skeptic FIXED-VERIFIED. Residual: "retention flag" is used
loosely (`-consumer-ttl` counts as explicit in code), and for `prune` with flags and no policy the
segment size stays at the default (decision 99).

## 14. A1-71: SegmentMaxAge is documented as "never when Retention is negative" (low, CONFIRMED, fixed)

**Claim.** An explicit positive `SegmentMaxAge` rotates quiet segments even when Retention is
negative; only the default (SegmentMaxAge = Retention) is disabled.

**Grounds.** log.go:38-39 (the doc comment) and userguide.md:139; log.go:109-110 (the default copies
Retention); retention.go:97 tests only `SegmentMaxAge > 0`. Verified by the auditor: `Open{Retention:
-1, SegmentMaxAge: 1s, PruneInterval: 1s}`, one Append and Flush, a 3.5 s wait, `Stat().Segments =
2`.

**Warrant, backing.** The doc states an unconditional rule; the code makes it depend on how the value
was set. Decision 69 says only that the default is never.

**Rebuttal.** Rotating with Retention off is arguably useful (it lets `quill archive` reach quiet
segments); the honest fix may be the doc.

**Verdict.** CONFIRMED by 2 of 2 votes (read, not run by the skeptics).

**Disposition.** Fixed in commit 88a44b94a207. Three wordings corrected (the Config comment, the
scope.md row, the userguide row) to say the rule holds for the default only; code unchanged;
CHANGELOG line. Fix-skeptic FIXED-VERIFIED: a scratch test confirmed the three claims in the new
wording (Retention -1 with the default age rotates never; explicit 10 m rotates; Retention 1 h with
explicit -1 never). Residual: the explicit-positive-under-negative-Retention case has no permanent
test.

## 15. A1-70: examples/README.md misdescribes the examples (low, CONFIRMED, fixed)

**Claim.** examples/README.md:3-4 says each example takes a directory to work in and cleans up after
itself unless told otherwise. Two of three examples take no directory and the third does not clean
up.

**Grounds.** examples/log/main.go:16-17 and examples/topics/main.go:16-17 hard-code a path under
`os.TempDir()`, have no flag, and always `defer os.RemoveAll`; examples/pipeline/main.go:20 takes
`-dir` but nothing removes it. Verified by the auditor: `go run ./examples/log` and `./topics` worked
with no argument and left nothing; the pipeline's `-dir` tree was still there.

**Warrant, backing.** The sentence is false for log and topics (no directory parameter) and for
pipeline (no cleanup); the table's Run column already shows the no-argument form.

**Rebuttal.** Cosmetic; a reader following the Run column succeeds.

**Verdict.** CONFIRMED by 2 of 2 votes (read, not run by the skeptics). Round-1 neighbour: docs-45.

**Disposition.** Fixed in commit 4aec960bd533. Reworded: log and topics work under the system temp
directory and remove it when they finish; pipeline takes `-dir` and leaves it for inspection. No
CHANGELOG line. Fix-skeptic FIXED-VERIFIED. Residual: "remove it" can be read as removing the system
temp directory itself; they remove only their own `quill-example-*` subdirectory.
