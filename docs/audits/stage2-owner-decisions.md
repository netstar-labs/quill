# Stage 2 audit: decisions held for the owner

Date 2026-10-29, against HEAD 1da37fc. The stage 2 audit (pass A1, round 2: adversarial audit in four
dimensions) applied every finding that is behaviour-preserving, or that fixes a clear defect with a regression
test, and held back every finding whose fix changes observable behaviour, an exported error class, an exit code,
a CLI grammar, an on-disk or output shape, or the wording of a guarantee. This file lists those held items, one
section each, with the claim, why it was held (what the fix would change), a recommended answer and whatever fix
is already written. The full items (grounds, warrant, backing, rebuttal, skeptic votes and reproductions) are in
the per-dimension reports: [correctness](a1-round2-correctness.md), [docs](a1-round2-docs.md) and
[dedup](a1-round2-dedup.md); the counts are in [a1-round2-findings.md](a1-round2-findings.md).

The recommended answers are the recorder's reading of the auditor's proposed fix and the skeptics' notes; they are
recommendations, not decisions. No held item has a fix committed. Where a fix is written out below it is the
proposed fix, or a prototype in the session's scratch area that passed the suite and was not committed. Ranked by
severity, then reach.

| id | sev | one line | recommended |
|----|-----|----------|-------------|
| A1-67 | high | the documented `quill commit` rewind of an ahead consumer fails with ErrAhead | fix the doc now; approve letting `commit` rewind |
| A1-80 | high | a live protected consumer stops the writer from restarting | approve |
| A1-25 | medium | after a failed rotation the active segment can outgrow the 1.5 GiB limit | approve a ceiling |
| A1-32 | medium | the reserved `_archive` consumer cannot be removed or rewound | approve |
| A1-47 | medium | extra positional arguments are silently ignored | approve (exit 2) |
| A1-48 | medium | `-from`, `-limit`, `-n` parse base 0 (`010` is octal) | approve |
| A1-82 | medium | a read I/O error becomes ErrCorrupt or a skippable gap | approve |
| A1-85 | medium | Reader-created consumer files ignore the log's Perm | approve |
| A1-13 | medium | Archive returns ErrCorrupt, not ErrFutureVersion, for a newer destination | approve |
| A1-69 | medium | MaxBatchBytes is checked against a 34-byte-per-record estimate | document only |
| A1-2 | low | a case-variant duplicate key in quill.tail wins silently | fix the duplicate check; document the rest |
| A1-29 | low | Open refuses a symlinked Dir that OpenReader accepts | follow the symlink |
| A1-35 | low | one future-stamped segment pins age expiry of every newer one | warn and document |
| A1-50 | low | `-json` passes C1 control characters raw | approve |
| A1-52 | low | Read can exceed MaxReadBytes by one record | reword the docs |
| A1-57 | low | `stat -json` holes use capitalised keys and `null` | approve before release |
| A1-87 | low | the clock guard rounds up where the text says "at most 25%" | reword the text |
| A1-91 | low | a quill.policy of `{}` is acted on as retention 0 | approve, low priority |

## A1-67: the documented rewind of an ahead consumer, `quill commit`, cannot work (high)

**Claim.** A consumer whose offset is above Head+1 (after a restore that rolled the log back) fails to open with
ErrAhead, and `quill commit` opens the consumer before committing, so the remedy named in docs/userguide.md:98 and
:222-223, scope.md:325 and decision 53 fails with the same error. `quill check` prints HEALTHY and `quill consumers`
shows the consumer with lag 0 and no note.

**Why held.** The only fix that makes the documented remedy true changes behaviour: `commit` (and the library path it
uses) would have to open an ahead consumer, which decisions 36 and 53 currently refuse at open. The doc-only fix is not
held in principle, but it names a sequence (unregister, register, commit) that the owner should agree is the supported
one, because `unregister` drops the protect flag.

