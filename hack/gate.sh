#!/usr/bin/env bash
# The commit gate (decision 70): one command, run before every commit. CI runs the same
# checks as separate steps; keep the tool pins below in step with .github/workflows/ci.yml.
#
# Fail-closed: a tool that is missing or a step that cannot run is red, never a skip.
set -euo pipefail
cd "$(dirname "$0")/.."

export GOWORK=off
# `go run tool@version` picks a toolchain from the tool's requirement, not this module's.
# Pin the toolchain to what go.mod declares so every step runs on the same Go.
toolchain=$(sed -n 's/^toolchain //p' go.mod)
[ -n "$toolchain" ] || toolchain="go$(sed -n 's/^go //p' go.mod)"
case "$toolchain" in go[0-9]*.[0-9]*.[0-9]*) ;; go[0-9]*.[0-9]*) toolchain="$toolchain.0" ;; esac
export GOTOOLCHAIN="$toolchain"
# Pinned, not @latest: a gate that starts failing with no change in the repo is not a gate.
STATICCHECK=v0.8.2
DEADCODE=v0.52.0
: "${FUZZTIME:=20s}"   # FUZZTIME=0 skips the fuzz step; the gate then says so, in red letters

step() { printf '==> %s\n' "$*"; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

step "no dependencies"
! grep -q '^require' go.mod || fail "go.mod has a require line (decision 14)"

step gofmt
unformatted=$(gofmt -l .)
[ -z "$unformatted" ] || fail "gofmt -w needed:"$'\n'"$unformatted"

# Enumerate packages before deciding to skip anything. Exit 0 with empty output is the one
# legitimate skip (a scaffold with no Go yet); a non-zero exit is a hard error.
step "go list"
list_err=$(mktemp); trap 'rm -f "$list_err"' EXIT
pkgs=$(go list ./... 2>"$list_err") || { cat "$list_err" >&2; fail "go list ./... failed; refusing to skip vet/build/test"; }
if [ -z "$pkgs" ]; then
	echo "no Go packages yet — skipping vet/build/test"
else
	step "go vet";        go vet ./...
	step "go build";      go build ./...
	step "unsupported platform must not build"
	! GOOS=freebsd go build ./... >/dev/null 2>&1 || fail "the module builds on freebsd; it must not (decision 19)"
	step "go test -race"; go test -race -count=1 ./...
	step "staticcheck";   go run honnef.co/go/tools/cmd/staticcheck@"$STATICCHECK" ./...
	step "deadcode"
	dead=$(go run golang.org/x/tools/cmd/deadcode@"$DEADCODE" -test ./...)
	[ -z "$dead" ] || fail "dead code:"$'\n'"$dead"
	if [ "$FUZZTIME" = 0 ]; then
		printf '\033[31mfuzz SKIPPED (FUZZTIME=0): this is not a green gate\033[0m\n'
	else
		step "fuzz ($FUZZTIME per target)"; FUZZTIME="$FUZZTIME" hack/fuzz.sh
	fi
fi

step "go mod tidy (no diff)"
go mod tidy
git diff --quiet -- go.mod go.sum || { git --no-pager diff -- go.mod go.sum; fail "go.mod/go.sum not tidy"; }

echo "OK: gate green"
