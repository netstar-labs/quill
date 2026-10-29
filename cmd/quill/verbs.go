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
			dirDoc: "the parent directory holding the logs", run: cmdLs},
		{name: "stat", summary: "print a log's head, first retained sequence, segments, bytes, records, holes and marker age", lock: "none",
			flags: func(fs *flag.FlagSet, c *cli) { fs.BoolVar(&c.json, "json", false, "one JSON object") },
			run:   cmdStat},
		{name: "read", summary: "print records from an offset (default the first retained)", lock: "none",
			flags: readFlags, run: cmdRead},
		{name: "cat", summary: "print every retained record, from the first", lock: "none",
			flags: readFlags, run: cmdRead},
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
	fs := v.flagSet(c, c.dir)
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
