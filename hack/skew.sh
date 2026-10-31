#!/usr/bin/env bash
# Cross-version skew check: hack/skew.sh <refA> <refB>
#
# Builds the quill CLI at two git refs (git archive, GOWORK=off go build) and runs the same
# experiments both ways: A writes and B reads, then B writes and A reads. Records of several
# size classes in several segments, a consumer committed in the middle, a byte-for-byte
# comparison of every read-back against the payloads that were generated, check on both,
# appends by the other version, kill -9 of a writer mid-append then recovery by each version on
# its own copy of the killed bytes (the two directories must be identical) and an append by
# the other version, and quill.policy and consumer files rewritten by each version.
#
# Exits 0 only if both versions agree everywhere; on the first divergence it prints the
# evidence, exits 1 and keeps the work directory. Bash 3.2 and BSD tools suffice (macOS and
# Linux). SKEW_KILL_DELAYS (seconds, space separated) sets the kill timings; SKEW_KEEP=1 keeps
# the work directory on success too.
set -Eeuo pipefail
export GOWORK=off

[ $# -eq 2 ] || { echo "usage: hack/skew.sh <refA> <refB>" >&2; exit 2; }
repo=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/quill-skew.XXXXXX")
trap 'rc=$?; if [ $rc -eq 0 ] && [ -z "${SKEW_KEEP:-}" ]; then rm -rf "$work"; else echo "work directory kept: $work" >&2; fi' EXIT

where=start
die() { echo "SKEW: $*" >&2; exit 1; }
step() { where=$*; printf '==> %s\n' "$*"; }
# A command nothing else guards: name the step, so the raw error is not all there is.
trap 'echo "SKEW: unexpected failure in step \"$where\" (line $LINENO): $BASH_COMMAND" >&2; exit 1' ERR

build() { # build <ref> <name>: the CLI and the skewopen helper at that ref
	local sha src="$work/src-$2"
	sha=$(git -C "$repo" rev-parse --verify --quiet "$1^{commit}") || die "no such ref: $1"
	mkdir -p "$src/cmd/skewopen" "$work/bin-$2"
	git -C "$repo" archive "$sha" | tar -x -C "$src"
	[ -d "$src/cmd/quill" ] || die "$1 has no cmd/quill"
	cp "$repo/hack/skewopen/main.go" "$src/cmd/skewopen/main.go"
	(cd "$src" && go build -o "$work/bin-$2/quill" ./cmd/quill && go build -o "$work/bin-$2/skewopen" ./cmd/skewopen) ||
		die "go build failed at $1"
	echo "$2 = $1 ($sha)"
}

logbytes() { ls -l "$1"/*.log | awk '{ s += $5 } END { print s + 0 }'; } # bytes in the segments
bigN=10000 # records in the stream a writer is killed in

q() { local v=$1 d=$2; shift 2; "$work/bin-$v/quill" -dir "$d" "$@"; }
open() { local v=$1 d=$2; shift 2; "$work/bin-$v/skewopen" -dir "$d" "$@"; }
text() { q "$1" "$2" cat | cut -f1,3; } # seq and quoted payload, without the time

# gen "<size>[*<count>] ..." <first index>: one printable line per record, payload sizes as given.
gen() {
	awk -v spec="$1" -v idx="$2" 'BEGIN {
		n = split(spec, s, " ")
		base = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		for (k = 1; k <= n; k++) {
			m = split(s[k], a, "*"); size = a[1] + 0; cnt = (m > 1) ? a[2] + 0 : 1
			t = base; while (length(t) < size + 24) t = t t
			for (i = 0; i < cnt; i++) { print substr(idx ":" t, 1, size); idx++ }
		}
	}'
}

same() { # same <what> <fileA> <fileB>: byte-identical or fail with the evidence
	cmp -s "$2" "$3" && return 0
	{ echo "SKEW: $1 differs"; cmp "$2" "$3" || true; wc -c "$2" "$3"; diff "$2" "$3" | cut -c1-200 | head -n 6; } >&2
	exit 1
}

# verify <writer> <reader> <dir> <expected-payload-file>: both versions read the log exactly
# as generated, agree on every output, and call it healthy.
verify() {
	local w=$1 r=$2 d=$3 pay=$4 v t="$work/v" out
	awk '{ print NR "\t\"" $0 "\"" }' "$pay" >"$t.want"
	for v in $w $r; do
		text "$v" "$d" >"$t.text-$v"
		same "records read by $v against the generated payloads" "$t.want" "$t.text-$v"
		q "$v" "$d" cat -json -b64 >"$t.json-$v"
		q "$v" "$d" consumers -json >"$t.cons-$v"
		out=0; q "$v" "$d" check >"$t.check-$v" 2>&1 || out=$?
		[ "$out" = 0 ] || { cat "$t.check-$v" >&2; die "check by $v exits $out"; }
	done
	same "cat -json -b64 ($w vs $r)" "$t.json-$w" "$t.json-$r"
	same "consumers -json ($w vs $r)" "$t.cons-$w" "$t.cons-$r"
	same "check ($w vs $r)" "$t.check-$w" "$t.check-$r"
}

direction() { # direction <writer> <reader>
	local w=$1 r=$2 base="$work/$1$2" d i pay head v
	d="$base/log"; pay="$base/payloads"
	mkdir -p "$d"; : >"$pay"
	step "$w writes, $r reads"
	append() { # append <version> <spec>: generate, record the payloads, append as one batch
		gen "$2" $(($(wc -l <"$pay") + 1)) >"$base/batch"
		cat "$base/batch" >>"$pay"
		q "$1" "$d" append <"$base/batch" >/dev/null
	}
	append $w "0 1 17 100 4095 4096 4097 65536 1048575 1048576 1048577"
	q $w "$d" roll
	append $w "30*200"
	q $w "$d" roll
	append $w "0 1 2 3 5000*5"
	q $w "$d" roll
	q $w "$d" commit -name mid -offset $(($(wc -l <"$pay") / 2))
	append $w "1*50 100*50"
	[ "$(ls "$d" | grep -c '\.log$')" -ge 4 ] || die "expected several segments"
	verify $w $r "$d" "$pay"
	echo "ok: size classes, segments, consumer, read-back, check"

	append $r "7*10 4097*3 0"
	q $r "$d" roll
	append $r "64*20"
	verify $w $r "$d" "$pay"
	append $w "9*5 70000"
	verify $w $r "$d" "$pay"
	echo "ok: append by the other version ($r), then by $w again"

	# kill -9 of $w in the middle of a long stream of records; $r recovers and appends. The kill waits
	# until the writer has put bytes in the log, then SKEW_KILL_DELAYS seconds more, so a fast
	# machine does not kill it before it has written anything.
	gen "1000*$bigN" 1 >"$base/big"
	awk '{ print "\"" $0 "\"" }' "$base/big" >"$base/bigq"
	partial=0
	for i in ${SKEW_KILL_DELAYS:-0 0.002 0.005 0.01 0.02 0.04 0.08 0.15 0.3 0.6}; do
		step "$w killed ${i}s after its first write, $r recovers"
		text $w "$d" >"$base/pre"
		size=$(logbytes "$d")
		"$work/bin-$w/skewopen" -dir "$d" -retention=-1ns -stream <"$base/big" >/dev/null 2>&1 &
		pid=$!
		tries=0
		while [ "$(logbytes "$d")" = "$size" ] && [ $((tries += 1)) -le 1000 ]; do sleep 0.001; done
		[ "$(logbytes "$d")" != "$size" ] || { kill -9 "$pid" 2>/dev/null || true; die "$w wrote nothing in 5s"; }
		sleep "$i"; kill -9 "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true
		text $w "$d" >"$base/killed-$w"   # both versions read the killed log before any recovery
		text $r "$d" >"$base/killed-$r"
		same "log of the killed writer ($w vs $r), read before recovery" "$base/killed-$w" "$base/killed-$r"
		# recovery itself: the same killed bytes, recovered by each version on its own copy, must leave
		# the same directory (segments, quill.tail, quill.format, consumers). Reading the killed log
		# and checking afterwards would miss a version that salvages or drops different frames.
		rm -rf "$base/rec-$w" "$base/rec-$r"
		cp -R "$d" "$base/rec-$w"; cp -R "$d" "$base/rec-$r"
		q $w "$base/rec-$w" recover >/dev/null || die "recover by $w after kill at ${i}s failed"
		q $r "$base/rec-$r" recover >/dev/null || die "recover by $r after kill at ${i}s failed"
		diff -r "$base/rec-$w" "$base/rec-$r" >"$base/rec.diff" || {
			head -n 20 "$base/rec.diff" >&2
			die "recovery of the same killed log by $w and by $r left different directories (kill at ${i}s)"
		}
		q $r "$d" recover >/dev/null || die "recover by $r after kill at ${i}s failed"
		head=$(text $r "$d" | wc -l | tr -d ' ')
		gen "13 1000 70000 0" $((head + 1)) >"$base/after"
		q $r "$d" append <"$base/after" >/dev/null || die "append by $r after recovery failed"
		text $w "$d" >"$base/now"
		text $r "$d" >"$base/now-r"
		same "log read by $w and $r after kill at ${i}s" "$base/now" "$base/now-r"
		# the records before the kill are untouched, then a prefix of the killed stream, then ours
		n=$(wc -l <"$base/pre" | tr -d ' ')
		m=$((head - n)); k=$(wc -l <"$base/after" | tr -d ' ')
		{ [ "$m" -ge 0 ] && [ "$m" -le "$bigN" ]; } || die "after kill at ${i}s $head records, $n before it: the stream left $m"
		head -n "$n" "$base/now" | cmp -s - "$base/pre" || die "records before the kill at ${i}s changed"
		awk '$1 != NR { exit 1 }' "$base/now" || die "sequence not contiguous after kill at ${i}s"
		sed -n "$((n + 1)),${head}p" "$base/now" | cut -f2 >"$base/mid"
		head -n "$m" "$base/bigq" | cmp -s - "$base/mid" || die "the $m records that survived of the killed stream differ from the stream written (kill at ${i}s)"
		tail -n "$k" "$base/now" | cut -f2 >"$base/tail"
		awk '{ print "\"" $0 "\"" }' "$base/after" | cmp -s - "$base/tail" || die "records appended after recovery differ (kill at ${i}s)"
		for v in $w $r; do
			q $v "$d" check >"$base/check-$v" 2>&1 || { cat "$base/check-$v" >&2; die "check by $v after kill at ${i}s"; }
		done
		same "check after kill ($w vs $r)" "$base/check-$w" "$base/check-$r"
		q $w "$d" cat -json -b64 >"$base/j-$w"; q $r "$d" cat -json -b64 >"$base/j-$r"
		same "cat -json -b64 after kill ($w vs $r)" "$base/j-$w" "$base/j-$r"
		if [ "$m" -gt 0 ] && [ "$m" -lt "$bigN" ]; then partial=$((partial + 1)); fi
		echo "kill +${i}s: $n records before, $(($(wc -l <"$base/killed-$w") - n)) of the stream readable before recovery, $m of $bigN kept, $k appended by $r"
	done
	[ "$partial" -gt 0 ] || die "no kill left part of the stream (0 < kept < $bigN); set SKEW_KILL_DELAYS for this machine"
	echo "ok: kill -9, recovery and append by $r ($partial kills left part of the stream, each compared with it)"

	# quill.policy and consumer files: the same writes by each version must leave the same bytes.
	for v in $w $r; do
		cp -R "$d" "$base/pol-$v"
		p="$base/pol-$v"
		open $v "$p" -retention 36h -max-bytes 4294967296 -max-records 1000000 -protect keep
		q $v "$p" register -name mid -protect >/dev/null
		q $v "$p" commit -name mid -offset 3 >/dev/null
		q $v "$p" consumers >/dev/null
		open $v "$p" -retention=-1ns -protect keep,extra
		q $v "$p" register -name keep -no-protect >/dev/null
		q $v "$p" unregister -name extra >/dev/null
		q $v "$p" register -name extra -protect >/dev/null
		q $v "$p" commit -name mid -offset 5 >/dev/null
	done
	diff -r "$base/pol-$w" "$base/pol-$r" >"$base/pol.diff" || { head -n 20 "$base/pol.diff" >&2; die "files rewritten by $w and by $r differ"; }
	for v in $w $r; do
		q $v "$base/pol-$w" consumers >"$base/pc-$v"
		q $v "$base" ls >"$base/ls-$v"
		q $v "$base/pol-$w" check >/dev/null 2>&1 || die "check by $v after the policy and consumer rewrites"
	done
	same "consumers after the rewrites ($w vs $r)" "$base/pc-$w" "$base/pc-$r"
	same "ls after the rewrites ($w vs $r)" "$base/ls-$w" "$base/ls-$r"
	[ -s "$base/pol-$w/quill.policy" ] || die "quill.policy was not written"
	echo "ok: quill.policy and consumer files rewritten by $w and by $r are byte-identical"
}

step "build"
build "$1" a
build "$2" b
direction a b
direction b a
echo "OK: no divergence between $1 and $2"
