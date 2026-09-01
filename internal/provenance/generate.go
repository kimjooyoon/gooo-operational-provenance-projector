package provenance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Generate(source, contractPath, fixturePath, historyPath, output, root string) (Report, error) {
	if !filepath.IsAbs(output) {
		return Report{}, fmt.Errorf("output must be an absolute caller-owned directory")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	outputAbs, err := filepath.Abs(output)
	if err != nil {
		return Report{}, err
	}
	if isWithin(outputAbs, rootAbs) || outputAbs == rootAbs {
		return Report{}, fmt.Errorf("output must be outside the source repository")
	}
	if err := ensureEmptyOutput(outputAbs); err != nil {
		return Report{}, err
	}

	graph, err := LoadGraph(source)
	if err != nil {
		return Report{}, err
	}
	contract, contractDigest, err := LoadContract(contractPath)
	if err != nil {
		return Report{}, err
	}
	fixture, fixtureDigest, err := LoadFixture(fixturePath, contract)
	if err != nil {
		return Report{}, err
	}
	history, err := LoadOperationalHistory(historyPath)
	if err != nil {
		return Report{}, err
	}
	inv, err := inventory(rootAbs, outputAbs)
	if err != nil {
		return Report{}, err
	}

	first, err := buildProjection(graph, contract, fixture, history, contractDigest, fixtureDigest, inv)
	if err != nil {
		return Report{}, err
	}
	second, err := buildProjection(graph, contract, fixture, history, contractDigest, fixtureDigest, inv)
	if err != nil {
		return Report{}, err
	}
	replay := ReplayReport{
		Requested:     true,
		EventsMatch:   bytes.Equal(first.EventNDJSON, second.EventNDJSON),
		ReceiptsMatch: equalByteMaps(first.Receipts, second.Receipts) && equalByteMaps(first.LayerReceipts, second.LayerReceipts),
		ReportsMatch:  equalReports(first.Report, second.Report),
	}
	replay.Deterministic = replay.EventsMatch && replay.ReceiptsMatch && replay.ReportsMatch
	first.Report.Replay = replay
	if !replay.Deterministic {
		first.Report.Decision = DecisionRefuted
	}
	first.Human = renderHuman(first.Report, history)

	if err := writeProjection(outputAbs, first); err != nil {
		return Report{}, err
	}
	if err := VerifyConformance(first.Report); err != nil {
		return Report{}, err
	}
	return first.Report, nil
}

func Check(source, contractPath, fixturePath string) (SemanticGraph, Contract, Fixture, error) {
	graph, err := LoadGraph(source)
	if err != nil {
		return SemanticGraph{}, Contract{}, Fixture{}, err
	}
	contract, _, err := LoadContract(contractPath)
	if err != nil {
		return SemanticGraph{}, Contract{}, Fixture{}, err
	}
	fixture, _, err := LoadFixture(fixturePath, contract)
	if err != nil {
		return SemanticGraph{}, Contract{}, Fixture{}, err
	}
	return graph, contract, fixture, nil
}

func buildProjection(graph SemanticGraph, contract Contract, fixture Fixture, history OperationalHistory, contractDigest, fixtureDigest string, inv InventoryReport) (Projection, error) {
	if len(graph.Cases) != len(contract.Cases) || len(contract.Cases) != len(fixture.Cases) {
		return Projection{}, fmt.Errorf("graph, contract, and fixture denominators do not have the same size")
	}
	if !semanticCasesMatch(graph.Cases, contract.Cases) {
		return Projection{}, fmt.Errorf(".gooo semantic cases and denominator contract disagree")
	}
	graphDigest, _, err := DigestJSON(graph)
	if err != nil {
		return Projection{}, err
	}
	events := make([]Event, 0, len(fixture.Cases))
	receipts := make([]LayerReceipt, 0, len(fixture.Cases))
	cases := make([]CaseReport, 0, len(fixture.Cases))
	var eventLines bytes.Buffer
	previousEventDigest := ""
	previousReceiptDigest := ""
	for index, input := range fixture.Cases {
		if input.ID != contract.Cases[index].ID {
			return Projection{}, fmt.Errorf("fixture order mismatch at case %d", index+1)
		}
		result, event, receipt := evaluateCase(input, contract.Cases[index], index+1, previousEventDigest, previousReceiptDigest)
		cases = append(cases, result)
		events = append(events, event)
		receipts = append(receipts, receipt)
		line, err := json.Marshal(event)
		if err != nil {
			return Projection{}, err
		}
		eventLines.Write(line)
		eventLines.WriteByte('\n')
		previousEventDigest = event.EventDigest
		previousReceiptDigest = receipt.ReceiptDigest
	}

	stateCounts := summarize(cases)
	decision := DecisionRefuted
	if stateCounts == (StateCounts{Closed: 4, Unknown: 4, Refuted: 4}) && expectedStatesMatch(cases, contract.Cases) {
		decision = DecisionClosed
	}
	authority := authorityReport(fixture)
	utility := UtilityReport{Status: DecisionUnknown, Unknown: UnknownRecord{
		Stage: "UTILITY", Step: "collect-independent-user-workload-evidence",
		Reason:        "external utility has no independent user evidence",
		UnknownClass:  "INDEPENDENT_USER_EVIDENCE_MISSING",
		NextOperation: fixture.Utility.NextOperation,
		BlockedBy:     []string{"independent_user_workload"},
	}}
	if utility.Unknown.NextOperation == "" {
		utility.Unknown.NextOperation = "collect-independent-user-workload-evidence"
	}
	operational := OperationalAudit{
		State: history.State, ExactCount: history.ExactCount, Commands: append([]string(nil), history.Commands...),
		Source: history.Source, PreservedExistingRefuted: history.PreserveExistingRefuted,
		ExecutedByCurrentRuntime: history.ExecutedByCurrentRuntime, Reason: history.Reason,
	}
	base := Report{
		Schema: "gooo/operational-provenance-projector/report/v1", Decision: decision,
		OperatorAPIAttempts: cloneInt(fixture.OperatorAPIAttempts), OperatorAPIAttemptsState: DecisionUnknown,
		Contract: contract.ID, ContractDigest: contractDigest, Fixture: fixture.ID, FixtureDigest: fixtureDigest,
		SemanticGraphDigest: graphDigest, Toolchain: Toolchain, Runner: Runner, Cases: cases,
		Summary: stateCounts, Metrics: ExactMetrics{Denominator: len(cases), States: stateCounts, EventCount: len(events), ReceiptCount: len(receipts), RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0},
		Authority: authority, OperationalAudit: operational, Utility: utility, ReceiptChain: chainReport(receipts),
		Replay: ReplayReport{Requested: true}, Inventory: inv, ArtifactDigests: map[string]string{},
	}
	generated := map[string][]byte{}
	semanticData, err := canonicalJSON(graph)
	if err != nil {
		return Projection{}, err
	}
	generated["semantic-ir.json"] = semanticData
	contractData, err := canonicalJSON(contract)
	if err != nil {
		return Projection{}, err
	}
	generated["contract-view.json"] = contractData
	contradictionData, err := contradictionReport(cases)
	if err != nil {
		return Projection{}, err
	}
	layerReceipts, err := buildLayerReceiptIndexes(receipts)
	if err != nil {
		return Projection{}, err
	}
	artifactDigests := map[string]string{"events.ndjson": DigestBytes(eventLines.Bytes()), "contradiction-report.json": DigestBytes(contradictionData)}
	for key, data := range generated {
		artifactDigests[filepath.ToSlash(filepath.Join("generated", key))] = DigestBytes(data)
	}
	for _, receipt := range receipts {
		artifactDigests[filepath.ToSlash(filepath.Join("receipts", receipt.CaseID+".json"))] = DigestBytes(mustCanonical(receipt))
	}
	for key, data := range layerReceipts {
		artifactDigests[filepath.ToSlash(filepath.Join("layer-receipts", key))] = DigestBytes(data)
	}
	base.ArtifactDigests = artifactDigests
	return Projection{Report: base, Events: events, EventNDJSON: eventLines.Bytes(), Receipts: receiptBytes(receipts), LayerReceipts: layerReceipts, Generated: generated, Contradiction: contradictionData, ReceiptObjects: receipts}, nil
}

func authorityReport(fixture Fixture) AuthorityReport {
	report := AuthorityReport{
		Layers:           []string{"CI_RUNTIME", "OPERATOR_AUTHORING", "ORCHESTRATOR_LOCAL"},
		RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0,
		Runtime:  RuntimeAuthority{SourceRepositoryWrites: 0, Commit: 0, Merge: 0, Tag: 0, Release: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0, OutputScope: "CALLER_OWNED_OUTPUT_OUTSIDE_SOURCE_REPOSITORY"},
		Operator: OperatorAuthority{OperatorAPIAttempts: cloneInt(fixture.OperatorAPIAttempts), OperatorAPIAttemptsState: DecisionUnknown},
	}
	for _, item := range fixture.Cases {
		switch item.Layer {
		case "OPERATOR_AUTHORING":
			report.Operator.AuthoringEvents++
		case "ORCHESTRATOR_LOCAL":
			if item.EventKind == "LOCAL_VALIDATION" || item.CommandClass == "VALIDATION" {
				report.Orchestrator.LocalValidationEvents++
			} else {
				report.Orchestrator.LocalAuthoringEvents++
			}
		}
	}
	return report
}

func expectedStatesMatch(actual []CaseReport, expected []ContractCase) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range expected {
		if actual[index].ID != expected[index].ID || actual[index].Expected != expected[index].Expected {
			return false
		}
	}
	return true
}

