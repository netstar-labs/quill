package quill

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

const testID = logID("0123456789abcdef0123456789abcdef")

func TestTailRoundTripsAndIsStrict(t *testing.T) {
	in := tail{Seq: 42, Seg: 3, Off: 1024, MaxSeg: 7}
	enc := encodeTail(in)
	if string(enc) != `{"seq":42,"seg":3,"off":1024,"maxseg":7}`+"\n" {
		t.Fatalf("encoding %q", enc)
	}
	out, err := decodeTail(enc)
	if err != nil || out != in {
		t.Fatalf("%+v %v", out, err)
	}
	bad := map[string]string{
		"missing field":              `{"seq":42,"seg":3,"off":1024}`,
		"extra field":                `{"seq":42,"seg":3,"off":1024,"maxseg":7,"x":1}`,
		"seg over maxseg":            `{"seq":42,"seg":8,"off":1024,"maxseg":7}`,
		"negative":                   `{"seq":-1,"seg":3,"off":1024,"maxseg":7}`,
		"float":                      `{"seq":4.2,"seg":3,"off":1024,"maxseg":7}`,
		"string":                     `{"seq":"42","seg":3,"off":1024,"maxseg":7}`,
		"null":                       `{"seq":null,"seg":3,"off":1024,"maxseg":7}`,
		"duplicate key":              `{"seq":1,"seq":42,"seg":3,"off":1024,"maxseg":7}`,
		"case-variant duplicate key": `{"seq":1,"SEQ":42,"seg":3,"off":1024,"maxseg":7}`,
		"trailing":                   `{"seq":42,"seg":3,"off":1024,"maxseg":7}{"seq":1,"seg":3,"off":1024,"maxseg":7}`,
		"truncated":                  `{"seq":42,"seg":3,"off":10`,
		"empty":                      ``,
		"zeros":                      "\x00\x00\x00\x00",
		"invalid utf-8":              "{\"seq\":42,\"seg\":3,\"off\":1024,\"maxseg\":7,\"\xff\":1}",
	}
	for name, in := range bad {
		if v, err := decodeTail([]byte(in)); err == nil {
			t.Errorf("%s: accepted %q as %+v", name, in, v)
		} else if !errors.Is(err, errBadState) {
			t.Errorf("%s: %v is not errBadState", name, err)
		}
	}
}

func TestFormatMarkerRoundTripsAndRefusesTheFuture(t *testing.T) {
	enc := encodeFormat(testID)
	if string(enc) != "quill/1 0123456789abcdef0123456789abcdef\n" {
		t.Fatalf("encoding %q", enc)
	}
	id, err := decodeFormat(enc)
	if err != nil || id != testID {
		t.Fatalf("%q %v", id, err)
	}
	_, err = decodeFormat([]byte("quill/2 0123456789abcdef0123456789abcdef\n"))
	if !errors.Is(err, ErrFutureVersion) {
		t.Fatalf("future: %v", err)
	}
	_, err = decodeFormat([]byte("quill/18446744073709551615 0123456789abcdef0123456789abcdef\n"))
	if !errors.Is(err, ErrFutureVersion) {
		t.Fatalf("far future: %v", err)
	}
	bad := map[string]string{
		"no newline":      "quill/1 0123456789abcdef0123456789abcdef",
		"two newlines":    "quill/1 0123456789abcdef0123456789abcdef\n\n",
		"crlf":            "quill/1 0123456789abcdef0123456789abcdef\r\n",
		"version 0":       "quill/0 0123456789abcdef0123456789abcdef\n",
		"leading zero":    "quill/01 0123456789abcdef0123456789abcdef\n",
		"plus":            "quill/+1 0123456789abcdef0123456789abcdef\n",
		"huge version":    "quill/99999999999999999999 0123456789abcdef0123456789abcdef\n",
		"upper-case id":   "quill/1 0123456789ABCDEF0123456789ABCDEF\n",
		"short id":        "quill/1 0123456789abcdef0123456789abcde\n",
		"long id":         "quill/1 0123456789abcdef0123456789abcdef0\n",
		"extra field":     "quill/1 0123456789abcdef0123456789abcdef x\n",
		"other name":      "other/1 0123456789abcdef0123456789abcdef\n",
		"no id":           "quill/1\n",
		"two spaces":      "quill/1  0123456789abcdef0123456789abcdef\n",
		"empty":           "",
		"not hex":         "quill/1 0123456789abcdef0123456789abcdeg\n",
		"leading space":   " quill/1 0123456789abcdef0123456789abcdef\n",
		"version newline": "quill/1\n0123456789abcdef0123456789abcdef\n",
	}
	for name, in := range bad {
		if id, err := decodeFormat([]byte(in)); err == nil {
			t.Errorf("%s: accepted %q as %q", name, in, id)
		} else if !errors.Is(err, errBadState) {
			t.Errorf("%s: %v is not errBadState", name, err)
		}
	}
	id, err = newLogID()
	if err != nil || !validLogID(string(id)) {
		t.Fatalf("new id %q %v", id, err)
	}
	if id2, _ := newLogID(); id2 == id {
		t.Fatal("two ids the same")
	}
}

