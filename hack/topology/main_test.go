package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// TestMain lets the harness start its roles as child processes of the test binary: run and
// selftest re-execute os.Executable with a role name, which here is this binary.
func TestMain(m *testing.M) {
	if os.Getenv("TOPOLOGY_AS_MAIN") == "1" {
		main()
		return
	}
	os.Setenv("TOPOLOGY_AS_MAIN", "1")
	os.Exit(m.Run())
}

// buildQuill builds the command line tool the operator and the pruner drive.
func buildQuill(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "quill")
	cmd := exec.Command("go", "build", "-o", bin, "../../cmd/quill")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the quill command: %v\n%s", err, out)
	}
	return bin
}

// The payload generator covers every size class, and the check accepts what it generates and
// refuses a payload that differs from it in any one byte, in length, in sequence or in run.
func TestPayloadsAreSelfVerifying(t *testing.T) {
	c := &config{Run: 0x1234}
	sizes := map[int]bool{}
	for seq := uint64(1); seq <= 3000; seq++ {
		p := c.payload(seq)
		sizes[len(p)] = true
		if err := c.check(seq, p); err != nil {
			t.Fatalf("record %d: %v", seq, err)
		}
		if len(p) > 64<<10 { // a few big ones are enough
			continue
		}
		for _, i := range []int{0, len(p) / 3, len(p) - 1} {
			if i < 0 || i >= len(p) {
				continue
			}
			q := slices.Clone(p)
			q[i] ^= 0x40
			if c.check(seq, q) == nil {
				t.Fatalf("record %d with byte %d flipped passes", seq, i)
			}
		}
		// An empty or one-byte payload carries no identity beyond its length: only a header does.
		if c.check(seq, append(slices.Clone(p), 0)) == nil || len(p) >= hdr && (c.check(seq+1, p) == nil || (&config{Run: 0x1235}).check(seq, p) == nil) {
			t.Fatalf("record %d passes as something else", seq)
		}
	}
	for _, n := range sizeClasses {
		near := sizes[n] // the small classes are exact, the others jitter by 16
		for d := range 33 {
			near = near || n >= 100 && sizes[n+d-16]
		}
		if !near {
			t.Errorf("no payload near %d bytes in 3000", n)
		}
	}
	huge := &config{Run: 1, HugeEvery: 1} // every record near the maximum
	if p := huge.payload(5); len(p) <= quill.MaxPayloadBytes-64 || huge.check(5, p) != nil {
		t.Errorf("a maximum-size record: length %d, %v", len(p), huge.check(5, p))
	}
}

// The harness must be able to fail: each injected fault has to be reported, and a clean run has
// to pass. This is `topology selftest -quick`: the faults that do not depend on retention
// timing. The full `topology selftest` (adding protected and ttl, which need a frozen consumer's
// segments deleted inside the run) is run on demand, not in the gate, because it can flake on a
// loaded machine.
func TestSelfTestDetectsInjectedFaults(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a dozen processes")
	}
	if code := selftestCmd([]string{"-dir", t.TempDir(), "-duration", "4s", "-quick"}); code != 0 {
		t.Fatalf("selftest exited %d", code)
	}
}

// A run under retention and caps: gaps are legal for readers and consumers and the verifier
// has to accept them, and everything else has to hold.
func TestRunWithRetentionAndCaps(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a dozen processes")
	}
	c := &config{Dir: t.TempDir(), Quill: buildQuill(t), Duration: 6 * time.Second, Readers: 2, Consumers: 3, Segment: 32 << 10,
		Retention: 2 * time.Second, MaxBytes: 256 << 10, KillEvery: 500 * time.Millisecond, Pace: 5 * time.Millisecond, Seed: 7, NoUnregister: true}
	s := execute(c)
	if !didWork(s) {
		t.Fatalf("run failed or did nothing: %+v", s)
	}
}

// didWork says whether a run that verified did anything: writers were killed and acknowledged
// records, and the log either holds records that verify read or was emptied after them. A run
// under a 2 s retention and a cap can end with the log empty (First is Head+1), which verify
// accepts, so an empty log is not a run that did nothing.
func didWork(s *summary) bool {
	return s.OK && s.Acked > 0 && s.Counters["chaos.kills_writer"] > 0 && (s.Verified > 0 || s.First == s.Head+1)
}

func TestDidWork(t *testing.T) {
	kills := map[string]int64{"chaos.kills_writer": 6}
	for _, c := range []struct {
		name string
		s    summary
		want bool
	}{
		{"records verified", summary{OK: true, Head: 6326, First: 6271, Acked: 6326, Verified: 56, Counters: kills}, true},
		{"emptied by retention, as a tmpfs run did", summary{OK: true, Head: 7888, First: 7889, Acked: 7888, Counters: kills}, true},
		{"records in the log that verify did not read", summary{OK: true, Head: 7888, First: 7800, Acked: 7888, Counters: kills}, false},
		{"nothing acknowledged", summary{OK: true, Head: 0, First: 1, Counters: kills}, false},
		{"no writer was killed", summary{OK: true, Head: 10, First: 1, Acked: 10, Verified: 10}, false},
		{"a violation", summary{Head: 10, First: 1, Acked: 10, Verified: 10, Counters: kills}, false},
	} {
		if got := didWork(&c.s); got != c.want {
			t.Errorf("%s: didWork = %v, want %v", c.name, got, c.want)
		}
	}
}

// A chaos kill that lands while the child is between its exit and its "ready" line is dropped
// by signal, and kills_<role> must not count it: the summary's figure is the number of SIGKILLs
// sent, which restarts_<role> can then be compared with.
func TestKillCountsOnlySentSignals(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	o := &orch{n: &counters{m: map[string]int64{}}}
	p := &proc{role: "writer", cmd: cmd} // started, not yet ready

	o.kill(p)
	if got := o.n.m["kills_writer"]; got != 0 {
		t.Fatalf("kills_writer = %d after a kill of a child that is not ready, want 0", got)
	}
	if p.hit {
		t.Fatal("a dropped kill marked the child as hit")
	}

	p.up = true
	o.kill(p)
	if got := o.n.m["kills_writer"]; got != 1 {
		t.Fatalf("kills_writer = %d after a kill of a ready child, want 1", got)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("the ready child was not killed")
	}

	p.stopped = true
	o.kill(p)
	if got := o.n.m["kills_writer"]; got != 1 {
		t.Fatalf("kills_writer = %d after a kill of a frozen role, want still 1", got)
	}
}

// A run that is cancelled early (a role recorded a violation) must stop the stall scheduler at
// once. It used to loop on wall-clock time alone, with every sleep returning at once on the
// cancelled context, so it spun at full CPU until the original deadline was near.
func TestStallsReturnsWhenRunIsCancelled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	cancel() // the run is over long before its deadline
	o := &orch{
		c:     &config{Retention: 2 * time.Second, Stall: time.Second, Seed: 1},
		ctx:   ctx,
		procs: []*proc{{role: "consumer", id: 0}},
		n:     &counters{m: map[string]int64{}},
	}
	done := make(chan struct{})
	go func() {
		o.stalls()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stalls was still running 2s after the run was cancelled")
	}
}
