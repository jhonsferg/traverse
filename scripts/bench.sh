#!/usr/bin/env bash
# bench.sh - Run traverse benchmarks locally with optional profiling.
#
# Usage:
#   ./scripts/bench.sh                        # all benchmarks, default settings
#   ./scripts/bench.sh -t 10s                 # custom benchtime
#   ./scripts/bench.sh -r BenchmarkThroughput # run specific benchmark
#   ./scripts/bench.sh -p                     # enable CPU + memory profiling
#   ./scripts/bench.sh -d ./benchmarks/02_throughput  # specific directory
#   ./scripts/bench.sh -p -d ./benchmarks/06_streaming_efficiency -t 5s
#
# Profiles are written to ./bench-profiles/ and can be inspected with:
#   go tool pprof bench-profiles/cpu.prof
#   go tool pprof bench-profiles/mem.prof

set -euo pipefail

BENCHTIME="3s"
BENCH_RUN="."
BENCH_DIR="./..."
PROFILE=false
OUTDIR="bench-profiles"

usage() {
  grep '^#' "$0" | grep -v '#!/' | sed 's/^# \{0,1\}//'
  exit 0
}

while getopts "t:r:d:ph" opt; do
  case $opt in
    t) BENCHTIME="$OPTARG" ;;
    r) BENCH_RUN="$OPTARG" ;;
    d) BENCH_DIR="$OPTARG/..." ;;
    p) PROFILE=true ;;
    h) usage ;;
    *) usage ;;
  esac
done

echo "==> traverse benchmark runner"
echo "    benchtime : $BENCHTIME"
echo "    filter    : $BENCH_RUN"
echo "    directory : $BENCH_DIR"
echo "    profiling : $PROFILE"
echo ""

EXTRA_FLAGS=""
if $PROFILE; then
  mkdir -p "$OUTDIR"
  EXTRA_FLAGS="-cpuprofile $OUTDIR/cpu.prof -memprofile $OUTDIR/mem.prof"
  echo "==> Profiles will be written to ./$OUTDIR/"
fi

go test \
  -bench="$BENCH_RUN" \
  -benchmem \
  -benchtime="$BENCHTIME" \
  -run='^$' \
  -count=1 \
  $EXTRA_FLAGS \
  $BENCH_DIR \
  | tee "$OUTDIR/bench-$(date +%Y%m%d-%H%M%S).txt" 2>&1

echo ""
echo "==> Done."

if $PROFILE; then
  echo ""
  echo "    Inspect CPU profile : go tool pprof $OUTDIR/cpu.prof"
  echo "    Inspect memory      : go tool pprof $OUTDIR/mem.prof"
  echo "    Web view (requires graphviz): go tool pprof -http=:8080 $OUTDIR/cpu.prof"
fi
