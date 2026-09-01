package provenance

import "encoding/json"

const (
	DecisionClosed  = "CLOSED"
	DecisionUnknown = "UNKNOWN"
	DecisionRefuted = "REFUTED"

	SchemaMeta     = "gooo/operational-provenance-projector/meta/v1"
	SchemaFixture  = "gooo/operational-provenance-projector/fixture/v1"
	SchemaContract = "gooo/operational-provenance-projector/denominator/v1"
	SchemaEvent    = "gooo/operational-provenance-projector/event/v1"
	SchemaReceipt  = "gooo/operational-provenance-projector/receipt/v1"
	Toolchain      = "go1.27.0"
	Runner         = "ubuntu-latest"
)

type SemanticCase struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Kind     string `json:"kind"`
}

type Indicator struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type SemanticGraph struct {
	Schema             string         `json:"schema"`
	Program            string         `json:"program"`
	Namespace          string         `json:"namespace"`
	Precedence         []string       `json:"precedence"`
	AuthorityLayers    []string       `json:"authority_layers"`
	EventKinds         []string       `json:"event_kinds"`
	CommandClasses     []string       `json:"command_classes"`
	AttemptStates      []string       `json:"attempt_states"`
	ReceiptChainRules  []string       `json:"receipt_chain_rules"`
	UnknownFields      []string       `json:"unknown_fields"`
	Indicators         []Indicator    `json:"indicators"`
	MetricPolicies     []string       `json:"metric_policies"`
	ImprovementPolicy  []string       `json:"improvement_policy"`
	OperationalHistory []string       `json:"operational_history"`
	Activities         []string       `json:"activities"`
	Cases              []SemanticCase `json:"cases"`
	ForbiddenEffects   []string       `json:"forbidden_effects"`
	SourcePath         string         `json:"source_path"`
	SourceDigest       string         `json:"source_digest"`
}

type ContractCase struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Kind     string `json:"kind"`
}

type Contract struct {
	Schema             string         `json:"schema"`
	ID                 string         `json:"id"`
	Version            string         `json:"version"`
	Fixed              bool           `json:"fixed"`
	Precedence         []string       `json:"precedence"`
	RequiredUnknown    []string       `json:"required_unknown_fields"`
	Cases              []ContractCase `json:"cases"`
}

type CountVector struct {
	Attempts *int64 `json:"attempts"`
	Success  *int64 `json:"success"`
	Failure  *int64 `json:"failure"`
	Unknown  *int64 `json:"unknown"`
}

type ReceiptInput struct {
	Present            bool   `json:"present"`
	Fresh              bool   `json:"fresh"`
	AuthoritativeEmpty bool   `json:"authoritative_empty"`
	State              string `json:"state"`
}

type FixtureCase struct {
	ID             string       `json:"id"`
	Layer          string       `json:"layer"`
	EventKind      string       `json:"event_kind"`
	CommandClass   string       `json:"command_class"`
	Activity       string       `json:"activity"`
	Observation    string       `json:"observation"`
	Signed         bool         `json:"signed"`
	HasDigest      bool         `json:"has_digest"`
	Explicit       bool         `json:"explicit"`
	Bounded        bool         `json:"bounded"`
	AuthorityProof string       `json:"authority_proof"`
	Receipt        ReceiptInput `json:"receipt"`
	Counts         CountVector  `json:"counts"`
	RepositoryWrites int        `json:"repository_writes"`
}

type UtilityInput struct {
	Status        string `json:"status"`
	Reason        string `json:"reason"`
	NextOperation string `json:"next_operation"`
}

type Fixture struct {
	Schema             string          `json:"schema"`
	ID                 string          `json:"id"`
	Fixed              bool            `json:"fixed"`
	OperatorAPIAttempts *int64         `json:"operator_api_attempts"`
	Cases              []FixtureCase   `json:"cases"`
	Utility            UtilityInput    `json:"utility"`
}

type OperationalHistory struct {
	Schema                 string   `json:"schema"`
	Immutable              bool     `json:"immutable"`
	Source                 string   `json:"source"`
	State                  string   `json:"state"`
	ExactCount             int      `json:"exact_count"`
	Commands               []string `json:"commands"`
	Reason                 string   `json:"reason"`
	PreserveExistingRefuted bool    `json:"preserve_existing_refuted"`
	ExecutedByCurrentRuntime bool    `json:"executed_by_current_runtime"`
}

