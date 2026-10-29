# Self-review: the recovery path and the reader

*Phase 4, 2026-10-28, before the A1 audit. A hostile read of `load.go`, `flush.go`,
`reader.go` and `read.go` by the build itself: for each load-bearing claim, what was checked,
what was found, and what was changed. The A1 audit should treat this as a list of places
already looked at, not as clearance.*

## Recovery (`load.go`, `recoverSegment`, `verifySegment`)

| Claim | Check | Finding |
|---|---|---|
| Bytes below the marker's offset must verify whole, and the marker's offset must sit on a frame boundary | `recoverSegment` walks `[0, durable)` with `w.size = durable`, so a frame crossing `durable` is torn, hence `SegmentError`; T2 tests every off-boundary offset | Holds |
| The marker cannot name an offset past the file | `durable > size` is `SegmentError`; the marker is written only after `fsync(file)` made the size durable, so this is corruption, never a lost size | Holds under the simulator's model; a filesystem that persists a rename before a prior `fsync` of another file would break it, and is outside the supported set |
| Parse-forward trusts nothing after the first bad frame | `walk` stops at the first error and `recoverSegment` returns `tail.end`; the embedded-segment and zeroed-page tests | Holds |
| A marker above the last valid frame is refused, except with no frames at all | `hasFrames && head < tl.Seq`; T2 | Holds |
| A sealed segment that fails to verify is a hole, never modified; the newest failing is `ErrCorrupt` | `loadSealed` sets `bad`; `load` returns the `SegmentError` for the newest | Holds. A damaged sealed segment that is the one a stale marker names is still a hole, since it is not the newest: consistent with decision 39 |
| Overlapping ranges are corruption | `s.first <= prevLast` and the active's first; T2 copy test | Holds |
| Truncation of a torn tail is fsynced | `Truncate` then `Sync`; the shadowed-error bug that ignored the `Sync` result was found by staticcheck in phase 2 and fixed | Holds now; the kind of bug worth a second look by the audit |
| Segment ids are never reused | `maxSeg = max(tl.MaxSeg, newest)`; rotation uses `maxSeg + 1` | Holds, and a resurrected unlink after a crash cannot collide because the marker's `maxseg` was published and the directory fsynced before the unlink |
| The marker is durable at rotation | **Found wanting**: rotation fsynced the directory after creating the segment but not after the marker rename, against decision 65 | **Fixed in phase 4**: `publishLocked` fsyncs the directory after the marker rename when a seal happened; a test pins the durable marker after `Roll` and a crash |

Residual, as designed: damage between a stale marker and the true end of the newest segment
reads as a torn tail. The window is now bounded by a segment's worth of flushes, as decision
39 says, since the marker is durable at every rotation.

## The flush and its failure classes (`flush.go`)

| Claim | Check | Finding |
|---|---|---|
| A write error leaves no partial frame | `Truncate(off)` back to the durable offset before the records are retried | Holds; T3 checks the bytes |
| An fsync error is sticky | `fail` sets `failed`; every entry point checks it first | Holds |
| Durable data is never rewritten after a marker failure | `a.durable` and `head` advance before `publishLocked`; the retry writes at the new offset | Holds; T3 compares the bytes |
| A directory fsync error is fatal | `rotateLocked` and now the post-rename fsync call `fail` | Holds |
| `Sync` holds no lock while waiting | The wait is on a channel outside the mutex; `flushGen` tells a failed flush from a stale one | Holds; T18 |
| `deleteLocked` calling `publishLocked` cannot recurse unboundedly | After a rotation the new active is empty, so the nested call does not rotate and `sealed` is false | Holds, but the shape is tangled; a candidate for the audit's simplification pass |

## The reader (`reader.go`, `read.go`)

| Claim | Check | Finding |
|---|---|---|
| A reader never serves a byte past the marker | The active segment is opened with the window bounded to `tl.Off`; `rec.Seq > v.head` stops the walk | Holds; T6 and T20 |
| A sealed segment's derived range never serves another segment's records | Ranges end at the next first frame; a scan reveals a shorter real end and the gap is reported, once | Holds after the phase-3 fix of a doubled hole |
| A cached first frame cannot serve wrong sequences after a writer crash | Only sealed segments are cached, and a sealed segment is immutable; the active segment is read afresh | Holds; a sealed segment replaced by a file of the same size would be served from the stale cache until a CRC failure, which the open-time size check does not catch. Accepted: sealed files are never rewritten by quill, and `Check` verifies whole |
| A cursor position stays valid across calls | Durable bytes never move; ids are never reused; the cursor is used only when the segment id and the sequence match | Holds |
| A gap after delivered records is reported consistently | **Found**: two error paths in `scanSource` returned nil after delivered records even in `Scan` mode | **Fixed in phase 4**: every such path honours `gapAfter` |
| A reader costs no file operation per cached segment | `segment(id)` returns a cached entry without I/O (decision 83); T19 measures it | Holds |
| `Check` examines the active segment only to the marker | `verifyBounded(f, tl.Off)` for `id == tl.Seg`; ids above are skipped | Holds |

## What the audit should look at first

1. The `publishLocked` / `deleteLocked` / `rotateLocked` shape under failure injection at
   every step, with a reader attached: the T1 phase-3 scenario covers the simulator; the audit
   should try to construct a sequence the scenario does not reach.
2. The reader's hole bookkeeping across a bad first sealed segment, a scanned segment with
   a short real end, and an empty active segment, in every order.
3. `ConsumerTTL`'s reads of the record at the offset (decision 82) while a prune pass runs
   concurrently in another goroutine: the read takes the lock briefly and the pass does not
   hold it meanwhile, so a commit can land between; that is accepted, but worth a reviewer's
   eye.
