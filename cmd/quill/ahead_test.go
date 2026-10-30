package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aheadConsumer registers a protected consumer and rewrites its file to an offset past the head, the
// state a log restored to an older backup leaves behind.
func aheadConsumer(t *testing.T, dir, name string, next string) string {
	t.Helper()
	mustRun(t, "", "register", "-dir", dir, "-name", name, "-protect")
	path := filepath.Join(dir, "consumers", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Fields(string(b))
	if len(f) != 3 {
		t.Fatalf("consumer file %q", b)
	}
	if err := os.WriteFile(path, []byte(f[0]+" "+next+" "+f[2]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The documented remedy for a consumer ahead of a restored log is `quill commit`: it must reach
// the consumer, rewind it, and keep its protect flag.
func TestCommitRewindsAConsumerAheadOfTheHead(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	path := aheadConsumer(t, dir, "billing", "8")
	if out := mustRun(t, "", "commit", "-dir", dir, "-name", "billing", "-offset", "2"); out != "billing 2\n" {
		t.Fatalf("commit printed %q", out)
	}
	b, _ := os.ReadFile(path)
	if f := strings.Fields(string(b)); len(f) != 3 || f[1] != "2" || f[2] != "1" {
		t.Fatalf("consumer file %q, want offset 2 and the protect flag kept", b)
	}
	// An offset past Head + 1 is still refused, and the file is left as it was.
	aheadConsumer(t, dir, "again", "8")
	if code, _, stderr := exec(t, "", "commit", "-dir", dir, "-name", "again", "-offset", "7"); code != exitError || !strings.Contains(stderr, "outside") {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "consumers", "again")); strings.Fields(string(b))[1] != "8" {
		t.Fatalf("a refused commit changed the file: %q", b)
	}
}

// register opens a consumer as before: an ahead one is still refused there.
func TestRegisterStillRefusesAConsumerAheadOfTheHead(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	aheadConsumer(t, dir, "billing", "8")
	if code, _, stderr := exec(t, "", "register", "-dir", dir, "-name", "billing"); code != exitError || !strings.Contains(stderr, "ahead") {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
}

// consumers and check say so when a consumer is ahead of the head; check stays HEALTHY (the log
// is intact) and exits 0.
func TestConsumersAndCheckFlagAConsumerAheadOfTheHead(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	aheadConsumer(t, dir, "billing", "8")
	if out := mustRun(t, "", "consumers", "-dir", dir); !strings.Contains(out, "ahead of the head") {
		t.Errorf("consumers shows no note:\n%s", out)
	}
	out := mustRun(t, "", "check", "-dir", dir)
	if !strings.HasPrefix(out, "HEALTHY\n") || !strings.Contains(out, "consumer") || !strings.Contains(out, "ahead of the head") {
		t.Errorf("check:\n%s", out)
	}
}

// The archiver's consumer is reserved but not untouchable: after a restore leaves it ahead of the
// head, `quill archive` fails with ErrAhead, and commit rewinds it and unregister removes it.
func TestTheArchiveConsumerCanBeRewoundAndRemoved(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	to := filepath.Join(t.TempDir(), "arch")
	mustRun(t, "", "archive", "-dir", dir, "-to", to)
	path := filepath.Join(dir, "consumers", "_archive")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Fields(string(b))
	if err := os.WriteFile(path, []byte(f[0]+" 99 "+f[2]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := exec(t, "", "archive", "-dir", dir, "-to", to); code != exitError || !strings.Contains(stderr, "ahead") {
		t.Fatalf("archive: exit %d, stderr %q", code, stderr)
	}
	if out := mustRun(t, "", "commit", "-dir", dir, "-name", "_archive", "-offset", "1"); out != "_archive 1\n" {
		t.Fatalf("commit printed %q", out)
	}
	if b, _ := os.ReadFile(path); strings.Fields(string(b))[2] != "1" {
		t.Fatalf("consumer file %q, want the protect flag kept", b)
	}
	mustRun(t, "", "archive", "-dir", dir, "-to", to)
	mustRun(t, "", "unregister", "-dir", dir, "-name", "_archive")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("the archive consumer file remains after unregister")
	}
	// Only that one name is let through: any other leading underscore is still reserved.
	for _, verb := range []string{"register", "unregister"} {
		if code, _, stderr := exec(t, "", verb, "-dir", dir, "-name", "_other"); code != exitError || !strings.Contains(stderr, "underscore") {
			t.Fatalf("%s _other: exit %d, stderr %q", verb, code, stderr)
		}
	}
}
