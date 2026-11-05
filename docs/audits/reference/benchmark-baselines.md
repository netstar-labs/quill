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

## macOS, Apple silicon (the build machine), 2026-10-29

Darwin 25.6.0, arm64, 10 cores, APFS on the internal SSD, Go 1.27.3, quill at the commit that
tags v0.1.0, re-recorded after the allocation pass (the tables before it are at commit 45d2d2b;
no median moved by more than noise except downward: a flush allocates 6 times instead of 14, a
100-record read 113 instead of 136, a `Reader` poll 162 instead of 543, a full scan 47 instead
of 119).

| Benchmark | Median ns/op | Range (min–max) | Runs | Median B/op | Median allocs/op | Other |
|---|---|---|---|---|---|---|
| `BenchmarkAppendBuffered` | 22,146 | 21,899–22,789 | 6 | 6 | 0 | 5.78 MB/s |
| `BenchmarkAppendFlushEach` | 9,793,069 | 9,615,704–10,242,545 | 6 | 610 | 6 | 0.01 MB/s |
| `BenchmarkAppendBatchSync` | 10,330,171 | 10,169,191–10,602,169 | 6 | 3,210 | 6 | 6.35 MB/s |
| `BenchmarkReadStart` | 24,592 | 24,446–25,546 | 6 | 18,836 | 113 | — |
| `BenchmarkReadTail` | 20,696 | 20,336–21,004 | 6 | 18,059 | 106 | — |
| `BenchmarkReaderReadActive` | 1,966,076 | 1,921,382–2,128,119 | 6 | 10,364 | 72 | — |
| `BenchmarkReadSealed` | 36,192 | 35,639–36,724 | 6 | 18,624 | 113 | — |
| `BenchmarkScanAll` | 18,896,942 | 18,646,579–19,513,642 | 6 | 22,379 | 47 | 21,271,603 records/s |
| `BenchmarkReadWait` | 10,201,346 | 10,028,294–10,417,794 | 6 | 18,894 | 16 | — |
| `BenchmarkColdStart` | 22,655,105 | 22,549,493–22,950,587 | 6 | 7,417,626 | 327 | — |
| `BenchmarkParallelLogsFlush` | 23,866,388 | 20,759,760–25,975,165 | 6 | 2,918 | 35 | 0.02 MB/s |
| `BenchmarkReaderFollow` | 10,414,117 | 9,893,177–10,795,930 | 6 | 72,470 | 162 | — |
| `BenchmarkDrainBacklog` | 112,877,826 | 110,578,075–114,572,958 | 6 | 522,350 | 88 | 2,378.11 MB/s, 16,599,194 records/s |
| `BenchmarkReadStartThousandSegments` | 2,250 | 2,228–2,305 | 6 | 11,300 | 30 | — |
| `BenchmarkReaderReadStartThousandSegments` | 163,128 | 162,244–165,086 | 6 | 360,952 | 2,060 | — |

**The cost of the marker.** A flush is two `F_FULLFSYNC`s on macOS: the segment's data and
the marker's temp file. `AppendFlushEach` (one record per flush) is about 10 ms per record,
about 100 durable records per second for a lone producer; `AppendBatchSync` (512 records per
flush) is the same 10 ms per flush, about 50 K durable records per second. The second fsync
is close to half of that time on this disk; it is what a lock-free reader in another process
costs (decision 8). `ReadWait` and `ReaderFollow` are dominated by the same flush: a waiting
in-process reader sees a record about 10 ms after `Append`, an external reader polling at
1 ms about 10.4 ms.

**Reads.** A 100-record read costs about 25 µs at the start of the log, 21 µs near the tail
and 36 µs at a random position in a sealed segment; the difference is opening a sealed
segment per read (decision 15) and the seek within it. The sparse index and the
first-chunk size (decision 91) keep the cost independent of the backlog (T19).

**Scans.** A full in-process `ScanAll` decodes about 21.3 M records per second warm; a
`Reader` draining a 256 MiB backlog decodes about 16.6 M records per second, about 2.4 GB/s.
The backlog fits in memory; the larger-than-memory measurement of decision 15 was run on
the Linux server (below).

**Cold start** of a 400 K-record, 58 MiB log, every byte read and CRC-checked: about 23 ms.

**Four logs flushing in parallel** take about 24 ms per round on this disk, not the 10 ms of
one: `F_FULLFSYNC`s overlap only partly here. Decision 13's per-log flush goroutines give
the overlap the device allows; this device allows little.

## Linux, a virtual server, 2026-10-29

Debian 13, Linux 6.12.113, 4 vCPUs, 8 GiB, ext4 (`data=ordered`) on a virtual disk, Go
1.27.3, the same commit (re-recorded after the allocation pass; the tables before it are at commit 45d2d2b), with the temporary directory on the disk (the server's `/tmp` is a
tmpfs, and a first run there measured memory, not a disk; those figures are discarded). Not
the baseline machine (the regression rule applies to the macOS table); recorded because the
cost of an fsync differs by an order of magnitude between the two and because this machine
can hold a backlog larger than its memory. Whether the virtual disk's `fsync` reaches anything
persistent is not known.

