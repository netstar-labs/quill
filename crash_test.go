package quill

import (
	"context"
	"errors"
	"math/rand/v2"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// T1, phase-2 part: the crash property against the simulator. A scenario of appends,
// flushes, syncs, rotation and Roll runs with a crash injected before every file operation,
// and for each crash point with a sample of the subsets of unsynced items that persist.
// After recovery: Open succeeds; the log is a contiguous prefix; it holds every record whose
// Flush or Sync returned; it may hold more; nothing durable is lost or duplicated; sequences
// are reassigned only for records that never became durable.

type scenarioState struct {
	appended map[uint64][]byte // every sequence handed out, with its payload
	acked    uint64            // the highest sequence whose Flush or Sync returned nil
}

// runScenario drives the script against the log until the crash fires. A step whose call
// fails after the crash is expected; a failure before it is a test failure.
func runScenario(t *testing.T, l *Log, crashed *bool) scenarioState {
	t.Helper()
	st := scenarioState{appended: map[uint64][]byte{}}
	add := func(n int) {
		for i := 0; i < n; i++ {
			if *crashed {
				return
			}
			seq := l.nextSeqForTest()
			p := payloadFor(seq)
			got, err := l.Append(p)
			if err != nil {
				if *crashed {
					return
				}
				t.Fatalf("append: %v", err)
			}
			if got != seq {
				t.Fatalf("append assigned %d, expected %d", got, seq)
			}
			st.appended[got] = p
		}
	}
	flush := func() {
		if *crashed {
			return
		}
		if err := l.Flush(); err == nil {
			st.acked = l.Head()
		} else if !*crashed {
			t.Fatalf("flush: %v", err)
		}
	}
	sync := func() {
		if *crashed {
			return
		}
		seq := l.nextSeqForTest() - 1
		if err := l.Sync(context.Background(), seq); err == nil {
			st.acked = max(st.acked, seq)
		} else if !*crashed {
			t.Fatalf("sync: %v", err)
		}
	}
	roll := func() {
		if *crashed {
			return
		}
		if err := l.Roll(); err == nil {
			st.acked = l.Head()
		} else if !*crashed {
			t.Fatalf("roll: %v", err)
		}
	}
	add(3)
	flush()
	add(2)
	sync()
	add(6) // rotates (SegmentBytes 512)
	flush()
	roll()
	add(4)
	flush()
	add(2)
	roll()
	add(3)
	sync()
	add(1)
	return st
}

func checkRecovered(t *testing.T, s *fsx.Sim, st scenarioState, label string) {
	t.Helper()
	l, err := Open(simConfig(s, "/log"))
	if err != nil {
		t.Fatalf("%s: reopen: %v", label, err)
	}
	defer l.Close()
	recs, err := readAll(l)
	if err != nil {
		t.Fatalf("%s: scan: %v", label, err)
	}
	if len(l.holes) != 0 {
		t.Fatalf("%s: holes %+v", label, l.holes)
	}
	for i, r := range recs {
		if r.Seq != uint64(i+1) {
			t.Fatalf("%s: not a contiguous prefix: %v", label, seqsOf(recs))
		}
		want, ok := st.appended[r.Seq]
		if !ok || string(want) != string(r.Payload) {
			t.Fatalf("%s: seq %d holds %q, appended %q", label, r.Seq, r.Payload, want)
		}
	}
	if uint64(len(recs)) < st.acked {
		t.Fatalf("%s: acknowledged up to %d, recovered %d records", label, st.acked, len(recs))
	}
	if l.Head() != uint64(len(recs)) {
		t.Fatalf("%s: head %d, records %d", label, l.Head(), len(recs))
	}
	// The next sequence continues from the recovered head.
	seq, err := l.Append([]byte("after"))
	if err != nil || seq != l.Head()+1 {
		t.Fatalf("%s: next sequence %d %v", label, seq, err)
	}
	if err := l.Flush(); err != nil {
		t.Fatalf("%s: flush after recovery: %v", label, err)
	}
}

func TestCrashPropertyAtEveryOperationAgainstTheSimulator(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	points := 0
	for at := 1; ; at++ {
		// Count the unsynced items at this point, then try subsets of them.
		var n int
		fired := false
		run := func(keep func(i, n int) bool) (scenarioState, *fsx.Sim) {
			s := fsx.NewSim()
			// Create the log durably first: a crash during creation is T11's subject, and
			// decision 62 allows it to leave a directory that is refused.
			created := openSim(t, s, nil)
			_ = created.Close()
			s.Crash(func(int, int) bool { return true })
			count := 0
			crashed := false
			s.CrashKeep = keep
			s.Inject = func(op fsx.Op, name string) error {
				count++
				if count == at {
					crashed, fired = true, true
					return fsx.ErrCrash
				}
				return nil
			}
			l, err := Open(simConfig(s, "/log"))
			if err != nil {
				if !crashed {
					t.Fatalf("open: %v", err)
				}
				s.Inject = nil
				return scenarioState{appended: map[uint64][]byte{}}, s // crashed inside Open
			}
			st := runScenario(t, l, &crashed)
			_ = l.Close()
			s.Inject = nil
			return st, s
		}
		st, s := run(func(i, total int) bool { n = total; return false })
		if !fired {
			break // the scenario ran to the end without a crash: every point was tried
		}
		points++
		checkRecovered(t, s, st, "keep none")
		masks := []uint64{1<<uint(n) - 1}
		if n <= 6 {
			for m := uint64(1); m < 1<<uint(n)-1; m++ {
				masks = append(masks, m)
			}
		} else {
			for i := 0; i < 6; i++ {
				masks = append(masks, rng.Uint64()&(1<<uint(n)-1))
			}
		}
		for _, m := range masks {
			st, s := run(func(i, _ int) bool { return m&(1<<uint(i)) != 0 })
			checkRecovered(t, s, st, "mask")
		}
	}
	if points < 30 {
		t.Fatalf("only %d crash points", points)
	}
}

func TestACrashDuringRecoveryItselfRecovers(t *testing.T) {
	// A log with a torn tail: recovery truncates and republishes. Crash before every
	// operation of that recovery; the next open must still succeed.
	for at := 1; ; at++ {
		s := fsx.NewSim()
		l := openSim(t, s, nil)
		appendN(t, l, 3)
		mustFlush(t, l)
		_ = l.Close()
		seg := "/log/" + segmentName(0)
		data := fileBytes(t, s, seg)
		writeFile(t, s, seg, append(data, image(4, 2, 0, func(i int) []byte { return payloadFor(uint64(4 + i)) })[:20]...))
		s.Crash(func(int, int) bool { return true })
		count := 0
		fired := false
		s.CrashKeep = func(i, n int) bool { return i%2 == 0 }
		s.Inject = func(op fsx.Op, name string) error {
			count++
			if count == at {
				fired = true
				return fsx.ErrCrash
			}
			return nil
		}
		l, err := Open(simConfig(s, "/log"))
		if err == nil {
			_ = l.Close()
		}
		s.Inject = nil
		if !fired {
			break
		}
		if err != nil && !errors.Is(err, fsx.ErrCrash) && !errors.Is(err, fsx.ErrDead) {
			t.Fatalf("crash at %d: open returned %v", at, err)
		}
		l = openSim(t, s, nil)
		recs, err := readAll(l)
		if err != nil || checkContiguous(t, recs, 1) != 3 {
			t.Fatalf("crash at %d: %v %v", at, seqsOf(recs), err)
		}
		_ = l.Close()
	}
}
