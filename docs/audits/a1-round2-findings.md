# A1 adversarial audit, round 2: findings

Date 2026-10-29, against HEAD 1da37fc (v0.1.0 plus one docs-only commit; working tree clean).
Updated 2026-10-30 with pass A1h (the owner-approved held items), recorded first below; the dispositions in the
index and in the tables are current as of that update, and the text under "Top line at first writing" is kept as
written on 2026-10-29.
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

## Pass A1h (2026-10-30): the owner-approved held items

Pass A1h is a continuation of pass A1 (and of A4-1 from pass A4). On 2026-10-30 the owner approved the
recommended answer for each of the 17 items below, all of which wave 1 had held for the owner; a fix was
then written for each, one commit per item, and each fix was re-attacked by a fix-skeptic. Two of the 17
(A1-82 and A4-1) were not finished when this record was first written: a fix existed, the fix-skeptic rejected it, and it was
not in the tree. A1-82 was fixed on a third attempt (bb608d3) and passed its fix-skeptic. A4-1 has since been fixed in this
commit, for its `build/quill` half only (the CI half was refuted: the pinned setup-go reads the toolchain line); the A4-1
statements below say so.
This pass raised no new candidate items and cast no new item votes; the verdicts and votes below are
wave 1's.

### Top line (pass A1h)

17 items, all verdict **CONFIRMED** (verified from the ledger; for A1-91 see the note under the table).

| disposition | items | ids |
|-------------|-------|-----|
| fixed, fix-skeptic FIXED-VERIFIED | 16 | A1-25, A1-32, A1-82, A1-85, A1-47, A1-48, A1-13, A1-69, A1-2, A1-29, A1-35, A1-50, A1-52, A1-57, A1-87, A1-91 |
| fixed in this commit, after rejections on the audit record only (three attempts, each on the no-new-gap lens alone; `build/quill` half; the CI half refuted) | 1 | A4-1 |
| held with no fix / deliberately not changed | 0 | |

- Fix commits: 17. Sixteen are ancestors of the tree before this commit (main at fa2a5fa: bb608d3, the A1-82 fix, is its
  second-newest commit, and the other 15 appear in the commit log below it, from 5c603fc down to c18a27a); the seventeenth is A4-1, this commit,
  a rebuilt version of the rejected 0b06a86. Rejected and superseded attempts, none on main: f6fcf26 and a5707a8 (A1-82,
  replaced by bb608d3), and f800bb7, 41c8aa4, 532f39d and c9e8aa9 (A4-1, replaced by 0b06a86 and then by this commit).
- Fix-skeptic verdicts for the 16: 16 FIXED-VERIFIED. They were each attacked through six or seven lenses (13 through seven,
  three through six: A1-69, A1-52 and A1-87, which are documentation fixes and had no independent-route lens), 109 lens
  verdicts in all, and every one is "pass" (derived by counting the ledger: 102 for the first 15, 7 for A1-82). A1-82 was
  passed on its third attempt; the two attempts before it were rejected by their own fix-skeptics (a5707a8 for the sealed
  first-frame probe, see the item). A4-1's attempts 532f39d, c9e8aa9 and 0b06a86 were each found correct in code and test and rejected on the
  no-new-gap lens alone, for stale statements in the audit record (41c8aa4 before them was rejected for the false CI claim);
  this record does not claim a verdict for this commit.
- Item skeptic votes cast in this pass: 0. Wave 1 had judged the 16 A1 items with 33 votes (31 CONFIRMED, 1
  PLAUSIBLE, 1 REFUTED; derived from the index below: A1-91 had three votes, P, R and C, and the other 15 items two
  CONFIRMED each), and A4-1 with two CONFIRMED votes (supply-chain record), 35 votes for the 17 items.
- Reproduced by execution: 14 of the 16 A1 items were reproduced by a wave-1 skeptic (all but A1-35 and A1-87;
  A1-35 was then reproduced by the fix-skeptic's own test of the warning, and A1-87 is a wording item with a
  mutant check instead). A1-82 was reproduced by wave 1 and again by the fix-skeptic of the second attempt;
  the accepted fix's skeptic showed 9 tests failing against the reverted code (see the item). A4-1: 0 reproduced in wave 1 (the auditor's experiment was not re-run).
  Every accepted code fix was also shown red before and green after by the fix-skeptic. For the three
  documentation-only fixes (A1-69, A1-52, A1-87) behaviour does not change, so the fix-skeptic used a mutant instead.
- Verdict note for A1-91: wave 1 recorded it PLAUSIBLE (votes P, R, C; the REFUTED voter held that code and spec
  agree). The A1h ledger carries CONFIRMED. The fix-skeptic then reproduced the defect with built binaries (`ls`
  showing RETENTION 0s and `prune` exiting 0 on a policy of `{}`, before the fix). Inferred: the upgrade rests on
  that reproduction and the owner's approval, not on a new vote. The whole-round counts below use CONFIRMED.

### Whole round 2 after pass A1h

73 distinct items (unchanged). Verdicts: CONFIRMED 59 (was 58), PLAUSIBLE 2 (A1-18 and A1-37, was 3), REFUTED 5,
prior-deliberate 7.

| disposition | items | what it means |
|-------------|-------|---------------|
| fixed | 58 | 42 from 2026-10-29 plus the 16 above; each passed the fix-skeptic |
| fix-failed | 1 | A1-62 (rejected on 2026-10-29); not integrated. A4-1 (pass A4, rejected in A1h, since fixed in the A4-1 commit) is not one of the 73 |
| held for the owner | 2 | A1-67 and A1-80, which this ledger does not carry |
| refuted | 5 | not changed, because the claim does not hold |
| round-1-decided | 7 | deliberately not changed in round 1 |