func semanticCasesMatch(semantic []SemanticCase, contract []ContractCase) bool {
	if len(semantic) != len(contract) {
		return false
	}
	for index := range contract {
		if semantic[index].ID != contract[index].ID || semantic[index].Expected != contract[index].Expected || semantic[index].Kind != contract[index].Kind {
			return false
		}
	}
	return true
}

func contradictionReport(cases []CaseReport) ([]byte, error) {
	refuted := make([]CaseReport, 0)
	for _, item := range cases {
		if item.Decision == DecisionRefuted {
			refuted = append(refuted, item)
		}
	}
	value := struct {
		Schema                   string       `json:"schema"`
		Precedence               []string     `json:"precedence"`
		Preserved                bool         `json:"preserved"`
		ExistingRefutedPreserved bool         `json:"existing_refuted_preserved"`
		RefutedCases             []CaseReport `json:"refuted_cases"`
	}{"gooo/operational-provenance-projector/contradiction-report/v1", []string{"REFUTED", "UNKNOWN", "CLOSED"}, true, true, refuted}
	return canonicalJSON(value)
}

func receiptBytes(receipts []LayerReceipt) map[string][]byte {
	result := map[string][]byte{}
	for _, receipt := range receipts {
		result[receipt.CaseID+".json"] = mustCanonical(receipt)
	}
	return result
}

