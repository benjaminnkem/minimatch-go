# Port audit

Evidence that [isaacs/minimatch](https://github.com/isaacs/minimatch) is a suitable Port Mortem target and a record of the snapshot we ported.

## Original repository

| Field | Value |
| --- | --- |
| **Name** | minimatch |
| **URL** | https://github.com/isaacs/minimatch |
| **Author** | Isaac Z. Schlueter (`isaacs`) |
| **Language** | TypeScript (dual ESM/CJS build via `tshy`) |
| **Published version** | `10.2.6` (`package.json`) |
| **Commit SHA** | `ded1bbd01beca62a5978bc1650ed0a1ccf9039d5` |
| **Commit subject** | `10.2.6` |
| **Commit date** | 2026-07-27 |
| **Local path (reference)** | `../minimatch/` when checked out beside this repo (read-only) |
| **License** | Blue Oak Model License 1.0.0 (`BlueOak-1.0.0`) |

The Go port lives in this repository and targets **observable parity** with this commit, not a file-by-file transcription.

## Why this repository qualifies

1. **Non-trivial algorithms.** Glob matching is not a thin wrapper: brace expansion, path-segment scanning, character classes (including POSIX classes), extglob AST (`!(…)`, `*(…)`, `+(…)`, `?(…)`, `@(…)`), globstar recursion, negation, and full-path `MakeRe` lookarounds.
2. **Rich, public test suite.** Upstream ships dozens of TAP tests (`patterns.js`, `tricky-negations.js`, Windows/UNC, partial, ReDoS guards). That gives a concrete oracle for differential testing.
3. **Real-world surface.** minimatch underpins much of the npm tooling ecosystem (`glob`, installers, bundlers). A faithful Go port is useful beyond the hackathon.
4. **Portable licence.** Blue Oak 1.0.0 is permissive; the port uses the same licence text.
5. **Bounded but deep scope.** Source is a few thousand lines — large enough for serious engineering decisions, small enough to finish with parity evidence in a hackathon window.

## Source lines of code (measurement)

Measured with `wc -l` on the audited tree (2026-08-01). Counts are physical lines, including comments and blanks.

### TypeScript reference (`../minimatch/src` when present)

| File | LOC |
| --- | --- |
| `src/index.ts` | 1485 |
| `src/ast.ts` | 977 |
| `src/brace-expressions.ts` | 172 |
| `src/unescape.ts` | 42 |
| `src/escape.ts` | 33 |
| `src/assert-valid-pattern.ts` | 12 |
| **Total `src/`** | **2721** |

| Additional | LOC |
| --- | --- |
| Upstream `test/` (all files) | ~2088 |
| Runtime dependency `brace-expansion` | separate package (JS), reimplemented under `internal/brace` |

Compiled `dist/` and `node_modules/` are **not** counted as source LOC.

### Go port (this repo, non-test)

| Scope | LOC |
| --- | --- |
| Production `.go` in this repo (excluding `*_test.go`) | **5064** |
| Tests (`*_test.go`) | **3016** |
| **Total Go** | **8080** |

Production code is larger than the TS sources primarily because:

- Brace expansion is in-tree (TS depends on `brace-expansion`).
- Explicit errors, platform helpers, and package docs replace implicit JS behaviour.
- Subsystem tests sit beside implementation under `internal/`.

## Existing Go port search

Search date: **2026-08-01**.

| Source | Query / method | Result |
| --- | --- | --- |
| **GitHub** | `minimatch language:Go` (repo search API) | One hit: `castaneai/minimatch` — an **Open Match** (game matchmaking) project, **not** a port of isaacs/minimatch. |
| **GitHub** | `isaacs minimatch port language:Go` | **0** relevant repositories. |
| **pkg.go.dev** | package search for `minimatch` | No published module that claims behavioural parity with isaacs/minimatch. |
| **Go Modules / general knowledge** | Adjacent ecosystem | Go has capable **glob** libraries (`path/filepath.Match`, `gobwas/glob`, `bmatcuk/doublestar`, etc.). These implement subsets of shell globs; none are documented drop-in ports of minimatch’s full options matrix (extglob AST, `MakeRe` lookarounds, Windows/UNC parity, `optimizationLevel`, etc.). |

**Conclusion:** No public, maintained Go library was found that ports isaacs/minimatch with API/options parity. This project fills that gap.

## Qualification checklist

| Criterion | Status |
| --- | --- |
| Original repo identified with commit SHA | ✓ |
| Licence compatible | ✓ Blue Oak 1.0.0 |
| LOC measured and recorded | ✓ ~2721 TS `src/` |
| Existing Go port search documented | ✓ GitHub, pkg.go.dev, module ecosystem |
| Non-trivial (algorithms + edge cases) | ✓ |
| Test oracle available | ✓ upstream TAP + generated JSON |

## Snapshot policy

- The TypeScript tree (`../minimatch` when checked out beside this repo) is **read-only specification**.
- Behavioural claims are relative to commit `ded1bbd01beca62a5978bc1650ed0a1ccf9039d5` / version **10.2.6**.
- Regenerating fixtures: build the reference (`npm run prepare` in `../minimatch`), then run `make fixtures` / `node testdata/generate_patterns.mjs`.
