package quill

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

// maxSegmentID is the largest segment id the ten-digit names can hold.
const maxSegmentID = 9_999_999_999

// parseSegmentName returns the id of a segment file name, or false for any other name.
func parseSegmentName(name string) (uint64, bool) {
	digits, ok := strings.CutSuffix(name, ".log")
	if !ok || len(digits) != 10 {
		return 0, false
	}
	id, err := strconv.ParseUint(digits, 10, 64)
	return id, err == nil
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

// appendTail appends quill.tail's bytes to dst: the same bytes json.Marshal gives the tail
// struct (fixed field order: seq, seg, off, maxseg) followed by a newline, without the
// reflection and the copies.
func appendTail(dst []byte, t tail) []byte {
	dst = append(dst, `{"seq":`...)
	dst = strconv.AppendUint(dst, t.Seq, 10)
	dst = append(dst, `,"seg":`...)
	dst = strconv.AppendUint(dst, t.Seg, 10)
	dst = append(dst, `,"off":`...)
	dst = strconv.AppendUint(dst, t.Off, 10)
	dst = append(dst, `,"maxseg":`...)
	dst = strconv.AppendUint(dst, t.MaxSeg, 10)
	return append(dst, '}', '\n')
}

// encodeTail is appendTail into a new slice, for the callers that write the marker once.
func encodeTail(t tail) []byte { return appendTail(nil, t) }

// decodeTail parses quill.tail strictly: exactly the four fields, every one present, seg at
// most maxseg, maxseg at most maxSegmentID (the ten-digit names hold no more) and off at
// most maxSegmentFileBytes (no segment is longer, and a reader would
// otherwise carry a value beyond an int64 into a slice bound). The bytes encodeTail writes take decodeTailFast; every other input,
// valid or not, takes decodeTailSlow, so its verdict and message are the strict decoder's
// (decision 105).
func decodeTail(data []byte) (tail, error) {
	if t, ok := decodeTailFast(data); ok {
		return t, nil
	}
	return decodeTailSlow(data)
}

// tailDigits reads a plain decimal number of one to 19 digits with no leading zero (a lone
// "0" is allowed) from the front of b: 19 digits always fit a uint64.
func tailDigits(b []byte) (v uint64, n int, ok bool) {
	for n < len(b) && b[n] >= '0' && b[n] <= '9' {
		if n == 19 {
			return 0, 0, false
		}
		v = v*10 + uint64(b[n]-'0')
		n++
	}
	if n == 0 || (b[0] == '0' && n > 1) {
		return 0, 0, false
	}
	return v, n, true
}

// decodeTailFast accepts only the exact bytes appendTail writes, with seg at most maxseg and
// off at most maxSegmentFileBytes,
// and reports ok=false for anything else without saying why. It allocates nothing.
func decodeTailFast(data []byte) (t tail, ok bool) {
	rest := data
	for _, f := range [...]struct {
		key string
		dst *uint64
	}{{`{"seq":`, &t.Seq}, {`,"seg":`, &t.Seg}, {`,"off":`, &t.Off}, {`,"maxseg":`, &t.MaxSeg}} {
		if len(rest) < len(f.key) || string(rest[:len(f.key)]) != f.key {
			return tail{}, false
		}
		v, n, ok := tailDigits(rest[len(f.key):])
		if !ok {
			return tail{}, false
		}
		*f.dst = v
		rest = rest[len(f.key)+n:]
	}
	if string(rest) != "}\n" || t.Seg > t.MaxSeg || t.MaxSeg > maxSegmentID || t.Off > maxSegmentFileBytes {
		return tail{}, false
	}
	return t, true
}

// decodeTailSlow is the strict JSON path: it gives the verdict and the error for every input
// decodeTailFast does not accept.
func decodeTailSlow(data []byte) (tail, error) {
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
	if out.MaxSeg > maxSegmentID {
		return tail{}, fmt.Errorf("%w: %s: maxseg %d above the largest segment id, %d", errBadState, tailName, out.MaxSeg, uint64(maxSegmentID))
	}
	if out.Off > maxSegmentFileBytes {
		return tail{}, fmt.Errorf("%w: %s: off %d above the largest segment, %d bytes", errBadState, tailName, out.Off, maxSegmentFileBytes)
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

// loadFormat reads and parses the quill.format at path, the one place the rule is written: a
// marker newer than this build is ErrFutureVersion, any other damage is ErrCorrupt (decision 6).
// A read failure, fs.ErrNotExist included, is returned as readState gives it, for the caller to
// classify; an oversize file or one that is not a regular file is already ErrCorrupt there.
func loadFormat(fsys fsx.FS, path string) (logID, error) {
	data, err := readState(fsys, path)
	if err != nil {
		return "", err
	}
	id, err := decodeFormat(data)
	if err != nil {
		if errors.Is(err, ErrFutureVersion) {
			return "", err
		}
		return "", fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	return id, nil
}

// checkLogID re-reads quill.format in dir and returns ErrWrongLog when it names a log other than
// id: the directory was replaced by another log while a handle opened on id was live. A missing
// marker is ErrNoLog; any other failure is returned as loadFormat gives it.
func checkLogID(fsys fsx.FS, dir string, id logID) error {
	have, err := loadFormat(fsys, filepath.Join(dir, formatName))
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s has no %s", ErrNoLog, dir, formatName)
	}
	if err != nil {
		return err
	}
	if have != id {
		return fmt.Errorf("%w: %s now holds log %s, this handle was opened on %s", ErrWrongLog, dir, have, id)
	}
	return nil
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
	if len(s) > 1 && s[0] == '0' {
		return 0, strconv.ErrSyntax
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
// returns ok == false with the zero policy. It never fails Open (decision 48). A policy
// with no retention (`{}`, null, only unknown fields) is a failure too: every writer writes
// it, an off one as a negative duration ("-1ns" for Retention: -1), and a missing one would
// be acted on as 0, the 24 h default.
func decodePolicy(data []byte) (policy, bool) {
	var p policy
	if err := json.Unmarshal(bytes.TrimSpace(data), &p); err != nil || p.Retention == "" {
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

// readState reads a whole state file through the seam, without following a symbolic link. A
// state file that is not a regular file (a symbolic link included) or is over maxStateBytes is
// damage: the error is both ErrCorrupt and errBadState, as one that does not parse is.
func readState(fsys fsx.FS, path string) ([]byte, error) {
	f, err := fsys.OpenFile(path, fsx.O_RDONLY, 0)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) { // O_NOFOLLOW met a link, a dangling one too
			return nil, fmt.Errorf("%w: %w: %s: a symbolic link", ErrCorrupt, errBadState, path)
		}
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %w: %s: not a regular file", ErrCorrupt, errBadState, path)
	}
	if info.Size() > maxStateBytes {
		return nil, fmt.Errorf("%w: %w: %s: %d bytes", ErrCorrupt, errBadState, path, info.Size())
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

// setStateMode makes the existing state file at path have exactly perm, in place: an fchmod
// of the file as it is, which needs no free block, where writeState needs one for its temp
// file. It does nothing when the mode is already right. Only the file's owner can do it.
func setStateMode(fsys fsx.FS, path string, perm fs.FileMode) error {
	f, err := fsys.OpenFile(path, fsx.O_RDONLY, 0) // never follows a link at the final component
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Mode().Perm() == perm {
		return nil
	}
	return f.Chmod(perm)
}

// ReadPolicy reads quill.policy, the settings the writer last opened the log with, as a
// Config with only those fields set (Retention, SegmentBytes, SegmentMaxAge, MaxBytes,
// MinFreeBytes, MaxRecords, MaxPayloadBytes, ConsumerTTL, ProtectedConsumers). ok is false
// when the file is missing or cannot be parsed, which includes a policy with no retention;
// an offline tool then refuses to act on it (decision 68).
func ReadPolicy(dir string) (cfg Config, ok bool) {
	data, err := readState(fsx.OS{}, filepath.Join(dir, policyName))
	if err != nil {
		return Config{}, false
	}
	p, ok := decodePolicy(data)
	if !ok {
		return Config{}, false
	}
	return p.config(), true
}

// config converts a policy decodePolicy accepted to the Config fields it carries. Its
// durations parsed in decodePolicy, so the parse errors dropped here cannot happen.
func (p policy) config() Config {
	dur := func(s string) time.Duration {
		d, _ := time.ParseDuration(s)
		return d
	}
	return Config{Retention: dur(p.Retention), SegmentBytes: p.SegmentBytes, SegmentMaxAge: dur(p.SegmentMaxAge),
		MaxBytes: p.MaxBytes, MinFreeBytes: p.MinFreeBytes, MaxRecords: p.MaxRecords, MaxPayloadBytes: int(p.MaxPayloadBytes),
		ConsumerTTL: dur(p.ConsumerTTL), ProtectedConsumers: p.ProtectedConsumers}
}
