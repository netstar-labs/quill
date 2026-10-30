package quill

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	write("go.mod", "module example.com/q\n\ngo 1.21\n", 0o644)
	write("cmd/quill/main.go", "package main\n\nvar version = \"unset\"\n\nfunc main() { _ = version }\n", 0o644)
	for _, f := range []string{"README.md", "CHANGELOG.md", "SECURITY.md", "LICENSE"} {
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
		if !sort.StringsAreSorted(names) || len(names) != 6 {
			t.Errorf("%s: entries %v, want 6 in name order", base, names)
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
