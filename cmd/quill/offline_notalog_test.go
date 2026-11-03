package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `quill roll`, `recover` and `prune` act on an existing log. At a -dir with no quill.format (a
// mistyped path: one that does not exist, or an existing empty or populated directory) they print the
// text the read verbs print, exit 1 and create nothing: not the log, not its parent
// directories, not quill.lock (A3-23). `append` creating the log on first use is intended and is not
// tested here.
func TestRollRecoverAndPruneRefuseADirectoryThatIsNotALog(t *testing.T) {
	verbs := map[string][]string{
		"roll":              {"roll"},
		"recover":           {"recover"},
		"prune":             {"prune", "-retention", "1h"},
		"prune bare":        {"prune"},
		"prune with policy": {"prune", "-retention", "1h", "-max-bytes", "100000000", "-max-records", "10"},
	}
	for name, args := range verbs {
		for _, kind := range []string{"missing", "empty", "populated"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				root := t.TempDir()
				dir := filepath.Join(root, "deeper", "ordrs")
				switch kind {
				case "empty":
					dir = filepath.Join(root, "ordrs")
					if err := os.Mkdir(dir, 0o700); err != nil {
						t.Fatal(err)
					}
				case "populated":
					dir = filepath.Join(root, "ordrs")
					if err := os.Mkdir(dir, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				before := tree(t, root)
				code, out, errw := exec(t, "", append([]string{"-dir", dir}, args...)...)
				if code != exitError {
					t.Fatalf("exit %d, want %d\n%s%s", code, exitError, out, errw)
				}
				if out != "" {
					t.Errorf("stdout %q, want nothing", out)
				}
				if want := "quill: not a log: " + dir + " has no quill.format\n"; errw != want {
					t.Errorf("stderr %q, want %q", errw, want)
				}
				if after := tree(t, root); after != before {
					t.Errorf("the directory tree changed:\nbefore %s\nafter  %s", before, after)
				}
			})
		}
	}
}

// `append` creating the log at a new or empty -dir on first use is intended (decision 62).
func TestAppendStillCreatesTheLogAtANewOrEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	empty := filepath.Join(root, "empty")
	if err := os.Mkdir(empty, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{empty, filepath.Join(root, "deeper", "new")} {
		if out := mustRun(t, "", "-dir", dir, "append", "first"); out != "1\n" {
			t.Errorf("append at %s printed %q, want 1", dir, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "quill.format")); err != nil {
			t.Errorf("append did not create the log: %v", err)
		}
	}
}

// tree lists every path under root, so a test can see that a verb created nothing.
func tree(t *testing.T, root string) string {
	t.Helper()
	var names []string
	err := filepath.WalkDir(root, func(p string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		names = append(names, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(names, "\n")
}
