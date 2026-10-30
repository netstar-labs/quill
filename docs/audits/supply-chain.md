# A4 audit: dependency, supply-chain and release hygiene

Date 2026-10-29, against v0.1.0 (commit 419e15a) and main as it stood when the auditor read it. Pass A4 of the
stage 2 audit, run with fresh eyes before a public release. One auditor covered the module graph, vulnerabilities,
licence and notice state, the build script, CI and the public-release hygiene of the tree and its history; ten items
were raised, and every one was attacked by two or three skeptic votes. Fixes were re-attacked by a fix-skeptic. What was
held back for the owner is in [stage2-owner-decisions.md](stage2-owner-decisions.md).

Labels: **verified** means an auditor or skeptic ran it or read the exact lines; **reported** names the document that says
it; **inferred** says from what. Line anchors are to the audited tree unless a sentence says "now"; the A4 fixes have since
moved some lines, notably in `build/quill` and `.github/workflows/ci.yml`. Throwaway programs ran against copies of the
repository in the session scratch area (written `scratchpad/<label>` below); nothing under audit was edited by an auditor
or a skeptic.

## Top line

Ten items, judged by **21 skeptic votes** (17 CONFIRMED, 4 REFUTED, no PLAUSIBLE). **Six of the ten items were reproduced by
execution** by at least one skeptic (A4-2, A4-4, A4-5, A4-7, A4-8, A4-9); nine of the 21 votes carry a reproduction. The
auditor's own reproductions are in the items and are not counted.

| verdict | items | ids |
|---------|-------|-----|
| CONFIRMED | 8 | A4-1, A4-3, A4-4, A4-5, A4-7, A4-8, A4-9, A4-10 |
| REFUTED | 2 | A4-2 (by 2 votes to 1), A4-6 (by 2 votes to 0) |

| disposition | items | ids |
|-------------|-------|-----|
| fixed, fix-skeptic FIXED-VERIFIED | 4 | A4-7, A4-8, A4-9, A4-10 |
| fixed in the tip only, fix-skeptic FIXED-VERIFIED; history is held for the owner | 1 | A4-4 |
| fix landed on main without a fix-skeptic (see A4-3) | 1 | A4-3 |
| fixed in this commit for the `build/quill` half; the CI half was refuted at the pinned setup-go (see A4-1) | 1 | A4-1 |
| held for the owner, no fix | 1 | A4-5 |
| deliberately not changed (REFUTED) | 2 | A4-2, A4-6 |

Severity as the auditor assigned it: medium 5 (A4-1, A4-2, A4-3, A4-4, A4-6), low 5 (A4-5, A4-7, A4-8, A4-9, A4-10). The
skeptics put A4-1, A4-3 and A4-4 at "low to medium". By the skeptics' reading nothing in the pass is a vulnerability, a wrong
binary or a broken guarantee; the findings are about whether the documented pins, notices and contacts are true, and about what
leaves the building at the public flip.

Two disposition records need a plain statement, because the workflow that applied the fixes did not run cleanly for them
(verified from `git log` and `git branch --contains` now):

- A4-3. The fix workflow's own report says the commit "did not land" because its pre-merge status check saw a modified
  `go.mod`, and that the fast-forward happened anyway. The commit is on main as f2516d7. No fix-skeptic ran on it and nobody
  ran `go build` or `go vet` on it. It changes only `SECURITY.md` and `CHANGELOG.md` (11 insertions, 2 deletions, verified
  with `git show --stat`), so no code path is affected.
- A4-1. The first fix commit f800bb7 (branch `stage2/A4-1-retry`, based on da2d638) was not landed as written: the fix-skeptic of a
  later attempt (41c8aa4) showed that the pinned setup-go already reads go.mod's `toolchain` line, so its CI half was false and was dropped; the
  `build/quill` half is fixed in this commit, with a reworded `ci.yml` comment (see `stage2-owner-decisions.md`). The stray `go.mod` change that blocked it was an audit probe committed straight to main as da2d638 (message
  "x") that set `toolchain go1.27.1`; c202882 restored `toolchain go1.27.3` (verified: `go.mod` now reads go1.27.3, and the
  `ci.yml` comment about the patch level was thought false; it is true at the pinned setup-go v7.0.0 and is reworded in this commit to say why).

## Coverage

Checked and found clean (all verified by the auditor unless marked otherwise):

- Module graph. `go.mod` has no `require` line and no `go.sum` exists. `go mod tidy` leaves no diff. `go mod verify` reports
  all modules verified. `go list -deps ./...` and the `-test` form contain only this module's packages and the standard
  library; `crypto/rand` is used by `internal/atomicfile` and `state.go`; non-test code imports no `net`, `os/exec` or cgo,
  and the binary has no network packages.
