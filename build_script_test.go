package quill

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
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
			write("go.mod", pinnedGoMod(t), 0o644)
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

// The archives and the SBOM are a function of the commit, not of the builder or the clock:
// two builds of one commit, seconds apart, give the same SHA256SUMS, and an archive carries
// no builder identity (uid, gid, user and group names), no wall-clock time and a stable
// entry order. The binaries were already identical; this covers everything around them.
func TestBuildScriptReproducibleArchives(t *testing.T) {
	for _, tool := range []string{"bash", "git", "go", "shasum", "tar", "gzip"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	script, err := os.ReadFile(filepath.Join("build", "quill"))
	if err != nil {
		t.Fatal(err)
	}
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
	write("go.mod", pinnedGoMod(t), 0o644)
	write("cmd/quill/main.go", "package main\n\nvar version = \"unset\"\n\nfunc main() { _ = version }\n", 0o644)
	for _, f := range []string{"README.md", "CHANGELOG.md", "SECURITY.md", "LICENSE", "THIRD-PARTY-NOTICES"} {
		write(f, f+"\n", 0o644)
	}
	write("build/quill", string(script), 0o755)
	write(".gitignore", "build/install/\nbuild/quill.version\n", 0o644) // as the real .gitignore: a second build sees a clean tree

	// A commit time far from the build's wall clock, so a leak of either shows.
	const commitTime = "2001-02-03T04:05:06Z"
	env := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
		"GIT_AUTHOR_DATE="+commitTime, "GIT_COMMITTER_DATE="+commitTime)
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
	ct := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)

	build := func() string {
		t.Helper()
		run("bash", "build/quill", "--dev")
		sums, err := os.ReadFile(filepath.Join(dir, "build", "install", "SHA256SUMS"))
		if err != nil {
			t.Fatal(err)
		}
		return string(sums)
	}
	first := build()
	archives, _ := filepath.Glob(filepath.Join(dir, "build", "install", "*.tar.gz"))
	if len(archives) != 4 {
		t.Fatalf("want 4 archives, got %v", archives)
	}
	for _, a := range archives {
		base := filepath.Base(a)
		f, err := os.Open(a)
		if err != nil {
			t.Fatal(err)
		}
		zr, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		if !zr.ModTime.IsZero() && zr.ModTime.Unix() != 0 {
			t.Errorf("%s: gzip header carries a time, %v", base, zr.ModTime)
		}
		var names []string
		var notices []byte
		tr := tar.NewReader(zr)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			names = append(names, h.Name)
			if strings.HasSuffix(h.Name, "/THIRD-PARTY-NOTICES") {
				if notices, err = io.ReadAll(tr); err != nil {
					t.Fatal(err)
				}
			}
			if h.Uid != 0 || h.Gid != 0 || h.Uname != "" || h.Gname != "" {
				t.Errorf("%s: %s carries an owner: uid %d gid %d uname %q gname %q",
					base, h.Name, h.Uid, h.Gid, h.Uname, h.Gname)
			}
			if !h.ModTime.Equal(ct) {
				t.Errorf("%s: %s mtime %v, want the commit time %v", base, h.Name, h.ModTime.UTC(), ct)
			}
			want := int64(0o644)
			if h.Typeflag == tar.TypeDir || strings.HasSuffix(h.Name, "/quill") {
				want = 0o755
			}
			if h.Mode&0o7777 != want {
				t.Errorf("%s: %s mode %o, want %o", base, h.Name, h.Mode, want)
			}
		}
		f.Close()
		if !sort.StringsAreSorted(names) || len(names) != 7 {
			t.Errorf("%s: entries %v, want 7 in name order", base, names)
		}
		// The Go runtime and standard library are linked into the binary, and their licence asks
		// for its notice in binary redistributions: every archive carries the repository's file.
		if string(notices) != "THIRD-PARTY-NOTICES\n" {
			t.Errorf("%s: THIRD-PARTY-NOTICES is %q, want the repository file's bytes", base, notices)
		}
	}
	sbom, _ := filepath.Glob(filepath.Join(dir, "build", "install", "*.cdx.json"))
	if len(sbom) != 1 {
		t.Fatalf("want one SBOM, got %v", sbom)
	}
	body, _ := os.ReadFile(sbom[0])
	if !strings.Contains(string(body), `"timestamp": "`+commitTime+`"`) {
		t.Errorf("SBOM timestamp is not the commit time %s:\n%s", commitTime, body)
	}

	// Every dependency edge names a component by its bom-ref; a ref nobody declares leaves the
	// graph dangling for any consumer that resolves it.
	var bom struct {
		Metadata struct {
			Component struct {
				BomRef string `json:"bom-ref"`
			} `json:"component"`
		} `json:"metadata"`
		Components []struct {
			BomRef string `json:"bom-ref"`
		} `json:"components"`
		Dependencies []struct {
			Ref       string   `json:"ref"`
			DependsOn []string `json:"dependsOn"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(body, &bom); err != nil {
		t.Fatalf("SBOM is not JSON: %v\n%s", err, body)
	}
	declared := map[string]bool{bom.Metadata.Component.BomRef: bom.Metadata.Component.BomRef != ""}
	for _, c := range bom.Components {
		declared[c.BomRef] = c.BomRef != ""
	}
	if len(bom.Dependencies) == 0 {
		t.Errorf("SBOM has no dependencies block:\n%s", body)
	}
	for _, d := range bom.Dependencies {
		for _, ref := range append([]string{d.Ref}, d.DependsOn...) {
			if !declared[ref] {
				t.Errorf("SBOM dependency ref %q is not the bom-ref of any component", ref)
			}
		}
	}

	time.Sleep(2100 * time.Millisecond)
	if second := build(); second != first {
		t.Errorf("two builds of one commit differ:\n%s\n---\n%s", first, second)
	}
}

// The release binaries statically link the Go runtime and standard library (BSD-3-Clause),
// whose licence asks for its text with a binary redistribution (audit A4-5). The repository's
// THIRD-PARTY-NOTICES carries the toolchain's LICENSE verbatim, so a toolchain bump that changes
// the licence text fails here until the file is refreshed.
func TestThirdPartyNoticesCarryGoLicense(t *testing.T) {
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		t.Skipf("go env GOROOT: %v", err)
	}
	lic, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), "LICENSE"))
	if err != nil {
		t.Skipf("the toolchain's LICENSE is not readable: %v", err)
	}
	notices, err := os.ReadFile("THIRD-PARTY-NOTICES")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notices), string(lic)) {
		t.Errorf("THIRD-PARTY-NOTICES does not contain the Go LICENSE text of %s verbatim", runtime.Version())
	}
}

// Decision 88: the release form of the build script (no --dev) refuses a dirty tree and a tree
// whose HEAD is not tagged, and builds a clean tagged one. The refusal is two short conditions
// that nothing else exercises; an edit to either could let a dirty or untagged release through
// with every other test green. Each case runs the real script in a throwaway repository.
func TestBuildScriptReleaseRefusal(t *testing.T) {
	for _, tool := range []string{"bash", "git", "go", "shasum"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	script, err := os.ReadFile(filepath.Join("build", "quill"))
	if err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")

	for _, tc := range []struct {
		name   string
		tag    bool
		mutate func(write func(name, body string))
		refuse string // substring of the refusal message; empty when the build must succeed
	}{
		{"clean and tagged builds", true, nil, ""},
		{"ignored file is not dirt", true, func(w func(string, string)) { w("build/quill.version", "x\n") }, ""},
		{"untagged", false, nil, "not tagged"},
		{"tagged but a tracked file is modified", true, func(w func(string, string)) { w("README.md", "changed\n") }, "dirty"},
		{"tagged but an untracked file exists", true, func(w func(string, string)) { w("untracked", "x\n") }, "dirty"},
		{"untagged and dirty", false, func(w func(string, string)) { w("untracked", "x\n") }, "dirty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write := func(name, body string) {
				t.Helper()
				mode := os.FileMode(0o644)
				if name == "build/quill" {
					mode = 0o755
				}
				p := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), mode); err != nil {
					t.Fatal(err)
				}
			}
			write("go.mod", pinnedGoMod(t))
			write("cmd/quill/main.go", "package main\n\nvar version = \"unset\"\n\nfunc main() { _ = version }\n")
			for _, f := range []string{"README.md", "CHANGELOG.md", "SECURITY.md"} {
				write(f, f+"\n")
			}
			write("build/quill", string(script))
			write(".gitignore", "build/install/\nbuild/quill.version\n")

			run := func(name string, args ...string) {
				t.Helper()
				cmd := exec.Command(name, args...)
				cmd.Dir = dir
				cmd.Env = env
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("%s %v: %v\n%s", name, args, err, out)
				}
			}
			run("git", "init", "-q")
			run("git", "add", ".")
			run("git", "commit", "-q", "-m", "x")
			if tc.tag {
				run("git", "tag", "v0.1.0")
			}
			if tc.mutate != nil {
				tc.mutate(write)
			}

			cmd := exec.Command("bash", "build/quill")
			cmd.Dir = dir
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			archives, _ := filepath.Glob(filepath.Join(dir, "build", "install", "quill_v0.1.0_*.tar.gz"))
			if tc.refuse == "" {
				if err != nil {
					t.Fatalf("release build: %v\n%s", err, out)
				}
				if len(archives) != 4 {
					t.Fatalf("want 4 archives for v0.1.0, got %v\n%s", archives, out)
				}
				return
			}
			if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
				t.Fatalf("release build must exit 1, got %v\n%s", err, out)
			}
			if !strings.Contains(string(out), tc.refuse) {
				t.Errorf("refusal message %q does not mention %q", out, tc.refuse)
			}
			if _, statErr := os.Stat(filepath.Join(dir, "build", "install")); statErr == nil {
				t.Errorf("a refused build left build/install behind")
			}
		})
	}
}

// The release build is the authority for what ships (decisions 70 and 88), so it must run on
// the Go that go.mod names, exactly as the commit gate does, not on whichever go is first on
// PATH: under GOTOOLCHAIN=auto a toolchain line is only a minimum. A stand-in go on PATH
// records the GOTOOLCHAIN every invocation ran under, so no real toolchain is downloaded.
func TestBuildScriptPinsToolchain(t *testing.T) {
	for _, tool := range []string{"bash", "git", "shasum"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	script, err := os.ReadFile(filepath.Join("build", "quill"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, gomod, want string }{
		{"toolchain line", "go 1.27\n\ntoolchain go1.27.1\n", "go1.27.1"},
		{"no toolchain line", "go 1.27\n", "go1.27.0"},
		{"patch in go line", "go 1.27.2\n", "go1.27.2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			bin := t.TempDir()
			log := filepath.Join(t.TempDir(), "calls")
			write := func(p, body string, mode os.FileMode) {
				t.Helper()
				p = filepath.Join(dir, p)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), mode); err != nil {
					t.Fatal(err)
				}
			}
			write("go.mod", "module example.com/q\n\n"+tc.gomod, 0o644)
			write("cmd/quill/main.go", "package main\n\nfunc main() {}\n", 0o644)
			for _, f := range []string{"README.md", "CHANGELOG.md", "SECURITY.md"} {
				write(f, f+"\n", 0o644)
			}
			write("build/quill", string(script), 0o755)
			fake := "#!/bin/sh\n" +
				"echo \"$1 ${GOTOOLCHAIN:-unpinned}\" >> '" + log + "'\n" +
				"case \"$1\" in\n" +
				"version) echo \"go version ${GOTOOLCHAIN:-unpinned} linux/amd64\" ;;\n" +
				"build) while [ $# -gt 0 ]; do [ \"$1\" = -o ] && { shift; mkdir -p \"$(dirname \"$1\")\"; echo x > \"$1\"; }; shift; done ;;\n" +
				"esac\n"
			if err := os.WriteFile(filepath.Join(bin, "go"), []byte(fake), 0o755); err != nil {
				t.Fatal(err)
			}

			var env []string
			for _, kv := range os.Environ() {
				if !strings.HasPrefix(kv, "GOTOOLCHAIN=") {
					env = append(env, kv)
				}
			}
			env = append(env, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
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
			run("bash", "build/quill", "--dev")

			calls, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
			if len(lines) < 5 { // one version query and four builds
				t.Fatalf("want the version query and four builds, got %q", lines)
			}
			for _, l := range lines {
				if !strings.HasSuffix(l, " "+tc.want) {
					t.Errorf("go ran as %q; want it pinned with GOTOOLCHAIN=%s", l, tc.want)
				}
			}
			sbom, err := os.ReadFile(filepath.Join(dir, "build", "install", "quill_dev-"+sha+".cdx.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(sbom), `"value": "`+tc.want+`"`) {
				t.Errorf("the SBOM does not record %s:\n%s", tc.want, sbom)
			}
		})
	}
}

// The compiler's architecture level changes the instructions it emits, so a builder's GOAMD64
// or GOARM64 (in the shell, or in the file `go env -w` writes, which the environment overrides)
// would change the shipped binaries and could make them die with SIGILL on an older CPU. The
// script pins the baseline levels, v1 and v8.0, for every build. A stand-in go on PATH records
// the two variables each `go build` ran with, under a builder that sets the higher levels.
func TestBuildScriptPinsArchLevel(t *testing.T) {
	for _, tool := range []string{"bash", "git", "shasum"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	script, err := os.ReadFile(filepath.Join("build", "quill"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	bin := t.TempDir()
	log := filepath.Join(t.TempDir(), "calls")
	write := func(p, body string, mode os.FileMode) {
		t.Helper()
		p = filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/q\n\ngo 1.27\n\ntoolchain go1.27.1\n", 0o644)
	write("cmd/quill/main.go", "package main\n\nfunc main() {}\n", 0o644)
	for _, f := range []string{"README.md", "CHANGELOG.md", "SECURITY.md"} {
		write(f, f+"\n", 0o644)
	}
	write("build/quill", string(script), 0o755)
	fake := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"version) echo \"go version go1.27.1 linux/amd64\" ;;\n" +
		"build) echo \"$GOOS/$GOARCH amd64=${GOAMD64:-unset} arm64=${GOARM64:-unset}\" >> '" + log + "'\n" +
		"  while [ $# -gt 0 ]; do [ \"$1\" = -o ] && { shift; mkdir -p \"$(dirname \"$1\")\"; echo x > \"$1\"; }; shift; done ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GOAMD64=v3", "GOARM64=v9.0",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"commit", "-q", "-m", "x"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	cmd := exec.Command("bash", "build/quill", "--dev")
	cmd.Dir = dir
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build/quill --dev: %v\n%s", err, out)
	}
	calls, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
	if len(lines) != 4 {
		t.Fatalf("want four builds, got %q", lines)
	}
	for _, l := range lines {
		if !strings.HasSuffix(l, " amd64=v1 arm64=v8.0") {
			t.Errorf("go build ran as %q; want GOAMD64=v1 and GOARM64=v8.0 whatever the builder sets", l)
		}
	}
}

// The script pins GOTOOLCHAIN to go.mod's toolchain line. A throwaway module for a test that
// runs the real go names the Go running the test, so the build neither downloads nor switches.
func pinnedGoMod(t *testing.T) string {
	t.Helper()
	if !regexp.MustCompile(`^go\d+\.\d+\.\d+$`).MatchString(runtime.Version()) {
		t.Skipf("%s is not a release toolchain", runtime.Version())
	}
	return "module example.com/q\n\ngo 1.21\n\ntoolchain " + runtime.Version() + "\n"
}
