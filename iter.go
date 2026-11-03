package quill

import (
	"context"
	"iter"
)

// Iterators (decision 58). Records reads from offset to the current end and stops; Follow
// does the same and then waits for more until the context ends. Payloads are copies, valid
// after the loop moves on. An error is yielded once and ends the sequence; a *GapError is
// resumed by iterating again from To + 1. Breaking out of the loop releases every buffer and
// descriptor: each batch is read, copied and released before it is yielded, and a batch is bounded
// by MaxReadBytes (each record counts its payload plus 48 bytes of overhead), so a log of empty
// records is read in several batches, never all at once.

// Records yields the records with seq >= offset up to the current head, in order.
func (l *Log) Records(ctx context.Context, offset uint64) iter.Seq2[Record, error] {
	return records(ctx, l, offset, false)
}

// Follow yields the records with seq >= offset and then waits for more, waking on each
// flush, until the context ends. When the log has failed (ErrFailed) it yields the records that
// exist and then the failure, which ends the sequence, instead of waiting for records that
// cannot come.
func (l *Log) Follow(ctx context.Context, offset uint64) iter.Seq2[Record, error] {
	return records(ctx, l, offset, true)
}

// Records yields the records with seq >= offset up to the marker, in order.
func (r *Reader) Records(ctx context.Context, offset uint64) iter.Seq2[Record, error] {
	return records(ctx, r, offset, false)
}

// Follow yields the records with seq >= offset and then polls quill.tail for more until the
// context ends.
func (r *Reader) Follow(ctx context.Context, offset uint64) iter.Seq2[Record, error] {
	return records(ctx, r, offset, true)
}

// Records yields the records from the consumer's committed offset to the current end. The
// offset is read when Records is called, not when the iterator is ranged. It never commits:
// the caller commits explicitly.
func (c *Consumer) Records(ctx context.Context) iter.Seq2[Record, error] {
	return records(ctx, c.src, c.Offset(), false)
}

// Follow yields the records from the consumer's committed offset, read when Follow is called
// and not when the iterator is ranged, and then waits for more until the context ends. It
// never commits. For a consumer taken from a Log that has failed
// (ErrFailed) it yields the failure at the head, as Log.Follow does; a consumer taken from a
// Reader cannot see the writer's failure.
func (c *Consumer) Follow(ctx context.Context) iter.Seq2[Record, error] {
	return records(ctx, c.src, c.Offset(), true)
}

func records(ctx context.Context, src source, offset uint64, follow bool) iter.Seq2[Record, error] {
	return func(yield func(Record, error) bool) {
		cur := cursor{seq: max(offset, 1)}
		for {
			if err := ctx.Err(); err != nil {
				yield(Record{}, err)
				return
			}
			var batch []Record
			next, err := scanSource(src, cur, 0, src.maxReadBytes(), true, false, nil, func(rec Record) bool {
				batch = append(batch, rec)
				return true
			})
			for _, rec := range batch {
				if !yield(rec, nil) {
					return
				}
			}
			if err != nil {
				yield(Record{}, err)
				return
			}
			if len(batch) == 0 {
				if !follow {
					return
				}
				if err := src.wait(ctx, cur.seq); err != nil {
					yield(Record{}, err)
					return
				}
				continue
			}
			cur = next
		}
	}
}
