#!/usr/bin/env bash
# Check that tfplanlens works with no network and touches nothing it was not
# asked to: run it on the example plans inside an empty network namespace under
# strace, and fail on any socket call, any other program started, or any file
# written other than the --out targets.
#
#   scripts/verify-offline.sh [path-to-tfplanlens]
#
# Needs Linux, strace and unshare (root, or unprivileged user namespaces).
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
bin=${1:-$root/tfplanlens}
for tool in strace unshare; do
  command -v "$tool" >/dev/null || { echo "skipping: $tool not installed"; exit 0; }
done
[ -x "$bin" ] || { echo "build first: go build -o tfplanlens ./cmd/tfplanlens" >&2; exit 2; }
unshare -n true 2>/dev/null || { echo "skipping: cannot create a network namespace (need root or user namespaces)"; exit 0; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cd "$root/simulation"

run() { # <name> <args...>
  local name=$1; shift
  strace -f -qq -o "$work/trace.$name" -e trace=network,openat,creat,execve,unlink,unlinkat,rename,renameat,renameat2,mkdir,mkdirat \
    unshare -n "$bin" "$@" >/dev/null 2>"$work/err.$name" || true
}

run text     report assets/input/plan.json --no-color
run html     report assets/input/plan.json -f html -o "$work/out.html"
run multi    report assets/input/plan.json assets/input/plan-revised.json -f json
run diff     diff assets/input/plan.json assets/input/plan-revised.json -f markdown -o "$work/out.md"
run lookalike report ../testdata/plans/secret-lookalikes.json --no-color

fail=0
net=$(cat "$work"/trace.* | grep -E '(^|[0-9] +)(socket|connect|bind|listen|accept4?|sendto|sendmsg|recvfrom)\(' || true)
if [ -n "$net" ]; then echo "FAIL: network syscalls:"; echo "$net" | head; fail=1; fi

procs=$(cat "$work"/trace.* | grep 'execve(' | grep -vE 'unshare|tfplanlens' || true)
if [ -n "$procs" ]; then echo "FAIL: other programs started:"; echo "$procs" | head; fail=1; fi

writes=$(cat "$work"/trace.* | grep -E 'openat\(.*(O_WRONLY|O_RDWR|O_CREAT|O_TRUNC)' | grep -vE '/dev/(null|tty|pts)|/proc/|/sys/' \
  | grep -vE "\"$work/out\.(html|md)\"" || true)
if [ -n "$writes" ]; then echo "FAIL: files written other than --out:"; echo "$writes" | head; fail=1; fi

[ "$fail" = 0 ] && echo "ok: no network syscalls, no other programs, no writes besides --out (5 runs, empty network namespace)"
exit "$fail"
