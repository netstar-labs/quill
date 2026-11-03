package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/netstar-labs/quill"
)

// summary is what a run prints as JSON.
type summary struct {
	RunID      string           `json:"run_id"`
	Seed       int64            `json:"seed"`
	Root       string           `json:"root"`
	Fault      string           `json:"fault,omitempty"`
	OK         bool             `json:"ok"`
	Head       uint64           `json:"head"`
	First      uint64           `json:"first"`
	Acked      uint64           `json:"acked"`
	Segments   int              `json:"segments"`
	Verified   int              `json:"records_verified"`
	Counters   map[string]int64 `json:"counters"`
	Violations []string         `json:"violations"`
}

func runCmd(args []string) int {
	c := &config{}
	if code := parseFlags("run", c, args); code != 0 {
		return code
	}
	return emit(execute(c))
}

// verifyCmd runs the verification again on a kept run (-keep, or any failed run), for example
// after the log was damaged by hand: the checks are the ones that end every run.
func verifyCmd(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	root := fs.String("root", "", "the run's directory (topology-<run id>)")
	if fs.Parse(args) != nil {
		return 2
	}
	c, err := loadConfig(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "topology:", err)
		return 2
	}
	_ = os.Rename(c.ctl("violations"), c.ctl("violations.prev")) // the run's own findings stay, aside
	s := &summary{RunID: c.RunID, Seed: c.Seed, Root: c.Root, Counters: map[string]int64{}}
	c.verify(s)
	c.collect(s, &counters{})
	s.OK = len(s.Violations) == 0
	return emit(s)
}

func emit(s *summary) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(s)
	if !s.OK {
		fmt.Fprintf(os.Stderr, "topology: FAILED, run kept in %s\n", s.Root)
		return 1
	}
	return 0
}

func parseFlags(name string, c *config, args []string) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&c.Dir, "dir", "", "the directory on the filesystem under test (required)")
	fs.StringVar(&c.Ctl, "ctl", "", "where the control files go (default: inside the run's directory; required with -fill, on another filesystem)")
	fs.StringVar(&c.Quill, "quill", os.Getenv("QUILL_BIN"), "the quill command line tool (default $QUILL_BIN; without it there is no operator and no pruner)")
	fs.DurationVar(&c.Duration, "duration", 30*time.Second, "how long the chaos runs")
	fs.IntVar(&c.Readers, "readers", 3, "reader processes")
	fs.IntVar(&c.Consumers, "consumers", 2, "consumer processes (even indexes are protected)")
	fs.Int64Var(&c.Segment, "segment", 256<<10, "segment size in bytes")
	fs.Int64Var(&c.MaxBytes, "max-bytes", 0, "cap on the log's bytes (0: off, else at least twice -segment)")
	fs.Int64Var(&c.MaxRecords, "max-records", 0, "cap on the records kept (0: off)")
	fs.Func("retention", `age after which sealed segments are deleted, a duration or "never" (default never)`, func(s string) (err error) {
		if s == "never" {
			c.Retention = -1
		} else {
			c.Retention, err = time.ParseDuration(s)
		}
		return err
	})
	fs.Int64Var(&c.Seed, "seed", 0, "seed of the chaos and the writer's batches (0: the clock; printed in the summary)")
	fs.DurationVar(&c.KillEvery, "kill-every", 2*time.Second, "mean time between chaos actions")
	fs.DurationVar(&c.Pace, "pace", 20*time.Millisecond, "the writer sleeps up to this between batches")
	fs.IntVar(&c.HugeEvery, "huge-every", 20000, "one record in this many is near the maximum payload size (0: never)")
	fs.BoolVar(&c.Fill, "fill", false, "also fill the filesystem under test: only on one you can afford to fill, with -ctl elsewhere")
	fs.BoolVar(&c.Keep, "keep", false, "keep the run's directory when it passes")
	fs.DurationVar(&c.Stall, "stall", 0, "how long the chaos freezes (SIGSTOP) a protected consumer while retention runs (default: 3 times -retention, at least 4s)")
	fs.BoolVar(&c.NoUnregister, "no-unregister", false, "the operator never unregisters its consumer: quill check can report a vanished consumer file as UNHEALTHY while it does, which fails the run")
	fs.BoolVar(&c.Quick, "quick", false, "selftest: skip the faults that depend on retention timing (protected, ttl); the go test suite uses it so the gate cannot flake on a loaded machine")
	fs.StringVar(&c.Fault, "fault", "", `self-test fault: "skip" or "skipquiet" (a reader drops a record, in the second the inline checks do not know), "ackearly" (the writer acknowledges before it flushes), "ttl" (protection of a stalled consumer lapses)`)
	c.Retention = -1
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if c.Dir == "" || (c.Fill && c.Ctl == "") || c.Readers < 0 || c.Consumers < 0 || c.KillEvery <= 0 || c.Pace < 0 {
		fs.Usage()
		return 2
	}
	return 0
}

