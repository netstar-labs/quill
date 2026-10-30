package quill

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// modeOf is the permission bits of a file or directory on the simulator.
func modeOf(t *testing.T, s *fsx.Sim, path string) fs.FileMode {
	t.Helper()
	info, err := s.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

// A1-85: a consumer file created through a Reader, and the files and directories Archive
// creates, take the log's mode, so the writer (another account in the log's group) can read
// them and does not treat the consumer as broken (decisions 41, 51, 67).
func TestAReaderCreatesConsumerFilesWithTheLogsModeAndArchiveCreatesTheDestinationWithIt(t *testing.T) {
	s, clk, l := retentionLog(t, func(c *Config) { c.Perm = 0o660; c.Retention = -1 })
	defer l.Close()
	fillSegments(t, l, clk, 2)
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	c, err := r.Consumer("viareader", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := archive(s, "/log", "/arch"); err != nil {
		t.Fatal(err)
	}
	want := map[string]fs.FileMode{
		"/log/consumers/viareader":                       0o660,
		"/log/consumers/viareader" + lockSuffix:          0o660,
		"/log/consumers/" + archiveConsumer:              0o660,
		"/log/consumers/" + archiveConsumer + lockSuffix: 0o660,
		filepath.Join("/arch", formatName):               0o660,
		filepath.Join("/arch", tailName):                 0o660,
		filepath.Join("/arch", segmentName(0)):           0o660,
		"/arch":                                          0o770,
		filepath.Join("/arch", consumersDir):             0o770,
	}
	for p, m := range want {
		if got := modeOf(t, s, p); got != m {
			t.Errorf("%s is %v, want %v", p, got, m)
		}
	}
	// The writer can list what the reader created: nothing is broken.
	list, err := l.Consumers()
	if err != nil {
		t.Fatal(err)
	}
	for _, ci := range list {
		if ci.Err != nil {
			t.Errorf("%s: %v", ci.Name, ci.Err)
		}
	}
}

func TestADefaultModeLogStillGivesReaderCreatedFiles0600(t *testing.T) {
	s, _, l := retentionLog(t, nil)
	defer l.Close()
	r, err := OpenReader("/log", ReaderOptions{fs: s})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	c, err := r.Consumer("a", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if got := modeOf(t, s, "/log/consumers/a"); got != 0o600 {
		t.Fatalf("%v", got)
	}
}

// On a real filesystem the offset file's mode is exact whatever the umask (decision 74).
func TestAReaderConsumerFileOnDiskTakesTheLogsModeWhateverTheUmask(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "log")
	l := openDir(t, dir, func(c *Config) { c.Perm = 0o640 })
	defer l.Close()
	r, err := OpenReader(dir, ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	c, err := r.Consumer("a", ConsumerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	info, err := os.Stat(filepath.Join(dir, consumersDir, "a"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("consumer file is %v, want 0640", info.Mode().Perm())
	}
}
