package quill

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A state file that is the wrong size or the wrong kind of file is damage, and damage is
// ErrCorrupt, as it is for a state file that does not parse (decision 6, guarantee 7): a caller
// that branches on ErrCorrupt must not see it as an I/O or configuration error.
func TestAnOversizeOrNonRegularStateFileIsErrCorrupt(t *testing.T) {
	plant := map[string]func(t *testing.T, path string){
		"oversize": func(t *testing.T, path string) {
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if err := f.Truncate(2 << 20); err != nil {
				t.Fatal(err)
			}
		},
		"directory": func(t *testing.T, path string) {
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		},
		"fifo": plantFIFO,
		"symlink": func(t *testing.T, path string) {
			victim := filepath.Join(t.TempDir(), "victim")
			if err := os.WriteFile(victim, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(victim, path); err != nil {
				t.Fatal(err)
			}
		},
		"dangling symlink": func(t *testing.T, path string) {
			if err := os.Symlink(filepath.Join(t.TempDir(), "nowhere"), path); err != nil {
				t.Fatal(err)
			}
		},
	}
	for kind, do := range plant {
		for _, name := range []string{tailName, formatName} {
			t.Run(kind+"/"+name, func(t *testing.T) {
				dir := makeLog(t)
				path := filepath.Join(dir, name)
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
				do(t, path)
				if r, err := OpenReader(dir, ReaderOptions{}); err == nil {
					r.Close()
					t.Error("OpenReader succeeded")
				} else if !errors.Is(err, ErrCorrupt) {
					t.Errorf("OpenReader: %v is not ErrCorrupt", err)
				}
				cfg := Config{Dir: dir, FlushInterval: time.Hour, Retention: -1}
				if l, err := Open(cfg); err == nil {
					l.Close()
					t.Error("Open succeeded")
				} else if !errors.Is(err, ErrCorrupt) {
					t.Errorf("Open: %v is not ErrCorrupt", err)
				}
			})
		}
	}
}

// The consumer offset files are read through the same path: an oversize one, or a symbolic
// link in its place, is listed with an ErrCorrupt and refuses the consumer's open with it.
func TestAnOversizeOrLinkedConsumerFileIsErrCorrupt(t *testing.T) {
	for _, kind := range []string{"oversize", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := makeLog(t)
			path := filepath.Join(dir, consumersDir, "x")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if kind == "oversize" {
				if err := os.WriteFile(path, make([]byte, 2<<20), 0o600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink(filepath.Join(t.TempDir(), "nowhere"), path); err != nil {
				t.Fatal(err)
			}
			r, err := OpenReader(dir, ReaderOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			list, err := r.Consumers()
			if err != nil || len(list) != 1 || !errors.Is(list[0].Err, ErrCorrupt) {
				t.Errorf("Consumers = %+v, %v; want one entry x whose Err is ErrCorrupt", list, err)
			}
			if c, err := r.Consumer("x", ConsumerOptions{}); err == nil {
				c.Close()
				t.Error("Consumer succeeded")
			} else if !errors.Is(err, ErrCorrupt) {
				t.Errorf("Consumer: %v is not ErrCorrupt", err)
			}
		})
	}
}
