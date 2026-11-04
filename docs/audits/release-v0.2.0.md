# Release record: v0.2.0 rehearsal

A rehearsal of the v0.2.0 release, run end to end against commit `35d1185b92e357847fc2943420318d2786e77f07` (committed
2026-11-04T04:57:42Z) in a scratch clone with a local annotated tag `v0.2.0`. Nothing was pushed, no tag was created in
the real repository (its only tag is still `v0.1.0`) and no release was published. This document is the record of what
the rehearsal showed, the commands the real release will run, and what stands between the tree and a public repository.

Labels: **verified** means the rehearsal ran it or read the exact lines; **reported** names the document that says it;
**inferred** says from what. Line anchors are to the tree as of this commit.

## Top line

**The build, the gate and the reproducibility check are green; the tree is not releasable as it stands, for one reason
that a single commit fixes, and not publishable, for three reasons that are the owner's.**

- **Release blocker (a commit fixes it).** `CHANGELOG.md:6` is still `## Unreleased` and nothing mentions v0.2.0, so
  archives built from a v0.2.0 tag would ship a changelog with no v0.2.0 section. Retitle it in a commit before the real
  tag; the tag then sits on that commit and every hash below changes (the binaries embed the commit).
- **Public-flip blocker: history.** An internal name appears in a scratch-directory path in main's history (commit
  `98ff9e7`, an ancestor of main and of `v0.1.0`; removed at `0b70e82`). Pushing `main` or `v0.1.0` as they are would
  publish it.
- **Public-flip blocker: no `LICENSE`.** Decisions 22 and 54 make it an owner decision.
- **Public-flip blocker: scratch refs.** The local repository carries 484 scratch branches; only `main` and the release
  tag may be pushed.

## Pre-flight (verified)

| check | result |
|-------|--------|
| `FUZZTIME=20s hack/gate.sh` | exit 0, "OK: gate green" |
| gate steps | no dependencies; gofmt; vet for linux and darwin; freebsd refused; `go test -race` (8 packages ok; root 63.7 s, `cmd/quill` 32.2 s, `hack/topology` 22.9 s); staticcheck v0.8.2; deadcode v0.52.0; 15 fuzz targets at 20 s each; `go mod tidy` with no diff |
| govulncheck v1.9.0 (database dated 2026-11-02), Go 1.27.3 | "No vulnerabilities found" |
| tree state | `git status --porcelain` empty; `HEAD` equals `v0.2.0^{commit}` |
| toolchain | pinned by `go.mod` (`go 1.27`, `toolchain go1.27.3`) |

The gate does not run govulncheck (it reads a live database, so it can go red with no change in the repository;
`hack/gate.sh` header). It is a separate step in the release procedure below and must be run again at tag time.

### Refusals of the release build (verified, in a second clone so the gated tree stayed clean)

`build/quill` refuses, exits 1 and produces nothing:

- untagged `HEAD`: `build/quill: HEAD is not tagged; tag it (v0.y.z) or use --dev`;
- a tagged tree with `README.md` modified: `the tree is dirty; commit or stash, or use --dev`;
- an untracked stray file: the same refusal.

Minor (verified for `git describe` only; the build was not run): the script accepts any tag name, though its message says
`v0.y.z`. A tag `nightly-x` makes `git describe --tags --exact-match` return `nightly-x`. This is audit item A4r-3 in
[supply-chain.md](supply-chain.md), deliberately not changed. The release engineer must spell the tag `v0.2.0`.

## Semver rationale

Quill is `v0.y.z` for ever (decision 23; `README.md:114`): breaking changes bump the minor with a migration note, fixes
and additions bump the patch. The last tag is `v0.1.0`. **v0.2.0, a minor bump**, because the changes since `v0.1.0`
include source-level and behavioural breaks, not only fixes. A patch release would be wrong. The tree has 184 changed
files outside `docs/` since `v0.1.0` (`git diff --stat v0.1.0 HEAD`, verified).

