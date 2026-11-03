package quill

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/netstar-labs/quill/internal/fsx"
)

// The read path as another process uses it (guarantees 1, 2 and 6): the window and walk with the
// chunk sizes a Reader starts with, and a Reader over a whole directory whose files and marker
// are whatever a damaged or hostile writer left. FuzzRecoverSegment reads one whole file from
// its start with the largest chunk; these two start mid-file, with chunks of every size, and
// go through Reader.view, segment and scanSource.

// walked is what one walk delivered: every record (payload copied) with its position, and how
// it ended.
type walked struct {
	recs []Record
	pos  []int64
	info segInfo
	err  error
}

func walkCollect(w *window, start int64, prev uint64, havePrev bool) walked {
	var out walked
	out.info, out.err = walk(w, start, prev, havePrev, func(rec Record, pos int64) bool {
		rec.Payload = append([]byte(nil), rec.Payload...)
		out.recs = append(out.recs, rec)
		out.pos = append(out.pos, pos)
		return true
	})
	return out
}

func sameWalk(t *testing.T, what string, got, want walked) {
	t.Helper()
	if got.err != want.err {
		t.Fatalf("%s: error %v, want %v", what, got.err, want.err)
	}
	if len(got.recs) != len(want.recs) {
		t.Fatalf("%s: %d records, want %d", what, len(got.recs), len(want.recs))
	}
	for i := range got.recs {
		if !recEqual(got.recs[i], want.recs[i]) || got.pos[i] != want.pos[i] {
			t.Fatalf("%s: record %d is %+v at %d, want %+v at %d", what, i, got.recs[i], got.pos[i], want.recs[i], want.pos[i])
		}
	}
	if got.info.end != want.info.end || got.info.frames != want.info.frames ||
		!recEqual(got.info.first, want.info.first) || !recEqual(got.info.last, want.info.last) {
		t.Fatalf("%s: info %+v, want %+v", what, got.info, want.info)
	}
}

func sameIndex(a, b []anchor) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func FuzzWalkChunks(f *testing.F) {
	f.Add(image(1, 5, 0, smallPayload), uint32(0), false)
	f.Add(image(1, 40, 7, smallPayload), uint32(3), true)
	f.Add(append(image(1, 5, 0, smallPayload), image(9, 3, 0, smallPayload)...), uint32(1), false) // a sequence jump
	f.Add(append(image(1, 5, 0, smallPayload), 0x80, 0x01), uint32(65535), false)                  // a torn tail
	// Several indexStride anchors: the resume points of a Reader's read.
	f.Add(image(1, 3*indexStride/1000+8, 100, func(i int) []byte { return bytes.Repeat([]byte{byte(i)}, 1000) }), uint32(4095), false)
	f.Add([]byte{0x01, 0x00, 0x00, 0x00, 0x00}, uint32(0), true)
	f.Fuzz(func(t *testing.T, data []byte, chunkSeed uint32, noIndex bool) {
		size := int64(len(data))
		chunk := 1 + int64(chunkSeed%65536)
		newWindow := func(chunk int64, noIndex bool) *window {
			return &window{r: bytes.NewReader(data), size: size, chunk: chunk, noIndex: noIndex}
		}
		// The reference: the whole file from the start, with the largest chunk, indexed.
		ref := walkCollect(newWindow(windowChunk, false), 0, 0, false)
		for i, rec := range ref.recs {
			frame := appendFrame(nil, &rec)
			if !bytes.HasPrefix(data[ref.pos[i]:], frame) {
				t.Fatalf("record %d accepted but not canonical at %d", i, ref.pos[i])
			}
			if i > 0 && rec.Seq != ref.recs[i-1].Seq+1 {
				t.Fatalf("record %d is seq %d after %d", i, rec.Seq, ref.recs[i-1].Seq)
			}
			if rec.Seq == 0 {
				t.Fatalf("record %d has sequence 0", i)
			}
		}
		// Any chunk size, with or without the index, delivers the same records and ends the same way.
		got := walkCollect(newWindow(chunk, noIndex), 0, 0, false)
		sameWalk(t, fmt.Sprintf("chunk %d noIndex %v", chunk, noIndex), got, ref)
		if noIndex && got.info.index != nil {
			t.Fatalf("noIndex but the index is %v", got.info.index)
		}
		if !noIndex && !sameIndex(got.info.index, ref.info.index) {
			t.Fatalf("chunk %d: index %v, want %v", chunk, got.info.index, ref.info.index)
		}
		// A read resumes at an anchor, or at the position a previous read ended, knowing nothing
		// before it: it must deliver exactly the suffix of the reference walk.
		var starts []int64
		for _, a := range ref.info.index {
			starts = append(starts, a.pos)
		}
		step := len(ref.pos)/16 + 1
		for i := 0; i < len(ref.pos); i += step {
			starts = append(starts, ref.pos[i])
		}
		for _, start := range starts {
			k := 0
			for ref.pos[k] != start {
				k++
			}
			want := walked{recs: ref.recs[k:], pos: ref.pos[k:], err: ref.err, info: segInfo{
				end: ref.info.end, frames: len(ref.recs) - k, first: Record{Seq: ref.recs[k].Seq, At: ref.recs[k].At}, last: ref.info.last,
			}}
			sameWalk(t, fmt.Sprintf("resume at %d chunk %d", start, chunk), walkCollect(newWindow(chunk, true), start, 0, false), want)
		}
	})
}

