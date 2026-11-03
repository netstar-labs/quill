package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/netstar-labs/quill"
)

// refPrint is the encoding/json, strconv.Quote and fmt formulation the printer's appenders
// must match byte for byte (decision 35), except that the JSON form also escapes DEL, the C1 controls and
// the Unicode format characters (category Cf).
func refPrint(w *bufio.Writer, jsonMode, b64 bool, rec quill.Record) error {
	at := time.Unix(rec.At, 0).UTC().Format(time.RFC3339)
	text := !b64 && utf8.Valid(rec.Payload)
	if jsonMode {
		type jsonRecord struct {
			Seq        uint64  `json:"seq"`
			At         int64   `json:"at"`
			Time       string  `json:"time"`
			Len        int     `json:"len"`
			Payload    *string `json:"payload,omitempty"`
			PayloadB64 *string `json:"payload_b64,omitempty"`
		}
		jr := jsonRecord{Seq: rec.Seq, At: rec.At, Time: at, Len: len(rec.Payload)}
		if text {
			s := string(rec.Payload)
			jr.Payload = &s
		} else {
			s := base64.StdEncoding.EncodeToString(rec.Payload)
			jr.PayloadB64 = &s
		}
		b, err := json.Marshal(jr)
		if err != nil {
			return err
		}
		// The one deliberate difference from encoding/json (stage 2 audit A1-50): DEL and the C1
		// controls U+0080 to U+009F are escaped. Only the payload can contain them.
		b = bytes.ReplaceAll(b, []byte{0x7f}, []byte(`\u007f`))
		for r := rune(0x80); r <= 0x9f; r++ {
			b = bytes.ReplaceAll(b, []byte(string(r)), []byte(fmt.Sprintf(`\u%04x`, r)))
		}
		// And, stage 2 audit A3-24, the format characters (bidi controls, zero-width characters, the
		// byte order mark), which the text form escapes because strconv.IsPrint rejects them: as \uXXXX,
		// a surrogate pair above U+FFFF. Written as a rune walk, apart from the printer's byte tables.
		var esc strings.Builder
		for _, r := range string(b) {
			if !unicode.Is(unicode.Cf, r) {
				esc.WriteRune(r)
				continue
			}
			for _, u := range utf16.Encode([]rune{r}) {
				fmt.Fprintf(&esc, `\u%04x`, u)
			}
		}
		b = []byte(esc.String())
		w.Write(b)
		return w.WriteByte('\n')
	}
	var payload string
	if text {
		payload = strconv.Quote(string(rec.Payload))
	} else {
		payload = "b64:" + base64.StdEncoding.EncodeToString(rec.Payload)
	}
	_, err := fmt.Fprintf(w, "%d\t%s\t%s\n", rec.Seq, at, payload)
	return err
}

func randPayload(rng *rand.Rand) []byte {
	pieces := []string{"abc", "hello world", "\"", "\\", "\b", "\f", "\n", "\r", "\t", "\x00", "\x1f", "\x7f",
		"<", ">", "&", "\u2028", "\u2029", "\u0080", "\u009f", "\u00a0", "\u00e9", "\ufffd", "\U0001F600",
		"\U0010ffff", "\u200b", "\ufeff", "'", "\xff", "\xc3", "\xe2\x82", "\x00\x01"}
	var b []byte
	for n := rng.Intn(12); n > 0; n-- {
		switch rng.Intn(4) {
		case 0:
			b = append(b, byte(rng.Intn(256)))
		case 1:
			b = utf8.AppendRune(b, rune(rng.Intn(0x110000)))
		default:
			b = append(b, pieces[rng.Intn(len(pieces))]...)
		}
	}
	return b
}

