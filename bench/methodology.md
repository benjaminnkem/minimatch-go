# Comparative benchmark methodology

Port Mortem deliverable: **original vs port** on a shared workload, with honest
metrics (not throughput-only).

## Workload

Same operations on both sides:

| Scenario | Description |
| --- | --- |
| `startup_cold` | Process start + one `Match("foo/bar.js", "**/*.js")` then exit |
| `match_oneshot` | Loop: full `Match` (compile + match) on `**/*.js` |
| `match_compiled` | Compile `**/*.js` once; loop match on `foo/bar/baz.js` |
| `match_extglob` | Compile `*(a\|b\|c)` once; loop match on `abc` |
| `match_complex_compile` | Loop compile only: `a/{b,c}/**/!(tmp)/*.{js,ts}` |
| `brace_expand` | Loop `BraceExpand("a{1..5}{b,c,d}x")` |
| `corpus` | 8 compiled patterns × 6 paths (same matrix as Go `BenchmarkMatchListPatternsCorpus`) |

Options: default / POSIX (`platform: 'linux'` on Node when set).

## Metrics

| Metric | Meaning |
| --- | --- |
| `ops` | Iterations in the timed loop (after warmup) |
| `ns_per_op_mean` | Mean wall time per op |
| `ns_per_op_p50` / `p95` / `p99` | Empirical quantiles from per-op samples (or batch-derived where noted) |
| `startup_ms` | Cold process time for `startup_cold` |
| `rss_bytes` | Process RSS after the workload (best-effort; see below) |
| `heap_bytes` | Language heap where available (Go `HeapAlloc`, Node `heapUsed`) |

## How to run

From the Go module root (sibling `../minimatch` built with `npm run prepare`):

```bash
make bench-compare
# or
./bench/run.sh
```

Artifacts:

- `bench/results.json` — machine-readable
- stdout summary table

## Implementation

| Side | Harness |
| --- | --- |
| Node (original) | `bench/node_bench.mjs` → loads `../minimatch/dist/esm/index.js` |
| Go (port) | `go run ./bench` → `github.com/benjaminnkem/minimatch-go` |
| Merge | `bench/run.sh` writes `results.json` |

## Fairness notes / confounders

1. **Different runtimes** — V8 vs Go GC; absolute times are not “which language is better,” they characterise *this* port vs *this* reference on one machine.
2. **RSS** — includes runtime baseline (Node especially). Report both RSS and heap.
3. **Compile vs match** — one-shot `Match` pays compile every time; production use is compile-once.
4. **Warmup** — each scenario warms before sampling to reduce JIT/GC noise; still single-machine, not multi-run statistical design.
5. **No claim of universal speedup** — Track C north star includes “starts measurably faster”; we report cold startup honestly either way.

## Hardware

Recorded in `results.json` → `meta.hardware` at run time (`uname`, Go version, Node version).