- Vulnerabilities. govulncheck v1.9.0 on a scratch copy of HEAD printed `No vulnerabilities found`; the verbose run scanned
  the go1.27.3 standard library and all 9 packages. go.dev/dl lists go1.27.3 as the newest stable release. staticcheck v0.8.2,
  deadcode v0.52.0 and govulncheck v1.9.0 are each the newest tag, and the staticcheck and deadcode pins agree between
  `hack/gate.sh` and `ci.yml`.
- Release script refusals, tried in scratch clones. An untagged HEAD, a modified tracked file and an untracked file each exit 1
  with a message; a clean checkout of v0.1.0 builds; an ignored file does not make the tree dirty (intended). `--dev` builds
  name the version `dev-<sha>` and `dev-<sha>-dirty`. `quill version` prints v0.1.0, and `go version -m` shows `-trimpath=true`,
  `CGO_ENABLED=0`, module v0.1.0 and no dependencies. Both `-s -w` and the version ldflag take effect.
- Binary identity. Binaries are byte-identical across two clones, and across a Mac host and the Linux box, for all four targets,
  when VCS stamping is made equal with `-buildvcs=false`. `SHA256SUMS` verifies (`shasum -c`, 5 of 5 OK). The SBOM is valid JSON
  and passes the CycloneDX 1.5 schema; it lists exactly the module and the Go standard library, matching `go version -m` and
  `go list -deps`. The tarballs carry no pax extended-attribute headers (Python `tarfile`).
- CI. Every `uses:` is pinned by a 40-hex commit SHA and each SHA equals its stated tag per `git ls-remote` (checkout 11d5960a is
  v4.4.0, setup-go 40f1582b is v5.6.0). `ci.yml` sets top-level `permissions: contents: read`. The aggregator `ci` job (needs
  all five jobs, `if: always()`) fails for skipped, failed and cancelled results and passes only when all succeed (tested under
  bash with the exact expression). `pr-issue-link.yml` passes the pull request body through `env`, so it has no injection path.
  The trigger set (push to main, `v*` tags, pull requests) is as intended.
- History-wide hygiene. Across all 124 commits on all branches (the auditor's count; the skeptics, counting main only, saw 120
  at that time) and the tag message, no commit message or tree contains any of the project's internal product, repository or
  path names, a co-author trailer or a tool attribution, except the one workspace path in A4-4. The only hits for the shortest
  forbidden word are inside ordinary words such as "describe". Tracked files contain no hostnames, IP addresses, e-mail
  addresses other than the GitHub no-reply address, `/home` or `/Users` paths, tokens, keys or passwords (the secret-pattern
  grep found only `Token()` calls in the strict JSON reader). The golden fixtures (`testdata/golden`, 500 bytes in all) hold no
  text of concern.
- Repository state. `git ls-files -ci --exclude-standard` is empty (nothing tracked is ignored); there are no untracked
  non-ignored files and no symbolic links; the only executables are `build/quill`, `hack/fuzz.sh` and `hack/gate.sh` (mode
  100755); no tracked file is over 200 kB. `.gitignore` covers `build/install`, `build/quill.version`, `*.test`, `docs/plan` and
  the local tool directory. The tag v0.1.0 is an annotated tag at 419e15a. The unmerged `stage2/*` branches hold stray
  `archive_test.go-E` and `reader_vanished_test.go-E` sed-backup files; none is on main.

Not covered:

- GitHub-side settings: whether private vulnerability reporting is enabled, whether branch protection requires the `ci` job, and
  the repository's default token permissions. None is visible from the tree or the API the auditor had, and none is asserted.
- The full gate and the test suite were not run by the auditor, and the auditor did not read the earlier audit conclusions, as
  instructed. The Go 1.23 build check (A4-2) covered `.` and `./cmd/...` only.
- Side effect, outside the repository: building with go1.23.12 made the go command write a toolchain into the user-level Go
  module cache. No repository file was modified, and nothing was pushed or tagged.
- Not examined by any skeptic: the node20 retirement timetable (inferred, A4-9) and the CycloneDX schema validity of the
  `bom-ref` fields added by the A4-8 fix (inferred from the field being standard in 1.5; the test checks that references
  resolve, not schema validity).

Auditors that failed to return: none.

## Module graph

