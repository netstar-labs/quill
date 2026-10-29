package quill

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// A golden test pins encoded bytes that outlive the process that wrote them to bytes recorded
// once. Round trips cannot do this: an encoder and decoder changed together still agree while
// disagreeing with every file written before. The recorded bytes come from running the
// encoder once (QUILL_RECORD_GOLDENS=1 go test -run Golden) and are checked in; they are
// never computed from the code under test at test time, which would be a tautology.
//
// Each golden checks both directions: the encoder still produces the recorded bytes, and the
// decoder still reads them back. A failure means the format changed. Do not re-record a golden
// to make the test pass: a format change needs a new quill.format version (docs/format.md).

const goldenDir = "testdata/golden"

func frameGoldens() []struct {
	name string
	rec  Record
} {
	return []struct {
		name string
		rec  Record
	}{
		{"typical", Record{Seq: 1, At: 1_700_000_000, Payload: []byte("hello")}},
		{"large-seq-empty-payload", Record{Seq: 1 << 40}},
		// The largest sequence with a negative At: At is written as a ten-byte unsigned
		// varint, and one that differs from the sequence, so a parser that swaps them shows.
		{"max-seq-negative-at", Record{Seq: math.MaxUint64, At: -2, Payload: []byte{0xff, 0xff, 0xff}}},
	}
}

// recordedFrames is the output of recording, pasted here: hex of each golden frame.
var recordedFrames = map[string]string{
	"typical":                 "0c0180e2cfaa060568656c6c6f3b5aa01c",
	"large-seq-empty-payload": "088080808080200000c44de4d0",
	"max-seq-negative-at":     "18ffffffffffffffffff01feffffffffffffffff0103ffffffdfce82f5",
}

// stateGoldens are the state files whose bytes are pinned, as (file, encoded) pairs.
func stateGoldens() map[string][]byte {
	return map[string][]byte{
		"quill.tail":       encodeTail(tail{Seq: 1234567, Seg: 42, Off: 8_000_001, MaxSeg: 45}),
		"quill.format":     encodeFormat(testID),
		"consumer":         encodeConsumer(consumerState{ID: testID, Next: 1234568, Protect: true}),
		"consumer-default": encodeConsumer(consumerState{ID: testID, Next: 1}),
		"quill.policy": encodePolicy(policy{Retention: "24h0m0s", SegmentBytes: 8 << 20, SegmentMaxAge: "24h0m0s",
			MaxBytes: 0, MinFreeBytes: 0, MaxRecords: 0, MaxPayloadBytes: maxPayloadBytes, ConsumerTTL: "0s",
			ProtectedConsumers: []string{"_archive", "billing"}}),
	}
}

func goldenImage() []byte {
	var image []byte
	for _, g := range frameGoldens() {
		image = appendFrame(image, &g.rec)
	}
	return image
}

func TestRecordGoldens(t *testing.T) {
	if os.Getenv("QUILL_RECORD_GOLDENS") == "" {
		t.Skip("set QUILL_RECORD_GOLDENS=1 to record; then paste the frame lines into recordedFrames")
	}
	for _, g := range frameGoldens() {
		t.Logf("\t%q: %q,", g.name, hex.EncodeToString(appendFrame(nil, &g.rec)))
	}
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := stateGoldens()
	files["segment.log"] = goldenImage()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(goldenDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func failGolden(t *testing.T, artifact, name, got, want string) {
	t.Helper()
	t.Fatalf("%s ENCODING CHANGED (%s).\n  now      %s\n  recorded %s\n"+
		"Every log written before this change is now read differently. If the change is intended it needs a new "+
		"quill.format version, not a new shape for this one. Do not re-record this golden to make the test pass.",
		artifact, name, got, want)
}

func TestFrameGolden(t *testing.T) {
	var frames [][]byte
	for _, g := range frameGoldens() {
		want, ok := recordedFrames[g.name]
		if !ok {
			t.Fatalf("no recorded bytes for %q", g.name)
		}
		if got := hex.EncodeToString(appendFrame(nil, &g.rec)); got != want {
			failGolden(t, "frame", g.name, got, want)
		}
		frame, err := hex.DecodeString(want)
		if err != nil {
			t.Fatal(err)
		}
		rec, n, err := decodeFrame(frame)
		if err != nil || n != len(frame) || !recEqual(rec, g.rec) {
			t.Fatalf("frame DECODING CHANGED (%s): %+v %d %v, want %+v", g.name, rec, n, err, g.rec)
		}
		frames = append(frames, frame)
	}
	// The recorded segment image: the frames in order, verified whole, with the recorded
	// sequences at the recorded positions. (They are not contiguous, so recovery parses only
	// the first; verification as a sealed segment likewise stops. Pin that too.)
	image, err := os.ReadFile(filepath.Join(goldenDir, "segment.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(image, bytes.Join(frames, nil)) {
		failGolden(t, "segment image", "segment.log", hex.EncodeToString(image), hex.EncodeToString(bytes.Join(frames, nil)))
	}
	info, cut, err := recoverSegment(bytes.NewReader(image), int64(len(image)), 0, "")
	if err != nil || !cut || info.frames != 1 || info.end != int64(len(frames[0])) {
		t.Fatalf("recovery of the golden image: %+v %v %v", info, cut, err)
	}
}

func TestStateFileGoldens(t *testing.T) {
	for name, want := range stateGoldens() {
		got, err := os.ReadFile(filepath.Join(goldenDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			failGolden(t, "state file", name, fmt.Sprintf("%q", want), fmt.Sprintf("%q", got))
		}
	}
	// Decoding the recorded files gives the recorded values.
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(goldenDir, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if tl, err := decodeTail(read("quill.tail")); err != nil || tl != (tail{Seq: 1234567, Seg: 42, Off: 8_000_001, MaxSeg: 45}) {
		t.Fatalf("quill.tail DECODING CHANGED: %+v %v", tl, err)
	}
	if id, err := decodeFormat(read("quill.format")); err != nil || id != testID {
		t.Fatalf("quill.format DECODING CHANGED: %q %v", id, err)
	}
	if c, err := decodeConsumer(read("consumer")); err != nil || c != (consumerState{testID, 1234568, true}) {
		t.Fatalf("consumer DECODING CHANGED: %+v %v", c, err)
	}
	if c, err := decodeConsumer(read("consumer-default")); err != nil || c != (consumerState{testID, 1, false}) {
		t.Fatalf("consumer DECODING CHANGED: %+v %v", c, err)
	}
	p, ok := decodePolicy(read("quill.policy"))
	if !ok || p.Retention != "24h0m0s" || p.SegmentBytes != 8<<20 || p.MaxPayloadBytes != maxPayloadBytes || len(p.ProtectedConsumers) != 2 {
		t.Fatalf("quill.policy DECODING CHANGED: %+v %v", p, ok)
	}
}
