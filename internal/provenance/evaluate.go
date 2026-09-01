package provenance

import (
	"fmt"
	"sort"
	"strings"
)

func evaluateCase(input FixtureCase, expected ContractCase, sequence int, previousEventDigest, previousReceiptDigest string) (CaseReport, Event, LayerReceipt) {
	decision := DecisionUnknown
	reason := "EVIDENCE_NOT_BOUND"
	var unknown *UnknownRecord
	refutations := []RefutationRecord{}

	if input.RepositoryWrites != 0 {
		refutations = append(refutations, RefutationRecord{
			Stage: "AUTHORITY", Step: "classify-source-repository-effects",
			Reason: "WRITE_ESCALATION", RefutationClass: "WRITE_ESCALATION",
			Evidence:      fmt.Sprintf("repository_writes=%d", input.RepositoryWrites),
			NextOperation: "retain-refuted-event-and-remove-escalated-effect",
			BlockedBy:     []string{"source_repository_write"},
		})
	}
	if isCountContradiction(input.Counts) {
		refutations = append(refutations, RefutationRecord{
			Stage: "METRICS", Step: "reconcile-attempt-counts",
			Reason: "CONTRADICTORY_COUNTS", RefutationClass: "COUNT_CONTRADICTION",
			Evidence:      fmt.Sprintf("attempts=%d success=%d failure=%d unknown=%d", countValue(input.Counts.Attempts), countValue(input.Counts.Success), countValue(input.Counts.Failure), countValue(input.Counts.Unknown)),
			NextOperation: "reconcile-counts-from-authoritative-receipt",
			BlockedBy:     []string{"attempts", "success", "failure", "unknown"},
		})
	}
	if input.EventKind == "LOCAL_VALIDATION" || input.CommandClass == "VALIDATION" || (input.CommandClass == "AUTHORING_ONLY" && strings.Contains(strings.ToLower(input.Observation), "validation")) {
		refutations = append(refutations, RefutationRecord{
			Stage: "COMMAND_CLASSIFICATION", Step: "distinguish-validation-from-authoring",
			Reason: "VALIDATION_DISGUISED_AS_AUTHORING", RefutationClass: "VALIDATION_DISGUISED_AS_AUTHORING",
			Evidence:      fmt.Sprintf("event_kind=%s command_class=%s", input.EventKind, input.CommandClass),
			NextOperation: "reclassify-event-as-validation",
			BlockedBy:     []string{"event_kind", "command_class"},
		})
	}
	if input.Layer == "CI_RUNTIME" && input.Signed && input.AuthorityProof != "GITHUB_ACTIONS" {
		refutations = append(refutations, RefutationRecord{
			Stage: "AUTHORITY", Step: "verify-ci-signature-authority",
			Reason: "FORGED_AUTHORITY", RefutationClass: "FORGED_AUTHORITY",
			Evidence:      fmt.Sprintf("authority_proof=%s", input.AuthorityProof),
			NextOperation: "obtain-trusted-github-actions-receipt",
			BlockedBy:     []string{"ci_signature_authority"},
		})
	}
	if input.Layer == "OPERATOR_AUTHORING" && input.Explicit && input.AuthorityProof != "OPERATOR_EXPLICIT" {
		refutations = append(refutations, RefutationRecord{
			Stage: "AUTHORITY", Step: "verify-explicit-operator-event",
			Reason: "FORGED_AUTHORITY", RefutationClass: "FORGED_AUTHORITY",
			Evidence:      fmt.Sprintf("authority_proof=%s", input.AuthorityProof),
			NextOperation: "obtain-explicit-operator-receipt",
			BlockedBy:     []string{"operator_event_authority"},
		})
	}

	event := buildEvent(input, sequence, previousEventDigest)
	if len(refutations) > 0 {
		decision = DecisionRefuted
		reason = refutations[0].Reason
	} else if !input.Receipt.Present {
		decision = DecisionUnknown
		reason = "MISSING_RECEIPT"
		unknown = unknownRecord("RECEIPT", "locate-layer-receipt", "receipt is missing; attempts are not observable", "MISSING_RECEIPT", "obtain-authoritative-layer-receipt", []string{"layer_receipt"})
	} else if !input.Receipt.Fresh {
		decision = DecisionUnknown
		reason = "STALE_RECEIPT"
		unknown = unknownRecord("RECEIPT", "check-receipt-freshness", "receipt exists but is stale for the current input", "STALE_RECEIPT", "refresh-layer-receipt-for-current-input", []string{"receipt_freshness", "fixture_digest"})
	} else if input.Layer == "AMBIGUOUS" || input.CommandClass == "UNKNOWN" || input.AuthorityProof == "UNRESOLVED" {
		decision = DecisionUnknown
		reason = "AMBIGUOUS_AUTHORITY_LAYER"
		unknown = unknownRecord("AUTHORITY", "resolve-authority-layer", "available evidence cannot select one authority layer", "AMBIGUOUS_LAYER", "bind-event-to-one-authority-layer", []string{"authority_layer"})
	} else if !input.Bounded {
		decision = DecisionUnknown
		reason = "UNBOUNDED_OBSERVATION"
		unknown = unknownRecord("OBSERVATION", "close-observation-window", "observation has no bounded end", "UNBOUNDED_OBSERVATION", "declare-bounded-observation-window", []string{"observation_boundary"})
	} else if !isComplete(input.Counts) {
		decision = DecisionUnknown
		reason = "MISSING_ATTEMPTS"
		unknown = unknownRecord("METRICS", "obtain-attempt-vector", "attempts, success, failure, or unknown is unavailable", "MISSING_ATTEMPTS", "obtain-complete-authoritative-attempt-vector", []string{"attempts", "success", "failure", "unknown"})
	} else if exactZero(input.Counts) && input.Receipt.AuthoritativeEmpty && input.EventKind == "EMPTY_AUTHORITATIVE_SET" {
		decision = DecisionClosed
		reason = "AUTHORITATIVE_EMPTY_RECEIPT"
	} else if input.Layer == "CI_RUNTIME" && input.EventKind == "CI_EXECUTION" && input.Signed && input.HasDigest && input.AuthorityProof == "GITHUB_ACTIONS" {
		decision = DecisionClosed
		reason = "SIGNED_DIGESTED_CI_EVENT"
	} else if input.Layer == "OPERATOR_AUTHORING" && input.EventKind == "OPERATOR_EVENT" && input.Explicit && input.AuthorityProof == "OPERATOR_EXPLICIT" {
		decision = DecisionClosed
		reason = "EXPLICIT_OPERATOR_EVENT"
	} else if input.Layer == "ORCHESTRATOR_LOCAL" && input.EventKind == "LOCAL_AUTHORING" && input.CommandClass == "AUTHORING_ONLY" {
		decision = DecisionClosed
		reason = "CLASSIFIED_AUTHORING_ONLY_LOCAL_EVENT"
	} else {
		decision = DecisionUnknown
		reason = "UNRESOLVED_PROVENANCE"
		unknown = unknownRecord("PROVENANCE", "resolve-event-provenance", "event does not satisfy a CLOSED proof rule", "UNRESOLVED_PROVENANCE", "supply-authoritative-layer-and-receipt", []string{"authority_layer", "event_receipt"})
	}

	receipt := buildReceipt(input, event, decision, previousReceiptDigest)
	result := CaseReport{
		ID: input.ID, Expected: expected.Expected, Decision: decision, Kind: expected.Kind,
		Layer: input.Layer, EventKind: input.EventKind, CommandClass: input.CommandClass,
		Activity: input.Activity, Reason: reason, Counts: cloneCountVector(input.Counts),
		EventID: event.EventID, EventDigest: event.EventDigest, ReceiptDigest: receipt.ReceiptDigest,
		Unknown: unknown, Refutations: refutations,
	}
	return result, event, receipt
}

