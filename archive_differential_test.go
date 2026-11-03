package quill

import (
	"errors"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/netstar-labs/quill/internal/fsx"
)

// TestArchiveAgainstTheSourceReaderOverDamagedSegments is the differential for Archive: a log of
// 80 records over small segments, then up to two damages (a flipped byte, a truncation, another
// segment's bytes written over it, a deleted segment) to sealed segments, then archive. Whatever
// the damage, archive must not panic, must fail only with a *GapError or ErrCorrupt, and the
// destination must hold only what the source Reader serves for the same sequence (decisions 28,
// 67, 68, 85: every CRC is verified and an overlap is refused), with no sequence skipped.
func TestArchiveAgainstTheSourceReaderOverDamagedSegments(t *testing.T) {
	const src, dst = "/src", "/dst"
	for seed := int64(0); seed < 300; seed++ {
		rng := rand.New(rand.NewSource(seed))
		sim := fsx.NewSim()
		cfg := simConfig(sim, src)
		cfg.SegmentBytes = 600
		l, err := Open(cfg)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 80; i++ {
			if _, err := l.Append(append([]byte{byte(i)}, make([]byte, rng.Intn(40))...)); err != nil {
				t.Fatal(err)
			}
			if i%7 == 0 {
				mustFlush(t, l)
			}
		}
		mustFlush(t, l)
		if err := l.Roll(); err != nil {
			t.Fatal(err)
		}
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
		damaged := 0
		for k := rng.Intn(3); k > 0; k-- {
			ids := segmentFiles(t, sim, src)
			sealed := ids[:len(ids)-1] // the last is the empty active segment Roll opened
			if len(sealed) == 0 {
				break
			}
			p := filepath.Join(src, segmentName(sealed[rng.Intn(len(sealed))]))
			data := fileBytes(t, sim, p)
			switch rng.Intn(4) {
			case 0:
				data[rng.Intn(len(data))] ^= byte(1 + rng.Intn(255))
			case 1:
				data = data[:rng.Intn(len(data))]
			case 2:
				data = fileBytes(t, sim, filepath.Join(src, segmentName(sealed[rng.Intn(len(sealed))])))
			case 3:
				if err := sim.Remove(p); err != nil {
					t.Fatal(err)
				}
				damaged++
				continue
			}
			writeFile(t, sim, p, data)
			damaged++
		}
		aerr := archive(sim, src, dst)
		var gap *GapError
		if aerr != nil && !errors.As(aerr, &gap) && !errors.Is(aerr, ErrCorrupt) {
			t.Fatalf("seed %d: archive failed with %v, want a *GapError or ErrCorrupt", seed, aerr)
		}
		if damaged == 0 && aerr != nil {
			t.Fatalf("seed %d: archive of an undamaged log failed: %v", seed, aerr)
		}
		dr, err := OpenReader(dst, ReaderOptions{fs: sim})
		if err != nil {
			continue // nothing was copied
		}
		sr, err := OpenReader(src, ReaderOptions{fs: sim})
		if err != nil {
			t.Fatal(err)
		}
		var got []Record
		_ = dr.ScanAll(func(r Record) bool {
			got = append(got, Record{Seq: r.Seq, At: r.At, Payload: append([]byte(nil), r.Payload...)})
			return true
		})
		for i, g := range got {
			if i > 0 && g.Seq != got[i-1].Seq+1 {
				t.Fatalf("seed %d: the destination skipped sequence %d to %d", seed, got[i-1].Seq, g.Seq)
			}
			recs, rerr := sr.Read(g.Seq, 1)
			if rerr != nil || len(recs) != 1 || !recEqual(recs[0], g) {
				t.Fatalf("seed %d: the destination serves %+v for sequence %d; the source serves %+v, %v", seed, g, g.Seq, recs, rerr)
			}
		}
		if damaged == 0 && len(got) != 80 {
			t.Fatalf("seed %d: an undamaged log archived %d records, want 80", seed, len(got))
		}
		dr.Close()
		sr.Close()
	}
}
