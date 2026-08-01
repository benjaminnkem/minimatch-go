# Changelog

All notable milestones for this Port Mortem submission and Go module
(`github.com/benjaminnkem/minimatch-go`).

## [0.1.0] — 2026-08-01

First tagged release of the Go port.

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

### Hardening

- Nested dual-body extglob guard to avoid `regexp2` hang ([BUGS.md](./BUGS.md) BUG-001)
- Differential fixtures force `platform=linux` when unset ([BUGS.md](./BUGS.md) BUG-002)
- Module import path normalized to `github.com/benjaminnkem/minimatch-go`

### Submission docs

- In-repo documentation: `README`, `DECISIONS`, `PORT_AUDIT`, `BENCHMARKS`, `BUGS`, `FUZZING`, `COMPATIBILITY`, `ARCHITECTURE`, `TESTING`
- `Dockerfile` + `Makefile` for judge-friendly builds
- `.port-mortem.toml`, `tests/original/` kickoff hashes, `DEMO_VIDEO.md`
- `make fuzz-diff` → `fuzz/log.txt` (60s differential, zero divergences)
- `make bench-compare` → `bench/results.json` (Node vs Go, p99 + startup)

### Notes

- Prefer `Match` over `MakeRe` when `optimizationLevel >= 2` (same guidance as upstream).
- `patterns.js` backslash-escape fixtures assume POSIX; differential tests force `platform=linux` when unset so CI on Windows hosts stays consistent. Real win32 path behaviour is covered separately.
- Segment/full-path regexes use [regexp2](https://github.com/dlclark/regexp2) for lookaround support.

---

## Pre-release engineering log (summary)

| Phase | Deliverable |
| --- | --- |
| Audit | Selected isaacs/minimatch 10.2.6; measured LOC; confirmed no public Go behavioural port |
| Foundation | Module, errors, platform, validate, Options zero-value semantics |
| Scanner | Path-segment tokenizer |
| Brace | In-tree brace expansion |
| AST | Parse, flatten, fillNegs, `ToRegExpSource` / patterns |
| Matcher | Segment match, globstar, public API, MakeRe |
| Hardening | Differential, fuzz, Windows tests, CI, ReDoS-class hang fix |
| Release | Tag `v0.1.0`, changelog, README status |
