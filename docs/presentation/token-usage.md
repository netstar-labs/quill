# Quill — model and token usage, first commit through v0.2.0

Labels: [V] re-derived when this file was written from the Claude Code transcripts and `git log`; [R]
reported by one source without re-derivation; [I] inferred. All times are UTC.

## Answer

Quill was built across five Claude Code sessions. **The v0.1.0 build ran mostly on Fable 5.1
(`claude-fable-5-1`); everything after the model switch on 2026-10-29 at 20:47 ran on Sonnet 5.5
(`claude-sonnet-5-5`), and Opus and Haiku never ran.** [V] Through 2026-11-05 12:45 the five sessions logged
**24,292 model calls, 17.8 million output tokens and 1.88 billion tokens in all**, 94 % of them cache reads.

| Phase | Model | Session | Calls | Output tokens | Total tokens |
|---|---|---|--:|--:|--:|
| A · Design (the quill part of the design session) | Sonnet 5.5 | `cf41ba29` | 135 | 468,981 | 47,506,167 |
| B · Build to v0.1.0, the session itself | **Fable 5.1** | `cb9b1313` | 292 | 578,311 | 131,308,216 |
| B · Build to v0.1.0, its Fable subagents | **Fable 5.1** | `cb9b1313` | 713 | 844,125 | 42,963,391 |
| B · Build to v0.1.0, its Sonnet subagents | Sonnet 5.5 | `cb9b1313` | 1,186 | 701,261 | 46,086,891 |
| B · the same session after the switch | Sonnet 5.5 | `cb9b1313` | 84 | 63,574 | 10,373,147 |
| C · Write the audit prompt, run the first waves | Sonnet 5.5 | `4849b262` | 3,390 | 2,484,393 | 214,655,302 |
| C · The audit run, resumed after a restart | Sonnet 5.5 | `95f7a480` | 1,654 | 1,280,764 | 125,629,048 |
| C · The waves, hardening and release to the v0.2.0 tag | Sonnet 5.5 | `70677139` | 16,709 | 11,297,578 | 1,228,872,587 |
| D · After the tag: housekeeping and the public flip | Sonnet 5.5 | `70677139` | 129 | 117,600 | 27,719,120 |
| **Total** | | | **24,292** | **17,836,587** | **1,875,113,869** |

By model: **Fable 5.1** 1,005 calls, 1,422,436 output tokens, 174,271,607 tokens in all (9.3 % of the tokens,
8.0 % of the output); **Sonnet 5.5** 23,287 calls, 16,414,151 output tokens, 1,700,842,262 tokens in all.
Through the v0.2.0 tag alone (phases A to C) the total is 1,847,394,749 tokens in 24,163 calls.