// TestPrintMatchesReference is the differential test: the appenders and the reference agree
// on the bytes for every mode.
func TestPrintMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	ats := []int64{0, 1, -1, -62135596800, 1700000000, 253402300799, 1 << 40}
	for _, mode := range []struct{ json, b64 bool }{{false, false}, {false, true}, {true, false}, {true, true}} {
		var got, want bytes.Buffer
		p := newPrinter(&got, mode.json, mode.b64)
		rw := bufio.NewWriter(&want)
		for i := 0; i < 20000; i++ {
			rec := quill.Record{Seq: rng.Uint64() >> uint(rng.Intn(64)), At: ats[rng.Intn(len(ats))] + int64(rng.Intn(100000)), Payload: randPayload(rng)}
			if i%50 == 0 {
				rec.Payload = nil
			}
			if err := p.print(rec); err != nil {
				t.Fatal(err)
			}
			if err := refPrint(rw, mode.json, mode.b64, rec); err != nil {
				t.Fatal(err)
			}
		}
		p.flush()
		rw.Flush()
		if !bytes.Equal(got.Bytes(), want.Bytes()) {
			g, w := got.Bytes(), want.Bytes()
			i := 0
			for i < len(g) && i < len(w) && g[i] == w[i] {
				i++
			}
			lo := max(0, i-60)
			t.Fatalf("json=%v b64=%v: output differs at byte %d:\n got %q\nwant %q", mode.json, mode.b64, i, g[lo:min(len(g), i+60)], w[lo:min(len(w), i+60)])
		}
	}
}

// TestPrintJSONEscapesC1AndDEL: the JSON form, like the text form, never passes DEL or a C1 control
// (U+0080 to U+009F, such as the CSI introducer U+009B) raw to the terminal; it stays valid JSON and lossless.
func TestPrintJSONEscapesC1AndDEL(t *testing.T) {
	payload := "a\u009b31mred\u0085b\x7f\u0080\u009f é"
	var got bytes.Buffer
	p := newPrinter(&got, true, false)
	if err := p.print(quill.Record{Seq: 1, At: 0, Payload: []byte(payload)}); err != nil {
		t.Fatal(err)
	}
	p.flush()
	want := `{"seq":1,"at":0,"time":"1970-01-01T00:00:00Z","len":` + strconv.Itoa(len(payload)) +
		`,"payload":"a\u009b31mred\u0085b\u007f\u0080\u009f` + " é" + `"}` + "\n"
	if got.String() != want {
		t.Fatalf("got %q\nwant %q", got.String(), want)
	}
	var rec struct{ Payload string }
	if err := json.Unmarshal(got.Bytes(), &rec); err != nil || rec.Payload != payload {
		t.Fatalf("not lossless: %q, %v", rec.Payload, err)
	}
}

func benchPrint(b *testing.B, jsonMode, b64 bool, payload []byte) {
	p := newPrinter(io.Discard, jsonMode, b64)
	rec := quill.Record{Seq: 123456, At: 1700000000, Payload: payload}
	b.ReportAllocs()
	for b.Loop() {
		if err := p.print(rec); err != nil {
			b.Fatal(err)
		}
	}
}

var (
	benchText = []byte("2024-05-01 12:00:00 INFO request served path=/api/v1/items status=200 \"quoted\"\n")
	benchBin  = []byte{0xff, 0xfe, 0x00, 0x01, 0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0, 0x7f, 0x00, 0x01, 0x02}
)

// countWriter records the size of every Write it receives.
type countWriter struct{ sizes []int }

func (c *countWriter) Write(p []byte) (int, error) {
	c.sizes = append(c.sizes, len(p))
	return len(p), nil
}

// The printer holds output in a 64 KiB buffer, so a large output reaches a pipe or file in
// writes of that size, not 4 KiB ones (L1-53); a flush still delivers what is held.
func TestPrinterWritesInLargeBlocks(t *testing.T) {
	var cw countWriter
	p := newPrinter(&cw, false, false)
	rec := quill.Record{Seq: 1, At: 1700000000, Payload: bytes.Repeat([]byte("x"), 75)}
	for i := 0; i < 20000; i++ {
		if err := p.print(rec); err != nil {
			t.Fatal(err)
		}
		rec.Seq++
	}
	if len(cw.sizes) == 0 {
		t.Fatal("nothing was written")
	}
	for _, n := range cw.sizes {
		if n < 32<<10 {
			t.Fatalf("a write of %d bytes before the flush; want blocks of at least 32 KiB", n)
		}
	}
	if err := p.flush(); err != nil {
		t.Fatal(err)
	}
	if p.w.Buffered() != 0 {
		t.Fatalf("%d bytes still held after flush", p.w.Buffered())
	}
}

func BenchmarkPrintJSONText(b *testing.B) { benchPrint(b, true, false, benchText) }
func BenchmarkPrintJSONBin(b *testing.B)  { benchPrint(b, true, false, benchBin) }
func BenchmarkPrintTextText(b *testing.B) { benchPrint(b, false, false, benchText) }
func BenchmarkPrintTextBin(b *testing.B)  { benchPrint(b, false, false, benchBin) }
