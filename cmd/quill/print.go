package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/netstar-labs/quill"
)

// printer writes records in the text form or as newline-delimited JSON (decision 35). Both
// are lossless and never pass a raw control character or invalid UTF-8 to the terminal: in
// JSON, a payload that is valid UTF-8 is a JSON string ("payload") and any other is base64
// ("payload_b64"); -b64 forces base64. The text form prints seq, the time and the payload
// quoted the same way, with the base64 form marked.
type printer struct {
	w    *bufio.Writer
	json bool
	b64  bool
	buf  []byte // the record being built, reused
}

func newPrinter(w io.Writer, jsonMode, b64 bool) *printer {
	// 64 KiB, not bufio's 4 KiB: a large output costs a sixteenth of the write calls. A
	// follow flushes after every record (read.go), so an interactive tail does not wait on it.
	return &printer{w: bufio.NewWriterSize(w, 64<<10), json: jsonMode, b64: b64}
}

func (p *printer) flush() error { return p.w.Flush() }

// print appends the record to p.buf, which is kept between records, and writes it with one
// call. The bytes are what encoding/json and strconv.Quote would produce for the same
// values (the JSON form also escapes DEL, C1 controls and format characters, see appendJSONString): the fixed
// field order of decision 35 is seq, at, time, len, then exactly one of payload and payload_b64.
func (p *printer) print(rec quill.Record) error {
	text := !p.b64 && utf8.Valid(rec.Payload)
	b := p.buf[:0]
	if p.json {
		b = append(b, `{"seq":`...)
		b = strconv.AppendUint(b, rec.Seq, 10)
		b = append(b, `,"at":`...)
		b = strconv.AppendInt(b, rec.At, 10)
		b = append(b, `,"time":"`...)
		b = time.Unix(rec.At, 0).UTC().AppendFormat(b, time.RFC3339)
		b = append(b, `","len":`...)
		b = strconv.AppendInt(b, int64(len(rec.Payload)), 10)
		if text {
			b = append(b, `,"payload":"`...)
			b = appendJSONString(b, rec.Payload)
		} else {
			b = append(b, `,"payload_b64":"`...)
			b = base64.StdEncoding.AppendEncode(b, rec.Payload)
		}
		b = append(b, "\"}\n"...)
	} else {
		b = strconv.AppendUint(b, rec.Seq, 10)
		b = append(b, '\t')
		b = time.Unix(rec.At, 0).UTC().AppendFormat(b, time.RFC3339)
		b = append(b, '\t')
		if text {
			b = appendQuoted(b, rec.Payload)
		} else {
			b = append(b, "b64:"...)
			b = base64.StdEncoding.AppendEncode(b, rec.Payload)
		}
		b = append(b, '\n')
	}
	p.buf = b
	_, err := p.w.Write(b)
	return err
}

const hexDigits = "0123456789abcdef"

