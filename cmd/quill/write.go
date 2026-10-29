package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/netstar-labs/quill"
)

// writer opens the log writable for an offline tool: no background maintenance (decision 68).
func (c *cli) writer(cfg quill.Config) (*quill.Log, int) {
	cfg.Dir = c.dir
	cfg.NoMaintenance = true
	l, err := quill.Open(cfg)
	if err != nil {
		if errors.Is(err, quill.ErrLocked) {
			return nil, c.errorf("%v (a writer holds the log; this verb needs its lock)", err)
		}
		return nil, c.errorf("%v", err)
	}
	return l, exitOK
}

func cmdAppend(c *cli, args []string) int {
	l, code := c.writer(quill.Config{Retention: -1})
	if code != exitOK {
		return code
	}
	defer l.Close()
	var payloads [][]byte
	if len(args) > 0 {
		for _, a := range args {
			payloads = append(payloads, []byte(a))
		}
	} else {
		sc := bufio.NewScanner(c.in)
		sc.Buffer(make([]byte, 1<<20), quill.MaxPayloadBytes)
		for sc.Scan() {
			payloads = append(payloads, append([]byte(nil), sc.Bytes()...))
		}
		if err := sc.Err(); err != nil {
			return c.errorf("stdin: %v", err)
		}
	}
	if len(payloads) == 0 {
		return c.usage("append: nothing to append")
	}
	first, err := l.AppendBatch(payloads)
	if err != nil {
		return c.errorf("%v", err)
	}
	last := first + uint64(len(payloads)) - 1
	if c.sync {
		if err := l.Sync(context.Background(), last); err != nil {
			return c.errorf("%v", err)
		}
	}
	fmt.Fprintf(c.out, "%d\n", last)
	return exitOK
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
			if _, err := fmt.Sscanf(f.s, "%d", f.dst); err != nil {
				return c.usage("%q: %v", f.s, err)
			}
		}
	}
	if c.ttl != "" {
		if cfg.ConsumerTTL, err = time.ParseDuration(c.ttl); err != nil {
			return c.usage("-consumer-ttl: %v", err)
		}
	}
	l, code := c.writer(cfg)
	if code != exitOK {
		return code
	}
	defer l.Close()
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
}

func cmdRoll(c *cli, args []string) int {
	l, code := c.writer(quill.Config{Retention: -1})
	if code != exitOK {
		return code
	}
	defer l.Close()
	if err := l.Roll(); err != nil {
		return c.errorf("%v", err)
	}
	return exitOK
}

func cmdRecover(c *cli, args []string) int {
	l, code := c.writer(quill.Config{Retention: -1})
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
