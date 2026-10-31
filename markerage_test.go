package quill

import (
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// An empty log has the zero marker, equal to what Open would publish, so Open's forced
// republish used to be skipped and the writer's Stats.MarkerAge read 0 ("just written")
// until the first real publish, however long ago Open ran.
func TestWriterMarkerAgeOfAnEmptyLogCountsFromOpen(t *testing.T) {
	s := fsx.NewSim()
	clk := newClock()
	s.Clock = clk.Now
	for _, pass := range []string{"new", "reopened"} {
		l := openSim(t, s, func(c *Config) { c.clock = clk.Now })
		clk.Advance(30 * time.Minute)
		st, err := l.Stat()
		if err != nil || st.Head != 0 {
			t.Fatalf("%s: head %d, %v", pass, st.Head, err)
		}
		if st.MarkerAge != 30*time.Minute {
			t.Fatalf("%s empty log: MarkerAge %v, want 30m0s (the marker was written at Open)", pass, st.MarkerAge)
		}
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
