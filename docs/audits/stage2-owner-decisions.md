# Stage 2 audit: decisions held for the owner

Date 2026-10-29, against HEAD 1da37fc; updated 2026-10-30 (pass A1h) with the owner's approvals and what became of each. The stage 2 audit (pass A1, round 2: adversarial audit in four
dimensions) applied every finding that is behaviour-preserving, or that fixes a clear defect with a regression
test, and held back every finding whose fix changes observable behaviour, an exported error class, an exit code,
a CLI grammar, an on-disk or output shape, or the wording of a guarantee. This file lists those held items, one
section each, with the claim, why it was held (what the fix would change), a recommended answer and whatever fix
is already written. The full items (grounds, warrant, backing, rebuttal, skeptic votes and reproductions) are in
the per-dimension reports: [correctness](a1-round2-correctness.md), [docs](a1-round2-docs.md) and
[dedup](a1-round2-dedup.md); the counts are in [a1-round2-findings.md](a1-round2-findings.md).

The recommended answers are the recorder's reading of the auditor's proposed fix and the skeptics' notes; they were
recommendations when written. Ranked by severity, then reach.

**Update 2026-10-30.** The owner approved the recommended answer for 17 of the items below (every A1 item in the table
except A1-67 and A1-80, plus A4-1) on 2026-10-30. Each section keeps its recommendation text as first written (its
"Fix written" paragraph is the state on 2026-10-29) and now carries an **Owner approval** and a **Disposition**
paragraph, which are current. Result: 17 fixed, each in its own commit. Sixteen were FIXED-VERIFIED by a fix-skeptic (109 lens
verdicts, all pass; A1-82 is the 16th, fixed in bb608d3 on its third attempt, 7 of 7 lenses). The 17th, A4-1, is fixed in this
commit for its `build/quill` half only (the CI half was refuted because the pinned setup-go reads the toolchain line), after
rejections on the audit record alone (three attempts, each on the no-new-gap lens only); no verdict is claimed here for this commit. 0 fix-failed.
A1-67, A1-80, A1-62 and A1-59 are not in the ledger this update was written
from: commits for them exist in the tree (ba7c650, b5ad955, 2978378 and 7ef431c, verified from `git log`), but their
sections and table rows are unchanged here. The per-item argument, reproduction and fix-skeptic outcome are in
[a1-round2-findings.md](a1-round2-findings.md), section "Pass A1h".

**Update 2026-10-31.** The owner approved the recommended answer for A1-67 and A1-80 as well. Their fixes (ba7c650,
b5ad955) were already on `main` without a recorded fix-skeptic outcome; an independent fix-skeptic per commit returned
FIXED-VERIFIED (7 of 7 lenses each), and their sections below now carry an **Owner approval** and a **Disposition**.
**Update 2026-11-02.** A1-62 and A1-59, whose commits (2978378 and 7ef431c) were also already on `main` with no recorded outcome, were integrated at the owner's request: an independent fix-skeptic per commit returned FIXED-VERIFIED, and their paragraphs under "Not held, but unresolved" below carry the disposition.

**Update 2026-10-31 (pass A5h).** The owner approved the recommended answer for the eleven held items of pass A5 (A5-10, A5-1, A5-36, A5-18, A5-22, A5-29, A5-3, A5-9, A5-23, A5-32 and A5-44; see section "Pass A5" below) on 2026-10-31. Ten are fixed, each in its own commit in the tree, and each was FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses each, 60 lens verdicts, all pass; reported by the skeptics). The eleventh, A5-1, was rejected once for a documentation defect and then fixed on a third attempt (99b140d, FIXED-VERIFIED for the approved scope; see its section). Each A5 section keeps its recommendation text as first written and now carries an **Owner approval** and a **Disposition** paragraph.

**Update 2026-11-02 (pass A3h).** Three items were approved by the owner on 2026-11-02 and fixed in one run (A3-10 needed a further round by hand): A3-10, A3-20
(both from pass A3, see "A3-10" and "A3-20" below) and A5-1f, the follow-up to A5-1 that the owner asked for (a dated note in
the A5-1 section, not a new item). No item was held for the owner in this pass, so no new held-item section is added.

