package main

import (
	"strings"
	"testing"
)

// `quill -h` and `quill --help` before any verb print the same usage as `quill help` and exit 0
// (decision 53, audit L5-1). A bare `quill` still prints the usage and exits 2.
func TestGlobalHelpFlagPrintsTheUsageOfTheHelpVerbAndExitsZero(t *testing.T) {
	code, want, werr := exec(t, "", "help")
	if code != 0 || want == "" || werr != "" {
		t.Fatalf("quill help: exit %d, stdout %q, stderr %q", code, want, werr)
	}
	for _, flag := range []string{"-h", "--help", "-help"} {
		code, out, errw := exec(t, "", flag)
		if code != 0 {
			t.Errorf("quill %s: exit %d, want 0 (stderr %q)", flag, code, errw)
		}
		if out != want {
			t.Errorf("quill %s: stdout %q, want the output of quill help %q", flag, out, want)
		}
		if errw != "" {
			t.Errorf("quill %s: stderr %q, want empty", flag, errw)
		}
	}
}

func TestBareQuillAndOtherGlobalFlagErrorsStayUsageErrors(t *testing.T) {
	code, out, _ := exec(t, "")
	if code != 2 || out == "" {
		t.Errorf("bare quill: exit %d, stdout %q; want the usage and exit 2", code, out)
	}
	for _, args := range [][]string{{"-bogus"}, {"-dir"}} {
		if code, out, errw := exec(t, "", args...); code != 2 || out != "" || errw == "" {
			t.Errorf("quill %v: exit %d, stdout %q, stderr %q; want a usage error on stderr, exit 2", args, code, out, errw)
		}
	}
}

// `quill --version` and `quill -version` print exactly what `quill version` prints, on the same
// stream, and exit 0, stamped or not (decision 53, audit L5-5). `-dir` may come with it and need
// not name a directory; a version flag after a verb stays that verb's unknown-flag usage error.
func TestGlobalVersionFlagIsTheVersionVerb(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })
	for _, stamp := range []string{"dev", "v0.2.0"} {
		version = stamp
		code, want, werr := exec(t, "", "version")
		if code != 0 || want != "quill "+stamp+"\n" || werr != "" {
			t.Fatalf("quill version: exit %d, stdout %q, stderr %q", code, want, werr)
		}
		for _, args := range [][]string{{"--version"}, {"-version"}, {"-dir", "/nonexistent/x", "--version"}, {"--version", "-dir", "/nonexistent/x"}} {
			code, out, errw := exec(t, "", args...)
			if code != 0 || out != want || errw != "" {
				t.Errorf("quill %v (stamp %s): exit %d, stdout %q, stderr %q; want exit 0, stdout %q, empty stderr", args, stamp, code, out, errw, want)
			}
		}
	}
	for _, args := range [][]string{{"read", "-version"}, {"read", "--version"}, {"version", "-version"}, {"-dir", t.TempDir(), "stat", "--version"}} {
		code, out, errw := exec(t, "", args...)
		if code != 2 || out != "" || !strings.Contains(errw, "flag provided but not defined: -version") {
			t.Errorf("quill %v: exit %d, stdout %q, stderr %q; want the verb's unknown-flag usage error, exit 2", args, code, out, errw)
		}
	}
}
