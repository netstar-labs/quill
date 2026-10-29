# quill — executive summary

## What it is

quill is a small Go library and command-line tool for a durable, ordered, replayable log
kept in one directory on local disk. One process writes a log; any number of consumers read
it, in that process or in others, each resuming where it stopped. It is embedded in the
program that uses it: there is no service to run, no network, and no third-party code.

## Why it exists

Many systems need "write events here, read them later, in order, without losing any" and
reach for a message broker, which brings a cluster, a protocol and an operations burden out of
proportion to the need. quill covers the case where one machine is enough: an audit trail, an
event feed between two processes on a host, a queue that must survive a crash, a buffer in
front of a network that is sometimes down. It replaces a private implementation whose
sharding, topic registry, network fan-in and second on-disk format had grown past what the
core needed; quill keeps that core's frame and guarantees and changes its shape (see the
[design review](design/review.md)).

## What it guarantees

- **Order and contiguity.** Records are numbered from 1 with no gaps, including after a crash.
- **Durable before visible.** Readers see only records that have been fsynced; a reader in
  another process sees only what the writer has published as durable.
- **A crash loses a suffix, never a middle.** At most the unflushed tail is lost; nothing
  acknowledged as durable is lost or duplicated.
- **Corruption is never served silently.** Damage is reported as a gap and never skipped.
- **Bounded disk.** With a byte cap set, a log's size stays within a stated bound; a full disk
  never corrupts a log.
- **One writer per log**, enforced by a lock; any number of readers, with no lock.

Each guarantee has a test that could falsify it, run under a real `SIGKILL` and against a
file-system simulator that models power loss.

## What it is not

Not a broker: no network, no replication, no consumer groups, no partitions. Not a database:
no keys, no transactions, no compaction. Not portable beyond Linux and macOS. Several topics
are several directories, each with its own writer.

## Where it stands

v0.1.0 is the first release: the format, the writer, the lock-free reader, consumers,
retention and caps, the `quill` command, examples, documentation, a crash-simulation test
suite and recorded benchmark baselines. v0.2.0 is the same code after the full audit
sequence. The on-disk format is versioned separately from the module and is frozen by golden
tests; the module stays at `v0` so that breaking changes remain possible with a minor bump.

*Read next:* [introduction](introduction.md) · [architecture](architecture.md) ·
[user guide](userguide.md)
