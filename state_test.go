package quill

import (
	"errors"
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
		"missing field":   `{"seq":42,"seg":3,"off":1024}`,
		"extra field":     `{"seq":42,"seg":3,"off":1024,"maxseg":7,"x":1}`,
		"seg over maxseg": `{"seq":42,"seg":8,"off":1024,"maxseg":7}`,
		"negative":        `{"seq":-1,"seg":3,"off":1024,"maxseg":7}`,
		"float":           `{"seq":4.2,"seg":3,"off":1024,"maxseg":7}`,
		"string":          `{"seq":"42","seg":3,"off":1024,"maxseg":7}`,
		"null":            `{"seq":null,"seg":3,"off":1024,"maxseg":7}`,
		"duplicate key":   `{"seq":1,"seq":42,"seg":3,"off":1024,"maxseg":7}`,
		"trailing":        `{"seq":42,"seg":3,"off":1024,"maxseg":7}{"seq":1,"seg":3,"off":1024,"maxseg":7}`,
		"truncated":       `{"seq":42,"seg":3,"off":10`,
		"empty":           ``,
		"zeros":           "\x00\x00\x00\x00",
		"invalid utf-8":   "{\"seq\":42,\"seg\":3,\"off\":1024,\"maxseg\":7,\"\xff\":1}",
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