Not carried by this ledger, and so not re-judged here: A1-67, A1-80, A1-62 and A1-59. Commits for them exist in the
tree (ba7c650 A1-67, b5ad955 A1-80, 2978378 A1-62r, 7ef431c A1-59; verified from `git log`), but their fix-skeptic
outcomes are not in the ledger this record was written from, so the table above keeps them at their 2026-10-29
disposition. Whoever records those commits should move them.

| dimension | items | fixed | fix-failed | held | refuted | round-1-decided |
|-----------|-------|-------|------------|------|---------|-----------------|
| simplify | 9 | 6 | 0 | 0 | 1 | 2 |
| dedup | 12 | 11 | 0 | 0 | 1 | 0 |
| correctness | 37 | 27 | 1 | 1 | 3 | 5 |
| docs | 15 | 14 | 0 | 1 | 0 | 0 |
| total | 73 | 58 | 1 | 2 | 5 | 7 |

### Coverage (pass A1h)

The ledger for this pass carries no auditor reports: its "clean checks" list is empty and no auditor is listed as
failing to return, because this pass ran no auditors. The auditor coverage in the Coverage section below is the
coverage of the findings; this section covers only what the fix-skeptics checked on the fixes.

Checked on every accepted fix (verified, per the fix-skeptic reports in the ledger): that a new test fails without
the fix and passes with it (a mutant for the documentation fixes); that the root cause is closed, including sibling
forms; that gofmt and `go vet` are clean and `go test -race -count=1 ./...` passes in every package; that no golden
bytes changed and `go doc -all` differs only where intended; that the amended decisions, the user guide,
scope.md, format.md, architecture.md (where each applies) and CHANGELOG.md agree with the code; that the changed code is reached by the
public path or the built binary.

Not covered, or covered only in part (verified from the fix-skeptic reports unless marked):

- staticcheck v0.8.2 and deadcode v0.52.0 were run, clean, for A1-32, A1-85, A1-47, A1-82 and A1-91; for A1-48 through
  `hack/gate.sh` with FUZZTIME=0, which passed every step but `go mod tidy` (an artifact of the git-archive copy, which
  has no repository); staticcheck only for A1-35. They were not run, or not reported, for A1-25, A1-13, A1-69, A1-2,
  A1-29, A1-50, A1-52, A1-57 and A1-87. The fixers' own gate runs are reported only in some summaries (A1-13 states
  gofmt, vet, race, staticcheck and deadcode clean); this recorder re-ran none of it (docs only, nothing heavy).
- The fuzz step of the gate was not run by any fix-skeptic (A1-32, A1-48 and A1-82 say so, and A1-82's also did not
  re-run `go mod tidy`; none of the changes touches a
  parser or the format, except A1-2's key-folding in `internal/strictjson`, whose differential test over all 2,994
  runes with a fold orbit replaced it).
- Linux x86-64 box runs: A1-25 (simulator tests only; the chmod reproduction cannot run as root), A1-82 (the relevant tests
  under `-race`), A1-32, A1-85 (tests;
  the cross-user `setpriv` case could not be run because creating a user on the shared box was refused), A1-48, A1-13,
  A1-2, A1-29, A1-50, A1-91. Not run on Linux: A1-47, A1-69, A1-35, A1-52, A1-57, A1-87 (platform-independent code
  paths, inferred).
