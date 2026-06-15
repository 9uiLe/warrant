package projection

type Violation struct {
	Code        string `json:"code"`
	Requirement string `json:"requirement"`
	Message     string `json:"message"`
}

type Node struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"` // "requirement" | "spec" | "test" | "constitution" | "rule" | "check"
	Label string `json:"label"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"` // "spec" | "test" | "basis" | "enforces"
}

type Graph struct {
	Verdict          string      `json:"verdict"` // "PASS" | "FAIL"
	RequirementCount int         `json:"requirementCount"`
	Violations       []Violation `json:"violations"`
	Nodes            []Node      `json:"nodes"`
	Edges            []Edge      `json:"edges"`
	GeneratedAt      string      `json:"generatedAt"` // RFC3339 UTC
}

type Report struct {
	Meta         ReportMeta          `json:"meta"`
	Summary      ReportSummary       `json:"summary"`
	Requirements []ReportRequirement `json:"requirements"`
}
type ReportMeta struct {
	GeneratedAt string `json:"generatedAt"`
	Repository  string `json:"repository"`
	Branch      string `json:"branch"`
	Commit      string `json:"commit"`
}
type ReportSummary struct {
	Total           int `json:"total"`
	Passed          int `json:"passed"`
	Failed          int `json:"failed"`
	Draft           int `json:"draft"`
	CoveragePercent int `json:"coveragePercent"`
}
type ReportRequirement struct {
	ID      string       `json:"id"`
	Title   string       `json:"title"`
	Status  string       `json:"status"`
	Verdict string       `json:"verdict"`
	Tests   []ReportTest `json:"tests"`
	Spec    ReportSpec   `json:"spec"`
}
type ReportTest struct {
	File   string `json:"file"`
	Status string `json:"status"` // linked|broken
}
type ReportSpec struct {
	Doc     string `json:"doc"`
	Section string `json:"section"`
	Status  string `json:"status"` // linked|broken|missing
}
