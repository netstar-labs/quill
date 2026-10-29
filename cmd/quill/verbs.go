package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/netstar-labs/quill"
)

// Exit codes (decision 53).
const (
	exitOK        = 0
	exitError     = 1
	exitUsage     = 2
	exitUnhealthy = 3
)

// verb is one command. The table drives help, docs/reference.md and the tests.
type verb struct {
	name    string
	summary string
	lock    string // "log", "consumer" or "none": what the verb takes (decision 53)
	flags   func(fs *flag.FlagSet, c *cli)
	run     func(c *cli, args []string) int
}

// cli holds the streams and the global flags of one invocation.
type cli struct {
	in       io.Reader
	out, err io.Writer
	dir      string
	// per-verb flags
	from, limit, n                     uint64
	since, at, name, to, offset        string
	json, b64, follow, protect, noProt bool
	force, sync                        bool
	retention, maxBytes, minFree       string
	maxRecords, ttl                    string
}

func (c *cli) errorf(format string, args ...any) int {
	fmt.Fprintf(c.err, "quill: "+format+"\n", args...)
	return exitError
}

func (c *cli) usage(format string, args ...any) int {
	fmt.Fprintf(c.err, "quill: "+format+"\n", args...)
	return exitUsage
}

var verbs []verb

func init() {
	verbs = []verb{
		{name: "ls", summary: "list the logs under a parent directory, with bytes, segments, retention and caps from quill.policy", lock: "none",
			run: cmdLs},
		{name: "stat", summary: "print a log's head, first retained sequence, segments, bytes, records, holes and marker age", lock: "none",
			flags: func(fs *flag.FlagSet, c *cli) { fs.BoolVar(&c.json, "json", false, "one JSON object") },
			run:   cmdStat},
		{name: "read", summary: "print records from an offset (default the first retained)", lock: "none",
			flags: readFlags, run: cmdRead},
		{name: "cat", summary: "print every retained record, from the first", lock: "none",
			flags: readFlags, run: cmdCat},
		{name: "tail", summary: "print the last records, and keep following with -follow", lock: "none",
			flags: func(fs *flag.FlagSet, c *cli) {
				readFlags(fs, c)
				fs.Uint64Var(&c.n, "n", 10, "how many of the last records")
				fs.BoolVar(&c.follow, "follow", false, "keep printing new records until interrupted")
			}, run: cmdTail},
		{name: "append", summary: "append the arguments as records, or each line of stdin; needs the writer's lock", lock: "log",
			flags: func(fs *flag.FlagSet, c *cli) {
				fs.BoolVar(&c.sync, "sync", true, "return only once the records are durable")
			},
			run: cmdAppend},
		{name: "consumers", summary: "list the consumers: offset, lag, protection", lock: "none",
			flags: func(fs *flag.FlagSet, c *cli) { fs.BoolVar(&c.json, "json", false, "one JSON object per line") },
			run:   cmdConsumers},
		{name: "commit", summary: "set a consumer's offset: -offset <next sequence> or -time <time>", lock: "consumer",
			flags: func(fs *flag.FlagSet, c *cli) {
				fs.StringVar(&c.name, "name", "", "the consumer")
				fs.StringVar(&c.offset, "offset", "", "the next sequence to read")
				fs.StringVar(&c.at, "time", "", "the time to seek to (RFC 3339, @unix, or a duration ago such as 2h)")
			}, run: cmdCommit},
		{name: "register", summary: "create a consumer at the first retained sequence, or change its protection", lock: "consumer",
			flags: func(fs *flag.FlagSet, c *cli) {
				fs.StringVar(&c.name, "name", "", "the consumer")
				fs.BoolVar(&c.protect, "protect", false, "retention keeps its unread segments")
				fs.BoolVar(&c.noProt, "no-protect", false, "retention does not wait for it (the default for a new consumer)")
			}, run: cmdRegister},
		{name: "unregister", summary: "delete a consumer's offset file (its lock file stays)", lock: "consumer",
			flags: func(fs *flag.FlagSet, c *cli) { fs.StringVar(&c.name, "name", "", "the consumer") },
			run:   cmdUnregister},
		{name: "check", summary: "verify every segment, hole, consumer file and the clock; exit 3 when UNHEALTHY", lock: "none",
			flags: func(fs *flag.FlagSet, c *cli) { fs.BoolVar(&c.json, "json", false, "one JSON object") },
			run:   cmdCheck},
		{name: "prune", summary: "run a retention pass now with the policy in quill.policy or the flags; needs the writer's lock", lock: "log",
			flags: func(fs *flag.FlagSet, c *cli) {
				fs.BoolVar(&c.force, "force", false, "lift the clock guard (delete every expired segment in one pass)")
				fs.StringVar(&c.retention, "retention", "", "age after which a sealed segment is deleted (a duration; negative never)")
				fs.StringVar(&c.maxBytes, "max-bytes", "", "cap on the log's segment bytes")
				fs.StringVar(&c.minFree, "min-free-bytes", "", "free-space floor on the filesystem")
				fs.StringVar(&c.maxRecords, "max-records", "", "keep roughly the newest N records")
				fs.StringVar(&c.ttl, "consumer-ttl", "", "a protected consumer behind for this long stops protecting")
			}, run: cmdPrune},
		{name: "roll", summary: "seal the active segment now; needs the writer's lock", lock: "log",
			run: cmdRoll},
		{name: "archive", summary: "copy sealed segments not yet archived to -to, verifying every CRC; runs beside a live writer", lock: "consumer",
			flags: func(fs *flag.FlagSet, c *cli) { fs.StringVar(&c.to, "to", "", "the destination directory") },
			run:   cmdArchive},
		{name: "recover", summary: "open the log writable, which repairs a torn tail and republishes quill.tail, then close it", lock: "log",
			run: cmdRecover},
		{name: "help", summary: "print this list, or a verb's flags", lock: "none", run: cmdHelp},
		{name: "version", summary: "print the version", lock: "none",
			run: func(c *cli, args []string) int { fmt.Fprintln(c.out, "quill", version); return exitOK }},
		{name: "reference", summary: "print docs/reference.md", lock: "none", run: cmdReference},
	}
}