Verified. The module `github.com/netstar-labs/quill` declares `go 1.27` and `toolchain go1.27.3` (`go.mod:3` and `:5`), has no
`require` line and no `go.sum`. The build graph is this module's own packages plus the standard library, including
`crypto/*`; the release binary links the Go runtime and standard library statically (`CGO_ENABLED=0`). The CI job "no module
dependencies, ever" fails on any `require` line, and `go mod tidy` is a diff check.

## Vulnerabilities (govulncheck)

Verified. govulncheck v1.9.0 against HEAD with go1.27.3: no vulnerabilities, so no reachability analysis was needed. In verbose
mode it scanned the go1.27.3 standard library and the nine packages of the module. go1.27.3 is the newest stable release on
go.dev/dl at the time of the pass. The check is a result at a point in time against a live database; `hack/gate.sh` deliberately
does not run it (A4-10), CI does.

## Licence and notice state

| state | evidence | verdict |
|-------|----------|---------|
| no `LICENSE`, no `THIRD-PARTY-NOTICES` | `git ls-files` (verified); decision 22 says so on purpose, README Status says so | intended while private (A4-6 REFUTED) |
| release archives carry README, CHANGELOG, SECURITY and `LICENSE` if present | `build/quill:63-65` now | a missing `LICENSE` is skipped silently; intended in the private period |
| archives ship the Go runtime and standard library (BSD-3-Clause) with no copy of that licence | A4-5 | CONFIRMED, held for the owner |
| a `LICENSE` lands before the repository goes public | decision 54 (reported) | not yet done; an owner action |

## Build and CI findings

A4-1 (pin not enforced), A4-7 (non-reproducible archives), A4-8 (dangling SBOM graph), A4-9 (CI action pins and token scope) and
A4-10 (untested release refusal) are below. The CI action pins were exact and correct as audited; the finding is that they were
behind and nothing moved them.

## Public-release hygiene

Verified. See Coverage for the clean checks. Findings: A4-3 (no usable private reporting channel), A4-4 (a workspace path in a
tracked document and in every commit since it was added), A4-5 (notice), A4-6 (refuted). For a flip, only main and the tag
v0.1.0 would be needed; the unmerged `stage2/*` branches must not be pushed (they hold the `-E` backup files). The tag v0.1.0
is at 419e15a and main is now 52 commits ahead (verified with `git rev-list --count v0.1.0..HEAD` in the worktree this record
was written in); `build/quill` correctly refuses to build HEAD as untagged.

## Items

### A4-1: the Go toolchain pin is enforced only by the gate (medium; CONFIRMED, 2 votes, 0 reproduced; the `build/quill` half fixed in this commit; the CI half REFUTED)

**Claim.** The pin `toolchain go1.27.3` in `go.mod` is enforced only by `hack/gate.sh`. CI installs the newest 1.27.x because
setup-go ignores the toolchain line, and `build/quill` builds releases with whatever `go` is on PATH. The comment at
`ci.yml:66-67`, which says the patch level comes from go.mod's toolchain line, is false for the pinned setup-go.
**Correction (this commit): the CI half is REFUTED.** At the setup-go version `ci.yml` pins now (v7.0.0, from the A4-9 change)
the action reads go.mod's `toolchain` directive before the `go` line unless `GOTOOLCHAIN=local`, so CI already installs the
pinned patch release and the `ci.yml` comment is true (reported by the fix-skeptic from the action's `src/installer.ts` at b7ad1dad;
the pin is at `ci.yml` lines 24, 44, 54, 75 and 87; not run on a runner). The "ignores the toolchain line" statement was true of
the v5.6.0 pin the auditor read, and is not true of the pin on main. The `build/quill` half stands and is fixed.

**Grounds.** (a) `hack/gate.sh:15-18` exports `GOTOOLCHAIN` from go.mod, so the gate is pinned exactly; `build/quill` set only `GOWORK`
and `CGO_ENABLED`, and recorded `go version` of PATH in the SBOM. (b) setup-go at the pinned SHA 40f1582b (v5.6.0), `src/installer.ts`,
`parseGoVersionFile`, matches only `^go (\d+(\.\d+)*)` for a go.mod, so `go 1.27` resolves to the latest 1.27.x and the toolchain line is
never read (**true of v5.6.0 only; REFUTED for the v7.0.0 pin now on main, which reads the `toolchain` line first, see the Claim**);
`ci.yml` uses `go-version-file: go.mod` in five jobs (now lines 26, 46, 56, 77 and 89). (c) Auditor's reproduction in a
scratch copy: with `toolchain go1.27.1` in go.mod, plain `go version` printed go1.27.3 (the newer local toolchain wins under
`GOTOOLCHAIN=auto`), and only `GOTOOLCHAIN=go1.27.1` forced 1.27.1.

