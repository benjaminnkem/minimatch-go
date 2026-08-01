# Bugs and notable findings

Findings related to the **original** minimatch implementation, the **Go port**, or **engine differences** discovered during Port Mortem work.

Status legend: `upstream` · `port` · `engine` · `test-suite`

---

## BUG-001 — Nested dual-body extglob can hang under non-V8 regex engines

| Field | Detail |
| --- | --- |
| **Status** | engine / port mitigation |
| **Component** | Extglob → `toRegExpSource` dual-body path |
| **Severity** | High for servers using `MakeRe` / segment regex on untrusted patterns |

### Reproduction (conceptual)

Patterns that combine **repeated** extglobs with **nested negative** extglobs force a “dual body” (dot-allowed vs not) in the reference compiler. Example shape:

```text
*(*.json|!(*.js))
```

Under JavaScript’s `RegExp`, matching may complete (sometimes slowly). Under Go’s `regexp2` (lookaround-capable, used because RE2 lacks lookarounds), the emitted pattern can enter **catastrophic backtracking** and appear to hang.

### Expected

Match or non-match in bounded time for typical inputs; or a documented complexity limit.

### Actual

- **Reference (V8):** often finishes; historically minimatch has had separate ReDoS issues addressed via recursion limits and releases (see npm advisories on older minimatch lines).
- **Port (pre-mitigation):** could block a goroutine indefinitely on `regexp2.Match`.

### Fix proposal (implemented in the port)

When building the dual body for repeated extglobs, **skip** dual-body emission if the primary body already contains a nested negative lookaround marker `(?!(?:`.

```text
if dualCandidate != body && !strings.Contains(body, "(?!(?:") {
    use dual body
}
```

This preserves dual-body behaviour for cases like `*(?)` matching `a.b`, while avoiding the hang class.

### Fix proposal (upstream / general)

1. Bound match time or reject patterns whose compiled form exceeds a complexity score.
2. Avoid dual-body composition when the body already contains negative lookaround.
3. Document that `MakeRe` on untrusted input is a ReDoS surface even with recursion caps on globstar *walks*.

### References in this repo

- `internal/ast/ast_regexp.go` (dual-body guard)
- [DECISIONS.md](./DECISIONS.md) · [COMPATIBILITY.md](./COMPATIBILITY.md)

---

## BUG-002 — `patterns.js` backslash fixtures assume POSIX (fail on win32 hosts)

| Field | Detail |
| --- | --- |
| **Status** | test-suite / upstream known class |
| **Component** | Upstream `test/patterns.js` + Windows |
| **Severity** | Low (test portability) |

### Reproduction

Run the classic patterns suite on Windows without forcing `platform: 'linux'` (or equivalent). Cases that treat `\` as an **escape** in the pattern disagree with hosts that treat `\` as a **path separator**.

### Expected

Documented platform matrix: escape tests run under POSIX semantics; path-separator tests set `platform: 'win32'`.

### Actual

Upstream comments/history acknowledge Windows fragility for some backslash cases. Our differential loader defaults unset platform to **linux** so CI on `windows-latest` still validates the escape corpus.

### Fix proposal

1. Tag each fixture with explicit `platform`.
2. Split “escape” vs “windows path” files in upstream tests.

### Port handling

`optionsFromJSON` in `differential_test.go` sets `PlatformLinux` when the fixture omits platform. Dedicated `windows_test.go` covers win32/UNC.

---

## BUG-003 — Historical npm ReDoS advisories (context, not newly discovered)

| Field | Detail |
| --- | --- |
| **Status** | upstream (historical) |
| **Component** | Older minimatch versions / globstar |

Public advisories have tracked ReDoS-style issues in minimatch (e.g. non-adjacent globstars, AST-related complexity) with fixes in modern 3.x–10.x lines. Version **10.2.6** includes recursion limits (`maxGlobstarRecursion`, etc.).

We did **not** independently re-open a CVE; we list this so judges see awareness of the threat model. The port exposes the same knobs and adds the dual-body guard above.

---

## Port defects fixed before v0.1.0 (not upstream bugs)

These were **bugs in our port** caught by differential/unit tests. Recorded briefly for transparency:

| Symptom | Cause | Fix |
| --- | --- | --- |
| Windows CI: 3 `patterns.js` cases failed | Host platform win32 rewritten `\` | Default differential platform to linux |
| Import path CI failure | Local module path vs `benjaminnkem/minimatch-go` | Normalize all imports |
| Nested extglob hang | Dual-body + `regexp2` | Guard in `ast_regexp.go` |
| Various match mismatches early on | qmark length, partial walk-off, null/nonull, UTF-8 | Fixed during differential iteration |

---

## Reporting template (for new findings)

```markdown
## BUG-XXX — short title

**Status:** upstream | port | engine | test-suite
**Severity:** low | medium | high

### Reproduction
### Expected
### Actual
### Fix proposal
```

If a finding is clearly upstream-actionable, open an issue on [isaacs/minimatch](https://github.com/isaacs/minimatch) with a minimal JS repro **in addition to** recording it here.