| counted over the 3 items | |
|---|---|
| verdict | 3 CONFIRMED (A3-10 and A3-20 as voted 2 of 2 in pass A3; A5-1f as the owner's follow-up, not put to a vote) |
| disposition | 3 fixed: A3-20 (53a5d3b) and A5-1f (531fb35) each FIXED-VERIFIED by its own fix-skeptic (7 of 7 and 6 of 6 lenses); A3-10 landed in badb254, 18662b6 and ed518a2 after two fix-skeptic rejections, the last for two comment lines that were then fixed, with no skeptic re-run on the final commit (see its section); 0 fix-failed; 0 held; 0 deliberately not changed |
| skeptic votes on the items | none in this pass (the ledger carries no votes; the pass A3 votes for A3-10 and A3-20 are in [threat-model.md](threat-model.md)) |
| fix-skeptic lens verdicts | 13, all pass (7 for A3-20, 6 for A5-1f). A3-10: the last skeptic (reported) passed its test, behaviour and no-new-gap lenses and rejected on the old-rule statements in two `archive.go` comments; that is not counted as a FIXED-VERIFIED |
| reproduced by execution | A3-10 and A3-20: yes, in pass A3 (both skeptics, per threat-model.md). A5-1f: the defect was not separately reproduced; the fix-skeptic probed the fix on real files (macOS) and saw one `ERROR` line (reported) |

**Coverage (pass A3h).** This is a fix pass with no auditors: the ledger lists no auditor clean checks and no auditor that
failed to return, and nothing was audited afresh. What each fix-skeptic checked is in its Disposition below. Not covered:
Windows (the library does not support it); the Linux box was used for A3-20 (commit 53a5d3b, `-race` on the overlap, archive,
reader and check tests) and A5-1f (the new tests at the commit); for A3-10 the full gate, with fuzz, ran green on macOS and Linux at
18662b6 and again at ed518a2 (verified, my own runs). Open points the skeptics left: A3-20's fresh-Reader limit and two wording
nits, and four A3-10 points (each in its section).

**Update 2026-11-02 (pass A1i).** One item, A1-67r, the owner-approved follow-up to A1-67 (its residuals R1 and R2; see the
dated note under "Follow-ups (2026-11-02)" in the A1-67 section, a note and not a new item). The owner approved the recommended
answer for both on 2026-11-02. No item was held for the owner in this pass, so no new held-item section is added.

| counted over the 1 item | |
|---|---|
| verdict | 1 CONFIRMED (the owner's follow-up, not put to a vote); 0 PLAUSIBLE; 0 REFUTED |
| disposition | 1 fixed, in 5b5e185, FIXED-VERIFIED by a fix-skeptic (6 of 6 lenses pass); 0 fix-failed; 0 held; 0 deliberately not changed |
| skeptic votes on the item | none (the ledger carries no votes) |
| reproduced by execution | the defect itself was not separately reproduced in this pass (it is the residual recorded in the A1-67 section, which states it as inferred, not run). The fix-skeptic ran the fixed tree on real files on macOS and on the Linux box (ext4) and saw exactly one `WARN` line (reported) |

**Coverage (pass A1i).** This is a fix pass with no auditors: the ledger lists no auditor clean checks and no auditor that
failed to return, and nothing was audited afresh. What the fix-skeptic checked and found clean (reported): the new tests fail
without the fix (the warn-once and warn-per-consumer tests give "0 warnings", the offline-verb test fails when the CLI filter is
widened); the warning fires on a real log with one ahead, one at Head+1 and one corrupt consumer, and only for the ahead one; the
corrupt or foreign file never reaches the new loop because the consumer scan leaves it out; `gofmt`, `go vet` and
`go test -race` on macOS clean; golden bytes untouched; `go doc` differs only in the `Config.Logger` comment; no new exported
symbol, flag, file or config field; the added text names no internal product. Not covered: the skeptic did not run
`staticcheck` or `deadcode` from `hack/gate.sh`; the commit adds no function, so there is nothing for `deadcode` to flag (the
skeptic's reasoning, inferred). One wording nit the skeptic left: the user guide calls `first` and `-time` "always a valid target",
which repeats the owner record and is slightly strong under a racing writer (`SeekToTime` retries once on `ErrOffsetRange`).

**Update 2026-11-03 (pass L1h).** Ten items: the owner took the recommended answer on 2026-11-03 for L1-4, L1-5, L1-11, L1-12,
L1-16 and L1-26 (pass L1, section "Pass L1" below) and for L2-53 (pass L2, section "Pass L2"), and asked for the three
fixes the fix-skeptic had rejected (L1-3, L1-10, L1-21; section "Open after a rejected fix") to be finished. For L1-12 only
the wording fix is approved; the optional refusal is not implemented and stays recorded as an open optional follow-up. L1-51
is not in this batch: the owner took the recommendation to keep the 64 KiB stride, so no code changes (noted in its own
section, not counted below). Each section keeps its recommendation text as first written and now carries an **Owner
approval** and a **Disposition** paragraph, which are current. The full per-item argument is in
[verification-optimization.md](verification-optimization.md) (pass L1) and [mutation-hardening.md](mutation-hardening.md) (L2-53).

| counted over the 10 items | |
|---|---|
| verdict | 8 CONFIRMED, 2 PLAUSIBLE (L1-10 and L1-12, as filed in pass L1), 0 REFUTED. The L1h ledger labels all ten CONFIRMED; the ledger carries no votes, so the recorder kept the verdicts of the pass records (inferred: the ledger label is a default for approved items, not a re-vote) |
| disposition | 7 fixed: L1-21 (41297fb), L1-16 (923623c), L1-5 (867f4ed), L1-26 (c2cddb7), L1-11 (ce47243), L1-12 (be8cb15, wording only) and L2-53 (637ed42), all on `main`. 3 fix written and rejected by the fix-skeptic, not on `main`: L1-3 (47df71b), L1-4 (de3cfd4), L1-10 (0a77730), each with the stated remaining work below. 0 held. 0 deliberately not changed (the L1-12 optional refusal is an open optional follow-up, not a decision against it) |
| skeptic votes on the items | none in this pass. The pass L1 and L2 votes on these items total 22 (20 CONFIRMED, 2 REFUTED, the two being one each on L1-10 and L1-12; L1-10 and L1-12 had 3 votes, the other eight items 2), reported by the pass records and not re-counted here |
| fix-skeptic | 10 fixes attacked: 7 FIXED-VERIFIED (46 lens verdicts, all pass: 6 for L1-21, L1-16 and L1-12, 7 for L1-5, L1-26, L1-11 and L2-53), 3 rejected (L1-3, L1-4, L1-10). For the three rejections the ledger carries the objection, not the lens verdicts |
| reproduced by execution | 9 of the 10 items, in passes L1 and L2 by an auditor or a voter (L1-3, L1-4, L1-5, L1-11, L1-12, L1-16, L1-21, L1-26, L2-53; L1-4 and L1-5 by the auditor only). L1-10 was settled by reading the lines; a fix-skeptic reproduced the one fact it rests on (6 identical `Warn`s for 6 failing passes). This pass reproduced nothing afresh: the recorder ran nothing |

**Coverage (pass L1h).** This is a fix-and-record pass with no auditors: the ledger lists no auditor clean checks and no
auditor that failed to return, and nothing was audited afresh. What the seven passing fix-skeptics checked and found clean
(reported): each new test is a real pin (for L1-5, L1-11, L1-26 and L2-53 it is red when the fix is reverted; for the
wording-only L1-12 and L1-16 it fails under a mutation of the behaviour it pins and passes on the parent by design, and L1-21 is
comments and docs with no new test); the
changed text was checked line by line against the code; `gofmt`, `go vet` and `go test -race -count=1 ./...` were green on
the fixed tree for each of them; golden bytes are untouched in every one; `go doc -all .` differs only in comment text (and
for L1-5 not at all in signatures); no new exported symbol, flag, file or config field; the added text names no internal
product. Not covered: `staticcheck`, `deadcode` and the fuzz step of `hack/gate.sh` were not run by the fix-skeptics of L1-5,
L1-11, L1-12, L1-26 and L2-53 (the commits add no function or one used constant, so there is nothing for `deadcode` to flag:
the skeptics' reasoning, inferred). The Linux x86-64 box was not used by the fix-skeptics of L1-5, L1-11, L1-12 and L1-26 (the
harness script was missing at the stated path for L1-11 and L1-26); for L2-53 the new tests ran on linux/arm64 in an Apple
container. The L1-21 fix-skeptic did not re-run the auditor's counting-filesystem reproduction. The three rejected fixes are
on side branches and were checked by the recorder only by reading the lines their objections cite (verified, below). The
residuals the fix-skeptics left are listed under "Open after a rejected fix".

**Update 2026-11-03 (pass A3r).** Round-1 decisions that pass A3 had raised again were put to the owner, who took the
recommended answer for all of them on 2026-11-03 and approved two further items later the same day. The answers: three
behaviour changes (A3-3, A3-23 and, later, A3-24); eight decisions kept as decided in round 1 with nothing changed
(A3-4, A3-9, A3-11, A5-7, A1-1, A1-5, A1-30, A1-54); four kept with documentation (A3-29, A3-31, A3-32 and, later, A3-7,
which gets one clarifying sentence and no code change). Three things were not approved and are held: the A3-7 code change,
the A3-24 range check on the `time` field, and any change for A3-26. The section "Round-1 decisions confirmed 2026-11-03"
near the end of this file lists every item with the answer; the section "Held after pass A3r" lists the three held ones.
The per-item argument is in [threat-model.md](threat-model.md), which carries a dated paragraph under the seven items that
changed.

| counted over the 7 items fixed in this pass | |
|---|---|
| verdict | 7 CONFIRMED, as the ledger labels them (inferred: the label is a default for approved items; this pass held no vote). At audit time pass A3 had them as A3-3, A3-29, A3-31, A3-32 prior-deliberate, A3-23 prior-refuted, and A3-7 and A3-24 REFUTED by 2 of 3 votes each. 0 PLAUSIBLE, 0 REFUTED in this pass |
| disposition | 7 fixed, each on `main` in its own commit and each FIXED-VERIFIED by a fix-skeptic: A3-3 116726b (code), A3-23 18145b9 (code), A3-24 d15ac45 (code, the `-json` escape set only), A3-29 d916114 (documentation), A3-31 1cb2c93 (documentation), A3-32 d09e975 (documentation and one pinning test), A3-7 b765290 (documentation). 0 fix-failed. 0 held among the seven. 0 deliberately not changed among the seven. Besides the seven: 8 decisions kept unchanged and recorded here only, and 3 held (A3-7 code change, A3-24 `time` range check, A3-26) |
| skeptic votes on the items | none in this pass (the ledger carries no votes). The pass A3 votes are in [threat-model.md](threat-model.md): A3-7 and A3-24 had three each (one CONFIRMED, two REFUTED); the other five were round-1 items and were not voted |
| fix-skeptic lens verdicts | 45, all pass: 7 each for A3-3, A3-23 and A3-24; 6 each for A3-29, A3-31, A3-32 and A3-7 (reported by the skeptics) |
| reproduced by execution | all seven have a reproduction on record from pass A3 (the auditors' for A3-3, A3-23, A3-29, A3-31 and A3-32, the voters' for A3-7 and A3-24; reported). In this pass the fix-skeptics re-ran the behaviour on a built binary or a real log for A3-3, A3-23, A3-24 and A3-7 (macOS, and linux/arm64 in an alpine container; reported), and showed the A3-32 pinning test red under a mutation of the deletion loop. The recorder of this section ran nothing |

**Coverage (pass A3r).** This is a fix-and-record pass with no auditors: the ledger lists no auditor clean checks and no
auditor that failed to return, and nothing was audited afresh. What the fix-skeptics checked and found clean (reported):
for the three code fixes, the new tests fail without the fix and pass with it; `gofmt`, `go vet` and
`go test -race -count=1 ./...` are green on each fixed tree; golden bytes and testdata are untouched in all seven; `go doc -all`
differs only in comment text (A3-29, A3-31, A3-32) or not at all (A3-3, A3-23, A3-24, A3-7); no new exported symbol, flag, file
or config field; the added text names no internal product. For A3-3 every `Fprint*` and `errorf` call in `cmd/quill` was grepped
and disk-derived strings reach the output only through the wrapped sites; for A3-24 all non-surrogate code points were run
through the CLI before and after (exactly 170 changed from raw to escaped, decoded payloads identical). The skeptics of A3-3,
A3-23 and A3-24 also ran the pinned staticcheck v0.8.2 and deadcode v0.52.0, both clean. Not covered: the skeptics of A3-29,
A3-31, A3-32 and A3-7 did not run staticcheck or deadcode (those commits add no function beyond one test; the skeptics'
reasoning, inferred); the A3-23 skeptic ran vet, race tests, staticcheck and deadcode but not `hack/gate.sh` itself or its
fuzz step;
the A3-24 skeptic did not run the gate's freebsd-build and dual vet steps. The Linux x86-64 box harness was absent at the
stated path for the A3-3, A3-23, A3-24 and A3-7 skeptics, so their Linux evidence is linux/arm64 in an alpine container
(reported). The eight kept decisions were not re-examined: they are recorded from the earlier passes' text. Windows is
unsupported and not covered.

**Update 2026-11-03 (pass L3c).** The two items held from pass L3b, L3b-7 and L3b-8 (sections below, under "Pass L3b: live multi-process validation"), were answered by the owner on 2026-11-03, who took the recommended answer for both. L3b-7: document it, no code change; the runner-up of wrapping the error as `Archive` does is not approved. L3b-8: fix it in two steps, in the order recommended. Both are fixed on `main`, each in its own commit and each FIXED-VERIFIED by a fix-skeptic. Each section keeps its text as first written and now carries an **Owner's answer 2026-11-03** and a **Disposition** paragraph, which are current. No item is held for the owner from this pass any more, so no new held-item section is added: the two sections that already list the held items are the ones updated. The full item (Claim, Grounds, Warrant, Backing, Rebuttal, Verdict, reproduction) is in [live-integration.md](live-integration.md).

| counted over the 2 items | |
|---|---|
| verdict | 2 CONFIRMED (2 of 2 skeptic votes each in pass L3b, as the L3b record has them); 0 PLAUSIBLE; 0 REFUTED |
| disposition | 2 fixed: L3b-7 in `7fa0744` (documentation only, with one pinning test) and L3b-8 in `29f5c6c` (code, tests and documentation); each FIXED-VERIFIED; 0 fix-failed; 0 held; 0 deliberately not changed |
| skeptic votes on the items | 4 in pass L3b (2 each, all CONFIRMED); none in this pass |
| fix-skeptic lens verdicts | 13, all pass (6 for L3b-7, 7 for L3b-8; reported by the fix-skeptics) |
| reproduced by execution | L3b-7: by the auditor in pass L3b and again by its fix-skeptic on the Linux box (the fixed behaviour is the old behaviour, as documented). L3b-8: by a skeptic and the auditor in pass L3b; the fix-skeptic reproduced the symptom on the parent and the fix on a real tmpfs at zero free bytes |

**Coverage (pass L3c).** This is a fix-and-record pass with no auditors: the ledger lists no auditor clean checks and no auditor that failed to return, and nothing was audited afresh. What the fix-skeptics checked and found clean (reported): for L3b-7, the new test fails when the error is wrapped with `ErrLocked` (the rejected runner-up, applied as a mutation), the documented behaviour on the Linux box matches the text for `roll`, `recover` and `prune` as another uid and for root, golden bytes and testdata untouched, `gofmt`, `go vet` and `go test -race -count=1 ./...` clean, `go doc -all .` differs only in the added paragraphs, no internal product named. For L3b-8, 7 of the 8 new tests fail with the fix reverted and mutations of the clamp and its guard are caught, the full gate with `FUZZTIME=20s bash hack/gate.sh` exited 0 (gofmt, vet on linux and darwin, race tests, staticcheck, deadcode, fuzz, `go mod tidy`), the race tests passed on macOS and on the Linux box, golden bytes untouched, no new exported symbol, flag or config field, a live run on a real 8 MiB tmpfs at zero free bytes with the CLI verbs and the public API, and `hack/topology.sh run -fill` for 90 s on the final commit (ok, 725 writer restarts, 4 fills, no violations). Not covered: the L3b-7 fix-skeptic did not run `staticcheck`, `deadcode` or the gate for that commit (it changes comments and one test; inferred from the diff), did not run `append` as another uid (script slip) and did not run the unreadable-`quill.policy` `prune` case live (read from the code); the L3b-8 fix-skeptic ran only the pending-publish and zero-free-block cases on tmpfs, not the other filesystems of the L3b matrix (xfs, f2fs, overlay, btrfs), which is where the harness first hit L3b-8; the recorder ran nothing and read the two commits' diffs only. Open points the fix-skeptics left, none blocking, are in the two sections (L3b-7: the `docs/reference.md` and `verbs.go` help text; L3b-8: a rotation due at `Open` still frees nothing, as in a running writer, and four wording and coverage nits).

**Update 2026-11-04 (pre-tag batch, pass L4b).** Five items, recorded in the section "Pre-tag batch 2026-11-04" near the end of this
file. One is A1d-9, the item held since 2026-11-03: the owner took option 1 on 2026-11-04 (fix it, by wording that cannot go stale).
The other four, L4b-1 to L4b-4, are documentation items the wave 6 completeness critic raised; the owner said "go" for the batch on the
condition that the Linux box is used on tmpfs only. All five are documentation only: no code change, and no change to a CLI message,
an exit code or an exported API. Three things the critic raised that do need a code change or a message change are **not** approved
and are held, each with a recommended answer, in "Held after the pre-tag batch" at the end of this file. This update does not
say that every finding was fixed or that every fix was skeptic-validated; see the last paragraph below.

| counted over the 5 items | |
|---|---|
| verdict | 5 CONFIRMED (the ledger labels them so; A1d-9 was voted 2 of 2 in pass A1d, the other four carry no votes); 0 PLAUSIBLE; 0 REFUTED |
| disposition | 5 fixed, each in its own commit on `main`, each FIXED-VERIFIED by a fix-skeptic: A1d-9 `03b61eb`, L4b-1 `e8909b6`, L4b-2 `bdb1522`, L4b-3 `e51c2e3`, L4b-4 `be9adfb`; 0 fix-failed; 0 held; 0 deliberately not changed. Three further things from the critic are held (below) and are not counted as items of this batch |
| skeptic votes on the items | 0 in this batch (the ledger carries no votes); the 2 of 2 for A1d-9 are in [a1-diff-scoped.md](a1-diff-scoped.md) |
| fix-skeptic lens verdicts | 30, all pass (6 lenses each: test-fails-without-fix, root-cause, behaviour-preserved, no-new-gap, reachable, ladder), reported by the fix-skeptics. For a documentation-only commit the first lens reads "not applicable" and the skeptic checked the text against the code or a run instead; the outcomes record that |
| reproduced by execution | Something was run for each item, but only L4b-3's defect is a run that shows the old text wrong: **L4b-3** (CLI built from the tree: archive exit 3, then `check` HEALTHY exit 0, second archive exit 0; run by the fixer and again by the fix-skeptic). **A1d-9**: `Open` was run under a recording handler for nine cases (listed in its entry) to write the new rule; the finding itself was settled by reading in pass A1d. **L4b-1**: the fix-skeptic ran 7 CLI cases against builds of `v0.1.0` and the fixed tree; about 30 of the 38 table rows were not re-run. **L4b-2**: the fix-skeptic ran `quill -h`, `quill --help`, and 32-bit and big-endian builds. **L4b-4**: every count was recomputed by a command |

**Coverage (pre-tag batch).** This is a fix-and-record batch with no auditors: the ledger lists no auditor clean checks and no
auditor that returned nothing (both lists are empty, verified from the ledger given to the recorder), so no area is recorded as
audited afresh and "an auditor failed to return" is not claimed for any. The items came from the owner's approval (A1d-9) and from
the wave 6 completeness critic (L4b-1 to L4b-4; reported, the critic's own text is not in the tree). What the fix-skeptics checked
and found clean (reported): `gofmt` and `go vet` green on the fixed tree for all five; `go test -race -count=1 ./...` green for A1d-9, L4b-1,
L4b-3 and L4b-4 (the L4b-2 skeptic did not run it, because only a Markdown file changed); golden bytes and testdata untouched in all five; `go doc -all` identical, or differing only in
the `Config.Logger` and `Archive` comments; no new exported symbol, flag, file or config field; no internal product name in the added
text. Not covered: `staticcheck` and `deadcode` were not run by any of the five fix-skeptics (documentation and comments cannot
change them; the skeptics' reasoning, inferred). The Linux gate for this wave ran on **tmpfs only**; the ext4 filesystem matrix and
the power-cut replays are **deferred by the owner** (no power-off is possible) and were not run. L4b-1: the classification of 58 of the 144
CHANGELOG entries as "not user-visible" is inferred from each entry's own words, and about 30 table rows were not re-run. L4b-2: the
statement that the Linux gate ran on tmpfs only is reported, not re-derived. Windows is unsupported and not covered.

**What this record does not say.** Not every finding of the stage 2 audit was fixed, and not every fix was skeptic-validated. Fixes
finished by hand with no fix-skeptic re-run are recorded in this file: L1-3, L1-4, L1-10, A7-18, A7-19, the `ed518a2` comments, and
commits `badb254`, `0aa9659` and `8e45aa0`. Findings held with no fix written: the A3-7 code hardening, the A3-24 range check on the
`time` field and any change for A3-26 (see "Held after pass A3r"). The release record's "Known limitations and open items at the tag"
rolls these up, with what was not run.

| id | sev | one line | recommended | status 2026-10-30 |
|----|-----|----------|-------------|----|
| A1-67 | high | the documented `quill commit` rewind of an ahead consumer fails with ErrAhead | fix the doc now; approve letting `commit` rewind | approved 2026-10-31; fixed ba7c650 (FIXED-VERIFIED, 7 of 7) |
| A1-80 | high | a live protected consumer stops the writer from restarting | approve | approved 2026-10-31; fixed b5ad955 (FIXED-VERIFIED, 7 of 7) |
| A1-25 | medium | after a failed rotation the active segment can outgrow the 1.5 GiB limit | approve a ceiling | approved; fixed c18a27a (FIXED-VERIFIED, 7 of 7 lenses) |
| A1-32 | medium | the reserved `_archive` consumer cannot be removed or rewound | approve | approved; fixed a3955cd (FIXED-VERIFIED, 7 of 7) |
| A1-47 | medium | extra positional arguments are silently ignored | approve (exit 2) | approved; fixed 2c4e4f4 (FIXED-VERIFIED, 7 of 7) |
| A1-48 | medium | `-from`, `-limit`, `-n` parse base 0 (`010` is octal) | approve | approved; fixed b4fe935 (FIXED-VERIFIED, 7 of 7) |
| A1-82 | medium | a read I/O error becomes ErrCorrupt or a skippable gap | approve | approved; fixed bb608d3 (FIXED-VERIFIED, 7 of 7) |
| A1-85 | medium | Reader-created consumer files ignore the log's Perm | approve | approved; fixed 279d7ce (FIXED-VERIFIED, 7 of 7) |
| A1-13 | medium | Archive returns ErrCorrupt, not ErrFutureVersion, for a newer destination | approve | approved; fixed b529db5 (FIXED-VERIFIED, 7 of 7) |
| A1-69 | medium | MaxBatchBytes is checked against a 34-byte-per-record estimate | document only | approved; fixed 5b30e7f, documentation only (FIXED-VERIFIED, 6 of 6) |
| A1-2 | low | a case-variant duplicate key in quill.tail wins silently | fix the duplicate check; document the rest | approved; fixed 89957ae (FIXED-VERIFIED, 7 of 7) |
| A1-29 | low | Open refuses a symlinked Dir that OpenReader accepts | follow the symlink | approved; fixed 8c0c531 (FIXED-VERIFIED, 7 of 7) |
| A1-35 | low | one future-stamped segment pins age expiry of every newer one | warn and document | approved; fixed 19c0996 (FIXED-VERIFIED, 7 of 7) |
| A1-50 | low | `-json` passes C1 control characters raw | approve | approved; fixed fd52463 (FIXED-VERIFIED, 7 of 7) |
| A1-52 | low | Read can exceed MaxReadBytes by one record | reword the docs | approved; fixed 25e96c2, documentation only (FIXED-VERIFIED, 6 of 6) |
| A1-57 | low | `stat -json` holes use capitalised keys and `null` | approve before release | approved; fixed c91329c (FIXED-VERIFIED, 7 of 7) |
| A1-87 | low | the clock guard rounds up where the text says "at most 25%" | reword the text | approved; fixed 50201e1, documentation only (FIXED-VERIFIED, 6 of 6) |
| A1-91 | low | a quill.policy of `{}` is acted on as retention 0 | approve, low priority | approved; fixed 5c603fc (FIXED-VERIFIED, 7 of 7) |

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

**Owner approval.** Approved on 2026-10-31, as recommended.

**Disposition.** Fixed in ba7c650 (on main since 2026-10-30; verified from `git log`); fix-skeptic FIXED-VERIFIED
(7 of 7 lenses pass; reported, from a skeptic that ran everything on a scratch copy of the tree). A new
`ConsumerOptions.Rewind` opens a consumer ahead of the head; `quill commit` is the only caller that sets it, and
`Commit` still accepts only offsets in [first, Head+1]. `register`, the default open, `Reader.Consumer`, `Open` with
`ProtectedConsumers` and the archiver still return ErrAhead. `quill consumers` notes the consumer ("ahead of the head:
rewind it with quill commit") and `-json` adds `"ahead": true` only when true; `quill check` warns and stays
HEALTHY. Reproduction after (verified by the skeptic, built binaries): a 5-record log, a consumer committed at 8, the
log restored to head 5; `commit -offset 2` prints `billing 2`, the protect flag is kept, and `register` still refuses.
The same works for `_archive`. The recommended documentation step (name the unregister, register, commit sequence) is
superseded: with the code fix the guide names `quill commit -offset N` directly. Decisions 36 and 53, scope.md, the user
guide and CHANGELOG are amended. Known and unchanged: a consumer ahead of the head protects no segments, so a writer
running with caps can delete the rewind target before `commit` runs (the commit then fails cleanly with
ErrOffsetRange); and once the head grows past a stale ahead offset the consumer silently resumes mid-log and `check`
stops flagging it. Test coverage is thin on the `-json` key, first-1 and Head+2 on an ahead consumer and `commit -time`;
the skeptic checked those by hand. The user guide omits `-dir` and `-name` from the example command (cosmetic).

**Follow-ups (2026-11-02).** Test cleanup done in 7a8564e, no behaviour change: tests that a rewind accepts exactly first through Head+1, that `consumers -json` carries `ahead` only on the row of an ahead consumer, and that `commit -time` rewinds an ahead consumer (before every record gives first, after every record gives Head+1); each fails against a mutant of the code it pins. The user guide now types the whole remedy, `quill commit -dir <log> -name <consumer> -offset N`, and how to list the consumers that are ahead (checked verbatim on a built binary). Two residuals the fix-skeptic left are recommended here, not implemented. (R1) An ahead consumer protects no segments, so a writer with caps can delete the rewind target before `commit` runs, and the commit then fails cleanly with ErrOffsetRange. Recommended: no code change, since `first` is always a valid target at commit time; tell the operator to read `quill stat` just before committing or to rewind to `first` or by `-time`, in the restore section of the user guide (inferred from the commit's range check). (R2) Once the writer's head grows past a stale ahead offset, the consumer silently resumes mid-log and `check` stops flagging it, although after a restore those sequence numbers name different records; a restored log keeps its log ID, so an ID check cannot detect it. Recommended: the writer's `Open` logs one Warn per consumer that is ahead of the head (name, offset, head, "rewind it with quill commit"), and the restore procedure says to rewind consumers before restarting the writer; additive, no format change (inferred, not run).

**Follow-up A1-67r (2026-11-02), fixed in 5b5e185.** The owner approved the recommended answers for R1 and R2 on 2026-11-02;
the recommendation text above is kept as written. Verdict CONFIRMED (the owner's follow-up, not voted); kind degradation, severity
low. Fix-skeptic outcome FIXED-VERIFIED (6 of 6 lenses pass; reported, from a skeptic that ran the commit on a scratch copy).
What changed: (R2) the writer's `Open` logs one `Warn` through `Config.Logger` for each consumer whose offset is above Head+1,
with the message "consumer offset is ahead of the head; rewind it with quill commit" and the attributes `consumer`, `offset` and
`head` (`log.go:435-441`, a loop over the result of the existing `l.consumers()` read in `Open`, so there is no second pass over
`consumers/`; it runs before `Open` returns, with no lock held, like the other `Open` warnings, decision 50). A consumer file that
is unreadable or for another log never reaches it, because the scan leaves those out of the offsets. No change to what `Open`
accepts or refuses, to `Stats`, to any exported symbol, flag or file format. The offline verbs do not print it: `cmd/quill`
already filters their logger by message prefix (`lossOnly`), and that filter was not widened. (R1) Documentation only: the
restore section of the user guide now says to rewind consumers before restarting the writer, that an ahead consumer protects no
segments, and to read `quill stat` just before committing or to rewind to the first retained sequence (the `First` that `stat`
prints) or by `-time`. Decisions 36 (dated note) and 50, the scope.md `Logger` row and restore line, the user guide `Logger`
row, the `Config.Logger` doc comment and CHANGELOG ("Added:" under Unreleased) are amended.

Reproduction after the fix (verified by the skeptic, real filesystem, macOS and Linux ext4): a log with head 3 and three consumers,
`bill` at offset 9, one at Head+1, and one corrupt file; `Open` with a text-handler logger printed exactly one line,
`level=WARN msg="consumer offset is ahead of the head; rewind it with quill commit" consumer=bill offset=9 head=3`. The R1 and R2
defects themselves were recorded in the A1-67 disposition above as known and unchanged (the skeptic's own runs); they were not
reproduced afresh before the fix in this pass. Tests added (`ahead_warn_test.go`, `cmd/quill/ahead_test.go`): one warning with the right
attributes for a consumer ahead; none at Head+1 or behind; two for two ahead consumers; none from this warning for a corrupt or
foreign consumer file; none when `Config.Logger` is nil; and `TestOfflineVerbsDoNotPrintTheAheadConsumerWarning` for the CLI. The
fix-skeptic confirmed they fail without the fix (the negative guards pass without it, as they should).

Disposition: fixed in 5b5e185, FIXED-VERIFIED. Left as is: the residual that an ahead consumer protects no segments is still true
(R1 is documented, not coded, as approved), and the warning is given once, at the writer's `Open`; a writer that is already running
when the head passes a stale offset does not repeat it (inferred from the code: the loop is in `Open` only).

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

**Owner approval.** Approved on 2026-10-31, as recommended.

**Disposition.** Fixed in b5ad955 (on main since 2026-10-30; verified from `git log`); fix-skeptic FIXED-VERIFIED
(7 of 7 lenses pass; reported, from a skeptic that ran everything on a scratch copy of the tree). `Open` accepts an
existing consumer file without its lock when the file parses, is this log's, is already protected and is not ahead of
the head; every other state goes through the old locked open with the old errors (ErrCorrupt, ErrWrongLog, ErrAhead).
`openWriter` drops `ProtectedConsumers`, so `quill prune` and the other offline write verbs no longer fail beside a
running consumer and no longer name the wrong lock; protection read from the consumer files on disk is still honoured
(an offline `prune -retention 1s -force` over 8 segments deleted nothing with the consumer protected). Reproduction
before and after (verified by the skeptic): a writer reopen beside a live consumer process took 223 ms and failed
with ErrLocked before; after it takes 16 ms on macOS and 3.3 ms on Linux with no error. A stress run under `-race`
(1402 commits against 410 opens, 15 s) showed no torn read and the flag never cleared. Decisions 67 and 99, scope.md,
the user guide and CHANGELOG are amended. Known and as approved: a protected name whose file says unprotected, under
a live consumer, still fails `Open` with ErrLocked after the 200 ms retry; offline verbs no longer create a missing
protected consumer file (decision 99); the lock-free accept path has an inherent check-then-return window if a
consumer clears protection at that instant (inferred by the skeptic, not a regression).

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

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in c18a27a; fix-skeptic FIXED-VERIFIED (7 of 7 lenses pass). While rotation keeps failing, an
Append or AppendBatch that would push the active segment past SegmentBytes + max(FlushBytes, MaxBufferBytes) +
max(MaxBatchBytes, MaxPayloadBytes+34) is refused with the rotation error and takes no sequence number; Open refuses
(ErrConfig) a Config whose ceiling reaches the 1.5 GiB limit; a segment file over the limit is reported as "refused
without being read" by Open, Check and `quill check`, not as "damaged at byte 0". Decisions 16 and 60(d), the user
guide, format.md, scope.md and architecture.md are amended and CHANGELOG has a Fixed line. Reproduction before and
after (verified, macOS): a 1,677,729,073-byte segment and a refused Open before; 398 of 400 appends refused, an
8,388,644-byte segment and a successful reopen after. Not changed: docs/design/build-prompt.md keeps the historical
keep-appending wording (a build record); `quill check` still prints "damaged at byte 0" for a segment that is not a
regular file.

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

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in a3955cd; fix-skeptic FIXED-VERIFIED (7 of 7). One unexported validator accepts exactly
`_archive` on `Consumer` (Log and Reader), `DeleteConsumer`, and the CLI `register`, `unregister` and `commit`; every
other leading-underscore name is still refused, and so is `_archive` in `Config.ProtectedConsumers`. The built binary
rewinds an ahead `_archive` with `commit -name _archive -offset 1`, then `quill archive` succeeds. No `-reset` flag
was added. Decisions 16, 41 and 67, the user guide, format.md, scope.md, build-prompt.md and CHANGELOG are amended.
Consequence of the approved answer: `register -no-protect -name _archive` unprotects the archiver until the next
`quill archive`.

## A1-47: extra positional arguments are silently ignored (medium)

**Claim.** Every verb except `append` ignores positional arguments, and Go's flag parsing stops at the first one, so
`quill -dir L read 100` prints from the first retained record and `quill -dir L tail 5 -json` prints text, both exit 0.

**Why held.** It turns previously successful (if misdirected) invocations into exit 2.

**Recommended answer.** Approve. After `fs.Parse`, if the verb does not take arguments and `len(fs.Args()) > 0`, return a
usage error ("unexpected argument ...; flags go before arguments"). `append` and `help` (which takes a verb name) are
exempt.

**Fix written.** None; the proposed change is one check in cmd/quill/verbs.go.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 2c4e4f4; fix-skeptic FIXED-VERIFIED (7 of 7). A verb other than `append` and `help` given a
positional argument exits 2 with `unexpected argument "X"; flags go before arguments`, before anything is opened; the
exemption is one unexported flag on the verb table. Decision 53, the user guide and CHANGELOG are amended. Left as is,
inside the approved exemption: `help read x` ignores the second positional and `append x -sync=false` records the flag
as a line.

## A1-48: `-from`, `-limit` and `-n` parse numbers in base 0 (medium)

**Claim.** `-n 010` is 8 records, `-from 0x10` is 16 and `-limit 1_0` is 10, while `-offset`, `-max-bytes`,
`-min-free-bytes`, `-max-records` and `@unix` times are plain decimal (`010` is 10; `0x10` is refused).

**Why held.** A previously accepted spelling (hex, underscores, octal) changes meaning or is refused.

**Recommended answer.** Approve: register the three flags through a small `flag.Value` that calls `parseDecimal`, as the others
do. The silent octal reading of a zero-padded number is the dangerous part; hex is arguably a feature, but one rule across the
CLI is simpler to document.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in b4fe935; fix-skeptic FIXED-VERIFIED (7 of 7). The three flags take plain base-10 through a
small `decimalFlag` over `parseDecimal`: `010` is ten; `0x10`, `1_0`, `0b1`, `0o7`, signs and exponents exit 2 with
nothing on stdout. Decision 96, the user guide and CHANGELOG are amended. Cosmetic: `help` shows `value` where it
showed `uint`. The full gate could not complete `go mod tidy` in a git-archive copy (no repository there); every other
gate step passed.

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

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in bb608d3 (on main; verified from the commit log), the third attempt; fix-skeptic FIXED-VERIFIED
(7 of 7 lenses, all pass). The two earlier attempts were rejected: f6fcf26 and a5707a8
(the second attempt). a5707a8 had fixed `scanSource`, `ensureIndex`, `OffsetForTime` and the Reader's
active-segment probe, and was rejected for one path: the Reader's sealed first-frame probe still turned one EIO into a
`*GapError` over the whole segment (reproduced by that fix-skeptic against the changed tree: with EIO injected on reads
of sealed segment 2, `Reader.Read(21, 1)` returned `*quill.GapError: records 21 to 32 are unavailable`, and segment 1
gave "records 1 to 20"), and its test discarded the error. bb608d3 started from a5707a8 and closed both defects.

What the fix does (verified by reading the tree). An I/O error met while reading, in any segment and at any point,
including a sealed segment's first frame, is returned as itself (wrapped with `%w`, so `errors.Is(err, syscall.EIO)`
works), uncached, so the next call reads again. `Reader.segment` now returns `(*readerSegment, error)`: a non-frame
error from the first-frame probe is returned before the segment is cached (reader.go:246-247) and surfaces from
`view()`. Open failures (EMFILE, EACCES, not a regular file) stay a hole for one view (decision 94), and real frame
damage stays a `*GapError` or ErrCorrupt (`isFrameError`; `scanFailure`, read.go:276). `Open` is the second half:
`loadSealed` (load.go:449) now returns a read I/O error from `verifySegment` as a second result instead of recording
it in the segment's `bad` field, and `load()` fails `Open` with the first such error in segment order (load.go:321-330),
as it already did for the newest segment; damage and open failures stay holes (decision 98). Amended in place with the
note "Stage 2 audit, owner-approved 2026-10-30": decisions 38, 39, 79, 94 and 98; scope.md guarantee 14; user guide
section 3; the `GapError` doc comment (a run-on line wrapped); CHANGELOG Unreleased (Fixed). No exported symbol, flag
or file was added, and golden bytes are unchanged (verified by the fix-skeptic with `git diff --stat` over testdata).

Regression test (verified). `TestAReaderFirstFrameProbeIOErrorOfASealedSegmentIsReturnedAsThatError`
(read_ioerror_test.go:158) injects EIO on the reads of a sealed segment through the simulator, for segment 1 and for a
middle segment, and asserts through `wantReadEIO` (:48) that the read that hits the fault gives `errors.Is(EIO)` true,
`errors.As(*GapError)` false and `errors.Is(ErrCorrupt)` false, and that the same read succeeds once the fault clears.
The fix-skeptic reverted only errors.go, load.go, read.go, reader.go and seek.go to the parent versions and kept the
new tests: 9 tests failed, each for the claimed reason (the sealed-probe test with "records 1 to 20 are unavailable" for
the first segment and "records 21 to 32" for the middle one; the active-probe test with ErrCorrupt; the `Open` test with
"Open succeeded ... Read gave ... gap true"). The same tests pass on the fixed tree. The existing
`TestATransientScanErrorIsNotCachedByTheReader` was edited by design (reader_test.go:617): a scan-time open error is
now returned as itself, not as a gap.

Sites checked for a sibling that turns an I/O error into a gap or ErrCorrupt (verified by reading, by the fixer and again
by the fix-skeptic): reader.go `segment()` open path, `segment()` first-frame probe (fixed), `view()` active first-frame
probe (already raw), `view()` open of the active file (raw), `readerSegment.scan` (raw, not cached), the `ref.open`
closures in `view()` and `sealedRefsLocked` (ErrNotExist is a gap for retention, other errors raw); read.go `scanSource`
and `scanFailure`; seek.go `offsetForTime`; load.go `verifySegment`, `recoverSegment`, `loadSealed` (fixed) and the
newest-segment recovery in `load()`; log.go's Open creation path; iter.go (through `scanSource`); archive.go
`copySegment` and Archive's own `GapError`s (arithmetic on listed ranges, not I/O); stat.go `verifyBounded` (an I/O error
is reported by `Check` as a fatal segment problem carrying the error text, neither a gap nor ErrCorrupt); retention.go's
resume read (reads `g.To+1` after a real gap only). The two remaining gap-on-I/O cases are the ones the owner's answer
keeps: an open failure in `segment()` and ErrNotExist for retention.

Gate (verified by the fix-skeptic in an archive copy): gofmt clean, `go vet` and `go build` pass, `go test -race -count=1
./...` ok in every package, staticcheck v0.8.2 and deadcode v0.52.0 exit 0; the relevant tests also pass under `-race` on
the Linux x86-64 box. Not run by the fix-skeptic: the fuzz targets and `go mod tidy`. The fixer's own gate run is not in
the ledger, and this recorder re-ran nothing (docs only).

Residual objections, none against the claim (reported by the fix-skeptic; the first was reproduced): (1) Availability
trade-off. One EIO on a sealed segment's first frame now fails every call on that Reader that builds a view, not only the
read that needs the segment: a head read, `Reader.Stat` and `OffsetForTime` all fail while the fault lasts. A persistent
EIO on any sealed segment now makes `Open` fail where it used to record a hole, so retention cannot age that segment out
until the fault is fixed. This follows from decision 79 and from how the newest segment already behaved; the amended
decisions 94 and 98 and the CHANGELOG say that `Open` fails, but decision 94's "returned by the read that needs it"
understates the view-wide failure. (2) Decision 94 still carries the phrase "in an intermediate form, made a hole for one
view" (docs/design/decisions.md:102, verified by grep), which refers to an attempt that never shipped; it should be
deleted. This commit changes docs/audits only, so the phrase is left for the owner or the next fix. (3) A real EIO on an
`os.File` was not exercised, only the simulator's injection (inferred to behave the same).

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

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 279d7ce; fix-skeptic FIXED-VERIFIED (7 of 7). `OpenReader` reads the permission bits of
`quill.tail` once (the writer sets them to Config.Perm exactly), ORs in 0o600, and the Reader's consumer files and
everything `Archive` creates use that mode; default-mode logs still get 0600. This is the first of the two options
recommended; no `ReaderOptions` field or flag was added. Decisions 41, 51 and 67, scope.md, format.md, the user guide,
the doc comments and CHANGELOG are amended. Not shown: a real second account (the cross-user case could not be run
because creating a user on the shared box was refused); the effect is shown by file modes and the passing tests. A
Reader-created lock file follows the umask (disclosed in the docs).

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

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in b529db5; fix-skeptic FIXED-VERIFIED (7 of 7). `Archive` into a `quill/2` destination now
returns ErrFutureVersion; the three `quill.format` loads (load.go, reader.go, archive.go) go through one `loadFormat`
in state.go. The `quill.tail` half (`loadTail`) was not done. Decisions 6 and 85 and CHANGELOG are amended and the
Archive doc comment says so. Cosmetic: the ErrCorrupt text for a damaged destination marker no longer says
"destination".

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

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 5b30e7f; fix-skeptic FIXED-VERIFIED (6 of 6). Documentation only, as recommended:
MaxBatchBytes is described as checked against `len(payload)+34` per record (an upper bound of the encoded size) and
`quill append` from stdin as one batch subject to it; the check is unchanged. The figure written is "about 479,000"
one-byte records at the default, not the "about 480,000" in the recommendation: the fix-skeptic measured 479,349
accepted and 479,350 refused (verified). Decision 46, scope.md, the user guide, the Config and AppendBatch comments,
the `append` verb summary and docs/reference.md (regenerated) and CHANGELOG are amended.

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

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 89957ae; fix-skeptic FIXED-VERIFIED (7 of 7). The duplicate-key check compares keys by the
case folding `encoding/json` uses, so `{"seq":1,"SEQ":2,...}` is refused; format.md lists the non-canonical shapes the
strict parser still accepts (whitespace, order, a missing newline). Decision 105's canonical-only question was not
reopened. Decisions 72 and 105 and CHANGELOG are amended.

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

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 8c0c531; fix-skeptic FIXED-VERIFIED (7 of 7), after an earlier attempt that missed a link in
an ancestor of a Dir that does not exist yet. `Open` follows a symbolic link anywhere in Dir (an unexported
`resolveDirLink` resolves it once with `filepath.EvalSymlinks`); a link to a file or to nothing stays ErrNoLog and the
message says "symbolic link"; files inside the log are still opened with O_NOFOLLOW. Decisions 51 and 62, scope.md,
the user guide, SECURITY.md and CHANGELOG are amended. A link swapped after open is not re-checked (inferred to be
inside the threat SECURITY.md excludes).

## A1-35: one future-stamped segment pins age expiry of every newer one (low)

**Claim.** The age sweep is a strict prefix rule (`if s.lastAt > cutoff { break }`), so a segment whose last record is stamped in the
future blocks age expiry of everything after it until the clock catches up; decision 44's guard covers only a clock that moves
forward. Nothing is logged. The caps still bound disk and `quill check` flags the record.

**Why held.** A fixed policy (judge such a segment by its seal time, or skip it) is a design choice; prefix deletion is deliberate.

**Recommended answer.** Minimum: log a warning when the sweep stops at a segment whose last stamp is more than a day in the future,
and document the hazard beside decision 44. Do not change the prefix rule.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 19c0996; fix-skeptic FIXED-VERIFIED (7 of 7). The prefix rule is unchanged; when the sweep
stops at a sealed segment whose newest record is stamped more than 24 h ahead, the prune pass logs a warning (segment
id, last_at, kept count). Decision 44, scope.md, the user guide and CHANGELOG are amended. The CLI sets no logger, so
`quill prune` shows no warning (as with the existing clock-guard warning); `quill check` still reports the record.

## A1-50: `-json` passes C1 control characters raw (low)

**Claim.** `appendJSONString` copies valid UTF-8 above 0x7f except U+2028 and U+2029, so U+0080 to U+009F (for example the CSI
introducer U+009B) reach the terminal raw in `-json`, contradicting userguide.md:198-199 and scope.md:188; the text form escapes them. DEL
(0x7f) also passes raw.

**Why held.** It changes the bytes `-json` emits (decision 35 pins them).

**Recommended answer.** Approve: escape U+0080 to U+009F (and 0x7f) as `\u00XX`; the JSON stays valid and lossless; update the test that
compares with `encoding/json`. Whether a given terminal honours 8-bit C1 in UTF-8 mode was not tested; the documentation promise is broken
either way.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in fd52463; fix-skeptic FIXED-VERIFIED (7 of 7). `-json` writes DEL and U+0080 to U+009F as
`\u00XX` in `payload`; the JSON stays valid and lossless (round-tripped through `encoding/json`). Decision 35,
scope.md, the user guide and CHANGELOG are amended. Whether a terminal honours 8-bit C1 in UTF-8 mode was not tested,
as the recommendation said.

## A1-52: Read can exceed MaxReadBytes by one record (low)

**Claim.** The cap is checked after each record, so a read returns up to cap plus one record (at defaults about 16 MiB plus up to 64 MiB - 32),
contradicting "at most `limit` records and `MaxReadBytes` of payload, but always at least one" (read.go:12-14, scope.md). Reproduced on macOS.

**Why held.** Either stopping earlier (an observable change to batch sizes) or rewording a documented bound.

**Recommended answer.** Reword the three docs to "stops once at least MaxReadBytes have been returned"; the shorter wordings already read
that way, the overshoot is bounded by one record, and a hard cap needs the frame length before cloning. Stopping before a record that would
exceed the cap unless `count == 0` is the alternative.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 25e96c2; fix-skeptic FIXED-VERIFIED (6 of 6). Documentation only: Read, ReadWait, Records
and Follow are documented as stopping once at least MaxReadBytes of payload have been returned, overshooting by at
most one record; a test pins the rule. Doc comments, the user guide, scope.md, decisions 50 and 79 and CHANGELOG are
amended.

## A1-57: `stat -json` holes use capitalised keys and `null` (low)

**Claim.** `quill stat -json` serialises a hole as `{"From":..,"To":..}` and an empty list as `null`, while every other CLI JSON key is
lowercase; no document fixes the stat, check or consumers schema.

**Why held.** The first public release freezes whatever scripts parse, so the shape is a decision, and changing it is a breaking change later.

**Recommended answer.** Approve before release: emit `[{"from":..,"to":..}]`, always an array, through a local tagged struct, and document the
stat, check and consumers JSON schemas in the reference.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in c91329c; fix-skeptic FIXED-VERIFIED (7 of 7). `quill stat -json` writes holes as
`[{"from":N,"to":N}]`, always an array, through a function-local tagged type (`quill.Hole` gains no tags); the stat,
check and consumers JSON schemas are documented in scope.md and pointed to from the user guide. Decision 35 and
CHANGELOG are amended.

## A1-87: the clock guard rounds up where the text says "at most 25%" (low)

**Claim.** `limit := (len(l.sealed) + 3) / 4` deletes the ceiling of a quarter (2 of 5, 4 of 15) where decision 44 and scope.md say "at most
25% (at least one)"; retention_test.go:266 pins the larger number.

**Why held.** Changing the code changes which segments a pass deletes and a test; changing the text amends a decision.

**Recommended answer.** Reword the text to "a quarter, rounded up, at least one" (the difference is at most three segments; the owner accepted
"25% per pass" without a rounding rule). The alternative is `max(1, n/4)` and a test change.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 50201e1; fix-skeptic FIXED-VERIFIED (6 of 6). Documentation only: every statement now reads
"at most a quarter of the sealed segments, rounded up (at least one)"; the code and retention_test.go are unchanged.
Decision 44 and CHANGELOG are amended.

## A1-91: a quill.policy of `{}` is acted on as retention 0 (low)

**Claim.** `{}`, `null` and `{"unknown":1}` parse as a valid policy of zeros; `quill ls` shows RETENTION 0s and `quill prune` runs with the default
24 h and no caps. A version-skewed policy (a future writer renaming fields) would produce such a file, and decision 68's rationale is that offline
tools must not prune with default retention. v0.1.0 writers always write every field (an off retention is `-1ns`).

**Why held.** The tolerant parse is deliberate (decisions 29 and 48) and one skeptic voted REFUTED (code and spec agree); the fix adds a refusal.

**Recommended answer.** Approve, low priority: treat a policy with no `retention` (or no known field) as unparseable for the offline tools, and print
the retention as "unknown" in `ls`. Declining is defensible.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-30, as recommended.

**Disposition.** Fixed in 5c603fc; fix-skeptic FIXED-VERIFIED (7 of 7). `decodePolicy` fails when `retention` is
absent or empty, so `{}`, `null`, unknown fields only, or caps without retention are unparseable: `prune` refuses
without explicit flags, `ls` shows `?` and "policy unreadable" (the existing convention, not the word "unknown" the
recommendation used), `check` warns "cannot be parsed". Decisions 29, 48, 68 and 87, format.md, the `ReadPolicy`
comment and CHANGELOG are amended. Weakest point: a policy with `retention` present and every cap missing still parses
as caps of zero.

## Not held, but unresolved

**A1-62: `Config.Logger` is called with the log's mutex held.** A handler that calls back into the Log deadlocks it, and a blocked handler stalls
Append, Read, Sync and the flusher. A fix was written (commit fdcd880fe2e4, branch `stage2/A1-62`, not in this tree): records are queued under the lock
and delivered after it, on the caller's goroutine and, for the flusher's records, on a delivery goroutine that Close waits for. The fix-skeptic rejected
it: `logf` still calls the handler's `Enabled` method under `l.mu` (retention.go:294), so an `Enabled` that calls `l.Head()` deadlocks the log, and a handler
whose `Enabled` and `Handle` share a mutex stalls the flusher behind a blocked `Handle` (an Append and Sync loop did not finish in 8 s; the control finished in
0.38 s). The new documentation in Config.Logger, userguide.md:148 and scope.md promises more than the code gives. Recommended answer: finish the fix by not
calling `Enabled` under the lock (queue unconditionally and filter at delivery, or call `Enabled` in the deliverer) and add a test with a blocking `Enabled`; or
take the cheaper route and document that the handler runs under the log's lock and must neither call the Log nor block.

**Disposition (2026-11-02).** Integrated as the documented-constraint route, not the queued-delivery fix: 2978378 (on main since 2026-10-30; verified from `git log`) states in `Config.Logger`, the user guide, scope.md and decision 50 that the handler, `Enabled` included, runs synchronously under the log's lock and must neither call the `Log` nor block, and `logger_lock_test.go` pins it. Fix-skeptic FIXED-VERIFIED as a documented-constraint resolution (reported; it ran everything on scratch copies). It found the rule untrue of `Open`'s own three log lines (`log.go:397`, the policy write at `log.go:745` and `751`, and the cut-segment warning at `load.go:420`), which run on the caller's goroutine before there is a `Log`, with no lock held; harmless, since the handler has no `Log` to call, but the docs said otherwise. The docs now say so and add the unstated lock-order rule (no lock that a caller of the `Log` may hold) and that a handler writing straight to a pipe can block (d7b51e0), and the lock-held test was made sticky because a mutant that released the lock for every record after the first survived the old probe (the new one fails it; verified). The original hazards remain by design and are documented: a handler whose `Handle` or `Enabled` calls `l.Head()` hangs the writer (the skeptic: a 3 s timeout), and a blocking `Handle` stalls `Head()` and `Append` for over a second. The skeptic's judgement, inferred, is that this is acceptable for a release if the docs are accurate, as Go libraries commonly document callbacks under a lock. The alternative, finishing the queued-delivery fix (queue unconditionally and filter at delivery, no `Enabled` under the lock; first attempt fdcd880 on branch `stage2/A1-62`), is a larger concurrency change and remains available if the owner wants the hazard removed rather than documented.

**A1-59: the Log's own reads return a raw ENOENT where the Reader returns a GapError.** Decided in round 1 (decision 103) and not held, but raised again
independently by two fresh auditors, with a real-concurrency reproduction (raw ENOENT in 25 of 1,601 reads on macOS and 17 of 1,483 on Linux). Recommended
answer: reopen the decision and take the one-line mapping at read.go:295, amending decision 103's sentence "a deleted one fails at its open as before", which
contradicts guarantee 14. See [a1-round2-correctness.md](a1-round2-correctness.md), item 16.

**Disposition (2026-11-02).** Fixed in 7ef431c (on main since 2026-10-30; verified from `git log`) with decisions 101 and 103 amended; fix-skeptic FIXED-VERIFIED on 7ef431c and on main (reported; macOS and the Linux box). The sealed ref's open maps `fs.ErrNotExist` to `GapError{first, last}` of that segment and returns every other error as it was. Verified by the skeptic with a writer (4 KiB segments, a 12 KiB cap) and 8 readers running `Read`, `Scan`, `ScanAll`, `Records` and `OffsetForTime` for 20 s, with a 10 ms sleep in the scan callback to widen the window (without it the base never failed): raw `*fs.PathError` results on the parent were 150 of 2,849 reads on macOS and 360 of 204,201 on Linux, and 0 on 7ef431c and on main on both. Against a `Reader` on the same directory after deleting a middle segment, every offset from 1 to 100 gave 0 differences on 7ef431c and main and 126 on the base; a read spanning the hole returns the earlier records and then the gap, never skipping it. Every Log read path goes through the one mapped open (`Read`, `ReadWait`, `Scan`, `ScanAll`, `Records`, `Follow`, `Consumer.Read` and `Scan`, `OffsetForTime`, `SeekToTime`). EACCES, EIO and EMFILE stay themselves, and an ENOENT on the active segment or a consumer file is not turned into a gap. One test-strength gap, closed in fe9fe31: no test showed that a non-"not found" open error stays raw, so a mutant mapping every open error to a gap passed the suite; EIO and EACCES on the open of a sealed segment now must come back wrapping themselves (the test fails on that mutant; verified). The round-1 decision 103 sentence "a deleted one fails at its open as before" survives only in the audit history.

## Pass A4: dependency, supply-chain and release hygiene

The A4 pass (record: [supply-chain.md](supply-chain.md)) held back four items. Three of them are decisions or actions that only the
owner can take before a public release; the fourth (A4-1) was a written fix that did not reach main as first written, and has since
landed in this commit as the `build/quill` half only (the CI half was refuted). The recommended answers are the recorder's reading of
the auditor's proposed fix and the skeptics' notes, not decisions. Ranked by what the public flip needs.

| id | sev | one line | recommended | status 2026-10-30 |
|----|-----|----------|-------------|----|
| A4-5 | low | the release archives ship the Go runtime and standard library with no copy of its licence | add the Go licence text at the flip; reword decisions 22 and 15 | not in the A1h ledger; unchanged |
| A4-4 | medium | a workspace path is still in the v0.1.0 tag and in earlier history (the tip is fixed) | decide squash or filter, or accept, before pushing | not in the A1h ledger; unchanged |
| A4-3 | medium | the private reporting channel named in `SECURITY.md` exists only if the setting is enabled | enable private vulnerability reporting | not in the A1h ledger; unchanged |
| A4-1 | medium | the Go toolchain pin is enforced by the gate only; the `build/quill` half is fixed in this commit, the CI half was refuted | land the fix, rebased, with a fix-skeptic | approved; fixed in this commit (`build/quill` half only; the CI half was refuted because the pinned setup-go reads the toolchain line) |

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

The "latest release only" support policy was introduced by f2516d7 and appears in no earlier document; confirm it or change it. The changelog line
(which ships in the release archives) no longer carries the enable-the-setting action; that action lives only here.

**Fix written.** f2516d7 on main (`SECURITY.md` and one changelog line), and a follow-up that removes the owner action from the changelog.

## A4-1: the Go toolchain pin is enforced only by the gate (medium)

**Claim.** The `toolchain go1.27.3` line in `go.mod` is a pin only in `hack/gate.sh`, which exports `GOTOOLCHAIN` from it. CI installs the newest 1.27.x
(setup-go ignores the toolchain line; **this half is REFUTED at the pinned v7.0.0, see the Correction below**) and `build/quill` uses whatever `go` is on PATH, although README.md:83-86 and CONTRIBUTING.md:40-42 call the go.mod
toolchain the one supported version. The comment at `ci.yml:65-66` ("the Go patch level comes from go.mod's toolchain line") was said to be false; it is true at the pinned setup-go (see the Correction below).

**Why held (as first written, 2026-10-29).** The fix was written but did not reach main: the fix workflow's merge check failed on a stray `go.mod` change (an audit probe, since restored by
c202882), and the commit is based on da2d638, before the A4-7 and A4-9 changes to `build/quill` and `ci.yml`. It has had no fix-skeptic. It changes what
the release build and every CI job run (the exact pinned Go instead of whatever is installed), which the owner should accept.

**Recommended answer.** Approve. Land f800bb7 (branch `stage2/A4-1-retry`) rebased onto main, then have a fix-skeptic attack it; confirm that CI still
passes with a workflow-level `GOTOOLCHAIN` on the new setup-go pin. If the owner prefers a floating patch in CI, drop the CI half, keep the `build/quill`
half, and reword README, CONTRIBUTING and the `ci.yml` comment to say that CI floats within the minor release.

**Fix written.** f800bb7: `build/quill` derives and exports `GOTOOLCHAIN` from go.mod's toolchain line as the gate does; `ci.yml` sets one workflow-level
`GOTOOLCHAIN` literal and corrects the comment; two tests (`toolchain_pin_test.go` and an edit to `build_script_test.go`) fail if the literal, the script
and go.mod drift apart (5 files, 159 insertions, 3 deletions).

**Owner approval.** Approved on 2026-10-30, as recommended (with its stated alternative of keeping only the
`build/quill` half).

**Disposition.** Fixed in this commit, for the `build/quill` half only. The recommended fix f800bb7 was superseded:
`build/quill` now derives and exports `GOTOOLCHAIN` from go.mod's toolchain line, as `hack/gate.sh` does, so the release
binaries and the SBOM come from the pinned compiler; `TestBuildScriptPinsToolchain` fails without it. The CI half of the
claim was refuted (see the correction below): the workflow pins setup-go v7.0.0 (`ci.yml` lines 24, 44, 54, 75 and 87),
which reads the toolchain line, so the CI half was dropped and the `ci.yml` comment about the patch level is true. The
first re-fix attempt (41c8aa4) carried the false CI claim and was not used.

**Correction after the fix-skeptic (2026-10-30).** The CI half of the claim and of f800bb7 is false at the pinned action. actions/setup-go at b7ad1dad
(v7.0.0), `src/installer.ts` `parseGoVersionFile` (read from the action's repository at that commit), reads go.mod's `toolchain` line first, whenever
`GOTOOLCHAIN` is not `local`, and uses the `go` line only as a fallback; so CI already installs exactly the toolchain `go.mod` names. `src/main.ts`
then exports `GOTOOLCHAIN=local` to later steps through the runner env file, which would override a workflow-level literal (inferred from the source,
not run on a runner). The literal and its test were therefore dropped, and so was the "ignores the toolchain line" wording. What landed is the
`build/quill` half (with `TestBuildScriptPinsToolchain`, which fails without it), the corrected `ci.yml` comment and the README, CONTRIBUTING, decision 88
and changelog statements. The owner's alternative (CI floats within the minor release) is not needed.

## Not held, but worth a line at the flip

**A4-2 (refuted, 2 votes to 1).** `go 1.27` is the documented latest-toolchain policy, so importers need Go 1.27 or later. If the owner wants a lower
consumer floor for the first public release, that is a product decision: set `go 1.23` (or the measured minimum) and keep the toolchain line, then rerun
the gate (the `cmd/quill` tests already need Go 1.24 for `b.Loop`). Otherwise state the floor outright in README.md:86, CONTRIBUTING.md:42 and
`scope.md:94`.

**A4-6 (refuted, 2 votes to 0).** `build/quill` skips a missing `LICENSE` silently, which is intended in the private period. At the flip, consider making
the release form fail without one, in the same change that adds the `LICENSE`.

## Pass A3: security review and threat model

Date 2026-10-31. Pass A3 (record: [threat-model.md](threat-model.md)) applied eleven fixes, each re-attacked by a fix-skeptic
(11 of 11 FIXED-VERIFIED), and held back four items whose fix would change a documented file mode, a CLI behaviour or a
documented performance statement: A3-5, A3-14, A3-18 and A3-19. None of the four has a fix written; the recommended answers
below are the recorder's reading of the auditors' proposals and the skeptics' notes, and are recommendations, not
decisions. Two further items had a fix that was written and rejected by its fix-skeptic (A3-10 and A3-20); they are
listed after the four, with the owner's approval of 2026-11-02 and what became of each (both fixed; A3-10 only after a
third round, see its section). Items the auditors raised that the project had
already decided in round 1, or that a vote refuted, are not held; they are listed last with the proposal on file.

| id | sev | one line | recommended | status |
|----|-----|----------|-------------|--------|
| A3-5 | medium | a reader-only process can flock `quill.lock` when Perm grants read, blocking writer Open | approve 0600 for the writer lock; settle Archive's probe first | approved 2026-10-31; fixed 47315f1 (FIXED-VERIFIED, 7 of 7) |
| A3-14 | medium | `quill archive -to` writes into a directory that has no `quill.format` | approve: require an empty directory | approved 2026-10-31; fixed 303b794 (FIXED-VERIFIED, 7 of 7) |
| A3-18 | low | a Reader's read cost scales with the log, not the request | reword now; cache sealed refs after the release | approved 2026-10-31; reworded 3c36fc0 (wording only); the cache is deferred |
| A3-19 | low | the first-frame probe reads and CRC-checks the whole first frame (up to 64 MiB) | correct decision 47's wording; leave the code | approved 2026-10-31; reworded 3c36fc0 (wording only); code unchanged |

## A3-5: the writer's lock file is readable by readers, so a reader can hold it (medium)

**Claim.** `quill.lock` is created with Config.Perm minus umask. When Perm grants group or other read, a process that can only
read can open the lock `O_RDONLY` and take `LOCK_EX` with flock, which makes every writer Open (a restart included) fail with
ErrLocked until the holder exits. Verified on Linux with two uids (uid 1000 writer, `nobody` holder); with Perm 0600 the
holder gets "permission denied". Consumer locks (`consumers/<name>.lock`) are created from the log mode too and are subject to
the same hold.

**Why held.** The fix changes a documented file mode. Decision 51 says a lock file "is created through the process umask",
deliberately, and the Perm row of scope.md calls a shared mode a supported configuration. The fix would make the writer lock
owner-only whatever Perm says, and so changes what an operator sees in the directory and what a different-uid tool can do with
the lock (see the Archive caveat below).

**Recommended answer.** Approve. Create and keep `quill.lock` at 0600 regardless of Config.Perm (only the writer and Archive's
destination probe touch it); record it in decision 51 and in the user guide's group-arrangement text; leave consumer locks
group-accessible, because a group consumer must be able to take its own lock, and state in decision 41 that a reader-only
process with read access can therefore hold a consumer's lock and stall that consumer (not the writer).

Caveat on the auditor's proposal (inferred, not run): the proposal changes Archive's destination probe (archive.go:116) to open
`O_RDONLY` so a non-owner archiver can still detect a live writer. A 0600 file owned by another uid cannot be opened `O_RDONLY`
either, so a different-uid archiver would get a permission error on the probe. Decide whether that error is treated as "a writer
may be live, refuse" (fail closed, recommended) and say so in the Archive doc. Test: a log created by one uid with Perm 0644,
a probe from another uid, before and after.

**Fix written.** None. The proposed change is in threat-model.md, A3-5.

**Owner approval.** Approved on 2026-10-31, as recommended.

**Disposition.** Fixed in 47315f1 (on main; verified from `git log`); fix-skeptic FIXED-VERIFIED (7 of 7 lenses pass; reported, from a skeptic that ran everything on scratch copies, on macOS and on the Linux box). `quill.lock` is always created 0600 whatever `Config.Perm` says; a wider lock left by an earlier build is tightened (fstat, then fchmod) before the flock is tried. A failed chmod does not fail `Open`: it logs a Warn on `Config.Logger`, because the held lock is what keeps the log single-writer and refusing would turn a hardening step into an outage (the fixer's choice where the recommendation was silent). Archive's destination probe fails closed: a permission error opening the destination's lock returns an error that is both `ErrLocked` and `fs.ErrPermission`. Consumer locks keep the log's mode, and decision 41 states that a reader-only process with read access can hold a consumer's lock and stall that consumer, never the writer. Decisions 41 and 51, scope.md, format.md, the user guide and CHANGELOG are amended. Reproduction before and after (verified by the fixer, Linux, uid 1000 against `nobody`): the lock was 0644 and `nobody` held it, so the owner's `Open` failed with a locked error; after the fix the lock is 0600, `nobody` gets "Permission denied" and the owner's `Open` succeeds. Gate green on macOS and Linux on the fix commit. Not observed: the Archive probe from a second account on an account-owned lock; the skeptic ran a root-owned 0600 destination lock on the Linux box (refused, both error classes satisfied, nothing written) and the fixer a mode-0000 lock. Two minor gaps, neither a regression: the CLI's offline verbs filter log output down to two messages, so the new "could not be tightened" Warn reaches only library users who set `Config.Logger` (A5-36 later widened that filter with other messages, not this one; the owner asked for it on 2026-11-01 and its prefix was added to `lossOrShortfall` in cmd/quill/write.go in 84efe58, with a test red before and green after, the gate green on macOS and Linux, and no separate fix-skeptic); and the Archive probe reports every permission error as "a writer may be live", including an unsearchable destination directory (fails closed, wording only).

**Follow-up (2026-11-02).** Wording: a destination directory that cannot be searched is no longer reported as "a writer may be live"; Archive refuses it with a permission error that names the directory and is not `ErrLocked`, which is kept for a `quill.lock` that can be seen but not opened (e293016, with a test that fails on the old code and that is skipped as root).

## A3-14: `archive -to` writes into a directory that is not a log (medium)

**Claim.** `archiveDest` (archive.go:108 now) checks only same-file as the source, a held writer lock and the format id when
`quill.format` exists; when it is absent it writes one. `copySegment` then renames over `NNNNNNNNNN.log` whenever the existing
file's size differs (archive.go:178 now). A mistyped `-to` that names an existing directory without `quill.format` therefore
replaces an unrelated file named like a segment and adds `quill.format`, `quill.tail` and `consumers/`. Writable Open refuses the
same directory ("holds files but no quill.format", log.go:508 now). Reproduced on macOS with the built binary and through
`quill.Archive`: a 24-byte text file named `0000000000.log` became a segment copy and the command exited 0.

**Why held.** It makes a previously working invocation fail (archiving into a populated directory that is not yet a log), and it
changes the error an operator sees (ErrNoLog from Archive), which is an exported error class on that function.

**Recommended answer.** Approve. When `quill.format` is absent in the destination, require the directory to be empty, ignoring
`.tmp-*` entries and `quill.lock`, as `Log.create()` does, and otherwise return ErrNoLog; add a library test and a CLI test
that plant an unrelated `0000000000.log` and assert it is untouched; amend decision 85. Archive writes `quill.format` first, so a
destination left by an interrupted earlier run already has one and is unaffected (inferred from archive.go, not run).

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended.

**Disposition.** Fixed in 303b794 (on main; verified from `git log`); fix-skeptic FIXED-VERIFIED (7 of 7 lenses pass; reported). When `quill.format` is absent the destination must be empty, ignoring `.tmp-*` files and `quill.lock`, and otherwise `Archive` returns an error wrapping `ErrNoLog` before writing anything; the CLI exits 1. Reproduction before and after (verified by the skeptic with the built CLI): a destination holding an unrelated `0000000000.log` and `notes.txt` was changed (the log replaced, `quill.format`, `quill.tail`, `consumers/` added, exit 0) before the fix; after it exits 1 naming the directory and every file and directory is byte-identical. A destination left by an interrupted run already has `quill.format` and is unaffected (tested at every successive write of the destination, and by deleting trailing files). Decision 85, the `Archive` doc comment, the user guide and CHANGELOG say plainly that archiving into a populated directory that is not yet a log now fails. Gate green on macOS and Linux on the fix commit. Two minor points, neither caused by the fix: the message says "holds files" even when the destination holds only a subdirectory and does not say that an empty directory is the remedy; and a refused run still touches the source's protected `_archive` consumer, because `archive()` opens it before `archiveDest` (true of the other destination refusals too).

**Follow-up (2026-11-02).** Wording: the refusal now says that the destination is not empty and has no `quill.format` and that Archive needs a new or empty directory or an existing archive, instead of "holds files" (also true for a subdirectory only); still `ErrNoLog`, exit 1 (e293016, with a test that fails on the old message).

## A3-18: a Reader's read cost scales with the log, not the request (low)

**Claim.** A `Reader.Read`, `ReadWait`, `Scan` or `Consumer.Read` without an iterator cursor re-walks and CRC-checks the active
segment from byte 0 to the requested offset (read.go:203 now), and each call rebuilds a per-segment view in O(sealed segments)
(reader.go:334, a `ReadDir` plus a `segRef` and two closures per segment). Measured on the Linux box with 3,000,000 records in
one 343 MB active segment: `Reader.Read(2,999,000,1)` 379 ms against 112 us for the writer's `Log.Read`; 100 Reads of 1,000
records from 2,000,000 took 26.3 s against 0.33 s for 100,000 records through Records. One skeptic's runs at 5,000 to 40,000
one-record segments grew linearly (5.4 to 51 ms, 2.4 to 16 MB per call). This contradicts scope.md's "Read cost does not grow
with backlog" for the Reader's active segment.

**Why held.** Either fix changes a documented statement or a design choice. Decision 63 chose "the active segment is never
cached" on purpose, for correctness after a writer crash, and decision 103 gives the Log, not the Reader, a sealed-ref cache.
Caching in the Reader changes how fresh its view of a directory is.

**Recommended answer.** Two steps. Now, reword: scope.md's Performance gate and decision 63 should say that a cursorless Reader
read of the active segment costs O(offset within the segment), bounded by SegmentBytes (a few milliseconds at the default 8
MiB, an extrapolation both skeptics made, and the 1,000,000-record macOS run is consistent with it), that Records and Follow carry
a byte cursor and avoid it, and that a polling consumer should use them. After the release, if wanted, cache the sealed refs
between views the way `Log.sealedRefsLocked` does. A per-Reader anchor for the active segment, or a cursor returned from `Read`,
is the larger change and only worth it if polling consumers are expected.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: reword now; the sealed-ref cache after the release is deferred.

**Disposition.** Wording only, in 3c36fc0 (on main; verified from `git log`); no code changed and no fix-skeptic ran. scope.md's performance gate and decision 63 now state that a read with no iterator cursor (`Reader.Read`, `ReadWait`, `Scan`, `Consumer.Read`) re-walks and CRC-checks the active segment from byte 0 to the requested offset, O(offset within the segment) and bounded by `SegmentBytes`, and rebuilds the sealed-segment view, O(sealed segments), on every call; `Records` and `Follow` avoid both, and the user guide tells polling consumers to use them. The measured 379 ms is the auditor's, on the Linux box, at the top of a 343 MB active segment; "a few milliseconds at the default 8 MiB segment" is an extrapolation and is labelled as one in scope.md. Not done: caching sealed refs between views as `Log` does (decision 103).

## A3-19: the first-frame probe reads and CRC-checks the whole first frame (low, PLAUSIBLE)

**Claim.** The Reader's probe of each uncached sealed segment (reader.go:249 now, `window.frameAt(0)`) reads, allocates and
CRC-checks the whole first frame, up to 64 MiB, to learn two integers; decision 47 says "a few bytes". A directory of valid 64
MiB frames stored as sparse files made `quill stat` take 8.99 s and allocate 6.4 GiB on the Linux box (auditor; not re-run by a
skeptic). A skeptic's smaller run confirmed one full frame allocated per segment (40 MiB for five 8 MiB first frames).

**Why held.** The auditor's proposed fix is not behaviour-preserving. One skeptic patched the probe to read 64 bytes and skip
the CRC: two existing tests that assert hole detection failed, because an unchecked first-frame sequence feeds the segment
ranges. Skipping the CRC only above a size threshold leaves exactly the large first frames unchecked. The verdict is PLAUSIBLE
because the voters split on the remedy, not on the mechanism.

**Recommended answer.** Correct the wording, leave the code. Change decision 47 from "a few bytes" to "the whole first frame,
CRC-checked, up to 64 MiB, once per Reader per segment", and state the cost in the user guide. The cost needs write access to a
directory the operator reads (a restored log, or a subdirectory under `quill ls`), and is bounded at 64 MiB per call and
collectable. Revisit only if a bounded cheaper integrity check is designed, for example reading a header prefix and deferring
the CRC to the first scan of that segment, with a test that a damaged first frame is still reported as a hole.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: correct the wording, leave the code.

**Disposition.** Wording only, in 3c36fc0 (on main; verified from `git log`); no code changed and no fix-skeptic ran. Decision 47 now says the probe reads and CRC-checks the whole first frame, up to 64 MiB, once per `Reader` per segment, with the reason it is not cheaper (two hole-detection tests fail if the check is skipped), and the user guide states the cost for a directory whose first frames are very large. A cheaper bounded check remains a possible later change, with a test that a damaged first frame is still reported as a hole.

## A3-10: a damaged oldest segment is skipped silently by a default-start read (low)

**Claim.** A damaged or truncated oldest segment makes `quill cat` and `read` (default start), `tail -n` reaching the oldest
record, `Log`/`Reader.ScanAll` and a new consumer silently start at the first good segment and exit 0; the lost range shows
only in `stat` and `check`. Decision 38 and scope.md say a gap is "reported as a gap, never skipped". Full item, with its
reproduction: [threat-model.md](threat-model.md), A3-10.

**State at recording (2026-10-31).** A fix was written (54b99d0): a read from the beginning (`ScanAll`, a scan from offset 0,
`quill read` and `cat` without `-from`, `tail -n` reaching the oldest record) and a new consumer start at sequence 1 when a
damaged oldest segment is a hole from 1, so they return its GapError (CLI exit 1); a prefix deleted by retention, `-since` and
`OffsetForTime` still start at `First`. Decision 41, scope.md's Consumer row, the CLI help, the regenerated docs/reference.md,
the user guide and the changelog carry the exception. The fix-skeptic found the code sound and rejected it on the no-new-gap
lens only: three statements still say a new consumer starts at the first retained sequence, the godoc for `Log.Consumer`
(consumer.go:330-331), the comment on `consumerStore.open` (consumer.go:78-79) and the `Config.ProtectedConsumers` row at
scope.md:121. Recommended answer: approve the behaviour (it is what decision 38 and scope.md already say: "reported as a gap,
never skipped"), update those three statements, and land the commit. If the owner prefers expiry-like treatment of a damaged
oldest segment, the alternative is to say so in decision 38 and change no code.

**Owner approval.** Approved on 2026-11-02, as recommended: the behaviour change is intended.

**Disposition.** Fixed and on `main` (2026-11-02): badb254 (code, tests and docs, the second attempt 64837ac rebased onto
`main` with one `scope.md` conflict resolved), 18662b6 (decisions 9, 67, 84 and 85 and the two user-guide lines that the second
rejection named; 85 carried the same sentence and was not named) and ed518a2 (two comments in `archive.go` and a user-guide
note). The behaviour is as the owner approved: a read from the beginning and a new consumer start at sequence 1 when a damaged
oldest segment is a hole from 1 and return its `GapError`; retention, `-since` and `OffsetForTime` still start at `First`. The
second attempt also fixed a root cause the first left: `deleteLocked` (retention.go) dropped the hole when the retention pass
deleted the clean prefix in front of a damaged segment, so a live `Log` skipped the damaged file silently while a `Reader` and a
fresh `Open` reported it; it now keeps a hole from 1 while a damaged segment is the oldest sealed file.

Sign-off, stated exactly. The fix-skeptic of 64837ac rejected it on the docs alone (decisions 84, 9 and 67 and two user-guide
lines; reported from the ledger). A fresh fix-skeptic on 18662b6 (reported, its own runs) found the code and tests sound and
rejected it for two statements of the old rule that survived in `archive.go` (the `Archive` godoc and one inline comment). Its
checks: four new tests and the reference-currency test fail without the production change (the last only because the help text was reverted) and pass with it; seven hand-made mutants of the new start-at-1
logic and of retention's hole-keeping were all killed; a differential run of 400 seeds x 5 prune passes (2000 comparisons, 1042
with a hole from 1) found a live `Log` and a fresh `Open` agreeing every time; `docs/reference.md` is what `quill reference`
prints; the CLI before and after is identical on a healthy log, on a prefix deleted by retention and with `-since`, and differs
only for a damaged oldest segment (exit 1, "records 1 to 10 are unavailable"). I made the two comment edits it asked for and
added the user-guide note below, then ran the full gate on both platforms (verified). **No skeptic re-ran on ed518a2**: the
diff since the skeptic's tree is comments and documentation only (verified: `git diff 18662b6 ed518a2` touches `archive.go`
comments and `docs/userguide.md`), and I grepped the tree for the same statement and found none left outside the frozen
original `docs/design/build-prompt.md` (lines 212 and 318, left as they are) and `docs/audits`. That is a check by me, not a
fix-skeptic's FIXED-VERIFIED.

Open points the fix-skeptic left, none fixed here:

1. When every sealed segment of the source is damaged, `Archive` copies nothing, so it does not commit and every run exits 3
   until `quill commit -name _archive -offset N` goes past the gap (observed by the skeptic; before the fix this case exited 0
   silently). The user guide now says so where it claimed the next run exits 0.
2. Pre-existing, not introduced here: after archiving a source with a damaged oldest segment, `quill check` on the destination
   prints HEALTHY (exit 0), because a hole from 1 leaves no damaged file in the destination. `userguide.md` and `scope.md:321`
   say `check -dir <dest>` keeps reporting the hole (exit 3); that holds only when a damaged file was copied.
3. `quill tail -n 21` on a log with 10 records in a hole from 1 and 20 readable records exits 1 ("records 10 to 10 are
   unavailable"), because 21 reaches one record into the hole. That follows the rule, but no test pins it and the owner's
   wording ("reaching the oldest record") is looser.
4. The A5-18 notes (decision 36, the user guide, `scope.md`) say a consumer behind `First` means the log is intact and `check`
   stays HEALTHY. That describes the retention case; with a damaged oldest segment `check` is UNHEALTHY for the hole and also warns
   of the new consumer at 1 (observed by the skeptic).

**Follow-up to open point 2 (L4b-3, owner-approved 2026-11-04).** Reproduced on a build of the tree at bdb1522 (verified): a log of three
sealed segments of 500 records each, the first byte of the first frame of segment 0 damaged. `quill check -dir <log>` exits 3;
`quill archive -dir <log> -to <dest>` prints `the source lost records 1 to 500; the archive holds the same hole` and exits 3;
`quill stat -dir <dest>` shows `first 501`, `records 1000`; `quill check -dir <dest>` prints HEALTHY and exits 0; a second
`quill archive` exits 0. (A damage in the middle of segment 0 instead makes `archive` fail with exit 1 at that segment, a different
path.) The existing test `TestArchiveExitsUnhealthyWhenTheSourceLostRecordsTheArchiveNeverCopied` pins the other case: with records
already archived below the hole, `check` on the destination exits 3. The documents now say this (documentation only, no code change):
`userguide.md`, `scope.md`, decision 85 (a dated note), the `Archive` godoc and the CHANGELOG. Still open, recorded in the release
record's known limitations and not changed here: the message text "the archive holds the same hole" is inaccurate when the
destination starts after the lost records, since the destination then holds no hole, it begins later.

## A3-20: a Reader serves overlapping segments (low)

**Claim.** When a sealed segment holds frames past the next segment's first sequence, `Reader.Read` and `ScanAll` deliver the
earlier segment's records for sequences the next segment also claims; writable `Open` refuses the same directory with
ErrCorrupt; `Check` reports a nonsensical hole ("sequences 11 to 4 are missing"). Full item and reproduction:
[threat-model.md](threat-model.md), A3-20.

**State at recording (2026-10-31).** A fix was written (e0df757, not in this tree): a Reader refuses overlapping
segments as ErrCorrupt and Check reports an "overlap" problem; "overlap" is added to the closed `kind` list for `quill check
-json` in scope.md. The fix-skeptic rejected it because Archive, another path through `Reader.view`, is not covered:
`copySegment` (archive.go:167-192) never compares the segment's real last sequence with the reference's derived last, so on a
hand-built overlap Archive copies both segments, writes a destination `quill.tail` with seq 8 and returns GapError{From:11,To:4}
("records 11 to 4 are unavailable"), the same as before the fix. Recommended answer: add an overlap check in `copySegment` or
`archive` (reject when `info.last.Seq` is above the reference's bounded last, or run the reference's `ensureIndex`), add an Archive
test on the layout in threat-model.md A3-20, and land the commit. The new `kind` value is an addition to a documented JSON
field, so the owner should accept it.

**Owner approval.** Approved on 2026-11-02, as recommended, including the new `overlap` value in the closed `kind` list for
`quill check -json` (scope.md:206); the CHANGELOG says so.

**Disposition.** Fixed in 53a5d3b (an ancestor of this commit; verified from `git log`); fix-skeptic **FIXED-VERIFIED**, 7 of
7 lenses pass (reported by the skeptic, who ran everything on scratch copies on macOS and on the Linux box). The commit
cherry-picks e0df757 and resolves conflicts in CHANGELOG.md, docs/userguide.md and stat.go. A Reader refuses overlapping
segments as ErrCorrupt and `Check` reports an `overlap` problem (stat.go:247); `copySegment` now compares the verified
`info.last.Seq` with the derived `s.last` and returns ErrCorrupt before anything is written (archive.go:235-236), reusing the
whole-segment verification Archive already runs (no second scan, no new symbol). Test: `reader_overlap_test.go` (three
Reader subtests, two Archive subtests). Evidence, reported from the skeptic: with archive.go, reader.go and stat.go reverted
and the test file kept, all five subtests fail for the claimed reasons (Read(1) and ScanAll return 1..8 with nil error; Check
says "sequences 11 to 4 are missing"; Archive says "records 11 to 4 are unavailable" and "offset out of range"); reverting
only archive.go fails only the two Archive subtests, and only reader.go fails Read and ScanAll, so both halves are
load-bearing. On real files the built `cat` and `archive` exit 1 with "corrupt log: segment 0 holds sequence 10, inside the next
segment (starts at 5)", the destination holds no segment file and a zero marker, and `check -json` reports kind `overlap` with
exit 3. Gate on the commit (skeptic): gofmt, vet, `go test -race -count=1 ./...`, staticcheck v0.8.2, deadcode v0.52.0 all
clean; golden files and testdata unchanged; no exported symbol changed; fuzz 10 s per target passed. Residuals, as the skeptic
reported them and the CHANGELOG states the first: a fresh Reader whose first read starts in the later segment (`Read(5)`,
`Read(9)`) still serves that segment's own records without error, because the earlier segment is never scanned, and `Stat` and
`OffsetForTime` do not scan; decision 85 ("stops at a corrupt segment") and the ErrCorrupt comment in errors.go (it says only
`Open` returns it for an overlap) are not themselves amended, though decision 102 is and neither statement is false.

## Not held, but worth a line

**A3-26 (refuted, 1 to 2): unlinking `quill.lock` under a live writer lets a second writer open.** Out of the threat model and a
stated design rule, so not changed. Two hardening options were written up by the auditor, if the owner wants defence in depth:
compare the held descriptor's device and inode with `Lstat(quill.lock)` before each marker publish and fail the log with an
ErrLocked-wrapping error on mismatch (narrows the window to one flush); or also flock the log directory's own descriptor (probed
as exclusive on macOS and Linux). Cheapest step: one sentence in format.md and the user guide, "never delete `quill.lock`;
flock releases itself on process death", naming the consequence (two acknowledged writers, one of them silently overwritten,
`check` HEALTHY).

**A3-7 (refuted, 1 to 2): a symlinked `consumers/` is followed.** Hardening only: open `consumers/` once with
`O_DIRECTORY|O_NOFOLLOW` (or `Lstat` it in `create()` and every consumer operation) and require a real directory, or say in
decision 51 that only the last path component is protected.

**A3-24 (refuted, 1 to 2): `-json` passes bidi and format controls raw.** Escape U+200B-U+200F, U+202A-U+202E, U+2060-U+2069,
U+FEFF and U+061C as `\uXXXX` (still lossless) and print `time` as null outside years 0000-9999. It changes the `-json` output
shape, so it needs approval if wanted.

**Raised again, decided in round 1 (not re-opened):** A3-3 (file names printed unescaped; quote file-derived strings, skip or
flag invalid consumer names), A3-4 (a non-consumer file in `consumers/` protects every segment; ignore invalid names, report them
in `check`), A3-6 (Archive keeps a same-size destination segment without verifying it; verify or compare, and re-copy on
difference), A3-9 (a Reader takes its file mode from `quill.tail`; clamp it and intersect with the umask), A3-11 (`quill prune`
trusts `quill.policy`; print the policy and parse it strictly), A3-23 (offline verbs create a log at a mistyped `-dir`; fail with
ErrNoLog unless `quill.format` exists), A3-29 (name the filesystems where flock is not coherent, correct the bind-mount sentence,
optionally warn from an `fstatfs` check), A3-31 (ConsumerTTL staleness is clock-derived; document it), A3-32 (MinFreeBytes
re-measure after each unlink; document the cost). Each carries its round-1 reference and the auditor's proposal in
[threat-model.md](threat-model.md). A3-3, A3-6 and A3-23 were raised again in this pass with reproductions on the current tree,
so the owner may want to confirm the round-1 decision for them before the release.

## Pass A6: format and version-skew audit

Date 2026-10-31. Pass A6 (record: [format-compatibility.md](format-compatibility.md)) applied seven fixes, each re-attacked
by a fix-skeptic (7 of 7 FIXED-VERIFIED): A6-1, A6-2, A6-3, A6-5, A6-6, A6-7 and A6-9. It left two ledger items open,
A6-4 (recorded as held for the owner) and A6-8 (fix written, commit did not land), and one built tool, the skew harness,
that its reviewer rejected. The recommended answers are the recorder's reading of the auditors' proposals and the
skeptics' notes, and are recommendations, not decisions.

**Update 2026-10-31.** The owner approved A6-4 (confirm and close) and A6-8 (do it), both recorded in their sections below. The skew harness the reviewer rejected was rebuilt as 01cee10 (`hack/skew.sh`, `hack/skewopen`, CONTRIBUTING.md). The missing check was added: in the kill phase each version runs `recover` on its own copy of the same killed directory and the two copies must be identical (`diff -r`). Verified on macOS: v0.1.0 against 3c36fc0 passes in both directions, ten kills each, every kill leaving part of the stream; and a deliberate fault (recovery stops at the marker, dropping the frames after it) now makes it fail with `rc=1` and the two recovered `quill.tail` files in the evidence (seq 490 against 425). The previous script is reported, by its reviewer, to have passed that same fault; I did not re-run it. The gate is green on macOS and Linux on the harness commit. Not run: the cross-version run on Linux (the Linux box ran only the gate).

| id | sev | one line | recommended | status |
|----|-----|----------|-------------|--------|
| A6-4 | low | the writer could rotate into segment id 10000000000, which no scan recognises | confirm and close; the fix is already in the tree as A3-8 (bb646c0) | approved 2026-10-31 (confirm and close); closed, reproduction re-run on the fixed tree |
| A6-8 | low | `MaxSegmentBytes` (1 GiB) and `maxSegmentBytes` (1.5 GiB) differ only by case | rename the unexported one, one commit | approved 2026-10-31 ("do it"); fixed 003de2c (FIXED-VERIFIED, 6 of 6) |

## A6-4: the segment id has no ceiling (low)

**Claim.** The segment id is written `%010d` and parsed as exactly ten digits, but nothing stopped the writer creating id
10000000000. That file is invisible to every scan; the newest-segment check then refuses the log (ErrCorrupt), and
records acknowledged into it are unreachable by a writable open (readers still serve them). Reproduced on macOS by the
auditor and a skeptic: a log whose segment was renamed 9999999999.log with `quill.tail` seg and maxseg at that value,
then Open, Append, Roll, Append, Close all succeeded and the next Open failed with "quill.tail names segment
10000000000, the newest on disk is 9999999999". A forged `maxseg` of 99999999999 in an ordinary log gave the same
failure after one roll.

**Why held.** The fix makes the writer stop with an error at a documented limit (a changed behaviour at the ceiling, like
the sequence ceiling) and states a new limit in format.md. The ledger marked it semantic. Reaching the ceiling by
rotation needs about 10^10 rotations, so no real log is expected to get there; the cost of the decision is low either way.

**Recommended answer.** Confirm and close. The proposed fix, refuse a rotation whose next id would exceed 9999999999 and say
so in format.md, is already in the tree: the A3 pass fixed the same defect as A3-8 in bb646c0 (an ancestor of the A6
record commit). Verified by the recorder from `git show bb646c0`: `decodeTail` rejects `maxseg`, and so `seg`, above
`maxSegmentID` (9,999,999,999) on both the fast and slow paths; `rotateLocked` returns "segment ids exhausted" when
`l.maxSeg >= maxSegmentID`; docs/format.md and CHANGELOG say so; `TestRotationRefusesAnIDTheSegmentNamesCannotHold` seeds
the top ids and passed when the recorder ran it. The A3 pass's fix-skeptic verdict, FIXED-VERIFIED, is reported from
[threat-model.md](threat-model.md). What the owner decides is only whether the A3-8 wording ("the largest id ten digits
hold", an error at maxseg 9999999999, so the id 9999999999 is the last one usable) is the intended limit.

**Fix written.** bb646c0, landed under A3-8. No separate A6 commit exists, and no A6 fix-skeptic ran on it. The A6-4
reproduction itself was not re-run against the fixed tree (inferred from the decoder and rotation lines).

**Owner approval.** Approved on 2026-10-31, as recommended (confirm and close).

**Disposition.** Closed. The fix is bb646c0 under A3-8 (fix-skeptic FIXED-VERIFIED in the A3 pass; reported, from [threat-model.md](threat-model.md)), so the A6 ledger item moves from held to closed with no new commit. The record said the A6-4 reproduction had not been re-run on the fixed tree; it was re-run on 2026-10-31 with built CLIs (verified). A log whose only segment is renamed 9999999999.log, with `quill.tail` `seg` and `maxseg` at that value, then `roll`: before bb646c0 (its parent), `roll` exited 0 and created 10000000000.log, and the next `append` failed with "corrupt log: quill.tail names segment 10000000000, the newest on disk is 9999999999"; on main, `roll` is refused ("segment ids exhausted: 9999999999 is the largest the ten-digit names hold", exit 1), no 10000000000.log is created and `append` still succeeds. A forged `maxseg` of 99999999999 is refused at read ("maxseg 99999999999 above the largest segment id, 9999999999"). The limit as written in A3-8 stands: 9999999999 is the last usable segment id.

## A6-8: two constants differ only by letter case (low)

**Claim.** `MaxSegmentBytes` (1 GiB, the SegmentBytes configuration ceiling, log.go:97 now) and `maxSegmentBytes` (1.5 GiB,
the segment file limit and the bound on `quill.tail` `off`, load.go:19 now) differ only by the case of the first letter.
Both are used correctly today and a change to the lowercase value is caught by several tests; the risk is a mistyped
capital moving an on-disk limit by 0.5 GiB with the compiler silent.

**Why held.** It was not held for a decision. A fix was written (rename the unexported constant to `maxSegmentFileBytes`,
keep the exported name, no API effect) and its commit did not land: the workflow handed its fix-skeptic the placeholder
`PENDING` in place of a sha, `git rev-parse --verify PENDING` failed, and nothing was merged or reviewed. The tree still
has both names (verified by the recorder, grep).

**Recommended answer.** Do the rename in one mechanical commit (five files use the lowercase name: fuzz_test.go,
load_test.go, load.go, log.go, state.go), gate it, and have a fix-skeptic check that no behaviour moved. Do not rename the
exported constant, which is public API. Declining is defensible: this is hygiene, and a mutation of the lowercase value is
already caught by the segment-limit tests.

**Fix written.** A proposal only; no commit.

**Owner approval.** Approved on 2026-10-31 ("do it"), as recommended: rename the unexported constant, keep the exported one.

**Disposition.** Fixed in 003de2c (on main; verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported). The unexported `maxSegmentBytes` (1536 MiB, the segment file limit and the bound on `quill.tail` `off`) is now `maxSegmentFileBytes` in the five files that used it; `MaxSegmentBytes` (1 GiB, the `SegmentBytes` ceiling) is untouched. Verified by the recorder: the word-level diff is 17 identifier pairs and nothing else, and the gate is green on macOS and Linux. The skeptic checked that the exported API (`go doc -all`) is byte-identical, the value is unchanged, no live document keeps the old name, and that changing the new constant to 1 GiB makes three tests fail, so the segment-limit tests still guard it.

## Not held, but unresolved

**The skew harness cannot see a cross-version difference in recovery.** A script to rerun the v0.1.0 against HEAD
experiments (hack/skew.sh in the working copy it was built in) was built in pass A6. It is not in the tree, and its
reviewer rejected it: with the new version's `recoverSegment` changed to drop the post-marker frames v0.1.0 salvages, the
run still printed "OK: no divergence between v0.1.0 and HEAD" and exited 0. Its kill phase recovers each killed
directory with one version only and checks that the survivors are a contiguous, unchanged prefix; it never recovers a
copy of the same killed bytes with both versions and compares. The frame-width, CRC and policy faults were caught. Recommended
answer: if the owner wants the harness in the release, copy each killed directory, recover one copy with each version,
and `diff -r` the two before the append; then repeat the reviewer's deliberate-fault test and require the run to fail.
If the harness is not wanted, nothing in the format verdict depends on it: the manual kill experiment (180 iterations,
recovery by each version of identical torn bytes) made exactly that comparison and found no difference, as
[format-compatibility.md](format-compatibility.md) records.

**A6-6 stays PLAUSIBLE.** No fixture in the tree is a frame written by the reference implementation; docs/format.md now
says so (ee11f57), and compatibility with the reference rests on the specification. If the owner can supply a short
reference-written segment (a few records, one with an empty payload and one with a negative timestamp), check it in under
testdata with its provenance given as "the reference implementation" and decode it in `TestFrameGolden`; that
would settle decision 3 by bytes.

## Pass A5: silent-degradation audit

Date 2026-10-31. Pass A5 (record: [silent-degradation.md](silent-degradation.md)) applied seven fixes, each re-attacked by a
fix-skeptic (7 of 7 FIXED-VERIFIED): A5-2, A5-16, A5-21, A5-31, A5-39, A5-40 and A5-42. It held eleven items for the owner.
Every held item changes observable behaviour (a `Stats` field, an `Err()` rule, an exit code, a CLI line, a validation
refusal or documented output), so none was applied. No fix is written for any of them; each section gives the auditors'
proposal as the recommended answer. The recommended answers are the recorder's reading of the auditors' proposals and the
skeptics' notes, and are recommendations, not decisions. The full items (grounds, warrant, backing, rebuttal, votes,
reproductions) are in the record.

**Update 2026-10-31 (pass A5h).** The paragraph above is the state when the pass was recorded. The owner approved all eleven recommended answers on 2026-10-31. Result: 10 fixed and FIXED-VERIFIED (A5-10 91c2fce, A5-22 1dda449, A5-18 7b87e64, A5-36 2cbfdf2, A5-9 46f30db, A5-3 1e0a8ed, A5-23 4d0e8c9, A5-44 29e7db1, A5-29 caad4c5, A5-32 16cb0c8), and A5-1, which failed its first fix-skeptic and was fixed on the third attempt (99b140d, FIXED-VERIFIED, 2026-10-31). The table's status column and each section's **Disposition** are current; each section's "Fix written" line is the state at recording. Two approvals differ from the recommendation text: A5-22 declined the "N consecutive statfs failures" option, and A5-29 reverses the round-1 exit-0 ruling (A1-58, decisions 85 and 53). The per-item reproduction and fix-skeptic evidence are in [silent-degradation.md](silent-degradation.md). In `git log` order the commits are A5-10, A5-22, A5-18, A5-36, A5-9, A5-3, A5-23, A5-44, A5-29, A5-32 (verified), all ancestors of 16cb0c8, on which the A5-32 skeptic ran staticcheck, deadcode and the full race suite, all clean (reported).

Three of the items share one design question and should be decided together: A5-10 (what `Err()` means), A5-1 (what a
background failure reports) and A5-22 (what `diskFull` reports). Two share another, the CLI's stderr: A5-36 and the
broken-consumer Warn left over from A5-17. The offline write verbs now carry a stderr Logger (cmd/quill/write.go, the
`lossOnly` filter, from A5-2 and A5-42) that passes two messages only; widening that filter is the smallest route for the
second group (inferred from reading write.go:74).

| id | sev | one line | recommended | status |
|----|-----|----------|-------------|--------|
| A5-10 | medium | `Err()` is one latched slot: a load error masks later background errors and a recovered error never clears | stop the masking, clear on recovery | approved 2026-10-31; fixed 91c2fce (FIXED-VERIFIED, 6 of 6) |
| A5-1 | medium | flush, publish, rotate and prune failures reach only `Err()`; nothing is logged, timed or counted | log each transition; add `Stats` fields | approved 2026-10-31; fixed 99b140d on the third attempt (FIXED-VERIFIED) |
| A5-36 | medium | `quill prune` exits 0 with no message when an unlink fails, a consumer file is corrupt or the clock guard withholds segments | widen the stderr filter; exit 1 on `Err()` | approved 2026-10-31; fixed 2cbfdf2 (FIXED-VERIFIED, 6 of 6) |
| A5-18 | medium | a consumer below the first retained sequence reads healthy in `quill consumers` and `quill check` | derive a NOTE from the files | approved 2026-10-31; fixed 7b87e64 (FIXED-VERIFIED, 6 of 6) |
| A5-22 | medium | a failing statfs leaves `MinFreeBytes` unenforced or `diskFull` frozen, and `Err()` stays nil | set `Err()`, add `Stats.DiskFull` | approved 2026-10-31 (no N-failure rule); fixed 1dda449 (FIXED-VERIFIED, 6 of 6) |
| A5-29 | medium | `quill archive` reports a source gap once with exit 0, and not at all after `unregister` | exit 3 on a gap; compare with the destination's tail | approved 2026-10-31; fixed caad4c5 (FIXED-VERIFIED, 6 of 6) |
| A5-3 | low | `ReadWait` and `Follow` on a failed log block until the context ends | return `ErrFailed` from the wait | approved 2026-10-31; fixed 1e0a8ed (FIXED-VERIFIED, 6 of 6) |
| A5-9 | low | `quill prune` and `quill roll` discard `Close`'s error | report it, exit 1 | approved 2026-10-31; fixed 46f30db (FIXED-VERIFIED, 6 of 6) |
| A5-23 | low | the published disk bound is exceeded when `MaxBufferBytes` is below `FlushBytes` | refuse that Config in `validate` | approved 2026-10-31; fixed 4d0e8c9 (FIXED-VERIFIED, 6 of 6) |
| A5-32 | low | an archive rerun keeps a same-size damaged destination segment | verify the CRC and replace it | approved 2026-10-31; fixed 16cb0c8 (FIXED-VERIFIED, 6 of 6) |
| A5-44 | low | `quill ls` omits a log whose reader cannot open | skip only `ErrNoLog`; print an error row | approved 2026-10-31; fixed 29e7db1 (FIXED-VERIFIED, 6 of 6) |

## A5-10: `Err()` is masked and never clears (medium)

**Claim.** `Log.Err` returns the failed state, then the sealed-segment load error, then the last background error, so an old
hole permanently hides every later rotation, flush or prune failure, and a background error is never cleared when the
operation recovers. After any recovery `Err()` reports a failure that is over. Reproduced by a program on the public API on
macOS: after a rotation blocker was removed and rotation succeeded, `Err()` still returned the old EEXIST; with a damaged
sealed segment, a live rotation failure was invisible in `Err()`.

**Why held.** The fix changes what the documented signal means (docs/userguide.md tells operators to watch `Err()`;
scope.md calls it the background error and the sticky load error), and either adds `Stats` fields or changes what `Err()`
returns. That is a public contract at v0.1.0.

**Recommended answer.** Keep the sticky load error but stop it shadowing: return the join of the failed or load error and
the background error, or add `Stats.BackgroundErr` and `Stats.BackgroundErrAt`. Clear the background error, or stamp a
recovery time, when the operation that set it succeeds (flush clears the flush error, a rotation the rotation error, a clean
prune pass the prune error). Drop the load error once the segment it names has been deleted and its hole pruned.

**Fix written.** None. One case already works this way since A5-39: a policy write that fails and later succeeds clears its
own `Err()` (log.go:664), which is the pattern to extend.

**Owner approval.** Approved on 2026-10-31, as recommended: the join (not the `Stats.BackgroundErr` alternative), clear on recovery, drop the load error with its hole.

**Disposition.** Fixed in 91c2fce (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). `Err()` (log.go:634) no longer masks or latches. Background errors are held one slot per operation in `Log.bg` (flush write, marker publish, rotation, prune unlink and policy write; A5-22 later added a disk slot), each set by `setBg` and cleared by `clearBg` when its operation succeeds (log.go:671-674). `Err()` returns the failed state, or the sticky load error when the log is not failed, joined by `errors.Join` with every live slot, so `errors.Is` matches each; a single error comes back unwrapped. The load error is recomputed to the first remaining damaged sealed segment, or nil, when retention deletes segments. The separate `policyErr` field and the displacement logic in `recordPolicy` are folded into the policy slot (reported, fix summary). Residuals, reported by the skeptic and not fixed. (1) `Roll` (flush.go:230) and the age rotation in the prune pass (retention.go:152) call `setBg(bgRotate, err)` without the `l.failed` guard that `publishLocked` has (flush.go:143), so when the rotation fails fatally on the directory fsync `Err()` holds the failed error twice and returns `errors.Join(failed, failed)`: the same line twice. `errors.Is` and the nil or non-nil contract are unaffected, and before the fix the error came back once. I read the three sites in the tree as it is now and the guard is still absent from the two (verified). A one-line `if l.failed == nil` in each, or a dedupe in `Err()`, removes it; cosmetic, recommended, not applied. (2) A failed age rotation stays in `Err()` after its blocker is gone until the next rotation of either kind, which is arguably accurate because none has succeeded since. Not run by the skeptic: the fuzz step and `go mod tidy`.

## A5-1: background failures are not logged, timed or counted (medium)

**Claim.** Every retryable background failure is reported through one latched field, surfaced only by `Err()`. Nothing
reaches `Config.Logger` although the field documents "warnings and failures", `Stats` has no last-flush time, buffered
count or failing flag, and nothing says how much acknowledged data is currently only in memory. Reproduced on the in-memory
file system (580 appends acknowledged with durable head 1, Logger empty) and, by a second voter, on real files with
`chmod 0500` (`Err()` still EACCES after the fault cleared and a clean flush).

**Why held.** The minimum fix writes new log lines (a changed output); the fuller one adds exported `Stats` fields, which is
an API addition.

**Recommended answer.** Do the minimum now: log once on each transition into and out of a failing state (flush write,
publish, rotate, prune) at Warn and Info through the existing Logger. Add `Stats.LastFlushOK`, `Stats.BufferedRecords` and
`BufferedBytes` and `Stats.FlushFailing` before the first release: adding them later is only additive, but dashboards will
be built on `Err()` in the meantime. Decide with A5-10.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: log each transition (Warn on entering, Info on leaving, never per retry) and add the four `Stats` fields, building on A5-10.

**Disposition.** Fixed in 99b140d on the third attempt (on main; verified from `git log`; authored as 27ccd31 on branch `stage2/A5-1-a3`); fix-skeptic FIXED-VERIFIED for the approved scope, with follow-ups that were closed before landing (reported, from a skeptic that ran everything on scratch copies, on macOS and on the Linux box). The first two attempts (574b078 and bab952f) are not merged. bab952f was rejected for one documentation defect: `LastFlushOK`'s docs said that `time.Since(LastFlushOK)` with `BufferedRecords` above zero is how long the oldest acknowledged record has waited, which false-alarms on a quiet log (the skeptic's reproduction: `BufferedRecords` 1, `FlushFailing` false, `time.Since(LastFlushOK)` 1h0m0s). The third attempt: through `Config.Logger` each retried background failure (flush write, marker publish, rotation, prune unlink) logs one Warn on entry (with `op` and `err`) and one Info on recovery, never per retry (the skeptic: 50 failing retries gave exactly one Warn each); `Stats` gains `BufferedRecords`, `BufferedBytes`, `FlushFailing` and `LastFlushOK`, additive only. `LastFlushOK` is documented, and now stamped, as the time of the last flush that wrote and published successfully (the time the writer opened the log before the first). The documented alert is `FlushFailing`, or `BufferedRecords` above zero together with it, and `Err()` wrapping `ErrFailed`. Verified by the skeptic on real files (RLIMIT_FSIZE) with a fake clock: over 50 failing flushes `FlushFailing` is true with `BufferedRecords` 4 and `BufferedBytes` 119; after the fault clears and one flush succeeds they are false, 0 and 0; an idle log given one `Append` after an hour does not trip the alert. The secondary edge was real (a prune pass whose pre-delete publish failed logged a false "recovered" for the unlink) and is fixed, with a test the skeptic showed red on the old code. Decisions 60, 66 and 78, scope.md, the user guide, the doc comments and CHANGELOG are amended. Two gaps the skeptic raised were closed before landing: a failed log (after a data fsync error) keeps its buffered records without setting `FlushFailing`, so the docs now also say to alert on `Err()` wrapping `ErrFailed`; and a test that a retried rotation on an idle log does not stamp `LastFlushOK`, which fails when the guard in `flushLocked` is removed (verified by the recorder). Left as is, low severity: a prune pass that finds nothing to delete after a failed unlink logs "recovered" and clears `Err()`, consistent with A5-10 though the unlink was never retried; an idle log whose only failure was a marker publish after `Roll` advances `LastFlushOK` when it lands although no record was written (the doc says "wrote"). Not logged: a failed log itself, outside the retryable scope of the approval. Gate green on macOS and Linux on 27ccd31.

**Note 2026-11-02 (A5-1f, the failed-log line).** The sentence above, "Not logged: a failed log itself", is closed. The owner
asked how a failed log is supposed to log itself and approved the answer on 2026-11-02: at the one moment the log becomes
failed, write one `Error` record through `Config.Logger`. Fixed in 531fb35 (an ancestor of this commit; verified from `git
log`); fix-skeptic FIXED-VERIFIED, 6 of 6 lenses pass (reported by the skeptic, on macOS and on the Linux box). `fail`
(flush.go:125) is the only place the sticky failed state is set (the skeptic's grep, reported); it now takes `(op, err)`,
returns at once if the log is already failed, which keeps the first cause and logs nothing for a later failure, and otherwise
logs "log failed: it accepts no more writes and every later call returns ErrFailed" with attributes `op` and `error`. The
callers pass `flush` (a truncate that cannot undo a failed write; a data fsync error), `rotation` (a directory fsync at
rotation or seal), `segment delete` (a directory fsync in the prune pass) and `lock check` (a lost `quill.lock`); the older
lost-lock `Error` line, which carried no cause, is removed, so that path still gives one line. `Stats` and the offline verbs'
standard-error filter are unchanged, as asked, because those verbs already return the error and exit non-zero. Four tests
(a data fsync error, a truncate failure, two failures asserting one line and the first cause, a lost lock) each fail with the
body of `fail` reverted (reported, from the skeptic); a probe on real files that deleted `quill.lock` under a live writer, opened
a second writer and flushed twice printed exactly one `level=ERROR` line with `op="lock check"` and both flushes returned
`ErrFailed`. Gate (skeptic): gofmt, vet, `go test -race -count=1 ./...`, staticcheck v0.8.2 and deadcode v0.52.0 clean; golden
files unchanged; `go doc` differs only in the `Config.Logger` comment. Decision 60 carries a dated note; the user guide, the
scope.md and `Config.Logger` Logger texts and the CHANGELOG are updated. Residual, not a defect: there is no line without a
`Logger`, and the first cause now wins where the old code kept the last, which the finding required.

**Follow-up (2026-11-02).** Wording, no behaviour change (d7b51e0): `LastFlushOK` is documented as the last flush that completed, that is wrote and published its records or landed a marker publish that had failed with nothing new written; and "recovered" is documented as a pass that completed without a failure, not a retry of the failing step, pinned by a new test (a protected consumer makes the next prune pass find nothing: `Err()` is nil, one Info, the segment still on disk). Not done: a failed log (`l.fail`) still writes nothing to `Config.Logger`. Recommended: one Error record at the single place the sticky failed state is set, which logs exactly once because the state is sticky and keeps its first cause.

## A5-36: `quill prune` exits 0 when it did less than asked (medium)

**Claim.** An unlink failure, a corrupt consumer file and the clock guard each leave `quill prune` printing a counts line
(`6 -> 6`, `10 -> 7`) and exiting 0 with empty stderr. Reproduced with the built CLI by both voters for the unlink failure
(`chflags uchg`) and the clock guard, and for the corrupt consumer file by one.

**Why held.** The fix adds stderr output to a verb and makes `quill prune` exit 1 in a case where it exits 0 today, and
changes the library's `Prune` to return the unlink error instead of nil.

**Recommended answer.** Widen the offline verbs' stderr filter to the three messages the pass can produce ("cannot delete
segment", "age sweep limited by the clock guard", "age sweep stopped at a segment stamped in the future") and add the
broken-consumer Warn (A5-17); have `cmdPrune` exit 1 when `Err()` is non-nil after the pass; have `Prune` return the unlink
error. The clock guard stays a warning with exit 0, since `-force` is the documented answer to it.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: three messages through the offline filter, the broken-consumer Warn, exit 1 on an unlink failure, library `Prune` returns the error, the clock guard stays a warning with exit 0.

**Disposition.** Fixed in 2cbfdf2 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI on macOS and on the Linux box). `quill prune` now says why it did less than asked and exits 1 when an unlink failed. The offline stderr filter (`lossOrShortfall`, cmd/quill/write.go:87) passes "cannot delete segment", the clock-guard message and the future-stamp message. A new Warn, "consumer file could not be read", names each unreadable consumer file (or an unlistable `consumers/` directory) on every pass (retention.go:133, `brokenWhy`). Library `Prune` returns the unlink error instead of nil, and `cmdPrune` exits 1 when `Prune` errs. The clock guard stays a warning with exit 0. Residuals, reported by the skeptic. (1) The skeptic read the new `Prune` doc ("the same error is in `Err()`") as untrue when a sealed-segment load error is latched, because `Err()` then returned the load error first. That was the A5-10 masking; A5-10 (91c2fce) is an ancestor of this commit and `Err()` now joins the load error with the unlink error (verified by reading log.go:634), so the objection no longer holds on this tree. (2) The unlink-failure CLI test is darwin-only; Linux coverage is the library test with an injected fault plus the skeptic's manual `chattr +i` run, which is not committed. (3) The consumer Warn fires on every pass, flusher passes included: approved, but noisy for a library user whose consumer file stays broken. (4) The other writable verbs (`roll`, `append`, `recover`) now print the new Warns on stderr when their open path hits an unlink failure, and still exit 0; outside the approved scope.

## A5-18: a consumer that fell out of retention reads healthy (medium)

**Claim.** A consumer below the first retained sequence, whether unprotected, overridden by a cap or stale under
`ConsumerTTL`, shows in `quill consumers` a lag that counts deleted records, `protected` true and an empty NOTE, and `quill
check` prints HEALTHY. `Dropped` and `StaleConsumers` are never printed by the CLI and `Dropped` is 0 after a writer
restart. Reproduced by both voters with the built CLI (lag 80 with 16 records left; `Dropped` 8 then 0 after restart).

**Why held.** The fix changes CLI output and the `ConsumerInfo` shape (a new field or note) and may change what `Lag`
counts. Decisions 12 and 66 deliberately keep the counters in writer memory.

**Recommended answer.** Derive, do not persist. `quill consumers` NOTE and a JSON field when the offset is below the first
retained sequence; compute staleness for protected consumers from the policy's `ConsumerTTL` and the file mtime; log Warn
with the consumer's name when a deletion passes a consumer's offset; print `dropped` in `quill prune`. This follows the
precedent of decision 36, which added the "ahead of the head" note.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: derive, do not persist (decisions 12 and 66 stand).

**Disposition.** Fixed in 7b87e64 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI against real logs on macOS and the new tests on the Linux box). Nothing is persisted. `quill consumers` adds a NOTE ("behind the first retained sequence N: records a to b are gone") and `-json` `behind_first` and `stale`, each present only when true, so existing parsers see no change. Staleness for a protected consumer is derived from the policy's `ConsumerTTL` and the consumer file's mtime (the CLI repeats decision 45's rule in a helper of about 15 lines, because the library exports no way to evaluate it and no new exported symbol was allowed). `quill check` warns of a consumer behind the first retained sequence and stays HEALTHY, which goes beyond the literal approved answer and follows decision 36's "ahead of the head" note. A deletion past a consumer's offset logs a Warn naming the consumer (keys `consumer`, `offset`, `segment`, `last`), the offline verbs print that line, and `quill prune` prints `, dropped N`. The only library change is internal: `Log.offsets` and `consumerView.offsets` carry `{name, offset}` instead of a bare offset. Residuals, reported by the skeptic. (1) The deletion Warn inherits the A3-27 lag that decision 12 documents: seal-time caps judge consumers by the offsets of the last prune pass, so a consumer created after the writer started and before the first pass gets no Warn and no `Dropped` count for earlier cap deletions (the CLI still flags it from the files). (2) The CLI repeats decision 45's staleness rule and no test cross-checks it against the library, so the two can drift. (3) An abandoned consumer below `First` gets a Warn for every later deletion: noisy, but what the owner approved. (4) Inferred, not run: `check` derives the first retained sequence from the first good segment and `consumers` from `Stat().First`, so they could word a damaged-oldest-segment case differently; that log is already UNHEALTHY.

## A5-22: `MinFreeBytes` fails open when statfs fails (medium)

**Claim.** On a statfs error `capsLocked` logs one Warn and leaves `diskFull` as it was, so the floor is unenforced, or a
frozen `diskFull` keeps every append failing with ENOSPC after space returns, and `Err()` is nil in every state; `diskFull`
is in no `Stats` field and its clearing is not logged. Reproduced on the statfs seam by one voter; the other traced the
code.

**Why held.** The fix sets `Err()` in a new state and adds `Stats` fields.

**Recommended answer.** Set the background error when statfs fails or `diskFull` is set, log when `diskFull` clears, add
`Stats.DiskFull` and `Stats.FreeBytesAtLastPass`; consider treating the floor as unmet only after N consecutive statfs
failures rather than on the first. Fold into the `Err()` decision (A5-10).

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended except that the option of waiting for N consecutive statfs failures was declined: one failure is the trigger, and the documents say so.

**Disposition.** Fixed in 1dda449 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). A failing statfs, or a set `diskFull`, now sets a background error (the `bgDisk` slot, set in `capsLocked`, retention.go:246), so `Err()` reports it; `Stats` gains `DiskFull` and `FreeBytesAtLastPass` (stat.go:36-40); the clearing of `diskFull` is logged at Info. A later statfs success recomputes the state and clears both `diskFull` and the slot, so a frozen `diskFull` cannot outlive a pass that reads room. An earlier attempt was rejected for a broken table row in decision 25; the recorded attempt amends it inside the Why cell (reported by the fix summary). Accepted limit, stated in the documents: while statfs fails persistently and `diskFull` is set, `diskFull` stays frozen, appends keep failing with ENOSPC, and `Err()` reports both the ENOSPC and the statfs error until a pass can read room. That is the owner's "one failure is the trigger, no N-failure rule". The skeptic did not run an injected statfs failure on the Linux box (the seam is internal); the new tests passed there.

## A5-29: `quill archive` and source loss (medium)

**Claim.** Archive reports a source gap once, on stderr, exit 0; the next run is silent while the destination is UNHEALTHY.
After `quill unregister -name _archive` (the documented reset) Archive prints nothing and exits 0 while the destination gets
the same hole, because it never compares the committed offset with the destination's `quill.tail`. Reproduced by both voters
with the built CLI. The user guide's "the verb says so" is false for the `unregister` case.

**Why held.** The fix changes an exit status and Archive's documented return. The exit-0 half was ruled deliberate in round 1
(A1-58, decisions 85 and 53); the silent-after-`unregister` half is new. One voter would not rule out that an explicit
reset counts as acknowledging the loss.

**Recommended answer.** Return a distinct non-zero exit (the unhealthy code, 3) for a gap, and have Archive compare the
committed offset with the destination's `quill.tail` so the `unregister` case returns the `*GapError` too. If the owner keeps
exit 0, correct the user guide sentence for the `unregister` case now.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: exit 3 for a source gap, Archive compares the destination's tail, reversing the round-1 exit-0 ruling (A1-58, decisions 85 and 53).

**Disposition.** Fixed in caad4c5 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI on macOS and the archive tests on the Linux box). `quill archive` exits 3, the code `check` uses for UNHEALTHY, when the source had lost records the archive never copied (cmd/quill/consumers.go:209 at the fix commit), and `Archive` compares the destination's `quill.tail` with the first segment it copies (archive.go:80-85 and 105-107 at the fix commit), so after `quill unregister -name _archive` it returns the `*GapError` instead of leaving a silent hole. It builds on A3-14's destination check (303b794, an ancestor). The run that reports a gap still commits past it, so a later run with nothing to copy exits 0; `quill check -dir <dest>` keeps reporting the hole. Residuals, reported by the skeptic, matching the approved answer. (1) A gap is reported only by the run that finds it, because that run commits past the hole; a later run with nothing to copy exits 0 while `check -dir <dest>` still exits 3. (2) The destination comparison runs only when a sealed segment is copied (archive.go:105 at the fix commit), so after unregister a loss ahead of a still-active first retained segment is reported on the first run after that segment seals; the offset path (archive.go:74) reports at once (inferred from reading, with a probe). (3) After a manual forward `quill commit -name _archive -offset N` the message says "the source lost records" although those segments may still be in the source; the archive does have the hole, so exit 3 is right and only the wording is loose.

## A5-3: waiting on a failed log never fails (low)

**Claim.** After an fsync error `Log.ReadWait`, `Consumer.ReadWait` and `Log.Follow` block until the context ends and return
only the context error, while `Append`, `Flush` and `Close` return `ErrFailed`. Reproduced by both voters on the in-memory
file system.

**Why held.** The fix changes documented `ReadWait` behaviour: it would return an error where it returned a context error.
A caller may prefer to keep draining.

**Recommended answer.** In `Log.wait` return the failed error when `l.failed != nil` and the head is below the offset; reads
of existing records stay valid. Document it on `ReadWait` and `Follow`.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: `ErrFailed` from the wait when the head is below the offset; reads of existing records stay valid.

**Disposition.** Fixed in 1e0a8ed (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). `Log.wait` (read.go:358) returns `l.failed` when the head is below the offset, and `fail()` now broadcasts, so a waiter already parked when the log fails is woken with the error. The broadcast in `fail()` answers the skeptic's objection to the first draft: `Roll` and the retention directory-fsync failure call `fail()` with no flush to broadcast after it, so the check in `wait()` alone left a parked waiter hanging. Four `broadcastLocked` calls that followed a `fail()` became redundant and were removed. `Log.ReadWait`, `Consumer.ReadWait`, `Log.Follow` and `Consumer.Follow` all go through `Log.wait`; `Reader.wait` is deliberately unchanged because a Reader runs in another process and cannot know the writer failed. Reads of records that exist still succeed. Residuals, reported by the skeptic. (1) scope.md:90 still says `Follow` waits until its context is cancelled, without the failed-log exception; the table row at scope.md:62 states it, so the prose is incomplete, not false. (2) A waiter parked on an idle log whose `quill.lock` was removed learns of it only at the next flush or prune: pre-existing, not A5-3. (3) The skeptic could not run staticcheck or deadcode (local binaries mismatched the toolchain and the pinned versions need the network); the A5-32 skeptic's later run on the tip covers both (see the update at the top).

## A5-9: `quill prune` and `quill roll` discard `Close`'s error (low)

**Claim.** `withWriter` (cmd/quill/write.go:85) discards the error of `Close`, which carries the final directory fsync, so
a fatal fsync failure after a successful prune or roll exits 0; `append` and `recover` already report it (decision 100).
Reproduced once, with an fsync error injected into a patched copy of the CLI (the shipped binary has no fault hook).

**Why held.** It changes an exit code.

**Recommended answer.** Report `Close` in `withWriter` as `cmdAppend` and `cmdRecover` do and exit 1.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: report `Close`'s error as `cmdAppend` and `cmdRecover` do (decision 100) and exit 1.

**Disposition.** Fixed in 46f30db (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). `withWriter` (cmd/quill/write.go:104, used by `quill prune` and `quill roll`) calls `Close` inline instead of deferring it and, when `fn` succeeded, prints `Close`'s error on stderr and exits 1; when `fn` failed its own exit code stands. `prune` has printed its counts line before `Close` runs, so the summary still appears. Residuals, reported by the skeptic. `withWriter` no longer defers `Close`, so a panic inside `fn` would skip it, which is moot because the process exits and the lock goes with it. The test makes the directory read-only inside `fn`, so it proves the publication-of-`quill.tail` path only, not the directory fsync or the lock release; all three reach the same `Close` return, the only thing the fix changes. The skeptic did not run deadcode (the commit adds no symbol).

## A5-23: the published disk bound is wrong for `MaxBufferBytes` below `FlushBytes` (low)

**Claim.** The bound in scope.md and decision 69 uses `MaxBufferBytes` for the flush term, but the code (`maxActiveBytes`,
log.go:203) uses the larger of it and `FlushBytes`, and `validate` accepts a smaller `MaxBufferBytes`. Reproduced by both
voters: with `MaxBufferBytes` 1 the on-disk peak was 4,200,444 bytes against a published bound of 3,949,577.

**Why held.** Either the guarantee's wording or `Open`'s acceptance of a Config changes.

**Recommended answer.** Refuse `MaxBufferBytes < FlushBytes` in `validate` with `ErrConfig`: one line, nobody has a reason to
set it, and the published formula becomes exact. The runner-up, rewording the formula with the maximum, loses because it
keeps a Config that cannot do what its name says.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: `validate` refuses `MaxBufferBytes < FlushBytes` with `ErrConfig`, so the published bound is exact.

**Disposition.** Fixed in 4d0e8c9 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic). `Config.validate` refuses `MaxBufferBytes < FlushBytes` with `ErrConfig` (log.go:180, verified), placed after the negative-value case so a wrapped default is still reported as negative; equal values are accepted. `max(FlushBytes, MaxBufferBytes)` stays in `maxActiveBytes` and in the ceiling text: it is now always `MaxBufferBytes`, and changing it would touch decision 60, the user guide and CHANGELOG for no behaviour change (the fixer's choice, stated in decision 69's note). Residual: `max(FlushBytes, MaxBufferBytes)` at log.go:216 (at the fix commit) is now redundant but true; a later reader may wonder why it stays, and decision 69's note says. The skeptic did not re-run staticcheck or deadcode; the A5-32 skeptic's later run on the tip covers both (see the update at the top).

## A5-32: an archive rerun keeps a same-size damaged destination segment (low)

**Claim.** archive.go:178 returns success for a destination segment that exists with the same size, without reading it, so a
damaged copy survives a rewound rerun, the run exits 0, and only `quill check -dir dst` finds it. Reproduced by one voter
with the built CLI.

**Why held.** It changes documented behaviour (the rerun is idempotent and cheap because it skips).

**Recommended answer.** Verify the CRC of a same-size destination file and replace it when it fails, or require a flag for a
forced re-copy. Failing both, document that a destination is audited with `quill check`, not by a rerun.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: read and CRC-verify a same-size destination segment and replace it when verification fails, no flag.

**Disposition.** Fixed in 16cb0c8 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI on macOS and on the Linux box). `copySegment` now calls `destIntact` (archive.go:250, verified) for a destination segment of the source's size: it must pass `verifySegment` (the CRC of every frame) and carry the source's first and last sequence, or it is replaced through the existing temp-file-and-rename. A healthy same-size file is left alone, not rewritten (its inode is unchanged). The sequence comparison is an addition to the approved answer: a valid-CRC segment of the same size from elsewhere would otherwise pass. A rewound run with a same-size destination segment that cannot be read fails with exit 1; decision 85 says so. Residuals, reported by the skeptic. (1) The first and last sequence branch of `destIntact` has no unit test; a mutation removing it would pass the new test (the skeptic exercised it only through the CLI). (2) The user guide does not mention the exit 1 for an unreadable same-size destination segment; decision 85 does. This skeptic ran staticcheck v0.8.2, deadcode v0.52.0 and the full `-race` suite, all clean, on 16cb0c8, the tip of the chain.

