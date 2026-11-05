# Release record: v0.1.0

Written 2026-11-05, after v0.2.0, by the assistant that ran the release; no skeptic read it. v0.1.0 was tagged on 2026-10-29, before most of the audit sequence existed, and had no release rehearsal, so this is a
short account of what the tag is and what the records say about it, not a checklist. Each statement says how it is known: **verified** (a command run for this record), **reported** (a named record says it).

## The tag

- **Verified (`git cat-file -p v0.1.0`, `git log -1 v0.1.0`).** `v0.1.0` is an annotated tag by `zxdev`, dated 2026-10-29, on `9364fb0` ("docs: benchmark baselines re-recorded on both machines after the allocation pass").
  Its message: "The first release: the on-disk format quill/1, the Log, the Reader, consumers, retention, Archive and the quill command, built in four phases from docs/design, then the A1 adversarial audit
  (46 candidates, 39 fixes) and the allocation pass (22 candidates, 12 fixes). Every gate green on macOS and Linux, T14 run on a real full filesystem, three real power cuts survived, baselines recorded on both
  machines." The message's counts are the tagger's, not re-derived here.
- **Verified.** The tag was re-created on `9364fb0` in the history rewrite of 2026-11-04; before it the tag was on `419e15a` ([commit-map.txt](commit-map.txt)). The two trees differ in one audit document only.
- **Verified (`git ls-remote origin`, `gh release list`).** The tag was pushed to `netstar-labs/quill` on 2026-11-05, after `v0.2.0`, so that `hack/skew.sh v0.1.0 HEAD` has its older ref. No GitHub release was made for it:
  the repository's one release is `v0.2.0`. No archives, `SHA256SUMS` or SBOM exist for `v0.1.0`.
- **Verified (`git ls-tree -r v0.1.0`).** The `v0.1.0` tree has **no `LICENSE` file** (and no file with "license" in its name). The Apache-2.0 `LICENSE` was added after it, on the way to v0.2.0 (the ledger section
  "Owner decisions 2026-11-04, after wave 7" in [stage2-owner-decisions.md](decisions/stage2-owner-decisions.md) records the `LICENSE` commit). Anyone who fetches the `v0.1.0` source gets code with no licence grant;
  `v0.2.0` and later have one.
- **Verified.** `go.mod` at `v0.1.0` has the module path `github.com/netstar-labs/quill` and `go 1.27`. The 32-bit refusal, `-h`, `--help` and `--version`, and the other pre-tag changes of v0.2.0 are not in it.

## What examined it

The audit passes did not all look at the same tree. In order (reported by the records named; the [README](README.md) has the dates):

| Pass | Tree it examined | Relation to `v0.1.0` |
|---|---|---|
| A1, round 1 ([passes/a1-round1/](passes/a1-round1/)) | `b52b746` (the record says "against HEAD b52b746") | An ancestor of the tag (verified: `git merge-base --is-ancestor b52b746 v0.1.0`). Its fixes are in the tag. The tag message counts 46 candidates and 39 fixes; the synthesis counts 45 raised and 44 surviving, so the two do not match and this record does not reconcile them. |
| The allocation pass | the tree before `a49064d` | In the tag (verified: `a49064d` is in `git log v0.1.0`). Recorded in [audit-followups.md](passes/a1-round1/audit-followups.md) and [benchmark-baselines.md](reference/benchmark-baselines.md) |
| A1, round 2 ([passes/a1-round2/](passes/a1-round2/)) | `1da37fc` ("v0.1.0 plus one docs-only commit", per the record; `b6c25ec` after the rewrite) | Effectively the tag |
| A4, supply chain ([supply-chain.md](passes/supply-chain.md)) | `v0.1.0` (then `419e15a`) and `main` as it stood | The tag. A clean checkout built, `quill version` printed v0.1.0, no dependencies (reported by the record) |
| A1d, the diff-scoped audit ([a1-diff-scoped.md](passes/a1-diff-scoped.md)) and every later pass | `v0.1.0..HEAD` or later trees | After the tag |

So the "audit of v0.1.0" wording of the [README](README.md) fits A1 round 2 and A4; A1 round 1 examined the tree on the way to it.

## What changed since

- **Verified (`git rev-list --count v0.1.0..v0.2.0`).** 304 commits between the tags.
- **Verified (`git diff --shortstat v0.1.0 v0.2.0 -- . ':!docs'`).** 189 files changed outside `docs/`, 22,417 insertions, 1,097 deletions.
- The user-visible changes, and what an upgrader must do, are under "Breaking changes and upgrade notes" at the top of the `v0.2.0` section of the CHANGELOG and in the migration table of
  [release-v0.2.0.md](release-v0.2.0.md). Format compatibility between the two is [format-compatibility.md](reference/format-compatibility.md); `hack/skew.sh v0.1.0 HEAD` is the cross-version check.

## What was not done for v0.1.0

No release rehearsal of `v0.1.0` is recorded here (the rehearsal in [release-v0.2.0.md](release-v0.2.0.md) is of v0.2.0), and no archives were built or checksummed for it (verified above). The live
validation of [passes/live-integration.md](passes/live-integration.md) ran from 2026-11-03, on later trees; GitHub CI first ran on 2026-11-05, on the `v0.2.0` push.
