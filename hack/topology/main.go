// Command topology runs quill as a real multi-process deployment on one machine and checks
// what it promises: one writer process, reader and consumer processes following the log from
// different offsets, an operator running the quill command line beside them, a pruner, and a
// chaos controller that kills processes at random moments, probes the writer's lock and can
// fill the filesystem. One binary, one subcommand per role:
//
//	topology run      -dir DIR [-quill QUILL] [-duration 30s] [-readers 3] [-consumers 2] ...
//	topology verify   -root ROOT                 check a kept run's directory again
//	topology selftest -dir DIR [-quill QUILL]    inject faults and require that they are reported
//	topology writer|reader|consumer|operator|pruner -root ROOT -id N    (started by run)
//
// A run lives in DIR/topology-<run id>: the log in logs/events, the control files (acknowledgements,
// observations, oracles, violations, counters) in ctl, which -ctl can move to another filesystem.
// With -retention the chaos also freezes (SIGSTOP) a protected consumer for -stall, and for the end
// of the run, so retention has a lagging protected consumer to honour; verify then checks it.
// Every record and every consumer name carries the run id. Scheduling is random, what is checked
// is not: when the run stops, verify checks every acknowledged record, contiguity, payloads,
// Check, the consumer offsets and what every reader observed against the final log. The summary
// is JSON on stdout; the exit status is 1 on any violated invariant, with the evidence on stderr.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/netstar-labs/quill"
)

const (
	hdr       = 24 // seq, run, length, checksum of the rest: the header of a payload of 24 bytes or more
	exitRetry = 75 // a role exit that is not a violation: the log was locked, or the disk is full under -fill
)

// config is written by run to ROOT/topology.json and read by every role.
type config struct {
	Dir, Ctl, Root, Log, Quill, RunID string
	Run                               uint64
	Seed                              int64
	Duration, KillEvery, Pace, Stall  time.Duration
	Readers, Consumers, HugeEvery     int
	Segment, MaxBytes, MaxRecords     int64
	Retention                         time.Duration
	Fill, Keep, NoUnregister          bool
	Fault                             string // the self-test faults: skip, skipquiet, ackearly, ttl
}

// lossy says retention or a cap may delete records; capped says a cap may delete what a
// protected consumer has not read (caps override protection).
func (c *config) lossy() bool  { return c.Retention >= 0 || c.capped() }
func (c *config) capped() bool { return c.MaxBytes > 0 || c.MaxRecords > 0 }

func (c *config) ctl(name string) string { return filepath.Join(c.Ctl, name) }

func (c *config) consumerName(id int) string { return fmt.Sprintf("c%d-%s", id, c.RunID) }

// protected says whether consumer id is protected: the writer creates those with ProtectedConsumers.
func protected(id int) bool { return id%2 == 0 }

func (c *config) logConfig() quill.Config {
	cfg := quill.Config{Dir: c.Log, Retention: c.Retention, SegmentBytes: c.Segment, MaxBytes: c.MaxBytes,
		MaxRecords: c.MaxRecords, PruneInterval: time.Second}
	if c.Fault == "ttl" { // protection lapses for a stalled consumer: what a retention that ignored it looks like
		cfg.ConsumerTTL = time.Second
	}
	for i := range c.Consumers {
		if protected(i) {
			cfg.ProtectedConsumers = append(cfg.ProtectedConsumers, c.consumerName(i))
		}
	}
	return cfg
}

// tolerated says whether err is what the disk filling up is expected to cause.
func (c *config) tolerated(err error) bool {
	return c.Fill && (errors.Is(err, syscall.ENOSPC) || errors.Is(err, quill.ErrFailed))
}

// record writes one violation to stderr and to ctl/violations.
func (c *config) record(who, format string, a ...any) {
	line := who + ": " + strings.ReplaceAll(fmt.Sprintf(format, a...), "\n", " | ")
	fmt.Fprintln(os.Stderr, "VIOLATION "+line)
	if f, err := os.OpenFile(c.ctl("violations"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
		fmt.Fprintln(f, line)
		_ = f.Close()
	}
}

// fatal records a violation and ends the role: the run stops at the first one.
func (c *config) fatal(who, format string, a ...any) {
	c.record(who, format, a...)
	os.Exit(1)
}

// mix is the splitmix64 finalizer: every pseudo-random choice below is a function of the run and the sequence.
func mix(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ x>>30) * 0xbf58476d1ce4e5b9
	x = (x ^ x>>27) * 0x94d049bb133111eb
	return x ^ x>>31
}

