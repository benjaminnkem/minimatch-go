# Differential fuzz harness

Port Mortem **Differential Fuzz Survivor** (+5) artifact.

```bash
# from module root; requires Node + built ../minimatch
make fuzz-diff
# go run ./fuzz -duration=60s -out=fuzz/log.txt
```

| File | Role |
| --- | --- |
| `harness.go` | Compares Go `MatchList` to Node `minimatch.match` |
| `log.txt` | ≥60s run log (zero divergences for bonus claim) |

Shared API only; random globs with `platform=linux` and a subset of options (`dot`, `nocase`, `nonegate`, `noglobstar`, `matchBase`).
