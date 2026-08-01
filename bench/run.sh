#!/usr/bin/env bash
# Run Node + Go comparative benches → bench/results.json
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

REF_DIST="../minimatch/dist/esm/index.js"
if [[ ! -f "$REF_DIST" ]]; then
  echo "missing $REF_DIST — run: (cd ../minimatch && npm run prepare)" >&2
  exit 2
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
REF_ABS="$(cd ../minimatch && pwd)/dist/esm/index.js"

echo "→ Node scenarios..." >&2
node bench/node_bench.mjs >"$TMP/node.json"

echo "→ build Go binary..." >&2
go build -o "$TMP/benchbin" ./bench

echo "→ cold startups (median of 5, discard first)..." >&2
START_GO_MS=$(python3 -c "
import subprocess, time, statistics
bin='$TMP/benchbin'
subprocess.check_call([bin, '-startup-only'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
samples=[]
for _ in range(5):
    t0=time.perf_counter()
    subprocess.check_call([bin, '-startup-only'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    samples.append((time.perf_counter()-t0)*1000)
print('%.3f' % statistics.median(samples))
")
START_NODE_MS=$(python3 - "$REF_ABS" "$TMP" <<'PY'
import subprocess, time, sys, pathlib, statistics
ref, tmp = sys.argv[1], pathlib.Path(sys.argv[2])
p = tmp / "s_node.mjs"
p.write_text(
    "import { pathToFileURL } from 'url';\n"
    f"const {{ minimatch }} = await import(pathToFileURL({ref!r}).href);\n"
    "minimatch('foo/bar.js', '**/*.js', { platform: 'linux' });\n"
)
def once():
    t0 = time.perf_counter()
    subprocess.check_call(["node", str(p)], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    return (time.perf_counter()-t0)*1000
once()
print(f"{statistics.median([once() for _ in range(5)]):.3f}")
PY
)

echo "→ Go scenarios..." >&2
"$TMP/benchbin" >"$TMP/go.json"

export START_NODE_MS START_GO_MS TMP
python3 <<'PY'
import json, platform, subprocess, os, datetime
from pathlib import Path

node = json.loads(Path(os.environ["TMP"], "node.json").read_text())
go = json.loads(Path(os.environ["TMP"], "go.json").read_text())
start_node = float(os.environ["START_NODE_MS"])
start_go = float(os.environ["START_GO_MS"])

node.setdefault("scenarios", {})["startup_cold"] = {
    "startup_ms": start_node,
    "note": "cold process: node load + one Match; median of 5 after discard-1",
}
go.setdefault("scenarios", {})["startup_cold"] = {
    "startup_ms": start_go,
    "note": "cold process: prebuilt Go binary + one Match; median of 5 after discard-1",
}

try:
    uname = subprocess.check_output(["uname", "-a"], text=True).strip()
except Exception:
    uname = platform.platform()
try:
    cpu = subprocess.check_output(["sysctl", "-n", "machdep.cpu.brand_string"], text=True).strip()
except Exception:
    cpu = platform.processor() or "unknown"

comparison = {}
for name in sorted(set(node.get("scenarios", {})) | set(go.get("scenarios", {}))):
    ns = node.get("scenarios", {}).get(name, {})
    gs = go.get("scenarios", {}).get(name, {})
    row = {"node": ns, "go": gs}
    if ns.get("ns_per_op_mean") and gs.get("ns_per_op_mean"):
        row["speedup_go_vs_node_mean"] = ns["ns_per_op_mean"] / gs["ns_per_op_mean"]
    if ns.get("ns_per_op_p99") and gs.get("ns_per_op_p99"):
        row["speedup_go_vs_node_p99"] = ns["ns_per_op_p99"] / gs["ns_per_op_p99"]
    if "startup_ms" in ns or "startup_ms" in gs:
        row["node_startup_ms"] = ns.get("startup_ms")
        row["go_startup_ms"] = gs.get("startup_ms")
    comparison[name] = row

out = {
    "meta": {
        "generated_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        "hardware": {"uname": uname, "cpu": cpu},
        "methodology": "bench/methodology.md",
        "shared_api": "Match / Minimatch.match / BraceExpand",
        "options": {"platform": "linux"},
    },
    "node": node,
    "go": go,
    "comparison": comparison,
}

path = Path("bench/results.json")
path.write_text(json.dumps(out, indent=2) + "\n")
print(f"wrote {path}")

print("\n=== comparative mean ns/op (lower is better) ===")
print(f"{'scenario':<28} {'node':>12} {'go':>12} {'speedup':>10}")
for name, row in sorted(comparison.items()):
    n, g = row.get("node", {}), row.get("go", {})
    if "ns_per_op_mean" not in n or "ns_per_op_mean" not in g:
        continue
    sp = row.get("speedup_go_vs_node_mean")
    print(f"{name:<28} {n['ns_per_op_mean']:>12.0f} {g['ns_per_op_mean']:>12.0f} {sp:>9.2f}x")

print(f"\n=== p99 ns/op ===")
print(f"{'scenario':<28} {'node_p99':>12} {'go_p99':>12}")
for name, row in sorted(comparison.items()):
    n, g = row.get("node", {}), row.get("go", {})
    if "ns_per_op_p99" not in n:
        continue
    print(f"{name:<28} {n.get('ns_per_op_p99', 0):>12.0f} {g.get('ns_per_op_p99', 0):>12.0f}")

print(f"\nstartup_cold: node={start_node:.2f} ms | go_binary={start_go:.2f} ms")
print(
    f"memory: node_rss={node.get('memory',{}).get('rss_bytes')} "
    f"node_heap={node.get('memory',{}).get('heap_bytes')} "
    f"go_heap={go.get('memory',{}).get('heap_bytes')}"
)
PY

echo "done." >&2
