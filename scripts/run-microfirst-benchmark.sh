#!/usr/bin/env bash
set -euo pipefail

# Day 10 — Micro-Model First benchmark runner.
# Builds embedding prototypes once, then runs the two-level inference benchmark
# and prints the resulting report.
#
# Requires: OPENAI_API_KEY in env, Go toolchain, network access to api.openai.com.

if [[ -z "${OPENAI_API_KEY:-}" ]]; then
  echo "ERROR: OPENAI_API_KEY is not set" >&2
  echo "Export it first:  export OPENAI_API_KEY=sk-..." >&2
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVICE_DIR="$REPO_ROOT/services/api-dashboard"
PROTO_FILE="$REPO_ROOT/finetune/confidence-check/data/prototypes.json"
REPORT_FILE="$REPO_ROOT/finetune/confidence-check/results/report_microfirst.md"

cd "$SERVICE_DIR"

if [[ "${REBUILD_PROTOTYPES:-0}" == "1" ]] || [[ ! -f "$PROTO_FILE" ]]; then
  echo ">>> Building embedding prototypes ..."
  BUILD_PROTOTYPES=1 go test -v -run TestBuildPrototypes ./finetune/
  echo ">>> Prototypes written: $PROTO_FILE"
else
  echo ">>> Prototypes already exist at: $PROTO_FILE"
  echo ">>> Set REBUILD_PROTOTYPES=1 to rebuild."
fi

echo ""
echo ">>> Running micro-first benchmark ..."
go test -v -run TestMicroFirstBenchmark ./finetune/ -timeout 15m

echo ""
echo "============================================================"
echo "REPORT: $REPORT_FILE"
echo "============================================================"
if [[ -f "$REPORT_FILE" ]]; then
  cat "$REPORT_FILE"
else
  echo "ERROR: report file was not produced" >&2
  exit 1
fi
