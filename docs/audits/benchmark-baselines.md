# Benchmark baselines

*T24 (decisions 21, 70). Six runs of every benchmark in `bench_test.go` on one machine, the
median and the range recorded. A later release has regressed a benchmark when the median of
six runs is more than 10% worse and the ranges do not overlap. Re-record with:*

```
GOWORK=off go test -run '^$' -bench . -benchmem -count 6 . | tee bench.txt
```

*128-byte payloads, the default configuration except `Retention: -1` and no background
flush (every flush is the benchmark's own). "Read" is `Read(offset, 100)` on a warm 400 K-record
log. On macOS every fsync is `F_FULLFSYNC`.*

## macOS, Apple silicon (the build machine), 2026-10-28

Darwin 25.6.0, arm64, 10 cores, APFS on the internal SSD, Go 1.27.3, quill at the commit that
tags v0.1.0.

| Benchmark | Median ns/op | Range (min–max) | Runs | Median B/op | Median allocs/op | Other |
|---|---|---|---|---|---|---|
| `BenchmarkAppendBuffered` | 22,655 | 21,324–23,981 | 6 | 8 | 0 | 5.65 MB/s |
| `BenchmarkAppendFlushEach` | 9,990,037 | 9,765,522–10,539,680 | 6 | 1,290 | 14 | 0.01 MB/s |
| `BenchmarkAppendBatchSync` | 10,337,037 | 10,219,898–10,474,447 | 6 | 3,882 | 14 | 6.34 MB/s |
| `BenchmarkReadStart` | 26,650 | 26,579–26,916 | 6 | 28,260 | 136 | — |
| `BenchmarkReadTail` | 22,732 | 22,356–23,265 | 6 | 27,366 | 129 | — |
| `BenchmarkReadSealed` | 38,792 | 37,474–39,937 | 6 | 27,708 | 136 | — |
| `BenchmarkScanAll` | 18,682,048 | 18,622,615–18,978,407 | 6 | 72,687 | 119 | 21,523,194 records/s |
| `BenchmarkReadWait` | 10,136,097 | 9,955,176–10,397,931 | 6 | 28,187 | 25 | — |
| `BenchmarkColdStart` | 23,044,597 | 22,550,728–23,646,150 | 6 | 7,421,258 | 366 | — |
| `BenchmarkParallelLogsFlush` | 24,255,678 | 23,260,447–25,145,656 | 6 | 5,653 | 66 | 0.02 MB/s |
| `BenchmarkReaderFollow` | 10,405,351 | 10,037,283–10,778,244 | 6 | 168,786 | 543 | — |
| `BenchmarkDrainBacklog` | 114,334,530 | 108,669,608–116,420,185 | 6 | 901,756 | 176 | 2,347.85 MB/s, 16,399,074 records/s |

**The cost of the marker.** A flush is two `F_FULLFSYNC`s on macOS: the segment's data and
the marker's temp file. `AppendFlushEach` (one record per flush) is about 10 ms per record,
about 100 durable records per second for a lone producer; `AppendBatchSync` (512 records per
flush) is the same 10 ms per flush, about 50 K durable records per second. The second fsync
is close to half of that time on this disk; it is what a lock-free reader in another process
costs (decision 8). `ReadWait` and `ReaderFollow` are dominated by the same flush: a waiting
in-process reader sees a record about 10 ms after `Append`, an external reader polling at
1 ms about 10.4 ms.

**Reads.** A 100-record read costs about 27 µs at the start of the log, 23 µs near the tail
and 39 µs at a random position in a sealed segment; the difference is opening a sealed
segment per read (decision 15) and the seek within it. The sparse index and the
first-chunk size (decision 91) keep the cost independent of the backlog (T19).

**Scans.** A full in-process `ScanAll` decodes about 21.5 M records per second warm; a
`Reader` draining a 256 MiB backlog decodes about 16.4 M records per second, about 2.3 GB/s.
The backlog fits in memory; the larger-than-memory measurement of decision 15 was not run
(see `untestable-without-x.md`).

**Cold start** of a 400 K-record, 58 MiB log, every byte read and CRC-checked: about 23 ms.

**Four logs flushing in parallel** take about 24 ms per round on this disk, not the 10 ms of
one: `F_FULLFSYNC`s overlap only partly here. Decision 13's per-log flush goroutines give
the overlap the device allows; this device allows little.

## Linux, a virtual server, 2026-10-28

Debian 13, Linux 6.12.113, 4 vCPUs, ext4 on a virtual disk, Go 1.27.3, the same commit. Not
the baseline machine (the regression rule applies to the macOS table); recorded because the
cost of an fsync differs by two orders of magnitude between the two, and the read figures
show what a slower CPU does. Whether this disk's `fsync` reaches anything persistent is not
known.

| Benchmark | Median ns/op | Range (min–max) | Runs | Median B/op | Median allocs/op | Other |
|---|---|---|---|---|---|---|
| `BenchmarkAppendBuffered` | 674 | 391–2,294 | 6 | 2 | 0 | 195.64 MB/s |
| `BenchmarkAppendFlushEach` | 33,980 | 32,604–47,727 | 6 | 992 | 14 | 3.77 MB/s |
| `BenchmarkAppendBatchSync` | 185,179 | 149,353–260,957 | 6 | 1,130 | 14 | 359.96 MB/s |
| `BenchmarkReadStart` | 49,472 | 44,302–51,723 | 6 | 27,716 | 136 | — |
| `BenchmarkReadTail` | 73,837 | 71,440–77,327 | 6 | 27,094 | 129 | — |
| `BenchmarkReadSealed` | 81,817 | 75,023–89,110 | 6 | 27,398 | 136 | — |
| `BenchmarkScanAll` | 60,018,922 | 58,776,928–81,695,657 | 6 | 109,218 | 119 | 6,665,072 records/s |
| `BenchmarkReadWait` | 81,452 | 78,565–89,066 | 6 | 2,516 | 25 | — |
| `BenchmarkColdStart` | 27,046,105 | 25,016,013–28,115,953 | 6 | 7,420,998 | 363 | — |
| `BenchmarkParallelLogsFlush` | 104,440 | 92,562–113,155 | 6 | 4,433 | 66 | 4.90 MB/s |
| `BenchmarkReaderFollow` | 811,055 | 626,103–864,446 | 6 | 158,610 | 136 | — |
| `BenchmarkDrainBacklog` | 341,556,897 | 334,723,663–351,416,329 | 6 | 1,930,648 | 196 | 785.92 MB/s, 5,458,232 records/s |

**fsync is cheap here**: one record per flush costs about 34 µs (about 30 K durable records
per second for a lone producer) and a 512-record batch about 185 µs (about 2.7 M records per
second), against 10 ms per flush on the macOS build machine. `ReadWait` sees a record about
80 µs after `Append`.

**Reads are CPU-bound on this VM**: a 100-record read costs 49 µs at the start, 74 µs near
the tail and 82 µs in a sealed segment. The tail read is the slowest because the sparse index
anchors every 64 KiB, so a read near the tail decodes a few hundred frames past its anchor
before the first one it wants; on the faster build machine the same work costs 23 µs.
Narrowing the anchor stride is a measured follow-up for the audit sequence.

**Before decision 91**, the same reads on this machine cost 230 µs at the start, 73 µs near
the tail and 226 µs in a sealed segment: the fixed 1 MiB first chunk dominated every read of
a sealed segment. The tail read did not change, which is how the anchor-stride cost above
was told apart from the chunk cost.
