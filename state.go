package quill

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"

	"github.com/netstar-labs/quill/internal/atomicfile"
	"github.com/netstar-labs/quill/internal/fsx"
	"github.com/netstar-labs/quill/internal/strictjson"
)

// The state files (docs/format.md). Each is strict JSON or one strict text line, written by
// temp file, fsync, rename, and refused when malformed, except quill.policy, which is
// informational and parsed tolerantly (decisions 8, 9, 29, 48).

// File names inside a log directory.
const (
	tailName     = "quill.tail"
	formatName   = "quill.format"
	policyName   = "quill.policy"
	lockName     = "quill.lock"
	consumersDir = "consumers"
	tempPrefix   = atomicfile.TempPrefix
)

// segmentName is the file name of segment id: ten decimal digits and ".log".
func segmentName(id uint64) string { return fmt.Sprintf("%010d.log", id) }

// parseSegmentName returns the id of a segment file name, or false for any other name.
func parseSegmentName(name string) (uint64, bool) {
	digits, ok := strings.CutSuffix(name, ".log")
	if !ok || len(digits) != 10 {
		return 0, false
	}
	var id uint64
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		id = id*10 + uint64(c-'0')
	}
	return id, true
}

// formatVersion is the version this build writes and the highest it reads.
const formatVersion = 1

// errBadState marks a state file that does not parse; the file name is in the message.
var errBadState = errors.New("quill: malformed state file")

// tail is quill.tail: the durable position and the highest segment id ever used.
type tail struct {
	Seq    uint64 `json:"seq"`    // the last durable sequence; 0 when none
	Seg    uint64 `json:"seg"`    // the active segment id
	Off    uint64 `json:"off"`    // the durable length of the active segment
	MaxSeg uint64 `json:"maxseg"` // the highest segment id ever created; never reused
}

func encodeTail(t tail) []byte {
	b, _ := json.Marshal(t) // fixed field order: seq, seg, off, maxseg
	return append(b, '\n')
}

// decodeTail parses quill.tail strictly: exactly the four fields, every one present, and
// seg at most maxseg.
func decodeTail(data []byte) (tail, error) {
	var t struct {
		Seq    *uint64 `json:"seq"`
		Seg    *uint64 `json:"seg"`
		Off    *uint64 `json:"off"`
		MaxSeg *uint64 `json:"maxseg"`
	}
	if err := strictjson.Unmarshal(data, &t); err != nil {
		return tail{}, fmt.Errorf("%w: %s: %v", errBadState, tailName, err)
	}
	if t.Seq == nil || t.Seg == nil || t.Off == nil || t.MaxSeg == nil {
		return tail{}, fmt.Errorf("%w: %s: a field is missing", errBadState, tailName)
	}
	out := tail{Seq: *t.Seq, Seg: *t.Seg, Off: *t.Off, MaxSeg: *t.MaxSeg}
	if out.Seg > out.MaxSeg {
		return tail{}, fmt.Errorf("%w: %s: seg %d above maxseg %d", errBadState, tailName, out.Seg, out.MaxSeg)
	}
	return out, nil
}

// logID is the 128-bit identity of a log, 32 lower-case hex digits in quill.format and in
// every consumer file (decision 6).
type logID string

const logIDLen = 32

func newLogID() (logID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return logID(hex.EncodeToString(b[:])), nil
}

