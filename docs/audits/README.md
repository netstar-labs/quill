# Audit records

This directory holds the records of the audit sequence that quill went through before its v0.2.0 release: what each pass looked at,
every candidate finding with the arguments for and against it, what was fixed, what was held back for the owner and decided, and what
was not run. [summary.md](summary.md) is one page: what came out of it and what is still open. [release-v0.2.0.md](release-v0.2.0.md) is the
live record of the release itself.

The records are point-in-time documents. Each says which tree it examined and is not updated when the code moves on, except where a
later pass added a dated note to a finding it resolved. The files under [reference/](reference/) are the ones the code and the other
documents cite, so they are the ones to keep current; two of them, the threat model and the format audit, are also dated pass records.

## Layout

```
README.md, summary.md        this file; the one-page outcome
release-v0.2.0.md            the release record: rehearsal, public-flip checklist, known limitations, what was not run
release-v0.1.0.md            what the v0.1.0 tag is, which passes examined it, what is not recorded for it
commit-map.txt               old to new commit hashes (see "Hashes" below)
reference/                   what the code and the other documents cite: threat model, format and version skew,
                             benchmark baselines, power-loss runs, what the tests cannot show, test ledger, harness inventory
passes/                      one record per audit pass, in the order below
decisions/                   the ledger of items held for the owner, with an index
```

## The passes, roughly in the order they ran

| Pass | What it examined | Date | Record |
|---|---|---|---|
| Build self-review | The recovery path and the reader, read hostilely by the build itself before any audit | 2026-10-28 | [passes/self-review-recovery-and-reader.md](passes/self-review-recovery-and-reader.md) |
| A1, round 1 | Adversarial audit, in four lenses (simplify, dedup, correctness, docs), of the tree on the way to v0.1.0 (`b52b746`; its fixes are in the tag, see [release-v0.1.0.md](release-v0.1.0.md)). `audit-findings.md` is the synthesis | 2026-10-29 | [passes/a1-round1/](passes/a1-round1/) |
| A1, round 2 | The same four dimensions again with fresh auditors; includes pass A1h, the held items the owner approved on 2026-10-30. `a1-round2-findings.md` is the synthesis | 2026-10-29 to 30 | [passes/a1-round2/](passes/a1-round2/) |
| A4 (refreshed as A4r) | Dependencies, supply chain and release hygiene of the tree and its history | 2026-10-29; A4r 2026-11-03 | [passes/supply-chain.md](passes/supply-chain.md) |
| A3 | Security review and threat model | 2026-10-31 | [reference/threat-model.md](reference/threat-model.md) |
| A5 | Silent degradation: places where the log can get worse without saying so | 2026-10-31 | [passes/silent-degradation.md](passes/silent-degradation.md) |
| A6 | On-disk format and version skew | 2026-10-31 | [reference/format-compatibility.md](reference/format-compatibility.md) |
| L1 | Verification and optimisation | 2026-11-02 | [passes/verification-optimization.md](passes/verification-optimization.md) |
| L2 | The test, benchmark and hardening suite: mutation testing and its defects log | 2026-11-03 | [passes/l2/](passes/l2/) |
| L3, L3b | Live multi-process validation, the Linux filesystem matrix and power-cut runs | 2026-11-03 | [passes/live-integration.md](passes/live-integration.md) |
| A7 | Gap audit: what the other passes could not see | 2026-11-03 | [passes/gap-audit.md](passes/gap-audit.md) |
| A1d | Adversarial audit of the v0.1.0..HEAD diff | 2026-11-03 | [passes/a1-diff-scoped.md](passes/a1-diff-scoped.md) |
| L4 | Rehearsal of the v0.2.0 release | 2026-11-04 | [release-v0.2.0.md](release-v0.2.0.md) |
| L4b, L5 | Pre-tag batches after the rehearsal: documentation items, then code fixes | 2026-11-04 | [decisions/](decisions/) and the release record |

