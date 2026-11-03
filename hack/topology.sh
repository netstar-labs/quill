#!/usr/bin/env bash
# Run the multi-process harness (hack/topology): hack/topology.sh run|selftest|verify [flags]
#
# Builds the quill command and the harness into a temporary directory, then runs the harness from
# the directory you are in, so a relative -dir means what it says. The harness starts one writer,
# readers, consumers, an operator, a pruner and a chaos controller against a log in -dir, which
# should be on the filesystem you want to test, and exits 1 on any violated invariant. Examples:
#
#	hack/topology.sh run -dir /mnt/test -duration 5m
#	hack/topology.sh run -dir /mnt/test -duration 5m -segment 65536 -retention 5s -max-bytes 1048576
#	hack/topology.sh run -dir /mnt/test -duration 5m -segment 16384 -retention 1s     # stalled protected consumers
#	hack/topology.sh run -dir /mnt/small -ctl /var/tmp -fill     # -dir on a filesystem you can fill
#	hack/topology.sh selftest -dir /var/tmp                      # the harness against injected faults
set -euo pipefail
export GOWORK=off
[ $# -ge 1 ] || { sed -n '2,3p' "$0" >&2; exit 2; }
repo=$(cd "$(dirname "$0")/.." && pwd)
bin=$(mktemp -d "${TMPDIR:-/tmp}/quill-topology.XXXXXX")
trap 'rm -rf "$bin"' EXIT
(cd "$repo" && go build -o "$bin/quill" ./cmd/quill && go build -o "$bin/topology" ./hack/topology)
QUILL_BIN="$bin/quill" "$bin/topology" "$@"
