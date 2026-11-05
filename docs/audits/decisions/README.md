# Decision ledger: index

[stage2-owner-decisions.md](stage2-owner-decisions.md) is the ledger of every item the stage 2 audit held for the owner: what the
finding was, why it was held, the recommended answer, and, where there is one, the owner's answer and date, the fix commit and the
fix-skeptic's outcome (some items are recorded as not approved, with no fix).
It is one file of about 3,000 lines, kept whole because other records cite it by section. Its text is unchanged by the reorganisation of
`docs/audits/`; this index was generated from its headings (so it lists every section, in file order, and nothing else).

To find an item: search the ledger for its id (for example `A3-26`, `L3b-8`, `A1d-9`, `L5-3`). The pass records under
[../passes/](../passes/) and [../reference/](../reference/) hold the audit's own text for the same ids.

The numbered decisions the ledger refers to (decision 51, decision 101 and so on) are a different list: [docs/design/decisions.md](../../design/decisions.md).

## Sections

- A1-67: the documented rewind of an ahead consumer, `quill commit`, cannot work (high)
- A1-80: a live protected consumer stops the writer from restarting (high)
- A1-25: after a failed rotation the active segment can outgrow the 1.5 GiB limit (medium)
- A1-32: the reserved `_archive` consumer cannot be removed or rewound (medium)
- A1-47: extra positional arguments are silently ignored (medium)
- A1-48: `-from`, `-limit` and `-n` parse numbers in base 0 (medium)
- A1-82: a read I/O error is reported as ErrCorrupt or a skippable gap (medium)
- A1-85: Reader-created consumer files ignore the log's Perm (medium)
- A1-13: Archive returns ErrCorrupt, not ErrFutureVersion, for a newer destination format (medium)
- A1-69: MaxBatchBytes is checked against a 34-byte-per-record estimate (medium)
- A1-2: a case-variant duplicate key in quill.tail wins silently (low)
- A1-29: Open refuses a symlinked Dir that OpenReader accepts (low)
- A1-35: one future-stamped segment pins age expiry of every newer one (low)
- A1-50: `-json` passes C1 control characters raw (low)
- A1-52: Read can exceed MaxReadBytes by one record (low)
- A1-57: `stat -json` holes use capitalised keys and `null` (low)
- A1-87: the clock guard rounds up where the text says "at most 25%" (low)
- A1-91: a quill.policy of `{}` is acted on as retention 0 (low)
- Not held, but unresolved
- Pass A4: dependency, supply-chain and release hygiene
- A4-5: the archives ship the Go runtime with no copy of its licence (low)
- A4-4: a workspace path remains in the v0.1.0 tag and in earlier history (medium; the tip is fixed)
- A4-3: the reporting channel in SECURITY.md needs a repository setting (medium)
- A4-1: the Go toolchain pin is enforced only by the gate (medium)
- Not held, but worth a line at the flip
- Pass A3: security review and threat model
- A3-5: the writer's lock file is readable by readers, so a reader can hold it (medium)
- A3-14: `archive -to` writes into a directory that is not a log (medium)
- A3-18: a Reader's read cost scales with the log, not the request (low)
- A3-19: the first-frame probe reads and CRC-checks the whole first frame (low, PLAUSIBLE)
- A3-10: a damaged oldest segment is skipped silently by a default-start read (low)
- A3-20: a Reader serves overlapping segments (low)
- Not held, but worth a line
- Pass A6: format and version-skew audit
- A6-4: the segment id has no ceiling (low)
- A6-8: two constants differ only by letter case (low)
- Not held, but unresolved
- Pass A5: silent-degradation audit
- A5-10: `Err()` is masked and never clears (medium)
- A5-1: background failures are not logged, timed or counted (medium)
- A5-36: `quill prune` exits 0 when it did less than asked (medium)
- A5-18: a consumer that fell out of retention reads healthy (medium)
- A5-22: `MinFreeBytes` fails open when statfs fails (medium)
- A5-29: `quill archive` and source loss (medium)
- A5-3: waiting on a failed log never fails (low)
- A5-9: `quill prune` and `quill roll` discard `Close`'s error (low)
- A5-23: the published disk bound is wrong for `MaxBufferBytes` below `FlushBytes` (low)
- A5-32: an archive rerun keeps a same-size damaged destination segment (low)
- A5-44: `quill ls` omits a log it cannot open (low)
- Not held, but worth the owner's eye
- Pass L1: verification and optimisation
- L1-11: `quill check` reports an empty oldest sealed segment healthy (medium)
- L1-12: `quill archive` into a new destination copies only a suffix and exits 0 (medium as filed, PLAUSIBLE)
- L1-4: "every later Sync fails" on a failed log is not what the code does (low)
- L1-5: the sequence returned with an append error is inconsistent and undocumented (low)
- L1-16: `Consumer.ReadWait` does not return when the consumer closes (low)
- L1-26: the `quill append` oversize message states a size that stdin does not have (low)
- L1-51: a 16 KiB index stride reads faster at random positions for 4x the index memory (low, PLAUSIBLE)
- Open after a rejected fix (not held for an owner decision)
  - Residuals the fix-skeptics left on the fixed items (2026-11-03)