**Recommended answer.** Both. Now: correct the user guide to name the working sequence (`quill unregister`, `quill
register [-protect]`, `quill commit -offset N`) and say that unregister drops the protect flag. Approve: let `quill
commit` (an option or an internal open path used only by commit) open an ahead consumer and accept a rewind to any
offset in [first, Head+1]; and have `quill check` or `quill consumers` flag a consumer ahead of the head.

**Fix written.** None. The proposed fix is the one above; the reproduction (a 5-record log, a consumer committed at 8,
the log restored to head 5, `commit -offset 2` failing) is in the docs report.

## A1-80: a live protected consumer stops the writer from restarting (high)

**Claim.** `Config.ProtectedConsumers` makes Open, and `quill prune` (which feeds the policy back in), fail with
ErrLocked whenever a live consumer process holds that consumer's lock; the writer cannot be redeployed while its
protected consumer runs. `quill prune` also prints "a writer holds the log", naming the wrong lock. Reproduced on macOS
and, by the auditor, Linux ext4: a reopen takes 202 to 233 ms and fails.

**Why held.** The fix changes what Open does with `ProtectedConsumers` (decision 67: protect a lazy consumer before its
process starts) and how it interacts with the per-consumer lock (guarantee 5): Open would stop needing exclusive
ownership of an existing consumer file. It also changes the `prune` verb's Config.

**Recommended answer.** Approve. In Open, create the consumer file when missing without taking exclusive ownership of
an existing one; where the file exists and is parseable, accept it and set the protect flag only when it is absent or
off (under a short retry if the lock is held); make `quill prune` drop ProtectedConsumers from the policy-derived Config
(decision 99 already says offline verbs open without protected consumers); report ErrLocked with the right lock name.
The workaround today is to drop ProtectedConsumers from the Config after the first run.

**Fix written.** None. A regression test is proposed: reopen with a held consumer lock.

## A1-25: after a failed rotation the active segment can outgrow the 1.5 GiB limit (medium)

**Claim.** While rotation keeps failing (decision 60(d) keeps appending), nothing bounds the active segment, so it can
pass the 1.5 GiB limit that Open and readers enforce; the log then cannot be opened although every record was
acknowledged durable. Reproduced on macOS: a read-only log directory, 400 appends of 4 MiB, a 1,677,729,073-byte segment,
and Open then refuses it. The same overshoot is reachable with no fault when SegmentBytes is 1 GiB and FlushBytes or
MaxBufferBytes exceed 512 MiB (read, not run).

**Why held.** The fix refuses appends in a state decision 60(d) deliberately keeps accepting them, so it changes the
contract of Append and amends the decision. It also adds a Config validation.

**Recommended answer.** Approve a ceiling. Once a pending rotation error exists, refuse new records with that error when
the active segment plus buffer would exceed SegmentBytes + MaxBufferBytes + max(MaxBatchBytes, MaxPayloadBytes+34), and
always stay below `maxSegmentBytes`; validate that SegmentBytes plus the largest single-flush overshoot is under
`maxSegmentBytes`; make the SegmentError for an oversize but intact file say so instead of "damaged at byte 0 (intact to
byte 0)"; amend decision 60(d) and the user guide. Alternative: keep the behaviour and document that a supervisor must
watch `Err()`.

**Fix written.** None.

## A1-32: the reserved `_archive` consumer cannot be removed or rewound (medium)

**Claim.** The consumer `quill archive` creates is protected, and `register`, `unregister`, `commit` and the library's
`DeleteConsumer` all refuse its name ("a leading underscore is reserved"). It pins segments at or above its offset until the
next archive; after a restore that leaves it ahead of the head, `quill archive` fails with ErrAhead and the userguide's
remedy (`quill commit`) is refused for exactly this consumer. Deleting `consumers/_archive` by hand, or setting ConsumerTTL,
clears it; neither is documented for it.

**Why held.** It changes an exported function's behaviour and the CLI's accepted names, and decision 41 reserves the prefix
on purpose.

**Recommended answer.** Approve: let `DeleteConsumer`, `unregister`, `commit` and `register` accept exactly the reserved
archiver name (a second validator), or give `quill archive` a `-reset` option; document the remedy in the user guide.

