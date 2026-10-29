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
	"testing"
	"time"
	"unicode/utf8"

	"github.com/netstar-labs/quill"
)

// refPrint is the encoding/json, strconv.Quote and fmt formulation the printer's appenders
// must match byte for byte (decision 35).
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

func BenchmarkPrintJSONText(b *testing.B) { benchPrint(b, true, false, benchText) }
func BenchmarkPrintJSONBin(b *testing.B)  { benchPrint(b, true, false, benchBin) }
func BenchmarkPrintTextText(b *testing.B) { benchPrint(b, false, false, benchText) }
func BenchmarkPrintTextBin(b *testing.B)  { benchPrint(b, false, false, benchBin) }
