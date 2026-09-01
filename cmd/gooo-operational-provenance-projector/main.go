package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-operational-provenance-projector/internal/provenance"
)

type options struct {
	source   string
	contract string
	fixture  string
	history  string
	output   string
	root     string
	report   string
}

func main() {
	if len(os.Args) < 2 { fatal("command is required: check, project, conformance, or verify") }
	switch os.Args[1] {
	case "check": check(os.Args[2:])
	case "project": project(os.Args[2:])
	case "conformance": project(os.Args[2:])
	case "verify": verify(os.Args[2:])
	default: fatal("unknown command %q", os.Args[1])
	}
}

func parse(command string, args []string, outputRequired bool) options {
	set := flag.NewFlagSet(command, flag.ExitOnError)
	values := options{}
	set.StringVar(&values.source, "source", ".gooo/operational-provenance-projector.gooo", "authoritative .gooo graph")
	set.StringVar(&values.contract, "contract", "contracts/denominator-v1.json", "fixed denominator contract")
	set.StringVar(&values.fixture, "fixture", "fixtures/canonical-v1.json", "canonical fixture")
	set.StringVar(&values.history, "history", "fixtures/v0.49-static-validation-history.json", "optional immutable operational history")
	set.StringVar(&values.output, "output", "", "absolute empty caller-owned output directory")
	set.StringVar(&values.output, "out", "", "alias for --output")
	set.StringVar(&values.root, "root", ".", "source repository root")
	set.StringVar(&values.report, "report", "", "generated report JSON")
	if err := set.Parse(args); err != nil { fatal(err.Error()) }
	if outputRequired && values.output == "" { fatal("%s requires --output", command) }
	return values
}

func check(args []string) {
	values := parse("check", args, false)
	graph, contract, fixture, err := provenance.Check(values.source, values.contract, values.fixture)
	if err != nil { fatal(err.Error()) }
	printJSON(map[string]any{
		"schema": graph.Schema,
		"graph_digest": graph.SourceDigest,
		"contract": contract.ID,
		"fixture": fixture.ID,
		"cases": len(fixture.Cases),
		"activities": len(graph.Activities),
		"authority_layers": graph.AuthorityLayers,
	})
}

func project(args []string) {
	values := parse("project", args, true)
	report, err := provenance.Generate(values.source, values.contract, values.fixture, values.history, values.output, values.root)
	if err != nil { fatal(err.Error()) }
	printJSON(map[string]any{
		"decision": report.Decision,
		"denominator": report.Metrics.Denominator,
		"closed": report.Summary.Closed,
		"unknown": report.Summary.Unknown,
		"refuted": report.Summary.Refuted,
		"replay_deterministic": report.Replay.Deterministic,
		"output": filepath.Clean(values.output),
	})
}

func verify(args []string) {
	values := parse("verify", args, false)
	if values.report == "" { fatal("verify requires --report") }
	data, err := os.ReadFile(values.report)
	if err != nil { fatal(err.Error()) }
	var report provenance.Report
	if err := json.Unmarshal(data, &report); err != nil { fatal(err.Error()) }
	if err := provenance.VerifyConformance(report); err != nil { fatal(err.Error()) }
	printJSON(map[string]any{"decision": report.Decision, "denominator": report.Metrics.Denominator})
}

func printJSON(value any) {
	data, err := json.Marshal(value)
	if err != nil { fatal(err.Error()) }
	fmt.Println(string(data))
}

func fatal(format string, values ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}

