# Quill — size of the code and the documents

Counted on `main` at `3e674df` (2026-11-05), before this folder existed. Labels: [V] re-derived by a second
route; [J] a judgment call.

## Summary

- The production code is **6,152 lines of code** in 32 files: the root library, `cmd/quill/` and `internal/`.
  The unit tests that guard it are **22,449** lines in 148 files, **3.65 : 1**. Adding the harness and tool
  code under `hack/` and the examples, everything that is not production is **3.97 : 1**. [V]
- Production code is **24 %** comments by code lines (1,476 / 6,152); test code is **8 %**. The root library is
  the densest at 30 %. [V]
- The repository holds **20,572 lines of Markdown** in 49 files, **2.5 times** the 8,117 lines (code, comments
  and blanks) of production Go. The audit folder alone, 16,903 lines, is 2.1 times it. [V]
- The suite is **556** top-level `Test` functions, **15** `Fuzz` targets and **19** `Benchmark` functions; table
  rows and subtests are not counted. [V]

## Go code: production, tests and the rest

A *code* line has at least one Go token on it, so a trailing comment does not change the line's class. A
*comment* line has only comment text. *Blank* is empty. Lines inside a raw string count as code.

| Bucket | Files | Code | Comment | Blank | Total | Comment / code |
|---|--:|--:|--:|--:|--:|--:|
| Production: library (root package) | 20 | 3,949 | 1,186 | 314 | 5,449 | 30.0 % |
| Production: `cmd/quill/` | 6 | 1,204 | 140 | 81 | 1,425 | 11.6 % |
| Production: `internal/` | 6 | 999 | 150 | 94 | 1,243 | 15.0 % |
| **Production total** | **32** | **6,152** | **1,476** | **489** | **8,117** | **24.0 %** |
| Tests: library (root package) | 111 | 17,723 | 1,369 | 987 | 20,079 | 7.7 % |
| Tests: `cmd/quill/` | 33 | 3,505 | 316 | 231 | 4,052 | 9.0 % |
| Tests: `internal/` | 4 | 1,221 | 53 | 68 | 1,342 | 4.3 % |
| **Unit test total** | **148** | **22,449** | **1,738** | **1,286** | **25,473** | **7.7 %** |
| `hack/` tools: topology, powertest, skewopen (Go, non-test) | 4 | 1,529 | 136 | 77 | 1,742 | 8.9 % |
| `hack/` tool tests | 2 | 204 | 24 | 19 | 247 | 11.8 % |
| Examples (Go, non-test) | 3 | 208 | 17 | 17 | 242 | 8.2 % |
| Examples' tests | 1 | 61 | 3 | 4 | 68 | 4.9 % |
| **Go total** | **190** | **30,603** | **3,394** | **1,892** | **35,889** | **11.1 %** |

Shell (four scripts under `hack/` and `build/quill`): 357 code, 87 comment, 27 blank, 471 lines. `#` lines
count as comments; a shebang counts as code.

Of the 30,603 code lines, 20.1 % are production, 73.4 % are unit tests, 5.7 % are the `hack/` tools and
their tests, and 0.9 % are examples.

### Production against tests, by area

| Area | Production code | Comments / code | Test code | Test : prod. |
|---|--:|--:|--:|--:|
| Library (root package) | 3,949 | 30 % | 17,723 | 4.5 : 1 |
| `cmd/quill/` | 1,204 | 12 % | 3,505 | 2.9 : 1 |
| `internal/` | 999 | 15 % | 1,221 | 1.2 : 1 |
| **Total** | **6,152** | **24 %** | **22,449** | **3.65 : 1** |

The library carries the most test per line of production code, and `internal/` the least. `internal/fsx`, the
file-system seam with its real and crash-simulating implementations, is 993 of `internal/`'s 1,243 production
lines, and the simulator is test infrastructure that other tests drive, which is probably why the area looks
under-tested by this measure. [I] The root package is one package of 20 files, so a finer split by package is
not possible; its 111 test files are named by what they attack (`retention_crash_test.go`,
`reader_overlap_test.go`, and so on). The gate runs `go test -race -count=1 ./...`; one test file is
darwin-only, so no single platform runs every file.

## Documents: by location

| Location | Files | Lines | Non-blank |
|---|--:|--:|--:|
| `docs/audits/` | 33 | 16,903 | 13,409 |
| `docs/design/` | 5 | 1,601 | 1,323 |
| `docs/` (root: guide, format, architecture, reference, overview) | 6 | 1,323 | 1,073 |
| Repo root (`README`, `CHANGELOG`, `CONTRIBUTING`, `SECURITY`) and `examples/README.md` | 5 | 745 | 624 |
| **Total** | **49** | **20,572** | **16,429** |

### Inside `docs/audits/`

| Part | Files | Lines | What is in it |
|---|--:|--:|---|
| `passes/` | 20 | 10,281 | One record per audit pass: the candidates, the votes, the fixes |
| `decisions/` | 2 | 3,180 | The ledger of items held for the owner, with the owner's answers |
| `reference/` | 7 | 2,637 | What the code and other documents cite: threat model, format and skew, baselines, power loss |
| Top level | 4 | 805 | `README`, `summary`, the two release records |
| **Total** | **33** | **16,903** | |

Eighty-two percent of the Markdown is the audit record. The single longest file is the owner-decisions ledger
(3,060 lines); the user guide, at 619 lines, is the longest of the documents an operator reads. [V] by
`wc -l`. What each document is for is in [the audit README](../audits/README.md).

## Method and limits

- Go was counted with `go/scanner` (comments kept as tokens), so raw strings and trailing comments classify
  correctly. Each line is code, comment or blank; the three sum to the file's line count. The count ran on an
  export of commit `3e674df`, so the figures reproduce from that hash.
- Cross-checks [V]: the Go total (35,889) equals `wc -l` over the same 190 files; the blank lines (1,892) and
  the comment lines (3,394, every one a `//`-only line; the code has no block comments) agree with a `grep`
  count over the same files, which leaves 30,603 code lines; the Markdown total (20,572) equals `wc -l` over
  the same 49 files.
- Not counted: `testdata/` fixtures, `LICENSE`, the YAML under `.github/`, and `go.mod`. A test file is any
  `_test.go`; `build_script_test.go`, which tests the release script, is filed with the root tests.
- This folder (`docs/presentation/`) is documents, so the next count includes it. The numbers move with every
  commit: re-count before quoting them in a new release.
