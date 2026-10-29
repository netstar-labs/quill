package quill

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// An Open that refuses with ErrCorrupt leaves the newest segment as it found it: the cut of
// an untrusted tail is made only once every check that can refuse has passed (decision 39;
// decision 64: after ErrCorrupt the repair is by hand, from what quill check reports).
func TestAnOpenRefusedAsCorruptDoesNotCutTheNewestSegment(t *testing.T) {
	stray := []byte{0xde, 0xad}
	cases := []struct {
		name   string
		damage func(t *testing.T, s *fsx.Sim, seg string) // seg is the newest segment, holding 2 frames
		want   string
	}{
		{"overlap with the previous segment", func(t *testing.T, s *fsx.Sim, seg string) {
			// A copy of segment 0 in place of the newest segment, the marker's offset at its
			// end so it verifies whole below it, and two stray bytes past that.
			copy0 := fileBytes(t, s, "/log/"+segmentName(0))
			writeFile(t, s, seg, append(append([]byte(nil), copy0...), stray...))
			tl := readTail(t, s, "/log")
			tl.Off = uint64(len(copy0))
			writeFile(t, s, "/log/"+tailName, encodeTail(tl))
		}, "inside the one before"},
		{"last valid sequence below the marker", func(t *testing.T, s *fsx.Sim, seg string) {
			writeFile(t, s, seg, append(fileBytes(t, s, seg), stray...))
			tl := readTail(t, s, "/log")
			tl.Seq += 5
			writeFile(t, s, "/log/"+tailName, encodeTail(tl))
		}, "is durable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := fsx.NewSim()
			l := openSim(t, s, func(c *Config) { c.SegmentBytes = 60 })
			for len(l.sealed) < 2 || l.active.frames == 0 {
				appendN(t, l, 1)
				mustFlush(t, l)
			}
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			ids := segmentFiles(t, s, "/log")
			seg := "/log/" + segmentName(ids[len(ids)-1])
			if len(ids) < 2 || len(fileBytes(t, s, seg)) == 0 {
				t.Fatalf("setup: want sealed segments and a newest one with frames, got %v", ids)
			}
			tc.damage(t, s, seg)
			before := fileBytes(t, s, seg)
			_, err := Open(simConfig(s, "/log"))
			if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("open: %v, want ErrCorrupt containing %q", err, tc.want)
			}
			if after := fileBytes(t, s, seg); !bytes.Equal(after, before) {
				t.Fatalf("the refused open changed the newest segment from %d to %d bytes", len(before), len(after))
			}
		})
	}
}
