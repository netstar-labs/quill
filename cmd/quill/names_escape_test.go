package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/netstar-labs/quill"
)

// hostileNames are file names an untrusted party could create: a newline and an ESC sequence
// (the A3-3 reproduction), a bidi override, a C1 control and a DEL.
var hostileNames = []string{
	"evil\n\x1b]0;PWNED\x07\x1b[2J",
	"x\x1b[31mRED\x1b[0m",
	"rtl\u202eexe.txt",
	"c1\u009bcsi",
	"del\x7fbyte",
}

// noRawControl fails the test if s holds a byte a terminal acts on (a C0 control other than
// the newline that ends a line, DEL, ESC) or a rune that is not printable (C1 controls, bidi
// controls, U+FFFD from invalid UTF-8), and returns the number of lines.
func noRawControl(t *testing.T, what, s string) int {
	t.Helper()
	if !utf8.ValidString(s) {
		t.Errorf("%s: invalid UTF-8 reaches the terminal: %q", what, s)
	}
	for _, r := range s {
		if r == '\n' || r == '\t' {
			continue
		}
		if !strconv.IsPrint(r) {
			t.Errorf("%s: raw non-printable %U reaches the terminal: %q", what, r, s)
			return 0
		}
	}
	return strings.Count(s, "\n")
}

func TestLsConsumersAndCheckEscapeFileDerivedNames(t *testing.T) {
	for _, name := range hostileNames {
		t.Run(strconv.Quote(name), func(t *testing.T) {
			logDir := newLog(t, 5, func(i int) []byte { return []byte("rec") })
			parent := filepath.Dir(logDir)

			// A consumer file, a stray segment-like file and a log directory, all hostile.
			if err := os.MkdirAll(filepath.Join(logDir, "consumers"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(logDir, "consumers", name), []byte("not a consumer file"), 0o600); err != nil {
				t.Skipf("the file system refuses the name: %v", err)
			}
			if err := os.WriteFile(filepath.Join(logDir, "0000000099"+name+".log"), []byte("junk"), 0o600); err != nil {
				t.Fatal(err)
			}
			evil := filepath.Join(parent, name)
			if err := os.MkdirAll(evil, 0o700); err != nil {
				t.Fatal(err)
			}
			for _, f := range []string{"quill.format", "quill.tail"} {
				b, err := os.ReadFile(filepath.Join(logDir, f))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(evil, f), b, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			// An unopenable log directory too, so the ls error row is exercised.
			bad := filepath.Join(parent, "bad"+name)
			if err := os.MkdirAll(bad, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bad, "quill.format"), []byte("garbage"), 0o600); err != nil {
				t.Fatal(err)
			}

			// ls: one header, "orders", the evil log, the bad log: four lines, none forged.
			_, out, errw := exec(t, "", "-dir", parent, "ls")
			if n := noRawControl(t, "ls stdout", out) + noRawControl(t, "ls stderr", errw); n != 4 {
				t.Errorf("ls printed %d lines, want 4 (header and three logs):\nstdout %q\nstderr %q", n, out, errw)
			}
			// consumers and check: the file is reported in one row or one problem line.
			_, out, errw = exec(t, "", "-dir", logDir, "consumers")
			if n := noRawControl(t, "consumers stdout", out) + noRawControl(t, "consumers stderr", errw); n != 2 {
				t.Errorf("consumers printed %d lines, want 2 (header and one consumer):\nstdout %q\nstderr %q", n, out, errw)
			}
			_, out, errw = exec(t, "", "-dir", logDir, "check")
			noRawControl(t, "check stdout", out)
			noRawControl(t, "check stderr", errw)
			for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
				line = strings.TrimPrefix(line, "\"") // a problem line that needed escaping is quoted whole
				if line != "HEALTHY" && line != "UNHEALTHY" && !strings.HasPrefix(line, "UNBOUNDED") &&
					!strings.HasPrefix(line, "consumer ") && !strings.HasPrefix(line, "segment ") && !strings.HasPrefix(line, "file ") &&
					!strings.HasPrefix(line, "name ") && !strings.HasPrefix(line, "stray ") {
					t.Errorf("check printed a line that is not a problem of a known kind (forged?): %q\nwhole output %q", line, out)
				}
			}
			// -json stays what encoding/json makes of it, and is one line per object.
			_, out, _ = exec(t, "", "-dir", logDir, "consumers", "-json")
			if strings.Count(out, "\n") != 1 || strings.ContainsAny(out[:len(out)-1], "\x1b\x07") {
				t.Errorf("consumers -json is not one escaped line: %q", out)
			}
		})
	}
}