var sizeClasses = []int{0, 1, 23, 24, 100, 4 << 10, 64 << 10, 1 << 20}

// size is the length of the payload of record seq: every size class, the empty record, the
// header-less ones below 24 bytes, around 4 KiB, 64 KiB and 1 MiB, and (HugeEvery) near the maximum.
func (c *config) size(seq uint64) int {
	h := mix(c.Run ^ mix(seq))
	if c.HugeEvery > 0 && h%uint64(c.HugeEvery) == 0 {
		return quill.MaxPayloadBytes - int(h>>40%64)
	}
	var i int
	switch p := h % 200; { // 1 MiB: 0.5%, 64 KiB: 8%
	case p < 30:
		i = 0
	case p < 60:
		i = 1
	case p < 80:
		i = 2
	case p < 90:
		i = 3
	case p < 140:
		i = 4
	case p < 180:
		i = 5
	case p < 199:
		i = 6
	default:
		i = 7
	}
	n := sizeClasses[i]
	if i >= 4 {
		n += int(h>>32%33) - 16
	}
	return n
}

func fill(b []byte, key uint64) {
	var w [8]byte
	for i := 0; i < len(b); i += 8 {
		binary.LittleEndian.PutUint64(w[:], mix(key+uint64(i)))
		copy(b[i:], w[:])
	}
}

// payload is the record for seq: header (sequence, run id, length, checksum of the rest) and a
// body that is a function of both. Below 24 bytes there is no room for a header and the whole
// payload is the function.
func (c *config) payload(seq uint64) []byte {
	p := make([]byte, c.size(seq))
	if len(p) < hdr {
		fill(p, c.Run^seq)
		return p
	}
	fill(p[hdr:], c.Run^seq)
	binary.BigEndian.PutUint64(p, seq)
	binary.BigEndian.PutUint64(p[8:], c.Run)
	binary.BigEndian.PutUint32(p[16:], uint32(len(p)))
	binary.BigEndian.PutUint32(p[20:], crc32.ChecksumIEEE(p[hdr:]))
	return p
}

// check says what is wrong with the payload p read as record seq, or nil.
func (c *config) check(seq uint64, p []byte) error {
	if want := c.size(seq); len(p) != want {
		return fmt.Errorf("length %d, want %d", len(p), want)
	}
	if len(p) < hdr {
		if !bytes.Equal(p, c.payload(seq)) {
			return fmt.Errorf("short payload %x differs from the generated one", p)
		}
		return nil
	}
	switch s, run, n, sum := binary.BigEndian.Uint64(p), binary.BigEndian.Uint64(p[8:]),
		binary.BigEndian.Uint32(p[16:]), binary.BigEndian.Uint32(p[20:]); {
	case s != seq:
		return fmt.Errorf("header holds sequence %d", s)
	case run != c.Run:
		return fmt.Errorf("header holds run %016x, want %016x", run, c.Run)
	case int(n) != len(p):
		return fmt.Errorf("header holds length %d, have %d", n, len(p))
	case crc32.ChecksumIEEE(p[hdr:]) != sum:
		return errors.New("checksum mismatch")
	}
	return nil
}

// counters are a role's statistics, saved to ctl every half second and at exit.
type counters struct {
	mu sync.Mutex
	m  map[string]int64
}

func (n *counters) add(name string, d int64) {
	n.mu.Lock()
	n.m[name] += d
	n.mu.Unlock()
}

func (n *counters) save(path string) {
	n.mu.Lock()
	b, _ := json.Marshal(n.m)
	n.mu.Unlock()
	if os.WriteFile(path+".tmp", b, 0o600) == nil {
		_ = os.Rename(path+".tmp", path)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: topology run|selftest|verify|writer|reader|consumer|operator|pruner [flags]")
		os.Exit(2)
	}
	roles := map[string]func(context.Context, *config, int, *counters) int{
		"writer": writerMain, "reader": readerMain, "consumer": consumerMain, "operator": operatorMain, "pruner": prunerMain}
	args := os.Args[2:]
	switch name := os.Args[1]; {
	case name == "run":
		os.Exit(runCmd(args))
	case name == "verify":
		os.Exit(verifyCmd(args))
	case name == "selftest":
		os.Exit(selftestCmd(args))
	case roles[name] != nil:
		os.Exit(roleMain(name, roles[name], args))
	}
	fmt.Fprintf(os.Stderr, "topology: unknown subcommand %q\n", os.Args[1])
	os.Exit(2)
}

