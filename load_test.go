package quill

import (
	"bytes"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

func crcChecksum(b []byte) uint32 { return crc32.Checksum(b, crcTable) }

func verify(data []byte) (segInfo, error) {
	return verifySegment(bytes.NewReader(data), int64(len(data)), "")
}

func recover_(data []byte, durable int64) (segInfo, bool, error) {
	return recoverSegment(bytes.NewReader(data), int64(len(data)), durable, "")
}

// frameEnds returns the byte offset after each frame of a well-formed image.
func frameEnds(t *testing.T, data []byte) []int64 {
	t.Helper()
	var ends []int64
	for pos := 0; pos < len(data); {
		_, n, err := decodeFrame(data[pos:])
		if err != nil {
			t.Fatalf("frameEnds at %d: %v", pos, err)
		}
		pos += n
		ends = append(ends, int64(pos))
	}
	return ends
}

func TestVerifySegmentAcceptsAWholeSegmentAndIndexesIt(t *testing.T) {
	data := image(100, 5000, 1_700_000_000, func(i int) []byte { return bytes.Repeat([]byte{byte(i)}, 100) })
	info, err := verify(data)
	if err != nil {
		t.Fatal(err)
	}
	if info.first.Seq != 100 || info.last.Seq != 5099 || info.frames != 5000 || info.end != int64(len(data)) {
		t.Fatalf("info %+v", info)
	}
	if info.first.At != 1_700_000_000 || info.last.At != 1_700_004_999 {
		t.Fatalf("At %d %d", info.first.At, info.last.At)
	}
	if len(info.index) < len(data)/indexStride || len(info.index) > len(data)/indexStride+2 {
		t.Fatalf("%d anchors for %d bytes", len(info.index), len(data))
	}
	if info.index[0].seq != 100 || info.index[0].pos != 0 {
		t.Fatalf("first anchor %+v", info.index[0])
	}
	for i := 1; i < len(info.index); i++ {
		a, b := info.index[i-1], info.index[i]
		if b.pos-a.pos < indexStride || b.seq <= a.seq || b.at <= a.at {
			t.Fatalf("anchors %+v then %+v", a, b)
		}
		if _, _, err := decodeFrame(data[b.pos:]); err != nil {
			t.Fatalf("anchor %d does not sit on a frame: %v", i, err)
		}
	}
}

func TestVerifySegmentReadsAcrossChunkBoundariesAndFramesLargerThanAChunk(t *testing.T) {
	big := make([]byte, windowChunk+windowChunk/2)
	for i := range big {
		big[i] = byte(i * 7)
	}
	data := image(1, 40, 0, func(i int) []byte {
		if i%13 == 0 {
			return big
		}
		return bytes.Repeat([]byte{byte(i)}, 100_000)
	})
	info, err := verify(data)
	if err != nil || info.frames != 40 || info.end != int64(len(data)) {
		t.Fatalf("%+v %v", info, err)
	}
	// Recovery from a durable offset inside a chunk agrees.
	ends := frameEnds(t, data)
	info, cut, err := recover_(data, ends[20])
	if err != nil || cut || info.frames != 40 {
		t.Fatalf("%+v %v %v", info, cut, err)
	}
}

func TestASealedSegmentWithDamageAnywhereIsCorruption(t *testing.T) {
	data := image(1, 20, 0, smallPayload)
	ends := frameEnds(t, data)
	if _, err := verify(nil); err == nil || !errors.Is(err, ErrCorrupt) {
		t.Fatalf("empty sealed segment: %v", err)
	}
	// A torn tail at every position is corruption in a sealed segment, and the error names
	// the end of the last whole frame.
	for i := 1; i < len(data); i++ {
		_, err := verify(data[:i])
		want, boundary := int64(0), false
		for _, e := range ends {
			if e <= int64(i) {
				want = e
			}
			boundary = boundary || e == int64(i)
		}
		if boundary {
			if err != nil {
				t.Fatalf("cut at a frame boundary %d is a whole segment: %v", i, err)
			}
			continue
		}
		var se *SegmentError
		if !errors.As(err, &se) || !errors.Is(err, ErrCorrupt) {
			t.Fatalf("torn at %d: %v", i, err)
		}
		if se.Intact != want || se.Offset != want {
			t.Fatalf("torn at %d: intact %d offset %d, want %d", i, se.Intact, se.Offset, want)
		}
	}
	// A flipped byte at every position.
	for i := range data {
		bad := bytes.Clone(data)
		bad[i] ^= 0x01
		if _, err := verify(bad); err == nil {
			t.Fatalf("flip at %d accepted", i)
		}
	}
	// A frame at or below its predecessor, or past it.
	for _, tc := range []struct {
		name string
		seq  uint64
	}{{"duplicate", 20}, {"lower", 3}, {"skip", 22}} {
		rec := Record{Seq: tc.seq, At: 1, Payload: []byte("x")}
		bad := appendFrame(bytes.Clone(data), &rec)
		_, err := verify(bad)
		var se *SegmentError
		if !errors.As(err, &se) || se.Offset != int64(len(data)) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
	// A first frame with sequence 0.
	rec := Record{Seq: 0, At: 1, Payload: []byte("x")}
	if _, err := verify(appendFrame(nil, &rec)); err == nil {
		t.Fatal("sequence 0 accepted")
	}
}

func TestRecoveryParsesForwardFromTheMarkerAndTruncatesAtTheFirstBadFrame(t *testing.T) {
	data := image(1, 30, 0, smallPayload)
	ends := frameEnds(t, data)
	for di, durable := range append([]int64{0}, ends...) {
		// A torn tail at every position past the marker cuts at the last frame boundary.
		for i := int(durable); i <= len(data); i++ {
			info, cut, err := recover_(data[:i], durable)
			if err != nil {
				t.Fatalf("durable %d torn at %d: %v", durable, i, err)
			}
			want := int64(0)
			frames := 0
			for _, e := range ends {
				if e <= int64(i) {
					want = e
					frames++
				}
			}
			if info.end != want || cut != (want != int64(i)) || info.frames != frames {
				t.Fatalf("durable %d torn at %d: end %d cut %v frames %d, want %d %v %d", durable, i, info.end, cut, info.frames, want, want != int64(i), frames)
			}
			if frames > 0 && (info.first.Seq != 1 || info.last.Seq != uint64(frames)) {
				t.Fatalf("durable %d torn at %d: first %d last %d", durable, i, info.first.Seq, info.last.Seq)
			}
		}
		// Damage below the marker is corruption; damage above it is a cut.
		for i := range data {
			bad := bytes.Clone(data)
			bad[i] ^= 0x01
			info, cut, err := recover_(bad, durable)
			if int64(i) < durable {
				if !errors.Is(err, ErrCorrupt) {
					t.Fatalf("durable %d flip at %d: %v, want corrupt", durable, i, err)
				}
				continue
			}
			if err != nil || !cut {
				t.Fatalf("durable %d flip at %d: cut %v err %v", durable, i, cut, err)
			}
			// The cut is at the boundary of the frame holding byte i, or earlier if the flip made
			// an earlier frame's length wrong; never later.
			if info.end > int64(i) {
				t.Fatalf("durable %d flip at %d: end %d is past the damage", durable, i, info.end)
			}
		}
		_ = di
	}
}

func TestRecoveryRefusesAMarkerOffABoundaryOrPastTheFile(t *testing.T) {
	data := image(1, 5, 0, smallPayload)
	ends := frameEnds(t, data)
	if _, _, err := recover_(data, int64(len(data))+1); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("marker past the file: %v", err)
	}
	for i := int64(1); i < int64(len(data)); i++ {
		onBoundary := false
		for _, e := range ends {
			onBoundary = onBoundary || e == i
		}
		_, _, err := recover_(data, i)
		if onBoundary && err != nil {
			t.Fatalf("marker at boundary %d: %v", i, err)
		}
		if !onBoundary && !errors.Is(err, ErrCorrupt) {
			t.Fatalf("marker inside a frame at %d: %v", i, err)
		}
	}
}

func TestNothingAfterTheFirstBadFrameIsTrustedWhateverItContains(t *testing.T) {
	// T2: a payload that embeds a whole valid segment, cut inside the embedded bytes: the
	// embedded frames are contiguous with the real ones, and must never be reached.
	inner := image(4, 10, 0, smallPayload)
	data := image(1, 3, 0, smallPayload)
	embed := Record{Seq: 4, At: 0, Payload: inner}
	ends := frameEnds(t, data)
	data = appendFrame(data, &embed)
	cutAt := len(data) - len(inner)/2
	info, cut, err := recover_(data[:cutAt], 0)
	if err != nil || !cut || info.end != ends[2] || info.frames != 3 || info.last.Seq != 3 {
		t.Fatalf("embedded segment: %+v %v %v", info, cut, err)
	}
	// A zeroed page (a lost write) followed by intact frames: truncated at the hole.
	head := image(1, 3, 0, smallPayload)
	hole := make([]byte, 4096)
	tailFrames := image(4, 3, 0, smallPayload)
	data = append(append(bytes.Clone(head), hole...), tailFrames...)
	info, cut, err = recover_(data, 0)
	if err != nil || !cut || info.end != int64(len(head)) || info.frames != 3 {
		t.Fatalf("zeroed page: %+v %v %v", info, cut, err)
	}
	// An intact frame with the wrong sequence, then intact contiguous frames: cut at it.
	wrong := Record{Seq: 9, At: 0, Payload: []byte("x")}
	data = appendFrame(bytes.Clone(head), &wrong)
	data = append(data, image(10, 3, 0, smallPayload)...)
	info, cut, err = recover_(data, 0)
	if err != nil || !cut || info.end != int64(len(head)) {
		t.Fatalf("wrong sequence: %+v %v %v", info, cut, err)
	}
	// A stale marker below acknowledged records keeps all of them.
	data = image(1, 50, 0, smallPayload)
	info, cut, err = recover_(data, 0)
	if err != nil || cut || info.frames != 50 || info.last.Seq != 50 {
		t.Fatalf("stale marker: %+v %v %v", info, cut, err)
	}
}

func TestACraftedTailCostsOnePassAndStopsAtTheFirstBadFrame(t *testing.T) {
	// A tail of plausible length prefixes: every byte decodes as a short frame length with
	// no CRC behind it. Parse-forward never backtracks, so it stops at the first.
	head := image(1, 3, 0, smallPayload)
	data := append(bytes.Clone(head), bytes.Repeat([]byte{0x01, 0x00}, 1<<20)...)
	counting := &countingReader{r: bytes.NewReader(data)}
	info, cut, err := recoverSegment(counting, int64(len(data)), 0, "")
	if err != nil || !cut || info.end != int64(len(head)) {
		t.Fatalf("%+v %v %v", info, cut, err)
	}
	if counting.bytes > int64(len(data))+2*windowChunk {
		t.Fatalf("read %d bytes for a %d byte file", counting.bytes, len(data))
	}
}

type countingReader struct {
	r     io.ReaderAt
	bytes int64
}

func (c *countingReader) ReadAt(p []byte, off int64) (int, error) {
	n, err := c.r.ReadAt(p, off)
	c.bytes += int64(n)
	return n, err
}

func TestAReaderShorterThanItsStatedSizeIsAnError(t *testing.T) {
	data := image(1, 3, 0, smallPayload)
	_, err := verifySegment(bytes.NewReader(data), int64(len(data))+10, "")
	if !errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, ErrCorrupt) {
		t.Fatalf("a short read is an I/O error, not corruption: %v", err)
	}
	_, _, err = recoverSegment(bytes.NewReader(data), int64(len(data))+10, 0, "")
	if err == nil {
		t.Fatal("short reader accepted")
	}
}