func readFlags(fs *flag.FlagSet, c *cli) {
	fs.Uint64Var(&c.from, "from", 0, "the first sequence to print (default the first retained)")
	fs.Uint64Var(&c.limit, "limit", 0, "stop after this many records (0: no limit)")
	fs.StringVar(&c.since, "since", "", "start at the first record at or after this time (RFC 3339, @unix, or a duration ago such as 2h)")
	fs.BoolVar(&c.json, "json", false, "newline-delimited JSON: seq, at, time, len, payload or payload_b64")
	fs.BoolVar(&c.b64, "b64", false, "print every payload as base64")
}

func findVerb(name string) *verb {
	for i := range verbs {
		if verbs[i].name == name {
			return &verbs[i]
		}
	}
	return nil
}

// run parses `quill [-dir DIR] <verb> [flags] [args]` and runs the verb.
func run(args []string, in io.Reader, out, errw io.Writer) int {
	c := &cli{in: in, out: out, err: errw}
	global := flag.NewFlagSet("quill", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	global.StringVar(&c.dir, "dir", "", "the log directory")
	if err := global.Parse(args); err != nil {
		return c.usage("%v", err)
	}
	rest := global.Args()
	if len(rest) == 0 {
		cmdHelp(c, nil)
		return exitUsage
	}
	v := findVerb(rest[0])
	if v == nil {
		return c.usage("unknown verb %q; see quill help", rest[0])
	}
	fs := flag.NewFlagSet(v.name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&c.dir, "dir", c.dir, "the log directory")
	if v.flags != nil {
		v.flags(fs, c)
	}
	if err := fs.Parse(rest[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printVerbHelp(c, v, fs)
			return exitOK
		}
		return c.usage("%s: %v", v.name, err)
	}
	if v.name != "help" && v.name != "version" && v.name != "reference" && c.dir == "" {
		return c.usage("%s: -dir is required", v.name)
	}
	return v.run(c, fs.Args())
}

func cmdHelp(c *cli, args []string) int {
	if len(args) == 1 {
		v := findVerb(args[0])
		if v == nil {
			return c.usage("unknown verb %q", args[0])
		}
		fs := flag.NewFlagSet(v.name, flag.ContinueOnError)
		fs.StringVar(&c.dir, "dir", "", "the log directory")
		if v.flags != nil {
			v.flags(fs, c)
		}
		printVerbHelp(c, v, fs)
		return exitOK
	}
	fmt.Fprintln(c.out, "usage: quill [-dir DIR] <verb> [flags]")
	fmt.Fprintln(c.out)
	w := tabwriter.NewWriter(c.out, 0, 8, 2, ' ', 0)
	for _, v := range verbs {
		if v.name == "reference" {
			continue
		}
		fmt.Fprintf(w, "  %s\t%s\n", v.name, v.summary)
	}
	_ = w.Flush()
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "Exit codes: 0 ok, 1 error, 2 usage, 3 check found the log UNHEALTHY.")
	return exitOK
}

