#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 1 ]]; then
  echo "usage: integration.sh PATH_TO_BINARY" >&2
  exit 64
fi

binary=$1
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
out=${INTEGRATION_OUTPUT_ROOT:?INTEGRATION_OUTPUT_ROOT is required}
mkdir -p "$out"
"$binary" project \
  --source "$root/.gooo/operational-provenance-projector.gooo" \
  --contract "$root/contracts/denominator-v1.json" \
  --fixture "$root/fixtures/canonical-v1.json" \
  --history "$root/fixtures/v0.49-static-validation-history.json" \
  --root "$root" \
  --output "$out/projection" > "$out/command-output.json"
jq -e '.decision == "CLOSED" and .metrics.denominator == 12 and .receipt_chain.valid and .replay.deterministic' "$out/projection/report.json" >/dev/null
jq -e '.schema == "gooo/operational-provenance-projector/artifact-manifest/v1" and ([.artifacts[].path] | index("events.ndjson")) != null' "$out/projection/artifact-manifest.json" >/dev/null
jq -n --arg digest "$(jq -r '.artifact_digests["events.ndjson"]' "$out/projection/report.json")" \
  '{schema:"gooo/operational-provenance-projector/integration/v1",decision:"CLOSED",events_digest:$digest,source_repository_writes:0,local_test_executions:0,cross_project_required_gates:0}' > "$out/integration.json"

