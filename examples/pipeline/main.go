// Example pipeline: two processes. The writer opens the log and appends continuously; the
// reader starts late, opens a Reader (no lock) and a consumer, catches up and keeps
// following. Run each role in its own terminal with the same -dir, or let the test do it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/netstar-labs/quill"
)

func main() {
	dir := flag.String("dir", "", "the log directory, shared by both roles")
	role := flag.String("role", "", "writer or reader")
	count := flag.Int("count", 50, "writer: records to append; reader: records to read before stopping (0: forever)")
	flag.Parse()
	if *dir == "" || (*role != "writer" && *role != "reader") {
		flag.Usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var err error
	if *role == "writer" {
		err = writer(ctx, *dir, *count)
	} else {
		err = reader(ctx, *dir, *count)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// writer appends count records, making each durable before acknowledging it.
func writer(ctx context.Context, dir string, count int) error {
	l, err := quill.Open(quill.Config{Dir: dir, SyncAppends: true})
	if err != nil {
		return err
	}
	defer l.Close()
	for i := 1; i <= count && ctx.Err() == nil; i++ {
		seq, err := l.Append([]byte(fmt.Sprintf("event %d", i)))
		if err != nil {
			return err
		}
		fmt.Printf("writer: %d durable\n", seq)
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// reader waits for the log to exist, then follows it from its consumer's offset.
func reader(ctx context.Context, dir string, count int) error {
	var r *quill.Reader
	for {
		var err error
		r, err = quill.OpenReader(dir, quill.ReaderOptions{})
		if err == nil {
			break
		}
		if !errors.Is(err, quill.ErrNoLog) {
			return err
		}
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	defer r.Close()
	c, err := r.Consumer("pipeline", quill.ConsumerOptions{})
	if err != nil {
		return err
	}
	defer c.Close()
	n := 0
	for rec, err := range c.Follow(ctx) {
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		fmt.Printf("reader: %d %s\n", rec.Seq, rec.Payload)
		if err := c.Commit(rec.Seq + 1); err != nil {
			return err
		}
		n++
		if count > 0 && n >= count {
			return nil
		}
	}
	return nil
}
