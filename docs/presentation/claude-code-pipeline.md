# How I Use Claude Code
*A repeatable, audited engineering pipeline — not a chat window. The case study: quill.*

Prepared by zxdev · Go engineering across NetSTAR Labs · Living practice, in daily use

---

## 1 · Five Things, One Pipeline
*Overview*

1. The reasoning harness — one standing operating manual, every session
2. The least-code agent — climb the ladder before a line gets written
3. The design ladder — review → scope → decisions → build prompt, before code
4. Build, harden, ship — four build phases, then the A1–A7 and L1–L5 passes
5. **In practice** — one week, from an empty directory to a public release *(case study)*

> Figures below come from this repository's own git history and audit records, and from the Claude Code
> transcripts of the five sessions that built it. How each number was derived is in
> [token-usage.md](token-usage.md) and [loc-count.md](loc-count.md).

---

## 2 · The Reasoning Harness
*1 · Standing operating manual*

A **harness** is a standing operating manual, loaded into context before any work starts — it changes how a
conclusion is reached, not what the task is. Each of its eight stages is named for one specific, recurring
way fluent output goes wrong, built backward from the mistake toward the check that would have caught it.

1. Read the request underneath the request
2. Decompose at verification boundaries
3. Find where the risk actually lives
4. **Verify by re-derivation, not recognition** — *run it, don't recall it*
5. Label known versus guessed, at the claim
6. Attack your conclusion before it ships
7. Answer, then reasoning, then risk
8. The mistakes that look like competence

> **In practice:** recovery code is the kind that reads as correct and is not. So it was not trusted on
> reading. A writer was crashed on a real machine four times with `echo b > /proc/sysrq-trigger`, and each
> time every acknowledged record came back, in order, with no holes. One run lost a marker rename and the 27
> records behind it; recovery found them again by parsing forward (decision 39). *Run, not recalled.*

Source: `handbook/prompts/claude-code-reasoning.md`, byte-identical to this repo's `.claude/CLAUDE.md`
(verified by `cmp`); `docs/audits/reference/power-loss.md`.

---

## 3 · The Least-Code Agent
*2 · A standing discipline*

Climb from the top, stop at the first rung that answers. No rung is passed silently — a change names the rung
it stopped at. Quill's audit prompt writes the ladder as: need, reuse, standard library, platform, one line,
the minimum that works.

| Rung | What it means | Where it showed up in quill |
|---|---|---|
| **Reuse** | Already in a codebase this account owns? | The frame bytes are the reference implementation's, unchanged (decision 3); everything else built fresh (decision 2). |
| **Standard library** | The standard library does it? | No dependencies: `go.mod` has no `require` line, and the gate checks that. |
| **Platform** | The OS or filesystem already guarantees it. | The single-writer lock is `flock`; a reader's view rests on `rename` and `fsync`. No broker, no sockets. |
| **One line, minimum** | One condition, not a new type; no extension points. | The gap audit records that every fix stopped at the lowest rung that closes it: one line, one pure move, one table or one test. |

> **A dependency is never a silent `go.mod` edit.** It is a labeled, costed proposal. Quill's answer was none.

Source: `handbook/agents/least-code.md`; `docs/design/audit-prompt.md` (each simplification names its rung);
`docs/audits/passes/gap-audit.md`.

---

## 4 · Prompt, Skill, Agent, Workflow
*2 · The vocabulary*

Four related words that mean four different things — what runs, how it is triggered, and who decides to run
it.

- **Prompt** — a markdown file, run once by request: the house A1–A7, L1–L4 and D0–D3 templates, and quill's
  own `build-prompt.md` and `audit-prompt.md`. Instructions for one pass, not a standing behavior.
- **Skill** — a prompt packaged to trigger itself, or called by name as `/name`.
- **Agent** — one delegated subagent for a single independent task. Quill defines two:
  `audit-skeptic` (tries to refute one finding) and `audit-fixer` (applies one finding in an isolated
  worktree). Both are pinned to Sonnet.
- **Workflow** — a scripted pipeline of many agents: parallel fan-out, phases, verification stages. Runs only
  with the owner's explicit opt-in.

> Across the five sessions that built quill: **25 workflow runs** and **1,839 subagent transcripts**. Agents
> made 94 % of the model calls.

Source: `handbook/prompts/README.md`; the session transcripts (counted in [token-usage.md](token-usage.md)).

---

## 5 · The Design Ladder
*3 · Before any code*

Quill's design starts from a review of an existing implementation, the reference. It was written down and
accepted by the owner on the day the build began, 2026-10-28.

1. **Review** (`review.md`) — what the reference design does, and what is left behind.
2. **Scope** (`scope.md`) — what is built, and the guarantees.
3. **Decisions** (`decisions.md`) — 105 rows, each a choice with its reason, **each accepted by the owner**.
4. **Build prompt** (`build-prompt.md`) — how the build runs: the layout, the format, the tests it must
   have, and four phases with a stop after each.

> Decision 2 is the one that set the shape: **build fresh from this design** instead of deleting code from the
> reference.

Source: `docs/design/`. The design session wrote and edited all four (18 write and edit calls in its
transcript).

---

## 6 · Build, Harden, Ship
*4 · After the design*

The build prompt runs in four phases and **stops after each**: do not start the next until the owner says so.