type Event struct {
	Schema          string      `json:"schema"`
	Sequence        int         `json:"sequence"`
	EventID         string      `json:"event_id"`
	CaseID          string      `json:"case_id"`
	Layer           string      `json:"layer"`
	EventKind       string      `json:"event_kind"`
	CommandClass    string      `json:"command_class"`
	Activity        string      `json:"activity"`
	Observation     string      `json:"observation"`
	Counts          CountVector `json:"counts"`
	Signed          bool        `json:"signed"`
	EvidenceDigest  string      `json:"evidence_digest"`
	AuthorityProof  string      `json:"authority_proof"`
	Bounded         bool        `json:"bounded"`
	RepositoryWrites int        `json:"repository_writes"`
	PreviousEventDigest string  `json:"previous_event_digest"`
	EventDigest     string      `json:"event_digest"`
}

type UnknownRecord struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type RefutationRecord struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	RefutationClass string `json:"refutation_class"`
	Evidence      string   `json:"evidence"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type LayerReceipt struct {
	Schema             string       `json:"schema"`
	Sequence           int          `json:"sequence"`
	ReceiptID          string       `json:"receipt_id"`
	CaseID             string       `json:"case_id"`
	Layer              string       `json:"layer"`
	EventID            string       `json:"event_id"`
	EventDigest        string       `json:"event_digest"`
	PreviousReceiptDigest string    `json:"previous_receipt_digest"`
	ReceiptState       string       `json:"receipt_state"`
	Counts             CountVector  `json:"counts"`
	Decision           string       `json:"decision"`
	Signed             bool         `json:"signed"`
	EvidenceDigest     string       `json:"evidence_digest"`
	AuthorityProof     string       `json:"authority_proof"`
	Bounded            bool         `json:"bounded"`
	RepositoryWrites   int          `json:"repository_writes"`
	ReceiptDigest      string       `json:"receipt_digest"`
}

type LayerReceiptIndex struct {
	Schema          string   `json:"schema"`
	Layer           string   `json:"layer"`
	CaseIDs         []string `json:"case_ids"`
	EventIDs        []string `json:"event_ids"`
	ReceiptDigests  []string `json:"receipt_digests"`
	ReceiptCount    int      `json:"receipt_count"`
}

type CaseReport struct {
	ID             string            `json:"id"`
	Expected       string            `json:"expected"`
	Decision       string            `json:"decision"`
	Kind           string            `json:"kind"`
	Layer          string            `json:"layer"`
	EventKind      string            `json:"event_kind"`
	CommandClass   string            `json:"command_class"`
	Activity       string            `json:"activity"`
	Reason         string            `json:"reason"`
	Counts         CountVector       `json:"counts"`
	EventID        string            `json:"event_id"`
	EventDigest    string            `json:"event_digest"`
	ReceiptDigest  string            `json:"receipt_digest"`
	Unknown        *UnknownRecord    `json:"unknown"`
	Refutations    []RefutationRecord `json:"refutations"`
}

type StateCounts struct {
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}

type RuntimeAuthority struct {
	SourceRepositoryWrites    int    `json:"source_repository_writes"`
	Commit                    int    `json:"commit"`
	Merge                     int    `json:"merge"`
	Tag                       int    `json:"tag"`
	Release                   int    `json:"release"`
	LocalTestExecutions       int    `json:"local_test_executions"`
	CrossProjectRequiredGates int    `json:"cross_project_required_gates"`
	OutputScope               string `json:"output_scope"`
}

type OperatorAuthority struct {
	OperatorAPIAttempts *int64 `json:"operator_api_attempts"`
	OperatorAPIAttemptsState string `json:"operator_api_attempts_state"`
	AuthoringEvents      int    `json:"authoring_events"`
}

type OrchestratorAuthority struct {
	LocalAuthoringEvents int `json:"local_authoring_events"`
	LocalValidationEvents int `json:"local_validation_events"`
}

type AuthorityReport struct {
	Layers       []string               `json:"layers"`
	RepositoryWrites        int         `json:"repository_writes"`
	LocalTestExecutions     int         `json:"local_test_executions"`
	CrossProjectRequiredGates int       `json:"cross_project_required_gates"`
	Runtime      RuntimeAuthority       `json:"runtime"`
	Operator     OperatorAuthority      `json:"operator"`
	Orchestrator OrchestratorAuthority  `json:"orchestrator"`
}

type OperationalAudit struct {
	State                    string   `json:"state"`
	ExactCount               int      `json:"exact_count"`
	Commands                 []string `json:"commands"`
	Source                   string   `json:"source"`
	PreservedExistingRefuted bool     `json:"preserved_existing_refuted"`
	ExecutedByCurrentRuntime bool     `json:"executed_by_current_runtime"`
	Reason                   string   `json:"reason"`
}

type UtilityReport struct {
	Status        string         `json:"status"`
	Unknown       UnknownRecord  `json:"unknown"`
}

type ReplayReport struct {
	Requested          bool   `json:"requested"`
	EventsMatch        bool   `json:"events_match"`
	ReceiptsMatch      bool   `json:"receipts_match"`
	ReportsMatch       bool   `json:"reports_match"`
	Deterministic      bool   `json:"deterministic"`
}

type ChainReport struct {
	Schema       string `json:"schema"`
	AppendOnly   bool   `json:"append_only"`
	Length       int    `json:"length"`
	HeadDigest   string `json:"head_digest"`
	TailDigest   string `json:"tail_digest"`
	Valid        bool   `json:"valid"`
}

type InventoryReport struct {
	RootReadmeExcluded bool `json:"root_readme_excluded"`
	GitExcluded        bool `json:"git_excluded"`
	OutputExcluded     bool `json:"caller_output_excluded"`
	RegularFiles       int  `json:"regular_files"`
	DescendantDirs     int  `json:"descendant_dirs"`
	GoFiles            int  `json:"go_files"`
	GoooFiles          int  `json:"gooo_files"`
}

type ExactMetrics struct {
	Denominator               int         `json:"denominator"`
	States                   StateCounts `json:"states"`
	EventCount               int         `json:"event_count"`
	ReceiptCount             int         `json:"receipt_count"`
	RepositoryWrites         int         `json:"repository_writes"`
	LocalTestExecutions      int         `json:"local_test_executions"`
	CrossProjectRequiredGates int        `json:"cross_project_required_gates"`
}

type Report struct {
	Schema             string             `json:"schema"`
	Decision           string             `json:"decision"`
	OperatorAPIAttempts *int64            `json:"operator_api_attempts"`
	OperatorAPIAttemptsState string        `json:"operator_api_attempts_state"`
	Contract           string             `json:"contract"`
	ContractDigest     string             `json:"contract_digest"`
	Fixture            string             `json:"fixture"`
	FixtureDigest      string             `json:"fixture_digest"`
	SemanticGraphDigest string            `json:"semantic_graph_digest"`
	Toolchain          string             `json:"toolchain"`
	Runner             string             `json:"runner"`
	Cases              []CaseReport       `json:"cases"`
	Summary            StateCounts        `json:"summary"`
	Metrics            ExactMetrics       `json:"metrics"`
	Authority          AuthorityReport    `json:"authority"`
	OperationalAudit   OperationalAudit   `json:"operational_audit"`
	Utility            UtilityReport      `json:"utility"`
	ReceiptChain       ChainReport        `json:"receipt_chain"`
	Replay             ReplayReport       `json:"replay"`
	Inventory          InventoryReport    `json:"inventory"`
	ArtifactDigests    map[string]string  `json:"artifact_digests"`
}

type Artifact struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	Digest string `json:"digest"`
}

type Projection struct {
	Report             Report
	Events             []Event
	EventNDJSON        []byte
	Receipts           map[string][]byte
	LayerReceipts      map[string][]byte
	Generated          map[string][]byte
	Contradiction      []byte
	Human              []byte
	ReceiptObjects     []LayerReceipt
}

type OutputReport struct {
	Report Report
	Artifacts []Artifact `json:"artifacts"`
}

func cloneCountVector(value CountVector) CountVector {
	return CountVector{Attempts: cloneInt(value.Attempts), Success: cloneInt(value.Success), Failure: cloneInt(value.Failure), Unknown: cloneInt(value.Unknown)}
}

func cloneInt(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func countValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func isComplete(value CountVector) bool {
	return value.Attempts != nil && value.Success != nil && value.Failure != nil && value.Unknown != nil
}

func (value CountVector) MarshalJSON() ([]byte, error) {
	type plain CountVector
	return json.Marshal(plain(value))
}