// readerDirSeed builds a directory with a real Log, one record at a time until the log has two
// sealed segments and an active one holding a record, and returns the three segment files and
// the marker's numbers.
func readerDirSeed(f *testing.F, payload func(i int) []byte, segBytes int64) (segs [3][]byte, seq, off uint64) {
	s := fsx.NewSim()
	cfg := simConfig(s, "/log")
	cfg.SegmentBytes, cfg.NoMaintenance = segBytes, true
	l, err := Open(cfg)
	if err != nil {
		f.Fatal(err)
	}
	for i := 0; ; i++ {
		if _, err := l.Append(payload(i)); err != nil {
			f.Fatal(err)
		}
		if err := l.Flush(); err != nil {
			f.Fatal(err)
		}
		if ids, err := listSegments(s, "/log"); err != nil || len(ids) > 3 {
			f.Fatalf("segments %v %v", ids, err)
		} else if len(ids) == 3 {
			break
		}
	}
	if err := l.Close(); err != nil {
		f.Fatal(err)
	}
	ids, _ := listSegments(s, "/log")
	for i, id := range ids {
		segs[i] = readFileOrFatal(f, s, "/log/"+segmentName(id))
	}
	tl, err := decodeTail(readFileOrFatal(f, s, "/log/"+tailName))
	if err != nil {
		f.Fatal(err)
	}
	if tl.Seg != 2 || ids[0] != 0 || ids[1] != 1 || ids[2] != 2 {
		f.Fatalf("segment ids %v, marker %+v", ids, tl)
	}
	return segs, tl.Seq, tl.Off
}