func buildLayerReceiptIndexes(receipts []LayerReceipt) (map[string][]byte, error) {
	byLayer := map[string][]LayerReceipt{}
	for _, receipt := range receipts {
		byLayer[receipt.Layer] = append(byLayer[receipt.Layer], receipt)
	}
	result := map[string][]byte{}
	layers := make([]string, 0, len(byLayer))
	for layer := range byLayer {
		layers = append(layers, layer)
	}
	sort.Strings(layers)
	for _, layer := range layers {
		items := byLayer[layer]
		index := LayerReceiptIndex{Schema: "gooo/operational-provenance-projector/layer-receipts/v1", Layer: layer, CaseIDs: []string{}, EventIDs: []string{}, ReceiptDigests: []string{}, ReceiptCount: len(items)}
		for _, item := range items {
			index.CaseIDs = append(index.CaseIDs, item.CaseID)
			index.EventIDs = append(index.EventIDs, item.EventID)
			index.ReceiptDigests = append(index.ReceiptDigests, item.ReceiptDigest)
		}
		data, err := canonicalJSON(index)
		if err != nil {
			return nil, err
		}
		result[strings.ToLower(strings.ReplaceAll(layer, "_", "-"))+".json"] = data
	}
	return result, nil
}

func equalByteMaps(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if !bytes.Equal(value, right[key]) {
			return false
		}
	}
	return true
}

func equalReports(left, right Report) bool {
	left.Replay = ReplayReport{Requested: true}
	right.Replay = ReplayReport{Requested: true}
	leftData, errLeft := canonicalJSON(left)
	rightData, errRight := canonicalJSON(right)
	return errLeft == nil && errRight == nil && bytes.Equal(leftData, rightData)
}

func mustCanonical(value any) []byte {
	data, err := canonicalJSON(value)
	if err != nil {
		return []byte("{}\n")
	}
	return data
}

