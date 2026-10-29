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
- **Audits**: [docs/audits/](docs/audits/): the test ledger, what the tests cannot show and why,
  the benchmark baselines, the real power-loss runs, and the self-review before the audit.
- **Start here**: [introduction](docs/introduction.md), [executive summary](docs/executive-summary.md).
- **Deep dive**: [architecture](docs/architecture.md).
- **Operations**: [user guide](docs/userguide.md), the generated [command reference](docs/reference.md).
- **Examples**: [examples/](examples/README.md).
- [CONTRIBUTING](CONTRIBUTING.md), [CHANGELOG](CHANGELOG.md), [SECURITY](SECURITY.md).

## Layout

The library is the root package, so the import path is `github.com/netstar-labs/quill`.
No file is over about 600 lines. All file I/O in the library goes through `internal/fsx`.

| Path | Purpose |
|---|---|
| `doc.go` | Package comment: the rules to know |
| `errors.go` | The exported errors |
| `codec.go` | The frame: encode, decode, CRC |
| `log.go` | `Log`, `Config`, defaults, validation, `Open` (the creation protocol), `Close` |
| `load.go` | Segment verification, parse-forward recovery, the directory scan, hole detection |
| `append.go` | `Append`, `AppendBatch`, the buffer and its bounds |
| `flush.go` | `Flush`, the flush goroutine, the failure classes, rotation, `Roll`, `Sync` |
| `read.go` | The read path shared by `Log`, `Reader` and `Consumer`: `Read`, `ReadWait`, `Scan`, `ScanAll`, segment ranges, gaps |
| `reader.go` | `OpenReader`: lock-free, bounded by `quill.tail`, a lazily built per-sealed-segment index, polling |
| `iter.go` | `Records` and `Follow` (`iter.Seq2`) for `Log`, `Reader` and `Consumer` |
| `seek.go` | `OffsetForTime` and `SeekToTime` |
| `stat.go` | `Log.Stat`, `Reader.Stat`, `Reader.Check` |
| `consumer.go` | Consumer handles: lifecycle, the per-consumer lock, the offset file, `Commit`, `Lag`, protection |
| `archive.go` | `Archive(srcDir, dstDir)`: the `_archive` consumer behind `quill archive` |
| `retention.go` | The prune pass: age sweep, protection, the clock guard, the caps, `ConsumerTTL`, `Dropped` |
| `disk.go` | Free-space reading with `statfs` |
| `lock_unix.go` | Taking `quill.lock` and `consumers/*.lock` |
| `state.go` | `quill.tail`, `quill.format`, `quill.policy` and consumer files: encode, strict decode, atomic write |
| `internal/fsx/` | The file-system seam: the interface, the real implementation, the crash simulator |
| `internal/atomicfile/` | Temp file, fsync, rename, optional directory fsync, through `fsx` |
| `internal/strictjson/` | State-file parsing that refuses damaged values and unknown fields |
| `testdata/` | Golden fixtures that freeze the format |
| `hack/gate.sh`, `hack/fuzz.sh` | The commit gate: gofmt, vet, build, `test -race`, staticcheck, deadcode, every fuzz target for 20 s, tidy |
| `cmd/quill/` | The command: `main.go`, `verbs.go` (the verb table drives help and the reference), `print.go` |
| `examples/` | `log/`, `topics/`, `pipeline/` |
| `build/quill` | The release build: version stamping, cross-compilation, archives, checksums, SBOM |
| `.github/workflows/` | CI on Linux and macOS, the same gate as separate steps, the full-disk job, and the PR-issue-link check |


## Building and the Go version

Standard library only; `go.mod` has no `require` line. **Go-version policy:** the toolchain
named in `go.mod` is the one supported version; quill tracks the latest stable Go release and
moves to the next one deliberately, together with the tool pins in `hack/gate.sh` and CI. The
iterator API needs Go 1.23 or later. Always run Go with `GOWORK=off`:

```
GOWORK=off go test -race ./...
hack/gate.sh            # the commit gate; FUZZTIME=0 skips the fuzz step (and says so)
```

Linux and darwin only (arm64, amd64); other platforms fail to build on purpose.

## Status

Private, pre-release, under construction by the phased build in the build prompt. Tags are
`v0.y.z` for ever: breaking changes bump the minor with a migration note. No `LICENSE` while
private; one lands before the repository goes public.
