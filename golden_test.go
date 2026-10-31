package quill

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
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
		"quill.policy":     encodePolicy(goldenPolicy()),
	}
}

// goldenPolicy gives every field a value no other field has, and none the writer cannot
// produce (a protected consumer cannot start with an underscore), so a decoder that swaps
// the meaning of two fields, as well as one that drops a field, shows. testdata/golden/
// quill.policy must equal what the writer writes for the same settings
// (TestOpenWritesEveryConfigFieldToItsOwnPolicyKey pins that layout, distinctPolicyJSON).
func goldenPolicy() policy {
	return policy{Retention: "36h0m0s", SegmentBytes: 1 << 20, SegmentMaxAge: "5h0m0s",
		MaxBytes: 64 << 20, MinFreeBytes: 3 << 20, MaxRecords: 12345, MaxPayloadBytes: 1 << 16,
		ConsumerTTL: "7h0m0s", ProtectedConsumers: []string{"billing", "audit"}}
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

// failGoldenDecode is the failure of the decode direction: a file already on disk, written
// by an earlier version, is now read differently. The remedy is the same as for the encoder.
func failGoldenDecode(t *testing.T, artifact, name, detail string) {
	t.Helper()
	t.Fatalf("%s DECODING CHANGED (%s): %s\n"+
		"A log written before this change is now read differently. If the change is intended it needs a new "+
		"quill.format version, not a new meaning for this one. Do not edit the expected values or re-record "+
		"this golden to make the test pass.", artifact, name, detail)
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
			failGoldenDecode(t, "frame", g.name, fmt.Sprintf("%+v %d %v, want %+v", rec, n, err, g.rec))
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
	if want := bytes.Join(frames, nil); !bytes.Equal(image, want) {
		// Both sides are recorded artifacts (the checked-in file and the pasted frames), so
		// neither is "now": the two records disagree, and the encoder is not involved.
		t.Fatalf("segment image DISAGREES with recordedFrames (segment.log).\n  segment.log    %s\n  recordedFrames %s\n"+
			"The checked-in file and the pasted frames do not agree: one was re-recorded or edited without the other. "+
			"Do not re-record either to make the test pass: a changed format needs a new quill.format version.",
			hex.EncodeToString(image), hex.EncodeToString(want))
	}
	info, cut, err := recoverSegment(bytes.NewReader(image), int64(len(image)), 0, "")
	if err != nil || !cut || info.frames != 1 || info.end != int64(len(frames[0])) {
		failGoldenDecode(t, "segment image", "segment.log", fmt.Sprintf("recovery: %+v %v %v", info, cut, err))
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
	// The recorded policy is a file the writer can produce: it is the one a writer opened with
	// the same settings writes.
	if got, _ := os.ReadFile(filepath.Join(goldenDir, "quill.policy")); string(got) != distinctPolicyJSON {
		t.Fatalf("testdata/golden/quill.policy is not what the writer writes for its settings:\n%s\nwant\n%s", got, distinctPolicyJSON)
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
		failGoldenDecode(t, "state file", "quill.tail", fmt.Sprintf("%+v %v", tl, err))
	}
	if id, err := decodeFormat(read("quill.format")); err != nil || id != testID {
		failGoldenDecode(t, "state file", "quill.format", fmt.Sprintf("%q %v", id, err))
	}
	if c, err := decodeConsumer(read("consumer")); err != nil || c != (consumerState{testID, 1234568, true}) {
		failGoldenDecode(t, "state file", "consumer", fmt.Sprintf("%+v %v", c, err))
	}
	if c, err := decodeConsumer(read("consumer-default")); err != nil || c != (consumerState{testID, 1, false}) {
		failGoldenDecode(t, "state file", "consumer-default", fmt.Sprintf("%+v %v", c, err))
	}
	p, ok := decodePolicy(read("quill.policy"))
	if want := goldenPolicy(); !ok || !reflect.DeepEqual(p, want) {
		failGoldenDecode(t, "state file", "quill.policy", fmt.Sprintf("%+v %v, want %+v", p, ok, want))
	}
}
