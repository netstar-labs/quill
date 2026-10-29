package quill

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

func TestAPruneThatDropsAHoleNeitherRacesReadersNorRewritesAStatSnapshot(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	l := openRetention(t, s, clk, nil)
	fillSegments(t, l, clk, 6)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	for _, i := range []uint64{1, 3} { // records 5..8 and 13..16
		if err := s.Remove("/log/" + segmentName(i)); err != nil {
			t.Fatal(err)
		}
	}
	l = openRetention(t, s, clk, func(c *Config) { c.MaxRecords = 16 })
	defer l.Close()
	want := []Hole{{From: 5, To: 8}, {From: 13, To: 16}}
	st, err := l.Stat()
	if err != nil || !slices.Equal(st.Holes, want) {
		t.Fatalf("setup: holes %+v %v", st.Holes, err)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = l.Read(10, 1)
				}
			}
		}()
	}
	time.Sleep(2 * time.Millisecond)
	err = l.Prune(true)
	close(stop)
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}

	if st.Holes[0] != want[0] || st.Holes[1] != want[1] {
		t.Fatalf("a Stat taken before the prune changed under the caller: %+v", st.Holes)
	}
	after, err := l.Stat()
	if err != nil || !slices.Equal(after.Holes, []Hole{{From: 13, To: 16}}) {
		t.Fatalf("holes after the prune: %+v %v", after.Holes, err)
	}
}
