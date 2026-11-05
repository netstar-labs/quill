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
            ├── quill.policy     the last writer's Config, for tools; offline verbs never rewrite it
            ├── quill.lock       the writer's flock
            └── consumers/       <name>: "<log id> <next> <protect>", one file per consumer
                                     ▲
                                     │  reads up to quill.tail, never past it; no lock
   another process ────────────── OpenReader ──► Reader / Consumer(name) ─── Commit(next)
```

**Where the log lives.** The directory must be on a local filesystem (ext4, xfs, btrfs, APFS) with one host or kernel
writing to it. The single-writer lock is a `flock`, and a reader's view rests on `rename` and `fsync`; NFS, SMB, FUSE
filesystems (sshfs, s3fs and the like) and a directory shared between VMs or containers with separate kernels (virtiofs,
9p) do not provide them coherently. Quill does not detect these: two writers there can both believe they hold the lock
and corrupt the log with no error at write time. See the user guide, "Filesystems".

## Documents

- **Design** (read first): [scope](docs/design/scope.md) (what is built and the guarantees),
  [review](docs/design/review.md) (what the reference design does and what is left behind),
  [decisions](docs/design/decisions.md) (every choice, with its reason), and the
  [build prompt](docs/design/build-prompt.md) (how the build runs, phase by phase).
- **Format**: [docs/format.md](docs/format.md), the normative on-disk format, frozen by golden
  tests.
- **Audits**: [docs/audits/](docs/audits/): the test ledger, what the tests cannot show and why,
  the benchmark baselines, the real power-loss runs, the self-review before the audit, and the audit reports
  and their findings.
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
| `config.go` | `Config`, its defaults and validation, consumer-name rules |
| `log.go` | `Log`, `Open` (the creation protocol), `Close` |
| `load.go` | Segment verification, parse-forward recovery, the directory scan |
| `holes.go` | `contiguity`: turns the ordered good segments into holes and the first overlap |
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
| `hack/gate.sh`, `hack/fuzz.sh` | The commit gate: no-require check, gofmt, vet, build, the cross-platform vet (linux and darwin), the unsupported-platform and 32-bit checks, `test -race`, staticcheck, deadcode, every fuzz target for 20 s, tidy |
| `hack/powertest/` | The real power-loss harness behind docs/audits/power-loss.md (decision 92); run by hand, not in the gate |
| `cmd/quill/` | The command: `main.go`, `verbs.go` (the verb table drives help and the reference), `read.go`, `consumers.go`, `write.go` (the verbs, grouped by what they touch), `print.go` |
| `examples/` | `log/`, `topics/`, `pipeline/` |
| `build/quill` | The release build: version stamping, cross-compilation, archives, checksums, SBOM |
| `.github/workflows/` | CI on Linux and macOS, the same gate as separate steps, the full-disk job, and the PR-issue-link check |


## Installing

The repository is private for now, so these paths apply once it is public; until then the owner builds with `build/quill`.

- **The library**: `go get github.com/netstar-labs/quill`, then import `github.com/netstar-labs/quill`.
- **The command**: `go install github.com/netstar-labs/quill/cmd/quill@v0.y.z` (a release tag), or unpack the archive for your platform from a release and put `quill` on your `PATH`.
- **Versions**: a go-installed binary prints `quill dev` for `quill version`; only the release archives are stamped with the version. The archives hold the binary, README, CHANGELOG, SECURITY, LICENSE and THIRD-PARTY-NOTICES (the licence of the Go runtime and standard library that the binary includes), not `docs/`, so the user guide and the format document are read in the repository.

## Building and the Go version

Standard library only; `go.mod` has no `require` line. **Go-version policy:** the toolchain
named in `go.mod` is the one supported version; quill tracks the latest stable Go release and
moves to the next one deliberately, together with the tool pins in `hack/gate.sh` and CI. The
gate and `build/quill` run Go at exactly that version (`GOTOOLCHAIN`), and CI installs it from `go.mod`. The
iterator API needs Go 1.23 or later. Always run Go with `GOWORK=off`:

```
GOWORK=off go test -race ./...
hack/gate.sh            # the commit gate; FUZZTIME=0 skips the fuzz step and ends PARTIAL, not OK
```

64-bit only; tested on arm64 and amd64; other platforms and 32-bit targets fail to build on purpose.

## Status

Private, pre-release, under construction by the phased build in the build prompt. Tags are
`v0.y.z` for ever: breaking changes bump the minor with a migration note. Licensed under the
Apache License 2.0 (see `LICENSE`).
