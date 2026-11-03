package quill

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A consumer name and a segment file name become path components, so each must be a single
// plain component whatever bytes it came from.

// consumerNameForm is the rule of validName written a second way: 1 to 200 characters of
// [a-z0-9_-], the first not an underscore.
var consumerNameForm = regexp.MustCompile(`^[a-z0-9-][a-z0-9_-]{0,199}$`)

// FuzzValidName: validName accepts exactly the names of consumerNameForm, refuses the rest with
// ErrInvalidName, and an accepted name is one path component (no separator, no dot, no NUL)
// that stays inside the directory it is joined to. The archiver's name is the one addition
// validConsumerName makes.
func FuzzValidName(f *testing.F) {
	for _, s := range []string{"", "a", "orders", "_archive", "_x", "a/b", "..", ".", "a.b", "A", "a b", "a\x00", "-", "é", strings.Repeat("a", 200), strings.Repeat("a", 201)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		err := validName(name)
		if (err == nil) != consumerNameForm.MatchString(name) {
			t.Fatalf("validName(%q) = %v, but the form says accepted=%v", name, err, consumerNameForm.MatchString(name))
		}
		if err != nil {
			if !errors.Is(err, ErrInvalidName) {
				t.Fatalf("validName(%q) = %v, not ErrInvalidName", name, err)
			}
			if name != archiveConsumer && validConsumerName(name) == nil {
				t.Fatalf("validConsumerName accepted %q", name)
			}
			return
		}
		if strings.ContainsAny(name, "/\\.\x00") || filepath.Base(name) != name || filepath.Join("d", name) != "d/"+name {
			t.Fatalf("accepted name %q is not a single plain path component", name)
		}
		if validConsumerName(name) != nil {
			t.Fatalf("validConsumerName refused %q, which validName accepted", name)
		}
	})
}

// FuzzParseSegmentName: an accepted name is exactly segmentName of the id it parses to (ten
// digits and ".log", so no separator and no other dot), and every id the format can hold
// names a file that parses back to it.
func FuzzParseSegmentName(f *testing.F) {
	for _, s := range []string{"0000000000.log", "9999999999.log", "0000000001.log", "000000001.log", "00000000001.log", "0000000000.LOG", "0000000000.log.tmp",
		"+000000001.log", "0x00000001.log", "1_00000000.log", "../00000.log", "0000000000", ".log", ""} {
		f.Add(s, uint64(0))
	}
	f.Add("", uint64(maxSegmentID))
	f.Fuzz(func(t *testing.T, name string, id uint64) {
		got, ok := parseSegmentName(name)
		if ok {
			if got > maxSegmentID || segmentName(got) != name {
				t.Fatalf("parseSegmentName(%q) = %d, which names %q", name, got, segmentName(got))
			}
			if strings.Count(name, ".") != 1 || strings.ContainsAny(name, "/\\\x00") {
				t.Fatalf("accepted segment name %q is not one plain file name", name)
			}
		}
		id %= maxSegmentID + 1
		if back, ok := parseSegmentName(segmentName(id)); !ok || back != id {
			t.Fatalf("segment id %d does not round-trip: %q -> %d, %v", id, segmentName(id), back, ok)
		}
	})
}
