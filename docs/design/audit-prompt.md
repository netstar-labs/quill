# Quill — audit prompt (release procedure, stage 2)

*The prompt of record for stage 2 of the release procedure in [build-prompt.md](build-prompt.md) (decision 55): the
complete house audit sequence, run as a workflow by a session that did not build the code, ending at v0.2.0. It is
the stage 1 audit (A1) generalised to every pass. Read [scope.md](scope.md), [decisions.md](decisions.md) and
[format.md](../format.md) first; the earlier audit records are in [docs/audits/](../audits/).*

The aim is a **sanity check with fresh eyes** before the repository goes public: find what the build and its first
audit did not assert. A green gate certifies only what someone remembered to assert.

## 1. What runs, and in what order

The passes run in the house order (A1 with A2, A4, A3, A5, A6, L1, L2, L3 and L3b, A7, A1 diff-scoped with A4
refreshed, L4), grouped into six **waves**. A wave is one workflow run; the owner or the driving session reads its
result before the next starts, so later passes read the code the earlier fixes produced.

| Wave | Passes | Why here |
|---|---|---|
| W1 | **A1** (with A2's four dimensions), **A4** supply chain | The baseline. Fixes doc drift and dead code first, so every later pass reads true docs. A4 is cheap and mechanical. |
| W2 | **A3** threat model, **A5** silent degradation, **A6** format compatibility | Independent of each other (A5 needs A1's doc fixes). Audited and verified concurrently, then fixed one pass at a time. A6 lands the skew harness. |
| W3 | **L1** verification and optimisation, **L2** test, benchmark and hardening | L1 consumes the audit instead of re-deriving it; L2 attacks the test suite itself with mutation testing. L1 measures first, then L2 loads the machines. |
| W4 | **L3** (not applicable) and **L3b** live multi-process validation | quill talks to no third-party service, so L3 is recorded as not applicable and says why. quill is multi-process by design (a writer and readers), which is what L3b is for. |
| W5 | **A7** gap audit | After L1 to L3b, when the yardstick has the most commitments to check. |
| W6 | **A1** diff-scoped, **A4** refreshed, **L4** release rehearsal, a completeness critic | The pre-flight for the tag. |

**L3 is not run.** It exercises a program against real external systems. quill has none; its "live" surface is real
filesystems, real processes and real kills, which is L3b.

## 2. Ground rules (given to every agent)

quill is an embeddable append-only log for Go on the standard library alone. A log is a directory with one assigned writer process and any number of reader and consumer processes; CRC-framed segment files, a durable marker file quill.tail, retention by age and caps, a quill command line tool; Linux and macOS only. It is at v0.1.0 and is being validated, with fresh eyes, before a public release. The specification is docs/design/scope.md (the guarantees), docs/design/decisions.md (numbered decisions) and docs/format.md. The code is what is under audit: a passing test is not evidence that anything is right, and a doc is not evidence that the code does what it says.

Ground rules:
- Go standard library only. Never add a dependency. Always run Go with GOWORK=off (the parent directory has a go.work that breaks ./... patterns). The toolchain is pinned by go.mod; hack/gate.sh is the commit gate and shows the pinned staticcheck and deadcode versions.
- No text you write (docs, comments, commit messages, test names, reports that will be committed) may name any internal product, repository, host, address, user, path or file name; the owner keeps a private list of such names and the gate screens for it. The origin is called "the reference implementation", nothing more.
- Never push, never create or move a git tag, never touch any directory outside the repository and the scratch directory.
- Throwaway programs and reproductions go in the scratch area under <scratch>/<your-label>/, run against a COPY of the repo: mkdir -p <scratch>/<your-label>/src && git archive HEAD | tar -x -C <scratch>/<your-label>/src. Never edit the repository unless your task says you are a writer working in your own worktree.
- Scratch copies are made ONLY with git archive (as above), never with cp -R of the repository or of a worktree: a copied worktree carries a .git file that points back at the real repository, and a git commit, init or tag run in such a copy lands on the real one. Before any git command in a scratch directory, run git rev-parse --show-toplevel and confirm the path is inside the scratch area; if you need version control there, git init a directory that has no .git entry at all.
- A Linux x86-64 box (Debian 13, ext4, 4 vCPU) is available: the remote runner <unique-label> <git-ref> -- '<command>' unpacks git archive <ref> of the repo into a fresh directory on the box and runs the command there with Go on PATH and TMPDIR on the disk (not the tmpfs /tmp). It is shared with other agents: prefix anything whose timing matters with flock ~/.bench.lock, and never run more than one heavy job per agent. To test an uncommitted copy, commit it in a scratch clone and pass that clone's path in QUILL_SRC=<path>.
- Apple containers: the container CLI runs linux/arm64 containers (alpine:3.20 is cached). Cross-compile on the host (GOOS=linux GOARCH=arm64 go test -c / go build) and run the binary inside an alpine container with a bind mount; do not build inside the container.
- Label each claim in your report at the claim: verified (you ran it or read the exact lines), reported (a doc says it, name the doc) or inferred (say from what). Cite file:line from the tree as it is now.

**Held for the owner.** A finding whose fix would change the on-disk format, an exported API, documented behaviour,
a CLI flag, output or exit code, a numbered decision or a guarantee is verified like any other, but **not applied**:
it is written to `docs/audits/stage2-owner-decisions.md` with a recommended answer and the fix already written out.
Everything else that survives the skeptics is fixed, CONFIRMED or PLAUSIBLE (REFUTED findings are recorded, not fixed).

## 3. The machinery of a pass

Every audit pass has the same shape; only the lenses differ.

1. **Audit.** Several report-only auditors run concurrently, one per lens. Where the pass says *blind*, an auditor is
   told:

   > You are an independent auditor with fresh eyes. Do NOT read anything under docs/audits/, nor CHANGELOG.md, nor git log messages about earlier audits: those are earlier results, and the value of this pass is that you reach conclusions on your own. docs/design/ (the spec), the other docs and the code are yours to read. Report-only: you edit nothing in the repository.

   The rest read the earlier records, because their job needs them (a baseline to compare, a yardstick to check).
2. **The finding form.** Every candidate is written as an argument a skeptic can attack one field at a time:

   > Write every candidate as an argument a skeptic can attack one field at a time: claim, grounds (file:line, or the reproducing program and its output), warrant (why the grounds entail the claim), backing (the guarantee, decision, spec or rule that licenses the warrant), rebuttal (the specific condition under which the claim does NOT hold; mandatory: a HIGH-confidence candidate with no honest rebuttal is downgraded). Give a concrete failure scenario for anything about correctness or security (inputs and state, then wrong output, crash or loss). Set semantic=true when the proposed fix would change anything a user or a document relies on: the on-disk format, an exported API signature or documented behaviour, a CLI flag, output or exit code, a numbered decision or a guarantee. Also list the checks you ran that came out clean, so the reader knows what was covered; an unlisted area will be read as not looked at. If you bound your coverage (sampling, one platform), say so. Return few, real candidates, not many speculative ones.

3. **Triage.** One agent merges duplicates raised by different auditors (independent re-discovery is evidence) and
   compares each item with the round-1 record: *prior-fixed* (and checks that the fix is really in the tree),
   *prior-refuted*, *prior-deliberate* or *new*. Items round 1 already refuted or decided on purpose are not
   re-litigated; they are listed, and a fresh auditor re-raising one is noted.
4. **Verify.** Each new item gets independent skeptics who see only the claim and the code and are told to **refute,
   defaulting to refuted if uncertain**. The lenses depend on the kind of finding: *bug, security, compat, degradation*
   get REFUTE and REPRODUCE (a throwaway program on a copy, on both platforms where the claim is platform-sensitive;
   CONFIRMED requires a reproduction); *perf, simplify, dedup* get REFUTE and LOAD-BEARING (including a measurement
   for performance claims: before and after, six runs, ranges not means); *doc, gap, hygiene, supply-chain* get
   REFUTE and DOC; *test* gets REFUTE and TEST-GAP (apply the mutation the claim describes; if the suite already fails,
   it is refuted). A split vote goes to an arbiter that does not simply side with the majority. Verdicts: **CONFIRMED**,
   **PLAUSIBLE**, **REFUTED** (kept, with the reason).
5. **Fix, serially.** One finding at a time, in a fresh git worktree, on a branch named for the finding:
   regression test first and shown failing for the reason in the claim; then the smallest root-cause change, naming the
   least-code rung it stopped at (need, reuse, standard library, platform, one line, minimum that works); docs and the
   changelog kept true; gofmt, vet, `test -race`, staticcheck and deadcode clean; one commit. A **fix-skeptic** (a new
   agent, given the claim and the commit, told to refute the fix and default to "not fixed") then runs separate lenses:
   the test fails without the fix (or, for a tests-only change, fails under the mutation); root cause and not only the
   reproduction (a sibling input, the other platform, the CLI form); behaviour preserved (the full gate, golden bytes
   unchanged, the exported API diffed); no new gap (unwired code, an undocumented flag, a doc now false);
   reachable from the real path; the ladder rung is named; and, for correctness and security, an independent route
   that re-derives the property. Outcomes: **FIXED-VERIFIED**, **FIXED-UNPROVEN** (say which lens could not run) or
   **REJECTED**; a rejected fix is reworked once, then held, and never merged. An accepted commit is fast-forwarded onto
   `main` by an integrator that checks the build and vet. Fixes never run in parallel, so each one lands on the last.
6. **Gate.** After the pass's fixes, the whole gate runs on macOS and on Linux (section 4). Red means a repair
   agent bisects and repairs or reverts; red after that stops the run for the owner.
7. **Record.** The pass's findings land in `docs/audits/` in the house form, with every verdict, reproduction,
   disposition and fix-skeptic outcome, a coverage statement (what was checked and found clean, what was not covered,
   any auditor that failed to return) and no silent caps.

The volume stages (skeptics, fixers, recorders, gate runners, integrators) run on a cheaper model through
`.claude/agents`; the auditors run on the session model. A cached call's options are never changed on a resume.

## 4. The gate

On macOS and on Linux, on the same commit:

- `hack/gate.sh` with `FUZZTIME=20s`: gofmt, vet, build, vet for linux and darwin, the unsupported platform must not build, `test -race`,
  staticcheck, deadcode, the fuzz targets, `go mod tidy` with no diff, no `require` line;
- `govulncheck` (reachable findings fail);
- the examples run, `build/quill --dev` and its checksums;
- on Linux, the full-disk tests (T14) on a real 4 MiB tmpfs, which must **run, not skip**;
- a screen of tracked files for the names the project never uses (the origin is only ever "the reference
  implementation") and for hosts, addresses and personal paths.

## 5. Waves and lenses

### W1 — A1 with A2, then A4

Outputs: `a1-round2-findings.md`, `a1-round2-simplify.md`, `a1-round2-dedup.md`, `a1-round2-correctness.md`,
`a1-round2-docs.md`, `supply-chain.md`.

#### `A-simplify`

Dimension A, simpler pathways. Read every non-test .go file (root package, cmd/quill, internal/*, hack/*, examples/*). Find code more complex than necessary that has a behaviour-preserving simpler equivalent: convoluted control flow, redundant checks, needless intermediates or abstractions, hand-rolled logic a standard-library call replaces, over-broad error handling, options and knobs nobody reads. Apply the least-code ladder (need, reuse, stdlib, platform, one line, minimum that works). Per finding: file:line, current shape, simpler shape, equivalence argument, risk, lines saved. Skip pure style. Call out code that looks reducible but is load-bearing, and why.

#### `B-dedup`

Dimension B, duplication. Find recurring patterns factorable into one helper (repeated error handling, decode-and-validate, path building, directory listing, lock handling, frame reading, option validation, CLI flag handling), including between the library and cmd/quill and between tests and helpers. Per cluster: what it does, ALL file:line occurrences, the single helper (signature and home), lines saved, and any subtlety that makes merging unsafe (similar but not identical; two copies that feed unrelated formats must stay separate). Only 3+ occurrences or 2 substantial. Include a "considered and NOT recommended" list.

#### `C1-write-path`

Dimension C, correctness, write path. Files to read closely: log.go (Open and the creation protocol, Close), flush.go, append.go, state.go, lock_unix.go, disk.go, codec.go, errors.go, internal/fsx, internal/atomicfile, hack/powertest. Hunt real defects against the guarantees in docs/design/scope.md: lost or reordered acknowledged records, a durability claim the fsync order does not support (data fsync before the marker rename; directory fsync after create, rename, unlink), the failure classes of a flush (a failed write truncates back and keeps the records buffered: is that true on every error path, including a short write and ENOSPC?), rotation and Roll, MaxBatchBytes/MaxPayloadBytes/MaxBufferBytes accounting and integer overflow, sequence exhaustion, the creation protocol after a crash at each step, ErrFailed stickiness, Sync/group-commit semantics (an error means unknown, not lost), resource leaks (descriptors, goroutines, timers). Each finding needs a concrete failure path (state, then wrong outcome), not speculation. Do the optimisation half too: hot-path allocations and syscalls that a measurement would show.

#### `C2-recovery-retention`

Dimension C, correctness, recovery, retention and consumers. Files to read closely: load.go, holes.go, retention.go, archive.go, consumer.go, stat.go, state.go, internal/strictjson, plus quill.policy and consumer-file handling. Hunt real defects against docs/design/scope.md and decisions.md: parse-forward recovery at every kind of damage (torn tail, bad CRC mid-segment, zero-filled tail, a sealed segment damaged, an empty file, a segment with a wrong first sequence, overlapping segments, a hole), whether recovery can ever delete or truncate data it should keep, the decision that quill.tail ahead of the data fails Open, retention arithmetic (age, MaxBytes, MinFreeBytes, MaxRecords, protected consumers, ConsumerTTL, the clock guard when the clock steps back or forward, Dropped accounting), deletion ordering against crashes, the consumer offset file protocol (temp, fsync, rename; per-consumer lock; ErrWrongLog, ErrAhead), consumer-name validation and path traversal, Archive's copy and verification and its resumability. Concrete failure paths only.

#### `C3-reader-cli`

Dimension C, correctness, readers and the command line. Files to read closely: reader.go, read.go, iter.go, seek.go, internal/strictjson, cmd/quill/*.go. Hunt real defects: a lock-free Reader racing a live writer (a read past quill.tail, a torn frame at the tail, a segment sealed or deleted between listing and open, the marker stale or ahead), index caching, Follow and ReadWait wake-up and cancellation, GapError ranges, OffsetForTime and SeekToTime boundaries (equal timestamps across a segment boundary, non-monotonic input, empty log, negative or huge times), iterator buffer lifetime and release, integer overflow in offsets and limits; the CLI: flag parsing, number parsing (huge, negative, scientific notation, whitespace), exit codes against the documented ones, every verb that writes or deletes (append, roll, prune, recover, commit, register, unregister, archive) run against the wrong directory or a live writer, -json output validity for every byte value, terminal-escape handling. Concrete failure paths only.

#### `C4-concurrency-crash`

Dimension C, cross-cutting: concurrency and crash consistency, over the whole module. Read for: every field touched by more than one goroutine and what protects it (mutex coverage, atomics, channels), lock ordering between the log mutex, the flush path, retention, consumers and readers, Close racing everything (Append, Flush, Read, Follow, Sync, Prune, Archive, Consumer.Commit), check-then-act and time-of-check/time-of-use against the filesystem, goroutine and timer lifecycles (leaks after Close, a flusher that outlives its log), context cancellation, channel closes (double close, send on closed), and the crash model in internal/fsx (what a crash at each file operation leaves; does the production code ever rely on something the model does not allow, or something a real filesystem does not give: rename atomicity without directory fsync, size-extending writes versus data, directory entry persistence). Write throwaway programs under the race detector in your scratch copy to confirm anything you claim; GOWORK=off go test -race on the Linux box as well. Concrete failure paths only.

#### `D1-user-docs`

Dimension D, doc versus code drift, user-facing material. Cross-check every concrete claim in README.md, docs/userguide.md, docs/architecture.md, docs/introduction.md, docs/executive-summary.md, docs/reference.md (find out how it is generated and whether it is current: regenerate it in your scratch copy and diff), CONTRIBUTING.md, SECURITY.md, examples/README.md, every example's comments, every doc comment in the Go source (exported API especially), and every CLI usage string and flag help text in cmd/quill, against the code: signatures, defaults, limits, error names, exit codes, file names, command output formats, procedures ("run this command, then you see that": run them). Per drift: the doc file:line and exact claim, the actual behaviour file:line, severity (misleading or wrong, outdated, cosmetic), the correction, and which side is wrong. Report the clean checks too. Also run every example and every command in the user guide against a scratch directory.

#### `D2-design-docs`

Dimension D, doc versus code drift, the design record. Cross-check docs/design/scope.md (every guarantee, every table row: Config defaults and ceilings, the API table, error list, the CLI table), docs/design/decisions.md (every row that states a behaviour, number, limit or default), docs/format.md (byte layouts, field widths, file names, the version marker, quill.tail's JSON shape, the policy file, consumer files) and docs/design/build-prompt.md's "Out of scope" list against the code. For each guarantee in scope.md state the mechanism and the test (T-number) that is meant to falsify it and check that the test really asserts it (a test that runs the path but asserts little counts as not verifying). Report decisions the code contradicts, guarantees with no mechanism, and format.md statements the encoder or decoder does not match. Report the clean checks too.

#### `supply-chain` (not blind: may read the earlier records)

Dependency, supply-chain and release-hygiene audit (the house A4 pass), report-only. Do not read docs/audits/ conclusions first; form your own. Check and report each: (1) go.mod/go.sum: no require lines (decision 14), go mod tidy no diff, go list -deps ./... is standard library only, go mod verify; the go and toolchain lines against known toolchain advisories (govulncheck at golang.org/x/vuln/cmd/govulncheck@v1.9.0, run it, report reachability), and whether the toolchain pin in go.mod, hack/gate.sh, build/quill and .github/workflows/ci.yml agree. (2) Build/release: build/quill refuses a dirty or untagged tree (try both in a scratch clone), -trimpath and version stamping, reproducibility (build the same tag twice in two scratch clones, compare checksums of the archives' binaries; report what differs), the SBOM's accuracy (does it list exactly what reaches the binary; is the CycloneDX JSON valid), SHA256SUMS correctness, cgo off. (3) CI: every action pinned by commit SHA and that SHA matches the stated tag (verify with the GitHub API or git ls-remote if you can reach it), token permissions, the required-check aggregator is not skippable, tool pins in step with hack/gate.sh. (4) Public-release hygiene: git grep -i and git log --all --format=%B for the owner's private list of internal names across tracked files, commit messages, docs and test fixtures (the origin is only ever called "the reference implementation"); personal paths or hostnames or IPs in any tracked file; secrets; LICENSE and THIRD-PARTY-NOTICES state against decision 22 (the licence is deliberately absent until the owner decides; report the state, and say what a public flip needs); SECURITY.md contact details; .gitignore against what is tracked; file modes. Each as a finding only if it is wrong or at risk.


### W2 — A3, A5, A6

The three passes audit and verify concurrently against the post-W1 tree, then fix one at a time (A3, A6, A5). A5 and
A6 are the two passes that apply only where the repository has what they audit: stated guarantees, and encoded
formats. Outputs: `threat-model.md`, `silent-degradation.md` (the ledger is the artifact), `format-compatibility.md`,
and the harness `hack/skew.sh`.

#### `A3-filesystem-boundary`

Security review (the house A3 pass), boundary 1: the log directory is attacker-influenced input. Model first: assets (acknowledged records, the integrity and ordering of a log, other files on the host the process can write, the consumer offsets, availability of the writer), adversaries (a local user of another uid with write access to a shared directory; a process that can only read; a hostile producer whose record bytes land in a log; a corrupted or restored-from-backup directory), and trust boundaries. Then attack the boundary: symlinks and hard links in place of segment, marker, policy, format, lock or consumer files (O_NOFOLLOW/O_EXCL coverage on every open, including the CLI verbs and Archive's destination), FIFOs, devices and sockets in the directory (an open that blocks forever), sparse or enormous files (a declared frame or file size used for an allocation), hostile quill.tail / quill.policy / quill.format / consumer-file contents (every field out of range, duplicated keys, huge numbers, NUL, deep nesting; the strict JSON decoder's limits), consumer names and path traversal through every API and CLI argument that becomes a path, file modes and umask (what mode is each created file and directory; who can read records; who can replace a marker), TOCTOU between listing and opening, the directory lock (can another uid hold or poison it; flock semantics), and rename or unlink races an attacker can win. Reproduce every CRITICAL or HIGH with a throwaway program on a copy, on macOS and on the Linux box where the platform matters. Rank CRITICAL > HIGH > MEDIUM > LOW in severity; each finding carries an attack scenario (attacker position, steps, impact).

#### `A3-input-and-dos`

Security review (the house A3 pass), boundary 2: bytes and arguments. Record payloads are opaque but cross into the CLI output, iterators and the file format; frames come back from disk; the CLI takes arguments. Attack: decoder and recovery on adversarial segment bytes (length fields, sequence fields, CRC collisions that a chosen-plaintext producer can construct, frames that decode to a record inside another record's payload: can a producer craft a payload that, after a torn write or a recovery scan, is parsed as a frame?, i.e. frame injection through parse-forward recovery), allocation from untrusted lengths (Read, Scan, Reader, Archive, check), amplification (a tiny input causing large reads, a segment index, a goroutine, a descriptor per call), unbounded growth (Follow, caches, consumer handles, buffers while flushes fail, the segment index of a Reader over millions of segments), terminal escape and JSON injection through every printing path of cmd/quill (every byte value 0-255, invalid UTF-8, U+2028, bidi controls, ANSI, OSC 52), integer overflow and sign issues in offsets, limits, sizes and time, panics reachable from the public API or the CLI with chosen input (write a fuzz-style driver in your scratch copy and run it for several minutes on the Linux box), and destructive CLI verbs (prune, recover, unregister, archive destination) pointed at the wrong place. Reproduce every CRITICAL or HIGH with a throwaway program on a copy. Findings carry attack scenarios (attacker position, steps, impact).

#### `A3-process-boundary`

Security review (the house A3 pass), boundary 3: other processes. One assigned writer, many readers and consumers in other processes, possibly other uids, possibly on a filesystem that is not what quill assumes. Attack: a second writer (another process, another uid, the same process opening the dir twice, a different path to the same directory via symlink or bind mount: is the lock really exclusive?), lock behaviour on SIGKILL, fork and exec (descriptor inheritance by a child that keeps the flock held after the parent dies), stale or recycled state, readers racing retention (deletion of a segment under a reader; a recycled file name or inode), a reader trusting quill.tail ahead of the data, consumers: two processes with the same consumer name, offset-file replacement while held, ErrAhead/ErrWrongLog bypass, network or shared filesystems (NFS, SMB, virtiofs, FUSE: what does quill assume, what does it check, what happens when flock is a no-op or not coherent; does the documentation say so?), clock manipulation by another party (retention and the clock guard: can a hostile clock step delete a log's data early; can it prevent deletion forever), disk-full induced by another process, and denial of service by a reader against the writer (does any reader operation ever take the writer's lock or block it). Reproduce every CRITICAL or HIGH with a throwaway multi-process program on a copy (macOS and the Linux box). Findings carry attack scenarios.


##### A5, step 0: enumerate the guarantees

Read docs/design/scope.md, docs/design/decisions.md, README.md, docs/userguide.md, every flag's help text in cmd/quill, and the exported doc comments. Enumerate every GUARANTEE quill advertises or implies, a promise it keeps while running: durability, ordering, contiguity, bounded disk use, retention bounds, consumer protection, gap reporting, recovery, the reader's view being bounded by quill.tail, archive completeness, lock exclusivity, and so on. A flag, option or Config field whose help text makes a promise is a guarantee (this is the highest-yield source). For each: id (G1...), the promise in one sentence, where it is stated (file:line or section), and the mechanism that provides it. Aim for completeness over brevity; typically 25 to 45 rows.

The list is split into five groups and each group is one lens:

#### `A5-group-N` (not blind)

Silent-degradation audit (the house A5 pass), report-only. The question for every guarantee below: in what state does quill KEEP RUNNING, KEEP ANSWERING and have STOPPED providing this guarantee, and COULD AN OPERATOR TELL from what quill exposes (errors returned, Err(), Stat() fields, Check, Dropped, the CLI's output and exit codes, anything logged)? Not a crash audit: a crash is loud. Look for: a bounded buffer or counter that gives up while looking drained; a correct refusal that silently reverts to a slower or weaker path; a retention window a consumer falls out of while reporting healthy; a periodic job that stops (the flusher, the prune sweep, age rotation) with nothing recording its last success; a degraded-but-serving condition the write path deliberately ignores; a capability that never initialised and runs as a no-op; an exported accessor computing the answer that nothing reads (run deadcode -test and grep for fields named Dropped, Failed, Degraded, Truncated, Stale, Oversize that are written and never read); a signal that freezes at its last good value when the loop dies; a signal that reads 0 both for "never ran" and "just succeeded". Distinguish ABSENT from ZERO (zero while not running is a lie in the reassuring direction). Mark every state: OK (a signal exists and an operator could act on it), LOG-ONLY (discoverable after the fact, not alertable), NONE (indistinguishable from health). Reproduce each degraded state you claim by execution on a copy (a throwaway program that drives quill into the state and prints what an operator can see). Candidates (kind "degradation") are the LOG-ONLY and NONE states, with the signal that should exist (and say if adding it changes the public API: semantic=true). In "clean", return the COMPLETE ledger as rows "guarantee id | degraded state | signal | mark" for every state you examined, including the OK ones; that ledger is the deliverable.\n\nGUARANTEES:\n${chunk.map(g => 

#### `A6-inventory-and-pins` (not blind: may read the earlier records)

Format and version-skew audit (the house A6 pass). Report-only: do not edit the repository; if a golden is missing, that is a candidate (kind "compat") whose proposedFix is the golden test. Inventory EVERY artifact whose bytes outlive the process or cross a version boundary: the segment frame, the segment file naming and ordering, quill.tail (its JSON shape and canonical encoding: decision 105 has a fast path that must agree with the strict decoder), quill.format and the log ID, quill.policy, the consumer offset files and their locks, the archive destination's markers, the CLI's -json output (a format scripts depend on), exit codes, and any hash, checksum or ID derivation. For each: does it carry a version; what is safe to change (is an unknown version rejected loudly) and what is unsafe (a layout change inside an existing type decodes as a wrong value, accepted); the blast radius of a silent change; whether the deployment mode makes mixed versions normal (a writer upgraded while old readers keep running in other processes: yes, for quill that is the normal state). Then check the pins: is there a golden test for each artifact beside its encoder (golden_test.go, testdata/)? Are the goldens pasted bytes, not computed from the code under test? Mutate in a scratch copy (reorder two fields in the encoder, change a constant, change a width) and confirm each golden FAILS; record each mutation and result. Does the failure message say what the person has done and forbid the repair of re-recording? Name collisions: two functions with one name computing different things. Is the docs/format.md text identical to what the pinned bytes show (decode a golden by hand against the doc). Verdicts are kind "compat" candidates only where a pin is missing, weak, or a silent-change path exists.

#### `A6-skew-run` (not blind: may read the earlier records)

Format and version-skew audit, the real run. Build the quill command-line tool from tag v0.1.0 (git -C the repository archive v0.1.0 into your scratch area; GOWORK=off go build -o old/quill ./cmd/quill) and from HEAD (new/quill). Using ONLY the two binaries (and small throwaway Go programs against each version's library when the CLI cannot express it), run these skew experiments on macOS and on the Linux box, in BOTH directions (old writes, new reads/recovers/prunes/appends/archives, and new writes, old reads), with records of every size class (empty, 1 byte, around 4 KiB, around 1 MiB, one near the maximum payload), several segments, a consumer committed at the middle, a protected consumer, retention and caps configured, a log with a hole and one with a torn tail: (1) read every record back and compare byte for byte; (2) check on both; (3) open the log for writing with the other version and append, then read with both; (4) kill -9 a writer mid-append (loop until a torn tail occurs), then recover and append with the other version; (5) the consumer files and quill.policy written by one version, read and rewritten by the other; (6) archive from one, open the archive with the other; (7) an old Reader process following while a new writer is running, and the reverse. Also: does the new version refuse a marker from the future cleanly (ErrFutureVersion) and does the old version refuse a directory the new one made, if it should? Report each experiment with its exact commands and output. A divergence, a wrong-value-accepted, a crash, or an undocumented incompatibility is a candidate (kind "compat" or "bug") with the reproduction. The experiments that worked belong in "clean".


##### The skew harness

`hack/skew.sh <refA> <refB>` builds the CLI at two refs and runs the skew experiments both ways (records of every
size class, several segments, a consumer committed in the middle, byte-for-byte read-back, `check` on both, append
with the other version, a `kill -9` of a writer mid-append followed by recovery and append with the other version,
`quill.policy` and consumer files rewritten by the other version); it exits nonzero with the evidence on any
divergence. Goldens are produced by running the encoder once and pasting the bytes in, never hand-written, and each
is mutation-checked: a golden that has never failed has not been shown to work.

### W3 — L1, then L2

Outputs: `verification-optimization.md`, `test-ledger.md` (updated), `defects-log.md`, `harness-inventory.md`,
`mutation-hardening.md`, `untestable-without-x.md` (updated). The macOS benchmark lens and the optimisation lens
never run at the same time, and L2's mutation runs start after L1's measurements are finished, so no benchmark is
timed under load. A benchmark regresses when the median of six runs is more than 10% worse and the ranges do not
overlap (decision 70); an optimisation is kept only if the gain is outside run-to-run noise and nothing regresses.

#### `L1-comments-1`

Verification pass (the house L1 pass), part 1. For the root-package files log.go, flush.go, append.go, state.go, codec.go, errors.go, lock_unix.go, disk.go, doc.go and the internal/ packages: extract every comment, doc comment and inline note and check each claim against the code path it describes: trace one concrete input through the code by hand (or run it), verify claimed invariants are enforced where the comment says, claimed error cases are reachable from the public API, "never"/"always"/"must" statements hold on every path, units (bytes, seconds, records) and off-by-one statements are right, and that every exported identifier has a doc comment that states its contract. A stale or misleading comment is a finding (kind "doc"; the fix is to correct the comment, or the code if the comment states the spec). Report the number of comments checked per file and the clean ones.

#### `L1-comments-2`

Verification pass (the house L1 pass), part 2. Same method, for load.go, holes.go, retention.go, archive.go, consumer.go, stat.go, seek.go, iter.go and the hack/ programs: extract every comment, doc comment and inline note and check each claim against the code path it describes (trace a concrete input; run it where cheap); invariants enforced where claimed; error cases reachable; "never"/"always"/"must" hold on every path; units and boundary statements; every exported identifier documented with its contract. A stale or misleading comment is a finding (kind "doc"). Report the number of comments checked per file and the clean ones.

#### `L1-comments-3`

Verification pass (the house L1 pass), part 3. Same method, for read.go, reader.go, cmd/quill/*.go and examples/: every comment, doc comment, usage string and inline note checked against the code path it describes (trace a concrete input; run the CLI where cheap): invariants, reachable errors, "never"/"always"/"must" statements, units and boundaries, exported identifiers documented with their contract. A stale or misleading comment is a finding (kind "doc"). Report the number of comments checked per file and the clean ones.

#### `L1-dead-and-debt`

Verification pass (the house L1 pass), dead code and technical debt. Run go vet, staticcheck and deadcode -test as hack/gate.sh pins them, and then look beyond them: functions only tests call, test helpers nothing uses, branches that cannot be reached (an error return no path can produce; a check implied by an earlier check), goroutines whose result is never read, fields written and never read, constants and options never consulted, TODO/FIXME/XXX/HACK text, skipped tests (t.Skip: why, and is the reason still true), tests that assert nothing about the result, duplicated test fixtures, and flags parsed and not used. For each: whether it is safe to delete (check git log -S for why it exists, public API, examples, docs). Apply the least-code ladder: dead code did not need to exist. Report as kind "simplify" (or "test" for a vacuous test).

#### `L1-operational-mac` (not blind: may read the earlier records)

Operational validation on macOS (the house L1 pass phase 3), report-only: do not edit the repository; work in a scratch copy for anything that writes. In the repository run, and report the real outcome of each: (1) GOWORK=off go test -race -count=1 -timeout 15m -v ./... | tail and a total count of tests run, passed, skipped (list every skipped test with its skip reason: a test that always skips on this machine is a gap in verification); (2) coverage per package (go test -coverprofile) and the ten least covered non-trivial functions; (3) the benchmark suite: GOWORK=off go test -run '^$' -bench . -benchmem -count=6 ./... against docs/audits/benchmark-baselines.md, using the house rule for a regression (the median of six runs more than 10% worse and the ranges do not overlap); report every benchmark with its new median and range next to the recorded one, flag regressions and unexpected allocations; the machine may differ from the baseline machine, so say what that does to the comparison; (4) every fuzz target (hack/fuzz.sh lists them) for 3 minutes each, one at a time, and report corpus growth and any crasher (a crasher is a finding with its input saved in your scratch area; do not commit it); (5) the race detector on the full suite twice more (-count=3). Findings: a failing or flaky test (run it 50 times to see), a regression, a crasher, a skip that hides coverage. Kind "bug", "perf" or "test".

#### `L1-operational-linux` (not blind: may read the earlier records)

Operational validation on Linux (the house L1 pass phase 3), report-only: do not edit the repository. Using the remote runner (unique labels; the box is shared and has 4 vCPUs: run one heavy job at a time) at git ref HEAD: (1) go test -race -count=1 -timeout 20m -v ./... with a count of tests run, passed and skipped (list every skip and its reason; T14's full-disk tests need QUILL_FULLDISK_DIR=/mnt/quill-small after the scratch area/harness/remote-t14.sh: confirm they RUN); (2) -count=5 on the full suite to flush out flakiness, and any test that fails once is a finding with the failing output; (3) the benchmark suite under flock ~/.bench.lock with TMPDIR on the ext4 disk: go test -run '^$' -bench . -benchmem -count=6 ./... against docs/audits/benchmark-baselines.md with the house rule for a regression (median of six more than 10% worse and the ranges do not overlap); report new median and range next to the recorded one; (4) every fuzz target (hack/fuzz.sh lists them) for 3 minutes each, one at a time; (5) GOMAXPROCS=1 and GOMAXPROCS=8 runs of the suite under -race to expose scheduling assumptions. Findings: a flaky or failing test, a regression, a crasher, a hidden skip. Kind "bug", "perf" or "test".

#### `L1-optimisation` (not blind: may read the earlier records)

Optimisation candidates (the house L1 pass phase 4), report-only: do not edit the repository; work in a scratch copy. An allocation pass was already done (decisions 103-105); do not re-derive it: look for what a PROFILE shows now. On macOS, profile (go test -bench with -cpuprofile and -memprofile, pprof -top) the append path (buffered; flush per call), a flush with fsync, Read near the head and the tail, Scan, a Reader following a live writer (poll cost per idle tick), cold Open of a log with many segments, Stat and Check on a large log, and the CLI's tail and cat. Propose only changes whose gain is MEASURED in your scratch copy (before and after, -count=6, the ranges not the means; keep only a gain outside run-to-run noise on a named benchmark and no regression on the others) and that preserve the on-disk bytes and documented behaviour; syscalls count too (strace or dtruss on the idle Reader poll and the flush path: an extra stat, open or fsync is a finding). Fsync counts per flush are a durability matter: do not propose removing one. Kind "perf" with the measurements as grounds. Also report what looks slow but is bound by the platform (fsync), so nobody chases it.

#### `L2-coverage-ledger` (not blind: may read the earlier records)

Test coverage ledger (the house L2 pass phase 0), report-only. Read docs/audits/test-ledger.md and the T1-T27 definitions in docs/design/build-prompt.md, then produce your own ledger from the code: go test -coverprofile per package on macOS and Linux (the Linux numbers differ: platform files; report both), the function-level uncovered list (go tool cover -func), and for every uncovered or partially covered NON-TRIVIAL function or branch decide whether it is (a) an error path that nothing triggers, (b) a platform branch only the other OS reaches, (c) a path that is only reachable with an injectable seam that does not exist, (d) dead. For (a) and (c): what exact test or seam would reach it. Findings (kind "test") are uncovered paths that carry risk to a documented guarantee (durability ordering, recovery decisions, retention deletion, lock handling, consumer offset writes, the CLI's destructive verbs, error mapping of GapError), each with the exact test to write as proposedFix. Also list every test that is vacuous or only asserts "no error".

#### `L2-mutation-write` (not blind: may read the earlier records)

Mutation testing of the test suite (the house L2 pass; a precedent for the method: hand-written textual mutants, one per production change, killed or survived by the real test suite), part 1: the write path. Target files: log.go, flush.go, append.go, state.go, codec.go, lock_unix.go, internal/atomicfile/*.go, internal/fsx/*.go. Build a small runner in your scratch area (a script that copies the repo, applies ONE textual mutation to ONE production file, runs go build (a mutant that does not build is invalid: rewrite it, never count it), then GOWORK=off go test -count=1 -timeout 150s over the packages that exercise the file, and records KILLED, SURVIVED or TIMEOUT). Write about 100 mutants chosen by RISK, not at random: off-by-one in bounds and sizes, flipped or dropped checks, a dropped error return, a removed fsync / directory fsync / rename / truncate-back, swapped arguments, changed constants and widths, removed locks, changed ordering of the data write and the marker publish, wrong CRC coverage. Run them in parallel using copies (this machine, and the Linux box via the remote runner for up to 4 at a time under flock ~/.bench.lock is NOT needed for correctness runs; just be considerate). Judge every survivor by reading the code: EQUIVALENT (no observable difference: say why), UNOBSERVABLE by black-box tests, or a REAL GAP. Findings (kind "test") are the real gaps: one candidate per surviving mutant cluster with the mutant diff as grounds and the assertion that would kill it as proposedFix. Report the mutation score before any fix: detected / (detected + survived), with the equivalent ones excluded and the number you excluded.

#### `L2-mutation-recovery` (not blind: may read the earlier records)

Mutation testing of the test suite (the house L2 pass), part 2: recovery, retention and consumers. Target files: load.go, holes.go, retention.go, archive.go, consumer.go, stat.go, errors.go, internal/strictjson/*.go. Method: a small runner in your scratch area that copies the repo, applies ONE textual mutation to ONE production file, runs go build (an invalid mutant is rewritten, never counted), then GOWORK=off go test -count=1 -timeout 150s over the relevant packages and records KILLED, SURVIVED or TIMEOUT. About 100 mutants chosen by RISK: off-by-one and comparison flips in retention arithmetic and the cap checks, dropped or reordered steps in parse-forward recovery and hole detection, a changed decision about what recovery may truncate or delete, protected-consumer and TTL conditions, the clock guard, Dropped accounting, consumer offset write ordering, name validation, strict JSON limits. Run in parallel on copies (here and on the Linux box via the remote runner). Judge survivors by reading: EQUIVALENT, UNOBSERVABLE, or REAL GAP. Findings (kind "test") are the real gaps with the mutant diff as grounds and the killing assertion as proposedFix. Report the score before any fix (equivalent excluded, and how many).

#### `L2-mutation-read-cli` (not blind: may read the earlier records)

Mutation testing of the test suite (the house L2 pass), part 3: readers and the CLI. Target files: reader.go, read.go, iter.go, seek.go, cmd/quill/*.go. Method: a small runner in your scratch area that copies the repo, applies ONE textual mutation to ONE production file, runs go build (an invalid mutant is rewritten, never counted), then GOWORK=off go test -count=1 -timeout 150s over the relevant packages and records KILLED, SURVIVED or TIMEOUT. About 100 mutants chosen by RISK: the Reader's bound by quill.tail, gap detection and GapError ranges, index caching, Follow/ReadWait wake-up, seek boundaries, MaxReadBytes and limit handling, the CLI's number parsing, exit-code mapping, which verbs open for writing and with what Config, output escaping, -json. Run in parallel on copies (here and on the Linux box via the remote runner). Judge survivors by reading: EQUIVALENT, UNOBSERVABLE, or REAL GAP. Findings (kind "test") are the real gaps with the mutant diff as grounds and the killing assertion as proposedFix. Report the score before any fix.

#### `L2-invariants-and-fuzz` (not blind: may read the earlier records)

Hardening and fuzzing review (the house L2 pass phases 3 and 4), report-only apart from scratch work. (1) Fuzz inventory: every function that parses bytes or crosses a trust boundary (frame decode, tail decode and its fast path, policy, format marker, consumer file, strict JSON, CLI number/time parsing, segment scan, Archive's reader) must have a fuzz target with explicit invariants (never panic, never hang, never allocate unboundedly, round-trip where a codec exists), seeded from real data. List each surface, its target (or the lack of one), its invariants, and whether the seeds are real. For each missing or weak target propose it (kind "test"). Run a differential fuzz of the decodeTail fast path against the strict decoder for 5 minutes in your scratch copy (same verdict, same value, on every input) and report. (2) Resource-bound invariants: list each (the buffer cap while flushes fail, MaxBytes, MaxRecords, MaxReadBytes, MaxPayloadBytes, descriptor use per Reader and per Follow, goroutines after Close, the Reader's segment index) and for each try to violate it with a throwaway program under -race, then check an existing test asserts it. A violated bound is a finding (kind "bug"); an unasserted one is kind "test". (3) Isolation invariants: reads never take the writer's lock; the offline CLI verbs never rewrite quill.policy; a Reader never shows beyond quill.tail. Try to violate each.


### W4 — L3 (not applicable) and L3b

First the harness is built, reviewed and landed (below); then four lenses run it. The two lenses that use the Linux
box run one after the other, because one of them may crash the machine on purpose. Output: `live-integration.md`
(what the tier proves, the platform constraints, and explicitly what it does not reach) and an updated
`harness-inventory.md`. Every defect a live run finds is distilled into a hermetic regression test by the fix that
closes it: live finds it, hermetic locks it down.

The seven rules this tier follows (from the house L3b prompt): run above the minimum viable topology (several
readers at different offsets and several consumers, not one of each); assert what only the new state can satisfy
(absence, not only presence); establish the fault before claiming anything about it (count the kills and the observed
gaps); use the client an application would use; kill the process, not the machine, and restart it in place; a crash
and a full disk are different failures, test both; make every run's data unique.

##### The harness

Build hack/topology: a standard-library Go program (package main under hack/topology, plus a driver script hack/topology.sh if one is needed) that runs quill as a REAL MULTI-PROCESS deployment on one machine, the way quill is meant to be used: one writer process, several reader processes following the log from different offsets, consumer processes that commit offsets, an operator running CLI verbs, and a chaos controller. Roles (one binary, subcommands): writer (appends records whose payload is self-verifying: sequence, a per-run id, length and a checksum, sizes drawn from every size class; flushes and writes an fsynced acknowledgement file after each acknowledged batch, like hack/powertest), reader (follows via OpenReader and checks every record it sees: contiguity, payload integrity, never a sequence above the tail marker it has observed, GapError handling by resuming at To+1 and counting gaps, monotonic progress), consumer (a named consumer that commits and resumes after being killed: the committed offset never regresses, never exceeds what the reader saw, protection honoured), pruner (retention pressure: tiny segments, short retention, caps), operator (loops quill ls/stat/check/read/tail/consumers and online verbs concurrently), chaos (SIGKILL the writer at random moments and restart it in place; SIGKILL readers and consumers; start a second writer and require ErrLocked; fill the filesystem under test), and verify (after the run: every acknowledged record present, the log contiguous, Check healthy, no record corrupt, consumer offsets sane, every reader's observations consistent with the final log). Every run's data and keys carry a per-run id. Results as a JSON summary and a nonzero exit code on any violated invariant, with the violating evidence printed. Parameterised by directory (the filesystem under test), duration, the number of readers and consumers, segment size, retention and caps, and a seed; deterministic in what it checks, not in scheduling. The harness must be able to fail: include a self-test mode that injects a deliberate fault (a reader told to skip a record, a writer told to ack before flush) and confirm the verifier reports it.

#### `L3b-macos` (not blind: may read the earlier records)

Live multi-process validation (the house L3b pass) on THIS Mac (APFS; fsync is F_FULLFSYNC here). Use hack/topology (build it from HEAD) and drive it: the baseline topology (one writer, 3 readers at different offsets, 2 consumers, an operator loop), then every chaos scenario: SIGKILL the writer repeatedly (at least 200 kills across runs, with readers following across the restarts), SIGKILL readers and consumers mid-commit, a second writer (ErrLocked), retention racing lagging readers (tiny segments, short retention, MaxBytes), a protected consumer holding data back while MaxBytes overrides it, archive running while the writer rotates, Roll under load, many logs in one process, the CLI's online verbs under load. Also run each scenario with GOMAXPROCS=1. For each: what it asserted, what it observed, and the exact command to repeat it. A violated invariant, a hang, a crash, a leaked descriptor or a wrong answer is a candidate (kind "bug" or "degradation") with the reproduction; assert ABSENCE where the claim is absence (after a crash, readers must not see unacknowledged-and-lost data; after retention, readers get a GapError, never a silent skip) and establish that each fault really happened (count kills, count observed gaps) before claiming anything about recovery. The scenarios that ran clean belong in "clean" with their counts.

#### `L3b-linux-ext4` (not blind: may read the earlier records)

Live multi-process validation (the house L3b pass) on the Linux box (ext4, data=ordered, 4 vCPU), through the remote runner (unique labels; one heavy job at a time; flock ~/.bench.lock for anything timing-sensitive; TMPDIR is already on disk, and the log directory must be on the ext4 disk too, not /tmp, which is a tmpfs). Use hack/topology (build it from HEAD) and drive the full scenario set: the baseline topology (one writer, 3 readers at different offsets, 2 consumers, an operator loop); at least 500 SIGKILLs of the writer with readers following across restarts; reader/consumer kills mid-commit; a second writer (ErrLocked), including a second writer under a different uid (sudo -u nobody) on a directory the first created; retention racing lagging readers; a protected consumer versus MaxBytes; archive during rotation; the full-disk case with live readers (the 4 MiB tmpfs: the test box's helper mounts it at /mnt/quill-small: writer, readers and consumers all running when ENOSPC hits, then free space and confirm the writer recovers or fails as documented, readers stay consistent, acknowledged records survive); descriptor leaks (count /proc/<pid>/fd before and after 10000 reader open/close cycles and after Follow cancellations); goroutine counts after Close; and the real power-cut style test hack/powertest (docs/audits/power-loss.md describes it; do ONE cycle of 90 seconds with sysrq b only if the box comes back on its own: check that it is a VM you can recover first (uptime, ssh back within 3 minutes) and if it does not come back after 5 minutes stop and say so). Report as the macOS lens does. Candidates (kind "bug" or "degradation") need a reproduction.

#### `L3b-fs-matrix` (not blind: may read the earlier records)

Live multi-process validation (the house L3b pass): the filesystem matrix on the Linux box through the remote runner. The default ext4 run is another lens's; yours are the other filesystems and mount options a user will meet, built as loop-device images under ~/fs on the box with sudo (the box has sudo without a password; install the missing tools with apt if you can reach the network, e.g. xfsprogs, btrfs-progs, f2fs-tools, and say what you installed): xfs, btrfs, ext4 with data=writeback and with data=journal and with nobarrier-equivalent options if the kernel allows, tmpfs, overlayfs on top of ext4, and if dm-flakey or dm-log-writes is available (modprobe dm-flakey; dmsetup) use dm-flakey with drop_writes on a loop device to drop writes after the writer is killed, then fsck and verify (this is the closest software gets to a real power cut with a lying disk cache: describe exactly what you did). On each filesystem: build hack/topology from HEAD and run the baseline topology plus the writer-SIGKILL, retention-versus-readers and full-disk scenarios (small images give you real ENOSPC), then hack/powertest's verify logic wherever the crash leaves a log behind. Report per filesystem what passed and what did not, what the filesystem makes of fsync and rename ordering, and anything quill assumes that a filesystem here does not give. Tear down every mount, loop device and dm target you created. Candidates (kind "bug" or "degradation") need a reproduction.

#### `L3b-apple-container` (not blind: may read the earlier records)

Live multi-process validation (the house L3b pass) in Apple containers (linux/arm64 VMs; the container CLI; alpine:3.20 is cached). Cross-compile hack/topology and the quill CLI on the host for linux/arm64 (CGO_ENABLED=0), and run: (1) the baseline topology and the writer-SIGKILL scenario with the log on the container's own filesystem, as separate PROCESSES inside one container; (2) the same with the log directory a bind mount from the host (virtiofs): writer and readers in the container, an operator process on the macOS host reading the same directory live through the quill CLI (a lock-free reader on the other side of a VM boundary); (3) restart behaviour: restart the container (does the data survive, does recovery happen, does the lock release), and kill -9 the writer inside it; (4) two containers sharing one bind-mounted log directory: a writer in one, readers in the other, and then the ABUSE case: a second WRITER in the second container on the same shared directory (the VMs have separate kernels, so flock cannot be coherent between them): what does quill do, does it detect it, what does the documentation promise about shared and network filesystems, and is anything silently corrupted? Report exactly. (5) a clean-room install check: in a fresh container with no Go toolchain, run the release-style binary: quill --version, an append/read/check cycle. Every container and image you create is removed afterwards. Candidates (kind "bug", "degradation" or "doc" when the docs over- or under-promise about shared filesystems) need a reproduction.


### W5 — A7

The yardstick is written first, then three sweepers each take the places "done" hides gaps. A skeptic rules on every
gap; a gap's fix may be to withdraw the promise. Output: `gap-audit.md`.

##### Step 0: the yardstick

The build is finished, the tests are green and the gate passes. Collect what "complete" was supposed to mean and QUOTE it: every guarantee in docs/design/scope.md, every acceptance sentence and "must", the T1-T27 test definitions and the phase 1-4 deliverables in docs/design/build-prompt.md (including the release procedure stage 1 and the sentence about replacing every figure in review.md and scope.md that came from the private reference before the repository goes public), the README, docs/userguide.md and every flag's help text (a flag is a promise in disguise), decisions.md rows that promise a follow-up or a later action, TODO/FIXME/XXX/HACK/"not yet"/"later"/"for now"/"known issue" in source, tests, comments and commit messages, CHANGELOG.md claims, the repo's CI workflow, the build script, the examples. Trust nothing that says "done" or "resolved": re-derive from the code. Output a NUMBERED yardstick: one line per commitment with its source (file:line or section) and the one-sentence commitment. Typically 80 to 150 rows. This is the reference every later finding must point back to.

##### The sweepers

Gap audit (the house A7 pass), report-only. Sweep the places listed below against the numbered yardstick. A gap must name its source of truth (a yardstick row); "this feels incomplete" is not a finding. Absence is evidence only after a real search: state the search that came up empty (the grep, the call graph, the listing). Do not trust "done" anywhere. Use kind "gap" (promised, unbuilt, unwired, unverified, undocumented, unhanded), "hygiene" (release hygiene) or "test" (unverified). In "grounds" put the yardstick row (Yn) and the grounds including the empty search, and in "rebuttal" the condition under which it is not a gap (handled elsewhere, out of scope by the yardstick's own words, already tracked). Withdrawing the promise (deleting the claim) is a valid proposedFix, often cheaper and safer than building it; when you propose building something, name the least-code rung that allows it. No silent caps: say which rows and which parts of the repo you did not sweep.

#### `A7-promised-built-wired`

YOUR PLACES: Places 1 and 2: "promised, not built" and "built, not wired". For every yardstick row: is there an implementation; is it called; is the flag read; is the Config field consulted; is the metric/Stat field ever set; is the exported function reachable (run deadcode -test; grep every flag against its uses).

#### `A7-wired-verified-documented`

YOUR PLACES: Places 3 and 4: "wired, not verified" and "verified, not documented". For every yardstick row: is there a test that reaches the path AND asserts its result (error branches, the second platform, restart and recovery, empty and huge input, the concurrent case; a test that runs the code and asserts nothing about its result counts as not verified); and does every behaviour, flag, limit, format and failure message the code has appear, correctly, in the docs (examples that no longer run, a changelog that skipped a change).

#### `A7-operable-handed-closed`

YOUR PLACES: Places 5, 6 and 7: "documented, not operable" (no install path, no upgrade or migration note, no sizing guidance, no way to tell it is healthy, no runbook for the documented failure messages, no rollback), "operable, not handed on" (the consumers and neighbours who must be told: README version policy, the Go version policy, CI config, repo metadata, CHANGELOG, SECURITY.md, CODEOWNERS, issue templates), and "handed on, not closed" (deferred work with no recorded home, a "deliberately not done" with no reason, a round-1 audit finding marked fixed whose fix is not in the tree: verify at least 15 of those by reading the code, not the ledger). Include the release-hygiene items a green build never covers: LICENSE and notices state (decision 22 says the owner decides; report what is needed to flip), private or internal names in public text and in git history and commit messages, the figures in review.md and scope.md that came from the reference and the build prompt says must be replaced by quill's own measurements or softened before the repository is public: find every one and say whether it was replaced; repo metadata files.


### W6 — A1 diff-scoped, A4 refreshed, L4, the critic

Outputs: `a1-diff-scoped.md`, `supply-chain.md` (a refresh section), `release-v0.2.0.md`.

#### `A1d-code` (not blind: may read the earlier records)

Adversarial audit, diff-scoped (the pre-flight of the release step). Everything the stage 2 audit changed is the diff v0.1.0..HEAD (git -C the repository diff v0.1.0..HEAD --stat, then read the whole diff of non-test code). Review that diff as a hostile reviewer: does any change introduce a defect, break a guarantee, alter the on-disk bytes (golden tests untouched?), change an exported API (go doc -all before and after: diff it) or a CLI contract without the CHANGELOG saying so, leave an inconsistent state between two changes that were each fine alone, or fix a symptom while the root cause remains? Interactions matter most: two fixes touching the same function or the same lock. Run the race detector on the interacting packages ten times. Report kind "bug"/"compat"/"simplify".

#### `A1d-tests` (not blind: may read the earlier records)

Adversarial audit, diff-scoped. Every test added or changed in v0.1.0..HEAD (git -C the repository diff v0.1.0..HEAD -- '*_test.go' and the hack/ harnesses): is it meaningful? For each new test, in a scratch copy revert the production change it was written for and confirm the test fails; a test that passes without its fix is vacuous (kind "test"). Look for tests that depend on timing (run each 50 times under -race on the Linux box, and with GOMAXPROCS=1), on the machine (path lengths, filesystem, uid, tmpfs), on ordering, or that leave files or processes behind. A flaky test is a finding with the failure output.

#### `A1d-docs` (not blind: may read the earlier records)

Adversarial audit, diff-scoped. Check that the documents match the diff v0.1.0..HEAD: CHANGELOG.md's Unreleased section lists every user-visible change in that diff, accurately and nothing that is not in it; every decision amended or added in decisions.md matches the code; docs/audits/ ledgers match the tree (every "fixed in <sha>" is in the history and the fix is in the tree; counts add up; no ledger names an internal product or host or IP address or personal path); README and the user guide still describe the code after the stage 2 changes; docs/reference.md is regenerated if the API changed. Kind "doc" or "hygiene".


##### A4 refreshed

The A4 lens above runs again, recording only what changed since the first pass (an advisory published since, a
toolchain change).

##### L4: the release rehearsal

The release notes are prepared and reviewed first (`CHANGELOG.md`: Unreleased becomes v0.2.0, grouped by impact, every
line traceable to a commit, a minor bump under the v0 policy, the exported API diffed against v0.1.0). Then, in a
scratch clone and never in the repository: the annotated tag is created in the clone; the full gate and `govulncheck`
run on the tagged tree; `build/quill` is shown to refuse a dirty and an untagged tree and then builds the four
archives; the checksums, the SBOM and the archives' contents are verified; the build is repeated at another path to
test reproducibility; each runnable binary prints its version and passes an append, read, check, stat and consumer
smoke on macOS, on the Linux box and in an Apple container with no Go toolchain; a source install from a throwaway
module is tried (the real `go install` from the module proxy is not possible while the repository is private, and
the record says so); a one-line rollback is written; and the public-flip checklist is recorded with its current state
(LICENSE per decision 22, notices, internal names in files and in history, the reference's figures in review.md and
scope.md, SECURITY.md, repository metadata, branch protection, pinned CI). Nothing is pushed and no repository setting
is changed.

##### The completeness critic

A last agent reads every record and asks what is missing: a modality not run, a claim that no command backs, a source
file no pass examined, a platform or filesystem not exercised, a record that disagrees with the tree or with another
record, a number that does not add up; and gives a verdict on whether the build is ready for the owner to review for a
public release, and on what condition.

## 6. Harnesses

| Harness | In the repository | What it is for |
|---|---|---|
| `hack/gate.sh`, `hack/fuzz.sh` | yes | the commit gate and the fuzz budget |
| `hack/powertest` | yes | a writer acknowledged-batches test for a machine that can be crashed without syncing |
| `hack/skew.sh` | yes, built in W2 | a cross-version skew check between any two refs |
| `hack/topology` | yes, built in W4 | the multi-process live harness: writer, readers, consumers, operator, chaos, verify; with a self-test that injects faults to prove it can fail |
| remote runner | **no** | a script outside the repository that unpacks `git archive <ref>` on the Linux test box and runs a command there; hosts, addresses and users never enter the repository |
| mutation runner | no | a throwaway script per lens that applies one textual mutant to a copy and runs the suite |

Committed harness code is standard library only, names no host, address, user or internal product (hosts and paths
come from flags or the environment), passes the gate, and is reviewed by a separate agent that must see it **fail**
when a fault is introduced: a harness that cannot fail proves nothing.

## 7. What this does not do

- It does not push, create a remote, publish a release or change a repository setting.
- It does not create the `v0.2.0` tag. The rehearsal tags a scratch clone; the owner tags `main` once the records
  are reviewed and the final gate is green.
- It does not add a `LICENSE` (decision 22 leaves that to the owner) or a dependency.
- It does not apply a held finding.
- It does not prove behaviour on a power cut with a lying disk cache; the filesystem lens narrows that gap with a
  dropped-writes device where the box allows, and the rest stays in `untestable-without-x.md`.

## 8. What to hand back

The findings by verdict for each pass, what was fixed (with its fix-skeptic outcome and commit), what was held and
the recommended answer for each, the final gate on both platforms, the release rehearsal result and the public-flip
checklist, the critic's list of what is missing, and anything in this prompt that proved wrong.
