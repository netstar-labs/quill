# quill — the pen that never lifts

*A log is a directory. One hand writes it; any number read it, now or later, from this
process or another.*

A quill writes in one direction only. It does not go back to erase, it does not write two
lines at once, and what it has written stays where it was put. That is the whole of what this
library asks of a disk: append, fsync, and never touch the past. Everything else, the
consumers that resume where they stopped, the readers in other processes, the retention
that forgets by whole segments, follows from that one discipline.

## What it actually is

An embeddable, single-node, append-only log. There is no server to run, no broker to size and
no cluster to keep healthy. You import it, point it at a directory, and it gives you what a
durable log needs: a log per topic, each its own directory; a sequence number per log that only
goes up, with no gaps even after a crash; records that become visible only once they are on
disk; consumer offsets that survive a restart; retention that forgets whole segments on
schedule unless a consumer asked to be protected; and an archiver. It is built on the Go
standard library alone, with no module dependencies. Think of it as the SQLite of commit logs,
not a lightweight Kafka.

## The one thing nobody else does this small

A reader in another process that needs no coordination with the writer. The writer publishes
one small file, `quill.tail`, after every fsync: the last durable sequence and the byte where
durable data ends. A reader trusts that file alone. It never reads a byte past it, never
guesses whether the writer is alive, and never needs the writer to know it exists. Recovery
treats the same file as a lower bound and parses forward from it, so a marker that is stale
after a power loss hides records for a moment and never exposes one that is not durable. Each
of the fourteen guarantees in [scope](design/scope.md) has a test that could prove it false,
run against a killed process and against a simulated power cut.

## What the operator keeps

Everything is a directory. Move it, back it up, delete it, inspect it with `quill cat`,
`quill check` and `quill stat`, archive its sealed segments with `quill archive`, and set its
retention and caps per log. The on-disk format is small, a record's bytes with a checksum and
a few numbers around them, specified in [format.md](format.md), and it will stay that way.
Durability is a setting, not a design commitment: buffer and flush for throughput, or `Sync`
every record for a durable write. Your choice.

## What quill will not do

No sharding, no consumer groups, no compaction, no network. If you need a socket or remote
fan-in, writing one on top is easy, but none ships, on purpose. A log that does one thing can
be read in an afternoon, and that is the entire point.

*Read next:* [executive summary](executive-summary.md) · [user guide](userguide.md) ·
[architecture](architecture.md) · [format](format.md) · [command reference](reference.md)