| Benchmark | Median ns/op | Range (min–max) | Runs | Median B/op | Median allocs/op | Other |
|---|---|---|---|---|---|---|
| `BenchmarkAppendBuffered` | 2,944 | 2,810–3,077 | 6 | 1 | 0 | 43.48 MB/s |
| `BenchmarkAppendFlushEach` | 994,156 | 950,052–1,044,095 | 6 | 472 | 6 | 0.13 MB/s |
| `BenchmarkAppendBatchSync` | 1,350,909 | 1,277,484–4,047,745 | 6 | 880 | 6 | 48.52 MB/s |
| `BenchmarkReadStart` | 41,656 | 38,466–48,065 | 6 | 18,239 | 113 | — |
| `BenchmarkReadTail` | 68,017 | 63,754–127,662 | 6 | 17,404 | 106 | — |
| `BenchmarkReaderReadActive` | 6,798,884 | 6,630,854–7,099,823 | 6 | 12,984 | 70 | — |
| `BenchmarkReadSealed` | 76,863 | 73,147–132,146 | 6 | 18,208 | 113 | — |
| `BenchmarkScanAll` | 59,850,301 | 58,908,592–70,607,948 | 6 | 59,743 | 48 | 6,688,301 records/s |
| `BenchmarkReadWait` | 1,031,078 | 1,024,260–1,082,676 | 6 | 6,920 | 16 | — |
| `BenchmarkColdStart` | 28,056,283 | 26,577,688–28,635,837 | 6 | 7,419,016 | 324 | — |
| `BenchmarkParallelLogsFlush` | 1,336,699 | 1,275,376–1,404,999 | 6 | 2,355 | 34 | 0.39 MB/s |
| `BenchmarkReaderFollow` | 1,407,812 | 1,387,655–1,475,206 | 6 | 17,009 | 108 | — |
| `BenchmarkDrainBacklog` | 355,361,852 | 348,752,705–369,701,060 | 6 | 1,552,917 | 113 | 755.48 MB/s, 5,250,530 records/s |
| `BenchmarkReadStartThousandSegments` | 5,742 | 5,498–6,526 | 6 | 11,178 | 30 | — |
| `BenchmarkReaderReadStartThousandSegments` | 416,622 | 407,842–451,619 | 6 | 352,538 | 2,066 | — |

**fsync costs about 1 ms here**: one record per flush about 1.0 ms (about 1,000 durable
records per second for a lone producer) and a 512-record batch about 1.35 ms (about 380 K
records per second), against 10 ms per flush on the macOS build machine. `ReadWait` sees a
record about 1 ms after `Append`. Four logs flushing in parallel take 1.3 ms per round
against 1.0 ms for one: the fsyncs overlap well on this device, which is what decision 13's
per-log flush goroutines are for.