1. **Format, segments and the seam** — the file-system seam with a crash simulator, the frame, the state files,
   recovery of one segment, the golden fixtures, and `docs/format.md` (the fixtures freeze it).
2. **The log** — `Open` and its creation protocol, the lock, recovery with hole detection, append, flush,
   rotation, `Sync`.
3. **Readers, consumers and retention** — the lock-free reader, consumers, protection, the caps, `Archive`.
4. **CLI, docs and release readiness** — the command, the examples, the guides, CI, the release script, and a
   hostile self-review of recovery and the reader before any audit.

> **In practice:** the code that became `v0.1.0` is **76 commits over 19 hours 44 minutes**, from the first
> commit to the commit the tag named (`9364fb0`). The session had been open 4 minutes 41 seconds when the
> first commit landed.

Source: `docs/design/build-prompt.md` ("Phased execution"); `git log`.

---

## 7 · The Audit Suite
*4 · After the build, and again before release*

Report-only review passes. Each candidate finding goes to independent skeptics told to refute it and to
default to refuted; verdicts are **CONFIRMED, PLAUSIBLE or REFUTED**, never just "found". Fixes land one at a
time, each attacked again by a fix-skeptic.

| Pass | What it examined |
|---|---|
| **A1** adversarial audit (rounds 1 and 2, then **A1d** on the diff) | Four lenses: simplify, dedup, correctness, docs |
| **A3** security | The threat model |
| **A4 / A4r** supply chain | Dependencies, history, release hygiene |
| **A5** silent degradation | Where the log can get worse without saying so |
| **A6** format and skew | The on-disk format against version skew |
| **A7** gap audit | What the other passes could not see |
| **L1 / L2** verification, tests | Dead code, debt, mutation testing and its defects log |
| **L3 / L3b** live | Real multi-process runs, a Linux filesystem matrix, real power cuts |
| **L4 / L4b / L5** release | The rehearsal, then the pre-tag documentation and code batches |

> **377 numbered candidates** across the passes. A candidate is a claim, not a confirmed defect; the status of
> each is in its record. The batches **L4b and L5 carry no skeptic votes**: the owner approved their items
> directly, and the records say so.

Source: `docs/audits/README.md`, `docs/audits/summary.md`.

---

## 8 · What the Audits Found, and What They Didn't
*5 · In practice, part one*

- **The build was changed a great deal after v0.1.0.** Between the tags, **304 commits** changed **188 files
  outside `docs/`** (22,215 lines added, 1,097 removed). 269 of the repository's 386 commits are typed `fix`.
- **Mutation testing found missing tests, not wrong code.** Hand-mutating the write path moved the score from
  80.1 % to 96.3 %, and recovery, retention and consumers from 60.1 % to 94.5 %. Every surviving mutant was a
  test that did not exist. The pass raised one production defect: `MaxReadBytes` bounded payload bytes only.
- **Real machines, not only simulators.** Four power cuts on a real Linux host, four clean recoveries, one with
  44,484,329 acknowledged records. A nine-filesystem Linux matrix (ext4 in five configurations, xfs, btrfs,
  f2fs, tmpfs); journal-less ext4 is outside the supported contract, and the records say so.
- **The record says what is still open.** At the tag, three findings (A3-7, A3-24, A3-26) have no fix written.

> The format `quill/1` did not change through any of it. **What the audits could not show** is written down
> too: `docs/audits/reference/untestable-without-x.md`.

Source: `git diff --shortstat v0.1.0 v0.2.0 -- . ':!docs'`; `docs/audits/passes/l2/`;
`docs/audits/reference/power-loss.md`; `docs/audits/summary.md`.

---

## 9 · One Week to v0.2.0
*5 · In practice, part two*

Different sessions, different models, one repository. The build ran mostly on Fable; the design and the audit
ran on Sonnet.

- **7 days 3 hours** — first commit to the tagged v0.2.0
- **386** — commits on `main`
- **24,163** — model calls to the tag; **17.7 million** output tokens; **1.85 billion** tokens in all (94 %
  cache reads)

> The split is deliberate: the reasoning model for the reasoning (the build ran on Fable 5.1), Sonnet 5.5 for
> directed single-task agent work (the audit skeptics and fixers). After v0.1.0 the owner switched the
> session to Sonnet with one command, and every audit agent from then on ran on Sonnet. The audit also moved
> to **fresh sessions**, so the reviewer had no build context to lean on, only the code, the documents and a
> shell. Opus and Haiku were never used. Public on
> 2026-11-05, with a release that carries archives for four platforms, checksums and an SBOM.

Source: `git log`; [token-usage.md](token-usage.md), [model-selection.md](model-selection.md); `gh release view v0.2.0`.

---

## 10 · Why This Is the Standard
*Summary*

1. One reasoning harness, loaded before every session — not re-invented per repo
2. The ladder runs before the code — a dependency is a named, costed decision, and here there are none
3. Design is written down and **decided by the owner** before it is built
4. **Layered review kept finding things after the first review had finished** *(the point)*

| What the pipeline cost | What it bought |
|---|---|
| Seven days, 1.85 billion tokens as logged, 17.7 million of them output | A released, public v0.2.0, with every pass's record and its open items published |
| An owner decision on every held item | A format unchanged through 304 commits, and four real power cuts survived |
| Passes that carry no skeptic, and say so | A record that separates what was verified, reported and inferred |
