#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 5 ]]; then
  echo "usage: assemble-ci-evidence.sh STAGE_DIR GO_TEST_EVENTS COUNTS INTEGRATION PAIR" >&2
  exit 64
fi

stage_dir=$1
test_events=$2
counts=$3
integration=$4
pair=$5
output=${CI_EVIDENCE_OUT:?CI_EVIDENCE_OUT is required}
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
run_id=${GITHUB_RUN_ID:-unknown}
job_id=${GITHUB_JOB:-unknown}
events_count=0
if [[ -s "$test_events" ]]; then events_count=$(wc -l < "$test_events" | tr -d ' '); fi
stages=$(jq -n \
  --slurpfile compile "$stage_dir/compile.json" \
  --slurpfile build "$stage_dir/build.json" \
  --slurpfile test_stage "$stage_dir/test.json" \
  --slurpfile conformance "$stage_dir/conformance.json" \
  --slurpfile integration_stage "$stage_dir/integration.json" \
  '{compile:$compile[0],build:$build[0],test:$test_stage[0],conformance:$conformance[0],integration:$integration_stage[0]}')
jq -n \
  --arg schema "gooo/operational-provenance-projector/ci-evidence/v1" \
  --arg run_id "$run_id" --arg job_id "$job_id" --argjson test_events "$events_count" \
  --argjson stages "$stages" --slurpfile report "$CONFORMANCE_WORK_ROOT/projection/report.json" \
  --slurpfile counts "$counts" --slurpfile integration "$integration" --slurpfile pair "$pair" \
  --arg source_digest "$(sha256sum "$root/.gooo/operational-provenance-projector.gooo" | awk '{print "sha256:"$1}')" \
  '{schema:$schema,verification_authority:"GITHUB_ACTIONS",run_id:$run_id,job_id:$job_id,source_digest:$source_digest,toolchain:$report[0].toolchain,runner:$report[0].runner,stages:$stages,go_test_event_lines:$test_events,denominator:$counts[0],integration:$integration[0],exact_metrics:$pair[0],report_summary:$report[0].summary,receipt_chain:$report[0].receipt_chain,replay:$report[0].replay,inventory:$report[0].inventory,operational_audit:$report[0].operational_audit,authority:{runtime:$report[0].authority.runtime,operator:$report[0].authority.operator,orchestrator:$report[0].authority.orchestrator}}' > "$output"

