# quill — the pen that never lifts

*A log is a directory. One hand writes it; any number read it, now or later, from this
process or another.*

A quill writes in one direction only. It does not go back to erase, it does not write two
lines at once, and what it has written stays where it was put. That is the whole of what this
library asks of a disk: append, fsync, and never touch the past. Everything else, the
consumers that resume where they stopped, the readers in other processes, the retention
that forgets by whole segments, follows from that one discipline.

**What it actually is.** quill is an embeddable, append-only log for Go on the standard
library alone. `quill.Open(dir)` returns a `Log` and takes an exclusive lock on the directory:
that process is the log's one assigned writer. Records are numbered from 1 and the numbers
are contiguous, including after a crash. A record is visible only once it is durable. A
consumer is a name and a committed offset, stored in its own file in the log directory,
written only by the process that holds it. Another process opens the same directory with
`OpenReader`, takes no lock at all, and sees exactly what the writer has published as durable
and nothing beyond it. Retention deletes whole sealed segments, oldest first, by age and by
caps, and nothing holds it back unless a consumer asked to be protected. There is no
broker, no socket, no replication, no dependency. "The SQLite of commit logs" is closer than
"a lightweight Kafka".

**The one thing nobody else does this small.** A reader in another process that needs no
coordination with the writer. The writer publishes one small file, `quill.tail`, after every
fsync: the last durable sequence and the byte where durable data ends. A reader trusts that
file alone. It never reads a byte past it, never guesses whether the writer is alive, and
never needs the writer to know it exists. Recovery treats the same file as a lower bound and
parses forward from it, so a marker that is stale after a power loss hides records for a
moment and never exposes one that is not durable. The guarantees in [scope](design/scope.md)
are fourteen sentences, and each has a test that could falsify it, run against a real
`SIGKILL` and against a simulated power loss.

**What the operator keeps.** Everything is a directory. Move it, back it up, delete it,
inspect it with `quill cat`, `quill check` and `quill stat`, archive its sealed segments with
`quill archive`, and set its retention and caps per log. The format is specified in
[format.md](format.md) and frozen by golden tests; the frame is a few varints, the payload
and a CRC, and will stay that way. Durability is a setting, not a design commitment: buffer
and flush for throughput, or `Sync` every record for a durable write. What quill will not do
is also written down: no sharding, no consumer groups, no compaction, no network. A log
that does one thing can be read in an afternoon, and that is the point.

*Read next:* [executive summary](executive-summary.md) · [user guide](userguide.md) ·
[architecture](architecture.md) · [format](format.md) · [command reference](reference.md)
