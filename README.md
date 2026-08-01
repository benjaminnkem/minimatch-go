# minimatch (Go)

[![ci](https://github.com/benjaminnkem/minimatch-go/actions/workflows/ci.yml/badge.svg)](https://github.com/benjaminnkem/minimatch-go/actions/workflows/ci.yml)

Idiomatic **Go** port of [isaacs/minimatch](https://github.com/isaacs/minimatch) (TypeScript), built for the **Port Mortem** hackathon: behavioural parity, documented decisions, differential tests, fuzzing, and a reproducible Docker build.

| | |
| --- | --- |
| **Original** | [isaacs/minimatch](https://github.com/isaacs/minimatch) **v10.2.6** (`ded1bbd01beca62a5978bc1650ed0a1ccf9039d5`) |
| **Module** | [`github.com/benjaminnkem/minimatch-go`](https://github.com/benjaminnkem/minimatch-go) **v0.1.0** |
| **License** | Blue Oak Model License 1.0.0 (same family as upstream) |
| **Status** | First release with Node differential coverage (196 + 46 fixtures) |

This package aims for **observable parity** (options, edge cases, ordering), not a line-by-line translation. When a sibling `../minimatch` tree is present, it is the behavioural specification; committed JSON fixtures work without it.

---

## Project overview

**minimatch** is the bash-style glob matcher used throughout the npm ecosystem (`*`, `?`, `**`, braces, extglobs, negation, Windows paths, etc.). This repository is the Go library and the Port Mortem submission entry point.

```go
import "github.com/benjaminnkem/minimatch-go"

ok, err := minimatch.Match("src/foo/bar.js", "**/*.js", minimatch.Options{})
// ok == true, err == nil
```

### Why this project was chosen

- **Non-trivial:** brace expansion, segment scanning, character classes, extglob AST, globstar, `MakeRe` lookarounds, Windows/UNC.
- **Strong oracle:** large public TAP suite → differential JSON fixtures.
- **Real utility:** glob matching is core infrastructure; a faithful Go port is useful beyond the hackathon.
- **No existing public behavioural Go port** of isaacs/minimatch (see [PORT_AUDIT.md](./PORT_AUDIT.md)).
- **Compatible licence** (Blue Oak 1.0.0).

---

## Install

```bash
go get github.com/benjaminnkem/minimatch-go@v0.1.0
```

Requires **Go 1.23+**.

---

## How to build

**One-command build (judges):**

```bash
docker build -t minimatch-port .
docker run --rm minimatch-port check
```

Local library build:

```bash
go build .
```

JSON differential tests run inside the image **without** Node. Optional live Node oracles / `make fuzz-diff` need a sibling `../minimatch` tree (see [TESTING.md](./TESTING.md)).

### Port Mortem submission map

| Deliverable | Location |
| --- | --- |
| Track / kickoff pin | [`.port-mortem.toml`](./.port-mortem.toml) |
| Original suite hashes | [`tests/original/`](./tests/original/) |
| DECISIONS | [`DECISIONS.md`](./DECISIONS.md) |
| Differential fuzz (60s+) | `make fuzz-diff` → [`fuzz/log.txt`](./fuzz/log.txt) |
| Comparative benches | `make bench-compare` → [`bench/results.json`](./bench/results.json) |
| Demo video script | [`DEMO_VIDEO.md`](./DEMO_VIDEO.md) |

---

## How to run

This is a **library**, not a long-running server.

```go
package main

import (
	"fmt"

	"github.com/benjaminnkem/minimatch-go"
)

func main() {
	ok, err := minimatch.Match("a/b/c.js", "**/*.js", minimatch.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(ok) // true

	m, err := minimatch.NewMinimatch("*.{js,ts}", minimatch.Options{MatchBase: true})
	if err != nil {
		panic(err)
	}
	fmt.Println(m.Match("pkg/index.ts")) // true
}
```

```bash
go test -run Example -v
```

### Main API

| Function / type | Purpose |
| --- | --- |
| `Match(path, pattern, opts)` | One-shot match |
| `NewMinimatch(pattern, opts)` | Compile once, match many |
| `MatchList(files, pattern, opts)` | Filter a list (`nonull` via `NoNull`) |
| `Filter(pattern, opts)` | Predicate for `slices` / manual loops |
| `BraceExpand(pattern, opts)` | Bash brace expansion only |
| `MakeRe(pattern, opts)` | Full-path regexp (prefer `Match` when possible) |
| `Escape` / `Unescape` | Literal-safe glob text |
| `NewDefaults(opts)` | Stack default options under per-call opts |
| `ParseGlob` / `AST` | Segment AST (advanced) |

Zero-value `Options{}` matches TypeScript `{}` for boolean flags. See `options.go` and [COMPATIBILITY.md](./COMPATIBILITY.md).

### Windows

Always prefer `/` in **patterns**. Backslashes in patterns are escapes unless `WindowsPathsNoEscape` is set.

```go
opts := minimatch.Options{Platform: minimatch.PlatformWin32, NoCase: true}
ok, _ := minimatch.Match(`C:\Users\x\file.txt`, "C:/Users/**/*.txt", opts)
```

---

## How to test

```bash
make check           # vet + test + differential
make test
make differential    # 196 + 46 Node-oracle fixtures (committed JSON)
make fuzz            # native fuzz ~10s
make fuzz-diff       # 60s Node↔Go differential → fuzz/log.txt
make bench
make bench-compare   # Node vs Go p99/startup → bench/results.json
```

Docker:

```bash
docker run --rm minimatch-port check
docker run --rm minimatch-port differential
docker run --rm minimatch-port fuzz
```

| Suite | What it proves |
| --- | --- |
| `go test ./...` | Unit + package behaviour |
| Differential JSON | Parity with upstream `patterns.js` / tricky negations |
| Fuzz | No panic on random input |
| Windows tests | UNC/drive/`PlatformWin32` |

Details: [TESTING.md](./TESTING.md) · [FUZZING.md](./FUZZING.md).

CI: [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) (Linux/macOS/Windows × Go 1.23/1.24).

---

## Benchmark results

**Summary (local, 2026-08-01 — full methodology in [BENCHMARKS.md](./BENCHMARKS.md)):**

| Benchmark | ~ns/op | ~B/op | allocs/op |
| --- | ---: | ---: | ---: |
| MatchSimple (compile+match) | 11 400 | 5 930 | 112 |
| MatchCompiled (match only) | 990 | 596 | 6 |
| NewMinimatchComplex | 120 000 | 63 700 | 1327 |
| MatchExtglob | 682 | 145 | 3 |
| MakeRe (rebuild) | 12 400 | 8 728 | 116 |
| BraceExpand | 4 730 | 1 208 | 54 |

```bash
make bench
# or
docker run --rm minimatch-port bench
```

Hardware for the table: Intel i7-9750H, macOS, `go1.26.5 darwin/amd64`, `-count=3`. **Do not quote without [BENCHMARKS.md](./BENCHMARKS.md).**

---

## Porting decisions

Short list (full write-up: **[DECISIONS.md](./DECISIONS.md)**):

| Decision | Choice |
| --- | --- |
| Layout | Public API at module root; stages under `internal/` |
| Regex engine | `regexp2` (lookarounds; RE2 insufficient) |
| Errors | `error` returns, not panics |
| Options | Zero value ≈ TS `{}`; pointers for tri-state numbers |
| Brace expansion | In-tree (no Node dependency at runtime) |
| Dual-body nested `!` | Skip dual body when hang risk in `regexp2` |
| Differential platform | Default fixtures to linux when unset |

---

## Known limitations

- **`MakeRe` source strings** are not guaranteed identical to JavaScript `RegExp` sources — compare match outcomes.
- **Nested dual-body + negative extglob** may differ from Node where we skip dual-body to avoid `regexp2` hangs ([BUGS.md](./BUGS.md) BUG-001).
- **API names** are idiomatic Go (`Match`, `NoBrace`), not the exact JS export shapes.
- **No claim** of being faster than Node without a paired harness.
- Live Node oracle tests **skip** if Node + `../minimatch` are absent; JSON fixtures always run.

See [COMPATIBILITY.md](./COMPATIBILITY.md).

---

## Documentation index (judges)

| Doc | Purpose |
| --- | --- |
| [DECISIONS.md](./DECISIONS.md) | **Why** the port looks the way it does |
| [PORT_AUDIT.md](./PORT_AUDIT.md) | Original SHA, LOC, prior Go search |
| [BENCHMARKS.md](./BENCHMARKS.md) | Methodology + raw results |
| [BUGS.md](./BUGS.md) | Upstream/engine findings |
| [FUZZING.md](./FUZZING.md) | Fuzz corpus and results |
| [COMPATIBILITY.md](./COMPATIBILITY.md) | Feature matrix |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Pipeline and module layout |
| [TESTING.md](./TESTING.md) | Differential strategy |
| [CHANGELOG.md](./CHANGELOG.md) | Milestones |
| [Dockerfile](./Dockerfile) | Reproducible build |

---

## Layout

```text
.
├── *.go                      # public API (package minimatch)
├── internal/
│   ├── brace/                # brace expansion
│   ├── scan/                 # segment tokenizer
│   ├── class/                # character classes
│   └── ast/                  # extglob AST + segment compile
├── testdata/                 # Node oracles, fixtures
├── scripts/                  # Docker entrypoint
├── .github/workflows/        # CI
├── DECISIONS.md … TESTING.md # Port Mortem docs
├── Dockerfile
├── Makefile
├── go.mod
└── LICENSE.md
```

Import only `github.com/benjaminnkem/minimatch-go`. Packages under `internal/` are implementation details.

Optional sibling for regenerating fixtures / live oracles:

```text
../minimatch/                 # isaacs/minimatch @ 10.2.6 (read-only reference)
```

---

## Development

```bash
make check          # vet + test + differential
make bench
make fuzz
make fixtures       # regenerate Node oracle (needs ../minimatch built)
```

---

## License

Blue Oak Model License 1.0.0 — see [`LICENSE.md`](./LICENSE.md).  
Aligned with upstream [isaacs/minimatch](https://github.com/isaacs/minimatch).