**Fix written.** None.

## A1-47: extra positional arguments are silently ignored (medium)

**Claim.** Every verb except `append` ignores positional arguments, and Go's flag parsing stops at the first one, so
`quill -dir L read 100` prints from the first retained record and `quill -dir L tail 5 -json` prints text, both exit 0.

**Why held.** It turns previously successful (if misdirected) invocations into exit 2.

**Recommended answer.** Approve. After `fs.Parse`, if the verb does not take arguments and `len(fs.Args()) > 0`, return a
usage error ("unexpected argument ...; flags go before arguments"). `append` and `help` (which takes a verb name) are
exempt.

**Fix written.** None; the proposed change is one check in cmd/quill/verbs.go.

## A1-48: `-from`, `-limit` and `-n` parse numbers in base 0 (medium)

**Claim.** `-n 010` is 8 records, `-from 0x10` is 16 and `-limit 1_0` is 10, while `-offset`, `-max-bytes`,
`-min-free-bytes`, `-max-records` and `@unix` times are plain decimal (`010` is 10; `0x10` is refused).

**Why held.** A previously accepted spelling (hex, underscores, octal) changes meaning or is refused.

**Recommended answer.** Approve: register the three flags through a small `flag.Value` that calls `parseDecimal`, as the others
do. The silent octal reading of a zero-padded number is the dangerous part; hex is arguably a feature, but one rule across the
CLI is simpler to document.

**Fix written.** None.

## A1-82: a read I/O error is reported as ErrCorrupt or a skippable gap (medium)

**Claim.** In `scanSource`, any error from `walk`, frame errors and I/O errors alike, becomes ErrCorrupt (active segment, with
the cause flattened) or a `*GapError` over the whole segment (sealed). Decision 79 says an I/O error is returned as that error,
never as ErrCorrupt, and scope.md's resume idiom ("iterate again from To + 1") makes a consumer skip a whole segment after one
transient EIO.

**Why held.** It changes which error class the read paths return (ErrCorrupt and `*GapError` become the underlying I/O error),
which callers may match on.

**Recommended answer.** Approve: return non-frame errors unwrapped (wrapped with `%w` so `errors.Is(EIO)` works) for both
segment kinds; keep `*GapError` for real frame damage; add a Sim EIO-on-read test. The sealed-segment half is arguable under
decision 38, but the active-segment half contradicts decision 79 outright.

**Fix written.** None.

## A1-85: Reader-created consumer files ignore the log's Perm (medium)

**Claim.** `Reader.consumerStore` hard-codes 0o600 and `Archive` hard-codes 0o600; `ReaderOptions` has no Perm. In the group
arrangement scope.md describes, a consumer process in another account creates a file the writer cannot read, and retention
then treats the consumer as broken (every segment protected, `check` UNHEALTHY). A skeptic ran the cross-user case on the Linux
box with `Config{Perm: 0660}` and confirmed it.

**Why held.** It changes the modes of files created through a Reader and the CLI, and may add a `ReaderOptions` field and a
flag.

**Recommended answer.** Approve: derive the mode of Reader-created files from the log (the permission bits of its existing
files, inferred as the least-surprise source), or add `ReaderOptions.Perm` and a CLI flag; use the same for Archive; document it.

**Fix written.** None.

## A1-13: Archive returns ErrCorrupt, not ErrFutureVersion, for a newer destination format (medium)

**Claim.** The state-file load sequence is written six times (three for quill.format, three for quill.tail); Archive's copy wraps
every `decodeFormat` error as ErrCorrupt, so a destination whose quill.format is `quill/2` gives ErrCorrupt where Open and
OpenReader give ErrFutureVersion (decision 6, errors.go:16-17, format.md:137).

**Why held.** The fix changes the error class an exported function returns; the dedup into `loadFormat` and `loadTail` is net
zero lines.

**Recommended answer.** Approve; decision 6 already says ErrFutureVersion. If the owner prefers "any unusable destination
marker is ErrCorrupt" for Archive, keep archive.go's wrapping and merge only the load.go and reader.go pair.