// proc is one role the run keeps alive.
type proc struct {
	role    string
	id      int
	mu      sync.Mutex
	cmd     *exec.Cmd // the running child
	up      bool      // it printed "ready"
	hit     bool      // the chaos killed it
	stopped bool      // the chaos froze it with SIGSTOP
}

// Write marks the child ready: a role prints only "ready", once its setup is done.
func (p *proc) Write(b []byte) (int, error) {
	p.mu.Lock()
	p.up = true
	p.mu.Unlock()
	return len(b), nil
}

// livePid is the pid of the child if it is up, else 0.
func (p *proc) livePid() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.up {
		return p.cmd.Process.Pid
	}
	return 0
}

func (p *proc) isStopped() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopped
}

// signal sends sig to the child and reports whether it did: a SIGKILL is dropped while the child
// is not up. A frozen child never gets to act on SIGTERM, so it is killed, which leaves its
// committed offset where the freeze put it.
func (p *proc) signal(sig syscall.Signal, ours bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped && sig == syscall.SIGTERM {
		sig = syscall.SIGKILL
	}
	if p.cmd != nil && (p.up || sig != syscall.SIGKILL) {
		p.hit = p.hit || ours
		return p.cmd.Process.Signal(sig) == nil
	}
	return false
}

type orch struct {
	c      *config
	exe    string
	ctx    context.Context
	cancel context.CancelFunc
	procs  []*proc
	n      *counters
}

