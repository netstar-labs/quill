package quill

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"math/bits"
)

// Record is one entry in a log.
type Record struct {
	Seq     uint64 // per log, from 1, contiguous
	At      int64  // the writer's wall clock, whole Unix seconds
	Payload []byte // opaque, at most MaxPayloadBytes
}

// The frame (decision 3, docs/format.md):
//
//	frame = uvarint(bodyLen) | body | crc32(body)        crc is 4 bytes, big-endian
//	body  = uvarint(seq) | uvarint(at) | uvarint(len(payload)) | payload
//
// At is written as uvarint(uint64(at)), so a negative At takes ten bytes. The CRC is IEEE.

// crcTable is the frame's CRC table, pinned by the golden tests.
var crcTable = crc32.MakeTable(crc32.IEEE)

// maxFrameBody is the largest declared body length a reader trusts. A declared length over
// it is corruption, not an allocation (guarantee 7).
const maxFrameBody = 64 << 20

// maxPayloadBytes is the ceiling on a payload: a frame costs at most 32 bytes
// beyond the payload (a length prefix of at most 4, varints of at most 10, 10 and 4, and the CRC),
// so a legal record's frame always fits maxFrameBody (decision 69).
const maxPayloadBytes = maxFrameBody - 32

// maxVarint is the longest encoding of a uint64.
const maxVarint = binary.MaxVarintLen64

// errBadFrame marks a frame that cannot be decoded: a bad length, a CRC mismatch, a body
// that is not three varints and exactly the declared payload, or a sequence out of order.
var errBadFrame = errors.New("quill: bad frame")

// errTornFrame marks a frame that ends past the available bytes.
var errTornFrame = errors.New("quill: torn frame")

// uvarintLen returns how many bytes binary.AppendUvarint emits for v.
func uvarintLen(v uint64) int {
	return (bits.Len64(v|1) + 6) / 7
}

// frameLen is the encoded size of a record.
func frameLen(rec *Record) int {
	body := bodyLen(rec)
	return uvarintLen(uint64(body)) + body + 4
}

func bodyLen(rec *Record) int {
	return uvarintLen(rec.Seq) + uvarintLen(uint64(rec.At)) + uvarintLen(uint64(len(rec.Payload))) + len(rec.Payload)
}

// appendFrame encodes rec as a frame and appends it to dst. A caller reusing dst appends
// with no allocation.
func appendFrame(dst []byte, rec *Record) []byte {
	dst = binary.AppendUvarint(dst, uint64(bodyLen(rec)))
	start := len(dst)
	dst = binary.AppendUvarint(dst, rec.Seq)
	dst = binary.AppendUvarint(dst, uint64(rec.At))
	dst = binary.AppendUvarint(dst, uint64(len(rec.Payload)))
	dst = append(dst, rec.Payload...)
	return binary.BigEndian.AppendUint32(dst, crc32.Checksum(dst[start:], crcTable))
}

// decodeFrame decodes the frame at the start of data. It returns the record, whose Payload
// aliases data, and the frame's total length. errTornFrame means data ends inside the frame
// (more bytes might complete it); errBadFrame means the bytes are not a frame.
func decodeFrame(data []byte) (Record, int, error) {
	bodyLen, n := binary.Uvarint(data)
	switch {
	case n < 0, n > 0 && bodyLen > maxFrameBody:
		return Record{}, 0, errBadFrame
	case n == 0:
		if len(data) >= maxVarint {
			return Record{}, 0, errBadFrame
		}
		return Record{}, 0, errTornFrame
	}
	total := n + int(bodyLen) + 4
	if len(data) < total {
		return Record{}, 0, errTornFrame
	}
	body := data[n : n+int(bodyLen)]
	if binary.BigEndian.Uint32(data[n+int(bodyLen):total]) != crc32.Checksum(body, crcTable) {
		return Record{}, 0, errBadFrame
	}
	rec, err := decodeBody(body)
	if err != nil {
		return Record{}, 0, err
	}
	return rec, total, nil
}

// decodeBody decodes a body whose CRC has been checked. The payload must end exactly at the
// body's end: a writer never leaves slack, so slack is damage.
func decodeBody(body []byte) (Record, error) {
	seq, n := binary.Uvarint(body)
	if n <= 0 {
		return Record{}, errBadFrame
	}
	body = body[n:]
	at, n := binary.Uvarint(body)
	if n <= 0 {
		return Record{}, errBadFrame
	}
	body = body[n:]
	plen, n := binary.Uvarint(body)
	if n <= 0 || uint64(len(body[n:])) != plen {
		return Record{}, errBadFrame
	}
	body = body[n:]
	// The capacity is capped so a caller that appends to Payload reallocates instead of
	// writing into the bytes after it.
	return Record{Seq: seq, At: int64(at), Payload: body[:plen:plen]}, nil
}

// frameHeader reads a frame's length and its first two body varints without checking the
// CRC, for frames already verified: the writer indexing the buffer it just made durable.
func frameHeader(data []byte) (seq uint64, at int64, total int, ok bool) {
	bodyLen, n := binary.Uvarint(data)
	if n <= 0 {
		return 0, 0, 0, false
	}
	total = n + int(bodyLen) + 4
	if len(data) < total {
		return 0, 0, 0, false
	}
	body := data[n:]
	seq, m := binary.Uvarint(body)
	if m <= 0 {
		return 0, 0, 0, false
	}
	a, k := binary.Uvarint(body[m:])
	if k <= 0 {
		return 0, 0, 0, false
	}
	return seq, int64(a), total, true
}
