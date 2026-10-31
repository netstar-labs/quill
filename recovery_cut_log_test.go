package quill

import (
	"bytes"
	"strings"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// A writable Open that cuts the newest segment past the marker must say so, and say whether
// the first rejected frame was incomplete (a routine torn tail) or complete-length but failing
// its checks (damage, which may have been an acknowledged record).
func TestOpenLogsTheCutItMakesToTheNewestSegment(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(data []byte) []byte
		want   []string
		cut    bool
	}{
		{"clean", func(d []byte) []byte { return d }, nil, false},
		{"torn", func(d []byte) []byte { return d[:len(d)-3] }, []string{"kind=torn", "bytes="}, true},
		{"damaged", func(d []byte) []byte { d = bytes.Clone(d); d[len(d)-1] ^= 0xff; return d },
			[]string{"kind=damaged", "bytes="}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, nil)
			appendN(t, l, 5)
			mustFlush(t, l)
			stale := fileBytes(t, s, "/log/"+tailName) // the marker names record 5
			appendN(t, l, 5)
			mustFlush(t, l)
			_ = l.Close()
			seg := "/log/" + segmentName(0)
			data := fileBytes(t, s, seg)
			writeFile(t, s, "/log/"+tailName, stale)
			writeFile(t, s, seg, tc.mutate(data))
			var lc logCapture
			l = openSim(t, s, func(c *Config) { c.Logger = lc.logger() })
			defer l.Close()
			if !tc.cut {
				if len(lc.msgs) != 0 {
					t.Fatalf("logged without a cut: %q", lc.msgs)
				}
				return
			}
			if !lc.has("cut the newest segment") || !lc.has("segment="+seg) {
				t.Fatalf("the cut was not logged with the segment: %q", lc.msgs)
			}
			for _, w := range tc.want {
				if !lc.has(w) {
					t.Errorf("log line lacks %q: %q", w, strings.Join(lc.msgs, ""))
				}
			}
		})
	}
}