func loadConfig(root string) (*config, error) {
	c := &config{}
	b, err := os.ReadFile(filepath.Join(root, "topology.json"))
	if err == nil {
		err = json.Unmarshal(b, c)
	}
	return c, err
}

// roleMain loads the run's config, stops on SIGTERM and keeps the role's counters.
func roleMain(name string, fn func(context.Context, *config, int, *counters) int, args []string) int {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	root := fs.String("root", "", "the run's directory")
	id := fs.Int("id", 0, "the role's index")
	_ = fs.Parse(args)
	c, err := loadConfig(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "topology:", err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	n := &counters{m: map[string]int64{}}
	path := c.ctl(fmt.Sprintf("stats-%s-%d-%d.json", name, *id, os.Getpid()))
	go func() {
		for range time.Tick(500 * time.Millisecond) {
			n.save(path)
		}
	}()
	code := fn(ctx, c, *id, n)
	n.save(path)
	return code
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func openAppend(c *config, name string) *os.File {
	f, err := os.OpenFile(c.ctl(name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "topology:", err)
		os.Exit(2)
	}
	return f
}

// lastWord returns the number ending the last line of the file that starts with prefix
// (acknowledgements, observations and oracles are append-only files of such lines).
func lastWord(path, prefix string) (n uint64) {
	b, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(b), "\n") {
		if f := strings.Fields(line); len(f) > 1 && f[0] == prefix {
			if v, err := strconv.ParseUint(f[len(f)-1], 10, 64); err == nil {
				n = v
			}
		}
	}
	return n
}

// ---- writer ----

// writerMain appends self-verifying records in batches of 1 to 24, flushes, then appends the
// acknowledgement to ctl/acks and fsyncs it. The ackearly fault acknowledges first and dies
// before the flush.
func writerMain(ctx context.Context, c *config, _ int, n *counters) int {
	acks := openAppend(c, "acks")
	cfg := c.logConfig()
	if c.Fault == "ackearly" {
		cfg.FlushBytes, cfg.FlushInterval = 8<<20, time.Hour // nothing becomes durable but by Flush
	}
	l, err := quill.Open(cfg)
	switch {
	case errors.Is(err, quill.ErrLocked), err != nil && c.tolerated(err):
		fmt.Fprintln(os.Stderr, "writer: open:", err)
		return exitRetry
	case err != nil:
		c.fatal("writer", "Open after a kill: %v", err)
	}
	acked := lastWord(c.ctl("acks"), "ack")
	if l.Head() < acked {
		c.fatal("writer", "acknowledged %d but the log opens at head %d: acknowledged records are lost", acked, l.Head())
	}
	fmt.Fprintf(acks, "start %d\n", l.Head())
	fmt.Println("ready")
	rng := rand.New(rand.NewSource(c.Seed ^ int64(os.Getpid())))
	next := l.Head() + 1
	fail := func(err error) int {
		if c.tolerated(err) {
			fmt.Fprintln(os.Stderr, "writer:", err)
			return exitRetry
		}
		c.fatal("writer", "%v", err)
		return 1
	}
	for ctx.Err() == nil {
		batch := 1 + rng.Intn(24)
		if c.Fault == "ackearly" {
			batch = 1 + rng.Intn(4)
		}
		for range batch {
			seq, err := l.Append(c.payload(next))
			if err != nil {
				return fail(err)
			}
			if seq != next {
				c.fatal("writer", "Append returned sequence %d, want %d", seq, next)
			}
			next++
		}
		if c.Fault == "ackearly" {
			fmt.Fprintf(acks, "ack %d\n", next-1)
			_ = acks.Sync()
			_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
		}
		if err := l.Flush(); err != nil {
			return fail(err)
		}
		fmt.Fprintf(acks, "ack %d\n", next-1)
		if err := acks.Sync(); err != nil {
			c.fatal("writer", "fsync of the acknowledgement file: %v", err)
		}
		n.add("batches", 1)
		n.add("records", int64(batch))
		sleep(ctx, time.Duration(rng.Int63n(int64(c.Pace)+1)))
	}
	if err := l.Close(); err != nil {
		return fail(err)
	}
	return 0
}

// ---- readers and consumers ----

// follower checks one stream of records: contiguity, payload integrity, the tail marker and
// gaps, and keeps the observations (ctl/obs-WHO) that verify compares with the final log.
type follower struct {
	c    *config
	who  string
	n    *counters
	obs  *os.File
	next uint64 // the sequence the stream must deliver next; 0 for "wherever the log starts"
	head uint64 // the highest tail marker observed
	lo   uint64 // the delivered span not yet written to obs, lo..hi; empty when hi is 0
	hi   uint64
}

// flush writes the pending span as "s lo hi". verify requires an observer's spans and gaps to
// leave no record out between them, which finds a record the stream dropped after the inline
// checks. A consumer flushes before it commits, so a restart never begins past a span.
func (f *follower) flush() {
	if f.hi != 0 {
		fmt.Fprintf(f.obs, "s %d %d\n", f.lo, f.hi)
		f.lo, f.hi = 0, 0
	}
}

func (f *follower) rec(rec quill.Record) {
	if f.next != 0 && rec.Seq != f.next {
		f.c.fatal(f.who, "not contiguous: record %d after %d", rec.Seq, f.next-1)
	}
	if err := f.c.check(rec.Seq, rec.Payload); err != nil {
		f.c.fatal(f.who, "record %d is corrupt: %v", rec.Seq, err)
	}
	f.next = rec.Seq + 1
	if f.hi != 0 && (rec.Seq != f.hi+1 || f.hi-f.lo >= 31) {
		f.flush()
	}
	if f.hi == 0 {
		f.lo = rec.Seq
	}
	f.hi = rec.Seq
	f.n.add("records", 1)
	if rec.Seq%8 == 0 {
		fmt.Fprintf(f.obs, "r %d %08x\n", rec.Seq, crc32.ChecksumIEEE(rec.Payload))
	}
}

// mark reads the tail marker after record seq was delivered: it must not be below seq (a
// record past the marker was served) and must never go back.
func (f *follower) mark(r *quill.Reader, seq uint64) {
	h, err := r.Head()
	switch {
	case err != nil:
		f.c.fatal(f.who, "Head: %v", err)
	case h < seq:
		f.c.fatal(f.who, "record %d was delivered above the tail marker %d", seq, h)
	case h < f.head:
		f.c.fatal(f.who, "the tail marker went back from %d to %d", f.head, h)
	}
	f.head = h
}

// gap handles a GapError and returns where to resume: To+1. Without retention or caps nothing
// is ever deleted, so there is no legitimate gap.
func (f *follower) gap(g *quill.GapError) uint64 {
	if g.To < g.From || (f.next != 0 && g.From != f.next) {
		f.c.fatal(f.who, "gap %d-%d does not start at the expected sequence %d", g.From, g.To, f.next)
	}
	if !f.c.lossy() {
		f.c.fatal(f.who, "gap %d-%d in a log that deletes nothing", g.From, g.To)
	}
	f.flush()
	f.n.add("gaps", 1)
	f.n.add("gap_records", int64(g.To-g.From+1))
	fmt.Fprintf(f.obs, "g %d %d\n", g.From, g.To)
	f.next = g.To + 1
	return f.next
}

func newFollower(c *config, role string, id int, n *counters) *follower {
	return &follower{c: c, who: fmt.Sprintf("%s-%d", role, id), n: n, obs: openAppend(c, fmt.Sprintf("obs-%s-%d", role, id))}
}

// readerMain follows the log from an offset that depends on its index: the start, the tail or a
// random record. Even readers use Follow, odd ones ReadWait. After a kill it resumes after its
// last span. The skip fault drops the fifth record it is delivered, the skipquiet fault drops it
// without telling the inline checks, which leaves it to verify.
func readerMain(ctx context.Context, c *config, id int, n *counters) int {
	r, err := quill.OpenReader(c.Log, quill.ReaderOptions{PollInterval: 10 * time.Millisecond})
	if err != nil {
		c.fatal("reader", "OpenReader: %v", err)
	}
	defer r.Close()
	f := newFollower(c, "reader", id, n)
	off := lastWord(c.ctl("obs-"+f.who), "s") + 1
	if off == 1 {
		st, err := r.Stat()
		if err != nil {
			c.fatal(f.who, "Stat: %v", err)
		}
		switch span := st.Head + 1 - st.First; id % 3 {
		case 0:
			off = 0
		case 1:
			off = st.Head + 1
		default:
			off = st.First + mix(c.Run+uint64(id))%max(span, 1)
		}
	}
	f.next = off
	fmt.Println("ready")
	defer f.flush()
	delivered := 0
	step := func(rec quill.Record) {
		if delivered++; delivered == 5 && (c.Fault == "skip" || c.Fault == "skipquiet") {
			if c.Fault == "skipquiet" { // the stream forgets the record too: nothing inline can see it
				f.next = rec.Seq + 1
			}
			off = rec.Seq + 1
			return
		}
		f.rec(rec)
		f.mark(r, rec.Seq)
		off = rec.Seq + 1
	}
	for ctx.Err() == nil {
		var err error
		if id%2 == 0 {
			for rec, e := range r.Follow(ctx, off) {
				if e != nil {
					err = e
					break
				}
				step(rec)
			}
		} else {
			var recs []quill.Record
			recs, err = r.ReadWait(ctx, off, 128)
			for _, rec := range recs {
				step(rec)
			}
		}
		var g *quill.GapError
		switch {
		case ctx.Err() != nil || err == nil:
		case errors.As(err, &g):
			off = f.gap(g)
		case c.tolerated(err):
			n.add("io_errors", 1)
			sleep(ctx, 100*time.Millisecond)
		default:
			c.fatal(f.who, "read at %d: %v", off, err)
		}
	}
	return 0
}

// consumerMain is a named consumer that reads in batches and commits after each. The oracle
// file ctl/oracle-NAME holds "intent N" before each Commit and "done N" after, so after a kill
// the offset on disk must be between the last done and the last intent, and it never goes back.
func consumerMain(ctx context.Context, c *config, id int, n *counters) int {
	r, err := quill.OpenReader(c.Log, quill.ReaderOptions{PollInterval: 10 * time.Millisecond})
	if err != nil {
		c.fatal("consumer", "OpenReader: %v", err)
	}
	defer r.Close()
	f := newFollower(c, "consumer", id, n)
	name := c.consumerName(id)
	co, err := r.Consumer(name, quill.ConsumerOptions{})
	if err != nil {
		c.fatal(f.who, "Consumer(%s): %v (the lock of a killed process must be free)", name, err)
	}
	defer co.Close()
	if co.Protected() != protected(id) {
		c.fatal(f.who, "consumer %s has protection %v, want %v", name, co.Protected(), protected(id))
	}
	oracle := c.ctl("oracle-" + name)
	done, intent := lastWord(oracle, "done"), lastWord(oracle, "intent")
	off := co.Offset()
	if intent > 0 && (off < done || off > intent) {
		c.fatal(f.who, "after a restart the committed offset is %d, but the last commit that returned was %d and the last attempted %d", off, done, intent)
	}
	orc := openAppend(c, "oracle-"+name)
	f.next = off
	fmt.Println("ready")
	commit := func(next uint64) {
		fmt.Fprintf(orc, "intent %d\n", next)
		err := co.Commit(next)
		switch {
		case err == nil:
			fmt.Fprintf(orc, "done %d\n", next)
			if got := co.Offset(); got != next || got < off {
				c.fatal(f.who, "Offset is %d after Commit(%d), having been %d", got, next, off)
			}
			n.add("commits", 1)
		case errors.Is(err, quill.ErrOffsetRange) && (!protected(id) || c.capped()):
			n.add("commit_range", 1) // retention passed an unprotected consumer, or a cap passed a protected one
		case c.tolerated(err):
			n.add("io_errors", 1)
		default:
			c.fatal(f.who, "Commit(%d): %v", next, err)
		}
		off = co.Offset()
		f.next = off
	}
	for ctx.Err() == nil {
		recs, err := co.ReadWait(ctx, 64)
		for _, rec := range recs {
			f.rec(rec)
		}
		if len(recs) > 0 {
			f.mark(r, recs[len(recs)-1].Seq)
			f.flush()
			commit(recs[len(recs)-1].Seq + 1)
		}
		var g *quill.GapError
		switch {
		case ctx.Err() != nil || err == nil:
		case errors.As(err, &g):
			if protected(id) && !c.capped() {
				c.fatal(f.who, "protected consumer %s met gap %d-%d with no cap configured", name, g.From, g.To)
			}
			commit(f.gap(g))
		case c.tolerated(err):
			sleep(ctx, 100*time.Millisecond)
		default:
			c.fatal(f.who, "read at %d: %v", off, err)
		}
	}
	return 0
}

// ---- operator and pruner ----

// cli runs the quill command line and returns its stdout, stderr and exit code (-2 once ctx is done).
func (c *config) cli(ctx context.Context, args ...string) (string, string, int) {
	cctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cctx, c.Quill, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", "", -2
	}
	if err != nil && cmd.ProcessState == nil {
		c.fatal("cli", "%v", err)
	}
	return out.String(), errb.String(), cmd.ProcessState.ExitCode()
}

