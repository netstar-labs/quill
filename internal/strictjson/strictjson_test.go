package strictjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

type tail struct {
	Seq    uint64 `json:"seq"`
	Seg    uint64 `json:"seg"`
	Off    uint64 `json:"off"`
	MaxSeg uint64 `json:"maxseg"`
}

func TestUnmarshalAcceptsExactlyTheExpectedShape(t *testing.T) {
	var v tail
	if err := Unmarshal([]byte(` {"seq":1,"seg":2,"off":3,"maxseg":4} `+"\n"), &v); err != nil {
		t.Fatal(err)
	}
	if v != (tail{1, 2, 3, 4}) {
		t.Fatalf("got %+v", v)
	}
}

func TestUnmarshalRefusesEveryDamagedShape(t *testing.T) {
	cases := map[string]string{
		"unknown field":     `{"seq":1,"seg":2,"off":3,"maxseg":4,"extra":5}`,
		"trailing value":    `{"seq":1,"seg":2,"off":3,"maxseg":4}{}`,
		"trailing garbage":  `{"seq":1,"seg":2,"off":3,"maxseg":4} x`,
		"negative":          `{"seq":-1,"seg":2,"off":3,"maxseg":4}`,
		"float":             `{"seq":1.5,"seg":2,"off":3,"maxseg":4}`,
		"string for number": `{"seq":"1","seg":2,"off":3,"maxseg":4}`,
		"too large":         `{"seq":18446744073709551616,"seg":2,"off":3,"maxseg":4}`,
		"truncated":         `{"seq":1,"seg":2,"off":3,"max`,
		"empty":             ``,
		"zeros":             "\x00\x00\x00",
		"array":             `[1,2,3,4]`,
		"duplicate key":     `{"seq":1,"seq":2,"seg":2,"off":3,"maxseg":4}`,
	}
	for name, in := range cases {
		var v tail
		if err := Unmarshal([]byte(in), &v); err == nil {
			t.Errorf("%s: accepted %q as %+v", name, in, v)
		}
	}
	var v tail
	if err := Unmarshal([]byte(`{"seq":1,"seg":2,"off":3,"maxseg":4} x`), &v); !errors.Is(err, ErrTrailing) {
		t.Fatalf("trailing: %v", err)
	}
	if err := Unmarshal([]byte("\"\xff\""), new(string)); !errors.Is(err, ErrBadString) {
		t.Fatalf("invalid utf-8: %v", err)
	}
	var nested struct {
		A map[string]any `json:"a"`
	}
	if err := Unmarshal([]byte(`{"a":{"x":1,"y":{"x":2,"x":3}}}`), &nested); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("nested duplicate: %v", err)
	}
	if err := Unmarshal([]byte(`{"a":{"x":1,"y":{"x":2},"z":[{"x":1},{"x":2}]}}`), &nested); err != nil {
		t.Fatalf("same key in different objects: %v", err)
	}
}

// encoding/json matches field names case-insensitively, so a key that differs only in case
// from an earlier one would silently win; a repeated key is refused however it is spelled.
func TestCaseVariantDuplicateKeyIsRefused(t *testing.T) {
	cases := map[string]string{
		"upper":          `{"seq":1,"SEQ":2,"seg":0,"off":0,"maxseg":0}`,
		"upper first":    `{"SEQ":1,"seq":2,"seg":0,"off":0,"maxseg":0}`,
		"mixed":          `{"seq":1,"Seq":2,"seg":0,"off":0,"maxseg":0}`,
		"long s":         `{"seq":1,"\u017feq":2,"seg":0,"off":0,"maxseg":0}`,
		"kelvin sign":    `{"seq":1,"seg":0,"off":0,"maxseg":0,"k":1,"\u212a":2}`,
		"escaped":        `{"seq":1,"\u0053EQ":2,"seg":0,"off":0,"maxseg":0}`,
		"nested variant": `{"seq":1,"seg":0,"off":0,"maxseg":0,"a":{"y":1,"Y":2}}`,
	}
	for name, in := range cases {
		var v struct {
			Seq, Seg, Off, MaxSeg uint64
			K                     uint64
			A                     map[string]uint64
		}
		if err := Unmarshal([]byte(in), &v); !errors.Is(err, ErrDuplicateKey) {
			t.Errorf("%s: %q: got %v, want ErrDuplicateKey", name, in, err)
		}
	}
}