func renderHuman(report Report, history OperationalHistory) []byte {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Operational provenance projector report\n\nDecision: `%s`\n\n", report.Decision)
	fmt.Fprintf(&builder, "Fixed denominator: %d cases (`CLOSED=%d`, `UNKNOWN=%d`, `REFUTED=%d`).\n\n", report.Metrics.Denominator, report.Summary.Closed, report.Summary.Unknown, report.Summary.Refuted)
	builder.WriteString("## Authority boundary\n\n")
	fmt.Fprintf(&builder, "Runtime authority: source repository writes=%d, commit=%d, merge=%d, tag=%d, release=%d, local test executions=%d, cross-project required gates=%d.\n\n", report.Authority.Runtime.SourceRepositoryWrites, report.Authority.Runtime.Commit, report.Authority.Runtime.Merge, report.Authority.Runtime.Tag, report.Authority.Runtime.Release, report.Authority.Runtime.LocalTestExecutions, report.Authority.Runtime.CrossProjectRequiredGates)
	fmt.Fprintf(&builder, "Operator API attempts: %v (%s). Operator authoring events: %d. Orchestrator local authoring events: %d; local validation events: %d.\n\n", report.Authority.Operator.OperatorAPIAttempts, report.Authority.Operator.OperatorAPIAttemptsState, report.Authority.Operator.AuthoringEvents, report.Authority.Orchestrator.LocalAuthoringEvents, report.Authority.Orchestrator.LocalValidationEvents)
	builder.WriteString("## Cases\n\n")
	for _, item := range report.Cases {
		fmt.Fprintf(&builder, "- `%s`: %s; layer=%s; event=%s; command=%s; reason=%s.\n", item.ID, item.Decision, item.Layer, item.EventKind, item.CommandClass, item.Reason)
	}
	builder.WriteString("\n## UNKNOWN records\n\n")
	for _, item := range report.Cases {
		if item.Unknown != nil {
			fmt.Fprintf(&builder, "- `%s`: stage=%s, step=%s, class=%s, next=%s, blocked_by=%s.\n", item.ID, item.Unknown.Stage, item.Unknown.Step, item.Unknown.UnknownClass, item.Unknown.NextOperation, strings.Join(item.Unknown.BlockedBy, ","))
		}
	}
	builder.WriteString("\n## Receipt chain and replay\n\n")
	fmt.Fprintf(&builder, "Receipt chain length=%d, append-only=%t, valid=%t, head=%s, tail=%s. Replay deterministic=%t (events=%t, receipts=%t, reports=%t).\n\n", report.ReceiptChain.Length, report.ReceiptChain.AppendOnly, report.ReceiptChain.Valid, report.ReceiptChain.HeadDigest, report.ReceiptChain.TailDigest, report.Replay.Deterministic, report.Replay.EventsMatch, report.Replay.ReceiptsMatch, report.Replay.ReportsMatch)
	builder.WriteString("## Preserved v0.49 operational history\n\n")
	fmt.Fprintf(&builder, "State=`%s`, exact local command count=%d, immutable=%t, existing REFUTED preserved=%t, executed by current runtime=%t.\n\n", history.State, history.ExactCount, history.Immutable, history.PreserveExistingRefuted, history.ExecutedByCurrentRuntime)
	for index, command := range history.Commands {
		fmt.Fprintf(&builder, "%d. `%s`\n", index+1, command)
	}
	builder.WriteString("\nThe v0.49 operational history is an optional immutable observation. It is not reclassified or erased by this projector. Utility remains UNKNOWN without independent user evidence.\n")
	return []byte(builder.String())
}

func ensureEmptyOutput(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0o755)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("output exists and is not a directory")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory must be empty")
	}
	return nil
}

func writeProjection(output string, projection Projection) error {
	if err := os.MkdirAll(filepath.Join(output, "generated"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(output, "receipts"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(output, "layer-receipts"), 0o755); err != nil {
		return err
	}
	eventsPath := filepath.Join(output, "events.ndjson")
	file, err := os.OpenFile(eventsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(projection.EventNDJSON); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	for _, key := range sortedKeys(projection.Receipts) {
		if err := os.WriteFile(filepath.Join(output, "receipts", key), projection.Receipts[key], 0o644); err != nil {
			return err
		}
	}
	for _, key := range sortedKeys(projection.LayerReceipts) {
		if err := os.WriteFile(filepath.Join(output, "layer-receipts", key), projection.LayerReceipts[key], 0o644); err != nil {
			return err
		}
	}
	for _, key := range sortedKeys(projection.Generated) {
		if err := os.WriteFile(filepath.Join(output, "generated", key), projection.Generated[key], 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(output, "contradiction-report.json"), projection.Contradiction, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "human-report.md"), projection.Human, 0o644); err != nil {
		return err
	}
	reportData, err := canonicalJSON(projection.Report)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "report.json"), reportData, 0o644); err != nil {
		return err
	}
	artifacts := make([]Artifact, 0)
	paths := []string{"events.ndjson", "contradiction-report.json", "human-report.md", "report.json"}
	for _, key := range sortedKeys(projection.Receipts) {
		paths = append(paths, filepath.ToSlash(filepath.Join("receipts", key)))
	}
	for _, key := range sortedKeys(projection.LayerReceipts) {
		paths = append(paths, filepath.ToSlash(filepath.Join("layer-receipts", key)))
	}
	for _, key := range sortedKeys(projection.Generated) {
		paths = append(paths, filepath.ToSlash(filepath.Join("generated", key)))
	}
	for _, relative := range paths {
		data, err := os.ReadFile(filepath.Join(output, filepath.FromSlash(relative)))
		if err != nil {
			return err
		}
		artifacts = append(artifacts, Artifact{Path: relative, Bytes: int64(len(data)), Digest: DigestBytes(data)})
	}
	manifest, err := canonicalJSON(struct {
		Schema    string     `json:"schema"`
		Artifacts []Artifact `json:"artifacts"`
	}{"gooo/operational-provenance-projector/artifact-manifest/v1", artifacts})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "artifact-manifest.json"), manifest, 0o644)
}