func TestOpenSegmentRefusesSymlinksFIFOsAndOversizeFiles(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "0000000000.log")
	if err := os.WriteFile(good, image(1, 2, 0, smallPayload), 0o600); err != nil {
		t.Fatal(err)
	}
	f, size, err := openSegment(fsx.OS{}, good)
	if err != nil || size == 0 {
		t.Fatal(err)
	}
	_ = f.Close()

	link := filepath.Join(dir, "0000000001.log")
	if err := os.Symlink(good, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openSegment(fsx.OS{}, link); err == nil {
		t.Fatal("opened a symlink")
	}
	fifo := filepath.Join(dir, "0000000002.log")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openSegment(fsx.OS{}, fifo); err == nil {
		t.Fatal("opened a FIFO")
	}
	huge := filepath.Join(dir, "0000000003.log")
	hf, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	if err := hf.Truncate(maxSegmentBytes + 1); err != nil {
		t.Skipf("cannot create a sparse %d byte file: %v", maxSegmentBytes+1, err)
	}
	_ = hf.Close()
	_, _, err = openSegment(fsx.OS{}, huge)
	var se *SegmentError
	if !errors.As(err, &se) {
		t.Fatalf("oversize: %v", err)
	}
	if _, _, err := openSegment(fsx.OS{}, filepath.Join(dir, "missing.log")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing: %v", err)
	}
}

