package quill

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// consumersProbeFS counts listings of consumers/ and how many ran while the log's lock was
// held. The writer must never read the consumer files under that lock (scope.md: a consumer
// must not block the writer): with many files in consumers/ the stall grows with their number.
type consumersProbeFS struct {
	fsx.FS
	log         atomic.Pointer[Log]
	reads, held atomic.Int32
}

func (p *consumersProbeFS) ReadDir(name string) ([]string, error) {
	if l := p.log.Load(); l != nil && strings.HasSuffix(name, "/"+consumersDir) {
		p.reads.Add(1)
		if l.mu.TryLock() {
			l.mu.Unlock()
		} else {
			p.held.Add(1)
		}
	}
	return p.FS.ReadDir(name)
}

func TestSealingNeverReadsTheConsumerFilesUnderTheLogsLockAndStillCountsDropped(t *testing.T) {
	for _, tc := range []struct {
		name string
		mut  func(*Config)
		drop bool
	}{
		{"no cap", nil, false},
		{"MaxRecords", func(c *Config) { c.MaxRecords = 5 }, true},
		{"MaxBytes", func(c *Config) { c.MaxBytes = 200 }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			clk := newClock()
			p := &consumersProbeFS{FS: s}
			s.Clock = clk.Now
			l := openSim(t, s, func(c *Config) {
				c.fs = p
				c.clock = clk.Now
				c.SegmentBytes = 60 // four records of payloadFor per segment
				c.Retention = time.Hour
				c.NoMaintenance = true
				if tc.mut != nil {
					tc.mut(c)
				}
			})
			defer l.Close()
			slow, err := l.Consumer("slow", ConsumerOptions{}) // stays at the first record
			if err != nil {
				t.Fatal(err)
			}
			defer slow.Close()
			if err := l.Prune(false); err != nil { // a prune pass is what tells the writer the offsets
				t.Fatal(err)
			}
			p.log.Store(l)
			p.reads.Store(0)
			fillSegments(t, l, clk, 8) // inline flushes seal; the caps then delete
			appendN(t, l, 1)
			if err := l.Roll(); err != nil { // Roll seals too
				t.Fatal(err)
			}
			if n := p.held.Load(); n != 0 {
				t.Fatalf("%d listings of consumers/ ran with the log's lock held", n)
			}
			if n := p.reads.Load(); n != 0 {
				t.Fatalf("%d listings of consumers/ outside a prune pass: sealing must not list them at all", n)
			}
			st, _ := l.Stat()
			if tc.drop && st.Dropped == 0 {
				t.Fatalf("the caps deleted segments under a consumer at the first record, Dropped is 0: %+v", st)
			}
			if !tc.drop && st.Dropped != 0 {
				t.Fatalf("nothing was deleted, Dropped is %d", st.Dropped)
			}
		})
	}
}
