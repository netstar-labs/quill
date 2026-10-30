//go:build linux || darwin

// Package strictjson parses a state file and refuses anything but exactly what was expected:
// unknown fields, trailing data, invalid UTF-8 and lone surrogate escapes (which
// encoding/json would silently turn into U+FFFD). Unknown shapes are refused, not guessed
// (guarantee 9).
package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrBadString is returned for data holding invalid UTF-8 or a lone surrogate escape.
var ErrBadString = errors.New("strictjson: invalid UTF-8 or a lone surrogate escape")

// ErrTrailing is returned when more than one JSON value is present.
var ErrTrailing = errors.New("strictjson: trailing data after the value")

// ErrDuplicateKey is returned when an object repeats a key, however it is spelled in case:
// the decoder matches field names case-insensitively and would keep the last.
var ErrDuplicateKey = errors.New("strictjson: duplicate key")

// Unmarshal decodes exactly one JSON value into v. v should be a pointer to a struct:
// fields the struct does not declare are an error, as are a repeated key, a second value,
// invalid UTF-8 anywhere in data, and a \u escape that is a lone surrogate.
func Unmarshal(data []byte, v any) error {
	if !utf8.Valid(data) {
		return ErrBadString
	}
	if err := checkEscapes(data); err != nil {
		return err
	}
	if err := checkDuplicateKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return ErrTrailing
	}
	return nil
}

// checkEscapes walks the bytes, skipping the byte after each backslash, and reads each \u
// escape: a lone low surrogate, or a high one not followed by a low one, is refused. A genuine
// U+FFFD, raw or escaped, is valid text and stays. Syntax is left to the decoder.
func checkEscapes(data []byte) error {
	for i := 0; i+1 < len(data); i++ {
		if data[i] != '\\' {
			continue
		}
		if data[i+1] != 'u' {
			i++
			continue
		}
		r, ok := hex4(data[i+2:])
		if !ok {
			i++
			continue // a malformed escape is the decoder's syntax error
		}
		switch {
		case r >= 0xDC00 && r <= 0xDFFF:
			return ErrBadString
		case r >= 0xD800 && r <= 0xDBFF:
			if i+12 > len(data) || data[i+6] != '\\' || data[i+7] != 'u' {
				return ErrBadString
			}
			lo, ok := hex4(data[i+8:])
			if !ok || lo < 0xDC00 || lo > 0xDFFF {
				return ErrBadString
			}
			i += 11
		default:
			i += 5
		}
	}
	return nil
}

// checkDuplicateKeys walks the tokens and refuses an object that repeats a key, comparing
// keys by the case folding encoding/json uses to match field names, so seq and SEQ are one
// key. Syntax errors are left to the decoder, which reports them better.
func checkDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	type frame struct {
		object    bool
		keys      map[string]bool
		expectKey bool
	}
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return nil // the decoder will report the syntax error
		}
		top := (*frame)(nil)
		if len(stack) > 0 {
			top = stack[len(stack)-1]
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				stack = append(stack, &frame{object: true, keys: map[string]bool{}, expectKey: true})
				continue
			case '[':
				stack = append(stack, &frame{})
				continue
			default:
				stack = stack[:len(stack)-1]
			}
		case string:
			if top != nil && top.object && top.expectKey {
				k := strings.Map(foldRune, t)
				if top.keys[k] {
					return ErrDuplicateKey
				}
				top.keys[k] = true
				top.expectKey = false
				continue
			}
		}
		if top != nil && top.object {
			top.expectKey = true // a value was consumed; the next string is a key
		}
	}
}

// foldRune maps a rune to the smallest rune of its simple case-folding orbit, so that two
// keys the decoder would match to one field (K, k and the Kelvin sign; s and the long s)
// map to one string.
func foldRune(r rune) rune {
	m := r
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		if f < m {
			m = f
		}
	}
	return m
}

func hex4(b []byte) (uint64, bool) {
	if len(b) < 4 {
		return 0, false
	}
	r, err := strconv.ParseUint(string(b[:4]), 16, 32)
	return r, err == nil
}