**Reads are CPU-bound on this VM**: a 100-record read costs 42 µs at the start, 68 µs near
the tail and 77 µs in a sealed segment. The tail read is the slowest because the sparse index
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
| write-4: `Log.tailPath` joined once in `Open` instead of `filepath.Join` in `publishLocked` on every flush | `BenchmarkAppendFlushEach` | 833–837 | 8 | 721–725 | 7 |
| write-4 (same change) | `BenchmarkParallelLogsFlush` | 3,845–4,113 | 43 | 3,486–3,626 | 39 |
| write-5: the `Log.flushed` broadcast channel is made by the first waiter (`waitChanLocked`, called by `Sync` and `ReadWait`) and closed and set to nil by `broadcastLocked`, instead of a new channel made after every flush attempt | `BenchmarkAppendFlushEach` | 721–725 | 7 | 608–611 | 6 |
| write-5 (same change; a waiter is present each flush, so the channel is still made, and B/op is dominated by the batch buffer) | `BenchmarkAppendBatchSync` | 3,225–3,290 | 7 | 3,176–3,615 | 6 |
| write-5 (same change; B/op varies run to run with the reader's buffers) | `BenchmarkReadWait` | 10,394–27,506 | 18–19 | 19,269–36,712 | 17 |
| read-7: `Log.view` reuses the sealed `segRef`s cached in `Log.sealedRefs` (decision 103) instead of building one `segRef` and one `open` closure per sealed segment on every call | `BenchmarkReadStart` | 27,971–28,278 | 136 | 26,976–27,011 | 121 |
| read-7 (same change) | `BenchmarkReadTail` | 27,655–27,787 | 129 | 26,162–26,253 | 114 |
| read-7 (same change; 1,000 sealed segments on `fsx.Sim`, `Read(1, 1)`; ns/op 36,762–38,573 before, 2,171–2,436 after) | `BenchmarkReadStartThousandSegments` | 177,925–179,073 | 2,041 | 11,437–11,481 | 31 |
| read-9: `readFrom` sizes its result slice once, from the view, when the first record is delivered (`scanSource`'s `expect` hook, decision 104), instead of growing it by `append` from nil | `BenchmarkReadStart` | 26,935–27,323 | 121 | 18,532–18,678 | 114 |
| read-9 (same change; ns/op 26,484–27,413 before, 24,826–24,905 after) | `BenchmarkReadTail` | 26,064–26,460 | 114 | 17,956–18,229 | 107 |
| read-9 (same change; ns/op 38,948–40,402 before, 35,523–37,895 after) | `BenchmarkReadSealed` | 26,673–26,906 | 121 | 18,542–18,644 | 114 |
| reader-15: `decodeTail` takes `decodeTailFast` for the exact bytes `appendTail` writes (decision 105) instead of the strict JSON pipeline on every `quill.tail` read; non-canonical input still takes the strict path (ns/op 10.14-10.57 ms before, 10.23-10.72 ms after: the 1 ms poll sleep dominates; B/op varies with the number of polls per iteration) | `BenchmarkReaderFollow` | 186,609-203,517 | 522-537 | 95,556-193,948 | 164-167 |
| reader-15 (same change; `Log.Read` never reads `quill.tail`, so it is unchanged, as expected) | `BenchmarkReadTail` | 18,061-18,256 | 107 | 18,008-18,500 | 107 |
| reader-16: `Reader.view` decodes the active segment's first frame in a buffer taken from `Reader.bufPool` (4 KiB first read) instead of a fresh 64 KiB `make` on every call (new benchmark; `Read(head, 1)` on a 400 K-record active segment; ns/op 1,964,412-2,034,195 before, 1,927,021-1,933,573 after) | `BenchmarkReaderReadActive` | 84,384-87,635 | 95 | 15,291-16,987 | 94 |
| reader-17: `Reader.view` sizes `sealed`, `cands`, `rs` and `v.segs` up front, allocates the segment references as one slab, and sets `ensureIndex` only on a segment that has no index yet (a scanned one never calls it) instead of one `*segRef` and two closures per sealed segment (new benchmark; `Read(1, 1)` through a `Reader` on 1,000 sealed segments on `fsx.Sim`; ns/op 192,822-194,041 before, 159,593-160,334 after) | `BenchmarkReaderReadStartThousandSegments` | 542,491-551,953 | 3,099-3,100 | 361,639-365,970 | 2,061 |
| reader-17 (same change; one active segment and a few sealed ones; ns/op 1,911,198-1,932,657 before, 1,921,260-1,937,592 after, within noise) | `BenchmarkReaderReadActive` | 15,296-17,694 | 94 | 14,749-15,185 | 80 |
| reader-19: `printer.print` appends the record to a reused `printer.buf` (`strconv.AppendUint`, `time.AppendFormat`, `base64.AppendEncode`, `appendJSONString`, `appendQuoted`) and makes one `Write`, instead of `time.Format`, `jsonRecord`, `json.Marshal`, `strconv.Quote` and `fmt.Fprintf` (new benchmark in `cmd/quill`, run with `GOWORK=off go test -run '^$' -bench BenchmarkPrint -benchmem -count 3 ./cmd/quill`; output bytes are identical, checked by `TestPrintMatchesReference`; ns/op 445.8-457.9 before, 297.8-305.5 after) | `BenchmarkPrintJSONText` | 424 | 6 | 0 | 0 |
| reader-19 (same change; ns/op 383.6-393.0 before, 55.8-57.2 after) | `BenchmarkPrintJSONBin` | 304 | 6 | 0 | 0 |
| reader-19 (same change; ns/op 533.0-535.7 before, 155.9-156.5 after) | `BenchmarkPrintTextText` | 368 | 7 | 0 | 0 |
| reader-19 (same change; ns/op 161.0-161.3 before, 42.4-43.5 after) | `BenchmarkPrintTextBin` | 120 | 6 | 0 | 0 |
| reader-20: `scanSource`'s window sets `noIndex`, so `walk` no longer appends an anchor every 64 KiB (`indexStride`) into a `segInfo.index` that the scan path discards; `verifySegment`, `recoverSegment` and `verifyBounded` still build the index (ns/op 106.0-114.5 ms before, 112.4-113.6 ms after: the first run's spread covers it, so within noise; B/op varies with the pooled buffers) | `BenchmarkDrainBacklog` | 705,070-833,891 | 134-137 | 522,328-522,976 | 88-89 |
| reader-20 (same change; ns/op 18.80-19.34 ms before, 18.69-18.98 ms after) | `BenchmarkScanAll` | 71,412-72,017 | 104 | 22,093-22,984 | 47 |
| reader-20 (same change; ns/op 1,924,933-1,946,905 before, 1,935,879-2,045,463 after) | `BenchmarkReaderReadActive` | 15,097-15,197 | 80 | 9,081-13,353 | 72 |