func TestConsumerFileRoundTripsAndIsStrict(t *testing.T) {
	for _, in := range []consumerState{{testID, 0, false}, {testID, 1, true}, {testID, 1<<64 - 1, false}} {
		enc := encodeConsumer(in)
		out, err := decodeConsumer(enc)
		if err != nil || out != in {
			t.Fatalf("%q: %+v %v", enc, out, err)
		}
	}
	if string(encodeConsumer(consumerState{testID, 7, true})) != "0123456789abcdef0123456789abcdef 7 1\n" {
		t.Fatalf("encoding %q", encodeConsumer(consumerState{testID, 7, true}))
	}
	bad := map[string]string{
		"no newline":   "0123456789abcdef0123456789abcdef 7 1",
		"crlf":         "0123456789abcdef0123456789abcdef 7 1\r\n",
		"two fields":   "0123456789abcdef0123456789abcdef 7\n",
		"four fields":  "0123456789abcdef0123456789abcdef 7 1 x\n",
		"protect 2":    "0123456789abcdef0123456789abcdef 7 2\n",
		"protect true": "0123456789abcdef0123456789abcdef 7 true\n",
		"leading zero": "0123456789abcdef0123456789abcdef 07 1\n",
		"plus":         "0123456789abcdef0123456789abcdef +7 1\n",
		"negative":     "0123456789abcdef0123456789abcdef -7 1\n",
		"too large":    "0123456789abcdef0123456789abcdef 18446744073709551616 1\n",
		"underscore":   "0123456789abcdef0123456789abcdef 1_000 1\n",
		"bad id":       "0123456789abcdef0123456789abcdeX 7 1\n",
		"tab":          "0123456789abcdef0123456789abcdef\t7\t1\n",
		"empty":        "",
		"zeros":        "\x00\x00\x00",
		"two lines":    "0123456789abcdef0123456789abcdef 7 1\n0123456789abcdef0123456789abcdef 8 1\n",
	}
	for name, in := range bad {
		if c, err := decodeConsumer([]byte(in)); err == nil {
			t.Errorf("%s: accepted %q as %+v", name, in, c)
		} else if !errors.Is(err, errBadState) {
			t.Errorf("%s: %v is not errBadState", name, err)
		}
	}
}

func TestPolicyIsTolerantAndNeverFatal(t *testing.T) {
	in := policy{Retention: "24h0m0s", SegmentBytes: 8 << 20, SegmentMaxAge: "24h0m0s", MaxBytes: 1 << 30,
		MaxPayloadBytes: maxPayloadBytes, ConsumerTTL: "0s", ProtectedConsumers: []string{"billing"}}
	out, ok := decodePolicy(encodePolicy(in))
	if !ok || out.Retention != in.Retention || out.MaxBytes != in.MaxBytes || len(out.ProtectedConsumers) != 1 {
		t.Fatalf("%+v %v", out, ok)
	}
	out, ok = decodePolicy([]byte(`{"retention":"1h","future_field":{"a":[1,2]},"max_bytes":5}`))
	if !ok || out.Retention != "1h" || out.MaxBytes != 5 {
		t.Fatalf("unknown field: %+v %v", out, ok)
	}
	for name, in := range map[string]string{
		"garbage":      "\xff\xfe",
		"empty":        "",
		"array":        "[]",
		"bad duration": `{"retention":"soon"}`,
		"bad type":     `{"max_bytes":"many"}`,
		"truncated":    `{"retention":"1h"`,
	} {
		if p, ok := decodePolicy([]byte(in)); ok {
			t.Errorf("%s: accepted %q as %+v", name, in, p)
		}
	}
}

// A policy with no retention is not a policy of zeros: a version-skewed writer that renamed
// its fields would leave such a file, and the offline tools would act on a retention of 0,
// which means the 24 h default (decision 68). Every writer writes retention, an off one as -1ns.
func TestPolicyWithoutRetentionIsUnparseable(t *testing.T) {
	for name, in := range map[string]string{
		"empty object":    `{}`,
		"null":            `null`,
		"unknown fields":  `{"unknown":1}`,
		"caps only":       `{"max_bytes":5}`,
		"empty retention": `{"retention":""}`,
	} {
		if p, ok := decodePolicy([]byte(in)); ok {
			t.Errorf("%s: accepted %q as %+v", name, in, p)
		}
	}
	for _, in := range []string{`{"retention":"-1ns"}`, `{"retention":"0s"}`} {
		if _, ok := decodePolicy([]byte(in)); !ok {
			t.Errorf("refused %q", in)
		}
	}
}