// execute makes the run's directory, starts the roles, lets the chaos work for the duration,
// stops everything, verifies and returns the summary.
func execute(c *config) *summary {
	if c.Seed == 0 {
		c.Seed = time.Now().UnixNano()
	}
	if c.Stall == 0 {
		c.Stall = max(3*c.Retention, 4*time.Second)
	}
	c.Run = mix(uint64(c.Seed) + uint64(time.Now().UnixNano()))
	c.RunID = fmt.Sprintf("%016x", c.Run)
	dir, err := filepath.Abs(c.Dir)
	if err != nil {
		return &summary{Violations: []string{err.Error()}}
	}
	c.Root = filepath.Join(dir, "topology-"+c.RunID)
	c.Log = filepath.Join(c.Root, "logs", "events")
	if c.Ctl == "" {
		c.Ctl = filepath.Join(c.Root, "ctl")
	} else {
		c.Ctl = filepath.Join(c.Ctl, "topology-"+c.RunID)
	}
	s := &summary{RunID: c.RunID, Seed: c.Seed, Root: c.Root, Fault: c.Fault, Counters: map[string]int64{}}
	b, _ := json.Marshal(c)
	for _, d := range []string{filepath.Dir(c.Log), c.Ctl} {
		if err = os.MkdirAll(d, 0o700); err != nil {
			break
		}
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(c.Root, "topology.json"), b, 0o600)
	}
	if err == nil && c.Fill && sameDevice(c.Root, c.Ctl) {
		err = errors.New("-fill needs -ctl on another filesystem than -dir: the control files must survive the disk filling up")
	}
	if err != nil {
		s.Violations = []string{err.Error()}
		return s
	}
	o := &orch{c: c, n: &counters{m: map[string]int64{}}}
	o.exe, _ = os.Executable()
	o.ctx, o.cancel = context.WithTimeout(context.Background(), c.Duration)
	defer o.cancel()
	o.procs = append(o.procs, &proc{role: "writer"})
	for i := range c.Readers {
		o.procs = append(o.procs, &proc{role: "reader", id: i})
	}
	for i := range c.Consumers {
		o.procs = append(o.procs, &proc{role: "consumer", id: i})
	}
	if c.Quill != "" {
		o.procs = append(o.procs, &proc{role: "operator"}, &proc{role: "pruner"})
	}
	var wg sync.WaitGroup
	wg.Go(func() { o.supervise(o.procs[0]) })
	for deadline := time.Now().Add(30 * time.Second); o.procs[0].livePid() == 0 && o.ctx.Err() == nil; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			c.record("run", "the writer was not ready after 30s")
			o.cancel()
		}
	}
	for _, p := range o.procs[1:] {
		wg.Go(func() { o.supervise(p) })
	}
	wg.Go(o.chaos)
	wg.Go(o.stalls)
	<-o.ctx.Done()
	for _, p := range o.procs {
		p.signal(syscall.SIGTERM, false)
	}
	stopped := make(chan struct{})
	go func() { wg.Wait(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		c.record("run", "the roles did not stop 30s after SIGTERM")
		for _, p := range o.procs {
			p.signal(syscall.SIGKILL, true)
		}
		<-stopped
	}
	c.verify(s)
	c.collect(s, o.n)
	s.OK = len(s.Violations) == 0
	if s.OK && !c.Keep {
		_ = os.RemoveAll(c.Root)
		_ = os.RemoveAll(c.Ctl)
	}
	return s
}

func sameDevice(a, b string) bool {
	var x, y syscall.Stat_t
	return syscall.Stat(a, &x) == nil && syscall.Stat(b, &y) == nil && x.Dev == y.Dev
}

// supervise keeps one role alive: a chaos kill and a retryable exit restart it, any other exit
// that nobody asked for is a violation and stops the run.
func (o *orch) supervise(p *proc) {
	rng := rand.New(rand.NewSource(o.c.Seed + int64(p.id)))
	for o.ctx.Err() == nil {
		cmd := exec.Command(o.exe, p.role, "-root", o.c.Root, "-id", strconv.Itoa(p.id))
		cmd.Stdout, cmd.Stderr = p, os.Stderr
		if err := cmd.Start(); err != nil {
			o.c.record("run", "starting %s: %v", p.role, err)
			o.cancel()
			return
		}
		p.mu.Lock()
		p.cmd, p.up = cmd, false
		p.mu.Unlock()
		if o.ctx.Err() != nil { // the run ended while this child started
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
		_ = cmd.Wait()
		p.mu.Lock()
		killed := p.hit
		p.cmd, p.up, p.hit, p.stopped = nil, false, false, false
		p.mu.Unlock()
		switch code := cmd.ProcessState.ExitCode(); {
		case o.ctx.Err() != nil && code <= 0: // stopped by us: SIGTERM, or SIGKILL after the watchdog
		case killed || code == exitRetry:
			o.n.add("restarts_"+p.role, 1)
		default:
			if code != 1 { // 1 is a violation the role recorded itself
				o.c.record("run", "%s %d exited with code %d, which nobody asked for (%s)", p.role, p.id, code, cmd.ProcessState)
			}
			o.cancel()
			return
		}
		select {
		case <-o.ctx.Done():
		case <-time.After(time.Duration(20+rng.Intn(180)) * time.Millisecond):
		}
	}
}

// kill SIGKILLs p, unless a frozen consumer has to stay frozen until the stall ends. It counts
// the signals sent, not the attempts: restarts_<role> is what remains of them (a kill sent to a
// child that already died and is not yet reaped counts here, and one restart covers both).
func (o *orch) kill(p *proc) {
	if p.isStopped() {
		return
	}
	if p.signal(syscall.SIGKILL, true) { // a child between its exit and "ready" is not signalled
		o.n.add("kills_"+p.role, 1)
	}
}

// chaos acts at random times until the run ends: SIGKILL the writer, a reader or a consumer,
// start a second writer, or fill the disk.
func (o *orch) chaos() {
	rng := rand.New(rand.NewSource(o.c.Seed))
	pick := func(role string) *proc {
		var ps []*proc
		for _, p := range o.procs {
			if p.role == role {
				ps = append(ps, p)
			}
		}
		if len(ps) == 0 {
			return o.procs[0]
		}
		return ps[rng.Intn(len(ps))]
	}
	kill := o.kill
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-time.After(o.c.KillEvery/2 + time.Duration(rng.Int63n(int64(o.c.KillEvery)))):
		}
		switch n := rng.Intn(100); {
		case n < 45:
			kill(o.procs[0])
		case n < 65:
			kill(pick("reader"))
		case n < 80:
			kill(pick("consumer"))
		case n < 90:
			o.probe()
		case o.c.Fill:
			o.fill(time.Duration(1+rng.Intn(3)) * time.Second)
		default:
			kill(o.procs[0])
		}
	}
}

