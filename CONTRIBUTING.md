# Contributing

quill is small on purpose. A change is welcome when it keeps the guarantees in
[docs/design/scope.md](docs/design/scope.md), stays on the standard library, and comes with a
test that could falsify it.

## Ground rules

- **Standard library only.** `go.mod` has no `require` line and the gate refuses one.
- **Every guarantee has a test.** A guarantee without a test that could falsify it is not
  done. Recovery behaviour is tested against the crash simulator in `internal/fsx` and under a
  real `SIGKILL`; see [docs/audits/untestable-without-x.md](docs/audits/untestable-without-x.md)
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

runs gofmt, vet, build, the unsupported-platform check, `go test -race`, staticcheck,
deadcode, every fuzz target for 20 seconds, and the tidy check. `FUZZTIME=0` skips the fuzz
step and says so in red; that is not a green gate.

## Go version

The toolchain in `go.mod` is the one supported Go version; the module uses the latest stable
release and moves to the next one deliberately, with the pins in `hack/gate.sh` and
`.github/workflows/ci.yml`. The iterator API needs Go 1.23 or later. Always run Go with
`GOWORK=off`.

## Benchmarks

`docs/audits/benchmark-baselines.md` records six runs of every benchmark. A change that makes
the median of six runs more than 10% worse, with the ranges not overlapping, is a regression
and needs a reason.

## Security

See [SECURITY.md](SECURITY.md). Report privately.
