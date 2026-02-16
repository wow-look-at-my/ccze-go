#!/usr/bin/env bash
set -euo pipefail

# Format Go benchmark output into a markdown table on GITHUB_STEP_SUMMARY.
# Usage: testdata/bench-summary.sh

ns_to_ms() {
  awk "BEGIN { printf \"%.4f\", ($1 + 0) / 1000000 }"
}

human_bytes() {
  awk "BEGIN {
    v = $1 + 0
    if (v < 1024)          printf \"%d B\", v
    else if (v < 1048576)  printf \"%.1f KB\", v/1024
    else                   printf \"%.1f MB\", v/1048576
  }"
}

bench_to_table() {
  echo "| Benchmark | Iterations | ms/op | Memory | Allocs |"
  echo "|-----------|------------|-------|--------|--------|"
  while IFS= read -r line; do
    if [[ "$line" =~ ^Benchmark ]]; then
      name=$(echo "$line" | awk '{print $1}' | sed 's/-[0-9]*$//')
      iters=$(echo "$line" | awk '{print $2}')
      nsop=$(echo "$line" | awk '{print $3}')
      bop=$(echo "$line" | awk '{print $5}')
      aop=$(echo "$line" | awk '{print $7}')
      ms=$(ns_to_ms "$nsop")
      mem_h=$(human_bytes "$bop")
      echo "| ${name} | ${iters} | ${ms} | ${mem_h} | ${aop} |"
    fi
  done
}

OUT="${GITHUB_STEP_SUMMARY:=/dev/stdout}"

echo "## Benchmark Results" >> "$OUT"
echo "" >> "$OUT"

# System info
CPULINE=$(go test -bench='BenchmarkColorTableInit' -benchtime=100ms -count=1 2>&1 | grep '^cpu:' | sed 's/^cpu: //')
if [ -n "$CPULINE" ]; then
  echo "> **CPU:** ${CPULINE}" >> "$OUT"
  echo "" >> "$OUT"
fi

echo "### Go Component Benchmarks" >> "$OUT"
echo "" >> "$OUT"
go test -bench='Benchmark(Color|Wordcolor|.*Plugin|Registry|FullPipeline)' -benchmem -benchtime=500ms -count=1 2>&1 | bench_to_table >> "$OUT"
echo "" >> "$OUT"

echo "### Go vs C Throughput (go test)" >> "$OUT"
echo "" >> "$OUT"
go test -bench='BenchmarkVsC' -benchmem -benchtime=1s -count=1 2>&1 | bench_to_table >> "$OUT"
echo "" >> "$OUT"

# --------------------------------------------------------------------------
# Hyperfine: real-world shell-out comparison
# --------------------------------------------------------------------------

# Build the Go binary
go build -o /tmp/ccze-go .

SMALL_FILE="testdata/mixed.log"
SMALL_LINES=$(wc -l < "$SMALL_FILE")

echo "### Hyperfine: Mixed Log (${SMALL_LINES} lines, includes startup)" >> "$OUT"
echo "" >> "$OUT"
echo '```' >> "$OUT"
hyperfine --warmup 2 --min-runs 5 \
  --command-name "ccze-go" "/tmp/ccze-go -A < $SMALL_FILE > /dev/null" \
  --command-name "ccze (C)" "ccze -A < $SMALL_FILE > /dev/null" \
  2>&1 | tee -a "$OUT"
echo '```' >> "$OUT"

echo "" >> "$OUT"
echo "### Hyperfine: Startup Time (--list + exit)" >> "$OUT"
echo "" >> "$OUT"
echo '```' >> "$OUT"
hyperfine --warmup 5 --min-runs 20 \
  --command-name "ccze-go" "/tmp/ccze-go -l > /dev/null" \
  --command-name "ccze (C)" "ccze -l > /dev/null" \
  2>&1 | tee -a "$OUT"
echo '```' >> "$OUT"