- Not run by anyone: a real EIO on an `os.File` for A1-82 (only the simulator's injected EIO, so inferred to match), a real second account for A1-85, a terminal that honours 8-bit C1 for A1-50, and the no-fault
  variant of A1-25 on a real filesystem (it is covered by a configuration-validation test).
- Auditors that failed to return: none. Auditors that returned nothing: none.

### Items

Each item states the wave-1 argument briefly (the full text is in the dimension report named in the heading), the
verdict, the reproduction, and what was done. "Fix-skeptic" outcomes are FIXED-VERIFIED unless said otherwise.

#### A1-25: after a failed rotation the active segment can outgrow the 1.5 GiB limit (medium, correctness report item 8; CONFIRMED, 2 votes; fixed c18a27a)

- **Claim.** While rotation keeps failing, decision 60(d) keeps appending and nothing bounds the active segment, so it
  can pass the 1.5 GiB limit that Open and readers enforce; the log then cannot be opened though every record was
  acknowledged durable.
- **Grounds.** flush.go records a rotation error and carries on; append.go and flush.go never look at the active segment
  size; load.go refuses a file over `maxSegmentBytes` (1536 MiB). The same overshoot is reachable with no fault when
  SegmentBytes is 1 GiB and FlushBytes or MaxBufferBytes exceeds 512 MiB (read, not run).
- **Warrant.** Decision 60(d) accepts appends after a failed rotation with no ceiling; decision 16 and format.md refuse a
  file over the limit. Rotation can fail (EMFILE, a permission change) while the open descriptor keeps working.
- **Backing.** Decisions 60(d), 16, 46, 69; guarantee 7.
- **Rebuttal.** The choice was deliberate and `Err()` reports it; reaching the limit needs about 1.5 GiB of writes under
  a persistent fault.
- **Reproduction (verified, macOS).** SegmentBytes 1 MiB, the log directory made read-only after Open, 400 appends of
  4 MiB: a 1,677,729,073-byte segment, and Open then returns "damaged at byte 0 (intact to byte 0)". After the fix, the
  same program on the real filesystem: 2 appends accepted, 398 refused with the permission error ("active segment would
  reach 12582982 bytes, ceiling 9437218"), a segment of 8,388,644 bytes, reopen succeeds at head 2.
- **Disposition.** Owner approved the ceiling on 2026-10-30. Fixed in c18a27a, fix-skeptic FIXED-VERIFIED (7 of 7
  lenses). While rotation is failing, an append that would push the segment past SegmentBytes +
  max(FlushBytes, MaxBufferBytes) + max(MaxBatchBytes, MaxPayloadBytes+34) is refused with the rotation error and takes
  no sequence number; Open refuses (ErrConfig) a Config whose ceiling reaches the limit; an oversize intact file is
  reported as "refused without being read" by Open, Check and `quill check`. Decisions 16 and 60(d), the user guide,
  format.md, scope.md and architecture.md are amended; CHANGELOG has a Fixed line. No new exported symbol. Left as
  is: docs/design/build-prompt.md still describes the historical keep-appending behaviour (a build record, not a
  spec); `quill check` still prints "damaged at byte 0" for a segment that is not a regular file.

#### A1-82: a read I/O error is reported as ErrCorrupt or a skippable gap (medium, correctness report item 12; CONFIRMED, 2 votes; fixed bb608d3)

- **Claim.** In `scanSource`, any error from `walk` becomes ErrCorrupt (active segment, cause flattened) or a
  `*GapError` over the whole sealed segment, so a consumer following the resume idiom ("iterate again from To + 1")
  skips a segment after one transient EIO.
- **Grounds.** read.go:249-257 in the audited tree; only the load path had a test.
- **Warrant.** Decision 79 says an I/O error met while scanning is returned as that error and never as ErrCorrupt;
  scope.md's resume idiom plus guarantee 14.
- **Backing.** Decisions 79 and 38; scope.md guarantees 6 and 14.
- **Rebuttal.** Decision 94 maps open-time I/O failures to a per-view hole, and decision 38 lists "unreadable" among
  hole causes, so the sealed half is arguable; the active half is not.
- **Reproduction (verified, macOS, simulator returning EIO on reads).** Sealed segment: a `*GapError` over its range, no
  ErrCorrupt and no EIO match. Active segment: `corrupt log: the active segment below the durable offset: input/output
  error`, `Is(ErrCorrupt)` true, `Is(EIO)` false.
- **Disposition.** Owner approved on 2026-10-30. **Fixed in bb608d3**, the third attempt, fix-skeptic FIXED-VERIFIED (7 of 7
  lenses: test-fails-without-fix, root-cause, behaviour-preserved, no-new-gap, reachable, ladder, independent-route).
  Attempt 1 (f6fcf26) and attempt 2 (a5707a8, which fixed `scanSource`, `ensureIndex`, `OffsetForTime` and the Reader's
  active-segment probe) were rejected; a5707a8's fix-skeptic reproduced the remaining defect: with EIO injected on reads of
  sealed segment 2, `Reader.Read(21, 1)` returned `*quill.GapError: records 21 to 32 are unavailable` (and `Read(1, 1)`
  "records 1 to 20" for segment 1), because the Reader's sealed first-frame probe set the segment bad and `view()`
  turned it into a hole; its test discarded the error (`_, _ = r.Read(mid, 1)`) and the new text saying an I/O error is
  never a gap was false for that path. bb608d3 started from a5707a8 and: (1) `Reader.segment` returns
  `(*readerSegment, error)`, and a non-frame error from the first-frame probe is returned from `view()` uncached
  (reader.go:246-247), while open failures (EMFILE, EACCES) stay a one-view hole (decision 94); (2) `loadSealed`
  (load.go:449) returns a read I/O error from `verifySegment` instead of recording a hole and `load()` fails `Open` with
  it (load.go:321-330), as it did for the newest segment, with damage and open failures still holes (decision 98). Docs
  amended in place with the note "Stage 2 audit, owner-approved 2026-10-30": decisions 38, 39, 79, 94, 98; scope.md
  guarantee 14; user guide section 3; the `GapError` comment; CHANGELOG Unreleased. No exported symbol, flag or file.
  Regression test: `TestAReaderFirstFrameProbeIOErrorOfASealedSegmentIsReturnedAsThatError` (read_ioerror_test.go:158)
  asserts for segment 1 and a middle segment that the read that hits the fault is `errors.Is(EIO)`, not a `*GapError`,
  not ErrCorrupt, and that the same read succeeds once the fault clears. With only the five production files
  (errors.go, load.go, read.go, reader.go, seek.go) reverted, 9 tests fail, each for the claimed reason (verified by the
  fix-skeptic). The fix-skeptic also probed with its own programs: a fresh Reader with EIO on a segment gave the EIO from
  `Records`, `Follow` and `OffsetForTime`; a first frame damaged with real bytes is still a `*GapError`; `Open` with
  persistent EIO on the oldest sealed segment fails with the EIO. Sites checked (verified by reading, every site in
  reader.go, read.go, seek.go, load.go, stat.go, archive.go, retention.go and iter.go that classifies a read error) are
  listed in stage2-owner-decisions.md, section A1-82. **Objections that stand, none against the claim:** a first-frame
  EIO now fails every call on that Reader that builds a view while the fault lasts (reproduced), and a persistent EIO on
  any sealed segment now makes `Open` fail; decision 94 still carries the phrase "in an intermediate form, made a hole
  for one view" (decisions.md:102), a state that never shipped, which should be deleted (left alone here: this record
  changes docs/audits only); real EIO on an `os.File` was not exercised.

#### A1-32: the reserved `_archive` consumer cannot be removed or rewound (medium, correctness report item 9; CONFIRMED, 2 votes; fixed a3955cd)

- **Claim.** `register`, `unregister`, `commit` and `DeleteConsumer` refuse `_archive` ("a leading underscore is
  reserved"); after a restore leaves it ahead of the head, `quill archive` fails with ErrAhead and the user guide's
  remedy is refused for exactly this consumer.
- **Grounds.** consumer.go skipped `validName` for the underscore name only for Archive's internal path, and `remove`
  validated unconditionally; retention keeps every segment at or above a protected consumer's offset.
- **Warrant.** A protected consumer whose offset file cannot be moved holds retention at its offset and wedges the
  archiver. **Backing.** Decisions 57, 36, 41, 67; user guide section 9.
- **Rebuttal.** Deleting `consumers/_archive` by hand or ConsumerTTL clears it; after an archive the offset is head+1.
- **Reproduction (verified, macOS).** `quill archive` then `unregister`, `register` and `commit -name _archive -offset 1`
  each exit 1 with the reserved-name error; with the offset file rewritten to 99, `quill archive` exits 1 with ErrAhead.
  After the fix, the built binary: `commit -name _archive -offset 1` prints `_archive 1`, `quill archive` succeeds,
  `unregister -name _archive` exits 0, and `register -name _other` is still refused.
- **Disposition.** Owner approved on 2026-10-30. Fixed in a3955cd, fix-skeptic FIXED-VERIFIED (7 of 7 lenses): one
  unexported validator, `validConsumerName`, accepts exactly `_archive`; `Config.ProtectedConsumers` still refuses it.
  Decisions 16, 41 and 67, the user guide, format.md, scope.md, build-prompt.md and CHANGELOG amended. Consequence the
  fix-skeptic noted and the owner's answer implies: `register -no-protect -name _archive` unprotects the archiver until the
  next `quill archive`.

#### A1-85: Reader-created consumer files ignore the log's Perm (medium, correctness report item 13; CONFIRMED, 2 votes; fixed 279d7ce)

- **Claim.** `Reader.consumerStore` and `Archive` hard-code 0600, so in the group arrangement scope.md describes, a
  consumer in another account creates a file the writer cannot read, and retention treats the consumer as broken.
- **Grounds.** consumer.go and archive.go hard-coded 0o600; `ReaderOptions` has no Perm. **Warrant.** scope.md's group
  arrangement; retention protects every segment from an unreadable consumer file. **Backing.** Decisions 41, 51, 67.
- **Rebuttal.** Single-user deployments are unaffected; the failure is on the safe side.
- **Reproduction (verified).** Linux box, writer Perm 0660 in a setgid directory, a reader run as another group member:
  the consumer file is created 0600, the writer's `Consumers()` returns permission denied, and `Prune(true)` keeps the
  segment it would otherwise delete.
- **Disposition.** Owner approved on 2026-10-30. Fixed in 279d7ce, fix-skeptic FIXED-VERIFIED (7 of 7 lenses):
  `OpenReader` reads the permission bits of `quill.tail` once (the writer sets them to Config.Perm exactly), ORs in
  0o600, and the Reader's consumer files and everything Archive creates use that mode. No new exported symbol, flag or
  option. Decisions 41, 51 and 67 amended. Residuals: the real cross-user case was not re-run (inferred from file
  modes and the passing tests); a lock file created by a Reader follows the umask (disclosed in the docs), so a second
  process of the same name from another group member could not open it read-write.

#### A1-47: extra positional arguments are silently ignored (medium, correctness report item 10; CONFIRMED, 2 votes; fixed 2c4e4f4)

- **Claim.** Every verb but `append` ignores positionals and flag parsing stops at the first one, so `quill -dir L read
  100` prints from the first record and `tail 5 -json` prints text, both exit 0.
- **Grounds.** cmd/quill/verbs.go parsed flags then never read the arguments. **Warrant.** The user's intent is dropped
  with status 0. **Backing.** scope.md exit code 2 is "usage".
- **Rebuttal.** The documented grammar is flags only; the change turns a previously successful, misdirected call into
  exit 2. `help` takes a verb name, so it is exempt with `append`.
- **Reproduction (verified, macOS).** 13 records; `read 10` starts at seq 1, exit 0; `tail 5 -json` prints ten text
  lines; `stat -json extra` and `version foo` exit 0.
- **Disposition.** Owner approved on 2026-10-30. Fixed in 2c4e4f4, fix-skeptic FIXED-VERIFIED (7 of 7 lenses):
  any other verb given a positional exits 2 with `unexpected argument "X"; flags go before arguments`, before anything
  is opened; one unexported `args` flag on the verb table exempts `append` and `help`. Decision 53, the user guide and
  CHANGELOG amended. Left as is (inside the approved exemption): `help read x` ignores the second positional, and
  `append x -sync=false` records the flag as a line.

#### A1-48: `-from`, `-limit` and `-n` parse numbers in base 0 (medium, correctness report item 11; CONFIRMED, 2 votes; fixed b4fe935)

- **Claim.** `-n 010` is 8, `-from 0x10` is 16, `-limit 1_0` is 10, while every other number flag is plain decimal.
- **Grounds.** `flag.Uint64Var` (base 0) in verbs.go against `parseDecimal` in read.go. **Warrant.** A zero-padded script
  variable silently becomes octal. **Backing.** `parseDecimal`'s comment and the refused-numbers table; decision 96.
- **Rebuttal.** It is the standard flag package's behaviour and hex may be a feature; only octal `010` is plainly
  dangerous; the harm is limited to read-only verbs.
- **Reproduction (verified, macOS and, after the fix, Debian).** 12 records: `tail -n 010` gives 8 lines, `-n 08` exits 2,
  `cat -from 0x2` starts at seq 2, `cat -limit 1_0` gives 10 lines. After the fix `tail -n 010` gives 10; `0x10`, `1_0`,
  `0b1`, `1e1`, `-1`, `+5`, an empty value and 2^64 all exit 2 with empty stdout; 2^64-1 is accepted.
- **Disposition.** Owner approved on 2026-10-30. Fixed in b4fe935, fix-skeptic FIXED-VERIFIED (7 of 7 lenses): a small
  `decimalFlag` value calls `parseDecimal`. Decision 96, the user guide and CHANGELOG amended. Cosmetic: help shows
  `-n value` where it showed `uint`.

#### A1-13: Archive returns ErrCorrupt, not ErrFutureVersion, for a newer destination (medium, dedup report item 1; CONFIRMED, 2 votes; fixed b529db5)

- **Claim.** The state-file load is written out three times for `quill.format`; Archive's copy wraps every decode error as
  ErrCorrupt, so a `quill/2` destination gives ErrCorrupt where Open and OpenReader give ErrFutureVersion.
- **Grounds.** load.go, reader.go and archive.go. **Warrant.** Two copies carry the rule and the third drifted.
  **Backing.** Decision 6, errors.go and format.md.
- **Rebuttal.** The prototype was net zero lines; it changes an exported error class.
- **Reproduction (verified).** A destination whose `quill.format` is `quill/2` gives `corrupt log: destination
  quill.format: ... newer than this build`, ErrFutureVersion false, ErrCorrupt true; after the fix the built CLI returns
  the ErrFutureVersion message.
- **Disposition.** Owner approved on 2026-10-30. Fixed in b529db5, fix-skeptic FIXED-VERIFIED (7 of 7 lenses): one
  `loadFormat` in state.go serves the three sites. Decisions 6 and 85 and CHANGELOG amended. Not done, by design: the
  `quill.tail` half of the dedup (`loadTail`). Cosmetic: the ErrCorrupt text for a damaged destination marker no longer
  says "destination".

#### A1-69: MaxBatchBytes is checked against a 34-byte-per-record estimate (medium, docs report item 4; CONFIRMED, 2 votes; fixed 5b30e7f)

- **Claim.** The docs call MaxBatchBytes the largest encoded batch but append.go checks `34 + len(payload)` per record,
  so 500,000 one-byte records (about 7.5 MB encoded) are refused and `quill append` from stdin refuses everything.
- **Grounds.** append.go:30-33; `quill append` sends all stdin lines as one `AppendBatch`. **Warrant.** The documented
  limit is on encoded bytes; the enforced one over-counts, and the disk-bound formula depends on the over-count.
  **Backing.** Decision 46, scope.md. **Rebuttal.** The error says "about" and the estimate is deliberately
  conservative; an exact check changes behaviour.
- **Reproduction (verified, macOS and Linux).** 400,000 one-byte records accepted; 500,000 lines piped to `quill append`
  refused ("batch of about 17500000 bytes"), exit 1, nothing appended. After the fix the same CLI accepts 479,349 and
  refuses 479,350, which agrees with 16 MiB / 35.
- **Disposition.** Owner approved the documentation-only answer on 2026-10-30. Fixed in 5b30e7f, fix-skeptic
  FIXED-VERIFIED (6 of 6 lenses): the limit is now described as checked against `len(payload)+34` per record, and
  `quill append` from stdin as one batch subject to it. The check is unchanged. The text says "about 479,000"; the
  recommended answer had said "about 480,000", which the measured 479,349 does not support. Decision 46, scope.md, the
  user guide, the Config and AppendBatch comments, the `append` verb summary, docs/reference.md (regenerated) and
  CHANGELOG amended. Residual: errors.go still says only "a batch above MaxBatchBytes" (not wrong).

#### A1-2: a case-variant duplicate key in quill.tail wins silently (low, docs report item 5; CONFIRMED, 2 votes; fixed 89957ae)

- **Claim.** `{"seq":1,"SEQ":2,...}` resolves to Seq 2 with no error because `checkDuplicateKeys` compares exact strings
  while `encoding/json` matches field names case-insensitively; format.md says a duplicate key is refused.
- **Grounds.** strictjson.go:30-49 and :123, state.go:138-158. **Warrant.** The rule "a duplicate key is refused" is
  broken for case variants. **Backing.** format.md, guarantee 9, decisions 8, 48, 72.
- **Rebuttal.** No writer emits such a file; decision 105 deliberately preserved the slow path; whitespace, order and a
  missing newline are not listed refusals.
- **Reproduction (verified, macOS).** `decodeTail` on that input returns Seq 2 and nil; after the fix it returns the
  duplicate-key error, and the built `quill stat` and `quill check` print it for a tampered file.
- **Disposition.** Owner approved on 2026-10-30. Fixed in 89957ae, fix-skeptic FIXED-VERIFIED (7 of 7 lenses): keys
  are compared after the case folding `encoding/json` uses (smallest rune of the `unicode.SimpleFold` orbit; a
  differential test over all 2,994 rune orbits found no mismatch). format.md now lists the shapes the strict parser
  still accepts. Decisions 72 and 105 amended. Residual: the fold also applies to map-typed targets, which `quill`
  never decodes into.

#### A1-29: Open refuses a symlinked Dir that OpenReader accepts (low, correctness report item 27; CONFIRMED, 2 votes; fixed 8c0c531)

- **Claim.** `Open(Config{Dir: link})` returns ErrNoLog "is not a directory" for a symlink to a directory while
  `OpenReader(link)` and a symlinked parent work.
- **Grounds.** `mkdirAll` uses Lstat (log.go:340-343 in the audited tree). **Warrant.** Decision 51's no-follow rule
  covers files inside the log, not the directory itself. **Backing.** Decisions 51, 62, scope.md. **Rebuttal.** It may be
  hardening against a planted link; nothing in the tree says so.
- **Reproduction (verified, macOS).** `Open` on a link and on `link/sub` both return the error; `OpenReader(link)` works.
- **Disposition.** Owner approved following the link on 2026-10-30. Fixed in 8c0c531, fix-skeptic FIXED-VERIFIED (7 of 7
  lenses), after an earlier attempt that missed a link in an ancestor of a Dir that does not exist yet. An unexported
  `resolveDirLink` resolves the deepest existing part of Dir once with `filepath.EvalSymlinks`; a link to a file or to
  nothing stays ErrNoLog and the message says "symbolic link"; files inside the log are still opened with O_NOFOLLOW.
  Decisions 51 and 62, scope.md, the user guide, SECURITY.md and CHANGELOG amended. Residual: the link is resolved
  once at open, so a swap afterwards is not re-checked (inside the threat SECURITY.md excludes; inferred).

#### A1-35: one future-stamped segment pins age expiry of every newer one (low, correctness report item 28; CONFIRMED, 2 votes; fixed 19c0996)

- **Claim.** The age sweep is a strict prefix, so a segment whose last record is stamped in the future blocks expiry of
  everything after it until the clock catches up; nothing is logged.
- **Grounds.** retention.go:111-123 in the audited tree. **Warrant.** The sweep stops at the first segment newer than the
  cutoff. **Backing.** Decision 44; scope.md. **Rebuttal.** Prefix deletion is deliberate; the caps still bound disk;
  `quill check` flags the record.
- **Reproduction (reported by the auditor, not re-run in wave 1).** Segment 2 stamped a year ahead, then +48 h at
  Retention 1 h: after 8 forced prunes `segs=8 first=21 head=90` and no log. The fix-skeptic reproduced the warning
  with its own scenario after the fix (verified).
- **Disposition.** Owner approved the minimum (warn and document) on 2026-10-30. Fixed in 19c0996, fix-skeptic
  FIXED-VERIFIED (7 of 7 lenses): when the sweep stops at a sealed segment whose newest record is stamped more than 24 h
  ahead, the prune pass logs a warning (segment id, last_at, kept count) through the configured logger. The prefix rule
  is unchanged. Decision 44, scope.md, the user guide and CHANGELOG amended. Residual: `quill` sets no logger, so a CLI
  prune shows no warning (as with the existing clock-guard warning); it repeats each pass while the pin lasts.

#### A1-50: `-json` passes C1 control characters raw (low, correctness report item 29; CONFIRMED, 2 votes; fixed fd52463)

- **Claim.** `-json` copies U+0080 to U+009F (for example the CSI introducer U+009B) and DEL raw although the guide and
  scope.md say no raw control character reaches the terminal; the text form escapes them.
- **Grounds.** `appendJSONString` in cmd/quill/print.go. **Warrant.** A terminal that honours 8-bit C1 in UTF-8 mode would
  act on U+009B (inferred, not tested). **Backing.** user guide and scope.md. **Rebuttal.** `encoding/json` also leaves C1
  raw; the payload author must already write to the log.
- **Reproduction (verified, macOS).** `quill cat -json` of `a\xc2\x9b31mred\xc2\x85b` emits bytes 302 233 and 302 205
  unescaped; the text form emits `\u009b` and `\u0085`.
- **Disposition.** Owner approved on 2026-10-30. Fixed in fd52463, fix-skeptic FIXED-VERIFIED (7 of 7 lenses): DEL and
  U+0080 to U+009F are written as `\u00XX` in `payload`; the JSON stays valid and lossless (round-tripped through
  `encoding/json`); `cat`, `read` and `tail` share the printer. Decision 35, scope.md, the user guide and CHANGELOG
  amended. Not covered: the effect on a real terminal. Residual: non-payload JSON (file names in `check`) still goes
  through `json.Marshal`.

#### A1-52: Read can exceed MaxReadBytes by one record (low, correctness report item 30; CONFIRMED, 2 votes; fixed 25e96c2)

- **Claim.** The cap is checked after each record, so a read returns up to cap plus one record, contradicting "at most
  `limit` records and `MaxReadBytes` of payload, but always at least one".
- **Grounds.** read.go:161, :235, :242 in the audited tree. **Warrant.** A record is admitted whenever the running total is
  below the cap. **Backing.** read.go doc comment, scope.md, guarantee 7. **Rebuttal.** The overshoot is bounded by one
  record; the shorter wordings already read as "stop once reached".
- **Reproduction (verified, macOS).** MaxReadBytes 16384, five 10 KiB records: 2 records, 20480 bytes; a 1 MiB cap with
  one 1 MiB-1 record and two 5 MiB records: 2 records, 6,291,455 bytes.
- **Disposition.** Owner approved the rewording on 2026-10-30. Fixed in 25e96c2, fix-skeptic FIXED-VERIFIED (6 of 6
  lenses): Read, ReadWait, Records and Follow are documented as stopping once at least MaxReadBytes of payload have been
  returned, overshooting by at most one record. A test pins the rule and a mutant of the check was caught by it. The doc
  comments, user guide, scope.md and decisions 50 and 79 amended; CHANGELOG has a Changed line. Residual: the user guide's
  short wording at section 3 omits "at least"; the Config table has the full rule.

#### A1-57: `stat -json` holes use capitalised keys and `null` (low, correctness report item 31; CONFIRMED, 2 votes; fixed c91329c)

- **Claim.** `quill stat -json` writes a hole as `{"From":..,"To":..}` and an empty list as `null`, while every other CLI
  JSON key is lowercase; no document fixes the stat, check or consumers schema.
- **Grounds.** cmd/quill/read.go marshals `[]quill.Hole`, which has no tags. **Warrant.** The first public release freezes
  whatever scripts parse. **Backing.** scope.md fixes only the per-record schema. **Rebuttal.** The shapes are
  undocumented, so there is no contract to break yet, which is the reason to settle it before release.
- **Reproduction (verified).** Wave 1 used a marshal program on an equivalent type; the fix-skeptic ran the built binary
  before and after: `"holes":null` became `"holes":[]`, and `[{"From":1,"To":72}]` became `[{"from":1,"to":72}]`.
- **Disposition.** Owner approved before release on 2026-10-30. Fixed in c91329c, fix-skeptic FIXED-VERIFIED (7 of 7
  lenses): holes are written through a function-local tagged type, always as an array; `quill.Hole` gains no tags. The
  stat, check and consumers JSON schemas are documented in scope.md and pointed to from the user guide. Decision 35 and
  CHANGELOG amended. Left as is (deliberate, A1-56): `stat` shows no hole for a deleted middle segment.

#### A1-87: the clock guard rounds up where the text says "at most 25%" (low, docs report item 6; CONFIRMED, 2 votes; fixed 50201e1)

- **Claim.** `limit := (len(l.sealed) + 3) / 4` deletes the ceiling of a quarter (2 of 5, 4 of 15) where decision 44 and
  scope.md say "at most 25% (at least one)"; a test pins the larger number.
- **Grounds.** retention.go:139 and retention_test.go:266 in the audited tree; a `max(1, n/4)` mutant fails that line.
  **Warrant.** "At most 25 percent" is a floor with a minimum of one. **Backing.** Decision 44, scope.md, T21.
  **Rebuttal.** Rounding up is defensible for tiny logs; the difference is at most three segments.
- **Reproduction.** Hand arithmetic (verified by the skeptics); the fix-skeptic re-ran the mutant: the test fails at the
  second pass (13 left, 12 expected).
- **Disposition.** Owner approved the rewording on 2026-10-30. Fixed in 50201e1, fix-skeptic FIXED-VERIFIED (6 of 6
  lenses): every statement now reads "at most a quarter of the sealed segments, rounded up (at least one)". No code
  change. Decision 44 and CHANGELOG amended. Cosmetic: a missing comma in decision 44.

#### A1-91: a quill.policy of `{}` is acted on as retention 0 (low, correctness report item 32; CONFIRMED in the A1h ledger, PLAUSIBLE in wave 1; fixed 5c603fc)

- **Claim.** `{}`, `null` and `{"unknown":1}` parse as a valid policy of zeros; `quill ls` shows RETENTION 0s and `quill
  prune` runs with the default 24 h and no caps.
- **Grounds.** state.go:300 `decodePolicy` accepts any object. **Warrant.** Decision 68's reason is that offline tools must
  not prune with default retention; a version-skewed policy would produce such a file. **Backing.** Decisions 29, 48,
  68, 99. **Rebuttal.** It needs a hand-edited or future-format file; v0.1.0 writers always write every field (an off
  retention is `-1ns`); one wave-1 skeptic held that code and spec agree.
- **Reproduction (verified, macOS).** `{}`, `null` and `{"unknown":1}` give RETENTION 0s and a `prune` that exits 0;
  `{"retention":"bogus"}` and an empty file are refused with exit 1.
- **Disposition.** Owner approved (low priority) on 2026-10-30. Fixed in 5c603fc, fix-skeptic FIXED-VERIFIED (7 of 7
  lenses): `decodePolicy` fails when `retention` is absent or empty, so such a policy is unparseable; `ReadPolicy` returns
  ok == false, `prune` refuses without explicit flags, `ls` shows `?` and "policy unreadable", `check` warns "cannot be
  parsed". Decisions 29, 48, 68 and 87, format.md, the `ReadPolicy` comment and CHANGELOG amended. Differences from the
  recommendation: `ls` shows `?` (the existing unreadable-policy convention), not "unknown". Weakest point: a policy with
  `retention` present and every cap missing (for example a skewed `{"retention":"1h"}`) still parses as caps of zero.

#### A4-1: the Go toolchain pin is enforced only by the gate (medium, supply-chain report; CONFIRMED, 2 votes, 0 reproduced; fixed in this commit for the `build/quill` half; the CI half REFUTED)

- **Claim.** The `toolchain go1.27.3` line in go.mod is a pin only in `hack/gate.sh`; `build/quill` uses whatever `go` is on
  PATH, and (at the audited tree) CI installed the newest 1.27.x. **The CI half is REFUTED at the pinned setup-go v7.0.0**, which
  reads go.mod's toolchain line before the go line unless `GOTOOLCHAIN=local` (see the Disposition).
- **Grounds.** `hack/gate.sh` exports `GOTOOLCHAIN` from go.mod; `build/quill` did not. Under `GOTOOLCHAIN=auto` the toolchain
  line is a minimum. The CI half rested on setup-go v5.6.0 not reading the toolchain line (reported by the auditor from
  its source; not reproduced; true of v5.6.0, false of the v7.0.0 pin now on main). **Warrant.** README.md and CONTRIBUTING.md call the go.mod toolchain the one supported
  version, and the release binary's bytes depend on the compiler. **Backing.** Decisions 70 and 88.
- **Rebuttal.** All places resolved to go1.27.3 at the time; the SBOM discloses the Go used.
- **Disposition.** Owner approved landing the pin on 2026-10-30 (the recommended answer, with its stated alternative of
  dropping the CI half). Fixed in this commit, `build/quill` half only: `build/quill` derives and exports `GOTOOLCHAIN` from
  go.mod's toolchain line as the gate does, and `TestBuildScriptPinsToolchain` fails without it. The CI half was refuted: the
  workflow pins setup-go v7.0.0 (`ci.yml` lines 24, 44, 54, 75 and 87), which the fix-skeptic reports reads the toolchain line
  before the go line unless `GOTOOLCHAIN=local` (reported, not run on a runner), so the workflow-level literal and its test were
  dropped and the `ci.yml` comment about the patch level is true. Earlier attempts (532f39d, c9e8aa9 and 0b06a86) were
  each found correct in code and test and rejected by their fix-skeptic on the no-new-gap lens only, because they left this record, the
  supply-chain record and the owner-decisions table saying the old thing; this commit corrects all three
  ([supply-chain.md](supply-chain.md), [stage2-owner-decisions.md](stage2-owner-decisions.md) and this file). The first attempt
  (f800bb7 and 41c8aa4) carried the false CI claim and was not used.

## Top line at first writing (2026-10-29)

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

**What matters most for release (as written 2026-10-29; updated by pass A1h above).** Held, not fixed: A1-80 and A1-67 (above), A1-25 (after a failed
rotation the active segment can outgrow the 1.5 GiB limit, and the log then cannot be reopened),
A1-82 (a transient read error is reported as ErrCorrupt, or as a gap that the documented resume
idiom skips) and A1-32 (the reserved archiver consumer cannot be removed or rewound). Since then A1-25,
A1-32 and A1-82 are fixed (c18a27a, a3955cd, bb608d3; A1-82 on its third attempt, after a5707a8 was rejected for the Reader's
sealed first-frame probe, which reported one EIO as a whole-segment gap). Fixed, and
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
tree). Pass A1h added 16 more fixes (all FIXED-VERIFIED, see its top line; A1-82 on its third attempt) and no further rejected
fix among the 73 items (A4-1, which belongs to pass A4, was fixed in the A4-1 commit after rejections on the audit record only).

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
| A1-25 | flush.go:104-130, load.go:76 | medium | CONFIRMED | CC | Y | fixed c18a27a |
| A1-32 | consumer.go:73-79, 267-269 | medium | CONFIRMED | CC | Y | fixed a3955cd |
| A1-47 | cmd/quill/verbs.go:159-169 | medium | CONFIRMED | CC | Y | fixed 2c4e4f4 |
| A1-48 | cmd/quill/verbs.go:71, 124-125 | medium | CONFIRMED | CC | Y | fixed b4fe935 |
| A1-82 | read.go:249-257 | medium | CONFIRMED | CC | Y | fixed bb608d3 |
| A1-85 | consumer.go:313, archive.go:25 | medium | CONFIRMED | CC | Y | fixed 279d7ce |
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
| A1-29 | log.go:338-358 | low | CONFIRMED | CC | Y | fixed 8c0c531 |
| A1-35 | retention.go:111-123 | low | CONFIRMED | CC | N | fixed 19c0996 |
| A1-50 | cmd/quill/print.go:81-107 | low | CONFIRMED | CC | Y | fixed fd52463 |
| A1-52 | read.go:161-242 | low | CONFIRMED | CC | Y | fixed 25e96c2 |
| A1-57 | cmd/quill/read.go:81-82 | low | CONFIRMED | CC | Y | fixed c91329c |
| A1-91 | state.go:300 | low | PLAUSIBLE, CONFIRMED in A1h | PRC | Y | fixed 5c603fc |
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
| A1-69 | append.go:30-33 | medium | CONFIRMED | CC | Y | fixed 5b30e7f |
| A1-70 | examples/README.md:3-4 | low | CONFIRMED | CC | N | fixed 4aec960 |
| A1-71 | log.go:38-39, retention.go:97 | low | CONFIRMED | CC | N | fixed 88a44b9 |
| A1-72 | docs/userguide.md:181-182 | low | CONFIRMED | CC | N | fixed da4d1ce |
| A1-73 | SECURITY.md:14 | low | CONFIRMED | CC | N | fixed 082bbcd |
| A1-74 | hack/gate.sh:48-59 | low | CONFIRMED | CC | N | fixed e498890 |
| A1-79 | docs/reference.md:5, README.md, godoc | low | CONFIRMED | CC | N | fixed a593c45 |
| A1-94 | retention_test.go:289-330 | low | CONFIRMED | CC | Y | fixed 47a5ebb |
| A1-95 | retention.go:238, 126, flush.go:130 | low | CONFIRMED | CC | Y | fixed 9810a56 |
| A1-97 | docs/format.md:179 and others | low | CONFIRMED | CC | N | fixed f1df4ae |
| A1-2 | strictjson.go:30-49, state.go:138-158 | low | CONFIRMED | CC | Y | fixed 89957ae |
| A1-87 | retention.go:139 | low | CONFIRMED | CC | N | fixed 50201e1 |

### Dedup (12 items, a1-round2-dedup.md)

| id | anchor | sev | verdict | votes | repro | disposition |
|----|--------|-----|---------|-------|-------|-------------|
| A1-13 | load.go:277-296, archive.go:99-111 | medium | CONFIRMED | CC | Y | fixed b529db5 |
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
