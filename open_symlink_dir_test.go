package quill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A symbolic link to a directory is followed wherever it sits in Dir, as OpenReader follows
// one: as the final component of an existing log directory, and as an ancestor of a log
// directory that does not exist yet (A1-29).
func TestOpenFollowsASymbolicLinkToADirectoryInDir(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	// Dir is the link, and the log is created in its target.
	l := openDir(t, link, nil)
	appendN(t, l, 3)
	mustFlush(t, l)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(real, tailName)); err != nil {
		t.Fatalf("the log is not in the link's target: %v", err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the link was replaced: %v %v", info, err)
	}
	r, err := OpenReader(link, ReaderOptions{})
	if err != nil {
		t.Fatalf("OpenReader through the link: %v", err)
	}
	if h, _ := r.Head(); h != 3 {
		t.Fatalf("head through the link %d, want 3", h)
	}
	r.Close()
	l = openDir(t, link, nil) // it reopens, taking the lock in the target
	if l.Head() != 3 {
		t.Fatalf("reopened head %d, want 3", l.Head())
	}
	if _, err := Open(Config{Dir: real}); !errors.Is(err, ErrLocked) {
		t.Fatalf("open of the target while the link's writer holds it: %v, want ErrLocked", err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	// Dir is below the link and does not exist yet: the first-run deployment whose data root
	// is a link. One level, and several.
	for _, rel := range []string{"sub", filepath.Join("a", "b")} {
		l := openDir(t, filepath.Join(link, rel), nil)
		appendN(t, l, 2)
		mustFlush(t, l)
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(real, rel, tailName)); err != nil {
			t.Fatalf("Dir %s: the log is not below the link's target: %v", rel, err)
		}
		r, err := OpenReader(filepath.Join(link, rel), ReaderOptions{})
		if err != nil {
			t.Fatalf("OpenReader(%s): %v", rel, err)
		}
		r.Close()
	}
}

// A symbolic link that does not lead to a directory is still refused with ErrNoLog, naming
// the link and saying so, and nothing is created behind it; a link inside the log directory
// is still not followed (decision 51).
func TestOpenStillRefusesASymbolicLinkThatIsNotADirectoryOrIsPlantedInside(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	toFile, dangling := filepath.Join(root, "tofile"), filepath.Join(root, "dangling")
	if err := os.Symlink(file, toFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "absent"), dangling); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{toFile, dangling, filepath.Join(dangling, "sub")} {
		_, err := Open(Config{Dir: dir})
		if !errors.Is(err, ErrNoLog) || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("Open(%s): %v, want ErrNoLog saying a symbolic link", dir, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, "absent")); err == nil {
		t.Fatal("Open created the target of a dangling link")
	}

	dir := filepath.Join(root, "log")
	l := openDir(t, dir, nil)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, tailName)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(file, filepath.Join(dir, tailName)); err != nil {
		t.Fatal(err)
	}
	if l, err := Open(Config{Dir: dir}); err == nil {
		_ = l.Close()
		t.Fatal("Open followed a quill.tail that is a symbolic link")
	}
}