// stalls freezes protected consumers while retention is running, which is the only way a protected
// consumer is ever behind enough for retention to want its segments: a killed one is back within
// 200ms. It freezes one for -stall at random times, and for the last -stall of the run, so the run
// ends with a consumer that is frozen far behind. Retention that honours protection leaves the
// frozen consumer's records alone; one that does not is found by the consumer when it thaws
// (a gap it must not meet) and by verify when it never does.
func (o *orch) stalls() {
	var ps []*proc
	for _, p := range o.procs {
		if p.role == "consumer" && protected(p.id) {
			ps = append(ps, p)
		}
	}
	if len(ps) == 0 || o.c.Retention < 0 {
		return
	}
	rng := rand.New(rand.NewSource(o.c.Seed + 1))
	d := o.c.Stall
	last, _ := o.ctx.Deadline()
	last = last.Add(-d)
	for o.ctx.Err() == nil { // a freeze and a stretch of running, if they end before the last freeze begins
		run := d/2 + time.Duration(rng.Int63n(int64(d/2)+1))
		if time.Now().Add(run + d + d/2).After(last) {
			break
		}
		sleep(o.ctx, run)
		o.stall(ps[rng.Intn(len(ps))], d)
	}
	sleep(o.ctx, time.Until(last))
	p := ps[rng.Intn(len(ps))]
	for o.ctx.Err() == nil && !o.stall(p, 2*d) { // a child that was just killed is not up yet
		sleep(o.ctx, 50*time.Millisecond)
	}
}

// stall freezes p for d unless the run ends first, and reports whether it did. If the run ends
// first the child stays frozen: signal kills it instead of asking it to stop.
func (o *orch) stall(p *proc, d time.Duration) bool {
	p.mu.Lock()
	if o.ctx.Err() != nil || p.cmd == nil || !p.up {
		p.mu.Unlock()
		return false
	}
	cmd := p.cmd
	p.stopped = true
	_ = cmd.Process.Signal(syscall.SIGSTOP)
	p.mu.Unlock()
	o.n.add("stalls", 1)
	sleep(o.ctx, d)
	p.mu.Lock()
	defer p.mu.Unlock()
	if o.ctx.Err() == nil && p.stopped && p.cmd == cmd {
		p.stopped = false
		_ = cmd.Process.Signal(syscall.SIGCONT)
	}
	return true
}

