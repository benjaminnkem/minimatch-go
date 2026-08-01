# Changelog

## [0.1.0] — 2026-08-01

First tagged release of the Go port of [isaacs/minimatch](https://github.com/isaacs/minimatch).

### Features

- `Match`, `NewMinimatch`, `MatchList`, `Filter`, `MakeRe`
- `BraceExpand`, `Escape`, `Unescape`, `ParseGlob` / AST
- `NewDefaults` for stacked default options
- Options aligned with TypeScript `MinimatchOptions` (including Windows/UNC via `Platform`)
- Production layout: public API at module root, implementation under `internal/`

### Parity

- Differential suite vs Node: **196** `patterns.js` cases, **46** `tricky-negations` cases
- Random Node oracle batch + fuzz (`FuzzMatchNoPanic`)
- Windows/UNC behavioural tests (`PlatformWin32`)

### Notes

- Prefer `Match` over `MakeRe` when `optimizationLevel >= 2` (same guidance as upstream).
- `patterns.js` backslash-escape fixtures assume POSIX; differential tests force `platform=linux` when unset so CI on Windows hosts stays consistent. Real win32 path behaviour is covered separately.
- Segment/full-path regexes use [regexp2](https://github.com/dlclark/regexp2) for lookaround support.
