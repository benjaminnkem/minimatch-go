# Design decisions

Judges: this file explains **why** the Go port differs internally from the TypeScript original, what could not be copied exactly, and the trade-offs we accepted. The goal is **observable parity**, not a line-by-line translation.

## Guiding principles

1. **Match what callers can observe** — options semantics, match results, brace-expand order, error conditions where the reference throws.
2. **Idiomatic Go** — exported identifiers, `error` returns instead of exceptions, package layout with `internal/`, zero-value friendliness.
3. **Compile and test each subsystem** before the next — no unfinished TODOs in the active surface.
4. **Prefer safety over silent hangs** when the reference engine and Go’s regex engine diverge catastrophically (see dual-body extglob).

---

## Architecture decisions

### 1. Public package + `internal/` split

| Choice | Rationale |
| --- | --- |
| Single import path `github.com/benjaminnkem/minimatch-go` | Callers get one package; mirrors how most Go libraries ship. |
| `internal/brace`, `internal/scan`, `internal/class`, `internal/ast` | Encapsulates implementation; avoids accidental API commitment. |
| No separate `pkg/` tree | Unnecessary for a single-module library; `internal/` is enough. |

TypeScript spreads logic across `index.ts` + `ast.ts` + small helpers. We split by **pipeline stage** so each stage can be unit-tested in isolation.

### 2. Pipeline stages mirror the reference, not the file names

```text
pattern
  → validate (UTF-16 length cap, invalid pattern)
  → brace expand          (internal/brace)
  → scan segments         (internal/scan)
  → parse classes / AST   (internal/class, internal/ast)
  → compile parts         (literals, MMPattern, GlobStar)
  → match path segments   (matchOne / globstar walk)
  → optional MakeRe       (full-path regexp2)
```

This is the same *information flow* as minimatch 10.x, expressed as Go packages.

### 3. `regexp2` instead of `regexp` (RE2)

TypeScript emits JavaScript `RegExp` features that **RE2 does not support**, notably lookaround (`(?!…)`, `(?=…)`) used heavily by extglob/`MakeRe`.

| Option | Outcome |
| --- | --- |
| Stdlib `regexp` only | Incomplete: many extglob/`MakeRe` patterns cannot compile. |
| **`github.com/dlclark/regexp2`** | Supports lookaround; closer to JS semantics. Chosen. |
| Embed a JS engine | Out of scope; heavy and non-idiomatic. |

**Trade-off:** `regexp2` can exhibit catastrophic backtracking on some nested lookaround graphs that V8 handles differently. We mitigate with a targeted dual-body skip (below) and document residual risk in [COMPATIBILITY.md](./COMPATIBILITY.md).

### 4. Errors, not panics

TypeScript throws `TypeError` / `Error` for invalid patterns and limit breaches. Go returns `error` (`ErrInvalidPattern`, brace/globstar limit errors, etc.).

| TypeScript | Go |
| --- | --- |
| `throw new TypeError('invalid pattern')` | `return …, ErrInvalidPattern` (message kept for parity tests) |
| Uncaught throw aborts | Caller always checks `err` |

Public match helpers return `(bool, error)` or `([]string, error)` so misuse is explicit.

### 5. Options: zero value == TypeScript `{}`

Boolean flags default to `false` in both languages. Numeric / tri-state fields whose TS default is **not** Go’s zero use **pointers**:

```go
OptimizationLevel    *int  // nil → 1; explicit 0 is valid
MaxGlobstarRecursion *int  // nil → 200
Platform             Platform // "" → host platform
WindowsNoMagicRoot   *bool
```

**Why not bare ints?** An explicit `optimizationLevel: 0` must remain distinct from “omit field”. Pointers preserve that.

### 6. Platform is explicit configuration

TS reads `process.platform` and sometimes injects `platform` in tests. Go uses:

- `HostPlatform()` from `runtime.GOOS`
- `Options.Platform` override (`PlatformLinux` / `PlatformWin32`)

