# Quill on-disk format, version 1

*Normative. The frame and every state file below are frozen by golden tests
(`golden_test.go`, `testdata/golden/`). A change to any of them needs a new version in
`quill.format` and a migration note; a binary reads the versions it knows and refuses a newer
one with `ErrFutureVersion`. The module version and the format version are separate.*

## A log is a directory

```
{log}/{id:010d}.log          segment files, ids ascending, never reused
{log}/quill.tail             the durable position: {"seq":N,"seg":ID,"off":BYTES,"maxseg":ID}
{log}/quill.format           "quill/1 <log id>"
{log}/quill.policy           the writer's Config, informational, parsed tolerantly
{log}/quill.lock             the writer's exclusive flock (advisory; the file is empty)
{log}/consumers/{name}       "<log id> <next> <protect>", one line
{log}/consumers/{name}.lock  held while a process has the consumer open; never deleted
{log}/**/.tmp-*              temp files; ignored by every scan; the writer removes those older than an hour
```

The directory's base name is the topic name; it is not stored anywhere inside. Once
`quill.format` exists, unknown files in the directory are ignored; a file named like a segment
that fails validation is a load error. Directories are created `0700` and files `0600` by
default.

## The frame

```
frame = uvarint(bodyLen) | body | crc32(body)        crc: 4 bytes, big-endian, IEEE polynomial
body  = uvarint(seq) | uvarint(at) | uvarint(len(payload)) | payload
```

- `uvarint` is the unsigned LEB128 of `encoding/binary` (`AppendUvarint`), at most ten bytes.
- `seq` is the record's sequence number, per log, from 1.
- `at` is the writer's wall clock in whole Unix seconds, an `int64` written as
  `uvarint(uint64(at))`; a negative value therefore takes ten bytes.
- `payload` is opaque bytes. The body is exactly three varints followed by exactly
  `len(payload)` bytes: a body with slack after the payload is damage.
- The CRC covers the body only, not the length prefix.
- Framing costs the three varints, the length prefix and the CRC: about 14 to 18 bytes per
  record for ordinary sequences and payload sizes.

Recorded frames (hex), from the golden tests:

| Record | Frame |
|---|---|
| seq 1, at 1700000000, payload `hello` | `0c 01 80e2cfaa06 05 68656c6c6f 3b5aa01c` |
| seq 2^40, at 0, empty payload | `08 8080808080 20 00 00 c44de4d0` |
| seq 2^64−1, at −2, payload `ff ff ff` | `18 ffffffffffffffffff01 feffffffffffffffff01 03 ffffff dfce82f5` |

### Limits

| Limit | Value | Rule |
|---|---|---|
| Declared body length | 64 MiB | A larger declared length is corruption, not an allocation |
| Payload | 64 MiB − 32 bytes | The ceiling of `MaxPayloadBytes`; the three varints are at most 30 bytes, so a legal body always fits |
| Segment file | 1.5 GiB | A larger file is refused |
| `SegmentBytes` | at most 1 GiB | A segment passes it by the flush that seals it |
| State file | 1 MiB | A larger one is damage |

## Segments

A segment file holds frames back to back with nothing between them: no header, no footer, no
padding. Sequences are **contiguous**: every frame's `seq` is exactly one more than the
previous frame's, within a segment and from one segment's last frame to the next segment's
first. Segment ids ascend and are never reused.

The newest segment (the highest id) is the **active** one and may be empty. A segment is
**sealed** exactly when a higher-numbered segment exists; nothing is written to mark it, and a
sealed segment with no frames is corruption.

Because sequences are contiguous, a sealed segment's range is `[its first sequence, the next
segment's first − 1]`, and the active segment's range ends at `quill.tail.seq`. A reader needs
only each segment's first frame to find a sequence.

### Verification and recovery

Let the marker `quill.tail` name segment S at offset X.

- Every byte of a sealed segment, and every byte of S below X, was fsynced before the marker
  named it. Those bytes must decode as intact, contiguous frames ending exactly at the file's
  end (a sealed segment) or exactly at X (the active one). Anything else is **corruption**:
  the file is never modified, readers see the segment's range as a gap, and a writable open of a
  log whose newest segment is corrupt fails with `ErrCorrupt`.
- From X (or from byte 0 when the marker names an older segment than the newest), recovery
  **parses forward**, accepting each frame that is intact (its CRC matches), within the length
  limit, and whose `seq` is exactly one more than the previous accepted frame's. It truncates
  the file at the first frame that is not, and trusts nothing after it, whatever it contains: an
  intact-looking frame inside a payload is never reached because its predecessor failed.
