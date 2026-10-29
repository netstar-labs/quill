package quill

import (
	"bytes"
	"testing"
)

// T12: the frame decoder, segment recovery and every state-file parser never panic, and what
// they accept round-trips.

func FuzzDecodeFrame(f *testing.F) {
	rec := Record{Seq: 1, At: 5, Payload: []byte("hello")}
	f.Add(appendFrame(nil, &rec))
	f.Add([]byte{0xff, 0x00, 0x01})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		rec, n, err := decodeFrame(data)
		if err != nil {
			if n != 0 {
				t.Fatalf("error with n=%d", n)
			}
			return
		}
		if n <= 0 || n > len(data) {
			t.Fatalf("n=%d for %d bytes", n, len(data))
		}
		again, m, err := decodeFrame(appendFrame(nil, &rec))
		if err != nil || !recEqual(again, rec) || m > n {
			t.Fatalf("re-encode: %+v %d %v (was %+v %d)", again, m, err, rec, n)
		}
	})
}

func FuzzRecoverSegment(f *testing.F) {
	f.Add(image(1, 5, 0, smallPayload), int64(0))
	f.Add(image(3, 2, 0, smallPayload), int64(10))
	f.Add([]byte{0x01, 0x00, 0x00, 0x00, 0x00}, int64(0))
	f.Fuzz(func(t *testing.T, data []byte, durable int64) {
		size := int64(len(data))
		info, cut, err := recoverSegment(bytes.NewReader(data), size, durable, "")
		if err != nil {
			return
		}
		if info.end > size || info.end < durable || cut != (info.end < size) {
			t.Fatalf("end %d size %d durable %d cut %v", info.end, size, durable, cut)
		}
		// The accepted prefix is a contiguous run of frames that a sealed verification
		// also accepts when it is non-empty.
		if info.frames > 0 {
			v, err := verifySegment(bytes.NewReader(data[:info.end]), info.end, "")
			if err != nil || v.frames != info.frames || v.first.Seq != info.first.Seq || v.last.Seq != info.last.Seq || v.first.At != info.first.At || v.last.At != info.last.At {
				t.Fatalf("verify of the accepted prefix: %+v %v, recovery said %+v", v, err, info)
			}
			if info.last.Seq-info.first.Seq+1 != uint64(info.frames) {
				t.Fatalf("not contiguous: %+v", info)
			}
		} else if info.end != durable {
			t.Fatalf("no frames but end %d, durable %d", info.end, durable)
		}
	})
}

func FuzzDecodeTail(f *testing.F) {
	f.Add(encodeTail(tail{Seq: 1, Seg: 2, Off: 3, MaxSeg: 4}))
	f.Add([]byte(`{"seq":1}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		tl, err := decodeTail(data)
		if err != nil {
			return
		}
		again, err := decodeTail(encodeTail(tl))
		if err != nil || again != tl {
			t.Fatalf("round trip: %+v %v", again, err)
		}
	})
}

func FuzzDecodeFormat(f *testing.F) {
	f.Add(encodeFormat(testID))
	f.Add([]byte("quill/2 x\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		id, err := decodeFormat(data)
		if err != nil {
			return
		}
		if again, err := decodeFormat(encodeFormat(id)); err != nil || again != id {
			t.Fatalf("round trip: %q %v", again, err)
		}
	})
}

func FuzzDecodeConsumer(f *testing.F) {
	f.Add(encodeConsumer(consumerState{testID, 7, true}))
	f.Add([]byte("x 1 0\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		c, err := decodeConsumer(data)
		if err != nil {
			return
		}
		if again, err := decodeConsumer(encodeConsumer(c)); err != nil || again != c {
			t.Fatalf("round trip: %+v %v", again, err)
		}
	})
}

func FuzzDecodePolicy(f *testing.F) {
	f.Add(encodePolicy(policy{Retention: "1h"}))
	f.Add([]byte("{"))
	f.Fuzz(func(t *testing.T, data []byte) {
		p, ok := decodePolicy(data) // must never panic; a failure is ok == false, never fatal
		if !ok {
			return
		}
		if _, ok := decodePolicy(encodePolicy(p)); !ok {
			t.Fatalf("re-encoded policy refused: %+v", p)
		}
	})
}
