// Command powertest is the real power-loss test (docs/audits/power-loss.md): on a machine
// that can be crashed without syncing (`echo b > /proc/sysrq-trigger`), `run` writes a log
// and acknowledges each flushed batch in a separate fsynced file until the machine dies;
// after the reboot `verify` opens the log and checks that every acknowledged record survived,
// that the log is a contiguous prefix with no holes, and that Check is healthy. It also
// records whether the marker was stale at the crash, which is the case decision 39 exists for.
//
//	powertest -dir DIR -acks FILE run
//	powertest -dir DIR -acks FILE verify
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/netstar-labs/quill"
)

func main() {
	dir := flag.String("dir", "", "the log directory, on the filesystem under test")
	acks := flag.String("acks", "", "the acknowledgement file, appended and fsynced after every flush")
	flag.Parse()
	if *dir == "" || *acks == "" || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	switch flag.Arg(0) {
	case "run":
		run(*dir, *acks)
	case "verify":
		verify(*dir, *acks)
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func payload(seq uint64) []byte {
	return []byte(fmt.Sprintf("power-%d-%s", seq, strings.Repeat("p", int(seq%61))))
}

// run appends in batches: a batch is flushed and acknowledged, then a few more records are
// appended and left in the buffer, so the crash always finds unflushed records too. Segments
// are 1 MiB, so rotations (which fsync the directory) happen regularly while ordinary
// flushes leave the marker's rename unsynced.
func run(dir, acks string) {
	l, err := quill.Open(quill.Config{Dir: dir, Retention: -1, SegmentBytes: 1 << 20, FlushInterval: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	a, err := os.OpenFile(acks, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(a, "start head=%d\n", l.Head())
	_ = a.Sync()
	next := l.Head() + 1
	for {
		for i := 0; i < 20; i++ {
			if _, err := l.Append(payload(next)); err != nil {
				log.Fatal(err)
			}
			next++
		}
		if err := l.Flush(); err != nil {
			log.Fatal(err)
		}
		if _, err := fmt.Fprintf(a, "ack %d\n", next-1); err != nil {
			log.Fatal(err)
		}
		if err := a.Sync(); err != nil {
			log.Fatal(err)
		}
		for i := 0; i < 7; i++ { // unflushed at the crash, unless a later flush takes them
			if _, err := l.Append(payload(next)); err != nil {
				log.Fatal(err)
			}
			next++
		}
	}
}

func verify(dir, acks string) {
	f, err := os.Open(acks)
	if err != nil {
		log.Fatal(err)
	}
	var acked, startHead uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "ack ") {
			if n, err := strconv.ParseUint(line[4:], 10, 64); err == nil {
				acked = n
			}
		}
		if strings.HasPrefix(line, "start head=") {
			if n, err := strconv.ParseUint(line[11:], 10, 64); err == nil {
				startHead = n
			}
		}
	}
	_ = f.Close()
	// What a reader sees before recovery: the marker as the crash left it.
	r, err := quill.OpenReader(dir, quill.ReaderOptions{})
	if err != nil {
		log.Fatalf("reader before recovery: %v", err)
	}
	markerHead, _ := r.Head()
	_ = r.Close()
	l, err := quill.Open(quill.Config{Dir: dir, Retention: -1, FlushInterval: time.Hour})
	if err != nil {
		log.Fatalf("FAIL: Open after the crash: %v", err)
	}
	defer l.Close()
	var last uint64
	n := 0
	err = l.ScanAll(func(rec quill.Record) bool {
		n++
		// Retention is -1, so nothing is deleted legitimately: the log starts at 1. A lost
		// oldest segment is neither a gap between scanned records nor a head below acked.
		if last == 0 && rec.Seq != 1 {
			log.Fatalf("FAIL: log starts at %d, not 1", rec.Seq)
		}
		if last != 0 && rec.Seq != last+1 {
			log.Fatalf("FAIL: not contiguous: %d after %d", rec.Seq, last)
		}
		if string(rec.Payload) != string(payload(rec.Seq)) {
			log.Fatalf("FAIL: record %d holds %q", rec.Seq, rec.Payload)
		}
		last = rec.Seq
		return true
	})
	if err != nil {
		log.Fatalf("FAIL: scan: %v", err)
	}
	st, _ := l.Stat()
	if len(st.Holes) != 0 {
		log.Fatalf("FAIL: holes %+v", st.Holes)
	}
	if last < acked {
		log.Fatalf("FAIL: acknowledged %d, recovered %d", acked, last)
	}
	rr, _ := quill.OpenReader(dir, quill.ReaderOptions{})
	res, err := rr.Check()
	_ = rr.Close()
	if err != nil || !res.Healthy {
		log.Fatalf("FAIL: check: %+v %v", res, err)
	}
	fmt.Printf("OK: start head %d, acknowledged %d, marker at crash %d (stale by %d), recovered %d records to %d, %d segments, check healthy\n",
		startHead, acked, markerHead, last-markerHead, n, last, st.Segments)
}