## A5-44: `quill ls` omits a log it cannot open (low)

**Claim.** cmd/quill/read.go:45-47 skips every `OpenReader` error, so a log with a corrupt `quill.format`, a future format or
no permission vanishes from the listing and `ls` exits 0, while `stat` and `check` on it exit 1. Reproduced by both voters.

**Why held.** It changes `ls` output and possibly its exit code.

**Recommended answer.** Skip only `errors.Is(err, quill.ErrNoLog)`; for anything else print an "error:" row as the `Stat`
failure branch does, and decide whether `ls` then exits 1.

**Fix written.** None.

**Owner approval.** Approved on 2026-10-31, as recommended: skip only `ErrNoLog`, print an "error:" row for anything else, list everything, and exit 1 afterwards as `stat` and `check` do for the same log.

**Disposition.** Fixed in 29e7db1 (in the tree: an ancestor of this commit, verified from `git log`); fix-skeptic FIXED-VERIFIED (6 of 6 lenses pass; reported by the skeptic, who ran the built CLI on macOS and the new tests on the Linux box). `cmdLs` (cmd/quill/read.go:34) skips a subdirectory only when `errors.Is(err, quill.ErrNoLog)`; any other open or `Stat` failure prints an "error:" row, the other logs are still listed, and `ls` exits 1. "no logs under" is printed only when nothing was listed and nothing failed. The `OpenReader` and `Stat` failures share the one existing error-row branch and a `failed` counter. Residuals, reported by the skeptic. (1) A directory with `quill.format` and no `quill.tail` is still `ErrNoLog` (reader.go:155), so `ls` omits it and exits 0 while `stat` exits 1 (reproduced); this follows the approved "skip only `ErrNoLog`" rule and the mid-creation design, and the user guide's "no quill.format" wording omits it. (2) Any unreadable non-log subdirectory, such as a root-only `lost+found`, now makes `ls` exit 1: a consequence of the approval. Staticcheck, deadcode and fuzz were not run by the skeptic.

