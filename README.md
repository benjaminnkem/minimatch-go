# minimatch (Go)

Idiomatic Go port of [isaacs/minimatch](https://github.com/isaacs/minimatch): bash-style glob matching used across the npm ecosystem.

The TypeScript tree in `../minimatch` is the behavioural specification. This package aims for **observable parity** (options, edge cases, ordering), not a line-by-line translation.

## Install

```bash
go get github.com/benjaminnkem/minimatch-go
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/benjaminnkem/minimatch-go"
)

func main() {
	ok, err := minimatch.Match("src/foo/bar.js", "**/*.js", minimatch.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(ok) // true

	m, err := minimatch.NewMinimatch("*.{js,ts}", minimatch.Options{MatchBase: true})
	if err != nil {
		panic(err)
	}
	fmt.Println(m.Match("pkg/index.ts")) // true

	files, _ := minimatch.MatchList(
		[]string{"a.js", "b.txt", "c.js"},
		"*.js",
		minimatch.Options{},
	)
	fmt.Println(files) // [a.js c.js]
}
```

## Main API

| Function / type                   | Purpose                                         |
| --------------------------------- | ----------------------------------------------- |
| `Match(path, pattern, opts)`      | One-shot match                                  |
| `NewMinimatch(pattern, opts)`     | Compile once, match many                        |
| `MatchList(files, pattern, opts)` | Filter a list (`nonull` via `NoNull`)           |
| `Filter(pattern, opts)`           | Predicate for `slices` / manual loops           |
| `BraceExpand(pattern, opts)`      | Bash brace expansion only                       |
| `MakeRe(pattern, opts)`           | Full-path regexp (prefer `Match` when possible) |
| `Escape` / `Unescape`             | Literal-safe glob text                          |
| `NewDefaults(opts)`               | Stack default options under per-call opts       |
| `ParseGlob` / `AST`               | Segment AST (advanced)                          |

### Options

See `Options` in `options.go`. Highlights:

- `Dot` — match leading `.` segments
- `NoCase` — case-insensitive
- `NoGlobStar` — treat `**` as `*`
- `NoExt` / `NoBrace` / `NoNegate` / `NoComment`
- `Partial` — prefix match for directory walks
- `MatchBase` — basename-only when pattern has no `/`
- `Platform` — set `PlatformWin32` for UNC/drive behaviour on any OS
- `OptimizationLevel` — `0` / `1` (default) / `≥2`
- `MaxGlobstarRecursion`, `MaxExtglobRecursion`, `BraceExpandMax` — safety limits

Zero-value `Options{}` matches TypeScript `{}` for boolean flags.

## Windows

Always prefer `/` in **patterns**. Backslashes in patterns are escapes unless `WindowsPathsNoEscape` is set.

```go
opts := minimatch.Options{Platform: minimatch.PlatformWin32, NoCase: true}
ok, _ := minimatch.Match(`C:\Users\x\file.txt`, "C:/Users/**/*.txt", opts)
```

Path arguments may use `\` on win32; they are normalized to `/` for comparison.

## Testing

```bash
go test ./...
go test -bench=. -benchmem
go test -fuzz=FuzzMatchNoPanic -fuzztime=10s
```

### Parity with Node

`testdata/patterns_node.json` is generated from the reference suite (`patterns.js`):

```bash
# from repo layout: minimatch/ (TS) next to minimatch-go/
cd ../minimatch && npm test   # builds dist
node ../minimatch-go/testdata/generate_patterns.mjs
cd ../minimatch-go && go test -run Differential -v
```

Random differential cases (requires Node + built reference):

```bash
go test -run FuzzDifferentialBatch -v
```

## Layout

```text
minimatch-go/
├── *.go                 # public API (package minimatch)
├── internal/
│   ├── brace/           # brace expansion
│   ├── scan/            # segment tokenizer
│   ├── class/           # character classes
│   └── ast/             # extglob AST + segment compile
├── testdata/            # Node oracles, fixtures
├── .github/workflows/   # CI
├── README.md
├── LICENSE.md
└── Makefile
```

Import only `github.com/benjaminnkem/minimatch-go`. The `internal/` packages are implementation details.

## Design notes

- Matching uses segment-wise compare (literals, compiled segment patterns, `**`).
- Segment regexps and `MakeRe` use [regexp2](https://github.com/dlclark/regexp2) so lookarounds from the TS sources work (stdlib RE2 does not).
- Pattern length is capped at 64KiB UTF-16 units (same as the reference).

## Development

```bash
make check          # vet + test + differential
make bench
make fuzz
make fixtures       # regenerate Node oracle (needs ../minimatch)
```

## License

Blue Oak Model License 1.0.0 — see [`LICENSE.md`](./LICENSE.md).
Aligned with upstream [isaacs/minimatch](https://github.com/isaacs/minimatch).
