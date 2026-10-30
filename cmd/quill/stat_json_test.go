package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The stat -json shape is what scripts parse (stage 2 audit A1-57): every key is lowercase, and
// holes is always an array of {"from","to"} objects, never null and never capitalised.
func TestStatJSONHolesAreAnAlwaysPresentArrayOfLowercaseObjects(t *testing.T) {
	dir := newLog(t, 150, func(i int) []byte { return bytes.Repeat([]byte{'x'}, 900) })

	// No hole: the key is present and is an empty array, not null.
	out := mustRun(t, "", "stat", "-dir", dir, "-json")
	if !strings.Contains(out, `"holes":[]`) {
		t.Fatalf("a log without holes must print \"holes\":[]: %s", out)
	}

	// Damage the first segment's first frame: the sequences it held are a hole.
	segs, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil || len(segs) < 3 {
		t.Fatalf("want at least 3 segments, got %v (%v)", segs, err)
	}
	b, err := os.ReadFile(segs[0])
	if err != nil {
		t.Fatal(err)
	}
	b[10] ^= 0x01
	if err := os.WriteFile(segs[0], b, 0o600); err != nil {
		t.Fatal(err)
	}
	out = mustRun(t, "", "stat", "-dir", dir, "-json")
	var got struct {
		Holes []map[string]json.RawMessage `json:"holes"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if len(got.Holes) != 1 {
		t.Fatalf("want one hole: %s", out)
	}
	h := got.Holes[0]
	if len(h) != 2 || h["from"] == nil || h["to"] == nil {
		t.Fatalf("a hole must be exactly {\"from\":N,\"to\":N}: %s", out)
	}
	var from, to uint64
	if json.Unmarshal(h["from"], &from) != nil || json.Unmarshal(h["to"], &to) != nil || from == 0 || to < from {
		t.Fatalf("bad hole range: %s", out)
	}
}