## Not held, but worth the owner's eye

- **A5-33 (closed by a pass A3 commit).** The umask finding was closed by 1f896c7, which documents that the writer's
  segments and directories go through the umask. The finding's optional second half, a `quill check` warning when a segment's
  mode differs from the marker's, was not applied: it would flag every writer run under a tighter umask, which the
  documents now call normal. Apply it only as a new decision.
- **A5-5 (refuted; closed for `Commit` and a Reader's wait, plain `Read` declined on cost; owner 2026-11-02).** `Commit` and a
  Reader's wait return `ErrWrongLog` on a replaced directory (495e457, A3-30). A Reader's plain `Read` and `Scan` still serve a
  replacement log from the old cursor, and that stays: on 2026-11-02 the owner said to stay with decision 83's refusal on cost
  if 83 declined it. The record's reading is that it did: the A5-5 verdict rests on decision 83 (a `Reader` does not re-check
  cached files on every call, so a read costs nothing per segment), and the voters recorded the one `quill.format` read per view
  as the cost. One caveat, inferred from decision 83's own text: 83 declines re-checking each cached segment (cost grows with the
  segment count), and a single `quill.format` read per view is constant, so 83 does not by itself rule it out; the decision to stay
  is the owner's, and the hardening is recorded as declined, not as open.
