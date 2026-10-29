// Example log: open a log, append, flush, read, take a consumer, commit, close, reopen and
// resume where the consumer stopped.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/netstar-labs/quill"
)

func main() {
	dir := filepath.Join(os.TempDir(), "quill-example-log")
	defer os.RemoveAll(dir)

	// Open creates the directory and takes its lock: this process is the one writer.
	l, err := quill.Open(quill.Config{Dir: dir})
	if err != nil {
		log.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		if _, err := l.Append([]byte(fmt.Sprintf("order %d", i))); err != nil {
			log.Fatal(err)
		}
	}
	// Append returns before the records are durable; Flush (or Sync) makes them so.
	if err := l.Flush(); err != nil {
		log.Fatal(err)
	}

	// A consumer is a name plus a committed offset: the next sequence to read.
	c, err := l.Consumer("billing", quill.ConsumerOptions{})
	if err != nil {
		log.Fatal(err)
	}
	recs, err := c.Read(3)
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range recs {
		fmt.Printf("billing read %d: %s\n", r.Seq, r.Payload)
	}
	// Commit once per batch: the next sequence to read.
	if err := c.Commit(recs[len(recs)-1].Seq + 1); err != nil {
		log.Fatal(err)
	}
	_ = c.Close()
	if err := l.Close(); err != nil {
		log.Fatal(err)
	}

	// Reopen: the consumer resumes where it committed.
	l, err = quill.Open(quill.Config{Dir: dir})
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	c, err = l.Consumer("billing", quill.ConsumerOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	for rec, err := range c.Records(context.Background()) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("billing resumed at %d: %s\n", rec.Seq, rec.Payload)
		if err := c.Commit(rec.Seq + 1); err != nil {
			log.Fatal(err)
		}
	}
	lag, _ := c.Lag()
	fmt.Printf("billing is caught up: offset %d, lag %d\n", c.Offset(), lag)
}
