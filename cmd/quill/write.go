package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/netstar-labs/quill"
)

// openWriter opens the log writable for an offline tool: no background maintenance
// (decision 68), no rewrite of quill.policy and no ProtectedConsumers, so it never needs a
// consumer's lock (decision 99). A Config with no SegmentBytes
// takes SegmentBytes and MaxPayloadBytes from the policy when it parses, else the ceilings,
// so opening never seals an active segment the writer's size allows (decision 99). The
// caller closes it. A Config with no Perm takes the mode of the log's quill.tail (owner
// read and write kept), so a verb that rewrites the marker or creates a segment does not
// reset a group-readable log to 0600: Reader, register and Archive take their mode from the
// marker.
func (c *cli) openWriter(cfg quill.Config) (*quill.Log, int) {
	cfg.Dir = c.dir
	cfg.NoMaintenance = true
	if cfg.Perm == 0 {
		if info, err := os.Lstat(filepath.Join(c.dir, "quill.tail")); err == nil && info.Mode().IsRegular() {
			cfg.Perm = info.Mode().Perm() | 0o600
		}
	}
	cfg.ProtectedConsumers = nil                    // even one read from the policy: never take a consumer's lock
	if cfg.SegmentBytes == 0 && cfg.MaxBytes == 0 { // MaxBytes must be at least twice SegmentBytes
		cfg.SegmentBytes, cfg.MaxPayloadBytes = quill.MaxSegmentBytes, quill.MaxPayloadBytes
		if p, ok := quill.ReadPolicy(c.dir); ok {
			if p.SegmentBytes > 0 && p.SegmentBytes <= quill.MaxSegmentBytes {
				cfg.SegmentBytes = p.SegmentBytes
			}
			if p.MaxPayloadBytes > 0 && p.MaxPayloadBytes <= quill.MaxPayloadBytes {
				cfg.MaxPayloadBytes = p.MaxPayloadBytes
			}
		}
	}
	l, err := quill.Open(cfg)
	if err != nil {
		if errors.Is(err, quill.ErrLocked) {
			return nil, c.errorf("%v (a writer holds the log; this verb needs its lock)", err)
		}
		return nil, c.errorf("%v", err)
	}
	return l, exitOK
}

// withWriter opens the log writable, runs fn, and closes the log, discarding Close's
// error; cmdAppend and cmdRecover report it and so use openWriter directly.
func (c *cli) withWriter(cfg quill.Config, fn func(l *quill.Log) int) int {
	l, code := c.openWriter(cfg)
	if code != exitOK {
		return code
	}
	defer l.Close()
	return fn(l)
}

// cmdAppend closes the log itself rather than deferring: Close is the only flush that
// reports an error when -sync=false, so its failure must reach the exit code (decision 100).
func cmdAppend(c *cli, args []string) int {
	l, code := c.openWriter(quill.Config{Retention: -1})
	if code != exitOK {
		return code
	}
	last, durable, code := c.appendRecords(l, args)
	cerr := l.Close()
	if code != exitOK {
		return code
	}
	return c.finishAppend(last, durable, cerr)
}

// finishAppend turns Close's result into the verb's output and exit code. A Close failure
// exits 1; when Sync had already made the records durable the sequence is still printed to
// stdout, beside the error on stderr, because an error means unknown, not lost.
func (c *cli) finishAppend(last uint64, durable bool, cerr error) int {
	if cerr != nil {
		if durable {
			fmt.Fprintf(c.out, "%d\n", last)
		}
		return c.errorf("%v", cerr)
	}
	fmt.Fprintf(c.out, "%d\n", last)
	return exitOK
}

