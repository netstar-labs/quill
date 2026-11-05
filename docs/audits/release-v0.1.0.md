# Release record: v0.1.0

Written 2026-11-05, after v0.2.0, by the assistant that ran the release; no skeptic read it. v0.1.0 was tagged on 2026-10-29, before most of the audit sequence existed, and had no release rehearsal, so this is a
short account of what the tag is and what the records say about it, not a checklist. Each statement says how it is known: **verified** (a command run for this record), **reported** (a named record says it).

## The tag

- **Verified (`git cat-file -p v0.1.0`, `git log -1 v0.1.0`).** `v0.1.0` is an annotated tag by `zxdev`, re-created 2026-11-05 on `a8dc02b` ("Add the Apache-2.0 LICENSE"), whose only parent is `9364fb0`
  ("docs: benchmark baselines re-recorded on both machines after the allocation pass", the commit the tag was on, dated 2026-10-29, until 2026-11-05) and whose only change is the added `LICENSE`. The code of the tag
  is the code of `9364fb0`. The tag's original message, kept and followed by a paragraph that says it was re-tagged: "The first release: the on-disk format quill/1, the Log, the Reader, consumers, retention, Archive and the quill command, built in four phases from docs/design, then the A1 adversarial audit
  (46 candidates, 39 fixes) and the allocation pass (22 candidates, 12 fixes). Every gate green on macOS and Linux, T14 run on a real full filesystem, three real power cuts survived, baselines recorded on both
  machines." The message's counts are the tagger's, not re-derived here.
- **Verified.** The tag's history: on `419e15a` when created on 2026-10-29; moved to `9364fb0` in the history rewrite of 2026-11-04 ([commit-map.txt](commit-map.txt); the two trees differ in one audit document only);
  pushed to `netstar-labs/quill` on 2026-11-05 on `9364fb0`; moved once more, before the repository was public, to `a8dc02b` the same day (object `497b133`, the tag on `9364fb0`, is replaced on the remote; a local copy of
  it was kept until the owner was satisfied). The tag is `v0.1.0` throughout; the commit it names changed twice.
- **Verified (`git ls-remote origin`, `gh release list`).** The tag is on `netstar-labs/quill` so that `hack/skew.sh v0.1.0 HEAD` has its older ref. No GitHub release was made for it: the repository's one release
  is `v0.2.0`. No archives, `SHA256SUMS` or SBOM exist for `v0.1.0`.
- **Verified (`git ls-tree -r`, `shasum -a 256`).** `9364fb0` has **no `LICENSE` file**. The Apache-2.0 `LICENSE` of `v0.2.0` was written after it. `a8dc02b` adds that same file (sha256 `cfc7749b…`) to `9364fb0`, so
  the tag now carries the licence and the tag's source before that move did not. The owner is the author and chose to license the older tag (2026-11-05). `a8dc02b` was merged into `main` by `0caf868`, a merge that
  changes no file (verified: the tree of `main` is identical before and after it), so `v0.1.0` is an ancestor of `main` again.
- **Verified (`hack/skew.sh`, 2026-11-05).** `hack/skew.sh 9364fb0 <main>` and `hack/skew.sh a8dc02b <main>` both end `OK: no divergence`; `git rev-list --count v0.1.0..v0.2.0` is 304 for the old and the new tag commit.
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
- **Verified (`git diff --shortstat v0.1.0 v0.2.0 -- . ':!docs'`).** 188 files changed outside `docs/`, 22,215 insertions, 1,097 deletions. Against `9364fb0`, the commit the tag named before it gained the `LICENSE`, the same command gives 189 files and 22,417 insertions; the difference is the 202-line `LICENSE`, which the tag now contains.
- The user-visible changes, and what an upgrader must do, are under "Breaking changes and upgrade notes" at the top of the `v0.2.0` section of the CHANGELOG and in the migration table of
  [release-v0.2.0.md](release-v0.2.0.md). Format compatibility between the two is [format-compatibility.md](reference/format-compatibility.md); `hack/skew.sh v0.1.0 HEAD` is the cross-version check.

## What was not done for v0.1.0

No release rehearsal of `v0.1.0` is recorded here (the rehearsal in [release-v0.2.0.md](release-v0.2.0.md) is of v0.2.0), and no archives were built or checksummed for it (verified above). The live
validation of [passes/live-integration.md](passes/live-integration.md) ran from 2026-11-03, on later trees; GitHub CI first ran on 2026-11-05, on the `v0.2.0` push.