func buildEvent(input FixtureCase, sequence int, previousEventDigest string) Event {
	evidenceDigest := ""
	if input.HasDigest {
		evidenceDigest = DigestBytes([]byte(strings.Join([]string{input.ID, input.Layer, input.EventKind, input.CommandClass, input.Observation}, "|")))
	}
	event := Event{
		Schema: SchemaEvent, Sequence: sequence, EventID: fmt.Sprintf("event-%03d-%s", sequence, input.ID),
		CaseID: input.ID, Layer: input.Layer, EventKind: input.EventKind, CommandClass: input.CommandClass,
		Activity: input.Activity, Observation: input.Observation, Counts: cloneCountVector(input.Counts),
		Signed: input.Signed, EvidenceDigest: evidenceDigest, AuthorityProof: input.AuthorityProof,
		Bounded: input.Bounded, RepositoryWrites: input.RepositoryWrites, PreviousEventDigest: previousEventDigest,
	}
	unsigned := event
	unsigned.EventDigest = ""
	event.EventDigest = digestValue(unsigned)
	return event
}

func buildReceipt(input FixtureCase, event Event, decision, previousReceiptDigest string) LayerReceipt {
	receipt := LayerReceipt{
		Schema: SchemaReceipt, Sequence: event.Sequence, ReceiptID: fmt.Sprintf("receipt-%03d-%s", event.Sequence, input.ID),
		CaseID: input.ID, Layer: input.Layer, EventID: event.EventID, EventDigest: event.EventDigest,
		PreviousReceiptDigest: previousReceiptDigest, ReceiptState: input.Receipt.State, Counts: cloneCountVector(input.Counts),
		Decision: decision, Signed: input.Signed, EvidenceDigest: event.EvidenceDigest,
		AuthorityProof: input.AuthorityProof, Bounded: input.Bounded, RepositoryWrites: input.RepositoryWrites,
	}
	unsigned := receipt
	unsigned.ReceiptDigest = ""
	receipt.ReceiptDigest = digestValue(unsigned)
	return receipt
}

