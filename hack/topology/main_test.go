package main

import (
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
	if !s.OK || s.Verified == 0 || s.Counters["chaos.kills_writer"] == 0 {
		t.Fatalf("run failed or did nothing: %+v", s)
	}
}
