# Original test suite (kickoff pin)

The **unmodified** isaacs/minimatch test suite lives in the reference checkout:

```text
../minimatch/test/          # when sibling tree is present
# or: https://github.com/isaacs/minimatch/tree/ded1bbd01beca62a5978bc1650ed0a1ccf9039d5/test
```

| Pin | Value |
| --- | --- |
| Commit | `ded1bbd01beca62a5978bc1650ed0a1ccf9039d5` |
| Version | `10.2.6` |
| Content SHA-256 (all `test/` files, sorted concat) | `660356c6ed36073248173d61b79134fbc02b6a1b7af2532102a1b55c26be7515` |

Per-file digests: [SHA256SUMS](./SHA256SUMS).

## How this port uses the original suite

We **do not edit** the upstream TAP tests. Parity is proven by:

1. **Generated oracles** — `testdata/patterns_node.json` (196 cases from `patterns.js`) and `testdata/tricky_negations.json` (46 cases), produced by reading the reference implementation’s behaviour.
2. **Live Node oracle** — `testdata/oracle.mjs` + `TestFuzzDifferentialBatch` / `fuzz/harness.go`.
3. **Go unit tests** — subsystem and Windows/UNC coverage.

Re-verify kickoff hashes (from a clean reference tree at that commit):

```bash
cd ../minimatch && git rev-parse HEAD   # expect ded1bbd…
find test -type f | sort | xargs cat | shasum -a 256
```