Standing references: [threat model](reference/threat-model.md), [format and version skew](reference/format-compatibility.md),
[benchmark baselines](reference/benchmark-baselines.md), [power loss, for real](reference/power-loss.md),
[what the tests cannot show](reference/untestable-without-x.md), [test ledger](reference/test-ledger.md),
[harness inventory](reference/harness-inventory.md).

## Finding ids

Every candidate has an id made of the pass and a number: `A3-26`, `L3b-8`, `A1d-9`, `L5-3`. The id appears in the heading of its
section, and other records and the CHANGELOG cite it as text, not as a link. To find everything about one finding:

```
grep -rn 'A3-26' docs/audits docs/design CHANGELOG.md
```

The ledger in [decisions/](decisions/) has every item held for the owner, with the owner's answer where one was given and the fix
commit and fix-skeptic outcome where a fix was written; some items are recorded as not approved, with no fix. The numbered decisions the records cite ("decision
51", "decision 101") are a different list: [docs/design/decisions.md](../design/decisions.md).

## How these audits were run

The auditors, skeptics and fixers were AI agents, run in waves through a workflow harness; a person, the owner of the repository,
made every decision the records call "held for the owner". The method, in order:

1. **Audit.** Several independent auditors each read the code with a lens (correctness, simplification, security, silent failure,
   format, docs, live behaviour) and wrote candidates as arguments a skeptic could attack: claim, grounds, warrant.
2. **Attack.** Each candidate went to one to four skeptics ("votes"), each told to refute it against the real code or to reproduce it
   in a scratch copy, and to default to refuted when unsure. A claim needing a reproduction is CONFIRMED only with one, and a split vote
   goes to an arbiter. Verdicts (CONFIRMED, PLAUSIBLE, REFUTED) are kept with their reasons.
3. **Decide.** Survivors that change behaviour, a guarantee, an exit code or an output shape are held for the owner, with a recommended
   answer. The rest are fixed.
4. **Fix.** One fixer per item, in an isolated worktree, test first where a test can be red. A fix-skeptic then attacks the fix. Fixes
   land one at a time.
5. **Gate.** `hack/gate.sh` (vet, race tests, staticcheck, deadcode, fuzz, tidy) is the gate each change was meant to pass on macOS and
   on Linux. Where it ran in full and where only parts of it ran is stated in the records; the pre-tag batches (L4b, L5) ran partial
   checks, and those batches carry no skeptic votes because the owner approved their items directly (see the release record).

The prompt that drove the audits is [docs/design/audit-prompt.md](../design/audit-prompt.md). Statements in the records carry one of
three labels: **verified** (run, or the exact lines read), **reported** (a named source says it) and **inferred** (constructed, with
what it rests on).

Where the records say a fix was "finished by hand" or a "hand finish", a person or the assistant completed or applied it with no
fix-skeptic re-run on the final state. Most were finished after a fix-skeptic rejected the fix on one stated objection; the release
record lists those. The history filter, the `LICENSE` commit and the kept `Reader` change had no fix-skeptic at all; the ledger section
"Owner decisions 2026-11-04, after wave 7" records them.

## Glossary

- **Candidate.** A claimed defect or wrong sentence, before the skeptics have seen it.
- **Skeptic, vote.** An independent attacker of one candidate; a vote is its verdict.
- **Held for the owner.** A change the audit would not make on its own because it alters behaviour or a promise.
- **Fix-skeptic.** The skeptic that attacks a fix, not a finding.
- **Hand finish.** See above.
- **Gate.** `hack/gate.sh`, the checks a change must pass.
- **Deferred, accepted, open.** The states in the release record: open stays unless someone acts, accepted was decided and left,
  deferred is after the release.

## Hashes

Before anything was pushed, the history was rewritten once to remove a scratch path that one audit document had quoted. A commit hash
cited in a record written before the rewrite is the hash from before it: look it up in [commit-map.txt](commit-map.txt), which maps old to
new for the 321 commits on `main` whose hash changed (the other 49 are unchanged). Commits made after the rewrite are cited by their
current hash and are not in the map; a few records of 2026-11-04, the release record among them, cite both kinds.