// probe starts a second writer beside the live one, with the library and with the command line:
// both must be refused with the lock error.
func (o *orch) probe() {
	w := o.procs[0]
	pid := w.livePid()
	if pid == 0 {
		return
	}
	bad := ""
	l, err := quill.Open(o.c.logConfig())
	switch {
	case err == nil:
		_ = l.Close()
		bad = "a second writer opened the log beside the live one"
	case !errors.Is(err, quill.ErrLocked):
		bad = fmt.Sprintf("a second Open failed with %v, not ErrLocked", err)
	case o.c.Quill != "":
		if _, errs, code := o.c.cli(o.ctx, "append", "-dir", o.c.Log, "probe-"+o.c.RunID); code != 1 && code != -2 || !strings.Contains(errs, "locked") && code != -2 {
			bad = fmt.Sprintf("quill append beside the writer exited %d: %s", code, errs)
		}
	}
	o.n.add("probes", 1)
	if bad != "" && w.livePid() == pid { // not if the writer went away meanwhile
		o.c.record("chaos", "%s", bad)
		o.cancel()
	}
}

// fill writes a file until the filesystem is full, holds it for d, then removes it.
func (o *orch) fill(d time.Duration) {
	path := filepath.Join(o.c.Root, "fill.bin")
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer os.Remove(path)
	defer f.Close()
	for size := 1 << 20; size >= 512; size /= 4 { // smaller and smaller writes take the last blocks
		for buf := make([]byte, size); ; {
			if _, err := f.Write(buf); err != nil {
				break
			}
		}
	}
	_ = f.Sync()
	o.n.add("fills", 1)
	select {
	case <-o.ctx.Done():
	case <-time.After(d):
	}
}

// observations reads what the readers and consumers saw: the spans of records each delivered and
// the gaps it met, the checksum of every 8th record, and the highest sequence any span reaches.
// An observer's spans and gaps must leave no record out, and two observers that disagree about a
// record are a violation themselves.
func (c *config) observations() (seen map[uint64]uint32, gaps [][2]uint64, top uint64) {
	seen = map[uint64]uint32{}
	files, _ := filepath.Glob(c.ctl("obs-*"))
	for _, name := range files {
		b, _ := os.ReadFile(name)
		var spans [][2]uint64
		for _, line := range strings.Split(string(b), "\n") {
			f := strings.Fields(line)
			if len(f) != 3 {
				continue // a torn last line
			}
			a, _ := strconv.ParseUint(f[1], 10, 64)
			switch to, _ := strconv.ParseUint(f[2], 10, 64); f[0] {
			case "g":
				gaps = append(gaps, [2]uint64{a, to})
				spans = append(spans, [2]uint64{a, to})
			case "s":
				spans = append(spans, [2]uint64{a, to})
				top = max(top, to)
			case "r":
				sum, _ := strconv.ParseUint(f[2], 16, 32)
				if old, ok := seen[a]; ok && old != uint32(sum) {
					c.record("verify", "two observers saw different bytes for record %d (checksums %08x and %08x)", a, old, sum)
				}
				seen[a] = uint32(sum)
			}
		}
		slices.SortFunc(spans, func(x, y [2]uint64) int { return cmp.Compare(x[0], y[0]) })
		var hi uint64
		for i, sp := range spans {
			if i > 0 && sp[0] > hi+1 {
				c.record("verify", "%s: records %d-%d are in none of its spans and in none of its gaps (it delivered up to %d, then from %d)", filepath.Base(name), hi+1, sp[0]-1, hi, sp[0])
			}
			hi = max(hi, sp[1])
		}
	}
	return seen, gaps, top
}

