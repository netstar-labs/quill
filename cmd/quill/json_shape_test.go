package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// jsonKind names the JSON type of a raw value: number, string, bool, array, object or null.
func jsonKind(raw json.RawMessage) string {
	switch b := bytes.TrimSpace(raw); {
	case len(b) == 0:
		return "empty"
	case b[0] == '"':
		return "string"
	case b[0] == '[':
		return "array"
	case b[0] == '{':
		return "object"
	case string(b) == "true" || string(b) == "false":
		return "bool"
	case string(b) == "null":
		return "null"
	default:
		return "number"
	}
}

// wantShape fails unless obj has exactly the keys of want, each of the named JSON type. The order of
// keys is not part of the contract (scope.md, Readable output), so only the set is compared.
func wantShape(t *testing.T, what string, obj map[string]json.RawMessage, want map[string]string) {
	t.Helper()
	got := map[string]string{}
	var keys []string
	for k, v := range obj {
		got[k] = jsonKind(v)
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for k, kind := range want {
		if got[k] != kind {
			t.Errorf("%s: key %q is %q, want %q (keys %v)", what, k, got[k], kind, keys)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: undocumented key %q (keys %v)", what, k, keys)
		}
	}
}

func decodeObject(t *testing.T, s string) map[string]json.RawMessage {
	t.Helper()
	var o map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &o); err != nil {
		t.Fatalf("%v: %s", err, s)
	}
	return o
}

// The documented -json shapes of stat, check and consumers (scope.md, Readable output) are what scripts
// parse: pin every key and its JSON type, so a renamed or dropped key fails here (stage 2 audit A6-3).
func TestStatJSONHasExactlyTheDocumentedKeys(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("r") })
	obj := decodeObject(t, mustRun(t, "", "stat", "-dir", dir, "-json"))
	wantShape(t, "stat", obj, map[string]string{
		"head": "number", "first": "number", "segments": "number", "bytes": "number",
		"records": "number", "holes": "array", "marker_age_seconds": "number",
	})
}

func TestCheckJSONHasExactlyTheDocumentedKeys(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("r") })
	top := map[string]string{"status": "string", "unbounded": "bool", "problems": "array"}

	// Healthy: problems is [], never null.
	code, out, _ := exec(t, "", "check", "-dir", dir, "-json")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, out)
	}
	obj := decodeObject(t, out)
	wantShape(t, "check", obj, top)
	if string(obj["problems"]) != "[]" || string(obj["status"]) != `"HEALTHY"` {
		t.Fatalf("a healthy log: %s", out)
	}

	// A consumer file that cannot be parsed is a problem with every documented key.
	if err := os.WriteFile(filepath.Join(dir, "consumers", "broken"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ = exec(t, "", "check", "-dir", dir, "-json")
	if code != exitUnhealthy {
		t.Fatalf("exit %d: %s", code, out)
	}
	obj = decodeObject(t, out)
	wantShape(t, "check", obj, top)
	var problems []map[string]json.RawMessage
	if err := json.Unmarshal(obj["problems"], &problems); err != nil || len(problems) == 0 {
		t.Fatalf("want at least one problem (%v): %s", err, out)
	}
	for _, p := range problems {
		wantShape(t, "check problem", p, map[string]string{
			"kind": "string", "path": "string", "offset": "number", "intact": "number", "detail": "string", "fatal": "bool",
		})
	}
}

func TestConsumersJSONHasExactlyTheDocumentedKeys(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("r") })
	mustRun(t, "", "register", "-dir", dir, "-name", "plain", "-protect")
	aheadConsumer(t, dir, "ahead", "8")
	if err := os.WriteFile(filepath.Join(dir, "consumers", "broken"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := mustRun(t, "", "consumers", "-dir", dir, "-json")
	base := map[string]string{"name": "string", "offset": "number", "lag": "number", "protected": "bool"}
	with := func(k, kind string) map[string]string {
		m := map[string]string{k: kind}
		for bk, bv := range base {
			m[bk] = bv
		}
		return m
	}
	want := map[string]map[string]string{"plain": base, "ahead": with("ahead", "bool"), "broken": with("error", "string")}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		obj := decodeObject(t, line)
		var name string
		if err := json.Unmarshal(obj["name"], &name); err != nil || want[name] == nil {
			t.Fatalf("unexpected consumer line %q", line)
		}
		seen[name] = true
		wantShape(t, "consumers "+name, obj, want[name])
		if name == "ahead" && string(obj["ahead"]) != "true" {
			t.Errorf("ahead must be true: %s", line)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("saw %v, want %d consumers: %s", seen, len(want), out)
	}
}
