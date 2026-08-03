package models

// Overview is the result of `repomap overview`.
type Overview struct {
	Module      string   `json:"module"`
	GoVersion   string   `json:"go_version,omitempty"`
	Packages    []PkgRef `json:"packages"`
	Entrypoints []PkgRef `json:"entrypoints"`
	Generated   []string `json:"generated"`
	Vendor      []string `json:"vendor"`
	Stats       Stats    `json:"stats"`
}

// Stats summarizes counts across the module.
type Stats struct {
	Packages    int `json:"packages"`
	Files       int `json:"files"`
	Entrypoints int `json:"entrypoints"`
}

// PkgRef is a lightweight reference to a package.
type PkgRef struct {
	// ImportPath is the full import path, e.g. github.com/acme/svc/internal/auth.
	ImportPath string `json:"import_path"`
	// Name is the package clause name, e.g. auth or main.
	Name string `json:"name"`
	// Dir is the directory relative to the module root.
	Dir string `json:"dir"`
}

// Package is the result of `repomap package <path>`.
type Package struct {
	ImportPath string   `json:"import_path"`
	Name       string   `json:"name"`
	Dir        string   `json:"dir"`
	Imports    []string `json:"imports"`
	ImportedBy []string `json:"imported_by"`
	Exports    []Symbol `json:"exports"`
	TestFiles  []string `json:"test_files"`
}

// Symbol is the result of `repomap symbol <name>` and is also used to describe
// exported symbols elsewhere.
type Symbol struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"` // func, type, var, const, method
	Package   string   `json:"package"`
	Location  Location `json:"location"`
	Signature string   `json:"signature,omitempty"`
	Doc       string   `json:"doc,omitempty"`
	Body      string   `json:"body,omitempty"`
	Recv      string   `json:"recv,omitempty"` // receiver type for methods
	Shape     *Struct  `json:"shape,omitempty"`
	Callers   []Caller `json:"callers,omitempty"`
	Tests     []Test   `json:"tests,omitempty"`
}

// Caller is a single call site.
type Caller struct {
	Symbol     string   `json:"symbol"`
	Package    string   `json:"package"`
	Location   Location `json:"location"`
	Context    string   `json:"context"`    // one-line source at the call site
	Confidence string   `json:"confidence"` // "direct" or "possible"
}

// Callers is the result of `repomap callers <name>`.
type Callers struct {
	Symbol   string   `json:"symbol"`
	Package  string   `json:"package"`
	Location Location `json:"location"`
	Direct   []Caller `json:"direct"`
	Indirect []Caller `json:"indirect"`
}

// Deps is the result of `repomap deps <path>`.
type Deps struct {
	Root  string    `json:"root"`
	Depth int       `json:"depth"`
	Nodes []string  `json:"nodes"`
	Edges []DepEdge `json:"edges"`
}

// DepEdge is a directed import edge from one package to another.
type DepEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Impact is the result of `repomap impact --base <ref>`.
type Impact struct {
	Base             string   `json:"base"`
	ChangedFiles     []string `json:"changed_files"`
	ChangedPackages  []string `json:"changed_packages"`
	AffectedPackages []string `json:"affected_packages"`
	PublicAPIChanged []Symbol `json:"public_api_changed"`
	RecommendedTests []string `json:"recommended_tests"`
	RiskScore        int      `json:"risk_score"` // 0-100
	RiskLevel        string   `json:"risk_level"` // low, medium, high
}

// Tests is the result of `repomap tests <name>`.
type Tests struct {
	Symbol      string `json:"symbol"`
	Unit        []Test `json:"unit"`
	Integration []Test `json:"integration"`
	Benchmark   []Test `json:"benchmark"`
}

// Test describes a single test function.
type Test struct {
	Name     string   `json:"name"`
	Package  string   `json:"package"`
	Location Location `json:"location"`
	Kind     string   `json:"kind"` // unit, integration, benchmark
}

// Endpoint is the result of `repomap endpoint <method> <path>`.
type Endpoint struct {
	Method        string     `json:"method"`
	Path          string     `json:"path"`
	Framework     string     `json:"framework"`
	Handler       *Symbol    `json:"handler,omitempty"`
	Route         Location   `json:"route"`
	Orchestration []Symbol   `json:"orchestration,omitempty"`
	Upstreams     []Upstream `json:"upstreams,omitempty"`
	RequestType   string     `json:"request_type,omitempty"`
	ResponseType  string     `json:"response_type,omitempty"`
	Confidence    string     `json:"confidence"`
}

// Upstreams is the result of `repomap upstreams <path>`.
type Upstreams struct {
	Root  string     `json:"root"`
	Calls []Upstream `json:"calls"`
}

// Upstream is a single outbound call.
type Upstream struct {
	Service    string   `json:"service,omitempty"`
	Method     string   `json:"method,omitempty"` // HTTP method if known
	URL        string   `json:"url,omitempty"`    // URL or path constant
	DecodeType string   `json:"decode_type,omitempty"`
	Location   Location `json:"location"`
	Confidence string   `json:"confidence"`
}

// Struct is the result of `repomap struct <name>` (and `symbol --shape`). It is
// a recursive description of a type's JSON contract.
type Struct struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// Field describes a single struct field's JSON contract.
type Field struct {
	Name     string  `json:"name"`
	JSONName string  `json:"json_name,omitempty"`
	Type     string  `json:"type"`
	Optional bool    `json:"optional,omitempty"`
	Nested   *Struct `json:"nested,omitempty"`
}