func TestSegmentErrorsFromVerifyAndRecoverNameTheSegmentPassedIn(t *testing.T) {
	const path = "0000000007.log"
	data := image(1, 3, 0, smallPayload)
	torn := data[:len(data)-1]
	flipped := bytes.Clone(data)
	flipped[len(flipped)-1] ^= 0x01
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"verify torn", func() error { _, err := verifySegment(bytes.NewReader(torn), int64(len(torn)), path); return err }},
		{"verify no frames", func() error { _, err := verifySegment(bytes.NewReader(nil), 0, path); return err }},
		{"recover below durable", func() error {
			_, _, err := recoverSegment(bytes.NewReader(flipped), int64(len(flipped)), int64(len(flipped)), path)
			return err
		}},
		{"recover durable past size", func() error {
			_, _, err := recoverSegment(bytes.NewReader(data), int64(len(data)), int64(len(data))+1, path)
			return err
		}},
	} {
		var se *SegmentError
		err := tc.call()
		if !errors.As(err, &se) {
			t.Fatalf("%s: want a *SegmentError, got %v", tc.name, err)
		}
		if se.Path != path || !strings.Contains(se.Error(), path) {
			t.Fatalf("%s: path %q, message %q; want the path %q named", tc.name, se.Path, se.Error(), path)
		}
	}
}
