package evalharness

type Case struct {
	ID             string   `json:"id"`
	Category       string   `json:"category,omitempty"`
	Input          string   `json:"input"`
	ExpectedRoute  string   `json:"expected_route,omitempty"`
	ExpectedTools  []string `json:"expected_tools,omitempty"`
	ForbiddenTools []string `json:"forbidden_tools,omitempty"`
	FinalContains  []string `json:"final_contains,omitempty"`
}

type Observation struct {
	ID     string   `json:"id"`
	Route  string   `json:"route,omitempty"`
	Tools  []string `json:"tools,omitempty"`
	Final  string   `json:"final,omitempty"`
	Errors []string `json:"errors,omitempty"`
}

type CaseResult struct {
	ID     string             `json:"id"`
	Pass   bool               `json:"pass"`
	Scores map[string]float64 `json:"scores"`
	Issues []string           `json:"issues,omitempty"`
}

type SuiteResult struct {
	Total    int                `json:"total"`
	Passed   int                `json:"passed"`
	Failed   int                `json:"failed"`
	PassRate float64            `json:"pass_rate"`
	Results  []CaseResult       `json:"results"`
	Averages map[string]float64 `json:"averages"`
}

type ABComparison struct {
	Baseline         SuiteResult        `json:"baseline"`
	Candidate        SuiteResult        `json:"candidate"`
	PassRateDelta    float64            `json:"pass_rate_delta"`
	AverageDeltas    map[string]float64 `json:"average_deltas"`
	Regressions      []CaseDelta        `json:"regressions,omitempty"`
	Improvements     []CaseDelta        `json:"improvements,omitempty"`
	ChangedScores    []CaseDelta        `json:"changed_scores,omitempty"`
	RegressionCount  int                `json:"regression_count"`
	ImprovementCount int                `json:"improvement_count"`
}

type CaseDelta struct {
	ID              string             `json:"id"`
	BaselinePass    bool               `json:"baseline_pass"`
	CandidatePass   bool               `json:"candidate_pass"`
	BaselineScores  map[string]float64 `json:"baseline_scores,omitempty"`
	CandidateScores map[string]float64 `json:"candidate_scores,omitempty"`
	ScoreDeltas     map[string]float64 `json:"score_deltas,omitempty"`
	BaselineIssues  []string           `json:"baseline_issues,omitempty"`
	CandidateIssues []string           `json:"candidate_issues,omitempty"`
}
