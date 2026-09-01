#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 1 ]]; then
  echo "usage: conformance.sh PATH_TO_BINARY" >&2
  exit 64
fi

binary=$1
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work=${CONFORMANCE_WORK_ROOT:?CONFORMANCE_WORK_ROOT is required}
counts_out=${CONFORMANCE_COUNTS_OUT:?CONFORMANCE_COUNTS_OUT is required}
pair_out=${PROVENANCE_PAIR_OUT:?PROVENANCE_PAIR_OUT is required}
mkdir -p "$work" "$(dirname "$counts_out")" "$(dirname "$pair_out")"
projection="$work/projection"

"$binary" conformance \
  --source "$root/.gooo/operational-provenance-projector.gooo" \
  --contract "$root/contracts/denominator-v1.json" \
  --fixture "$root/fixtures/canonical-v1.json" \
  --history "$root/fixtures/v0.49-static-validation-history.json" \
  --root "$root" \
  --output "$projection" > "$work/command-output.json"

jq -e '
  .schema == "gooo/operational-provenance-projector/report/v1" and
  .decision == "CLOSED" and
  .metrics.denominator == 12 and
  .summary == {closed:4,unknown:4,refuted:4} and
  (.cases | length) == 12 and
  ([.cases[].decision] | sort) == (["CLOSED","CLOSED","CLOSED","CLOSED","REFUTED","REFUTED","REFUTED","REFUTED","UNKNOWN","UNKNOWN","UNKNOWN","UNKNOWN"] | sort) and
  .authority.runtime == {source_repository_writes:0,commit:0,merge:0,tag:0,release:0,local_test_executions:0,cross_project_required_gates:0,output_scope:"CALLER_OWNED_OUTPUT_OUTSIDE_SOURCE_REPOSITORY"} and
  .authority.repository_writes == 0 and .authority.local_test_executions == 0 and .authority.cross_project_required_gates == 0 and
  .authority.operator.operator_api_attempts == null and
  .authority.operator.operator_api_attempts_state == "UNKNOWN" and
  .operator_api_attempts == null and
  .operator_api_attempts_state == "UNKNOWN" and
  .operational_audit.state == "OPERATIONAL_REFUTED" and
  .operational_audit.exact_count == 5 and
  (.operational_audit.commands | length) == 5 and
  .operational_audit.preserved_existing_refuted == true and
  .operational_audit.executed_by_current_runtime == false and
  .receipt_chain.append_only == true and .receipt_chain.valid == true and
  .replay.deterministic == true and
  .utility.status == "UNKNOWN"
' "$projection/report.json" >/dev/null

jq -e '
  ([.cases[] | select(.decision == "UNKNOWN") | .unknown |
    (.stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by | length) > 0)] | all) and
  ([.cases[] | select(.decision == "REFUTED") | .refutations | length] | all(. > 0)) and
  ([.cases[] | select(.id == "unknown-missing-receipt") | .counts.attempts] == [null]) and
  ([.cases[] | select(.id == "closed-exact-zero-empty-receipt") | .reason] == ["AUTHORITATIVE_EMPTY_RECEIPT"]) and
  ([.cases[] | select(.id == "refuted-contradictory-counts") | .reason] == ["CONTRADICTORY_COUNTS"]) and
  ([.cases[] | select(.id == "refuted-validation-disguised-as-authoring") | .reason] == ["VALIDATION_DISGUISED_AS_AUTHORING"]) and
  ([.cases[] | select(.id == "refuted-forged-authority") | .reason] == ["FORGED_AUTHORITY"]) and
  ([.cases[] | select(.id == "refuted-write-escalation") | .reason] == ["WRITE_ESCALATION"])
' "$projection/report.json" >/dev/null

test "$(wc -l < "$projection/events.ndjson" | tr -d ' ')" = 12
test -f "$projection/receipts/closed-ci-signed-digested.json"
test -f "$projection/layer-receipts/ci-runtime.json"
test -f "$projection/generated/semantic-ir.json"
test -f "$projection/contradiction-report.json"
test -f "$projection/human-report.md"
jq -e '.preserved == true and .existing_refuted_preserved == true and (.refuted_cases | length) == 4' "$projection/contradiction-report.json" >/dev/null

jq -n --slurpfile report "$projection/report.json" \
  '{schema:"gooo/operational-provenance-projector/conformance-counts/v1",denominator:$report[0].metrics.denominator,closed:$report[0].summary.closed,unknown:$report[0].summary.unknown,refuted:$report[0].summary.refuted,repository_writes:$report[0].authority.runtime.source_repository_writes,local_test_executions:$report[0].authority.runtime.local_test_executions,cross_project_required_gates:$report[0].authority.runtime.cross_project_required_gates}' > "$counts_out"

jq -n --slurpfile report "$projection/report.json" \
  '{schema:"gooo/operational-provenance-projector/exact-metrics/v1",scenario:"canonical-operational-provenance-v1",input:"canonical-v1.json",contract:"denominator-v1.json",fixture:"canonical-v1.json",toolchain:$report[0].toolchain,runner:$report[0].runner,job:((env.GITHUB_RUN_ID // "unknown") + "/" + (env.GITHUB_JOB // "unknown")),indicator_vector:[{name:"attempts",value:null,state:"UNKNOWN"},{name:"success",value:null,state:"UNKNOWN"},{name:"failure",value:null,state:"UNKNOWN"},{name:"unknown",value:null,state:"UNKNOWN"},{name:"repository_writes",value:0,state:"CLOSED"},{name:"local_test_executions",value:0,state:"CLOSED"},{name:"cross_project_required_gates",value:0,state:"CLOSED"}],improvement:{status:"UNKNOWN",reason:"NO_EXACT_SAME_SCENARIO_INPUT_CONTRACT_FIXTURE_TOOLCHAIN_RUNNER_JOB_BEFORE_AFTER_PAIR"}}' > "$pair_out"
