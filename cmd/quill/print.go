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
}

func newPrinter(w io.Writer, jsonMode, b64 bool) *printer {
	return &printer{w: bufio.NewWriter(w), json: jsonMode, b64: b64}
}

func (p *printer) flush() { _ = p.w.Flush() }

// jsonRecord has the fixed field order of decision 35: seq, at, time, len, then exactly one
// of payload and payload_b64.
type jsonRecord struct {
	Seq        uint64  `json:"seq"`
	At         int64   `json:"at"`
	Time       string  `json:"time"`
	Len        int     `json:"len"`
	Payload    *string `json:"payload,omitempty"`
	PayloadB64 *string `json:"payload_b64,omitempty"`
}

func (p *printer) print(rec quill.Record) error {
	at := time.Unix(rec.At, 0).UTC().Format(time.RFC3339)
	text := !p.b64 && utf8.Valid(rec.Payload)
	if p.json {
		jr := jsonRecord{Seq: rec.Seq, At: rec.At, Time: at, Len: len(rec.Payload)}
		if text {
			s := string(rec.Payload)
			jr.Payload = &s
		} else {
			s := base64.StdEncoding.EncodeToString(rec.Payload)
			jr.PayloadB64 = &s
		}
		b, err := json.Marshal(jr) // a valid-UTF-8 string is never altered by the encoder
		if err != nil {
			return err
		}
		if _, err := p.w.Write(b); err != nil {
			return err
		}
		return p.w.WriteByte('\n')
	}
	var payload string
	if text {
		payload = strconv.Quote(string(rec.Payload)) // escapes control characters and quotes
	} else {
		payload = "b64:" + base64.StdEncoding.EncodeToString(rec.Payload)
	}
	_, err := fmt.Fprintf(p.w, "%d\t%s\t%s\n", rec.Seq, at, payload)
	return err
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
