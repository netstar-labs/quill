package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/netstar-labs/quill"
)

// withReader opens the log read-only, runs fn, and closes the reader. A verb checks its
// usage before calling it, so a bad flag still exits 2 when -dir cannot be opened.
func (c *cli) withReader(fn func(r *quill.Reader) int) int {
	r, err := quill.OpenReader(c.dir, quill.ReaderOptions{})
	if err != nil {
		return c.errorf("%v", err)
	}
	defer r.Close()
	return fn(r)
}

// cmdLs lists the logs under a parent directory. A directory that is not a log (ErrNoLog) is
// skipped; a log that cannot be opened or read gets an "error:" row, the rest are still listed,
// and the exit code is 1, as stat and check give for the same log (decision 53, A5-44).
func cmdLs(c *cli, args []string) int {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return c.errorf("%v", err)
	}
	w := tabwriter.NewWriter(c.out, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "LOG\tHEAD\tRECORDS\tSEGMENTS\tBYTES\tRETENTION\tMAX-BYTES\tMIN-FREE\tMAX-RECORDS\tNOTE")
	found, failed := 0, 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(c.dir, e.Name())
		r, err := quill.OpenReader(dir, quill.ReaderOptions{})
		if errors.Is(err, quill.ErrNoLog) {
			continue
		}
		var st quill.Stats
		if err == nil {
			st, err = r.Stat()
			_ = r.Close()
		}
		if err != nil {
			failed++
			fmt.Fprintf(w, "%s\t\t\t\t\t\t\t\t\terror: %v\n", e.Name(), err)
			continue
		}
		found++
		note := ""
		ret, mb, mf, mr := "?", "?", "?", "?"
		if p, ok := quill.ReadPolicy(dir); ok {
			ret, mb, mf, mr = p.Retention.String(), fmt.Sprint(p.MaxBytes), fmt.Sprint(p.MinFreeBytes), fmt.Sprint(p.MaxRecords)
			if p.MaxBytes == 0 && p.MinFreeBytes == 0 {
				note = "UNBOUNDED"
			}
		} else {
			note = "policy unreadable"
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\n", e.Name(), st.Head, st.Records, st.Segments, st.Bytes, ret, mb, mf, mr, note)
	}
	_ = w.Flush()
	if found+failed == 0 {
		fmt.Fprintf(c.err, "quill: no logs under %s\n", c.dir)
	}
	if failed > 0 {
		return exitError
	}
	return exitOK
}

func cmdStat(c *cli, args []string) int {
	return c.withReader(func(r *quill.Reader) int {
		st, err := r.Stat()
		if err != nil {
			return c.errorf("%v", err)
		}
		if c.json {
			// A hole is written through a local tagged type: quill.Hole has no tags, so it would print
			// as {"From":..,"To":..}, and a nil list as null (stage 2 audit A1-57).
			type jsonHole struct {
				From uint64 `json:"from"`
				To   uint64 `json:"to"`
			}
			holes := make([]jsonHole, 0, len(st.Holes))
			for _, h := range st.Holes {
				holes = append(holes, jsonHole{h.From, h.To})
			}
			return printJSON(c, map[string]any{"head": st.Head, "first": st.First, "segments": st.Segments, "bytes": st.Bytes,
				"records": st.Records, "holes": holes, "marker_age_seconds": st.MarkerAge.Seconds()})
		}
		fmt.Fprintf(c.out, "head %d\nfirst %d\nrecords %d\nsegments %d\nbytes %d\nmarker age %s\n", st.Head, st.First, st.Records, st.Segments, st.Bytes, st.MarkerAge.Round(time.Millisecond))
		for _, h := range st.Holes {
			fmt.Fprintf(c.out, "hole %d-%d\n", h.From, h.To)
		}
		return exitOK
	})
}

// start resolves -from and -since to the first sequence to print.
func (c *cli) start(r *quill.Reader) (uint64, int) {
	if c.since != "" {
		t, err := parseTime(c.since, time.Now())
		if err != nil {
			return 0, c.usage("-since: %v", err)
		}
		off, err := r.OffsetForTime(t)
		if err != nil {
			return 0, c.errorf("%v", err)
		}
		return off, exitOK
	}
	if c.from > 0 {
		return c.from, exitOK
	}
	st, err := r.Stat()
	if err != nil {
		return 0, c.errorf("%v", err)
	}
	return beginning(st), exitOK
}

