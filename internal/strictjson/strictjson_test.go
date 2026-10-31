package strictjson

import (
	"errors"
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
		"unknown field":      `{"seq":1,"seg":2,"off":3,"maxseg":4,"extra":5}`,
		"trailing value":     `{"seq":1,"seg":2,"off":3,"maxseg":4}{}`,
		"trailing garbage":   `{"seq":1,"seg":2,"off":3,"maxseg":4} x`,
		"negative":           `{"seq":-1,"seg":2,"off":3,"maxseg":4}`,
		"float":              `{"seq":1.5,"seg":2,"off":3,"maxseg":4}`,
		"string for number":  `{"seq":"1","seg":2,"off":3,"maxseg":4}`,
		"too large":          `{"seq":18446744073709551616,"seg":2,"off":3,"maxseg":4}`,
		"truncated":          `{"seq":1,"seg":2,"off":3,"max`,
		"empty":              ``,
		"zeros":              "\x00\x00\x00",
		"invalid utf-8":      "{\"seq\":1,\"seg\":2,\"off\":3,\"maxseg\":4,\"x\":\"\xff\"}",
		"lone low surrogate": `{"seq":1,"seg":2,"off":3,"maxseg":4,"x":"\udc00"}`,
		"lone high":          `{"seq":1,"seg":2,"off":3,"maxseg":4,"x":"\ud800"}`,
		"high then non-low":  `{"seq":1,"seg":2,"off":3,"maxseg":4,"x":"\ud800A"}`,
		"array":              `[1,2,3,4]`,
		"duplicate key":      `{"seq":1,"seq":2,"seg":2,"off":3,"maxseg":4}`,
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
