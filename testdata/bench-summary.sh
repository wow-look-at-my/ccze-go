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
go test -bench='Benchmark(Color|Wordcolor|.*Plugin|Registry|FullPipeline)' -benchmem -benchtime=2s -count=1 2>&1 | bench_to_table >> "$OUT"
echo "" >> "$OUT"

echo "### Go vs C Throughput Comparison" >> "$OUT"
echo "" >> "$OUT"
go test -bench='BenchmarkVsC' -benchmem -benchtime=3s -count=1 2>&1 | bench_to_table >> "$OUT"
echo "" >> "$OUT"

# Calculate speedup
LINES=$(wc -l < testdata/mixed.log)
GO_NS=$(go test -bench='BenchmarkVsCThroughput/Go' -benchtime=2s -count=1 2>&1 | grep 'BenchmarkVsCThroughput/Go' | awk '{print $3}')
C_NS=$(go test -bench='BenchmarkVsCThroughput/C' -benchtime=2s -count=1 2>&1 | grep 'BenchmarkVsCThroughput/C' | awk '{print $3}')

if [ -n "$GO_NS" ] && [ -n "$C_NS" ] && [ "$GO_NS" -gt 0 ]; then
  SPEEDUP=$(echo "scale=1; $C_NS / $GO_NS" | bc)
  GO_H=$(ns_to_ms "$GO_NS")
  C_H=$(ns_to_ms "$C_NS")
  echo "### Summary" >> "$OUT"
  echo "" >> "$OUT"
  echo "| Metric | Go | C |" >> "$OUT"
  echo "|--------|-----|---|" >> "$OUT"
  echo "| ms/op (${LINES} lines) | ${GO_H} | ${C_H} |" >> "$OUT"
  echo "| **Speedup** | **${SPEEDUP}x faster** | baseline |" >> "$OUT"
fi
