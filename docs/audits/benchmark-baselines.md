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

Debian 13, Linux 6.12.113, 4 vCPUs, 8 GiB, ext4 (`data=ordered`) on a virtual disk, Go
1.27.3, the same commit, with the temporary directory on the disk (the server's `/tmp` is a
tmpfs, and a first run there measured memory, not a disk; those figures are discarded). Not
the baseline machine (the regression rule applies to the macOS table); recorded because the
cost of an fsync differs by an order of magnitude between the two and because this machine
can hold a backlog larger than its memory. Whether the virtual disk's `fsync` reaches anything
persistent is not known.

| Benchmark | Median ns/op | Range (min–max) | Runs | Median B/op | Median allocs/op | Other |
|---|---|---|---|---|---|---|
| `BenchmarkAppendBuffered` | 2,842 | 2,645–3,434 | 6 | 3 | 0 | 45.04 MB/s |
| `BenchmarkAppendFlushEach` | 929,308 | 898,101–964,887 | 6 | 1,089 | 14 | 0.14 MB/s |
| `BenchmarkAppendBatchSync` | 1,215,706 | 1,200,227–3,506,327 | 6 | 1,489 | 14 | 53.92 MB/s |
| `BenchmarkReadStart` | 44,972 | 41,128–46,444 | 6 | 27,700 | 136 | — |
| `BenchmarkReadTail` | 66,596 | 64,344–74,104 | 6 | 26,766 | 129 | — |
| `BenchmarkReadSealed` | 79,549 | 73,205–152,317 | 6 | 27,257 | 136 | — |
| `BenchmarkScanAll` | 60,129,846 | 57,447,278–63,601,067 | 6 | 109,405 | 119 | 6,657,584 records/s |
| `BenchmarkReadWait` | 1,022,843 | 945,475–1,155,069 | 6 | 6,674 | 25 | — |
| `BenchmarkColdStart` | 27,531,304 | 26,510,346–28,207,397 | 6 | 7,422,030 | 363 | — |
| `BenchmarkParallelLogsFlush` | 1,374,841 | 1,285,289–1,421,308 | 6 | 4,820 | 66 | 0.37 MB/s |
| `BenchmarkReaderFollow` | 1,552,604 | 1,483,489–1,600,729 | 6 | 179,436 | 239 | — |
| `BenchmarkDrainBacklog` | 362,929,796 | 339,691,184–409,661,358 | 6 | 1,933,728 | 198 | 739.66 MB/s, 5,140,637 records/s |

**fsync costs about 1 ms here**: one record per flush about 0.93 ms (about 1,100 durable
records per second for a lone producer) and a 512-record batch about 1.2 ms (about 420 K
records per second), against 10 ms per flush on the macOS build machine. `ReadWait` sees a
record about 1 ms after `Append`. Four logs flushing in parallel take 1.4 ms per round
against 0.93 ms for one: the fsyncs overlap well on this device, which is what decision 13's
per-log flush goroutines are for.

**Reads are CPU-bound on this VM**: a 100-record read costs 45 µs at the start, 67 µs near
the tail and 80 µs in a sealed segment. The tail read is the slowest because the sparse index
anchors every 64 KiB, so a read near the tail decodes a few hundred frames past its anchor
before the first one it wants; on the faster build machine the same work costs 23 µs.
Narrowing the anchor stride is a measured follow-up for the audit sequence. Before decision 91
the start and sealed reads cost about 230 µs on this machine; the tail read did not change,
which is how the anchor-stride cost was told apart from the chunk cost.

### A backlog larger than memory (decision 15)

`BenchmarkDrainBacklog` with `QUILL_BENCH_BACKLOG_BYTES` set to 12 GiB on the 8 GiB server:
about 89 million records written, then drained by a `Reader` with nothing of it left in the
page cache.

| `BenchmarkDrainBacklog` | 39,105,102,614 | 38,890,552,202–41,496,919,403 | 6 | 246,715,088 | 10,358 | 329.50 MB/s, 2,288,182 records/s |

The drain runs at about 330 MB/s, 2.3 M records per second. For context: the same `Reader`
drains the warm 256 MiB backlog at about 740 MB/s, and the disk reads a 2 GiB file
sequentially at about 1.1 GB/s after the caches are dropped (`dd`, a file of zeros on a
virtual disk, so an upper bound). The cold drain is slower than both, and close to their sum:
12 GiB at 1.1 GB/s is about 11 s of reading and at 740 MB/s about 17 s of decoding, against
39 s measured. Reading and decoding do not overlap: each chunk is read with `pread` and then
decoded before the next is read. So decision 15's assumption ("sequential `pread` with large
chunks should be limited by the disk either way") holds only in the sense that `pread` is not
the limit; a drain from disk pays the disk and the decoder in series. Overlapping them (a
read-ahead of the next chunk) is a measured candidate for the audit sequence's optimisation
pass, with this table as the before.

## Allocation pass (A1 follow-up)

One row per change, on the macOS build machine, three runs each of
`GOWORK=off go test -run '^$' -bench '^<Name>$' -benchmem -count 3 .`, before and after.
Wall time is dominated by `F_FULLFSYNC` and did not move beyond noise.

| Change | Benchmark | Before B/op | Before allocs/op | After B/op | After allocs/op |
|---|---|---|---|---|---|
| write-1: `quill.tail` encoded by `appendTail` into a reused `Log.tailBuf` instead of `json.Marshal` | `BenchmarkAppendFlushEach` | 1,291–1,295 | 14 | 1,177–1,182 | 11 |
| write-1 (same change) | `BenchmarkParallelLogsFlush` | 5,727–6,108 | 66–68 | 5,317–5,442 | 55 |
| write-2: `fsx.OS.Rename` calls `syscall.Rename` (retrying `EINTR`, wrapping in `*os.LinkError`) instead of `os.Rename`, which `Lstat`s the destination first | `BenchmarkAppendFlushEach` | 1,181–1,184 | 11 | 860–861 | 9 |
| write-2 (same change) | `BenchmarkParallelLogsFlush` | 5,296–5,407 | 55 | 4,042–4,153 | 47 |
| write-3: `atomicfile.WriteFrom` builds the temp name with `hex.Encode` into a stack array and one concatenation (`tempName`) instead of `filepath.Join` plus `hex.EncodeToString` | `BenchmarkAppendFlushEach` | 857 | 9 | 833–836 | 8 |
| write-3 (same change) | `BenchmarkParallelLogsFlush` | 3,977–4,213 | 47 | 3,869–4,280 | 43–44 |
