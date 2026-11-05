# Contributing

quill is small on purpose. A change is welcome when it keeps the guarantees in
[docs/design/scope.md](docs/design/scope.md), stays on the standard library, and comes with a
test that could falsify it.

## Ground rules

- **Standard library only.** `go.mod` has no `require` line and the gate refuses one.
- **Every guarantee has a test.** A guarantee without a test that could falsify it is not
  done. Recovery behaviour is tested against the crash simulator in `internal/fsx` and under a
  real `SIGKILL`; see [docs/audits/reference/untestable-without-x.md](docs/audits/reference/untestable-without-x.md)
  for what cannot be tested and why.
- **The format is frozen.** `docs/format.md` is normative and the golden tests pin its bytes.
  A change needs a new version in `quill.format`, a migration note, and new goldens, never a
  re-recorded one.
- **Decisions are recorded.** A choice the design left open goes into
  `docs/design/decisions.md` with its reason. Prefer the smaller, safer behaviour.
- **Files stay under about 600 lines**, one responsibility each.
- **All file I/O goes through `internal/fsx`**, so the simulator can test it.

## Workflow

Issue, then a branch off `main`, then a PR that closes the issue (`Closes #N`); CI must be
green. Commit messages are conventional (`feat:`, `fix:`, `docs:`, `test:`, `chore:`).

Before every commit:

```
hack/gate.sh
```

runs the no-require check, gofmt, vet, build, a vet for linux and for darwin, the unsupported-platform check, `go test -race`, staticcheck,
deadcode, every fuzz target for 20 seconds, and the tidy check. `FUZZTIME=0` skips the fuzz
step, says so in red and ends with `PARTIAL` instead of `OK: gate green` (the exit status stays 0);
that is not a green gate.

## Go version

The toolchain in `go.mod` is the one supported Go version; the module uses the latest stable
release and moves to the next one deliberately, with the pins in `hack/gate.sh` and
`.github/workflows/ci.yml`. The gate and `build/quill` run Go at exactly that version
(`GOTOOLCHAIN`), and CI installs it from `go.mod`. The iterator API needs Go 1.23 or later. Always run Go with
`GOWORK=off`.

## Benchmarks

`docs/audits/reference/benchmark-baselines.md` records six runs of every benchmark. A change that makes
the median of six runs more than 10% worse, with the ranges not overlapping, is a regression
and needs a reason.

## Release checks

`hack/skew.sh <refA> <refB>` checks that two versions of quill can share a log directory. It builds the
`quill` command at both refs into a temporary directory (`git archive`, then `GOWORK=off go build`), plus
`hack/skewopen`, a few lines that open a log with a `Config` and close it, since no command rewrites
`quill.policy` (it can also append stdin one record at a time, which the kill phase uses). It then runs the same
experiments in both directions, A writing and B reading and the reverse: records in every size class from empty to
over 1 MiB across several segments, a consumer committed in the middle, and a byte-for-byte comparison of what each
version reads against the payloads that were generated, with `cat -json -b64`, `consumers -json` and `check`
required to be identical and healthy on both. The other version then appends. A writer is then killed with `kill -9`
ten times, each time after its first bytes reach the log and a little later, while it appends a stream of 1000-byte
records; each version then runs `recover` on its own copy of the same killed directory and the two copies must be
identical (segments, `quill.tail`, everything), since reading the log afterwards would not show a version that salvages
or drops different frames; the other version then runs `recover` on the original and appends, and the log must be an unchanged prefix, then the first
records of the stream exactly as they were generated (each compared with its line of the stream), then the new
records, in contiguous sequence, read identically by both. The run fails if no kill left part of the stream.
Finally each version rewrites `quill.policy` and the consumer files of a copy of the log, and the two copies must be
byte-identical. It exits nonzero with the first divergence, naming the step, and keeps its temporary directory
(`SKEW_KEEP=1` keeps it on success; `SKEW_KILL_DELAYS` sets the extra delays, in seconds). It needs bash and the
standard tools of macOS or Linux. Differences the changelog announces, such as `prune` re-protecting a consumer in an older version, are not
exercised and will not show; run it before a release against the previous tag.

## Security

See [SECURITY.md](SECURITY.md). Report privately.
