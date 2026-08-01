#!/bin/sh
# Entrypoint for the minimatch-port image (build context: this repo).
set -e
cd /src

# Optional live oracle: if a reference tree is mounted at /minimatch,
# expose it as ../minimatch relative to the module (what fuzz_test expects).
if [ -d /minimatch/dist ] || [ -f /minimatch/package.json ]; then
  mkdir -p /work
  # /src is the module; parent of module should be /work with sibling minimatch
  if [ ! -e /work/minimatch-go ]; then
    ln -sfn /src /work/minimatch-go
  fi
  if [ ! -e /work/minimatch ]; then
    ln -sfn /minimatch /work/minimatch
  fi
  cd /work/minimatch-go
fi

target="${1:-check}"
case "$target" in
  test)
    go test ./...
    ;;
  vet)
    go vet ./...
    ;;
  differential)
    go test -run 'DifferentialPatterns|DifferentialPartial|DifferentialTricky' -count=1
    ;;
  bench)
    go test -bench=. -benchmem -count=1
    ;;
  fuzz)
    go test -fuzz=FuzzMatchNoPanic -fuzztime=10s
    ;;
  oracle)
    go test -run TestFuzzDifferentialBatch -v -count=1
    ;;
  check)
    go vet ./...
    go test ./...
    go test -run 'DifferentialPatterns|DifferentialPartial|DifferentialTricky' -count=1
    ;;
  all)
    go vet ./...
    go test ./...
    go test -run 'DifferentialPatterns|DifferentialPartial|DifferentialTricky' -count=1
    go test -run TestFuzzDifferentialBatch -count=1
    go test -bench=. -benchmem -count=1
    ;;
  sh|bash)
    exec /bin/bash
    ;;
  *)
    echo "unknown target: $target" >&2
    echo "usage: test|vet|differential|bench|fuzz|oracle|check|all|bash" >&2
    exit 2
    ;;
esac
