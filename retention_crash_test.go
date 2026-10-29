package quill

import (
	"errors"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A prune pass that deletes several sealed segments must not leave a hole in the middle of
// the retained records when power is lost part way (scope.md, model line 3). The simulator
// lets each unsynced unlink persist on its own (decision 61), so a later unlink could
// persist without an earlier one unless the directory is synced between them.
func TestAMultiSegmentPruneLeavesNoHoleAfterACrash(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 9))
	holes := 0
	for at := 1; ; at++ {
		fired := false
		for k := 0; k < 40; k++ {
			mask := rng.Uint64()
			s := fsx.NewSim()
			clk := newClock()
			s.Clock = clk.Now
			cfg := Config{Dir: "/log", FlushInterval: time.Hour, FlushBytes: 1 << 20, SegmentBytes: 100,
				Retention: time.Hour, NoMaintenance: true, fs: s, clock: clk.Now}
			l, err := Open(cfg)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 12; i++ {
				appendN(t, l, 3)
				mustFlush(t, l)
			}
			if err := l.Roll(); err != nil {
				t.Fatal(err)
			}
			s.Crash(func(int, int) bool { return true }) // everything so far is durable
			_ = l.Close()
			if l, err = Open(cfg); err != nil {
				t.Fatal(err)
			}
			clk.Advance(2 * time.Hour) // every sealed segment is now expired
			count, crashed := 0, false
			s.CrashKeep = func(i, _ int) bool { return mask&(1<<uint(i%64)) != 0 }
			s.Inject = func(fsx.Op, string) error {
				count++
				if crashed {
					return errors.New("dead")
				}
				if count == at {
					crashed, fired = true, true
					return fsx.ErrCrash
				}
				return nil
			}
			_ = l.Prune(true)
			s.Inject = nil
			if !crashed {
				break
			}
			l2, err := Open(Config{Dir: "/log", FlushInterval: time.Hour, SegmentBytes: 100, Retention: -1, fs: s})
			if err != nil {
				t.Fatalf("crash before operation %d, mask %x: reopen: %v", at, mask, err)
			}
			if len(l2.holes) != 0 {
				holes++
				if holes <= 2 {
					names, _ := s.ReadDir("/log")
					t.Errorf("crash before operation %d, mask %x: holes %v, files %v", at, mask, l2.holes, names)
				}
			}
			_ = l2.Close()
		}
		if !fired {
			break
		}
	}
	if holes > 0 {
		t.Errorf("%d crash states left a hole", holes)
	}
}
