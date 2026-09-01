# Gooo operational provenance projector

This repository separates three authority layers that were previously collapsed into one number:

- `CI_RUNTIME`: signed and digested GitHub Actions execution evidence.
- `OPERATOR_AUTHORING`: explicit human/operator authoring events.
- `ORCHESTRATOR_LOCAL`: bounded local authoring activity and its command classification.

The authoritative meaning is [`.gooo/operational-provenance-projector.gooo`](.gooo/operational-provenance-projector.gooo). Go is the parser, generator, evaluator, and runtime. The fixed denominator is exactly 12 cases: `CLOSED=4`, `UNKNOWN=4`, and `REFUTED=4`; precedence is `REFUTED > UNKNOWN > CLOSED`.

Missing attempts are `null` plus `UNKNOWN`. Zero is `CLOSED` only when a bounded authoritative empty receipt proves the empty set. A stale or missing receipt, ambiguous authority layer, or unbounded observation remains `UNKNOWN` with exactly these six fields: `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.

Contradictory counts, validation disguised as authoring, forged authority, and write escalation are preserved as `REFUTED`. The four refuted fixtures are intentionally retained so a successful conformance run does not erase contradictory evidence.

## Outputs

The runtime writes only to an absolute, empty, caller-owned directory outside the source repository:

```text
gooo-operational-provenance-projector conformance \
  --source .gooo/operational-provenance-projector.gooo \
  --contract contracts/denominator-v1.json \
  --fixture fixtures/canonical-v1.json \
  --history fixtures/v0.49-static-validation-history.json \
  --root . \
  --output /absolute/path/to/empty/output
```

The output contains `events.ndjson` (append-only), per-case receipts, per-layer receipt indexes, generated semantic IR, `contradiction-report.json`, `report.json`, `human-report.md`, and an artifact manifest. The receipt chain and a second in-memory projection must replay identically before the report can be `CLOSED`.

`repository_writes=0`, `local_test_executions=0`, and `cross_project_required_gates=0` describe the projector runtime. Operator authoring is reported separately. `operator_api_attempts` is deliberately `null` plus `UNKNOWN`; the projector does not manufacture an operator API receipt.

## v0.49 operational history

[`fixtures/v0.49-static-validation-history.json`](fixtures/v0.49-static-validation-history.json) is an optional immutable observation fixture. It preserves the exact five historical local commands and their `OPERATIONAL_REFUTED` state:

1. `go test ./...`
2. `go build ./...`
3. `go vet ./...`
4. `gofmt -l $(git ls-files '*.go')`
5. `bash -n scripts/*.sh`

Those commands are not executed by the current runtime. The current repository uses GitHub Actions as its only validation authority.

## Verification and release

Pull requests and pushes to `main` use [`.github/workflows/conformance.yml`](.github/workflows/conformance.yml). It performs formatting, vet, build, tests, fixed-denominator conformance, integration, and artifact capture on the GitHub runner. No local validation result is substituted for CI evidence.

Release is manual and PR-first: first enable the repository's platform immutable-release setting as the owner, then provide the exact merged commit and its successful main CI run to [`.github/workflows/release.yml`](.github/workflows/release.yml). The workflow verifies that precondition, creates an annotated tag once, creates a draft release through the GitHub API, uploads the evidence assets, publishes once, and verifies the published `immutable=true` field, annotated tag target, asset count, and GitHub asset digests. Failed runs, tags, drafts, releases, and pull requests are retained as evidence.