- **A5-17 (prior-refuted).** The age sweep is still suspended silently by an unparseable consumer file; a Warn per pass naming
  it was recorded as "would close it (not done)" in round 1 and is part of the A5-36 recommendation above.
- **Residuals of the seven fixes** (reported by the fix-skeptics, in the record): A5-16 does not stop a second writer that
  opens before the first writer's next flush; A5-21 leaves a pending marker publish on a full disk blocking every deletion,
  which the skeptic flagged as a possible separate finding and did not test; A5-2 and A5-42 report only to a configured
  Logger in the library.

## Pass L1: verification and optimisation

Date 2026-11-02, against 5a70379. Pass L1 (comment against code, dead code and debt, operational validation on macOS and
Linux, optimisation) filed 50 items; the full record, with every item's claim, grounds, warrant, backing, rebuttal, votes and
reproduction, is [verification-optimization.md](verification-optimization.md). 27 items were fixed and verified by a
fix-skeptic, 4 are recorded as safe cleanups not applied, 5 were refuted and 4 were decided in round 1. The seven items below
were held because the fix changes a guarantee's wording, an API contract, an exit code, an error text or a memory cost. Three
more items had a fix written and rejected by the fix-skeptic; they are listed after the held items so that nothing open is
only in the record. The recommended answers are the recorder's reading of the auditors' proposed fixes and the skeptics'
notes; they were not approved when written. **Update 2026-11-03:** the owner approved the recommended answer for all seven (L1-11,
L1-12, L1-4, L1-5, L1-16, L1-26 and L1-51) and asked for the three rejected fixes to be finished; each section below now carries an
**Owner approval** and a **Disposition**, and the recommendation text is unchanged.

| Id | Item | Severity | Verdict | Recommendation | Status 2026-11-03 |
|---|---|---|---|---|---|
| L1-11 | `check` reports a log healthy when its oldest sealed segment is empty | medium | CONFIRMED | Report it, as `Open` does | Approved 2026-11-03; fixed ce47243, FIXED-VERIFIED (7 of 7) |
| L1-12 | `archive` into a new destination copies only a suffix and exits 0 | medium as filed | PLAUSIBLE | Fix the wording now; the refusal is optional | Approved 2026-11-03, wording only; fixed be8cb15, FIXED-VERIFIED (6 of 6); refusal an open optional follow-up |
| L1-4 | "every later Sync fails" on a failed log is not what the code does | low | CONFIRMED | Reword; the code is right | Approved 2026-11-03; fix de3cfd4 REJECTED by the fix-skeptic, not on `main` (unswept `build-prompt.md:300`) |
| L1-5 | The sequence returned with an append error is inconsistent and undocumented | low | CONFIRMED | Return `first` on every post-assignment failure | Approved 2026-11-03; fixed 867f4ed, FIXED-VERIFIED (7 of 7) |
| L1-16 | `Consumer.ReadWait` does not return when the consumer closes | low | CONFIRMED | Reword; do not change the code | Approved 2026-11-03; fixed 923623c, FIXED-VERIFIED (6 of 6) |
| L1-26 | The `quill append` oversize message states a size that stdin does not have | low | CONFIRMED | Reword the message | Approved 2026-11-03; fixed c2cddb7, FIXED-VERIFIED (7 of 7) |
| L1-51 | `indexStride` 16 KiB is 22-27% faster to read at random positions, 4x the index memory | low | PLAUSIBLE | Keep 64 KiB | Approved 2026-11-03 (outside the L1h batch); deliberately not changed |

## L1-11: `quill check` reports an empty oldest sealed segment healthy (medium)

