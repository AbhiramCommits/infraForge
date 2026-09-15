#!/usr/bin/env bash
# Demo: enforce cgroup v2 limits with `infraforge limit`.
#
# Requirements: Linux with cgroups v2, root, Go.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== building =="
go build -o bin/infraforge ./cmd/infraforge
go build -o bin/burner ./hack

echo
echo "== demo 1: 256M memory cap, burner allocates 512M =="
set +e
bin/infraforge limit --name demo-oom --memory-max 256M -- ./bin/burner -mem-mb 512 -cpu 0
rc=$?
set -e
echo "demo 1 exit code: $rc"

echo
echo "== demo 2: 25% CPU cap, burner spins 2 workers =="
bin/infraforge limit --name demo-cpu --cpu-max 25% -- ./bin/burner -mem-mb 32 -cpu 2 -duration 15s
echo "demo 2 exit code: 0"
