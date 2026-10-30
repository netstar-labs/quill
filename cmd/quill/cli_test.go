package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// exec runs the command in-process.
func exec(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errw bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errw)
	return code, out.String(), errw.String()
}

func mustRun(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	code, out, errw := exec(t, stdin, args...)
	if code != exitOK {
		t.Fatalf("quill %v: exit %d: %s%s", args, code, out, errw)
	}
	return out
}

// buildLog opens a log under a fresh temporary directory with cfg (Dir is set here), appends n
// records with payload(i), closes it and returns the directory.
func buildLog(t *testing.T, cfg quill.Config, n int, payload func(i int) []byte) string {
	t.Helper()
	cfg.Dir = filepath.Join(t.TempDir(), "orders")
	l, err := quill.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if _, err := l.Append(payload(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return cfg.Dir
}

// newLog writes n records with a library writer and closes it.
func newLog(t *testing.T, n int, payload func(i int) []byte) string {
	t.Helper()
	return buildLog(t, quill.Config{Retention: -1, SegmentBytes: 4096, FlushInterval: time.Hour}, n, payload)
}

func TestEveryDocumentedVerbExistsAndTheReferenceIsCurrent(t *testing.T) {
	for _, name := range []string{"ls", "stat", "read", "tail", "cat", "append", "consumers", "commit", "register", "unregister", "check", "prune", "roll", "archive", "recover"} {
		if findVerb(name) == nil {
			t.Errorf("documented verb %q does not exist", name)
		}
	}
	var out bytes.Buffer
	if code := run([]string{"reference"}, nil, &out, os.Stderr); code != exitOK {
		t.Fatal("reference failed")
	}
	want, err := os.ReadFile("../../docs/reference.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatal("docs/reference.md is stale: run `go run ./cmd/quill reference > docs/reference.md`")
	}
	// Every verb's lock class is one of the three, and the matrix is decision 53's.
	want53 := map[string]string{"append": "log", "roll": "log", "prune": "log", "recover": "log",
		"commit": "consumer", "register": "consumer", "unregister": "consumer", "archive": "consumer",
		"cat": "none", "read": "none", "tail": "none", "stat": "none", "ls": "none", "consumers": "none", "check": "none"}
	for name, lock := range want53 {
		if v := findVerb(name); v.lock != lock {
			t.Errorf("%s takes %q, decision 53 says %q", name, v.lock, lock)
		}
	}
}

func TestUsageErrorsExitTwoAndHelpWorks(t *testing.T) {
	if code, _, _ := exec(t, ""); code != exitUsage {
		t.Fatalf("no verb: %d", code)
	}
	if code, _, _ := exec(t, "", "frobnicate"); code != exitUsage {
		t.Fatalf("unknown verb: %d", code)
	}
	if code, _, _ := exec(t, "", "stat"); code != exitUsage {
		t.Fatalf("missing -dir: %d", code)
	}
	if code, _, _ := exec(t, "", "read", "-dir", "x", "-bogus"); code != exitUsage {
		t.Fatalf("bad flag: %d", code)
	}
	out := mustRun(t, "", "help")
	if !strings.Contains(out, "archive") || !strings.Contains(out, "Exit codes") {
		t.Fatalf("%s", out)
	}
	out = mustRun(t, "", "help", "tail")
	if !strings.Contains(out, "-follow") {
		t.Fatalf("%s", out)
	}
	if code, _, _ := exec(t, "", "stat", "-dir", filepath.Join(t.TempDir(), "missing")); code != exitError {
		t.Fatalf("missing log: %d", code)
	}
}

func TestAnUnopenableDirExitsOneAndUsageErrorsStillWinExitTwo(t *testing.T) {
	// -dir names a path under a regular file, so no verb can open it.
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(file, "log")
	for _, args := range [][]string{
		{"stat"}, {"check"}, {"read"}, {"cat"}, {"tail"}, {"consumers"}, // reader verbs
		{"register", "-name", "c"}, {"unregister", "-name", "c"}, {"commit", "-name", "c", "-offset", "1"}, // consumer verbs
		{"append", "x"}, {"roll"}, {"prune", "-retention", "-1s"}, // writer verbs
	} {
		if code, _, errw := exec(t, "", append([]string{args[0], "-dir", bad}, args[1:]...)...); code != exitError {
			t.Errorf("%v on an unopenable -dir: exit %d, want %d (%s)", args, code, exitError, errw)
		}
	}
	for _, args := range [][]string{
		{"commit", "-name", "c"},                                // neither -offset nor -time
		{"commit", "-name", "c", "-offset", "1", "-time", "1h"}, // both
		{"register", "-name", "c", "-protect", "-no-protect"},
		{"unregister"}, // no -name
		{"prune", "-retention", "bogus"},
	} {
		if code, _, errw := exec(t, "", append([]string{args[0], "-dir", bad}, args[1:]...)...); code != exitUsage {
			t.Errorf("%v on an unopenable -dir: exit %d, want %d (%s)", args, code, exitUsage, errw)
		}
	}
}

func TestReadableOutputIsLosslessInTextAndJSON(t *testing.T) {
	// T20.
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	payloads := [][]byte{
		all,
		[]byte("\xff\xfe invalid utf-8"),
		[]byte("nul\x00inside"),
		[]byte("two\nlines\r\n"),
		[]byte(`quotes " and \ backslash`),
		bytes.Repeat([]byte("m"), 1<<20),
		{},
		[]byte("escape \x1b[31mred\x1b[0m and bell \a"),
		[]byte("plain text"),
	}
	dir := newLog(t, len(payloads), func(i int) []byte { return payloads[i] })
	for _, mode := range []string{"", "-b64"} {
		args := []string{"cat", "-dir", dir, "-json"}
		if mode != "" {
			args = append(args, mode)
		}
		out := mustRun(t, "", args...)
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) != len(payloads) {
			t.Fatalf("%s: %d lines", mode, len(lines))
		}
		for i, line := range lines {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				t.Fatalf("%s: line %d is not JSON: %v", mode, i, err)
			}
			_, hasText := obj["payload"]
			_, hasB64 := obj["payload_b64"]
			if hasText == hasB64 {
				t.Fatalf("%s: line %d has payload=%v payload_b64=%v", mode, i, hasText, hasB64)
			}
			if mode == "-b64" && !hasB64 {
				t.Fatalf("-b64: line %d has a text payload", i)
			}
			var rec struct {
				Seq        uint64 `json:"seq"`
				At         int64  `json:"at"`
				Time       string `json:"time"`
				Len        int    `json:"len"`
				Payload    string `json:"payload"`
				PayloadB64 string `json:"payload_b64"`
			}
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				t.Fatal(err)
			}
			got := []byte(rec.Payload)
			if hasB64 {
				var err error
				if got, err = base64.StdEncoding.DecodeString(rec.PayloadB64); err != nil {
					t.Fatal(err)
				}
			}
			if rec.Seq != uint64(i+1) || !bytes.Equal(got, payloads[i]) || rec.Len != len(payloads[i]) {
				t.Fatalf("%s: line %d: seq %d len %d, payload differs: %v", mode, i, rec.Seq, rec.Len, !bytes.Equal(got, payloads[i]))
			}
			if _, err := time.Parse(time.RFC3339, rec.Time); err != nil {
				t.Fatal(err)
			}
			// The fixed field order.
			if !strings.HasPrefix(line, `{"seq":`) || strings.Index(line, `"at":`) > strings.Index(line, `"time":`) || strings.Index(line, `"time":`) > strings.Index(line, `"len":`) {
				t.Fatalf("field order: %s", line[:60])
			}
			if strings.ContainsAny(line, "\x1b\x07\x00") {
				t.Fatalf("a raw control character reached the output: %q", line[:60])
			}
		}
	}
	// The text form escapes too.
	out := mustRun(t, "", "cat", "-dir", dir)
	if strings.ContainsAny(out, "\x1b\x07\x00") {
		t.Fatal("a raw control character reached the text output")
	}
	if !strings.Contains(out, `\x1b[31mred`) && !strings.Contains(out, `\u001b[31mred`) {
		t.Fatalf("escape sequence not escaped: %q", out)
	}
	if strings.Count(out, "\n") != len(payloads) {
		t.Fatalf("%d lines", strings.Count(out, "\n"))
	}
	// -from, -limit, read and tail.
	out = mustRun(t, "", "read", "-dir", dir, "-from", "3", "-limit", "2")
	if !strings.HasPrefix(out, "3\t") || strings.Count(out, "\n") != 2 {
		t.Fatalf("%q", out)
	}
	out = mustRun(t, "", "tail", "-dir", dir, "-n", "2")
	if !strings.HasPrefix(out, "8\t") || strings.Count(out, "\n") != 2 {
		t.Fatalf("%q", out)
	}
	// -n larger than the head starts at the first record; -n 0 shows none.
	out = mustRun(t, "", "tail", "-dir", dir, "-n", "50")
	if !strings.HasPrefix(out, "1\t") || strings.Count(out, "\n") != len(payloads) {
		t.Fatalf("%q", out)
	}
	if out = mustRun(t, "", "tail", "-dir", dir, "-n", "0"); out != "" {
		t.Fatalf("%q", out)
	}
}