func printVerbHelp(c *cli, v *verb, fs *flag.FlagSet) {
	fmt.Fprintf(c.out, "quill %s: %s\n", v.name, v.summary)
	fmt.Fprintf(c.out, "lock: %s\n", lockText(v.lock))
	fs.SetOutput(c.out)
	fs.PrintDefaults()
}

func lockText(lock string) string {
	switch lock {
	case "log":
		return "the writer's lock (fails with ErrLocked beside a live writer)"
	case "consumer":
		return "the consumer's lock only (runs beside a live writer)"
	}
	return "none (runs beside a live writer, sees records up to quill.tail)"
}

// cmdReference prints docs/reference.md from the verb table.
func cmdReference(c *cli, args []string) int {
	fmt.Fprintln(c.out, "# quill command reference")
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "*Generated from the verb table in `cmd/quill/verbs.go` (`quill reference`); a test keeps it current. Do not edit by hand.*")
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "`quill [-dir DIR] <verb> [flags]`. Every verb but `help`, `version` and `ls` takes `-dir`, the log directory")
	fmt.Fprintln(c.out, "(`ls` takes the parent). Exit codes: 0 ok, 1 error, 2 usage, 3 `check` found the log UNHEALTHY.")
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "| Verb | Lock | Summary |")
	fmt.Fprintln(c.out, "|---|---|---|")
	for _, v := range verbs {
		if v.name == "reference" {
			continue
		}
		fmt.Fprintf(c.out, "| `%s` | %s | %s |\n", v.name, v.lock, v.summary)
	}
	for _, v := range verbs {
		if v.name == "reference" || v.name == "help" || v.name == "version" {
			continue
		}
		fmt.Fprintf(c.out, "\n## `quill %s`\n\n%s.\n\nLock: %s.\n", v.name, strings.ToUpper(v.summary[:1])+v.summary[1:], lockText(v.lock))
		fs := flag.NewFlagSet(v.name, flag.ContinueOnError)
		fs.StringVar(&c.dir, "dir", "", "the log directory")
		if v.flags != nil {
			v.flags(fs, c)
		}
		var names []string
		fs.VisitAll(func(f *flag.Flag) { names = append(names, f.Name) })
		sort.Strings(names)
		fmt.Fprintln(c.out)
		fmt.Fprintln(c.out, "| Flag | Default | Meaning |")
		fmt.Fprintln(c.out, "|---|---|---|")
		for _, n := range names {
			f := fs.Lookup(n)
			def := f.DefValue
			if def == "" {
				def = "—"
			}
			fmt.Fprintf(c.out, "| `-%s` | `%s` | %s |\n", f.Name, def, f.Usage)
		}
	}
	return exitOK
}

// Readers and writers.

func (c *cli) reader() (*quill.Reader, int) {
	r, err := quill.OpenReader(c.dir, quill.ReaderOptions{})
	if err != nil {
		return nil, c.errorf("%v", err)
	}
	return r, exitOK
}

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

