package quill

import (
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// padFirstLength rewrites the first frame of image so its length prefix is the non-minimal
// two-byte form of the same value. The CRC covers only the body, so it still verifies.
func padFirstLength(t *testing.T, image []byte) []byte {
	t.Helper()
	if image[0] >= 0x80 {
		t.Fatal("setup: the first length prefix is already two bytes")
	}
	return append([]byte{image[0] | 0x80, 0x00}, image[1:]...)
}

// The format says uvarint is the canonical LEB128 of AppendUvarint. A decoder that accepted
// a padded varint would verify a frame whose real length differs from the length a reader
// computes for its record, so a read that ends a batch after it resumed inside the frame and
// reported a torn frame in a log Check called healthy.
func TestDecodeFrameRefusesANonMinimalVarint(t *testing.T) {
	rec := Record{Seq: 5, At: 9, Payload: []byte("payload")}
	frame := appendFrame(nil, &rec)
	if _, _, err := decodeFrame(padFirstLength(t, frame)); !errors.Is(err, errBadFrame) {
		t.Fatalf("padded length prefix: %v, want bad frame", err)
	}
	// A padded varint inside the body, with the length and CRC made right so only the padding
	// is wrong: seq 5 in three bytes.
	body := append([]byte{0x85, 0x80, 0x00}, frame[2:len(frame)-4]...)
	crafted := binary.AppendUvarint(nil, uint64(len(body)))
	crafted = append(crafted, body...)
	crafted = binary.BigEndian.AppendUint32(crafted, crc32.Checksum(body, crcTable))
	if _, _, err := decodeFrame(crafted); !errors.Is(err, errBadFrame) {
		t.Fatalf("padded sequence: %v, want bad frame", err)
	}
	if _, _, err := decodeFrame(frame); err != nil {
		t.Fatalf("control: the minimal frame: %v", err)
	}
}

func TestANonMinimalLengthPrefixIsCorruptToCheckOpenAndRead(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, func(c *Config) { c.SegmentBytes = 1 << 20 })
	for _, p := range []string{"one", "two", "three"} {
		if _, err := l.Append([]byte(p)); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	path := "/log/" + segmentName(0)
	writeFile(t, s, path, padFirstLength(t, fileBytes(t, s, path)))
	tl := readTail(t, s, "/log")
	tl.Off++
	writeFile(t, s, "/log/"+tailName, encodeTail(tl))

	r, err := OpenReader("/log", ReaderOptions{fs: s, MaxReadBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	res, err := r.Check()
	if err != nil {
		t.Fatal(err)
	}
	if res.Healthy {
		var got []uint64
		var rerr error
		for rec, err := range r.Records(context.Background(), 1) {
			if err != nil {
				rerr = err
				break
			}
			got = append(got, rec.Seq)
		}
		t.Fatalf("Check says healthy, but Records yielded %v then %v", got, rerr)
	}
	if w, err := Open(simConfig(s, "/log")); err == nil {
		w.Close()
		t.Fatal("a writable Open accepted a segment with a non-minimal length prefix")
	} else if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Open: %v, want ErrCorrupt", err)
	}
}
