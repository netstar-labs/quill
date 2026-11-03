package quill

import (
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A consumer file that an unregister removes after the directory was listed is not damage
// and not a consumer: Consumers omits it, Check stays healthy and a prune pass does not treat
// the consumer view as broken. A file that is really unreadable is still reported.
func TestAConsumerFileRemovedAfterTheListingIsNotDamageNorAConsumer(t *testing.T) {
	const victim = "/log/consumers/op"
	setup := func(t *testing.T) (*fsx.Sim, *Log, *Reader) {
		t.Helper()
		s := fsx.NewSim()
		l := openSim(t, s, nil)
		t.Cleanup(func() { _ = l.Close() })
		appendN(t, l, 3)
		mustFlush(t, l)
		c, err := l.Consumer("op", ConsumerOptions{Protect: ProtectOn})
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
		r := openReaderSim(t, s)
		t.Cleanup(func() { _ = r.Close() })
		return s, l, r
	}
	t.Run("Check", func(t *testing.T) {
		s, _, r := setup(t)
		removeOnOpen(s, victim, 0)
		res, err := r.Check()
		if err != nil || !res.Healthy || len(res.Problems) != 0 {
			t.Fatalf("Check: %+v, %v", res, err)
		}
	})
	t.Run("Reader.Consumers", func(t *testing.T) {
		s, _, r := setup(t)
		removeOnOpen(s, victim, 0)
		list, err := r.Consumers()
		if err != nil || len(list) != 0 {
			t.Fatalf("Consumers: %+v, %v", list, err)
		}
	})
	t.Run("the prune pass's view", func(t *testing.T) {
		s, l, _ := setup(t)
		removeOnOpen(s, victim, 0)
		cv := l.consumers(l.cfg.clock())
		if cv.broken || len(cv.offsets) != 0 {
			t.Fatalf("view: %+v", cv)
		}
	})
	t.Run("an unreadable file is still reported", func(t *testing.T) {
		s, l, r := setup(t)
		s.Inject = func(op fsx.Op, name string) error {
			if op == fsx.OpOpen && name == victim {
				return syscall.EIO
			}
			return nil
		}
		res, err := r.Check()
		if err != nil || res.Healthy {
			t.Fatalf("Check: %+v, %v", res, err)
		}
		list, err := r.Consumers()
		if err != nil || len(list) != 1 || list[0].Err == nil {
			t.Fatalf("Consumers: %+v, %v", list, err)
		}
		if cv := l.consumers(l.cfg.clock()); !cv.broken {
			t.Fatalf("view: %+v", cv)
		}
	})
}