func readFileOrFatal(f *testing.F, fsys fsx.FS, path string) []byte {
	file, err := fsys.OpenFile(path, fsx.O_RDONLY, 0)
	if err != nil {
		f.Fatal(err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		f.Fatal(err)
	}
	buf := make([]byte, info.Size())
	if _, err := file.ReadAt(buf, 0); err != nil && len(buf) > 0 {
		f.Fatal(err)
	}
	return buf
}

// FuzzReaderDir opens a Reader on a directory of three segment files (ids 0 to 2) and a marker
// (segment 2 active) with fuzzed contents, and checks what every read of it may and may not do:
// never a record above the marker's sequence, never sequence 0, never a jump in sequence without
// an error, never a record from Read that the scan of the same directory delivered differently.
func FuzzReaderDir(f *testing.F) {
	// A directory a real writer made: small records, so each sealed segment is a few frames.
	segs, seq, off := readerDirSeed(f, func(i int) []byte { return payloadFor(uint64(i + 1)) }, 400)
	f.Add(segs[0], segs[1], segs[2], seq, off)
	f.Add(segs[0], segs[1], segs[2], seq-1, off) // a marker below what the files hold
	f.Add(segs[0], segs[1], segs[2], seq+5, off) // and above it
	f.Add(segs[0], segs[1], segs[2], seq, off-1) // an offset inside the last frame
	// The same with segments past several sparse-index strides: a read resumes at an anchor.
	segs, seq, off = readerDirSeed(f, func(i int) []byte { return bytes.Repeat([]byte{byte(i)}, 1000) }, 3*indexStride)
	f.Add(segs[0], segs[1], segs[2], seq, off)
	f.Fuzz(func(t *testing.T, a, b, c []byte, seq, off uint64) {
		// A marker the decoder refuses is FuzzDecodeTail's business, and a reader retries it
		// five times, which would only slow this target.
		if seq == ^uint64(0) {
			seq--
		}
		off %= maxSegmentFileBytes + 1
		s := fsx.NewSim()
		if err := s.Mkdir("/log", 0o700); err != nil {
			t.Fatal(err)
		}
		writeFile(t, s, "/log/"+formatName, encodeFormat(testID))
		for i, data := range [][]byte{a, b, c} {
			writeFile(t, s, "/log/"+segmentName(uint64(i)), data)
		}
		writeFile(t, s, "/log/"+tailName, encodeTail(tail{Seq: seq, Seg: 2, Off: off, MaxSeg: 2}))
		r, err := OpenReader("/log", ReaderOptions{fs: s, PollInterval: time.Millisecond})
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		defer r.Close()
		if head, err := r.Head(); err != nil || head != seq {
			t.Fatalf("head %d %v, marker says %d", head, err, seq)
		}

		// checkRun: records ascend by one, none is 0 or above the marker.
		checkRun := func(what string, recs []Record) {
			t.Helper()
			for i, rec := range recs {
				if rec.Seq == 0 || rec.Seq > seq {
					t.Fatalf("%s: record %d has seq %d, marker %d", what, i, rec.Seq, seq)
				}
				if i > 0 && rec.Seq != recs[i-1].Seq+1 {
					t.Fatalf("%s: seq %d follows %d", what, rec.Seq, recs[i-1].Seq)
				}
			}
		}
		scanned, _ := readerAll(r)
		checkRun("ScanAll", scanned)
		byseq := make(map[uint64]Record, len(scanned))
		for _, rec := range scanned {
			byseq[rec.Seq] = rec
		}
		var viaIter []Record
		for rec, err := range r.Records(context.Background(), 0) {
			if err != nil {
				break
			}
			rec.Payload = append([]byte(nil), rec.Payload...)
			viaIter = append(viaIter, rec)
		}
		checkRun("Records", viaIter)

		// Read from the offsets around what the scan saw, a few records at a time.
		offsets := []uint64{0, 1, seq, seq + 1}
		if len(scanned) > 0 {
			step := len(scanned)/24 + 1
			for i := 0; i < len(scanned); i += step {
				offsets = append(offsets, scanned[i].Seq-1, scanned[i].Seq, scanned[i].Seq+1)
			}
		}
		// The sequences either side of every sparse-index anchor: where a read resumes mid-file.
		for _, data := range [][]byte{a, b, c} {
			w := &window{r: bytes.NewReader(data), size: int64(len(data)), chunk: windowChunk}
			info, _ := walk(w, 0, 0, false, nil)
			for _, an := range info.index {
				offsets = append(offsets, an.seq-1, an.seq, an.seq+1)
			}
		}
		for _, offset := range offsets {
			recs, _ := r.Read(offset, 3)
			if _, seen := byseq[offset]; seen && len(recs) > 0 && recs[0].Seq != offset {
				t.Fatalf("Read(%d) starts at seq %d, though the scan delivered %d", offset, recs[0].Seq, offset)
			}
			checkRun(fmt.Sprintf("Read(%d)", offset), recs)
			if len(recs) > 0 && recs[0].Seq < max(offset, 1) {
				t.Fatalf("Read(%d) starts at seq %d", offset, recs[0].Seq)
			}
			for _, rec := range recs {
				if prior, ok := byseq[rec.Seq]; ok && !recEqual(prior, rec) {
					t.Fatalf("Read(%d) gave %+v for seq %d, the scan %+v", offset, rec, rec.Seq, prior)
				}
			}
		}

		if st, err := r.Stat(); err == nil && st.Head != seq {
			t.Fatalf("stat head %d, marker %d", st.Head, seq)
		}
		_, _ = r.Check() // no panic, no hang
		for _, at := range []int64{0, 1, 1_700_000_000, 1 << 40} {
			if o, err := r.OffsetForTime(time.Unix(at, 0)); err == nil && (o == 0 || o > seq+1) {
				t.Fatalf("OffsetForTime(%d) = %d, marker %d", at, o, seq)
			}
		}
	})
}
