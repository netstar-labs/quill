package quill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// T24: the benchmark baselines (decision 21). Run six times on one machine and record the
// medians in docs/audits/benchmark-baselines.md:
//
//	GOWORK=off go test -run '^$' -bench . -benchmem -count 6 . | tee bench.txt
//
// Every benchmark uses a real temporary directory, 128-byte payloads, and the default
// configuration except where stated.

const benchPayload = 128

func benchLog(b *testing.B, mut func(*Config)) (*Log, string) {
	b.Helper()
	dir := filepath.Join(b.TempDir(), "bench")
	cfg := Config{Dir: dir, Retention: -1, FlushInterval: time.Hour}
	if mut != nil {
		mut(&cfg)
	}
	l, err := Open(cfg)
	if err != nil {
		b.Fatal(err)
	}
	return l, dir
}

func fillBench(b *testing.B, l *Log, n int) {
	b.Helper()
	p := make([]byte, benchPayload)
	batch := make([][]byte, 512)
	for i := range batch {
		batch[i] = p
	}
	for i := 0; i < n; i += len(batch) {
		if _, err := l.AppendBatch(batch); err != nil {
			b.Fatal(err)
		}
		if i%(len(batch)*16) == 0 {
			if err := l.Flush(); err != nil {
				b.Fatal(err)
			}
		}
	}
	if err := l.Flush(); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkAppendBuffered: Append with the background flush (here, a flush every 64 KiB).
func BenchmarkAppendBuffered(b *testing.B) {
	l, _ := benchLog(b, nil)
	defer l.Close()
	p := make([]byte, benchPayload)
	b.SetBytes(benchPayload)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := l.Append(p); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	_ = l.Flush()
}

// BenchmarkAppendFlushEach: one durable record per call: write, fsync, publish the marker.
func BenchmarkAppendFlushEach(b *testing.B) {
	l, _ := benchLog(b, nil)
	defer l.Close()
	p := make([]byte, benchPayload)
	b.SetBytes(benchPayload)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := l.Append(p); err != nil {
			b.Fatal(err)
		}
		if err := l.Flush(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAppendBatchSync: 512-record batches made durable with Sync: the throughput figure.
func BenchmarkAppendBatchSync(b *testing.B) {
	l, _ := benchLog(b, nil)
	defer l.Close()
	batch := make([][]byte, 512)
	for i := range batch {
		batch[i] = make([]byte, benchPayload)
	}
	b.SetBytes(benchPayload * 512)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		first, err := l.AppendBatch(batch)
		if err != nil {
			b.Fatal(err)
		}
		if err := l.Sync(context.Background(), first+511); err != nil {
			b.Fatal(err)
		}
	}
}

const benchRecords = 400_000

// BenchmarkReadStart: Read(limit 100) from the first record of a warm 400 K-record log.
func BenchmarkReadStart(b *testing.B) {
	l, _ := benchLog(b, nil)
	defer l.Close()
	fillBench(b, l, benchRecords)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if recs, err := l.Read(1, 100); err != nil || len(recs) != 100 {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadTail: Read(limit 100) just below the head.
func BenchmarkReadTail(b *testing.B) {
	l, _ := benchLog(b, nil)
	defer l.Close()
	fillBench(b, l, benchRecords)
	head := l.Head()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if recs, err := l.Read(head-100, 100); err != nil || len(recs) != 100 {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadSealed: Read(limit 100) at a random position in a sealed segment.
func BenchmarkReadSealed(b *testing.B) {
	l, _ := benchLog(b, nil)
	defer l.Close()
	fillBench(b, l, benchRecords)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		off := uint64(1 + (i*7919)%(benchRecords/2))
		if recs, err := l.Read(off, 100); err != nil || len(recs) != 100 {
			b.Fatal(err)
		}
	}
}

// BenchmarkScanAll: a full in-process scan, records per second.
func BenchmarkScanAll(b *testing.B) {
	l, _ := benchLog(b, nil)
	defer l.Close()
	fillBench(b, l, benchRecords)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := 0
		if err := l.ScanAll(func(Record) bool { n++; return true }); err != nil || n != benchRecords {
			b.Fatal(err, n)
		}
	}
	b.ReportMetric(float64(benchRecords)*float64(b.N)/b.Elapsed().Seconds(), "records/s")
}

// BenchmarkReadWait: the latency from a flush to a waiting in-process reader.
func BenchmarkReadWait(b *testing.B) {
	l, _ := benchLog(b, nil)
	defer l.Close()
	p := make([]byte, benchPayload)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		next := l.Head() + 1
		done := make(chan error, 1)
		go func() {
			_, err := l.ReadWait(context.Background(), next, 1)
			done <- err
		}()
		if _, err := l.Append(p); err != nil {
			b.Fatal(err)
		}
		if err := l.Flush(); err != nil {
			b.Fatal(err)
		}
		if err := <-done; err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkColdStart: Open of a 400 K-record log (every byte read and CRC-checked).
func BenchmarkColdStart(b *testing.B) {
	l, dir := benchLog(b, nil)
	fillBench(b, l, benchRecords)
	if err := l.Close(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l, err := Open(Config{Dir: dir, Retention: -1, FlushInterval: time.Hour})
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		_ = l.Close()
		b.StartTimer()
	}
}

// BenchmarkParallelLogsFlush: four logs each flushing one record per iteration, in parallel:
// per-log flush goroutines mean the fsyncs overlap (decision 13).
func BenchmarkParallelLogsFlush(b *testing.B) {
	const n = 4
	logs := make([]*Log, n)
	for i := range logs {
		l, err := Open(Config{Dir: filepath.Join(b.TempDir(), "log"+strconv.Itoa(i)), Retention: -1, FlushInterval: time.Hour})
		if err != nil {
			b.Fatal(err)
		}
		logs[i] = l
		defer l.Close()
	}
	p := make([]byte, benchPayload)
	b.SetBytes(benchPayload * n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		done := make(chan error, n)
		for _, l := range logs {
			go func(l *Log) {
				if _, err := l.Append(p); err != nil {
					done <- err
					return
				}
				done <- l.Flush()
			}(l)
		}
		for range logs {
			if err := <-done; err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkReaderFollow: a Reader in the same process following a writer: the latency from
// a flush to the poll that sees it, with a 1 ms poll interval.
func BenchmarkReaderFollow(b *testing.B) {
	l, dir := benchLog(b, nil)
	defer l.Close()
	r, err := OpenReader(dir, ReaderOptions{PollInterval: time.Millisecond})
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()
	p := make([]byte, benchPayload)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		next := l.Head() + 1
		done := make(chan error, 1)
		go func() {
			_, err := r.ReadWait(context.Background(), next, 1)
			done <- err
		}()
		if _, err := l.Append(p); err != nil {
			b.Fatal(err)
		}
		if err := l.Flush(); err != nil {
			b.Fatal(err)
		}
		if err := <-done; err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDrainBacklog: a Reader draining a backlog through pread, reported in records/s and
// bytes/s. The size comes from QUILL_BENCH_BACKLOG_BYTES (default 256 MiB); set it above the
// machine's memory for the real-disk measurement of decision 15.
func BenchmarkDrainBacklog(b *testing.B) {
	size := int64(256 << 20)
	if s := os.Getenv("QUILL_BENCH_BACKLOG_BYTES"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			b.Fatal(err)
		}
		size = v
	}
	l, dir := benchLog(b, func(c *Config) { c.SegmentBytes = 64 << 20 })
	defer l.Close()
	records := int(size / (benchPayload + 16))
	fillBench(b, l, records)
	r, err := OpenReader(dir, ReaderOptions{})
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()
	b.SetBytes(size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := 0
		if err := r.ScanAll(func(Record) bool { n++; return true }); err != nil {
			b.Fatal(err)
		}
		if n < records {
			b.Fatalf("%d records, want %d", n, records)
		}
	}
	b.ReportMetric(float64(records)*float64(b.N)/b.Elapsed().Seconds(), "records/s")
	_ = fmt.Sprint
}