The on-disk format is unchanged: it is `quill/1` and versioned separately in `quill.format` (reported:
`CHANGELOG.md` header and the "Upgrading and rolling back" paragraph in the user guide, section 9). So replacing the
binary needs no migration. This is reported by those documents, not re-derived here: the rehearsal ran no cross-version
test. The audit record [format-compatibility.md](format-compatibility.md) covers the format.

Breaking changes, as the `CHANGELOG.md` section "Unreleased" states them (reported: the line numbers are in that file):

| change | who is affected | `CHANGELOG.md` |
|--------|-----------------|----------------|
| `Problem` and `SegmentError` each gained an unexported field, so an **unkeyed composite literal** of either from another package no longer compiles | the only compile-time break; callers that build these types (tests, mostly) must use field names | line 9 |
| `Open` refuses a `MaxBufferBytes` below `FlushBytes` with `ErrConfig` | a `Config` that set `MaxBufferBytes` under `FlushBytes`, including one under the 64 KiB default `FlushBytes`; the default is unaffected | line 69 |
| `MaxReadBytes` bounds the size of a batch, each record counting its payload plus 48 bytes | a caller that read with limit 0 and expected the whole tail of a log of small records now gets a first batch and must call again from the next sequence; `Records`, `Follow` and the command line tool continue across batches | line 29 |
| `ReadWait` and `Follow` on a failed log return `ErrFailed` at the head instead of blocking until the context ends | a consumer loop that treated a block as "idle" now sees an error | line 70 |
| a writable `Open` writes `quill.tail` only when recovery moved the position; it can return with the marker publish pending and `Err()` set | a caller that took a nil error from `Open` to mean "the marker is published" should check `Err()` | line 12 |
| `quill roll`, `recover` and `prune` (with a retention or cap flag) at a directory that is not a log exit 1 and create nothing; they used to create a new empty log | scripts that relied on a mistyped `-dir` creating a log | line 25 |
| `quill archive` exits 3 when the source had lost records the archive never copied; it used to exit 0 | `quill archive && notify-ok` now reports the incomplete archive | line 67 |
| `quill cat`, `read` and `tail` with `-json` escape Unicode format and bidi controls in a `payload` as `\uXXXX` | consumers that parse the JSON (the value decodes to the same string) | line 21 |

Other entries (the remaining `Fixed` and `Changed` items, the documentation-only changes) are not breaking by that file's
own account. This record did not re-audit the classification (inferred: it rests on the file's own wording, and the A1
and A5 audit records that the entries cite). The release notes must carry the table above as the migration note.

## Commands that will be run

Run from a clean checkout of `main` on the release engineer's machine (macOS with bsdtar and Apple gzip: see
"Reproducibility"). Nothing here pushes. All `go` commands run with `GOWORK=off`.

```sh
# 1. The release commit: retitle the changelog section (the heading style of v0.1.0 is "## v0.1.0").
#    Edit CHANGELOG.md line 6 from "## Unreleased" to "## v0.2.0" and put the migration table above under it.
git add CHANGELOG.md
git commit -m "docs: changelog for v0.2.0"

# 2. Pre-flight on that commit.
FUZZTIME=20s hack/gate.sh
GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
test -z "$(git status --porcelain)"

# 3. The annotated tag, on the release commit. Annotated; never move it, recreate it.
git tag -a v0.2.0 -m "quill v0.2.0"
test "$(git rev-parse v0.2.0^{commit})" = "$(git rev-parse HEAD)"

# 4. The release build: refuses a dirty or untagged tree; wipes and fills build/install/.
build/quill
(cd build/install && shasum -a 256 -c SHA256SUMS)

# 5. Smoke each binary (see "Smoke test"), then stop. The release engineer's step ends here.
```

The owner's step, **not run by the rehearsal and not to be run until the public-flip checklist below is closed**. Push
the branch and the one tag by name; never `--all` or `--mirror` (the scratch branches must not leave the machine):

```sh
git push origin main
git push origin v0.2.0
awk '/^## v0.2.0/{f=1;next} /^## /{f=0} f' CHANGELOG.md > build/install/notes.md   # the section of this release
gh release create v0.2.0 --verify-tag --title "quill v0.2.0" --notes-file build/install/notes.md \
    build/install/*.tar.gz build/install/*.cdx.json build/install/SHA256SUMS
```