// appendRecords appends the arguments, or each line of stdin, as one batch (so MaxBatchBytes applies,
// decision 46), and syncs when -sync is set.
// durable reports that Sync succeeded.
func (c *cli) appendRecords(l *quill.Log, args []string) (last uint64, durable bool, code int) {
	var payloads [][]byte
	if len(args) > 0 {
		for _, a := range args {
			payloads = append(payloads, []byte(a))
		}
	} else {
		sc := bufio.NewScanner(c.in)
		sc.Buffer(make([]byte, 1<<20), quill.MaxPayloadBytes)
		// The library's check runs only once the batch is built, so apply it as the lines arrive: a
		// batch of more than one record is refused when the sum of len+34 passes MaxBatchBytes,
		// which also bounds the line count. The CLI opens the log with the default.
		size := 0
		for sc.Scan() {
			size += len(sc.Bytes()) + 34
			if len(payloads) > 0 && size > quill.DefaultMaxBatchBytes {
				return 0, false, c.errorf("%v: stdin is a batch of more than %d bytes, MaxBatchBytes is %d",
					quill.ErrPayloadTooLarge, quill.DefaultMaxBatchBytes, quill.DefaultMaxBatchBytes)
			}
			payloads = append(payloads, append([]byte(nil), sc.Bytes()...))
		}
		if err := sc.Err(); err != nil {
			return 0, false, c.errorf("stdin: %v", err)
		}
	}
	if len(payloads) == 0 {
		return 0, false, c.usage("append: nothing to append")
	}
	first, err := l.AppendBatch(payloads)
	if err != nil {
		return 0, false, c.errorf("%v", err)
	}
	last = first + uint64(len(payloads)) - 1
	if c.sync {
		if err := l.Sync(context.Background(), last); err != nil {
			return 0, false, c.errorf("%v", err)
		}
		durable = true
	}
	return last, durable, exitOK
}

func cmdPrune(c *cli, args []string) int {
	cfg, ok := quill.ReadPolicy(c.dir)
	explicit := c.retention != "" || c.maxBytes != "" || c.minFree != "" || c.maxRecords != "" || c.ttl != ""
	if !ok && !explicit {
		return c.errorf("quill.policy cannot be read; pass the retention flags explicitly")
	}
	if !ok {
		cfg = quill.Config{Retention: -1}
	}
	var err error
	if c.retention != "" {
		if cfg.Retention, err = time.ParseDuration(c.retention); err != nil {
			return c.usage("-retention: %v", err)
		}
	}
	for _, f := range []struct {
		s   string
		dst *int64
	}{{c.maxBytes, &cfg.MaxBytes}, {c.minFree, &cfg.MinFreeBytes}, {c.maxRecords, &cfg.MaxRecords}} {
		if f.s != "" {
			n, err := parseDecimal(f.s)
			if err != nil || n > math.MaxInt64 {
				return c.usage("%q is not a plain non-negative integer", f.s)
			}
			*f.dst = int64(n)
		}
	}
	if c.ttl != "" {
		if cfg.ConsumerTTL, err = time.ParseDuration(c.ttl); err != nil {
			return c.usage("-consumer-ttl: %v", err)
		}
	}
	return c.withWriter(cfg, func(l *quill.Log) int {
		before, _ := l.Stat()
		if err := l.Prune(c.force); err != nil {
			return c.errorf("%v", err)
		}
		after, err := l.Stat()
		if err != nil {
			return c.errorf("%v", err)
		}
		fmt.Fprintf(c.out, "segments %d -> %d, bytes %d -> %d, first %d -> %d\n", before.Segments, after.Segments, before.Bytes, after.Bytes, before.First, after.First)
		return exitOK
	})
}

func cmdRoll(c *cli, args []string) int {
	return c.withWriter(quill.Config{Retention: -1}, func(l *quill.Log) int {
		if err := l.Roll(); err != nil {
			return c.errorf("%v", err)
		}
		return exitOK
	})
}

func cmdRecover(c *cli, args []string) int {
	l, code := c.openWriter(quill.Config{Retention: -1})
	if code != exitOK {
		return code
	}
	st, _ := l.Stat()
	if err := l.Close(); err != nil {
		return c.errorf("%v", err)
	}
	fmt.Fprintf(c.out, "recovered: head %d, first %d, %d segments\n", st.Head, st.First, st.Segments)
	for _, h := range st.Holes {
		fmt.Fprintf(c.out, "hole %d-%d\n", h.From, h.To)
	}
	return exitOK
}