func TestParseUintIsPlainDecimalOnly(t *testing.T) {
	for _, s := range []string{"0", "1", "42", "18446744073709551615"} {
		if _, err := parseUint(s); err != nil {
			t.Errorf("%q: %v", s, err)
		}
	}
	for _, s := range []string{"", "00", "01", "+1", "-1", " 1", "1 ", "0x10", "1_0", "18446744073709551616", "１"} {
		if v, err := parseUint(s); err == nil {
			t.Errorf("%q accepted as %d", s, v)
		}
	}
}

func TestParseSegmentNameIsTenDigitsAndLog(t *testing.T) {
	for name, want := range map[string]uint64{"0000000000.log": 0, "0000000001.log": 1, "9999999999.log": 9999999999} {
		if id, ok := parseSegmentName(name); !ok || id != want {
			t.Errorf("%q: got %d, %v; want %d", name, id, ok, want)
		}
	}
	for _, name := range []string{"", ".log", "1.log", "000000001.log", "00000000001.log", "0000000001", "0000000001.logx",
		"+000000001.log", "-000000001.log", "000000_001.log", "0x00000001.log", "000000000１.log", "0000000001.tmp"} {
		if id, ok := parseSegmentName(name); ok {
			t.Errorf("%q accepted as %d", name, id)
		}
	}
}