// policyConfig turns quill.policy into a Config, so offline tools use the writer's settings.
func (c *cli) policyConfig() (quill.Config, bool) {
	p, ok := quill.ReadPolicy(c.dir)
	if !ok {
		return quill.Config{}, false
	}
	return p, true
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

func cmdRead(c *cli, args []string) int { return printFrom(c, false) }
func cmdCat(c *cli, args []string) int  { return printFrom(c, false) }

func printFrom(c *cli, follow bool) int {
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
	from, code := c.start(r)
	if code != exitOK {
		return code
	}
	return printRecords(c, r, from, follow)
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
		from = st.Head + 1
		if c.n > 0 && st.Head >= c.n {
			from = st.Head + 1 - c.n
		} else if c.n > 0 {
			from = 1
		}
		from = max(from, st.First)
	}
	return printRecords(c, r, from, c.follow)
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

func cmdConsumers(c *cli, args []string) int {
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
	list, err := r.Consumers()
	if err != nil {
		return c.errorf("%v", err)
	}
	head, err := r.Head()
	if err != nil {
		return c.errorf("%v", err)
	}
	if c.json {
		for _, ci := range list {
			row := map[string]any{"name": ci.Name, "offset": ci.Offset, "protected": ci.Protected, "lag": lag(head, ci.Offset)}
			if ci.Err != nil {
				row["error"] = ci.Err.Error()
			}
			if code := printJSON(c, row); code != exitOK {
				return code
			}
		}
		return exitOK
	}
	w := tabwriter.NewWriter(c.out, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tOFFSET\tLAG\tPROTECTED\tNOTE")
	for _, ci := range list {
		note := ""
		if ci.Err != nil {
			note = "error: " + ci.Err.Error()
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%v\t%s\n", ci.Name, ci.Offset, lag(head, ci.Offset), ci.Protected, note)
	}
	_ = w.Flush()
	return exitOK
}

func lag(head, offset uint64) uint64 {
	if offset > head+1 {
		return 0
	}
	return head + 1 - offset
}

func (c *cli) consumer(r *quill.Reader, opts quill.ConsumerOptions) (*quill.Consumer, int) {
	if c.name == "" {
		return nil, c.usage("-name is required")
	}
	cons, err := r.Consumer(c.name, opts)
	if err != nil {
		return nil, c.errorf("%v", err)
	}
	return cons, exitOK
}

func cmdCommit(c *cli, args []string) int {
	if (c.offset == "") == (c.at == "") {
		return c.usage("commit: exactly one of -offset and -time")
	}
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
	cons, code := c.consumer(r, quill.ConsumerOptions{})
	if code != exitOK {
		return code
	}
	defer cons.Close()
	if c.at != "" {
		t, err := parseTime(c.at, time.Now())
		if err != nil {
			return c.usage("-time: %v", err)
		}
		if err := cons.SeekToTime(t); err != nil {
			return c.errorf("%v", err)
		}
	} else {
		var next uint64
		if _, err := fmt.Sscanf(c.offset, "%d", &next); err != nil {
			return c.usage("-offset: %v", err)
		}
		if err := cons.Commit(next); err != nil {
			return c.errorf("%v", err)
		}
	}
	fmt.Fprintf(c.out, "%s %d\n", c.name, cons.Offset())
	return exitOK
}

func cmdRegister(c *cli, args []string) int {
	if c.protect && c.noProt {
		return c.usage("register: -protect and -no-protect exclude each other")
	}
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
	opts := quill.ConsumerOptions{}
	switch {
	case c.protect:
		opts.Protect = quill.ProtectOn
	case c.noProt:
		opts.Protect = quill.ProtectOff
	}
	cons, code := c.consumer(r, opts)
	if code != exitOK {
		return code
	}
	defer cons.Close()
	fmt.Fprintf(c.out, "%s %d protected=%v\n", c.name, cons.Offset(), cons.Protected())
	return exitOK
}

func cmdUnregister(c *cli, args []string) int {
	if c.name == "" {
		return c.usage("-name is required")
	}
	r, code := c.reader()
	if code != exitOK {
		return code
	}
	defer r.Close()
	if err := r.DeleteConsumer(c.name); err != nil {
		return c.errorf("%v", err)
	}
	return exitOK
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

func cmdPrune(c *cli, args []string) int {
	cfg, ok := c.policyConfig()
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

func cmdArchive(c *cli, args []string) int {
	if c.to == "" {
		return c.usage("archive: -to is required")
	}
	err := quill.Archive(c.dir, c.to)
	var gap *quill.GapError
	if errors.As(err, &gap) {
		fmt.Fprintf(c.err, "quill: the source lost records %d to %d; the archive holds the same hole\n", gap.From, gap.To)
		return exitOK
	}
	if err != nil {
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
