package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// The exit code of a verb that found the log unhealthy (decision 53) is kept when its output
// cannot be written: only a verb that would have exited 0 is turned into an error, so a script
// that tests for 3 still sees 3.
func TestUnhealthyExitIsKeptWhenStandardOutputFails(t *testing.T) {
	damaged := newLog(t, 3, func(i int) []byte { return []byte("r") })
	if err := os.WriteFile(filepath.Join(damaged, "consumers", "broken"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	var errw bytes.Buffer
	if code := run([]string{"check", "-dir", damaged}, strings.NewReader(""), failWriter{}, &errw); code != exitUnhealthy {
		t.Fatalf("check of a damaged log with a failing stdout: exit %d, want %d; stderr %q", code, exitUnhealthy, errw.String())
	}
}

// A stdin batch whose accounted size (len+34 per line) is exactly MaxBatchBytes is accepted, and
// one byte more is refused, as the library does for the same batch (decision 46).
func TestAppendStdinExactlyAtMaxBatchBytes(t *testing.T) {
	const records = 493447 // 34 accounted bytes each, and 18 payload bytes more reach the limit
	lines := func(extra int) string {
		return strings.Repeat("\n", records-1) + strings.Repeat("x", 18+extra) + "\n"
	}
	if got := (records-1)*34 + 18 + 34; got != quill.DefaultMaxBatchBytes {
		t.Fatalf("setup: the batch accounts %d bytes, want exactly %d", got, quill.DefaultMaxBatchBytes)
	}
	dir := t.TempDir()
	if out := mustRun(t, lines(0), "append", "-dir", dir); strings.TrimSpace(out) != fmt.Sprint(records) {
		t.Fatalf("append of a batch of exactly MaxBatchBytes: %q, want %d", out, records)
	}
	dir = t.TempDir()
	code, out, errw := exec(t, lines(1), "append", "-dir", dir)
	if code != exitError || out != "" || !strings.Contains(errw, "MaxBatchBytes") {
		t.Fatalf("append of a batch one byte over: exit %d, stdout %q, stderr %q", code, out, errw)
	}
}

// The top of each documented number range is accepted: the unix form of a time up to MaxInt64
// seconds (decision 96), a duration of zero ago, a cap of MaxInt64, a consumer TTL that parses.
func TestFlagParsingEdges(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	if got, err := parseTime("@9223372036854775807", now); err != nil || got.Unix() != 1<<63-1 {
		t.Errorf("parseTime(@MaxInt64) = %v, %v; want accepted", got, err)
	}
	if got, err := parseTime("0s", now); err != nil || !got.Equal(now) {
		t.Errorf("parseTime(0s) = %v, %v; want now", got, err)
	}
	if _, err := parseTime("-1s", now); err == nil {
		t.Error("parseTime(-1s) accepted: a duration ago is not negative")
	}

	dir := newLog(t, 5, func(i int) []byte { return []byte("r") })
	for _, args := range [][]string{
		{"read", "-dir", dir, "-since", "0s"},
		{"read", "-dir", dir, "-since", "@9223372036854775807"},
	} {
		if code, out, errw := exec(t, "", args...); code != exitOK {
			t.Errorf("quill %v: exit %d, want %d: %s%s", args, code, exitOK, out, errw)
		}
	}

	policied := buildLog(t, quill.Config{Retention: -1, SegmentBytes: 256, FlushInterval: time.Hour, NoMaintenance: true}, 40,
		func(i int) []byte { return []byte(fmt.Sprintf("record %02d", i)) })
	before := snapshot(t, policied)
	for _, flag := range []string{"-max-bytes", "-max-records"} {
		if code, out, errw := exec(t, "", "prune", "-dir", policied, flag, "9223372036854775807"); code != exitOK {
			t.Errorf("prune %s MaxInt64: exit %d, want %d: %s%s", flag, code, exitOK, out, errw)
		}
		if !reflect.DeepEqual(snapshot(t, policied), before) {
			t.Fatalf("prune %s MaxInt64 changed the log: a cap that high holds everything", flag)
		}
	}

	code, out, _ := exec(t, "", "prune", "-dir", policied, "-consumer-ttl", "tomorrow")
	if code != exitUsage || out != "" {
		t.Errorf("prune -consumer-ttl tomorrow: exit %d, stdout %q; want %d and nothing", code, out, exitUsage)
	}
	if !reflect.DeepEqual(snapshot(t, policied), before) {
		t.Fatal("prune with a bad -consumer-ttl changed the log")
	}
}
