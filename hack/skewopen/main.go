// Command skewopen opens a log writable with the given settings and closes it again. It is
// the one way, short of writing Go, to make a writer rewrite quill.policy and re-protect
// consumers, which no CLI verb does (decision 99). With -stream it also appends each line of
// stdin as its own record, so a writer killed part way leaves a prefix of the lines on disk
// (the CLI appends one batch, written whole). hack/skew.sh copies it into each ref it tests
// and builds it there, so it may use only the API that every such ref has.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/netstar-labs/quill"
)

func main() {
	dir := flag.String("dir", "", "the log directory")
	retention := flag.Duration("retention", 36*time.Hour, "Config.Retention (negative: never)")
	maxBytes := flag.Int64("max-bytes", 0, "Config.MaxBytes")
	maxRecords := flag.Int64("max-records", 0, "Config.MaxRecords")
	protect := flag.String("protect", "", "Config.ProtectedConsumers, comma separated")
	stream := flag.Bool("stream", false, "append each line of stdin as one record, unsynced")
	flag.Parse()
	cfg := quill.Config{Dir: *dir, Retention: *retention, MaxBytes: *maxBytes, MaxRecords: *maxRecords}
	if *protect != "" {
		cfg.ProtectedConsumers = strings.Split(*protect, ",")
	}
	l, err := quill.Open(cfg)
	if err == nil {
		sc := bufio.NewScanner(os.Stdin)
		for *stream && err == nil && sc.Scan() {
			_, err = l.Append(sc.Bytes())
		}
		if cerr := l.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "skewopen:", err)
		os.Exit(1)
	}
}