**Claim.** `Reader.Check` (and so `quill check`) reports a log HEALTHY when its oldest sealed segment has been truncated to zero
bytes, while `Open` and the Reader's `Stat` report a hole from 1 and the writer's `Err()` names the file ("a sealed segment with no
frames"). Check, Open and the Reader disagree about damage, and `check` is the tool operators use to find it. Item:
[verification-optimization.md](verification-optimization.md), L1-11.

**Reproduction (verified by both skeptics).** A 120-record, 3-segment log with `0000000000.log` truncated to 0 bytes: `Check`
returns healthy with no problems; the writer's `Stat` shows holes `[{1 120}]`; the Reader's shows holes `[{1 120}]`, first 121. A
400-record variant gave healthy, first 401, holes `[{1 400}]`; the same file overwritten with garbage made `Check` report a
"segment" problem.

**Why held.** The fix adds a fatal finding where `check` today exits 0, so it changes an exit code (0 to 3) and the problem set on
a layout. It is a behaviour change on the diagnostic surface, though the documented behaviour (decision 43, format.md:88: "a sealed
segment with no frames is corruption"; guarantee 6) is what the fix restores. quill cannot produce an empty sealed file itself
(`Roll` on an empty segment is a no-op), so it needs outside damage: a tmp cleaner, a bad restore. A hole in the middle is still
reported (as a hole naming the next file).

**Recommended answer.** Approve. In `Check`, when `id < tl.Seg` and `info.frames == 0`, report a fatal `segment` problem ("a sealed
segment with no frames", Offset 0, Intact 0), as `verifySegment` does. Only the active segment may be empty. Add a test for the
oldest and for a middle position. If the owner prefers `check` to be lenient about empty sealed files, say so in decision 43 and
change no code; the doc text in stat.go (`Check`: "every sealed segment whole") would then be what changes.

**Fix written.** None. The proposed change is the paragraph above.

**Owner approval (2026-11-03).** Approved: the recommended answer (a fatal `segment` finding for an empty sealed segment; only the
active segment may be empty).

**Disposition.** Fixed in ce47243 (on `main`), FIXED-VERIFIED by a fix-skeptic, 7 of 7 lenses pass (reported). `Reader.Check` now
reports a sealed segment (id below the marker's segment) with no frames as a fatal `segment` problem ("a sealed segment with no
frames", Offset 0, Intact 0), at stat.go:272, so `quill check` exits 3 where it exited 0 (verified: the line is in the tree).
Documented in decision 43 (dated note), scope.md guarantee 6, the user guide's Checking section, the `Check` doc in stat.go and a
CHANGELOG `Fixed:` line. Tests: `TestCheckReportsAnEmptySealedSegmentAndAgreesWithOpenAndStat` (check_empty_sealed_test.go:16) and
`TestCheckReportsAnEmptySealedSegmentWithExitThree` (cmd/quill/cli_test.go:371). Reproduction (reported by the fix-skeptic): with
only the parent's stat.go restored, the oldest-truncated case fails with Healthy true and no problems, the middle-truncated case
shows only a hole and no `segment` problem, and the CLI test fails for both files; the garbage-bytes and empty-active-segment cases
pass on both trees. With the fixed binary, truncating each of three sealed segments gave "damaged at byte 0, intact to byte 0: a
sealed segment with no frames" and exit 3, and emptying the active segment stayed HEALTHY, exit 0. The fix-skeptic left one
residual, recorded under "Open after a rejected fix" (the Reader's `Stat` for a middle or last sealed empty segment).

## L1-12: `quill archive` into a new destination copies only a suffix and exits 0 (medium as filed, PLAUSIBLE)

**Claim.** After the `_archive` consumer has advanced, `Archive` into a new or empty destination copies only records at or above the
consumer's offset, returns nil (exit 0) and gives no signal that the new destination is incomplete, although the source lost
nothing. The comment at archive.go:80-82 says a zero destination `Seq` means "the whole source is then the archive's to copy", the
`Archive` doc says the consumer "has not yet copied to dstDir", and the user guide says to archive into "a new or empty
directory". The offset belongs to the source log, not to the destination. The same holds for an older destination after a newer
one consumed the offset. Item: L1-12.

**Reproduction (verified by all three skeptics).** 30 records, `Roll`, `Archive(src, dA)`: nil. 30 more, `Roll`, `Archive(src, dB)`
with `dB` new: nil. dA: first 1, head 30, healthy. dB: first 31, head 60, `check` HEALTHY exit 0, holes `[]`. The source: first 1,
head 60. By the CLI: `archive -to dA`, then `archive -to dB`, both exit 0.

**Why held.** The behaviour follows decision 28 (one `_archive` consumer per source) and the exit-3 contract covers records the
*source* lost, which did not happen here; one skeptic called it the documented design (the third vote found that decision 85 does
not say so for a zero-`Seq` destination). What is clearly wrong is the comment and the text. Refusing a new destination that is
behind the consumer would change an exit code and an error and needs an owner decision. The single standing destination, the
intended deployment, is unaffected, and `quill commit -name _archive -offset N` rewinds the consumer.

**Recommended answer.** Fix the wording now: archive.go:80-82, the `Archive` godoc and the user guide (and scope.md's Archiving
paragraph) say the offset is per source and a new destination receives only records at or above it. Treat a refusal as optional
and separate: when the destination is new (`destSeq == 0`) and the source's first retained sequence is below the offset, return
an error that says the destination is new but the source's `_archive` offset is N, so `archive && notify-ok` cannot succeed on a
partial archive. It must not be a `GapError` ("records First to N-1 are unavailable"), because those records are available at the
source.

**Owner decision (2026-11-03).** Approved: the recommended answer, wording only. The comment in archive.go, the `Archive` godoc,
the user guide, scope.md's Archiving paragraph and decision 85 (dated note) now say the `_archive` offset belongs to the source, a
new or empty destination receives only records at or above it, a destination archived after the consumer advanced is a suffix, and
`quill commit -name _archive -offset N` rewinds the consumer to copy more. No code change; the test
`TestArchiveIntoANewDestinationAfterTheConsumerAdvancedCopiesOnlyTheSuffix` pins it (30 records, `Roll`, archive to dA; 30 more,
`Roll`, archive to a new dB: nil, dB's first sequence 31, the source's `_archive` offset 31 before the run).

**Open optional follow-up (not done).** Refuse a new destination that is behind the consumer: when the destination is new
(`destSeq == 0`) and the source's first retained sequence is below the `_archive` offset, `Archive` returns an error that says the
destination is new but the source's `_archive` offset is N (so `archive && notify-ok` cannot succeed on a partial archive). It must
not be a `*GapError` ("records First to N-1 are unavailable"), because those records are available at the source. It changes an
exit code and an error, so it needs its own owner decision.

**Fix written.** Wording only (see the owner decision above).

**Disposition.** Fixed (wording only) in be8cb15 (on `main`), FIXED-VERIFIED by a fix-skeptic, 6 of 6 lenses pass (reported). The
comment in archive.go, the `Archive` godoc, the user guide, scope.md's Archiving paragraph and decision 85 (dated note) now say
the offset belongs to the source and a new or empty destination receives only records at or above it; CHANGELOG `Changed:`. No
code changed, so the new test (archive_test.go:370) passes on the parent by design; the fix-skeptic checked that it pins the
behaviour by mutating archive.go to copy the whole source into a new destination, and the test then failed ("dB first 1 head 60,
want 31 and 60"). The CLI run (reported): archive to dA, more records and a roll, then archive to a new dB gave exit 0, `quill
check` HEALTHY, first 4, head 6; after `quill commit -name _archive -offset 1`, a new dC had first 1, head 6. The optional refusal
stays open, as recorded above.

## L1-4: "every later Sync fails" on a failed log is not what the code does (low)

**Claim.** `ErrFailed`'s doc (errors.go:31-36), decision 37 and docs/userguide.md:48-50 say that on a failed log "every later Append,
Flush and Sync fails". `Sync(seq)` for a sequence that became durable before the failure returns nil, and `Sync` of an unassigned
sequence returns `ErrBadSeq`. scope.md:245 uses the same "every" wording (noted by a skeptic). Item: L1-4.

**Reproduction (auditor, on `fsx.Sim`; the skeptics read the code).** 3 records appended, 1-2 flushed, a data-fsync error injected:
`Flush` returns "quill: log failed; close and reopen: EIO"; `Sync(ctx,1)` returns nil, `Sync(ctx,3)` returns the `ErrFailed`
error, `Sync(ctx,99)` returns `ErrBadSeq` (flush.go:285-288, 293-296).

**Why held.** The code is right: decision 42 gives `ErrBadSeq` for an unassigned sequence and "unless its sequence became durable"
for a waiter, and a durable record is durable. The fix rewrites the wording of an exported error's doc and of decision 37, which
is guarantee text.

**Recommended answer.** Approve the wording: "every later Append and Flush, and every Sync of a sequence not already durable, fails
with `ErrFailed`; `Sync` of an unassigned sequence is `ErrBadSeq`". Make the same edit in errors.go, decision 37, the user guide
and scope.md.

**Fix written.** None; the wording is above.

**Owner approval (2026-11-03).** Approved: the recommended answer (reword; the code is right; do not change the code).

**Disposition.** Fix written (de3cfd4, branch `stage2/L1-4-a2`, on top of 41297fb; verified not an ancestor of `main`) and
REJECTED by the fix-skeptic; not landed. The fix rewords errors.go, decision 37, the user guide and scope.md, adds a test on
`fsx.Sim` (Sync of a durable sequence nil, of an assigned non-durable sequence an error wrapping `ErrFailed`, of an unassigned one
`ErrBadSeq`; an oversized payload and an empty batch return `ErrPayloadTooLarge`, not `ErrFailed`, on a failed log) and changes no
code. The skeptic found the test a real pin (three mutations each fail it), the gate green and the goldens untouched. The
objection (reported; the recorder read each line on the branch and it is as stated): the sweep for "every later" is unfinished.
docs/design/build-prompt.md:300 (test T3) still says that after an fsync failure "every later call returns `ErrFailed`"; the
runtime message at flush.go:135 ("it accepts no more writes and every later call returns ErrFailed") and its quotation at
docs/userguide.md:53 keep the unqualified absolute, which is false for `Sync` of a durable sequence and for reads of existing
records. Remaining work: reword build-prompt.md:300; for the log message and its quotation either change the string (text only,
but the owner's call, since a script may match it) or say in the commit and CHANGELOG that it is left on purpose; optionally tighten
"unassigned" to "above the last assigned sequence", since `Sync(ctx, 0)` returns nil. Rebasing note (verified with `git
merge-tree`): merging the branch onto the current `main` conflicts in CHANGELOG.md only; errors.go, which L1-16 (923623c) also
edited, merges cleanly.

## L1-5: the sequence returned with an append error is inconsistent and undocumented (low)

**Claim.** `Append` and `AppendBatch` document no return value on the error path, and the code is inconsistent. When an inline
flush's fsync fails, `appendLocked` returns `(first, ErrFailed)` (append.go:91-97), but `AppendBatch` discards `first` and returns
`(0, err)` (append.go:38-41); after a `SyncAppends` failure it returns `(first, err)` (append.go:42-46). The sequence is
consumed and the record may survive a restart, so the caller cannot reconcile the failed append; docs/userguide.md:52-53 says
producers must tolerate duplicates. Item: L1-5.

**Reproduction (auditor, `fsx.Sim`, `FlushBytes=100`, a data-fsync error injected).** `Append` of 200 bytes returned seq 0 with the fatal
error while `nextSeq` was 2 and `bufCount` 1; `AppendBatch` also returned 0. Neither skeptic ran it; both traced the straight-line
code. Record survival after reopen is inferred from format.md:104 and the recovery description, not run.

**Why held.** Either choice is an API contract. Returning `first` changes what callers see on an error path; documenting 0 pins
a value that today differs between two call sites. The user guide says elsewhere that a rotation-ceiling failure "assigns no
sequence" (docs/userguide.md:61), so a 0 return reads as "none assigned", which is untrue for the inline-flush case.

**Recommended answer.** Return `first` alongside the error on every failure that happens after sequences were assigned (change
`AppendBatch` at append.go:41), return 0 only where nothing was assigned (`ErrClosed`, a refused batch, `ErrPayloadTooLarge`), say both
in the `Append` and `AppendBatch` docs and in `Config.SyncAppends`' text (log.go:33-35), and add a test that pins it. The sequence
is what lets a caller look for the record after reopening (the same reason the CLI prints it beside an error, decision 100).

**Fix written.** None.

**Owner approval (2026-11-03).** Approved: the recommended answer (return `first` with the error on every failure after sequences
were assigned; 0 only where nothing was assigned).

**Disposition.** Fixed in 867f4ed (on `main`), FIXED-VERIFIED by a fix-skeptic, 7 of 7 lenses pass (reported). `AppendBatch` now
returns `(first, err)` when the inline flush's fsync fails (append.go:52; verified in the tree), as `appendLocked` and the
`SyncAppends` path already did; `first` is 0 on every pre-assignment return, so no new branch was needed. Documented on `Append`,
`AppendBatch` and `Config.SyncAppends`, in the user guide, scope.md, decisions 42 and 100 (dated notes) and a CHANGELOG `Fixed:`
line; the CLI is unchanged and a comment at cmd/quill/write.go says why. Tests: append_error_seq_test.go:24 (an inline-flush
failure on `Append` and `AppendBatch`, and the `SyncAppends` failure) and :61 (the refused cases return 0). Reproduction (reported
by the fix-skeptic): with only the parent's append.go restored, the inline-flush subtests fail ("(0, quill: log failed ...), want
(1, ErrFailed)"; "want (2, ErrFailed)") and the `SyncAppends` subtest passes, as that path already returned `first`. An independent
program (FlushBytes 300, three appends first, then a 2-record batch with an injected data-fsync error) returned first 4 with an
error wrapping `ErrFailed`; after reopen, records 4 and 5 were the batch's. Residual, recorded under "Open after a rejected fix":
the text does not warn that an unpersisted record's sequence can be reassigned.

## L1-16: `Consumer.ReadWait` does not return when the consumer closes (low)

**Claim.** consumer.go:249 says `ReadWait` blocks "until a record is available, ctx is done or the handle closes". Closing the
`Consumer` does not wake or end a blocked `ReadWait`; only `ctx` or the underlying Log or Reader closing does, and `Read`, `Scan`
and `Records` keep working after `Consumer.Close`. In consumer.go "handle" otherwise means the Consumer, and the sentence looks
copied from `Reader.ReadWait`, where it means the Reader. A skeptic also noted errors.go:23 says `ErrClosed` covers "a call on a closed
Log, Reader or Consumer", but only `Commit` and `Close` return it on a closed Consumer. Item: L1-16.

**Reproduction (auditor `TestT4ReadWaitClose`; the skeptics read the code).** A caught-up consumer, `ReadWait` in a goroutine,
`c.Close()` returns nil, `ReadWait` still blocked 2 s later; afterwards `Read` returns 0 records and nil, and `Records` iterates normally.

**Why held.** The spec does not call for `Close` to end a wait, so the doc is the wrong side by default; but making `Close` wake
waiters and return `ErrClosed` is a behaviour change to an exported type, and narrowing errors.go:23 changes the stated contract.

**Recommended answer.** Reword consumer.go:249 to "ctx is done or the underlying Log or Reader closes"; say that `Read`, `ReadWait`,
`Scan`, `Records` and `Follow` ignore `Consumer.Close`; narrow errors.go:23 to match. Do not change the code.

**Fix written.** None.

**Owner approval (2026-11-03).** Approved: the recommended answer (reword; do not change the code).

**Disposition.** Fixed (wording only) in 923623c (on `main`), FIXED-VERIFIED by a fix-skeptic, 6 of 6 lenses pass (reported). The
`Consumer` godoc, the `ReadWait` doc, `ErrClosed` (errors.go), user guide section 4, scope.md (the Consumer row and the Errors
paragraph), decision 41 (dated note) and a CHANGELOG `Changed:` line now say that `Consumer.Close` releases only the lock, that
`Read`, `ReadWait`, `Scan`, `Records` and `Follow` ignore it, that a blocked `ReadWait` ends with `ctx` or when the underlying Log or
Reader closes, and that on a closed Consumer only `Commit`, `SeekToTime` (which commits, seek.go) and a second `Close` return
`ErrClosed`; `SeekToTime` is named because the fix-skeptic of an earlier attempt asked for it. The code is unchanged. Test:
`TestConsumerCloseDoesNotEndReadsOnlyCommitsAndSeeks` (consumer_close_test.go:14). Reproduction (reported): the production code is
unchanged, so the test passes on the parent by design; mutating `Consumer.Read` to return `ErrClosed` after `Close`, or removing the
closed check from `Commit`, each made it fail; it passed 30 times under `-race -count=30`; a probe showed a `ReadWait` on a closed
consumer returns `ErrClosed` when the Log closes. One minor point (reported): the test relies on fixed sleeps (100 ms to park, 300
ms to show the wait stays blocked); it was stable over 30 runs.

## L1-26: the `quill append` oversize message states a size that stdin does not have (low)

**Claim.** When `quill append` reads an oversize stdin it prints "quill: payload too large: stdin is a batch of more than 16777216
bytes, MaxBatchBytes is 16777216". The number is not stdin's size: the limit is applied to the sum of `len(line)+34` per line
(cmd/quill/write.go:162-168, decision 46), an upper bound on the encoded size. A stdin of about 0.96 MB trips it. Item: L1-26.

**Reproduction (verified by both skeptics).** `yes x | head -n 479349 | quill -dir t3 append` exits 0; with 479,350 lines
(958,700 bytes on stdin) it prints the message above and exits 1. The same count confirms the append summary's "about 479,000
one-byte lines".

**Why held.** The CLI's error text is part of its output shape, and a script or test may match it; the threshold itself is intended
(decision 46) and the code comment at write.go:159-161 is right.

**Recommended answer.** Reword, for example: "stdin would be a batch over MaxBatchBytes (%d); each line is counted as its length plus
34 bytes". Update any test that pins the old text. No change to the limit.

**Fix written.** None.

**Owner approval (2026-11-03).** Approved: the recommended answer (reword the message; no change to the limit).

**Disposition.** Fixed in c2cddb7 (on `main`), FIXED-VERIFIED by a fix-skeptic, 7 of 7 lenses pass (reported). The message is now
"payload too large: stdin would be a batch over MaxBatchBytes (16777216); each line is counted as its length plus 34 bytes"
(cmd/quill/write.go:166). The limit, the error identity (`ErrPayloadTooLarge`) and the exit code (1) are unchanged; decision 46 has
a dated note and the CHANGELOG `Fixed:` line says a script that matched the old text will break. docs/reference.md says "about
479,000 one-byte lines", which stays true. Test: `TestAppendStdinRefusalMessageStatesTheAccounting`
(cmd/quill/append_stdin_test.go:74), which also has the 479,349-line success case. Reproduction (reported by the fix-skeptic): with
only write.go restored to the parent, the test fails on the old text ("more than", and the missing accounting sentence); built
binaries: 479,349 lines exit 0 and print 479349; 479,350 lines (958,700 bytes) exit 1 with the new text; hand check 479,350 x 35 =
16,777,250 > 16,777,216 and 479,349 x 35 = 16,777,215. One residual outside the claim, recorded under "Open after a rejected fix".

## L1-51: a 16 KiB index stride reads faster at random positions for 4x the index memory (low, PLAUSIBLE)

**Claim.** Narrowing `indexStride` (load.go:23, 64 KiB) to 16 KiB makes a 100-record `Read` at a random sealed position or near the
tail 22-27% faster (macOS `ReadSealed` 37.3-38.3 us to 29.0-29.8 us, Linux 74.3-82.8 to 52.6-59.2 us; reads that start at an anchor are
unchanged), at 4x the resident index (24 bytes per anchor: about 0.15% of the log's size against 0.037%, about 1.5 GB per TiB of
retained log, per Log and per Reader that scanned every segment). benchmark-baselines.md:105 already lists it as a follow-up. Item: L1-51.

**Evidence.** Auditor benchmarks on both platforms (reported). Skeptics: one saw about 20-25% on random positions (verified, with
the full suite passing at 16 KiB); one saw 37.8 to about 32 us on `ReadSealed` and did not reproduce the 22-27%. The `Tail`
benchmark reads at one fixed offset, so its alignment effect is partly an artefact (32 KiB shows nothing on `Tail`).

**Why held.** It is a tuning trade-off of memory for latency, not a defect; nothing in the spec fixes the stride, but
docs/architecture.md:42 states "one anchor per 64 KiB".

**Recommended answer.** Keep 64 KiB for the first public release. The gain is 8-17 us on a read that already costs 40-80 us, it
appears only at random sealed positions, and the cost is a fourfold index per Log and per Reader. If a real workload shows
random-position reads dominating, take 32 KiB (about half the gain at half the memory), and update architecture.md:42, the
`load.go` comment and the baselines in the same commit.

**Fix written.** None.

**Owner approval (2026-11-03).** Approved: the recommended answer, keep the 64 KiB stride. This item is not part of the L1h batch
(no fix, no fix-skeptic), and it is noted here only so that nothing in this section is left without a decision.

**Disposition.** Deliberately not changed: no code change. The reason is the recommendation's: the gain is 8-17 us on a read that
already costs 40-80 us, it appears only at random sealed positions, and the cost is a fourfold index per Log and per Reader.
Reopen only if a real workload shows random-position reads dominating; then take 32 KiB and update architecture.md:42, the
`load.go` comment and the baselines together.

## Open after a rejected fix (not held for an owner decision)

Three items had a fix written whose fix-skeptic rejected it on a stated defect. No owner decision was needed beyond letting the
corrected fix land, and the owner asked on 2026-11-03 for the three to be finished. **Update 2026-11-03 (pass L1h):** L1-21 is
fixed and verified (41297fb, on `main`). L1-3 and L1-10 each had a further attempt written (47df71b, 0a77730) and each was
rejected again, on a different defect; L1-4, the approved wording fix above, was also rejected, on an unfinished sweep
(de3cfd4). Those three are not on `main` (verified: `git branch --contains` lists only their side branches `stage2/L1-3-a2`,
`stage2/L1-4-a2` and `stage2/L1-10-a4`) and each is open, with the remaining work stated. The earlier attempts' text is kept
below as first written, so the history stays readable.

**Finished by hand (2026-11-03, after the second rejection).** The owner asked for the three to be finished; the workflow's second
round rejected L1-3, L1-4 and L1-10 each on a documentation residual, so the recorder cherry-picked the last attempt of each onto
`main` and closed exactly the objection named, the way A3-10 was finished. **No fix-skeptic re-ran on these final states.** What
backs them is the recorder's own work (verified: each test run with `-race -count=1` in the worktree, and the L1-3 tests
mutation-checked) and the gate at the landing tip. L1-4 (9c1784d, a4ddc8f): `docs/design/build-prompt.md` T3 now says "every later
valid `Append` and `Flush`, and every `Sync` of a sequence not already durable"; the runtime text of the failed-log `Error` record
("every later call returns ErrFailed", flush.go:135) is left as written on purpose, because changing it is a code change nobody
approved, and decision 37 and CHANGELOG say so; the optional tightening of "unassigned" (`Sync(ctx, 0)` returns nil) was not made.
L1-3 (0aa9659, 66c7ec3, 2aa5a21): the Unreleased `Fixed:` entry about a `Log`'s reads no longer says "starting at the offset asked
for"; `Records` and `Follow` are in the From rule in decisions 101 and 103 and the user guide; `TestLogGapFromIsTheFirstUnavailableSequence`
covers `Log.Scan`, `ScanAll`, `Records` and `Follow` (verified: it and the Reader test fail when the `GapError{From: cur}` at
read.go:234 is mutated to `from.seq`). L1-10 (041cc34, 8e45aa0): log.go, scope.md, the user guide and the decision 25 and 50 notes
say a deletion is one record (plus a `Warn` per consumer it passes) and a segment already gone gives its `Warn` and then the same
deletion record, so two; the seal sources are "a `Roll`, or any flush that rotates" (an `Append`'s inline flush, `Flush`, `Sync`,
`Close` or the flusher's), read from `publishLocked`; `TestASegmentAlreadyGoneGivesItsWarnAndThenTheDeletionRecord` pins the two
records. The one thing not checked is the skeptic's inference that a `Sync` or `Close` flush that rotates runs the caps at that
seal: the recorder read `publishLocked` (flush.go:166-190), which every flush calls, and did not run a test for it.

| Id | Attempt | Fix-skeptic | State on 2026-11-03 |
|---|---|---|---|
| L1-21 | 41297fb | FIXED-VERIFIED, 6 of 6 lenses | Fixed, on `main` |
| L1-3 | 47df71b (earlier attempt 52be0dd) | REJECTED twice | Finished by hand 2026-11-03: 0aa9659, 66c7ec3, 2aa5a21; no fix-skeptic re-run |
| L1-10 | 0a77730 (earlier attempt 8192ca0) | REJECTED twice | Finished by hand 2026-11-03: 041cc34, 8e45aa0; no fix-skeptic re-run |
| L1-4 | de3cfd4 | REJECTED | Finished by hand 2026-11-03: 9c1784d, a4ddc8f; no fix-skeptic re-run |

- **L1-3: `GapError.From` is "the requested offset" only if nothing was delivered.** Attempt 52be0dd (docs only). Rejected:
  `Consumer.Scan` also returns the gap immediately after delivering records, so the CHANGELOG sentence that "Read, ReadWait and the
  consumers' reads leave such a gap for the next call" is false for it, and decisions 101 and 103 list "Scan, ScanAll or
  Reader.Scan" and omit `Consumer.Scan`. Remaining: name `Consumer.Read` and `Consumer.ReadWait` in that sentence, add
  `Consumer.Scan` to the lists, add a `Consumer.Scan` test case. The text to land: "`From` is the first unavailable sequence: the
  requested offset when nothing was delivered before the gap, otherwise the sequence after the last record the scan delivered;
  `To` is the last missing sequence, so a consumer resumes with `Commit(To+1)`." Behaviour stays as decision 84 says.
  **Owner approval (2026-11-03):** finish it; behaviour stays as decision 84 says (a code change making `From` always the requested
  offset would contradict decision 84 and was not approved). **Disposition (2026-11-03):** attempt 47df71b (branch
  `stage2/L1-3-a2`, docs and tests only) documents `From` as the first unavailable sequence for every call form that can deliver
  records before a gap, `Records` and `Follow` on Log, Reader and Consumer included, with a new test per form; it was REJECTED by
  its fix-skeptic. Objection (reported; the recorder read the line on the branch and it is as stated): CHANGELOG.md:197, an
  Unreleased `Fixed:` entry, still says a Log's `Read`, `Scan`, `ScanAll`, `Records`, `Follow` and consumer reads return a
  `GapError` "starting at the offset asked for", which is false for `Scan`, `ScanAll` and `Consumer.Scan` after delivered records
  and for `Records` and `Follow`, where `From` is the sequence after the last record delivered (read.go:222; the `scanSource`
  `gapAfter` at read.go:141 and iter.go:64 in the attempt tree). The same commit amended decisions 101 and 103 for this sentence
  but left the changelog copy, so the Unreleased notes contradict themselves (line 27 against line 197). Also reported: the new
  test covers the Reader and a Reader-backed consumer only, not the Log forms the new text names. Remaining: a one-clause edit to
  CHANGELOG.md:197, and a test over the Log forms (`Log.Scan`, `ScanAll`, `Records`, `Follow`, `Consumer.Scan`). Rebasing note
  (verified with `git merge-tree`): the branch conflicts with `main` in docs/design/decisions.md and iter.go.
- **L1-10: the `Config.Logger` doc omits records the code writes.** Attempt 8192ca0 (a complete grouped list in log.go, scope.md:122
  and userguide.md:205). Rejected: it says the MinFreeBytes records are "one record per transition, not per pass", but the "cannot
  read free space" Warn (retention.go:268) is written on every pass where `statfs` fails (6 identical Warns for 6 passes,
  reproduced). Remaining: say that Warn is written on every failing pass and keep "one per transition" for the below-floor Warn
  and the back-above Info; or add a transition guard in code (a behaviour change, so the owner's call).
  **Owner approval (2026-11-03):** finish it as documentation only; no transition guard in code (a behaviour change nobody
  approved). **Disposition (2026-11-03):** attempt 0a77730 (branch `stage2/L1-10-a4`; log.go, scope.md, the user guide, decisions
  25 and 50 and CHANGELOG, with tests) says the "cannot read free space" Warn is written by every pass and every seal whose
  `statfs` fails, that the deletion and MinFreeBytes records come from the caps, which run at each seal as well as in a prune pass,
  and that only the below-floor Warn and back-above Info are one per transition; it was REJECTED by its fix-skeptic. Objection
  (reported, reproduced by the skeptic with a scratch test; the recorder read `deleteLocked` on the branch, retention.go:345-374,
  and it is as stated): the text says a deleted or already-gone segment "is one record when it goes" in log.go,
  docs/design/scope.md:122, docs/userguide.md:206 and the notes on decisions 25 and 50. In `deleteLocked`, the already-gone
  branch logs the Warn "segment was already gone" and does not skip what follows, so it also logs the Info "deleted segment" (or
  the Warn "deleted damaged segment" for a damaged one): a segment removed by hand and then a `Prune(true)` wrote both records for
  `0000000000.log`, which is two records. The godoc also lists "Info for each segment deleted" and "Warn for a segment already
  gone" as separate kinds, which a reader will take to mean the already-gone segment gets only the Warn. None of the three new
  tests exercised the already-gone path. Smaller point, inferred by the skeptic from reading and not run: the list of seal sources
  names `Append`, `Flush`, `Roll` and the flusher, but a `Sync` or `Close` whose flush rotates also runs the caps at that seal.
  Remaining: correct the "one record" sentence in all four places (log.go, scope.md, userguide.md, decisions 25 and 50) to say the
  already-gone case writes the Warn and then the deletion record, add a test over the already-gone path, and add `Sync` and
  `Close` to the seal sources. Rebasing note (verified with `git merge-tree`): the branch conflicts with `main` in
  docs/design/decisions.md.
- **L1-21: the O(log segments) cost of `OffsetForTime`.** Attempt 9a3f648 (seek.go, scope.md, decisions 59 and 63, CHANGELOG state
  the cost as O(log segments) comparisons plus one segment scan over a view that costs O(sealed segments) per call and, on a
  Reader's first call, one open per sealed segment; counting filesystem: 403 opens on the first call, 2 on the second, for 401
  segments). Rejected: docs/design/build-prompt.md:385 (test T19) still says "cost touches O(log segments) first frames and
  one segment". Remaining: that one sentence.
  **Owner approval (2026-11-03):** finish it. **Disposition (2026-11-03):** fixed in 41297fb (on `main`), FIXED-VERIFIED by a
  fix-skeptic, 6 of 6 lenses pass (reported). It reuses the earlier attempt (cherry-picked 5d4a157, branch `stage2/L1-21-a3`, which
  already corrects build-prompt.md:385 and the other sites) and changes only the dated notes on decisions 59 and 63 to read "Stage 2
  audit, owner-approved 2026-11-03", which is what the previous rejection turned on. Docs and comments only: seek.go, scope.md,
  decisions 59 and 63, build-prompt.md T19, a comment in seek_test.go and a CHANGELOG `Changed:` line. Reported by the
  fix-skeptic: the non-comment lines of seek.go and seek_test.go are identical before and after, `go doc -all .` is byte-identical,
  and a grep of the tree for "O(log", "log segments", "first frames" and "touches" finds only the corrected statements (and, in
  docs/audits, quotations of the old sentence as history). The new text agrees with the code the skeptic read: `Reader.view`
  lists the directory and opens each sealed segment (reader.go:332-355), `Log.view` copies the cached sealed refs
  (read.go:348-358), and `offsetForTime` binary-searches the view and scans one segment (seek.go:50-110). Not re-run by the
  skeptic: the auditor's counting-filesystem reproduction (403 opens, then 2, for 401 segments), which rests on the auditor's
  report.

### Residuals the fix-skeptics left on the fixed items (2026-11-03)

None blocks a fix; each is a separate point, listed so that it does not live only in a skeptic's notes. Reported by the named
fix-skeptic unless it says otherwise.

- **L1-11, sibling: the Reader's `Stat` does not report an empty middle or last sealed segment as a hole.** With
  `0000000001.log` truncated, `quill stat` says "first 1, records 120" with no hole line, while `quill cat` stops with "records
  41 to 80 are unavailable". Not caused by the fix and outside its claim (`Check`). The fixer disclosed it only in a test comment,
  not in the CHANGELOG. Recommended: file it as its own item (a `Reader.Stat` sibling of L1-11) rather than leave it in a comment.
  Also untested: a marker that names a sealed segment at offset 0 over an emptied file would stay HEALTHY; the skeptic judged it
  unreachable without a forged marker (inferred).
- **L1-5: the new text does not warn that a record that was not persisted leaves its sequence free for reassignment.** It tells
  callers to look for the record by its returned sequence after a reopen; docs/userguide.md:403 says for crashes that the sequence
  can be reassigned to a later, different record. The wording is the owner-approved wording; recommended: one sentence in the
  `Append` doc pointing at that paragraph, at the owner's discretion.
- **L1-26: the library's own message prints a sum, not a size.** append.go:46 says "batch of about N bytes", where N is the
  accounted sum (length plus 34 per record). It is labelled "about" and outside the claim (the CLI's stdin); the owner approved
  only the CLI rewording.
- **L2-53: the user guide says the cap "bounds the memory of a read".** A bounded read still allocates about 4 times the cap
  (70 MB against 16 MiB, measured by the fix-skeptic); the comment on the constant in read.go says so honestly. Recommended:
  reword userguide.md:207 to "bounds the size of a batch", as the other texts do.
- **L1-16: the new test is partly timing-based** (fixed 100 ms and 300 ms sleeps); it was stable for 30 runs under `-race`.

## Pass L2: test, benchmark and hardening suite

Date 2026-11-03, against `f274b51`. Pass L2 (a coverage ledger on two platforms, three hand-mutation runs and an invariants-and-fuzz run) filed 53 items after merging five duplicates; the full record, with every item's claim, grounds, warrant,
backing, rebuttal, votes and reproduction, is [mutation-hardening.md](mutation-hardening.md), with [defects-log.md](defects-log.md), [harness-inventory.md](harness-inventory.md), [test-ledger.md](test-ledger.md) and
[untestable-without-x.md](untestable-without-x.md) alongside. 52 items were test gaps and were fixed with tests, each fix FIXED-VERIFIED by a fix-skeptic (312 lens verdicts, all pass). One item is held, below: it is the only
one that is a production behaviour, and a fix changes what a read returns and the wording of a guarantee. The recommended answer is the recorder's reading of the auditor's proposed fix and the voters' notes; it was not approved when written.

| counted over the 53 items | |
|---|---|
| verdict | 47 CONFIRMED, 6 PLAUSIBLE, 0 REFUTED |
| disposition | 52 fixed, 1 held for the owner (L2-53), 0 deliberately not changed. **2026-11-03:** L2-53 approved and fixed (637ed42), so 53 fixed, 0 held |
| finding-skeptic votes | 109 (101 CONFIRMED, 5 PLAUSIBLE, 3 REFUTED) |
| reproduced by execution | 52 of 53 items by at least one voter |

| Id | Item | Severity | Verdict | Recommendation | Status 2026-11-03 |
|---|---|---|---|---|---|
| L2-53 | `MaxReadBytes` bounds payload bytes only, so a read of tiny records allocates without a bound | medium | CONFIRMED (2 of 2 voters, both reproduced) | Charge a per-record overhead against the cap | | Approved 2026-11-03; fixed 637ed42, FIXED-VERIFIED (7 of 7) |

Other points the owner may want to see, none held: the tests that pin behaviour no decision states are listed in [mutation-hardening.md](mutation-hardening.md), section "Behaviour pinned that the spec does not state" (L2-12, L2-50, L2-27, L2-18, L2-31,
L2-47, L2-52, L2-19); the L2-31 fix edited decision 45's wording from "older than" to "at least as old", which is a decision text changed by a fixer.

## L2-53: `MaxReadBytes` bounds payload bytes, not the memory of a read (medium)

**Claim.** `scanSource` counts payload bytes only against `MaxReadBytes` (read.go lines 259 and 266 in the tree: `bytes += len(rec.Payload)`, stop at `bytes >= maxBytes`), and `Read(offset, 0)`, `Records` and `Follow` pass a limit of 0, which means no limit
(iter.go:63). A record with an empty or tiny payload adds 0 or 1 to a total that must reach the cap, so one call can return, and allocate for, every record of the log. The documented 16 MiB read bound therefore does not bound the memory a read takes. Item: L2-53.

**Evidence.** Reproduced by the auditor and by both voters, default `Config`, real disk (reported; the recorder ran nothing). 4,000,000 empty-payload records: `Log.Read(1,0)` returned 4,000,000 records and allocated 788 MB, `Records()` 787 MB. 17,000,000 one-byte records:
`Read(1,0)` returned 16,777,216 records, allocated 3,892 MB and the process peaked at 2.78 GB resident; `Records()` allocated 3,935 MB. Voter runs: 1,000,000 empty payloads, 206 MB; 2,000,000 empty payloads, about 403 MB per call, against a 16 MiB cap. That is about 232 bytes allocated per
record at the largest size (inferred: 3,892 MB over 16,777,216 records). I read the loop condition and the accumulation in the tree and they match the claim.

**Why held.** The fix changes observable behaviour. A read of small records would return fewer records per call than it does now, which a caller that depends on the batch size (for example a loop that calls `Read` with limit 0 and expects the whole tail) would notice, and the wording of
`MaxReadBytes` ("payload bytes", in `log.go`, scope.md and decision 79) and the guarantee 7 text would change. By the letter the current behaviour is as documented, and a caller of `Read` can pass a limit; but `Records`, `Follow` and the consumers' iterators have no limit parameter, and decisions 50 and 79
justify the cap by "a read into memory needs a byte bound". The default is one an ordinary workload meets (counters, heartbeats), not only a hostile one.

**Recommended answer.** Charge each record `len(payload) + 48` bytes against `MaxReadBytes` (the auditor's proposal named `frameLen` or a constant of about 48; 48 is a little over the 40-byte `Record` the auditor cites, inferred, not measured), keep "at least one record per call", and reword the cap as "payload
bytes plus 48 bytes of overhead per record, a bound on the size of a batch, not on the sum of payloads". At the default cap a read of one-byte records then returns about 342,000 records per call instead of 16.7 million (16,777,216 / 49, inferred), about 49 times fewer, and the memory per call falls from gigabytes to
tens of megabytes (inferred from the measured 232 bytes per record; it stays a small multiple of the cap, not equal to it). The runner-up is the constant 34 that decision 46 already charges at append, for consistency (479,349 one-byte records per call; about 111 MB at 232 bytes each, so a larger multiple of the cap). The
second runner-up is to leave `Read` as documented and give only the iterators a record-count bound of their own (iter.go passes no limit today), plus a sentence in the `Read` doc that `limit` 0 bounds payload bytes only; that changes nothing a caller can see except the iterators' internal batch size. If the owner
prefers not to change behaviour at all, document that `MaxReadBytes` does not bound the record count and recommend `limit`.

**Fix written.** Owner-approved 2026-11-03 (the recommended answer) and written: see [defects-log.md](defects-log.md), D1, and `read_overhead_test.go`. The test as proposed: 1,000,000 empty-payload records, default `MaxReadBytes`; `len(Read(1,0))` far below 1,000,000, and `Records` batches bounded. See [defects-log.md](defects-log.md), D1.

**Owner approval (2026-11-03).** Approved: the recommended answer (charge each record `len(payload) + 48` bytes against
`MaxReadBytes`; keep "at least one record per call"; reword the cap everywhere). The owner accepts that a read of small records
returns fewer records per call.

**Disposition.** Fixed in 637ed42 (on `main`), FIXED-VERIFIED by a fix-skeptic, 7 of 7 lenses pass (reported). `scanSource` now
adds `len(rec.Payload) + readRecordOverhead` per record (read.go:271 in the fix-skeptic's tree; the constant is at read.go:125 in
the tree now, verified, value 48). Every read path (`Read`, `ReadWait`, `Reader.Read`, the consumers' reads, `Records`, `Follow`)
reaches `scanSource` with the cap; `Scan` passes no cap and is unaffected. At the default cap a read returns 349,526 empty-payload
records (the loop stops at the first record where n x 48 reaches 16,777,216; 16,777,216 / 48 = 349,525.33, so record 349,526) or
342,393 one-byte records per call, against 1,000,000 and 16,777,216 before. Decisions 50, 58 and 79 (dated notes), scope.md and its
guarantee 7, the user guide, the `Config` and `ReaderOptions` comments, the audit records and a CHANGELOG `Changed:` line carry the
new rule. Tests: `read_overhead_test.go` (`TestMaxReadBytesChargesEveryRecordItsOverheadSoEmptyPayloadsAreBounded` at :27 and
`TestMaxReadBytesOverheadStillReturnsARecordWhoseOwnPayloadPassesTheCap` at :91) and the adapted existing tests in append_test.go
(16 to 100, and 20 to 2 x (10 + 48)).

Reproduction (reported by the fix-skeptic, not re-run by the recorder). Reverting the one accumulation line makes the new tests
fail: "Read(1, 0) of empty payloads returned 1000000 records, want 349526", and "33 records ... want 21". On the fixed tree, a
separate program on 1,000,000 empty records at the default cap: `Log.Read(1,0)`, `Reader.Read(1,0)`, `Consumer.Read(0)` and
`Consumer.ReadWait(0)` each returned 349,526 records; a `Reader.Read` from 349,527 returned 349,526 more, starting there;
`Reader.Records` and `Log.Follow` yielded all 1,000,000 in order with no loss. The built `quill` gave `cat` 1,000,000 lines,
`tail -n 5` the last five and `read -from 349520 -limit 10` ten lines. Allocation for `Read(1,0)` on 1,000,000 empty records:
216 MB and 1,000,000 records before, 70 MB and 349,526 after. `unsafe.Sizeof(Record{})` is 40, as the recommendation inferred. The
new tests also passed on linux/arm64 in an Apple container (the Linux x86-64 harness script was missing). Not run by the
fix-skeptic: `staticcheck` and `deadcode`.

Two points the fix-skeptic left (reported, neither blocks): the user guide (userguide.md:207) still says the cap "bounds the memory
of a read", although a bounded read allocates about 4 times the cap (70 MB against 16 MiB); and mutation-hardening.md said L2-53
was held with no fix written, which this update corrects. See "Residuals the fix-skeptics left" above.

## Pass L3b: live multi-process validation

Date 2026-11-03, against the tree at `a63bd6b` with the four L3b fixes on top (`136577c`, `53fcb36`, `d88c9c1`, `ba30fee`). The pass ran the multi-process harness (`hack/topology`) and dedicated drivers on macOS, on a
Linux box (ext4 and eight other filesystem configurations, one real power cut and crash-state replays) and in Apple containers; the record is [live-integration.md](live-integration.md). It produced 12 items: four were fixed
(L3b-4, L3b-3, L3b-6, L3b-14, each FIXED-VERIFIED), three were refuted, three were decided in round 1 and left unchanged, and **two are held below for the owner**, neither of which has a fix written. *Update 2026-11-03: the owner answered both the same day and both are now fixed (L3b-7 by documentation, in `7fa0744`; L3b-8 in `29f5c6c`); see "Update 2026-11-03 (pass L3c)" near the head of this file and the two sections below.*

## L3b-7: a second writer under another uid gets a bare permission error, not `ErrLocked` (low)

**Claim.** On a log created by one uid, a second writer or a writer-lock verb under a different uid fails with `open .../quill.lock: permission denied` (EACCES), not `ErrLocked`. Verified on Linux ext4: log
created with umask 000 and `Perm` 0666, writer live, `sudo -u nobody quill append`, `prune`, `roll` and `recover` all print the permission error and exit 1; as the owner they print `locked by another holder`.
`stat`, `check`, `read` and `register` as the other uid work. Nothing was modified after the refusal. Cause: `takeWriterLock` opens `quill.lock` with `O_RDWR|O_CREATE` (lock_unix.go:42) and returns the error
unwrapped, and the lock file is always 0600 (decision 51). `ErrLocked` is added only after a flock conflict. The documents say `ErrLocked` without this case: doc.go:10, userguide.md:249 (`no: ErrLocked` for
`append`, `roll`, `prune`, `recover`); only the `archive` verb documents the combined error (userguide.md:343).

**Why held.** Either fix is visible: wrapping the error changes what `errors.Is(err, ErrLocked)` returns for another uid (an exported error class, and the exit text of four CLI verbs), and documenting it changes the
wording of the `ErrLocked` contract. The behaviour itself is deliberate (decision 51: a reader-only user must not be able to take the writer's lock), nothing is damaged, and the configuration (a writer under a second uid on
a log it did not create) is unusual. CONFIRMED by 2 of 2 skeptics (by reading the code; the auditor's Linux run is the only observation).

**Recommended answer.** Document it, and leave the error a plain permission error that names `quill.lock`: add to the `ErrLocked` row of userguide.md:249 and to doc.go:10 that a writer under a different account
than the log's creator gets a permission error on `quill.lock` and not `ErrLocked`, and that a script detecting a live writer must treat both as "not this account's log or a writer may be live". Reason (inferred): for
`Open` an unreadable 0600 lock owned by another account almost always means someone else's log, and `ErrLocked` would say "a writer is live" when none may be; `archive` can treat it as live because it only refuses. The
runner-up is to wrap it as `archive` does, so the error matches both `ErrLocked` and `fs.ErrPermission`; that gives callers one test and the CLI the "this verb needs its lock" message, at the cost of the
misleading reading. 

**Fix written.** None. The auditor's proposal is the text above or the wrap; no code was drafted.

**Owner's answer 2026-11-03.** The owner took the recommended answer: document it. The error stays a plain permission error that names `quill.lock`; no code change. The runner-up, wrapping the error as `archive` does so that it matches both `ErrLocked` and `fs.ErrPermission`, is not approved.

**Disposition.** Fixed by documentation only in `7fa0744`; fix-skeptic FIXED-VERIFIED, 6 of 6 lenses pass (reported by the fix-skeptic). Behaviour, the exported error class and every CLI message are unchanged. What the commit does (verified by the recorder from the diff): the `Open` godoc (`log.go`) and `doc.go` say that a writable open under a different account than the log's creator fails with the permission error from opening the 0600 `quill.lock` (EACCES, matching `fs.ErrPermission`, naming the file), not `ErrLocked`, whether or not a writer is live, and that a caller detecting a live writer treats both as "not this account's log, or a writer may be live"; the user guide table row now reads `no: ErrLocked` with "(for another account, see below)" and a paragraph under it gives the message, exit 1 and nothing modified; scope guarantee 5 and the Locks and exit codes paragraph say the same; decision 51 carries the dated note "Stage 2 audit, owner-approved 2026-11-03, L3b-7"; CHANGELOG has a `Changed:` line. The `Open` godoc and scope say that `Archive` differs because it only refuses, and the reason is given as inferred (an unreadable 0600 lock owned by another account almost always means someone else's log, and `ErrLocked` would say a writer is live when none may be).

Two points differ from the recommended text, both from the fix-skeptic's objection to the first draft, which claimed the permission error for every verb. Root bypasses file permissions, so root gets `ErrLocked`; and `prune` with no retention flags reads `quill.policy` first, so an account that cannot read that file gets `quill.policy cannot be read; pass the retention flags explicitly` instead (`cmd/quill/write.go:213-222`, read by the fix-skeptic, not run). The text now says both.

**Test.** `open_lock_perm_test.go`, `TestOpenUnderAnAccountThatCannotOpenTheLockIsAPermissionErrorNotErrLocked`, through the existing `fsx.Sim` `Inject` seam, no root needed: a second `Open` under an unopenable lock returns an error that is not `ErrLocked`, matches `fs.ErrPermission` and names `quill.lock`; with the lock openable a second `Open` returns `ErrLocked`. It pins current behaviour, so it passes on the parent and cannot be shown red there. The fix-skeptic applied the rejected wrap as a mutation of `takeWriterLock` and the test failed with `error matches ErrLocked` (reported). The Linux run (Debian, ext4, uid `nobody` via `sudo`) gave the documented message and exit 1 for `roll`, `recover` and `prune`; `append` as `nobody` was not exercised (script slip), and root's `ErrLocked` was seen.

**Residual** (reported by the fix-skeptic, not held). `docs/reference.md` lines 100, 170, 186 and 207, generated from `cmd/quill/verbs.go:265`, still say "fails with ErrLocked beside a live writer". The sentence is true for an account that can open the lock, and the approved answer forbids changing CLI text, so it stays.

## L3b-8: a completely full filesystem makes every writable open of an existing log fail with ENOSPC (medium)

**Claim.** With exactly zero free bytes for the unprivileged user, `Open` of an existing log fails with ENOSPC, and so do the CLI verbs `prune`, `recover`, `roll` and `append`. `Open` fails before the writer's
retention, `MinFreeBytes` and `MaxBytes` sweeps can run, so the features meant to free space cannot run, and the log stays unopenable until something outside quill frees a block. The same condition inside a running writer
is retried by design (decision 60 d); the restart path has no equivalent. Cause: `load` sets `l.pendingPublish = true` unconditionally (load.go:456) and returns the error of `publishLocked()`; the flush then rewrites
`quill.tail` through a temp file, fsync and rename even when the marker on disk already equals the recovered state (flush.go:179-181), and that temp file needs a free block. By contrast a failed `quill.policy` write at `Open`
is tolerated and retried (log.go:448 and the policy writer).

Verified reproduction (a skeptic, Linux, 8 MiB tmpfs; the auditor also on a 16 MiB ext4 image): open with 64 KiB segments, `MinFreeBytes` 2 MiB, append 3,072 records of 1 KiB, close (head 3,072, 49 segments); fill the rest
until `df` shows 0 available; `Open` fails with `write .../.tmp-...: no space left on device` (exit 3 in the program), `quill prune` and `quill recover` exit 1 with the same error; truncate the filler by one 4 KiB block and `Open`
succeeds, and within 3 s the `MinFreeBytes` sweep takes the log from 49 to 17 segments (3,191,681 to 1,063,936 bytes). The harness hit it on its own in the full-disk scenario on xfs, f2fs and overlayfs ("Open after the
run: open .../.tmp-...: no space left on device"), and passed on ext4, btrfs and tmpfs only because the writer had not quite filled the disk when the verifier ran. CONFIRMED by 2 of 2 skeptics.

**Why held.** The fix changes what `Open` does on a full disk: it would succeed where it now fails, with the marker possibly not republished and `Err()` reporting the pending publish, which a caller that treats a
successful `Open` as "the log is fully published" would notice. It also touches the recovery path of `Open` (decisions 39, 60 d and 81), whose republish was made unconditional on purpose (the comment at load.go:451-455:
an empty log's marker is the zero value and `MarkerAge` read 0). Narrow precondition: free space of exactly zero for the unprivileged user (one free block avoids it; on ext4 the root-reserved blocks make it rarer),
an operator can free space outside quill, and no document promises that `Open` works on a full disk. Medium because it defeats the stated self-recovery on a full disk (scope.md guarantee 11, `MinFreeBytes`; the
CHANGELOG entry that a prune pass must not be skipped exactly when the disk is full; userguide.md:48-60), and it is the end state of the one resource a long-running deployment does exhaust.

**Recommended answer.** Fix it, in two steps and in this order. (1) When the recovered marker equals the one on disk and the data is durable, skip the rewrite at `Open`; this alone makes the reproduced case work, since
a clean restart on a full disk then touches no new block. Keep the rewrite for the empty-log case that the comment names (set `markerAt` without a write, or write once at creation) so `MarkerAge` stays right. (2) When
the republish is needed (parse-forward recovery moved the head, decision 39) and fails, treat it as a failed publish at flush: keep `pendingPublish` set, report it through `Err()` and the logger, return the log, and let
the flush and prune paths retry (decision 60 d) so the sweeps that free space can run. Step 2 is inferred to be safe because the data is already durable and the marker is a lower bound (decision 39); it is the part to
review hardest, since a reader of a stale marker sees an older head until the retry lands. The runner-up is step 1 alone, which closes the reproduced case and the harness failures and leaves a parse-forward recovery on a
full disk refusing; the other runner-up is documenting that `Open` needs one free block, which keeps the code and states the limit.

**Fix written.** None. The auditor's proposed test: an `fsx.Sim` that injects ENOSPC on the marker temp-file create at `Open`, requiring `Open` to succeed, the sweep to run, and the marker to be published once space
returns.

**Owner's answer 2026-11-03.** The owner took the recommended answer: fix it, in the two steps and in the order given. Step 1 first (no rewrite of `quill.tail` at `Open` when the marker is already right, `MarkerAge` kept right), then step 2 (a republish that is needed and fails is a pending publish, not an `Open` failure).

**Disposition.** Fixed in `29f5c6c`; fix-skeptic FIXED-VERIFIED, 7 of 7 lenses pass (reported by the fix-skeptic). The commit carries an earlier attempt's step 1 (cherry-picked) and a corrected step 2; the earlier attempt was objected to for the reason below.

- *Step 1* (verified from the diff). `load()` keeps the marker it decoded and sets `markerAt` from the clock (`load.go`), so `publishLocked` writes only when recovery found a different head, segment, offset or highest segment id. A cleanly closed log writes nothing at `Open`. The empty-log case the old comment named (marker is the zero value, `MarkerAge` read 0) is handled by setting `markerAt` without a write; `TestWriterMarkerAgeOfARestartedLogWithRecordsCountsFromOpen` pins a restarted log with records. The ladder: reuse of `publishLocked`'s existing change detection.
- *Step 2* (verified from the diff). When the publish is needed and fails, `load()` returns nil with `pendingPublish` set (an unexported `openUnpublished` is set; a failed log still fails `Open`). `Err()` wraps the cause, `Config.Logger` gets the `marker publish` Warn, a `Reader` sees the older head, and the flusher, `Close` and prune passes retry (decision 60 d, not weakened: a publish that fails in a running writer still stops a prune pass).
- *What the owner's text did not spell out.* With only the above, a prune pass stopped at the publish that precedes its unlinks, so the sweeps that were meant to free space still freed nothing; that was the first fix-skeptic objection. `deleteLocked` now, while the `Open`-time publish is pending, unlinks the chosen sealed segments below the segment the marker on disk names and keeps that segment and every later one (`retention.go`, decision 65 note). The marker is a lower bound that recovery parses forward from (decision 39); the directory sync before each unlink makes it durable first (the fix-skeptic traced a crash by hand; `TestACrashAtEveryStepOfASweepUnderAPendingOpenTimeMarkerReopensWhole` exercises it). This is an addition to the approved text, made to carry out its stated aim. It is flagged for the owner; removing it brings back the first objection.
- *Documents.* Decisions 39, 60, 65 and 81 carry notes dated "Stage 2 audit, owner-approved 2026-11-03, L3b-8"; scope, user guide (the full-disk paragraph, near `userguide.md:48-60`), architecture, `docs/reference.md` and the `recover` help text, the `Open` and `MarkerAge` godoc, and a `Changed:` CHANGELOG line say that `Open` can succeed with a marker publish pending, that a caller who took a nil error to mean "published" should check `Err()`, and that `MarkerAge` counts from `Open` when no marker was written. The decision 65 note records that `Stat().DiskFull` can read false after a clamped pass.
- *Tests* (`open_fulldisk_test.go`, 8). With the five source files reverted and the test file kept, 7 fail with `open: no space left on device` from `Open`; the eighth pins preserved behaviour and passes on the parent by design (reported). Mutating `>=` to `>` in the clamp fails the sweep test, and dropping the `!l.openUnpublished` guard fails `TestAPrunePassWhoseMarkerPublishFailsLogsNoRecoveryForTheUnlink` and `TestAMarkerPublishFailingAfterTheOpenTimeOneLandedStillStopsAPrunePass` (reported).
- *Fix-skeptic's attacks, as the owner asked for step 2* (reported). A reader of a stale marker: a `Reader` and `quill check` beside the pending writer saw head 2500 of a true 3072, HEALTHY, and after space returned the marker reached 3072, with no record lost or duplicated (head 3072, first 2049 after the sweep). CLI verbs on a real 8 MiB tmpfs at zero free bytes: `recover` and `roll` exit 0 on a clean log; with a stale marker `recover` exits 1 with a clear ENOSPC message but first cuts a torn tail (file 62358 to 62340 bytes, Warn `kind=torn`), and after a `prune` freed space `recover` exited 0 and wrote the marker; `prune` took 49 to 17 and 48 to 16 segments and exited 0; `append` exits 1 with a clear ENOSPC message. Crash at each point: the existing crash and fault-injection tests in the root package, the new crash tests and the full gate with `FUZZTIME=20s` passed (macOS and the Linux box). Live harness: `hack/topology.sh run -fill`, 16 MiB tmpfs, 90 s, on the final commit: ok, 725 writer restarts, 4 fills, no violations. The fixer's commit message says the harness was run on the earlier attempt only and gives a prune figure (49 to 25) that differs from the fix-skeptic's; they are separate runs and the recorder re-ran neither.

**Residuals** (reported by the fix-skeptic; none a regression of L3b-8, none held). (a) With the active segment over `SegmentBytes` and segment creation failing (a rotation due at `Open`), a prune pass frees nothing (sealed 6 to 6); a running writer in the same state does the same (decision 60 d), so it is a separate, pre-existing finding that leaves the "sweeps cannot free space" symptom reachable on a full disk after a failed rotation. (b) Only the sweep test guards the "keep the marker's own segment" rule; removing the rule did not fail either crash test. (c) The scope `Config.Logger` paragraph's list of the warnings `Open` gives omits the marker-publish Warn. (d) `docs/format.md:241` and `docs/design/build-prompt.md:133` still say the next open "republishes" the marker unconditionally (still true when the marker is behind; the build prompt is historical). (e) `MarkerAge` counts from `Open` while a publish is pending and `Err()` reports it. Only tmpfs was run after the fix; the xfs, f2fs and overlay cases where the harness first hit the problem were not re-run (inferred to be covered, since the cause is the same ENOSPC on the temp file).

## Round-1 decisions confirmed 2026-11-03

Pass A3r. Date 2026-11-03, against `main` at 955f83c with the seven fixes below on top. Pass A3 and the passes before it had
recorded these items as decided in round 1, or refuted, and had said the owner might want to confirm them before the
release (section "Not held, but worth a line" under pass A3 above, and [threat-model.md](threat-model.md)). The owner took
the recommended answer for each on 2026-11-03; A3-24 and A3-7 were approved later the same day. The recommendations are the
ones in the earlier text of this file and in threat-model.md. Each item keeps its earlier text where it is; this section adds
the answer and what became of it. The top-line counts and the coverage statement are in "Update 2026-11-03 (pass A3r)" near
the head of this file.

| id | sev | one line | owner's answer 2026-11-03 | disposition |
|----|-----|----------|---------------------------|-------------|
| A3-3 | medium | `ls`, `consumers` and `check` print file-derived names unescaped | change: quote file-derived strings | fixed 116726b, FIXED-VERIFIED (7 of 7 lenses) |
| A3-23 | low | `roll` and `recover` create a log at a mistyped `-dir` and exit 0 | change: refuse unless `quill.format` exists | fixed 18145b9, FIXED-VERIFIED (7 of 7 lenses) |
| A3-24 | low | `-json` passes bidi and format controls raw; text mode escapes them | change, approved later the same day: escape set only | fixed d15ac45, FIXED-VERIFIED (7 of 7 lenses); `time` range check not approved, held |
| A3-29 | medium | no detection of filesystems where `flock` is not coherent | keep the behaviour, document by name | documented d916114, FIXED-VERIFIED (6 of 6 lenses) |
| A3-31 | low | ConsumerTTL staleness is derived from the clock | keep the behaviour, document it | documented 1cb2c93, FIXED-VERIFIED (6 of 6 lenses); one deviation from the approved text, below |
| A3-32 | low | MinFreeBytes overrides protected consumers | keep the behaviour, document it | documented d09e975 with one pinning test, FIXED-VERIFIED (6 of 6 lenses) |
| A3-7 | low | a symlinked `consumers/` directory is followed | keep, with one clarifying sentence (approved later the same day); no code change | wording b765290, FIXED-VERIFIED (6 of 6 lenses); code change not approved, held |
| A3-4 | medium | a non-consumer file in `consumers/` makes retention protect everything | keep as decided in round 1 | deliberately not changed; recorded here only |
| A3-9 | low | a Reader takes the mode of its files from `quill.tail` | keep as decided in round 1 | deliberately not changed; recorded here only |
| A3-11 | low | `quill prune` acts on `quill.policy` as trusted configuration | keep as decided in round 1 | deliberately not changed; recorded here only |
| A5-7 | low | `quill stat`, `quill ls` and `Reader.Stat` do not show a mid-file damaged sealed segment | keep as decided in round 1 | deliberately not changed; recorded here only |
| A1-1 | medium | `quill.tail` is decoded by two parsers; make it canonical-only | keep as decided in round 1 (decision 105) | deliberately not changed; recorded here only |
| A1-5 | low | two hand-optimised helpers save less than the noise of the syscalls around them | keep as decided in round 1 (allocation pass) | deliberately not changed; recorded here only |
| A1-30 | low | every inline flush stalls the Appends behind it | keep as decided in round 1 (decision 13) | deliberately not changed; recorded here only |
| A1-54 | low | the Reader fails when a cap deletes the marker's own segment mid-view | keep as decided in round 1 (decision 101) | deliberately not changed; recorded here only |

### The seven items fixed in this pass

Each is recorded as Claim, Grounds, Warrant, Backing, Rebuttal, Verdict and Disposition. The grounds and reproductions are
the pass A3 ones (reported, from [threat-model.md](threat-model.md), where the full text is); the recorder ran nothing.

#### A3-3: `ls`, `consumers` and `check` print file-derived names unescaped (medium, security)

- **Claim.** A file or directory name controlled by an untrusted party is printed raw by `quill ls`, `quill consumers` and
  `quill check`, so a newline forges a row and an ESC byte reaches the operator's terminal. Payloads were already quoted.
- **Grounds.** `ls` printed subdirectory names with `%s`; `consumers` printed names taken from the directory listing with no
  `validName` filter; `check` printed paths built from them. Reproduction (the auditor, macOS, reported): a subdirectory named
  newline, `ESC ]0;PWNED BEL`, `ESC [2J` holding a minimal `quill.format` and `quill.tail` made `ls | cat -v` print a forged
  row, and a file `consumers/x ESC[31mRED ESC[0m` made `consumers` and `check` print raw ESC bytes.
- **Warrant.** Bytes below 0x20 written to a terminal are interpreted by it, and a name is the same class of untrusted input as
  a payload.
- **Backing.** scope.md "Readable output" and SECURITY.md "Hostile or damaged files on disk"; decision 35 (the payload escaper).
- **Rebuttal.** The attacker needs create access in the parent directory or in `consumers/`, which in the default 0700 setup is
  only the operator's own uid; `-json` was already safe for names.
- **Verdict.** CONFIRMED (round 1 had decided it as prior-deliberate; owner-approved change 2026-11-03).
- **Disposition.** Fixed in 116726b. In text output every string derived from the disk goes through one helper, `safeText`:
  unchanged if it is valid UTF-8 and every rune is printable, otherwise quoted by the existing payload escaper
  `appendQuoted`. `appendQuoted` now writes a non-UTF-8 byte as `\xNN` (it printed U+FFFD); payloads reach it only after
  `utf8.Valid`, so their output is unchanged. `-json`, the golden bytes and `docs/reference.md` are unchanged; decision 35
  carries a dated note and the CHANGELOG a "Fixed:" line. Fix-skeptic: FIXED-VERIFIED, 7 of 7 lenses (reported). It showed the
  new test red with the parent's printing and green now; ran the built binary on macOS and, on linux/arm64 in an alpine
  container, with names holding a newline, ESC, a bidi override and invalid UTF-8, and saw only quoted names; and grepped every
  `Fprint*` and `errorf` in `cmd/quill`. Skeptic's notes: the "stray segment-like file" in the test is ignored by `check` and
  asserts nothing; whole-string quoting would also quote a legitimate multi-line error, and no CLI path prints one. The auditor's
  second half (skip or flag names that fail `validConsumerName`) is A3-4 and is not taken.

#### A3-23: `roll` and `recover` create a log at a mistyped `-dir` and exit 0 (low, degradation)

- **Claim.** Pass A3 said `recover`, `prune`, `roll` and `append` create a log, with every missing parent, at a mistyped
  `-dir` and exit 0, and leave an empty `quill.lock` when `-dir` is a populated non-log directory. Narrowed on 2026-11-03 by a
  check of c768634 (the recorder's, reported): `ls`, `stat`, `check`, `read`, `consumers` and a bare `prune` already exit 1
  and create nothing, `archive` was fixed by A3-14, and `roll` and `recover` still created the log.
- **Grounds.** `Open` calls `mkdirAll` and `create()`, and the CLI's writer path does not require an existing log. Reproduction
  (the auditor, macOS, reported): `quill -dir data/ordrs recover` printed "recovered: head 0, first 1, 1 segments", exit 0,
  and created the log; `quill -dir deeper/a/b/c roll` created four directories.
- **Warrant.** Creating a log is documented for `Open` and for `append`, not for verbs whose job is to act on an existing log.
- **Backing.** scope.md describes `recover` as "open writable, repair a torn tail, republish quill.tail, close".
- **Rebuttal.** The new logs hold no data and the lock file is empty; a user who wants `append` to create a log relies on it.
- **Verdict.** CONFIRMED (round 1 had it as prior-refuted, A1-49; owner-approved change 2026-11-03).
- **Disposition.** Fixed in 18145b9. `roll`, `recover` and `prune` now check for `quill.format` in `-dir` before opening the
  log; if it is missing they print `quill: not a log: <abs dir> has no quill.format`, exit 1 and create nothing, not even the
  lock file; usage errors still exit 2 first. `append` creating the log on first use is intended and unchanged (decision 62; the
  userguide now says so). The fix-skeptic's run of the parent binary found that `prune` with a policy flag (`-retention`) also
  created the log, so `prune` is covered. `commit`, `register`, `unregister` and `archive` already refused. The check is an
  `Lstat` of `quill.format`, not `OpenReader`, so `recover` still repairs a log whose `quill.tail` was lost. Decision 53 carries
  a dated note; the userguide, scope.md and the reference summaries are updated; the CHANGELOG has a "Changed:" line saying
  `roll` and `recover` at a directory that is not a log now exit 1 where they exited 0. Fix-skeptic: FIXED-VERIFIED, 7 of 7
  lenses (reported): the new test failed 15 of 15 subtests with the parent's code restored; every writable verb was run against
  a missing, an empty, a regular-file and a populated directory with both binaries and on linux/arm64. Limits: `hack/gate.sh`
  itself and its fuzz step were not run; `append` at a populated non-log directory still refuses and leaves an empty
  `quill.lock`, listed in the commit as out of scope.

#### A3-24: `-json` passes bidi and format controls raw while text mode escapes them (low, security)

- **Claim.** `-json` passes Unicode format and bidi controls through raw in the `payload` string while text mode escapes them;
  the `time` field is not RFC 3339 outside years 0000-9999.
- **Grounds.** `appendJSONString` escaped only controls, DEL, `<`, `>`, `&`, C1 and U+2028/2029, while `appendQuoted` uses
  `strconv.IsPrint`. Reproduction (a pass A3 voter, macOS, reported): payload U+202E U+200B U+202E printed as escapes in text and
  as raw bytes in `cat -json`; `At` 253402300800 printed "10000-01-01T00:00:00Z".
- **Warrant.** The two output modes disagree about what is safe.
- **Backing.** format.md and decision 35 (DEL and C1 are the owner-approved deviation from `encoding/json`).
- **Rebuttal.** By the letter of the documentation JSON escapes exactly the listed set; bidi controls are not terminal escapes;
  the raw form decodes losslessly; how a bidi-aware viewer renders the raw bytes was not verified.
- **Verdict.** REFUTED in pass A3 (1 CONFIRMED to 2 REFUTED), kept here with that reason: it breaks no stated guarantee. The owner
  approved the escape-set change on 2026-11-03 regardless.
- **Disposition.** Fixed in d15ac45 for the escape set only. `-json` escapes every Unicode format character (category Cf, 170
  code points) as `\uXXXX`, with a surrogate pair above U+FFFF. All 170 are rejected by `strconv.IsPrint`, pinned by a test, so
  text and JSON agree. The decoded value of every payload is unchanged and a plain payload prints byte-identically; other
  non-printable code points (U+00A0, private use, unassigned) stay raw. Decision 35 carries a dated note, and the CHANGELOG a
  "Changed:" line says the raw bytes change and the decoded value does not. The `time` range check was not approved and is held
  below. Fix-skeptic: FIXED-VERIFIED, 7 of 7 lenses (reported): five tests failed with the parent's `print.go` restored; a log
  of every non-surrogate code point run through `quill cat -json` before and after showed identical decoded payloads and exactly
  170 characters changed; the Linux evidence is arm64. Residual: row B5 of threat-model.md's boundary table still states the
  2026-10-31 behaviour.

#### A3-29: no detection of filesystems where `flock` is not coherent (medium, gap)

- **Claim.** Quill does not detect filesystems on which `flock` is not coherent between holders, and the documentation did not
  name them. On a virtiofs share a host writer and a container writer both opened the directory without `ErrLocked`.
- **Grounds.** `TryLock` is a bare `syscall.Flock` and no filesystem-type check exists in `Open`. Reproduction (the auditor, macOS
  host and an Apple container, reported): both sides acknowledged 30 durable flushes on one directory and `quill check` printed
  UNHEALTHY ("bad frame"); a positive control on Linux ext4 with `unshare -rm` and `mount --bind` returned `ErrLocked`.
- **Warrant.** `flock` from a guest kernel is guest-local, so holders on two kernels never see each other.
- **Backing.** scope.md "Filesystems", decision 51, guarantee 5; live-integration.md for the virtiofs double writer.
- **Rebuttal.** The limitation was documented in outline; a same-kernel bind mount is exclusive, so the "at the operator's risk"
  sentence was too pessimistic. NFS, SMB and FUSE were not available to the auditors, so those claims rest on the documents.
- **Verdict.** CONFIRMED as a documentation gap (round 1: prior-deliberate).
- **Disposition.** Documented in d916114; the behaviour is unchanged and no detection or `fstatfs` warning is added (the owner
  approved "document it by name"). The README, userguide, scope.md, the godoc of Open, Config.Dir and the package, SECURITY.md,
  decisions 51 and 62 (dated notes) and the CHANGELOG say that the single-writer lock and a Reader's view need a local filesystem
  (ext4, xfs, btrfs, APFS) written by one host or kernel; that NFS, SMB/CIFS, FUSE and directories shared between kernels
  (virtiofs, 9p, container bind mounts of a host directory) are unsupported and not detected; and that two writers there can both
  believe they hold the lock. The bind-mount sentence is corrected. Fix-skeptic: FIXED-VERIFIED, 6 of 6 lenses (reported); no
  test can be red for a documentation change, and each factual claim was checked against live-integration.md and the lock code.
  Residual: 9p is listed without a measurement, and the NFS, SMB and FUSE statements are a documented limit, not a measurement.

#### A3-31: ConsumerTTL staleness is derived from the clock (low, degradation)

- **Claim.** With ConsumerTTL set, a forward wall-clock step makes every protected consumer stale at the next prune pass, so
  its unread segments are deleted at the clock guard's pace (about 25% of sealed segments per pass).
- **Grounds.** retention.go: stale means the offset file's modification time and the record's `At` are both at least ConsumerTTL
  old against the writer's clock; the guard limits only the age sweep. Reproduction (the auditor, in-package fake clock of +400
  days, 21 sealed segments, reported): ConsumerTTL=1h took the log from 21 to 1 segment over ten passes; ConsumerTTL=0 deleted
  nothing. A real `date -s` was not used.
- **Warrant.** Staleness uses the same wall clock as the age sweep, with no skew guard of its own.
- **Backing.** scope.md "A wrong clock" and decision 44 ("slows the damage ... does not remove the hazard"); guarantee 12.
- **Rebuttal.** The hazard class is documented, ConsumerTTL is opt-in, and a step of at least ConsumerTTL is needed.
- **Verdict.** CONFIRMED as a documentation gap (round 1: prior-deliberate).
- **Disposition.** Documented in 1cb2c93; no code change. The Config.ConsumerTTL godoc, userguide, scope.md, architecture.md,
  decisions 44 and 45 (dated notes, "at least as old" wording) and the CHANGELOG now say that staleness comes from the offset
  file's mtime and the record's `At` against the writer's clock, so a clock step, a restore that resets mtimes, a copy that does
  not keep them or a filesystem with wrong timestamps can make a live consumer look stale or a dead one look live; a
  future-stamped file or record is never stale; the clock guard limits only the age sweep. **Deviation from the approved text:**
  it said "Protect makes a consumer exempt from the TTL". retention.go skips an unprotected consumer before the TTL logic and
  decision 57 says ConsumerTTL applies to protected consumers only, so the sentence is backwards; the documentation states the
  code's rule and the CHANGELOG says so. The owner should confirm that reading. Fix-skeptic: FIXED-VERIFIED, 6 of 6 lenses
  (reported), who checked every new claim against retention.go and found the deviation correct. Nit left: the userguide's
  `cp` and `rsync` "without -p/-t" wording is slightly loose.

#### A3-32: MinFreeBytes overrides protected consumers (low, degradation)

- **Claim.** With MinFreeBytes set, another process's disk use makes this log delete its own sealed history, overriding
  protected consumers and bypassing the clock guard, and the code estimates each deletion's gain instead of re-measuring.
- **Grounds.** retention.go: one `statfs`, then `free += sealed[n].size` per deleted segment; `capsLocked` runs after the guard.
  Reproduction (the auditor, in-package fake `statfs`, reported): 21 segments, a protected consumer at 1, `statfs` faked to 0, one
  pass left 1 segment and `dropped` 20. A real full disk was not run.
- **Warrant.** A neighbour's disk use is an unbounded deletion trigger for this log's sealed data.
- **Backing.** scope.md "Disk safety" and guarantee 11; decisions 24 and 25 ("caps override consumer protection").
- **Rebuttal.** This is the documented design; on a real filesystem the unlinks free real bytes, so deletion stops near the floor.
- **Verdict.** CONFIRMED as a documentation gap (round 1: prior-deliberate).
- **Disposition.** Documented in d09e975; no production code changed, and the auditor's re-measure-after-each-unlink proposal is
  not taken (the owner approved "keep the behaviour, document it"). The MinFreeBytes, ProtectedConsumers and Consumer.Protect
  godoc, doc.go, userguide, scope.md, architecture.md, decisions 25 and 57 (dated notes) and the CHANGELOG state the rule for each
  cap: the age sweep stops at the first segment a protected consumer has not passed; MaxBytes, MaxRecords and the MinFreeBytes
  floor override protection, so a consumer can lose records, and the Warn naming the consumer is how that is seen. A new test,
  `TestMinFreeBytesDeletesSegmentsAProtectedConsumerHasNotRead`, pins it. Fix-skeptic: FIXED-VERIFIED, 6 of 6 lenses (reported);
  the test fails when the MinFreeBytes deletion loop is disabled. `staticcheck` and `deadcode` were not run by that skeptic.

#### A3-7: a symlinked `consumers/` directory is followed (low, security)

- **Claim.** Only the final component of a state file's path is opened with `O_NOFOLLOW`, so a symlinked `consumers/` makes
  `register` create files in the link's target, and `recover` or `Open` remove `.tmp-*` files older than an hour there.
- **Grounds.** `OpenFile` applies `O_NOFOLLOW` to the last element only; `removeTempsBefore` lists and removes in the resolved
  directory. Reproduction (verified by a pass A3 voter on macOS and Linux, and again by the fix-skeptic on macOS and on
  linux/arm64; reported): with `consumers/` replaced by a link to an empty directory, `register -name notes` exited 1 with ELOOP
  after leaving `notes` and `notes.lock` in the target, and `recover` removed a 2020-dated `.tmp-abc` there.
- **Warrant.** The kernel follows intermediate symlinks and no code `Lstat`s the `consumers/` component.
- **Backing.** SECURITY.md "Symlink substitution"; decision 51 ("every state file ... is opened without following symlinks").
- **Rebuttal.** It needs write access to the log directory, which the threat model excludes, and no stated guarantee is broken
  once the rule is read as covering the final component.
- **Verdict.** REFUTED in pass A3 (1 CONFIRMED to 2 REFUTED), kept with that reason: a hardening gap, not a defect against a
  stated guarantee. The mechanism itself reproduced.
- **Disposition.** Wording only, b765290, approved later on 2026-11-03: decision 51 (dated note) and SECURITY.md's "Symlink
  substitution" paragraph now say the rule covers the final component of each state file's path, and that a log directory (and
  its `consumers/`) that an untrusted party can replace with a symlink is outside the threat model, as write access to the log
  directory is; the CHANGELOG has a "Changed:" line. No code, test, flag or symbol. The code hardening is not approved and is
  held below. Fix-skeptic: FIXED-VERIFIED, 6 of 6 lenses (reported); only `CHANGELOG.md`, `SECURITY.md` and `decisions.md`
  differ from the parent. Nit: the decision 51 note is long.

### The eight decisions kept unchanged

Nothing in code or documents changed for these; the owner confirmed the round-1 decision and this section is the record. The
claims and the round-1 reasons are in [threat-model.md](threat-model.md) (A3-4, A3-9, A3-11),
[silent-degradation.md](silent-degradation.md) (A5-7) and [a1-round2-findings.md](a1-round2-findings.md) (A1-1, A1-5, A1-30,
A1-54). Each carries the verdict it had (prior-deliberate or prior-refuted, not voted) and the disposition "deliberately not
changed". The eight were not re-examined in this pass (see the coverage statement).

| id | claim | why it stays | proposal not taken |
|----|-------|--------------|--------------------|
| A3-4 | any non-consumer file in `consumers/` (`.DS_Store`, an editor backup) makes retention protect everything, with no log line | the fail-safe for an unparseable consumer file is decision 41; a party who can write `consumers/` could pin segments with a real consumer anyway, and the caps still override | ignore names that fail `validConsumerName` in retention, report them in `check` and the Logger |
| A3-9 | a Reader takes the mode of its files from `quill.tail`'s permission bits, so a 0666 marker makes `archive` write 0666 copies | owner-approved earlier as A1-85 and decision 51; the writer reapplies Config.Perm at its next flush and the directory mode limits the exposure | clamp the derived mode and intersect it with the umask, or add `ReaderOptions.Perm` |
| A3-11 | `quill prune` acts on `quill.policy` as trusted configuration, so a stale or edited policy can delete all sealed segments in one pass | decisions 68, 87, 29 and 99; a party who can write the policy could delete the segments directly, and invalid values fail closed (A1-91 fixed in 5c603fc) | print the policy about to be applied and parse it strictly for destructive action |
| A5-7 | `quill stat`, `quill ls` and `Reader.Stat` compute records and holes from first frames only, so a segment damaged mid-file shows healthy until a read or `quill check` | decision 47 (no scan on a Reader) and decision 38; `check` and a read detect it and exit non-zero | a documentation note on those three that `Records` and `Holes` do not verify segment bytes and that `check` does |
| A1-1 | `quill.tail` is decoded by a fast decoder and by the strict one; make it canonical-only and delete `strictjson` (about 170 non-test lines) | decision 105: the fast decoder exists because the strict pipeline cost about 30 allocations and 1.5 KB for a 40-byte file on every poll; the round-1 text argues the cost of the second parser is worth it, and canonical-only would make the marker accept only the writer's exact bytes | delete the strict decoder |
| A1-5 | two hand-optimised helpers (atomicfile.go and fsx/os.go) save about 63 ns of a 5 ms flush | the allocation pass of round 1 | replace them with the plain forms |
| A1-30 | every inline flush holds the log's lock through write and fsync, stalling the Appends behind it | decision 13: the stall is accepted; moving the fsync outside the lock is a measured follow-up | move the fsync outside the lock |
| A1-54 | the Reader fails on the marker's own segment when a cap deletes it between the marker read and the open | decision 101 makes the error deliberate; the window is narrow (inferred; not caught live) | return a gap instead |

## Held after pass A3r

Three things the owner did not approve on 2026-11-03. None has a fix written. The recommended answers are the recorder's
reading (inferred) and are recommendations, not decisions.

### A3-7 (code hardening): open `consumers/` without following a symlink

**Claim.** A symlinked `consumers/` directory is followed by `register`, `recover` and `Open` (see "A3-7" above). The owner
approved the wording of decision 51 and SECURITY.md, not a code change.

**Why held.** Opening `consumers/` once with `O_DIRECTORY|O_NOFOLLOW`, or `Lstat`-ing it in `create()` and in every consumer
operation, would make a log whose `consumers/` is a symlink refuse to open where it opens today (inferred: someone may have put
consumer state on another volume that way). That changes behaviour and needs the owner's decision.

**Recommended answer.** Leave the code. The exposure needs write access to the log directory, which the threat model excludes,
and the sentence now says so. If the owner wants defence in depth, the smaller change is the `Lstat` in `create()` and in the
consumer path helper, with a test per verb against a symlinked and a real `consumers/`.

**Fix written.** None. The auditor's proposal is in threat-model.md, A3-7.

### A3-24 (`time` field): range-check the printed time

**Claim.** The `time` field is `time.Unix(rec.At, 0).UTC()` formatted as RFC 3339 with no range check: `At` 253402300800
prints "10000-01-01T00:00:00Z" and `At` MinInt64 prints the same string as MaxInt64 ("292277026596-12-04T15:30:08Z"); the `at`
field stays exact. `At` outliers need a hand-crafted file, because the writer's clock is the only source of `At` (reported,
threat-model.md, A3-24).

**Why held.** Printing `time` as null (or omitting it) outside years 0000-9999 changes the `-json` shape for those records, and
the text-mode time column has the same format, so it would change that too. The owner approved the escape set, not this.

**Recommended answer.** Do not change. The `at` field is exact and is the machine-readable value, text mode has the same
unchecked format, and the case needs a crafted file. If the owner wants it, print `time` as null outside 0..253402300799 in
`-json` only and document it in format.md next to the `time` definition.

**Fix written.** None. The auditor's proposal is in threat-model.md, A3-24.

### A3-26: unlinking `quill.lock` under a live writer lets a second writer open

**Claim.** Removing or replacing `quill.lock` while a writer holds it lets a second writer open the log and overwrite the
first's records, and `check` stays HEALTHY. Pass A3 refuted it (two skeptics, out of the threat model and a stated design rule);
the mechanism was reproduced on macOS and Linux (reported, threat-model.md, A3-26).

**Why held.** Any further change would alter the lock protocol or add a second lock (the auditor's second option, a `flock` on
the log directory's own descriptor), which changes who can open a log in a shared directory. The owner did not approve any
change.

**Recommended answer.** No further change now. Since pass A3, 6acc23e (A5-16) compares `fstat` of the held lock file with
`lstat` of `quill.lock` before each flush and prune pass and fails the writer (`ErrFailed`, one Error log) when the file is
missing or different (read from the commit message and diffstat; the recorder did not run it). That is the auditor's first
option, applied to the first writer, so what is left (inferred) is the interval until its next flush or prune pass and the
second writer's own records. The record of A3-26 gets a dated note about 6acc23e by hand after this batch; this file does not
state it as the disposition. The sentence "never delete `quill.lock`" suggested in the earlier text is the cheapest remaining
step.

**Fix written.** None for A3-26 itself; 6acc23e is the related fix for the first writer.

## Held after pass A7

Pass A7 (gap audit, 2026-11-03; the full record is [gap-audit.md](gap-audit.md)) fixed seven items and held three. A7-15 was
held for an owner choice, made 2026-11-03, and then fixed in 08d7f77 (see its section); that makes eight fixed. A7-18 and A7-19 each have a written fix that the fix-skeptic rejected on one objection; neither commit
is merged into this branch (verified: `git merge-base --is-ancestor` fails for both, and both objects exist in the
repository). They are listed here so the written fix and the objection are not lost.

### A7-15: figures from the reference remain in review.md and decisions 3, 15 and 32 (medium)

**Claim.** Decisions 70 and 90 say every figure in review.md and scope.md that came from the reference implementation was
replaced by quill's own measurement or softened to "the reference measured" without numbers. review.md still states the
reference's line counts (`review.md:3, 18-20, 63, 64, 71, 94`) and ratios (finding 1, `review.md:61`), and decisions 3, 15 and
32 (`decisions.md:11, 23, 40`) still carry its timings, rates and framing size. Verified by reading and grep at 34947b2, and by
both skeptics (CONFIRMED, 2 of 2).

**Why held.** The fix changes what the project says publicly about its origin and which numbers it stands behind, and rewrites
three recorded decisions. Decision 70 allows "or softens", so keeping the figures is a legitimate owner choice; but then
decision 90's sentence is false and has to change. review.md is also linked from the README as design documentation and
describes the reference's architecture in detail, which the owner may want trimmed for the same reason.

**Recommended answer.** Delete the numbers before the repository is public. In review.md replace the line counts with
orders of magnitude or remove them, and drop the absolute ceiling at `:94` in favour of quill's own line count. Reduce finding
1 and decisions 3, 15 and 32 to "the reference measured X faster or slower" with no figures, pointing at
`benchmark-baselines.md` for quill's own. Reword decision 90 to say what was actually done. Optionally trim the review.md
table that describes the reference so it carries no more than the comparison needs. If the owner instead accepts the figures,
keep them and correct decision 90 only.

**Owner's answer, 2026-11-03.** Approved the recommended answer: delete the reference's numbers; documentation only. The
alternative (accept the figures and correct only decision 90) was not approved.

**Fix commit.** 08d7f77 on `main` (verified: it is the tip, with ee220c8, the rejected first attempt, as its parent). Disposition: fixed.

**Fix-skeptic outcome.** Attempt 1 (ee220c8) rejected on the two defects named under "Fix written"; attempt 2 (08d7f77)
FIXED-VERIFIED, 6 of 6 lenses pass (reported, the fix-skeptic's evidence; details in [gap-audit.md](gap-audit.md), section A7-15).
State of Y128 and decision 90: Y128 is met by deletion; decision 90 says what was done and its row has the same five fields as the
other rows. Residue: `review.md:74` keeps "minutes for 100+ GB of never-expiring log", which the skeptic read as a scale
scenario and not a reference figure (inferred); the owner may delete it.

**Fix written.** Branch `stage2/A7-15-a1` (rejected by the fix-skeptic on two defects: decision 90's row had a duplicated cell, and a reference-derived framing range survived in `scope.md` and `format.md` while decision 32 gave another), then `stage2/A7-15-a2`, which cherry-picks it and corrects both; fix-skeptic outcome for attempt 2: FIXED-VERIFIED (above).
Figures found by search of every tracked file outside `docs/audits/` and `.claude/`, and what became of each:

- `review.md`: the reference's line counts in the header, the table (the column was dropped; the rows are kept), findings 3,
  4, 6 and 11 and the verdict's "at most about 5,400 of its 9,000" ceiling: removed or turned into "thousands of lines" or
  "a large share". The ceiling is replaced by quill's own count, 5,370 lines (`wc -l` over the root package's non-test `*.go`
  files, 2026-11-03), and the "Counts" risk bullet now says the one line count is quill's. Finding 1's ratios and finding
  13's "a few microseconds" reduced to "faster or slower"; finding 2's "230-byte topic limit" to "a topic-name length limit".
- decisions 3 (85% of read CPU), 5 (3-5 times), 7 (three of thirteen limitations), 15 (timings, rates, segment and frame
  size of the benchmark), 16 (230-byte limit), 20 (3,300 lines), 32 (13.9-17.9 bytes per record): the figure removed, the
  conclusion kept, a dated note on each. Decision 21 said the scope.md orientation figures came from the reference; they
  are quill's baselines, so its sentence was corrected. Decisions 70 and 90: dated notes; 90 now says what was done.
- Framing figure, corrected in attempt 2: the attempt-1 text left "about 14 to 18 bytes per record" in `scope.md:153, 291` and
  `format.md:51` and put "13 to 17" in decision 32, and the fix-skeptic rejected it: the 14 to 18 matched the reference's
  old range, not a quill measurement, and decision 32 disagreed with it. All four places now carry one figure, computed
  from quill's own `frameLen` (`codec.go:64`) over payloads of 0 to 16,383 and 16,384 bytes, sequences from 1 to about two
  million and a current Unix-second timestamp: 12 to 18 bytes per record (12 for a sequence under 128 and a payload under
  128; 14 to 18 for sequences between 16,384 and 2,097,151; one more byte per extra varint byte beyond that). Verified by a
  throwaway test in the scratch area, 2026-11-03. The baselines' 128-byte records cost 145 bytes, inside the range for their
  sequence numbers' size. The 64 MiB, 1.5 GiB and 1 GiB limits
  (decision 16, `format.md`) are quill's own constants now, pinned by tests. "Several fsyncs per durable flush" (finding 8)
  and finding 6's "one known limitation" concern the reference's structure, not a measurement. `scope.md` holds only quill's
  own figures and the qualitative "the reference ... " statements, which carry no number.
- Not classifiable: none.

### A7-18: no upgrade, migration or rollback note for an operator (low)

**Claim.** Nothing in the operator documents says what upgrading a live log between releases requires or whether the old
binary can be put back. The evidence exists only in `format-compatibility.md` and `hack/skew.sh`. The user guide never
mentions `ErrFutureVersion`. Verified by search; 2 skeptic votes (CONFIRMED, PLAUSIBLE), verdict PLAUSIBLE.

**Why held.** The first fix was rejected. 08553bd adds a 14-line "Upgrading and rolling back" paragraph at the end of user
guide section 9 and a CHANGELOG line, and the fix-skeptic found one error in it: it says `format-compatibility.md` records
"seven announced differences in edge cases (for example prune -force after register -no-protect, and the mode of quill.lock
and of archive files)". In that audit "seven" counts decoder changes and fuzz divergences (lines 63-65 and 251), the three
examples are not among them, its own list of differences (lines 84-90) is longer, and the `quill.lock` mode change is in
CHANGELOG.md:69 and came after the audit. Everything else held: the `ErrFutureVersion` behaviour, the calls that re-read
`quill.format`, the exit code, the `skew.sh` usage, the link and the gate (gofmt, vet, `go test -race`). Whether an operator
note is wanted at all is also an owner call, since no specification requires one while the format is unchanged.

**Recommended answer.** Add the note, with the one correction: drop the count or say "several", point the reader to the
CHANGELOG Unreleased section, and name only differences that the audit or CHANGELOG records, without crediting the audit with
the lock mode. A fix-skeptic should then re-run on the corrected commit.

**Fix written.** 08553bd (docs/userguide.md section 9 and CHANGELOG.md), not merged; it needs the correction above.

**Hand finish 2026-11-03 (the assistant, not the owner; no fix-skeptic re-run).** The corrected text is merged as e2f02b6: cherry-picked from 08553bd onto the A7 tip (CHANGELOG and user guide conflicts with A7-20 resolved by keeping both), and the sentence the fix-skeptic objected to now reads "The differences between the two versions are listed under \"Unreleased\" in the CHANGELOG, which you should read before upgrading": no count and no examples. The recommended answer asked for a fix-skeptic re-run and said whether the note is wanted is an owner call; neither happened. The commit is local and unpushed; `git revert e2f02b6` removes it.

### A7-19: no install path documented; a go-installed binary reports `dev` (low)

**Claim.** README, user guide and introduction never say how to obtain the `quill` command or the library, and a binary built
with `go install` prints `quill dev` because the version is stamped only by `build/quill`'s `-X` flag
(`cmd/quill/main.go:7-8`, no `ReadBuildInfo` fallback; inferred from the code, not run, since there is no remote and no
network). Release archives carry README, CHANGELOG, SECURITY and LICENSE if present, not the user guide, format or reference
(`build/quill:69-71`). Verdict PLAUSIBLE (2 of 2 PLAUSIBLE).

**Why held.** The first fix was rejected. 1a701fb adds an "Installing" section to README.md and a CHANGELOG line, and the
fix-skeptic found one error: README states as fact that the archives hold LICENSE, but `build/quill:70` copies it only if it
exists and the tree has none (`README.md:113-114` and decision 22 say no LICENSE while the repository is private); the fixer
dropped the finding's own "if present". The `quill dev` statement and the omission of `docs/` from the archives were verified
correct, the gate passes and the exported API is unchanged. The install text can only be true once the repository is public,
which is an owner decision, and adding a `debug.ReadBuildInfo` fallback would change `quill version` output and was not
proposed.

**Recommended answer.** Keep the section and fix the one sentence: say the archives carry LICENSE once one lands, or drop it
from the list. Leave `quill version` as it is. Merge only after a fix-skeptic re-run, and only if the owner wants install text
before the flip.

**Fix written.** 1a701fb (README.md and CHANGELOG.md), not merged; it needs the correction above.

**Hand finish 2026-11-03 (the assistant, not the owner; no fix-skeptic re-run).** The corrected text is merged as ffae168: cherry-picked from 1a701fb, and the archive-contents sentence now reads "README, CHANGELOG and SECURITY (and LICENSE once the repository has one)", which is what `build/quill:69-71` does. The recommended answer said to merge only after a fix-skeptic re-run and only if the owner wants install text before the flip; neither happened. The commit is local and unpushed; `git revert ffae168` removes it.

## Held after pass A1d

Pass A1d (adversarial audit of the v0.1.0..HEAD diff, 2026-11-03; the full record is [a1-diff-scoped.md](a1-diff-scoped.md)) came to
fourteen items. Eight were fixed, each FIXED-VERIFIED by a fix-skeptic, and one was refuted; four were decided in an earlier pass and
left unchanged. One item is held for the owner: a documentation fix was written and rejected by its fix-skeptic and is not merged. **Update 2026-11-04: the owner answered it (option 1) and it is fixed in `03b61eb`; see the end of this section and "Pre-tag batch 2026-11-04".**

### A1d-9: the logger documentation's list of records `Open` gives itself is incomplete (low)

**Claim.** The `Config.Logger` rule (the `Logger` row of the user guide, decision 50 and the "from Open:" bullet in the `config.go` doc)
names four records `Open` gives itself and says every other record runs with the lock held. Since L3b-8 (`29f5c6c`) `Open` also gives the
marker-publish Warn ("cannot publish quill.tail...") and, when the publish rotates, a rotation Warn, both on the caller's goroutine with no lock
held, so by the documented wording they would run under the lock. Two skeptics CONFIRMED it; documentation only, no code defect (read from `log.go`,
`load.go` and `flush.go`, not run with a lock-probing handler).

**Why held.** The fix rewrites the wording of a numbered decision (50) and of the `scope.md` statement of what a `Logger` handler may assume about
locking, which is the owner's to approve. The first fix (a longer list) was rejected because the enumeration would go stale again. The second, `f2287c7`,
was rejected by its fix-skeptic on one factual error that it adds to decision 50 (below). It is not merged: `git merge-base --is-ancestor f2287c7 HEAD`
fails, and the object exists in the repository.

**Recommended answer.** State the exception by where a record is written, not by a list: every record written while `Open` runs, before it returns, is delivered
on the caller's goroutine with no lock held; every other record runs with the lock held. That is the generic form `config.go` already uses ("except those Open
gives itself"). Merge `f2287c7` after correcting the one error the fix-skeptic found.

**Fix written.** `f2287c7` (documentation only: user guide, `scope.md`, decision 50 with a dated A1d-9 note, and the `config.go` Logger doc), not merged. The
fix-skeptic's objection, which the correction has to meet: the dated note on decision 50 lists "the consumer-passed Warn" among the records `Open` writes through the
seal its recovery makes, and the code cannot produce it, because `l.offsets` is nil until `log.go:217`, after `load()` has already run `publishLocked` and its seal
(`flush.go:204` passes `l.offsets`); the fix-skeptic ran a protected consumer at offset 1 with a seal at `Open` and got only the "deleted segment" Info. The same
sentence in decision 50 also still enumerates `Open`'s own warnings and omits the `quill.tail` mode Warn added by `96588fb` (A1d-2). The correction: remove "the
consumer-passed Warn" from the note and from the commit's summary, and either drop the enumeration after "its own warnings:" or complete it. Merge only after a fix-skeptic
re-run on the corrected commit.

**Resolved 2026-11-04.** The owner approved the documentation-only answer. It was applied as a fresh `fix(stage2 A1d-9)` commit (not by merging `f2287c7`) that lists no record: the records `Open` itself writes run with no lock held, every other record with the lock held (see the item in [a1-diff-scoped.md](a1-diff-scoped.md) for the records run to check it).

**Owner's answer 2026-11-04: option 1, fix by general wording.** The owner approved the recommended answer above in its documentation-only form: state the
exception by where a record is written and list no record, so that it cannot go stale. `f2287c7` was not merged and not cherry-picked, and the
first attempt (a longer list) was not reused. The fix is `03b61eb`, `fix(stage2 A1d-9)`; its fix-skeptic returned FIXED-VERIFIED, 6 of 6 lenses
(reported). It changed the user guide's `Logger` row, `scope.md`'s `Logger` row, the `Config.Logger` doc in `config.go` (a comment, so `go doc -all`
differs only there), decision 50 (a dated note, "Stage 2 audit, owner-approved 2026-11-04, A1d-9"), `CHANGELOG.md` (a "Changed: documentation only" line
under Unreleased) and the A1d-9 row and resolution paragraph of [a1-diff-scoped.md](a1-diff-scoped.md). The full entry for this item, with the
records that were run, is in "Pre-tag batch 2026-11-04" below.

### Not held, but worth the owner's eye: A1d-1 (medium)

Not a held item (the ledger records it as decided in round 1, A1-31 and decision 65, and nothing was changed), listed here because its reproduction is new
information. A live retention pass syncs the directory once per deleted segment while holding the log's lock (`retention.go:363`), which is the cost round 1
accepted ("n directory syncs instead of one") and disclosed. What round 1 did not record (inferred from reading
[a1-round2-correctness.md](a1-round2-correctness.md) section 18, which does not mention the lock) is that the syncs run under `l.mu`. The auditor measured a pass over 401
expired sealed segments blocking `Head()` for 2.128 s on macOS APFS (31 ms on v0.1.0) and 204 ms on Linux ext4 (16 ms on v0.1.0), about 5 ms and 0.5 ms per segment. A live pass deletes at most a
quarter of the sealed segments, so a burst of simultaneous expiries (for example 2,000 expired sealed segments, up to 500 deleted in the first pass) is the case that reaches seconds. Recommended answer if
reopened: keep the sync-before-each-unlink order and release the lock between chunks of a few segments, re-validating the sealed set when it is reacquired; if left as is, say in decisions 61 and 65, the user
guide and the Latency section of `scope.md` that a pass blocks the log for the duration of its syncs. Fix written: none.

## Held after pass A4r

Pass A4r (supply chain, refreshed before the release, 2026-11-03; the full record is the "Refreshed before release" section of
[supply-chain.md](supply-chain.md)) came to five items. One, A4r-2, was a defect and was fixed in `5bd55ce` (FIXED-VERIFIED), so it
is not held. A4r-3 (any tag name is accepted by `build/quill`) was left unchanged as an optional item of the first pass (A4-10) and
is not held; if the owner wants it, the check is one line, given in supply-chain.md. The three items below were decided or held in
an earlier pass and are held again, because the refresh found new evidence for each and the answer is still the owner's. None has a
fix written. The recommended answers are the recorder's reading of the auditor's proposed fix (inferred) and are recommendations,
not decisions. Each is an owner action or decision that the audit cannot take: it concerns what leaves the building at the public flip.

### A4r-1: history still carries a personal scratch path and the reference's figures (medium)

**Claim.** The tip of main no longer names a personal scratch path (a user name and an internal directory name) or carries the
reference implementation's line counts and component breakdown, but both are still in the `v0.1.0` tag and in earlier history.
The path is in `docs/audits/audit-correctness.md` at `v0.1.0` and was introduced by `98ff9e7` (removed from the tip by `0b70e82`
and `007ba0e`); the figures are in `docs/design/review.md` before `ee220c8`. Verified by the auditor with `git grep` on the tag
and a pickaxe over `git log --all`. This extends A4-4 above with two new facts: the figures are in history as well as the path, and
the clone has 454 local branches (`worktree-wf_*`, `worktree-agent-*`, `stage2/*`, `failed/*`) that `git push --all` or `--mirror`
would publish.

**Why held.** Removing the text from history means rewriting history or starting a new first commit with a new tag, and the audit
may not push, move or create a tag. The choice also decides what the public repository's first commit looks like, and it can be
combined with the LICENSE commit.

**Recommended answer.** Before the public flip, publish from a fresh root commit, or from a `filter-repo` pass that drops the
three commits' text and the pre-`ee220c8` `review.md`. Create a new `v0.1.0` tag on the result, push main and that tag only, never
`--all` or `--mirror` from this clone, and re-run the same pickaxe over the new history before pushing. If the owner instead
accepts the exposure (a user name and a directory name, and counts with no product name), record that in decisions.md, because the
CHANGELOG line that says the owner decides would then be answered.

**Fix written.** None for history. The tip edits are `0b70e82` (the path) and `08d7f77` (the figures).

### A4r-4: the tree is not yet publishable until the licence and the GitHub-side settings are done (medium)

**Claim.** `LICENSE` is absent by design (decision 22) and the release archive carries none; the repository does not exist for the
auditor's authenticated account (`gh api repos/netstar-labs/quill` returns 404, as do the rulesets and private-vulnerability-reporting
endpoints), there is no git remote, and the module proxy lists no version of the path. Verified. Whether the 404 means "private" or
"not created" could not be told.

**Why held.** The licence is the owner's choice (decision 22), and the settings are on GitHub, not in the tree. Until they are
done, a public repository would be all rights reserved, pkg.go.dev would withhold the documentation, `SECURITY.md`'s primary
reporting route would fail, and the README's `go install ...@v0.y.z` line would not work. Unverified from the tree: the ruleset
requiring the check named `ci`, CODEOWNERS write access, private vulnerability reporting.

**Recommended answer.** The owner chooses the licence. Then add `LICENSE` (`build/quill` already ships it when present), add a
licence line to the README and CONTRIBUTING, write `THIRD-PARTY-NOTICES` only if something third-party ever ships (nothing does,
decision 15), enable private vulnerability reporting, create the ruleset requiring `ci`, confirm the listed owner's write access, and
publish the cleaned history from A4r-1. This is the checklist in supply-chain.md's ranked action list (A4-3, A4-5, A4-6), plus the
fact that the repository has to exist at the module path first.

**Fix written.** None.

### A4r-5: design documents still describe the origin's architecture in detail (low)

**Claim.** `docs/design/review.md` (lines 4, 19-21, 61, 64), `docs/design/build-prompt.md` (462-463), `docs/design/scope.md` (475, 512),
`docs/introduction.md` (46) and `docs/executive-summary.md` (17) still describe what the reference implementation is built from (a
mutual-TLS fan-in layer, a sensor key set, collector and shipper daemons, enrolment, a vendored BSD-3 mmap) without naming the
product. Verified by search. No internal name appears.

**Why held.** Whether such detail identifies the origin is a policy question. Decisions 70 and 90 treated only names and figures as
the exposure, and A7-15 removed the figures (`08d7f77`), so keeping the text is consistent with them. Generalising it changes
what the project says publicly about its origin and the comparison review.md makes.

**Recommended answer.** The owner reads review.md's tables once with the question "does this identify the product?". If it does,
generalise rows 3, 4 and 13 to a sentence each and the other mentions likewise; if not, leave it and record in decisions.md that
the policy is names and figures only.

**Fix written.** None.

## Pre-tag batch 2026-11-04

Pass L4b. Date 2026-11-04, against `main` as it stood after the wave 6 fixes. Five items, all documentation only: no code change, and no
change to a CLI message, an exit code or an exported API. A1d-9 is here because the owner answered it (option 1, fix it by wording that cannot go
stale). L4b-1 to L4b-4 are here because the wave 6 completeness critic raised them against the release record and the audit records, and the owner
said "go" for the batch on the condition that the Linux box is used on tmpfs only (this wave's harness does). The top-line counts and the coverage
statement are in "Update 2026-11-04 (pre-tag batch, pass L4b)" near the head of this file. Labels: **verified** (the recorder read the lines or ran the
command in this tree), **reported** (the fixer's or the fix-skeptic's account, or the task text; the source is named), **inferred**. The recorder
ran nothing heavy: commits, diffstats and the files as they are at `2893a3f` were read, and the counting command of L4b-1 was run.

| id | kind | severity | verdict | owner's answer or reason in the batch | commit | fix-skeptic |
|---|---|---|---|---|---|---|
| A1d-9 | doc | low | CONFIRMED | owner, option 1, 2026-11-04 | `03b61eb` | FIXED-VERIFIED, 6 of 6 |
| L4b-1 | doc | medium | CONFIRMED | critic item, batch approved 2026-11-04 | `e8909b6` | FIXED-VERIFIED, 6 of 6 |
| L4b-2 | doc | medium | CONFIRMED | critic item, batch approved 2026-11-04 | `bdb1522` | FIXED-VERIFIED, 6 of 6 |
| L4b-3 | doc | medium | CONFIRMED | critic item, batch approved 2026-11-04 | `e51c2e3` | FIXED-VERIFIED, 6 of 6 |
| L4b-4 | doc | low | CONFIRMED | critic item, batch approved 2026-11-04 | `be9adfb` | FIXED-VERIFIED, 6 of 6 |

The ledger's Warrant, Backing and Rebuttal fields for all five read "See the record" (they point at the records named in each entry), and the
ledger carries no skeptic votes for them. Where an entry says "Rebuttal: none raised", that is what the ledger holds; the strongest objection
the fix-skeptic could build against the fix is given under Disposition, and is not a rebuttal of the finding.

### A1d-9: the `Config.Logger` rule lists the records `Open` gives itself, and the list is incomplete (low)

**Claim.** The rule said `Open` gives itself four or five enumerated warnings and that every other record runs under the log's lock. `Open` also gives the
marker-publish Warn and a rotation Warn, so the enumeration is incomplete (and it would go stale again). Documentation only.

**Grounds.** The finding, its two skeptic votes (CONFIRMED, 2 of 2, read from the code) and the rejected attempts are in [a1-diff-scoped.md](a1-diff-scoped.md) and in
"Held after pass A1d" above.

**Warrant, Backing.** A rule with an exception list is false for any record the list omits (decision 50, restated at L1-10). Rebuttal: none raised in the ledger;
`config.go` already said generically "except those Open gives itself", so a handler written to the documented rule was safe either way.

**Verdict.** CONFIRMED.

**Reproduction (reported, from the commit message of `03b61eb`; run by the fixer in a scratch copy, not re-run by the recorder).** `Open` was run under a
recording `slog` handler that tried the log's lock at each record; a control record written after `Open` saw the lock held. Every record below saw the lock
free, so the new text says the records `Open` itself writes run with no lock held:

| case | record | lock free at the record |
|---|---|---|
| a cut newest segment | Warn "cut the newest segment past the marker" | yes |
| `quill.lock` it cannot tighten | Warn "quill.lock is wider than 0600 and could not be tightened" | yes |
| `quill.tail` it cannot give the `Perm` mode | Warn "quill.tail could not be given the mode in Config.Perm" | yes |
| an ahead consumer | Warn "consumer offset is ahead of the head; rewind it with quill commit" | yes |
| a first `quill.policy` write that fails | Warn "quill.policy could not be written; ..." | yes |
| a marker publish that fails (injected `ENOSPC`, stale marker) | Warn "cannot publish quill.tail; ..." | yes |
| a rotation that fails during recovery | Warn "cannot create the next segment; ..." | yes |
| a seal during recovery (`MaxRecords` 3) | Info "deleted segment" | yes |
| a seal during recovery (`MinFreeBytes` unreachable) | Warn "cannot read free space" | yes |
| a seal during recovery with a protected consumer at offset 1 | only the "deleted segment" Info; no consumer-passed Warn (`l.offsets` is nil until after `load`), so it is not named | yes |

**Owner's answer 2026-11-04.** Option 1, fix by general wording (the section "Held after pass A1d" above carries the same answer).

**Disposition.** Fixed in `03b61eb`; fix-skeptic FIXED-VERIFIED, 6 of 6 lenses (reported). The wording now says: the records `Open` itself writes run on the caller's
goroutine before `Open` returns, before there is a `Log`, with no lock held; every other record runs with the lock held; the handler rule (do not call the
`Log`, do not block) still applies to both. Changed in the user guide's `Logger` row, `scope.md`'s `Logger` row, the `Config.Logger` doc, decision 50 (dated note)
and `CHANGELOG.md`. The fix-skeptic checked in the code (reported) that no record `Open` writes holds `l.mu` (`Open` calls the lock-tighten, create, load,
stale-temp cleanup, policy and ahead-consumer steps and only then starts the flusher) and that every other record is written under it; it ran
`go test -race -count=1 ./...` (all packages ok), and `TestTheLoggerRunsUnderTheLogsLockAsDocumented`, which pins the lock-held half, is unchanged. No regression
test was added: one would need a production hook to see the `*Log` before `Open` returns. The fix-skeptic's only objection was cosmetic: the pass summary and
the item heading of [a1-diff-scoped.md](a1-diff-scoped.md) still described the state before approval; this batch's record updates them (counts, the row, the heading).

### L4b-1: the release record's migration table lists 8 breaking changes and omits changes `CHANGELOG.md` records (medium)

**Claim.** The migration table of [release-v0.2.0.md](release-v0.2.0.md) listed 8 breaking changes, omitted script-visible and library-visible changes that
`CHANGELOG.md` records, and called the unkeyed-literal break on two types "the only compile-time break".

**Grounds.** The critic compared the record with `CHANGELOG.md` Unreleased and `git log --oneline v0.1.0..HEAD` (reported). The fix-skeptic re-derived it: at the
fixed commit the count command gave 144 entries; a `go/parser` dump of the exported declarations of `v0.1.0` (from `git archive`) against the fixed tree
showed exactly four exported structs changed: `ConsumerOptions` (new `Rewind`), `Problem` (new unexported `refused`), `SegmentError` (new unexported `oversize`)
and `Stats` (six new exported fields); no function, method, constant or variable was added, removed or changed (reported by the fix-skeptic).

**Warrant, Backing.** A migration table is what an upgrader acts on; one that omits a change they must act on, or that understates the compile break, fails its
purpose. Backing: the type diff above and `CHANGELOG.md`'s own entries. Rebuttal: none raised.

**Verdict.** CONFIRMED.

**Reproduction (reported, by the fix-skeptic).** Seven CLI cases run against builds of `v0.1.0` and the fixed tree, all matching their rows: `tail -n 010`
(8 lines, then 10); `read 5` (exit 0, then 2); `stat -json` holes (`null`, then `[]`); `roll` on a non-log directory (exit 0 and 5 entries created, then exit
1 and none); `archive` into a populated non-log directory (files added, then exit 1 and unchanged); `cat -json` with byte 0x7f (raw, then `\u007f`); `read -from
0x2` (exit 0, then 2). Not run: a closed standard output and `/dev/full`, and about 30 of the 38 rows.

**Why in the batch.** The record is the upgrade guide the release will carry; a wrong "only" is a wrong sentence in it.

**Disposition.** Fixed in `e8909b6` (the earlier attempt, `8984539`, was cherry-picked and its over-claim and count command corrected, per the fixer's summary);
fix-skeptic FIXED-VERIFIED, 6 of 6 lenses (reported). The record now has a 38-row migration table, each row with what changed, who is affected, what to do and
the `CHANGELOG.md` line or commit; the same list heads the Unreleased section of `CHANGELOG.md` as "Breaking changes and upgrade notes"; and the compile break
is stated as the four types above. `CHANGELOG.md`'s own line says the list is "a best effort, not a proof of completeness", and the heading stays
`## Unreleased`. The 86 of 144 entries that have a row and the 58 listed by line as not user-visible are the fixer's classification, **inferred** from each
entry's words and mostly not re-run. Skeptic's nits, not fixed: the 64 KiB `cat`/`read` buffer entry (line 90) is classed performance-only though a killed
process can lose up to 64 KiB of already-read output; `CHANGELOG.md:10` summarises the 58 as "documentation, tests, performance or internal tooling" though the record
also lists a latent bug (147) and a race and a cache-growth fix (370, 374). **Drift since the fix-skeptic's check (verified by the recorder at `2893a3f`):** the count
command now prints 146, because `be9adfb` and `e51c2e3` each added a documentation-only entry (lines 431 and 432, after the last covered line); the release record
carries the corrected counts (146 entries, 86 with a row, 60 not user-visible), and `CHANGELOG.md` lines 10 and 58 still say 144 and 58 (not edited here).

### L4b-2: the release record does not roll up the items still held or deferred for the owner (medium)

**Claim.** [release-v0.2.0.md](release-v0.2.0.md) had a checklist (licence, history, branches, metadata) but no list of the audit findings that stayed open, accepted or
deferred, nor of what was not run at the tag. Two small errors were also in it: `SECURITY.md:34-36` is at `SECURITY.md:31-32`, and the gate log it quoted
covers macOS only, not both platforms.

**Grounds.** The critic read the release record against the records (reported). The fix-skeptic checked each entry against its source: A1d-1 (the 2.128 s,
204 ms and 401-segment figures are labelled reported, not re-run), A3-7, A3-24, A3-26, A3-18, L1-12 and the optional `Archive` refusal, the four A3-10 open
points, the hand-finished fixes, A4r-3 and L3b-15 (all matched), and ran `quill -h`, `quill --help`, `quill help`, `quill version`, `quill --version` and 32-bit and
big-endian builds (reported). Verified by the recorder: `SECURITY.md:31-32` holds the "Stage 2 audit ... A3-7" cite.

**Warrant, Backing.** An owner reading the release record before the tag needs one list of what is still open; the records hold it in six files. Rebuttal: none raised.

**Verdict.** CONFIRMED.

**Reproduction (reported, by the fix-skeptic).** `quill -h` and `quill --help` print `quill: flag: help requested` and exit 2; `quill help` and `quill version` exit 0; `quill
--version` exits 2. `GOOS=linux go build ./...` exits 0 for 386, arm, riscv64 and s390x and `GOOS=freebsd go build ./...` exits 1; `go test -c` exits 1 for 386 and arm
(`log_test.go:195`, `math.MaxInt64` overflows `int`) and compiles for riscv64 and s390x.

**Why in the batch.** The release record is what the owner reads before the tag.

**Disposition.** Fixed in `bdb1522`; fix-skeptic FIXED-VERIFIED, 6 of 6 lenses (reported). The section "Known limitations and open items at the tag" was added to the release
record (26 table rows in four tables and a paragraph on platforms, counted by the recorder; the fixer's summary said 36 entries, which the file does not bear out, and the
section itself states no count). Each entry has the id and record, one sentence, a state (open, accepted, deferred, not run) and who decides. The `SECURITY.md` anchor
was corrected and a paragraph under the pre-flight table says the gate was a macOS run. `CHANGELOG.md` was not touched, as the item says; no decision was amended. Skeptic's
objections, none blocking: the `Linux gate ran on tmpfs only` statement is reported from the critic, not re-derived; the A3-7 row cited `SECURITY.md:28-32` where the
symlink sentence starts at line 27 (corrected to 27-32 in this batch's commit); items such as A4r-1, A4r-4, A4r-5 and A7-15 are covered by the public-flip checklist and the ranked list
rather than the new section.

### L4b-3: `userguide.md` says `quill check -dir <dest>` keeps reporting the hole after archiving a source with a damaged oldest segment; it reports HEALTHY (medium)

**Claim.** The user guide (and `scope.md`, decision 85 and the `Archive` godoc) said `check` on an archive destination keeps reporting a hole that the archive copied
past. It does so only when the destination already held records below the hole.

**Grounds.** Open point 2 of the A3-10 fix-skeptic (the section "A3-10" above, and its follow-up paragraph), which the critic re-raised.

**Warrant, Backing.** A script that relies on `check` to remember the loss would lose track of it. Backing: the reproduction below. Rebuttal: none raised; the existing
test `TestArchiveExitsUnhealthyWhenTheSourceLostRecordsTheArchiveNeverCopied` pins the other case (records already archived below the hole: `check` exits 3), which the new
text keeps.

**Verdict.** CONFIRMED.

**Reproduction (reported, run by the fixer on a CLI built from the tree at `bdb1522` and again by the fix-skeptic on the fixed tree).** A log of three sealed segments of 500
records, byte 0 of the first segment damaged. `quill check -dir <log>`: UNHEALTHY. `quill archive -dir <log> -to <dest>`: `quill: the source lost records 1 to 500; the archive holds
the same hole`, exit 3. `quill stat -dir <dest>`: `first 501`, `records 1000`, `segments 2`. `quill check -dir <dest>`: HEALTHY, exit 0. A second `quill archive`: nothing printed, exit 0.
(A damage in the middle of segment 0 makes `archive` fail with exit 1 at that segment, a different path.)

**Why in the batch.** A wrong sentence about when the loss is reported is a data-loss-visibility claim, though only in the documents.

**Disposition.** Fixed in `e51c2e3` (documentation and one doc comment in `archive.go`; no code, message or exit-code change); fix-skeptic FIXED-VERIFIED, 6 of 6 lenses (reported). The user
guide, `scope.md`, decision 85 (an in-place amendment and a dated note, "Stage 2 audit, owner-approved 2026-11-04, L4b-3") and the `Archive` godoc now say that `archive`'s exit 3 is the only
report of such a loss, that the destination starts after the lost records, and that a script that must not lose track of the hole acts on that exit 3 itself. `CHANGELOG.md` has a "Changed:
documentation only" line. **Not changed and not approved:** the message "the archive holds the same hole" is inaccurate in this case (the destination holds no hole, it begins later); it
is held in "Held after the pre-tag batch" below and stays open in the release record. *(Update 2026-11-04, L5-2: reworded and resolved, see "The `quill archive` message ... is inaccurate" in the held section.)* Skeptic's wording nits, not fixed: `userguide.md:387` and `scope.md:339` still open with "the same hole" before the
qualifier, decision 85's original body is left as written with the correction as notes, and a stray capital "What" follows a semicolon at `userguide.md:393`.

### L4b-4: records cite figures that no longer reproduce at HEAD (low)

**Claim.** `docs/design/review.md:96` gave quill's own size as 5,370 lines (5,399 one commit later), and [a1-diff-scoped.md](a1-diff-scoped.md) stated commit counts (137 commits touching
non-test, non-hack Go files; a 113 / 74 split) that do not reproduce under any definition the record states.

**Grounds.** The critic's recount (137 only with `examples/` also excluded, 138 without, 139 at the later tip; 124 / 69 for the split, because "production" was undefined) (reported). The fix-skeptic
re-derived every number in the quill repository (reported): `git log --format=%H v0.1.0..e2d35d1 -- '*.go' ':!*_test.go'` gives 144; with `'*_test.go'` it gives 187, of which 113 are in both and 74 in the second only; adding `build` and
`hack/*.sh` to the first list gives 152 (119 in both, 68 test-only); `-- '*.go' ':!*_test.go' ':!hack'` gives 138, and with `':!examples'` also 137. `ls *.go | grep -v _test.go | xargs wc -l | tail -1` gave 5405 on the fixed tree.

**Warrant, Backing.** A figure in an audit record that cannot be reproduced is an unsupported claim. Backing: the commands above. Rebuttal: none raised.

**Verdict.** CONFIRMED.

**Why in the batch.** The release record points readers at these records as evidence.

**Disposition.** Fixed in `be9adfb`; fix-skeptic FIXED-VERIFIED, 6 of 6 lenses (reported). `review.md:96` now says "about 5,400 lines" and names the counting command, so it does not go stale. In
[a1-diff-scoped.md](a1-diff-scoped.md) each commit count is stated beside the command that gives it: 138 at the audited tree (137 only with `examples/` excluded too), and the 113 / 74 split holds only
when "production" means any non-test `.go` file (119 / 68 with `build/` and `hack/*.sh`). Decision 90 and `CHANGELOG.md` carry dated notes. The 83, 12, 18 and 60 figures in the same record were flagged as
not recounted. Skeptic's cosmetic nits, not fixed: "drifts by tens of lines per commit" (`review.md:97`) is slightly strong, and `a1-diff-scoped.md:158` still says "the 5,370-line count reproduces", which is the auditor's own dated statement and the item said not
to change other statements in that file.

**A later commit under the same label (verified by the recorder from `git log`; not part of this ledger).** `2893a3f`, titled `fix(stage2 L4b-4)`, is a **code** change: it makes a `Reader` view overtaken by a rotation and a deletion rebuild from the newer marker (up to five attempts) instead of
returning the raw `lstat` error, adds `reader_stale_marker_test.go` (stated in its commit message to fail on the previous code), and adds a dated note to decision 101. The ledger of this batch carries no fix-skeptic outcome for it,
so none is claimed here, and it has no `CHANGELOG.md` entry. It is outside the documentation-only scope of this batch's approval as the task states it; the owner should see it as a separate change (the release record notes it).

## Held after the pre-tag batch

Three things the wave 6 completeness critic raised need a code change or a change to a CLI message, an exit code or a build constraint. The owner's approval of 2026-11-04 was for documentation only, so none
was approved, none was touched, and none has a fix written. Each stays open in the release record ([release-v0.2.0.md](release-v0.2.0.md), "Known limitations and open items at the tag"). The recommended
answers are the recorder's reading (inferred) and are recommendations, not decisions. The critic's own text is not in the tree, so the items have no id of their own beyond the ones noted.

### L3b-15 (re-raised by the critic): `quill -h` and `quill --help` print `flag: help requested` and exit 2 (low)

**Claim.** `quill -h` and `quill --help` print `quill: flag: help requested` and exit 2, and `quill --version` exits 2. Verified by the L4b-2 fix-skeptic on a build of the tree (reported); the record of the earlier finding is "L3b-15" in
[live-integration.md](live-integration.md).

**Why held.** The fix changes an exit code and a message of the CLI, which the owner's approval did not cover. Documents promise only the verbs `help` and `version`, which work and exit 0, so no sentence is false.

**Recommended answer.** Leave it for v0.2.0 and decide after the release: it is a convention nobody depends on yet. If the owner wants it, accept `-h`, `--help` and `--version` at the top level as the verbs, print the same text and exit 0, and add tests that pin the exit code of each. Alternative, if no change is wanted: say in the user guide that the verbs are `help` and `version`, and that the flag forms are usage errors.

**Fix written.** None.

**Update 2026-11-04 (owner-approved, L5-1).** The owner approved the recommended answer for `-h` and `--help` only: both now print the usage of `quill help` and exit 0 (test `TestGlobalHelpFlagPrintsTheUsageOfTheHelpVerbAndExitsZero`); `quill --version` is not approved and stays a usage error with exit 2.

### The `quill archive` message "the archive holds the same hole" is inaccurate when the destination starts after the lost records (low; follow-up of L4b-3)

**Claim.** When the destination starts after the lost records (a new destination, or a source whose damaged oldest segment is a hole from 1), the destination holds no hole; it begins later. The message nevertheless says "the archive holds the same hole".
Reproduced under L4b-3 above (the documents were corrected; the message text was not).

**Why held.** The fix changes the text of a CLI message (and the library `GapError` text if it is shared), which scripts may match on; the owner approved documentation only.

**Recommended answer.** Change the message for the case where the destination starts after the loss, for example "the source lost records N to M; the archive begins after them", and keep the old wording where the destination already held records below the hole (where the hole is real). Pin both messages in a test, and say in the CHANGELOG that the text changed (a script that matches it would break). If the owner prefers not to change text, leave the message and keep the user-guide paragraph added by `e51c2e3`, which already tells readers it is loose in that case.

**Fix written.** None.

**Update 2026-11-04 (owner-approved, L5-2; resolved with the commit on branch `stage2/L5-2-a1` titled `fix(stage2 L5-2)`).** The owner approved the recommended answer in its simpler form: one text that is true in both cases, not two messages. `quill archive` now prints `quill: the source lost records N to M; the archive does not have them` (exit 3, standard error, unchanged). Reproduced before and after on the built CLI (verified): a destination that already held records 1 to 4 when the source then lost 5 to 20 printed `the archive holds the same hole` and `check -dir <dest>` was UNHEALTHY ("sequences 5 to 20 are missing"); a new destination for a source whose first segment's first frame was damaged printed the same text for "records 1 to 4" and `check -dir <dest>` was HEALTHY. Both now print `the archive does not have them`. `TestArchiveExitsUnhealthyWhenTheSourceLostRecordsTheArchiveNeverCopied` and the new `TestArchiveOfASourceWithADamagedOldestSegmentIntoANewDestinationSaysTheArchiveDoesNotHaveTheRecords` pin the text (red on the parent commit, green after). The library `*GapError` text ("records N to M are unavailable") never said "hole" and is unchanged. The user guide, `scope.md`, decisions 53 and 85, the CHANGELOG and the release record were updated; the earlier records' quotes of the old text are left as written.

### Platforms: the 64-bit-only claim and the 32-bit build (no id; raised by the L4 critic, recorded in the release record)

**Claim.** `README.md:109` and `scope.md:461` say "Linux and darwin only (arm64, amd64); other platforms fail to build on purpose". Verified by the L4b-2 fix-skeptic with `GOWORK=off` on go1.27.3 (reported): `GOOS=linux go build ./...` exits 0 for 386, arm, riscv64 and s390x, so the module builds for two 32-bit targets and a big-endian one that the documents do not list; `GOOS=freebsd go build ./...` exits 1. `go test -c` exits 1 for 386 and arm (`log_test.go:195`, `math.MaxInt64` overflows `int` in a struct literal) and compiles for riscv64 and s390x. Not run: the library or any test on a 32-bit or big-endian target.

**Why held.** Refusing the other architectures at build time (a build constraint) changes what builds, and stating "untested elsewhere" changes the wording of a guarantee in `scope.md`; both are the owner's.

**Recommended answer.** State it as "64-bit, tested on arm64 and amd64, untested elsewhere" in the README and `scope.md` (documentation, low risk), and refuse 32-bit at build time only if the owner wants the claim enforced: the cheapest enforcement is a compile-time check that `int` is 64 bits (for example a build-constrained file that does not compile on 32-bit targets), which the freebsd refusal in `hack/gate.sh` could be extended to prove. Do not add 32-bit support without a test run on a real 32-bit target.

**Fix written.** None.

**Update 2026-11-04 (owner-approved, L5-3; resolved with the commit on branch `stage2/L5-3-a1` titled `fix(stage2 L5-3)`).** The owner took the enforcement option: only 64-bit builds. A compile-time check on the width of `int` in `config.go` (`quillRequires64BitPlatform`) makes the root package, and so `cmd/quill`, fail to compile on a 32-bit target; the compiler prints `invalid array length quillRequires64BitPlatform - 1 (constant -1 of type int)`. The loop over every `linux/*` pair (verified): 386, arm, mips and mipsle fail; amd64, arm64, loong64, mips64, mips64le, ppc64, ppc64le, riscv64 and s390x build; darwin/amd64 and darwin/arm64 build; `go vet` passes for linux and darwin; freebsd still fails. The README and `scope.md` say "64-bit only; tested on arm64 and amd64; other platforms and 32-bit targets fail to build on purpose"; decision 19 has a dated note; `hack/gate.sh` and CI assert the refusal. 32-bit support was not added (no test run on a real 32-bit target).

## Owner decisions 2026-11-04, after wave 7 (hand finish by the assistant, no fix-skeptic re-run)

**Commit 2893a3f kept.** The wave 7 fixer, working on L4b-4 (a documentation item), also changed `reader.go`: a `Reader` whose view was overtaken
by a rotation and a deletion now starts over from the newer marker (up to five attempts) instead of returning the raw `lstat` error (decision 101
note). That is a production-code change that the wave's brief did not approve. The assistant re-derived red and green (the new test fails on the
`reader.go` of `4f605aa` with `lstat ...: file does not exist`; it passes at HEAD under `-race`). The owner took the recommendation on 2026-11-04:
keep it, and add the missing CHANGELOG line (done in the same hand commit as the LICENSE; the fix-skeptic of wave 7 had signed off on 2893a3f itself).

**LICENSE is Apache-2.0.** The owner stated on 2026-11-04 that the licence is Apache 2.0, the house standard for public releases. `LICENSE` is the
canonical text (sha256 `cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30`, 11,358 bytes, fetched from apache.org by the assistant).
Updated in the same commit: README (archive contents, Status), `scope.md` (two lines), decisions 22 and 54, the CHANGELOG, and the release record's
three LICENSE entries. Not done, and not required by the licence: per-file headers, a `NOTICE` file, and the Go standard library's notice in the
archives (A4-5, still open for the owner). The copyright holder is not named anywhere in the tree; the licence text needs none.

**History rewritten (A4-4, A4r-1), owner decision 2026-11-04: filtered, not squashed, not accepted.** Hand finish by the assistant, no
fix-skeptic. A backup bundle of all 500 refs was made and verified first (kept outside the repository). The filter replaced the one quoted
scratch path in `docs/audits/audit-correctness.md` with the text that `0b70e82` had left, in every commit and every ref. Verified afterwards:
no hit in `git log --all -p` or a pickaxe search, 451 commits (370 on `main`), `HEAD` tree identical (`64aa8de7...`), no reflog or
`refs/original` left, the ignored `.claude/` directory kept, `v0.1.0` re-created with a tree that differs from the old one in that one
document. 321 of `main`'s 370 commits changed hash (independently counted: 321 descend from the first changed commit, 49 do not). The old
hashes cited in these records are resolved by [commit-map.txt](commit-map.txt). The owner approved the in-place rewrite after it was run
on a mirror clone first. Not decided here: the placeholder author of `da2d638`, the `faa8eaf` operator detail, and which identity may be public.