- Pass L2: test, benchmark and hardening suite
- L2-53: `MaxReadBytes` bounds payload bytes, not the memory of a read (medium)
- Pass L3b: live multi-process validation
- L3b-7: a second writer under another uid gets a bare permission error, not `ErrLocked` (low)
- L3b-8: a completely full filesystem makes every writable open of an existing log fail with ENOSPC (medium)
- Round-1 decisions confirmed 2026-11-03
  - The seven items fixed in this pass
  - The eight decisions kept unchanged
- Held after pass A3r
  - A3-7 (code hardening): open `consumers/` without following a symlink
  - A3-24 (`time` field): range-check the printed time
  - A3-26: unlinking `quill.lock` under a live writer lets a second writer open
- Held after pass A7
  - A7-15: figures from the reference remain in review.md and decisions 3, 15 and 32 (medium)
  - A7-18: no upgrade, migration or rollback note for an operator (low)
  - A7-19: no install path documented; a go-installed binary reports `dev` (low)
- Held after pass A1d
  - A1d-9: the logger documentation's list of records `Open` gives itself is incomplete (low)
  - Not held, but worth the owner's eye: A1d-1 (medium)
- Held after pass A4r
  - A4r-1: history still carries a personal scratch path and the reference's figures (medium)
  - A4r-4: the tree is not yet publishable until the licence and the GitHub-side settings are done (medium)
  - A4r-5: design documents still describe the origin's architecture in detail (low)
- Pre-tag batch 2026-11-04
  - A1d-9: the `Config.Logger` rule lists the records `Open` gives itself, and the list is incomplete (low)
  - L4b-1: the release record's migration table lists 8 breaking changes and omits changes `CHANGELOG.md` records (medium)
  - L4b-2: the release record does not roll up the items still held or deferred for the owner (medium)
  - L4b-3: `userguide.md` says `quill check -dir <dest>` keeps reporting the hole after archiving a source with a damaged oldest segment; it reports HEALTHY (medium)
  - L4b-4: records cite figures that no longer reproduce at HEAD (low)
- Held after the pre-tag batch
  - L3b-15 (re-raised by the critic): `quill -h` and `quill --help` print `flag: help requested` and exit 2 (low)
  - The `quill archive` message "the archive holds the same hole" is inaccurate when the destination starts after the lost records (low; follow-up of L4b-3)
  - Platforms: the 64-bit-only claim and the 32-bit build (no id; raised by the L4 critic, recorded in the release record)
- Owner decisions 2026-11-04, after wave 7 (hand finish by the assistant, no fix-skeptic re-run)
- Pre-tag code batch 2026-11-04
  - L5-1: `quill -h` and `quill --help` print `flag: help requested` and exit 2 (low)
  - L5-2: the `quill archive` message "the archive holds the same hole" is wrong when the destination starts after the lost records (low)
  - L5-3: the module builds for 32-bit targets that the documents do not list and the tests cannot compile for (medium)
  - L5-4 (A4-5): the release archives ship the Go runtime and standard library with no copy of its licence (low)
  - L5-5 (follow-up of L3b-15): `quill --version` and `quill -version` are usage errors (low)
- Held after the pre-tag code batch
  - `quill --version` (follow-up of L3b-15), and the test that pins it
  - A3-7 (code hardening): a symlinked `consumers/` is followed
  - A3-24 (`time` field): range-check the printed time
  - L3b-7: wrap the different-uid permission error like `Archive`
