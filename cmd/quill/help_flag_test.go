package main

import "testing"

// `quill -h` and `quill --help` before any verb print the same usage as `quill help` and exit 0
// (decision 53, audit L5-1). A bare `quill` still prints the usage and exits 2, and `--version`
// stays a usage error: only the help flag was approved.
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
	for _, args := range [][]string{{"--version"}, {"-bogus"}} {
		if code, out, errw := exec(t, "", args...); code != 2 || out != "" || errw == "" {
			t.Errorf("quill %v: exit %d, stdout %q, stderr %q; want a usage error on stderr, exit 2", args, code, out, errw)
		}
	}
}
