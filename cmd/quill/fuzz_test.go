package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/netstar-labs/quill"
)

// The fuzz targets of the command line: the places where an operator's or a writer's bytes
// become values (a number, a time, a printed record). hack/fuzz.sh runs each of them in the gate.

// FuzzParseDecimal: a count is a plain base-10 number or it is refused. An accepted string is
// digits only and the value prints back as the string without its leading zeros.
func FuzzParseDecimal(f *testing.F) {
	for _, s := range []string{"0", "1", "010", "0x10", "1e6", "1_0", "+1", "-1", " 1", "", "18446744073709551615", "18446744073709551616", "١"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		n, err := parseDecimal(s)
		if err != nil {
			return
		}
		if s == "" || strings.Trim(s, "0123456789") != "" {
			t.Fatalf("parseDecimal(%q) accepted a string that is not plain digits", s)
		}
		want := strings.TrimLeft(s, "0")
		if want == "" {
			want = "0"
		}
		if got := strconv.FormatUint(n, 10); got != want {
			t.Fatalf("parseDecimal(%q) = %d, which prints as %q", s, n, got)
		}
	})
}

// FuzzParseTime: -since and a consumer's start time never panic on any text, and an accepted
// value means what its form says: "@N" is N unix seconds with N plain digits, a duration is
// never after now, and every other accepted form is an absolute RFC 3339 time. The accepted
// time is then fed to Reader.OffsetForTime, which must answer with a sequence in
// [first, head+1] (the range Commit accepts, decision 59).
func FuzzParseTime(f *testing.F) {
	for _, s := range []string{"2024-01-02T03:04:05Z", "@1700000000", "@9223372036854775807", "@9223372036854775808", "@-1", "@+1", "@",
		"1h30m", "0s", "-1h", "2562047h", "9223372036854775807ns", "0001-01-01T00:00:00Z", "9999-12-31T23:59:59.999999999+23:59", ""} {
		f.Add(s)
	}
	// A log of several segments, so the search has more than one segment to choose from.
	dir := filepath.Join(f.TempDir(), "log")
	l, err := quill.Open(quill.Config{Dir: dir, Retention: -1, SegmentBytes: 256, FlushInterval: time.Hour})
	if err != nil {
		f.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if _, err := l.Append([]byte("record " + strconv.Itoa(i))); err != nil {
			f.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		f.Fatal(err)
	}
	r, err := quill.OpenReader(dir, quill.ReaderOptions{})
	if err != nil {
		f.Fatal(err)
	}
	f.Cleanup(func() { r.Close() })
	st, err := r.Stat()
	if err != nil || st.Records != 40 {
		f.Fatalf("setup: %+v %v", st, err)
	}
	now := time.Unix(1_700_000_000, 0)
	f.Fuzz(func(t *testing.T, s string) {
		tm, err := parseTime(s, now)
		if err != nil {
			return
		}
		if rest, ok := strings.CutPrefix(s, "@"); ok {
			want := strings.TrimLeft(rest, "0")
			if want == "" {
				want = "0"
			}
			if rest == "" || strings.Trim(rest, "0123456789") != "" || strconv.FormatInt(tm.Unix(), 10) != want {
				t.Fatalf("parseTime(%q) = %v: not the unix seconds the text says", s, tm)
			}
		} else if _, err := time.Parse(time.RFC3339, s); err != nil && tm.After(now) {
			t.Fatalf("parseTime(%q) = %v: a duration ago is after now", s, tm)
		}
		off, err := r.OffsetForTime(tm)
		if err != nil {
			t.Fatalf("OffsetForTime(%v) from %q: %v", tm, s, err)
		}
		if off < st.First || off > st.Head+1 {
			t.Fatalf("OffsetForTime(%v) from %q = %d, outside [%d, %d]", tm, s, off, st.First, st.Head+1)
		}
	})
}

// FuzzPrinter: for any record the printer, in every mode, writes exactly one line that holds
// no raw control character, no DEL and no raw C1 control (decision 35), is byte for byte the
// reference formulation of print_test.go, and, in JSON, parses and loses nothing: seq, at,
// len and the payload (or its base64) come back as they went in.
func FuzzPrinter(f *testing.F) {
	f.Add(uint64(1), int64(0), []byte("hello"))
	f.Add(uint64(1<<64-1), int64(-1<<63), []byte("\x1b[31m \xc2\x9b\x7f<>&\"\\"))
	f.Add(uint64(2), int64(1<<62), []byte{0xff, 0xfe})
	f.Add(uint64(3), int64(253402300800), []byte("\u2028\u2029\u0085\ufeff\x00"))
	f.Add(uint64(4), int64(-62135596801), []byte(nil))
	f.Fuzz(func(t *testing.T, seq uint64, at int64, payload []byte) {
		rec := quill.Record{Seq: seq, At: at, Payload: payload}
		for _, mode := range []struct{ json, b64 bool }{{false, false}, {false, true}, {true, false}, {true, true}} {
			var out, want bytes.Buffer
			p := newPrinter(&out, mode.json, mode.b64)
			if err := p.print(rec); err != nil {
				t.Fatal(err)
			}
			if err := p.flush(); err != nil {
				t.Fatal(err)
			}
			rw := bufio.NewWriter(&want)
			if err := refPrint(rw, mode.json, mode.b64, rec); err != nil {
				t.Fatal(err)
			}
			rw.Flush()
			b := out.Bytes()
			if !bytes.Equal(b, want.Bytes()) {
				t.Fatalf("json=%v b64=%v: got %q, reference %q", mode.json, mode.b64, b, want.Bytes())
			}
			if len(b) == 0 || b[len(b)-1] != '\n' || bytes.Count(b, []byte("\n")) != 1 {
				t.Fatalf("json=%v b64=%v: not exactly one line: %q", mode.json, mode.b64, b)
			}
			for i, c := range b[:len(b)-1] {
				if (c < 0x20 && (c != '\t' || mode.json)) || c == 0x7f {
					t.Fatalf("json=%v b64=%v: raw control %#x at %d in %q", mode.json, mode.b64, c, i, b)
				}
			}
			for _, r := range string(b) {
				if r >= 0x80 && r <= 0x9f {
					t.Fatalf("json=%v b64=%v: raw C1 control %U in %q", mode.json, mode.b64, r, b)
				}
			}
			if !mode.json {
				if bytes.Count(b, []byte("\t")) != 2 {
					t.Fatalf("text form: want three tab-separated fields: %q", b)
				}
				continue
			}
			var m struct {
				Seq        uint64  `json:"seq"`
				At         int64   `json:"at"`
				Len        int     `json:"len"`
				Payload    *string `json:"payload"`
				PayloadB64 *string `json:"payload_b64"`
			}
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatalf("invalid JSON %q: %v", b, err)
			}
			if m.Seq != seq || m.At != at || m.Len != len(payload) {
				t.Fatalf("lossy header: %q", b)
			}
			if (m.Payload == nil) == (m.PayloadB64 == nil) {
				t.Fatalf("want exactly one of payload and payload_b64: %q", b)
			}
			if m.Payload != nil && (mode.b64 || !utf8.Valid(payload) || *m.Payload != string(payload)) {
				t.Fatalf("lossy or misplaced payload: %q", b)
			}
		}
	})
}