- If the log holds frames and the last valid sequence is below `quill.tail.seq`, acknowledged
  data is missing and a writable open fails with `ErrCorrupt`. If it holds no frames at all,
  the next sequence is `quill.tail.seq + 1`.
- Residual: damage between a stale marker and the true end of the newest segment looks like a
  torn tail and is truncated. The marker is made durable (directory fsync) at every rotation, at
  close and before any deletion, which bounds that window to the flushes within one segment.

A **hole** is a discontinuity between one segment's last sequence and the next segment's
first, including one left by a file that cannot be parsed or a segment deleted by hand.
Retention deletes only the oldest segments, so a middle hole is always damage.

## State files

Every state file is written by temp file, fsync, rename. Every one except `quill.policy` is
refused when malformed: unknown fields, a missing field, trailing data, a duplicate key, invalid
UTF-8, a lone surrogate escape, a number with a sign, a leading zero or a fraction are all refused.
A write has no directory fsync unless stated, so after a power loss a state file may be the
previous version, never a torn one.

### `quill.tail`

One JSON object, fixed field order, trailing newline:

```
{"seq":1234567,"seg":42,"off":8000001,"maxseg":45}
```

| Field | Meaning |
|---|---|
| `seq` | The last durable sequence; `0` when the log has none |
| `seg` | The id of the segment the writer is appending to |
| `off` | The durable length of that segment in bytes |
| `maxseg` | The highest segment id ever created; a new segment's id is above it and above every id on disk |

`seg` is at most `maxseg`. The writer rewrites the file after each flush that changed it and
before any segment is deleted. A reader trusts the marker alone: it never reads past `off` in
segment `seg`, never past `seq`, and treats segments above `seg` as not yet published.

### `quill.format`

One line: `quill/<version> <log id>` and a newline. The version is a plain decimal; the log
id is 32 lower-case hex digits, 128 random bits chosen when the log is created.

```
quill/1 0123456789abcdef0123456789abcdef
```

A version above the one a build knows is `ErrFutureVersion`. The file is required: a
directory with segments and no marker is refused. It is written first when a log is created,
and `quill.tail` last, so a directory with `quill.format` but no `quill.tail` and no frames is an
interrupted creation that the next writable open finishes.

### `consumers/{name}`

One line, three fields separated by single spaces, and a newline:

```
0123456789abcdef0123456789abcdef 1234568 1
```

| Field | Meaning |
|---|---|
| log id | Must equal the log's; otherwise the consumer belongs to another log (`ErrWrongLog`) |
| next | The next sequence to read: a plain decimal. A caught-up consumer has `Head + 1` |
| protect | `1` if retention keeps the consumer's unread segments, else `0` |

Names are `[a-z0-9_-]`, 1 to 200 characters; a leading underscore is reserved (`_archive` is
the archiver). The file is written only by the process holding `consumers/{name}.lock`.

### `quill.policy`

JSON, indented, the settings the writer last opened with. Informational: `Config` at open is
authoritative and rewrites it; offline tools read it. Unknown fields are ignored and a parse
failure never blocks an open; an offline tool that cannot parse it refuses to act on it.

```
{
  "retention": "24h0m0s",
  "segment_bytes": 8388608,
  "segment_max_age": "24h0m0s",
  "max_bytes": 0,
  "min_free_bytes": 0,
  "max_records": 0,
  "max_payload_bytes": 67108832,
  "consumer_ttl": "0s",
  "protected_consumers": ["_archive", "billing"]
}
```

Durations are Go duration strings; a cap that is off is `0`; a negative `retention` disables
the age sweep.

### `quill.lock` and `consumers/{name}.lock`

Empty files that exist to be locked with `flock(LOCK_EX | LOCK_NB)`. The lock is released
when the holder closes the file or its process ends, so a crashed holder never blocks a
successor. Lock files are never deleted.

## Durability points

| Event | What is fsynced |
|---|---|
| Flush | The segment's data, then `quill.tail` (its temp file) before the rename. No directory fsync |
| Rotation | The data, the new empty segment's creation (directory fsync), then `quill.tail` |
| Close | The data, `quill.tail`, then the directory |
| Deletion | `quill.tail` is published and the directory fsynced **before** any segment is unlinked |
| Commit | The consumer file's temp file before the rename; no directory fsync |
| Creation | `quill.format`, `consumers/`, the first segment, `quill.tail`, then the directory |

After a process crash the marker is at most one flush behind the data; after a power loss it
may be older by the flushes since the last directory fsync. Either way it hides records and
never exposes one that is not durable, and the next writable open republishes it.
