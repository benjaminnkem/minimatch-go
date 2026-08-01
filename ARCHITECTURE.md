# Architecture

How the Go port is structured and how data flows from a glob pattern to a match decision.

## Repository layout

```text
.                                    # this repository (github.com/benjaminnkem/minimatch-go)
├── *.go                             # public API (package minimatch)
├── internal/
│   ├── brace/                       # bash brace expansion
│   ├── scan/                        # path-segment tokenizer
│   ├── class/                       # character classes + POSIX
│   └── ast/                         # extglob AST, flatten, fillNegs, regexp source
├── testdata/                        # Node oracles + generators
├── scripts/                         # Docker entrypoint
├── .github/workflows/ci.yml
├── go.mod
├── Makefile
├── Dockerfile
├── README.md                        # submission + library overview
├── DECISIONS.md
├── PORT_AUDIT.md
├── BENCHMARKS.md
├── BUGS.md
├── FUZZING.md
├── COMPATIBILITY.md
├── ARCHITECTURE.md                  # this file
├── TESTING.md
├── CHANGELOG.md
└── LICENSE.md

# Optional sibling (not part of this git repo):
../minimatch/                        # TS reference (isaacs/minimatch @ 10.2.6)
```

There is no separate `pkg/` directory: the module root *is* the public package. Implementation detail lives under `internal/` so other modules cannot import it.

## Public surface

| API | Role |
| --- | --- |
| `Match` | One-shot compile + match |
| `NewMinimatch` | Compile once; reuse `(*Minimatch).Match` |
| `MatchList` / `Filter` | List filtering (`NoNull` / nonull behaviour) |
| `BraceExpand` | Brace expansion only |
| `MakeRe` / `(*Minimatch).MakeRe` | Full-path `regexp2` pattern |
| `Escape` / `Unescape` | Literal-safe glob text |
| `ParseGlob` / `AST` | Advanced AST access |
| `NewDefaults` | Stack default options |
| `Options` | Full option set (see field docs in `options.go`) |

Callers import only:

```go
import "github.com/benjaminnkem/minimatch-go"
```

## Pipeline

```text
                    Options (platform, flags, limits)
                              │
 pattern ──► validate ──► brace expand ──► split path parts
                 │              │                  │
                 │              ▼                  ▼
                 │         internal/brace    strip slashes /
                 │                              │
                 │                              ▼
                 │                         scan segments
                 │                         (internal/scan)
                 │                              │
                 │              ┌────────────────┼────────────────┐
                 │              ▼                ▼                ▼
                 │           literal        class parse      extglob AST
                 │           parts        (internal/class)  (internal/ast)
                 │              │                │                │
                 │              └────────┬───────┘                │
                 │                       ▼                       ▼
                 │                 MMPattern / star           flatten
                 │                 qmark / re                 fillNegs
                 │                       │                  toRegExpSource
                 │                       └────────┬───────────┘
                 │                                ▼
                 │                         Minimatch.set
                 │                         (compiled parts)
                 │                                │
                 │              ┌─────────────────┴─────────────────┐
                 │              ▼                                   ▼
                 │         matchOne / globstar                   MakeRe
                 │         (segment walk)                    (full path RE)
                 ▼
              (bool, error)
```

### Stage notes

1. **Validate** — empty/invalid patterns; UTF-16 length ≤ 64KiB; aligns with TS `assertValidPattern`.
2. **Brace expand** — produces one or more alternative pattern strings (`NoBrace` skips).
3. **Scan** — classifies tokens: literals, `*`, `?`, `**`, extglob openers, etc.
4. **Class / AST** — character classes become regexp fragments; extglobs become an AST that is flattened and negation-filled like the TS `AST` class.
5. **Compile parts** — each path segment becomes a matcher part (literal, compiled pattern, or globstar).
6. **Match** — walk path segments against pattern parts; globstar recursion capped by `MaxGlobstarRecursion`.
7. **MakeRe** — optional full-path regular expression with lookarounds via `regexp2`.

## Why `regexp2`?

Segment and full-path patterns use **lookaround**, which Go’s RE2-based `regexp` package rejects. `github.com/dlclark/regexp2` provides a closer match to JS `RegExp` features. Trade-offs are documented in [DECISIONS.md](./DECISIONS.md).

## Concurrency

- `Options` values are immutable configuration after construction (callers should not mutate shared option structs while matching).
- A compiled `*Minimatch` is safe for concurrent **read-only** `Match` if no method mutates it; `MakeRe` caches on the receiver — treat cache fill as requiring external sync if sharing across goroutines that also rebuild (typical use: compile once per pattern, match concurrently only if you do not clear caches). For hackathon scope, document “compile once per goroutine or sync MakeRe” as the conservative rule; primary `Match` path after compile is the hot path.

## Dependencies

| Dependency | Purpose |
| --- | --- |
| `github.com/dlclark/regexp2` | Lookaround-capable regexp |
| Go stdlib | Everything else |

No cgo. No Node at runtime.

## Mapping to TypeScript sources

| TS (`minimatch/src`, upstream) | Go |
| --- | --- |
| `index.ts` (Minimatch class, match, defaults) | `minimatch.go`, `match.go`, `public.go`, `make_re.go`, `options.go` |
| `ast.ts` | `internal/ast/*` |
| `brace-expressions.ts` + `brace-expansion` pkg | `internal/brace/*` |
| `escape.ts` / `unescape.ts` | `escape.go` / `unescape.go` |
| `assert-valid-pattern.ts` | `validate.go` |
| character class logic in index/ast | `internal/class` + scan |

## Design constraints

- **Observable parity** over structural isomorphism.
- **Errors not panics** for invalid input and limit breaches.
- **Zero-value `Options{}`** ≈ TypeScript `{}`.
- **Host-independent tests** via explicit `Platform` in fixtures.
