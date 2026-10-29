package quill

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"math/rand/v2"
	"testing"
)

func recEqual(a, b Record) bool {
	return a.Seq == b.Seq && a.At == b.At && bytes.Equal(a.Payload, b.Payload)
}

// image encodes records with sequences first..first+n-1, At from at, payloads from fn.
func image(first uint64, n int, at int64, fn func(i int) []byte) []byte {
	var buf []byte
	for i := 0; i < n; i++ {
		rec := Record{Seq: first + uint64(i), At: at + int64(i), Payload: fn(i)}
		buf = appendFrame(buf, &rec)
	}
	return buf
}

func smallPayload(i int) []byte { return []byte{byte(i), byte(i >> 8), 'p'} }

func TestFrameRoundTripsEveryShape(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	cases := []Record{
		{Seq: 1},
		{Seq: 1, At: 1_700_000_000, Payload: []byte("hello")},
		{Seq: math.MaxUint64, At: math.MaxInt64, Payload: []byte{0}},
		{Seq: 1 << 63, At: math.MinInt64, Payload: bytes.Repeat([]byte{0xff}, 1000)},
		{Seq: 7, At: -1, Payload: []byte{}},
	}
	for i := 0; i < 200; i++ {
		p := make([]byte, rng.IntN(300))
		for j := range p {
			p[j] = byte(rng.Uint32())
		}
		cases = append(cases, Record{Seq: rng.Uint64(), At: int64(rng.Uint64()), Payload: p})
	}
	for _, rec := range cases {
		frame := appendFrame(nil, &rec)
		if len(frame) != frameLen(&rec) {
			t.Fatalf("frameLen %d, encoded %d", frameLen(&rec), len(frame))
		}
		got, n, err := decodeFrame(frame)
		if err != nil || n != len(frame) || !recEqual(got, rec) {
			t.Fatalf("decode %+v: %+v %d %v", rec, got, n, err)
		}
		// With bytes after it, the frame length is still exact.
		got, n, err = decodeFrame(append(frame, 0xff, 0xff))
		if err != nil || n != len(frame) || !recEqual(got, rec) {
			t.Fatalf("decode with trailing bytes: %v", err)
		}
		if got.Payload != nil && cap(got.Payload) != len(got.Payload) {
			t.Fatal("payload capacity reaches past the payload")
		}
	}
}

func TestUvarintLenMatchesTheEncoder(t *testing.T) {
	for _, v := range []uint64{0, 1, 127, 128, 16383, 16384, 1 << 35, math.MaxUint64 - 1, math.MaxUint64} {
		if got, want := uvarintLen(v), len(binary.AppendUvarint(nil, v)); got != want {
			t.Fatalf("uvarintLen(%d) = %d, want %d", v, got, want)
		}
	}
}

func TestDecodeFrameRefusesEveryDamage(t *testing.T) {
	rec := Record{Seq: 5, At: 9, Payload: []byte("payload")}
	frame := appendFrame(nil, &rec)
	// Torn at every length short of the whole frame.
	for i := 0; i < len(frame); i++ {
		if _, _, err := decodeFrame(frame[:i]); !errors.Is(err, errTornFrame) {
			t.Fatalf("prefix of %d bytes: %v, want torn", i, err)
		}
	}
	// One bit flipped at every position.
	for i := range frame {
		bad := bytes.Clone(frame)
		bad[i] ^= 0x40
		_, _, err := decodeFrame(bad)
		if err == nil {
			t.Fatalf("flip at byte %d accepted", i)
		}
	}
	// A declared length over the limit is bad, never an allocation.
	over := binary.AppendUvarint(nil, maxFrameBody+1)
	if _, _, err := decodeFrame(append(over, make([]byte, 20)...)); !errors.Is(err, errBadFrame) {
		t.Fatalf("oversize length: %v", err)
	}
	if _, _, err := decodeFrame(binary.AppendUvarint(nil, maxFrameBody)); !errors.Is(err, errTornFrame) {
		t.Fatalf("length at the limit with no body: %v, want torn", err)
	}
	// A varint that never ends is bad once it is ten bytes long, torn before that.
	if _, _, err := decodeFrame(bytes.Repeat([]byte{0x80}, 10)); !errors.Is(err, errBadFrame) {
		t.Fatalf("ten continuation bytes: %v", err)
	}
	if _, _, err := decodeFrame(bytes.Repeat([]byte{0x80}, 9)); !errors.Is(err, errTornFrame) {
		t.Fatalf("nine continuation bytes: %v", err)
	}
	if _, _, err := decodeFrame(append(bytes.Repeat([]byte{0xff}, 9), 0x7f)); !errors.Is(err, errBadFrame) {
		t.Fatalf("overflowing varint: %v", err)
	}
	// Slack after the payload, with a correct CRC, is bad: a writer never leaves any.
	body := binary.AppendUvarint(nil, 5)
	body = binary.AppendUvarint(body, 9)
	body = binary.AppendUvarint(body, 2)
	body = append(body, 'a', 'b', 'c')
	slack := binary.AppendUvarint(nil, uint64(len(body)))
	slack = append(slack, body...)
	slack = binary.BigEndian.AppendUint32(slack, crcChecksum(body))
	if _, _, err := decodeFrame(slack); !errors.Is(err, errBadFrame) {
		t.Fatalf("slack: %v", err)
	}
	if _, _, err := decodeFrame(nil); !errors.Is(err, errTornFrame) {
		t.Fatalf("empty: %v", err)
	}
}
