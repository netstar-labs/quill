package quill

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// T1 and T18 under a real SIGKILL: a writer process acknowledges sequences as they become
// durable; it is killed at a random moment; the recovered log holds every acknowledged
// record, is a contiguous prefix, and continues from its head.

func killWriter(t *testing.T, dir string, sync bool, after int) (acked uint64) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "QUILL_HELPER=writer", "QUILL_HELPER_DIR="+dir, "QUILL_HELPER_SEGBYTES=2048")
	if sync {
		cmd.Env = append(cmd.Env, "QUILL_HELPER_SYNC=1")
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(out)
	for i := 0; i < after && sc.Scan(); i++ {
		line := sc.Text()
		if !strings.HasPrefix(line, "ack ") {
			t.Fatalf("helper said %q", line)
		}
		n, err := strconv.ParseUint(line[4:], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		acked = n
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	return acked
}

func TestAKilledWriterLosesNoAcknowledgedRecordAndRecoversAContiguousPrefix(t *testing.T) {
	for round, sync := range []bool{false, true, false, true} {
		dir := filepath.Join(t.TempDir(), "log")
		acked := killWriter(t, dir, sync, 20+round*17)
		if acked == 0 {
			t.Fatal("nothing acknowledged")
		}
		l, err := Open(Config{Dir: dir, Retention: -1, FlushInterval: time.Hour})
		if err != nil {
			t.Fatalf("round %d: reopen: %v", round, err)
		}
		recs, err := readAll(l)
		if err != nil {
			t.Fatal(err)
		}
		last := checkContiguous(t, recs, 1)
		if last < acked {
			t.Fatalf("round %d: acknowledged %d, recovered %d", round, acked, last)
		}
		if len(l.holes) != 0 {
			t.Fatalf("holes %+v", l.holes)
		}
		seq, err := l.Append(payloadFor(last + 1))
		if err != nil || seq != last+1 {
			t.Fatalf("round %d: next %d %v", round, seq, err)
		}
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
		t.Logf("round %d (sync %v): acked %d, recovered %d", round, sync, acked, last)
		_ = fmt.Sprint
		if !errors.Is(err, nil) {
			t.Fatal(err)
		}
	}
}