// The ls success row: a real, openable log in a hostile-named directory (the case above gives its
// hostile logs no segment, so they take the error row). The name must print quoted, in one row.
func TestLsSuccessRowEscapesTheLogDirectoryName(t *testing.T) {
	for _, name := range hostileNames {
		t.Run(strconv.Quote(name), func(t *testing.T) {
			parent := t.TempDir()
			l, err := quill.Open(quill.Config{Dir: filepath.Join(parent, name), Retention: -1, FlushInterval: time.Hour})
			if err != nil {
				t.Skipf("the file system refuses the name: %v", err)
			}
			if _, err := l.Append([]byte("rec")); err != nil {
				t.Fatal(err)
			}
			if err := l.Close(); err != nil {
				t.Fatal(err)
			}
			code, out, errw := exec(t, "", "-dir", parent, "ls")
			if code != exitOK {
				t.Fatalf("ls exit %d: %q %q", code, out, errw)
			}
			if n := noRawControl(t, "ls stdout", out) + noRawControl(t, "ls stderr", errw); n != 2 {
				t.Errorf("ls printed %d lines, want 2 (header and one log):\nstdout %q\nstderr %q", n, out, errw)
			}
			if !strings.Contains(out, strconv.Quote(name)) || strings.Contains(out, "error:") {
				t.Errorf("ls did not print the quoted name of a healthy log: %q", out)
			}
		})
	}
}

// The common error path: a -dir that is not a log and holds a hostile byte comes back in the
// message (a path built from the operator's or a script's input), which must be escaped.
func TestErrorMessageEscapesAHostileDirPath(t *testing.T) {
	for _, name := range hostileNames {
		t.Run(strconv.Quote(name), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), name)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Skipf("the file system refuses the name: %v", err)
			}
			code, out, errw := exec(t, "", "-dir", dir, "stat")
			if code != exitError {
				t.Fatalf("stat exit %d, want %d: %q %q", code, exitError, out, errw)
			}
			if n := noRawControl(t, "stat stdout", out) + noRawControl(t, "stat stderr", errw); n != 1 {
				t.Errorf("stat printed %d lines, want 1:\nstdout %q\nstderr %q", n, out, errw)
			}
			if !strings.Contains(errw, "not a log") {
				t.Errorf("stat stderr does not say the directory is not a log: %q", errw)
			}
		})
	}
}

// A file name that is not valid UTF-8 cannot be created on macOS; where it can, it must be
// quoted (the replacement character would hide it).
func TestNamesThatAreNotUTF8AreEscaped(t *testing.T) {
	logDir := newLog(t, 1, func(i int) []byte { return []byte("rec") })
	if err := os.MkdirAll(filepath.Join(logDir, "consumers"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "consumers", "bad\xff\xfename"), []byte("x"), 0o600); err != nil {
		t.Skipf("the file system refuses a name that is not UTF-8: %v", err)
	}
	_, out, errw := exec(t, "", "-dir", logDir, "consumers")
	noRawControl(t, "consumers stdout", out)
	noRawControl(t, "consumers stderr", errw)
	if !strings.Contains(out, `\xff\xfe`) {
		t.Errorf("the invalid bytes are not shown as escapes: %q", out)
	}
}

// safeText leaves a plain name alone and quotes anything a terminal could act on.
func TestSafeText(t *testing.T) {
	plain := []string{"", "orders", "a-b_c.9", "with space", "héllo", "日本語", `say "hi"`, `back\slash`}
	for _, s := range plain {
		if got := safeText(s); got != s {
			t.Errorf("safeText(%q) = %q, want it unchanged", s, got)
		}
	}
	for s, want := range map[string]string{
		"a\nb":         `"a\nb"`,
		"a\x1b[2Jb":    `"a\x1b[2Jb"`,
		"a\tb":         `"a\tb"`,
		"a\xffb":       `"a\xffb"`,
		"a\u202eb":     `"a\u202eb"`,
		"a\u009bb":     `"a\u009bb"`,
		"a\x7fb":       `"a\x7fb"`,
		"a\U000e0001b": `"a\U000e0001b"`,
		"a\"\x00\\":    `"a\"\x00\\"`,
	} {
		if got := safeText(s); got != want {
			t.Errorf("safeText(%q) = %s, want %s", s, got, want)
		}
		if got := safeText(s); got != strconv.Quote(s) {
			t.Errorf("safeText(%q) = %s differs from strconv.Quote %s", s, got, strconv.Quote(s))
		}
	}
}

// A plain log, consumer and problem list print as they always did.
func TestPlainNamesPrintUnchanged(t *testing.T) {
	logDir := newLog(t, 3, func(i int) []byte { return []byte("rec") })
	_, out, _ := exec(t, "", "-dir", filepath.Dir(logDir), "ls")
	if !strings.Contains(out, "\norders  ") {
		t.Errorf("ls row for a plain name changed:\n%s", out)
	}
	mustRun(t, "", "-dir", logDir, "register", "-name", "worker-1")
	out = mustRun(t, "", "-dir", logDir, "consumers")
	if !strings.Contains(out, "\nworker-1  ") {
		t.Errorf("consumers row for a plain name changed:\n%s", out)
	}
}
