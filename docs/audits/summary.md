# Summary: what the audits found and what is still open

One page. Everything here is read from the records it names; nothing was re-derived for this page except the counts in the first
table. The live record is [release-v0.2.0.md](release-v0.2.0.md), which carries the full lists. How the audits were run, and what the
words mean, is in [README.md](README.md).

## What was done

The audit sequence (the passes in [README.md](README.md)) examined v0.1.0 and then the v0.1.0..HEAD diff: correctness, simplification, duplication, documentation, security,
supply chain, silent degradation, the on-disk format and version skew, the test suite itself (mutation testing), live multi-process
behaviour on macOS and Linux (with a Linux filesystem matrix), and a gap audit of what the others could not see. Most candidate findings
were attacked by independent skeptics before anything was changed, and most fixes were attacked again before they landed. The exceptions
are in the records: the pre-tag batches L4b and L5 carry no skeptic votes (the owner approved their items directly), and the
hand-finished fixes listed below had no fix-skeptic re-run on their final state. The records hold
29 Markdown files of about 16,500 lines.

Distinct ids that appear in the records, by pass (counted with a search of `docs/audits/` for the pass prefix, a hyphen, digits and one
optional trailing letter, so a follow-up note such as `A1-62r` counts as its own id; without those, A1 is 97, A5 is 45 and L1 is 53. An id
is a numbered candidate, not a confirmed defect, and the status of each is in its record):

| A1 | A1d | A3 | A4 | A4r | A5 | A6 | A7 | L1 | L2 | L3b | L4b | L5 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 99 | 14 | 33 | 10 | 5 | 46 | 9 | 25 | 54 | 58 | 15 | 4 | 5 |

## What the release record reports as done

- **Build, gate and reproducibility** were green in the rehearsal of the v0.2.0 release (its gate ran on macOS; the pre-tag batches
  after it were gated on macOS and on Linux, the Linux runs on tmpfs only).
- **Pre-tag changes, owner-approved 2026-11-04:** `LICENSE` is Apache-2.0; the Go licence text ships in every archive as
  `THIRD-PARTY-NOTICES`; only 64-bit builds are supported and a 32-bit target fails to compile; `quill -h`, `--help` and `--version`
  print and exit 0; the `quill archive` message says "the archive does not have them"; the history was filtered before any push to
  remove a scratch path ([commit-map.txt](commit-map.txt)).
- **The user-visible changes since v0.1.0** (rebuilt from the CHANGELOG and the git log by audit item L4b-1, whose CHANGELOG line calls
  the list "a best effort, not a proof of completeness") are listed under "Breaking changes and upgrade notes" at the top of the CHANGELOG's
  Unreleased section and in the migration table of the release record.

## What is still open at the tag

State words are the release record's: **open** stays unless someone acts, **accepted** was decided and left, **deferred** is after the
release. The full table, with the section of the record that holds each item, is "Known limitations and open items at the tag" in
[release-v0.2.0.md](release-v0.2.0.md).

- **Open, no fix written:** A3-7 (a symlinked `consumers/` directory is followed), A3-24 (the printed `time` field has no range check),
  A3-26 (an unlinked `quill.lock` lets a second writer open until the first writer's next flush or prune pass).
- **Open, other:** A3-10 points 1, 3 and 4 (the fix-skeptic's open points on `archive`, `tail` and `check` around a damaged oldest
  segment) and the optional `Archive` refusal (L1-12, optional, not approved).
- **Accepted:** A1d-1 (a live retention pass syncs the directory once per deleted segment while holding the log's lock; the cost is not
  documented), A5-5 (a long-lived `Reader` does not notice that its directory was replaced), A4r-3 (the build script accepts any tag
  name).
- **Deferred:** A3-18 (a `Reader` read costs in proportion to the log, not the request).
- **Finished by hand, no fix-skeptic re-run on the final state:** A7-18 and A7-19 (the upgrade note and the README install section),
  L1-3, L1-4, L1-10 and the last comments of A3-10. Also finished by hand with no fix-skeptic: the history filter, the `LICENSE`
  commit and the `Reader` change kept at the owner's word (a view overtaken by a rotation restarts), recorded in the ledger section
  "Owner decisions 2026-11-04, after wave 7".
- **Public-flip items, the owner's:** the `CODEOWNERS` handle; the `SECURITY.md` contact (it relies on GitHub private vulnerability
  reporting, which does not exist until the repository does) and its internal wording at lines 31-32; repository metadata, and that the
  repository must live at exactly the module path `github.com/netstar-labs/quill`; branch protection and the `ci` check; the 484 local
  scratch branches (push `main` and the tag by name only); the commit `da2d638` ("x", a placeholder author) and whether the one GitHub
  noreply identity on the other commits is meant to be public; operator detail in `docs/design/audit-prompt.md`; and the reference
  implementation's description in `docs/design/review.md` and elsewhere (A4r-5). **At tag time**, also run govulncheck again: the gate
  does not run it.
- **At tag time:** the CHANGELOG heading is still `## Unreleased` and must be retitled in the release commit.

## What was not run

The main items the release record lists as not run (its "What was not tested, and why" and "What was not run at the tag" sections
have the full list): the power-cut replays and the Linux filesystem matrix were run once, in pass L3b,
on an earlier tree, and were not re-run on the tag (deferred; a power cut was not possible); there was no soak of hours and no log
with 10^5 segments; `go install ...@v0.2.0` from the module proxy, branch rulesets and private vulnerability reporting cannot run
until the repository is public; the library was not run on any 32-bit or big-endian machine, and 64-bit targets other than arm64 and
amd64 are built but untested; the L5 code fixes were checked by their fix-skeptics with partial checks, not the whole gate; and there
was no native darwin/amd64 run (Rosetta, `quill version` only), no verification of published artifacts, no build on a GNU tar and gzip
toolchain, and no run of GitHub CI, whose first execution will be the first push.
