package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/netstar-labs/quill"
)

func (c *cli) reader() (*quill.Reader, int) {
	r, err := quill.OpenReader(c.dir, quill.ReaderOptions{})
	if err != nil {
		return nil, c.errorf("%v", err)
	}
	return r, exitOK
}

func cmdLs(c *cli, args []string) int {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return c.errorf("%v", err)
	}
	w := tabwriter.NewWriter(c.out, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "LOG\tHEAD\tRECORDS\tSEGMENTS\tBYTES\tRETENTION\tMAX-BYTES\tMIN-FREE\tMAX-RECORDS\tNOTE")
	found := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(c.dir, e.Name())
		r, err := quill.OpenReader(dir, quill.ReaderOptions{})
		if err != nil {
			continue
		}
		st, err := r.Stat()
		_ = r.Close()
		if err != nil {
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
	if found == 0 {
		fmt.Fprintf(c.err, "quill: no logs under %s\n", c.dir)
	}
	return exitOK
}

func cmdStat(c *cli, args []string) int {
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
	st, err := r.Stat()
	if err != nil {
		return c.errorf("%v", err)
	}
	if c.json {
		return printJSON(c, map[string]any{"head": st.Head, "first": st.First, "segments": st.Segments, "bytes": st.Bytes,
			"records": st.Records, "holes": st.Holes, "marker_age_seconds": st.MarkerAge.Seconds()})
	}
	fmt.Fprintf(c.out, "head %d\nfirst %d\nrecords %d\nsegments %d\nbytes %d\nmarker age %s\n", st.Head, st.First, st.Records, st.Segments, st.Bytes, st.MarkerAge.Round(time.Millisecond))
	for _, h := range st.Holes {
		fmt.Fprintf(c.out, "hole %d-%d\n", h.From, h.To)
	}
	return exitOK
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
	return st.First, exitOK
}

func cmdRead(c *cli, args []string) int { return printFrom(c) }

func printFrom(c *cli) int {
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
	from, code := c.start(r)
	if code != exitOK {
		return code
	}
	return printRecords(c, r, from, false)
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
			p.flush()
			return c.errorf("%v", err)
		}
		if err := p.print(rec); err != nil {
			return c.errorf("%v", err)
		}
		n++
		if c.limit > 0 && n >= c.limit {
			return exitOK
		}
	}
	return exitOK
}

func cmdTail(c *cli, args []string) int {
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
	var from uint64
	if c.since != "" || c.from > 0 {
		from, code = c.start(r)
		if code != exitOK {
			return code
		}
	} else {
		st, err := r.Stat()
		if err != nil {
			return c.errorf("%v", err)
		}
		from = max(st.Head+1-min(c.n, st.Head), st.First)
	}
	return printRecords(c, r, from, c.follow)
}

func cmdCheck(c *cli, args []string) int {
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
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
}

// parseTime accepts RFC 3339, "@<unix seconds>", or a duration meaning that long ago.
func parseTime(s string, now time.Time) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if strings.HasPrefix(s, "@") {
		var unix int64
		if _, err := fmt.Sscanf(s[1:], "%d", &unix); err == nil {
			return time.Unix(unix, 0), nil
		}
	}
	if d, err := time.ParseDuration(s); err == nil && d >= 0 {
		return now.Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("%q is not RFC 3339, @unix, or a duration", s)
}
