// Example topics: three logs in one process, each a directory with its own lock, buffer,
// flush goroutine and consumers. Topics share no lock, buffer or flush goroutine, so
// one log's flush never waits on another's; on one disk they still share the device.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/netstar-labs/quill"
)

func main() {
	root := filepath.Join(os.TempDir(), "quill-example-topics")
	defer os.RemoveAll(root)

	topics := map[string]*quill.Log{}
	for _, name := range []string{"orders", "payments", "audit"} {
		l, err := quill.Open(quill.Config{Dir: filepath.Join(root, name)})
		if err != nil {
			log.Fatal(err)
		}
		defer l.Close()
		topics[name] = l
	}

	// Producers write to their own topic concurrently; each topic flushes on its own.
	var wg sync.WaitGroup
	for name, l := range topics {
		wg.Add(1)
		go func(name string, l *quill.Log) {
			defer wg.Done()
			for i := 1; i <= 3; i++ {
				if _, err := l.Append([]byte(fmt.Sprintf("%s event %d", name, i))); err != nil {
					log.Fatal(err)
				}
			}
			if err := l.Flush(); err != nil {
				log.Fatal(err)
			}
		}(name, l)
	}
	wg.Wait()

	// Each topic has its own consumers; a name in one topic says nothing about another.
	for name, l := range topics {
		c, err := l.Consumer("reporting", quill.ConsumerOptions{})
		if err != nil {
			log.Fatal(err)
		}
		recs, err := c.Read(10)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: %d records, first %q, head %d\n", name, len(recs), recs[0].Payload, l.Head())
		if err := c.Commit(recs[len(recs)-1].Seq + 1); err != nil {
			log.Fatal(err)
		}
		_ = c.Close()
	}
}