func VerifyConformance(report Report) error {
	if report.Schema != "gooo/operational-provenance-projector/report/v1" || report.Decision != DecisionClosed {
		return fmt.Errorf("report is not CLOSED")
	}
	if report.Metrics.Denominator != 12 || report.Summary != (StateCounts{Closed: 4, Unknown: 4, Refuted: 4}) {
		return fmt.Errorf("report does not preserve exact 12-case denominator")
	}
	if len(report.Cases) != 12 {
		return fmt.Errorf("report case count is not 12")
	}
	for _, item := range report.Cases {
		if item.Decision == DecisionUnknown {
			if item.Unknown == nil || item.Unknown.Stage == "" || item.Unknown.Step == "" || item.Unknown.Reason == "" || item.Unknown.UnknownClass == "" || item.Unknown.NextOperation == "" || len(item.Unknown.BlockedBy) == 0 {
				return fmt.Errorf("UNKNOWN case %s does not contain the six required fields", item.ID)
			}
		}
		if item.Decision == DecisionRefuted && len(item.Refutations) == 0 {
			return fmt.Errorf("REFUTED case %s has no contradiction record", item.ID)
		}
	}
	if report.ReceiptChain.Length != 12 || !report.ReceiptChain.AppendOnly || !report.ReceiptChain.Valid {
		return fmt.Errorf("receipt chain is invalid")
	}
	if !report.Replay.Requested || !report.Replay.Deterministic || !report.Replay.EventsMatch || !report.Replay.ReceiptsMatch || !report.Replay.ReportsMatch {
		return fmt.Errorf("deterministic replay is not CLOSED")
	}
	runtime := report.Authority.Runtime
	if runtime.SourceRepositoryWrites != 0 || runtime.Commit != 0 || runtime.Merge != 0 || runtime.Tag != 0 || runtime.Release != 0 || runtime.LocalTestExecutions != 0 || runtime.CrossProjectRequiredGates != 0 {
		return fmt.Errorf("runtime authority is not zero")
	}
	if report.Authority.RepositoryWrites != 0 || report.Authority.LocalTestExecutions != 0 || report.Authority.CrossProjectRequiredGates != 0 {
		return fmt.Errorf("flat runtime authority is not zero")
	}
	if report.Authority.Operator.OperatorAPIAttempts != nil || report.Authority.Operator.OperatorAPIAttemptsState != DecisionUnknown {
		return fmt.Errorf("operator API attempts must remain null plus UNKNOWN")
	}
	if report.OperatorAPIAttempts != nil || report.OperatorAPIAttemptsState != DecisionUnknown {
		return fmt.Errorf("top-level operator API attempts must remain null plus UNKNOWN")
	}
	if report.OperationalAudit.State != "OPERATIONAL_REFUTED" || report.OperationalAudit.ExactCount != 5 || len(report.OperationalAudit.Commands) != 5 || !report.OperationalAudit.PreservedExistingRefuted || report.OperationalAudit.ExecutedByCurrentRuntime {
		return fmt.Errorf("v0.49 operational history was not preserved")
	}
	if report.Utility.Status != DecisionUnknown || report.Utility.Unknown.Stage == "" || report.Utility.Unknown.Step == "" || report.Utility.Unknown.Reason == "" || report.Utility.Unknown.UnknownClass == "" || report.Utility.Unknown.NextOperation == "" || len(report.Utility.Unknown.BlockedBy) == 0 {
		return fmt.Errorf("utility must remain UNKNOWN with six fields")
	}
	return nil
}