// verify checks the stopped log against everything the run recorded.
func (c *config) verify(s *summary) {
	acked := lastWord(c.ctl("acks"), "ack")
	s.Acked = acked
	seen, gaps, top := c.observations()
	r, err := quill.OpenReader(c.Log, quill.ReaderOptions{})
	if err != nil {
		c.record("verify", "OpenReader: %v", err)
		return
	}
	defer r.Close()
	healthy := func(stage string) {
		res, err := r.Check()
		if err != nil {
			c.record("verify", "Check %s: %v", stage, err)
		}
		for _, p := range res.Problems {
			if p.Fatal {
				c.record("verify", "Check %s: %s", stage, p)
			}
		}
	}
	healthy("before recovery")
	st, err := r.Stat()
	if err != nil {
		c.record("verify", "Stat: %v", err)
		return
	}
	s.Head, s.First, s.Segments = st.Head, st.First, st.Segments
	if len(st.Holes) > 0 {
		c.record("verify", "the log has holes: %+v", st.Holes)
	}
	if acked > st.Head {
		c.record("verify", "acknowledged record %d, but the log ends at %d: acknowledged records are lost", acked, st.Head)
	}
	if !c.lossy() && st.First != 1 && st.Head > 0 {
		c.record("verify", "the log starts at %d, but nothing may be deleted", st.First)
	}
	var last uint64
	err = r.ScanAll(func(rec quill.Record) bool {
		if last != 0 && rec.Seq != last+1 || last == 0 && rec.Seq != st.First {
			c.record("verify", "not contiguous: record %d after %d (first %d)", rec.Seq, last, st.First)
		}
		if err := c.check(rec.Seq, rec.Payload); err != nil {
			c.record("verify", "record %d is corrupt: %v", rec.Seq, err)
		}
		if sum, ok := seen[rec.Seq]; ok && sum != crc32.ChecksumIEEE(rec.Payload) {
			c.record("verify", "record %d reads differently from what an observer saw", rec.Seq)
		}
		delete(seen, rec.Seq)
		last = rec.Seq
		s.Verified++
		return true
	})
	if err != nil {
		c.record("verify", "ScanAll: %v", err)
	}
	if last != st.Head && st.First <= st.Head { // First is Head+1 when retention left nothing but an empty active segment
		c.record("verify", "the scan ended at %d, head is %d", last, st.Head)
	}
	if top > st.Head {
		c.record("verify", "an observer was delivered records up to %d, above the final head %d", top, st.Head)
	}
	var above, lost []uint64 // observed, but not in the log: one line each, not one per record
	for seq := range seen {
		if seq > st.Head {
			above = append(above, seq)
		} else if !c.lossy() || seq >= st.First {
			lost = append(lost, seq)
		}
	}
	if len(above) > 0 {
		c.record("verify", "observers saw %d records above the final head %d, such as %d", len(above), st.Head, slices.Min(above))
	}
	if len(lost) > 0 {
		c.record("verify", "observers saw %d records that the log does not hold, such as %d", len(lost), slices.Min(lost))
	}
	for _, g := range gaps {
		if !c.lossy() || g[1] >= st.First {
			c.record("verify", "an observer met gap %d-%d; the log still holds %d onwards and may delete nothing it did not have to", g[0], g[1], st.First)
		}
	}
	// What Open makes of the killed writer's log: recovery, then the same questions, then the consumers.
	cfg := c.logConfig()
	cfg.Retention, cfg.MaxBytes, cfg.MaxRecords, cfg.ProtectedConsumers, cfg.NoMaintenance = -1, 0, 0, nil, true
	l, err := quill.Open(cfg)
	if err != nil {
		c.record("verify", "Open after the run: %v", err)
		return
	}
	defer l.Close()
	if l.Head() < st.Head {
		c.record("verify", "recovery moved the head back from %d to %d", st.Head, l.Head())
	}
	healthy("after recovery")
	after, _ := l.Stat()
	infos, err := l.Consumers()
	if err != nil {
		c.record("verify", "Consumers: %v", err)
	}
	for _, ci := range infos {
		var id int
		if _, err := fmt.Sscanf(ci.Name, "c%d-", &id); err != nil || !strings.HasSuffix(ci.Name, "-"+c.RunID) {
			continue
		}
		done, intent := lastWord(c.ctl("oracle-"+ci.Name), "done"), lastWord(c.ctl("oracle-"+ci.Name), "intent")
		switch {
		case ci.Err != nil:
			c.record("verify", "consumer %s: %v", ci.Name, ci.Err)
		case ci.Offset > l.Head()+1:
			c.record("verify", "consumer %s is at %d, above head+1 (%d)", ci.Name, ci.Offset, l.Head()+1)
		case intent > 0 && (ci.Offset < done || ci.Offset > intent):
			c.record("verify", "consumer %s is at %d; its last returned commit was %d and its last attempted %d", ci.Name, ci.Offset, done, intent)
		case ci.Protected != protected(id):
			c.record("verify", "consumer %s has protection %v, want %v", ci.Name, ci.Protected, protected(id))
		case ci.Protected && !c.capped() && after.First > ci.Offset:
			c.record("verify", "protected consumer %s is at %d but retention deleted up to %d", ci.Name, ci.Offset, after.First-1)
		}
	}
}