**Fix written.** A prototype (the two helpers in state.go, six sites switched, +42/-42 lines) passed the full suite in a scratch
copy; it was not committed. A test that Archive into a `quill/2` destination matches ErrFutureVersion is proposed.

## A1-69: MaxBatchBytes is checked against a 34-byte-per-record estimate (medium)

**Claim.** The docs call MaxBatchBytes the largest encoded batch, but append.go:30 checks `34 + len(payload)` per record, so 500,000
one-byte records (about 7.5 MB encoded, under half of 16 MiB) are refused, and `quill append` from stdin (one batch) refuses
everything. Reproduced on macOS and Linux.

**Why held.** Making the check exact, or chunking stdin, changes behaviour and (for stdin) the all-or-nothing property of one
batch.

**Recommended answer.** Document it now: say the limit is checked against `len(payload)+34` per record (an upper bound on the
encoded size) and that `quill append` from stdin is subject to it (about 480,000 one-byte lines per call at the default). Leave
the check conservative; the disk-bound formula depends on the over-count. Chunking stdin is optional and gives up atomicity.

**Fix written.** None.

## A1-2: a case-variant duplicate key in quill.tail wins silently (low)

**Claim.** The slow path accepts a key such as `SEQ`, and `{"seq":1,"SEQ":2,...}` resolves to Seq 2 with no error, because
`checkDuplicateKeys` compares exact strings while `encoding/json` matches field names case-insensitively; it also accepts trailing
whitespace, reordered keys and a missing newline. format.md:103-105 says a duplicate key is refused.

**Why held.** The strictness of a state file is a guarantee-9 question, and decision 105 deliberately preserved the slow path's
verdicts; A1-1 (round-1-decided) would instead make the marker canonical-only.

**Recommended answer.** Make the duplicate check case-insensitive (the only part that contradicts "a duplicate key is refused") and
document in format.md which non-canonical shapes are tolerated (whitespace, order, a missing newline). Do not reopen decision 105's
canonical-only question unless taking A1-1.

**Fix written.** None.

## A1-29: Open refuses a symlinked Dir that OpenReader accepts (low)

**Claim.** `mkdirAll` uses Lstat, so `Open(Config{Dir: link})` returns ErrNoLog "is not a directory" for a symlink to a directory, while
`OpenReader(link)` and a symlinked parent both work. Nothing documents the writer-side restriction.

**Why held.** Either choice changes behaviour (follow the link, or keep refusing with a different error), and the refusal may be
deliberate hardening.

**Recommended answer.** Follow a symlinked final component of `Dir` (`os.Stat`), keeping O_NOFOLLOW for everything inside the log
directory, and make OpenReader and the CLI agree (a deployment that symlinks its data directory is common; the planted-link threat
needs write access to the parent, which SECURITY.md excludes, inferred). Otherwise keep the refusal, document it and say "Dir is a
symbolic link" in the error.

**Fix written.** None.

## A1-35: one future-stamped segment pins age expiry of every newer one (low)

**Claim.** The age sweep is a strict prefix rule (`if s.lastAt > cutoff { break }`), so a segment whose last record is stamped in the
future blocks age expiry of everything after it until the clock catches up; decision 44's guard covers only a clock that moves
forward. Nothing is logged. The caps still bound disk and `quill check` flags the record.

**Why held.** A fixed policy (judge such a segment by its seal time, or skip it) is a design choice; prefix deletion is deliberate.

**Recommended answer.** Minimum: log a warning when the sweep stops at a segment whose last stamp is more than a day in the future,
and document the hazard beside decision 44. Do not change the prefix rule.

**Fix written.** None.

## A1-50: `-json` passes C1 control characters raw (low)

**Claim.** `appendJSONString` copies valid UTF-8 above 0x7f except U+2028 and U+2029, so U+0080 to U+009F (for example the CSI
introducer U+009B) reach the terminal raw in `-json`, contradicting userguide.md:198-199 and scope.md:188; the text form escapes them. DEL
(0x7f) also passes raw.

