package quill

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

func TestCheckReportsAnUnreadableConsumersDirectoryAsUnhealthy(t *testing.T) {
	s := fsx.NewSim()
	l := openSim(t, s, nil)
	defer l.Close()
	appendN(t, l, 3)
	mustFlush(t, l)
	r := openReaderSim(t, s)
	defer r.Close()
	if res, err := r.Check(); err != nil || !res.Healthy {
		t.Fatalf("before the injection: %+v %v", res, err)
	}
	s.Inject = func(op fsx.Op, name string) error {
		if op == fsx.OpReadDir && filepath.Clean(name) == "/log/consumers" {
			return &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrPermission}
		}
		return nil
	}
	res, err := r.Check()
	s.Inject = nil
	if err != nil {
		t.Fatal(err)
	}
	if res.Healthy || len(res.Problems) != 1 {
		t.Fatalf("unreadable consumers/: %+v", res)
	}
	p := res.Problems[0]
	if p.Kind != "consumer" || !p.Fatal || p.Path != "/log/consumers" || !strings.Contains(p.Detail, "permission") {
		t.Fatalf("problem %+v", p)
	}
}
