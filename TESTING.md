# Testing

How we verify the Go port against the TypeScript reference and its own unit/fuzz suite.

## Strategy (layers)

```text
┌─────────────────────────────────────────────────────────┐
│  1. Unit tests per subsystem (brace, scan, class, ast)  │
├─────────────────────────────────────────────────────────┤
│  2. Package tests (Match, Options, Windows, Escape, …)  │
├─────────────────────────────────────────────────────────┤
│  3. Differential fixtures (Node oracle → JSON → Go)     │
├─────────────────────────────────────────────────────────┤
│  4. Live Node oracle batch (random cases, optional)     │
├─────────────────────────────────────────────────────────┤
│  5. Native fuzz (no panic)                              │
├─────────────────────────────────────────────────────────┤
│  6. Benchmarks (performance characterisation)           │
└─────────────────────────────────────────────────────────┘
```

Philosophy: **the TypeScript implementation is the specification**. When Go and Node disagree, we fix Go unless we document an intentional engine limitation ([COMPATIBILITY.md](./COMPATIBILITY.md), [DECISIONS.md](./DECISIONS.md)).

---

## Quick commands

From this repository:

```bash
make test          # go test ./...
make vet
make differential  # Differential* tests
make check         # vet + test + differential
make fuzz          # short native fuzz
make bench         # go benchmarks

# equivalent without make:
go test ./...
go vet ./...
go test -run 'Differential' -v -count=1
go test -fuzz=FuzzMatchNoPanic -fuzztime=10s
go test -bench=. -benchmem -count=1
```

Docker (reproducible):

```bash
docker build -t minimatch-port .
docker run --rm minimatch-port test
docker run --rm minimatch-port differential
```

---

## Layer 1–2: Go unit tests

| Area | Location |
| --- | --- |
| Brace expansion | `internal/brace/*_test.go` |
| Scanner | `internal/scan/*_test.go` |
| Character classes | `internal/class/*_test.go` |
| AST / flatten / fillNegs / regexp | `internal/ast/*_test.go` |
| Options / platform / validate | `options_test.go`, `platform_test.go`, `validate_test.go` |
| Match / public API | `match_test.go`, `public_test.go`, `minimatch_test.go` |
| Escape / unescape | `escape_test.go` |
| Windows / UNC | `windows_test.go` |
| Examples | `example_test.go` |

Run all:

```bash
go test ./...
```

CI runs this on Ubuntu, macOS, and Windows × Go 1.23 / 1.24 (see `.github/workflows/ci.yml`).

---

## Layer 3: Differential fixtures (primary parity evidence)

### How fixtures are produced

1. Build the reference TypeScript package:

   ```bash
   cd ../minimatch && npm ci && npm run prepare
   ```

2. Generate JSON from upstream test data:

   ```bash
   node testdata/generate_patterns.mjs
   ```

3. Checked-in artifacts:

   | File | Source | Cases |
   | --- | --- | --- |
   | `testdata/patterns_node.json` | Upstream `patterns.js` style corpus | **196** |
   | `testdata/tricky_negations.json` | Tricky negation corpus | **46** |

Fixtures embed **pattern**, **files**, **options**, and **expected** match lists (or per-file want bool for tricky negations).

### How Go consumes them

`differential_test.go`:

1. Load JSON.
2. Map option keys → `Options` (`optionsFromJSON`).
3. Default `Platform` to **linux** when unset (POSIX escape semantics; see [BUGS.md](./BUGS.md) BUG-002).
4. Run `MatchList` / `Match` and compare to expected (order-normalized where needed).

```bash
go test -run 'DifferentialPatterns|DifferentialPartial|DifferentialTricky' -v -count=1
```

These tests **do not require Node at runtime** — only the committed JSON — so CI on pure Go runners stays green.

### Regenerating after upstream bumps

```bash
make fixtures
```

Re-run differential tests and commit JSON only when intentional.

---

## Layer 4: Live Node oracle

`testdata/oracle.mjs` accepts a JSON array of `{ files, pattern, options }` and prints Node minimatch results.

`TestFuzzDifferentialBatch` (and helpers) spawn Node when:

- `node` is on `PATH`
- Reference package resolves (sibling `../minimatch` with built `dist/`)

```bash
go test -run TestFuzzDifferentialBatch -v -count=1
```

If Node/reference is missing, the test **skips** (not fails).

---

## Layer 5: Fuzzing

See [FUZZING.md](./FUZZING.md).

```bash
go test -fuzz=FuzzMatchNoPanic -fuzztime=10s
```

---

## Layer 6: Benchmarks

See [BENCHMARKS.md](./BENCHMARKS.md).

```bash
go test -bench=. -benchmem -count=3
```

---

## Mapping original TypeScript tests → Go

| Upstream (`minimatch/test`) | Go coverage approach |
| --- | --- |
| `patterns.js` | `patterns_node.json` + `DifferentialPatterns` |
| `tricky-negations.js` | `tricky_negations.json` + `DifferentialTricky` |
| `brace-expand.js` | `internal/brace` tests + public `BraceExpand` |
| `escaping.js` / `escape-has-magic.js` | `escape_test.go` |
| `win-path-sep.js` / `windows-no-magic-root.ts` / `unc.ts` | `windows_test.go` + platform options |
| `partial.ts` | Differential partial cases + unit tests |
| `defaults.js` | `NewDefaults` / options tests |
| `nested-extglob.ts` / extglob edges | AST tests + differential + hang mitigation |
| `redos.js` | Limits on recursion; dual-body guard; fuzz soak |
| Snapshots under `tap-snapshots/` | Not re-hosted; behaviour covered via oracles |

We do **not** re-run the TAP runner from Go. We **extract expectations** (JSON) or **reimplement critical cases** as Go tests.

---

## CI

`.github/workflows/ci.yml`:

| Job | Matrix | Steps |
| --- | --- | --- |
| `test` | OS: ubuntu, macos, windows × Go 1.23, 1.24 | `go test ./...`, `go vet ./...` |
| `differential` | ubuntu, Go 1.24 | Differential fixture tests (JSON only) |

`Dockerfile` in this repository provides an offline reproducible path for judges without GitHub Actions.

---

## What “green” means for submission

| Check | Command | Expected |
| --- | --- | --- |
| Unit + package | `make test` | PASS |
| Vet | `make vet` | PASS |
| Differential fixtures | `make differential` | PASS (196 + 46) |
| Fuzz (short) | `make fuzz` | PASS, 0 crashes |
| Docker | `docker build` + `test` | PASS |

Optional: live oracle batch when Node + `minimatch/` are present.
