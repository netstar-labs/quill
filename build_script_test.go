package quill

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The --dev form of the build script is what CI runs on a fresh checkout, which is a clean
// tree. The script runs under errexit, so a version string computed with a command whose
// status is 1 on a clean tree ends it silently: no output, exit 1, no build/install.
// This runs the real script in a throwaway module, once clean and once dirty.
func TestBuildScriptDevVersion(t *testing.T) {
	for _, tool := range []string{"bash", "git", "go", "shasum"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	script, err := os.ReadFile(filepath.Join("build", "quill"))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name  string
		dirty bool
	}{{"clean", false}, {"dirty", true}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write := func(name, body string, mode os.FileMode) {
				t.Helper()
				p := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), mode); err != nil {
					t.Fatal(err)
				}
			}
			write("go.mod", "module example.com/q\n\ngo 1.21\n", 0o644)
			write("cmd/quill/main.go", "package main\n\nvar version = \"unset\"\n\nfunc main() { _ = version }\n", 0o644)
			for _, f := range []string{"README.md", "CHANGELOG.md", "SECURITY.md"} {
				write(f, f+"\n", 0o644)
			}
			write("build/quill", string(script), 0o755)

			env := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
				"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
				"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
			run := func(name string, args ...string) []byte {
				t.Helper()
				cmd := exec.Command(name, args...)
				cmd.Dir = dir
				cmd.Env = env
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("%s %v: %v\n%s", name, args, err, out)
				}
				return out
			}
			run("git", "init", "-q")
			run("git", "add", ".")
			run("git", "commit", "-q", "-m", "x")
			sha := strings.TrimSpace(string(run("git", "rev-parse", "--short", "HEAD")))
			want := "dev-" + sha
			if tc.dirty {
				write("untracked", "x\n", 0o644)
				want += "-dirty"
			}

			cmd := exec.Command("bash", "build/quill", "--dev")
			cmd.Dir = dir
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("build/quill --dev on a %s tree: %v\n%s", tc.name, err, out)
			}
			matches, _ := filepath.Glob(filepath.Join(dir, "build", "install", "quill_"+want+"_*.tar.gz"))
			if len(matches) != 4 {
				t.Fatalf("want 4 archives for version %q, got %v\n%s", want, matches, out)
			}
			if fi, err := os.Stat(filepath.Join(dir, "build", "install", "SHA256SUMS")); err != nil || fi.Size() == 0 {
				t.Fatalf("SHA256SUMS missing or empty: %v", err)
			}
		})
	}
}
