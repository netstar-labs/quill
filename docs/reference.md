# quill command reference

*Generated from the verb table in `cmd/quill/verbs.go` (`quill reference`); a test keeps it current. Do not edit by hand.*

`quill [-dir DIR] <verb> [flags]`. Every verb but `help` and `version` takes `-dir`, the log directory
(for `ls`, the parent directory holding the logs). Exit codes: 0 ok, 1 error, 2 usage, 3 `check` found the log UNHEALTHY.

| Verb | Lock | Summary |
|---|---|---|
| `ls` | none | list the logs under a parent directory, with bytes, segments, retention and caps from quill.policy |
| `stat` | none | print a log's head, first retained sequence, segments, bytes, records, holes and marker age |
| `read` | none | print records from an offset (default the first retained) |
| `cat` | none | print every retained record, from the first |
| `tail` | none | print the last records, and keep following with -follow |
| `append` | log | append the arguments as records, or each line of stdin, as one batch (MaxBatchBytes applies, about 479,000 one-byte lines); needs the writer's lock |
| `consumers` | none | list the consumers: offset, lag, protection |
| `commit` | consumer | set a consumer's offset: -offset <next sequence> or -time <time> |
| `register` | consumer | create a consumer at the first retained sequence, or change its protection |
| `unregister` | consumer | delete a consumer's offset file (its lock file stays) |
| `check` | none | verify every segment, hole, consumer file and the clock; exit 3 when UNHEALTHY |
| `prune` | log | run a retention pass now with the policy in quill.policy or the flags; needs the writer's lock |
| `roll` | log | seal the active segment now; needs the writer's lock |
| `archive` | consumer | copy sealed segments not yet archived to -to, verifying every CRC; runs beside a live writer |
| `recover` | log | open the log writable, which repairs a torn tail and republishes quill.tail, then close it |
| `help` | none | print this list, or a verb's flags |
| `version` | none | print the version |

## `quill ls`

List the logs under a parent directory, with bytes, segments, retention and caps from quill.policy.

Lock: none (runs beside a live writer, sees records up to quill.tail).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the parent directory holding the logs |

## `quill stat`

Print a log's head, first retained sequence, segments, bytes, records, holes and marker age.

Lock: none (runs beside a live writer, sees records up to quill.tail).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
| `-json` | `false` | one JSON object |

## `quill read`

Print records from an offset (default the first retained).

Lock: none (runs beside a live writer, sees records up to quill.tail).

| Flag | Default | Meaning |
|---|---|---|
| `-b64` | `false` | print every payload as base64 |
| `-dir` | `—` | the log directory |
| `-from` | `0` | the first sequence to print (default the first retained) |
| `-json` | `false` | newline-delimited JSON: seq, at, time, len, payload or payload_b64 |
| `-limit` | `0` | stop after this many records (0: no limit) |
| `-since` | `—` | start at the first record at or after this time (RFC 3339, @unix, or a duration ago such as 2h) |

## `quill cat`

Print every retained record, from the first.

Lock: none (runs beside a live writer, sees records up to quill.tail).

| Flag | Default | Meaning |
|---|---|---|
| `-b64` | `false` | print every payload as base64 |
| `-dir` | `—` | the log directory |
| `-from` | `0` | the first sequence to print (default the first retained) |
| `-json` | `false` | newline-delimited JSON: seq, at, time, len, payload or payload_b64 |
| `-limit` | `0` | stop after this many records (0: no limit) |
| `-since` | `—` | start at the first record at or after this time (RFC 3339, @unix, or a duration ago such as 2h) |

## `quill tail`

Print the last records, and keep following with -follow.

Lock: none (runs beside a live writer, sees records up to quill.tail).

| Flag | Default | Meaning |
|---|---|---|
| `-b64` | `false` | print every payload as base64 |
| `-dir` | `—` | the log directory |
| `-follow` | `false` | keep printing new records until interrupted |
| `-from` | `0` | the first sequence to print (default the first retained) |
| `-json` | `false` | newline-delimited JSON: seq, at, time, len, payload or payload_b64 |
| `-limit` | `0` | stop after this many records (0: no limit) |
| `-n` | `10` | how many of the last records |
| `-since` | `—` | start at the first record at or after this time (RFC 3339, @unix, or a duration ago such as 2h) |

## `quill append`

Append the arguments as records, or each line of stdin, as one batch (MaxBatchBytes applies, about 479,000 one-byte lines); needs the writer's lock.

Lock: the writer's lock (fails with ErrLocked beside a live writer).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
| `-sync` | `true` | return only once the records are durable |

## `quill consumers`

List the consumers: offset, lag, protection.

Lock: none (runs beside a live writer, sees records up to quill.tail).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
| `-json` | `false` | one JSON object per line |

## `quill commit`

Set a consumer's offset: -offset <next sequence> or -time <time>.

Lock: the consumer's lock only (runs beside a live writer).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
| `-name` | `—` | the consumer |
| `-offset` | `—` | the next sequence to read |
| `-time` | `—` | the time to seek to (RFC 3339, @unix, or a duration ago such as 2h) |

## `quill register`

Create a consumer at the first retained sequence, or change its protection.

Lock: the consumer's lock only (runs beside a live writer).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
| `-name` | `—` | the consumer |
| `-no-protect` | `false` | retention does not wait for it (the default for a new consumer) |
| `-protect` | `false` | retention keeps its unread segments |

## `quill unregister`

Delete a consumer's offset file (its lock file stays).

Lock: the consumer's lock only (runs beside a live writer).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
| `-name` | `—` | the consumer |

## `quill check`

Verify every segment, hole, consumer file and the clock; exit 3 when UNHEALTHY.

Lock: none (runs beside a live writer, sees records up to quill.tail).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
| `-json` | `false` | one JSON object |

## `quill prune`

Run a retention pass now with the policy in quill.policy or the flags; needs the writer's lock.

Lock: the writer's lock (fails with ErrLocked beside a live writer).

| Flag | Default | Meaning |
|---|---|---|
| `-consumer-ttl` | `—` | a protected consumer behind for this long stops protecting |
| `-dir` | `—` | the log directory |
| `-force` | `false` | lift the clock guard (delete every expired segment in one pass) |
| `-max-bytes` | `—` | cap on the log's segment bytes |
| `-max-records` | `—` | keep roughly the newest N records |
| `-min-free-bytes` | `—` | free-space floor on the filesystem |
| `-retention` | `—` | age after which a sealed segment is deleted (a duration; negative never) |

## `quill roll`

Seal the active segment now; needs the writer's lock.

Lock: the writer's lock (fails with ErrLocked beside a live writer).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |

## `quill archive`

Copy sealed segments not yet archived to -to, verifying every CRC; runs beside a live writer.

Lock: the consumer's lock only (runs beside a live writer).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
| `-to` | `—` | the destination directory |

## `quill recover`

Open the log writable, which repairs a torn tail and republishes quill.tail, then close it.

Lock: the writer's lock (fails with ErrLocked beside a live writer).

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `—` | the log directory |
