package quill

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// eofAtEnd is an io.ReaderAt that, as the io.ReaderAt contract allows, returns io.EOF together
// with a read that ends exactly at the end of the data and was filled in full.
type eofAtEnd struct{ b []byte }

func (r eofAtEnd) ReadAt(p []byte, off int64) (int, error) {
	n := copy(p, r.b[off:])
	if off+int64(n) == int64(len(r.b)) {
		return n, io.EOF
	}
	return n, nil
}

// A tail that declares a body of exactly maxFrameBody is an incomplete frame (torn), not an
// invalid one: the limit is inclusive, as in decodeFrame (codec_test.go), and frameAt carries
// its own copy of the comparison.
func TestRecoverTornTailDeclaringExactlyMaxFrameBodyIsTornNotDamaged(t *testing.T) {
	head := image(1, 2, 0, smallPayload)
	for _, tc := range []struct {
		declared uint64
		want     error
	}{
		{maxFrameBody, errTornFrame},
		{maxFrameBody + 1, errBadFrame},
	} {
		data := binary.AppendUvarint(bytes.Clone(head), tc.declared)
		info, cut, err := recover_(data, int64(len(head)))
		if err != nil || !cut || info.end != int64(len(head)) || !errors.Is(info.stop, tc.want) {
			t.Fatalf("declared %d: %+v cut %v err %v, want stop %v", tc.declared, info, cut, err, tc.want)
		}
	}
}

// The window reads at most ten bytes for a length prefix. A tail of exactly ten bytes that all
// continue a varint is a bad frame (no varint is that long and unfinished), while nine such
// bytes are torn: more bytes could still end the varint.
func TestRecoverTenByteVarintRunIsBadAndNineIsTorn(t *testing.T) {
	head := image(1, 2, 0, smallPayload)
	for _, tc := range []struct {
		run  int
		want error
	}{
		{maxVarint - 1, errTornFrame},
		{maxVarint, errBadFrame},
	} {
		data := append(bytes.Clone(head), bytes.Repeat([]byte{0x80}, tc.run)...)
		info, cut, err := recover_(data, int64(len(head)))
		if err != nil || !cut || !errors.Is(info.stop, tc.want) {
			t.Fatalf("%d continuation bytes: %+v cut %v err %v, want stop %v", tc.run, info, cut, err, tc.want)
		}
	}
}

// An io.ReaderAt may return io.EOF together with a read that ended at the end of the data and
// was full. That is a whole read, not a short file.
func TestWindowAcceptsEOFWithAFullFinalRead(t *testing.T) {
	data := image(1, 3, 0, smallPayload)
	info, err := verifySegment(eofAtEnd{data}, int64(len(data)), "")
	if err != nil || info.frames != 3 {
		t.Fatalf("EOF with a full read: %+v %v", info, err)
	}
	// And a file shorter than its stated size is still an unexpected EOF.
	_, err = verifySegment(bytes.NewReader(data), int64(len(data))+1, "")
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("a file shorter than its size: %v", err)
	}
}

// The size limit is inclusive: a segment file of exactly maxSegmentFileBytes is opened, one
// byte more is refused (TestOpenSegmentRefusesSymlinksFIFOsAndOversizeFiles).
func TestOpenSegmentAcceptsAFileOfExactlyTheSizeLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), segmentName(3))
	hf, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := hf.Truncate(maxSegmentFileBytes); err != nil {
		_ = hf.Close()
		t.Skipf("cannot create a sparse %d byte file: %v", int64(maxSegmentFileBytes), err)
	}
	_ = hf.Close()
	f, size, err := openSegment(fsx.OS{}, path)
	if err != nil || size != maxSegmentFileBytes {
		t.Fatalf("size %d err %v", size, err)
	}
	_ = f.Close()
}

// A marker's MaxSeg above the newest segment on disk (the newest was removed, or the marker
// was edited) still bounds the next segment id: an id once used is never reused.
func TestOpenNeverReusesASegmentIDBelowTheMarkersMaxSeg(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Retention = -1 })
	fillSegments(t, l, clk, 2) // segments 0 and 1 sealed, 2 active and empty
	_ = l.Close()
	tl := readTail(t, s, "/log")
	tl.MaxSeg = 7
	writeFile(t, s, "/log/"+tailName, encodeTail(tl))
	l = openRetention(t, s, clk, func(c *Config) { c.Retention = -1 })
	defer l.Close()
	appendN(t, l, 4)
	mustFlush(t, l)
	if ids := segmentFiles(t, s, "/log"); ids[len(ids)-1] != 8 {
		t.Fatalf("segments %v: the next id after MaxSeg 7 is 8", ids)
	}
}