Who made the calls: the sessions themselves 1,417 calls and 390,716,074 tokens; the subagents they started
22,875 calls and 1,484,397,795 tokens, **94 % of the calls and 79 % of the tokens**. The five sessions started
25 workflow runs and 1,839 subagent transcripts (each workflow run's folder also holds one file that is not an
agent's transcript; the count leaves those out). [V]

The windows overlap in time. Phase C's first session opened at 19:51 on 10-29 while the build was still
running, so the phases are not strictly sequential.

## What the phases were

- **A · Design.** The four design documents under `docs/design/` (`review`, `scope`, `decisions` and
  `build-prompt`) were written and edited in this session, 18 write and edit calls in its transcript [V], and
  the owner accepted every decision in `decisions.md` on 2026-10-28, the day of the first commit (23:44). The session ran
  on until 04:15 on 10-29, so it overlaps the build. It opened with an unrelated question; its first 73 calls
  (9,461,258 tokens) came before any working directory in the quill repository and are left out. The rest still
  holds some discussion that was not about quill, so the row is an upper bound. [I]
- **B · Build.** The first message of `cb9b1313` asked it to build quill from `docs/design/build-prompt.md`.
  Its first call is at 23:39:44 on 10-28; the first commit is 4 minutes 41 seconds later. The code that became
  `v0.1.0` is 76 commits, from the first commit to `9364fb0` (2026-10-29 19:28:52), 19 hours 44 minutes. The
  first Sonnet call in the session is at 20:47:58 on 10-29 and the model never changes back. [V]
- **C · Audit and harden.** Three sessions: one wrote `docs/design/audit-prompt.md` and ran the first waves,
  one resumed the run after a restart of the editor, and the third ran the remaining waves, the hardening and
  the release rehearsal to the tag. All three are Sonnet 5.5, and so is every subagent they started. [V]
- **D · After the tag.** The v0.2.0 tag was created on 2026-11-05 at 02:50:11. The row is the same session
  as the one before it, cut at that moment and at a fixed end, 12:45 on the same day; the session was still
  running when this file was written, so the last row grows if you count later. [V]

## Methodology

- **Source.** The transcripts Claude Code writes for each session, one file for the session and one per
  subagent, including the agents a workflow starts (these sit one directory deeper than the others; counting
  only the top level of that folder misses most of them).
- **One call is one message id, not one line.** A transcript line is written for each content block of a
  response, so a call with a thinking block, some text and two tool calls appears on up to five lines, each
  carrying the same `usage`. [V] Summing the `usage` of every line over these five sessions gives 3,677,226,672
  tokens in 44,363 lines; counting each message id once gives 1,888,108,948 in 24,420 messages (a snapshot taken
  while the session was still running), a factor of 1.95. The tables here count each id once. In every repeated
  id checked (all 24,421 ids in the set), the repeated lines have identical usage, so which line is counted
  makes no difference.
- **Tokens.** `input_tokens` + `cache_creation_input_tokens` + `cache_read_input_tokens` + `output_tokens`
  from each call's `usage` block. The model is the call's own `message.model`. Twenty-four `<synthetic>`
  messages, all with zero usage, are left out.
- **Phase boundaries** are timestamps, converted to UTC in code from the transcripts' own UTC stamps and from
  git's offset-bearing dates: the model switch (the first Sonnet call in `cb9b1313`), the first call with a
  working directory in the quill repository (in `cf41ba29`), the creation of the v0.2.0 tag, and the fixed end.
- **Which sessions are quill.** Five sessions, chosen by their first request and their working directory, then
  checked against the commits. Of the 77 commits authored before the model switch, 41 are named by hash (old
  or new, see `docs/audits/commit-map.txt`) in the build session `cb9b1313`, 29 are named in no transcript, and
  7 are named only in later sessions, which cite older commits in the audits and the history rewrite; those 7
  are placed by their dates, which is an inference. Of the 309 commits after the switch, 238 are named in
  `70677139`, 57 in `4849b262` and 31 in `95f7a480`, and 26 in none. No other session in any project directory
  on this machine names a quill commit hash. A session that never printed a hash would not show, so this check
  shows completeness only for sessions that committed. [V]
- **One caution about the build window.** The 77 commits authored before the switch are not the same set as
  the ancestors of `v0.1.0`, though both have 77 members: the tag's code is 76 commits, and the tag now also
  contains the later `LICENSE` commit; one more commit before the switch (`b6c25ec`, 19:42, a documentation
  change) is not in the tag. [V]

## What this doesn't establish

- **Not a cost figure.** `cache_read_input_tokens` is 94 % of the total here (an already-cached, growing
  conversation read again on every call) and is billed at a small fraction of fresh input. These are token
  counts as logged, not spend.
- **Not a per-task ledger.** Claude Code does not tag usage by milestone. Phases are reconstructed from
  model-switch times, tag times and the hash cross-check above, which is the best available reconstruction and
  not an exact accounting. Two sessions that ran at the same time cannot be separated by anything but their
  own files, which is what this does.
- **Not a measure of who wrote which line.** A commit is attributed to a session by being named in it, and a
  session's subagents write commits too, in isolated worktrees that the session then merges. The attribution
  says which session handled a commit, not which model typed its diff.
- **Moving.** This is a snapshot: the transcripts of a running session grow while they are read. Counts of
  the whole set taken a few minutes apart differed by one or two messages (24,420, then 24,421, then 24,422
  message ids); the tables use a fixed end, and two runs of the same script against that end were
  byte-identical.