// collect adds the roles' counters and the violations to the summary.
func (c *config) collect(s *summary, o *counters) {
	for k, v := range o.m {
		s.Counters["chaos."+k] = v
	}
	files, _ := filepath.Glob(c.ctl("stats-*.json"))
	for _, name := range files {
		var m map[string]int64
		if b, err := os.ReadFile(name); err == nil && json.Unmarshal(b, &m) == nil {
			role := strings.Split(filepath.Base(name), "-")[1]
			for k, v := range m {
				s.Counters[role+"."+k] += v
			}
		}
	}
	b, _ := os.ReadFile(c.ctl("violations"))
	s.Violations = []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line != "" {
			s.Violations = append(s.Violations, line)
		}
	}
}

// selftestCmd runs the harness against itself: a clean run must pass, and each injected fault
// must make a run fail with the violation that fault causes. It exits 1 if any verdict is wrong.
func selftestCmd(args []string) int {
	c := &config{}
	if code := parseFlags("selftest", c, args); code != 0 {
		return code
	}
	// No operator and no pruner: the faults are in the roles and the verifier. A fault must be
	// reported by the check named in want, so skipquiet and ttl are found by verify, not inline.
	c.Retention, c.HugeEvery, c.Quill = -1, 0, ""
	bad := 0
	for _, t := range []struct{ fault, want string }{
		{"", ""},
		{"skip", "not contiguous"},
		{"skipquiet", "verify: obs-"},
		{"ackearly", "acknowledged"},
		{"protected", ""}, // the control of ttl: the same run, protection holds
		{"ttl", "verify: protected consumer"},
	} {
		timed := t.fault == "protected" || t.fault == "ttl"
		if timed && c.Quick {
			continue
		}
		// The ttl fault needs a frozen consumer's segments to be deleted inside the run, which
		// depends on the machine keeping up: it gets three attempts (a different seed each), and
		// the attempt that found it is printed. The clean runs and the other faults get one.
		attempts := 1
		if t.fault == "ttl" {
			attempts = 3
		}
		found := false
		for a := 0; a < attempts && !found; a++ {
			run := *c
			run.Fault, run.Seed = t.fault, c.Seed+int64(a)
			if timed {
				run.Retention, run.Segment, run.Duration = time.Second, 16<<10, max(c.Duration, 14*time.Second)
				run.Consumers = max(c.Consumers, 1)
			}
			if t.fault == "protected" {
				run.Fault = ""
			}
			s := execute(&run)
			found = t.want == "" && s.OK
			for _, v := range s.Violations {
				found = found || t.want != "" && strings.Contains(v, t.want)
			}
			fmt.Fprintf(os.Stderr, "selftest fault=%q attempt %d/%d: ok=%v violations=%d detected=%v\n", t.fault, a+1, attempts, s.OK, len(s.Violations), found)
		}
		if !found {
			bad++
		}
	}
	if bad > 0 {
		fmt.Fprintf(os.Stderr, "selftest: FAILED, %d verdicts wrong\n", bad)
		return 1
	}
	fmt.Fprintln(os.Stderr, "selftest: ok, every injected fault was reported and the clean run passed")
	return 0
}
