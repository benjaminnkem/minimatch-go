# Fuzzing

How the Go port is fuzzed, what corpus we use, and what we found.

## Goals

1. **No panics** on arbitrary pattern/path bytes (library safety).
2. **Differential signal** against Node when the reference tree is available.
3. Keep runs cheap enough for local + CI (`-fuzztime` seconds, not hours, in default docs).

## Harness overview

| Target / test | File | Type | Purpose |
| --- | --- | --- | --- |
| `FuzzMatchNoPanic` | `fuzz_test.go` | Go native fuzz | Random pattern/path → `Match` / `NewMinimatch` / `MakeRe` / `HasMagic` must not panic |
| `TestFuzzDifferentialBatch` | `fuzz_test.go` | Seeded PRNG + Node oracle | 80 structured random cases compared to Node |

Native fuzzing uses Go 1.18+ `testing.F`. Differential batch uses a small deterministic generator and `testdata/oracle.mjs`.

## Corpus and generators

### Native fuzz seed corpus

Hard-coded seeds in `FuzzMatchNoPanic`:

```text
*  **  a*  ?.js  !(a)  a/**/b  [a-z]  [[:alpha:]]  (empty)
```

The fuzzer mutates `(pattern, path)` pairs. Inputs are sanitized:

- Truncated to **256 bytes** (well under the 64KiB UTF-16 pattern cap).
- Invalid UTF-8 replaced via `strings.ToValidUTF8`.
- Patterns longer than `MaxPatternLength` are skipped (would only test the validator).

Option axes exercised inside the fuzz body:

- default `Options{}`
- `{Dot, NoCase, Partial}`
- `{NoNegate, NoGlobStar}`
- compiled path: `NewMinimatch` → `Match` / `MakeRe` / `HasMagic`

### Structured random generator (differential)

Alphabet (glob-relevant):

```text
abc*?[]!.-/_{},+@()\
```

Generators build short random globs and paths (bounded lengths) and randomize a subset of options (`dot`, `nocase`, `nonegate`, `noglobstar`, `matchBase`) with **platform forced to `linux`** for oracle stability.

Default batch size: **80** cases per `TestFuzzDifferentialBatch` run (seed fixed for reproducibility of the *generator*, not of Node).

## How to run

```bash
# from this repository
make fuzz

# native fuzz (example durations)
go test -fuzz=FuzzMatchNoPanic -fuzztime=10s
go test -fuzz=FuzzMatchNoPanic -fuzztime=60s   # longer soak

# differential batch (requires Node + built ../minimatch)
go test -run TestFuzzDifferentialBatch -v -count=1
```

Docker:

```bash
docker build -t minimatch-port .
docker run --rm minimatch-port fuzz
```

## Measured run (sample)

| Item | Value |
| --- | --- |
| Date | 2026-08-01 |
| Command | `go test -fuzz=FuzzMatchNoPanic -fuzztime=15s` |
| Hardware | Intel i7-9750H, darwin/amd64 |
| Result | **PASS** |
| Execs | ~188 612 in ~15s (~12k–21k exec/s varying) |
| Baseline coverage entries | 291 |
| New interesting inputs | +82 (total interesting ~373) |
| Crashes / panics | **0** |

Longer soaks are encouraged before claiming production hardening; this sample is evidence of the harness working, not a lifetime fuzz budget.

## Crashes found

| ID | Finding | Status |
| --- | --- | --- |
| — | No panic crashes in the recorded 15s native fuzz run | Clean |
| — | Historical development issues (nil deref, index bounds) fixed before release | Fixed in tree |

Interesting corpus entries are stored under Go’s fuzz cache locally (`testdata/fuzz/` when generated with `-fuzz` failures; none checked in as failure reproducers at v0.1.0).

## Behavioural differences surfaced by fuzzing / random oracles

Fuzzing is primarily **crash-oriented**. Behavioural diffs vs Node are handled by:

1. Embedded JSON differential suites (196 + 46 cases).
2. `TestFuzzDifferentialBatch` (strict list equality when Node succeeds).

During development, random/differential work helped catch:

| Issue class | Resolution |
| --- | --- |
| Platform `\` escape vs path sep on Windows CI | Force `platform=linux` for fixture JSON when unset |
| Nested extglob dual-body hang in `regexp2` | Skip dual body when `(?!(?:` present ([DECISIONS.md](./DECISIONS.md)) |
| Option mapping gaps (`nonull`, optimization level) | Fixed in `optionsFromJSON` / Options |

When Node throws and Go returns an error (or non-match), the batch test **skips strict compare** for that case — throws are not always 1:1 with Go errors for arbitrary junk strings.

## Coverage philosophy

| Layer | Role |
| --- | --- |
| Unit tests | Invariants of scan/class/ast/brace |
| Differential JSON | High-signal real patterns from upstream |
| Native fuzz | Panics / unexpected crashes on garbage |
| Random Node oracle | Sparse behavioural exploration |

Fuzzing does **not** replace the oracle suites; it complements them.

## Future improvements

- Persist a checked-in seed corpus under `testdata/fuzz/FuzzMatchNoPanic/`.
- Structure-aware mutator for balanced extglob parentheses.
- CI job with `-fuzztime=30s` on a schedule (not every PR, to save minutes).
- Crash triage template linking to [BUGS.md](./BUGS.md) if Node diverges interestingly.