**Why held.** It changes the bytes `-json` emits (decision 35 pins them).

**Recommended answer.** Approve: escape U+0080 to U+009F (and 0x7f) as `\u00XX`; the JSON stays valid and lossless; update the test that
compares with `encoding/json`. Whether a given terminal honours 8-bit C1 in UTF-8 mode was not tested; the documentation promise is broken
either way.

**Fix written.** None.

## A1-52: Read can exceed MaxReadBytes by one record (low)

**Claim.** The cap is checked after each record, so a read returns up to cap plus one record (at defaults about 16 MiB plus up to 64 MiB - 32),
contradicting "at most `limit` records and `MaxReadBytes` of payload, but always at least one" (read.go:12-14, scope.md). Reproduced on macOS.

**Why held.** Either stopping earlier (an observable change to batch sizes) or rewording a documented bound.

**Recommended answer.** Reword the three docs to "stops once at least MaxReadBytes have been returned"; the shorter wordings already read
that way, the overshoot is bounded by one record, and a hard cap needs the frame length before cloning. Stopping before a record that would
exceed the cap unless `count == 0` is the alternative.

**Fix written.** None.

## A1-57: `stat -json` holes use capitalised keys and `null` (low)

**Claim.** `quill stat -json` serialises a hole as `{"From":..,"To":..}` and an empty list as `null`, while every other CLI JSON key is
lowercase; no document fixes the stat, check or consumers schema.

**Why held.** The first public release freezes whatever scripts parse, so the shape is a decision, and changing it is a breaking change later.

**Recommended answer.** Approve before release: emit `[{"from":..,"to":..}]`, always an array, through a local tagged struct, and document the
stat, check and consumers JSON schemas in the reference.

**Fix written.** None.

## A1-87: the clock guard rounds up where the text says "at most 25%" (low)

**Claim.** `limit := (len(l.sealed) + 3) / 4` deletes the ceiling of a quarter (2 of 5, 4 of 15) where decision 44 and scope.md say "at most
25% (at least one)"; retention_test.go:266 pins the larger number.

**Why held.** Changing the code changes which segments a pass deletes and a test; changing the text amends a decision.

**Recommended answer.** Reword the text to "a quarter, rounded up, at least one" (the difference is at most three segments; the owner accepted
"25% per pass" without a rounding rule). The alternative is `max(1, n/4)` and a test change.

**Fix written.** None.

## A1-91: a quill.policy of `{}` is acted on as retention 0 (low)

**Claim.** `{}`, `null` and `{"unknown":1}` parse as a valid policy of zeros; `quill ls` shows RETENTION 0s and `quill prune` runs with the default
24 h and no caps. A version-skewed policy (a future writer renaming fields) would produce such a file, and decision 68's rationale is that offline
tools must not prune with default retention. v0.1.0 writers always write every field (an off retention is `-1ns`).

**Why held.** The tolerant parse is deliberate (decisions 29 and 48) and one skeptic voted REFUTED (code and spec agree); the fix adds a refusal.

**Recommended answer.** Approve, low priority: treat a policy with no `retention` (or no known field) as unparseable for the offline tools, and print
the retention as "unknown" in `ls`. Declining is defensible.

**Fix written.** None.

## Not held, but unresolved

**A1-62: `Config.Logger` is called with the log's mutex held.** A handler that calls back into the Log deadlocks it, and a blocked handler stalls
Append, Read, Sync and the flusher. A fix was written (commit fdcd880fe2e4, branch `stage2/A1-62`, not in this tree): records are queued under the lock
and delivered after it, on the caller's goroutine and, for the flusher's records, on a delivery goroutine that Close waits for. The fix-skeptic rejected
it: `logf` still calls the handler's `Enabled` method under `l.mu` (retention.go:294), so an `Enabled` that calls `l.Head()` deadlocks the log, and a handler
whose `Enabled` and `Handle` share a mutex stalls the flusher behind a blocked `Handle` (an Append and Sync loop did not finish in 8 s; the control finished in
0.38 s). The new documentation in Config.Logger, userguide.md:148 and scope.md promises more than the code gives. Recommended answer: finish the fix by not
calling `Enabled` under the lock (queue unconditionally and filter at delivery, or call `Enabled` in the deliverer) and add a test with a blocking `Enabled`; or
take the cheaper route and document that the handler runs under the log's lock and must neither call the Log nor block.

