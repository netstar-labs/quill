package quill

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A consumer that commits to the head while Check is scanning, and so is above the marker Check
// read first, is not a restored-log leftover: Commit refuses anything above Head+1, so the offset
// was legal when it was written. Check beside a live writer must not tell the operator to rewind
// it (decision 36), and a consumer that is ahead of the head as it stands now still gets the
// warning.
func TestCheckDoesNotWarnOfAConsumerThatCommittedToTheHeadDuringTheScan(t *testing.T) {
	for _, tc := range []struct {
		name     string
		next     uint64 // the consumer's offset, written during the scan; the head becomes 5
		wantWarn bool
	}{
		{"commits to the grown head plus one", 6, false},
		{"genuinely ahead of the grown head", 9, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
			t.Cleanup(func() { _ = l.Close() })
			appendN(t, l, 3)
			mustFlush(t, l)
			r := openReaderSim(t, s)
			t.Cleanup(func() { _ = r.Close() })
			active := "/log/" + segmentName(0)
			var fired atomic.Bool
			var hookErr atomic.Value
			s.Inject = func(op fsx.Op, name string) error {
				if op != fsx.OpOpen || name != active || !fired.CompareAndSwap(false, true) {
					return nil
				}
				// The writer grows the log and a consumer commits to Head+1, after Check
				// read the marker and before it lists the consumers.
				for i := 0; i < 2; i++ {
					if _, err := l.Append([]byte("x")); err != nil {
						hookErr.Store(err)
					}
				}
				if err := l.Flush(); err != nil {
					hookErr.Store(err)
				}
				f, err := s.OpenFile("/log/consumers/cc", fsx.O_RDWR|fsx.O_CREATE, 0o600)
				if err != nil {
					hookErr.Store(err)
					return nil
				}
				defer f.Close()
				if _, err := f.WriteAt(encodeConsumer(consumerState{ID: logID(l.ID()), Next: tc.next}), 0); err != nil {
					hookErr.Store(err)
				}
				return nil
			}
			res, err := r.Check()
			if err != nil {
				t.Fatal(err)
			}
			if e := hookErr.Load(); e != nil || !fired.Load() {
				t.Fatalf("hook: fired %v, error %v", fired.Load(), e)
			}
			if !res.Healthy {
				t.Fatalf("not healthy: %+v", res.Problems)
			}
			warned := false
			for _, p := range res.Problems {
				if p.Kind == "consumer" && strings.Contains(p.Detail, "ahead of the head") {
					warned = true
					if tc.wantWarn && !strings.Contains(p.Detail, "ahead of the head 5:") {
						t.Fatalf("the warning names a stale head: %q", p.Detail)
					}
				}
			}
			if warned != tc.wantWarn {
				t.Fatalf("warned %v, want %v: %+v", warned, tc.wantWarn, res.Problems)
			}
		})
	}
}
