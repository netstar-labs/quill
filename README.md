# quill

An embeddable, append-only log for Go on the standard library alone. **A log is a directory**:
one assigned writer, any number of consumers, in the writer's process or another. No broker, no
sockets, no dependencies. Closer to "the SQLite of commit logs" than to a lightweight Kafka.

```
                 Append / AppendBatch              Read / Scan / Records / Follow
   producer ──────────────────────► Log ◄──────────────────────── in-process consumer
                                     │  Flush: write, fsync,
                                     │  publish quill.tail
                                     ▼
            orders/                          one log = one directory
            ├── 0000000000.log   sealed      CRC-framed segments, ids never reused
            ├── 0000000001.log   active
            ├── quill.tail       the durable position (seq, seg, off, maxseg)
            ├── quill.format     "quill/1 <log id>"
            ├── quill.policy     the writer's Config, for tools
            ├── quill.lock       the writer's flock
            └── consumers/       <name>: "<log id> <next> <protect>", one file per consumer
                                     ▲
                                     │  reads up to quill.tail, never past it; no lock
   another process ────────────── OpenReader ──► Reader / Consumer(name) ─── Commit(next)
```

## Documents

- **Design** (read first): [scope](docs/design/scope.md) (what is built and the guarantees),
  [review](docs/design/review.md) (what the reference design does and what is left behind),
  [decisions](docs/design/decisions.md) (every choice, with its reason), and the
  [build prompt](docs/design/build-prompt.md) (how the build runs, phase by phase).
- **Format**: [docs/format.md](docs/format.md), the normative on-disk format, frozen by golden
  tests.
- **Audits**: [docs/audits/](docs/audits/), what the tests cannot show and why, the test ledger,
  and the benchmark baselines.
- Written in phase 4 of the build: `docs/introduction.md`, `docs/executive-summary.md`,
  `docs/architecture.md`, `docs/userguide.md`, the generated `docs/reference.md`, and
  `examples/`.

## Layout

The library is the root package, so the import path is `github.com/netstar-labs/quill`.
No file is over about 600 lines. All file I/O in the library goes through `internal/fsx`.

| Path | Purpose |
|---|---|
| `doc.go` | Package comment: the rules to know |
| `codec.go` | The frame: encode, decode, CRC |
| `load.go` | Segment verification and parse-forward recovery; in phase 2, the directory scan and hole detection |
| `state.go` | `quill.tail`, `quill.format`, `quill.policy` and consumer files: encode, strict decode, atomic write |
| `internal/fsx/` | The file-system seam: the interface, the real implementation, the crash simulator |
| `internal/atomicfile/` | Temp file, fsync, rename, optional directory fsync, through `fsx` |
| `internal/strictjson/` | State-file parsing that refuses damaged values and unknown fields |
| `testdata/` | Golden fixtures that freeze the format |
| `hack/gate.sh` | The commit gate: gofmt, vet, build, `test -race`, staticcheck, deadcode, fuzz, tidy |
| `.github/workflows/` | CI on Linux and macOS, the same gate as separate steps, and the PR-issue-link check |

Phases 2 to 4 add `log.go`, `append.go`, `flush.go`, `read.go`, `reader.go`, `iter.go`,
`seek.go`, `stat.go`, `consumer.go`, `archive.go`, `retention.go`, `disk.go`, `lock_unix.go`,
`cmd/quill/`, `examples/` and `build/quill`.

## Building

Standard library only; `go.mod` has no `require` line. Go is the toolchain named in `go.mod`
(the iterator API needs Go 1.23 or later). Always run Go with `GOWORK=off`:

```
GOWORK=off go test -race ./...
hack/gate.sh            # the commit gate; FUZZTIME=0 skips the fuzz step (and says so)
```

Linux and darwin only (arm64, amd64); other platforms fail to build on purpose.

## Status

Private, pre-release, under construction by the phased build in the build prompt. Tags are
`v0.y.z` for ever: breaking changes bump the minor with a migration note. No `LICENSE` while
private; one lands before the repository goes public.