func validLogID(s string) bool {
	if len(s) != logIDLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// encodeFormat is the quill.format line: "quill/1 <log id>".
func encodeFormat(id logID) []byte {
	return []byte(fmt.Sprintf("quill/%d %s\n", formatVersion, id))
}

// decodeFormat parses quill.format. A version above formatVersion is ErrFutureVersion; the
// line must be exactly "quill/<n> <32 hex>" and a newline.
func decodeFormat(data []byte) (logID, error) {
	line, ok := strings.CutSuffix(string(data), "\n")
	if !ok {
		return "", fmt.Errorf("%w: %s: no trailing newline", errBadState, formatName)
	}
	rest, ok := strings.CutPrefix(line, "quill/")
	if !ok {
		return "", fmt.Errorf("%w: %s: not a quill marker", errBadState, formatName)
	}
	ver, id, ok := strings.Cut(rest, " ")
	if !ok {
		return "", fmt.Errorf("%w: %s: no log id", errBadState, formatName)
	}
	v, err := parseUint(ver)
	if err != nil {
		return "", fmt.Errorf("%w: %s: bad version %q", errBadState, formatName, ver)
	}
	if v > formatVersion {
		return "", fmt.Errorf("%w: quill/%d, this build reads up to quill/%d", ErrFutureVersion, v, formatVersion)
	}
	if v == 0 {
		return "", fmt.Errorf("%w: %s: version 0", errBadState, formatName)
	}
	if !validLogID(id) {
		return "", fmt.Errorf("%w: %s: bad log id", errBadState, formatName)
	}
	return logID(id), nil
}

// consumerState is one consumer file: "<log id> <next> <protect>\n".
type consumerState struct {
	ID      logID
	Next    uint64 // the next sequence to read
	Protect bool
}

func encodeConsumer(c consumerState) []byte {
	p := "0"
	if c.Protect {
		p = "1"
	}
	return []byte(fmt.Sprintf("%s %d %s\n", c.ID, c.Next, p))
}

func decodeConsumer(data []byte) (consumerState, error) {
	line, ok := strings.CutSuffix(string(data), "\n")
	if !ok {
		return consumerState{}, fmt.Errorf("%w: consumer file: no trailing newline", errBadState)
	}
	fields := strings.Split(line, " ")
	if len(fields) != 3 {
		return consumerState{}, fmt.Errorf("%w: consumer file: want 3 fields, have %d", errBadState, len(fields))
	}
	if !validLogID(fields[0]) {
		return consumerState{}, fmt.Errorf("%w: consumer file: bad log id", errBadState)
	}
	next, err := parseUint(fields[1])
	if err != nil {
		return consumerState{}, fmt.Errorf("%w: consumer file: bad offset %q", errBadState, fields[1])
	}
	var protect bool
	switch fields[2] {
	case "0":
	case "1":
		protect = true
	default:
		return consumerState{}, fmt.Errorf("%w: consumer file: bad protect flag %q", errBadState, fields[2])
	}
	return consumerState{ID: logID(fields[0]), Next: next, Protect: protect}, nil
}

// parseUint is strconv.ParseUint restricted to plain decimal digits: no sign, no leading
// zeros except "0" itself, no underscores.
func parseUint(s string) (uint64, error) {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, strconv.ErrSyntax
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, strconv.ErrSyntax
		}
	}
	return strconv.ParseUint(s, 10, 64)
}

// policy is quill.policy: the settings the writer last opened with, for offline tools and
// quill ls. Informational: Config at open is authoritative and rewrites it. Durations are
// written as Go duration strings; a value that is off is written as its zero or negative value.
type policy struct {
	Retention          string   `json:"retention"`
	SegmentBytes       int64    `json:"segment_bytes"`
	SegmentMaxAge      string   `json:"segment_max_age"`
	MaxBytes           int64    `json:"max_bytes"`
	MinFreeBytes       int64    `json:"min_free_bytes"`
	MaxRecords         int64    `json:"max_records"`
	MaxPayloadBytes    int64    `json:"max_payload_bytes"`
	ConsumerTTL        string   `json:"consumer_ttl"`
	ProtectedConsumers []string `json:"protected_consumers"`
}

func encodePolicy(p policy) []byte {
	b, _ := json.MarshalIndent(p, "", "  ")
	return append(b, '\n')
}

// decodePolicy parses quill.policy tolerantly: unknown fields are ignored, and any failure
// returns ok == false with the zero policy. It never fails Open (decision 48).
func decodePolicy(data []byte) (policy, bool) {
	var p policy
	if err := json.Unmarshal(bytes.TrimSpace(data), &p); err != nil {
		return policy{}, false
	}
	for _, d := range []string{p.Retention, p.SegmentMaxAge, p.ConsumerTTL} {
		if d != "" {
			if _, err := time.ParseDuration(d); err != nil {
				return policy{}, false
			}
		}
	}
	return p, true
}

// maxStateBytes bounds a state file read: the files are a few hundred bytes, and a huge one
// is damage, not a bigger buffer (guarantee 7).
const maxStateBytes = 1 << 20

// readState reads a whole state file through the seam, without following a symbolic link.
func readState(fsys fsx.FS, path string) ([]byte, error) {
	f, err := fsys.OpenFile(path, fsx.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s: not a regular file", errBadState, path)
	}
	if info.Size() > maxStateBytes {
		return nil, fmt.Errorf("%w: %s: %d bytes", errBadState, path, info.Size())
	}
	buf := make([]byte, info.Size())
	n, err := f.ReadAt(buf, 0)
	if err != nil && n < len(buf) {
		return nil, err
	}
	return buf, nil
}

// writeState replaces a state file atomically, with no directory fsync (decision 40).
func writeState(fsys fsx.FS, path string, data []byte, perm fs.FileMode) error {
	return atomicfile.Write(fsys, path, data, perm)
}
