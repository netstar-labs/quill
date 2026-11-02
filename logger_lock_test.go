package quill

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

// lockProbe is a slog.Handler that records whether the log's lock was held when its Enabled and
// Handle methods ran: 1 if every call saw it held, 2 if any call saw it free. Config.Logger documents that both run under the lock; this pins it.
type lockProbe struct {
	log                     *atomic.Pointer[Log]
	enabledHeld, handleHeld *atomic.Int32 // 1: the lock was held, 2: it was free
}

func (p lockProbe) note(into *atomic.Int32) {
	l := p.log.Load()
	if l == nil {
		return
	}
	if l.mu.TryLock() {
		l.mu.Unlock()
		into.Store(2) // sticky: one call with the lock free is enough to fail the test
		return
	}
	into.CompareAndSwap(0, 1)
}

func (p lockProbe) Enabled(context.Context, slog.Level) bool {
	p.note(p.enabledHeld)
	return true
}
func (p lockProbe) Handle(context.Context, slog.Record) error { p.note(p.handleHeld); return nil }
func (p lockProbe) WithAttrs([]slog.Attr) slog.Handler        { return p }
func (p lockProbe) WithGroup(string) slog.Handler             { return p }

// Config.Logger's handler, Enabled included, runs with the log's lock held: the documentation
// tells callers it must neither call the Log nor block, and this is the behaviour that says so.
func TestTheLoggerRunsUnderTheLogsLockAsDocumented(t *testing.T) {
	var lp atomic.Pointer[Log]
	var enabled, handled atomic.Int32
	_, clk, l := retentionLog(t, func(c *Config) {
		c.MaxRecords = 5 // every seal then deletes a segment, so the log has something to say
		c.Logger = slog.New(lockProbe{&lp, &enabled, &handled})
	})
	lp.Store(l)
	for i := 0; i < 3; i++ {
		appendN(t, l, 4)
		mustFlush(t, l)
		clk.Advance(time.Minute)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if handled.Load() == 0 {
		t.Fatal("the handler was never given a record")
	}
	if handled.Load() != 1 || enabled.Load() != 1 {
		t.Fatalf("lock held in Enabled=%d Handle=%d (1 held, 2 free); the documentation says both run under the lock",
			enabled.Load(), handled.Load())
	}
}