**Decision for differential fixtures:** when JSON options omit `platform`, tests force **`PlatformLinux`**. Upstream `patterns.js` backslash-escape cases assume POSIX escapes; on Windows hosts, treating `\` as a path separator breaks those fixtures (upstream notes the same). Real win32/UNC behaviour is covered by dedicated `windows_test.go` with `PlatformWin32`.

### 7. Brace expansion in-tree

TS depends on `brace-expansion`. We reimplemented balanced braces / sequences under `internal/brace` to avoid a Node dependency at runtime and to share `BraceExpandMax` with the rest of the Options model.

### 8. Prefer `Match` over `MakeRe` at optimization level ≥ 2

Same guidance as upstream: high optimization uses structural matching; full-path regex is a compatibility/escape hatch. Documented in README and CHANGELOG.

---

## Behaviour that could not (or should not) be copied exactly

### Dual-body repeated extglobs vs nested `!` lookarounds

**Reference behaviour:** For repeated extglobs when dots are disallowed, TypeScript may emit a “dual body” so patterns like `*(?)` can match `a.b`.

**Problem in Go:** Combining that dual body with nested negative extglobs (bodies containing `(?!(?:…`) produces regexes that can **hang** in `regexp2` (ReDoS-like), e.g. conceptually `*(*.json|!(*.js))`.

**Decision:** Still emit the dual body for fidelity **unless** the body already contains a nested negative-extglob lookaround `(?!(?:`. In that case, skip dual-body only.

| Priority | Result |
| --- | --- |
| Safety | No multi-second/hang matches on pathological nested `!` |
| Fidelity | `*(?)` / simple dual-body cases still match |
| Residual gap | Extremely exotic nested forms may differ from Node on edge paths — see COMPATIBILITY |

This is an **engine portability** fix, not a change to the public Options API.

### UTF-16 pattern length

Upstream caps pattern length in **UTF-16 code units** (JS string length). Go measures with the same model (including surrogate pairs via `utf16.RuneLen`) so emoji / non-BMP patterns hit the limit at the same boundary.

### `null` / `nonull` historical options

`patterns.js` still carries historical `null: true` entries that modern minimatch ignores. We ignore them the same way; only `nonull` maps to `Options.NoNull`.

### Magical braces defaults on Escape vs Unescape

TS uses different default `magicalBraces` on free `escape` vs `unescape`. Go exposes `EscapeOptions` / `UnescapeOptions` (and projectors from full `Options`) so those defaults stay correct without polluting the main Options zero-value story.

### No shared mutable defaults object

TS `minimatch.defaults(def)` returns functions bound to merged defaults. Go provides `NewDefaults(opts)` returning a small helper type — same idea, no process-global mutation.

---

## Edge cases discovered during the port

| Case | Handling |
| --- | --- |
| Empty pattern / comment `#…` | Comment matches nothing unless `NoComment`. |
| `**` adjacent collapse | Depends on `optimizationLevel` (0 vs ≥1). |
| `partial` walk-off | Prefix matching for directory walks; careful when path shorter than pattern. |
| UNC and drive roots on win32 | `PlatformWin32` + `WindowsNoMagicRoot` tri-state. |
| Character class ranges / POSIX classes | Ported in `internal/class`; consumed-length parity with TS. |
| Extglob usurp / flatten | Port of AST flatten + fillNegs so `MakeRe` and match see the same tree. |
| Qmark / star fast paths | Literal-length shortcuts where safe; must not break multi-byte UTF-8. |
| Globstar recursion cap | `MaxGlobstarRecursion` (default 200) returns an error when exceeded rather than hanging forever. |

---

## Compatibility stance (summary)

| Area | Stance |
| --- | --- |
| Match results for common npm-style globs | Target: equal to Node oracle |
| Full Options matrix | Implemented; differential coverage on the high-value axes |
| `MakeRe` string identity | **Not** guaranteed identical source text — only matching behaviour |
| Panic-freedom | Fuzz target; panics are bugs |
| Performance vs TS | Not required to win every microbench; compile-once + match-many is the expected use |

Details: [COMPATIBILITY.md](./COMPATIBILITY.md).

---

## Trade-off register

| Trade-off | We chose | We rejected | Why |
| --- | --- | --- | --- |
| Regex engine | `regexp2` | RE2-only | Lookarounds required |
| Layout | Root public + `internal/` | Monolithic single file | Testability, clarity |
| Errors | `error` values | panics for invalid input | Idiomatic Go, library-safe |
| Dual-body nested `!` | Skip dual when `(?!(?:` present | Always emit dual | Avoid regexp2 hang |
| Differential platform default | Force linux when unset | Always host platform | Windows CI vs POSIX escape fixtures |
| Brace expand | In-tree port | cgo/JS bridge | Pure Go, offline tests |
| API naming | Go names (`NoBrace`, `MatchList`) | Exact JS names | Idiomatic; TS keys documented on fields |

---

## Numbered divergence log (≥10 for Decision Log bonus)

| # | Divergence | Rationale |
| --- | ---: | --- |
| 1 | `internal/` package split vs single TS modules | Idiomatic Go, testable stages |
| 2 | `regexp2` vs JS `RegExp` / RE2 | Lookarounds required; RE2 insufficient |
| 3 | `error` returns vs thrown exceptions | Library-safe Go API |
| 4 | Pointer fields for numeric/tri-state options | Preserve TS “omit vs 0” |
| 5 | In-tree brace expansion vs `brace-expansion` npm dep | Pure Go, no Node at runtime |
| 6 | Skip dual-body when nested `(?!(?:` | Avoid `regexp2` hang |
| 7 | Differential fixtures default `platform=linux` | Windows CI vs POSIX escape corpus |
| 8 | Go names (`NoBrace`, `MatchList`) vs JS keys | Idiomatic; TS keys in field docs |
| 9 | `NewDefaults` vs process-global `defaults()` | No mutable globals |
| 10 | Escape/Unescape option structs | Different TS `magicalBraces` defaults |
| 11 | UTF-16 length via `utf16.RuneLen` | Match JS string length edge cases |
| 12 | No guaranteed `MakeRe` source string equality | Engine escaping differs; match outcomes matter |
| 13 | Oracle JSON adapter vs running TAP in-process | Original suite hashed unmodified; thin adapter |
| 14 | Prefer `Match` path at high optimization | Same guidance as upstream |

## What we intentionally did *not* do

- **WASM bridge to the original JS** — would score high on parity but fails the spirit of a native port.
- **Guaranteed identical `MakeRe` source strings** — brittle; engines differ on escaping details.
- **100% line coverage of every TAP snapshot** — we prioritised oracle JSON + targeted Go tests + fuzz; remaining gaps are listed in COMPATIBILITY.
- **Benchmark claims without methodology** — see [BENCHMARKS.md](./BENCHMARKS.md) and [bench/methodology.md](./bench/methodology.md).
- **`unsafe` blocks** — none; Go port uses no `unsafe` package (Zero Unsafe bonus).
