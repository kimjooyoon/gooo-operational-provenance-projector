# Operational provenance protocol v1

The `.gooo` graph owns authority layers, event kinds, command classes, the receipt chain, status precedence, denominator, UNKNOWN schema, metric rules, activity inventory, and forbidden effects. The Go implementation does not redefine those meanings; it materializes them into deterministic evidence.

## Authority layers

`CI_RUNTIME` proves runtime execution only through a signed event with an evidence digest and trusted `GITHUB_ACTIONS` authority. `OPERATOR_AUTHORING` closes only when an operator event is explicit and carries the operator authoring proof. `ORCHESTRATOR_LOCAL` closes only for a bounded `LOCAL_AUTHORING` event classified `AUTHORING_ONLY`. A validation event cannot be made authoring-only by its label.

The runtime authority boundary is independent of event observations. The projector itself reports zero source-repository writes, commits, merges, tags, releases, local tests, and cross-project required gates. Operator authoring events are counted in their own layer. A source write declared by an observed event is a `REFUTED` case; it does not turn the projector's runtime boundary into a write.

## Counts and receipts

The count vector is the integer tuple `(attempts, success, failure, unknown)`. A complete vector must satisfy `attempts = success + failure + unknown` and contain no negative values. A missing member remains JSON `null`; it is never coerced to zero.

The first receipt has an empty previous digest. Every later receipt carries the prior receipt digest. Each event and receipt has a SHA-256 digest over its canonical representation with its own digest field empty. `events.ndjson` is appended in sequence order. A replay uses the same source, contract, fixture, toolchain, runner, and job identity and must produce identical events, receipts, generated IR, contradiction report, and machine report.

An exact zero is closed only by a bounded `EMPTY_AUTHORITATIVE_SET` event and an authoritative empty receipt. Zero values inferred from absent or stale evidence remain `UNKNOWN`.

## Status and denominator

The fixed fixture has twelve cases, four in each decision state. Case-level `REFUTED` evidence remains visible in the contradiction report even when the conformance suite itself is `CLOSED` because the expected fixed denominator and replay contract were satisfied. The suite decision describes the integrity of the projection; case decisions describe the evidence.

Every UNKNOWN contains exactly six causal fields: `stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`. Utility is a separate UNKNOWN claim until independent user-workload evidence exists.

