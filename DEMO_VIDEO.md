# Demo video script (≤ 5 minutes)

Port Mortem deliverable: *show the original suite / oracle passing against the port live*.

## Recording setup

- Terminal font large; repo root = `minimatch-go/`
- Sibling `../minimatch` present and built (`npm run prepare`)
- Optional: split screen README + terminal

## Script (~4:30)

### 0:00 – 0:30 · Hook

> “This is a Port Mortem Track C port: **isaacs/minimatch** TypeScript → **Go**.  
> Goal: behavioural parity, not a line-by-line translate. Original tests untouched.”

Show: GitHub README + `.port-mortem.toml` (track C, commit pin).

### 0:30 – 1:00 · One-command build

```bash
docker build -t minimatch-port .
# if Docker unavailable on the recording machine:
make check
```

> “One command builds and verifies.”

### 1:00 – 2:15 · Differential parity (north star)

```bash
make differential
# expect: 196 patterns + 46 tricky-negations PASS
```

```bash
go test -run TestFuzzDifferentialBatch -v
# expect: 80 random cases match Node
```

Show briefly: `testdata/patterns_node.json` header + `tests/original/SHA256SUMS` (kickoff pin).

> “We don’t edit the TAP suite. We hash it at kickoff and drive oracles from the reference binary.”

### 2:15 – 3:15 · Differential fuzz survivor

```bash
make fuzz-diff
# or: go run ./fuzz -duration=60s -out=fuzz/log.txt
```

Scroll `fuzz/log.txt` summary: `divergences: 0`, `result: PASS`.

### 3:15 – 4:00 · Benchmarks (honest)

```bash
make bench-compare
```

Show table from stdout / `bench/results.json`: mean + **p99**, startup node vs go binary.

> “We report p99 and startup, not just hot-loop averages. Here’s where Go wins and where Node still leads.”

### 4:00 – 4:30 · Decisions + close

Flash `DECISIONS.md` (regexp2, dual-body hang guard, platform defaults).  
Optional: `BUGS.md` BUG-001.

> “Unsafe count: N/A in Go — zero `unsafe` blocks. Port is a pure Go module.  
> Repo: github.com/benjaminnkem/minimatch-go — thanks.”

## B-roll checklist

- [ ] `make check` green  
- [ ] `fuzz/log.txt` with ≥60s and zero divergences  
- [ ] `bench/results.json` committed or regenerated on camera  
- [ ] No secrets / private paths  

## Upload

Title suggestion: `Port Mortem 2026 — minimatch TS→Go (Track C) demo`  
Link the public GitHub repo in the description.
