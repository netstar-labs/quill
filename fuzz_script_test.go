package quill

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hack/fuzz.sh says it runs every fuzz target in the module. A package that fails to
// compile makes `go test -list` fail with no output, which the script used to read as
// "this package has no fuzz targets", so it exited 0 with that package's targets never run.
// This runs the real script in a throwaway module: it must pass when every target lists and
// runs, and fail when a package does not compile or when no target ran at all.
func TestFuzzScriptFailsClosed(t *testing.T) {
	for _, tool := range []string{"bash", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	script, err := os.ReadFile(filepath.Join("hack", "fuzz.sh"))
	if err != nil {
		t.Fatal(err)
	}
	goMod := pinnedGoMod(t)

	const fuzzTarget = "package a\n\nimport \"testing\"\n\nfunc FuzzA(f *testing.F) { f.Add(1); f.Fuzz(func(t *testing.T, n int) {}) }\n"
	const plainTest = "package a\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n"

	for _, tc := range []struct {
		name  string
		files map[string]string
		pass  bool
		want  string // text the output must hold
	}{
		{"all targets run", map[string]string{"a/a.go": "package a\n", "a/a_test.go": fuzzTarget}, true, "fuzz example.com/q/a FuzzA"},
		{"a package does not compile", map[string]string{
			"a/a.go": "package a\n", "a/a_test.go": fuzzTarget,
			"b/b.go": "package b\n\nvar x int = \"not an int\"\n", "b/b_test.go": strings.Replace(plainTest, "package a", "package b", 1),
		}, false, "example.com/q/b"},
		{"a test file does not compile", map[string]string{
			"a/a.go": "package a\n", "a/a_test.go": fuzzTarget + "\nvar y int = \"no\"\n",
		}, false, "example.com/q/a"},
		{"no fuzz target at all", map[string]string{"a/a.go": "package a\n", "a/a_test.go": plainTest}, false, "no fuzz target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{"go.mod": goMod, "hack/fuzz.sh": string(script)}
			for k, v := range tc.files {
				files[k] = v
			}
			for name, body := range files {
				p := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("bash", "hack/fuzz.sh")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "FUZZTIME=1x", "GOFLAGS=-mod=mod")
			out, err := cmd.CombinedOutput()
			if tc.pass && err != nil {
				t.Fatalf("fuzz.sh failed: %v\n%s", err, out)
			}
			if !tc.pass && err == nil {
				t.Fatalf("fuzz.sh exited 0 though it must fail\n%s", out)
			}
			if !strings.Contains(string(out), tc.want) {
				t.Fatalf("output does not mention %q:\n%s", tc.want, out)
			}
		})
	}
}
