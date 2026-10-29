package main

import (
	"strings"
	"testing"
)

// Library errors already begin with "quill: "; the command line must not print the prefix twice.
func TestErrorsCarryOneQuillPrefix(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	notLog := t.TempDir()
	for _, args := range [][]string{
		{"commit", "-dir", dir, "-name", "c", "-offset", "99"},
		{"stat", "-dir", notLog},
		{"cat", "-dir", notLog},
		{"register", "-dir", dir, "-name", "Bad Name"},
	} {
		code, _, stderr := exec(t, "", args...)
		if code != exitError {
			t.Errorf("quill %v: exit %d, want %d (stderr %q)", args, code, exitError, stderr)
		}
		if !strings.HasPrefix(stderr, "quill: ") || strings.Contains(stderr, "quill: quill: ") {
			t.Errorf("quill %v: stderr %q, want exactly one %q prefix", args, stderr, "quill: ")
		}
	}
}

// A commit that fails must not leave behind the consumer it would have created.
func TestFailedCommitLeavesNoNewConsumer(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	if code, _, stderr := exec(t, "", "commit", "-dir", dir, "-name", "nope", "-offset", "99"); code != exitError {
		t.Fatalf("exit %d, want %d: %s", code, exitError, stderr)
	}
	if out := mustRun(t, "", "consumers", "-dir", dir); strings.Contains(out, "nope") {
		t.Errorf("a failed commit registered the consumer:\n%s", out)
	}
}

// A failed commit of an existing consumer leaves it as it was.
func TestFailedCommitKeepsExistingConsumer(t *testing.T) {
	dir := newLog(t, 5, func(i int) []byte { return []byte("record") })
	mustRun(t, "", "commit", "-dir", dir, "-name", "keep", "-offset", "3")
	if code, _, _ := exec(t, "", "commit", "-dir", dir, "-name", "keep", "-offset", "99"); code != exitError {
		t.Fatalf("exit %d, want %d", code, exitError)
	}
	if out := mustRun(t, "", "consumers", "-dir", dir); !strings.Contains(out, "keep") {
		t.Errorf("the existing consumer is gone:\n%s", out)
	}
}