**A1-59: the Log's own reads return a raw ENOENT where the Reader returns a GapError.** Decided in round 1 (decision 103) and not held, but raised again
independently by two fresh auditors, with a real-concurrency reproduction (raw ENOENT in 25 of 1,601 reads on macOS and 17 of 1,483 on Linux). Recommended
answer: reopen the decision and take the one-line mapping at read.go:295, amending decision 103's sentence "a deleted one fails at its open as before", which
contradicts guarantee 14. See [a1-round2-correctness.md](a1-round2-correctness.md), item 16.

## Pass A4: dependency, supply-chain and release hygiene

The A4 pass (record: [supply-chain.md](supply-chain.md)) held back four items. Three of them are decisions or actions that only the
owner can take before a public release; one is a written fix that did not reach main. The recommended answers are the recorder's reading of
the auditor's proposed fix and the skeptics' notes, not decisions. Ranked by what the public flip needs.

| id | sev | one line | recommended |
|----|-----|----------|-------------|
| A4-5 | low | the release archives ship the Go runtime and standard library with no copy of its licence | add the Go licence text at the flip; reword decisions 22 and 15 |
| A4-4 | medium | a workspace path is still in the v0.1.0 tag and in earlier history (the tip is fixed) | decide squash or filter, or accept, before pushing |
| A4-3 | medium | the private reporting channel named in `SECURITY.md` exists only if the setting is enabled | enable private vulnerability reporting |
| A4-1 | medium | the Go toolchain pin is enforced by the gate only; the fix is written and not on main | land the fix, rebased, with a fix-skeptic |

## A4-5: the archives ship the Go runtime with no copy of its licence (low)

**Claim.** The release archives contain a statically linked Go runtime and standard library (BSD-3-Clause) and no copy of that licence or notice;
decision 22's reason for having no `THIRD-PARTY-NOTICES` ("no third-party code ships", decision 15) holds for vendored source but not for the linked
binaries. Two skeptic votes confirmed it; one read the licence's binary-redistribution clause and listed an existing archive.

**Why held.** The fix changes what ships in every archive and amends two recorded decisions (22 and 15), and whether a link satisfies the notice duty
is the owner's policy call. Decision 22 is explicitly a private-period rule that decision 54 supersedes when the `LICENSE` lands, so the right moment is
the flip.

**Recommended answer.** At the flip, add the Go licence text to each archive and to the repository (or a `THIRD-PARTY-NOTICES` file naming the Go
standard library and its licence) in the same change as quill's own `LICENSE`; copy it in `build/quill` beside the existing `LICENSE` copy; reword
decisions 22 and 15 to say that no third-party source is vendored while the standard library is linked in. Declining is defensible if the owner judges
the notice duty met by a link; record that choice in decision 22.

**Fix written.** None.

## A4-4: a workspace path remains in the v0.1.0 tag and in earlier history (medium; the tip is fixed)

