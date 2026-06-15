package projection

type Violation struct {
	Code        string `json:"code"`
	Requirement string `json:"requirement"`
	Message     string `json:"message"`
}

type Node struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`  // "requirement" | "spec" | "test"
	Label string `json:"label"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"` // "spec" | "test"
}

type Graph struct {
	Verdict          string      `json:"verdict"`          // "PASS" | "FAIL"
	RequirementCount int         `json:"requirementCount"`
	Violations       []Violation `json:"violations"`
	Nodes            []Node      `json:"nodes"`
	Edges            []Edge      `json:"edges"`
	GeneratedAt      string      `json:"generatedAt"` // RFC3339 UTC
}
