package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The exit codes are a contract with scripts (`quill check || alert`, decision 53). Every other
// test compares a status with the exitOK, exitError, exitUsage and exitUnhealthy constants, so
// changing a constant's value would leave them all green. This one compares with literals.
func TestExitCodesArePinnedToTheirDocumentedNumbers(t *testing.T) {
	healthy := newLog(t, 3, func(i int) []byte { return []byte("r") })
	damaged := newLog(t, 3, func(i int) []byte { return []byte("r") })
	if err := os.WriteFile(filepath.Join(damaged, "consumers", "broken"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "absent")

	for _, tc := range []struct {
		name string
		want int
		args []string
	}{
		{"healthy check", 0, []string{"check", "-dir", healthy}},
		{"help", 0, []string{"help"}},
		{"version", 0, []string{"version"}},
		{"missing directory", 1, []string{"stat", "-dir", missing}},
		{"unknown flag", 2, []string{"stat", "-dir", healthy, "-bogus"}},
		{"unknown verb", 2, []string{"frobnicate"}},
		{"no verb", 2, nil},
		{"missing -dir", 2, []string{"stat"}},
		{"damaged log", 3, []string{"check", "-dir", damaged}},
		{"damaged log, json", 3, []string{"check", "-dir", damaged, "-json"}},
	} {
		if code, out, errw := exec(t, "", tc.args...); code != tc.want {
			t.Errorf("%s: quill %v exits %d, want %d\n%s%s", tc.name, tc.args, code, tc.want, out, errw)
		}
	}
}

// The prose that tells a script author what the codes mean is a string literal; pin it to the
// same numbers the test above pins the behaviour to.
func TestExitCodeProseMatchesTheNumbers(t *testing.T) {
	const prose = "0 ok, 1 error, 2 usage, 3"
	_, out, _ := exec(t, "", "help")
	if !strings.Contains(out, "Exit codes: "+prose+" check found the log UNHEALTHY.") {
		t.Errorf("quill help does not list the exit codes as %q:\n%s", prose, out)
	}
	_, ref, _ := exec(t, "", "reference")
	if !strings.Contains(ref, "Exit codes: "+prose+" `check` found the log UNHEALTHY.") {
		t.Errorf("quill reference does not list the exit codes as %q", prose)
	}
}