func TestAGenuineReplacementCharacterAndAValidPairStay(t *testing.T) {
	var s string
	for _, in := range []string{`"�"`, "\"ï¿½\"", `"😀"`, `"a\\u"`, `"\\ud800"`} {
		if err := Unmarshal([]byte(in), &s); err != nil {
			t.Errorf("%s: %v", in, err)
		}
	}
}

// A closed array or object is a consumed value: the enclosing object must expect a key
// again, or every key after the first container-valued member goes unrecorded.
func TestDuplicateKeyAfterAContainerValueIsRefused(t *testing.T) {
	cases := map[string]string{
		"after empty array":    `{"a":[],"b":1,"B":2}`,
		"after empty object":   `{"a":{},"b":1,"b":2}`,
		"key of the container": `{"a":{"x":1},"a":2}`,
		"after nested array":   `{"a":[1,[2,3],{"k":1}],"b":1,"b":2}`,
		"in an array element":  `[{"a":[],"b":1,"b":2}]`,
		"after two containers": `{"a":[],"b":{},"c":1,"c":2}`,
	}
	for name, in := range cases {
		if err := checkDuplicateKeys([]byte(in)); !errors.Is(err, ErrDuplicateKey) {
			t.Errorf("%s: %q: got %v, want ErrDuplicateKey", name, in, err)
		}
	}
	for _, in := range []string{`{"a":[],"b":1,"c":{}}`, `{"a":{"x":1},"b":{"x":2}}`, `[{"a":1},{"a":2}]`} {
		if err := checkDuplicateKeys([]byte(in)); err != nil {
			t.Errorf("%q: got %v, want nil", in, err)
		}
	}
}

// textField declares the member the surrogate rows use, so that a refusal can only come from
// the string checks: the rows are otherwise a valid document for it.
type textField struct {
	X string `json:"x"`
}

// esc builds a \u escape at run time, so no source line holds a surrogate escape literally.
func esc(hex ...string) string {
	var b strings.Builder
	for _, h := range hex {
		b.WriteString(`\u` + h)
	}
	return b.String()
}

func TestSurrogateEscapesAreCheckedInADeclaredField(t *testing.T) {
	doc := func(inner string) []byte { return []byte(`{"x":"` + inner + `"}`) }
	refused := map[string]string{
		"lone low, first":        esc("dc00"),
		"lone low, last":         esc("dfff"),
		"lone high, first":       esc("d800"),
		"lone high, last":        esc("dbff"),
		"high then letter":       esc("d800") + "x",
		"high then plain escape": esc("d800", "0041"),
		"high then high":         esc("d800", "d800"),
		"high then e000":         esc("d800", "e000"),
		"high, junk, then low":   esc("d800") + `xu` + "dc00",
		"high, newline, low":     esc("d800") + `\n` + "dc00",
		"high then dbff":         esc("d800", "dbff"),
		"high then short escape": esc("d800") + `\u00`,
		"low after a pair":       esc("d800", "dc00", "dc00"),
		"invalid utf-8":          "\xff",
	}
	for name, in := range refused {
		var v textField
		if err := Unmarshal(doc(in), &v); !errors.Is(err, ErrBadString) {
			t.Errorf("%s: %q: got %v, want ErrBadString", name, in, err)
		}
	}
	accepted := map[string]struct{ in, want string }{
		"pair, lowest low":   {esc("d800", "dc00"), "\U00010000"},
		"pair, highest low":  {esc("d800", "dfff"), "\U000103ff"},
		"pair, highest":      {esc("dbff", "dfff"), "\U0010ffff"},
		"just below the run": {esc("d7ff"), "\ud7ff"},
		"just above the run": {esc("e000"), "\ue000"},
		"replacement":        {esc("fffd"), "\ufffd"},
		"pair then text":     {esc("d83d", "de00") + "a", "\U0001f600a"},
	}
	for name, c := range accepted {
		var v textField
		if err := Unmarshal(doc(c.in), &v); err != nil {
			t.Errorf("%s: %q: %v", name, c.in, err)
		} else if v.X != c.want {
			t.Errorf("%s: got %q, want %q", name, v.X, c.want)
		}
	}
}

