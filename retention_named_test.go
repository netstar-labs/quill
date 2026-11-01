package quill

import (
	"strings"
	"testing"
	"time"
)

// A consumer file that cannot be read makes the pass protect every segment from the age sweep,
// and the pass says so by name (A5-36, from A5-17): without the line the sweep is suspended
// silently for as long as the file stays broken.
func TestAPassWarnsOfAConsumerFileItCannotRead(t *testing.T) {
	var cap logCapture
	s, clk, l := retentionLog(t, func(c *Config) { c.Logger = cap.logger(); c.Retention = time.Hour })
	defer l.Close()
	fillSegments(t, l, clk, 4)
	clk.Advance(2 * time.Hour)
	writeFile(t, s, "/log/consumers/broken", []byte("garbage"))
	if err := l.Prune(true); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.Stat(); st.Segments != 5 {
		t.Fatalf("the sweep ran past an unreadable consumer file: %+v", st)
	}
	if !cap.has("consumer file could not be read") || !cap.has("broken") {
		t.Fatalf("no Warn names the unreadable consumer file: %q", cap.msgs)
	}
}

// A deletion that passes a consumer's offset is logged at Warn with the consumer's name and
// offset, whether the consumer is protected (a cap overrides it) or not (A5-18): the Info
// line for the segment names nobody, and Dropped is only a count in the writer's memory.
func TestADeletionPastAConsumersOffsetIsLoggedWithItsName(t *testing.T) {
	var cap logCapture
	s, clk, l := retentionLog(t, func(c *Config) { c.Logger = cap.logger(); c.MaxRecords = 8 })
	defer l.Close()
	plain, err := l.Consumer("plain", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()
	prot, err := l.Consumer("billing", ConsumerOptions{Protect: ProtectOn})
	if err != nil {
		t.Fatal(err)
	}
	defer prot.Close()
	if err := l.Prune(false); err != nil { // the seals below judge Dropped by the offsets this pass reads
		t.Fatal(err)
	}
	fillSegments(t, l, clk, 6)
	clk.Advance(time.Minute)
	if err := l.Prune(false); err != nil {
		t.Fatal(err)
	}
	if st, _ := l.Stat(); st.First <= 1 || st.Dropped == 0 {
		t.Fatalf("the cap deleted nothing: %+v (files %v)", st, segmentFiles(t, s, "/log"))
	}
	var warns []string
	cap.mu.Lock()
	for _, m := range cap.msgs {
		if strings.Contains(m, "level=WARN") && strings.Contains(m, "consumer=") && strings.Contains(m, "lost records") {
			warns = append(warns, m)
		}
	}
	cap.mu.Unlock()
	for _, name := range []string{"plain", "billing"} {
		found := false
		for _, m := range warns {
			found = found || strings.Contains(m, "consumer="+name+" ")
		}
		if !found {
			t.Errorf("no Warn names consumer %q: %q", name, warns)
		}
	}
}
