package quill

import (
	"bytes"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

func TestOpenWithNoMaintenanceNeitherWritesNorRewritesThePolicy(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.Retention = time.Hour; c.MaxRecords = 77 })
	appendN(t, l, 3)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	want := fileBytes(t, s, "/log/quill.policy")

	l = openSim(t, s, func(c *Config) { c.NoMaintenance = true; c.Retention = -1; c.SegmentBytes = 1 << 20 })
	appendN(t, l, 3)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if got := fileBytes(t, s, "/log/quill.policy"); !bytes.Equal(got, want) {
		t.Fatalf("policy rewritten under NoMaintenance:\n%s\nwant\n%s", got, want)
	}

	// A fresh log opened with NoMaintenance has no policy until a writer opens it.
	s2 := fsx.NewSim()
	l = openSim(t, s2, func(c *Config) { c.NoMaintenance = true })
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Lstat("/log/quill.policy"); err == nil {
		t.Fatal("NoMaintenance created quill.policy")
	}
}
