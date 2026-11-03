package main

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/netstar-labs/quill"
)

// formatControls returns every code point of the Unicode format category (Cf), which holds the bidi
// controls, the zero-width characters, the byte order mark and the tag characters. The text output
// escapes all of them because strconv.IsPrint rejects them; TestFormatControlsAreAllNonPrintable
// pins that, so the JSON output and the text output name the same set (stage 2 audit A3-24).
func formatControls() []rune {
	var set []rune
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if unicode.Is(unicode.Cf, r) {
			set = append(set, r)
		}
	}
	return set
}

func printJSONLine(t *testing.T, payload string) []byte {
	t.Helper()
	var got bytes.Buffer
	p := newPrinter(&got, true, false)
	if err := p.print(quill.Record{Seq: 1, At: 0, Payload: []byte(payload)}); err != nil {
		t.Fatal(err)
	}
	if err := p.flush(); err != nil {
		t.Fatal(err)
	}
	return got.Bytes()
}

// TestFormatControlsAreAllNonPrintable: the set the JSON output escapes (Cf) is a subset of what the
// text output escapes (not strconv.IsPrint). If a Go release made a Cf character printable the two
// outputs would disagree again, and this test says so.
func TestFormatControlsAreAllNonPrintable(t *testing.T) {
	set := formatControls()
	if len(set) < 100 {
		t.Fatalf("only %d format characters found", len(set))
	}
	for _, r := range set {
		if strconv.IsPrint(r) {
			t.Errorf("%U is Cf and printable: text mode passes it raw, so JSON must too, or the set must be listed", r)
		}
	}
}

// TestPrintJSONEscapesFormatAndBidi: no format or bidi control reaches the terminal raw in the -json
// payload, as none does in text mode, and the decoded value is the original payload.
func TestPrintJSONEscapesFormatAndBidi(t *testing.T) {
	for _, r := range formatControls() {
		payload := "a" + string(r) + "b"
		line := printJSONLine(t, payload)
		if bytes.Contains(line, []byte(string(r))) {
			t.Errorf("%U: raw bytes in %q", r, line)
			continue
		}
		var rec struct{ Payload string }
		if err := json.Unmarshal(line, &rec); err != nil || rec.Payload != payload {
			t.Errorf("%U: not lossless: %q, %v", r, rec.Payload, err)
		}
	}
}

// TestPrintJSONFormatControlsExact: the escaped forms, a BMP character as \uXXXX and a character above
// U+FFFF as a surrogate pair, the way encoding/json writes one.
func TestPrintJSONFormatControlsExact(t *testing.T) {
	payload := "x\u202ey\u200bz\ufeff\u061c\u2066\u00ad\U000e0041"
	want := `{"seq":1,"at":0,"time":"1970-01-01T00:00:00Z","len":` + strconv.Itoa(len(payload)) +
		`,"payload":"x\u202ey\u200bz\ufeff\u061c\u2066\u00ad\udb40\udc41"}` + "\n"
	if got := string(printJSONLine(t, payload)); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// TestPrintJSONMixedHostile: a payload that mixes every kind of character the printer treats
// specially, with the neighbours of the escaped ranges that stay raw, is lossless and holds no
// raw control, C1, line separator or format byte sequence.
func TestPrintJSONMixedHostile(t *testing.T) {
	payload := "\u202e\u200b\u202e" + `"quoted" <a&b>` + "\x1b[31m\u009b\u0085\x7f\u2028\u2029\ufeff" +
		"\u00e9\u65e5\u672c\u8a9e\u03a9\u0639\u00a0\u200a\u2065\u2070\U0001f600\u06dd\u0600" + "\u202a\u2069\u061c\u2060\U000e007f\u00c2\u00e2"
	line := printJSONLine(t, payload)
	var rec struct{ Payload string }
	if err := json.Unmarshal(line, &rec); err != nil || rec.Payload != payload {
		t.Fatalf("not lossless: %q, %v", rec.Payload, err)
	}
	for _, r := range string(line) {
		if r < 0x20 && r != '\n' || r == 0x7f || r >= 0x80 && r <= 0x9f || r == 0x2028 || r == 0x2029 || unicode.Is(unicode.Cf, r) {
			t.Fatalf("raw %U in %q", r, line)
		}
	}
	if !strings.Contains(string(line), "\u00e9\u65e5\u672c\u8a9e\u03a9\u0639") {
		t.Fatalf("printable text was escaped: %q", line)
	}
}