**Warrant.** Under `GOTOOLCHAIN=auto` the toolchain line is a minimum, not a pin. The gate turns it into a pin; the release
script did not, so the gate and a release could run different Go versions. (CI is not part of this warrant: **the CI half is REFUTED**,
because the pinned setup-go reads the toolchain line, see the Correction in the Claim.) The release binary's bytes depend on the compiler, and the
SBOM records which one.

**Backing.** README.md:83-86 and CONTRIBUTING.md:40-42 say the toolchain named in go.mod is the one supported Go version and moves
together with the gate and CI pins; decisions 70 and 88 make the build script the release authority.

**Rebuttal.** Today all four places resolve to go1.27.3 (verified against go.dev/dl), govulncheck is clean, the SBOM discloses the Go
actually used, and an older local Go would auto-upgrade, so the toolchain line still works as a floor. The defect that stands is that the release script did
not honour the documented single pin; CI did (see the Correction in the Claim).

**Verdict.** CONFIRMED, 2 of 2. Both voters read the files and the setup-go source and neither re-ran the auditor's experiment
(reported by the auditor, not reproduced). Both put the real severity at low to medium.

**Disposition.** Fixed in this commit, for the `build/quill` half only (owner-approved 2026-10-30): `build/quill` derives and exports
`GOTOOLCHAIN` from go.mod's toolchain line as `hack/gate.sh` does, so the release binaries and the SBOM come from the pinned compiler;
`TestBuildScriptPinsToolchain` fails without it. The CI half was refuted (see the Correction in the Claim): the pinned setup-go already
reads the toolchain line, so nothing in CI changed and the `ci.yml` comment was reworded to say why it is true. The earlier attempts
did not land: f800bb7 (branch `stage2/A4-1-retry`, based on da2d638) added a workflow-level `GOTOOLCHAIN` literal and a test for
it, which rested on the false CI claim, and the next attempt (41c8aa4) repeated the false claim. Decision 88, the README, CONTRIBUTING and
the changelog are amended. See [stage2-owner-decisions.md](stage2-owner-decisions.md).

### A4-2: go.mod says `go 1.27` while the docs say the iterator API needs Go 1.23 (rated medium by the auditor; REFUTED, 2 votes to 1)

**Claim.** `go 1.27` makes every importer need Go 1.27 or later, while README and CONTRIBUTING say the iterator API "needs Go 1.23 or
later" and the code builds on 1.23.

**Grounds.** `go.mod:3`. Reproduction (verified, the auditor and the CONFIRMED voter independently): a consumer module on `go 1.23` with a
`replace` to the repository, built with go1.23.12 and `GOTOOLCHAIN=local`, fails with `go: module ../src requires go >= 1.27 (running go
1.23.12; GOTOOLCHAIN=local)`; with the copy's go line lowered to 1.23 it builds. The library and `cmd/quill` compile on 1.23.12. The
`cmd/quill` tests do not compile there (`b.Loop` is Go 1.24).

**Warrant.** Since Go 1.21 the go line is a hard minimum for importers.

**Backing.** README.md:83-86, CONTRIBUTING.md:40-42, docs/design/scope.md:94.

**Rebuttal (the refuting reason).** The behaviour is the stated policy: the toolchain named in go.mod is the one supported Go version and
quill tracks the latest stable release (README.md:83-86, CONTRIBUTING.md:40-42). Decision 58 says the iterator API "needs Go 1.23 or later,
which the latest-toolchain rule already satisfies", so the author knew the two numbers differ. The sentence about 1.23 says when the language
feature appeared; it is not a consumer floor. The reproduction is not disputed, only the characterisation as a defect.

**Verdict.** REFUTED (2 REFUTED, 1 CONFIRMED). What survives is a wording ambiguity: `scope.md:94`, README.md:86 and CONTRIBUTING.md:42 do not
say outright that consumers need Go 1.27 or later because of the go line. Whether a public release wants a lower consumer floor is a product
decision, not a defect against the stated policy.

**Disposition.** Deliberately not changed, because the high go line is the documented policy. The one-line wording clarification is an optional
follow-up (see the action list); it is not held as a decision.

### A4-3: SECURITY.md gives no usable private reporting channel (medium; CONFIRMED, 2 votes, 0 reproduced; fix landed unchecked)

**Claim.** `SECURITY.md` said to report privately to "the repository owner (see `.github/CODEOWNERS`)"; CODEOWNERS holds only one GitHub login, with no
e-mail, advisory URL or key.

