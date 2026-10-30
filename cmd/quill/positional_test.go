package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// A verb that takes no positional arguments refuses one with a usage error: flag parsing stops
// at the first positional, so `read 100` or `tail 5 -json` would otherwise run with the flags
// that follow it ignored and exit 0 (A1-47).
func TestExtraPositionalArgumentsAreAUsageError(t *testing.T) {
	dir := newLog(t, 13, func(i int) []byte { return []byte{'r', byte('a' + i)} })
	parent := filepath.Dir(dir)
	for _, args := range [][]string{
		{"-dir", dir, "read", "10"},
		{"-dir", dir, "read", "-limit", "1", "10"},
		{"-dir", dir, "tail", "5", "-json"},
		{"-dir", dir, "cat", "extra"},
		{"-dir", dir, "stat", "-json", "extra"},
		{"-dir", dir, "consumers", "x"},
		{"-dir", dir, "check", "x"},
		{"-dir", dir, "register", "-name", "a", "x"},
		{"-dir", dir, "roll", "x"},
		{"-dir", dir, "recover", "x"},
		{"-dir", dir, "prune", "x"},
		{"-dir", parent, "ls", "x"},
		{"version", "foo"},
		{"-dir", dir, "read", "-json", "--", "x"},
	} {
		before := snapshot(t, dir)
		code, out, errw := exec(t, "", args...)
		if code != exitUsage {
			t.Errorf("quill %v: exit %d, want %d (stdout %q, stderr %q)", args, code, exitUsage, out, errw)
			continue
		}
		if out != "" || !strings.Contains(errw, "unexpected argument") || !strings.Contains(errw, "flags go before arguments") {
			t.Errorf("quill %v: stdout %q, stderr %q", args, out, errw)
		}
		if got := snapshot(t, dir); len(got) != len(before) {
			t.Errorf("quill %v changed the log directory", args)
		}
	}
}

// append takes its records as arguments and help takes a verb name; both stay accepted.
func TestPositionalArgumentsStayAcceptedWhereTheVerbTakesThem(t *testing.T) {
	dir := newLog(t, 1, func(i int) []byte { return []byte("a") })
	if out := mustRun(t, "", "-dir", dir, "append", "x", "y"); strings.TrimSpace(out) != "3" {
		t.Errorf("append printed %q, want 3", out)
	}
	if out := mustRun(t, "", "help", "read"); !strings.Contains(out, "quill read:") {
		t.Errorf("help read printed %q", out)
	}
}