// checkEscapes reads past the end of what it was given only through hex4 and the pair length
// test, so the shortest inputs are the ones that pin them.
func TestCheckEscapesAtTheEndOfData(t *testing.T) {
	for in, want := range map[string]error{
		esc("d800"):                       ErrBadString,
		esc("d800") + `\u00`:              ErrBadString,
		esc("d800") + `\udc0`:             ErrBadString,
		esc("d800", "dc00"):               nil,
		esc("dc00"):                       ErrBadString,
		esc("d7ff"):                       nil,
		esc("d800")[:5]:                   nil,
		esc("dc00")[:5]:                   nil,
		`\`:                               nil,
		`\u`:                              nil,
		`\\` + esc("d800"):                ErrBadString,
		`\\` + esc("d800", "dc00"):        nil,
		esc("0041") + esc("d800", "dc00"): nil,
	} {
		if err := checkEscapes([]byte(in)); err != want {
			t.Errorf("%q: got %v, want %v", in, err, want)
		}
	}
}

func TestDuplicateKeyAfterAStringValueIsRefused(t *testing.T) {
	var v struct {
		ID string `json:"id"`
		N  int    `json:"n"`
	}
	for _, in := range []string{
		`{"id":"abc","n":1,"n":2}`,
		`{"id":"abc","id":"def","n":1}`,
		`{"id":"a","n":"1","n":2}`,
	} {
		if err := Unmarshal([]byte(in), &v); !errors.Is(err, ErrDuplicateKey) {
			t.Errorf("%q: got %v, want ErrDuplicateKey", in, err)
		}
	}
}

// An array is not an object: its string elements are values, never keys, so repeating one is fine.
func TestStringValuesAreNeverDuplicateKeys(t *testing.T) {
	var v struct {
		A []string `json:"a"`
		B int      `json:"b"`
	}
	for _, in := range []string{`{"a":["x","x","x"],"b":1}`, `{"b":1,"a":["x","x"]}`} {
		if err := Unmarshal([]byte(in), &v); err != nil {
			t.Errorf("%q: %v", in, err)
		}
	}
	// A string value is a value even when it spells a key already seen.
	var m map[string]string
	for _, in := range []string{`{"a":"x","b":"x"}`, `{"id":"id","n":"n"}`, `{"a":"b","b":"a"}`} {
		if err := Unmarshal([]byte(in), &m); err != nil {
			t.Errorf("%q: %v", in, err)
		}
	}
	var raw []string
	if err := Unmarshal([]byte(`["x","x","x"]`), &raw); err != nil || len(raw) != 3 {
		t.Errorf("top-level array: %v %v", raw, err)
	}
}

// FuzzUnmarshalNeverPanics also holds what Unmarshal accepts to its contract: whatever it
// accepts into an untyped value is one valid JSON document in valid UTF-8, with no lone
// surrogate escape and no key repeated under case folding, as acceptedViolation re-checks.
func FuzzUnmarshalNeverPanics(f *testing.F) {
	f.Add([]byte(`{"seq":1,"seg":2,"off":3,"maxseg":4}`))
	f.Add([]byte(`"𐀀"`))
	f.Add([]byte(`\u`))
	f.Add([]byte(`{"a":"` + esc("d83d", "de00") + `","b":[{"c":1},{"c":2}],"d":{}}`))
	f.Add([]byte(`{"a":1,"A":2}`))
	f.Add([]byte(`{"a":"` + esc("d800") + `"}`))
	f.Add([]byte(`["` + esc("dc00") + `"]`))
	f.Add([]byte(`{"a":1} {}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var v tail
		_ = Unmarshal(data, &v)
		var s string
		_ = Unmarshal(data, &s)
		var a any
		if Unmarshal(data, &a) == nil {
			if msg := acceptedViolation(data); msg != "" {
				t.Fatalf("accepted %q: %s", data, msg)
			}
		}
	})
}

