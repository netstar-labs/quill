package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
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
	return &printer{w: bufio.NewWriter(w), json: jsonMode, b64: b64}
}

func (p *printer) flush() error { return p.w.Flush() }

// print appends the record to p.buf, which is kept between records, and writes it with one
// call. The bytes are what encoding/json and strconv.Quote would produce for the same
// values (the JSON form also escapes DEL and C1 controls, see appendJSONString): the fixed
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
// < > & and the line and paragraph separators U+2028 and U+2029. It also escapes DEL and the C1 controls
// U+0080 to U+009F, which encoding/json passes and a terminal may act on (stage 2 audit A1-50, decision 35).
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
		case c == 0xc2 && i+1 < len(s) && s[i+1]&^0x1f == 0x80:
			b = append(b, s[start:i]...)
			c = s[i+1]
			b = append(b, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			i++
		case c == 0xe2 && i+2 < len(s) && s[i+1] == 0x80 && s[i+2]&^1 == 0xa8:
			b = append(b, s[start:i]...)
			b = append(b, '\\', 'u', '2', '0', '2', hexDigits[s[i+2]&0xf])
			i += 2
		default:
			continue // 0xc2 or 0xe2 that starts nothing to escape: part of the plain run
		}
		start = i + 1
	}
	return append(b, s[start:]...)
}

// jsonPlain marks the bytes appendJSONString may copy as they are without looking at the next
// ones: printable ASCII except the quote, the backslash and < > &, and every byte of 0x80 or
// above except 0xc2 and 0xe2, the two that can begin a C1 control or U+2028 and U+2029.
var jsonPlain = func() (t [256]bool) {
	for c := 0x20; c < 0x7f; c++ {
		t[c] = c != '"' && c != '\\' && c != '<' && c != '>' && c != '&'
	}
	for c := 0x80; c < 0x100; c++ {
		t[c] = c != 0xc2 && c != 0xe2
	}
	return
}()

// appendQuoted appends s as strconv.Quote would, for valid UTF-8 text: printable runes
// as they are, the rest escaped.
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
		if strconv.IsPrint(r) {
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