**Grounds.** `SECURITY.md:8-9` and `.github/CODEOWNERS:6-12` as audited. A grep of tracked files for "advisor", "private vulnerability" and e-mail
addresses found nothing relevant. `CODEOWNERS:1-4` itself says GitHub ignores unresolved owners without error.

**Warrant.** A login is not a private channel; the only private route GitHub offers is private vulnerability reporting, which must be enabled and
which the file did not mention. A reporter's remaining choices were a public issue (forbidden by the file) or guessing an address.

**Backing.** `SECURITY.md:8` ("acknowledgement within a week"); decision 54.

**Rebuttal.** If the owner enables private vulnerability reporting at the flip, the Security tab supplies the channel. The setting is not
verifiable from the tree, and the file should say so either way. Both voters thought the severity low to medium.

**Verdict.** CONFIRMED, 2 of 2; documentation finding, no reproduction applies.

**Disposition.** Fixed in f2516d7 (on main): `SECURITY.md` now names the Security tab "Report a vulnerability" route, gives a fallback if the option
is not shown (open an issue asking for a private contact without describing the problem) and states that only the latest release is supported; a
changelog line says the owner must enable the setting before the flip. No fix-skeptic ran (see the top line), so the outcome is "landed, not
re-attacked"; the diff is documentation only (verified). What the fix cannot do is enable the setting: that is the owner's action, held in
[stage2-owner-decisions.md](stage2-owner-decisions.md). Note that the tag v0.1.0 still carries the old wording (verified: `git show v0.1.0:SECURITY.md`).

### A4-4: a tracked document names a personal workspace path (medium; CONFIRMED, 2 votes, both reproduced; tip fixed, history held)

**Claim.** `docs/audits/audit-correctness.md` named an absolute session path that carries a macOS user name and an internal parent-directory name,
and it was present in every commit since the file was added, including the tagged v0.1.0 tree.

**Grounds.** `docs/audits/audit-correctness.md:6-8` as audited (the path itself is not repeated here). A loop over every commit tree matched only that
file for the project's forbidden names and for `/Users` and IP patterns, in 75 commits (the auditor's count; one voter counted 120 commits in all, with
the file added at 98ff9e7, and did not recount the 75). Tracked files at HEAD contained no other match.

**Warrant.** Tracked text names a private workspace layout. The tag and main history both contain it, so a public flip exposes it unless the text is
edited in the tip and history is rewritten or squashed, or the owner accepts it. A tag must not be moved here, so removing it from the tagged tree needs
a rewrite of history (a later tag on the same history would not remove it).

**Backing.** The project rule that no internal product, repository or path name appears in text that ships. Decision 54 (release gates) is weaker
backing than the finding stated: it lists LICENSE and similar gates and does not itself forbid paths (verified by one voter at `decisions.md:62`).

**Rebuttal.** The parent-directory name is not on the explicit forbidden-name list and a session scratch path is not a secret, so the exposure is low; the
finding stands only if the rule covers workspace paths and user names, which both voters read it to do.

**Reproduction (verified, both votes).** `git grep -nIE` for the path fragments at HEAD, at the v0.1.0 tag and over every commit from `git rev-list --all`,
read only: one file matches, at HEAD and at the tag.

**Verdict.** CONFIRMED, 2 of 2.

**Disposition.** Fixed in the tip: the paragraph now says "under the session scratchpad (written below as `scratchpad/`)" and a changelog line records it.
Fix-skeptic FIXED-VERIFIED on the auditor-side commit 007ba0e (branch `stage2/A4-4`); the same change is on main as 0b70e82 (verified from `git log`).
The skeptic's six lenses passed: only the two files changed, no sibling copy of the path remains in the tree, gofmt and vet were clean, and the race tests
passed. The skeptic's objection stands as the residual: the text is still in the v0.1.0 tag and in earlier history, so the flip stays exposed until the
owner squashes or filters history, or accepts it. No gate check stops a path from being reintroduced (deliberately not added). History is
held for the owner.

### A4-5: the archives ship the Go runtime with no copy of its licence (low; CONFIRMED, 2 votes, 1 reproduced; held for the owner)

**Claim.** The release archives contain a statically linked Go runtime and standard library (BSD-3-Clause) with no copy of that licence or notice, so
decision 22's "no third-party code ships, so no THIRD-PARTY-NOTICES" does not hold for the binaries.

**Grounds.** The archive step copies README, CHANGELOG, SECURITY and `LICENSE` if present (`build/quill:63-65` now; `:44-48` as audited). `go version -m` of
an extracted binary shows go1.27.3 and `CGO_ENABLED=0`. `go list -deps ./cmd/quill` lists only standard-library packages. `decisions.md:30` (decision 22).

