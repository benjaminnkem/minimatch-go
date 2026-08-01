# Benchmarks

All claims below include **methodology**, **hardware**, **commands**, and **raw results**.
Do not cite these numbers without that context.

## Methodology

1. **What is measured** — Go `testing.B` benchmarks in `benchmark_test.go`.
2. **What is not measured** — No cross-language “Go is Nx faster than Node” claim is made here without a paired harness. A Node `benchmark.js` exists upstream; comparing runtimes fairly requires the same patterns, warmup, and harness (future work).
3. **How to run**

   ```bash
   make bench
   # or
   go test -bench=. -benchmem -count=3
   ```

4. **Flags**
   - `-bench=.` — all benchmarks in the module root package
   - `-benchmem` — allocs/op and B/op
   - `-count=3` — three runs; report median or show all (we show all)
5. **Build** — default `go test` build (`-gcflags` unset). Module: Go 1.23+.
6. **Stability** — machine was otherwise idle enough for local hackathon reporting; treat ±10–15% as noise across machines.

## Hardware & software (this report)

| Item | Value |
| --- | --- |
| Machine | MacBook Pro (local dev) |
| CPU | Intel(R) Core(TM) i7-9750H @ 2.60GHz (12 logical CPUs reported by Go) |
| Arch | `amd64` (`x86_64`) |
| OS | macOS 26.6 (`darwin`) |
| Go | `go1.26.5 darwin/amd64` |
| Date | 2026-08-01 |
| Module | `github.com/benjaminnkem/minimatch-go` @ local tree |

## Benchmark inventory

| Name | What it stresses |
| --- | --- |
| `BenchmarkMatchSimple` | One-shot `Match` (compile + match each iteration) on `**/*.js` |
| `BenchmarkMatchCompiled` | `NewMinimatch` once, then `Match` only |
| `BenchmarkNewMinimatchComplex` | Compile cost of braces + extglob + globstar |
| `BenchmarkMatchExtglob` | Compiled `*(a\|b\|c)` match |
| `BenchmarkMatchListPatternsCorpus` | Small multi-pattern × multi-file matrix |
| `BenchmarkMakeRe` | Full-path regexp build (cache cleared each iter) |
| `BenchmarkBraceExpand` | Brace expansion only |

## Results (raw)

Command:

```text
go test -bench=. -benchmem -count=3
```

```text
goos: darwin
goarch: amd64
pkg: github.com/benjaminnkem/minimatch-go
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkMatchSimple-12                	  101688	     11408 ns/op	    5931 B/op	     112 allocs/op
BenchmarkMatchSimple-12                	  104386	     11995 ns/op	    5935 B/op	     112 allocs/op
BenchmarkMatchSimple-12                	  107926	     11316 ns/op	    5930 B/op	     112 allocs/op
BenchmarkMatchCompiled-12              	 1248036	      1098 ns/op	     596 B/op	       6 allocs/op
BenchmarkMatchCompiled-12              	 1253199	       987.1 ns/op	     596 B/op	       6 allocs/op
BenchmarkMatchCompiled-12              	 1259428	       943.6 ns/op	     596 B/op	       6 allocs/op
BenchmarkNewMinimatchComplex-12        	    9006	    123052 ns/op	   63746 B/op	    1327 allocs/op
BenchmarkNewMinimatchComplex-12        	    9602	    120371 ns/op	   63716 B/op	    1327 allocs/op
BenchmarkNewMinimatchComplex-12        	    9474	    119888 ns/op	   63716 B/op	    1327 allocs/op
BenchmarkMatchExtglob-12               	 1749235	       680.3 ns/op	     145 B/op	       3 allocs/op
BenchmarkMatchExtglob-12               	 1714861	       682.0 ns/op	     145 B/op	       3 allocs/op
BenchmarkMatchExtglob-12               	 1761237	       683.9 ns/op	     145 B/op	       3 allocs/op
BenchmarkMatchListPatternsCorpus-12    	   39730	     30175 ns/op	   11301 B/op	     160 allocs/op
BenchmarkMatchListPatternsCorpus-12    	   39603	     30035 ns/op	   11291 B/op	     160 allocs/op
BenchmarkMatchListPatternsCorpus-12    	   39998	     30029 ns/op	   11303 B/op	     160 allocs/op
BenchmarkMakeRe-12                     	   96170	     12489 ns/op	    8728 B/op	     116 allocs/op
BenchmarkMakeRe-12                     	   96270	     12464 ns/op	    8728 B/op	     116 allocs/op
BenchmarkMakeRe-12                     	   96642	     12333 ns/op	    8728 B/op	     116 allocs/op
BenchmarkBraceExpand-12                	  248029	      4728 ns/op	    1208 B/op	      54 allocs/op
BenchmarkBraceExpand-12                	  250581	      4733 ns/op	    1208 B/op	      54 allocs/op
BenchmarkBraceExpand-12                	  249374	      4736 ns/op	    1208 B/op	      54 allocs/op
```

## Summary table (approx. median of 3)

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| MatchSimple (compile+match) | ~11 400 | ~5 930 | 112 |
| MatchCompiled (match only) | ~990 | 596 | 6 |
| NewMinimatchComplex | ~120 000 | ~63 700 | 1327 |
| MatchExtglob | ~682 | 145 | 3 |
| MatchListPatternsCorpus | ~30 000 | ~11 300 | 160 |
| MakeRe (rebuild) | ~12 400 | 8 728 | 116 |
| BraceExpand | ~4 730 | 1 208 | 54 |

## Interpretation (honest)

- **Compile once, match many** is an order of magnitude cheaper than one-shot `Match` for `**/*.js` on this machine (~1 µs vs ~11 µs per op in the simple benchmarks).
- Complex patterns with braces + negative extglob are **compile-heavy** (~0.12 ms, >1k allocs) — expected given AST + regex build.
- These numbers characterise **this port on this hardware**, not a claim of superiority over the TypeScript implementation.

## Reproducing in Docker

```bash
docker build -t minimatch-port .
docker run --rm minimatch-port bench
```

(See root `Dockerfile` — `bench` target runs the same `go test -bench` command inside the image. CPU model will differ; always re-record hardware when publishing new numbers.)

## Future work (not claimed yet)

- Side-by-side harness vs `minimatch/benchmark.js` with identical pattern/path sets.
- Allocation profiles (`-memprofile`) for compile path.
- Throughput under parallel `b.RunParallel` for server-style matchers.
