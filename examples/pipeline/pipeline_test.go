package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain lets the test binary play a role, so the test can run the writer and the
// reader as two real processes.
func TestMain(m *testing.M) {
	if os.Getenv("PIPELINE_ROLE") != "" {
		os.Args = append([]string{os.Args[0]}, strings.Fields(os.Getenv("PIPELINE_ARGS"))...)
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func start(t *testing.T, args string) (*exec.Cmd, *bufio.Scanner) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "PIPELINE_ROLE=1", "PIPELINE_ARGS="+args)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd, bufio.NewScanner(out)
}

func TestAReaderProcessStartsLateCatchesUpAndFollowsAWriterProcess(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pipe")
	writer, wout := start(t, "-dir "+dir+" -role writer -count 30")
	// Let the writer get ahead, then start the reader late.
	for i := 0; i < 10 && wout.Scan(); i++ {
	}
	reader, rout := start(t, "-dir "+dir+" -role reader -count 30")
	var got []string
	deadline := time.After(20 * time.Second)
	done := make(chan struct{})
	go func() {
		for rout.Scan() {
			got = append(got, rout.Text())
		}
		close(done)
	}()
	select {
	case <-done:
	case <-deadline:
		_ = reader.Process.Kill()
		t.Fatal("the reader did not finish")
	}
	for wout.Scan() {
	}
	_ = writer.Wait()
	_ = reader.Wait()
	if len(got) != 30 || !strings.HasPrefix(got[0], "reader: 1 event 1") || !strings.HasPrefix(got[29], "reader: 30 event 30") {
		t.Fatalf("reader saw %d lines: %v", len(got), got)
	}
}