(Inferred: the push and `gh` lines were not run and their flags were not exercised, because the repository is private
and unpushed. The `build/quill` and tag steps were run in the rehearsal clone as shown, with the tag named
`v0.2.0` there.)

## Artifacts (verified, from the rehearsal build of `35d1185`)

`build/install/` holds 4 archives, the SBOM and `SHA256SUMS`. `shasum -a 256 -c SHA256SUMS` reports all 5 OK.

```
e6123b5f5862d74876c700eebf976a18302c6de36b51b6aca5bd0aff5d49a775  quill_v0.2.0_darwin_amd64.tar.gz
4eceb144e91d788c074a7902813b7fd3a9efa0f9d9a3451bae2cf827deb1fb5c  quill_v0.2.0_darwin_arm64.tar.gz
6e4acc030793e2d4084093cce9619a4d7d6c2d5163ec2f4d759c85af52d8878c  quill_v0.2.0_linux_amd64.tar.gz
4a3ed7381912c9563b2ca03e5b59029a1776b6eb2f05009aae2450131995ba22  quill_v0.2.0_linux_arm64.tar.gz
b89582e03501aee506f4b50002ea224c402ac69696ab9f3c570bf11022c5a867  quill_v0.2.0.cdx.json
```

**These are the rehearsal's hashes, for the tag on `35d1185`. The real tag will sit on the changelog commit, so every one
of them will differ.** Record the real `SHA256SUMS` in the release notes after step 4.

The binaries inside the archives (verified, SHA-256 of the extracted `quill`):

| platform | SHA-256 of the binary |
|----------|-----------------------|
| darwin/amd64 | `ac2b9e610a6d5632affd98934c70ff25c42abb1f68d5ef5ebd4c9cc4026083fa` |
| darwin/arm64 | `8835d7b978986666c0ae5f9f9c8fcb9829a4a711e49b575590960189762146a8` |
| linux/amd64 | `6edbbe79ea2dfa1153c133040b884f1b9d18e50b6e61fc09a30b872770ee3e0d` |
| linux/arm64 | `70a64f61a801edaa25005c129a4f4c6903f9b9d235cc72a972021ec8df0697f7` |

What is in them (verified):

- Each archive holds one directory with `CHANGELOG.md` (84,716 bytes), `README.md` (7,989 bytes), `SECURITY.md` (2,800 bytes)
  and `quill` (3.4 to 3.6 MB). **No `LICENSE`**: none exists in the repository, and the script copies one only if present
  (`build/quill` copies `LICENSE` under `[ -f LICENSE ]`), so adding one needs no script change.
- Entries are owned by 0/0 with no names, and every mtime is the commit time.
- The SBOM is valid JSON (parsed with a JSON loader), CycloneDX 1.5. Its root component is `quill v0.2.0` with
  `vcs.commit` `35d1185b...` and `go.version` `go1.27.3`; it has exactly one library component, `go (standard library)`
  `go1.27.3` (purl `pkg:golang/stdlib@1.27.3`), and one dependency edge. `go version -m` on a linux and a darwin binary
  lists no dependency lines, so the SBOM matches what reaches the binary. (Inferred, harmless: the SBOM's stdlib version
  string carries a `go` prefix and its purl does not.)
- The binaries record `GOAMD64=v1` or `GOARM64=v8.0`, `-trimpath=true`, `CGO_ENABLED=0` and toolchain `go1.27.3`.
- `build/quill` leaves the tree clean (`build/quill.version` and `build/install` are ignored by git).
- **`SHA256SUMS` is unsigned**: no `.sig` or `.asc` is produced. A checksum file served beside the archives proves
  integrity against download damage, not against a compromised release page.

## Reproducibility (verified)

**Reproducible.** A second clone at a different, deeper path, same tag, built from a different working directory with
`TZ=Asia/Tokyo` and `GOFLAGS` empty, gave byte-identical checksums for all four archives, the SBOM and `SHA256SUMS`. The
extracted binaries match the table above. No builder path is embedded in the binaries.

Limits, as the `build/quill` header says (reported, and the rehearsal agrees):