**Claim.** `docs/audits/audit-correctness.md` named an absolute session path with a macOS user name and an internal parent-directory name. The tip no
longer does (main, commit 0b70e82, fix-skeptic FIXED-VERIFIED), but the text is still in the v0.1.0 tag tree and in every commit that contains the file
(75 commits by the auditor's count).

**Why held.** Removing it from history means rewriting history or moving to a new tag, and the audit may not move or create tags or push. The choice
also decides what the public repository's first commit looks like.

**Recommended answer.** Before the first push, choose one. (a) Publish a squashed or filtered history (for example `git filter-repo` replacing the
path), tag the result, and treat v0.1.0 as a private tag; this closes the exposure in the tag and the history, and the changelog line that mentions it
can then be reworded. (b) Accept the exposure: the path is not a secret and its parent-directory name is not on the forbidden-name list; the cost is a
user name and a directory name in public history. (c) Ship from a new tag on main and push only main plus that tag; this does not help, because main's
history still contains the path, so it is not an answer to the exposure and is listed only to rule it out. Recommendation: (a), done once, together with the
LICENSE commit. If (a) is taken, the filtered history must be re-checked with the same grep over every commit and the tag message.

**Fix written.** The tip edit (0b70e82). None for history.

## A4-3: the reporting channel in SECURITY.md needs a repository setting (medium)

**Claim.** `SECURITY.md` now (main, f2516d7) tells reporters to use the Security tab "Report a vulnerability" route (GitHub private vulnerability
reporting), with a fallback if the option is not shown, and says only the latest release is supported. The route exists only if the owner enables it;
the tree and the API the auditor had could not show whether it is enabled. The fix landed without a fix-skeptic and without a build or vet run (it is
documentation only), and the v0.1.0 tag still carries the old wording that pointed at `CODEOWNERS`.

**Why held.** Enabling a repository setting and choosing a contact are owner actions; the audit must not invent a mailbox or a key.

**Recommended answer.** Enable private vulnerability reporting before the flip and confirm that the Security tab shows "Report a vulnerability". If the
owner prefers a monitored role mailbox or a PGP key, name it in `SECURITY.md` in place of the fallback sentence. Release from a tag that contains the new
text (the v0.1.0 tag does not).

**Fix written.** f2516d7 on main (`SECURITY.md` and one changelog line).

## A4-1: the Go toolchain pin is enforced only by the gate (medium)

**Claim.** The `toolchain go1.27.3` line in `go.mod` is a pin only in `hack/gate.sh`, which exports `GOTOOLCHAIN` from it. CI installs the newest 1.27.x
(setup-go ignores the toolchain line) and `build/quill` uses whatever `go` is on PATH, although README.md:83-86 and CONTRIBUTING.md:40-42 call the go.mod
toolchain the one supported version. The comment at `ci.yml:65-66` ("the Go patch level comes from go.mod's toolchain line") is false now.

**Why held.** The fix is written but did not reach main: the fix workflow's merge check failed on a stray `go.mod` change (an audit probe, since restored by
c202882), and the commit is based on da2d638, before the A4-7 and A4-9 changes to `build/quill` and `ci.yml`. It has had no fix-skeptic. It changes what
the release build and every CI job run (the exact pinned Go instead of whatever is installed), which the owner should accept.

**Recommended answer.** Approve. Land f800bb7 (branch `stage2/A4-1-retry`) rebased onto main, then have a fix-skeptic attack it; confirm that CI still
passes with a workflow-level `GOTOOLCHAIN` on the new setup-go pin. If the owner prefers a floating patch in CI, drop the CI half, keep the `build/quill`
half, and reword README, CONTRIBUTING and the `ci.yml` comment to say that CI floats within the minor release.

**Fix written.** f800bb7: `build/quill` derives and exports `GOTOOLCHAIN` from go.mod's toolchain line as the gate does; `ci.yml` sets one workflow-level
`GOTOOLCHAIN` literal and corrects the comment; two tests (`toolchain_pin_test.go` and an edit to `build_script_test.go`) fail if the literal, the script
and go.mod drift apart (5 files, 159 insertions, 3 deletions).

## Not held, but worth a line at the flip

**A4-2 (refuted, 2 votes to 1).** `go 1.27` is the documented latest-toolchain policy, so importers need Go 1.27 or later. If the owner wants a lower
consumer floor for the first public release, that is a product decision: set `go 1.23` (or the measured minimum) and keep the toolchain line, then rerun
the gate (the `cmd/quill` tests already need Go 1.24 for `b.Loop`). Otherwise state the floor outright in README.md:86, CONTRIBUTING.md:42 and
`scope.md:94`.

**A4-6 (refuted, 2 votes to 0).** `build/quill` skips a missing `LICENSE` silently, which is intended in the private period. At the flip, consider making
the release form fail without one, in the same change that adds the `LICENSE`.
