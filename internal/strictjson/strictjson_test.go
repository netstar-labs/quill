package strictjson

import (
	"errors"
	"strings"
	"testing"
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

func FuzzUnmarshalNeverPanics(f *testing.F) {
	f.Add([]byte(`{"seq":1,"seg":2,"off":3,"maxseg":4}`))
	f.Add([]byte(`"𐀀"`))
	f.Add([]byte(`\u`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var v tail
		_ = Unmarshal(data, &v)
		var s string
		_ = Unmarshal(data, &s)
	})
}