**Warrant.** BSD-3-Clause redistribution in binary form requires the notice and licence text in the documentation or other materials. Verified by one voter
reading clause 2 of the licence in the local GOROOT, and by listing an existing archive (the binary and three documents).

**Backing.** Decisions 22 and 15: decision 15 is about not vendoring source and says nothing about the linked runtime (verified at `decisions.md:23`).

**Rebuttal.** Decisions 22 and 15 are accurate about vendored source; decision 22 is explicitly a private-period rule, and decision 54 already requires a
`LICENSE` before the flip, but that would be quill's own, not Go's. Practice around unnotified Go binaries is lax, and the owner may judge the risk negligible.

**Verdict.** CONFIRMED, 2 of 2; the owner's reading of the duty (a link or a copy) is a policy call.

**Disposition.** Held for the owner, with the recommended answer: at the flip add the Go licence text to each archive and to the repository (or a
THIRD-PARTY-NOTICES file naming the Go standard library), and reword decisions 22 and 15. No fix written; it is a semantic change to a recorded decision.

### A4-6: `build/quill` does not enforce a LICENSE and README Status reads "Private" (rated medium by the auditor; REFUTED, 2 votes to 0)

**Claim.** There is no LICENSE, as decision 22 says, but the release build silently builds archives without one and README Status still reads "Private,
pre-release".

**Grounds.** `git ls-files` shows no LICENSE or notice file. `build/quill:47` as audited (now `:65`) skips a missing LICENSE without a message. The auditor
built the v0.1.0 tag in a scratch clone and got four archives and a checking `SHA256SUMS`, none with a LICENSE. `README.md:97-99`.

**Warrant.** An unlicensed public archive is all rights reserved, which decision 54 names as the problem.

**Rebuttal (the refuting reason).** Decision 54 does not ask the build script to refuse a missing LICENSE; it names a build that refuses a dirty or untagged
tree, which the script does. The README Status matches decisions 22 and 70 and the repository is still private, so it is current. The auditor's own rebuttal
conceded the absence is deliberate and owner-controlled. Neither voter rebuilt the archives.

**Verdict.** REFUTED, 2 votes to 0.

**Disposition.** Deliberately not changed: a guard would break the release build that decisions 22 and 70 call for during the private period. The residual is a
flip-time checklist: add LICENSE (and the A4-5 notice), update the README Status, confirm the SECURITY contact, and tag a new version. An opt-in or
flip-time guard in `build/quill` is a hardening idea for the owner, not a defect.

### A4-7: release archives and the SBOM were not reproducible and carried the builder's account (low; CONFIRMED, 2 votes, both reproduced; fixed)

**Claim.** The binaries are byte-identical across builds and hosts, but the archives and the SBOM were not, and the archives embedded the builder's account name.

**Grounds.** Two builds of v0.1.0 in two scratch clones, three seconds apart, gave different checksums for all four tarballs and the SBOM. Causes: the SBOM
`metadata.timestamp` taken from the wall clock (`build/quill:36` as audited), tar entry mtimes, and the gzip header mtime. Python `tarfile` showed uid 501, a user
name and the group `wheel` or `staff`. The inner binaries had identical hashes in all four targets across the two clones, and a build on the Linux box with
`GOFLAGS=-buildvcs=false` matched a Mac build with the same flag. Without that flag the binary embeds `vcs.revision` and `vcs.time`, which differ for a build from a
git archive with no `.git`.

**Warrant.** A verifier cannot rebuild an archive and compare it with `SHA256SUMS`; only the contained binaries are comparable. Extracted as root, GNU tar restores the
uid and name, so builder identity is published in every archive.

**Backing.** `build/quill` as audited (lines 36, 48, 86); decision 88. No document claimed reproducibility, so this is hardening, not a broken claim.

**Rebuttal.** The docs make no reproducibility promise and binary identity (verified) is the property that matters most; `SHA256SUMS` was still correct for the files
produced.

**Reproduction (verified, both votes).** Voter 1 ran `tar -czf` with the script's options on a small directory and listed it: owner and group names and the current mtime.
Voter 2 extracted `git archive HEAD` into two scratch directories, made a repository and the tag in each, ran `GOFLAGS=-buildvcs=false build/quill` in each two seconds
apart: all five `SHA256SUMS` lines differed, the extracted binaries were identical, and `tarfile` showed uid 501, a user name and gname `wheel`.

**Verdict.** CONFIRMED, 2 of 2.