- Archive identity holds only for the same tar and gzip implementation. Both rehearsal builds used the same bsdtar 3.5.3
  and Apple gzip. A verifier on another system compares the extracted binaries, not the archives.
- **A cross-builder check** (verified): building the tag from a `git archive` on a Debian 13 machine (Go 1.27.3) with the
  script's flags gave the same four binary hashes as the same build on the macOS machine (`linux/amd64` `9cb6a1a3...`,
  `linux/arm64` `29fdfe9d...`, `darwin/arm64` `f18f94b4...`, `darwin/amd64` `ae0401fd...`). They differ from the release
  archives' binaries because a release builds inside a git checkout and `go build` then stamps `vcs.revision` and
  `vcs.time`. Building the clone with `GOFLAGS=-buildvcs=false` reproduces `9cb6a1a3...` for `linux/amd64`, so the vcs
  stamp is the only difference.
- **A third-party verifier must pass `-buildvcs=false`, or build inside a checkout of the tag.** The release notes should
  say so.

## Version stamp and smoke test (verified)

The documented way to read the version is the verb: `quill version` prints `quill v0.2.0` on every binary
(`docs/reference.md:26` lists the verb). `quill --version` is not supported: it exits 2 with
`flag provided but not defined: -version`. That is a command line polish item, not a defect against the docs.

The smoke script runs, per binary: append 100 records from standard input (`append` prints 100); `read` gives 100 lines,
`rec-1` to `rec-100`; `read -json -from 100` gives `rec-100`; `stat -json` shows 100 records; `check` exits 0 (HEALTHY);
`register -protect` then `commit -offset 51`; `consumers` shows offset 51, lag 50.

| platform | how it ran | result |
|----------|-----------|--------|
| darwin/arm64 | the release binary, natively on the build machine | SMOKE OK |
| linux/amd64 | a Debian 13 machine (kernel 6.12); the binary copied there, SHA-256 `6edbbe79...` equal to the archive's | SMOKE OK |
| linux/arm64 | an `alpine:3.20` container on arm64, no Go toolchain present, binary bind-mounted, SHA-256 `70a64f61...` equal to the archive's | SMOKE OK |
| darwin/amd64 | under Rosetta on the build machine | `quill version` only: prints `quill v0.2.0`; the smoke script was not run |

Observation (verified, not a failure): `quill check` on a log made by `quill append` prints
`UNBOUNDED: neither MaxBytes nor MinFreeBytes is set` and a `quill.policy ... no such file or directory` line while
exiting 0 HEALTHY. It reads as noise for a log that no library writer ever opened.

### Source install in a clean module (verified, with a substitute)

A throwaway module outside the clone required `github.com/netstar-labs/quill v0.2.0` with a `replace` to the clone
(`GOFLAGS=-mod=mod GOPROXY=off GOWORK=off GOTOOLCHAIN=local`), and ran a copy of `examples/log/main.go`: it printed the
orders read, the resume, and `caught up: offset 6, lag 0`, exit 0. `go.mod` stayed unchanged, there is no `go.sum`, and the
dependency list is quill's three internal packages plus the standard library. From the clone, `go run ./examples/log`
and `go run ./examples/topics` exit 0 with correct output; the pipeline example needs `-role` (writer 20, then reader 20)
and exits 0. This proves the module builds and imports cleanly from source. It does **not** prove the module-proxy path.

## Rollback

One line, valid while nothing has been pushed (verified for the tag in a scratch clone; `git describe --exact-match`
then fails):

```sh
git tag -d v0.2.0 && rm -rf build/install build/quill.version
```

and revert or reset the changelog commit that carries the release notes. **After a push, once any Go module proxy has
fetched `v0.2.0`, it cannot be recalled** (the proxy cache is permanent; inferred from how the proxy is documented to
behave, not run here). Then the answer is a new tag (say `v0.2.1`) plus a `retract v0.2.0` line in `go.mod`,
`git push origin :refs/tags/v0.2.0`, and deleting the release. The tag is annotated: recreate it, never move it.

## What was not tested, and why

