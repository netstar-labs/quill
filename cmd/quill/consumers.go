package main

import (
	"errors"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/netstar-labs/quill"
)

func cmdConsumers(c *cli, args []string) int {
	return c.withReader(func(r *quill.Reader) int {
		list, err := r.Consumers()
		if err != nil {
			return c.errorf("%v", err)
		}
		head, err := r.Head()
		if err != nil {
			return c.errorf("%v", err)
		}
		if c.json {
			for _, ci := range list {
				row := map[string]any{"name": ci.Name, "offset": ci.Offset, "protected": ci.Protected, "lag": lag(head, ci.Offset)}
				if ci.Err != nil {
					row["error"] = ci.Err.Error()
				}
				if code := printJSON(c, row); code != exitOK {
					return code
				}
			}
			return exitOK
		}
		w := tabwriter.NewWriter(c.out, 0, 8, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tOFFSET\tLAG\tPROTECTED\tNOTE")
		for _, ci := range list {
			note := ""
			if ci.Err != nil {
				note = "error: " + ci.Err.Error()
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%v\t%s\n", ci.Name, ci.Offset, lag(head, ci.Offset), ci.Protected, note)
		}
		_ = w.Flush()
		return exitOK
	})
}

func lag(head, offset uint64) uint64 {
	if offset > head+1 {
		return 0
	}
	return head + 1 - offset
}

func (c *cli) consumer(r *quill.Reader, opts quill.ConsumerOptions) (*quill.Consumer, int) {
	if c.name == "" {
		return nil, c.usage("-name is required")
	}
	cons, err := r.Consumer(c.name, opts)
	if err != nil {
		return nil, c.errorf("%v", err)
	}
	return cons, exitOK
}

func cmdCommit(c *cli, args []string) int {
	if (c.offset == "") == (c.at == "") {
		return c.usage("commit: exactly one of -offset and -time")
	}
	// Parse before opening anything: a malformed number must not register the consumer.
	var at time.Time
	var next uint64
	if c.at != "" {
		t, err := parseTime(c.at, time.Now())
		if err != nil {
			return c.usage("-time: %v", err)
		}
		at = t
	} else {
		n, err := parseDecimal(c.offset)
		if err != nil {
			return c.usage("-offset: %q is not a plain non-negative integer", c.offset)
		}
		next = n
	}
	return c.withReader(func(r *quill.Reader) int {
		cons, code := c.consumer(r, quill.ConsumerOptions{})
		if code != exitOK {
			return code
		}
		defer cons.Close()
		if c.at != "" {
			if err := cons.SeekToTime(at); err != nil {
				return c.errorf("%v", err)
			}
		} else if err := cons.Commit(next); err != nil {
			return c.errorf("%v", err)
		}
		fmt.Fprintf(c.out, "%s %d\n", c.name, cons.Offset())
		return exitOK
	})
}

func cmdRegister(c *cli, args []string) int {
	if c.protect && c.noProt {
		return c.usage("register: -protect and -no-protect exclude each other")
	}
	return c.withReader(func(r *quill.Reader) int {
		opts := quill.ConsumerOptions{}
		switch {
		case c.protect:
			opts.Protect = quill.ProtectOn
		case c.noProt:
			opts.Protect = quill.ProtectOff
		}
		cons, code := c.consumer(r, opts)
		if code != exitOK {
			return code
		}
		defer cons.Close()
		fmt.Fprintf(c.out, "%s %d protected=%v\n", c.name, cons.Offset(), cons.Protected())
		return exitOK
	})
}

func cmdUnregister(c *cli, args []string) int {
	if c.name == "" {
		return c.usage("-name is required")
	}
	return c.withReader(func(r *quill.Reader) int {
		if err := r.DeleteConsumer(c.name); err != nil {
			return c.errorf("%v", err)
		}
		return exitOK
	})
}

func cmdArchive(c *cli, args []string) int {
	if c.to == "" {
		return c.usage("archive: -to is required")
	}
	err := quill.Archive(c.dir, c.to)
	var gap *quill.GapError
	if errors.As(err, &gap) {
		fmt.Fprintf(c.err, "quill: the source lost records %d to %d; the archive holds the same hole\n", gap.From, gap.To)
		return exitOK
	}
	if err != nil {
		return c.errorf("%v", err)
	}
	return exitOK
}
