package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
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
	dirDoc  string // the -dir meaning in help and the reference; empty: "the log directory"
	noDir   bool   // runs without -dir, and has no flags of its own to document
	hidden  bool   // left out of the help list and the reference
	args    bool   // takes positional arguments (append: the records; help: a verb name); any other verb refuses one
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

// errorf prints one "quill: " prefix: the library's errors already begin with it.
func (c *cli) errorf(format string, args ...any) int {
	fmt.Fprintf(c.err, "quill: %s\n", strings.TrimPrefix(fmt.Sprintf(format, args...), "quill: "))
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
			dirDoc: "the parent directory holding the logs", run: cmdLs},
		{name: "stat", summary: "print a log's head, first retained sequence, segments, bytes, records, holes and marker age", lock: "none",
			flags: func(fs *flag.FlagSet, c *cli) { fs.BoolVar(&c.json, "json", false, "one JSON object") },
			run:   cmdStat},
		{name: "read", summary: "print records from an offset (default the start of the log: the first retained sequence, or 1 when a damaged oldest segment left a gap)", lock: "none",
			flags: readFlags, run: cmdRead},
		{name: "cat", summary: "print the retained records from the start of the log, or from -from or -since; a damaged oldest segment is reported as a gap", lock: "none",
			flags: readFlags, run: cmdRead},
		{name: "tail", summary: "print the last records, and keep following with -follow", lock: "none",
			flags: func(fs *flag.FlagSet, c *cli) {
				readFlagsFrom(fs, c, "the first sequence to print (default: the last -n records; -n is ignored when -from or -since is given)")
				c.n = 10
				fs.Var((*decimalFlag)(&c.n), "n", "how many of the last records")
				fs.BoolVar(&c.follow, "follow", false, "keep printing new records until interrupted")
			}, run: cmdTail},
		{name: "append", summary: "append the arguments as records, or each line of stdin, as one batch (MaxBatchBytes applies, about 479,000 one-byte lines); needs the writer's lock", lock: "log", args: true,
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
		{name: "register", summary: "create a consumer at the first retained sequence (at 1 when a damaged oldest segment left a gap), or change its protection", lock: "consumer",
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
		{name: "archive", summary: "copy sealed segments not yet archived to -to, verifying every CRC; runs beside a live writer; exit 3 when the source had lost records", lock: "consumer",
			flags: func(fs *flag.FlagSet, c *cli) { fs.StringVar(&c.to, "to", "", "the destination directory") },
			run:   cmdArchive},
		{name: "recover", summary: "open the log writable, which repairs a torn tail and republishes quill.tail, then close it", lock: "log",
			run: cmdRecover},
		{name: "help", summary: "print this list, or a verb's flags", lock: "none", noDir: true, args: true, run: cmdHelp},
		{name: "version", summary: "print the version", lock: "none", noDir: true,
			run: func(c *cli, args []string) int { fmt.Fprintln(c.out, "quill", version); return exitOK }},
		{name: "reference", summary: "print docs/reference.md", lock: "none", noDir: true, hidden: true, run: cmdReference},
	}
}

func readFlags(fs *flag.FlagSet, c *cli) {
	readFlagsFrom(fs, c, "the first sequence to print (default the start of the log: the first retained sequence, or 1 when a damaged oldest segment left a gap)")
}

// readFlagsFrom is readFlags with the verb's own -from text, for a verb whose default start differs.
func readFlagsFrom(fs *flag.FlagSet, c *cli, fromDoc string) {
	fs.Var((*decimalFlag)(&c.from), "from", fromDoc)
	fs.Var((*decimalFlag)(&c.limit), "limit", "stop after this many records (0: no limit)")
	fs.StringVar(&c.since, "since", "", "start at the first record at or after this time (RFC 3339, @unix, or a duration ago such as 2h); wins over -from")
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

// run executes one invocation and returns the exit code. Standard output is wrapped so that
// a write that fails (a full disk, a closed descriptor, an unwritable file) is not lost
// behind the buffering of the verbs: a verb that would have exited 0 exits 1 instead.
func run(args []string, in io.Reader, out, errw io.Writer) int {
	ow := &errWriter{w: out}
	code := dispatch(args, in, ow, errw)
	if code == exitOK && ow.err != nil {
		return (&cli{err: errw}).errorf("%v", ow.err)
	}
	return code
}

// errWriter remembers the first error of the writer it wraps and refuses every write after.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) Write(b []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	n, err := e.w.Write(b)
	e.err = err
	return n, err
}

// dispatch parses `quill [-dir DIR] <verb> [flags] [args]` and runs the verb.
func dispatch(args []string, in io.Reader, out, errw io.Writer) int {
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
	fs := v.flagSet(c, c.dir)
	if err := fs.Parse(rest[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printVerbHelp(c, v, fs)
			return exitOK
		}
		return c.usage("%s: %v", v.name, err)
	}
	if extra := fs.Args(); !v.args && len(extra) > 0 {
		return c.usage("%s: unexpected argument %q; flags go before arguments", v.name, extra[0])
	}
	if !v.noDir && c.dir == "" {
		return c.usage("%s: -dir is required", v.name)
	}
	return v.run(c, fs.Args())
}

// flagSet builds the verb's flag set: -dir (defaulting to dirDefault) plus the
// verb's own flags. Output is discarded; printVerbHelp redirects it when help
// is wanted.
func (v *verb) flagSet(c *cli, dirDefault string) *flag.FlagSet {
	fs := flag.NewFlagSet(v.name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dirDoc := v.dirDoc
	if dirDoc == "" {
		dirDoc = "the log directory"
	}
	fs.StringVar(&c.dir, "dir", dirDefault, dirDoc)
	if v.flags != nil {
		v.flags(fs, c)
	}
	return fs
}

func cmdHelp(c *cli, args []string) int {
	if len(args) == 1 {
		v := findVerb(args[0])
		if v == nil {
			return c.usage("unknown verb %q", args[0])
		}
		fs := v.flagSet(c, "")
		printVerbHelp(c, v, fs)
		return exitOK
	}
	fmt.Fprintln(c.out, "usage: quill [-dir DIR] <verb> [flags]")
	fmt.Fprintln(c.out)
	w := tabwriter.NewWriter(c.out, 0, 8, 2, ' ', 0)
	for _, v := range verbs {
		if v.hidden {
			continue
		}
		fmt.Fprintf(w, "  %s\t%s\n", v.name, v.summary)
	}
	_ = w.Flush()
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "Exit codes: 0 ok, 1 error, 2 usage, 3 check found the log UNHEALTHY or archive found the source had lost records.")
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
	fmt.Fprintln(c.out, "`quill [-dir DIR] <verb> [flags]`. Every verb but `help` and `version` takes `-dir`, the log directory")
	fmt.Fprintln(c.out, "(for `ls`, the parent directory holding the logs). Exit codes: 0 ok, 1 error, 2 usage, 3 `check` found the log UNHEALTHY or `archive` found the source had lost records.")
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "| Verb | Lock | Summary |")
	fmt.Fprintln(c.out, "|---|---|---|")
	for _, v := range verbs {
		if v.hidden {
			continue
		}
		fmt.Fprintf(c.out, "| `%s` | %s | %s |\n", v.name, v.lock, v.summary)
	}
	for _, v := range verbs {
		if v.hidden || v.noDir {
			continue
		}
		fmt.Fprintf(c.out, "\n## `quill %s`\n\n%s.\n\nLock: %s.\n", v.name, strings.ToUpper(v.summary[:1])+v.summary[1:], lockText(v.lock))
		fs := v.flagSet(c, "")
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
