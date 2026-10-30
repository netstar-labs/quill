package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netstar-labs/quill"
)

// bigActive writes a log whose single active segment is about 9 MiB, above the 8 MiB default
// segment size, under a policy that says SegmentBytes is 1 GiB, with caps and a protected
// consumer so a rewrite of the policy would be visible.
func bigActive(t *testing.T) string {
	t.Helper()
	rec := bytes.Repeat([]byte("x"), 1<<20)
	return buildLog(t, quill.Config{Retention: time.Hour, SegmentBytes: quill.MaxSegmentBytes,
		MaxBytes: 4 << 30, MaxRecords: 1000, ProtectedConsumers: []string{"billing"}, FlushInterval: time.Hour}, 9,
		func(int) []byte { return rec })
}

func segmentFiles(t *testing.T, dir string) int {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		t.Fatal(err)
	}
	return len(m)
}

func readPolicyFile(t *testing.T, dir string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "quill.policy"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestOfflineVerbsLeaveQuillPolicyByteIdentical(t *testing.T) {
	dir := bigActive(t)
	want := readPolicyFile(t, dir)
	for _, args := range [][]string{
		{"append", "-dir", dir, "hello"},
		{"roll", "-dir", dir},
		{"recover", "-dir", dir},
		{"prune", "-dir", dir},
		{"prune", "-dir", dir, "-retention", "1s", "-max-records", "5"},
	} {
		mustRun(t, "", args...)
		if got := readPolicyFile(t, dir); !bytes.Equal(got, want) {
			t.Fatalf("after quill %v the policy changed:\n%s\nwant\n%s", args, got, want)
		}
	}
	// A flagless prune still acts on the writer's settings: retention 1h, nothing is old enough.
	out := mustRun(t, "", "prune", "-dir", dir)
	if !strings.Contains(out, "segments") {
		t.Fatalf("prune output %q", out)
	}
}

func TestOfflineAppendAndRecoverDoNotRotateAnActiveSegmentWithinThePolicySize(t *testing.T) {
	for _, verb := range [][]string{{"append", "more"}, {"recover"}} {
		dir := bigActive(t)
		if n := segmentFiles(t, dir); n != 1 {
			t.Fatalf("setup: %d segments", n)
		}
		args := append([]string{verb[0], "-dir", dir}, verb[1:]...)
		mustRun(t, "", args...)
		if n := segmentFiles(t, dir); n != 1 {
			t.Errorf("quill %s with a 1 GiB policy and a 9 MiB active segment left %d segments; want 1", verb[0], n)
		}
	}
}

func TestOfflineAppendTakesTheSegmentSizeFromThePolicy(t *testing.T) {
	dir := bigActive(t)
	p := filepath.Join(dir, "quill.policy")
	edited := strings.Replace(string(readPolicyFile(t, dir)), "1073741824", "65536", 1)
	if edited == string(readPolicyFile(t, dir)) {
		t.Fatal("setup: segment_bytes not found in the policy")
	}
	if err := os.WriteFile(p, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "", "append", "-dir", dir, "more")
	if n := segmentFiles(t, dir); n != 2 {
		t.Errorf("a 64 KiB policy and a 9 MiB active segment: %d segments after append; want 2", n)
	}
	if got := string(readPolicyFile(t, dir)); got != edited {
		t.Errorf("policy rewritten:\n%s", got)
	}
}

func TestOfflineVerbsWithoutAReadablePolicyUseTheSegmentCeilingAndWriteNoPolicy(t *testing.T) {
	dir := bigActive(t)
	if err := os.Remove(filepath.Join(dir, "quill.policy")); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "", "append", "-dir", dir, "more")
	mustRun(t, "", "recover", "-dir", dir)
	if n := segmentFiles(t, dir); n != 1 {
		t.Errorf("%d segments; want 1 (no policy: the ceiling is the segment size)", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "quill.policy")); !os.IsNotExist(err) {
		t.Errorf("an offline verb created quill.policy: %v", err)
	}
}

func TestRollRotatesRegardlessOfThePolicySize(t *testing.T) {
	dir := bigActive(t)
	mustRun(t, "", "roll", "-dir", dir)
	if n := segmentFiles(t, dir); n != 2 {
		t.Errorf("%d segments after roll; want 2", n)
	}
}

// A1-80: the policy lists a protected consumer whose process is running and whose file says
// unprotected (it holds the consumer's lock). prune borrows the log's lock only, so it must
// open without the policy's ProtectedConsumers and not need the consumer's lock.
func TestPruneBesideARunningConsumerTheLogsPolicyProtects(t *testing.T) {
	dir := bigActive(t)
	r, err := quill.OpenReader(dir, quill.ReaderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	c, err := r.Consumer("billing", quill.ConsumerOptions{Protect: quill.ProtectOff})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	out := mustRun(t, "", "prune", "-dir", dir)
	if !strings.Contains(out, "segments") {
		t.Fatalf("prune output %q", out)
	}
}