func unknownRecord(stage, step, reason, class, next string, blocked []string) *UnknownRecord {
	return &UnknownRecord{Stage: stage, Step: step, Reason: reason, UnknownClass: class, NextOperation: next, BlockedBy: blocked}
}

func digestValue(value any) string {
	digest, _, err := DigestJSON(value)
	if err != nil {
		return "sha256:error"
	}
	return digest
}

func isCountContradiction(value CountVector) bool {
	if !isComplete(value) {
		return false
	}
	if countValue(value.Attempts) < 0 || countValue(value.Success) < 0 || countValue(value.Failure) < 0 || countValue(value.Unknown) < 0 {
		return true
	}
	return countValue(value.Attempts) != countValue(value.Success)+countValue(value.Failure)+countValue(value.Unknown)
}

func exactZero(value CountVector) bool {
	return isComplete(value) && countValue(value.Attempts) == 0 && countValue(value.Success) == 0 && countValue(value.Failure) == 0 && countValue(value.Unknown) == 0
}

func summarize(cases []CaseReport) StateCounts {
	counts := StateCounts{}
	for _, item := range cases {
		switch item.Decision {
		case DecisionClosed:
			counts.Closed++
		case DecisionUnknown:
			counts.Unknown++
		case DecisionRefuted:
			counts.Refuted++
		}
	}
	return counts
}

func chainReport(receipts []LayerReceipt) ChainReport {
	report := ChainReport{Schema: "gooo/operational-provenance-projector/receipt-chain/v1", AppendOnly: true, Length: len(receipts), Valid: true}
	if len(receipts) == 0 {
		report.Valid = false
		return report
	}
	report.HeadDigest = receipts[0].ReceiptDigest
	report.TailDigest = receipts[len(receipts)-1].ReceiptDigest
	previous := ""
	for _, receipt := range receipts {
		if receipt.PreviousReceiptDigest != previous || receipt.EventDigest == "" || receipt.ReceiptDigest == "" {
			report.Valid = false
		}
		unsigned := receipt
		unsigned.ReceiptDigest = ""
		if digestValue(unsigned) != receipt.ReceiptDigest {
			report.Valid = false
		}
		previous = receipt.ReceiptDigest
	}
	return report
}

func sortedKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
