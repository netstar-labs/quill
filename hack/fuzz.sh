#!/usr/bin/env bash
# Run every fuzz target in the module for FUZZTIME each (default 20s, decision 70).
# Used by hack/gate.sh and CI. A crash is never retried; the fuzz engine's own
# "context deadline exceeded" with no failing input is retried once.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off
: "${FUZZTIME:=20s}"
log=$(mktemp); trap 'rm -f "$log"' EXIT
status=0
for pkg in $(go list ./...); do
	for target in $(go test -list '^Fuzz' "$pkg" | grep '^Fuzz' || true); do
		ok=0
		for attempt in 1 2; do
			printf '==> fuzz %s %s (%s)\n' "$pkg" "$target" "$FUZZTIME"
			if go test -run '^$' -fuzz "^${target}\$" -fuzztime "$FUZZTIME" "$pkg" >"$log" 2>&1; then
				ok=1; break
			fi
			cat "$log"
			if grep -q 'Failing input written to' "$log" || ! grep -q 'context deadline exceeded' "$log"; then
				break
			fi
			echo "warning: $target: fuzz engine deadline error, retrying once"
		done
		[ "$ok" = 1 ] || status=1
	done
done
exit $status