// beginning is where a read of the whole log starts: the first retained sequence, or sequence 1
// when a damaged oldest segment left a hole below it, which is loss, not expiry, so the read
// meets the gap instead of starting past it (decision 38).
func beginning(st quill.Stats) uint64 {
	if len(st.Holes) > 0 && st.Holes[0].From == 1 {
		return 1
	}
	return st.First
}

func cmdRead(c *cli, args []string) int { return printFrom(c) }

func printFrom(c *cli) int {
	return c.withReader(func(r *quill.Reader) int {
		from, code := c.start(r)
		if code != exitOK {
			return code
		}
		return printRecords(c, r, from, false)
	})
}

// printRecords prints from `from`, stopping at the limit, a gap, or the end (or never with
// follow, until interrupted).
func printRecords(c *cli, r *quill.Reader, from uint64, follow bool) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	p := newPrinter(c.out, c.json, c.b64)
	defer p.flush()
	var seq func(func(quill.Record, error) bool)
	if follow {
		seq = r.Follow(ctx, from)
	} else {
		seq = r.Records(ctx, from)
	}
	var n uint64
	for rec, err := range seq {
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return exitOK
			}
			_ = p.flush()
			return c.errorf("%v", err)
		}
		if err := p.print(rec); err != nil {
			return c.errorf("%v", err)
		}
		if follow {
			// The next step may block for as long as the log is quiet, so what has been
			// printed must not wait in the buffer for it. A follow that cannot write stops.
			if err := p.flush(); err != nil {
				return c.errorf("%v", err)
			}
		}
		n++
		if c.limit > 0 && n >= c.limit {
			return exitOK
		}
	}
	return exitOK
}

func cmdTail(c *cli, args []string) int {
	return c.withReader(func(r *quill.Reader) int {
		var from uint64
		if c.since != "" || c.from > 0 {
			var code int
			from, code = c.start(r)
			if code != exitOK {
				return code
			}
		} else {
			st, err := r.Stat()
			if err != nil {
				return c.errorf("%v", err)
			}
			from = max(st.Head+1-min(c.n, st.Head), beginning(st))
		}
		return printRecords(c, r, from, c.follow)
	})
}

func cmdCheck(c *cli, args []string) int {
	return c.withReader(func(r *quill.Reader) int {
		res, err := r.Check()
		if err != nil {
			return c.errorf("%v", err)
		}
		status := "HEALTHY"
		if !res.Healthy {
			status = "UNHEALTHY"
		}
		if c.json {
			problems := []map[string]any{}
			for _, p := range res.Problems {
				problems = append(problems, map[string]any{"kind": p.Kind, "path": p.Path, "offset": p.Offset, "intact": p.Intact, "detail": p.Detail, "fatal": p.Fatal})
			}
			if code := printJSON(c, map[string]any{"status": status, "unbounded": res.Unbounded, "problems": problems}); code != exitOK {
				return code
			}
		} else {
			fmt.Fprintln(c.out, status)
			if res.Unbounded {
				fmt.Fprintln(c.out, "UNBOUNDED: neither MaxBytes nor MinFreeBytes is set; a fast writer can fill the disk")
			}
			for _, p := range res.Problems {
				fmt.Fprintln(c.out, p.String())
			}
		}
		if !res.Healthy {
			return exitUnhealthy
		}
		return exitOK
	})
}

// parseDecimal parses a plain base-10 unsigned integer: digits only, so "1e6", "1M",
// "0x10", "-1", "+1", "1_000", "" and trailing junk are all refused.
func parseDecimal(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// decimalFlag is a flag.Value for a plain base-10 count: -from, -limit and -n take the same
// digits-only rule as every other number on the command line, where flag.Uint64Var would read
// 010 as octal and 0x10 or 1_0 as hex and a separated number.
type decimalFlag uint64

func (d *decimalFlag) Set(s string) error {
	n, err := parseDecimal(s)
	if err != nil {
		return errors.New("not a plain decimal integer")
	}
	*d = decimalFlag(n)
	return nil
}

func (d *decimalFlag) String() string { return strconv.FormatUint(uint64(*d), 10) }

// parseTime accepts RFC 3339, "@<unix seconds>", or a duration meaning that long ago.
func parseTime(s string, now time.Time) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if strings.HasPrefix(s, "@") {
		unix, err := parseDecimal(s[1:])
		if err != nil || unix > math.MaxInt64 {
			return time.Time{}, fmt.Errorf("%q is not @ and a plain non-negative integer of unix seconds", s)
		}
		return time.Unix(int64(unix), 0), nil
	}
	if d, err := time.ParseDuration(s); err == nil && d >= 0 {
		return now.Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("%q is not RFC 3339, @unix, or a duration", s)
}