**Disposition.** Fixed in 10e8062 (on main), fix-skeptic FIXED-VERIFIED with seven lenses. The SBOM timestamp and every entry mtime are the commit time, entries are
sorted, format is ustar, owner is 0 with no names, gzip runs with `-n`, modes are fixed. `TestBuildScriptReproducibleArchives` fails on the parent's `build/quill` (two builds of
one commit differ) and passes with the fix on macOS and on the Linux box (GNU tar 1.35). Honest limit, recorded in decision 88, the changelog and the script comment: archive
equality holds only for the same tar, gzip and Go, not across GNU tar and bsdtar; across systems the extracted files are the comparison, and `-buildvcs=false` is needed on both
sides for binaries. The skeptic confirmed the extracted documents were identical across the two hosts but did not compare binaries across hosts for the fixed tree (different commits).
busybox tar is untested and takes the bsdtar branch.

### A4-8: the SBOM dependency graph named identifiers no component declared (low; CONFIRMED, 2 votes, 1 reproduced; fixed)

**Claim.** The SBOM is schema-valid CycloneDX 1.5, but `dependencies[].ref` and `dependsOn` named identifiers that no component declared as a `bom-ref`.

**Grounds.** `build/quill:81` as audited emitted a `dependencies` entry with purl strings, while neither `metadata.component` nor the stdlib component had a `bom-ref`. The
auditor validated the SBOM against the CycloneDX 1.5 JSON schema (Draft-07): no errors; a script collecting `bom-ref` values found the empty set.

**Warrant.** CycloneDX defines dependency refs as references to a component's `bom-ref` (reported from the specification, not re-read here). A consumer that resolves the graph
finds no edge.

**Backing.** Decision 88 requires an SBOM naming the module and the toolchain; it asks for no graph, but does not make dangling refs intended.

**Rebuttal.** Schema validation passes, and the component list is accurate: it lists exactly what `go version -m` and `go list -deps` show.

**Reproduction (verified, voter 2).** A `build/quill --dev` run in a scratch copy, then a Python script comparing every `dependencies` ref and `dependsOn` entry against the
set of `bom-ref` values: both reported dangling. Whether a real consumer drops the edge is inferred from the specification and untested.

**Verdict.** CONFIRMED, 2 of 2.

**Disposition.** Fixed in 1b17ea5 (on main), fix-skeptic FIXED-VERIFIED with seven lenses: both components carry a `bom-ref` equal to their purl and the test resolves every
reference against the declared set. Without the fix the test fails with the two refs unresolved. Residual: schema validity of the new field was inferred, not re-run; the
optional BSD-3-Clause licence on the stdlib component was not added.

### A4-9: CI action pins were exact but old, nothing moved them, and one workflow had no `permissions` (low; CONFIRMED, 2 votes, both reproduced; fixed)

**Claim.** The pins are exact and correct but several majors behind and on the node20 action runtime, with no dependabot or renovate configuration; `pr-issue-link.yml` declared
no `permissions:` block.

**Grounds.** `ci.yml:24, 44, 54, 74, 86` as audited (the pins are now at 24, 44, 54, 75 and 87) pinned checkout 11d5960a (v4.4.0) and setup-go 40f1582b (v5.6.0). `git ls-remote` shows both equal their tags. Upstream `action.yml` at
those SHAs says node20, while checkout v7.0.1 and setup-go v7.0.0 say node24. No `dependabot.yml`. `pr-issue-link.yml` had no top-level or job `permissions`, whereas `ci.yml:10-11` sets
`contents: read`. `ci.yml:15` set a `GOPRIVATE` value that means nothing for a module with no dependencies.

**Warrant.** A workflow without `permissions` gets the repository's default token scope, which may be read and write on older repositories and organisations; it has no checkout or write
needs, so it should declare none. The node20 point is that GitHub is retiring node20 for actions (inferred from memory, not read).

**Backing.** `ci.yml:9-11` states the least-privilege intent.

**Rebuttal.** The workflow triggers on `pull_request`, so a fork's token is read-only, and the step uses `env` for the body, so there is no injection path. SHA pinning is the stronger
posture and the pins were valid. The repository default token scope is not verifiable from the tree.

**Reproduction (verified, both votes).** Read both workflows, listed `.github`, ran `git ls-remote --tags` for both actions and compared the SHAs; one voter fetched `action.yml` at the
pinned SHAs and at v7.0.1 and grepped `using:`.

**Verdict.** CONFIRMED, 2 of 2.

