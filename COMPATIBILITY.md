# Compatibility

What this Go port supports relative to **isaacs/minimatch@10.2.6**
(`ded1bbd01beca62a5978bc1650ed0a1ccf9039d5`).

## Support matrix

### Supported (intended parity)

| Feature | Notes |
| --- | --- |
| Basic globs `*`, `?`, `**` | Segment-wise match; `NoGlobStar` flattens `**` → `*` |
| Brace expansion `{a,b}`, `{1..n}` | `internal/brace`; `NoBrace`, `BraceExpandMax` |
| Negation `!pattern` | Leading `!` toggle; `NoNegate`, `FlipNegate` |
| Comments `#…` | `NoComment` |
| Character classes `[abc]`, ranges, escapes | `internal/class` |
| POSIX classes `[[:alpha:]]` etc. | Via class parser |
| Extglob `!(…)`, `*(…)`, `+(…)`, `?(…)`, `@(…)` | AST parse, flatten, fillNegs, compile |
| `Dot` / leading-dot rules | Including start-of-segment no-dot prefixes in regexes |
| `NoCase` / `NoCaseMagicOnly` | Case folding aligned with reference intent |
| `MatchBase` | Basename match when pattern has no `/` |
| `Partial` | Prefix match for walks |
| `optimizationLevel` 0 / 1 / ≥2 | Pointer field; default 1 |
| Windows / UNC / drives | `PlatformWin32`, path `\` → `/` normalization |
| `WindowsPathsNoEscape` / `AllowWindowsEscape` | Escape vs separator semantics |
| `WindowsNoMagicRoot` | Tri-state pointer |
| `PreserveMultipleSlashes` | Option wired |
| `MagicalBraces` | Option + Escape/Unescape defaults |
| Escape / Unescape | Public API |
| `MakeRe` / full-path regexp | Via `regexp2` |
| `Match`, compiled `Minimatch`, `MatchList`, `Filter` | Primary API |
| `NewDefaults` | Stacked defaults helper |
| Pattern length limit | 64KiB UTF-16 units (same as reference) |
| Recursion limits | Globstar / extglob / brace expand maxima |

### Unsupported or out of scope

| Item | Status |
| --- | --- |
| Exact `RegExp` source string equality with JS | **Not** guaranteed |
| Browser / ESM bundle of the Go port | N/A (Go library) |
| Drop-in identical function *names* (`minimatch()` default export) | Go idioms (`Match`, `NewMinimatch`) |
| Process-global mutable `minimatch.defaults` singleton | Replaced by `NewDefaults` |
| Streaming filesystem walker | minimatch is a matcher; walkers are out of scope (same as upstream library role) |

### Behavioural differences (known)

| Topic | TypeScript | Go port | Impact |
| --- | --- | --- | --- |
| Invalid pattern | Throws | Returns `error` | Call sites must check `err` |
| Nested dual-body + `!(…)` lookarounds | May complete under V8 (or run long) | Dual body **skipped** when body contains `(?!(?:` to avoid `regexp2` hang | Rare nested extglob forms |
| Differential fixture platform | Often host-dependent in Node tests | JSON fixtures default to **linux** unless `platform` set | CI stability; win32 tested separately |
| `MakeRe` pattern text | V8-oriented escaping | `regexp2`-oriented source | Prefer `Match` for parity-critical paths |
| Error types | `TypeError` / `Error` | Sentinel / wrapped Go errors | Message text kept where tests require it |

---

## Feature coverage vs upstream tests

| Oracle / suite | Cases | Result (local) |
| --- | --- | --- |
| `patterns.js` → `testdata/patterns_node.json` | **196** | Differential PASS |
| `tricky-negations` → `testdata/tricky_negations.json` | **46** | Differential PASS |
| Random Node oracle batch | **80** / run (`TestFuzzDifferentialBatch`) | PASS when Node + reference present |
| Windows / UNC unit tests | Dedicated Go tests | PASS |
| Go unit tests (`go test ./...`) | Package + `internal/*` | PASS |

Commands: see [TESTING.md](./TESTING.md).

---

## Options name mapping

| TypeScript (`MinimatchOptions`) | Go (`Options`) |
| --- | --- |
| `nobrace` | `NoBrace` |
| `nocomment` | `NoComment` |
| `nonegate` | `NoNegate` |
| `debug` | `Debug` |
| `noglobstar` | `NoGlobStar` |
| `noext` | `NoExt` |
| `nonull` | `NoNull` |
| `windowsPathsNoEscape` | `WindowsPathsNoEscape` |
| `allowWindowsEscape` | `AllowWindowsEscape` (`*bool`) |
| `partial` | `Partial` |
| `dot` | `Dot` |
| `nocase` | `NoCase` |
| `nocaseMagicOnly` | `NoCaseMagicOnly` |
| `magicalBraces` | `MagicalBraces` |
| `matchBase` | `MatchBase` |
| `flipNegate` | `FlipNegate` |
| `preserveMultipleSlashes` | `PreserveMultipleSlashes` |
| `optimizationLevel` | `OptimizationLevel` (`*int`) |
| `platform` | `Platform` |
| `windowsNoMagicRoot` | `WindowsNoMagicRoot` (`*bool`) |
| `braceExpandMax` | `BraceExpandMax` (`*int`) |
| `maxGlobstarRecursion` | `MaxGlobstarRecursion` (`*int`) |
| `maxExtglobRecursion` | `MaxExtglobRecursion` (`*int`) |

---

## Recommended usage for maximum parity

```go
// Prefer compiled matcher for repeated matches.
m, err := minimatch.NewMinimatch("**/*.{js,ts}", minimatch.Options{
    Dot: true,
})
if err != nil { /* … */ }
ok := m.Match(path)

// Force POSIX path/escape semantics regardless of host OS:
opts := minimatch.Options{Platform: minimatch.PlatformLinux}

// Windows path matching:
opts = minimatch.Options{Platform: minimatch.PlatformWin32, NoCase: true}
```

Avoid relying on the exact string returned by `MakeRe` when comparing to Node; compare **match outcomes** instead.