// operatorMain loops the read-only verbs and the verbs that run beside a live writer, in three
// concurrent loops, and checks what they print against the payload rules and each other.
func operatorMain(ctx context.Context, c *config, _ int, n *counters) int {
	log := c.Log
	var mu sync.Mutex
	var head, first uint64
	// call runs a verb: exit 0 is the only success (allow is one more exit code that is fine), but a deletion racing a read, a commit below the
	// first retained sequence and a full disk are expected where the config or the chaos causes them.
	call := func(allow int, args ...string) (string, bool) {
		out, errs, code := c.cli(ctx, args...)
		switch {
		case code == 0:
			n.add(args[0], 1)
			return out, true
		case code == -2 || code == allow:
			return out, false
		case c.lossy() && (strings.Contains(errs, "unavailable") || strings.Contains(errs, "out of range")),
			c.Fill && strings.Contains(errs, "no space"):
			n.add("tolerated", 1)
			return out, false
		}
		c.fatal("operator", "quill %s exited %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, out, errs)
		return "", false
	}
	records := func(out string, from uint64) { // contiguous from `from` (0: wherever it starts), every payload intact
		next := from
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			var rec struct {
				Seq uint64
				B64 []byte `json:"payload_b64"` // encoding/json decodes base64 into bytes
			}
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				c.fatal("operator", "unparsable record line %q: %v", line, err)
			}
			err := c.check(rec.Seq, rec.B64)
			if err != nil || (next != 0 && rec.Seq != next) {
				c.fatal("operator", "record %d printed after %d (want %d): %v", rec.Seq, next-1, next, err)
			}
			next = rec.Seq + 1
		}
	}
	ls := func() {
		if out, ok := call(0, "ls", "-dir", filepath.Dir(log)); ok && !strings.Contains(out, filepath.Base(log)) {
			c.fatal("operator", "ls does not list %s:\n%s", filepath.Base(log), out)
		}
	}
	stat := func() {
		out, ok := call(0, "stat", "-dir", log, "-json")
		if !ok {
			return
		}
		var st struct {
			Head, First uint64
			Holes       []struct{ From, To uint64 }
		}
		if err := json.Unmarshal([]byte(out), &st); err != nil {
			c.fatal("operator", "stat: %v %s", err, out)
		}
		// One finding is known and counted, not hidden (decision 94): when retention deletes the oldest
		// segment between the reader's listing and its open, that view calls it a hole from 1.
		if h := st.Holes; len(h) == 1 && h[0].From == 1 && h[0].To == st.First-1 && c.lossy() {
			n.add("stat_vanished_lead_hole", 1)
		} else if len(h) > 0 {
			c.fatal("operator", "stat reports holes: %s", out)
		}
		mu.Lock()
		defer mu.Unlock()
		if st.Head < head || st.First < first {
			c.fatal("operator", "stat went back: head %d then %d, first %d then %d", head, st.Head, first, st.First)
		}
		head, first = st.Head, st.First
	}
	consumers := func() {
		out, ok := call(0, "consumers", "-dir", log, "-json")
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			var ci struct {
				Name      string
				Protected bool
			}
			var id int
			if ok && json.Unmarshal([]byte(line), &ci) == nil && strings.HasSuffix(ci.Name, "-"+c.RunID) {
				if _, err := fmt.Sscanf(ci.Name, "c%d-", &id); err == nil && ci.Protected != protected(id) {
					c.fatal("operator", "consumers: %s is protected=%v, want %v", ci.Name, ci.Protected, protected(id))
				}
			}
		}
	}
	op := "op-" + c.RunID
	check := func() {
		out, ok := call(3, "check", "-dir", log, "-json")
		var res struct{ Healthy bool }
		switch {
		case ok:
			n.add("healthy", 1)
		case json.Unmarshal([]byte(out), &res) == nil && !res.Healthy:
			// Not excused: while the operator unregisters its own consumer, check can list the file
			// and then fail to read it, and call that fatal. That is a finding about the library,
			// and it fails the run until the library or a decision settles it (-no-unregister).
			n.add("check_unhealthy", 1)
			c.record("operator", "quill check exited 3: %s", out)
		}
	}
	read := func() {
		mu.Lock()
		from := first + mix(uint64(time.Now().UnixNano()))%max(head+1-first, 1)
		mu.Unlock()
		if out, ok := call(0, "read", "-dir", log, "-json", "-b64", "-from", strconv.FormatUint(max(from, 1), 10), "-limit", "40"); ok {
			records(out, max(from, 1))
		}
	}
	tail := func() {
		if out, ok := call(0, "tail", "-dir", log, "-json", "-b64", "-n", "20"); ok {
			records(out, 0)
		}
	}
	steps, registered := 0, false
	online := func() {
		mu.Lock()
		h := head
		mu.Unlock()
		switch steps++; steps % 4 {
		case 0:
			_, registered = call(0, "register", "-dir", log, "-name", op, "-protect")
		case 1:
			if registered { // a register that a full disk refused leaves nothing to commit or remove
				call(0, "commit", "-dir", log, "-name", op, "-offset", strconv.FormatUint(h+1, 10))
			}
		case 2:
			if registered && !c.NoUnregister {
				_, ok := call(0, "unregister", "-dir", log, "-name", op)
				registered = !ok
			}
		case 3: // exit 3: the source had lost records, which retention and caps may cause
			allow := 0
			if c.lossy() {
				allow = 3
			}
			call(allow, "archive", "-dir", log, "-to", c.ctl("archive"))
		}
	}
	fmt.Println("ready")
	var wg sync.WaitGroup
	for i, loop := range [][]func(){{ls, stat, consumers}, {check, read}, {tail, online}} {
		wg.Go(func() {
			rng := rand.New(rand.NewSource(c.Seed + int64(i)))
			for k := 0; ctx.Err() == nil; k++ {
				loop[k%len(loop)]()
				sleep(ctx, time.Duration(rng.Intn(100))*time.Millisecond)
			}
		})
	}
	wg.Wait()
	return 0
}

// prunerMain runs the verbs that need the writer's lock. Beside a live writer they must fail
// with a lock error; while the chaos has the writer down they run, and race its restart.
func prunerMain(ctx context.Context, c *config, _ int, n *counters) int {
	fmt.Println("ready")
	rng := rand.New(rand.NewSource(c.Seed))
	verbs := [][]string{{"prune"}, {"roll"}, {"prune", "-force"}}
	for k := 0; ctx.Err() == nil; k++ {
		v := verbs[k%len(verbs)]
		out, errs, code := c.cli(ctx, append(v, "-dir", c.Log)...)
		switch {
		case code == 0:
			n.add(v[0]+"_ran", 1)
		case code == -2:
		case code == 1 && strings.Contains(errs, "locked"):
			n.add(v[0]+"_locked", 1)
		case c.Fill && strings.Contains(errs, "no space"):
		default:
			c.fatal("pruner", "quill %s exited %d\nstdout: %s\nstderr: %s", strings.Join(v, " "), code, out, errs)
		}
		sleep(ctx, time.Duration(100+rng.Intn(500))*time.Millisecond)
	}
	return 0
}
