package quill

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// aheadRecords captures structured records, so a test can check the attributes of a Warn.
type aheadRecords struct {
	mu   sync.Mutex
	recs []slog.Record
}

func (c *aheadRecords) Enabled(context.Context, slog.Level) bool { return true }
func (c *aheadRecords) WithAttrs([]slog.Attr) slog.Handler       { return c }
func (c *aheadRecords) WithGroup(string) slog.Handler            { return c }
func (c *aheadRecords) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recs = append(c.recs, r.Clone())
	return nil
}

// aheadWarns returns the attributes of every record whose message ends in the rewind advice.
func (c *aheadRecords) aheadWarns() []map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []map[string]any
	for _, r := range c.recs {
		if !strings.HasSuffix(r.Message, "rewind it with quill commit") {
			continue
		}
		m := map[string]any{"msg": r.Message, "level": r.Level}
		r.Attrs(func(a slog.Attr) bool { m[a.Key] = a.Value.Any(); return true })
		out = append(out, m)
	}
	return out
}

// aheadLog makes a log with three records (head 3), closes it and returns its simulator and ID, so a
// test can plant consumer files and reopen it.
func aheadLog(t *testing.T) (*fsx.Sim, string) {
	t.Helper()
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	appendN(t, l, 3)
	id := l.ID()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return s, id
}

func reopenWithLogger(t *testing.T, s *fsx.Sim, logger *slog.Logger) {
	t.Helper()
	l := openSim(t, s, func(c *Config) { c.Logger = logger })
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
}

// A log restored to an older state with a consumer ahead of the head gives exactly one Warn from the
// writer's Open, naming the consumer, its offset and the head; a consumer at Head+1 and one behind
// give none.
func TestOpenWarnsOnceForAConsumerAheadOfTheHead(t *testing.T) {
	s, id := aheadLog(t)
	writeFile(t, s, "/log/consumers/ahead", encodeConsumer(consumerState{ID: logID(id), Next: 9, Protect: true}))
	writeFile(t, s, "/log/consumers/edge", encodeConsumer(consumerState{ID: logID(id), Next: 4}))
	writeFile(t, s, "/log/consumers/behind", encodeConsumer(consumerState{ID: logID(id), Next: 2}))
	rec := &aheadRecords{}
	reopenWithLogger(t, s, slog.New(rec))
	got := rec.aheadWarns()
	if len(got) != 1 {
		t.Fatalf("%d warnings, want 1: %v", len(got), got)
	}
	w := got[0]
	if w["level"] != slog.LevelWarn || w["consumer"] != "ahead" || w["offset"] != uint64(9) || w["head"] != uint64(3) {
		t.Fatalf("attributes %v", w)
	}
	if !strings.Contains(w["msg"].(string), "ahead of the head") {
		t.Fatalf("message %q", w["msg"])
	}
}

// A consumer at Head+1 or behind gives no warning.
func TestOpenDoesNotWarnForAConsumerAtOrBehindTheNextSequence(t *testing.T) {
	s, id := aheadLog(t)
	writeFile(t, s, "/log/consumers/edge", encodeConsumer(consumerState{ID: logID(id), Next: 4}))
	writeFile(t, s, "/log/consumers/behind", encodeConsumer(consumerState{ID: logID(id), Next: 1, Protect: true}))
	rec := &aheadRecords{}
	reopenWithLogger(t, s, slog.New(rec))
	if got := rec.aheadWarns(); len(got) != 0 {
		t.Fatalf("warnings %v", got)
	}
}

// Two ahead consumers give two warnings, one each.
func TestOpenWarnsOncePerAheadConsumer(t *testing.T) {
	s, id := aheadLog(t)
	writeFile(t, s, "/log/consumers/a", encodeConsumer(consumerState{ID: logID(id), Next: 9}))
	writeFile(t, s, "/log/consumers/b", encodeConsumer(consumerState{ID: logID(id), Next: 5, Protect: true}))
	rec := &aheadRecords{}
	reopenWithLogger(t, s, slog.New(rec))
	got := rec.aheadWarns()
	if len(got) != 2 {
		t.Fatalf("%d warnings, want 2: %v", len(got), got)
	}
	names := map[any]bool{got[0]["consumer"]: true, got[1]["consumer"]: true}
	if !names["a"] || !names["b"] {
		t.Fatalf("names %v", got)
	}
}

// A corrupt consumer file, or one for another log, is not this warning's business.
func TestOpenAheadWarningIgnoresUnreadableAndForeignConsumers(t *testing.T) {
	s, _ := aheadLog(t)
	writeFile(t, s, "/log/consumers/corrupt", []byte("not a consumer file\n"))
	writeFile(t, s, "/log/consumers/foreign", encodeConsumer(consumerState{ID: logID("0123456789abcdef0123456789abcdef"), Next: 9}))
	rec := &aheadRecords{}
	reopenWithLogger(t, s, slog.New(rec))
	if got := rec.aheadWarns(); len(got) != 0 {
		t.Fatalf("warnings %v", got)
	}
}

// With no Logger, Open succeeds with an ahead consumer (nothing to write to, nothing to fail).
func TestOpenAheadWarningAbsentWithoutALogger(t *testing.T) {
	s, id := aheadLog(t)
	writeFile(t, s, "/log/consumers/ahead", encodeConsumer(consumerState{ID: logID(id), Next: 9}))
	reopenWithLogger(t, s, nil)
}