- **`go install github.com/netstar-labs/quill/cmd/quill@v0.2.0` from the module proxy.** The repository is private and
  unpushed. With the `replace` removed, `go` fails with `reading github.com/netstar-labs/quill/go.mod at revision v0.2.0
  ... Repository not found` (verified). The host's `GOPRIVATE` made `go` go direct to the host despite `GOPROXY=off`; the
  probe was read-only. The README install line and the module path are therefore untested against a real remote.
- **Verification of published artifacts**: downloading the archives from a release page and checking `SHA256SUMS`
  against them, and checking that the release page serves what `build/install` held. There is no release page.
- **A build by a person other than the release engineer on a different archive toolchain** (GNU tar and gzip). Binary
  identity was shown across two builders; archive identity only on one toolchain.
- **Native darwin/amd64 hardware.** The binary was run under Rosetta for `quill version` only.
- **GitHub CI on the real repository** (the `ci` job, Dependabot, branch rulesets, private vulnerability reporting).
  None of them exist until the repository does; the gate is the local equivalent of CI's checks (`hack/gate.sh`
  header). The CI workflow's action pins were checked against the remote tags (see the checklist).
- **Signature verification**: there is no signature to verify.
- **Anything on a platform quill does not support** (Windows; FreeBSD is refused by the gate's vet step by design).
- **A fresh run of govulncheck after the database moves.** It reads a live database; run it again at tag time.

## Public-flip checklist

"Who decides" is the owner unless it says otherwise. States are as of this commit.

| item | state | who decides |
|------|-------|-------------|
| `LICENSE` (decisions 22, 54) | **OPEN.** No `LICENSE` file exists (verified: the tree and the archives). `decisions.md:30` and `:62`, `README.md:114` and `scope.md:499` say it lands before the repository goes public. The house default is Apache-2.0 with the company copyright line, to be confirmed. Once added, `build/quill` ships it with no change. | owner |
| Third-party notices | **CLEAN, one caveat.** Verified: `go.mod` has no `require` line and there is no `go.sum`; `go version -m` of the binaries lists no dependencies; no copyright, SPDX, "adapted from" or licence strings in any tracked `.go`, `.sh` or `go.mod` file; `testdata/golden` is generated by quill. No `THIRD-PARTY-NOTICES` is needed for code. Inferred, for the owner or counsel: the binaries statically link the Go runtime and standard library (BSD-3, which asks for its notice in binary redistributions), the SBOM lists the standard library, and the archives carry no Go licence text. Many projects ship without; decide knowingly (see A4-5 in [supply-chain.md](supply-chain.md)). | owner, or counsel |
| No internal names in tracked files at `HEAD` | **PASS** (verified): word-bounded and prefix searches for the internal product and repository names over `HEAD` find nothing. Items to decide, none a product name: `.github/CODEOWNERS` names a personal account handle and the organisation's repository path; `docs/design/audit-prompt.md` lines 39, 229, 241 and 280 still carry operator detail of the private test setup (a lock-file path in a home directory, a Linux machine, a remote runner), which `docs/audits/a1-diff-scoped.md:451` already reported; `docs/audits/` carries working-process text (agents, stage 2 owner decisions). | owner |
| No internal names in history | **FAIL** (verified). A history search finds an internal name in main's history: commit `98ff9e7` ("docs(audit): A1 adversarial audit findings") added a scratch path that names an internal product and a home directory, and `0b70e82` removed it. `98ff9e7` is an ancestor of `main` and of `v0.1.0`, so a normal push of either publishes it. Commit and tag messages are clean (verified: zero hits). Also in history: `faa8eaf` added the audit prompt with the operator detail above (still at `HEAD`); commit `da2d638` has the message "x" and a placeholder author; the other 360 commits carry one GitHub noreply identity (owner to confirm it is meant to be public). Decide: publish a fresh squashed or filtered history, or accept the leak. `v0.1.0` contains the same path, so it must not be pushed as it is either. See A4r-1 in [supply-chain.md](supply-chain.md). | owner |
| Scratch refs must not be pushed | **OPEN** (verified). The local repository has 484 branches (`stage2/*`, `failed/*`, `worktree-*`) and `git log --all` reaches them. Push `main` and the release tag by name, never `--all` or `--mirror` (`docs/audits/gap-audit.md:511` and `supply-chain.md` say the same). | owner |
| Figures from the reference implementation in `review.md` and `scope.md` | **MOSTLY DONE** (verified by reading; the A7-15 record in [stage2-owner-decisions.md](stage2-owner-decisions.md) lists what was removed). Residue: `review.md:74` keeps "minutes for 100+ GB of never-expiring log", which the record says the owner may delete. `review.md` still describes the reference's architecture in detail without a name (a network fan-in layer with a mutually authenticated transport, collector and shipper daemons, a key set); how much of the origin to describe is an owner call (A4r-5). `scope.md` has no figure taken from the reference. | owner |
| `SECURITY.md` contact | **OPEN.** `SECURITY.md` relies on GitHub private vulnerability reporting (Security tab) with an "open an issue asking for a private contact" fallback and no email address. It works only if the owner enables private reporting in the repository settings once the repository exists; it cannot be checked before then. `SECURITY.md:34-36` cites "Stage 2 audit, owner-approved 2026-11-03, A3-7", internal process wording; consider softening. | owner |
| Repository metadata (description, topics, homepage, default branch) | **PENDING**, not run. Set a description, topics (`go`, `append-only-log`, `wal`, `embedded`, `durable-log`), a homepage and the default branch `main` (the local default is `main`, verified). `README.md:93` and `go.mod` hard-code `github.com/netstar-labs/quill`: the public repository must live at exactly that path, or the module path, imports and examples change first. | owner |
| Branch protection and the `ci` aggregator check | **PENDING.** `.github/workflows/ci.yml` defines one `ci` job (`needs:` every other job in the file, `if: always()`, fails unless all results are success; verified). The ruleset must require the check named `ci`, and can only do so after it has run once on a push or pull request to `main`. `pr-issue-link.yml` makes every pull request body carry `Closes #N`: confirm that contributor friction is intended. Rulesets and private reporting do not exist yet. | owner |
| CI actions pinned | **PASS** (verified). All 10 `uses:` lines are full commit SHAs: `actions/checkout@3d3c42e5...` (v7.0.1) and `actions/setup-go@b7ad1dad...` (v7.0.0); `git ls-remote` against the actions' repositories confirms the SHAs are those tags. Dependabot is configured for GitHub Actions (weekly). Workflow permissions are `contents: read`; `pr-issue-link.yml` has `permissions: {}`. The tool pins in `ci.yml` match `hack/gate.sh` (staticcheck v0.8.2, deadcode v0.52.0), and govulncheck v1.9.0 is pinned in CI. | none |
| `CHANGELOG.md` ready for the tag | **FAIL** for tagging as the tree stands (verified): `CHANGELOG.md:6` is `## Unreleased`. Retitle it `## v0.2.0` in a commit before the real tag, with the migration table; the tag then sits on that commit and the rehearsal's hashes will differ. | release engineer |

## Failures and findings, ranked

1. **Release blocker.** `CHANGELOG.md:6` is `## Unreleased`; fix in a commit before tagging.
2. **Public-flip blocker.** The internal name in history at `98ff9e7` (ancestor of `main` and `v0.1.0`); filter or start
   fresh. Also `da2d638` and the 484 scratch branches that must not be pushed.
3. **Public-flip blocker.** No `LICENSE` (owner decision per decisions 22 and 54); no Go licence text in the archives.
4. **Minor.** `quill --version` exits 2 (only the `version` verb works); `build/quill` accepts any tag name;
   `docs/design/audit-prompt.md` still carries operator detail; `.github/CODEOWNERS` names a personal handle;
   `SHA256SUMS` is unsigned.
5. **Not testable.** The real `go install ...@v0.2.0` path, private vulnerability reporting, rulesets and repository
   metadata, until the repository exists.

Deviation from the planned order (verified): the gate ran in the main rehearsal clone, while the two build refusals were
proven in a separate scratch clone of the same tag, so the refusals did not dirty the gated tree. The clean build and
the reproducibility runs were then made in the clones the plan named.