func TestStateFilesAreWrittenAtomicallyAndReadWithoutFollowingLinks(t *testing.T) {
	s := fsx.NewSim()
	if err := s.Mkdir("/log", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncDir("/"); err != nil {
		t.Fatal(err)
	}
	if err := writeState(s, "/log/"+tailName, encodeTail(tail{Seq: 1, Seg: 0, Off: 10, MaxSeg: 0}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncDir("/log"); err != nil {
		t.Fatal(err)
	}
	if err := writeState(s, "/log/"+tailName, encodeTail(tail{Seq: 2, Seg: 0, Off: 20, MaxSeg: 0}), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := readState(s, "/log/"+tailName)
	if err != nil {
		t.Fatal(err)
	}
	if tl, err := decodeTail(data); err != nil || tl.Seq != 2 {
		t.Fatalf("%+v %v", tl, err)
	}
	names, _ := s.ReadDir("/log")
	if len(names) != 1 {
		t.Fatalf("directory holds %v", names)
	}
	// Written with no directory fsync: the rename may be undone by a crash, never torn.
	s.Crash(nil)
	data, err = readState(s, "/log/"+tailName)
	if err != nil {
		t.Fatal(err)
	}
	if tl, err := decodeTail(data); err != nil || (tl.Seq != 1 && tl.Seq != 2) {
		t.Fatalf("after a crash: %+v %v", tl, err)
	}

	dir := t.TempDir()
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, encodeTail(tail{}), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, tailName)
	if err := os.Symlink(victim, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readState(fsx.OS{}, link); err == nil {
		t.Fatal("read through a symlink")
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	// Opening a FIFO read-only blocks until a writer appears; open it non-blocking from the
	// test so readState's own open does not hang if the regular-file check ever moves after it.
	w, err := os.OpenFile(fifo, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := readState(fsx.OS{}, fifo); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("fifo: %v", err)
	}
	huge := filepath.Join(dir, "huge")
	hf, _ := os.Create(huge)
	_ = hf.Truncate(maxStateBytes + 1)
	_ = hf.Close()
	if _, err := readState(fsx.OS{}, huge); err == nil || !errors.Is(err, errBadState) {
		t.Fatalf("huge: %v", err)
	}
}

// TestAppendTailMatchesJSON pins the hand-written encoder to json.Marshal of the struct
// plus a newline, byte for byte, and checks that it appends to the buffer it is given.
func TestAppendTailMatchesJSON(t *testing.T) {
	max := uint64(math.MaxUint64)
	for _, in := range []tail{{}, {Seq: 1, Seg: 2, Off: 3, MaxSeg: 4}, {Seq: max, Seg: max, Off: max, MaxSeg: max}, {Seq: 42, Seg: 3, Off: 1024, MaxSeg: 7}} {
		want, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, '\n')
		if got := appendTail(nil, in); string(got) != string(want) {
			t.Fatalf("%+v: got %q, want %q", in, got, want)
		}
		if got := appendTail([]byte("xy"), in); string(got) != "xy"+string(want) {
			t.Fatalf("%+v: append to a prefix: %q", in, got)
		}
	}
}

// TestDecodeTailFastAgreesWithSlow: whenever the fast path accepts, the strict path gives
// the same tail; a mutated marker never makes the two disagree on validity or value.
func TestDecodeTailFastAgreesWithSlow(t *testing.T) {
	seeds := []tail{{}, {1, 1, 1, 1}, {1234567, 42, 8_000_001, 45}, {1<<64 - 1, 1<<64 - 1, 1<<64 - 1, 1<<64 - 1}, {9_999_999_999_999_999_999, 3, 4, 5}, {1, 5, 4, 3}}
	rng := rand.New(rand.NewSource(1))
	fast := 0
	check := func(in []byte) {
		f, ok := decodeTailFast(in)
		s, err := decodeTailSlow(in)
		if ok && (err != nil || f != s) {
			t.Fatalf("fast accepted %q as %v, strict gave %v, %v", in, f, s, err)
		}
		if ok {
			fast++
		}
		if got, gerr := decodeTail(in); got != s || (gerr == nil) != (err == nil) || (err != nil && gerr.Error() != err.Error()) {
			t.Fatalf("decodeTail(%q) = %v, %v; strict %v, %v", in, got, gerr, s, err)
		}
	}
	for _, sd := range seeds {
		enc := encodeTail(sd)
		check(enc)
		for i := 0; i < 20000; i++ {
			m := append([]byte(nil), enc...)
			for k := rng.Intn(3) + 1; k > 0 && len(m) > 0; k-- {
				switch rng.Intn(4) {
				case 0:
					m[rng.Intn(len(m))] = "0123456789{}:,\" \n-+e.x"[rng.Intn(22)]
				case 1:
					i := rng.Intn(len(m))
					m = append(m[:i], m[i+1:]...)
				case 2:
					i := rng.Intn(len(m) + 1)
					m = append(m[:i], append([]byte{"0123456789 \n,"[rng.Intn(13)]}, m[i:]...)...)
				case 3:
					m = m[:rng.Intn(len(m)+1)]
				}
			}
			check(m)
		}
	}
	if fast == 0 {
		t.Fatal("the fast path never accepted a mutated input")
	}
}

// TestTailRefusesIDsTheSegmentNamesCannotHold: segment names carry ten decimal digits, so a
// marker whose maxseg (and therefore seg) is above 9999999999 is refused by both decode
// paths; one at the limit is accepted.
func TestTailRefusesIDsTheSegmentNamesCannotHold(t *testing.T) {
	for _, in := range []string{
		`{"seq":3,"seg":0,"off":39,"maxseg":10000000000}` + "\n",
		`{"seq":3,"seg":10000000000,"off":39,"maxseg":10000000000}` + "\n",
		`{"seq":3,"seg":0,"off":39,"maxseg":18446744073709551615}` + "\n",
		`{"seq":3,"seg":0,"off":39,"maxseg":18446744073709551615}`,
	} {
		if v, err := decodeTail([]byte(in)); err == nil {
			t.Errorf("accepted %q as %+v", in, v)
		} else if !errors.Is(err, errBadState) {
			t.Errorf("%q: %v is not errBadState", in, err)
		}
	}
	top := tail{Seq: 3, Seg: maxSegmentID, Off: 39, MaxSeg: maxSegmentID}
	if v, err := decodeTail(encodeTail(top)); err != nil || v != top {
		t.Fatalf("the largest representable id: %+v, %v", v, err)
	}
}

// TestRotationRefusesAnIDTheSegmentNamesCannotHold: at maxseg 9999999999 the next id would
// be an eleven-digit name no scan recognises, and at 2^64-1 it would wrap to 0. The seal
// fails instead and no such file is created.
func TestRotationRefusesAnIDTheSegmentNamesCannotHold(t *testing.T) {
	for _, top := range []uint64{maxSegmentID, 1<<64 - 1} {
		s := fsx.NewSim()
		l := openSim(t, s, nil)
		appendN(t, l, 1)
		mustFlush(t, l)
		l.mu.Lock()
		l.maxSeg = top
		l.mu.Unlock()
		if err := l.Roll(); err == nil {
			t.Errorf("maxseg %d: Roll sealed into an id the names cannot hold", top)
		}
		names, err := s.ReadDir("/log")
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range names {
			if strings.HasSuffix(n, ".log") && n != segmentName(0) {
				t.Errorf("maxseg %d: created %s", top, n)
			}
		}
		_ = l.Close()
	}
}
