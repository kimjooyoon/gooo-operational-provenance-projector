package provenance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadGraph(path string) (SemanticGraph, error) {
	digest, data, err := DigestFile(path)
	if err != nil {
		return SemanticGraph{}, err
	}
	graph := SemanticGraph{
		Schema: SchemaMeta, Precedence: []string{}, AuthorityLayers: []string{},
		EventKinds: []string{}, CommandClasses: []string{}, AttemptStates: []string{},
		ReceiptChainRules: []string{}, UnknownFields: []string{}, Indicators: []Indicator{},
		MetricPolicies: []string{}, ImprovementPolicy: []string{}, OperationalHistory: []string{},
		Activities: []string{}, Cases: []SemanticCase{}, ForbiddenEffects: []string{},
		SourcePath: path, SourceDigest: digest,
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "program":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: program requires one value", lineNumber) }
			graph.Program = fields[1]
		case "namespace":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: namespace requires one value", lineNumber) }
			graph.Namespace = fields[1]
		case "schema":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: schema requires one value", lineNumber) }
			graph.Schema = fields[1]
		case "precedence":
			graph.Precedence = strings.Fields(strings.ReplaceAll(strings.TrimPrefix(line, "precedence "), ">", " "))
		case "authority_layer":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: authority_layer requires one value", lineNumber) }
			graph.AuthorityLayers = append(graph.AuthorityLayers, fields[1])
		case "event_kind":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: event_kind requires one value", lineNumber) }
			graph.EventKinds = append(graph.EventKinds, fields[1])
		case "command_class":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: command_class requires one value", lineNumber) }
			graph.CommandClasses = append(graph.CommandClasses, fields[1])
		case "attempt_state":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: attempt_state requires one value", lineNumber) }
			graph.AttemptStates = append(graph.AttemptStates, fields[1])
		case "receipt_chain":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: receipt_chain requires one value", lineNumber) }
			graph.ReceiptChainRules = append(graph.ReceiptChainRules, fields[1])
		case "unknown_field":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: unknown_field requires one value", lineNumber) }
			graph.UnknownFields = append(graph.UnknownFields, fields[1])
		case "indicator":
			if len(fields) != 3 { return SemanticGraph{}, fmt.Errorf("line %d: indicator requires name and kind", lineNumber) }
			graph.Indicators = append(graph.Indicators, Indicator{Name: fields[1], Kind: fields[2]})
		case "metric_policy":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: metric_policy requires one value", lineNumber) }
			graph.MetricPolicies = append(graph.MetricPolicies, fields[1])
		case "improvement_policy":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: improvement_policy requires one value", lineNumber) }
			graph.ImprovementPolicy = append(graph.ImprovementPolicy, fields[1])
		case "operational_history":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: operational_history requires one value", lineNumber) }
			graph.OperationalHistory = append(graph.OperationalHistory, fields[1])
		case "activity":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: activity requires one value", lineNumber) }
			graph.Activities = append(graph.Activities, fields[1])
		case "case":
			if len(fields) != 4 { return SemanticGraph{}, fmt.Errorf("line %d: case requires id, expected state, and kind", lineNumber) }
			graph.Cases = append(graph.Cases, SemanticCase{ID: fields[1], Expected: fields[2], Kind: fields[3]})
		case "forbid_effect":
			if len(fields) != 2 { return SemanticGraph{}, fmt.Errorf("line %d: forbid_effect requires one value", lineNumber) }
			graph.ForbiddenEffects = append(graph.ForbiddenEffects, fields[1])
		default:
			return SemanticGraph{}, fmt.Errorf("line %d: unknown semantic directive %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil { return SemanticGraph{}, err }
	if err := validateGraph(graph); err != nil { return SemanticGraph{}, err }
	return graph, nil
}

func LoadContract(path string) (Contract, string, error) {
	digest, data, err := DigestFile(path)
	if err != nil { return Contract{}, "", err }
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil { return Contract{}, "", err }
	if err := validateContract(contract); err != nil { return Contract{}, "", err }
	return contract, digest, nil
}

func LoadFixture(path string, contract Contract) (Fixture, string, error) {
	digest, data, err := DigestFile(path)
	if err != nil { return Fixture{}, "", err }
	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil { return Fixture{}, "", err }
	if fixture.Schema != SchemaFixture || !fixture.Fixed || len(fixture.Cases) != len(contract.Cases) {
		return Fixture{}, "", fmt.Errorf("fixture is not the fixed operational provenance fixture")
	}
	for index, expected := range contract.Cases {
		actual := fixture.Cases[index]
		if actual.ID != expected.ID {
			return Fixture{}, "", fmt.Errorf("fixture case %d is %q, expected %q", index+1, actual.ID, expected.ID)
		}
	}
	return fixture, digest, nil
}

func LoadOperationalHistory(path string) (OperationalHistory, error) {
	data, err := os.ReadFile(path)
	if err != nil { return OperationalHistory{}, err }
	var history OperationalHistory
	if err := json.Unmarshal(data, &history); err != nil { return OperationalHistory{}, err }
	if history.Immutable != true || history.State != "OPERATIONAL_REFUTED" || history.ExactCount != 5 || len(history.Commands) != 5 || !history.PreserveExistingRefuted || history.ExecutedByCurrentRuntime {
		return OperationalHistory{}, fmt.Errorf("operational history is not an immutable five-command OPERATIONAL_REFUTED fixture")
	}
	return history, nil
}

func validateGraph(graph SemanticGraph) error {
	if graph.Schema != SchemaMeta || graph.Program == "" || graph.Namespace == "" { return fmt.Errorf("semantic graph header is incomplete") }
	if strings.Join(graph.Precedence, ",") != "REFUTED,UNKNOWN,CLOSED" { return fmt.Errorf("semantic precedence must be REFUTED,UNKNOWN,CLOSED") }
	if !exactStrings(graph.AuthorityLayers, []string{"CI_RUNTIME", "OPERATOR_AUTHORING", "ORCHESTRATOR_LOCAL"}) { return fmt.Errorf("authority layer set is not exact") }
	if !containsAll(graph.EventKinds, []string{"CI_EXECUTION", "OPERATOR_EVENT", "LOCAL_AUTHORING", "LOCAL_VALIDATION", "EMPTY_AUTHORITATIVE_SET"}) { return fmt.Errorf("event kind set is incomplete") }
	if !containsAll(graph.CommandClasses, []string{"CI_VALIDATION", "OPERATOR_AUTHORING", "AUTHORING_ONLY", "VALIDATION", "UNKNOWN"}) { return fmt.Errorf("command class set is incomplete") }
	if !containsAll(graph.AttemptStates, []string{"ATTEMPTS", "SUCCESS", "FAILURE", "UNKNOWN"}) { return fmt.Errorf("attempt state set is incomplete") }
	if !containsAll(graph.ReceiptChainRules, []string{"append_only", "previous_digest_required", "event_digest_required", "deterministic_replay"}) { return fmt.Errorf("receipt chain rules are incomplete") }
	if !exactStrings(graph.UnknownFields, []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}) { return fmt.Errorf("UNKNOWN field set is not exact") }
	if len(graph.Cases) != 12 { return fmt.Errorf("semantic denominator must contain exactly 12 cases") }
	counts := map[string]int{}
	for _, item := range graph.Cases { counts[item.Expected]++ }
	if counts[DecisionClosed] != 4 || counts[DecisionUnknown] != 4 || counts[DecisionRefuted] != 4 { return fmt.Errorf("semantic denominator must contain four CLOSED, four UNKNOWN, and four REFUTED cases") }
	if len(graph.Activities) != 12 { return fmt.Errorf("semantic activity set must contain exactly 12 activities") }
	if len(graph.ForbiddenEffects) != 7 { return fmt.Errorf("semantic forbidden effect set is incomplete") }
	if len(graph.Indicators) != 7 { return fmt.Errorf("semantic indicator vector must contain exactly seven indicators") }
	if !containsAll(graph.MetricPolicies, []string{"missing-is-null", "exact-zero-requires-authoritative-empty-receipt", "no-score-percentage-average"}) { return fmt.Errorf("metric policy is incomplete") }
	if !containsAll(graph.ImprovementPolicy, []string{"exact-same-scenario-input-contract-fixture-toolchain-runner-job", "otherwise-UNKNOWN"}) { return fmt.Errorf("improvement policy is incomplete") }
	if !containsAll(graph.OperationalHistory, []string{"v0.49-optional-immutable-fixture", "existing-refuted-preserved"}) { return fmt.Errorf("operational history policy is incomplete") }
	return nil
}

func validateContract(contract Contract) error {
	if contract.Schema != SchemaContract || contract.ID == "" || contract.Version != "1" || !contract.Fixed { return fmt.Errorf("invalid fixed denominator contract") }
	if strings.Join(contract.Precedence, ",") != "REFUTED,UNKNOWN,CLOSED" { return fmt.Errorf("contract precedence is invalid") }
	if !exactStrings(contract.RequiredUnknown, []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}) { return fmt.Errorf("contract UNKNOWN fields are invalid") }
	if len(contract.Cases) != 12 { return fmt.Errorf("contract denominator must contain exactly 12 cases") }
	counts := map[string]int{}
	for _, item := range contract.Cases { counts[item.Expected]++ }
	if counts[DecisionClosed] != 4 || counts[DecisionUnknown] != 4 || counts[DecisionRefuted] != 4 { return fmt.Errorf("contract denominator state counts are invalid") }
	return nil
}

func exactStrings(actual, expected []string) bool {
	if len(actual) != len(expected) { return false }
	for index := range expected { if actual[index] != expected[index] { return false } }
	return true
}

func containsAll(actual, expected []string) bool {
	for _, wanted := range expected {
		found := false
		for _, value := range actual { if value == wanted { found = true; break } }
		if !found { return false }
	}
	return true
}

