# Model selection: what ran on quill, and what that does and does not show

Labels: [V] re-derived when this file was written from Claude Code's transcripts, the agent definitions and
`git log`; [R] reported by a source without independent re-derivation; [I] inferred. The counts are in
[token-usage.md](token-usage.md), which also says how they were derived.

---

## Answer

**Quill's build ran mostly on Fable 5.1, the top tier, and its audit and hardening ran on Sonnet 5.5, the
tier below Opus; Opus and Haiku never ran.** [V] The split was deliberate. The rule, in the owner's words
[R, the owner, 2026-11-05]: a reasoning model for the building, and Sonnet for directed single-task agent
work, the audit and the fixes. The one switch is `/model sonnet` at 2026-10-29 20:47:43, after v0.1.0 was built
and tagged [V]; the transcripts record the command, not the reason. The repository shows that this split
produced a released, audited library; it does not show that either choice was needed. [I]

There was a second, separate choice [R, the owner, 2026-11-05]: the audit was moved into **fresh sessions** for
a fresh perspective, so the Sonnet auditor did not have the build's context to lean on, only what a stranger to
the build would have: the code, the documents and a shell, the way the least-code ladder and the reasoning
harness instruct. The transcripts show the sessions are separate (the audit sessions have their own ids and
begin with their own first request).

The record fits the first rule loosely, not exactly. The skeptic and fixer agents are pinned to Sonnet, which is
the rule applied, and the audit sessions ran on Sonnet because of the second choice. But the design session's
own calls ran on Sonnet, and 83 untyped workflow agents in the build window ran on Fable while 42 ran on
Sonnet. [V] The first audit round, A1 round 1, also ran inside the build session, through its own skeptic and
fixer agents (44 and 13 of them, [V]), so the fresh-session separation starts after it, not before.

| Model | Used on this repo? | Where |
|---|---|---|
| Haiku 5.5 | **Never** | |
| Sonnet 5.5 | **Yes**, 96 % of the calls and 91 % of the tokens | The design session; 1,186 calls in the build window (skeptics, fixers and some workflow agents); all of the audit, hardening and release |
| Opus 5.5 | **Never** | |
| Fable 5.1 | **Yes**, 1,005 calls, 9.3 % of the tokens | The build session itself (292 calls) and 83 of its workflow agents (713 calls), through the build of v0.1.0 |

## What each model actually did here

[V], from the model field on every call in the five sessions (the counts and the method are in
[token-usage.md](token-usage.md)).

- **Sonnet drafted the design.** The design session wrote `review.md`, `scope.md`, `decisions.md` and
  `build-prompt.md`, then ran on after the first commit.
- **Fable built v0.1.0.** The first call of the build session is at 23:39:44 on 2026-10-28; the first commit
  is 4 minutes 41 seconds later; the code that became `v0.1.0` is 76 commits over 19 hours 44 minutes. In the
  build window the session made 292 calls itself and started agents that made 1,899 more: 713 on Fable, 1,186
  on Sonnet. That window also holds the first adversarial audit round and the allocation pass, which the tag
  message counts as 39 and 12 fixes [R, the tagger's counts, not re-derived].
- **The skeptics and fixers were Sonnet by definition.** The `audit-skeptic` and `audit-fixer` agent
  definitions each say `model: sonnet`, and in the build window 44 skeptic and 13 fixer agents ran on Sonnet
  while 83 untyped workflow agents ran on Fable. [V] Another 42 untyped workflow agents in the same window ran
  on Sonnet; which setting put them there was not determined. The definitions sit under `.claude/agents/`,
  which this repository does not commit, so they are not in the tree.
- **Sonnet ran everything after.** Three sessions, about 22,000 calls, every one of the subagents they started:
  the audit prompt, the waves, the hardening, the live and power-cut runs, the release rehearsal and the tag.

## Strengths and weaknesses, for this repo's kind of task

This repository is a durable log: recovery from a crash at any byte, a file format an operator or an attacker
can damage, and an adversarial audit suite designed to doubt its own findings before they become fixes. What
follows is limited to what that record shows about each choice.

| Choice | What the record shows | What it does not show |
|---|---|---|
| **Top tier for the build** | The log it built survived four real power cuts, three of them before v0.1.0 was tagged, and the on-disk format (`quill/1`) did not change through the whole audit. [R] [power-loss](../audits/reference/power-loss.md), the CHANGELOG's v0.2.0 upgrade notes | That a lower tier would have produced a worse log. No run of the build on another model exists |
| **Mid tier for the audit** | Sonnet made 23,287 calls, 22,162 of them in subagents, and 269 of the repository's 386 commits are typed `fix`. [V] The L2 mutation pass's surviving mutants were all missing tests, not wrong code; it raised one production defect (D1, `MaxReadBytes` bounded payload bytes only), which the owner approved and which was fixed. [R] [defects-log](../audits/passes/l2/defects-log.md) | That a higher tier would have found nothing more. Between v0.1.0 and v0.2.0, 304 commits changed 188 files outside `docs/` (22,215 lines added, 1,097 removed) [V], so what the top tier built was changed a great deal afterwards, mostly by audit passes [I]; no comparison says whether the same audit on another model would have found the same set |
| **Skeptics pinned to the mid tier** | Most candidates went to independent skeptics told to refute them and to default to refuted, and the records keep the REFUTED verdicts. [R] [audit README](../audits/README.md), [summary](../audits/summary.md) | Whether a stronger skeptic would have refuted a candidate the Sonnet ones confirmed, or confirmed one they refuted |

## What this doesn't establish

- **Not a benchmark.** No controlled comparison ran Opus, Haiku or a second model on the same tasks. That the
  split worked is **inferred** from the outcome, not tested.
- **Two changes at once.** From the audit sessions on, the model changed and the context changed together: a
  different model, in a session that had not seen the build. The record cannot separate how much of the audit's
  independence came from the one and how much from the other. [I]
- **Not separable from the process.** The build ran from a written build prompt in four phases with a stop
  after each, and the audit ran from a written audit prompt; a model's part in the result cannot be isolated
  from how much of the work those prompts had already decided. [I]
- **Not a rule.** The reusable part is the method of matching the work to the tier: open-ended reasoning, such
  as the build, where one session held the whole design, on the top tier; the many small, independently
  checkable tasks, such as an audit candidate or one fix, where each call is narrow and directed, on the tier
  below Opus. Whether that matching is right for another repository depends on its risk, and nothing here
  measures that.
