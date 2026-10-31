package quill

import (
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A behind, silent protected consumer whose ConsumerTTL probe fails (the segment at its offset
// cannot be read, or its offset file cannot be stat'ed) is kept as live, which protects data,
// but the operator must be told that the TTL is not being evaluated. When the failure clears,
// the consumer goes stale as usual.
func TestAFailedConsumerTTLProbeIsLoggedAndTheConsumerStaysLive(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   fsx.Op
		hit  func(name string) bool // the n-th matching operation of the pass fails when n >= from
		from int
	}{
		{"segment read", fsx.OpOpen, func(n string) bool { return strings.HasSuffix(n, ".log") }, 1},
		{"offset file lstat", fsx.OpLstat, func(n string) bool { return strings.HasSuffix(n, "/consumers/slow") }, 2}, // the first lstat is the state read of list
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			clk := newClock()
			var lc logCapture
			l := openRetention(t, s, clk, func(c *Config) {
				c.ConsumerTTL = 30 * time.Minute
				c.Logger = lc.logger()
			})
			defer l.Close()
			fillSegments(t, l, clk, 4)
			c, err := l.Consumer("slow", ConsumerOptions{Protect: ProtectOn})
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Commit(5); err != nil {
				t.Fatal(err)
			}
			_ = c.Close()
			clk.Advance(time.Hour + 10*time.Minute)
			failing, hits := true, 0
			s.Inject = func(op fsx.Op, name string) error {
				if failing && op == tc.op && tc.hit(name) {
					if hits++; hits >= tc.from {
						return syscall.EACCES
					}
				}
				return nil
			}
			if err := l.Prune(true); err != nil {
				t.Fatal(err)
			}
			if st, _ := l.Stat(); len(st.StaleConsumers) != 0 {
				t.Fatalf("a consumer whose probe failed was marked stale: %+v", st)
			}
			if !lc.has("staleness probe failed") || !lc.has("slow") || !lc.has("permission denied") {
				t.Fatalf("the failed probe was not logged with the consumer and the error: %q", lc.msgs)
			}
			failing = false
			if err := l.Prune(true); err != nil {
				t.Fatal(err)
			}
			if st, _ := l.Stat(); len(st.StaleConsumers) != 1 || st.StaleConsumers[0] != "slow" {
				t.Fatalf("after the failure cleared: %+v", st)
			}
		})
	}
}