// appendJSONString appends the body of a JSON string (without the quotes) for valid UTF-8 text, escaping as
// encoding/json does by default: the quote, the backslash, the controls, the HTML characters
// < > & and the line and paragraph separators U+2028 and U+2029. It also escapes what encoding/json passes
// and a terminal or a bidi-aware viewer may act on: DEL, the C1 controls U+0080 to U+009F (stage 2 audit
// A1-50) and the Unicode format characters (category Cf: the bidi controls U+202A to U+202E and U+2066 to
// U+2069, the zero-width characters U+200B to U+200F, U+2060, U+FEFF, U+061C and the rest), which the text
// output escapes because strconv.IsPrint rejects them (stage 2 audit A3-24, decision 35). A character
// above U+FFFF is written as a surrogate pair, as encoding/json does, so the decoded value never changes.
func appendJSONString(b, s []byte) []byte {
	start := 0 // s[start:i] is plain text not yet appended
	for i := 0; i < len(s); i++ {
		c := s[i]
		if jsonPlain[c] {
			continue
		}
		switch {
		case c == '"' || c == '\\':
			b = append(b, s[start:i]...)
			b = append(b, '\\', c)
		case c == '\b':
			b = append(b, s[start:i]...)
			b = append(b, '\\', 'b')
		case c == '\f':
			b = append(b, s[start:i]...)
			b = append(b, '\\', 'f')
		case c == '\n':
			b = append(b, s[start:i]...)
			b = append(b, '\\', 'n')
		case c == '\r':
			b = append(b, s[start:i]...)
			b = append(b, '\\', 'r')
		case c == '\t':
			b = append(b, s[start:i]...)
			b = append(b, '\\', 't')
		case c < 0x20 || c == 0x7f || c == '<' || c == '>' || c == '&':
			b = append(b, s[start:i]...)
			b = append(b, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
		default:
			// A lead byte that may begin a character to escape (see jsonPlain); most of the time it
			// begins one that stays as it is, and the run goes on.
			r, n := utf8.DecodeRune(s[i:])
			if !jsonEscapeRune(r) {
				i += n - 1
				continue
			}
			b = append(b, s[start:i]...)
			if r >= 0x10000 {
				r -= 0x10000
				b = appendU4(b, 0xd800+r>>10)
				b = appendU4(b, 0xdc00+r&0x3ff)
			} else {
				b = appendU4(b, r)
			}
			i += n - 1
		}
		start = i + 1
	}
	return append(b, s[start:]...)
}

// appendU4 appends \uXXXX, four lower-case hex digits, for a value below 0x10000.
func appendU4(b []byte, r rune) []byte {
	return append(b, '\\', 'u', hexDigits[r>>12], hexDigits[r>>8&0xf], hexDigits[r>>4&0xf], hexDigits[r&0xf])
}

// jsonEscapeRune reports whether appendJSONString escapes r although it is not a byte below 0x80: a C1
// control, U+2028 or U+2029, or a format character (Cf). Every Cf code point is rejected by
// strconv.IsPrint, so the text output escapes the same characters; a test pins that.
func jsonEscapeRune(r rune) bool {
	return r >= 0x80 && r <= 0x9f || r == 0x2028 || r == 0x2029 || unicode.Is(unicode.Cf, r)
}

// jsonPlain marks the bytes appendJSONString may copy as they are without looking at the next
// ones: printable ASCII except the quote, the backslash and < > &, and every byte of 0x80 or
// above except the lead bytes of the characters jsonEscapeRune accepts. Those lead bytes are
// derived from the same tables, not listed: the C1 controls and U+2028 and U+2029 begin with 0xc2
// and 0xe2, the format characters with a handful of others (Cyrillic, Greek, Latin and CJK text
// begin with none of them, so it is still copied without a lookup).
var jsonPlain = func() (t [256]bool) {
	for c := 0x20; c < 0x7f; c++ {
		t[c] = c != '"' && c != '\\' && c != '<' && c != '>' && c != '&'
	}
	for c := 0x80; c < 0x100; c++ {
		t[c] = true
	}
	mark := func(r rune) {
		var e [utf8.UTFMax]byte
		utf8.EncodeRune(e[:], r)
		t[e[0]] = false
	}
	mark(0x80)
	mark(0x2028)
	for _, g := range unicode.Cf.R16 {
		for r := rune(g.Lo); r <= rune(g.Hi); r += rune(g.Stride) {
			mark(r)
		}
	}
	for _, g := range unicode.Cf.R32 {
		for r := rune(g.Lo); r <= rune(g.Hi); r += rune(g.Stride) {
			mark(r)
		}
	}
	return
}()

// safeText is how the text output prints a string that came from the disk (a directory entry, a
// file's content, or an error message built from one): s as it is when it is valid UTF-8 and every
// rune is printable (strconv.IsPrint, which admits the ASCII space), so the names the library creates print
// unchanged, otherwise s quoted by appendQuoted as a record payload is. No newline, ESC or other
// control, C1 or bidi control, and no invalid UTF-8, then reaches the terminal (stage 2 audit
// A3-3, decision 35).
func safeText(s string) string {
	plain := utf8.ValidString(s)
	for _, r := range s {
		if !strconv.IsPrint(r) {
			plain = false
			break
		}
	}
	if plain {
		return s
	}
	return string(appendQuoted(nil, []byte(s)))
}

// appendQuoted appends s as strconv.Quote would: printable runes as they are, the rest
// escaped, and a byte that is not valid UTF-8 as \xNN.
func appendQuoted(b, s []byte) []byte {
	b = append(b, '"')
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			// A run of printable ASCII other than the quote and the backslash is copied whole.
			j := i
			for j < len(s) && quotedPlain[s[j]] {
				j++
			}
			if j > i {
				b = append(b, s[i:j]...)
				i = j
				continue
			}
			i++
			switch {
			case c == '"' || c == '\\':
				b = append(b, '\\', c)
			case c >= 0x20 && c < 0x7f:
				b = append(b, c)
			default:
				b = appendQuotedControl(b, c)
			}
			continue
		}
		r, n := utf8.DecodeRune(s[i:])
		if r == utf8.RuneError && n == 1 {
			b = append(b, '\\', 'x', hexDigits[c>>4], hexDigits[c&0xf])
		} else if strconv.IsPrint(r) {
			b = append(b, s[i:i+n]...)
		} else if r < 0x10000 {
			b = append(b, '\\', 'u', hexDigits[r>>12], hexDigits[r>>8&0xf], hexDigits[r>>4&0xf], hexDigits[r&0xf])
		} else {
			b = append(b, '\\', 'U', '0', '0', hexDigits[r>>20&0xf], hexDigits[r>>16&0xf], hexDigits[r>>12&0xf], hexDigits[r>>8&0xf], hexDigits[r>>4&0xf], hexDigits[r&0xf])
		}
		i += n
	}
	return append(b, '"')
}

// quotedPlain marks the ASCII bytes appendQuoted copies as they are.
var quotedPlain = func() (t [256]bool) {
	for c := 0x20; c < 0x7f; c++ {
		t[c] = c != '"' && c != '\\'
	}
	return
}()

// appendQuotedControl escapes an ASCII control character or DEL the way strconv.Quote does.
func appendQuotedControl(b []byte, c byte) []byte {
	switch c {
	case '\a':
		return append(b, '\\', 'a')
	case '\b':
		return append(b, '\\', 'b')
	case '\f':
		return append(b, '\\', 'f')
	case '\n':
		return append(b, '\\', 'n')
	case '\r':
		return append(b, '\\', 'r')
	case '\t':
		return append(b, '\\', 't')
	case '\v':
		return append(b, '\\', 'v')
	}
	return append(b, '\\', 'x', hexDigits[c>>4], hexDigits[c&0xf])
}

// printJSON writes one object and a newline.
func printJSON(c *cli, v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return c.errorf("%v", err)
	}
	fmt.Fprintln(c.out, string(b))
	return exitOK
}