**Disposition.** Fixed in edefefa (on main), fix-skeptic FIXED-VERIFIED: `permissions: {}` in `pr-issue-link.yml`; all ten checkout and setup-go pins in `ci.yml` moved to v7.0.1 (3d3c42e5) and v7.0.0
(b7ad1dad) by SHA; `.github/dependabot.yml` for `github-actions` (weekly); the `GOPRIVATE` line removed. No Go test can cover workflow configuration; the skeptic parsed the YAML and checked the SHAs
and the node24 runtime upstream. Residual: CI has not been run on the new action versions from this session (the skeptic could not execute Actions); static evidence says they should work.
`dependabot.yml` sits outside `.github/workflows/`, so the CODEOWNERS second-look rule for workflows does not cover it (the `*` rule does).

### A4-10: the release build's refusal had no automated test, and the gate omits govulncheck (low; CONFIRMED, 2 votes, 0 reproduced; fixed)

**Claim.** The refusal of a dirty or untagged tree, the property decision 88 names, had no automated test, and `hack/gate.sh` does not run govulncheck although CI does.

**Grounds.** `build_script_test.go` drove only `build/quill --dev`; no test ran the script without it. `hack/gate.sh` runs staticcheck v0.8.2 and deadcode v0.52.0 matching `ci.yml`, with no
govulncheck step, while `ci.yml` runs govulncheck@v1.9.0. By hand in scratch clones the auditor confirmed that an untagged HEAD, a modified tracked file and an untracked file each exit 1 with
a clear message and that an ignored file does not dirty the tree.

**Warrant.** The refusal works today, so this is regression risk only: a later edit to the status or describe lines could silently allow a dirty or untagged release. The script also accepts any
exact-match tag, not only `v0.y.z`.

**Backing.** Decision 88; decision 70 and the gate header ("CI runs the same checks as separate steps").

**Rebuttal.** CI and the gate catch most regressions; the refusal is a one-line condition run by a human; govulncheck needs the network and is arguably CI-only.

**Verdict.** CONFIRMED, 2 of 2 (both voters read the files; neither ran the refusal paths).

**Disposition.** Fixed in 43d0dee (on main), fix-skeptic FIXED-VERIFIED. `TestBuildScriptReleaseRefusal` runs the real script without `--dev` in a throwaway repository (untagged, tracked
modified, untracked, untagged and dirty, ignored file, clean tagged); the skeptic mutated the script four ways (status check disabled, `--always`, `--untracked-files=no`, `--ignored`) and each
mutation failed the matching subtest; it passes on macOS and on the Linux box. A comment in `hack/gate.sh` says govulncheck is CI-only: it needs a live database and can go red with no repository
change, which the gate's "pinned, not latest" principle forbids. Not done, as the claim called it optional: tag-format validation, so a malformed exact-match tag is still accepted. A wording tension
remains in the historical build prompt (`docs/design/build-prompt.md:21`, govulncheck "clean at every commit"), which the gate does not enforce.

## Ranked action list

Ranked by what the public flip needs, then by reach. Items 1 to 4 are owner actions or decisions; none is something the audit can do for the owner.

1. Add the `LICENSE`, and the Go licence notice in the archives and the repository (A4-5, decision 54); reword decisions 22 and 15; update the README Status (A4-6 residual).
2. Decide the history question before pushing anything (A4-4 residual): squash or filter so that neither the v0.1.0 tree nor earlier commits contain the workspace path, or accept it.
   Whatever is chosen, remember that the v0.1.0 tag also carries the old `SECURITY.md` and none of the A4 fixes, that main is 52 commits ahead, and that `build/quill` refuses an untagged HEAD, so the
   public release needs a new tag on a clean tree.
3. Enable private vulnerability reporting in the repository settings (A4-3); until then the fallback in `SECURITY.md` is what a reporter sees.
4. Done in this commit (A4-1): `build/quill` now pins the toolchain as the gate does. The CI half was refuted (the pinned setup-go reads the toolchain line), so the `ci.yml:65-67` comment stands, reworded; nothing is left to land.
5. At the flip, push only main and the chosen tag, never the `stage2/*` branches (they hold stray `-E` backup files); set branch protection to require the `ci` job and confirm the default token
   permissions (not verifiable from the tree).
6. Optional wording: state in README, CONTRIBUTING and `scope.md:94` that consumers need Go 1.27 or later because of the go line (A4-2); optionally an opt-in guard or checklist item for a missing
   `LICENSE` in `build/quill` (A4-6); optional tag-format validation in `build/quill` (A4-10).
7. Watch the first CI run on the new action pins (A4-9) and the first dependabot pull requests.