// acceptedViolation re-checks, by a route that shares no code with Unmarshal, what an input
// Unmarshal accepted must satisfy. It returns "" when the input does.
func acceptedViolation(data []byte) string {
	if !utf8.Valid(data) {
		return "invalid UTF-8"
	}
	if !json.Valid(data) { // exactly one value, nothing after it
		return "not exactly one valid JSON value"
	}
	if msg := surrogateViolation(data); msg != "" {
		return msg
	}
	if _, msg := scanValue(data, skipSpace(data, 0)); msg != "" {
		return msg
	}
	return ""
}

func skipSpace(b []byte, i int) int {
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r') {
		i++
	}
	return i
}

// surrogateViolation walks the strings of a valid document and checks that every surrogate
// escape is half of an escaped pair, high first.
func surrogateViolation(b []byte) string {
	hex := func(i int) (uint64, bool) {
		if i+4 > len(b) {
			return 0, false
		}
		v, err := strconv.ParseUint(string(b[i:i+4]), 16, 16)
		return v, err == nil
	}
	inString := false
	for i := 0; i < len(b); i++ {
		switch {
		case !inString:
			inString = b[i] == '"'
		case b[i] == '"':
			inString = false
		case b[i] == '\\' && b[i+1] != 'u':
			i++
		case b[i] == '\\':
			v, ok := hex(i + 2)
			if !ok {
				return fmt.Sprintf("bad \\u escape at %d in a valid document", i)
			}
			i += 5 // the last hex digit
			switch {
			case v >= 0xDC00 && v <= 0xDFFF:
				return fmt.Sprintf("lone low surrogate at %d", i-5)
			case v >= 0xD800 && v <= 0xDBFF:
				lo, ok := hex(i + 3)
				if !ok || b[i+1] != '\\' || b[i+2] != 'u' || lo < 0xDC00 || lo > 0xDFFF {
					return fmt.Sprintf("lone high surrogate at %d", i-5)
				}
				i += 6
			}
		}
	}
	return ""
}

// scanValue walks one value of a valid document from b[i] and returns the index after it. An
// object that repeats a key under simple case folding is a violation.
func scanValue(b []byte, i int) (int, string) {
	switch b[i] {
	case '"':
		return stringEnd(b, i), ""
	case '[':
		if i = skipSpace(b, i+1); b[i] == ']' {
			return i + 1, ""
		}
		for {
			end, msg := scanValue(b, skipSpace(b, i))
			if msg != "" {
				return 0, msg
			}
			i = skipSpace(b, end) + 1 // the comma or the closing bracket
			if b[i-1] == ']' {
				return i, ""
			}
		}
	case '{':
		var keys []string
		if i = skipSpace(b, i+1); b[i] == '}' {
			return i + 1, ""
		}
		for {
			i = skipSpace(b, i)
			end := stringEnd(b, i)
			var k string
			if err := json.Unmarshal(b[i:end], &k); err != nil {
				return 0, "key does not unquote: " + err.Error()
			}
			for _, prev := range keys {
				if strings.EqualFold(prev, k) {
					return 0, fmt.Sprintf("duplicate key %q and %q", prev, k)
				}
			}
			keys = append(keys, k)
			i = skipSpace(b, end) + 1 // the colon
			vend, msg := scanValue(b, skipSpace(b, i))
			if msg != "" {
				return 0, msg
			}
			i = skipSpace(b, vend) + 1 // the comma or the closing brace
			if b[i-1] == '}' {
				return i, ""
			}
		}
	}
	// A number, true, false or null ends at a delimiter.
	for i < len(b) && !strings.ContainsRune(",]} \t\r\n", rune(b[i])) {
		i++
	}
	return i, ""
}

// stringEnd returns the index after the string that opens at b[i].
func stringEnd(b []byte, i int) int {
	for i++; b[i] != '"'; i++ {
		if b[i] == '\\' {
			i++
		}
	}
	return i + 1
}