func TestTheVerbsRunBesideALiveWriterAndShowNothingBeyondTheMarker(t *testing.T) {
	// T23 (the CLI lock matrix) and the second half of T20.
	dir := filepath.Join(t.TempDir(), "orders")
	l, err := quill.Open(quill.Config{Dir: dir, Retention: -1, SegmentBytes: 256, FlushInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	for i := 0; i < 20; i++ {
		if _, err := l.Append([]byte(fmt.Sprintf("record %d", i+1))); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Flush(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		l.Append([]byte("unflushed"))
	}
	arch := filepath.Join(t.TempDir(), "arch")
	ok := map[string][]string{
		"stat":       {"stat", "-dir", dir},
		"cat":        {"cat", "-dir", dir, "-json"},
		"read":       {"read", "-dir", dir, "-from", "19"},
		"tail":       {"tail", "-dir", dir, "-n", "3"},
		"consumers":  {"consumers", "-dir", dir},
		"check":      {"check", "-dir", dir},
		"ls":         {"ls", "-dir", filepath.Dir(dir)},
		"register":   {"register", "-dir", dir, "-name", "billing", "-protect"},
		"commit":     {"commit", "-dir", dir, "-name", "billing", "-offset", "5"},
		"archive":    {"archive", "-dir", dir, "-to", arch},
		"unregister": {"unregister", "-dir", dir, "-name", "billing"},
	}
	for _, name := range []string{"stat", "cat", "read", "tail", "consumers", "check", "ls", "register", "commit", "archive", "unregister"} {
		out := mustRun(t, "", ok[name]...)
		if strings.Contains(out, "unflushed") {
			t.Fatalf("%s showed an unflushed record", name)
		}
		switch name {
		case "cat":
			if strings.Count(out, "\n") != 20 {
				t.Fatalf("cat: %d lines", strings.Count(out, "\n"))
			}
		case "stat":
			if !strings.Contains(out, "head 20\n") {
				t.Fatalf("stat: %s", out)
			}
		case "register":
			if !strings.Contains(out, "protected=true") {
				t.Fatalf("register: %s", out)
			}
		case "check":
			if !strings.Contains(out, "HEALTHY") || !strings.Contains(out, "UNBOUNDED") {
				t.Fatalf("check: %s", out)
			}
		case "ls":
			if !strings.Contains(out, "orders") || !strings.Contains(out, "UNBOUNDED") {
				t.Fatalf("ls: %s", out)
			}
		}
	}
	for _, args := range [][]string{
		{"append", "-dir", dir, "x"},
		{"roll", "-dir", dir},
		{"prune", "-dir", dir, "-force"},
		{"recover", "-dir", dir},
	} {
		code, _, errw := exec(t, "", args...)
		if code != exitError || !strings.Contains(errw, "locked") {
			t.Fatalf("%s beside a live writer: exit %d %q", args[0], code, errw)
		}
	}
	// With the writer gone, they work.
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	out := mustRun(t, "", "append", "-dir", dir, "a", "b")
	if strings.TrimSpace(out) != "27" { // 20 flushed + 5 flushed at close + 2
		t.Fatalf("append: %q", out)
	}
	out = mustRun(t, "line one\nline two\n", "append", "-dir", dir)
	if strings.TrimSpace(out) != "29" {
		t.Fatalf("append stdin: %q", out)
	}
	mustRun(t, "", "roll", "-dir", dir)
	out = mustRun(t, "", "prune", "-dir", dir, "-force")
	if !strings.Contains(out, "segments") {
		t.Fatalf("prune: %s", out)
	}
	out = mustRun(t, "", "recover", "-dir", dir)
	if !strings.Contains(out, "head 29") {
		t.Fatalf("recover: %s", out)
	}
	// prune needs a policy or explicit flags.
	if err := os.Remove(filepath.Join(dir, "quill.policy")); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := exec(t, "", "prune", "-dir", dir); code != exitError {
		t.Fatalf("prune without a policy: %d", code)
	}
	mustRun(t, "", "prune", "-dir", dir, "-retention", "1h")
}

// A segment file over the size limit is refused unread, and check says so instead of
// "damaged at byte 0, intact to byte 0" (Stage 2 audit A1-25).
func TestCheckReportsAnOversizeSegmentFileAsRefusedNotDamaged(t *testing.T) {
	dir := newLog(t, 3, func(i int) []byte { return []byte("r") })
	if err := os.Truncate(filepath.Join(dir, "0000000000.log"), 1700<<20); err != nil { // sparse
		t.Fatal(err)
	}
	code, out, _ := exec(t, "", "check", "-dir", dir)
	if code != exitUnhealthy || !strings.Contains(out, "refused without being read") || !strings.Contains(out, "over the 1610612736 limit") || strings.Contains(out, "damaged at byte") {
		t.Fatalf("exit %d: %s", code, out)
	}
	code, out, _ = exec(t, "", "check", "-dir", dir, "-json")
	if code != exitUnhealthy || !strings.Contains(out, "over the 1610612736 limit") || strings.Contains(out, "damaged") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestCheckReportsUnhealthyWithExitThreeAndFlagsTheClock(t *testing.T) {
	// T9 and T21, the CLI parts.
	dir := newLog(t, 5, func(i int) []byte { return []byte("r") })
	if err := os.WriteFile(filepath.Join(dir, "consumers", "broken"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := exec(t, "", "check", "-dir", dir)
	if code != exitUnhealthy || !strings.Contains(out, "UNHEALTHY") || !strings.Contains(out, "consumer") {
		t.Fatalf("exit %d: %s", code, out)
	}
	code, out, _ = exec(t, "", "check", "-dir", dir, "-json")
	if code != exitUnhealthy || !strings.Contains(out, `"status":"UNHEALTHY"`) {
		t.Fatalf("exit %d: %s", code, out)
	}
	// register then unregister leaves the lock file; -protect and -no-protect toggle.
	mustRun(t, "", "register", "-dir", dir, "-name", "b", "-protect")
	out = mustRun(t, "", "register", "-dir", dir, "-name", "b")
	if !strings.Contains(out, "protected=true") {
		t.Fatalf("default keeps the flag: %s", out)
	}
	out = mustRun(t, "", "register", "-dir", dir, "-name", "b", "-no-protect")
	if !strings.Contains(out, "protected=false") {
		t.Fatalf("%s", out)
	}
	if code, _, _ := exec(t, "", "register", "-dir", dir, "-name", "b", "-protect", "-no-protect"); code != exitUsage {
		t.Fatalf("%d", code)
	}
	mustRun(t, "", "unregister", "-dir", dir, "-name", "b")
	if _, err := os.Stat(filepath.Join(dir, "consumers", "b.lock")); err != nil {
		t.Fatal("lock file deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, "consumers", "b")); err == nil {
		t.Fatal("offset file remains")
	}
	// commit -time and -offset; exactly one of them.
	mustRun(t, "", "register", "-dir", dir, "-name", "c")
	out = mustRun(t, "", "commit", "-dir", dir, "-name", "c", "-offset", "4")
	if strings.TrimSpace(out) != "c 4" {
		t.Fatalf("%q", out)
	}
	out = mustRun(t, "", "commit", "-dir", dir, "-name", "c", "-time", "@0")
	if strings.TrimSpace(out) != "c 1" {
		t.Fatalf("%q", out)
	}
	out = mustRun(t, "", "commit", "-dir", dir, "-name", "c", "-time", time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	if strings.TrimSpace(out) != "c 6" { // after every record: head + 1
		t.Fatalf("%q", out)
	}
	out = mustRun(t, "", "commit", "-dir", dir, "-name", "c", "-time", "1h")
	if strings.TrimSpace(out) != "c 1" { // an hour ago: before every record
		t.Fatalf("%q", out)
	}
	if code, _, _ := exec(t, "", "commit", "-dir", dir, "-name", "c"); code != exitUsage {
		t.Fatalf("%d", code)
	}
	if code, _, _ := exec(t, "", "commit", "-dir", dir, "-name", "c", "-offset", "99"); code != exitError {
		t.Fatalf("%d", code)
	}
	// -since on read.
	out = mustRun(t, "", "read", "-dir", dir, "-since", "@0", "-limit", "1")
	if !strings.HasPrefix(out, "1\t") {
		t.Fatalf("%q", out)
	}
	if _, err := parseTime("2026-10-28T14:00:00Z", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := parseTime("yesterday", time.Now()); err == nil {
		t.Fatal("accepted nonsense")
	}
}

func TestHelpDescribesDirAsTheParentOnlyForLs(t *testing.T) {
	for _, v := range verbs {
		if v.name == "reference" || v.name == "version" {
			continue
		}
		var out bytes.Buffer
		if code := run([]string{"help", v.name}, nil, &out, os.Stderr); code != exitOK {
			t.Fatalf("help %s: exit %d", v.name, code)
		}
		want := "the log directory"
		if v.name == "ls" {
			want = "the parent directory holding the logs"
		}
		if !strings.Contains(out.String(), want) {
			t.Errorf("help %s does not describe -dir as %q:\n%s", v.name, want, out.String())
		}
		if v.name == "ls" && strings.Contains(out.String(), "the log directory") {
			t.Errorf("help ls still calls -dir the log directory:\n%s", out.String())
		}
	}
}

// lockedBuffer is a writer one goroutine writes and another reads.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func TestTailFollowPrintsAtOnceNotAtExit(t *testing.T) {
	// A follow that has printed a record and is now waiting must have delivered it to the
	// writer: output to a pipe or file was held in a 4 KiB buffer until interrupt (A1-42).
	dir := newLog(t, 1, func(i int) []byte { return []byte("first") })
	var out, errw lockedBuffer
	done := make(chan int, 1)
	go func() { done <- run([]string{"tail", "-dir", dir, "-follow"}, strings.NewReader(""), &out, &errw) }()
	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(out.String(), "first") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	got := out.String()
	// Stop the follow (the command stops on SIGINT) whether or not the record arrived.
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	if code := <-done; code != exitOK {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	if !strings.Contains(got, "first") {
		t.Fatalf("the record was not written while the follow was waiting; before the interrupt: %q, after: %q", got, out.String())
	}
}
